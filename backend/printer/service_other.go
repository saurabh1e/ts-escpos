//go:build !windows

package printer

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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

func PrintRaw(ctx context.Context, printerName string, data []byte) error {
	msg := fmt.Sprintf("[Printer] Printing %d bytes to '%s' via lp", len(data), printerName)
	logToFrontend(ctx, msg)

	// lp -d <printer> -o raw
	cmd := exec.Command("lp", "-d", printerName, "-o", "raw")
	cmd.Stdin = bytes.NewReader(data)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := fmt.Sprintf("[Printer] Error printing to '%s': %v. Output: %s", printerName, err, string(output))
		logToFrontend(ctx, errMsg)
		return fmt.Errorf("failed to print: %v, output: %s", err, string(output))
	}
	successMsg := fmt.Sprintf("[Printer] Successfully sent job to '%s'. Output: %s", printerName, string(output))
	logToFrontend(ctx, successMsg)
	return nil
}

func ClearPrinterQueue(ctx context.Context, printerName string) error {
	msg := fmt.Sprintf("[Printer] Clearing queue for '%s'", printerName)
	logToFrontend(ctx, msg)

	// cancel -a <printer>
	cmd := exec.Command("cancel", "-a", printerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try lprm if cancel fails
		// lprm -P <printer> -
		cmd2 := exec.Command("lprm", "-P", printerName, "-")
		output2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			errMsg := fmt.Sprintf("[Printer] Failed to clear queue for '%s': %v / %v. Output: %s / %s", printerName, err, err2, string(output), string(output2))
			logToFrontend(ctx, errMsg)
			return fmt.Errorf("failed to clear queue: %v", err)
		}
		output = output2
	}

	successMsg := fmt.Sprintf("[Printer] Queue cleared for '%s'. Output: %s", printerName, string(output))
	logToFrontend(ctx, successMsg)
	return nil
}

func logToFrontend(ctx context.Context, msg string) {
	fmt.Println(msg)
	if ctx != nil {
		runtime.EventsEmit(ctx, "backend_log", msg)
	}
}
