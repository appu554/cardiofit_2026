# KB-4 Patient Safety DevOps Migration Plan

## Executive Summary

This plan outlines the comprehensive DevOps upgrade for KB-4 Patient Safety system, transitioning from basic PostgreSQL/Redis setup to an advanced multi-environment deployment with TimescaleDB, Kafka streaming, and intelligent rollout strategies.

## Current State Assessment

### Existing Infrastructure
- ✅ Docker containerization (Go service + PostgreSQL + Redis)
- ✅ Basic Prometheus metrics and alerting rules
- ✅ Health check endpoints
- ✅ Multi-stage Docker builds
- ✅ Circuit breaker patterns
- ⚠️ Single environment deployment
- ⚠️ PostgreSQL without time-series optimization
- ❌ No Kafka integration
- ❌ No canary deployment capability

### Service Architecture
```
Current: Client → Safety Gateway (Go:8030) → CAE Service (gRPC:8027) → PostgreSQL/Redis
Target:  Client → Load Balancer → [Shadow|Canary|Prod] → TimescaleDB/Redis/Kafka
```

## Phase 1: Infrastructure Foundation (Week 1-2)

### 1.1 TimescaleDB Migration Strategy

**Migration Approach**: Blue-Green with Zero Downtime
```bash
# Step 1: Deploy TimescaleDB alongside PostgreSQL
# Step 2: Dual-write to both databases during migration
# Step 3: Validate data consistency
# Step 4: Switch reads to TimescaleDB
# Step 5: Decommission PostgreSQL
```

**Implementation**:
- Install TimescaleDB extension on new PostgreSQL instance
- Create hypertables for time-series safety data
- Implement dual-write mechanism in Safety Gateway
- Data validation and consistency checks

### 1.2 Kafka Integration Architecture

**Topic Design**:
```
safety-events-raw          # Raw safety assessment events
safety-events-processed    # Processed safety decisions
safety-alerts-critical     # Critical safety violations
safety-override-tokens     # Override token usage events
safety-metrics-stream      # Real-time metrics for monitoring
```

**Event Schema**:
```json
{
  "event_id": "uuid",
  "timestamp": "2025-01-15T10:30:00Z",
  "patient_id": "patient-123",
  "request_id": "req-456",
  "safety_tier": "TierVetoCritical",
  "decision": "unsafe|safe|warning",
  "engines_evaluated": ["cae", "allergy", "protocol"],
  "response_time_ms": 95,
  "override_used": false
}
```

### 1.3 Redis Enhancement Strategy

**Multi-Tier Caching**:
```
L1: In-Memory Go Cache (2-second TTL, hot data)
L2: Redis Cluster (5-minute TTL, warm data) 
L3: TimescaleDB (cold data, historical analytics)
```

## Phase 2: Multi-Environment Setup (Week 2-3)

### 2.1 Environment Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SHADOW ENV    │    │   CANARY ENV    │    │    PROD ENV     │
│  (Mirror Only)  │    │   (1-5% Live)   │    │  (95-99% Live)  │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ safety-shadow   │    │ safety-canary   │    │ safety-prod     │
│ Port: 8031      │    │ Port: 8032      │    │ Port: 8030      │
│ TimescaleDB     │    │ TimescaleDB     │    │ TimescaleDB     │
│ Redis Cluster   │    │ Redis Cluster   │    │ Redis Cluster   │
│ Kafka Topics    │    │ Kafka Topics    │    │ Kafka Topics    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 2.2 Traffic Routing Strategy

**Load Balancer Configuration**:
```nginx
# nginx.conf for traffic routing
upstream safety_shadow {
    server safety-shadow:8031;
}

upstream safety_canary {
    server safety-canary:8032;
}

upstream safety_prod {
    server safety-prod:8030;
}

server {
    location /api/safety {
        # Route based on deployment strategy
        if ($deployment_mode = "shadow") {
            proxy_pass http://safety_shadow;
            # Mirror to shadow without affecting response
        }
        
        if ($deployment_mode = "canary") {
            # 5% traffic to canary, 95% to prod
            proxy_pass http://safety_canary weight=5;
            proxy_pass http://safety_prod weight=95;
        }
        
        # Default to production
        proxy_pass http://safety_prod;
    }
}
```

## Phase 3: Monitoring & Observability (Week 3-4)

### 3.1 Clinical KPIs Dashboard

**Safety Gateway Metrics**:
```yaml
# prometheus.yml - Clinical KPIs
clinical_kpis:
  safety_decision_latency_p95: 95th percentile safety decision time
  unsafe_decision_rate: Rate of unsafe medication decisions
  engine_consensus_rate: Agreement rate between safety engines
  override_token_usage: Critical override usage patterns
  patient_safety_events: Safety events per patient per hour
  drug_interaction_blocks: Prevented dangerous interactions
  allergy_contraindication_blocks: Prevented allergic reactions
  clinical_protocol_adherence: Protocol compliance percentage
```

**Grafana Dashboard Panels**:
1. **Real-time Safety Decisions** (time series)
2. **Safety Engine Performance** (heatmap)
3. **Critical Override Usage** (alert table)
4. **Patient Risk Distribution** (histogram)
5. **System Health Overview** (stat panels)

### 3.2 Advanced Alerting Rules

**Tiered Alert System**:
```yaml
# Enhanced alerting rules
critical_alerts:
  - safety_gateway_down: Page on-call immediately
  - multiple_unsafe_decisions: Page clinical safety team
  - cae_service_unavailable: Auto-failover + page
  
warning_alerts:
  - high_response_latency: Slack notification
  - low_cache_hit_rate: Email to DevOps team
  - unusual_override_patterns: Notify clinical team

business_alerts:
  - daily_safety_summary: Email safety report
  - weekly_performance_review: Dashboard update
```

## Phase 4: Kubernetes Deployment (Week 4-5)

### 4.1 K8s Manifests Structure

```
k8s/
├── base/
│   ├── safety-gateway-deployment.yaml
│   ├── safety-gateway-service.yaml
│   ├── safety-gateway-configmap.yaml
│   └── safety-gateway-secrets.yaml
├── overlays/
│   ├── shadow/
│   ├── canary/
│   └── production/
└── monitoring/
    ├── servicemonitor.yaml
    ├── prometheusrule.yaml
    └── grafana-dashboard.yaml
```

### 4.2 Pod Disruption Budgets

```yaml
# PodDisruptionBudget for safety-critical service
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: safety-gateway-pdb
spec:
  minAvailable: 2  # Always maintain 2 replicas minimum
  selector:
    matchLabels:
      app: safety-gateway
```

### 4.3 Resource Management

```yaml
# Resource requests and limits
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"

# Horizontal Pod Autoscaler
spec:
  scaleTargetRef:
    kind: Deployment
    name: safety-gateway
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Phase 5: Shadow→Canary→Production Rollout (Week 5-6)

### 5.1 Shadow Mode Implementation

**Traffic Mirroring Without Enforcement**:
```go
// Shadow mode: Mirror traffic, don't affect responses
func (s *SafetyGateway) handleShadowRequest(ctx context.Context, req *SafetyRequest) {
    // Process in shadow environment
    shadowResult := s.shadowEngine.Evaluate(ctx, req)
    
    // Log differences but don't block request
    if shadowResult != prodResult {
        s.logger.Warn("Shadow/Prod result mismatch", 
            "request_id", req.ID,
            "shadow_result", shadowResult,
            "prod_result", prodResult)
    }
    
    // Continue with production result
    return prodResult
}
```

**Shadow Metrics Collection**:
```yaml
shadow_metrics:
  - shadow_prod_result_agreement_rate
  - shadow_decision_latency_comparison
  - shadow_error_rate_vs_prod
  - shadow_engine_consensus_differences
```

### 5.2 Canary Rollout Strategy

**Gradual Traffic Shift**:
```
Week 1: 1% canary, 99% prod
Week 2: 5% canary, 95% prod  
Week 3: 10% canary, 90% prod
Week 4: 25% canary, 75% prod
Week 5: 50% canary, 50% prod
Week 6: 100% canary (becomes new prod)
```

**Canary Success Criteria**:
```yaml
success_criteria:
  error_rate: < 0.1% (10x better than baseline)
  p95_latency: < 100ms (same as baseline)
  safety_decision_accuracy: > 99.9%
  no_critical_alerts: 0 critical alerts in 48h
  clinical_team_approval: Required sign-off
```

### 5.3 Auto-Rollback Triggers

**Automated Rollback Conditions**:
```yaml
rollback_triggers:
  immediate_rollback:
    - error_rate > 1%
    - p95_latency > 200ms
    - critical_safety_violations > 0
    - service_availability < 99%
    
  delayed_rollback:
    - sustained_high_latency: p95 > 150ms for 5min
    - cache_miss_rate > 50%
    - unusual_decision_patterns: deviation > 3 standard deviations
```

**Rollback Implementation**:
```bash
#!/bin/bash
# auto-rollback.sh
detect_rollback_condition() {
    if [ "$ERROR_RATE" -gt "1" ]; then
        echo "CRITICAL: Error rate $ERROR_RATE% exceeds threshold"
        execute_rollback "immediate"
    fi
}

execute_rollback() {
    local rollback_type=$1
    echo "Executing $rollback_type rollback..."
    
    # Switch traffic back to production
    kubectl patch deployment safety-gateway-canary -p '{"spec":{"replicas":0}}'
    kubectl patch service safety-gateway -p '{"spec":{"selector":{"version":"prod"}}}'
    
    # Send alerts
    send_alert "ROLLBACK_EXECUTED" "Canary deployment rolled back due to $rollback_type condition"
}
```

## Phase 6: CI/CD Pipeline Enhancement (Week 6-7)

### 6.1 Safety-Critical Pipeline

```yaml
# .github/workflows/safety-gateway-deploy.yml
name: Safety Gateway Deployment Pipeline

on:
  push:
    branches: [main]
    paths: ['backend/services/safety-gateway-platform/**']

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - name: Security Scan
        run: |
          gosec ./...
          docker run --rm -v $(pwd):/app clair-scanner
          
  comprehensive-testing:
    runs-on: ubuntu-latest
    steps:
      - name: Unit Tests
        run: go test -v ./... -coverage
        
      - name: Integration Tests
        run: |
          make test-integration
          
      - name: Load Testing
        run: |
          make load-test
          hey -n 10000 -c 50 http://localhost:8030/api/safety/evaluate
          
      - name: Security Testing
        run: |
          # Test safety-critical scenarios
          ./scripts/test-critical-safety-scenarios.sh
          
  deploy-shadow:
    needs: [security-scan, comprehensive-testing]
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to Shadow
        run: |
          kubectl apply -k k8s/overlays/shadow/
          ./scripts/verify-shadow-deployment.sh
          
  deploy-canary:
    needs: [deploy-shadow]
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - name: Deploy Canary
        run: |
          kubectl apply -k k8s/overlays/canary/
          ./scripts/start-canary-monitoring.sh
          
      - name: Canary Analysis
        run: |
          # 15-minute canary analysis
          ./scripts/canary-analysis.sh --duration=15m
          
      - name: Auto-promote or Rollback
        run: |
          if [ "$CANARY_SUCCESS" == "true" ]; then
            ./scripts/promote-canary.sh
          else
            ./scripts/rollback-canary.sh
          fi
```

### 6.2 Deployment Verification Scripts

```bash
#!/bin/bash
# verify-shadow-deployment.sh

echo "Verifying shadow deployment..."

# Health check
health_response=$(curl -f -s http://safety-shadow:8031/health)
if [ $? -ne 0 ]; then
    echo "FAIL: Shadow health check failed"
    exit 1
fi

# Metrics validation
metrics_available=$(curl -f -s http://safety-shadow:9090/metrics | grep -c "safety_gateway")
if [ "$metrics_available" -lt 10 ]; then
    echo "FAIL: Insufficient metrics available"
    exit 1
fi

# Database connectivity
db_status=$(curl -f -s http://safety-shadow:8031/health/database | jq -r '.status')
if [ "$db_status" != "healthy" ]; then
    echo "FAIL: Database connectivity issues"
    exit 1
fi

echo "SUCCESS: Shadow deployment verified"
```

## Phase 7: Integration with Medication Service (Week 7)

### 7.1 Docker Compose Integration

**Enhanced medication-service docker-compose.yml**:
```yaml
# Add to medication-service/docker-compose.safety.yml
version: '3.8'
services:
  safety-gateway:
    build: ../safety-gateway-platform
    ports:
      - "8030:8030"
    environment:
      - ENVIRONMENT=production
      - TIMESCALEDB_URL=postgresql://user:pass@timescaledb:5432/safety
      - KAFKA_BROKERS=kafka:9092
      - REDIS_URL=redis://redis:6379
    depends_on:
      - timescaledb
      - kafka
      - redis
    healthcheck:
      test: ["CMD", "./safety-gateway", "-health-check"]
      interval: 30s
      timeout: 10s
      retries: 3

  timescaledb:
    image: timescale/timescaledb:latest-pg15
    environment:
      - POSTGRES_DB=safety
      - POSTGRES_USER=safety_user
      - POSTGRES_PASSWORD=safety_password
    ports:
      - "5434:5432"
    volumes:
      - timescaledb_data:/var/lib/postgresql/data
      
  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      - KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181
      - KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092
    depends_on:
      - zookeeper
      
volumes:
  timescaledb_data:
```

### 7.2 Makefile Integration

**Enhanced safety-gateway Makefile targets**:
```makefile
# Add to existing Makefile

# Multi-environment targets
run-shadow:
	@echo "Starting Safety Gateway in shadow mode..."
	ENVIRONMENT=shadow PORT=8031 $(BUILD_DIR)/$(BINARY_NAME) -config=config.shadow.yaml

run-canary:
	@echo "Starting Safety Gateway in canary mode..."
	ENVIRONMENT=canary PORT=8032 $(BUILD_DIR)/$(BINARY_NAME) -config=config.canary.yaml

# Integration with medication service
integrate-medication:
	@echo "Starting integrated safety + medication services..."
	cd ../medication-service && make run-safety-integrated

# Deployment targets
deploy-k8s-shadow:
	@echo "Deploying to Kubernetes shadow environment..."
	kubectl apply -k k8s/overlays/shadow/

deploy-k8s-canary:
	@echo "Deploying to Kubernetes canary environment..."
	kubectl apply -k k8s/overlays/canary/
	./scripts/canary-analysis.sh

deploy-k8s-prod:
	@echo "Promoting canary to production..."
	kubectl apply -k k8s/overlays/production/

# Rollback targets
rollback-canary:
	@echo "Rolling back canary deployment..."
	./scripts/rollback-canary.sh

rollback-prod:
	@echo "Rolling back production deployment..."
	./scripts/rollback-production.sh
```

## Timeline & Resource Requirements

### Implementation Timeline (7 weeks)
```
Week 1-2: Infrastructure Foundation (TimescaleDB, Kafka, Redis)
Week 3: Multi-environment setup and traffic routing
Week 4: Monitoring, alerting, and observability
Week 5: Kubernetes deployment and PDB setup
Week 6: Shadow/Canary rollout implementation
Week 7: CI/CD pipeline and integration testing
```

### Resource Requirements
- **DevOps Engineers**: 2 FTE for 7 weeks
- **Go Developers**: 1 FTE for shadow/canary implementation
- **Infrastructure**: 3x current compute resources (shadow/canary/prod)
- **Clinical Safety Team**: Weekly review and sign-off

## Success Metrics

### Technical KPIs
- Zero-downtime deployment success rate: >99%
- Rollback execution time: <2 minutes
- Shadow/Production result agreement: >99.9%
- Canary false-positive rate: <0.1%

### Clinical KPIs  
- Patient safety event detection: 100% accuracy
- Critical drug interaction blocks: >99.9%
- Override token abuse detection: Real-time alerting
- Audit compliance: 100% (HIPAA, FDA)

## Risk Mitigation

### High-Risk Scenarios
1. **Data loss during TimescaleDB migration**
   - Mitigation: Dual-write period with validation
   - Rollback: Immediate switch back to PostgreSQL

2. **Canary false positives causing rollbacks**
   - Mitigation: Gradual traffic increase with validation
   - Monitoring: Clinical team oversight for edge cases

3. **Kafka message loss affecting audit trails**
   - Mitigation: Persistent queues with replication
   - Backup: Synchronous audit log to TimescaleDB

This comprehensive plan provides a production-ready approach to upgrading KB-4 Patient Safety with advanced DevOps practices while maintaining the critical safety requirements of a healthcare system.