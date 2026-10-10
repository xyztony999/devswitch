# PRD：Android SDK 纳管（androidsdk）

| 属性 | 值 |
|------|-----|
| 版本 | v1.0 |
| 状态 | 草稿（待评审） |
| 最后更新 | 2026-10-08 |
| 模块 | androidsdk（S-01/S-02/S-03/C-01 扩展 + D-01 注册项） |

## 1. 业务目标

安卓开发者的本机 Android SDK（含 platform-tools 的 adb/fastboot）纳入 DevSwitch 统一管理：扫描收拢、ANDROID_HOME 切换、shim 接管、镜像下载，与现有四工具同体验。**为谁**：使用 DevSwitch 且从事安卓开发/真机调试的本机用户。

## 2. 功能范围

**做什么**：
- 扫描检测：Android Studio 默认目录（Windows `%LOCALAPPDATA%\Android\Sdk`、Linux `~/Android/Sdk`）、`ANDROID_HOME`/`ANDROID_SDK_ROOT` 指向、PATH 中 adb 回溯定位 SDK；识别 platform-tools 版本
- 手动导入：指定 SDK 根目录，校验 `platform-tools/`（含 adb）后纳管
- 下载安装：`install androidsdk`（官方 dl.google.com / 腾讯镜像）下载 cmdline-tools + platform-tools 解压至用户目录并切换
- 切换：`use androidsdk <ver|home>` → 写 ANDROID_HOME 用户变量 + adb/fastboot shim + platform-tools 进会话 PATH
- 治理：export/apply 记录 androidsdk 选择；doctor 增查 ANDROID_HOME 一致性与 adb shim

**不做什么**：
- 不做组件级管理（platforms/build-tools/system-images 的安装卸载与切换，后续版本另立 PRD）
- 不内置 sdkmanager 交互（licenses 确认等仍由用户经 cmdline-tools 自行处理）
- 不管理 Android Studio 本体、NDK、模拟器镜像
- 不修改既有四工具任何行为（生成物字节级不变，golden 基线保持）

## 3. 角色定义

| 角色 | 职责 |
|------|------|
| 开发者 | 本机唯一用户，执行扫描/导入/下载/切换/诊断 |

## 4. 领域对象与关系

| 对象 | 说明 | 关系 |
|------|------|------|
| SDK 实例（Runtime{tool=androidsdk}） | 一个 SDK 根目录 + 元数据 | 归属工具注册表 androidsdk 项 |
| platform-tools 版本 | SDK 实例的主版本标识（adb 版本号） | SDK 实例的属性 |
| ANDROID_HOME 引用 | 当前选中的 SDK 实例（state.json current） | 单引用 |

对象关系：一个 androidsdk 实例 ↔ 一个 SDK 根目录（1:1）；当前选择全局唯一（每工具单选，同现有模型）。

## 5. 业务字段

| 对象 | 字段 | 说明 |
|------|------|------|
| SDK 实例 | home | SDK 根目录（WINDOWS 反斜杠规范同现有） |
| | version | platform-tools 的 adb 版本（如 35.0.2） |
| | label | `Android SDK (platform-tools x.y.z)` |
| | source | system（AS 默认目录）/ env（ANDROID_HOME 指向）/ path（PATH 回溯）/ imported / downloaded |
| | binary | `<home>\platform-tools\adb(.exe)` |
| 工具注册表 androidsdk 项 | shims | adb、fastboot |
| | homeVar | ANDROID_HOME |
| | 下载源 | 官方 dl.google.com / 腾讯 mirrors.cloud.tencent.com/AndroidSDK/ |

## 6. 状态流转

无流程状态机。数据生命周期同现有工具模型：**未纳管 → 已纳管 →（可选）当前激活 → 已清理（cleanup 移除引用）**。"当前激活"是引用状态而非审批状态。

## 7. 业务规则

| # | 规则 |
|---|------|
| BR-1 | 同一 SDK 根目录重复纳管按 home 去重（与现有去重键一致：tool+home） |
| BR-2 | 导入校验失败（无 platform-tools/adb）即拒绝，报"目录里没有可用的 Android SDK" |
| BR-3 | ANDROID_HOME 写用户级（HKCU REG_EXPAND_SZ 语义同现有），同步广播；旧名 ANDROID_SDK_ROOT 只识别不写 |
| BR-4 | 下载走 URL 白名单（dl.google.com / mirrors.cloud.tencent.com）+ 公网 IP 校验 + 重定向同检（复用现有 SSRF 防护） |
| BR-5 | zip 解压拒绝 `..` 越界与链接成员（复用安全解压） |
| BR-6 | shim 语义与其他工具一致：未激活不留桩；Windows 双写 .cmd + sh twin |
| BR-7 | PATH 中 adb 回溯定位：取 adb 真实路径上溯两级（platform-tools → SDK 根） |
| BR-8 | 腾讯镜像 URL 形态：`mirrors.cloud.tencent.com/AndroidSDK/platform-tools-latest-{windows,linux}.zip` 与 cmdline-tools 对应包 |

## 8. 异常与边界场景

| 场景 | 处理 |
|------|------|
| ANDROID_HOME 指向不存在目录 | 忽略该项（扫描容错） |
| 同时存在 ANDROID_HOME 与 ANDROID_SDK_ROOT 且不一致 | 两处都尝试，按 home 去重合并 |
| AS 默认目录存在但无 platform-tools | 纳管但 doctor 提示 `no-pt`（warn） |
| 下载中断 | 沿用现有下载器重试/清理语义 |
| Linux 下 adb 无执行位 | 沿用解压后置位逻辑（chmod 0755） |

## 9. 非功能性要求

- 扫描无子进程（元数据解析），单次扫描 ≤ 现有四工具量级（<100ms 增量）
- golden 基线：现有四工具生成物不受影响（CI 字节级校验保持绿）
- 无遥测；更新检查不涉及本模块

## 10. 历史需求

不适用（新增）。

---

## 附A：PRD 完备性自检（D2-1）

功能类型判定：**主数据类**（聚合根=SDK 实例；数据生命周期状态；无流程流转）。

| # | 检查项 | 结果 |
|---|--------|------|
| G1 业务目标 | ✅ §1 |
| G2 功能范围 | ✅ §2（做/不做双列） |
| G3 角色定义 | ✅ §3 |
| G4 领域对象 | ✅ §4 |
| G5 对象关系 | ✅ §4 |
| T1 业务字段 | ✅ §5 |
| T1a 被引用关系 | ✅ 被 C-01/S-04/S-05 引用（切换/共享/诊断） |
| N1 非功能性 | ✅ §9 |
| N2 异常场景 | ✅ §8 |
| N3 业务规则细节 | ✅ §7 |

**结论：通过（9/9，主数据类必检项全过）**，进入领域建模。

## 附B：领域建模（D2-2）

PRD 标注模块名 → **范围明确**，跳过战略设计（Step 1~3），直接战术设计。

### 本域信息

| 项目 | 内容 |
|------|------|
| 本域名称 | androidsdk 纳管（归属"运行时纳管/切换"既有域，非独立业务域） |
| 领域分层 | 支撑域（为核心切换能力提供第五类工具候选） |
| 聚合根 | SDK 实例（Runtime{tool=androidsdk}，复用既有 Runtime 聚合，本 PRD 仅扩展注册项） |

### 领域对象清单

| 对象名 | 分类 | 与上级关系 | 核心业务字段 | 关键枚举字段 | 备注 |
|--------|------|-----------|------------|------------|------|
| SDK 实例 | 聚合根（Runtime 扩展） | — | home/version/label/binary | source | 版本=platform-tools 的 adb 版本 |
| platform-tools 版本 | 值对象 | SDK 实例属性 | 版本串 | — | 不可变，替换 |
| 工具注册表 androidsdk 项 | 基础数据 | 工具注册表 | shims/homeVar/下载源 | — | D-01 新增项 |

判定依据（A1-A4）：一致性边界=单实例目录自洽（A1 ✓）；被 state.current 引用（A2 ✓）；创建唯一入口=扫描/导入/下载（A3 ✓）；驱动 shim/env 生成（A4 ✓）→ 聚合根。platform-tools 版本无标识、值等价 → 值对象。

### 子流程 I/O 契约

无子流程（下载安装是一次性动作接口，非独立状态机； licenses 确认明确出范围）。

### 关联领域对象

| 关联领域对象 | 所属业务域 | 关联字段 | 需获取信息 | 触发关系 | 映射模式 |
|------------|--------|---------|----------|---------|---------|
| Runtime（既有四工具） | 运行时纳管域 | tool 枚举新增值 | 无 | 共用聚合结构 | 共享内核 |

### 过程数据分类表

| 数据项 | 类别 | 处理方式 |
|--------|------|---------|
| 镜像选项（official/tencent） | P4 请求载体 | GUI/CLI 参数 |
| 下载进度提示 | P2 展示数据 | 现有下载器语义 |
