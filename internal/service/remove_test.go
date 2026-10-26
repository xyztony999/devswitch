package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/paths"
)

// removeTestEnv 沙箱：伪造用户主目录 + state（node 两实例，其一激活）。
func removeTestEnv(t *testing.T) (home string, activeHome string, spareHome string, restore func()) {
	t.Helper()
	tmp := t.TempDir()
	root := filepath.Join(tmp, "home")
	_ = os.MkdirAll(root, 0o755)

	origHome, origConfig, origData, origBin := paths.Home, paths.ConfigDir, paths.DataDir, paths.LocalBin
	paths.Home = func() string { return root }
	paths.ConfigDir = func() string { return filepath.Join(root, ".config", "devswitch") }
	paths.DataDir = func() string { return filepath.Join(root, ".local", "share", "devswitch") }
	paths.LocalBin = func() string { return filepath.Join(root, ".local", "bin") }

	activeHome = filepath.Join(root, "node-v22.1.0")
	spareHome = filepath.Join(root, "node-v20.1.0")
	for _, h := range []string{activeHome, spareHome} {
		_ = os.MkdirAll(filepath.Join(h, "bin"), 0o755)
		_ = os.WriteFile(filepath.Join(h, "bin", "node"), []byte{0}, 0o755)
	}
	_ = os.MkdirAll(paths.ConfigDir(), 0o755)
	return root, activeHome, spareHome, func() {
		paths.Home, paths.ConfigDir, paths.DataDir, paths.LocalBin = origHome, origConfig, origData, origBin
	}
}

func TestRemoveBlocksActive(t *testing.T) {
	_, activeHome, _, restore := removeTestEnv(t)
	defer restore()
	state := models.NewState()
	state.Current[models.Node] = activeHome
	state.Runtimes = []models.Runtime{
		{Tool: models.Node, Version: "22.1.0", Major: "22", Home: activeHome, Source: "local"},
	}
	saveTestState(t, &state)

	if _, _, err := RemoveRuntime(models.Node, "22", false, true); err == nil {
		t.Fatal("激活版本应被阻止移除")
	}
}

func TestRemoveListOnlyKeepsDisk(t *testing.T) {
	_, _, spareHome, restore := removeTestEnv(t)
	defer restore()
	state := models.NewState()
	state.Current[models.Node] = ""
	state.Runtimes = []models.Runtime{
		{Tool: models.Node, Version: "20.1.0", Major: "20", Home: spareHome, Source: "local"},
	}
	saveTestState(t, &state)

	if _, _, err := RemoveRuntime(models.Node, "20", false, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(spareHome); err != nil {
		t.Error("无 --purge 不应动磁盘")
	}
	after := loadTestState(t)
	if len(after.Runtimes) != 0 {
		t.Errorf("清单未移除：%d 条", len(after.Runtimes))
	}
}

func TestRemovePurgeDeletesUserDirOnly(t *testing.T) {
	root, _, spareHome, restore := removeTestEnv(t)
	defer restore()
	state := models.NewState()
	state.Runtimes = []models.Runtime{
		{Tool: models.Node, Version: "20.1.0", Major: "20", Home: spareHome, Source: "downloaded"},
	}
	saveTestState(t, &state)

	if _, _, err := RemoveRuntime(models.Node, spareHome, true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(spareHome); !os.IsNotExist(err) {
		t.Error("purge 应删除用户目录内实例")
	}
	_ = root
}

func TestRemovePurgeForeignDegrades(t *testing.T) {
	_, _, spareHome, restore := removeTestEnv(t)
	defer restore()
	state := models.NewState()
	state.Runtimes = []models.Runtime{
		{Tool: models.Node, Version: "20.1.0", Major: "20", Home: spareHome, Source: "system"},
	}
	saveTestState(t, &state)

	_, degraded, err := RemoveRuntime(models.Node, "20", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !degraded {
		t.Error("外来安装 --purge 应降级")
	}
	if _, err := os.Stat(spareHome); err != nil {
		t.Error("外来安装磁盘不应被动")
	}
}

func TestRemovePurgeOutsideHomeRefused(t *testing.T) {
	_, _, _, restore := removeTestEnv(t)
	defer restore()
	outside := filepath.Join(filepath.VolumeName(paths.Home()), string(filepath.Separator), "definitely", "not", "home")
	state := models.NewState()
	state.Runtimes = []models.Runtime{
		{Tool: models.Node, Version: "20.1.0", Major: "20", Home: outside, Source: "downloaded"},
	}
	saveTestState(t, &state)

	_, degraded, err := RemoveRuntime(models.Node, outside, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !degraded {
		t.Error("用户主目录外的 --purge 应降级为仅移清单")
	}
}

func TestRemoveUnconfirmed(t *testing.T) {
	_, _, spareHome, restore := removeTestEnv(t)
	defer restore()
	state := models.NewState()
	state.Runtimes = []models.Runtime{
		{Tool: models.Node, Version: "20.1.0", Major: "20", Home: spareHome, Source: "downloaded"},
	}
	saveTestState(t, &state)

	if _, _, err := RemoveRuntime(models.Node, "20", true, false); err == nil {
		t.Fatal("purge 未确认应报错且 state 不变")
	}
	after := loadTestState(t)
	if len(after.Runtimes) != 1 {
		t.Error("未确认时 state 不应被修改")
	}
}
