// Doctor：环境诊断，平移自 service.py 的 doctor_issues/doctor_fix。
package service

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xyztony999/devswitch/internal/apply"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
	"github.com/xyztony999/devswitch/internal/store"
	"github.com/xyztony999/devswitch/internal/winenv"
)

type Issue struct {
	Level   string // error / warn / info
	Code    string
	Message string
}

func normFold(s string) string {
	if paths.IsWindows {
		return strings.ToLower(s)
	}
	return s
}

func DoctorIssues() []Issue {
	issues := []Issue{}
	state := store.Load()
	localBin := paths.LocalBin()

	// 本次会话 PATH
	parts := filepath.SplitList(os.Getenv("PATH"))
	found := false
	index := -1
	for i, p := range parts {
		if normFold(p) == normFold(localBin) {
			found = true
			index = i
			break
		}
	}
	if !found {
		issues = append(issues, Issue{"warn", "path-missing",
			localBin + " 不在本次 PATH 里，终端可能仍走系统版本。"})
	} else if index > 0 {
		keywords := []string{"node", "jvm", "jdk", "java"}
		if !paths.IsWindows {
			keywords = []string{"node", "jvm"}
		}
		ahead := []string{}
		for _, p := range parts[:index] {
			lower := strings.ToLower(p)
			for _, kw := range keywords {
				if strings.Contains(lower, kw) {
					ahead = append(ahead, p)
					break
				}
			}
		}
		if len(ahead) > 0 {
			issues = append(issues, Issue{"warn", "path-order",
				"PATH 里有更靠前的 Node/Java 目录，可能抢在 DevSwitch shim 前面：" + strings.Join(ahead, ", ")})
		}
	}

	if paths.IsWindows {
		userHas := false
		for _, entry := range winenv.UserPathEntries() {
			if normFold(entry) == normFold(localBin) {
				userHas = true
				break
			}
		}
		if !userHas {
			issues = append(issues, Issue{"warn", "user-path-missing",
				localBin + " 不在用户 PATH（注册表）里，新开的终端会找不到 shim。可运行 devswitch scan。"})
		}
		for tool, varName := range models.ToolHomeVars {
			r := state.CurrentRuntime(tool)
			if r == nil {
				continue
			}
			regHome, ok := winenv.GetUserValue(varName)
			if !ok || normFold(filepath.Clean(regHome)) != normFold(filepath.Clean(r.Home)) {
				shown := regHome
				if !ok {
					shown = "未设置"
				}
				issues = append(issues, Issue{"warn", string(tool) + "-home-registry",
					"用户环境变量 " + varName + "（注册表）与当前选中版本不一致：" + shown + "。"})
			}
		}
	}

	envProbe := paths.CurrentEnv()
	if paths.IsWindows {
		envProbe = paths.EnvFile()
	}
	if _, err := os.Stat(envProbe); err != nil {
		issues = append(issues, Issue{"error", "no-env", "还没有生成切换配置，先运行 devswitch scan。"})
	}

	probeShims := map[string][]string{
		"node": {"node", "npm"}, "java": {"java", "javac"},
		"maven": {"mvn"}, "gradle": {"gradle"},
	}
	for _, tool := range models.Tools {
		if state.CurrentRuntime(tool) == nil {
			continue
		}
		for _, name := range probeShims[string(tool)] {
			shimName := name
			if paths.IsWindows {
				shimName = name + ".cmd"
			}
			shim := filepath.Join(localBin, shimName)
			if !fileExistsAnywhere(shim) {
				issues = append(issues, Issue{"warn", "no-shim", "缺少 shim：" + shim})
			} else if !apply.IsOurShim(shim) {
				issues = append(issues, Issue{"warn", "foreign-shim", shim + " 还不是 DevSwitch 接管的入口。"})
			}
		}
	}

	hookTargets := paths.HookTargets()
	hookAny := false
	for _, target := range hookTargets {
		if apply.HookInstalled(target) {
			hookAny = true
			break
		}
	}
	if !hookAny {
		if paths.IsWindows {
			issues = append(issues, Issue{"warn", "no-hook",
				"还没有写入 PowerShell / Git Bash 钩子，新开终端可能没有 *_HOME，也可能抢不过机器级安装。"})
		} else {
			issues = append(issues, Issue{"warn", "no-hook",
				"还没有写入 ~/.bashrc 钩子，新开终端可能没有 *_HOME。"})
		}
	}

	for _, tool := range models.Tools {
		if len(state.ForTool(tool)) == 0 {
			issues = append(issues, Issue{"info", "no-runtime",
				"本机没有发现 " + models.ToolLabels[tool] + "。安装后执行扫描，或 devswitch install 下载。"})
		} else if state.CurrentRuntime(tool) == nil {
			issues = append(issues, Issue{"info", "no-current",
				"尚未选择当前 " + models.ToolLabels[tool] + " 版本。"})
		}
	}
	return issues
}

func fileExistsAnywhere(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var nodePathLineRe = regexp.MustCompile(`^(\s*export\s+PATH=.*(?:node-v|/n/versions/node|\.nvm/versions/node).*)`)

// DoctorFix 重建生效物并注释冲突的手写 Node PATH 行。
func DoctorFix() []string {
	actions := []string{}
	EnsureApplied()
	if paths.IsWindows {
		actions = append(actions, "已写入 shim、env.sh、PowerShell / Git Bash 钩子和用户环境变量")
	} else {
		actions = append(actions, "已写入 shim、env.sh 和 shell 钩子")
	}
	changed := []string{}
	for _, target := range []string{paths.Bashrc(), paths.Profile()} {
		changed = append(changed, commentConflictingLines(target)...)
	}
	if len(changed) > 0 {
		actions = append(actions, "已注释冲突 PATH："+strings.Join(changed, "；"))
	}
	return actions
}

func commentConflictingLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	changed := []string{}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if nodePathLineRe.MatchString(trimmed) && !strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			out = append(out, "# DevSwitch: 已改由 shim 接管")
			out = append(out, "# "+trimmed)
			changed = append(changed, strings.TrimSpace(trimmed))
		} else {
			out = append(out, line)
		}
	}
	if len(changed) > 0 {
		_ = os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)
	}
	return changed
}
