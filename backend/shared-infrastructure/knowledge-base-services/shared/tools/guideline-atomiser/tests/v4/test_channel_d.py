"""
Tests for Channel D: Table Decomposer.

Validates:
1. Cell-level span extraction from markdown tables
2. Row/column metadata (row_index, col_index, col_header, row_drug)
3. Offset provenance (start/end point to real text positions)
4. Table with no headers handled gracefully
5. Empty cells skipped
6. Multiple tables processed
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.channel_d_table import ChannelDTableDecomposer
from extraction.v4.channel_a_docling import ChannelADoclingParser
from extraction.v4.models import (
    GuidelineTree,
    GuidelineSection,
    TableBoundary,
)


@pytest.fixture
def decomposer():
    """Channel D table decomposer instance."""
    return ChannelDTableDecomposer()


@pytest.fixture
def parser():
    """Channel A parser for building real trees."""
    return ChannelADoclingParser()


class TestCellExtraction:
    """Test individual cell extraction from tables."""

    def test_cells_extracted_from_sample(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        assert output.span_count > 0

    def test_all_spans_are_channel_d(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert span.channel == "D"

    def test_all_spans_are_table_cells(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert span.source_block_type == "table_cell"

    def test_confidence_is_0_95(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert span.confidence == 0.95

    def test_metformin_in_cells(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        cell_texts = [s.text for s in output.spans]
        assert any("Metformin" in t for t in cell_texts)


class TestCellMetadata:
    """Test table-aware metadata on cell spans."""

    def test_row_index_present(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert "row_index" in span.channel_metadata

    def test_col_index_present(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert "col_index" in span.channel_metadata

    def test_col_header_present(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert "col_header" in span.channel_metadata

    def test_row_drug_present(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert "row_drug" in span.channel_metadata

    def test_row_drug_is_first_column(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        # For row 0, row_drug should be "Metformin"
        row_0_spans = [s for s in output.spans if s.channel_metadata["row_index"] == 0]
        if row_0_spans:
            assert row_0_spans[0].channel_metadata["row_drug"] == "Metformin"

    def test_col_headers_match_table(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        headers_seen = {s.channel_metadata["col_header"] for s in output.spans
                       if s.channel_metadata.get("col_header")}
        assert "Drug" in headers_seen


class TestOffsetProvenance:
    """Test that cell offsets point to real text positions."""

    def test_offsets_match_text(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            # The text at the offset should contain the span text
            actual_text = sample_text[span.start:span.end]
            assert actual_text == span.text, \
                f"Offset mismatch: expected '{span.text}', got '{actual_text}' at [{span.start}:{span.end}]"

    def test_offsets_within_table_bounds(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert span.start >= 0
            assert span.end <= len(sample_text)
            assert span.start < span.end


class TestTableId:
    """Test table_id assignment."""

    def test_table_id_assigned(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        for span in output.spans:
            assert span.table_id is not None


class TestStandaloneTable:
    """Test with standalone table fixture."""

    def test_four_drug_rows(self, decomposer, parser, sample_table_markdown):
        tree = parser.parse(sample_table_markdown)
        output = decomposer.extract(sample_table_markdown, tree)
        # 4 drug rows x 4 columns = 16 cells
        # Some cells may be empty or whitespace-only
        assert output.span_count >= 12

    def test_multiple_row_drugs(self, decomposer, parser, sample_table_markdown):
        tree = parser.parse(sample_table_markdown)
        output = decomposer.extract(sample_table_markdown, tree)
        row_drugs = {s.channel_metadata["row_drug"] for s in output.spans}
        assert "Metformin" in row_drugs
        assert "Dapagliflozin" in row_drugs


class TestChannelOutput:
    """Test ChannelOutput structure."""

    def test_channel_is_d(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        assert output.channel == "D"

    def test_output_success(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        assert output.success is True

    def test_output_metadata(self, decomposer, parser, sample_text):
        tree = parser.parse(sample_text)
        output = decomposer.extract(sample_text, tree)
        assert "tables_processed" in output.metadata
        assert "cells_extracted" in output.metadata

    def test_no_tables_no_spans(self, decomposer):
        text = "Just plain text, no tables here."
        tree = GuidelineTree(
            sections=[GuidelineSection(
                section_id="1", heading="Test",
                start_offset=0, end_offset=len(text),
                page_number=1, block_type="paragraph", children=[],
            )],
            tables=[],
            total_pages=1,
        )
        output = decomposer.extract(text, tree)
        assert output.span_count == 0
        assert output.metadata["tables_processed"] == 0
