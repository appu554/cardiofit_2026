# Observation Service GraphQL API

This document describes the GraphQL API for the Observation Service, which is part of the Clinical Synthesis Hub.

## Getting Started

### Prerequisites

- Python 3.8+
- FastAPI
- Graphene
- MongoDB
- Redis (for caching)

### Installation

1. Clone the repository
2. Install dependencies:
   ```
   pip install -r requirements.txt
   ```
3. Set up environment variables (copy `.env.example` to `.env` and update values)
4. Run the service:
   ```
   uvicorn app.main:app --reload --host 0.0.0.0 --port 8004
   ```

## GraphQL API

The GraphQL API is available at `/graphql`.

### Queries

#### Get Observation by ID

```graphql
query GetObservation($id: ID!) {
  observation(id: $id) {
    id
    status
    code {
      coding {
        system
        code
        display
      }
      text
    }
    subject {
      reference
      type
      display
    }
    valueQuantity {
      value
      unit
      system
      code
    }
    effectiveDateTime
    issued
  }
}
```

#### Search Observations

```graphql
query SearchObservations(
  $patientId: String
  $category: String
  $code: String
  $date: String
  $page: Int
  $count: Int
) {
  observations(
    patientId: $patientId
    category: $category
    code: $code
    date: $date
    page: $page
    count: $count
  ) {
    edges {
      node {
        id
        status
        code {
          text
        }
        valueQuantity {
          value
          unit
        }
        effectiveDateTime
      }
      cursor
    }
    pageInfo {
      hasNextPage
      hasPreviousPage
      startCursor
      endCursor
    }
    totalCount
  }
}
```

### Mutations

#### Create Observation

```graphql
mutation CreateObservation($input: CreateObservationInput!) {
  createObservation(input: $input) {
    observation {
      id
      status
      code {
        text
      }
      valueQuantity {
        value
        unit
      }
      effectiveDateTime
    }
  }
}
```

#### Update Observation

```graphql
mutation UpdateObservation($id: ID!, $input: UpdateObservationInput!) {
  updateObservation(id: $id, input: $input) {
    observation {
      id
      status
      code {
        text
      }
      valueQuantity {
        value
        unit
      }
      effectiveDateTime
    }
  }
}
```

#### Delete Observation

```graphql
mutation DeleteObservation($id: ID!) {
  deleteObservation(id: $id) {
    success
  }
}
```

## Authentication

The GraphQL API requires authentication. Include a valid JWT token in the `Authorization` header:

```
Authorization: Bearer <your-jwt-token>
```

## Error Handling

Errors are returned in the following format:

```json
{
  "errors": [
    {
      "message": "Error message",
      "locations": [{"line": 2, "column": 3}],
      "path": ["queryName"]
    }
  ],
  "data": null
}
```

## Rate Limiting

API requests are rate limited to prevent abuse. The current limits are:

- 1000 requests per hour per IP address
- 100 requests per minute per user

## Caching

Responses are cached for 5 minutes by default. You can bypass the cache by including the `Cache-Control: no-cache` header.

## Federation

This service is part of a federated GraphQL architecture. It can be composed with other services using Apollo Federation.

## Testing

Run the test suite:

```
python -m pytest tests/
```

## Deployment

Deploy using Docker:

```bash
docker build -t observation-service .
docker run -d -p 8004:8004 --env-file .env observation-service
```

## Monitoring

Metrics are available at `/metrics` in Prometheus format.

## Logging

Logs are written to `logs/observation-service.log` and include request/response details for debugging.
