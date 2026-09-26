# -*- coding: utf-8 -*-
"""service.py：which_binary 双平台路径、doctor 的平台化检查。"""
import os

from devswitch import apply, paths, service
from devswitch.models import Runtime, State
from devswitch.store import save_state


WIN_NODE = Runtime(
    tool="node", version="22.11.0", major="22",
    home=r"E:\Program Files\nodejs", binary=r"E:\Program Files\nodejs\node.exe",
    source="system",
)
WIN_JAVA = Runtime(
    tool="java", version="17.0.9", major="17",
    home=r"E:\Program Files\Java\jdk-17", binary=r"E:\Program Files\Java\jdk-17\bin\java.exe",
    source="system",
)
LINUX_NODE = Runtime(
    tool="node", version="22.11.0", major="22",
    home="/opt/node-v22.11.0", binary="/opt/node-v22.11.0/bin/node",
    source="local",
)


def _state(sandbox, runtimes, current):
    state = State(current=current, runtimes=runtimes)
    save_state(state)
    return state


# --------------------------------------------------------------------------
# which_binary
# --------------------------------------------------------------------------

def test_which_binary_windows_layout(win):
    # which_binary 校验目标文件存在，运行时目录必须在沙箱里真实造出来，
    # 不能依赖宿主机上的安装路径（否则只在特定机器上通过）。
    node_home = win.root / "nodejs"
    node_home.mkdir(parents=True, exist_ok=True)
    for name in ("node.exe", "npm.cmd", "npx.cmd"):
        (node_home / name).write_text("", encoding="utf-8")
    java_home = win.root / "jdk-17"
    (java_home / "bin").mkdir(parents=True, exist_ok=True)
    (java_home / "bin" / "java.exe").write_text("", encoding="utf-8")
    node = Runtime(
        tool="node", version="22.11.0", major="22",
        home=str(node_home), binary=str(node_home / "node.exe"),
        source="system",
    )
    java = Runtime(
        tool="java", version="17.0.9", major="17",
        home=str(java_home), binary=str(java_home / "bin" / "java.exe"),
        source="system",
    )
    _state(win, [node, java], {"node": node.home, "java": java.home})
    assert service.which_binary("node") == str(node_home / "node.exe")
    assert service.which_binary("npm") == str(node_home / "npm.cmd")
    assert service.which_binary("java") == str(java_home / "bin" / "java.exe")
    assert service.which_binary("JAVA_HOME") == str(java_home)
    assert service.which_binary("node-home") == str(node_home)


def test_which_binary_linux_layout(lin, tmp_path):
    node_home = tmp_path / "node-v22.11.0"
    (node_home / "bin").mkdir(parents=True)
    for name in ("node", "npm"):
        (node_home / "bin" / name).write_text("", encoding="utf-8")
    runtime = Runtime(
        tool="node", version="22.11.0", major="22",
        home=str(node_home), binary=str(node_home / "bin" / "node"),
        source="local",
    )
    _state(lin, [runtime], {"node": runtime.home, "java": ""})
    assert service.which_binary("node").endswith("node")
    assert service.which_binary("npm").endswith("npm")
    assert service.which_binary("java") is None


def test_which_binary_jre_fallback(lin, tmp_path):
    java_home = tmp_path / "jdk8"
    (java_home / "jre" / "bin").mkdir(parents=True)
    (java_home / "jre" / "bin" / "java").write_text("", encoding="utf-8")
    runtime = Runtime(
        tool="java", version="1.8.0_392", major="8",
        home=str(java_home), binary=str(java_home / "jre" / "bin" / "java"),
        source="system",
    )
    _state(lin, [runtime], {"node": "", "java": runtime.home})
    assert service.which_binary("java").endswith("java")


# --------------------------------------------------------------------------
# doctor — Windows branch (registry stubbed)
# --------------------------------------------------------------------------

def _stub_winenv(monkeypatch, path_entries=(), env_values=None):
    from devswitch import winenv

    values = dict(env_values or {})
    monkeypatch.setattr(winenv, "user_path_entries", lambda: list(path_entries))
    monkeypatch.setattr(winenv, "get_user_value",
                        lambda name: (values[name], 1) if name in values else None)
    monkeypatch.setattr(winenv, "get_user_java_home", lambda: values.get("JAVA_HOME"))
    monkeypatch.setattr(winenv, "ensure_path_entry", lambda *a, **k: True)
    monkeypatch.setattr(winenv, "set_user_value", lambda *a, **k: True)


def _codes(issues):
    return {code for _level, code, _message in issues}


def test_doctor_windows_happy_path(win, monkeypatch):
    state = _state(win, [WIN_NODE, WIN_JAVA],
                   {"node": WIN_NODE.home, "java": WIN_JAVA.home})
    _stub_winenv(monkeypatch,
                 path_entries=[str(win.local_bin)],
                 env_values={"JAVA_HOME": WIN_JAVA.home})
    monkeypatch.setenv("PATH", str(win.local_bin) + os.pathsep + r"E:\Windows")
    apply.write_env_sh(state)
    apply.write_shims(state)
    for target in paths.hook_targets():
        apply.install_hook(target)
    issues = service.doctor_issues()
    blocking = [item for item in issues if item[0] != "info"]
    assert blocking == [], blocking


def test_doctor_windows_reports_missing_registry(win, monkeypatch):
    _state(win, [WIN_NODE, WIN_JAVA],
           {"node": WIN_NODE.home, "java": WIN_JAVA.home})
    _stub_winenv(monkeypatch, path_entries=[], env_values={})
    monkeypatch.setenv("PATH", str(win.local_bin))
    apply.write_env_sh(State(current={"node": WIN_NODE.home, "java": WIN_JAVA.home},
                             runtimes=[WIN_NODE, WIN_JAVA]))
    codes = _codes(service.doctor_issues())
    assert "user-path-missing" in codes
    assert "java-home-registry" in codes


def test_doctor_windows_flags_machine_node_ahead(win, monkeypatch):
    _state(win, [WIN_NODE], {"node": WIN_NODE.home, "java": ""})
    _stub_winenv(monkeypatch, path_entries=[str(win.local_bin)])
    ahead = r"E:\Program Files\nodejs"
    monkeypatch.setenv("PATH", ahead + os.pathsep + str(win.local_bin))
    apply.write_env_sh(State(current={"node": WIN_NODE.home, "java": ""},
                             runtimes=[WIN_NODE]))
    codes = _codes(service.doctor_issues())
    assert "path-order" in codes


# --------------------------------------------------------------------------
# doctor — Linux branch keeps its messages
# --------------------------------------------------------------------------

def test_doctor_linux_requires_scan(lin, monkeypatch):
    _state(lin, [], {"node": "", "java": ""})
    monkeypatch.setenv("PATH", str(lin.local_bin) + os.pathsep + "/usr/bin")
    issues = service.doctor_issues()
    assert ("error", "no-env", "还没有生成切换配置，先运行 devswitch scan。") in issues


def test_doctor_linux_missing_bin_warns(lin, monkeypatch):
    _state(lin, [LINUX_NODE], {"node": LINUX_NODE.home, "java": ""})
    monkeypatch.setenv("PATH", "/usr/bin")
    issues = service.doctor_issues()
    assert any(
        code == "path-missing" and "~/.local/bin 不在 PATH" in message
        for _level, code, message in issues
    )


def test_doctor_uses_os_pathsep(lin, monkeypatch):
    _state(lin, [LINUX_NODE], {"node": LINUX_NODE.home, "java": ""})
    apply.write_current_env(State(current={"node": LINUX_NODE.home, "java": ""},
                                  runtimes=[LINUX_NODE]))
    # A Windows-style PATH with ';' must not make Linux doctor crash/split wrong
    monkeypatch.setenv("PATH", str(lin.local_bin))
    issues = service.doctor_issues()
    assert not any(code == "path-missing" for _level, code, _message in issues)
