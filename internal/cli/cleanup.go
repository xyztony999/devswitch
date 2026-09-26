// cleanup 子命令：移除 DevSwitch 在用户级生成的全部配置
// （shim、用户 PATH 项、*_HOME 用户变量、shell 钩子、生成的 env 文件）。
// 状态（state.json）与已下载的运行时目录属于用户数据，保留不动。
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xyztony999/devswitch/internal/apply"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
	"github.com/xyztony999/devswitch/internal/winenv"
)

func cmdCleanup() int {
	removed := 0

	// 1. shim（含 Windows 的 .cmd 与无扩展 twin）
	if entries, err := os.ReadDir(paths.LocalBin()); err == nil {
		for _, entry := range entries {
			full := filepath.Join(paths.LocalBin(), entry.Name())
			if apply.IsOurShim(full) {
				if os.Remove(full) == nil {
					removed++
				}
			}
		}
	}

	// 2. shell 钩子（bashrc / profile / PowerShell profile / Git Bash bashrc）
	for _, target := range paths.HookTargets() {
		if apply.RemoveHook(target) {
			removed++
		}
	}

	// 3. 生成的 env 文件
	for _, f := range []string{paths.CurrentEnv(), paths.EnvFile()} {
		if os.Remove(f) == nil {
			removed++
		}
	}

	// 4. Windows 用户 PATH 项与 *_HOME 用户变量
	if paths.IsWindows {
		winenv.RemovePathEntry(paths.LocalBin())
		for _, varName := range models.ToolHomeVars {
			winenv.DeleteValue(varName)
		}
	}

	fmt.Println("已清理 DevSwitch 的用户级配置（shim / PATH / 环境变量 / shell 钩子）。")
	fmt.Printf("版本选择与已下载的运行时未动：可用 `devswitch scan` 重新接管，或手动删除 %s\n", paths.DataDir())
	return 0
}
