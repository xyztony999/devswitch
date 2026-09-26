# -*- coding: utf-8 -*-
"""GTK3 + WebKit2 backend (Linux). Windows uses gui_win.py."""
from __future__ import print_function, unicode_literals

import json
import os
import sys

os.environ.setdefault("WEBKIT_DISABLE_COMPOSITING_MODE", "1")

import gi

gi.require_version("Gtk", "3.0")
gi.require_version("Gdk", "3.0")
gi.require_version("WebKit2", "4.0")
from gi.repository import Gdk, Gio, Gtk, WebKit2

from . import __app_name__, paths, service, uicore
from .store import load_state
from .tray import TrayController, notify_send


APP_ID = "io.github.xyztony999.devswitch"


def _js_payload(message):
    if hasattr(message, "get_js_value"):
        value = message.get_js_value()
        if value is None:
            return {}
        try:
            if hasattr(value, "is_string") and value.is_string():
                return json.loads(value.to_string())
            if hasattr(value, "to_json"):
                return json.loads(value.to_json(0))
            return json.loads(value.to_string())
        except (ValueError, TypeError):
            return {}
    return {}


class GtkIo(object):
    """uicore.UiController callbacks bound to the GTK window."""

    def __init__(self, window):
        self.window = window

    def push(self, payload):
        self.window.view.run_javascript(uicore.state_script(payload), None, None, None)

    def refresh_tray(self):
        app = self.window.get_application()
        if app is not None:
            app.refresh_tray()

    def notify(self, title, body):
        notify_send(title, body)

    def is_visible(self):
        return bool(self.window.get_mapped())

    def run_async(self, fn):
        import threading

        threading.Thread(target=fn, daemon=True).start()

    def dispatch(self, fn):
        # 后台线程只能通过 idle_add 回到 GTK 主线程
        GLib.idle_add(fn)

    def choose_folder(self, title):
        dialog = Gtk.FileChooserDialog(
            title=title, parent=self.window, action=Gtk.FileChooserAction.SELECT_FOLDER
        )
        dialog.add_button("取消", Gtk.ResponseType.CANCEL)
        dialog.add_button("导入", Gtk.ResponseType.OK)
        dialog.set_current_folder(str(paths.home()))
        response = dialog.run()
        chosen = dialog.get_filename()
        dialog.destroy()
        if response == Gtk.ResponseType.OK and chosen:
            return chosen
        return None


class MainWindow(Gtk.ApplicationWindow):
    def __init__(self, app):
        Gtk.ApplicationWindow.__init__(self, application=app)
        self.set_title(__app_name__)
        self.set_default_size(1100, 720)
        self.set_size_request(900, 560)

        rgba = Gdk.RGBA()
        rgba.parse("#1a1a1a")
        self.override_background_color(Gtk.StateFlags.NORMAL, rgba)

        manager = WebKit2.UserContentManager()
        manager.register_script_message_handler("devswitch")
        manager.connect("script-message-received::devswitch", self.on_js)

        self.view = WebKit2.WebView.new_with_user_content_manager(manager)
        settings = self.view.get_settings()
        settings.set_enable_developer_extras(False)
        self.view.connect("load-changed", self.on_load)
        self.view.connect("load-failed", self.on_load_failed)
        self.add(self.view)
        self.view.load_html(uicore._inline_html(), "file:///devswitch/")
        self.connect("delete-event", self.on_delete)
        self.controller = uicore.UiController(GtkIo(self))
        self.show_all()

    def on_delete(self, *_args):
        app = self.get_application()
        if app is not None and app.close_to_tray:
            self.hide()
            app.notify_still_running()
            return True
        return False

    def on_load_failed(self, _view, _event, uri, error):
        sys.stderr.write("DevSwitch UI load failed: {} {}\n".format(uri, error))
        return False

    def on_load(self, _view, event):
        if event == WebKit2.LoadEvent.FINISHED:
            self.controller.ready()

    def on_js(self, _manager, message):
        self.controller.handle_message(_js_payload(message))

    def apply_use(self, tool, home, from_tray=False):
        self.controller.apply_use(tool, home, from_tray=from_tray)


class Application(Gtk.Application):
    def __init__(self):
        Gtk.Application.__init__(
            self, application_id=APP_ID, flags=Gio.ApplicationFlags.FLAGS_NONE
        )
        self.win = None
        self.tray = None
        self.close_to_tray = True
        self._quitting = False
        self._held = False

    def do_activate(self):
        if self.win is not None:
            self.show_window()
            return
        self.win = MainWindow(self)
        try:
            self.tray = TrayController(self)
            self.tray.create()
        except Exception as exc:
            sys.stderr.write("DevSwitch tray failed: {}\n".format(exc))
            self.tray = None
            self.close_to_tray = False
        if self.tray is not None and not self._held:
            self.hold()
            self._held = True
        self.win.present()

    def show_window(self):
        if self.win is None:
            return
        self.win.present()
        self.win.deiconify()

    def refresh_tray(self):
        if self.tray is not None:
            self.tray.rebuild()

    def notify_still_running(self):
        if self.tray is None or self.tray.hide_notified:
            return
        self.tray.hide_notified = True
        notify_send(__app_name__ + " 仍在运行", "可从托盘打开主窗口。退出请用托盘菜单「退出」。")

    def switch_from_tray(self, tool, home, version):
        if self.win is None:
            return
        self.win.apply_use(tool, home, from_tray=True)

    def quit_from_tray(self):
        self._quitting = True
        self.close_to_tray = False
        if self.win is not None:
            self.win.destroy()
            self.win = None
        if self._held:
            self.release()
            self._held = False
        self.quit()


def run_gui(argv=None):
    paths.local_bin()
    if not load_state().runtimes:
        service.merge_scan()
        service.ensure_applied()
    app = Application()
    return app.run(argv if argv is not None else sys.argv)
