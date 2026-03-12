# KB-5 Drug Interactions Service - Testing Guide

## Service Overview

**Service Name**: KB-5 Enhanced Drug Interactions Service
**Directory**: `/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions`
**Language**: Go 1.21
**Framework**: Gin (HTTP) + gRPC
**Version**: 2.0.0 (Enhanced)

## Service Architecture

### Primary Technologies
- **Go Framework**: Gin Web Framework
- **Database**: PostgreSQL (GORM ORM)
- **Caching**: Redis (dual hot/warm cache architecture)
- **Protocol**: HTTP REST + gRPC
- **Monitoring**: Prometheus metrics

### Port Configuration

| Service Type | Port | Protocol | Purpose |
|--------------|------|----------|---------|
| **HTTP Server** | **8085** | HTTP | REST API endpoints |
| **gRPC Server** | 8086 | gRPC | High-performance RPC calls |
| Database | 5432 | TCP | PostgreSQL connection |
| Hot Cache | 6379 | TCP | Redis DB 5 |
| Warm Cache | 6379 | TCP | Redis DB 6 |

## Build Status

### Current Build Issues ❌

The service has compilation errors that must be resolved before testing:

```
# Duplicate function declarations in internal/services/
- NewPharmacogenomicEngine (pgx_engine.go vs simplified_constructors.go)
- NewClassInteractionEngine (class_interaction_engine.go vs simplified_constructors.go)
- NewFoodAlcoholHerbalEngine (modifier_engine.go vs simplified_constructors.go)
- NewHotCacheService (hot_cache_service.go vs simplified_constructors.go)
- NewEnhancedIntegrationService (enhanced_integration_service.go vs simplified_constructors.go)

# Missing metrics methods in internal/metrics/collector.go
- RecordClassInteractionCheck
- RecordClassInteractionsFound
- RecordTripleWhammyDetection

# Type errors
- EnhancedInteractionMatrixServiceService undefined
- Duplicate map key in class_interaction_engine.go
```

### Resolution Required

Before testing, these issues must be fixed:

1. **Remove duplicate constructors**: Delete `internal/services/simplified_constructors.go` or merge into existing files
2. **Add missing metrics methods** to `internal/metrics/collector.go`
3. **Fix type definition** for `EnhancedInteractionMatrixServiceService`
4. **Fix duplicate map keys** in `internal/services/class_interaction_engine.go`

## Database Configuration

### PostgreSQL Setup

```bash
# Environment Variables
export DATABASE_URL="postgres://kb5_user:password@localhost:5432/kb_drug_interactions?sslmode=disable"
export DATABASE_PASSWORD=""  # Optional override
export DATABASE_MAX_CONNECTIONS=25
export DATABASE_CONN_MAX_LIFETIME=5m
```

### Database Schema

The service uses GORM auto-migration. Key tables:
- `drug_interactions` - Core interaction data
- `drug_metadata` - Drug information
- `pgx_variants` - Pharmacogenomic markers
- `class_interactions` - Therapeutic class patterns
- `modifier_interactions` - Food/alcohol/herbal interactions

### Migration

```bash
# Run database migrations
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions
chmod +x migrate_database.sh
./migrate_database.sh
```

## Redis Configuration

### Cache Architecture

The service implements a **dual-tier caching system**:

1. **Hot Cache** (50k entries): Ultra-fast L1 cache for most common interactions
2. **Warm Cache** (200k entries): L2 cache for frequent but less critical lookups

### Redis Environment Variables

```bash
# Hot Cache Configuration
export REDIS_URL="redis://localhost:6379"
export REDIS_PASSWORD=""  # Optional
export REDIS_DB=5  # KB-5 uses Redis DB 5

# Cache Behavior
export CACHE_INTERACTION_RESULTS=true
export INTERACTION_CACHE_TTL=1h
export ENABLE_MATRIX_CACHING=true
```

### Cache Performance Targets

- **Hot Cache Hit Rate**: >95%
- **P95 Latency**: <80ms
- **Memory Usage**: ~200MB (hot) + ~800MB (warm)

## gRPC Service Definition

### Proto File Location

`/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions/api/kb5.proto`

### Key gRPC Methods

| RPC Method | Input | Output | Purpose |
|------------|-------|--------|---------|
| `CheckInteractions` | `InteractionCheckRequest` | `InteractionCheckResponse` | Comprehensive interaction check |
| `BatchCheckInteractions` | `BatchInteractionCheckRequest` | `BatchInteractionCheckResponse` | Parallel batch processing |
| `FastLookup` | `FastLookupRequest` | `FastLookupResponse` | Sub-millisecond pairwise lookup |
| `HealthCheck` | `HealthCheckRequest` | `HealthCheckResponse` | Service health status |
| `GetMatrixStatistics` | `MatrixStatisticsRequest` | `MatrixStatisticsResponse` | Performance metrics |

### Generate gRPC Code

```bash
# Install protoc compiler and Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code from proto
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions
chmod +x generate_proto.sh
./generate_proto.sh
```

This generates:
- `api/pb/kb5.pb.go` - Protobuf message definitions
- `api/pb/kb5_grpc.pb.go` - gRPC service stubs

## Enhanced Features

### 1. Pharmacogenomic (PGx) Engine

Analyzes genetic variants affecting drug metabolism:

**Supported Variants:**
- **CYP2D6**: Poor/intermediate/normal/ultrarapid metabolizers
- **CYP2C19**: *2, *3, *17 variants (clopidogrel efficacy)
- **SLCO1B1**: *5 variant (statin myopathy risk)
- **CYP3A5**: *3 variant (tacrolimus metabolism)

**Example Request:**
```json
{
  "transaction_id": "pgx-test-001",
  "drug_codes": ["codeine", "atorvastatin"],
  "patient_context": {
    "pgx": {
      "CYP2D6": "poor_metabolizer",
      "SLCO1B1": "*5_variant"
    },
    "age_band": "adult"
  }
}
```

### 2. Drug Class Pattern Detection

Identifies dangerous therapeutic class combinations:

**Clinical Patterns:**
- **Triple Whammy**: ACE-I + Diuretic + NSAID → Acute kidney injury
- **Bleeding Synergy**: Anticoagulant + Antiplatelet → Hemorrhage risk
- **QTc Prolongation**: Multiple QT-prolonging drugs → Torsades risk
- **Serotonin Syndrome**: SSRI + MAOI + Tramadol → Serotonin toxicity

### 3. Food/Alcohol/Herbal Modifier Engine

Detects lifestyle interactions:

**Categories:**
- **Grapefruit**: CYP3A4 inhibition (statins, calcium channel blockers)
- **Tyramine-rich foods**: MAOI + aged cheese → Hypertensive crisis
- **St. John's Wort**: CYP3A4 induction (oral contraceptives, warfarin)
- **Alcohol**: CNS depression, hepatotoxicity, metabolic interference

### 4. Hot/Warm Cache Optimization

Performance-critical caching system:

**Hot Cache (50k entries):**
- Most common drug combinations
- Critical contraindications
- <10ms lookup time

**Warm Cache (200k entries):**
- Frequent but less critical interactions
- <50ms lookup time

## HTTP REST API Testing

### Health Check

```bash
curl http://localhost:8085/health

# Expected Response
{
  "status": "healthy",
  "components": {
    "database": "healthy",
    "cache": "healthy",
    "hot_cache": "healthy",
    "warm_cache": "healthy"
  },
  "version": "2.0.0",
  "environment": "development"
}
```

### Prometheus Metrics

```bash
curl http://localhost:8085/metrics

# Expected Response (sample)
# TYPE kb5_interactions_checked_total counter
kb5_interactions_checked_total 1523
# TYPE kb5_cache_hit_rate gauge
kb5_cache_hit_rate 0.967
# TYPE kb5_p95_latency_ms gauge
kb5_p95_latency_ms 78.3
```

### Basic Interaction Check

```bash
curl -X POST http://localhost:8085/api/v1/interactions/check \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "test-001",
    "drug_codes": ["warfarin", "aspirin"],
    "dataset_version": "2024.3",
    "severity_filter": ["major", "contraindicated"]
  }'

# Expected Response
{
  "transaction_id": "test-001",
  "dataset_version": "2024.3",
  "interactions": [
    {
      "drug1_code": "warfarin",
      "drug2_code": "aspirin",
      "severity": "major",
      "mechanism": "PD",
      "clinical_effects": "Increased bleeding risk",
      "management_strategy": "Monitor INR closely",
      "evidence": "established",
      "confidence": 0.95
    }
  ],
  "summary": {
    "total_interactions": 1,
    "severity_counts": {"major": 1},
    "highest_severity": "major"
  }
}
```

### Comprehensive Check (All Engines)

```bash
curl -X POST http://localhost:8085/api/v1/interactions/comprehensive \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "comprehensive-test-001",
    "drug_codes": ["lisinopril", "hydrochlorothiazide", "ibuprofen"],
    "patient_context": {
      "pgx": {"CYP2C19": "*2"},
      "hepatic_stage": "ChildPugh_A",
      "renal_stage": "CKD_2",
      "age_band": "older_adult"
    },
    "expand_classes": true,
    "include_alternatives": true,
    "include_monitoring": true
  }'

# Expected Response includes:
# - PGx-specific warnings
# - Triple Whammy detection
# - Alternative drug suggestions
# - Monitoring recommendations
```

### Batch Check

```bash
curl -X POST http://localhost:8085/api/v1/interactions/batch-check \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {
        "transaction_id": "batch-001",
        "drug_codes": ["metformin", "lisinopril"]
      },
      {
        "transaction_id": "batch-002",
        "drug_codes": ["atorvastatin", "amlodipine"]
      }
    ],
    "options": {
      "parallel": true,
      "max_concurrency": 5
    }
  }'
```

## gRPC Testing

### Prerequisites

```bash
# Install grpcurl for gRPC testing
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Or use Homebrew (macOS)
brew install grpcurl
```

### List Available Services

```bash
grpcurl -plaintext localhost:8086 list

# Expected Output
kb5.v1.DrugInteractionService
```

### List Service Methods

```bash
grpcurl -plaintext localhost:8086 list kb5.v1.DrugInteractionService

# Expected Output
kb5.v1.DrugInteractionService.CheckInteractions
kb5.v1.DrugInteractionService.BatchCheckInteractions
kb5.v1.DrugInteractionService.FastLookup
kb5.v1.DrugInteractionService.HealthCheck
kb5.v1.DrugInteractionService.GetMatrixStatistics
```

### Health Check (gRPC)

```bash
grpcurl -plaintext localhost:8086 kb5.v1.DrugInteractionService/HealthCheck

# Expected Response
{
  "status": "healthy",
  "componentHealth": {
    "cache": "healthy",
    "database": "healthy",
    "matrix": "healthy"
  },
  "version": "1.0.0",
  "timestamp": "2025-11-20T...",
  "datasetVersion": "2024.3",
  "totalInteractions": 125340
}
```

### Check Interactions (gRPC)

```bash
grpcurl -plaintext -d '{
  "transaction_id": "grpc-test-001",
  "drug_codes": ["warfarin", "aspirin"],
  "dataset_version": "2024.3",
  "patient_context": {
    "pgx": {"CYP2C9": "*2/*3"},
    "age_band": "adult"
  },
  "expand_classes": false,
  "include_contextuals": true,
  "include_alternatives": true,
  "include_monitoring": true,
  "severity_filter": ["major", "contraindicated"]
}' localhost:8086 kb5.v1.DrugInteractionService/CheckInteractions
```

### Fast Lookup (gRPC)

```bash
grpcurl -plaintext -d '{
  "drug_a_code": "warfarin",
  "drug_b_code": "aspirin",
  "dataset_version": "2024.3"
}' localhost:8086 kb5.v1.DrugInteractionService/FastLookup

# Expected Response
{
  "interactionFound": true,
  "interaction": {...},
  "cacheHit": true,
  "lookupTimeMs": 2.3
}
```

### Matrix Statistics (gRPC)

```bash
grpcurl -plaintext localhost:8086 kb5.v1.DrugInteractionService/GetMatrixStatistics

# Expected Response
{
  "totalDrugs": 4523,
  "totalInteractions": 125340,
  "matrixDensity": 0.0123,
  "lastUpdated": "2025-11-20T...",
  "memoryUsageMb": 234.5,
  "currentDatasetVersion": "2024.3",
  "lookupPerformance": {
    "averageLookupTimeNs": 2340000,
    "cacheHitRate": 0.967,
    "totalLookups": 152340,
    "p95LatencyNs": 78000000,
    "p99LatencyNs": 120000000
  },
  "cacheStatistics": {
    "hotCacheHitRate": 0.982,
    "warmCacheHitRate": 0.943,
    "hotCacheEntries": 47823,
    "warmCacheEntries": 195234
  }
}
```

## Building and Running

### Step 1: Fix Compilation Errors

**Before building, resolve these issues:**

```bash
# Option 1: Remove simplified_constructors.go (if redundant)
rm internal/services/simplified_constructors.go

# Option 2: Add missing metrics methods to collector.go
# Edit: internal/metrics/collector.go
# Add: RecordClassInteractionCheck, RecordClassInteractionsFound, RecordTripleWhammyDetection

# Fix duplicate map key in class_interaction_engine.go
# Edit: internal/services/class_interaction_engine.go (line 227)
```

### Step 2: Build Service

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions

# Install dependencies
go mod tidy

# Build binary
go build -o bin/kb5-server .

# Verify build
./bin/kb5-server --version  # (if version flag supported)
```

### Step 3: Start Service

```bash
# Set environment variables
export PORT=8085
export DATABASE_URL="postgres://kb5_user:password@localhost:5432/kb_drug_interactions?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export REDIS_DB=5
export ENVIRONMENT=development
export LOG_LEVEL=info

# Run service
./bin/kb5-server

# Or run directly with Go
go run main.go
```

### Step 4: Verify Service Started

```bash
# Check logs for startup messages
# Expected output:
# ========================================
# KB-5 Enhanced Drug Interactions Service
# ========================================
# Service: kb-5-drug-interactions
# HTTP Port: 8085
# Version: 2.0.0 (Enhanced)
# Environment: development
# ========================================
```

## Performance Testing

### Load Testing with Apache Bench

```bash
# Install Apache Bench (if not installed)
# macOS: Already installed
# Linux: apt-get install apache2-utils

# Simple health check load test
ab -n 1000 -c 10 http://localhost:8085/health

# Interaction check load test (requires request file)
ab -n 100 -c 5 -p interaction_request.json -T application/json \
   http://localhost:8085/api/v1/interactions/check
```

### Cache Performance Testing

```bash
# Test hot cache performance
curl -X POST http://localhost:8085/api/v1/admin/cache/stats

# Expected Response
{
  "hot_cache": {
    "entries": 48234,
    "hit_rate": 0.982,
    "memory_mb": 187.3
  },
  "warm_cache": {
    "entries": 193425,
    "hit_rate": 0.943,
    "memory_mb": 823.7
  }
}
```

## Admin Endpoints

### Engine Health Check

```bash
curl http://localhost:8085/api/v1/admin/engines/health

# Expected Response
{
  "pgx_engine": "healthy",
  "class_engine": "healthy",
  "modifier_engine": "healthy",
  "matrix_service": "healthy",
  "hot_cache_service": "healthy"
}
```

### Performance Metrics

```bash
curl http://localhost:8085/api/v1/admin/performance

# Expected Response
{
  "p50_latency_ms": 23.4,
  "p95_latency_ms": 78.2,
  "p99_latency_ms": 142.7,
  "requests_per_second": 1234,
  "cache_hit_rate": 0.967
}
```

### Dataset Management

```bash
curl -X POST http://localhost:8085/api/v1/admin/dataset/update \
  -H "Content-Type: application/json" \
  -d '{
    "dataset_version": "2024.4",
    "update_strategy": "rolling"
  }'
```

## Clinical Test Scenarios

### Scenario 1: Triple Whammy Detection

```bash
curl -X POST http://localhost:8085/api/v1/interactions/comprehensive \
  -H "Content-Type: application/json" \
  -d '{
    "drug_codes": ["lisinopril", "hydrochlorothiazide", "ibuprofen"],
    "patient_context": {
      "renal_stage": "CKD_3a"
    }
  }'

# Expected: CRITICAL alert for Triple Whammy pattern
```

### Scenario 2: Pharmacogenomic Interaction

```bash
curl -X POST http://localhost:8085/api/v1/interactions/comprehensive \
  -H "Content-Type: application/json" \
  -d '{
    "drug_codes": ["clopidogrel"],
    "patient_context": {
      "pgx": {"CYP2C19": "*2/*2"}
    }
  }'

# Expected: Reduced clopidogrel efficacy warning
```

### Scenario 3: Food-Drug Interaction

```bash
curl -X POST http://localhost:8085/api/v1/interactions/comprehensive \
  -H "Content-Type: application/json" \
  -d '{
    "drug_codes": ["atorvastatin"],
    "clinical_context": {
      "diet": "grapefruit_consumption"
    }
  }'

# Expected: Grapefruit-statin interaction warning
```

## Integration Testing

### Test Data Seeding

```bash
# Run database seeding script (if available)
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions/tests
go run seed_test_data.go
```

### Integration Tests

```bash
# Run all integration tests
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions
go test -tags=integration ./tests/integration/...

# Run specific test suite
go test -v -tags=integration ./tests/integration/api_test.go
go test -v -tags=integration ./tests/integration/grpc_test.go
```

## Troubleshooting

### Common Issues

#### 1. Service Won't Build
**Error**: Compilation errors
**Solution**: Fix duplicate function declarations and missing metrics methods (see Build Status section)

#### 2. Database Connection Failed
**Error**: `failed to connect to database`
**Solution**:
```bash
# Check PostgreSQL is running
pg_isready -h localhost -p 5432

# Verify credentials
psql -U kb5_user -h localhost -p 5432 -d kb_drug_interactions
```

#### 3. Redis Connection Failed
**Error**: `failed to connect to Redis`
**Solution**:
```bash
# Check Redis is running
redis-cli ping
# Expected: PONG

# Check Redis DB 5 is accessible
redis-cli -n 5 ping
```

#### 4. gRPC Not Available
**Error**: `Note: gRPC endpoints available after protoc compilation`
**Solution**:
```bash
# Generate gRPC code
./generate_proto.sh

# Rebuild service
go build -o bin/kb5-server .
```

#### 5. Port Already in Use
**Error**: `bind: address already in use`
**Solution**:
```bash
# Find process using port 8085
lsof -i :8085

# Kill process
kill -9 <PID>

# Or use different port
export PORT=8090
```

## Performance Benchmarks

### Expected Performance Metrics

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| P50 Latency | <25ms | Prometheus metrics |
| P95 Latency | <80ms | Prometheus metrics |
| P99 Latency | <150ms | Prometheus metrics |
| Cache Hit Rate | >95% | Admin endpoint |
| Throughput | >1000 RPS | Load testing |
| Memory Usage | <1.2GB | OS metrics |

### Monitoring

```bash
# View real-time metrics
watch -n 1 "curl -s http://localhost:8085/metrics | grep kb5"

# Check memory usage
ps aux | grep kb5-server

# Check goroutine count
curl http://localhost:8085/debug/pprof/goroutine
```

## Next Steps

1. **Fix compilation errors** before attempting to run the service
2. **Setup database and Redis** with proper credentials
3. **Generate gRPC code** if gRPC testing is required
4. **Build and start service** following the steps above
5. **Run health checks** to verify all components are operational
6. **Test basic REST endpoints** before moving to advanced features
7. **Test gRPC endpoints** if high-performance RPC is needed
8. **Run integration tests** to validate end-to-end functionality
9. **Perform load testing** to verify performance targets

## Additional Resources

- **Main README**: `/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions/README.md`
- **Enhancement Documentation**: `KB5_ENHANCEMENT_IMPLEMENTATION.md`
- **Compliance Checklist**: `COMPLIANCE_CHECKLIST.md`
- **Proto Definition**: `api/kb5.proto`
- **Demo Scripts**: `minimal_enhanced_demo.go`, `demo_enhanced_features.go`

## Contact & Support

For issues or questions:
1. Check the main Knowledge Base Services documentation
2. Review the CLAUDE.md file in the knowledge-base-services directory
3. Examine the service logs for detailed error messages
4. Use the admin health endpoints to diagnose component failures
