package tray

import (
	"context"
	"fmt"
)

func (t *TrayApp) Start(ctx context.Context) {
	fmt.Println("System Tray is disabled on macOS to prevent duplicate symbol linker errors with Wails.")
}
