# 交互设计：版本卡片移除（PageSpec 增量）

| 属性 | 值 |
|------|-----|
| 版本 | v1.0 |
| 信息来源 | `../api/api-remove.md` |

## 1. 结构变更

五工具版本卡片（ver-card）动作区追加「移除」按钮；点击弹确认层（NModal）。

## 2. PageSpec

### 2.1 ver-card 动作区

| 元素 | 规格 |
|------|------|
| 移除按钮 | quaternary、size=small、danger 色；仅非激活卡显示（激活卡已有 disabled「已激活」） |
| 布局 | `[切换到此版本] [移除]` 并列；激活卡 `[已激活]`（无移除） |

### 2.2 确认层（NModal preset=dialog）

| 元素 | 规格 |
|------|------|
| 标题 | `移除 <label>` |
| 正文 | `<home>`（code 全路径）+ 说明文案：默认"仅从 DevSwitch 清单移除，不删除磁盘文件" |
| purge 勾选 | NCheckbox"同时删除磁盘目录"——仅当 `canPurge(item)`（前端依据 source ∉ {system,env,path} 且 home 在用户目录内估算；最终以后端判定为准）时显示；勾选后说明变"将永久删除上述目录"（danger 色） |
| 按钮 | `[取消]`（默认）`[确认移除]`（purge 勾选时 danger primary） |
| 提交 | `send({op:'remove', home, purge, confirmed: true})` |

### 2.3 反馈

| 场景 | flash |
|------|-------|
| 成功（仅清单） | `已移除 <label>（磁盘未动）` |
| 成功（purge） | `已移除 <label>，并删除目录` |
| 后端降级（外来安装） | 后端文案原样（"已仅移出清单"） |
| 激活阻止 | 理论不达（前端已隐藏按钮），后端兜底文案 |

## 3. 质量校验

- [x] 字段/文案全部可追溯 API §3/§4
- [x] 复用既有组件（NModal/NCheckbox/danger 态），无新范式
- [x] 危险操作三重防呆：非激活才显示 → 确认层 → purge 默认不勾
