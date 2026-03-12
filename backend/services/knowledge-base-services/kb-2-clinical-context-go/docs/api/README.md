# KB-2 Clinical Context API Documentation

## Overview

The KB-2 Clinical Context API provides comprehensive clinical phenotyping, risk assessment, and treatment recommendation services through a high-performance Go microservice. This API transforms basic patient data into actionable clinical intelligence.

## Base URL

```
Production: https://api.cardiofit.health/kb2-clinical-context/v1
Staging: https://staging-api.cardiofit.health/kb2-clinical-context/v1
Development: http://localhost:8088/v1
```

## Authentication

All API endpoints require authentication using JWT bearer tokens obtained from the auth service.

### Headers

```http
Authorization: Bearer <jwt_token>
Content-Type: application/json
X-Client-ID: <client_identifier>
X-Request-ID: <unique_request_id>
```

### Authentication Examples

```bash
# Get authentication token
curl -X POST https://api.cardiofit.health/auth/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "clinician@hospital.org",
    "password": "secure_password"
  }'

# Use token in API calls
curl -X POST http://localhost:8088/v1/phenotypes/evaluate \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{"patients": [...]}'
```

## API Endpoints

### Core Clinical Intelligence Endpoints

| Endpoint | Method | SLA | Description |
|----------|--------|-----|-------------|
| [`/v1/phenotypes/evaluate`](#phenotype-evaluation) | POST | 100ms | Batch phenotype evaluation for multiple patients |
| [`/v1/phenotypes/explain`](#phenotype-explanation) | POST | 150ms | Detailed phenotype reasoning and explanation |
| [`/v1/risk/assess`](#risk-assessment) | POST | 200ms | Comprehensive multi-category risk assessment |
| [`/v1/treatment/preferences`](#treatment-preferences) | POST | 50ms | Institutional treatment recommendations |
| [`/v1/context/assemble`](#context-assembly) | POST | 200ms | Complete clinical context compilation |

### Information and Management Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| [`/v1/phenotypes`](#available-phenotypes) | GET | List available phenotypes and their definitions |
| [`/v1/risk/categories`](#risk-categories) | GET | Available risk assessment categories |
| [`/v1/treatment/guidelines`](#treatment-guidelines) | GET | Supported treatment guidelines and preferences |
| [`/v1/context/history/{patient_id}`](#context-history) | GET | Patient clinical context history |

### System Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Service health check |
| `/ready` | GET | Service readiness probe |
| `/metrics` | GET | Prometheus metrics endpoint |
| `/v1/docs` | GET | Interactive API documentation |

## Request/Response Format

### Standard Response Format

All API responses follow RFC 7807 Problem Details format for consistency:

```json
{
  "status": "success|error",
  "data": {
    // Response payload
  },
  "metadata": {
    "request_id": "req_123456789",
    "timestamp": "2025-01-15T10:30:00Z",
    "processing_time_ms": 25,
    "version": "1.0.0"
  },
  "errors": [
    // Error details if applicable
  ]
}
```

### Error Response Format

```json
{
  "status": "error",
  "errors": [
    {
      "code": "INVALID_PATIENT_DATA",
      "message": "Patient age is required for risk assessment",
      "field": "patient_data.age",
      "severity": "error"
    }
  ],
  "metadata": {
    "request_id": "req_123456789",
    "timestamp": "2025-01-15T10:30:00Z",
    "processing_time_ms": 5
  }
}
```

## Common Data Models

### Patient Data Model

```json
{
  "id": "patient_12345",
  "age": 65,
  "gender": "male|female|other",
  "weight": {
    "value": 80.5,
    "unit": "kg"
  },
  "height": {
    "value": 175,
    "unit": "cm"
  },
  "conditions": [
    "diabetes_type_2",
    "hypertension",
    "coronary_artery_disease"
  ],
  "medications": [
    {
      "name": "metformin",
      "dosage": "1000mg",
      "frequency": "twice_daily",
      "start_date": "2024-06-01"
    }
  ],
  "labs": {
    "hba1c": {
      "value": 8.2,
      "unit": "%",
      "date": "2025-01-10"
    },
    "total_cholesterol": {
      "value": 240,
      "unit": "mg/dL",
      "date": "2025-01-10"
    },
    "creatinine": {
      "value": 1.2,
      "unit": "mg/dL",
      "date": "2025-01-10"
    }
  },
  "vitals": {
    "systolic_bp": 145,
    "diastolic_bp": 90,
    "heart_rate": 78,
    "date": "2025-01-10"
  }
}
```

### Clinical Context Model

```json
{
  "patient_id": "patient_12345",
  "timestamp": "2025-01-15T10:30:00Z",
  "phenotypes": [
    {
      "id": "high_cardiovascular_risk",
      "name": "High Cardiovascular Risk",
      "category": "cardiovascular",
      "positive": true,
      "confidence": 0.95,
      "contributing_factors": [
        "age >= 65",
        "diabetes_present",
        "hypertension_present"
      ]
    }
  ],
  "risk_assessments": [
    {
      "category": "cardiovascular",
      "score": 0.78,
      "risk_level": "high",
      "ten_year_risk": 0.25,
      "factors": {
        "age": 0.3,
        "diabetes": 0.2,
        "hypertension": 0.15,
        "cholesterol": 0.13
      }
    }
  ],
  "treatment_preferences": [
    {
      "condition": "diabetes",
      "recommendations": [
        {
          "medication_class": "sglt2_inhibitors",
          "preference_score": 0.92,
          "rationale": "Cardiovascular benefits with diabetes control"
        }
      ]
    }
  ]
}
```

## Rate Limiting

The API implements rate limiting to ensure system stability and fair usage:

```http
X-RateLimit-Limit: 1000      # Requests per hour limit
X-RateLimit-Remaining: 999   # Remaining requests in window
X-RateLimit-Reset: 1642176000 # Reset time (Unix timestamp)
```

### Rate Limit Tiers

| Client Type | Requests/Hour | Burst Limit |
|-------------|---------------|-------------|
| Individual User | 1,000 | 50 |
| Service Account | 10,000 | 500 |
| Batch Processing | 50,000 | 1,000 |

### Rate Limit Exceeded Response

```json
{
  "status": "error",
  "errors": [
    {
      "code": "RATE_LIMIT_EXCEEDED",
      "message": "Request rate limit exceeded. Try again later.",
      "retry_after": 3600
    }
  ]
}
```

## Error Codes

### Client Errors (4xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request data |
| `AUTHENTICATION_REQUIRED` | 401 | Missing or invalid authentication |
| `INSUFFICIENT_PERMISSIONS` | 403 | Inadequate access permissions |
| `RESOURCE_NOT_FOUND` | 404 | Requested resource not found |
| `INVALID_PATIENT_DATA` | 422 | Patient data validation failed |
| `RATE_LIMIT_EXCEEDED` | 429 | Request rate limit exceeded |

### Server Errors (5xx)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INTERNAL_SERVER_ERROR` | 500 | Unexpected server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |
| `TIMEOUT_ERROR` | 504 | Request processing timeout |
| `DEPENDENCY_ERROR` | 502 | External dependency failure |

## GraphQL Federation Schema

The KB-2 service participates in Apollo Federation with the following schema extensions:

### Federated Types

```graphql
extend type Patient @key(fields: "id") {
  id: ID! @external
  clinicalContext: ClinicalContext
}

type ClinicalContext {
  patientId: ID!
  timestamp: DateTime!
  phenotypes: [Phenotype!]!
  riskAssessments: [RiskAssessment!]!
  treatmentPreferences: [TreatmentPreference!]!
}

type Phenotype {
  id: String!
  name: String!
  category: PhenotypeCategory!
  positive: Boolean!
  confidence: Float!
  contributingFactors: [String!]!
}

type RiskAssessment {
  category: RiskCategory!
  score: Float!
  riskLevel: RiskLevel!
  tenYearRisk: Float
  factors: [RiskFactor!]!
}

enum PhenotypeCategory {
  CARDIOVASCULAR
  DIABETES
  MEDICATION
  FALL
  BLEEDING
}

enum RiskCategory {
  CARDIOVASCULAR
  DIABETES
  MEDICATION
  FALL
  BLEEDING
}

enum RiskLevel {
  LOW
  MODERATE
  HIGH
  VERY_HIGH
}
```

### GraphQL Resolvers

```graphql
type Query {
  # Phenotype queries
  availablePhenotypes: [PhenotypeDefinition!]!
  phenotypesByCategory(category: PhenotypeCategory!): [PhenotypeDefinition!]!
  
  # Risk assessment queries
  riskCategories: [RiskCategoryInfo!]!
  riskModels: [RiskModelInfo!]!
  
  # Treatment preference queries
  treatmentGuidelines: [TreatmentGuideline!]!
  treatmentOptions(condition: String!): [TreatmentOption!]!
}

type Mutation {
  # Clinical context operations
  evaluatePhenotypes(input: PhenotypeEvaluationInput!): PhenotypeEvaluationResult!
  assessRisk(input: RiskAssessmentInput!): RiskAssessmentResult!
  generateTreatmentPreferences(input: TreatmentPreferenceInput!): TreatmentPreferenceResult!
  assembleClinicalContext(input: ContextAssemblyInput!): ClinicalContext!
}
```

## SDK and Client Libraries

### Go SDK

```go
import "github.com/cardiofit/kb2-client-go"

client := kb2.NewClient(&kb2.Config{
    BaseURL: "http://localhost:8088/v1",
    Token:   "your_jwt_token",
    Timeout: 30 * time.Second,
})

// Evaluate phenotypes
result, err := client.Phenotypes.Evaluate(ctx, &kb2.PhenotypeRequest{
    Patients: []kb2.Patient{patient},
})
```

### Python SDK

```python
from kb2_client import KB2Client

client = KB2Client(
    base_url="http://localhost:8088/v1",
    token="your_jwt_token",
    timeout=30
)

# Assess risk
result = await client.risk.assess(
    patient_id="patient_123",
    patient_data=patient_data,
    risk_categories=["cardiovascular", "diabetes"]
)
```

### JavaScript/TypeScript SDK

```typescript
import { KB2Client } from '@cardiofit/kb2-client';

const client = new KB2Client({
  baseURL: 'http://localhost:8088/v1',
  token: 'your_jwt_token',
  timeout: 30000
});

// Assemble clinical context
const context = await client.context.assemble({
  patientId: 'patient_123',
  patientData: patientData,
  detailLevel: 'comprehensive'
});
```

## Webhook Support

The KB-2 service supports webhooks for real-time clinical context updates:

### Webhook Configuration

```json
{
  "webhook_url": "https://your-system.com/webhooks/kb2-updates",
  "events": [
    "phenotype.evaluated",
    "risk.assessed",
    "treatment.recommended"
  ],
  "secret": "webhook_secret_key",
  "active": true
}
```

### Webhook Payload

```json
{
  "event_type": "phenotype.evaluated",
  "timestamp": "2025-01-15T10:30:00Z",
  "patient_id": "patient_123",
  "data": {
    "phenotypes": [...],
    "evaluation_id": "eval_456789",
    "confidence_scores": {...}
  },
  "signature": "sha256=..."
}
```

## Testing and Validation

### API Testing Tools

- **Postman Collection**: [KB2-Clinical-Context.postman_collection.json](./postman/KB2-Clinical-Context.postman_collection.json)
- **OpenAPI Specification**: [openapi.yaml](../api/openapi.yaml)
- **Test Data Sets**: [test-data/](./test-data/)

### Validation Endpoints

```http
POST /v1/validation/phenotypes
POST /v1/validation/risk-models
POST /v1/validation/treatment-rules
```

## Performance Guidelines

### Request Optimization

- **Batch Processing**: Use batch endpoints for multiple patients
- **Selective Data**: Request only needed data fields
- **Caching**: Implement client-side caching where appropriate
- **Compression**: Enable gzip compression for large responses

### Response Time SLAs

All endpoints include performance SLAs. Monitor response times using:

```bash
# Check current performance
curl -w "@curl-format.txt" -s -o /dev/null http://localhost:8088/v1/health

# Monitor specific endpoint
curl -w "Total: %{time_total}s\n" -s -o /dev/null \
  -X POST http://localhost:8088/v1/phenotypes/evaluate \
  -H "Content-Type: application/json" \
  -d '{"patients": [...]}'
```

## Support and Troubleshooting

### Health Checks

```bash
# Basic health check
curl http://localhost:8088/health

# Readiness check
curl http://localhost:8088/ready

# Detailed system status
curl http://localhost:8088/v1/system/status
```

### Debug Information

Enable debug mode by setting `X-Debug: true` header:

```json
{
  "status": "success",
  "data": {...},
  "debug": {
    "processing_steps": [...],
    "rule_evaluations": [...],
    "cache_hits": {...},
    "performance_breakdown": {...}
  }
}
```

### Common Issues

1. **Authentication Failures**: Verify JWT token validity and permissions
2. **Rate Limiting**: Implement exponential backoff and request queuing
3. **Validation Errors**: Check patient data completeness and format
4. **Performance Issues**: Use batch endpoints and optimize request size

### Contact Information

- **API Support**: api-support@cardiofit.health
- **Clinical Questions**: clinical-informatics@cardiofit.health
- **Technical Issues**: engineering@cardiofit.health
- **Documentation**: docs@cardiofit.health

---

**API Version**: 1.0.0  
**Last Updated**: 2025-01-15  
**Next Review**: 2025-04-15