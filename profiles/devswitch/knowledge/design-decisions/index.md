# 设计决策（源自 README）

| 决策 | 结论 | 理由 |
|------|------|------|
| D-1 2.0 重写语言 | Go 单文件 | 自包含二进制零依赖；弃 Python |
| D-2 兼容策略 | 1.x 命令与数据格式完全兼容 | 覆盖安装即升级，state.json 原地保留 |
| D-3 权限模型 | 全用户级 | 免管理员/sudo；不碰系统级管理器 |
| D-4 生效机制 | shim 执行期定位 | 无需重开终端 |
| D-5 GUI 依赖 | Windows WebView2 / Linux GTK3+WebKitGTK（弱依赖） | 缺失时 CLI 不受影响 |
| D-6 分发 | Inno Setup（win）/ deb+rpm（双二进制 CLI+GUI）/ tar.gz | x64+arm64 |
