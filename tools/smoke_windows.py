# -*- coding: utf-8 -*-
"""Sandboxed Windows smoke driver.

Runs the real CLI against a sandbox APPDATA/LOCALAPPDATA/USERPROFILE with the
registry writers replaced by recorders, so `use`/`hook install` never touch
the real HKCU environment. Usage:

  APPDATA=... LOCALAPPDATA=... USERPROFILE=... DS_DOCS=... \
      python tools/smoke_windows.py <devswitch args...>
"""
import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

SANDBOX = Path(os.environ["DS_SANDBOX"])

from devswitch import cli, paths, winenv  # noqa: E402

CALLS = []

paths.documents_dir = lambda: SANDBOX / "Documents"
winenv.ensure_path_entry = lambda target, first=True: CALLS.append(("path", target)) or True
winenv.set_user_java_home = lambda home: CALLS.append(("java_home", home)) or True
winenv.user_path_entries = lambda: []
winenv.get_user_java_home = lambda: None
winenv.broadcast_environment_change = lambda: CALLS.append(("broadcast", None)) or None


def main():
    argv = sys.argv[1:]
    code = cli.main(argv)
    if CALLS:
        print("[smoke] winenv calls:")
        for name, value in CALLS:
            print("[smoke]   {} -> {}".format(name, value))
    return code


if __name__ == "__main__":
    sys.exit(main())
