# Vaidshala Platform Deployment Guide (GCP)

Version: April 2026 alignment with Vaidshala Complete Architecture Registry + Flink Architecture v3

---

## 1. Scope & Goals
This document captures the production deployment blueprint for every major runtime in the Vaidshala CDS platform on Google Cloud Platform:
- Trajectory-aware capture & scribe workflow (Module 0)
- Flink streaming modules 1–13 and Kafka backbone
- Module 13 fan-in + KB-20/23 persistence
- Knowledge Base services (snapshot + runtime tiers)
- V-MCU engines and ICU Intelligence layer
- API ingress, networking, CI/CD, security & compliance controls
Use it as the single reference for Ops, SRE, and clinical engineering teams when provisioning or auditing the environment.

---

## 2. Regional Footprint & Projects
| Region | Project(s) | Workloads | Notes |
|--------|------------|-----------|-------|
| asia-south1 | `vaidshala-core` | Flink Modules 0–6, API Gateway, Pub/Sub, capture services | Low latency to India ops team, cheaper stateless compute |
| asia-southeast1 | `vaidshala-core-dr` | Warm standby for Modules 0–6 via Config Sync | Enables rapid failover if Mumbai degraded |
| australia-southeast1 | `vaidshala-clinical` | Flink Modules 7–13, KB-20/23, V-MCU, KB runtimes, Module 13 sinks | Clinical data sovereignty (Australian patients) |
Shared VPC spans projects; Private Service Connect exposes Confluent Cloud, Cloud SQL, Memorystore; Cloud NAT is centralized.

---

## 3. Capture → Flink → KB Flow
1. **Scribe capture** (iPad with on-device Whisper + VAD) sends encrypted transcript segments to API Gateway endpoint `/v1/capture/segments` under Cloud Armor protection; JWT issued via Workload Identity, device attested via BeyondCorp.
2. API Gateway writes validated payloads to Pub/Sub topic `clinical-capture-raw` (Avro schema, CMEK, 24h retention + DLQ).
3. **Flink Module 0** (asia-south1) consumes Pub/Sub with exactly-once connector, runs JWT validation, PHI de-identification, KB-2 enrichment, emits canonical events into Confluent Kafka topic `clinical.events.canonical` (Rule 8 adapter serializer ensures `source_module`).
4. Flink Modules 1–6 perform normalization, CDS enrichment, and push to downstream Kafka topics (ingestion cluster).
5. Cross-region replication sends topics to australia-southeast1 where Modules 7–12 process domain-specific logic (BP Variability, CID, Engagement, Meal/Activity correlators, Intervention monitors).
6. **Module 13** fans in all domain outputs, computes CKM velocities, and writes idempotent state rows to KB-20 Postgres; also emits `clinical.state-change-events` for KB-23 real-time cards.
7. KB-23, KB-26, and V-MCU read fused state from KB-20 instead of raw Kafka topics (Gap G10 resolved).

---

## 4. Flink Runtime Architecture
- **Clusters:** `gke-core-streaming` (asia-south1) and `gke-domain-streaming` (australia-southeast1) on GKE Standard.
- **Node pools:**
  - `jm-pool` e2-medium (3–6 nodes)
  - `tm-stateless` c3-highmem-4 (6–24 nodes, Spot enabled)
  - `tm-stateful` c3-highmem-8 + SSD PD (6–18 nodes)
- **Config overrides:**
  ```properties
  execution.checkpointing.interval=10s
  execution.checkpointing.min-pause=5s
  execution.checkpointing.timeout=60s
  execution.checkpointing.mode=EXACTLY_ONCE
  state.backend=rocksdb
  state.backend.incremental=true
  state.backend.rocksdb.localdir=/mnt/ssd/rocksdb
  taskmanager.memory.managed.fraction=0.40
  restart-strategy.fixed-delay.attempts=3
  restart-strategy.fixed-delay.delay=10s
  metrics.latency.interval=5s
  ```
- **Checkpoints & savepoints:** Stored in CMEK bucket `gs://vaidshala-prod-flink-checkpoints` (dual-region). Retain 8 checkpoints, daily savepoints (14 days). Chaos tests validate 10-second recovery RTO.
- **Deployment:** Cloud Build builds shared runtime image `registry-docker.pkg.dev/<proj>/flink/vaishala-runtime:<gitsha>` (Rule 8 adapters). Cloud Deploy rolls out with 10% TM canary, wait for healthy checkpoint, then 100%.

---

## 5. Module 13 Fan-in & KB-20 Persistence
- Namespace `module13` with HPA (CPU 60%, custom `processing_latency_ms`).
- Sink pattern: Flink 2PC JDBC + Cloud SQL Postgres (regional HA, private IP, PITR 7d, CMEK). Redis caches fed via Datastream CDC from Postgres—no dual writes.
- Idempotent schema:
  ```sql
  CREATE TABLE kb20_state_updates (
      patient_id UUID,
      state_channel TEXT,
      observation_id UUID,
      event_time TIMESTAMPTZ NOT NULL,
      payload JSONB NOT NULL,
      measurement_confidence NUMERIC(5,4) NOT NULL,
      PRIMARY KEY (patient_id, state_channel, observation_id)
  );
  CREATE TABLE clinical_state_change_events (
      event_id UUID PRIMARY KEY,
      patient_id UUID NOT NULL,
      change_type TEXT NOT NULL,
      emitted_at TIMESTAMPTZ NOT NULL,
      payload JSONB NOT NULL
  );
  ```
- Circuit breaker trips after 5 slow/w failed writes (>2s); emits `clinical.state-delay` metric. Runbook dictates scaling TMs or throttling ingestion.
- Cutover: (1) Shadow tables, (2) reader switch, (3) retire legacy consumers.

---

## 6. Knowledge Base Services
- **Snapshot KBs** (KB-1 Drug Rules, KB-4 Safety, KB-5 Interactions, KB-6 Formulary, KB-8 Calculators, KB-11 Population, KB-16 Lab Interpretation, KB-17 Registry): packaged as Go/CQL microservices on Cloud Run (min instances per SLA); cold start <1s. KB-7 Terminology runs on GKE Autopilot w/ Cloud Spanner backend due to 5.3M+ code set.
- **Runtime KBs** (KB-2 Context Aggregator, KB-3 Guidelines, KB-9 Care Gaps, KB-10 Rules Engine, KB-12 Order Sets, KB-13 Quality Measures) reside on `gke-kb-runtime` (australia-southeast1) with namespaces per tier, OpenTelemetry exporters, dual HPAs (CPU + `grpc_server_handled_latency`).
- **Persistence:** Postgres workloads on Cloud SQL (HA, private IP, PITR). Nightly dumps to CMEK GCS for DR; quarterly restore drills. Terminology/lookup data on Spanner or partitioned Postgres per component.
- **Ingress:** Internal HTTP(S) Load Balancers for service mesh, API Gateway + Cloud Armor for external partners. Auth via JWT (Workload Identity Federation) + IAM Conditions.

---

## 7. V-MCU Engines & ICU Intelligence
- Dedicated `gke-vmcu` cluster (Shielded Nodes, Binary Authorization, Anthos Service Mesh). Medication Advisor Engine, Execution Layer, KB-18/19 governance, ICU Intelligence (Tier 7) deploy here with strict PodDisruptionBudgets (min 2 replicas).
- Auxiliary engagement/BCE services run on Cloud Run or Autopilot but communicate through PSC-secured Pub/Sub topics (e.g., `alerts.relapse-risk`).
- V-MCU run cycles pull fused state from KB-20 and push actions through Messaging (WhatsApp, email) via dedicated DMZ connectors.

---

## 8. API Gateway & Frontend Access
- External HTTPS Load Balancer + Cloud Armor WAF in front of API Gateway; TLS certs managed via Certificate Manager (`api.vaidshala.health`).
- Gateway definitions per client type (clinician app, patient portal, partner APIs). Routes target internal LBs for KB services, ASM ingress for V-MCU, and Cloud Run for UI APIs.
- Scribe capture path uses same gateway with device-specific JWT + BeyondCorp policies.

---

## 9. Event Backbone & Schema Governance
- Confluent Cloud Kafka as system-of-record; PSC endpoints provide private connectivity. Consumer groups follow `flink-module{N}-{function}-v1` convention.
- Schema Registry compatibility = `BACKWARD_TRANSITIVE`; prod jobs pin schema versions (no auto-register). Release flow: register schema → deploy readers → wait ≥2 checkpoints → enable producers → deprecate old version after 24h.
- Module 0 uses Pub/Sub pre-stage for replay safety; DLQ monitors invalid payloads.

---

## 10. CI/CD & Observability
- Cloud Build builds containers (Flink runtime, KB services, V-MCU). Container Analysis vulnerability scanning gates release.
- Cloud Deploy manages progressive rollouts with manual approval gates linked to clinical validation board.
- Config Sync + Policy Controller enforce guardrails (namespaces, PodSecurity, network policies).
- Observability: Managed Service for Prometheus scrapes Flink/KB metrics; Cloud Logging centralizes logs; BigQuery sink stores audit trails. Alerting covers `processing_latency_ms`, `events_rejected`, Module 13 circuit breaker, KB-20 freshness, CPU saturation.

---

## 11. Security & Compliance
- VPC Service Controls wrap prod projects; Access Context Manager restricts console/API access to BeyondCorp-managed devices.
- Secrets in Secret Manager with automatic rotation (Terraform-managed). CMEK everywhere (GCS, disks, Cloud SQL, Pub/Sub).
- Cloud DLP scans exported datasets; differential privacy for analytics. Compliance targets IEC 62304, HIPAA-equivalent safeguards, Australian privacy principles.

---

## 12. Operational Runbooks (Summary)
1. **Flink lag >2s:** check TM utilization, Kafka lag, scale or reconfigure; use savepoint + redeploy if persistent.
2. **KB-20 write failures:** inspect `pg_stat_activity`, PgBouncer pool, Module 13 breaker metrics, Cloud SQL query insights.
3. **Scribe mismatch/trajectory stale:** verify KB-20 `max(updated_at)`, Module 13 checkpoint age, restart from latest savepoint.
4. **Schema rollout incident:** roll back producer to previous schema, redeploy readers pinned to prior version, replay via Pub/Sub DLQ.

---

## 13. Next Steps
- Finish Module 0 implementation + schema enforcement in Gateway/Pub/Sub.
- Apply KB-20 DDL + configure Datastream feeds to analytics/caches.
- Bake Flink config overrides + managed memory fraction into Helm charts; run failover drills.
- Document Module 13 cutover/rollback, train Ops on runbooks, and launch clinical validation (Fatal Risks FR1–FR3).

