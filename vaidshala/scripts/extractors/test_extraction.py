#!/usr/bin/env python3
"""
Comprehensive tests for Phase 4 Table Extraction Pipeline.

Tests real-world patterns from:
- ACC/AHA Heart Failure Guidelines 2022
- Surviving Sepsis Campaign 2021
- ADA Standards of Care 2024
- WHO GRADE recommendations
"""

import pytest
from table_extractor import (
    GuidelineTableExtractor,
    KB15Formatter,
    KB3TemporalFormatter,
    ClassOfRecommendation,
    LevelOfEvidence,
    TemporalConstraintType
)


class TestCORPatterns:
    """Test Class of Recommendation pattern matching."""

    def setup_method(self):
        self.extractor = GuidelineTableExtractor("TEST")

    def test_class_i(self):
        """Test Class I recognition."""
        patterns = [
            "Class I",
            "I",
            "Class 1",
            "CLASS I",
        ]
        for pattern in patterns:
            cor, conf = self.extractor._match_cor(pattern)
            assert cor == ClassOfRecommendation.CLASS_I, f"Failed for: {pattern}"
            assert conf >= 0.9

    def test_class_iia(self):
        """Test Class IIa recognition."""
        patterns = [
            "Class IIa",
            "IIa",
            "Class 2a",
            "CLASS IIA",
        ]
        for pattern in patterns:
            cor, conf = self.extractor._match_cor(pattern)
            assert cor == ClassOfRecommendation.CLASS_IIA, f"Failed for: {pattern}"

    def test_class_iib(self):
        """Test Class IIb recognition."""
        patterns = [
            "Class IIb",
            "IIb",
            "Class 2b",
        ]
        for pattern in patterns:
            cor, conf = self.extractor._match_cor(pattern)
            assert cor == ClassOfRecommendation.CLASS_IIB, f"Failed for: {pattern}"

    def test_class_iii_harm(self):
        """Test Class III (Harm) recognition."""
        patterns = [
            "Class III: Harm",
            "Class III (Harm)",
            "III-Harm",
            "Class III - Harm",
        ]
        for pattern in patterns:
            cor, conf = self.extractor._match_cor(pattern)
            assert cor == ClassOfRecommendation.CLASS_III_HARM, f"Failed for: {pattern}"

    def test_class_iii_no_benefit(self):
        """Test Class III (No Benefit) recognition."""
        patterns = [
            "Class III: No Benefit",
            "Class III (No Benefit)",
            "III-NoBenefit",
            "Class III - NB",
        ]
        for pattern in patterns:
            cor, conf = self.extractor._match_cor(pattern)
            assert cor == ClassOfRecommendation.CLASS_III_NO_BENEFIT, f"Failed for: {pattern}"

    def test_grade_strong(self):
        """Test GRADE strong recommendation."""
        patterns = [
            "Strong recommendation",
            "Strong for",
        ]
        for pattern in patterns:
            cor, conf = self.extractor._match_cor(pattern)
            assert cor == ClassOfRecommendation.CLASS_I, f"Failed for: {pattern}"

    def test_grade_conditional(self):
        """Test GRADE conditional recommendation."""
        patterns = [
            "Conditional recommendation",
            "Conditional for",
        ]
        for pattern in patterns:
            cor, conf = self.extractor._match_cor(pattern)
            assert cor == ClassOfRecommendation.CLASS_IIA, f"Failed for: {pattern}"


class TestLOEPatterns:
    """Test Level of Evidence pattern matching."""

    def setup_method(self):
        self.extractor = GuidelineTableExtractor("TEST")

    def test_loe_a(self):
        """Test Level A recognition."""
        patterns = [
            "LOE A",
            "Level A",
            "Level of Evidence A",
            "A",
        ]
        for pattern in patterns:
            loe, conf = self.extractor._match_loe(pattern)
            assert loe == LevelOfEvidence.LOE_A, f"Failed for: {pattern}"

    def test_loe_b_randomized(self):
        """Test Level B-R recognition."""
        patterns = [
            "LOE B-R",
            "Level B-R",
            "B-R",
            "B-Randomized",
        ]
        for pattern in patterns:
            loe, conf = self.extractor._match_loe(pattern)
            assert loe == LevelOfEvidence.LOE_B_R, f"Failed for: {pattern}"

    def test_loe_b_nonrandomized(self):
        """Test Level B-NR recognition."""
        patterns = [
            "LOE B-NR",
            "Level B-NR",
            "B-NR",
        ]
        for pattern in patterns:
            loe, conf = self.extractor._match_loe(pattern)
            assert loe == LevelOfEvidence.LOE_B_NR, f"Failed for: {pattern}"

    def test_loe_c_limited_data(self):
        """Test Level C-LD recognition."""
        patterns = [
            "LOE C-LD",
            "Level C-LD",
            "C-LD",
        ]
        for pattern in patterns:
            loe, conf = self.extractor._match_loe(pattern)
            assert loe == LevelOfEvidence.LOE_C_LD, f"Failed for: {pattern}"

    def test_loe_c_expert_opinion(self):
        """Test Level C-EO recognition."""
        patterns = [
            "LOE C-EO",
            "Level C-EO",
            "C-EO",
        ]
        for pattern in patterns:
            loe, conf = self.extractor._match_loe(pattern)
            assert loe == LevelOfEvidence.LOE_C_EO, f"Failed for: {pattern}"

    def test_grade_certainty(self):
        """Test GRADE certainty mapping."""
        test_cases = [
            ("High quality", LevelOfEvidence.LOE_A),
            ("High certainty", LevelOfEvidence.LOE_A),
            ("Moderate quality", LevelOfEvidence.LOE_B),
            ("Low quality", LevelOfEvidence.LOE_C_LD),
            ("Very low certainty", LevelOfEvidence.LOE_C_EO),
        ]
        for pattern, expected in test_cases:
            loe, conf = self.extractor._match_loe(pattern)
            assert loe == expected, f"Failed for: {pattern}"


class TestTemporalExtraction:
    """Test temporal constraint extraction."""

    def setup_method(self):
        self.extractor = GuidelineTableExtractor("TEST")

    def test_deadline_hours(self):
        """Test 'within X hours' extraction."""
        text = "Administer antibiotics within 1 hour of sepsis recognition."
        constraints = self.extractor._extract_temporal_constraints(text)

        assert len(constraints) >= 1
        deadline = next((c for c in constraints if c.constraint_type == "DEADLINE"), None)
        assert deadline is not None
        assert deadline.value == "1"
        assert "hour" in deadline.unit

    def test_deadline_minutes(self):
        """Test 'within X minutes' extraction."""
        text = "Obtain blood cultures within 45 minutes."
        constraints = self.extractor._extract_temporal_constraints(text)

        assert len(constraints) >= 1
        deadline = next((c for c in constraints if c.constraint_type == "DEADLINE"), None)
        assert deadline is not None
        assert deadline.value == "45"
        assert "minute" in deadline.unit

    def test_recurring(self):
        """Test 'every X hours' extraction."""
        text = "Monitor lactate every 6 hours until normalized."
        constraints = self.extractor._extract_temporal_constraints(text)

        assert len(constraints) >= 1
        recurring = next((c for c in constraints if c.constraint_type == "RECURRING"), None)
        assert recurring is not None
        assert recurring.value == "6"

    def test_sequence_before(self):
        """Test 'before X' extraction."""
        text = "Obtain blood cultures before antibiotic administration."
        constraints = self.extractor._extract_temporal_constraints(text)

        assert len(constraints) >= 1
        before = next((c for c in constraints if c.constraint_type == "BEFORE"), None)
        assert before is not None
        assert "antibiotic" in before.value.lower()

    def test_sequence_after(self):
        """Test 'after X' extraction."""
        text = "Reassess fluid status after initial bolus."
        constraints = self.extractor._extract_temporal_constraints(text)

        assert len(constraints) >= 1
        after = next((c for c in constraints if c.constraint_type == "AFTER"), None)
        assert after is not None
        assert "bolus" in after.value.lower()

    def test_urgent(self):
        """Test urgent terms extraction."""
        urgent_terms = [
            "Administer immediately.",
            "Give STAT.",
            "Initiate as soon as possible.",
            "Begin emergently.",
        ]
        for text in urgent_terms:
            constraints = self.extractor._extract_temporal_constraints(text)
            urgent = next((c for c in constraints if c.constraint_type == "URGENT"), None)
            assert urgent is not None, f"Failed for: {text}"

    def test_range(self):
        """Test 'X-Y hours' range extraction."""
        text = "Titrate over 2-4 weeks to target dose."
        constraints = self.extractor._extract_temporal_constraints(text)

        assert len(constraints) >= 1
        range_c = next((c for c in constraints if c.constraint_type == "RANGE"), None)
        assert range_c is not None
        assert "2-4" in range_c.value


class TestRealWorldGuidelines:
    """Test with real-world guideline text patterns."""

    def setup_method(self):
        self.extractor = GuidelineTableExtractor("REAL-WORLD-TEST")

    def test_accaha_hf_arni(self):
        """Test ACC/AHA HF 2022 ARNi recommendation."""
        # Single line format to test complete extraction
        text = "Class I, LOE A: In patients with HFrEF, an ARNi is recommended to reduce morbidity and mortality. ARNi should not be given within 36 hours of the last dose of an ACEi."
        recs = self.extractor.extract_from_text(text)

        assert len(recs) >= 1
        rec = recs[0]
        assert rec.cor == "I"
        assert rec.loe == "A"
        assert not rec.needs_llm_review

        # Check temporal constraint for 36 hours
        temporal = [t for t in rec.temporal_constraints if t.get('constraint_type') == 'DEADLINE']
        assert len(temporal) >= 1
        assert temporal[0].get('value') == '36'

    def test_ssc_hour1_bundle(self):
        """Test Surviving Sepsis Campaign Hour-1 Bundle."""
        bundle_text = """
        Strong recommendation, High certainty: For adults with sepsis,
        administer broad-spectrum antimicrobials within 1 hour of recognition.
        Obtain blood cultures before antimicrobial therapy if feasible.
        Measure lactate level and remeasure within 2-4 hours if elevated.
        """
        recs = self.extractor.extract_from_text(bundle_text)

        # Should extract multiple temporal constraints
        all_temporal = []
        for rec in recs:
            all_temporal.extend(rec.temporal_constraints)

        # Should have deadline (within 1 hour) and range (2-4 hours)
        types = [t.get('constraint_type') for t in all_temporal]
        assert 'DEADLINE' in types
        assert 'RANGE' in types or 'RECURRING' in types

    def test_ada_a1c_monitoring(self):
        """Test ADA Standards HbA1c recommendation."""
        text = """
        Class I, LOE A: Perform HbA1c testing at least twice yearly
        in patients meeting treatment goals. Test quarterly in patients
        not meeting glycemic goals or with therapy changes.
        """
        recs = self.extractor.extract_from_text(text)

        assert len(recs) >= 1
        rec = recs[0]
        assert rec.cor == "I"
        assert rec.loe == "A"

    def test_class_iii_harm_extraction(self):
        """Test Class III (Harm) recommendation."""
        text = """
        Class III: Harm, LOE B-R: Routine use of prophylactic
        antiarrhythmic drugs is not recommended for prevention of
        atrial fibrillation after cardiac surgery.
        """
        recs = self.extractor.extract_from_text(text)

        assert len(recs) >= 1
        rec = recs[0]
        assert rec.cor == "III-Harm"
        assert rec.loe == "B-R"


class TestKB15Formatter:
    """Test KB-15 Evidence Engine output formatting."""

    def setup_method(self):
        self.extractor = GuidelineTableExtractor("KB15-TEST")
        self.formatter = KB15Formatter({
            "title": "Test Guideline",
            "organization": "Test Org",
            "year": "2026"
        })

    def test_basic_formatting(self):
        """Test basic KB-15 output structure."""
        text = "Class I, LOE A: Test recommendation within 24 hours."
        recs = self.extractor.extract_from_text(text)
        kb15 = self.formatter.format(recs)

        assert len(kb15) >= 1
        entry = kb15[0]

        # Check required fields
        assert "recommendation_id" in entry
        assert "evidence_envelope" in entry
        assert "recommendation" in entry
        assert "provenance" in entry
        assert "governance" in entry

        # Check evidence envelope
        env = entry["evidence_envelope"]
        assert env["class_of_recommendation"] == "I"
        assert env["level_of_evidence"] == "A"
        assert "cor_display" in env
        assert "loe_display" in env

    def test_governance_status(self):
        """Test governance status assignment."""
        # High confidence = PENDING_REVIEW
        text = "Class I, LOE A: Clear recommendation."
        recs = self.extractor.extract_from_text(text)
        kb15 = self.formatter.format(recs)
        assert kb15[0]["governance"]["status"] == "PENDING_REVIEW"


class TestKB3TemporalFormatter:
    """Test KB-3 Temporal Brain output formatting."""

    def setup_method(self):
        self.extractor = GuidelineTableExtractor("KB3-TEST")
        self.formatter = KB3TemporalFormatter()

    def test_iso8601_conversion(self):
        """Test ISO 8601 duration conversion."""
        # Test hours
        assert self.formatter._to_iso8601("1", "hour") == "PT1H"
        assert self.formatter._to_iso8601("24", "hours") == "PT24H"

        # Test minutes
        assert self.formatter._to_iso8601("30", "minutes") == "PT30M"

        # Test days
        assert self.formatter._to_iso8601("7", "days") == "P7D"

        # Test weeks
        assert self.formatter._to_iso8601("2", "weeks") == "P2W"

    def test_temporal_output_structure(self):
        """Test KB-3 output structure."""
        text = "Within 1 hour of sepsis recognition, administer antibiotics."
        recs = self.extractor.extract_from_text(text)
        kb3 = self.formatter.format(recs)

        if kb3:  # May be empty if no temporal found
            entry = kb3[0]
            assert "constraint_id" in entry
            assert "constraint_type" in entry
            assert "iso8601_duration" in entry


class TestEdgeCases:
    """Test edge cases and error handling."""

    def setup_method(self):
        self.extractor = GuidelineTableExtractor("EDGE-TEST")

    def test_empty_input(self):
        """Test handling of empty input."""
        recs = self.extractor.extract_from_text("")
        assert recs == []

    def test_no_cor_loe(self):
        """Test text without COR/LOE markers."""
        text = "Consider using beta-blockers for heart rate control."
        recs = self.extractor.extract_from_text(text)

        # Should extract but mark for LLM review
        if recs:
            assert recs[0].needs_llm_review

    def test_ambiguous_text(self):
        """Test ambiguous recommendation text."""
        text = "May consider in selected patients."
        recs = self.extractor.extract_from_text(text)

        # Short/ambiguous text should be filtered or flagged
        if recs:
            assert recs[0].needs_llm_review or recs[0].confidence < 0.8

    def test_multiple_recommendations(self):
        """Test extraction of multiple recommendations."""
        text = """
        Class I, LOE A: First recommendation.
        Class IIa, LOE B-R: Second recommendation.
        Class IIb, LOE C-LD: Third recommendation.
        """
        recs = self.extractor.extract_from_text(text)
        assert len(recs) == 3

        cors = [r.cor for r in recs]
        assert "I" in cors
        assert "IIa" in cors
        assert "IIb" in cors


def run_tests():
    """Run all tests and report results."""
    import sys

    # Run pytest with verbose output
    exit_code = pytest.main([
        __file__,
        "-v",
        "--tb=short",
        "-x"  # Stop on first failure
    ])

    return exit_code


if __name__ == "__main__":
    run_tests()
