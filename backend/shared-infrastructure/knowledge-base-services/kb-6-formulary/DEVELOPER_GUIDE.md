# KB-6 Formulary Management Service - Developer Guide

## 🚀 Getting Started

### Prerequisites
- **Go 1.21+** - Latest stable Go version
- **Docker & Docker Compose** - For infrastructure services
- **Make** - Build automation (optional)
- **Git** - Version control
- **PostgreSQL Client** - Database operations (optional)
- **Redis CLI** - Cache debugging (optional)

### Development Environment Setup

#### 1. Clone and Setup
```bash
# Clone the repository
git clone <repository-url>
cd kb-6-formulary

# Install Go dependencies
go mod download

# Install development tools
go install github.com/air-verse/air@latest          # Live reload
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest  # Linting
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest       # Security scanning
```

#### 2. Infrastructure Services
```bash
# Start all infrastructure services
docker-compose up -d

# Verify services are running
docker-compose ps

# View logs
docker-compose logs -f
```

#### 3. Database Setup
```bash
# Run database migrations
psql -h localhost -p 5433 -U postgres -d kb6_formulary -f migrations/001_initial_schema.sql

# Load development data (optional)
go run main.go -load-mock-data

# Verify database setup
psql -h localhost -p 5433 -U postgres -d kb6_formulary -c "SELECT COUNT(*) FROM formulary_entries;"
```

#### 4. Configuration
```bash
# Copy example configuration
cp config/config.example.yaml config/config.yaml

# Edit configuration for development
nano config/config.yaml
```

Example development configuration:
```yaml
server:
  port: "8086"
  environment: "development"
  
database:
  host: "localhost"
  port: "5433"
  database: "kb6_formulary"
  username: "postgres"
  password: "postgres"
  max_connections: 10
  
redis:
  address: "localhost:6380"
  database: 1
  password: ""
  
elasticsearch:
  enabled: true
  addresses: ["http://localhost:9200"]
  
logging:
  level: "debug"
  format: "text"
```

## 🏗️ Project Structure

### **Code Organization**
```
kb-6-formulary/
├── main.go                    # Application entry point
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── internal/                  # Private application code
│   ├── config/               # Configuration management
│   ├── database/             # Data access layer
│   ├── cache/                # Caching layer
│   ├── services/             # Business logic
│   ├── handlers/             # HTTP request handlers
│   ├── grpc/                 # gRPC server implementation
│   ├── server/               # HTTP server setup
│   ├── middleware/           # HTTP middleware
│   └── models/               # Data models
├── proto/                    # Protocol buffer definitions
├── migrations/               # Database migrations
├── config/                   # Configuration files
├── api/                      # API specifications
└── schemas/                  # JSON schemas
```

### **Key Files and Directories**

#### **Core Application** 
- `main.go` - Application bootstrap and dependency injection
- `internal/services/formulary_service.go` - Main business logic (2000+ lines)
- `internal/services/inventory_service.go` - Inventory management logic
- `internal/handlers/` - HTTP request handlers for REST API
- `internal/grpc/server.go` - gRPC service implementation

#### **Data Layer**
- `internal/database/connection.go` - PostgreSQL connection management
- `internal/database/elasticsearch_connection.go` - Search integration
- `internal/cache/redis_manager.go` - Redis caching operations
- `migrations/001_initial_schema.sql` - Database schema definition

#### **Configuration & Deployment**
- `proto/kb6.proto` - gRPC service definitions
- `docker-compose.yml` - Development infrastructure
- `api/openapi.yaml` - REST API specification
- `config/` - Environment-specific configurations

## 💻 Development Workflow

### **Daily Development**
```bash
# Start development infrastructure
docker-compose up -d

# Run with live reload
air

# Or run without live reload
go run main.go

# In another terminal - test the service
curl http://localhost:8087/health
```

### **Code Quality Checks**
```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Security scan
gosec ./...

# Run tests
go test ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### **Building**
```bash
# Development build
go build -o bin/kb6-formulary

# Production build with optimizations
go build -ldflags="-w -s" -o bin/kb6-formulary-prod

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o bin/kb6-formulary-linux
GOOS=windows GOARCH=amd64 go build -o bin/kb6-formulary.exe
```

## 🧩 Code Architecture

### **Service Layer Development**

#### **Adding New Endpoints**
```go
// 1. Define request/response types in internal/services/types.go
type NewFeatureRequest struct {
    ID        string `json:"id" validate:"required"`
    Parameter string `json:"parameter" validate:"min=1,max=100"`
    RequestID string `json:"request_id,omitempty"`
}

type NewFeatureResponse struct {
    Result    string    `json:"result"`
    Timestamp time.Time `json:"timestamp"`
    RequestID string    `json:"request_id"`
}

// 2. Add business logic to service (internal/services/formulary_service.go)
func (fs *FormularyService) ProcessNewFeature(ctx context.Context, req *NewFeatureRequest) (*NewFeatureResponse, error) {
    start := time.Now()
    log.Printf("Processing new feature for ID: %s", req.ID)
    
    // Business logic here
    result, err := fs.performBusinessLogic(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to process feature: %w", err)
    }
    
    response := &NewFeatureResponse{
        Result:    result,
        Timestamp: time.Now(),
        RequestID: req.RequestID,
    }
    
    log.Printf("New feature processed in %v", time.Since(start))
    return response, nil
}

// 3. Add HTTP handler (internal/handlers/formulary_handler.go)
func (h *FormularyHandler) ProcessNewFeature(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()
    
    var req services.NewFeatureRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Validate request
    if err := h.validate.Struct(req); err != nil {
        http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
        return
    }
    
    // Set request ID if not provided
    if req.RequestID == "" {
        req.RequestID = generateRequestID()
    }
    
    response, err := h.formularyService.ProcessNewFeature(ctx, &req)
    if err != nil {
        log.Printf("Error processing new feature: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// 4. Register route (internal/server/http_server.go)
mux.HandleFunc(apiPrefix+"/new-feature", s.formularyHandler.ProcessNewFeature)
```

#### **Adding gRPC Methods**
```go
// 1. Update proto/kb6.proto
service KB6Service {
    rpc ProcessNewFeature(NewFeatureRequest) returns (NewFeatureResponse);
}

message NewFeatureRequest {
    string transaction_id = 1;
    string id = 2;
    string parameter = 3;
}

message NewFeatureResponse {
    string result = 1;
    google.protobuf.Timestamp response_time = 2;
}

// 2. Regenerate protobuf code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/kb6.proto

// 3. Implement in gRPC server (internal/grpc/server.go)
func (s *Server) ProcessNewFeature(ctx context.Context, req *pb.NewFeatureRequest) (*pb.NewFeatureResponse, error) {
    // Convert protobuf to service types
    serviceReq := &services.NewFeatureRequest{
        ID:        req.Id,
        Parameter: req.Parameter,
        RequestID: req.TransactionId,
    }
    
    result, err := s.formularyService.ProcessNewFeature(ctx, serviceReq)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to process: %v", err)
    }
    
    return &pb.NewFeatureResponse{
        Result:       result.Result,
        ResponseTime: timestamppb.New(result.Timestamp),
    }, nil
}
```

### **Database Operations**

#### **Adding New Queries**
```go
// internal/services/formulary_service.go
func (fs *FormularyService) queryNewData(ctx context.Context, id string) (*DataResult, error) {
    query := `
        SELECT 
            column1,
            column2,
            created_at
        FROM table_name 
        WHERE id = $1 
            AND status = 'active'
        ORDER BY created_at DESC
        LIMIT 1`
    
    var result DataResult
    err := fs.db.QueryRow(ctx, query, id).Scan(
        &result.Column1,
        &result.Column2,
        &result.CreatedAt,
    )
    
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("failed to query data: %w", err)
    }
    
    return &result, nil
}
```

#### **Database Migrations**
```sql
-- migrations/002_new_feature.sql
-- Add migration description and version
-- Migration: Add new feature table
-- Version: 002
-- Date: 2025-09-03

CREATE TABLE new_feature_data (
    id SERIAL PRIMARY KEY,
    feature_id VARCHAR(50) NOT NULL UNIQUE,
    data_value TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for performance
CREATE INDEX idx_new_feature_lookup ON new_feature_data (feature_id, status);
CREATE INDEX idx_new_feature_timestamp ON new_feature_data (created_at);

-- Add constraints
ALTER TABLE new_feature_data ADD CONSTRAINT check_status 
    CHECK (status IN ('active', 'inactive', 'archived'));
```

### **Caching Implementation**

#### **Adding Cache Operations**
```go
// internal/cache/redis_manager.go
func (rm *RedisManager) SetNewFeatureCache(key string, data interface{}, ttl time.Duration) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal cache data: %w", err)
    }
    
    return rm.client.Set(context.Background(), key, jsonData, ttl).Err()
}

func (rm *RedisManager) GetNewFeatureCache(key string) ([]byte, error) {
    data, err := rm.client.Get(context.Background(), key).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, ErrCacheNotFound
        }
        return nil, fmt.Errorf("failed to get cache data: %w", err)
    }
    
    return []byte(data), nil
}

// Cache invalidation
func (rm *RedisManager) InvalidateNewFeatureCache(pattern string) error {
    return rm.InvalidatePattern(fmt.Sprintf("kb6:newfeature:%s", pattern))
}
```

#### **Service-Level Caching**
```go
func (fs *FormularyService) getDataWithCache(ctx context.Context, id string) (*DataResult, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("kb6:newfeature:%s", id)
    if cachedData, err := fs.cache.GetNewFeatureCache(cacheKey); err == nil {
        var result DataResult
        if err := json.Unmarshal(cachedData, &result); err == nil {
            log.Printf("Cache hit for new feature: %s", id)
            return &result, nil
        }
    }
    
    // Cache miss - query database
    result, err := fs.queryNewData(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    if err := fs.cache.SetNewFeatureCache(cacheKey, result, 15*time.Minute); err != nil {
        log.Printf("Warning: failed to cache result: %v", err)
    }
    
    return result, nil
}
```

## 🧪 Testing

### **Unit Testing**
```go
// internal/services/formulary_service_test.go
package services

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock database for testing
type MockDatabase struct {
    mock.Mock
}

func (m *MockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
    mockArgs := m.Called(ctx, query, args)
    return mockArgs.Get(0).(*sql.Row)
}

func TestFormularyService_CheckCoverage(t *testing.T) {
    // Setup
    mockDB := new(MockDatabase)
    mockCache := new(MockRedisManager)
    service := &FormularyService{
        db:    mockDB,
        cache: mockCache,
    }
    
    // Test data
    request := &CoverageRequest{
        TransactionID: "test-001",
        DrugRxNorm:    "197361",
        PayerID:       "aetna-001",
        PlanID:        "aetna-standard-2025",
    }
    
    // Mock expectations
    mockCache.On("GetCoverage", mock.AnythingOfType("string")).Return(nil, ErrCacheNotFound)
    mockDB.On("QueryRow", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(mockRow)
    
    // Execute
    result, err := service.CheckCoverage(context.Background(), request)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "test-001", result.Evidence.Provenance["transaction_id"])
    
    // Verify mocks
    mockDB.AssertExpectations(t)
    mockCache.AssertExpectations(t)
}
```

### **Integration Testing**
```go
// tests/integration/formulary_integration_test.go
//go:build integration
// +build integration

package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/suite"
)

type FormularyIntegrationTestSuite struct {
    suite.Suite
    service *services.FormularyService
    db      *database.Connection
    cache   *cache.RedisManager
}

func (suite *FormularyIntegrationTestSuite) SetupSuite() {
    // Setup test database and cache
    config := loadTestConfig()
    
    db, err := database.NewConnection(config)
    suite.Require().NoError(err)
    
    cache, err := cache.NewRedisManager(&config.Redis)
    suite.Require().NoError(err)
    
    suite.db = db
    suite.cache = cache
    suite.service = services.NewFormularyService(db, cache, nil)
    
    // Load test data
    suite.loadTestData()
}

func (suite *FormularyIntegrationTestSuite) TestCoverageAnalysisFlow() {
    request := &services.CoverageRequest{
        TransactionID: "integration-test-001",
        DrugRxNorm:    "197361",
        PayerID:       "test-payer",
        PlanID:        "test-plan",
        PlanYear:      2025,
        Quantity:      30,
    }
    
    result, err := suite.service.CheckCoverage(context.Background(), request)
    
    suite.NoError(err)
    suite.NotNil(result)
    suite.Equal("integration-test-001", result.Evidence.Provenance["transaction_id"])
}

func TestFormularyIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(FormularyIntegrationTestSuite))
}
```

### **Load Testing**
```go
// tests/load/load_test.go
//go:build load
// +build load

package load

import (
    "context"
    "sync"
    "testing"
    "time"
)

func BenchmarkCoverageAnalysis(b *testing.B) {
    service := setupTestService()
    request := &services.CoverageRequest{
        DrugRxNorm: "197361",
        PayerID:    "test-payer",
        PlanID:     "test-plan",
    }
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := service.CheckCoverage(context.Background(), request)
            if err != nil {
                b.Error(err)
            }
        }
    })
}

func BenchmarkCostAnalysisConcurrent(b *testing.B) {
    service := setupTestService()
    
    const concurrency = 10
    const requestsPerWorker = 100
    
    b.ResetTimer()
    
    var wg sync.WaitGroup
    start := time.Now()
    
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < requestsPerWorker; j++ {
                request := &services.CostAnalysisRequest{
                    DrugRxNorms: []string{"197361", "308136"},
                    PayerID:     "test-payer",
                    PlanID:      "test-plan",
                }
                
                _, err := service.AnalyzeCosts(context.Background(), request)
                if err != nil {
                    b.Error(err)
                }
            }
        }()
    }
    
    wg.Wait()
    duration := time.Since(start)
    
    totalRequests := concurrency * requestsPerWorker
    b.Logf("Processed %d requests in %v (%.2f req/sec)", 
        totalRequests, duration, float64(totalRequests)/duration.Seconds())
}
```

## 🔧 Configuration Management

### **Environment-Specific Configurations**

#### **Development Configuration**
```yaml
# config/development.yaml
server:
  port: "8086"
  environment: "development"
  
database:
  host: "localhost"
  port: "5433"
  max_connections: 10
  log_queries: true
  
redis:
  address: "localhost:6380"
  database: 1
  
elasticsearch:
  enabled: true
  addresses: ["http://localhost:9200"]
  
logging:
  level: "debug"
  format: "text"
  output: "stdout"
  
cost_analysis:
  cache_ttl_minutes: 5  # Shorter TTL for development
  max_alternatives_per_drug: 15  # More alternatives for testing
```

#### **Production Configuration**
```yaml
# config/production.yaml
server:
  port: "${GRPC_PORT:-8086}"
  environment: "production"
  
database:
  host: "${DB_HOST}"
  port: "${DB_PORT:-5432}"
  database: "${DB_NAME}"
  username: "${DB_USER}"
  password: "${DB_PASSWORD}"
  max_connections: 25
  ssl_mode: "require"
  
redis:
  address: "${REDIS_URL}"
  password: "${REDIS_PASSWORD}"
  database: 1
  pool_size: 20
  
elasticsearch:
  enabled: true
  addresses: ["${ES_URL}"]
  username: "${ES_USERNAME}"
  password: "${ES_PASSWORD}"
  
logging:
  level: "info"
  format: "json"
  output: "/var/log/kb6-formulary.log"
  
security:
  jwt_secret: "${JWT_SECRET}"
  tls_enabled: true
  rate_limit_rpm: 100
```

### **Feature Flags**
```go
// internal/config/config.go
type FeatureFlags struct {
    CostAnalysisEnabled   bool `yaml:"cost_analysis_enabled" default:"true"`
    SemanticSearchEnabled bool `yaml:"semantic_search_enabled" default:"true"`
    AdvancedCachingEnabled bool `yaml:"advanced_caching_enabled" default:"true"`
    MetricsEnabled        bool `yaml:"metrics_enabled" default:"true"`
    AuditLoggingEnabled   bool `yaml:"audit_logging_enabled" default:"true"`
}

func (c *Config) IsFeatureEnabled(feature string) bool {
    switch feature {
    case "cost_analysis":
        return c.Features.CostAnalysisEnabled
    case "semantic_search":
        return c.Features.SemanticSearchEnabled
    case "advanced_caching":
        return c.Features.AdvancedCachingEnabled
    default:
        return false
    }
}
```

## 🐛 Debugging

### **Logging Best Practices**
```go
// Use structured logging with consistent fields
log.WithFields(logrus.Fields{
    "transaction_id": req.TransactionID,
    "drug_rxnorm":   req.DrugRxNorm,
    "payer_id":      req.PayerID,
    "duration_ms":   time.Since(start).Milliseconds(),
}).Info("Coverage analysis completed")

// Error logging with context
log.WithFields(logrus.Fields{
    "transaction_id": req.TransactionID,
    "error":         err.Error(),
    "query":         query,
}).Error("Database query failed")
```

### **Performance Profiling**
```go
// Enable pprof in development
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

```bash
# Profile CPU usage
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Profile memory usage
go tool pprof http://localhost:6060/debug/pprof/heap

# View goroutines
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### **Database Debugging**
```sql
-- Enable query logging in PostgreSQL
ALTER SYSTEM SET log_statement = 'all';
SELECT pg_reload_conf();

-- Monitor slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
WHERE mean_time > 100 
ORDER BY mean_time DESC;

-- Check connection usage
SELECT count(*) as connections, state 
FROM pg_stat_activity 
GROUP BY state;
```

### **Redis Debugging**
```bash
# Monitor Redis operations
redis-cli -h localhost -p 6380 monitor

# Check memory usage
redis-cli -h localhost -p 6380 info memory

# Analyze key patterns
redis-cli -h localhost -p 6380 --scan --pattern "kb6:*" | head -20

# Monitor slow queries
redis-cli -h localhost -p 6380 slowlog get 10
```

## 📊 Monitoring & Observability

### **Custom Metrics**
```go
// internal/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    CostAnalysisRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kb6_cost_analysis_requests_total",
            Help: "Total number of cost analysis requests",
        },
        []string{"optimization_goal", "status"},
    )
    
    AlternativesFoundTotal = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "kb6_alternatives_found_total",
            Help: "Number of alternatives found per request",
            Buckets: []float64{0, 1, 5, 10, 15, 20, 25, 30},
        },
        []string{"discovery_strategy"},
    )
    
    CacheHitRate = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "kb6_cache_hit_rate",
            Help: "Cache hit rate percentage",
        },
        []string{"cache_type"},
    )
)

// Usage in service
func (fs *FormularyService) AnalyzeCosts(ctx context.Context, req *CostAnalysisRequest) {
    metrics.CostAnalysisRequestsTotal.WithLabelValues(req.OptimizationGoal, "started").Inc()
    
    defer func() {
        if err != nil {
            metrics.CostAnalysisRequestsTotal.WithLabelValues(req.OptimizationGoal, "error").Inc()
        } else {
            metrics.CostAnalysisRequestsTotal.WithLabelValues(req.OptimizationGoal, "success").Inc()
        }
    }()
    
    // Business logic...
}
```

### **Health Check Development**
```go
// Custom health checks
func (fs *FormularyService) DetailedHealthCheck(ctx context.Context) map[string]interface{} {
    health := make(map[string]interface{})
    
    // Database connection check
    start := time.Now()
    err := fs.db.PingContext(ctx)
    dbLatency := time.Since(start)
    
    health["database"] = map[string]interface{}{
        "status":     getStatusString(err),
        "latency_ms": dbLatency.Milliseconds(),
        "error":      getErrorString(err),
    }
    
    // Cache connection check
    start = time.Now()
    err = fs.cache.Ping()
    cacheLatency := time.Since(start)
    
    health["cache"] = map[string]interface{}{
        "status":     getStatusString(err),
        "latency_ms": cacheLatency.Milliseconds(),
        "error":      getErrorString(err),
    }
    
    // Elasticsearch check (if enabled)
    if fs.es != nil {
        start = time.Now()
        err = fs.es.HealthCheck(ctx)
        esLatency := time.Since(start)
        
        health["elasticsearch"] = map[string]interface{}{
            "status":     getStatusString(err),
            "latency_ms": esLatency.Milliseconds(),
            "error":      getErrorString(err),
        }
    }
    
    return health
}
```

## 🚀 Deployment

### **Docker Development**
```dockerfile
# Dockerfile.dev
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bin/kb6-formulary

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/bin/kb6-formulary .
COPY --from=builder /app/config/ ./config/
COPY --from=builder /app/migrations/ ./migrations/

CMD ["./kb6-formulary"]
```

### **Multi-Stage Production Build**
```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o bin/kb6-formulary

# Final stage
FROM scratch

# Copy certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy application binary and configuration
COPY --from=builder /app/bin/kb6-formulary /kb6-formulary
COPY --from=builder /app/config/ /config/
COPY --from=builder /app/migrations/ /migrations/

# Expose ports
EXPOSE 8086 8087

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/kb6-formulary", "-health-check"]

# Run application
ENTRYPOINT ["/kb6-formulary"]
```

### **Docker Compose for Development**
```yaml
# docker-compose.dev.yml
version: '3.8'

services:
  kb6-formulary:
    build:
      context: .
      dockerfile: Dockerfile.dev
    ports:
      - "8086:8086"  # gRPC
      - "8087:8087"  # HTTP
      - "6060:6060"  # pprof
    environment:
      - DB_HOST=postgres
      - REDIS_URL=redis:6379
      - ES_URL=http://elasticsearch:9200
      - LOG_LEVEL=debug
    depends_on:
      - postgres
      - redis
      - elasticsearch
    volumes:
      - ./config:/app/config
      - ./logs:/var/log
    networks:
      - kb6-network

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=kb6_formulary
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
    ports:
      - "5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    networks:
      - kb6-network

  redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    volumes:
      - redis_data:/data
    networks:
      - kb6-network

  elasticsearch:
    image: elasticsearch:8.12.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms1g -Xmx1g"
    ports:
      - "9200:9200"
    volumes:
      - es_data:/usr/share/elasticsearch/data
    networks:
      - kb6-network

volumes:
  postgres_data:
  redis_data:
  es_data:

networks:
  kb6-network:
    driver: bridge
```

## 📚 Best Practices

### **Code Quality Standards**

#### **Error Handling**
```go
// Good: Specific error types
var (
    ErrNotFound     = errors.New("resource not found")
    ErrInvalidInput = errors.New("invalid input parameter")
    ErrCacheNotFound = errors.New("cache entry not found")
)

// Good: Wrapped errors with context
func (fs *FormularyService) getFormularyData(ctx context.Context, id string) error {
    data, err := fs.db.Query(ctx, query, id)
    if err != nil {
        return fmt.Errorf("failed to query formulary data for id %s: %w", id, err)
    }
    return nil
}

// Good: Error handling with appropriate actions
result, err := service.AnalyzeCosts(ctx, request)
if err != nil {
    // Log error with context
    log.WithFields(logrus.Fields{
        "transaction_id": request.TransactionID,
        "error": err,
    }).Error("Cost analysis failed")
    
    // Return appropriate HTTP status
    if errors.Is(err, ErrNotFound) {
        http.Error(w, "Resource not found", http.StatusNotFound)
        return
    }
    
    http.Error(w, "Internal server error", http.StatusInternalServerError)
    return
}
```

#### **Context Usage**
```go
// Good: Proper context usage
func (fs *FormularyService) longRunningOperation(ctx context.Context) error {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Pass context to downstream operations
    result, err := fs.db.QueryContext(ctx, query)
    if err != nil {
        return err
    }
    
    // Check context again for long operations
    for i, item := range result {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        if err := fs.processItem(ctx, item); err != nil {
            return err
        }
    }
    
    return nil
}
```

#### **Concurrency Safety**
```go
// Good: Proper synchronization
type SafeCounter struct {
    mu    sync.RWMutex
    count map[string]int
}

func (c *SafeCounter) Increment(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count[key]++
}

func (c *SafeCounter) Get(key string) int {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.count[key]
}
```

### **Performance Guidelines**

#### **Database Optimization**
```go
// Good: Prepared statements for repeated queries
type FormularyService struct {
    db                 *database.Connection
    getCoverageStmt    *sql.Stmt
    getAlternativesStmt *sql.Stmt
}

func (fs *FormularyService) initPreparedStatements() error {
    var err error
    
    fs.getCoverageStmt, err = fs.db.Prepare(getCoverageQuery)
    if err != nil {
        return fmt.Errorf("failed to prepare coverage statement: %w", err)
    }
    
    fs.getAlternativesStmt, err = fs.db.Prepare(getAlternativesQuery)
    if err != nil {
        return fmt.Errorf("failed to prepare alternatives statement: %w", err)
    }
    
    return nil
}

// Good: Batch operations
func (fs *FormularyService) processMultipleDrugs(ctx context.Context, drugIDs []string) error {
    // Prepare batch query
    query := "SELECT * FROM formulary_entries WHERE drug_rxnorm = ANY($1)"
    
    rows, err := fs.db.Query(ctx, query, pq.Array(drugIDs))
    if err != nil {
        return err
    }
    defer rows.Close()
    
    // Process results
    for rows.Next() {
        // Process each row
    }
    
    return rows.Err()
}
```

#### **Memory Management**
```go
// Good: Memory-efficient processing
func (fs *FormularyService) processLargeDataset(ctx context.Context) error {
    // Use streaming to avoid loading everything into memory
    rows, err := fs.db.Query(ctx, "SELECT * FROM large_table")
    if err != nil {
        return err
    }
    defer rows.Close()
    
    // Process one row at a time
    for rows.Next() {
        var item DataItem
        if err := rows.Scan(&item); err != nil {
            return err
        }
        
        // Process item
        if err := fs.processItem(ctx, &item); err != nil {
            return err
        }
        
        // Optional: yield to scheduler for long-running operations
        runtime.Gosched()
    }
    
    return rows.Err()
}
```

## 🔄 Contributing

### **Pull Request Process**
```bash
# 1. Create feature branch
git checkout -b feature/new-cost-algorithm

# 2. Make changes with tests
# 3. Run quality checks
make test
make lint
make security-scan

# 4. Commit with descriptive message
git commit -m "feat: add new cost optimization algorithm

- Implement portfolio-level optimization
- Add therapeutic class clustering
- Include comprehensive tests
- Update documentation"

# 5. Push and create PR
git push origin feature/new-cost-algorithm
```

### **Code Review Checklist**
- [ ] **Functionality**: Code works as intended
- [ ] **Tests**: Comprehensive test coverage (>80%)
- [ ] **Performance**: No performance regressions
- [ ] **Security**: No security vulnerabilities
- [ ] **Documentation**: Updated documentation and comments
- [ ] **Backwards Compatibility**: API changes are backwards compatible
- [ ] **Error Handling**: Proper error handling and logging
- [ ] **Code Style**: Follows Go conventions and project standards

---

## 📋 Developer Guide Summary

This comprehensive developer guide provides everything needed to contribute effectively to the KB-6 Formulary Management Service:

### **🚀 Quick Setup**
- Complete development environment setup with Docker infrastructure
- Database migration and mock data loading procedures
- Live reload development workflow with Air

### **🏗️ Architecture Understanding** 
- Detailed code organization and structure explanation
- Service layer development patterns and best practices
- Database, caching, and search integration guidelines

### **🧪 Testing Excellence**
- Unit testing with mocks and comprehensive assertions
- Integration testing with real infrastructure components
- Load testing and performance benchmarking procedures

### **🔧 Operational Knowledge**
- Configuration management for different environments
- Debugging techniques with profiling and monitoring tools
- Deployment strategies with Docker and production optimizations

### **📊 Quality Standards**
- Code quality guidelines with error handling best practices
- Performance optimization techniques and memory management
- Security considerations and audit compliance requirements

**Developer Readiness**: ✅ **Complete** - All information needed for productive development and contribution to the KB-6 service codebase.