package printer

import "context"

// Global Logger function for printer package
type PrinterLogger func(ctx context.Context, msg string)

var Logger PrinterLogger

// PrinterInfo holds core information relative to a printer
type PrinterInfo struct {
	Name      string `json:"name"`
	UniqueID  string `json:"uniqueId"`
	WindowsID string `json:"windowsId"`
	Status    string `json:"status"` // "Ready", "Offline", etc.
	IsDefault bool   `json:"isDefault"`
}
