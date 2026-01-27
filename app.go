package main

import (
	"context"
	_ "embed"
	"fmt"
	"runtime"
	"ts-escpos/backend/config"
	"ts-escpos/backend/receipt"
	"ts-escpos/backend/tray"

	"ts-escpos/backend/jobs"
	"ts-escpos/backend/printer"
	"ts-escpos/backend/server"
	"ts-escpos/backend/updater"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/appicon.png
var appIcon []byte

const (
	// Update this with your actual repository "owner/repo"
	GithubRepo = "saurabh1e/ts-escpos"
)

var AppVersion = "0.0.1"

// App struct
type App struct {
	ctx        context.Context
	store      *jobs.Store
	cfg        *config.Config
	server     *server.Server
	tray       *tray.TrayApp
	IsQuitting bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	cfg := config.LoadConfig()
	store := jobs.NewStore()
	srv := server.NewServer(store, cfg)
	t := tray.NewTrayApp(appIcon)

	return &App{
		store:  store,
		cfg:    cfg,
		server: srv,
		tray:   t,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Pass context to server for notifications
	a.server.SetContext(ctx)

	// Enable Auto Start on Windows
	if err := SetAutoStart(true); err != nil {
		fmt.Printf("Failed to set auto-start: %v\n", err)
	}

	// Start System Tray
	a.tray.Start(ctx)

	// Start HTTP Server
	a.server.Start()

	// System Tray logic removed due to Wails v2 API limitations

}

// EnableAutoStart can be called from frontend to toggle auto-start behavior
func (a *App) EnableAutoStart(enable bool) error {
	return SetAutoStart(enable)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) GetPrinters() ([]printer.PrinterInfo, error) {
	return printer.GetPrinters()
}

func (a *App) GetPrintJobs() []jobs.PrintJob {
	return a.store.GetJobs()
}

type UpdateResponse struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Body      string `json:"body"`
	URL       string `json:"url"`
}

func (a *App) CheckForUpdates() (*UpdateResponse, error) {
	release, err := updater.CheckForUpdates(AppVersion, GithubRepo)
	if err != nil {
		return nil, err
	}

	if release != nil {
		// Find the suitable asset
		var downloadURL string
		for _, asset := range release.Assets {
			// Simple logic: look for .exe on Windows
			if runtime.GOOS == "windows" && len(asset.BrowserDownloadURL) > 4 && asset.BrowserDownloadURL[len(asset.BrowserDownloadURL)-4:] == ".exe" {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}

		// If no specific asset found, use the first one or generic logic
		if downloadURL == "" && len(release.Assets) > 0 {
			downloadURL = release.Assets[0].BrowserDownloadURL
		}

		return &UpdateResponse{
			Available: true,
			Version:   release.TagName,
			Body:      release.Body,
			URL:       downloadURL,
		}, nil
	}

	return &UpdateResponse{Available: false}, nil
}

func (a *App) InstallUpdate(url string) error {
	if url == "" {
		return fmt.Errorf("no download URL provided")
	}

	err := updater.DownloadAndInstall(url)
	if err != nil {
		return err
	}

	// Quit the app so the installer can run
	wailsRuntime.Quit(a.ctx)
	return nil
}

func (a *App) GetMachineID() (string, error) {
	return config.GetMachineID()
}

func (a *App) GetServerStatus() map[string]interface{} {
	// runtime.SystemTraySetTitle("Test")
	return map[string]interface{}{
		"port":    a.cfg.HTTPPort,
		"running": true,
	}
}

func (a *App) TestPrint(printerName string) error {
	printers, err := printer.GetPrinters()
	if err != nil {
		return fmt.Errorf("failed to get printers: %w", err)
	}

	var selectedPrinter printer.PrinterInfo
	found := false
	for _, p := range printers {
		if p.Name == printerName {
			selectedPrinter = p
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("printer '%s' not found", printerName)
	}

	// Basic check for obvious issues
	if selectedPrinter.Status != "Ready" && selectedPrinter.Status != "" {
		// Just a warning log, but we try anyway? Or fail?
		// Usually if it's "Offline" we probably shouldn't try, but windows spooler might queue it.
		// Let's just proceed.
	}

	adapter := printer.NewEscposAdapter()
	// Pass the printer name to the adapter so it knows where to print
	// adapter.SetPrinterName(printerName)

	sampleData := receipt.GetSampleOrderData()
	receipt.RenderBill(adapter, sampleData, "80mm") // Defaulting to 80mm for test

	return printer.PrintRaw(printerName, adapter.GetBytes())
}
