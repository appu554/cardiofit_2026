# KB-9 Care Gaps Service - Implementation Plan

## Executive Summary

KB-9 is a **Care Gaps Detection and Quality Measure Evaluation Service** that leverages the existing **vaidshala CQL infrastructure** to evaluate clinical quality measures against FHIR patient data.

| Attribute | Value |
|-----------|-------|
| **Port** | 8089 |
| **Language** | Go 1.21 |
| **Framework** | Gin |
| **Pattern** | Stateless (like KB-8) |
| **CQL Source** | `vaidshala/clinical-knowledge-core` |
| **CQL Engine** | `vaidshala/clinical-runtime-platform` |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        KB-9 Care Gaps Service                       │
│                            (Port 8089)                              │
├─────────────────────────────────────────────────────────────────────┤
│  REST API        │  GraphQL API      │  FHIR Operations            │
│  /api/v1/*       │  /graphql         │  /fhir/Measure/$care-gaps   │
├─────────────────────────────────────────────────────────────────────┤
│                      Care Gap Evaluator                             │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐           │
│  │ CQL Executor  │  │ Gap Detector  │  │ Intervention  │           │
│  │ (vaidshala)   │  │               │  │ Generator     │           │
│  └───────────────┘  └───────────────┘  └───────────────┘           │
├─────────────────────────────────────────────────────────────────────┤
│                      External Integrations                          │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐           │
│  │ FHIR Client   │  │ KB-7 Termin.  │  │ Redis Cache   │           │
│  │ (patient data)│  │ (valuesets)   │  │ (results)     │           │
│  └───────────────┘  └───────────────┘  └───────────────┘           │
└─────────────────────────────────────────────────────────────────────┘
```

---

## CQL Integration Strategy

### Using Existing Vaidshala Infrastructure

**Location**: `/Users/apoorvabk/Downloads/cardiofit/vaidshala/`

| Component | Path | Purpose |
|-----------|------|---------|
| CQL Libraries | `clinical-knowledge-core/tier-4-guidelines/` | CMS measures (CMS122, CMS165, etc.) |
| CQL Engine | `clinical-runtime-platform/engines/cql_engine.go` | Fact evaluation |
| Measure Engine | `clinical-runtime-platform/engines/measure_engine.go` | Gap detection |
| FHIRHelpers | `clinical-knowledge-core/tier-0-fhir/helpers/` | FHIR type conversion |
| ValueSets | KB-7 Terminology Service (port 8087) | Code system lookups |

### Execution Flow

```
1. Patient ID received
2. FHIR Client fetches patient data
3. Build ClinicalExecutionContext (frozen snapshot)
4. CQL Engine evaluates clinical facts
5. Measure Engine determines care gaps
6. Generate interventions and recommendations
7. Return CareGapReport
```

---

## Supported Measures (Phase 1)

| Measure | CMS ID | CQL Library | Status |
|---------|--------|-------------|--------|
| Diabetes HbA1c | CMS122 | `CMS122-DiabetesHbA1c.cql` | ✓ Exists |
| Blood Pressure | CMS165 | `CMS165-BloodPressure.cql` | ✓ Exists |
| Depression Screening | CMS2 | `CMS2-DepressionScreening.cql` | ✓ Exists |
| Diabetic Kidney | CMS134 | `CMS134-DiabetesKidney.cql` | ✓ Exists |
| India Diabetes | Custom | To create | Planned |
| India Hypertension | Custom | To create | Planned |

---

## Directory Structure

```
kb-9-care-gaps/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point
├── internal/
│   ├── api/
│   │   ├── server.go                  # Gin router setup
│   │   ├── handlers.go                # REST handlers
│   │   ├── graphql_handlers.go        # GraphQL resolvers
│   │   └── health_handlers.go         # Health endpoints
│   ├── caregaps/
│   │   ├── service.go                 # Core orchestration
│   │   ├── evaluator.go               # CQL execution wrapper
│   │   └── gap_detector.go            # Gap identification
│   ├── cql/
│   │   ├── executor.go                # Wraps vaidshala CQL engine
│   │   └── context_builder.go         # Builds ClinicalExecutionContext
│   ├── fhir/
│   │   ├── client.go                  # FHIR server client
│   │   └── queries.go                 # FHIR query builders
│   ├── measures/
│   │   ├── registry.go                # Measure definitions
│   │   ├── cms122.go                  # CMS122 adapter
│   │   ├── cms165.go                  # CMS165 adapter
│   │   └── cms2.go                    # CMS2 adapter
│   ├── deqm/
│   │   └── operations.go              # Da Vinci DEQM support
│   ├── models/
│   │   ├── care_gap.go                # Domain models
│   │   ├── measure_report.go          # FHIR MeasureReport
│   │   └── evidence.go                # CQL evidence
│   └── config/
│       └── config.go                  # Environment config
├── api/
│   └── schema.graphql                 # GraphQL schema (from kb9-schema.graphql)
├── Dockerfile
├── go.mod
└── README.md
```

---

## Implementation Phases

### Phase 1: Foundation (Day 1)
- [x] Create directory structure
- [ ] Initialize Go module
- [ ] Implement config loading
- [ ] Create main.go with Gin server
- [ ] Health endpoints (/health, /ready, /live, /metrics)

### Phase 2: FHIR Client (Day 2)
- [ ] FHIR client interface
- [ ] Patient data queries
- [ ] Observation queries (labs, vitals)
- [ ] Condition queries

### Phase 3: CQL Integration (Days 3-4)
- [ ] Import vaidshala CQL engine
- [ ] Build ClinicalExecutionContext
- [ ] Execute CQL libraries
- [ ] Map results to domain models

### Phase 4: Measure Implementation (Days 5-6)
- [ ] Measure registry
- [ ] CMS122 (Diabetes HbA1c) adapter
- [ ] CMS165 (Blood Pressure) adapter
- [ ] Gap detection logic

### Phase 5: API Implementation (Days 7-8)
- [ ] REST endpoints
- [ ] GraphQL resolvers
- [ ] DEQM operations ($care-gaps)
- [ ] Response formatting

### Phase 6: Testing & Docker (Days 9-10)
- [ ] Unit tests
- [ ] Integration tests
- [ ] Dockerfile
- [ ] Docker Compose integration
- [ ] Makefile targets

---

## Environment Variables

```bash
# Server
PORT=8089
ENVIRONMENT=development
LOG_LEVEL=info

# FHIR Server
FHIR_SERVER_URL=http://hapi-fhir:8080/fhir
FHIR_TIMEOUT=30s

# CQL (Vaidshala)
CQL_LIBRARY_PATH=../../vaidshala/clinical-knowledge-core
CQL_ENGINE_PATH=../../vaidshala/clinical-runtime-platform

# Terminology (KB-7)
TERMINOLOGY_URL=http://localhost:8087

# Caching
REDIS_URL=redis://kb-redis:6379/9
CACHE_TTL=5m

# GraphQL
ENABLE_PLAYGROUND=true
FEDERATION_ENABLED=true

# Metrics
METRICS_ENABLED=true
```

---

## Docker Compose Addition

```yaml
# Add to docker-compose.kb-only.yml
kb-9-care-gaps:
  build:
    context: ./kb-9-care-gaps
    dockerfile: Dockerfile
  ports:
    - "8089:8089"
  environment:
    - PORT=8089
    - ENVIRONMENT=development
    - FHIR_SERVER_URL=http://hapi-fhir:8080/fhir
    - TERMINOLOGY_URL=http://kb-7-terminology:8087
    - REDIS_URL=redis://kb-redis:6379/9
    - METRICS_ENABLED=true
    - ENABLE_PLAYGROUND=true
  depends_on:
    kb-redis:
      condition: service_healthy
  networks:
    - kb-network
  healthcheck:
    test: ["CMD", "wget", "--spider", "-q", "http://localhost:8089/health"]
    interval: 30s
    timeout: 5s
    retries: 3
```

---

## Makefile Targets

```makefile
# Add to root Makefile
build-kb-9:
	@echo "Building KB-9 Care Gaps service..."
	cd kb-9-care-gaps && go build -o bin/kb-9-care-gaps ./cmd/server

test-kb-9:
	@echo "Running tests for KB-9 Care Gaps..."
	cd kb-9-care-gaps && go test -v ./...

run-kb-9:
	@echo "Running KB-9 Care Gaps service..."
	cd kb-9-care-gaps && go run ./cmd/server
```

---

## API Endpoints

### REST API
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/care-gaps` | Get patient care gaps |
| POST | `/api/v1/measure/evaluate` | Evaluate single measure |
| POST | `/api/v1/measure/evaluate-population` | Population evaluation |
| GET | `/api/v1/measures` | List available measures |
| GET | `/api/v1/measures/{type}` | Get measure details |

### FHIR Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/fhir/Measure/$care-gaps` | Da Vinci DEQM $care-gaps |
| POST | `/fhir/Measure/{id}/$evaluate-measure` | FHIR $evaluate-measure |

### Health/Metrics
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Service health |
| GET | `/ready` | Readiness probe |
| GET | `/live` | Liveness probe |
| GET | `/metrics` | Prometheus metrics |

---

## Success Criteria

- [ ] Health endpoint responding on port 8089
- [ ] CMS122 (Diabetes HbA1c) gap detection working
- [ ] CMS165 (Blood Pressure) gap detection working
- [ ] GraphQL queries returning care gaps
- [ ] Da Vinci DEQM $care-gaps operation functional
- [ ] Integration with vaidshala CQL engine verified
- [ ] Docker image builds and runs
- [ ] <500ms latency for single patient evaluation

---

## Key Files to Modify

1. `Makefile` - Add build-kb-9, test-kb-9, run-kb-9 targets
2. `docker-compose.kb-only.yml` - Add KB-9 service
3. `integration_tests/Makefile` - Add KB-9 health check (port 8089)

---

## Timeline

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| Foundation | 1 day | Running service skeleton |
| FHIR Client | 1 day | Patient data queries |
| CQL Integration | 2 days | CQL execution working |
| Measures | 2 days | CMS122, CMS165 gaps |
| API | 2 days | REST + GraphQL |
| Testing/Docker | 2 days | Production ready |
| **Total** | **10 days** | **MVP KB-9** |
