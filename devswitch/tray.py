# -*- coding: utf-8 -*-
from __future__ import print_function, unicode_literals

import subprocess
from pathlib import Path

import cairo
import gi

gi.require_version("Gtk", "3.0")
gi.require_version("AppIndicator3", "0.1")
from gi.repository import AppIndicator3, Gtk

from . import __app_name__, paths
from .store import load_state


MENU_CAP = 20


def ensure_tray_png():
    dest = paths.config_dir() / "tray.png"
    if dest.exists() and dest.stat().st_size > 0:
        return dest
    surface = cairo.ImageSurface(cairo.FORMAT_ARGB32, 24, 24)
    ctx = cairo.Context(surface)
    ctx.set_source_rgb(0.06, 0.09, 0.16)
    ctx.rectangle(0, 0, 24, 24)
    ctx.fill()
    ctx.set_source_rgb(0.11, 0.30, 0.83)
    ctx.arc(12, 12, 9, 0, 6.2832)
    ctx.fill()
    ctx.set_source_rgb(0.97, 0.98, 0.99)
    ctx.set_line_width(2.2)
    ctx.move_to(8, 12)
    ctx.line_to(16, 12)
    ctx.stroke()
    ctx.arc(16, 12, 3.2, 0, 6.2832)
    ctx.set_source_rgb(0.97, 0.98, 0.99)
    ctx.fill()
    surface.write_to_png(str(dest))
    return dest


class TrayController(object):
    def __init__(self, app):
        self.app = app
        self.indicator = None
        self.menu = None
        self._building = False
        self.hide_notified = False

    def create(self):
        icon = ensure_tray_png()
        theme_dir = str(paths.config_dir())
        icon_name = "tray" if icon is not None and icon.suffix == ".png" else "devswitch"
        self.indicator = AppIndicator3.Indicator.new_with_path(
            "devswitch",
            icon_name,
            AppIndicator3.IndicatorCategory.APPLICATION_STATUS,
            theme_dir,
        )
        self.indicator.set_title(__app_name__)
        self.indicator.set_status(AppIndicator3.IndicatorStatus.ACTIVE)
        self.rebuild()
        return True

    def rebuild(self):
        if self.indicator is None:
            return
        self._building = True
        menu = Gtk.Menu()
        state = load_state()

        open_item = Gtk.MenuItem(label="打开主窗口")
        open_item.connect("activate", lambda *_: self.app.show_window())
        menu.append(open_item)
        self.indicator.set_secondary_activate_target(open_item)
        menu.append(Gtk.SeparatorMenuItem())

        menu.append(self._tool_submenu("Node.js", "node", state))
        menu.append(self._tool_submenu("Java", "java", state))
        menu.append(Gtk.SeparatorMenuItem())

        quit_item = Gtk.MenuItem(label="退出")
        quit_item.connect("activate", lambda *_: self.app.quit_from_tray())
        menu.append(quit_item)

        menu.show_all()
        self.menu = menu
        self.indicator.set_menu(menu)
        self._building = False

    def _tool_submenu(self, title, tool, state):
        parent = Gtk.MenuItem(label=title)
        sub = Gtk.Menu()
        items = state.for_tool(tool)
        current = state.current_runtime(tool)
        current_home = current.home if current else ""
        if not items:
            empty = Gtk.MenuItem(label="本机未发现")
            empty.set_sensitive(False)
            sub.append(empty)
        else:
            group = []
            shown = items[:MENU_CAP]
            for runtime in shown:
                item = Gtk.RadioMenuItem.new_with_label(group, runtime.version)
                group = item.get_group()
                item.set_active(runtime.home == current_home)
                item.connect("activate", self._on_pick, tool, runtime.home, runtime.version)
                sub.append(item)
            if len(items) > MENU_CAP:
                more = Gtk.MenuItem(label="在主窗口中查看全部…")
                more.connect("activate", lambda *_: self.app.show_window())
                sub.append(more)
        parent.set_submenu(sub)
        return parent

    def _on_pick(self, item, tool, home, version):
        if self._building or not item.get_active():
            return
        self.app.switch_from_tray(tool, home, version)


def notify_send(title, body):
    try:
        subprocess.Popen(
            ["notify-send", "-a", __app_name__, "-i", "devswitch", title, body],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
    except OSError:
        pass
