# ADR: Streaming Pipeline Architecture Choice — Layer 2 Substrate

**Status:** Proposed (Wave 2.7)
**Date:** 2026-05-06
**Decision-makers:** TBD (Engineering Lead, Platform Lead)
**Consulted:** Layer 2 doc §3.4 author, existing stream-services Stage 1 owners

## Context

The v2 substrate (Layer 2) requires a real-time pipeline for:
- Identity matching of incoming records (eNRMC, MHR, hospital discharge)
- AMT/SNOMED-CT-AU normalisation
- Substrate write (MedicineUse + Observation + Event)
- Clinical state machine updates (running baselines, active concern lifecycle)

Per Layer 2 doc §3.4, expected volume at scale: tens of thousands of events daily across
200-bed facility cohorts. Per-facility: 1,000-2,000 observation events daily.

Existing footprint:
- Confluent Cloud Kafka in production (per CLAUDE.md root)
- Stream Stage 1 (Java, Spring Boot + Kafka Streams 3.6.0) and Stage 2 (Python) services already deployed
- Java/Spring expertise on the team

## Options Considered

### Option A — Apache Flink (separate cluster)

**Pros:**
- Mature window-join support
- Stronger exactly-once semantics across multi-topic joins
- Existing operational expertise in some Vaidshala teams

**Cons:**
- Separate cluster to deploy/operate (Flink JobManager + TaskManagers)
- Operational overhead vs library mode
- Higher latency for simple stateless transforms

### Option B — Kafka Streams (library mode)

**Pros:**
- Library — deploys as a JVM sidecar alongside any Java service
- No separate cluster
- Operational simplicity matches existing Stage 1 footprint
- Confluent Cloud Kafka Streams works without modification
- Sufficient for current Layer 2 use cases (stateless transforms, single-topic stateful aggregations)

**Cons:**
- Stateful joins across multiple topics are clunkier than Flink
- Re-balancing on scaling can be slow
- Some windowing semantics weaker than Flink

### Option C — Re-use existing Stage 1 (Stage 1 service in Java)

**Pros:**
- No new module
- Single team owns it

**Cons:**
- Stage 1 has different scope (regulatory ingest validation); muddying responsibilities is bad design
- Coupling Substrate stream lifecycle to regulatory stream lifecycle creates change-amplification risk

## Decision

**Option B — Kafka Streams library mode**, deployed as a Java sidecar at
`backend/stream-services/substrate-pipeline/`.

Rationale: lowest operational overhead, matches existing Stage 1 footprint (same Kafka Streams
major version, same Confluent Cloud cluster), and Layer 2 §3.4 topology is dominated by
stateless transforms + single-topic stateful aggregations that Streams handles well.

Revisit Flink (Option A) only when one of the following triggers fires:
- Multi-topic window-join requirement that Streams cannot express simply
- Operational scaling requirements that exceed library-mode practical capacity
  (>50K events/sec sustained)
- Strong exactly-once requirement across cross-topic transactions

## Consequences

**Positive:**
- New `substrate-pipeline` module isolated; can iterate without touching Stage 1
- Same Confluent Cloud cluster — no infrastructure cost
- Java team owns end-to-end

**Negative / risks:**
- Library-mode Streams has weaker fault-tolerance than Flink for complex joins; mitigated
  because current Layer 2 topology is stateless or single-topic stateful
- Future Flink migration is non-trivial if triggered; the ADR re-opens then

## Implementation sequencing

Wave 2.7 produces (this ADR's deliverable):
- Module skeleton + topology doc + load test plan
- ADR captured with status=Proposed

V1 work (post Wave 2.7) fills in:
- `IdentityMatchingProcessor` real logic (kb-20 `/v2/identity/match` HTTP client + retry + DLQ)
- `NormalisationProcessor` real logic (kb-7-terminology client + cache)
- `SubstrateWriterProcessor` with kb-20 REST integration (POST observations / medicine_uses / events)
- Operational runbook + Prometheus metrics per processor
- Production load test execution (per `load_test_plan.md`)
- Confluent Cloud topic provisioning (append substrate topics to `setup-kafka-topics.py`)

## Open questions

- Confluent Cloud quota for substrate topics (TBD with infra team)
- Schema registry strategy for substrate event payloads (TBD; Avro vs Protobuf)
- Dead-letter-topic policy for unparseable inbound events (TBD)
- Whether substrate-pipeline should run as a sidecar to kb-20 pods or as an independent
  StatefulSet on the cluster (TBD; Streams library mode supports both)
