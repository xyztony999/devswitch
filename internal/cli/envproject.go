// env-project 子命令：输出项目环境的会话级 POSIX 片段（自动切换钩子 eval 用）。
package cli

import (
	"fmt"
	"os"

	"github.com/xyztony999/devswitch/internal/service"
)

func cmdEnvProject(args []string) int {
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}
	out, warnings := service.ProjectEnv(dir)
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}
	if out != "" {
		fmt.Println(out)
	}
	return 0
}
