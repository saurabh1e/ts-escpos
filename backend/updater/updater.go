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
	// List releases instead of just the latest one (which might be "latest" tag which fails semver)
	return fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
}

var currentOS = runtime.GOOS

var currentArch = runtime.GOARCH

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

	// Set query params to limit results
	q := req.URL.Query()
	q.Add("per_page", "5") // Check top 5 releases
	req.URL.RawQuery = q.Encode()

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

	// resp.Body decode change to slice of Release
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil // No releases found
	}

	vCurrent, err := semver.Make(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return nil, fmt.Errorf("invalid current version: %v", err)
	}

	// Iterate to find the first valid semver release that is greater than current
	var bestRelease *Release

	for i := range releases {
		rel := &releases[i]

		// Skip if draft
		// (API usually filters drafts for public repos unless authenticated, but good to be safe if we add struct field later)
		// We don't have IsDraft in struct yet, assuming public API behavior.

		// Parse version
		cleanTag := strings.TrimPrefix(rel.TagName, "v")
		vRel, err := semver.Make(cleanTag)
		if err != nil {
			// Skip invalid semver tags (like "latest" or "beta")
			continue
		}

		if vRel.GT(vCurrent) {
			// Found a newer version.
			// Since the list is ordered by date (newest first), the first one we find
			// that is > current is the one we want (it's the newest valid release).
			bestRelease = rel
			break
		}
	}

	if bestRelease != nil {
		return bestRelease, nil
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
	targetArch := currentArch

	for _, expectedName := range preferredWindowsInstallerNames(targetArch) {
		for index := range assets {
			asset := &assets[index]
			if matchesWindowsAssetName(asset, expectedName) {
				return asset, nil
			}
		}
	}

	for index := range assets {
		asset := &assets[index]
		if !isWindowsInstallerCandidate(asset) {
			continue
		}

		if !windowsAssetMatchesArch(asset, targetArch) {
			continue
		}

		return asset, nil
	}

	return nil, fmt.Errorf("no suitable Windows installer asset found for arch %s", targetArch)
}

func preferredWindowsInstallerNames(targetArch string) []string {
	if targetArch == "386" {
		return []string{
			"ts-escpos-386-installer.exe",
			"ts-escpos-386-setup.exe",
		}
	}

	return []string{
		"ts-escpos-amd64-installer.exe",
		"ts-escpos-amd64-setup.exe",
		"ts-escpos-000-amd64-installer.exe",
	}
}

func matchesWindowsAssetName(asset *ReleaseAsset, expectedName string) bool {
	if asset == nil {
		return false
	}

	expectedName = strings.ToLower(expectedName)
	name := strings.ToLower(asset.Name)
	if name == expectedName {
		return true
	}

	return assetDownloadBaseName(asset) == expectedName
}

func isWindowsInstallerCandidate(asset *ReleaseAsset) bool {
	if asset == nil {
		return false
	}

	name := strings.ToLower(asset.Name)
	downloadURL := strings.ToLower(asset.BrowserDownloadURL)

	if !strings.HasSuffix(name, ".exe") && !strings.HasSuffix(downloadURL, ".exe") {
		return false
	}

	if !strings.Contains(name, "installer") && !strings.Contains(downloadURL, "installer") && !strings.Contains(name, "setup") && !strings.Contains(downloadURL, "setup") {
		return false
	}

	if strings.Contains(name, "lite") || strings.Contains(downloadURL, "lite") {
		return false
	}

	return true
}

func windowsAssetMatchesArch(asset *ReleaseAsset, targetArch string) bool {
	if asset == nil {
		return false
	}

	name := strings.ToLower(asset.Name)
	downloadURL := strings.ToLower(asset.BrowserDownloadURL)

	if targetArch == "386" {
		if strings.Contains(name, "amd64") || strings.Contains(downloadURL, "amd64") || strings.Contains(name, "x64") || strings.Contains(downloadURL, "x64") || strings.Contains(name, "arm64") || strings.Contains(downloadURL, "arm64") {
			return false
		}

		return true
	}

	if strings.Contains(name, "386") || strings.Contains(downloadURL, "386") || strings.Contains(name, "x86") || strings.Contains(downloadURL, "x86") || strings.Contains(name, "arm64") || strings.Contains(downloadURL, "arm64") {
		return false
	}

	return true
}

func assetDownloadBaseName(asset *ReleaseAsset) string {
	parsedURL, err := url.Parse(asset.BrowserDownloadURL)
	if err == nil && parsedURL.Path != "" {
		return strings.ToLower(path.Base(parsedURL.Path))
	}

	return strings.ToLower(path.Base(asset.BrowserDownloadURL))
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
