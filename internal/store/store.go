// Package store 负责 state.json 的读写，格式与 Python 版逐字段兼容
// （json.dumps ensure_ascii=False, indent=2 + 换行）。
package store

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
)

// pythonState 镜像 Python 侧 JSON 字段名；Tool 作为 map key 会由
// encoding/json 按字符串处理，键名即 "node"/"java"/…
type pythonState struct {
	Schema   int                       `json:"schema"`
	Current  map[string]string         `json:"current"`
	Runtimes []models.Runtime          `json:"runtimes"`
}

func Load() models.State {
	state := models.NewState()
	data, err := os.ReadFile(paths.StateFile())
	if err != nil {
		return state
	}
	var raw pythonState
	if err := json.Unmarshal(data, &raw); err != nil {
		return state
	}
	if raw.Schema > 0 {
		state.Schema = raw.Schema
	}
	for k, v := range raw.Current {
		state.Current[models.Tool(k)] = v
	}
	state.Runtimes = raw.Runtimes
	return state
}

func Save(state models.State) error {
	if err := os.MkdirAll(paths.ConfigDir(), 0o755); err != nil {
		return err
	}
	raw := pythonState{
		Schema:   state.Schema,
		Current:  make(map[string]string, len(state.Current)),
		Runtimes: state.Runtimes,
	}
	if raw.Runtimes == nil {
		raw.Runtimes = []models.Runtime{}
	}
	for tool, home := range state.Current {
		raw.Current[string(tool)] = home
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(raw); err != nil {
		return err
	}
	return os.WriteFile(paths.StateFile(), buf.Bytes(), 0o644)
}
