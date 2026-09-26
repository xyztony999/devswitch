package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
)

// ShimScript 生成 POSIX sh shim（读 current.env，Linux 语义）。
func ShimScript(kind, name string) string {
	envPath := filepath.ToSlash(paths.CurrentEnv())
	var lookup string
	switch kind {
	case "node":
		lookup = strings.Join([]string{
			fmt.Sprintf(`if [ -n "$DEVSWITCH_NODE_HOME" ] && [ -x "$DEVSWITCH_NODE_HOME/bin/%s" ]; then`, name),
			fmt.Sprintf(`  exec "$DEVSWITCH_NODE_HOME/bin/%s" "$@"`, name),
			"fi",
		}, "\n")
	case "node-tool":
		return strings.Join([]string{
			"#!/bin/sh",
			"# DevSwitch shim — " + name,
			"ENVFILE=" + quote(envPath),
			`[ -f "$ENVFILE" ] && . "$ENVFILE"`,
			fmt.Sprintf(`if [ -n "$DEVSWITCH_NODE_HOME" ] && [ -x "$DEVSWITCH_NODE_HOME/bin/%s" ]; then`, name),
			`  PATH="$DEVSWITCH_NODE_HOME/bin:$PATH"`,
			fmt.Sprintf(`  exec "$DEVSWITCH_NODE_HOME/bin/%s" "$@"`, name),
			"fi",
			`echo "DevSwitch: 当前 Node 没有 ` + name + `。用这套 Node 执行 npm i -g 安装，或切到装过它的版本。" >&2`,
			"exit 127",
			"",
		}, "\n")
	case "maven", "gradle":
		varName := "DEVSWITCH_" + strings.ToUpper(kind) + "_HOME"
		lookup = strings.Join([]string{
			fmt.Sprintf(`if [ -n "$%s" ] && [ -x "$%s/bin/%s" ]; then`, varName, varName, name),
			fmt.Sprintf(`  exec "$%s/bin/%s" "$@"`, varName, name),
			"fi",
			"",
		}, "\n")
	default: // java
		lookup = strings.Join([]string{
			`if [ -n "$DEVSWITCH_JAVA_HOME" ]; then`,
			fmt.Sprintf(`  if [ -x "$DEVSWITCH_JAVA_HOME/bin/%s" ]; then`, name),
			fmt.Sprintf(`    exec "$DEVSWITCH_JAVA_HOME/bin/%s" "$@"`, name),
			fmt.Sprintf(`  elif [ -x "$DEVSWITCH_JAVA_HOME/jre/bin/%s" ]; then`, name),
			fmt.Sprintf(`    exec "$DEVSWITCH_JAVA_HOME/jre/bin/%s" "$@"`, name),
			"  fi",
			"fi",
		}, "\n")
	}
	return strings.Join([]string{
		"#!/bin/sh",
		"# DevSwitch shim — " + name,
		"ENVFILE=" + quote(envPath),
		`[ -f "$ENVFILE" ] && . "$ENVFILE"`,
		lookup,
		`echo "` + missingMessage(kind, name) + `" >&2`,
		"exit 127",
		"",
	}, "\n")
}

// winShimTargets 返回 Windows 下该 shim 的真实目标（按优先级）。
func winShimTargets(kind, name string, state *models.State) []string {
	runtime := state.CurrentRuntime(models.Tool(kind))
	if runtime == nil {
		return nil
	}
	home := WinHome(runtime.Home)
	switch kind {
	case "node":
		return []string{home + "\\" + WinNodeTargets[name]}
	case "maven", "gradle":
		return []string{home + "\\bin\\" + WinBuildtoolTargets[name]}
	default: // java
		return []string{home + "\\bin\\" + name + ".exe", home + "\\jre\\bin\\" + name + ".exe"}
	}
}

// ShimScriptWindows 生成 .cmd shim（cmd/PowerShell 入口）。
func ShimScriptWindows(kind, name string, state *models.State) string {
	targets := winShimTargets(kind, name, state)
	lines := []string{"@echo off", "rem DevSwitch shim — " + name}
	if len(targets) == 0 {
		lines = append(lines,
			"rem (no runtime selected)",
			"echo DevSwitch: 未配置所选版本，或当前版本没有 "+name+"。运行 devswitch list / use 1>&2",
			"exit /b 127",
			"")
		return strings.Join(lines, "\r\n")
	}
	labels := make([]string, len(targets))
	for i, target := range targets {
		labels[i] = "run"
		if i > 0 {
			labels[i] = fmt.Sprintf("run%d", i)
		}
		lines = append(lines, `if exist "`+target+`" goto `+labels[i])
	}
	lines = append(lines,
		"echo "+missingMessage(kind, name)+" 1>&2",
		"exit /b 127")
	for i, target := range targets {
		lines = append(lines, ":"+labels[i], `"`+target+`" %*`)
	}
	lines = append(lines, "")
	return strings.Join(lines, "\r\n")
}

// ShimScriptWindowsSh 生成无扩展名 sh twin（Git Bash 在 PATH 解析中
// 会先命中它而不是 .cmd）。
func ShimScriptWindowsSh(kind, name string, state *models.State) string {
	var lines []string
	for _, target := range winShimTargets(kind, name, state) {
		msys := ToMsysPath(target)
		// MSYS bash 把 .cmd/.bat 视为不可执行；用 -f 兜底，exec 会转交 cmd.exe。
		test := `[ -x ` + quote(msys) + ` ]`
		if strings.HasSuffix(strings.ToLower(msys), ".cmd") || strings.HasSuffix(strings.ToLower(msys), ".bat") {
			test = `[ -f ` + quote(msys) + ` ]`
		}
		lines = append(lines, "if "+test+"; then", `  exec `+quote(msys)+` "$@"`, "fi")
	}
	lines = append(lines,
		`echo "`+missingMessage(kind, name)+`" >&2`,
		"exit 127",
		"")
	header := []string{"#!/bin/sh", "# DevSwitch shim — " + name}
	return strings.Join(append(header, lines...), "\n")
}

// WriteShims 按当前状态生成全部 shim。只接管已选中的工具链，
// 未选中的不留桩（保持 command not found 语义）。
func WriteShims(state *models.State) ([]string, error) {
	written := []string{}
	wanted := map[string]bool{}
	for _, tool := range models.Tools {
		if state.CurrentRuntime(tool) == nil {
			continue
		}
		for _, name := range models.ToolShims[tool] {
			pathsWritten, err := writeOneShim(string(tool), name, state)
			if err != nil {
				return written, err
			}
			written = append(written, pathsWritten...)
			wanted[name] = true
		}
	}
	keep := map[string]bool{
		"devswitch": true, "devswitch-gui": true,
		"devswitch.cmd": true, "devswitch-gui.cmd": true,
	}
	for name := range wanted {
		keep[name] = true
		if paths.IsWindows {
			keep[name+".cmd"] = true
		}
	}
	entries, err := os.ReadDir(paths.LocalBin())
	if err == nil {
		for _, entry := range entries {
			if keep[entry.Name()] {
				continue
			}
			full := filepath.Join(paths.LocalBin(), entry.Name())
			if IsOurShim(full) {
				_ = os.Remove(full)
			}
		}
	}
	return written, nil
}

func writeOneShim(kind, name string, state *models.State) ([]string, error) {
	binDir := paths.LocalBin()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return nil, err
	}
	written := []string{}
	if paths.IsWindows {
		cmdDest := filepath.Join(binDir, name+".cmd")
		if err := writeFile(cmdDest, ShimScriptWindows(kind, name, state)); err != nil {
			return nil, err
		}
		shDest := filepath.Join(binDir, name)
		if err := writeExecutable(shDest, ShimScriptWindowsSh(kind, name, state)); err != nil {
			return nil, err
		}
		return append(written, cmdDest, shDest), nil
	}
	dest := filepath.Join(binDir, name)
	full := name
	if kind == "node-tool" {
		full = "node-tool"
	}
	_ = full
	if err := writeExecutable(dest, ShimScript(kind, name)); err != nil {
		return nil, err
	}
	return append(written, dest), nil
}

// WriteLaunchers 在 shim 目录生成 devswitch / devswitch-gui 启动器（仅 Windows）。
// 程序本体装在应用目录（如 %LOCALAPPDATA%\devswitch\app），不在用户 PATH 上，
// 启动器使命令行随处可用；内容含 ShimMarker，随 cleanup 一并清理，
// 且会覆盖 1.x（Python 版）升级后残留的失效启动器。
func WriteLaunchers() error {
	if !paths.IsWindows {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exe)
	binDir := paths.LocalBin()
	if strings.EqualFold(filepath.Clean(exeDir), filepath.Clean(binDir)) {
		return nil // 本体已在 PATH 目录（便携解压等场景），无需自引用启动器
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	targets := [][2]string{
		{"devswitch", ""},        // CLI
		{"devswitch-gui", "gui"}, // GUI
	}
	for _, t := range targets {
		name, extra := t[0], t[1]
		args := "%*"
		shArgs := `"$@"`
		if extra != "" {
			args = extra + " %*"
			shArgs = extra + ` "$@"`
		}
		cmd := strings.Join([]string{
			"@echo off",
			"rem DevSwitch shim — launcher",
			`"` + exe + `" ` + args,
		}, "\r\n") + "\r\n"
		if err := writeFile(filepath.Join(binDir, name+".cmd"), cmd); err != nil {
			return err
		}
		sh := strings.Join([]string{
			"#!/bin/sh",
			"# DevSwitch shim — launcher",
			`exec ` + quote(ToMsysPath(exe)) + ` ` + shArgs,
			"",
		}, "\n")
		if err := writeExecutable(filepath.Join(binDir, name), sh); err != nil {
			return err
		}
	}
	return nil
}
