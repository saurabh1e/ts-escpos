//go:build windows

package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	id, err := resolveMachineID()
	if err != nil {
		return "", err
	}
	cachedMachineID = id
	return cachedMachineID, nil
}

// MachineGuid is unique per Windows install. The SMBIOS UUID is only a
// fallback: cheap boards ship a placeholder shared by every unit, and wmic is
// gone from Windows 11 24H2 onwards.
func resolveMachineID() (string, error) {
	guid, guidErr := readMachineGUID()
	if guidErr == nil && isUsableMachineID(guid) {
		return guid, nil
	}
	fmt.Printf("[MachineID] MachineGuid unusable (%q, err: %v), trying SMBIOS UUID\n", guid, guidErr)

	hardwareID := readSMBIOSUUID()
	if isUsableMachineID(hardwareID) {
		return hardwareID, nil
	}
	fmt.Printf("[MachineID] SMBIOS UUID unusable (%q), using generated ID\n", hardwareID)

	return loadOrCreateMachineID(generatedMachineIDPath())
}

func readMachineGUID() (string, error) {
	// WOW64_64KEY: a 32-bit build on 64-bit Windows is otherwise redirected to
	// WOW6432Node, which has no MachineGuid.
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}
	defer k.Close()

	val, _, err := k.GetStringValue("MachineGuid")
	return strings.TrimSpace(val), err
}

func readSMBIOSUUID() string {
	out, err := exec.Command("wmic", "csproduct", "get", "UUID").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != "UUID" {
			return trimmed
		}
	}
	return ""
}

// ProgramData rather than the per-user config dir: the ID belongs to the PC,
// so every Windows user on it must get the same one, and it must not roam.
func generatedMachineIDPath() string {
	if base := os.Getenv("ProgramData"); base != "" {
		return filepath.Join(base, "ts-escpos", "machine-id")
	}
	return filepath.Join(configDir, "machine-id")
}
