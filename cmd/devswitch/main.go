// DevSwitch 2.0 — Go 实现。
// 平移自 Python 版（行为以 tests/golden 为规格），模块划分一一对应：
//   models / paths / store / apply / detect / service / downloader / cli。
package main

import (
	"os"

	"github.com/xyztony999/devswitch/internal/cli"
	"github.com/xyztony999/devswitch/internal/gui"
)

func main() {
	args := os.Args[1:]
	attachParentConsole() // windowsgui 子系统下把 CLI 输出接回终端；其他平台空操作
	if len(args) == 0 && defaultToGUI() {
		os.Exit(gui.Run())
	}
	os.Exit(cli.Main(args))
}
