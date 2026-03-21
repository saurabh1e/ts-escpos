//go:build darwin

package server

import (
	"fmt"
	"os/exec"
	"strings"
)

func sendSystemNotification(title, message, icon string) error {
	script := fmt.Sprintf(`display notification %s with title %s`, appleScriptString(message), appleScriptString(title))
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func appleScriptString(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return fmt.Sprintf(`"%s"`, replacer.Replace(value))
}
