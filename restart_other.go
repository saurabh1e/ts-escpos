//go:build !windows

package main

// ConfigureAutoRestart is a no-op on non-Windows systems
func ConfigureAutoRestart() {
	// Not supported/needed for dev environment
}
