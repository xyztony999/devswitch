# -*- coding: utf-8 -*-
from __future__ import print_function, unicode_literals

import os
from pathlib import Path
from typing import List, Optional, Tuple

from . import apply, detect, paths
from .models import Runtime, State
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
    for tool in ("node", "java"):
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
    if tool == "node":
        runtime = detect.inspect_node(home)
    elif tool == "java":
        runtime = detect.inspect_java(home)
    else:
        raise ValueError("不支持的工具：{}".format(tool))
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


WINDOWS_NODE_TARGETS = {
    "node": "node.exe",
    "npm": "npm.cmd",
    "npx": "npx.cmd",
    "corepack": "corepack.cmd",
}


def which_binary(name):
    # type: (str) -> Optional[str]
    state = load_state()
    if name in ("java-home", "JAVA_HOME"):
        runtime = state.current_runtime("java")
        return runtime.home if runtime else None
    if name in ("node-home",):
        runtime = state.current_runtime("node")
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
        java = state.current_runtime("java")
        if java is not None:
            reg_home = winenv.get_user_java_home()
            if not reg_home or _norm(os.path.normpath(reg_home)) != _norm(os.path.normpath(java.home)):
                issues.append(
                    (
                        "warn",
                        "java-home-registry",
                        "用户环境变量 JAVA_HOME（注册表）与当前选中版本不一致：{}。".format(
                            reg_home or "未设置"
                        ),
                    )
                )

    env_probe = paths.env_file() if paths.IS_WINDOWS else paths.current_env_file()
    if not env_probe.exists():
        issues.append(("error", "no-env", "还没有生成切换配置，先运行 devswitch scan。"))

    shim_names = []
    if state.current_runtime("node") is not None:
        shim_names.extend(("node", "npm"))
    if state.current_runtime("java") is not None:
        shim_names.extend(("java", "javac"))
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

    labels = {"node": "Node.js", "java": "Java"}
    for tool in ("node", "java"):
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
