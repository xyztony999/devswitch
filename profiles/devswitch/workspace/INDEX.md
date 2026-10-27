---
pipeline:
  - { id: business-arch, title: "业务架构 BA v1.0（3 L2 / 8 L3 / 15 L4）", path: "00-architecture/business-arch.md", status: complete }
  - { id: application-arch, title: "应用架构 AA v1.0（C×2 S×5 D×1 P×2）", path: "00-architecture/application-arch.md", status: complete }
  - { id: function-list, title: "功能清单 FL v1.0（开发者单章）", path: "00-architecture/function-list.md", status: complete }
  - { id: adr-001, title: "ADR-001 技术栈池接入决策 v1.0", path: "00-architecture/adr-001-techstack-pool.md", status: complete }
  - { id: enum-pool, title: "枚举池 v1.0（TOOL/SOURCE/MIRROR/ISSUE-LEVEL）", path: "01-solution/common-spec/enum-pool.md", status: complete }
  - { id: prd-androidsdk, title: "PRD androidsdk（主数据类，完备性 9/9）", path: "02-detailed/prd/androidsdk.md", status: complete }
  - { id: api-androidsdk, title: "API androidsdk v1.0（CLI 命令 + GUI op 契约）", path: "02-detailed/api/api-androidsdk.md", status: complete }
  - { id: interaction-androidsdk, title: "PageSpec androidsdk v1.0（第五导航页）", path: "02-detailed/interaction/androidsdk-page.md", status: complete }
  - { id: gate1-androidsdk, title: "Gate 1 报告 androidsdk（PASS）", path: "05-reports/gate1-androidsdk.md", status: complete }
  - { id: gate1-techstack-pool, title: "Gate 1 报告 techstack-pool-decision（PASS）", path: "05-reports/gate1-techstack-pool.md", status: complete }
reference:
  - { title: "引擎 Profile 契约（02-profile-spec）", path: "{engine.doc.profile-spec}" }
  - { title: "引擎接入指南（03-extension-guide）", path: "{engine.doc.extension-guide}" }
modules:
  - name: onboarding
    title: "接入与知识库冷启动"
    domain: platform
    stages:
      requirement: { status: complete, note: "profile 骨架 + README 溯源知识库（2026-09-27）" }
      api: { status: complete, note: "N/A：接入事务，无设计产物" }
      interaction: { status: complete, note: "N/A" }
  - name: androidsdk
    title: "Android SDK + platform-tools 纳管"
    domain: core
    stages:
      requirement: { status: complete, note: "D0-D2 完成（四项决策/PRD 9/9），2026-10-08" }
      api: { status: complete, note: "D3 api-androidsdk v1.0（CLI 命令 + GUI op 契约）" }
      interaction: { status: complete, note: "D4 androidsdk-page v1.0" }
      backend: { status: complete, note: "B1 五工具全链路完成（检测/下载/切换/治理/GUI）；v2.1.0 已发布" }
      validation: { status: complete, note: "Gate 2 PASS（gate2-androidsdk.md）；CLI 全链路 + GUI CDP + 真机 install 155MB 验收" }
    gates:
      gate1: { status: pass, note: "gate1-androidsdk.md，2026-10-08" }
      gate2: { status: pass, note: "gate2-androidsdk.md，2026-10-08；执行偏差已消解（cmdline-tools 清单动态解析）" }
  - name: remove
    title: "版本移除（remove / --purge）"
    domain: core
    stages:
      requirement: { status: complete, note: "D0 四决策用户确认；PRD 9/9，2026-10-09" }
      api: { status: complete, note: "api-remove.md v1.0（红线三查 + 双端确认）" }
      interaction: { status: complete, note: "remove-ui.md（卡片按钮 + 确认层）" }
      backend: { status: complete, note: "CLI+GUI+service 单测 6/6；真机四场景验收" }
      validation: { status: complete, note: "Gate 2 PASS 无偏差（gate2-remove.md）" }
    gates:
      gate1: { status: pass, note: "gate1-remove.md，2026-10-09" }
      gate2: { status: pass, note: "gate2-remove.md，2026-10-09；v2.2.0 发布" }
  - name: golang-python-autoswitch
    title: "Go 工具链 / Python / 项目级自动切换"
    domain: core
    stages:
      requirement: { status: complete, note: "D0 四决策用户确认；合并式设计文档（PRD/API/交互），2026-10-27" }
      api: { status: complete, note: "golang-python-autoswitch.md v1.0（两清单 API 实测验证）" }
      interaction: { status: complete, note: "七工具导航 + 下载条镜像变体；autoswitch 无 GUI 载体" }
      backend: { status: complete, note: "检测/下载/切换/治理全链路 + env-project + 独立钩子块" }
      validation: { status: complete, note: "Gate 2 PASS 无偏差；真机双下载/pip 实跑/七导航 CDP" }
    gates:
      gate1: { status: pass, note: "设计文档内自检 PASS，2026-10-27" }
      gate2: { status: pass, note: "gate2-golang-python-autoswitch.md；v2.3.0 发布" }
  - name: techstack-pool-decision
    title: "技术栈池接入决策（ADR-001）"
    domain: platform
    stages:
      requirement: { status: complete, note: "D0 澄清+决策分析 ADR-001 v1.0；D2 以 §2.6 等价物覆盖" }
      api: { status: complete, note: "N/A：治理决策模块无接口产物（ADR §1.3 裁剪）" }
      interaction: { status: complete, note: "N/A：无 UI 载体（ADR §1.3 裁剪）" }
      backend: { status: todo, note: "实施清单 I1-I6 待编码阶段（规则文件+assets 挂载）" }
    gates:
      gate1: { status: pass, note: "gate1-techstack-pool.md；G1-03~08 N/A 已注明原因" }
      gate2: { status: pending, note: "I1-I6 落地后执行" }
---

# DevSwitch 设计工作区索引

| 属性 | 值 |
|------|-----|
| Profile | devswitch（business，无技术栈依赖） |
| 更新 | 2026-10-08 |

## 文档清单

```
workspace/
├── INDEX.md                                ← 本索引
├── 00-architecture/
│   ├── business-arch.md                    BA v1.0
│   ├── application-arch.md                 AA v1.0
│   ├── function-list.md                    FL v1.0
│   └── adr-001-techstack-pool.md           ADR-001 技术栈池决策
├── 01-solution/common-spec/
│   ├── enum-pool.md                        枚举池 v1.0
│   ├── common-enums.md                     （占位）
│   └── select-pool.md                      （占位）
├── 02-detailed/
│   ├── prd/androidsdk.md
│   ├── api/api-androidsdk.md
│   └── interaction/androidsdk-page.md
└── 05-reports/
    ├── gate1-androidsdk.md                 Gate 1 PASS
    └── gate1-techstack-pool.md             Gate 1 PASS
```

## 变更历史

| 日期 | 变更 |
|------|------|
| 2026-10-08 | androidsdk 模块 D0-D4 + Gate 1 PASS（并行会话产物自根 workspace/ 迁入并合并） |
| 2026-10-08 | techstack-pool-decision 模块：ADR-001 决策"暂不入栈池（方案 C）"+ Gate 1 PASS |
| 2026-10-08 | androidsdk：B1 实现 + Gate 2 PASS + v2.1.0 发布（端到端流水线首次全程走通） |
| 2026-10-09 | remove 模块：D0-D4 + 双 Gate PASS + v2.2.0 发布（变更管理级联首跑） |
| 2026-10-27 | golang/python/autoswitch 三模块一批：D0-D4 + 双 Gate PASS + v2.3.0 发布 |

## 遗留决策点（编码阶段复核）

1. ~~腾讯镜像 cmdline-tools 包名核实（androidsdk API D5）~~ → 已消解：运行时解析 repository2-3.xml（官方/腾讯字节一致），真机验证 commandlinetools-win-16111833_latest.zip
2. ~~platform-tools 会话 PATH 钩子产物的 golden 新用例（androidsdk PRD §9）~~ → 已消解：交叉断言 + TestEnvShWithAndroidSDKSelected 落地
3. ADR-001 实施清单 I1-I6（go-cli-conventions 规则文件 + assets 挂载 + check 复验）
4. ADR-001 升级触发条件 T1/T2/T3 任一达成时，按蓝图立项 go-cli 栈
