# Medication Service V2 - Startup Guide

## Overview
The Medication Service V2 is a Go-based HTTP service that provides medication management capabilities. It uses the Gin web framework and implements RESTful APIs for medication proposals and clinical decision support.

## Prerequisites

### Required Software
- **Go 1.21+** - Programming language runtime
- **Git** - Version control (for dependency management)

### System Dependencies
- Network access for Go module downloads
- Available ports: 8005 (default HTTP port)

## Quick Start

### 1. Navigate to Service Directory
```bash
cd backend/services/medication-service-v2
```

### 2. Install Go Dependencies
```bash
# Download and install Go modules
go mod download

# Tidy up dependencies (optional)
go mod tidy
```

### 3. Build the Service
```bash
# Build the service binary
go build -o medication-service-v2 cmd/simple-server/main.go
```

### 4. Start the Service
```bash
# Run the built binary
./medication-service-v2

# Alternative: Run directly without building
go run cmd/simple-server/main.go
```

## Service Configuration

### Default Settings
- **Port**: 8005
- **Host**: All interfaces (0.0.0.0)
- **Framework**: Gin (Go web framework)
- **Log Level**: Production (structured JSON)

### Environment Variables
```bash
export MEDICATION_SERVICE_PORT=8005
export GIN_MODE=release        # For production
export LOG_LEVEL=info
```

### Port Configuration
```go
// In main.go, change the port variable:
port := "8005"  // Change to desired port
```

## Health Check Verification

### Test Service Health
```bash
# Check if service is running
curl http://localhost:8005/health

# Expected response:
# {
#   "service": "medication-service-v2",
#   "status": "healthy",
#   "version": "1.0.0"
# }
```

### Test Service Endpoints
```bash
# Test medication endpoints
curl http://localhost:8005/api/v1/medications

# Test medication proposals
curl -X POST http://localhost:8005/api/v1/medications/proposals \
  -H "Content-Type: application/json" \
  -d '{"patient_id": "test-123"}'
```

## Service Architecture

### Key Components
- **Gin Router**: HTTP request routing and middleware
- **Zap Logger**: Structured, high-performance logging
- **HTTP Handlers**: RESTful API endpoint implementations
- **JSON Responses**: Standardized API response format

### Service Endpoints
```go
// Health check
GET  /health

// Medication operations
GET  /api/v1/medications
POST /api/v1/medications/proposals

// Service information
GET  /
```

## Development Workflow

### Local Development
```bash
# 1. Make code changes
# 2. Test compilation
go build -o medication-service-v2 cmd/simple-server/main.go

# 3. Run service
./medication-service-v2

# 4. Test endpoints
curl http://localhost:8005/health
```

### Hot Reloading (Development)
```bash
# Install air for hot reloading (optional)
go install github.com/cosmtrek/air@latest

# Create .air.toml config and run
air
```

## Building and Deployment

### Local Build
```bash
# Standard build
go build -o medication-service-v2 cmd/simple-server/main.go

# Cross-compilation examples
GOOS=linux GOARCH=amd64 go build -o medication-service-v2-linux cmd/simple-server/main.go
GOOS=windows GOARCH=amd64 go build -o medication-service-v2.exe cmd/simple-server/main.go
```

### Production Build
```bash
# Optimized production build
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o medication-service-v2 cmd/simple-server/main.go
```

### Docker Deployment
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o medication-service-v2 cmd/simple-server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/medication-service-v2 .
EXPOSE 8005
CMD ["./medication-service-v2"]
```

## Troubleshooting

### Common Issues

#### **Go Module Issues**
```bash
# Error: Module not found or version conflicts
# Solution: Clean and reinstall modules
go clean -modcache
go mod download
go mod tidy
```

#### **Port Already in Use**
```bash
# Error: bind: address already in use
# Solution: Kill existing process or change port
lsof -ti:8005 | xargs kill -9
# OR change port in main.go
```

#### **Build Failures**
```bash
# Error: Compilation errors
# Solution: Check Go version and dependencies
go version  # Should be 1.21+
go mod verify
go build -v cmd/simple-server/main.go
```

#### **Import Path Issues**
```bash
# Error: Package import issues
# Solution: Ensure proper module structure
go mod init medication-service-v2  # if needed
go mod tidy
```

### Debug Mode
```bash
# Run with debug information
export GIN_MODE=debug
go run cmd/simple-server/main.go

# Check verbose build output
go build -v -o medication-service-v2 cmd/simple-server/main.go
```

## Service Integration

### With Other Services
```bash
# Test integration with other services
curl http://localhost:8003/health  # Patient Service
curl http://localhost:8005/health  # Medication Service (this)
curl http://localhost:8117/health  # Context Gateway
```

### API Integration Examples
```bash
# Create medication proposal
curl -X POST http://localhost:8005/api/v1/medications/proposals \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-123",
    "medication": "aspirin",
    "dosage": "100mg",
    "frequency": "daily"
  }'
```

## Performance and Monitoring

### Performance Tuning
```bash
# Set GOMAXPROCS for optimal CPU usage
export GOMAXPROCS=4

# Configure garbage collection
export GOGC=100
```

### Monitoring Endpoints
- **Health**: `GET /health` - Service health status
- **Metrics**: Custom metrics can be added using Prometheus
- **Profiling**: Enable pprof for performance analysis

### Logging
```go
// Zap logger provides structured logging
logger.Info("Server starting", zap.String("port", port))
logger.Error("Server error", zap.Error(err))
```

## Production Considerations

### Security
- Implement authentication middleware
- Add request rate limiting
- Configure CORS policies
- Use HTTPS in production

### Scalability
- Horizontal scaling with load balancer
- Health check endpoints for service discovery
- Graceful shutdown handling
- Connection pooling for database operations

### Configuration Management
```bash
# Use environment variables for configuration
export MEDICATION_SERVICE_PORT=8005
export DATABASE_URL=postgresql://...
export JWT_SECRET=your-secret-key
```

## Advanced Configuration

### Custom Middleware
```go
// Add custom middleware to Gin router
router.Use(gin.Logger())
router.Use(gin.Recovery())
router.Use(customAuthMiddleware())
```

### Structured Configuration
```go
type Config struct {
    Port        string `env:"MEDICATION_SERVICE_PORT" envDefault:"8005"`
    DatabaseURL string `env:"DATABASE_URL"`
    LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
}
```

## Support and Maintenance

### Common Commands
```bash
# Build service
go build -o medication-service-v2 cmd/simple-server/main.go

# Run service
./medication-service-v2

# Check running processes
ps aux | grep medication-service-v2

# Stop service
pkill -f medication-service-v2

# View Go environment
go env
```

### Logs and Debugging
- Service logs are output to stdout/stderr
- Use structured JSON logging for production
- Configure log rotation for persistent deployments
- Enable debug mode for development troubleshooting

### Version Management
```bash
# Check service version
curl http://localhost:8005/health | jq .version

# Update Go modules
go get -u ./...
go mod tidy
```