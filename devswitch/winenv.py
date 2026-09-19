# -*- coding: utf-8 -*-
"""Windows user-environment helpers: HKCU\\Environment plus the
WM_SETTINGCHANGE broadcast, so new terminals and Explorer-launched apps pick
up PATH / JAVA_HOME without rebooting. Pure stdlib."""
from __future__ import print_function, unicode_literals

import os
from typing import List, Optional, Tuple

ENV_VALUE = "Environment"
HWND_BROADCAST = 0xFFFF
WM_SETTINGCHANGE = 0x001A
SMTO_ABORTIFHUNG = 0x0002


def _winreg():
    import winreg

    return winreg


def _open_key(winreg, write=False):
    access = winreg.KEY_SET_VALUE | winreg.KEY_QUERY_VALUE if write else winreg.KEY_QUERY_VALUE
    return winreg.OpenKey(winreg.HKEY_CURRENT_USER, ENV_VALUE, 0, access)


def get_user_value(name):
    # type: (str) -> Optional[Tuple[str, int]]
    """Return (raw_value, reg_type) or None when unset. No expansion."""
    winreg = _winreg()
    try:
        with _open_key(winreg) as key:
            value, reg_type = winreg.QueryValueEx(key, name)
            return str(value), int(reg_type)
    except FileNotFoundError:
        return None
    except OSError:
        return None


def set_user_value(name, value):
    # type: (str, str) -> bool
    """Write a user env var; values containing % stay expandable (REG_EXPAND_SZ)."""
    winreg = _winreg()
    try:
        with _open_key(winreg, write=True) as key:
            reg_type = winreg.REG_EXPAND_SZ if "%" in value else winreg.REG_SZ
            winreg.SetValueEx(key, name, 0, reg_type, value)
        broadcast_environment_change()
        return True
    except OSError:
        return False


def delete_user_value(name):
    # type: (str) -> bool
    winreg = _winreg()
    try:
        with _open_key(winreg, write=True) as key:
            winreg.DeleteValue(key, name)
        broadcast_environment_change()
        return True
    except FileNotFoundError:
        return True
    except OSError:
        return False


def broadcast_environment_change():
    # type: () -> None
    try:
        import ctypes

        ctypes.windll.user32.SendMessageTimeoutW(
            HWND_BROADCAST, WM_SETTINGCHANGE, 0, "Environment", SMTO_ABORTIFHUNG, 5000, None
        )
    except Exception:
        pass


def split_path(raw):
    # type: (str) -> List[str]
    return [item.strip() for item in (raw or "").split(";") if item.strip()]


def _same_entry(entry, target):
    # type: (str, str) -> bool
    if os.path.normcase(entry) == os.path.normcase(target):
        return True
    try:
        if os.path.normcase(os.path.expandvars(entry)) == os.path.normcase(target):
            return True
    except (OSError, ValueError):
        pass
    return False


def user_path_entries():
    # type: () -> List[str]
    found = get_user_value("Path")
    return split_path(found[0]) if found else []


def ensure_path_entry(target, first=True):
    # type: (str, bool) -> bool
    """Make sure `target` is in the user PATH (moved to front when `first`).
    Returns True when the registry was modified."""
    entries = user_path_entries()
    kept = [item for item in entries if not _same_entry(item, target)]
    if len(kept) == len(entries) and entries and _same_entry(entries[0], target):
        return False
    if len(kept) == len(entries) and not first:
        return False
    new_entries = [target] + kept if first else kept + [target]
    raw = ";".join(new_entries)
    # Keep Path expandable like Windows itself stores it.
    winreg = _winreg()
    try:
        with _open_key(winreg, write=True) as key:
            winreg.SetValueEx(key, "Path", 0, winreg.REG_EXPAND_SZ, raw)
        broadcast_environment_change()
        return True
    except OSError:
        return False


def remove_path_entry(target):
    # type: (str) -> bool
    entries = user_path_entries()
    kept = [item for item in entries if not _same_entry(item, target)]
    if len(kept) == len(entries):
        return False
    winreg = _winreg()
    try:
        with _open_key(winreg, write=True) as key:
            if kept:
                winreg.SetValueEx(key, "Path", 0, winreg.REG_EXPAND_SZ, ";".join(kept))
            else:
                winreg.DeleteValue(key, "Path")
        broadcast_environment_change()
        return True
    except OSError:
        return False


def get_user_java_home():
    # type: () -> Optional[str]
    found = get_user_value("JAVA_HOME")
    if not found:
        return None
    value = found[0]
    return value if value.strip() else None


def set_user_java_home(home_path):
    # type: (str) -> bool
    return set_user_value("JAVA_HOME", home_path)


def to_msys_path(path):
    # type: (str) -> str
    """C:\\dir\\sub -> /c/dir/sub (UNC \\\\srv\\share -> //srv/share) for Git Bash."""
    path = str(path)
    if path.startswith("\\\\"):
        return path.replace("\\", "/")
    if len(path) >= 2 and path[1] == ":":
        drive = path[0].lower()
        return "/" + drive + path[2:].replace("\\", "/")
    return path.replace("\\", "/")
