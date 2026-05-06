# Layer 2 Streaming Topology

Per Layer 2 doc §3.4 + ADR `docs/adr/2026-05-06-streaming-pipeline-choice.md`.

Status: **Wave 2.7 scaffold** — runtime to be filled in by V1 work.

## Topology diagram

```
[Source connectors: CSV poller, MHR FHIR Gateway, HL7 listener]
         ↓ produce records
[Topic: raw_inbound_events]
         ↓
[Processor: IdentityMatchingProcessor]
   - reads raw event
   - calls kb-20 /v2/identity/match (HTTP)
   - on HIGH/MEDIUM: emits resident_ref onto identified_events
   - on LOW/NONE: writes to identity_review_queue + does NOT emit
         ↓
[Topic: identified_events]
         ↓
[Processor: NormalisationProcessor]
   - reads identified event
   - calls kb-7-terminology for AMT lookup
   - calls kb-7 for SNOMED indication lookup (when free-text indication present)
   - emits normalised event
         ↓
[Topic: normalised_events]
         ↓
[Processor: SubstrateWriterProcessor]
   - reads normalised event
   - calls kb-20 REST (POST /v2/observations | /v2/medicine_uses | /v2/events) —
     preserves transactional ownership in Go; the Java side is a thin proxy
   - emits to substrate_updates with kb-20 response (id + delta payload)
         ↓
[Topic: substrate_updates]
         ↓ consumed by:
   - kb-20 outbox for downstream listeners
   - clinical_state_updater (Wave 2.3 active-concern engine OnEvent etc.)
   - eventually rule_trigger_evaluator (Layer 3)
```

## Topics

| Topic                | Schema (proposed)         | Retention | Notes                                           |
|----------------------|---------------------------|-----------|-------------------------------------------------|
| raw_inbound_events   | TBD (Avro proposed)       | 7 days    | source-of-truth for replay                      |
| identified_events    | TBD                       | 7 days    | adds resident_ref + match metadata              |
| normalised_events    | TBD                       | 7 days    | adds AMT/SNOMED codes                           |
| substrate_updates    | TBD                       | 30 days   | downstream consumers (rules, dashboards)        |

> TODO(wave-2.7): append substrate topic creation to
> `backend/stream-services/setup-kafka-topics.py`. The existing file hardcodes
> Confluent Cloud credentials and a single `TOPICS_CONFIG` list — V1 work should
> add the four substrate topics there (12 partitions, RF=3, snappy compression,
> min.insync.replicas=2) following the existing entry shape.

## Processor scaling

Each processor scales horizontally; partitioning key per topic = `resident_ref`
(after IdentityMatching) or `inbound_record_id` (raw_inbound_events).

## Module layout

```
backend/stream-services/substrate-pipeline/
├── pom.xml
├── Dockerfile
├── README.md
└── src/main/
    ├── java/health/vaidshala/substrate/
    │   ├── Main.java
    │   ├── SubstrateStreamApp.java
    │   └── processors/
    │       ├── IdentityMatchingProcessor.java
    │       ├── NormalisationProcessor.java
    │       └── SubstrateWriterProcessor.java
    └── resources/
        └── application.properties
```

## TODO(wave-2.7-runtime)

- Concrete Avro schemas for the four topics
- IdentityMatchingProcessor full logic (HTTP retry policy, dead-letter routing for transport errors)
- NormalisationProcessor full logic (kb-7-terminology client + cache)
- SubstrateWriterProcessor full logic (HTTP retry, idempotency keys)
- Confluent Cloud topic provisioning automation
- Prometheus metrics per processor
- Load test execution per `load_test_plan.md`
