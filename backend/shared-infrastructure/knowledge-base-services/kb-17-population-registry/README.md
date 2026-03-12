# KB-17 Population Registry Service

A comprehensive disease registry management service providing patient population tracking, criteria-based enrollment, risk stratification, and care gap integration for chronic disease management.

## Features

- **8 Pre-configured Disease Registries**: Diabetes, Hypertension, Heart Failure, CKD, COPD, Pregnancy, Opioid Use, Anticoagulation
- **Kafka-driven Auto-enrollment**: Real-time patient enrollment based on clinical events
- **Criteria-based Evaluation**: Flexible AND/OR logic for eligibility determination
- **Risk Stratification**: Rules-based and score-based patient tiering (LOW, MODERATE, HIGH, CRITICAL)
- **KB Service Integration**: Connects with KB-2, KB-8, KB-9, KB-14 for comprehensive patient context

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    KB-17 Population Registry                        │
├─────────────────────────────────────────────────────────────────────┤
│  Kafka Consumer  →  Criteria Engine  →  Enrollment Store           │
│       ↓                   ↓                    ↓                    │
│  Clinical Events    Registry Defs      Risk Stratification         │
│       ↓                   ↓                    ↓                    │
│  Auto-Enrollment    ICD-10/LOINC         Task Creation             │
│       ↓                   ↓                    ↓                    │
│  Event Producer     Care Gap Sync        KB-14 Integration         │
└─────────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Using Docker Compose

```bash
# Start all services
make docker-run

# Start with Kafka for event-driven enrollment
make docker-run-kafka

# Check health
make health
```

### Local Development

```bash
# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgres://kb17user:kb17password@localhost:5439/kb_population_registry?sslmode=disable"
export REDIS_URL="redis://localhost:6389/0"

# Start infrastructure
docker-compose up -d postgres redis

# Run the service
make run

# Run with hot reload
make run-dev
```

## API Endpoints

### Registry Management
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/registries` | List all registries |
| GET | `/api/v1/registries/:code` | Get specific registry |
| POST | `/api/v1/registries` | Create custom registry |
| GET | `/api/v1/registries/:code/patients` | List registry patients |

### Enrollment Management
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/enrollments` | List enrollments |
| POST | `/api/v1/enrollments` | Create enrollment |
| GET | `/api/v1/enrollments/:id` | Get enrollment |
| PUT | `/api/v1/enrollments/:id` | Update enrollment |
| DELETE | `/api/v1/enrollments/:id` | Disenroll patient |
| POST | `/api/v1/enrollments/bulk` | Bulk enrollment |

### Patient-centric
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/patients/:id/registries` | Patient's registries |
| GET | `/api/v1/patients/:id/enrollment/:code` | Specific enrollment |
| POST | `/api/v1/evaluate` | Evaluate eligibility |

### Analytics
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/stats` | All registry stats |
| GET | `/api/v1/stats/:code` | Registry-specific stats |
| GET | `/api/v1/high-risk` | High-risk patients |
| GET | `/api/v1/care-gaps` | Patients with care gaps |

### Events
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/events` | Process clinical event |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8017 | HTTP server port |
| `ENVIRONMENT` | development | Environment (development/production) |
| `LOG_LEVEL` | info | Logging level |
| `DATABASE_URL` | - | PostgreSQL connection URL |
| `REDIS_URL` | redis://localhost:6379/0 | Redis connection URL |
| `KAFKA_BROKERS` | localhost:9092 | Kafka broker addresses |
| `KAFKA_ENABLED` | true | Enable Kafka consumer |
| `KB2_URL` | http://localhost:8082 | KB-2 Clinical Context URL |
| `KB8_URL` | http://localhost:8080 | KB-8 Calculator URL |
| `KB9_URL` | http://localhost:8089 | KB-9 Care Gaps URL |
| `KB14_URL` | http://localhost:8091 | KB-14 Care Navigator URL |

## KB Service Integration

KB-17 integrates with the following Knowledge Base services:

| Service | Port | Container Name | Purpose |
|---------|------|----------------|---------|
| KB-2 Clinical Context | 8082 | kb-2-clinical-context | Patient clinical data retrieval |
| KB-8 Calculator | 8080 | kb-8-calculator-service | Risk score calculations (eGFR, ASCVD, etc.) |
| KB-9 Care Gaps | 8089 | kb-9-care-gaps | Care gap identification |
| KB-14 Care Navigator | 8091 | kb-14-care-navigator | Task creation for enrollment actions |

### Running with Other KB Services

To connect to other KB services running in Docker:

```bash
# First, ensure other KB services are running and their networks exist
docker network ls | grep -E "kb-network|kb9-network|kb-14-network"

# Start KB-17 (will connect to external networks)
docker-compose up -d
```

If external networks don't exist, create them or start the dependent KB services first:
```bash
# Start main KB services (creates kb-network)
cd ../
docker-compose up -d

# Start KB-9 (creates kb9-network)
cd kb-9-care-gaps && docker-compose up -d

# Start KB-14 (creates kb-14-network)
cd ../kb-14-care-navigator && docker-compose up -d

# Now start KB-17
cd ../kb-17-population-registry && docker-compose up -d
```

## Registry Codes

| Code | Name | Description |
|------|------|-------------|
| `DIABETES` | Diabetes Mellitus | Type 1 & Type 2 diabetes |
| `HYPERTENSION` | Hypertension | Essential and secondary hypertension |
| `HEART_FAILURE` | Heart Failure | CHF and systolic/diastolic HF |
| `CKD` | Chronic Kidney Disease | CKD stages 1-5 |
| `COPD` | COPD | Chronic obstructive pulmonary disease |
| `PREGNANCY` | Pregnancy | High-risk pregnancy management |
| `OPIOID` | Opioid Use | Opioid use disorder |
| `ANTICOAGULATION` | Anticoagulation | Patients on anticoagulant therapy |

## Risk Tiers

- **LOW**: Minimal intervention required
- **MODERATE**: Regular monitoring needed
- **HIGH**: Intensive care management
- **CRITICAL**: Immediate attention required

## Kafka Topics

### Inbound (Clinical Events)
- `clinical.diagnosis` - New diagnosis events
- `clinical.lab-results` - Lab result events
- `clinical.medications` - Medication start/stop events
- `clinical.vitals` - Vital sign events

### Outbound (Registry Events)
- `registry.enrolled` - New enrollments
- `registry.disenrolled` - Disenrollments
- `registry.risk-changed` - Risk tier changes

## Development

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Tidy modules
make tidy
```

## License

Proprietary - CardioFit Clinical Synthesis Hub
