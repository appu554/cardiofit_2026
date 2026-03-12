# KB-9 Care Gaps Service - User Guide

## Overview

KB-9 is a **Care Gaps Detection and Quality Measure Evaluation Service** that identifies gaps in patient care based on clinical quality measures. It uses the **Da Vinci DEQM (Data Exchange for Quality Measures)** standard for FHIR-compliant care gap reporting.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    KB-9: ACCOUNTABILITY ENGINE                          │
│                                                                          │
│   "What care obligations exist for this patient?"                       │
│                                                                          │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                │
│   │ FHIR Data   │───▶│ CQL Engine  │───▶│ Care Gaps   │                │
│   │ (Patient)   │    │ (Measures)  │    │ (Report)    │                │
│   └─────────────┘    └─────────────┘    └─────────────┘                │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## What It Does

### Core Capabilities

| Capability | Description |
|------------|-------------|
| **Gap Detection** | Identifies patients who are due for preventive care or chronic disease management |
| **Measure Evaluation** | Evaluates CMS quality measures (CMS122, CMS165, CMS130, CMS2) |
| **Temporal Awareness** | Integrates with KB-3 to add due dates and overdue status |
| **FHIR Compliance** | Full Da Vinci DEQM $care-gaps operation support |
| **Population Health** | Evaluate measures across patient cohorts |

### Supported Quality Measures

| Measure ID | Name | Description | Domain |
|------------|------|-------------|--------|
| **CMS122** | Diabetes HbA1c Poor Control | Patients with diabetes whose HbA1c > 9% | Chronic Disease |
| **CMS165** | Blood Pressure Control | Hypertensive patients with BP < 140/90 | Chronic Disease |
| **CMS130** | Colorectal Cancer Screening | Patients 50-75 with appropriate screening | Preventive Care |
| **CMS2** | Depression Screening | Depression screening with follow-up | Behavioral Health |
| **INDIA-DM** | India Diabetes Care | ICMR diabetes management guidelines | Chronic Disease |
| **INDIA-HTN** | India Hypertension Care | ICMR hypertension guidelines | Chronic Disease |

## Architecture

### Tier 7: Longitudinal Intelligence

KB-9 is part of the **Tier 7 Longitudinal Intelligence Platform**, working alongside KB-3:

```
┌────────────────────────────────────────────────────────────────────────┐
│                    TIER 7: LONGITUDINAL INTELLIGENCE                    │
├────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   KB-9 (Care Gaps)              KB-3 (Temporal)                        │
│   "What is due?"                "When is it due?"                      │
│   ┌─────────────────┐           ┌─────────────────┐                    │
│   │ CQL Engine      │──WHAT────▶│ PathwayEngine   │                    │
│   │ Measure Engine  │           │ SchedulingEngine│                    │
│   │ Gap Detection   │◀───WHEN───│ Temporal Ops    │                    │
│   └─────────────────┘           └─────────────────┘                    │
│                                                                         │
└────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
1. Request comes in (Patient ID + Measures)
        ↓
2. FHIR Client queries Google Healthcare API for patient data
        ↓
3. CQL Engine evaluates quality measures against patient data
        ↓
4. Gap Detection identifies unmet quality targets
        ↓
5. KB-3 Integration enriches gaps with temporal context
        ↓
6. Response returned (CareGapReport or FHIR Bundle)
```

## Quick Start

### Prerequisites

- Go 1.21+
- Redis (optional, for caching)
- Google Cloud credentials (for FHIR access)
- KB-3 running (optional, for temporal features)

### Running Locally

```bash
# Clone and navigate
cd backend/shared-infrastructure/knowledge-base-services/kb-9-care-gaps

# Build
make build

# Run with minimal config
PORT=8089 \
ENVIRONMENT=development \
GOOGLE_CLOUD_PROJECT_ID=your-project \
GOOGLE_CLOUD_LOCATION=us-central1 \
GOOGLE_CLOUD_DATASET_ID=your-dataset \
GOOGLE_CLOUD_FHIR_STORE_ID=your-fhir-store \
./bin/kb-9-care-gaps
```

### Running with Docker

```bash
# Start KB-9 with Redis
docker-compose up -d

# Start with FHIR server (for testing)
docker-compose --profile fhir up -d

# Check status
docker-compose ps
```

### Verify Service

```bash
# Health check
curl http://localhost:8089/health

# List available measures
curl http://localhost:8089/api/v1/measures
```

## API Reference

### REST Endpoints

#### Get Patient Care Gaps

**POST** `/api/v1/care-gaps`

Retrieve care gaps for a specific patient.

```bash
curl -X POST http://localhost:8089/api/v1/care-gaps \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "period": {
      "start": "2024-01-01",
      "end": "2024-12-31"
    },
    "measures": ["CMS122_DIABETES_HBA1C", "CMS165_BP_CONTROL"],
    "includeEvidence": true,
    "includeClosedGaps": false,
    "createScheduleItems": true
  }'
```

**Response:**
```json
{
  "patientId": "patient-123",
  "reportDate": "2024-12-22T10:30:00Z",
  "measurementPeriod": {
    "start": "2024-01-01",
    "end": "2024-12-31"
  },
  "openGaps": [
    {
      "id": "gap-uuid-123",
      "measure": {
        "type": "CMS122_DIABETES_HBA1C",
        "cmsId": "CMS122v11",
        "name": "Diabetes: Hemoglobin A1c Poor Control"
      },
      "status": "open",
      "priority": "high",
      "reason": "HbA1c 9.5% exceeds 9.0% target",
      "recommendation": "Review diabetes management and order HbA1c test if due",
      "temporalContext": {
        "daysUntilDue": 30,
        "status": "approaching",
        "isRecurring": true,
        "recurrenceMonths": 3
      },
      "interventions": [
        {
          "type": "lab_order",
          "description": "Order HbA1c test",
          "code": "4548-4",
          "codeSystem": "http://loinc.org"
        }
      ]
    }
  ],
  "summary": {
    "totalOpenGaps": 1,
    "highPriorityGaps": 1,
    "qualityScore": 90.0
  }
}
```

#### List Available Measures

**GET** `/api/v1/measures`

```bash
curl http://localhost:8089/api/v1/measures
```

#### Evaluate Single Measure

**POST** `/api/v1/measure/evaluate`

```bash
curl -X POST http://localhost:8089/api/v1/measure/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "measureType": "CMS122_DIABETES_HBA1C",
    "period": {
      "start": "2024-01-01",
      "end": "2024-12-31"
    }
  }'
```

#### Evaluate Population

**POST** `/api/v1/measure/evaluate-population`

```bash
curl -X POST http://localhost:8089/api/v1/measure/evaluate-population \
  -H "Content-Type: application/json" \
  -d '{
    "patientIds": ["patient-1", "patient-2", "patient-3"],
    "measureType": "CMS122_DIABETES_HBA1C",
    "period": {
      "start": "2024-01-01",
      "end": "2024-12-31"
    },
    "limit": 100
  }'
```

### FHIR Operations (Da Vinci DEQM)

#### $care-gaps Operation

**POST** `/fhir/Measure/$care-gaps`

Standard FHIR $care-gaps operation returning a Bundle with MeasureReport resources.

```bash
curl -X POST http://localhost:8089/fhir/Measure/\$care-gaps \
  -H "Content-Type: application/fhir+json" \
  -d '{
    "resourceType": "Parameters",
    "parameter": [
      {"name": "periodStart", "valueDate": "2024-01-01"},
      {"name": "periodEnd", "valueDate": "2024-12-31"},
      {"name": "subject", "valueString": "Patient/patient-123"},
      {"name": "status", "valueString": "open-gap"}
    ]
  }'
```

**Response:** FHIR Bundle containing:
- `Composition` (care gaps document)
- `MeasureReport` (for each evaluated measure)
- `DetectedIssue` (for each identified gap)

#### $evaluate-measure Operation

**POST** `/fhir/Measure/{measureId}/$evaluate-measure`

```bash
curl -X POST http://localhost:8089/fhir/Measure/CMS122/\$evaluate-measure \
  -H "Content-Type: application/fhir+json" \
  -d '{
    "resourceType": "Parameters",
    "parameter": [
      {"name": "periodStart", "valueDate": "2024-01-01"},
      {"name": "periodEnd", "valueDate": "2024-12-31"},
      {"name": "subject", "valueString": "Patient/patient-123"},
      {"name": "reportType", "valueString": "individual"}
    ]
  }'
```

### GraphQL API

**POST** `/graphql`

#### Query: Get Care Gaps

```graphql
query GetCareGaps($patientId: ID!, $period: PeriodInput!) {
  getPatientCareGaps(
    patientId: $patientId
    period: $period
    includeEvidence: true
  ) {
    patientId
    reportDate
    openGaps {
      id
      measure {
        type
        name
        cmsId
      }
      status
      priority
      reason
      recommendation
      temporalContext {
        daysUntilDue
        status
      }
    }
    summary {
      totalOpenGaps
      highPriorityGaps
      qualityScore
    }
  }
}
```

Variables:
```json
{
  "patientId": "patient-123",
  "period": {
    "start": "2024-01-01",
    "end": "2024-12-31"
  }
}
```

#### Query: Available Measures

```graphql
query {
  availableMeasures {
    type
    cmsId
    name
    description
    domain
    steward
  }
}
```

#### Mutation: Record Gap Addressed

```graphql
mutation RecordIntervention($input: GapAddressedInput!) {
  recordGapAddressed(input: $input) {
    id
    status
    closedDate
  }
}
```

### Gap Management Endpoints

#### Mark Gap Addressed

**POST** `/api/v1/gaps/{gapId}/addressed`

```bash
curl -X POST http://localhost:8089/api/v1/gaps/gap-123/addressed \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "intervention": "lab_order",
    "notes": "HbA1c test ordered"
  }'
```

#### Dismiss Gap

**POST** `/api/v1/gaps/{gapId}/dismiss`

```bash
curl -X POST http://localhost:8089/api/v1/gaps/gap-123/dismiss \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "reason": "Patient declined intervention"
  }'
```

#### Snooze Gap

**POST** `/api/v1/gaps/{gapId}/snooze`

```bash
curl -X POST http://localhost:8089/api/v1/gaps/gap-123/snooze \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "snoozeUntil": "2025-03-01",
    "reason": "Patient traveling"
  }'
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | 8089 | No |
| `ENVIRONMENT` | development/staging/production | development | No |
| `LOG_LEVEL` | debug/info/warn/error | info | No |
| **FHIR Configuration** | | | |
| `GOOGLE_CLOUD_PROJECT_ID` | GCP project ID | - | Yes |
| `GOOGLE_CLOUD_LOCATION` | GCP region | us-central1 | No |
| `GOOGLE_CLOUD_DATASET_ID` | Healthcare API dataset | - | Yes |
| `GOOGLE_CLOUD_FHIR_STORE_ID` | FHIR store ID | - | Yes |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to credentials | - | Yes |
| `FHIR_TIMEOUT` | FHIR request timeout | 30s | No |
| **KB-3 Integration** | | | |
| `KB3_URL` | KB-3 service URL | http://kb-3-guidelines:8083 | No |
| `KB3_TIMEOUT` | KB-3 request timeout | 10s | No |
| `KB3_ENABLED` | Enable temporal features | true | No |
| **Caching** | | | |
| `REDIS_URL` | Redis connection URL | - | No |
| `CACHE_TTL` | Cache TTL | 5m | No |
| **GraphQL** | | | |
| `FEDERATION_ENABLED` | Enable GraphQL | true | No |
| `PLAYGROUND_ENABLED` | Enable GraphQL Playground | true | No |
| **CQL Engine** | | | |
| `USE_CQL_ENGINE` | Use vaidshala CQL engine | true | No |
| `REGION` | Region (US/IN/AU) | US | No |

### Example .env File

```bash
# Server
PORT=8089
ENVIRONMENT=development
LOG_LEVEL=info

# Google Cloud FHIR
GOOGLE_CLOUD_PROJECT_ID=my-healthcare-project
GOOGLE_CLOUD_LOCATION=us-central1
GOOGLE_CLOUD_DATASET_ID=clinical-data
GOOGLE_CLOUD_FHIR_STORE_ID=patient-records
GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json

# KB-3 Temporal Integration
KB3_URL=http://localhost:8083
KB3_ENABLED=true

# Redis Cache
REDIS_URL=redis://localhost:6379/9
CACHE_TTL=5m

# Features
FEDERATION_ENABLED=true
PLAYGROUND_ENABLED=true
USE_CQL_ENGINE=true
REGION=US
```

## Use Cases

### 1. Pre-Visit Planning

Before a patient appointment, retrieve their care gaps:

```bash
# Get all care gaps for upcoming visit
curl -X POST http://localhost:8089/api/v1/care-gaps \
  -d '{"patientId": "patient-123", "period": {"start": "2024-01-01", "end": "2024-12-31"}}'
```

**Use in EHR:**
- Display gaps in patient banner
- Create checklist for visit
- Pre-order recommended tests

### 2. Population Health Dashboard

Identify patients with care gaps across your panel:

```bash
# Evaluate diabetes measure for cohort
curl -X POST http://localhost:8089/api/v1/measure/evaluate-population \
  -d '{
    "patientIds": ["p1", "p2", "p3", ...],
    "measureType": "CMS122_DIABETES_HBA1C",
    "period": {"start": "2024-01-01", "end": "2024-12-31"}
  }'
```

**Use for:**
- Quality reporting
- Risk stratification
- Outreach campaigns

### 3. CDS Hooks Integration

Embed care gap alerts in clinical workflow:

```javascript
// CDS Hooks service calling KB-9
const response = await fetch('http://kb-9:8089/api/v1/care-gaps', {
  method: 'POST',
  body: JSON.stringify({
    patientId: context.patient,
    period: { start: '2024-01-01', end: '2024-12-31' }
  })
});

const gaps = await response.json();

// Return CDS cards for high-priority gaps
return gaps.openGaps
  .filter(g => g.priority === 'high')
  .map(gap => ({
    summary: gap.measure.name,
    detail: gap.recommendation,
    indicator: 'warning',
    suggestions: gap.interventions.map(i => ({
      label: i.description,
      actions: [{ type: 'create', resource: createOrder(i) }]
    }))
  }));
```

### 4. Quality Reporting (MIPS/HEDIS)

Generate FHIR-compliant measure reports:

```bash
# Da Vinci DEQM $care-gaps for official reporting
curl -X POST http://localhost:8089/fhir/Measure/\$care-gaps \
  -H "Content-Type: application/fhir+json" \
  -d '{
    "resourceType": "Parameters",
    "parameter": [
      {"name": "periodStart", "valueDate": "2024-01-01"},
      {"name": "periodEnd", "valueDate": "2024-12-31"},
      {"name": "subject", "valueString": "Group/all-diabetics"},
      {"name": "status", "valueString": "open-gap"}
    ]
  }'
```

## Monitoring

### Health Endpoints

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `/health` | Overall health | `{"status": "healthy", ...}` |
| `/ready` | Ready to accept traffic | `{"status": "ready"}` |
| `/live` | Container alive | `{"status": "alive"}` |
| `/metrics` | Prometheus metrics | Prometheus format |

### Key Metrics

```
# Request metrics
http_requests_total{endpoint="/api/v1/care-gaps"}
http_request_duration_seconds{endpoint="/api/v1/care-gaps"}

# Cache metrics
cache_hits_total
cache_misses_total
cache_memory_items

# Business metrics
care_gaps_detected_total{measure="CMS122"}
measure_evaluations_total{measure="CMS122"}
```

## Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests (requires service running)
make test-integration

# Run with coverage
make coverage
```

## Troubleshooting

### Common Issues

**1. FHIR Connection Failed**
```
Error: failed to connect to FHIR server
```
- Verify `GOOGLE_CLOUD_PROJECT_ID` is correct
- Check credentials file path
- Ensure Healthcare API is enabled in GCP

**2. KB-3 Not Available**
```
Warn: KB-3 temporal enrichment failed
```
- Service continues without temporal features
- Start KB-3 on port 8083 for full functionality

**3. Cache Connection Failed**
```
Warn: Redis connection failed, using in-memory cache only
```
- Service uses L1 memory cache as fallback
- Start Redis for distributed caching

### Debug Mode

```bash
# Run with debug logging
LOG_LEVEL=debug ./bin/kb-9-care-gaps
```

## Support

- **Documentation**: [KB-9 Implementation Plan](./KB9_IMPLEMENTATION_PLAN.md)
- **GraphQL Schema**: [api/schema.graphql](./api/schema.graphql)
- **Integration Tests**: [tests/integration/](./tests/integration/)
