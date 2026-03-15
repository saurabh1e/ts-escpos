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
	targetArch := runtime.GOARCH

	for index := range assets {
		asset := &assets[index]
		name := strings.ToLower(asset.Name)
		downloadURL := strings.ToLower(asset.BrowserDownloadURL)

		// Basic filters
		if !strings.HasSuffix(name, ".exe") && !strings.HasSuffix(downloadURL, ".exe") {
			continue
		}

		// We want an installer, not the raw binary
		if !strings.Contains(name, "installer") && !strings.Contains(downloadURL, "installer") {
			continue
		}

		// Filter out Lite versions if we are not running Lite (headless/lite code handles its own updates)
		// But wait, the headless/lite code in cmd/headless/main.go has its OWN logic (performSelfUpdate).
		// This file (backend/updater/updater.go) is used by keys/wails MAIN app.
		// So we must exclude "lite" assets here.
		if strings.Contains(name, "lite") || strings.Contains(downloadURL, "lite") {
			continue
		}

		// Match architecture
		if targetArch == "amd64" {
			if strings.Contains(name, "386") || strings.Contains(name, "x86") {
				continue // Skip 32-bit assets on 64-bit system unless no other choice?
				// Actually we should strictly prefer amd64
			}
		} else if targetArch == "386" {
			if strings.Contains(name, "amd64") || strings.Contains(name, "x64") {
				continue // Skip 64-bit assets on 32-bit system
			}
		}

		return asset, nil
	}

	return nil, fmt.Errorf("no suitable Windows installer asset found for arch %s", targetArch)
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
	return fmt.Sprintf(`@echo off
setlocal
set PID=%d
set APP_EXE=%s

echo Waiting for application to close...
:wait_current_process
tasklist /FI "PID eq %%PID%%" 2>NUL | find /I "%%PID%%" >NUL
if not errorlevel 1 (
    timeout /T 2 /NOBREAK >NUL
    goto wait_current_process
)

echo App closed. Cleaning up any stuck instances...
taskkill /F /T /IM "%%APP_EXE%%" >NUL 2>NUL
set /A CLOSE_ATTEMPTS=0

:wait_other_instances
tasklist /FI "IMAGENAME eq %%APP_EXE%%" 2>NUL | find /I "%%APP_EXE%%" >NUL
if errorlevel 1 goto start_installer

echo Stuck instance found. Killing...
set /A CLOSE_ATTEMPTS+=1
if %%CLOSE_ATTEMPTS%% GEQ 15 (
    echo Failed to close all instances. Proceeding anyway...
    goto start_installer
)

taskkill /F /T /IM "%%APP_EXE%%" >NUL 2>NUL
timeout /T 2 /NOBREAK >NUL
goto wait_other_instances

:start_installer
echo Starting installer...
start "" "%s" /S
del "%%~f0"
`, currentPID, appExecutableName, installerPath)
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
