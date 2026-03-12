# KB-9 Care Gaps Service

**Care gaps detection and quality measure evaluation for the Clinical Knowledge Platform**

[![SaMD Classification](https://img.shields.io/badge/SaMD-Class%20IIa-blue)](docs/samd-compliance.md)
[![CQL Pattern](https://img.shields.io/badge/CQL-Query--Based-orange)](docs/cql-pattern.md)
[![Da Vinci DEQM](https://img.shields.io/badge/FHIR-Da%20Vinci%20DEQM-green)](docs/deqm.md)

## Overview

KB-9 Care Gaps Service provides clinical quality measure evaluation and care gap detection. It uses the **Query-Based pattern** where CQL queries FHIR directly for longitudinal patient data.

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
| **Depression Screening** | CMS2 | PHQ-2/PHQ-9 screening | Behavioral Health |
| **India Diabetes Care** | Custom | Comprehensive annual care | Chronic Disease |
| **India Hypertension** | Custom | BP + kidney function | Chronic Disease |

## Quick Start

### Build

```bash
# From knowledge-base-services directory
make build-kb-9

# Or from kb-9-care-gaps directory
go build -o bin/kb-9-care-gaps ./cmd/server
```

### Run

```bash
# Using built binary
./bin/kb-9-care-gaps

# Or using go run
go run ./cmd/server
```

### Docker

```bash
docker build -t kb-9-care-gaps .
docker run -p 8089:8089 \
  -e FHIR_SERVER_URL=http://hapi-fhir:8080/fhir \
  -e TERMINOLOGY_URL=http://kb-7-terminology:8087 \
  kb-9-care-gaps
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8089 | Server port |
| `ENVIRONMENT` | development | Environment mode |
| `FHIR_SERVER_URL` | http://localhost:8080/fhir | FHIR Server URL |
| `TERMINOLOGY_URL` | http://localhost:8087 | KB-7 Terminology URL |
| `CQL_LIBRARY_PATH` | ../../vaidshala/clinical-knowledge-core | CQL libraries path |
| `REDIS_URL` | redis://localhost:6379/9 | Redis cache URL |
| `ENABLE_PLAYGROUND` | true | Enable GraphQL Playground |
| `METRICS_ENABLED` | true | Enable Prometheus metrics |

## API Reference

### REST API

#### Get Patient Care Gaps

```bash
curl -X POST http://localhost:8089/api/v1/care-gaps \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "periodStart": "2025-01-01",
    "periodEnd": "2025-12-31",
    "includeEvidence": true
  }'
```

#### Evaluate Single Measure

```bash
curl -X POST http://localhost:8089/api/v1/measure/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "measure": "CMS122_DIABETES_HBA1C",
    "periodStart": "2025-01-01",
    "periodEnd": "2025-12-31"
  }'
```

#### List Available Measures

```bash
curl http://localhost:8089/api/v1/measures
```

### FHIR Operations (Da Vinci DEQM)

#### $care-gaps

```bash
curl -X POST http://localhost:8089/fhir/Measure/\$care-gaps \
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

## Health & Monitoring

| Endpoint | Purpose |
|----------|---------|
| `/health` | Service health |
| `/ready` | Kubernetes readiness |
| `/live` | Kubernetes liveness |
| `/metrics` | Prometheus metrics |

## CQL Integration

KB-9 integrates with the **vaidshala** CQL infrastructure:

- **CQL Libraries**: `vaidshala/clinical-knowledge-core/tier-4-guidelines/`
- **CQL Engine**: `vaidshala/clinical-runtime-platform/engines/`
- **ValueSets**: KB-7 Terminology Service

## Documentation

| Document | Description |
|----------|-------------|
| [User Guide](docs/USER_GUIDE.md) | Comprehensive usage guide with examples |
| [API Reference](docs/API_REFERENCE.md) | Complete API documentation |
| [GraphQL Schema](api/schema.graphql) | GraphQL type definitions |
| [Implementation Plan](KB9_IMPLEMENTATION_PLAN.md) | Architecture decisions |

## Directory Structure

```
kb-9-care-gaps/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── api/                    # HTTP handlers (REST, FHIR, GraphQL)
│   ├── cache/                  # Redis caching layer
│   ├── caregaps/               # Core service logic
│   ├── config/                 # Configuration
│   ├── cql/                    # CQL engine integration
│   ├── deqm/                   # Da Vinci DEQM operations
│   ├── fhir/                   # FHIR client (Google Healthcare API)
│   ├── kb3/                    # KB-3 temporal integration
│   └── models/                 # Domain models
├── api/schema.graphql          # GraphQL schema
├── docs/                       # Documentation
├── tests/
│   ├── integration/            # API integration tests
│   └── unit/                   # Unit tests
├── docker-compose.yml          # Local development environment
├── Dockerfile
├── Makefile                    # Build and test commands
├── go.mod
└── README.md
```

## Testing

```bash
# Run all tests
make test

# Unit tests only
make test-unit

# Integration tests (requires service running)
make test-integration

# With coverage report
make coverage
```

## Development

```bash
# Build binary
make build

# Run locally
make run

# Run with hot reload (dev mode)
make run-dev

# Docker development environment
make docker-up

# Check service health
make health
```

## Tier 7 Integration

KB-9 works with KB-3 as part of the **Tier 7 Longitudinal Intelligence Platform**:

```
KB-9 (Accountability Engine)     KB-3 (Temporal Brain)
"What care is needed?"    ──▶    "When is it due?"
                          ◀──    "What's overdue?"
```

Enable KB-3 integration:
```bash
KB3_URL=http://localhost:8083 KB3_ENABLED=true ./bin/kb-9-care-gaps
```

## License

Proprietary - Clinical Knowledge Platform
