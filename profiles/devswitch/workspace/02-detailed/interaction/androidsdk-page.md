# 交互设计：Android SDK 页（PageSpec）

| 属性 | 值 |
|------|-----|
| 版本 | v1.0 |
| 状态 | 草稿（Gate 1 待检） |
| 最后更新 | 2026-10-08 |
| 信息来源 | `../api/api-androidsdk.md`（唯一来源） |

## 1. 页面结构推断

依据 API §3.4/§4.4：androidsdk 复用既有工具页骨架（导航项 + hero + 下载条 + 版本卡片网格），无新页面类型。

### 1.1 导航变更

| 位置 | 变更 |
|------|------|
| 左侧导航「运行时」组 | 第五项：`Android SDK`（排序：Node.js / Java / Maven / Gradle / **Android SDK**） |
| 顶栏当前激活 chips | 第五枚 chip：`androidsdk <platform-tools 版本>`（未选 `—`） |
| 图标 | BrandLogo kind=androidsdk：Android 官方绿 #3DDC84 机印简化头形（simple-icons `android`，CC0；深底可直接用官方绿，无需提亮） |
| nav-item-sub | 当前版本号（如 `35.0.2`）或 `未选择` |

### 1.2 页面骨架（page=androidsdk）

```
┌ tool-hero ──────────────────────────────────────────┐
│ [logo-xl]  Android SDK   [当前 35.0.2 徽章]          │
│            副标题：adb / fastboot · ANDROID_HOME…   [导入目录]
├ dl-card ────────────────────────────────────────────┤
│ 下载 Android SDK   [镜像▾ 官方源/腾讯源]  [下载并切换] │
├ ver-grid ───────────────────────────────────────────┤
│ ┌ ver-card ─────────┐  ┌ ver-card ─────────┐        │
│ │ 35.0.2  ● 当前     │  │ 34.0.5           │        │
│ │ system · <home>    │  │ imported · <home> │        │
│ │ <home>\…\adb.exe   │  │                   │        │
│ │ [已激活]           │  │ [切换到此版本]      │        │
│ └───────────────────┘  └───────────────────┘        │
└ 说明行：切换后当前终端的 adb / fastboot 立即生效 ────┘
```

## 2. PageSpec（字段级）

### 2.1 tool-hero

| 元素 | 规格 |
|------|------|
| 标题 | `Android SDK` |
| 激活徽章 | 有 current 时显示 `Android SDK (platform-tools <ver>)`，绿色 hero-current-tag |
| 副标题（TOOL_BLURBS） | `Android 开发套件 · adb / fastboot 即时生效，同步维护 ANDROID_HOME` |
| 右侧按钮 | `导入目录`（secondary，触发 op:import，复用 SHBrowseForFolder/zenity 对话框，标题"选择 Android SDK 安装目录"） |

### 2.2 dl-card（下载条）

| 元素 | 规格 |
|------|------|
| 标签 | `下载 Android SDK` |
| 版本输入 | **隐藏**（区别于四工具：仅 latest，无大版本概念；不展示输入框） |
| 镜像选择 | NSelect：`官方源(official)` / `腾讯源(tencent)`，默认 official |
| 按钮 | `下载并切换`（loading=store.installing；点击 `send({op:'install', version:'', mirror})`） |
| 下载中提示 | 复用 NSpin 行：`正在下载 cmdline-tools 与 platform-tools，视网络可能需要几分钟…` |

### 2.3 ver-card（版本卡片）

| 元素 | 规格 |
|------|------|
| 头部 | BrandLogo(lg) + 标题 `label`（如 `Android SDK (platform-tools 35.0.2)`）+ 激活时 `● 当前` |
| meta 行 1 | `版本 <code>35.0.2</code> · <source 文案>`；source 文案映射：system→Android Studio 安装 / env→ANDROID_HOME 指向 / path→PATH 发现 / imported→手动导入 / downloaded→本工具下载 |
| meta 行 2 | `<home>`（code 样式，任意换行） |
| meta 行 3 | `<binary>`（adb 完整路径，code） |
| 动作 | 未激活：`[切换到此版本]`（primary，op:use）；激活：`[已激活]`（disabled） |
| 卡片状态 | 激活卡 active 样式（绿边+浅底） |

### 2.4 空态

| 元素 | 规格 |
|------|------|
| 文案 | `本机没有发现 Android SDK` |
| 引导按钮 | `[扫描本机]` `[导入目录]` |

### 2.5 说明行

`当前终端的 adb / fastboot 已指向激活版本；切换后立即生效，无需重开终端。`

## 3. 全局联动

| 场景 | 行为 |
|------|------|
| 顶栏 chip 点击 androidsdk | setPage('androidsdk') |
| 托盘菜单 | 第五组 `Android SDK`（子项=各实例，勾选当前；MENU_CAP=20 同现有） |
| 诊断页 | 新增 issue 卡片渲染（androidsdk-home-registry / no-shim / no-pt），级别着色同现有 |
| export/apply | 无 UI（CLI 行为）；flash 成功文案含 androidsdk |
| 使用中切页 | 与四工具一致（导航常驻） |

## 4. 质量校验

- [x] 所有字段可追溯到 API 文档（label/binary/source/镜像枚举/占位行为）
- [x] 无新交互范式（全部复用既有组件：hero/下载条/卡片/空态/对话框）
- [x] 与四工具页差异显式标注（无版本输入框——仅 latest；导入对话框标题专属）
- [x] 深色主题可读性：#3DDC84 在 --bg 上对比度充分，无需提亮
- [x] 图标来源合规（simple-icons CC0）

## 更新记录

| 版本 | 日期 | 更新说明 |
|------|------|---------|
| v1.0 | 2026-10-08 | 首次生成 |
