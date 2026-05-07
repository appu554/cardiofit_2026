"""Pytest setup — make the cql-toolchain package importable when tests
are run from the repo root or from this directory."""

from __future__ import annotations

import sys
from pathlib import Path

# Add the parent of cql-toolchain (i.e. shared/) to sys.path so that
# `import cql_toolchain.*` resolves. We import via the on-disk path
# (cql-toolchain/) and alias under cql_toolchain so Python's identifier
# rules work.
_THIS = Path(__file__).resolve()
_TOOLCHAIN_DIR = _THIS.parents[1]   # shared/cql-toolchain
_SHARED_DIR = _TOOLCHAIN_DIR.parent  # shared/

# Add toolchain dir directly to allow `from rule_specification_validator
# import ...` flat-imports as a backup.
sys.path.insert(0, str(_TOOLCHAIN_DIR))
sys.path.insert(0, str(_SHARED_DIR))
