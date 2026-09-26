//go:build !windows

package main

// defaultToGUI：Linux 无参数打印用法（命令行优先），GUI 经 devswitch gui 进入。
func defaultToGUI() bool { return false }
