# Context Gateway Go Service

A high-performance Go implementation of the Clinical Context Gateway, providing recipe-based clinical snapshot management with dual-layer storage architecture.

## 🏗️ Architecture

### Core Components

- **Context Gateway Service**: Main gRPC service implementing clinical snapshot and recipe management
- **Dual-Layer Storage**: Hot storage (Redis) + Cold storage (MongoDB) for optimal performance
- **Recipe Management**: Clinical workflow recipe system with governance controls
- **Data Source Registry**: Federated clinical data source management
- **Audit Logging**: High-priority clinical audit trail
- **Metrics Collection**: Performance and operational metrics

### Design Patterns

- **Dual-Layer Storage**: L1 (Redis) cache + L2 (MongoDB) persistence
- **Recipe-Based Assembly**: Governance-approved clinical context recipes
- **Circuit Breaker**: Fault tolerance for external service calls
- **Atomic Transactions**: Ensure data consistency across storage layers
- **Live Fetch Governance**: Controlled real-time data fetching with permissions

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or later
- Redis 6.0+ (for hot storage)
- MongoDB 4.4+ (for cold storage)
- Protocol Buffers compiler (`protoc`)

### Installation

1. **Clone and enter directory**:
   ```bash
   cd backend/services/context-gateway-go
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Generate Protocol Buffer code**:
   ```bash
   chmod +x scripts/generate_proto.sh
   ./scripts/generate_proto.sh
   ```

4. **Build the service**:
   ```bash
   chmod +x scripts/build.sh
   ./scripts/build.sh
   ```

### Running the Service

**Using default configuration**:
```bash
./build/context-gateway
```

**With custom configuration**:
```bash
./build/context-gateway \
  -grpc-port :8017 \
  -http-port :8117 \
  -redis-addr localhost:6379 \
  -mongo-uri mongodb://localhost:27017 \
  -db-name clinical_context_go \
  -env development
```

**Using environment variables**:
```bash
export GRPC_PORT=8017
export HTTP_PORT=8117
export REDIS_ADDR=localhost:6379
export MONGO_URI=mongodb://localhost:27017
export DB_NAME=clinical_context_go
export ENVIRONMENT=development

./build/context-gateway
```

## 📡 API Endpoints

### gRPC Service (Port 8017)

The Context Gateway implements the following gRPC methods:

- `CreateSnapshot` - Create clinical snapshots using workflow recipes
- `GetSnapshot` - Retrieve clinical snapshots with access tracking
- `ValidateSnapshot` - Validate snapshot integrity and expiration
- `InvalidateSnapshot` - Invalidate snapshots with audit trail
- `FetchLiveFields` - Live clinical data fetching with governance
- `GetServiceHealth` - Service and dependency health status
- `GetMetrics` - Performance and operational metrics

### HTTP Endpoints (Port 8117)

- `GET /health` - Health check endpoint
- `GET /ready` - Readiness check endpoint
- `GET /status` - Detailed service status with dependencies
- `GET /metrics` - Prometheus-compatible metrics
- `GET /` - Service information and capabilities

## 🔧 Configuration

### Command Line Arguments

| Argument | Default | Description |
|----------|---------|-------------|
| `-grpc-port` | `:8017` | gRPC server port |
| `-http-port` | `:8117` | HTTP server port |
| `-redis-addr` | `localhost:6379` | Redis server address |
| `-mongo-uri` | `mongodb://localhost:27017` | MongoDB connection URI |
| `-db-name` | `clinical_context_go` | Database name |
| `-env` | `development` | Environment (development/production) |

### Environment Variables

All command line arguments can be overridden with environment variables:

- `GRPC_PORT` - gRPC port (without colon)
- `HTTP_PORT` - HTTP port (without colon)  
- `REDIS_ADDR` - Redis address
- `MONGO_URI` - MongoDB URI
- `DB_NAME` - Database name
- `ENVIRONMENT` - Environment setting

## 📊 Monitoring

### Health Checks

- **Liveness**: `GET /health` - Basic service health
- **Readiness**: `GET /ready` - Service ready to accept requests
- **Detailed Status**: `GET /status` - Comprehensive health with dependencies

### Metrics

Prometheus-compatible metrics available at `/metrics`:

- **Snapshot Metrics**: Creation, access, invalidation rates
- **Recipe Metrics**: Usage patterns, performance by recipe
- **Data Source Metrics**: Call rates, error rates, latency
- **Cache Metrics**: Hit ratios, eviction rates
- **Quality Metrics**: Completeness scores, data quality issues

### Audit Trail

All clinical operations generate high-priority audit events:

- Snapshot creation/access/invalidation
- Live data fetching
- Recipe validation
- Security events
- Data quality issues

## 🧪 Testing

### Unit Tests
```bash
go test -v ./...
```

### Integration Tests
```bash
go test -v -tags=integration ./...
```

### Load Testing
```bash
# Using grpcurl for gRPC endpoint testing
grpcurl -plaintext localhost:8017 list
grpcurl -plaintext localhost:8017 context_gateway.ContextGateway/GetServiceHealth
```

## 🏭 Production Deployment

### Docker Deployment

Build Docker image:
```bash
docker build -t context-gateway-go:latest .
```

Run with Docker Compose:
```bash
version: '3.8'
services:
  context-gateway:
    image: context-gateway-go:latest
    ports:
      - "8017:8017"
      - "8117:8117"
    environment:
      - ENVIRONMENT=production
      - REDIS_ADDR=redis:6379
      - MONGO_URI=mongodb://mongo:27017
    depends_on:
      - redis
      - mongo
  
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
  
  mongo:
    image: mongo:7
    ports:
      - "27017:27017"
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: context-gateway-go
spec:
  replicas: 3
  selector:
    matchLabels:
      app: context-gateway-go
  template:
    metadata:
      labels:
        app: context-gateway-go
    spec:
      containers:
      - name: context-gateway
        image: context-gateway-go:latest
        ports:
        - containerPort: 8017
        - containerPort: 8117
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: REDIS_ADDR
          value: "redis:6379"
        - name: MONGO_URI
          value: "mongodb://mongo:27017"
```

## 🔄 Integration with Existing Services

### Service Discovery

The Context Gateway integrates with the existing microservice ecosystem:

- **Port Separation**: Uses port 8017 (gRPC) and 8117 (HTTP) to avoid conflicts with Python context-service (port 8016)
- **Data Source Integration**: Connects to existing services via the Data Source Registry
- **Recipe Compatibility**: Loads and validates recipes from YAML files compatible with existing format

### Migration Strategy

1. **Phase 1**: Deploy Context Gateway Go alongside Python service
2. **Phase 2**: Route new snapshot requests to Go service
3. **Phase 3**: Migrate existing snapshots from Python to Go service
4. **Phase 4**: Deprecate Python service after validation

## 🛠️ Development

### Project Structure

```
context-gateway-go/
├── cmd/main.go                 # Main application entry point
├── internal/
│   ├── models/                 # Data models and structures
│   │   ├── snapshot.go         # Clinical snapshot models
│   │   └── recipe.go           # Workflow recipe models
│   ├── services/               # Business logic services
│   │   ├── context_gateway.go  # Main gRPC service implementation
│   │   ├── data_source_registry.go # Data source management
│   │   ├── recipe_service.go   # Recipe management
│   │   ├── audit_logger.go     # Clinical audit logging
│   │   └── metrics_collector.go # Metrics collection
│   └── storage/                # Data persistence layer
│       └── snapshot_store.go   # Dual-layer storage implementation
├── proto/                      # Protocol Buffer definitions
│   └── context_gateway.proto   # gRPC service definitions
├── scripts/                    # Build and deployment scripts
│   ├── build.sh               # Build script
│   └── generate_proto.sh      # Protocol Buffer generation
├── go.mod                     # Go module definition
└── README.md                  # This file
```

### Adding New Features

1. **Add gRPC method**: Update `proto/context_gateway.proto`
2. **Regenerate code**: Run `./scripts/generate_proto.sh`
3. **Implement service**: Add method to `internal/services/context_gateway.go`
4. **Add tests**: Create corresponding test files
5. **Update documentation**: Update this README

## 🔒 Security

### Clinical Data Protection

- **Encryption at Rest**: MongoDB storage with encryption
- **Encryption in Transit**: gRPC with TLS/mTLS
- **Audit Trail**: All clinical operations logged with high priority
- **Access Control**: Service-level authentication and authorization
- **Data Integrity**: Cryptographic checksums and digital signatures

### Security Best Practices

- No sensitive data in logs
- Secure credential management
- Regular security updates
- Principle of least privilege
- Secure communication channels

## 📝 License

This Context Gateway Go Service is part of the Clinical Synthesis Hub CardioFit platform.

## 🤝 Contributing

1. Follow Go coding standards
2. Add unit tests for new features
3. Update documentation
4. Ensure all tests pass
5. Submit pull request with detailed description

---

**Context Gateway Go Service** - High-Performance Clinical Context Management
*Part of the Clinical Synthesis Hub CardioFit Platform*