package detect

import (
	"os"
	"path/filepath"
	"testing"
)

// 沙箱构造合法 SDK 根（platform-tools + adb + source.properties），
// 验证检测、版本解析与来源分类；非法目录应返回 nil。
func TestInspectAndroidSDK(t *testing.T) {
	sdk := t.TempDir()
	pt := filepath.Join(sdk, "platform-tools")
	if err := os.MkdirAll(pt, 0o755); err != nil {
		t.Fatal(err)
	}
	adb := "adb"
	if isWindows {
		adb = "adb.exe"
	}
	if err := os.WriteFile(filepath.Join(pt, adb), []byte{0}, 0o755); err != nil {
		t.Fatal(err)
	}
	props := "Pkg.Revision=35.0.2\nPkg.Desc=Android SDK Platform-Tools\n"
	if err := os.WriteFile(filepath.Join(pt, "source.properties"), []byte(props), 0o644); err != nil {
		t.Fatal(err)
	}

	r := inspectAndroidSDK(sdk)
	if r == nil {
		t.Fatal("合法 SDK 根未被识别")
	}
	if r.Version != "35.0.2" || r.Major != "35" {
		t.Errorf("版本解析错误：%q / %q", r.Version, r.Major)
	}
	if r.Tool != "androidsdk" {
		t.Errorf("tool 应为 androidsdk：%s", r.Tool)
	}
	if want := filepath.Join(pt, adb); r.Binary != want {
		t.Errorf("binary 路径错误：%s（want %s）", r.Binary, want)
	}

	// 无 platform-tools 的目录拒绝
	empty := t.TempDir()
	if r := inspectAndroidSDK(empty); r != nil {
		t.Error("无 platform-tools 的目录不应被识别")
	}
}

func TestSDKVersionFallback(t *testing.T) {
	pt := t.TempDir()
	if err := os.WriteFile(filepath.Join(pt, "adb"), []byte{0}, 0o755); err != nil {
		t.Fatal(err)
	}
	if v := sdkVersionFromSourceProps(pt); v != "unknown" {
		t.Errorf("无 source.properties 应回退 unknown，得到 %q", v)
	}
}
