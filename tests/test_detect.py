# -*- coding: utf-8 -*-
"""detect.py：家目录推导、PATH 枚举、扫描根与版本探测的双平台行为。"""
import os
from pathlib import Path

from devswitch import detect


# --------------------------------------------------------------------------
# node / java home derivation
# --------------------------------------------------------------------------

def test_node_home_from_binary_linux_layout(lin, tmp_path):
    node = tmp_path / "node-v22" / "bin" / "node"
    node.parent.mkdir(parents=True)
    node.write_text("", encoding="utf-8")
    assert detect.node_home_from_binary(node) == node.parent.parent


def test_node_home_from_binary_windows_layout(win, tmp_path):
    node = tmp_path / "nodejs" / "node.exe"
    node.parent.mkdir(parents=True)
    node.write_text("", encoding="utf-8")
    assert detect.node_home_from_binary(node) == node.parent


def test_java_home_from_binary_jre_layout(lin, tmp_path):
    java = tmp_path / "jdk8" / "jre" / "bin" / "java"
    java.parent.mkdir(parents=True)
    java.write_text("", encoding="utf-8")
    java.chmod(0o755)  # Linux 分支检查执行位；空文件默认无 +x
    assert detect.java_home_from_binary(java) == tmp_path / "jdk8"


def test_java_home_from_binary_windows(win, tmp_path):
    java = tmp_path / "jdk17" / "bin" / "java.exe"
    java.parent.mkdir(parents=True)
    java.write_text("", encoding="utf-8")
    assert detect.java_home_from_binary(java) == tmp_path / "jdk17"


def test_java_home_rejects_other_names(tmp_path, sandbox):
    java = tmp_path / "bin" / "javaw"
    java.parent.mkdir(parents=True)
    java.write_text("", encoding="utf-8")
    assert detect.java_home_from_binary(java) is None


# --------------------------------------------------------------------------
# _looks_system
# --------------------------------------------------------------------------

def test_looks_system_linux(lin):
    # pathlib renders "/usr/..." with backslashes on a Windows host, so this
    # branch can only be asserted faithfully on POSIX.
    if os.name != "posix":
        import pytest

        pytest.skip("Linux branch needs a POSIX host")
    assert detect._looks_system(Path("/usr/lib/jvm/jdk-17"))
    assert not detect._looks_system(Path("/opt/node-v22"))


def test_looks_system_windows(win, monkeypatch):
    monkeypatch.setenv("ProgramFiles", r"C:\Program Files")
    assert detect._looks_system(Path(r"E:\Program Files\Java\jdk-21"))
    assert detect._looks_system(Path(r"C:\Program Files\nodejs"))
    assert not detect._looks_system(Path(r"D:\tools\node22"))


# --------------------------------------------------------------------------
# _which_all on Windows: pure-Python PATHEXT enumeration
# --------------------------------------------------------------------------

def test_which_all_windows_prefers_exe_over_cmd(win, tmp_path, monkeypatch):
    d1 = tmp_path / "d1"
    d1.mkdir()
    (d1 / "node.exe").write_text("", encoding="utf-8")
    (d1 / "node.cmd").write_text("", encoding="utf-8")
    d2 = tmp_path / "d2"
    d2.mkdir()
    (d2 / "node.exe").write_text("", encoding="utf-8")
    monkeypatch.setenv("PATH", str(d1) + os.pathsep + str(d2))
    found = detect._which_all_windows("node")
    assert [Path(item).parent for item in found] == [d1, d2]


# --------------------------------------------------------------------------
# candidate roots
# --------------------------------------------------------------------------

def test_candidate_node_homes_windows_roots(win, tmp_path, monkeypatch):
    home = tmp_path / "userhome"
    (home / "node-v22.11.0-win-x64").mkdir(parents=True)
    monkeypatch.setattr(detect.paths, "home", lambda: home)
    monkeypatch.setenv("ProgramFiles", r"C:\Program Files")
    monkeypatch.setenv("ProgramFiles(x86)", "")
    monkeypatch.setenv("ProgramW6432", "")
    monkeypatch.setattr(detect, "_which_all", lambda name: [])
    candidates = {str(item) for item in detect._candidate_node_homes()}
    assert str(home / "node-v22.11.0-win-x64") in candidates


def test_candidate_java_homes_windows_roots(win, tmp_path, monkeypatch):
    pf = tmp_path / "ProgramFiles"
    (pf / "Java" / "jdk-21").mkdir(parents=True)
    (pf / "Eclipse Adoptium" / "jdk-17.0.9").mkdir(parents=True)
    monkeypatch.setenv("ProgramFiles", str(pf))
    monkeypatch.setenv("ProgramFiles(x86)", "")
    monkeypatch.setenv("ProgramW6432", "")
    monkeypatch.setattr(detect, "_which_all", lambda name: [])
    candidates = {str(item) for item in detect._candidate_java_homes()}
    assert str(pf / "Java" / "jdk-21") in candidates
    assert str(pf / "Eclipse Adoptium" / "jdk-17.0.9") in candidates


# --------------------------------------------------------------------------
# version probing (fake _run)
# --------------------------------------------------------------------------

def _fake_run(monkeypatch, responses):
    def fake(cmd, timeout=4):
        return responses.get(tuple(cmd) if isinstance(cmd, list) else cmd, "")
    monkeypatch.setattr(detect, "_run", fake)


def test_inspect_node_windows_root_layout(win, tmp_path, monkeypatch):
    home = tmp_path / "nodejs"
    home.mkdir()
    (home / "node.exe").write_text("", encoding="utf-8")
    _fake_run(monkeypatch, {(str(home / "node.exe"), "-v"): "v22.11.0\n"})
    runtime = detect.inspect_node(home)
    assert runtime is not None
    assert runtime.version == "22.11.0"
    assert runtime.binary.endswith("node.exe")


def test_inspect_java_vendor_detection(win, tmp_path, monkeypatch):
    home = tmp_path / "jdk17"
    (home / "bin").mkdir(parents=True)
    (home / "bin" / "java.exe").write_text("", encoding="utf-8")
    output = 'openjdk version "17.0.9" 2023-10-17\nTemurin Runtime Environment'
    _fake_run(monkeypatch, {(str(home / "bin" / "java.exe"), "-version"): output})
    runtime = detect.inspect_java(home)
    assert runtime.vendor == "Temurin"
    assert runtime.version == "17.0.9"


def test_inspect_node_rejects_dir_without_binary(win, tmp_path):
    empty = tmp_path / "empty"
    empty.mkdir()
    assert detect.inspect_node(empty) is None


# --------------------------------------------------------------------------
# infer_active
# --------------------------------------------------------------------------

def test_infer_active_by_java_home_env(win, monkeypatch, tmp_path):
    from devswitch.models import Runtime

    runtime = Runtime(
        tool="java", version="17.0.9", major="17",
        home=r"E:\jdks\jdk-17", binary=r"E:\jdks\jdk-17\bin\java.exe",
        source="local",
    )
    monkeypatch.setattr(detect.shutil, "which", lambda name: None)
    monkeypatch.setenv("JAVA_HOME", r"e:\jdks\jdk-17")  # case-insensitive match
    assert detect.infer_active([runtime], "java") is runtime
