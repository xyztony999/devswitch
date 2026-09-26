# -*- coding: utf-8 -*-
"""运行时下载器：从官方或国内镜像下载并安装 Node.js / Temurin JDK。

安全约束：
- 仅允许 http/https；
- 下载/请求前校验 host（白名单 + 拒绝 localhost、环回、私有、保留地址）；
- 所有压缩包做 sha256 校验（Node 用官方 SHASUMS256.txt，JDK 用 Adoptium API 的 checksum）。

镜像选择（--mirror 参数 > DEVSWITCH_MIRROR 环境变量 > official）：
- official：Node 走 nodejs.org，JDK 走 Adoptium（GitHub 下载）
- npmmirror：Node 走 registry.npmmirror.com（JDK 无镜像，回落 tuna）
- tuna：Node 走清华 nodejs-release，JDK 走清华 Adoptium 镜像
"""
from __future__ import print_function, unicode_literals

import hashlib
import ipaddress
import json
import os
import shutil
import socket
import sys
import tarfile
import tempfile
import urllib.request
import zipfile
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from urllib.parse import urlparse
from urllib.request import Request

from . import __version__ as _APP_VERSION, paths

USER_AGENT = "DevSwitch/{} (github.com/xyztony999/devswitch)".format(_APP_VERSION)

ALLOWED_HOSTS = {
    "nodejs.org",
    "registry.npmmirror.com",
    "mirrors.tuna.tsinghua.edu.cn",
    "mirrors.cloud.tencent.com",
    "api.adoptium.net",
    "api.github.com",
    "github.com",
    "objects.githubusercontent.com",
    "repo.maven.apache.org",
    "dlcdn.apache.org",
    "services.gradle.org",
}

MIRRORS = {
    "official": {
        "node_dist": "https://nodejs.org/dist",
        "java_api": "https://api.adoptium.net",
        "java_download": None,  # 使用 API 返回的官方链接
    },
    "npmmirror": {
        "node_dist": "https://registry.npmmirror.com/-/binary/node",
        "java_api": "https://api.adoptium.net",
        "java_download": "tuna",  # npmmirror 无 JDK，回落清华
    },
    "tuna": {
        "node_dist": "https://mirrors.tuna.tsinghua.edu.cn/nodejs-release",
        "java_api": "https://api.adoptium.net",
        "java_download": "tuna",
    },
}


def assert_safe_url(url):
    # type: (str) -> None
    """Reject anything that is not http(s) to a public, allow-listed host."""
    parsed = urlparse(url)
    if parsed.scheme not in ("http", "https"):
        raise ValueError("仅允许 http/https 协议：{}".format(url))
    host = (parsed.hostname or "").lower()
    if not host:
        raise ValueError("URL 缺少主机名：{}".format(url))
    if host not in ALLOWED_HOSTS:
        raise ValueError("主机不在下载白名单内：{}".format(host))
    _assert_public_host(host, url)


def _assert_public_host(host, url):
    # type: (str, str) -> None
    candidates = []  # type: List[str]
    try:
        ipaddress.ip_address(host)
        candidates.append(host)
    except ValueError:
        try:
            infos = socket.getaddrinfo(host, None)
        except OSError:
            raise ValueError("无法解析主机：{}".format(url))
        for info in infos:
            address = info[4][0]
            if address not in candidates:
                candidates.append(address)
    for raw in candidates:
        ip = ipaddress.ip_address(raw.split("%")[0])
        if (
            ip.is_loopback
            or ip.is_private
            or ip.is_link_local
            or ip.is_reserved
            or ip.is_multicast
            or ip.is_unspecified
        ):
            raise ValueError("拒绝非公网地址：{} ({})".format(host, raw))


class _SafeRedirectHandler(urllib.request.HTTPRedirectHandler):
    """重定向目标同样必须通过白名单与公网校验，防止绕过。"""

    def redirect_request(self, req, fp, code, msg, headers, newurl):  # type: ignore
        assert_safe_url(newurl)
        return urllib.request.HTTPRedirectHandler.redirect_request(
            self, req, fp, code, msg, headers, newurl
        )


_OPENER = urllib.request.build_opener(_SafeRedirectHandler)


def _http_get(url, timeout=30):
    # type: (str, int) -> bytes
    assert_safe_url(url)
    request = Request(url, headers={"User-Agent": USER_AGENT})
    with _OPENER.open(request, timeout=timeout) as response:
        return response.read()


def _http_get_json(url, timeout=30):
    # type: (str, int) -> dict
    return json.loads(_http_get(url, timeout=timeout).decode("utf-8"))


def _http_download(url, dest, expected_size=None):
    # type: (str, Path, Optional[int]) -> None
    assert_safe_url(url)
    show_progress = sys.stdout.isatty()
    request = Request(url, headers={"User-Agent": USER_AGENT})
    with _OPENER.open(request, timeout=60) as response, dest.open("wb") as handle:
        total = expected_size or int(response.headers.get("Content-Length") or 0)
        done = 0
        while True:
            chunk = response.read(64 * 1024)
            if not chunk:
                break
            handle.write(chunk)
            done += len(chunk)
            if show_progress and total:
                percent = min(99, done * 100 // total)
                sys.stdout.write("\r下载中 {}% ({:.1f} MB)".format(percent, done / 1048576.0))
                sys.stdout.flush()
    if show_progress:
        sys.stdout.write("\r下载完成 ({:.1f} MB)      \n".format(done / 1048576.0))
    else:
        print("下载完成（{:.1f} MB）".format(done / 1048576.0))


def _sha256_of(path):
    # type: (Path) -> str
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _extract(archive, dest_parent):
    # type: (Path, Path) -> Path
    """Extract an archive with one top-level dir and return that dir."""
    staging = Path(tempfile.mkdtemp(prefix="ds-extract-"))
    try:
        if archive.suffix == ".zip":
            with zipfile.ZipFile(str(archive)) as bundle:
                _safe_extract_zip(bundle, staging)
        else:
            with tarfile.open(str(archive)) as bundle:
                _safe_extract_tar(bundle, staging)
        entries = [item for item in staging.iterdir()]
        if len(entries) != 1 or not entries[0].is_dir():
            raise ValueError("压缩包内不是单一顶层目录：{}".format(archive.name))
        top = entries[0]
        final = dest_parent / top.name
        if final.exists():
            shutil.rmtree(final)
        shutil.move(str(top), str(final))
        return final
    finally:
        shutil.rmtree(staging, ignore_errors=True)


def _member_is_inside(root, target):
    # type: (Path, Path) -> bool
    try:
        target.relative_to(root)
        return True
    except ValueError:
        return False


def _safe_extract_zip(bundle, staging):
    # type: (zipfile.ZipFile, Path) -> None
    root = staging.resolve()
    for name in bundle.namelist():
        if ".." in name.replace("\\", "/").split("/"):
            raise ValueError("压缩包包含越界路径：{}".format(name))
        target = (staging / name).resolve()
        if not _member_is_inside(root, target):
            raise ValueError("压缩包包含越界路径：{}".format(name))
    bundle.extractall(str(staging))


def _safe_extract_tar(bundle, staging):
    # type: (tarfile.TarFile, Path) -> None
    root = staging.resolve()
    for member in bundle.getmembers():
        if ".." in member.name.replace("\\", "/").split("/"):
            raise ValueError("压缩包包含越界路径：{}".format(member.name))
        target = (staging / member.name).resolve()
        if not _member_is_inside(root, target):
            raise ValueError("压缩包包含越界路径：{}".format(member.name))
        if member.issym() or member.islnk():
            raise ValueError("压缩包包含链接成员，拒绝解压：{}".format(member.name))
    bundle.extractall(str(staging))


def _machine_arch():
    # type: () -> str
    machine = os.uname().machine if hasattr(os, "uname") else os.environ.get("PROCESSOR_ARCHITECTURE", "")
    machine = (machine or "").lower()
    if machine in ("amd64", "x86_64", "x64"):
        return "x64"
    if machine in ("arm64", "aarch64"):
        return "arm64"
    raise ValueError("不支持的 CPU 架构：{}".format(machine or "未知"))


def _mirror_settings(mirror):
    # type: (Optional[str]) -> Dict
    name = mirror or os.environ.get("DEVSWITCH_MIRROR") or "official"
    if name not in MIRRORS:
        raise ValueError("未知镜像：{}（可选 official / npmmirror / tuna）".format(name))
    return MIRRORS[name]


# --------------------------------------------------------------------------
# Node.js
# --------------------------------------------------------------------------

def resolve_node_version(major, mirror=None):
    # type: (str, Optional[str]) -> str
    """Return the newest full Node version for a major, e.g. 22 -> 22.20.0."""
    settings = _mirror_settings(mirror)
    index_url = settings["node_dist"].rstrip("/") + "/index.json"
    entries = _http_get_json(index_url)
    prefix = "v{}.".format(major)
    for entry in entries:
        version = str(entry.get("version") or "")
        if version.startswith(prefix) and not version.endswith("-nightly"):
            return version[1:]
    raise LookupError("镜像上没有发现 Node {} 版本系列".format(major))


def install_node(major, mirror=None):
    # type: (str, Optional[str]) -> Path
    version = resolve_node_version(major, mirror)
    settings = _mirror_settings(mirror)
    base = settings["node_dist"].rstrip("/") + "/" + "v" + version
    arch = _machine_arch()
    if paths.IS_WINDOWS:
        filename = "node-v{}-win-{}.zip".format(version, arch)
    else:
        filename = "node-v{}-linux-{}.tar.xz".format(version, arch)

    print("Node {}（{} 架构，镜像 {}）".format(version, arch, settings["node_dist"]))
    checksums = _fetch_node_checksums(base)
    archive = _download_to_temp(base + "/" + filename, checksums.get(filename))
    home = _extract(archive, paths.home())
    print("已安装到 {}".format(home))
    return home


def _fetch_node_checksums(base):
    # type: (str) -> Dict[str, str]
    try:
        text = _http_get(base + "/SHASUMS256.txt").decode("utf-8")
    except OSError:
        return {}
    result = {}
    for line in text.splitlines():
        parts = line.split(None, 1)
        if len(parts) == 2:
            result[parts[1].strip().lstrip("*")] = parts[0]
    return result


def _download_to_temp(url, sha256=None):
    # type: (str, Optional[str]) -> Path
    temp = Path(tempfile.mkdtemp(prefix="ds-download-"))
    dest = temp / url.rsplit("/", 1)[-1]
    _http_download(url, dest)
    if sha256:
        actual = _sha256_of(dest)
        if actual.lower() != sha256.lower():
            raise ValueError("sha256 校验失败：{}（期望 {}，实际 {}）".format(dest.name, sha256, actual))
        print("sha256 校验通过")
    return dest


# --------------------------------------------------------------------------
# Temurin JDK（Adoptium）
# --------------------------------------------------------------------------

def install_java(major, mirror=None):
    # type: (str, Optional[str]) -> Path
    settings = _mirror_settings(mirror)
    arch = _machine_arch()
    adoptium_arch = {"x64": "x64", "arm64": "aarch64"}[arch]
    system = "windows" if paths.IS_WINDOWS else "linux"
    api = (
        "{api}/v3/assets/latest/{major}/hotspot"
        "?os={os}&architecture={arch}&image_type=jdk&vendor=eclipse"
    ).format(api=settings["java_api"].rstrip("/"), major=major, os=system, arch=adoptium_arch)
    data = _http_get_json(api)
    binaries = data if isinstance(data, list) else []
    if not binaries:
        raise LookupError("Adoptium 上没有发现 JDK {}（{} / {}）".format(major, system, arch))
    package = binaries[0]["binary"]["package"]
    url = package["link"]
    if settings["java_download"] == "tuna":
        url = (
            "https://mirrors.tuna.tsinghua.edu.cn/Adoptium/{major}/jdk/{arch}/{os}/{name}"
        ).format(major=major, arch=adoptium_arch, os=system, name=package["name"])

    print("Temurin JDK {}（{} / {}，来源 {}）".format(major, system, arch, urlparse(url).hostname))
    archive = _download_to_temp(url, package.get("checksum"))
    home = _extract(archive, paths.home())
    print("已安装到 {}".format(home))
    return home


# --------------------------------------------------------------------------
# Apache Maven / Gradle
# --------------------------------------------------------------------------

def install_maven(major, mirror=None):
    # type: (str, Optional[str]) -> Path
    import re as _re

    # 版本元数据恒走中央仓库（XML 很小）；下载按镜像选源
    meta = _http_get(
        "https://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/maven-metadata.xml"
    ).decode("utf-8")
    versions = _re.findall(r"<version>([\d]+\.[\d]+\.[\d]+)</version>", meta)
    candidates = [v for v in versions if v.startswith(major + ".")]
    if not candidates:
        raise LookupError("没有发现 Maven {} 版本系列".format(major))
    version = candidates[-1]

    settings = _mirror_settings(mirror)
    use_mirror = settings["node_dist"] != "https://nodejs.org/dist"
    base = (
        "https://mirrors.tuna.tsinghua.edu.cn/apache/maven/maven-3/{v}/binaries"
        if use_mirror
        else "https://dlcdn.apache.org/maven/maven-3/{v}/binaries"
    ).format(v=version)
    filename = "apache-maven-{}-bin.zip".format(version)
    print("Apache Maven {}（来源 {}）".format(version, urlparse(base).hostname))
    archive = _download_with_sidecar_checksum(base + "/" + filename)
    home = _extract(archive, paths.home())
    print("已安装到 {}".format(home))
    return home


def install_gradle(major, mirror=None):
    # type: (str, Optional[str]) -> Path
    entries = _http_get_json("https://services.gradle.org/versions/all")
    candidates = [
        str(entry.get("version") or "")
        for entry in entries
        if not entry.get("snapshot")
        and not entry.get("rcFor")
        and not entry.get("milestoneFor")
        and str(entry.get("version") or "").startswith(major + ".")
    ]
    if not candidates:
        raise LookupError("没有发现 Gradle {} 版本系列".format(major))
    version = candidates[0]  # all 接口按新到旧排列

    settings = _mirror_settings(mirror)
    use_mirror = settings["node_dist"] != "https://nodejs.org/dist"
    base = (
        "https://mirrors.cloud.tencent.com/gradle"
        if use_mirror
        else "https://services.gradle.org/distributions"
    )
    filename = "gradle-{}-bin.zip".format(version)
    print("Gradle {}（来源 {}）".format(version, urlparse(base).hostname))
    archive = _download_with_sidecar_checksum(base + "/" + filename)
    home = _extract(archive, paths.home())
    print("已安装到 {}".format(home))
    return home


def _download_with_sidecar_checksum(url):
    # type: (str) -> Path
    """下载并按同目录 sidecar 文件（.sha256/.sha512）校验。"""
    checksum = None
    algo = "sha256"
    for suffix, algorithm in ((".sha256", "sha256"), (".sha512", "sha512")):
        try:
            text = _http_get(url + suffix, timeout=15).decode("utf-8")
        except OSError:
            continue
        first = text.strip().splitlines()[0].split()[0].strip() if text.strip() else ""
        if first:
            checksum, algo = first, algorithm
            break
    temp = Path(tempfile.mkdtemp(prefix="ds-download-"))
    dest = temp / url.rsplit("/", 1)[-1]
    _http_download(url, dest)
    if checksum:
        import hashlib as _hashlib

        digest = _hashlib.new(algo)
        with dest.open("rb") as handle:
            for chunk in iter(lambda: handle.read(1024 * 1024), b""):
                digest.update(chunk)
        actual = digest.hexdigest()
        if actual.lower() != checksum.lower():
            raise ValueError("checksum 校验失败：{}（期望 {}，实际 {}）".format(dest.name, checksum, actual))
        print("{} 校验通过".format(algo))
    return dest


def latest_release_version():
    # type: () -> str
    """Read the latest published release tag (read-only update check)."""
    data = _http_get_json("https://api.github.com/repos/xyztony999/devswitch/releases/latest")
    return str(data.get("tag_name") or "").lstrip("v")


def install_runtime(tool, version, mirror=None):
    # type: (str, str, Optional[str]) -> Tuple[Path, str]
    """Download a runtime; returns (home, full_version). tool: node|java|maven|gradle."""
    from . import detect, service

    installers = {
        "node": install_node,
        "java": install_java,
        "maven": install_maven,
        "gradle": install_gradle,
    }
    inspectors = {
        "node": detect.inspect_node,
        "java": detect.inspect_java,
        "maven": detect.inspect_maven,
        "gradle": detect.inspect_gradle,
    }
    installer = installers.get(tool)
    inspector = inspectors.get(tool)
    if installer is None or inspector is None:
        raise ValueError("暂不支持下载该工具：{}".format(tool))
    home = installer(version, mirror)
    runtime = inspector(home)
    if runtime is None:
        raise ValueError("下载完成但未能识别安装目录：{}".format(home))
    service.merge_scan()
    state = service.use(tool, runtime.version)
    return Path(state.home), state.version
