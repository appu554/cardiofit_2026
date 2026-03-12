# Notification Service Deployment Guide

Complete deployment guide for the CardioFit Notification Service across different environments.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Local Development](#local-development)
3. [Docker Deployment](#docker-deployment)
4. [Kubernetes Deployment](#kubernetes-deployment)
5. [Production Configuration](#production-configuration)
6. [Monitoring & Alerting](#monitoring--alerting)
7. [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Services
- **Kafka**: 3.0+ (message streaming)
- **Redis**: 6.0+ (state management)
- **Java**: 17+ (application runtime)
- **Maven**: 3.8+ (build tool)

### External Service Accounts
1. **Twilio** (SMS notifications)
   - Account SID
   - Auth Token
   - Phone Number

2. **SendGrid** (Email notifications)
   - API Key
   - Verified sender email

3. **Firebase** (Push notifications)
   - Firebase project
   - Service account credentials JSON

## Local Development

### Step 1: Clone and Configure

```bash
# Navigate to service directory
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/module6-services/notification-service

# Copy environment template
cp .env.example .env

# Edit configuration
nano .env
```

### Step 2: Configure Environment Variables

```bash
# .env file
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
REDIS_HOST=localhost
REDIS_PORT=6379

# Twilio
TWILIO_ACCOUNT_SID=your_account_sid_here
TWILIO_AUTH_TOKEN=your_auth_token_here
TWILIO_PHONE_NUMBER=+1234567890

# SendGrid
SENDGRID_API_KEY=SG.your_api_key_here
SENDGRID_FROM_EMAIL=alerts@cardiofit.com

# Firebase
FIREBASE_CREDENTIALS_PATH=firebase-credentials.json
FIREBASE_ENABLED=true
```

### Step 3: Setup Firebase

```bash
# Download service account key from Firebase Console
# Save as firebase-credentials.json
# Set proper permissions
chmod 600 firebase-credentials.json
```

### Step 4: Start Dependencies

```bash
# Start Kafka (if not running)
docker run -d --name kafka \
  -p 9092:9092 \
  -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
  confluentinc/cp-kafka:7.5.0

# Start Redis (if not running)
docker run -d --name redis \
  -p 6379:6379 \
  redis:7-alpine

# Create Kafka topic
kafka-topics --bootstrap-server localhost:9092 \
  --create --topic composed-alerts \
  --partitions 3 --replication-factor 1
```

### Step 5: Build and Run

```bash
# Build application
make build

# Run service
make run

# Or use script directly
./run.sh
```

### Step 6: Verify Service

```bash
# Check health
curl http://localhost:8070/api/v1/notifications/health

# Run integration tests
make integration-test
```

## Docker Deployment

### Step 1: Configure Environment

```bash
# Create .env file with credentials
cat > .env << EOF
TWILIO_ACCOUNT_SID=your_sid
TWILIO_AUTH_TOKEN=your_token
TWILIO_PHONE_NUMBER=+1234567890
SENDGRID_API_KEY=your_key
SENDGRID_FROM_EMAIL=alerts@cardiofit.com
FIREBASE_ENABLED=true
EOF
```

### Step 2: Add Firebase Credentials

```bash
# Copy Firebase credentials to project root
cp /path/to/firebase-credentials.json .
```

### Step 3: Start Services

```bash
# Build and start all services
make docker-up

# Or manually
docker-compose up -d
```

### Step 4: Verify Deployment

```bash
# Check service logs
docker-compose logs -f notification-service

# Run health check
curl http://localhost:8070/actuator/health

# View metrics
curl http://localhost:8070/actuator/prometheus
```

### Step 5: Access Monitoring

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Service API**: http://localhost:8070

## Kubernetes Deployment

### Step 1: Create Secrets

```bash
# Create namespace
kubectl create namespace cardiofit

# Create Twilio secret
kubectl create secret generic twilio-credentials \
  --from-literal=account-sid=$TWILIO_ACCOUNT_SID \
  --from-literal=auth-token=$TWILIO_AUTH_TOKEN \
  --from-literal=phone-number=$TWILIO_PHONE_NUMBER \
  -n cardiofit

# Create SendGrid secret
kubectl create secret generic sendgrid-credentials \
  --from-literal=api-key=$SENDGRID_API_KEY \
  --from-literal=from-email=$SENDGRID_FROM_EMAIL \
  -n cardiofit

# Create Firebase secret
kubectl create secret generic firebase-credentials \
  --from-file=credentials.json=firebase-credentials.json \
  -n cardiofit
```

### Step 2: Create ConfigMap

```yaml
# notification-service-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: notification-service-config
  namespace: cardiofit
data:
  KAFKA_BOOTSTRAP_SERVERS: "kafka.cardiofit.svc.cluster.local:9092"
  REDIS_HOST: "redis.cardiofit.svc.cluster.local"
  REDIS_PORT: "6379"
  ENVIRONMENT: "production"
```

```bash
kubectl apply -f notification-service-config.yaml
```

### Step 3: Create Deployment

```yaml
# notification-service-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
  namespace: cardiofit
spec:
  replicas: 3
  selector:
    matchLabels:
      app: notification-service
  template:
    metadata:
      labels:
        app: notification-service
    spec:
      containers:
      - name: notification-service
        image: cardiofit-notification-service:latest
        ports:
        - containerPort: 8070
        env:
        - name: SPRING_PROFILES_ACTIVE
          value: "production"
        envFrom:
        - configMapRef:
            name: notification-service-config
        - secretRef:
            name: twilio-credentials
        - secretRef:
            name: sendgrid-credentials
        volumeMounts:
        - name: firebase-credentials
          mountPath: /app/firebase-credentials.json
          subPath: credentials.json
          readOnly: true
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /actuator/health/liveness
            port: 8070
          initialDelaySeconds: 60
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /actuator/health/readiness
            port: 8070
          initialDelaySeconds: 30
          periodSeconds: 5
      volumes:
      - name: firebase-credentials
        secret:
          secretName: firebase-credentials
```

```bash
kubectl apply -f notification-service-deployment.yaml
```

### Step 4: Create Service

```yaml
# notification-service-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: notification-service
  namespace: cardiofit
spec:
  selector:
    app: notification-service
  ports:
  - port: 8070
    targetPort: 8070
  type: ClusterIP
```

```bash
kubectl apply -f notification-service-service.yaml
```

### Step 5: Verify Deployment

```bash
# Check pods
kubectl get pods -n cardiofit -l app=notification-service

# Check logs
kubectl logs -f deployment/notification-service -n cardiofit

# Port forward for testing
kubectl port-forward svc/notification-service 8070:8070 -n cardiofit
```

## Production Configuration

### Security Hardening

```yaml
# application-production.yml
spring:
  data:
    redis:
      ssl: true
      password: ${REDIS_PASSWORD}

  kafka:
    properties:
      security.protocol: SASL_SSL
      sasl.mechanism: PLAIN
      sasl.jaas.config: |
        org.apache.kafka.common.security.plain.PlainLoginModule required
        username="${KAFKA_USERNAME}"
        password="${KAFKA_PASSWORD}";

notification:
  rate-limit:
    max-alerts-per-hour: 30  # Increase for production
  retry:
    max-attempts: 5  # More retries in production
```

### Resource Scaling

```yaml
# Kubernetes HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: notification-service-hpa
  namespace: cardiofit
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: notification-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Monitoring Configuration

```yaml
# ServiceMonitor for Prometheus Operator
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: notification-service
  namespace: cardiofit
spec:
  selector:
    matchLabels:
      app: notification-service
  endpoints:
  - port: http
    path: /actuator/prometheus
    interval: 30s
```

## Monitoring & Alerting

### Key Metrics to Monitor

```promql
# Alert processing rate
rate(alerts_received_total[5m])

# Delivery success rate
rate(notifications_channel_total{status="success"}[5m]) /
rate(notifications_channel_total[5m])

# Rate limiting events
rate(alerts_rate_limited_total[5m])

# Kafka consumer lag
kafka_consumer_lag{topic="composed-alerts"}
```

### Alert Rules

```yaml
# alerts.yml
groups:
- name: notification-service
  rules:
  - alert: HighNotificationFailureRate
    expr: |
      rate(notifications_channel_total{status="failure"}[5m]) /
      rate(notifications_channel_total[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High notification failure rate"

  - alert: RateLimitingExcessive
    expr: rate(alerts_rate_limited_total[5m]) > 10
    for: 10m
    annotations:
      summary: "Excessive rate limiting events"

  - alert: KafkaConsumerLag
    expr: kafka_consumer_lag > 1000
    for: 5m
    annotations:
      summary: "High Kafka consumer lag"
```

## Troubleshooting

### Service Won't Start

```bash
# Check dependencies
docker ps | grep -E "kafka|redis"

# Check environment variables
env | grep -E "TWILIO|SENDGRID|FIREBASE|KAFKA|REDIS"

# Check logs
tail -f logs/notification-service.log

# Validate Kafka connection
kafka-topics --bootstrap-server localhost:9092 --list
```

### Notifications Not Sending

```bash
# Check Twilio status
curl -u "$TWILIO_ACCOUNT_SID:$TWILIO_AUTH_TOKEN" \
  https://api.twilio.com/2010-04-01/Accounts/$TWILIO_ACCOUNT_SID.json

# Test SendGrid
curl -X POST https://api.sendgrid.com/v3/mail/send \
  -H "Authorization: Bearer $SENDGRID_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"personalizations":[{"to":[{"email":"test@test.com"}]}],"from":{"email":"'$SENDGRID_FROM_EMAIL'"},"subject":"Test","content":[{"type":"text/plain","value":"Test"}]}'

# Verify Firebase credentials
cat firebase-credentials.json | jq .
```

### High Memory Usage

```bash
# Check JVM settings
docker exec notification-service java -XX:+PrintFlagsFinal -version | grep MaxHeapSize

# Adjust in docker-compose.yml
environment:
  JAVA_OPTS: "-Xmx1g -XX:MaxRAMPercentage=75.0"
```

### Kafka Consumer Lag

```bash
# Check consumer group
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group notification-service-group --describe

# Reset offsets if needed
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group notification-service-group --reset-offsets \
  --to-latest --topic composed-alerts --execute
```

## Performance Tuning

### Kafka Consumer

```yaml
spring:
  kafka:
    consumer:
      max-poll-records: 500  # Increase for higher throughput
      concurrency: 5  # More consumer threads
```

### Redis Connection Pool

```yaml
spring:
  data:
    redis:
      lettuce:
        pool:
          max-active: 16
          max-idle: 8
```

### Async Thread Pool

```yaml
# application.yml
spring:
  task:
    execution:
      pool:
        core-size: 10
        max-size: 20
        queue-capacity: 100
```

## Backup & Recovery

### Redis Backup

```bash
# Backup Redis data
docker exec cardiofit-redis redis-cli BGSAVE

# Export RDB file
docker cp cardiofit-redis:/data/dump.rdb ./backup/
```

### Configuration Backup

```bash
# Backup all configs
tar -czf notification-service-config-backup.tar.gz \
  .env firebase-credentials.json application*.yml
```

## Support & Resources

- **Documentation**: See README.md for detailed API documentation
- **Health Check**: `GET /actuator/health`
- **Metrics**: `GET /actuator/prometheus`
- **Logs**: Check `logs/notification-service.log`

For additional support, contact the CardioFit platform team.
