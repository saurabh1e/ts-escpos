//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procRegisterApplicationRestart = kernel32.NewProc("RegisterApplicationRestart")
)

// ConfigureAutoRestart registers the application to restart automatically if it crashes or hangs.
// It uses the Windows RegisterApplicationRestart API.
func ConfigureAutoRestart() {
	const (
		// RESTART_NO_CRASH = 1
		// RESTART_NO_HANG = 2
		// RESTART_NO_PATCH = 4
		// RESTART_NO_REBOOT = 8

		// 0 means restart on everything (Crash, Hang, Patch, Reboot)
		RESTART_FLAGS = 0
	)

	// Check if API is available (Vista+)
	if err := procRegisterApplicationRestart.Find(); err != nil {
		return
	}

	// We pass an empty command line string (nil) to restart with same arguments,
	// or we could pass specific flags.
	// The first argument is command line args (pointer to WCHAR), second is flags.

	// Create a UTF16 pointer for empty string (or valid args if needed)
	// Passing nil might reset it, so let's pass empty string.
	emptyStr, _ := syscall.UTF16PtrFromString("")

	procRegisterApplicationRestart.Call(
		uintptr(unsafe.Pointer(emptyStr)),
		uintptr(RESTART_FLAGS),
	)
}
