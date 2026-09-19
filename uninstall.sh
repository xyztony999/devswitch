#!/usr/bin/env bash
set -euo pipefail

PREFIX="${PREFIX:-${HOME}/.local}"
BIN_DIR="${PREFIX}/bin"
APP_DIR="${PREFIX}/share/applications"
ICON_DIR="${PREFIX}/share/icons/hicolor/scalable/apps"
SHARE_DIR="${PREFIX}/share/devswitch"

rm -f "$BIN_DIR/devswitch" "$BIN_DIR/devswitch-gui"
rm -f "$APP_DIR/devswitch.desktop"
rm -f "$ICON_DIR/devswitch.svg"
rm -rf "$SHARE_DIR"

echo "已移除程序文件和开始菜单项。"
echo "~/.local/bin 里的 node/java shim 仍保留，避免突然掉回系统旧版本。"
echo "若要连 shim 一并去掉，备份在："
echo "  ~/.config/devswitch/backup"
echo "配置在 ~/.config/devswitch/，shell 钩子在 ~/.bashrc / ~/.profile 的 # >>> devswitch >>> 段。"
