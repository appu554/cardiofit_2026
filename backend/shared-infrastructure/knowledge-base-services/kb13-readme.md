# KB-13 Quality Measures Engine

Clinical quality measure calculation and reporting engine supporting HEDIS, CMS, NQF, and custom measures.

## Overview

KB-13 provides comprehensive quality measure management including:

| Feature | Description |
|---------|-------------|
| **Measure Definitions** | YAML-based measure specifications with populations, stratifications, and supplemental data |
| **Calculation Engine** | CQL-powered evaluation with caching and concurrent processing |
| **Reporting** | Individual, subject-list, and summary reports with trend analysis |
| **Care Gap Identification** | Automated detection of patients missing quality goals |
| **Scheduling** | Automated daily, monthly, and quarterly calculations |
| **Dashboard** | Real-time quality performance visualization |

## Supported Programs

| Program | Description |
|---------|-------------|
| HEDIS | Healthcare Effectiveness Data and Information Set |
| CMS | Centers for Medicare & Medicaid Services quality measures |
| MIPS | Merit-based Incentive Payment System |
| ACO | Accountable Care Organization measures |
| PCMH | Patient-Centered Medical Home measures |
| NQF | National Quality Forum endorsed measures |
| Custom | Organization-specific quality measures |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      KB-13 QUALITY MEASURES ENGINE                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Measure    │  │ Calculation  │  │   Report     │  │  Care Gap    │   │
│  │  Definitions │  │   Engine     │  │  Generator   │  │  Identifier  │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
│         │                 │                 │                 │           │
│  ┌──────┴─────────────────┴─────────────────┴─────────────────┴──────┐    │
│  │                         Core Services                              │    │
│  ├────────────────────────────────────────────────────────────────────┤    │
│  │  • YAML Loader (Hot Reload)    • Result Cache (TTL-based)         │    │
│  │  • Population Evaluator        • Stratification Engine            │    │
│  │  • Risk Adjustment             • Score Calculator                 │    │
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

## Measure Types

| Type | Scoring | Description |
|------|---------|-------------|
| Process | Proportion | Measures whether an action occurred |
| Outcome | Proportion | Measures clinical outcomes |
| Structure | Ratio | Measures organizational resources |
| Efficiency | Continuous | Measures cost/utilization |
| Composite | Weighted | Combines multiple measures |
| Intermediate | Proportion | Measures intermediate outcomes |

## Clinical Domains

- **Diabetes** - HbA1c control, eye exams, kidney monitoring
- **Cardiovascular** - Blood pressure, cholesterol, aspirin use
- **Respiratory** - Asthma control, COPD management
- **Preventive** - Immunizations, cancer screenings
- **Behavioral Health** - Depression screening, follow-up care
- **Maternal** - Prenatal care, postpartum care
- **Pediatric** - Well-child visits, immunizations
- **Patient Safety** - Hospital-acquired conditions

## API Endpoints

### Measures
```
GET  /api/v1/measures                    # List all measures
GET  /api/v1/measures/:id                # Get measure by ID
GET  /api/v1/measures/program/:program   # Get by program
GET  /api/v1/measures/domain/:domain     # Get by domain
POST /api/v1/measures/reload             # Reload definitions
```

### Calculations
```
POST /api/v1/calculate                   # Calculate measure
POST /api/v1/calculate/batch             # Batch calculation
POST /api/v1/calculate/async             # Async calculation
GET  /api/v1/calculate/job/:id           # Get job status
```

### Reports
```
GET /api/v1/reports/:id                  # Get report
GET /api/v1/reports/measure/:measureId   # Get measure reports
GET /api/v1/reports/latest/:measureId    # Get latest report
```

### Care Gaps
```
GET  /api/v1/care-gaps/patient/:id       # Get patient care gaps
PUT  /api/v1/care-gaps/:id/status        # Update gap status
POST /api/v1/care-gaps/identify/:measureId  # Identify gaps
```

### Dashboard
```
GET /api/v1/dashboard                    # Quality dashboard
GET /api/v1/dashboard/trend/:measureId   # Measure trend
GET /api/v1/dashboard/comparison         # Facility comparison
```

## Measure Definition Example

```yaml
type: measure
measure:
  id: HBD
  version: "2024"
  name: Hemoglobin A1c Control for Patients with Diabetes
  title: HbA1c Control (<8%)
  
  type: PROCESS
  scoring: proportion
  domain: DIABETES
  program: HEDIS
  
  nqf_number: "0059"
  cms_number: "CMS122v11"
  hedis_code: HBD
  
  measurement_period:
    type: rolling
    duration: P1Y
  
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
          - lab_code: "4548-4"  # HbA1c LOINC
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

## Population Criteria

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equals | `value: 8.0` |
| `>` | Greater than | `value: 7.0` |
| `<` | Less than | `value: 8.0` |
| `>=` | Greater or equal | `value: 65` |
| `<=` | Less or equal | `value: 75` |
| `between` | Range | `value: [18, 75]` |
| `in` | In list | `value: ["active", "on-hold"]` |

## Integration Map

```
┌─────────────────────────────────────────────────────────────────┐
│                    KB-13 Integration Points                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Vaidshala (CQL Engine)                                        │
│  └─ Evaluates CQL expressions for population criteria          │
│                                                                 │
│  KB-7 (Terminology Service)                                    │
│  └─ Resolves value sets for diagnosis/procedure codes          │
│                                                                 │
│  KB-19 (Protocol Orchestrator)                                 │
│  └─ Receives quality alerts for protocol adjustments           │
│                                                                 │
│  KB-18 (Governance Engine)                                     │
│  └─ Tracks quality performance for compliance reporting        │
│                                                                 │
│  Patient Service                                               │
│  └─ Provides patient clinical data for evaluation              │
│                                                                 │
│  EHR / Analytics                                               │
│  └─ Consumes reports and dashboards                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
kb-13-quality-measures/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── api/
│   │   └── server.go            # HTTP API server
│   ├── calculator/
│   │   ├── engine.go            # Calculation engine
│   │   └── cache.go             # Result caching
│   ├── config/
│   │   └── config.go            # Configuration
│   ├── database/
│   │   └── postgres.go          # Database layer
│   ├── loader/
│   │   └── loader.go            # YAML measure loader
│   ├── metrics/
│   │   └── metrics.go           # Metrics collector
│   ├── models/
│   │   ├── measure.go           # Domain models
│   │   └── store.go             # Measure store
│   └── scheduler/
│       └── scheduler.go         # Scheduled calculations
├── measures/
│   ├── hedis/                   # HEDIS measure definitions
│   │   ├── diabetes.yaml
│   │   ├── cardiovascular.yaml
│   │   └── preventive.yaml
│   └── cms/                     # CMS measure definitions
│       ├── quality.yaml
│       └── readmission.yaml
├── cql/
│   └── tier-6-application/
│       ├── QualityMeasures-1.0.0.cql
│       └── DiabetesMeasures-1.0.0.cql
├── tests/
│   └── engine_test.go           # Unit tests
├── Dockerfile
├── docker-compose.yaml
├── go.mod
└── README.md
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KB13_PORT` | 8113 | HTTP server port |
| `KB13_MEASURES_PATH` | ./measures | Path to measure definitions |
| `KB13_LOG_LEVEL` | info | Log level |
| `KB13_DB_HOST` | localhost | PostgreSQL host |
| `KB13_DB_PORT` | 5432 | PostgreSQL port |
| `KB13_DB_NAME` | kb13_quality | Database name |
| `KB13_DB_USER` | postgres | Database user |
| `KB13_DB_PASSWORD` | | Database password |
| `KB13_ENABLE_CACHING` | true | Enable result caching |
| `KB13_CACHE_TTL` | 15m | Cache TTL |
| `KB13_MAX_CONCURRENT` | 50 | Max concurrent calculations |
| `KB13_CALC_TIMEOUT` | 60s | Calculation timeout |
| `KB13_SCHEDULER_ENABLED` | false | Enable scheduler |
| `VAIDSHALA_URL` | http://localhost:8096 | CQL engine URL |
| `PATIENT_SERVICE_URL` | http://localhost:8080 | Patient service URL |

## Quick Start

```bash
# Build and run
docker-compose up -d

# Calculate a measure
curl -X POST http://localhost:8113/api/v1/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "measure_id": "HBD",
    "report_type": "individual",
    "subject_id": "patient-123",
    "period_start": "2024-01-01T00:00:00Z",
    "period_end": "2024-12-31T23:59:59Z"
  }'

# Get quality dashboard
curl http://localhost:8113/api/v1/dashboard

# List measures by program
curl http://localhost:8113/api/v1/measures/program/HEDIS

# Get care gaps for patient
curl http://localhost:8113/api/v1/care-gaps/patient/patient-123
```

## Hot Reload

Send SIGHUP to reload measure definitions:
```bash
docker kill -s HUP kb13-quality-measures
# or
curl -X POST http://localhost:8113/api/v1/measures/reload
```

## Implementation Status

| Component | Status | LOC |
|-----------|--------|-----|
| Core Engine | ✅ Complete | ~600 |
| Measure Store | ✅ Complete | ~450 |
| API Server | ✅ Complete | ~650 |
| Database | ✅ Complete | ~500 |
| YAML Loader | ✅ Complete | ~350 |
| Scheduler | ✅ Complete | ~400 |
| Metrics | ✅ Complete | ~250 |
| HEDIS Measures | ✅ Complete | ~1,000 |
| CMS Measures | ✅ Complete | ~600 |
| CQL Libraries | ✅ Complete | ~500 |
| Tests | ✅ Complete | ~450 |
| **Total** | **Complete** | **~6,000** |

## Related Knowledge Bases

| KB | Integration |
|----|-------------|
| KB-7 | Terminology resolution for value sets |
| KB-10 | Rule engine for quality alerts |
| KB-18 | Governance reporting |
| KB-19 | Protocol quality tracking |
| Vaidshala | CQL expression evaluation |

## License

Proprietary - Vaidshala Platform
