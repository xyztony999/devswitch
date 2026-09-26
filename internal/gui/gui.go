//go:build gui

// Package gui 是 Go 2.0 的 GUI 宿主：webview（WebView2/WKWebView/GTK WebKit）
// 承载内嵌的 Vue 前端，systray 提供托盘。用 -tags gui 构建。
package gui

import (
	_ "embed"
	"runtime"
	"strings"
	"sync"

	"github.com/getlantern/systray"
	webview "github.com/webview/webview_go"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/service"
	"github.com/xyztony999/devswitch/internal/uicore"
)

//go:embed ui/dist/index.html
var uiBundle string

type host struct {
	mu         sync.Mutex
	controller *uicore.Controller
	window     webview.WebView
	windowMu   sync.Mutex
}

var h = &host{}

// Run 启动 GUI（托盘为主循环，窗口按需创建）。
func Run() int {
	service.MergeScan()
	service.EnsureApplied()
	systray.Run(onTrayReady, onTrayExit)
	return 0
}

func onTrayReady() {
	systray.SetIcon(trayIconICO())
	systray.SetTitle("DevSwitch")
	systray.SetTooltip("DevSwitch — Node/Java/Maven/Gradle 版本切换")
	buildTray()
	openWindow()
}

func onTrayExit() {}

// trayEntries 持有全部版本菜单项：systray 无 ResetMenu，切换后只更新勾选。
var trayEntries = map[string]*systray.MenuItem{}

func buildTray() {
	mOpen := systray.AddMenuItem("打开主窗口", "显示 DevSwitch 主窗口")
	systray.AddSeparator()
	for _, tool := range models.Tools {
		sub := systray.AddMenuItem(models.ToolLabels[tool], models.ToolLabels[tool]+" 版本")
		current := uicore.CurrentHome(tool)
		items := menuItemsFor(tool)
		for _, item := range items {
			version, home := item[0], item[1]
			entry := sub.AddSubMenuItemCheckbox(version, home, home == current)
			trayEntries[string(tool)+"|"+home] = entry
			go func(t models.Tool, hh string, e *systray.MenuItem) {
				for range e.ClickedCh {
					withController(func(c *uicore.Controller) { c.ApplyUse(t, hh, true) })
				}
			}(tool, home, entry)
		}
		if len(items) == 0 {
			disabled := sub.AddSubMenuItem("本机未发现", "")
			disabled.Disable()
		}
	}
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出 DevSwitch")
	go func() {
		for range mOpen.ClickedCh {
			openWindow()
		}
	}()
	go func() {
		for range mQuit.ClickedCh {
			systray.Quit()
		}
	}()
}

// refreshTrayChecks 切换后更新托盘勾选（菜单结构保持不变）。
func refreshTrayChecks() {
	for _, tool := range models.Tools {
		current := uicore.CurrentHome(tool)
		for key, entry := range trayEntries {
			if !strings.HasPrefix(key, string(tool)+"|") {
				continue
			}
			home := strings.TrimPrefix(key, string(tool)+"|")
			if home == current {
				entry.Check()
			} else {
				entry.Uncheck()
			}
		}
	}
}

func menuItemsFor(tool models.Tool) [][2]string {
	items := [][2]string{}
	for _, item := range uicore.TrayMenuItems() {
		if item[0] == string(tool) {
			items = append(items, [2]string{item[1], item[2]})
		}
	}
	if len(items) > 20 { // 与 Python 版 MENU_CAP 一致
		items = items[:20]
	}
	return items
}

func withController(fn func(*uicore.Controller)) {
	h.mu.Lock()
	controller := h.controller
	h.mu.Unlock()
	if controller != nil {
		fn(controller)
	}
}

// openWindow 创建主窗口。WebView 不支持 show/hide 循环：关闭即销毁，
// 可随时从托盘「打开主窗口」重建。
func openWindow() {
	h.windowMu.Lock()
	if h.window != nil {
		h.windowMu.Unlock()
		return
	}
	h.windowMu.Unlock()

	go func() {
		// WebView2 的消息循环必须留在创建窗口的 OS 线程上，
		// 否则 goroutine 被调度器迁移后窗口消息无人泵送（表现为“未响应”）。
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		io := &webviewIO{}
		controller := uicore.NewController(io)
		w := webview.New(false)
		defer w.Destroy()
		io.w = w

		h.windowMu.Lock()
		h.window = w
		h.windowMu.Unlock()
		h.mu.Lock()
		h.controller = controller
		h.mu.Unlock()

		w.SetTitle("DevSwitch")
		w.SetSize(1100, 720, webview.HintNone)
		_ = w.Bind("__nativeSend", func(msg string) {
			controller.HandleMessage(parseMessage(msg))
		})
		w.SetHtml(uicore.InlineHTML(uiBundle))
		// 页面加载完成后推送首帧状态
		w.Dispatch(func() { controller.Ready() })
		w.Run()

		h.windowMu.Lock()
		h.window = nil
		h.windowMu.Unlock()
		h.mu.Lock()
		h.controller = nil
		h.mu.Unlock()
	}()
}
