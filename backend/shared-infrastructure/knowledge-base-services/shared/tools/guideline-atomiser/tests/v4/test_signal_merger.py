"""
Tests for Signal Merger.

Validates:
1. Union of all channel spans
2. Clustering of overlapping spans (≥50% overlap)
3. Confidence boosting by channel count
4. Longest text selection for merged span
5. Disagreement detection
6. Section assignment from tree
7. Edge cases (empty input, single channel, all same text)
"""

import sys
from pathlib import Path
from uuid import uuid4

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.signal_merger import SignalMerger
from extraction.v4.models import (
    ChannelOutput,
    GuidelineTree,
    GuidelineSection,
    MergedSpan,
    RawSpan,
)


@pytest.fixture
def merger():
    return SignalMerger()


@pytest.fixture
def job_id():
    return uuid4()


@pytest.fixture
def simple_tree():
    return GuidelineTree(
        sections=[
            GuidelineSection(
                section_id="1",
                heading="Test",
                start_offset=0,
                end_offset=100000,
                page_number=1,
                block_type="paragraph",
                children=[],
            )
        ],
        tables=[],
        total_pages=1,
    )


class TestUnion:
    """Test that all channel spans are collected."""

    def test_empty_channels(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[]),
            ChannelOutput(channel="C", spans=[]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert len(result) == 0

    def test_single_channel_spans(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert len(result) == 1

    def test_multi_channel_non_overlapping(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="C", spans=[
                RawSpan(channel="C", text="eGFR >= 30", start=50, end=60, confidence=0.95),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert len(result) == 2

    def test_failed_channel_ignored(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[], error="GLiNER failed"),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert len(result) == 1


class TestClustering:
    """Test overlapping span clustering."""

    def test_exact_overlap_clustered(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="metformin", start=10, end=19, confidence=0.78),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        # Should cluster into 1 merged span
        assert len(result) == 1
        assert sorted(result[0].contributing_channels) == ["B", "E"]

    def test_partial_overlap_clustered(self, merger, job_id, simple_tree):
        # "metformin" (10-19) and "metformin 500mg" (10-25)
        # overlap = 9, shorter = 9, ratio = 100% — should cluster
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="metformin 500mg", start=10, end=25, confidence=0.7),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert len(result) == 1

    def test_no_overlap_separate(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="finerenone", start=100, end=110, confidence=1.0),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert len(result) == 2


class TestConfidenceBoost:
    """Test confidence boosting by channel count."""

    def test_single_channel_no_boost(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=0.90),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert result[0].merged_confidence == pytest.approx(0.90, abs=0.01)

    def test_two_channels_boost_0_05(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="metformin", start=10, end=19, confidence=0.80),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        # avg(1.0, 0.80) = 0.90 + 0.05 boost = 0.95
        assert result[0].merged_confidence == pytest.approx(0.95, abs=0.01)

    def test_three_channels_boost_0_10(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="C", spans=[
                RawSpan(channel="C", text="metformin", start=10, end=19, confidence=0.85),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="metformin", start=10, end=19, confidence=0.80),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        # avg(1.0, 0.85, 0.80) = 0.883 + 0.10 = 0.983
        assert result[0].merged_confidence > 0.95

    def test_confidence_capped_at_1(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="C", spans=[
                RawSpan(channel="C", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="D", spans=[
                RawSpan(channel="D", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="metformin", start=10, end=19, confidence=1.0),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert result[0].merged_confidence <= 1.0


class TestTextSelection:
    """Test longest text selection for merged span."""

    def test_longest_text_selected(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="metformin 500mg", start=10, end=25, confidence=0.7),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert result[0].text == "metformin 500mg"


class TestDisagreement:
    """Test disagreement detection."""

    def test_same_text_no_disagreement(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="metformin", start=10, end=19, confidence=0.8),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert result[0].has_disagreement is False

    def test_different_text_flags_disagreement(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="metformin 500mg", start=10, end=25, confidence=0.7),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        assert result[0].has_disagreement is True
        assert result[0].disagreement_detail is not None

    def test_case_difference_is_not_disagreement(self, merger, job_id, simple_tree):
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
            ]),
            ChannelOutput(channel="E", spans=[
                RawSpan(channel="E", text="Metformin", start=10, end=19, confidence=0.8),
            ]),
        ]
        result = merger.merge(job_id, outputs, simple_tree)
        # Case-only difference is NOT disagreement (compared lowercase)
        assert result[0].has_disagreement is False


class TestSectionAssignment:
    """Test section_id assignment from tree."""

    def test_section_assigned(self, merger, job_id, sample_guideline_tree, sample_text):
        metformin_start = sample_text.index("metformin in patients")
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=metformin_start,
                       end=metformin_start + 9, confidence=1.0),
            ]),
        ]
        result = merger.merge(job_id, outputs, sample_guideline_tree)
        assert result[0].section_id == "4.1.1"


class TestJobId:
    """Test job_id propagation."""

    def test_job_id_on_all_spans(self, merger, simple_tree):
        jid = uuid4()
        outputs = [
            ChannelOutput(channel="B", spans=[
                RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
                RawSpan(channel="B", text="finerenone", start=100, end=110, confidence=1.0),
            ]),
        ]
        result = merger.merge(jid, outputs, simple_tree)
        for span in result:
            assert span.job_id == jid


class TestFullPipelineIntegration:
    """Test with fixtures from conftest."""

    def test_merge_sample_raw_spans(self, merger, sample_job_id, sample_guideline_tree, sample_raw_spans):
        # Group raw spans by channel to create ChannelOutputs
        channel_spans: dict[str, list[RawSpan]] = {}
        for span in sample_raw_spans:
            channel_spans.setdefault(span.channel, []).append(span)

        outputs = [
            ChannelOutput(channel=ch, spans=spans)
            for ch, spans in channel_spans.items()
        ]
        result = merger.merge(sample_job_id, outputs, sample_guideline_tree)
        # Should produce merged spans
        assert len(result) >= 2  # at least metformin cluster + eGFR
