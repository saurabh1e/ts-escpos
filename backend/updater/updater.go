package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
)

var githubAPIClient = &http.Client{Timeout: 15 * time.Second}

var downloadClient = &http.Client{Timeout: 10 * time.Minute}

var latestReleaseURL = func(repo string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
}

var currentOS = runtime.GOOS

var commandRunner = exec.Command

var executablePathProvider = os.Executable

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
	Body    string         `json:"body"`
}

func CheckForUpdates(currentVersion string, repo string) (*Release, error) {
	req, err := http.NewRequest(http.MethodGet, latestReleaseURL(repo), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ts-escpos-updater")

	resp, err := githubAPIClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return nil, fmt.Errorf("failed to check for updates: %s", resp.Status)
		}

		message := strings.TrimSpace(string(body))
		if message == "" {
			return nil, fmt.Errorf("failed to check for updates: %s", resp.Status)
		}

		return nil, fmt.Errorf("failed to check for updates: %s: %s", resp.Status, message)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	vCurrent, err := semver.Make(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return nil, fmt.Errorf("invalid current version: %v", err)
	}

	vLatest, err := semver.Make(strings.TrimPrefix(release.TagName, "v"))
	if err != nil {
		return nil, fmt.Errorf("invalid latest version: %v", err)
	}

	if vLatest.GT(vCurrent) {
		return &release, nil
	}

	return nil, nil // No update needed
}

func SelectDownloadURL(release *Release) (string, error) {
	asset, err := SelectReleaseAsset(release)
	if err != nil {
		return "", err
	}

	return asset.BrowserDownloadURL, nil
}

func SelectReleaseAsset(release *Release) (*ReleaseAsset, error) {
	if release == nil {
		return nil, fmt.Errorf("release is required")
	}

	if len(release.Assets) == 0 {
		return nil, fmt.Errorf("release %s does not contain installable assets", release.TagName)
	}

	if currentOS == "windows" {
		return selectWindowsAsset(release.Assets)
	}

	return &release.Assets[0], nil
}

func selectWindowsAsset(assets []ReleaseAsset) (*ReleaseAsset, error) {
	var fallback *ReleaseAsset

	for index := range assets {
		asset := &assets[index]
		name := strings.ToLower(asset.Name)
		downloadURL := strings.ToLower(asset.BrowserDownloadURL)
		isExecutable := strings.HasSuffix(name, ".exe") || strings.HasSuffix(downloadURL, ".exe")
		if !isExecutable {
			continue
		}

		if strings.Contains(name, "installer") || strings.Contains(downloadURL, "installer") {
			return asset, nil
		}

		if fallback == nil {
			fallback = asset
		}
	}

	if fallback != nil {
		return fallback, nil
	}

	return nil, fmt.Errorf("no Windows installer asset found")
}

func DownloadAndInstall(downloadUrl string) error {
	installerPath, err := DownloadReleaseAsset(downloadUrl)
	if err != nil {
		return err
	}

	executableName, err := runningExecutableName()
	if err != nil {
		return err
	}

	return LaunchInstaller(installerPath, os.Getpid(), executableName)
}

func DownloadReleaseAsset(downloadUrl string) (string, error) {
	if downloadUrl == "" {
		return "", fmt.Errorf("no download URL provided")
	}

	if currentOS != "windows" {
		return "", fmt.Errorf("automatic install not fully supported on this OS, please download manually")
	}

	req, err := http.NewRequest(http.MethodGet, downloadUrl, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "ts-escpos-updater")

	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download update: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "ts-escpos-update-*"+installerExtension(downloadUrl))
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", err
	}

	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpPath, nil
}

func LaunchInstaller(installerPath string, currentPID int, appExecutableName string) error {
	if installerPath == "" {
		return fmt.Errorf("installer path is required")
	}

	if appExecutableName == "" {
		return fmt.Errorf("app executable name is required")
	}

	if currentOS != "windows" {
		return fmt.Errorf("automatic install not fully supported on this OS, please download manually")
	}

	scriptPath, err := createWindowsInstallScript(installerPath, currentPID, appExecutableName)
	if err != nil {
		return err
	}

	cmd := commandRunner("cmd", "/C", scriptPath)
	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

func createWindowsInstallScript(installerPath string, currentPID int, appExecutableName string) (string, error) {
	scriptFile, err := os.CreateTemp("", "ts-escpos-install-*.cmd")
	if err != nil {
		return "", err
	}
	defer scriptFile.Close()

	if _, err := scriptFile.WriteString(windowsInstallScript(installerPath, currentPID, appExecutableName)); err != nil {
		return "", err
	}

	return scriptFile.Name(), nil
}

func windowsInstallScript(installerPath string, currentPID int, appExecutableName string) string {
	return fmt.Sprintf("@echo off\r\nsetlocal\r\nset PID=%d\r\nset APP_EXE=%s\r\n:wait_current_process\r\ntasklist /FI \"PID eq %%PID%%\" 2>NUL | find /I \"%%PID%%\" >NUL\r\nif not errorlevel 1 (\r\n    timeout /T 2 /NOBREAK >NUL\r\n    goto wait_current_process\r\n)\r\ntaskkill /F /T /IM \"%%APP_EXE%%\" >NUL 2>NUL\r\nset /A CLOSE_ATTEMPTS=0\r\n:wait_other_instances\r\ntasklist /FI \"IMAGENAME eq %%APP_EXE%%\" 2>NUL | find /I \"%%APP_EXE%%\" >NUL\r\nif errorlevel 1 goto start_installer\r\nset /A CLOSE_ATTEMPTS+=1\r\nif %%CLOSE_ATTEMPTS%% GEQ 15 goto start_installer\r\ntaskkill /F /T /IM \"%%APP_EXE%%\" >NUL 2>NUL\r\ntimeout /T 2 /NOBREAK >NUL\r\ngoto wait_other_instances\r\n:start_installer\r\nstart \"\" \"%s\" /S\r\ndel \"%%~f0\"\r\n", currentPID, appExecutableName, installerPath)
}

func runningExecutableName() (string, error) {
	executablePath, err := executablePathProvider()
	if err != nil {
		return "", err
	}

	normalizedPath := strings.ReplaceAll(executablePath, "\\", "/")
	executableName := path.Base(normalizedPath)
	if executableName == normalizedPath {
		executableName = filepath.Base(executablePath)
	}
	if executableName == "" || executableName == "." {
		return "", fmt.Errorf("failed to determine running executable name")
	}

	return executableName, nil
}

func installerExtension(downloadUrl string) string {
	parsedURL, err := url.Parse(downloadUrl)
	if err == nil && parsedURL.Path != "" {
		extension := strings.ToLower(path.Ext(parsedURL.Path))
		if extension != "" {
			return extension
		}
	}

	extension := strings.ToLower(path.Ext(downloadUrl))
	if extension != "" {
		return extension
	}

	if currentOS == "windows" {
		return ".exe"
	}

	return ""
}
