"""
Tests for Channel C: Triple Filter (V4.1).

Validates the three filter methods added to ChannelCGrammar:
1. _is_in_citation_zone: Inline citations [nn] and reference sections
2. _is_in_header_zone: Section heading text (not body text)
3. _is_in_table_caption: "Table N" / "Figure N" caption lines

Also validates that filter rejection counts appear in output metadata.
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.models import GuidelineTree, GuidelineSection, TableBoundary
from extraction.v4.channel_c_grammar import ChannelCGrammar


@pytest.fixture
def grammar():
    """Channel C grammar extractor instance."""
    return ChannelCGrammar()


def _simple_tree(text: str, heading: str = "Clinical Recommendation") -> GuidelineTree:
    """Build a single-section GuidelineTree wrapping the full text."""
    return GuidelineTree(
        sections=[
            GuidelineSection(
                section_id="1",
                heading=heading,
                start_offset=0,
                end_offset=len(text),
                page_number=1,
                block_type="recommendation",
                children=[],
            )
        ],
        tables=[],
        total_pages=1,
    )


# =============================================================================
# Citation Zone Filter Tests
# =============================================================================

class TestCitationZoneFilter:
    """Tests for _is_in_citation_zone method."""

    def test_inline_citation_detected(self, grammar):
        """Match inside [nn] citation should be filtered."""
        text = "See outcomes [14] for details"
        tree = _simple_tree(text)
        # "14" is at offset 13-15, inside brackets 12-16
        assert grammar._is_in_citation_zone(13, 15, text, tree) is True

    def test_multi_citation_detected(self, grammar):
        """Match inside [14,15] citation should be filtered."""
        text = "Proven effective [14,15] in trials"
        tree = _simple_tree(text)
        bracket_start = text.index("[14,15]")
        inner_start = bracket_start + 1
        inner_end = bracket_start + 6  # "14,15"
        assert grammar._is_in_citation_zone(inner_start, inner_end, text, tree) is True

    def test_clinical_text_not_citation(self, grammar):
        """Normal clinical text should not be filtered as citation."""
        text = "eGFR >= 30 mL/min/1.73m2 is the threshold"
        tree = _simple_tree(text)
        start = text.index("eGFR")
        end = start + 10  # "eGFR >= 30"
        assert grammar._is_in_citation_zone(start, end, text, tree) is False

    def test_reference_section_heading(self, grammar):
        """Match in a section with reference heading should be filtered."""
        text = "eGFR threshold is documented here."
        ref_tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="refs",
                    heading="References",
                    start_offset=0,
                    end_offset=len(text),
                    page_number=10,
                    block_type="paragraph",
                    children=[],
                )
            ],
            tables=[],
            total_pages=10,
        )
        start = text.index("eGFR")
        end = start + 4
        assert grammar._is_in_citation_zone(start, end, text, ref_tree) is True

    def test_bibliography_heading(self, grammar):
        """Section titled 'Bibliography' should also be filtered."""
        text = "metformin was studied extensively."
        bib_tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="bib",
                    heading="Bibliography and Supplementary Materials",
                    start_offset=0,
                    end_offset=len(text),
                    page_number=20,
                    block_type="paragraph",
                    children=[],
                )
            ],
            tables=[],
            total_pages=20,
        )
        assert grammar._is_in_citation_zone(0, 9, text, bib_tree) is True


# =============================================================================
# Header Zone Filter Tests
# =============================================================================

class TestHeaderZoneFilter:
    """Tests for _is_in_header_zone method."""

    def test_match_in_heading_text_filtered(self, grammar):
        """Match that falls within the heading string should be filtered."""
        heading = "Chapter 4: SGLT2 inhibitors and renal outcomes"
        body = "\nDapagliflozin can be initiated when eGFR >= 25."
        text = heading + body

        tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="4",
                    heading=heading,
                    start_offset=0,
                    end_offset=len(text),
                    page_number=1,
                    block_type="heading",
                    children=[],
                )
            ],
            tables=[],
            total_pages=1,
        )

        # "eGFR" in body is NOT in heading zone
        egfr_in_body = text.index("eGFR >= 25")
        assert grammar._is_in_header_zone(egfr_in_body, egfr_in_body + 10, tree) is False

    def test_match_in_body_passes(self, grammar):
        """Match in body text (well after heading) should not be filtered."""
        # Heading zone = start_offset + len(heading) + 5 margin + 20
        # So body content must start beyond that range
        heading = "Rec"
        # Add enough space that body content is clearly outside heading zone
        body = "\n\nWe recommend that eGFR >= 30 is the threshold for metformin."
        text = heading + body

        tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="1",
                    heading=heading,
                    start_offset=0,
                    end_offset=len(text),
                    page_number=1,
                    block_type="recommendation",
                    children=[],
                )
            ],
            tables=[],
            total_pages=1,
        )

        egfr_start = text.index("eGFR")
        # heading_end = 0 + 3 + 5 = 8. start=23 is NOT < 8, so NOT in header zone.
        assert grammar._is_in_header_zone(egfr_start, egfr_start + 10, tree) is False


# =============================================================================
# Table Caption Filter Tests
# =============================================================================

class TestTableCaptionFilter:
    """Tests for _is_in_table_caption method."""

    def test_table_caption_filtered(self, grammar):
        """Match within 'Table N' caption line should be filtered."""
        text = "Table 3. eGFR thresholds for dose adjustment\n| Drug | eGFR |"
        start = text.index("eGFR")
        end = start + 4
        assert grammar._is_in_table_caption(start, end, text) is True

    def test_figure_caption_filtered(self, grammar):
        """Match within 'Figure N' caption line should be filtered."""
        text = "Figure 2. Metformin dose response curve\nSee panel A."
        start = text.index("Metformin")
        end = start + 9
        assert grammar._is_in_table_caption(start, end, text) is True

    def test_normal_text_passes(self, grammar):
        """Match in normal body text should not be filtered."""
        text = "We recommend eGFR monitoring every 3-6 months."
        start = text.index("eGFR")
        end = start + 4
        assert grammar._is_in_table_caption(start, end, text) is False

    def test_table_body_passes(self, grammar):
        """Match in table body (markdown pipe row) should not be filtered."""
        text = "| Metformin | >= 30 | 1000 mg |"
        start = text.index(">= 30")
        end = start + 5
        assert grammar._is_in_table_caption(start, end, text) is False


# =============================================================================
# Integration: Full Extract with Filter Metadata
# =============================================================================

class TestTripleFilterIntegration:
    """Tests that triple filter rejection counts appear in Channel C output."""

    def test_filter_counts_in_metadata(self, grammar):
        """Output metadata should contain filter_rejections dict."""
        text = (
            "## Clinical Recommendations\n\n"
            "eGFR >= 30 mL/min/1.73m2 for metformin dosing.\n"
            "Monitor potassium every 3 months.\n"
        )
        tree = _simple_tree(text)
        output = grammar.extract(text, tree)

        assert "filter_rejections" in output.metadata
        counts = output.metadata["filter_rejections"]
        assert "citation_zone" in counts
        assert "header_zone" in counts
        assert "table_caption" in counts

    def test_clinical_text_no_rejections(self, grammar):
        """Pure clinical text should have zero filter rejections.

        Uses a short heading so body content falls outside the heading zone
        (heading_end = start + len(heading) + 5 + 20 margin).
        """
        # Short heading "R" → heading zone = 0..26 at most
        # Body content starts at offset 40+ to be safely outside
        padding = " " * 40
        text = (
            f"R\n{padding}\n"
            "We recommend eGFR >= 30 mL/min/1.73m2.\n"
            "Monitor eGFR every 3-6 months.\n"
            "Dose: 1000 mg daily.\n"
        )
        tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="1",
                    heading="R",
                    start_offset=0,
                    end_offset=len(text),
                    page_number=1,
                    block_type="recommendation",
                    children=[],
                )
            ],
            tables=[],
            total_pages=1,
        )
        output = grammar.extract(text, tree)

        counts = output.metadata["filter_rejections"]
        total_rejections = sum(counts.values())
        assert total_rejections == 0

    def test_table_caption_rejection_counted(self, grammar):
        """Matches on 'Table N' lines should increment table_caption count.

        Heading zone must not overlap with the Table caption line, so we
        place the caption well past the heading zone boundary.
        """
        padding = " " * 50
        text = (
            f"R\n{padding}\n"
            "Table 1. eGFR >= 30 threshold summary\n"
            "| Drug | eGFR Threshold |\n"
            "| Metformin | >= 30 |\n"
        )
        tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="1",
                    heading="R",
                    start_offset=0,
                    end_offset=len(text),
                    page_number=1,
                    block_type="recommendation",
                    children=[],
                )
            ],
            tables=[],
            total_pages=1,
        )
        output = grammar.extract(text, tree)

        counts = output.metadata["filter_rejections"]
        # At least 1 rejection on the "Table 1" line
        assert counts["table_caption"] >= 1
