package downloader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/xyztony999/devswitch/internal/paths"
)

// Go 工具链与 Python 下载安装。
// Go：golang.google.cn 官方中国站的 dl 清单 JSON（含 sha256）。
// Python：python-build-standalone（uv 同源，完整 CPython 含 pip），
// npmmirror 镜像两级目录 JSON（根目录列日期目录，日期目录列文件）。
// 两者都把解压出的顶层目录 rename 为 <home>/{go,python}-<version>，
// 并写入 .devswitch-version 元数据供检测器读完整版本。

// ---------------------------------------------------------------------------
// Go
// ---------------------------------------------------------------------------

type goDLRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		SHA256   string `json:"sha256"`
	} `json:"files"`
}

var goVersionTrimRe = regexp.MustCompile(`^go`)

func installGolang(major, mirror string) (string, error) {
	data, err := httpGet("https://golang.google.cn/dl/?mode=json&include=all", 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("读取 Go 版本清单失败：%w", err)
	}
	releases := []goDLRelease{}
	if err := json.Unmarshal(data, &releases); err != nil {
		return "", fmt.Errorf("解析 Go 版本清单失败：%w", err)
	}
	goos, arch := "linux", "amd64"
	if runtime.GOOS == "windows" {
		goos = "windows"
	}
	file, version, fileSHA := "", "", ""
	for _, r := range releases {
		if !r.Stable {
			continue
		}
		v := goVersionTrimRe.ReplaceAllString(r.Version, "")
		if major != "" && !strings.HasPrefix(v, major+".") && v != major {
			continue
		}
		for _, f := range r.Files {
			if f.OS == goos && f.Arch == arch && !strings.HasSuffix(f.Filename, ".msi") {
				file, version, fileSHA = f.Filename, v, f.SHA256
				break
			}
		}
		if file != "" {
			break
		}
	}
	if file == "" {
		return "", fmt.Errorf("没有发现 Go %s 系列的 %s/%s 安装包", major, goos, arch)
	}
	fmt.Println("Go", version)
	// 官方清单自带 sha256，显式校验（golang.google.cn 对 .sha256 sidecar 返回 HTML 页）
	archivePath, err := downloadWithChecksum("https://golang.google.cn/dl/"+file, fileSHA, "sha256")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(filepath.Dir(archivePath))
	// 发行包顶层目录是 go/，必须改名 go-<version>（~/go 是 GOPATH 默认值，冲突）
	staging, err := os.MkdirTemp(paths.Home(), ".ds-go-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)
	if _, err := safeExtractSingleRoot(archivePath, staging); err != nil {
		return "", err
	}
	dest := filepath.Join(paths.Home(), "go-"+version)
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := os.Rename(filepath.Join(staging, "go"), dest); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dest, ".devswitch-version"), []byte(version), 0o644); err != nil {
		return "", err
	}
	return dest, nil
}

// ---------------------------------------------------------------------------
// Python（python-build-standalone via npmmirror）
// ---------------------------------------------------------------------------

type pbsEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

var pbsVersionRe = regexp.MustCompile(`^cpython-(\d+\.\d+(?:\.\d+)?)\+(\d{8})-`)

func installPython(versionWanted, mirror string) (string, error) {
	base := "https://registry.npmmirror.com/-/binary/python-build-standalone"
	rootData, err := httpGet(base+"/", 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("读取 Python 发行版目录失败：%w", err)
	}
	rootEntries := []pbsEntry{}
	if err := json.Unmarshal(rootData, &rootEntries); err != nil {
		return "", fmt.Errorf("解析 Python 发行版目录失败：%w", err)
	}
	// 日期目录倒序取最新可用（个别日期目录可能 404，逐个回退）
	dates := []string{}
	for _, e := range rootEntries {
		if e.Type == "dir" && len(e.Name) == 9 && e.Name[4:5] == "-" {
			continue
		}
		if e.Type == "dir" && regexp.MustCompile(`^\d{8}/$`).MatchString(e.Name) {
			dates = append(dates, strings.TrimSuffix(e.Name, "/"))
		}
	}
	// 列表已按日期升序，倒序遍历
	platform := "x86_64-unknown-linux-gnu"
	if runtime.GOOS == "windows" {
		platform = "x86_64-pc-windows-msvc"
	}
	suffix := platform + "-install_only.tar.gz"
	for i := len(dates) - 1; i >= 0; i-- {
		date := dates[i]
		dirData, err := httpGet(base+"/"+date+"/", 30*time.Second)
		if err != nil {
			continue
		}
		entries := []pbsEntry{}
		if err := json.Unmarshal(dirData, &entries); err != nil {
			continue
		}
		for _, e := range entries {
			m := pbsVersionRe.FindStringSubmatch(e.Name)
			if m == nil || !strings.HasSuffix(e.Name, suffix) {
				continue
			}
			if strings.Contains(e.Name, "freethreaded") {
				continue
			}
			version := m[1]
			if versionWanted != "" {
				w := strings.TrimPrefix(versionWanted, "v")
				if version != w && !strings.HasPrefix(version, w+".") {
					continue
				}
			}
			fmt.Println("Python", version, "("+date+" build)")
			archivePath, err := downloadWithSidecar(base + "/" + date + "/" + e.Name)
			if err != nil {
				return "", err
			}
			defer os.RemoveAll(filepath.Dir(archivePath))
			staging, err := os.MkdirTemp(paths.Home(), ".ds-py-")
			if err != nil {
				return "", err
			}
			defer os.RemoveAll(staging)
			if _, err := safeExtractSingleRoot(archivePath, staging); err != nil {
				return "", err
			}
			dest := filepath.Join(paths.Home(), "python-"+version)
			if err := os.RemoveAll(dest); err != nil {
				return "", err
			}
			if err := os.Rename(filepath.Join(staging, "python"), dest); err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(dest, ".devswitch-version"), []byte(version), 0o644); err != nil {
				return "", err
			}
			return dest, nil
		}
	}
	return "", fmt.Errorf("没有发现匹配的 Python %s 安装包（python-build-standalone）", versionWanted)
}
