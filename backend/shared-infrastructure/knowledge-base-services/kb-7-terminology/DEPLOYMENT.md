# KB-7 Terminology Service - Deployment Guide

This guide provides comprehensive instructions for deploying the KB-7 Terminology Service across different environments.

## 🎯 Deployment Overview

The KB-7 Terminology Service supports multiple deployment patterns:
- **Local Development**: Single instance with local dependencies
- **Docker Compose**: Containerized with orchestrated dependencies
- **Kubernetes**: Production-ready orchestration with scaling
- **Cloud Managed**: Azure/AWS with managed services

## 📋 Prerequisites

### System Requirements
- **CPU**: 4+ cores (8+ recommended for production)
- **Memory**: 8GB RAM minimum (16GB+ for production)
- **Storage**: 100GB+ for full terminology datasets
- **Network**: Stable internet connection for terminology updates

### Software Dependencies
- **Go**: 1.21 or later
- **PostgreSQL**: 13+ with extensions (uuid-ossp, pg_trgm)
- **Redis**: 6+ for caching
- **Docker**: 20.10+ (if using containers)

## 🔧 Environment Configuration

### Environment Variables

Create a `.env` file or set these environment variables:

```bash
# Server Configuration
PORT=8087
ENVIRONMENT=production
VERSION=1.0.0
LOG_LEVEL=info

# Database Configuration
DATABASE_URL=postgresql://kb_user:kb_password@localhost:5433/clinical_governance
MIGRATIONS_PATH=./migrations

# Cache Configuration
REDIS_URL=redis://localhost:6380/7

# Regional Support
SUPPORTED_REGIONS=US,EU,CA,AU
TERMINOLOGY_DB=clinical_governance

# GraphQL Configuration
GRAPHQL_ENDPOINT=/graphql
GRAPHQL_INTROSPECT=false
GRAPHQL_PLAYGROUND=false

# Federation Configuration
FEDERATION_ENABLED=true
GATEWAY_URL=http://localhost:4000/graphql

# Monitoring Configuration
METRICS_ENABLED=true
HEALTH_ENDPOINT=/health

# Security Configuration
API_RATE_LIMIT=1000
JWT_SECRET=your-jwt-secret-here
CORS_ORIGINS=https://your-frontend-domain.com
```

### Production Security Settings

```bash
# Disable development features
GRAPHQL_PLAYGROUND=false
GRAPHQL_INTROSPECT=false
DEBUG_MODE=false

# Enable security features
ENABLE_HTTPS=true
TLS_CERT_PATH=/etc/ssl/certs/kb7.crt
TLS_KEY_PATH=/etc/ssl/private/kb7.key
CORS_STRICT=true

# Authentication
JWT_SECRET=your-256-bit-secret
API_KEY_REQUIRED=true
```

## 🐳 Docker Deployment

### Single Container

1. **Build the Docker image**
   ```bash
   docker build -t kb-7-terminology:latest .
   ```

2. **Run with Docker**
   ```bash
   docker run -d \
     --name kb7-terminology \
     -p 8087:8087 \
     -e DATABASE_URL="postgresql://user:pass@host:5432/db" \
     -e REDIS_URL="redis://redis-host:6379/7" \
     -e ENVIRONMENT=production \
     --restart unless-stopped \
     kb-7-terminology:latest
   ```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  kb7-terminology:
    build: .
    ports:
      - "8087:8087"
    environment:
      - DATABASE_URL=postgresql://kb_user:kb_password@postgres:5432/clinical_governance
      - REDIS_URL=redis://redis:6379/7
      - ENVIRONMENT=production
    depends_on:
      - postgres
      - redis
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8087/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: clinical_governance
      POSTGRES_USER: kb_user
      POSTGRES_PASSWORD: kb_password
    ports:
      - "5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped
    command: redis-server --appendonly yes

volumes:
  postgres_data:
  redis_data:
```

Deploy with:
```bash
docker-compose up -d
```

## ☸️ Kubernetes Deployment

### Namespace and ConfigMap

```yaml
# kb7-namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kb7-terminology

---
# kb7-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kb7-config
  namespace: kb7-terminology
data:
  ENVIRONMENT: "production"
  LOG_LEVEL: "info"
  METRICS_ENABLED: "true"
  GRAPHQL_PLAYGROUND: "false"
  FEDERATION_ENABLED: "true"
```

### Secrets

```yaml
# kb7-secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: kb7-secrets
  namespace: kb7-terminology
type: Opaque
data:
  database-url: cG9zdGdyZXNxbDovL3VzZXI6cGFzc0Bob3N0OjU0MzIvZGI=  # base64 encoded
  redis-url: cmVkaXM6Ly9yZWRpcy1ob3N0OjYzNzkvNw==  # base64 encoded
  jwt-secret: eW91ci1qd3Qtc2VjcmV0LWhlcmU=  # base64 encoded
```

### Deployment

```yaml
# kb7-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kb7-terminology
  namespace: kb7-terminology
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: kb7-terminology
  template:
    metadata:
      labels:
        app: kb7-terminology
    spec:
      containers:
      - name: kb7-terminology
        image: kb-7-terminology:latest
        ports:
        - containerPort: 8087
          name: http
        envFrom:
        - configMapRef:
            name: kb7-config
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: kb7-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: kb7-secrets
              key: redis-url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: kb7-secrets
              key: jwt-secret
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8087
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8087
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
```

### Service and Ingress

```yaml
# kb7-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: kb7-terminology-service
  namespace: kb7-terminology
spec:
  selector:
    app: kb7-terminology
  ports:
  - port: 80
    targetPort: 8087
    name: http
  type: ClusterIP

---
# kb7-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kb7-terminology-ingress
  namespace: kb7-terminology
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/rate-limit: "1000"
spec:
  tls:
  - hosts:
    - api.hospital.com
    secretName: kb7-tls-secret
  rules:
  - host: api.hospital.com
    http:
      paths:
      - path: /kb7
        pathType: Prefix
        backend:
          service:
            name: kb7-terminology-service
            port:
              number: 80
```

Deploy to Kubernetes:
```bash
kubectl apply -f kb7-namespace.yaml
kubectl apply -f kb7-configmap.yaml
kubectl apply -f kb7-secrets.yaml
kubectl apply -f kb7-deployment.yaml
kubectl apply -f kb7-service.yaml
kubectl apply -f kb7-ingress.yaml
```

## 🌩️ Cloud Deployment

### Azure Container Instances

```bash
# Create resource group
az group create --name kb7-terminology-rg --location eastus

# Deploy container
az container create \
  --resource-group kb7-terminology-rg \
  --name kb7-terminology \
  --image kb-7-terminology:latest \
  --dns-name-label kb7-terminology \
  --ports 8087 \
  --environment-variables \
    ENVIRONMENT=production \
    LOG_LEVEL=info \
    METRICS_ENABLED=true \
  --secure-environment-variables \
    DATABASE_URL="postgresql://user:pass@host:5432/db" \
    REDIS_URL="redis://host:6379/7" \
    JWT_SECRET="your-secret" \
  --cpu 2 \
  --memory 4
```

### AWS ECS

Create `ecs-task-definition.json`:

```json
{
  "family": "kb7-terminology",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "kb7-terminology",
      "image": "kb-7-terminology:latest",
      "portMappings": [
        {
          "containerPort": 8087,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {"name": "ENVIRONMENT", "value": "production"},
        {"name": "LOG_LEVEL", "value": "info"},
        {"name": "METRICS_ENABLED", "value": "true"}
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:kb7/database-url"
        },
        {
          "name": "REDIS_URL", 
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:kb7/redis-url"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/kb7-terminology",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

## 🗄️ Database Setup

### PostgreSQL Configuration

1. **Create database and user**
   ```sql
   CREATE DATABASE clinical_governance;
   CREATE USER kb_user WITH ENCRYPTED PASSWORD 'kb_password';
   GRANT ALL PRIVILEGES ON DATABASE clinical_governance TO kb_user;
   
   -- Connect to the database
   \c clinical_governance;
   
   -- Create required extensions
   CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
   CREATE EXTENSION IF NOT EXISTS "pg_trgm";
   
   -- Grant schema privileges
   GRANT ALL ON SCHEMA public TO kb_user;
   ```

2. **Production PostgreSQL settings**
   ```ini
   # postgresql.conf
   max_connections = 200
   shared_buffers = 2GB
   effective_cache_size = 6GB
   maintenance_work_mem = 512MB
   checkpoint_completion_target = 0.9
   wal_buffers = 16MB
   default_statistics_target = 100
   random_page_cost = 1.1
   effective_io_concurrency = 200
   
   # Enable logging
   log_statement = 'mod'
   log_min_duration_statement = 1000
   ```

### Redis Configuration

```ini
# redis.conf
maxmemory 4gb
maxmemory-policy allkeys-lru
appendonly yes
appendfsync everysec
save 900 1
save 300 10
save 60 10000
```

## 📊 Monitoring Setup

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'kb7-terminology'
    static_configs:
      - targets: ['kb7-terminology:8087']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Grafana Dashboard

Import the dashboard from `monitoring/terminology-operational-dashboard.json` to visualize:
- Request rates and latency
- Database connection pool status
- Cache hit rates
- Error rates by endpoint
- System resource usage

## 🔍 Health Checks

### Application Health
```bash
curl http://localhost:8087/health
```

Expected response:
```json
{
  "status": "healthy",
  "service": "kb-7-terminology",
  "version": "1.0.0",
  "checks": {
    "database": {"status": "healthy", "response_time_ms": 2.1},
    "cache": {"status": "healthy", "response_time_ms": 0.8}
  }
}
```

### Database Health
```sql
-- Check connection and basic functionality
SELECT COUNT(*) FROM terminology_systems;

-- Verify indexes are being used
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM terminology_concepts 
WHERE search_terms @@ plainto_tsquery('paracetamol');
```

### Cache Health
```bash
redis-cli -h localhost -p 6380
> SELECT 7
> PING
> INFO memory
```

## 🚀 Performance Optimization

### Database Optimization

1. **Index Maintenance**
   ```sql
   -- Reindex terminology tables weekly
   REINDEX TABLE terminology_concepts;
   REINDEX TABLE terminology_systems;
   
   -- Update table statistics
   ANALYZE terminology_concepts;
   ANALYZE concept_mappings;
   ```

2. **Connection Pool Tuning**
   ```go
   // Adjust in internal/database/connection.go
   db.SetMaxOpenConns(50)     // Production: 50-100
   db.SetMaxIdleConns(10)     // Production: 10-20
   db.SetConnMaxLifetime(5 * time.Minute)
   ```

### Cache Optimization

```bash
# Redis memory optimization
CONFIG SET maxmemory-policy allkeys-lru
CONFIG SET timeout 300

# Monitor cache performance
INFO stats
```

## 📝 Logging Configuration

### Structured Logging
```json
{
  "timestamp": "2023-01-15T10:30:00Z",
  "level": "info",
  "service": "kb-7-terminology",
  "method": "GET",
  "path": "/v1/concepts/snomed/387517004",
  "status": 200,
  "duration": "3.2ms",
  "client_ip": "192.168.1.100"
}
```

### Log Aggregation
For production, configure log shipping to:
- **ELK Stack**: Elasticsearch, Logstash, Kibana
- **Splunk**: For enterprise log management
- **DataDog**: For cloud-native monitoring
- **Azure Monitor**: For Azure deployments

## 🔐 Security Hardening

### SSL/TLS Configuration
```bash
# Generate certificates
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout kb7.key -out kb7.crt \
  -subj "/CN=kb7-terminology.hospital.com"

# Set environment variables
export TLS_CERT_PATH="/etc/ssl/certs/kb7.crt"
export TLS_KEY_PATH="/etc/ssl/private/kb7.key"
export ENABLE_HTTPS="true"
```

### Network Security
- Configure firewall rules to only allow necessary ports
- Use VPC/VNet isolation in cloud deployments
- Implement network policies in Kubernetes
- Enable connection encryption for database and cache

### Access Control
```bash
# API rate limiting
API_RATE_LIMIT=1000

# IP restrictions
ALLOWED_IPS="10.0.0.0/8,192.168.0.0/16"

# CORS configuration
CORS_ORIGINS="https://clinical-app.hospital.com"
```

## 🔄 Backup and Recovery

### Database Backup
```bash
# Daily backup script
#!/bin/bash
DATE=$(date +%Y%m%d)
pg_dump -h localhost -U kb_user clinical_governance > "backup_$DATE.sql"

# Retention: keep last 30 days
find /backups -name "backup_*.sql" -mtime +30 -delete
```

### Redis Backup
```bash
# Enable AOF and RDB persistence
redis-cli CONFIG SET save "900 1 300 10 60 10000"
redis-cli CONFIG SET appendonly yes

# Manual backup
redis-cli --rdb /backup/redis_backup.rdb
```

### Disaster Recovery
1. **Recovery Time Objective (RTO)**: < 4 hours
2. **Recovery Point Objective (RPO)**: < 1 hour
3. **Backup Strategy**: Daily full + hourly incremental
4. **Failover**: Automated with health checks

## 📞 Troubleshooting

### Common Deployment Issues

1. **Service Won't Start**
   ```bash
   # Check logs
   docker logs kb7-terminology
   kubectl logs -f deployment/kb7-terminology -n kb7-terminology
   
   # Verify environment variables
   env | grep -E "(DATABASE_URL|REDIS_URL)"
   ```

2. **Database Connection Issues**
   ```bash
   # Test connection
   psql "postgresql://kb_user:kb_password@localhost:5433/clinical_governance"
   
   # Check PostgreSQL logs
   tail -f /var/log/postgresql/postgresql-15-main.log
   ```

3. **Performance Issues**
   ```bash
   # Monitor system resources
   htop
   iostat -x 1
   
   # Check application metrics
   curl http://localhost:8087/metrics
   ```

## 📋 Deployment Checklist

### Pre-Deployment
- [ ] Environment variables configured
- [ ] Database initialized and migrated
- [ ] Cache configured and accessible
- [ ] SSL certificates installed
- [ ] Firewall rules configured
- [ ] Monitoring stack deployed

### Post-Deployment
- [ ] Health checks passing
- [ ] Metrics collection working
- [ ] Log aggregation configured
- [ ] Backup procedures tested
- [ ] Performance baselines established
- [ ] Security scan completed
- [ ] Documentation updated

### Production Readiness
- [ ] Load testing completed
- [ ] Disaster recovery tested
- [ ] Monitoring alerts configured
- [ ] Runbook documentation complete
- [ ] On-call procedures defined
- [ ] Change management process in place

For additional support, contact the Clinical Platform Team at clinical-platform@hospital.com.