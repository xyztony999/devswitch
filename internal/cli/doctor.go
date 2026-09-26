// doctor 子命令：检查 PATH / 注册表 / shim / 钩子的冲突配置。
// --fix 重建全部生效物并注释手写的 Node PATH 行（先备份）。
package cli

import (
	"fmt"
	"os"

	"github.com/xyztony999/devswitch/internal/service"
)

func cmdDoctor(args []string) int {
	fix := false
	for _, arg := range args {
		switch arg {
		case "--fix", "-f":
			fix = true
		default:
			fmt.Fprintf(os.Stderr, "未知参数：%s\n", arg)
			return 2
		}
	}
	if fix {
		for _, line := range service.DoctorFix() {
			fmt.Println(line)
		}
		return 0
	}
	issues := service.DoctorIssues()
	if len(issues) == 0 {
		fmt.Println("一切正常：PATH、shim 与 shell 钩子均无冲突。")
		return 0
	}
	hasErr := false
	for _, issue := range issues {
		mark := "⚠"
		if issue.Level == "error" {
			mark = "✗"
			hasErr = true
		}
		fmt.Printf("%s [%s] %s\n", mark, issue.Code, issue.Message)
	}
	fmt.Println("\n提示：devswitch doctor --fix 可自动修复多数问题。")
	if hasErr {
		return 1
	}
	return 0
}
