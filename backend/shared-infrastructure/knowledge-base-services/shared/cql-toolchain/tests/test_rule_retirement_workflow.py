"""Wave 6 Task 1 — pytest coverage for rule_retirement_workflow."""

from __future__ import annotations

import pytest

from rule_retirement_workflow import (
    ClinicalOverride,
    Decision,
    RetirementWorkflow,
    RuleStats,
)


@pytest.fixture
def stats_corpus():
    return [
        RuleStats(
            rule_id="RULE_QUIET",
            fire_count=20, override_count=1,
            override_rate=0.05, flag_retire=False,
        ),
        RuleStats(
            rule_id="RULE_NOISY",
            fire_count=2, override_count=8,
            override_rate=0.80, flag_retire=True,
        ),
        RuleStats(
            rule_id="RULE_NOISY_BUT_KEPT",
            fire_count=1, override_count=9,
            override_rate=0.90, flag_retire=True,
        ),
    ]


def test_quiet_rule_kept_below_threshold(stats_corpus):
    wf = RetirementWorkflow()
    decisions = wf.evaluate(stats_corpus)
    quiet = next(d for d in decisions if d.rule_id == "RULE_QUIET")
    assert quiet.decision == Decision.KEEP_BELOW_THRESHOLD


def test_noisy_rule_retired(stats_corpus):
    wf = RetirementWorkflow()
    decisions = wf.evaluate(stats_corpus)
    noisy = next(d for d in decisions if d.rule_id == "RULE_NOISY")
    assert noisy.decision == Decision.RETIRE


def test_clinical_override_keeps_rule(stats_corpus):
    wf = RetirementWorkflow()
    wf.register_clinical_override(
        ClinicalOverride(
            rule_id="RULE_NOISY_BUT_KEPT",
            rationale="critical safety surface; override-and-keep",
            approver="Dr. Lead",
            approved_at="2026-05-06T10:00:00Z",
        )
    )
    decisions = wf.evaluate(stats_corpus)
    kept = next(d for d in decisions if d.rule_id == "RULE_NOISY_BUT_KEPT")
    assert kept.decision == Decision.KEEP_CLINICAL_OVERRIDE
    assert "Dr. Lead" in kept.rationale


def test_retirement_queue_only_returns_retire(stats_corpus):
    wf = RetirementWorkflow()
    queue = wf.retirement_queue(stats_corpus)
    assert {d.rule_id for d in queue} == {"RULE_NOISY", "RULE_NOISY_BUT_KEPT"}


def test_clinical_override_requires_rationale():
    wf = RetirementWorkflow()
    with pytest.raises(ValueError):
        wf.register_clinical_override(
            ClinicalOverride(
                rule_id="X", rationale="   ",
                approver="Dr. Lead", approved_at="2026-05-06T10:00:00Z",
            )
        )
