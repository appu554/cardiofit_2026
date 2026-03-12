# Notification Service Architecture

## Overview

The Notification Service is a critical component of the CardioFit platform, responsible for delivering clinical alerts through multiple channels with intelligent routing, fatigue management, and escalation policies.

## System Components

### 1. Kafka Consumer
- Consumes clinical alerts from `clinical-alerts` topic
- Manages consumer group offsets
- Provides graceful shutdown and error handling
- Supports parallel message processing

### 2. Routing Engine
- Priority-based channel selection
- User preference integration
- Fatigue check coordination
- Retry policy management

### 3. Delivery Manager
- Multi-provider orchestration
- Channel-specific delivery logic
- Error handling and retry logic
- Delivery result tracking

### 4. Fatigue Manager
- Redis-based rate limiting
- Quiet hours enforcement
- Per-recipient tracking
- Priority-based bypass rules

### 5. Escalation Engine
- Failed delivery detection
- Progressive channel escalation
- Configurable delays and limits
- Critical alert prioritization

## Data Flow

```
1. Clinical Alert Generated (Module 3/4)
   ↓
2. Kafka Topic: clinical-alerts
   ↓
3. Notification Service Consumer
   ↓
4. Routing Engine
   ├─→ Fatigue Check (Redis)
   └─→ Channel Selection
   ↓
5. Delivery Manager
   ├─→ Email (SendGrid)
   ├─→ SMS (Twilio)
   └─→ Push (Firebase)
   ↓
6. Result Logging (PostgreSQL)
   ↓
7. Escalation (if failed & critical)
```

## Database Schema

### PostgreSQL Tables

```sql
-- Notification delivery tracking
CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY,
    alert_id VARCHAR(255) NOT NULL,
    channel VARCHAR(50) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    message_id VARCHAR(255),
    error_message TEXT,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_alert_id (alert_id),
    INDEX idx_recipient (recipient),
    INDEX idx_created_at (created_at)
);

-- User notification preferences
CREATE TABLE notification_preferences (
    user_id VARCHAR(255) PRIMARY KEY,
    preferred_channels JSONB,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    enabled_alert_types JSONB,
    priority_threshold VARCHAR(50),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Escalation history
CREATE TABLE escalations (
    id UUID PRIMARY KEY,
    alert_id VARCHAR(255) NOT NULL,
    from_channel VARCHAR(50),
    to_channel VARCHAR(50),
    reason TEXT,
    escalated_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_alert_id (alert_id)
);
```

### Redis Keys

```
# Fatigue tracking
fatigue:{user_id} → counter (expires after window duration)

# Delivery status cache
delivery:{alert_id} → JSON status object (TTL: 1 hour)

# Preference cache
preferences:{user_id} → JSON preferences (TTL: 1 hour)
```

## External Integrations

### SendGrid (Email)
- API-based email delivery
- Template support
- Bounce and spam tracking
- Webhook integration for delivery status

### Twilio (SMS)
- REST API for SMS delivery
- Support for long messages
- Delivery receipts
- International number support

### Firebase (Push)
- FCM for push notifications
- Device token management
- Rich notification support
- Platform-specific customization

## Error Handling

### Retry Strategy
1. Immediate retry (for transient errors)
2. Exponential backoff (30s, 60s, 120s)
3. Dead letter queue after max retries
4. Escalation for critical alerts

### Error Categories
- **Transient**: Network issues, provider rate limits
- **Permanent**: Invalid recipient, disabled account
- **Configuration**: Missing credentials, invalid config
- **Provider**: External service outages

## Scalability

### Horizontal Scaling
- Stateless service design
- Kafka consumer groups for load distribution
- Redis for shared state
- Connection pooling for databases

### Performance Optimization
- Batch delivery where possible
- Connection reuse for HTTP clients
- Goroutine pools for concurrent delivery
- Circuit breakers for provider failures

## Security

### Data Protection
- TLS for all external communications
- Credential encryption at rest
- PHI handling compliance (HIPAA)
- Audit logging for all deliveries

### Access Control
- Service-to-service authentication
- API key management
- Role-based access control
- Network isolation

## Monitoring

### Key Metrics
- Delivery success rate by channel
- Average delivery latency
- Fatigue suppression rate
- Escalation frequency
- Provider error rates

### Alerts
- High failure rate (>5%)
- Increased latency (>5s)
- Provider unavailability
- Kafka consumer lag

## Future Enhancements

1. **Advanced Routing**
   - ML-based channel prediction
   - A/B testing for delivery strategies
   - Time-zone aware scheduling

2. **Enhanced Preferences**
   - Channel preferences by alert type
   - Custom quiet hours per day
   - Snooze functionality

3. **Additional Channels**
   - Voice calls (Twilio Voice)
   - In-app notifications
   - WebSocket real-time push

4. **Analytics**
   - Delivery analytics dashboard
   - Engagement metrics
   - User behavior analysis
