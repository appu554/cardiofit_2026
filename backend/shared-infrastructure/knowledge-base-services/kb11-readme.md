# KB-11 Population Health Engine

Comprehensive population health management platform providing patient registry, risk stratification, cohort management, and population analytics.

## Overview

KB-11 provides enterprise-grade population health capabilities including:

| Feature | Description |
|---------|-------------|
| **Patient Registry** | Centralized patient registry with clinical, demographic, and social data |
| **Risk Stratification** | Multi-model risk scoring with configurable algorithms |
| **Cohort Management** | Static and dynamic cohorts with automatic refresh |
| **Analytics Engine** | Population-level analytics, trends, and comparisons |
| **Care Program Management** | Program enrollment, eligibility, and outcome tracking |
| **SDOH Integration** | Social determinants of health screening and tracking |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       KB-11 POPULATION HEALTH ENGINE                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Patient    │  │     Risk     │  │    Cohort    │  │  Analytics   │   │
│  │   Registry   │  │   Engine     │  │   Manager    │  │   Engine     │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
│         │                 │                 │                 │           │
│  ┌──────┴─────────────────┴─────────────────┴─────────────────┴──────┐    │
│  │                         Core Services                              │    │
│  ├────────────────────────────────────────────────────────────────────┤    │
│  │  • Patient Cache           • Risk Model Loader                    │    │
│  │  • Attribution Engine      • Cohort Refresh Scheduler             │    │
│  │  • Care Gap Tracker        • Intervention Generator               │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │                      External Integrations                          │    │
│  ├────────────────────────────────────────────────────────────────────┤    │
│  │  Vaidshala │ KB-13       │ Patient    │ EHR        │ Claims       │    │
│  │  (CQL)     │ Quality     │ Service    │ System     │ Data         │    │
│  └────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Risk Models

| Model | Type | Description |
|-------|------|-------------|
| **Hospitalization** | HOSPITALIZATION | 30-day admission prediction |
| **Readmission** | READMISSION | 30-day readmission prediction |
| **ED Utilization** | ED_UTILIZATION | High ED utilizer identification |
| **Diabetes Progression** | DIABETES_PROGRESSION | Complication risk |
| **CHF Exacerbation** | CHF_EXACERBATION | Heart failure decompensation |
| **Frailty** | FRAILTY | Functional decline risk |

## Risk Tiers

| Tier | Description | Typical Actions |
|------|-------------|-----------------|
| **LOW** | Minimal risk | Standard preventive care |
| **MODERATE** | Elevated risk | Enhanced monitoring |
| **HIGH** | Significant risk | Care management enrollment |
| **VERY_HIGH** | Critical risk | Intensive care coordination |
| **RISING** | Trending upward | Proactive intervention |

## Cohort Types

| Type | Description |
|------|-------------|
| **STATIC** | Fixed membership, manually maintained |
| **DYNAMIC** | Rule-based, automatically refreshed |
| **SNAPSHOT** | Point-in-time capture of a population |

## API Endpoints

### Patients
```
GET    /api/v1/patients                    # List patients
GET    /api/v1/patients/:id                # Get patient
POST   /api/v1/patients                    # Create patient
PUT    /api/v1/patients/:id                # Update patient
DELETE /api/v1/patients/:id                # Delete patient
GET    /api/v1/patients/:id/risk           # Get patient risk scores
POST   /api/v1/patients/:id/risk/calculate # Calculate patient risk
GET    /api/v1/patients/high-risk          # Get high-risk patients
GET    /api/v1/patients/rising-risk        # Get rising-risk patients
```

### Risk
```
GET  /api/v1/risk/models                   # List risk models
GET  /api/v1/risk/models/:id               # Get risk model
POST /api/v1/risk/calculate                # Calculate risk
POST /api/v1/risk/calculate/batch          # Batch calculate
GET  /api/v1/risk/distribution             # Get risk distribution
```

### Cohorts
```
GET    /api/v1/cohorts                     # List cohorts
POST   /api/v1/cohorts                     # Create cohort
GET    /api/v1/cohorts/:id                 # Get cohort
PUT    /api/v1/cohorts/:id                 # Update cohort
DELETE /api/v1/cohorts/:id                 # Delete cohort
GET    /api/v1/cohorts/:id/patients        # Get cohort patients
POST   /api/v1/cohorts/:id/patients        # Add patient
DELETE /api/v1/cohorts/:id/patients/:pid   # Remove patient
POST   /api/v1/cohorts/:id/refresh         # Refresh cohort
POST   /api/v1/cohorts/union               # Union cohorts
POST   /api/v1/cohorts/intersect           # Intersect cohorts
```

### Analytics
```
GET  /api/v1/analytics/summary             # Population summary
GET  /api/v1/analytics/risk-distribution   # Risk analytics
GET  /api/v1/analytics/utilization         # Utilization report
GET  /api/v1/analytics/care-gaps           # Care gap report
POST /api/v1/analytics/query               # Custom query
POST /api/v1/analytics/compare-cohorts     # Compare cohorts
```

## Data Models

### Patient
```json
{
  "id": "patient-123",
  "mrn": "MRN001",
  "first_name": "John",
  "last_name": "Doe",
  "date_of_birth": "1955-03-15",
  "gender": "male",
  "address": { "city": "Boston", "state": "MA" },
  "primary_care_provider": "dr-smith",
  "current_risk_tier": "HIGH",
  "risk_scores": {
    "hospitalization-30day": { "score": 72, "tier": "HIGH" }
  },
  "active_conditions": [
    { "code": "I50.9", "display": "Heart failure, unspecified" }
  ],
  "open_care_gaps": 3
}
```

### Risk Assessment
```json
{
  "id": "uuid",
  "patient_id": "patient-123",
  "assessment_date": "2024-01-15T10:30:00Z",
  "scores": {
    "hospitalization-30day": {
      "score": 72,
      "risk_tier": "HIGH",
      "top_factors": [
        { "name": "Prior Admissions", "contribution": 28.5 }
      ]
    }
  },
  "overall_risk_tier": "HIGH",
  "interventions": [
    { "type": "care_management", "priority": "high" }
  ],
  "eligible_programs": ["complex-care-management"]
}
```

### Cohort
```json
{
  "id": "uuid",
  "name": "High-Risk Diabetes Patients",
  "type": "DYNAMIC",
  "definition": {
    "criteria": [
      { "field": "condition", "operator": "in", "value": "E11.*" },
      { "field": "risk_tier", "operator": "in", "value": ["HIGH", "VERY_HIGH"] }
    ],
    "logic": "AND"
  },
  "patient_count": 1250,
  "refresh_schedule": "daily",
  "statistics": {
    "avg_age": 62.5,
    "gender_distribution": { "male": 680, "female": 570 }
  }
}
```

## Directory Structure

```
kb-11-population-health/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── api/server.go               # HTTP API
│   ├── analytics/engine.go         # Analytics engine
│   ├── cohort/manager.go           # Cohort management
│   ├── config/config.go            # Configuration
│   ├── database/postgres.go        # Database layer
│   ├── metrics/metrics.go          # Metrics collector
│   ├── models/models.go            # Domain models
│   ├── registry/registry.go        # Patient registry
│   └── risk/engine.go              # Risk stratification
├── models/
│   └── hospitalization-risk.yaml   # Risk model definitions
├── Dockerfile
├── docker-compose.yaml
├── go.mod
└── README.md
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KB11_PORT` | 8111 | HTTP server port |
| `KB11_LOG_LEVEL` | info | Log level |
| `KB11_DB_HOST` | localhost | PostgreSQL host |
| `KB11_DB_PORT` | 5432 | PostgreSQL port |
| `KB11_DB_NAME` | kb11_population | Database name |
| `KB11_CACHE_ENABLED` | true | Enable patient cache |
| `KB11_CACHE_TTL` | 15m | Cache TTL |
| `KB11_MAX_CONCURRENT` | 50 | Max concurrent calculations |
| `KB11_RISK_MODELS_PATH` | ./models | Risk models directory |
| `KB11_MAX_COHORT_SIZE` | 100000 | Max cohort size |
| `KB11_COHORT_REFRESH` | 1h | Cohort refresh interval |
| `VAIDSHALA_URL` | http://localhost:8096 | CQL engine URL |

## Quick Start

```bash
# Build and run
docker-compose up -d

# Get population summary
curl http://localhost:8111/api/v1/analytics/summary

# Calculate risk for a patient
curl -X POST http://localhost:8111/api/v1/patients/patient-123/risk/calculate

# Create a dynamic cohort
curl -X POST http://localhost:8111/api/v1/cohorts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High-Risk CHF",
    "type": "DYNAMIC",
    "definition": {
      "criteria": [
        {"field": "condition", "operator": "contains", "value": "I50"},
        {"field": "risk_tier", "value": "HIGH"}
      ]
    },
    "refresh_schedule": "daily"
  }'

# Compare cohorts
curl -X POST http://localhost:8111/api/v1/analytics/compare-cohorts \
  -H "Content-Type: application/json" \
  -d '{
    "cohort_ids": ["uuid1", "uuid2"],
    "metrics": ["risk_distribution", "utilization"]
  }'
```

## Integration Map

```
┌─────────────────────────────────────────────────────────────────┐
│                    KB-11 Integration Points                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Vaidshala (CQL Engine)                                        │
│  └─ Evaluates CQL expressions for complex risk calculations    │
│                                                                 │
│  KB-13 (Quality Measures)                                      │
│  └─ Receives care gap data, shares quality scores              │
│                                                                 │
│  KB-7 (Terminology Service)                                    │
│  └─ Resolves value sets for condition filtering                │
│                                                                 │
│  Patient Service                                               │
│  └─ Provides clinical data for risk calculation                │
│                                                                 │
│  EHR / ADT Feeds                                               │
│  └─ Real-time patient data updates                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Status

| Component | Status | LOC |
|-----------|--------|-----|
| Patient Registry | ✅ Complete | ~400 |
| Risk Engine | ✅ Complete | ~550 |
| Cohort Manager | ✅ Complete | ~400 |
| Analytics Engine | ✅ Complete | ~450 |
| API Server | ✅ Complete | ~500 |
| Database Layer | ✅ Complete | ~500 |
| Domain Models | ✅ Complete | ~550 |
| Metrics | ✅ Complete | ~250 |
| Risk Models | ✅ Complete | ~300 |
| **Total** | **Complete** | **~4,500** |

## License

Proprietary - Vaidshala Platform
