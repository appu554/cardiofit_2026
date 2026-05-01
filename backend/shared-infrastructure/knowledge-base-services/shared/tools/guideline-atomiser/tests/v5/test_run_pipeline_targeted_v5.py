"""V5 Task 9: run_pipeline_targeted threads v5_bbox_provenance flag.

Verifies the orchestrator resolves the BBOX_PROVENANCE feature flag from the
loaded GuidelineProfile and forwards both the flag and the profile into
``signal_merger.merge(...)``.

Strategy: exercise the small helper ``_merge_with_v5_flag`` that
``run_pipeline_targeted.pipeline_1`` calls at every merge call site. Stubbing
the entire MonkeyOCR + multi-channel pipeline is intractable; isolating the
flag-resolution + dispatch logic into a pure helper keeps the test fast and
focused while still exercising the exact code path used by the orchestrator.
"""
from __future__ import annotations

import importlib
import sys
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest


# Add the data/ directory to sys.path so we can import run_pipeline_targeted as
# a module without triggering its __main__ block. The module-level argparse
# uses parse_known_args() and all required args have defaults, so import is
# side-effect tolerant under pytest.
_DATA_DIR = Path(__file__).resolve().parents[2] / "data"
if str(_DATA_DIR) not in sys.path:
    sys.path.insert(0, str(_DATA_DIR))


@pytest.fixture
def run_pipeline_module():
    """Import (or reimport) run_pipeline_targeted with a clean argv."""
    # Module-level code reads sys.argv via argparse. Pin to a minimal valid
    # invocation so importing succeeds in any environment.
    saved_argv = sys.argv
    sys.argv = ["run_pipeline_targeted.py", "--guideline", "kdigo"]
    try:
        if "run_pipeline_targeted" in sys.modules:
            mod = importlib.reload(sys.modules["run_pipeline_targeted"])
        else:
            mod = importlib.import_module("run_pipeline_targeted")
    finally:
        sys.argv = saved_argv
    return mod


def _profile_with_flag(value):
    """Duck-typed profile for is_v5_enabled."""
    return SimpleNamespace(v5_features={"bbox_provenance": value} if value is not None else {})


def test_merge_with_v5_flag_passes_true_when_profile_enables(run_pipeline_module):
    profile = _profile_with_flag(True)
    merger = MagicMock()
    merger.merge.return_value = []

    run_pipeline_module._merge_with_v5_flag(
        merger,
        "job-id",
        ["channel_outputs"],
        "tree",
        classifier="cls",
        profile=profile,
    )

    merger.merge.assert_called_once()
    _, kwargs = merger.merge.call_args
    assert kwargs["v5_bbox_provenance"] is True
    assert kwargs["profile"] is profile


def test_pipeline_1_call_sites_use_helper(run_pipeline_module):
    """All signal_merger.merge() invocations in pipeline_1 must go through
    _merge_with_v5_flag so the v5_bbox_provenance flag is always threaded."""
    src = Path(run_pipeline_module.__file__).read_text()
    # Find pipeline_1 body bounds.
    p1_start = src.index("def pipeline_1()")
    p1_end = src.index("\ndef ", p1_start + 1)
    body = src[p1_start:p1_end]
    # No raw merger.merge( inside pipeline_1 — must use the helper.
    assert "merger.merge(" not in body, (
        "pipeline_1 still calls merger.merge() directly; route through "
        "_merge_with_v5_flag so the V5 flag is threaded."
    )
    assert "_merge_with_v5_flag(" in body


def test_merge_with_v5_flag_passes_false_when_profile_empty(run_pipeline_module):
    # V4 default: profile has no v5_features overrides → flag resolves False.
    profile = _profile_with_flag(None)
    merger = MagicMock()
    merger.merge.return_value = []

    run_pipeline_module._merge_with_v5_flag(
        merger,
        "job-id",
        ["channel_outputs"],
        "tree",
        classifier="cls",
        profile=profile,
    )

    merger.merge.assert_called_once()
    _, kwargs = merger.merge.call_args
    assert kwargs["v5_bbox_provenance"] is False
    assert kwargs["profile"] is profile
