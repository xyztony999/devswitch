//go:build gui && windows

package gui

import (
	"syscall"
	"unsafe"
)

// webview.h 把窗口类图标显式设成了系统默认图标（IDI_APPLICATION），
// exe 内嵌的图标资源因此不会出现在标题栏/任务栏。窗口创建后用
// WM_SETICON 覆盖：图标来自 exe 资源 ID 1（rsrc 注入的 RT_GROUP_ICON）。
var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procFindWindowW  = user32.NewProc("FindWindowW")
	procSendMessageW = user32.NewProc("SendMessageW")
	procLoadImageW   = user32.NewProc("LoadImageW")
)

const (
	wmSetIcon    = 0x0080
	iconSmall    = 0
	iconBig      = 1
	imageIcon    = 1
	lrDefaultClr = 0x00000040
)

func setWindowIcon(title string) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	hInst, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	small, _, _ := procLoadImageW.Call(
		hInst, uintptr(1), imageIcon, 16, 16, lrDefaultClr)
	big, _, _ := procLoadImageW.Call(
		hInst, uintptr(1), imageIcon, 32, 32, lrDefaultClr)
	if small == 0 && big == 0 {
		return
	}
	name, err := syscall.UTF16PtrFromString("webview") // webview.h 的窗口类名
	if err != nil {
		return
	}
	caption, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(caption)))
	if hwnd == 0 {
		return
	}
	if small != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconSmall, small)
	}
	if big != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconBig, big)
	}
}
