//go:build !windows

package main

func SetAutoStart(enable bool) error {
	return nil
}
