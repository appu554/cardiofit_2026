# substrate-pipeline (Wave 2.7 SKELETON)

Kafka Streams sidecar driving the Layer 2 v2-substrate streaming topology:

```
raw_inbound_events
  → IdentityMatchingProcessor → identified_events
  → NormalisationProcessor    → normalised_events
  → SubstrateWriterProcessor  → substrate_updates
```

## Status

This module is a **scaffold only**. It exists to:

1. Land the architectural decision (see ADR
   `docs/adr/2026-05-06-streaming-pipeline-choice.md` — Kafka Streams library mode chosen).
2. Capture the topology diagram (see
   `backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/streaming/topology.md`).
3. Reserve the module shape so V1 runtime work can fill in processor logic
   without re-litigating naming / package / dependency decisions.

It does **not** ship working stream processing. Every method body and processor
implementation is marked with `TODO(wave-2.7-runtime)`.

## Layout

```
substrate-pipeline/
├── pom.xml                                                    # Maven build descriptor
├── Dockerfile                                                 # Java 17 + maven-built jar
├── README.md                                                  # this file
└── src/main/
    ├── java/health/vaidshala/substrate/
    │   ├── Main.java                                          # entrypoint (loads config, starts streams)
    │   ├── SubstrateStreamApp.java                            # buildTopology() — three placeholder streams
    │   └── processors/
    │       ├── IdentityMatchingProcessor.java                 # interface stub — calls kb-20 /v2/identity/match
    │       ├── NormalisationProcessor.java                    # interface stub — calls kb-7-terminology
    │       └── SubstrateWriterProcessor.java                  # interface stub — calls kb-20 REST (NOT direct DB)
    └── resources/
        └── application.properties                             # placeholder Kafka config
```

## Why a thin REST proxy and not direct DB writes?

`SubstrateWriterProcessor` calls kb-20's REST surface. Go owns the substrate
transaction boundary (outbox semantics, baseline_state recompute, downstream
event fan-out). Letting Java write directly would split transactional ownership
and break the v2 substrate's invariants. This is intentional and is captured in
the ADR's "Decision" section.

## Building (V1 — not Wave 2.7)

```
mvn -B package
```

The pom is syntactically reasonable but Wave 2.7 does **not** require a green
Maven build. V1 will (a) add real processor implementations, (b) wire integration
tests via `kafka-streams-test-utils`, (c) execute the load test in
`shared/v2_substrate/streaming/load_test_plan.md`.

## V1 acceptance (not Wave 2.7 acceptance)

Per the plan, "topology deployed in dev" + "load test green" are V1 deliverables:

- 2,000 obs/day per facility, 200 residents
- e2e p95 < 5s
- baseline_state lag p95 < 30s
- Grafana dashboard observable

See `shared/v2_substrate/streaming/load_test_plan.md`.

## Topic provisioning

The four substrate topics (`raw_inbound_events`, `identified_events`,
`normalised_events`, `substrate_updates`) are **not yet** in
`backend/stream-services/setup-kafka-topics.py`. V1 must append them — the
existing file uses a `TOPICS_CONFIG` list with hardcoded Confluent Cloud
credentials, so the addition is straightforward but requires touching a file
that holds secrets and is best edited deliberately rather than during scaffolding.
See the TODO marker in `topology.md`.
