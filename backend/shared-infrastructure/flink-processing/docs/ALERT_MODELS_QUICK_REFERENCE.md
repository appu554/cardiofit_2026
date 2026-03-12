# Alert Models Quick Reference

## Alert Data Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        CLINICAL ALERTING PIPELINE                        │
└─────────────────────────────────────────────────────────────────────────┘

MODULE 2: Context Assembly & Enrichment
┌──────────────────────────────┐
│   Threshold Detection        │
│   - Heart rate > 140 bpm     │    ┌─────────────────┐
│   - SystolicBP < 90 mmHg     │───▶│  SimpleAlert    │
│   - MEWS score >= 5          │    │  (Threshold)    │
└──────────────────────────────┘    └─────────────────┘
                                              │
                                              │
MODULE 4: CEP Pattern Matching                │
┌──────────────────────────────┐             │
│   Complex Event Processing   │             │
│   - Sepsis pattern           │    ┌─────────────────┐
│   - Deterioration pattern    │───▶│  ClinicalAlert  │
│   - Multi-signal patterns    │    │  (CEP)          │
└──────────────────────────────┘    └─────────────────┘
                                              │
                                              │
                                              ▼
MODULE 6: Alert Composition & Deduplication
┌───────────────────────────────────────────────────────────────┐
│                     AlertComposer                             │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  1. Check AlertHistory (Flink MapState)             │    │
│  │     - alertKey = alertType + ":" + patientId        │    │
│  │     - 30-minute suppression window                  │    │
│  │     - Severity escalation bypass                    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  2. Compose Multi-Source Alert                      │    │
│  │     - Aggregate evidence from all sources           │    │
│  │     - Calculate confidence score                    │    │
│  │     - Generate recommended actions                  │    │
│  │     - Track composition strategy                    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  3. Update Deduplication State                      │    │
│  │     - Update AlertHistory with firing time          │    │
│  │     - Increment suppression count if suppressed     │    │
│  └─────────────────────────────────────────────────────┘    │
└───────────────────────────────────────────────────────────────┘
                                │
                                ▼
                    ┌─────────────────────┐
                    │   ComposedAlert     │ → Kafka Topic: clinical-alerts
                    │   (Final Output)    │ → Routing by severity
                    └─────────────────────┘ → EHR Integration
```

## Model Comparison Matrix

| Feature | SimpleAlert | ComposedAlert | AlertHistory |
|---------|-------------|---------------|--------------|
| **Purpose** | Threshold detection | Multi-source composition | Deduplication state |
| **Source Module** | Module 2 | Module 6 | Module 6 (state) |
| **Patient ID** | ✅ Required | ✅ Required | ✅ Tracking |
| **Alert Type** | AlertType enum | Inherited from sources | String (tracking) |
| **Severity** | AlertSeverity enum | AlertSeverity enum | AlertSeverity (last) |
| **Evidence** | context Map | evidence Map (aggregated) | N/A |
| **Confidence** | ❌ (implicit 1.0) | ✅ 0.0-1.0 | N/A |
| **Sources** | sourceModule (single) | sources List (multi) | N/A |
| **Actions** | ❌ | recommendedActions List | N/A |
| **Suppression** | ❌ | suppressionCount | shouldSuppress() |
| **Serializable** | ✅ Flink | ✅ Flink | ✅ Flink MapState |
| **JSON Output** | ✅ Jackson | ✅ Jackson | ❌ (internal state) |

## Severity Level Guide

```
┌─────────────────────────────────────────────────────────────────┐
│                    CLINICAL SEVERITY LEVELS                      │
└─────────────────────────────────────────────────────────────────┘

CRITICAL (Severity Score: 4)
├─ Response Time: < 15 minutes (IMMEDIATE)
├─ Clinical Actions: Direct physician notification, bedside evaluation
├─ Examples:
│  ├─ Heart rate > 180 bpm or < 40 bpm
│  ├─ Systolic BP < 70 mmHg
│  ├─ SpO2 < 85%
│  └─ Sepsis pattern with shock
└─ Method: requiresImmediateAction() → true

HIGH (Severity Score: 3)
├─ Response Time: < 1 hour (URGENT)
├─ Clinical Actions: Urgent clinical review, escalate care team
├─ Examples:
│  ├─ Heart rate 140-180 bpm
│  ├─ Systolic BP 70-90 mmHg
│  ├─ MEWS score >= 5
│  └─ Deterioration pattern detected
└─ Method: requiresClinicalReview() → true

WARNING (Severity Score: 2)
├─ Response Time: < 4-6 hours (MONITORING)
├─ Clinical Actions: Increase monitoring, review trending
├─ Examples:
│  ├─ Heart rate 100-140 bpm
│  ├─ Temperature 38-38.5°C
│  ├─ Lactate 2-4 mmol/L
│  └─ Single vital sign abnormality
└─ Method: requiresClinicalReview() → false

INFO (Severity Score: 1)
├─ Response Time: < 24 hours (ROUTINE)
├─ Clinical Actions: Routine monitoring, documentation
├─ Examples:
│  ├─ Mild vital sign variations
│  ├─ Non-critical lab values
│  └─ Clinical score trends
└─ Method: requiresImmediateAction() → false
```

## Deduplication Algorithm

```java
// Pseudocode for Module 6 Alert Composition

void processAlert(Alert alert) {
    // Step 1: Generate deduplication key
    String alertKey = alert.getAlertType() + ":" + alert.getPatientId();

    // Step 2: Retrieve alert history from Flink MapState
    AlertHistory history = alertHistoryState.get(alertKey);

    // Step 3: Check suppression window (30 minutes)
    if (history != null) {
        long timeSinceLastFired = currentTime - history.getLastFiredTime();
        boolean withinWindow = timeSinceLastFired < 30 * 60 * 1000L;
        boolean severityEscalated = history.hasEscalatedSeverity(alert.getSeverity());

        if (withinWindow && !severityEscalated) {
            // SUPPRESS: Within window, no escalation
            history.incrementSuppressedCount();
            alertHistoryState.put(alertKey, history);
            return; // Don't fire alert
        }
    }

    // Step 4: Compose alert from sources
    ComposedAlert composed = composeFromSources(alert, otherSources);

    // Step 5: Emit composed alert
    out.collect(composed);

    // Step 6: Update alert history
    AlertHistory newHistory = new AlertHistory(
        alert.getAlertType().name(),
        alert.getPatientId(),
        currentTime,
        alert.getSeverity()
    );
    alertHistoryState.put(alertKey, newHistory);
}
```

## Evidence Field Examples

### Threshold Alert Evidence (SimpleAlert):
```json
{
  "heart_rate": 165,
  "threshold": 140,
  "critical_threshold": 180,
  "measurement_time": 1696525200000
}
```

### CEP Pattern Alert Evidence:
```json
{
  "sepsis_pattern_matched": true,
  "fever": true,
  "tachycardia": true,
  "elevated_lactate": true,
  "pattern_duration_minutes": 45,
  "pattern_confidence": 0.92
}
```

### Composed Alert Evidence (Multi-Source):
```json
{
  "heart_rate": 165,
  "threshold": 140,
  "lactate": 3.5,
  "lactate_baseline": 1.2,
  "sepsis_pattern_matched": true,
  "fever": true,
  "tachycardia": true,
  "on_vasopressors": true,
  "mews_score": 6.0,
  "pattern_confidence": 0.92
}
```

## Recommended Actions Library

### Escalation Actions:
- `ESCALATE_TO_PHYSICIAN` - Direct physician notification
- `ESCALATE_TO_ICU_TEAM` - ICU team consultation
- `ACTIVATE_RAPID_RESPONSE` - Rapid response team activation

### Monitoring Actions:
- `INCREASE_MONITORING` - Increase vital sign frequency
- `CONTINUOUS_MONITORING` - Switch to continuous telemetry
- `ADD_CARDIAC_MONITORING` - Add cardiac monitor

### Clinical Interventions:
- `REVIEW_MEDICATIONS` - Medication review required
- `ORDER_LABS` - Order stat laboratory tests
- `OXYGEN_THERAPY` - Consider oxygen supplementation
- `FLUID_RESUSCITATION` - Consider IV fluid bolus

### Documentation Actions:
- `DOCUMENT_ASSESSMENT` - Document clinical assessment
- `UPDATE_CARE_PLAN` - Update plan of care
- `NOTIFY_FAMILY` - Family notification recommended

## Builder Pattern Examples

### SimpleAlert Builder:
```java
SimpleAlert alert = SimpleAlert.builder()
    .patientId("P12345")                          // Required
    .alertType(AlertType.VITAL_THRESHOLD_BREACH)  // Required
    .severity(AlertSeverity.CRITICAL)             // Required
    .message("Heart rate critically elevated")    // Required
    .addContext("heart_rate", 165)                // Optional
    .addContext("threshold", 140)                 // Optional
    .sourceModule("MODULE_2_THRESHOLD")           // Optional (defaults)
    .alertId(UUID.randomUUID().toString())        // Optional (auto-generated)
    .timestamp(System.currentTimeMillis())        // Optional (auto-generated)
    .build();
```

### ComposedAlert Builder:
```java
ComposedAlert alert = ComposedAlert.builder()
    .patientId("P12345")                          // Required
    .severity(AlertSeverity.CRITICAL)             // Required
    .confidence(0.95)                             // Required (0.0-1.0)
    .addSource("MODULE_2_THRESHOLD")              // Required (at least one)
    .addSource("MODULE_4_CEP_SEPSIS")             // Optional (multi-source)
    .addEvidence("heart_rate", 165)               // Optional
    .addEvidence("lactate", 3.5)                  // Optional
    .addRecommendedAction("ESCALATE_TO_PHYSICIAN") // Optional
    .compositionStrategy(CompositionStrategy.COMBINED) // Optional (auto-inferred)
    .build();
```

### AlertHistory Constructor:
```java
AlertHistory history = new AlertHistory(
    "VITAL_THRESHOLD_BREACH",  // alertType
    "P12345",                   // patientId
    System.currentTimeMillis(), // lastFiredTime
    AlertSeverity.CRITICAL      // lastSeverity
);

// Check suppression
boolean shouldSuppress = history.shouldSuppress(
    System.currentTimeMillis(),
    30 * 60 * 1000L  // 30 minutes in milliseconds
);

// Check escalation
boolean escalated = history.hasEscalatedSeverity(AlertSeverity.CRITICAL);
```

## Validation Rules

### SimpleAlert Validation:
- ✅ `patientId` must not be null or empty
- ✅ `alertType` must not be null
- ✅ `severity` must not be null
- ✅ `message` must not be null or empty
- ✅ `alertId` auto-generated if null
- ✅ `timestamp` auto-generated if zero
- ✅ `sourceModule` defaults to "MODULE_2_THRESHOLD"

### ComposedAlert Validation:
- ✅ `patientId` must not be null or empty
- ✅ `severity` must not be null
- ✅ `confidence` must be between 0.0 and 1.0
- ✅ `sources` list must contain at least one source
- ✅ `alertId` auto-generated if null
- ✅ `lastUpdated` auto-generated if zero
- ✅ `compositionStrategy` auto-inferred from sources if null

### AlertHistory Validation:
- ✅ No required fields (can be empty)
- ✅ `shouldSuppress()` returns false if `lastFiredTime` is zero
- ✅ `hasEscalatedSeverity()` returns false if `lastSeverity` is null

## State Management in Flink

### MapState Configuration:
```java
@Override
public void open(Configuration parameters) {
    // Configure MapState for alert history
    MapStateDescriptor<String, AlertHistory> descriptor =
        new MapStateDescriptor<>(
            "alert-history",           // State name
            String.class,              // Key type (alertKey)
            AlertHistory.class         // Value type
        );

    alertHistoryState = getRuntimeContext().getMapState(descriptor);
}
```

### State Key Strategy:
```java
// Patient-specific deduplication
String alertKey = alertType + ":" + patientId;

// Examples:
"VITAL_THRESHOLD_BREACH:P12345"
"SEPSIS_PATTERN:P67890"
"LAB_CRITICAL_VALUE:P12345"
```

## Performance Considerations

### Memory Footprint:
- **SimpleAlert**: ~500 bytes (with 5-10 context entries)
- **ComposedAlert**: ~1-2 KB (with evidence aggregation)
- **AlertHistory**: ~200 bytes per key in MapState

### Suppression Impact:
- 30-minute window: ~60 alerts/patient/hour → 2 alerts/patient/hour (97% reduction)
- State cleanup: Implement TTL on MapState (24-48 hours recommended)

### Throughput Optimization:
- Use `addSource()` method for incremental source addition (no list rebuilding)
- Use `addContext()` / `addEvidence()` for incremental map building
- Builder pattern avoids intermediate object creation

## Integration Checklist

- [ ] Configure Kafka serializers for ComposedAlert
- [ ] Set up MapState TTL for AlertHistory (24-48 hours)
- [ ] Implement AlertComposer CoProcessFunction
- [ ] Add severity-based routing (CRITICAL → immediate action topic)
- [ ] Configure monitoring for suppression rates
- [ ] Set up alerting dashboards (Grafana)
- [ ] Implement clinical action recommendation engine
- [ ] Add unit tests for deduplication logic
- [ ] Add integration tests for multi-source composition
- [ ] Configure production alert thresholds with clinical team
