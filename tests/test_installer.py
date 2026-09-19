# -*- coding: utf-8 -*-
"""installer.py：启动器生成、命令行引用、安装副本判定。"""
from devswitch import installer, paths


def test_cmdline_quotes_only_spaces():
    assert installer._cmdline(["py", "-3"]) == "py -3"
    assert installer._cmdline([r"C:\Program Files\Python313\python.exe"]) == \
        r'"C:\Program Files\Python313\python.exe"'
    assert installer._cmdline(["py", "-3", "-u"]) == "py -3 -u"


def test_build_launchers_content(win):
    written = installer.build_launchers(
        win.data_dir / "app", win.local_bin, ["py", "-3"], ["pyw", "-3"]
    )
    names = {item.name for item in written}
    assert names == {"devswitch", "devswitch.cmd", "devswitch-gui", "devswitch-gui.cmd"}

    cli_cmd = (win.local_bin / "devswitch.cmd").read_bytes().decode("utf-8")
    # py launcher parses the raw command line: a quoted "-3" would break it
    assert "py -3 -m devswitch" in cli_cmd
    assert '"-3"' not in cli_cmd
    assert 'PYTHONPATH={};%PYTHONPATH%'.format(win.data_dir / "app") in cli_cmd
    assert "\r\n" in cli_cmd and "\r\r" not in cli_cmd

    gui_cmd = (win.local_bin / "devswitch-gui.cmd").read_bytes().decode("utf-8")
    assert 'start "" pyw -3 -m devswitch gui' in gui_cmd

    sh = (win.local_bin / "devswitch").read_bytes().decode("utf-8")
    assert sh.startswith("#!/bin/sh")
    assert 'exec py -3 -m devswitch "$@"' in sh
    assert "\r" not in sh

    sh_gui = (win.local_bin / "devswitch-gui").read_bytes().decode("utf-8")
    assert 'exec pyw -3 -m devswitch gui "$@"' in sh_gui


def test_build_launchers_pythonw_fallback(win):
    installer.build_launchers(
        win.data_dir / "app", win.local_bin, [r"C:\Py 313\python.exe"], None
    )
    gui_cmd = (win.local_bin / "devswitch-gui.cmd").read_bytes().decode("utf-8")
    # no pyw found: fall back to the quoted python path (spaces kept quoted)
    assert 'start "" "C:\\Py 313\\python.exe" -m devswitch gui' in gui_cmd


def test_is_installed_copy(win, monkeypatch):
    monkeypatch.setattr(paths, "package_root", lambda: win.data_dir / "app")
    assert installer.is_installed_copy()
    monkeypatch.setattr(paths, "package_root", lambda: win.tmp_path / "repo")
    assert not installer.is_installed_copy()


def test_run_install_rejects_linux(lin, monkeypatch, capsys):
    monkeypatch.setattr(installer, "IS_WINDOWS_HOST", False)
    assert installer.run_install() == 1
    assert "install.sh" in capsys.readouterr().err


def test_run_uninstall_full_cleanup(win, monkeypatch):
    # 切到 Windows 分支但不碰 os.name——patch os.name 会让 pathlib 在
    # 非 Windows 宿主上构造 WindowsPath 直接抛错。
    monkeypatch.setattr(installer, "IS_WINDOWS_HOST", True)
    from devswitch import apply, winenv
    from devswitch.store import save_state
    from tests.test_service import WIN_JAVA, WIN_NODE
    from devswitch.models import State

    calls = []
    monkeypatch.setattr(winenv, "remove_path_entry",
                        lambda target: calls.append(("rm-path", target)) or True)
    monkeypatch.setattr(winenv, "delete_user_value",
                        lambda name: calls.append(("rm-val", name)) or True)
    monkeypatch.setattr(winenv, "get_user_java_home", lambda: r"E:\jdk")

    # 造出安装态：包目录、启动器、shim、钩子、env 文件
    app = win.data_dir / "app"
    (app / "devswitch").mkdir(parents=True, exist_ok=True)
    (app / "unins000.exe").write_bytes(b"fake")
    apply.write_shims(State(current={"node": WIN_NODE.home, "java": WIN_JAVA.home},
                            runtimes=[WIN_NODE, WIN_JAVA]))
    installer.build_launchers(app, win.local_bin, ["py", "-3"], ["pyw", "-3"])
    for target in paths.hook_targets():
        apply.install_hook(target)
    apply.write_env_sh(State(current={"node": WIN_NODE.home, "java": WIN_JAVA.home},
                             runtimes=[WIN_NODE, WIN_JAVA]))
    foreign = win.local_bin / "user-tool.cmd"
    foreign.write_text("@echo off\nrem not ours\n", encoding="utf-8")

    assert installer.run_uninstall(keep_app_files=True) == 0

    # keep_app_files：包目录保留（由向导删除），启动器与 shim 清除
    assert (app / "devswitch").is_dir()
    assert not (win.local_bin / "devswitch.cmd").exists()
    assert not (win.local_bin / "node.cmd").exists()
    assert not (win.local_bin / "node").exists()
    # 非我们的文件不动
    assert foreign.exists()
    # 注册表与钩子清理
    assert ("rm-path", str(win.local_bin)) in calls
    assert ("rm-val", "JAVA_HOME") in calls
    for target in paths.hook_targets():
        assert not apply.hook_installed(target)
    assert not paths.env_file().exists()
