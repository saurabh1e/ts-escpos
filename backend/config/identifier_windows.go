//go:build windows

package config

import (
	"os/exec"
	"strings"
	"sync"
	"syscall"

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

	// 1. Try WMIC (Hardware UUID)
	// cmd: wmic csproduct get UUID
	cmd := exec.Command("wmic", "csproduct", "get", "UUID")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && trimmed != "UUID" {
				cachedMachineID = trimmed
				return cachedMachineID, nil
			}
		}
	}

	// 2. Fallback: Registry MachineGuid (Software UUID)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	val, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", err
	}

	cachedMachineID = val
	return cachedMachineID, nil

	defer k.Close()

	guid, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", err
	}
	return guid, nil
}
