package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
)

// golden 目录来自 Python 版的基线快照；本测试证明 Go 平移与
// Python 生成物逐字节一致（在沙箱路径归一化后比较）。
const goldenDir = "../../tests/golden/linux"

// fakePaths 把目录决策重定向到 t.TempDir，并强制走 Linux 分支
// （host 无关，Windows 上也能跑）。
func fakePaths(t *testing.T) (home string, restore func()) {
	t.Helper()
	origIsWindows := paths.IsWindows
	origHome, origConfig, origData, origLocalBin := paths.Home, paths.ConfigDir, paths.DataDir, paths.LocalBin

	tmp := t.TempDir()
	root := filepath.Join(tmp, "home")
	_ = os.MkdirAll(root, 0o755)

	paths.IsWindows = false
	paths.Home = func() string { return root }
	paths.ConfigDir = func() string { return filepath.Join(root, ".config", "devswitch") }
	paths.DataDir = func() string { return filepath.Join(root, ".local", "share", "devswitch") }
	paths.LocalBin = func() string { return filepath.Join(root, ".local", "bin") }

	return root, func() {
		paths.IsWindows = origIsWindows
		paths.Home, paths.ConfigDir, paths.DataDir, paths.LocalBin = origHome, origConfig, origData, origLocalBin
	}
}

func testState() *models.State {
	state := models.NewState()
	state.Current[models.Node] = "/opt/node-v22.11.0"
	state.Current[models.Java] = "/usr/lib/jvm/jdk-17.0.9"
	state.Runtimes = []models.Runtime{
		{Tool: models.Node, Version: "22.11.0", Major: "22",
			Home: "/opt/node-v22.11.0", Binary: "/opt/node-v22.11.0/bin/node", Source: "local"},
		{Tool: models.Java, Version: "17.0.9", Major: "17",
			Home: "/usr/lib/jvm/jdk-17.0.9", Binary: "/usr/lib/jvm/jdk-17.0.9/bin/java", Source: "system"},
	}
	return &state
}

// normalize 把沙箱临时目录映射回 golden 的规范 home（/home/tester）。
// 临时目录在 Windows host 上可能是反斜杠路径，统一转 / 后再替换。
func normalize(text, sandboxRoot string) string {
	slashRoot := filepath.ToSlash(sandboxRoot)
	text = strings.ReplaceAll(text, slashRoot, "/home/tester")
	return strings.ReplaceAll(text, "\\", "/")
}

func golden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(goldenDir, name))
	if err != nil {
		t.Fatalf("读取 golden 失败: %v", err)
	}
	return string(data)
}

func TestGoldenShimNode(t *testing.T) {
	root, restore := fakePaths(t)
	defer restore()
	got := normalize(ShimScript("node", "node"), root)
	if want := golden(t, "shim-node"); got != want {
		t.Errorf("shim-node 不一致:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGoldenShimNodeTool(t *testing.T) {
	root, restore := fakePaths(t)
	defer restore()
	got := normalize(ShimScript("node-tool", "pnpm"), root)
	if want := golden(t, "shim-node-tool"); got != want {
		t.Errorf("shim-node-tool 不一致:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGoldenShimJava(t *testing.T) {
	root, restore := fakePaths(t)
	defer restore()
	got := normalize(ShimScript("java", "java"), root)
	if want := golden(t, "shim-java"); got != want {
		t.Errorf("shim-java 不一致:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestGoldenHookSnippet(t *testing.T) {
	root, restore := fakePaths(t)
	defer restore()
	got := normalize(HookSnippet(), root)
	if want := golden(t, "hook-snippet"); got != want {
		t.Errorf("hook-snippet 不一致:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// stripAndroidSDKLines 删除 androidsdk 接入带来的增量行（赋值行 + 专属段），
// 余下内容与 Python 时代的 golden 逐字节一致——即"四工具语义不回归"的交叉断言。
func stripAndroidSDKLines(text string) string {
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	skipBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, "DEVSWITCH_ANDROIDSDK_HOME") && strings.HasSuffix(trimmed, "]; then") {
			skipBlock = true
			continue
		}
		if skipBlock {
			if trimmed == "fi" {
				skipBlock = false
			}
			continue
		}
		if strings.Contains(line, "DEVSWITCH_ANDROIDSDK_HOME") {
			continue // 赋值行
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func TestGoldenCurrentEnvAndEnvSh(t *testing.T) {
	root, restore := fakePaths(t)
	defer restore()
	state := testState()
	if err := WriteCurrentEnv(state); err != nil {
		t.Fatal(err)
	}
	if err := WriteEnvSh(state); err != nil {
		t.Fatal(err)
	}
	currentData, _ := os.ReadFile(paths.CurrentEnv())
	envData, _ := os.ReadFile(paths.EnvFile())

	// 交叉断言：剔除 androidsdk 增量后与旧 golden 字节一致（四工具不回归）
	if got, want := stripAndroidSDKLines(normalize(string(currentData), root)), golden(t, "current.env"); got != want {
		t.Errorf("current.env（剔除 androidsdk 行）不一致:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if got, want := stripAndroidSDKLines(normalize(string(envData), root)), golden(t, "env.sh"); got != want {
		t.Errorf("env.sh（剔除 androidsdk 段）不一致:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}

	// androidsdk 增量行本身：未选中时为空赋值
	if !strings.Contains(string(envData), "DEVSWITCH_ANDROIDSDK_HOME=\"\"") {
		t.Error("env.sh 应含 androidsdk 空赋值行")
	}
}

func TestEnvShWithAndroidSDKSelected(t *testing.T) {
	root, restore := fakePaths(t)
	defer restore()
	state := testState()
	state.Runtimes = append(state.Runtimes, models.Runtime{
		Tool: models.AndroidSDK, Version: "35.0.2", Major: "35",
		Home: "/opt/android-sdk", Binary: "/opt/android-sdk/platform-tools/adb", Source: "system",
	})
	state.Current[models.AndroidSDK] = "/opt/android-sdk"
	if err := WriteEnvSh(state); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(paths.EnvFile())
	got := normalize(string(data), root)
	for _, want := range []string{
		`DEVSWITCH_ANDROIDSDK_HOME="/opt/android-sdk"`,
		`if [ -n "$DEVSWITCH_ANDROIDSDK_HOME" ]; then`,
		`  export ANDROID_HOME="$DEVSWITCH_ANDROIDSDK_HOME"`,
		`  _devswitch_front "$ANDROID_HOME/platform-tools"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("env.sh 缺少 androidsdk 段：%q\n%s", want, got)
		}
	}
}

func TestWindowsShimsWriteBothTwins(t *testing.T) {
	tmp := t.TempDir()
	origIsWindows, origLocalBin := paths.IsWindows, paths.LocalBin
	paths.IsWindows = true
	paths.LocalBin = func() string { return filepath.Join(tmp, "bin") }
	defer func() { paths.IsWindows, paths.LocalBin = origIsWindows, origLocalBin }()

	state := testState()
	if _, err := WriteShims(state); err != nil {
		t.Fatal(err)
	}
	cmdData, err := os.ReadFile(filepath.Join(tmp, "bin", "node.cmd"))
	if err != nil {
		t.Fatal("缺少 node.cmd:", err)
	}
	if !strings.Contains(string(cmdData), "\r\n") || strings.Contains(string(cmdData), "\r\r") {
		t.Error("node.cmd 换行应为 CRLF 且无重复 CR")
	}
	shData, err := os.ReadFile(filepath.Join(tmp, "bin", "node"))
	if err != nil {
		t.Fatal("缺少 sh twin:", err)
	}
	if strings.Contains(string(shData), "\r") {
		t.Error("sh twin 换行应为 LF")
	}
}

func TestWindowsShimSkipsUnselectedTools(t *testing.T) {
	tmp := t.TempDir()
	origIsWindows, origLocalBin := paths.IsWindows, paths.LocalBin
	paths.IsWindows = true
	paths.LocalBin = func() string { return filepath.Join(tmp, "bin") }
	defer func() { paths.IsWindows, paths.LocalBin = origIsWindows, origLocalBin }()

	state := models.NewState()
	state.Current[models.Node] = "/opt/node"
	state.Runtimes = []models.Runtime{{Tool: models.Node, Version: "22.0.0", Major: "22",
		Home: "/opt/node", Binary: "/opt/node/bin/node", Source: "local"}}
	if _, err := WriteShims(&state); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "bin", "java.cmd")); !os.IsNotExist(err) {
		t.Error("未选中的 java 不应生成 shim")
	}
}
