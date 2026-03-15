//go:build !windows

package main

func HideConsole() {
	// No-op on non-Windows
}

func DisableCloseButton() {
	// No-op on non-Windows
}
