package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"ts-escpos/backend/config"
	"ts-escpos/backend/jobs"
	"ts-escpos/backend/printer"
	"ts-escpos/backend/server"
	"ts-escpos/backend/updater"
)

var AppVersion = "0.0.19"

const GithubRepo = "saurabh1e/ts-escpos"
const updateCheckInterval = 30 * time.Minute

func main() {
	// Disable the Close button on the console window immediately
	// This prevents accidental closure via the X button.
	DisableCloseButton()

	// Simple command line argument handling
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--install-autostart" || arg == "-install" {
			fmt.Println("Installing auto-start entry...")
			if err := SetAutoStart(true); err != nil {
				fmt.Printf("Failed to set auto-start: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Auto-start installed successfully.")
			return // Exit after installation
		}
		if arg == "--remove-autostart" || arg == "-uninstall" {
			fmt.Println("Removing auto-start entry...")
			if err := SetAutoStart(false); err != nil {
				fmt.Printf("Failed to remove auto-start: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Auto-start removed successfully.")
			return // Exit after removal
		}
		if arg == "--background" || arg == "-bg" {
			fmt.Println("Running in background mode (console hidden)...")
			HideConsole()
		}
	}

	// Prevent immediate closure on panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("CRITICAL ERROR: %v\n", r)
			fmt.Println("Stack Trace:")
			debug.PrintStack()
			fmt.Println("\nPress Enter to exit...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
			os.Exit(1)
		}
	}()

	fmt.Printf("Starting TS-ESCPOS Headless Server (v%s)...\n", AppVersion)

	// Load configuration
	cfg := config.LoadConfig()
	fmt.Printf("Loaded config: Port %d\n", cfg.HTTPPort)

	// Fetch & Print Machine ID
	machineID, err := config.GetMachineID()
	if err != nil {
		fmt.Printf("WARNING: Failed to retrieve Machine ID: %v\n", err)
	} else {
		fmt.Printf("Machine ID: %s\n", machineID)
	}

	// Set printer logger
	printer.Logger = func(ctx context.Context, msg string) {
		fmt.Printf("[PRINTER] %s\n", msg)
	}

	// Initialize Job Store
	store := jobs.NewStore()

	// Initialize Server
	srv := server.NewServer(store, cfg)

	// Set a simple logger for events
	srv.SetEventEmitter(func(ctx context.Context, eventName string, optionalData ...interface{}) {
		// Log interesting events to stdout
		if eventName == "backend_log" {
			if len(optionalData) > 0 {
				fmt.Printf("[LOG] %v\n", optionalData[0])
			}
		} else if eventName == "error_notification" {
			fmt.Printf("[ERROR] %v\n", optionalData)
		}
	})

	// Set background context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.SetContext(ctx)

	// Start Server
	// Note: srv.Start() runs in a goroutine, so we need to block main
	srv.Start()

	fmt.Printf("Health check available at: http://localhost:%d/health\n", cfg.HTTPPort)

	// Start auto-update checks
	go startAutoUpdateChecks()

	fmt.Println("Server is running. Press Ctrl+C to stop.")

	// Wait for termination signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Keep running until killed explicitly
	for {
		sig := <-c
		fmt.Printf("\nReceived signal: %v. Ignoring to stay running.\n", sig)
		fmt.Println("Use Task Manager or 'kill' command to terminate properly.")
	}
}

func startAutoUpdateChecks() {
	fmt.Println("Automatic update checks started. Checking every 30 minutes.")
	checkForUpdates()

	ticker := time.NewTicker(updateCheckInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		checkForUpdates()
	}
}

func checkForUpdates() {
	release, err := updater.CheckForUpdates(AppVersion, GithubRepo)
	if err != nil {
		fmt.Printf("Update check failed: %v\n", err)
		return
	}

	if release != nil {
		fmt.Printf("Update available: %s\n", release.TagName)

		// Find the correct asset for this lite version
		targetName := "ts-escpos-lite.exe"
		if runtime.GOARCH == "386" {
			targetName = "ts-escpos-lite-32.exe"
		}

		// If running on non-windows (mac/linux), usually no extension in the release script?
		// but the release script currently outputs just `ts-escpos-lite` for unix.
		if runtime.GOOS != "windows" {
			targetName = "ts-escpos-lite"
		}

		var downloadURL string
		for _, asset := range release.Assets {
			if asset.Name == targetName {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}

		if downloadURL != "" {
			fmt.Printf("Found matching asset: %s\n", targetName)
			fmt.Println("Downloading and applying update...")
			if err := performSelfUpdate(downloadURL); err != nil {
				fmt.Printf("Failed to apply update: %v\n", err)
				fmt.Printf("Please download manually from: https://github.com/%s/releases/latest\n", GithubRepo)
				fmt.Println("\nPress Enter to continue...")
				bufio.NewReader(os.Stdin).ReadBytes('\n')
			} else {
				fmt.Println("Update applied successfully! Restarting...")

				// Get current executable path (which is now the new one)
				exePath, err := os.Executable()
				if err != nil {
					fmt.Printf("Failed to get executable path for restart: %v\n", err)
					os.Exit(0)
				}

				// Launch new process
				cmd := exec.Command(exePath, os.Args[1:]...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				// Detach? For console app we might want to just replace.
				// But Start() is enough if we exit.
				if err := cmd.Start(); err != nil {
					fmt.Printf("Failed to restart: %v\n", err)
				}

				os.Exit(0)
			}
		} else {
			fmt.Printf("No matching lite binary found in release. Please download manually from: https://github.com/%s/releases/latest\n", GithubRepo)
		}
	}
}

func performSelfUpdate(url string) error {
	// 1. Download the new binary
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download bad status: %s", resp.Status)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "ts-escpos-new-*")
	if err != nil {
		return fmt.Errorf("creating temp file failed: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Cleanup if we fail before move

	// Ensure executable permissions if on unix
	if runtime.GOOS != "windows" {
		tmpFile.Chmod(0755)
	}

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close() // Close explicitly
	if err != nil {
		return fmt.Errorf("writing update failed: %v", err)
	}

	// 2. Identify current executable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path failed: %v", err)
	}

	// Resolve symlinks just in case
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolving executable path failed: %v", err)
	}

	// 3. Rename current executable to .old
	// On Windows, you can rename running executables, but not delete them.
	oldPath := exePath + ".old"

	// If .old exists (from previous update), try to remove it
	_ = os.Remove(oldPath)

	if err := os.Rename(exePath, oldPath); err != nil {
		return fmt.Errorf("renaming current executable failed: %v", err)
	}

	// 4. Move new file to executable path
	// Use copy if move fails across devices (though temp is usually same drive on windows if %TEMP% is normal)
	// But os.Rename often fails across volumes.
	// Try Rename first
	err = os.Rename(tmpPath, exePath)
	if err != nil {
		// Fallback to copy
		// We need to re-open the temp file because we closed it
		src, errOpen := os.Open(tmpPath)
		if errOpen != nil {
			// Try to revert the rename
			_ = os.Rename(oldPath, exePath)
			return fmt.Errorf("moving new executable failed (and cannot open temp): %v", err)
		}
		defer src.Close()

		dst, errCreate := os.Create(exePath)
		if errCreate != nil {
			// Try to revert
			_ = os.Rename(oldPath, exePath)
			return fmt.Errorf("creating new executable file failed: %v", errCreate)
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			_ = os.Rename(oldPath, exePath)
			return fmt.Errorf("copying content to new executable failed: %v", err)
		}

		if runtime.GOOS != "windows" {
			dst.Chmod(0755)
		}
	}

	// Explicitly allow the temp file helper to cleanup via defer os.Remove(tmpPath) (it's okay if it fails because we renamed it or copied it)
	// If we renamed it, Remove fails (generic error), which is fine.

	return nil
}
