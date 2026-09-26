// Package uicore 是 GUI 的平台无关核心（状态 payload + 消息状态机），
// 平移自 devswitch/uicore.py。宿主（webview/GTK）只负责 IO 回调。
package uicore

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xyztony999/devswitch/internal/downloader"
	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
	"github.com/xyztony999/devswitch/internal/service"
	"github.com/xyztony999/devswitch/internal/store"
)

func binDir() string   { return paths.LocalBin() }
func configDir() string { return paths.ConfigDir() }

// BRIDGE_ADAPTER 注入 webkit 兼容层：前端 bundle 只认
// window.webkit.messageHandlers.devswitch。宿主提供 pywebview.api（GTK/WebView2
// 旧宿主）或 __nativeSend（Go webview 宿主）之一即可。
const BRIDGE_ADAPTER = `<script>
(function () {
  function install() {
    if (window.webkit && window.webkit.messageHandlers) { return true; }
    var api = window.pywebview && window.pywebview.api;
    if (api && api.send) {
      window.webkit = {
        messageHandlers: { devswitch: { postMessage: function (m) { api.send(m); } } }
      };
      return true;
    }
    if (typeof window.__nativeSend === 'function') {
      window.webkit = {
        messageHandlers: { devswitch: { postMessage: function (m) { window.__nativeSend(m); } } }
      };
      return true;
    }
    return false;
  }
  if (!install()) {
    var tries = 0;
    var timer = setInterval(function () {
      if (install() || ++tries > 200) { clearInterval(timer); }
    }, 50);
  }
})();
</script>`

// InlineHTML 返回注入桥接适配后的单文件 UI。
func InlineHTML(uiHTML string) string {
	if idx := strings.Index(uiHTML, "<head>"); idx >= 0 {
		head := idx + len("<head>")
		return uiHTML[:head] + BRIDGE_ADAPTER + uiHTML[head:]
	}
	return BRIDGE_ADAPTER + uiHTML
}

// StateScript 生成推送脚本（与 Python 版 window.__setState 协议一致）。
func StateScript(payload map[string]interface{}) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	// json.Marshal 会转义 <>&；前端 JSON.parse 不受影响，无需取消转义。
	return fmt.Sprintf("window.__setState && window.__setState(%s);", string(data))
}

// IO 是宿主必须提供的回调集合。
type IO interface {
	Push(payload map[string]interface{})       // 推送状态到前端
	RefreshTray()                              // 重建托盘菜单
	Notify(title, body string)                 // 系统通知
	ChooseFolder(title string) (string, error) // 目录选择对话框
	IsVisible() bool                           // 主窗口是否可见
	RunAsync(fn func())                        // 后台执行（下载等耗时操作）
	Dispatch(fn func())                        // 调度回 UI 线程
	OpenURL(url string) error                  // 用系统默认浏览器打开（仅白名单 URL）
}

// 更新检查只允许打开项目自身的页面。
func isAllowedURL(url string) bool {
	for _, prefix := range []string{
		"https://github.com/xyztony999/devswitch/",
	} {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

// Controller 是唯一行为大脑：页签、选择、下载状态与消息分发。
type Controller struct {
	io         IO
	Page       string
	Selected   string
	Installing bool
	ready      bool
	latest     string // 最新发布版本（空 = 尚未检查）
	checking   bool
	fixing     bool
}

func NewController(io IO) *Controller {
	return &Controller{io: io, Page: "node"}
}

func (c *Controller) Ready() {
	c.ready = true
	c.Push()
	// 启动后延迟静默检查更新（只读，失败不打扰）。
	c.io.RunAsync(func() {
		time.Sleep(3 * time.Second)
		c.checkUpdate(false)
	})
}

func (c *Controller) Push(flash ...map[string]interface{}) {
	payload := c.BuildState(nil, flash...)
	if sel, ok := payload["selected"].(string); ok && sel != "" {
		c.Selected = sel
	}
	c.io.Push(payload)
}

// BuildState 构造与 Python 版字段兼容的完整状态。
func (c *Controller) BuildState(flash map[string]interface{}, extraFlash ...map[string]interface{}) map[string]interface{} {
	if flash == nil && len(extraFlash) > 0 {
		flash = extraFlash[0]
	}
	state := store.Load()
	current := map[string]interface{}{}
	for _, tool := range models.Tools {
		if r := state.CurrentRuntime(tool); r != nil {
			current[string(tool)] = runtimeDict(r)
		} else {
			current[string(tool)] = nil
		}
	}
	runtimes := []map[string]interface{}{}
	selected := c.Selected
	for _, tool := range models.Tools {
		if string(tool) != c.Page {
			continue
		}
		for _, r := range state.ForTool(tool) {
			runtimes = append(runtimes, runtimeDict(&r))
		}
	}
	issues := service.DoctorIssues()
	issueList := make([]map[string]interface{}, 0, len(issues))
	for _, issue := range issues {
		issueList = append(issueList, map[string]interface{}{
			"level": issue.Level, "code": issue.Code, "message": issue.Message,
		})
	}
	var flashValue interface{}
	if flash != nil {
		flashValue = flash
	}
	if selected == "" && c.Page != "doctor" {
		if r := state.CurrentRuntime(models.Tool(c.Page)); r != nil {
			selected = r.Home
		} else if len(runtimes) > 0 {
			if home, ok := runtimes[0]["home"].(string); ok {
				selected = home
			}
		}
	}
	if c.Page == "doctor" && selected == "" {
		selected = "0"
	}
	return map[string]interface{}{
		"page":       c.Page,
		"version":    models.AppVersion,
		"installing": c.Installing,
		"current":    current,
		"runtimes":   runtimes,
		"issues":     issueList,
		"selected":   selected,
		"flash":      flashValue,
		"latest":     c.latest,
		"updateAvailable": c.latest != "" && c.latest != models.AppVersion,
		"updateChecking":  c.checking,
		"paths": map[string]interface{}{
			"localBin": binDir(), "config": configDir(),
		},
	}
}

func runtimeDict(r *models.Runtime) map[string]interface{} {
	return map[string]interface{}{
		"tool": string(r.Tool), "version": r.Version, "major": r.Major,
		"home": r.Home, "binary": r.Binary, "source": r.Source,
		"label": r.Label, "vendor": r.Vendor,
	}
}

// HandleMessage 处理前端 op 消息（与 Python 协议一致）。
func (c *Controller) HandleMessage(data map[string]interface{}) {
	switch data["op"] {
	case "page":
		if page, ok := data["page"].(string); ok && page != "" {
			c.Page = page
		} else {
			c.Page = "node"
		}
		c.Selected = ""
		c.Push()
	case "select":
		c.Selected, _ = data["home"].(string)
		c.Push()
	case "scan":
		service.MergeScan()
		service.EnsureApplied()
		c.Selected = ""
		c.Push(map[string]interface{}{"text": "已重新扫描本机运行时。", "kind": "ok"})
		c.io.RefreshTray()
	case "use":
		home, _ := data["home"].(string)
		c.ApplyUse(models.Tool(c.Page), home, false)
	case "install":
		version, _ := data["version"].(string)
		mirror, _ := data["mirror"].(string)
		c.StartInstall(version, mirror)
	case "import":
		c.doImport()
	case "fix":
		if c.fixing {
			return
		}
		c.fixing = true
		c.Push(map[string]interface{}{"text": "正在修复……", "kind": "ok"})
		// 修复含多次注册表写入与环境广播（同步等待各窗口响应），
		// 在 UI 线程执行会把窗口卡住十几秒，状态迟迟不刷新；放后台。
		c.io.RunAsync(func() {
			lines := service.DoctorFix()
			text := "没有需要修复的项。"
			if len(lines) > 0 {
				text = strings.Join(lines, "；")
			}
			finalText := text
			c.io.Dispatch(func() {
				c.fixing = false
				c.Push(map[string]interface{}{"text": finalText, "kind": "ok"})
			})
		})
	case "check-update":
		if c.checking {
			return
		}
		c.checking = true
		c.Push()
		c.io.RunAsync(func() { c.checkUpdate(true) })
	case "open-url":
		url, _ := data["url"].(string)
		if !isAllowedURL(url) {
			c.Push(map[string]interface{}{"text": "不允许打开这个链接。", "kind": "error"})
			return
		}
		if err := c.io.OpenURL(url); err != nil {
			c.Push(map[string]interface{}{"text": "打开浏览器失败，请手动访问：" + url, "kind": "error"})
		}
	}
}

func (c *Controller) ApplyUse(tool models.Tool, home string, fromTray bool) {
	state := store.Load()
	if current := state.CurrentRuntime(tool); current != nil &&
		strings.TrimRight(current.Home, "/") == strings.TrimRight(home, "/") {
		if fromTray {
			c.io.Notify("DevSwitch", "该版本已经激活")
		} else {
			c.Push(map[string]interface{}{"text": "该版本已经激活", "kind": "ok"})
		}
		return
	}
	runtime, err := service.Use(tool, home)
	if err != nil {
		if fromTray {
			c.io.Notify("切换失败", err.Error())
		} else {
			c.Push(map[string]interface{}{"text": err.Error(), "kind": "error"})
		}
		return
	}
	c.Page = string(tool)
	c.Selected = runtime.Home
	name := models.ToolLabels[tool]
	if fromTray {
		c.io.Notify("已切换", fmt.Sprintf("%s → %s", name, runtime.Version))
		if c.ready && c.io.IsVisible() {
			c.Push()
		}
	} else {
		c.Push(map[string]interface{}{
			"text": fmt.Sprintf("已切换到 %s %s。", name, runtime.Version), "kind": "ok",
		})
	}
	c.io.RefreshTray()
}

// StartInstall 是 GUI 版下载器入口：后台执行，完成回 UI 线程刷新。
func (c *Controller) StartInstall(version, mirror string) {
	tool := models.Tool(c.Page)
	if c.Installing {
		c.Push(map[string]interface{}{"text": "已有下载任务进行中，请稍候。", "kind": "error"})
		return
	}
	version = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(version), "v"))
	major := strings.Split(version, ".")[0]
	if version == "" || !isDigits(major) {
		c.Push(map[string]interface{}{"text": "请输入大版本号，例如 22 或 17。", "kind": "error"})
		return
	}
	c.Installing = true
	c.Selected = ""
	c.Push(map[string]interface{}{
		"text": fmt.Sprintf("开始下载 %s %s（视网络可能需要几分钟）……",
			models.ToolLabels[tool], version),
		"kind": "ok",
	})

	c.io.RunAsync(func() {
		fullVersion := ""
		var installErr error
		defer func() {
			if r := recover(); r != nil {
				installErr = fmt.Errorf("%v", r)
			}
			c.io.Dispatch(func() {
				c.Installing = false
				if installErr != nil {
					c.Push(map[string]interface{}{"text": "下载失败：" + installErr.Error(), "kind": "error"})
				} else {
					c.Push(map[string]interface{}{
						"text": fmt.Sprintf("已安装并切换 %s %s。", models.ToolLabels[tool], fullVersion),
						"kind": "ok",
					})
				}
				c.io.RefreshTray()
			})
		}()
		_, v, err := service.InstallRuntime(tool, version, mirror)
		fullVersion = v
		installErr = err
	})
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (c *Controller) doImport() {
	tool := models.Tool(c.Page)
	title := fmt.Sprintf("选择 %s 安装目录", models.ToolLabels[tool])
	chosen, err := c.io.ChooseFolder(title)
	if err != nil || chosen == "" {
		return
	}
	runtime, importErr := service.ImportPath(tool, chosen)
	if importErr != nil {
		c.Push(map[string]interface{}{"text": importErr.Error(), "kind": "error"})
		return
	}
	c.Page = string(tool)
	c.Selected = runtime.Home
	c.Push(map[string]interface{}{"text": fmt.Sprintf("已导入 %s。", runtime.Version), "kind": "ok"})
	c.io.RefreshTray()
}

// TrayMenuItems 返回托盘菜单数据（标题, 版本, home），供宿主渲染。
func TrayMenuItems() [][3]string {
	state := store.Load()
	items := [][3]string{}
	for _, tool := range models.Tools {
		for _, r := range state.ForTool(tool) {
			items = append(items, [3]string{string(tool), r.Version, r.Home})
		}
	}
	return items
}

// CurrentHome 返回工具当前 home（托盘打点用）。
func CurrentHome(tool models.Tool) string {
	state := store.Load()
	if r := state.CurrentRuntime(tool); r != nil {
		return r.Home
	}
	return ""
}

// ReleasePageURL 是更新提示指向的发布页。
const ReleasePageURL = "https://github.com/xyztony999/devswitch/releases/latest"

// checkUpdate 查询 GitHub 最新发布并推回 UI。
// manual=true 时结果带 flash 文案；静默检查只在发现新版本时提示。
func (c *Controller) checkUpdate(manual bool) {
	latest, err := downloader.LatestReleaseVersion()
	c.io.Dispatch(func() {
		c.checking = false
		if err != nil {
			if manual {
				c.Push(map[string]interface{}{"text": "检查更新失败：" + err.Error(), "kind": "error"})
			}
			return
		}
		c.latest = latest
		if latest == models.AppVersion {
			if manual {
				c.Push(map[string]interface{}{
					"text": fmt.Sprintf("已是最新版本（%s）。", latest), "kind": "ok",
				})
			}
			return
		}
		c.Push(map[string]interface{}{
			"text": fmt.Sprintf("发现新版本：%s（当前 %s）。", latest, models.AppVersion),
			"kind": "ok",
		})
	})
}
