//go:build gui && windows

package gui

import (
	"syscall"
	"unsafe"
)

var procShellExecuteW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

// openURL 用系统默认浏览器打开（ShellExecuteW 纯 API，不派生子进程）。
func openURL(url string) error {
	verb, _ := syscall.UTF16PtrFromString("open")
	target, _ := syscall.UTF16PtrFromString(url)
	rc, _, _ := procShellExecuteW.Call(
		0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(target)), 0, 0, 1 /*SW_SHOWNORMAL*/)
	if rc <= 32 {
		return syscall.GetLastError()
	}
	return nil
}
