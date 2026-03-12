# KB7 Terminology Query Router

Intelligent hybrid query router service for KB7 Terminology Phase 3.5.2 that provides optimal routing between PostgreSQL and GraphDB based on query intent classification.

## Overview

The KB7 Query Router implements intelligent query routing to optimize performance by directing different types of queries to the most appropriate data store:

- **PostgreSQL**: Fast exact lookups, cross-terminology mappings, text search
- **GraphDB**: Semantic reasoning, subsumption queries, drug interactions
- **Redis**: Caching layer for frequent queries

## Features

### 🎯 Query Intent Classification
- **LookupIntent**: Fast exact code lookup (→ PostgreSQL, <10ms)
- **ReasoningIntent**: Semantic reasoning/subsumption (→ GraphDB, <50ms)
- **MappingIntent**: Cross-terminology mapping (→ PostgreSQL, <15ms)
- **SearchIntent**: Fuzzy text search (→ PostgreSQL, <50ms)
- **RelationshipIntent**: Concept relationships (→ Hybrid, <25ms)

### ⚡ Performance Features
- Redis caching with >90% cache hit ratio target
- Circuit breakers for fault tolerance
- Query performance metrics and monitoring
- OpenTelemetry distributed tracing
- Prometheus metrics export

### 🛡️ Reliability
- Health checks and readiness probes
- Graceful shutdown
- Connection pooling
- Retry logic with exponential backoff

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Client Apps   │───▶│   Query Router   │───▶│  Cache (Redis)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                   ┌────────────┴────────────┐
                   ▼                         ▼
           ┌───────────────┐         ┌───────────────┐
           │  PostgreSQL   │         │    GraphDB    │
           │ (Fast Lookups)│         │ (Reasoning)   │
           └───────────────┘         └───────────────┘
```

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL with KB7 terminology data
- GraphDB with RDF/SPARQL endpoint
- Redis for caching

### Development Setup

1. **Clone and setup**:
```bash
cd query-router
make dev-setup
```

2. **Configure environment**:
```bash
cp .env.example .env
# Edit .env with your database connections
```

3. **Run locally**:
```bash
make run
```

### Docker Deployment

1. **Build image**:
```bash
make docker-build
```

2. **Run container**:
```bash
make docker-run
```

## API Endpoints

### Core Query Endpoints

#### Concept Lookup (PostgreSQL)
```bash
GET /api/v1/concepts/{system}/{code}

# Example
GET /api/v1/concepts/snomed-ct/73211009
```

#### Subconcept Query (GraphDB)
```bash
GET /api/v1/concepts/{system}/{code}/subconcepts?limit=50

# Example
GET /api/v1/concepts/snomed-ct/73211009/subconcepts?limit=20
```

#### Cross-Terminology Mapping (PostgreSQL)
```bash
GET /api/v1/mappings/{fromSystem}/{fromCode}/{toSystem}

# Example
GET /api/v1/mappings/snomed-ct/73211009/icd-10
```

#### Drug Interactions (GraphDB)
```bash
POST /api/v1/interactions
Content-Type: application/json

{
  "medication_codes": ["387207008", "387562000"]
}
```

#### Concept Relationships (Hybrid)
```bash
GET /api/v1/concepts/{system}/{code}/relationships?type=all

# Example
GET /api/v1/concepts/snomed-ct/73211009/relationships?type=parent
```

#### Text Search (PostgreSQL)
```bash
GET /api/v1/search?q=diabetes&system=snomed-ct&limit=20
```

### Monitoring Endpoints

#### Health Check
```bash
GET /health
# Returns service health status
```

#### Readiness Check
```bash
GET /ready
# Returns service readiness status
```

#### Query Metrics
```bash
GET /api/v1/metrics
# Returns performance metrics
```

#### Prometheus Metrics
```bash
GET :8088/metrics
# Prometheus format metrics
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | 8087 | Main API server port |
| `METRICS_PORT` | 8088 | Metrics server port |
| `POSTGRES_URL` | - | PostgreSQL connection string |
| `REDIS_URL` | redis://localhost:6379 | Redis connection string |
| `GRAPHDB_ENDPOINT` | http://localhost:7200 | GraphDB SPARQL endpoint |
| `DEFAULT_CACHE_TTL` | 1h | Default cache time-to-live |
| `MAX_CACHE_SIZE` | 10000 | Maximum cache entries |
| `QUERY_TIMEOUT` | 30s | Query timeout duration |
| `LOG_LEVEL` | info | Logging level |
| `JAEGER_ENDPOINT` | - | Jaeger tracing endpoint |

### Query Routing Matrix

| Query Type | Intent | Target Store | Cache TTL | Target Latency |
|------------|--------|--------------|-----------|----------------|
| Exact Lookup | Lookup | PostgreSQL | 60min | <10ms |
| Subsumption | Reasoning | GraphDB | 30min | <50ms |
| Mapping | Mapping | PostgreSQL | 120min | <15ms |
| Drug Interaction | Reasoning | GraphDB | 15min | <100ms |
| Relationships | Relationship | Hybrid | 45min | <25ms |
| Text Search | Search | PostgreSQL | 30min | <50ms |

## Development

### Available Commands

```bash
make help              # Show all available commands
make build             # Build the binary
make run               # Run locally
make test              # Run tests
make test-coverage     # Run tests with coverage
make lint              # Run linter
make fmt               # Format code
make clean             # Clean build artifacts
make health            # Check service health
make metrics           # Get service metrics
```

### Project Structure

```
query-router/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── cache/
│   │   └── redis.go         # Redis caching implementation
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── graphdb/
│   │   └── client.go        # GraphDB SPARQL client
│   ├── postgres/
│   │   └── client.go        # PostgreSQL client
│   └── router/
│       └── router.go        # Core routing logic
├── go.mod                   # Go module definition
├── Dockerfile              # Container definition
├── Makefile                # Build automation
├── .env.example            # Environment configuration
└── README.md               # This file
```

### Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Test specific package
go test ./internal/router -v
```

### Adding New Query Types

1. **Define Intent**: Add new intent to `QueryIntent` enum
2. **Update Routing**: Add routing decision to `QueryRouting` map
3. **Implement Handler**: Create handler function in router
4. **Add Route**: Register route in main.go
5. **Update Tests**: Add test cases for new functionality

## Performance Metrics

### Prometheus Metrics

- `kb7_query_duration_seconds`: Query execution time by intent/store
- `kb7_cache_hits_total`: Cache hit counter by query type
- `kb7_cache_misses_total`: Cache miss counter by query type
- `kb7_query_errors_total`: Error counter by intent/store/type

### Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Cache Hit Ratio | >90% | Hot path queries |
| Lookup Latency | <10ms | PostgreSQL exact lookups |
| Reasoning Latency | <50ms | GraphDB semantic queries |
| Availability | >99.9% | Service uptime |
| Error Rate | <0.1% | Query success rate |

## Monitoring and Observability

### Health Monitoring

```bash
# Check all services
curl http://localhost:8087/health

# Check readiness
curl http://localhost:8087/ready
```

### Metrics Collection

```bash
# Get JSON metrics
curl http://localhost:8087/api/v1/metrics

# Get Prometheus metrics
curl http://localhost:8088/metrics
```

### Distributed Tracing

Configure Jaeger endpoint to enable distributed tracing:

```bash
export JAEGER_ENDPOINT=http://jaeger:14268/api/traces
```

## Deployment

### Docker Compose

```yaml
version: '3.8'
services:
  query-router:
    build: .
    ports:
      - "8087:8087"
      - "8088:8088"
    environment:
      - POSTGRES_URL=postgres://user:pass@postgres:5432/kb7
      - REDIS_URL=redis://redis:6379
      - GRAPHDB_ENDPOINT=http://graphdb:7200
    depends_on:
      - postgres
      - redis
      - graphdb
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kb7-query-router
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kb7-query-router
  template:
    metadata:
      labels:
        app: kb7-query-router
    spec:
      containers:
      - name: query-router
        image: kb7-query-router:latest
        ports:
        - containerPort: 8087
        - containerPort: 8088
        env:
        - name: POSTGRES_URL
          valueFrom:
            secretKeyRef:
              name: kb7-secrets
              key: postgres-url
        livenessProbe:
          httpGet:
            path: /health
            port: 8087
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8087
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Troubleshooting

### Common Issues

1. **Connection Failures**:
   - Check database URLs in environment
   - Verify network connectivity
   - Check service health endpoints

2. **Cache Performance**:
   - Monitor cache hit ratios
   - Adjust TTL values if needed
   - Check Redis memory usage

3. **Query Timeouts**:
   - Increase `QUERY_TIMEOUT` if needed
   - Check database performance
   - Review query complexity

### Debugging

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Check service logs
docker logs kb7-query-router

# Monitor metrics
watch -n 5 'curl -s http://localhost:8087/api/v1/metrics | jq .'
```

## Contributing

1. Follow Go best practices and project conventions
2. Add tests for new functionality
3. Update documentation for API changes
4. Run linter and formatter before submitting
5. Ensure all tests pass

## License

Copyright (c) 2024 CardioFit Platform - All Rights Reserved

---

**Version**: 1.0.0  
**Last Updated**: 2024-09-22  
**Maintainer**: CardioFit Platform Team