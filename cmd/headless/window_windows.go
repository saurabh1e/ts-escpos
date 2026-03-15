package main

import (
	"syscall"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	user32           = syscall.NewLazyDLL("user32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	showWindow       = user32.NewProc("ShowWindow")
)

const SW_HIDE = 0

func HideConsole() {
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		showWindow.Call(hwnd, uintptr(SW_HIDE))
	}
}
