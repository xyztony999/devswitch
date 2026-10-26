// remove：把单个已纳管版本移出清单，可选按红线删除磁盘目录。
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
	"github.com/xyztony999/devswitch/internal/store"
)

// foreignSources：这些来源的安装不由本工具管理，--purge 永不动盘。
var foreignSources = map[string]bool{"system": true, "env": true, "path": true}

// PurgeAllowed 红线判定：目录在用户主目录内，且不是外来安装。
func PurgeAllowed(r *models.Runtime) bool {
	if foreignSources[r.Source] {
		return false
	}
	home := normalizePath(r.Home)
	root := normalizePath(paths.Home())
	if home == root || !strings.HasPrefix(home, root+string(filepath.Separator)) {
		return false
	}
	return true
}

func normalizePath(p string) string {
	return strings.ToLower(filepath.Clean(p))
}

// RemoveRuntime 移除一个版本实例。
// purge=true 时删除磁盘目录（须 PurgeAllowed 且 confirmed）；
// confirmed=false 且需要确认时返回错误（state 不变）。
// 返回值 degraded=true 表示外来安装/目录越界的 --purge 被降级为仅移清单。
func RemoveRuntime(tool models.Tool, query string, purge, confirmed bool) (*models.Runtime, bool, error) {
	state := store.Load()
	runtime := state.Find(tool, query)
	if runtime == nil {
		state = MergeScan()
		runtime = state.Find(tool, query)
	}
	if runtime == nil {
		return nil, false, fmt.Errorf("没有找到 %s 版本：%s", tool, query)
	}
	if cur := state.CurrentRuntime(tool); cur != nil &&
		strings.TrimRight(cur.Home, "/\\") == strings.TrimRight(runtime.Home, "/\\") {
		return nil, false, fmt.Errorf(
			"%s 是当前激活版本，请先 devswitch use %s 切换到其他版本后再移除", runtime.Home, tool)
	}

	degraded := false
	willDelete := purge && PurgeAllowed(runtime)
	if purge && !willDelete {
		degraded = true // 外来安装或目录在用户主目录外：降级为仅移清单
	}
	if willDelete && !confirmed {
		return nil, false, fmt.Errorf(
			"--purge 需要确认：将删除目录 %s（交互确认或 --yes）", runtime.Home)
	}

	// 先移出并保存（删盘失败不回滚清单）
	kept := state.Runtimes[:0]
	for _, r := range state.Runtimes {
		if r.Tool == tool && strings.TrimRight(r.Home, "/\\") == strings.TrimRight(runtime.Home, "/\\") {
			continue
		}
		kept = append(kept, r)
	}
	state.Runtimes = kept
	if err := store.Save(state); err != nil {
		return nil, false, err
	}

	if willDelete {
		if err := os.RemoveAll(runtime.Home); err != nil {
			return runtime, degraded, fmt.Errorf("已移出清单，但删除目录失败（%v），请手动处理：%s", err, runtime.Home)
		}
	}

	applyAll(&state)
	return runtime, degraded, nil
}
