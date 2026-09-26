// Package cli 是命令行入口（平移进行中，先提供 --version / list / current）。
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/store"
)

var Version = "2.0.0-dev"

func Main(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}
	switch args[0] {
	case "--version", "version":
		fmt.Println("DevSwitch", Version)
		return 0
	case "current":
		return cmdCurrent()
	case "list":
		return cmdList()
	default:
		fmt.Fprintf(os.Stderr, "未知命令：%s（Go 2.0 平移进行中）\n", args[0])
		printUsage()
		return 2
	}
}

func printUsage() {
	fmt.Println("用法：devswitch <命令> [参数]")
	fmt.Println("命令：scan list current use import install setup uninstall export apply which doctor hook completion update gui")
}

func cmdCurrent() int {
	state := store.Load()
	for _, tool := range models.Tools {
		if r := state.CurrentRuntime(tool); r != nil {
			fmt.Printf("%s  %s  (%s)\n", tool, r.Version, r.Home)
		} else {
			fmt.Printf("%s  (未选择)\n", tool)
		}
	}
	return 0
}

func cmdList() int {
	state := store.Load()
	for _, tool := range models.Tools {
		for _, r := range state.ForTool(tool) {
			mark := " "
			if state.Current[tool] == r.Home {
				mark = "*"
			}
			fmt.Printf("%s %-6s %-12s %-24s %-8s %s\n", mark, tool, r.Version, r.Label, r.Source, r.Home)
		}
	}
	fmt.Println("\n* 表示当前选中。")
	_ = strings.TrimSpace
	return 0
}
