# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## IMPORTANT: Service Location Has Changed

The Knowledge Base Services have been moved to shared infrastructure for better platform-wide accessibility.

**New Location**: `backend/shared-infrastructure/knowledge-base-services/`

**Please refer to the comprehensive documentation at**:
`/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/CLAUDE.md`

## Legacy Documentation (Deprecated)

This directory may contain older KB service code. For active development, use the shared infrastructure location.

## Project Overview (Historical)

The Knowledge Base Services is a microservices architecture providing clinical intelligence for the Clinical Synthesis Hub. The system implements multiple Go-based services that handle drug rules, clinical pathways, and other healthcare knowledge management functions with TOML-based rule definitions, digital signatures, and comprehensive caching.

## Architecture

```
Knowledge Base Services (Go/Gin) → PostgreSQL/Redis/Supabase → Docker Infrastructure
```

Current services:
- **KB-Drug-Rules** (`kb-drug-rules/`): TOML drug calculation rules with validation and caching (port 8081)  
- **KB-Clinical-Pathways** (`kb-clinical-pathways/`): Clinical decision pathways (port 8084)

## Common Development Commands

### Quick Start (Recommended)
```bash
# Initialize and start everything
./init                           # Full setup with Docker
./init --quick                   # Fast Docker setup
./init --mode local              # Local development
```

### Makefile Commands (Primary Interface)
```bash
make help                        # Show all available commands
make run-kb-docker              # Start with Docker PostgreSQL (recommended)
make stop-kb                     # Stop services
make test                        # Run all tests
make build                       # Build services
make health                      # Check service health
make logs-kb                     # View service logs
```

### Manual Service Operations
```bash
# KB-Drug-Rules service
cd kb-drug-rules
go build -o bin/kb-drug-rules ./cmd/server
go run cmd/server/main.go

# KB-Clinical-Pathways service  
cd kb-clinical-pathways
go build -o bin/kb-pathways ./cmd/server
go run cmd/server/main.go

# Testing
go test ./...                    # Run tests in current service
go test -v ./internal/api/...    # Test specific package
```

### Database Operations
```bash
# PostgreSQL (Docker)
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules

# Supabase setup
make setup-supabase              # Setup Supabase configuration
make test-supabase              # Test Supabase connection
```

## Development Workflow

### Starting Services
1. **Infrastructure**: `make run-kb-docker` starts PostgreSQL (port 5433), Redis (port 6380), and Adminer (port 8082)
2. **Services**: Built and started automatically via Docker compose
3. **Testing**: `make test-kb-docker` validates the setup

### Service Architecture Pattern
Each service follows a consistent Go structure:
```
service-name/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── api/                    # HTTP handlers (Gin)
│   ├── config/                 # Configuration management
│   ├── database/               # Database connections
│   ├── models/                 # Data models
│   ├── cache/                  # Redis caching
│   ├── metrics/                # Prometheus metrics
│   └── validation/             # Business logic validation
├── migrations/                 # Database migrations
└── tests/integration/          # Integration tests
```

## Key Technical Details

### TOML Rule System
The KB-Drug-Rules service implements a sophisticated TOML-based rule engine:
- **Rule Structure**: Drug calculation rules defined in TOML format
- **Validation**: Schema validation and business rule checking via `internal/validation/`
- **Versioning**: Immutable versioning with content SHA256 hashing
- **Regional Support**: Multi-region rule variations (US, EU, CA, AU)

### Caching Architecture
- **Tiered Caching**: Local cache → Redis → Database
- **Cache Keys**: Drug ID + version + region combinations
- **Invalidation**: Event-driven cache clearing on rule updates

### Database Integration
- **PostgreSQL**: Primary storage for rule packs and clinical pathways
- **Supabase**: Alternative cloud database option
- **GORM**: ORM for database operations with migration support
- **Connection Pools**: Optimized connection management

### API Design
- **Gin Framework**: HTTP routing and middleware
- **REST Endpoints**: Standardized API patterns
- **Health Checks**: `/health` endpoints for monitoring
- **Metrics**: Prometheus metrics at `/metrics`

### Governance & Security
- **Digital Signatures**: Content signing and verification
- **Approval Workflows**: Clinical + technical review process
- **Audit Trails**: Complete change history tracking
- **Hot Reloading**: Zero-downtime rule updates

## Service Ports and URLs
- KB-Drug-Rules: http://localhost:8081
- KB-Clinical-Pathways: http://localhost:8084  
- PostgreSQL: localhost:5433 (Docker) / localhost:5432 (local)
- Redis: localhost:6380 (Docker) / localhost:6379 (local)
- Adminer (DB UI): http://localhost:8082
- Prometheus: http://localhost:9090 (if using full dev setup)
- Grafana: http://localhost:3000 (if using full dev setup)

## Testing Strategy

### Test Types
- **Unit Tests**: Service-specific logic testing
- **Integration Tests**: Cross-service workflow validation  
- **TOML Tests**: Rule validation and parsing
- **API Tests**: HTTP endpoint testing
- **Performance Tests**: Load and latency testing

### Common Test Commands
```bash
# Run specific test suites
make test                        # All tests
make test-integration           # Integration tests only
make test-performance           # Performance tests
make test-kb-docker             # Docker setup validation
go test ./internal/api/...      # API tests
go test -race ./...             # Race condition detection
```

## Configuration Management

### Environment Variables
```bash
# Database
DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules
REDIS_URL=redis://localhost:6380/0

# Service Configuration
PORT=8081
DEBUG=true
SUPPORTED_REGIONS=US,EU,CA,AU
DEFAULT_REGION=US

# Security & Governance  
SIGNING_KEY_PATH=/app/keys/signing.key
JWT_SECRET=kb-jwt-secret-key-for-development
REQUIRE_APPROVAL=false
REQUIRE_SIGNATURE=false

# Monitoring
METRICS_ENABLED=true
TRACING_ENABLED=false
```

### Configuration Files
- `.env.docker`: Docker environment configuration
- `.env.local`: Local development configuration  
- `.env.supabase.example`: Supabase configuration template

## Important Notes

### Development Best Practices
- Use the Makefile for all service management - it handles complex Docker orchestration
- Always run health checks after starting services: `make health`
- The `/init` script provides the most reliable setup experience
- Services use isolated ports (5433, 6380) to avoid conflicts with system databases

### Docker Integration
- `docker-compose.kb-only.yml`: Minimal services for KB development
- `docker-compose.dev.yml`: Full development environment
- Services auto-configure with environment-specific settings

### TOML Rule Development
- Rules must pass schema validation before deployment
- Regional variations use fallback hierarchy: specific → jurisdiction → global
- Content SHA256 validation ensures rule integrity
- Hot-reload capability allows zero-downtime updates

### Database Design
- Each service maintains separate database schemas
- GORM handles migrations and model management
- Optimized indexes for drug_id, version, and region queries

### Monitoring and Observability  
- Prometheus metrics integrated into all services
- Structured logging with correlation IDs
- Health check endpoints for service discovery
- Performance metrics track sub-10ms response times