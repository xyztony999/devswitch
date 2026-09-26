// Package cli 是命令行入口，平移自 devswitch/cli.py。
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/xyztony999/devswitch/internal/downloader"
	"github.com/xyztony999/devswitch/internal/gui"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/service"
)

var usage = `用法：devswitch <命令> [参数]

命令：
  scan                    扫描本机已有版本
  list [tool] [--json]    查看列表，* 为当前
  current                 显示当前选中的版本
  use <tool> <version>    切换版本（node/java/maven/gradle）
  install <tool> <ver>    下载安装并切换（--mirror npmmirror|tuna）
  import <tool> <path>    导入一个本地安装目录
  export [file]           导出版本选择到 .devswitch
  apply <file>            按文件对齐版本（--install-missing）
  which <name>            打印当前实际二进制路径
  doctor                  检查 PATH / 冲突配置
  cleanup                 清理用户级配置（卸载配套，运行时数据保留）
  update                  检查新版本（只读）
  version                 显示版本`

func Main(args []string) int {
	if len(args) == 0 {
		fmt.Println(usage)
		return 2
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "--version", "version":
		fmt.Println("DevSwitch", models.AppVersion)
		return 0
	case "scan":
		return cmdScan()
	case "list":
		return cmdList(rest)
	case "current":
		return cmdCurrent()
	case "use":
		return cmdUse(rest)
	case "install":
		return cmdInstall(rest)
	case "import":
		return cmdImport(rest)
	case "export":
		return cmdExport(rest)
	case "apply":
		return cmdApply(rest)
	case "which":
		return cmdWhich(rest)
	case "doctor":
		return cmdDoctor(rest)
	case "cleanup":
		return cmdCleanup()
	case "update":
		return cmdUpdate()
	case "gui":
		return gui.Run()
	default:
		fmt.Fprintf(os.Stderr, "未知命令：%s\n%s\n", cmd, usage)
		return 2
	}
}

func parseTool(arg string) (models.Tool, bool) {
	switch arg {
	case "node", "java", "maven", "gradle":
		return models.Tool(arg), true
	}
	return "", false
}

func cmdScan() int {
	state := service.MergeScan()
	service.EnsureApplied()
	fmt.Printf("已扫描到 %d 个运行时。\n", len(state.Runtimes))
	return cmdList(nil)
}

func cmdList(args []string) int {
	toolFilter := ""
	jsonOut := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
		} else if _, ok := parseTool(arg); ok {
			toolFilter = arg
		}
	}
	state := service.MergeScan()
	if jsonOut {
		fmt.Println("[")
		for i, r := range state.Runtimes {
			comma := ","
			if i == len(state.Runtimes)-1 {
				comma = ""
			}
			fmt.Printf("  {\"tool\": %q, \"version\": %q, \"home\": %q, \"source\": %q}%s\n",
				r.Tool, r.Version, r.Home, r.Source, comma)
		}
		fmt.Println("]")
		return 0
	}
	count := 0
	for _, tool := range models.Tools {
		if toolFilter != "" && string(tool) != toolFilter {
			continue
		}
		for _, r := range state.ForTool(tool) {
			mark := " "
			if state.Current[tool] == r.Home {
				mark = "*"
			}
			fmt.Printf("%s %-6s %-14s %-26s %-8s %s\n", mark, tool, r.Version, r.Label, r.Source, r.Home)
			count++
		}
	}
	if count > 0 {
		fmt.Println("\n* 表示当前选中。切换：devswitch use node 22")
	} else {
		fmt.Println("还没有发现运行时。安装后执行：devswitch scan 或 devswitch install node 22")
	}
	return 0
}

func cmdCurrent() int {
	state := service.MergeScan()
	for _, tool := range models.Tools {
		if r := state.CurrentRuntime(tool); r != nil {
			fmt.Printf("%s  %s  (%s)\n", tool, r.Version, r.Home)
		} else {
			fmt.Printf("%s  (未选择)\n", tool)
		}
	}
	return 0
}

func cmdUse(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "用法：devswitch use <tool> <version>")
		return 1
	}
	tool, ok := parseTool(args[0])
	if !ok {
		fmt.Fprintf(os.Stderr, "不支持的工具：%s（node/java/maven/gradle）\n", args[0])
		return 1
	}
	runtime, err := service.Use(tool, args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("已切换 %s -> %s (%s)\n", tool, runtime.Version, runtime.Home)
	if models.ToolHomeVars[tool] != "" {
		fmt.Printf("%s 对新开终端生效；当前终端的命令已立即生效。\n", models.ToolHomeVars[tool])
	} else {
		fmt.Println("当前终端里的命令已立即生效。")
	}
	return 0
}

func cmdInstall(args []string) int {
	toolArg, version, mirror := "", "", ""
	positional := []string{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "--mirror=") {
			mirror = strings.TrimPrefix(arg, "--mirror=")
		} else if arg == "--mirror" {
			// 占位
		} else {
			positional = append(positional, arg)
		}
	}
	if len(positional) >= 1 {
		toolArg = positional[0]
	}
	if len(positional) >= 2 {
		version = positional[1]
	}
	// 兼容 --mirror <name>
	for i, arg := range args {
		if arg == "--mirror" && i+1 < len(args) {
			mirror = args[i+1]
		}
	}
	tool, ok := parseTool(toolArg)
	if !ok || version == "" {
		fmt.Fprintln(os.Stderr, "用法：devswitch install <node|java|maven|gradle> <大版本号> [--mirror official|npmmirror|tuna]")
		return 1
	}
	home, fullVersion, err := service.InstallRuntime(tool, version, mirror)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("已切换 %s -> %s (%s)\n", tool, fullVersion, home)
	return 0
}

func cmdImport(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "用法：devswitch import <tool> <目录>")
		return 1
	}
	tool, ok := parseTool(args[0])
	if !ok {
		fmt.Fprintf(os.Stderr, "不支持的工具：%s\n", args[0])
		return 1
	}
	runtime, err := service.ImportPath(tool, args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("已导入 %s %s (%s)\n", runtime.Tool, runtime.Version, runtime.Home)
	return 0
}

func cmdExport(args []string) int {
	path := ".devswitch"
	if len(args) > 0 {
		path = args[0]
	}
	written, err := service.ExportVersions(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("已导出当前版本选择到 %s\n", written)
	return 0
}

func cmdApply(args []string) int {
	file := ""
	installMissing, mirror := false, ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--install-missing":
			installMissing = true
		case args[i] == "--mirror" && i+1 < len(args):
			mirror = args[i+1]
			i++
		case file == "":
			file = args[i]
		}
	}
	if file == "" {
		fmt.Fprintln(os.Stderr, "用法：devswitch apply <file> [--install-missing] [--mirror <name>]")
		return 1
	}
	results, err := service.ApplyVersions(file, installMissing, mirror)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	failed := false
	for _, r := range results {
		fmt.Printf("  %s  %s  %s\n", r.Outcome, r.Tool, r.Version)
		if strings.HasPrefix(r.Outcome, "missing") || strings.HasPrefix(r.Outcome, "failed") || strings.HasPrefix(r.Outcome, "unknown") {
			failed = true
		}
	}
	if failed {
		fmt.Println("\n有未对齐的版本：可先 devswitch scan，或用 --install-missing 自动下载。")
		return 1
	}
	fmt.Println("团队版本已对齐。")
	return 0
}

func cmdWhich(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "用法：devswitch which <name>（node/npm/mvn/JAVA_HOME…）")
		return 1
	}
	if path, ok := service.WhichBinary(args[0]); ok {
		fmt.Println(path)
		return 0
	}
	fmt.Fprintf(os.Stderr, "未找到 %s\n", args[0])
	return 1
}

func cmdUpdate() int {
	latest, err := downloader.LatestReleaseVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "检查更新失败：%v\n", err)
		return 1
	}
	if latest == models.AppVersion {
		fmt.Printf("已是最新版本（%s）。\n", models.AppVersion)
	} else {
		fmt.Printf("发现新版本：%s（当前 %s）。\n下载：https://github.com/xyztony999/devswitch/releases/latest\n", latest, models.AppVersion)
	}
	return 0
}
