//go:build !darwin

package tray

import (
	"context"

	"github.com/getlantern/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (t *TrayApp) Start(ctx context.Context) {
	t.ctx = ctx
	// Run systray in a goroutine to avoid blocking the Wails main loop.
	// On Windows/Linux this usually works fine.
	go systray.Run(t.onReady, t.onExit)
}

func (t *TrayApp) onReady() {
	systray.SetIcon(t.iconData)
	systray.SetTitle("TS-ESCPOS")
	systray.SetTooltip("TS-ESCPOS Printer Service")

	mShow := systray.AddMenuItem("Show Window", "Show the application window")
	mHide := systray.AddMenuItem("Hide Window", "Hide the application window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				if t.ctx != nil {
					wailsRuntime.WindowShow(t.ctx)
				}
			case <-mHide.ClickedCh:
				if t.ctx != nil {
					wailsRuntime.WindowHide(t.ctx)
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func (t *TrayApp) onExit() {
	if t.ctx != nil {
		wailsRuntime.Quit(t.ctx)
	}
}
