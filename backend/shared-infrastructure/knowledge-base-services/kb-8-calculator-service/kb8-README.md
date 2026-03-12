# KB-8 Calculator Service

**Clinical calculator microservice for the Clinical Knowledge Platform**

[![SaMD Classification](https://img.shields.io/badge/SaMD-Class%20IIa-blue)](docs/samd-compliance.md)
[![CQL Version](https://img.shields.io/badge/CQL-1.5-green)](docs/cql-integration.md)
[![Apollo Federation](https://img.shields.io/badge/Apollo-Federation%20v2-purple)](docs/graphql.md)

## Overview

KB-8 Calculator Service provides standardized clinical score calculations for the Clinical Knowledge Platform. It integrates with the 4-phase medication workflow to:

- **Phase 2 Context Assembly**: Enrich `CompleteContextPayload` with computed scores (eGFR, SOFA, BMI)
- **Phase 3b Dose Calculation**: Provide eGFR for renal dose adjustments
- **Real-time Monitoring**: Stream score updates via Flink integration

## Supported Calculators

| Calculator | Use Case | Score Range | Reference |
|------------|----------|-------------|-----------|
| **SOFA** | ICU mortality prediction | 0-24 | Vincent JL, 1996 |
| **qSOFA** | Bedside sepsis screening | 0-3 | Singer M, 2016 |
| **CHA₂DS₂-VASc** | AFib stroke risk | 0-9 | Lip GY, 2010 |
| **eGFR (CKD-EPI 2021)** | Kidney function | >0 mL/min/1.73m² | Inker LA, 2021 |
| **BMI** | Body mass index | >0 kg/m² | WHO |
| **Corrected Calcium** | Albumin-adjusted calcium | mg/dL | Payne RB, 1973 |
| **Anion Gap** | Metabolic acidosis | mEq/L | Emmett M, 1977 |

## Quick Start

### Docker

```bash
docker build -t kb8-calculator-service .
docker run -p 8080:8080 \
  -e CQL_SERVICE_URL=http://cql-service:8080 \
  -e FHIR_SERVER_URL=http://fhir-server:8080/fhir \
  kb8-calculator-service
```

### Local Development

```bash
go mod download
go run cmd/server/main.go
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Server port |
| `CQL_SERVICE_URL` | http://cql-evaluation-service:8080 | CQL Evaluation Service URL |
| `FHIR_SERVER_URL` | http://fhir-server:8080/fhir | FHIR Server URL |
| `TERMINOLOGY_URL` | http://terminology-service:8080/fhir | Terminology Service URL |
| `ENABLE_PLAYGROUND` | true | Enable GraphQL Playground |
| `READ_TIMEOUT` | 30s | HTTP read timeout |
| `WRITE_TIMEOUT` | 30s | HTTP write timeout |

## API Reference

### REST API

#### Calculate SOFA Score

```bash
curl -X POST http://localhost:8080/api/v1/calculate/sofa \
  -H "Content-Type: application/json" \
  -d '{"patientId": "patient-123"}'
```

**Response:**
```json
{
  "total": 8,
  "respiration": {"score": 2, "dataAvailable": true},
  "coagulation": {"score": 1, "dataAvailable": true},
  "liver": {"score": 1, "dataAvailable": true},
  "cardiovascular": {"score": 2, "dataAvailable": true},
  "cns": {"score": 1, "dataAvailable": true},
  "renal": {"score": 1, "dataAvailable": true},
  "interpretation": "Moderate mortality risk (15-20%)",
  "mortalityRisk": "15-20%",
  "riskLevel": "MODERATE",
  "provenance": {
    "cqlLibrary": "ClinicalCalculatorsCommon",
    "cqlVersion": "1.0.000",
    "cqlExpression": "SOFA Total Score",
    "calculatedAt": "2025-11-29T12:00:00Z",
    "dataQuality": "COMPLETE"
  }
}
```

#### Calculate eGFR

```bash
curl -X POST http://localhost:8080/api/v1/calculate/egfr \
  -H "Content-Type: application/json" \
  -d '{"patientId": "patient-123"}'
```

**Response:**
```json
{
  "value": 42.5,
  "unit": "mL/min/1.73m²",
  "ckdStage": "G3B_MODERATE_SEVERE",
  "ckdStageDisplay": "G3b (Moderately to severely decreased)",
  "requiresRenalDoseAdjustment": true,
  "doseAdjustmentGuidance": "Review medications for renal dose adjustment (moderate impairment)",
  "equation": "CKD-EPI 2021 (race-free)",
  "inputs": {
    "serumCreatinine": {"value": 1.8, "unit": "mg/dL"},
    "ageYears": 72,
    "sex": "male"
  },
  "interpretation": "eGFR 42.5 mL/min/1.73m² - G3b. Renal dose adjustment required."
}
```

#### Batch Calculation (Phase 2 Context Assembly)

```bash
curl -X POST http://localhost:8080/api/v1/calculate/batch \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "calculators": ["EGFR", "SOFA", "QSOFA", "BMI"],
    "includeIndiaAdjustments": true
  }'
```

### GraphQL API

The service exposes a GraphQL endpoint at `/graphql` compatible with Apollo Federation v2.

```graphql
query CalculateForContext($input: CalculatorBatchInput!) {
  calculateAllForContext(input: $input) {
    patientId
    calculatedAt
    overallDataQuality
    summary {
      egfr
      ckdStage
      requiresRenalDoseAdjustment
      sofaTotal
      sofaRiskLevel
      qsofaPositive
      bmi
      bmiCategory
    }
    results {
      type
      success
      egfr {
        value
        ckdStage
        requiresRenalDoseAdjustment
      }
      sofa {
        total
        riskLevel
      }
    }
    failures {
      type
      error
      missingData
    }
  }
}
```

## Integration with 4-Phase Workflow

### Phase 2: Context Assembly

The `calculateAllForContext` query is designed for Phase 2 integration:

```go
// In Context Integration Service
batchResult, err := calculatorClient.CalculateAllForContext(ctx, &CalculatorBatchInput{
    PatientID:   patientID,
    Calculators: []CalculatorType{EGFR, SOFA, QSOFA, BMI, CHA2DS2_VASC},
    AsOf:        &contextTimestamp,
})

// Add to CompleteContextPayload
payload.ComputedScores = batchResult.Summary
payload.EGFRValue = batchResult.Summary.EGFR
payload.RequiresRenalDoseAdjustment = batchResult.Summary.RequiresRenalDoseAdjustment
```

### Phase 3b: Dose Calculation

The Rust DoseCalculationEngine queries eGFR for renal adjustments:

```rust
// In DoseCalculationEngine
let egfr = calculator_client.calculate_egfr(&patient_id).await?;

if egfr.requires_renal_dose_adjustment {
    // Apply renal dose adjustment from kb_dosing_rules
    let adjusted_dose = apply_renal_adjustment(&drug_rule, egfr.value);
}
```

## CQL Integration

The service evaluates expressions from `ClinicalCalculatorsCommon.cql`:

| CQL Expression | Calculator |
|----------------|------------|
| `SOFA Total Score` | SOFA |
| `SOFA Respiration Score` | SOFA component |
| `qSOFA Total Score` | qSOFA |
| `qSOFA Positive` | qSOFA |
| `CHA2DS2-VASc Total Score` | CHA₂DS₂-VASc |
| `Anticoagulation Recommended` | CHA₂DS₂-VASc |
| `eGFR CKD-EPI` | eGFR |
| `CKD Stage` | eGFR |
| `Requires Renal Dose Adjustment` | eGFR |
| `BMI Calculated` | BMI |
| `BMI Category India Adjusted` | BMI (India) |
| `Corrected Calcium` | Corrected Calcium |
| `Anion Gap` | Anion Gap |

## SaMD Compliance

### Traceability

Every calculation includes full provenance:

```json
{
  "provenance": {
    "cqlLibrary": "ClinicalCalculatorsCommon",
    "cqlVersion": "1.0.000",
    "cqlExpression": "eGFR CKD-EPI",
    "calculatedAt": "2025-11-29T12:00:00Z",
    "dataSources": [
      {
        "resourceType": "Observation",
        "resourceId": "obs-creatinine-123",
        "code": "2160-0",
        "display": "Serum Creatinine"
      }
    ],
    "dataQuality": "COMPLETE",
    "missingData": [],
    "warnings": []
  }
}
```

### Data Quality Indicators

- **COMPLETE**: All required data present and recent
- **PARTIAL**: Some optional data missing
- **INCOMPLETE**: Critical data missing (result may be unreliable)
- **STALE**: Data too old for reliable calculation

### Audit Trail

All calculations are logged with:
- Patient ID
- Calculator type
- CQL expression evaluated
- Input data sources (FHIR resource IDs)
- Result values
- Timestamp

## Monitoring & Observability

### Health Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/health` | Service health status |
| `/ready` | Kubernetes readiness probe |
| `/live` | Kubernetes liveness probe |
| `/metrics` | Prometheus metrics |

### Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `kb8_calculator_requests_total` | Counter | Total calculation requests |
| `kb8_calculator_latency_seconds` | Histogram | Calculation latency |
| `kb8_calculator_errors_total` | Counter | Calculation errors |
| `kb8_cql_evaluation_latency_seconds` | Histogram | CQL evaluation latency |

## Directory Structure

```
kb8-calculator-service/
├── api/
│   └── schema.graphql          # GraphQL schema (Federation v2)
├── cmd/
│   └── server/
│       └── main.go             # Entry point
├── internal/
│   ├── calculator/
│   │   └── service.go          # Calculator service
│   ├── cql/
│   │   └── client.go           # CQL Evaluation Service client
│   ├── graphql/
│   │   └── resolver.go         # GraphQL resolvers (gqlgen)
│   └── models/
│       └── models.go           # Domain models
├── pkg/
│   └── health/
│       └── health.go           # Health check utilities
├── config/
│   └── config.yaml             # Configuration
├── test/
│   └── integration/
│       └── calculator_test.go  # Integration tests
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

## Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires CQL service)
go test ./test/integration/... -tags=integration

# Load testing
k6 run test/load/calculator_load.js
```

## License

Proprietary - Clinical Knowledge Platform

## References

- Vincent JL, et al. The SOFA score. Intensive Care Med. 1996
- Singer M, et al. Sepsis-3 definitions. JAMA. 2016
- Lip GY, et al. CHA₂DS₂-VASc score. Chest. 2010
- Inker LA, et al. CKD-EPI 2021. N Engl J Med. 2021
- WHO Asia-Pacific BMI Guidelines
