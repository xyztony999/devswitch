# -*- coding: utf-8 -*-
from __future__ import print_function, unicode_literals

import os
from pathlib import Path
from typing import List, Optional, Tuple

from . import apply, detect, paths
from .models import Runtime, State, TOOLS, TOOL_LABELS
from .store import load_state, save_state


def merge_scan(state=None):
    # type: (Optional[State]) -> State
    state = state or load_state()
    scanned = detect.scan_runtimes()
    seen_homes = set()
    merged = []
    for runtime in scanned:
        key = (runtime.tool, runtime.home.rstrip("/"))
        if key in seen_homes:
            continue
        seen_homes.add(key)
        previous = state.by_home(runtime.tool, runtime.home)
        if previous and previous.source == "imported":
            runtime.source = "imported"
            runtime.label = previous.label or runtime.label
        merged.append(runtime)
    for runtime in state.runtimes:
        key = (runtime.tool, runtime.home.rstrip("/"))
        if key in seen_homes:
            continue
        if runtime.source == "imported" and Path(runtime.home).exists():
            refreshed = (
                detect.inspect_node(Path(runtime.home))
                if runtime.tool == "node"
                else detect.inspect_java(Path(runtime.home))
            )
            if refreshed is not None:
                refreshed.source = "imported"
                merged.append(refreshed)
                seen_homes.add(key)
    state.runtimes = merged
    for tool in TOOLS:
        current = state.current_runtime(tool)
        if current is None:
            guessed = detect.infer_active(state.runtimes, tool)
            state.current[tool] = guessed.home if guessed is not None else ""
    save_state(state)
    return state


def use(tool, query, state=None):
    # type: (str, str, Optional[State]) -> Runtime
    state = state or load_state()
    runtime = state.find(tool, query)
    if runtime is None:
        state = merge_scan(state)
        runtime = state.find(tool, query)
    if runtime is None:
        if not state.for_tool(tool):
            raise LookupError(
                "本机没有发现 {}。安装后执行 devswitch scan，或：devswitch import {} <目录>".format(
                    tool, tool
                )
            )
        raise LookupError("没有找到 {} 版本：{}".format(tool, query))
    apply.apply_switch(state, tool, runtime)
    apply.apply_hooks(state)
    return runtime


def import_path(tool, path):
    # type: (str, str) -> Runtime
    home = Path(os.path.expanduser(path)).resolve()
    inspector = {
        "node": detect.inspect_node,
        "java": detect.inspect_java,
        "maven": detect.inspect_maven,
        "gradle": detect.inspect_gradle,
    }.get(tool)
    if inspector is None:
        raise ValueError("不支持的工具：{}".format(tool))
    runtime = inspector(home)
    if runtime is None:
        raise LookupError("目录里没有可用的 {}：{}".format(tool, home))
    runtime.source = "imported"
    state = load_state()
    state.upsert(runtime)
    save_state(state)
    if state.current_runtime(tool) is None:
        apply.apply_switch(state, tool, runtime)
        apply.apply_hooks(state)
    return runtime


def ensure_applied(state=None):
    # type: (Optional[State]) -> State
    state = state or load_state()
    apply.apply_all(state)
    return state


def export_versions(path):
    # type: (str) -> str
    """Write the selected versions to a shareable .devswitch file."""
    import json

    state = load_state()
    tools = {}
    for tool in TOOLS:
        runtime = state.current_runtime(tool)
        if runtime is not None:
            tools[tool] = runtime.version
    payload = {"devswitch": 1, "tools": tools}
    Path(path).write_text(
        json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    return str(Path(path))


def apply_versions(path, install_missing=False, mirror=None):
    # type: (str, bool, Optional[str]) -> List[Tuple[str, str, str]]
    """Apply a .devswitch file: switch each tool to the recorded version.
    Returns [(tool, version, result)] with result in
    switched / installed / missing-version / unknown-tool / failed: …"""
    import json

    try:
        payload = json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        raise ValueError("无法读取 {}: {}".format(path, exc))
    results = []
    for tool, version in ((payload or {}).get("tools") or {}).items():
        if tool not in TOOLS:
            results.append((tool, version, "unknown-tool"))
            continue
        try:
            use(tool, version)
            results.append((tool, version, "switched"))
        except LookupError:
            if not install_missing:
                results.append((tool, version, "missing-version"))
                continue
            try:
                from . import downloader

                downloader.install_runtime(tool, version.split(".")[0], mirror)
                results.append((tool, version, "installed"))
            except (LookupError, ValueError, OSError) as exc:
                results.append((tool, version, "failed: {}".format(exc)))
    return results


WINDOWS_NODE_TARGETS = {
    "node": "node.exe",
    "npm": "npm.cmd",
    "npx": "npx.cmd",
    "corepack": "corepack.cmd",
}
WINDOWS_BUILDTOOL_TARGETS = {
    "mvn": "mvn.cmd",
    "mvnDebug": "mvnDebug.cmd",
    "gradle": "gradle.bat",
}


def which_binary(name):
    # type: (str) -> Optional[str]
    state = load_state()
    # <tool>-home / <TOOL>_HOME / 大写工具名 → home 目录；裸命令名走下面的二进制分支
    for tool in TOOLS:
        if name in (tool + "-home", tool.upper() + "_HOME", tool.upper()):
            runtime = state.current_runtime(tool)
            return runtime.home if runtime else None
    if name in ("node", "npm", "npx", "corepack"):
        runtime = state.current_runtime("node")
        if runtime is None:
            return None
        if paths.IS_WINDOWS:
            candidate = Path(runtime.home) / WINDOWS_NODE_TARGETS[name]
        else:
            candidate = Path(runtime.home) / "bin" / name
        return str(candidate) if candidate.exists() else None
    if name in ("mvn", "mvnDebug", "gradle"):
        tool = "maven" if name.startswith("mvn") else "gradle"
        runtime = state.current_runtime(tool)
        if runtime is None:
            return None
        if paths.IS_WINDOWS:
            candidate = Path(runtime.home) / "bin" / WINDOWS_BUILDTOOL_TARGETS[name]
        else:
            candidate = Path(runtime.home) / "bin" / name
        return str(candidate) if candidate.exists() else None
    runtime = state.current_runtime("java")
    if runtime is None:
        return None
    exe = name + ".exe" if paths.IS_WINDOWS else name
    for rel in ("bin/" + exe, "jre/bin/" + exe):
        candidate = Path(runtime.home) / rel
        if candidate.exists():
            return str(candidate)
    return None


def _norm(text):
    # type: (str) -> str
    return os.path.normcase(text) if paths.IS_WINDOWS else text


def doctor_issues():
    # type: () -> List[Tuple[str, str, str]]
    """Return list of (level, code, message)."""
    issues = []
    state = load_state()
    local_bin = str(paths.local_bin())
    path_env = os.environ.get("PATH") or ""
    parts = [item for item in path_env.split(os.pathsep) if item]
    norm_parts = [_norm(item) for item in parts]

    if _norm(local_bin) not in norm_parts:
        if paths.IS_WINDOWS:
            issues.append(
                (
                    "warn",
                    "path-missing",
                    "{} 不在本次 PATH 里，终端可能仍走系统版本。".format(local_bin),
                )
            )
        else:
            issues.append(
                (
                    "warn",
                    "path-missing",
                    "~/.local/bin 不在 PATH 里，图形程序和部分终端可能仍走系统版本。",
                )
            )
    elif norm_parts and norm_parts[0] != _norm(local_bin):
        keywords = ("node", "jvm", "jdk", "java") if paths.IS_WINDOWS else ("node", "jvm")
        ahead = [
            item
            for index, item in enumerate(parts[: norm_parts.index(_norm(local_bin))])
            if any(word in _norm(item) for word in keywords)
        ]
        if ahead:
            if paths.IS_WINDOWS:
                issues.append(
                    (
                        "warn",
                        "path-order",
                        "PATH 里有更靠前的 Node/Java 目录（常见于机器级安装），可能抢在 DevSwitch shim 前面：{}".format(
                            ", ".join(ahead)
                        ),
                    )
                )
            else:
                issues.append(
                    (
                        "warn",
                        "path-order",
                        "PATH 里有更靠前的 Node/Java 目录，可能抢在 DevSwitch shim 前面：{}".format(
                            ", ".join(ahead)
                        ),
                    )
                )

    if paths.IS_WINDOWS:
        from . import winenv

        user_entries = [_norm(item) for item in winenv.user_path_entries()]
        if _norm(local_bin) not in user_entries:
            issues.append(
                (
                    "warn",
                    "user-path-missing",
                    "{} 不在用户 PATH（注册表）里，新开的终端会找不到 shim。可运行 devswitch hook install。".format(
                        local_bin
                    ),
                )
            )
        for tool, var in apply.TOOL_HOME_VARS.items():
            runtime = state.current_runtime(tool)
            if runtime is None:
                continue
            reg_home = winenv.get_user_value(var)
            reg_home = reg_home[0] if reg_home else None
            if not reg_home or _norm(os.path.normpath(reg_home)) != _norm(os.path.normpath(runtime.home)):
                issues.append(
                    (
                        "warn",
                        tool + "-home-registry",
                        "用户环境变量 {}（注册表）与当前选中版本不一致：{}。".format(
                            var, reg_home or "未设置"
                        ),
                    )
                )

    env_probe = paths.env_file() if paths.IS_WINDOWS else paths.current_env_file()
    if not env_probe.exists():
        issues.append(("error", "no-env", "还没有生成切换配置，先运行 devswitch scan。"))

    probe_shims = {"node": ("node", "npm"), "java": ("java", "javac"),
                   "maven": ("mvn",), "gradle": ("gradle",)}
    shim_names = []
    for tool, names in probe_shims.items():
        if state.current_runtime(tool) is not None:
            shim_names.extend(names)
    for name in shim_names:
        shim = paths.local_bin() / paths.shim_filename(name)
        if not shim.exists():
            issues.append(("warn", "no-shim", "缺少 shim：{}".format(shim)))
        elif shim.is_symlink() or "DevSwitch shim" not in shim.read_text(encoding="utf-8", errors="ignore"):
            issues.append(("warn", "foreign-shim", "{} 还不是 DevSwitch 接管的入口。".format(shim)))

    if paths.IS_WINDOWS:
        hook_targets = paths.hook_targets()
        missing = [str(item) for item in hook_targets if not apply.hook_installed(item)]
        if len(missing) == len(hook_targets):
            issues.append(
                (
                    "warn",
                    "no-hook",
                    "还没有写入 PowerShell / Git Bash 钩子，新开终端可能没有 JAVA_HOME，也可能抢不过机器级安装。",
                )
            )
    elif not apply.hook_installed(paths.bashrc()):
        issues.append(("warn", "no-hook", "还没有写入 ~/.bashrc 钩子，新开终端可能没有 JAVA_HOME。"))

    for file_path, line in apply.conflicting_bashrc_lines():
        issues.append(
            (
                "warn",
                "bashrc-path",
                "{} 里仍有手写 Node PATH，容易和切换打架：{}".format(file_path, line),
            )
        )

    labels = TOOL_LABELS
    for tool in TOOLS:
        if not state.for_tool(tool):
            issues.append(
                (
                    "info",
                    "no-runtime",
                    "本机没有发现 {}。安装后执行扫描，或用导入指定目录。".format(labels[tool]),
                )
            )
        elif state.current_runtime(tool) is None:
            issues.append(("info", "no-current", "尚未选择当前 {} 版本。".format(labels[tool])))

    return issues


def doctor_fix():
    # type: () -> List[str]
    actions = []
    state = merge_scan()
    apply.apply_all(state)
    if paths.IS_WINDOWS:
        actions.append("已写入 shim、env.sh、PowerShell / Git Bash 钩子和用户环境变量")
    else:
        actions.append("已写入 shim、env.sh 和 shell 钩子")
    changed = apply.comment_conflicting_path_lines(paths.bashrc())
    changed += apply.comment_conflicting_path_lines(paths.profile())
    if changed:
        actions.append("已注释冲突 PATH：" + "；".join(changed))
    return actions
