package tray

import (
	"context"
	_ "embed"
)

type TrayApp struct {
	ctx      context.Context
	iconData []byte
	onQuit   func()
}

func NewTrayApp(iconData []byte) *TrayApp {
	return &TrayApp{
		iconData: iconData,
	}
}

func (t *TrayApp) SetOnQuit(fn func()) {
	t.onQuit = fn
}
