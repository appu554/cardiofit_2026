# SLA Monitoring Service

Real-time SLA monitoring and alerting system for CardioFit Runtime Layer services with automated compliance tracking, violation detection, and multi-channel alerting.

## Features

- **Real-time Monitoring**: Continuous SLA compliance tracking with 30-second evaluation cycles
- **Multi-Metric Support**: Availability, response time, error rate, throughput, cache hit rate, and ML prediction accuracy
- **Advanced Alerting**: Multi-channel alerting (Slack, email, webhook) with severity-based escalation
- **Violation Tracking**: Comprehensive violation lifecycle management with grace periods and auto-resolution
- **Historical Analysis**: Trend analysis and compliance reporting with 30-day retention
- **RESTful API**: Complete API for SLA configuration, monitoring, and reporting
- **Dashboard Integration**: Pre-built Grafana dashboards for visualization
- **Service Discovery**: Auto-discovery of runtime layer services via Prometheus and health endpoints

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  SLA Monitoring Service                     │
├─────────────────┬─────────────────┬─────────────────────────┤
│   FastAPI App   │  SLA Monitor    │   Alert Manager        │
│   (Port 8050)   │  (Evaluator)    │   (Multi-channel)      │
└─────────────────┴─────────────────┴─────────────────────────┘
         │                │                        │
         ▼                ▼                        ▼
┌─────────────────┬─────────────────┬─────────────────────────┐
│  Metrics        │   Configuration │     Alert Channels     │
│  Collector      │   Manager       │   (Slack/Email/Hook)   │
└─────────────────┴─────────────────┴─────────────────────────┘
         │                │                        │
         ▼                ▼                        ▼
┌─────────────────┬─────────────────┬─────────────────────────┐
│   Prometheus    │    MongoDB      │      Grafana           │
│   (Metrics)     │   (SLA Data)    │   (Visualization)      │
└─────────────────┴─────────────────┴─────────────────────────┘
```

## SLA Targets

The service monitors these default SLA targets:

### Flink Stream Processor
- **Availability**: ≥99.9% (Critical)
- **Response Time**: ≤500ms (High)

### Evidence Envelope Service
- **Availability**: ≥99.95% (Critical)
- **Response Time**: ≤200ms (High)

### L1 Cache Prefetcher Service
- **Availability**: ≥99.99% (Critical)
- **Response Time**: ≤10ms (High)
- **Cache Hit Rate**: ≥85% (Medium)
- **ML Prediction Accuracy**: ≥70% (Medium)

## Quick Start

### Using Docker Compose (Recommended)

1. **Start the complete monitoring stack:**
   ```bash
   cd backend/shared-infrastructure/runtime-layer
   docker-compose -f docker-compose.sla-monitoring.yml up -d
   ```

2. **Verify services are running:**
   ```bash
   curl http://localhost:8050/health
   curl http://localhost:8050/health/ready
   ```

3. **Access monitoring interfaces:**
   - SLA API: http://localhost:8050/docs
   - Grafana: http://localhost:3003 (admin/sla_grafana_admin_password)
   - Prometheus: http://localhost:9092
   - AlertManager: http://localhost:9093

### Manual Setup

1. **Install dependencies:**
   ```bash
   cd sla-monitoring-service
   pip install -r requirements.txt
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Start dependencies:**
   ```bash
   # MongoDB
   docker run -d --name mongodb-sla -p 27020:27017 mongo:7.0

   # Redis
   docker run -d --name redis-sla -p 6381:6379 redis:7.2-alpine

   # Prometheus
   docker run -d --name prometheus-sla -p 9092:9090 prom/prometheus
   ```

4. **Run the service:**
   ```bash
   python -m uvicorn src.main:app --host 0.0.0.0 --port 8050 --reload
   ```

## API Endpoints

### Health & Monitoring
- `GET /health` - Basic health check
- `GET /health/ready` - Readiness check with dependencies
- `GET /metrics` - Prometheus metrics

### SLA Configuration
- `GET /api/v1/configuration` - Get current SLA configuration
- `PUT /api/v1/configuration` - Update SLA configuration
- `GET /api/v1/targets` - List SLA targets (filterable)
- `POST /api/v1/targets` - Create new SLA target
- `DELETE /api/v1/targets/{target_id}` - Delete SLA target

### Monitoring & Reporting
- `GET /api/v1/dashboard` - Get dashboard data
- `GET /api/v1/violations` - List SLA violations (filterable)
- `GET /api/v1/reports/{service_name}` - Generate service SLA report
- `POST /api/v1/evaluate` - Trigger manual SLA evaluation

### Alert Management
- `GET /api/v1/alerts` - List active alerts
- `POST /api/v1/alerts/{alert_id}/resolve` - Resolve alert

## Usage Examples

### Creating Custom SLA Target

```bash
curl -X POST "http://localhost:8050/api/v1/targets" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "custom-service",
    "metric_type": "availability",
    "target_value": 99.5,
    "operator": "gte",
    "unit": "percent",
    "measurement_window_minutes": 5,
    "evaluation_frequency_seconds": 30,
    "severity": "high"
  }'
```

### Getting Service Health Report

```bash
curl "http://localhost:8050/api/v1/reports/l1-cache-prefetcher-service?hours=24" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Response includes:
{
  "service_name": "l1-cache-prefetcher-service",
  "overall_compliance_percentage": 98.7,
  "uptime_percentage": 99.99,
  "total_violations": 2,
  "critical_violations": 0,
  "metric_summaries": [...],
  "sla_status": "warning"
}
```

### Triggering Manual Evaluation

```bash
curl -X POST "http://localhost:8050/api/v1/evaluate?service_name=flink-stream-processor" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | INFO | Logging level |
| `MONGODB_URL` | mongodb://localhost:27017 | MongoDB connection URL |
| `REDIS_URL` | redis://localhost:6379/0 | Redis connection URL |
| `JWT_SECRET` | cardiofit-sla-secret | JWT signing secret |
| `SLA_ADMIN_USERS` | admin,sre-team | Admin user list |
| `SLA_ALERT_EMAILS` | | Default alert email recipients |
| `SLACK_WEBHOOK_URL` | | Slack webhook for alerts |

### SLA Configuration File

SLA targets are managed via configuration file (JSON/YAML):

```json
{
  "enabled": true,
  "default_measurement_window_minutes": 5,
  "default_evaluation_frequency_seconds": 30,
  "alert_cooldown_minutes": 10,
  "violation_grace_period_minutes": 2,
  "targets": [...],
  "alert_channels": ["email", "slack"],
  "measurement_retention_days": 30
}
```

## Monitoring Stack

### Prometheus Metrics

The service exposes these key metrics:

- `sla_evaluations_total` - Total SLA evaluations by service/status
- `sla_violations_active` - Active violations by service/severity
- `sla_compliance_percentage` - Compliance percentage by service/metric
- `sla_monitoring_response_time_seconds` - API response times

### Grafana Dashboards

Pre-built dashboards include:

1. **SLA Overview Dashboard**
   - System health score
   - Active violations count
   - Service availability gauges
   - Compliance trends
   - Response time analysis

2. **Service Drill-down**
   - Per-service detailed metrics
   - Historical violation analysis
   - Performance trends

3. **Alert Management**
   - Active alerts timeline
   - Alert frequency analysis
   - Resolution tracking

### AlertManager Integration

Advanced alerting rules for:
- Critical SLA violations (immediate)
- Degradation warnings (trend-based)
- Service unavailability
- Evaluation failures

## Authentication

### JWT Token Authentication

All API endpoints require JWT authentication:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### User Roles

- **Admin Users**: Full SLA management access
- **Service Accounts**: Automated monitoring operations
- **Viewers**: Read-only access to health and metrics

### Creating Service Tokens

```python
from src.middleware.auth_middleware import AuthMiddleware

auth = AuthMiddleware()
token = auth.create_service_token("prometheus", expires_hours=24)
```

## Development

### Running Tests

```bash
# Unit tests
pytest tests/unit/ -v

# Integration tests
pytest tests/integration/ -v

# Full test suite with coverage
pytest --cov=src tests/ --cov-report=html
```

### Code Quality

```bash
# Linting and formatting
black src/ tests/
isort src/ tests/
flake8 src/ tests/
mypy src/
```

### Local Development Setup

```bash
# Install development dependencies
pip install -r requirements.txt -r requirements-dev.txt

# Start development stack
docker-compose -f docker-compose.sla-monitoring.yml up mongodb-sla redis-sla prometheus-sla

# Run service with hot reload
uvicorn src.main:app --reload --host 0.0.0.0 --port 8050
```

## Troubleshooting

### Common Issues

1. **Service evaluation failures**
   - Check service health endpoints are accessible
   - Verify Prometheus metrics are being scraped
   - Review service endpoint configuration

2. **Alert delivery issues**
   - Validate Slack webhook URL
   - Check email SMTP configuration
   - Review alert cooldown settings

3. **High memory usage**
   - Check metrics collection frequency
   - Review cache TTL settings
   - Monitor MongoDB storage growth

### Health Diagnostics

```bash
# Check service readiness
curl http://localhost:8050/health/ready

# View current configuration
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8050/api/v1/configuration

# Manual evaluation test
curl -X POST -H "Authorization: Bearer TOKEN" \
  http://localhost:8050/api/v1/evaluate
```

### Log Analysis

```bash
# Service logs
docker logs cardiofit-sla-monitoring

# MongoDB logs
docker logs cardiofit-mongodb-sla

# Check metrics collection
grep "metrics_collection_completed" /app/logs/sla-monitoring.log
```

## Performance Characteristics

- **Evaluation Cycle**: 30-second default frequency
- **Metric Collection**: <2 seconds for all services
- **API Response Time**: <200ms for most endpoints
- **Memory Usage**: ~100MB baseline + 50MB per 10,000 measurements
- **Storage**: ~1MB per service per day of measurements
- **Concurrent Users**: 100+ with JWT caching

## Security

### Data Protection
- JWT-based authentication with role-based access
- TLS encryption for all external communications
- MongoDB authentication and authorization
- Secrets management via environment variables

### Access Control
- Admin-only configuration management
- Service account isolation
- Rate limiting on API endpoints
- Audit logging for all configuration changes

## Production Deployment

### Infrastructure Requirements
- **CPU**: 2 cores minimum, 4 cores recommended
- **Memory**: 2GB minimum, 4GB recommended
- **Storage**: 10GB minimum for 30-day retention
- **Network**: Access to all monitored services

### High Availability Setup
```bash
# Deploy with multiple replicas
docker-compose -f docker-compose.sla-monitoring.yml up -d --scale sla-monitoring-service=3

# External load balancer required for HA
```

### Backup Strategy
```bash
# MongoDB backup
docker exec cardiofit-mongodb-sla mongodump --out /backup

# Configuration backup
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8050/api/v1/configuration > sla-config-backup.json
```

## License

Proprietary - CardioFit Clinical Synthesis Hub