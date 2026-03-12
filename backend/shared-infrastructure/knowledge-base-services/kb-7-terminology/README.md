# KB-7 Terminology Service

A comprehensive clinical terminology management microservice providing standardized code lookups, validation, and cross-system mapping for healthcare terminology systems including SNOMED CT, ICD-10, RxNorm, and LOINC.

## 🏥 Clinical Value

The KB-7 Terminology Service provides critical healthcare data standardization capabilities:
- **Code Validation**: Ensures clinical codes are current and valid across all supported terminologies
- **Cross-System Mapping**: Enables seamless translation between different medical coding systems
- **Hierarchical Navigation**: Provides parent/child relationship traversal for clinical concepts
- **Full-Text Search**: Supports fuzzy matching and multi-terminology search capabilities
- **FHIR Compliance**: Implements FHIR R4 terminology service standards

## 🚀 Quick Start

### Prerequisites
- Go 1.21 or later
- PostgreSQL 13+ 
- Redis 6+
- Docker (optional)

### Installation

1. **Clone and navigate to service directory**
   ```bash
   cd backend/services/medication-service/knowledge-bases/kb-7-terminology
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your database and Redis URLs
   ```

4. **Initialize database**
   ```bash
   # Run database migrations
   go run ./cmd/server/main.go --migrate
   ```

5. **Load terminology data**
   ```bash
   # Load all terminologies (SNOMED CT, RxNorm, LOINC)
   ./load-data.sh
   
   # Or on Windows
   load-data.bat
   
   # Load specific terminology only
   ./load-data.sh --systems snomed
   
   # Validate data files without loading
   ./load-data.sh --validate-only
   ```

6. **Start the service**
   ```bash
   go run ./cmd/server/main.go
   ```

The service will start on port 8087 (configurable via PORT environment variable).

### Docker Deployment

```bash
# Build image
docker build -t kb-7-terminology .

# Run with environment variables
docker run -p 8087:8087 \
  -e DATABASE_URL="postgresql://user:password@host:5432/db" \
  -e REDIS_URL="redis://localhost:6379/7" \
  kb-7-terminology
```

## 📋 API Documentation

### Health Check
```http
GET /health
```

### Individual Code Lookup
```http
GET /v1/concepts/{system}/{code}
```
**Example:**
```bash
curl "http://localhost:8087/v1/concepts/snomed/387517004"
```

### Full-Text Search
```http
GET /v1/concepts?q={search_term}&system={system}&count={limit}
```
**Example:**
```bash
curl "http://localhost:8087/v1/concepts?q=paracetamol&system=snomed&count=10"
```

### Code Validation
```http
POST /v1/concepts/validate
Content-Type: application/json

{
  "code": "387517004",
  "system": "http://snomed.info/sct"
}
```

### Cross-System Mapping
```http
GET /v1/mappings/{source_system}/{source_code}/{target_system}
```

### Batch Operations
```http
POST /v1/concepts/batch-lookup
Content-Type: application/json

{
  "requests": [
    {"system": "snomed", "code": "387517004"},
    {"system": "icd10", "code": "Z51.11"}
  ]
}
```

## 🗄️ Supported Terminologies

| System | URI | Version | Status |
|--------|-----|---------|---------|
| **SNOMED CT** | `http://snomed.info/sct` | 20220131 | ✅ Active |
| **ICD-10-CM** | `http://hl7.org/fhir/sid/icd-10-cm` | 2022 | ✅ Active |
| **RxNorm** | `http://www.nlm.nih.gov/research/umls/rxnorm` | 2022-01 | ✅ Active |
| **LOINC** | `http://loinc.org` | 2.72 | ✅ Active |

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8087 | HTTP server port |
| `DATABASE_URL` | `postgresql://...` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6380/7` | Redis connection string |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `ENVIRONMENT` | `development` | Environment (development, production) |
| `METRICS_ENABLED` | `true` | Enable Prometheus metrics |

### Performance Tuning

- **Database Connection Pool**: 25 max connections, 5 idle
- **Redis Cache TTL**: 1 hour for lookups, 30 minutes for search results
- **Request Timeout**: 30 seconds for server operations
- **Rate Limiting**: Configurable per endpoint

## 🔧 ETL Data Pipeline

### Loading Terminology Data

The service includes comprehensive terminology datasets and automated loading scripts:

#### Quick Load (Recommended)
```bash
# Load all terminologies with optimal settings
./load-data.sh

# Windows
load-data.bat
```

#### Individual System Loading
```bash
# Load SNOMED CT only
./load-data.sh --systems snomed

# Load RxNorm with debug logging
./load-data.sh --systems rxnorm --debug

# Load LOINC with custom batch size
./load-data.sh --systems loinc --batch-size 5000

# Validate all data without loading
./load-data.sh --validate-only
```

#### Manual ETL Commands
```bash
# SNOMED CT RF2 Files
go run ./cmd/etl/main.go --data=./data/snomed/snapshot --system=snomed --debug

# RxNorm RRF Files  
go run ./cmd/etl/main.go --data=./data/rxnorm/rrf --system=rxnorm

# LOINC Files
go run ./cmd/etl/main.go --data=./data/loinc/snapshot --system=loinc
```

### Data Sources

**✅ Included Datasets** (Ready to use):
- **SNOMED CT**: International RF2 Production 20250701 - Complete terminology with 400K+ concepts
- **RxNorm**: Full Release 07/07/2025 - Drug terminology with relationships and mappings  
- **LOINC**: LO1010000_20250321 - Laboratory codes and reference sets

**📁 Data Location**: `./data/`
- `snomed/` - SNOMED CT RF2 files (Snapshot and Full)
- `rxnorm/` - RxNorm RRF files (Full and Prescribe datasets)
- `loinc/` - LOINC RF2 format files (Snapshot and RefSets)

## 📊 Monitoring & Metrics

### Prometheus Metrics
```http
GET /metrics
```

**Key Metrics:**
- `kb7_terminology_requests_total` - Total HTTP requests
- `kb7_terminology_request_duration_seconds` - Request latency histogram
- `kb7_terminology_cache_hits_total` - Cache hit/miss counters
- `kb7_terminology_db_connections` - Database connection pool status
- `kb7_terminology_concepts_total` - Total concepts per terminology

### Health Monitoring
```json
{
  "status": "healthy",
  "service": "kb-7-terminology",
  "version": "1.0.0",
  "checks": {
    "database": {"status": "healthy", "response_time_ms": 2.1},
    "cache": {"status": "healthy", "response_time_ms": 0.8}
  }
}
```

## 🔐 Security

### Authentication
- API Key authentication via `X-API-Key` header
- JWT token support for service-to-service communication
- Rate limiting per API key

### Authorization
- Role-based access control (RBAC)
- Audit logging for all operations
- IP-based restrictions (configurable)

### Data Protection
- All patient-identifiable information is excluded
- Terminology data is public domain or properly licensed
- HIPAA compliance for deployment environment

## 📈 Performance

### Response Time Targets
- **P50**: < 3ms for cached lookups
- **P95**: < 15ms for database queries  
- **P99**: < 50ms for complex searches

### Throughput
- **Sustained**: 25,000 requests/second
- **Peak**: 50,000 requests/second
- **Concurrent Connections**: 2,000

### Caching Strategy
- **L1**: In-memory Ristretto cache (100MB)
- **L2**: Redis distributed cache
- **TTL**: 1 hour for concepts, 4 hours for mappings

## 🐛 Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Verify PostgreSQL is running and accessible
   - Check DATABASE_URL format and credentials
   - Ensure database migrations have been run

2. **Cache Connection Failures**
   - Verify Redis is running on specified URL
   - Check Redis database number (default: 7)
   - Monitor Redis memory usage

3. **ETL Loading Issues**
   - Verify data file formats match expected schemas
   - Check file permissions and paths
   - Monitor database disk space during imports

4. **Performance Issues**
   - Enable debug logging to identify bottlenecks
   - Monitor database query performance
   - Check cache hit rates via metrics endpoint

### Debug Mode
```bash
LOG_LEVEL=debug go run ./cmd/server/main.go
```

### Database Query Analysis
```sql
-- Check terminology system counts
SELECT system_name, concept_count FROM terminology_systems;

-- Analyze search performance
EXPLAIN ANALYZE SELECT * FROM terminology_concepts 
WHERE search_terms @@ plainto_tsquery('paracetamol');
```

## 🏗️ Development

### Project Structure
```
kb-7-terminology/
├── cmd/
│   ├── etl/           # ETL command-line tool
│   └── server/        # HTTP server main
├── internal/
│   ├── api/           # HTTP handlers and routing
│   ├── cache/         # Multi-layer caching
│   ├── config/        # Configuration management
│   ├── database/      # Database connections
│   ├── etl/           # ETL processing logic
│   ├── metrics/       # Prometheus metrics
│   ├── models/        # Data models
│   ├── security/      # Authentication & authorization
│   └── services/      # Business logic
├── migrations/        # Database schema migrations
├── api/              # OpenAPI specification
├── schemas/          # JSON Schema definitions
└── framework.yaml    # Service framework configuration
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Contributing
1. Fork the repository
2. Create a feature branch
3. Add comprehensive tests
4. Update documentation
5. Submit a pull request

## 🔗 Integration

### GraphQL Federation
The service supports Apollo Federation for schema composition:
```graphql
type Concept @key(fields: "code system") {
  code: String!
  system: String!
  display: String!
  definition: String
}
```

### FHIR Integration
Compatible with FHIR R4 terminology operations:
- `$lookup` - Individual concept lookup
- `$validate-code` - Code validation
- `$expand` - Value set expansion
- `$translate` - Concept mapping

### Clinical Decision Support
Integrates with clinical decision support systems for:
- Drug interaction checking
- Clinical guideline enforcement  
- Quality measure calculation
- Risk stratification algorithms

## 📄 License

Proprietary software. All rights reserved.

## 📞 Support

- **Clinical Platform Team**: clinical-platform@hospital.com
- **Documentation**: https://internal.hospital.com/docs/kb7
- **Issue Tracker**: https://github.com/hospital/kb-7-terminology/issues