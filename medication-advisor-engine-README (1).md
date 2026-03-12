# MedicationAdvisorEngine

**Phase 3 Medication Recommendation Engine with Full KB Orchestration**

The primary CDSS orchestrator for the Clinical Knowledge Platform, integrating Recipe Resolution, Snapshot management, and Evidence Envelope for production-grade clinical decision support.

## Overview

MedicationAdvisorEngine is a **Tier 6 Application Engine** that:

1. **Receives** enriched context from Phase 2 (patient data with computed scores)
2. **Orchestrates** KB-1 through KB-6 in a structured 3-phase workflow
3. **Produces** ranked medication recommendations with full audit trail
4. **Supports** Calculate → Validate → Commit workflow for safe ordering

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         MEDICATION ADVISOR ENGINE                            │
│                                                                             │
│  ┌─────────────┐    ┌─────────────────────────────────────┐    ┌─────────┐ │
│  │   INPUT     │    │           PHASE 3 PROCESSING         │    │ OUTPUT  │ │
│  │             │    │                                     │    │         │ │
│  │ Patient     │    │  3a: Candidate Generation           │    │Proposal │ │
│  │ Context     │───▶│      KB-3 Guidelines → KB-4 Safety  │───▶│         │ │
│  │             │    │                                     │    │Snapshot │ │
│  │ Computed    │    │  3b: Dose Calculation               │    │         │ │
│  │ Scores      │    │      KB-1 Dosing + KB-2 Interactions│    │Evidence │ │
│  │             │    │                                     │    │Envelope │ │
│  │ Clinical    │    │  3c: Scoring & Ranking              │    │         │ │
│  │ Question    │    │      KB-5 Monitoring + KB-6 Efficacy│    │         │ │
│  └─────────────┘    └─────────────────────────────────────┘    └─────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Features

### 1. Recipe Resolution

Declarative data fetching specifications that ensure consistent, reproducible data retrieval.

```go
// Example: SGLT2i Evaluation Recipe
recipe := recipe.SGLT2iEvaluationRecipe()
// Required: demographics, conditions, medications, allergies, creatinine, HbA1c
// Computed: eGFR, CKD stage, BMI
// Validations: creatinine within 90 days, eGFR calculable
```

**Benefits:**
- Consistent data requirements across the platform
- Validation of data completeness before processing
- Recipe inheritance for specialized use cases
- Full documentation of data dependencies

### 2. Snapshot Management

Version tracking for the Calculate → Validate → Commit workflow.

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  CALCULATE   │────▶│   VALIDATE   │────▶│    COMMIT    │
│              │     │              │     │              │
│ POST /process│     │POST /validate│     │ POST /commit │
│              │     │              │     │              │
│ Returns:     │     │ Checks:      │     │ Creates:     │
│ - Proposal   │     │ - Conflicts  │     │ - FHIR Rx    │
│ - SnapshotID │     │ - Expiration │     │ - Audit Log  │
└──────────────┘     └──────────────┘     └──────────────┘
```

**Features:**
- Exact versioning of all FHIR resources used
- Conflict detection before commit (hard vs soft conflicts)
- Snapshot expiration (default 1 hour TTL)
- Support for recalculation on conflict

### 3. Evidence Envelope

Complete audit trail for every clinical decision.

```
Evidence Envelope
├── Input Evidence
│   ├── Demographics, Conditions, Medications, Labs
│   ├── Computed Scores (from KB-8)
│   └── Clinical Question
├── Processing Evidence
│   ├── Phase 3a: Guidelines query, Safety check, Exclusions
│   ├── Phase 3b: Dose calculations, Interactions
│   └── Phase 3c: Scoring details, Final ranking
├── Decision Evidence
│   ├── Primary recommendation with rationale
│   ├── Alternatives with score breakdown
│   └── All alerts generated
└── Metadata & Signatures
    ├── Engine version, KB versions, CQL libraries
    └── Content hash for integrity verification
```

**Enables:**
- Regulatory compliance (SaMD IIa)
- "Why was X excluded?" queries
- "Why was Y ranked lower?" explanations
- Full reproducibility of calculations

## Architecture

### Knowledge Base Integration

| KB | Purpose | Phase | Pattern |
|----|---------|-------|---------|
| KB-1 | Dosing Rules | 3b | ATOMIC |
| KB-2 | Drug Interactions | 3b | QUERY_BASED |
| KB-3 | Clinical Guidelines | 3a | QUERY_BASED |
| KB-4 | Safety/Contraindications | 3a | QUERY_BASED |
| KB-5 | Monitoring Protocols | 3c | QUERY_BASED |
| KB-6 | Efficacy Evidence | 3c | QUERY_BASED |

### Scoring Dimensions

| Dimension | Weight | Source | Description |
|-----------|--------|--------|-------------|
| Guideline | 30% | KB-3 | Evidence-based recommendation strength |
| Safety | 25% | KB-4 | Contraindication and warning assessment |
| Efficacy | 20% | KB-6 | Clinical trial evidence quality |
| Interaction | 15% | KB-2 | Drug-drug interaction severity |
| Monitoring | 10% | KB-5 | Monitoring complexity (simpler = higher) |

## API Reference

### Calculate - POST /api/v1/process

Generate medication recommendations.

**Request:**
```json
{
  "patientId": "patient-123",
  "clinicalQuestion": {
    "text": "Add SGLT2 inhibitor for diabetes with CKD",
    "intent": "ADD_MEDICATION",
    "targetDrugClass": "SGLT2i"
  },
  "patientContext": {
    "age": 72,
    "sex": "male",
    "conditions": [
      {"code": "44054006", "display": "Type 2 Diabetes", "status": "active"}
    ],
    "currentMedications": [],
    "allergies": [],
    "computedScores": {
      "egfr": 42.5,
      "ckdStage": "G3b",
      "requiresRenalDoseAdjustment": true
    }
  },
  "requestId": "req-001"
}
```

**Response:**
```json
{
  "requestId": "req-001",
  "proposalId": "prop_20251129_143052_abc123",
  "snapshotId": "snap_20251129_143052_xyz789",
  "primaryRecommendation": {
    "rank": 1,
    "rxNormCode": "1545653",
    "drugName": "Empagliflozin",
    "drugClass": "SGLT2i",
    "dose": "10mg",
    "frequency": "once daily",
    "route": "oral",
    "score": 91.5,
    "scoreBreakdown": {
      "guideline": 95,
      "safety": 90,
      "efficacy": 95,
      "interaction": 100,
      "monitoring": 85
    },
    "rationale": "Recommended by KDIGO (2024) - STRONG recommendation",
    "guidelineRef": {
      "name": "KDIGO 2024 Clinical Practice Guideline for Diabetes Management in CKD",
      "organization": "Kidney Disease: Improving Global Outcomes",
      "year": 2024,
      "strength": "STRONG"
    }
  },
  "alternatives": [...],
  "alerts": [...],
  "monitoringPlan": [...],
  "evidenceEnvelopeId": "env_20251129_143055_def456",
  "processingTimeMs": 245,
  "confidence": 0.915
}
```

### Validate - POST /api/v1/validate

Check snapshot for conflicts before commit.

**Request:**
```json
{
  "snapshotId": "snap_20251129_143052_xyz789",
  "proposalId": "prop_20251129_143052_abc123"
}
```

**Response:**
```json
{
  "snapshotId": "snap_20251129_143052_xyz789",
  "valid": true,
  "expired": false,
  "expiresAt": "2025-11-29T15:30:52Z",
  "validationErrors": [],
  "warnings": [],
  "recommendation": "PROCEED"
}
```

### Commit - POST /api/v1/commit

Commit the recommendation (creates FHIR MedicationRequest).

**Request:**
```json
{
  "snapshotId": "snap_20251129_143052_xyz789",
  "proposalId": "prop_20251129_143052_abc123",
  "userId": "prescriber-001",
  "overrides": []
}
```

### Explain - POST /api/v1/explain

Query evidence envelope for explanations.

**Request:**
```json
{
  "evidenceId": "env_20251129_143055_def456",
  "question": "whyExcluded",
  "drugCode": "1158517"
}
```

**Response:**
```json
{
  "evidenceId": "env_20251129_143055_def456",
  "question": "whyExcluded",
  "explanation": {
    "drug": "Canagliflozin",
    "reason": "eGFR 42.5 < 60 - maximum dose 100mg",
    "source": "KB-4",
    "severity": "RELATIVE"
  }
}
```

## Quick Start

### Running Locally

```bash
# Build
go build -o server ./cmd/server

# Run
./server

# Or with custom port
PORT=9000 ./server
```

### Using Docker

```bash
# Build
docker build -t medication-advisor-engine .

# Run
docker run -p 8080:8080 medication-advisor-engine
```

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/process \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "clinicalQuestion": {
      "text": "Add SGLT2i for T2DM with CKD",
      "intent": "ADD_MEDICATION",
      "targetDrugClass": "SGLT2i"
    },
    "patientContext": {
      "age": 72,
      "sex": "male",
      "conditions": [
        {"code": "44054006", "display": "Type 2 Diabetes", "status": "active"}
      ],
      "computedScores": {
        "egfr": 42.5,
        "ckdStage": "G3b"
      }
    }
  }'
```

## Directory Structure

```
medication-advisor-engine/
├── cmd/
│   └── server/
│       └── main.go           # HTTP server with all endpoints
├── pkg/
│   ├── advisor/
│   │   └── engine.go         # Main engine with Phase 3a/3b/3c
│   ├── recipe/
│   │   └── recipe.go         # Recipe Resolution system
│   ├── snapshot/
│   │   └── snapshot.go       # Snapshot management
│   ├── evidence/
│   │   └── envelope.go       # Evidence Envelope system
│   └── kbclients/
│       └── clients.go        # KB-1 through KB-6 clients
├── test/
│   └── engine_test.go        # Comprehensive tests
├── Dockerfile
├── go.mod
└── README.md
```

## Code Statistics

| Component | Lines | Description |
|-----------|-------|-------------|
| advisor/engine.go | ~1,200 | Main engine with Phase 3 orchestration |
| recipe/recipe.go | ~800 | Recipe Resolution system |
| snapshot/snapshot.go | ~650 | Snapshot management |
| evidence/envelope.go | ~1,100 | Evidence Envelope system |
| kbclients/clients.go | ~900 | KB-1 through KB-6 clients |
| cmd/server/main.go | ~500 | HTTP server |
| test/engine_test.go | ~500 | Tests |
| **Total** | **~5,650** | Production-ready Go code |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `HOST` | `0.0.0.0` | Server bind address |
| `KB1_DOSING_URL` | `http://kb1-dosing:8080` | KB-1 Dosing Service |
| `KB2_INTERACTIONS_URL` | `http://kb2-interactions:8080` | KB-2 Interactions Service |
| `KB3_GUIDELINES_URL` | `http://kb3-guidelines:8080` | KB-3 Guidelines Service |
| `KB4_SAFETY_URL` | `http://kb4-safety:8080` | KB-4 Safety Service |
| `KB5_MONITORING_URL` | `http://kb5-monitoring:8080` | KB-5 Monitoring Service |
| `KB6_EFFICACY_URL` | `http://kb6-efficacy:8080` | KB-6 Efficacy Service |

## Testing

```bash
# Run all tests
go test ./test/... -v

# Run with coverage
go test ./test/... -cover -coverprofile=coverage.out

# View coverage
go tool cover -html=coverage.out
```

## SaMD Compliance

This engine supports **SaMD Class IIa** requirements:

| Requirement | Implementation |
|-------------|----------------|
| **Audit Trail** | Evidence Envelope with full provenance |
| **Traceability** | Snapshot tracks all FHIR resource versions |
| **Reproducibility** | Same inputs + same KB versions = same output |
| **Explainability** | Query API for "why" questions |
| **Validation** | Recipe validation, conflict detection |

## Integration Points

### With Phase 2 (Context Assembly)
- Receives `PatientContext` with pre-computed scores from KB-8
- eGFR, BMI, CKD stage already calculated - NOT recalculated

### With Phase 4 (Output)
- Produces `Proposal` with ranked recommendations
- Includes alerts, monitoring plan, full provenance
- Snapshot enables safe commit workflow

### With FHIR Server
- Snapshot tracks resource versions for conflict detection
- Commit creates MedicationRequest resource

## Next Steps

1. **Implement remaining engines:**
   - ConditionsAdvisorEngine (KB-9 care gaps)
   - ScribeValidatorEngine (KB-10, KB-7)
   - CDIQueryEngine (KB-11, KB-7)

2. **Add GraphQL layer** using gqlgen

3. **Deploy to Kubernetes** with Helm charts

4. **Integrate with Flink** for streaming use cases

## License

Proprietary - Clinical Knowledge Platform
