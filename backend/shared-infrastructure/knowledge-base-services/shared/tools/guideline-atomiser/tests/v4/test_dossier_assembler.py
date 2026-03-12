"""
Tests for Dossier Assembler.

Validates:
1. Drug anchor identification (Channel B rxnorm + match_type exact/class)
2. Signal-to-drug association (row_drug, section co-location, proximity, global nearest)
3. Per-drug dossier building (dedup, sections, pages, source text, signal summary)
4. Edge cases (no drug anchors, single drug, all signals in one section)
5. Full pipeline integration with sample KDIGO fixtures
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.dossier_assembler import DossierAssembler
from extraction.v4.models import (
    DrugDossier,
    GuidelineSection,
    GuidelineTree,
    TableBoundary,
    VerifiedSpan,
)


@pytest.fixture
def assembler():
    return DossierAssembler()


@pytest.fixture
def simple_tree():
    return GuidelineTree(
        sections=[
            GuidelineSection(
                section_id="1",
                heading="Test Section",
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


class TestDrugAnchorIdentification:
    """Test identification of drug anchor spans."""

    def test_rxnorm_candidate_is_anchor(self, assembler):
        spans = [
            VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"],
                extraction_context={"channel_B_rxnorm_candidate": "860975"},
            ),
        ]
        anchors = assembler._find_drug_anchors(spans)
        assert len(anchors) == 1
        assert anchors[0][0] == "metformin"

    def test_exact_match_type_is_anchor(self, assembler):
        spans = [
            VerifiedSpan(
                text="dapagliflozin", start=50, end=63, confidence=1.0,
                contributing_channels=["B"],
                extraction_context={"match_type": "exact"},
            ),
        ]
        anchors = assembler._find_drug_anchors(spans)
        assert len(anchors) == 1

    def test_class_match_type_is_anchor(self, assembler):
        spans = [
            VerifiedSpan(
                text="SGLT2 inhibitors", start=50, end=66, confidence=1.0,
                contributing_channels=["B"],
                extraction_context={"match_type": "class"},
            ),
        ]
        anchors = assembler._find_drug_anchors(spans)
        assert len(anchors) == 1

    def test_signal_span_not_anchor(self, assembler):
        spans = [
            VerifiedSpan(
                text="eGFR >= 30", start=100, end=110, confidence=0.95,
                contributing_channels=["C"],
                extraction_context={"channel_C_pattern": "egfr_threshold"},
            ),
        ]
        anchors = assembler._find_drug_anchors(spans)
        assert len(anchors) == 0

    def test_multiple_anchors(self, assembler):
        spans = [
            VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"],
                extraction_context={"channel_B_rxnorm_candidate": "860975"},
            ),
            VerifiedSpan(
                text="finerenone", start=200, end=210, confidence=1.0,
                contributing_channels=["B"],
                extraction_context={"channel_B_rxnorm_candidate": "2555902"},
            ),
        ]
        anchors = assembler._find_drug_anchors(spans)
        assert len(anchors) == 2
        drug_names = {a[0] for a in anchors}
        assert "metformin" in drug_names
        assert "finerenone" in drug_names


class TestSignalSpanIdentification:
    """Test filtering of non-drug signal spans."""

    def test_signal_spans_filtered(self, assembler):
        spans = [
            VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"],
                extraction_context={"channel_B_rxnorm_candidate": "860975"},
            ),
            VerifiedSpan(
                text="eGFR >= 30", start=100, end=110, confidence=0.95,
                contributing_channels=["C"],
                extraction_context={"channel_C_pattern": "egfr_threshold"},
            ),
            VerifiedSpan(
                text="contraindicated", start=150, end=165, confidence=0.92,
                contributing_channels=["C"],
                extraction_context={"channel_C_pattern": "contraindication"},
            ),
        ]
        signals = assembler._find_signal_spans(spans)
        assert len(signals) == 2
        texts = {s.text for s in signals}
        assert "metformin" not in texts
        assert "eGFR >= 30" in texts


class TestSignalAssociation:
    """Test signal-to-drug association logic."""

    def test_row_drug_direct_association(self, assembler, simple_tree):
        """Table cell row_drug gives highest priority association."""
        signal = VerifiedSpan(
            text="1000 mg", start=500, end=507, confidence=0.95,
            contributing_channels=["D"], section_id="1",
            extraction_context={"row_drug": "metformin"},
        )
        anchors = [
            ("metformin", VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"], section_id="1",
                extraction_context={"channel_B_rxnorm_candidate": "860975"},
            )),
            ("finerenone", VerifiedSpan(
                text="finerenone", start=200, end=210, confidence=1.0,
                contributing_channels=["B"], section_id="1",
                extraction_context={"channel_B_rxnorm_candidate": "2555902"},
            )),
        ]
        result = assembler._associate_signal(signal, anchors, simple_tree)
        assert result == ["metformin"]

    def test_same_section_single_drug(self, assembler, simple_tree):
        """Signal in same section as exactly one drug → that drug."""
        signal = VerifiedSpan(
            text="eGFR >= 30", start=100, end=110, confidence=0.95,
            contributing_channels=["C"], section_id="sec_A",
            extraction_context={},
        )
        anchors = [
            ("metformin", VerifiedSpan(
                text="metformin", start=80, end=89, confidence=1.0,
                contributing_channels=["B"], section_id="sec_A",
                extraction_context={},
            )),
            ("finerenone", VerifiedSpan(
                text="finerenone", start=500, end=510, confidence=1.0,
                contributing_channels=["B"], section_id="sec_B",
                extraction_context={},
            )),
        ]
        result = assembler._associate_signal(signal, anchors, simple_tree)
        assert result == ["metformin"]

    def test_same_section_multiple_drugs_proximity(self, assembler, simple_tree):
        """Multiple drugs in section, closest within 200 chars → that one drug."""
        signal = VerifiedSpan(
            text="eGFR >= 30", start=100, end=110, confidence=0.95,
            contributing_channels=["C"], section_id="sec_A",
            extraction_context={},
        )
        # metformin is 20 chars away, finerenone is 150 chars away
        anchors = [
            ("metformin", VerifiedSpan(
                text="metformin", start=80, end=89, confidence=1.0,
                contributing_channels=["B"], section_id="sec_A",
                extraction_context={},
            )),
            ("finerenone", VerifiedSpan(
                text="finerenone", start=250, end=260, confidence=1.0,
                contributing_channels=["B"], section_id="sec_A",
                extraction_context={},
            )),
        ]
        result = assembler._associate_signal(signal, anchors, simple_tree)
        # Closest drug (metformin, 20 chars) is within 200 threshold
        assert result == ["metformin"]

    def test_same_section_multiple_drugs_beyond_proximity(self, assembler, simple_tree):
        """Multiple drugs in section, closest is beyond 200 chars → all drugs."""
        signal = VerifiedSpan(
            text="monitor every 3 months", start=1000, end=1022, confidence=0.90,
            contributing_channels=["C"], section_id="sec_A",
            extraction_context={},
        )
        # Both drugs are >200 chars from signal
        anchors = [
            ("metformin", VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"], section_id="sec_A",
                extraction_context={},
            )),
            ("finerenone", VerifiedSpan(
                text="finerenone", start=50, end=60, confidence=1.0,
                contributing_channels=["B"], section_id="sec_A",
                extraction_context={},
            )),
        ]
        result = assembler._associate_signal(signal, anchors, simple_tree)
        # Both drugs are >200 chars away → associate with all
        assert len(result) == 2
        assert "metformin" in result
        assert "finerenone" in result

    def test_no_drugs_in_section_nearest_global(self, assembler, simple_tree):
        """No drugs in same section → nearest drug globally."""
        signal = VerifiedSpan(
            text="check potassium", start=500, end=515, confidence=0.90,
            contributing_channels=["C"], section_id="sec_C",
            extraction_context={},
        )
        anchors = [
            ("metformin", VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"], section_id="sec_A",
                extraction_context={},
            )),
            ("finerenone", VerifiedSpan(
                text="finerenone", start=480, end=490, confidence=1.0,
                contributing_channels=["B"], section_id="sec_B",
                extraction_context={},
            )),
        ]
        result = assembler._associate_signal(signal, anchors, simple_tree)
        # finerenone is closer (20 chars) vs metformin (490 chars)
        assert result == ["finerenone"]


class TestDossierBuilding:
    """Test per-drug dossier construction."""

    def test_dossier_drug_name(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(sample_verified_spans, sample_guideline_tree, sample_text)
        drug_names = {d.drug_name for d in dossiers}
        assert "metformin" in drug_names

    def test_dossier_count(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(sample_verified_spans, sample_guideline_tree, sample_text)
        # 3 drug anchors: metformin, Dapagliflozin, Finerenone
        assert len(dossiers) == 3

    def test_dossier_rxnorm(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(sample_verified_spans, sample_guideline_tree, sample_text)
        metformin_dossier = [d for d in dossiers if d.drug_name == "metformin"][0]
        assert metformin_dossier.rxnorm_candidate == "860975"

    def test_dossier_spans_include_signals(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(sample_verified_spans, sample_guideline_tree, sample_text)
        metformin_dossier = [d for d in dossiers if d.drug_name == "metformin"][0]
        span_texts = {s.text for s in metformin_dossier.verified_spans}
        # metformin itself + eGFR >= 30 + contraindicated... (all in section 4.1.1)
        assert "metformin" in span_texts
        assert "eGFR >= 30" in span_texts

    def test_dossier_source_sections(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(sample_verified_spans, sample_guideline_tree, sample_text)
        metformin_dossier = [d for d in dossiers if d.drug_name == "metformin"][0]
        assert "4.1.1" in metformin_dossier.source_sections

    def test_dossier_signal_summary(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(sample_verified_spans, sample_guideline_tree, sample_text)
        metformin_dossier = [d for d in dossiers if d.drug_name == "metformin"][0]
        # Should have drug_anchor from metformin itself + channel_C patterns
        assert isinstance(metformin_dossier.signal_summary, dict)
        assert len(metformin_dossier.signal_summary) > 0

    def test_dossier_deduplication(self, assembler, simple_tree):
        """Duplicate spans should be deduplicated in the dossier."""
        spans = [
            VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"],
                extraction_context={"channel_B_rxnorm_candidate": "860975"},
                section_id="1",
            ),
            # Duplicate span (same start, end, text)
            VerifiedSpan(
                text="metformin", start=10, end=19, confidence=0.90,
                contributing_channels=["E"],
                extraction_context={"channel_B_rxnorm_candidate": "860975"},
                section_id="1",
            ),
        ]
        text = " " * 100  # padding
        dossiers = assembler.assemble(spans, simple_tree, text)
        assert len(dossiers) == 1
        # Should have deduplicated the duplicate span
        assert len(dossiers[0].verified_spans) == 1


class TestEdgeCases:
    """Test edge cases and boundary conditions."""

    def test_no_drug_anchors_returns_empty(self, assembler, simple_tree):
        """No drug anchors → no dossiers."""
        spans = [
            VerifiedSpan(
                text="eGFR >= 30", start=100, end=110, confidence=0.95,
                contributing_channels=["C"],
                extraction_context={"channel_C_pattern": "egfr_threshold"},
            ),
        ]
        text = " " * 200
        dossiers = assembler.assemble(spans, simple_tree, text)
        assert len(dossiers) == 0

    def test_empty_spans_returns_empty(self, assembler, simple_tree):
        """No spans at all → no dossiers."""
        dossiers = assembler.assemble([], simple_tree, "some text")
        assert len(dossiers) == 0

    def test_single_drug_only(self, assembler, simple_tree):
        """Single drug anchor with no signals → dossier with just the anchor."""
        spans = [
            VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"], section_id="1",
                extraction_context={"channel_B_rxnorm_candidate": "860975"},
            ),
        ]
        text = " " * 100
        dossiers = assembler.assemble(spans, simple_tree, text)
        assert len(dossiers) == 1
        assert dossiers[0].drug_name == "metformin"
        assert len(dossiers[0].verified_spans) == 1

    def test_dossier_source_text_extracted(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        """Source text should be extracted from section ranges."""
        dossiers = assembler.assemble(sample_verified_spans, sample_guideline_tree, sample_text)
        metformin_dossier = [d for d in dossiers if d.drug_name == "metformin"][0]
        # Source text should contain the recommendation text
        assert len(metformin_dossier.source_text) > 0


class TestSourceTextExtraction:
    """Test _get_source_text and _flatten_sections."""

    def test_flatten_nested_sections(self, assembler):
        """Nested sections should be flattened."""
        tree = GuidelineTree(
            sections=[
                GuidelineSection(
                    section_id="1", heading="Root",
                    start_offset=0, end_offset=100,
                    page_number=1, block_type="heading",
                    children=[
                        GuidelineSection(
                            section_id="1.1", heading="Child",
                            start_offset=10, end_offset=50,
                            page_number=1, block_type="paragraph",
                            children=[],
                        ),
                    ],
                ),
            ],
            tables=[], total_pages=1,
        )
        flat = assembler._flatten_sections(tree.sections)
        assert len(flat) == 2
        ids = [s.section_id for s in flat]
        assert "1" in ids
        assert "1.1" in ids

    def test_source_text_from_section(self, assembler, sample_guideline_tree, sample_text):
        """Source text should match the section's text range."""
        source = assembler._get_source_text(
            ["4.1.1"], sample_guideline_tree, sample_text
        )
        assert "metformin" in source
        assert "eGFR >= 30" in source


class TestSignalSummary:
    """Test _summarize_signals method."""

    def test_channel_c_patterns_counted(self, assembler):
        spans = [
            VerifiedSpan(
                text="eGFR >= 30", start=100, end=110, confidence=0.95,
                contributing_channels=["C"],
                extraction_context={"channel_C_pattern": "egfr_threshold"},
            ),
            VerifiedSpan(
                text="every 3-6 months", start=200, end=216, confidence=0.90,
                contributing_channels=["C"],
                extraction_context={"channel_C_pattern": "monitoring_frequency"},
            ),
        ]
        summary = assembler._summarize_signals(spans)
        assert summary["egfr_threshold"] == 1
        assert summary["monitoring_frequency"] == 1

    def test_drug_anchors_counted(self, assembler):
        spans = [
            VerifiedSpan(
                text="metformin", start=10, end=19, confidence=1.0,
                contributing_channels=["B"],
                extraction_context={"channel_B_rxnorm_candidate": "860975"},
            ),
        ]
        summary = assembler._summarize_signals(spans)
        assert summary["drug_anchor"] == 1

    def test_table_cells_counted(self, assembler):
        spans = [
            VerifiedSpan(
                text="1000 mg", start=500, end=507, confidence=0.95,
                contributing_channels=["D"],
                extraction_context={"row_drug": "metformin"},
            ),
        ]
        summary = assembler._summarize_signals(spans)
        assert summary["table_cell"] == 1

    def test_propositions_counted(self, assembler):
        spans = [
            VerifiedSpan(
                text="metformin should be reduced", start=100, end=126, confidence=0.85,
                contributing_channels=["F"],
                extraction_context={"extraction_method": "nuextract"},
            ),
        ]
        summary = assembler._summarize_signals(spans)
        assert summary["proposition"] == 1

    def test_other_category(self, assembler):
        spans = [
            VerifiedSpan(
                text="something", start=100, end=109, confidence=0.70,
                contributing_channels=["E"],
                extraction_context={"gliner_label": "unknown"},
            ),
        ]
        summary = assembler._summarize_signals(spans)
        assert summary["other"] == 1


class TestFullPipelineIntegration:
    """Test complete assembly with sample KDIGO fixtures."""

    def test_full_assembly(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(
            sample_verified_spans, sample_guideline_tree, sample_text
        )
        # 3 drug anchors: metformin, Dapagliflozin, Finerenone
        assert len(dossiers) == 3

        drug_names = {d.drug_name for d in dossiers}
        assert "metformin" in drug_names
        assert "Dapagliflozin" in drug_names
        assert "Finerenone" in drug_names

    def test_metformin_gets_signals(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(
            sample_verified_spans, sample_guideline_tree, sample_text
        )
        metformin_dossier = [d for d in dossiers if d.drug_name == "metformin"][0]
        # metformin is in 4.1.1, along with eGFR >= 30 and contraindication
        assert len(metformin_dossier.verified_spans) >= 2

    def test_each_dossier_has_source_text(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(
            sample_verified_spans, sample_guideline_tree, sample_text
        )
        for dossier in dossiers:
            # Each dossier should have non-empty source text if sections are valid
            assert isinstance(dossier.source_text, str)

    def test_each_dossier_has_signal_summary(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(
            sample_verified_spans, sample_guideline_tree, sample_text
        )
        for dossier in dossiers:
            assert isinstance(dossier.signal_summary, dict)

    def test_all_dossiers_are_drugdossier(self, assembler, sample_verified_spans, sample_guideline_tree, sample_text):
        dossiers = assembler.assemble(
            sample_verified_spans, sample_guideline_tree, sample_text
        )
        for dossier in dossiers:
            assert isinstance(dossier, DrugDossier)
