//go:build gui && !windows

package gui

// 标题栏图标覆盖是 Windows 特有需求（webview.h 写死默认图标）；其他平台无此问题。
func setWindowIcon(string) {}
