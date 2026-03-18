# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The Knowledge Base Services is a comprehensive microservices architecture providing clinical intelligence infrastructure for the CardioFit platform. This shared infrastructure implements multiple Go-based services that handle drug rules, clinical pathways, guidelines, safety protocols, and terminology management with TOML-based rule definitions, digital signatures, and comprehensive caching.

## Architecture

```
Clinical Services → Knowledge Base Services (Go/Gin) → PostgreSQL/Redis/Supabase → Docker Infrastructure
                                                    ↓
                               Clinical Reasoning Service (Python/Neo4j)
```

Current services:
- **KB-Drug-Rules** (`kb-drug-rules/`): TOML drug calculation rules with validation and caching (port 8081)
- **KB-Guideline-Evidence** (`kb-guideline-evidence/`): Clinical guidelines and evidence-based protocols (port 8084)
- **KB-1-Drug-Rules** (`kb-1-drug-rules/`): Foundational drug rules service
- **KB-2-Clinical-Context** (`kb-2-clinical-context/`): Clinical context and patient state management
- **KB-3-Guidelines** (`kb-3-guidelines/`): Comprehensive clinical guidelines repository
- **KB-4-Patient-Safety** (`kb-4-patient-safety/`): Patient safety protocols and alerts
- **KB-5-Drug-Interactions** (`kb-5-drug-interactions/`): Drug-drug interaction rules
- **KB-6-Formulary** (`kb-6-formulary/`): Institutional formulary management
- **KB-7-Terminology** (`kb-7-terminology/`): Medical terminology and coding systems
- **KB-Cross-Dependency-Manager** (`kb-cross-dependency-manager/`): Inter-KB dependency coordination

### Vaidshala Clinical Runtime Services (KB-19+)
- **KB-19-Protocol-Orchestrator** (`kb-19-protocol-orchestrator/`): Protocol arbitration engine with conflict resolution and CQL integration (port 8103)
- **KB-20-Patient-Profile** (`kb-20-patient-profile/`): Patient stratum engine, eGFR trajectory, lab plausibility checking (port 8131)
- **KB-21-Behavioral-Intelligence** (`kb-21-behavioral-intelligence/`): Adherence scoring, answer reliability, behavioural gap detection (port 8133)
- **KB-22-HPI-Engine** (`kb-22-hpi-engine/`): History of Present Illness session engine with Bayesian differential diagnosis (port 8132)
- **KB-23-Decision-Cards** (`kb-23-decision-cards/`): Decision card rendering with MCU gate, confidence tiers, SLA monitoring (port 8134)
- **KB-25-Lifestyle-Knowledge-Graph** (`kb-25-lifestyle-knowledge-graph/`): Causal reasoning graph for food/exercise interventions with EffectDescriptors, safety hard-stops, comparator engine (port 8136)
- **KB-26-Metabolic-Digital-Twin** (`kb-26-metabolic-digital-twin/`): Persisted derived twin state, coupled forward simulation, Bayesian patient-specific calibration (port 8137)

## Common Development Commands

### Quick Start (Recommended)
```bash
# From knowledge-base-services directory
./init                           # Full setup with Docker
./init --quick                   # Fast Docker setup
./init --mode local              # Local development mode
```

### Makefile Commands (Primary Interface)
```bash
make help                        # Show all available commands
make run-kb-docker              # Start all KB services with Docker PostgreSQL (recommended)
make stop-kb                     # Stop all services
make test                        # Run all tests across services
make build                       # Build all service binaries
make health                      # Check health of all services
make logs-kb                     # View service logs
```

### Individual Service Operations
```bash
# Navigate to specific service
cd kb-drug-rules
cd kb-guideline-evidence
cd kb-2-clinical-context
# ... etc

# Build service
go build -o bin/<service-name> ./cmd/server

# Run service locally
go run cmd/server/main.go

# Test service
go test ./...                    # All tests
go test -v ./internal/api/...    # API tests only
go test -tags=integration ./tests/integration/...  # Integration tests
```

### Database Operations
```bash
# PostgreSQL (Docker - isolated ports)
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules

# Supabase setup
make setup-supabase              # Setup Supabase configuration
make test-supabase              # Test Supabase connection

# Database migrations
make migrate-up                  # Apply migrations
make migrate-down               # Rollback migrations
```

## Development Workflow

### Starting Services
1. **Infrastructure First**: `make run-kb-docker` starts PostgreSQL (port 5433), Redis (port 6380), and Adminer (port 8082)
2. **Services Auto-Start**: All KB services are built and started automatically via Docker compose
3. **Validation**: `make health` checks all service endpoints
4. **Monitoring**: Access Adminer at http://localhost:8082 for database management

### Service Architecture Pattern
Each KB service follows a consistent Go structure:
```
kb-service-name/
├── cmd/server/main.go          # Application entry point
├── internal/
│   ├── api/                    # Gin HTTP handlers and routing
│   │   ├── handlers.go         # Core CRUD operations
│   │   ├── toml_handlers.go    # TOML-specific endpoints (if applicable)
│   │   └── server.go           # Gin server configuration
│   ├── cache/                  # Redis and tiered caching
│   ├── config/                 # Viper configuration management
│   ├── database/               # GORM database operations
│   ├── governance/             # Digital signature and approval workflows
│   ├── metrics/                # Prometheus metrics collection
│   ├── models/                 # Data models and GORM entities
│   ├── services/               # Business logic layer
│   └── validation/             # Schema and business rule validation
├── migrations/                 # Database migration files
├── sample_rules/               # Example TOML rule files (if applicable)
├── tests/integration/          # Integration test suite
└── docs/                       # Service-specific documentation
```

## Key Technical Details

### TOML Rule System
KB services implement sophisticated TOML-based rule engines:
- **Rule Structure**: Clinical rules defined in structured TOML format
- **Validation**: Schema validation and business rule checking via `internal/validation/`
- **Versioning**: Immutable versioning with content SHA256 hashing
- **Regional Support**: Multi-region rule variations (US, EU, CA, AU)
- **Hot Reloading**: Zero-downtime rule updates with governance approval

### Caching Architecture
- **Tiered Caching**: Local in-memory → Redis → PostgreSQL/Supabase
- **Cache Keys**: Drug/guideline/entity ID + version + region combinations
- **Invalidation**: Event-driven cache clearing on rule updates
- **Performance**: Sub-10ms P95 latency target for cached queries

### Database Integration
- **PostgreSQL**: Primary storage for all KB rule packs and clinical data
- **Supabase**: Alternative cloud database option with real-time capabilities
- **GORM**: ORM for database operations with migration support
- **Connection Pools**: Optimized connection management per service
- **Isolated Ports**: Docker services use port 5433 to avoid conflicts with system databases

### API Design
- **Gin Framework**: HTTP routing and middleware for all services
- **REST Endpoints**: Standardized API patterns across all KB services
- **Health Checks**: `/health` endpoints for monitoring and orchestration
- **Metrics**: Prometheus metrics at `/metrics` for observability
- **Versioned APIs**: `/v1/` prefix for API stability

### Clinical Governance & Security
- **Digital Signatures**: Ed25519 cryptographic signing for rule authenticity
- **Dual Approval**: Clinical reviewer + technical reviewer validation required
- **Audit Trails**: Complete change history tracking for regulatory compliance
- **Content Integrity**: SHA256 hashing to detect unauthorized modifications
- **Regional Compliance**: Support for jurisdiction-specific variations

### Service Communication
- **REST APIs**: Primary communication between KB services
- **Event Streaming**: Kafka integration for rule change notifications
- **gRPC**: Optional high-performance communication (future enhancement)
- **Service Discovery**: Health check based discovery for orchestration

## Service Ports and URLs

### KB Services
- KB-Drug-Rules: http://localhost:8081
- KB-Guideline-Evidence: http://localhost:8084
- KB-1-Drug-Rules: http://localhost:8085
- KB-2-Clinical-Context: http://localhost:8086
- KB-3-Guidelines: http://localhost:8087
- KB-4-Patient-Safety: http://localhost:8088
- KB-5-Drug-Interactions: http://localhost:8089
- KB-6-Formulary: http://localhost:8091
- KB-7-Terminology: http://localhost:8092

### Vaidshala Clinical Runtime Services
- KB-19-Protocol-Orchestrator: http://localhost:8103
- KB-20-Patient-Profile: http://localhost:8131
- KB-21-Behavioral-Intelligence: http://localhost:8133
- KB-22-HPI-Engine: http://localhost:8132
- KB-23-Decision-Cards: http://localhost:8134
- KB-25-Lifestyle-Knowledge-Graph: http://localhost:8136
- KB-26-Metabolic-Digital-Twin: http://localhost:8137

### Infrastructure Services
- PostgreSQL: localhost:5433 (Docker) / localhost:5432 (local)
- Redis: localhost:6380 (Docker) / localhost:6379 (local)
- Adminer (DB UI): http://localhost:8082
- Prometheus: http://localhost:9090 (if using full dev setup)
- Grafana: http://localhost:3000 (if using full dev setup)

## Testing Strategy

### Test Types
- **Unit Tests**: Service-specific logic testing with Go testing framework
- **Integration Tests**: Cross-service workflow validation with tags
- **TOML Tests**: Rule validation and parsing verification
- **API Tests**: HTTP endpoint testing with mock data
- **Performance Tests**: Load testing and latency validation
- **Contract Tests**: API contract verification between services

### Common Test Commands
```bash
# Run all tests across all services
make test                        # All unit tests
make test-integration           # Integration tests only
make test-performance           # Performance tests with load testing
make test-coverage              # Generate coverage reports (target >90%)

# Service-specific testing
cd kb-drug-rules
go test ./...                   # All tests for this service
go test -v ./internal/api/...   # Verbose API tests
go test -race ./...             # Race condition detection
go test -tags=integration ./tests/integration/...  # Integration tests only
```

## Configuration Management

### Environment Variables
```bash
# Database Configuration
DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules
REDIS_URL=redis://localhost:6380/0

# Service Configuration
PORT=8081                        # Service-specific port
DEBUG=true                       # Enable debug logging
SUPPORTED_REGIONS=US,EU,CA,AU    # Supported jurisdictions
DEFAULT_REGION=US                # Default region if not specified

# Security & Governance
SIGNING_KEY_PATH=/app/keys/signing.key
JWT_SECRET=kb-jwt-secret-key-for-development
REQUIRE_APPROVAL=false           # Enable dual approval workflow
REQUIRE_SIGNATURE=false          # Enable digital signatures

# Monitoring & Observability
METRICS_ENABLED=true             # Enable Prometheus metrics
TRACING_ENABLED=false            # Enable distributed tracing
LOG_LEVEL=info                   # Logging level (debug, info, warn, error)
```

### Configuration Files
- `.env.docker`: Docker environment configuration
- `.env.local`: Local development configuration
- `.env.supabase.example`: Supabase configuration template
- `docker-compose.kb-only.yml`: Minimal KB services for development
- `docker-compose.dev.yml`: Full development environment with monitoring

## Integration with Platform Services

### Flow2 Orchestrator Integration
KB services integrate with the Flow2 Go orchestrator for clinical decision support:
```go
// Flow2 client usage example
kbClient := NewKBDrugRulesClient("http://localhost:8081")
rules, err := kbClient.GetDrugRules("metformin", "2.1.0", "US")
if err != nil {
    return err
}

// Use rules for dose calculation
dose := calculateDose(rules.Content.DoseCalculation, patientContext)
```

### Medication Service Integration
The Python medication service at `backend/services/medication-service/` consumes KB services for:
- Drug dosing calculations
- Clinical guideline validation
- Safety verification
- Drug interaction checking

### Clinical Reasoning Service
Integration with Python/Neo4j clinical reasoning service for:
- Knowledge graph enrichment
- Evidence-based recommendations
- Clinical pathway navigation

## Performance Targets

| Metric | Target | Implementation Strategy |
|--------|--------|------------------------|
| P95 Latency | < 10ms | 3-tier caching + optimized queries |
| Cache Hit Rate | > 95% | Redis + in-memory caching |
| Throughput | 10K RPS | Gin framework + connection pooling |
| Availability | 99.9% | Health checks + graceful shutdown |
| Data Integrity | 100% | SHA256 validation + digital signatures |

## Important Notes

### Development Best Practices
- **Use Makefile**: Always use `make` commands for service management - handles complex Docker orchestration
- **Health Checks**: Run `make health` after starting services to validate all endpoints
- **Init Script**: The `./init` script provides the most reliable setup experience
- **Isolated Ports**: Services use isolated ports (5433, 6380) to avoid conflicts with system databases
- **Test Coverage**: Maintain >90% test coverage for all services

### Docker Integration
- **kb-only Compose**: `docker-compose.kb-only.yml` provides minimal services for KB development
- **Full Dev Environment**: `docker-compose.dev.yml` includes monitoring and observability
- **Auto-Configuration**: Services auto-configure with environment-specific settings
- **Volume Persistence**: PostgreSQL and Redis data persisted across container restarts

### TOML Rule Development
- **Schema Validation**: Rules must pass schema validation before deployment
- **Regional Hierarchy**: Regional variations use fallback: specific → jurisdiction → global
- **Content Integrity**: SHA256 validation ensures rule integrity during transit and storage
- **Hot Reload**: Zero-downtime rule updates with governance approval workflow
- **Sample Rules**: Each service includes sample TOML files in `sample_rules/` directory

### Database Design
- **Separate Schemas**: Each service maintains separate database schemas
- **GORM Migrations**: Auto-migration handles schema updates on startup
- **Optimized Indexes**: Indexes on drug_id, version, region, and content_sha for performance
- **Supabase Support**: Cloud database option for production deployments

### Monitoring and Observability
- **Prometheus Integration**: All services expose metrics at `/metrics`
- **Structured Logging**: JSON logging with correlation IDs for request tracking
- **Health Endpoints**: Service discovery and orchestration via health checks
- **Performance Metrics**: Track P95 latency, cache hit rates, and throughput

### Security Considerations
- **Authentication**: JWT-based authentication for all API endpoints
- **Authorization**: Role-based access control for clinical vs technical users
- **Encryption**: TLS/SSL for data in transit, encryption at rest for sensitive data
- **Audit Compliance**: Complete audit trails support HIPAA and regulatory requirements

## Migration from Medication Service

The KB services were originally located at `backend/services/medication-service/knowledge-bases/` and have been migrated to shared infrastructure for better reusability across the platform.

**Key Changes**:
- **Location**: Now at `backend/shared-infrastructure/knowledge-base-services/`
- **Access**: Available to all services via network (not just medication service)
- **Ports**: Standardized port allocation (8081-8092 range)
- **Configuration**: Unified environment variable naming across services
- **Documentation**: Comprehensive CLAUDE.md at both root and service levels

**Migration Benefits**:
- Shared infrastructure reduces duplication
- Better service isolation and scalability
- Standardized development workflows
- Improved testing and monitoring capabilities
