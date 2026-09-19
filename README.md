# DevSwitch

跨 **Windows / Linux** 的 Node / npm / Java 版本管理器。命令行和图形界面都能用，切换发生在当前用户目录，不需要管理员 / sudo，也不去动系统级的版本管理（`update-alternatives`、机器级安装等）。

DevSwitch 会扫描本机已有的运行时，收拢到一张列表里，点一下（或一条命令）完成切换。

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

命令行只需 Python 3.8+。

- **Linux**：图形界面还需要 GTK 3、WebKit2 和 PyGObject（主流 Debian 系桌面一般已带）：
  ```bash
  sudo apt install python3 python3-gi gir1.2-gtk-3.0 gir1.2-webkit2-4.0 gir1.2-appindicator3-0.1
  ```
  托盘图标缺 `gir1.2-appindicator3-0.1` 时仍可开主窗口。
- **Windows**：图形界面需要 WebView2 运行时（Win10/11 一般已带）与三个 Python 包，安装脚本会自动尝试：
  ```
  pip install --user pywebview pystray Pillow
  ```
  缺依赖时命令行不受影响。

## 安装

安装到当前用户目录，**不需要管理员**。装完后可以删掉克隆下来的仓库。

**Linux**：从 [Releases](https://github.com/xyztony999/devswitch/releases) 下载对应格式的包。不确定架构时运行 `uname -m`：显示 `x86_64` 选 amd64/x86_64 包，显示 `aarch64` 选 arm64/aarch64 包（包内容为纯 Python，两种架构安装体验一致）：

```bash
# Debian 系（Ubuntu / Debian / Deepin / openKylin / UOS 等）
sudo apt install ./devswitch_1.1.0_amd64.deb     # arm64 机器换成 _arm64.deb

# Fedora / RHEL 系
sudo dnf install ./devswitch-1.1.0-1.x86_64.rpm  # arm64 机器换成 _aarch64.rpm
```

系统包只放置程序与 `/usr/bin/devswitch` 入口；**首次运行 `devswitch` 时自动完成用户级初始化**（扫描运行时、写 shim、shell 钩子），无需任何 root 配置步骤。

也可以克隆仓库用脚本装：`./install.sh`。

**Windows**：从 [Releases](https://github.com/xyztony999/devswitch/releases) 下载 `DevSwitch-Setup-x.x.x-x64.exe`（Intel/AMD 及大多数设备；ARM 设备如骁龙笔记本用 `-arm64` 后缀，或直接运行 x64 包走兼容层），双击安装。前提是已装 [Python 3.8+](https://www.python.org/downloads/)（勾选 py launcher），安装向导会自动检测并引导。

- 标准安装向导，全程**用户级、无需管理员**
- 自动注册到「设置 → 应用 / 控制面板 → 程序和功能」，可从那里**一键卸载**（自带独立卸载程序）
- 开始菜单快捷方式 + 可选桌面快捷方式
- 卸载会清理程序、shim、用户 PATH / JAVA_HOME 与 shell 钩子，仅保留你的版本选择记录（state.json）

命令行等价入口：`python -m devswitch install`（配置）/ `devswitch uninstall`（完整卸载）。

装完后：开始菜单 / 应用列表搜索 **DevSwitch**，或运行 `devswitch-gui`；直接运行 `devswitch` 也会打开图形界面。升级：重新下载安装包覆盖安装（Linux 再拉一次代码重跑 `./install.sh`）。卸载：控制面板 / `devswitch uninstall` / `./uninstall.sh`。

## 命令行

```bash
devswitch scan              # 扫描本机已有版本
devswitch list              # 查看列表，* 为当前
devswitch current
devswitch use node 22       # 也支持完整版本号或安装目录
devswitch use java 17
devswitch import node D:\some-node     # 导入一个本地安装目录
devswitch import java /usr/lib/jvm/java-17-openjdk
devswitch which node
devswitch which JAVA_HOME
devswitch doctor            # 检查 PATH / 冲突的 shell 配置
devswitch doctor --fix      # 写入钩子，并注释手写 Node PATH
devswitch hook install      # （重）安装 shell 钩子
```

## 数据放哪

| 内容 | Linux | Windows |
|---|---|---|
| 程序 | `~/.local/share/devswitch` | `%LOCALAPPDATA%\devswitch\app` |
| shim | `~/.local/bin` | `%LOCALAPPDATA%\devswitch\bin` |
| 状态 / env | `~/.config/devswitch` | `%APPDATA%\devswitch` |
| 备份 | `~/.config/devswitch/backup/` | `%APPDATA%\devswitch\backup\` |

## 注意

- **Windows 的一个机制限制**：新终端的 PATH 是「系统 PATH 在前 + 用户 PATH 在后」。机器级（MSI）安装的 Node / Java 仍可能排在用户级 shim 前面；DevSwitch 的 PowerShell / Git Bash 钩子会把 shim 目录前置到会话 PATH 里解决终端场景。`devswitch doctor` 会对这种情况告警。
- Windows 上全局 npm 包默认装在共享的 `%APPDATA%\npm`，各 Node 版本共用（与 Linux「跟随所选版本」不同）；这些命令会经由 PATH 里的 `node` shim 使用当前选中的版本。
- 不覆盖系统 Java 的 `update-alternatives`（Linux）。没有 sudo 时，系统级 `/usr/bin/java` 仍可能指向另一套 JDK；只要 shim 目录在 PATH 前面，用的就是 DevSwitch 选中的版本。
- `devswitch doctor --fix` 会注释 `~/.bashrc`（Linux）/ Git Bash 配置里手写的 Node `PATH` 行（先备份）。

## 开发

图形界面是 Vue 3 + Naive UI，由 Python 加载 `devswitch/ui/dist`（Linux 走 WebKit2，Windows 走 WebView2，共享同一份界面与消息协议）。改界面后：

```bash
cd frontend
npm install
npm run build
```

开发时可直接运行仓库里的 `bin/devswitch` / `bin/devswitch-gui`（Linux）或 `python -m devswitch`（双平台），不必先安装。

打包 Windows 安装程序（需 Inno Setup 6）：`ISCC devswitch.iss`（ARM64 加 `/DArchMode=arm64`），产物在 `dist\`。构建 Linux 包：`python3 tools/build_packages.py`（需 dpkg-deb / rpmbuild）。推送 `v*` 标签后 CI 会自动构建全部六个产物（Windows x64/ARM64 安装器 + deb/rpm × amd64/arm64）并发布到 Release（标签需与 `__init__.py` 版本一致）。

测试：`python -m pytest tests/`（双平台可跑，Linux 侧有 golden 基线保证生成物不变）。CI：GitHub Actions 在 ubuntu / windows × Python 3.9 / 3.13 矩阵上运行。

## 许可

MIT
