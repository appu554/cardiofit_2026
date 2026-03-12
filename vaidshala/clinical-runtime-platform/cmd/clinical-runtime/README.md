# Clinical Runtime Platform - Full Orchestrator

Production-ready HTTP server exposing the **COMPLETE ENGINE FLOW** with FDA SaMD compliant 3-Phase API.

## Quick Start

```bash
# Build
cd /Users/apoorvabk/Downloads/cardiofit/vaidshala/clinical-runtime-platform
go build -o bin/clinical-runtime ./cmd/clinical-runtime/

# Run
./bin/clinical-runtime

# Server starts on port 8090
```

## Server Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PORT` | `8090` | Server port |
| `ENVIRONMENT` | `development` | Environment mode |
| `REGION` | `AU` | Default region (AU, IN, US) |
| `KB2_URL` | `http://localhost:8082` | KB-2 Clinical Context URL |
| `KB6_URL` | `http://localhost:8087` | KB-6 Formulary URL |
| `KB7_URL` | `http://localhost:8092` | KB-7 Terminology URL |
| `KB8_URL` | `http://localhost:8097` | KB-8 Calculator URL |

## Engines Loaded

| Engine | Purpose | Output |
|--------|---------|--------|
| **CQL Engine** | Clinical truth determination | `ClinicalFacts` |
| **Measure Engine** | Care accountability (CMS measures) | `MeasureResults`, `CareGaps` |
| **Medication Engine** | Drug recommendations | `Recommendations`, `Alerts` |

---

## 3-Phase API (FDA SaMD Compliant)

### Phase 1: CALCULATE
**Runs ALL engines, returns recommendations (NO patient changes)**

```bash
POST /v1/calculate
```

**Request:**
```json
{
  "patient_id": "patient-001",
  "patient_data": {
    "demographics": {
      "birth_date": "1970-03-15",
      "gender": "male",
      "region": "AU"
    },
    "conditions": [
      {
        "code": "44054006",
        "system": "http://snomed.info/sct",
        "display": "Type 2 diabetes mellitus",
        "clinical_status": "active"
      }
    ],
    "medications": [
      {
        "code": "6809",
        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
        "display": "Metformin 500mg",
        "status": "active",
        "dose_value": 500,
        "dose_unit": "mg"
      }
    ],
    "lab_results": [
      {
        "code": "4548-4",
        "system": "http://loinc.org",
        "display": "Hemoglobin A1c",
        "value": 9.5,
        "unit": "%",
        "timestamp": "2026-01-01T00:00:00Z"
      }
    ],
    "vital_signs": [
      {
        "systolic_bp": 150,
        "diastolic_bp": 95,
        "timestamp": "2026-01-09T00:00:00Z"
      }
    ],
    "encounters": [
      {
        "encounter_id": "enc-001",
        "class": "ambulatory",
        "status": "finished"
      }
    ]
  },
  "requested_by": "dr.smith@hospital.com"
}
```

**Response:**
```json
{
  "session_id": "a8619af1-8ad4-4407-9f7a-62956212623a",
  "success": true,
  "engine_results": [
    {"engine_name": "cql-engine", "success": true, "facts_produced": 16},
    {"engine_name": "measure-engine", "success": true, "recommendations_produced": 1},
    {"engine_name": "medication-advisor", "success": true}
  ],
  "clinical_facts": [...],
  "measure_results": [...],
  "recommendations": [...],
  "care_gaps": [...],
  "next_step": "POST /v1/validate with session_id: a8619af1-..."
}
```

---

### Phase 2: VALIDATE
**Clinician reviews and approves recommendations (NO patient changes)**

```bash
POST /v1/validate
```

**Request:**
```json
{
  "session_id": "a8619af1-8ad4-4407-9f7a-62956212623a",
  "validated_by": "dr.smith@hospital.com",
  "decisions": [
    {
      "recommendation_id": "REC-CMS2-CARE-GAP",
      "approved": true,
      "reason": "Patient needs depression screening"
    }
  ],
  "clinical_notes": "Approved PHQ-9 screening for this patient"
}
```

**Response:**
```json
{
  "session_id": "a8619af1-8ad4-4407-9f7a-62956212623a",
  "status": "validated",
  "validated_by": "dr.smith@hospital.com",
  "validated_at": "2026-01-09T12:11:28Z",
  "approved_count": 1,
  "rejected_count": 0,
  "approved_actions": ["Care Gap: Depression Screening"],
  "next_step": "POST /v1/commit with session_id: a8619af1-..."
}
```

---

### Phase 3: COMMIT
**Executes approved actions, creates FHIR resources (MAKES patient changes)**

```bash
POST /v1/commit
```

**Request:**
```json
{
  "session_id": "a8619af1-8ad4-4407-9f7a-62956212623a",
  "committed_by": "dr.smith@hospital.com",
  "confirmed": true
}
```

**Response:**
```json
{
  "session_id": "a8619af1-8ad4-4407-9f7a-62956212623a",
  "status": "committed",
  "committed_by": "dr.smith@hospital.com",
  "committed_at": "2026-01-09T12:15:00Z",
  "actions_committed": 1,
  "fhir_resources": [
    {
      "resource_type": "Task",
      "resource_id": "uuid-generated",
      "action": "created"
    }
  ],
  "audit_trail": {
    "session_id": "a8619af1-...",
    "patient_id": "patient-001",
    "requested_by": "dr.smith@hospital.com",
    "requested_at": "2026-01-09T12:10:00Z",
    "validated_by": "dr.smith@hospital.com",
    "validated_at": "2026-01-09T12:11:28Z",
    "committed_by": "dr.smith@hospital.com",
    "committed_at": "2026-01-09T12:15:00Z",
    "engines_run": ["cql-engine", "measure-engine", "medication-engine"],
    "recommendations_total": 1,
    "approved": 1,
    "rejected": 0,
    "committed": 1
  }
}
```

---

## Other Endpoints

### Health Check
```bash
GET /health

# Response:
{
  "status": "healthy",
  "service": "clinical-runtime-platform",
  "engines": {
    "cql_engine": "cql-engine",
    "measure_engine": "measure-engine",
    "medication_engine": "medication-advisor"
  }
}
```

### Readiness Check
```bash
GET /ready

# Response:
{"ready": true}
```

### Get Session Status
```bash
GET /v1/session/{session_id}

# Response:
{
  "session_id": "...",
  "patient_id": "patient-001",
  "status": "validated",
  "recommendations": 1,
  "care_gaps": 1
}
```

### Delete Session
```bash
DELETE /v1/session/{session_id}
```

---

## Complete Test Flow

```bash
# 1. Start the server
./bin/clinical-runtime

# 2. Check health
curl http://localhost:8090/health

# 3. Phase 1: Calculate (save session_id from response)
curl -X POST http://localhost:8090/v1/calculate \
  -H "Content-Type: application/json" \
  -d @test_patient.json

# 4. Phase 2: Validate (use session_id from step 3)
curl -X POST http://localhost:8090/v1/validate \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "SESSION_ID_FROM_STEP_3",
    "validated_by": "dr.smith@hospital.com",
    "decisions": [{"recommendation_id": "REC-CMS2-CARE-GAP", "approved": true}]
  }'

# 5. Phase 3: Commit (use same session_id)
curl -X POST http://localhost:8090/v1/commit \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "SESSION_ID_FROM_STEP_3",
    "committed_by": "dr.smith@hospital.com",
    "confirmed": true
  }'
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    FULL ORCHESTRATOR HTTP API (Port 8090)                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  POST /v1/calculate    ──→  CQL Engine ──→ Measure Engine ──→ Medication   │
│                              (truths)       (care gaps)       (drugs)       │
│                                                                             │
│  POST /v1/validate     ──→  Clinician Review (Human-in-the-loop)           │
│                                                                             │
│  POST /v1/commit       ──→  Create FHIR Resources + Audit Trail            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## FDA SaMD Compliance

- **Human-in-the-loop**: AI recommendations require clinician approval before execution
- **Audit trail**: Complete traceability (who requested, validated, committed + timestamps)
- **Immutable snapshots**: Clinical context frozen at calculate time
- **Session expiry**: Sessions expire after 30 minutes if not validated

## Session States

| State | Description |
|-------|-------------|
| `pending_validation` | Calculate complete, awaiting clinician review |
| `validated` | Clinician approved, ready to commit |
| `committed` | Actions executed, FHIR resources created |
| `expired` | Session timed out (30 min) |
