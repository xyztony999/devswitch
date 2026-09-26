//go:build gui

package gui

import (
	"encoding/json"

	webview "github.com/webview/webview_go"

	"github.com/xyztony999/devswitch/internal/uicore"
)

// webviewIO 实现 uicore.IO：推送到前端走主循环 Dispatch，
// 耗时操作（下载）在 goroutine，完成后回 Dispatch。
type webviewIO struct {
	w webview.WebView
}

func (io *webviewIO) Push(payload map[string]interface{}) {
	if io.w == nil {
		return
	}
	script := uicore.StateScript(payload)
	io.w.Dispatch(func() { io.w.Eval(script) })
}

func (io *webviewIO) RefreshTray() { refreshTrayChecks() }

// Notify：systray 1.2 无原生 toast；托盘场景窗口通常可见，
// 状态经前端 flash 呈现，这里同步更新托盘 tooltip 兜底。
func (io *webviewIO) Notify(title, body string) {
	setTooltipSafe(title + " — " + body)
}

// ChooseFolder：系统文件夹选择对话框（Windows IFileOpenDialog / Linux zenity）。
// 取消时返回空串（静默），失败返回带指引的错误。
func (io *webviewIO) ChooseFolder(title string) (string, error) {
	return chooseFolderNative(title)
}

func (io *webviewIO) IsVisible() bool { return io.w != nil }

func (io *webviewIO) RunAsync(fn func()) { go fn() }

// OpenURL 打开白名单内的项目页面（系统默认浏览器）。
func (io *webviewIO) OpenURL(url string) error { return openURL(url) }

func (io *webviewIO) Dispatch(fn func()) {
	if io.w != nil {
		io.w.Dispatch(fn)
		return
	}
	fn()
}

func parseMessage(msg string) map[string]interface{} {
	data := map[string]interface{}{}
	_ = json.Unmarshal([]byte(msg), &data)
	return data
}
