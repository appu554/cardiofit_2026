# Medication Service V2 - Monitoring & Observability

## Overview

This document describes the comprehensive monitoring and observability infrastructure for the Medication Service V2, designed with healthcare-grade requirements and HIPAA compliance in mind.

## 🏥 Healthcare-Grade Monitoring Stack

### Core Components

- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization and dashboards
- **Jaeger** - Distributed tracing
- **Alertmanager** - Alert routing and notifications
- **OpenTelemetry Collector** - Telemetry data processing
- **Elasticsearch + Kibana** - Log aggregation and analysis
- **Custom Health Checks** - Multi-level health monitoring

### Key Features

- ✅ **HIPAA Compliant** - Secure, auditable logging and monitoring
- ✅ **Patient Safety Focus** - Zero-tolerance safety violation monitoring
- ✅ **Clinical Decision Tracking** - Complete audit trail for clinical decisions
- ✅ **Performance SLA Monitoring** - <250ms end-to-end response time tracking
- ✅ **Real-time Alerting** - Critical, high, medium, and low severity alerts
- ✅ **Comprehensive Health Checks** - Liveness, readiness, and dependency monitoring

## 🚀 Quick Start

### 1. Start Monitoring Stack

```bash
# Start all monitoring services
docker-compose -f docker-compose.monitoring.yml up -d

# Check service status
docker-compose -f docker-compose.monitoring.yml ps

# View logs for specific service
docker-compose -f docker-compose.monitoring.yml logs -f grafana
```

### 2. Access Monitoring Interfaces

- **Grafana Dashboards**: http://localhost:3000 (admin/admin123)
- **Prometheus**: http://localhost:9090
- **Alertmanager**: http://localhost:9093
- **Jaeger Tracing**: http://localhost:16686
- **Kibana Logs**: http://localhost:5601

### 3. Import Dashboards

Dashboards are automatically provisioned from `config/monitoring/dashboards/`:

1. **Operational Dashboard** - General service health and performance
2. **Clinical Safety Dashboard** - Patient safety and clinical metrics
3. **Performance Dashboard** - SLA and performance monitoring
4. **Security Dashboard** - Security events and compliance

## 📊 Monitoring Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Medication     │    │  Prometheus     │    │  Grafana        │
│  Service V2     │───▶│  (Metrics)      │───▶│  (Visualization)│
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │
         │              ┌─────────────────┐    ┌─────────────────┐
         │              │  Alertmanager   │    │  Notification   │
         │              │  (Alerting)     │───▶│  Channels       │
         │              │                 │    │                 │
         │              └─────────────────┘    └─────────────────┘
         │
         ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  OpenTelemetry  │    │  Jaeger         │    │  Elasticsearch  │
│  Collector      │───▶│  (Tracing)      │    │  + Kibana       │
│                 │    │                 │    │  (Logging)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🔍 Key Metrics

### Patient Safety Metrics
- `medication_service_safety_violations_total` - Safety violations (CRITICAL)
- `medication_service_dosage_calculation_accuracy` - Dosage accuracy (target: >95%)
- `medication_service_validation_failures_total` - Validation failures
- `medication_service_compliance_checks_total` - HIPAA compliance checks

### Performance Metrics
- `medication_service_http_request_duration_seconds` - API latency (target: <250ms P95)
- `medication_service_proposal_processing_duration_seconds` - Proposal processing time
- `medication_service_recipe_resolution_duration_seconds` - Recipe resolution time
- `medication_service_clinical_calculation_duration_seconds` - Clinical calculation time

### Business Metrics
- `medication_service_proposals_total` - Total medication proposals
- `medication_service_cache_hit_ratio` - Cache performance
- `medication_service_database_connections_active` - Database health
- `medication_service_errors_total` - Error rates by type

### Clinical Data Quality
- `medication_service_clinical_data_freshness_score` - Data freshness (target: >85%)
- `medication_service_patient_context_age_seconds` - Patient data age distribution

## 🚨 Alerting

### Alert Severity Levels

#### 🔴 CRITICAL (Immediate Response Required)
- Patient safety violations
- Service completely down
- Database connectivity failure
- Dosage calculation accuracy <95%
- Audit trail failures

#### 🟠 HIGH (Response within 15 minutes)
- API response time >250ms P95
- Error rate >5%
- Clinical processing delays >150ms
- Data freshness <80%

#### 🟡 MEDIUM (Response within 1 hour)
- Moderate performance degradation
- Cache hit ratio <80%
- External service failures

#### 🟢 LOW/INFO (Monitor and document)
- Request rate anomalies
- Resource utilization warnings

### Notification Channels

1. **Critical Alerts**: PagerDuty → On-call engineer
2. **High Alerts**: Slack + Email → Team leads
3. **Medium/Low Alerts**: Email → Team distribution list

### Escalation Rules

1. **Level 1** (5 minutes): Notify on-call engineer via Slack
2. **Level 2** (15 minutes): Escalate to team lead via email + Slack
3. **Level 3** (30 minutes): Page director via PagerDuty

## 🏥 Healthcare Compliance

### HIPAA Compliance Features

- **Audit Trail**: All clinical decisions logged with correlation IDs
- **Data Sanitization**: PHI automatically sanitized in logs
- **Access Logging**: All patient data access tracked
- **Retention Policy**: 7-year retention for clinical data
- **Encryption**: All monitoring data encrypted in transit and at rest

### Clinical Safety Monitoring

- **Zero Tolerance**: Any safety violation triggers immediate critical alert
- **Drug Interactions**: Comprehensive interaction checking monitoring
- **Dosage Validation**: Real-time accuracy tracking
- **Clinical Decision Audit**: Complete decision trail with reasoning

### Compliance Reports

Generate compliance reports using Grafana:

1. **HIPAA Audit Report** - Access logs and security events
2. **Clinical Safety Report** - Safety violations and accuracy metrics
3. **Performance SLA Report** - Service level agreement compliance
4. **Data Quality Report** - Clinical data freshness and completeness

## 📋 Health Checks

### Multi-Level Health Checks

#### Liveness Check (`/health/live`)
- Basic service responsiveness
- Used by Kubernetes liveness probe

#### Readiness Check (`/health/ready`)
- Database connectivity
- Redis connectivity
- External service availability
- Used by Kubernetes readiness probe

#### Detailed Health Check (`/health/status`)
- Comprehensive system status
- Dependency health scoring
- Clinical system status
- Performance metrics

### Health Scoring Algorithm

```
Overall Health Score = (
  Database Health * 0.4 +
  Cache Health * 0.1 +
  External APIs * 0.3 +
  Internal Services * 0.2
)
```

## 🔧 Configuration

### Environment Variables

```bash
# Tracing
JAEGER_ENDPOINT=http://jaeger-collector:14268/api/traces
OTEL_SERVICE_NAME=medication-service-v2
OTEL_SERVICE_VERSION=2.0.0

# Alerting
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
PAGERDUTY_API_KEY=your-pagerduty-key
PAGERDUTY_SERVICE_KEY=your-service-key

# Database
DB_HOST=localhost
DB_PORT=5432
REDIS_HOST=localhost
REDIS_PORT=6379

# Healthcare Compliance
ENABLE_HIPAA_LOGGING=true
ENABLE_AUDIT_TRAIL=true
LOG_RETENTION_DAYS=2555  # 7 years
```

### Configuration Files

- `config/monitoring/monitoring.yaml` - Main monitoring configuration
- `config/monitoring/logging.yaml` - Logging configuration
- `config/monitoring/alerting/prometheus-rules.yml` - Prometheus alerting rules
- `config/monitoring/prometheus/prometheus.yml` - Prometheus configuration

## 🎯 Performance Targets

### SLA Targets
- **Availability**: 99.9% (8.76 hours downtime per year)
- **Latency P95**: <250ms
- **Latency P99**: <500ms
- **Error Rate**: <0.1%

### Clinical Performance
- **Recipe Resolution**: <50ms average
- **Context Snapshot Creation**: <100ms average
- **Clinical Calculation**: <75ms average
- **Proposal Generation**: <200ms average

### Data Quality Targets
- **Data Freshness**: >85%
- **Data Completeness**: >95%
- **Data Accuracy**: >98%
- **Dosage Calculation Accuracy**: >95%

## 🛠 Maintenance

### Daily Checks
- Review critical alerts
- Check system health status
- Verify SLA compliance
- Review clinical safety metrics

### Weekly Tasks
- Analyze performance trends
- Review alert thresholds
- Update dashboards as needed
- Check log retention

### Monthly Tasks
- Generate compliance reports
- Review and update alert rules
- Performance optimization analysis
- Capacity planning review

### Quarterly Tasks
- Full system health review
- Update monitoring documentation
- Security assessment
- Disaster recovery testing

## 🚨 Troubleshooting

### Common Issues

#### High Memory Usage
```bash
# Check container memory usage
docker stats medication-service-v2

# Review memory metrics
curl http://localhost:9090/api/v1/query?query=process_resident_memory_bytes
```

#### Database Connection Issues
```bash
# Check database connectivity
curl http://localhost:8080/health/ready

# Review database metrics
curl http://localhost:9090/api/v1/query?query=medication_service_database_connections_active
```

#### Alert Fatigue
1. Review alert thresholds in `prometheus-rules.yml`
2. Adjust severity levels based on actual impact
3. Implement alert silencing for known issues
4. Use alert grouping to reduce noise

### Log Analysis

#### Search Clinical Events
```bash
# In Kibana, search for:
event_type:"clinical_decision" AND outcome:"success"
```

#### Find Safety Violations
```bash
# In Kibana, search for:
event_category:"safety_violation" AND severity:"critical"
```

#### Trace Request Flow
1. Get correlation ID from logs
2. Search in Jaeger: `http://localhost:16686`
3. Analyze span timings and errors

## 📚 Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/languages/go/)
- [HIPAA Compliance Guide](https://www.hhs.gov/hipaa/)
- [Healthcare Data Security Best Practices](https://www.healthit.gov/topic/privacy-security-and-hipaa)

## 🤝 Support

For monitoring and observability issues:
1. Check the troubleshooting section above
2. Review Grafana dashboards for system health
3. Search logs in Kibana for error details
4. Contact the platform team for escalation

---

**Remember**: In healthcare environments, patient safety is the top priority. Any safety-related alerts should be treated as critical and addressed immediately.