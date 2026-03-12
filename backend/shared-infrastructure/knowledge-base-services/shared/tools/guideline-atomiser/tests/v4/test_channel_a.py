"""
Tests for Channel A: Docling Structure Parser.

Validates:
1. GuidelineTree construction from Docling markdown
2. Section extraction with correct section_ids
3. Table boundary detection
4. Page marker parsing
5. Parent-child hierarchy from dot-separated IDs
6. Block type classification
7. Edge cases (no headings, no tables, empty text)
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.channel_a_docling import ChannelADoclingParser
from extraction.v4.models import GuidelineSection, GuidelineTree, TableBoundary


@pytest.fixture
def parser():
    """Channel A parser instance."""
    return ChannelADoclingParser()


class TestSectionExtraction:
    """Test section extraction from ATX headings."""

    def test_extract_three_recommendations(self, parser, sample_text):
        tree = parser.parse(sample_text)
        # Flatten all sections including children
        all_sections = self._flatten(tree.sections)
        rec_ids = [s.section_id for s in all_sections if "4." in s.section_id and s.section_id != "4"]
        assert "4.1.1" in rec_ids
        assert "4.1.2" in rec_ids
        assert "4.2.1" in rec_ids

    def test_chapter_heading_extracted(self, parser, sample_text):
        tree = parser.parse(sample_text)
        assert len(tree.sections) >= 1
        top_section = tree.sections[0]
        assert "4" == top_section.section_id or "Chapter" in top_section.heading

    def test_section_offsets_cover_text(self, parser, sample_text):
        tree = parser.parse(sample_text)
        all_sections = self._flatten(tree.sections)
        # No section should start after text ends
        for section in all_sections:
            assert section.start_offset < len(sample_text)
            assert section.end_offset <= len(sample_text)
            assert section.start_offset < section.end_offset

    def test_recommendation_block_type(self, parser, sample_text):
        tree = parser.parse(sample_text)
        all_sections = self._flatten(tree.sections)
        rec_sections = [s for s in all_sections if s.section_id == "4.1.1"]
        assert len(rec_sections) == 1
        assert rec_sections[0].block_type == "recommendation"

    def test_heading_block_type_for_chapter(self, parser, sample_text):
        tree = parser.parse(sample_text)
        top_section = tree.sections[0]
        assert top_section.block_type == "heading"

    def _flatten(self, sections: list[GuidelineSection]) -> list[GuidelineSection]:
        result = []
        for s in sections:
            result.append(s)
            result.extend(self._flatten(s.children))
        return result


class TestHierarchy:
    """Test parent-child hierarchy building."""

    def test_chapter_has_children(self, parser, sample_text):
        tree = parser.parse(sample_text)
        # Chapter 4 should have recommendation children
        chapter = tree.sections[0]
        assert len(chapter.children) >= 2  # at least 4.1.1 and 4.1.2

    def test_child_section_ids(self, parser, sample_text):
        tree = parser.parse(sample_text)
        chapter = tree.sections[0]
        child_ids = [c.section_id for c in chapter.children]
        # 4.1.1 and 4.1.2 should be children, possibly 4.2.1 too
        assert any("4.1" in cid for cid in child_ids)

    def test_flat_text_single_root(self, parser):
        text = "No headings here, just plain text about drugs."
        tree = parser.parse(text)
        assert len(tree.sections) == 1
        assert tree.sections[0].section_id == "1"
        assert tree.sections[0].heading == "Document"


class TestTableExtraction:
    """Test table boundary detection."""

    def test_table_detected(self, parser, sample_text):
        tree = parser.parse(sample_text)
        assert len(tree.tables) >= 1

    def test_table_headers(self, parser, sample_text):
        tree = parser.parse(sample_text)
        table = tree.tables[0]
        assert "Drug" in table.headers
        assert len(table.headers) == 4

    def test_table_row_count(self, parser, sample_text):
        tree = parser.parse(sample_text)
        table = tree.tables[0]
        # 3 data rows (Metformin, Dapagliflozin, Finerenone)
        assert table.row_count == 3

    def test_table_offsets_within_text(self, parser, sample_text):
        tree = parser.parse(sample_text)
        for table in tree.tables:
            assert table.start_offset >= 0
            assert table.end_offset <= len(sample_text)
            assert table.start_offset < table.end_offset

    def test_table_has_parent_section(self, parser, sample_text):
        tree = parser.parse(sample_text)
        table = tree.tables[0]
        assert table.section_id is not None

    def test_standalone_table(self, parser, sample_table_markdown):
        tree = parser.parse(sample_table_markdown)
        # Should detect the table even without headings
        assert len(tree.tables) >= 1
        table = tree.tables[0]
        assert table.row_count == 4  # 4 drug rows


class TestPageMarkers:
    """Test page marker parsing."""

    def test_page_markers_increment(self, parser):
        text = """\
<!-- PAGE 1 -->
## Chapter 1: Introduction

Some text here.

<!-- PAGE 2 -->
## Chapter 2: Methods

More text here.

<!-- PAGE 3 -->
## Chapter 3: Results
"""
        tree = parser.parse(text)
        assert tree.total_pages == 3

    def test_no_page_markers_defaults_to_1(self, parser, sample_text):
        tree = parser.parse(sample_text)
        assert tree.total_pages == 1

    def test_page_number_assignment(self, parser):
        text = """\
<!-- PAGE 1 -->
## Section 1

Content on page 1.

<!-- PAGE 2 -->
## Section 2

Content on page 2.
"""
        tree = parser.parse(text)
        all_sections = []
        for s in tree.sections:
            all_sections.append(s)
            all_sections.extend(s.children)

        # Section 2 should be on page 2
        section_2 = [s for s in all_sections if s.section_id == "2"]
        assert len(section_2) == 1
        assert section_2[0].page_number == 2


class TestTreeLookups:
    """Test GuidelineTree offset-based lookups with parser-generated tree."""

    def test_find_section_for_offset(self, parser, sample_text):
        tree = parser.parse(sample_text)
        # Find offset of "metformin in patients"
        offset = sample_text.index("metformin in patients")
        section = tree.find_section_for_offset(offset)
        assert section is not None
        assert section.section_id == "4.1.1"

    def test_find_section_for_dapagliflozin(self, parser, sample_text):
        tree = parser.parse(sample_text)
        offset = sample_text.index("Dapagliflozin can")
        section = tree.find_section_for_offset(offset)
        assert section is not None
        assert section.section_id == "4.1.2"

    def test_find_table_for_offset(self, parser, sample_text):
        tree = parser.parse(sample_text)
        offset = sample_text.index("| Drug |")
        table = tree.find_table_for_offset(offset)
        assert table is not None

    def test_no_section_for_invalid_offset(self, parser, sample_text):
        tree = parser.parse(sample_text)
        section = tree.find_section_for_offset(999999)
        assert section is None


class TestBlockTypeClassification:
    """Test classify_block_types method."""

    def test_table_cells_classified(self, parser, sample_text):
        tree = parser.parse(sample_text)
        block_types = parser.classify_block_types(sample_text, tree)
        # Find a table offset
        table_offset = sample_text.index("| Drug |")
        # The line starting with | should be table_cell
        classified_as_table = any(
            v == "table_cell"
            for k, v in block_types.items()
            if abs(k - table_offset) < 5
        )
        assert classified_as_table

    def test_paragraph_classified(self, parser, sample_text):
        tree = parser.parse(sample_text)
        block_types = parser.classify_block_types(sample_text, tree)
        # Should have some paragraph blocks
        paragraphs = [k for k, v in block_types.items() if v == "paragraph"]
        assert len(paragraphs) > 0

    def test_heading_classified(self, parser, sample_text):
        tree = parser.parse(sample_text)
        block_types = parser.classify_block_types(sample_text, tree)
        headings = [k for k, v in block_types.items() if v == "heading"]
        assert len(headings) > 0


class TestEdgeCases:
    """Test edge cases and unusual inputs."""

    def test_empty_text(self, parser):
        tree = parser.parse("")
        assert len(tree.sections) == 1
        assert tree.sections[0].section_id == "1"

    def test_text_with_only_table(self, parser):
        text = """\
| A | B |
| --- | --- |
| 1 | 2 |
| 3 | 4 |
"""
        tree = parser.parse(text)
        assert len(tree.tables) >= 1
        assert tree.tables[0].row_count == 2

    def test_deeply_nested_sections(self, parser):
        text = """\
# Section 1
## Section 1.1
### Section 1.1.1
#### Section 1.1.1.1

Deep content here.
"""
        tree = parser.parse(text)
        # Should build hierarchy
        assert len(tree.sections) >= 1
        # Check nesting happened
        root = tree.sections[0]
        assert len(root.children) >= 1

    def test_heading_without_number(self, parser):
        text = """\
## Introduction

Some introduction text.

## Methods

Some methods text.
"""
        tree = parser.parse(text)
        assert len(tree.sections) >= 2
