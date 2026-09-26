# -*- coding: utf-8 -*-
"""WebView2 backend (Windows): pywebview window + pystray tray.

Dependency-free import: webview/pystray/PIL are only imported inside
run_gui()/tray helpers, so the CLI works without them installed."""
from __future__ import print_function, unicode_literals

import json
import os
import sys
from typing import Optional

from . import __app_name__, paths, service, uicore
from .store import load_state

MENU_CAP = 20


def _tray_image():
    # Same artwork as the cairo version in tray.py, drawn with PIL.
    from PIL import Image, ImageDraw

    size = 32
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    draw.rectangle([0, 0, size - 1, size - 1], fill=(15, 23, 41, 255))
    draw.ellipse([4, 4, size - 5, size - 5], fill=(28, 77, 212, 255))
    draw.line([11, 16, 22, 16], fill=(247, 250, 253, 255), width=3)
    draw.ellipse([18, 12, 25, 19], fill=(247, 250, 253, 255))
    return img


class WinTray(object):
    def __init__(self, app):
        self.app = app
        self.icon = None
        self.hide_notified = False

    def create(self):
        import pystray

        self.icon = pystray.Icon("devswitch", _tray_image(), __app_name__, menu=self._menu())
        self.icon.run_detached()
        return True

    def rebuild(self):
        if self.icon is None:
            return
        self.icon.menu = self._menu()
        try:
            self.icon.update_menu()
        except Exception:
            pass

    def _menu(self):
        import pystray

        state = load_state()
        return pystray.Menu(
            pystray.MenuItem("打开主窗口", lambda _item: self.app.show_window(), default=True),
            pystray.Menu.SEPARATOR,
            self._submenu("Node.js", "node", state),
            self._submenu("Java", "java", state),
            self._submenu("Maven", "maven", state),
            self._submenu("Gradle", "gradle", state),
            pystray.Menu.SEPARATOR,
            pystray.MenuItem("退出", lambda _item: self.app.quit_from_tray()),
        )

    def _submenu(self, title, tool, state):
        import pystray

        items = state.for_tool(tool)
        current = state.current_runtime(tool)
        current_home = current.home if current else ""
        if not items:
            return pystray.MenuItem(
                title, pystray.Menu(pystray.MenuItem("本机未发现", None, enabled=False))
            )
        entries = []
        shown = items[:MENU_CAP]
        for runtime in shown:
            # pystray validates the positional arg count of callables
            # (0/1/2 only); factories keep the closures at one argument.
            entries.append(
                pystray.MenuItem(
                    runtime.version,
                    self._pick_handler(tool, runtime.home),
                    radio=True,
                    checked=self._checked_handler(runtime.home, current_home),
                )
            )
        if len(items) > MENU_CAP:
            entries.append(
                pystray.MenuItem("在主窗口中查看全部…", lambda _item: self.app.show_window())
            )
        return pystray.MenuItem(title, pystray.Menu(*entries))

    def _pick_handler(self, tool, home):
        def handler(_item=None):
            self.app.switch_from_tray(tool, home)

        return handler

    @staticmethod
    def _checked_handler(home, current_home):
        def handler(_item):
            return os.path.normcase(home) == os.path.normcase(current_home)

        return handler

    def notify(self, title, body):
        if self.icon is None:
            return
        try:
            self.icon.notify(body, title=title)
        except Exception:
            pass


class WinApp(object):
    """uicore.UiController IO plus window/tray control for the pywebview app."""

    def __init__(self):
        self.window = None
        self.controller = None
        self.tray = None
        self.close_to_tray = True
        self._quitting = False

    # -- uicore IO ------------------------------------------------------
    def push(self, payload):
        if self.window is None:
            return
        try:
            self.window.evaluate_js(uicore.state_script(payload))
        except Exception:
            pass

    def refresh_tray(self):
        if self.tray is not None:
            self.tray.rebuild()

    def notify(self, title, body):
        if self.tray is not None:
            self.tray.notify(title, body)

    def is_visible(self):
        try:
            return bool(getattr(self.window, "visible", True))
        except Exception:
            return True

    def choose_folder(self, title):
        # type: (str) -> Optional[str]
        import webview

        try:
            result = self.window.create_file_dialog(
                webview.FOLDER_DIALOG, allow_multiple=False
            )
        except Exception:
            return None
        if isinstance(result, (list, tuple)):
            return result[0] if result else None
        return result or None

    # -- window / tray control -------------------------------------------
    def show_window(self):
        if self.window is None:
            return
        try:
            self.window.show()
        except Exception:
            pass

    def switch_from_tray(self, tool, home):
        if self.controller is not None:
            self.controller.apply_use(tool, home, from_tray=True)

    def notify_still_running(self):
        if self.tray is None or self.tray.hide_notified:
            return
        self.tray.hide_notified = True
        self.tray.notify(
            __app_name__ + " 仍在运行", "可从托盘打开主窗口。退出请用托盘菜单「退出」。"
        )

    def on_closing(self):
        if self._quitting or not self.close_to_tray:
            return None  # allow the window to close and the app to exit
        try:
            self.window.hide()
        except Exception:
            pass
        self.notify_still_running()
        return False  # cancel the close, keep running from the tray

    def quit_from_tray(self):
        self._quitting = True
        self.close_to_tray = False
        if self.tray is not None and self.tray.icon is not None:
            try:
                self.tray.icon.stop()
            except Exception:
                pass
        if self.window is not None:
            try:
                self.window.destroy()
            except Exception:
                pass


class _Api(object):
    """pywebview js_api surface: one send(json-string) method, mirroring the
    WebKit2 script-message handler.

    pywebview 5.1+ walks PUBLIC attributes of the js_api object to build the
    JS-side proxy, so every reference must stay underscore-private — a public
    `app` attribute would drag the whole WinApp/window/.NET graph into the
    walk."""

    def __init__(self):
        self._handler = None  # bound UiController.handle_message, set in run_gui

    def bind(self, handler):
        self._handler = handler

    def send(self, message):
        if isinstance(message, str):
            try:
                data = json.loads(message)
            except ValueError:
                data = {}
        elif isinstance(message, dict):
            data = message
        else:
            data = {}
        if self._handler is not None:
            self._handler(data)
        return ""


def run_gui(argv=None):
    try:
        import webview
    except ImportError:
        print("DevSwitch 图形界面需要 pywebview（WebView2）：")
        print("  pip install --user pywebview pystray pillow")
        print("命令行功能不受影响：devswitch list / use / doctor")
        return 1
    paths.local_bin()
    if not load_state().runtimes:
        service.merge_scan()
        service.ensure_applied()
    app = WinApp()
    api = _Api()
    window = webview.create_window(
        __app_name__,
        html=uicore._inline_html(),
        js_api=api,
        width=1100,
        height=720,
        min_size=(900, 560),
    )
    app.window = window
    app.controller = uicore.UiController(app)
    api.bind(app.controller.handle_message)
    window.events.loaded += app.controller.ready
    window.events.closing += app.on_closing
    try:
        app.tray = WinTray(app)
        app.tray.create()
    except Exception as exc:
        sys.stderr.write("DevSwitch tray failed: {}\n".format(exc))
        app.tray = None
        app.close_to_tray = False
    webview.start()
    return 0
