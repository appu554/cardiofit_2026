# 🎯 Final Alert Flow Architecture

## ✅ Complete Data Flow (Verified from Code)

```
┌─────────────────────────────────────────────────────────────────┐
│                     MODULE 4: Pattern Detection                 │
│                     (Creates PatternEvent alerts)               │
└─────────────────────────────────────────────────────────────────┘
                               │
                ┌──────────────┴──────────────┐
                │                             │
                ▼                             ▼
     ┌─────────────────────┐       ┌─────────────────────┐
     │ pattern-events.v1   │       │ alert-management.v1 │
     │ (General patterns)  │       │ (Deterioration)     │
     └──────────┬──────────┘       └──────────┬──────────┘
                │                             │
                │         (Currently reads from both)
                │                             │
                └──────────────┬──────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│               MODULE 6: Alert Composition                       │
│  Inputs:                                                        │
│  ├─ SimpleAlert from alert-management.v1 (BROKEN - expects     │
│  │  SimpleAlert but gets PatternEvent)                         │
│  └─ PatternEvent from clinical-patterns.v1 ✅                  │
│                                                                  │
│  Processing:                                                    │
│  ├─ Deduplication (30-minute window)                           │
│  ├─ Merging related alerts                                     │
│  ├─ Enrichment with context                                    │
│  └─ Severity-based routing                                     │
└─────────────────────────────────────────────────────────────────┘
                               │
                ┌──────────────┴──────────────┐
                │                             │
                ▼                             ▼
     ┌─────────────────────┐       ┌─────────────────────┐
     │ composed-alerts.v1  │       │ urgent-alerts.v1    │
     │ (ALL alerts)        │       │ (HIGH/CRITICAL)     │
     └──────────┬──────────┘       └─────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────────┐
│               MODULE 6.3: Analytics Engine                      │
│  Reads: composed-alerts.v1                                      │
│  Aggregates: 1-minute tumbling windows                          │
│  Outputs: analytics-alert-metrics (with patient_ids) ✅        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📋 Module 4 Outputs (Verified)

**File**: Module4_PatternDetection.java

| Output Topic | Line | Content | Purpose |
|--------------|------|---------|---------|
| **pattern-events.v1** | 1497 | PatternEvent (general patterns) | All clinical patterns |
| **alert-management.v1** | 1524 | PatternEvent (deterioration) | Deterioration-specific patterns |

**Both contain PatternEvent format!**

---

## 📋 Module 6 Alert Composition (Current State)

### Inputs (Line 536-579):
```java
// Input 1: BROKEN
SimpleAlert from alert-management.v1
└─ Expects: SimpleAlert format
└─ Actually contains: PatternEvent
└─ Result: DESERIALIZATION FAILS ❌

// Input 2: WORKS
PatternEvent from clinical-patterns.v1 ✅
```

### Outputs (Line 591-644):
```java
// Output 1: All alerts
ComposedAlert → composed-alerts.v1

// Output 2: Urgent alerts only (HIGH/CRITICAL severity)
ComposedAlert → urgent-alerts.v1
```

---

## 📊 Module 6.3 Analytics

### Input:
- `composed-alerts.v1` (ComposedAlert format)

### Output:
- `analytics-alert-metrics` (with patient_ids field) ✅

---

## 🔧 What Needs to be Fixed

### Module 6 Alert Composition - Remove Broken Source

**Current (Broken)**:
```java
// Line 94-106: Merges TWO sources
DataStream<SimpleAlert> simpleAlerts = createSimpleAlertSource(env);  // ❌ FAILS
DataStream<PatternEvent> patternEvents = createPatternEventSource(env);  // ✅ WORKS

DataStream<SimpleAlert> convertedPatternAlerts = patternEvents
    .map(new PatternToAlertConverter());

DataStream<SimpleAlert> allAlerts = simpleAlerts
    .union(convertedPatternAlerts);  // ❌ simpleAlerts stream fails!
```

**Fixed (Recommended)**:
```java
// Use ONLY PatternEvent sources (both from Module 4)
DataStream<PatternEvent> patternEventsGeneral = createPatternEventSource(env, "pattern-events.v1");
DataStream<PatternEvent> patternEventsDeterior = createPatternEventSource(env, "alert-management.v1");

// Union both PatternEvent streams
DataStream<PatternEvent> allPatternEvents = patternEventsGeneral
    .union(patternEventsDeterior);

// Convert to SimpleAlert for processing
DataStream<SimpleAlert> allAlerts = allPatternEvents
    .map(new PatternToAlertConverter());
```

---

## ✅ Final Alert Output

**Module 6 Alert Composition produces TWO outputs**:

### 1. composed-alerts.v1 (All Alerts)
- **Format**: ComposedAlert
- **Content**: ALL alerts after deduplication/composition
- **Consumers**:
  - Module 6.3 Analytics ✅
  - Alert dashboards
  - Alert history services

### 2. urgent-alerts.v1 (Urgent Alerts)
- **Format**: ComposedAlert
- **Content**: ONLY HIGH/CRITICAL severity alerts
- **Consumers**:
  - Paging systems
  - Notification services
  - Real-time alert monitors

---

## 🎯 Summary

**Question**: "Module 6 needs to read PatternEvents directly and output alerts?"

**Answer**:

1. ✅ **Module 6 SHOULD read PatternEvents** from:
   - `pattern-events.v1` (general patterns from Module 4)
   - `alert-management.v1` (deterioration patterns from Module 4)
   - Both contain PatternEvent format

2. ✅ **Module 6 outputs ComposedAlerts** to:
   - `composed-alerts.v1` (all alerts)
   - `urgent-alerts.v1` (HIGH/CRITICAL only)

3. ✅ **Module 6.3 Analytics reads** composed-alerts.v1 → produces alert metrics with patient_ids

**The output alerts = ComposedAlert objects in composed-alerts.v1 topic!**
