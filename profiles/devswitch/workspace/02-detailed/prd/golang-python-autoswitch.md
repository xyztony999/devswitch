# 设计文档：Go 工具链 / Python / 项目级自动切换（golang · python · autoswitch）

| 属性 | 值 |
|------|-----|
| 版本 | v1.0（合并式：PRD + API + 交互，三模块一批 v2.3.0） |
| 状态 | Gate 1 待检 |
| 最后更新 | 2026-10-27 |
| D0 决策 | 三项全做；Go 源 golang.google.cn；Python 走 pbs+npmmirror；自动切换会话级 |

## 模块一：golang（Go 工具链）

### PRD 要点（完备性自检 9/9，主数据类）

- **做什么**：检测（GOROOT/PATH 回溯/常见位）+ 下载（golang.google.cn 清单）+ 切换（GOROOT 用户变量 + go/gofmt shim）+ 治理接入
- **不做什么**：不管理 GOPATH/GOPROXY/GOTOOLCHAIN（用户自管）；不装 msi
- 字段：version=VERSION 文件（"go1.25.2"→"1.25.2"）、binary=`<home>/bin/go(.exe)`、label=`Go 1.25.2`
- 规则：下载目录 `~/go-<version>`（**不得装成 ~/go**——与 GOPATH 默认值冲突）；清单含 sha256 校验

### API 契约

| 接口 | 行为 |
|------|------|
| scan | GOROOT → whichAll("go") 回溯两级 → Program Files\Go、~/go-*、scoop |
| install golang <ver> | GET `golang.google.cn/dl/?mode=json&include=all` → major 匹配最新 stable → `<file>.zip/.tar.gz` → 解压 staging → rename `~/go-<ver>` → 写 .devswitch-version → use |
| use golang <ver> | GOROOT 用户变量 + shim + env.sh 段（`<home>/bin` 布局并入 MAVEN/GRADLE 循环组） |
| which go/gofmt | `<home>/bin/go(.exe)` |
| doctor | GOROOT 一致性 + go shim（自动） |

白名单新增：golang.google.cn、go.dev。镜像：official=golang.google.cn（国内快）；npmmirror 不维护 Go。

## 模块二：python（python-build-standalone）

### PRD 要点（9/9，主数据类）

- **不设 PYTHONHOME**（官方不推荐，坑多）；PATH 语义由 shim 承载
- shim：python / python3 / pip；**pip 的 POSIX shim 特判**为 `exec "$.../python3" -m pip "$@"`（pbs 的 pip 脚本 shebang 指向构建机绝对路径，不可用）；Windows 用 pip.exe 直 exec
- 版本：inspector 优先读 `<home>/.devswitch-version`（InstallRuntime 通用机制：下载后写入完整版本）；检测实例只有 major.minor（目录/注册表推导）——`use python 3.12` 可用，`use python 3.12.14` 仅对下载实例可用（文档注明）
- 布局：pbs install_only——Windows 根目录 `python.exe`/`pip.exe`；Linux `bin/python3`+`lib/`

### API 契约

| 接口 | 行为 |
|------|------|
| scan | whichAll("python") 回溯（Windows 回溯两级 `Python3xx/` 根；Linux 真实安装少见，PATH 命中即可）→ `%LOCALAPPDATA%\Programs\Python\Python3xx*` → 注册表卸载表 |
| install python <ver> | npmmirror 根 JSON（日期目录列表）→ 最新日期目录 JSON → `cpython-<ver>+<date>-<plat>-install_only.tar.gz`（win=x86_64-pc-windows-msvc / linux=x86_64-unknown-linux-gnu）→ 解压 staging → rename `~/python-<ver>` → .devswitch-version → use |
| use python <ver> | shim 三件套 + env.sh 赋值行（无 HOME 导出段）；PATH 前置不加（shim 覆盖） |
| which python/pip | `<home>/python.exe`、`<home>/Scripts/pip.exe`（官方布局）/`<home>/pip.exe`（pbs） |

下载 URL：`https://registry.npmmirror.com/-/binary/python-build-standalone/<date>/<file>`（白名单已有 registry.npmmirror.com）。

## 模块三：autoswitch（项目级自动切换）

### PRD 要点（9/9；无新聚合——编排用例）

- **会话级**（direnv 语义）：仅改当前 shell 的环境变量与 PATH，**绝不写 state.json**
- 标记文件 v1：`.devswitch`（自家格式）与 `.nvmrc`（node）；从 CWD 向上最多 3 层找
- 零开销路径：无标记文件时输出空（钩子 eval 空串）；同目录重复触发时输出空（比对 `DEVSWITCH_AUTO_DIR` 环境变量）
- PowerShell/cmd 不做（v1 仅 bash/Git Bash，README 注明）
- **golden 红线**：自动切换钩子为**独立块**（`# DevSwitch auto-switch begin/end`），原 HookSnippet 与 golden 完全不变

### API 契约

| 接口 | 行为 |
|------|------|
| `devswitch env-project [dir]` | dir 缺省 CWD；向上找标记 → 解析所需工具/版本 → Find 实例 → 输出 POSIX 片段：`DEVSWITCH_X_HOME` 赋值 + `_devswitch_front` PATH 前置 + `export`（复用 env.sh 的运行时语义）；缺实例输出 stderr 提示 `devswitch: 项目要求 node 18，未安装（devswitch install node 18）`；无标记输出空 |
| 钩子块 | InstallHook 附写第二块：定义 `_devswitch_autoswitch()`（`out=$(devswitch env-project 2>/dev/null) && eval "$out"`）+ 交互 shell 时挂 PROMPT_COMMAND；RemoveHook 清两块；HookInstalled 查任一 |

## 交互设计（GUI 增量）

- golang/python 成为第六/七个导航工具页（BrandLogo：Go 官方蓝 #00ADD8 字 G 图形、Python 官方蓝黄 #3776AB/#FFD43B 双蛇简化——均 simple-icons CC0）
- 下载条：golang 镜像选项 官方源(golang.google.cn)；python 镜像 npmmirror；版本输入可空=最新
- autoswitch 无 GUI（shell 机制），关于页/README 说明

## Gate 1 自检

- PRD 完备性：三模块 9/9（golang/python 主数据类；autoswitch 无聚合编排用例）
- 关键技术前提已实测：golang.google.cn JSON（369 版本含 sha256）✓；npmmirror pbs 两级 JSON ✓
- 契约对照齐（各接口行为/白名单/布局/rename 规则）；枚举扩展：TOOL 增 golang/python、MIRROR 语义复用（golang=official 单源）
- 风险点已识别：~/go 目录冲突（rename 规避）、pip shebang（-m pip 规避）、golden 红线（独立钩子块规避）

**判定：PASS**
