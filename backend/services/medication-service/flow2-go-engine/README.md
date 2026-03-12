# Flow 2 Go Enhanced Orchestrator

The **Go Enhanced Orchestrator** is Service 1 of the Flow 2 Greenfield implementation - a high-performance, production-ready orchestration service that coordinates clinical medication intelligence workflows.

## 🎯 **Purpose**

This service acts as the "Front Office" of the Flow 2 architecture:
- **Receives** API requests from clients (HTTP/GraphQL)
- **Assembles** clinical context from multiple sources
- **Coordinates** with the Rust Clinical Recipe Engine
- **Optimizes** and formats responses
- **Provides** comprehensive observability

## 🏗️ **Architecture**

```
Client Request → Go Orchestrator → Rust Recipe Engine → Optimized Response
                      ↓                    ↓
                Context Assembly      Recipe Execution
                Response Optimization  Clinical Logic
                Metrics & Caching     Parallel Processing
```

## 🚀 **Features**

### **Core Capabilities**
- ✅ **Flow 2 Execution**: Complete medication intelligence workflows
- ✅ **Medication Intelligence**: AI-powered medication analysis
- ✅ **Dose Optimization**: ML-guided dose calculations
- ✅ **Safety Validation**: Comprehensive safety checking
- ✅ **Clinical Intelligence**: Outcome prediction and insights

### **Performance Features**
- ✅ **Parallel Context Assembly**: Concurrent data gathering
- ✅ **Smart Caching**: Multi-level caching with Redis
- ✅ **Circuit Breaker**: Automatic fallback mechanisms
- ✅ **Connection Pooling**: Optimized resource usage

### **Observability**
- ✅ **Structured Logging**: JSON-formatted logs with correlation IDs
- ✅ **Prometheus Metrics**: Comprehensive performance metrics
- ✅ **Health Checks**: Liveness, readiness, and dependency checks
- ✅ **Distributed Tracing**: Request tracing across services

## 📋 **API Endpoints**

### **Flow 2 Endpoints**
```
POST /api/v1/flow2/execute                    # Main Flow 2 execution
POST /api/v1/flow2/medication-intelligence    # Medication intelligence
POST /api/v1/flow2/dose-optimization          # Dose optimization
POST /api/v1/flow2/safety-validation          # Safety validation
POST /api/v1/flow2/clinical-intelligence      # Clinical intelligence
```

### **Analytics Endpoints**
```
POST /api/v1/flow2/analytics/collect          # Collect analytics
GET  /api/v1/flow2/analytics/{patient_id}     # Patient analytics
GET  /api/v1/flow2/recommendations/{patient_id} # Patient recommendations
```

### **GraphQL Endpoint**
```
POST /graphql                                 # GraphQL interface
```

### **System Endpoints**
```
GET /health                                   # Health check
GET /health/ready                             # Readiness check
GET /health/live                              # Liveness check
GET /metrics                                  # Prometheus metrics
```

## 🛠️ **Development Setup**

### **Prerequisites**
- Go 1.21+
- Docker & Docker Compose
- **Redis running on localhost:6379** (REQUIRED - no fallback)
- **Rust Recipe Engine running on localhost:50051** (REQUIRED - no fallback)
- Context Service (when implemented)
- Medication API (when implemented)

### **Quick Start**

1. **Initialize Go Module**
```bash
cd flow2-go-engine
go mod tidy
```

2. **Run with Docker Compose**
```bash
cd ../
python start_flow2_development.py --build
```

3. **Run Locally (Development)**
```bash
cd flow2-go-engine
go run cmd/server/main.go
```

4. **Test the Service**
```bash
cd ../
python test_flow2_go_engine.py
```

### **Configuration**

The service uses environment variables and YAML configuration:

```yaml
# configs/config.yaml
server:
  port: 8080
  environment: development

rust_engine:
  address: "localhost:50051"
  timeout: 30s

redis:
  address: "localhost:6379"
  pool_size: 10
```

**Environment Variables:**
- `RUST_ENGINE_ADDRESS`: Rust engine gRPC address
- `REDIS_URL`: Redis connection URL
- `SERVER_PORT`: HTTP server port
- `LOG_LEVEL`: Logging level

## 🧪 **Testing**

### **Unit Tests**
```bash
go test ./...
```

### **Integration Tests**
```bash
python test_flow2_go_engine.py
```

### **Load Testing**
```bash
# Run performance test
python test_flow2_go_engine.py --performance
```

## 📊 **Monitoring**

### **Metrics Available**
- `flow2_execution_duration_seconds`: Flow 2 execution time
- `flow2_execution_total`: Total Flow 2 executions
- `http_request_duration_seconds`: HTTP request latency
- `rust_engine_latency_seconds`: Rust engine call latency
- `cache_hits_total`: Cache hit counts

### **Health Checks**
- **Liveness**: Service is running
- **Readiness**: Service can handle requests
- **Dependencies**: Rust engine, Redis, Context service status

### **Logging**
Structured JSON logs with fields:
- `request_id`: Unique request identifier
- `patient_id`: Patient identifier
- `execution_time_ms`: Execution time
- `overall_status`: Result status

## 🔧 **Architecture Details**

### **Components**

1. **Orchestrator** (`internal/flow2/orchestrator.go`)
   - Main request handler
   - Endpoint implementations
   - Error handling

2. **Context Assembler** (`internal/flow2/context_assembler.go`)
   - Parallel context gathering
   - Clinical data aggregation
   - Context enrichment

3. **Recipe Coordinator** (`internal/flow2/recipe_coordinator.go`)
   - Rust engine communication
   - Recipe execution coordination
   - Response handling

4. **Response Optimizer** (`internal/flow2/response_optimizer.go`)
   - Response formatting
   - Analytics building
   - Performance optimization

### **Clients**

1. **Rust Recipe Client** (`internal/clients/rust_recipe_client.go`)
   - gRPC communication with Rust engine
   - Mock client for development
   - Circuit breaker integration

2. **Context Service Client** (`internal/clients/interfaces.go`)
   - GraphQL queries to Context Service
   - Patient data retrieval
   - Mock implementation

### **Services**

1. **Cache Service** (`internal/services/cache_service.go`)
   - Redis integration
   - Multi-level caching
   - Mock cache for development

2. **Metrics Service** (`internal/services/metrics_service.go`)
   - Prometheus metrics
   - Performance tracking
   - Custom metrics

3. **Health Service** (`internal/services/health_service.go`)
   - Health checks
   - Dependency monitoring
   - Status reporting

## 🚀 **Deployment**

### **Docker**
```bash
docker build -t flow2-go-engine .
docker run -p 8080:8080 flow2-go-engine
```

### **Kubernetes**
```bash
kubectl apply -f k8s/
```

### **Production Configuration**
- Use real Redis cluster
- Configure proper logging
- Set up monitoring alerts
- Enable distributed tracing

## 🔄 **Development Workflow**

### **Real Services Only**
This service is designed for **production-ready development** with real services:

1. **No Mocks**: Only connects to real services - fails fast if dependencies unavailable
2. **Real Integration**: Always connects to actual Rust engine via gRPC
3. **Contract-First**: Well-defined gRPC interface ensures compatibility

### **Adding New Endpoints**
1. Add endpoint to `orchestrator.go`
2. Define request/response models in `models/`
3. Add tests
4. Update documentation

### **Performance Optimization**
- Use context assembler for parallel data gathering
- Implement caching for frequently accessed data
- Monitor metrics and optimize bottlenecks
- Use connection pooling for external services

## 📈 **Performance Targets**

- **Latency**: <50ms P99 for Flow 2 execution
- **Throughput**: >5,000 requests/second
- **Memory**: <128MB per instance
- **CPU**: <30% utilization under normal load

## 🤝 **Contributing**

1. Follow Go best practices
2. Add tests for new features
3. Update documentation
4. Use structured logging
5. Monitor performance impact

## 📚 **Related Documentation**

- [Flow 2 Greenfield Implementation](../FLOW2_GREENFIELD_MEDICATION_SERVICE.md)
- [Rust Recipe Engine](../rust-recipe-engine/README.md)
- [API Documentation](./docs/api.md)
- [Deployment Guide](./docs/deployment.md)

---

**Ready for parallel development with the Rust Recipe Engine!** 🚀
