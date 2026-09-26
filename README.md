# DevSwitch

跨 **Windows / Linux** 的 Node / npm / Java / Maven / Gradle 版本管理器。命令行和图形界面都能用，切换发生在当前用户目录，不需要管理员 / sudo，也不去动系统级的版本管理（`update-alternatives`、机器级安装等）。

DevSwitch 会扫描本机已有的运行时（或直接下载安装），收拢到一张列表里，点一下（或一条命令）完成切换。

> **2.0 起为 Go 单文件实现**：自包含二进制，不再依赖 Python；1.x（Python）的命令与数据格式完全兼容，覆盖安装即可升级，版本选择记录（state.json）原地保留。

## 它解决什么问题

多套运行时共存时的常见麻烦：

- `PATH` 里同时出现解压版 Node、包管理器装的 Node、以及系统自带的版本
- 多套 JDK 切换要管理员权限，图形程序和终端还不一定一致
- 只能改 shell 配置文件，改完还得重开终端

DevSwitch 的做法：

1. **扫描**已有安装（系统 JDK、各厂商发行版、解压的 Node 压缩包、nvm-windows / scoop / sdkman / n 等、手动导入的目录）
2. 往用户目录写入 `node` / `npm` / `npx` / `java` / `javac` 等 **shim**（Windows 下同时生成 `.cmd` 与 sh 双份，PowerShell / cmd / Git Bash 各取所需）
3. shim 每次执行时定位当前选中的版本，所以**已打开终端里的 node/java 会马上变成新版本**
4. 同时维护 `JAVA_HOME`（Linux：`~/.bashrc` / `~/.profile` 钩子；Windows：用户环境变量 + PowerShell profile + Git Bash 钩子）

## 依赖

无。Windows 图形界面需要 WebView2 运行时（Win10/11 一般已带）；Linux 图形界面需要系统 GTK3 + WebKitGTK（主流发行版仓库都有，deb / rpm 已声明为弱依赖，缺失时命令行不受影响）。

## 安装

安装到当前用户目录，**不需要管理员**。

**Windows**：从 [Releases](https://github.com/xyztony999/devswitch/releases) 下载安装程序并双击（Intel / AMD 用 `-x64.exe`，骁龙等 ARM 笔记本用 `-arm64.exe`），或者下载便携版 zip 解压即用。

- 标准安装向导，全程**用户级、无需管理员**
- 自动注册到「设置 → 应用 / 控制面板 → 程序和功能」，可从那里**一键卸载**（自带独立卸载程序）
- 卸载会清理程序、shim、用户 PATH / JAVA_HOME 与 shell 钩子，仅保留你的版本选择记录（state.json）
- 开始菜单快捷方式 + 可选桌面快捷方式；`devswitch` 与 `devswitch-gui` 命令同时可用

**Linux**：从 [Releases](https://github.com/xyztony999/devswitch/releases) 下载对应格式的包。不确定架构时运行 `uname -m`：显示 `x86_64` 选 amd64/x86_64 包，显示 `aarch64` 选 arm64/aarch64 包：

```bash
# Debian 系（Ubuntu / Debian / Deepin / openKylin / UOS 等）
sudo apt install ./devswitch_2.0.0_amd64.deb     # arm64 机器换成 _arm64.deb

# Fedora / RHEL 系
sudo dnf install ./devswitch-2.0.0-1.x86_64.rpm  # arm64 机器换成 _aarch64.rpm
```

包内含两个二进制：`/usr/bin/devswitch`（静态 CLI，零系统依赖）与 `/usr/bin/devswitch-gui`（图形界面）。**首次运行 `devswitch scan` 或图形界面时自动完成用户级初始化**（扫描运行时、写 shim、shell 钩子），无需任何 root 配置步骤。非 deb / rpm 发行版可用便携版 tar.gz。

装完后：Windows 开始菜单 / 应用列表搜索 **DevSwitch**，或任意终端运行 `devswitch`（无参数直接进图形界面）；Linux 运行 `devswitch-gui` 或 `devswitch gui`。升级：重新下载安装包覆盖安装（Linux 重新装包）。卸载：控制面板 / `devswitch cleanup`（清理用户级配置，运行时数据保留）。

## 命令行

```bash
devswitch scan              # 扫描本机已有版本
devswitch install node 22   # 下载安装指定大版本并切换（--mirror npmmirror|tuna 走国内镜像）
devswitch install java 17
devswitch list              # 查看列表，* 为当前
devswitch current
devswitch use node 22       # 也支持完整版本号或安装目录
devswitch use maven 3       # Maven / Gradle 同一套用法
devswitch export            # 导出当前版本选择到 .devswitch（可提交进仓库，团队对齐用）
devswitch apply .devswitch  # 按文件对齐版本（--install-missing 自动下载缺少的）
devswitch import java /usr/lib/jvm/java-17-openjdk
devswitch which node
devswitch which MAVEN_HOME
devswitch doctor            # 检查 PATH / 冲突的 shell 配置
devswitch doctor --fix      # 写入钩子，并注释手写 Node PATH
devswitch cleanup           # 清理用户级配置（shim / PATH / 环境变量 / 钩子）
devswitch update            # 检查新版本（只读，不自动升级）
devswitch completion bash   # shell 补全（bash / zsh / powershell）
```

## 数据放哪

| 内容 | Linux | Windows |
|---|---|---|
| 程序 | `/usr/bin`（系统包） | `%LOCALAPPDATA%\devswitch\app` |
| shim | `~/.local/bin` | `%LOCALAPPDATA%\devswitch\bin` |
| 状态 / env | `~/.config/devswitch` | `%APPDATA%\devswitch` |
| 备份 | `~/.config/devswitch/backup/` | `%APPDATA%\devswitch\backup/` |
| 下载的运行时 | `~/devswitch-runtimes` | `%USERPROFILE%\devswitch-runtimes` |

## 注意

- **Windows 的一个机制限制**：新终端的 PATH 是「系统 PATH 在前 + 用户 PATH 在后」。机器级（MSI）安装的 Node / Java 仍可能排在用户级 shim 前面；DevSwitch 的 PowerShell / Git Bash 钩子会把 shim 目录前置到会话 PATH 里解决终端场景。`devswitch doctor` 会对这种情况告警。
- Windows 上全局 npm 包默认装在共享的 `%APPDATA%\npm`，各 Node 版本共用（与 Linux「跟随所选版本」不同）；这些命令会经由 PATH 里的 `node` shim 使用当前选中的版本。
- 不覆盖系统 Java 的 `update-alternatives`（Linux）。没有 sudo 时，系统级 `/usr/bin/java` 仍可能指向另一套 JDK；只要 shim 目录在 PATH 前面，用的就是 DevSwitch 选中的版本。
- `devswitch doctor --fix` 会注释 `~/.bashrc`（Linux）/ Git Bash 配置里手写的 Node `PATH` 行（先备份）。

## 开发

本仓库是 Go 单模块（`cmd/devswitch` + `internal/`），图形界面是 Vue 3 + Naive UI，构建后内嵌进二进制（`internal/gui/ui/dist`）。

```bash
# CLI（零 CGO，随处可编）
go build ./cmd/devswitch

# GUI（需 CGO：Windows 装 MSYS2 的 mingw-w64-ucrt-x86_64-gcc，Linux 装 libgtk-3-dev
#      + libwebkit2gtk-4.0-dev + libayatana-appindicator3-dev）
go build -tags gui ./cmd/devswitch

# 改前端后重新构建并同步内嵌产物
cd frontend && npm install && npm run build
```

从源码直接试运行：`go run ./cmd/devswitch`（或 `go run -tags gui ./cmd/devswitch gui`）。1.x 的 Python 实现仍在仓库中（`devswitch/`，维护模式），golden 测试（`tests/`）以 Python 侧行为为规格基线，Go 侧生成物逐字节对齐。

打包 Windows 安装程序（需 Inno Setup 6）：`ISCC devswitch.iss`（arm64 加 `/DArchMode=arm64 /DBinSource=dist\bin\devswitch-arm64.exe`）。构建 Linux 包：先构建 `dist/bin/devswitch`（静态）与 `dist/bin/devswitch-gui`，再 `python3 tools/build_packages.py --version x.y.z`（需 dpkg-deb / rpmbuild，在本机架构上原生打包）。推送 `v*` 标签后 CI 自动构建全部产物（Windows x64/ARM64 安装器 + 便携 zip、deb/rpm × amd64/arm64、Linux 便携 tar.gz）并发布 Release（标签需与 `internal/models` 的 AppVersion 一致，`-dev` 后缀忽略）。

测试：Go 侧 `go test ./internal/...`（golden 字节对齐）；Python 侧 `python -m pytest tests/`。CI：GitHub Actions 在 ubuntu / windows 上运行构建、测试与 GUI 编译检查。

## 许可

MIT
