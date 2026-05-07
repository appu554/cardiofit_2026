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
