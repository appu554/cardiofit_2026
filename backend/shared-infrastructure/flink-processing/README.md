# Flink Clinical Intelligence Pipeline

Apache Flink stream processing for the CardioFit platform. Processes patient vitals, labs, medications, and engagement signals through 10 clinical intelligence modules (M7-M13) to produce BP variability metrics, comorbidity safety alerts, meal/activity responses, intervention tracking, and CKM risk velocity scores.

## Architecture

```
                     enriched-patient-events-v1
                              │
          ┌───────────────────┼───────────────────────────┐
          │                   │                           │
       ┌──▼──┐  ┌──▼──┐  ┌──▼──┐  ┌──▼───┐  ┌──▼───┐  ┌──▼──┐
       │ M7  │  │ M8  │  │ M9  │  │ M10  │  │ M11  │  │ M12 │
       │ BP  │  │ CID │  │ Eng │  │ Meal │  │ Act  │  │ Int │
       └──┬──┘  └──┬──┘  └──┬──┘  └──┬───┘  └──┬───┘  └──┬──┘
          │        │        │        │          │         │
          └────────┴────────┴────────┴──────────┴─────────┘
                              │
                        ┌─────▼─────┐
                        │    M13    │
                        │ Clinical  │
                        │ State Sync│
                        └─────┬─────┘
                              │
                   clinical.state-change-events
```

## Modules

| Module | Name | Kafka Output Topic | Timer |
|--------|------|-------------------|-------|
| **M7** | BP Variability Engine | `flink.bp-variability-metrics` | Event-time |
| **M8** | Comorbidity Interaction | `alerts.comorbidity-interactions` | Event-time |
| **M9** | Engagement Monitor | `flink.engagement-signals` | Daily 23:59 UTC |
| **M10** | Meal Response Correlator | `flink.meal-response` | Meal + 3h05m |
| **M10b** | Meal Pattern Aggregator | `flink.meal-patterns` | Weekly Monday |
| **M11** | Activity Response | `flink.activity-response` | Activity + 2h05m |
| **M11b** | Fitness Pattern Aggregator | `flink.fitness-patterns` | Weekly Monday |
| **M12** | Intervention Window Monitor | `clinical.intervention-window-signals` | Processing-time |
| **M12b** | Intervention Delta Computer | `flink.intervention-deltas` | On WINDOW_CLOSED |
| **M13** | Clinical State Synchroniser | `clinical.state-change-events` | 7-day rotation |

## Quick Start

### Prerequisites

- Docker + Docker Compose
- Maven 3.9+ / Java 17+
- Kafka running on `cardiofit-lite` network:
  ```bash
  cd ../kafka && docker compose -f docker-compose.hpi-lite.yml up -d
  ```

### One-Command Deploy

```bash
make deploy
```

This runs: `build JAR` -> `start Flink` -> `create Kafka topics` -> `submit 10 jobs` -> `verify`

### Step-by-Step (if you prefer manual)

```bash
# 1. Build
make build-quick

# 2. Start Flink (JobManager + TaskManager + auto-submitter)
make start-e2e

# 3. Wait for Flink to be ready
make wait-flink

# 4. Create all Kafka topics
make create-topics

# 5. Submit all 10 module jobs
make submit-all

# 6. Verify
make verify
```

## Makefile Reference

### Deployment

| Command | Description |
|---------|-------------|
| `make deploy` | Full deploy: build + start + topics + submit + verify |
| `make deploy-prod` | Production deploy (2 TaskManagers, 3x replication) |
| `make redeploy` | Rebuild JAR + restart jobs (keeps Flink running) |
| `make stop` | Stop all Flink containers |
| `make clean` | Stop + remove build artifacts |

### Monitoring

| Command | Description |
|---------|-------------|
| `make status` | Job status + container health |
| `make health` | Flink slot availability + job counts |
| `make jobs` | List running jobs (compact) |
| `make topic-counts` | Message counts per M7-M13 output topic |
| `make logs` | Tail TaskManager logs |

### Operations

| Command | Description |
|---------|-------------|
| `make submit-all` | Submit all 10 module jobs |
| `make cancel-all` | Cancel all running jobs |
| `make create-topics` | Create all M7-M13 Kafka topics |
| `make reset-topics` | Delete + recreate output topics (with confirmation) |
| `make list-topics` | List relevant Kafka topics |

### Testing

| Command | Description |
|---------|-------------|
| `make test` | Run all unit tests |
| `make test-m13` | Run Module 13 tests only |
| `make e2e` | Generate dataset + run 3-patient 14-day E2E test |

## Docker Compose Files

| File | Use Case | Services |
|------|----------|----------|
| `docker-compose.e2e-flink.yml` | Development / E2E testing | 1 JobManager, 1 TaskManager, auto-submitter, Kafka UI |
| `docker-compose.yml` | Production | 1 JobManager, 2 TaskManagers, Prometheus, Grafana |

### E2E Stack (Development)

```bash
docker compose -f docker-compose.e2e-flink.yml up -d
```

- Flink Web UI: http://localhost:8181
- Kafka UI: http://localhost:9080
- Memory: ~2.5 GB (768 MB JobManager + 2 GB TaskManager)
- Auto-submits all 10 M7-M13 jobs on startup
- Connects to external Kafka on `cardiofit-lite` network

### Production Stack

```bash
docker compose up -d
```

- Flink Web UI: http://localhost:8081
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- Memory: ~14 GB (2 GB JobManager + 6 GB per TaskManager x 2)
- RocksDB state backend with exactly-once checkpointing (30s)
- Requires manual job submission: `make submit-all-prod`

## Kafka Topics

### Input Topics (consumed by M7-M13)

| Topic | Producer | Consumers |
|-------|----------|-----------|
| `ingestion.vitals` | Ingestion Service | M7 |
| `enriched-patient-events-v1` | Module 1b | M7, M8, M9, M10, M11, M12, M13 |
| `clinical.intervention-events` | Clinical Service | M12, M12b |

### Output Topics (produced by M7-M13)

| Topic | Module | Partitions | Retention |
|-------|--------|------------|-----------|
| `flink.bp-variability-metrics` | M7 | 8 | 30d |
| `alerts.comorbidity-interactions` | M8 | 4 | 90d |
| `flink.engagement-signals` | M9 | 4 | 30d |
| `flink.meal-response` | M10 | 8 | 30d |
| `flink.meal-patterns` | M10b | 4 | 90d |
| `flink.activity-response` | M11 | 8 | 30d |
| `flink.fitness-patterns` | M11b | 4 | 90d |
| `clinical.intervention-window-signals` | M12 | 4 | 90d |
| `flink.intervention-deltas` | M12b | 4 | 90d |
| `clinical.state-change-events` | M13 | 4 | 90d |

## E2E Testing

The 3-patient 14-day E2E test validates all modules with clinically realistic data:

| Patient | Role | Key Features Tested |
|---------|------|-------------------|
| **Rajesh Kumar** | Deteriorator | HTN escalation, triple-whammy AKI, DKA risk, engagement collapse, CKM amplification |
| **Priya Sharma** | Improver | BP control improving, positive reinforcement, ARB initiation tracking |
| **Amit Patel** | Edge Case | Masked HTN, non-dipper, acute surge, no false CID-02 |

```bash
# Run full E2E
make e2e

# Or step by step
python3 e2e_14day_generator.py --start-date 2026-04-09
python3 scripts/flink_e2e_3patient_14day.py --process-wait 50 --consume-timeout 20
```

**Expected results:** 11/11 hard PASS, 18/25 total (7 soft failures are timer-dependent modules that need wall-clock time).

## Service Ports

| Service | Port | URL |
|---------|------|-----|
| Flink Web UI (E2E) | 8181 | http://localhost:8181 |
| Flink Web UI (Prod) | 8081 | http://localhost:8081 |
| Flink RPC | 6123 | - |
| Kafka UI | 9080 | http://localhost:9080 |
| Prometheus | 9090 | http://localhost:9090 |
| Grafana | 3000 | http://localhost:3000 |
| Flink Metrics (Prometheus) | 9249 | http://localhost:9249/metrics |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BOOTSTRAP_SERVERS` | `kafka-lite:29092` | Kafka broker addresses |
| `MODULE13_ENABLED` | `true` | Master kill-switch for M13 |
| `MODULE13_DRY_RUN` | `false` | Log state changes without emitting |
| `MODULE13_KB20_WRITEBACK_ENABLED` | `false` | Write M13 state back to KB-20 |
| `MODULE13_PERSONALIZED_TARGETS_ENABLED` | `false` | Use KB-20 personalised thresholds |
| `USE_GOOGLE_HEALTHCARE_API` | `false` | FHIR store integration |

### Flink Configuration

Production defaults in `config/flink-conf.yaml`:
- State backend: RocksDB
- Checkpointing: 60s, exactly-once
- Parallelism: 8
- TaskManager memory: 6 GB
- TaskManager slots: 4

## Performance

- Event processing latency: < 500ms end-to-end
- Throughput: 100,000 events/second
- State size: up to 10 GB per TaskManager
- Checkpoint interval: 30-60 seconds
- M7 output: 1:1 with BP input (one metric per reading)
- M8 suppression: 4h HALT window, 72h PAUSE/SOFT_FLAG window
- M13 snapshot rotation: 7-day event-time intervals
