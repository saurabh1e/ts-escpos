//go:build windows

package config

import (
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

	// Use Registry MachineGuid (Software UUID)
	// This is much faster than wmic and avoids opening a console window (conhost.exe).
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
}
