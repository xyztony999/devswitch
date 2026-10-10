package detect

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
)

// Android SDK 检测：SDK 根目录为 home，platform-tools 的 adb 版本为主版本。
// 全程无子进程：版本读 platform-tools/source.properties 的 Pkg.Revision。

var pkgRevisionRe = regexp.MustCompile(`(?m)^Pkg\.Revision=(.+)$`)

// sdkVersionFromSourceProps 解析 platform-tools/source.properties 的 Pkg.Revision，
// 缺失时回退目录名（无法判定则 unknown）。
func sdkVersionFromSourceProps(platformToolsDir string) string {
	if data, err := os.ReadFile(filepath.Join(platformToolsDir, "source.properties")); err == nil {
		if m := pkgRevisionRe.FindStringSubmatch(string(data)); m != nil {
			v := strings.TrimSpace(m[1])
			if v != "" {
				return v
			}
		}
	}
	// platform-tools 目录本身不带版本号；AS 布局下无法从目录名取版本
	if entries, err := os.ReadDir(platformToolsDir); err == nil && len(entries) > 0 {
		return "unknown"
	}
	return "unknown"
}

func adbName() string {
	if isWindows {
		return "adb.exe"
	}
	return "adb"
}

// inspectAndroidSDK 校验 home 是合法 SDK 根（platform-tools/adb 存在）并构造 Runtime。
func inspectAndroidSDK(home string) *models.Runtime {
	ptDir := filepath.Join(home, "platform-tools")
	if _, err := os.Stat(filepath.Join(ptDir, adbName())); err != nil {
		return nil
	}
	version := sdkVersionFromSourceProps(ptDir)
	major := version
	if i := strings.Index(version, "."); i > 0 {
		major = version[:i]
	}
	binary := filepath.Join(ptDir, adbName())
	return &models.Runtime{
		Tool:    models.AndroidSDK,
		Version: version,
		Major:   major,
		Home:    home,
		Binary:  binary,
		Source:  classifyAndroidSDKSource(home),
		Label:   "Android SDK (platform-tools " + version + ")",
	}
}

// classifyAndroidSDKSource 按发现渠道标注来源（见 PRD ENUM-SOURCE）。
func classifyAndroidSDKSource(home string) string {
	norm := filepath.Clean(home)
	if strings.EqualFold(filepath.Clean(androidStudioSdkDir()), norm) {
		return "system"
	}
	for _, env := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if v := os.Getenv(env); v != "" {
			if strings.EqualFold(filepath.Clean(v), norm) {
				return "env"
			}
		}
	}
	return "path"
}

// androidStudioSdkDir 是 Android Studio 的默认 SDK 目录。
func androidStudioSdkDir() string {
	if isWindows {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Android", "Sdk")
		}
		return ""
	}
	home := paths.Home()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "Android", "Sdk")
}

// candidateAndroidSDKHomes 汇总四类检测根（AS 默认目录 / ANDROID_HOME /
// ANDROID_SDK_ROOT / PATH 中 adb 回溯两级），按探测顺序去重。
func candidateAndroidSDKHomes() []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(home string) {
		if home == "" {
			return
		}
		clean := filepath.Clean(home)
		key := strings.ToLower(clean)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, clean)
	}
	add(androidStudioSdkDir())
	for _, env := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		add(os.Getenv(env))
	}
	for _, bin := range whichAll("adb") {
		if isShim(bin) {
			continue
		}
		resolved := resolvePath(bin)
		// <sdk>/platform-tools/adb(.exe) → SDK 根
		add(filepath.Dir(filepath.Dir(resolved)))
	}
	return out
}
