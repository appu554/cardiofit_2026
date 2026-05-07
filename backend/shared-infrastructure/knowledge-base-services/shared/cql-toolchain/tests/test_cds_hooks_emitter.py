"""Wave 1 Task 4 — CDS Hooks v2.0 emitter round-trip tests.

Synthetic rule fire → PlanDefinition $apply → RequestOrchestration →
CDS Hooks v2.0 response card. Schema validation against the in-code
v2.0 shape (no live validator available).
"""

from __future__ import annotations

from pathlib import Path

import pytest

from cds_hooks_emitter import (
    RuleFire,
    apply_plan_definition,
    emit_cds_hooks_response,
    load_bundle,
    validate_cds_hooks_v2_response,
)

PLAN_DEFS = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "plan-definitions"
)


@pytest.fixture
def ppi_bundle():
    return load_bundle(PLAN_DEFS / "example-ppi-deprescribe.json")


@pytest.fixture
def ppi_fire():
    return RuleFire(
        rule_id="PPI_LONG_TERM_NO_INDICATION",
        summary="PPI active >56d without indication",
        indicator="warning",
        detail=(
            "Patient has been on a PPI for 60 days with no documented "
            "indication and no active GI bleed watch."
        ),
        recommendation_text="Trial PPI cessation with H2RA prn cover",
        links=[
            {
                "label": "MedicationRequest under review",
                "url": "https://example.org/MedicationRequest/abc",
                "type": "absolute",
            }
        ],
    )


# ---------------------------------------------------------------------------
# PlanDefinition $apply
# ---------------------------------------------------------------------------


def test_plan_definition_apply_yields_request_orchestration(
    ppi_bundle, ppi_fire,
):
    req_orch = apply_plan_definition(ppi_bundle, ppi_fire)
    assert req_orch["resourceType"] == "RequestOrchestration"
    assert req_orch["status"] == "draft"
    assert req_orch["intent"] == "proposal"
    assert req_orch["instantiatesCanonical"] == [
        "https://vaidshala.cardiofit/PlanDefinition/PPI_LONG_TERM_NO_INDICATION"
    ]
    assert req_orch["action"][0]["title"] == "Trial PPI cessation with H2RA prn cover"


# ---------------------------------------------------------------------------
# CDS Hooks v2.0 response shape
# ---------------------------------------------------------------------------


def test_emitted_response_passes_v2_validation(ppi_bundle, ppi_fire):
    req_orch = apply_plan_definition(ppi_bundle, ppi_fire)
    response = emit_cds_hooks_response(ppi_fire, req_orch, hook_type="order-select")
    errors = validate_cds_hooks_v2_response(response)
    assert errors == [], errors

    card = response["cards"][0]
    assert card["summary"] == "PPI active >56d without indication"
    assert card["indicator"] == "warning"
    assert card["source"]["label"] == "Vaidshala Clinical Reasoning"
    assert len(card["suggestions"]) == 1
    sug = card["suggestions"][0]
    assert sug["actions"][0]["resource"]["resourceType"] == "RequestOrchestration"
    # Must have at least one absolute link to a FHIR resource URI
    assert any(
        link["type"] == "absolute" and link["url"].startswith("http")
        for link in card["links"]
    )


def test_invalid_indicator_rejected(ppi_fire):
    bad = RuleFire(
        rule_id=ppi_fire.rule_id,
        summary=ppi_fire.summary,
        indicator="critical-error",
    )
    with pytest.raises(ValueError):
        emit_cds_hooks_response(bad)


def test_invalid_hook_type_rejected(ppi_fire):
    with pytest.raises(ValueError):
        emit_cds_hooks_response(ppi_fire, hook_type="patient-view")


def test_validate_catches_missing_required_keys():
    bad = {"cards": [{"summary": "x"}]}  # missing uuid/indicator/source
    errors = validate_cds_hooks_v2_response(bad)
    assert any("uuid" in e for e in errors)
    assert any("indicator" in e for e in errors)
    assert any("source" in e for e in errors)


def test_round_trip_with_no_request_orchestration(ppi_fire):
    response = emit_cds_hooks_response(ppi_fire)
    errors = validate_cds_hooks_v2_response(response)
    assert errors == []
    assert response["cards"][0]["selectionBehavior"] == "at-most-one"
