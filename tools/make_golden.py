# -*- coding: utf-8 -*-
"""Capture golden outputs of the Linux-side generators (run BEFORE the refactor).

Writes tests/golden/linux/* from the current apply.py output for a fixed fake
state. On a Windows host path separators are normalized to "/" so the files
match byte-for-byte what a Linux host would produce.
"""
import os
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from devswitch import apply, paths  # noqa: E402
from devswitch.models import Runtime, State  # noqa: E402

FAKE_HOME = "/home/tester"

STATE = State(
    current={
        "node": "/opt/node-v22.11.0",
        "java": "/usr/lib/jvm/jdk-17.0.9",
    },
    runtimes=[
        Runtime(
            tool="node",
            version="22.11.0",
            major="22",
            home="/opt/node-v22.11.0",
            binary="/opt/node-v22.11.0/bin/node",
            source="local",
        ),
        Runtime(
            tool="java",
            version="17.0.9",
            major="17",
            home="/usr/lib/jvm/jdk-17.0.9",
            binary="/usr/lib/jvm/jdk-17.0.9/bin/java",
            source="system",
        ),
    ],
)


class FakePaths(object):
    """Point the generators at a real temp dir, then map it back to a
    canonical POSIX-style home string in the captured output."""

    def __init__(self):
        self.real = Path(tempfile.mkdtemp(prefix="dsgolden-"))
        self.config = self.real / ".config" / "devswitch"
        self.config.mkdir(parents=True, exist_ok=True)
        self.bin = self.real / ".local" / "bin"
        self.bin.mkdir(parents=True, exist_ok=True)

    def config_dir(self):
        return self.config

    def env_file(self):
        return self.config / "env.sh"

    def current_env_file(self):
        return self.config / "current.env"

    def local_bin(self):
        return self.bin

    def backup_dir(self):
        return self.config / "backup"


def norm(text, fake):
    text = text.replace(str(fake.real), FAKE_HOME)
    return text.replace("\\", "/")


def main():
    out_dir = ROOT / "tests" / "golden" / "linux"
    out_dir.mkdir(parents=True, exist_ok=True)
    saved = {
        "config_dir": paths.config_dir,
        "env_file": paths.env_file,
        "current_env_file": paths.current_env_file,
        "local_bin": paths.local_bin,
        "backup_dir": paths.backup_dir,
    }
    saved_windows = paths.IS_WINDOWS
    paths.IS_WINDOWS = False  # always capture the Linux branch
    fake = FakePaths()
    paths.config_dir = fake.config_dir
    paths.env_file = fake.env_file
    paths.current_env_file = fake.current_env_file
    paths.local_bin = fake.local_bin
    paths.backup_dir = fake.backup_dir

    files = {
        "shim-node": apply._shim_script("node", "node"),
        "shim-node-tool": apply._shim_script("node-tool", "pnpm"),
        "shim-java": apply._shim_script("java", "java"),
        "hook-snippet": apply.hook_snippet(),
    }
    apply.write_current_env(STATE)
    apply.write_env_sh(STATE)
    files["current.env"] = fake.current_env_file().read_text(encoding="utf-8")
    files["env.sh"] = fake.env_file().read_text(encoding="utf-8")

    for name, saved_fn in saved.items():
        setattr(paths, name, saved_fn)
    paths.IS_WINDOWS = saved_windows

    for name, text in files.items():
        dest = out_dir / name
        dest.write_text(norm(text, fake), encoding="utf-8", newline="\n")
        print("wrote", dest)


if __name__ == "__main__":
    main()
