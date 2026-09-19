# -*- coding: utf-8 -*-
from devswitch.models import (
    Runtime,
    State,
    _version_key,
    major_of,
    node_label,
    parse_java_version,
    parse_node_version,
)


def _runtime(tool="node", version="22.11.0", home="/opt/node-v22"):
    return Runtime(
        tool=tool,
        version=version,
        major=major_of(tool, version),
        home=home,
        binary=home + "/bin/" + ("node" if tool == "node" else "java"),
        source="local",
    )


def test_parse_node_version():
    assert parse_node_version("v22.11.0\n") == "22.11.0"
    assert parse_node_version("22.11.0") == "22.11.0"
    assert parse_node_version("") == ""


def test_parse_java_version():
    text = 'openjdk version "17.0.9" 2023-10-17\nOpenJDK Runtime Environment'
    assert parse_java_version(text) == "17.0.9"
    assert parse_java_version("") == ""


def test_major_of():
    assert major_of("node", "22.11.0") == "22"
    assert major_of("java", "17.0.9") == "17"
    assert major_of("java", "1.8.0_392") == "8"


def test_matches_by_major_version_and_path():
    item = _runtime(version="22.11.0", home="/opt/node-v22.11.0")
    assert item.matches("22")
    assert item.matches("v22")
    assert item.matches("22.11.0")
    assert item.matches("22.11")
    assert item.matches("/opt/node-v22.11.0")
    assert not item.matches("20")


def test_find_prefers_exact_then_major():
    state = State(
        runtimes=[
            _runtime(version="22.11.0", home="/opt/a"),
            _runtime(version="22.5.1", home="/opt/b"),
            _runtime(version="20.18.0", home="/opt/c"),
        ]
    )
    assert state.find("node", "22").home == "/opt/a"
    assert state.find("node", "22.5.1").home == "/opt/b"
    assert state.find("node", "/opt/c").home == "/opt/c"
    assert state.find("node", "99") is None


def test_version_key_orders_java_legacy():
    assert _version_key("1.8.0_392") == (8, 0, 392)
    assert _version_key("17.0.9") > _version_key("11.0.20")
    assert _version_key("22.11.0") > _version_key("9.9.9")


def test_for_tool_sorted_desc():
    state = State(
        runtimes=[
            _runtime(version="18.20.0", home="/opt/old"),
            _runtime(version="22.11.0", home="/opt/new"),
        ]
    )
    assert [item.version for item in state.for_tool("node")] == ["22.11.0", "18.20.0"]


def test_node_label_lts():
    assert "LTS" in node_label("22.11.0")
    assert "LTS" not in node_label("21.7.3")
