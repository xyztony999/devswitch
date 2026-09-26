// Package detect 扫描本机运行时，平移自 devswitch/detect.py。
//
// 与 Python 版的实现差异：全程不执行子进程，版本信息一律来自
// 安装目录的静态元数据——
//   - Java：home/release 文件（JAVA_VERSION / IMPLEMENTOR，JDK 8+ 均有）
//   - Maven：lib/maven-core-*.jar 文件名
//   - Gradle：lib/ 下 gradle 核心 jar 文件名
//   - Node：目录名（node-vX.Y.Z-平台-架构），Windows 系统安装再查注册表卸载信息
// 优点是毫秒级完成（无需等待 JVM 启动）；代价是无元数据的罕见布局
// （如手工裸拷贝的二进制）不可识别，可用 devswitch import 手动补录。
package detect

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
)

var isWindows = runtime.GOOS == "windows"

func normcase(s string) string {
	if isWindows {
		return strings.ToLower(s)
	}
	return s
}

func resolvePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return real
}

func uniquePaths(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value == "" {
			continue
		}
		key := normcase(resolvePath(value))
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

// probeWhitelist：whichAll 只探测这些内部常量命令名。
var probeWhitelist = map[string]bool{
	"node": true, "java": true, "mvn": true, "gradle": true,
}

// whichAll 枚举 PATH 上的全部命中（纯目录扫描，无进程执行）。
func whichAll(name string) []string {
	if !probeWhitelist[name] {
		return nil
	}
	found := []string{}
	extensions := []string{""}
	if isWindows {
		extensions = []string{".exe", ".cmd", ".bat"}
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		dir = strings.Trim(dir, `"`)
		if dir == "" {
			continue
		}
		for _, ext := range extensions {
			candidate := filepath.Join(dir, name+ext)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				found = append(found, candidate)
				break
			}
		}
	}
	return uniquePaths(found)
}

func isShim(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), paths.ShimMarker)
}

// looksSystem 判断是否系统级安装（大小写自算，跨宿主语义一致）。
func looksSystem(home string) bool {
	if isWindows {
		lower := strings.ToLower(filepath.ToSlash(home))
		if strings.Contains(lower, "/program files/") || strings.Contains(lower, "/program files (x86)/") ||
			strings.HasSuffix(lower, "/program files") || strings.HasSuffix(lower, "/program files (x86)") {
			return true
		}
		for _, varName := range []string{"ProgramFiles", "ProgramFiles(x86)", "ProgramW6432"} {
			base := os.Getenv(varName)
			if base != "" && strings.HasPrefix(lower, strings.ToLower(filepath.ToSlash(base))+"/") {
				return true
			}
		}
		windir := os.Getenv("SystemRoot")
		if windir == "" {
			windir = `C:\Windows`
		}
		return strings.HasPrefix(lower, strings.ToLower(filepath.ToSlash(windir))+"/")
	}
	return strings.HasPrefix(home, "/usr")
}

func sourceOf(home string) string {
	if looksSystem(home) {
		return "system"
	}
	return "local"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func fileExistsAny(pathsToTry ...string) bool {
	for _, p := range pathsToTry {
		if fileExists(p) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// 元数据解析
// ---------------------------------------------------------------------------

var nodeDirVersionRe = regexp.MustCompile(`^node-v(\d+\.\d+\.\d+)-`)

// nodeVersionFromDirName 从 node-v22.23.3-win-x64 之类的目录名取版本。
func nodeVersionFromDirName(home string) string {
	m := nodeDirVersionRe.FindStringSubmatch(filepath.Base(filepath.Clean(home)))
	if m != nil {
		return m[1]
	}
	return ""
}

var releaseVersionRe = regexp.MustCompile(`(?m)^JAVA_VERSION="([^"]+)"`)
var releaseVendorRe = regexp.MustCompile(`(?m)^IMPLEMENTOR="([^"]+)"`)

// javaVersionFromRelease 读 home/release（JDK 8+ 标准文件）。
func javaVersionFromRelease(home string) (version string, vendor string) {
	data, err := os.ReadFile(filepath.Join(home, "release"))
	if err != nil {
		return "", ""
	}
	text := string(data)
	if m := releaseVersionRe.FindStringSubmatch(text); m != nil {
		version = m[1]
	}
	if m := releaseVendorRe.FindStringSubmatch(text); m != nil {
		vendor = m[1]
	}
	return version, vendor
}

var mavenCoreJarRe = regexp.MustCompile(`^maven-core-(\d+\.\d+(?:\.\d+)?(?:-[A-Za-z0-9.]+)?)\.jar$`)

// mavenVersionFromLibs 从 lib/maven-core-X.Y.Z.jar 取版本。
func mavenVersionFromLibs(home string) string {
	libDir := filepath.Join(home, "lib")
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if m := mavenCoreJarRe.FindStringSubmatch(entry.Name()); m != nil {
			return m[1]
		}
	}
	return ""
}

var gradleCoreJarRe = regexp.MustCompile(`^gradle-(?:launcher|core|base|wrapper)-(?:\d+\.\d+(?:\.\d+)?(?:-[A-Za-z0-9.]+)?)[\.-].*\.jar$`)

// gradleVersionFromLibs 从 lib 下的核心 jar 文件名取版本。
func gradleVersionFromLibs(home string) string {
	libDir := filepath.Join(home, "lib")
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "gradle-") && strings.HasSuffix(name, ".jar") &&
			!strings.Contains(name, "api") && !strings.Contains(name, "internal") {
			// gradle-launcher-8.14.5.jar / gradle-wrapper-main-8.14.jar 等
			inner := strings.TrimSuffix(strings.TrimPrefix(name, "gradle-"), ".jar")
			parts := strings.SplitN(inner, "-", 2)
			if len(parts) == 2 && regexp.MustCompile(`^\d`).MatchString(parts[1]) {
				return parts[1]
			}
		}
	}
	return ""
}

func normalizeJavaVendor(vendor string) string {
	switch {
	case vendor == "":
		return "OpenJDK"
	case strings.Contains(strings.ToLower(vendor), "temurin"), strings.Contains(strings.ToLower(vendor), "eclipse"):
		return "Temurin"
	case strings.Contains(strings.ToLower(vendor), "microsoft"):
		return "Microsoft"
	case strings.Contains(strings.ToLower(vendor), "zulu"), strings.Contains(strings.ToLower(vendor), "azul"):
		return "Zulu"
	case strings.Contains(strings.ToLower(vendor), "corretto"), strings.Contains(strings.ToLower(vendor), "amazon"):
		return "Corretto"
	case strings.Contains(strings.ToLower(vendor), "liberica"), strings.Contains(strings.ToLower(vendor), "bellsoft"):
		return "Liberica"
	case strings.Contains(strings.ToLower(vendor), "graal"):
		return "GraalVM"
	case strings.Contains(strings.ToLower(vendor), "oracle"):
		return "Oracle"
	default:
		return vendor
	}
}

// ---------------------------------------------------------------------------
// 各工具 inspect
// ---------------------------------------------------------------------------

func nodeHomeFromBinary(binary string) (string, bool) {
	resolved := resolvePath(binary)
	name := filepath.Base(resolved)
	if isWindows {
		if name != "node.exe" {
			return "", false
		}
		return filepath.Dir(resolved), true
	}
	if name != "node" {
		return "", false
	}
	parent := filepath.Dir(resolved)
	if filepath.Base(parent) == "bin" {
		return filepath.Dir(parent), true
	}
	return parent, true
}

func inspectNode(home string) *models.Runtime {
	var binary string
	if isWindows {
		binary = filepath.Join(home, "node.exe")
	} else {
		binary = filepath.Join(home, "bin", "node")
	}
	if !fileExists(binary) {
		return nil
	}
	version := nodeVersionFromDirName(home)
	if version == "" && isWindows {
		version = nodeVersionFromRegistry(home)
	}
	if version == "" {
		return nil // 无元数据的布局（如 MSI 系统安装），可 devswitch import 补录
	}
	return &models.Runtime{
		Tool: models.Node, Version: version, Major: models.MajorOf(models.Node, version),
		Home: home, Binary: binary,
		Source: sourceOf(home), Label: models.NodeLabel(version), Vendor: "Node.js",
	}
}

func javaHomeFromBinary(binary string) (string, bool) {
	resolved := resolvePath(binary)
	want := "java"
	if isWindows {
		want = "java.exe"
	}
	if filepath.Base(resolved) != want {
		return "", false
	}
	parent := filepath.Dir(resolved)
	if filepath.Base(parent) == "jre" {
		return filepath.Dir(filepath.Dir(parent)), true
	}
	if filepath.Base(parent) == "bin" {
		return filepath.Dir(parent), true
	}
	return parent, true
}

func inspectJava(home string) *models.Runtime {
	exe := "java"
	if isWindows {
		exe = "java.exe"
	}
	binary := filepath.Join(home, "bin", exe)
	if !fileExists(binary) {
		binary = filepath.Join(home, "jre", "bin", exe)
		if !fileExists(binary) {
			return nil
		}
	}
	binary = resolvePath(binary)
	if resolvedHome, ok := javaHomeFromBinary(binary); ok {
		home = resolvedHome
	}
	version, vendor := javaVersionFromRelease(home)
	if version == "" {
		return nil
	}
	vendor = normalizeJavaVendor(vendor)
	return &models.Runtime{
		Tool: models.Java, Version: version, Major: models.MajorOf(models.Java, version),
		Home: home, Binary: binary,
		Source: sourceOf(home), Label: models.JavaLabel(version, vendor), Vendor: vendor,
	}
}

func inspectMaven(home string) *models.Runtime {
	binary := filepath.Join(home, "bin", "mvn.cmd")
	if !isWindows {
		binary = filepath.Join(home, "bin", "mvn")
	}
	if !fileExists(binary) {
		return nil
	}
	version := mavenVersionFromLibs(home)
	if version == "" {
		return nil
	}
	return &models.Runtime{
		Tool: models.Maven, Version: version, Major: models.MajorOf(models.Maven, version),
		Home: home, Binary: binary,
		Source: sourceOf(home), Label: "Apache Maven " + version, Vendor: "Apache",
	}
}

func inspectGradle(home string) *models.Runtime {
	binary := filepath.Join(home, "bin", "gradle.bat")
	if !isWindows {
		binary = filepath.Join(home, "bin", "gradle")
	}
	if !fileExists(binary) {
		return nil
	}
	version := gradleVersionFromLibs(home)
	if version == "" {
		return nil
	}
	return &models.Runtime{
		Tool: models.Gradle, Version: version, Major: models.MajorOf(models.Gradle, version),
		Home: home, Binary: binary,
		Source: sourceOf(home), Label: "Gradle " + version, Vendor: "Gradle",
	}
}

func Inspectors() map[models.Tool]func(string) *models.Runtime {
	return map[models.Tool]func(string) *models.Runtime{
		models.Node: inspectNode, models.Java: inspectJava,
		models.Maven: inspectMaven, models.Gradle: inspectGradle,
	}
}

// ---------------------------------------------------------------------------
// 候选目录
// ---------------------------------------------------------------------------

func globAll(patterns []string) []string {
	found := []string{}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		found = append(found, matches...)
	}
	return found
}

func programFilesVars() []string {
	return []string{
		os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramW6432"),
	}
}

func candidateNodeHomes() []string {
	home := paths.Home()
	var patterns []string
	if isWindows {
		for _, base := range programFilesVars() {
			if base != "" {
				patterns = append(patterns, filepath.Join(base, "nodejs"))
			}
		}
		nvmRoot := os.Getenv("NVM_HOME")
		if nvmRoot == "" {
			nvmRoot = filepath.Join(home, "AppData", "Roaming", "nvm")
		}
		patterns = append(patterns, filepath.Join(nvmRoot, "v*"))
		patterns = append(patterns,
			filepath.Join(home, "node-v*-win-*"),
			filepath.Join(home, "scoop", "apps", "nodejs", "*"),
			filepath.Join(home, ".local", "share", "node-v*-win-*"),
		)
	} else {
		patterns = append(patterns,
			filepath.Join(home, "node-v*-linux-*"),
			filepath.Join(home, ".local", "share", "node-v*-linux-*"),
			filepath.Join(home, ".nvm", "versions", "node", "*"),
			"/usr/local/n/versions/node/*",
			"/opt/node*",
			"/usr/local/lib/nodejs/*",
		)
	}
	found := globAll(patterns)
	for _, binary := range whichAll("node") {
		if isShim(binary) {
			continue
		}
		if home, ok := nodeHomeFromBinary(binary); ok {
			found = append(found, home)
		}
	}
	return uniquePaths(found)
}

func candidateJavaHomes() []string {
	home := paths.Home()
	found := []string{}
	if isWindows {
		for _, base := range programFilesVars() {
			if base == "" {
				continue
			}
			for _, vendorDir := range []string{"Java", "Eclipse Adoptium", "Microsoft", "Zulu", "Amazon Corretto", "BellSoft"} {
				found = append(found, globAll([]string{filepath.Join(base, vendorDir, "*")})...)
			}
			found = append(found, globAll([]string{filepath.Join(base, "Android", "Android Studio", "jbr")})...)
		}
		found = append(found, globAll([]string{
			filepath.Join(home, "jdk*"), filepath.Join(home, "java*"), filepath.Join(home, ".jdks", "*"),
		})...)
	} else {
		for _, item := range globAll([]string{"/usr/lib/jvm/*"}) {
			name := filepath.Base(item)
			if name == "default-java" || strings.HasPrefix(name, "openjdk-") {
				continue
			}
			found = append(found, item)
		}
		found = append(found, globAll([]string{
			filepath.Join(home, "jdk*"), filepath.Join(home, "java*"),
			filepath.Join(home, ".sdkman", "candidates", "java", "*"),
			"/opt/jdk*", "/opt/java*", "/usr/java/*",
		})...)
	}
	if envHome := os.Getenv("JAVA_HOME"); envHome != "" {
		found = append(found, envHome)
	}
	for _, binary := range whichAll("java") {
		if isShim(binary) {
			continue
		}
		if home, ok := javaHomeFromBinary(binary); ok {
			found = append(found, home)
		}
	}
	return uniquePaths(found)
}

func candidateBuildtoolHomes(tool models.Tool) []string {
	home := paths.Home()
	namePattern := "apache-maven-*"
	plain := "maven"
	if tool == models.Gradle {
		namePattern = "gradle-*"
		plain = "gradle"
	}
	var patterns []string
	if isWindows {
		patterns = append(patterns,
			filepath.Join(home, namePattern),
			filepath.Join(home, ".local", "share", namePattern),
			filepath.Join(home, "scoop", "apps", plain, "*"),
		)
		for _, base := range programFilesVars() {
			if base == "" {
				continue
			}
			if tool == models.Maven {
				patterns = append(patterns, filepath.Join(base, "apache-maven-*"))
			} else {
				patterns = append(patterns, filepath.Join(base, "*gradle*"))
			}
		}
	} else {
		optPattern := "/opt/apache-maven-*"
		if tool == models.Gradle {
			optPattern = "/opt/gradle-*"
		}
		patterns = append(patterns,
			filepath.Join(home, namePattern),
			filepath.Join(home, ".sdkman", "candidates", string(tool), "*"),
			optPattern,
			"/usr/share/"+string(tool)+"*",
		)
	}
	found := globAll(patterns)
	envVar := "MAVEN_HOME"
	cmd := "mvn"
	if tool == models.Gradle {
		envVar = "GRADLE_HOME"
		cmd = "gradle"
	}
	if envHome := os.Getenv(envVar); envHome != "" {
		found = append(found, envHome)
	}
	for _, binary := range whichAll(cmd) {
		if isShim(binary) {
			continue
		}
		found = append(found, filepath.Dir(filepath.Dir(binary)))
	}
	return uniquePaths(found)
}

// ---------------------------------------------------------------------------
// 扫描与推断
// ---------------------------------------------------------------------------

type probe struct {
	tool      models.Tool
	homes     func() []string
	inspector func(string) *models.Runtime
}

func ScanRuntimes() []models.Runtime {
	probes := []probe{
		{models.Node, candidateNodeHomes, inspectNode},
		{models.Java, candidateJavaHomes, inspectJava},
		{models.Maven, func() []string { return candidateBuildtoolHomes(models.Maven) }, inspectMaven},
		{models.Gradle, func() []string { return candidateBuildtoolHomes(models.Gradle) }, inspectGradle},
	}
	found := []models.Runtime{}
	seen := map[string]bool{}
	for _, p := range probes {
		for _, home := range p.homes() {
			runtime := p.inspector(home)
			if runtime == nil {
				continue
			}
			key := string(p.tool) + ":" + normcase(runtime.Binary)
			if seen[key] {
				continue
			}
			seen[key] = true
			found = append(found, *runtime)
		}
	}
	// 排序对齐 Python 版 (tool, version) 双键倒序
	for i := 1; i < len(found); i++ {
		for j := i; j > 0; j-- {
			a, b := found[j-1], found[j]
			if string(a.Tool) < string(b.Tool) ||
				(string(a.Tool) == string(b.Tool) && models.ParseVersionKey(a.Version).After(models.ParseVersionKey(b.Version))) {
				break
			}
			found[j-1], found[j] = found[j], found[j-1]
		}
	}
	return found
}

func InferActive(runtimes []models.Runtime, tool models.Tool) *models.Runtime {
	cmd := string(tool)
	if tool == models.Maven {
		cmd = "mvn"
	}
	for _, binary := range whichAll(cmd) {
		if isShim(binary) {
			continue
		}
		resolved := resolvePath(binary)
		for i := range runtimes {
			if runtimes[i].Tool == tool && runtimes[i].Binary == resolved {
				return &runtimes[i]
			}
		}
	}
	if tool == models.Java {
		envHome := strings.TrimRight(os.Getenv("JAVA_HOME"), "/\\")
		for i := range runtimes {
			home := strings.TrimRight(runtimes[i].Home, "/\\")
			if home == envHome || (isWindows && normcase(home) == normcase(envHome)) {
				return &runtimes[i]
			}
		}
	}
	for i := range runtimes {
		if runtimes[i].Tool == tool {
			return &runtimes[i]
		}
	}
	return nil
}
