# EvidenceTrace audit query patterns

**Audience:** regulator-audit responder, clinical informatics
partner, on-call backend engineer.
**Source contract:** Layer 2 doc §1.6 (EvidenceTrace) + Part 7
Recommendation 3 ("EvidenceTrace as queryable graph from day 1").

## What EvidenceTrace is

The clinical-reasoning audit graph: every state-machine transition in
the substrate writes one `evidence_trace_nodes` row, with directed
`evidence_trace_edges` linking it to its inputs (`derived_from` /
`evidence_for`) and outputs (`led_to`). The graph is **bidirectional
from day 1** — a regulator query can walk forward (what did this
observation cause?) or backward (what evidence produced this
recommendation?).

## Three canonical audit queries

### Query 1: "What evidence produced this recommendation?"

`GET /v2/evidence-trace/recommendations/{id}/lineage?depth=10`

Returns a JSON envelope shaped:

```json
{
  "target_node_id": "...",
  "nodes": [
    {"id": "...", "state_machine": "Monitoring",
     "state_change_type": "observation_recorded",
     "recorded_at": "...", "occurred_at": "..."}
  ],
  "max_depth": 3
}
```

Walks `derived_from` and `evidence_for` edges backward. Use this
when the regulator asks "show me what reasoning led to this drug
adjustment".

### Query 2: "What did this observation cause?"

`GET /v2/evidence-trace/observations/{id}/consequences?depth=10`

Symmetric forward walk via `led_to` edges. Use this when investigating
"this BP reading flagged a delta — show me what happened next".

### Query 3: "Per-resident audit window"

`GET /v2/residents/{id}/reasoning-window?from=...&to=...`

Returns a regulator-audit-ready rollup over the time window:

```json
{
  "resident_ref": "...",
  "from": "2026-04-01T00:00:00Z",
  "to":   "2026-05-01T00:00:00Z",
  "total_nodes": 87,
  "nodes_by_state_machine": {
    "Monitoring": 60, "Recommendation": 12,
    "ClinicalState": 8, "Authorisation": 5, "Consent": 2
  },
  "recommendation_count": 12,
  "decision_count": 9,
  "average_evidence_per_recommendation": 4.2,
  "nodes": [...]
}
```

Suitable for direct ACQSC submission. The `nodes` array is ordered
chronologically.

## When to use which query

| Regulator question | Query |
|--------------------|-------|
| "Why did the system recommend this drug?" | Query 1 (lineage) |
| "Show me everything that flowed from the fall on Sunday" | Query 2 (consequences) |
| "Provide the audit window for this resident for last quarter" | Query 3 (reasoning-window) |
| "Show me one specific transition as FHIR" | `GET /v2/evidence-trace/{id}/fhir` (returns Provenance OR AuditEvent per the Wave 5.3 dispatcher) |

## Materialised views

For high-volume audit responses, three materialised views ship with
migration 022 and are refreshed via `refresh_evidence_trace_views()`:

- `mv_recommendation_lineage` — pre-rolled per-Recommendation upstream
  evidence and downstream outcomes.
- `mv_observation_consequences` — pre-rolled per-Observation downstream
  Recommendation count and acted count.
- `mv_resident_reasoning_summary` — 30-day rolling summary per resident.

Refresh cadence is configured by the operator. Default V0:
NOTIFY-driven worker triggers a CONCURRENT refresh on every node /
edge change. V1 will lock in cadence (incremental via outbox vs
nightly full).

## FHIR egress

Every `evidence_trace_nodes` row maps to **exactly one** FHIR resource
per the Layer 2 doc §1.6 dual-resource pattern:

- Provenance for state changes on Recommendation / Monitoring /
  ClinicalState / Authorisation / Consent (where the change is a
  resource transition).
- AuditEvent for system events tagged with state_change_type in
  {rule_fire, credential_check, query_recorded, login_propagated}.

Use `fhir.MapEvidenceTrace(node)` programmatically or
`GET /v2/evidence-trace/{id}/fhir` over the wire. The X-FHIR-Resource-Type
response header indicates the dispatch outcome.

## Performance SLOs

Per Wave 5.4 plan task (V1 verification target):

| Operation | p95 target |
|-----------|-----------|
| Forward / backward depth=5 traversal | <200ms |
| Lineage / consequences via materialised view | <100ms |
| ReasoningWindow over 30-day window | <150ms |

See `docs/slo/v2-substrate-slos.md` for the full SLO table.

## Worked example

The Sunday-night-fall pilot scenario is the canonical worked example:
see [sunday-night-fall-walkthrough.md](sunday-night-fall-walkthrough.md).
The Saturday checkpoint runs all three audit queries against the
substrate as it stood at end of week.

## Common pitfalls

### Pitfall: depth too shallow

A regulator query at `depth=2` will miss long causal chains.
Default to `depth=10` for full lineage; only use shallow depths for
performance-bounded operational dashboards.

### Pitfall: the node was deleted

`evidence_trace_nodes` rows are append-only by design — they are
NEVER deleted. If a query returns a stale ID that doesn't resolve,
investigate as a substrate bug (likely a foreign-key cascade left
the edge but removed the node).

### Pitfall: confusing inputs vs outputs vs edges

- A node's `inputs` JSONB array carries opaque references (Observation
  ID, MedicineUse ID) — these may NOT all have corresponding
  `evidence_trace_nodes` rows.
- An edge's endpoints are always EvidenceTrace node IDs.
- If you need to walk to a non-EvidenceTrace input (e.g. the
  underlying observations row), follow the `inputs[*].input_ref` to
  the canonical store directly.

## See also

- Layer 2 doc §1.6.
- [sunday-night-fall-walkthrough.md](sunday-night-fall-walkthrough.md).
- [identity-match-queue-triage.md](identity-match-queue-triage.md).
- `docs/slo/v2-substrate-slos.md`.
