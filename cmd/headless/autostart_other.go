//go:build !windows

package main

func SetAutoStart(enable bool) error {
	// Not implemented for non-Windows platforms
	return nil
}
