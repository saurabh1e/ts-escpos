//go:build windows

package server

import (
	"syscall"
)

func playSystemSound() {
	// Play 'SystemAsterisk' sound (0x00000040)
	// Reference: https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-messagebeep
	dll := syscall.NewLazyDLL("user32.dll")
	proc := dll.NewProc("MessageBeep")
	proc.Call(0x40)
}
