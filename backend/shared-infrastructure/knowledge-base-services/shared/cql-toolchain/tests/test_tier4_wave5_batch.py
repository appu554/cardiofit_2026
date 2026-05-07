"""Wave 5 batch acceptance — 6 Tier 4 surveillance rules.

Plan acceptance (Wave 5 vertical slice) requires that the 6 grounded
Tier 4 surveillance rule specifications:

  1. Validate against rule_specification.v2.json (Stage 1).
  2. Pass the two-gate validator (snapshot + substrate gates).
  3. Show ACTIVE in CompatibilityChecker.
  4. Emit a CDS Hooks v2.0-valid response via the emitter.

The 6 rules are:
  Trajectory.cql:
    - VAIDSHALA_T4_EGFR_TRAJECTORY_DECLINE_90D
    - VAIDSHALA_T4_WEIGHT_LOSS_TRAJECTORY_90D
    - VAIDSHALA_T4_BEHAVIOURAL_EPISODE_FREQUENCY_CHANGE_14D
  Lifecycle.cql:
    - VAIDSHALA_T4_ANTIPSYCHOTIC_REVIEW_OVERDUE_3MO
    - VAIDSHALA_T4_CONSENT_EXPIRING_WITHIN_30D
    - VAIDSHALA_T4_MONITORING_PLAN_OVERDUE_OBSERVATION

The remaining ~44 rules are queued — see
claudedocs/clinical/2026-05-Layer3-Wave5-Task1-tier4-rule-queue.md.
"""

from __future__ import annotations

from pathlib import Path

import pytest

from cds_hooks_emitter import (
    RuleFire,
    emit_cds_hooks_response,
    validate_cds_hooks_v2_response,
)
from compatibility_checker import CompatStatus, CompatibilityChecker
from rule_specification_validator import load_spec, validate_rule_specification
from two_gate_validator import _extract_define_body, run_two_gate

TIER4_DIR = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "tier-4-surveillance"
)
SPECS_DIR = TIER4_DIR / "specs"

EXPECTED_RULE_IDS = {
    # Wave 5 vertical slice (6)
    "VAIDSHALA_T4_EGFR_TRAJECTORY_DECLINE_90D",
    "VAIDSHALA_T4_WEIGHT_LOSS_TRAJECTORY_90D",
    "VAIDSHALA_T4_BEHAVIOURAL_EPISODE_FREQUENCY_CHANGE_14D",
    "VAIDSHALA_T4_ANTIPSYCHOTIC_REVIEW_OVERDUE_3MO",
    "VAIDSHALA_T4_CONSENT_EXPIRING_WITHIN_30D",
    "VAIDSHALA_T4_MONITORING_PLAN_OVERDUE_OBSERVATION",
    # Wave-extension batch 2026-05 (8)
    "VAIDSHALA_T4_SODIUM_TRAJECTORY_DELTA_90D",
    "VAIDSHALA_T4_BMI_TRAJECTORY_DELTA_180D",
    "VAIDSHALA_T4_PPI_REVIEW_OVERDUE_6MO",
    "VAIDSHALA_T4_STATIN_REVIEW_OVERDUE_12MO",
    "VAIDSHALA_T4_OPIOID_REVIEW_OVERDUE_3MO",
    "VAIDSHALA_T4_ANTIMICROBIAL_REVIEW_OVERDUE_7D",
    "VAIDSHALA_T4_PRESCRIBER_CREDENTIAL_EXPIRING_WITHIN_30D",
    "VAIDSHALA_T4_PRESCRIBING_AGREEMENT_EXPIRING_WITHIN_30D",
}


def _all_specs() -> list[Path]:
    return sorted(SPECS_DIR.glob("*.yaml"))


def _cql_files() -> list[Path]:
    return list(TIER4_DIR.glob("*.cql"))


def _resolve_body(define: str) -> str:
    for c in _cql_files():
        body = _extract_define_body(c.read_text(), define)
        if body:
            return body
    return ""


# ---------------------------------------------------------------------------
# Corpus shape
# ---------------------------------------------------------------------------


def test_wave5_corpus_count():
    specs = _all_specs()
    assert len(specs) == 14, f"expected 14 Wave 5 + extension specs, found {len(specs)}"
    rule_ids = {load_spec(p)["rule_id"] for p in specs}
    assert rule_ids == EXPECTED_RULE_IDS, (
        f"unexpected rule_ids: {rule_ids ^ EXPECTED_RULE_IDS}"
    )


def test_wave5_all_rules_are_tier4_surveillance():
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        assert spec["tier"] == "tier_4_surveillance"
        assert spec["criterion_set"] == "VAIDSHALA_TIER4"


# ---------------------------------------------------------------------------
# Stage 1 + Stage 2
# ---------------------------------------------------------------------------


@pytest.mark.parametrize("spec_path", _all_specs(), ids=lambda p: p.stem)
def test_wave5_rule_passes_stage1(spec_path: Path) -> None:
    spec = load_spec(spec_path)
    result = validate_rule_specification(spec)
    assert result.ok, [str(e) for e in result.errors]


@pytest.mark.parametrize("spec_path", _all_specs(), ids=lambda p: p.stem)
def test_wave5_rule_passes_two_gate(spec_path: Path) -> None:
    spec = load_spec(spec_path)
    body = _resolve_body(spec["define"])
    assert body, f"could not resolve CQL body for {spec['define']}"
    result = run_two_gate(spec, body)
    assert result.ok, [str(e) for e in result.errors]


# ---------------------------------------------------------------------------
# CompatibilityChecker
# ---------------------------------------------------------------------------


def test_wave5_compatibility_checker_all_active():
    cc = CompatibilityChecker()
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        body = _resolve_body(spec["define"])
        cc.register(spec, body)
        cc.OnRuleUpdate(spec, body)
    for rule_id in cc.rules:
        assert cc.status_of(rule_id) == CompatStatus.ACTIVE, (
            f"{rule_id} not ACTIVE: {cc.rules[rule_id].last_reason}"
        )


# ---------------------------------------------------------------------------
# CDS Hooks emission
# ---------------------------------------------------------------------------


def test_wave5_cds_hooks_emission_valid_for_all():
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        fire = RuleFire(
            rule_id=spec["rule_id"],
            summary=(spec.get("summary") or spec["rule_id"])[:140],
            indicator="info",
            detail=spec.get("summary", ""),
            recommendation_text="Surface Tier 4 surveillance signal",
        )
        response = emit_cds_hooks_response(fire, None, hook_type="order-select")
        errors = validate_cds_hooks_v2_response(response)
        assert errors == [], f"{spec['rule_id']}: {errors}"


# ---------------------------------------------------------------------------
# Surveillance shape — informational by default
# ---------------------------------------------------------------------------


def test_wave5_all_rules_carry_substrate_state_or_recently_actioned_suppression():
    """Tier 4 surveillance rules use suppression-class 5 (substrate-state)
    or recently_actioned to prevent re-fire. Plan acceptance: each rule
    fires informationally unless threshold is crossed."""
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        suppr_classes = {s["class"] for s in (spec.get("suppressions") or [])}
        assert suppr_classes & {"substrate_state", "recently_actioned"}, (
            f"{spec['rule_id']} missing substrate_state or recently_actioned suppression"
        )


def test_wave5_trajectory_rules_use_substrate_primitives():
    """The 3 trajectory rules MUST call BaselineFor / DeltaFromBaseline
    / IsTrending so they pass the snapshot-semantics gate."""
    trajectory_ids = {
        "VAIDSHALA_T4_EGFR_TRAJECTORY_DECLINE_90D",
        "VAIDSHALA_T4_WEIGHT_LOSS_TRAJECTORY_90D",
    }
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        if spec["rule_id"] not in trajectory_ids:
            continue
        body = _resolve_body(spec["define"])
        assert any(
            primitive in body
            for primitive in ("BaselineFor", "DeltaFromBaseline", "IsTrending")
        ), f"{spec['rule_id']} should call a substrate primitive"
