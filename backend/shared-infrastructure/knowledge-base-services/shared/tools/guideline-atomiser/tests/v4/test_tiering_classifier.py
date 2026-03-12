"""
Tests for Tiering Classifier (Phase 6).

Validates:
1. TieringClassifier ABC cannot be instantiated directly
2. RuleBasedTieringClassifier:
   - NOISE: short text, table artifacts, single Channel D, very low confidence
   - TIER_1: 3+ channel corroboration, 2-channel agreement, single B/C high confidence
   - TIER_2: disagreement, moderate confidence, fallback
3. TrainedTieringClassifier raises NotImplementedError
4. TieringResult structure
5. Integration with SignalMerger (tier field populated on MergedSpan)
"""

import sys
from pathlib import Path
from uuid import uuid4

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.tiering_classifier import (
    TieringClassifier,
    RuleBasedTieringClassifier,
    TrainedTieringClassifier,
    TieringResult,
    TierLabel,
)


# ── ABC Tests ──────────────────────────────────────────────────────────

class TestTieringClassifierABC:
    """Verify abstract base class behavior."""

    def test_cannot_instantiate_abc(self):
        with pytest.raises(TypeError):
            TieringClassifier()

    def test_tiering_result_fields(self):
        result = TieringResult(tier="TIER_1", confidence=0.95, reason="test")
        assert result.tier == "TIER_1"
        assert result.confidence == 0.95
        assert result.reason == "test"


# ── RuleBasedTieringClassifier: NOISE ──────────────────────────────────

class TestNoiseClassification:
    """Cases that should classify as NOISE."""

    def setup_method(self):
        self.classifier = RuleBasedTieringClassifier()

    def test_very_short_text(self):
        """Text ≤ 4 chars → NOISE."""
        result = self.classifier.classify(
            text="mg",
            merged_confidence=0.80,
            contributing_channels=["B"],
            channel_confidences={"B": 0.80},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"
        assert "too short" in result.reason.lower()

    def test_table_artifact_single_d(self):
        """Table artifact pattern with single Channel D → NOISE."""
        result = self.classifier.classify(
            text="Cell | Value | Data",
            merged_confidence=0.50,
            contributing_channels=["D"],
            channel_confidences={"D": 0.50},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"
        assert "table artifact" in result.reason.lower()

    def test_single_d_low_confidence(self):
        """Single Channel D with confidence < 0.6 → NOISE."""
        result = self.classifier.classify(
            text="Some table cell content",
            merged_confidence=0.45,
            contributing_channels=["D"],
            channel_confidences={"D": 0.45},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"
        assert "single channel d" in result.reason.lower()

    def test_very_low_confidence(self):
        """Confidence < 0.3 → NOISE."""
        result = self.classifier.classify(
            text="Some clinical text about dosing",
            merged_confidence=0.25,
            contributing_channels=["B"],
            channel_confidences={"B": 0.25},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"
        assert "very low confidence" in result.reason.lower()


# ── RuleBasedTieringClassifier: TIER_1 ─────────────────────────────────

class TestTier1Classification:
    """Cases that should classify as TIER_1."""

    def setup_method(self):
        self.classifier = RuleBasedTieringClassifier()

    def test_three_channel_corroboration(self):
        """3+ channels with confidence ≥ 0.7 → TIER_1."""
        result = self.classifier.classify(
            text="Reduce metformin dose when eGFR < 30",
            merged_confidence=0.85,
            contributing_channels=["B", "C", "D"],
            channel_confidences={"B": 0.90, "C": 0.85, "D": 0.70},
            has_disagreement=False,
        )
        assert result.tier == "TIER_1"
        assert "3-channel" in result.reason

    def test_four_channel_corroboration(self):
        """4 channels → TIER_1."""
        result = self.classifier.classify(
            text="Prescribe dapagliflozin 10 mg daily",
            merged_confidence=0.90,
            contributing_channels=["B", "C", "D", "E"],
            channel_confidences={"B": 0.95, "C": 0.90, "D": 0.80, "E": 0.75},
            has_disagreement=False,
        )
        assert result.tier == "TIER_1"

    def test_two_channel_agreement(self):
        """2 channels without disagreement, confidence ≥ 0.7 → TIER_1."""
        result = self.classifier.classify(
            text="Monitor potassium when using finerenone",
            merged_confidence=0.80,
            contributing_channels=["B", "C"],
            channel_confidences={"B": 0.85, "C": 0.75},
            has_disagreement=False,
        )
        assert result.tier == "TIER_1"
        assert "2-channel agreement" in result.reason

    def test_single_b_very_high_confidence(self):
        """Single Channel B with confidence ≥ 0.9 → TIER_1."""
        result = self.classifier.classify(
            text="lisinopril for renal protection",
            merged_confidence=0.92,
            contributing_channels=["B"],
            channel_confidences={"B": 0.92},
            has_disagreement=False,
        )
        assert result.tier == "TIER_1"
        assert "single primary channel" in result.reason.lower()

    def test_single_c_very_high_confidence(self):
        """Single Channel C with confidence ≥ 0.9 → TIER_1."""
        result = self.classifier.classify(
            text="eGFR < 30 mL/min/1.73m2",
            merged_confidence=0.95,
            contributing_channels=["C"],
            channel_confidences={"C": 0.95},
            has_disagreement=False,
        )
        assert result.tier == "TIER_1"


# ── RuleBasedTieringClassifier: TIER_2 ─────────────────────────────────

class TestTier1WithDisagreement:
    """Calibration 3: Disagreement inversion fix.

    Golden dataset shows has_disagreement=True has 57.4% precision vs 24.2% without.
    Multi-channel with disagreement at confidence ≥ 0.7 should be TIER_1.
    """

    def setup_method(self):
        self.classifier = RuleBasedTieringClassifier()

    def test_two_channel_disagreement_high_conf_is_tier1(self):
        """2 channels WITH disagreement, conf ≥ 0.7 → TIER_1 (57.4% precision)."""
        result = self.classifier.classify(
            text="Metformin 500 mg twice daily",
            merged_confidence=0.75,
            contributing_channels=["B", "C"],
            channel_confidences={"B": 0.80, "C": 0.70},
            has_disagreement=True,
        )
        assert result.tier == "TIER_1"
        assert "disagreement" in result.reason.lower()

    def test_two_channel_disagreement_low_conf_is_tier2(self):
        """2 channels WITH disagreement, conf < 0.7 → TIER_2."""
        result = self.classifier.classify(
            text="Metformin 500 mg twice daily",
            merged_confidence=0.55,
            contributing_channels=["B", "C"],
            channel_confidences={"B": 0.60, "C": 0.50},
            has_disagreement=True,
        )
        assert result.tier == "TIER_2"


class TestBareClassAbbrevNoise:
    """Calibration 1: Bare drug class abbreviations as single-channel spans.

    Golden dataset: ARB 0/70 confirmed, MRA 0/66, ACEi 1/79 (only at 4-channel).
    Single-channel bare abbreviations are overwhelmingly noise.
    """

    def setup_method(self):
        self.classifier = RuleBasedTieringClassifier()

    def test_arb_single_channel_is_noise(self):
        """Bare 'ARB' (3 chars) → NOISE via short-text gate."""
        result = self.classifier.classify(
            text="ARB",
            merged_confidence=0.90,
            contributing_channels=["B"],
            channel_confidences={"B": 0.90},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"
        # 3 chars ≤ MIN_CLINICAL_LENGTH, caught by short-text gate
        assert "too short" in result.reason.lower()

    def test_mra_single_channel_is_noise(self):
        """Bare 'MRA' (3 chars) → NOISE via short-text gate."""
        result = self.classifier.classify(
            text="MRA",
            merged_confidence=0.85,
            contributing_channels=["B"],
            channel_confidences={"B": 0.85},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"

    def test_acei_single_channel_is_noise(self):
        """Bare 'ACEi' (4 chars) → NOISE via short-text gate (≤ MIN_CLINICAL_LENGTH)."""
        result = self.classifier.classify(
            text="ACEi",
            merged_confidence=0.90,
            contributing_channels=["B"],
            channel_confidences={"B": 0.90},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"

    def test_sglt2_single_channel_is_noise(self):
        """Bare 'SGLT2' (5 chars) → NOISE via noise archetype (standalone_drug_name).
        The noise archetype check fires before the bare abbreviation check since
        Stage 1.2 added the 8-archetype taxonomy.
        """
        result = self.classifier.classify(
            text="SGLT2",
            merged_confidence=0.88,
            contributing_channels=["B"],
            channel_confidences={"B": 0.88},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"
        assert "standalone_drug_name" in result.reason.lower() or "bare class abbreviation" in result.reason.lower()

    def test_nsaid_single_channel_is_noise(self):
        """Bare 'NSAID' (5 chars) → NOISE via noise archetype (standalone_drug_name)."""
        result = self.classifier.classify(
            text="NSAID",
            merged_confidence=0.85,
            contributing_channels=["B"],
            channel_confidences={"B": 0.85},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"
        assert "standalone_drug_name" in result.reason.lower() or "bare class abbreviation" in result.reason.lower()

    def test_nsaids_single_channel_is_noise(self):
        """Bare 'NSAIDs' (6 chars) → NOISE via noise archetype (standalone_drug_name)."""
        result = self.classifier.classify(
            text="NSAIDs",
            merged_confidence=0.85,
            contributing_channels=["B"],
            channel_confidences={"B": 0.85},
            has_disagreement=False,
        )
        assert result.tier == "NOISE"
        assert "standalone_drug_name" in result.reason.lower() or "bare class abbreviation" in result.reason.lower()

    def test_sglt2_multi_channel_not_noise(self):
        """SGLT2 with 3-channel corroboration → NOT NOISE (multi-channel elevates)."""
        result = self.classifier.classify(
            text="SGLT2",
            merged_confidence=0.85,
            contributing_channels=["B", "C", "D"],
            channel_confidences={"B": 0.90, "C": 0.85, "D": 0.70},
            has_disagreement=False,
        )
        assert result.tier != "NOISE"

    def test_full_sentence_with_class_name_not_noise(self):
        """Full sentence containing 'ARB' is NOT bare abbreviation → not caught."""
        result = self.classifier.classify(
            text="ARB or ACEi should be prescribed for renal protection",
            merged_confidence=0.85,
            contributing_channels=["B"],
            channel_confidences={"B": 0.85},
            has_disagreement=False,
        )
        # Full sentence ≠ bare abbreviation, should NOT be NOISE
        assert result.tier != "NOISE"


class TestTier2Classification:
    """Cases that should classify as TIER_2."""

    def setup_method(self):
        self.classifier = RuleBasedTieringClassifier()

    def test_moderate_confidence(self):
        """Single channel with moderate confidence (0.5-0.89) → TIER_2."""
        result = self.classifier.classify(
            text="Consider dose adjustment in elderly patients",
            merged_confidence=0.65,
            contributing_channels=["C"],
            channel_confidences={"C": 0.65},
            has_disagreement=False,
        )
        assert result.tier == "TIER_2"
        assert "moderate confidence" in result.reason.lower()

    def test_fallback(self):
        """Low-but-not-very-low confidence single channel → TIER_2 fallback."""
        result = self.classifier.classify(
            text="Some partially matched clinical text content",
            merged_confidence=0.35,
            contributing_channels=["E"],
            channel_confidences={"E": 0.35},
            has_disagreement=False,
        )
        assert result.tier == "TIER_2"

    def test_single_d_moderate_confidence(self):
        """Single Channel D with decent confidence → TIER_2 (not NOISE)."""
        result = self.classifier.classify(
            text="Drug dosing table cell: metformin 1000mg",
            merged_confidence=0.65,
            contributing_channels=["D"],
            channel_confidences={"D": 0.65},
            has_disagreement=False,
        )
        assert result.tier == "TIER_2"

    def test_multi_channel_below_threshold(self):
        """2 channels but confidence < 0.7 → TIER_2."""
        result = self.classifier.classify(
            text="Some clinical text about medication dosing",
            merged_confidence=0.55,
            contributing_channels=["B", "C"],
            channel_confidences={"B": 0.60, "C": 0.50},
            has_disagreement=False,
        )
        assert result.tier == "TIER_2"


# ── TrainedTieringClassifier ──────────────────────────────────────────

class TestTrainedClassifier:
    """Verify trained classifier requires a valid model file."""

    def test_raises_file_not_found_for_missing_model(self):
        """TrainedTieringClassifier delegates to TierAssignerClassifier which
        requires an actual model file. Missing model raises FileNotFoundError."""
        with pytest.raises(FileNotFoundError):
            TrainedTieringClassifier(model_path="/some/model.pkl")


# ── Integration with SignalMerger ──────────────────────────────────────

class TestMergerIntegration:
    """Verify tiering classifier works within SignalMerger."""

    def test_tier_field_populated_on_merged_span(self):
        """When classifier is passed to merge(), MergedSpan.tier should be set."""
        from extraction.v4.signal_merger import SignalMerger
        from extraction.v4.models import ChannelOutput, RawSpan, GuidelineTree, GuidelineSection

        text = "Reduce metformin dose when eGFR < 30"
        tree = GuidelineTree(
            sections=[GuidelineSection(
                section_id="1", heading="Dosing",
                start_offset=0, end_offset=len(text),
                page_number=1, block_type="recommendation", children=[],
            )],
            tables=[], total_pages=1,
        )

        b_span = RawSpan(channel="B", text="metformin", start=7, end=16, confidence=0.90)
        c_span = RawSpan(channel="C", text="eGFR < 30", start=27, end=36, confidence=0.85)

        b_output = ChannelOutput(channel="B", spans=[b_span])
        c_output = ChannelOutput(channel="C", spans=[c_span])

        merger = SignalMerger()
        classifier = RuleBasedTieringClassifier()
        job_id = uuid4()

        merged = merger.merge(job_id, [b_output, c_output], tree, classifier=classifier)
        assert len(merged) >= 1
        for span in merged:
            assert span.tier is not None
            assert span.tier in ("TIER_1", "TIER_2", "NOISE")
            assert span.tier_reason is not None

    def test_no_classifier_leaves_tier_none(self):
        """Without classifier, MergedSpan.tier should remain None."""
        from extraction.v4.signal_merger import SignalMerger
        from extraction.v4.models import ChannelOutput, RawSpan, GuidelineTree, GuidelineSection

        text = "metformin 500 mg daily"
        tree = GuidelineTree(
            sections=[GuidelineSection(
                section_id="1", heading="Rx",
                start_offset=0, end_offset=len(text),
                page_number=1, block_type="recommendation", children=[],
            )],
            tables=[], total_pages=1,
        )

        b_span = RawSpan(channel="B", text="metformin", start=0, end=9, confidence=0.85)
        b_output = ChannelOutput(channel="B", spans=[b_span])

        merger = SignalMerger()
        job_id = uuid4()

        merged = merger.merge(job_id, [b_output], tree)
        assert len(merged) == 1
        assert merged[0].tier is None
        assert merged[0].tier_reason is None
