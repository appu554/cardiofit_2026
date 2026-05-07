# Layer 3 Wave 0 — Trigger surface mapping (Layer 3 events → Layer 2 producers)

**Date:** 2026-05-06
**Plan:** [2026-05-04-layer3-rule-encoding-plan.md](../plans/2026-05-04-layer3-rule-encoding-plan.md) — Wave 0 Task 4
**Source spec:** [Layer3_v2_Rule_Encoding_Implementation_Guidelines (1).md](../../../Layer3_v2_Rule_Encoding_Implementation_Guidelines%20(1).md) — Part 0.5.5 (lines 189-204)
**Layer 2 plan:** [docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md](../plans/2026-05-04-layer2-substrate-plan.md)
**Layer 2 handoff:** [docs/handoff/layer-2-to-layer-3-handoff.md](../../handoff/layer-2-to-layer-3-handoff.md)
**Status:** Draft pending sign-off.

---

## Purpose

For every trigger event source the Layer 3 v2 doc Part 0.5.5 enumerates
(the v2 expanded trigger surface), identify the Layer 2 producer that
emits the event, plus an example payload and current delivery status.

A "Layer 2 producer" is the Layer 2 component (file path + function or
event name) that writes the event onto the substrate change-stream that
Layer 3 listens on. The change-stream itself is the EvidenceTrace
write side (Layer 2 plan Wave 1R.2) plus the Layer 2 outbox / Flink
streaming pipeline (Layer 2 plan Wave 2.5).

---

## Status legend

| Symbol | Meaning |
|---|---|
| delivered | Layer 2 producer is in `main` and emitting the event today |
| in-flight | Layer 2 producer is being implemented under the published Layer 2 plan |
| partial   | producer exists but with a known gap (e.g. payload missing a required field) |
| gap       | producer not planned; would block Layer 3 rule firing |

---

## Trigger surface table

| # | Trigger source name (Layer 3 v2 Part 0.5.5) | Layer 2 producer | Example event payload (abbrev.) | Status |
|---|---|---|---|---|
| 1 | **medication_change** — new prescription, dose change, cessation | `kb-20-patient-profile` MedicineUse handler — `internal/api/medicine_use_handlers.go::OnUpsert`; emits to EvidenceTrace `medicine_use_changed` node | `{type: "medicine_use_changed", resident_ref: "...", medicine_use_ref: "...", change_kind: "dose_change", from: "10mg", to: "20mg"}` | delivered |
| 2 | **condition_change** — new diagnosis or condition resolution | `kb-20-patient-profile` Condition handler — `internal/api/condition_handlers.go`; backed by Layer 2 plan Wave 1R.4 CSV ingestor + MHR ingest pipeline | `{type: "condition_changed", resident_ref: "...", condition_ref: "...", action: "added"}` | delivered |
| 3 | **observation_update** — lab, vital, weight | `shared/v2_substrate/delta/persistent_baseline_provider.go::Recompute`; observation insert is transactional with baseline-state upsert | `{type: "observation_inserted", resident_ref: "...", kind: "potassium", value: 5.1, observed_at: "..."}` | delivered |
| 4 | **baseline_delta** — sedation 4/7 vs baseline 0/7, eGFR drop >20% in 14d | `shared/v2_substrate/delta/trajectory_detector.go::DetectAndEmit`; emits when `flagged_baseline_delta` flips on Observation insert (Layer 2 plan Wave 2.1) | `{type: "baseline_delta", resident_ref: "...", kind: "potassium", baseline: 4.2, current: 5.1, delta: 0.9, window_days: 7}` | delivered |
| 5 | **active_concern_resolution** — "watching for delayed head injury 72h" expires | `shared/v2_substrate/clinical_state/concern_lifecycle.go::OnResolve`; lifecycle states = open / resolved_stop_criteria / escalated / expired_unresolved (Layer 2 plan §2.3) | `{type: "active_concern_resolved", resident_ref: "...", concern_ref: "...", resolution_status: "expired_unresolved"}` | delivered |
| 6 | **monitoring_threshold_crossed** — "K+ trending up" hits 5.5 | `shared/v2_substrate/monitoring/threshold_evaluator.go::OnObservationInsert`; references MonitoringPlan thresholds (Layer 2 doc §2.4) | `{type: "monitoring_threshold_crossed", resident_ref: "...", plan_ref: "...", kind: "potassium", threshold: 5.5, observed: 5.7}` | delivered |
| 7 | **consent_expiry_approaching** — antipsychotic consent expires in 14 days | `shared/v2_substrate/consent/expiry_scanner.go::Scan`; runs on a 1h tick + on Consent write; emits at the configured warning windows (default: 14d / 7d / 1d) | `{type: "consent_expiry_approaching", resident_ref: "...", consent_class: "Antipsychotic_deprescribe_review", expires_at: "...", days_remaining: 14}` | delivered |
| 8 | **authorisation_expiry_approaching** — ACOP credential expires in 30 days | `kb-30-authorisation-evaluator/internal/invalidation/expiry_scanner.go::Scan` (planned Wave 3 of Layer 3 plan; backed by Layer 2 Authorisation seam shipped in Phase 1B-β.2) | `{type: "authorisation_expiry_approaching", role_ref: "...", authorisation_class: "ACOP_S4", expires_at: "...", days_remaining: 30}` | in-flight |
| 9 | **care_intensity_transition** — active treatment → palliative | `shared/v2_substrate/clinical_state/care_intensity_transition.go::OnTransition`; CareIntensity write is transactional with the transition Event (Layer 2 plan §2.2) | `{type: "care_intensity_transition", resident_ref: "...", from: "active_treatment", to: "comfort_focused", at: "..."}` | delivered |
| 10 | **care_transition** — hospital discharge, RACF admission | `shared/v2_substrate/models/event.go` Event with `event_type IN ('hospital_discharge', 'racf_admission')`; emitted by Layer 2 plan Wave 4 discharge reconciliation | `{type: "event", event_type: "hospital_discharge", resident_ref: "...", from_facility: "...", to_facility: "...", at: "..."}` | in-flight |
| 11 | **observation_velocity_flag** (additional, surfaced by Tier 1 inventory rule 11 + 12) | `shared/v2_substrate/delta/velocity_flag.go::OnInsert`; emitted alongside #4 when `BaselineConfig.FlagVelocity = true` (Layer 2 plan §2.2 sub-task) | `{type: "observation_velocity_flag", resident_ref: "...", kind: "eGFR", velocity_per_day: -2.1, look_back_days: 14}` | delivered |
| 12 | **scope_rule_change** (additional, drives Event D in CompatibilityChecker) | `kb-31-scope-rules/internal/store/diff_emitter.go::OnApply` (planned Wave 4 of Layer 3 plan); listens to kb-3 source ingest | `{type: "scope_rule_changed", rule_id: "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01", action: "activated", effective_from: "2026-07-01"}` | in-flight |

---

## Gaps surfaced

**No producer gaps identified.**

Every trigger source from Layer 3 v2 doc Part 0.5.5 maps to a Layer 2
deliverable that has either shipped (`delivered`) or is in scope of
the published Layer 2 plan or the Layer 3 plan's own Wave 3-4
(`in-flight`).

The two `in-flight` rows are:

* **authorisation_expiry_approaching** (#8) — gated on Layer 3 Wave 3
  Authorisation evaluator deployment. Tier 1 rules that depend on it
  (none in the Wave 0 Task 1 inventory) would suppress until Wave 3
  ships.
* **care_transition** (#10) — gated on Layer 2 Wave 4 hospital
  discharge reconciliation. Tier 1 rules in scope today do not
  consume care-transition events; the Wave 5+ deprescribing-on-
  discharge rules will. No Tier 1 blocker.
* **scope_rule_change** (#12) — gated on Layer 3 Wave 4 ScopeRule
  engine. CompatibilityChecker's Event D path will be live when this
  ships; Wave 1 CompatibilityChecker can synthesise the event for
  testing.

If a Wave 1+ rule authoring session surfaces a new trigger source
that this table does not cover, the expected escalation path is:

1. Add a new row here under the appropriate wave anchor.
2. Open a Layer 2 backlog ticket if the producer does not exist.
3. Hold the affected rule until the producer is delivered.

---

## Cross-references

* Per-trigger CQL helper that reads the trigger payload: see Wave 0
  Task 2 spec (`MonitoringHelpers.ObservationOverdueBy` for #6,
  `ClinicalStateHelpers.DeltaFromBaseline` for #4, etc.).
* Per-rule trigger declarations: see the example specs at
  `shared/cql-libraries/examples/*.yaml`.
* Layer 2 producer file paths anchor against the Layer 2 plan §
  references; if a path moves under Layer 2 refactor, update this
  table under the same git commit so the contract stays in sync.

---

## Open questions

1. **Outbox vs direct subscription.** Layer 3 currently consumes via
   the EvidenceTrace write side (every substrate write fans an event
   into the trace). For high-fan-out events (#3 observation_update,
   #11 observation_velocity_flag), the Layer 2 plan Wave 2.5 Flink
   pipeline may be the more efficient subscription point. Decision
   deferred to Layer 3 Wave 1 when the rule-firing engine plumbing is
   designed.
2. **Per-event SLA.** Layer 2 handoff documents per-API SLOs (read-side
   latency); the trigger-event side does not yet have a published
   end-to-end SLA. Recommendation: target <2s p95 from substrate
   write to Layer 3 rule fire for #1-#6 (the hot-path safety triggers);
   <30s p95 for #7-#10 (lifecycle / scheduled triggers). Confirm with
   Layer 2 lead during sign-off.

These tracked as Wave 1 backlog.
