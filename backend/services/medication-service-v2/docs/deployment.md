# Medication Service V2 Deployment Guide

Comprehensive deployment guide for the Go/Rust Medication Service V2 with Recipe & Snapshot architecture.

## Table of Contents
- [Deployment Overview](#deployment-overview)
- [Environment Configuration](#environment-configuration)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Migration Strategy](#migration-strategy)
- [Production Considerations](#production-considerations)
- [Monitoring & Observability](#monitoring--observability)
- [Backup & Recovery](#backup--recovery)
- [Rollback Procedures](#rollback-procedures)

## Deployment Overview

### Architecture Components
```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│  Load Balancer      │────│  Medication         │────│  External Services  │
│  (Nginx/HAProxy)    │    │  Service V2         │    │  (Context Gateway)  │
└─────────────────────┘    │  (Port 8005)        │    │  (Apollo Federation)│
                           └─────────────────────┘    └─────────────────────┘
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        │                             │                             │
┌───────────────┐              ┌──────────────┐              ┌──────────────┐
│  Flow2 Go     │              │ Rust Clinical│              │ Knowledge    │
│  Engine       │              │ Engine       │              │ Bases        │
│  (Port 8085)  │              │ (Port 8095)  │              │ (8086, 8089) │
└───────────────┘              └──────────────┘              └──────────────┘
        │                             │                             │
        └─────────────────────────────┼─────────────────────────────┘
                                      │
        ┌─────────────────────────────┼─────────────────────────────┐
        │                             │                             │
┌───────────────┐              ┌──────────────┐              ┌──────────────┐
│ PostgreSQL    │              │ Redis Cache  │              │ Monitoring   │
│ (Port 5434)   │              │ (Port 6381)  │              │ Stack        │
└───────────────┘              └──────────────┘              └──────────────┘
```

### Service Dependencies
- **PostgreSQL**: Primary database for medication data and proposals
- **Redis**: Caching layer for performance optimization
- **Context Gateway**: External service for clinical snapshots
- **Apollo Federation**: GraphQL gateway for knowledge base access
- **Monitoring Stack**: Prometheus, Grafana, Jaeger for observability

## Environment Configuration

### 1. Environment Files

Create `.env.production`:
```bash
# Service Configuration
SERVICE_NAME=medication-service-v2
SERVICE_VERSION=1.0.0
SERVICE_ENV=production
SERVICE_PORT=8005

# Database Configuration
DATABASE_HOST=postgres-v2
DATABASE_PORT=5432
DATABASE_NAME=medication_v2
DATABASE_USER=medication_user
DATABASE_PASSWORD=secure_password_here
DATABASE_SSL_MODE=require
DATABASE_MAX_CONNECTIONS=100
DATABASE_MAX_IDLE=20

# Redis Configuration
REDIS_URL=redis://redis-v2:6379
REDIS_MAX_CONNECTIONS=50
REDIS_TIMEOUT=5s

# External Service URLs
CONTEXT_GATEWAY_URL=https://context-gateway.clinical.internal
APOLLO_FEDERATION_URL=https://api-gateway.clinical.internal/graphql

# Component Service URLs
FLOW2_GO_ENGINE_URL=http://flow2-go-engine-v2:8085
RUST_CLINICAL_ENGINE_URL=http://rust-clinical-engine-v2:8095
KB_DRUG_RULES_URL=http://kb-drug-rules-v2:8086
KB_GUIDELINES_URL=http://kb-guidelines-v2:8089

# Security Configuration
JWT_SECRET_KEY=your_jwt_secret_key_here
API_KEY_SECRET=your_api_key_secret_here
ENCRYPTION_KEY=your_encryption_key_here

# Performance Configuration
MAX_CONCURRENT_REQUESTS=1000
REQUEST_TIMEOUT=30s
RECIPE_CACHE_TTL=10m
CALCULATION_TIMEOUT=30s

# Monitoring Configuration
METRICS_ENABLED=true
TRACING_ENABLED=true
LOG_LEVEL=info
JAEGER_ENDPOINT=http://jaeger-collector:14268/api/traces

# Health Check Configuration
HEALTH_CHECK_INTERVAL=30s
DEPENDENCY_TIMEOUT=5s
```

### 2. Configuration Validation

Create `scripts/validate-config.sh`:
```bash
#!/bin/bash
set -e

echo "Validating deployment configuration..."

# Check required environment variables
required_vars=(
    "SERVICE_NAME"
    "SERVICE_VERSION"
    "DATABASE_HOST"
    "DATABASE_USER"
    "DATABASE_PASSWORD"
    "REDIS_URL"
    "CONTEXT_GATEWAY_URL"
    "JWT_SECRET_KEY"
)

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "ERROR: Required environment variable $var is not set"
        exit 1
    fi
done

# Validate service connectivity
echo "Testing database connectivity..."
pg_isready -h $DATABASE_HOST -p $DATABASE_PORT -U $DATABASE_USER

echo "Testing Redis connectivity..."
redis-cli -u $REDIS_URL ping

echo "Testing external service connectivity..."
curl -f --max-time 10 $CONTEXT_GATEWAY_URL/health || echo "WARNING: Context Gateway not responding"

echo "Configuration validation completed successfully!"
```

## Docker Deployment

### 1. Production Docker Configuration

Create `docker-compose.production.yml`:
```yaml
version: '3.8'

services:
  medication-service-v2:
    build:
      context: .
      dockerfile: deployments/Dockerfile.production
      args:
        - VERSION=${SERVICE_VERSION}
    image: clinical-platform/medication-service-v2:${SERVICE_VERSION}
    container_name: medication-service-v2
    restart: unless-stopped
    ports:
      - "8005:8005"
      - "8006:8006"  # gRPC port
    environment:
      - SERVICE_ENV=production
    env_file:
      - .env.production
    volumes:
      - ./configs/production.yaml:/app/configs/service.yaml:ro
      - ./logs:/app/logs
    depends_on:
      postgres-v2:
        condition: service_healthy
      redis-v2:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8005/health/ready"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '1.5'
        reservations:
          memory: 1G
          cpus: '0.5'
    networks:
      - clinical-network
    labels:
      - "com.clinical.service=medication-service-v2"
      - "com.clinical.version=${SERVICE_VERSION}"

  flow2-go-engine-v2:
    build:
      context: ./flow2-go-engine-v2
      dockerfile: Dockerfile.production
    image: clinical-platform/flow2-go-engine-v2:${SERVICE_VERSION}
    container_name: flow2-go-engine-v2
    restart: unless-stopped
    ports:
      - "8085:8085"
    env_file:
      - .env.production
    depends_on:
      postgres-v2:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8085/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - clinical-network

  rust-clinical-engine-v2:
    build:
      context: ./flow2-rust-engine-v2
      dockerfile: Dockerfile.production
    image: clinical-platform/rust-clinical-engine-v2:${SERVICE_VERSION}
    container_name: rust-clinical-engine-v2
    restart: unless-stopped
    ports:
      - "8095:8095"
    env_file:
      - .env.production
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8095/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - clinical-network

  kb-drug-rules-v2:
    build:
      context: ./knowledge-bases-v2/kb-drug-rules
      dockerfile: Dockerfile.production
    image: clinical-platform/kb-drug-rules-v2:${SERVICE_VERSION}
    container_name: kb-drug-rules-v2
    restart: unless-stopped
    ports:
      - "8086:8086"
    env_file:
      - .env.production
    depends_on:
      postgres-v2:
        condition: service_healthy
    networks:
      - clinical-network

  kb-guidelines-v2:
    build:
      context: ./knowledge-bases-v2/kb-guideline-evidence
      dockerfile: Dockerfile.production
    image: clinical-platform/kb-guidelines-v2:${SERVICE_VERSION}
    container_name: kb-guidelines-v2
    restart: unless-stopped
    ports:
      - "8089:8089"
    env_file:
      - .env.production
    depends_on:
      postgres-v2:
        condition: service_healthy
    networks:
      - clinical-network

  postgres-v2:
    image: postgres:15-alpine
    container_name: postgres-v2
    restart: unless-stopped
    ports:
      - "5434:5432"
    environment:
      POSTGRES_DB: ${DATABASE_NAME}
      POSTGRES_USER: ${DATABASE_USER}
      POSTGRES_PASSWORD: ${DATABASE_PASSWORD}
      POSTGRES_INITDB_ARGS: "--auth-host=md5"
    volumes:
      - postgres_v2_data:/var/lib/postgresql/data
      - ./scripts/postgres-init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DATABASE_USER} -d ${DATABASE_NAME}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - clinical-network

  redis-v2:
    image: redis:7-alpine
    container_name: redis-v2
    restart: unless-stopped
    ports:
      - "6381:6379"
    volumes:
      - redis_v2_data:/data
      - ./configs/redis.conf:/usr/local/etc/redis/redis.conf:ro
    command: redis-server /usr/local/etc/redis/redis.conf
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - clinical-network

  nginx-v2:
    image: nginx:alpine
    container_name: nginx-v2
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./configs/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
      - ./logs/nginx:/var/log/nginx
    depends_on:
      - medication-service-v2
    networks:
      - clinical-network

networks:
  clinical-network:
    driver: bridge

volumes:
  postgres_v2_data:
    driver: local
  redis_v2_data:
    driver: local
```

### 2. Production Dockerfile

Create `deployments/Dockerfile.production`:
```dockerfile
# Build stage for Go application
FROM golang:1.21-alpine AS go-builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev}" \
    -o medication-service-v2 \
    ./cmd/medication-server

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    postgresql-client \
    redis

# Create non-root user
RUN addgroup -g 1000 appgroup && \
    adduser -D -s /bin/sh -u 1000 -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /app/medication-service-v2 ./medication-service-v2

# Copy configuration files
COPY --chown=appuser:appgroup configs/ ./configs/
COPY --chown=appuser:appgroup migrations/ ./migrations/

# Create log directory
RUN mkdir -p logs && chown appuser:appgroup logs

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8005 8006

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=40s \
    CMD curl -f http://localhost:8005/health/ready || exit 1

# Run the application
CMD ["./medication-service-v2"]
```

### 3. Deployment Scripts

Create `scripts/deploy.sh`:
```bash
#!/bin/bash
set -e

# Configuration
SERVICE_NAME="medication-service-v2"
DOCKER_COMPOSE_FILE="docker-compose.production.yml"
BACKUP_DIR="/backup/medication-service-v2"
LOG_FILE="/var/log/${SERVICE_NAME}-deploy.log"

# Logging function
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a $LOG_FILE
}

# Pre-deployment checks
pre_deployment_checks() {
    log "Starting pre-deployment checks..."
    
    # Check if required files exist
    if [[ ! -f "$DOCKER_COMPOSE_FILE" ]]; then
        log "ERROR: Docker compose file not found: $DOCKER_COMPOSE_FILE"
        exit 1
    fi
    
    # Validate configuration
    ./scripts/validate-config.sh
    
    # Check disk space
    AVAILABLE_SPACE=$(df / | awk 'NR==2 {print $4}')
    if [[ $AVAILABLE_SPACE -lt 5000000 ]]; then  # 5GB in KB
        log "ERROR: Insufficient disk space. Available: ${AVAILABLE_SPACE}KB"
        exit 1
    fi
    
    log "Pre-deployment checks completed successfully"
}

# Create backup
create_backup() {
    log "Creating backup..."
    
    # Create backup directory
    mkdir -p "$BACKUP_DIR/$(date +%Y%m%d_%H%M%S)"
    CURRENT_BACKUP="$BACKUP_DIR/$(date +%Y%m%d_%H%M%S)"
    
    # Backup database
    if docker ps | grep -q postgres-v2; then
        log "Backing up database..."
        docker exec postgres-v2 pg_dump -U $DATABASE_USER -d $DATABASE_NAME > "$CURRENT_BACKUP/database.sql"
    fi
    
    # Backup configuration
    cp -r configs "$CURRENT_BACKUP/"
    
    log "Backup created at: $CURRENT_BACKUP"
}

# Deploy new version
deploy_new_version() {
    log "Deploying new version..."
    
    # Pull latest images
    log "Pulling latest images..."
    docker-compose -f $DOCKER_COMPOSE_FILE pull
    
    # Stop services gracefully
    log "Stopping services..."
    docker-compose -f $DOCKER_COMPOSE_FILE stop medication-service-v2
    
    # Run database migrations
    log "Running database migrations..."
    docker-compose -f $DOCKER_COMPOSE_FILE run --rm medication-service-v2 \
        ./medication-service-v2 -migrate-only
    
    # Start services
    log "Starting services..."
    docker-compose -f $DOCKER_COMPOSE_FILE up -d
    
    # Wait for services to be healthy
    log "Waiting for services to be healthy..."
    sleep 30
    
    # Health check
    if ! curl -f --max-time 30 http://localhost:8005/health/ready; then
        log "ERROR: Health check failed after deployment"
        return 1
    fi
    
    log "Deployment completed successfully"
}

# Post-deployment verification
post_deployment_verification() {
    log "Starting post-deployment verification..."
    
    # Test API endpoints
    if curl -f http://localhost:8005/health/live; then
        log "✓ Liveness check passed"
    else
        log "✗ Liveness check failed"
        return 1
    fi
    
    if curl -f http://localhost:8005/health/ready; then
        log "✓ Readiness check passed"  
    else
        log "✗ Readiness check failed"
        return 1
    fi
    
    if curl -f http://localhost:8005/health/deps; then
        log "✓ Dependency check passed"
    else
        log "✗ Dependency check failed"
        return 1
    fi
    
    # Test core functionality
    log "Testing core functionality..."
    # Add specific API tests here
    
    log "Post-deployment verification completed successfully"
}

# Main deployment flow
main() {
    log "Starting deployment of $SERVICE_NAME"
    
    pre_deployment_checks
    create_backup
    
    if deploy_new_version; then
        if post_deployment_verification; then
            log "Deployment successful!"
        else
            log "Post-deployment verification failed, consider rollback"
            exit 1
        fi
    else
        log "Deployment failed, rolling back..."
        rollback_deployment
        exit 1
    fi
}

# Rollback function
rollback_deployment() {
    log "Starting rollback..."
    
    # Get latest backup
    LATEST_BACKUP=$(ls -1t $BACKUP_DIR | head -1)
    
    if [[ -z "$LATEST_BACKUP" ]]; then
        log "ERROR: No backup found for rollback"
        exit 1
    fi
    
    log "Rolling back to backup: $LATEST_BACKUP"
    
    # Stop current services
    docker-compose -f $DOCKER_COMPOSE_FILE down
    
    # Restore database
    if [[ -f "$BACKUP_DIR/$LATEST_BACKUP/database.sql" ]]; then
        docker-compose -f $DOCKER_COMPOSE_FILE up -d postgres-v2
        sleep 10
        cat "$BACKUP_DIR/$LATEST_BACKUP/database.sql" | \
            docker exec -i postgres-v2 psql -U $DATABASE_USER -d $DATABASE_NAME
    fi
    
    # Restore configuration  
    cp -r "$BACKUP_DIR/$LATEST_BACKUP/configs/"* configs/
    
    # Start services with previous configuration
    docker-compose -f $DOCKER_COMPOSE_FILE up -d
    
    log "Rollback completed"
}

# Execute main function
main "$@"
```

## Kubernetes Deployment

### 1. Kubernetes Manifests

Create `deployments/k8s/namespace.yaml`:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: medication-service-v2
  labels:
    name: medication-service-v2
    environment: production
```

Create `deployments/k8s/configmap.yaml`:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: medication-service-v2-config
  namespace: medication-service-v2
data:
  service.yaml: |
    service:
      name: medication-service-v2
      version: "1.0.0"
      port: 8005
      environment: production

    database:
      host: postgres-v2-service
      port: 5432
      name: medication_v2
      user: medication_user
      ssl_mode: require
      max_connections: 100
      max_idle_connections: 20

    redis:
      url: redis://redis-v2-service:6379
      max_connections: 50
      timeout: 5s

    recipe_resolver:
      cache_enabled: true
      cache_ttl: 10m
      default_recipe_ttl: 1h

    clinical_engine:
      rust_engine_url: http://rust-clinical-engine-v2-service:8095
      timeout: 30s
      max_retries: 3

    context_gateway:
      base_url: http://context-gateway-service:8020
      timeout: 15s
      max_retries: 2

    knowledge_bases:
      drug_rules_url: http://kb-drug-rules-v2-service:8086
      guidelines_url: http://kb-guidelines-v2-service:8089
      apollo_federation_url: http://apollo-federation-service:4000/graphql
      cache_ttl: 5m

    monitoring:
      metrics_enabled: true
      tracing_enabled: true
      log_level: info
      health_check_interval: 30s
```

Create `deployments/k8s/secrets.yaml`:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: medication-service-v2-secrets
  namespace: medication-service-v2
type: Opaque
data:
  database-password: <base64-encoded-password>
  jwt-secret-key: <base64-encoded-jwt-secret>
  api-key-secret: <base64-encoded-api-secret>
  encryption-key: <base64-encoded-encryption-key>
```

Create `deployments/k8s/deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: medication-service-v2
  namespace: medication-service-v2
  labels:
    app: medication-service-v2
    version: "1.0.0"
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: medication-service-v2
  template:
    metadata:
      labels:
        app: medication-service-v2
        version: "1.0.0"
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8005"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: medication-service-v2
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: medication-service-v2
        image: clinical-platform/medication-service-v2:1.0.0
        imagePullPolicy: Always
        ports:
        - containerPort: 8005
          name: http
          protocol: TCP
        - containerPort: 8006
          name: grpc
          protocol: TCP
        env:
        - name: SERVICE_ENV
          value: "production"
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: medication-service-v2-secrets
              key: database-password
        - name: JWT_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: medication-service-v2-secrets
              key: jwt-secret-key
        volumeMounts:
        - name: config-volume
          mountPath: /app/configs
          readOnly: true
        - name: logs-volume
          mountPath: /app/logs
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1500m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8005
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8005
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 5
          failureThreshold: 3
        startupProbe:
          httpGet:
            path: /health/ready
            port: 8005
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 30
      volumes:
      - name: config-volume
        configMap:
          name: medication-service-v2-config
      - name: logs-volume
        emptyDir: {}
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - medication-service-v2
              topologyKey: kubernetes.io/hostname
---
apiVersion: v1
kind: Service
metadata:
  name: medication-service-v2-service
  namespace: medication-service-v2
  labels:
    app: medication-service-v2
spec:
  selector:
    app: medication-service-v2
  ports:
  - name: http
    port: 8005
    targetPort: 8005
    protocol: TCP
  - name: grpc
    port: 8006
    targetPort: 8006
    protocol: TCP
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: medication-service-v2-hpa
  namespace: medication-service-v2
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: medication-service-v2
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
```

### 2. Supporting Services Deployment

Create `deployments/k8s/postgres.yaml`:
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres-v2
  namespace: medication-service-v2
spec:
  serviceName: postgres-v2-service
  replicas: 1
  selector:
    matchLabels:
      app: postgres-v2
  template:
    metadata:
      labels:
        app: postgres-v2
    spec:
      containers:
      - name: postgres
        image: postgres:15
        env:
        - name: POSTGRES_DB
          value: medication_v2
        - name: POSTGRES_USER
          value: medication_user
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: medication-service-v2-secrets
              key: database-password
        ports:
        - containerPort: 5432
          name: postgres
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        - name: postgres-config
          mountPath: /etc/postgresql/postgresql.conf
          subPath: postgresql.conf
        livenessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - medication_user
            - -d
            - medication_v2
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - medication_user
            - -d
            - medication_v2
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: postgres-config
        configMap:
          name: postgres-config
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres-v2-service
  namespace: medication-service-v2
spec:
  selector:
    app: postgres-v2
  ports:
  - port: 5432
    targetPort: 5432
  clusterIP: None
```

### 3. Kubernetes Deployment Script

Create `scripts/k8s-deploy.sh`:
```bash
#!/bin/bash
set -e

NAMESPACE="medication-service-v2"
KUBECTL_TIMEOUT="300s"
IMAGE_TAG="${SERVICE_VERSION:-1.0.0}"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Create namespace
create_namespace() {
    log "Creating namespace..."
    kubectl apply -f deployments/k8s/namespace.yaml
}

# Deploy secrets and config
deploy_config() {
    log "Deploying configuration..."
    kubectl apply -f deployments/k8s/secrets.yaml
    kubectl apply -f deployments/k8s/configmap.yaml
}

# Deploy database
deploy_database() {
    log "Deploying PostgreSQL..."
    kubectl apply -f deployments/k8s/postgres.yaml
    
    log "Waiting for PostgreSQL to be ready..."
    kubectl wait --for=condition=ready pod -l app=postgres-v2 \
        --namespace=$NAMESPACE --timeout=$KUBECTL_TIMEOUT
}

# Deploy Redis
deploy_redis() {
    log "Deploying Redis..."
    kubectl apply -f deployments/k8s/redis.yaml
    
    log "Waiting for Redis to be ready..."
    kubectl wait --for=condition=ready pod -l app=redis-v2 \
        --namespace=$NAMESPACE --timeout=$KUBECTL_TIMEOUT
}

# Run database migrations
run_migrations() {
    log "Running database migrations..."
    kubectl run migration-job --image=clinical-platform/medication-service-v2:$IMAGE_TAG \
        --namespace=$NAMESPACE --rm -i --restart=Never \
        --command -- ./medication-service-v2 -migrate-only
}

# Deploy supporting services
deploy_supporting_services() {
    log "Deploying supporting services..."
    
    # Flow2 Go Engine
    kubectl apply -f deployments/k8s/flow2-go-engine.yaml
    
    # Rust Clinical Engine
    kubectl apply -f deployments/k8s/rust-clinical-engine.yaml
    
    # Knowledge Bases
    kubectl apply -f deployments/k8s/knowledge-bases.yaml
    
    log "Waiting for supporting services to be ready..."
    kubectl wait --for=condition=available deployment/flow2-go-engine-v2 \
        --namespace=$NAMESPACE --timeout=$KUBECTL_TIMEOUT
    kubectl wait --for=condition=available deployment/rust-clinical-engine-v2 \
        --namespace=$NAMESPACE --timeout=$KUBECTL_TIMEOUT
}

# Deploy main service
deploy_main_service() {
    log "Deploying main medication service..."
    
    # Update image tag in deployment
    sed -i "s|clinical-platform/medication-service-v2:.*|clinical-platform/medication-service-v2:$IMAGE_TAG|" \
        deployments/k8s/deployment.yaml
    
    kubectl apply -f deployments/k8s/deployment.yaml
    
    log "Waiting for deployment to be ready..."
    kubectl wait --for=condition=available deployment/medication-service-v2 \
        --namespace=$NAMESPACE --timeout=$KUBECTL_TIMEOUT
}

# Verify deployment
verify_deployment() {
    log "Verifying deployment..."
    
    # Check pod status
    kubectl get pods -n $NAMESPACE
    
    # Test service health
    log "Testing service health..."
    kubectl port-forward -n $NAMESPACE svc/medication-service-v2-service 8005:8005 &
    PORT_FORWARD_PID=$!
    sleep 5
    
    if curl -f http://localhost:8005/health/ready; then
        log "✓ Health check passed"
    else
        log "✗ Health check failed"
        kill $PORT_FORWARD_PID
        exit 1
    fi
    
    kill $PORT_FORWARD_PID
    log "Deployment verification completed successfully"
}

# Main deployment flow
main() {
    log "Starting Kubernetes deployment..."
    
    create_namespace
    deploy_config
    deploy_database
    deploy_redis
    run_migrations
    deploy_supporting_services
    deploy_main_service
    verify_deployment
    
    log "Kubernetes deployment completed successfully!"
    log "Service endpoints:"
    kubectl get svc -n $NAMESPACE
}

# Rollback function
rollback() {
    PREVIOUS_VERSION=${1:-"previous"}
    
    log "Rolling back to version: $PREVIOUS_VERSION"
    
    kubectl rollout undo deployment/medication-service-v2 -n $NAMESPACE
    kubectl rollout status deployment/medication-service-v2 -n $NAMESPACE --timeout=$KUBECTL_TIMEOUT
    
    log "Rollback completed"
}

# Handle script arguments
case "${1:-deploy}" in
    deploy)
        main
        ;;
    rollback)
        rollback $2
        ;;
    *)
        echo "Usage: $0 {deploy|rollback [version]}"
        exit 1
        ;;
esac
```

## Migration Strategy

### 1. Blue-Green Deployment Strategy

Create `scripts/blue-green-migration.sh`:
```bash
#!/bin/bash
set -e

# Configuration
BLUE_ENV="medication-service"      # Current production (Python)
GREEN_ENV="medication-service-v2"  # New service (Go/Rust)
LOAD_BALANCER_CONFIG="/etc/nginx/upstream.conf"
MIGRATION_LOG="/var/log/migration.log"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a $MIGRATION_LOG
}

# Phase 1: Deploy Green environment
deploy_green() {
    log "Phase 1: Deploying Green environment (Go/Rust service)"
    
    # Deploy new service without routing traffic
    docker-compose -f docker-compose.production.yml up -d
    
    # Wait for health checks
    sleep 60
    
    # Verify Green environment
    if curl -f http://localhost:8005/health/ready; then
        log "✓ Green environment is healthy"
    else
        log "✗ Green environment health check failed"
        return 1
    fi
    
    # Run integration tests
    log "Running integration tests on Green environment..."
    ./scripts/integration-tests.sh http://localhost:8005
}

# Phase 2: Gradual traffic migration
gradual_migration() {
    log "Phase 2: Starting gradual traffic migration"
    
    # Start with 10% traffic to Green
    update_load_balancer_weights 90 10
    log "Routing 10% traffic to Green environment"
    sleep 300  # Monitor for 5 minutes
    
    # Increase to 50% if no issues
    if check_error_rates; then
        update_load_balancer_weights 50 50
        log "Routing 50% traffic to Green environment"
        sleep 300
    else
        log "Error rates too high, rolling back"
        rollback_to_blue
        return 1
    fi
    
    # Final migration to 100% Green
    if check_error_rates; then
        update_load_balancer_weights 0 100
        log "Routing 100% traffic to Green environment"
        sleep 300
    else
        log "Error rates too high, rolling back"
        rollback_to_blue
        return 1
    fi
}

# Phase 3: Complete migration
complete_migration() {
    log "Phase 3: Completing migration"
    
    # Final health check
    if check_error_rates && check_performance_metrics; then
        log "Migration successful, shutting down Blue environment"
        docker-compose -f ../medication-service/docker-compose.yml down
        
        # Update monitoring alerts
        update_monitoring_targets
        
        log "Migration completed successfully!"
    else
        log "Issues detected, keeping Blue environment running"
        return 1
    fi
}

# Update load balancer weights
update_load_balancer_weights() {
    BLUE_WEIGHT=$1
    GREEN_WEIGHT=$2
    
    cat > $LOAD_BALANCER_CONFIG << EOF
upstream medication_service {
    server localhost:8004 weight=$BLUE_WEIGHT max_fails=3 fail_timeout=30s;
    server localhost:8005 weight=$GREEN_WEIGHT max_fails=3 fail_timeout=30s;
}
EOF
    
    # Reload nginx
    nginx -s reload
    log "Updated load balancer: Blue=$BLUE_WEIGHT%, Green=$GREEN_WEIGHT%"
}

# Check error rates
check_error_rates() {
    ERROR_RATE=$(curl -s http://localhost:9090/api/v1/query?query='rate(http_requests_total{status=~"5.."}[5m])' | \
                 jq -r '.data.result[0].value[1] // "0"')
    
    if (( $(echo "$ERROR_RATE < 0.01" | bc -l) )); then
        log "✓ Error rate acceptable: $ERROR_RATE"
        return 0
    else
        log "✗ Error rate too high: $ERROR_RATE"
        return 1
    fi
}

# Check performance metrics
check_performance_metrics() {
    AVG_RESPONSE_TIME=$(curl -s 'http://localhost:9090/api/v1/query?query=rate(http_request_duration_seconds_sum[5m])/rate(http_request_duration_seconds_count[5m])' | \
                        jq -r '.data.result[0].value[1] // "0"')
    
    if (( $(echo "$AVG_RESPONSE_TIME < 0.25" | bc -l) )); then
        log "✓ Response time acceptable: ${AVG_RESPONSE_TIME}s"
        return 0
    else
        log "✗ Response time too high: ${AVG_RESPONSE_TIME}s"
        return 1
    fi
}

# Rollback to Blue environment
rollback_to_blue() {
    log "Rolling back to Blue environment"
    
    update_load_balancer_weights 100 0
    
    # Stop Green environment
    docker-compose -f docker-compose.production.yml down
    
    log "Rollback completed - all traffic on Blue environment"
}

# Main migration function
main() {
    log "Starting Blue-Green migration from Python to Go/Rust service"
    
    if deploy_green; then
        if gradual_migration; then
            complete_migration
        else
            log "Migration failed during gradual phase"
            exit 1
        fi
    else
        log "Green environment deployment failed"
        exit 1
    fi
}

main "$@"
```

### 2. Data Migration Strategy

Create `scripts/data-migration.sh`:
```bash
#!/bin/bash
set -e

SOURCE_DB="medication_db"
TARGET_DB="medication_v2"
MIGRATION_LOG="/var/log/data-migration.log"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a $MIGRATION_LOG
}

# Migrate core data
migrate_medications() {
    log "Migrating medications data..."
    
    # Export from source
    psql -h localhost -p 5432 -U medication_user -d $SOURCE_DB -c \
        "COPY (SELECT * FROM medications) TO STDOUT WITH CSV HEADER" > /tmp/medications.csv
    
    # Import to target
    psql -h localhost -p 5434 -U medication_user -d $TARGET_DB -c \
        "\COPY medications FROM '/tmp/medications.csv' WITH CSV HEADER"
    
    log "Migrated $(wc -l < /tmp/medications.csv) medications"
}

# Migrate proposal data
migrate_proposals() {
    log "Migrating proposal data..."
    
    # Only migrate recent proposals (last 30 days)
    psql -h localhost -p 5432 -U medication_user -d $SOURCE_DB -c \
        "COPY (SELECT * FROM prescriptions WHERE proposal_timestamp > NOW() - INTERVAL '30 days') 
         TO STDOUT WITH CSV HEADER" > /tmp/proposals.csv
    
    # Transform and import
    python3 scripts/transform-proposals.py /tmp/proposals.csv | \
        psql -h localhost -p 5434 -U medication_user -d $TARGET_DB -c \
        "\COPY medication_proposals FROM STDIN WITH CSV HEADER"
    
    log "Migrated $(wc -l < /tmp/proposals.csv) proposals"
}

# Verify data integrity
verify_migration() {
    log "Verifying data migration..."
    
    # Count records
    SOURCE_COUNT=$(psql -h localhost -p 5432 -U medication_user -d $SOURCE_DB -t -c \
        "SELECT COUNT(*) FROM medications")
    TARGET_COUNT=$(psql -h localhost -p 5434 -U medication_user -d $TARGET_DB -t -c \
        "SELECT COUNT(*) FROM medications")
    
    if [[ "$SOURCE_COUNT" == "$TARGET_COUNT" ]]; then
        log "✓ Medication count matches: $SOURCE_COUNT"
    else
        log "✗ Medication count mismatch: Source=$SOURCE_COUNT, Target=$TARGET_COUNT"
        return 1
    fi
    
    # Verify data integrity with checksums
    SOURCE_CHECKSUM=$(psql -h localhost -p 5432 -U medication_user -d $SOURCE_DB -t -c \
        "SELECT MD5(STRING_AGG(rxnorm_code || generic_name, '' ORDER BY medication_id)) FROM medications")
    TARGET_CHECKSUM=$(psql -h localhost -p 5434 -U medication_user -d $TARGET_DB -t -c \
        "SELECT MD5(STRING_AGG(rxnorm_code || generic_name, '' ORDER BY medication_id)) FROM medications")
    
    if [[ "$SOURCE_CHECKSUM" == "$TARGET_CHECKSUM" ]]; then
        log "✓ Data integrity verified"
    else
        log "✗ Data integrity check failed"
        return 1
    fi
}

main() {
    log "Starting data migration..."
    
    migrate_medications
    migrate_proposals
    verify_migration
    
    log "Data migration completed successfully"
}

main "$@"
```

## Production Considerations

### 1. Security Configuration

Create `configs/security.yaml`:
```yaml
security:
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/medication-service-v2.crt
    key_file: /etc/ssl/private/medication-service-v2.key
    min_version: "1.2"
    
  jwt:
    algorithm: "RS256"
    public_key_file: /etc/ssl/jwt/public.pem
    private_key_file: /etc/ssl/jwt/private.pem
    token_expiry: 24h
    
  api_keys:
    enabled: true
    encryption_key_file: /etc/ssl/api/encryption.key
    
  rate_limiting:
    enabled: true
    requests_per_minute: 1000
    burst_size: 100
    
  cors:
    enabled: true
    allowed_origins:
      - https://clinical-dashboard.hospital.com
      - https://provider-portal.hospital.com
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Authorization", "Content-Type", "X-Requested-With"]
    
  audit:
    enabled: true
    log_file: /var/log/audit.log
    log_requests: true
    log_responses: false  # Don't log patient data
```

### 2. Performance Tuning

Create `configs/performance.yaml`:
```yaml
performance:
  database:
    max_connections: 100
    max_idle_connections: 20
    connection_max_lifetime: 1h
    query_timeout: 30s
    slow_query_threshold: 100ms
    
  redis:
    max_connections: 50
    max_idle_connections: 10
    connection_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s
    
  http:
    max_concurrent_requests: 1000
    request_timeout: 30s
    keep_alive_timeout: 60s
    read_header_timeout: 10s
    
  clinical_engine:
    calculation_timeout: 30s
    max_parallel_calculations: 10
    cache_calculation_results: true
    
  memory:
    max_heap_size: 1.5GB
    garbage_collection: "parallel"
    
  monitoring:
    metrics_collection_interval: 10s
    health_check_timeout: 5s
```

## Monitoring & Observability

### 1. Prometheus Configuration

Create `monitoring/prometheus.yml`:
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "medication-service-v2-alerts.yml"

scrape_configs:
  - job_name: 'medication-service-v2'
    static_configs:
      - targets: ['localhost:8005']
    metrics_path: /metrics
    scrape_interval: 10s
    
  - job_name: 'flow2-go-engine-v2'
    static_configs:
      - targets: ['localhost:8085']
    metrics_path: /metrics
    
  - job_name: 'rust-clinical-engine-v2'
    static_configs:
      - targets: ['localhost:8095']
    metrics_path: /metrics
    
  - job_name: 'postgres-v2'
    static_configs:
      - targets: ['localhost:9187']
    
  - job_name: 'redis-v2'
    static_configs:
      - targets: ['localhost:9121']

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### 2. Alert Rules

Create `monitoring/medication-service-v2-alerts.yml`:
```yaml
groups:
- name: medication-service-v2
  rules:
  - alert: MedicationServiceDown
    expr: up{job="medication-service-v2"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Medication Service V2 is down"
      description: "Medication Service V2 has been down for more than 1 minute"
      
  - alert: HighErrorRate
    expr: rate(medication_v2_requests_total{status=~"5.."}[5m]) > 0.01
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate in Medication Service V2"
      description: "Error rate is {{ $value }} errors per second"
      
  - alert: HighLatency
    expr: histogram_quantile(0.95, rate(medication_v2_request_duration_seconds_bucket[5m])) > 0.25
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High latency in Medication Service V2"
      description: "95th percentile latency is {{ $value }} seconds"
      
  - alert: DatabaseConnectionsHigh
    expr: postgres_stat_database_numbackends{datname="medication_v2"} > 80
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "High database connections"
      description: "Database has {{ $value }} active connections"
```

### 3. Grafana Dashboard

Create `monitoring/grafana-dashboard.json` (abbreviated):
```json
{
  "dashboard": {
    "title": "Medication Service V2",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(medication_v2_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph", 
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(medication_v2_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(medication_v2_request_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      }
    ]
  }
}
```

## Backup & Recovery

### 1. Backup Strategy

Create `scripts/backup.sh`:
```bash
#!/bin/bash
set -e

BACKUP_DIR="/backup/medication-service-v2"
S3_BUCKET="clinical-platform-backups"
RETENTION_DAYS=30

create_backup() {
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_PATH="$BACKUP_DIR/$TIMESTAMP"
    
    mkdir -p "$BACKUP_PATH"
    
    # Database backup
    docker exec postgres-v2 pg_dump -U medication_user -d medication_v2 \
        --verbose --no-acl --no-owner > "$BACKUP_PATH/database.sql"
    
    # Configuration backup
    tar -czf "$BACKUP_PATH/configs.tar.gz" configs/
    
    # Application logs (last 7 days)
    find logs/ -name "*.log" -mtime -7 -exec cp {} "$BACKUP_PATH/" \;
    
    # Compress backup
    tar -czf "$BACKUP_PATH.tar.gz" -C "$BACKUP_DIR" "$TIMESTAMP"
    rm -rf "$BACKUP_PATH"
    
    echo "Backup created: $BACKUP_PATH.tar.gz"
}

upload_to_s3() {
    aws s3 cp "$BACKUP_PATH.tar.gz" "s3://$S3_BUCKET/medication-service-v2/"
    echo "Backup uploaded to S3"
}

cleanup_old_backups() {
    find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete
    echo "Cleaned up backups older than $RETENTION_DAYS days"
}

main() {
    create_backup
    upload_to_s3
    cleanup_old_backups
}

main "$@"
```

### 2. Recovery Procedures

Create `scripts/restore.sh`:
```bash
#!/bin/bash
set -e

restore_from_backup() {
    BACKUP_FILE="$1"
    
    if [[ ! -f "$BACKUP_FILE" ]]; then
        echo "Backup file not found: $BACKUP_FILE"
        exit 1
    fi
    
    echo "Restoring from backup: $BACKUP_FILE"
    
    # Stop services
    docker-compose -f docker-compose.production.yml down
    
    # Extract backup
    RESTORE_DIR="/tmp/restore_$(date +%s)"
    mkdir -p "$RESTORE_DIR"
    tar -xzf "$BACKUP_FILE" -C "$RESTORE_DIR"
    
    # Restore database
    docker-compose -f docker-compose.production.yml up -d postgres-v2
    sleep 10
    
    cat "$RESTORE_DIR"/*/database.sql | \
        docker exec -i postgres-v2 psql -U medication_user -d medication_v2
    
    # Restore configuration
    tar -xzf "$RESTORE_DIR"/*/configs.tar.gz -C .
    
    # Start services
    docker-compose -f docker-compose.production.yml up -d
    
    # Cleanup
    rm -rf "$RESTORE_DIR"
    
    echo "Restore completed successfully"
}

main() {
    if [[ $# -eq 0 ]]; then
        echo "Usage: $0 <backup-file>"
        echo "Available backups:"
        ls -la /backup/medication-service-v2/*.tar.gz
        exit 1
    fi
    
    restore_from_backup "$1"
}

main "$@"
```

This comprehensive deployment guide provides all the necessary procedures and configurations for successfully deploying the Go/Rust Medication Service V2 in production environments while ensuring high availability, security, and observability.