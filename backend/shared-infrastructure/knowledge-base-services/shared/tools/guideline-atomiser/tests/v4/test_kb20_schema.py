"""
Tests for KB-20 Contextual Modifiers & ADR Profiles Schema.

Validates:
1. ContextualModifierFact completeness grading (FULL/PARTIAL/STUB)
2. AdverseReactionProfile completeness grading
3. KB20ExtractionResult auto-calculated totals and completeness summary
4. Pydantic alias serialization (snake_case -> camelCase)
5. model_post_init behavior for auto-grading
6. Required vs optional field validation
"""

import sys
from pathlib import Path

import pytest

SHARED_DIR = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(SHARED_DIR))

from extraction.schemas.kb20_contextual import (
    AdverseReactionProfile,
    ClinicalGovernance,
    ContextualModifierFact,
    KB20ExtractionResult,
)


# =============================================================================
# Factory Helpers
# =============================================================================

def _governance(**overrides) -> ClinicalGovernance:
    """Create a ClinicalGovernance with sensible defaults."""
    defaults = {
        "source_authority": "KDIGO",
        "source_document": "KDIGO 2022 Diabetes in CKD",
        "source_section": "4.1.1",
        "evidence_level": "1A",
        "effective_date": "2022-11-01",
        "guideline_doi": "10.1016/j.kint.2022.06.008",
    }
    defaults.update(overrides)
    return ClinicalGovernance(**defaults)


def _lab_modifier(**overrides) -> ContextualModifierFact:
    """Create a LAB_VALUE modifier with defaults."""
    defaults = {
        "modifier_type": "LAB_VALUE",
        "modifier_value": "eGFR < 30 mL/min/1.73m2",
        "effect": "contraindicated",
        "lab_parameter": "eGFR",
        "lab_operator": "<",
        "lab_threshold": 30.0,
        "lab_unit": "mL/min/1.73m2",
    }
    defaults.update(overrides)
    return ContextualModifierFact(**defaults)


def _population_modifier(**overrides) -> ContextualModifierFact:
    """Create a POPULATION modifier with defaults."""
    defaults = {
        "modifier_type": "POPULATION",
        "modifier_value": "elderly >75 years",
        "effect": "increased risk of hypoglycemia",
        "effect_magnitude": "MAJOR",
    }
    defaults.update(overrides)
    return ContextualModifierFact(**defaults)


def _full_adr(**overrides) -> AdverseReactionProfile:
    """Create a FULL-grade ADR profile with all fields populated."""
    defaults = {
        "rxnorm_code": "860975",
        "drug_name": "metformin",
        "drug_class": "Biguanide",
        "reaction": "lactic acidosis",
        "mechanism": "impaired hepatic lactate clearance in renal impairment",
        "symptom": "breathlessness",
        "onset_window": "2-4 weeks",
        "onset_category": "SUBACUTE",
        "frequency": "RARE",
        "severity": "CRITICAL",
        "risk_factors": ["renal impairment", "hepatic impairment", "dehydration"],
        "contextual_modifiers": [_lab_modifier()],
        "source_snippet": "Metformin is contraindicated when eGFR falls below 30",
        "governance": _governance(),
    }
    defaults.update(overrides)
    return AdverseReactionProfile(**defaults)


# =============================================================================
# ContextualModifierFact Tests
# =============================================================================

class TestContextualModifierFact:
    """Tests for ContextualModifierFact completeness grading."""

    def test_lab_value_full_grade(self):
        """LAB_VALUE with all structured fields should be FULL."""
        modifier = _lab_modifier()
        assert modifier.completeness_grade == "FULL"

    def test_lab_value_partial_grade(self):
        """LAB_VALUE with core but missing structured fields is PARTIAL."""
        modifier = ContextualModifierFact(
            modifier_type="LAB_VALUE",
            modifier_value="eGFR < 30",
            effect="contraindicated",
            # No lab_parameter, lab_operator, lab_threshold
        )
        assert modifier.completeness_grade == "PARTIAL"

    def test_lab_value_stub_grade(self):
        """LAB_VALUE with missing core fields is STUB."""
        modifier = ContextualModifierFact(
            modifier_type="LAB_VALUE",
            modifier_value="",  # empty
            effect="",  # empty
        )
        assert modifier.completeness_grade == "STUB"

    def test_population_full_grade(self):
        """POPULATION with effect_magnitude should be FULL."""
        modifier = _population_modifier()
        assert modifier.completeness_grade == "FULL"

    def test_population_partial_no_magnitude(self):
        """POPULATION without effect_magnitude should be PARTIAL."""
        modifier = ContextualModifierFact(
            modifier_type="POPULATION",
            modifier_value="elderly >75",
            effect="increased risk",
            # No effect_magnitude
        )
        assert modifier.completeness_grade == "PARTIAL"

    def test_concomitant_drug_full_grade(self):
        """CONCOMITANT_DRUG with drug name should be FULL."""
        modifier = ContextualModifierFact(
            modifier_type="CONCOMITANT_DRUG",
            modifier_value="concurrent NSAID use",
            effect="increases AKI risk",
            concomitant_drug="ibuprofen",
        )
        assert modifier.completeness_grade == "FULL"

    def test_concomitant_drug_partial(self):
        """CONCOMITANT_DRUG without drug name should be PARTIAL."""
        modifier = ContextualModifierFact(
            modifier_type="CONCOMITANT_DRUG",
            modifier_value="concurrent NSAID use",
            effect="increases AKI risk",
            # No concomitant_drug
        )
        assert modifier.completeness_grade == "PARTIAL"

    def test_temporal_full_with_magnitude(self):
        """TEMPORAL with effect_magnitude should be FULL."""
        modifier = ContextualModifierFact(
            modifier_type="TEMPORAL",
            modifier_value="first 3 months of initiation",
            effect="higher risk of DKA",
            effect_magnitude="MODERATE",
        )
        assert modifier.completeness_grade == "FULL"

    def test_alias_serialization(self):
        """Aliases should produce camelCase keys for Go consumption."""
        modifier = _lab_modifier()
        data = modifier.model_dump(by_alias=True)
        assert "modifierType" in data
        assert "modifierValue" in data
        assert "labParameter" in data
        assert "labOperator" in data
        assert "labThreshold" in data
        assert "labUnit" in data
        assert "completenessGrade" in data

    def test_snake_case_serialization(self):
        """Default serialization should use snake_case."""
        modifier = _lab_modifier()
        data = modifier.model_dump()
        assert "modifier_type" in data
        assert "modifier_value" in data
        assert "completeness_grade" in data


# =============================================================================
# AdverseReactionProfile Tests
# =============================================================================

class TestAdverseReactionProfile:
    """Tests for AdverseReactionProfile completeness grading."""

    def test_full_profile(self):
        """Profile with all fields should be FULL."""
        adr = _full_adr()
        assert adr.completeness_grade == "FULL"

    def test_partial_no_onset(self):
        """Profile with mechanism but no onset_window is PARTIAL."""
        adr = _full_adr(onset_window=None, contextual_modifiers=[])
        assert adr.completeness_grade == "PARTIAL"

    def test_partial_no_mechanism(self):
        """Profile with onset but no mechanism is PARTIAL."""
        adr = _full_adr(mechanism=None, contextual_modifiers=[])
        assert adr.completeness_grade == "PARTIAL"

    def test_stub_no_onset_no_mechanism(self):
        """Profile with neither onset nor mechanism is STUB."""
        adr = AdverseReactionProfile(
            rxnorm_code="860975",
            drug_name="metformin",
            reaction="nausea",
            governance=_governance(),
        )
        assert adr.completeness_grade == "STUB"

    def test_stub_missing_reaction(self):
        """Profile without reaction should be STUB (even with mechanism)."""
        adr = AdverseReactionProfile(
            rxnorm_code="860975",
            drug_name="metformin",
            reaction="",  # empty
            mechanism="hepatic clearance",
            governance=_governance(),
        )
        assert adr.completeness_grade == "STUB"

    def test_alias_serialization(self):
        """AdverseReactionProfile should serialize with camelCase aliases."""
        adr = _full_adr()
        data = adr.model_dump(by_alias=True)
        assert "rxnormCode" in data
        assert "drugName" in data
        assert "drugClass" in data
        assert "onsetWindow" in data
        assert "onsetCategory" in data
        assert "riskFactors" in data
        assert "contextualModifiers" in data
        assert "completenessGrade" in data
        assert "sourceSnippet" in data
        assert "sourceAuthority" in data["governance"]

    def test_onset_categories(self):
        """All onset category values should be accepted."""
        for cat in ("IMMEDIATE", "ACUTE", "SUBACUTE", "CHRONIC", "DELAYED"):
            adr = _full_adr(onset_category=cat)
            assert adr.onset_category == cat

    def test_frequency_values(self):
        """All frequency values should be accepted."""
        for freq in ("VERY_COMMON", "COMMON", "UNCOMMON", "RARE", "VERY_RARE", "UNKNOWN"):
            adr = _full_adr(frequency=freq)
            assert adr.frequency == freq

    def test_severity_values(self):
        """All severity values should be accepted."""
        for sev in ("CRITICAL", "HIGH", "MODERATE", "LOW"):
            adr = _full_adr(severity=sev)
            assert adr.severity == sev


# =============================================================================
# KB20ExtractionResult Tests
# =============================================================================

class TestKB20ExtractionResult:
    """Tests for KB20ExtractionResult auto-calculated fields."""

    def test_totals_auto_calculated(self):
        """total_adr_profiles and total_contextual_modifiers should auto-calc."""
        adr1 = _full_adr()  # has 1 contextual_modifier
        adr2 = _full_adr(
            drug_name="dapagliflozin",
            rxnorm_code="1488564",
            reaction="DKA",
            contextual_modifiers=[
                _lab_modifier(),
                _population_modifier(),
            ],
        )
        standalone = _population_modifier()

        result = KB20ExtractionResult(
            adr_profiles=[adr1, adr2],
            standalone_modifiers=[standalone],
            extraction_date="2026-03-02",
            source_guideline="KDIGO 2022 Diabetes in CKD",
        )

        assert result.total_adr_profiles == 2
        # 1 (adr1) + 2 (adr2) + 1 (standalone) = 4
        assert result.total_contextual_modifiers == 4

    def test_completeness_summary(self):
        """Completeness summary should count by grade."""
        full_adr = _full_adr()  # FULL
        partial_adr = _full_adr(
            drug_name="enalapril",
            rxnorm_code="3827",
            mechanism=None,
            contextual_modifiers=[],
        )  # PARTIAL
        stub_adr = AdverseReactionProfile(
            rxnorm_code="5856",
            drug_name="insulin",
            reaction="",
            governance=_governance(),
        )  # STUB

        result = KB20ExtractionResult(
            adr_profiles=[full_adr, partial_adr, stub_adr],
            extraction_date="2026-03-02",
            source_guideline="KDIGO 2022",
        )

        assert result.completeness_summary == {"FULL": 1, "PARTIAL": 1, "STUB": 1}

    def test_empty_result(self):
        """Empty result should have zeroes everywhere."""
        result = KB20ExtractionResult(
            adr_profiles=[],
            extraction_date="2026-03-02",
            source_guideline="Test",
        )
        assert result.total_adr_profiles == 0
        assert result.total_contextual_modifiers == 0
        assert result.completeness_summary == {"FULL": 0, "PARTIAL": 0, "STUB": 0}

    def test_alias_serialization(self):
        """Top-level result should serialize with camelCase aliases."""
        result = KB20ExtractionResult(
            adr_profiles=[_full_adr()],
            extraction_date="2026-03-02",
            source_guideline="KDIGO 2022",
        )
        data = result.model_dump(by_alias=True)
        assert "adrProfiles" in data
        assert "standaloneModifiers" in data
        assert "extractionDate" in data
        assert "extractorVersion" in data
        assert "sourceGuideline" in data
        assert "totalAdrProfiles" in data
        assert "totalContextualModifiers" in data
        assert "completenessSummary" in data

    def test_default_extractor_version(self):
        """Default extractor version should be set."""
        result = KB20ExtractionResult(
            adr_profiles=[],
            extraction_date="2026-03-02",
            source_guideline="Test",
        )
        assert result.extractor_version == "v4.0.0-facts"

    def test_round_trip_json(self):
        """Serialize and deserialize should produce equivalent result."""
        original = KB20ExtractionResult(
            adr_profiles=[_full_adr()],
            standalone_modifiers=[_population_modifier()],
            extraction_date="2026-03-02",
            source_guideline="KDIGO 2022",
        )

        json_str = original.model_dump_json(by_alias=True)
        restored = KB20ExtractionResult.model_validate_json(json_str)

        assert restored.total_adr_profiles == original.total_adr_profiles
        assert restored.total_contextual_modifiers == original.total_contextual_modifiers
        assert restored.completeness_summary == original.completeness_summary
        assert restored.adr_profiles[0].drug_name == "metformin"
        assert restored.adr_profiles[0].completeness_grade == "FULL"


# =============================================================================
# ClinicalGovernance Tests
# =============================================================================

class TestClinicalGovernance:
    """Tests for ClinicalGovernance provenance model."""

    def test_required_fields(self):
        """Minimum required fields should construct successfully."""
        gov = ClinicalGovernance(
            source_authority="KDIGO",
            source_document="KDIGO 2022",
            effective_date="2022-11-01",
        )
        assert gov.source_authority == "KDIGO"
        assert gov.evidence_level == ""  # default
        assert gov.guideline_doi is None

    def test_alias_serialization(self):
        """Governance should serialize with camelCase aliases."""
        gov = _governance()
        data = gov.model_dump(by_alias=True)
        assert "sourceAuthority" in data
        assert "sourceDocument" in data
        assert "sourceSection" in data
        assert "evidenceLevel" in data
        assert "effectiveDate" in data
        assert "guidelineDoi" in data
