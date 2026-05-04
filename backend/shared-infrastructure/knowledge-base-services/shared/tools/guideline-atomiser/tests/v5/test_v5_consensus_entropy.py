"""V5 Consensus Entropy Gate unit tests.

Run from guideline-atomiser/:
    PYTHONPATH=. V5_CONSENSUS_ENTROPY=1 pytest tests/v5/test_v5_consensus_entropy.py -v
"""
from __future__ import annotations

import statistics
from uuid import uuid4

import pytest

from extraction.v4.signal_merger import SignalMerger
from extraction.v4.models import (
    ChannelOutput,
    GuidelineTree,
    MergedSpan,
    RawSpan,
)


def _make_tree() -> GuidelineTree:
    return GuidelineTree(sections=[], tables=[], total_pages=2)


def _make_raw_span(channel: str, text: str, confidence: float, start: int, end: int) -> RawSpan:
    return RawSpan(
        channel=channel,
        text=text,
        start=start,
        end=end,
        confidence=confidence,
    )


def _make_channel_output(channel: str, spans: list[RawSpan]) -> ChannelOutput:
    return ChannelOutput(channel=channel, spans=spans, metadata={})


# ─── Flag routing ──────────────────────────────────────────────────────────

def test_ce_gate_off_by_default(monkeypatch):
    """When V5_CONSENSUS_ENTROPY is absent, all spans are returned unchanged."""
    monkeypatch.delenv("V5_CONSENSUS_ENTROPY", raising=False)
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    merger = SignalMerger()
    tree = _make_tree()
    # Low-confidence single-channel span
    co = _make_channel_output("B", [_make_raw_span("B", "low conf", 0.20, 0, 8)])
    result = merger.merge(uuid4(), [co], tree)
    assert len(result) == 1
    assert not result[0].ce_flagged


def test_ce_gate_on_flags_low_single_channel(monkeypatch):
    """When V5_CONSENSUS_ENTROPY=1, single-channel spans below median are flagged and removed."""
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    merger = SignalMerger()
    tree = _make_tree()

    # 5 spans: 4 high-confidence (multi or high-single), 1 low single-channel
    # median will be ~0.90; the 0.20-confidence span should be flagged
    high_spans_b = [_make_raw_span("B", f"drug{i}", 0.90, i * 20, i * 20 + 10) for i in range(4)]
    low_span_c = _make_raw_span("C", "noise", 0.20, 100, 105)

    co_b = _make_channel_output("B", high_spans_b)
    co_c = _make_channel_output("C", [low_span_c])

    result = merger.merge(uuid4(), [co_b, co_c], tree)

    # The high-confidence B spans (non-overlapping) stay; the low C span is filtered
    non_flagged = [s for s in result if not s.ce_flagged]
    flagged = [s for s in result if s.ce_flagged]

    assert len(non_flagged) == 4, f"Expected 4 non-flagged spans, got {len(non_flagged)}"
    assert len(flagged) == 1, f"Expected 1 flagged span, got {len(flagged)}"
    assert flagged[0].text == "noise"


def test_disable_all_overrides_ce_gate(monkeypatch):
    """V5_DISABLE_ALL=1 forces V4 path — no CE gate applied."""
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    monkeypatch.setenv("V5_DISABLE_ALL", "1")

    merger = SignalMerger()
    tree = _make_tree()
    low_span = _make_raw_span("B", "low", 0.10, 0, 3)
    high_spans = [_make_raw_span("B", f"h{i}", 0.95, i * 20, i * 20 + 10) for i in range(4)]
    co = _make_channel_output("B", [low_span] + high_spans)

    result = merger.merge(uuid4(), [co], tree)
    assert all(not s.ce_flagged for s in result), "No spans should be flagged when V5_DISABLE_ALL=1"


def test_multi_channel_span_not_flagged_even_if_low_confidence(monkeypatch):
    """Multi-channel span is NEVER CE-flagged regardless of confidence."""
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    merger = SignalMerger()
    tree = _make_tree()

    # Two channels, same text region → will cluster into one multi-channel span
    span_b = _make_raw_span("B", "metformin 500mg", 0.15, 0, 15)
    span_c = _make_raw_span("C", "metformin 500mg", 0.15, 0, 15)
    # High-confidence spans to drive median up
    high = [_make_raw_span("B", f"h{i}", 0.95, i * 30, i * 30 + 10) for i in range(4)]

    co_b = _make_channel_output("B", [span_b] + high)
    co_c = _make_channel_output("C", [span_c])

    result = merger.merge(uuid4(), [co_b, co_c], tree)

    # The merged span from B+C should NOT be flagged (multi-channel)
    merged_mc = next(
        (s for s in result if "B" in s.contributing_channels and "C" in s.contributing_channels),
        None,
    )
    assert merged_mc is not None, "Expected a B+C merged span"
    assert not merged_mc.ce_flagged, "Multi-channel span must not be CE-flagged"


def test_fp_rate_drops_with_ce_gate(monkeypatch):
    """FP rate (single-channel below median) drops by ≥20% relative when CE gate is on."""
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    merger = SignalMerger()
    tree = _make_tree()

    # Construct: 4 high-conf B spans + 2 low-conf C spans (single-channel)
    # V4 FP rate = 2/6 = 33.3%; V5 target = FP rate should drop ≥20% relative → ≤26.7%
    high = [_make_raw_span("B", f"high{i}", 0.92, i * 30, i * 30 + 10) for i in range(4)]
    low = [_make_raw_span("C", f"low{i}", 0.20, 200 + i * 20, 210 + i * 20) for i in range(2)]

    # V4 baseline (no CE gate)
    monkeypatch.delenv("V5_CONSENSUS_ENTROPY", raising=False)
    co_b = _make_channel_output("B", high)
    co_c = _make_channel_output("C", low)
    v4_result = merger.merge(uuid4(), [co_b, co_c], tree)

    def fp_rate(spans):
        if not spans:
            return 0.0
        median_conf = statistics.median(s.merged_confidence for s in spans)
        fp = sum(
            1 for s in spans
            if len(s.contributing_channels) == 1
            and s.merged_confidence < median_conf
        )
        return fp / len(spans) * 100

    v4_fp = fp_rate(v4_result)

    # V5 with CE gate
    monkeypatch.setenv("V5_CONSENSUS_ENTROPY", "1")
    co_b2 = _make_channel_output("B", high)
    co_c2 = _make_channel_output("C", low)
    v5_result = merger.merge(uuid4(), [co_b2, co_c2], tree)

    # Only count non-flagged spans in V5 FP rate
    v5_non_flagged = [s for s in v5_result if not s.ce_flagged]
    v5_fp = fp_rate(v5_non_flagged)

    if v4_fp > 0:
        relative_drop = (v4_fp - v5_fp) / v4_fp * 100
        assert relative_drop >= 20.0, (
            f"FP relative drop {relative_drop:.1f}% < 20% threshold "
            f"(v4={v4_fp:.1f}%, v5={v5_fp:.1f}%)"
        )
