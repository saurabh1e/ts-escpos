//go:build linux

package config

import (
	"fmt"
	"os"
	"strings"
)

func GetMachineID() (string, error) {
	// Try /etc/machine-id
	id, err := os.ReadFile("/etc/machine-id")
	if err == nil {
		return strings.TrimSpace(string(id)), nil
	}
	// Try /var/lib/dbus/machine-id
	id, err = os.ReadFile("/var/lib/dbus/machine-id")
	if err == nil {
		return strings.TrimSpace(string(id)), nil
	}
	return "", fmt.Errorf("failed to read machine-id")
}
