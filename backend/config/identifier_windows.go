//go:build windows

package config

import (
	"os/exec"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"
)

var (
	cachedMachineID string
	machineIDMu     sync.Mutex
)

func GetMachineID() (string, error) {
	machineIDMu.Lock()
	defer machineIDMu.Unlock()

	if cachedMachineID != "" {
		return cachedMachineID, nil
	}

	// Method 1: Use Registry MachineGuid (Software UUID)
	// This is much faster than wmic and avoids opening a console window (conhost.exe).
	var machineID string
	var regErr error

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		val, _, errReg := k.GetStringValue("MachineGuid")
		if errReg == nil && val != "" {
			machineID = val
		} else {
			regErr = errReg
		}
	} else {
		regErr = err
	}

	// Method 2: Fallback to wmic (Hardware UUID)
	if machineID == "" {
		cmd := exec.Command("wmic", "csproduct", "get", "UUID")
		// Hide window
		// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // Add if needed, requires syscall import
		output, errWmic := cmd.Output()
		if errWmic == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && trimmed != "UUID" {
					machineID = trimmed
					break
				}
			}
		}
	}

	if machineID == "" {
		if regErr != nil {
			return "", regErr
		}
		return "", registry.ErrNotExist
	}

	cachedMachineID = machineID
	return cachedMachineID, nil
}
