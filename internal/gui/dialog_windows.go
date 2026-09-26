//go:build gui && windows

package gui

import (
	"syscall"
	"unsafe"
)

// chooseFolderNative：系统文件夹选择对话框（SHBrowseForFolder，shell32）。
// 一次调用、结构简单可靠；BIF_NEWDIALOGSTYLE 提供可缩放的新版树形界面。
var (
	shell32Browse      = syscall.NewLazyDLL("shell32.dll")
	procSHBrowse       = shell32Browse.NewProc("SHBrowseForFolderW")
	procSHGetPathFromID = shell32Browse.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFreeD = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")
)

const (
	bifReturnOnlyFSDirs = 0x1
	bifNewDialogStyle   = 0x40
	bifStatusText       = 0x4
)

type browseInfo struct {
	owner       uintptr
	pidlRoot    uintptr
	displayName *uint16
	title       *uint16
	flags       uint32
	callback    uintptr
	lParam      uintptr
	image       int32
}

func chooseFolderNative(title string) (string, error) {
	display := make([]uint16, 260)
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		titlePtr, _ = syscall.UTF16PtrFromString("选择文件夹")
	}
	bi := browseInfo{
		owner:       ownerHwnd(), // 以主窗口为父，保持模态置前
		displayName: &display[0],
		title:       titlePtr,
		flags:       bifReturnOnlyFSDirs | bifNewDialogStyle | bifStatusText,
	}
	pidl, _, _ := procSHBrowse.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", nil // 用户取消
	}
	defer procCoTaskMemFreeD.Call(pidl)

	path := make([]uint16, 260)
	ok, _, _ := procSHGetPathFromID.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return "", nil
	}
	return syscall.UTF16ToString(path), nil
}

// ownerHwnd 找到 DevSwitch 主窗口作为对话框宿主；找不到就退化为无宿主。
func ownerHwnd() uintptr {
	cls, _ := syscall.UTF16PtrFromString("webview")
	cap, _ := syscall.UTF16PtrFromString("DevSwitch")
	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(cap)))
	return hwnd
}
