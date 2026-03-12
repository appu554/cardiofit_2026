# Clinical Alert Data Models Implementation Summary

## Overview
Implemented three core alert data models for the 6-module clinical alerting pipeline in the Flink streaming platform. These models support threshold-based detection, CEP pattern matching, alert composition, and deduplication.

## Files Created/Updated

### 1. Enums
- **AlertSeverity.java** - Clinical alert severity levels with urgency mapping
  - `INFO` / `WARNING` / `HIGH` / `CRITICAL`
  - Includes severity scoring and immediate action checks
  - Backward compatibility with deprecated `LOW` / `MODERATE`

- **AlertType.java** - Alert categorization types
  - `VITAL_THRESHOLD_BREACH`, `LAB_CRITICAL_VALUE`, `MEDICATION_MISSED`, etc.
  - Supports routing and clinical decision support

- **CompositionStrategy.java** - Alert composition methodology
  - `THRESHOLD_ONLY`, `CEP_ONLY`, `COMBINED`, `ML_ENRICHED`
  - Indicates how alerts were composed from detection modules

### 2. SimpleAlert.java (Updated)
**Purpose**: Threshold-based alerts from Module 2 (Context Assembly & Enrichment)

**Key Features**:
- **Fields**:
  - `alertId` (UUID) - Unique alert identifier
  - `patientId` (String) - Patient identifier
  - `alertType` (AlertType enum) - Type-safe categorization
  - `severity` (AlertSeverity enum) - Clinical urgency level
  - `message` (String) - Human-readable description
  - `context` (Map<String, Object>) - Supporting clinical data (vital values, thresholds)
  - `timestamp` (long) - Event time in milliseconds
  - `sourceModule` (String) - Originating module (e.g., "MODULE_2_THRESHOLD")

- **Design Patterns**:
  - Builder pattern with validation
  - Serializable for Flink state backend
  - Jackson annotations for JSON serialization
  - Required field validation (patientId, alertType, severity, message)
  - Auto-generated UUID and timestamp defaults

**Example Usage**:
```java
SimpleAlert alert = SimpleAlert.builder()
    .patientId("P12345")
    .alertType(AlertType.VITAL_THRESHOLD_BREACH)
    .severity(AlertSeverity.CRITICAL)
    .message("Heart rate critically elevated: 165 bpm")
    .addContext("heart_rate", 165)
    .addContext("threshold", 140)
    .sourceModule("MODULE_2_THRESHOLD")
    .build();
```

### 3. ComposedAlert.java (New)
**Purpose**: High-level alerts composed from multiple detection sources (Module 6)

**Key Features**:
- **Multi-Source Aggregation**:
  - `sources` (List<String>) - Contributing modules ("MODULE_2_THRESHOLD", "MODULE_4_CEP_SEPSIS")
  - `evidence` (Map<String, Object>) - All supporting clinical data from contributing alerts
  - `confidence` (double 0.0-1.0) - Detection consensus confidence score

- **Clinical Decision Support**:
  - `recommendedActions` (List<String>) - Clinical actions ("ESCALATE_TO_PHYSICIAN", "INCREASE_MONITORING")
  - `compositionStrategy` (CompositionStrategy) - How alert was composed

- **Deduplication Tracking**:
  - `suppressionCount` (int) - How many duplicate alerts were suppressed
  - `lastUpdated` (long) - Last update timestamp

- **Methods**:
  - `addSource(String source)` - Append contributing module
  - `incrementSuppressionCount()` - Track deduplication

- **Design Patterns**:
  - Builder pattern with confidence validation (0.0-1.0 range)
  - Auto-inference of composition strategy from sources
  - Full audit trail for clinical decision support
  - Serializable for Flink state

**Example Usage**:
```java
ComposedAlert alert = ComposedAlert.builder()
    .patientId("P12345")
    .severity(AlertSeverity.CRITICAL)
    .confidence(0.95)
    .addSource("MODULE_2_THRESHOLD")
    .addSource("MODULE_4_CEP_SEPSIS")
    .addEvidence("heart_rate", 165)
    .addEvidence("lactate", 3.5)
    .addEvidence("sepsis_pattern_matched", true)
    .addRecommendedAction("ESCALATE_TO_PHYSICIAN")
    .addRecommendedAction("INCREASE_MONITORING")
    .compositionStrategy(CompositionStrategy.COMBINED)
    .build();
```

### 4. AlertHistory.java (New)
**Purpose**: Alert firing history for 30-minute deduplication window (Module 6)

**Key Features**:
- **Deduplication State**:
  - `alertType` (String) - Type of alert being tracked
  - `patientId` (String) - Patient identifier
  - `lastFiredTime` (long) - When alert last fired
  - `suppressedCount` (int) - How many duplicates suppressed
  - `lastSeverity` (AlertSeverity) - Severity of last firing

- **Methods**:
  - `shouldSuppress(currentTime, suppressionWindowMs)` - Time-based suppression check
  - `incrementSuppressedCount()` - Track suppression
  - `updateFiring(currentTime, severity)` - Update on new firing
  - `hasEscalatedSeverity(currentSeverity)` - Detect severity escalation

- **Design Patterns**:
  - Serializable for Flink MapState
  - 30-minute suppression window per architecture specification
  - Severity escalation detection (bypass suppression on escalation)

**Example Usage in Flink**:
```java
MapStateDescriptor<String, AlertHistory> descriptor =
    new MapStateDescriptor<>("alert-history", String.class, AlertHistory.class);
MapState<String, AlertHistory> state = getRuntimeContext().getMapState(descriptor);

String alertKey = alert.getAlertType() + ":" + alert.getPatientId();
AlertHistory history = state.get(alertKey);

if (history != null && history.shouldSuppress(System.currentTimeMillis(), 30 * 60 * 1000L)) {
    history.incrementSuppressedCount();
    state.put(alertKey, history);
    return; // Suppress duplicate
}

// Fire new alert
state.put(alertKey, new AlertHistory(
    alert.getAlertType().name(),
    alert.getPatientId(),
    System.currentTimeMillis(),
    alert.getSeverity()
));
```

## Alert Model Relationships

```
┌─────────────────┐
│  SimpleAlert    │ (Module 2: Threshold-based)
│  - alertId      │
│  - patientId    │     ┌──────────────────────┐
│  - alertType    │────▶│  ComposedAlert       │ (Module 6: Composition)
│  - severity     │     │  - Multi-source      │
│  - context      │     │  - Confidence score  │
└─────────────────┘     │  - Evidence trail    │
                        │  - Recommended acts  │
┌─────────────────┐     │  - Dedup tracking    │
│  ClinicalAlert  │────▶│                      │
│  (CEP Pattern)  │     └──────────────────────┘
└─────────────────┘                │
                                   │ Uses for deduplication
                                   ▼
                        ┌──────────────────────┐
                        │  AlertHistory        │ (Flink MapState)
                        │  - 30-min window     │
                        │  - Suppression count │
                        │  - Severity tracking │
                        └──────────────────────┘
```

## Deduplication Logic (Module 6)

### How Deduplication Works:

1. **Alert Firing**:
   - New alert arrives (SimpleAlert or ClinicalAlert)
   - Generate alertKey: `alertType + ":" + patientId`
   - Check AlertHistory state for this key

2. **Suppression Check**:
   ```java
   if (history.shouldSuppress(currentTime, 30 * 60 * 1000L)) {
       history.incrementSuppressionCount();
       return; // Don't fire, just track
   }
   ```

3. **Severity Escalation Bypass**:
   - If `history.hasEscalatedSeverity(newSeverity)`, fire alert despite suppression window
   - Example: WARNING → CRITICAL always fires

4. **New Alert Firing**:
   - Create/update ComposedAlert with evidence
   - Update AlertHistory with new firing time
   - Reset suppression count

5. **30-Minute Window**:
   - Per architecture specification (line 748 in C05_10 doc)
   - Prevents alert fatigue
   - Preserves clinical safety with severity escalation bypass

### Suppression Window Configuration:
```java
private static final long SUPPRESSION_WINDOW_MS = 30 * 60 * 1000L; // 30 minutes
```

## Clinical Information Preserved in Evidence Field

The `evidence` field in ComposedAlert preserves complete clinical context:

### 1. Vital Sign Data:
- Actual values: `heart_rate: 165`, `systolic_bp: 85`, `spo2: 89`
- Thresholds: `threshold: 140`, `critical_threshold: 180`
- Trends: `heart_rate_trend: "INCREASING"`, `bp_trend: "DECREASING"`

### 2. Laboratory Values:
- Critical values: `lactate: 3.5`, `creatinine: 2.1`, `wbc: 15.2`
- Baselines: `baseline_creatinine: 1.0`
- Abnormal flags: `lactate_elevated: true`

### 3. Clinical Patterns (from CEP):
- Pattern matches: `sepsis_pattern_matched: true`
- Pattern evidence: `fever: true`, `tachycardia: true`, `elevated_lactate: true`
- Timeframe: `pattern_duration_minutes: 45`

### 4. Medication Context:
- Active medications: `on_vasopressors: true`, `recent_med_change: true`
- Timing: `medication_ordered_at: 1696525200000`

### 5. Clinical Scores:
- Score values: `mews: 6.0`, `sofa: 4.0`
- Score thresholds: `mews_threshold: 5.0`

### 6. Audit Trail:
- Detection sources: `sources: ["MODULE_2_THRESHOLD", "MODULE_4_CEP_SEPSIS"]`
- Composition strategy: `compositionStrategy: COMBINED`
- Confidence: `confidence: 0.95`

**Clinical Safety**: All evidence is preserved for:
- Clinical decision support
- Audit compliance
- Root cause analysis
- Quality improvement

## Design Decisions & Trade-offs

### 1. Type Safety vs Flexibility
**Decision**: Use enums (AlertType, AlertSeverity) instead of strings
**Rationale**:
- Type safety at compile time
- IDE autocomplete support
- Prevents typos in alert types
**Trade-off**: Less flexibility for dynamic alert types (mitigated by comprehensive enum values)

### 2. Builder Pattern with Validation
**Decision**: Immutable builders with validation in build() method
**Rationale**:
- Prevents invalid alert objects
- Clear error messages at construction time
- Fluent API for readability
**Trade-off**: Slightly more verbose than constructors

### 3. Confidence Score (0.0-1.0)
**Decision**: Double precision for confidence, validated range
**Rationale**:
- Standard ML confidence representation
- Enables probabilistic alert ranking
- Supports multi-source evidence weighting
**Trade-off**: Not needed for pure threshold alerts (use 1.0)

### 4. Map<String, Object> for Context/Evidence
**Decision**: Generic map instead of typed fields
**Rationale**:
- Flexibility for different alert types
- Supports evolving clinical data requirements
- JSON serialization friendly
**Trade-off**: No compile-time type safety for context data (acceptable for clinical flexibility)

### 5. Suppression by alertType + patientId
**Decision**: Composite key for deduplication state
**Rationale**:
- Patient-specific suppression (different patients can have same alert type)
- Prevents cross-patient alert suppression
**Trade-off**: More state storage (acceptable for clinical safety)

### 6. Severity Escalation Bypass
**Decision**: Allow severity escalation to bypass suppression window
**Rationale**:
- Clinical safety: worsening conditions must alert
- Example: WARNING at T+0, CRITICAL at T+10 minutes → both fire
**Implementation**: `hasEscalatedSeverity()` method in AlertHistory

### 7. Auto-inference of Composition Strategy
**Decision**: Infer strategy from sources if not explicitly set
**Rationale**:
- Reduces boilerplate
- Consistent strategy assignment
- Can be overridden if needed
**Logic**:
- Both threshold + CEP sources → COMBINED
- CEP sources only → CEP_ONLY
- Threshold sources only → THRESHOLD_ONLY

## Clinical Safety Requirements Met

✅ **SimpleAlert Severity Mapping**:
- CRITICAL = immediate response required (< 15 minutes)
- HIGH = urgent attention (< 1 hour)
- WARNING = monitoring and potential intervention (< 4-6 hours)
- INFO = routine monitoring (< 24 hours)

✅ **ComposedAlert Evidence Preservation**:
- Complete audit trail with all contributing sources
- All clinical context preserved in evidence field
- Confidence scoring for multi-source validation
- Recommended actions for clinical decision support

✅ **AlertHistory Suppression Window**:
- 30-minute suppression per architecture (C05_10 line 748)
- Severity escalation bypass for clinical safety
- Suppression count tracking for alert fatigue metrics
- Patient-specific suppression (no cross-contamination)

## Integration with Flink Pipeline

### Module 2 (Context Assembly & Enrichment):
```java
// Generate SimpleAlert
SimpleAlert alert = SimpleAlert.builder()
    .patientId(patientId)
    .alertType(AlertType.VITAL_THRESHOLD_BREACH)
    .severity(AlertSeverity.CRITICAL)
    .message("Heart rate critically elevated: " + heartRate + " bpm")
    .addContext("heart_rate", heartRate)
    .addContext("threshold", 140)
    .sourceModule("MODULE_2_THRESHOLD")
    .build();

// Emit to downstream
out.collect(alert);
```

### Module 6 (Alert Composition & Deduplication):
```java
// Check deduplication
String alertKey = simpleAlert.getAlertType() + ":" + simpleAlert.getPatientId();
AlertHistory history = alertHistoryState.get(alertKey);

if (history != null && history.shouldSuppress(System.currentTimeMillis(), SUPPRESSION_WINDOW_MS)) {
    if (!history.hasEscalatedSeverity(simpleAlert.getSeverity())) {
        history.incrementSuppressedCount();
        alertHistoryState.put(alertKey, history);
        return; // Suppress
    }
}

// Compose alert
ComposedAlert composed = ComposedAlert.builder()
    .patientId(simpleAlert.getPatientId())
    .severity(simpleAlert.getSeverity())
    .confidence(1.0) // Threshold alerts have 100% confidence
    .addSource("MODULE_2_THRESHOLD")
    .evidence(simpleAlert.getContext())
    .compositionStrategy(CompositionStrategy.THRESHOLD_ONLY)
    .build();

out.collect(composed);

// Update history
alertHistoryState.put(alertKey, new AlertHistory(
    simpleAlert.getAlertType().name(),
    simpleAlert.getPatientId(),
    System.currentTimeMillis(),
    simpleAlert.getSeverity()
));
```

## File Locations

```
/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/
├── AlertSeverity.java          (Updated with severity scoring)
├── AlertType.java              (New enum)
├── CompositionStrategy.java    (New enum)
├── SimpleAlert.java            (Updated with patientId, alertId, AlertType enum)
├── ComposedAlert.java          (New - alert composition model)
└── AlertHistory.java           (New - deduplication state model)
```

## Testing Recommendations

### Unit Tests:
1. **SimpleAlert.Builder validation**:
   - Test required field validation
   - Test default value generation (UUID, timestamp)
   - Test context map immutability

2. **ComposedAlert.Builder validation**:
   - Test confidence range validation (0.0-1.0)
   - Test source list management
   - Test composition strategy auto-inference

3. **AlertHistory suppression logic**:
   - Test 30-minute window calculation
   - Test severity escalation bypass
   - Test suppression count increment

### Integration Tests:
1. **Deduplication flow**:
   - Send duplicate alerts within 30 minutes → verify suppression
   - Send severity escalation → verify fires despite window
   - Send alert after 30 minutes → verify new firing

2. **Multi-source composition**:
   - Combine SimpleAlert + ClinicalAlert → verify evidence aggregation
   - Verify confidence calculation
   - Verify recommended actions merging

## Compilation Status

✅ All alert model files compile successfully
✅ No compilation errors in SimpleAlert, ComposedAlert, AlertHistory
✅ Jackson serialization annotations validated
✅ Flink Serializable interface implemented

Note: Unrelated compilation errors exist in EncounterContextSerializer.java and PatientSnapshotSerializer.java (not part of this implementation).

## Next Steps

1. **Implement Alert Composition Logic** (Module 6):
   - Create `AlertComposer` CoProcessFunction
   - Implement multi-source alert merging
   - Add ML prediction enrichment (Module 5 integration)

2. **Add Kafka Sinks**:
   - Configure Kafka serializers for ComposedAlert
   - Set up alert routing topics by severity

3. **Monitoring & Metrics**:
   - Track suppression rates
   - Monitor alert composition strategies
   - Alert severity distribution metrics

4. **Clinical Validation**:
   - Review alert thresholds with clinical team
   - Validate recommended actions
   - Tune 30-minute suppression window if needed
