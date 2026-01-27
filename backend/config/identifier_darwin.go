//go:build darwin

package config

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func GetMachineID() (string, error) {
	// ioreg -d2 -c IOPlatformExpertDevice | awk -F\" '/IOPlatformUUID/{print $(NF-1)}'
	cmd := exec.Command("ioreg", "-d2", "-c", "IOPlatformExpertDevice")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}

	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				// "IOPlatformUUID" = "XXXXX..."
				id := strings.TrimSpace(parts[1])
				id = strings.Trim(id, "\"")
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("failed to find IOPlatformUUID")
}
