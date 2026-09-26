#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""从已发布的 Release 资产生成各分发渠道的清单文件。

用法（发布后运行，hash 从 Release 资产计算）：
    python tools/make_manifests.py --version 1.1.0 \
        --x64-exe dist/devswitch-setup-1.1.0-x64.exe \
        --arm64-exe dist/devswitch-setup-1.1.0-arm64.exe

产出（dist/manifests/）：
    winget/devswitch.installer.yaml + devswitch.locale.zh-CN.yaml + devswitch.yaml（便携清单）
    scoop/devswitch.json
    homebrew/devswitch.rb

说明：winget 正式入库需向 microsoft/winget-pkgs 提交 PR；
scoop 可放进自建 bucket 仓库；Homebrew 需自建 tap 仓库（见 distributions/README.md）。
"""
from __future__ import print_function, unicode_literals

import argparse
import hashlib
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
GITHUB = "https://github.com/xyztony999/devswitch/releases/download/v{version}"


def _sha256(path):
    digest = hashlib.sha256()
    with open(str(path), "rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def render_winget(version, x64_sha, arm64_sha):
    base = GITHUB.format(version=version)
    installer = """# Produced by tools/make_manifests.py — submit to microsoft/winget-pkgs
PackageIdentifier: xyztony999.devswitch
PackageVersion: {version}
MinimumOSVersion: 10.0.0.0
Platform:
- Windows.Desktop
InstallerType: inno
Scope: user
InstallModes:
- interactive
- silent
- silentWithProgress
InstallerSwitches:
  Silent: /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
  SilentWithProgress: /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
  InstallLocation: /DIR="<INSTALLPATH>"
Installers:
- Architecture: x64
  InstallerUrl: {base}/devswitch-setup-{version}-x64.exe
  InstallerSha256: {x64_sha}
- Architecture: arm64
  InstallerUrl: {base}/devswitch-setup-{version}-arm64.exe
  InstallerSha256: {arm64_sha}
ManifestType: installer
ManifestVersion: 1.6.0
""".format(version=version, base=base, x64_sha=x64_sha, arm64_sha=arm64_sha)

    locale = """# Produced by tools/make_manifests.py
PackageIdentifier: xyztony999.devswitch
PackageVersion: {version}
PackageLocale: zh-CN
Publisher: xyztony999
PackageName: DevSwitch
PackageUrl: https://github.com/xyztony999/devswitch
License: MIT
ShortDescription: 用户级 Node / npm / Java / Maven / Gradle 版本切换器
ManifestType: locale
ManifestVersion: 1.6.0
""".format(version=version)

    version_yaml = """# Produced by tools/make_manifests.py
PackageIdentifier: xyztony999.devswitch
PackageVersion: {version}
DefaultLocale: zh-CN
ManifestType: version
ManifestVersion: 1.6.0
""".format(version=version)
    return {
        "winget/devswitch.installer.yaml": installer,
        "winget/devswitch.locale.zh-CN.yaml": locale,
        "winget/devswitch.yaml": version_yaml,
    }


def render_scoop(version, x64_sha, arm64_sha):
    base = GITHUB.format(version=version)
    manifest = {
        "version": version,
        "description": "用户级 Node / npm / Java / Maven / Gradle 版本切换器",
        "homepage": "https://github.com/xyztony999/devswitch",
        "license": "MIT",
        "notes": "需要 Python 3.8+（py launcher）。装完重开终端运行 devswitch。",
        "architecture": {
            "64bit": {
                "url": base + "/devswitch-setup-{v}-x64.exe".format(v=version),
                "hash": "sha256:" + x64_sha,
                "installer": {"file": "devswitch-setup-{v}-x64.exe".format(v=version),
                              "args": ["/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART"]},
            },
            "arm64": {
                "url": base + "/devswitch-setup-{v}-arm64.exe".format(v=version),
                "hash": "sha256:" + arm64_sha,
                "installer": {"file": "devswitch-setup-{v}-arm64.exe".format(v=version),
                              "args": ["/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART"]},
            },
        },
    }
    return {"scoop/devswitch.json": json.dumps(manifest, ensure_ascii=False, indent=2) + "\n"}


def render_homebrew(version, x64_sha):
    base = GITHUB.format(version=version)
    formula = """# Produced by tools/make_manifests.py — put into your own tap repo (e.g. homebrew-devswitch)
class Devswitch < Formula
  desc "User-level Node/npm/Java/Maven/Gradle version switcher"
  homepage "https://github.com/xyztony999/devswitch"
  url "{base}/devswitch_{{v}}_amd64.deb".freeze
  sha256 "{sha}"
  license "MIT"

  # 从 deb 提取纯 Python 包与入口脚本
  def install
    mkdir_p libexec
    system "ar", "x", cached_download
    system "tar", "-xf", "data.tar.*", "-C", "."
    prefix.install Dir["usr/lib/devswitch/*"]
    bin.install Dir["usr/bin/devswitch*"]
  end

  def caveats
    "首次运行 devswitch 会自动完成用户级初始化。"
  end

  test do
    system bin/"devswitch", "--version"
  end
end
""".format(base=base, v=version, sha=x64_sha)
    return {"homebrew/devswitch.rb": formula}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", required=True)
    parser.add_argument("--x64-exe", required=True, help="x64 安装器路径（计算 sha256）")
    parser.add_argument("--arm64-exe", required=True)
    args = parser.parse_args()

    x64_sha = _sha256(Path(args.x64_exe))
    arm64_sha = _sha256(Path(args.arm64_exe))
    files = {}
    files.update(render_winget(args.version, x64_sha, arm64_sha))
    files.update(render_scoop(args.version, x64_sha, arm64_sha))
    files.update(render_homebrew(args.version, x64_sha))

    out = ROOT / "dist" / "manifests"
    for rel, content in files.items():
        dest = out / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(content, encoding="utf-8", newline="\n")
        print("wrote", dest)
    return 0


if __name__ == "__main__":
    sys.exit(main())
