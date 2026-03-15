//go:build !windows

package main

func HideConsole() {
	// No-op on non-Windows
}
