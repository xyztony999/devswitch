# -*- coding: utf-8 -*-
"""Platform-neutral GUI core shared by the GTK (Linux) and WebView2 (Windows)
backends: HTML loading with a bridge adapter, the state payload builder, and
the message/op state machine."""
from __future__ import print_function, unicode_literals

import json
from pathlib import Path
from typing import Dict, Optional

from . import __version__, paths, service
from .store import load_state

UI_DIR = Path(__file__).resolve().parent / "ui"

# The bundled UI talks to Python through window.webkit.messageHandlers.devswitch
# (WebKit2). Under pywebview the transport is window.pywebview.api.send; this
# adapter builds the WebKit-shaped facade so the same bundle works in both.
BRIDGE_ADAPTER = """<script>
(function () {
  function install() {
    if (window.webkit && window.webkit.messageHandlers) { return true; }
    var api = window.pywebview && window.pywebview.api;
    if (!api || !api.send) { return false; }
    window.webkit = {
      messageHandlers: {
        devswitch: { postMessage: function (m) { api.send(m); } }
      }
    };
    return true;
  }
  if (!install()) {
    window.addEventListener('pywebviewready', function () { install(); });
  }
})();
</script>"""


def _inline_html():
    # type: () -> str
    built = UI_DIR / "dist" / "index.html"
    if built.exists():
        html = built.read_text(encoding="utf-8")
    else:
        html = (UI_DIR / "index.html").read_text(encoding="utf-8")
        css = (UI_DIR / "app.css").read_text(encoding="utf-8")
        js = (UI_DIR / "app.js").read_text(encoding="utf-8")
        html = html.replace(
            '<link rel="stylesheet" href="app.css" />',
            "<style>\n" + css + "\n</style>",
        )
        html = html.replace('<script src="app.js"></script>', "<script>\n" + js + "\n</script>")
    if "<head>" in html:
        # Inject first inside <head> so the facade exists before the app bundle runs.
        html = html.replace("<head>", "<head>" + BRIDGE_ADAPTER, 1)
    elif "</head>" in html:
        html = html.replace("</head>", BRIDGE_ADAPTER + "</head>", 1)
    else:
        html = BRIDGE_ADAPTER + html
    return html


def _runtime_dict(runtime):
    return runtime.to_dict() if runtime is not None else None


def build_state(page, selected="", flash=None):
    state = load_state()
    payload = {
        "page": page,
        "version": __version__,
        "current": {
            "node": _runtime_dict(state.current_runtime("node")),
            "java": _runtime_dict(state.current_runtime("java")),
        },
        "runtimes": [item.to_dict() for item in state.for_tool(page)]
        if page in ("node", "java")
        else [],
        "issues": [
            {"level": level, "code": code, "message": message}
            for level, code, message in service.doctor_issues()
        ],
        "selected": selected,
        "flash": flash,
        "paths": {
            "localBin": str(paths.local_bin()),
            "config": str(paths.config_dir()),
        },
    }
    if page in ("node", "java") and not payload["selected"]:
        current = state.current_runtime(page)
        if current is not None:
            payload["selected"] = current.home
        elif payload["runtimes"]:
            payload["selected"] = payload["runtimes"][0]["home"]
    if page == "doctor" and payload["selected"] == "":
        payload["selected"] = "0"
    return payload


def state_script(payload):
    # type: (Dict) -> str
    return "window.__setState && window.__setState({});".format(
        json.dumps(payload, ensure_ascii=False)
    )


class UiController(object):
    """One behavioral brain for every backend: tracks the active page and
    selection, dispatches ops coming from the web UI, and reports back
    through the `io` callbacks."""

    def __init__(self, io):
        self.io = io
        self.page = "node"
        self.selected = ""
        self._ready = False

    # -- backend signals ------------------------------------------------
    def ready(self):
        self._ready = True
        self.push()

    def push(self, flash=None):
        # type: (Optional[Dict]) -> None
        payload = build_state(self.page, self.selected, flash)
        self.selected = payload.get("selected") or self.selected
        self.io.push(payload)

    # -- messages from the web UI ---------------------------------------
    def handle_message(self, data):
        # type: (Dict) -> None
        op = data.get("op")
        if op == "page":
            self.page = data.get("page") or "node"
            self.selected = ""
            self.push()
        elif op == "select":
            self.selected = str(data.get("home") or "")
            self.push()
        elif op == "scan":
            service.merge_scan()
            service.ensure_applied()
            self.selected = ""
            self.push({"text": "已重新扫描本机 Node / Java。", "kind": "ok"})
            self.io.refresh_tray()
        elif op == "use":
            tool = "java" if self.page == "java" else "node"
            self.apply_use(tool, data.get("home") or "")
        elif op == "import":
            self._import()
        elif op == "fix":
            lines = service.doctor_fix()
            self.push(
                {
                    "text": "；".join(lines) if lines else "没有需要修复的项。",
                    "kind": "ok",
                }
            )

    def apply_use(self, tool, home, from_tray=False):
        current = load_state().current_runtime(tool)
        if current is not None and current.home.rstrip("/").rstrip("\\") == (home or "").rstrip("/").rstrip("\\"):
            if from_tray:
                self.io.notify("DevSwitch", "该版本已经激活")
            else:
                self.push({"text": "该版本已经激活", "kind": "ok"})
            return
        try:
            runtime = service.use(tool, home)
        except LookupError as exc:
            if from_tray:
                self.io.notify("切换失败", str(exc))
            else:
                self.push({"text": str(exc), "kind": "error"})
            return
        self.page = tool
        self.selected = runtime.home
        name = "Node.js" if tool == "node" else "Java"
        if from_tray:
            self.io.notify("已切换", "{} → {}".format(name, runtime.version))
            if self._ready and self.io.is_visible():
                self.push()
        else:
            self.push(
                {
                    "text": "已切换到 {} {}。".format(name, runtime.version),
                    "kind": "ok",
                }
            )
        self.io.refresh_tray()

    def _import(self):
        tool = "java" if self.page == "java" else "node"
        title = "选择 {} 安装目录".format("JDK" if tool == "java" else "Node.js")
        chosen = self.io.choose_folder(title)
        if not chosen:
            return
        try:
            runtime = service.import_path(tool, chosen)
        except (LookupError, ValueError) as exc:
            self.push({"text": str(exc), "kind": "error"})
            return
        self.page = tool
        self.selected = runtime.home
        self.push({"text": "已导入 {}。".format(runtime.version), "kind": "ok"})
        self.io.refresh_tray()
