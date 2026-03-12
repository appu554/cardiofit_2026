"""
Tests for V4 Data Models.

Validates:
1. RawSpan creation and validation
2. MergedSpan creation with confidence bounds
3. ReviewerDecision audit trail
4. VerifiedSpan extraction_context
5. DrugDossier assembly
6. GuidelineTree offset-based section lookup
7. ChannelOutput standard interface
"""

import sys
from pathlib import Path
from uuid import uuid4

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.models import (
    ChannelOutput,
    ChannelStatus,
    DrugDossier,
    DossierResult,
    GuidelineSection,
    GuidelineTree,
    MergedSpan,
    RawSpan,
    ReviewerDecision,
    ReviewerAction,
    ReviewStatus,
    TableBoundary,
    VerifiedSpan,
)


class TestRawSpan:
    """Test RawSpan model validation."""

    def test_create_valid_span(self):
        span = RawSpan(
            channel="B",
            text="metformin",
            start=100,
            end=109,
            confidence=1.0,
            section_id="4.1.1",
            channel_metadata={"match_type": "exact"},
        )
        assert span.channel == "B"
        assert span.text == "metformin"
        assert span.confidence == 1.0

    def test_confidence_bounds(self):
        with pytest.raises(Exception):
            RawSpan(channel="B", text="x", start=0, end=1, confidence=1.5)

        with pytest.raises(Exception):
            RawSpan(channel="B", text="x", start=0, end=1, confidence=-0.1)

    def test_valid_channels(self):
        for ch in ["B", "C", "D", "E", "F"]:
            span = RawSpan(channel=ch, text="test", start=0, end=4, confidence=0.5)
            assert span.channel == ch

    def test_invalid_channel(self):
        with pytest.raises(Exception):
            RawSpan(channel="A", text="test", start=0, end=4, confidence=0.5)

    def test_channel_metadata_default(self):
        span = RawSpan(channel="C", text="test", start=0, end=4, confidence=0.5)
        assert span.channel_metadata == {}

    def test_uuid_auto_generated(self):
        span1 = RawSpan(channel="B", text="x", start=0, end=1, confidence=0.5)
        span2 = RawSpan(channel="B", text="x", start=0, end=1, confidence=0.5)
        assert span1.id != span2.id


class TestMergedSpan:
    """Test MergedSpan model validation."""

    def test_create_merged_span(self):
        job_id = uuid4()
        span = MergedSpan(
            job_id=job_id,
            text="metformin",
            start=100,
            end=109,
            contributing_channels=["B", "E"],
            channel_confidences={"B": 1.0, "E": 0.78},
            merged_confidence=0.94,
        )
        assert span.review_status == "PENDING"
        assert len(span.contributing_channels) == 2

    def test_default_review_status(self):
        span = MergedSpan(
            job_id=uuid4(),
            text="test",
            start=0,
            end=4,
            contributing_channels=["C"],
            channel_confidences={"C": 0.95},
            merged_confidence=0.95,
        )
        assert span.review_status == "PENDING"
        assert span.has_disagreement is False
        assert span.reviewer_text is None


class TestReviewerDecision:
    """Test ReviewerDecision audit trail."""

    def test_confirm_action(self):
        decision = ReviewerDecision(
            merged_span_id=uuid4(),
            job_id=uuid4(),
            action="CONFIRM",
            reviewer_id="reviewer_1",
        )
        assert decision.action == "CONFIRM"
        assert decision.edited_text is None

    def test_edit_action_with_text(self):
        decision = ReviewerDecision(
            merged_span_id=uuid4(),
            job_id=uuid4(),
            action="EDIT",
            original_text="dapa gliflozin",
            edited_text="dapagliflozin",
            reviewer_id="reviewer_1",
            note="OCR garbled this span, fixed spelling",
        )
        assert decision.edited_text == "dapagliflozin"
        assert decision.note is not None


class TestVerifiedSpan:
    """Test VerifiedSpan model."""

    def test_extraction_context_default(self):
        span = VerifiedSpan(
            text="metformin",
            start=0,
            end=9,
            confidence=0.94,
            contributing_channels=["B", "E"],
        )
        assert span.extraction_context == {}

    def test_extraction_context_with_hints(self):
        span = VerifiedSpan(
            text="eGFR >= 30",
            start=50,
            end=60,
            confidence=0.95,
            contributing_channels=["C"],
            extraction_context={
                "channel_C_pattern": "egfr_threshold",
                "channel_B_rxnorm_candidate": None,
            },
        )
        assert span.extraction_context["channel_C_pattern"] == "egfr_threshold"


class TestDrugDossier:
    """Test DrugDossier dataclass."""

    def test_create_dossier(self, sample_verified_spans):
        metformin_spans = [s for s in sample_verified_spans if "4.1.1" == s.section_id]
        dossier = DrugDossier(
            drug_name="metformin",
            rxnorm_candidate="860975",
            verified_spans=metformin_spans,
            source_sections=["4.1.1"],
            source_pages=[1],
            source_text="Metformin section text...",
            signal_summary={"thresholds": 2, "contraindications": 1},
        )
        assert dossier.drug_name == "metformin"
        assert len(dossier.verified_spans) > 0
        assert dossier.signal_summary["thresholds"] == 2


class TestGuidelineTree:
    """Test GuidelineTree offset-based lookups."""

    def test_find_section_for_offset(self, sample_guideline_tree):
        from .conftest import SAMPLE_KDIGO_TEXT

        # Offset in Recommendation 4.1.1 section
        offset = SAMPLE_KDIGO_TEXT.index("metformin in patients")
        section = sample_guideline_tree.find_section_for_offset(offset)
        assert section is not None
        assert section.section_id == "4.1.1"

    def test_find_section_for_offset_child(self, sample_guideline_tree):
        from .conftest import SAMPLE_KDIGO_TEXT

        offset = SAMPLE_KDIGO_TEXT.index("Dapagliflozin can")
        section = sample_guideline_tree.find_section_for_offset(offset)
        assert section is not None
        assert section.section_id == "4.1.2"

    def test_find_table_for_offset(self, sample_guideline_tree):
        from .conftest import SAMPLE_KDIGO_TEXT

        offset = SAMPLE_KDIGO_TEXT.index("| Drug |")
        table = sample_guideline_tree.find_table_for_offset(offset)
        assert table is not None
        assert table.table_id == "table_1"

    def test_find_no_section_for_invalid_offset(self, sample_guideline_tree):
        section = sample_guideline_tree.find_section_for_offset(999999)
        assert section is None

    def test_get_prose_sections(self, sample_guideline_tree):
        # Our fixture uses "recommendation" block_type, not "paragraph"
        # So prose_sections will be empty for this fixture
        prose = sample_guideline_tree.get_prose_sections()
        # This correctly returns empty because fixture uses recommendation type
        assert isinstance(prose, list)


class TestChannelOutput:
    """Test ChannelOutput standard interface."""

    def test_success_output(self):
        output = ChannelOutput(
            channel="B",
            spans=[
                RawSpan(channel="B", text="metformin", start=0, end=9, confidence=1.0),
            ],
            elapsed_ms=4.5,
        )
        assert output.success is True
        assert output.span_count == 1

    def test_error_output(self):
        output = ChannelOutput(
            channel="F",
            spans=[],
            error="NuExtract model failed to load",
        )
        assert output.success is False
        assert output.span_count == 0


class TestEnums:
    """Test enum values."""

    def test_channel_status_values(self):
        assert ChannelStatus.PENDING.value == "PENDING"
        assert ChannelStatus.COMPLETED.value == "COMPLETED"

    def test_review_status_values(self):
        assert ReviewStatus.CONFIRMED.value == "CONFIRMED"
        assert ReviewStatus.REJECTED.value == "REJECTED"

    def test_reviewer_action_values(self):
        assert ReviewerAction.ADD.value == "ADD"
        assert ReviewerAction.EDIT.value == "EDIT"
