# -*- coding: utf-8 -*-
"""Windows 安装/卸载实现。

入口有两个：`devswitch install` 子命令（命令行兜底），以及安装向导
（devswitch.iss 编译出的 setup.exe，最终也调用这里）。Linux 一直使用 install.sh。

只做用户级操作：拷贝文件、写启动器、改用户 PATH，无需管理员。
"""
from __future__ import print_function, unicode_literals

import os
import shutil
import subprocess
import sys
from pathlib import Path
from typing import List, Optional, Tuple

from . import paths

GUI_DEPS = ("pywebview", "pystray", "Pillow")

# 宿主是否为 Windows。守卫用它而非直接读 os.name，测试可以安全地切换
# （monkeypatch os.name 会连带破坏 pathlib 的平台分派）。
IS_WINDOWS_HOST = os.name == "nt"


def _cmdline(prefix):
    # type: (List[str]) -> str
    """Join a command prefix; quote only tokens with spaces. The py launcher
    parses the raw command line and does not recognize a quoted "-3"."""
    def quote(token):
        return '"{}"'.format(token) if " " in token else token

    return " ".join(quote(item) for item in prefix)


def _run(cmd, env=None, timeout=120):
    # type: (List[str], Optional[dict], int) -> Tuple[int, str]
    proc = subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        env=env,
        timeout=timeout,
        check=False,
        creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0) if os.name == "nt" else 0,
    )
    return proc.returncode, (proc.stdout or b"").decode("utf-8", "replace")


def find_python():
    # type: () -> Tuple[List[str], str]
    """Return (args_prefix, exe_path) for a usable Python >= 3.8."""
    candidates = []  # type: List[Tuple[List[str], str]]
    py_exe = shutil.which("py")
    if py_exe:
        candidates.append((["py", "-3"], py_exe))
    exe = Path(sys.executable)
    candidates.append(([str(exe)], str(exe)))
    for prefix, exe_path in candidates:
        code, _out = _run(prefix + ["-c", "import sys; sys.exit(0 if sys.version_info >= (3, 8) else 1)"])
        if code == 0:
            return prefix, exe_path
    raise SystemExit("需要 Python 3.8+。请从 python.org 安装后重试。")


def find_pythonw(python_exe):
    # type: (str) -> Optional[List[str]]
    pyw = shutil.which("pyw")
    if pyw:
        return ["pyw", "-3"]
    guess = Path(python_exe).parent / "pythonw.exe"
    if guess.exists():
        return [str(guess)]
    return None


def build_launchers(app_dir, bin_dir, py_prefix, pyw_prefix):
    # type: (Path, Path, List[str], Optional[List[str]]) -> List[Path]
    """Write devswitch / devswitch-gui launchers (.cmd + sh twins)."""
    python_path = str(app_dir)
    cli_cmd = _cmdline(py_prefix)
    gui_cmd = _cmdline(pyw_prefix or py_prefix)

    templates = {
        "devswitch.cmd": (
            "@echo off\r\n"
            'set "PYTHONPATH={p};%PYTHONPATH%"\r\n'
            "{py} -m devswitch %*\r\n"
        ),
        "devswitch-gui.cmd": (
            "@echo off\r\n"
            'set "PYTHONPATH={p};%PYTHONPATH%"\r\n'
            'start "" {py} -m devswitch gui %*\r\n'
        ),
        "devswitch": (
            "#!/bin/sh\n"
            "# DevSwitch launcher\n"
            'export PYTHONPATH="{p};$PYTHONPATH"\n'
            'exec {py} -m devswitch "$@"\n'
        ),
        "devswitch-gui": (
            "#!/bin/sh\n"
            "# DevSwitch launcher\n"
            'export PYTHONPATH="{p};$PYTHONPATH"\n'
            'exec {py} -m devswitch gui "$@"\n'
        ),
    }
    gui_names = {"devswitch-gui", "devswitch-gui.cmd"}
    written = []
    for name, template in templates.items():
        dest = bin_dir / name
        # Path.open 透传 io.open：newline 参数在 Python 3.8+ 可用
        #（Path.write_text 的同名参数 3.10 才有）。
        with dest.open("w", encoding="utf-8", newline="") as handle:
            handle.write(template.format(p=python_path, py=gui_cmd if name in gui_names else cli_cmd))
        written.append(dest)
    return written


def is_installed_copy():
    # type: () -> bool
    return paths.package_root() == paths.data_dir() / "app"


def run_install(skip_gui_deps=False, in_place=False):
    # type: (bool, bool) -> int
    """Install DevSwitch into the user profile.

    in_place=True is used by the setup wizard (Inno Setup): the package has
    already been unpacked into %LOCALAPPDATA%\\devswitch\\app by the wizard,
    so only launchers / PATH / deps / hooks are set up here.
    """
    if not IS_WINDOWS_HOST:
        print("devswitch install 目前用于 Windows；Linux 请执行 ./install.sh", file=sys.stderr)
        return 1
    if not in_place and is_installed_copy():
        print("当前已在安装副本里运行。请用安装包，或克隆仓库后执行 python -m devswitch install。", file=sys.stderr)
        return 1

    from . import winenv

    try:
        py_prefix, python_exe = find_python()
    except SystemExit as exc:
        print(str(exc), file=sys.stderr)
        return 1
    pyw_prefix = find_pythonw(python_exe)

    app_dir = paths.data_dir() / "app"        # %LOCALAPPDATA%\devswitch\app
    bin_dir = paths.local_bin()               # %LOCALAPPDATA%\devswitch\bin
    app_dir.mkdir(parents=True, exist_ok=True)

    if in_place:
        if not (app_dir / "devswitch").is_dir():
            print("安装目录里没有 devswitch 包：{}".format(app_dir), file=sys.stderr)
            return 1
    else:
        pkg_src = paths.package_root() / "devswitch"
        pkg_dest = app_dir / "devswitch"
        if not pkg_src.is_dir():
            print("仓库里找不到 devswitch 包：{}".format(pkg_src), file=sys.stderr)
            return 1
        if pkg_dest.exists():
            shutil.rmtree(pkg_dest)
        shutil.copytree(
            pkg_src,
            pkg_dest,
            ignore=shutil.ignore_patterns("__pycache__", "*.pyc"),
        )
        print("已拷贝程序到 {}".format(pkg_dest))

    build_launchers(app_dir, bin_dir, py_prefix, pyw_prefix)
    print("已生成启动器：{}".format(bin_dir))

    if winenv.ensure_path_entry(str(bin_dir), first=True):
        print("已把 {} 加入用户 PATH（对新开终端生效）".format(bin_dir))
    else:
        print("用户 PATH 已包含 {}".format(bin_dir))

    if not skip_gui_deps:
        print("正在安装图形界面依赖（可选，失败不影响命令行）……")
        code, out = _run(
            py_prefix + ["-m", "pip", "install", "--user", "--quiet"] + list(GUI_DEPS),
            timeout=300,
        )
        if code == 0:
            print("图形界面依赖安装完成。")
        else:
            print("依赖安装失败（可稍后自行执行：{} -m pip install --user {}）".format(
                _cmdline(py_prefix), " ".join(GUI_DEPS)))
            print((out or "").strip()[-400:])

    env = dict(os.environ)
    env["PYTHONPATH"] = str(app_dir)
    for args in (["scan"], ["hook", "install"]):
        code, out = _run(py_prefix + ["-m", "devswitch"] + args, env=env)
        print((out or "").strip())
        if code != 0:
            print("步骤失败：devswitch {}".format(" ".join(args)), file=sys.stderr)

    print()
    print("DevSwitch 已安装到 {}（不依赖本次克隆的目录，克隆可删）。".format(app_dir))
    print("  命令行：devswitch list / use / doctor")
    print("  图形界面：devswitch-gui   或直接运行 devswitch")
    print()
    print("请重开一次 PowerShell / Git Bash 让 PATH 与 JAVA_HOME 生效。")
    print("若 PowerShell / Git Bash 里有手写的 Node PATH，可运行：devswitch doctor --fix")
    print("卸载：devswitch uninstall")
    return 0


def run_uninstall(keep_app_files=False):
    # type: (bool) -> int
    """Full uninstall: program, launchers, shims, PATH/JAVA_HOME registry
    entries and shell hooks. Only user data (state.json / backups) is kept.

    keep_app_files=True is used by the setup wizard's uninstaller: Inno Setup
    removes the files it installed itself, including this running copy."""
    if not IS_WINDOWS_HOST:
        print("devswitch uninstall 目前用于 Windows；Linux 请执行 ./uninstall.sh", file=sys.stderr)
        return 1

    from . import apply, winenv

    app_dir = paths.data_dir() / "app"
    bin_dir = paths.local_bin()

    if keep_app_files:
        print("程序文件由卸载向导删除。")
    elif app_dir.exists():
        shutil.rmtree(app_dir)
        print("已删除程序目录：{}".format(app_dir))
    else:
        print("程序目录不存在：{}".format(app_dir))

    removed = []
    for name in ("devswitch", "devswitch.cmd", "devswitch-gui", "devswitch-gui.cmd"):
        target = bin_dir / name
        if target.exists():
            target.unlink()
            removed.append(name)
    if removed:
        print("已删除启动器：{}".format("、".join(removed)))

    # 我们的 shim（node.cmd / node 等），不动其他文件
    shims = []
    if bin_dir.is_dir():
        for entry in bin_dir.iterdir():
            if apply._is_our_shim(entry):
                entry.unlink()
                shims.append(entry.name)
    if shims:
        print("已删除 shim：{} 等 {} 个".format(shims[0], len(shims)))

    if winenv.remove_path_entry(str(bin_dir)):
        print("已从用户 PATH 移除 {}".format(bin_dir))
    if winenv.get_user_java_home() is not None and winenv.delete_user_value("JAVA_HOME"):
        print("已移除用户环境变量 JAVA_HOME")

    hooks = []
    for target in paths.hook_targets():
        if apply.remove_hook(target):
            hooks.append(str(target))
    if hooks:
        print("已移除 shell 钩子：{}".format("、".join(hooks)))

    for generated in (paths.env_file(), paths.current_env_file()):
        if generated.exists():
            generated.unlink()

    print()
    print("已保留用户数据：{}（state.json 与备份；确认不再使用时可手动删除）".format(paths.config_dir()))
    print("已打开的终端里 node / java 将恢复为系统版本，请重开终端。")
    return 0
