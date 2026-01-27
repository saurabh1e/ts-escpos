package printer

// PrinterInfo holds core information relative to a printer
type PrinterInfo struct {
	Name      string `json:"name"`
	UniqueID  string `json:"uniqueId"`
	WindowsID string `json:"windowsId"`
	Status    string `json:"status"` // "Ready", "Offline", etc.
}
