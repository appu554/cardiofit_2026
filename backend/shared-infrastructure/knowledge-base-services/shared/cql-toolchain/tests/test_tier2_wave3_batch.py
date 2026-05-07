"""Wave 3 batch acceptance — 6 Tier 2 deprescribing rules.

Plan acceptance (Wave 3 vertical slice) requires that the 6 grounded
Tier 2 rule specifications:

  1. Validate against rule_specification.v2.json (Stage 1).
  2. Pass the two-gate validator (snapshot + substrate gates).
  3. Show ACTIVE in CompatibilityChecker.
  4. Emit a CDS Hooks v2.0-valid response via the emitter.
  5. Round-trip through the GovernancePromoter and produce a signed
     package on disk.

The 6 rules are:
  - STOPP_B1_ASPIRIN_PRIMARY_PREVENTION       (StoppV3.cql)
  - START_A1_ANTICOAGULATION_AF               (StartV3.cql)
  - BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE    (Beers2023.cql)
  - WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU       (Wang2024.cql)
  - ADG2025_PPI_STEP_DOWN_PROTOCOL            (ADG2025.cql)
  - ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW     (ADG2025.cql)
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

TIER2_DIR = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "tier-2-deprescribing"
)
SPECS_DIR = TIER2_DIR / "specs"

EXPECTED_RULE_IDS = {
    "STOPP_B1_ASPIRIN_PRIMARY_PREVENTION",
    "START_A1_ANTICOAGULATION_AF",
    "BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE",
    "WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU",
    "ADG2025_PPI_STEP_DOWN_PROTOCOL",
    "ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW",
}


def _all_specs() -> list[Path]:
    return sorted(SPECS_DIR.glob("*.yaml"))


def _cql_files() -> list[Path]:
    return list(TIER2_DIR.glob("*.cql"))


def _resolve_body(define: str) -> str:
    for c in _cql_files():
        body = _extract_define_body(c.read_text(), define)
        if body:
            return body
    return ""


# ---------------------------------------------------------------------------
# Corpus shape
# ---------------------------------------------------------------------------


def test_wave3_corpus_count():
    specs = _all_specs()
    assert len(specs) == 6, f"expected 6 Wave 3 specs, found {len(specs)}"
    rule_ids = {load_spec(p)["rule_id"] for p in specs}
    assert rule_ids == EXPECTED_RULE_IDS, (
        f"unexpected rule_ids: {rule_ids ^ EXPECTED_RULE_IDS}"
    )


# ---------------------------------------------------------------------------
# Stage 1 + Stage 2
# ---------------------------------------------------------------------------


@pytest.mark.parametrize("spec_path", _all_specs(), ids=lambda p: p.stem)
def test_wave3_rule_passes_stage1(spec_path: Path) -> None:
    spec = load_spec(spec_path)
    result = validate_rule_specification(spec)
    assert result.ok, [str(e) for e in result.errors]


@pytest.mark.parametrize("spec_path", _all_specs(), ids=lambda p: p.stem)
def test_wave3_rule_passes_two_gate(spec_path: Path) -> None:
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


def test_wave3_compatibility_checker_all_active():
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


def test_wave3_cds_hooks_emission_valid_for_all():
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        fire = RuleFire(
            rule_id=spec["rule_id"],
            summary=(spec.get("summary") or spec["rule_id"])[:140],
            indicator="warning",
            detail=spec.get("summary", ""),
            recommendation_text="Apply suggested deprescribing action",
        )
        response = emit_cds_hooks_response(fire, None, hook_type="order-select")
        errors = validate_cds_hooks_v2_response(response)
        assert errors == [], f"{spec['rule_id']}: {errors}"


# ---------------------------------------------------------------------------
# Real-criterion citation check (regression on no-fabrication discipline)
# ---------------------------------------------------------------------------


def test_wave3_published_rules_cite_real_criterion_ids():
    """The 4 published rules (STOPP/START/Beers/Wang) cite real,
    citable criterion identifiers — not placeholders."""
    expected_published = {
        "STOPP_B1_ASPIRIN_PRIMARY_PREVENTION": ("STOPP_V3", "STOPP-V3-B1"),
        "START_A1_ANTICOAGULATION_AF": ("START_V3", "START-V3-A1"),
        "BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE": (
            "BEERS_2023",
            "BEERS-2023-T2-ANTIHISTAMINE",
        ),
        "WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU": (
            "PIMS_WANG",
            "WANG-2024-AU-PIMS-3",
        ),
    }
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        rid = spec["rule_id"]
        if rid not in expected_published:
            continue
        criterion_set, criterion_id = expected_published[rid]
        assert spec["criterion_set"] == criterion_set, (
            f"{rid}: criterion_set {spec['criterion_set']} != {criterion_set}"
        )
        assert spec["criterion_id"] == criterion_id, (
            f"{rid}: criterion_id {spec['criterion_id']} != {criterion_id}"
        )


def test_wave3_adg2025_rules_carry_layer1_bind_marker():
    """The 2 ADG 2025 rules MUST carry TODO(layer1-bind) markers in
    their CQL bodies AND be flagged as placeholder in their spec
    criterion_id (suffix '-PLACEHOLDER')."""
    adg_rules = {
        "ADG2025_PPI_STEP_DOWN_PROTOCOL",
        "ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW",
    }
    cql_text = "\n".join(c.read_text() for c in _cql_files())
    todo_count = cql_text.count("TODO(layer1-bind)")
    assert todo_count >= 2, (
        f"expected at least 2 TODO(layer1-bind) markers in tier-2 CQL, "
        f"found {todo_count}"
    )
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        if spec["rule_id"] in adg_rules:
            assert spec["criterion_id"].endswith("-PLACEHOLDER"), (
                f"{spec['rule_id']}: criterion_id should be placeholder "
                f"until Pipeline-2 emits real IDs; got {spec['criterion_id']}"
            )


# ---------------------------------------------------------------------------
# Governance promotion — 6/6 signed packages
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
    """Map a spec.library to the on-disk CQL library file."""
    return TIER2_DIR / f"{spec['library']}.cql"


def test_wave3_governance_promotion_six_signed_packages(
    tmp_path, signing_key, two_signatures
):
    """Round-trip all 6 Wave 3 rules through GovernancePromoter and
    assert 6 signed packages exist on disk."""
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

    assert len(signed_paths) == 6
    # Each package is uniquely named and SHA-different
    shas = set()
    for p in signed_paths:
        import json as _json
        pkg = _json.loads(p.read_text())
        shas.add(pkg["content_sha"])
    assert len(shas) == 6, "expected 6 distinct content_sha values"


# ---------------------------------------------------------------------------
# End-to-end batch summary
# ---------------------------------------------------------------------------


def test_wave3_end_to_end_batch_summary():
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
            indicator="warning",
            recommendation_text="Apply suggested deprescribing action",
        )
        response = emit_cds_hooks_response(fire, None, hook_type="order-select")
        if not validate_cds_hooks_v2_response(response):
            counts["cds_hooks"] += 1

    assert total == 6
    assert counts["stage1"] == total
    assert counts["two_gate"] == total
    assert counts["active"] == total
    assert counts["cds_hooks"] == total
