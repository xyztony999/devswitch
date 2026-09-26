// Package paths 集中全部目录决策，平移自 devswitch/paths.py。
// 目录决策导出为函数变量，测试可整体替换到沙箱目录。
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	AppID      = "devswitch"
	HookBegin  = "# >>> devswitch >>>"
	HookEnd    = "# <<< devswitch <<<"
	ShimMarker = "DevSwitch shim"
)

var IsWindows = runtime.GOOS == "windows"

var Home = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

// ConfigDir 对应 Python 版 config_dir：Windows %APPDATA%，XDG ~/.config。
var ConfigDir = func() string {
	var base string
	if IsWindows {
		base = os.Getenv("APPDATA")
		if base == "" {
			base = filepath.Join(Home(), "AppData", "Roaming")
		}
	} else {
		base = os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(Home(), ".config")
		}
	}
	return filepath.Join(base, AppID)
}

var DataDir = func() string {
	var base string
	if IsWindows {
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(Home(), "AppData", "Local")
		}
	} else {
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(Home(), ".local", "share")
		}
	}
	return filepath.Join(base, AppID)
}

func StateFile() string  { return filepath.Join(ConfigDir(), "state.json") }
func CurrentEnv() string { return filepath.Join(ConfigDir(), "current.env") }
func EnvFile() string    { return filepath.Join(ConfigDir(), "env.sh") }
func BackupDir() string  { return filepath.Join(ConfigDir(), "backup") }

var LocalBin = func() string {
	if IsWindows {
		return filepath.Join(DataDir(), "bin")
	}
	return filepath.Join(Home(), ".local", "bin")
}

var Bashrc = func() string { return filepath.Join(Home(), ".bashrc") }
var Profile = func() string { return filepath.Join(Home(), ".profile") }

// DocumentsDir 定位用户文档目录（OneDrive 重定向优先）。
var DocumentsDir = func() string {
	for _, candidate := range []string{
		os.Getenv("OneDrive"),
		filepath.Join(Home(), "Documents"),
	} {
		if candidate != "" {
			dir := filepath.Join(candidate, "Documents")
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return dir
			}
		}
	}
	return filepath.Join(Home(), "Documents")
}

// PowerShellProfiles 返回 Windows PowerShell 5.1（必写）与 PowerShell 7
// （已存在才写）的 all-hosts profile 路径。
var PowerShellProfiles = func() []string {
	docs := DocumentsDir()
	profiles := []string{filepath.Join(docs, "WindowsPowerShell", "profile.ps1")}
	ps7 := filepath.Join(docs, "PowerShell", "profile.ps1")
	if dirOK(filepath.Dir(ps7)) || fileOK(ps7) {
		profiles = append(profiles, ps7)
	}
	return profiles
}

// HookTargets 是当前平台需要写入 env 钩子的全部文件。
var HookTargets = func() []string {
	if IsWindows {
		return append(PowerShellProfiles(), Bashrc(), Profile())
	}
	return []string{Bashrc(), Profile()}
}

// ShimFilename 返回 shim 文件名（Windows 另有无扩展 sh twin）。
func ShimFilename(name string) string {
	if IsWindows {
		return name + ".cmd"
	}
	return name
}

func dirOK(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileOK(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
