//go:build !gui

// Package gui 的默认构建存根：GUI 需要 -tags gui 专用构建
// （webview/systray 依赖），CLI 构建保持零依赖纯净。
package gui

import "fmt"

func Run() int {
	fmt.Println("此构建未包含图形界面。请使用带 GUI 的发布包，或用 -tags gui 自行构建。")
	fmt.Println("命令行功能不受影响：devswitch list / use / install / doctor")
	return 1
}
