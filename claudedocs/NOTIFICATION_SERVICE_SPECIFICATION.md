# Notification Service Specification

**Service Name**: `notification-service`
**Technology**: **Go** (Recommended)
**Module**: Module 6 - Component 6C: Multi-Channel Notification System
**Date**: November 10, 2025
**Status**: Design Specification

---

## Executive Summary

The Notification Service is a high-performance, multi-channel alerting platform that delivers clinical notifications through SMS, Email, Push, Pager, and Voice channels with intelligent routing, alert fatigue mitigation, and comprehensive delivery tracking.

### Key Capabilities
- ✅ Smart alert routing based on severity, role, and preferences
- ✅ Multi-channel delivery (SMS, Email, Push, Pager, Voice)
- ✅ Alert fatigue mitigation and bundling
- ✅ Escalation workflows with acknowledgment tracking
- ✅ Real-time delivery status and retry management
- ✅ Integration with Kafka event streams

---

## Technology Decision: Go vs Rust

### ✅ **Recommendation: Go**

#### Rationale:
1. **API Integration Heavy**: Service primarily integrates with external HTTP APIs (Twilio, SendGrid, FCM) - Go excels at HTTP client implementations
2. **Business Logic Focus**: Alert routing, fatigue tracking, escalation workflows are business logic heavy, not CPU-bound
3. **Concurrency Model**: Go's goroutines and channels are ideal for managing multiple concurrent notification deliveries
4. **Existing Patterns**: Safety Gateway and Flow2 Go Engine provide proven Go patterns in the CardioFit architecture
5. **Development Velocity**: Faster to implement complex business logic in Go compared to Rust
6. **gRPC + HTTP Support**: Go has excellent support for both gRPC (internal) and HTTP (external APIs)

#### When Rust Would Be Better:
- High-throughput stream processing (Rust excels)
- Memory-constrained environments (Rust's zero-cost abstractions)
- Ultra-low latency requirements (< 10ms P99)

#### For This Use Case:
- **Workload**: IO-bound (HTTP calls, Kafka consumers)
- **Latency Target**: Sub-100ms (Go easily achieves this)
- **Complexity**: High business logic, moderate performance requirements

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    NOTIFICATION SERVICE (Go)                     │
│                                                                   │
│  ┌────────────────┐   ┌──────────────────┐   ┌──────────────┐  │
│  │  Kafka         │──▶│  Alert Router    │──▶│  Delivery    │  │
│  │  Consumers     │   │  & Fatigue       │   │  Channels    │  │
│  │                │   │  Tracker         │   │              │  │
│  └────────────────┘   └──────────────────┘   └──────────────┘  │
│         │                      │                      │          │
│         ▼                      ▼                      ▼          │
│  ┌────────────────┐   ┌──────────────────┐   ┌──────────────┐  │
│  │  Alert         │   │  User            │   │  Delivery    │  │
│  │  Topics        │   │  Preferences     │   │  Status      │  │
│  │  (3 topics)    │   │  Service         │   │  Tracker     │  │
│  └────────────────┘   └──────────────────┘   └──────────────┘  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴──────────┐
                    ▼                    ▼
        ┌──────────────────┐   ┌──────────────────┐
        │  External APIs    │   │  PostgreSQL DB   │
        │  - Twilio (SMS)   │   │  - Notifications │
        │  - SendGrid       │   │  - User Prefs    │
        │  - FCM (Push)     │   │  - Delivery Log  │
        │  - PagerDuty      │   │                  │
        └──────────────────┘   └──────────────────┘
```

---

## Service Specifications

### 1. Service Configuration

**Service Name**: `notification-service`
**Port**: 8060 (HTTP), 50060 (gRPC)
**Language**: Go 1.21+
**Dependencies**:
- Kafka (Confluent Cloud)
- PostgreSQL (port 5432)
- Redis (port 6379) - for caching and rate limiting
- Twilio API
- SendGrid API
- Firebase Cloud Messaging
- PagerDuty (optional)

**Resource Requirements**:
- Memory: 512MB - 1GB
- CPU: 2-4 cores
- Disk: 10GB (logs and local storage)

---

## 2. Kafka Topics (Consumers)

The notification service consumes from **3 Kafka topics**:

### Input Topics:

1. **`ml-risk-alerts.v1`** (from Module 5 - ML Inference)
   - Sepsis alerts
   - Deterioration warnings
   - Mortality risk notifications
   - Readmission risk alerts

2. **`clinical-patterns.v1`** (from Module 4 - CEP)
   - Complex event pattern alerts
   - Vital sign anomalies
   - Clinical threshold violations
   - Patient deterioration patterns

3. **`alert-management.v1`** (from Module 4 - Alert Management)
   - Critical alert routing
   - Alert priority changes
   - Manual alert triggers
   - Escalation events

### Alert Event Schema:

```json
{
  "alert_id": "string",
  "patient_id": "string",
  "hospital_id": "string",
  "department_id": "string",
  "alert_type": "SEPSIS_ALERT | MORTALITY_RISK | VITAL_SIGN_ANOMALY | ...",
  "severity": "CRITICAL | HIGH | MODERATE | LOW",
  "confidence": 0.95,
  "message": "Patient PAT-001 sepsis risk elevated to 92%",
  "recommendations": ["Immediate physician review", "Blood culture", "Antibiotics within 1h"],
  "patient_location": {
    "room": "ICU-5",
    "bed": "A"
  },
  "vital_signs": {
    "heart_rate": 125,
    "blood_pressure_systolic": 85,
    "temperature": 39.2
  },
  "timestamp": 1699564800000,
  "metadata": {
    "source_module": "MODULE5_ML_INFERENCE",
    "model_version": "1.2.3",
    "requires_escalation": true
  }
}
```

---

## 3. Core Components

### Component 3A: Kafka Consumer Service

**Responsibilities**:
- Consume alerts from 3 Kafka topics
- Deserialize and validate alert events
- Route to Alert Router for processing
- Handle consumer group management and offset commits

**Implementation**:
```go
// internal/kafka/consumer.go

type AlertConsumer struct {
    consumer     *kafka.Consumer
    router       *AlertRouter
    config       *ConsumerConfig
    logger       *zap.Logger
}

type ConsumerConfig struct {
    Brokers       []string
    GroupID       string
    Topics        []string  // ["ml-risk-alerts.v1", "clinical-patterns.v1", "alert-management.v1"]
    AutoCommit    bool
    MaxRetries    int
}

func (c *AlertConsumer) Start(ctx context.Context) error {
    // Subscribe to all alert topics
    // Process messages in parallel with worker pool
    // Forward to AlertRouter
    // Commit offsets after successful routing
}

func (c *AlertConsumer) processMessage(msg *kafka.Message) error {
    var alert Alert
    if err := json.Unmarshal(msg.Value, &alert); err != nil {
        return fmt.Errorf("failed to unmarshal alert: %w", err)
    }

    return c.router.RouteAlert(context.Background(), &alert)
}
```

---

### Component 3B: Alert Router

**Responsibilities**:
- Determine target users based on alert severity and department
- Check alert fatigue rules
- Apply user preferences
- Route to appropriate notification channels
- Schedule escalations if required

**Alert Routing Logic**:

| Severity | Target Users | Channels | Escalation |
|----------|-------------|----------|------------|
| CRITICAL | Attending + Charge Nurse | Pager, SMS, Voice | 5 min |
| HIGH | Primary Nurse + Resident | SMS, Push | 15 min |
| MODERATE | Primary Nurse | Push, In-App | 30 min |
| LOW | Primary Nurse | In-App | None |
| ML_ALERT | Clinical Informatics Team | Email, Push | None |

**Implementation**:
```go
// internal/routing/alert_router.go

type AlertRouter struct {
    fatigueTracker   *AlertFatigueTracker
    userService      *UserPreferenceService
    deliveryService  *NotificationDeliveryService
    escalationMgr    *EscalationManager
    logger           *zap.Logger
}

func (r *AlertRouter) RouteAlert(ctx context.Context, alert *Alert) error {
    // 1. Determine target users
    users := r.determineTargetUsers(alert)

    // 2. Check alert fatigue for each user
    for _, user := range users {
        if r.fatigueTracker.ShouldSuppress(alert, user) {
            r.logger.Info("Alert suppressed due to fatigue",
                zap.String("user_id", user.ID),
                zap.String("reason", "rate_limit_exceeded"))
            continue
        }

        // 3. Get user channel preferences
        channels := r.userService.GetPreferredChannels(user, alert.Severity)

        // 4. Format and send notifications
        for _, channel := range channels {
            notification := r.buildNotification(alert, user, channel)

            go func(notif *Notification) {
                if err := r.deliveryService.Send(ctx, notif); err != nil {
                    r.logger.Error("Failed to send notification",
                        zap.Error(err),
                        zap.String("notification_id", notif.ID))
                }
            }(notification)
        }

        // 5. Record notification sent for fatigue tracking
        r.fatigueTracker.RecordNotification(user.ID, alert)
    }

    // 6. Schedule escalation if critical
    if alert.Severity == "CRITICAL" && alert.Metadata.RequiresEscalation {
        r.escalationMgr.ScheduleEscalation(ctx, alert, 5*time.Minute)
    }

    return nil
}

func (r *AlertRouter) determineTargetUsers(alert *Alert) []*User {
    var users []*User

    switch alert.Severity {
    case "CRITICAL":
        users = append(users, r.userService.GetAttendingPhysician(alert.DepartmentID)...)
        users = append(users, r.userService.GetChargeNurse(alert.DepartmentID)...)
    case "HIGH":
        users = append(users, r.userService.GetPrimaryNurse(alert.PatientID)...)
        users = append(users, r.userService.GetResident(alert.DepartmentID)...)
    case "MODERATE":
        users = append(users, r.userService.GetPrimaryNurse(alert.PatientID)...)
    default:
        users = append(users, r.userService.GetPrimaryNurse(alert.PatientID)...)
    }

    // Special routing for ML alerts
    if alert.Metadata.SourceModule == "MODULE5_ML_INFERENCE" {
        users = append(users, r.userService.GetClinicalInformaticsTeam()...)
    }

    return users
}
```

---

### Component 3C: Alert Fatigue Tracker

**Responsibilities**:
- Track notification volume per user
- Implement rate limiting (max 20 alerts/hour)
- Detect and suppress duplicate alerts
- Bundle similar non-critical alerts
- Provide fatigue metrics

**Fatigue Mitigation Rules**:
1. **Rate Limiting**: Max 20 alerts per user per hour
2. **Duplicate Suppression**: Same alert type + patient + severity within 5 minutes
3. **Bundling**: 3+ similar MODERATE alerts → 1 bundled notification
4. **Critical Exception**: CRITICAL alerts always bypass rate limits
5. **Quiet Hours**: Respect user quiet hours for non-critical alerts

**Implementation**:
```go
// internal/fatigue/fatigue_tracker.go

type AlertFatigueTracker struct {
    userHistories sync.Map  // map[string]*UserAlertHistory
    redis         *redis.Client
    config        *FatigueConfig
}

type FatigueConfig struct {
    MaxAlertsPerHour     int           // 20
    DuplicateWindowMs    int64         // 5 minutes
    BundleThreshold      int           // 3 alerts
    QuietHoursEnabled    bool
}

type UserAlertHistory struct {
    UserID       string
    RecentAlerts []AlertRecord
    BundleQueue  []AlertRecord
    LastCleanup  time.Time
}

func (t *AlertFatigueTracker) ShouldSuppress(alert *Alert, user *User) bool {
    history := t.getUserHistory(user.ID)

    // 1. Check rate limit
    if t.isRateLimited(history, alert) {
        // Allow CRITICAL through, suppress others
        if alert.Severity != "CRITICAL" {
            return true
        }
    }

    // 2. Check for duplicates
    if t.isDuplicate(history, alert) {
        return true
    }

    // 3. Check if should bundle
    if t.shouldBundle(history, alert) {
        history.BundleQueue = append(history.BundleQueue, AlertRecord{
            AlertID:   alert.AlertID,
            Type:      alert.AlertType,
            Severity:  alert.Severity,
            Timestamp: time.Now(),
        })

        // Trigger bundle send if threshold reached
        if len(history.BundleQueue) >= t.config.BundleThreshold {
            t.sendBundledAlert(history.BundleQueue, user)
            history.BundleQueue = []AlertRecord{}
        }

        return true  // Suppress individual alert
    }

    // 4. Check quiet hours
    if t.config.QuietHoursEnabled && t.isQuietHours(user) {
        if alert.Severity != "CRITICAL" && alert.Severity != "HIGH" {
            return true
        }
    }

    return false
}

func (t *AlertFatigueTracker) isRateLimited(history *UserAlertHistory, alert *Alert) bool {
    oneHourAgo := time.Now().Add(-1 * time.Hour)
    count := 0

    for _, record := range history.RecentAlerts {
        if record.Timestamp.After(oneHourAgo) {
            count++
        }
    }

    return count >= t.config.MaxAlertsPerHour
}

func (t *AlertFatigueTracker) isDuplicate(history *UserAlertHistory, alert *Alert) bool {
    duplicateWindow := time.Duration(t.config.DuplicateWindowMs) * time.Millisecond
    cutoff := time.Now().Add(-duplicateWindow)

    for _, record := range history.RecentAlerts {
        if record.Timestamp.Before(cutoff) {
            continue
        }

        if record.PatientID == alert.PatientID &&
           record.Type == alert.AlertType &&
           record.Severity == alert.Severity {
            return true
        }
    }

    return false
}

func (t *AlertFatigueTracker) RecordNotification(userID string, alert *Alert) {
    history := t.getUserHistory(userID)

    history.RecentAlerts = append(history.RecentAlerts, AlertRecord{
        AlertID:   alert.AlertID,
        PatientID: alert.PatientID,
        Type:      alert.AlertType,
        Severity:  alert.Severity,
        Timestamp: time.Now(),
    })

    // Cleanup old records
    if time.Since(history.LastCleanup) > 1*time.Hour {
        t.cleanupHistory(history)
    }
}
```

---

### Component 3D: Notification Delivery Service

**Responsibilities**:
- Send notifications via multiple channels
- Format messages per channel requirements
- Handle API authentication and rate limits
- Track delivery status and errors
- Implement retry logic with exponential backoff

**Delivery Channels**:

1. **SMS (Twilio)**
   - Max 160 characters
   - Format: "CRITICAL: PAT-001 Sepsis Alert (92%) - ICU Bed 5"

2. **Email (SendGrid)**
   - Full details with recommendations
   - HTML formatting
   - Subject: severity-based

3. **Push (Firebase Cloud Messaging)**
   - Title + body + data payload
   - Deep link to patient dashboard

4. **Pager (Twilio/PagerDuty)**
   - Ultra-short alphanumeric
   - Format: "CRIT PAT-001 SEPSIS ICU-5"

5. **Voice (Twilio Voice)**
   - Text-to-speech for critical escalations
   - Callback on user press 1 to acknowledge

**Implementation**:
```go
// internal/delivery/delivery_service.go

type NotificationDeliveryService struct {
    twilioClient   *twilio.RestClient
    sendGridClient *sendgrid.Client
    fcmClient      *messaging.Client
    pagerDuty      *pagerduty.Client
    statusTracker  *DeliveryStatusTracker
    logger         *zap.Logger
}

func (s *NotificationDeliveryService) Send(ctx context.Context, notification *Notification) error {
    // Update status to SENDING
    s.statusTracker.UpdateStatus(notification.ID, "SENDING")

    var result *DeliveryResult
    var err error

    switch notification.Channel {
    case ChannelSMS:
        result, err = s.sendSMS(ctx, notification)
    case ChannelEmail:
        result, err = s.sendEmail(ctx, notification)
    case ChannelPush:
        result, err = s.sendPush(ctx, notification)
    case ChannelPager:
        result, err = s.sendPager(ctx, notification)
    case ChannelVoice:
        result, err = s.initiateVoiceCall(ctx, notification)
    default:
        return fmt.Errorf("unsupported channel: %s", notification.Channel)
    }

    if err != nil {
        // Update status to FAILED and schedule retry
        s.statusTracker.UpdateStatus(notification.ID, "FAILED")
        s.scheduleRetry(ctx, notification)
        return err
    }

    // Update status to SENT
    s.statusTracker.UpdateStatus(notification.ID, "SENT")
    s.statusTracker.RecordExternalID(notification.ID, result.ExternalID)

    return nil
}

func (s *NotificationDeliveryService) sendSMS(ctx context.Context, notification *Notification) (*DeliveryResult, error) {
    // Format SMS message (max 160 chars)
    message := s.formatSMSMessage(notification)

    params := &twilioApi.CreateMessageParams{}
    params.SetTo(notification.User.PhoneNumber)
    params.SetFrom(s.twilioConfig.FromNumber)
    params.SetBody(message)

    resp, err := s.twilioClient.Api.CreateMessage(params)
    if err != nil {
        return nil, fmt.Errorf("twilio error: %w", err)
    }

    return &DeliveryResult{
        Success:    true,
        ExternalID: *resp.Sid,
    }, nil
}

func (s *NotificationDeliveryService) sendEmail(ctx context.Context, notification *Notification) (*DeliveryResult, error) {
    // Format full email with recommendations
    subject := fmt.Sprintf("Clinical Alert: %s - %s", notification.Alert.Severity, notification.Alert.AlertType)
    body := s.formatEmailBody(notification)

    message := mail.NewSingleEmail(
        mail.NewEmail("CardioFit Alerts", s.sendGridConfig.FromEmail),
        subject,
        mail.NewEmail(notification.User.Name, notification.User.Email),
        body,
        body, // HTML body
    )

    response, err := s.sendGridClient.Send(message)
    if err != nil {
        return nil, fmt.Errorf("sendgrid error: %w", err)
    }

    if response.StatusCode >= 200 && response.StatusCode < 300 {
        return &DeliveryResult{
            Success:    true,
            ExternalID: response.Headers.Get("X-Message-Id"),
        }, nil
    }

    return nil, fmt.Errorf("sendgrid failed with status %d", response.StatusCode)
}

func (s *NotificationDeliveryService) sendPush(ctx context.Context, notification *Notification) (*DeliveryResult, error) {
    message := &messaging.Message{
        Token: notification.User.FCMToken,
        Notification: &messaging.Notification{
            Title: fmt.Sprintf("%s Alert", notification.Alert.Severity),
            Body:  s.formatPushMessage(notification),
        },
        Data: map[string]string{
            "alert_id":   notification.Alert.AlertID,
            "patient_id": notification.Alert.PatientID,
            "severity":   notification.Alert.Severity,
            "type":       notification.Alert.AlertType,
            "deep_link":  fmt.Sprintf("/patients/%s", notification.Alert.PatientID),
        },
        Android: &messaging.AndroidConfig{
            Priority: "high",
        },
        APNS: &messaging.APNSConfig{
            Payload: &messaging.APNSPayload{
                Aps: &messaging.Aps{
                    Sound: "critical_alert.wav",
                },
            },
        },
    }

    messageID, err := s.fcmClient.Send(ctx, message)
    if err != nil {
        return nil, fmt.Errorf("fcm error: %w", err)
    }

    return &DeliveryResult{
        Success:    true,
        ExternalID: messageID,
    }, nil
}

// Message formatters
func (s *NotificationDeliveryService) formatSMSMessage(notification *Notification) string {
    // "CRITICAL: PAT-001 Sepsis Alert (92%) - ICU Bed 5"
    return fmt.Sprintf(
        "%s: %s %s (%.0f%%) - %s",
        notification.Alert.Severity,
        notification.Alert.PatientID,
        notification.Alert.AlertType,
        notification.Alert.Confidence * 100,
        notification.Alert.PatientLocation.Room,
    )
}

func (s *NotificationDeliveryService) formatEmailBody(notification *Notification) string {
    var sb strings.Builder

    sb.WriteString(fmt.Sprintf("Alert: %s\n", notification.Alert.AlertType))
    sb.WriteString(fmt.Sprintf("Severity: %s\n", notification.Alert.Severity))
    sb.WriteString(fmt.Sprintf("Patient: %s\n", notification.Alert.PatientID))
    sb.WriteString(fmt.Sprintf("Location: %s - %s\n\n", notification.Alert.DepartmentID, notification.Alert.PatientLocation.Room))

    sb.WriteString("Details:\n")
    sb.WriteString(fmt.Sprintf("%s\n\n", notification.Alert.Message))

    if len(notification.Alert.Recommendations) > 0 {
        sb.WriteString("Recommended Actions:\n")
        for _, rec := range notification.Alert.Recommendations {
            sb.WriteString(fmt.Sprintf("- %s\n", rec))
        }
    }

    sb.WriteString(fmt.Sprintf("\nConfidence: %.1f%%\n", notification.Alert.Confidence * 100))
    sb.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Unix(notification.Alert.Timestamp/1000, 0).Format(time.RFC3339)))

    return sb.String()
}
```

---

### Component 3E: Escalation Manager

**Responsibilities**:
- Schedule escalation timers for critical alerts
- Track alert acknowledgment status
- Escalate to higher authority if not acknowledged
- Trigger voice calls for critical escalations
- Maintain escalation audit trail

**Escalation Rules**:
- CRITICAL alerts: Escalate after 5 minutes if not acknowledged
- HIGH alerts: Escalate after 15 minutes if not acknowledged
- Escalation chain: Primary → Charge Nurse → Attending → Voice Call

**Implementation**:
```go
// internal/escalation/escalation_manager.go

type EscalationManager struct {
    timers        sync.Map  // map[string]*time.Timer
    ackTracker    *AcknowledgmentTracker
    notifyService *NotificationDeliveryService
    userService   *UserPreferenceService
    logger        *zap.Logger
}

type EscalationChain struct {
    AlertID      string
    CurrentLevel int
    Levels       []EscalationLevel
    StartTime    time.Time
}

type EscalationLevel struct {
    Level      int
    Users      []*User
    Channels   []NotificationChannel
    Timeout    time.Duration
}

func (m *EscalationManager) ScheduleEscalation(ctx context.Context, alert *Alert, timeout time.Duration) {
    chain := m.buildEscalationChain(alert)

    timer := time.AfterFunc(timeout, func() {
        if !m.ackTracker.IsAcknowledged(alert.AlertID) {
            m.escalate(ctx, alert, chain)
        }
    })

    m.timers.Store(alert.AlertID, timer)
}

func (m *EscalationManager) CancelEscalation(alertID string) {
    if timer, ok := m.timers.Load(alertID); ok {
        timer.(*time.Timer).Stop()
        m.timers.Delete(alertID)
    }
}

func (m *EscalationManager) escalate(ctx context.Context, alert *Alert, chain *EscalationChain) {
    chain.CurrentLevel++

    if chain.CurrentLevel >= len(chain.Levels) {
        // Final escalation: Voice call
        m.triggerVoiceEscalation(ctx, alert)
        return
    }

    level := chain.Levels[chain.CurrentLevel]

    m.logger.Warn("Escalating alert",
        zap.String("alert_id", alert.AlertID),
        zap.Int("level", level.Level))

    // Send notifications to next level
    for _, user := range level.Users {
        for _, channel := range level.Channels {
            notification := &Notification{
                ID:       uuid.New().String(),
                AlertID:  alert.AlertID,
                UserID:   user.ID,
                User:     user,
                Channel:  channel,
                Priority: 1,  // Highest
                Message:  fmt.Sprintf("ESCALATION: %s", alert.Message),
                CreatedAt: time.Now(),
                Status:   "PENDING",
            }

            go m.notifyService.Send(ctx, notification)
        }
    }

    // Schedule next escalation
    timer := time.AfterFunc(level.Timeout, func() {
        if !m.ackTracker.IsAcknowledged(alert.AlertID) {
            m.escalate(ctx, alert, chain)
        }
    })

    m.timers.Store(alert.AlertID, timer)
}

func (m *EscalationManager) buildEscalationChain(alert *Alert) *EscalationChain {
    return &EscalationChain{
        AlertID:      alert.AlertID,
        CurrentLevel: 0,
        StartTime:    time.Now(),
        Levels: []EscalationLevel{
            {
                Level:    1,
                Users:    m.userService.GetPrimaryNurse(alert.PatientID),
                Channels: []NotificationChannel{ChannelSMS, ChannelPush},
                Timeout:  5 * time.Minute,
            },
            {
                Level:    2,
                Users:    m.userService.GetChargeNurse(alert.DepartmentID),
                Channels: []NotificationChannel{ChannelPager, ChannelSMS},
                Timeout:  5 * time.Minute,
            },
            {
                Level:    3,
                Users:    m.userService.GetAttendingPhysician(alert.DepartmentID),
                Channels: []NotificationChannel{ChannelPager, ChannelVoice},
                Timeout:  5 * time.Minute,
            },
        },
    }
}
```

---

## 4. Database Schema

### PostgreSQL Tables:

```sql
-- Notifications table
CREATE TABLE notifications (
    id                   UUID PRIMARY KEY,
    alert_id             VARCHAR(255) NOT NULL,
    user_id              VARCHAR(255) NOT NULL,
    channel              VARCHAR(50) NOT NULL,  -- SMS, EMAIL, PUSH, PAGER, VOICE
    priority             INTEGER NOT NULL,      -- 1 (highest) to 5 (lowest)
    message              TEXT NOT NULL,
    status               VARCHAR(50) NOT NULL,  -- PENDING, SENT, DELIVERED, FAILED, ACKNOWLEDGED
    retry_count          INTEGER DEFAULT 0,
    external_id          VARCHAR(255),          -- Twilio SID, SendGrid message ID, etc.
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    sent_at              TIMESTAMP,
    delivered_at         TIMESTAMP,
    acknowledged_at      TIMESTAMP,
    error_message        TEXT,
    metadata             JSONB
);

CREATE INDEX idx_notifications_alert_id ON notifications(alert_id);
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

-- User preferences table
CREATE TABLE user_preferences (
    user_id                 VARCHAR(255) PRIMARY KEY,
    channel_preferences     JSONB NOT NULL,  -- {"SMS": true, "EMAIL": false, ...}
    severity_channels       JSONB NOT NULL,  -- {"CRITICAL": ["PAGER", "SMS"], ...}
    quiet_hours_enabled     BOOLEAN DEFAULT FALSE,
    quiet_hours_start       INTEGER,         -- Hour (0-23)
    quiet_hours_end         INTEGER,
    max_alerts_per_hour     INTEGER DEFAULT 20,
    fcm_token               VARCHAR(512),
    phone_number            VARCHAR(20),
    email                   VARCHAR(255),
    pager_number            VARCHAR(50),
    updated_at              TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Escalation log table
CREATE TABLE escalation_log (
    id                   UUID PRIMARY KEY,
    alert_id             VARCHAR(255) NOT NULL,
    escalation_level     INTEGER NOT NULL,
    escalated_to_user    VARCHAR(255) NOT NULL,
    escalated_at         TIMESTAMP NOT NULL DEFAULT NOW(),
    acknowledged_at      TIMESTAMP,
    acknowledged_by      VARCHAR(255),
    outcome              VARCHAR(50),  -- ACKNOWLEDGED, ESCALATED_FURTHER, TIMEOUT
    metadata             JSONB
);

CREATE INDEX idx_escalation_alert_id ON escalation_log(alert_id);

-- Delivery metrics table (for analytics)
CREATE TABLE delivery_metrics (
    id                   UUID PRIMARY KEY,
    date                 DATE NOT NULL,
    channel              VARCHAR(50) NOT NULL,
    total_sent           INTEGER DEFAULT 0,
    total_delivered      INTEGER DEFAULT 0,
    total_failed         INTEGER DEFAULT 0,
    avg_delivery_time_ms INTEGER,
    p95_delivery_time_ms INTEGER,
    p99_delivery_time_ms INTEGER,
    error_count_by_type  JSONB
);

CREATE INDEX idx_delivery_metrics_date_channel ON delivery_metrics(date, channel);
```

---

## 5. API Endpoints

### gRPC Service (Port 50060)

```protobuf
// proto/notification_service.proto

service NotificationService {
  // Send a notification manually
  rpc SendNotification(SendNotificationRequest) returns (SendNotificationResponse);

  // Acknowledge an alert
  rpc AcknowledgeAlert(AcknowledgeAlertRequest) returns (AcknowledgeAlertResponse);

  // Get notification status
  rpc GetNotificationStatus(GetNotificationStatusRequest) returns (GetNotificationStatusResponse);

  // Get user preferences
  rpc GetUserPreferences(GetUserPreferencesRequest) returns (GetUserPreferencesResponse);

  // Update user preferences
  rpc UpdateUserPreferences(UpdateUserPreferencesRequest) returns (UpdateUserPreferencesResponse);

  // Get delivery metrics
  rpc GetDeliveryMetrics(GetDeliveryMetricsRequest) returns (GetDeliveryMetricsResponse);

  // Health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

### REST API (Port 8060)

```
POST   /api/v1/notifications/send              # Manual notification trigger
POST   /api/v1/notifications/acknowledge       # Acknowledge alert
GET    /api/v1/notifications/{id}/status       # Get notification status
GET    /api/v1/notifications/user/{userId}     # Get user's notifications

GET    /api/v1/preferences/{userId}            # Get user preferences
PUT    /api/v1/preferences/{userId}            # Update user preferences

GET    /api/v1/metrics/delivery                # Delivery metrics
GET    /api/v1/metrics/fatigue/{userId}        # User alert fatigue stats

GET    /health                                 # Health check
GET    /metrics                                # Prometheus metrics
```

---

## 6. Configuration

### Environment Variables:

```bash
# Service
SERVICE_NAME=notification-service
HTTP_PORT=8060
GRPC_PORT=50060
ENVIRONMENT=production
LOG_LEVEL=info

# Kafka
KAFKA_BROKERS=pkc-xxxxx.us-east-1.aws.confluent.cloud:9092
KAFKA_GROUP_ID=notification-service-consumers
KAFKA_TOPICS=ml-risk-alerts.v1,clinical-patterns.v1,alert-management.v1
KAFKA_USERNAME=xxx
KAFKA_PASSWORD=xxx

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=cardiofit_notifications
POSTGRES_USER=cardiofit
POSTGRES_PASSWORD=xxx

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=xxx

# Twilio
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_FROM_NUMBER=+1234567890
TWILIO_PAGER_GATEWAY=pager.provider.com

# SendGrid
SENDGRID_API_KEY=SG.xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
SENDGRID_FROM_EMAIL=alerts@cardiofit.com

# Firebase
FIREBASE_PROJECT_ID=cardiofit-prod
FIREBASE_CREDENTIALS_PATH=/etc/secrets/firebase-credentials.json

# PagerDuty (Optional)
PAGERDUTY_API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
PAGERDUTY_INTEGRATION_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Alert Fatigue Config
ALERT_FATIGUE_MAX_PER_HOUR=20
ALERT_FATIGUE_DUPLICATE_WINDOW_MS=300000
ALERT_FATIGUE_BUNDLE_THRESHOLD=3
```

### config.yaml:

```yaml
service:
  name: notification-service
  http_port: 8060
  grpc_port: 50060
  environment: production
  version: 1.0.0

kafka:
  brokers:
    - pkc-xxxxx.us-east-1.aws.confluent.cloud:9092
  group_id: notification-service-consumers
  topics:
    - ml-risk-alerts.v1
    - clinical-patterns.v1
    - alert-management.v1
  consumer_config:
    session_timeout_ms: 30000
    heartbeat_interval_ms: 10000
    max_poll_records: 100
    auto_commit: true

database:
  host: localhost
  port: 5432
  database: cardiofit_notifications
  user: cardiofit
  max_connections: 50
  connection_timeout: 30s

redis:
  host: localhost
  port: 6379
  pool_size: 20
  connection_timeout: 5s

delivery:
  workers: 10
  retry_max_attempts: 3
  retry_initial_delay: 1s
  retry_max_delay: 30s
  retry_multiplier: 2

alert_fatigue:
  max_alerts_per_hour: 20
  duplicate_window_ms: 300000
  bundle_threshold: 3
  quiet_hours_enabled: true

escalation:
  critical_timeout_minutes: 5
  high_timeout_minutes: 15
  moderate_timeout_minutes: 30

observability:
  metrics_port: 8061
  tracing_enabled: true
  log_level: info
```

---

## 7. Monitoring & Observability

### Prometheus Metrics:

```go
// Notification metrics
notification_total{channel="sms|email|push|pager|voice", status="sent|delivered|failed"}
notification_duration_seconds{channel="sms|email|push|pager|voice"}
notification_retry_total{channel="sms|email|push|pager|voice"}

// Alert routing metrics
alert_routed_total{severity="critical|high|moderate|low"}
alert_suppressed_total{reason="rate_limit|duplicate|bundled|quiet_hours"}
alert_bundled_total

// Escalation metrics
escalation_triggered_total{level="1|2|3"}
escalation_acknowledged_total{level="1|2|3"}
escalation_timeout_total{level="1|2|3"}

// Channel-specific metrics
twilio_api_calls_total{status="success|failure"}
sendgrid_api_calls_total{status="success|failure"}
fcm_api_calls_total{status="success|failure"}

// Fatigue metrics
fatigue_suppression_total{reason="rate_limit|duplicate|bundled"}
fatigue_user_alert_count{user_id="xxx", period="1h"}
```

### Health Checks:

```go
// Health endpoints
GET /health        → Overall service health
GET /health/live   → Liveness probe (service running)
GET /health/ready  → Readiness probe (can handle traffic)

// Health check response
{
  "status": "healthy",
  "timestamp": "2025-11-10T12:00:00Z",
  "dependencies": {
    "kafka": "healthy",
    "postgres": "healthy",
    "redis": "healthy",
    "twilio": "healthy",
    "sendgrid": "healthy",
    "fcm": "healthy"
  },
  "metrics": {
    "kafka_lag": 15,
    "active_notifications": 42,
    "pending_escalations": 3
  }
}
```

### Logging:

```json
{
  "level": "info",
  "timestamp": "2025-11-10T12:00:00Z",
  "service": "notification-service",
  "request_id": "req_abc123",
  "alert_id": "alert_xyz789",
  "user_id": "user_001",
  "channel": "SMS",
  "status": "sent",
  "external_id": "SM123456789",
  "duration_ms": 245,
  "message": "Notification sent successfully"
}
```

---

## 8. Deployment

### Docker Build:

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o notification-service cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/notification-service .
COPY --from=builder /app/configs ./configs

EXPOSE 8060 50060

CMD ["./notification-service"]
```

### Kubernetes Deployment:

```yaml
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
        image: cardiofit/notification-service:1.0.0
        ports:
        - containerPort: 8060
          name: http
        - containerPort: 50060
          name: grpc
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: KAFKA_BROKERS
          valueFrom:
            secretKeyRef:
              name: kafka-config
              key: brokers
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: password
        - name: TWILIO_AUTH_TOKEN
          valueFrom:
            secretKeyRef:
              name: twilio-credentials
              key: auth_token
        - name: SENDGRID_API_KEY
          valueFrom:
            secretKeyRef:
              name: sendgrid-credentials
              key: api_key
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8060
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8060
          initialDelaySeconds: 10
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: notification-service
  namespace: cardiofit
spec:
  type: ClusterIP
  selector:
    app: notification-service
  ports:
  - name: http
    port: 8060
    targetPort: 8060
  - name: grpc
    port: 50060
    targetPort: 50060
```

---

## 9. Testing Strategy

### Unit Tests:
- Alert routing logic
- Fatigue detection algorithms
- Message formatting
- Channel-specific delivery

### Integration Tests:
- Kafka consumer processing
- Database operations
- Redis caching
- External API integration (with mocks)

### End-to-End Tests:
- Complete alert flow: Kafka → Router → Delivery → Acknowledgment
- Escalation workflows
- Multi-channel delivery
- Alert fatigue scenarios

### Load Tests:
- 1000 alerts/second throughput
- Concurrent user notification load
- Kafka consumer lag under load

---

## 10. Security Considerations

### API Security:
- JWT authentication for REST endpoints
- mTLS for gRPC communication
- API rate limiting per user

### Data Security:
- Encrypt PII in database (phone numbers, emails)
- Secure storage of API credentials (Kubernetes secrets)
- Audit logging for all notifications

### Network Security:
- Firewall rules for external API access
- VPC peering for database access
- TLS for all external communication

---

## 11. Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Alert Processing Latency | < 100ms P99 | From Kafka to routing decision |
| Notification Delivery | < 5s P99 | Time to external API |
| SMS Delivery | < 10s | Twilio delivery time |
| Email Delivery | < 30s | SendGrid delivery time |
| Push Delivery | < 5s | FCM delivery time |
| Kafka Consumer Lag | < 100 messages | Under normal load |
| Database Query | < 50ms P95 | PostgreSQL query time |
| Throughput | 1000 alerts/sec | Sustained throughput |

---

## 12. Implementation Timeline

### Phase 1: Core Infrastructure (Week 1)
- ✅ Project setup and Go module initialization
- ✅ Kafka consumer service
- ✅ PostgreSQL schema and migrations
- ✅ Basic alert routing logic
- ✅ Unit tests for core components

### Phase 2: Delivery Channels (Week 2)
- ✅ Twilio SMS integration
- ✅ SendGrid email integration
- ✅ FCM push notification integration
- ✅ PagerDuty integration
- ✅ Delivery status tracking

### Phase 3: Intelligence Layer (Week 3)
- ✅ Alert fatigue tracker
- ✅ Duplicate detection
- ✅ Alert bundling
- ✅ User preference service
- ✅ Integration tests

### Phase 4: Escalation & Monitoring (Week 4)
- ✅ Escalation manager
- ✅ Acknowledgment tracking
- ✅ Voice call escalation
- ✅ Prometheus metrics
- ✅ Health checks and logging

### Phase 5: Testing & Deployment (Week 5)
- ✅ End-to-end testing
- ✅ Load testing
- ✅ Security audit
- ✅ Docker and Kubernetes setup
- ✅ Production deployment

---

## 13. Integration Points

### Upstream Dependencies:
- **Flink Module 4**: Consumes from `clinical-patterns.v1`
- **Flink Module 5**: Consumes from `ml-risk-alerts.v1`, `inference-results.v1`
- **Alert Management Service**: Consumes from `alert-management.v1`

### Downstream Dependencies:
- **Twilio**: SMS and Voice API
- **SendGrid**: Email API
- **Firebase**: Push notification API
- **PagerDuty**: Pager integration API

### Internal Dependencies:
- **Auth Service** (port 8001): JWT validation
- **Patient Service** (port 8003): Patient context
- **Observation Service** (port 8010): Latest vitals
- **PostgreSQL**: Notification and user preference storage
- **Redis**: Caching and rate limiting

---

## 14. Open Questions & Decisions

### Questions for Product/Clinical Team:
1. Should we implement in-app notifications or focus on external channels first?
2. What's the desired escalation chain for different departments (ICU vs Emergency)?
3. Should we integrate with hospital-specific pager systems beyond PagerDuty?
4. Do we need multi-language support for notifications?
5. Should we implement notification templates or use dynamic messages?

### Technical Decisions:
1. **State Management**: In-memory vs Redis for alert fatigue tracking?
   - **Recommendation**: Redis for persistence and scalability
2. **Message Queue**: Direct Kafka consumer vs internal queue?
   - **Recommendation**: Direct Kafka with worker pool
3. **Retry Strategy**: Exponential backoff vs linear?
   - **Recommendation**: Exponential with max 3 retries
4. **Database**: PostgreSQL vs TimescaleDB for time-series data?
   - **Recommendation**: PostgreSQL with partitioning, migrate to TimescaleDB if needed

---

## 15. Success Criteria

### Functional Success:
- ✅ All 3 Kafka topics consumed successfully
- ✅ Notifications delivered via all 5 channels
- ✅ Alert fatigue correctly suppresses duplicate/bundled alerts
- ✅ Escalation workflows trigger appropriately
- ✅ User preferences respected for all notifications

### Performance Success:
- ✅ P99 latency < 100ms for alert processing
- ✅ Kafka consumer lag < 100 messages under load
- ✅ 99.9% delivery success rate for critical alerts
- ✅ Zero data loss during service restarts

### Operational Success:
- ✅ Service uptime > 99.9%
- ✅ Mean time to recovery (MTTR) < 5 minutes
- ✅ Comprehensive monitoring and alerting
- ✅ Clear runbooks for common issues

---

## 16. Future Enhancements

### Phase 2 Features:
- Machine learning-based alert prioritization
- Intelligent quiet hours based on user behavior
- Natural language generation for context-aware messages
- Integration with hospital communication systems (Vocera, Spok)
- Two-way SMS for acknowledgment
- Mobile app integration for richer notifications

### Advanced Features:
- Predictive alert delivery (pre-notify based on ML predictions)
- Multi-tenant support for multiple hospitals
- Advanced analytics dashboard for notification patterns
- A/B testing for notification effectiveness
- Integration with EHR systems for context-aware routing

---

## Appendix A: Kafka Topic Schemas

### ml-risk-alerts.v1

```json
{
  "alert_id": "string",
  "patient_id": "string",
  "alert_type": "SEPSIS_ALERT | MORTALITY_RISK | DETERIORATION | READMISSION_RISK",
  "severity": "CRITICAL | HIGH | MODERATE",
  "risk_score": 0.95,
  "confidence": 0.88,
  "message": "string",
  "recommendations": ["string"],
  "timestamp": 1699564800000,
  "metadata": {
    "model_version": "1.2.3",
    "feature_importance": {},
    "requires_escalation": true
  }
}
```

### clinical-patterns.v1

```json
{
  "pattern_id": "string",
  "patient_id": "string",
  "pattern_type": "VITAL_SIGN_ANOMALY | TREND_DETERIORATION | THRESHOLD_VIOLATION",
  "severity": "CRITICAL | HIGH | MODERATE",
  "affected_parameters": ["heart_rate", "blood_pressure"],
  "message": "string",
  "timestamp": 1699564800000,
  "metadata": {
    "window_size_minutes": 15,
    "detection_algorithm": "CEP"
  }
}
```

---

## Appendix B: Example Workflows

### Workflow 1: Critical Sepsis Alert

```
1. Flink Module 5 detects sepsis risk (95% confidence)
2. Publishes to ml-risk-alerts.v1
3. Notification Service consumes alert
4. Router determines targets: Attending + Charge Nurse
5. Fatigue Tracker checks: No suppression (CRITICAL always passes)
6. Delivery Service sends:
   - Pager to Attending
   - SMS to Attending
   - SMS to Charge Nurse
7. Escalation Manager schedules 5-minute timer
8. At T+5 min: Alert not acknowledged
9. Escalation to next level: Senior physician + Voice call
10. At T+2 min: Charge Nurse acknowledges via mobile app
11. Escalation timer cancelled
12. Audit log updated with acknowledgment
```

### Workflow 2: Alert Fatigue Bundling

```
1. 5 MODERATE lab result alerts arrive within 3 minutes for same patient
2. Alert Router processes first alert → Delivered
3. Alert Router processes second alert → Duplicate suppressed
4. Third alert → Added to bundle queue
5. Fourth alert → Added to bundle queue (threshold = 3)
6. Fifth alert → Triggers bundled notification
7. Single notification sent: "BUNDLED: 4 similar LAB_RESULT alerts for PAT-001"
8. Bundle queue cleared
```

---

**End of Specification**

---

**Next Steps**:
1. Review and approve specification
2. Set up Go project structure
3. Implement Phase 1: Core Infrastructure
4. Integrate with existing Kafka topics
5. Deploy to development environment for testing
