"""Wave 6 Task 3 — pytest coverage for source_update_tracker."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

import pytest

from source_update_tracker import (
    DEFAULT_SLA_DAYS,
    SLABreach,
    SourceClass,
    SourceUpdate,
    SourceUpdateTracker,
)


UTC = timezone.utc


def test_in_flight_within_sla_no_breach():
    t = SourceUpdateTracker()
    detected = datetime(2026, 5, 1, 0, 0, 0, tzinfo=UTC)
    t.record_detected(
        SourceUpdate(
            update_id="u1",
            source_class=SourceClass.CLINICAL_GUIDELINE,
            source_id="racgp-t2dm-2024-rev3",
            detected_at=detected,
        )
    )
    now = detected + timedelta(days=3)
    assert t.breaches(now) == []


def test_in_flight_overdue_breach():
    t = SourceUpdateTracker()
    detected = datetime(2026, 5, 1, 0, 0, 0, tzinfo=UTC)
    t.record_detected(
        SourceUpdate(
            update_id="u2",
            source_class=SourceClass.CLINICAL_GUIDELINE,
            source_id="racgp-bpsd-2024-rev1",
            detected_at=detected,
        )
    )
    now = detected + timedelta(days=10)
    breaches = t.breaches(now)
    assert len(breaches) == 1
    assert breaches[0].breach_kind == "in_flight_overdue"


def test_propagated_late_breach():
    t = SourceUpdateTracker()
    detected = datetime(2026, 5, 1, 0, 0, 0, tzinfo=UTC)
    t.record_detected(
        SourceUpdate(
            update_id="u3",
            source_class=SourceClass.SUBSTRATE_SCHEMA,
            source_id="kb20-baseline-egfr",
            detected_at=detected,
        )
    )
    t.mark_propagated("u3", detected + timedelta(days=10))
    now = detected + timedelta(days=15)
    breaches = t.breaches(now)
    assert len(breaches) == 1
    assert breaches[0].breach_kind == "propagated_late"


def test_propagated_within_sla_no_breach():
    t = SourceUpdateTracker()
    detected = datetime(2026, 5, 1, 0, 0, 0, tzinfo=UTC)
    t.record_detected(
        SourceUpdate(
            update_id="u4",
            source_class=SourceClass.SOURCE_AUTHORITY_PIN,
            source_id="apc-version-pin",
            detected_at=detected,
        )
    )
    t.mark_propagated("u4", detected + timedelta(days=4))
    now = detected + timedelta(days=15)
    assert t.breaches(now) == []


def test_regulatory_enforcement_deadline_at_risk():
    """The Victorian PCW exclusion: regulatory ScopeRule whose enforcement
    deadline is 29 Sep 2026. If detected only on 25 Sep 2026 and not yet
    propagated, the 7-day SLA window breaches the enforcement
    deadline -> at risk."""
    t = SourceUpdateTracker()
    detected = datetime(2026, 9, 25, 0, 0, 0, tzinfo=UTC)
    enforcement = datetime(2026, 9, 29, 0, 0, 0, tzinfo=UTC)
    t.record_detected(
        SourceUpdate(
            update_id="u-vic",
            source_class=SourceClass.REGULATORY_SCOPE_RULE,
            source_id="vic-dpcsa-amend-2025",
            detected_at=detected,
            enforcement_deadline=enforcement,
        )
    )
    now = detected + timedelta(hours=1)
    breaches = t.breaches(now)
    kinds = {b.breach_kind for b in breaches}
    assert "enforcement_deadline_at_risk" in kinds, (
        f"expected enforcement_deadline_at_risk; got {kinds}"
    )


def test_default_sla_is_seven_days_for_all_classes():
    for cls in SourceClass:
        assert DEFAULT_SLA_DAYS[cls] == 7


def test_record_detected_validates_tz():
    t = SourceUpdateTracker()
    with pytest.raises(ValueError):
        t.record_detected(
            SourceUpdate(
                update_id="u",
                source_class=SourceClass.CLINICAL_GUIDELINE,
                source_id="x",
                detected_at=datetime(2026, 5, 1, 0, 0, 0),  # naive
            )
        )


def test_mark_propagated_unknown_update_raises():
    t = SourceUpdateTracker()
    with pytest.raises(KeyError):
        t.mark_propagated("missing", datetime.now(UTC))


def test_in_flight_count():
    t = SourceUpdateTracker()
    base = datetime(2026, 5, 1, 0, 0, 0, tzinfo=UTC)
    for i in range(3):
        t.record_detected(
            SourceUpdate(
                update_id=f"u{i}",
                source_class=SourceClass.CLINICAL_GUIDELINE,
                source_id=f"src-{i}",
                detected_at=base,
            )
        )
    t.mark_propagated("u0", base + timedelta(days=1))
    assert t.in_flight_count() == 2
