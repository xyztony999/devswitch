# -*- coding: utf-8 -*-
from __future__ import print_function, unicode_literals

import glob
import os
import re
import shutil
import subprocess
from pathlib import Path
from typing import Iterable, List, Optional, Set

from . import paths
from .models import Runtime, java_label, major_of, node_label, parse_java_version, parse_node_version


def _run(cmd, timeout=4):
    # type: (List[str], int) -> str
    kwargs = {}
    if paths.IS_WINDOWS:
        # Keep console-less invocations (GUI, scanning) from flashing a window.
        kwargs["creationflags"] = getattr(subprocess, "CREATE_NO_WINDOW", 0x08000000)
    try:
        proc = subprocess.run(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            timeout=timeout,
            check=False,
            **kwargs
        )
        out = proc.stdout or b""
        return out.decode("utf-8", "replace")
    except (OSError, subprocess.TimeoutExpired):
        return ""


def _unique_paths(values):
    # type: (Iterable[object]) -> List[Path]
    seen = set()  # type: Set[str]
    result = []
    for value in values:
        if not value:
            continue
        path = Path(os.path.expanduser(str(value)))
        try:
            resolved = str(path.resolve())
        except OSError:
            resolved = str(path)
        key = os.path.normcase(resolved) if paths.IS_WINDOWS else resolved
        if key in seen:
            continue
        seen.add(key)
        result.append(path)
    return result


def _which_all_windows(name):
    # type: (str) -> List[Path]
    found = []
    extensions = [".exe", ".cmd", ".bat"]
    for directory in os.environ.get("PATH", "").split(os.pathsep):
        directory = directory.strip('"')
        if not directory:
            continue
        base = Path(directory) / name
        for ext in extensions:
            candidate = Path(str(base) + ext)
            if candidate.is_file():
                found.append(candidate)
                break
    return found


def _which_all(name):
    # type: (str) -> List[Path]
    if paths.IS_WINDOWS:
        return _unique_paths(_which_all_windows(name))
    found = []
    which = shutil.which(name)
    if which:
        found.append(Path(which))
    try:
        proc = subprocess.run(
            ["bash", "-lc", "type -a -p {}".format(name)],
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            timeout=3,
            check=False,
        )
        for line in (proc.stdout or b"").decode("utf-8", "replace").splitlines():
            line = line.strip()
            if line:
                found.append(Path(line))
    except (OSError, subprocess.TimeoutExpired):
        pass
    return _unique_paths(found)


def _is_shim(path):
    # type: (Path) -> bool
    try:
        if not path.is_file() or path.is_symlink():
            # Our shims are regular files; old user links are symlinks.
            if path.is_file() and not path.is_symlink():
                text = path.read_text(encoding="utf-8", errors="ignore")
                return "DevSwitch shim" in text
            return False
        text = path.read_text(encoding="utf-8", errors="ignore")
        return "DevSwitch shim" in text
    except (OSError, UnicodeError):
        return False


def _node_binary_names():
    # type: () -> tuple
    return ("node.exe",) if paths.IS_WINDOWS else ("node",)


def node_home_from_binary(binary):
    # type: (Path) -> Optional[Path]
    binary = Path(binary)
    try:
        binary = binary.resolve()
    except OSError:
        return None
    if binary.name not in _node_binary_names() or not binary.is_file():
        return None
    parent = binary.parent
    if parent.name == "bin":
        return parent.parent
    return parent


def java_home_from_binary(binary):
    # type: (Path) -> Optional[Path]
    binary = Path(binary)
    try:
        binary = binary.resolve()
    except OSError:
        return None
    names = ("java.exe",) if paths.IS_WINDOWS else ("java",)
    if binary.name not in names:
        return None
    if not paths.IS_WINDOWS and not os.access(str(binary), os.X_OK):
        return None
    parent = binary.parent
    if parent.name == "bin" and parent.parent.name == "jre":
        return parent.parent.parent
    if parent.name == "bin":
        return parent.parent
    return parent


def _inspect_node_binary(binary, home):
    # type: (Path, Path) -> Optional[Runtime]
    output = _run([str(binary), "-v"])
    version = parse_node_version(output)
    if not version:
        return None
    source = "system" if _looks_system(home) else "local"
    return Runtime(
        tool="node",
        version=version,
        major=major_of("node", version),
        home=str(home),
        binary=str(binary),
        source=source,
        label=node_label(version),
        vendor="Node.js",
    )


def inspect_node(home):
    # type: (Path) -> Optional[Runtime]
    home = Path(home)
    if paths.IS_WINDOWS:
        binary = home / "node.exe"
        if not binary.is_file():
            return None
        return _inspect_node_binary(binary, home)
    binary = home / "bin" / "node"
    if not binary.is_file():
        return None
    try:
        binary = binary.resolve()
        home = binary.parent.parent
    except OSError:
        return None
    return _inspect_node_binary(binary, home)


def _looks_system(home):
    # type: (Path) -> bool
    text = str(home)
    if paths.IS_WINDOWS:
        # 自己转小写而非依赖 os.path.normcase——它在非 Windows 宿主上是
        # 恒等函数，会让这条分支在跨平台测试里大小写不匹配。
        lower = text.replace("/", "\\").lower()
        if "\\program files\\" in lower or "\\program files (x86)\\" in lower:
            return True
        if lower.endswith("\\program files") or lower.endswith("\\program files (x86)"):
            return True
        for var in ("ProgramFiles", "ProgramFiles(x86)", "ProgramW6432"):
            base = os.environ.get(var)
            if base and lower.startswith(base.replace("/", "\\").lower() + "\\"):
                return True
        windir = os.environ.get("SystemRoot") or r"C:\Windows"
        return lower.startswith(windir.replace("/", "\\").lower() + "\\")
    return text.startswith("/usr")


def inspect_java(home):
    # type: (Path) -> Optional[Runtime]
    home = Path(home)
    exe = "java.exe" if paths.IS_WINDOWS else "java"
    binary = home / "bin" / exe
    if not binary.is_file():
        binary = home / "jre" / "bin" / exe
    if not binary.is_file():
        return None
    try:
        binary = binary.resolve()
        resolved_home = java_home_from_binary(binary) or home
    except OSError:
        return None
    output = _run([str(binary), "-version"])
    version = parse_java_version(output)
    if not version:
        return None
    vendor = "OpenJDK"
    if re.search(r"\bTemurin\b", output, re.I):
        vendor = "Temurin"
    elif re.search(r"\bZulu\b", output, re.I):
        vendor = "Zulu"
    elif re.search(r"\bGraalVM\b", output, re.I):
        vendor = "GraalVM"
    elif re.search(r"\bMicrosoft\b", output, re.I):
        vendor = "Microsoft"
    elif re.search(r"\bCorretto\b", output, re.I):
        vendor = "Corretto"
    elif re.search(r"\bLiberica\b", output, re.I):
        vendor = "Liberica"
    source = "system" if _looks_system(resolved_home) else "local"
    return Runtime(
        tool="java",
        version=version,
        major=major_of("java", version),
        home=str(resolved_home),
        binary=str(binary),
        source=source,
        label=java_label(version, vendor),
        vendor=vendor,
    )


def _glob_all(patterns):
    # type: (List[str]) -> List[Path]
    found = []
    for pattern in patterns:
        for item in glob.glob(pattern):
            found.append(Path(item))
    return found


def _candidate_node_homes():
    # type: () -> List[Path]
    home = paths.home()
    if paths.IS_WINDOWS:
        program_files = [
            os.environ.get(var)
            for var in ("ProgramFiles", "ProgramFiles(x86)", "ProgramW6432")
        ]
        patterns = []
        for base in program_files:
            if base:
                patterns.append(os.path.join(base, "nodejs"))
        nvm_root = os.environ.get("NVM_HOME") or os.path.join(
            str(home), "AppData", "Roaming", "nvm"
        )
        patterns.append(os.path.join(nvm_root, "v*"))
        patterns.append(str(home / "node-v*-win-*"))
        patterns.append(str(home / "scoop" / "apps" / "nodejs" / "*"))
        patterns.append(str(home / ".local" / "share" / "node-v*-win-*"))
        found = _glob_all(patterns)
    else:
        patterns = [
            str(home / "node-v*-linux-*"),
            str(home / ".local" / "share" / "node-v*-linux-*"),
            str(home / ".nvm" / "versions" / "node" / "*"),
            "/usr/local/n/versions/node/*",
            "/opt/node*",
            "/usr/local/lib/nodejs/*",
        ]
        found = _glob_all(patterns)
        usr_node = Path("/usr/bin/node")
        if usr_node.exists():
            found.append(Path("/usr"))
    for binary in _which_all("node"):
        if _is_shim(binary):
            continue
        derived = node_home_from_binary(binary)
        if derived is not None:
            found.append(derived)
    return _unique_paths(found)


def _candidate_java_homes():
    # type: () -> List[Path]
    home = paths.home()
    found = []  # type: List[Path]
    if paths.IS_WINDOWS:
        program_files = [
            os.environ.get(var)
            for var in ("ProgramFiles", "ProgramFiles(x86)", "ProgramW6432")
        ]
        patterns = []
        for base in program_files:
            if not base:
                continue
            for vendor_dir in (
                "Java",
                "Eclipse Adoptium",
                "Microsoft",
                "Zulu",
                "Amazon Corretto",
                "BellSoft",
            ):
                patterns.append(os.path.join(base, vendor_dir, "*"))
            # Android Studio ships a bundled JetBrains Runtime.
            patterns.append(os.path.join(base, "Android", "Android Studio", "jbr"))
        patterns.append(str(home / "jdk*"))
        patterns.append(str(home / "java*"))
        patterns.append(str(home / ".jdks" / "*"))
        found = _glob_all(patterns)
    else:
        for item in glob.glob("/usr/lib/jvm/*"):
            path = Path(item)
            name = path.name
            if name in ("default-java",) or name.startswith("openjdk-"):
                continue
            found.append(path)
        for pattern in [
            str(home / "jdk*"),
            str(home / "java*"),
            str(home / ".sdkman" / "candidates" / "java" / "*"),
            "/opt/jdk*",
            "/opt/java*",
            "/usr/java/*",
        ]:
            found.extend(_glob_all([pattern]))
    env_home = os.environ.get("JAVA_HOME")
    if env_home:
        found.append(Path(env_home))
    for binary in _which_all("java"):
        if _is_shim(binary):
            continue
        derived = java_home_from_binary(binary)
        if derived is not None:
            found.append(derived)
    return _unique_paths(found)


def scan_runtimes():
    # type: () -> List[Runtime]
    found = []
    seen = set()  # type: Set[str]
    for home in _candidate_node_homes():
        runtime = inspect_node(home)
        if runtime is None:
            continue
        key = "node:" + os.path.normcase(runtime.binary) if paths.IS_WINDOWS else "node:" + runtime.binary
        if key in seen:
            continue
        seen.add(key)
        found.append(runtime)
    for home in _candidate_java_homes():
        runtime = inspect_java(home)
        if runtime is None:
            continue
        key = "java:" + os.path.normcase(runtime.binary) if paths.IS_WINDOWS else "java:" + runtime.binary
        if key in seen:
            continue
        seen.add(key)
        found.append(runtime)
    found.sort(key=lambda item: (item.tool, item.version), reverse=True)
    return found


def infer_active(runtimes, tool):
    # type: (List[Runtime], str) -> Optional[Runtime]
    binary_name = "node.exe" if (tool == "node" and paths.IS_WINDOWS) else (
        "java.exe" if (tool == "java" and paths.IS_WINDOWS) else ("node" if tool == "node" else "java")
    )
    which = shutil.which(binary_name)
    if which:
        path = Path(which)
        if not _is_shim(path):
            try:
                resolved = str(path.resolve())
            except OSError:
                resolved = str(path)
            for runtime in runtimes:
                if runtime.tool == tool and runtime.binary == resolved:
                    return runtime
    if tool == "java":
        env_home = (os.environ.get("JAVA_HOME") or "").rstrip("/").rstrip("\\")
        for runtime in runtimes:
            home = runtime.home.rstrip("/").rstrip("\\")
            if runtime.tool == tool and (
                home == env_home or (paths.IS_WINDOWS and os.path.normcase(home) == os.path.normcase(env_home))
            ):
                return runtime
    items = [item for item in runtimes if item.tool == tool]
    return items[0] if items else None
