package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Register for auto-restart on crash/hang (Windows only)
	ConfigureAutoRestart()

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "ts-escpos",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if app.IsQuitting {
				return false
			}
			// Strictly prevent closing
			runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
				Type:    runtime.InfoDialog,
				Title:   "Restricted",
				Message: "This application runs in the background and cannot be closed.",
			})
			return true
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
