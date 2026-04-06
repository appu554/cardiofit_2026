# Module 13 Clinical State Synchroniser — Production Runbook

## Overview

Module 13 is the **7-source fan-in state synchroniser** that maintains per-patient
`ClinicalStateSummary` state, computes CKM risk velocity (AHA 2023 Advisory), detects
clinical state changes, and writes back to KB-20 via a coalesced side output.

**Kafka consumer group**: `module13-clinical-state-sync`
**Input topics**: `module7-bp-variability-out`, `module8-comorbidity-out`,
`module9-engagement-out`, `module10b-meal-patterns-out`, `module11b-fitness-patterns-out`,
`module12-intervention-window-out`, `module12b-intervention-delta-out`, `enriched-patient-events-v1`
**Output topic**: `module13-state-changes`
**Side output**: KB-20 state updates (via `KB20AsyncSinkFunction`)

---

## 1. Feature Flag Reference

| Environment Variable | Default | Description |
|---|---|---|
| `MODULE13_ENABLED` | `true` | Master kill-switch. `false` = processElement is a no-op |
| `MODULE13_CKM_VELOCITY_ENABLED` | `true` | Enable CKM risk velocity computation |
| `MODULE13_STATE_CHANGES_ENABLED` | `true` | Enable state change detection and emission |
| `MODULE13_KB20_WRITEBACK_ENABLED` | `true` | Enable KB-20 coalesced writeback |
| `MODULE13_PERSONALIZED_TARGETS_ENABLED` | `true` | Enable A1 personalised target extraction |
| `MODULE13_DRY_RUN` | `false` | Shadow mode: compute all, emit nothing |
| `MODULE13_PILOT_PATIENT_PREFIX` | _(empty)_ | Cohort filter: only process matching patient IDs |
| `MODULE13_MAX_STATE_CHANGES_PER_EVENT` | `10` | Safety cap on changes per event |

---

## 2. Phased Rollout Plan

### Phase 0: Shadow Mode (Week 1)
```bash
MODULE13_ENABLED=true
MODULE13_DRY_RUN=true
MODULE13_PILOT_PATIENT_PREFIX=pilot-
```
- Deploy alongside existing pipeline
- All computation runs, zero downstream impact
- Validate metrics: check `module13_events_processed_total` is incrementing
- Validate latency: `module13_events_process_latency_ms` P95 < 5ms
- Validate velocity distribution: `module13_ckm_velocity_{improving,stable,deteriorating}_total`

### Phase 1: Pilot Cohort (Week 2)
```bash
MODULE13_ENABLED=true
MODULE13_DRY_RUN=false
MODULE13_PILOT_PATIENT_PREFIX=pilot-
```
- Enable real outputs for pilot patients only
- Monitor state change emission rates
- Verify KB-20 writeback: `module13_coalescing_kb20_updates_emitted_total`
- Clinical review of state change events for pilot patients

### Phase 2: Expand to 25% (Week 3)
```bash
MODULE13_ENABLED=true
MODULE13_DRY_RUN=false
MODULE13_PILOT_PATIENT_PREFIX=
```
- Remove patient prefix filter (all patients processed)
- Deploy with parallelism=1 initially
- Monitor throughput: target >5,000 events/sec per slot
- Monitor memory via TaskManager metrics

### Phase 3: Full Production (Week 4+)
```bash
MODULE13_ENABLED=true
MODULE13_DRY_RUN=false
MODULE13_PILOT_PATIENT_PREFIX=
```
- Scale parallelism to 2 (matching production config)
- Enable Grafana alerting rules
- Full clinical validation sign-off

---

## 3. Rollback Procedures

### Level 1: Soft Rollback (Outputs Only)
**Symptom**: Unexpected state changes, noisy alerts, incorrect velocity classifications
**Action**: Suppress outputs while keeping computation alive for debugging
```bash
# Set dry-run mode
MODULE13_DRY_RUN=true
# Restart Flink job (applies on open() re-initialization)
flink cancel <job-id> -s /checkpoints/module13-soft-rollback
flink run -s /checkpoints/module13-soft-rollback -d module13-job.jar clinical-state-sync
```
**Recovery time**: ~30 seconds (savepoint + restart)
**Data impact**: None — state is preserved, outputs suppressed

### Level 2: Feature Disable (Selective)
**Symptom**: Specific subsystem causing issues (e.g., CKM velocity is wrong, KB-20 writeback failing)
**Action**: Disable the specific feature flag
```bash
# Example: disable KB-20 writeback only
MODULE13_KB20_WRITEBACK_ENABLED=false
# Restart from latest checkpoint (no savepoint needed)
```

### Level 3: Full Disable
**Symptom**: Module 13 causing TaskManager instability, checkpoint failures, OOM
**Action**: Kill-switch the entire operator
```bash
MODULE13_ENABLED=false
# Restart — operator becomes a pass-through no-op
```
**Recovery time**: ~30 seconds
**Data impact**: State frozen at last update; resumes when re-enabled

### Level 4: Remove from Job Graph
**Symptom**: Even disabled operator causes serialization/deserialization issues
**Action**: Redeploy without Module 13 in the Flink job graph
```bash
# Take savepoint
flink cancel <job-id> -s /checkpoints/module13-removal

# Redeploy orchestrator without module13 case
# (Requires code change to FlinkJobOrchestrator or separate job JAR)
flink run -d flink-pipeline-no-module13.jar <job-type>
```
**Recovery time**: ~5 minutes (build + deploy)
**Data impact**: Module 13 Flink state is orphaned; must re-bootstrap on reintroduction

---

## 4. Health Checks

### Flink Metrics (Prometheus)
```promql
# Is Module 13 processing events?
rate(flink_module13_events_processed_total[5m]) > 0

# P95 processing latency
histogram_quantile(0.95, rate(flink_module13_events_process_latency_ms_bucket[5m]))

# CKM velocity distribution (should not be 100% UNKNOWN)
flink_module13_ckm_velocity_unknown_total / flink_module13_events_processed_total < 0.1

# Data completeness (should be > 0.5 for active patients)
histogram_quantile(0.50, rate(flink_module13_data_quality_completeness_score_bucket[5m]))

# KB-20 writeback health
rate(flink_module13_coalescing_flush_total[5m]) > 0

# No buffer overflow (evictions should be rare)
rate(flink_module13_coalescing_eviction_total[5m]) < 1
```

### KB-20 Service Metrics
```promql
# Target computation throughput
rate(kb20_target_computations_total[5m])

# Target computation latency
histogram_quantile(0.95, rate(kb20_target_compute_duration_seconds_bucket[5m]))

# Cache effectiveness
kb20_target_cache_hits_total / (kb20_target_cache_hits_total + kb20_target_cache_misses_total)

# Outbox relay errors (should be 0)
rate(kb20_outbox_relay_errors_total[5m]) == 0
```

---

## 5. Alert Rules

### Critical (Page)
| Alert | Condition | Action |
|---|---|---|
| Module13Down | `rate(flink_module13_events_processed_total[10m]) == 0` AND job running | Check TaskManager logs, restart job |
| Module13HighLatency | P95 > 50ms for 5 minutes | Check GC pressure, reduce parallelism |
| KB20WritebackFailing | `rate(kb20_outbox_relay_errors_total[5m]) > 0` | Check KB-20 service health, network |
| CheckpointFailure | 3 consecutive checkpoint failures | Check state size, RocksDB compaction |

### Warning (Slack)
| Alert | Condition | Action |
|---|---|---|
| HighUnroutableRate | >5% events unroutable over 15m | Check source module tagging |
| CoalescingOverflow | eviction_total increasing | Increase MAX_COALESCING_BUFFER_SIZE or reduce COALESCING_WINDOW_MS |
| LowCompleteness | Median completeness < 0.3 for 1h | Check upstream module health |
| AllVelocityUnknown | >50% UNKNOWN velocity for 30m | Check snapshot rotation, data freshness |

---

## 6. Operational Procedures

### Savepoint Before Config Change
```bash
# Always take savepoint before changing feature flags
SAVEPOINT_PATH=$(flink cancel <job-id> -s /checkpoints/$(date +%Y%m%d-%H%M%S)-pre-change)
echo "Savepoint: $SAVEPOINT_PATH"

# Apply new environment variables, then restart
flink run -s $SAVEPOINT_PATH -d module13-job.jar clinical-state-sync
```

### State Inspection
```bash
# Check state size per key (requires Flink REST API)
curl http://flink-jobmanager:8081/jobs/<job-id>/vertices/<vertex-id>/subtasks/metrics?get=State.stateSize

# Check checkpoint duration
curl http://flink-jobmanager:8081/jobs/<job-id>/checkpoints/details/<checkpoint-id>
```

### Emergency Patient Exclusion
If a specific patient is causing issues (corrupt state, infinite loop):
```bash
# Option 1: Set pilot prefix to exclude (if using prefix filtering)
MODULE13_PILOT_PATIENT_PREFIX=safe-

# Option 2: Clear patient state via savepoint manipulation
# (Requires Flink State Processor API — advanced)
```

---

## 7. Dependencies

| Dependency | Impact if Down | Mitigation |
|---|---|---|
| Kafka broker | No events processed | Job pauses at consumer, auto-resumes |
| KB-20 service | Personalised targets unavailable, writeback fails | Falls back to population defaults; writeback buffered |
| Upstream modules (7-12) | Partial data, lower completeness scores | Data absence alerts fire; velocity becomes UNKNOWN |
| RocksDB state backend | Checkpoint failures | Monitor compaction, disk space |
| Prometheus | No observability | Flink Web UI still shows basic metrics |

---

## 8. Key Contacts

| Role | Responsibility |
|---|---|
| Flink Platform Team | Job deployment, savepoints, cluster health |
| Clinical Engineering | State change validation, CKM velocity thresholds |
| KB-20 Service Owner | Target computation, writeback API health |
| On-Call SRE | Alert response, rollback execution |

---

## 9. Revision History

| Date | Change | Author |
|---|---|---|
| 2026-04-06 | Initial runbook for Module 13 production pilot | CardioFit Engineering |
