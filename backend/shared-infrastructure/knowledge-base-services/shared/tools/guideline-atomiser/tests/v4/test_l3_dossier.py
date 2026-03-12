"""
Tests for V4 L3 Dossier-Based Extraction.

Tests the extract_facts_from_dossier() method and _build_dossier_prompt() on
KBFactExtractor — the V4 Pipeline 2 entry point that accepts DrugDossier
instead of raw markdown_text + gliner_entities.

Coverage:
    - Prompt construction from DrugDossier (all 3 KB targets)
    - Extraction context hints appear in prompt
    - RxNorm candidate propagation
    - Signal summary and source metadata in prompt
    - API call structure (tool_choice, tools, messages)
    - Response parsing (tool_use block -> KB schema)
    - Error handling (invalid KB, no client, JSON string response)
    - Integration with validate_extraction()
"""

import json
import sys
from dataclasses import dataclass
from datetime import date
from pathlib import Path
from typing import Optional
from unittest.mock import MagicMock, patch

import pytest

# Path setup — same as conftest.py
SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.v4.models import DrugDossier, VerifiedSpan

# Import the class under test
ATOMISER_DIR = SHARED_DIR / "tools" / "guideline-atomiser"
sys.path.insert(0, str(ATOMISER_DIR))

from fact_extractor import KBFactExtractor
from extraction.schemas.kb1_dosing import KB1ExtractionResult
from extraction.schemas.kb4_safety import KB4ExtractionResult
from extraction.schemas.kb16_labs import KB16ExtractionResult


# =============================================================================
# Fixtures
# =============================================================================

@pytest.fixture
def guideline_context():
    """Standard KDIGO guideline context for all tests."""
    return {
        "authority": "KDIGO",
        "document": "KDIGO 2022 Diabetes-CKD",
        "effective_date": "2022-11-01",
        "doi": "10.1016/j.kint.2022.06.008",
        "verified_rxnorm_codes": {
            "metformin": {"code": "860975", "display": "metformin hydrochloride"},
            "dapagliflozin": {"code": "1488564", "display": "dapagliflozin"},
        },
    }


@pytest.fixture
def metformin_dossier() -> DrugDossier:
    """A realistic metformin dossier with multiple verified spans."""
    return DrugDossier(
        drug_name="metformin",
        rxnorm_candidate="860975",
        verified_spans=[
            VerifiedSpan(
                text="metformin",
                start=100,
                end=109,
                confidence=0.94,
                contributing_channels=["B", "E"],
                section_id="4.1.1",
                page_number=12,
                extraction_context={
                    "channel_B_rxnorm_candidate": "860975",
                    "channel_B_confidence": 1.0,
                    "channel_E_confidence": 0.78,
                },
            ),
            VerifiedSpan(
                text="eGFR >= 30 mL/min/1.73m²",
                start=150,
                end=175,
                confidence=0.95,
                contributing_channels=["C"],
                section_id="4.1.1",
                page_number=12,
                extraction_context={
                    "channel_C_pattern": "egfr_threshold",
                    "channel_C_confidence": 0.95,
                },
            ),
            VerifiedSpan(
                text="reduce maximum daily dose of metformin to 1000 mg",
                start=200,
                end=250,
                confidence=0.92,
                contributing_channels=["C", "F"],
                section_id="4.1.1",
                page_number=12,
                extraction_context={
                    "channel_C_pattern": "dose_threshold",
                    "channel_C_confidence": 0.92,
                    "channel_F_confidence": 0.88,
                },
            ),
            VerifiedSpan(
                text="contraindicated when eGFR falls below 30",
                start=260,
                end=301,
                confidence=0.97,
                contributing_channels=["C", "E"],
                section_id="4.1.1",
                page_number=12,
                extraction_context={
                    "channel_C_pattern": "contraindication",
                },
            ),
        ],
        source_sections=["4.1.1"],
        source_pages=[12],
        source_text=(
            "We recommend using metformin in patients with T2D and CKD "
            "with eGFR >= 30 mL/min/1.73m². When eGFR is 30-44, reduce "
            "maximum daily dose of metformin to 1000 mg. Metformin is "
            "contraindicated when eGFR falls below 30 mL/min/1.73m²."
        ),
        signal_summary={
            "egfr_threshold": 1,
            "dose_threshold": 1,
            "contraindication": 1,
            "drug_anchor": 1,
        },
    )


@pytest.fixture
def empty_spans_dossier() -> DrugDossier:
    """A dossier with no verified spans (edge case)."""
    return DrugDossier(
        drug_name="orphan_drug",
        rxnorm_candidate=None,
        verified_spans=[],
        source_sections=[],
        source_pages=[],
        source_text="",
        signal_summary={},
    )


@pytest.fixture
def table_row_dossier() -> DrugDossier:
    """A dossier with table-derived spans (row_drug context)."""
    return DrugDossier(
        drug_name="Dapagliflozin",
        rxnorm_candidate="1488564",
        verified_spans=[
            VerifiedSpan(
                text="Dapagliflozin",
                start=400,
                end=413,
                confidence=1.0,
                contributing_channels=["B"],
                section_id="4.1.2",
                extraction_context={"channel_B_rxnorm_candidate": "1488564"},
            ),
            VerifiedSpan(
                text=">= 25 mL/min/1.73m²",
                start=420,
                end=440,
                confidence=0.90,
                contributing_channels=["D"],
                table_id="table_1",
                extraction_context={
                    "row_drug": "Dapagliflozin",
                    "column_header": "eGFR Threshold",
                },
            ),
            VerifiedSpan(
                text="10 mg daily",
                start=450,
                end=461,
                confidence=0.88,
                contributing_channels=["D"],
                table_id="table_1",
                extraction_context={
                    "row_drug": "Dapagliflozin",
                    "column_header": "Max Dose",
                },
            ),
        ],
        source_sections=["4.1.2"],
        source_pages=[13],
        source_text="Dapagliflozin can be initiated when eGFR >= 25.",
        signal_summary={"drug_anchor": 1, "table_cell": 2},
    )


def _mock_tool_use_block(input_data: dict):
    """Create a mock tool_use content block."""
    block = MagicMock()
    block.type = "tool_use"
    block.input = input_data
    return block


def _mock_text_block(text: str):
    """Create a mock text content block."""
    block = MagicMock()
    block.type = "text"
    block.text = text
    return block


def _make_kb1_response() -> dict:
    """A valid KB1ExtractionResult as a dict (for tool_use block input)."""
    return {
        "drugs": [
            {
                "rxnormCode": "860975",
                "drugName": "metformin",
                "drugClass": "biguanides",
                "renalAdjustments": [
                    {
                        "egfrMin": 30,
                        "egfrMax": 44,
                        "adjustmentFactor": 0.5,
                        "maxDose": 1000,
                        "maxDoseUnit": "mg",
                        "recommendation": "Reduce max daily dose to 1000 mg",
                        "contraindicated": False,
                        "actionType": "REDUCE_DOSE",
                    },
                    {
                        "egfrMin": 0,
                        "egfrMax": 30,
                        "recommendation": "Contraindicated below eGFR 30",
                        "contraindicated": True,
                        "actionType": "CONTRAINDICATED",
                    },
                ],
                "sourcePage": 12,
                "sourceSnippet": "Reduce maximum daily dose of metformin to 1000 mg when eGFR 30-44",
                "guidelineVersion": "KDIGO 2022",
            },
        ],
        "extractionDate": date.today().isoformat(),
        "sourceGuideline": "KDIGO 2022 Diabetes-CKD",
    }


def _make_kb4_response() -> dict:
    """A valid KB4ExtractionResult as a dict."""
    return {
        "contraindications": [
            {
                "rxnormCode": "860975",
                "drugName": "metformin",
                "conditionCodes": ["N18.4"],
                "conditionDescriptions": ["Chronic kidney disease, stage 4"],
                "type": "absolute",
                "severity": "CRITICAL",
                "clinicalRationale": "Risk of lactic acidosis with severely impaired renal clearance",
                "sourceSnippet": "Metformin is contraindicated when eGFR falls below 30",
                "governance": {
                    "sourceAuthority": "KDIGO",
                    "sourceDocument": "KDIGO 2022 Diabetes-CKD",
                    "sourceSection": "4.1.1",
                    "effectiveDate": "2022-11-01",
                },
            },
        ],
        "extractionDate": date.today().isoformat(),
        "sourceGuideline": "KDIGO 2022 Diabetes-CKD",
    }


def _make_kb16_response() -> dict:
    """A valid KB16ExtractionResult as a dict."""
    return {
        "labRequirements": [
            {
                "rxnormCode": "860975",
                "drugName": "metformin",
                "labs": [
                    {
                        "labName": "eGFR",
                        "loincCode": "62238-1",
                        "frequency": "every 3-6 months",
                        "baselineTiming": "before initiation",
                    },
                ],
                "baselineRequired": True,
                "sourceSnippet": "Monitor eGFR every 3-6 months",
                "governance": {
                    "sourceAuthority": "KDIGO",
                    "sourceDocument": "KDIGO 2022 Diabetes-CKD",
                    "effectiveDate": "2022-11-01",
                },
            },
        ],
        "extractionDate": date.today().isoformat(),
        "sourceGuideline": "KDIGO 2022 Diabetes-CKD",
    }


def _make_mock_client(response_data: dict) -> MagicMock:
    """Create a mock Anthropic client that returns a tool_use response."""
    client = MagicMock()
    mock_response = MagicMock()
    mock_response.content = [_mock_tool_use_block(response_data)]
    client.messages.create.return_value = mock_response
    return client


# =============================================================================
# Tests: _build_dossier_prompt() — Pure Logic (No Mocking Needed)
# =============================================================================

class TestBuildDossierPrompt:
    """Tests for the prompt construction from DrugDossier."""

    def test_prompt_contains_drug_name(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "## Drug: metformin" in prompt

    def test_prompt_contains_rxnorm_candidate(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "RxNorm candidate (from Channel B, verify): 860975" in prompt

    def test_prompt_omits_rxnorm_when_none(self, empty_spans_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            empty_spans_dossier, "dosing", guideline_context
        )
        assert "RxNorm candidate" not in prompt

    def test_prompt_uses_verified_spans_not_gliner(self, metformin_dossier, guideline_context):
        """V4 prompts should use 'Reviewer-Verified Text Spans', not GLiNER entities."""
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "## Reviewer-Verified Text Spans:" in prompt
        assert "Pre-tagged Clinical Entities" not in prompt
        assert "GLiNER" not in prompt

    def test_prompt_contains_all_verified_span_texts(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        for span in metformin_dossier.verified_spans:
            assert span.text in prompt

    def test_prompt_contains_span_confidence(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "confidence: 0.94" in prompt
        assert "confidence: 0.95" in prompt

    def test_prompt_contains_contributing_channels(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "channels: [B, E]" in prompt
        assert "channels: [C]" in prompt

    def test_prompt_contains_page_info(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "[page 12]" in prompt

    def test_prompt_contains_section_info(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "[section 4.1.1]" in prompt

    def test_prompt_contains_extraction_context_hints(self, metformin_dossier, guideline_context):
        """Extraction context hints (channel metadata) should appear in the prompt."""
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        # Channel B rxnorm candidate hint
        assert "channel_B_rxnorm_candidate=860975" in prompt
        # Channel C pattern hint
        assert "channel_C_pattern=egfr_threshold" in prompt
        assert "channel_C_pattern=dose_threshold" in prompt
        assert "channel_C_pattern=contraindication" in prompt

    def test_prompt_skips_confidence_in_context_hints(self, metformin_dossier, guideline_context):
        """Confidence values are already shown separately — should be skipped in hints."""
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        # _build_dossier_prompt skips keys matching channel_*_confidence
        # Check that the hint section doesn't re-state channel confidences
        lines = prompt.split("\n")
        for line in lines:
            if "channel_B_confidence=1.0" in line:
                # This should NOT appear in the hint parenthetical
                assert False, "channel_B_confidence should be skipped in hints"

    def test_prompt_contains_table_context(self, table_row_dossier, guideline_context):
        """Table-derived spans should include row_drug and column_header hints."""
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            table_row_dossier, "dosing", guideline_context
        )
        assert "row_drug=Dapagliflozin" in prompt
        assert "column_header=eGFR Threshold" in prompt
        assert "column_header=Max Dose" in prompt

    def test_prompt_contains_signal_summary(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "Signal summary:" in prompt
        assert "egfr_threshold" in prompt
        assert "contraindication" in prompt

    def test_prompt_contains_source_sections(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "Source sections: 4.1.1" in prompt

    def test_prompt_contains_source_pages(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "Source pages: 12" in prompt

    def test_prompt_contains_source_text(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "## Source Text (enclosing sections):" in prompt
        assert "We recommend using metformin" in prompt

    def test_prompt_contains_guideline_context(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "Authority: KDIGO" in prompt
        assert "Document: KDIGO 2022 Diabetes-CKD" in prompt
        assert "Effective Date: 2022-11-01" in prompt
        assert "DOI: 10.1016/j.kint.2022.06.008" in prompt

    def test_prompt_contains_verified_rxnorm_codes(self, metformin_dossier, guideline_context):
        """Pre-verified RxNorm codes from KB-7 should appear in prompt."""
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "## Pre-verified RxNorm Codes (from KB-7):" in prompt
        assert "metformin: 860975" in prompt
        assert "dapagliflozin: 1488564" in prompt

    def test_prompt_handles_no_verified_codes(self, metformin_dossier):
        """When no verified codes are available, prompt should say so."""
        ctx = {"authority": "KDIGO", "document": "Test"}
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(metformin_dossier, "dosing", ctx)
        assert "No pre-verified codes available" in prompt

    def test_prompt_empty_spans_shows_placeholder(self, empty_spans_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            empty_spans_dossier, "dosing", guideline_context
        )
        assert "No spans available." in prompt

    def test_prompt_contains_extraction_date(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert date.today().isoformat() in prompt

    def test_prompt_contains_verified_text_instruction(self, metformin_dossier, guideline_context):
        """The prompt should tell L3 that span text is verified correct."""
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "The text is VERIFIED CORRECT" in prompt
        assert "Machine-generated extraction context" in prompt

    # --- KB-specific instruction variants ---

    def test_dosing_prompt_contains_kb1_instructions(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert "## Target: KB-1 Drug Dosing Facts" in prompt
        assert "eGFR thresholds" in prompt
        assert "Adjustment factor" in prompt

    def test_safety_prompt_contains_kb4_instructions(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "safety", guideline_context
        )
        assert "## Target: KB-4 Patient Safety Facts" in prompt
        assert "CONTRAINDICATION FACTS" in prompt
        assert "Severity: CRITICAL, HIGH, MODERATE, LOW" in prompt

    def test_monitoring_prompt_contains_kb16_instructions(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "monitoring", guideline_context
        )
        assert "## Target: KB-16 Lab Monitoring Facts" in prompt
        assert "LAB MONITORING FACTS" in prompt
        assert "Monitoring frequency" in prompt

    def test_prompt_numbered_spans(self, metformin_dossier, guideline_context):
        """Spans should be numbered sequentially starting from 1."""
        extractor = KBFactExtractor(client=MagicMock())
        prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        assert '1. "metformin"' in prompt
        assert '2. "eGFR >= 30' in prompt
        assert '3. "reduce maximum daily dose' in prompt
        assert '4. "contraindicated when eGFR' in prompt


# =============================================================================
# Tests: extract_facts_from_dossier() — API Interaction
# =============================================================================

class TestExtractFactsFromDossier:
    """Tests for the full extraction flow with mock Anthropic client."""

    def test_accepts_drug_dossier(self, metformin_dossier, guideline_context):
        """Method signature accepts DrugDossier, not markdown_text + entities."""
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            dossier=metformin_dossier,
            target_kb="dosing",
            guideline_context=guideline_context,
        )
        assert isinstance(result, KB1ExtractionResult)

    def test_returns_kb1_for_dosing(self, metformin_dossier, guideline_context):
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        assert isinstance(result, KB1ExtractionResult)
        assert len(result.drugs) == 1
        assert result.drugs[0].drug_name == "metformin"
        assert len(result.drugs[0].renal_adjustments) == 2

    def test_returns_kb4_for_safety(self, metformin_dossier, guideline_context):
        client = _make_mock_client(_make_kb4_response())
        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "safety", guideline_context
        )
        assert isinstance(result, KB4ExtractionResult)
        assert len(result.contraindications) == 1
        assert result.contraindications[0].severity == "CRITICAL"

    def test_returns_kb16_for_monitoring(self, metformin_dossier, guideline_context):
        client = _make_mock_client(_make_kb16_response())
        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "monitoring", guideline_context
        )
        assert isinstance(result, KB16ExtractionResult)
        assert len(result.lab_requirements) == 1
        assert result.lab_requirements[0].labs[0].lab_name == "eGFR"

    def test_raises_value_error_for_invalid_kb(self, metformin_dossier, guideline_context):
        client = _make_mock_client({})
        extractor = KBFactExtractor(client=client)
        with pytest.raises(ValueError, match="Invalid target_kb"):
            extractor.extract_facts_from_dossier(
                metformin_dossier, "interactions", guideline_context
            )

    def test_raises_runtime_error_without_client(self, metformin_dossier, guideline_context):
        extractor = KBFactExtractor(client=None)
        with pytest.raises(RuntimeError, match="Anthropic client not available"):
            extractor.extract_facts_from_dossier(
                metformin_dossier, "dosing", guideline_context
            )

    def test_api_call_uses_tool_choice_any(self, metformin_dossier, guideline_context):
        """The API call should force tool use with tool_choice=any."""
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client)
        extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        call_kwargs = client.messages.create.call_args.kwargs
        assert call_kwargs["tool_choice"] == {"type": "any"}

    def test_api_call_has_correct_tool_name(self, metformin_dossier, guideline_context):
        """Tool name should match the target KB."""
        for kb_name in ("dosing", "safety", "monitoring"):
            response_map = {
                "dosing": _make_kb1_response(),
                "safety": _make_kb4_response(),
                "monitoring": _make_kb16_response(),
            }
            client = _make_mock_client(response_map[kb_name])
            extractor = KBFactExtractor(client=client)
            extractor.extract_facts_from_dossier(
                metformin_dossier, kb_name, guideline_context
            )
            call_kwargs = client.messages.create.call_args.kwargs
            tool_name = call_kwargs["tools"][0]["name"]
            assert tool_name == f"extract_{kb_name}_facts"

    def test_api_call_has_schema_in_tool(self, metformin_dossier, guideline_context):
        """The tool definition should include the KB schema as input_schema."""
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client)
        extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        call_kwargs = client.messages.create.call_args.kwargs
        tool_schema = call_kwargs["tools"][0]["input_schema"]
        # KB1ExtractionResult schema should have "drugs" in properties
        assert "drugs" in tool_schema.get("properties", {}) or \
               "drugs" in json.dumps(tool_schema)

    def test_api_call_prompt_is_dossier_prompt(self, metformin_dossier, guideline_context):
        """The message content should be the dossier prompt, not the V3 prompt."""
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client)
        extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        call_kwargs = client.messages.create.call_args.kwargs
        message_content = call_kwargs["messages"][0]["content"]
        assert "Reviewer-Verified Text Spans" in message_content
        assert "metformin" in message_content

    def test_api_call_uses_configured_model(self, metformin_dossier, guideline_context):
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client, model="claude-opus-4-20250514")
        extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        call_kwargs = client.messages.create.call_args.kwargs
        assert call_kwargs["model"] == "claude-opus-4-20250514"

    def test_handles_json_string_response(self, metformin_dossier, guideline_context):
        """If the API returns a JSON string instead of dict, it should be parsed."""
        client = MagicMock()
        mock_response = MagicMock()
        block = MagicMock()
        block.type = "tool_use"
        block.input = json.dumps(_make_kb1_response())  # string, not dict
        mock_response.content = [block]
        client.messages.create.return_value = mock_response

        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        assert isinstance(result, KB1ExtractionResult)

    def test_raises_on_no_tool_use_in_response(self, metformin_dossier, guideline_context):
        """If response has no tool_use block, should raise RuntimeError."""
        client = MagicMock()
        mock_response = MagicMock()
        mock_response.content = [_mock_text_block("I cannot extract facts.")]
        client.messages.create.return_value = mock_response

        extractor = KBFactExtractor(client=client)
        with pytest.raises(RuntimeError, match="No tool use response"):
            extractor.extract_facts_from_dossier(
                metformin_dossier, "dosing", guideline_context
            )

    def test_handles_text_then_tool_use_response(self, metformin_dossier, guideline_context):
        """Response may have a text block before the tool_use block."""
        client = MagicMock()
        mock_response = MagicMock()
        mock_response.content = [
            _mock_text_block("Analyzing the dossier..."),
            _mock_tool_use_block(_make_kb1_response()),
        ]
        client.messages.create.return_value = mock_response

        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        assert isinstance(result, KB1ExtractionResult)

    def test_max_tokens_is_8192(self, metformin_dossier, guideline_context):
        """Extraction should request 8192 max tokens for structured output."""
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client)
        extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        call_kwargs = client.messages.create.call_args.kwargs
        assert call_kwargs["max_tokens"] == 8192


# =============================================================================
# Tests: Validation Integration
# =============================================================================

class TestDossierExtractionValidation:
    """Tests for validate_extraction() on dossier-extracted results."""

    def test_validate_kb1_result(self, metformin_dossier, guideline_context):
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        validation = extractor.validate_extraction(result)
        assert validation["valid"] is True
        assert validation["total_issues"] == 0

    def test_validate_kb4_result(self, metformin_dossier, guideline_context):
        client = _make_mock_client(_make_kb4_response())
        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "safety", guideline_context
        )
        validation = extractor.validate_extraction(result)
        assert validation["valid"] is True

    def test_validate_kb16_result(self, metformin_dossier, guideline_context):
        client = _make_mock_client(_make_kb16_response())
        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "monitoring", guideline_context
        )
        validation = extractor.validate_extraction(result)
        assert validation["valid"] is True

    def test_validate_warns_on_missing_rxnorm(self, metformin_dossier, guideline_context):
        """Validation should warn when RxNorm code is missing."""
        kb1_data = _make_kb1_response()
        kb1_data["drugs"][0]["rxnormCode"] = "UNKNOWN"
        client = _make_mock_client(kb1_data)
        extractor = KBFactExtractor(client=client)
        result = extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )
        validation = extractor.validate_extraction(result)
        assert validation["total_warnings"] > 0
        assert any("Missing RxNorm" in w for w in validation["warnings"])


# =============================================================================
# Tests: V3 vs V4 API Contract Comparison
# =============================================================================

class TestV3V4Contract:
    """Verify that V4 dossier path and V3 entity path share the same output types."""

    def test_both_paths_return_same_kb1_type(self, metformin_dossier, guideline_context):
        """extract_facts() and extract_facts_from_dossier() both return KB1ExtractionResult."""
        client = _make_mock_client(_make_kb1_response())
        extractor = KBFactExtractor(client=client)

        v4_result = extractor.extract_facts_from_dossier(
            metformin_dossier, "dosing", guideline_context
        )

        # Reset mock for V3 call
        client.messages.create.return_value.content = [
            _mock_tool_use_block(_make_kb1_response())
        ]
        v3_result = extractor.extract_facts(
            markdown_text="metformin eGFR >= 30",
            gliner_entities=[{"text": "metformin", "label": "DRUG"}],
            target_kb="dosing",
            guideline_context=guideline_context,
        )

        assert type(v4_result) is type(v3_result)
        assert isinstance(v4_result, KB1ExtractionResult)

    def test_dossier_prompt_differs_from_entity_prompt(self, metformin_dossier, guideline_context):
        """V4 dossier prompt should be structurally different from V3 entity prompt."""
        extractor = KBFactExtractor(client=MagicMock())

        v4_prompt = extractor._build_dossier_prompt(
            metformin_dossier, "dosing", guideline_context
        )
        v3_prompt = extractor._build_prompt(
            "dosing",
            "metformin eGFR >= 30",
            [{"text": "metformin", "label": "DRUG"}],
            guideline_context,
        )

        # V4 uses verified spans section, V3 uses GLiNER entities section
        assert "Reviewer-Verified Text Spans" in v4_prompt
        assert "Reviewer-Verified Text Spans" not in v3_prompt
        assert "Pre-tagged Clinical Entities" in v3_prompt
        assert "Pre-tagged Clinical Entities" not in v4_prompt
