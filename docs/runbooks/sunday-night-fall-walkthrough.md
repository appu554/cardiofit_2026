# Sunday-night-fall walkthrough — Layer 2 substrate pilot rehearsal

**Status:** rehearsed end-to-end against in-memory mocks (Wave 6.3).
Production rehearsal against the kb-20 PostgreSQL deployment is on the
V1 readiness checklist.

This runbook documents the Sunday-night-fall scenario — the closing
walkthrough from Layer 2 doc Part 8 — and explains what every substrate
write should look like at each checkpoint. Clinical informatics partners
can use it to cross-check actual production data against the substrate
contract.

## Cast

- **Resident:** Mrs M, 84, lives in a residential aged-care facility.
  Background: type 2 diabetes (metformin), atrial fibrillation
  (apixaban), mild cognitive impairment, gait unsteadiness.
- **Substrate writers:** kb-20 (Patient Profile / canonical
  EvidenceTrace store), kb-22 (HPI Engine), kb-23 (Decision Cards), kb-19
  (Protocol Orchestrator), V-MCU (titration / closed-loop).
- **Clinical actors:** RN on duty (Sunday PM), GP on call (Monday AM),
  hospital ED registrar (Monday PM), ACOP pharmacist (Wednesday),
  pathology lab (Thursday), care manager + family (Friday).

## Day-by-day expected substrate state

### Sunday 19:00 — fall recorded

The RN witnesses Mrs M slipping from her chair and records a fall Event
through the nursing UI.

**Substrate writes:**

- One `events` row (kind=`fall_event`, severity=`moderate`, witness
  recorded).
- One `evidence_trace_nodes` row (state_machine=`Monitoring`,
  state_change_type=`fall_event_recorded`, actor_role=RN). The node's
  `inputs` array carries a single Observation reference (the witnessed
  fall).
- One outbox emission for downstream consumers (kb-19, kb-22).

**Verification queries:**

- `SELECT * FROM evidence_trace_nodes WHERE resident_ref=$id AND
  state_change_type='fall_event_recorded'` → 1 row.
- The node's `recorded_at` and `occurred_at` should match (the fall is
  witnessed live).

### Monday 08:00 — post-fall vitals + agitation episode

Day-shift RN takes vitals during morning rounds. BP 165/92 (delta-flagged
versus the 90-day baseline of 132/78). Behavioural chart logs an
agitation episode at 09:00.

**Substrate writes:**

- Two `observations` rows (BP + agitation). The BP row's
  `flagged_baseline_delta=true` once the BaselineStore recompute lands
  via the outbox.
- Two `evidence_trace_nodes` rows
  (state_change_type=`baseline_delta_flagged` for BP,
  `agitation_episode_recorded` for the behavioural entry).
- Two `evidence_trace_edges` rows linking the Sunday fall node →
  these two nodes (`led_to`).

**Verification queries:**

- `mv_observation_consequences` for the Sunday fall node should show
  these two downstream Monitoring transitions once the materialised
  view refreshes.

### Monday 16:00 — hospital admission for head CT

GP on call orders a non-contrast head CT to rule out subdural. ED
registers Mrs M as a hospital_admission.

**Substrate writes:**

- One `events` row (kind=`hospital_admission`).
- One `evidence_trace_nodes` row
  (state_change_type=`hospital_admission_recorded`).
- Two edges into the new node from (a) the fall (`led_to`) and (b) the
  delta-flagged BP (`evidence_for`).

### Tuesday 11:00 — hospital discharge with new anti-emetic

Head CT clear; Mrs M discharged with a new medication (ondansetron PRN
for nausea).

**Substrate writes:**

- One `events` row (kind=`hospital_discharge`).
- One `evidence_trace_nodes` row
  (state_change_type=`hospital_discharge_recorded`).
- One `medicine_uses` row for the new anti-emetic (Intent.Category set
  per the discharge summary; if blank, see Failure Mode 3 — rules with
  `intent_required` will suppress).
- One edge linking hospital_admission → hospital_discharge (`led_to`).

### Wednesday — ACOP pharmacist reconciliation

Pharmacist reviews the discharge medication list against the resident's
existing list, completes reconciliation through the kb-20 reconciliation
endpoint.

**Substrate writes:**

- One `discharge_reconciliations` row.
- One `evidence_trace_nodes` row
  (state_machine=`Recommendation`,
  state_change_type=`reconciliation_completed`). Inputs include the
  discharge node and the active medicine_uses snapshot.
- Edges: discharge → reconciliation (`led_to` and `derived_from`).

### Thursday — MHR pathology result, AKI active concern, baseline recompute

A My Health Record SOAP poll surfaces a pathology result from a private
pathology lab: serum creatinine 138 (baseline 78 → 78% increase). The
substrate raises a mild AKI active concern (`AKI_watching`) and re-runs
the potassium baseline recompute with the AKI window excluded
(BaselineConfig.ExcludeDuringActiveConcerns wired to drop observations
inside the active window — the Failure 4 defence).

**Substrate writes:**

- One `observations` row (creatinine).
- One `active_concerns` row (kind=`AKI_watching`,
  open=true).
- One `evidence_trace_nodes` row
  (state_machine=`Monitoring`,
  state_change_type=`pathology_result_received`).
- One `evidence_trace_nodes` row
  (state_machine=`ClinicalState`,
  state_change_type=`active_concern_opened_AKI_watching`).
- One `evidence_trace_nodes` row
  (state_change_type=`baseline_recomputed_excluding_aki_window`).
- Edges: pathology → AKI concern → baseline recompute (`led_to`).

**Verification:** the baseline recompute's resulting baseline value
should NOT include the high-creatinine reading; if it does, Failure 4
defence has regressed.

### Friday — care intensity transition

Care manager + family meeting. CFS reassessed at 7 ("severe frailty");
combined with the fall, AKI, and behavioural deterioration, family agree
to transition to a comfort-focused care intensity.

**Substrate writes:**

- One `cfs_scores` row (score=7).
- One `evidence_trace_nodes` row
  (state_machine=`ClinicalState`,
  state_change_type=`care_intensity_active_treatment_to_comfort_focused`).
- Edges into this node from (a) the AKI concern (`led_to`), (b) the
  reconciliation (`evidence_for`), and (c) the original Sunday fall
  (`evidence_for`).

A worklist hint for the care intensity review should already have been
written when the CFS=7 capture landed (Failure 5 defence).

### Saturday — full traversal demonstrates lineage

Operations runs a full forward + backward EvidenceTrace traversal:

- `GET /v2/evidence-trace/{fall_event_id}/forward?depth=10` — must
  surface every downstream node from Monday through Friday.
- `GET /v2/evidence-trace/{care_transition_id}/backward?depth=10` —
  must surface every upstream node back to Sunday's fall.
- `GET /v2/residents/{id}/reasoning-window?from=Sun&to=Sat+1d` — must
  return the per-resident audit summary suitable for ACQSC submission.

The end-to-end test
`kb-20-patient-profile/tests/pilot_scenarios/sunday_night_fall_test.go`
asserts each of these checkpoints in-process; production rehearsal
against the live kb-20 deployment is the V1 readiness gate.

## Failure-mode coverage demonstrated by this scenario

| Failure mode | Demonstrated where |
|--------------|-------------------|
| F1 — compute-on-write perf | Monday's vitals delta-flag must complete in <30s p95 |
| F2 — identity match | If the MHR pathology arrives with a typo'd IHI, Thursday's pathology row queues for review (LOW confidence) rather than auto-routing to the wrong resident |
| F3 — intent sparseness | Tuesday's ondansetron MedicineUse arrives without an Intent.Category — any rule with `intent_required` correctly suppresses |
| F4 — baseline contamination | Thursday's baseline recompute excludes the AKI window |
| F5 — care intensity lag | Friday's CFS=7 capture writes the worklist hint within 60s |
| F6 — graph perf | Saturday's full-week traversal completes in <200ms p95 at depth=5 |

## Regulator-audit submission

The Saturday `reasoning-window` query returns the per-resident audit
summary structured as JSON suitable for ACQSC submission:

```json
{
  "resident_ref": "...",
  "from": "2026-05-03T18:00:00Z",
  "to":   "2026-05-10T00:00:00Z",
  "total_nodes": 10,
  "nodes_by_state_machine": {
    "Monitoring": 7,
    "Recommendation": 1,
    "ClinicalState": 2
  },
  "recommendation_count": 1,
  "decision_count": 0,
  "average_evidence_per_recommendation": 1.0,
  "nodes": [...]
}
```

This is the ACQSC-ready audit envelope — it is referenced by the
[evidencetrace-audit-query runbook](evidencetrace-audit-query.md).

## When to use this runbook

- Pilot dry-runs with the clinical informatics partner.
- Regression testing after kb-20 schema migrations.
- Onboarding new on-call engineers — read this scenario before reading
  the substrate code.
- Regulator preparation — this is the canonical worked example.
