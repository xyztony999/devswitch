//go:build gui && !windows

package gui

import (
	"errors"
	"os/exec"
)

// chooseFolderNative：优先 zenity，其次 kdialog（主流发行版二选一）。
// 命令名固定、无用户输入拼接；两者都没有时回退错误，由 UI 提示走 CLI。
func chooseFolderNative(title string) (string, error) {
	out, err := exec.Command("zenity", "--file-selection", "--directory", "--title="+title).Output()
	switch code(err) {
	case 0:
		return trimDialogOutput(out), nil
	case 1:
		return "", nil // 用户取消
	}
	out, err = exec.Command("kdialog", "--getexistingdirectory", "--title", title).Output()
	switch code(err) {
	case 0:
		return trimDialogOutput(out), nil
	case 1, 2:
		return "", nil // 用户取消
	}
	return "", errors.New("系统没有可用的文件夹选择器（zenity / kdialog），请在命令行执行：devswitch import <工具> <目录>")
}

// code：err 为 nil 返回 0；*exec.ExitError 返回退出码；可执行不存在返回 127。
func code(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 127
}

func trimDialogOutput(b []byte) string {
	s := string(b)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
