# -*- coding: utf-8 -*-
from __future__ import print_function, unicode_literals

import json
from typing import Optional

from .models import State
from . import paths


def load_state():
    # type: () -> State
    path = paths.state_file()
    if not path.exists():
        return State()
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return State()
    return State.from_dict(data)


def save_state(state):
    # type: (State) -> None
    path = paths.state_file()
    payload = json.dumps(state.to_dict(), ensure_ascii=False, indent=2)
    path.write_text(payload + "\n", encoding="utf-8")


def remember_current(state, tool, runtime):
    # type: (State, str, Optional[object]) -> None
    state.current[tool] = runtime.home if runtime is not None else ""
    save_state(state)
