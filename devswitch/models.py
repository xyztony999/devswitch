# -*- coding: utf-8 -*-
from __future__ import print_function, unicode_literals

import re
from dataclasses import asdict, dataclass, field
from typing import Any, Dict, List, Optional


TOOLS = ("node", "java", "maven", "gradle")

NODE_SHIMS = ("node", "npm", "npx", "corepack")
JAVA_SHIMS = (
    "java",
    "javac",
    "jar",
    "javadoc",
    "javap",
    "jshell",
    "keytool",
    "jps",
    "jcmd",
)
MAVEN_SHIMS = ("mvn", "mvnDebug")
GRADLE_SHIMS = ("gradle",)

TOOL_LABELS = {"node": "Node.js", "java": "Java", "maven": "Maven", "gradle": "Gradle"}


@dataclass
class Runtime:
    tool: str
    version: str
    major: str
    home: str
    binary: str
    source: str
    label: str = ""
    vendor: str = ""

    def to_dict(self):
        # type: () -> Dict[str, Any]
        return asdict(self)

    @classmethod
    def from_dict(cls, data):
        # type: (Dict[str, Any]) -> Runtime
        return cls(
            tool=data["tool"],
            version=data["version"],
            major=str(data.get("major") or ""),
            home=data["home"],
            binary=data["binary"],
            source=data.get("source") or "local",
            label=data.get("label") or "",
            vendor=data.get("vendor") or "",
        )

    def matches(self, query):
        # type: (str) -> bool
        q = (query or "").strip().lstrip("v")
        if not q:
            return False
        if self.home.rstrip("/") == q.rstrip("/"):
            return True
        if self.version == q or self.major == q:
            return True
        if self.version.startswith(q + "."):
            return True
        return False


@dataclass
class State:
    schema: int = 1
    current: Dict[str, str] = field(default_factory=lambda: {tool: "" for tool in TOOLS})
    runtimes: List[Runtime] = field(default_factory=list)

    def to_dict(self):
        return {
            "schema": self.schema,
            "current": self.current,
            "runtimes": [item.to_dict() for item in self.runtimes],
        }

    @classmethod
    def from_dict(cls, data):
        # type: (Dict[str, Any]) -> State
        data = data or {}
        runtimes = [Runtime.from_dict(item) for item in data.get("runtimes") or []]
        current = {tool: "" for tool in TOOLS}
        current.update(data.get("current") or {})
        return cls(schema=int(data.get("schema") or 1), current=current, runtimes=runtimes)

    def for_tool(self, tool):
        # type: (str) -> List[Runtime]
        items = [item for item in self.runtimes if item.tool == tool]
        items.sort(key=lambda item: _version_key(item.version), reverse=True)
        return items

    def find(self, tool, query):
        # type: (str, str) -> Optional[Runtime]
        matches = [item for item in self.for_tool(tool) if item.matches(query)]
        if not matches:
            return None
        exact = [item for item in matches if item.version == query.lstrip("v")]
        if exact:
            return exact[0]
        major = [item for item in matches if item.major == query.lstrip("v")]
        if major:
            return major[0]
        return matches[0]

    def by_home(self, tool, home):
        # type: (str, str) -> Optional[Runtime]
        home = (home or "").rstrip("/")
        for item in self.runtimes:
            if item.tool == tool and item.home.rstrip("/") == home:
                return item
        return None

    def current_runtime(self, tool):
        # type: (str) -> Optional[Runtime]
        return self.by_home(tool, self.current.get(tool) or "")

    def upsert(self, runtime):
        # type: (Runtime) -> None
        existing = self.by_home(runtime.tool, runtime.home)
        if existing:
            existing.version = runtime.version
            existing.major = runtime.major
            existing.binary = runtime.binary
            existing.source = runtime.source
            existing.label = runtime.label or existing.label
            existing.vendor = runtime.vendor or existing.vendor
        else:
            self.runtimes.append(runtime)


def parse_node_version(text):
    # type: (str) -> str
    text = (text or "").strip()
    match = re.search(r"v?(\d+\.\d+\.\d+)", text)
    return match.group(1) if match else text.lstrip("v")


def parse_java_version(text):
    # type: (str) -> str
    text = text or ""
    match = re.search(r'version\s+"([^"]+)"', text)
    if match:
        return match.group(1)
    match = re.search(r"(\d+\.\d+\.\d+)", text)
    return match.group(1) if match else ""


def major_of(tool, version):
    # type: (str, str) -> str
    version = (version or "").lstrip("v")
    if tool == "java" and version.startswith("1."):
        parts = version.split(".")
        if len(parts) >= 2:
            return parts[1]
    return version.split(".")[0] if version else ""


def node_label(version):
    # type: (str) -> str
    major = major_of("node", version)
    lts = {"18": "LTS", "20": "LTS", "22": "LTS", "24": "LTS"}
    tag = lts.get(major)
    if tag:
        return "Node.js {} {}".format(version, tag)
    return "Node.js {}".format(version)


def java_label(version, vendor=""):
    # type: (str, str) -> str
    major = major_of("java", version)
    vendor = vendor or "OpenJDK"
    return "{} {}".format(vendor, major or version)


def _version_key(version):
    # type: (str) -> tuple
    version = (version or "").lstrip("v")
    if version.startswith("1.") and version[2:3].isdigit():
        version = version[2:]
    parts = []
    for token in re.split(r"[^0-9]+", version):
        if token:
            parts.append(int(token))
    return tuple(parts or [0])
