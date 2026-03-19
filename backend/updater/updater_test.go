package updater

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckForUpdatesReturnsReleaseWhenNewer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "ts-escpos-updater" {
			t.Fatalf("expected User-Agent header to be set")
		}

		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Fatalf("expected Accept header to be set")
		}

		fmt.Fprint(w, `[{"tag_name":"v0.0.11","assets":[{"name":"installer.exe","browser_download_url":"https://example.com/installer.exe"}],"body":"Bug fixes"}]`)
	}))
	defer server.Close()

	restore := stubLatestReleaseEndpoint(server.URL, server.Client())
	defer restore()

	release, err := CheckForUpdates("0.0.10", "ignored/repo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if release == nil {
		t.Fatalf("expected release to be returned")
	}

	if release.TagName != "v0.0.11" {
		t.Fatalf("expected tag v0.0.11, got %s", release.TagName)
	}
}

func TestCheckForUpdatesReturnsNilWhenCurrentVersionMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"tag_name":"v0.0.10","assets":[],"body":"Current"}]`)
	}))
	defer server.Close()

	restore := stubLatestReleaseEndpoint(server.URL, server.Client())
	defer restore()

	release, err := CheckForUpdates("0.0.10", "ignored/repo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if release != nil {
		t.Fatalf("expected no release, got %+v", release)
	}
}

func TestCheckForUpdatesReturnsAPIErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message":"API rate limit exceeded"}`)
	}))
	defer server.Close()

	restore := stubLatestReleaseEndpoint(server.URL, server.Client())
	defer restore()

	release, err := CheckForUpdates("0.0.10", "ignored/repo")
	if err == nil {
		t.Fatalf("expected an error")
	}

	if release != nil {
		t.Fatalf("expected no release, got %+v", release)
	}

	if !strings.Contains(err.Error(), "API rate limit exceeded") {
		t.Fatalf("expected error to include response body, got %v", err)
	}
}

func TestSelectReleaseAssetPrefersWindowsInstaller(t *testing.T) {
	restore := stubCurrentOS("windows")
	defer restore()
	restoreArch := stubCurrentArch("amd64")
	defer restoreArch()

	release := &Release{
		TagName: "v0.0.11",
		Assets: []ReleaseAsset{
			{Name: "ts-escpos-386-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-386-installer.exe"},
			{Name: "ts-escpos.exe", BrowserDownloadURL: "https://example.com/ts-escpos.exe"},
			{Name: "ts-escpos-amd64-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-amd64-installer.exe"},
		},
	}

	asset, err := SelectReleaseAsset(release)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if asset.Name != "ts-escpos-amd64-installer.exe" {
		t.Fatalf("expected installer asset, got %s", asset.Name)
	}
}

func TestSelectReleaseAssetReturnsErrorWithoutWindowsInstaller(t *testing.T) {
	restore := stubCurrentOS("windows")
	defer restore()
	restoreArch := stubCurrentArch("amd64")
	defer restoreArch()

	release := &Release{
		TagName: "v0.0.11",
		Assets: []ReleaseAsset{
			{Name: "ts-escpos.zip", BrowserDownloadURL: "https://example.com/ts-escpos.zip"},
		},
	}

	asset, err := SelectReleaseAsset(release)
	if err == nil {
		t.Fatalf("expected an error")
	}

	if asset != nil {
		t.Fatalf("expected no asset, got %+v", asset)
	}
}

func TestSelectReleaseAssetUsesExactAMD64InstallerEvenWhenLegacyBridgeComesFirst(t *testing.T) {
	restore := stubCurrentOS("windows")
	defer restore()
	restoreArch := stubCurrentArch("amd64")
	defer restoreArch()

	release := &Release{
		TagName: "v0.0.25",
		Assets: []ReleaseAsset{
			{Name: "ts-escpos-000-amd64-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-000-amd64-installer.exe"},
			{Name: "ts-escpos-386-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-386-installer.exe"},
			{Name: "ts-escpos-lite-amd64-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-lite-amd64-installer.exe"},
			{Name: "ts-escpos-amd64-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-amd64-installer.exe"},
		},
	}

	asset, err := SelectReleaseAsset(release)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if asset.Name != "ts-escpos-amd64-installer.exe" {
		t.Fatalf("expected amd64 installer asset, got %s", asset.Name)
	}
}

func TestSelectReleaseAssetSkipsLegacyAMD64BridgeOn386(t *testing.T) {
	restore := stubCurrentOS("windows")
	defer restore()
	restoreArch := stubCurrentArch("386")
	defer restoreArch()

	release := &Release{
		TagName: "v0.0.25",
		Assets: []ReleaseAsset{
			{Name: "ts-escpos-000-amd64-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-000-amd64-installer.exe"},
			{Name: "ts-escpos-amd64-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-amd64-installer.exe"},
			{Name: "ts-escpos-386-installer.exe", BrowserDownloadURL: "https://example.com/ts-escpos-386-installer.exe"},
		},
	}

	asset, err := SelectReleaseAsset(release)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if asset.Name != "ts-escpos-386-installer.exe" {
		t.Fatalf("expected 386 installer asset, got %s", asset.Name)
	}
}

func TestWindowsInstallScriptWaitsForCurrentProcess(t *testing.T) {
	script := windowsInstallScript(`C:\Temp\ts-escpos-amd64-installer.exe`, 4321, `ts-escpos.exe`)

	if !strings.Contains(script, `set PID=4321`) {
		t.Fatalf("expected script to contain the current pid, got %s", script)
	}

	if !strings.Contains(script, `set APP_EXE=ts-escpos.exe`) {
		t.Fatalf("expected script to include the app executable name, got %s", script)
	}

	if !strings.Contains(script, `taskkill /F /T /IM "%APP_EXE%"`) {
		t.Fatalf("expected script to terminate running app instances, got %s", script)
	}

	if !strings.Contains(script, `IMAGENAME eq %APP_EXE%`) {
		t.Fatalf("expected script to wait for remaining instances to close, got %s", script)
	}

	if !strings.Contains(script, `start "" "C:\Temp\ts-escpos-amd64-installer.exe" /S`) {
		t.Fatalf("expected script to launch the installer, got %s", script)
	}

	if !strings.Contains(script, `goto wait_current_process`) {
		t.Fatalf("expected script to wait for the app to exit, got %s", script)
	}
}

func TestRunningExecutableNameUsesCurrentExecutablePath(t *testing.T) {
	previousProvider := executablePathProvider
	executablePathProvider = func() (string, error) {
		return `C:\Program Files\ts-escpos\ts-escpos.exe`, nil
	}
	defer func() {
		executablePathProvider = previousProvider
	}()

	executableName, err := runningExecutableName()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if executableName != "ts-escpos.exe" {
		t.Fatalf("expected executable name ts-escpos.exe, got %s", executableName)
	}
}

func TestInstallerExtensionUsesURLPath(t *testing.T) {
	restore := stubCurrentOS("windows")
	defer restore()

	extension := installerExtension("https://example.com/download/ts-escpos-amd64-installer.exe?token=abc")
	if extension != ".exe" {
		t.Fatalf("expected .exe extension, got %s", extension)
	}
}

func stubLatestReleaseEndpoint(url string, client *http.Client) func() {
	previousURL := latestReleaseURL
	previousClient := githubAPIClient

	latestReleaseURL = func(repo string) string {
		return url
	}
	githubAPIClient = client

	return func() {
		latestReleaseURL = previousURL
		githubAPIClient = previousClient
	}
}

func stubCurrentOS(os string) func() {
	previousOS := currentOS
	currentOS = os

	return func() {
		currentOS = previousOS
	}
}

func stubCurrentArch(arch string) func() {
	previousArch := currentArch
	currentArch = arch

	return func() {
		currentArch = previousArch
	}
}
