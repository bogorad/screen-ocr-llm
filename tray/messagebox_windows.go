//go:build windows

package tray

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	MB_OK                = 0x00000000
	MB_ICONINFORMATION   = 0x00000040
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	procMessageBoxW  = user32.NewProc("MessageBoxW")
)

// showWindowsMessageBox displays a Windows message box
func showWindowsMessageBox(title, message string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	messagePtr, _ := syscall.UTF16PtrFromString(message)
	
	procMessageBoxW.Call(
		0, // hwnd (no parent window)
		uintptr(unsafe.Pointer(messagePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(MB_OK|MB_ICONINFORMATION),
	)
}
