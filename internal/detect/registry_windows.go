//go:build windows

package detect

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// nodeVersionFromRegistry 兜底：MSI 安装的 nodejs 目录名无版本，
// 从卸载表读 DisplayVersion，并用 InstallLocation 匹配候选目录。
func nodeVersionFromRegistry(home string) string {
	roots := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Node.js`},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Node.js`},
	}
	for _, r := range roots {
		key, err := registry.OpenKey(r.root, r.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		version, _, err := key.GetStringValue("DisplayVersion")
		location, _, _ := key.GetStringValue("InstallLocation")
		key.Close()
		if err != nil || version == "" {
			continue
		}
		if location == "" || sameDir(location, home) {
			return strings.TrimPrefix(version, "v")
		}
	}
	return ""
}

func sameDir(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
