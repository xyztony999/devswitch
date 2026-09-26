//go:build windows

// Package winenv 封装 HKCU\Environment 读写与 WM_SETTINGCHANGE 广播，
// 平移自 devswitch/winenv.py。
package winenv

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	envValue       = `Environment`
	hwndBroadcast  = uintptr(0xFFFF)
	wmSettingChg   = 0x001A
	smtoAbortIfHub = 0x0002
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procSendMessageTime = user32.NewProc("SendMessageTimeoutW")
)

// GetUserValue 返回 (值, 是否存在)。
func GetUserValue(name string) (string, bool) {
	key, err := registry.OpenKey(registry.CURRENT_USER, envValue, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer key.Close()
	value, _, err := key.GetStringValue(name)
	if err != nil {
		return "", false
	}
	return value, true
}

// SetValue 写用户环境变量；含 % 的值用 REG_EXPAND_SZ 保持可展开语义。
func SetValue(name, value string) bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, envValue, registry.SET_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	hasPercent := false
	for i := 0; i < len(value); i++ {
		if value[i] == '%' {
			hasPercent = true
			break
		}
	}
	var setErr error
	if hasPercent {
		setErr = key.SetExpandStringValue(name, value)
	} else {
		setErr = key.SetStringValue(name, value)
	}
	if setErr != nil {
		return false
	}
	BroadcastChange()
	return true
}

// DeleteValue 删除用户环境变量（不存在也返回 true）。
func DeleteValue(name string) bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, envValue, registry.SET_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_ = key.DeleteValue(name)
	BroadcastChange()
	return true
}

// BroadcastChange 通知新进程环境变量已变。
func BroadcastChange() {
	env, _ := syscall.UTF16PtrFromString("Environment")
	procSendMessageTime.Call(
		hwndBroadcast, wmSettingChg, 0,
		uintptr(unsafe.Pointer(env)), smtoAbortIfHub, 5000, 0,
	)
}

// UserPathEntries 返回用户 PATH 的分项。
func UserPathEntries() []string {
	raw, _ := GetUserValue("Path")
	return SplitPath(raw)
}

// SplitPath 按分号切分并去空。
func SplitPath(raw string) []string {
	entries := []string{}
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == ';' {
			item := trimSpaces(raw[start:i])
			if item != "" {
				entries = append(entries, item)
			}
			start = i + 1
		}
	}
	return entries
}

func trimSpaces(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func sameEntry(entry, target string) bool {
	if equalFold(entry, target) {
		return true
	}
	return equalFold(expandEntry(entry), target)
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func expandEntry(entry string) string {
	if len(entry) < 2 || entry[0] != '%' {
		return entry
	}
	for i := 1; i < len(entry); i++ {
		if entry[i] == '%' && i > 1 {
			name := entry[1:i]
			value, ok := GetUserValue(name)
			if ok {
				return value + entry[i+1:]
			}
			// 系统 PATH 里的变量在用户表查不到时再试系统表
			if key, err := registry.OpenKey(registry.LOCAL_MACHINE,
				`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
				registry.QUERY_VALUE); err == nil {
				defer key.Close()
				if v, _, err := key.GetStringValue(name); err == nil {
					return v + entry[i+1:]
				}
			}
			return entry
		}
	}
	return entry
}

// EnsurePathEntry 把 target 放进用户 PATH（first=true 时置顶），返回是否修改。
func EnsurePathEntry(target string, first bool) bool {
	entries := UserPathEntries()
	if len(entries) > 0 && sameEntry(entries[0], target) {
		return false // 已在首位，无需修改
	}
	exists := false
	kept := make([]string, 0, len(entries))
	for _, item := range entries {
		if sameEntry(item, target) {
			exists = true
			continue
		}
		kept = append(kept, item)
	}
	if exists && !first {
		return false // 已存在且无需置顶
	}
	next := append([]string{target}, kept...)
	key, err := registry.OpenKey(registry.CURRENT_USER, envValue, registry.SET_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	if err := key.SetStringValue("Path", joinSemi(next)); err != nil {
		return false
	}
	BroadcastChange()
	return true
}

// RemovePathEntry 从用户 PATH 移除 target。
func RemovePathEntry(target string) bool {
	entries := UserPathEntries()
	kept := make([]string, 0, len(entries))
	changed := false
	for _, item := range entries {
		if sameEntry(item, target) {
			changed = true
			continue
		}
		kept = append(kept, item)
	}
	if !changed {
		return false
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, envValue, registry.SET_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	if len(kept) > 0 {
		_ = key.SetStringValue("Path", joinSemi(kept))
	} else {
		_ = key.DeleteValue("Path")
	}
	BroadcastChange()
	return true
}

func joinSemi(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ";"
		}
		out += item
	}
	return out
}
