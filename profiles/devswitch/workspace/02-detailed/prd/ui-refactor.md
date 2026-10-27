# 设计文档：UI 架构重构（tool-registry 组件化）

| 属性 | 值 |
|------|-----|
| 版本 | v1.0（合并式；纯架构重构 + 配置驱动 + 顺手微调，发布 v2.3.1） |
| 状态 | Gate 1 自检 PASS |
| 最后更新 | 2026-10-27 |
| D0 决策 | 前端注册表+组件化（后端 uicore/op 不动）；视觉顺手微调；修 versionInput 不一致 bug |

## 问题与目标

MainLayout.vue 巨石（700+ 行）：每接一个工具需改前端 6-7 处硬编码（TOOLS 数组/三张文案 map/镜像 map/模板 v-if 特判），且 androidsdk 特判名单未覆盖 golang/python（版本输入框 bug 的根因）。

**目标**：加第 8 个工具时前端只改 `tools.ts` 一处（+BrandLogo 图标）。

## 方案

### 1. 工具注册表 `frontend/src/tools.ts`

```ts
export interface ToolConfig {
  id: Tool
  label: string
  commands: string        // hero 说明用的命令串
  blurb: string           // hero 副标题
  mirrors: { label: string; value: string }[]
  versionInput: 'major' | 'none'  // 'none'=latest 通道（androidsdk/golang/python），隐藏输入框
  accent: string          // hero 图标底染的品牌色
}
export const TOOL_CONFIGS: ToolConfig[] = [ /* 七工具 */ ]
```

镜像随 config 走（切工具时回落逻辑已有）；versionInput:'none' 自动隐藏输入框并显示"最新版"文案——bug 随架构消除。

### 2. 组件树

```
MainLayout.vue（壳：顶栏 + 侧栏 + 分发，~200 行）
├── ToolPage.vue     hero + 下载条 + 版本卡网格 + 空态（全 config 驱动）
│   └── VersionCard.vue   单卡（logo/版本/meta/切换/移除按钮）
│       └── RemoveDialog.vue（移除确认层，props: target）
├── DoctorPage.vue   诊断 hero + issue 卡片流
└── AboutPage.vue    版本信息 + 更新检查 + 简介
```

store/bridge/types 不动；BrandLogo 保留（图标本就每工具一份）。

### 3. 顺手微调清单（小而克制）

- hero 图标底加各工具品牌色浅染（accent tint，如 Go #00ADD8/8% 底）
- 顶栏 chips 加 wrap（窄窗口 7 chips 不挤压）
- 下载条 'none' 模式的"最新版"说明与 androidsdk 文案统一

### 4. 验收（Gate 2）

- CDP 回归：七工具页 hero/卡片数/下载条形态、诊断/关于页、移除确认层流、chips 直达
- 视觉：逐页与重构前截图对照（布局不变，仅微调项）
- `go test` 不受影响（纯前端）；npm build 体积变化合理

## Gate 1 自检

PRD 完备性（重构类）：目标/边界/方案/验收齐（G1-G5 等价覆盖）；无新聚合/接口（op 协议不动）；风险（重构引入回归）由 CDP 全页回归覆盖。**PASS**
