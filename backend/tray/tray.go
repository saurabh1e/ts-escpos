package tray

import (
	"context"
	_ "embed"
)

type TrayApp struct {
	ctx      context.Context
	iconData []byte
}

func NewTrayApp(iconData []byte) *TrayApp {
	return &TrayApp{
		iconData: iconData,
	}
}
