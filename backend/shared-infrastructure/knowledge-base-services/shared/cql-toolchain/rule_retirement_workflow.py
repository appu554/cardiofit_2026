"""Rule retirement workflow — Wave 6 Task 1.

Given an override-rate report (the JSON shape produced by the kb-30
analytics tracker), this module produces retirement candidates and
captures the clinical-lead override decision per Layer 3 v2 doc Part
4.2.

A rule reaches the retirement queue when its override rate exceeds the
threshold (default 70%) over the rolling window AND the clinical lead
has NOT marked it for an override-and-keep. The output is a structured
RetirementDecision list suitable for governance routing.

This module is intentionally an in-memory orchestrator: persistence to
the kb-30 audit store and the actual rule retire/promote actions are
V2 work. Wave 6 Task 1 acceptance: the framework ships with tests; the
< 5% library-wide override-rate target is monitored via Tracker.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Iterable


class Decision(str, Enum):
    """Per-rule disposition after the retirement workflow runs."""

    RETIRE = "RETIRE"
    KEEP_CLINICAL_OVERRIDE = "KEEP_CLINICAL_OVERRIDE"
    KEEP_BELOW_THRESHOLD = "KEEP_BELOW_THRESHOLD"


@dataclass
class RuleStats:
    """Mirror of the kb-30 analytics.RuleStats shape."""

    rule_id: str
    fire_count: int
    override_count: int
    override_rate: float
    flag_retire: bool
    window_days: int = 30


@dataclass
class ClinicalOverride:
    """A clinical-lead-override entry. The lead can keep a rule on the
    library despite a high override rate, but must record the
    rationale for the audit chain."""

    rule_id: str
    rationale: str
    approver: str
    approved_at: str  # ISO8601


@dataclass
class RetirementDecision:
    """One row in the retirement output."""

    rule_id: str
    decision: Decision
    override_rate: float
    window_days: int
    rationale: str = ""


@dataclass
class RetirementWorkflow:
    """Orchestrates retirement decisions over a list of RuleStats and a
    list of ClinicalOverrides."""

    threshold: float = 0.70
    overrides: dict[str, ClinicalOverride] = field(default_factory=dict)

    def register_clinical_override(self, override: ClinicalOverride) -> None:
        if not override.rationale.strip():
            raise ValueError(
                "clinical override requires non-empty rationale "
                "(audit chain requirement)"
            )
        self.overrides[override.rule_id] = override

    def evaluate(
        self, stats: Iterable[RuleStats]
    ) -> list[RetirementDecision]:
        out: list[RetirementDecision] = []
        for s in stats:
            if not s.flag_retire:
                out.append(
                    RetirementDecision(
                        rule_id=s.rule_id,
                        decision=Decision.KEEP_BELOW_THRESHOLD,
                        override_rate=s.override_rate,
                        window_days=s.window_days,
                        rationale=(
                            f"override rate {s.override_rate:.2%} below "
                            f"retirement threshold {self.threshold:.0%}"
                        ),
                    )
                )
                continue
            if s.rule_id in self.overrides:
                ov = self.overrides[s.rule_id]
                out.append(
                    RetirementDecision(
                        rule_id=s.rule_id,
                        decision=Decision.KEEP_CLINICAL_OVERRIDE,
                        override_rate=s.override_rate,
                        window_days=s.window_days,
                        rationale=(
                            f"clinical override by {ov.approver} on "
                            f"{ov.approved_at}: {ov.rationale}"
                        ),
                    )
                )
                continue
            out.append(
                RetirementDecision(
                    rule_id=s.rule_id,
                    decision=Decision.RETIRE,
                    override_rate=s.override_rate,
                    window_days=s.window_days,
                    rationale=(
                        f"override rate {s.override_rate:.2%} exceeds "
                        f"retirement threshold {self.threshold:.0%} "
                        f"and no clinical-lead override on file"
                    ),
                )
            )
        return out

    def retirement_queue(
        self, stats: Iterable[RuleStats]
    ) -> list[RetirementDecision]:
        return [d for d in self.evaluate(stats) if d.decision == Decision.RETIRE]
