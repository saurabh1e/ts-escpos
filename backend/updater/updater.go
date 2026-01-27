package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/blang/semver"
)

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	Body string `json:"body"`
}

func CheckForUpdates(currentVersion string, repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to check for updates: %s", resp.Status)
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

func DownloadAndInstall(downloadUrl string) error {
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create a temporary file
	var ext string
	if runtime.GOOS == "windows" {
		ext = ".exe"
	} else if runtime.GOOS == "darwin" {
		// MacOS installation is more complex (dmg/pkg), for now we might just open browser or handle .zip
		// Assuming dmg or zip for mac, but automated install is harder without specific logic
		// If it's a raw binary, we can replace it. If it's an installer, we run it.
		// For now, let's focus on Windows as requested ("installer for windows")
		return fmt.Errorf("automatic install not fully supported on this OS, please download manually")
	}

	tmpFile, err := os.CreateTemp("", "update-*"+ext)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()
	tmpFile.Close() // Close explicitly before executing

	if runtime.GOOS == "windows" {
		// Run the installer
		cmd := exec.Command(tmpPath)
		if err := cmd.Start(); err != nil {
			return err
		}
		// We should exit to allow the installer to overwrite files if needed
		// But the caller will handle exit
		return nil
	}

	return fmt.Errorf("unsupported OS for auto-install")
}
