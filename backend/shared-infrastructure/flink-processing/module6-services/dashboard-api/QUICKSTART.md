# Dashboard API - Quick Start Guide

Get the Dashboard API running in 5 minutes with full infrastructure.

## Prerequisites

- Docker and Docker Compose installed
- Node.js 18+ (for local development)
- 4GB RAM available

## Option 1: Docker Compose (Recommended)

### Start Everything

```bash
# From dashboard-api directory
docker-compose up -d
```

This starts:
- Dashboard API (port 4000)
- Kafka + Zookeeper
- Redis (port 6379)
- PostgreSQL (port 5432)
- InfluxDB (port 8086)

### Check Status

```bash
# Check all services
docker-compose ps

# Check API health
curl http://localhost:4000/health

# View logs
docker-compose logs -f dashboard-api
```

### Access GraphQL Playground

Open browser: http://localhost:4000/graphql

### Stop Everything

```bash
docker-compose down

# To remove volumes (data)
docker-compose down -v
```

## Option 2: Local Development

### 1. Install Dependencies

```bash
npm install
```

### 2. Start Infrastructure

```bash
# Start only infrastructure services
docker-compose up -d zookeeper kafka redis postgres influxdb

# Wait 30 seconds for services to be ready
sleep 30
```

### 3. Configure Environment

```bash
cp .env.example .env

# Edit .env with local settings:
# KAFKA_BROKERS=localhost:9092
# REDIS_HOST=localhost
# POSTGRES_HOST=localhost
# INFLUX_URL=http://localhost:8086
```

### 4. Run in Development Mode

```bash
npm run dev
```

The API starts at http://localhost:4000 with hot reload enabled.

## Testing the API

### Create Kafka Topics

```bash
# Connect to Kafka container
docker exec -it dashboard-api-kafka-1 bash

# Create topics
kafka-topics --create --topic hospital-kpis --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1
kafka-topics --create --topic department-metrics --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1
kafka-topics --create --topic patient-risk-profiles --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1
kafka-topics --create --topic sepsis-surveillance --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1
kafka-topics --create --topic quality-metrics --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1

exit
```

### Send Test Data

```bash
# Use the Kafka console producer
docker exec -it dashboard-api-kafka-1 bash

kafka-console-producer --topic hospital-kpis --bootstrap-server localhost:9092

# Paste this JSON (one line):
{"hospitalId":"HOSP001","timestamp":"2025-11-04T10:00:00Z","windowStart":"2025-11-04T09:00:00Z","windowEnd":"2025-11-04T10:00:00Z","totalBeds":500,"occupiedBeds":425,"availableBeds":75,"occupancyRate":0.85,"icuOccupancyRate":0.92,"totalAdmissions":45,"totalDischarges":38,"averageLengthOfStay":4.2,"readmissionRate":0.08,"mortalityRate":0.02,"adverseEventRate":0.03,"infectionRate":0.05,"averageWaitTime":32.5,"bedTurnoverRate":0.15,"staffUtilizationRate":0.78,"patientSatisfactionScore":4.3,"clinicalQualityScore":0.89}

# Press Ctrl+C, then exit
```

### Query the Data

GraphQL Playground at http://localhost:4000/graphql:

```graphql
query {
  hospitalKpis(hospitalId: "HOSP001") {
    hospitalId
    timestamp
    occupancyRate
    icuOccupancyRate
    totalAdmissions
    totalDischarges
    mortalityRate
    patientSatisfactionScore
  }
}
```

### Test Dashboard Summary

```graphql
query {
  dashboardSummary(hospitalId: "HOSP001") {
    timestamp
    hospitalKpis {
      occupancyRate
      totalAdmissions
      availableBeds
    }
    realtimeStats {
      activePatients
      availableBeds
      lastUpdated
    }
  }
}
```

## Health Check Endpoints

```bash
# Full health check
curl http://localhost:4000/health | jq

# Readiness probe
curl http://localhost:4000/ready

# Liveness probe
curl http://localhost:4000/live

# Metrics
curl http://localhost:4000/metrics | jq
```

## Example Queries for Each Data Type

### 1. Hospital KPIs

```graphql
query HospitalPerformance {
  hospitalKpis(hospitalId: "HOSP001") {
    occupancyRate
    icuOccupancyRate
    averageLengthOfStay
    readmissionRate
    mortalityRate
    patientSatisfactionScore
    clinicalQualityScore
  }
}
```

### 2. Department Metrics

```graphql
query DepartmentStatus {
  departmentMetrics(hospitalId: "HOSP001") {
    departmentName
    occupancyRate
    currentPatients
    staffingLevel
    criticalAlerts
    warningAlerts
  }
}
```

### 3. High Risk Patients

```graphql
query CriticalPatients {
  highRiskPatients(
    hospitalId: "HOSP001"
    riskLevel: CRITICAL
    limit: 10
  ) {
    patientId
    overallRiskScore
    riskLevel
    mortalityRisk
    sepsisRisk
    activeAlerts {
      alertType
      severity
      message
    }
    recommendedInterventions
  }
}
```

### 4. Sepsis Surveillance

```graphql
query SepsisMonitoring {
  sepsisSurveillance(
    hospitalId: "HOSP001"
    alertLevel: CRITICAL
  ) {
    alertId
    patientId
    sepsisStage
    sofaScore
    qSofaScore
    bundleCompliance {
      overallCompliance
      lactateDrawn
      antibioticsAdministered
      fluidResuscitationStarted
    }
    timeToIntervention
  }
}
```

### 5. Quality Metrics

```graphql
query QualityPerformance {
  qualityMetrics(
    hospitalId: "HOSP001"
    metricType: SEPSIS_BUNDLE_COMPLIANCE
  ) {
    metricName
    metricValue
    targetValue
    performanceStatus
    trendDirection
    complianceRate
  }

  complianceScore(hospitalId: "HOSP001") {
    overallScore
    trendDirection
    categoryScores
  }
}
```

### 6. Real-time Dashboard

```graphql
query LiveDashboard {
  realtimeStats(hospitalId: "HOSP001") {
    activePatients
    criticalPatients
    activeSepsisAlerts
    availableBeds
    staffOnDuty
    lastUpdated
  }
}
```

## Monitoring

### View Logs

```bash
# API logs
docker-compose logs -f dashboard-api

# Kafka logs
docker-compose logs -f kafka

# All services
docker-compose logs -f
```

### Database Access

```bash
# PostgreSQL
docker exec -it dashboard-api-postgres-1 psql -U postgres -d clinical_analytics

# Check tables
\dt

# Query data
SELECT hospital_id, timestamp, data->'occupancyRate' as occupancy
FROM hospital_kpis
ORDER BY timestamp DESC LIMIT 5;
```

### Redis Cache Inspection

```bash
# Connect to Redis
docker exec -it dashboard-api-redis-1 redis-cli

# List all keys
KEYS *

# Get cached data
GET hospital-kpis:HOSP001:latest

# Check TTL
TTL hospital-kpis:HOSP001:latest
```

## Performance Testing

### Load Test with Apache Bench

```bash
# Install Apache Bench
# macOS: brew install httpie
# Ubuntu: apt-get install apache2-utils

# Load test health endpoint
ab -n 1000 -c 10 http://localhost:4000/health

# Load test GraphQL (create query.json first)
ab -n 100 -c 5 -T 'application/json' -p query.json http://localhost:4000/graphql
```

### Monitor Resource Usage

```bash
# Container stats
docker stats

# API memory usage
curl http://localhost:4000/metrics | jq '.memory'
```

## Troubleshooting

### Kafka Connection Failed

```bash
# Check Kafka is running
docker-compose ps kafka

# Check Kafka logs
docker-compose logs kafka

# Restart Kafka
docker-compose restart kafka
```

### Database Connection Error

```bash
# Check PostgreSQL
docker-compose ps postgres

# Test connection
docker exec -it dashboard-api-postgres-1 psql -U postgres -c "SELECT 1"

# Restart database
docker-compose restart postgres
```

### Redis Connection Error

```bash
# Check Redis
docker-compose ps redis

# Test connection
docker exec -it dashboard-api-redis-1 redis-cli PING

# Restart Redis
docker-compose restart redis
```

### API Not Starting

```bash
# Check logs
docker-compose logs dashboard-api

# Rebuild container
docker-compose build --no-cache dashboard-api
docker-compose up -d dashboard-api

# Check health
curl http://localhost:4000/health
```

## Production Deployment

### Build Production Image

```bash
docker build -t dashboard-api:1.0.0 .
```

### Environment Variables

Required for production:
- `KAFKA_BROKERS` - Kafka cluster addresses
- `REDIS_HOST` - Redis server
- `POSTGRES_*` - Database credentials
- `INFLUX_*` - InfluxDB credentials
- `CORS_ORIGIN` - Frontend URL
- `LOG_LEVEL=warn` - Reduce log verbosity
- `GRAPHQL_PLAYGROUND=false` - Disable playground

### Kubernetes Deployment

See README.md for Kubernetes manifests.

### Security Checklist

- [ ] Change all default passwords
- [ ] Enable TLS for Kafka
- [ ] Enable Redis authentication
- [ ] Use secrets for credentials
- [ ] Configure CORS properly
- [ ] Enable rate limiting
- [ ] Set up monitoring alerts

## Next Steps

1. Integrate with existing Flink jobs (Module 6)
2. Connect to real Kafka topics
3. Add authentication/authorization
4. Configure alerts and monitoring
5. Set up backup for databases
6. Deploy to production environment

## Support

For issues or questions:
- Check logs: `docker-compose logs -f`
- Review README.md for detailed documentation
- Check health endpoint: http://localhost:4000/health
