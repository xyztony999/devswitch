// Package service 承载业务编排，平移自 devswitch/service.py。
package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyztony999/devswitch/internal/apply"
	"github.com/xyztony999/devswitch/internal/detect"
	"github.com/xyztony999/devswitch/internal/downloader"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
	"github.com/xyztony999/devswitch/internal/store"
)

func MergeScan() models.State {
	state := store.Load()
	scanned := detect.ScanRuntimes()
	seen := map[string]bool{}
	merged := []models.Runtime{}
	for i := range scanned {
		key := string(scanned[i].Tool) + ":" + strings.TrimRight(scanned[i].Home, "/")
		if seen[key] {
			continue
		}
		seen[key] = true
		if previous := state.ByHome(scanned[i].Tool, scanned[i].Home); previous != nil && previous.Source == "imported" {
			scanned[i].Source = "imported"
			if previous.Label != "" {
				scanned[i].Label = previous.Label
			}
		}
		merged = append(merged, scanned[i])
	}
	for i := range state.Runtimes {
		r := state.Runtimes[i]
		key := string(r.Tool) + ":" + strings.TrimRight(r.Home, "/")
		if seen[key] {
			continue
		}
		if r.Source == "imported" && fileExists(r.Home) {
			inspector := detect.Inspectors()[r.Tool]
			if inspector != nil {
				if refreshed := inspector(r.Home); refreshed != nil {
					refreshed.Source = "imported"
					merged = append(merged, *refreshed)
					seen[key] = true
				}
			}
		}
	}
	state.Runtimes = merged
	for _, tool := range models.Tools {
		if state.CurrentRuntime(tool) == nil {
			if guessed := detect.InferActive(state.Runtimes, tool); guessed != nil {
				state.Current[tool] = guessed.Home
			} else {
				state.Current[tool] = ""
			}
		}
	}
	_ = store.Save(state)
	return state
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Use 切换版本并让全部生效动作落地。
func Use(tool models.Tool, query string) (*models.Runtime, error) {
	state := store.Load()
	runtime := state.Find(tool, query)
	if runtime == nil {
		state = MergeScan()
		runtime = state.Find(tool, query)
	}
	if runtime == nil {
		if len(state.ForTool(tool)) == 0 {
			return nil, fmt.Errorf("本机没有发现 %s。安装后执行 devswitch scan，或：devswitch install %s <版本>", tool, tool)
		}
		return nil, fmt.Errorf("没有找到 %s 版本：%s", tool, query)
	}
	state.Current[tool] = runtime.Home
	_ = store.Save(state)
	applyAll(&state)
	return runtime, nil
}

// applyAll 生成一切生效物（shim/启动器/env/钩子/用户环境变量）。
func applyAll(state *models.State) {
	_ = apply.WriteCurrentEnv(state)
	_ = apply.WriteEnvSh(state)
	_, _ = apply.WriteShims(state)
	_ = apply.WriteLaunchers()
	applyHooks(state)
	ensureProcessPath()
}

// ensureProcessPath 把 shim 目录前置到本进程 PATH：
// 注册表/钩子只对新进程生效，正在运行的进程（尤其 GUI 里点「一键修复」后）
// 不会自动获得，doctor 的会话 PATH 检查会一直报旧状态。这里同步进程内环境。
func ensureProcessPath() {
	localBin := paths.LocalBin()
	parts := filepath.SplitList(os.Getenv("PATH"))
	for i, p := range parts {
		if normFold(p) == normFold(localBin) {
			if i == 0 {
				return
			}
			rest := append(append([]string{}, parts[:i]...), parts[i+1:]...)
			next := append([]string{localBin}, rest...)
			os.Setenv("PATH", strings.Join(next, string(os.PathListSeparator)))
			return
		}
	}
	os.Setenv("PATH", localBin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func applyHooks(state *models.State) {
	for _, target := range paths.HookTargets() {
		_, _ = apply.InstallHook(target)
	}
	writeUserEnv(state)
}

// EnsureApplied 对当前状态重新生成全部生效物。
func EnsureApplied() models.State {
	state := store.Load()
	applyAll(&state)
	return state
}

func writeUserEnv(state *models.State) {
	if !paths.IsWindows {
		return
	}
	writeUserEnvPlatform(state)
}

// ImportPath 导入一个本地安装目录。
func ImportPath(tool models.Tool, path string) (*models.Runtime, error) {
	home, err := filepath.Abs(os.ExpandEnv(path))
	if err != nil {
		home = path
	}
	inspector := detect.Inspectors()[tool]
	if inspector == nil {
		return nil, fmt.Errorf("不支持的工具：%s", tool)
	}
	runtime := inspector(home)
	if runtime == nil {
		return nil, fmt.Errorf("目录里没有可用的 %s：%s", tool, home)
	}
	runtime.Source = "imported"
	state := store.Load()
	state.Upsert(*runtime)
	_ = store.Save(state)
	if state.CurrentRuntime(tool) == nil {
		state.Current[tool] = runtime.Home
		_ = store.Save(state)
		applyAll(&state)
	}
	return runtime, nil
}

var winNodeTargets = map[string]string{
	"node": "node.exe", "npm": "npm.cmd", "npx": "npx.cmd", "corepack": "corepack.cmd",
}
var winBuildtoolTargets = map[string]string{
	"mvn": "mvn.cmd", "mvnDebug": "mvnDebug.cmd", "gradle": "gradle.bat",
}

// WhichBinary 返回当前选中工具的真实二进制 / home。
func WhichBinary(name string) (string, bool) {
	state := store.Load()
	// <tool>-home / <TOOL>_HOME / 大写工具名 → home
	for _, tool := range models.Tools {
		upper := strings.ToUpper(string(tool))
		if name == string(tool)+"-home" || name == upper+"_HOME" || name == upper {
			if r := state.CurrentRuntime(tool); r != nil {
				return r.Home, true
			}
			return "", false
		}
	}
	exists := func(p string) (string, bool) {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, true
		}
		return "", false
	}
	switch name {
	case "node", "npm", "npx", "corepack":
		r := state.CurrentRuntime(models.Node)
		if r == nil {
			return "", false
		}
		if paths.IsWindows {
			return exists(filepath.Join(r.Home, winNodeTargets[name]))
		}
		return exists(filepath.Join(r.Home, "bin", name))
	case "mvn", "mvnDebug", "gradle":
		tool := models.Maven
		if name == "gradle" {
			tool = models.Gradle
		}
		r := state.CurrentRuntime(tool)
		if r == nil {
			return "", false
		}
		if paths.IsWindows {
			return exists(filepath.Join(r.Home, "bin", winBuildtoolTargets[name]))
		}
		return exists(filepath.Join(r.Home, "bin", name))
	}
	r := state.CurrentRuntime(models.Java)
	if r == nil {
		return "", false
	}
	exe := name
	if paths.IsWindows {
		exe = name + ".exe"
	}
	for _, rel := range []string{filepath.Join("bin", exe), filepath.Join("jre", "bin", exe)} {
		if p, ok := exists(filepath.Join(r.Home, rel)); ok {
			return p, true
		}
	}
	return "", false
}

// ---------------------------------------------------------------------------
// export / apply
// ---------------------------------------------------------------------------

type devswitchFile struct {
	Devswitch int               `json:"devswitch"`
	Tools     map[string]string `json:"tools"`
}

func ExportVersions(path string) (string, error) {
	state := store.Load()
	tools := map[string]string{}
	for _, tool := range models.Tools {
		if r := state.CurrentRuntime(tool); r != nil {
			tools[string(tool)] = r.Version
		}
	}
	data, err := json.MarshalIndent(devswitchFile{Devswitch: 1, Tools: tools}, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// InstallRuntime 下载安装并自动切换，返回 (home, 完整版本)。
func InstallRuntime(tool models.Tool, version, mirror string) (string, string, error) {
	inspector := detect.Inspectors()[tool]
	if inspector == nil {
		return "", "", fmt.Errorf("暂不支持下载该工具：%s", tool)
	}
	home, err := downloader.InstallTool(tool, version, mirror)
	if err != nil {
		return "", "", err
	}
	runtime := inspector(home)
	if runtime == nil {
		return "", "", fmt.Errorf("下载完成但未能识别安装目录：%s", home)
	}
	_ = MergeScan()
	if _, err := Use(tool, runtime.Version); err != nil {
		return "", "", err
	}
	return runtime.Home, runtime.Version, nil
}

type ApplyResult struct {
	Tool    string
	Version string
	Outcome string
}

func ApplyVersions(path string, installMissing bool, mirror string) ([]ApplyResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取 %s: %v", path, err)
	}
	var payload devswitchFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("无法读取 %s: %v", path, err)
	}
	results := []ApplyResult{}
	// 固定顺序遍历（models.Tools），避免 map 迭代随机
	for _, tool := range models.Tools {
		version, ok := payload.Tools[string(tool)]
		if !ok {
			continue
		}
		if _, err := Use(tool, version); err == nil {
			results = append(results, ApplyResult{string(tool), version, "switched"})
			continue
		}
		if !installMissing {
			results = append(results, ApplyResult{string(tool), version, "missing-version"})
			continue
		}
		major := strings.Split(version, ".")[0]
		if _, _, err := InstallRuntime(tool, major, mirror); err != nil {
			results = append(results, ApplyResult{string(tool), version, "failed: " + err.Error()})
		} else {
			if _, err := Use(tool, version); err != nil {
				results = append(results, ApplyResult{string(tool), version, "failed: " + err.Error()})
			} else {
				results = append(results, ApplyResult{string(tool), version, "installed"})
			}
		}
	}
	return results, nil
}
