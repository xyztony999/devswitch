package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xyztony999/devswitch/internal/apply"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
	"github.com/xyztony999/devswitch/internal/store"
)

// 项目级自动切换：读项目目录的 .devswitch / .nvmrc，
// 输出在当前 shell 会话内生效的 POSIX 片段（direnv 语义）。
// 绝不写 state.json——全局选择只能由 use/cleanup 改变。

var devswitchFileLineRe = regexp.MustCompile(`^\s*([a-z]+):\s*(\S+)`)

// projectMarker 在 dir 及其祖先（最多 3 层）找标记文件。
func projectMarker(dir string) (kind string, path string) {
	for i := 0; i < 3; i++ {
		if dir == "" || dir == "." {
			return "", ""
		}
		if _, err := os.Stat(filepath.Join(dir, ".nvmrc")); err == nil {
			return "nvmrc", filepath.Join(dir, ".nvmrc")
		}
		if _, err := os.Stat(filepath.Join(dir, ".devswitch")); err == nil {
			return "devswitch", filepath.Join(dir, ".devswitch")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ""
		}
		dir = parent
	}
	return "", ""
}

// ProjectEnv 返回会话级环境片段。返回值直接可被 eval：
// 复用 env.sh 已注入的 _devswitch_front（幂等去重前置 PATH），
// 并记录 DEVSWITCH_AUTO_DIR 供同目录 fast-path（连续提示符不重复求值）。
// 提示走 stderr（warnings）；绝不写 state.json。
func ProjectEnv(dir string) (string, []string) {
	warnings := []string{}
	if dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	kind, markerPath := projectMarker(dir)
	if markerPath == "" {
		return "", warnings
	}
	// 同目录 fast-path：上次已求值且目录未变，输出空（省掉重复 eval）
	if auto := os.Getenv("DEVSWITCH_AUTO_DIR"); auto != "" && paths.IsWindows {
		// Windows 大小写不敏感比对
		if strings.EqualFold(filepath.Clean(auto), filepath.Clean(dir)) {
			return "", warnings
		}
	} else if auto != "" && filepath.Clean(auto) == filepath.Clean(dir) {
		return "", warnings
	}
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return "", warnings
	}
	state := store.Load()
	wanted := map[models.Tool]string{}
	switch kind {
	case "nvmrc":
		wanted[models.Node] = strings.TrimSpace(string(data))
	case "devswitch":
		for _, line := range strings.Split(string(data), "\n") {
			if m := devswitchFileLineRe.FindStringSubmatch(line); m != nil {
				if t := models.Tool(m[1]); models.ToolShims[t] != nil {
					wanted[t] = m[2]
				}
			}
		}
	}
	if len(wanted) == 0 {
		return "", warnings
	}
	lines := []string{"# devswitch project env"}
	for _, tool := range models.Tools {
		ver, ok := wanted[tool]
		if !ok || ver == "" {
			continue
		}
		runtime := state.Find(tool, ver)
		if runtime == nil {
			warnings = append(warnings,
				fmt.Sprintf("devswitch: 项目要求 %s %s，未安装（devswitch install %s %s）", tool, ver, tool, ver))
			continue
		}
		upper := strings.ToUpper(string(tool))
		lines = append(lines, fmt.Sprintf("DEVSWITCH_%s_HOME=%q", upper, runtime.Home))
		lines = append(lines, frontBinLines(tool, runtime.Home)...)
	}
	if len(lines) == 1 {
		return "", warnings // 全部缺失：不加片段，提示已有
	}
	lines = append(lines, "export PATH")
	lines = append(lines, fmt.Sprintf("DEVSWITCH_AUTO_DIR=%q", filepath.Clean(dir)))
	return strings.Join(lines, "\n"), warnings
}

// frontBinLines 返回把该工具可执行目录前置到 PATH 的语句。
// 用 _devswitch_front（env.sh 注入的幂等函数：已存在则移到最前），
// 避免每次提示符重复 eval 时无限叠加 PATH。
func frontBinLines(tool models.Tool, home string) []string {
	var bin string
	switch tool {
	case models.Node, models.Golang:
		bin = filepath.Join(home, "bin")
	case models.AndroidSDK:
		bin = filepath.Join(home, "platform-tools")
	case models.Python:
		if paths.IsWindows {
			bin = home // Windows pbs：python.exe 在根
		} else {
			bin = filepath.Join(home, "bin")
		}
	default: // java/maven/gradle
		bin = filepath.Join(home, "bin")
	}
	posix := filepath.ToSlash(bin)
	if paths.IsWindows {
		posix = apply.ToMsysPath(bin)
	}
	return []string{
		`_devswitch_front ` + strconv.Quote(posix),
	}
}
