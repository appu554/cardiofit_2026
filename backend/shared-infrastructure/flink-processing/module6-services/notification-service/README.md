# CardioFit Notification Service

Multi-channel notification delivery service with comprehensive alert fatigue mitigation for the CardioFit clinical platform.

## Features

### Multi-Channel Delivery
- **SMS**: Twilio integration for text message alerts
- **Email**: SendGrid integration for detailed email notifications
- **Push Notifications**: Firebase Cloud Messaging for mobile alerts
- **Pager**: SMS gateway integration for pager systems

### Alert Fatigue Mitigation
- **Rate Limiting**: Maximum 20 alerts per hour per user (configurable)
- **Duplicate Suppression**: 5-minute deduplication window
- **Alert Bundling**: Group similar alerts into summary notifications
- **Smart Routing**: Severity-based channel selection

### User Management
- **Preference Management**: Per-user notification preferences
- **Quiet Hours**: Configurable do-not-disturb periods
- **On-Call Scheduling**: Integration with on-call rotation
- **Escalation**: Automatic escalation to backup users

### Monitoring & Observability
- **Prometheus Metrics**: Alert rates, delivery success, channel performance
- **Delivery Tracking**: Complete audit trail of notification delivery
- **Health Checks**: Kubernetes-ready health and readiness probes
- **Grafana Dashboards**: Pre-configured monitoring dashboards

## Architecture

```
Kafka (composed-alerts) → NotificationRouter → AlertFatigueTracker → DeliveryService
                              ↓                         ↓
                        UserPreferences          Rate Limiting
                                                 Deduplication
                                                 Bundling
```

## Prerequisites

- Java 17+
- Maven 3.8+
- Redis 6.0+
- Kafka 3.0+
- Docker & Docker Compose (for containerized deployment)

### External Service Accounts
- Twilio account (SMS)
- SendGrid account (Email)
- Firebase project (Push notifications)

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Kafka
KAFKA_BOOTSTRAP_SERVERS=localhost:9092

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Twilio
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_PHONE_NUMBER=+1234567890

# SendGrid
SENDGRID_API_KEY=your_api_key
SENDGRID_FROM_EMAIL=alerts@cardiofit.com

# Firebase
FIREBASE_CREDENTIALS_PATH=firebase-credentials.json
FIREBASE_ENABLED=true
```

### Firebase Setup

1. Create a Firebase project at https://console.firebase.google.com
2. Generate a service account key (JSON)
3. Save as `firebase-credentials.json` in the project root
4. Add to `.gitignore` (already configured)

## Building

### Maven Build

```bash
# Build without tests
mvn clean package -DskipTests

# Build with tests
mvn clean package

# Build Docker image
docker build -t cardiofit-notification-service:latest .
```

## Running

### Local Development

```bash
# Set environment variables
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092
export REDIS_HOST=localhost
export TWILIO_ACCOUNT_SID=your_sid
export TWILIO_AUTH_TOKEN=your_token
export TWILIO_PHONE_NUMBER=+1234567890
export SENDGRID_API_KEY=your_key
export SENDGRID_FROM_EMAIL=alerts@cardiofit.com

# Run application
java -jar target/notification-service-1.0.0.jar
```

### Docker Compose

```bash
# Start all services (notification-service, Kafka, Redis, Prometheus, Grafana)
docker-compose up -d

# View logs
docker-compose logs -f notification-service

# Stop services
docker-compose down
```

### Kubernetes

```bash
# Apply Kubernetes manifests (if available)
kubectl apply -f k8s/

# Check pod status
kubectl get pods -l app=notification-service

# View logs
kubectl logs -f deployment/notification-service
```

## API Endpoints

### Health & Monitoring

```bash
# Health check
curl http://localhost:8070/api/v1/notifications/health

# Statistics
curl http://localhost:8070/api/v1/notifications/stats

# Prometheus metrics
curl http://localhost:8070/actuator/prometheus
```

### User Preferences

```bash
# Get user preferences
curl http://localhost:8070/api/v1/notifications/preferences/{userId}

# Update user preferences
curl -X PUT http://localhost:8070/api/v1/notifications/preferences/{userId} \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user123",
    "email": "user@example.com",
    "phoneNumber": "+1234567890",
    "enabledChannels": ["EMAIL", "SMS", "PUSH"],
    "alertBundlingEnabled": true
  }'
```

### Rate Limiting

```bash
# Get rate limit status
curl http://localhost:8070/api/v1/notifications/rate-limit/{userId}

# Reset rate limit (admin)
curl -X POST http://localhost:8070/api/v1/notifications/rate-limit/{userId}/reset
```

### Delivery Tracking

```bash
# Get delivery record
curl http://localhost:8070/api/v1/notifications/delivery/{deliveryId}
```

## Testing

### Unit Tests

```bash
mvn test
```

### Integration Tests

```bash
# With Testcontainers (requires Docker)
mvn verify
```

### Manual Testing with Kafka

```bash
# Produce test alert to Kafka
kafka-console-producer --broker-list localhost:9092 --topic composed-alerts

# Paste JSON:
{
  "alert_id": "test-123",
  "patient_id": "patient-456",
  "patient_name": "John Doe",
  "alert_type": "CRITICAL_VITAL_SIGN",
  "severity": "CRITICAL",
  "title": "Critical Heart Rate",
  "message": "Heart rate exceeds threshold: 180 bpm",
  "timestamp": "2025-11-04T10:30:00Z",
  "assigned_to": ["user123"],
  "priority_score": 0.95
}
```

## User Preference Configuration

### Example User Preference JSON

```json
{
  "userId": "user123",
  "email": "clinician@hospital.com",
  "phoneNumber": "+1234567890",
  "fcmToken": "firebase_token_here",
  "enabledChannels": ["EMAIL", "SMS", "PUSH"],
  "severityThresholds": {
    "CRITICAL": ["SMS", "PUSH", "PAGER", "EMAIL"],
    "HIGH": ["SMS", "PUSH", "EMAIL"],
    "MEDIUM": ["PUSH", "EMAIL"],
    "LOW": ["EMAIL"]
  },
  "quietHours": {
    "enabled": true,
    "startHour": 22,
    "endHour": 7,
    "overrideCritical": true
  },
  "onCallSchedule": {
    "onCall": true,
    "shiftStart": "08:00",
    "shiftEnd": "20:00",
    "escalationUserId": "backup-user"
  },
  "alertBundlingEnabled": true,
  "bundlingWindowMinutes": 10
}
```

## Alert Fatigue Mitigation

### Rate Limiting
- Default: 20 alerts/hour per user
- Critical alerts bypass rate limiting
- Configurable per deployment
- Redis-backed for distributed systems

### Deduplication
- 5-minute window (configurable)
- Based on: patient + alert type + severity
- Prevents duplicate notifications
- Logged for audit trail

### Bundling
- Groups similar alerts into summaries
- 10-minute window (configurable)
- Maximum 5 alerts per bundle
- Critical alerts never bundled
- User-configurable opt-in

## Monitoring

### Prometheus Metrics

```
# Alert processing
alerts.received - Total alerts received from Kafka
alerts.processed - Total alerts successfully processed
alerts.rate_limited - Alerts blocked by rate limiting
alerts.suppressed - Alerts suppressed as duplicates
alerts.bundled - Alerts added to bundles

# Channel delivery
notifications.channel{channel="SMS",status="success"}
notifications.channel{channel="EMAIL",status="success"}
notifications.channel{channel="PUSH",status="success"}
```

### Grafana Dashboard

Access at: http://localhost:3000 (default: admin/admin)

Pre-configured panels:
- Alert processing rate
- Delivery success rate by channel
- Rate limiting events
- Fatigue mitigation metrics

## Production Deployment

### Resource Requirements

```yaml
Minimum:
  CPU: 1 core
  Memory: 1GB

Recommended:
  CPU: 2 cores
  Memory: 2GB
  Redis: 512MB
```

### Scaling Considerations

- Kafka consumer concurrency: 3 (configurable)
- Horizontal scaling: Multiple instances share consumer group
- Redis: Single instance or cluster for distributed rate limiting
- Async delivery: Non-blocking notification sending

### Security

- Credentials via environment variables only
- Firebase credentials file permissions: 0600
- TLS for Redis in production
- Kafka SASL/SSL for production
- API authentication (add as needed)

## Troubleshooting

### Common Issues

**Twilio errors:**
```bash
# Verify credentials
curl -u "$TWILIO_ACCOUNT_SID:$TWILIO_AUTH_TOKEN" \
  https://api.twilio.com/2010-04-01/Accounts.json
```

**SendGrid errors:**
```bash
# Test API key
curl -X POST https://api.sendgrid.com/v3/mail/send \
  -H "Authorization: Bearer $SENDGRID_API_KEY" \
  -H "Content-Type: application/json"
```

**Firebase initialization:**
```bash
# Verify credentials file
cat firebase-credentials.json | jq .
```

**Kafka connection:**
```bash
# List topics
kafka-topics --bootstrap-server localhost:9092 --list

# Check consumer group
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group notification-service-group --describe
```

### Logs

```bash
# View application logs
tail -f logs/notification-service.log

# View Docker logs
docker-compose logs -f notification-service

# Increase log level (application.yml)
logging.level.com.cardiofit: DEBUG
```

## Development

### Adding New Notification Channels

1. Add channel to `UserPreference.NotificationChannel` enum
2. Implement delivery method in `DeliveryService`
3. Add routing logic in `NotificationRouter`
4. Update user preference defaults
5. Add configuration properties
6. Update tests

### Extending Alert Fatigue Logic

Edit `AlertFatigueTracker.java`:
- Modify rate limits
- Adjust deduplication rules
- Customize bundling logic
- Add new mitigation strategies

## License

Copyright 2025 CardioFit Platform. All rights reserved.

## Support

For issues or questions:
- GitHub Issues: [repository]/issues
- Email: support@cardiofit.com
- Documentation: https://docs.cardiofit.com
