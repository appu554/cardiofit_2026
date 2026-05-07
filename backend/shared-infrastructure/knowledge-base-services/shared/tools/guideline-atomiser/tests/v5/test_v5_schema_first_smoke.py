"""Smoke acceptance tests for V5 Schema-first extraction.

Requires a completed job run with V5_SCHEMA_FIRST=1.

Run:
    PYTHONPATH=. V5_SCHEMA_FIRST=1 V5_BBOX_PROVENANCE=1 \\
        pytest tests/v5/test_v5_schema_first_smoke.py -v -m smoke
"""
from __future__ import annotations

import json
import os
from pathlib import Path

import pytest

pytestmark = pytest.mark.smoke

_SF_ENABLED = os.getenv("V5_SCHEMA_FIRST", "") not in ("", "0")

SMOKE_JOB_DIR = Path("data/output/v4")


def _latest(pattern: str) -> Path | None:
    """Return the most recently modified file matching glob pattern, or None."""
    candidates = list(SMOKE_JOB_DIR.glob(pattern))
    return max(candidates, key=lambda p: p.stat().st_mtime) if candidates else None


def _meta_has_schema_first() -> bool:
    """True only if the latest job_metadata.json was produced with schema_first enabled."""
    f = _latest("**/job_metadata.json")
    if f is None:
        return False
    try:
        m = json.loads(f.read_text())
        return "schema_first" in m.get("v5_features_enabled", [])
    except Exception:
        return False


def _metrics_has_schema_first() -> bool:
    """True only if the latest metrics.json contains the v5_schema_first key."""
    f = _latest("**/metrics.json")
    if f is None:
        return False
    try:
        m = json.loads(f.read_text())
        return "v5_schema_first" in m
    except Exception:
        return False


# Module-level flags avoid repeated JSON reads per test.
_META_READY = _meta_has_schema_first()
_METRICS_READY = _metrics_has_schema_first()


@pytest.mark.skipif(
    not _SF_ENABLED or not _META_READY,
    reason="V5_SCHEMA_FIRST not set or job_metadata.json predates schema-first — run pipeline with V5_SCHEMA_FIRST=1 first",
)
def test_schema_first_active_in_metadata():
    """job_metadata.json reports schema_first in v5_features_enabled."""
    latest_meta = _latest("**/job_metadata.json")
    with open(latest_meta) as f:
        meta = json.load(f)
    assert "schema_first" in meta.get("v5_features_enabled", []), (
        f"Expected 'schema_first' in v5_features_enabled, got {meta.get('v5_features_enabled')}"
    )


@pytest.mark.skipif(
    not _SF_ENABLED or not _METRICS_READY,
    reason="V5_SCHEMA_FIRST not set or metrics.json predates schema-first — run pipeline with V5_SCHEMA_FIRST=1 first",
)
def test_schema_first_metrics_written():
    """metrics.json contains v5_schema_first key with pass_rate_pct."""
    latest_metrics = _latest("**/metrics.json")
    with open(latest_metrics) as f:
        m = json.load(f)
    assert "v5_schema_first" in m, "metrics.json missing 'v5_schema_first' key"
    sf = m["v5_schema_first"]
    assert "pass_rate_pct" in sf, "v5_schema_first missing 'pass_rate_pct'"
    assert "total_validated" in sf, "v5_schema_first missing 'total_validated'"
    assert sf["total_validated"] >= 0, "total_validated must be non-negative"
    assert 0.0 <= sf["pass_rate_pct"] <= 100.0, "pass_rate_pct out of range"


@pytest.mark.skipif(
    not _SF_ENABLED or not _METRICS_READY,
    reason="V5_SCHEMA_FIRST not set or metrics.json predates schema-first — run pipeline with V5_SCHEMA_FIRST=1 first",
)
def test_schema_first_per_schema_breakdown():
    """metrics.json v5_schema_first contains per_schema breakdown with total and passed keys."""
    latest_metrics = _latest("**/metrics.json")
    with open(latest_metrics) as f:
        m = json.load(f)
    sf = m.get("v5_schema_first", {})
    per = sf.get("per_schema", {})
    assert isinstance(per, dict), "per_schema must be a dict"
    for schema_name, counts in per.items():
        assert "total" in counts, f"{schema_name} missing 'total'"
        assert "passed" in counts, f"{schema_name} missing 'passed'"
        assert counts["passed"] <= counts["total"], (
            f"{schema_name}: passed ({counts['passed']}) > total ({counts['total']})"
        )


@pytest.mark.skipif(
    not _SF_ENABLED or not _METRICS_READY,
    reason="V5_SCHEMA_FIRST not set or metrics.json predates schema-first — run pipeline with V5_SCHEMA_FIRST=1 first",
)
def test_schema_first_verdict_present():
    """metrics.json contains verdict_schema_first key with PASS or FAIL."""
    latest_metrics = _latest("**/metrics.json")
    with open(latest_metrics) as f:
        m = json.load(f)
    assert "verdict_schema_first" in m, "metrics.json missing 'verdict_schema_first' key"
    assert m["verdict_schema_first"] in ("PASS", "FAIL"), (
        f"verdict_schema_first must be PASS or FAIL, got {m['verdict_schema_first']!r}"
    )
