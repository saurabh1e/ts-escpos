//go:build !windows && !darwin

package server

import "os/exec"

func sendSystemNotification(title, message, icon string) error {
	cmd := exec.Command("notify-send", title, message)
	return cmd.Run()
}
