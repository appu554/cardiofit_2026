# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The KB-Drug-Rules service is a Go-based microservice within the Knowledge Base Services architecture that provides TOML-formatted drug calculation rules with clinical governance, digital signatures, and high-performance caching. It serves as a critical component for clinical decision support systems, particularly the Flow2 orchestrator for dose calculations and safety verification.

## Architecture

```
Flow2 Orchestrator → KB-Drug-Rules Service → PostgreSQL/Supabase
                                         ↓
                                    Redis Cache
```

Key components:
- **Go/Gin HTTP Server**: REST API for drug rule management (port 8081)
- **TOML Rule Engine**: Structured clinical drug rules with validation
- **Governance System**: Ed25519 digital signatures with dual approval workflow
- **3-Tier Caching**: In-memory + Redis + database for sub-10ms performance
- **Database Layer**: PostgreSQL/Supabase with GORM ORM

## Common Development Commands

### Quick Start (Recommended)
```bash
cd backend/services/knowledge-base-services

# Start with Docker (isolated ports)
make run-kb-docker              # PostgreSQL on 5433, Redis on 6380

# Build and test
make build                      # Build kb-drug-rules binary
make test                       # Run all tests
make health                     # Check service health
```

### Service-Specific Commands
```bash
cd kb-drug-rules

# Build
go build -o bin/kb-drug-rules ./cmd/server

# Run locally
go run cmd/server/main.go

# Testing
go test ./...                   # All tests
go test -v ./internal/api/...   # API tests only
go test -tags=integration ./tests/integration/...  # Integration tests
```

### Database Operations
```bash
# Local PostgreSQL setup
make setup-local

# Supabase setup
make setup-supabase             # Creates .env.supabase template
make test-supabase             # Test Supabase connection
```

### Development Tools
```bash
make lint                       # Run golangci-lint
make format                     # Format code with gofmt
make test-coverage             # Generate coverage report
make dev-setup                 # Install development tools
```

## Development Workflow

### Starting the Service
1. **Infrastructure**: `make run-kb-docker` (recommended) starts isolated PostgreSQL and Redis
2. **Service**: Automatically built and started via Docker, or manually with `go run cmd/server/main.go`
3. **Validation**: `make health` checks service endpoints

### Service Architecture
```
kb-drug-rules/
├── cmd/server/main.go          # Application entry point
├── internal/
│   ├── api/                    # Gin HTTP handlers and routing
│   │   ├── handlers.go         # Core CRUD operations
│   │   ├── toml_handlers.go    # TOML-specific endpoints
│   │   └── server.go           # Gin server configuration
│   ├── cache/                  # Redis and tiered caching
│   ├── config/                 # Viper configuration management
│   ├── database/               # GORM database operations
│   ├── governance/             # Digital signature and approval workflows
│   ├── metrics/                # Prometheus metrics collection
│   ├── models/                 # Data models and GORM entities
│   ├── services/               # Business logic layer
│   └── validation/             # TOML schema validation
├── migrations/                 # Database migration files
├── sample_rules/               # Example TOML rule files
└── tests/integration/          # Integration test suite
```

## Key Technical Details

### TOML Rule System
The service manages structured drug rules in TOML format:
```toml
[meta]
drug_name = "Metformin"
therapeutic_class = ["Antidiabetic", "Biguanide"]

[dose_calculation]
base_formula = "500mg BID"
max_daily_dose = 2000.0
min_daily_dose = 500.0

[[dose_calculation.adjustment_factors]]
factor = "renal_function"
condition = "egfr < 30"
multiplier = 0.5

[safety_verification]
[[safety_verification.contraindications]]
condition = "Severe renal impairment"
icd10_code = "N18.6"
severity = "absolute"
```

### API Endpoints
- `GET /v1/items/{drug_id}` - Retrieve drug rules with optional region/version filtering
- `POST /v1/validate` - Validate TOML rule content and schema
- `POST /v1/hotload` - Deploy new rules with governance workflow
- `GET /health` - Health check with database connectivity
- `GET /metrics` - Prometheus metrics endpoint

### Data Model
```go
type DrugRulePack struct {
    DrugID         string          `json:"drug_id" gorm:"primaryKey"`
    Version        string          `json:"version" gorm:"primaryKey"`
    ContentSHA     string          `json:"content_sha"`
    SignedBy       string          `json:"signed_by"`
    SignatureValid bool            `json:"signature_valid"`
    Regions        StringArray     `json:"regions" gorm:"type:text[]"`
    Content        json.RawMessage `json:"content" gorm:"type:jsonb"`
    CreatedAt      time.Time       `json:"created_at"`
    UpdatedAt      time.Time       `json:"updated_at"`
}
```

### Caching Strategy
Three-tier caching for optimal performance:
1. **L1 Cache**: In-memory Go map for active rules (sub-millisecond)
2. **L2 Cache**: Redis with TTL for frequently accessed rules (1-5ms)
3. **L3 Storage**: PostgreSQL/Supabase for persistence (5-20ms)

### Clinical Governance
- **Digital Signatures**: Ed25519 cryptographic signing for rule authenticity
- **Dual Approval**: Clinical reviewer + technical reviewer validation required
- **Content Integrity**: SHA256 hashing to detect unauthorized modifications
- **Regional Compliance**: Support for US/EU/CA/AU jurisdictional variations
- **Audit Trail**: Complete change history for regulatory compliance

### Database Configuration
Supports both PostgreSQL and Supabase:

**PostgreSQL (Local/Docker)**:
```bash
DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules
```

**Supabase**:
```bash
SUPABASE_URL=https://your-project-ref.supabase.co
SUPABASE_API_KEY=your-supabase-anon-key
SUPABASE_DB_PASSWORD=your-database-password
```

## Service Configuration

### Environment Variables
```bash
# Server
PORT=8081
DEBUG=true

# Database (choose one)
DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules
# OR
SUPABASE_URL=https://your-project-ref.supabase.co
SUPABASE_API_KEY=your-supabase-anon-key

# Cache
REDIS_URL=redis://localhost:6380/0

# Clinical Governance
SIGNING_KEY_PATH=/app/keys/signing.key
REQUIRE_SIGNATURE=true
REQUIRE_APPROVAL=true
SUPPORTED_REGIONS=US,EU,CA,AU
DEFAULT_REGION=US

# Monitoring
METRICS_ENABLED=true
```

## Testing Strategy

### Test Categories
- **Unit Tests**: `go test ./internal/...` - Service-specific logic
- **Integration Tests**: `go test -tags=integration ./tests/integration/...` - End-to-end workflows
- **API Tests**: Test HTTP endpoints with mock data
- **Performance Tests**: Validate sub-10ms response targets
- **TOML Validation Tests**: Schema compliance and rule parsing

### Key Test Commands
```bash
make test                       # Run all unit tests
make test-integration          # Run integration tests
make test-coverage            # Generate coverage report (target >90%)
make test-performance         # Load testing with Apache Bench
```

## Integration with Flow2

The service is designed to integrate with the Flow2 orchestrator:
```go
// Flow2 client usage
kbClient := NewKBDrugRulesClient("http://localhost:8081")
rules, err := kbClient.GetDrugRules("metformin", "2.1.0", "US")
if err != nil {
    return err
}

// Use rules for dose calculation
dose := calculateDose(rules.Content.DoseCalculation, patientContext)
```

## Performance Targets

| Metric | Target | Implementation |
|--------|--------|---------------|
| P95 Latency | < 10ms | 3-tier caching + optimized queries |
| Cache Hit Rate | > 95% | Redis + in-memory caching |
| Throughput | 10K RPS | Gin framework + connection pooling |
| Availability | 99.9% | Health checks + graceful shutdown |

## Service Ports and Dependencies

- **KB-Drug-Rules Service**: http://localhost:8081
- **PostgreSQL**: localhost:5433 (Docker) / localhost:5432 (local)
- **Redis**: localhost:6380 (Docker) / localhost:6379 (local)
- **Health Check**: http://localhost:8081/health
- **Metrics**: http://localhost:8081/metrics

## Important Notes

### Development Best Practices
- Use the parent Makefile (`make run-kb-docker`) for reliable service startup
- Always validate TOML rules before deployment using `/v1/validate` endpoint
- Run health checks after service changes: `make health`
- Maintain test coverage above 90%: `make test-coverage`

### Database Migration
- GORM auto-migration handles schema updates on startup
- Manual migrations available in `migrations/` directory
- Supabase setup requires running SQL scripts in the Supabase dashboard

### TOML Rule Development
- Rules must pass both schema validation and clinical governance approval
- Regional variations use hierarchical fallback: specific → jurisdiction → global
- Content SHA256 validation ensures rule integrity during transit and storage
- Hot-reload capability allows zero-downtime rule updates

### Security Considerations
- Ed25519 digital signatures prevent unauthorized rule modifications
- Dual approval workflow ensures clinical and technical validation
- Complete audit trails support regulatory compliance requirements
- Regional compliance variations handle jurisdiction-specific regulations