// remove 子命令：移出版本清单；--purge 按红线删除磁盘目录。
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/xyztony999/devswitch/internal/service"
)

func cmdRemove(args []string) int {
	toolArg, query := "", ""
	purge, yes := false, false
	for _, arg := range args {
		switch arg {
		case "--purge":
			purge = true
		case "--yes", "-y":
			yes = true
		default:
			if toolArg == "" {
				toolArg = arg
			} else if query == "" {
				query = arg
			}
		}
	}
	tool, ok := parseTool(toolArg)
	if !ok || query == "" {
		fmt.Fprintln(os.Stderr, "用法：devswitch remove <tool> <版本|目录> [--purge] [--yes]")
		fmt.Fprintln(os.Stderr, "      默认仅移出清单；--purge 才删除磁盘目录（须确认，且仅限用户目录内）")
		return 2
	}

	confirmed := yes
	if purge && !yes {
		// 交互式二次确认：显示匹配路径，默认 N。
		fmt.Printf("将删除目录：%s\n确认删除？[y/N] ", query)
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		confirmed = strings.HasPrefix(strings.TrimSpace(strings.ToLower(line)), "y")
		if !confirmed {
			fmt.Println("已取消。")
			return 0
		}
	}

	runtime, degraded, err := service.RemoveRuntime(tool, query, purge, confirmed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	label := runtime.Label
	if label == "" {
		label = runtime.Version
	}
	if degraded {
		fmt.Printf("已移除 %s（外来安装或目录在用户主目录外，磁盘未动）。\n", label)
		return 0
	}
	if purge {
		fmt.Printf("已移除 %s，并删除目录 %s。\n", label, runtime.Home)
		return 0
	}
	fmt.Printf("已移除 %s（磁盘未动）。\n", label)
	return 0
}
