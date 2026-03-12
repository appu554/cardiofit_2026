# Module 6: Analytics Dashboard API

Comprehensive GraphQL API for real-time clinical analytics and predictive dashboards.

## 📋 Overview

The Analytics Dashboard API provides real-time access to:
- **Hospital-wide KPIs**: Patient counts, risk distributions, alert metrics
- **Department metrics**: Per-department performance and risk assessment
- **Patient risk profiles**: Individual patient risk scores and predictions
- **Time-series analytics**: Vital sign trends and historical patterns
- **Alert management**: Active alerts, metrics, and resolution tracking
- **ML model performance**: Prediction accuracy and drift detection
- **Quality metrics**: Protocol adherence and safety indicators

## 🏗️ Architecture

```
Flink Module 6 → Kafka Topics → Analytics Service → Redis Cache
                                        ↓
                                  GraphQL API
                                        ↓
                      Apollo Federation Gateway (port 4000)
```

### Data Flow

1. **Flink Streaming Analytics** (Module 6) produces metrics to Kafka:
   - `analytics-population-health`: Department-level metrics every 60s
   - `inference-results.v1`: ML predictions for patient risk profiles
   - `analytics-vital-timeseries`: Vital sign aggregations

2. **Analytics Service** (port 8050):
   - Consumes Kafka topics in real-time
   - Caches metrics in Redis (5-min TTL)
   - Aggregates hospital-wide KPIs every 60s
   - Exposes GraphQL API

3. **Apollo Federation Gateway** (port 4000):
   - Composes Analytics Service with other microservices
   - Provides unified GraphQL endpoint

## 🚀 Quick Start

### Prerequisites

```bash
# Required services
✅ Redis running on localhost:6379
✅ Kafka running on localhost:9092
✅ Flink Module 6 Analytics Engine running

# Optional (for full functionality)
□ PostgreSQL on localhost:5432 (for historical data)
```

### Installation

```bash
cd apollo-federation

# Install dependencies (already available)
npm install

# Optional: Add PostgreSQL and Kafka drivers
npm install pg kafkajs
```

### Running the Service

```bash
# Start standalone analytics service
npm run start:analytics

# Or with auto-reload during development
npm run dev:analytics
```

Service will start on: **http://localhost:8050/graphql**

### Testing

```bash
# Run test suite
npm run test:analytics

# Manual testing via GraphQL Playground
open http://localhost:8050/graphql
```

## 📊 GraphQL Schema

### Core Queries

#### 1. Hospital-Wide KPIs

```graphql
query {
  hospitalKPIs {
    timestamp
    totalPatients
    highRiskPatients
    criticalPatients
    avgMortalityRisk
    avgSepsisRisk
    avgReadmissionRisk
    activeAlerts
    criticalAlerts
    modelAccuracy
  }
}
```

#### 2. Department Metrics

```graphql
query {
  allDepartmentMetrics {
    department
    totalPatients
    highRiskPatients
    criticalPatients
    avgMortalityRisk
    avgSepsisRisk
    riskDistribution {
      LOW
      MODERATE
      HIGH
      CRITICAL
    }
    departmentRiskLevel
    highRiskPercentage
    criticalPercentage
    overallRiskScore
    requiresImmediateAttention
  }
}
```

#### 3. Specific Department

```graphql
query {
  departmentMetrics(department: "ICU") {
    department
    totalPatients
    highRiskPatients
    departmentRiskLevel
    requiresImmediateAttention
  }
}
```

#### 4. High-Risk Patients

```graphql
query {
  highRiskPatients(limit: 20) {
    patientId
    department
    mortalityRisk
    sepsisRisk
    readmissionRisk
    overallRiskScore
    riskLevel
    isHighRisk
    isCritical
    lastUpdated
  }
}
```

#### 5. Patient Risk Profile

```graphql
query {
  patientRiskProfile(patientId: "PAT-001") {
    patientId
    department
    mortalityRisk
    sepsisRisk
    readmissionRisk
    overallRiskScore
    riskLevel
    lastUpdated
  }
}
```

#### 6. Alert Metrics

```graphql
query {
  alertMetrics(
    startTime: "2025-11-08T00:00:00Z"
    endTime: "2025-11-08T23:59:59Z"
  ) {
    totalAlerts
    criticalAlerts
    warningAlerts
    avgResolutionTime
  }
}
```

#### 7. Health Check

```graphql
query {
  analyticsHealth {
    status
    timestamp
    redisConnected
    postgresConnected
    kafkaConnected
  }
}
```

### Subscriptions (Future)

```graphql
subscription {
  hospitalKPIsUpdated {
    totalPatients
    highRiskPatients
    criticalPatients
  }
}

subscription {
  departmentMetricsUpdated(department: "ICU") {
    totalPatients
    requiresImmediateAttention
  }
}
```

## 🔧 Configuration

### Environment Variables

```bash
# Analytics Service Port
ANALYTICS_SERVICE_PORT=8050

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0

# PostgreSQL Configuration (optional)
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=analytics_db
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# Kafka Configuration
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=analytics-dashboard-api
```

## 📦 Project Structure

```
apollo-federation/
├── schemas/
│   └── analytics-dashboard-schema.graphql    # GraphQL type definitions
├── resolvers/
│   └── analytics-dashboard-resolvers.js      # Query/mutation resolvers
├── services/
│   ├── analytics-data-service.js            # Data access layer (Redis/PG/Kafka)
│   └── analytics-service.js                 # Standalone GraphQL service
├── test-analytics-dashboard.js              # Test suite
└── ANALYTICS_DASHBOARD_README.md            # This file
```

## 🔄 Data Lifecycle

### Real-Time Updates (Every 60 seconds)

1. **Flink Module 6** produces population health metrics to Kafka
2. **Analytics Service** consumes Kafka messages
3. **Redis cache** is updated with latest metrics (5-min TTL)
4. **Hospital KPIs** are recalculated from all departments
5. **GraphQL queries** return cached data with <5s latency

### Cache Strategy

- **Redis TTL**: 5 minutes for all cached metrics
- **Automatic refresh**: Kafka consumers continuously update cache
- **Graceful degradation**: Returns empty data if cache is stale

## 🧪 Testing

### Manual Testing with curl

```bash
# Health check
curl -X POST http://localhost:8050/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ analyticsHealth { status redisConnected } }"}'

# Hospital KPIs
curl -X POST http://localhost:8050/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ hospitalKPIs { totalPatients highRiskPatients } }"}'

# Department metrics
curl -X POST http://localhost:8050/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ allDepartmentMetrics { department totalPatients } }"}'
```

### Automated Test Suite

```bash
# Run comprehensive test suite
npm run test:analytics

# Expected output:
# ✅ Health Check
# ✅ Hospital KPIs
# ✅ Department Metrics
# ✅ Specific Department
# ✅ High-Risk Patients
# ✅ Alert Metrics
```

## 🔗 Integration with Apollo Federation Gateway

To add the analytics service to the main federation gateway:

```javascript
// In index.js
const federationServices = [
  { name: 'analytics', url: 'http://localhost:8050/graphql' },
  { name: 'patients', url: 'http://localhost:8003/api/federation' },
  // ... other services
];
```

Then query through the gateway:

```bash
curl -X POST http://localhost:4000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ hospitalKPIs { totalPatients } }"}'
```

## 📈 Performance Considerations

### Redis Cache Benefits
- **Sub-second query latency** for all dashboard queries
- **Reduced Kafka consumer lag** by buffering real-time data
- **Scalable read operations** with minimal backend load

### Optimization Tips
1. **Batch queries**: Use GraphQL fragments to fetch related data
2. **Limit result sets**: Use `limit` parameter on patient queries
3. **Monitor Redis memory**: Set appropriate TTLs for cache cleanup
4. **Index PostgreSQL**: Create indexes on timestamp columns for historical queries

## 🐛 Troubleshooting

### Service won't start

```bash
# Check Redis connection
redis-cli ping  # Should return PONG

# Check Kafka connection
docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092

# Check port availability
lsof -i :8050  # Should be empty
```

### No data in responses

```bash
# Verify Flink Module 6 is running
docker logs flink-jobmanager

# Check Kafka topics have data
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic analytics-population-health \
  --from-beginning --max-messages 1

# Check Redis cache
redis-cli KEYS "analytics:*"
redis-cli GET "analytics:hospital:kpis"
```

### PostgreSQL not connecting

```bash
# Install driver
npm install pg

# Test connection
psql -h localhost -U postgres -d analytics_db -c "SELECT 1"
```

## 🚀 Future Enhancements

### Phase 2: Real-Time Subscriptions
- WebSocket support for live dashboard updates
- GraphQL subscriptions for alert notifications
- Server-Sent Events (SSE) for streaming metrics

### Phase 3: Historical Analytics
- PostgreSQL time-series tables for trend analysis
- Multi-resolution time-series queries (1min/5min/1hr/1day)
- Comparative analytics (yesterday vs today)

### Phase 4: Advanced Features
- Alerting rules engine
- Predictive analytics API
- Custom dashboard configurations
- Export APIs (CSV, PDF reports)

## 📚 Related Documentation

- [Module 6 Analytics Engine](../backend/shared-infrastructure/flink-processing/src/docs/module_6/Module_6_Advanced_Analytics_&_Predictive_Dashboards.txt)
- [Flink Analytics Implementation](../backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/analytics/)
- [Apollo Federation Guide](./FEDERATION_SETUP.md)

## 🤝 Support

For issues or questions:
1. Check logs: `docker logs analytics-service`
2. Verify health: `curl http://localhost:8050/graphql -d '{"query":"{ analyticsHealth { status } }"}'`
3. Review test output: `npm run test:analytics`

---

**Status**: ✅ Ready for Integration Testing
**Version**: 1.0.0
**Last Updated**: November 2025
