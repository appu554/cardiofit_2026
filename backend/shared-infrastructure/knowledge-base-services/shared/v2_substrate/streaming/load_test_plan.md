# Layer 2 Streaming Pipeline Load Test Plan

Status: **Plan only** — execution scheduled for V1 hardening (post Wave 2.7).

## Goal

Validate that the substrate streaming pipeline meets Layer 2 doc §3.4 throughput target:

- 1,000-2,000 observations/day per facility
- 10 facilities concurrently → ~20,000 observations/day cohort
- 200 residents per facility
- **End-to-end p95 latency < 5 seconds** (raw_inbound_events → substrate_updates)
- **baseline_state recompute lag p95 < 30 seconds** (substrate_writer → clinical_state_updater)

## Test scenarios

### S1: Steady-state — 2,000 obs/day per facility, 10 facilities, 200 residents/facility

- Generator: synthesise observations at 0.023 events/sec/facility
  (2,000 events / 24h / 3,600s)
- Aggregate rate: ~0.23 events/sec across 10 facilities
- Per-resident rate: ~10 events/day/resident (200 residents × 10 = 2,000/facility)
- Duration: 24 hours
- Assertions:
  - p95 e2e latency < 5s
  - baseline_state lag p95 < 30s
  - identity_review_queue depth < 50 entries
  - No data loss across 24h window

### S2: Burst — 5x sustained rate for 1 hour

- Rate: ~1.15 events/sec aggregate
- Duration: 1 hour
- Assertions:
  - p99 e2e latency < 30s during burst
  - No data loss
  - Recovery to S1 latency targets within 10 minutes after burst ends

### S3: Failure injection

- Kill `IdentityMatchingProcessor` for 60 seconds during S1
- Assert: no data loss; backlog drains within 5 minutes after restart
- Repeat for `NormalisationProcessor` and `SubstrateWriterProcessor`

## Tooling

- Generator: Go binary at `backend/stream-services/substrate-pipeline/loadgen/` (V1)
- Metrics: Prometheus + Grafana dashboard at `monitoring/dashboards/substrate-pipeline.json` (V1)
- Topic depth: `kafka-consumer-groups.sh --describe`
- e2e timing: include `inbound_event_id` + `inbound_ts` in event headers; SubstrateWriter
  emits `substrate_updates` with `e2e_latency_ms = now - inbound_ts`

## Acceptance bar (Layer 2 §3.4)

| Metric                          | Target  | Source        |
|---------------------------------|---------|---------------|
| Observations/day per facility   | 2,000   | Layer 2 §3.4  |
| Residents per facility          | 200     | Layer 2 §3.4  |
| End-to-end latency (p95)        | < 5s    | Plan §2.7     |
| baseline_state lag (p95)        | < 30s   | Plan §2.7     |
| Sustained burst headroom        | 5x      | Internal      |

## Out of scope for Wave 2.7

- Actual test execution
- Grafana dashboard authoring
- Load generator implementation

These are V1 deliverables; this document captures the design only.
