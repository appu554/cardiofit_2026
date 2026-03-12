# Production Deployment Guide
## Calculate > Validate > Commit Workflow Engine

This guide provides comprehensive instructions for deploying the workflow engine in production with monitoring, observability, and operational excellence.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          Load Balancer                          │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────────────┐
│                Apollo Federation Gateway                        │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────────────┐
│            Workflow Engine Service (Port 8050)                 │
│  ┌─────────────────┬┼┬─────────────────┐                       │
│  │    Calculate    │││    Validate     │    Commit             │
│  │   (Flow2 Go)    │││ (Safety Gateway)│ (Medication Service)  │
│  └─────────────────┴┼┴─────────────────┴───────────────────────┘
└─────────────────────┼───────────────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────────────┐
│                Monitoring Stack                                 │
│  Prometheus │ Grafana │ AlertManager │ Jaeger │ Loki           │
└─────────────────────┼───────────────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────────────┐
│              Data & Message Layer                               │
│  PostgreSQL │ Redis │ Kafka │ Neo4j                            │
└─────────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### 1. Prerequisites
```bash
# Install dependencies
pip install -r requirements.txt

# Set up environment
cp config/production.env .env
# Edit .env with your production values
```

### 2. Start Monitoring Infrastructure
```bash
# Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# Verify services
curl http://localhost:9090/targets  # Prometheus
curl http://localhost:3000         # Grafana (admin/workflow_admin_2024)
```

### 3. Deploy Workflow Engine
```bash
# Production deployment
uvicorn app.main:app \
  --host 0.0.0.0 \
  --port 8050 \
  --workers 4 \
  --access-log \
  --loop uvloop
```

## 🔧 Configuration Management

### Environment Variables

#### Core Service Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | workflow-engine-service | Service identifier |
| `SERVICE_VERSION` | 1.0.0 | Service version |
| `ENVIRONMENT` | production | Environment (dev/staging/production) |
| `DEBUG` | false | Enable debug mode |
| `LOG_LEVEL` | INFO | Logging level |

#### Performance Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `MAX_CONCURRENT_WORKFLOWS` | 100 | Maximum concurrent workflows |
| `CALCULATE_TIMEOUT_MS` | 175 | Calculate phase timeout |
| `VALIDATE_TIMEOUT_MS` | 100 | Validate phase timeout |
| `COMMIT_TIMEOUT_MS` | 50 | Commit phase timeout |

#### Monitoring Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_PROMETHEUS_METRICS` | true | Enable Prometheus metrics |
| `ENABLE_DISTRIBUTED_TRACING` | true | Enable Jaeger tracing |
| `TRACE_SAMPLE_RATE` | 0.1 | Tracing sample rate (10%) |

### Database Configuration
```bash
# PostgreSQL for proposal persistence
DATABASE_URL=postgresql://workflow_user:secure_password@postgresql:5432/workflow_proposals
DATABASE_POOL_SIZE=20
DATABASE_MAX_OVERFLOW=30

# Redis for caching
REDIS_URL=redis://redis:6379/0
REDIS_CACHE_TTL_SECONDS=300
```

## 📊 Monitoring & Observability

### Health Check Endpoints
```bash
# Liveness probe (Kubernetes)
curl http://localhost:8050/monitoring/health/live

# Readiness probe (Kubernetes)
curl http://localhost:8050/monitoring/health/ready

# Comprehensive health check
curl http://localhost:8050/monitoring/health

# Current metrics snapshot
curl http://localhost:8050/monitoring/metrics/current
```

### Prometheus Metrics
Available at `http://localhost:8050/monitoring/metrics`

#### Key Metrics
- `workflow_requests_total` - Total workflow requests by status
- `workflow_phase_duration_seconds` - Phase execution time histogram
- `safety_gateway_validations_total` - Safety validation counters
- `proposal_operations_total` - Database operation counters
- `workflow_performance_target_adherence_ratio` - Performance target adherence

### Grafana Dashboards
Access Grafana at `http://localhost:3000` (admin/workflow_admin_2024)

Pre-configured dashboards:
- **Workflow Overview**: High-level workflow performance metrics
- **Phase Performance**: Detailed Calculate/Validate/Commit timing
- **Safety Gateway**: Validation patterns and safety metrics
- **System Health**: Infrastructure and resource utilization

### Alerting Rules
Critical alerts configured in `config/alert_rules.yml`:

#### Performance Alerts
- `HighWorkflowErrorRate`: Error rate > 10% for 2 minutes
- `WorkflowPerformanceTargetMiss`: < 80% workflows meeting targets
- `HighPhaseExecutionTime`: 95th percentile > 200ms

#### System Health Alerts
- `WorkflowServiceDown`: Service unavailable for > 1 minute
- `SafetyGatewayDown`: Safety Gateway unavailable
- `HighMemoryUsage`: Memory usage > 90%
- `HighCPUUsage`: CPU usage > 80%

#### Business Logic Alerts
- `UnusualValidationPattern`: > 30% unsafe validations in 1 hour
- `LowWorkflowThroughput`: < 0.1 requests/second for 10 minutes

## 🔐 Security & Compliance

### Authentication
- JWT token validation
- Service-to-service authentication
- Rate limiting (1000 requests/minute)

### Data Protection
- PHI encryption enabled
- Audit logging for compliance
- 7-year data retention for medical records
- HIPAA compliance features

### Network Security
```yaml
# Allowed origins
CORS_ORIGINS: ["https://cardiofit.clinical-synthesis-hub.com"]

# Allowed hosts  
ALLOWED_HOSTS: ["*.clinical-synthesis-hub.com"]
```

## 📈 Performance Optimization

### Performance Targets
- **Total workflow**: < 325ms (95th percentile)
- **Calculate phase**: < 175ms
- **Validate phase**: < 100ms  
- **Commit phase**: < 50ms

### Scaling Configuration
```bash
# Horizontal scaling
replicas: 3
maxReplicas: 10
targetCPUUtilizationPercentage: 70
targetMemoryUtilizationPercentage: 80

# Resource limits
resources:
  limits:
    memory: "2Gi"
    cpu: "1000m"
  requests:
    memory: "1Gi" 
    cpu: "500m"
```

### Database Optimization
```sql
-- Indexes for proposal queries
CREATE INDEX CONCURRENTLY idx_proposals_correlation_id ON workflow_proposals(correlation_id);
CREATE INDEX CONCURRENTLY idx_proposals_status_created ON workflow_proposals(status, created_at);
CREATE INDEX CONCURRENTLY idx_proposals_patient_id ON workflow_proposals(patient_id);

-- Connection pooling
DATABASE_POOL_SIZE=20
DATABASE_MAX_OVERFLOW=30
```

## 🚨 Incident Response

### Runbook: High Error Rate
```bash
# 1. Check service health
curl http://localhost:8050/monitoring/health

# 2. Check recent errors in logs
kubectl logs -f deployment/workflow-engine --tail=100

# 3. Check database connectivity
kubectl exec -it postgres-pod -- psql -U workflow_user -c "SELECT 1;"

# 4. Check Safety Gateway status
curl http://safety-gateway:8080/health
```

### Runbook: Performance Degradation
```bash
# 1. Check current metrics
curl http://localhost:8050/monitoring/performance

# 2. Identify slow phases
curl "http://localhost:8050/monitoring/metrics/timeseries?metric_name=validate_duration_ms&hours=1"

# 3. Check resource utilization
kubectl top pods
kubectl describe pod workflow-engine-pod

# 4. Scale if needed
kubectl scale deployment workflow-engine --replicas=5
```

## 🔄 Backup & Recovery

### Database Backup
```bash
# Automated daily backup at 2 AM
BACKUP_SCHEDULE="0 2 * * *"
BACKUP_RETENTION_DAYS=90

# Manual backup
pg_dump -U workflow_user workflow_proposals > backup_$(date +%Y%m%d).sql
```

### Configuration Backup
```bash
# Backup configuration
kubectl get configmap workflow-config -o yaml > config-backup.yaml

# Backup secrets
kubectl get secret workflow-secrets -o yaml > secrets-backup.yaml
```

## 🔍 Troubleshooting

### Common Issues

#### Issue: Workflow Timeouts
```bash
# Symptoms: High timeout errors in logs
# Solution: Check downstream services
curl http://safety-gateway:8080/health
curl http://medication-service:8004/health

# Increase timeout if needed
WORKFLOW_TIMEOUT_SECONDS=45
```

#### Issue: High Memory Usage
```bash
# Symptoms: OOMKilled events
# Solution: Check for memory leaks
curl http://localhost:8050/monitoring/metrics | grep memory

# Scale up resources
kubectl patch deployment workflow-engine -p '{"spec":{"template":{"spec":{"containers":[{"name":"workflow-engine","resources":{"limits":{"memory":"4Gi"}}}]}}}}'
```

#### Issue: Database Connection Pool Exhausted  
```bash
# Symptoms: "connection pool exhausted" errors
# Solution: Increase pool size
DATABASE_POOL_SIZE=40
DATABASE_MAX_OVERFLOW=60

# Monitor pool usage
curl http://localhost:8050/monitoring/metrics | grep pool
```

## 📋 Pre-deployment Checklist

### Infrastructure
- [ ] PostgreSQL database configured with proper indexes
- [ ] Redis cache available and configured
- [ ] Safety Gateway HTTP endpoints accessible
- [ ] Medication Service API endpoints accessible
- [ ] Network policies configured for service-to-service communication

### Monitoring
- [ ] Prometheus configured with workflow metrics endpoints
- [ ] Grafana dashboards imported and configured
- [ ] AlertManager rules configured with proper notification channels
- [ ] Log aggregation (Loki/ELK) configured
- [ ] Distributed tracing (Jaeger) configured

### Security
- [ ] JWT secrets configured and rotated
- [ ] Database credentials secured
- [ ] Network policies applied
- [ ] CORS origins configured for production domains
- [ ] Rate limiting configured

### Performance
- [ ] Resource limits and requests configured
- [ ] Horizontal Pod Autoscaler configured
- [ ] Database connection pooling tuned
- [ ] Cache TTL configured appropriately

### Compliance
- [ ] Audit logging enabled
- [ ] PHI encryption enabled
- [ ] Data retention policies configured
- [ ] HIPAA compliance features verified

## 🏥 Production Readiness Score

Use this checklist to assess production readiness:

| Category | Requirements | Status |
|----------|-------------|---------|
| **Functionality** | All tests passing, Integration tests successful | ✅ |
| **Performance** | < 325ms total workflow time, Load testing completed | ✅ |
| **Reliability** | Health checks configured, Circuit breakers enabled | ✅ |
| **Monitoring** | Metrics/Logs/Traces configured, Dashboards created | ✅ |
| **Security** | Authentication/Authorization, Encryption enabled | ✅ |
| **Operability** | Deployment automation, Runbooks documented | ✅ |
| **Compliance** | Audit trails, Data retention, HIPAA features | ✅ |

## 📞 Support & Escalation

### Contact Information
- **Primary On-call**: ops@clinical-synthesis-hub.com
- **Development Team**: dev@clinical-synthesis-hub.com
- **Architecture Team**: arch@clinical-synthesis-hub.com

### Escalation Matrix
1. **P1 (Critical)**: Service down, data loss → Immediate escalation
2. **P2 (High)**: Performance degradation → 1-hour response
3. **P3 (Medium)**: Feature issues → 4-hour response  
4. **P4 (Low)**: Enhancement requests → 24-hour response

---

## 🔗 Additional Resources

- [API Documentation](http://localhost:8050/api/docs)
- [Grafana Dashboards](http://localhost:3000)
- [Prometheus Metrics](http://localhost:9090)
- [Apollo Federation Schema](http://localhost:4000/graphql)
- [Integration Test Results](./tests/integration/)

For questions or issues, please refer to the troubleshooting section or contact the support team.