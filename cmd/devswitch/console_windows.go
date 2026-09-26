//go:build windows

package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

const attachParentProcess = ^uintptr(0) // ATTACH_PARENT_PROCESS (DWORD)-1

var procAttachConsole = syscall.NewLazyDLL("kernel32.dll").NewProc("AttachConsole")

// attachParentConsole 在 windowsgui 子系统下把 stdio 接回父控制台。
// 仅当本进程没有继承到有效标准句柄时才附加（交互式 cmd/PowerShell 场景）：
// 从 bash/管道/重定向启动时句柄已经可用，绝不能覆盖，否则输出会丢。
func attachParentConsole() {
	if hasStdHandle(windows.STD_OUTPUT_HANDLE) || hasStdHandle(windows.STD_ERROR_HANDLE) {
		return
	}
	if r1, _, _ := procAttachConsole.Call(attachParentProcess); r1 == 0 {
		return // 父进程没有控制台（双击 / Explorer / 安装器启动）
	}
	if f, err := os.OpenFile("CONOUT$", os.O_RDWR, 0); err == nil {
		os.Stdout = f
		os.Stderr = f
	}
	if f, err := os.OpenFile("CONIN$", os.O_RDWR, 0); err == nil {
		os.Stdin = f
	}
}

func hasStdHandle(id uint32) bool {
	h, err := windows.GetStdHandle(id)
	return err == nil && h != 0 && h != windows.InvalidHandle
}
