# Clinical Data Hub Rust Service - Startup Guide

## Overview

The Clinical Data Hub is a high-performance Rust service that provides clinical data intelligence and caching capabilities for the CardioFit platform. It runs on port 8118 and integrates with Apollo Federation for GraphQL schema composition.

## Quick Start

### Prerequisites
- Rust 1.70+ installed
- Port 8118 available

### Start the Service

```bash
# Using default port 8118
cargo run --release

# Or with custom port
HTTP_PORT=8118 cargo run --release

# Or run the compiled binary directly
HTTP_PORT=8118 ./target/release/clinical-data-hub-rust
```

### Verify Service Health

```bash
# Health check
curl http://localhost:8118/health

# Readiness probe
curl http://localhost:8118/ready

# Prometheus metrics
curl http://localhost:8118/metrics

# Apollo Federation schema
curl http://localhost:8118/api/federation
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `8118` | HTTP server port |
| `ENVIRONMENT` | `development` | Environment (development/staging/production) |
| `LOG_LEVEL` | `info` | Log level (trace/debug/info/warn/error) |

### Command Line Arguments

```bash
clinical-data-hub-rust --help

# Example with custom configuration
clinical-data-hub-rust --http-port 8118 --environment production --log-level warn
```

## Service Endpoints

### Health & Monitoring

- **Health Check**: `GET /health`
  - Returns service health status
  - Response: `{"status": "healthy", "service": "clinical-data-hub-rust", ...}`

- **Readiness Probe**: `GET /ready`
  - Returns readiness status with system checks
  - Response: `{"ready": true, "checks": {"http_server": "ok", ...}}`

- **Metrics**: `GET /metrics`
  - Prometheus-compatible metrics endpoint
  - Format: Plain text Prometheus exposition format

### Apollo Federation

- **Federation Schema**: `GET /api/federation`
  - GraphQL Federation schema definition (SDL)
  - Defines `ClinicalData` entity with `@key(fields: "patientId")`

- **GraphQL Endpoint**: `POST /api/federation`
  - Full GraphQL endpoint for Apollo Federation
  - Supports introspection queries: `{ _service { sdl } }`
  - Supports entity resolution: `{ _entities(representations: [...]) { ... } }`
  - Content-Type: `application/json`

## Apollo Federation Integration

The service provides a GraphQL federation schema that defines clinical data entities:

```graphql
type ClinicalData @key(fields: "patientId") {
    patientId: ID!
    aggregatedData: JSON
    cacheLayer: String
    lastUpdated: DateTime
}
```

### Adding to Federation Gateway

To integrate with Apollo Federation, add this service to your supergraph configuration:

```yaml
# supergraph.yaml
federation_version: 2
subgraphs:
  clinical-data-hub:
    routing_url: http://localhost:8118/api/federation
    schema:
      subgraph_url: http://localhost:8118/api/federation
```

## Development

### Building from Source

```bash
# Development build
cargo build

# Release build (optimized)
cargo build --release

# Run with live reload during development
cargo watch -x run
```

### Project Structure

```
src/
├── main.rs              # Main service entry point
├── main_complex.rs      # Full implementation (future)
├── models/             # Data models and structures
├── cache/              # Multi-tier caching system
└── services/           # Business logic services
```

## Production Deployment

### Docker (Future Enhancement)

```dockerfile
# Dockerfile example
FROM rust:1.70-alpine as builder
WORKDIR /app
COPY . .
RUN cargo build --release

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/target/release/clinical-data-hub-rust /usr/local/bin/
EXPOSE 8118
CMD ["clinical-data-hub-rust"]
```

### Systemd Service

```ini
# /etc/systemd/system/clinical-data-hub.service
[Unit]
Description=Clinical Data Hub Rust Service
After=network.target

[Service]
Type=simple
User=clinical-hub
WorkingDirectory=/opt/clinical-data-hub
ExecStart=/opt/clinical-data-hub/clinical-data-hub-rust
Environment=HTTP_PORT=8118
Environment=ENVIRONMENT=production
Environment=LOG_LEVEL=info
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Monitoring

The service provides comprehensive monitoring through:

- **Health endpoints** for load balancer health checks
- **Prometheus metrics** for operational monitoring
- **Structured logging** with configurable log levels
- **Graceful shutdown** handling SIGINT/SIGTERM

### Metrics Available

```
clinical_data_hub_uptime_seconds     # Service uptime
clinical_data_hub_requests_total     # Total HTTP requests
clinical_data_hub_memory_usage_bytes # Memory usage
```

## Troubleshooting

### Common Issues

**Port already in use:**
```bash
# Check what's using port 8118
lsof -i :8118

# Kill process if needed
kill $(lsof -t -i:8118)
```

**Build errors:**
```bash
# Clean and rebuild
cargo clean
cargo build --release
```

**Service not responding:**
```bash
# Check service logs
journalctl -u clinical-data-hub -f

# Or if running manually, check console output
```

### Logs

The service uses structured logging with the following format:

- **Development**: Pretty-printed colorized logs
- **Production**: JSON structured logs for log aggregation

Log levels: `TRACE` > `DEBUG` > `INFO` > `WARN` > `ERROR`

## Integration Examples

### Curl Examples

```bash
# Health monitoring
curl -f http://localhost:8118/health || echo "Service unhealthy"

# Metrics scraping (Prometheus)
curl http://localhost:8118/metrics

# Federation schema introspection (GET)
curl -s http://localhost:8118/api/federation | jq '.data._service.sdl'

# GraphQL federation queries (POST)
curl -X POST http://localhost:8118/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "{ _service { sdl } }"}' | jq .

# Entity resolution query
curl -X POST http://localhost:8118/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "{ _entities(representations: [{__typename: \"ClinicalData\", patientId: \"123\"}]) { ... on ClinicalData { patientId aggregatedData } } }"}' | jq .
```

### JavaScript/Node.js Integration

```javascript
// Health check
async function checkHealth() {
    const response = await fetch('http://localhost:8118/health');
    const health = await response.json();
    return health.status === 'healthy';
}

// Federation schema (GET)
async function getFederationSchema() {
    const response = await fetch('http://localhost:8118/api/federation');
    const result = await response.json();
    return result.data._service.sdl;
}

// GraphQL federation query (POST)
async function queryFederation(query, variables = {}) {
    const response = await fetch('http://localhost:8118/api/federation', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            query,
            variables
        })
    });
    return await response.json();
}

// Get federation schema via GraphQL
async function getFederationSchemaQL() {
    return await queryFederation('{ _service { sdl } }');
}

// Resolve entities
async function resolveEntities(representations) {
    const query = `
        query($representations: [_Any!]!) {
            _entities(representations: $representations) {
                ... on ClinicalData {
                    patientId
                    aggregatedData
                    cacheLayer
                    lastUpdated
                }
            }
        }
    `;
    return await queryFederation(query, { representations });
}
```

## Next Steps

1. **Full Implementation**: The current service is a minimal HTTP-only version. The complete implementation with multi-tier caching, gRPC support, and advanced features is available in `main_complex.rs`.

2. **Database Integration**: Future versions will include PostgreSQL and Redis integration for persistent caching.

3. **Authentication**: Integration with the platform's JWT authentication system.

4. **Advanced Caching**: Multi-layer caching with L1 (memory), L2 (Redis), and L3 (PostgreSQL) tiers.

## Support

For issues or questions:
- Check the health endpoints first
- Review logs for error details
- Ensure all prerequisites are met
- Verify port availability and permissions

The service is designed for high availability with automatic health checks and graceful shutdown handling.