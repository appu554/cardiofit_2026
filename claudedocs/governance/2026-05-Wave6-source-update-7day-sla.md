# Wave 6 — Source-update 7-day SLA runbook + on-call rotation template

**Authoritative reference:** Layer 3 v2 Implementation Guidelines doc
Part 4.3 — source updates must propagate through the rule library
within 7 days of detection.

**Implementation:** `shared/cql-toolchain/source_update_tracker.py`
(in-memory tracker + breach detection). PagerDuty / Opsgenie wiring is
V2 work — this runbook is the human-readable companion.

## The four source classes

| Class | What it covers | Examples | SLA |
|---|---|---|---|
| `clinical_guideline` | RACGP / NHFA / KDIGO / RCPA / NSQHS revisions | RACGP T2DM 2024 → 2024-rev3 | 7 days |
| `regulatory_scope_rule` | kb-31 ScopeRule changes | Victorian DPCSA Amendment 2025 | 7 days + tighter against enforcement deadline |
| `substrate_schema` | Layer 2 substrate-change manifests | new fact `baseline_state.weight_kg` | 7 days |
| `source_authority_pin` | version-pin of a source authority used by Layer 1 Pipeline-2 | APC version pin bumped | 7 days |

## Tighter SLA: regulatory ScopeRule with enforcement deadline

The Victorian PCW exclusion ScopeRule
(`AUS-VIC-PCW-S4-EXCLUSION-2026-07-01`) commences 1 July 2026 with a
90-day grace period; hard enforcement begins 29 September 2026.

If a Vaidshala v2 deploys after 22 September 2026, the 7-day SLA window
overlaps the enforcement deadline → `enforcement_deadline_at_risk`
breach kind. The on-call rotation MUST treat such breaches as P1.

## SourceUpdateTracker breach kinds

```
in_flight_overdue              — detected, not yet propagated, > 7 days
propagated_late                — propagated, but took > 7 days
enforcement_deadline_at_risk   — regulatory rule + deadline within SLA window
```

## On-call rotation template

| Role | Responsibility | Coverage |
|---|---|---|
| Primary on-call (engineering lead) | Acknowledge breach within 30 min, drive remediation | 24x7 weekly rotation |
| Secondary on-call (engineering) | Backup; escalation path if primary unreachable | 24x7 weekly rotation |
| Clinical lead | Sign off on clinical-guideline class breaches | Business hours; defer to next-day for in_flight_overdue |
| Regulatory lead | Sign off on regulatory_scope_rule breaches | Business hours; P1 escalates immediately for enforcement_deadline_at_risk |

### Suggested rotation (V2: Vaidshala / CardioFit pilot)

```yaml
rotation:
  cadence: weekly
  hours: 24x7
  members:
    - role: primary
      members: [eng_lead_a, eng_lead_b, eng_lead_c]
    - role: secondary
      members: [eng_b, eng_c, eng_a]
  business_hours_overlay:
    - role: clinical_lead
      members: [clin_a, clin_b]
    - role: regulatory_lead
      members: [reg_a]

handoff:
  day: monday
  time: "09:00 Australia/Melbourne"
  artefacts:
    - "open breach list from SourceUpdateTracker"
    - "in-flight ScopeRule changes from kb-31"
    - "weekly override-rate trend from kb-30 analytics"
```

## Breach response runbook

### in_flight_overdue (P3 default; P1 for regulatory)

1. Check kb-30 CompatibilityChecker status for the rules touched by
   the source update. Any rule still STALE > 7 days after detection is
   a process miss.
2. Identify the blocker:
   - Toolchain failure → fix and re-run.
   - Clinical sign-off pending → escalate to clinical lead.
   - Regulatory sign-off pending → escalate to regulatory lead.
3. Re-promote via GovernancePromoter; record outcome.

### propagated_late (P3)

1. Retrospective only. Capture root cause (process delay, toolchain
   slowness, sign-off delay).
2. File improvement task. Don't roll back unless the late propagation
   shipped a defect.

### enforcement_deadline_at_risk (P1)

1. **Immediately** notify regulatory lead + clinical lead.
2. Confirm the ScopeRule is in kb-31 with `status: ACTIVE` and the
   correct effective_period.
3. Confirm CompatibilityChecker has marked dependent Layer 3 rules
   STALE and re-promotion is in flight.
4. If propagation cannot complete before the deadline, escalate to the
   service owner; deploy a feature flag that defaults the affected
   action class to the most-restrictive scope until propagation
   completes (fail-closed posture).

## Reporting

Weekly digest emailed to clinical lead + regulatory lead on Mondays:

```
Source updates detected this week:           N
In-flight at start of week:                  N
Propagated this week:                        N
Breaches in week (by kind):
  - in_flight_overdue:                       N
  - propagated_late:                         N
  - enforcement_deadline_at_risk:            N
Library-wide override rate trend (30d):      X.X% (target < 5%)
Notable changes:                             ...
```

The digest is the input to the Wednesday Class 5/6 tuning session
described in `claudedocs/clinical/2026-05-Wave6-suppression-tuning-runbook.md`.

## Audit chain

Every breach is captured in:
- The kb-30 audit store (audit query Q3 by jurisdiction).
- The kb-31 ScopeRule lineage (`Lineage(rule_id)`).
- The GovernancePromoter signed package (which carries the
  `content_sha` of every promoted rule version).

## Continuous improvement

When the SLA budget breaches twice in a quarter:
1. Review on-call rotation — is coverage actually 24x7?
2. Review toolchain throughput — is Stage 1 / two-gate / governance
   signing the bottleneck?
3. Review source-detection latency — are we noticing source updates
   inside their 7-day window in the first place?

The SLA is meaningful only if detection latency is well under the SLA
window. As a rule of thumb: detection should be under 1 day; the
remaining 6 days are for propagation + sign-off.
