package downloader

import "testing"

// 清单解析：真实 repository2-3.xml 的块结构；覆盖 win/linux 选择、
// 包名白名单形态与缺失场景。
func TestCmdlineToolsFromManifest(t *testing.T) {
	manifest := []byte(`<?xml version='1.0' encoding='UTF-8'?>
<sdk-repository>
  <remotePackage path="platform-tools"><archives><archive><complete><url>platform-tools-latest-windows.zip</url></complete></archive></archives></remotePackage>
  <remotePackage path="cmdline-tools;latest"><archives>
    <archive><complete><size>123</size><checksum>aa</checksum><url>commandlinetools-linux-16111833_latest.zip</url></complete></archive>
    <archive><complete><size>123</size><checksum>bb</checksum><url>commandlinetools-win-16111833_latest.zip</url></complete></archive>
  </archives></remotePackage>
</sdk-repository>`)

	if got, err := cmdlineToolsFromManifest(manifest, "windows"); err != nil || got != "commandlinetools-win-16111833_latest.zip" {
		t.Errorf("windows: got=%q err=%v", got, err)
	}
	if got, err := cmdlineToolsFromManifest(manifest, "linux"); err != nil || got != "commandlinetools-linux-16111833_latest.zip" {
		t.Errorf("linux: got=%q err=%v", got, err)
	}
	if _, err := cmdlineToolsFromManifest([]byte("<sdk-repository></sdk-repository>"), "windows"); err == nil {
		t.Error("缺失条目应报错")
	}
	// 包名白名单：伪造清单里的恶意文件名应被拒绝
	bad := []byte(`<remotePackage path="cmdline-tools;latest"><archives><archive><complete>` +
		`<url>commandlinetools-win-1_../../evil.zip</url></complete></archive></archives></remotePackage>`)
	if _, err := cmdlineToolsFromManifest(bad, "windows"); err == nil {
		t.Error("非常规包名应被拒绝")
	}
}
