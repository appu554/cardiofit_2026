"""Unit tests for compute_v5_ce_metrics().

Test cases:
1. Empty span list → ce_flagged_spans=0, suppression_pct=0.0
2. All spans non-flagged → ce_flagged_spans=0, suppression_pct=0.0
3. 2 of 4 spans flagged → ce_flagged_spans=2, suppression_pct=50.0
4. ce_flagged missing from span dict → treated as non-flagged
5. ce_flagged=False explicitly → treated as non-flagged
"""
from __future__ import annotations

import sys
from pathlib import Path

import pytest

# Add data/ dir to path so we can import v5_metrics standalone.
DATA_DIR = Path(__file__).resolve().parents[2] / "data"
sys.path.insert(0, str(DATA_DIR))

from v5_metrics import compute_v5_ce_metrics  # noqa: E402


class TestComputeV5CeMetrics:
    def test_empty_span_list(self):
        result = compute_v5_ce_metrics([])
        gate = result["v5_ce_gate"]
        assert gate["total_spans"] == 0
        assert gate["ce_flagged_spans"] == 0
        assert gate["suppression_pct"] == 0.0

    def test_all_spans_non_flagged(self):
        spans = [
            {"text": "span A", "ce_flagged": False},
            {"text": "span B", "ce_flagged": False},
            {"text": "span C", "ce_flagged": False},
        ]
        result = compute_v5_ce_metrics(spans)
        gate = result["v5_ce_gate"]
        assert gate["total_spans"] == 3
        assert gate["ce_flagged_spans"] == 0
        assert gate["suppression_pct"] == 0.0

    def test_two_of_four_spans_flagged(self):
        spans = [
            {"text": "span A", "ce_flagged": True},
            {"text": "span B", "ce_flagged": False},
            {"text": "span C", "ce_flagged": True},
            {"text": "span D", "ce_flagged": False},
        ]
        result = compute_v5_ce_metrics(spans)
        gate = result["v5_ce_gate"]
        assert gate["total_spans"] == 4
        assert gate["ce_flagged_spans"] == 2
        assert gate["suppression_pct"] == 50.0

    def test_ce_flagged_missing_treated_as_non_flagged(self):
        spans = [
            {"text": "span A"},
            {"text": "span B"},
        ]
        result = compute_v5_ce_metrics(spans)
        gate = result["v5_ce_gate"]
        assert gate["total_spans"] == 2
        assert gate["ce_flagged_spans"] == 0
        assert gate["suppression_pct"] == 0.0

    def test_ce_flagged_false_explicit_treated_as_non_flagged(self):
        spans = [
            {"text": "span A", "ce_flagged": False},
            {"text": "span B", "ce_flagged": True},
        ]
        result = compute_v5_ce_metrics(spans)
        gate = result["v5_ce_gate"]
        assert gate["total_spans"] == 2
        assert gate["ce_flagged_spans"] == 1
        assert gate["suppression_pct"] == 50.0

    def test_top_level_key_is_v5_ce_gate(self):
        result = compute_v5_ce_metrics([{"ce_flagged": True}])
        assert "v5_ce_gate" in result

    def test_suppression_pct_rounded_to_two_decimal_places(self):
        # 1 of 3 flagged → 33.33%
        spans = [
            {"ce_flagged": True},
            {"ce_flagged": False},
            {"ce_flagged": False},
        ]
        result = compute_v5_ce_metrics(spans)
        gate = result["v5_ce_gate"]
        assert gate["suppression_pct"] == 33.33
