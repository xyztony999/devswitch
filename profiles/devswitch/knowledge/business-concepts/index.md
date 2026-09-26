# 业务概念

- **运行时（Runtime）**：被管理对象——node/npm/java/javac/maven/gradle
- **版本记录（state.json）**：当前选择；升级覆盖安装、卸载清理均保留
- **shim**：用户目录下的代理命令；执行期动态定位选中版本 → 已开终端即时生效
- **扫描源**：系统 JDK / 各厂商发行版 / 解压 Node 压缩包 / nvm-windows / scoop / sdkman / n / 手动导入
- **.devswitch 文件**：export/apply 版本对齐（可提交进仓库，团队对齐；--install-missing 自动补装）
- **doctor**：PATH/冲突 shell 配置体检（--fix 写钩子并注释手写 Node PATH）
