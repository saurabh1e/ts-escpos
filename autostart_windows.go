package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func SetAutoStart(enable bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err := filepath.Abs(exe)
	if err != nil {
		return err
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	appName := "ts-escpos"

	if enable {
		return k.SetStringValue(appName, exePath)
	} else {
		// Check if exists before deleting to avoid error
		_, _, err := k.GetStringValue(appName)
		if err == nil {
			return k.DeleteValue(appName)
		}
		return nil
	}
}
