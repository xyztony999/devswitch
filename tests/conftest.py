# -*- coding: utf-8 -*-
import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from devswitch import paths  # noqa: E402


class Sandbox(object):
    """Redirects every paths.* decision into tmp_path and lets tests force
    either platform branch regardless of the host OS."""

    def __init__(self, tmp_path, monkeypatch):
        self.tmp_path = tmp_path
        self.monkeypatch = monkeypatch
        self.is_windows = paths.IS_WINDOWS
        self._apply()

    def use(self, windows):
        self.is_windows = windows
        self._apply()

    # -- layout ---------------------------------------------------------
    @property
    def root(self):
        return self.tmp_path / ("win" if self.is_windows else "lin")

    @property
    def home(self):
        # Linux: the sandbox root plays $HOME directly so golden paths like
        # /home/tester/.config map 1:1 after the root substitution.
        return self.root / "home" if self.is_windows else self.root

    @property
    def config_dir(self):
        if self.is_windows:
            return self.root / "Roaming" / "devswitch"
        return self.home / ".config" / "devswitch"

    @property
    def data_dir(self):
        if self.is_windows:
            return self.root / "Local" / "devswitch"
        return self.home / ".local" / "share" / "devswitch"

    @property
    def local_bin(self):
        if self.is_windows:
            return self.data_dir / "bin"
        return self.home / ".local" / "bin"

    @property
    def documents(self):
        return self.root / "Documents"

    def _apply(self):
        for name in ("home", "config_dir", "data_dir", "local_bin", "documents_dir"):
            # bind via closure over self, refreshed per _apply()
            self.monkeypatch.setattr(paths, name, getattr(self, "_{}".format(name)))
        self.monkeypatch.setattr(paths, "IS_WINDOWS", self.is_windows)
        self.monkeypatch.setattr(
            paths, "bashrc", lambda: self.home / ".bashrc"
        )
        self.monkeypatch.setattr(
            paths, "profile", lambda: self.home / ".profile"
        )
        self.monkeypatch.setattr(
            paths,
            "powershell_profiles",
            lambda: [self.documents / "WindowsPowerShell" / "profile.ps1"],
        )
        self.monkeypatch.setattr(
            paths,
            "hook_targets",
            lambda: (
                paths.powershell_profiles() + [paths.bashrc(), paths.profile()]
                if self.is_windows
                else [paths.bashrc(), paths.profile()]
            ),
        )
        for prop in ("home", "config_dir", "data_dir", "local_bin"):
            getattr(self, prop).mkdir(parents=True, exist_ok=True)
        self.documents.mkdir(parents=True, exist_ok=True)

    def _home(self):
        return self.home

    def _config_dir(self):
        return self.config_dir

    def _data_dir(self):
        return self.data_dir

    def _local_bin(self):
        return self.local_bin

    def _documents_dir(self):
        return self.documents


@pytest.fixture
def sandbox(tmp_path, monkeypatch):
    return Sandbox(tmp_path, monkeypatch)


@pytest.fixture
def win(sandbox):
    sandbox.use(True)
    return sandbox


@pytest.fixture
def lin(sandbox):
    sandbox.use(False)
    return sandbox
