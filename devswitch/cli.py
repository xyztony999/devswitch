# -*- coding: utf-8 -*-
from __future__ import print_function, unicode_literals

import argparse
import json
import os
import sys
from typing import List

from . import __app_name__, __version__, apply, paths, service
from .models import TOOLS
from .store import load_state


def _print_table(rows, headers):
    # type: (List[List[str]], List[str]) -> None
    widths = [len(h) for h in headers]
    for row in rows:
        for i, cell in enumerate(row):
            widths[i] = max(widths[i], len(cell))
    fmt = "  ".join("{:<" + str(w) + "}" for w in widths)
    print(fmt.format(*headers))
    print(fmt.format(*["-" * w for w in widths]))
    for row in rows:
        print(fmt.format(*row))


def cmd_scan(_args):
    state = service.merge_scan()
    service.ensure_applied(state)
    count = len(state.runtimes)
    print("已扫描到 {} 个运行时。".format(count))
    if count == 0:
        print("本机没有发现 Node 或 Java。安装后再扫描，或：devswitch import node|java <目录>")
        return 0
    cmd_list(argparse.Namespace(json=False, tool=None))
    return 0


def cmd_list(args):
    state = load_state()
    if not state.runtimes:
        state = service.merge_scan()
    items = state.runtimes
    if args.tool:
        items = state.for_tool(args.tool)
    if getattr(args, "json", False):
        print(json.dumps([item.to_dict() for item in items], ensure_ascii=False, indent=2))
        return 0
    if not items:
        if args.tool:
            print(
                "还没有发现 {}。安装后执行 devswitch scan，或：devswitch import {} <目录>".format(
                    args.tool, args.tool
                )
            )
        else:
            print("还没有发现运行时。安装 Node / Java 后执行：devswitch scan")
        return 0
    rows = []
    for item in items:
        current = state.current.get(item.tool) or ""
        mark = "*" if item.home.rstrip("/") == current.rstrip("/") else " "
        rows.append(
            [mark, item.tool, item.version, item.label, item.source, item.home]
        )
    _print_table(rows, ["", "工具", "版本", "名称", "来源", "目录"])
    print("\n* 表示当前选中。切换：devswitch use node 22   或   devswitch use java 17")
    return 0


def cmd_current(_args):
    state = load_state()
    for tool in TOOLS:
        runtime = state.current_runtime(tool)
        if runtime:
            print("{}  {}  ({})".format(tool, runtime.version, runtime.home))
        else:
            print("{}  (未选择)".format(tool))
    return 0


def cmd_use(args):
    try:
        runtime = service.use(args.tool, args.version)
    except LookupError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    print("已切换 {} -> {} ({})".format(args.tool, runtime.version, runtime.home))
    if paths.IS_WINDOWS:
        print("已打开的终端里 node/java 已立即生效；JAVA_HOME 和新终端请重开一次 PowerShell / Git Bash。")
    else:
        print("新开终端会带上 JAVA_HOME；当前终端里 node/java 命令已通过 ~/.local/bin 立即生效。")
    return 0


def cmd_import(args):
    try:
        runtime = service.import_path(args.tool, args.path)
    except (LookupError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    print("已导入 {} {} ({})".format(runtime.tool, runtime.version, runtime.home))
    return 0


def cmd_which(args):
    path = service.which_binary(args.name)
    if not path:
        print("未找到 {}".format(args.name), file=sys.stderr)
        return 1
    print(path)
    return 0


def cmd_doctor(args):
    if args.fix:
        for line in service.doctor_fix():
            print("· " + line)
        print()
    issues = service.doctor_issues()
    if not issues:
        print("看起来正常。当前：")
        return cmd_current(args)
    levels = {"error": "错误", "warn": "注意", "info": "提示"}
    for level, _code, message in issues:
        print("[{}] {}".format(levels.get(level, level), message))
    if not args.fix and any(item[0] != "info" for item in issues):
        print("\n可执行：devswitch doctor --fix")
    return 0 if not any(item[0] == "error" for item in issues) else 1


def cmd_hook(args):
    if paths.IS_WINDOWS:
        targets = paths.hook_targets()
        if args.action == "status":
            for target in targets:
                print(
                    str(target).ljust(50),
                    "已安装" if apply.hook_installed(target) else "未安装",
                )
            from . import winenv

            local_bin = str(paths.local_bin())
            print(
                "用户 PATH（注册表）".ljust(50),
                "已包含" if any(
                    os.path.normcase(item) == os.path.normcase(local_bin)
                    for item in winenv.user_path_entries()
                ) else "未包含",
            )
            return 0
        apply.apply_hooks(load_state())
        print("已写入 PowerShell profile、Git Bash 钩子，并把 shim 目录加入用户 PATH。")
        return 0
    if args.action == "status":
        print("bashrc  ", "已安装" if apply.hook_installed(paths.bashrc()) else "未安装")
        print("profile ", "已安装" if apply.hook_installed(paths.profile()) else "未安装")
        return 0
    apply.apply_hooks(load_state())
    print("已写入 ~/.bashrc 和 ~/.profile")
    return 0


def cmd_gui(_args):
    if paths.IS_WINDOWS:
        from .gui_win import run_gui

        return run_gui()
    from .gui import run_gui

    return run_gui()


def cmd_setup(args):
    from . import installer

    return installer.run_install(
        skip_gui_deps=getattr(args, "skip_gui_deps", False),
        in_place=getattr(args, "in_place", False),
    )


def cmd_uninstall(args):
    from . import installer

    return installer.run_uninstall(keep_app_files=getattr(args, "keep_app_files", False))


def cmd_install(args):
    from . import downloader

    try:
        home, version = downloader.install_runtime(args.tool, args.version, getattr(args, "mirror", None))
    except (LookupError, ValueError, OSError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    print("已切换 {} -> {} ({})".format(args.tool, version, home))
    print("当前终端里命令已立即生效；JAVA_HOME 对新开终端生效。")
    return 0


def cmd_export(args):
    path = service.export_versions(args.file)
    print("已导出当前版本选择到 {}".format(path))
    return 0


def cmd_apply(args):
    try:
        results = service.apply_versions(args.file, install_missing=args.install_missing,
                                         mirror=getattr(args, "mirror", None))
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    failed = False
    for tool, version, outcome in results:
        print("  {}  {}  {}".format(outcome, tool, version))
        if outcome.startswith(("missing", "failed", "unknown")):
            failed = True
    if failed:
        print("\n有未对齐的版本：missing-version 可先 devswitch scan，或用 --install-missing 自动下载。")
        return 1
    print("团队版本已对齐。")
    return 0


BASH_COMPLETION = """\
_devswitch_complete() {
  local cur cmds tools
  cur="${COMP_WORDS[COMP_CWORD]}"
  cmds="scan list current use import install setup uninstall export apply which doctor hook completion update gui"
  tools="node java maven gradle"
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=($(compgen -W "$cmds" -- "$cur"))
  else
    case "${COMP_WORDS[1]}" in
      use|import|install)
        COMPREPLY=($(compgen -W "$tools" -- "$cur")) ;;
    esac
  fi
}
complete -F _devswitch_complete devswitch
"""

ZSH_COMPLETION = """\
#compdef devswitch
_devswitch() {
  local -a cmds tools
  cmds=(scan list current use import install setup uninstall export apply which doctor hook completion update gui)
  tools=(node java maven gradle)
  if (( CURRENT == 2 )); then
    _describe 'command' cmds
  else
    case $words[2] in
      use|import|install) _describe 'tool' tools ;;
    esac
  fi
}
_compdef _devswitch devswitch
"""

PWSH_COMPLETION = """\
Register-ArgumentCompleter -CommandName devswitch -ScriptBlock {
  param($wordToComplete, $commandAst, $cursorPosition)
  $cmds = 'scan','list','current','use','import','install','setup','uninstall',
          'export','apply','which','doctor','hook','completion','update','gui'
  $tools = 'node','java','maven','gradle'
  $prior = $commandAst.CommandElements | Select-Object -Skip 1 -First 1
  if ($null -eq $prior -or $prior -is [System.Management.Automation.Language.CommandAst]) {
    $cmds | Where-Object { $_ -like "$wordToComplete*" }
  } elseif ($prior.Value -in @('use','import','install')) {
    $tools | Where-Object { $_ -like "$wordToComplete*" }
  }
}
"""


def cmd_completion(args):
    scripts = {"bash": BASH_COMPLETION, "zsh": ZSH_COMPLETION, "powershell": PWSH_COMPLETION}
    sys.stdout.write(scripts[args.shell])
    return 0


def cmd_update(_args):
    from . import downloader

    try:
        latest = downloader.latest_release_version()
    except (LookupError, ValueError, OSError) as exc:
        print("检查更新失败：{}".format(exc), file=sys.stderr)
        return 1
    from . import __version__

    if latest == __version__:
        print("已是最新版本（{}）。".format(__version__))
    else:
        print("发现新版本：{}（当前 {}）。".format(latest, __version__))
        print("下载：https://github.com/xyztony999/devswitch/releases/latest")
    return 0


def build_parser():
    parser = argparse.ArgumentParser(
        prog="devswitch",
        description="{} — 管理并切换 Node / npm / Java 版本（Linux / Windows）。".format(
            __app_name__
        ),
    )
    parser.add_argument("--version", action="version", version="{} {}".format(__app_name__, __version__))
    sub = parser.add_subparsers(dest="command")

    p_scan = sub.add_parser("scan", help="扫描本机已安装的 Node / Java")
    p_scan.set_defaults(func=cmd_scan)

    p_list = sub.add_parser("list", help="列出已发现的版本")
    p_list.add_argument("tool", nargs="?", choices=TOOLS)
    p_list.add_argument("--json", action="store_true")
    p_list.set_defaults(func=cmd_list)

    p_current = sub.add_parser("current", help="显示当前选中的版本")
    p_current.set_defaults(func=cmd_current)

    p_use = sub.add_parser("use", help="切换版本，例如：devswitch use node 22")
    p_use.add_argument("tool", choices=TOOLS)
    p_use.add_argument("version")
    p_use.set_defaults(func=cmd_use)

    p_import = sub.add_parser("import", help="导入一个本地安装目录")
    p_import.add_argument("tool", choices=TOOLS)
    p_import.add_argument("path")
    p_import.set_defaults(func=cmd_import)

    p_which = sub.add_parser("which", help="打印当前实际二进制路径")
    p_which.add_argument("name")
    p_which.set_defaults(func=cmd_which)

    p_doctor = sub.add_parser("doctor", help="检查 PATH / shim / 冲突配置")
    p_doctor.add_argument("--fix", action="store_true", help="自动写入钩子并注释冲突 PATH")
    p_doctor.set_defaults(func=cmd_doctor)

    p_hook = sub.add_parser("hook", help="安装或查看 shell 钩子")
    p_hook.add_argument("action", choices=("install", "status"), nargs="?", default="install")
    p_hook.set_defaults(func=cmd_hook)

    p_gui = sub.add_parser("gui", help="打开图形界面")
    p_gui.set_defaults(func=cmd_gui)

    p_setup = sub.add_parser(
        "setup",
        help="安装/刷新 DevSwitch 自身到用户目录（推荐直接用安装包）",
    )
    p_setup.add_argument("--skip-gui-deps", action="store_true", help="跳过 pywebview/pystray/Pillow 安装")
    p_setup.add_argument("--in-place", action="store_true", help="安装向导内部使用：包已在目标目录")
    p_setup.set_defaults(func=cmd_setup)

    p_install = sub.add_parser(
        "install",
        help="下载并安装一个运行时，例如：devswitch install node 22",
    )
    p_install.add_argument("tool", choices=("node", "java", "maven", "gradle"), help="node / java / maven / gradle")
    p_install.add_argument("version", help="大版本号，如 22 / 17（自动取该系列最新）")
    p_install.add_argument(
        "--mirror", choices=("official", "npmmirror", "tuna"),
        help="下载镜像（默认 official；也可用环境变量 DEVSWITCH_MIRROR）",
    )
    p_install.set_defaults(func=cmd_install)

    p_export = sub.add_parser("export", help="导出当前版本选择到 .devswitch 文件")
    p_export.add_argument("file", nargs="?", default=".devswitch")
    p_export.set_defaults(func=cmd_export)

    p_apply = sub.add_parser("apply", help="按 .devswitch 文件对齐版本")
    p_apply.add_argument("file")
    p_apply.add_argument("--install-missing", action="store_true", help="缺少的版本自动下载")
    p_apply.add_argument("--mirror", choices=("official", "npmmirror", "tuna"))
    p_apply.set_defaults(func=cmd_apply)

    p_completion = sub.add_parser("completion", help="输出 shell 补全脚本（bash/zsh/powershell）")
    p_completion.add_argument("shell", choices=("bash", "zsh", "powershell"))
    p_completion.set_defaults(func=cmd_completion)

    p_update = sub.add_parser("update", help="检查是否有新版本（只读，不自动升级）")
    p_update.set_defaults(func=cmd_update)

    p_uninstall = sub.add_parser("uninstall", help="卸载并清理（保留 state.json 与备份）")
    p_uninstall.add_argument("--keep-app-files", action="store_true", help="卸载向导内部使用：文件由向导删除")
    p_uninstall.set_defaults(func=cmd_uninstall)

    return parser


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    parser = build_parser()
    if not argv:
        argv = ["gui"]
    args = parser.parse_args(argv)
    if not hasattr(args, "func"):
        parser.print_help()
        return 2
    return args.func(args)
