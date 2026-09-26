# 架构选型

- Go 单二进制：/usr/bin/devswitch（静态 CLI）+ /usr/bin/devswitch-gui（GUI）
- 分发面：Windows 安装向导+便携 zip+独立卸载器；Debian/Fedora 双格式包
- frontend/ 内嵌 GUI 资源；1.x Python 实现已被替代（数据兼容层保留）
