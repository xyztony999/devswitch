package apply

import (
	"os"
	"regexp"
	"strings"

	"github.com/xyztony999/devswitch/internal/paths"
)

// 项目级自动切换钩子（独立于主钩子块）：
// 交互 shell 的 PROMPT_COMMAND 在每次提示符前运行 env-project，
// 在当前会话内 eval 项目所需版本（direnv 语义，不写 state.json）。
// 无标记文件或目录未变时输出为空，几乎零开销。
func HookSnippetAuto() string {
	return strings.Join([]string{
		paths.HookAutoBegin,
		"_devswitch_autoswitch() {",
		`  local out`,
		`  out="$(devswitch env-project 2>/dev/null)" && [ -n "$out" ] && eval "$out"`,
		`  return 0`,
		"}",
		`case "$-" in *i*)`,
		`  PROMPT_COMMAND="_devswitch_autoswitch${PROMPT_COMMAND:+;$PROMPT_COMMAND}"`,
		`  ;;`,
		"esac",
		paths.HookAutoEnd,
		"",
	}, "\n")
}

// InstallHookAuto 写入/更新自动切换块（不含主钩子）。
func InstallHookAuto(path string) (bool, error) {
	snippet := strings.TrimRight(HookSnippetAuto(), "\n")
	text := ""
	if data, err := os.ReadFile(path); err == nil {
		text = string(data)
	}
	re := regexp.MustCompile(regexp.QuoteMeta(paths.HookAutoBegin) + `(?s).*?` + regexp.QuoteMeta(paths.HookAutoEnd))
	if re.MatchString(text) {
		updated := re.ReplaceAllString(text, snippet)
		if !strings.HasSuffix(updated, "\n") {
			updated += "\n"
		}
		return false, writeFile(path, updated)
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return true, writeFile(path, text+"\n"+snippet+"\n")
}

// RemoveHookAuto 删除自动切换块。
func RemoveHookAuto(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), paths.HookAutoBegin) {
		return false
	}
	re := regexp.MustCompile(`\n?` + regexp.QuoteMeta(paths.HookAutoBegin) + `(?s).*?` + regexp.QuoteMeta(paths.HookAutoEnd) + `\n?`)
	updated := re.ReplaceAllString(string(data), "")
	if !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	return writeFile(path, updated) == nil
}
