# KB-17: Population Registry Service

A comprehensive disease registry management service with Kafka-driven auto-enrollment, criteria-based eligibility evaluation, and risk stratification.

## Overview

KB-17 Population Registry Service provides:

- **Disease Registries**: Pre-configured registries for Diabetes, Hypertension, Heart Failure, CKD, COPD, Pregnancy, Opioid Use, Anticoagulation, and more
- **Auto-Enrollment**: Kafka event-driven automatic patient enrollment based on diagnoses, lab results, and medications
- **Criteria Engine**: Flexible inclusion/exclusion criteria evaluation
- **Risk Stratification**: Rules-based and score-based patient risk tiering
- **Care Gap Integration**: Links to KB-9 Quality Measures for care gap tracking
- **Event Production**: Downstream events for KB-14 Task creation and KB-18 Governance

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        KB-17 Population Registry                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                │
│  │   Registry   │     │   Criteria   │     │   Patient    │                │
│  │ Definitions  │────▶│   Engine     │────▶│    Store     │                │
│  │              │     │              │     │              │                │
│  │ • Diabetes   │     │ • Inclusion  │     │ • Enrollments│                │
│  │ • HTN        │     │ • Exclusion  │     │ • Indexes    │                │
│  │ • HF         │     │ • Risk Rules │     │ • History    │                │
│  │ • CKD        │     │              │     │              │                │
│  │ • ...        │     │              │     │              │                │
│  └──────────────┘     └──────────────┘     └──────────────┘                │
│                              ▲                    │                         │
│                              │                    ▼                         │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐                │
│  │    Kafka     │────▶│   Event      │     │    Event     │───▶ KB-14     │
│  │   Consumer   │     │  Consumer    │     │   Producer   │───▶ KB-18     │
│  │              │     │              │     │              │───▶ KB-9      │
│  │ • Diagnosis  │     │ • Auto-enroll│     │ • Enrolled   │                │
│  │ • Lab Result │     │ • Evaluate   │     │ • Risk Chg   │                │
│  │ • Medication │     │ • Disenroll  │     │ • Care Gaps  │                │
│  └──────────────┘     └──────────────┘     └──────────────┘                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Pre-Configured Registries

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

## Auto-Enrollment Events

The service listens for Kafka events to automatically evaluate and enroll patients:

```go
// Event types supported
EventTypeDiagnosisCreated    // ICD-10 diagnosis → registry evaluation
EventTypeLabResultCreated    // Lab result → registry evaluation
EventTypeMedicationStarted   // Medication → registry evaluation
EventTypeProblemAdded        // Problem list → registry evaluation
```

### Example: Diabetes Auto-Enrollment Flow

```
1. Kafka Event: diagnosis.created (E11.9 - Type 2 DM)
2. Consumer receives event for patient P001
3. Criteria Engine evaluates against Diabetes Registry
4. Patient meets inclusion criteria
5. Risk tier determined (HbA1c-based if available)
6. Patient enrolled in DIABETES registry
7. Event produced: registry.enrolled
8. KB-14 creates "Initial DM Assessment" task
```

## API Endpoints

### Registries
```
GET    /api/v1/registries              # List all registries
GET    /api/v1/registries/{code}       # Get registry definition
POST   /api/v1/registries              # Create custom registry
GET    /api/v1/registries/{code}/patients  # Get registry patients
```

### Enrollments
```
GET    /api/v1/enrollments             # Query enrollments
POST   /api/v1/enrollments             # Manual enrollment
GET    /api/v1/enrollments/{id}        # Get enrollment details
PUT    /api/v1/enrollments/{id}        # Update enrollment
DELETE /api/v1/enrollments/{id}        # Disenroll
POST   /api/v1/enrollments/bulk        # Bulk enrollment
```

### Patient-Centric
```
GET    /api/v1/patients/{id}/registries           # Patient's registries
GET    /api/v1/patients/{id}/enrollment/{code}    # Specific enrollment
```

### Criteria Evaluation
```
POST   /api/v1/evaluate                # Evaluate patient eligibility
```

### Analytics
```
GET    /api/v1/stats                   # All registry statistics
GET    /api/v1/stats/{code}            # Registry-specific stats
GET    /api/v1/high-risk               # High-risk patients
GET    /api/v1/care-gaps               # Patients with care gaps
```

### Events
```
POST   /api/v1/events                  # Process clinical event
```

## Data Models

### RegistryPatient
```go
type RegistryPatient struct {
    ID               string           // Unique enrollment ID
    RegistryCode     RegistryCode     // DIABETES, HYPERTENSION, etc.
    PatientID        string           // Patient identifier
    Status           EnrollmentStatus // ACTIVE, PENDING, DISENROLLED
    EnrollmentSource EnrollmentSource // DIAGNOSIS, LAB_RESULT, MANUAL
    RiskTier         RiskTier         // LOW, MODERATE, HIGH, CRITICAL
    Metrics          map[string]*MetricValue  // Key metrics
    CareGaps         []string         // Active care gaps
    // ... additional fields
}
```

### CriteriaEvaluationResult
```go
type CriteriaEvaluationResult struct {
    PatientID         string
    RegistryCode      RegistryCode
    MeetsInclusion    bool
    MeetsExclusion    bool
    Eligible          bool
    SuggestedRiskTier RiskTier
    MatchedCriteria   []string
    // ... additional fields
}
```

## Integration Points

### Upstream (Consumes From)
- **Kafka**: Clinical events (diagnosis, lab, medication)
- **KB-2**: Patient context for criteria evaluation
- **KB-8**: Risk scores for stratification
- **KB-11**: Population attribution

### Downstream (Produces To)
- **KB-9**: Care gap updates
- **KB-14**: Task creation for new enrollments
- **KB-15**: Patient outreach triggers
- **KB-18**: Governance enforcement inputs

## Running the Service

### Local Development
```bash
# Build
go build -o kb17-server ./cmd/server

# Run
./kb17-server -port 8017

# With Kafka
./kb17-server -port 8017 -kafka localhost:9092
```

### Docker
```bash
docker build -t kb17-population-registry .
docker run -p 8017:8017 kb17-population-registry
```

### Environment Variables
```
KB17_PORT=8017
KB17_KAFKA_BROKERS=localhost:9092
KB17_KB2_URL=http://localhost:8002
KB17_KB9_URL=http://localhost:8009
KB17_KB14_URL=http://localhost:8014
```

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test file
go test ./tests/criteria_test.go

# Run with verbose output
go test -v ./tests/...
```

## Custom Registry Creation

```json
POST /api/v1/registries
{
  "code": "CUSTOM_PROGRAM",
  "name": "Custom Care Program",
  "description": "Custom registry for specific program",
  "category": "Custom",
  "autoEnroll": true,
  "inclusionCriteria": [
    {
      "id": "custom-1",
      "operator": "OR",
      "criteria": [
        {
          "type": "DIAGNOSIS",
          "field": "code",
          "operator": "STARTS_WITH",
          "value": "Z99",
          "codeSystem": "ICD-10"
        }
      ]
    }
  ],
  "riskStratification": {
    "method": "rules",
    "rules": [
      {
        "tier": "HIGH",
        "priority": 1,
        "criteria": [...]
      }
    ]
  }
}
```

## File Structure

```
kb17-population-registry/
├── cmd/
│   └── server/
│       └── main.go           # HTTP server entry point
├── pkg/
│   ├── consumer/
│   │   └── kafka_consumer.go # Kafka event consumer
│   ├── criteria/
│   │   └── engine.go         # Criteria evaluation engine
│   ├── integration/
│   │   └── clients.go        # KB-2, KB-8, KB-9, KB-14, KB-16 clients
│   ├── producer/
│   │   └── event_producer.go # Registry event producer
│   ├── registry/
│   │   └── definitions.go    # Registry definitions
│   ├── server/
│   │   └── server.go         # HTTP handlers
│   ├── store/
│   │   └── patient_store.go  # Patient enrollment storage
│   └── types/
│       └── types.go          # Data models
├── tests/
│   ├── criteria_test.go      # Criteria engine tests
│   ├── consumer_test.go      # Kafka consumer tests
│   ├── store_test.go         # Store operation tests
│   └── server_test.go        # HTTP endpoint tests
├── go.mod
├── Dockerfile
└── README.md
```

## Metrics & Monitoring

The service exposes the following metrics:

- `registry_enrollments_total`: Total enrollments by registry
- `registry_disenrollments_total`: Total disenrollments
- `registry_events_processed`: Kafka events processed
- `registry_criteria_evaluations`: Criteria evaluations performed
- `registry_high_risk_patients`: Current high/critical risk count

## License

Proprietary - Healthcare Platform Services

## Version

KB-17 Population Registry Service v1.0.0
