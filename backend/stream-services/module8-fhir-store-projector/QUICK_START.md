# FHIR Store Projector - Quick Start Guide

## 30-Second Setup

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services/module8-fhir-store-projector

# 1. Install dependencies
pip install -r requirements.txt

# 2. Configure environment
cp .env.example .env
# Edit .env with your Kafka and Google Cloud credentials

# 3. Validate setup
python3 validate_setup.py

# 4. Run service
python3 run.py
```

## First Test

```bash
# In another terminal, check health
curl http://localhost:8056/health

# Expected response:
{
  "status": "healthy",
  "service": "fhir-store-projector",
  "fhir_store": {
    "project_id": "cardiofit-905a8",
    "location": "us-central1",
    "dataset_id": "cardiofit_fhir_dataset",
    "store_id": "cardiofit_fhir_store"
  },
  "stats": {
    "total_upserts": 0,
    "success_rate": 0.0
  }
}
```

## Sample FHIR Resource Write

The projector consumes from `prod.ehr.fhir.upsert` in this format:

```json
{
  "resourceType": "Observation",
  "resourceId": "obs-12345",
  "patientId": "patient-67890",
  "lastUpdated": 1731706800000,
  "fhirData": {
    "resourceType": "Observation",
    "id": "obs-12345",
    "status": "final",
    "category": [{
      "coding": [{
        "system": "http://terminology.hl7.org/CodeSystem/observation-category",
        "code": "vital-signs",
        "display": "Vital Signs"
      }]
    }],
    "code": {
      "coding": [{
        "system": "http://loinc.org",
        "code": "8867-4",
        "display": "Heart rate"
      }],
      "text": "Heart Rate"
    },
    "subject": {
      "reference": "Patient/patient-67890"
    },
    "effectiveDateTime": "2025-11-15T21:00:00Z",
    "valueQuantity": {
      "value": 72,
      "unit": "beats/minute",
      "system": "http://unitsofmeasure.org",
      "code": "/min"
    }
  }
}
```

## Monitoring

```bash
# Check metrics
curl http://localhost:8056/metrics | jq

# View logs (in service terminal)
# JSON structured logs show each operation

# Reset stats (for testing)
curl http://localhost:8056/stats/reset
```

## Resource Type Examples

### 1. RiskAssessment (Sepsis Prediction)

```json
{
  "resourceType": "RiskAssessment",
  "resourceId": "risk-sepsis-123",
  "patientId": "patient-67890",
  "lastUpdated": 1731706800000,
  "fhirData": {
    "resourceType": "RiskAssessment",
    "id": "risk-sepsis-123",
    "status": "final",
    "subject": {"reference": "Patient/patient-67890"},
    "occurrenceDateTime": "2025-11-15T21:00:00Z",
    "prediction": [{
      "outcome": {
        "coding": [{
          "system": "http://snomed.info/sct",
          "code": "91302008",
          "display": "Sepsis"
        }],
        "text": "Sepsis Risk"
      },
      "probabilityDecimal": 0.35,
      "whenPeriod": {
        "start": "2025-11-15T21:00:00Z",
        "end": "2025-11-16T21:00:00Z"
      }
    }]
  }
}
```

### 2. Condition (Hypertension)

```json
{
  "resourceType": "Condition",
  "resourceId": "cond-htn-456",
  "patientId": "patient-67890",
  "lastUpdated": 1731706800000,
  "fhirData": {
    "resourceType": "Condition",
    "id": "cond-htn-456",
    "clinicalStatus": {
      "coding": [{
        "system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
        "code": "active"
      }]
    },
    "verificationStatus": {
      "coding": [{
        "system": "http://terminology.hl7.org/CodeSystem/condition-ver-status",
        "code": "confirmed"
      }]
    },
    "code": {
      "coding": [{
        "system": "http://snomed.info/sct",
        "code": "38341003",
        "display": "Hypertension"
      }],
      "text": "Hypertension"
    },
    "subject": {"reference": "Patient/patient-67890"},
    "onsetDateTime": "2025-01-01T00:00:00Z"
  }
}
```

## Verification in Google Cloud

```bash
# Search for resources
gcloud healthcare fhir-stores search cardiofit_fhir_store \
  --dataset=cardiofit_fhir_dataset \
  --location=us-central1 \
  --resource-type=Observation \
  --search-string="subject=Patient/patient-67890"

# Get specific resource
gcloud healthcare fhir-stores get-resource cardiofit_fhir_store \
  --dataset=cardiofit_fhir_dataset \
  --location=us-central1 \
  --resource-type=Observation \
  --resource-id=obs-12345
```

## Performance Expectations

- **Throughput**: ~200 resources/sec (limited by Google Healthcare API)
- **Latency**: 50-100ms per resource (API call)
- **Batch Size**: 20 resources (optimal for API rate limits)
- **Success Rate**: >95% (with retry logic)

## Common Issues

### 1. Credentials Error
```
Error: google.auth.exceptions.DefaultCredentialsError
Solution: Verify credentials file exists and is valid JSON
```

### 2. Permission Denied
```
Error: 403 Permission Denied
Solution: Grant serviceAccount roles/healthcare.fhirResourceEditor
```

### 3. FHIR Store Not Found
```
Error: 404 Not Found
Solution: Create FHIR Store in Google Cloud Console or with gcloud CLI
```

### 4. Validation Error
```
Error: Unsupported resource type
Solution: Check resource type is in SUPPORTED_RESOURCE_TYPES list
```

## Architecture Flow

```
Device Data → Stage 1 (Java) → Stage 2 (Python) → Enrichment
                                                        ↓
                                              Module 6 FHIR Transform
                                                        ↓
                                              prod.ehr.fhir.upsert
                                                        ↓
                                           [FHIR Store Projector] ← You are here
                                                        ↓
                                        Google Cloud Healthcare API
                                                        ↓
                                                  FHIR Store
                                          (HIPAA-compliant storage)
```

## Key Files Reference

| File | Purpose |
|------|---------|
| `app/services/fhir_store_handler.py` | Google Healthcare API integration |
| `app/services/projector.py` | Kafka consumer and batch processor |
| `app/config.py` | Environment configuration |
| `app/main.py` | Health and metrics endpoints |
| `run.py` | Service launcher (Kafka + FastAPI) |
| `test_projector.py` | Integration tests |
| `validate_setup.py` | Setup validation script |
| `.env` | Configuration (create from .env.example) |
| `credentials/google-credentials.json` | Service account credentials |

## Support

For issues or questions, refer to:
- **Full Documentation**: `README.md`
- **Delivery Confirmation**: `DELIVERY_CONFIRMATION.md`
- **Module 8 Architecture**: `/Users/apoorvabk/Downloads/cardiofit/backend/stream-services/MODULE_8_HYBRID_ARCHITECTURE_IMPLEMENTATION_PLAN.md`
