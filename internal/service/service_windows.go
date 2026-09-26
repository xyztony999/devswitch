//go:build windows

package service

import (
	"github.com/xyztony999/devswitch/internal/apply"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
	"github.com/xyztony999/devswitch/internal/winenv"
)

// writeUserEnvPlatform 写 HKCU\Environment：shim 目录置顶 + 各 *_HOME。
// 用 ApplyUserEnv 合并成单次广播，避免逐次写入逐次等待。
func writeUserEnvPlatform(state *models.State) {
	homes := map[string]string{}
	for tool, varName := range models.ToolHomeVars {
		if r := state.CurrentRuntime(tool); r != nil {
			homes[varName] = apply.WinHome(r.Home)
		}
	}
	winenv.ApplyUserEnv(paths.LocalBin(), true, homes)
}
