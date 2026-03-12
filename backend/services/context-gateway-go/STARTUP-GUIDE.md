# Context Gateway Service - Startup Guide

## Overview
The Context Gateway Service is a sophisticated Go-based service that provides clinical snapshot and recipe management with dual-layer storage architecture. It implements both gRPC and HTTP APIs for comprehensive clinical context assembly and federation capabilities.

## Prerequisites

### Required Software
- **Go 1.23+** - Programming language runtime
- **Protocol Buffers Compiler** - For gRPC code generation
- **Docker & Docker Compose** - For database dependencies

### System Dependencies
- **MongoDB** - Document storage (via Docker)
- **Redis** - Caching layer (via Docker)
- **Network Ports**: 8017 (gRPC), 8117 (HTTP), 27017 (MongoDB), 6379 (Redis)

### Protocol Buffers Setup
```bash
# Install protoc compiler
# macOS:
brew install protobuf

# Linux:
sudo apt-get install protobuf-compiler

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Quick Start

### 1. Navigate to Service Directory
```bash
cd backend/services/context-gateway-go
```

### 2. Start Infrastructure Dependencies
```bash
# Start MongoDB and Redis using Docker Compose
docker-compose up -d

# Verify containers are running
docker ps
```

### 3. Install Go Dependencies
```bash
# Download Go modules
go mod download

# Tidy dependencies
go mod tidy
```

### 4. Generate Protocol Buffer Code (if needed)
```bash
# Regenerate protobuf files
export PATH="$PATH:$(go env GOPATH)/bin"
protoc --go_out=. --go-grpc_out=. --proto_path=. proto/context_gateway.proto
```

### 5. Build the Service
```bash
# Build the service binary
go build -o context-gateway cmd/main.go
```

### 6. Start the Service
```bash
# Run the service
./context-gateway

# Alternative: Run directly
go run cmd/main.go
```

## Service Configuration

### Default Settings
- **gRPC Port**: 8017
- **HTTP Port**: 8117
- **Redis**: localhost:6379
- **MongoDB**: localhost:27017
- **Database**: clinical_context_go

### Environment Variables
```bash
export GRPC_PORT=8017
export HTTP_PORT=8117
export REDIS_ADDR=localhost:6379
export MONGO_URI=mongodb://localhost:27017
export DB_NAME=clinical_context_go
export ENVIRONMENT=development
```

### Command Line Flags
```bash
./context-gateway \
  --grpc-port :8017 \
  --http-port :8117 \
  --redis-addr localhost:6379 \
  --mongo-uri mongodb://localhost:27017 \
  --db-name clinical_context_go \
  --env development
```

## Infrastructure Setup

### Docker Compose Configuration
The service uses Docker for its database dependencies:

```yaml
# docker-compose.yml
version: '3.8'
services:
  mongodb:
    image: mongo:7.0
    container_name: context-gateway-mongodb
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: password123
      MONGO_INITDB_DATABASE: clinical_context_go

  redis:
    image: redis:7-alpine
    container_name: context-gateway-redis
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
```

### Infrastructure Commands
```bash
# Start all dependencies
docker-compose up -d

# Stop dependencies
docker-compose down

# View logs
docker-compose logs -f

# Reset data (careful!)
docker-compose down -v
```

## Health Check Verification

### Test Service Health
```bash
# HTTP health check
curl http://localhost:8117/health

# Expected response:
# {"status":"healthy","service":"context-gateway-go","timestamp":"2025-09-15T..."}

# Comprehensive status
curl http://localhost:8117/status | python3 -m json.tool
```

### Test gRPC Health
```bash
# Using grpcurl (if installed)
grpcurl -plaintext localhost:8017 grpc.health.v1.Health/Check
```

### Test Federation Endpoint
```bash
# GraphQL federation endpoint
curl -X POST http://localhost:8117/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "{ _service { sdl } }"}'
```

## Service Architecture

### Dual-Layer Storage
- **Hot Storage (Redis)**: Fast cache for active snapshots
- **Cold Storage (MongoDB)**: Persistent storage for all snapshots
- **Automatic Failover**: Seamless fallback from hot to cold storage

### Service Components
- **gRPC Server**: High-performance API for service-to-service communication
- **HTTP Server**: RESTful API for web integration and health checks
- **Apollo Federation**: GraphQL schema composition for microservices
- **Recipe Engine**: Clinical workflow recipe management
- **Snapshot Manager**: Clinical data snapshot lifecycle management

### Core APIs

#### gRPC Services (Port 8017)
```protobuf
service ContextGateway {
  rpc CreateSnapshot(CreateSnapshotRequest) returns (ClinicalSnapshot);
  rpc GetSnapshot(GetSnapshotRequest) returns (ClinicalSnapshot);
  rpc ValidateSnapshot(ValidateSnapshotRequest) returns (ValidateSnapshotResponse);
  rpc LoadRecipe(LoadRecipeRequest) returns (WorkflowRecipe);
  rpc FetchLiveFields(LiveFetchRequest) returns (LiveFetchResponse);
  rpc StreamContextUpdates(StreamRequest) returns (stream ContextUpdate);
}
```

#### HTTP Endpoints (Port 8117)
```
GET  /health              - Service health check
GET  /ready               - Readiness probe
GET  /status              - Detailed service status
GET  /metrics             - Prometheus metrics
POST /api/federation      - GraphQL federation endpoint
GET  /                    - Service information
```

## Development Workflow

### Local Development
```bash
# 1. Start dependencies
docker-compose up -d

# 2. Make code changes
# 3. Rebuild if needed
go build -o context-gateway cmd/main.go

# 4. Restart service
./context-gateway

# 5. Test changes
curl http://localhost:8117/health
```

### Protocol Buffer Development
```bash
# After modifying .proto files
protoc --go_out=. --go-grpc_out=. --proto_path=. proto/context_gateway.proto

# Rebuild service
go build -o context-gateway cmd/main.go
```

## Advanced Configuration

### MongoDB Authentication
```bash
# If using authenticated MongoDB
export MONGO_URI="mongodb://admin:password123@localhost:27017"
```

### Redis Configuration
```bash
# Redis with password
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD="your-password"
```

### Production Environment
```bash
export ENVIRONMENT=production
export GRPC_PORT=8017
export HTTP_PORT=8117
```

## Troubleshooting

### Common Issues

#### **Docker Dependencies Not Running**
```bash
# Check container status
docker ps

# Restart containers
docker-compose down
docker-compose up -d

# Check container logs
docker-compose logs mongodb
docker-compose logs redis
```

#### **Port Conflicts**
```bash
# Check what's using ports
lsof -i :8017  # gRPC port
lsof -i :8117  # HTTP port
lsof -i :27017 # MongoDB
lsof -i :6379  # Redis

# Kill conflicting processes
kill -9 <PID>
```

#### **Go Module Issues**
```bash
# Clear Go module cache
go clean -modcache

# Reinstall dependencies
rm go.sum
go mod tidy
go mod download
```

#### **Protobuf Generation Issues**
```bash
# Ensure protoc plugins are installed
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Add to PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Regenerate
protoc --go_out=. --go-grpc_out=. --proto_path=. proto/context_gateway.proto
```

#### **MongoDB Connection Issues**
```bash
# Test MongoDB connection
docker exec -it context-gateway-mongodb mongosh

# Check authentication
docker exec -it context-gateway-mongodb mongosh -u admin -p password123
```

#### **Redis Connection Issues**
```bash
# Test Redis connection
docker exec -it context-gateway-redis redis-cli ping

# Should return: PONG
```

### Debug Mode
```bash
# Run with verbose logging
export ENVIRONMENT=development
./context-gateway --env development
```

## Performance and Monitoring

### Metrics Collection
The service exposes Prometheus-compatible metrics:
```bash
curl http://localhost:8117/metrics
```

### Health Monitoring
```bash
# Basic health
curl http://localhost:8117/health

# Detailed status with dependencies
curl http://localhost:8117/status
```

### Cache Statistics
```bash
# Get cache performance metrics
curl http://localhost:8117/status | jq '.cache_stats'
```

## Production Deployment

### Build for Production
```bash
# Optimized production build
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o context-gateway cmd/main.go
```

### Docker Deployment
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o context-gateway cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/context-gateway .
EXPOSE 8017 8117
CMD ["./context-gateway"]
```

### Production Configuration
```bash
# Production environment variables
export ENVIRONMENT=production
export GRPC_PORT=8017
export HTTP_PORT=8117
export REDIS_ADDR=redis-cluster:6379
export MONGO_URI=mongodb://mongodb-cluster:27017/clinical_context_go
export LOG_LEVEL=info
```

## Service Integration

### With Other Services
The Context Gateway is designed to work with:
- **Patient Service**: Clinical data aggregation
- **Medication Service**: Drug interaction checking
- **Apollo Federation**: GraphQL schema composition

### gRPC Client Example
```go
conn, err := grpc.Dial("localhost:8017", grpc.WithInsecure())
defer conn.Close()

client := pb.NewContextGatewayClient(conn)
snapshot, err := client.GetSnapshot(ctx, &pb.GetSnapshotRequest{
    SnapshotId: "snapshot-123",
    RequestingService: "patient-service",
})
```

### HTTP Client Example
```bash
# Create snapshot via HTTP
curl -X POST http://localhost:8117/api/snapshots \
  -H "Content-Type: application/json" \
  -d '{
    "recipe_id": "patient_context_v1",
    "patient_id": "patient-123"
  }'
```

## Advanced Features

### Recipe Management
The service supports clinical workflow recipes:
- Medication prescribing workflows
- Safety gateway contexts
- Emergency response contexts

### Live Field Fetching
Real-time data fetching with governance controls:
- Permission-based field access
- Audit trail generation
- Performance monitoring

### Cryptographic Integrity
- Snapshot checksums for data integrity
- Digital signatures for authentication
- Tamper detection capabilities

## Support and Maintenance

### Common Commands
```bash
# Start infrastructure
docker-compose up -d

# Build service
go build -o context-gateway cmd/main.go

# Run service
./context-gateway

# Stop service
pkill -f context-gateway

# Check processes
ps aux | grep context-gateway

# View service info
curl http://localhost:8117/
```

### Backup and Recovery
```bash
# Backup MongoDB
docker exec context-gateway-mongodb mongodump --out /backup

# Backup Redis
docker exec context-gateway-redis redis-cli BGSAVE
```

### Log Management
- Service logs to stdout/stderr in structured format
- Configure log aggregation for production
- Monitor error patterns and performance metrics

This comprehensive startup guide covers all aspects of running the Context Gateway Service, from basic setup to advanced production deployment scenarios.