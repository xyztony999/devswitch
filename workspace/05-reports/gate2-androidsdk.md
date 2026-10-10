# Gate 2 验证报告 — 代码→交付门控

**模块**：androidsdk（Android SDK 纳管）
**检查时间**：2026-10-08
**实现范围**：B1 代码生成（按 `02-detailed/api/api-androidsdk.md` v1.0）
**形态说明**：本机 Go CLI/桌面工具，无激活栈 manifest。BE-xx/FE-xx 企业栈清单 N/A；按引擎机制项（GR）+ 等价物执行。

## 引擎机制项（GR）

| 检查项 | 结果 | 证据 |
|--------|------|------|
| 编译必过 | ✅ | `go build ./...` / `go vet ./...` 全绿；`-tags gui` 构建通过 |
| 测试必过 | ✅ | `go test ./internal/...`：apply（golden 交叉断言）+ detect（androidsdk 检测/版本回退）全绿 |
| E2E 必补 | ✅ | 见下方"端到端验收" |
| 前后端契约对齐 | ✅ | 见下方"契约对照" |

## 契约对照（API 文档 §4.4 vs 实现）

| 契约项 | 实现 | 状态 |
|--------|------|------|
| Tool 枚举第五值 androidsdk | models.go 常量+Tools+Shims{adb,fastboot}+HomeVars{ANDROID_HOME} | ✅ |
| 检测：AS 目录/ANDROID_HOME/SDK_ROOT/PATH 回溯 | detect/androidsdk.go candidateAndroidSDKHomes 四来源+去重 | ✅（实测发现本机 E:\Program Files\Android 37.0.1） |
| 版本=Pkg.Revision（无子进程） | sdkVersionFromSourceProps + 回退 unknown | ✅（单测覆盖 35.0.2 解析与回退） |
| use → ANDROID_HOME + adb/fastboot shim + 会话 PATH | apply env 段（Linux/GitBash 双版，WIN 兜底变量）+ shim 三形态 | ✅（实测注册表/双 shim/env 段） |
| install androidsdk [--mirror official\|tencent] | downloader.installAndroidSDK，白名单+dl.google.com | ✅（构建级验证；网络下载走 CI 发布链路复检） |
| import 校验 platform-tools | inspectAndroidSDK INV-1 | ✅（实测非法目录拒绝） |
| which adb | service.WhichBinary androidsdk 分支 | ✅（实测输出 platform-tools\adb.exe） |
| export/apply 含 androidsdk | 按 Tools 遍历自动生效 | ✅（实测 .devswitch 五键） |
| doctor：home-registry/no-shim/no-pt | doctor.go 三项 | ✅（实测"一切正常"） |
| GUI 第五页（PageSpec） | 前端 TOOLS+图标+下载条变体（无版本输入/腾讯镜像） | ✅（CDP 断言全过） |

## 端到端验收记录

| # | 场景 | 结果 |
|---|------|------|
| 1 | 假 SDK import→list→use 35→current→which adb | ✅ 全链路正确 |
| 2 | 真实环境 scan | ✅ 发现本机既有 SDK（PATH 回溯，37.0.1） |
| 3 | shim 双写（.cmd + sh twin）内容 | ✅ 指向 platform-tools，含 missing 文案 |
| 4 | HKCU ANDROID_HOME 写入 | ✅ REG_SZ 正确 |
| 5 | GUI CDP：5 chips/7 导航项/第五页 hero+卡片/下载条无版本输入/chip 直达 | ✅ |
| 6 | golden 交叉断言（剔除 androidsdk 增量后与 Python 基线字节一致） | ✅ 四工具不回归 |

## 规则扫描（等价物）

无后端服务/无 MQ/无鉴权（本机工具）。安全约束复用既有防护并已核对：URL 白名单（新增 dl.google.com/mirrors.cloud.tencent.com 已在池）、解压越界拒绝、无子进程检测。

## 执行偏差处理记录

| # | 原偏差 | 处理结果 |
|---|--------|---------|
| 1 | cmdline-tools 未随 install 下载（包名 build 号漂移） | **已解决**：改为运行时解析 Android 仓库清单 `repository2-3.xml`（官方与腾讯镜像字节一致，sdkmanager 同源机制）动态获取当前包名；文件名过 `^commandlinetools-(win\|linux)-[0-9]+_latest\.zip$` 白名单正则防注入；归位标准布局 `<sdk>/cmdline-tools/latest`。真机端到端验证：解析出 `commandlinetools-win-16111833_latest.zip`，下载 147.8MB 成功，sdkmanager 全家桶就位，adb 可执行（v1.0.41） |

## 端到端验收补充（偏差修复后）

| # | 场景 | 结果 |
|---|------|------|
| 7 | `install androidsdk` 真机全流程（官方源） | ✅ platform-tools(7.7MB)+cmdline-tools(147.8MB) 下载/解压/归位/自动切换 |
| 8 | 跨盘安装修复（TMP 在 E:、目标 C:，rename 跨卷失败） | ✅ safeExtractSingleRoot 加 copyTree 回退 + cmdline-tools 同盘 staging（既有工具同样受益） |
| 9 | 同版本多实例精确切换 | ✅ InstallRuntime 改为入册后按 home 精确 Use（版本匹配会撞旧实例） |
| 10 | 清单解析单测（win/linux 选择、缺失、恶意包名拒绝） | ✅ |

## 测试覆盖率

单测：detect androidsdk 2 例、apply golden 交叉断言 2 例（改造）+ 既有 8 例保持。CLI/GUI E2E 如上表。覆盖率分层不适用（无 Service/Controller 分层）。

## 总结

**判定：PASS（含 1 项执行偏差待确认）**

- 引擎机制项全过；契约 10/10 对照一致；E2E 6/6 场景通过
- 唯一需要决策：cmdline-tools 是否接受 v1 不随装（偏差 #1）

**等待用户确认后进入交付（提交 → CI → v2.1.0 发布 → 工作区归档）。**
