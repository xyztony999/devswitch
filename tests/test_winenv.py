# -*- coding: utf-8 -*-
"""winenv.py：路径工具的双主机可移植性 +（仅 Windows）注册表读写往返。"""
import os

import pytest

from devswitch import winenv


def test_to_msys_path():
    assert winenv.to_msys_path(r"C:\Users\zhang\x") == "/c/Users/zhang/x"
    assert winenv.to_msys_path(r"e:\Program Files\nodejs") == "/e/Program Files/nodejs"
    assert winenv.to_msys_path(r"\\server\share\dir") == "//server/share/dir"
    assert winenv.to_msys_path("/already/posix") == "/already/posix"


def test_split_path_drops_empty_entries():
    assert winenv.split_path("a;;b;") == ["a", "b"]
    assert winenv.split_path("") == []


def test_same_entry_case_and_expand(monkeypatch):
    if os.name == "nt":
        assert winenv._same_entry(r"c:\X\Y", r"C:\x\y")
    monkeypatch.setenv("DS_TEST_BASE", r"C:\Base")
    assert not os.name == "nt" or winenv._same_entry(
        r"%DS_TEST_BASE%\bin", r"C:\Base\bin"
    )


@pytest.mark.skipif(os.name != "nt", reason="Windows registry only")
def test_registry_roundtrip():
    name = "DEVSWITCH_PYTEST_DUMMY"
    try:
        winenv.set_user_value(name, "hello")
        assert winenv.get_user_value(name) == ("hello", winenv._winreg().REG_SZ)
        winenv.set_user_value(name, "with %VAR% expand")
        value, reg_type = winenv.get_user_value(name)
        assert value == "with %VAR% expand"
        assert reg_type == winenv._winreg().REG_EXPAND_SZ
    finally:
        assert winenv.delete_user_value(name)
        assert winenv.get_user_value(name) is None
