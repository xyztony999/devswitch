# 枚举池（DevSwitch）

| 属性 | 值 |
|------|-----|
| 版本 | v1.0 |
| 最后更新 | 2026-10-08 |

> 本产品为本机 CLI/桌面工具（无激活栈 manifest），枚举池集中登记跨模块共享的枚举定义。

## 业务枚举

### ENUM-TOOL 工具标识

| 值 | 含义 | 使用模块 |
|----|------|---------|
| node | Node.js | 全部（C-01/S-01~S-05） |
| java | Java | 全部 |
| maven | Maven | 全部 |
| gradle | Gradle | 全部 |
| androidsdk | Android SDK（v1.0 新增） | 全部 |

### ENUM-SOURCE 运行时来源

| 值 | 含义 | 使用模块 |
|----|------|---------|
| system | 系统安装/默认目录检测（含 Android Studio 目录） | S-01 |
| env | 环境变量指向（ANDROID_HOME 等，仅 androidsdk 产生） | S-01 |
| path | PATH 回溯发现（仅 androidsdk 产生） | S-01 |
| imported | 手动导入 | S-02 |
| downloaded | 本工具下载 | S-03 |
| local | 本机目录检测（node/maven/gradle 解压版） | S-01 |

### ENUM-MIRROR 下载镜像

| 值 | 含义 | 适用工具 |
|----|------|---------|
| official | 官方源 | 全部 |
| npmmirror | npmmirror 镜像 | node |
| tuna | 清华源 | java/maven/gradle |
| tencent | 腾讯 AndroidSDK 镜像 | androidsdk（v1.0 新增） |

### ENUM-ISSUE-LEVEL 诊断级别

| 值 | 含义 |
|----|------|
| error | 需要处理 |
| warn | 建议关注 |
| info | 提示 |

## 更新记录

| 版本 | 日期 | 更新说明 |
|------|------|---------|
| v1.0 | 2026-10-08 | 首次建池；登记 androidsdk 相关枚举（TOOL 增值、SOURCE 增 env/path、MIRROR 增 tencent） |
