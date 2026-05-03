"""V5 Table Specialist unit tests.

Run from guideline-atomiser/:
    PYTHONPATH=. V5_TABLE_SPECIALIST=1 pytest tests/v5/test_v5_table_specialist.py -v
"""
from __future__ import annotations

import csv
import os
from pathlib import Path

import pytest

from marker_extractor import TableBlock

try:
    from marker_extractor import BoundingBox as MBBox
    _BBOX = MBBox(x=10.0, y=20.0, width=400.0, height=120.0)
except Exception:
    _BBOX = None

from extraction.v4.channel_d_table import ChannelDTableDecomposer
from extraction.v4.models import GuidelineTree, TableBoundary


GOLDEN_DIR = Path(__file__).parent / "golden" / "tables"


def _make_minimal_tree() -> GuidelineTree:
    return GuidelineTree(sections=[], tables=[], total_pages=3)


def test_v4_path_used_when_flag_off(monkeypatch):
    """When V5_TABLE_SPECIALIST is absent, extract() uses V4 paths."""
    monkeypatch.delenv("V5_TABLE_SPECIALIST", raising=False)
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "eGFR", "Action"],
        rows=[["Metformin", "≥30 mL/min", "Continue"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    assert out.channel == "D"
    assert not out.metadata.get("table_specialist_used")


def test_docling_path_used_when_flag_on(monkeypatch):
    """When V5_TABLE_SPECIALIST=1 and l1_tables provided, Docling path is used."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "eGFR", "Action"],
        rows=[["Metformin", "≥30 mL/min", "Continue"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    assert out.metadata.get("table_specialist_used") is True
    assert len(out.spans) >= 3


def test_disable_all_overrides_table_specialist(monkeypatch):
    """V5_DISABLE_ALL=1 forces V4 path even if V5_TABLE_SPECIALIST=1."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.setenv("V5_DISABLE_ALL", "1")

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "eGFR", "Action"],
        rows=[["Metformin", "≥30 mL/min", "Continue"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    assert not out.metadata.get("table_specialist_used")


def test_docling_table_cells_match_golden(monkeypatch):
    """Cells extracted from a 3-row×3-col TableBlock match golden CSV."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "eGFR", "Action"],
        rows=[
            ["Metformin", "≥30 mL/min", "Continue"],
            ["SGLT2i", "<45 mL/min", "Reduce dose"],
            ["GLP-1 RA", "Any eGFR", "Continue"],
        ],
        page_number=2,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])

    golden: dict[tuple[int, int], str] = {}
    with open(GOLDEN_DIR / "sample_table_01.csv") as f:
        for row in csv.DictReader(f):
            golden[(int(row["row_idx"]), int(row["col_idx"]))] = row["expected_text"]

    extracted: dict[tuple[int, int], str] = {}
    for span in out.spans:
        if span.source_block_type == "table_cell":
            r = span.channel_metadata.get("row_index", -1)
            c = span.channel_metadata.get("col_index", -1)
            extracted[(r, c)] = span.text.strip()

    correct = sum(
        1 for (r, c), expected in golden.items()
        if extracted.get((r, c), "").lower() == expected.lower()
    )
    accuracy = correct / len(golden) * 100
    assert accuracy >= 85.0, f"Cell accuracy {accuracy:.1f}% < 85% threshold"


def test_header_detection(monkeypatch):
    """Headers row col_header metadata is set correctly on data spans."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["Drug", "Dose", "Frequency"],
        rows=[["Metformin", "500 mg", "BD"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    data_spans = [s for s in out.spans if s.source_block_type == "table_cell"
                  and s.channel_metadata.get("row_index", 0) >= 0]
    headers_seen = {s.channel_metadata.get("col_header") for s in data_spans}
    assert "Drug" in headers_seen
    assert "Dose" in headers_seen
    assert "Frequency" in headers_seen


def test_page_number_propagated(monkeypatch):
    """Page number from TableBlock is propagated to every cell RawSpan."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["A"],
        rows=[["val1"], ["val2"]],
        page_number=7,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    for span in out.spans:
        if span.source_block_type == "table_cell":
            assert span.page_number == 7


def test_empty_cells_skipped(monkeypatch):
    """Empty cells are not emitted as spans."""
    monkeypatch.setenv("V5_TABLE_SPECIALIST", "1")
    monkeypatch.delenv("V5_DISABLE_ALL", raising=False)

    decomp = ChannelDTableDecomposer()
    tree = _make_minimal_tree()
    tb = TableBlock(
        headers=["A", "B"],
        rows=[["val", ""], ["", "val2"]],
        page_number=1,
        confidence=1.0,
        table_index=0,
    )
    out = decomp.extract("", tree, l1_tables=[tb])
    texts = [s.text for s in out.spans if s.source_block_type == "table_cell"]
    assert "" not in texts
