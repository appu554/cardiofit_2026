# Medication Service V2: Go/Rust Architecture

A high-performance medication management service built with Go and Rust, implementing the Recipe & Snapshot architecture pattern for clinical decision support.

## Overview

Medication Service V2 is a complete rewrite of the medication management system using Go for orchestration and Rust for high-performance clinical calculations. This service implements the 4-phase workflow pattern with internal Recipe Resolution and Context Gateway integration for immutable clinical snapshots.

### Key Features

- **4-Phase Workflow**: Ingestion & Recipe Resolution → Context Assembly → Clinical Intelligence → Proposal Generation
- **Internal Recipe Resolution**: No external dependencies for recipe logic
- **Immutable Clinical Snapshots**: Ensures data consistency across Calculate → Validate → Commit pattern
- **High Performance**: <250ms end-to-end processing with <10ms recipe resolution
- **Multi-Language Architecture**: Go for orchestration, Rust for clinical calculations
- **Knowledge Base Integration**: Apollo Federation for clinical knowledge access

## Architecture

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   Go Main Service   │────│  Rust Clinical      │────│  Knowledge Bases    │
│   (Port 8005)       │    │  Engine (Port 8095) │    │  (Ports 8086, 8089)│
│                     │    │                     │    │                     │
│ • Recipe Resolution │    │ • Dose Calculations │    │ • Drug Rules        │
│ • Context Gateway   │    │ • Safety Checks     │    │ • Guidelines        │
│ • Apollo Federation │    │ • Performance Opt   │    │ • Evidence Base     │
│ • 4-Phase Workflow  │    │ • Clinical Logic    │    │ • TOML Rule Engine  │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
         │                           │                           │
         └───────────────────────────┼───────────────────────────┘
                                     │
         ┌─────────────────────────────────────────────────────────┐
         │              Flow2 Go Engine (Port 8085)                │
         │          Clinical Orchestration & Intelligence          │
         └─────────────────────────────────────────────────────────┘
```

## Service Components

### Go Main Service (Port 8005)
- **Recipe Resolver**: Internal recipe management and resolution
- **Context Gateway Client**: Immutable snapshot creation and management
- **Apollo Federation Client**: Knowledge base access and integration
- **4-Phase Orchestrator**: Complete medication workflow management
- **gRPC/HTTP APIs**: External and internal service communication

### Rust Clinical Engine (Port 8095)
- **High-Performance Calculations**: Weight-based, BSA-based, AUC-based dosing
- **Safety Constraint Engine**: Real-time safety validation and constraint checking
- **Adjustment Algorithms**: Renal, hepatic, age-based dose adjustments
- **Rounding Rules Engine**: Medication-specific dose rounding and precision

### Flow2 Go Engine V2 (Port 8085)
- **Clinical Intelligence**: Advanced clinical reasoning and decision support
- **Candidate Generation**: Therapeutic option generation and evaluation
- **Scoring Engine**: Multi-criteria medication ranking and optimization
- **Protocol Management**: Complex treatment protocol orchestration

### Knowledge Base Services V2
- **KB Drug Rules** (Port 8086): Dosing guidelines, drug interactions, safety rules
- **KB Guideline Evidence** (Port 8089): Clinical guidelines, evidence-based protocols

## Quick Start

### Prerequisites
- Go 1.21+
- Rust 1.70+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### Installation

1. **Clone and Setup**
```bash
cd backend/services/medication-service-v2
make setup  # Copy and configure components from existing service
```

2. **Start All Services**
```bash
make run-all
```

3. **Verify Health**
```bash
make health-all
```

### Service URLs
- Go Main Service: http://localhost:8005
- Rust Clinical Engine: http://localhost:8095
- Flow2 Go Engine: http://localhost:8085
- KB Drug Rules: http://localhost:8086
- KB Guidelines: http://localhost:8089

## API Overview

### Core Endpoints

#### Medication Proposals
```http
POST /api/v1/medications/propose
{
  "patient_id": "patient-123",
  "indication": "hypertension",
  "clinical_context": {
    "weight_kg": 70,
    "age_years": 45
  }
}
```

#### Recipe Resolution
```http
POST /api/v1/recipes/resolve
{
  "protocol_id": "hypertension-standard",
  "context_needs": {
    "calculation_fields": ["weight", "age"],
    "safety_fields": ["allergies", "conditions"]
  }
}
```

#### Clinical Snapshots
```http
POST /api/v1/snapshots/create
{
  "patient_id": "patient-123",
  "recipe": {...},
  "freshness_requirements": {...}
}
```

## Performance Targets

- **End-to-End Latency**: <250ms (95th percentile)
- **Recipe Resolution**: <10ms
- **Snapshot Creation**: <100ms
- **Clinical Calculations**: <50ms
- **Throughput**: >1000 requests/second

## Configuration

### Environment Variables
```bash
# Database Configuration
DATABASE_URL=postgresql://user:pass@localhost:5434/medication_v2
REDIS_URL=redis://localhost:6381

# Service Dependencies
CONTEXT_GATEWAY_URL=http://localhost:8020
APOLLO_FEDERATION_URL=http://localhost:4000/graphql

# Performance Tuning
MAX_CONCURRENT_CALCULATIONS=100
CACHE_TTL_SECONDS=300
SNAPSHOT_EXPIRY_HOURS=24
```

### Service Configuration
```yaml
# config/service.yaml
service:
  name: medication-service-v2
  version: "1.0.0"
  port: 8005

recipe_resolver:
  cache_enabled: true
  cache_ttl: "10m"
  default_ttl: "1h"

clinical_engine:
  rust_engine_url: "http://localhost:8095"
  timeout: "30s"
  max_retries: 3

knowledge_bases:
  drug_rules_url: "http://localhost:8086"
  guidelines_url: "http://localhost:8089"
  apollo_federation_url: "http://localhost:4000/graphql"
```

## Development

### Running Tests
```bash
# Go tests
go test ./...

# Rust tests  
cd flow2-rust-engine-v2 && cargo test

# Integration tests
make test-integration

# Performance tests
make test-performance
```

### Building
```bash
# Build all components
make build-all

# Build specific components
make build-go
make build-rust
make build-knowledge-bases
```

### Development Commands
```bash
make help              # Show all available commands
make setup             # Initial setup and component migration
make run-dev           # Run in development mode with hot reload
make logs              # View service logs
make clean             # Clean build artifacts and temporary files
```

## Monitoring & Observability

### Metrics
- Prometheus metrics exposed on `/metrics`
- Custom business metrics for clinical operations
- Performance and error rate monitoring

### Health Checks
- Liveness probe: `/health/live`
- Readiness probe: `/health/ready`
- Dependency health: `/health/deps`

### Logging
- Structured JSON logging
- Distributed tracing with OpenTelemetry
- Clinical audit trails for compliance

## Migration from Python Service

The service is designed to run alongside the existing Python medication service during migration:

### Port Mapping
| Component | Python Service | Go/Rust Service V2 |
|-----------|---------------|-------------------|
| Main API | 8004 | 8005 |
| Flow2 Go | 8080 | 8085 |
| Rust Engine | 8090 | 8095 |
| KB Drug Rules | 8081 | 8086 |
| KB Guidelines | 8084 | 8089 |

### Migration Strategy
1. **Parallel Deployment**: Run both services simultaneously
2. **Feature Flagging**: Route specific requests to V2 for testing
3. **Performance Validation**: Compare response times and accuracy
4. **Gradual Cutover**: Incrementally shift traffic to Go/Rust service

## Contributing

### Code Standards
- Go: Follow effective Go guidelines, use gofmt and golint
- Rust: Follow Rust API guidelines, use rustfmt and clippy
- Testing: Maintain >80% code coverage
- Documentation: Update docs for all public APIs

### Pull Request Process
1. Create feature branch from main
2. Implement changes with tests
3. Run full test suite: `make test-all`
4. Update documentation if needed
5. Submit PR with detailed description

## Support & Resources

- **Architecture Docs**: [docs/architecture.md](docs/architecture.md)
- **API Reference**: [docs/api-reference.md](docs/api-reference.md)
- **Deployment Guide**: [docs/deployment.md](docs/deployment.md)
- **Troubleshooting**: [docs/troubleshooting.md](docs/troubleshooting.md)

## License

Part of the Clinical Synthesis Hub CardioFit platform. See main project LICENSE for details.