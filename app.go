package main

import (
	"context"
	_ "embed"
	"fmt"
	"runtime"
	"sync"
	"time"
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

var AppVersion = "0.0.11"

const updateCheckInterval = 30 * time.Minute

// App struct
type App struct {
	ctx        context.Context
	store      *jobs.Store
	cfg        *config.Config
	server     *server.Server
	tray       *tray.TrayApp
	IsQuitting bool

	printerCache []printer.PrinterInfo
	printerMutex sync.RWMutex

	updateCheckCancel  context.CancelFunc
	updateCheckDone    chan struct{}
	updateStateMutex   sync.Mutex
	isCheckingUpdates  bool
	isInstallingUpdate bool
	updateStatus       UpdateStatus
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
		updateStatus: UpdateStatus{
			State:   "idle",
			Message: "Automatic updates enabled. Waiting for the next check.",
		},
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

	// Pre-fetch printers on startup
	go func() {
		_, _ = a.RefreshPrinters()
	}()

	// Start System Tray
	a.tray.Start(ctx)

	// Start HTTP Server
	a.server.Start()

	a.startAutoUpdateChecks()

	// System Tray logic removed due to Wails v2 API limitations

}

func (a *App) shutdown(ctx context.Context) {
	a.stopAutoUpdateChecks()
}

// EnableAutoStart can be called from frontend to toggle auto-start behavior
func (a *App) EnableAutoStart(enable bool) error {
	return SetAutoStart(enable)
}

// GetVersion returns the current app version
func (a *App) GetVersion() string {
	return AppVersion
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) GetPrinters() ([]printer.PrinterInfo, error) {
	a.printerMutex.RLock()
	if len(a.printerCache) > 0 {
		defer a.printerMutex.RUnlock()
		return a.printerCache, nil
	}
	a.printerMutex.RUnlock()

	return a.RefreshPrinters()
}

func (a *App) RefreshPrinters() ([]printer.PrinterInfo, error) {
	a.Log("Fetching printer list...")
	printers, err := printer.GetPrinters()
	if err != nil {
		return nil, err
	}

	a.printerMutex.Lock()
	a.printerCache = printers
	a.printerMutex.Unlock()

	return printers, nil
}

func (a *App) GetPrintJobs() []jobs.PrintJob {
	// Periodic logging might be too noisy, so maybe only on change?
	// Or just a simple log if explicitly called from frontend manually (but it's called on interval)
	// Let's skip periodic logs to avoid spamming the log window.
	return a.store.GetJobs()
}

// Helper to log from App
func (a *App) Log(msg string) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "backend_log", fmt.Sprintf("[App] %s", msg))
	}
}

type UpdateResponse struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Body      string `json:"body"`
	URL       string `json:"url"`
}

type UpdateStatus struct {
	State         string `json:"state"`
	Message       string `json:"message"`
	Version       string `json:"version"`
	LastCheckedAt string `json:"lastCheckedAt"`
}

func (a *App) CheckForUpdates() (*UpdateResponse, error) {
	release, err := updater.CheckForUpdates(AppVersion, GithubRepo)
	if err != nil {
		return nil, err
	}

	if release != nil {
		downloadURL, assetErr := updater.SelectDownloadURL(release)
		if assetErr != nil {
			if runtime.GOOS == "windows" {
				return nil, assetErr
			}

			downloadURL = ""
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

func (a *App) GetUpdateStatus() UpdateStatus {
	a.updateStateMutex.Lock()
	defer a.updateStateMutex.Unlock()

	return a.updateStatus
}

func (a *App) CheckForUpdatesNow() (UpdateStatus, error) {
	if !a.beginUpdateCheck() {
		return a.GetUpdateStatus(), nil
	}
	defer a.finishUpdateCheck()

	err := a.performUpdateCheck("manual")
	return a.GetUpdateStatus(), err
}

func (a *App) startAutoUpdateChecks() {
	a.updateStateMutex.Lock()
	if a.updateCheckCancel != nil {
		a.updateStateMutex.Unlock()
		return
	}

	checkCtx, cancel := context.WithCancel(context.Background())
	a.updateCheckCancel = cancel
	a.updateCheckDone = make(chan struct{})
	done := a.updateCheckDone
	a.updateStateMutex.Unlock()

	a.Log("Automatic update checks started. Checking every 30 minutes.")

	go func() {
		defer close(done)
		a.runScheduledUpdateCheck()

		ticker := time.NewTicker(updateCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-checkCtx.Done():
				return
			case <-ticker.C:
				a.runScheduledUpdateCheck()
			}
		}
	}()
}

func (a *App) stopAutoUpdateChecks() {
	a.updateStateMutex.Lock()
	cancel := a.updateCheckCancel
	done := a.updateCheckDone
	a.updateCheckCancel = nil
	a.updateCheckDone = nil
	a.updateStateMutex.Unlock()

	if cancel == nil {
		return
	}

	cancel()
	if done != nil {
		<-done
	}
}

func (a *App) runScheduledUpdateCheck() {
	if !a.beginUpdateCheck() {
		return
	}
	defer a.finishUpdateCheck()

	if err := a.performUpdateCheck("automatic"); err != nil {
		a.Log(fmt.Sprintf("Automatic update flow failed: %v", err))
	}
}

func (a *App) performUpdateCheck(trigger string) error {
	a.setUpdateStatus("checking", updateCheckMessage(trigger), "", false)

	update, err := a.CheckForUpdates()
	if err != nil {
		message := fmt.Sprintf("Update check failed: %v", err)
		a.setUpdateStatus("error", message, "", true)
		return err
	}

	if !update.Available {
		a.setUpdateStatus("up_to_date", "You already have the latest version.", "", true)
		return nil
	}

	a.Log(fmt.Sprintf("Update %s is available.", update.Version))
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "update_available", update)
	}

	if runtime.GOOS != "windows" {
		a.setUpdateStatus("update_available", fmt.Sprintf("Update %s is available. Automatic install is supported on Windows only.", update.Version), update.Version, true)
		return nil
	}

	if update.URL == "" {
		err = fmt.Errorf("no installer asset found for update %s", update.Version)
		a.setUpdateStatus("error", err.Error(), update.Version, true)
		return err
	}

	return a.installUpdate(update)
}

func (a *App) beginUpdateCheck() bool {
	a.updateStateMutex.Lock()
	defer a.updateStateMutex.Unlock()

	if a.isCheckingUpdates || a.isInstallingUpdate {
		return false
	}

	a.isCheckingUpdates = true
	return true
}

func (a *App) finishUpdateCheck() {
	a.updateStateMutex.Lock()
	a.isCheckingUpdates = false
	a.updateStateMutex.Unlock()
}

func (a *App) beginUpdateInstall() bool {
	a.updateStateMutex.Lock()
	defer a.updateStateMutex.Unlock()

	if a.isInstallingUpdate {
		return false
	}

	a.isInstallingUpdate = true
	return true
}

func (a *App) finishUpdateInstall() {
	a.updateStateMutex.Lock()
	a.isInstallingUpdate = false
	a.updateStateMutex.Unlock()
}

func (a *App) InstallUpdate(url string) error {
	if url == "" {
		return fmt.Errorf("no download URL provided")
	}

	update := &UpdateResponse{
		Available: true,
		URL:       url,
	}

	return a.installUpdate(update)
}

func (a *App) installUpdate(update *UpdateResponse) error {
	if update == nil || update.URL == "" {
		return fmt.Errorf("no download URL provided")
	}

	if !a.beginUpdateInstall() {
		return nil
	}

	message := "Downloading update installer..."
	if update.Version != "" {
		message = fmt.Sprintf("Downloading update %s...", update.Version)
	}
	a.setUpdateStatus("downloading", message, update.Version, true)

	err := updater.DownloadAndInstall(update.URL)
	if err != nil {
		a.finishUpdateInstall()
		a.setUpdateStatus("error", fmt.Sprintf("Failed to install update: %v", err), update.Version, true)
		return err
	}

	message = "Installer downloaded. Closing app to continue update..."
	if update.Version != "" {
		message = fmt.Sprintf("Installing update %s. Closing app...", update.Version)
	}
	a.setUpdateStatus("installing", message, update.Version, true)
	a.IsQuitting = true
	wailsRuntime.Quit(a.ctx)
	return nil
}

func (a *App) setUpdateStatus(state string, message string, version string, shouldStampCheckTime bool) {
	a.updateStateMutex.Lock()
	status := a.updateStatus
	status.State = state
	status.Message = message
	status.Version = version
	if shouldStampCheckTime {
		status.LastCheckedAt = time.Now().Format(time.RFC3339)
	}
	a.updateStatus = status
	snapshot := status
	a.updateStateMutex.Unlock()

	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "update_status", snapshot)
	}
}

func updateCheckMessage(trigger string) string {
	if trigger == "manual" {
		return "Checking for updates now..."
	}

	return "Checking for updates in the background..."
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
	printers, err := a.GetPrinters()
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
	fmt.Printf("TestPrint: Generating sample receipt for %s\n", printerName)

	sampleData := receipt.GetSampleOrderData()
	receipt.RenderBill(adapter, sampleData, "80mm") // Defaulting to 80mm for test

	fmt.Printf("TestPrint: Sending %d bytes to printer\n", len(adapter.GetBytes()))
	return printer.PrintRaw(a.ctx, printerName, adapter.GetBytes())
}

func (a *App) ClearPrinterQueue(printerName string) error {
	fmt.Printf("ClearPrinterQueue: Clearing queue for %s\n", printerName)
	return printer.ClearPrinterQueue(a.ctx, printerName)
}
