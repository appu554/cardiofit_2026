# Module 6: Complete Deployment Guide

**CardioFit Real-Time Analytics & Predictive Dashboards**

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Prerequisites](#prerequisites)
4. [Deployment Options](#deployment-options)
5. [Automated Deployment](#automated-deployment)
6. [Manual Deployment](#manual-deployment)
7. [Verification](#verification)
8. [Troubleshooting](#troubleshooting)
9. [Monitoring](#monitoring)
10. [Maintenance](#maintenance)

---

## Overview

Module 6 implements real-time analytics and predictive dashboards for the CardioFit platform. It consumes data from Modules 1-5 and provides:

- **Real-Time Analytics**: Flink SQL-based materialized views
- **GraphQL API**: Type-safe API with subscriptions
- **WebSocket Updates**: Sub-second real-time dashboard updates
- **Multi-Channel Notifications**: SMS, Email, Push, Pager alerts
- **Executive Dashboards**: Hospital-wide KPI visualization
- **Clinical Dashboards**: Department-level patient management
- **Patient Detail Views**: Individual risk profiles

### Components

| Component | Technology | Port | Purpose |
|-----------|------------|------|---------|
| **Analytics Engine** | Flink SQL/Java | N/A | Materialized views from Kafka |
| **Dashboard API** | Node.js + Apollo | 4001 | GraphQL API with Kafka consumers |
| **WebSocket Server** | Node.js + ws | 8080 | Real-time push notifications |
| **Notification Service** | Spring Boot | 8090 | Multi-channel alert delivery |
| **Dashboard UI** | React + MUI | 3000 | Web-based dashboards |
| **Redis** | Redis 7 | 6379 | Real-time cache |
| **PostgreSQL** | PostgreSQL 15 | 5433 | Historical analytics data |
| **InfluxDB** | InfluxDB 2.7 | 8086 | Time-series metrics |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kafka Topics (Modules 1-5)                   │
│  enriched-patient-events.v1  │  clinical-patterns.v1  │         │
│  ml-predictions.v1           │  composed-alerts.v1    │         │
└──────────────────┬───────────────────────────────────────────────┘
                   │
    ┌──────────────┴──────────────┬────────────────────────────┐
    │                             │                            │
    ▼                             ▼                            ▼
┌─────────────────┐   ┌───────────────────────┐   ┌──────────────────┐
│ Flink Analytics │   │   Dashboard API       │   │  Notification    │
│     Engine      │   │  (Kafka Consumers)    │   │    Service       │
│  (Module 6)     │   └──────────┬────────────┘   └────────┬─────────┘
└────────┬────────┘              │                         │
         │                       │                         │
         ▼                       ▼                         ▼
  ┌──────────────┐      ┌────────────────┐      ┌──────────────────┐
  │ Kafka Output │      │ Redis + Postgres│      │ Twilio/SendGrid  │
  │   Topics     │      │   + InfluxDB    │      │    + Firebase    │
  └──────┬───────┘      └────────┬────────┘      └──────────────────┘
         │                       │
         │                       │
         ▼                       ▼
   ┌───────────────────────────────────────┐
   │       WebSocket Server                │
   │   (Real-time Broadcasting)            │
   └───────────────┬───────────────────────┘
                   │
                   ▼
           ┌───────────────┐
           │  Dashboard UI │
           │  (React SPA)  │
           └───────────────┘
```

---

## Prerequisites

### System Requirements

- **OS**: Linux, macOS, or Windows with WSL2
- **Java**: JDK 17 or higher
- **Node.js**: v18 or higher
- **Maven**: 3.8+
- **Docker**: 20.10+ with Docker Compose
- **Memory**: 16GB RAM minimum
- **Disk**: 50GB available space

### Required Services

Before deploying Module 6, ensure these are running:

1. **Apache Flink Cluster**
   - JobManager accessible at `localhost:8081`
   - TaskManagers with sufficient slots (≥4)

2. **Apache Kafka**
   - Bootstrap servers accessible
   - Topics from Modules 1-5 exist and have data

3. **Modules 1-5**
   - All prerequisite Flink jobs are running
   - Producing data to expected topics

### Verification Commands

```bash
# Check Flink cluster
curl http://localhost:8081/overview

# Check Kafka connectivity
kafka-broker-api-versions --bootstrap-server localhost:9092

# Check Flink jobs
flink list -r

# Check Kafka topics
kafka-topics --list --bootstrap-server localhost:9092 | grep -E "(enriched-patient|clinical-patterns|ml-predictions)"
```

---

## Deployment Options

### Option 1: Automated Deployment (Recommended)

Single command deployment using provided script:

```bash
cd backend/shared-infrastructure/flink-processing
./deploy-module6.sh
```

This script:
- ✅ Verifies prerequisites
- ✅ Creates Kafka topics
- ✅ Initializes PostgreSQL database
- ✅ Builds Flink job
- ✅ Deploys to Flink cluster
- ✅ Starts Docker services
- ✅ Runs health checks

### Option 2: Docker Compose Only

Deploy only the services (assumes Flink job is already running):

```bash
docker-compose -f docker-compose-module6.yml up -d
```

### Option 3: Manual Step-by-Step

Full control over each component (see [Manual Deployment](#manual-deployment) section).

---

## Automated Deployment

### Step 1: Configuration

Set environment variables (optional):

```bash
# Kafka configuration
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092

# Flink configuration
export FLINK_JOB_MANAGER=localhost:8081

# PostgreSQL configuration
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5433
export POSTGRES_PASSWORD=cardiofit_analytics_pass
```

### Step 2: Run Deployment Script

```bash
cd backend/shared-infrastructure/flink-processing
./deploy-module6.sh
```

### Step 3: Monitor Deployment

The script will output progress for each step:

```
[1/8] Verifying prerequisites...
[2/8] Creating Kafka topics for Module 6...
[3/8] Initializing PostgreSQL analytics database...
[4/8] Building Flink Analytics Engine...
[5/8] Deploying Flink Analytics Engine to cluster...
[6/8] Starting Module 6 services with Docker Compose...
[7/8] Verifying data flow...
[8/8] Running health checks...
```

### Step 4: Access Services

Once deployment completes:

- **Dashboard UI**: http://localhost:3000
- **GraphQL Playground**: http://localhost:4001/graphql
- **WebSocket Test**: ws://localhost:8080/dashboard/realtime
- **Flink Web UI**: http://localhost:8081

---

## Manual Deployment

### Step 1: Create Kafka Topics

```bash
cd backend/shared-infrastructure/flink-processing
chmod +x create-module6-topics.sh
./create-module6-topics.sh
```

Verify topics were created:

```bash
kafka-topics --list --bootstrap-server localhost:9092 | grep analytics
```

Expected output:
```
analytics-alert-metrics
analytics-department-workload
analytics-ml-performance
analytics-patient-census
analytics-sepsis-surveillance
```

### Step 2: Initialize PostgreSQL Database

```bash
# Start PostgreSQL if not running
docker-compose -f docker-compose-module6.yml up -d postgres-analytics

# Wait for database to be ready
sleep 10

# Run initialization script
PGPASSWORD=cardiofit_analytics_pass psql \
  -h localhost \
  -p 5433 \
  -U cardiofit \
  -d cardiofit_analytics \
  -f sql/init-analytics-db.sql
```

Verify schema:

```bash
PGPASSWORD=cardiofit_analytics_pass psql \
  -h localhost -p 5433 -U cardiofit -d cardiofit_analytics \
  -c "\dt"
```

### Step 3: Build and Deploy Flink Analytics Engine

```bash
# Build JAR
mvn clean package -DskipTests

# Verify build
ls -lh target/flink-ehr-intelligence-1.0.0.jar

# Submit to Flink
flink run \
  -c com.cardiofit.flink.analytics.Module6_AnalyticsEngine \
  target/flink-ehr-intelligence-1.0.0.jar

# Verify job is running
flink list -r
```

### Step 4: Start Infrastructure Services

```bash
# Start Redis
docker-compose -f docker-compose-module6.yml up -d redis-analytics

# Start PostgreSQL (if not already running)
docker-compose -f docker-compose-module6.yml up -d postgres-analytics

# Start InfluxDB
docker-compose -f docker-compose-module6.yml up -d influxdb

# Wait for health checks
sleep 15
```

### Step 5: Start Dashboard API

```bash
# Method 1: Docker
docker-compose -f docker-compose-module6.yml up -d dashboard-api

# Method 2: Local development
cd module6-services/dashboard-api
npm install
cp .env.example .env
npm run dev
```

### Step 6: Start WebSocket Server

```bash
# Method 1: Docker
docker-compose -f docker-compose-module6.yml up -d websocket-server

# Method 2: Local development
cd module6-services/websocket-server
npm install
cp .env.example .env
npm run dev
```

### Step 7: Start Notification Service

```bash
# Method 1: Docker
docker-compose -f docker-compose-module6.yml up -d notification-service

# Method 2: Local development
cd module6-services/notification-service
mvn spring-boot:run
```

### Step 8: Start Dashboard UI

```bash
# Method 1: Docker
docker-compose -f docker-compose-module6.yml up -d dashboard-ui

# Method 2: Local development
cd module6-services/dashboard-ui
npm install
npm start
```

---

## Verification

### 1. Check All Services Are Running

```bash
docker-compose -f docker-compose-module6.yml ps
```

Expected output: All services should show "Up" status.

### 2. Verify Flink Job

```bash
# Check job is running
flink list -r

# Check job metrics in Flink UI
open http://localhost:8081
```

Look for "Module 6: Real-Time Analytics Engine" in running jobs.

### 3. Verify Kafka Data Flow

```bash
# Check if analytics topics have data
kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic analytics-patient-census \
  --max-messages 5

# Should see JSON messages with patient census data
```

### 4. Verify Redis Cache

```bash
# Connect to Redis
docker exec -it cardiofit-redis-analytics redis-cli

# Check for cached data
KEYS census:*
GET census:ICU

# Should see JSON data
```

### 5. Test GraphQL API

```bash
# Health check
curl http://localhost:4001/health

# Test query
curl -X POST http://localhost:4001/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{ hospitalKPIs { totalPatients criticalRiskCount activeAlerts } }"
  }'
```

Expected response:
```json
{
  "data": {
    "hospitalKPIs": {
      "totalPatients": 134,
      "criticalRiskCount": 15,
      "activeAlerts": 28
    }
  }
}
```

### 6. Test WebSocket Connection

```bash
# Check WebSocket health
curl http://localhost:8080/health

# Test connection (using wscat)
npm install -g wscat
wscat -c ws://localhost:8080/dashboard/realtime

# Send subscription request
> {"type":"SUBSCRIBE","payload":{"rooms":["hospital-wide"]}}

# Should receive real-time updates
```

### 7. Access Dashboard UI

```bash
open http://localhost:3000
```

Expected:
- Executive Dashboard shows hospital-wide KPIs
- Clinical Dashboard shows department metrics
- Patient Detail shows individual risk profiles
- Real-time updates occur automatically

---

## Troubleshooting

### Issue: Flink Job Fails to Start

**Symptoms**: Job submission fails or job status is "FAILED"

**Solutions**:

```bash
# Check Flink logs
docker logs flink-jobmanager
docker logs flink-taskmanager

# Verify TaskManager slots available
curl http://localhost:8081/taskmanagers

# Check for dependency issues
flink run -c com.cardiofit.flink.analytics.Module6_AnalyticsEngine \
  target/flink-ehr-intelligence-1.0.0.jar \
  --verbose
```

### Issue: No Data in Kafka Topics

**Symptoms**: `analytics-*` topics are empty

**Solutions**:

```bash
# Verify Modules 1-5 are producing data
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic enriched-patient-events.v1 --max-messages 1

# Check Flink job is consuming
curl http://localhost:8081/jobs/<JOB_ID>

# Verify Kafka consumer groups
kafka-consumer-groups --bootstrap-server localhost:9092 --list
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group analytics-engine --describe
```

### Issue: Dashboard API Can't Connect to Kafka

**Symptoms**: Dashboard API health check fails, no data in Redis

**Solutions**:

```bash
# Check environment variables
docker exec cardiofit-dashboard-api env | grep KAFKA

# Check Dashboard API logs
docker logs cardiofit-dashboard-api

# Test Kafka connectivity from container
docker exec cardiofit-dashboard-api \
  kafkacat -b localhost:9092 -L

# Restart service
docker-compose -f docker-compose-module6.yml restart dashboard-api
```

### Issue: Redis Cache is Empty

**Symptoms**: API queries slow, no cached data

**Solutions**:

```bash
# Check Redis is running
docker exec cardiofit-redis-analytics redis-cli ping

# Check Dashboard API Kafka consumers
docker logs cardiofit-dashboard-api | grep "Kafka consumer"

# Manually verify cache
docker exec cardiofit-redis-analytics redis-cli
> KEYS *
> TTL census:ICU

# Check for connection errors
docker logs cardiofit-dashboard-api | grep -i error
```

### Issue: WebSocket Connections Fail

**Symptoms**: Dashboard shows "Disconnected", no real-time updates

**Solutions**:

```bash
# Check WebSocket server health
curl http://localhost:8080/health

# Check WebSocket logs
docker logs cardiofit-websocket-server

# Test connection manually
wscat -c ws://localhost:8080/dashboard/realtime

# Check Kafka consumers
docker logs cardiofit-websocket-server | grep "Kafka consumer"

# Restart WebSocket server
docker-compose -f docker-compose-module6.yml restart websocket-server
```

### Issue: Dashboard UI Shows "Loading..." Forever

**Symptoms**: UI loads but shows no data

**Solutions**:

```bash
# Check browser console for errors (F12 in browser)

# Verify API is accessible
curl http://localhost:4001/health

# Check CORS configuration
docker logs cardiofit-dashboard-api | grep -i cors

# Test GraphQL query
curl -X POST http://localhost:4001/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ __typename }"}'

# Rebuild and restart UI
docker-compose -f docker-compose-module6.yml up -d --build dashboard-ui
```

### Issue: Notifications Not Being Sent

**Symptoms**: Alerts generated but not delivered

**Solutions**:

```bash
# Check notification service logs
docker logs cardiofit-notification-service

# Verify Twilio/SendGrid credentials
docker exec cardiofit-notification-service env | grep -E "(TWILIO|SENDGRID)"

# Check Kafka consumer
docker logs cardiofit-notification-service | grep "Kafka"

# Test notification manually (REST API)
curl -X POST http://localhost:8090/api/notifications/test

# Check Redis rate limiting
docker exec cardiofit-redis-analytics redis-cli
> KEYS rate_limit:*
```

---

## Monitoring

### Health Checks

```bash
# All services status
docker-compose -f docker-compose-module6.yml ps

# Individual health checks
curl http://localhost:4001/health       # Dashboard API
curl http://localhost:8080/health       # WebSocket Server
curl http://localhost:8090/actuator/health  # Notification Service
curl http://localhost:3000              # Dashboard UI
```

### Metrics

```bash
# WebSocket server metrics
curl http://localhost:8080/metrics

# Flink job metrics
curl http://localhost:8081/jobs/<JOB_ID>/metrics

# Redis metrics
docker exec cardiofit-redis-analytics redis-cli INFO stats

# PostgreSQL metrics
PGPASSWORD=cardiofit_analytics_pass psql \
  -h localhost -p 5433 -U cardiofit -d cardiofit_analytics \
  -c "SELECT COUNT(*) FROM patient_metrics;"
```

### Logs

```bash
# View all logs
docker-compose -f docker-compose-module6.yml logs -f

# Individual service logs
docker logs -f cardiofit-dashboard-api
docker logs -f cardiofit-websocket-server
docker logs -f cardiofit-notification-service

# Flink job logs
curl http://localhost:8081/jobs/<JOB_ID>/exceptions
```

### Kafka Monitoring

```bash
# Consumer lag
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group dashboard-api --describe

kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group websocket-server --describe

# Topic metrics
kafka-topics --bootstrap-server localhost:9092 \
  --describe --topic analytics-patient-census
```

---

## Maintenance

### Restart Services

```bash
# Restart all services
docker-compose -f docker-compose-module6.yml restart

# Restart specific service
docker-compose -f docker-compose-module6.yml restart dashboard-api
```

### Update Services

```bash
# Rebuild and restart
docker-compose -f docker-compose-module6.yml up -d --build

# Update specific service
docker-compose -f docker-compose-module6.yml up -d --build dashboard-ui
```

### Clean Up Old Data

```bash
# Run PostgreSQL cleanup function
PGPASSWORD=cardiofit_analytics_pass psql \
  -h localhost -p 5433 -U cardiofit -d cardiofit_analytics \
  -c "SELECT cleanup_old_analytics_data();"

# Clear Redis cache
docker exec cardiofit-redis-analytics redis-cli FLUSHDB

# Delete old Kafka messages (30 days)
kafka-configs --bootstrap-server localhost:9092 \
  --entity-type topics \
  --entity-name analytics-patient-census \
  --alter --add-config retention.ms=2592000000
```

### Backup Data

```bash
# Backup PostgreSQL
docker exec cardiofit-postgres-analytics pg_dump \
  -U cardiofit cardiofit_analytics > analytics-backup-$(date +%Y%m%d).sql

# Backup Redis
docker exec cardiofit-redis-analytics redis-cli SAVE

# Export Kafka topic
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic analytics-patient-census \
  --from-beginning --timeout-ms 5000 > patient-census-backup.json
```

### Stop All Services

```bash
# Stop but keep data
docker-compose -f docker-compose-module6.yml stop

# Stop and remove containers (data persists in volumes)
docker-compose -f docker-compose-module6.yml down

# Stop, remove containers AND volumes (DANGER: deletes all data)
docker-compose -f docker-compose-module6.yml down -v
```

---

## Performance Tuning

### Flink Job Optimization

```bash
# Increase parallelism
flink run -p 8 \
  -c com.cardiofit.flink.analytics.Module6_AnalyticsEngine \
  target/flink-ehr-intelligence-1.0.0.jar

# Adjust memory
flink run -ytm 2048 \
  -c com.cardiofit.flink.analytics.Module6_AnalyticsEngine \
  target/flink-ehr-intelligence-1.0.0.jar
```

### Redis Optimization

```bash
# Increase memory limit
docker exec cardiofit-redis-analytics redis-cli CONFIG SET maxmemory 4gb

# Adjust eviction policy
docker exec cardiofit-redis-analytics redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

### PostgreSQL Optimization

```bash
# Analyze tables
PGPASSWORD=cardiofit_analytics_pass psql \
  -h localhost -p 5433 -U cardiofit -d cardiofit_analytics \
  -c "ANALYZE VERBOSE;"

# Vacuum tables
PGPASSWORD=cardiofit_analytics_pass psql \
  -h localhost -p 5433 -U cardiofit -d cardiofit_analytics \
  -c "VACUUM ANALYZE patient_metrics;"
```

---

## Additional Resources

- **Implementation Guide**: [MODULE_6_IMPLEMENTATION_GUIDE.md](./MODULE_6_IMPLEMENTATION_GUIDE.md)
- **Quick Start**: [MODULE_6_QUICKSTART.md](./MODULE_6_QUICKSTART.md)
- **Original Documentation**: [Module_6_Advanced_Analytics_&_Predictive_Dashboards.txt](./src/docs/module_6/Module_6_Advanced_Analytics_&_Predictive_Dashboards.txt)
- **Dashboard API README**: [module6-services/dashboard-api/README.md](./module6-services/dashboard-api/README.md)
- **WebSocket Server README**: [module6-services/websocket-server/README.md](./module6-services/websocket-server/README.md)

---

## Support

For issues or questions:
1. Check logs: `docker-compose -f docker-compose-module6.yml logs`
2. Verify prerequisites are running
3. Review troubleshooting section above
4. Check Kafka topics for data flow
5. Verify Flink job status

---

**Module 6 Deployment Complete!** 🎉
