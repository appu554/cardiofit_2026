# Apollo Federation Testing Guide

## Overview

This guide provides examples for testing the Apollo Federation Gateway with both Patient and Observation services integrated using Google Healthcare API.

## Prerequisites

1. **Services Running:**
   - Patient Service: `http://localhost:8003`
   - Observation Service: `http://localhost:8007`
   - Apollo Federation Gateway: `http://localhost:4000`

2. **Authentication:**
   - You'll need a valid JWT token for authenticated requests
   - Add `Authorization: Bearer <your-token>` header

## Testing Endpoints

### Apollo Federation Gateway
- **GraphQL Endpoint:** `http://localhost:4000/graphql`
- **Health Check:** `http://localhost:4000/health`

### Direct Service Endpoints (for comparison)
- **Patient Service:** `http://localhost:8003/graphql`
- **Observation Service:** `http://localhost:8007/graphql`

## Test Mutations

### 1. Create Simple Observation

**File:** `mutations/create-observation-simple.json`

```bash
curl -X POST http://localhost:4000/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d @mutations/create-observation-simple.json
```

### 2. Create Heart Rate Observation

**File:** `mutations/create-observation.json`

```bash
curl -X POST http://localhost:4000/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d @mutations/create-observation.json
```

### 3. Create Blood Pressure Observation

**File:** `mutations/create-blood-pressure-observation.json`

```bash
curl -X POST http://localhost:4000/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d @mutations/create-blood-pressure-observation.json
```

## Test Queries

### 1. Get Observations

**File:** `queries/get-observations.json`

```bash
curl -X POST http://localhost:4000/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d @queries/get-observations.json
```

### 2. Combined Patient and Observation Query

**File:** `queries/combined-patient-observations.json`

```bash
curl -X POST http://localhost:4000/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d @queries/combined-patient-observations.json
```

## PowerShell Examples

### Create Observation (PowerShell)

```powershell
$headers = @{
    "Content-Type" = "application/json"
    "Authorization" = "Bearer YOUR_JWT_TOKEN"
}

$body = Get-Content "mutations/create-observation-simple.json" -Raw

Invoke-WebRequest -Uri "http://localhost:4000/graphql" -Method POST -Headers $headers -Body $body
```

### Get Observations (PowerShell)

```powershell
$headers = @{
    "Content-Type" = "application/json"
    "Authorization" = "Bearer YOUR_JWT_TOKEN"
}

$body = Get-Content "queries/get-observations.json" -Raw

Invoke-WebRequest -Uri "http://localhost:4000/graphql" -Method POST -Headers $headers -Body $body
```

## GraphQL Playground

You can also test using GraphQL Playground or any GraphQL client:

1. Open `http://localhost:4000/graphql` in your browser
2. Add authentication header:
   ```json
   {
     "Authorization": "Bearer YOUR_JWT_TOKEN"
   }
   ```

## Sample Mutations and Queries

### Create Observation Mutation

```graphql
mutation CreateObservation($input: ObservationInput!) {
  createObservation(input: $input) {
    success
    message
    observation {
      id
      resourceType
      status
      code {
        text
      }
      subject {
        reference
      }
      effectiveDateTime
      valueQuantity {
        value
        unit
      }
    }
    errors
  }
}
```

### Get Observations Query

```graphql
query GetObservations($page: Int, $count: Int) {
  observations(page: $page, count: $count) {
    id
    status
    code {
      text
    }
    subject {
      reference
    }
    effectiveDateTime
    valueQuantity {
      value
      unit
    }
  }
}
```

### Combined Query

```graphql
query GetPatientsAndObservations {
  patients(page: 1, limit: 5) {
    items {
      id
      name {
        given
        family
      }
      gender
      birthDate
    }
    total
  }
  
  observations(page: 1, count: 10) {
    id
    status
    code {
      text
    }
    subject {
      reference
    }
    effectiveDateTime
    valueQuantity {
      value
      unit
    }
  }
}
```

## Expected Responses

### Successful Observation Creation

```json
{
  "data": {
    "createObservation": {
      "success": true,
      "message": "Observation created successfully",
      "observation": {
        "id": "generated-observation-id",
        "resourceType": "Observation",
        "status": "final",
        "code": {
          "text": "Heart rate"
        },
        "subject": {
          "reference": "Patient/example-patient-id"
        },
        "effectiveDateTime": "2024-01-15T10:30:00Z",
        "valueQuantity": {
          "value": 72.0,
          "unit": "beats/min"
        }
      },
      "errors": null
    }
  }
}
```

## Troubleshooting

### Common Issues

1. **Authentication Errors:** Ensure JWT token is valid and not expired
2. **Service Unavailable:** Check if all services are running
3. **Schema Errors:** Verify supergraph schema is properly generated
4. **CORS Issues:** Ensure proper CORS headers are set

### Debug Commands

```bash
# Check service health
curl http://localhost:8003/health
curl http://localhost:8007/health
curl http://localhost:4000/health

# Check federation endpoints
curl -X POST http://localhost:8003/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "{ _service { sdl } }"}'

curl -X POST http://localhost:8007/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "{ _service { sdl } }"}'
```

## Next Steps

1. Test the mutations and queries provided
2. Create additional observation types (temperature, weight, etc.)
3. Implement cross-service relationships
4. Add subscription support for real-time updates
