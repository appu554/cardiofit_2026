# Vaidshala Streaming & KB Deployment (GCP)

## 1. Purpose
This README distills the production deployment architecture for the Vaidshala Flink streaming layer, Module 13 fan-in, KB services, V-MCU engines, and scribe capture path on Google Cloud Platform. It is the operational guide for Platform/Ops teams preparing the dual-region rollout described in the April 2026 architecture docs.

## 2. Regional Footprint
| Region | Workloads | Notes |
|--------|-----------|-------|
| asia-south1 (Mumbai) | Flink Modules 0-6 + API Gateway + Pub/Sub | Near ingestion sources, cost efficient |
| asia-southeast1 (Singapore) | Failover for Modules 0-6 | Warm standby via Config Sync |
| australia-southeast1 (Sydney) | Flink Modules 7-13 + Module 13 sinks + KB-20/23 + V-MCU core | Clinical data sovereignty |

## 3. Capture-to-KB Flow
1. **iPad / Trajectory Scribe** → on-device Whisper streams transcript segments over mTLS to API Gateway (`/v1/capture/segments`).
2. API Gateway validates JWT (Workload Identity) + Cloud Armor, publishes payload to Pub/Sub `clinical-capture-raw` (Avro schema, 24h retention, CMEK).
3. **Flink Module 0** (asia-south1) subscribes via Pub/Sub connector, runs:
   - `ValidateJwtFunction`
   - `DeidentifyMapFunction` (PHI hashing + audit log)
   - `EnrichAsyncFunction` (KB-2 context lookup)
   - Emits canonical events to Confluent Kafka topic `clinical.events.canonical` with Rule 8 adapter serializer.
4. Modules 1-6 normalize + enrich, pass to Modules 7-12 in Sydney.
5. **Module 13** fans in all domain topics, computes CKM velocity, writes idempotent state rows to KB-20 (Cloud SQL Postgres) and emits `clinical.state-change-events` for KB-23 real-time cards.

## 4. Flink Runtime
- Two regional GKE Standard clusters: `gke-core-streaming` (asia-south1) and `gke-domain-streaming` (australia-southeast1).
- Node pools:
  - `jm-pool` e2-medium (3-6 nodes)
  - `tm-stateless` c3-highmem-4 (6-24 nodes, Spot enabled)
  - `tm-stateful` c3-highmem-8 + SSD PD (6-18 nodes)
- Critical config:
  ```properties
  execution.checkpointing.interval=10s
  execution.checkpointing.min-pause=5s
  execution.checkpointing.timeout=60s
  state.backend=rocksdb
  state.backend.rocksdb.localdir=/mnt/ssd/rocksdb
  taskmanager.memory.managed.fraction=0.40
  restart-strategy.fixed-delay.attempts=3
  ```
- Checkpoints stored in CMEK bucket `gs://vaidshala-prod-flink-checkpoints`; daily savepoints retained 14 days.
- Cloud Deploy rollout policy: canary 10% TMs, wait for healthy checkpoint, then full rollout.

## 5. Module 13 Fan-in + KB-20 Persistence
- Dedicated namespace + HPA (CPU 60%, `processing_latency_ms` custom metric).
- Sinks use Flink two-phase-commit JDBC connector → Cloud SQL Postgres (HA, private IP, PITR 7d).
- Tables enforce idempotency:
  ```sql
  CREATE TABLE kb20_state_updates (
      patient_id UUID,
      state_channel TEXT,
      observation_id UUID,
      event_time TIMESTAMPTZ,
      payload JSONB,
      measurement_confidence NUMERIC(5,4),
      PRIMARY KEY (patient_id, state_channel, observation_id)
  );
  ```
- Redis caches fed by Datastream/CDC from Postgres (no dual writes). Circuit breaker trips after 5 slow calls (>2s) and emits `clinical.state-delay` side output.
- Cutover steps: shadow writes → reader switch → retire direct topic consumers (resolves Gap G10).

## 6. KB Services & V-MCU
- Snapshot KBs (KB-1,4,5,6,8,11,16,17) on Cloud Run (min instances 0/1). KB-7 terminology on Spanner-backed Autopilot GKE.
- Runtime KBs (KB-2,3,9,10,12,13) on `gke-kb-runtime` with namespaces per tier; OpenTelemetry exports `grpc_server_handled_latency` for HPA.
- KB-20/23 + V-MCU core on hardened `gke-vmcu` (Shielded Nodes, Binary Authorization, ASM mTLS). ICU Intelligence has PodDisruptionBudget 2/3.

## 7. Networking & Security
- Shared VPC with workload subnets + Private Service Connect endpoints (Confluent, Cloud SQL, Memorystore).
- API Gateway + External HTTPS LB + Cloud Armor as the single public entry for frontends.
- IAM via Workload Identity; secrets in Secret Manager (auto-rotation). VPC Service Controls + BeyondCorp for human access.

## 8. Schema & Release Discipline
- Confluent Schema Registry compatibility `BACKWARD_TRANSITIVE`; prod jobs pin exact versions (no auto-register).
- Deployment flow: register new schema → deploy reader jobs → wait 2 checkpoints → cut producer.
- Module 0, 13, KB services built via Cloud Build, scanned (Container Analysis), promoted with Cloud Deploy manual gates tied to clinical validation sign-off.

## 9. Runbooks
1. **Flink lag >2s**: check TM CPU, Kafka lag, bump parallelism, inspect RocksDB, savepoint + redeploy if needed.
2. **KB-20 write failures**: inspect `pg_stat_activity`, PgBouncer pool, Module 13 circuit breaker metrics, Cloud SQL query insights.
3. **Scribe mismatch**: verify KB-20 freshness (`max(updated_at)`), Module 13 checkpoint age, restart from latest savepoint if stale.

## 10. Next Actions
- Implement Module 0 job + API Gateway schema enforcement.
- Apply KB-20 DDL + Datastream to analytics/cache targets.
- Bake Flink config overrides into Helm charts; run chaos tests for checkpoint recovery.
- Document Module 13 cutover + rollback in ops wiki and train on runbooks.
