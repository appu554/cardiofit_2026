"""
Tests for Channel H: Cross-Channel Recovery (Phase 3b).

Validates:
1. No single-channel spans → no recovery
2. Drug name in single-channel span (not B) → recovery
3. Threshold pattern in single-channel span (not C) → recovery
4. Drug class in single-channel span (not B) → recovery
5. Multi-channel spans are NOT analyzed
6. Channel B single-channel span is NOT flagged for drug recovery
7. Recovery metadata (reason, detail, original_channel)
8. One recovery per span (break after first drug match)
9. Drug class not double-counted when drug name already found
"""

import sys
from pathlib import Path
from uuid import uuid4

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.channel_h_recovery import ChannelH, ChannelHRecovery
from extraction.v4.models import GuidelineTree, GuidelineSection, MergedSpan


# ── Test Fixtures ──────────────────────────────────────────────────────

def _make_tree(text_len: int) -> GuidelineTree:
    return GuidelineTree(
        sections=[GuidelineSection(
            section_id="1",
            heading="Test Section",
            start_offset=0,
            end_offset=text_len,
            page_number=1,
            block_type="recommendation",
            children=[],
        )],
        tables=[],
        total_pages=1,
    )


def _make_merged_span(
    text: str,
    channels: list[str],
    start: int = 0,
    confidence: float = 0.80,
) -> MergedSpan:
    """Build a test MergedSpan."""
    return MergedSpan(
        job_id=uuid4(),
        text=text,
        start=start,
        end=start + len(text),
        contributing_channels=channels,
        channel_confidences={ch: confidence for ch in channels},
        merged_confidence=confidence,
        page_number=1,
        section_id="1",
    )


# ── Tests ──────────────────────────────────────────────────────────────

class TestChannelHBasic:
    """Basic Channel H properties."""

    def test_alias(self):
        assert ChannelH is ChannelHRecovery

    def test_channel_tag(self):
        h = ChannelH()
        assert h.CHANNEL == "H"

    def test_confidence(self):
        h = ChannelH()
        assert h.CONFIDENCE == 0.60

    def test_version(self):
        h = ChannelH()
        assert h.VERSION == "4.3.0"


class TestNoRecovery:
    """Cases where no recovery should be produced."""

    def test_no_single_channel_spans(self):
        """Multi-channel spans should not trigger recovery."""
        text = "Prescribe metformin when eGFR > 30."
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["B", "C"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        assert len(output.spans) == 0
        assert output.metadata["single_channel_spans_analyzed"] == 0

    def test_single_channel_b_with_drug(self):
        """Channel B single-channel with drug → should NOT flag (B caught it)."""
        text = "metformin 500 mg"
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["B"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        # B caught the drug itself — no recovery needed for drug_missed_by_b
        drug_recoveries = [
            s for s in output.spans
            if s.channel_metadata.get("recovery_reason") == "drug_missed_by_b"
        ]
        assert len(drug_recoveries) == 0

    def test_empty_merged_spans(self):
        text = "Some text."
        tree = _make_tree(len(text))
        h = ChannelH()
        output = h.extract([], text, tree)
        assert len(output.spans) == 0


class TestDrugRecovery:
    """Drug name missed by Channel B."""

    def test_drug_in_non_b_channel(self):
        """If single-channel C has a drug name, recovery should flag it."""
        text = "Reduce metformin dose when eGFR < 30"
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["C"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        drug_recoveries = [
            s for s in output.spans
            if s.channel_metadata.get("recovery_reason") == "drug_missed_by_b"
        ]
        assert len(drug_recoveries) == 1
        assert drug_recoveries[0].channel == "H"
        assert drug_recoveries[0].confidence == 0.60
        assert "'metformin'" in drug_recoveries[0].channel_metadata["recovery_detail"]

    def test_drug_in_channel_d(self):
        """Channel D table cell mentioning a drug → recovery."""
        text = "dapagliflozin | 10 mg | daily"
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["D"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        drug_recoveries = [
            s for s in output.spans
            if s.channel_metadata.get("recovery_reason") == "drug_missed_by_b"
        ]
        assert len(drug_recoveries) == 1


class TestThresholdRecovery:
    """Threshold pattern missed by Channel C."""

    def test_threshold_in_non_c_channel(self):
        """If single-channel B has a threshold pattern, recovery should flag it."""
        text = "eGFR >= 45 mL/min"
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["B"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        threshold_recoveries = [
            s for s in output.spans
            if s.channel_metadata.get("recovery_reason") == "threshold_missed_by_c"
        ]
        assert len(threshold_recoveries) == 1
        assert threshold_recoveries[0].channel_metadata["original_channel"] == "B"

    def test_threshold_in_channel_c_not_flagged(self):
        """If single-channel C already has the threshold, don't flag."""
        text = "HbA1c < 7.0%"
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["C"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        threshold_recoveries = [
            s for s in output.spans
            if s.channel_metadata.get("recovery_reason") == "threshold_missed_by_c"
        ]
        assert len(threshold_recoveries) == 0


class TestDrugClassRecovery:
    """Drug class pattern missed by Channel B."""

    def test_drug_class_in_non_b_channel(self):
        """SGLT2 class mention in non-B channel → recovery."""
        text = "Consider SGLT2 inhibitors for cardiovascular benefit"
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["C"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        class_recoveries = [
            s for s in output.spans
            if s.channel_metadata.get("recovery_reason") == "class_missed_by_b"
        ]
        assert len(class_recoveries) == 1

    def test_no_double_count_drug_and_class(self):
        """If drug name already found, don't also flag drug class."""
        text = "Prescribe dapagliflozin (SGLT2 inhibitor) daily"
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["C"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        # dapagliflozin should trigger drug_missed_by_b
        # SGLT2 should NOT also trigger class_missed_by_b (dedup)
        reasons = [s.channel_metadata["recovery_reason"] for s in output.spans]
        assert "drug_missed_by_b" in reasons
        assert "class_missed_by_b" not in reasons


class TestRecoveryMetadata:
    """Verify recovery span metadata structure."""

    def test_recovery_span_structure(self):
        text = "Reduce metformin dose in CKD"
        tree = _make_tree(len(text))
        merged = [_make_merged_span(text, ["D"])]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        assert len(output.spans) >= 1
        span = output.spans[0]
        assert span.channel == "H"
        assert span.channel_metadata["original_channel"] == "D"
        assert "original_confidence" in span.channel_metadata

    def test_output_metadata_counts(self):
        text = "metformin dose eGFR < 30"
        tree = _make_tree(len(text))
        merged = [
            _make_merged_span("metformin dose", ["D"], start=0),
            _make_merged_span("eGFR < 30", ["B", "D"], start=15),  # multi-channel
        ]
        h = ChannelH()
        output = h.extract(merged, text, tree)
        assert output.metadata["single_channel_spans_analyzed"] == 1
        assert output.metadata["total_merged_spans"] == 2
        assert "recovery_reasons" in output.metadata
        assert "recovery_rate_pct" in output.metadata
