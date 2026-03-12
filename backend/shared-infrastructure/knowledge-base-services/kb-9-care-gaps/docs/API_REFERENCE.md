# KB-9 Care Gaps Service - API Reference

**Base URL:** `http://localhost:8089`

## Quick Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/measures` | List all measures |
| GET | `/api/v1/measures/{type}` | Get measure details |
| POST | `/api/v1/care-gaps` | Get patient care gaps |
| POST | `/api/v1/measure/evaluate` | Evaluate single measure |
| POST | `/api/v1/measure/evaluate-population` | Population evaluation |
| POST | `/api/v1/gaps/{id}/addressed` | Mark gap addressed |
| POST | `/api/v1/gaps/{id}/dismiss` | Dismiss gap |
| POST | `/api/v1/gaps/{id}/snooze` | Snooze gap |
| POST | `/fhir/Measure/$care-gaps` | FHIR $care-gaps operation |
| POST | `/fhir/Measure/{id}/$evaluate-measure` | FHIR evaluate measure |
| POST | `/graphql` | GraphQL endpoint |

---

## REST API

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "kb-9-care-gaps",
  "version": "1.0.0",
  "uptime": "2h30m",
  "checks": {
    "fhir": "ok",
    "redis": "ok",
    "kb3": "ok"
  }
}
```

---

### Get Patient Care Gaps

```http
POST /api/v1/care-gaps
Content-Type: application/json
```

**Request:**
```json
{
  "patientId": "patient-123",
  "period": {
    "start": "2024-01-01",
    "end": "2024-12-31"
  },
  "measures": ["CMS122_DIABETES_HBA1C"],
  "includeEvidence": true,
  "includeClosedGaps": false,
  "createScheduleItems": true
}
```

**Response:**
```json
{
  "patientId": "patient-123",
  "reportDate": "2024-12-22T10:30:00Z",
  "measurementPeriod": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-12-31T23:59:59Z"
  },
  "openGaps": [{
    "id": "gap-uuid",
    "measure": {
      "type": "CMS122_DIABETES_HBA1C",
      "cmsId": "CMS122v11",
      "name": "Diabetes: Hemoglobin A1c Poor Control",
      "domain": "Chronic Disease"
    },
    "status": "open",
    "priority": "high",
    "reason": "HbA1c 9.5% exceeds 9.0% target",
    "recommendation": "Order HbA1c test",
    "identifiedDate": "2024-12-22T10:30:00Z",
    "temporalContext": {
      "daysUntilDue": 30,
      "status": "approaching",
      "isRecurring": true,
      "recurrenceMonths": 3
    },
    "interventions": [{
      "type": "lab_order",
      "description": "Order HbA1c test",
      "code": "4548-4",
      "codeSystem": "http://loinc.org",
      "priority": "high"
    }],
    "evidence": {
      "libraryId": "CMS122-DiabetesHbA1c",
      "libraryVersion": "11.0.0",
      "populations": [{
        "population": "denominator",
        "isMember": true,
        "reason": "Has diabetes diagnosis"
      }],
      "dataElements": [{
        "name": "Most Recent HbA1c",
        "value": "9.5",
        "contributedToGap": true
      }]
    }
  }],
  "closedGaps": [],
  "upcomingDue": [],
  "summary": {
    "totalOpenGaps": 1,
    "urgentGaps": 0,
    "highPriorityGaps": 1,
    "qualityScore": 90.0,
    "gapsByDomain": [{
      "domain": "Chronic Disease",
      "count": 1
    }]
  },
  "dataCompleteness": "complete"
}
```

---

### List Available Measures

```http
GET /api/v1/measures
```

**Response:**
```json
{
  "measures": [
    {
      "type": "CMS122_DIABETES_HBA1C",
      "cmsId": "CMS122v11",
      "name": "Diabetes: Hemoglobin A1c Poor Control",
      "description": "Patients 18-75 with diabetes who had HbA1c > 9.0%",
      "domain": "Chronic Disease",
      "steward": "NCQA",
      "version": "11.0.0"
    },
    {
      "type": "CMS165_BP_CONTROL",
      "cmsId": "CMS165v11",
      "name": "Controlling High Blood Pressure",
      "description": "Patients 18-85 with hypertension and BP < 140/90",
      "domain": "Chronic Disease",
      "steward": "NCQA",
      "version": "11.0.0"
    }
  ],
  "count": 6
}
```

---

### Evaluate Single Measure

```http
POST /api/v1/measure/evaluate
Content-Type: application/json
```

**Request:**
```json
{
  "patientId": "patient-123",
  "measureType": "CMS122_DIABETES_HBA1C",
  "period": {
    "start": "2024-01-01",
    "end": "2024-12-31"
  }
}
```

**Response:**
```json
{
  "id": "report-uuid",
  "measure": {
    "type": "CMS122_DIABETES_HBA1C",
    "name": "Diabetes: Hemoglobin A1c Poor Control"
  },
  "patientId": "patient-123",
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-12-31T23:59:59Z"
  },
  "status": "complete",
  "type": "individual",
  "populations": [
    {"population": "initial-population", "count": 1},
    {"population": "denominator", "count": 1},
    {"population": "numerator", "count": 0}
  ],
  "generatedAt": "2024-12-22T10:30:00Z"
}
```

---

### Evaluate Population

```http
POST /api/v1/measure/evaluate-population
Content-Type: application/json
```

**Request:**
```json
{
  "patientIds": ["patient-1", "patient-2", "patient-3"],
  "measureType": "CMS122_DIABETES_HBA1C",
  "period": {
    "start": "2024-01-01",
    "end": "2024-12-31"
  },
  "limit": 100
}
```

**Response:**
```json
{
  "id": "pop-report-uuid",
  "measure": {
    "type": "CMS122_DIABETES_HBA1C",
    "name": "Diabetes: Hemoglobin A1c Poor Control"
  },
  "period": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-12-31T23:59:59Z"
  },
  "totalPatients": 3,
  "populations": [
    {"population": "initial-population", "count": 3},
    {"population": "denominator", "count": 3},
    {"population": "numerator", "count": 1}
  ],
  "performanceRate": 33.33,
  "patientsWithGaps": [
    {"patientId": "patient-2", "status": "open", "recommendation": "Order HbA1c"},
    {"patientId": "patient-3", "status": "open", "recommendation": "Order HbA1c"}
  ],
  "generatedAt": "2024-12-22T10:30:00Z",
  "processingTimeMs": 150
}
```

---

### Gap Management

#### Mark Gap Addressed

```http
POST /api/v1/gaps/{gapId}/addressed
Content-Type: application/json
```

**Request:**
```json
{
  "patientId": "patient-123",
  "intervention": "lab_order",
  "notes": "HbA1c test ordered"
}
```

#### Dismiss Gap

```http
POST /api/v1/gaps/{gapId}/dismiss
Content-Type: application/json
```

**Request:**
```json
{
  "patientId": "patient-123",
  "reason": "Patient declined"
}
```

#### Snooze Gap

```http
POST /api/v1/gaps/{gapId}/snooze
Content-Type: application/json
```

**Request:**
```json
{
  "patientId": "patient-123",
  "snoozeUntil": "2025-03-01",
  "reason": "Patient traveling"
}
```

---

## FHIR Operations (Da Vinci DEQM)

### $care-gaps Operation

```http
POST /fhir/Measure/$care-gaps
Content-Type: application/fhir+json
```

**Request:**
```json
{
  "resourceType": "Parameters",
  "parameter": [
    {"name": "periodStart", "valueDate": "2024-01-01"},
    {"name": "periodEnd", "valueDate": "2024-12-31"},
    {"name": "subject", "valueString": "Patient/patient-123"},
    {"name": "status", "valueString": "open-gap"},
    {"name": "measure", "valueString": "CMS122"}
  ]
}
```

**Response:** FHIR Bundle (type: document)
```json
{
  "resourceType": "Bundle",
  "type": "document",
  "timestamp": "2024-12-22T10:30:00Z",
  "entry": [
    {
      "resource": {
        "resourceType": "Composition",
        "title": "Care Gaps Report",
        "subject": {"reference": "Patient/patient-123"}
      }
    },
    {
      "resource": {
        "resourceType": "MeasureReport",
        "measure": "Measure/CMS122",
        "status": "complete",
        "type": "individual"
      }
    },
    {
      "resource": {
        "resourceType": "DetectedIssue",
        "status": "final",
        "code": {"text": "Care gap detected"},
        "severity": "high"
      }
    }
  ]
}
```

---

## GraphQL API

**Endpoint:** `POST /graphql`

### Queries

```graphql
# Get care gaps for a patient
query GetCareGaps($patientId: ID!, $period: PeriodInput!) {
  getPatientCareGaps(patientId: $patientId, period: $period) {
    openGaps {
      id
      measure { type name }
      status
      priority
      reason
    }
    summary {
      totalOpenGaps
      qualityScore
    }
  }
}

# List available measures
query {
  availableMeasures {
    type
    cmsId
    name
    domain
  }
}

# Evaluate measure
query EvaluateMeasure($patientId: ID!, $measureType: MeasureType!, $period: PeriodInput!) {
  evaluateMeasure(patientId: $patientId, measureType: $measureType, period: $period) {
    id
    status
    populations {
      population
      count
    }
  }
}

# Health check
query {
  healthCheck {
    status
    version
    checks {
      fhir
      redis
      kb3
    }
  }
}
```

### Mutations

```graphql
# Record gap addressed
mutation RecordAddressed($input: GapAddressedInput!) {
  recordGapAddressed(input: $input) {
    id
    status
  }
}

# Dismiss gap
mutation DismissGap($input: DismissGapInput!) {
  dismissGap(input: $input) {
    id
    status
  }
}

# Snooze gap
mutation SnoozeGap($input: SnoozeGapInput!) {
  snoozeGap(input: $input) {
    id
    status
    dueDate
  }
}
```

---

## Error Responses

All endpoints return errors in this format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid patient ID format",
    "details": {
      "field": "patientId",
      "constraint": "required"
    }
  }
}
```

| HTTP Code | Error Code | Description |
|-----------|------------|-------------|
| 400 | VALIDATION_ERROR | Invalid request |
| 404 | NOT_FOUND | Resource not found |
| 500 | INTERNAL_ERROR | Server error |
| 503 | SERVICE_UNAVAILABLE | Dependency unavailable |

---

## Rate Limits

| Endpoint | Limit |
|----------|-------|
| `/api/v1/care-gaps` | 100 req/min |
| `/api/v1/measure/evaluate-population` | 10 req/min |
| Others | 1000 req/min |
