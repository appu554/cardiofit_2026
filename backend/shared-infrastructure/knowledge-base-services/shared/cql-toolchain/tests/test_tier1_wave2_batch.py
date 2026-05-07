"""Wave 2 batch acceptance — all 25 Tier 1 rules.

Plan acceptance (Wave 2 exit) requires that ~25 Tier 1 rule
specifications:

  1. Validate against rule_specification.v2.json (Stage 1).
  2. Pass the two-gate validator (snapshot + substrate gates).
  3. Show ACTIVE in CompatibilityChecker.
  4. Emit a CDS Hooks v2.0-valid response via the emitter.

This module exercises 1-4 in a single batch over every spec under
shared/cql-libraries/tier-1-immediate-safety/specs/ and reports
counts per gate.

Additionally, asserts:
  - At least 5 rules carry a Class 5 (substrate-state) suppression.
  - At least 5 rules carry a Class 6 (authorisation-context) suppression.
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

TIER1_DIR = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "tier-1-immediate-safety"
)
SPECS_DIR = TIER1_DIR / "specs"


def _all_specs() -> list[Path]:
    return sorted(SPECS_DIR.glob("*.yaml"))


def _cql_files() -> list[Path]:
    return list(TIER1_DIR.glob("*.cql"))


def _resolve_body(define: str) -> str:
    for c in _cql_files():
        body = _extract_define_body(c.read_text(), define)
        if body:
            return body
    return ""


def test_wave2_corpus_count():
    specs = _all_specs()
    # ~25 Tier 1 rules per Wave 2 plan exit.
    assert len(specs) == 25, f"expected 25 Tier 1 specs, found {len(specs)}"


@pytest.mark.parametrize("spec_path", _all_specs(), ids=lambda p: p.stem)
def test_wave2_rule_passes_stage1(spec_path: Path) -> None:
    spec = load_spec(spec_path)
    result = validate_rule_specification(spec)
    assert result.ok, [str(e) for e in result.errors]


@pytest.mark.parametrize("spec_path", _all_specs(), ids=lambda p: p.stem)
def test_wave2_rule_passes_two_gate(spec_path: Path) -> None:
    spec = load_spec(spec_path)
    body = _resolve_body(spec["define"])
    assert body, f"could not resolve CQL body for {spec['define']}"
    result = run_two_gate(spec, body)
    assert result.ok, [str(e) for e in result.errors]
    assert result.snapshot_gate.ok
    assert result.substrate_gate.ok


def test_wave2_compatibility_checker_all_active():
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


def test_wave2_cds_hooks_emission_valid_for_all():
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        fire = RuleFire(
            rule_id=spec["rule_id"],
            summary=(spec.get("summary") or spec["rule_id"])[:140],
            indicator="warning",
            detail=spec.get("summary", ""),
            recommendation_text="Apply suggested action",
        )
        response = emit_cds_hooks_response(fire, None, hook_type="order-select")
        errors = validate_cds_hooks_v2_response(response)
        assert errors == [], f"{spec['rule_id']}: {errors}"


def test_wave2_class5_substrate_state_suppression_coverage():
    """Wave 2 exit: at least 5 rules demonstrate Class 5 (substrate_state)."""
    count = 0
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        for s in spec.get("suppressions", []) or []:
            if s.get("class") == "substrate_state":
                count += 1
                break
    assert count >= 5, f"only {count} rules use substrate_state suppression; expected >=5"


def test_wave2_class6_authorisation_context_suppression_coverage():
    """Wave 2 exit: at least 5 rules demonstrate Class 6 (authorisation_context)."""
    count = 0
    for spec_path in _all_specs():
        spec = load_spec(spec_path)
        for s in spec.get("suppressions", []) or []:
            if s.get("class") == "authorisation_context":
                count += 1
                break
        else:
            # Fall back: rule has authorisation_gating with fallback_routing
            # but no explicit Class 6 suppression — count it only if we
            # have an explicit Class-6 entry above.
            continue
    assert count >= 5, f"only {count} rules use authorisation_context suppression; expected >=5"


def test_wave2_end_to_end_batch_summary():
    """Single-pass end-to-end: validator + CompatibilityChecker + CDS Hooks
    emitter for all 25 rules, with summary counts."""
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
            recommendation_text="Apply suggested action",
        )
        response = emit_cds_hooks_response(fire, None, hook_type="order-select")
        if not validate_cds_hooks_v2_response(response):
            counts["cds_hooks"] += 1

    assert counts["stage1"] == total
    assert counts["two_gate"] == total
    assert counts["active"] == total
    assert counts["cds_hooks"] == total
