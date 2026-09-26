// Package models 定义运行时数据模型与版本解析，平移自 devswitch/models.py。
package models

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Tool string

const (
	Node   Tool = "node"
	Java   Tool = "java"
	Maven  Tool = "maven"
	Gradle Tool = "gradle"
)

var Tools = []Tool{Node, Java, Maven, Gradle}

// AppVersion 全局唯一版本号来源，发布构建用
// -ldflags "-X .../internal/models.AppVersion=x.y.z" 注入。
var AppVersion = "2.0.1-dev"

var ToolLabels = map[Tool]string{
	Node: "Node.js", Java: "Java", Maven: "Maven", Gradle: "Gradle",
}

// 各工具的 shim 命令名（不带平台后缀）。
var (
	NodeShims   = []string{"node", "npm", "npx", "corepack"}
	JavaShims   = []string{"java", "javac", "jar", "javadoc", "javap", "jshell", "keytool", "jps", "jcmd"}
	MavenShims  = []string{"mvn", "mvnDebug"}
	GradleShims = []string{"gradle"}
)

var ToolShims = map[Tool][]string{
	Node: NodeShims, Java: JavaShims, Maven: MavenShims, Gradle: GradleShims,
}

// ToolHomeVars 是各工具暴露给 IDE / 构建脚本的环境变量名。
var ToolHomeVars = map[Tool]string{
	Java: "JAVA_HOME", Maven: "MAVEN_HOME", Gradle: "GRADLE_HOME",
}

type Runtime struct {
	Tool    Tool    `json:"tool"`
	Version string  `json:"version"`
	Major   string  `json:"major"`
	Home    string  `json:"home"`
	Binary  string  `json:"binary"`
	Source  string  `json:"source"`
	Label   string  `json:"label,omitempty"`
	Vendor  string  `json:"vendor,omitempty"`
}

// Matches 与 Python 版语义一致：支持 home 路径、完整版本、大版本、前缀。
func (r *Runtime) Matches(query string) bool {
	q := strings.TrimPrefix(strings.TrimSpace(query), "v")
	if q == "" {
		return false
	}
	if strings.TrimRight(r.Home, "/") == strings.TrimRight(q, "/") {
		return true
	}
	if r.Version == q || r.Major == q {
		return true
	}
	if strings.HasPrefix(r.Version, q+".") {
		return true
	}
	return false
}

type State struct {
	Schema   int                 `json:"schema"`
	Current  map[Tool]string     `json:"current"`
	Runtimes []Runtime           `json:"runtimes"`
}

func NewState() State {
	current := make(map[Tool]string, len(Tools))
	for _, tool := range Tools {
		current[tool] = ""
	}
	return State{Schema: 1, Current: current}
}

// ForTool 返回按版本降序排列的指定工具运行时。
func (s *State) ForTool(tool Tool) []Runtime {
	items := make([]Runtime, 0)
	for _, r := range s.Runtimes {
		if r.Tool == tool {
			items = append(items, r)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return ParseVersionKey(items[i].Version).After(ParseVersionKey(items[j].Version))
	})
	return items
}

// Find 与 Python 版优先级一致：精确版本 > 大版本 > 首个匹配。
func (s *State) Find(tool Tool, query string) *Runtime {
	matches := make([]*Runtime, 0)
	for i := range s.Runtimes {
		r := &s.Runtimes[i]
		if r.Tool == tool && r.Matches(query) {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	bare := strings.TrimPrefix(query, "v")
	for _, r := range matches {
		if r.Version == bare {
			return r
		}
	}
	for _, r := range matches {
		if r.Major == bare {
			return r
		}
	}
	return matches[0]
}

func (s *State) ByHome(tool Tool, home string) *Runtime {
	home = strings.TrimRight(home, "/")
	for i := range s.Runtimes {
		if s.Runtimes[i].Tool == tool && strings.TrimRight(s.Runtimes[i].Home, "/") == home {
			return &s.Runtimes[i]
		}
	}
	return nil
}

func (s *State) CurrentRuntime(tool Tool) *Runtime {
	return s.ByHome(tool, s.Current[tool])
}

func (s *State) Upsert(r Runtime) {
	if existing := s.ByHome(r.Tool, r.Home); existing != nil {
		existing.Version = r.Version
		existing.Major = r.Major
		existing.Binary = r.Binary
		existing.Source = r.Source
		if r.Label != "" {
			existing.Label = r.Label
		}
		if r.Vendor != "" {
			existing.Vendor = r.Vendor
		}
		return
	}
	s.Runtimes = append(s.Runtimes, r)
}

var nodeVersionRe = regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)

func ParseNodeVersion(text string) string {
	text = strings.TrimSpace(text)
	if m := nodeVersionRe.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return strings.TrimPrefix(text, "v")
}

var javaVersionRe = regexp.MustCompile(`version\s+"([^"]+)"`)
var javaLooseRe = regexp.MustCompile(`(\d+\.\d+\.\d+)`)

func ParseJavaVersion(text string) string {
	if m := javaVersionRe.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	if m := javaLooseRe.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}

func MajorOf(tool Tool, version string) string {
	version = strings.TrimPrefix(version, "v")
	if tool == Java && strings.HasPrefix(version, "1.") {
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	if version == "" {
		return ""
	}
	return strings.Split(version, ".")[0]
}

var nodeLTS = map[string]bool{"18": true, "20": true, "22": true, "24": true}

func NodeLabel(version string) string {
	major := MajorOf(Node, version)
	if nodeLTS[major] {
		return "Node.js " + version + " LTS"
	}
	return "Node.js " + version
}

func JavaLabel(version, vendor string) string {
	if vendor == "" {
		vendor = "OpenJDK"
	}
	major := MajorOf(Java, version)
	if major != "" {
		return vendor + " " + major
	}
	return vendor + " " + version
}

// VersionKey 是可比较的版本序（语义对齐 Python 的 _version_key）。
type VersionKey []int

func parseVersionKey(version string) VersionKey {
	version = strings.TrimPrefix(version, "v")
	// Java 老版本 1.8.0_392 → 8.0.392
	if strings.HasPrefix(version, "1.") && len(version) > 2 && version[2] >= '0' && version[2] <= '9' {
		version = version[2:]
	}
	re := regexp.MustCompile(`\d+`)
	parts := VersionKey{}
	for _, token := range re.FindAllString(version, -1) {
		if n, err := strconv.Atoi(token); err == nil {
			parts = append(parts, n)
		}
	}
	if len(parts) == 0 {
		return VersionKey{0}
	}
	return parts
}

func ParseVersionKey(version string) VersionKey { return parseVersionKey(version) }

func (v VersionKey) After(other VersionKey) bool {
	for i := 0; i < len(v) || i < len(other); i++ {
		var a, b int
		if i < len(v) {
			a = v[i]
		}
		if i < len(other) {
			b = other[i]
		}
		if a != b {
			return a > b
		}
	}
	return false
}
