# KB Services Unified Monitoring Dashboard

Comprehensive monitoring, alerting, and observability stack for all 7 Knowledge Base (KB) services with multi-database technology support.

## 🎯 Overview

This unified monitoring solution provides:

- **Real-time metrics** for all KB services
- **Comprehensive dashboards** with service health, performance, and business metrics  
- **Intelligent alerting** with severity-based routing
- **Distributed tracing** for request flow analysis
- **Log aggregation** across all services
- **Database monitoring** for PostgreSQL, MongoDB, Neo4j, and Elasticsearch
- **Cache performance** tracking for multi-layer cache architecture
- **Security monitoring** with compliance tracking

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   KB Services   │    │   Exporters      │    │   Monitoring    │
│                 │    │                  │    │                 │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ ┌─────────────┐ │
│ │ KB-1 :8081  │ ├────┤ │ Node Exp.    │ ├────┤ │ Prometheus  │ │
│ │ KB-2 :8082  │ │    │ │ PostgreSQL   │ │    │ │ :9090       │ │
│ │ KB-3 :8084  │ │    │ │ MongoDB      │ │    │ └─────────────┘ │
│ │ KB-4 :8085  │ │    │ │ Redis        │ │    │ ┌─────────────┐ │
│ │ KB-5 :8086  │ │    │ │ Elasticsearch│ │    │ │ Grafana     │ │
│ │ KB-6 :8087  │ │    │ │ Neo4j        │ │    │ │ :3000       │ │
│ │ KB-7 :8088  │ │    │ └──────────────┘ │    │ └─────────────┘ │
│ └─────────────┘ │    └──────────────────┘    │ ┌─────────────┐ │
└─────────────────┘                            │ │Alertmanager │ │
                                               │ │ :9093       │ │
┌─────────────────┐    ┌──────────────────┐    │ └─────────────┘ │
│   Databases     │    │   Long-term      │    │ ┌─────────────┐ │
│                 │    │   Storage        │    │ │ Loki        │ │
│ PostgreSQL:5432 │    │                  │    │ │ :3100       │ │
│ MongoDB:27017   │    │ VictoriaMetrics  │    │ └─────────────┘ │
│ Neo4j:7474      │    │ :8428            │    │ ┌─────────────┐ │
│ Elasticsearch   │    │                  │    │ │ Jaeger      │ │
│ :9200           │    │ (12 month        │    │ │ :16686      │ │
│ Redis:6379/6380 │    │  retention)      │    │ └─────────────┘ │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## 📊 Components

### Core Monitoring
- **Prometheus** - Metrics collection and alerting engine
- **Grafana** - Visualization and dashboards
- **Alertmanager** - Alert routing and notification management
- **VictoriaMetrics** - Long-term metrics storage (12 months)

### Observability
- **Loki** - Log aggregation and analysis  
- **Promtail** - Log collection agent
- **Jaeger** - Distributed tracing and request flow

### Database Monitoring
- **PostgreSQL Exporter** - KB-1, KB-4, KB-5, KB-7 metrics
- **MongoDB Exporter** - KB-2 Clinical Context metrics  
- **Neo4j Exporter** - KB-3 Guideline Evidence metrics
- **Elasticsearch Exporter** - KB-6 Formulary metrics
- **Redis Exporter** - Cache layer performance

### System Monitoring
- **Node Exporter** - System-level metrics
- **Docker monitoring** - Container performance

## 🚀 Quick Start

### Prerequisites
- Docker and Docker Compose
- 8GB+ RAM recommended
- Ports 3000, 9090, 9093, 3100, 16686 available

### Deployment

```bash
# Clone or navigate to monitoring directory
cd monitoring/

# Deploy complete stack
make deploy

# Or using individual commands
make start
make health
make dashboard
```

### Windows Deployment
```powershell
# Windows-specific deployment
make deploy-windows
make health-windows
```

## 🎛️ Access Points

| Service | URL | Credentials | Description |
|---------|-----|-------------|-------------|
| **Grafana** | http://localhost:3000 | admin / kb-admin-2024 | Main dashboards and visualization |
| **Prometheus** | http://localhost:9090 | - | Metrics queries and targets |
| **Alertmanager** | http://localhost:9093 | - | Alert management |
| **Jaeger** | http://localhost:16686 | - | Distributed tracing |
| **Loki** | http://localhost:3100 | - | Log queries |
| **VictoriaMetrics** | http://localhost:8428 | - | Long-term storage |

## 📈 Key Dashboards

### 1. Unified KB Services Dashboard
**URL:** http://localhost:3000/d/kb-unified

**Features:**
- Service health overview for all 7 KB services
- Real-time request rates and response times
- Error rates with SLA compliance
- Database performance by technology type
- Cache hit rates across L1/L2/L3 layers
- Resource utilization (CPU, memory, connections)
- Business metrics (validations, calculations, interactions)

**Key Panels:**
```
┌─────────────────────────────────────────────────────┐
│ Service Health Overview                             │
│ ✅ KB-1  ✅ KB-2  ✅ KB-3  ❌ KB-4                   │
│ ✅ KB-5  ✅ KB-6  ✅ KB-7                            │
└─────────────────────────────────────────────────────┘

┌───────────────┬───────────────┬───────────────────────┐
│ Total RPS     │ Avg Response  │ Error Rate           │
│ 1,247 req/sec │ 45ms         │ 0.12%                │
└───────────────┴───────────────┴───────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Database Performance by Technology                   │
│ PostgreSQL: 12ms avg │ MongoDB: 8ms avg             │
│ Neo4j: 45ms avg      │ Elasticsearch: 23ms avg     │
└─────────────────────────────────────────────────────┘
```

### 2. Individual Service Dashboards
Each KB service has dedicated dashboards:
- KB-1 Drug Rules: Drug calculations, TOML validations  
- KB-2 Clinical Context: Phenotype matching, MongoDB operations
- KB-3 Guidelines: Graph traversals, Neo4j performance
- KB-4 Patient Safety: Safety alerts, TimescaleDB metrics
- KB-5 Drug Interactions: Interaction matrix lookups
- KB-6 Formulary: Elasticsearch searches, tier analysis  
- KB-7 Terminology: Code validations, SNOMED operations

## 🚨 Alerting System

### Alert Categories

| Category | Severity | Target Team | Response Time |
|----------|----------|-------------|---------------|
| **Critical** | Service Down | On-call + Management | < 5 minutes |
| **Security** | Unauthorized access | Security Team | < 2 minutes |
| **Safety** | Clinical safety risk | Clinical Safety Team | < 1 minute |
| **Compliance** | HIPAA/Audit violations | Compliance Team | < 30 minutes |
| **Performance** | High latency/errors | Platform Team | < 15 minutes |
| **Database** | DB connection issues | Database Team | < 10 minutes |

### Alert Rules Examples

```yaml
# Service Down - Critical
- alert: KBServiceDown
  expr: up{job=~"kb-.*"} == 0
  for: 1m
  labels:
    severity: critical
  
# High Error Rate - Warning  
- alert: KBServiceHighErrorRate
  expr: kb:error_rate_5m > 0.05
  for: 2m
  labels:
    severity: warning

# Security Violation - Critical
- alert: UnauthorizedAccess
  expr: increase(authorization_failures_total[1m]) > 10
  for: 0m
  labels:
    severity: critical
    category: security
```

### Notification Channels
- **Email** - All alert categories
- **Slack** - Real-time notifications with actions  
- **PagerDuty** - Critical alerts only
- **Webhooks** - Custom integrations

## 📋 Business Metrics

### Clinical Operations
- **Terminology Validations** - SNOMED, ICD-10, RxNorm, LOINC accuracy
- **Drug Calculations** - Dosing calculations per second
- **Interaction Checks** - Drug interaction analysis rate
- **Safety Alerts** - Clinical safety alert generation
- **Guideline Searches** - Evidence-based recommendation lookups
- **Phenotype Matching** - Clinical context analysis

### System Performance  
- **Request Rates** - Operations per second per service
- **Response Times** - P50, P95, P99 latencies
- **Error Rates** - Success/failure ratios
- **Cache Performance** - L1/L2/L3 hit rates
- **Database Performance** - Query times by technology

### Compliance & Security
- **Authentication Success Rate** - Login success/failure
- **Authorization Violations** - Access control failures  
- **Rate Limit Violations** - API throttling events
- **License Compliance** - Usage within licensed limits
- **Audit Trail Coverage** - Completeness of audit logs

## 🔧 Management Commands

### Basic Operations
```bash
# Start monitoring stack
make start

# Stop monitoring stack  
make stop

# Restart all services
make restart

# Check service health
make health

# View logs (all services)
make logs

# View logs (specific service)
make logs SERVICE=prometheus
```

### Configuration Management
```bash
# Validate configurations
make validate-config

# Reload configurations (hot reload)
make reload-config

# Update Docker images
make update-images
```

### Maintenance
```bash
# Backup monitoring data
make backup

# Complete cleanup
make clean

# Reset (clean + redeploy)
make reset
```

### Development & Testing
```bash
# Start in development mode
make dev-mode

# Send test alert
make test-alerts

# Generate load for testing
make generate-load
```

## 📊 Metrics Collection

### Service-Level Indicators (SLIs)
- **Availability** - Service uptime percentage
- **Latency** - Response time percentiles (P50, P95, P99)
- **Error Rate** - Percentage of failed requests
- **Throughput** - Requests per second

### Service-Level Objectives (SLOs)
- **Availability SLO** - 99.9% uptime
- **Latency SLO** - 95% of requests < 100ms
- **Error Rate SLO** - < 0.1% error rate

### Recording Rules
Pre-computed metrics for efficient queries:
```promql
# Request rates per service
kb:request_rate_5m = sum(rate(http_requests_total{job=~"kb-.*"}[5m])) by (job)

# Error rates per service  
kb:error_rate_5m = sum(rate(http_requests_total{job=~"kb-.*", status=~"5.."}[5m])) by (job) / 
                   sum(rate(http_requests_total{job=~"kb-.*"}[5m])) by (job)

# Response time percentiles
kb:response_time_p95_5m = histogram_quantile(0.95, 
  sum(rate(http_request_duration_seconds_bucket{job=~"kb-.*"}[5m])) by (job, le))
```

## 🗄️ Database Monitoring Details

### PostgreSQL (KB-1, KB-4, KB-5, KB-7)
- **Connection Pool** - Active/idle/max connections
- **Query Performance** - Slow queries, average execution time
- **Locks** - Lock contention analysis  
- **Replication** - Lag and health for replicas
- **Table Statistics** - Row counts, index usage

### MongoDB (KB-2)
- **Operations** - Insert/update/delete rates
- **Connections** - Active connections and pool usage
- **Memory** - Working set size and cache usage
- **Locks** - Global and database-level locking
- **Sharding** - Chunk distribution (if applicable)

### Neo4j (KB-3)
- **Transactions** - Active and committed transactions
- **Memory** - Heap usage and page cache
- **Store Files** - Node/relationship store sizes
- **Queries** - Cypher query performance
- **Graph Statistics** - Node/relationship counts

### Elasticsearch (KB-6)
- **Cluster Health** - Red/yellow/green status
- **Indexing Rate** - Documents indexed per second
- **Search Performance** - Query latency and throughput
- **Node Statistics** - CPU, memory, disk per node
- **Index Statistics** - Size, document count, shards

### Redis (Cache Layer)
- **Memory Usage** - Used vs. available memory
- **Key Statistics** - Total keys, expired keys
- **Command Statistics** - Commands per second by type
- **Replication** - Master-slave lag
- **Eviction** - Cache eviction events

## 🔒 Security Monitoring

### Authentication & Authorization
- **Login Attempts** - Success/failure rates
- **Session Management** - Active sessions, timeouts
- **JWT Token Validation** - Token verification events
- **Role-Based Access** - Permission check failures

### Rate Limiting & DDoS Protection
- **Rate Limit Violations** - API throttling events
- **Request Patterns** - Unusual traffic patterns
- **IP Blocking** - Blocked IP addresses
- **Geographic Analysis** - Request origin analysis

### Compliance Monitoring
- **HIPAA Compliance** - PHI access logging
- **Audit Trail** - Complete action logging
- **Data Encryption** - Encryption status monitoring
- **License Compliance** - Usage within licensed limits

## 📱 Mobile & API Monitoring

### API Performance
- **Endpoint Latency** - Per-endpoint response times
- **Payload Size** - Request/response sizes
- **API Versioning** - Version usage statistics
- **Client Types** - Mobile vs. web vs. API clients

### Mobile-Specific Metrics
- **App Version Distribution** - Client version adoption
- **Device Performance** - Performance by device type
- **Network Conditions** - 3G/4G/WiFi performance
- **Crash Rates** - Application stability metrics

## 🎯 Capacity Planning

### Growth Trending
- **Request Volume Growth** - 24h/7d/30d trends
- **Resource Utilization Growth** - CPU/memory trends  
- **Database Growth** - Storage usage trends
- **Cache Usage Trends** - Cache hit rate evolution

### Forecasting
- **Linear Regression** - Simple growth forecasting
- **Seasonal Patterns** - Weekly/monthly patterns
- **Capacity Alerts** - Proactive capacity warnings
- **Scale-up Recommendations** - Automated scaling suggestions

## 🔧 Troubleshooting

### Common Issues

#### Services Not Starting
```bash
# Check Docker service status
make status

# View specific service logs
make logs SERVICE=prometheus

# Validate configurations
make validate-config
```

#### Dashboard Not Loading
```bash
# Check Grafana health
curl http://localhost:3000/api/health

# Restart Grafana
docker-compose -f docker-compose.monitoring.yml restart grafana

# Check dashboard provisioning
make logs SERVICE=grafana
```

#### Missing Metrics
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Verify KB services are exposing metrics
curl http://localhost:8081/metrics

# Check Prometheus configuration
make validate-config
```

#### Alerts Not Firing
```bash
# Check Alertmanager status
curl http://localhost:9093/api/v1/status

# Verify alert rules
curl http://localhost:9090/api/v1/rules

# Send test alert
make test-alerts
```

### Performance Optimization

#### High Memory Usage
- Adjust retention periods in `prometheus-unified.yml`
- Optimize recording rules to reduce cardinality
- Configure appropriate resource limits

#### Slow Queries
- Review dashboard query complexity
- Use recording rules for expensive calculations
- Implement query result caching

#### Storage Growth
- Configure data retention policies
- Enable compression in VictoriaMetrics
- Implement log rotation for Loki

## 📚 Additional Resources

### Documentation
- [Prometheus Query Language](https://prometheus.io/docs/prometheus/latest/querying/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)

### Runbooks
- Service Health Runbook: `/docs/runbooks/service-health.md`
- Database Performance: `/docs/runbooks/database-performance.md`  
- Security Incident Response: `/docs/runbooks/security-incidents.md`

### Support
- **Internal Documentation** - See `/docs/` directory
- **Issue Tracking** - Use GitHub issues for bugs/features
- **On-call Support** - Contact platform team via Slack #platform-support

## 🔄 Version History

- **v1.0.0** - Initial unified monitoring deployment
  - All 7 KB services monitoring
  - Multi-database technology support
  - Comprehensive alerting system
  - Security and compliance monitoring

---

**🎉 KB Services Unified Monitoring is now deployed and ready!**

Access your main dashboard at: **http://localhost:3000/d/kb-unified**