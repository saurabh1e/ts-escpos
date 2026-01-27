//go:build windows

package config

import (
	"os/exec"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func GetMachineID() (string, error) {
	// 1. Try WMIC (Hardware UUID)
	// cmd: wmic csproduct get UUID
	out, err := exec.Command("wmic", "csproduct", "get", "UUID").Output()
	if err == nil {
		lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && trimmed != "UUID" {
				return trimmed, nil
			}
		}
	}

	// 2. Fallback: Registry MachineGuid (Software UUID)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	guid, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", err
	}
	return guid, nil
}
