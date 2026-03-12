# Safety Gateway Platform - Phase 2 Deployment Guide

## Advanced Orchestration Enhancement Deployment

This guide covers the deployment of Safety Gateway Platform Phase 2, which introduces advanced orchestration, intelligent batch processing, adaptive load balancing, and comprehensive metrics collection.

## 🚀 Phase 2 Features

### Core Enhancements
- **Advanced Orchestration Engine**: Intelligent routing with adaptive load balancing
- **Enhanced Batch Processing**: Patient-grouped, snapshot-optimized, and parallel-direct strategies
- **Intelligent Routing**: Rule-based routing with priority evaluation and fallback chains
- **Comprehensive Metrics**: Performance, load, routing, batch, and snapshot metrics
- **Performance Optimization**: Memory pooling, connection pooling, and adaptive throttling

### New API Endpoints
- `/api/v1/batch/validate` - Batch validation and submission
- `/api/v1/batch/{batch_id}/status` - Batch status monitoring
- `/api/v1/orchestration/stats` - Real-time orchestration statistics
- `/api/v1/orchestration/metrics` - Detailed performance metrics
- `/api/v1/health/orchestration` - Orchestration-specific health checks

## 📋 Prerequisites

### System Requirements
- **Memory**: 2GB+ per pod (Kubernetes) / 4GB+ total (Docker Compose)
- **CPU**: 2+ cores recommended for production
- **Storage**: 5GB+ for logs, metrics, and cache
- **Network**: Low-latency network for optimal performance

### Required Tools
- Docker 20.10+
- Kubernetes 1.21+ (for K8s deployment)
- kubectl configured for target cluster
- Helm 3.0+ (for advanced deployments)
- Docker Compose 2.0+ (for local deployment)

### Dependencies
- **TimescaleDB**: For time-series metrics storage
- **Redis**: For caching and session management
- **Kafka**: For event streaming and batch processing
- **Clinical Reasoning Service (CAE)**: Must be running on port 8027

## 🛠 Deployment Options

### Option 1: Kubernetes Deployment (Recommended for Production)

#### Quick Deployment
```bash
# Deploy to staging environment
./scripts/deploy-phase2.sh staging kubernetes

# Deploy to production environment
./scripts/deploy-phase2.sh production kubernetes
```

#### Manual Deployment Steps
```bash
# 1. Build and push image
docker build -t safety-gateway-platform:v2.0.0-phase2 .
docker tag safety-gateway-platform:v2.0.0-phase2 your-registry/safety-gateway-platform:v2.0.0-phase2
docker push your-registry/safety-gateway-platform:v2.0.0-phase2

# 2. Create namespace
kubectl create namespace safety-gateway

# 3. Apply configurations
kubectl apply -f devops/k8s/overlays/phase2/safety-gateway-phase2-configmap.yaml -n safety-gateway
kubectl apply -f devops/k8s/overlays/phase2/safety-gateway-phase2-deployment.yaml -n safety-gateway

# 4. Verify deployment
kubectl get pods -n safety-gateway -l version=v2.0.0
kubectl logs -n safety-gateway deployment/safety-gateway-phase2
```

#### Kubernetes Configuration Files
- `devops/k8s/overlays/phase2/safety-gateway-phase2-deployment.yaml` - Main deployment
- `devops/k8s/overlays/phase2/safety-gateway-phase2-configmap.yaml` - Configuration
- `devops/k8s/base/safety-gateway-deployment.yaml` - Base deployment (for reference)

### Option 2: Docker Compose Deployment (Development/Testing)

#### Quick Start
```bash
# Start all services for development
./scripts/deploy-phase2.sh development docker-compose

# Or manually with Docker Compose
docker-compose -f docker-compose.phase2.yml up -d

# With development utilities
docker-compose -f docker-compose.phase2.yml --profile development up -d

# With testing profile
docker-compose -f docker-compose.phase2.yml --profile testing up -d
```

#### Service Endpoints (Docker Compose)
- **Safety Gateway API**: http://localhost:8030
- **Batch Processing API**: http://localhost:8031
- **Health Check**: http://localhost:8032/health
- **Orchestration Management**: http://localhost:8034
- **Metrics**: http://localhost:9091/metrics
- **Grafana Dashboard**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Adminer (DB Admin)**: http://localhost:8080
- **Kafka UI**: http://localhost:8081
- **Redis UI**: http://localhost:8082

## ⚙️ Configuration

### Environment-Specific Overrides

#### Production Configuration
```yaml
advanced_orchestration:
  load_balancing:
    strategy: "adaptive"
    performance_window_size: 500
  batch_processing:
    max_batch_size: 50
    worker_pool_size: 20
  metrics:
    metrics_interval: "10s"
    export_prometheus: true
logging:
  level: "warn"
security:
  rate_limiting:
    requests_per_second: 2000
    burst_size: 3000
```

#### Staging Configuration
```yaml
advanced_orchestration:
  load_balancing:
    strategy: "least_loaded"
  batch_processing:
    max_batch_size: 30
    worker_pool_size: 15
  metrics:
    metrics_interval: "5s"
logging:
  level: "info"
```

#### Development Configuration
```yaml
advanced_orchestration:
  load_balancing:
    strategy: "round_robin"
  batch_processing:
    max_batch_size: 10
    worker_pool_size: 5
  metrics:
    metrics_interval: "2s"
logging:
  level: "debug"
  enable_colors: true
monitoring:
  pprof_enabled: true
```

### Environment Variables

#### Required Variables
```bash
# Database
POSTGRES_HOST=your-postgres-host
POSTGRES_USER=safety_user
POSTGRES_PASSWORD=your-secure-password
POSTGRES_DB=safety_gateway_phase2

# Redis
REDIS_HOST=your-redis-host
REDIS_PASSWORD=your-redis-password

# Kafka
KAFKA_BOOTSTRAP_SERVERS=your-kafka-brokers
KAFKA_USERNAME=your-kafka-user
KAFKA_PASSWORD=your-kafka-password

# External Services
CAE_SERVICE_HOST=your-cae-service-host

# Security
JWT_SECRET=your-jwt-secret-256-bits-minimum
```

#### Optional Variables
```bash
# Performance Tuning
SGP_MAX_CONCURRENT_REQUESTS=1000
SGP_BATCH_MAX_SIZE=50
SGP_LOAD_BALANCING_STRATEGY=adaptive

# Monitoring
PROMETHEUS_PUSHGATEWAY_URL=your-pushgateway-url
GRAFANA_PASSWORD=your-grafana-password

# Feature Flags
SGP_ENABLE_ADVANCED_ORCHESTRATION=true
SGP_ENABLE_BATCH_PROCESSING=true
SGP_ENABLE_INTELLIGENT_ROUTING=true
```

## 🔍 Monitoring and Observability

### Metrics Endpoints
- **Prometheus Metrics**: `/metrics` (port 9090)
- **Health Checks**: `/health`, `/health/detailed`, `/health/orchestration`
- **Orchestration Stats**: `/api/v1/orchestration/stats`
- **Real-time Metrics**: `/api/v1/orchestration/metrics`

### Key Metrics to Monitor

#### Performance Metrics
- `safety_gateway_requests_per_second` - Request throughput
- `safety_gateway_response_time_histogram` - Response time distribution
- `safety_gateway_error_rate` - Error rate percentage
- `safety_gateway_orchestration_latency` - Orchestration processing time

#### Batch Processing Metrics
- `safety_gateway_batch_queue_size` - Current batch queue size
- `safety_gateway_batch_processing_time` - Batch processing duration
- `safety_gateway_batch_success_rate` - Batch success percentage
- `safety_gateway_batch_efficiency_ratio` - Processing efficiency

#### Load Balancing Metrics
- `safety_gateway_engine_load_score` - Engine load distribution
- `safety_gateway_routing_decisions_total` - Routing decision counts
- `safety_gateway_engine_health_score` - Engine health status

### Alerting Rules

#### Critical Alerts
```yaml
- alert: SafetyGatewayDown
  expr: up{job="safety-gateway-phase2"} == 0
  for: 1m
  severity: critical

- alert: HighErrorRate
  expr: rate(safety_gateway_errors_total[5m]) > 0.1
  for: 2m
  severity: critical

- alert: HighLatency
  expr: histogram_quantile(0.95, safety_gateway_response_time_histogram) > 1
  for: 5m
  severity: warning
```

### Grafana Dashboards
Pre-built dashboards are available in `monitoring/grafana/dashboards/`:
- **Phase 2 Overview Dashboard** - High-level metrics and KPIs
- **Orchestration Performance Dashboard** - Advanced orchestration metrics
- **Batch Processing Dashboard** - Batch processing analytics
- **Engine Health Dashboard** - Engine performance and health metrics

## 🧪 Testing

### Integration Tests
```bash
# Run Phase 2 integration tests
go test -v ./tests/integration/phase2_orchestration_test.go

# Run with race detection
go test -v -race ./tests/integration/phase2_orchestration_test.go

# Run specific test suites
go test -v ./tests/integration/phase2_orchestration_test.go -run TestAdvancedOrchestrationProcessing
go test -v ./tests/integration/phase2_orchestration_test.go -run TestBatchProcessingIntegration
```

### Load Testing
```bash
# Run load tests using k6
k6 run tests/load/phase2-load-test.js

# High load test
k6 run tests/load/phase2-load-test.js --vus 50 --duration 10m

# Stress test
k6 run tests/load/phase2-stress-test.js --vus 100 --duration 5m
```

### API Testing
```bash
# Test batch processing endpoint
curl -X POST http://localhost:8031/api/v1/batch/validate \
  -H "Content-Type: application/json" \
  -d @tests/fixtures/sample-batch.json

# Test orchestration stats
curl http://localhost:8034/api/v1/orchestration/stats

# Test health endpoints
curl http://localhost:8032/health/orchestration
```

## 🔧 Troubleshooting

### Common Issues

#### 1. Deployment Fails to Start
```bash
# Check pod logs
kubectl logs -n safety-gateway deployment/safety-gateway-phase2

# Check events
kubectl get events -n safety-gateway --sort-by='.lastTimestamp'

# Check resource usage
kubectl top pods -n safety-gateway
```

#### 2. High Memory Usage
```bash
# Check memory metrics
curl http://localhost:9091/metrics | grep memory

# Adjust memory limits in deployment
kubectl patch deployment safety-gateway-phase2 -n safety-gateway -p '{"spec":{"template":{"spec":{"containers":[{"name":"safety-gateway","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
```

#### 3. Batch Processing Issues
```bash
# Check batch processor logs
kubectl logs -n safety-gateway deployment/safety-gateway-phase2 -c safety-gateway | grep batch

# Monitor batch queue size
curl http://localhost:8034/api/v1/orchestration/stats | jq '.batch_queue_size'

# Check batch processing configuration
kubectl get configmap safety-gateway-phase2-config -n safety-gateway -o yaml
```

#### 4. Load Balancing Problems
```bash
# Check engine health
curl http://localhost:8034/api/v1/orchestration/stats | jq '.engine_health'

# Monitor routing decisions
curl http://localhost:8034/api/v1/orchestration/metrics | jq '.routing'

# Check load balancing strategy
kubectl get configmap safety-gateway-phase2-config -n safety-gateway -o yaml | grep strategy
```

### Performance Tuning

#### CPU Optimization
```yaml
# In deployment configuration
resources:
  requests:
    cpu: "1000m"
  limits:
    cpu: "2000m"

# Environment variables
- name: SGP_MAX_CONCURRENT_REQUESTS
  value: "1000"
- name: SGP_GOROUTINE_POOL_SIZE
  value: "200"
```

#### Memory Optimization
```yaml
# In deployment configuration
resources:
  requests:
    memory: "1Gi"
  limits:
    memory: "2Gi"

# Environment variables
- name: SGP_MAX_MEMORY_MB
  value: "1536"
- name: SGP_ENABLE_MEMORY_OPTIMIZATION
  value: "true"
```

#### Network Optimization
```yaml
# Environment variables
- name: SGP_CONNECTION_POOL_SIZE
  value: "100"
- name: SGP_KEEP_ALIVE_TIMEOUT
  value: "60s"
- name: SGP_MAX_IDLE_CONNECTIONS
  value: "50"
```

## 🔄 Rollback Procedures

### Kubernetes Rollback
```bash
# Check rollout history
kubectl rollout history deployment/safety-gateway-phase2 -n safety-gateway

# Rollback to previous version
kubectl rollout undo deployment/safety-gateway-phase2 -n safety-gateway

# Rollback to specific revision
kubectl rollout undo deployment/safety-gateway-phase2 -n safety-gateway --to-revision=1
```

### Docker Compose Rollback
```bash
# Stop current deployment
docker-compose -f docker-compose.phase2.yml down

# Switch to previous image version
# Edit docker-compose.phase2.yml to use previous image tag

# Start with previous version
docker-compose -f docker-compose.phase2.yml up -d
```

## 📈 Scaling

### Horizontal Pod Autoscaler (HPA)
```yaml
# Already configured in deployment
spec:
  minReplicas: 5
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 60
  - type: Pods
    pods:
      metric:
        name: safety_gateway_requests_per_second
      target:
        type: AverageValue
        averageValue: "200"
```

### Manual Scaling
```bash
# Scale deployment
kubectl scale deployment safety-gateway-phase2 --replicas=10 -n safety-gateway

# Check scaling status
kubectl get hpa safety-gateway-phase2-hpa -n safety-gateway
```

## 🔐 Security Considerations

### Production Security Checklist
- [ ] Use strong, unique passwords for all services
- [ ] Enable TLS/SSL for all communications
- [ ] Configure proper RBAC in Kubernetes
- [ ] Use secrets management (Vault, K8s secrets)
- [ ] Enable network policies
- [ ] Configure security scanning in CI/CD
- [ ] Set up log aggregation and monitoring
- [ ] Regular security updates and patches

### Security Configuration
```yaml
security:
  enable_tls: true
  cert_file: "/etc/ssl/certs/safety-gateway.crt"
  key_file: "/etc/ssl/private/safety-gateway.key"
  min_tls_version: "1.2"
  rate_limiting:
    enabled: true
    requests_per_second: 2000
    burst_size: 3000
```

## 📞 Support

### Getting Help
- **Documentation**: This README and inline code documentation
- **Logs**: Use kubectl logs or docker-compose logs for troubleshooting
- **Monitoring**: Grafana dashboards for real-time insights
- **Metrics**: Prometheus endpoints for detailed metrics

### Contributing
1. Follow the existing code style and patterns
2. Add comprehensive tests for new features
3. Update documentation for any changes
4. Ensure all CI/CD checks pass

### Emergency Contacts
For production issues:
1. Check monitoring dashboards first
2. Review recent deployments and changes
3. Check system resource usage
4. Review application logs
5. Contact the on-call engineering team

---

## ✅ Deployment Checklist

### Pre-Deployment
- [ ] Verify prerequisites are installed
- [ ] Validate configuration files
- [ ] Check resource availability
- [ ] Backup existing deployment (if applicable)
- [ ] Review security settings

### Deployment
- [ ] Build and tag Phase 2 images
- [ ] Run database migrations
- [ ] Deploy configuration maps
- [ ] Deploy application
- [ ] Verify pod startup
- [ ] Check health endpoints

### Post-Deployment
- [ ] Run integration tests
- [ ] Verify Phase 2 endpoints
- [ ] Check monitoring dashboards
- [ ] Configure alerting
- [ ] Update documentation
- [ ] Notify stakeholders

### Validation
- [ ] Performance benchmarks meet targets
- [ ] All Phase 2 features are functional
- [ ] Monitoring and alerting are working
- [ ] Security scans pass
- [ ] Load testing successful

---

**Phase 2 Advanced Orchestration Enhancement** brings significant improvements to the Safety Gateway Platform's performance, scalability, and observability. Follow this guide carefully to ensure a successful deployment and optimal system performance.