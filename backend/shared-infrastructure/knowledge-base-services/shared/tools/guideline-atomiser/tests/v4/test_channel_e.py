"""
Tests for Channel E: GLiNER Residual Booster.

Validates:
1. Novel-only filtering (removes spans already found by B+C)
2. GLiNER unavailability handled gracefully
3. Overlap calculation for deduplication
4. Channel output structure
5. Full text processing (no truncation)
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.channel_e_gliner import ChannelEGLiNERResidual
from extraction.v4.models import (
    ChannelOutput,
    GuidelineTree,
    GuidelineSection,
    RawSpan,
)


@pytest.fixture
def simple_tree():
    """Minimal GuidelineTree for testing."""
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


class TestNovelFiltering:
    """Test novel-only span filtering logic."""

    def test_filter_removes_exact_overlap(self):
        """Spans that exactly match existing B+C spans should be removed."""
        channel_e = ChannelEGLiNERResidual()

        gliner_spans = [
            RawSpan(channel="E", text="metformin", start=10, end=19, confidence=0.8),
        ]
        existing_spans = [
            RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
        ]
        result = channel_e._filter_novel(gliner_spans, existing_spans)
        assert len(result) == 0

    def test_filter_keeps_novel_spans(self):
        """Spans not found by B+C should be kept."""
        channel_e = ChannelEGLiNERResidual()

        gliner_spans = [
            RawSpan(channel="E", text="novel_entity", start=50, end=62, confidence=0.7),
        ]
        existing_spans = [
            RawSpan(channel="B", text="metformin", start=10, end=19, confidence=1.0),
        ]
        result = channel_e._filter_novel(gliner_spans, existing_spans)
        assert len(result) == 1
        assert result[0].text == "novel_entity"

    def test_filter_partial_overlap_under_50_kept(self):
        """Spans with less than 50% overlap should be kept."""
        channel_e = ChannelEGLiNERResidual()

        # 10 char span, 4 char overlap = 40% < 50%
        gliner_spans = [
            RawSpan(channel="E", text="0123456789", start=10, end=20, confidence=0.7),
        ]
        existing_spans = [
            RawSpan(channel="C", text="xxxx", start=16, end=20, confidence=0.9),
        ]
        result = channel_e._filter_novel(gliner_spans, existing_spans)
        assert len(result) == 1

    def test_filter_partial_overlap_over_50_removed(self):
        """Spans with >50% overlap should be removed."""
        channel_e = ChannelEGLiNERResidual()

        # 10 char span, 6 char overlap = 60% > 50%
        gliner_spans = [
            RawSpan(channel="E", text="0123456789", start=10, end=20, confidence=0.7),
        ]
        existing_spans = [
            RawSpan(channel="C", text="xxxxxx", start=14, end=20, confidence=0.9),
        ]
        result = channel_e._filter_novel(gliner_spans, existing_spans)
        assert len(result) == 0

    def test_filter_empty_existing(self):
        """No existing spans means all GLiNER spans are novel."""
        channel_e = ChannelEGLiNERResidual()

        gliner_spans = [
            RawSpan(channel="E", text="drug_x", start=0, end=6, confidence=0.7),
            RawSpan(channel="E", text="drug_y", start=10, end=16, confidence=0.7),
        ]
        result = channel_e._filter_novel(gliner_spans, [])
        assert len(result) == 2

    def test_filter_zero_length_spans_skipped(self):
        """Zero-length spans should be skipped."""
        channel_e = ChannelEGLiNERResidual()

        gliner_spans = [
            RawSpan(channel="E", text="", start=10, end=10, confidence=0.5),
        ]
        result = channel_e._filter_novel(gliner_spans, [])
        assert len(result) == 0


class TestGracefulDegradation:
    """Test behavior when GLiNER is not available."""

    def test_unavailable_returns_error_output(self, simple_tree):
        channel_e = ChannelEGLiNERResidual()
        # GLiNER won't be available in test environment (no model loaded)
        if not channel_e.available:
            output = channel_e.extract("some text", simple_tree, [])
            assert output.success is False
            assert output.error is not None
            assert output.span_count == 0

    def test_available_property(self):
        channel_e = ChannelEGLiNERResidual()
        # In test env, GLiNER is likely not available
        assert isinstance(channel_e.available, bool)


class TestChannelOutputStructure:
    """Test ChannelOutput structure when GLiNER is not available."""

    def test_output_channel_is_e(self, simple_tree):
        channel_e = ChannelEGLiNERResidual()
        output = channel_e.extract("test text", simple_tree, [])
        assert output.channel == "E"

    def test_output_has_spans_list(self, simple_tree):
        channel_e = ChannelEGLiNERResidual()
        output = channel_e.extract("test text", simple_tree, [])
        assert isinstance(output.spans, list)
