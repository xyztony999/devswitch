//go:build gui && !windows

package gui

import "os/exec"

// openURL 交给桌面环境的默认打开器（xdg-open）。
// URL 已在 uicore 层做白名单校验，命令名固定、不含用户输入拼接。
func openURL(url string) error {
	return exec.Command("xdg-open", url).Start()
}
