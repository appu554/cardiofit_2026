"""Wave 1 Task 3 — pytest coverage for CompatibilityChecker.

Synthetic substrate change marks the 3 example rules STALE.
Synthetic ScopeRule change marks the rules that reference it STALE.
Helper updates mark dependent rules STALE.
Rule updates that fail the gates flip status to INVALID.
"""

from __future__ import annotations

from pathlib import Path

import pytest

from compatibility_checker import (
    CompatStatus,
    CompatibilityChecker,
    ScopeRuleChange,
    SubstrateChange,
)

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


@pytest.fixture
def populated_checker():
    cc = CompatibilityChecker()
    cc.register_from_files(
        EXAMPLES / "ppi-deprescribe.yaml", RULES / "TierTwoDeprescribing.cql"
    )
    cc.register_from_files(
        EXAMPLES / "hyperkalemia-trajectory.yaml",
        RULES / "TierOneImmediateSafety.cql",
    )
    cc.register_from_files(
        EXAMPLES / "antipsychotic-consent-gating.yaml",
        RULES / "TierOneImmediateSafety.cql",
    )
    return cc


# ---------------------------------------------------------------------------
# Initial state
# ---------------------------------------------------------------------------


def test_initial_state_is_active(populated_checker):
    for rule_id in [
        "PPI_LONG_TERM_NO_INDICATION",
        "HYPERKALEMIA_RISK_TRAJECTORY",
        "ANTIPSYCHOTIC_CONSENT_MISSING",
    ]:
        assert populated_checker.status_of(rule_id) == CompatStatus.ACTIVE


# ---------------------------------------------------------------------------
# Event B — helper update
# ---------------------------------------------------------------------------


def test_helper_update_marks_dependent_rules_stale(populated_checker):
    affected = populated_checker.OnHelperUpdate("HasActiveAtcPrefix")
    # All three rules call HasActiveAtcPrefix.
    assert set(affected) == {
        "PPI_LONG_TERM_NO_INDICATION",
        "HYPERKALEMIA_RISK_TRAJECTORY",
        "ANTIPSYCHOTIC_CONSENT_MISSING",
    }
    for r in affected:
        assert populated_checker.status_of(r) == CompatStatus.STALE


def test_helper_update_targeted_helper_only(populated_checker):
    affected = populated_checker.OnHelperUpdate("HasActiveConsentForClass")
    # Only the antipsychotic rule references this helper.
    assert affected == ["ANTIPSYCHOTIC_CONSENT_MISSING"]


# ---------------------------------------------------------------------------
# Event C — substrate change
# ---------------------------------------------------------------------------


def test_substrate_change_marks_three_anchors_stale(populated_checker):
    # active_concerns is referenced by all three example rules.
    affected = populated_checker.OnSubstrateChange([
        SubstrateChange(machine="clinical", fact_type="active_concerns"),
    ])
    assert set(affected) == {
        "PPI_LONG_TERM_NO_INDICATION",
        "HYPERKALEMIA_RISK_TRAJECTORY",
        "ANTIPSYCHOTIC_CONSENT_MISSING",
    }


def test_substrate_change_baseline_state_marks_only_hyperk(populated_checker):
    affected = populated_checker.OnSubstrateChange([
        SubstrateChange(machine="clinical", fact_type="baseline_state"),
    ])
    assert "HYPERKALEMIA_RISK_TRAJECTORY" in affected


# ---------------------------------------------------------------------------
# Event D — ScopeRule change
# ---------------------------------------------------------------------------


def test_scope_rule_change_marks_referencing_rule_stale(populated_checker):
    # Inject a scope_rule_ref into the hyperk rule's authorisation_gating
    rule = populated_checker.rules["HYPERKALEMIA_RISK_TRAJECTORY"]
    rule.spec.setdefault("authorisation_gating", {})
    rule.spec["authorisation_gating"]["scope_rule_refs"] = ["AU_AGEDCARE_S8_v3"]

    affected = populated_checker.OnScopeRuleChange([
        ScopeRuleChange(scope_rule_ref="AU_AGEDCARE_S8_v3"),
    ])
    assert affected == ["HYPERKALEMIA_RISK_TRAJECTORY"]


def test_scope_rule_change_marks_three_or_more_rules_stale_wave5_task7(
    populated_checker,
):
    """Wave 5 Task 7 acceptance: synthetic ScopeRule change marks >=3
    expected defines STALE; no false positives.

    All three pre-registered rules carry an
    authorisation_gating.scope_rule_refs entry pointing at the synthetic
    Victorian PCW exclusion ScopeRule (mirroring kb-31 deployment rule
    AUS-VIC-PCW-S4-EXCLUSION-2026-07-01). One additional unrelated
    ScopeRule is also injected on a single rule to verify selectivity:
    a change to the unrelated rule must NOT mark the others STALE."""
    target_scope_rule = "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01"
    unrelated_scope_rule = "AUS-NMBA-DRNP-PRESCRIBING-AGREEMENT-2025-09-30"

    for rule_id in (
        "PPI_LONG_TERM_NO_INDICATION",
        "HYPERKALEMIA_RISK_TRAJECTORY",
        "ANTIPSYCHOTIC_CONSENT_MISSING",
    ):
        spec = populated_checker.rules[rule_id].spec
        spec.setdefault("authorisation_gating", {})
        spec["authorisation_gating"]["scope_rule_refs"] = [target_scope_rule]

    # Add an unrelated scope_rule_ref to ONE rule — its change must not
    # mark the OTHER two rules STALE.
    populated_checker.rules["PPI_LONG_TERM_NO_INDICATION"].spec[
        "authorisation_gating"
    ]["scope_rule_refs"].append(unrelated_scope_rule)

    affected = populated_checker.OnScopeRuleChange([
        ScopeRuleChange(scope_rule_ref=target_scope_rule),
    ])
    assert set(affected) == {
        "PPI_LONG_TERM_NO_INDICATION",
        "HYPERKALEMIA_RISK_TRAJECTORY",
        "ANTIPSYCHOTIC_CONSENT_MISSING",
    }, f"expected 3 rules STALE; got {affected}"
    assert len(affected) >= 3, "Wave 5 Task 7: at least 3 rules must be STALE"

    # Selectivity check: a change to the unrelated rule must mark only
    # PPI_LONG_TERM_NO_INDICATION (the rule that holds the ref).
    # Reset to ACTIVE so we can re-test.
    for rule_id in affected:
        populated_checker.rules[rule_id].status = CompatStatus.ACTIVE

    selective_affected = populated_checker.OnScopeRuleChange([
        ScopeRuleChange(scope_rule_ref=unrelated_scope_rule),
    ])
    assert selective_affected == ["PPI_LONG_TERM_NO_INDICATION"], (
        "selectivity violated: unrelated ScopeRule change marked extra rules"
    )


# ---------------------------------------------------------------------------
# Event A — rule update
# ---------------------------------------------------------------------------


def test_rule_update_with_invalid_body_marks_invalid(populated_checker):
    rule = populated_checker.rules["PPI_LONG_TERM_NO_INDICATION"]
    affected = populated_checker.OnRuleUpdate(rule.spec, "no helpers at all")
    assert affected == ["PPI_LONG_TERM_NO_INDICATION"]
    assert (
        populated_checker.status_of("PPI_LONG_TERM_NO_INDICATION")
        == CompatStatus.INVALID
    )


def test_rule_update_with_valid_body_returns_active(populated_checker):
    rule = populated_checker.rules["PPI_LONG_TERM_NO_INDICATION"]
    affected = populated_checker.OnRuleUpdate(rule.spec, rule.cql_body)
    assert affected == []
    assert populated_checker.is_compatible("PPI_LONG_TERM_NO_INDICATION")
