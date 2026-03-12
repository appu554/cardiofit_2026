"""
Tests for Channel C: Grammar/Regex Pattern Extractor.

Validates:
1. eGFR threshold detection (eGFR < 30, eGFR >= 30)
2. eGFR range detection (eGFR 30-45)
3. Monitoring frequency detection (every 3-6 months, Q3-6mo)
4. Contraindication markers (contraindicated, avoid)
5. Lab test identification (eGFR, potassium, HbA1c)
6. Recommendation ID detection (Recommendation 4.1.1)
7. Dose value detection (10 mg daily, 1000 mg)
8. Potassium threshold detection (K+ > 5.5 mEq/L)
9. Evidence grade detection (Grade 1A, strong recommendation)
10. Overlap prevention (>50% overlap skips lower-priority)
11. Drug-agnostic extraction (no drug names in output)
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.channel_c_grammar import ChannelCGrammar
from extraction.v4.models import GuidelineTree, GuidelineSection, TableBoundary


@pytest.fixture
def grammar():
    """Channel C grammar extractor instance."""
    return ChannelCGrammar()


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


class TestEGFRThresholds:
    """Test eGFR threshold pattern matching."""

    def test_egfr_gte_30(self, grammar, simple_tree):
        text = "Use metformin when eGFR >= 30 mL/min/1.73m²"
        output = grammar.extract(text, simple_tree)
        egfr_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "egfr_threshold"]
        assert len(egfr_spans) >= 1

    def test_egfr_less_than(self, grammar, simple_tree):
        text = "Contraindicated when eGFR < 30"
        output = grammar.extract(text, simple_tree)
        egfr_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "egfr_threshold"]
        assert len(egfr_spans) >= 1

    def test_egfr_greater_than_text(self, grammar, simple_tree):
        text = "eGFR greater than 45 mL/min"
        output = grammar.extract(text, simple_tree)
        egfr_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "egfr_threshold"]
        assert len(egfr_spans) >= 1

    def test_crcl_threshold(self, grammar, simple_tree):
        text = "CrCl < 30 mL/min indicates severe impairment"
        output = grammar.extract(text, simple_tree)
        egfr_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "egfr_threshold"]
        assert len(egfr_spans) >= 1


class TestEGFRRanges:
    """Test eGFR range pattern matching."""

    def test_egfr_range_30_45(self, grammar, simple_tree):
        text = "For patients with eGFR 30-45 mL/min/1.73m²"
        output = grammar.extract(text, simple_tree)
        range_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "egfr_range"]
        assert len(range_spans) >= 1


class TestMonitoringFrequency:
    """Test monitoring frequency pattern matching."""

    def test_every_3_6_months(self, grammar, simple_tree):
        text = "Monitor eGFR every 3-6 months"
        output = grammar.extract(text, simple_tree)
        freq_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "monitoring_frequency"]
        assert len(freq_spans) >= 1

    def test_q3_6mo(self, grammar, simple_tree):
        text = "K+ monitoring Q3-6mo after initiation"
        output = grammar.extract(text, simple_tree)
        freq_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "monitoring_frequency"]
        assert len(freq_spans) >= 1

    def test_at_baseline(self, grammar, simple_tree):
        text = "Monitor potassium at baseline"
        output = grammar.extract(text, simple_tree)
        freq_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "monitoring_frequency"]
        assert len(freq_spans) >= 1

    def test_annually(self, grammar, simple_tree):
        text = "Check lipids annually"
        output = grammar.extract(text, simple_tree)
        freq_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "monitoring_frequency"]
        assert len(freq_spans) >= 1

    def test_within_weeks_of_initiation(self, grammar, simple_tree):
        text = "within 4 weeks of initiation"
        output = grammar.extract(text, simple_tree)
        freq_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "monitoring_frequency"]
        assert len(freq_spans) >= 1

    def test_after_weeks(self, grammar, simple_tree):
        text = "Reassess after 2 weeks"
        output = grammar.extract(text, simple_tree)
        freq_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "monitoring_frequency"]
        assert len(freq_spans) >= 1


class TestContraindication:
    """Test contraindication marker detection."""

    def test_contraindicated(self, grammar, simple_tree):
        text = "Metformin is contraindicated in severe renal impairment"
        output = grammar.extract(text, simple_tree)
        contra_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "contraindication"]
        assert len(contra_spans) >= 1

    def test_avoid(self, grammar, simple_tree):
        text = "Avoid NSAIDs in CKD patients"
        output = grammar.extract(text, simple_tree)
        contra_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "contraindication"]
        assert len(contra_spans) >= 1

    def test_do_not_use(self, grammar, simple_tree):
        text = "Do not use in patients with eGFR < 15"
        output = grammar.extract(text, simple_tree)
        contra_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "contraindication"]
        assert len(contra_spans) >= 1

    def test_not_recommended(self, grammar, simple_tree):
        text = "This drug is not recommended for dialysis patients"
        output = grammar.extract(text, simple_tree)
        contra_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "contraindication"]
        assert len(contra_spans) >= 1


class TestActionMarkers:
    """Test action markers (discontinue, stop, hold)."""

    def test_discontinue(self, grammar, simple_tree):
        text = "Discontinue if potassium exceeds 6.0 mEq/L"
        output = grammar.extract(text, simple_tree)
        action_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "action_marker"]
        assert len(action_spans) >= 1

    def test_hold(self, grammar, simple_tree):
        text = "Hold finerenone if K+ is elevated"
        output = grammar.extract(text, simple_tree)
        action_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "action_marker"]
        assert len(action_spans) >= 1


class TestLabTests:
    """Test lab test name detection."""

    def test_egfr_detected(self, grammar, simple_tree):
        text = "Monitor eGFR regularly"
        output = grammar.extract(text, simple_tree)
        lab_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "lab_test"]
        assert len(lab_spans) >= 1

    def test_potassium_detected(self, grammar, simple_tree):
        text = "Check serum potassium levels"
        output = grammar.extract(text, simple_tree)
        # Could match as lab_test or potassium_threshold
        assert output.span_count >= 1

    def test_hba1c_detected(self, grammar, simple_tree):
        text = "Target HbA1c of less than 7%"
        output = grammar.extract(text, simple_tree)
        lab_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "lab_test"]
        assert len(lab_spans) >= 1


class TestRecommendationIDs:
    """Test recommendation ID extraction."""

    def test_recommendation_4_1_1(self, grammar, simple_tree):
        text = "See Recommendation 4.1.1 for details"
        output = grammar.extract(text, simple_tree)
        rec_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "recommendation_id"]
        assert len(rec_spans) == 1
        assert "4.1.1" in rec_spans[0].text

    def test_recommendation_high_confidence(self, grammar, simple_tree):
        text = "Recommendation 4.2.1 states that..."
        output = grammar.extract(text, simple_tree)
        rec_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "recommendation_id"]
        assert len(rec_spans) == 1
        assert rec_spans[0].confidence == 0.98


class TestDoseValues:
    """Test dose value extraction."""

    def test_mg_dose(self, grammar, simple_tree):
        text = "Start with 500 mg daily"
        output = grammar.extract(text, simple_tree)
        dose_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "dose_value"]
        assert len(dose_spans) >= 1

    def test_mg_per_day(self, grammar, simple_tree):
        text = "Maximum dose is 1000 mg/day"
        output = grammar.extract(text, simple_tree)
        dose_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "dose_value"]
        assert len(dose_spans) >= 1

    def test_mcg_dose(self, grammar, simple_tree):
        text = "Semaglutide 250 mcg weekly"
        output = grammar.extract(text, simple_tree)
        dose_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "dose_value"]
        assert len(dose_spans) >= 1


class TestPotassiumThresholds:
    """Test potassium threshold detection."""

    def test_potassium_gt_5_5(self, grammar, simple_tree):
        text = "Hold if potassium > 5.5 mEq/L"
        output = grammar.extract(text, simple_tree)
        k_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "potassium_threshold"]
        assert len(k_spans) >= 1

    def test_k_plus_threshold(self, grammar, simple_tree):
        text = "When K+ > 5.0 mmol/L, reduce dose"
        output = grammar.extract(text, simple_tree)
        k_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "potassium_threshold"]
        assert len(k_spans) >= 1


class TestEvidenceGrades:
    """Test evidence/recommendation grade detection."""

    def test_grade_1a(self, grammar, simple_tree):
        text = "This is a Grade 1A recommendation"
        output = grammar.extract(text, simple_tree)
        grade_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "evidence_grade"]
        assert len(grade_spans) >= 1

    def test_strong_recommendation(self, grammar, simple_tree):
        text = "This is a strong recommendation based on high quality evidence"
        output = grammar.extract(text, simple_tree)
        grade_spans = [s for s in output.spans if s.channel_metadata["pattern"] == "evidence_grade"]
        assert len(grade_spans) >= 1


class TestChannelOutput:
    """Test ChannelOutput structure."""

    def test_channel_is_c(self, grammar, sample_text, sample_guideline_tree):
        output = grammar.extract(sample_text, sample_guideline_tree)
        assert output.channel == "C"

    def test_output_success(self, grammar, sample_text, sample_guideline_tree):
        output = grammar.extract(sample_text, sample_guideline_tree)
        assert output.success is True

    def test_output_metadata(self, grammar, sample_text, sample_guideline_tree):
        output = grammar.extract(sample_text, sample_guideline_tree)
        assert "patterns_compiled" in output.metadata
        assert "matches_found" in output.metadata

    def test_all_spans_channel_c(self, grammar, sample_text, sample_guideline_tree):
        output = grammar.extract(sample_text, sample_guideline_tree)
        for span in output.spans:
            assert span.channel == "C"

    def test_spans_sorted_by_offset(self, grammar, sample_text, sample_guideline_tree):
        output = grammar.extract(sample_text, sample_guideline_tree)
        offsets = [s.start for s in output.spans]
        assert offsets == sorted(offsets)

    def test_drug_agnostic_no_drug_names(self, grammar, simple_tree):
        """Channel C should NOT extract drug names — that's Channel B's job."""
        text = "metformin dapagliflozin finerenone"
        output = grammar.extract(text, simple_tree)
        # None of the matches should be drug names
        drug_categories = {"drug_ingredient", "drug_class"}
        for span in output.spans:
            assert span.channel_metadata["pattern"] not in drug_categories


class TestOverlapPrevention:
    """Test that overlapping spans are handled correctly."""

    def test_no_duplicate_overlapping_spans(self, grammar, simple_tree):
        text = "eGFR >= 30 mL/min/1.73m²"
        output = grammar.extract(text, simple_tree)
        # Should not have multiple overlapping spans for the same text
        for i, s1 in enumerate(output.spans):
            for j, s2 in enumerate(output.spans):
                if i >= j:
                    continue
                # Check that no two spans have >50% overlap
                overlap_start = max(s1.start, s2.start)
                overlap_end = min(s1.end, s2.end)
                if overlap_start < overlap_end:
                    overlap_len = overlap_end - overlap_start
                    shorter = min(s1.end - s1.start, s2.end - s2.start)
                    assert overlap_len / shorter <= 0.5, \
                        f"Overlapping spans: '{s1.text}' and '{s2.text}'"


class TestFullTextExtraction:
    """Test extraction on the full KDIGO sample text."""

    def test_multiple_categories_found(self, grammar, sample_text, sample_guideline_tree):
        output = grammar.extract(sample_text, sample_guideline_tree)
        categories = {s.channel_metadata["pattern"] for s in output.spans}
        # Should find at least eGFR thresholds, monitoring, and dose values
        assert len(categories) >= 3

    def test_reasonable_span_count(self, grammar, sample_text, sample_guideline_tree):
        output = grammar.extract(sample_text, sample_guideline_tree)
        # The sample text has many clinical patterns
        assert output.span_count >= 5

    def test_elapsed_ms_recorded(self, grammar, sample_text, sample_guideline_tree):
        output = grammar.extract(sample_text, sample_guideline_tree)
        assert output.elapsed_ms > 0
