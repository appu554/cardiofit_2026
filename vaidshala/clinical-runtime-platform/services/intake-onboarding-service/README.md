# Intake-Onboarding Service

Patient enrollment, safety screening, biweekly check-in, and pharmacist review queue for the Vaidshala Clinical Runtime Platform. Manages the full intake lifecycle from initial enrollment through safety evaluation, slot collection, pharmacist review, and ongoing M0-CI check-in cycles.

**Port**: `8141`
**Gateway prefix**: `/api/v1/intake` (stripped before forwarding)

## Architecture

```
Channels                     Core Engine                      Downstream
────────                     ───────────                      ──────────
Flutter App      ─┐
WhatsApp Bot     ─┤    ┌──────────┐   ┌────────────┐
ASHA Tablet      ─┤───→│ Flow     │──→│ Safety     │──→ Slot Collection ──→ Review Queue
ABDM/ABHA        ─┤    │ Engine   │   │ Engine     │         │                  │
                  │    └──────────┘   └────────────┘         │                  │
                  │                                          ▼                  ▼
                  │                                     FHIR Store         ENROLLED
                  │                                     Kafka Events       (→ V-MCU)
                  │
                  └─── Check-in (M0-CI biweekly) ──→ Trajectory ──→ KB-20
```

### Enrollment State Machine
```
INTAKE_IN_PROGRESS → INTAKE_COMPLETED → PENDING_REVIEW → ENROLLED
                          ↑                    │
                          └── CLARIFICATION ───┘
```

### Check-in State Machine (7 states)
```
CS1_SCHEDULED → CS2_REMINDED → CS3_COLLECTING → CS4_PAUSED
                                      │              │
                                      ▼              │
                                CS5_SCORING ←────────┘
                                      │
                                      ▼
                                CS6_DISPATCHED → CS7_CLOSED
```

## Prerequisites

| Dependency | Default | Required |
|------------|---------|----------|
| PostgreSQL | `localhost:5433` | Yes |
| Redis | `localhost:6380` | Yes |
| Kafka | `localhost:9092` | No (events skipped if unavailable) |
| FHIR Store | Disabled | No (`FHIR_ENABLED=true` to activate) |
| Ingestion Service | `localhost:8140` | For HPI data forwarding |

## Quick Start

```bash
cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service

# 1. Start infrastructure
cd ../../.. && make run-kb-docker && cd -

# 2. Run database migrations (in order)
psql "$DATABASE_URL" -f migrations/001_init.sql
psql "$DATABASE_URL" -f migrations/002_asha_abdm.sql
psql "$DATABASE_URL" -f migrations/003_checkin.sql

# 3. Build and run
go build -o intake-service ./cmd/intake
./intake-service
```

The service starts on port **8141** by default.

## Environment Variables

### Server
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8141` | HTTP listen port |
| `ENVIRONMENT` | `development` | `development` or `production` |
| `LOG_LEVEL` | `info` | Logging level |

### PostgreSQL
| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://intake_user:intake_password@localhost:5433/intake_service?sslmode=disable` | Connection string |
| `DATABASE_MAX_CONNECTIONS` | `10` | Connection pool size |
| `DATABASE_CONN_MAX_LIFETIME_MINUTES` | `30` | Max connection age |

### Redis
| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_URL` | `redis://localhost:6380` | Redis connection URL |
| `REDIS_PASSWORD` | _(empty)_ | Redis password |
| `REDIS_DB` | `3` | Redis database number |

### Kafka
| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated broker list |
| `KAFKA_GROUP_ID` | `intake-onboarding-service` | Consumer group ID |

### FHIR Store (Google Healthcare API)
| Variable | Default | Description |
|----------|---------|-------------|
| `FHIR_ENABLED` | `false` | Enable FHIR Store writes |
| `FHIR_PROJECT_ID` | _(empty)_ | GCP project ID |
| `FHIR_LOCATION` | _(empty)_ | GCP region |
| `FHIR_DATASET_ID` | _(empty)_ | Healthcare dataset |
| `FHIR_STORE_ID` | _(empty)_ | FHIR store name |
| `GOOGLE_APPLICATION_CREDENTIALS` | _(empty)_ | Service account key path |

### WhatsApp Business API
| Variable | Default | Description |
|----------|---------|-------------|
| `WHATSAPP_PHONE_NUMBER_ID` | _(empty)_ | WhatsApp Business phone number |
| `WHATSAPP_ACCESS_TOKEN` | _(empty)_ | Meta API access token |
| `WHATSAPP_APP_SECRET` | _(empty)_ | App secret for webhook verification |
| `WHATSAPP_VERIFY_TOKEN` | `cardiofit-intake-verify` | Webhook verify token |

### ABDM (Ayushman Bharat Digital Mission)
| Variable | Default | Description |
|----------|---------|-------------|
| `ABDM_BASE_URL` | `https://abdm.gov.in` | ABDM API base URL |
| `ABDM_CLIENT_ID` | _(empty)_ | ABDM client ID |
| `ABDM_CLIENT_SECRET` | _(empty)_ | ABDM client secret |
| `ABDM_SANDBOX` | `true` | Use ABDM sandbox environment |

### Inter-Service
| Variable | Default | Description |
|----------|---------|-------------|
| `INGESTION_SERVICE_URL` | `http://localhost:8140` | Ingestion service base URL |

### OpenTelemetry
| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | OTLP HTTP endpoint |
| `OTEL_SERVICE_NAME` | `intake-onboarding-service` | Service name in traces |

## API Endpoints

### Infrastructure
| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe |
| GET | `/startupz` | Startup probe |
| GET | `/metrics` | Prometheus metrics |

### Enrollment & Safety (Phase 3 — Live)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/fhir/Patient/$enroll` | Start patient enrollment |
| POST | `/fhir/Patient/:id/$evaluate-safety` | Run safety screening (11 hard stops, 8 soft flags) |
| POST | `/fhir/Encounter/:id/$fill-slot` | Fill an intake slot |

### Pharmacist Review Queue (Phase 5 — Live)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/fhir/Encounter/:encounterID/$submit-review` | Submit completed intake for review |
| POST | `/fhir/ReviewEntry/:entryID/$approve` | Approve → transitions to ENROLLED |
| POST | `/fhir/ReviewEntry/:entryID/$request-clarification` | Request clarification → reverts to INTAKE_IN_PROGRESS |
| POST | `/fhir/ReviewEntry/:entryID/$escalate` | Escalate to senior pharmacist |

### Biweekly Check-in (Phase 5 — Live)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/fhir/Patient/:id/$checkin` | Start a new check-in session |
| POST | `/fhir/CheckinSession/:id/$checkin-slot` | Fill a check-in slot value |

### FHIR CRUD (Stubs — Phase 6)
| Method | Path | Description |
|--------|------|-------------|
| POST/GET/PUT | `/fhir/Patient`, `/fhir/Observation`, `/fhir/Encounter`, `/fhir/MedicationStatement`, `/fhir/Condition` | Standard FHIR CRUD _(stub — 501)_ |

### Stub Operations (Phase 6)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/fhir/Patient/:id/$verify-otp` | OTP verification _(stub)_ |
| POST | `/fhir/Patient/:id/$link-abha` | ABHA linking _(stub)_ |
| POST | `/fhir/Patient/:id/$register-co-enrollee` | Co-enrollee registration _(stub)_ |

### Channel Adapters
| Method | Path | Description |
|--------|------|-------------|
| GET | `/webhook/whatsapp` | WhatsApp webhook verification (Meta challenge) |
| POST | `/webhook/whatsapp` | WhatsApp incoming message handler |
| POST | `/channel/asha/submit` | ASHA tablet batch submission |
| GET | `/channel/asha/sync/:deviceId` | ASHA tablet sync status |

## Accessing via API Gateway

All endpoints are accessible through the API Gateway (port `8000`) under the `/api/v1/intake` prefix:

```bash
# Direct access (development)
curl -X POST http://localhost:8141/fhir/Patient/\$enroll -d '...'

# Via API Gateway (production)
curl -X POST http://localhost:8000/api/v1/intake/fhir/Patient/\$enroll \
  -H "Authorization: Bearer <token>" \
  -d '...'
```

The gateway strips the `/api/v1/intake` prefix before forwarding and injects `X-User-ID`, `X-User-Role`, and `X-User-Permissions` headers.

### Required Permissions (RBAC)
| Permission | Endpoints |
|------------|-----------|
| `intake:enroll` | `$enroll`, `$verify-otp`, `$link-abha` |
| `intake:write` | All other POST/PUT endpoints |
| `intake:read` | All GET endpoints |
| `intake:review` | `$approve`, `$escalate`, `$request-clarification` |
| `intake:checkin` | `$checkin`, `$checkin-slot` |
| `safety:read` | `DetectedIssue` queries |

## Safety Engine

The safety engine evaluates patient data at enrollment time and produces:
- **11 hard stops** — conditions that block enrollment (e.g., eGFR < 15, acute MI < 6 weeks)
- **8 soft flags** — conditions requiring pharmacist attention (e.g., polypharmacy ≥ 5 meds, elderly > 75)

Hard stops result in `HALT`; soft flags contribute to risk stratification for the review queue.

## Review Queue Risk Stratification

| Risk Level | Criteria |
|------------|----------|
| **HIGH** | Any hard stop, ≥ 3 soft flags, eGFR < 30, or age ≥ 80 with ≥ 5 medications |
| **MEDIUM** | Age > 75, polypharmacy (≥ 5 meds), or 1-2 soft flags |
| **LOW** | All others |

## Check-in Slots

Each biweekly check-in session collects 12 slots (8 required):

| Slot | Domain | Required |
|------|--------|----------|
| `weight_kg` | vitals | Yes |
| `systolic_bp` | vitals | Yes |
| `diastolic_bp` | vitals | Yes |
| `heart_rate` | vitals | Yes |
| `fasting_glucose` | labs | Yes |
| `hba1c` | labs | No |
| `creatinine` | labs | No |
| `potassium` | labs | No |
| `symptom_score` | symptoms | Yes |
| `med_adherence_pct` | adherence | Yes |
| `exercise_minutes` | lifestyle | Yes |
| `sleep_hours` | lifestyle | Yes |

## Trajectory Computation

After check-in completion, the trajectory computer produces one of four signals:

| Signal | Meaning |
|--------|---------|
| `STABLE` | ≤ 25% of scored slots worsened |
| `FRAGILE` | 25-50% of scored slots worsened |
| `FAILURE` | > 50% worsened, or 2+ consecutive FRAGILE |
| `DISENGAGE` | ≥ 3 required slots missing |

## Flow Engine

The intake flow graph is loaded from `configs/flows/intake_full.yaml`. If the graph file is missing, the service runs in stub mode (flow operations return defaults).

## Database Migrations

```bash
psql "$DATABASE_URL" -f migrations/001_init.sql         # enrollments, slots, events
psql "$DATABASE_URL" -f migrations/002_asha_abdm.sql     # ASHA offline queue, ABDM consents
psql "$DATABASE_URL" -f migrations/003_checkin.sql       # checkin_sessions, checkin_slot_events
```

## Observability

- **Prometheus metrics**: Exposed at `/metrics`
- **OpenTelemetry tracing**: OTLP HTTP exporter, 10% parent-based sampling, Gin middleware
- **Structured logging**: zap JSON logger

## Testing

```bash
go test ./...
```
