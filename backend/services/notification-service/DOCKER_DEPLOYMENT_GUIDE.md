# Docker Deployment Guide - Notification Service

Complete guide for deploying the CardioFit Notification Service using Docker.

---

## Quick Start (5 Minutes)

### 1. Start Infrastructure Only (No External Services)

```bash
# Start PostgreSQL, Redis, and Kafka only
docker-compose up -d postgres redis zookeeper kafka

# Wait for services to be healthy (~30 seconds)
docker-compose ps

# Run migrations
docker exec -it cardiofit-postgres psql -U cardiofit_user -d cardiofit_db -f /docker-entrypoint-initdb.d/001_create_notification_schema.up.sql
```

### 2. Start with Mock Notifications (Development)

```bash
# Copy environment file
cp .env.docker.example .env.docker

# Edit .env.docker and set:
# NOTIFICATION_DELIVERY_MODE=mock

# Start everything
docker-compose --env-file .env.docker up -d

# View logs
docker-compose logs -f notification-service
```

### 3. Start with Real External Services (Production)

```bash
# Copy and configure environment
cp .env.docker.example .env.docker

# Add your real API keys to .env.docker:
# - TWILIO_ACCOUNT_SID
# - TWILIO_AUTH_TOKEN
# - SENDGRID_API_KEY
# - FIREBASE_CREDENTIALS_PATH

# Start everything
docker-compose --env-file .env.docker up -d
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                   Docker Compose Stack                       │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐        ┌──────────────────┐          │
│  │  Notification    │───────▶│   PostgreSQL     │          │
│  │  Service (Go)    │        │   (User Prefs)   │          │
│  │  Port: 8080      │        │   Port: 5432     │          │
│  │  Port: 50051     │        └──────────────────┘          │
│  └────────┬─────────┘                                        │
│           │                                                  │
│           │              ┌──────────────────┐               │
│           ├─────────────▶│      Redis       │               │
│           │              │  (Cache/Fatigue) │               │
│           │              │   Port: 6379     │               │
│           │              └──────────────────┘               │
│           │                                                  │
│           │              ┌──────────────────┐               │
│           └─────────────▶│      Kafka       │               │
│                          │   (Alerts Feed)  │               │
│                          │   Port: 9092     │               │
│                          └────────┬─────────┘               │
│                                   │                          │
│                                   │                          │
│                          ┌────────▼─────────┐               │
│                          │   Zookeeper      │               │
│                          │   Port: 2181     │               │
│                          └──────────────────┘               │
│                                                               │
│  Optional Monitoring:                                        │
│  ┌──────────────────┐        ┌──────────────────┐          │
│  │   Prometheus     │◀───────│     Grafana      │          │
│  │   Port: 9090     │        │   Port: 3000     │          │
│  └──────────────────┘        └──────────────────┘          │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────┐
        │     External Services          │
        ├────────────────────────────────┤
        │  ▪ Twilio (SMS/Voice)          │
        │  ▪ SendGrid (Email)            │
        │  ▪ Firebase (Push)             │
        └────────────────────────────────┘
```

---

## Service Ports

| Service | Port(s) | Purpose |
|---------|---------|---------|
| **Notification Service** | 8080 | HTTP API |
| **Notification Service** | 50051 | gRPC API |
| PostgreSQL | 5432 | Database |
| Redis | 6379 | Cache |
| Kafka | 9092, 29092 | Message Broker |
| Zookeeper | 2181 | Kafka Coordination |
| Prometheus | 9090 | Metrics (optional) |
| Grafana | 3000 | Dashboards (optional) |
| Redis Commander | 8081 | Redis UI (optional) |
| Kafka UI | 8082 | Kafka UI (optional) |

---

## Deployment Modes

### Mode 1: Development (Mock External Services)

**Use Case**: Local development without real SMS/email sending

```bash
# .env.docker
NOTIFICATION_DELIVERY_MODE=mock
TWILIO_ENABLED=false
SENDGRID_ENABLED=false
FIREBASE_ENABLED=false

# Start services
docker-compose --env-file .env.docker up -d

# Notifications will be logged but not actually sent
```

**What Happens**:
- ✅ All infrastructure services run (Postgres, Redis, Kafka)
- ✅ Notification service processes alerts
- ✅ Delivery methods return success
- ❌ No actual SMS/emails/push notifications sent
- ✅ All delivery attempts logged

### Mode 2: Staging (Real Services, Test Credentials)

**Use Case**: Integration testing with real external APIs but test numbers/emails

```bash
# .env.docker
NOTIFICATION_DELIVERY_MODE=live
TWILIO_ACCOUNT_SID=your_test_account_sid
TWILIO_AUTH_TOKEN=your_test_auth_token
TWILIO_FROM_NUMBER=+15005550006  # Twilio test number
SENDGRID_API_KEY=your_test_api_key
SENDGRID_FROM_EMAIL=test@yourdomain.com

# Start services
docker-compose --env-file .env.docker up -d
```

**What Happens**:
- ✅ Real API calls to Twilio/SendGrid
- ✅ Test phone numbers and emails only
- ✅ Actual API quotas consumed (use free tier)
- ⚠️ Limited to verified test recipients

### Mode 3: Production (Full Setup)

**Use Case**: Production deployment with real notifications

```bash
# .env.docker
NOTIFICATION_DELIVERY_MODE=live
ENVIRONMENT=production
LOG_LEVEL=info

# Real production credentials
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your_production_token
TWILIO_FROM_NUMBER=+18005551234
SENDGRID_API_KEY=SG.production_key_here
FIREBASE_CREDENTIALS_PATH=/path/to/firebase-prod.json

# Start services
docker-compose --env-file .env.docker up -d

# Enable monitoring (optional)
docker-compose --profile monitoring up -d
```

**What Happens**:
- ✅ Full notification delivery to real users
- ✅ All external services active
- ✅ Production-grade logging
- ⚠️ Real costs for Twilio/SendGrid

---

## Step-by-Step Setup

### Step 1: Prerequisites

```bash
# Install Docker and Docker Compose
docker --version  # Should be 20.10+
docker-compose --version  # Should be 1.29+

# Clone repository and navigate to notification service
cd backend/services/notification-service
```

### Step 2: Configure Environment

```bash
# Copy example environment file
cp .env.docker.example .env.docker

# Edit with your favorite editor
nano .env.docker  # or vim, code, etc.
```

**Required Configuration** (for production):

```bash
# External Services
TWILIO_ACCOUNT_SID=AC...    # From Twilio Console
TWILIO_AUTH_TOKEN=...       # From Twilio Console
TWILIO_FROM_NUMBER=+1...    # Your Twilio phone number

SENDGRID_API_KEY=SG...      # From SendGrid Settings
SENDGRID_FROM_EMAIL=...     # Your verified sender email

FIREBASE_CREDENTIALS_PATH=./firebase-credentials.json
FIREBASE_PROJECT_ID=...     # Your Firebase project ID
```

### Step 3: Add Firebase Credentials (if using Push)

```bash
# Download from Firebase Console:
# Project Settings → Service Accounts → Generate new private key

# Save as firebase-credentials.json in notification service directory
# Make sure path in .env.docker matches
```

### Step 4: Start Infrastructure Services

```bash
# Start database and cache first
docker-compose up -d postgres redis

# Wait for health checks
docker-compose ps

# You should see:
# cardiofit-postgres  ... Up (healthy)
# cardiofit-redis     ... Up (healthy)
```

### Step 5: Run Database Migrations

```bash
# Check if migration files exist
ls -la migrations/

# If migrations exist, they run automatically on first start
# Otherwise, run manually:
docker exec -it cardiofit-postgres psql -U cardiofit_user -d cardiofit_db <<'EOF'
CREATE SCHEMA IF NOT EXISTS notification_service;

CREATE TABLE notification_service.user_preferences (
    user_id VARCHAR(255) PRIMARY KEY,
    channel_preferences JSONB NOT NULL DEFAULT '{}',
    severity_channels JSONB NOT NULL DEFAULT '{}',
    quiet_hours_enabled BOOLEAN DEFAULT false,
    quiet_hours_start INTEGER,
    quiet_hours_end INTEGER,
    max_alerts_per_hour INTEGER DEFAULT 20,
    phone_number VARCHAR(50),
    email VARCHAR(255),
    pager_number VARCHAR(50),
    fcm_token TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_user_preferences_updated ON notification_service.user_preferences(updated_at);
EOF
```

### Step 6: Start Kafka

```bash
# Start Zookeeper and Kafka
docker-compose up -d zookeeper kafka

# Wait for Kafka to be healthy (~30 seconds)
docker-compose ps kafka

# Create required topics
docker exec -it cardiofit-kafka kafka-topics --create --topic enriched-patient-events-v1 --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1
docker exec -it cardiofit-kafka kafka-topics --create --topic clinical-patterns.v1 --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1
docker exec -it cardiofit-kafka kafka-topics --create --topic composed-alerts.v1 --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1
docker exec -it cardiofit-kafka kafka-topics --create --topic urgent-alerts.v1 --bootstrap-server localhost:9092 --partitions 4 --replication-factor 1
```

### Step 7: Build and Start Notification Service

```bash
# Build the Docker image
docker-compose build notification-service

# Start notification service
docker-compose --env-file .env.docker up -d notification-service

# View logs
docker-compose logs -f notification-service
```

**Expected Output**:
```
cardiofit-notification-service-go | 2025-11-11 10:30:00 INFO  Starting Notification Service
cardiofit-notification-service-go | 2025-11-11 10:30:01 INFO  Connected to PostgreSQL
cardiofit-notification-service-go | 2025-11-11 10:30:01 INFO  Connected to Redis
cardiofit-notification-service-go | 2025-11-11 10:30:02 INFO  Connected to Kafka (bootstrap: kafka:9092)
cardiofit-notification-service-go | 2025-11-11 10:30:02 INFO  HTTP server listening on :8080
cardiofit-notification-service-go | 2025-11-11 10:30:02 INFO  gRPC server listening on :50051
cardiofit-notification-service-go | 2025-11-11 10:30:02 INFO  Notification Service ready
```

### Step 8: Verify Services

```bash
# Check all services are running
docker-compose ps

# Test notification service health
curl http://localhost:8080/health

# Expected: {"status":"healthy","timestamp":"..."}

# Test database connection
docker exec -it cardiofit-postgres psql -U cardiofit_user -d cardiofit_db -c "SELECT 1;"

# Test Redis connection
docker exec -it cardiofit-redis redis-cli ping

# Test Kafka connection
docker exec -it cardiofit-kafka kafka-broker-api-versions --bootstrap-server localhost:9092
```

### Step 9: Start Optional Monitoring (Optional)

```bash
# Start Prometheus and Grafana
docker-compose --profile monitoring up -d

# Access Grafana at http://localhost:3000
# Username: admin
# Password: (from GRAFANA_PASSWORD in .env.docker, default: admin)
```

### Step 10: Start Optional Tools (Optional)

```bash
# Start Redis Commander and Kafka UI
docker-compose --profile tools up -d

# Redis Commander: http://localhost:8081
# Kafka UI: http://localhost:8082
```

---

## Testing the Setup

### Test 1: Send a Test Notification via HTTP API

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "alert_id": "test_alert_001",
    "patient_id": "patient_12345",
    "severity": "CRITICAL",
    "type": "CARDIAC_ARREST",
    "title": "Test Alert",
    "message": "This is a test notification",
    "target_roles": ["attending_physician"],
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
  }'
```

### Test 2: Check Redis for Alert Tracking

```bash
# Check alert counter
docker exec -it cardiofit-redis redis-cli GET "alert:counter:test_user:1h"

# Check recent alerts
docker exec -it cardiofit-redis redis-cli LRANGE "alert:recent:test_user" 0 -1

# Check cached preferences
docker exec -it cardiofit-redis redis-cli GET "user:prefs:test_user"
```

### Test 3: Check PostgreSQL for User Preferences

```bash
docker exec -it cardiofit-postgres psql -U cardiofit_user -d cardiofit_db -c "SELECT * FROM notification_service.user_preferences LIMIT 5;"
```

### Test 4: Send Test Message to Kafka

```bash
# Produce a test message
docker exec -it cardiofit-kafka kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 <<EOF
{
  "alert_id": "kafka_test_001",
  "patient_id": "patient_67890",
  "severity": "HIGH",
  "type": "ABNORMAL_VITAL",
  "title": "High Heart Rate",
  "message": "Patient heart rate: 120 bpm",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

# Check notification service logs for processing
docker-compose logs -f notification-service | grep kafka_test_001
```

---

## Common Commands

### Service Management

```bash
# Start all services
docker-compose up -d

# Start specific service
docker-compose up -d notification-service

# Stop all services
docker-compose down

# Stop and remove volumes (DANGER: deletes all data)
docker-compose down -v

# Restart a service
docker-compose restart notification-service

# View logs
docker-compose logs -f notification-service

# View logs for all services
docker-compose logs -f
```

### Debugging

```bash
# Enter notification service container
docker exec -it cardiofit-notification-service-go sh

# Check environment variables
docker exec cardiofit-notification-service-go env

# Check process status
docker exec cardiofit-notification-service-go ps aux

# View health check
curl http://localhost:8080/health | jq

# View metrics (if Prometheus enabled)
curl http://localhost:8080/metrics
```

### Database Operations

```bash
# Connect to PostgreSQL
docker exec -it cardiofit-postgres psql -U cardiofit_user -d cardiofit_db

# Backup database
docker exec cardiofit-postgres pg_dump -U cardiofit_user cardiofit_db > backup.sql

# Restore database
cat backup.sql | docker exec -i cardiofit-postgres psql -U cardiofit_user -d cardiofit_db
```

### Kafka Operations

```bash
# List topics
docker exec -it cardiofit-kafka kafka-topics --list --bootstrap-server localhost:9092

# Describe topic
docker exec -it cardiofit-kafka kafka-topics --describe --topic enriched-patient-events-v1 --bootstrap-server localhost:9092

# Consume messages
docker exec -it cardiofit-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning

# Check consumer group lag
docker exec -it cardiofit-kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --describe \
  --group notification-service
```

---

## Troubleshooting

### Issue: Notification Service Won't Start

**Check logs**:
```bash
docker-compose logs notification-service
```

**Common Causes**:
1. **Database not ready**: Wait for `cardiofit-postgres` to be healthy
2. **Missing environment variables**: Check `.env.docker` file
3. **Port conflict**: Another service using port 8080 or 50051
4. **Build failure**: Run `docker-compose build --no-cache notification-service`

### Issue: Cannot Connect to PostgreSQL

```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Test connection from host
docker exec -it cardiofit-postgres pg_isready

# Check logs
docker-compose logs postgres

# Verify credentials
docker exec -it cardiofit-postgres psql -U cardiofit_user -d cardiofit_db -c "SELECT 1;"
```

### Issue: Redis Connection Failed

```bash
# Check Redis is running
docker-compose ps redis

# Test connection
docker exec -it cardiofit-redis redis-cli ping

# Check logs
docker-compose logs redis
```

### Issue: Kafka Not Receiving Messages

```bash
# Check Kafka and Zookeeper are running
docker-compose ps kafka zookeeper

# Check Kafka logs
docker-compose logs kafka

# Verify topics exist
docker exec -it cardiofit-kafka kafka-topics --list --bootstrap-server localhost:9092

# Test producing a message
docker exec -it cardiofit-kafka kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1
# Type a message and press Enter

# Test consuming
docker exec -it cardiofit-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning
```

### Issue: External Services Not Sending

**Check delivery mode**:
```bash
docker exec cardiofit-notification-service-go env | grep NOTIFICATION_DELIVERY_MODE
```

**If "mock"**: Notifications won't actually send (expected behavior)

**If "live"**: Check external service credentials:
```bash
docker exec cardiofit-notification-service-go env | grep TWILIO
docker exec cardiofit-notification-service-go env | grep SENDGRID
docker exec cardiofit-notification-service-go env | grep FIREBASE
```

**Test Twilio**:
```bash
curl -X GET "https://api.twilio.com/2010-04-01/Accounts/$TWILIO_ACCOUNT_SID/Messages.json" \
  -u "$TWILIO_ACCOUNT_SID:$TWILIO_AUTH_TOKEN"
```

---

## Production Deployment Checklist

### Before Production

- [ ] All external service credentials configured
- [ ] Firebase credentials file uploaded securely
- [ ] Database backups configured
- [ ] Persistent volumes configured
- [ ] Monitoring enabled (Prometheus + Grafana)
- [ ] Log aggregation configured
- [ ] Health checks working
- [ ] Resource limits set (CPU, memory)
- [ ] Security: non-root user, minimal image
- [ ] Network security: firewall rules, VPC
- [ ] SSL/TLS certificates for external endpoints

### Production Environment Variables

```bash
# Production settings
ENVIRONMENT=production
LOG_LEVEL=info
NOTIFICATION_DELIVERY_MODE=live

# Use managed services for production
DATABASE_URL=postgres://user:pass@prod-db.rds.amazonaws.com:5432/cardiofit?sslmode=require
REDIS_ADDR=prod-redis.cache.amazonaws.com:6379
KAFKA_BROKERS=kafka1.prod.com:9092,kafka2.prod.com:9092,kafka3.prod.com:9092
```

### Resource Limits (docker-compose.yml)

```yaml
notification-service:
  deploy:
    resources:
      limits:
        cpus: '2.0'
        memory: 2G
      reservations:
        cpus: '1.0'
        memory: 1G
```

---

## Next Steps

1. ✅ Review [EXTERNAL_SERVICES_GUIDE.md](./EXTERNAL_SERVICES_GUIDE.md) for external service setup
2. ✅ Configure `.env.docker` with your credentials
3. ✅ Test in mock mode first
4. ✅ Test with real external services in staging
5. ✅ Deploy to production
6. ✅ Enable monitoring and alerting
7. ✅ Set up log aggregation
8. ✅ Configure automated backups

For integration testing, see [tests/integration/README.md](./tests/integration/README.md)
