"""Wave 1 acceptance — three anchor rules end-to-end.

Plan acceptance (Wave 1 exit) requires that the three anchor defines:
  - PPI deprescribing (Tier 2)
  - Hyperkalemia Trajectory (Tier 1)
  - Antipsychotic Consent Gating (Tier 1)

each:
  1. Compile under the validator (two_gate_validator both gates green)
  2. Pass CompatibilityChecker (status == ACTIVE)
  3. Round-trip through CDS Hooks emitter (v2.0-valid response)
  4. End-to-end promote to a signed package via governance promoter

This module exercises 1–4 in a single integration walk per rule and
serves as the acceptance gate for the Wave 1 dispatch.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from cds_hooks_emitter import (
    RuleFire,
    apply_plan_definition,
    emit_cds_hooks_response,
    load_bundle,
    validate_cds_hooks_v2_response,
)
from compatibility_checker import CompatStatus, CompatibilityChecker
from governance_promoter import GovernancePromoter, Signature
from rule_specification_validator import load_spec
from two_gate_validator import _extract_define_body, run_two_gate

EXAMPLES = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "examples"
)
RULES = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "rules"
)
PLAN_DEFS = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "plan-definitions"
)


ANCHORS = [
    {
        "rule_id": "PPI_LONG_TERM_NO_INDICATION",
        "spec": "ppi-deprescribe.yaml",
        "library": "TierTwoDeprescribing.cql",
        "plan_def": "example-ppi-deprescribe.json",  # has a PD
        "indicator": "warning",
    },
    {
        "rule_id": "HYPERKALEMIA_RISK_TRAJECTORY",
        "spec": "hyperkalemia-trajectory.yaml",
        "library": "TierOneImmediateSafety.cql",
        "plan_def": None,  # no PD scaffolded for this rule yet
        "indicator": "critical",
    },
    {
        "rule_id": "ANTIPSYCHOTIC_CONSENT_MISSING",
        "spec": "antipsychotic-consent-gating.yaml",
        "library": "TierOneImmediateSafety.cql",
        "plan_def": None,
        "indicator": "warning",
    },
]


@pytest.fixture
def signing_key() -> Ed25519PrivateKey:
    return Ed25519PrivateKey.generate()


@pytest.fixture
def two_signatures() -> list[Signature]:
    return [
        Signature(role="CLINICAL_REVIEWER", signer_id="dr.jane@vaidshala"),
        Signature(role="MEDICAL_DIRECTOR", signer_id="dr.bob@vaidshala"),
    ]


@pytest.mark.parametrize("anchor", ANCHORS, ids=lambda a: a["rule_id"])
def test_anchor_rule_end_to_end(anchor, tmp_path, signing_key, two_signatures):
    spec = load_spec(EXAMPLES / anchor["spec"])
    cql_body = _extract_define_body(
        (RULES / anchor["library"]).read_text(),
        spec["define"],
    )

    # 1. Both gates green
    gate = run_two_gate(spec, cql_body)
    assert gate.ok, [str(e) for e in gate.errors]
    assert gate.snapshot_gate.ok
    assert gate.substrate_gate.ok

    # 2. CompatibilityChecker reports ACTIVE
    cc = CompatibilityChecker()
    cc.register(spec, cql_body)
    cc.OnRuleUpdate(spec, cql_body)
    assert cc.status_of(anchor["rule_id"]) == CompatStatus.ACTIVE

    # 3. CDS Hooks emitter round-trip
    fire = RuleFire(
        rule_id=anchor["rule_id"],
        summary=spec.get("summary", anchor["rule_id"])[:140],
        indicator=anchor["indicator"],
        detail=spec.get("summary", ""),
        recommendation_text="Apply suggested action",
    )
    req_orch = None
    if anchor["plan_def"]:
        bundle = load_bundle(PLAN_DEFS / anchor["plan_def"])
        req_orch = apply_plan_definition(bundle, fire)
    response = emit_cds_hooks_response(fire, req_orch, hook_type="order-select")
    errors = validate_cds_hooks_v2_response(response)
    assert errors == [], errors

    # 4. End-to-end promote
    promoter = GovernancePromoter(
        signing_key=signing_key,
        signed_dir=tmp_path / "signed",
        pending_dir=tmp_path / "pending",
    )
    result = promoter.promote(
        spec_path=EXAMPLES / anchor["spec"],
        cql_library_path=RULES / anchor["library"],
        signatures=two_signatures,
    )
    assert result.ok, result.errors
    package = json.loads(result.signed_package_path.read_text())
    assert package["rule_id"] == anchor["rule_id"]
    assert len(package["approver_signatures"]) == 2
