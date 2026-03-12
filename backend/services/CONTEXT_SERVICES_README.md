# Context Services Implementation

## Overview

The Context Services implementation converts the original Python context-service into two specialized, high-performance services:

- **Context Gateway (Go)** - Recipe & snapshot management with clinical governance
- **Clinical Data Hub (Rust)** - High-performance multi-layer caching and data intelligence

This separation provides better performance, scalability, and maintainability while avoiding conflicts with the legacy Python service.

## Architecture

```
┌─────────────────┐    ┌────────────────────┐    ┌──────────────────┐
│ Apollo          │    │ Context Gateway    │    │ Clinical Data    │
│ Federation      │◄──►│ (Go)               │◄──►│ Hub (Rust)       │
│ :4000           │    │ gRPC: :8017        │    │ gRPC: :8018      │
└─────────────────┘    │ HTTP: :8117        │    │ HTTP: :8118      │
                       └────────────────────┘    └──────────────────┘
                                │                          │
                       ┌────────▼────────┐    ┌────────────▼─────────┐
                       │ Recipe Engine   │    │ Multi-Layer Cache    │
                       │ Snapshot Mgmt   │    │ L1: Memory <1ms      │
                       │ Clinical Safety │    │ L2: Redis 1-5ms      │
                       │ Governance      │    │ L3: Persistent 5-50ms│
                       └─────────────────┘    └──────────────────────┘
```

### Service Specialization

#### Context Gateway (Go) - Port 8017/8117
- **Primary Role**: Recipe execution and clinical snapshot management
- **Key Features**:
  - Clinical recipe processing with governance controls
  - Snapshot creation, validation, and lifecycle management
  - Clinical safety checks and compliance validation
  - Integration with clinical decision support systems
  - Real-time context assembly for patient workflows

#### Clinical Data Hub (Rust) - Port 8018/8118  
- **Primary Role**: High-performance caching and data intelligence
- **Key Features**:
  - Multi-layer caching (Memory → Redis → Persistent)
  - Sub-millisecond data retrieval for hot paths
  - Intelligent cache warming and eviction policies
  - Data compression and storage optimization
  - Performance analytics and cache hit optimization

## Quick Start

### Prerequisites

- **Go 1.21+** for Context Gateway
- **Rust 1.70+** with Cargo for Clinical Data Hub
- **Redis** for caching layer
- **PostgreSQL** for persistent storage (optional)
- **Node.js** for Apollo Federation and testing scripts

### 1. Start Infrastructure Services

```bash
# Start Redis (required)
redis-server

# Start PostgreSQL (optional, for persistent storage)
# Follow your system's PostgreSQL installation guide
```

### 2. Start Context Services

```bash
# Option A: Use the automated startup script (recommended)
cd backend/services/scripts
node start-context-services.js

# Option B: Start services manually
cd backend/services/context-gateway-go
go run cmd/main.go

# In another terminal:
cd backend/services/clinical-data-hub-rust
cargo run
```

### 3. Start Apollo Federation

```bash
cd apollo-federation
npm install
npm start
```

### 4. Validate Integration

```bash
# Run comprehensive validation
cd backend/services/scripts
node validate-context-services.js

# Test federation integration
node test-federation-integration.js
```

## Service URLs

### Context Gateway (Go)
- **gRPC**: `localhost:8017` - High-performance binary protocol
- **HTTP**: `http://localhost:8117` - REST API and GraphQL federation
- **Health**: `http://localhost:8117/health` - Service health check
- **Federation**: `http://localhost:8117/api/federation` - Apollo Federation endpoint
- **Metrics**: `http://localhost:8117/metrics` - Prometheus metrics

### Clinical Data Hub (Rust)
- **gRPC**: `localhost:8018` - High-performance caching operations
- **HTTP**: `http://localhost:8118` - REST API and GraphQL federation
- **Health**: `http://localhost:8118/health` - Service health check
- **Federation**: `http://localhost:8118/api/federation` - Apollo Federation endpoint
- **Metrics**: `http://localhost:8118/metrics` - Prometheus metrics

### Apollo Federation
- **GraphQL**: `http://localhost:4000/graphql` - Unified GraphQL gateway
- **Health**: `http://localhost:4000/health` - Gateway health check

## Service Interactions

### Clinical Snapshot Creation Flow

```
1. Client → Apollo Federation: Create snapshot request
2. Apollo → Context Gateway: Recipe processing request
3. Context Gateway → Clinical Data Hub: Cache lookup for patient data
4. Clinical Data Hub → Context Gateway: Cached/fresh data response
5. Context Gateway: Apply clinical rules and safety checks
6. Context Gateway → Clinical Data Hub: Store processed snapshot
7. Context Gateway → Apollo Federation: Return validated snapshot
8. Apollo Federation → Client: Complete clinical context
```

### Data Flow Patterns

#### Hot Path (< 1ms)
```
Client → Apollo → Context Gateway → Clinical Data Hub L1 (Memory) → Response
```

#### Warm Path (1-5ms)
```
Client → Apollo → Context Gateway → Clinical Data Hub L2 (Redis) → Response
```

#### Cold Path (5-50ms)
```
Client → Apollo → Context Gateway → Clinical Data Hub L3 (Database) → Response
```

### Inter-Service Communication

- **gRPC**: Used for high-performance binary communication between services
- **HTTP/GraphQL**: Used for Apollo Federation integration and external APIs
- **Message Queues**: Used for asynchronous processing (future enhancement)

## GraphQL Schema Integration

### Context Gateway Types

```graphql
type ClinicalSnapshot {
  id: ID!
  recipeId: String!
  patientId: String!
  createdAt: DateTime!
  status: SnapshotStatus!
  data: JSON!
  metadata: SnapshotMetadata!
}

type SnapshotMetadata {
  version: String!
  checksum: String!
  performance: PerformanceMetrics!
  governance: GovernanceInfo!
}

type Recipe {
  id: ID!
  name: String!
  version: String!
  fields: [RecipeField!]!
  rules: [ClinicalRule!]!
}
```

### Clinical Data Hub Types

```graphql
type CachedData {
  key: String!
  layer: CacheLayer!
  value: JSON!
  ttl: Int!
  metadata: CacheMetadata!
}

type CacheMetadata {
  accessCount: Int!
  lastAccessed: DateTime!
  compressionRatio: Float!
  hitRate: Float!
}

enum CacheLayer {
  MEMORY
  REDIS
  PERSISTENT
}
```

## Development

### Building Services

#### Context Gateway (Go)
```bash
cd backend/services/context-gateway-go

# Install dependencies
go mod tidy

# Generate proto files (if needed)
protoc --go_out=. --go-grpc_out=. proto/*.proto

# Build
go build cmd/main.go

# Run tests
go test ./...

# Run with debugging
go run cmd/main.go --debug
```

#### Clinical Data Hub (Rust)
```bash
cd backend/services/clinical-data-hub-rust

# Build (development)
cargo build

# Build (optimized)
cargo build --release

# Run tests
cargo test

# Run with debugging
RUST_LOG=debug cargo run

# Check code quality
cargo clippy
```

### Configuration

#### Context Gateway Configuration

Environment variables:
```bash
# gRPC Configuration
GRPC_PORT=8017
HTTP_PORT=8117

# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=context_gateway
POSTGRES_USER=context_user
POSTGRES_PASSWORD=context_pass

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# Clinical Configuration
SAFETY_CHECK_ENABLED=true
GOVERNANCE_MODE=strict
RECIPE_CACHE_TTL=300
```

#### Clinical Data Hub Configuration

Environment variables:
```bash
# Service Configuration
GRPC_PORT=8018
HTTP_PORT=8118

# Cache Configuration
MEMORY_CACHE_SIZE=1GB
REDIS_HOST=localhost
REDIS_PORT=6379

# Performance Tuning
CACHE_WARMING_ENABLED=true
COMPRESSION_ENABLED=true
METRICS_ENABLED=true
```

## Monitoring and Observability

### Prometheus Metrics

Both services expose metrics at `/metrics`:

#### Context Gateway Metrics
- `context_gateway_snapshots_created_total`
- `context_gateway_recipes_executed_total`
- `context_gateway_safety_checks_total`
- `context_gateway_governance_validations_total`
- `context_gateway_request_duration_seconds`

#### Clinical Data Hub Metrics
- `clinical_hub_cache_hits_total`
- `clinical_hub_cache_misses_total`
- `clinical_hub_cache_evictions_total`
- `clinical_hub_cache_memory_usage_bytes`
- `clinical_hub_request_duration_seconds`

### Health Checks

Health endpoints provide detailed service status:

```bash
# Context Gateway health
curl http://localhost:8117/health

# Clinical Data Hub health
curl http://localhost:8118/health
```

Response format:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "dependencies": {
    "redis": "connected",
    "postgres": "connected",
    "grpc_server": "listening"
  },
  "performance": {
    "uptime_seconds": 3600,
    "memory_usage_mb": 125,
    "cpu_usage_percent": 2.5
  }
}
```

## Testing

### Unit Tests

```bash
# Context Gateway (Go)
cd backend/services/context-gateway-go
go test ./... -v

# Clinical Data Hub (Rust)
cd backend/services/clinical-data-hub-rust
cargo test -- --nocapture
```

### Integration Tests

```bash
# Comprehensive service validation
node scripts/validate-context-services.js

# Federation integration tests
node scripts/test-federation-integration.js

# Load testing (requires artillery)
npm install -g artillery
artillery run tests/load-test.yml
```

### Test Data

Sample GraphQL queries for testing:

```graphql
# Create clinical snapshot
mutation CreateSnapshot($input: CreateSnapshotInput!) {
  createSnapshot(input: $input) {
    id
    status
    metadata {
      performance {
        executionTimeMs
      }
    }
  }
}

# Query patient context
query GetPatientContext($patientId: ID!) {
  patient(id: $patientId) {
    id
    snapshots {
      id
      recipeId
      createdAt
    }
    cachedData {
      key
      layer
      metadata {
        accessCount
        hitRate
      }
    }
  }
}
```

## Deployment

### Docker Deployment

```bash
# Build and start with Docker Compose
cd backend/services
docker-compose -f docker-compose.context-services.yml up --build

# Scale services
docker-compose -f docker-compose.context-services.yml up --scale context-gateway=2 --scale clinical-hub=2
```

### Production Configuration

#### Load Balancing
- Use Nginx or HAProxy for HTTP load balancing
- Use gRPC load balancers for gRPC traffic
- Consider service mesh (Istio) for advanced routing

#### Security
- Enable TLS for all inter-service communication
- Implement proper authentication and authorization
- Use network policies to restrict service access
- Enable audit logging for clinical data access

#### Scalability
- Context Gateway: Scale based on recipe execution load
- Clinical Data Hub: Scale based on cache hit ratio and memory usage
- Monitor cache effectiveness and adjust cache sizes accordingly

## Troubleshooting

### Common Issues

#### Service Won't Start
```bash
# Check port availability
netstat -an | grep :8017
netstat -an | grep :8018

# Check service logs
tail -f logs/context-gateway.log
tail -f logs/clinical-hub.log
```

#### Federation Errors
```bash
# Validate federation endpoints
curl -X POST http://localhost:8117/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "{ _service { sdl } }"}'

# Check Apollo Federation logs
tail -f apollo-federation/logs/gateway.log
```

#### Performance Issues
```bash
# Check cache hit rates
curl http://localhost:8118/metrics | grep cache_hits

# Monitor memory usage
curl http://localhost:8117/health
curl http://localhost:8118/health

# Profile Go service
go tool pprof http://localhost:8117/debug/pprof/profile
```

### Debug Mode

Enable debug logging:

```bash
# Context Gateway
RUST_LOG=debug ./context-gateway

# Clinical Data Hub
export DEBUG=true && go run cmd/main.go
```

## Migration from Python Context Service

### Data Migration
1. Export existing snapshots and recipes from Python service
2. Transform data format to match Go/Rust schemas
3. Import data using provided migration scripts
4. Validate data integrity and performance

### API Compatibility
- GraphQL schema maintains backward compatibility
- REST endpoints provide similar functionality
- gRPC offers enhanced performance for new clients

### Gradual Migration Strategy
1. Deploy new services alongside Python service
2. Route read traffic to new services
3. Gradually migrate write operations
4. Monitor performance and rollback if needed
5. Decommission Python service once stable

## Future Enhancements

### Planned Features
- **Event Streaming**: Real-time clinical events via Kafka
- **Machine Learning**: Predictive caching and clinical insights
- **Multi-Region**: Geographic distribution for global deployments
- **Advanced Security**: Fine-grained RBAC and data encryption
- **Mobile SDK**: Native mobile app integration

### Performance Optimizations
- **Query Optimization**: Smarter query planning and execution
- **Caching Strategies**: Intelligent cache warming and prefetching
- **Data Compression**: Advanced compression for large clinical datasets
- **Connection Pooling**: Optimized database connection management

## Support

### Documentation
- API documentation: `docs/api/`
- Architecture diagrams: `docs/architecture/`
- Deployment guides: `docs/deployment/`

### Community
- GitHub Issues: Report bugs and feature requests
- Development Slack: `#context-services` channel
- Monthly Reviews: Architecture and performance discussions

---

**Version**: 1.0.0  
**Last Updated**: January 2025  
**Maintainers**: Clinical Platform Team