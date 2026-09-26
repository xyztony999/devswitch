// DevSwitch 2.0 — Go 实现。
// 平移自 Python 版（行为以 tests/golden 为规格），模块划分一一对应：
//   models / paths / store / apply / detect / service / downloader / cli。
package main

import (
	"os"

	"github.com/xyztony999/devswitch/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
