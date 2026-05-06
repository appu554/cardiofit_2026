# Layer 2 → Layer 3 handoff

**Status:** the contract Layer 3 consumes from the Layer 2 substrate.
This is the canonical reference for Layer 3 implementers; deviation
from these contracts must be coordinated with the Layer 2 lead.

## Substrate read APIs Layer 3 will consume

Layer 3 (state machines: Authorisation, Recommendation, Monitoring,
ClinicalState, Consent) consumes the following substrate read paths.
Each entry lists the endpoint, the typical Layer 3 consumer, an
example payload, and the SLO.

### Resident snapshot

`GET /v2/residents/{id}` → `models.Resident`

Consumer: every state machine on every transition (the resident is the
top-level join key).

```json
{
  "id": "...",
  "facility_id": "...",
  "given_name": "Jane",
  "family_name": "Smith",
  "dob": "1948-03-15",
  "ihi": "8003600166666666",
  ...
}
```

SLO: p95 <30ms.

### Medicine use list

`GET /v2/residents/{id}/medicine_uses` → `[]models.MedicineUse`

Consumer: Recommendation (drug-burden inputs, intent_required predicate
gating).

Pay attention to `Intent.Category`: an empty or "unknown" value means
intent-required rules MUST suppress (Failure 3 defence).

SLO: p95 <60ms.

### Observations

`GET /v2/residents/{id}/observations/{kind}` → `[]models.Observation`

Consumer: Monitoring (delta detection, trajectory state), Recommendation
(evidence inputs).

Observations carry `flagged_baseline_delta` once the baseline recompute
pipeline has caught up. Layer 3 should not auto-fire on a delta that's
older than 30 seconds without confirming the recompute has landed.

SLO: p95 <100ms.

### Active concerns

`GET /v2/active_concerns?resident_ref=...&open=true` → `[]ActiveConcern`

Consumer: Recommendation (suppression on overlapping concerns),
ClinicalState (concern lifecycle).

SLO: p95 <60ms.

### Care intensity

`GET /v2/care_intensity/{resident_id}` → `CareIntensityState`

Consumer: Recommendation (deprescribing thresholds shift in
comfort_focused), Authorisation (some drug classes off-limits in
comfort_focused).

SLO: p95 <30ms.

### Capacity assessment

`GET /v2/capacity_assessment/{resident_id}` → latest `CapacityAssessment`

Consumer: Consent (capacity outcome change triggers re-eval).

SLO: p95 <30ms.

### Baselines

`GET /v2/baselines/{resident_id}/{observation_type}` → `Baseline`

Consumer: Monitoring (delta computation against the baseline value),
Recommendation (target band derivation).

Pay attention to `confidence_tier`: LOW confidence means Layer 3 should
treat the baseline as advisory only; do not fire automated state
transitions on a LOW-confidence baseline.

SLO: p95 <40ms.

### EvidenceTrace lineage / consequences / window

The Wave 5.2 query API:

- `GET /v2/evidence-trace/recommendations/{id}/lineage?depth=10`
- `GET /v2/evidence-trace/observations/{id}/consequences?depth=10`
- `GET /v2/residents/{id}/reasoning-window?from=...&to=...`

Consumer: any state machine that needs to "show its work" in a UI;
regulator-audit responder.

See `docs/runbooks/evidencetrace-audit-query.md` for full payload
shapes and `docs/slo/v2-substrate-slos.md` for the latency table.

### Identity match

`GET /v2/identity/match` (POST is also supported for confidential
identifiers) → `MatchResult`

Consumer: any inbound source (eNRMC, MHR, hospital discharge) before
binding to a Resident.

LOW / NONE confidence MUST queue for review (RequiresReview=true);
Layer 3 must NOT auto-bind. Failure 2 defence.

SLO: p95 <150ms.

## Substrate write APIs Layer 3 will use

### EvidenceTrace nodes + edges

`POST /v2/evidence-trace/nodes` and `POST /v2/evidence-trace/edges`.

Consumer: every state machine on every transition. Each transition MUST
write at least one EvidenceTrace node; if the transition derives from
upstream nodes, the corresponding `evidence_trace_edges` rows MUST also
be written.

### Active concerns

`POST /v2/active_concerns` opens a concern. Closing a concern is a
PATCH on the concern row (sets closed_at).

### Care intensity

`POST /v2/care_intensity/{resident_id}` writes a transition.

### Capacity assessment

`POST /v2/capacity_assessment` writes one assessment outcome.

## Outbox events Layer 3 will subscribe to

The kb-20 outbox (event_outbox table → Kafka) emits events for every
substrate write. Layer 3 subscribes selectively:

| Topic | Layer 3 consumer |
|-------|-----------------|
| `kb20.observation.upserted` | Monitoring (delta detection) |
| `kb20.medicine_use.upserted` | Recommendation, Authorisation |
| `kb20.event.upserted` | every state machine (events are cross-cutting) |
| `kb20.active_concern.opened` | Recommendation (suppression refresh) |
| `kb20.active_concern.closed` | Recommendation, Monitoring |
| `kb20.baseline.recomputed` | Monitoring (delta thresholds shift) |
| `kb20.capacity_assessment.upserted` | Consent (re-eval trigger) |
| `kb20.cfs_score.upserted` | Monitoring, ClinicalState (CFS≥7 hint) |

Each event payload carries the canonical resource ID; consumers fetch
the full row via the read API as needed.

## Invariants Layer 3 must respect

1. **EvidenceTrace is append-only.** Never delete a node or edge; mark
   it suppressed via the `suppressed` edge_kind if reasoning later
   determined it shouldn't have fired.
2. **Substrate writes ARE the audit trail.** Don't bypass the
   substrate to write to a downstream cache and skip the
   EvidenceTrace.
3. **LOW-confidence baselines / matches don't auto-fire.** Layer 3
   must check confidence before firing automated transitions.
4. **Outbox-driven recompute is eventually consistent.** A read
   immediately after a write may not reflect the recompute yet
   (within the 30s p95 lag window).
5. **State machine transitions write Provenance not AuditEvent.** Use
   `fhir.MapEvidenceTrace` (Wave 5.3 dispatcher) — don't pick the
   target FHIR resource manually.

## Sample Layer 3 consumer flow

A Recommendation state machine receiving a baseline-delta event:

1. Subscribe to `kb20.baseline.recomputed` on Kafka.
2. On event, fetch the affected baseline via `GET /v2/baselines/...`.
3. Fetch active concerns for the resident via `GET /v2/active_concerns`
   to apply suppression rules.
4. Fetch care intensity to apply deprescribing-aware thresholds.
5. Run rule engine; on rule fire, write:
   - One Recommendation `evidence_trace_nodes` row
     (state_change_type=`draft -> submitted`).
   - One `evidence_trace_edges` row (`derived_from`) per upstream
     observation node.
   - The Recommendation resource itself via the Recommendation
     service (kb-23 / Decision Cards).
6. Subscribe to clinical action on the Recommendation; when it's
   accepted/rejected, write the lifecycle transition as a new
   EvidenceTrace node with a `led_to` edge from the original.

## Open questions for Layer 3

- **Decision-keyword closed set.** The Wave 5.2 ReasoningWindow
  rollup uses a permissive substring match for "decision count"
  (`decided` / `accepted` / `rejected` / `approved` / `declined`).
  Layer 3 should publish a closed set of state_change_type tags that
  count as decisions.
- **System event subtypes.** The Wave 5.3 dispatcher routes the
  closed set {rule_fire, credential_check, query_recorded,
  login_propagated} to AuditEvent. Layer 3 needs to coordinate any
  additions.
- **Outbox event schema.** The current outbox emits canonical resource
  IDs and minimal metadata; Layer 3 may need richer payloads for
  some consumers (TBD).

## See also

- Layer 2 doc Part 4 (state machine integration).
- `docs/runbooks/sunday-night-fall-walkthrough.md` — worked example.
- `docs/slo/v2-substrate-slos.md` — the performance contract.
- `docs/security/v2-substrate-security-review.md` — auth + audit.
