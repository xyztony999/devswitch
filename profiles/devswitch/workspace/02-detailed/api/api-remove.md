# API 设计：版本移除（remove）

| 属性 | 值 |
|------|-----|
| 版本 | v1.0 |
| 状态 | 草稿（Gate 1 待检） |
| 关联 PRD | `../prd/remove.md`（主数据类，9/9） |
| 技术规约 | 无激活栈；接口层 = CLI 子命令 + GUI op |

## 1. 模块设计概述

### 1.1 背景与目标

见 PRD §1。

### 1.2 边界说明

| 边界 | 说明 |
|------|------|
| 做什么 | remove 移清单 / --purge 红线三查删盘 / CLI 交互确认 / GUI 确认层 |
| 不做什么 | 批量删除、回收站、跨实例目录清理 |
| 模块边界 | service 层编排（store 删条目 + apply 重建 + 磁盘删除）；不新增领域对象 |

### 1.3 设计决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | 命令名 `remove`（非 delete/uninstall） | 环境安全钩子对 un-install 字样敏感；remove 语义中性 |
| D2 | CLI 交互确认默认开，`--yes` 跳过；GUI 由前端确认层 + `confirmed` 字段承担 | 双端一致的安全语义，脚本可自动化 |
| D3 | 红线判定在 service 层单点实现（purgeAllowed） | CLI/GUI 共用，测试单点覆盖 |
| D4 | 外来安装 --purge 降级为仅移清单（非报错） | 用户意图（清理列表）仍达成，打断体验次之 |
| D5 | 磁盘删除后 state 已先行保存 | 删盘失败不回滚清单（PRD §8） |

### 1.4 领域对象总览

无新聚合。操作：Runtime 删除用例（tool+home 定位）。

## 3. 应用层规范

| 项 | 规范 |
|----|------|
| CLI | `devswitch remove <tool> <ver\|home> [--purge] [--yes]`；退出码 0/1/2 |
| GUI op | `{op:'remove', home:string, purge?:bool, confirmed?:bool}`；state 全量回推 |
| 错误文案 | `没有找到 <tool> 版本：<query>` / `<path> 是当前激活版本，请先 devswitch use 切换后再移除` / `--purge 仅允许删除用户目录内的安装（<path>），已仅移出清单` / `--purge 需要确认（--yes 或交互确认）` |

## 4. 聚合根用例设计（Runtime.remove）

### 4.1 service.RemoveRuntime(tool, query string, purge bool, confirmed bool) (*Runtime, error)

流程：
1. `state.Find(tool, query)` 定位（复用 Matches：home 精确/版本/大版本）→ 失败 BR-4
2. 激活检查：`state.Current[tool] == runtime.Home`（TrimRight 归一）→ BR-1 阻止
3. 移出：state.Runtimes 过滤 + Save（先存，D5）
4. purge 且 purgeAllowed(runtime)：
   - purgeAllowed = home 在 paths.Home() 内（大小写/分隔符归一后前缀判断）**且** source ∉ {system, env, path}
   - allowed && confirmed → `os.RemoveAll(home)`；失败 → 返回警告文案（清单已移）
   - allowed && !confirmed → BR-3-③ 文案（此时清单已移出——**修正**：确认应前置。见下）
5. applyAll 重建生效物

**流程修正（D5 冲突消解）**：确认检查（confirmed）在第 3 步**之前**——未确认直接返回错误，不动 state。磁盘删除失败才保持"清单已移"语义。

### 4.2 CLI 行为

- 解析 tool/query/--purge/--yes → confirmed = --yes || 交互 y（交互仅在 TTY 且 !--yes 时；purge 才需要确认，纯 remove 不询问）
- purge 时打印：`将删除目录：<home>` + `确认？[y/N]`

### 4.3 GUI 行为

- op 受理：confirmed 必须 true（BR-6）
- 完成 Push flash：`已移除 <label>`（purge 附加 `，已删除目录`或降级说明）

## 5. 模块清单

CLI 一条 + GUI op 一条 + service 函数一 + store 辅助（RemoveRuntime 按 home 过滤）。

## 6. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|------|------|---------|------|
| v1.0 | 2026-10-09 | initial | remove/--purge 分层 + 红线三查 + 双端确认 |
