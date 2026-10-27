# Gate 2 报告 — golang / python / autoswitch（v2.3.0 批）

**时间**：2026-10-27　**判定：PASS（无执行偏差）**

## 引擎机制项

| 检查项 | 结果 |
|--------|------|
| 编译/vet/测试 | ✅ 全绿（golden 交叉断言泛化为 stripAddedToolLines，覆盖三家新增工具，原始四工具语义零回归） |
| E2E | ✅ 见下表 |

## 端到端验收记录（真机）

| # | 场景 | 结果 |
|---|------|------|
| 1 | scan 发现本机 Go 1.24.0（PATH 回溯 + VERSION 文件） | ✅ |
| 2 | `install python 3.12`（npmmirror 44.9MB）→ 切换 + which python | ✅ |
| 3 | pip 链路：which pip → python.exe；pip.cmd `-m pip %*`；真机 `pip --version` | ✅ |
| 4 | `install golang 1.27`（golang.google.cn 75.4MB，清单 sha256 显式校验）→ `~/go-1.27.3` + GOROOT 注册表 + go shim | ✅ |
| 5 | env-project：.nvmrc 单工具 / .devswitch 双工具 / 无标记空输出 | ✅ |
| 6 | env-project 幂等性（_devswitch_front）+ 同目录 fast-path | ✅ |
| 7 | GUI：7 导航（全带图标）+ 7 chips + Go 页两卡 + Python 页 | ✅ CDP 断言 |

## 途中修复（对既有代码的增益）

| # | 修复 |
|---|------|
| 1 | sidecar sha256 误信 HTML 错误页（golang.google.cn 对 .sha256 返回 200 HTML）——加十六进制长度防御 + downloadWithChecksum 显式通道，全部工具受益 |
| 2 | pbs 无独立 pip.exe——Windows .cmd/sh twin 特判 `python -m pip` |
| 3 | cleanup 钩子清理短路 bug（双块都要删） |
| 4 | cdn.npmmirror.com 入白名单（npmmirror binaries 实际文件 CDN） |

## D0 决策落实对照

三项全做 ✓ / golang.google.cn 主源 ✓ / pbs+npmmirror ✓ / 会话级（PROMPT_COMMAND + _devswitch_front，不写 state.json）✓。

**无偏差，进入交付（v2.3.0）。**
