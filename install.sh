#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
PREFIX="${PREFIX:-${HOME}/.local}"
BIN_DIR="${PREFIX}/bin"
APP_DIR="${PREFIX}/share/applications"
ICON_DIR="${PREFIX}/share/icons/hicolor/scalable/apps"
SHARE_DIR="${PREFIX}/share/devswitch"

need_python() {
  if ! command -v python3 >/dev/null 2>&1; then
    echo "需要 python3。Debian 系发行版可安装：python3" >&2
    exit 1
  fi
}

need_gui_libs() {
  if python3 - <<'PY' >/dev/null 2>&1
import gi
gi.require_version("Gtk", "3.0")
gi.require_version("WebKit2", "4.0")
from gi.repository import Gtk, WebKit2
PY
  then
    return 0
  fi
  echo "图形界面需要 GTK 3 和 WebKit2（python3-gi）。" >&2
  echo "Debian 系发行版可安装：" >&2
  echo "  sudo apt install python3-gi gir1.2-gtk-3.0 gir1.2-webkit2-4.0 gir1.2-appindicator3-0.1" >&2
  echo "没有这些库时，命令行（devswitch list / use）仍可使用。" >&2
}

need_python
need_gui_libs

mkdir -p "$BIN_DIR" "$APP_DIR" "$ICON_DIR"

if [ "$ROOT" != "$SHARE_DIR" ]; then
  rm -rf "$SHARE_DIR"
  mkdir -p "$SHARE_DIR"
  cp -a "$ROOT/bin" "$ROOT/devswitch" "$SHARE_DIR/"
  find "$SHARE_DIR" -type d -name '__pycache__' -prune -exec rm -rf {} + 2>/dev/null || true
fi

chmod +x "$SHARE_DIR/bin/devswitch" "$SHARE_DIR/bin/devswitch-gui"
ln -sfn "$SHARE_DIR/bin/devswitch" "$BIN_DIR/devswitch"
ln -sfn "$SHARE_DIR/bin/devswitch-gui" "$BIN_DIR/devswitch-gui"

sed "s|^Exec=.*|Exec=${BIN_DIR}/devswitch-gui|" \
  "$ROOT/share/applications/devswitch.desktop" > "$APP_DIR/devswitch.desktop"
cp "$ROOT/share/icons/hicolor/scalable/apps/devswitch.svg" "$ICON_DIR/devswitch.svg"

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$APP_DIR" >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t "${PREFIX}/share/icons/hicolor" >/dev/null 2>&1 || true
fi

python3 "$BIN_DIR/devswitch" scan
python3 "$BIN_DIR/devswitch" hook install

echo
echo "DevSwitch 已安装到 ${SHARE_DIR}（不依赖本次克隆的目录，克隆可删）。"
echo "  命令行：devswitch list / use / doctor"
echo "  图形界面：devswitch-gui   或在开始菜单搜索 DevSwitch"
echo
echo "若 ~/.bashrc 里还有手写的 Node PATH，可运行：devswitch doctor --fix"
echo "当前已打开的终端请重开一次，或执行：source ~/.config/devswitch/env.sh"
echo "卸载：卸载脚本在仓库里，或删除 ${SHARE_DIR} 与 ${BIN_DIR}/devswitch*"
