#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""构建 Linux 系统包（deb / rpm），在 Linux（WSL 或 CI）上运行。

2.0 起打包 Go 二进制（需先构建好）：
    dist/bin/devswitch      静态 CLI（CGO_ENABLED=0，无系统依赖）
    dist/bin/devswitch-gui  GUI（CGO + webkit2gtk-4.0，动态链接）

用法：
    python3 tools/build_packages.py [--version 2.0.0] [--type deb|rpm|all]

在本机架构上原生打包（CI 里 amd64 与 arm64 各跑一次）：
    devswitch_<version>_amd64.deb   / devswitch_<version>_arm64.deb
    devswitch-<version>-1.x86_64.rpm / devswitch-<version>-1.aarch64.rpm

包内容：/usr/bin/devswitch（CLI）、/usr/bin/devswitch-gui（GUI）、
桌面入口与图标。GUI 的 webkit/gtk 依赖声明为弱依赖（Recommends）：
CLI 保证任何环境可用，GUI 在有图形栈的发行版上开箱即用。
用户级配置（shim、shell 钩子）由首次运行 devswitch scan 或 GUI 启动时自动完成。
"""
from __future__ import print_function, unicode_literals

import argparse
import platform
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CLI_BIN = ROOT / "dist" / "bin" / "devswitch"
GUI_BIN = ROOT / "dist" / "bin" / "devswitch-gui"

DEB_GUI_RECOMMENDS = "libgtk-3-0, libwebkit2gtk-4.0-37, libayatana-appindicator3-1"
RPM_GUI_RECOMMENDS = "gtk3, webkit2gtk4.0, libayatana-appindicator3-gtk3"


def native_deb_arch():
    # type: () -> str
    try:
        out = subprocess.check_output(
            ["dpkg", "--print-architecture"], stderr=subprocess.DEVNULL
        ).decode().strip()
        if out:
            return out
    except (OSError, subprocess.CalledProcessError):
        pass
    machine = platform.machine().lower()
    return "arm64" if machine in ("aarch64", "arm64") else "amd64"


def build_staging(staging):
    # type: (Path) -> None
    if not CLI_BIN.exists():
        raise SystemExit("缺少 %s，请先构建静态 CLI 二进制" % CLI_BIN)
    if not GUI_BIN.exists():
        raise SystemExit("缺少 %s，请先构建 GUI 二进制" % GUI_BIN)
    bin_dir = staging / "usr/bin"
    bin_dir.mkdir(parents=True, exist_ok=True)
    shutil.copy2(str(CLI_BIN), str(bin_dir / "devswitch"))
    shutil.copy2(str(GUI_BIN), str(bin_dir / "devswitch-gui"))
    (bin_dir / "devswitch").chmod(0o755)
    (bin_dir / "devswitch-gui").chmod(0o755)
    apps = staging / "usr/share/applications"
    apps.mkdir(parents=True, exist_ok=True)
    shutil.copy2(str(ROOT / "share/applications/devswitch.desktop"), str(apps / "devswitch.desktop"))
    icons = staging / "usr/share/icons/hicolor/scalable/apps"
    icons.mkdir(parents=True, exist_ok=True)
    shutil.copy2(str(ROOT / "share/icons/hicolor/scalable/apps/devswitch.svg"), str(icons / "devswitch.svg"))


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
            "Recommends: {recommends}\n"
            "Section: utils\n"
            "Priority: optional\n"
            "Homepage: https://github.com/xyztony999/devswitch\n"
            "Description: Node/Java/Maven/Gradle version manager via user-level shims\n"
            " Scan, manage and switch Node.js / Java / Maven / Gradle versions per\n"
            " user, with an optional tray GUI. The CLI binary is static; the GUI\n"
            " binary additionally needs a GTK/WebKit stack (weak dependency).\n"
            " Configuration happens in the user profile on first run.\n".format(
                version=version, arch=arch, recommends=DEB_GUI_RECOMMENDS
            ),
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
Summary: Node/Java/Maven/Gradle version manager via user-level shims
License: MIT
URL: https://github.com/xyztony999/devswitch
Recommends: __RECOMMENDS__
# CLI 为静态二进制；关闭自动依赖分析，避免为 GUI 二进制生成硬性 webkit 依赖
AutoReqProv: no
%description
Scan, manage and switch Node.js / Java / Maven / Gradle versions per user,
with an optional tray GUI. The CLI binary is static; the GUI binary
additionally needs a GTK/WebKit stack (weak dependency). Configuration
happens in the user profile on first run.

%install
cp -a __STAGING__/usr %{buildroot}/

%files
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
            RPM_SPEC.replace("__VERSION__", version)
            .replace("__RECOMMENDS__", RPM_GUI_RECOMMENDS)
            .replace("__STAGING__", str(staging))
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
    parser = argparse.ArgumentParser(description="构建 DevSwitch 的 deb/rpm 包（本机架构）")
    parser.add_argument("--version", required=True)
    parser.add_argument("--type", choices=("deb", "rpm", "all"), default="all")
    args = parser.parse_args()

    arch = native_deb_arch()
    rpm_arch = {"amd64": "x86_64", "arm64": "aarch64"}[arch]
    types = ("deb", "rpm") if args.type == "all" else (args.type,)
    for kind in types:
        builder = build_deb if kind == "deb" else build_rpm
        target_arch = arch if kind == "deb" else rpm_arch
        dest = builder(args.version, target_arch)
        print("built", dest)
    return 0


if __name__ == "__main__":
    sys.exit(main())
