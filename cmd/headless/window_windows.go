package main

import (
	"syscall"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	user32           = syscall.NewLazyDLL("user32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	showWindow       = user32.NewProc("ShowWindow")
	getSystemMenu    = user32.NewProc("GetSystemMenu")
	enableMenuItem   = user32.NewProc("EnableMenuItem")
)

const (
	SW_HIDE      = 0
	SC_CLOSE     = 0xF060
	MF_BYCOMMAND = 0x00000000
	MF_GRAYED    = 0x00000001
	MF_DISABLED  = 0x00000002
)

func HideConsole() {
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		showWindow.Call(hwnd, uintptr(SW_HIDE))
	}
}

func DisableCloseButton() {
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		hMenu, _, _ := getSystemMenu.Call(hwnd, uintptr(0)) // Pass FALSE (0)
		if hMenu != 0 {
			// SC_CLOSE = 0xF060
			// MF_BYCOMMAND = 0x00000000
			// MF_GRAYED = 0x00000001
			enableMenuItem.Call(hMenu, uintptr(SC_CLOSE), uintptr(MF_BYCOMMAND|MF_GRAYED))
		}
	}
}
