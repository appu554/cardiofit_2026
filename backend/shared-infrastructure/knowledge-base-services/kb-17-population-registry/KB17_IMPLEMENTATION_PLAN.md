# KB-17 Population Registry Service - Implementation Plan

## Overview

KB-17 is a comprehensive disease registry management service with Kafka-driven auto-enrollment, criteria-based eligibility evaluation, and risk stratification. This plan outlines the implementation phases following established KB service patterns (KB-14, KB-9).

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        KB-17 Population Registry                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                │
│  │   Registry   │     │   Criteria   │     │   Patient    │                │
│  │ Definitions  │────▶│   Engine     │────▶│    Store     │                │
│  └──────────────┘     └──────────────┘     └──────────────┘                │
│         │                    ▲                    │                         │
│         ▼                    │                    ▼                         │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                │
│  │    Kafka     │────▶│   Event      │     │    Event     │───▶ KB-14     │
│  │   Consumer   │     │  Processor   │     │   Producer   │───▶ KB-18     │
│  └──────────────┘     └──────────────┘     └──────────────┘───▶ KB-9      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Technology Stack (Aligned with Existing KB Services)

| Component | Technology | Version |
|-----------|------------|---------|
| Language | Go | 1.22 |
| HTTP Framework | Gin | v1.9.1 |
| ORM | GORM | v1.25.7 |
| Database | PostgreSQL | 15+ |
| Cache | Redis | v9.4.0 |
| Message Queue | Kafka (Confluent) | - |
| Config | Viper | v1.18.2 |
| Logging | Logrus | v1.9.3 |
| Metrics | Prometheus | v1.18.0 |
| Testing | Testify | v1.8.4 |

## Project Structure

```
kb-17-population-registry/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point with graceful shutdown
├── internal/
│   ├── api/
│   │   ├── server.go              # HTTP server setup
│   │   ├── routes.go              # Route definitions
│   │   ├── handlers.go            # Request handlers
│   │   ├── middleware.go          # Auth, logging middleware
│   │   └── responses.go           # Response helpers
│   ├── cache/
│   │   └── redis.go               # Redis cache operations
│   ├── clients/
│   │   ├── kb2_client.go          # KB-2 Clinical Context client
│   │   ├── kb8_client.go          # KB-8 Calculator client
│   │   ├── kb9_client.go          # KB-9 Care Gaps client
│   │   └── kb14_client.go         # KB-14 Task Engine client
│   ├── config/
│   │   └── config.go              # Configuration management
│   ├── consumer/
│   │   └── kafka_consumer.go      # Kafka event consumer
│   ├── criteria/
│   │   └── engine.go              # Criteria evaluation engine
│   ├── database/
│   │   ├── connection.go          # DB connection management
│   │   └── repository.go          # Patient enrollment repository
│   ├── models/
│   │   ├── registry.go            # Registry definitions
│   │   ├── enrollment.go          # Patient enrollment models
│   │   ├── criteria.go            # Criteria models
│   │   ├── events.go              # Event models
│   │   └── responses.go           # API response models
│   ├── producer/
│   │   └── event_producer.go      # Registry event producer
│   ├── registry/
│   │   └── definitions.go         # Pre-configured registries
│   ├── services/
│   │   ├── enrollment_service.go  # Enrollment business logic
│   │   ├── evaluation_service.go  # Criteria evaluation service
│   │   └── analytics_service.go   # Analytics and stats
│   └── workers/
│       └── worker_manager.go      # Background workers
├── migrations/
│   ├── 001_create_registries.sql
│   ├── 002_create_enrollments.sql
│   └── 003_create_events.sql
├── tests/
│   ├── criteria_test.go
│   ├── consumer_test.go
│   ├── store_test.go
│   └── server_test.go
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## Implementation Phases

### Phase 1: Foundation (Core Types & Models)
**Duration**: Step 1
**Files**:
- `internal/models/registry.go`
- `internal/models/enrollment.go`
- `internal/models/criteria.go`
- `internal/models/events.go`

**Deliverables**:
1. Registry type definitions (RegistryCode, RegistryCategory)
2. Enrollment status types (EnrollmentStatus, EnrollmentSource)
3. Risk tier definitions (RiskTier: LOW, MODERATE, HIGH, CRITICAL)
4. Criteria types (CriteriaType, CriteriaOperator)
5. Event types for Kafka (EventType constants)
6. GORM models with JSONB support for metrics/metadata

**Key Types**:
```go
// Registry codes for pre-configured registries
type RegistryCode string
const (
    RegistryDiabetes      RegistryCode = "DIABETES"
    RegistryHypertension  RegistryCode = "HYPERTENSION"
    RegistryHeartFailure  RegistryCode = "HEART_FAILURE"
    RegistryCKD           RegistryCode = "CKD"
    RegistryCOPD          RegistryCode = "COPD"
    RegistryPregnancy     RegistryCode = "PREGNANCY"
    RegistryOpioidUse     RegistryCode = "OPIOID_USE"
    RegistryAnticoagulation RegistryCode = "ANTICOAGULATION"
)

// Enrollment status
type EnrollmentStatus string
const (
    EnrollmentStatusActive      EnrollmentStatus = "ACTIVE"
    EnrollmentStatusPending     EnrollmentStatus = "PENDING"
    EnrollmentStatusDisenrolled EnrollmentStatus = "DISENROLLED"
    EnrollmentStatusSuspended   EnrollmentStatus = "SUSPENDED"
)

// Risk tiers
type RiskTier string
const (
    RiskTierLow      RiskTier = "LOW"
    RiskTierModerate RiskTier = "MODERATE"
    RiskTierHigh     RiskTier = "HIGH"
    RiskTierCritical RiskTier = "CRITICAL"
)
```

---

### Phase 2: Registry Definitions
**Duration**: Step 2
**Files**:
- `internal/registry/definitions.go`
- `internal/models/registry_definition.go`

**Deliverables**:
1. Pre-configured registry definitions for 8 disease registries
2. Inclusion/exclusion criteria definitions
3. Risk stratification rules per registry
4. ICD-10 code mappings
5. Lab result thresholds
6. Medication-based triggers

**Registry Configuration Table**:
| Registry | ICD-10 Codes | Key Labs | Risk Stratification |
|----------|--------------|----------|---------------------|
| Diabetes | E10.*, E11.*, E13.* | HbA1c, FPG | HbA1c thresholds |
| Hypertension | I10, I11.*, I12.*, I13.* | BP | BP thresholds |
| Heart Failure | I50.*, I42.* | BNP, NT-proBNP | BNP/diagnosis-based |
| CKD | N18.* | eGFR, UACR, Cr | eGFR staging |
| COPD | J44.*, J43.9 | FEV1 | GOLD staging |
| Pregnancy | Z34.*, O* | HCG, GCT | Age, complications |
| Opioid Use | F11.* | UDS | ORT score |
| Anticoagulation | Medication-based | INR, eGFR | HAS-BLED score |

---

### Phase 3: Criteria Engine
**Duration**: Step 3
**Files**:
- `internal/criteria/engine.go`
- `internal/criteria/evaluator.go`
- `internal/criteria/risk_calculator.go`

**Deliverables**:
1. Criteria evaluation engine
2. Support for multiple criteria types:
   - DIAGNOSIS (ICD-10 code matching)
   - LAB_RESULT (value comparisons)
   - MEDICATION (RxNorm matching)
   - PROBLEM_LIST (active problems)
3. Operators: EQUALS, STARTS_WITH, IN, GREATER_THAN, LESS_THAN, BETWEEN
4. AND/OR logic for complex criteria groups
5. Risk tier calculation based on registry rules

**Evaluation Result**:
```go
type CriteriaEvaluationResult struct {
    PatientID         string
    RegistryCode      RegistryCode
    MeetsInclusion    bool
    MeetsExclusion    bool
    Eligible          bool
    SuggestedRiskTier RiskTier
    MatchedCriteria   []string
    EvaluatedAt       time.Time
}
```

---

### Phase 4: Patient Store & Enrollment
**Duration**: Step 4
**Files**:
- `internal/database/connection.go`
- `internal/database/repository.go`
- `internal/services/enrollment_service.go`

**Deliverables**:
1. PostgreSQL database connection with GORM
2. Repository pattern for enrollments
3. CRUD operations for RegistryPatient
4. Bulk enrollment support
5. Query by patient, registry, status, risk tier
6. Enrollment history tracking
7. Metrics storage (JSONB)

**Database Schema**:
```sql
-- Registries table
CREATE TABLE registries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    auto_enroll BOOLEAN DEFAULT true,
    inclusion_criteria JSONB,
    exclusion_criteria JSONB,
    risk_stratification JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Patient enrollments table
CREATE TABLE registry_patients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    registry_code VARCHAR(50) NOT NULL,
    patient_id VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    enrollment_source VARCHAR(30) NOT NULL,
    risk_tier VARCHAR(20) DEFAULT 'MODERATE',
    metrics JSONB DEFAULT '{}',
    care_gaps TEXT[],
    enrolled_at TIMESTAMPTZ DEFAULT NOW(),
    disenrolled_at TIMESTAMPTZ,
    disenroll_reason TEXT,
    last_evaluated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(registry_code, patient_id)
);

-- Enrollment history
CREATE TABLE enrollment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL,
    action VARCHAR(30) NOT NULL,
    old_status VARCHAR(30),
    new_status VARCHAR(30),
    old_risk_tier VARCHAR(20),
    new_risk_tier VARCHAR(20),
    reason TEXT,
    actor_id VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

### Phase 5: Kafka Consumer (Auto-Enrollment)
**Duration**: Step 5
**Files**:
- `internal/consumer/kafka_consumer.go`
- `internal/consumer/event_handler.go`

**Deliverables**:
1. Kafka consumer for clinical events
2. Event types supported:
   - `diagnosis.created` - ICD-10 diagnosis triggers
   - `lab.result.created` - Lab result triggers
   - `medication.started` - Medication triggers
   - `problem.added` - Problem list triggers
3. Auto-enrollment flow:
   - Receive event → Parse patient context
   - Evaluate against all registries
   - Enroll if criteria met
   - Calculate risk tier
   - Produce enrollment event

**Event Processing Flow**:
```
Kafka Event → Consumer → Criteria Engine → Enrollment Service → Producer
                              ↓
                      KB-2 (Patient Context)
                      KB-8 (Risk Scores)
```

---

### Phase 6: Event Producer
**Duration**: Step 6
**Files**:
- `internal/producer/event_producer.go`

**Deliverables**:
1. Kafka producer for registry events
2. Event types produced:
   - `registry.enrolled` - New enrollment
   - `registry.disenrolled` - Patient disenrolled
   - `registry.risk_changed` - Risk tier change
   - `registry.care_gap_updated` - Care gap changes
3. Event routing to downstream services:
   - KB-14: Task creation for new enrollments
   - KB-18: Governance enforcement inputs
   - KB-9: Care gap updates

---

### Phase 7: Integration Clients
**Duration**: Step 7
**Files**:
- `internal/clients/kb2_client.go`
- `internal/clients/kb8_client.go`
- `internal/clients/kb9_client.go`
- `internal/clients/kb14_client.go`

**Deliverables**:
1. HTTP clients for upstream services:
   - **KB-2**: Get patient clinical context
   - **KB-8**: Get risk scores (HAS-BLED, ASCVD, etc.)
2. HTTP clients for downstream services:
   - **KB-9**: Update care gaps
   - **KB-14**: Create tasks for new enrollments
3. Circuit breaker pattern for resilience
4. Retry logic with exponential backoff

---

### Phase 8: HTTP API Server
**Duration**: Step 8
**Files**:
- `internal/api/server.go`
- `internal/api/routes.go`
- `internal/api/handlers.go`
- `internal/api/middleware.go`

**Deliverables**:

**Registry Endpoints**:
```
GET    /api/v1/registries              # List all registries
GET    /api/v1/registries/{code}       # Get registry definition
POST   /api/v1/registries              # Create custom registry
GET    /api/v1/registries/{code}/patients  # Get registry patients
```

**Enrollment Endpoints**:
```
GET    /api/v1/enrollments             # Query enrollments
POST   /api/v1/enrollments             # Manual enrollment
GET    /api/v1/enrollments/{id}        # Get enrollment details
PUT    /api/v1/enrollments/{id}        # Update enrollment
DELETE /api/v1/enrollments/{id}        # Disenroll
POST   /api/v1/enrollments/bulk        # Bulk enrollment
```

**Patient-Centric Endpoints**:
```
GET    /api/v1/patients/{id}/registries           # Patient's registries
GET    /api/v1/patients/{id}/enrollment/{code}    # Specific enrollment
```

**Criteria Evaluation**:
```
POST   /api/v1/evaluate                # Evaluate patient eligibility
```

**Analytics**:
```
GET    /api/v1/stats                   # All registry statistics
GET    /api/v1/stats/{code}            # Registry-specific stats
GET    /api/v1/high-risk               # High-risk patients
GET    /api/v1/care-gaps               # Patients with care gaps
```

**Events**:
```
POST   /api/v1/events                  # Process clinical event
```

---

### Phase 9: Configuration & Entry Point
**Duration**: Step 9
**Files**:
- `internal/config/config.go`
- `cmd/server/main.go`

**Deliverables**:
1. Environment-based configuration
2. Graceful shutdown handling
3. Background worker management
4. Health check endpoint
5. Prometheus metrics endpoint

**Environment Variables**:
```
KB17_PORT=8017
KB17_KAFKA_BROKERS=localhost:9092
KB17_KAFKA_GROUP_ID=kb17-population-registry
KB17_DATABASE_URL=postgres://...
KB17_REDIS_URL=redis://localhost:6379
KB17_KB2_URL=http://localhost:8002
KB17_KB8_URL=http://localhost:8008
KB17_KB9_URL=http://localhost:8009
KB17_KB14_URL=http://localhost:8014
```

---

### Phase 10: Testing
**Duration**: Step 10
**Files**:
- `tests/criteria_test.go`
- `tests/consumer_test.go`
- `tests/store_test.go`
- `tests/server_test.go`

**Test Coverage**:
1. **Unit Tests**:
   - Criteria evaluation logic
   - Risk tier calculation
   - Model validation
2. **Integration Tests**:
   - Database operations
   - Kafka consumer/producer
   - HTTP endpoints
3. **E2E Tests**:
   - Complete enrollment flow
   - Auto-enrollment via Kafka

---

### Phase 11: Docker & Deployment
**Duration**: Step 11
**Files**:
- `Dockerfile`
- `docker-compose.yml`
- `Makefile`
- `go.mod`

**Deliverables**:
1. Multi-stage Dockerfile
2. Docker Compose with dependencies
3. Makefile targets:
   - `make build`
   - `make run`
   - `make test`
   - `make docker-build`
   - `make docker-run`

---

## Metrics & Monitoring

**Prometheus Metrics**:
- `kb17_enrollments_total{registry,status}` - Total enrollments
- `kb17_disenrollments_total{registry}` - Total disenrollments
- `kb17_events_processed_total{event_type}` - Kafka events processed
- `kb17_criteria_evaluations_total{registry}` - Criteria evaluations
- `kb17_high_risk_patients{registry}` - Current high/critical risk count
- `kb17_api_request_duration_seconds` - API latency histogram

---

## Dependencies

### Upstream (Consumes From)
| Service | Purpose | Required |
|---------|---------|----------|
| Kafka | Clinical events | Yes |
| KB-2 | Patient clinical context | Yes |
| KB-8 | Risk score calculation | Optional |
| KB-11 | Population attribution | Optional |

### Downstream (Produces To)
| Service | Purpose | Events |
|---------|---------|--------|
| KB-9 | Care gap updates | `registry.care_gap_updated` |
| KB-14 | Task creation | `registry.enrolled` |
| KB-15 | Patient outreach | `registry.enrolled`, `registry.risk_changed` |
| KB-18 | Governance | All events |

---

## Execution Order

| Step | Phase | Files | Est. LOC |
|------|-------|-------|----------|
| 1 | Core Models | 4 files | ~600 |
| 2 | Registry Definitions | 2 files | ~400 |
| 3 | Criteria Engine | 3 files | ~500 |
| 4 | Database & Store | 3 files | ~400 |
| 5 | Kafka Consumer | 2 files | ~300 |
| 6 | Event Producer | 1 file | ~200 |
| 7 | Integration Clients | 4 files | ~400 |
| 8 | HTTP API Server | 4 files | ~600 |
| 9 | Config & Main | 2 files | ~250 |
| 10 | Tests | 4 files | ~500 |
| 11 | Docker & Build | 4 files | ~150 |

**Total Estimated**: ~4,300 LOC

---

## Success Criteria

1. ✅ All 8 pre-configured registries operational
2. ✅ Kafka auto-enrollment functional
3. ✅ Risk stratification working per registry rules
4. ✅ All API endpoints responding correctly
5. ✅ Integration with KB-2, KB-8, KB-9, KB-14 verified
6. ✅ Event production to downstream services working
7. ✅ Test coverage > 80%
8. ✅ Docker deployment successful

---

## Ready for Execution

Proceed with implementation starting from **Phase 1: Foundation (Core Types & Models)**.
