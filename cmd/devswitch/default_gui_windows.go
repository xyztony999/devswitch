//go:build windows

package main

// defaultToGUI：Windows 双击运行（无参数）直接进图形界面，更符合桌面习惯；
// 终端内 `devswitch` 不带参数仍打印用法（见 cli.Main）——本函数只在无参数时
// 生效，而终端用户通常会带子命令。
func defaultToGUI() bool { return true }
