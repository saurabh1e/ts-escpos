//go:build !windows

package printer

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func GetPrinters() ([]PrinterInfo, error) {
	cmd := exec.Command("lpstat", "-e")
	output, err := cmd.Output()
	if err != nil {
		// If command fails, we might not have cups installed or just no printers
		return []PrinterInfo{}, nil
	}

	printerNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	var printers []PrinterInfo

	for _, name := range printerNames {
		if name == "" {
			continue
		}

		// Get status for each printer
		// lpstat -p <name>
		statusCmd := exec.Command("lpstat", "-p", name)
		statusOut, _ := statusCmd.Output()
		statusStr := string(statusOut)

		status := "Unknown"
		if strings.Contains(statusStr, "is idle") {
			status = "Ready"
		} else if strings.Contains(statusStr, "printing") {
			status = "Printing"
		} else if strings.Contains(statusStr, "disabled") {
			status = "Paused"
		}

		printers = append(printers, PrinterInfo{
			Name:      name,
			UniqueID:  name, // CUPS printer name is unique enough for local reference
			WindowsID: name, // Using name as ID
			Status:    status,
		})
	}

	return printers, nil
}

func PrintRaw(printerName string, data []byte) error {
	// lp -d <printer> -o raw
	cmd := exec.Command("lp", "-d", printerName, "-o", "raw")
	cmd.Stdin = bytes.NewReader(data)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to print: %v, output: %s", err, string(output))
	}
	return nil
}
