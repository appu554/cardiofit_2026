"""Wave 4A batch acceptance — 8 Tier 3 quality-gap rules.

Plan acceptance (Wave 4A vertical slice) requires that the 8 grounded
Tier 3 rule specifications:

  1. Validate against rule_specification.v2.json (Stage 1).
  2. Pass the two-gate validator (snapshot + substrate gates).
  3. Show ACTIVE in CompatibilityChecker.
  4. Emit a CDS Hooks v2.0-valid response via the emitter.
  5. Round-trip through the GovernancePromoter and produce a signed
     package on disk.

The 8 rules are:
  - VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE          (PharmaCareStandard5.cql)
  - VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS               (PharmaCareStandard5.cql)
  - VAIDSHALA_PC_D3_ACB_ABOVE_3                        (PharmaCareStandard5.cql)
  - VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM           (PharmaCareStandard5.cql)
  - VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_72H          (CareTransition.cql)
  - VAIDSHALA_TX_RMMR_OVERDUE_6MO                      (CareTransition.cql)
  - VAIDSHALA_ANACC_FUNCTIONAL_DECLINE                 (ANACCDefensibility.cql)
  - VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE               (ANACCDefensibility.cql)
"""

from __future__ import annotations

from pathlib import Path

import pytest
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from cds_hooks_emitter import (
    RuleFire,
    emit_cds_hooks_response,
    validate_cds_hooks_v2_response,
)
from compatibility_checker import CompatStatus, CompatibilityChecker
from governance_promoter import GovernancePromoter, Signature
from rule_specification_validator import load_spec, validate_rule_specification
from two_gate_validator import _extract_define_body, run_two_gate

TIER3_DIR = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "tier-3-quality-gap"
)
SPECS_DIR = TIER3_DIR / "specs"

EXPECTED_RULE_IDS = {
    # Wave 4A vertical slice (8)
    "VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE",
    "VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS",
    "VAIDSHALA_PC_D3_ACB_ABOVE_3",
    "VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM",
    "VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_72H",
    "VAIDSHALA_TX_RMMR_OVERDUE_6MO",
    "VAIDSHALA_ANACC_FUNCTIONAL_DECLINE",
    "VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE",
    # Wave-extension batch 2026-05 (10)
    "VAIDSHALA_PC_D5_PAIN_ASSESSMENT_DOCUMENTATION_GAP",
    "VAIDSHALA_PC_D1_POLYPHARMACY_5PLUS_HIGH_RISK",
    "VAIDSHALA_PC_D3_FALLS_RISK_DRUG_BURDEN",
    "VAIDSHALA_PC_D4_RESTRAINT_WITHOUT_SDM_CONSENT",
    "VAIDSHALA_TX_ADMISSION_RECON_NOT_STARTED_24H",
    "VAIDSHALA_TX_CARE_PLAN_REVIEW_OVERDUE_POST_HOSPITAL",
    "VAIDSHALA_TX_STEWARDSHIP_HANDOFF_DOCUMENTATION_MISSING",
    "VAIDSHALA_ANACC_COGNITIVE_STATUS_EVIDENCE",
    "VAIDSHALA_ANACC_CONTINENCE_BURDEN_EVIDENCE",
    "VAIDSHALA_ANACC_PRESSURE_INJURY_RISK_EVIDENCE",
}


def _all_specs() -> list[Path]:
    return sorted(SPECS_DIR.glob("*.yaml"))


def _cql_files() -> list[Path]:
    return list(TIER3_DIR.glob("*.cql"))


def _resolve_body(define: str) -> str:
    for c in _cql_files():
        body = _extract_define_body(c.read_text(), define)
        if body:
            return body
    return ""


# ---------------------------------------------------------------------------
# Corpus shape
# ---------------------------------------------------------------------------


def test_wave4a_corpus_count():
    specs = _all_specs()
    assert len(specs) == 18, f"expected 18 Wave 4A + extension specs, found {len(specs)}"
    rule_ids = {load_spec(p)["rule_id"] for p in specs}
    assert rule_ids == EXPECTED_RULE_IDS, (
        f"unexpected rule_ids: {rule_ids ^ EXPECTED_RULE_IDS}"
    )


def test_wave4a_all_rules_are_tier3_quality_gap():
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        assert spec["tier"] == "tier_3_quality_gap"
        assert spec["criterion_set"] == "VAIDSHALA_TIER3"


# ---------------------------------------------------------------------------
# Stage 1 + Stage 2
# ---------------------------------------------------------------------------


@pytest.mark.parametrize("spec_path", _all_specs(), ids=lambda p: p.stem)
def test_wave4a_rule_passes_stage1(spec_path: Path) -> None:
    spec = load_spec(spec_path)
    result = validate_rule_specification(spec)
    assert result.ok, [str(e) for e in result.errors]


@pytest.mark.parametrize("spec_path", _all_specs(), ids=lambda p: p.stem)
def test_wave4a_rule_passes_two_gate(spec_path: Path) -> None:
    spec = load_spec(spec_path)
    body = _resolve_body(spec["define"])
    assert body, f"could not resolve CQL body for {spec['define']}"
    result = run_two_gate(spec, body)
    assert result.ok, [str(e) for e in result.errors]
    assert result.snapshot_gate.ok
    assert result.substrate_gate.ok


# ---------------------------------------------------------------------------
# CompatibilityChecker
# ---------------------------------------------------------------------------


def test_wave4a_compatibility_checker_all_active():
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


def test_wave4a_cds_hooks_emission_valid_for_all():
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        fire = RuleFire(
            rule_id=spec["rule_id"],
            summary=(spec.get("summary") or spec["rule_id"])[:140],
            indicator="info",
            detail=spec.get("summary", ""),
            recommendation_text="Review per Tier 3 quality indicator",
        )
        response = emit_cds_hooks_response(fire, None, hook_type="order-select")
        errors = validate_cds_hooks_v2_response(response)
        assert errors == [], f"{spec['rule_id']}: {errors}"


# ---------------------------------------------------------------------------
# Real-framework citation evidence
# ---------------------------------------------------------------------------


def test_wave4a_pharmacare_rules_carry_clinical_author_marker():
    """The 4 PHARMA-Care rules MUST carry TODO(clinical-author) markers
    in their CQL bodies until the published v1 framework PDF lands."""
    pharmacare_rules = {
        "VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS",
        "VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE",
        "VAIDSHALA_PC_D3_ACB_ABOVE_3",
        "VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM",
    }
    cql_text = "\n".join(c.read_text() for c in _cql_files())
    todo_count = cql_text.count("TODO(clinical-author)")
    assert todo_count >= 4, (
        f"expected at least 4 TODO(clinical-author) markers in tier-3 CQL, "
        f"found {todo_count}"
    )
    rule_ids = {load_spec(p)["rule_id"] for p in _all_specs()}
    assert pharmacare_rules.issubset(rule_ids)


def test_wave4a_anacc_rules_cite_real_framework():
    """The 2 AN-ACC rules cite the real AN-ACC published indicators
    (AKPS for functional decline, behavioural-incident counts for
    classes 9-13). The criterion_id prefix carries 'ANACC' and the
    summary mentions the published indicator."""
    anacc_rules = {
        "VAIDSHALA_ANACC_FUNCTIONAL_DECLINE": "akps",
        "VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE": "behavioural",
    }
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        rid = spec["rule_id"]
        if rid not in anacc_rules:
            continue
        assert "ANACC" in spec["criterion_id"], (
            f"{rid}: criterion_id should reference AN-ACC; got {spec['criterion_id']}"
        )
        keyword = anacc_rules[rid]
        assert keyword.lower() in (spec.get("summary") or "").lower(), (
            f"{rid}: summary should reference the {keyword} AN-ACC indicator"
        )


# ---------------------------------------------------------------------------
# Governance promotion — 8/8 signed packages
# ---------------------------------------------------------------------------


@pytest.fixture
def signing_key() -> Ed25519PrivateKey:
    return Ed25519PrivateKey.generate()


@pytest.fixture
def two_signatures() -> list[Signature]:
    return [
        Signature(role="CLINICAL_REVIEWER", signer_id="dr.jane.reviewer@vaidshala"),
        Signature(role="MEDICAL_DIRECTOR", signer_id="dr.bob.director@vaidshala"),
    ]


def _spec_to_library_path(spec: dict) -> Path:
    return TIER3_DIR / f"{spec['library']}.cql"


def test_wave4a_governance_promotion_eight_signed_packages(
    tmp_path, signing_key, two_signatures
):
    """Round-trip all 8 Wave 4A rules through GovernancePromoter and
    assert 8 signed packages exist on disk."""
    promoter = GovernancePromoter(
        signing_key=signing_key,
        signed_dir=tmp_path / "signed",
        pending_dir=tmp_path / "pending",
    )
    signed_paths = []
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        lib_path = _spec_to_library_path(spec)
        assert lib_path.exists(), f"library file missing: {lib_path}"
        result = promoter.promote(
            spec_path=spec_path,
            cql_library_path=lib_path,
            signatures=two_signatures,
        )
        assert result.ok, (spec_path.name, result.errors)
        assert result.signed_package_path is not None
        assert result.signed_package_path.exists()
        signed_paths.append(result.signed_package_path)

    assert len(signed_paths) == 18
    shas = set()
    for p in signed_paths:
        import json as _json
        pkg = _json.loads(p.read_text())
        shas.add(pkg["content_sha"])
    assert len(shas) == 18, "expected 18 distinct content_sha values"


# ---------------------------------------------------------------------------
# End-to-end batch summary
# ---------------------------------------------------------------------------


def test_wave4a_end_to_end_batch_summary():
    cc = CompatibilityChecker()
    counts = {"stage1": 0, "two_gate": 0, "active": 0, "cds_hooks": 0}
    total = 0
    for spec_path in _all_specs():
        total += 1
        spec = load_spec(spec_path)
        body = _resolve_body(spec["define"])

        if validate_rule_specification(spec).ok:
            counts["stage1"] += 1
        if run_two_gate(spec, body).ok:
            counts["two_gate"] += 1

        cc.register(spec, body)
        cc.OnRuleUpdate(spec, body)
        if cc.status_of(spec["rule_id"]) == CompatStatus.ACTIVE:
            counts["active"] += 1

        fire = RuleFire(
            rule_id=spec["rule_id"],
            summary=(spec.get("summary") or spec["rule_id"])[:140],
            indicator="info",
            recommendation_text="Review per Tier 3 quality indicator",
        )
        response = emit_cds_hooks_response(fire, None, hook_type="order-select")
        if not validate_cds_hooks_v2_response(response):
            counts["cds_hooks"] += 1

    assert total == 18
    assert counts["stage1"] == total
    assert counts["two_gate"] == total
    assert counts["active"] == total
    assert counts["cds_hooks"] == total
