// Package downloader 下载并安装运行时，平移自 devswitch/downloader.py。
//
// 安全约束（与 Python 版一致）：
//   - 仅 http/https；
//   - host 白名单；DNS 解析后拒绝环回/私网/链路本地/保留地址；
//   - 重定向目标同样过校验（http.Client CheckRedirect）；
//   - 压缩包 sha256/sha512 校验；解压拒绝 .. 越界路径与符号链接成员。
package downloader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
)

var allowedHosts = map[string]bool{
	"nodejs.org": true, "registry.npmmirror.com": true,
	"mirrors.tuna.tsinghua.edu.cn": true, "mirrors.cloud.tencent.com": true,
	"api.adoptium.net": true, "api.github.com": true,
	"github.com": true, "objects.githubusercontent.com": true,
	"repo.maven.apache.org": true, "dlcdn.apache.org": true, "services.gradle.org": true,
}

func assertSafeURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("非法 URL：%s", raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("仅允许 http/https 协议：%s", raw)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("URL 缺少主机名：%s", raw)
	}
	if !allowedHosts[host] {
		return fmt.Errorf("主机不在下载白名单内：%s", host)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("无法解析主机：%s", host)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
			ip.IsUnspecified() || ip.IsMulticast() {
			return fmt.Errorf("拒绝非公网地址：%s (%s)", host, ip)
		}
	}
	return nil
}

var client = &http.Client{
	Timeout: 10 * time.Minute,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if err := assertSafeURL(req.URL.String()); err != nil {
			return err
		}
		if len(via) >= 10 {
			return fmt.Errorf("重定向过多")
		}
		return nil
	},
}

func httpGet(rawURL string, timeout time.Duration) ([]byte, error) {
	if err := assertSafeURL(rawURL); err != nil {
		return nil, err
	}
	noRedirect := &http.Client{Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if err := assertSafeURL(req.URL.String()); err != nil {
			return err
		}
		if len(via) >= 10 {
			return fmt.Errorf("重定向过多")
		}
		return nil
	}}
	resp, err := noRedirect.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d：%s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}

func httpGetJSON(rawURL string) (map[string]interface{}, error) {
	data, err := httpGet(rawURL, 30*time.Second)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// httpDownloadFile 流式下载并做进度提示（stdout 是终端时）。
func httpDownloadFile(rawURL, dest string) error {
	if err := assertSafeURL(rawURL); err != nil {
		return err
	}
	resp, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d：%s", resp.StatusCode, rawURL)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("下载完成（%.1f MB）\n", float64(written)/1048576)
	return nil
}

func checksumOf(path, algo string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var sum string
	switch algo {
	case "sha512":
		digest := sha512.Sum512(data)
		sum = hex.EncodeToString(digest[:])
	default:
		digest := sha256.Sum256(data)
		sum = hex.EncodeToString(digest[:])
	}
	return sum, nil
}

func verifyChecksum(path, expected, algo string) error {
	actual, err := checksumOf(path, algo)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, strings.TrimSpace(expected)) {
		return fmt.Errorf("%s 校验失败（期望 %s，实际 %s）", algo, expected, actual)
	}
	fmt.Println(algo + " 校验通过")
	return nil
}

// safeExtractSingleRoot 解压「单一顶层目录」的压缩包到 destParent。
// 拒绝 .. 越界路径与符号链接/硬链接成员。
func safeExtractSingleRoot(archivePath, destParent string) (string, error) {
	staging, err := os.MkdirTemp("", "ds-extract-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)

	switch strings.ToLower(filepath.Ext(archivePath)) {
	case ".zip":
		err = extractZip(archivePath, staging)
	default:
		err = extractTar(archivePath, staging)
	}
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(staging)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return "", fmt.Errorf("压缩包内不是单一顶层目录：%s", filepath.Base(archivePath))
	}
	final := filepath.Join(destParent, entries[0].Name())
	if err := os.RemoveAll(final); err != nil {
		return "", err
	}
	if err := os.Rename(filepath.Join(staging, entries[0].Name()), final); err != nil {
		return "", err
	}
	return final, nil
}

func memberInside(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func extractZip(archivePath, staging string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	root, _ := filepath.Abs(staging)
	for _, f := range reader.File {
		if strings.Contains(f.Name, "..") {
			return fmt.Errorf("压缩包包含越界路径：%s", f.Name)
		}
		target := filepath.Join(staging, f.Name)
		abs, _ := filepath.Abs(target)
		if !memberInside(root, abs) {
			return fmt.Errorf("压缩包包含越界路径：%s", f.Name)
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			in.Close()
			return err
		}
		_, err = io.Copy(out, in)
		in.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTar(archivePath, staging string) error {
	data, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer data.Close()

	var reader io.Reader = data
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".gz"):
		gz, err := gzip.NewReader(data)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader = gz
	case strings.HasSuffix(lower, ".xz"):
		xr, err := xz.NewReader(data)
		if err != nil {
			return err
		}
		reader = xr
	}

	root, _ := filepath.Abs(staging)
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("压缩包包含越界路径：%s", header.Name)
		}
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return fmt.Errorf("压缩包包含链接成员，拒绝解压：%s", header.Name)
		}
		target := filepath.Join(staging, header.Name)
		abs, _ := filepath.Abs(target)
		if !memberInside(root, abs) {
			return fmt.Errorf("压缩包包含越界路径：%s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func machineArch() (string, error) {
	arch := os.Getenv("PROCESSOR_ARCHITECTURE")
	if arch == "" {
		arch = runtimeArch()
	}
	switch strings.ToLower(arch) {
	case "amd64", "x86_64", "x64":
		return "x64", nil
	case "arm64", "aarch64":
		return "arm64", nil
	}
	return "", fmt.Errorf("不支持的 CPU 架构：%s", arch)
}

func runtimeArch() string {
	switch filepath.Separator {
	case '\\':
		return os.Getenv("PROCESSOR_ARCHITECTURE")
	default:
		out, err := os.ReadFile("/proc/sys/kernel/arch") // best-effort
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// 镜像
// ---------------------------------------------------------------------------

type mirrorSetting struct {
	NodeDist      string
	JavaDownload  string // "" 官方 / "tuna"
}

var mirrors = map[string]mirrorSetting{
	"official":   {NodeDist: "https://nodejs.org/dist"},
	"npmmirror":  {NodeDist: "https://registry.npmmirror.com/-/binary/node", JavaDownload: "tuna"},
	"tuna":       {NodeDist: "https://mirrors.tuna.tsinghua.edu.cn/nodejs-release", JavaDownload: "tuna"},
}

func mirrorSettingFor(mirror string) (mirrorSetting, error) {
	if mirror == "" {
		mirror = os.Getenv("DEVSWITCH_MIRROR")
	}
	if mirror == "" {
		mirror = "official"
	}
	setting, ok := mirrors[mirror]
	if !ok {
		return mirrorSetting{}, fmt.Errorf("未知镜像：%s（可选 official / npmmirror / tuna）", mirror)
	}
	return setting, nil
}

// ---------------------------------------------------------------------------
// Node
// ---------------------------------------------------------------------------

type nodeIndexEntry struct {
	Version string `json:"version"`
}

func resolveNodeVersion(major, mirror string) (string, error) {
	setting, err := mirrorSettingFor(mirror)
	if err != nil {
		return "", err
	}
	data, err := httpGet(setting.NodeDist+"/index.json", 30*time.Second)
	if err != nil {
		return "", err
	}
	entries := []nodeIndexEntry{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return "", err
	}
	prefix := "v" + major + "."
	for _, entry := range entries {
		if strings.HasPrefix(entry.Version, prefix) && !strings.HasSuffix(entry.Version, "-nightly") {
			return strings.TrimPrefix(entry.Version, "v"), nil
		}
	}
	return "", fmt.Errorf("镜像上没有发现 Node %s 版本系列", major)
}

func installNode(major, mirror string) (string, error) {
	version, err := resolveNodeVersion(major, mirror)
	if err != nil {
		return "", err
	}
	setting, _ := mirrorSettingFor(mirror)
	arch, err := machineArch()
	if err != nil {
		return "", err
	}
	base := setting.NodeDist + "/v" + version
	filename := fmt.Sprintf("node-v%s-win-%s.zip", version, arch)
	if !paths.IsWindows {
		filename = fmt.Sprintf("node-v%s-linux-%s.tar.xz", version, arch)
	}
	fmt.Printf("Node %s（%s 架构，镜像 %s）\n", version, arch, setting.NodeDist)
	temp, _ := os.MkdirTemp("", "ds-download-")
	defer os.RemoveAll(temp)
	archivePath := filepath.Join(temp, filename)
	if err := httpDownloadFile(base+"/"+filename, archivePath); err != nil {
		return "", err
	}
	if sum := nodeChecksum(base, filename); sum != "" {
		if err := verifyChecksum(archivePath, sum, "sha256"); err != nil {
			return "", err
		}
	}
	return safeExtractSingleRoot(archivePath, paths.Home())
}

func nodeChecksum(base, filename string) string {
	data, err := httpGet(base+"/SHASUMS256.txt", 20*time.Second)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if len(parts) == 2 && strings.TrimPrefix(parts[1], "*") == filename {
			return parts[0]
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Java（Adoptium）
// ---------------------------------------------------------------------------

func installJava(major, mirror string) (string, error) {
	setting, err := mirrorSettingFor(mirror)
	if err != nil {
		return "", err
	}
	arch, err := machineArch()
	if err != nil {
		return "", err
	}
	adoptiumArch := map[string]string{"x64": "x64", "arm64": "aarch64"}[arch]
	system := "linux"
	if paths.IsWindows {
		system = "windows"
	}
	apiURL := fmt.Sprintf(
		"https://api.adoptium.net/v3/assets/latest/%s/hotspot?os=%s&architecture=%s&image_type=jdk&vendor=eclipse",
		major, system, adoptiumArch)
	data, err := httpGet(apiURL, 30*time.Second)
	if err != nil {
		return "", err
	}
	assets := []map[string]interface{}{}
	if err := json.Unmarshal(data, &assets); err != nil {
		return "", err
	}
	if len(assets) == 0 {
		return "", fmt.Errorf("Adoptium 上没有发现 JDK %s（%s / %s）", major, system, arch)
	}
	binary, _ := assets[0]["binary"].(map[string]interface{})
	pkg, _ := binary["package"].(map[string]interface{})
	link, _ := pkg["link"].(string)
	name, _ := pkg["name"].(string)
	checksum, _ := pkg["checksum"].(string)
	if link == "" || name == "" {
		return "", fmt.Errorf("Adoptium 返回缺少下载信息")
	}
	if setting.JavaDownload == "tuna" {
		link = fmt.Sprintf("https://mirrors.tuna.tsinghua.edu.cn/Adoptium/%s/jdk/%s/%s/%s",
			major, adoptiumArch, system, name)
	}
	fmt.Printf("Temurin JDK %s（%s / %s）\n", major, system, arch)
	temp, _ := os.MkdirTemp("", "ds-download-")
	defer os.RemoveAll(temp)
	archivePath := filepath.Join(temp, name)
	if err := httpDownloadFile(link, archivePath); err != nil {
		return "", err
	}
	if checksum != "" {
		if err := verifyChecksum(archivePath, checksum, "sha256"); err != nil {
			return "", err
		}
	}
	return safeExtractSingleRoot(archivePath, paths.Home())
}

// ---------------------------------------------------------------------------
// Maven / Gradle（sidecar 校验文件）
// ---------------------------------------------------------------------------

func downloadWithSidecar(rawURL string) (string, error) {
	temp, _ := os.MkdirTemp("", "ds-download-")
	checksum, algo := "", "sha256"
	for _, suffix := range []string{".sha256", ".sha512"} {
		if data, err := httpGet(rawURL+suffix, 15*time.Second); err == nil {
			first := strings.Fields(strings.TrimSpace(string(data)))
			if len(first) > 0 {
				checksum = first[0]
				algo = strings.TrimPrefix(suffix, ".")
			}
			break
		}
	}
	archivePath := filepath.Join(temp, filepath.Base(rawURL))
	if err := httpDownloadFile(rawURL, archivePath); err != nil {
		os.RemoveAll(temp)
		return "", err
	}
	if checksum != "" {
		if err := verifyChecksum(archivePath, checksum, algo); err != nil {
			os.RemoveAll(temp)
			return "", err
		}
	}
	return archivePath, nil
}

var mavenMetaVersionRe = regexp.MustCompile(`<version>(\d+\.\d+\.\d+)</version>`)

func installMaven(major, mirror string) (string, error) {
	setting, err := mirrorSettingFor(mirror)
	if err != nil {
		return "", err
	}
	data, err := httpGet(
		"https://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/maven-metadata.xml",
		30*time.Second)
	if err != nil {
		return "", err
	}
	versions := mavenMetaVersionRe.FindAllStringSubmatch(string(data), -1)
	version := ""
	for _, m := range versions {
		if strings.HasPrefix(m[1], major+".") {
			version = m[1]
		}
	}
	if version == "" {
		return "", fmt.Errorf("没有发现 Maven %s 版本系列", major)
	}
	base := fmt.Sprintf("https://dlcdn.apache.org/maven/maven-3/%s/binaries", version)
	if setting.JavaDownload == "tuna" {
		base = fmt.Sprintf("https://mirrors.tuna.tsinghua.edu.cn/apache/maven/maven-3/%s/binaries", version)
	}
	filename := fmt.Sprintf("apache-maven-%s-bin.zip", version)
	fmt.Printf("Apache Maven %s\n", version)
	archivePath, err := downloadWithSidecar(base + "/" + filename)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(filepath.Dir(archivePath))
	return safeExtractSingleRoot(archivePath, paths.Home())
}

func installGradle(major, mirror string) (string, error) {
	data, err := httpGet("https://services.gradle.org/versions/all", 30*time.Second)
	if err != nil {
		return "", err
	}
	entries := []map[string]interface{}{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return "", err
	}
	version := ""
	for _, entry := range entries {
		if entry["snapshot"] == true || entry["rcFor"] != nil || entry["milestoneFor"] != nil {
			continue
		}
		v, _ := entry["version"].(string)
		if strings.HasPrefix(v, major+".") {
			version = v
			break
		}
	}
	if version == "" {
		return "", fmt.Errorf("没有发现 Gradle %s 版本系列", major)
	}
	setting, _ := mirrorSettingFor(mirror)
	base := "https://services.gradle.org/distributions"
	if setting.JavaDownload == "tuna" {
		base = "https://mirrors.cloud.tencent.com/gradle"
	}
	filename := fmt.Sprintf("gradle-%s-bin.zip", version)
	fmt.Printf("Gradle %s\n", version)
	archivePath, err := downloadWithSidecar(base + "/" + filename)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(filepath.Dir(archivePath))
	return safeExtractSingleRoot(archivePath, paths.Home())
}

// ---------------------------------------------------------------------------
// 汇总入口
// ---------------------------------------------------------------------------

// LatestReleaseVersion 只读查询最新发布版本。
func LatestReleaseVersion() (string, error) {
	data, err := httpGetJSON("https://api.github.com/repos/xyztony999/devswitch/releases/latest")
	if err != nil {
		return "", err
	}
	tag, _ := data["tag_name"].(string)
	return strings.TrimPrefix(tag, "v"), nil
}

// InstallTool 下载并解压到用户目录，返回安装 home。
// 切换等编排由 service 层完成（避免循环依赖）。
func InstallTool(tool models.Tool, version, mirror string) (string, error) {
	switch tool {
	case models.Node:
		return installNode(version, mirror)
	case models.Java:
		return installJava(version, mirror)
	case models.Maven:
		return installMaven(version, mirror)
	case models.Gradle:
		return installGradle(version, mirror)
	}
	return "", fmt.Errorf("暂不支持下载该工具：%s", tool)
}
