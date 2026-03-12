# KB-11 Population Health Engine - Implementation Plan

> **Generated**: Brainstorming Session (Revised with CTO/CMO Corrections)
> **Timeline**: 5 Weeks (Production-Ready Core)
> **Scope**: Population Intelligence Layer (Refined)
> **Data Source**: FHIR Store + KB-17 Integration (Read-Only)

---

## 🎯 North Star Definition

> **KB-11 answers population-level questions, NOT patient-level decisions.**

### In Scope ✅
- "Which cohorts are deteriorating?"
- "Which PCPs manage the highest-risk panels?"
- "What is the risk distribution across my population?"
- "20% of diabetics have open HbA1c gaps" (aggregate analytics)

### Out of Scope ❌
- "Should this patient be escalated?" → **KB-19** (Care Navigator)
- "Is this care compliant?" → **KB-13** (Quality Measures)
- "What care gap does this patient have?" → **KB-13** (source of truth)
- "What governance approval is needed?" → **KB-18** (Governance)

---

## System Boundary Contract

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         KB ECOSYSTEM BOUNDARIES                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  KB-17 (Population Registry)     KB-13 (Quality Measures)                  │
│  └─ SOURCE OF TRUTH: Patients    └─ SOURCE OF TRUTH: Care Gaps             │
│  └─ Enrollment authority         └─ Gap identification                     │
│                                                                             │
│  KB-18 (Governance)              KB-19 (Care Navigator)                    │
│  └─ SOURCE OF TRUTH: Approvals   └─ Patient-level decisions                │
│  └─ Risk model governance        └─ Escalation authority                   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    KB-11 (Population Health)                         │   │
│  │                                                                       │   │
│  │  ROLE: Population Intelligence Layer                                 │   │
│  │  • CONSUMES patient data from KB-17/FHIR (read-only)                │   │
│  │  • CONSUMES care gaps from KB-13 (aggregation only)                 │   │
│  │  • EMITS risk scores governed by KB-18                              │   │
│  │  • PROVIDES population analytics, cohorts, stratification           │   │
│  │                                                                       │   │
│  │  NOT A: Registry | Gap definer | Decision engine | Care manager     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Executive Summary

KB-11 is the **population intelligence layer** providing risk stratification, cohort management, and population analytics. It operates as a **read-through cache** consuming data from authoritative sources (FHIR Store, KB-17, KB-13) and producing governed risk scores via KB-18.

### Target Metrics
| Metric | Target |
|--------|--------|
| **Total LOC** | ~3,500 (refined scope) |
| **API Endpoints** | 20+ |
| **Risk Models** | 3 initial, 6 total |
| **Cohort Types** | 3 (Static, Dynamic, Snapshot) |
| **Test Coverage** | >80% |
| **Determinism** | 100% (same input = same output) |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                 KB-11 POPULATION INTELLIGENCE LAYER                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  Population  │  │  Governed    │  │    Cohort    │  │  Analytics   │   │
│  │  Projection  │  │  Risk Engine │  │   Manager    │  │   Engine     │   │
│  │  (Read-Only) │  │  (KB-18)     │  │              │  │              │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
│         │                 │                 │                 │           │
│  ┌──────┴─────────────────┴─────────────────┴─────────────────┴──────┐    │
│  │                         Core Services                              │    │
│  ├────────────────────────────────────────────────────────────────────┤    │
│  │  • Patient Cache (TTL)     • Risk Model Loader (versioned)        │    │
│  │  • Attribution Overlay     • Cohort Refresh Scheduler             │    │
│  │  • Gap Aggregator          • Governance Emitter (KB-18)           │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                    UPSTREAM (Read-Only Sources)                     │    │
│  ├────────────────────────────────────────────────────────────────────┤    │
│  │  FHIR Store   │ KB-17       │ KB-13       │ KB-7       │ Redis     │    │
│  │  (Patients)   │ Registry    │ Care Gaps   │ Terminology│ Cache     │    │
│  │  [CONSUME]    │ [CONSUME]   │ [CONSUME]   │ [CONSUME]  │ [CACHE]   │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                    DOWNSTREAM (Governed Output)                     │    │
│  ├────────────────────────────────────────────────────────────────────┤    │
│  │  KB-18 Governance          │ KB-19 Navigator    │ Analytics APIs   │    │
│  │  [EMIT risk governance]    │ [PROVIDE scores]   │ [EXPOSE data]    │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase Breakdown (CTO-Approved Execution Order)

> **Build Philosophy**: Each phase delivers a measurable outcome before proceeding.

---

### Phase A: Population Projection + Read Models (Week 1-2)
**Goal**: "I can see my population" - Establish read-only data foundation

🎯 **Outcome**: KB-11 can ingest and display population data without being a source of truth

#### A.1 Project Scaffolding
```
kb-11-population-health/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── api/
│   │   ├── server.go               # Gin server setup
│   │   ├── middleware.go           # Request middleware
│   │   ├── projection_handlers.go  # Population view handlers (READ-ONLY)
│   │   └── health_handlers.go      # Health check endpoints
│   ├── config/
│   │   └── config.go               # Configuration management
│   ├── models/
│   │   ├── patient_projection.go   # Denormalized patient view (NOT source of truth)
│   │   ├── enums.go                # Risk tiers, cohort types
│   │   └── requests.go             # API request/response structs
│   ├── database/
│   │   ├── connection.go           # PostgreSQL connection
│   │   ├── migrations.go           # Schema migrations
│   │   └── projection_repository.go # Read-only patient projection
│   ├── projection/
│   │   ├── service.go              # Population projection service
│   │   ├── sync.go                 # FHIR/KB-17 sync (ingest only)
│   │   └── cache.go                # TTL-based patient cache
│   └── clients/
│       ├── fhir_client.go          # FHIR Store client (READ-ONLY)
│       ├── kb17_client.go          # KB-17 Registry client (READ-ONLY)
│       └── kb13_client.go          # KB-13 Care Gap client (READ-ONLY)
├── migrations/
│   └── 001_projection_schema.sql   # Projection tables (denormalized)
├── models/
│   └── risk-models/                # Risk model YAML definitions
├── tests/
│   ├── integration/                # Integration tests
│   └── determinism/                # Determinism guarantee tests
├── Dockerfile
├── docker-compose.yaml
├── go.mod
└── README.md
```

#### A.2 Database Schema (Population Projection Cache)
```sql
-- Population Projection (DENORMALIZED VIEW - NOT SOURCE OF TRUTH)
-- This is a read-through cache, data synced from FHIR Store + KB-17
CREATE TABLE patient_projections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- External references (source of truth is elsewhere)
    fhir_id VARCHAR(100) NOT NULL UNIQUE,  -- From FHIR Store
    kb17_patient_id UUID,                   -- From KB-17 Registry
    mrn VARCHAR(50),

    -- Cached demographics (synced from upstream)
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    date_of_birth DATE,
    gender VARCHAR(20),

    -- Attribution overlay (KB-11 enrichment)
    attributed_pcp VARCHAR(100),
    attributed_practice VARCHAR(100),
    attribution_date DATE,

    -- Computed fields (KB-11 owns these)
    current_risk_tier VARCHAR(20) DEFAULT 'UNSCORED',
    latest_risk_score DECIMAL(5,2),

    -- Aggregated from KB-13 (NOT source of truth)
    care_gap_count INTEGER DEFAULT 0,

    -- Sync metadata
    last_synced_at TIMESTAMP DEFAULT NOW(),
    sync_source VARCHAR(50),  -- 'FHIR' or 'KB17'

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Risk scores (KB-11 OWNS this data, governed by KB-18)
CREATE TABLE risk_assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_fhir_id VARCHAR(100) NOT NULL,

    -- Model governance (KB-18 integration)
    model_name VARCHAR(50) NOT NULL,
    model_version VARCHAR(20) NOT NULL,

    -- Score data
    score DECIMAL(5,2) NOT NULL,
    risk_tier VARCHAR(20) NOT NULL,
    contributing_factors JSONB,

    -- Determinism guarantee
    input_hash VARCHAR(64) NOT NULL,      -- SHA-256 of input data
    calculation_hash VARCHAR(64) NOT NULL, -- SHA-256 of score computation

    -- Governance emission
    governance_event_id UUID,              -- Reference to KB-18 event

    calculated_at TIMESTAMP DEFAULT NOW(),
    valid_until TIMESTAMP,

    UNIQUE(patient_fhir_id, model_name)
);

-- Indexes for performance
CREATE INDEX idx_projections_fhir ON patient_projections(fhir_id);
CREATE INDEX idx_projections_risk_tier ON patient_projections(current_risk_tier);
CREATE INDEX idx_projections_pcp ON patient_projections(attributed_pcp);
CREATE INDEX idx_assessments_patient ON risk_assessments(patient_fhir_id);
CREATE INDEX idx_assessments_tier ON risk_assessments(risk_tier);
CREATE INDEX idx_assessments_model ON risk_assessments(model_name, model_version);
```

#### A.3 Deliverables
- [ ] Project structure with go.mod
- [ ] Configuration management (Viper)
- [ ] PostgreSQL connection with pooling
- [ ] **FHIR Store sync client (READ-ONLY, no writes)**
- [ ] **KB-17 Registry sync client (READ-ONLY)**
- [ ] Patient projection cache (TTL-based, Redis)
- [ ] Health check endpoints (with upstream dependency status)
- [ ] Logging middleware with correlation IDs
- [ ] **NO patient create/update/delete endpoints** ❌

#### A.4 API Endpoints (Phase A) - READ-ONLY
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check (includes upstream status) |
| GET | `/api/v1/population` | List population (paginated) |
| GET | `/api/v1/population/:fhir_id` | Get patient projection |
| POST | `/api/v1/population/sync` | Trigger sync from FHIR/KB-17 |
| GET | `/api/v1/population/sync/status` | Sync status |

**⚠️ IMPORTANT**: No CREATE/UPDATE/DELETE for patients. KB-11 is not a registry.

---

### Phase B: Governed Risk Engine v1 (Week 2-3)
**Goal**: "I can stratify my population" - Risk scoring with KB-18 governance

🎯 **Outcome**: Every risk calculation is versioned, auditable, and governed

#### B.1 Risk Models (Start with 3, expand to 6)
```yaml
# models/risk-models/hospitalization-30day.yaml
name: hospitalization-30day
type: HOSPITALIZATION
version: "1.0.0"
description: "30-day hospital admission risk prediction"

# GOVERNANCE METADATA (KB-18 Integration)
governance:
  owner: "Population Health Team"
  clinical_reviewer: "CMO Office"
  last_approved: "2025-01-01"
  approval_id: "GOV-2025-001"
  requires_validation: true

factors:
  - name: prior_admissions
    weight: 0.25
    source: claims
    lookback_days: 365

  - name: chronic_conditions
    weight: 0.20
    conditions:
      - "I50.*"  # Heart failure
      - "J44.*"  # COPD
      - "E11.*"  # Type 2 diabetes

  - name: age_factor
    weight: 0.15
    thresholds:
      - range: [0, 65]
        score: 0.1
      - range: [65, 75]
        score: 0.3
      - range: [75, 999]
        score: 0.5

  - name: medication_complexity
    weight: 0.15
    threshold: 10  # medications

  - name: recent_ed_visits
    weight: 0.15
    lookback_days: 90

  - name: social_risk
    weight: 0.10
    sdoh_factors:
      - food_insecurity
      - housing_instability
      - transportation_barriers

tiers:
  LOW:
    range: [0, 30]
    intervention: "standard_preventive"
  MODERATE:
    range: [30, 50]
    intervention: "enhanced_monitoring"
  HIGH:
    range: [50, 75]
    intervention: "care_management"
  VERY_HIGH:
    range: [75, 100]
    intervention: "intensive_coordination"
```

#### B.2 Governed Risk Engine Components
```go
// internal/risk/engine.go
type GovernedRiskEngine struct {
    models        map[string]*RiskModel
    calculator    *RiskCalculator
    cache         *redis.Client
    kb7Client     *KB7Client           // Terminology lookups
    kb18Client    *KB18GovernanceClient // GOVERNANCE INTEGRATION
    vaidshala     *VaidshalaClient      // CQL evaluation (optional)
}

// Calculate risk with governance emission
func (e *GovernedRiskEngine) CalculateRisk(ctx context.Context, req *RiskRequest) (*GovernedRiskAssessment, error) {
    // 1. Load versioned model
    // 2. Compute input hash (determinism)
    // 3. Calculate score
    // 4. Compute output hash (determinism)
    // 5. EMIT to KB-18 governance
    // 6. Return with governance event ID
}

// GovernedRiskAssessment includes audit trail
type GovernedRiskAssessment struct {
    PatientFHIRID       string                 `json:"patient_fhir_id"`
    Score               float64                `json:"score"`
    RiskTier            string                 `json:"risk_tier"`
    ContributingFactors []ContributingFactor   `json:"contributing_factors"`

    // Governance fields (required for audit/payer scrutiny)
    ModelName           string                 `json:"model_name"`
    ModelVersion        string                 `json:"model_version"`
    InputHash           string                 `json:"input_hash"`     // SHA-256
    CalculationHash     string                 `json:"calculation_hash"` // SHA-256
    GovernanceEventID   string                 `json:"governance_event_id"` // KB-18 reference
    CalculatedAt        time.Time              `json:"calculated_at"`
}

// Batch calculation with governance
func (e *GovernedRiskEngine) BatchCalculate(ctx context.Context, fhirIDs []string, modelTypes []string) ([]*GovernedRiskAssessment, error)

// Get risk distribution across population
func (e *GovernedRiskEngine) GetDistribution(ctx context.Context, filters *DistributionFilters) (*RiskDistribution, error)
```

#### B.3 KB-18 Governance Integration
```go
// internal/clients/kb18_client.go
type KB18GovernanceClient struct {
    baseURL string
    client  *http.Client
}

// EmitRiskCalculationEvent sends governance event to KB-18
func (c *KB18GovernanceClient) EmitRiskCalculationEvent(ctx context.Context, event *RiskGovernanceEvent) (string, error)

type RiskGovernanceEvent struct {
    EventType       string    `json:"event_type"` // "RISK_CALCULATION"
    PatientFHIRID   string    `json:"patient_fhir_id"`
    ModelName       string    `json:"model_name"`
    ModelVersion    string    `json:"model_version"`
    Score           float64   `json:"score"`
    RiskTier        string    `json:"risk_tier"`
    InputHash       string    `json:"input_hash"`
    CalculationHash string    `json:"calculation_hash"`
    CalculatedAt    time.Time `json:"calculated_at"`
    CalculatedBy    string    `json:"calculated_by"` // "KB-11"
}
```

#### B.4 Deliverables
- [ ] Risk model YAML loader (with governance metadata)
- [ ] **Governed risk calculation engine**
- [ ] **3 initial risk models** (Hospitalization, Readmission, ED Utilization)
- [ ] Risk tier assignment logic
- [ ] Contributing factors extraction
- [ ] **KB-18 governance event emission**
- [ ] **Determinism hashing (input + calculation)**
- [ ] KB-7 integration (ICD/SNOMED lookups)
- [ ] Batch calculation with concurrency control
- [ ] Risk score caching (TTL-based)
- [ ] **Determinism tests** (same input = same output)

#### B.5 API Endpoints (Phase B)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/risk/models` | List available risk models |
| GET | `/api/v1/risk/models/:name` | Get risk model details (with governance info) |
| POST | `/api/v1/risk/calculate` | Calculate risk (returns governance event ID) |
| POST | `/api/v1/risk/calculate/batch` | Batch calculation (governed) |
| GET | `/api/v1/risk/distribution` | Population risk distribution |
| GET | `/api/v1/population/:fhir_id/risk` | Get patient's risk scores |
| POST | `/api/v1/population/:fhir_id/risk/calculate` | Calculate patient risk (governed) |
| GET | `/api/v1/population/high-risk` | List high-risk patients |
| GET | `/api/v1/population/rising-risk` | List rising-risk patients |

**⚠️ GOVERNANCE**: Every calculation emits to KB-18 for audit trail.

---

### Phase C: Cohort Management (Week 3-4)
**Goal**: "I can define who matters" - Cohort capabilities with snapshots

🎯 **Outcome**: Define, refresh, and analyze patient cohorts

#### C.1 Database Schema Extension
```sql
-- Cohorts table
CREATE TABLE cohorts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL,  -- STATIC, DYNAMIC, SNAPSHOT
    definition JSONB,           -- For DYNAMIC cohorts
    patient_count INTEGER DEFAULT 0,
    refresh_schedule VARCHAR(50),
    last_refreshed TIMESTAMP,
    statistics JSONB,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Cohort membership table
CREATE TABLE cohort_members (
    cohort_id UUID REFERENCES cohorts(id) ON DELETE CASCADE,
    patient_id UUID REFERENCES patients(id) ON DELETE CASCADE,
    added_at TIMESTAMP DEFAULT NOW(),
    added_by VARCHAR(100),
    removal_reason VARCHAR(200),
    PRIMARY KEY (cohort_id, patient_id)
);

-- Cohort snapshots for point-in-time analysis
CREATE TABLE cohort_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cohort_id UUID REFERENCES cohorts(id) ON DELETE CASCADE,
    snapshot_date TIMESTAMP DEFAULT NOW(),
    patient_ids UUID[],
    statistics JSONB,
    created_by VARCHAR(100)
);

CREATE INDEX idx_cohorts_type ON cohorts(type);
CREATE INDEX idx_cohort_members_patient ON cohort_members(patient_id);
```

#### C.2 Cohort Definition DSL
```json
{
  "name": "High-Risk Diabetes Patients",
  "type": "DYNAMIC",
  "definition": {
    "criteria": [
      {
        "field": "condition",
        "operator": "matches",
        "value": "E11.*",
        "source": "icd10"
      },
      {
        "field": "risk_tier",
        "operator": "in",
        "value": ["HIGH", "VERY_HIGH"]
      },
      {
        "field": "age",
        "operator": "gte",
        "value": 65
      },
      {
        "field": "last_hba1c",
        "operator": "gt",
        "value": 9.0,
        "lookback_days": 90
      }
    ],
    "logic": "AND",
    "exclusions": [
      {
        "field": "deceased",
        "operator": "eq",
        "value": true
      }
    ]
  },
  "refresh_schedule": "0 2 * * *"
}
```

#### C.3 Cohort Manager Components
```go
// internal/cohort/manager.go
type CohortManager struct {
    repo        *CohortRepository
    evaluator   *CriteriaEvaluator
    scheduler   *RefreshScheduler
    registry    *registry.Service
}

// Create cohort with validation
func (m *CohortManager) CreateCohort(ctx context.Context, req *CreateCohortRequest) (*Cohort, error)

// Refresh dynamic cohort membership
func (m *CohortManager) RefreshCohort(ctx context.Context, cohortID uuid.UUID) error

// Set operations
func (m *CohortManager) UnionCohorts(ctx context.Context, cohortIDs []uuid.UUID) (*Cohort, error)
func (m *CohortManager) IntersectCohorts(ctx context.Context, cohortIDs []uuid.UUID) (*Cohort, error)

// Statistics
func (m *CohortManager) GetStatistics(ctx context.Context, cohortID uuid.UUID) (*CohortStatistics, error)
```

#### C.4 Deliverables
- [ ] Cohort CRUD operations
- [ ] Criteria definition DSL parser
- [ ] Dynamic cohort evaluator
- [ ] Cohort refresh scheduler (cron-based)
- [ ] Set operations (union, intersect, except)
- [ ] Cohort statistics calculator
- [ ] Snapshot creation and retrieval
- [ ] Membership management (add/remove)
- [ ] Cohort comparison functionality
- [ ] Integration tests for cohort operations

#### C.5 API Endpoints (Phase C)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/cohorts` | List cohorts |
| POST | `/api/v1/cohorts` | Create cohort |
| GET | `/api/v1/cohorts/:id` | Get cohort |
| PUT | `/api/v1/cohorts/:id` | Update cohort |
| DELETE | `/api/v1/cohorts/:id` | Delete cohort |
| GET | `/api/v1/cohorts/:id/patients` | Get cohort members |
| POST | `/api/v1/cohorts/:id/patients` | Add patient to cohort |
| DELETE | `/api/v1/cohorts/:id/patients/:pid` | Remove patient |
| POST | `/api/v1/cohorts/:id/refresh` | Refresh cohort |
| POST | `/api/v1/cohorts/union` | Union cohorts |
| POST | `/api/v1/cohorts/intersect` | Intersect cohorts |
| GET | `/api/v1/cohorts/:id/snapshots` | List snapshots |
| POST | `/api/v1/cohorts/:id/snapshots` | Create snapshot |

---

### Phase D: Analytics Engine (Week 4-5)
**Goal**: "I can explain population behavior" - Analytics and comparisons

🎯 **Outcome**: Population-level insights, trends, and cohort comparisons

#### D.1 Analytics Components (Care Gaps from KB-13)
```go
// internal/analytics/engine.go
type AnalyticsEngine struct {
    db           *sql.DB
    cache        *redis.Client
    registry     *registry.Service
    riskEngine   *risk.Engine
    cohortMgr    *cohort.Manager
}

// Population summary
func (e *AnalyticsEngine) GetSummary(ctx context.Context, filters *SummaryFilters) (*PopulationSummary, error)

// Risk distribution analytics
func (e *AnalyticsEngine) GetRiskAnalytics(ctx context.Context, filters *RiskFilters) (*RiskAnalytics, error)

// Utilization reporting
func (e *AnalyticsEngine) GetUtilization(ctx context.Context, filters *UtilizationFilters) (*UtilizationReport, error)

// Care gap analysis
func (e *AnalyticsEngine) GetCareGapReport(ctx context.Context, filters *CareGapFilters) (*CareGapReport, error)

// Custom query execution
func (e *AnalyticsEngine) ExecuteQuery(ctx context.Context, query *AnalyticsQuery) (*QueryResult, error)

// Cohort comparison
func (e *AnalyticsEngine) CompareCohorts(ctx context.Context, req *CompareCohortsRequest) (*ComparisonResult, error)
```

#### D.2 Analytics Data Models
```go
// Population summary
type PopulationSummary struct {
    TotalPatients      int                    `json:"total_patients"`
    RiskDistribution   map[string]int         `json:"risk_distribution"`
    AgeDistribution    map[string]int         `json:"age_distribution"`
    GenderDistribution map[string]int         `json:"gender_distribution"`
    TopConditions      []ConditionCount       `json:"top_conditions"`
    AverageRiskScore   float64                `json:"average_risk_score"`
    CareGapsSummary    CareGapsSummary        `json:"care_gaps_summary"`
    TrendData          []TrendPoint           `json:"trend_data"`
    GeneratedAt        time.Time              `json:"generated_at"`
}

// Cohort comparison
type ComparisonResult struct {
    Cohorts            []CohortSummary        `json:"cohorts"`
    Metrics            []ComparisonMetric     `json:"metrics"`
    StatisticalTests   []StatisticalTest      `json:"statistical_tests"`
    Visualizations     []VisualizationData    `json:"visualizations"`
}
```

#### D.3 Deliverables
- [ ] Population summary aggregator
- [ ] Risk distribution calculator
- [ ] Utilization metrics collector
- [ ] **Care gap AGGREGATION (consume from KB-13, NOT define)**
- [ ] Custom query builder & executor
- [ ] Cohort comparison with statistics
- [ ] Trend analysis (time-series)
- [ ] Report generation framework
- [ ] Analytics caching layer
- [ ] **KB-13 client for care gap consumption**

**⚠️ CARE GAPS**: KB-11 AGGREGATES gaps from KB-13 (e.g., "20% have open gaps").
KB-11 does NOT define or adjudicate individual patient gaps.

#### D.4 API Endpoints (Phase D)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/analytics/summary` | Population summary |
| GET | `/api/v1/analytics/risk-distribution` | Risk analytics |
| GET | `/api/v1/analytics/utilization` | Utilization report |
| GET | `/api/v1/analytics/care-gaps` | Care gap report |
| POST | `/api/v1/analytics/query` | Custom analytics query |
| POST | `/api/v1/analytics/compare-cohorts` | Compare cohorts |
| GET | `/api/v1/analytics/trends` | Trend analysis |

---

### Phase E: Care Programs & SDOH (DEFERRED - Future Milestone)

> ⏸️ **DEFERRED BY CTO DECISION**
>
> Care Programs & SDOH is valuable but represents a **separate commercial product tier**.
> This phase is gated until:
> - KB-11 core (Phases A-D) is deployed and stable
> - Real usage patterns are observed
> - Business case for care management tier is validated

**Rationale**: Avoid scope creep. Deliver population intelligence first, then expand to care management.

#### E.1 Future Scope (When Activated)
- Care program CRUD operations
- Eligibility criteria evaluator
- Program enrollment workflows
- SDOH assessment recording
- Social risk scoring integration
- Intervention recommendation engine

#### E.2 Prerequisites Before Activation
- [ ] KB-11 Phase A-D in production >30 days
- [ ] >1000 risk calculations performed successfully
- [ ] >10 cohorts actively used
- [ ] Business approval for care management tier

---

### Phase F: Production Readiness & Testing Guarantees (Week 5+)
**Goal**: Production hardening with enterprise-grade testing guarantees

🎯 **Outcome**: KB-11 passes enterprise review with auditable, deterministic operations

#### F.1 External Integrations
```
┌─────────────────────────────────────────────────────────────────┐
│                    KB-11 Integration Points                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  FHIR Store (Google Healthcare API)                            │
│  └─ Patient sync, clinical data retrieval                      │
│  └─ Status: Phase 1 foundation                                 │
│                                                                 │
│  KB-7 (Terminology Service, port 8092)                         │
│  └─ ICD-10, SNOMED, LOINC lookups for cohort criteria          │
│  └─ Value set resolution for condition filtering               │
│  └─ Status: Phase 2-3 integration                              │
│                                                                 │
│  KB-13 (Quality Measures)                                      │
│  └─ Care gap data sharing                                      │
│  └─ Quality score integration in analytics                     │
│  └─ Status: Phase 4 integration                                │
│                                                                 │
│  Vaidshala (CQL Engine, port 8096)                             │
│  └─ Complex risk calculations using CQL                        │
│  └─ Clinical quality measure evaluation                        │
│  └─ Status: Phase 2 integration                                │
│                                                                 │
│  Redis (Cache, port 6380)                                      │
│  └─ Patient cache, risk score cache                            │
│  └─ Analytics result caching                                   │
│  └─ Status: Phase 1 foundation                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### F.2 Production Hardening Checklist
- [ ] Rate limiting middleware
- [ ] Request validation middleware
- [ ] Comprehensive error handling
- [ ] Structured logging (JSON, correlation IDs)
- [ ] Prometheus metrics endpoints
- [ ] Health check with dependency status
- [ ] Graceful shutdown handling
- [ ] Connection pooling optimization
- [ ] Query performance optimization
- [ ] Index verification and tuning

#### F.3 Security Checklist
- [ ] Authentication middleware (JWT)
- [ ] Authorization (role-based access)
- [ ] Input sanitization
- [ ] SQL injection prevention (parameterized queries)
- [ ] Audit logging for PHI access
- [ ] HIPAA compliance validation
- [ ] Secrets management (environment variables)

#### F.4 Documentation
- [ ] API documentation (OpenAPI/Swagger)
- [ ] Integration guide
- [ ] Deployment guide
- [ ] Operations runbook
- [ ] Risk model configuration guide
- [ ] Cohort definition DSL reference

#### F.5 Critical Testing Guarantees (Enterprise Review Requirements)

> **⚠️ Without these tests, KB-11 will fail enterprise review.**

| Test Category | Purpose | Pass Criteria |
|---------------|---------|---------------|
| **Determinism Tests** | Risk scores must not drift | Same input → Same output (100%) |
| **Cohort Stability Tests** | Same data = same cohort | Membership deterministic |
| **Snapshot Immutability** | Legal defensibility | Snapshots never modified after creation |
| **Scale Tests** | Production readiness | Handle 100k+ patients |
| **Governance Emission Tests** | KB-18 integration | Every calculation emits governance event |

##### F.5.1 Determinism Test Suite
```go
// tests/determinism/risk_determinism_test.go
func TestRiskCalculationDeterminism(t *testing.T) {
    // Given: Same patient data
    patientData := fixtures.LoadPatient("patient-001")

    // When: Calculate risk 100 times
    results := make([]*RiskAssessment, 100)
    for i := 0; i < 100; i++ {
        results[i] = engine.CalculateRisk(ctx, patientData, "hospitalization-30day")
    }

    // Then: All results must be identical
    firstHash := results[0].CalculationHash
    for i, result := range results {
        assert.Equal(t, firstHash, result.CalculationHash,
            "Calculation %d produced different hash", i)
        assert.Equal(t, results[0].Score, result.Score,
            "Calculation %d produced different score", i)
    }
}
```

##### F.5.2 Cohort Stability Test Suite
```go
// tests/stability/cohort_stability_test.go
func TestDynamicCohortStability(t *testing.T) {
    // Given: Same population data
    // When: Refresh cohort 10 times without data changes
    // Then: Membership must be identical each time
}
```

##### F.5.3 Snapshot Immutability Test
```go
// tests/immutability/snapshot_immutability_test.go
func TestSnapshotNeverChanges(t *testing.T) {
    // Given: Create snapshot
    snapshot := cohortMgr.CreateSnapshot(ctx, cohortID)
    originalPatients := snapshot.PatientIDs

    // When: Time passes, patients change, cohort refreshes
    // Then: Snapshot patients unchanged
    reloadedSnapshot := repo.GetSnapshot(ctx, snapshot.ID)
    assert.Equal(t, originalPatients, reloadedSnapshot.PatientIDs)
}
```

##### F.5.4 Scale Test Requirements
```yaml
scale_tests:
  population_sync:
    target: 100,000 patients
    max_duration: 10 minutes

  batch_risk_calculation:
    target: 10,000 patients
    max_duration: 5 minutes

  cohort_refresh:
    target: 50,000 member cohort
    max_duration: 2 minutes

  analytics_query:
    target: 100,000 patients
    max_duration: 5 seconds
```

##### F.5.5 Governance Emission Test
```go
// tests/governance/kb18_emission_test.go
func TestEveryCalculationEmitsGovernance(t *testing.T) {
    // Given: KB-18 mock server
    kb18Mock := httptest.NewServer(governanceMockHandler)

    // When: Calculate risk for patient
    result := engine.CalculateRisk(ctx, patientData, "hospitalization-30day")

    // Then: Governance event was emitted
    assert.NotEmpty(t, result.GovernanceEventID)
    assert.True(t, kb18Mock.ReceivedEvent(result.GovernanceEventID))
}
```

#### F.6 Standard Testing Requirements
- [ ] Unit tests (>80% coverage)
- [ ] Integration tests (all endpoints)
- [ ] Load testing (concurrent calculations)
- [ ] Security testing (OWASP top 10)

---

## Timeline Summary (Revised)

| Week | Phase | Focus | Key Deliverables | Measurable Outcome |
|------|-------|-------|------------------|-------------------|
| 1-2 | **Phase A** | Population Projection | FHIR/KB-17 sync, read-only cache | "I can see my population" |
| 2-3 | **Phase B** | Governed Risk Engine | 3 risk models, KB-18 governance | "I can stratify my population" |
| 3-4 | **Phase C** | Cohort Management | Dynamic cohorts, snapshots | "I can define who matters" |
| 4-5 | **Phase D** | Analytics Engine | Population analytics, KB-13 gaps | "I can explain population behavior" |
| 5+ | **Phase F** | Production Ready | Testing guarantees, hardening | Enterprise review passed |
| ⏸️ | **Phase E** | Care Programs (DEFERRED) | Program enrollment, SDOH | Future milestone |

### Critical Path
```
Week 1 ──▶ Week 2 ──▶ Week 3 ──▶ Week 4 ──▶ Week 5+
   A          B          C          D          F
 [SEE]    [STRATIFY]  [DEFINE]  [EXPLAIN]  [HARDEN]
```

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| FHIR Store connectivity issues | Medium | High | Implement retry logic, circuit breaker |
| Risk model accuracy | Medium | Medium | Validate with clinical SMEs, A/B testing |
| Cohort refresh performance | Medium | Medium | Batch processing, background workers |
| Integration complexity with KB-7/13 | Low | Medium | Clear interface contracts, mock services |
| Data volume scalability | Low | High | Pagination, caching, query optimization |

---

## Dependencies

### Required Services
| Service | Port | Purpose | Phase Required |
|---------|------|---------|----------------|
| PostgreSQL | 5433 | Primary database | Phase 1 |
| Redis | 6380 | Caching | Phase 1 |
| KB-7 Terminology | 8092 | Code lookups | Phase 2 |
| Vaidshala CQL | 8096 | Complex calculations | Phase 2 |
| KB-13 Quality | TBD | Care gaps | Phase 4 |
| FHIR Store | Google API | Patient data | Phase 1 |

### Development Dependencies
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL client
- Redis CLI

---

## Environment Variables

```bash
# Server
KB11_PORT=8111
KB11_LOG_LEVEL=info
KB11_ENVIRONMENT=development

# Database
KB11_DB_HOST=localhost
KB11_DB_PORT=5433
KB11_DB_NAME=kb11_population
KB11_DB_USER=postgres
KB11_DB_PASSWORD=password
KB11_DB_MAX_CONNS=50

# Cache
KB11_CACHE_ENABLED=true
KB11_CACHE_TTL=15m
KB11_REDIS_URL=redis://localhost:6380

# Risk Engine
KB11_RISK_MODELS_PATH=./models/risk-models
KB11_MAX_CONCURRENT=50

# Cohort Manager
KB11_MAX_COHORT_SIZE=100000
KB11_COHORT_REFRESH=1h

# External Services
VAIDSHALA_URL=http://localhost:8096
KB7_URL=http://localhost:8092
KB13_URL=http://localhost:8113
FHIR_STORE_URL=https://healthcare.googleapis.com/v1/...

# Security
KB11_JWT_SECRET=your-secret-key
KB11_CORS_ORIGINS=http://localhost:3000
```

---

## Success Criteria

### Functional
- [ ] Patient registry syncs with FHIR Store
- [ ] Risk scores calculate correctly for all 6 models
- [ ] Dynamic cohorts refresh automatically
- [ ] Analytics queries return within 2 seconds
- [ ] Care program enrollment workflows complete

### Non-Functional
- [ ] API response time < 200ms (p95)
- [ ] Batch calculation throughput > 1000 patients/minute
- [ ] System availability > 99.9%
- [ ] Test coverage > 80%
- [ ] Zero critical security vulnerabilities

---

## Next Steps

1. **Immediate**: Create project directory structure
2. **Day 1-2**: Implement Phase 1 foundation
3. **Weekly**: Progress review and adjustment
4. **Ongoing**: Integration testing with dependent services

---

*This plan follows established KB service patterns from KB-1, KB-7, and KB-14 to ensure architectural consistency across the platform.*
