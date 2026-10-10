# ADR-001：devswitch 技术栈是否接入 ai-project-engine 技术栈池

| 属性 | 值 |
|------|-----|
| 版本 | v1.0 |
| 状态 | 已接受 |
| 最后更新 | 2026-10-08 |
| 决策类型 | 工程治理（非产品功能） |
| 模块名 | techstack-pool-decision |
| 触发需求 | 将 devswitch 的技术栈是否需要放到 ai-project-engine 的技术栈池内？ |

## 第一部分：需求澄清（D0）

### 1.1 概念解析

| 提取维度 | 内容 |
|---------|------|
| 业务术语 | 技术栈池（techstack profile，L2 层资产）；devswitch 技术栈（Go 2.0 单文件 CLI+GUI、1.x Python 命令/数据兼容层、state.json、shim 机制）；三层叠加（引擎核心 ⊕ 技术栈 ⊕ 业务 profile） |
| 业务流程 | 栈接入契约（extension-guide §4）：profile.yaml → stack.codegen（编排 agent + specs）→ stack.validation.rules → build 命令；业务 profile 经 `depends` 声明 → `/profile use` 激活叠加 |
| 业务角色 | 引擎维护者（栈池所有者/上游）；devswitch 项目维护者（栈消费方）；AI 流水线（/design、/feature 的执行者，规约消费者） |
| 业务规则 | C-01~C-11 接入门禁；`stack.layers` 枚举仅 `backend \| frontend`；backend 栈必填 `build.command`；`codegen` 必填 orchestrator_agent + specs；ER-01 叠加原则（引擎静态文本禁止 profile 专名） |
| 业务数据 | techstack profile.yaml、栈规约文档（api_spec/specs）、编排 agent、Gate 2 规则集（CRITICAL/WARNING 分级）、挂载记录（.profile-mounts.json） |

### 1.2 架构匹配

| 匹配对象 | 匹配来源 | 结果 |
|---------|---------|------|
| 产品业务域 | BA 文档（运行时纳管/版本切换/版本治理） | **无匹配** —— 本需求不涉及产品功能，属 profile 工程治理层 |
| 引擎栈契约 | 02-profile-spec §2.3、03-extension-guide §4 | **完全匹配**（作为变更目标架构，契约已完备定义） |
| devswitch profile 现状 | profile.yaml（depends: []）、.profile-mounts.json（stacks: []） | **部分匹配** —— 现为无栈激活（引擎 C-03 "纯 CLI 业务先例"即本项目），技术栈信息现以 `slots.language` 形式留在业务层 |
| 设计先例 | androidsdk 模块 D3/Gate 1（2026-10-08） | **部分匹配** —— 无栈形态的 Web 条款 N/A + CLI 等价物替代检查已在该模块实践过一次 |

### 1.3 需求类型判定与流水线裁剪

**判定：治理决策型需求**（governance decision），不是产品功能需求——不新增/修改任何 L2/L3/L4 业务能力，BA/AA/FL 零变更。

**流水线裁剪**（依 D0 判定驱动，非跳过）：

| 阶段 | 处理 | 理由 |
|------|------|------|
| D1 | 产出本文档（决策记录 + 蓝图） | 决策产物是架构决策记录，落 {ws.architecture} |
| D2 | N/A（§2.6 提供轻量领域视角备查） | 无功能场景/角色/实体需 PRD 化；完备性检查项 G1-G5 无一适用 |
| D3 | N/A | 无接口/命令/数据格式新增；"若建栈"的 D3 规约形态以蓝图形式并入 §3.3 |
| D4 | N/A | 无页面/交互新增；决策无 UI 载体 |

### 1.4 澄清结论（按澄清优先级逐项）

| 优先级 | 问题 | 结论（证据） |
|--------|------|------------|
| 业务边界 | 属于哪个业务域？ | 不属于产品业务域；属于**引擎 profile 体系**的 L2（技术栈层）治理。变更对象是 profile 组织方式，非 devswitch 产品 |
| 用户类型 | 涉及哪些用户？ | 单一终端角色：开发者（本机用户）。治理动作的执行者是引擎维护者与项目维护者（同一人时无协同成本） |
| 核心实体 | 核心对象与关系？ | techstack profile（被消费）— depends 声明（引用）— business profile（消费者）；激活链 = 引擎 ⊕ Σ栈 ⊕ 业务（见 §2.6） |
| 状态流转 | 有无状态流转？ | 无业务状态机。仅有资产生命周期：草稿 → 孵化（业务侧资产）→ 独立成栈 → （可选）贡献回引擎池 |
| 业务规则 | 必须满足的约束？ | ①栈契约四要件（layers/requires_env/build/codegen/validation）②C-01~C-11 门禁 ③ER-01 叠加原则 ④验收口径："D3 采用新栈规约 + /feature 由新栈 agent 执行" |
| 集成关系 | 与其他模块/系统交互？ | 引擎为上游（契约提供方）；androidsdk 等产品模块为下游（D3 规约消费方）；无外部系统集成 |

**D0 完成判定**：所有概念已匹配 ✓ 业务边界已确认 ✓ 用户类型明确 ✓ 核心实体与关系已识别 ✓ 无状态流转场景已确认 ✓ 关键规则已确认 ✓ 集成关系明确 ✓

---

## 第二部分：决策分析（D1 级架构决策）

### 2.1 现状事实（证据清单）

| # | 事实 | 证据 |
|---|------|------|
| F1 | devswitch 以无栈方式激活，是引擎 C-03 认可的先例 | profile.yaml `depends: []`；profile-spec v2.2 C-03 注记"空=无栈依赖，纯 CLI 业务先例" |
| F2 | 栈契约的形态枚举不含 CLI/桌面形态 | profile-spec §2.3 `stack.layers: backend \| frontend`，无 cli/desktop |
| F3 | 栈接入是重资产行为：编排 agent + specs + Gate 2 规则集 + build | extension-guide §4 四要件 + 验收口径 |
| F4 | devswitch 技术规约中"通用 Go"占比低，"项目特有"占比高 | 单文件布局、1.x Python 命令兼容、state.json 格式、shim 机制、双平台分发——均为本项目特有约定 |
| F5 | 无栈形态的流水线适配已有一次成功先例 | androidsdk 模块 D3+Gate 1："Web 条款按 N/A+原因处理，等价物（CLI 命令/GUI op/状态文件）替代检查"，Gate 1 PASS |
| F6 | 当前无第二个 Go/CLI 项目接入引擎的证据 | 栈池参考实现仅 java-microservice/vue3-engine（引擎设计文档引用）；本项目 profiles/ 下仅 devswitch |
| F7 | 技术栈信息已在业务层留有痕迹 | profile.yaml `slots.language: "Go（2.0 单文件实现；1.x Python…）"` |
| F8 | 引擎仓纯净性约束：引擎不持有 profile（D-05），栈入"引擎池"意味着上游引擎仓变更 | profile-spec v2.0 §2.4 注记；贡献回池需走引擎版本变更流程 |

### 2.2 候选方案

| 方案 | 内容 |
|------|------|
| A. 立即建栈入池 | 新建 `go-cli` techstack profile（或先入本项目 profiles/，或直接贡献引擎池），devswitch `depends: [go-cli]` |
| B. 维持现状 | 不建栈、不做任何沉淀；技术规约继续散落在 CLAUDE.md/knowledge/slots |
| C. 业务侧沉淀 + 触发式升级 | **推荐**。暂不建栈；将 Go 约定以业务 profile 自有资产（assets.rules）形式沉淀；明确升级为独立栈的触发条件与蓝图（第三部分） |

### 2.3 方案评估（对照栈接入契约）

| 评估维度 | 方案 A（立即建栈） | 方案 B（维持现状） | 方案 C（沉淀+触发） |
|---------|------------------|------------------|-------------------|
| 契约匹配 | ✗ `layers` 无 cli/desktop 枚举，需先改引擎契约（F2/F8），影响面溢出本项目 | ✓ 无契约接触 | ✓ 无契约接触；业务 assets 挂载是既有能力（profile-spec §2.2） |
| 分层纪律 | ✗ 规约大头是项目特有约定（F4），入通用栈违反"业务条目归业务 profile"（profile-spec §5.2） | ✗ 技术规约滞留业务层（slots.language，F7），纯粹性持续受损但无恶化 | △ 接受"暂由业务层托管"，以触发条件约束升级时机 |
| 复用 ROI | ✗ 单项目消费（F6），重资产摊不薄（F3） | — 不投入 | ✓ 零前置投入；第二次使用出现时再抽取（抽取时机符合"三次法则"） |
| 流水线可用性 | ✓ D3 槽位 {stack.api_spec} 填充；/feature 有栈 agent；GR-02 有编译命令 | ✗ 三项均缺，但 F5 证明无栈可跑通设计阶段 | △ 设计阶段依赖 N/A+等价物惯例（F5 固化为规则）；代码阶段缺口由升级触发条件覆盖（T2） |
| 维护成本 | 高：栈资产随引擎 apiVersion 演进维护 | 零 | 低：一份规则文件 + 蓝图文档 |
| 回退性 | 建栈后撤除需清理 depends/挂载 | — | 随时可执行（仅移除 assets 引用） |

### 2.4 决策

> **决策 DS-ADR-001：暂不将 devswitch 技术栈放入 ai-project-engine 技术栈池。**
> 以方案 C 过渡：Go 约定沉淀为 devswitch 业务 profile 自有资产（assets.rules），CLI 形态的设计阶段适配惯例固化为规则；满足任一升级触发条件（§3.1）时，再按蓝图（§3.2/§3.3）立项建 `go-cli` 栈——先在本项目 profiles/ 孵化，稳定后再评估贡献回引擎池（上游变更，F8）。

**核心理由**（对应证据）：

1. **契约缺口前置**：`stack.layers` 无 CLI/桌面枚举（F2），立即接入要么硬塞 backend 扭曲语义、要么先推引擎契约演进——两者影响面都超出本项目决策边界。
2. **分层纪律**：待沉淀规约中项目特有约定占大头（F4），现在入通用栈 = 把业务规则错放进栈层。
3. **复用时机未到**：栈池价值在跨项目复用，当前仅一个消费者（F6）；androidsdk 已证明无栈可完整跑通设计阶段（F5），缺口集中在未来的 /feature 代码生成——那正是触发条件 T2。

### 2.5 影响与实施清单（本清单为设计产物，编码/配置阶段执行，本次不落地）

| # | 动作 | 位置 | 说明 |
|---|------|------|------|
| I1 | 新建 Go 约定规则文件（CRITICAL/WARNING 分级：单文件约束、跨平台构建 `go build ./...`+`go vet`、1.x 命令与 state.json 兼容守卫、gofmt、GUI 弱依赖不侵入 CLI） | `profiles/devswitch/rules/go-cli-conventions.md` | 替代 slots.language 承载技术约定的职责 |
| I2 | profile.yaml 增挂载声明 | `assets.rules: [rules/go-cli-conventions.md]` | 挂载后随激活链生效（C-07 校验） |
| I3 | slots.language 保留但降格为"语言事实陈述" | profile.yaml | 不再作为唯一技术规约来源 |
| I4 | 设计阶段 CLI 适配惯例成文：Web 条款 N/A + 等价物（命令规格/GUI op/状态文件 schema）替代 | 并入 I1 规则文件"设计阶段"节 | 将 F5 先例固化为可复用惯例 |
| I5 | 知识库登记本决策 | knowledge/design-decisions/index.md 增行 D-7 | 保持决策可追溯 |
| I6 | I1-I2 落地后跑 `/profile check devswitch`（C-07 挂载校验） | — | 门禁复验 |

**对既有产物的影响**：无。BA/AA/FL/PRD/API/PageSpec/枚举池（androidsdk 模块全套）不因本决策变更。

### 2.6 领域视角（D2 级轻量建模，备查）

| 对象 | 分类 | 说明 |
|------|------|------|
| techstack profile | 实体（L2 层资产） | 契约四要件齐全方可激活；本决策判定其"暂不实例化" |
| business profile | 实体（L3 层资产） | devswitch 现役；`depends` 是其对栈的唯一引用方式 |
| 激活链 | 值对象（派生） | 引擎 ⊕ Σ栈 ⊕ 业务；由 ACTIVE_PROFILE/ACTIVE_STACKS 派生，随 use/off 重建 |
| Go 约定（go-cli-conventions） | 实体（业务侧资产，方案 C 新增） | 生命周期挂在 devswitch profile 下；升级时整体迁出为栈的 validation.rules + specs |
| 升级触发条件 | 值对象（不变式） | §3.1 三条件，满足其一即不可继续推迟 |

无状态机（资产生命周期见 §1.4）；无跨域集成（引擎为上游契约方，遵奉其 schema）。

---

## 第三部分：go-cli 栈蓝图（触发条件达成后启用）

### 3.1 升级触发条件（满足其一即立项）

| # | 条件 | 动机 |
|---|------|------|
| T1 | 第二个 Go/CLI 形态项目接入引擎 | 复用成立，抽取成本被摊薄（三次法则） |
| T2 | devswitch 启用 /feature 代码生成（B1+ 阶段） | 需要栈编排 agent 与 Gate 2 栈级规则；GR-02 编译命令需 `stack.build.command` 提供 |
| T3 | 引擎 `stack.layers` 契约扩展出 `cli`/`desktop` 枚举 | 契约缺口（F2）消除，接入的语义障碍不复存在 |

### 3.2 目标结构与 profile.yaml 草案

```
profiles/go-cli/
├── profile.yaml
├── agents/
│   └── orchestrator.md          # /feature 代码生成编排（CLI 语义：命令注册/状态文件读写/shim 生成）
├── skills/
│   └── go-cli-specs/SKILL.md    # D3 规约入口（§3.3）；api_spec = 命令规格 + state.json 契约
└── rules/
    └── backend-rules.md         # Gate 2 规则（由 I1 规则文件整体迁入并泛化）
```

```yaml
# 草案 —— layers 取值依赖 T3；未扩展前以 backend 占位并在 title 显式标注形态
apiVersion: 1
profile:
  name: go-cli
  type: techstack
  title: Go CLI/桌面单二进制栈
stack:
  layers: [backend]          # T3 达成后改为 [cli]
  requires_env: []           # 单仓项目无需代码根环境变量
  build: {command: "go build ./... && go vet ./...", module_pattern: "{module}"}
  codegen:
    orchestrator_agent: agents/orchestrator.md
    specs: skills/go-cli-specs/SKILL.md
  validation:
    rules: [rules/backend-rules.md]
```

devswitch 侧变更：`depends: [go-cli]`，重跑 `/profile check` + `/profile use`（挂载与 ACTIVE_STACKS 更新）。

### 3.3 D3/D4 的 CLI 语义适配要点（建栈时规约的核心内容）

| 引擎语义（Web 形态） | go-cli 栈等价物 | 先例 |
|--------------------|----------------|------|
| 接口（HTTP 路径/请求/响应/错误码） | 命令规格：命令族/子命令/参数/flags/退出码/输出格式 | api-androidsdk.md v1.0 已实践 |
| 表结构/DDL | state.json schema 与读写契约（升级/卸载保留不变量） | 同上（状态文件契约） |
| 领域事件/MQ | 无等价物 → N/A；GUI op（frontend 调用面）替代集成语义 | 同上（GUI op 契约） |
| 权限矩阵 | N/A（用户级单角色）→ doctor 诊断规则替代 | 同上 |
| PageSpec（D4） | 仅覆盖 GUI 页面；CLI 交互以输出/进度/错误呈现规格替代 | androidsdk-page.md 已实践 |

> 适配要点的沉淀路径：先按 I4 固化进业务侧规则文件（本决策），建栈时整体迁入栈 specs——避免规约写两遍。

## 附录：证据索引

| 引用 | 位置 |
|------|------|
| 栈契约 schema | {engine.doc.profile-spec} §2.3（02-profile-spec.md） |
| 栈接入四要件与验收口径 | {engine.doc.extension-guide} §4（03-extension-guide.md） |
| C-03 无栈先例注记 | 02-profile-spec.md v2.2 更新记录 |
| GR-02 编译命令来源 | engine-rules.md 三（门控流程规则） |
| 无栈设计先例 | 05-reports/gate1-androidsdk.md（形态说明节） |
| 现状配置 | profiles/devswitch/profile.yaml；.claude/.profile-mounts.json |

## 更新记录

| 版本 | 日期 | 更新说明 |
|------|------|---------|
| v1.0 | 2026-10-08 | 首次决策：暂不入池（方案 C）；含 D0 澄清报告、方案评估、go-cli 栈蓝图与升级触发条件 |
