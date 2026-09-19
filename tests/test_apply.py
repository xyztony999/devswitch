# -*- coding: utf-8 -*-
"""apply.py 生成物测试：Linux 侧与 golden 基线逐字节一致；Windows 侧精确断言。"""
import os
from pathlib import Path

import pytest

from devswitch import apply, paths
from devswitch.models import Runtime, State
from tests.conftest import ROOT

GOLDEN_DIR = ROOT / "tests" / "golden" / "linux"

LINUX_NODE = Runtime(
    tool="node",
    version="22.11.0",
    major="22",
    home="/opt/node-v22.11.0",
    binary="/opt/node-v22.11.0/bin/node",
    source="local",
)
LINUX_JAVA = Runtime(
    tool="java",
    version="17.0.9",
    major="17",
    home="/usr/lib/jvm/jdk-17.0.9",
    binary="/usr/lib/jvm/jdk-17.0.9/bin/java",
    source="system",
)


def _linux_state():
    return State(current={"node": LINUX_NODE.home, "java": LINUX_JAVA.home},
                 runtimes=[LINUX_NODE, LINUX_JAVA])


def _norm(text, sandbox):
    text = text.replace(str(sandbox.root), "/home/tester")
    return text.replace("\\", "/")


def _golden(name):
    return (GOLDEN_DIR / name).read_text(encoding="utf-8")


# --------------------------------------------------------------------------
# Linux branch — byte-identical to the pre-refactor baseline
# --------------------------------------------------------------------------

def test_linux_shim_node(lin):
    assert _norm(apply._shim_script("node", "node"), lin) == _golden("shim-node")


def test_linux_shim_node_tool(lin):
    assert _norm(apply._shim_script("node-tool", "pnpm"), lin) == _golden("shim-node-tool")


def test_linux_shim_java(lin):
    assert _norm(apply._shim_script("java", "java"), lin) == _golden("shim-java")


def test_linux_hook_snippet(lin):
    assert _norm(apply.hook_snippet(), lin) == _golden("hook-snippet")


def test_linux_current_env_and_env_sh(lin):
    state = _linux_state()
    apply.write_current_env(state)
    apply.write_env_sh(state)
    current = _norm(paths.current_env_file().read_text(encoding="utf-8"), lin)
    envsh = _norm(paths.env_file().read_text(encoding="utf-8"), lin)
    assert current == _golden("current.env")
    assert envsh == _golden("env.sh")


def test_linux_shims_are_executable(lin):
    written = apply.write_shims(_linux_state())
    assert written
    for path in written:
        assert os.access(path, os.X_OK)
    assert (lin.local_bin / "node").exists()
    assert not (lin.local_bin / "node.cmd").exists()


def test_linux_write_user_env_is_noop(lin):
    # Returns before the winenv import; must not raise on any host.
    assert apply.write_user_env(_linux_state()) is None


# --------------------------------------------------------------------------
# Windows branch
# --------------------------------------------------------------------------

WIN_NODE = Runtime(
    tool="node",
    version="22.11.0",
    major="22",
    home=r"E:\Program Files\nodejs",
    binary=r"E:\Program Files\nodejs\node.exe",
    source="system",
)
WIN_JAVA = Runtime(
    tool="java",
    version="17.0.9",
    major="17",
    home=r"E:\Program Files\Java\jdk-17",
    binary=r"E:\Program Files\Java\jdk-17\bin\java.exe",
    source="system",
)


def _win_state():
    return State(current={"node": WIN_NODE.home, "java": WIN_JAVA.home},
                 runtimes=[WIN_NODE, WIN_JAVA])


def test_windows_node_cmd_shim(win):
    text = apply._shim_script_windows("node", "node", _win_state())
    assert text.startswith("@echo off\r\n")
    assert 'if exist "E:\\Program Files\\nodejs\\node.exe" goto run' in text
    assert '"E:\\Program Files\\nodejs\\node.exe" %*' in text
    assert text.endswith("\r\n")


def test_windows_npm_targets_npm_cmd(win):
    text = apply._shim_script_windows("node", "npm", _win_state())
    assert "nodejs\\\\npm.cmd" in text or "nodejs\\npm.cmd" in text


def test_windows_java_cmd_shim_has_jre_fallback(win):
    text = apply._shim_script_windows("java", "keytool", _win_state())
    assert "jdk-17\\bin\\keytool.exe" in text
    assert "jdk-17\\jre\\bin\\keytool.exe" in text
    assert ":run" in text and ":run1" in text


def test_windows_sh_twin_uses_msys_paths(win):
    text = apply._shim_script_windows_sh("node", "node", _win_state())
    assert 'exec "/e/Program Files/nodejs/node.exe" "$@"' in text
    assert "#!/bin/sh" in text


def test_windows_sh_twin_npm_uses_dash_f(win):
    text = apply._shim_script_windows_sh("node", "npm", _win_state())
    assert '[ -f "/e/Program Files/nodejs/npm.cmd" ]' in text


def test_windows_write_shims_writes_both_twins(win):
    written = apply.write_shims(_win_state())
    names = {Path(item).name for item in written}
    assert {"node", "node.cmd", "npm", "npm.cmd", "java", "java.cmd", "javac", "javac.cmd"} <= names
    node_cmd = (win.local_bin / "node.cmd").read_bytes().decode("utf-8")
    assert "\r\n" in node_cmd and "\r\r" not in node_cmd
    node_sh = (win.local_bin / "node").read_bytes().decode("utf-8")
    assert "\r" not in node_sh


def test_windows_write_shims_skips_unselected_tools(win):
    state = State(current={"node": WIN_NODE.home, "java": ""}, runtimes=[WIN_NODE])
    apply.write_shims(state)
    assert (win.local_bin / "node.cmd").exists()
    assert not (win.local_bin / "java.cmd").exists()
    assert not (win.local_bin / "java").exists()


def test_windows_env_sh_gdbash(win):
    apply.write_env_sh(_win_state())
    text = paths.env_file().read_text(encoding="utf-8")
    assert 'DEVSWITCH_NODE_HOME="/e/Program Files/nodejs"' in text
    assert 'DEVSWITCH_JAVA_HOME_WIN="E:\\Program Files\\Java\\jdk-17"' in text
    assert 'DEVSWITCH_JAVA_HOME_MSYS="/e/Program Files/Java/jdk-17"' in text
    assert "_devswitch_front" in text
    # PATH entries must be colon-safe (no C:\ inside the PATH list)
    assert "_devswitch_front \"/" in text


def test_windows_hook_snippet_is_msys(win):
    snippet = apply.hook_snippet()
    assert snippet.startswith(paths.HOOK_BEGIN)
    assert '"/' in snippet  # POSIX-style env.sh path


def test_windows_ps1_hook_snippet(win):
    snippet = apply.hook_snippet_ps1()
    assert paths.HOOK_BEGIN in snippet and paths.HOOK_END in snippet
    assert "$env:Path.StartsWith($_ds_bin)" in snippet
    # single-quoted literal path, backslashes intact
    assert "'{}'".format(win.local_bin) in snippet


def test_windows_current_env_skipped(win):
    apply.write_current_env(_win_state())
    assert not paths.current_env_file().exists()


def test_windows_write_user_env_sets_registry(win, monkeypatch):
    calls = []
    from devswitch import winenv

    monkeypatch.setattr(winenv, "ensure_path_entry",
                        lambda target, first=True: calls.append(("path", target)) or True)
    monkeypatch.setattr(winenv, "set_user_java_home",
                        lambda home: calls.append(("java", home)) or True)
    apply.write_user_env(_win_state())
    assert ("path", str(win.local_bin)) in calls
    assert ("java", r"E:\Program Files\Java\jdk-17") in calls


def test_hook_install_and_replace_roundtrip(lin):
    rc_file = lin.home / ".bashrc"
    first = apply.install_hook(rc_file)
    assert first
    text_one = rc_file.read_text(encoding="utf-8")
    assert apply.hook_installed(rc_file)
    second = apply.install_hook(rc_file)
    assert not second  # idempotent update, not duplicate append
    text_two = rc_file.read_text(encoding="utf-8")
    assert text_two.count(paths.HOOK_BEGIN) == 1
    assert text_two == text_one


def test_hook_install_ps1_replaces_block(win):
    from devswitch import winenv  # noqa: F401  (winenv import guard)
    profile = win.documents / "WindowsPowerShell" / "profile.ps1"
    profile.parent.mkdir(parents=True, exist_ok=True)
    profile.write_text("Write-Host hi\n", encoding="utf-8")
    assert apply.install_hook(profile)
    assert apply.install_hook(profile) is False
    text = profile.read_text(encoding="utf-8")
    assert text.count(paths.HOOK_BEGIN) == 1
    assert text.startswith("Write-Host hi")


def test_remove_hook(win):
    profile = win.documents / "WindowsPowerShell" / "profile.ps1"
    profile.parent.mkdir(parents=True, exist_ok=True)
    profile.write_text("before\n", encoding="utf-8")
    apply.install_hook(profile)
    assert apply.remove_hook(profile)
    assert paths.HOOK_BEGIN not in profile.read_text(encoding="utf-8")
    assert profile.read_text(encoding="utf-8") == "before\n"


def test_comment_conflicting_path_lines(lin):
    rc = lin.home / ".bashrc"
    rc.write_text(
        "export PATH=$HOME/node-v20/bin:$PATH\n"
        "export OTHER=1\n",
        encoding="utf-8",
    )
    changed = apply.comment_conflicting_path_lines(rc)
    assert changed == ["export PATH=$HOME/node-v20/bin:$PATH"]
    text = rc.read_text(encoding="utf-8")
    assert "# export PATH=$HOME/node-v20/bin:$PATH" in text
    assert "export OTHER=1" in text
