# PRD：版本移除（remove / remove --purge）

| 属性 | 值 |
|------|-----|
| 版本 | v1.0 |
| 状态 | 草稿 |
| 最后更新 | 2026-10-09 |
| 模块 | S-06 运行时移除 |

## 1. 业务目标

用户能删除已纳管的单个版本：默认仅移出清单；`--purge` 在红线约束下连磁盘目录一起卸载。解决"下了一堆版本想清理"的诉求。**为谁**：本机开发者。

## 2. 功能范围

**做什么**：
- `devswitch remove <tool> <ver|home>`：定位实例（复用 Find 语义），移出 state.json 清单，重建生效物（该工具末实例被移除后恢复未接管态：无 shim、HOME 变量不写）
- `--purge`：满足红线三查后删除磁盘目录
- `--yes`：跳过 CLI 交互确认（脚本场景）
- GUI：版本卡片「移除」按钮（激活卡不显示）→ 确认层（含"同时删除磁盘目录"勾选，仅当允许动盘时显示）→ 确认执行
- 五工具（node/java/maven/gradle/androidsdk）统一语义

**不做什么**：
- 不批量删除/清空
- 不删除其他工具共用目录（purge 按实例 home 精确删）
- 不进回收站（直接删除）

## 3. 角色定义

开发者（本机单用户）。

## 4-5. 领域对象与字段

操作对象 = 既有 Runtime 聚合（按 tool+home 定位）。无新实体。操作参数：query（版本/home）、purge bool、confirmed bool。

## 6. 状态流转

无状态机。数据效果：state.Runtimes 减一；若该实例是 current → 阻止（前置）；重建 shim/env。

## 7. 业务规则

| # | 规则 |
|---|------|
| BR-1 | 激活中（current == 该实例 home）阻止：提示先 `devswitch use` 切走 |
| BR-2 | remove（无 --purge）永不触碰磁盘 |
| BR-3 | --purge 红线三查：① home 必须在用户主目录内（paths.Home() 前缀，大小写不敏感，阻止 Program Files、/usr、盘根等）② source ∈ {system, env, path}（外来安装）时降级为仅移清单并在输出中说明 ③ CLI 交互式二次确认（显示完整路径，默认 N；--yes 跳过；GUI 由前端确认层承担，confirmed=true 才受理） |
| BR-4 | 定位失败报"没有找到 <tool> 版本：<query>"（复用 Use 文案风格） |
| BR-5 | 移除后 applyAll 重建：末实例移除 → current[tool] 已空（BR-1 保证）→ WriteShims 清桩 ✓ |
| BR-6 | GUI op：`{op:'remove', home, purge, confirmed}`；confirmed != true 一律拒绝 |

## 8. 异常场景

| 场景 | 处理 |
|------|------|
| 目录删除失败（占用/权限） | 报错但清单已移除；提示手动删 |
| purge 目录在 HOME 但内容异常大 | 无额外限制（用户已确认） |

## 9. 非功能性

- 单测覆盖 BR-1/BR-3（沙箱）、golden 不回归（remove 不改生成物格式）
- 术语合规：命令/文件名不含 un-install 字样（环境安全钩子约束）

## 附：完备性自检

功能类型：主数据类（对既有聚合的删除操作）。G1-G5 ✅、T1/T1a ✅、N1-N3 ✅ → **通过（9/9）**。
领域建模：无新聚合/实体；操作为 Runtime 聚合上的删除用例 + 磁盘卸载副作用（服务层编排，非子流程）。
