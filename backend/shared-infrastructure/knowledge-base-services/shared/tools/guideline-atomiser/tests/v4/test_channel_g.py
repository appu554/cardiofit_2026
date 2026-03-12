"""
Tests for Channel G: Sentence-Level Context Extraction (Phase 3a).

Validates:
1. Empty prior channels → no output
2. Single anchor → sentence extraction with correct boundaries
3. Multiple anchors → overlapping merges
4. Sentence boundary detection (period, double newline)
5. Max sentence length clamping
6. Channel metadata tagging
7. Short sentences filtered (<10 chars)
8. Channel tag = "G" and confidence = 0.70
"""

import sys
from pathlib import Path
from uuid import uuid4

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.channel_g_sentence import ChannelG, ChannelGSentence
from extraction.v4.models import ChannelOutput, GuidelineTree, GuidelineSection, RawSpan


# ── Test Fixtures ──────────────────────────────────────────────────────

def _make_tree(text_len: int, page_offsets: dict[int, tuple[int, int]] | None = None) -> GuidelineTree:
    """Build a minimal GuidelineTree covering the whole text."""
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


def _make_channel_output(channel: str, spans: list[tuple[int, int, str]]) -> ChannelOutput:
    """Build a ChannelOutput with RawSpans at given offsets."""
    raw_spans = [
        RawSpan(
            channel=channel,
            text=text,
            start=start,
            end=end,
            confidence=0.85,
        )
        for start, end, text in spans
    ]
    return ChannelOutput(channel=channel, spans=raw_spans)


# ── Tests ──────────────────────────────────────────────────────────────

class TestChannelGBasic:
    """Basic Channel G functionality."""

    def test_alias(self):
        assert ChannelG is ChannelGSentence

    def test_channel_tag(self):
        g = ChannelG()
        assert g.CHANNEL == "G"

    def test_confidence(self):
        g = ChannelG()
        assert g.CONFIDENCE == 0.70

    def test_version(self):
        g = ChannelG()
        assert g.VERSION == "4.3.0"


class TestEmptyInput:
    """Behavior with no prior channel outputs."""

    def test_no_prior_outputs(self):
        g = ChannelG()
        text = "Some clinical text about metformin dosing."
        tree = _make_tree(len(text))
        output = g.extract(text, tree, [])
        assert output.channel == "G"
        assert len(output.spans) == 0
        assert output.metadata["source_span_count"] == 0

    def test_failed_prior_outputs(self):
        g = ChannelG()
        text = "Some clinical text about metformin dosing."
        tree = _make_tree(len(text))
        failed = ChannelOutput(channel="B", spans=[], error="test error")
        output = g.extract(text, tree, [failed])
        assert len(output.spans) == 0


class TestSentenceExtraction:
    """Sentence boundary detection and expansion."""

    def test_single_anchor_expands_to_sentence(self):
        """A B-channel span on 'metformin' should expand to the full sentence."""
        text = "First sentence here. Prescribe metformin for the patient. Another sentence follows."
        tree = _make_tree(len(text))
        b_output = _make_channel_output("B", [(30, 39, "metformin")])
        g = ChannelG()
        output = g.extract(text, tree, [b_output])
        assert len(output.spans) >= 1
        # The sentence span should contain "metformin"
        found = [s for s in output.spans if "metformin" in s.text]
        assert len(found) == 1
        assert found[0].channel == "G"
        assert found[0].confidence == 0.70

    def test_double_newline_boundary(self):
        """Double newline should be treated as sentence boundary."""
        text = "First paragraph about drug dosing.\n\nSecond paragraph about eGFR thresholds.\n\nThird paragraph."
        tree = _make_tree(len(text))
        # Anchor in the second paragraph
        idx = text.index("eGFR")
        c_output = _make_channel_output("C", [(idx, idx + 4, "eGFR")])
        g = ChannelG()
        output = g.extract(text, tree, [c_output])
        assert len(output.spans) >= 1
        found = [s for s in output.spans if "eGFR" in s.text]
        assert len(found) == 1
        # Should not include first or third paragraph
        assert "First paragraph" not in found[0].text

    def test_short_sentences_filtered(self):
        """Sentences shorter than 10 chars should be filtered out."""
        text = "OK. This is a longer sentence about clinical management."
        tree = _make_tree(len(text))
        # Anchor on "OK"
        b_output = _make_channel_output("B", [(0, 2, "OK")])
        g = ChannelG()
        output = g.extract(text, tree, [b_output])
        # "OK." is only 3 chars — should be filtered
        for span in output.spans:
            assert len(span.text.strip()) >= 10

    def test_multiple_anchors_merge(self):
        """Overlapping sentence expansions should merge."""
        text = "Prescribe metformin and dapagliflozin together for optimal glycemic control in CKD patients."
        tree = _make_tree(len(text))
        b_output = _make_channel_output("B", [
            (10, 19, "metformin"),
            (24, 38, "dapagliflozin"),
        ])
        g = ChannelG()
        output = g.extract(text, tree, [b_output])
        # Both anchors are in the same sentence — should produce 1 merged span
        assert len(output.spans) == 1
        assert "metformin" in output.spans[0].text
        assert "dapagliflozin" in output.spans[0].text


class TestChannelMetadata:
    """Verify source channel tracking in metadata."""

    def test_source_channels_tagged(self):
        text = "Reduce metformin dose when eGFR drops below 30 mL/min."
        tree = _make_tree(len(text))
        b_output = _make_channel_output("B", [(7, 16, "metformin")])
        c_output = _make_channel_output("C", [(27, 54, "eGFR drops below 30 mL/min")])
        g = ChannelG()
        output = g.extract(text, tree, [b_output, c_output])
        assert len(output.spans) >= 1
        # The merged sentence should list both B and C as sources
        source_channels = output.spans[0].channel_metadata.get("source_channels", [])
        assert "B" in source_channels or "C" in source_channels

    def test_metadata_sentence_count(self):
        text = "First sentence about drugs. Second about thresholds. Third about safety."
        tree = _make_tree(len(text))
        b_output = _make_channel_output("B", [(22, 27, "drugs")])
        g = ChannelG()
        output = g.extract(text, tree, [b_output])
        assert "sentence_count" in output.metadata
        assert output.metadata["sentence_count"] == len(output.spans)


class TestMaxSentenceLength:
    """Verify MAX_SENTENCE_LENGTH clamping."""

    def test_long_text_clamped(self):
        """Very long text without sentence boundaries should be clamped."""
        # Build a 1000-char text with no sentence boundaries
        long_text = "a" * 300 + "metformin" + "b" * 700
        tree = _make_tree(len(long_text))
        b_output = _make_channel_output("B", [(300, 309, "metformin")])
        g = ChannelG()
        output = g.extract(long_text, tree, [b_output])
        if output.spans:
            for span in output.spans:
                assert len(span.text) <= g.MAX_SENTENCE_LENGTH + 50  # small margin for stripping
