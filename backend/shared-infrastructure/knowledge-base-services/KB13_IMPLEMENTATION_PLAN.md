# KB-13 Quality Measures Engine - Implementation Plan

> **Architecture Gate Status**: ✅ APPROVED with refinements (CTO/CMO Review 2026-01-05)

## Executive Summary

KB-13 is an **Enterprise Quality Measures Engine** providing organization-wide clinical quality measure calculation, reporting, and performance dashboards. It complements KB-9 (individual patient care gaps) by focusing on population-level quality measurement and regulatory reporting.

**Target Scope**: ~6,200 Lines of Code (Go) *(+200 LOC for period resolver)*
**Port**: 8113
**Dependencies**: Vaidshala CQL Engine, KB-7 Terminology, KB-18 Governance, KB-19 Protocol Orchestrator

---

## ⚠️ Critical Architecture Constraints (CTO/CMO Gate)

These constraints are **non-negotiable** and must be enforced before Phase 3:

### 🔴 1. Batch CQL Evaluation ONLY
```go
// ❌ FORBIDDEN - Will not scale
for _, patient := range patients {
    cqlClient.Evaluate(expression, patient)
}

// ✅ REQUIRED - Vectorized evaluation
cqlClient.EvaluateBatch(expression, []PatientContext)
```
**Rationale**: CQL engines are optimized for set logic. Per-patient calls break determinism under load.

### 🔴 2. Measurement Period Resolver Required
All date/period logic MUST go through `internal/period/` module.
CMS audit disputes are **80% date-related**. This module is mandatory.

### 🔴 3. Care Gaps Marked as "Derived"
KB-13 care gaps are **secondary signals**, not patient CDS truth.
```go
CareGap.Source = "QUALITY_MEASURE"  // Required field
```
KB-9 owns authoritative patient-level gap truth.

### 🟡 4. Execution Context Versioning (Audit Trail)
Every calculation must record:
- CQL library version
- Terminology version
- Measure YAML version
- KB-13 engine version

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      KB-13 QUALITY MEASURES ENGINE                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Measure    │  │ Calculation  │  │   Report     │  │  Care Gap    │   │
│  │  Definitions │  │   Engine     │  │  Generator   │  │  Identifier  │   │
│  │  (YAML/Go)   │  │  (CQL-based) │  │  (FHIR MR)   │  │  (Async)     │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
│         │                 │                 │                 │           │
│  ┌──────┴─────────────────┴─────────────────┴─────────────────┴──────┐    │
│  │                         Core Services                              │    │
│  ├────────────────────────────────────────────────────────────────────┤    │
│  │  • YAML Loader (Hot Reload)    • Result Cache (Redis TTL)         │    │
│  │  • Population Evaluator        • Stratification Engine            │    │
│  │  • Risk Adjustment             • Score Calculator                 │    │
│  │  • Async Job Manager           • Metrics Collector                │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                      External Integrations                          │    │
│  ├────────────────────────────────────────────────────────────────────┤    │
│  │  Vaidshala │ Patient  │ KB-7        │ KB-19        │ KB-18        │    │
│  │  (CQL)     │ Service  │ Terminology │ Protocol     │ Governance   │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
kb-13-quality-measures/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point (~150 LOC)
├── internal/
│   ├── api/
│   │   ├── server.go                  # HTTP server setup (~200 LOC)
│   │   ├── handlers.go                # REST API handlers (~300 LOC)
│   │   ├── health_handlers.go         # Health endpoints (~50 LOC)
│   │   └── middleware.go              # Auth, logging, metrics (~100 LOC)
│   ├── calculator/
│   │   ├── engine.go                  # Core calculation engine (~400 LOC)
│   │   ├── population.go              # Population evaluator (~200 LOC)
│   │   ├── stratification.go          # Stratification engine (~150 LOC)
│   │   ├── risk_adjustment.go         # Risk adjustment logic (~150 LOC)
│   │   └── cache.go                   # Result caching (~150 LOC)
│   ├── config/
│   │   └── config.go                  # Configuration (~150 LOC)
│   ├── cql/
│   │   ├── client.go                  # Vaidshala CQL client - BATCH ONLY (~250 LOC)
│   │   └── context_builder.go         # CQL context construction (~150 LOC)
│   ├── database/
│   │   ├── postgres.go                # PostgreSQL layer (~300 LOC)
│   │   ├── migrations.go              # DB migrations (~100 LOC)
│   │   └── queries.go                 # SQL queries (~200 LOC)
│   ├── integrations/
│   │   ├── kb7_client.go              # KB-7 Terminology client (~150 LOC)
│   │   ├── kb18_client.go             # KB-18 Governance client (~100 LOC)
│   │   ├── kb19_client.go             # KB-19 Protocol client (~100 LOC)
│   │   └── patient_service.go         # Patient service client (~150 LOC)
│   ├── loader/
│   │   ├── loader.go                  # YAML measure loader (~250 LOC)
│   │   ├── validator.go               # Measure validation (~150 LOC)
│   │   └── hot_reload.go              # Hot reload support (~100 LOC)
│   ├── metrics/
│   │   └── metrics.go                 # Prometheus metrics (~150 LOC)
│   ├── models/
│   │   ├── measure.go                 # Measure domain models (~300 LOC)
│   │   ├── report.go                  # Report models (~200 LOC)
│   │   ├── care_gap.go                # Care gap models - WITH SOURCE (~120 LOC)
│   │   ├── execution_context.go       # Versioned execution context (~100 LOC)
│   │   └── store.go                   # Measure store (~200 LOC)
│   ├── period/                        # 🔴 CRITICAL - Measurement Period Resolver
│   │   ├── resolver.go                # Period resolution logic (~100 LOC)
│   │   ├── rolling.go                 # Rolling period handling (~80 LOC)
│   │   ├── calendar.go                # Calendar period handling (~80 LOC)
│   │   └── alignment.go               # Stratification alignment (~60 LOC)
│   ├── reporter/
│   │   ├── generator.go               # Report generator (~250 LOC)
│   │   ├── fhir_measure_report.go     # FHIR MeasureReport output (~200 LOC)
│   │   └── templates.go               # Report templates (~100 LOC)
│   ├── scheduler/                     # 🟡 DEFERRED - Add after Phase 4 pilot
│   │   ├── scheduler.go               # Calculation scheduler (~200 LOC)
│   │   └── jobs.go                    # Job definitions (~150 LOC)
│   └── dashboard/
│       ├── service.go                 # Dashboard data service (~200 LOC)
│       └── trends.go                  # Trend analysis (~150 LOC)
├── measures/
│   ├── hedis/
│   │   ├── diabetes.yaml              # HBD, CDC measures (NO embedded benchmarks)
│   │   ├── cardiovascular.yaml        # CBP measures
│   │   ├── preventive.yaml            # Immunization, screening
│   │   └── behavioral.yaml            # Depression screening
│   └── cms/
│       ├── quality.yaml               # CMS quality measures
│       └── readmission.yaml           # Readmission measures
├── benchmarks/                        # 🟡 VERSIONED - Separate from measures
│   ├── HBD/
│   │   ├── 2023.yaml                  # Historical benchmarks
│   │   └── 2024.yaml                  # Current year benchmarks
│   ├── CBP/
│   │   ├── 2023.yaml
│   │   └── 2024.yaml
│   └── README.md                      # Benchmark versioning policy
├── cql/
│   └── tier-6-application/
│       ├── QualityMeasures-1.0.0.cql  # Quality measure CQL
│       └── DiabetesMeasures-1.0.0.cql # Diabetes-specific CQL
├── migrations/
│   ├── 001_initial_schema.sql
│   └── 002_add_reporting.sql
├── tests/
│   ├── unit/
│   │   ├── engine_test.go
│   │   ├── loader_test.go
│   │   ├── period_test.go             # 🔴 Period resolver tests (critical)
│   │   └── calculator_test.go
│   └── integration/
│       └── api_test.go
├── Dockerfile
├── docker-compose.yaml
├── go.mod
├── go.sum
└── README.md
```

---

## Implementation Phases

> **Phase Sequencing (CTO/CMO Approved)**:
> Phase 1-3 → Phase 4 → **🧪 PILOT** → Phase 5 → Phase 6

### Phase 1: Foundation (~1,200 LOC)
**Components**: cmd, config, models, api/server, api/health_handlers

| File | LOC | Description |
|------|-----|-------------|
| `cmd/server/main.go` | 150 | Entry point with graceful shutdown |
| `internal/config/config.go` | 150 | Environment configuration |
| `internal/models/measure.go` | 300 | Core domain models |
| `internal/models/store.go` | 200 | In-memory measure store |
| `internal/api/server.go` | 200 | HTTP server setup |
| `internal/api/health_handlers.go` | 50 | Health/ready endpoints |
| `internal/api/middleware.go` | 100 | Logging, auth middleware |

**Deliverables**:
- Working HTTP server on port 8113
- Health check endpoints
- Configuration from environment
- Basic logging and metrics

---

### Phase 2: Measure Definitions + Period Resolver (~970 LOC)
**Components**: loader, period, YAML measures, models/report

| File | LOC | Description |
|------|-----|-------------|
| `internal/loader/loader.go` | 250 | YAML loader with validation |
| `internal/loader/validator.go` | 150 | Schema validation |
| `internal/loader/hot_reload.go` | 100 | SIGHUP-based reload |
| `internal/period/resolver.go` | 100 | 🔴 Period resolution logic |
| `internal/period/rolling.go` | 80 | 🔴 Rolling period handling |
| `internal/period/calendar.go` | 80 | 🔴 Calendar period handling |
| `internal/period/alignment.go` | 60 | 🔴 Stratification alignment |
| `internal/models/report.go` | 200 | Report data structures |
| `measures/hedis/diabetes.yaml` | - | HbA1c, eye exam, nephropathy |
| `measures/hedis/cardiovascular.yaml` | - | Blood pressure control |
| `measures/cms/quality.yaml` | - | CMS122, CMS165, etc. |
| `benchmarks/HBD/2024.yaml` | - | 🟡 Versioned benchmarks |

**Deliverables**:
- YAML measure loading with hot reload
- Measure validation against schema
- 🔴 **MeasurementPeriodResolver** (rolling, calendar, alignment)
- 10+ HEDIS/CMS measure definitions
- 🟡 Versioned benchmarks separate from measures
- GET /api/v1/measures endpoints

---

### Phase 3: Calculation Engine (~1,420 LOC)
**Components**: calculator, cql, database, execution_context

| File | LOC | Description |
|------|-----|-------------|
| `internal/calculator/engine.go` | 400 | Core calculation orchestration |
| `internal/calculator/population.go` | 200 | Population criteria evaluation |
| `internal/calculator/stratification.go` | 150 | Age/demographic stratification |
| `internal/calculator/cache.go` | 150 | Redis result caching |
| `internal/cql/client.go` | 250 | 🔴 Vaidshala CQL - **BATCH ONLY** |
| `internal/cql/context_builder.go` | 150 | Clinical context construction |
| `internal/models/execution_context.go` | 100 | 🟡 Versioned execution context |
| `internal/database/postgres.go` | 300 | PostgreSQL persistence |

**🔴 CRITICAL CONSTRAINT**: CQL client MUST use batch evaluation:
```go
// ✅ REQUIRED interface
func (c *CQLClient) EvaluateBatch(
    ctx context.Context,
    expression string,
    patients []PatientContext,
) ([]EvaluationResult, error)
```

**Deliverables**:
- 🔴 **Batch CQL-powered** measure evaluation
- Population identification (IP, denominator, numerator)
- Stratification by age, gender, etc.
- 🟡 **ExecutionContextVersion** tracking for audits
- Result caching with TTL
- POST /api/v1/calculate endpoint

---

### Phase 4: Reporting & Care Gaps (~870 LOC)
**Components**: reporter, care_gap, api/handlers

| File | LOC | Description |
|------|-----|-------------|
| `internal/reporter/generator.go` | 250 | Report generation |
| `internal/reporter/fhir_measure_report.go` | 200 | FHIR R4 MeasureReport |
| `internal/reporter/templates.go` | 100 | Report templates |
| `internal/models/care_gap.go` | 120 | 🔴 Care gap models **WITH SOURCE** |
| `internal/api/handlers.go` | 300 | Full REST API handlers |

**🔴 CRITICAL CONSTRAINT**: Care gaps must include source annotation:
```go
type CareGap struct {
    // ... other fields
    Source       CareGapSource `json:"source"`  // 🔴 REQUIRED: "QUALITY_MEASURE"
    IsAuthoritative bool       `json:"is_authoritative"` // false for KB-13
}
```

**Deliverables**:
- Individual, subject-list, summary reports
- FHIR MeasureReport generation
- 🔴 Care gap identification with **DERIVED** source annotation
- Full REST API implementation

---

### 🧪 PILOT GATE (After Phase 4)

Before proceeding to Phase 5:
1. Deploy to staging environment
2. Run 3+ real measures against test population
3. Validate FHIR MeasureReport output
4. Verify care gaps integrate correctly with KB-9/KB-18
5. CTO/CMO sign-off required

---

### Phase 5: Dashboard & Integrations (~550 LOC)
**Components**: dashboard, integrations (scheduler DEFERRED)

| File | LOC | Description |
|------|-----|-------------|
| `internal/dashboard/service.go` | 200 | Dashboard data service |
| `internal/dashboard/trends.go` | 150 | Trend analysis |
| `internal/integrations/kb7_client.go` | 150 | KB-7 terminology |
| `internal/integrations/kb18_client.go` | 100 | KB-18 governance |
| `internal/integrations/kb19_client.go` | 100 | KB-19 protocol |

> **🟡 DEFERRED**: Scheduler (Phase 5b) - Add after pilot stabilizes
> Early pilots run on-demand via `POST /calculate`. Cron adds complexity.

**Deliverables**:
- Quality dashboard endpoint
- Trend analysis over time
- Facility comparison
- KB-7/KB-18/KB-19 integrations

---

### Phase 5b: Scheduling (DEFERRED - ~350 LOC)
**Add after pilot stabilizes**

| File | LOC | Description |
|------|-----|-------------|
| `internal/scheduler/scheduler.go` | 200 | Cron-based scheduling |
| `internal/scheduler/jobs.go` | 150 | Job definitions |

**Deliverables**:
- Automated daily/monthly/quarterly calculations
- External trigger support (Kafka, webhook)

---

### Phase 6: Tests & Polish (~600 LOC)
**Components**: tests, migrations, Docker

| File | LOC | Description |
|------|-----|-------------|
| `tests/unit/engine_test.go` | 200 | Calculator tests |
| `tests/unit/loader_test.go` | 100 | Loader tests |
| `tests/unit/period_test.go` | 100 | 🔴 Period resolver tests |
| `tests/integration/api_test.go` | 150 | API integration tests |
| `migrations/*.sql` | 100 | Database schema |

**Deliverables**:
- 80%+ test coverage
- 🔴 Period resolver tests (critical for audit defense)
- Database migrations
- Docker deployment
- Documentation

---

## Data Models

### Core Measure Definition (YAML)

```yaml
type: measure
measure:
  id: HBD
  version: "2024"
  name: Hemoglobin A1c Control for Patients with Diabetes
  title: HbA1c Control (<8%)

  type: PROCESS          # PROCESS | OUTCOME | STRUCTURE | EFFICIENCY | COMPOSITE
  scoring: proportion    # proportion | ratio | continuous | composite
  domain: DIABETES       # Clinical domain
  program: HEDIS         # HEDIS | CMS | MIPS | ACO | PCMH | NQF | CUSTOM

  # External identifiers
  nqf_number: "0059"
  cms_number: "CMS122v11"
  hedis_code: HBD

  measurement_period:
    type: rolling        # rolling | calendar
    duration: P1Y        # ISO 8601 duration

  populations:
    - id: initial-population
      type: initial-population
      description: Patients 18-75 with diabetes
      cql_expression: InInitialPopulation

    - id: denominator
      type: denominator
      description: Equals initial population
      cql_expression: InDenominator

    - id: denominator-exclusion
      type: denominator-exclusion
      description: Hospice, ESRD, palliative care
      cql_expression: HasDenominatorExclusion

    - id: numerator
      type: numerator
      description: HbA1c < 8%
      cql_expression: InNumerator
      criteria:
        lab_results:
          - lab_code: "4548-4"    # HbA1c LOINC
            operator: "<"
            value: 8.0
            time_window: "during measurement period"

  stratifications:
    - id: age-strat
      description: Age stratification
      components: ["18-44", "45-64", "65-75"]

  improvement_notation: increase

  benchmarks:
    - year: 2023
      percentile: 50
      value: 65.0
    - year: 2023
      percentile: 90
      value: 80.0

  evidence:
    level: A
    source: ADA Guidelines 2024
    guideline: ADA Standards of Medical Care
```

### Go Domain Models

```go
// Measure represents a quality measure definition
type Measure struct {
    ID                   string              `json:"id" yaml:"id"`
    Version              string              `json:"version" yaml:"version"`
    Name                 string              `json:"name" yaml:"name"`
    Title                string              `json:"title" yaml:"title"`
    Type                 MeasureType         `json:"type" yaml:"type"`
    Scoring              ScoringType         `json:"scoring" yaml:"scoring"`
    Domain               ClinicalDomain      `json:"domain" yaml:"domain"`
    Program              QualityProgram      `json:"program" yaml:"program"`
    NQFNumber            string              `json:"nqf_number,omitempty" yaml:"nqf_number"`
    CMSNumber            string              `json:"cms_number,omitempty" yaml:"cms_number"`
    HEDISCode            string              `json:"hedis_code,omitempty" yaml:"hedis_code"`
    MeasurementPeriod    MeasurementPeriod   `json:"measurement_period" yaml:"measurement_period"`
    Populations          []Population        `json:"populations" yaml:"populations"`
    Stratifications      []Stratification    `json:"stratifications" yaml:"stratifications"`
    ImprovementNotation  string              `json:"improvement_notation" yaml:"improvement_notation"`
    BenchmarkRef         string              `json:"benchmark_ref" yaml:"benchmark_ref"` // 🟡 Reference to versioned benchmark
    Evidence             Evidence            `json:"evidence" yaml:"evidence"`
}

// 🟡 ExecutionContextVersion - REQUIRED for audit trail
type ExecutionContextVersion struct {
    KB13Version        string `json:"kb13_version"`         // e.g., "1.0.0"
    CQLLibraryVersion  string `json:"cql_library_version"`  // e.g., "QualityMeasures-1.0.0"
    TerminologyVersion string `json:"terminology_version"`  // KB-7 version at execution time
    MeasureYAMLVersion string `json:"measure_yaml_version"` // e.g., "HBD-2024-v1"
    ExecutedAt         time.Time `json:"executed_at"`
}

// CalculationResult represents measure calculation output
type CalculationResult struct {
    MeasureID           string                 `json:"measure_id"`
    ReportType          ReportType             `json:"report_type"`
    PeriodStart         time.Time              `json:"period_start"`
    PeriodEnd           time.Time              `json:"period_end"`
    InitialPopulation   int                    `json:"initial_population"`
    Denominator         int                    `json:"denominator"`
    DenominatorExclusion int                   `json:"denominator_exclusion"`
    DenominatorException int                   `json:"denominator_exception"`
    Numerator           int                    `json:"numerator"`
    NumeratorExclusion  int                    `json:"numerator_exclusion"`
    Score               float64                `json:"score"`
    Stratifications     []StratificationResult `json:"stratifications"`
    CareGaps            []CareGap              `json:"care_gaps"`
    ExecutionTimeMs     int64                  `json:"execution_time_ms"`

    // 🟡 REQUIRED: Versioning for audit trail
    ExecutionContext    ExecutionContextVersion `json:"execution_context"`
}

// CareGapSource indicates where the care gap was identified
type CareGapSource string

const (
    CareGapSourceQualityMeasure CareGapSource = "QUALITY_MEASURE"  // KB-13 (derived)
    CareGapSourcePatientCDS     CareGapSource = "PATIENT_CDS"      // KB-9 (authoritative)
)

// CareGap represents an identified care gap
// 🔴 CRITICAL: KB-13 gaps are DERIVED, not authoritative
type CareGap struct {
    ID              string          `json:"id"`
    MeasureID       string          `json:"measure_id"`
    SubjectID       string          `json:"subject_id"`
    GapType         string          `json:"gap_type"`
    Description     string          `json:"description"`
    Priority        Priority        `json:"priority"`
    Status          CareGapStatus   `json:"status"`
    DueDate         *time.Time      `json:"due_date,omitempty"`
    Intervention    string          `json:"intervention"`
    CreatedAt       time.Time       `json:"created_at"`

    // 🔴 REQUIRED: Source annotation
    Source          CareGapSource   `json:"source"`           // Must be "QUALITY_MEASURE"
    IsAuthoritative bool            `json:"is_authoritative"` // Always false for KB-13
}
```

### 🔴 Period Resolver Interface (Critical)

```go
// PeriodResolver handles all measurement period logic
// CMS audit disputes are 80% date-related - this module is mandatory
type PeriodResolver interface {
    // Resolve calculates the actual period based on measure definition
    Resolve(measurePeriod MeasurementPeriod, referenceDate time.Time) (start, end time.Time, err error)

    // AlignToStratification ensures period aligns with stratification boundaries
    AlignToStratification(period Period, stratification Stratification) (Period, error)

    // ValidatePatientInPeriod checks if patient data falls within period
    ValidatePatientInPeriod(patientData PatientContext, period Period) bool
}

// RollingPeriodResolver implements rolling period logic (e.g., "last 12 months")
type RollingPeriodResolver struct{}

// CalendarPeriodResolver implements calendar period logic (e.g., "CY 2024")
type CalendarPeriodResolver struct{}
```

### 🟡 Versioned Benchmark Structure

```yaml
# benchmarks/HBD/2024.yaml
measure_id: HBD
year: 2024
source: NCQA
effective_date: 2024-01-01
benchmarks:
  - percentile: 25
    value: 55.0
  - percentile: 50
    value: 65.0
  - percentile: 75
    value: 75.0
  - percentile: 90
    value: 80.0
notes: |
  Benchmarks updated annually by NCQA.
  Source: HEDIS Audit Methodology 2024.
```

---

## API Endpoints

### Measures
```
GET  /api/v1/measures                    # List all measures
GET  /api/v1/measures/:id                # Get measure by ID
GET  /api/v1/measures/program/:program   # Get by program (HEDIS, CMS, etc.)
GET  /api/v1/measures/domain/:domain     # Get by domain (DIABETES, etc.)
POST /api/v1/measures/reload             # Hot reload definitions
```

### Calculations
```
POST /api/v1/calculate                   # Calculate measure
POST /api/v1/calculate/batch             # Batch calculation
POST /api/v1/calculate/async             # Async calculation (returns job ID)
GET  /api/v1/calculate/job/:id           # Get async job status
```

### Reports
```
GET /api/v1/reports/:id                  # Get report by ID
GET /api/v1/reports/measure/:measureId   # Get all reports for measure
GET /api/v1/reports/latest/:measureId    # Get latest report
GET /api/v1/reports/:id/fhir             # Export as FHIR MeasureReport
```

### Care Gaps
```
GET  /api/v1/care-gaps/patient/:id       # Get patient care gaps
GET  /api/v1/care-gaps/measure/:measureId # Get all gaps for measure
PUT  /api/v1/care-gaps/:id/status        # Update gap status
POST /api/v1/care-gaps/identify/:measureId # Identify gaps for measure
```

### Dashboard
```
GET /api/v1/dashboard                    # Quality dashboard summary
GET /api/v1/dashboard/trend/:measureId   # Measure trend over time
GET /api/v1/dashboard/comparison         # Facility comparison
GET /api/v1/dashboard/program/:program   # Program-level dashboard
```

---

## Integration Points

### Vaidshala CQL Engine (Primary) - 🔴 BATCH ONLY

KB-13 uses Vaidshala's CQL Engine for clinical fact evaluation.
**CRITICAL**: All CQL evaluation MUST use batch/vectorized calls.

```go
// 🔴 CQLClient interface - BATCH ONLY
type CQLClient interface {
    // EvaluateBatch evaluates a CQL expression against multiple patients
    // This is the ONLY allowed evaluation method
    EvaluateBatch(ctx context.Context, expression string, patients []PatientContext) ([]EvaluationResult, error)

    // GetLibraryVersion returns the CQL library version for audit trail
    GetLibraryVersion(libraryName string) (string, error)
}

// ❌ FORBIDDEN - Do NOT implement per-patient evaluation
// func Evaluate(ctx context.Context, expression string, patient PatientContext) (EvaluationResult, error)

// ✅ CORRECT: Batch evaluation pattern
func (c *Calculator) EvaluatePopulation(ctx context.Context, measureID string, patientIDs []string) (*CalculationResult, error) {
    measure := c.store.GetMeasure(measureID)

    // Build batch patient contexts
    patients, err := c.patientService.GetBatchContext(ctx, patientIDs)
    if err != nil {
        return nil, fmt.Errorf("failed to get patient contexts: %w", err)
    }

    // Resolve measurement period using PeriodResolver (🔴 CRITICAL)
    period, err := c.periodResolver.Resolve(measure.MeasurementPeriod, time.Now())
    if err != nil {
        return nil, fmt.Errorf("failed to resolve period: %w", err)
    }

    result := &CalculationResult{
        MeasureID:   measureID,
        PeriodStart: period.Start,
        PeriodEnd:   period.End,
        ExecutionContext: ExecutionContextVersion{
            KB13Version:        version,
            CQLLibraryVersion:  c.cqlClient.GetLibraryVersion("QualityMeasures"),
            TerminologyVersion: c.kb7Client.GetVersion(),
            MeasureYAMLVersion: measure.Version,
            ExecutedAt:         time.Now(),
        },
    }

    // Evaluate each population using BATCH calls
    for _, population := range measure.Populations {
        // ✅ BATCH evaluation - all patients at once
        evalResults, err := c.cqlClient.EvaluateBatch(ctx, population.CQLExpression, patients)
        if err != nil {
            return nil, fmt.Errorf("CQL evaluation failed for %s: %w", population.ID, err)
        }

        // Count results by population type
        count := countTrueResults(evalResults)
        switch population.Type {
        case "initial-population":
            result.InitialPopulation = count
        case "denominator":
            result.Denominator = count
        case "denominator-exclusion":
            result.DenominatorExclusion = count
        case "numerator":
            result.Numerator = count
        }
    }

    // Calculate score
    result.Score = calculateScore(result)

    return result, nil
}
```

### KB-7 Terminology Service
Value set resolution for diagnosis codes, procedure codes:

```go
// Resolve value set for diabetes diagnosis codes
valueSet, err := kb7Client.GetValueSet(ctx, "2.16.840.1.113883.3.464.1003.103.12.1001")
```

### KB-18 Governance Engine
Quality performance tracking for compliance:

```go
// Report quality performance
kb18Client.ReportQualityPerformance(ctx, &GovernanceReport{
    MeasureID: "HBD",
    Score:     72.5,
    Period:    "2024-Q4",
})
```

### KB-19 Protocol Orchestrator
Quality alerts for protocol adjustments:

```go
// Send alert when quality measure below threshold
kb19Client.SendQualityAlert(ctx, &QualityAlert{
    MeasureID: "HBD",
    Threshold: 70.0,
    Actual:    65.0,
    Action:    "Review protocol adherence",
})
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KB13_PORT` | 8113 | HTTP server port |
| `KB13_MEASURES_PATH` | ./measures | Path to measure definitions |
| `KB13_LOG_LEVEL` | info | Log level (debug, info, warn, error) |
| `KB13_DB_HOST` | localhost | PostgreSQL host |
| `KB13_DB_PORT` | 5432 | PostgreSQL port |
| `KB13_DB_NAME` | kb13_quality | Database name |
| `KB13_DB_USER` | postgres | Database user |
| `KB13_DB_PASSWORD` | | Database password |
| `KB13_REDIS_URL` | redis://localhost:6379 | Redis URL for caching |
| `KB13_ENABLE_CACHING` | true | Enable result caching |
| `KB13_CACHE_TTL` | 15m | Cache TTL |
| `KB13_MAX_CONCURRENT` | 50 | Max concurrent calculations |
| `KB13_CALC_TIMEOUT` | 60s | Calculation timeout |
| `KB13_SCHEDULER_ENABLED` | false | Enable scheduler |
| `VAIDSHALA_URL` | http://localhost:8096 | CQL engine URL |
| `KB7_URL` | http://localhost:8092 | KB-7 Terminology URL |
| `KB18_URL` | http://localhost:8118 | KB-18 Governance URL |
| `KB19_URL` | http://localhost:8119 | KB-19 Protocol URL |
| `PATIENT_SERVICE_URL` | http://localhost:8080 | Patient service URL |

---

## Database Schema

```sql
-- Measure Reports (with execution context versioning)
CREATE TABLE measure_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    measure_id VARCHAR(50) NOT NULL,
    report_type VARCHAR(20) NOT NULL,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    initial_population INT NOT NULL,
    denominator INT NOT NULL,
    denominator_exclusion INT DEFAULT 0,
    denominator_exception INT DEFAULT 0,
    numerator INT NOT NULL,
    numerator_exclusion INT DEFAULT 0,
    score DECIMAL(5,2) NOT NULL,
    stratifications JSONB,

    -- 🟡 Execution context versioning (audit trail)
    kb13_version VARCHAR(20) NOT NULL,
    cql_library_version VARCHAR(50) NOT NULL,
    terminology_version VARCHAR(50) NOT NULL,
    measure_yaml_version VARCHAR(50) NOT NULL,
    execution_context JSONB NOT NULL,       -- Full context snapshot

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Care Gaps (with source annotation)
-- 🔴 CRITICAL: source field distinguishes KB-13 (derived) from KB-9 (authoritative)
CREATE TABLE care_gaps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    measure_id VARCHAR(50) NOT NULL,
    subject_id VARCHAR(100) NOT NULL,
    gap_type VARCHAR(50) NOT NULL,
    description TEXT,
    priority VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'open',
    due_date DATE,
    intervention TEXT,
    closed_at TIMESTAMP,

    -- 🔴 REQUIRED: Source annotation
    source VARCHAR(30) NOT NULL DEFAULT 'QUALITY_MEASURE',  -- 'QUALITY_MEASURE' or 'PATIENT_CDS'
    is_authoritative BOOLEAN NOT NULL DEFAULT FALSE,        -- Always FALSE for KB-13

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Constraint to enforce KB-13 gaps are never authoritative
    CONSTRAINT chk_kb13_not_authoritative CHECK (
        source != 'QUALITY_MEASURE' OR is_authoritative = FALSE
    )
);

-- Calculation Jobs (async)
CREATE TABLE calculation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    measure_id VARCHAR(50) NOT NULL,
    report_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    progress INT DEFAULT 0,
    result JSONB,
    error TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 🟡 Versioned Benchmarks table
CREATE TABLE benchmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    measure_id VARCHAR(50) NOT NULL,
    year INT NOT NULL,
    source VARCHAR(50) NOT NULL,          -- e.g., 'NCQA', 'CMS'
    effective_date DATE NOT NULL,
    percentile_25 DECIMAL(5,2),
    percentile_50 DECIMAL(5,2),
    percentile_75 DECIMAL(5,2),
    percentile_90 DECIMAL(5,2),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(measure_id, year, source)
);

-- Indexes
CREATE INDEX idx_reports_measure ON measure_reports(measure_id);
CREATE INDEX idx_reports_period ON measure_reports(period_start, period_end);
CREATE INDEX idx_reports_version ON measure_reports(kb13_version, cql_library_version);
CREATE INDEX idx_gaps_subject ON care_gaps(subject_id);
CREATE INDEX idx_gaps_measure ON care_gaps(measure_id);
CREATE INDEX idx_gaps_status ON care_gaps(status);
CREATE INDEX idx_gaps_source ON care_gaps(source);             -- 🔴 Important for KB-9/KB-13 queries
CREATE INDEX idx_jobs_status ON calculation_jobs(status);
CREATE INDEX idx_benchmarks_measure ON benchmarks(measure_id, year);
```

---

## Implementation Timeline

> **Revised with CTO/CMO Refinements**

| Phase | Components | LOC | Priority | Gate |
|-------|------------|-----|----------|------|
| 1 | Foundation (cmd, config, models, api/server) | ~1,200 | P0 - Critical | — |
| 2 | Measure Definitions + 🔴 Period Resolver | ~970 | P0 - Critical | — |
| 3 | Calculation Engine (🔴 batch CQL, 🟡 exec versioning) | ~1,420 | P0 - Critical | — |
| 4 | Reporting & Care Gaps (🔴 source annotation) | ~870 | P0 - Critical | — |
| — | **🧪 PILOT GATE** | — | — | CTO/CMO |
| 5 | Dashboard & Integrations | ~550 | P1 - Important | — |
| 5b | Scheduling (DEFERRED) | ~350 | P2 - Deferred | — |
| 6 | Tests & Polish (🔴 period tests) | ~600 | P1 - Important | — |
| **Total** | | **~6,200** | | |

### Critical Path Dependencies

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → 🧪 PILOT → Phase 5 → Phase 6
              ↓
      Period Resolver (🔴)
      MUST complete before Phase 3
```

---

## Success Criteria

### 1. Functional Requirements
- [ ] Calculate HEDIS/CMS quality measures with CQL evaluation
- [ ] Generate individual, subject-list, and summary reports
- [ ] Identify care gaps for patients not meeting quality goals
- [ ] Support automated scheduled calculations (Phase 5b)
- [ ] Provide real-time quality dashboard

### 2. Performance Requirements
- [ ] Calculate single measure for 1,000 patients in < 30 seconds
- [ ] Support 50 concurrent calculations
- [ ] Cache results with configurable TTL
- [ ] Sub-second response for dashboard queries

### 3. Integration Requirements
- [ ] Integrate with Vaidshala CQL Engine
- [ ] Connect to KB-7 for terminology resolution
- [ ] Report to KB-18 for governance tracking
- [ ] Alert KB-19 for protocol adjustments

### 4. Quality Requirements
- [ ] 80%+ test coverage
- [ ] FHIR R4 MeasureReport compliance
- [ ] Hot reload measure definitions without restart
- [ ] Prometheus metrics for monitoring

### 5. 🔴 CTO/CMO Gate Requirements (Non-Negotiable)
- [ ] **Batch CQL ONLY**: No per-patient CQL calls exist in codebase
- [ ] **Period Resolver**: All date logic goes through `internal/period/`
- [ ] **Care Gap Source**: All gaps have `Source = "QUALITY_MEASURE"`
- [ ] **Execution Versioning**: All results include `ExecutionContextVersion`
- [ ] **Versioned Benchmarks**: Benchmarks stored separately from measures
- [ ] **Period Tests**: `period_test.go` covers rolling, calendar, alignment

---

## Next Steps

### Immediate (Before Phase 1)
1. ✅ **Plan Approved**: CTO/CMO gate passed with refinements
2. **Create Directory**: Initialize `kb-13-quality-measures/` structure
3. **Initialize Go Module**: `go mod init kb-13-quality-measures`

### Phase Execution
```
Phase 1 (Foundation)     → Working server on port 8113
Phase 2 (Definitions)    → YAML measures + 🔴 Period Resolver
Phase 3 (Calculation)    → 🔴 Batch CQL + 🟡 Versioning
Phase 4 (Reporting)      → FHIR MeasureReport + 🔴 Source annotation
         ↓
    🧪 PILOT GATE
         ↓
Phase 5 (Dashboard)      → Integrations + trends
Phase 5b (Scheduler)     → DEFERRED until pilot stable
Phase 6 (Polish)         → Tests + Docker + docs
```

### Go-Live Checklist
- [ ] 3+ measures validated against test population
- [ ] FHIR MeasureReport passes FHIR validator
- [ ] Care gaps integrate correctly with KB-9/KB-18
- [ ] Period resolver handles edge cases (leap year, partial periods)
- [ ] CTO/CMO sign-off on pilot results

---

## References

- [FHIR MeasureReport](https://www.hl7.org/fhir/measurereport.html)
- [HEDIS Specifications](https://www.ncqa.org/hedis/)
- [CMS eCQM Library](https://ecqi.healthit.gov/ecqms)
- [CQL Specification](https://cql.hl7.org/)
