# KB-9 Care Gaps Service

**Care gaps detection and quality measure evaluation for the Clinical Knowledge Platform**

[![SaMD Classification](https://img.shields.io/badge/SaMD-Class%20IIa-blue)](docs/samd-compliance.md)
[![CQL Pattern](https://img.shields.io/badge/CQL-Query--Based-orange)](docs/cql-pattern.md)
[![Da Vinci DEQM](https://img.shields.io/badge/FHIR-Da%20Vinci%20DEQM-green)](docs/deqm.md)

## Overview

KB-9 Care Gaps Service provides clinical quality measure evaluation and care gap detection. Unlike KB-8 (Atomic/stateless), KB-9 uses the **Query-Based pattern** where CQL queries FHIR directly for longitudinal patient data.

### The Split Brain Architecture

| Service | Pattern | Data Source | Latency | Use Case |
|---------|---------|-------------|---------|----------|
| **KB-8 Calculators** | ATOMIC | Caller provides values | ~5ms | Real-time scores, Flink streams |
| **KB-9 Care Gaps** | QUERY-BASED | CQL queries FHIR | ~200ms | Longitudinal measures, eCQM |

## Supported Measures

| Measure | CMS ID | Description | Domain |
|---------|--------|-------------|--------|
| **Diabetes HbA1c** | CMS122 | HbA1c poor control (>9%) | Chronic Disease |
| **BP Control** | CMS165 | Blood pressure <140/90 | Chronic Disease |
| **Colorectal Screening** | CMS130 | Colonoscopy/FIT/FOBT | Preventive Care |
| **India Diabetes Care** | Custom | Comprehensive annual care | Chronic Disease |
| **India Hypertension** | Custom | BP + kidney function | Chronic Disease |

## Quick Start

### Docker

```bash
docker build -t kb9-caregaps-service .
docker run -p 8081:8081 \
  -e FHIR_SERVER_URL=http://hapi-fhir:8080/fhir \
  -e CQL_SERVICE_URL=http://hapi-fhir:8080/fhir \
  kb9-caregaps-service
```

### Local Development

```bash
go mod download
go run cmd/server/main.go
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8081 | Server port |
| `FHIR_SERVER_URL` | http://hapi-fhir:8080/fhir | FHIR Server URL |
| `CQL_SERVICE_URL` | http://hapi-fhir:8080/fhir | CQL Evaluation Service URL |
| `TERMINOLOGY_URL` | http://terminology-service:8080/fhir | Terminology Service URL |
| `ENABLE_PLAYGROUND` | true | Enable GraphQL Playground |

## API Reference

### REST API

#### Get Patient Care Gaps

```bash
curl -X POST http://localhost:8081/api/v1/care-gaps \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "periodStart": "2025-01-01",
    "periodEnd": "2025-12-31",
    "includeEvidence": true
  }'
```

**Response:**
```json
{
  "patientId": "patient-123",
  "reportDate": "2025-11-29T12:00:00Z",
  "measurementPeriod": {
    "start": "2025-01-01",
    "end": "2025-12-31"
  },
  "openGaps": [
    {
      "id": "gap-uuid",
      "measure": {
        "type": "CMS122_DIABETES_HBA1C",
        "cmsId": "CMS122v11",
        "name": "Diabetes: Hemoglobin A1c Poor Control"
      },
      "status": "OPEN",
      "priority": "HIGH",
      "reason": "HbA1c 10.2% - above 9% target",
      "recommendation": "Review diabetes management - HbA1c 10.2%",
      "interventions": [
        {
          "type": "LAB_ORDER",
          "description": "Order HbA1c test",
          "code": "4548-4",
          "codeSystem": "http://loinc.org"
        }
      ],
      "evidence": {
        "libraryId": "CareGapsEngine",
        "populations": [
          {"population": "DENOMINATOR", "isMember": true},
          {"population": "NUMERATOR", "isMember": false}
        ],
        "dataElements": [
          {
            "name": "Most Recent HbA1c",
            "value": "10.2%",
            "contributedToGap": true
          }
        ]
      }
    }
  ],
  "summary": {
    "totalOpenGaps": 1,
    "urgentGaps": 0,
    "highPriorityGaps": 1,
    "qualityScore": 90
  }
}
```

#### Evaluate Single Measure

```bash
curl -X POST http://localhost:8081/api/v1/measure/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "measure": "CMS122_DIABETES_HBA1C",
    "periodStart": "2025-01-01",
    "periodEnd": "2025-12-31"
  }'
```

#### Evaluate Population

```bash
curl -X POST http://localhost:8081/api/v1/measure/evaluate-population \
  -H "Content-Type: application/json" \
  -d '{
    "patientIds": ["patient-1", "patient-2", "patient-3"],
    "measure": "CMS122_DIABETES_HBA1C",
    "periodStart": "2025-01-01",
    "periodEnd": "2025-12-31"
  }'
```

### Da Vinci DEQM FHIR Operations

#### $care-gaps

```bash
curl -X POST http://localhost:8081/fhir/Measure/\$care-gaps \
  -H "Content-Type: application/fhir+json" \
  -d '{
    "resourceType": "Parameters",
    "parameter": [
      {"name": "subject", "valueString": "Patient/patient-123"},
      {"name": "periodStart", "valueDate": "2025-01-01"},
      {"name": "periodEnd", "valueDate": "2025-12-31"}
    ]
  }'
```

#### $evaluate-measure

```bash
curl -X POST http://localhost:8081/fhir/Measure/CMS122_DIABETES_HBA1C/\$evaluate-measure \
  -H "Content-Type: application/fhir+json" \
  -d '{
    "resourceType": "Parameters",
    "parameter": [
      {"name": "subject", "valueString": "Patient/patient-123"},
      {"name": "periodStart", "valueDate": "2025-01-01"},
      {"name": "periodEnd", "valueDate": "2025-12-31"}
    ]
  }'
```

## CQL Integration

### Query-Based Pattern

KB-9 CQL queries FHIR directly for patient data:

```cql
// CQL queries FHIR - NOT parameter-driven like KB-8
define "Most Recent HbA1c":
  Last(
    [Observation: "HbA1c Laboratory Test"] O
      where O.status in { 'final', 'amended', 'corrected' }
        and O.effective during "Measurement Period"
      sort by effective
  )

define "Has Diabetes HbA1c Gap":
  "CMS122 Denominator"
    and not "CMS122 Denominator Exclusions"
    and not "CMS122 Numerator"
```

### Why Query-Based for Care Gaps?

1. **Longitudinal Data**: Care gaps require 10+ years of history (e.g., colonoscopy in last 10 years)
2. **CMS Compatibility**: CMS eCQM libraries are designed with FHIR queries built-in
3. **Complex Logic**: Exclusion criteria require multiple resource types
4. **Batch Processing**: Population health doesn't need real-time (<5ms) latency

## Integration with 4-Phase Workflow

### Phase 4: Proposal Augmentation

Care gaps can be included in medication proposal responses:

```go
// In Phase 4 Proposal Generator
careGaps, _ := careGapsClient.GetPatientCareGaps(patientID)

// Augment medication response with relevant gaps
for _, gap := range careGaps.OpenGaps {
    if isRelevantToMedication(gap, proposedMedication) {
        proposal.CareGapAlerts = append(proposal.CareGapAlerts, gap)
    }
}
```

### Conditions Advisor Integration

The Conditions Advisor application is the primary consumer:

```graphql
query GetPatientCareGaps($patientId: ID!) {
  getPatientCareGaps(input: {
    patientId: $patientId
    includeEvidence: true
  }) {
    openGaps {
      measure { name }
      priority
      recommendation
      interventions { type description }
    }
    summary {
      totalOpenGaps
      qualityScore
    }
  }
}
```

## Directory Structure

```
kb9-caregaps-service/
├── api/
│   └── schema.graphql          # GraphQL schema (Federation v2)
├── cmd/
│   └── server/
│       └── main.go             # Entry point
├── internal/
│   ├── caregaps/
│   │   └── service.go          # Care gaps service
│   ├── cql/
│   │   └── client.go           # CQL client (Query-Based)
│   └── models/
│       └── models.go           # Domain models
├── cql-libraries/
│   └── CareGapsEngine-1.0.000.cql  # Care gaps CQL
├── pkg/
│   └── deqm/
│       └── operations.go       # Da Vinci DEQM support
├── Dockerfile
├── go.mod
└── README.md
```

## Comparison with KB-8

| Aspect | KB-8 (Calculators) | KB-9 (Care Gaps) |
|--------|-------------------|------------------|
| **CQL Pattern** | ATOMIC | QUERY-BASED |
| **Data Source** | Parameters from caller | CQL queries FHIR |
| **Latency** | ~5ms | ~200ms |
| **FHIR Queries** | None in CQL | Yes - [Observation:...] |
| **Use Case** | Real-time scores | Longitudinal measures |
| **Flink Compatible** | ✅ Yes | ❌ No |
| **CMS Library Compatible** | ❌ No | ✅ Yes |

## Monitoring

### Health Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/health` | Service health |
| `/ready` | Kubernetes readiness |
| `/live` | Kubernetes liveness |
| `/metrics` | Prometheus metrics |

### Metrics

| Metric | Description |
|--------|-------------|
| `kb9_care_gaps_requests_total` | Total gap detection requests |
| `kb9_measure_evaluations_total` | Total measure evaluations |
| `kb9_population_evaluations_total` | Population evaluations |
| `kb9_cql_evaluation_latency_seconds` | CQL evaluation latency |

## License

Proprietary - Clinical Knowledge Platform
