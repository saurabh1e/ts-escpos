//go:build darwin

package server

import "os/exec"

func playSystemSound() {
	// Play a system sound on macOS
	// Pre-installed sounds are usually in /System/Library/Sounds
	// 'Glass' or 'Ping' are common notifications
	exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Run()
}
