#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""构建 Linux 系统包（deb / rpm），在 Linux（WSL 或 CI）上运行。

用法：
    python3 tools/build_packages.py [--version 1.1.0] [--type deb|rpm|all] [--arch all|amd64|arm64]

产物（输出到 dist/）：
    devswitch_<version>_amd64.deb  / devswitch_<version>_arm64.deb
    devswitch-<version>-1.x86_64.rpm / devswitch-<version>-1.aarch64.rpm

包内容为纯 Python 源码，各架构包体相同、仅 Architecture 字段不同；
系统包只放置文件与 /usr/bin 入口，用户级配置（shim、shell 钩子）由
/usr/bin/devswitch 在首次运行时自动完成。
"""
from __future__ import print_function, unicode_literals

import argparse
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
PACKAGE_DIR = "usr/lib/devswitch"          # Python 包安装位置（系统级、只读）
PY = "/usr/bin/python3"

WRAPPER_CLI = """#!/bin/sh
# DevSwitch 系统包入口：首次运行自动完成用户级初始化（扫描、shim、shell 钩子）
PACKAGE_DIR={package_dir}
INIT_MARKER="${{XDG_CONFIG_HOME:-$HOME/.config}}/devswitch/state.json"
if [ ! -f "$INIT_MARKER" ]; then
    PYTHONPATH="$PACKAGE_DIR" {python} -m devswitch scan >/dev/null 2>&1 || true
fi
exec env PYTHONPATH="$PACKAGE_DIR" {python} -m devswitch "$@"
""".format(package_dir="/" + PACKAGE_DIR, python=PY)

WRAPPER_GUI = """#!/bin/sh
# DevSwitch 图形界面入口（系统包）
exec env PYTHONPATH=/{package_dir} {python} -m devswitch gui "$@"
""".format(package_dir=PACKAGE_DIR, python=PY)


def read_version():
    text = (ROOT / "devswitch" / "__init__.py").read_text(encoding="utf-8")
    match = re.search(r'__version__ = "([^"]+)"', text)
    return match.group(1)


def build_staging(staging):
    # type: (Path) -> None
    pkg_dest = staging / PACKAGE_DIR / "devswitch"
    pkg_dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(
        ROOT / "devswitch",
        pkg_dest,
        ignore=shutil.ignore_patterns("__pycache__", "*.pyc"),
    )
    bin_dir = staging / "usr/bin"
    bin_dir.mkdir(parents=True, exist_ok=True)
    for name, content in (("devswitch", WRAPPER_CLI), ("devswitch-gui", WRAPPER_GUI)):
        target = bin_dir / name
        target.write_text(content, encoding="utf-8", newline="\n")
        target.chmod(0o755)
    apps = staging / "usr/share/applications"
    apps.mkdir(parents=True, exist_ok=True)
    shutil.copy2(ROOT / "share/applications/devswitch.desktop", apps / "devswitch.desktop")
    icons = staging / "usr/share/icons/hicolor/scalable/apps"
    icons.mkdir(parents=True, exist_ok=True)
    shutil.copy2(ROOT / "share/icons/hicolor/scalable/apps/devswitch.svg", icons / "devswitch.svg")


def build_deb(version, arch):
    # type: (str, str) -> Path
    staging = Path(tempfile.mkdtemp(prefix="ds-deb-"))
    try:
        build_staging(staging)
        control_dir = staging / "DEBIAN"
        control_dir.mkdir(exist_ok=True)
        (control_dir / "control").write_text(
            "Package: devswitch\n"
            "Version: {version}\n"
            "Architecture: {arch}\n"
            "Maintainer: xyztony999 <42613048+xyztony999@users.noreply.github.com>\n"
            "Depends: python3 (>= 3.8)\n"
            "Section: utils\n"
            "Priority: optional\n"
            "Homepage: https://github.com/xyztony999/devswitch\n"
            "Description: Node/npm/Java version manager via user-level shims\n"
            " Scan, manage and switch Node.js / npm / Java versions per user,\n"
            " with an optional tray GUI. Configuration happens in the user\n"
            " profile on first run; no root setup steps.\n".format(version=version, arch=arch),
            encoding="utf-8",
            newline="\n",
        )
        dest = ROOT / "dist" / "devswitch_{}_{}.deb".format(version, arch)
        dest.parent.mkdir(parents=True, exist_ok=True)
        subprocess.check_call(["dpkg-deb", "--build", str(staging), str(dest)])
        return dest
    finally:
        shutil.rmtree(staging, ignore_errors=True)


RPM_SPEC = """Name: devswitch
Version: __VERSION__
Release: 1
Summary: Node/npm/Java version manager via user-level shims
License: MIT
URL: https://github.com/xyztony999/devswitch
Requires: python3 >= 3.8
# 关闭自动依赖分析：避免对 Python 源码生成 python(abi) 版本依赖
AutoReqProv: no
%description
Scan, manage and switch Node.js / npm / Java versions per user, with an
optional tray GUI. Configuration happens in the user profile on first run.

%install
cp -a __STAGING__/usr %{buildroot}/

%files
/usr/lib/devswitch
/usr/bin/devswitch
/usr/bin/devswitch-gui
/usr/share/applications/devswitch.desktop
/usr/share/icons/hicolor/scalable/apps/devswitch.svg
"""


def build_rpm(version, arch):
    # type: (str, str) -> Path
    staging = Path(tempfile.mkdtemp(prefix="ds-rpm-"))
    work = Path(tempfile.mkdtemp(prefix="ds-rpmbuild-"))
    try:
        build_staging(staging)
        spec = (
            RPM_SPEC.replace("__VERSION__", version).replace("__STAGING__", str(staging))
        )
        (work / "devswitch.spec").write_text(spec, encoding="utf-8", newline="\n")
        for sub in ("BUILD", "RPMS", "SOURCES", "SPECS", "SRPMS"):
            (work / sub).mkdir(exist_ok=True)
        subprocess.check_call(
            [
                "rpmbuild", "-bb",
                "--target", "{}-linux".format(arch),
                "--define", "_topdir {}".format(work),
                str(work / "devswitch.spec"),
            ]
        )
        produced = list((work / "RPMS").rglob("*.rpm"))
        if not produced:
            raise SystemExit("rpmbuild 未产出 rpm")
        dest = ROOT / "dist" / "devswitch-{}-1.{}.rpm".format(version, arch)
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(str(produced[0]), str(dest))
        return dest
    finally:
        shutil.rmtree(staging, ignore_errors=True)
        shutil.rmtree(work, ignore_errors=True)


def main():
    parser = argparse.ArgumentParser(description="构建 DevSwitch 的 deb/rpm 包")
    parser.add_argument("--version", default=read_version())
    parser.add_argument("--type", choices=("deb", "rpm", "all"), default="all")
    parser.add_argument("--arch", choices=("all", "amd64", "arm64"), default="all")
    args = parser.parse_args()

    types = ("deb", "rpm") if args.type == "all" else (args.type,)
    archs = ("amd64", "arm64") if args.arch == "all" else (args.arch,)
    for kind in types:
        for arch in archs:
            rpm_arch = {"amd64": "x86_64", "arm64": "aarch64"}[arch]
            builder = build_deb if kind == "deb" else build_rpm
            target_arch = arch if kind == "deb" else rpm_arch
            dest = builder(args.version, target_arch)
            print("built", dest)
    return 0


if __name__ == "__main__":
    sys.exit(main())
