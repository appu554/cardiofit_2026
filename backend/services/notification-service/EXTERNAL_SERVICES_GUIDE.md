# External Services Required for Notification Service

This guide covers all external services required for the CardioFit Notification Service to function properly.

---

## Overview

The Notification Service integrates with **7 external services** across 3 categories:

### 1. **Infrastructure Services** (Required)
- PostgreSQL (Database)
- Redis (Cache & Alert Fatigue)
- Kafka (Event Streaming)

### 2. **Notification Delivery Services** (Required for production)
- Twilio (SMS & Voice calls)
- SendGrid (Email)
- Firebase Cloud Messaging (Push notifications)

### 3. **Monitoring Services** (Optional but recommended)
- Prometheus (Metrics)
- Grafana (Dashboards)

---

## 1. Infrastructure Services (REQUIRED)

### PostgreSQL Database

**Purpose**: Store user preferences, notification history, and configuration

**Required Version**: PostgreSQL 14+

**Setup Options**:

#### Option A: Docker (Development)
```bash
docker run --name cardiofit-postgres \
  -e POSTGRES_USER=cardiofit_user \
  -e POSTGRES_PASSWORD=cardiofit_pass \
  -e POSTGRES_DB=cardiofit_db \
  -p 5432:5432 \
  -d postgres:14
```

#### Option B: Managed Service (Production)
- **AWS RDS**: PostgreSQL 14+
- **Google Cloud SQL**: PostgreSQL 14+
- **Azure Database**: PostgreSQL 14+

**Required Schema**:
```sql
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
```

**Connection String Format**:
```
postgres://username:password@host:port/database?sslmode=require
```

**Environment Variables**:
```bash
DATABASE_URL="postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
```

---

### Redis Cache

**Purpose**: Cache user preferences, track alert fatigue, manage rate limiting

**Required Version**: Redis 6+

**Setup Options**:

#### Option A: Docker (Development)
```bash
docker run --name cardiofit-redis \
  -p 6379:6379 \
  -d redis:7-alpine
```

#### Option B: Managed Service (Production)
- **AWS ElastiCache**: Redis 6+
- **Google Cloud Memorystore**: Redis 6+
- **Azure Cache**: Redis 6+

**Required Configuration**:
```conf
# redis.conf
maxmemory 2gb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
save 60 10000
```

**Environment Variables**:
```bash
REDIS_ADDR="localhost:6379"
REDIS_PASSWORD=""  # Leave empty if no password
REDIS_DB=0
```

**Key Patterns Used**:
```
user:prefs:{user_id}              # User preference cache (5 min TTL)
alert:counter:{user_id}:{window}  # Alert count tracking
alert:recent:{user_id}            # Recent alerts list (last 100)
alert:dedup:{alert_id}            # Alert deduplication (5 min TTL)
session:{session_id}              # Session data
```

---

### Apache Kafka

**Purpose**: Consume alert events from Flink processing pipeline

**Required Version**: Kafka 3.0+

**Setup Options**:

#### Option A: Docker (Development)
```bash
# Zookeeper
docker run --name cardiofit-zookeeper \
  -e ZOOKEEPER_CLIENT_PORT=2181 \
  -e ZOOKEEPER_TICK_TIME=2000 \
  -p 2181:2181 \
  -d confluentinc/cp-zookeeper:7.5.0

# Kafka
docker run --name cardiofit-kafka \
  -e KAFKA_BROKER_ID=1 \
  -e KAFKA_ZOOKEEPER_CONNECT=localhost:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  -p 9092:9092 \
  -d confluentinc/cp-kafka:7.5.0
```

#### Option B: Managed Service (Production)
- **Confluent Cloud**: Fully managed Kafka
- **AWS MSK**: Managed Streaming for Kafka
- **Azure Event Hubs**: Kafka-compatible

**Required Topics**:
```bash
# Alert topics consumed by notification service
kafka-topics --create --topic enriched-patient-events-v1 --partitions 4 --replication-factor 3
kafka-topics --create --topic clinical-patterns.v1 --partitions 8 --replication-factor 3
kafka-topics --create --topic composed-alerts.v1 --partitions 4 --replication-factor 3
kafka-topics --create --topic urgent-alerts.v1 --partitions 4 --replication-factor 3
```

**Environment Variables**:
```bash
KAFKA_BROKERS="localhost:9092"
KAFKA_CONSUMER_GROUP="notification-service"
KAFKA_TOPICS="enriched-patient-events-v1,clinical-patterns.v1,composed-alerts.v1,urgent-alerts.v1"
```

---

## 2. Notification Delivery Services

### Twilio (SMS & Voice)

**Purpose**: Send SMS and make voice calls for critical alerts

**Required for**: SMS and Pager (voice call) notifications

**Setup Steps**:

1. **Create Twilio Account**: https://www.twilio.com/try-twilio
2. **Get API Credentials**:
   - Account SID
   - Auth Token
   - Phone Number

3. **Configure Phone Number**:
   - Purchase a phone number capable of SMS
   - Enable voice calling if using pager functionality

**Pricing** (as of 2024):
- SMS: $0.0075 per message (US)
- Voice: $0.0130 per minute (US)
- Free tier: $15 credit for testing

**Environment Variables**:
```bash
TWILIO_ACCOUNT_SID="ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
TWILIO_AUTH_TOKEN="your_auth_token_here"
TWILIO_FROM_NUMBER="+12345678900"
```

**Testing Without Real Account**:
```bash
# Use test credentials (no actual SMS sent)
TWILIO_ACCOUNT_SID="test_account_sid"
TWILIO_AUTH_TOKEN="test_auth_token"
TWILIO_FROM_NUMBER="+15005550006"  # Twilio test number
```

**Code Example**:
```go
import "github.com/twilio/twilio-go"

client := twilio.NewRestClientWithParams(twilio.ClientParams{
    Username: twilioAccountSID,
    Password: twilioAuthToken,
})

params := &api.CreateMessageParams{}
params.SetTo("+1234567890")
params.SetFrom(twilioFromNumber)
params.SetBody("Critical: Patient heart rate 180 bpm")

message, err := client.Api.CreateMessage(params)
```

**Rate Limits**:
- **Development**: 1 message/second per number
- **Production**: 100 messages/second per account
- **Voice calls**: 10 concurrent calls per account

---

### SendGrid (Email)

**Purpose**: Send email notifications

**Required for**: Email channel notifications

**Setup Steps**:

1. **Create SendGrid Account**: https://signup.sendgrid.com/
2. **Create API Key**:
   - Settings → API Keys → Create API Key
   - Permission: Full Access (or Mail Send only)
3. **Verify Sender Identity**:
   - Sender Authentication → Verify a Single Sender
   - Or set up Domain Authentication for production

**Pricing** (as of 2024):
- **Free tier**: 100 emails/day
- **Essentials**: $19.95/month (50,000 emails/month)
- **Pro**: $89.95/month (1.5M emails/month)

**Environment Variables**:
```bash
SENDGRID_API_KEY="SG.xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
SENDGRID_FROM_EMAIL="notifications@cardiofit.com"
SENDGRID_FROM_NAME="CardioFit Notifications"
```

**Testing Without Real Account**:
```bash
# Use mock mode (no actual emails sent)
SENDGRID_API_KEY="test_api_key"
SENDGRID_FROM_EMAIL="test@example.com"
NOTIFICATION_DELIVERY_MODE="mock"  # Enable mock mode
```

**Code Example**:
```go
import "github.com/sendgrid/sendgrid-go/helpers/mail"

from := mail.NewEmail("CardioFit", "notifications@cardiofit.com")
to := mail.NewEmail("Dr. Smith", "dr.smith@hospital.com")
subject := "Critical Alert: Patient Status Change"
plainText := "Patient heart rate: 180 bpm. Immediate intervention required."
htmlContent := "<strong>Critical Alert</strong><br>Patient heart rate: 180 bpm"

message := mail.NewSingleEmail(from, subject, to, plainText, htmlContent)
client := sendgrid.NewSendClient(sendgridAPIKey)
response, err := client.Send(message)
```

**Rate Limits**:
- **Free tier**: 100 emails/day
- **Paid plans**: No rate limit (throttled by plan)
- **Recommended**: 1,000 emails/minute per account

---

### Firebase Cloud Messaging (Push Notifications)

**Purpose**: Send push notifications to mobile devices

**Required for**: Push notification channel

**Setup Steps**:

1. **Create Firebase Project**: https://console.firebase.google.com/
2. **Add Android/iOS App**: Project Settings → Add app
3. **Download Service Account Key**:
   - Project Settings → Service Accounts
   - Generate new private key (JSON file)
4. **Enable FCM API**: Cloud Messaging → Enable API

**Pricing**:
- **Free**: Unlimited push notifications
- **No cost** for Firebase Cloud Messaging

**Environment Variables**:
```bash
FIREBASE_CREDENTIALS_PATH="/path/to/firebase-service-account.json"
FIREBASE_PROJECT_ID="cardiofit-prod"
```

**Service Account JSON Structure**:
```json
{
  "type": "service_account",
  "project_id": "cardiofit-prod",
  "private_key_id": "xxxxxxxxxxxxxxxxxxxxx",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "client_email": "firebase-adminsdk-xxxxx@cardiofit-prod.iam.gserviceaccount.com",
  "client_id": "xxxxxxxxxxxxxxxxxxxxx",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/..."
}
```

**Code Example**:
```go
import firebase "firebase.google.com/go/v4"

app, err := firebase.NewApp(context.Background(), &firebase.Config{
    ProjectID: "cardiofit-prod",
}, option.WithCredentialsFile("/path/to/service-account.json"))

messaging, err := app.Messaging(ctx)

message := &messaging.Message{
    Token: "user_device_token_here",
    Notification: &messaging.Notification{
        Title: "Critical Alert",
        Body: "Patient heart rate: 180 bpm",
    },
    Data: map[string]string{
        "alert_id": "alert_12345",
        "severity": "CRITICAL",
        "patient_id": "patient_67890",
    },
}

response, err := messaging.Send(ctx, message)
```

**Testing Without Real Firebase**:
```bash
# Use mock mode
FIREBASE_ENABLED=false
NOTIFICATION_DELIVERY_MODE="mock"
```

**Rate Limits**:
- **No hard limits** from Firebase
- **Recommended**: 1,000 messages/second
- **Batch support**: Up to 500 messages per batch

---

## 3. Monitoring Services (OPTIONAL)

### Prometheus (Metrics)

**Purpose**: Collect and store metrics from notification service

**Setup**:
```bash
docker run --name cardiofit-prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  -d prom/prometheus:latest
```

**prometheus.yml**:
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'notification-service'
    static_configs:
      - targets: ['notification-service:8080']
    metrics_path: '/metrics'
```

---

### Grafana (Dashboards)

**Purpose**: Visualize metrics and create dashboards

**Setup**:
```bash
docker run --name cardiofit-grafana \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  -d grafana/grafana:latest
```

**Access**: http://localhost:3000 (admin/admin)

---

## Complete Environment Variables

### Production Configuration

```bash
# Infrastructure
DATABASE_URL="postgres://user:pass@db.cardiofit.com:5432/cardiofit?sslmode=require"
REDIS_ADDR="redis.cardiofit.com:6379"
REDIS_PASSWORD="secure_redis_password"
KAFKA_BROKERS="kafka1.cardiofit.com:9092,kafka2.cardiofit.com:9092,kafka3.cardiofit.com:9092"

# Twilio (SMS/Voice)
TWILIO_ACCOUNT_SID="ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
TWILIO_AUTH_TOKEN="your_production_auth_token"
TWILIO_FROM_NUMBER="+18005551234"

# SendGrid (Email)
SENDGRID_API_KEY="SG.production_api_key_here"
SENDGRID_FROM_EMAIL="notifications@cardiofit.com"
SENDGRID_FROM_NAME="CardioFit Alerts"

# Firebase (Push)
FIREBASE_CREDENTIALS_PATH="/etc/secrets/firebase-prod.json"
FIREBASE_PROJECT_ID="cardiofit-prod"

# Service Configuration
SERVER_PORT=8080
GRPC_PORT=50051
LOG_LEVEL=info
ENVIRONMENT=production
```

### Development Configuration

```bash
# Infrastructure (Docker)
DATABASE_URL="postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
REDIS_ADDR="localhost:6379"
REDIS_PASSWORD=""
KAFKA_BROKERS="localhost:9092"

# External Services (Test Mode)
NOTIFICATION_DELIVERY_MODE="mock"  # Don't send real notifications
TWILIO_ACCOUNT_SID="test_account_sid"
TWILIO_AUTH_TOKEN="test_auth_token"
SENDGRID_API_KEY="test_api_key"
FIREBASE_ENABLED=false

# Service Configuration
SERVER_PORT=8080
GRPC_PORT=50051
LOG_LEVEL=debug
ENVIRONMENT=development
```

---

## Cost Estimation

### Monthly Cost Breakdown (10,000 alerts/month)

| Service | Usage | Cost |
|---------|-------|------|
| **PostgreSQL (AWS RDS)** | db.t3.small | $25/month |
| **Redis (ElastiCache)** | cache.t3.micro | $12/month |
| **Kafka (MSK)** | kafka.t3.small x2 | $80/month |
| **Twilio SMS** | 5,000 messages | $37.50/month |
| **SendGrid** | 3,000 emails | $0 (free tier) |
| **Firebase** | 10,000 push | $0 (free) |
| **Total Infrastructure** | | **$154.50/month** |

### Production Scale (100,000 alerts/month)

| Service | Usage | Cost |
|---------|-------|------|
| **PostgreSQL** | db.m5.large | $180/month |
| **Redis** | cache.m5.large | $100/month |
| **Kafka** | kafka.m5.large x3 | $600/month |
| **Twilio SMS** | 50,000 messages | $375/month |
| **SendGrid** | 30,000 emails | $19.95/month |
| **Firebase** | 100,000 push | $0 (free) |
| **Total** | | **$1,274.95/month** |

---

## Testing Without External Services

### Mock Mode

Enable mock mode to run the notification service without real external API calls:

```bash
# .env.test
NOTIFICATION_DELIVERY_MODE="mock"
TWILIO_ENABLED=false
SENDGRID_ENABLED=false
FIREBASE_ENABLED=false
```

**Mock behavior**:
- All delivery methods return success
- No actual SMS/emails/push notifications sent
- Delivery status can be queried
- Suitable for integration testing

### Local Testing Stack

```bash
# Start only infrastructure services
docker-compose up -d postgres redis kafka

# Run notification service with mock delivery
go run cmd/notification-service/main.go --mock-delivery
```

---

## Security Best Practices

### API Keys and Secrets

1. **Never commit secrets** to version control
2. **Use environment variables** or secret management services
3. **Rotate credentials** regularly (every 90 days)
4. **Use different credentials** for dev/staging/production

### Secret Management Services

- **AWS Secrets Manager**: Store and rotate secrets
- **Google Secret Manager**: Centralized secret storage
- **HashiCorp Vault**: Enterprise secret management
- **Azure Key Vault**: Cloud secret storage

### Network Security

1. **Use SSL/TLS** for all external connections
2. **Whitelist IP addresses** for database and Redis
3. **Enable VPC peering** for internal services
4. **Use private subnets** for sensitive services

---

## Troubleshooting

### Cannot Connect to PostgreSQL

```bash
# Test connection
psql -h localhost -U cardiofit_user -d cardiofit_db

# Common issues:
# 1. Firewall blocking port 5432
# 2. PostgreSQL not accepting remote connections
# 3. Wrong credentials

# Solution: Check pg_hba.conf and postgresql.conf
```

### Cannot Connect to Redis

```bash
# Test connection
redis-cli -h localhost -p 6379 ping

# Common issues:
# 1. Redis not running
# 2. Protected mode enabled
# 3. Wrong host/port

# Solution: Check redis.conf
```

### Twilio SMS Not Sending

```bash
# Check Twilio logs
curl -X GET "https://api.twilio.com/2010-04-01/Accounts/$TWILIO_ACCOUNT_SID/Messages.json" \
     -u "$TWILIO_ACCOUNT_SID:$TWILIO_AUTH_TOKEN"

# Common issues:
# 1. Invalid phone number format (must include country code)
# 2. Insufficient balance
# 3. Number not verified (trial accounts)
```

### SendGrid Emails Not Sending

```bash
# Check SendGrid activity
curl "https://api.sendgrid.com/v3/messages" \
     -H "Authorization: Bearer $SENDGRID_API_KEY"

# Common issues:
# 1. Sender not verified
# 2. API key permissions insufficient
# 3. Domain not authenticated (production)
```

---

## Next Steps

1. ✅ Review this guide and choose your external services
2. ✅ Set up infrastructure services (PostgreSQL, Redis, Kafka)
3. ✅ Create accounts for Twilio, SendGrid, Firebase
4. ✅ Configure environment variables
5. ✅ Test connection to each service
6. ✅ Run integration tests
7. ✅ Deploy to production

For Docker setup, see [Docker Compose Configuration](./docker-compose.yml)
