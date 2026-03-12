# KB-6 Formulary Management Service - Deployment Guide

## Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Elasticsearch 8+ (optional but recommended for semantic search)

### 🚀 One-Command Deployment
```bash
# Clone and start everything
git clone <repository>
cd kb-6-formulary
make docker-up && make run
```

## Detailed Deployment Steps

### 1. Infrastructure Setup

#### Docker Infrastructure (Recommended)
```bash
# Start all infrastructure services
make docker-up

# Verify infrastructure health
make docker-health

# View infrastructure logs
make docker-logs
```

This starts:
- PostgreSQL (port 5433)
- Redis (port 6380) 
- Elasticsearch (port 9200)
- Grafana (port 3000)
- Prometheus (port 9090)

#### Manual Infrastructure Setup
```bash
# PostgreSQL
createdb kb6_formulary
psql -d kb6_formulary -f migrations/001_initial_schema.sql

# Redis
redis-server --port 6379

# Elasticsearch (optional)
./elasticsearch-8.x.x/bin/elasticsearch
```

### 2. Database Initialization

```bash
# Run database migrations
make migrate

# Load mock data for development
make load-mock-data

# Verify database setup
make db-health
```

### 3. Service Configuration

#### Environment Configuration
```bash
# Copy example configuration
cp config/config.example.yaml config/config.yaml

# Edit configuration for your environment
nano config/config.yaml
```

#### Key Configuration Sections
```yaml
server:
  port: "8086"          # gRPC port (HTTP will be 8087)
  environment: "production"

database:
  host: "localhost"
  port: "5433"
  database: "kb6_formulary"
  username: "postgres"
  password: "${DB_PASSWORD}"

redis:
  address: "localhost:6380"
  database: 1
  password: "${REDIS_PASSWORD}"

elasticsearch:
  enabled: true
  addresses: ["http://localhost:9200"]
  username: "${ES_USERNAME}"
  password: "${ES_PASSWORD}"

cost_analysis:
  max_alternatives_per_drug: 10
  cache_ttl_minutes: 15
  semantic_search_enabled: true
  optimization_strategies:
    - "enhanced_generic"
    - "therapeutic"
    - "tier_optimized"
    - "semantic_match"
```

### 4. Build and Run Service

#### Development Mode
```bash
# Build the service
go build -o bin/kb6-formulary

# Run with live reload (requires air)
air

# Or run directly
./bin/kb6-formulary
```

#### Production Mode
```bash
# Build optimized binary
make build-production

# Run with production configuration
KB6_CONFIG_FILE=config/production.yaml ./bin/kb6-formulary
```

## Production Deployment

### 🐳 Docker Deployment

#### Single Container
```dockerfile
# Use provided Dockerfile
docker build -t kb6-formulary:latest .
docker run -p 8086:8086 -p 8087:8087 \
  -e DB_HOST=your-db-host \
  -e REDIS_URL=your-redis-url \
  kb6-formulary:latest
```

#### Docker Compose (Full Stack)
```yaml
# Use provided docker-compose.production.yml
docker-compose -f docker-compose.production.yml up -d
```

### ☸️ Kubernetes Deployment

#### Helm Chart
```bash
# Install using provided Helm chart
helm install kb6-formulary ./charts/kb6-formulary \
  --set image.tag=v1.0.0 \
  --set database.host=your-postgres \
  --set redis.host=your-redis
```

#### Manual Kubernetes
```bash
# Apply Kubernetes manifests
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### 🌐 Load Balancer Configuration

#### NGINX Configuration
```nginx
upstream kb6_formulary_grpc {
    server kb6-formulary-1:8086;
    server kb6-formulary-2:8086;
    server kb6-formulary-3:8086;
}

upstream kb6_formulary_http {
    server kb6-formulary-1:8087;
    server kb6-formulary-2:8087;
    server kb6-formulary-3:8087;
}

# gRPC endpoint
server {
    listen 443 http2 ssl;
    server_name kb6-grpc.yourdomain.com;
    
    location / {
        grpc_pass grpc://kb6_formulary_grpc;
        grpc_set_header X-Real-IP $remote_addr;
    }
}

# REST API endpoint
server {
    listen 443 ssl;
    server_name kb6-api.yourdomain.com;
    
    location / {
        proxy_pass http://kb6_formulary_http;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Health Checks and Monitoring

### 🏥 Health Check Endpoints

```bash
# Global health check
curl http://localhost:8087/health

# Formulary service specific
curl http://localhost:8087/health/formulary

# Inventory service specific  
curl http://localhost:8087/health/inventory
```

### 📊 Metrics and Observability

#### Prometheus Metrics
```bash
# Metrics endpoint (if enabled)
curl http://localhost:8087/metrics
```

Key metrics:
- `kb6_formulary_requests_total`
- `kb6_formulary_request_duration_seconds`
- `kb6_formulary_cache_hit_rate`
- `kb6_formulary_cost_analysis_duration_seconds`
- `kb6_formulary_alternatives_found_total`

#### Grafana Dashboard
```bash
# Access Grafana (if using Docker)
http://localhost:3000
# Login: admin/admin

# Import provided dashboard
grafana/kb6-formulary-dashboard.json
```

### 🚨 Alerting Configuration

#### Prometheus Alerting Rules
```yaml
groups:
  - name: kb6-formulary
    rules:
      - alert: KB6FormularyDown
        expr: up{job="kb6-formulary"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          description: "KB-6 Formulary service is down"
          
      - alert: KB6FormularyHighLatency
        expr: histogram_quantile(0.95, kb6_formulary_request_duration_seconds_bucket) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          description: "KB-6 Formulary 95th percentile latency is above 100ms"
          
      - alert: KB6FormularyCacheHitRateLow
        expr: kb6_formulary_cache_hit_rate < 0.8
        for: 15m
        labels:
          severity: warning
        annotations:
          description: "KB-6 Formulary cache hit rate is below 80%"
```

## Security Configuration

### 🔐 Authentication & Authorization

#### JWT Token Configuration
```yaml
auth:
  jwt_secret: "${JWT_SECRET}"
  token_expiry: "24h"
  required_scopes:
    - "formulary:read"
    - "formulary:cost-analysis"
```

#### API Rate Limiting
```yaml
rate_limiting:
  enabled: true
  requests_per_minute: 100
  burst_size: 20
  cleanup_interval: "1m"
```

### 🛡️ Network Security

#### Firewall Rules
```bash
# Allow gRPC traffic
ufw allow 8086/tcp

# Allow HTTP REST API traffic  
ufw allow 8087/tcp

# Restrict database access to service subnet
ufw allow from 10.0.1.0/24 to any port 5432
```

#### TLS Configuration
```yaml
tls:
  enabled: true
  cert_file: "/etc/ssl/certs/kb6-formulary.crt"
  key_file: "/etc/ssl/private/kb6-formulary.key"
  min_version: "1.2"
```

## Backup and Recovery

### 💾 Database Backup
```bash
# Daily backup script
#!/bin/bash
pg_dump -h localhost -p 5433 -U postgres kb6_formulary \
  > backups/kb6_formulary_$(date +%Y%m%d).sql

# Compress and upload to S3
gzip backups/kb6_formulary_$(date +%Y%m%d).sql
aws s3 cp backups/kb6_formulary_$(date +%Y%m%d).sql.gz \
  s3://your-backup-bucket/kb6-formulary/
```

### 🔄 Recovery Procedures
```bash
# Restore from backup
createdb kb6_formulary_restored
psql -d kb6_formulary_restored -f backups/kb6_formulary_20250903.sql

# Verify data integrity
make verify-data-integrity

# Switch to restored database
# Update configuration and restart service
```

## Troubleshooting

### 🔍 Common Issues

#### Service Won't Start
```bash
# Check logs
docker-compose logs kb6-formulary

# Verify configuration
make verify-config

# Check port conflicts
netstat -tlnp | grep 808[67]
```

#### Database Connection Issues
```bash
# Test database connectivity
make test-db-connection

# Check database health
psql -h localhost -p 5433 -U postgres -c "SELECT version();"
```

#### Cache Performance Issues
```bash
# Monitor Redis performance
redis-cli --latency-history -h localhost -p 6380

# Check cache hit rates
curl http://localhost:8087/metrics | grep cache_hit_rate
```

#### Elasticsearch Issues
```bash
# Check Elasticsearch health
curl http://localhost:9200/_cluster/health

# Rebuild search indices
make rebuild-search-indices
```

### 🔧 Performance Tuning

#### Database Optimization
```sql
-- Index optimization for cost analysis
CREATE INDEX CONCURRENTLY idx_formulary_entries_cost_analysis 
ON formulary_entries (payer_id, plan_id, plan_year, tier);

CREATE INDEX CONCURRENTLY idx_generic_equivalents_lookup
ON generic_equivalents (brand_rxnorm, bioequivalence_rating, cost_ratio);

-- Update table statistics
ANALYZE formulary_entries;
ANALYZE generic_equivalents;
ANALYZE therapeutic_alternatives;
```

#### Cache Optimization
```yaml
redis:
  max_memory: "2gb"
  max_memory_policy: "allkeys-lru"
  cache_ttl_minutes: 15
  connection_pool_size: 20
```

### 📞 Support and Maintenance

#### Log Levels
```yaml
logging:
  level: "info"          # debug, info, warn, error
  format: "json"         # json, text
  output: "/var/log/kb6-formulary.log"
```

#### Maintenance Commands
```bash
# Graceful restart
make restart

# Update formulary data
make update-formulary-data

# Rebuild caches
make rebuild-caches

# Database maintenance
make db-maintenance
```

## API Testing

### 🧪 Test Suite
```bash
# Run all tests
make test-all

# Run cost analysis specific tests
make test-cost-analysis

# Run integration tests
make test-integration

# Performance tests
make test-performance
```

### 📋 Test Data
```bash
# Load test data
make load-test-data

# Generate test scenarios
make generate-test-scenarios

# Cleanup test data
make clean-test-data
```

---

**Deployment Status**: ✅ **Production Ready**
- Comprehensive infrastructure setup
- Security hardened configuration
- Full monitoring and alerting
- Backup and recovery procedures
- Performance optimization guides
- Complete troubleshooting documentation

For support: Create issues at `project-repository/issues` or contact the development team.