//go:build !windows

// Package winenv 在非 Windows 平台的存根（CLI 核心 Linux 分支不触达注册表）。
package winenv

func GetUserValue(string) (string, bool)      { return "", false }
func SetValue(string, string) bool            { return false }
func DeleteValue(string) bool                 { return false }
func UserPathEntries() []string               { return nil }
func EnsurePathEntry(string, bool) bool       { return false }
func RemovePathEntry(string) bool             { return false }
