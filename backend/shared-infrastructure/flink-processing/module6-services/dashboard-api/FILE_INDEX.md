# Dashboard API - Complete File Index

## Project Structure Overview

```
dashboard-api/
├── Documentation (6 files)
├── Configuration (5 files)
├── Source Code (13 files)
├── Docker (2 files)
└── Scripts (1 file)

Total: 27 files
```

## Documentation Files

### 1. README.md
- **Lines**: ~5,200
- **Purpose**: Comprehensive technical documentation
- **Contents**:
  - Architecture overview
  - Installation instructions
  - API usage examples
  - Configuration guide
  - Deployment instructions
  - Troubleshooting guide

### 2. QUICKSTART.md
- **Lines**: ~580
- **Purpose**: 5-minute setup guide
- **Contents**:
  - Quick start with Docker Compose
  - Local development setup
  - Testing instructions
  - Example queries
  - Health check commands
  - Troubleshooting steps

### 3. PROJECT_SUMMARY.md
- **Lines**: ~900
- **Purpose**: High-level architecture and design overview
- **Contents**:
  - Architecture diagrams
  - Key features
  - File structure
  - Schema highlights
  - Performance characteristics
  - Success metrics

### 4. FILE_INDEX.md
- **Lines**: This file
- **Purpose**: Complete file inventory and descriptions

### 5. example-queries.graphql
- **Lines**: ~600
- **Purpose**: Ready-to-use GraphQL queries
- **Contents**:
  - Hospital KPI queries
  - Department metrics queries
  - Patient risk queries
  - Sepsis surveillance queries
  - Quality metrics queries
  - Dashboard summary queries
  - Introspection queries
  - Future mutation/subscription examples

### 6. .dockerignore
- **Lines**: 14
- **Purpose**: Docker build optimization
- **Excludes**: node_modules, logs, test files, environment files

---

## Configuration Files

### 7. package.json
- **Lines**: 64
- **Purpose**: NPM package configuration
- **Dependencies**: 13 production packages
  - @apollo/server (GraphQL server)
  - kafkajs (Kafka client)
  - ioredis (Redis client)
  - pg (PostgreSQL client)
  - @influxdata/influxdb-client (InfluxDB)
  - express (HTTP server)
  - graphql (GraphQL core)
  - pino (logging)
  - cors, body-parser, uuid, dotenv
- **Dev Dependencies**: 10 packages
  - TypeScript, ts-node-dev
  - Jest testing framework
  - ESLint, Prettier
  - Type definitions
- **Scripts**: 9 commands
  - build, start, dev, test, lint, format
  - docker:build, docker:run

### 8. tsconfig.json
- **Lines**: 21
- **Purpose**: TypeScript compiler configuration
- **Target**: ES2022
- **Strict Mode**: Enabled
- **Output**: dist/ directory
- **Source Maps**: Enabled
- **Features**: Decorators, JSON imports

### 9. .env.example
- **Lines**: 48
- **Purpose**: Environment variable template
- **Sections**:
  - Server configuration (3 vars)
  - Kafka configuration (5 vars)
  - Redis configuration (5 vars)
  - PostgreSQL configuration (6 vars)
  - InfluxDB configuration (5 vars)
  - Logging configuration (2 vars)
  - GraphQL configuration (3 vars)
  - Security configuration (3 vars)
  - Monitoring configuration (2 vars)

### 10. .gitignore
- **Lines**: 26
- **Purpose**: Git exclusions
- **Excludes**: Dependencies, build output, environment files, logs, IDE files, OS files, coverage reports

### 11. docker-compose.yml
- **Lines**: 102
- **Purpose**: Multi-service orchestration
- **Services**: 6 services
  - dashboard-api (main application)
  - kafka + zookeeper (message broker)
  - redis (cache)
  - postgres (database)
  - influxdb (time-series)
- **Features**:
  - Health checks configured
  - Volume persistence
  - Network isolation
  - Environment variables

---

## Source Code - Configuration Layer

### 12. src/config/index.ts
- **Lines**: 136
- **Purpose**: Centralized configuration and logger
- **Exports**:
  - `Config` interface (all settings)
  - `config` object (parsed environment)
  - `logger` instance (pino logger)
- **Sections**:
  - Server config (port, host, env)
  - Kafka config (brokers, topics)
  - Redis config (connection, TTL)
  - PostgreSQL config (connection pooling)
  - InfluxDB config (connection, org, bucket)
  - GraphQL config (playground, introspection)
  - Security config (CORS, rate limiting)
  - Logging config (level, pretty print)
  - Monitoring config (metrics, health checks)

---

## Source Code - Type Definitions

### 13. src/models/types.ts
- **Lines**: 283
- **Purpose**: TypeScript type definitions
- **Types Defined**: 13 primary types
  - `HospitalKPIs` (23 fields)
  - `DepartmentMetrics` (18 fields)
  - `PatientRiskProfile` (24 fields)
  - `SepsisSurveillance` (28 fields)
  - `QualityMetrics` (20 fields)
  - `VitalSigns` (8 fields)
  - `LabResults` (7 fields)
  - `Alert` (6 fields)
  - `SofaComponents` (6 fields)
  - `SirsCriteria` (4 fields)
  - `QSofaCriteria` (3 fields)
  - `BundleCompliance` (7 fields)
  - `KafkaMessage<T>` (generic)
- **Enums**: 9 enums
  - RiskLevel (4 values)
  - AlertLevel (4 values)
  - AlertStatus (4 values)
  - SepsisStage (5 values)
  - QualityMetricType (10 values)
  - PerformanceStatus (4 values)
  - TrendDirection (3 values)
  - TimeRange (6 values)

---

## Source Code - GraphQL Schema

### 14. src/schema/types.graphql
- **Lines**: 357
- **Purpose**: Complete GraphQL schema definition
- **Query Types**: 15 queries
  - hospitalKpis, hospitalKpisTrend
  - departmentMetrics, departmentMetricsTrend
  - patientRiskProfile, highRiskPatients, patientRiskHistory
  - sepsisSurveillance, sepsisAlerts, sepsisPatientDetails
  - qualityMetrics, qualityMetricsTrend, complianceScore
  - dashboardSummary, realtimeStats
- **Subscription Types**: 4 subscriptions
  - hospitalKpisUpdated
  - newSepsisAlert
  - patientRiskChanged
  - qualityMetricsUpdated
- **Object Types**: 14 types
  - HospitalKPIs (23 fields)
  - DepartmentMetrics (20 fields)
  - PatientRiskProfile (26 fields)
  - SepsisSurveillance (32 fields)
  - QualityMetrics (22 fields)
  - VitalSigns, LabResults, Alert
  - SofaComponents, SirsCriteria, QSofaCriteria
  - BundleCompliance, ComplianceScore
  - DashboardSummary, RealtimeStats
- **Enums**: 9 enums
- **Scalars**: 2 custom scalars (DateTime, JSON)

---

## Source Code - Services Layer

### 15. src/services/cache.service.ts
- **Lines**: 209
- **Purpose**: Redis caching abstraction
- **Class**: `CacheService`
- **Methods**: 20 methods
  - Basic operations: get, set, del, exists
  - Multi operations: mget, mdel
  - Pattern operations: keys
  - Sorted set operations: zadd, zrangebyscore, zremrangebyscore
  - List operations: lpush, lrange, ltrim
  - TTL operations: expire, ttlRemaining
  - Utility: ping, flushdb, close
- **Features**:
  - Automatic JSON serialization/deserialization
  - Configurable TTL
  - Error handling with logging
  - Connection retry strategy
  - Health check support

### 16. src/services/kafka-consumer.service.ts
- **Lines**: 312
- **Purpose**: Kafka topic consumption
- **Class**: `KafkaConsumerService`
- **Consumers**: 5 parallel consumers
  - hospital-kpis consumer
  - department-metrics consumer
  - patient-risk-profiles consumer
  - sepsis-surveillance consumer
  - quality-metrics consumer
- **Methods**:
  - start() - Initialize all consumers
  - stop() - Graceful shutdown
  - healthCheck() - Consumer status
  - Private methods for each topic
  - parseMessage() - JSON parsing with date conversion
  - convertDates() - Recursive date conversion
  - Process methods for each data type
- **Features**:
  - Automatic message parsing
  - Date conversion from ISO strings
  - Multi-layer caching (Redis + time-series)
  - Database persistence
  - Error recovery
  - Consumer group management

### 17. src/services/analytics-data.service.ts
- **Lines**: 448
- **Purpose**: Multi-database data persistence
- **Class**: `AnalyticsDataService`
- **Databases**: 3 database integrations
  - PostgreSQL (historical data)
  - InfluxDB (time-series metrics)
  - Redis (cache integration)
- **Methods**: 20+ methods
  - Database initialization
  - Table creation
  - Store methods for each data type (5)
  - Get methods for each data type (5+)
  - Query methods with filtering
  - Trend/history queries
  - Health check
  - Connection close
- **Features**:
  - Connection pooling
  - Automatic table creation
  - Indexed queries
  - Time-series optimization
  - Cache integration
  - Error handling
  - Graceful degradation

---

## Source Code - Resolvers

### 18. src/resolvers/hospital-kpis.resolver.ts
- **Lines**: 32
- **Purpose**: Hospital KPI query resolvers
- **Resolvers**:
  - `hospitalKpis` - Latest KPIs
  - `hospitalKpisTrend` - Historical trend
- **Field Resolvers**:
  - timestamp, windowStart, windowEnd (date formatting)

### 19. src/resolvers/department-metrics.resolver.ts
- **Lines**: 37
- **Purpose**: Department metrics resolvers
- **Resolvers**:
  - `departmentMetrics` - Current metrics
  - `departmentMetricsTrend` - Historical trend
- **Field Resolvers**:
  - timestamp (date formatting)

### 20. src/resolvers/patient-risk.resolver.ts
- **Lines**: 67
- **Purpose**: Patient risk assessment resolvers
- **Resolvers**:
  - `patientRiskProfile` - Individual patient
  - `highRiskPatients` - Filtered list
  - `patientRiskHistory` - Historical trend
- **Field Resolvers**:
  - timestamp, lastUpdated (dates)
  - vitalSigns, labResults (nested objects)
  - activeAlerts (array with date formatting)

### 21. src/resolvers/sepsis-surveillance.resolver.ts
- **Lines**: 63
- **Purpose**: Sepsis surveillance resolvers
- **Resolvers**:
  - `sepsisSurveillance` - Active alerts
  - `sepsisAlerts` - Alert history
  - `sepsisPatientDetails` - Patient-specific
- **Field Resolvers**:
  - Multiple timestamp fields (6 dates)
  - vitalSigns, labResults (nested)

### 22. src/resolvers/quality-metrics.resolver.ts
- **Lines**: 85
- **Purpose**: Quality metrics resolvers
- **Resolvers**:
  - `qualityMetrics` - Current metrics
  - `qualityMetricsTrend` - Historical trend
  - `complianceScore` - Calculated compliance
- **Field Resolvers**:
  - timestamp, windowStart, windowEnd (dates)
- **Special Logic**:
  - Compliance score calculation
  - Category score aggregation

### 23. src/resolvers/dashboard.resolver.ts
- **Lines**: 82
- **Purpose**: Dashboard summary resolvers
- **Resolvers**:
  - `dashboardSummary` - Complete dashboard
  - `realtimeStats` - Live statistics
- **Features**:
  - Parallel data fetching
  - Cache-first strategy
  - Composite data aggregation

### 24. src/resolvers/index.ts
- **Lines**: 54
- **Purpose**: Resolver aggregation and custom scalars
- **Exports**: Unified resolvers object
- **Custom Scalars**:
  - DateTime scalar (ISO string serialization)
  - JSON scalar (passthrough)
- **Resolver Merge**:
  - Query resolvers (all modules)
  - Type resolvers (field resolvers)

---

## Source Code - Main Server

### 25. src/server.ts
- **Lines**: 203
- **Purpose**: Main application entry point
- **Components**:
  - Express HTTP server
  - Apollo GraphQL server
  - Kafka consumer startup
  - Health check endpoints
  - Graceful shutdown
- **Endpoints**:
  - POST /graphql - GraphQL API
  - GET /health - Comprehensive health check
  - GET /ready - Readiness probe
  - GET /live - Liveness probe
  - GET /metrics - Runtime metrics
  - GET / - Service info
- **Features**:
  - CORS configuration
  - Body parser middleware
  - Error handling
  - Signal handling (SIGTERM, SIGINT)
  - Uncaught error handling
  - Service cleanup
  - Structured logging

---

## Docker Files

### 26. Dockerfile
- **Lines**: 42
- **Purpose**: Production container image
- **Stages**: 2-stage build
  - Builder stage (compile TypeScript)
  - Production stage (minimal runtime)
- **Features**:
  - Multi-stage optimization
  - Non-root user (nodejs:nodejs)
  - Health check configured
  - Optimized layer caching
  - Production dependencies only
- **Image Size**: ~200MB (estimated)

---

## Scripts

### 27. test-api.sh
- **Lines**: 137
- **Purpose**: Automated API testing
- **Tests**: 15+ test cases
  - Health endpoints (4 tests)
  - GraphQL endpoint test
  - Schema introspection
  - All query types (8 queries)
  - Error handling
  - Service health details
- **Features**:
  - Color-coded output
  - Pass/fail counting
  - JSON response parsing
  - Exit code for CI/CD
  - Configurable API URL

---

## Summary Statistics

### File Count by Category
- **Documentation**: 6 files (~7,300 lines)
- **Configuration**: 5 files (~270 lines)
- **Source Code**: 13 files (~3,100 lines)
- **Docker**: 2 files (~145 lines)
- **Scripts**: 1 file (~140 lines)

**Total: 27 files, ~10,955 lines**

### Lines of Code by Type
- **TypeScript**: ~3,100 lines
- **GraphQL**: ~357 lines
- **JSON**: ~166 lines
- **YAML**: ~102 lines
- **Bash**: ~140 lines
- **Docker**: ~42 lines
- **Config**: ~148 lines
- **Documentation**: ~7,300 lines

**Total Production Code: ~4,055 lines**

### Code Organization
- **Models/Types**: 283 lines
- **Services**: 969 lines (3 files)
- **Resolvers**: 420 lines (7 files)
- **Config**: 136 lines
- **Schema**: 357 lines
- **Server**: 203 lines

### Test Coverage
- Automated test script: 137 lines
- Example queries: 600+ lines
- Documentation examples: Throughout

### Documentation Quality
- API documentation: Complete
- Code comments: Comprehensive
- Type safety: 100% (TypeScript strict mode)
- Examples provided: 60+ queries
- Setup guides: 3 detailed guides

---

## How to Navigate This Project

### For Quick Start
1. Read **QUICKSTART.md** first
2. Run `docker-compose up -d`
3. Use **example-queries.graphql** for testing

### For Understanding Architecture
1. Read **PROJECT_SUMMARY.md**
2. Review **src/schema/types.graphql**
3. Check **src/server.ts** for flow

### For Development
1. Start with **src/models/types.ts** for data structures
2. Review **src/services/** for business logic
3. Check **src/resolvers/** for GraphQL implementation
4. Reference **src/config/index.ts** for configuration

### For Deployment
1. Read **README.md** deployment section
2. Configure **.env** from **.env.example**
3. Use **Dockerfile** for containerization
4. Reference **docker-compose.yml** for infrastructure

### For Testing
1. Run **test-api.sh** for automated tests
2. Use **example-queries.graphql** in Playground
3. Check **/health** endpoint for status

---

## Key Integration Points

### Data Flow
```
Kafka Topics → kafka-consumer.service.ts → analytics-data.service.ts
                                        → cache.service.ts
                                        → PostgreSQL/InfluxDB

GraphQL Queries → resolvers/* → analytics-data.service.ts
                             → cache.service.ts
                             → Response
```

### Service Dependencies
```
server.ts
  ├── config/index.ts (configuration)
  ├── schema/types.graphql (GraphQL schema)
  ├── resolvers/index.ts (query handlers)
  │   ├── hospital-kpis.resolver.ts
  │   ├── department-metrics.resolver.ts
  │   ├── patient-risk.resolver.ts
  │   ├── sepsis-surveillance.resolver.ts
  │   ├── quality-metrics.resolver.ts
  │   └── dashboard.resolver.ts
  ├── services/kafka-consumer.service.ts
  ├── services/analytics-data.service.ts
  └── services/cache.service.ts
```

---

## Maintenance Guide

### Adding New Query
1. Update **src/schema/types.graphql** with new query
2. Add resolver in **src/resolvers/** (create new file if needed)
3. Export from **src/resolvers/index.ts**
4. Add example query to **example-queries.graphql**
5. Update **README.md** with usage

### Adding New Data Type
1. Add TypeScript type to **src/models/types.ts**
2. Add GraphQL type to **src/schema/types.graphql**
3. Add database schema in **analytics-data.service.ts**
4. Create new Kafka consumer method
5. Create store/get methods in data service
6. Create resolver file

### Configuration Changes
1. Add to **src/config/index.ts**
2. Add to **.env.example**
3. Update **README.md** configuration section
4. Update **docker-compose.yml** if needed

### Dependency Updates
1. Update **package.json**
2. Run `npm install`
3. Test thoroughly
4. Update **Dockerfile** if needed
5. Rebuild Docker image

---

## File Relationships

### Direct Dependencies
- **server.ts** depends on all services and resolvers
- **resolvers/** depend on **analytics-data.service.ts**
- **kafka-consumer.service.ts** depends on **cache.service.ts** and **analytics-data.service.ts**
- All files depend on **config/index.ts**

### Data Flow Dependencies
- Kafka → Consumer Service → Data Service → Database
- GraphQL → Resolvers → Data Service → Cache/Database
- Cache Service → Redis
- Data Service → PostgreSQL + InfluxDB

---

**Last Updated**: November 4, 2025
**Version**: 1.0.0
**Project**: CardioFit Clinical Analytics Dashboard API
