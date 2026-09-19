# -*- coding: utf-8 -*-
"""uicore.py：状态 payload、HTML 桥接适配注入、UiController 状态机（无 GUI 依赖）。"""
import json

import pytest

from devswitch import paths, service, uicore
from devswitch.models import Runtime, State
from devswitch.store import save_state

LINUX_NODE = Runtime(
    tool="node", version="22.11.0", major="22",
    home="/opt/node-v22.11.0", binary="/opt/node-v22.11.0/bin/node",
    source="local",
)


class FakeIo(object):
    def __init__(self):
        self.pushed = []
        self.tray_refreshes = 0
        self.notifications = []
        self.folder = None
        self.visible = True

    def push(self, payload):
        self.pushed.append(payload)

    def refresh_tray(self):
        self.tray_refreshes += 1

    def notify(self, title, body):
        self.notifications.append((title, body))

    def is_visible(self):
        return self.visible

    def choose_folder(self, title):
        return self.folder


def test_inline_html_injects_bridge_adapter():
    html = uicore._inline_html()
    assert "pywebviewready" in html
    assert "messageHandlers" in html
    # the adapter's webkit facade must be defined before the app bundle runs
    assert html.index("window.webkit = {") < html.index("</body>")


def test_state_script_escapes_windows_paths():
    script = uicore.state_script({"paths": {"localBin": "C:\\devswitch\\bin"}})
    # json.dumps escapes the backslash; the JS string literal stays valid
    assert "C:\\\\devswitch\\\\bin" in script
    payload = json.loads(script[script.index("(") + 1 : script.rindex(")")])
    assert payload["paths"]["localBin"] == "C:\\devswitch\\bin"


def test_build_state_shape(lin):
    save_state(State(
        current={"node": LINUX_NODE.home, "java": ""},
        runtimes=[LINUX_NODE],
    ))
    payload = uicore.build_state("node")
    assert payload["page"] == "node"
    assert payload["current"]["node"]["home"] == LINUX_NODE.home
    assert payload["current"]["java"] is None
    assert payload["runtimes"][0]["version"] == "22.11.0"
    assert payload["selected"] == LINUX_NODE.home
    assert payload["paths"]["localBin"] == str(lin.local_bin)


def test_controller_page_and_select(lin):
    io = FakeIo()
    controller = uicore.UiController(io)
    controller.ready()
    assert io.pushed and io.pushed[0]["page"] == "node"
    controller.handle_message({"op": "page", "page": "java"})
    assert controller.page == "java"
    assert io.pushed[-1]["page"] == "java"
    controller.handle_message({"op": "select", "home": "/opt/x"})
    assert controller.selected == "/opt/x"


def test_controller_use_unknown_version_flashes_error(lin, monkeypatch):
    monkeypatch.setattr("devswitch.detect.scan_runtimes", lambda: [])
    io = FakeIo()
    controller = uicore.UiController(io)
    controller.ready()
    controller.handle_message({"op": "use", "home": "/opt/missing"})
    assert io.pushed[-1]["flash"]["kind"] == "error"


def test_controller_use_switches_and_refreshes_tray(lin, monkeypatch):
    monkeypatch.setattr("devswitch.detect.scan_runtimes", lambda: [])
    save_state(State(current={"node": LINUX_NODE.home, "java": ""},
                     runtimes=[LINUX_NODE]))
    io = FakeIo()
    controller = uicore.UiController(io)
    controller.ready()
    controller.handle_message({"op": "use", "home": LINUX_NODE.home})
    flash = io.pushed[-1]["flash"]
    assert flash is None or flash["kind"] != "error"  # same runtime: "已激活"
    assert io.tray_refreshes == 0  # no switch actually happened
    other = Runtime(
        tool="node", version="20.18.0", major="20",
        home="/opt/node-v20.18.0", binary="/opt/node-v20.18.0/bin/node",
        source="local",
    )
    save_state(State(current={"node": LINUX_NODE.home, "java": ""},
                     runtimes=[LINUX_NODE, other]))
    controller.handle_message({"op": "use", "home": other.home})
    assert io.pushed[-1]["flash"]["kind"] == "ok"
    assert io.tray_refreshes == 1
    assert (lin.local_bin / "node").exists()


def test_controller_use_from_tray_notifies(lin, monkeypatch):
    monkeypatch.setattr("devswitch.detect.scan_runtimes", lambda: [])
    other = Runtime(
        tool="node", version="20.18.0", major="20",
        home="/opt/node-v20.18.0", binary="/opt/node-v20.18.0/bin/node",
        source="local",
    )
    save_state(State(current={"node": LINUX_NODE.home, "java": ""},
                     runtimes=[LINUX_NODE, other]))
    io = FakeIo()
    controller = uicore.UiController(io)
    controller.ready()
    controller.apply_use("node", other.home, from_tray=True)
    assert io.notifications and io.notifications[0][0] == "已切换"


def test_controller_scan_op(lin, monkeypatch):
    monkeypatch.setattr("devswitch.detect.scan_runtimes", lambda: [])
    io = FakeIo()
    controller = uicore.UiController(io)
    controller.ready()
    controller.handle_message({"op": "scan"})
    assert io.pushed[-1]["flash"]["kind"] == "ok"
    assert io.tray_refreshes == 1


def test_controller_import_via_folder_dialog(lin, monkeypatch, tmp_path):
    fake_node = tmp_path / "node-distro"
    (fake_node / "bin").mkdir(parents=True)
    (fake_node / "bin" / "node").write_text("", encoding="utf-8")
    monkeypatch.setattr(
        "devswitch.detect.inspect_node",
        lambda home: Runtime(
            tool="node", version="20.1.0", major="20",
            home=str(home), binary=str(home / "bin" / "node"), source="imported",
        ),
    )
    io = FakeIo()
    io.folder = str(fake_node)
    controller = uicore.UiController(io)
    controller.ready()
    controller.handle_message({"op": "import"})
    assert io.pushed[-1]["flash"]["kind"] == "ok"
    assert io.pushed[-1]["selected"]


def test_gui_win_imports_without_deps():
    # The Windows backend must stay importable with no GUI deps installed
    # (CLI machines and the Linux GTK host both exercise this file's import).
    import devswitch.gui_win  # noqa: F401
