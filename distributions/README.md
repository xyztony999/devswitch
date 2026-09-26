# 分发渠道

主分发渠道是 [GitHub Releases](https://github.com/xyztony999/devswitch/releases)
（六个产物：Windows x64/ARM64 安装器 + deb/rpm × amd64/arm64）。
以下渠道的清单由 `python tools/make_manifests.py` 生成，入库需少量人工步骤：

## winget（Windows 包管理器，目标：`winget install devswitch`）

1. 发布新版本后，从 Release 下载两个 exe 并运行：
   `python tools/make_manifests.py --version x.y.z --x64-exe ... --arm64-exe ...`
2. `dist/manifests/winget/` 下生成三份 yaml（installer / locale / version）
3. fork [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)，按其
   `manifests/x/xyztony999/devswitch/<版本>/` 目录规范提交 PR，等待合并

## scoop（Windows）

把 `dist/manifests/scoop/devswitch.json` 放进自建 bucket 仓库（如 `scoop-bucket`），
用户即可：
```powershell
scoop bucket add devswitch https://github.com/xyztony999/scoop-bucket
scoop install devswitch
```

## Homebrew（macOS / Linux）

把 `dist/manifests/homebrew/devswitch.rb` 放进自建 tap 仓库（如 `homebrew-devswitch`）：
```bash
brew tap xyztony999/devswitch https://github.com/xyztony999/homebrew-devswitch
brew install devswitch
```
（formula 从 deb 资产提取；macOS 适配并入主线后可切换为源码安装。）
