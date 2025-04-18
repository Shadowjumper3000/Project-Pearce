//go:build windows
// +build windows

package main

import (
	"syscall"
)

var (
    user32 = syscall.NewLazyDLL("user32.dll")
    kernel32 = syscall.NewLazyDLL("kernel32.dll")
    
    getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
    showWindow = user32.NewProc("ShowWindow")
)

const SW_HIDE = 0

// This will prevent the command window from appearing
func init() {
    // Hide console window
    hwnd, _, _ := getConsoleWindow.Call()
    if hwnd != 0 {
        showWindow.Call(hwnd, SW_HIDE)
    }
}