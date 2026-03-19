//go:build !windows

package printer

import (
	"bytes"
	"context"
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

	deviceOutput, _ := exec.Command("lpstat", "-v").Output()
	deviceInfoByPrinter := parseDeviceInfo(string(deviceOutput))

	defaultPrinterName, _ := GetDefaultPrinterName()

	printerNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	var printers []PrinterInfo

	for _, name := range printerNames {
		if name == "" {
			continue
		}

		deviceInfo := deviceInfoByPrinter[name]
		if !IsSupportedPrinter(name, "", deviceInfo) {
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

		isDefault := false
		if defaultPrinterName != "" && name == defaultPrinterName {
			isDefault = true
		}

		printers = append(printers, PrinterInfo{
			Name:      name,
			UniqueID:  name, // CUPS printer name is unique enough for local reference
			WindowsID: name, // Using name as ID
			Status:    status,
			IsDefault: isDefault,
		})
	}

	return printers, nil
}

func parseDeviceInfo(output string) map[string]string {
	deviceInfoByPrinter := make(map[string]string)

	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if !strings.HasPrefix(line, "device for ") {
			continue
		}

		entry := strings.TrimPrefix(line, "device for ")
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}

		printerName := strings.TrimSpace(parts[0])
		deviceInfoByPrinter[printerName] = strings.TrimSpace(parts[1])
	}

	return deviceInfoByPrinter
}

func GetDefaultPrinterName() (string, error) {
	cmd := exec.Command("lpstat", "-d")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default printer")
	}
	// Output format: "system default destination: PrinterName" or "no system default destination"
	str := string(output)
	if strings.Contains(str, "no system default destination") {
		return "", fmt.Errorf("no default printer set")
	}
	parts := strings.Split(str, ":")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1]), nil
	}
	return "", fmt.Errorf("unknown format: %s", str)
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
	if Logger != nil {
		Logger(ctx, msg)
	}
}
