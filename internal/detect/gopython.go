package detect

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
)

// Go 工具链与 Python 检测（零子进程）。
// 下载安装的实例带 .devswitch-version 元数据（InstallRuntime 写入），
// 检测优先读它获得完整版本；系统检测只能推到已知精度（Go 的 VERSION 文件
// 本身完整；Python 目录/注册表只有 major.minor）。

var goVersionFileRe = regexp.MustCompile(`^go(\d+\.\d+(?:\.\d+)?)`)

// MetaVersion 读安装目录下的 .devswitch-version（下载实例元数据）。
func MetaVersion(home string) string {
	if data, err := os.ReadFile(filepath.Join(home, ".devswitch-version")); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			return v
		}
	}
	return ""
}

func exeName(base string) string {
	if isWindows {
		return base + ".exe"
	}
	return base
}

// ---------------------------------------------------------------------------
// Go
// ---------------------------------------------------------------------------

func inspectGolang(home string) *models.Runtime {
	binary := filepath.Join(home, "bin", exeName("go"))
	if !fileExists(binary) {
		return nil
	}
	version := ""
	if data, err := os.ReadFile(filepath.Join(home, "VERSION")); err == nil {
		if m := goVersionFileRe.FindStringSubmatch(strings.TrimSpace(string(data))); m != nil {
			version = m[1]
		}
	}
	if version == "" {
		version = MetaVersion(home)
	}
	if version == "" {
		return nil
	}
	major := version
	if i := strings.Index(version, "."); i > 0 {
		major = version[:i]
	}
	return &models.Runtime{
		Tool: models.Golang, Version: version, Major: major,
		Home: home, Binary: binary, Source: golangSourceOf(home),
		Label: "Go " + version,
	}
}

func golangSourceOf(home string) string {
	clean := filepath.Clean(home)
	if v := os.Getenv("GOROOT"); v != "" && strings.EqualFold(filepath.Clean(v), clean) {
		return "env"
	}
	return "path"
}

func candidateGolangHomes() []string {
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
	add(os.Getenv("GOROOT"))
	if isWindows {
		if pf := os.Getenv("ProgramFiles"); pf != "" {
			add(filepath.Join(pf, "Go"))
		}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			add(filepath.Join(local, "Programs", "Go"))
		}
	} else {
		add("/usr/local/go")
		add("/usr/lib/go")
	}
	if home := paths.Home(); home != "" {
		// 本工具下载的 ~/go-<ver>
		if entries, err := os.ReadDir(home); err == nil {
			for _, e := range entries {
				if e.IsDir() && strings.HasPrefix(e.Name(), "go-") {
					add(filepath.Join(home, e.Name()))
				}
			}
		}
	}
	for _, bin := range whichAll("go") {
		if isShim(bin) {
			continue
		}
		// <root>/bin/go → root
		add(filepath.Dir(filepath.Dir(resolvePath(bin))))
	}
	return out
}

// ---------------------------------------------------------------------------
// Python
// ---------------------------------------------------------------------------

var pyDirVersionRe = regexp.MustCompile(`[Pp]ython(\d+)\.(\d+)`)

func inspectPython(home string) *models.Runtime {
	// 合法布局：Windows 根 python.exe；Linux bin/python3
	var binary string
	if isWindows {
		binary = filepath.Join(home, "python.exe")
	} else {
		binary = filepath.Join(home, "bin", "python3")
	}
	if !fileExists(binary) {
		return nil
	}
	version := MetaVersion(home)
	if version == "" {
		// 目录名（Python312）或 lib/python3.12 推 major.minor
		if m := pyDirVersionRe.FindStringSubmatch(home); m != nil {
			version = m[1] + "." + m[2]
		} else if entries, err := os.ReadDir(filepath.Join(home, "lib")); err == nil {
			for _, e := range entries {
				if m := pyDirVersionRe.FindStringSubmatch(e.Name()); m != nil {
					version = m[1] + "." + m[2]
					break
				}
			}
		}
	}
	if version == "" {
		return nil
	}
	major := version
	if i := strings.Index(version, "."); i > 0 {
		major = version[:i]
	}
	return &models.Runtime{
		Tool: models.Python, Version: version, Major: major,
		Home: home, Binary: binary, Source: pythonSourceOf(home),
		Label: "Python " + version,
	}
}

func pythonSourceOf(home string) string {
	clean := filepath.Clean(home)
	if isWindows {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			base := filepath.Join(local, "Programs", "Python")
			if strings.EqualFold(filepath.Dir(clean), base) {
				return "system"
			}
		}
	}
	if h := paths.Home(); h != "" {
		if strings.HasPrefix(strings.ToLower(clean), strings.ToLower(filepath.Join(h, "python-"))) {
			return "downloaded"
		}
	}
	return "path"
}

func candidatePythonHomes() []string {
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
	if isWindows {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			base := filepath.Join(local, "Programs", "Python")
			if entries, err := os.ReadDir(base); err == nil {
				for _, e := range entries {
					if e.IsDir() && strings.HasPrefix(e.Name(), "Python3") {
						add(filepath.Join(base, e.Name()))
					}
				}
			}
		}
	}
	if home := paths.Home(); home != "" {
		if entries, err := os.ReadDir(home); err == nil {
			for _, e := range entries {
				if e.IsDir() && strings.HasPrefix(e.Name(), "python-") {
					add(filepath.Join(home, e.Name()))
				}
			}
		}
	}
	// PATH 回溯：Windows python.exe 在根（回溯一级）；Linux bin/python3（回溯两级）
	for _, bin := range whichAll("python") {
		if isShim(bin) {
			continue
		}
		resolved := resolvePath(bin)
		if isWindows {
			add(filepath.Dir(resolved))
		} else {
			add(filepath.Dir(filepath.Dir(resolved)))
		}
	}
	return out
}
