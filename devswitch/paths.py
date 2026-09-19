# -*- coding: utf-8 -*-
from __future__ import print_function, unicode_literals

import os
from pathlib import Path


APP_ID = "devswitch"
HOOK_BEGIN = "# >>> devswitch >>>"
HOOK_END = "# <<< devswitch <<<"
SHIM_MARKER = "DevSwitch shim"

IS_WINDOWS = os.name == "nt"


def home():
    return Path(os.path.expanduser("~"))


def config_dir():
    if IS_WINDOWS:
        base = Path(os.environ.get("APPDATA") or home() / "AppData" / "Roaming")
    else:
        xdg = os.environ.get("XDG_CONFIG_HOME")
        base = Path(xdg) if xdg else home() / ".config"
    path = base / APP_ID
    path.mkdir(parents=True, exist_ok=True)
    return path


def data_dir():
    if IS_WINDOWS:
        base = Path(os.environ.get("LOCALAPPDATA") or home() / "AppData" / "Local")
    else:
        xdg = os.environ.get("XDG_DATA_HOME")
        base = Path(xdg) if xdg else home() / ".local" / "share"
    path = base / APP_ID
    path.mkdir(parents=True, exist_ok=True)
    return path


def state_file():
    return config_dir() / "state.json"


def env_file():
    return config_dir() / "env.sh"


def current_env_file():
    return config_dir() / "current.env"


def backup_dir():
    path = config_dir() / "backup"
    path.mkdir(parents=True, exist_ok=True)
    return path


def local_bin():
    if IS_WINDOWS:
        path = data_dir() / "bin"
    else:
        path = home() / ".local" / "bin"
    path.mkdir(parents=True, exist_ok=True)
    return path


def bashrc():
    # On Windows this is the Git Bash startup file; the hook format is the same.
    return home() / ".bashrc"


def profile():
    # On Windows this is the Git Bash login file; unused elsewhere there.
    return home() / ".profile"


def documents_dir():
    # type: () -> Path
    if not IS_WINDOWS:
        return home() / "Documents"
    try:
        import ctypes
        from ctypes import wintypes

        class GUID(ctypes.Structure):
            _fields_ = [
                ("Data1", wintypes.DWORD),
                ("Data2", wintypes.WORD),
                ("Data3", wintypes.WORD),
                ("Data4", ctypes.c_ubyte * 8),
            ]

        # FOLDERID_Documents {FDD39AD0-238F-46AF-ADB4-6C85480369C7}
        folderid = GUID(
            0xFDD39AD0, 0x238F, 0x46AF, (ctypes.c_ubyte * 8)(
                0xAD, 0xB4, 0x6C, 0x85, 0x48, 0x03, 0x69, 0xC7
            )
        )
        path_ptr = ctypes.c_wchar_p()
        if ctypes.windll.shell32.SHGetKnownFolderPath(
            ctypes.byref(folderid), 0, None, ctypes.byref(path_ptr)
        ) == 0:
            return Path(path_ptr.value)
    except Exception:
        pass
    return home() / "Documents"


def powershell_profiles():
    # type: () -> list
    """All-hosts profile for Windows PowerShell 5.1 and PowerShell 7+.

    The 5.1 profile is always written (may create the directory); the PS 7
    profile is only written when the user already has PowerShell 7."""
    docs = documents_dir()
    ps5 = docs / "WindowsPowerShell" / "profile.ps1"
    ps7 = docs / "PowerShell" / "profile.ps1"
    profiles = [ps5]
    if ps7.parent.exists() or ps7.exists():
        profiles.append(ps7)
    return profiles


def hook_targets():
    # type: () -> list
    """Startup files the env hook is installed into for this platform."""
    if IS_WINDOWS:
        return powershell_profiles() + [bashrc(), profile()]
    return [bashrc(), profile()]


def shim_filename(name):
    # type: (str) -> str
    return name + ".cmd" if IS_WINDOWS else name


def package_root():
    return Path(__file__).resolve().parent.parent
