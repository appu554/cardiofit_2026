# Alert Router - Routing Logic Quick Reference

## Severity-Based Routing Matrix

### CRITICAL Severity

**Triggered by**: Sepsis alerts, severe deterioration, life-threatening conditions

**Target Users**:
- ✅ Attending Physician (department-wide)
- ✅ Charge Nurse (department-wide)

**Channels** (in order):
1. **Pager** - Immediate notification
2. **SMS** - Text to mobile
3. **Voice** - Phone call (if not acknowledged)

**Escalation**:
- ✅ Enabled
- ⏱️ Timeout: 5 minutes
- 🔄 Chain: Attending → Senior Physician → Voice Call

**Fatigue Override**:
- ❌ NEVER suppressed - always delivered

**Code**:
```go
case models.SeverityCritical:
    attending := userService.GetAttendingPhysician(alert.DepartmentID)
    chargeNurse := userService.GetChargeNurse(alert.DepartmentID)
    channels := {ChannelPager, ChannelSMS, ChannelVoice}
```

---

### HIGH Severity

**Triggered by**: Patient deterioration, significant vital sign changes

**Target Users**:
- ✅ Primary Nurse (patient-assigned)
- ✅ Resident (on-duty for department)

**Channels** (in order):
1. **SMS** - Text to mobile
2. **Push** - Mobile app notification

**Escalation**:
- ✅ Enabled
- ⏱️ Timeout: 15 minutes
- 🔄 Chain: Primary Nurse → Charge Nurse → Attending

**Fatigue Override**:
- ⚠️ Limited suppression - only duplicate detection active
- ✅ Rate limits bypassed

**Code**:
```go
case models.SeverityHigh:
    primaryNurse := userService.GetPrimaryNurse(alert.PatientID)
    resident := userService.GetResident(alert.DepartmentID)
    channels := {ChannelSMS, ChannelPush}
```

---

### MODERATE Severity

**Triggered by**: Trend changes, non-urgent anomalies, lab result alerts

**Target Users**:
- ✅ Primary Nurse (patient-assigned)

**Channels** (in order):
1. **Push** - Mobile app notification
2. **In-App** - Dashboard notification

**Escalation**:
- ✅ Enabled
- ⏱️ Timeout: 30 minutes
- 🔄 Chain: Primary Nurse → Charge Nurse (if no acknowledgment)

**Fatigue Handling**:
- ✅ Full fatigue mitigation active
- ✅ Rate limiting enforced (max 20/hour)
- ✅ Duplicate suppression enabled
- ✅ Bundling after 3 similar alerts

**Code**:
```go
case models.SeverityModerate:
    primaryNurse := userService.GetPrimaryNurse(alert.PatientID)
    channels := {ChannelPush, ChannelInApp}
```

---

### LOW Severity

**Triggered by**: Informational alerts, routine notifications

**Target Users**:
- ✅ Primary Nurse (patient-assigned)

**Channels**:
1. **In-App** - Dashboard notification only

**Escalation**:
- ❌ Disabled - No automatic escalation

**Fatigue Handling**:
- ✅ Full fatigue mitigation active
- ✅ Aggressive bundling (up to 10 alerts)
- ✅ Quiet hours respected

**Code**:
```go
case models.SeverityLow:
    primaryNurse := userService.GetPrimaryNurse(alert.PatientID)
    channels := {ChannelInApp}
```

---

### ML_ALERT Severity

**Triggered by**: Machine learning model predictions, risk scores

**Target Users**:
- ✅ Clinical Informatics Team
- ✅ Data Science Team
- ℹ️ (Not sent to clinical staff)

**Channels**:
1. **Email** - Detailed report with model metrics
2. **Push** - Mobile notification for team

**Escalation**:
- ❌ Disabled - ML alerts are for monitoring, not immediate action

**Fatigue Handling**:
- ⚠️ Custom rate limits (max 50/hour for ML team)
- ✅ Grouped by model type

**Special Behavior**:
- 📊 Includes model version, feature importance
- 📈 Aggregated in daily ML summary reports

**Code**:
```go
case models.SeverityMLAlert:
    informaticsTeam := userService.GetClinicalInformaticsTeam()
    channels := {ChannelEmail, ChannelPush}
```

---

## Special Routing Rules

### Rule 1: ML-Sourced Alert Override

**Condition**: `alert.Metadata.SourceModule == "MODULE5_ML_INFERENCE"`

**Action**:
- ✅ Always add Clinical Informatics Team to recipients (in addition to severity-based routing)
- 📧 Send detailed email with model metadata

**Example**:
```go
if alert.Metadata.SourceModule == "MODULE5_ML_INFERENCE" {
    informaticsTeam := userService.GetClinicalInformaticsTeam()
    users = mergeUsers(users, informaticsTeam)
}
```

---

### Rule 2: Escalation Override

**Condition**: `alert.Metadata.RequiresEscalation == true`

**Action**:
- ✅ Schedule escalation regardless of severity
- ⏱️ Use severity-default timeout or 15 minutes

**Example**:
```go
if alert.Metadata.RequiresEscalation || alert.Severity == SeverityCritical {
    escalationMgr.ScheduleEscalation(ctx, alert, timeout)
}
```

---

### Rule 3: User Preference Override

**Condition**: User has custom channel preferences set

**Action**:
- ✅ Use user's preferred channels instead of severity defaults
- ⚠️ Minimum: At least one channel must be enabled
- ❌ Exception: CRITICAL alerts always include Pager

**Example**:
```go
channels := userService.GetPreferredChannels(user, severity)
if len(channels) == 0 {
    channels = DefaultSeverityChannels[severity]
}
// CRITICAL: Always ensure Pager is included
if severity == SeverityCritical && !contains(channels, ChannelPager) {
    channels = append([]NotificationChannel{ChannelPager}, channels...)
}
```

---

## Message Formatting Rules

### SMS (Channel: ChannelSMS)

**Constraint**: Maximum 160 characters

**Format**: `{SEVERITY}: {PATIENT_ID} {ALERT_TYPE} ({CONFIDENCE}%) - {LOCATION}`

**Example**:
```
CRITICAL: PAT-001 SEPSIS_ALERT (92%) - ICU-5
```

**Implementation**:
```go
fmt.Sprintf("%s: %s %s (%.0f%%) - %s",
    alert.Severity,
    alert.PatientID,
    alert.AlertType,
    alert.Confidence * 100,
    alert.PatientLocation.Room)
```

---

### Pager (Channel: ChannelPager)

**Constraint**: Ultra-short alphanumeric (< 100 chars)

**Format**: `{ABBREV_SEVERITY} {PATIENT_ID} {SHORT_ALERT_TYPE} {LOCATION}`

**Example**:
```
CRIT PAT-001 SEPSIS ICU-5
```

**Implementation**:
```go
severityShort := string(alert.Severity)[:4]
alertTypeShort := truncate(string(alert.AlertType), 10)
fmt.Sprintf("%s %s %s %s",
    severityShort,
    alert.PatientID,
    alertTypeShort,
    alert.PatientLocation.Room)
```

---

### Push Notification (Channel: ChannelPush)

**Format**: Title + Body + Data Payload

**Title**: `{SEVERITY} Alert`

**Body**: `{SEVERITY} Alert: {ALERT_TYPE} for patient {PATIENT_ID} in {LOCATION}. Confidence: {CONFIDENCE}%`

**Data Payload**:
```json
{
    "alert_id": "alert-123",
    "patient_id": "PAT-001",
    "severity": "CRITICAL",
    "type": "SEPSIS_ALERT",
    "deep_link": "/patients/PAT-001"
}
```

**Example**:
```
Title: "CRITICAL Alert"
Body: "CRITICAL Alert: SEPSIS_ALERT for patient PAT-001 in ICU-5. Confidence: 92%"
```

---

### Email (Channel: ChannelEmail)

**Format**: Full details with HTML formatting

**Subject**: `Clinical Alert: {SEVERITY} - {ALERT_TYPE}`

**Body Template**:
```
Alert: {ALERT_TYPE}
Severity: {SEVERITY}
Patient: {PATIENT_ID}
Location: {DEPARTMENT} - {ROOM}

Details:
{MESSAGE}

Recommended Actions:
- {RECOMMENDATION_1}
- {RECOMMENDATION_2}
- ...

Confidence: {CONFIDENCE}%
Timestamp: {TIMESTAMP}
```

---

### Voice (Channel: ChannelVoice)

**Format**: Clear text-to-speech message

**Template**: `Critical alert. {SEVERITY}. Patient {PATIENT_ID} in {LOCATION} has {ALERT_TYPE} with {CONFIDENCE} percent confidence.`

**Example**:
```
"Critical alert. CRITICAL. Patient PAT-001 in ICU-5 has SEPSIS_ALERT with 92 percent confidence."
```

---

### In-App (Channel: ChannelInApp)

**Format**: Full alert message with rich formatting

**Content**:
- Alert type badge with severity color
- Patient details with location
- Full message text
- Vital signs graph (if available)
- Recommendations list
- Action buttons (Acknowledge, View Details)

---

## Priority Assignment

### Severity to Priority Mapping

| Severity | Priority | Description |
|----------|----------|-------------|
| CRITICAL | 1 | Highest - Immediate attention required |
| HIGH | 2 | High - Prompt attention needed |
| MODERATE | 3 | Medium - Attention within 30 minutes |
| ML_ALERT | 3 | Medium - For monitoring and analysis |
| LOW | 4 | Low - Informational |

**Code**:
```go
func severityToPriority(severity AlertSeverity) int {
    switch severity {
    case SeverityCritical: return 1
    case SeverityHigh: return 2
    case SeverityModerate: return 3
    case SeverityMLAlert: return 3
    case SeverityLow: return 4
    default: return 5
    }
}
```

---

## Escalation Chains

### CRITICAL Alert Escalation

**Initial**: Attending + Charge Nurse → Pager + SMS

**After 5 min** (no acknowledgment):
- **Level 2**: Charge Nurse → Pager + Voice Call

**After 10 min** (no acknowledgment):
- **Level 3**: Senior Attending → Pager + Voice Call

**After 15 min** (no acknowledgment):
- **Level 4**: Department Head + Voice Call + Incident Report

---

### HIGH Alert Escalation

**Initial**: Primary Nurse + Resident → SMS + Push

**After 15 min** (no acknowledgment):
- **Level 2**: Charge Nurse → SMS + Pager

**After 30 min** (no acknowledgment):
- **Level 3**: Attending → Pager + SMS

---

### MODERATE Alert Escalation

**Initial**: Primary Nurse → Push + In-App

**After 30 min** (no acknowledgment):
- **Level 2**: Charge Nurse → Push

**After 60 min** (no acknowledgment):
- **Level 3**: Bundled summary to Charge Nurse via Email

---

## Alert Fatigue Mitigation

### Rate Limiting

| Severity | Max Alerts/Hour | Action on Exceed |
|----------|----------------|------------------|
| CRITICAL | ∞ (unlimited) | Always deliver |
| HIGH | 30 | Bundle non-urgent |
| MODERATE | 20 | Bundle and delay |
| LOW | 10 | Bundle and delay |
| ML_ALERT | 50 | Group by model |

### Duplicate Detection

**Window**: 5 minutes

**Criteria**: Same patient + same alert type + same severity

**Action**: Suppress duplicate, keep original

### Alert Bundling

**Trigger**: 3+ similar alerts within 10 minutes

**Action**:
1. Suppress individual alerts
2. Send single bundled notification
3. Format: "BUNDLED: 4 similar LAB_RESULT alerts for PAT-001"

**Code**:
```go
if shouldBundle(history, alert) {
    history.BundleQueue = append(history.BundleQueue, alert)
    if len(history.BundleQueue) >= BUNDLE_THRESHOLD {
        sendBundledAlert(history.BundleQueue, user)
        history.BundleQueue = []AlertRecord{}
    }
    return true  // Suppress individual
}
```

---

## Decision Tree

```
Alert Received
│
├─ Is Severity = CRITICAL?
│  └─ YES → Target: Attending + Charge Nurse
│           Channels: Pager + SMS + Voice
│           Escalation: 5 min
│           Fatigue: BYPASS
│
├─ Is Severity = HIGH?
│  └─ YES → Target: Primary Nurse + Resident
│           Channels: SMS + Push
│           Escalation: 15 min
│           Fatigue: Limited
│
├─ Is Severity = MODERATE?
│  └─ YES → Target: Primary Nurse
│           Channels: Push + In-App
│           Escalation: 30 min
│           Fatigue: Full
│
├─ Is Severity = LOW?
│  └─ YES → Target: Primary Nurse
│           Channels: In-App
│           Escalation: None
│           Fatigue: Aggressive
│
└─ Is Severity = ML_ALERT?
   └─ YES → Target: Informatics Team
            Channels: Email + Push
            Escalation: None
            Fatigue: Custom

THEN:
│
├─ Is SourceModule = ML_INFERENCE?
│  └─ YES → ADD Informatics Team to recipients
│
├─ Check FatigueTracker.ShouldSuppress()
│  ├─ YES → Log suppression, skip user
│  └─ NO → Continue
│
├─ Get User Preferred Channels
│  └─ If empty → Use default severity channels
│
├─ Format message for each channel
│
├─ Send notifications (async)
│
├─ Record notification in fatigue tracker
│
└─ Schedule escalation (if required)
```

---

## Testing Checklist

### Severity Tests
- [ ] CRITICAL alerts route to Attending + Charge Nurse
- [ ] HIGH alerts route to Primary Nurse + Resident
- [ ] MODERATE alerts route to Primary Nurse only
- [ ] LOW alerts route to Primary Nurse only
- [ ] ML_ALERT routes to Informatics Team

### Channel Tests
- [ ] SMS format ≤ 160 characters
- [ ] Pager format is ultra-short
- [ ] Push includes data payload with deep link
- [ ] Email includes full details + recommendations
- [ ] Voice format is speech-friendly
- [ ] In-App shows rich formatting

### Escalation Tests
- [ ] CRITICAL escalates after 5 minutes
- [ ] HIGH escalates after 15 minutes
- [ ] MODERATE escalates after 30 minutes
- [ ] LOW does not escalate
- [ ] ML_ALERT does not escalate

### Fatigue Tests
- [ ] CRITICAL bypasses all fatigue rules
- [ ] HIGH bypasses rate limits
- [ ] MODERATE respects rate limits
- [ ] Duplicate detection works (5 min window)
- [ ] Bundling activates after 3 similar alerts

### Special Rules Tests
- [ ] ML-sourced alerts add Informatics Team
- [ ] RequiresEscalation metadata triggers escalation
- [ ] User preferences override default channels
- [ ] Empty preferences fall back to defaults

---

## Quick Code Snippets

### Check if alert should escalate
```go
func shouldScheduleEscalation(alert *Alert) bool {
    return alert.Severity == SeverityCritical ||
           alert.Severity == SeverityHigh ||
           alert.Metadata.RequiresEscalation
}
```

### Get escalation timeout
```go
func getEscalationTimeout(alert *Alert) time.Duration {
    if timeout, ok := DefaultEscalationTimeouts[alert.Severity]; ok {
        return timeout
    }
    return 15 * time.Minute
}
```

### Merge user lists without duplicates
```go
func mergeUsers(existing, additional []*User) []*User {
    userMap := make(map[string]*User)
    for _, user := range existing {
        userMap[user.ID] = user
    }
    for _, user := range additional {
        if _, exists := userMap[user.ID]; !exists {
            userMap[user.ID] = user
        }
    }
    result := make([]*User, 0, len(userMap))
    for _, user := range userMap {
        result = append(result, user)
    }
    return result
}
```

---

## Metrics to Monitor

### Key Metrics
```
alerts_routed_total{severity="critical"}
alerts_routed_total{severity="high"}
alerts_routed_total{severity="moderate"}
alerts_routed_total{severity="low"}
alerts_routed_total{severity="ml_alert"}

routing_duration_seconds (histogram)
users_targeted_total
alerts_suppressed_total{reason="rate_limit"}
alerts_suppressed_total{reason="duplicate"}
alerts_suppressed_total{reason="bundled"}
escalations_scheduled_total
```

### Alert Thresholds
- routing_duration_seconds P99 > 150ms → Performance issue
- alerts_suppressed_total{reason="rate_limit"} spike → Fatigue problem
- escalations_scheduled_total high rate → Acknowledgment issues

---

**Last Updated**: November 10, 2025
**Version**: 1.0
**Maintained By**: Clinical Engineering Team
