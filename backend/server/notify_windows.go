//go:build windows

package server

func sendSystemNotification(title, message, icon string) error {
	return nil
}
