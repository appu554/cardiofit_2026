"""Wave-extension batch (2026-05) cross-tier validation.

Acceptance test for the gap-analysis dispatch that ships:
  - 15 more Tier 2 deprescribing rules (4 STOPP, 4 START, 4 Beers, 3 Wang)
  - 10 more Tier 3 quality-gap rules (4 PHARMA-Care, 3 transition, 3 AN-ACC)
  - 8  more Tier 4 surveillance rules (2 trajectory, 6 lifecycle)
  Total: 33 new rules across all three tiers.

This test guarantees that when run as a single batch, every newly-added
rule clears Stage 1, the two-gate validator, the CompatibilityChecker
(ACTIVE state), and CDS Hooks v2.0 emission. It complements the
per-tier batch tests by enforcing the cross-tier total counts.
"""

from __future__ import annotations

from pathlib import Path

from cds_hooks_emitter import (
    RuleFire,
    emit_cds_hooks_response,
    validate_cds_hooks_v2_response,
)
from compatibility_checker import CompatStatus, CompatibilityChecker
from rule_specification_validator import load_spec, validate_rule_specification
from two_gate_validator import _extract_define_body, run_two_gate

LIB_ROOT = Path(__file__).resolve().parents[2] / "cql-libraries"
TIER2_DIR = LIB_ROOT / "tier-2-deprescribing"
TIER3_DIR = LIB_ROOT / "tier-3-quality-gap"
TIER4_DIR = LIB_ROOT / "tier-4-surveillance"

# Wave-extension rule_ids (must equal what the per-tier tests assert).
WAVE_EXT_TIER2 = {
    "STOPP_D5_LONG_TERM_BENZODIAZEPINE_HYPNOTIC",
    "STOPP_F2_PPI_UNCOMPLICATED_PUD_OVER_8_WEEKS",
    "STOPP_K1_ANTICHOLINERGIC_IN_DELIRIUM_OR_DEMENTIA",
    "STOPP_J6_SULFONYLUREA_HBA1C_BELOW_7",
    "START_B5_BETA_BLOCKER_POST_MI_REDUCED_LVEF",
    "START_D2_CALCIUM_VITAMIN_D_OSTEOPOROSIS",
    "START_E1_BONE_PROTECTIVE_THERAPY_OSTEOPOROSIS",
    "START_F4_INFLUENZA_VACCINE_ANNUAL_ELDERLY",
    "BEERS_2023_SLIDING_SCALE_INSULIN_NURSING_HOME",
    "BEERS_2023_STRONG_OPIOID_FIRST_LINE_ELDERLY",
    "BEERS_2023_NSAID_IN_CKD_STAGE_3_PLUS",
    "BEERS_2023_BENZODIAZEPINE_IN_ELDERLY",
    "WANG_2024_ANTICHOLINERGIC_COGNITIVE_IMPAIRMENT_AU",
    "WANG_2024_STRONG_OPIOID_WITHOUT_CANCER_PAIN_AU",
    "WANG_2024_LONG_TERM_PPI_WITHOUT_INDICATION_AU",
}
WAVE_EXT_TIER3 = {
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
WAVE_EXT_TIER4 = {
    "VAIDSHALA_T4_SODIUM_TRAJECTORY_DELTA_90D",
    "VAIDSHALA_T4_BMI_TRAJECTORY_DELTA_180D",
    "VAIDSHALA_T4_PPI_REVIEW_OVERDUE_6MO",
    "VAIDSHALA_T4_STATIN_REVIEW_OVERDUE_12MO",
    "VAIDSHALA_T4_OPIOID_REVIEW_OVERDUE_3MO",
    "VAIDSHALA_T4_ANTIMICROBIAL_REVIEW_OVERDUE_7D",
    "VAIDSHALA_T4_PRESCRIBER_CREDENTIAL_EXPIRING_WITHIN_30D",
    "VAIDSHALA_T4_PRESCRIBING_AGREEMENT_EXPIRING_WITHIN_30D",
}

ALL_WAVE_EXT = WAVE_EXT_TIER2 | WAVE_EXT_TIER3 | WAVE_EXT_TIER4


def _all_specs(tier_dir: Path) -> list[Path]:
    return sorted((tier_dir / "specs").glob("*.yaml"))


def _resolve_body(define: str, tier_dir: Path) -> str:
    for c in tier_dir.glob("*.cql"):
        body = _extract_define_body(c.read_text(), define)
        if body:
            return body
    return ""


def _wave_ext_specs() -> list[tuple[Path, Path]]:
    """Return list of (spec_path, tier_dir) tuples for wave-extension rules."""
    out: list[tuple[Path, Path]] = []
    for tier_dir in (TIER2_DIR, TIER3_DIR, TIER4_DIR):
        for sp in _all_specs(tier_dir):
            spec = load_spec(sp)
            if spec["rule_id"] in ALL_WAVE_EXT:
                out.append((sp, tier_dir))
    return out


def test_wave_extension_total_count():
    pairs = _wave_ext_specs()
    assert len(pairs) == 33, (
        f"expected 33 wave-extension specs (15+10+8), found {len(pairs)}"
    )


def test_wave_extension_per_tier_counts():
    pairs = _wave_ext_specs()
    rule_ids = {load_spec(sp)["rule_id"] for sp, _ in pairs}
    assert rule_ids & WAVE_EXT_TIER2 == WAVE_EXT_TIER2, "Tier 2 wave-ext set mismatch"
    assert rule_ids & WAVE_EXT_TIER3 == WAVE_EXT_TIER3, "Tier 3 wave-ext set mismatch"
    assert rule_ids & WAVE_EXT_TIER4 == WAVE_EXT_TIER4, "Tier 4 wave-ext set mismatch"


def test_wave_extension_all_pass_stage1():
    for sp, _ in _wave_ext_specs():
        spec = load_spec(sp)
        result = validate_rule_specification(spec)
        assert result.ok, (sp.name, [str(e) for e in result.errors])


def test_wave_extension_all_pass_two_gate():
    for sp, tier_dir in _wave_ext_specs():
        spec = load_spec(sp)
        body = _resolve_body(spec["define"], tier_dir)
        assert body, f"could not resolve CQL body for {spec['define']}"
        result = run_two_gate(spec, body)
        assert result.ok, (sp.name, [str(e) for e in result.errors])


def test_wave_extension_all_compatibility_active():
    cc = CompatibilityChecker()
    for sp, tier_dir in _wave_ext_specs():
        spec = load_spec(sp)
        body = _resolve_body(spec["define"], tier_dir)
        cc.register(spec, body)
        cc.OnRuleUpdate(spec, body)
    for rid in cc.rules:
        if rid in ALL_WAVE_EXT:
            assert cc.status_of(rid) == CompatStatus.ACTIVE, (
                f"{rid} not ACTIVE: {cc.rules[rid].last_reason}"
            )


def test_wave_extension_all_emit_valid_cds_hooks():
    indicator_by_tier = {
        "tier_2_deprescribing": "warning",
        "tier_3_quality_gap": "info",
        "tier_4_surveillance": "info",
    }
    for sp, _ in _wave_ext_specs():
        spec = load_spec(sp)
        fire = RuleFire(
            rule_id=spec["rule_id"],
            summary=(spec.get("summary") or spec["rule_id"])[:140],
            indicator=indicator_by_tier[spec["tier"]],
            recommendation_text="Apply suggested action",
        )
        response = emit_cds_hooks_response(fire, None, hook_type="order-select")
        errors = validate_cds_hooks_v2_response(response)
        assert errors == [], f"{spec['rule_id']}: {errors}"


def test_wave_extension_real_published_citations_for_published_tier2():
    """Citation discipline regression: 15 Tier 2 rules cite real
    STOPP/START/Beers/Wang criterion identifiers (no placeholders)."""
    expected = {
        "STOPP_D5_LONG_TERM_BENZODIAZEPINE_HYPNOTIC": ("STOPP_V3", "STOPP-V3-D5"),
        "STOPP_F2_PPI_UNCOMPLICATED_PUD_OVER_8_WEEKS": ("STOPP_V3", "STOPP-V3-F2"),
        "STOPP_K1_ANTICHOLINERGIC_IN_DELIRIUM_OR_DEMENTIA": ("STOPP_V3", "STOPP-V3-K1"),
        "STOPP_J6_SULFONYLUREA_HBA1C_BELOW_7": ("STOPP_V3", "STOPP-V3-J6"),
        "START_B5_BETA_BLOCKER_POST_MI_REDUCED_LVEF": ("START_V3", "START-V3-B5"),
        "START_D2_CALCIUM_VITAMIN_D_OSTEOPOROSIS": ("START_V3", "START-V3-D2"),
        "START_E1_BONE_PROTECTIVE_THERAPY_OSTEOPOROSIS": ("START_V3", "START-V3-E1"),
        "START_F4_INFLUENZA_VACCINE_ANNUAL_ELDERLY": ("START_V3", "START-V3-F4"),
        "BEERS_2023_SLIDING_SCALE_INSULIN_NURSING_HOME": (
            "BEERS_2023", "BEERS-2023-K1-SLIDING-SCALE-INSULIN",
        ),
        "BEERS_2023_STRONG_OPIOID_FIRST_LINE_ELDERLY": (
            "BEERS_2023", "BEERS-2023-K7-OPIOID-FIRST-LINE",
        ),
        "BEERS_2023_NSAID_IN_CKD_STAGE_3_PLUS": (
            "BEERS_2023", "BEERS-2023-H-NSAID-CKD",
        ),
        "BEERS_2023_BENZODIAZEPINE_IN_ELDERLY": (
            "BEERS_2023", "BEERS-2023-G-BENZODIAZEPINE",
        ),
        "WANG_2024_ANTICHOLINERGIC_COGNITIVE_IMPAIRMENT_AU": (
            "PIMS_WANG", "WANG-2024-AU-PIMS-1",
        ),
        "WANG_2024_STRONG_OPIOID_WITHOUT_CANCER_PAIN_AU": (
            "PIMS_WANG", "WANG-2024-AU-PIMS-7",
        ),
        "WANG_2024_LONG_TERM_PPI_WITHOUT_INDICATION_AU": (
            "PIMS_WANG", "WANG-2024-AU-PIMS-11",
        ),
    }
    for sp, _ in _wave_ext_specs():
        spec = load_spec(sp)
        rid = spec["rule_id"]
        if rid in expected:
            cs, cid = expected[rid]
            assert spec["criterion_set"] == cs, f"{rid}: {spec['criterion_set']} != {cs}"
            assert spec["criterion_id"] == cid, f"{rid}: {spec['criterion_id']} != {cid}"
