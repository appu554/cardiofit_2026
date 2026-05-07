"""Source-update 7-day SLA tracker — Wave 6 Task 3.

Per Layer 3 v2 doc Part 4.3, four classes of source updates must
propagate through the rule library within 7 days:

  1. clinical_guideline       — RACGP / NHFA / KDIGO / etc.
  2. regulatory_scope_rule    — kb-31 ScopeRule changes
  3. substrate_schema         — Layer 2 substrate change manifest
  4. source_authority_pin     — version-pin update for a source authority

The Victorian PCW exclusion ScopeRule has a tighter 0-day SLA against
its enforcement deadline (29 September 2026): the rule library must
already have surfaced and validated the ScopeRule before that date,
otherwise the 7-day clock collapses to a hard miss.

This tracker is intentionally an in-memory orchestrator producing
breach lists for the on-call rotation. Wiring to PagerDuty / Opsgenie /
the kb-30 audit store is V2 work.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from enum import Enum
from typing import Iterable


class SourceClass(str, Enum):
    CLINICAL_GUIDELINE = "clinical_guideline"
    REGULATORY_SCOPE_RULE = "regulatory_scope_rule"
    SUBSTRATE_SCHEMA = "substrate_schema"
    SOURCE_AUTHORITY_PIN = "source_authority_pin"


# Default SLA windows (days). Regulatory ScopeRules with a hard
# enforcement deadline (e.g. Victorian PCW 29 Sep 2026) have an
# additional zero-tolerance check against the deadline.
DEFAULT_SLA_DAYS: dict[SourceClass, int] = {
    SourceClass.CLINICAL_GUIDELINE: 7,
    SourceClass.REGULATORY_SCOPE_RULE: 7,
    SourceClass.SUBSTRATE_SCHEMA: 7,
    SourceClass.SOURCE_AUTHORITY_PIN: 7,
}


@dataclass
class SourceUpdate:
    """A single source update in flight or completed."""

    update_id: str
    source_class: SourceClass
    source_id: str            # e.g. "vic-dpcsa-amend-2025"
    detected_at: datetime
    propagated_at: datetime | None = None  # None = still in flight
    enforcement_deadline: datetime | None = None  # for regulatory only


@dataclass
class SLABreach:
    update_id: str
    source_id: str
    source_class: SourceClass
    detected_at: datetime
    sla_days: int
    breach_kind: str   # "in_flight_overdue" | "propagated_late" | "enforcement_deadline_at_risk"
    detail: str = ""


@dataclass
class SourceUpdateTracker:
    sla_days: dict[SourceClass, int] = field(default_factory=lambda: dict(DEFAULT_SLA_DAYS))
    updates: list[SourceUpdate] = field(default_factory=list)

    def record_detected(self, update: SourceUpdate) -> None:
        if not update.update_id:
            raise ValueError("update_id is required")
        if update.detected_at.tzinfo is None:
            raise ValueError("detected_at must be timezone-aware")
        self.updates.append(update)

    def mark_propagated(self, update_id: str, propagated_at: datetime) -> None:
        for u in self.updates:
            if u.update_id == update_id:
                u.propagated_at = propagated_at
                return
        raise KeyError(f"update {update_id!r} not found")

    def breaches(self, now: datetime) -> list[SLABreach]:
        """Return SLA breaches as of `now`. Three breach kinds:

          - in_flight_overdue: update detected, not yet propagated, and
            now > detected_at + sla_days.
          - propagated_late: update propagated, but propagation took
            longer than sla_days.
          - enforcement_deadline_at_risk: regulatory ScopeRule with an
            enforcement_deadline within 7 days and not yet propagated.
        """
        out: list[SLABreach] = []
        for u in self.updates:
            sla = self.sla_days.get(u.source_class, 7)
            sla_window = timedelta(days=sla)
            if u.propagated_at is None:
                if now - u.detected_at > sla_window:
                    out.append(
                        SLABreach(
                            update_id=u.update_id,
                            source_id=u.source_id,
                            source_class=u.source_class,
                            detected_at=u.detected_at,
                            sla_days=sla,
                            breach_kind="in_flight_overdue",
                            detail=(
                                f"detected {u.detected_at.isoformat()}, "
                                f"still in flight after {sla} days"
                            ),
                        )
                    )
                if (
                    u.source_class == SourceClass.REGULATORY_SCOPE_RULE
                    and u.enforcement_deadline is not None
                    and (u.enforcement_deadline - now) <= sla_window
                ):
                    out.append(
                        SLABreach(
                            update_id=u.update_id,
                            source_id=u.source_id,
                            source_class=u.source_class,
                            detected_at=u.detected_at,
                            sla_days=sla,
                            breach_kind="enforcement_deadline_at_risk",
                            detail=(
                                f"enforcement deadline "
                                f"{u.enforcement_deadline.isoformat()} "
                                f"within {sla}-day SLA window and "
                                f"propagation incomplete"
                            ),
                        )
                    )
            else:
                if u.propagated_at - u.detected_at > sla_window:
                    out.append(
                        SLABreach(
                            update_id=u.update_id,
                            source_id=u.source_id,
                            source_class=u.source_class,
                            detected_at=u.detected_at,
                            sla_days=sla,
                            breach_kind="propagated_late",
                            detail=(
                                f"propagation took "
                                f"{(u.propagated_at - u.detected_at).days} "
                                f"days (SLA: {sla})"
                            ),
                        )
                    )
        return out

    def in_flight_count(self) -> int:
        return sum(1 for u in self.updates if u.propagated_at is None)

    def by_source_class(
        self, source_class: SourceClass
    ) -> Iterable[SourceUpdate]:
        return (u for u in self.updates if u.source_class == source_class)
