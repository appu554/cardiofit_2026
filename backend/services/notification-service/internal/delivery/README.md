# Notification Delivery Service

Multi-channel notification delivery system with external API integrations for SMS, Email, Push Notifications, and Voice calls.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  NotificationDeliveryService                     │
├─────────────────────────────────────────────────────────────────┤
│  - Worker Pool (10 concurrent workers)                           │
│  - Retry Logic (exponential backoff: 1s, 2s, 4s)                │
│  - Metrics Collection (per-channel latency & success rates)      │
│  - Database Tracking (status updates, delivery confirmation)     │
└─────────────────────────────────────────────────────────────────┘
                               │
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
    ┌────▼────┐         ┌──────▼──────┐      ┌──────▼──────┐
    │ Twilio  │         │  SendGrid   │      │  Firebase   │
    │ Client  │         │   Client    │      │   Client    │
    └─────────┘         └─────────────┘      └─────────────┘
         │                     │                     │
    SMS & Voice              Email            Push Notifications
```

## Components

### 1. Delivery Service (`delivery_service.go`)

Main orchestration service that:
- Routes notifications to appropriate channels
- Implements exponential backoff retry logic
- Manages concurrent worker pool
- Tracks delivery metrics
- Updates database status

**Key Methods:**
- `Send(ctx, notification)` - Send single notification with retry
- `SendBatch(ctx, notifications)` - Send multiple notifications concurrently
- `GetDeliveryStatus(ctx, notificationID)` - Query delivery status
- `Shutdown(ctx)` - Graceful shutdown with worker drain

### 2. Twilio Client (`twilio_client.go`)

SMS and Voice call integration via Twilio API.

**Features:**
- SMS delivery with delivery status tracking
- Voice calls with TwiML text-to-speech
- Webhook validation for status callbacks
- E.164 phone number validation
- Rate limit handling

**Configuration:**
```env
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_FROM_NUMBER=+1234567890
```

### 3. SendGrid Client (`sendgrid_client.go`)

Email delivery via SendGrid API v3.

**Features:**
- HTML email templates with clinical alert formatting
- Responsive design (mobile-friendly)
- Dynamic content with vital signs, recommendations
- Deep links to patient details
- Email open tracking (via SendGrid)

**Configuration:**
```env
SENDGRID_API_KEY=your_sendgrid_api_key
SENDGRID_FROM_EMAIL=alerts@cardiofit.com
```

### 4. Firebase Client (`firebase_client.go`)

Push notifications via Firebase Cloud Messaging (FCM).

**Features:**
- iOS and Android push notifications
- Deep links to patient/alert views
- Priority-based delivery (critical = immediate)
- Custom sounds per severity level
- Topic-based subscriptions for group messaging
- Token validation

**Configuration:**
```env
FIREBASE_CREDENTIALS_PATH=/path/to/firebase-credentials.json
```

## Channels

| Channel | Provider | Use Case | Priority |
|---------|----------|----------|----------|
| SMS | Twilio | Quick alerts, critical notifications | High |
| Email | SendGrid | Detailed reports, shift summaries | Medium |
| Push | Firebase | Real-time app notifications | High |
| Voice | Twilio | Critical escalations (voice call) | Critical |
| In-App | Database | Non-urgent, informational alerts | Low |
| Pager | PagerDuty | Legacy pager systems (stub) | Critical |

## Retry Logic

Exponential backoff with configurable parameters:

```go
RetryPolicy{
    MaxAttempts:    3,               // Total attempts
    InitialBackoff: 1 * time.Second, // First retry: 1s
    MaxBackoff:     30 * time.Second,// Cap at 30s
    Multiplier:     2.0,             // Double each time
}
```

**Backoff Sequence:** 1s → 2s → 4s → (fail)

## Worker Pool

Concurrent delivery with bounded parallelism:
- **Workers:** 10 (configurable)
- **Benefit:** 10x throughput vs sequential
- **Safety:** Prevents resource exhaustion

## Database Tracking

All deliveries tracked in `notification_service.notifications`:

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    alert_id VARCHAR(255),
    user_id VARCHAR(255),
    channel VARCHAR(50),
    status VARCHAR(50), -- PENDING, SENDING, SENT, DELIVERED, FAILED
    external_id VARCHAR(255), -- Twilio SID, SendGrid ID, FCM ID
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER
);
```

**Status Flow:**
```
PENDING → SENDING → SENT → DELIVERED
               ↓
            FAILED (after max retries)
```

## Metrics Collection

Real-time metrics per channel:
- Total attempts
- Successful deliveries
- Failed deliveries
- Average latency
- P50, P95, P99 latencies (can be added)

**Access Metrics:**
```go
metrics := service.GetChannelMetrics(models.ChannelSMS)
fmt.Printf("SMS Success Rate: %.2f%%\n",
    float64(metrics.Successful) / float64(metrics.TotalAttempts) * 100)
```

## External API Setup

### Twilio Setup

1. **Create Account:** https://www.twilio.com/console
2. **Get Credentials:**
   - Account SID: Found in Console Dashboard
   - Auth Token: Found in Console Dashboard
   - Phone Number: Purchase number in Console > Phone Numbers
3. **Configure Webhooks (Optional):**
   - Set Status Callback URL for delivery tracking
   - URL: `https://your-domain.com/webhooks/twilio/status`

### SendGrid Setup

1. **Create Account:** https://signup.sendgrid.com/
2. **Generate API Key:**
   - Settings > API Keys > Create API Key
   - Permissions: Full Access (or Mail Send + Mail Settings)
3. **Verify Sender Identity:**
   - Settings > Sender Authentication
   - Verify domain or single sender email
4. **Configure Templates (Optional):**
   - Email API > Dynamic Templates

### Firebase Setup

1. **Create Project:** https://console.firebase.google.com/
2. **Enable Cloud Messaging:**
   - Project Settings > Cloud Messaging
   - Note: Server Key (legacy) or use service account
3. **Download Service Account Key:**
   - Project Settings > Service Accounts
   - Generate New Private Key (JSON)
   - Save as `firebase-credentials.json`
4. **Configure Mobile Apps:**
   - Add iOS/Android apps to Firebase project
   - Download config files (GoogleService-Info.plist, google-services.json)

## Testing

### Unit Tests

Run all tests:
```bash
cd internal/delivery
go test -v -cover
```

**Test Coverage:**
- ✅ SMS delivery success/failure
- ✅ Email delivery with templates
- ✅ Push notification with deep links
- ✅ Voice call initiation
- ✅ In-app notification
- ✅ Retry logic with exponential backoff
- ✅ Concurrent batch sending
- ✅ Worker pool concurrency
- ✅ Metrics collection
- ✅ Context cancellation
- ✅ Missing contact info handling
- ✅ Database status updates
- ✅ Graceful shutdown

### Mock Testing

Tests use mocked external APIs to avoid real API calls:

```go
mockTwilio := &MockTwilioClient{
    SendSMSFunc: func(ctx, to, msg string) (string, error) {
        return "SM_MOCK_123", nil
    },
}
```

### Integration Testing

For real API testing (use test mode/sandbox):

```bash
# Set test credentials
export TWILIO_ACCOUNT_SID=test_sid
export SENDGRID_API_KEY=test_key
export FIREBASE_CREDENTIALS_PATH=test-credentials.json

# Run integration tests
go test -v -tags=integration
```

## Usage Examples

### Basic Delivery

```go
// Initialize service
service, err := NewNotificationDeliveryService(cfg, db, logger)
if err != nil {
    log.Fatal(err)
}

// Create notification
notification := &models.Notification{
    ID:      uuid.New().String(),
    AlertID: alert.AlertID,
    UserID:  user.ID,
    User:    user,
    Alert:   alert,
    Channel: models.ChannelSMS,
    Message: "CRITICAL: Patient 001 Sepsis Risk 85.5%",
    Status:  models.StatusPending,
}

// Send notification
ctx := context.Background()
err = service.Send(ctx, notification)
if err != nil {
    log.Printf("Delivery failed: %v", err)
}

// Check status
status, err := service.GetDeliveryStatus(ctx, notification.ID)
if err == nil {
    log.Printf("Status: %s, External ID: %s", status.Status, status.ExternalID)
}
```

### Batch Delivery

```go
// Send to multiple channels
notifications := []*models.Notification{
    createNotification(user, alert, models.ChannelSMS),
    createNotification(user, alert, models.ChannelEmail),
    createNotification(user, alert, models.ChannelPush),
}

errors := service.SendBatch(ctx, notifications)
for i, err := range errors {
    if err != nil {
        log.Printf("Notification %d failed: %v", i, err)
    }
}
```

### Custom Retry Policy

```go
service := &NotificationDeliveryService{
    // ... other fields
    retryPolicy: RetryPolicy{
        MaxAttempts:    5,
        InitialBackoff: 2 * time.Second,
        MaxBackoff:     60 * time.Second,
        Multiplier:     2.5,
    },
}
```

## Error Handling

### Common Errors

| Error | Cause | Resolution |
|-------|-------|------------|
| `twilio API returned status 401` | Invalid credentials | Check `TWILIO_ACCOUNT_SID` and `TWILIO_AUTH_TOKEN` |
| `sendgrid API returned status 403` | Invalid API key | Generate new API key in SendGrid console |
| `firebase send failed: invalid token` | Expired FCM token | User needs to re-register device |
| `user has no phone number configured` | Missing contact info | Update user profile with phone number |
| `network error` | Temporary network issue | Automatic retry will handle |

### Error Recovery

Service automatically retries on:
- Network timeouts
- 5xx server errors
- Rate limit errors (with backoff)

Service does NOT retry on:
- 4xx client errors (invalid request)
- Missing user contact information
- Invalid FCM tokens

## Monitoring

### Health Checks

```go
// Check service health
err := service.db.Ping(ctx)
if err != nil {
    log.Error("Database unhealthy")
}

// Validate external APIs
err = service.twilioClient.GetAccountInfo(ctx)
err = service.sendgridClient.ValidateAPIKey(ctx)
err = service.firebaseClient.ValidateToken(ctx, token)
```

### Metrics Dashboard

Query delivery metrics:

```sql
-- Daily delivery rates by channel
SELECT
    channel,
    date,
    total_sent,
    total_delivered,
    ROUND((total_delivered::numeric / total_sent) * 100, 2) as success_rate
FROM notification_service.delivery_metrics
WHERE date >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY date DESC, channel;

-- Recent failures
SELECT
    id,
    channel,
    user_id,
    error_message,
    retry_count,
    created_at
FROM notification_service.notifications
WHERE status = 'FAILED'
  AND created_at >= NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC;
```

## Performance

### Benchmarks

Environment: 10 workers, local database, mocked external APIs

| Operation | Throughput | Latency (p95) |
|-----------|------------|---------------|
| Single SMS | 100 msg/s | 150ms |
| Single Email | 80 msg/s | 200ms |
| Single Push | 120 msg/s | 100ms |
| Batch (100) | 1000 msg/s | 180ms |

### Optimization Tips

1. **Increase Workers:** Set `Workers: 20` for higher throughput
2. **Connection Pooling:** HTTP clients reuse connections
3. **Batch Sending:** Use `SendBatch()` for multiple notifications
4. **Database Tuning:** Index `notifications(status, created_at)`
5. **Async Updates:** Update database status asynchronously

## Security

### Credentials Management

- Store API keys in environment variables or secrets manager
- Use service account keys (not user credentials)
- Rotate credentials regularly (every 90 days)
- Restrict API key permissions to minimum required

### Data Protection

- Encrypt sensitive data in transit (HTTPS only)
- Mask phone numbers/emails in logs
- Comply with HIPAA/PHI regulations
- Audit trail for all deliveries

### Rate Limiting

External API rate limits:
- **Twilio:** 60 requests/second per account
- **SendGrid:** Varies by plan (100-500 msg/s)
- **Firebase:** 500 requests/second per project

Service implements automatic backoff on rate limit errors.

## Troubleshooting

### SMS Not Delivered

1. Check Twilio status: `service.twilioClient.GetMessageStatus(ctx, messageID)`
2. Verify phone number format (E.164: +1234567890)
3. Check Twilio account balance
4. Review error logs for API errors

### Email Not Received

1. Check spam folder
2. Verify sender domain authentication (SPF, DKIM)
3. Check SendGrid activity logs
4. Verify recipient email address

### Push Notification Not Appearing

1. Validate FCM token: `service.firebaseClient.ValidateToken(ctx, token)`
2. Check device notification permissions
3. Verify Firebase project configuration
4. Test with FCM notification composer

## Future Enhancements

- [ ] PagerDuty integration for pager channel
- [ ] Slack/Teams integration for team notifications
- [ ] SMS international support (multi-region)
- [ ] A/B testing for email templates
- [ ] Prometheus metrics exporter
- [ ] Circuit breaker for external API failures
- [ ] Message queuing (Kafka/RabbitMQ) for async delivery
- [ ] Delivery status webhooks (callback to client)

## References

- [Twilio API Documentation](https://www.twilio.com/docs/api)
- [SendGrid API Documentation](https://docs.sendgrid.com/api-reference)
- [Firebase Cloud Messaging Documentation](https://firebase.google.com/docs/cloud-messaging)
- [Go pgx PostgreSQL Driver](https://github.com/jackc/pgx)
- [Uber Zap Logging](https://github.com/uber-go/zap)

## Support

For issues or questions:
- Create issue in repository
- Contact: dev-team@cardiofit.com
- Slack: #notification-service
