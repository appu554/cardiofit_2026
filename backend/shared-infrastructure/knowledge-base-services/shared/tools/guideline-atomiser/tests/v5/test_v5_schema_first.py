"""V5 Schema-first extraction unit tests — Subsystem #3.

Run from guideline-atomiser/:
    PYTHONPATH=. V5_SCHEMA_FIRST=1 pytest tests/v5/test_v5_schema_first.py -v
"""
from __future__ import annotations

import pytest

from extraction.v4.schema_first import (
    SCHEMA_NAMES,
    SCHEMA_REGISTRY,
    AlgorithmStep,
    ContraindicationStatement,
    DoseAdjustmentRow,
    DrugConditionMatrix,
    EGFRThresholdTable,
    EvidenceGradeBlock,
    FollowUpScheduleEntry,
    MonitoringFrequencyRow,
    RecommendationStatement,
    RiskScoreCalculator,
    SchemaFirstValidator,
    SchemaValidationResult,
    route_span_to_schema,
    validate_span,
)


# ─── Schema registry ──────────────────────────────────────────────────────────

def test_schema_registry_has_10_entries():
    assert len(SCHEMA_REGISTRY) == 10


def test_all_schema_names_listed():
    expected = {
        "RecommendationStatement", "DrugConditionMatrix", "EGFRThresholdTable",
        "MonitoringFrequencyRow", "EvidenceGradeBlock", "AlgorithmStep",
        "ContraindicationStatement", "DoseAdjustmentRow", "RiskScoreCalculator",
        "FollowUpScheduleEntry",
    }
    assert set(SCHEMA_NAMES) == expected


# ─── RecommendationStatement ──────────────────────────────────────────────────

def test_recommendation_valid_minimal():
    r = RecommendationStatement(text="SGLT2 inhibitors are recommended for HF patients.")
    assert r.text.startswith("SGLT2")


def test_recommendation_valid_with_class_and_level():
    r = RecommendationStatement(
        text="ACE inhibitors should be started.",
        strength="I",
        evidence_level="A",
    )
    assert r.strength.value == "I"
    assert r.evidence_level.value == "A"


def test_recommendation_invalid_empty_text():
    with pytest.raises(Exception):
        RecommendationStatement(text="")


def test_recommendation_invalid_class():
    with pytest.raises(Exception):
        RecommendationStatement(text="Some text", strength="IV")


# ─── DrugConditionMatrix ──────────────────────────────────────────────────────

def test_drug_condition_valid():
    d = DrugConditionMatrix(
        drug_name="Metformin",
        condition="Type 2 diabetes + CKD G3a",
        recommendation="Use with eGFR monitoring",
    )
    assert d.drug_name == "Metformin"


def test_drug_condition_contraindicated_flag():
    d = DrugConditionMatrix(
        drug_name="NSAID",
        condition="CKD G4",
        recommendation="Avoid",
        contraindicated=True,
    )
    assert d.contraindicated is True


def test_drug_condition_missing_drug_name():
    with pytest.raises(Exception):
        DrugConditionMatrix(drug_name="", condition="CKD", recommendation="Avoid")


# ─── EGFRThresholdTable ───────────────────────────────────────────────────────

def test_egfr_valid():
    e = EGFRThresholdTable(egfr_threshold=45.0, action="Reduce metformin dose")
    assert e.egfr_threshold == 45.0
    assert "mL/min" in e.egfr_unit


def test_egfr_below_zero_invalid():
    with pytest.raises(Exception):
        EGFRThresholdTable(egfr_threshold=-1.0, action="Stop drug")


def test_egfr_above_200_invalid():
    with pytest.raises(Exception):
        EGFRThresholdTable(egfr_threshold=201.0, action="Normal dosing")


# ─── MonitoringFrequencyRow ───────────────────────────────────────────────────

def test_monitoring_valid():
    m = MonitoringFrequencyRow(parameter="eGFR", frequency="every 3 months")
    assert m.parameter == "eGFR"


def test_monitoring_missing_frequency():
    with pytest.raises(Exception):
        MonitoringFrequencyRow(parameter="HbA1c", frequency="")


# ─── EvidenceGradeBlock ───────────────────────────────────────────────────────

def test_evidence_grade_valid():
    e = EvidenceGradeBlock(grade="A", recommendation_text="Strong evidence supports this intervention.")
    assert e.grade.value == "A"


def test_evidence_grade_invalid_letter():
    with pytest.raises(Exception):
        EvidenceGradeBlock(grade="Z", recommendation_text="Some text")


def test_evidence_grade_empty_text():
    with pytest.raises(Exception):
        EvidenceGradeBlock(grade="B", recommendation_text="Hi")  # min_length=5


# ─── AlgorithmStep ────────────────────────────────────────────────────────────

def test_algorithm_step_valid():
    a = AlgorithmStep(action="Initiate SGLT2 inhibitor", step_number=2)
    assert a.step_number == 2


def test_algorithm_step_zero_number_invalid():
    with pytest.raises(Exception):
        AlgorithmStep(action="Start treatment", step_number=0)


# ─── ContraindicationStatement ────────────────────────────────────────────────

def test_contraindication_valid():
    c = ContraindicationStatement(
        drug_or_treatment="Metformin",
        contraindication="eGFR < 30 mL/min/1.73m²",
        severity="absolute",
    )
    assert c.severity.value == "absolute"


def test_contraindication_invalid_severity():
    with pytest.raises(Exception):
        ContraindicationStatement(
            drug_or_treatment="Aspirin",
            contraindication="Active bleeding",
            severity="maybe",
        )


# ─── DoseAdjustmentRow ────────────────────────────────────────────────────────

def test_dose_adjustment_valid():
    d = DoseAdjustmentRow(
        drug="Rivaroxaban",
        condition="CKD G4 (eGFR 15-29)",
        dose_adjustment="Reduce to 15 mg daily",
    )
    assert "Rivaroxaban" in d.drug


def test_dose_adjustment_empty_drug():
    with pytest.raises(Exception):
        DoseAdjustmentRow(drug="", condition="CKD", dose_adjustment="Halve dose")


# ─── RiskScoreCalculator ──────────────────────────────────────────────────────

def test_risk_score_valid():
    r = RiskScoreCalculator(
        score_name="CHA₂DS₂-VASc",
        variables=["CHF", "Hypertension", "Age≥75", "Diabetes", "Stroke"],
        threshold="2",
        interpretation="Score ≥2 → anticoagulation recommended",
    )
    assert len(r.variables) == 5


def test_risk_score_empty_variables():
    with pytest.raises(Exception):
        RiskScoreCalculator(score_name="TIMI", variables=[])


# ─── FollowUpScheduleEntry ────────────────────────────────────────────────────

def test_follow_up_valid():
    f = FollowUpScheduleEntry(
        condition="Post-MI discharge",
        interval="2 weeks",
        assessment="Review medication tolerability and BP",
    )
    assert f.interval == "2 weeks"


def test_follow_up_missing_condition():
    with pytest.raises(Exception):
        FollowUpScheduleEntry(condition="", interval="monthly")


# ─── Router ───────────────────────────────────────────────────────────────────

def test_router_egfr_keyword():
    assert route_span_to_schema("eGFR < 30 mL/min — stop metformin") == "EGFRThresholdTable"


def test_router_contraindication_keyword():
    assert route_span_to_schema("Metformin is contraindicated in CKD G4") == "ContraindicationStatement"


def test_router_dose_adjustment():
    assert route_span_to_schema("Dose adjust rivaroxaban in renal impairment") == "DoseAdjustmentRow"


def test_router_algorithm_step():
    assert route_span_to_schema("Step 2: administer aspirin 300 mg", "Figure 1 — Algorithm") == "AlgorithmStep"


def test_router_monitoring_frequency():
    assert route_span_to_schema("Monitor eGFR every 3 months in CKD patients") == "MonitoringFrequencyRow"


def test_router_grade_block():
    assert route_span_to_schema("GRADE A evidence supports dual antiplatelet therapy.") == "EvidenceGradeBlock"


def test_router_fallback_is_recommendation():
    schema = route_span_to_schema("Patients should be advised to exercise regularly.")
    assert schema == "RecommendationStatement"


def test_router_drug_condition_heuristic():
    assert route_span_to_schema("SGLT2 inhibitor use in heart failure with reduced ejection fraction") == "DrugConditionMatrix"


# ─── validate_span() ──────────────────────────────────────────────────────────

def test_validate_span_returns_result_object():
    result = validate_span("SGLT2 inhibitors are recommended for HFrEF patients.")
    assert isinstance(result, SchemaValidationResult)


def test_validate_span_valid_recommendation():
    result = validate_span("All patients with HFrEF should receive ACE inhibitors. Class I, Level A.")
    assert result.is_valid
    assert result.schema_name == "RecommendationStatement"


def test_validate_span_schema_hint_overrides_router():
    result = validate_span(
        text="eGFR 45 — reduce metformin to 500 mg",
        schema_hint="MonitoringFrequencyRow",
        extra_fields={"parameter": "eGFR", "frequency": "every 3 months"},
    )
    assert result.schema_name == "MonitoringFrequencyRow"


def test_validate_span_egfr_valid():
    result = validate_span(
        text="eGFR < 30 mL/min — stop metformin",
        extra_fields={"egfr_threshold": 30.0, "action": "Stop metformin"},
    )
    assert result.schema_name == "EGFRThresholdTable"
    assert result.is_valid


def test_validate_span_egfr_invalid_threshold():
    result = validate_span(
        text="eGFR measurements",
        schema_hint="EGFRThresholdTable",
        extra_fields={"egfr_threshold": -5.0, "action": "Something"},
    )
    assert not result.is_valid


# ─── SchemaFirstValidator ─────────────────────────────────────────────────────

def _make_span(text: str, ce_flagged: bool = False) -> dict:
    return {"text": text, "ce_flagged": ce_flagged, "section_id": "1.1"}


def test_validator_returns_results_list():
    v = SchemaFirstValidator()
    spans = [_make_span("SGLT2 inhibitors recommended for HFrEF (Class I, Level A).")]
    results = v.validate_all(spans)
    assert isinstance(results, list)
    assert len(results) == 1


def test_validator_skips_ce_flagged_spans():
    v = SchemaFirstValidator()
    spans = [
        _make_span("Legitimate recommendation text here.", ce_flagged=False),
        _make_span("Low-confidence text.", ce_flagged=True),
    ]
    results = v.validate_all(spans)
    assert len(results) == 1


def test_validator_skips_empty_text():
    v = SchemaFirstValidator()
    spans = [_make_span(""), _make_span("ACE inhibitors are recommended for HFrEF.")]
    results = v.validate_all(spans)
    assert len(results) == 1


def test_validator_metrics_pass_rate_present():
    v = SchemaFirstValidator()
    spans = [_make_span("ACE inhibitors are recommended for all HFrEF patients.") for _ in range(10)]
    v.validate_all(spans)
    m = v.metrics()
    assert "v5_schema_first" in m
    assert "pass_rate_pct" in m["v5_schema_first"]
    assert m["v5_schema_first"]["total_validated"] == 10


def test_validator_metrics_empty_spans():
    v = SchemaFirstValidator()
    v.validate_all([])
    m = v.metrics()
    assert m["v5_schema_first"]["total_validated"] == 0
    assert m["v5_schema_first"]["pass_rate_pct"] == 0.0


def test_validator_metrics_includes_primary_and_verdict():
    v = SchemaFirstValidator()
    spans = [_make_span("SGLT2 inhibitor use in HFrEF reduces hospitalisation (Class I, Level A).")]
    v.validate_all(spans)
    m = v.metrics()
    assert "primary" in m
    assert "verdict_schema_first" in m


def test_validator_per_schema_breakdown():
    v = SchemaFirstValidator()
    spans = [
        _make_span("eGFR < 30 — stop drug"),
        _make_span("Monitor eGFR every 3 months"),
    ]
    v.validate_all(spans)
    m = v.metrics()
    per = m["v5_schema_first"]["per_schema"]
    assert isinstance(per, dict)
    assert all("total" in v_ and "passed" in v_ for v_ in per.values())
