# DevSwitch 知识注入

跨 Windows/Linux 的 **Node/npm/Java/Maven/Gradle 版本管理器**，命令行+图形界面双形态。
切换发生在**当前用户目录**（免管理员/sudo，不动系统级 update-alternatives）。

核心机制：**shim**（往用户目录写 node/npm/java 等；Windows 同生 .cmd 与 sh 双份）每次执行时定位当前选中版本，
**已打开终端即时生效**；同时维护 JAVA_HOME（.bashrc/.profile 钩子 / Windows 用户环境变量 + PowerShell profile + Git Bash 钩子）。

2.0 起为 **Go 单文件实现**（自包含二进制，弃 Python；1.x 命令与 state.json 数据格式完全兼容）。
命令族：scan/install/list/current/use/export/apply/import/which/doctor/cleanup。
