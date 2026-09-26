//go:build windows

package service

import (
	"github.com/xyztony999/devswitch/internal/apply"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
	"github.com/xyztony999/devswitch/internal/winenv"
)

// writeUserEnvPlatform 写 HKCU\Environment：shim 目录置顶 + 各 *_HOME。
func writeUserEnvPlatform(state *models.State) {
	winenv.EnsurePathEntry(paths.LocalBin(), true)
	for tool, varName := range models.ToolHomeVars {
		if r := state.CurrentRuntime(tool); r != nil {
			winenv.SetValue(varName, apply.WinHome(r.Home))
		}
	}
}
