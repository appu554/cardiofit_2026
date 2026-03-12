# Alert Router Implementation

## Overview

The Alert Router is a core component of the Notification Service that implements intelligent alert routing logic with severity-based user targeting. It routes clinical alerts from Kafka topics to appropriate healthcare staff through multiple channels (SMS, Email, Push, Pager, Voice, In-App) based on alert severity and user preferences.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Alert Router                              │
│                                                                   │
│  RouteAlert(alert) ──┐                                           │
│                      │                                           │
│                      ├──▶ 1. determineTargetUsers()              │
│                      │    └──▶ GetAttending/Nurse/Resident/Team  │
│                      │                                           │
│                      ├──▶ 2. Check AlertFatigueTracker          │
│                      │    └──▶ ShouldSuppress(alert, user)?     │
│                      │                                           │
│                      ├──▶ 3. GetPreferredChannels()              │
│                      │    └──▶ User preferences or defaults     │
│                      │                                           │
│                      ├──▶ 4. buildNotification()                │
│                      │    └──▶ Format for each channel          │
│                      │                                           │
│                      ├──▶ 5. NotificationDeliveryService.Send()  │
│                      │    └──▶ Async delivery via channels      │
│                      │                                           │
│                      ├──▶ 6. RecordNotification()               │
│                      │    └──▶ Update fatigue tracker           │
│                      │                                           │
│                      └──▶ 7. ScheduleEscalation()                │
│                           └──▶ If CRITICAL/HIGH severity         │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Severity-Based Routing Rules

| Severity | Target Users | Channels | Escalation | Timeout |
|----------|-------------|----------|------------|---------|
| **CRITICAL** | Attending Physician + Charge Nurse | Pager, SMS, Voice | Yes | 5 min |
| **HIGH** | Primary Nurse + Resident | SMS, Push | Yes | 15 min |
| **MODERATE** | Primary Nurse | Push, In-App | Yes | 30 min |
| **LOW** | Primary Nurse | In-App | No | N/A |
| **ML_ALERT** | Clinical Informatics Team | Email, Push | No | N/A |

### Special Routing Rules

1. **ML-Sourced Alerts**: Any alert with `metadata.source_module = "MODULE5_ML_INFERENCE"` is automatically routed to the Clinical Informatics Team in addition to the severity-based routing.

2. **Escalation Override**: If `metadata.requires_escalation = true`, escalation is scheduled regardless of severity level.

3. **User Preference Override**: User-defined channel preferences override the default severity channels.

## Components

### 1. AlertRouter

Main routing orchestrator that coordinates all routing operations.

**Key Methods:**
- `RouteAlert(ctx, alert)` - Main entry point for alert routing
- `GetRoutingDecision(ctx, alert)` - Preview routing without sending notifications
- `determineTargetUsers(alert)` - Select target users based on severity and role
- `buildNotification(alert, user, channel)` - Create notification objects
- `formatMessageForChannel(alert, channel)` - Format messages per channel constraints

### 2. Interfaces

**AlertFatigueTracker**
```go
type AlertFatigueTracker interface {
    ShouldSuppress(alert *models.Alert, user *models.User) (bool, string)
    RecordNotification(userID string, alert *models.Alert)
}
```

**UserPreferenceService**
```go
type UserPreferenceService interface {
    GetAttendingPhysician(departmentID string) ([]*models.User, error)
    GetChargeNurse(departmentID string) ([]*models.User, error)
    GetPrimaryNurse(patientID string) ([]*models.User, error)
    GetResident(departmentID string) ([]*models.User, error)
    GetClinicalInformaticsTeam() ([]*models.User, error)
    GetPreferredChannels(user *models.User, severity models.AlertSeverity) []models.NotificationChannel
}
```

**NotificationDeliveryService**
```go
type NotificationDeliveryService interface {
    Send(ctx context.Context, notification *models.Notification) error
}
```

**EscalationManager**
```go
type EscalationManager interface {
    ScheduleEscalation(ctx context.Context, alert *models.Alert, timeout time.Duration) error
}
```

## Models

### Alert Model

```go
type Alert struct {
    AlertID           string
    PatientID         string
    HospitalID        string
    DepartmentID      string
    AlertType         AlertType
    Severity          AlertSeverity
    Confidence        float64
    Message           string
    Recommendations   []string
    PatientLocation   PatientLocation
    VitalSigns        *VitalSigns
    Timestamp         int64
    Metadata          AlertMetadata
}
```

### User Model

```go
type User struct {
    ID           string
    Name         string
    Email        string
    PhoneNumber  string
    PagerNumber  string
    FCMToken     string
    Role         string
    DepartmentID string
    Preferences  *UserPreferences
}
```

### Notification Model

```go
type Notification struct {
    ID              string
    AlertID         string
    UserID          string
    User            *User
    Alert           *Alert
    Channel         NotificationChannel
    Priority        int  // 1 (highest) to 5 (lowest)
    Message         string
    Status          NotificationStatus
    RetryCount      int
    ExternalID      string
    CreatedAt       time.Time
    SentAt          *time.Time
    DeliveredAt     *time.Time
    AcknowledgedAt  *time.Time
    ErrorMessage    string
    Metadata        map[string]interface{}
}
```

## Message Formatting by Channel

### SMS (Twilio)
- **Constraint**: Max 160 characters
- **Format**: `CRITICAL: PAT-001 Sepsis Alert (92%) - ICU Bed 5`
- **Example**:
  ```
  CRITICAL: PAT-001 SEPSIS_ALERT (92%) - ICU-5
  ```

### Pager
- **Constraint**: Ultra-short alphanumeric
- **Format**: `CRIT PAT-001 SEPSIS ICU-5`
- **Example**:
  ```
  CRIT PAT-001 SEPSIS_ALERT ICU-5
  ```

### Push (FCM)
- **Format**: Title + body with confidence
- **Example**:
  ```
  Title: "CRITICAL Alert"
  Body: "CRITICAL Alert: SEPSIS_ALERT for patient PAT-001 in ICU-5. Confidence: 92%"
  Data: { alert_id, patient_id, severity, deep_link }
  ```

### Email (SendGrid)
- **Format**: Full details with recommendations
- **Example**:
  ```
  Subject: Clinical Alert: CRITICAL - SEPSIS_ALERT

  Alert: SEPSIS_ALERT
  Severity: CRITICAL
  Patient: PAT-001
  Location: DEPT-ICU - ICU-5

  Details:
  Patient PAT-001 sepsis risk elevated to 92%

  Recommended Actions:
  - Immediate physician review
  - Blood culture
  - Antibiotics within 1h

  Confidence: 92.0%
  Timestamp: 2025-11-10T12:00:00Z
  ```

### Voice (Twilio Voice)
- **Format**: Text-to-speech clear spoken message
- **Example**:
  ```
  "Critical alert. CRITICAL. Patient PAT-001 in ICU-5 has SEPSIS_ALERT with 92 percent confidence."
  ```

### In-App
- **Format**: Full alert message
- **Example**: Original alert message with all details

## Metrics

The router exposes the following Prometheus metrics:

```go
// Total alerts routed by severity and type
alerts_routed_total{severity="critical|high|moderate|low", alert_type="..."}

// Routing duration in seconds
routing_duration_seconds (histogram)

// Total users targeted for notifications
users_targeted_total

// Total alerts suppressed by reason
alerts_suppressed_total{reason="rate_limit|duplicate|bundled|quiet_hours"}

// Total escalations scheduled
escalations_scheduled_total
```

## Usage Examples

### Basic Routing

```go
// Create router with dependencies
router := NewAlertRouter(
    fatigueTracker,
    userService,
    deliveryService,
    escalationMgr,
    logger,
)

// Route a critical sepsis alert
alert := &models.Alert{
    AlertID:      "alert-123",
    PatientID:    "PAT-001",
    DepartmentID: "DEPT-ICU",
    AlertType:    models.AlertTypeSepsis,
    Severity:     models.SeverityCritical,
    Confidence:   0.92,
    Message:      "Patient PAT-001 sepsis risk elevated to 92%",
    // ... other fields
}

ctx := context.Background()
err := router.RouteAlert(ctx, alert)
if err != nil {
    log.Fatal(err)
}
```

### Preview Routing Decision

```go
// Get routing decision without sending notifications
decision, err := router.GetRoutingDecision(ctx, alert)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Target Users: %d\n", len(decision.TargetUsers))
fmt.Printf("Requires Escalation: %v\n", decision.RequiresEscalation)
fmt.Printf("Escalation Timeout: %v\n", decision.EscalationTimeout)

for userID, channels := range decision.UserChannels {
    fmt.Printf("User %s: %v\n", userID, channels)
}

for userID, reason := range decision.SuppressedUsers {
    fmt.Printf("User %s suppressed: %s\n", userID, reason)
}
```

## Testing

Comprehensive unit tests are provided in `alert_router_test.go`:

```bash
# Run all tests
go test ./internal/routing/...

# Run with coverage
go test -cover ./internal/routing/...

# Run specific test
go test -run TestAlertRouter_RouteAlert_Critical ./internal/routing/...

# Run with verbose output
go test -v ./internal/routing/...
```

### Test Coverage

- ✅ Critical alert routing (Attending + Charge Nurse)
- ✅ High alert routing (Primary Nurse + Resident)
- ✅ Moderate alert routing (Primary Nurse only)
- ✅ ML alert routing (Clinical Informatics Team)
- ✅ Alert fatigue suppression
- ✅ Channel preference handling
- ✅ Message formatting for all channels
- ✅ Priority assignment
- ✅ Routing decision preview
- ✅ User deduplication

## Integration Points

### Upstream (Input)
- **Kafka Consumer**: Consumes alerts from 3 topics:
  - `ml-risk-alerts.v1` (ML inference alerts)
  - `clinical-patterns.v1` (CEP pattern alerts)
  - `alert-management.v1` (manual alerts)

### Downstream (Output)
- **AlertFatigueTracker**: Rate limiting, duplicate detection, bundling
- **UserPreferenceService**: User lookup and preference management
- **NotificationDeliveryService**: Multi-channel delivery (SMS, Email, Push, Pager, Voice)
- **EscalationManager**: Escalation workflow scheduling

### External APIs
- **Twilio**: SMS and Voice delivery
- **SendGrid**: Email delivery
- **Firebase Cloud Messaging**: Push notifications
- **PagerDuty**: Pager integration

## Configuration

### Default Severity Channels

```go
var DefaultSeverityChannels = map[AlertSeverity][]NotificationChannel{
    SeverityCritical: {ChannelPager, ChannelSMS, ChannelVoice},
    SeverityHigh:     {ChannelSMS, ChannelPush},
    SeverityModerate: {ChannelPush, ChannelInApp},
    SeverityLow:      {ChannelInApp},
    SeverityMLAlert:  {ChannelEmail, ChannelPush},
}
```

### Default Escalation Timeouts

```go
var DefaultEscalationTimeouts = map[AlertSeverity]time.Duration{
    SeverityCritical: 5 * time.Minute,
    SeverityHigh:     15 * time.Minute,
    SeverityModerate: 30 * time.Minute,
    SeverityLow:      0,  // No escalation
    SeverityMLAlert:  0,  // No escalation
}
```

## Error Handling

The router implements robust error handling:

1. **User Lookup Failures**: Logged but don't fail entire routing
2. **Delivery Failures**: Handled by delivery service with retries
3. **Escalation Failures**: Logged but don't fail routing
4. **Invalid Alerts**: Rejected before routing

All errors are logged with structured logging (zap) including:
- Alert ID
- User ID
- Error message
- Context (severity, department, etc.)

## Performance Considerations

1. **Async Delivery**: Notifications are sent asynchronously via goroutines
2. **No Blocking**: User lookup and channel determination don't block delivery
3. **Metrics**: Low-overhead Prometheus metrics collection
4. **Logging**: Structured logging with appropriate levels

## Security Considerations

1. **PII Protection**: User contact info (phone, email) is masked in logs
2. **Alert Validation**: Alerts validated before routing
3. **Channel Security**: All external API calls use TLS
4. **Audit Trail**: All routing decisions logged for compliance

## Future Enhancements

1. **Machine Learning**: ML-based alert prioritization
2. **Smart Routing**: Context-aware routing based on staff workload
3. **A/B Testing**: Test different routing strategies
4. **Multi-Language**: Support for internationalized messages
5. **Template System**: Configurable message templates per alert type

## Dependencies

```go
require (
    github.com/google/uuid v1.3.0
    github.com/prometheus/client_golang v1.17.0
    go.uber.org/zap v1.26.0
    github.com/stretchr/testify v1.8.4
)
```

## File Structure

```
internal/routing/
├── README.md              # This file
├── alert_router.go        # Main router implementation
├── alert_router_test.go   # Comprehensive unit tests
└── interfaces.go          # (Optional) Interface definitions
```

## Contributing

When modifying the alert router:

1. **Add Tests**: All new functionality must have unit tests
2. **Update Metrics**: Add new metrics for significant features
3. **Document Routing**: Update routing rules in this README
4. **Log Appropriately**: Use structured logging with context
5. **Maintain Interfaces**: Keep interface contracts stable

## License

Copyright (c) 2025 CardioFit Clinical Synthesis Hub. All rights reserved.
