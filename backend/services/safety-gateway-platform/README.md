# Safety Gateway Platform

A production-ready safety validation platform that orchestrates multiple in-process safety engines to provide sub-200ms clinical decision support.

## Overview

The Safety Gateway Platform is designed to provide real-time safety validation for clinical decisions with the following key features:

- **Sub-200ms Response Time**: Optimized for real-time clinical workflows
- **In-Process Engine Architecture**: All safety engines run in the same process for maximum performance
- **Fail-Closed Safety**: Critical engines failures result in unsafe decisions to ensure patient safety
- **Multi-Level Caching**: L1 (in-memory) and L2 (Redis) caching for clinical context
- **Circuit Breaker Protection**: Automatic engine protection and recovery
- **Comprehensive Observability**: Structured logging, metrics, and audit trails
- **Override Token System**: Secure clinical override capabilities with RBAC

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   gRPC Client   │───▶│  Safety Gateway  │───▶│  Safety Engines │
│                 │    │    Platform      │    │   (In-Process)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │ Context Assembly │
                       │   Service        │
                       └──────────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
            ┌──────────────┐    ┌──────────────┐
            │ FHIR Service │    │   GraphDB    │
            └──────────────┘    └──────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc)
- Redis (optional, for L2 caching)
- Docker (optional)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd safety-gateway-platform
```

2. Install dependencies:
```bash
make deps
```

3. Generate protobuf files:
```bash
make proto
```

4. Build the application:
```bash
make build
```

5. Run the application:
```bash
make run
```

The service will start on port 8030 by default.

### Docker

Build and run with Docker:

```bash
make docker
make docker-run
```

## Configuration

The service is configured via `config.yaml`. Key configuration sections:

### Service Configuration
```yaml
service:
  name: "safety-gateway-platform"
  port: 8030
  version: "1.0.0"
  environment: "development"
```

### Performance Configuration
```yaml
performance:
  max_concurrent_requests: 1000
  request_timeout_ms: 200
  context_assembly_timeout_ms: 20
  engine_execution_timeout_ms: 150
```

### Engine Configuration
```yaml
engines:
  cae_service:
    enabled: true
    timeout_ms: 100
    priority: 10
    tier: 1  # TierVetoCritical
```

## API Reference

### gRPC Service

The platform exposes a gRPC service with the following methods:

#### ValidateSafety
Validates a clinical safety request.

```protobuf
rpc ValidateSafety(SafetyRequest) returns (SafetyResponse);
```

**Request:**
```json
{
  "request_id": "req_123",
  "patient_id": "patient_456",
  "clinician_id": "clinician_789",
  "action_type": "medication_order",
  "priority": "normal",
  "medication_ids": ["med_1", "med_2"],
  "condition_ids": ["cond_1"],
  "allergy_ids": ["allergy_1"]
}
```

**Response:**
```json
{
  "request_id": "req_123",
  "status": "SAFETY_STATUS_SAFE",
  "risk_score": 0.15,
  "engine_results": [...],
  "processing_time_ms": 45,
  "explanation": {...}
}
```

#### GetEngineStatus
Returns the status of safety engines.

```protobuf
rpc GetEngineStatus(EngineStatusRequest) returns (EngineStatusResponse);
```

## Safety Engines

The platform includes several built-in safety engines:

### Clinical Assertion Engine (CAE)
- **Capabilities**: drug_interaction, contraindication, dosing
- **Tier**: 1 (Veto-Critical)
- **Timeout**: 100ms

### Allergy Check Engine
- **Capabilities**: allergy_check, contraindication
- **Tier**: 1 (Veto-Critical)
- **Timeout**: 80ms

### Protocol Engine
- **Capabilities**: clinical_protocol, guideline_compliance
- **Tier**: 2 (Advisory)
- **Timeout**: 80ms

### Constraint Validator
- **Capabilities**: hard_constraints, safety_limits
- **Tier**: 1 (Veto-Critical)
- **Timeout**: 60ms

## Safety Decision Logic

The platform uses tier-based aggregation rules:

1. **Tier 1 (Veto-Critical)**: Any UNSAFE result → Final: UNSAFE
2. **Tier 1 Failures**: Any failure → Final: UNSAFE (fail-closed)
3. **Tier 2 (Advisory)**: Failures → WARNING (degraded)
4. **All Tier 1 SAFE**: Proceed to Tier 2 evaluation

## Development

### Running Tests
```bash
make test
```

### Running with Coverage
```bash
make coverage
```

### Linting
```bash
make lint
```

### Formatting
```bash
make fmt
```

### Building for Multiple Platforms
```bash
make build-all
```

## Monitoring and Observability

### Health Checks
```bash
curl http://localhost:8030/health
```

### Metrics
The service exposes Prometheus metrics on port 9090 (configurable).

### Logging
Structured JSON logging with configurable levels:
- Request/response logging
- Engine execution logging
- Audit logging for safety decisions
- Circuit breaker events

### Audit Trail
All safety decisions are logged with:
- Request ID and patient ID (hashed)
- Decision status and risk score
- Engine results and failures
- Processing time
- Override attempts

## Security

### Authentication
- gRPC metadata-based authentication
- Configurable auth interceptor
- Support for JWT tokens

### Authorization
- Role-based access control (RBAC)
- Override token system with cryptographic signatures
- Configurable override levels (resident, attending, pharmacist, chief)

### Compliance
- HIPAA-compliant logging (PII hashing)
- Audit retention (configurable, default 7 years)
- Encryption at rest and in transit

## Performance

### Benchmarks
- Target: Sub-200ms response time
- Context assembly: <20ms
- Engine execution: <150ms
- Concurrent requests: 1000+

### Caching
- L1 Cache: In-memory with LRU eviction
- L2 Cache: Redis with configurable TTL
- Context versioning for cache invalidation

### Circuit Breaker
- Configurable failure thresholds
- Automatic recovery with half-open state
- Per-engine circuit breaker isolation

## Deployment

### Environment Variables
```bash
SGP_PORT=8030
SGP_DB_HOST=localhost
SGP_DB_USER=postgres
SGP_DB_PASSWORD=password
SGP_REDIS_ADDRESS=localhost:6379
SGP_SIGNING_KEY=your-secret-key
```

### Docker Compose
```yaml
version: '3.8'
services:
  safety-gateway:
    image: safety-gateway-platform:latest
    ports:
      - "8030:8030"
    environment:
      - SGP_ENVIRONMENT=production
    depends_on:
      - redis
      - postgres
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make test lint fmt`
6. Submit a pull request

## License

[License information]

## Support

For support and questions, please contact the development team or create an issue in the repository.
