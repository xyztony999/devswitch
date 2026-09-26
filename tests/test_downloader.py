# -*- coding: utf-8 -*-
"""downloader：URL 安全校验、镜像选择、版本解析；service.export/apply。"""
import pytest

from devswitch import downloader


def test_assert_safe_url_rejects_bad_scheme():
    with pytest.raises(ValueError):
        downloader.assert_safe_url("ftp://nodejs.org/dist")
    with pytest.raises(ValueError):
        downloader.assert_safe_url("file:///etc/passwd")


def test_assert_safe_url_rejects_unknown_host():
    with pytest.raises(ValueError):
        downloader.assert_safe_url("https://evil.example.com/node.zip")


def test_assert_safe_url_rejects_localhost(monkeypatch):
    monkeypatch.setattr(downloader.socket, "getaddrinfo",
                        lambda host, *a: [(None, None, None, "", ("127.0.0.1", 0))])
    with pytest.raises(ValueError):
        downloader.assert_safe_url("https://nodejs.org/dist")


def test_assert_safe_url_rejects_private_resolution(monkeypatch):
    monkeypatch.setattr(downloader.socket, "getaddrinfo",
                        lambda host, *a: [(None, None, None, "", ("192.168.1.10", 0))])
    with pytest.raises(ValueError):
        downloader.assert_safe_url("https://api.adoptium.net/v3")


def test_assert_safe_url_accepts_public(monkeypatch):
    monkeypatch.setattr(downloader.socket, "getaddrinfo",
                        lambda host, *a: [(None, None, None, "", ("104.20.23.46", 0))])
    downloader.assert_safe_url("https://nodejs.org/dist/index.json")


def test_mirror_priority(monkeypatch):
    assert downloader._mirror_settings(None)["node_dist"] == "https://nodejs.org/dist"
    monkeypatch.setenv("DEVSWITCH_MIRROR", "npmmirror")
    assert "npmmirror" in downloader._mirror_settings(None)["node_dist"]
    assert downloader._mirror_settings("tuna")["java_download"] == "tuna"
    with pytest.raises(ValueError):
        downloader._mirror_settings("nope")


def test_resolve_node_version(monkeypatch):
    fake = [{"version": "v22.20.0"}, {"version": "v22.11.0"}, {"version": "v20.18.0"}]
    monkeypatch.setattr(downloader, "_http_get_json", lambda url, timeout=30: fake)
    assert downloader.resolve_node_version("22") == "22.20.0"
    with pytest.raises(LookupError):
        downloader.resolve_node_version("19")


def test_latest_release_version(monkeypatch):
    monkeypatch.setattr(downloader, "_http_get_json",
                        lambda url, timeout=30: {"tag_name": "v9.9.9"})
    assert downloader.latest_release_version() == "9.9.9"


def test_safe_extract_rejects_traversal(tmp_path, monkeypatch):
    import tarfile, io

    payload = io.BytesIO()
    with tarfile.open(fileobj=payload, mode="w") as bundle:
        info = tarfile.TarInfo("../evil.txt")
        data = b"boom"
        info.size = len(data)
        bundle.addfile(info, io.BytesIO(data))
    payload.seek(0)
    staging = tmp_path / "staging"
    staging.mkdir()
    with tarfile.open(fileobj=payload) as bundle:
        with pytest.raises(ValueError):
            downloader._safe_extract_tar(bundle, staging)


def test_export_apply_roundtrip(lin, tmp_path, monkeypatch):
    monkeypatch.setattr("devswitch.detect.scan_runtimes", lambda: [])
    from devswitch import service
    from devswitch.models import Runtime
    from devswitch.store import save_state

    node = Runtime(tool="node", version="22.11.0", major="22",
                   home="/opt/node-v22", binary="/opt/node-v22/bin/node", source="local")
    save_state(__import__("devswitch.models", fromlist=["State"]).State(
        current={"node": node.home, "java": "", "maven": "", "gradle": ""},
        runtimes=[node]))
    # 指向 /opt/... 的 which 存在性检查会失败：改用沙箱内可存在的目录
    real_home = lin.local_bin.parent / "node-v22"
    (real_home / "bin").mkdir(parents=True, exist_ok=True)
    node2 = Runtime(tool="node", version="22.11.0", major="22",
                    home=str(real_home), binary=str(real_home / "bin" / "node"), source="local")
    (real_home / "bin" / "node").write_text("", encoding="utf-8")
    save_state(__import__("devswitch.models", fromlist=["State"]).State(
        current={"node": node2.home, "java": "", "maven": "", "gradle": ""},
        runtimes=[node2]))

    env_file = tmp_path / ".devswitch"
    service.export_versions(str(env_file))
    text = env_file.read_text(encoding="utf-8")
    assert '"node": "22.11.0"' in text

    results = service.apply_versions(str(env_file))
    assert results == [("node", "22.11.0", "switched")]
