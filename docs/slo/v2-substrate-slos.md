# Layer 2 substrate — performance SLOs

**Status:** target SLOs for V0 production deployment. V1 will lock in
verified numbers after the Wave 5.4 production load test runs against
the kb-20 PostgreSQL deployment.

This document defines the performance contract every Layer 2 read API
publishes to its consumers (Layer 3 state machines, regulator-audit
tooling, internal dashboards).

## Read-path SLOs

| Endpoint | p50 | p95 | p99 | Notes |
|----------|-----|-----|-----|-------|
| `GET /v2/residents/{id}` | 5ms | 30ms | 80ms | Direct PK lookup |
| `GET /v2/residents/{id}/medicine_uses` | 15ms | 60ms | 150ms | Indexed list |
| `GET /v2/residents/{id}/observations` | 25ms | 100ms | 250ms | Indexed list, paginated |
| `GET /v2/residents/{id}/observations/{kind}` | 15ms | 60ms | 150ms | Type-filtered |
| `GET /v2/observations/{id}` | 5ms | 30ms | 80ms | Direct PK |
| `GET /v2/medicine_uses/{id}` | 5ms | 30ms | 80ms | Direct PK |
| `GET /v2/events/{id}` | 5ms | 30ms | 80ms | Direct PK |
| `GET /v2/residents/{id}/events` | 25ms | 100ms | 250ms | Indexed list |
| `GET /v2/active_concerns/...` | 15ms | 60ms | 150ms | Per-resident list |
| `GET /v2/scoring/cfs/{id}` | 5ms | 30ms | 80ms | Direct PK |
| `GET /v2/care_intensity/{resident_id}` | 5ms | 30ms | 80ms | Direct PK |
| `GET /v2/baselines/{resident_id}/{type}` | 10ms | 40ms | 100ms | Indexed lookup |
| `GET /v2/identity/match` | 30ms | 150ms | 400ms | Includes fuzzy match |
| `GET /v2/identity/review-queue` | 20ms | 80ms | 200ms | Paginated list |

## EvidenceTrace SLOs (Wave 5)

| Endpoint | p50 | p95 | p99 | Notes |
|----------|-----|-----|-----|-------|
| `GET /v2/evidence-trace/nodes/{id}` | 5ms | 30ms | 80ms | Direct PK |
| `GET /v2/evidence-trace/{id}/forward?depth=5` | 50ms | 200ms | 500ms | Recursive CTE |
| `GET /v2/evidence-trace/{id}/backward?depth=5` | 50ms | 200ms | 500ms | Recursive CTE |
| `GET /v2/evidence-trace/recommendations/{id}/lineage` | 30ms | 100ms | 250ms | Materialised view |
| `GET /v2/evidence-trace/observations/{id}/consequences` | 30ms | 100ms | 250ms | Materialised view |
| `GET /v2/residents/{id}/reasoning-window` | 50ms | 150ms | 400ms | Per-resident scan |
| `GET /v2/evidence-trace/{id}/fhir` | 10ms | 50ms | 120ms | Direct PK + map |

Materialised view refresh:

| Operation | Target |
|-----------|--------|
| Incremental refresh after node insert | <30s p95 |
| Full nightly refresh | <10min |

## Write-path SLOs

| Endpoint | p50 | p95 | p99 | Notes |
|----------|-----|-----|-----|-------|
| `POST /v2/observations` | 10ms | 50ms | 120ms | + outbox emission |
| `POST /v2/medicine_uses` | 15ms | 60ms | 150ms | + intent validator |
| `POST /v2/events` | 10ms | 50ms | 120ms | |
| `POST /v2/evidence-trace/nodes` | 10ms | 50ms | 120ms | |
| `POST /v2/evidence-trace/edges` | 5ms | 30ms | 80ms | Idempotent on PK |
| `POST /v2/active_concerns` | 15ms | 60ms | 150ms | Triggers baseline recompute outbox |

## End-to-end pipeline SLOs

| Pipeline | Target |
|----------|--------|
| Observation insert → BaselineStore recompute persisted | p95 <30s (Failure 1 defence) |
| CFS≥7 score insert → worklist hint written | p95 <60s (Failure 5 defence) |
| MHR pathology poll → substrate row created | p95 <90s (per-poll) |
| eNRMC CSV import → substrate row created | p95 <2min/row |

## Availability SLOs

| Service | Availability target |
|---------|--------------------|
| kb-20 read API | 99.9% (4.4min/month) |
| kb-20 write API | 99.5% (3.7hr/month) |
| MHR poll | 99% (excluding upstream MHR outages) |

## Capacity targets

The substrate must support sustained:

- 2,000 observations / day / facility
- 5 facilities concurrent (10,000 observations / day cluster)
- 200 residents per facility
- 6-month retention with all queries hitting their SLO
- 1M-node EvidenceTrace graph (covers 6 months at the above rate)

The Wave 5.4 graph load-test harness verifies the EvidenceTrace
performance targets at this capacity. Production verification of every
SLO above is a V1 readiness gate.

## How SLO violations are tracked

- **Real-time:** Prometheus alerts trigger on any p95 violation
  sustained for >5 minutes.
- **Daily:** automated SLO report compares yesterday's actual p95 to
  the target. Drift >20% files a ticket against the responsible
  service team.
- **Quarterly:** SLO review with clinical informatics partners;
  targets re-tuned based on observed clinical impact.

## What "p95" means here

For each endpoint, "p95 <X" means:

> Of all production requests in a rolling 5-minute window, the 95th
> percentile latency must be below X milliseconds, measured from
> Gin handler entry to response body flush.

Network and client-side latency are out of scope.

## See also

- Layer 2 doc Part 6 Failure 1 / Failure 6.
- `docs/handoff/layer-2-to-layer-3-handoff.md` — the contract Layer 3
  consumes against these SLOs.
- `docs/runbooks/evidencetrace-audit-query.md` — how the EvidenceTrace
  SLOs map to specific audit queries.
