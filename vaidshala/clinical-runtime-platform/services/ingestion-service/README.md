# Ingestion Service

Multi-source clinical data ingestion pipeline for the Vaidshala Clinical Runtime Platform. Receives observations from FHIR endpoints, lab systems, EHR integrations, wearable devices, ABDM Health Information, and patient-reported data — normalises, validates, maps to FHIR R4, and publishes to Kafka.

**Port**: `8140`
**Gateway prefix**: `/api/v1/ingest` (stripped before forwarding)

## Architecture

```
Sources                          Pipeline                         Sinks
───────                          ────────                         ─────
FHIR Observation ─┐
Lab Webhooks     ─┤              ┌───────────┐   ┌──────────┐
EHR (HL7v2/FHIR)─┤─→ Adapter ──→│ Normalizer │──→│ Validator│──→ FHIR Mapper ──→ Topic Router
Devices          ─┤              └───────────┘   └──────────┘         │
Wearables        ─┤                                                   ├──→ FHIR Store (Google Healthcare API)
App Check-in     ─┤                                                   ├──→ Kafka (per-topic routing)
WhatsApp NLU     ─┤                                                   └──→ DLQ (PostgreSQL)
ABDM Data Push   ─┤
HPI (internal)   ─┘
```

## Prerequisites

| Dependency | Default | Required |
|------------|---------|----------|
| PostgreSQL | `localhost:5433` | Yes |
| Redis | `localhost:6380` | Yes |
| Kafka | `localhost:9092` | No (publishes skipped if unavailable) |
| FHIR Store | Disabled | No (`FHIR_ENABLED=true` to activate) |

## Quick Start

```bash
cd vaidshala/clinical-runtime-platform/services/ingestion-service

# 1. Start infrastructure (from shared KB docker-compose)
cd ../../.. && make run-kb-docker && cd -

# 2. Run database migrations
psql "$DATABASE_URL" -f migrations/001_init.sql
psql "$DATABASE_URL" -f migrations/002_lab_adapters.sql

# 3. Build and run
go build -o ingestion-service ./cmd/ingestion
./ingestion-service
```

The service starts on port **8140** by default.

## Environment Variables

### Server
| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8140` | HTTP listen port |
| `ENVIRONMENT` | `development` | `development` or `production` |
| `LOG_LEVEL` | `info` | Logging level |

### PostgreSQL
| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://ingestion_user:ingestion_password@localhost:5433/ingestion_service?sslmode=disable` | Connection string |
| `DATABASE_MAX_CONNECTIONS` | `10` | Connection pool size |
| `DATABASE_CONN_MAX_LIFETIME_MINUTES` | `30` | Max connection age |

### Redis
| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_URL` | `redis://localhost:6380` | Redis connection URL |
| `REDIS_PASSWORD` | _(empty)_ | Redis password |
| `REDIS_DB` | `2` | Redis database number |

### Kafka
| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated broker list |
| `KAFKA_GROUP_ID` | `ingestion-service` | Consumer group ID |

### FHIR Store (Google Healthcare API)
| Variable | Default | Description |
|----------|---------|-------------|
| `FHIR_ENABLED` | `false` | Enable FHIR Store writes |
| `FHIR_PROJECT_ID` | _(empty)_ | GCP project ID |
| `FHIR_LOCATION` | _(empty)_ | GCP region |
| `FHIR_DATASET_ID` | _(empty)_ | Healthcare dataset |
| `FHIR_STORE_ID` | _(empty)_ | FHIR store name |
| `GOOGLE_APPLICATION_CREDENTIALS` | _(empty)_ | Service account key path |

### Lab API Keys
| Variable | Description |
|----------|-------------|
| `LAB_API_KEY_THYROCARE` | Thyrocare API key |
| `LAB_API_KEY_REDCLIFFE` | Redcliffe Labs API key |
| `LAB_API_KEY_SRL_AGILUS` | SRL/Agilus API key |
| `LAB_API_KEY_DR_LAL` | Dr Lal PathLabs API key |
| `LAB_API_KEY_METROPOLIS` | Metropolis Healthcare API key |
| `LAB_API_KEY_ORANGE_HEALTH` | Orange Health API key |

### OpenTelemetry
| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | OTLP HTTP endpoint |
| `OTEL_SERVICE_NAME` | `ingestion-service` | Service name in traces |

## API Endpoints

### Infrastructure
| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |
| GET | `/startupz` | Startup probe |
| GET | `/metrics` | Prometheus metrics |

### FHIR Inbound
| Method | Path | Description |
|--------|------|-------------|
| POST | `/fhir/Observation` | Ingest a FHIR-like observation |
| POST | `/fhir/DiagnosticReport` | _(stub — Phase 6)_ |
| POST | `/fhir/MedicationStatement` | _(stub — Phase 6)_ |
| POST | `/fhir` | FHIR Transaction Bundle _(stub)_ |

### DLQ (Dead Letter Queue)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/fhir/OperationOutcome` | List pending DLQ entries |
| POST | `/fhir/OperationOutcome/:id/$replay` | Replay a DLQ entry |
| GET | `/admin/dlq` | List DLQ entries (filtered) |
| GET | `/admin/dlq/:id` | Get single DLQ entry |
| POST | `/admin/dlq/:id/$discard` | Discard a DLQ entry |
| GET | `/admin/dlq/$count` | Count DLQ entries by status |

### Source Adapters
| Method | Path | Description |
|--------|------|-------------|
| POST | `/labs/:labId` | Lab webhook (thyrocare, redcliffe, srl_agilus, dr_lal, metropolis, orange_health, generic_csv) |
| POST | `/ehr/hl7v2` | HL7v2 message ingest |
| POST | `/ehr/fhir` | FHIR passthrough from EHR |
| POST | `/devices` | IoT/medical device data |
| POST | `/wearables/:provider` | Wearable data (health_connect, ultrahuman, apple_health) |
| POST | `/app-checkin` | Patient self-report from Flutter app |
| POST | `/whatsapp` | NLU-parsed WhatsApp messages |
| POST | `/abdm/data-push` | ABDM Health Information push |

### Internal (Service-to-Service)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/internal/hpi` | HPI slot data from Intake service |

### Admin
| Method | Path | Description |
|--------|------|-------------|
| GET | `/$source-status` | Source connectivity status _(stub)_ |

## Accessing via API Gateway

All endpoints are accessible through the API Gateway (port `8000`) under the `/api/v1/ingest` prefix:

```bash
# Direct access (development)
curl -X POST http://localhost:8140/fhir/Observation -d '...'

# Via API Gateway (production)
curl -X POST http://localhost:8000/api/v1/ingest/fhir/Observation \
  -H "Authorization: Bearer <token>" \
  -d '...'
```

The gateway strips the `/api/v1/ingest` prefix before forwarding. It also injects `X-User-ID`, `X-User-Role`, and `X-User-Permissions` headers from the authenticated JWT.

### Required Permissions (RBAC)
| Permission | Endpoints |
|------------|-----------|
| `ingest:write` | All POST endpoints (fallback) |
| `ingest:lab` | `/labs/*` |
| `ingest:ehr` | `/ehr/*` |
| `ingest:abdm` | `/abdm/*` |
| `ingest:device` | `/devices`, `/wearables/*` |
| `ingest:admin` | `/$source-status`, `/fhir/OperationOutcome` |

## Wearable Providers

### Health Connect (`POST /wearables/health_connect`)
Accepts Android Health Connect records. Supports 10 record types: HeartRate, BloodPressure (split into systolic + diastolic), Steps, BloodGlucose, OxygenSaturation, BodyTemperature, Weight, Height, RespiratoryRate, RestingHeartRate. Quality score: **0.85**.

### Ultrahuman (`POST /wearables/ultrahuman`)
Accepts CGM aggregation payloads with raw glucose readings. Computes 7 metrics: TIR, TAR, TBR, CV, MAG, GMI, MeanGlucose. Minimum 12 readings required. Critical flag emitted when TBR > 4%.

### Apple HealthKit (`POST /wearables/apple_health`)
Accepts 12 HKQuantityType identifiers. Performs unit conversions: lbs→kg, °F→°C, mmol/L→mg/dL, kPa→mmHg. Quality score: **0.90**.

## Pipeline Stages

1. **Adapter** — source-specific parsing to `CanonicalObservation`
2. **Normalizer** — unit standardisation, timestamp alignment
3. **Validator** — range checks, completeness, plausibility
4. **FHIR Mapper** — maps to FHIR R4 Observation/DiagnosticReport/MedicationStatement
5. **Topic Router** — selects Kafka topic based on observation type and source
6. **DLQ** — failed observations land in PostgreSQL dead-letter queue for replay

## Observability

- **Prometheus metrics**: `messages_received_total`, `pipeline_duration_seconds`, `critical_values_total`, `dlq_messages_total`
- **OpenTelemetry tracing**: OTLP HTTP exporter, 10% parent-based sampling, Gin middleware auto-creates server spans
- **Structured logging**: zap JSON logger with correlation fields

## Database Migrations

```bash
psql "$DATABASE_URL" -f migrations/001_init.sql       # DLQ table, ingestion_events
psql "$DATABASE_URL" -f migrations/002_lab_adapters.sql # Lab webhook tracking
```

## Testing

```bash
go test ./...
```
