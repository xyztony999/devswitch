# API 设计：androidsdk 纳管

| 属性 | 值 |
|------|-----|
| 版本 | v1.0 |
| 状态 | 草稿（Gate 1 待检） |
| 最后更新 | 2026-10-08 |
| 关联 PRD | `../prd/androidsdk.md`（主数据类，完备性 9/9 通过） |
| 技术规约 | 无激活栈 manifest；按引擎默认结构，接口层语义 = CLI 子命令 + GUI 消息协议（本产品形态） |

## 1. 模块设计概述

### 1.1 业务背景及目标

见 PRD §1：Android SDK（含 platform-tools）作为第五工具接入既有纳管/切换/下载/治理链路。

### 1.2 边界说明

| 边界 | 说明 |
|------|------|
| 做什么 | 检测（AS 目录/环境变量/PATH 回溯）、导入校验、镜像下载、ANDROID_HOME+adb/fastboot 切换、export/apply/doctor 扩展 |
| 不做什么 | 组件级管理（platforms/build-tools 装卸）、sdkmanager 交互、NDK/模拟器、改动既有四工具行为 |
| 模块边界 | 复用 Runtime 聚合与工具注册表；仅在注册表新增 androidsdk 项 + 各模块新增 androidsdk 分支 |

### 1.3 设计决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | tool 标识取 `androidsdk`（非 android/android-sdk） | 与现有单词风格一致；export 文件键可读 |
| D2 | 版本来源 = `platform-tools/source.properties` 的 `Pkg.Revision`；缺失回退目录名/`unknown` | 无子进程约束（Mimosa 防护 + 性能）；source.properties 是 SDK 组件标准元数据 |
| D3 | ANDROID_SDK_ROOT 只读识别不写 | 官方已废弃该变量；写会加剧双变量漂移 |
| D4 | 下载内容 = cmdline-tools（最新）+ platform-tools（latest）两 zip，解压到统一 SDK 根 | sdkmanager 需 cmdline-tools 引导后续组件；platform-tools 满足 adb/fastboot 即用 |
| D5 | 腾讯镜像包名 `platform-tools-latest-{windows,linux}.zip`、`commandlinetools-{win,linux}-*.zip` | 腾讯 AndroidSDK 镜像目录既有命名 |
| D6 | shim 定位脚本指向 `<home>/platform-tools`（Windows：adb.exe/fastboot.exe） | 与 JAVA_HOME→bin 的间接层同构 |

### 1.4 领域对象总览

| 分类 | 对象 | 说明 |
|------|------|------|
| 聚合根 | Runtime{tool: androidsdk} | 复用既有 Runtime；version=platform-tools 版本 |
| 值对象 | platform-tools 版本（Pkg.Revision） | 不可变 |
| 基础数据 | 工具注册表 androidsdk 项 | shims=adb,fastboot；homeVar=ANDROID_HOME；检测器/下载源 |

无子流程、无新增关联领域对象（共享内核：Runtime/tool 枚举）。

### 1.5 上下文映射

| 对方上下文 | 关系 | 模式 |
|-----------|------|------|
| 既有四工具纳管/切换 | 同域扩展（tool 枚举加值） | 共享内核 |
| 环境诊断（S-05） | 诊断规则新增 androidsdk 条目 | 遵奉者（复用 DoctorIssues 框架） |
| 版本选择共享（S-04） | .devswitch 新增键 | 共享内核（文件格式版本化） |

## 3. 应用/基础设施层设计

> 第2章（跨聚合根）裁剪：单聚合根模块，不适用。

### 3.1 接口规范（CLI）

沿用全局 CLI 规范：子命令 + 中文输出；退出码 0/1/2（成功/业务失败/用法错误）。无鉴权（本机单用户工具，权限守卫不适用，标注：无权限模型）。

### 3.2 依赖与约束

- 复用 downloader 的 SSRF 防护（白名单 + 公网 IP + 重定向同检 + 安全解压）
- 复用 winenv 单次广播（ApplyUserEnv）；复用 apply 双写 shim
- **约束**：既有四工具 golden 生成物字节不变（回归红线）

### 3.3 公共规范（枚举/错误码/配置）

| 枚举/常量 | 值 |
|-----------|-----|
| Tool | `node / java / maven / gradle / androidsdk`（新增第五值） |
| source | `system / env / path / imported / downloaded`（androidsdk 新增 env、path 两值，其余工具不产生） |
| 下载镜像 | `official / npmmirror / tuna / tencent`（tencent 仅 androidsdk 有效） |
| .devswitch 键 | `androidsdk: <platform-tools 版本>`（文件 `# devswitch versions` 块追加，旧文件缺键容忍） |
| 错误码（CLI 文案） | `目录里没有可用的 Android SDK：<path>`（导入校验失败）；`未找到 androidsdk <ver>`（use 失败，同现有） |

### 3.4 GUI 消息协议（op 层）

`page=androidsdk`（新页面值）；`install/use/import/scan` 等既有 op 的 page 上下文扩展。State payload 的 `current/runtimes` 天然携带 androidsdk（协议无破坏性变更）。

## 4. 聚合根详细设计（Runtime / androidsdk）

### 4.1 领域模型

**表结构（状态文件扩展，非关系库）**：state.json 的 runtimes[] 增加tool=androidsdk 条目；current 增 `androidsdk` 键。无独立表。

**领域对象及角色**：

| 对象 | 角色 | 字段 |
|------|------|------|
| Runtime{androidsdk} | 聚合根实例 | tool/home/version/major/label/vendor/source/binary |
| platform-tools 版本 | 值对象 | 取 source.properties → Pkg.Revision；回退目录名 |

**字段规范**：
- home = SDK 根目录（Windows 反斜杠，同现有规范）
- version = `35.0.2` 形态；major = 首段（"35"）
- label = `Android SDK (platform-tools 35.0.2)`
- binary = `<home>/platform-tools/adb`（Windows `adb.exe`）

**不变量**：
- INV-1：home 目录必须存在 `platform-tools/`（adb 所在）
- INV-2：同 home 去重键 `androidsdk:<home>`（与既有一致）
- INV-3：current.androidsdk 必须指向 runtimes 中的某个 androidsdk 实例或为空

**检测根（无子进程）**：

| 来源 | 路径 | source 值 |
|------|------|----------|
| Android Studio 默认 | `%LOCALAPPDATA%\Android\Sdk` / `~/Android/Sdk` | system |
| 环境变量 | `ANDROID_HOME` → `ANDROID_SDK_ROOT`（均尝试） | env |
| PATH 回溯 | whichAll(adb) 命中 → 上溯两级 | path |
| 导入/下载 | 用户指定 / 用户目录 | imported / downloaded |

**领域服务**：`sdkVersionFromSourceProps(dir) string`（解析 Pkg.Revision，正则 `(?m)^Pkg.Revision=(.+)$`）。

### 4.2 状态模型

主数据类裁剪：无流程状态机（PRD §6）。数据生命周期：未纳管 → 已纳管 →（当前激活）→ 已清理，由既有 state 语义承载。

### 4.3 子流程设计

无子流程（PRD 附B）。

### 4.4 接口设计

#### 接口清单（CLI，androidsdk 增量行为）

| 命令 | 行为（androidsdk 分支） | 守卫/规则 |
|------|------------------------|----------|
| `scan` | 合并四检测根，去重后入清单 | INV-1/2 |
| `list [androidsdk]` | 列 SDK 实例（version+label+source） | — |
| `current` | 输出 `androidsdk <ver> (<home>)` | — |
| `use androidsdk <ver\|home>` | ANDROID_HOME 写用户变量 + adb/fastboot shim + platform-tools 入会话 PATH；托盘/页面刷新 | BR-3/6 |
| `install androidsdk [ver] [--mirror tencent\|official]` | 下载 cmdline-tools+platform-tools → 解压至 `<data>/runtimes/android-sdk` → 自动 use；ver 参数忽略 latest 之外值时提示 | BR-4/5/8 |
| `import androidsdk <path>` | 校验 platform-tools/adb → 纳管 | BR-2 |
| `which adb` | 输出当前 SDK 的 adb 真实路径 | — |
| `export` | .devswitch 增 `androidsdk:` 键 | §3.4 |
| `apply <file> [--install-missing]` | 对齐 androidsdk；缺则 install | — |
| `doctor [--fix]` | 增查：ANDROID_HOME 一致性（warn，码 `androidsdk-home-registry`）、adb/fastboot shim 存在（warn，码 `no-shim`）、platform-tools 缺失（warn，码 `no-pt`） | §3.3 |

#### 接口详情（关键两条）

**use androidsdk**：
- 生效物：①shim `adb`/`fastboot`（cmd: `"%ANDROID_PT%\adb.exe" %*` 形态经 current.env 解析，同现有间接层；sh: exec "$SDK_HOME/platform-tools/adb"）②HKCU `ANDROID_HOME`（REG_SZ/EXPAND_SZ 按含 % 判定，单次广播）③会话 PATH 注入 `<home>\platform-tools`（钩子 env.sh 追加）
- 校验：目标实例存在（version 或 home 精确/首段匹配，复用 Find）
- 幂等：重复切换同 home 直接返回"该版本已经激活"

**install androidsdk**：
- 流程：解析镜像 URL（tencent/official）→ 下载 zip（白名单校验）→ 安全解压合并至 SDK 根（cmdline-tools/ 与 platform-tools/）→ chmod 0755（Linux）→ 构造 Runtime(source=downloaded) → service.Use
- 失败：任一步失败清理半成品目录，返回错误文案

#### GUI op 增量

| op | page=androidsdk 语义 |
|----|---------------------|
| page | 左导航新增第五项（Android 官方绿 #3DDC84 机器人头简化图形，simple-icons `android`，CC0） |
| install | 下载条镜像选项含"腾讯源"；占位符"如 platform-tools 最新"（输入可空=latest） |
| use/import/scan/doctor | 与四工具同构 |

### 4.5 领域事件与集成

无 MQ/事件（单机工具）。集成点：winenv（环境变量）、托盘（菜单第五组）、更新检查（不涉及）。

## 5. 模块清单汇总

### 5.1 对外服务清单

即 §4.4 CLI 命令表（10 条增量行为）+ GUI op 表。无跨系统服务。

### 5.2 SQL 清单 / DDL

不适用（无关系库）。状态文件结构变更：state.json 无 schema 变化（tool 枚举扩值天然兼容）；.devswitch 文件追加键。

## 6. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|------|------|---------|------|
| v1.0 | 2026-10-08 | initial | androidsdk 纳管首版设计（检测×4 来源/下载×2 镜像/切换 ANDROID_HOME+adb shim/治理扩展） |
