# Dashboard API Service

Production-ready GraphQL API for real-time clinical analytics dashboards. Consumes analytics from Kafka topics and provides unified access via GraphQL with Redis caching and multi-database persistence.

## Architecture

```
Kafka Topics → Consumers → Redis Cache (5-min TTL)
                        ↓
                   PostgreSQL (Historical)
                        ↓
                   InfluxDB (Time-series)
                        ↓
                Apollo GraphQL Server → Clients
```

## Features

- **Apollo Server GraphQL API** - Type-safe, self-documenting API
- **Real-time Data** - Kafka consumers for 5 analytics topics
- **Multi-layer Caching** - Redis for hot data (5-min TTL)
- **Historical Storage** - PostgreSQL for queryable analytics
- **Time-series Metrics** - InfluxDB for temporal analysis
- **Complete Schema** - Hospital KPIs, Department Metrics, Patient Risk, Sepsis Surveillance, Quality Metrics
- **Health Monitoring** - Health, readiness, liveness probes
- **Production-ready** - Graceful shutdown, error handling, logging
- **Docker Support** - Multi-stage build, health checks

## Prerequisites

- Node.js >= 18.0.0
- Kafka cluster running
- Redis server
- PostgreSQL database
- InfluxDB instance

## Installation

```bash
npm install
```

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key configurations:
- `KAFKA_BROKERS` - Kafka cluster addresses
- `REDIS_HOST` - Redis server address
- `POSTGRES_*` - PostgreSQL connection details
- `INFLUX_*` - InfluxDB connection details

## Development

```bash
# Start in development mode with hot reload
npm run dev

# Build TypeScript
npm run build

# Run production build
npm start
```

## Docker

```bash
# Build image
docker build -t dashboard-api:latest .

# Run container
docker run -p 4000:4000 --env-file .env dashboard-api:latest

# Or use npm scripts
npm run docker:build
npm run docker:run
```

## API Usage

### GraphQL Endpoint

```
http://localhost:4000/graphql
```

### Example Queries

**Hospital KPIs:**
```graphql
query {
  hospitalKpis(hospitalId: "HOSP001") {
    hospitalId
    timestamp
    occupancyRate
    icuOccupancyRate
    totalAdmissions
    mortalityRate
    patientSatisfactionScore
  }
}
```

**High Risk Patients:**
```graphql
query {
  highRiskPatients(hospitalId: "HOSP001", riskLevel: CRITICAL, limit: 10) {
    patientId
    overallRiskScore
    riskLevel
    activeAlerts {
      alertType
      severity
      message
    }
    recommendedInterventions
  }
}
```

**Sepsis Surveillance:**
```graphql
query {
  sepsisSurveillance(hospitalId: "HOSP001", alertLevel: CRITICAL) {
    alertId
    patientId
    sepsisStage
    sofaScore
    qSofaScore
    bundleCompliance {
      overallCompliance
      lactateDrawn
      antibioticsAdministered
    }
    timeToIntervention
  }
}
```

**Dashboard Summary:**
```graphql
query {
  dashboardSummary(hospitalId: "HOSP001") {
    hospitalKpis {
      occupancyRate
      totalAdmissions
    }
    topDepartments {
      departmentName
      occupancyRate
      criticalAlerts
    }
    highRiskPatients {
      patientId
      riskLevel
    }
    activeSepsisAlerts {
      alertId
      patientId
      sepsisStage
    }
    realtimeStats {
      activePatients
      criticalPatients
      availableBeds
    }
  }
}
```

## Health Endpoints

- **Health Check:** `GET /health` - Comprehensive service health
- **Readiness:** `GET /ready` - Kubernetes readiness probe
- **Liveness:** `GET /live` - Kubernetes liveness probe
- **Metrics:** `GET /metrics` - Basic runtime metrics

## Data Flow

1. **Kafka Topics** - 5 analytics topics produce enriched data
2. **Consumers** - Dedicated consumer per topic with error handling
3. **Redis Cache** - Latest data cached with 5-min TTL
4. **PostgreSQL** - Historical data with indexed queries
5. **InfluxDB** - Time-series metrics for trending
6. **GraphQL** - Unified query interface with resolvers

## Kafka Topics Consumed

- `hospital-kpis` - Hospital-wide KPIs and metrics
- `department-metrics` - Department-level operational data
- `patient-risk-profiles` - Patient risk assessments
- `sepsis-surveillance` - Sepsis alerts and surveillance
- `quality-metrics` - Clinical quality indicators

## Performance Optimization

- **Redis Caching** - 5-min TTL reduces database load
- **Connection Pooling** - PostgreSQL connection pool (20 connections)
- **Parallel Queries** - Promise.all for independent data fetches
- **Time-series Cleanup** - Automatic cleanup of old cache entries
- **Sorted Sets** - Redis sorted sets for efficient time-based queries

## Error Handling

- Comprehensive try-catch in all resolvers
- Kafka consumer error recovery with retries
- Database connection error handling
- Graceful degradation for service failures
- Structured logging with pino

## Monitoring

- Health check reports all service statuses
- Uptime and memory metrics
- Request logging
- Error tracking
- Performance metrics endpoint

## Security

- CORS configuration
- Environment-based secrets
- Non-root Docker user
- Input validation in resolvers
- Rate limiting ready (configurable)

## Testing

```bash
# Run tests
npm test

# Run tests with coverage
npm test -- --coverage

# Lint code
npm run lint

# Format code
npm run format
```

## Deployment

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dashboard-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: dashboard-api
  template:
    metadata:
      labels:
        app: dashboard-api
    spec:
      containers:
      - name: dashboard-api
        image: dashboard-api:latest
        ports:
        - containerPort: 4000
        env:
        - name: KAFKA_BROKERS
          value: "kafka:9092"
        - name: REDIS_HOST
          value: "redis"
        - name: POSTGRES_HOST
          value: "postgres"
        livenessProbe:
          httpGet:
            path: /live
            port: 4000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 4000
          initialDelaySeconds: 15
          periodSeconds: 5
```

## Troubleshooting

**Kafka connection issues:**
- Check `KAFKA_BROKERS` configuration
- Verify network connectivity
- Check Kafka cluster health

**Cache misses:**
- Verify Redis connection
- Check TTL configuration
- Monitor Redis memory

**Slow queries:**
- Check PostgreSQL indexes
- Monitor connection pool
- Review InfluxDB performance

## Project Structure

```
dashboard-api/
├── src/
│   ├── config/           # Configuration and logger
│   ├── models/           # TypeScript types
│   ├── resolvers/        # GraphQL resolvers
│   ├── schema/           # GraphQL schema
│   ├── services/         # Business logic services
│   │   ├── kafka-consumer.service.ts
│   │   ├── analytics-data.service.ts
│   │   └── cache.service.ts
│   └── server.ts         # Main entry point
├── Dockerfile            # Multi-stage production build
├── tsconfig.json         # TypeScript configuration
├── package.json          # Dependencies
└── .env.example          # Environment template
```

## Contributing

1. Follow TypeScript best practices
2. Add tests for new features
3. Update GraphQL schema for new types
4. Document environment variables
5. Maintain error handling patterns

## License

MIT
