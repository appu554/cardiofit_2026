# Module 6: Clinical Action & Delivery Layer — Implementation Guidelines

> **Pipeline Position:** Modules 3, 4, 5 → **Module 6 (Action & Delivery)** → External Systems
> **Operator Pattern:** `KeyedProcessFunction` with multi-sink side outputs + `KeyedCoProcessFunction` for acknowledgment feedback
> **Tech Stack:** Java 17, Flink 2.1.0, Jackson 2.17, Kafka Sinks

---

## 1. Architecture Overview

Module 6 is the output layer of the Flink pipeline. It consumes clinical intelligence from Modules 3, 4, and 5 — CDS recommendations, detected patterns, and ML predictions — and routes them to the right destination with the right urgency. It is NOT a simple Kafka sink. It is a clinical action engine that determines: what needs to happen, who needs to know, how urgent it is, and where the data needs to be persisted.

### Position in the DAG

```
Module 3 (CDS)                         Module 6 (Action & Delivery)
comprehensive-cds-events.v1 ────────►┌───────────────────────────────┐
                                      │                               │──► FHIR Store (system of record)
Module 4 (Patterns)                   │  Action Classification       │──► Elasticsearch (search/analytics)
clinical-patterns.v1 ────────────────►│  Alert Lifecycle Management  │──► Neo4j (graph summaries)
                                      │  Notification Routing        │──► PostgreSQL (alert state)
Module 5 (ML Predictions)            │  Multi-Sink Distribution     │──► Notification Service (FCM/SMS/email)
ml-predictions.v1 ───────────────────►│  Audit Trail Generation      │──► Dashboard WebSocket
high-risk-predictions.v1 ────────────►│  FHIR Write-back             │──► clinical-audit.v1 (Kafka)
                                      │  Escalation Management       │──► clinical-actions.v1 (Kafka)
                                      └───────────────────────────────┘
```

### Why Module 6 Exists as a Separate Module

Without Module 6, every upstream module (3, 4, 5) would need its own sinks, its own alert deduplication, its own notification logic, and its own FHIR write-back. Module 6 unifies the output path so that:

- A patient with a HIGH sepsis prediction (Module 5) + CLINICAL_DETERIORATION pattern (Module 4) + elevated NEWS2 CDS event (Module 3) generates ONE coordinated clinical alert, not three independent ones
- Alert fatigue management happens in one place
- FHIR write-back is atomic — one write per clinical decision, not one per module
- Audit trail is complete — every clinical action is logged with full provenance across all modules

---

## 2. Input Contracts

Module 6 consumes from four Kafka topics. Each carries different clinical intelligence with different urgency profiles.

### 2.1 CDS Events (from Module 3)

**Topic:** `comprehensive-cds-events.v1` | **Key:** `patientId`

These are the richest events in the pipeline — full patient state, risk indicators, semantic enrichment, and CDS recommendations. Module 6 uses them for: updating FHIR Store, publishing to dashboards, evaluating CDS recommendations for notification, and populating Elasticsearch.

**Key fields for action routing:**
```
cdsEvent.patientState.news2Score                    → urgency classification
cdsEvent.patientState.qsofaScore                    → urgency classification
cdsEvent.patientState.riskIndicators.activeAlerts   → structured alert triggers
cdsEvent.cdsRecommendations.monitoringFrequency     → ROUTINE vs URGENT vs STAT
cdsEvent.semanticEnrichment.evidenceBasedAlerts     → clinical evidence alerts
```

**Schema cautions (from E2E testing):**
- Vital sign keys are lowercase-no-separator: `heartrate`, `systolicbloodpressure`
- `latestVitals` may contain `age`, `gender` — filter during processing
- `LabResult.getValue()` CAN return null — always null-check
- `riskIndicators.activeAlerts` may be absent (only present after Module 3 lab-alerting fix)
- `onAnticoagulation` may be missing if Module 3 aggregator overwrites Module 2 flags

### 2.2 Pattern Events (from Module 4)

**Topic:** `clinical-patterns.v1` | **Key:** `patientId`

Pattern events represent temporal clinical findings. Module 6 uses them for generating clinical alerts for HIGH/CRITICAL patterns, updating Neo4j, and enriching dashboards.

**Key fields:**
```
patternEvent.patternType          → action type mapping
patternEvent.severity             → urgency classification
patternEvent.confidence           → alert confidence threshold
patternEvent.recommendedActions   → suggested clinical actions
patternEvent.tags                 → SEVERITY_ESCALATION triggers immediate routing
```

### 2.3 ML Predictions (from Module 5)

**Topics:** `ml-predictions.v1` (all) and `high-risk-predictions.v1` (HIGH/CRITICAL only)
**Key:** `patientId`

ML predictions are probabilistic — they need different handling than deterministic rules. Module 6 uses them for: predictive dashboards, early-warning notifications, FHIR write-back as RiskAssessment resources, and audit trail with model provenance.

**Key fields:**
```
prediction.predictionCategory     → routing (sepsis → ICU team, fall → nursing)
prediction.calibratedScore        → threshold for action
prediction.riskLevel              → urgency classification
prediction.confidence             → minimum confidence for notification
prediction.contextDepth           → INITIAL predictions get lower notification priority
prediction.explanationTexts       → included in clinician-facing alerts
```

---

## 3. Clinical Action Classification

### 3.1 Three-Tier Alert Severity Model

| Tier | Label | Notification | Dashboard | Pipeline Effect | Escalation |
|------|-------|-------------|-----------|----------------|------------|
| **HALT** | Critical safety | SMS + FCM + phone fallback | RED banner, blocks actions | All Decision Cards PAUSED | 30 min → coordinator, 2 hr → supervisor |
| **PAUSE** | Needs physician review | FCM push + email | ORANGE badge | Affected domain paused | 24 hr → reminder, 72 hr → coordinator |
| **SOFT_FLAG** | Advisory | Attached to next Decision Card | YELLOW flag on card | No effect | No escalation |

### 3.2 Action Classification Logic

```java
public class ClinicalActionClassifier {

    public static ActionTier classify(ClinicalEvent event) {

        // ══ HALT conditions (immediate danger) ══
        if (event.getNews2Score() >= 10) return ActionTier.HALT;
        if (event.getQsofaScore() >= 2 && hasSepsisIndicators(event)) return ActionTier.HALT;

        // Lab emergencies (from Module 3 activeAlerts)
        if (hasActiveAlert(event, "HYPERKALEMIA_ALERT", "CRITICAL")) return ActionTier.HALT;
        if (hasActiveAlert(event, "ANTICOAGULATION_RISK", "CRITICAL")) return ActionTier.HALT;
        if (hasActiveAlert(event, "AKI_RISK") &&
            "STAGE_3".equals(getAlertDetail(event, "AKI_RISK", "stage"))) return ActionTier.HALT;

        // ML predictions at critical threshold
        if (event.hasPrediction("sepsis") &&
            event.getPrediction("sepsis").getCalibratedScore() >= 0.60 - 1e-9) return ActionTier.HALT;

        // Pattern escalation
        if (event.hasPattern("CLINICAL_DETERIORATION") &&
            "CRITICAL".equals(event.getPattern("CLINICAL_DETERIORATION").getSeverity()))
            return ActionTier.HALT;

        // ══ PAUSE conditions (needs physician review) ══
        if (event.getNews2Score() >= 7) return ActionTier.PAUSE;
        if (event.getQsofaScore() >= 1) return ActionTier.PAUSE;

        if (hasActiveAlert(event, "AKI_RISK", "HIGH")) return ActionTier.PAUSE;
        if (hasActiveAlert(event, "ANTICOAGULATION_RISK", "HIGH")) return ActionTier.PAUSE;
        if (hasActiveAlert(event, "BLEEDING_RISK", "HIGH")) return ActionTier.PAUSE;

        if (event.hasPrediction("deterioration") &&
            event.getPrediction("deterioration").getCalibratedScore() >= 0.45 - 1e-9)
            return ActionTier.PAUSE;
        if (event.hasPrediction("sepsis") &&
            event.getPrediction("sepsis").getCalibratedScore() >= 0.35 - 1e-9)
            return ActionTier.PAUSE;

        if (event.hasPatternWithSeverity("HIGH")) return ActionTier.PAUSE;

        // ══ SOFT_FLAG conditions (advisory) ══
        if (event.getNews2Score() >= 5) return ActionTier.SOFT_FLAG;
        if (hasActiveAlert(event, "AKI_RISK", "MODERATE")) return ActionTier.SOFT_FLAG;
        if (event.hasAnyPredictionAbove(0.25)) return ActionTier.SOFT_FLAG;
        if (event.hasPatternWithSeverity("MODERATE")) return ActionTier.SOFT_FLAG;

        return ActionTier.ROUTINE;
    }
}
```

### 3.3 Cross-Module Deduplication

A septic patient generates alerts from three modules simultaneously. Module 6 deduplicates across modules so the physician gets one coordinated alert, not three.

```java
public class CrossModuleDeduplicator {

    private static final long HALT_DEDUP_WINDOW_MS = 5 * 60 * 1000;
    private static final long PAUSE_DEDUP_WINDOW_MS = 30 * 60 * 1000;
    private static final long SOFT_FLAG_DEDUP_WINDOW_MS = 60 * 60 * 1000;

    public boolean shouldEmit(String patientId, ActionTier tier,
        String clinicalCategory, long eventTime) {

        String dedupKey = patientId + ":" + tier + ":" + clinicalCategory;
        Long lastEmitted = recentAlertState.get(dedupKey);

        long window = switch (tier) {
            case HALT -> HALT_DEDUP_WINDOW_MS;
            case PAUSE -> PAUSE_DEDUP_WINDOW_MS;
            case SOFT_FLAG -> SOFT_FLAG_DEDUP_WINDOW_MS;
            default -> Long.MAX_VALUE;
        };

        if (lastEmitted != null && (eventTime - lastEmitted) < window) {
            return false;  // within dedup window — suppress
        }

        recentAlertState.put(dedupKey, eventTime);
        return true;
    }
}
```

**This dedup is different from Module 4's pattern dedup.** Module 4 deduplicates within a single pattern type. Module 6 deduplicates across modules — three modules detecting the same clinical situation produce one alert.

---

## 4. Alert Lifecycle Management

### 4.1 Alert State Machine

```
                    ┌──────────────┐
                    │   ACTIVE     │
                    └──────┬───────┘
              ┌────────────┼────────────┐
              ▼            ▼            ▼
     ┌──────────────┐ ┌──────────┐ ┌──────────────┐
     │ ACKNOWLEDGED │ │ AUTO_    │ │  ESCALATED   │
     │              │ │ RESOLVED │ │              │
     └──────┬───────┘ └──────────┘ └──────┬───────┘
            ▼                              │
     ┌──────────────┐                      │
     │  ACTIONED    │◄─────────────────────┘
     └──────┬───────┘
            ▼
     ┌──────────────┐
     │  RESOLVED    │
     └──────────────┘
```

### 4.2 Alert Entity

```java
public class ClinicalAlert implements Serializable {
    private String alertId;
    private String patientId;
    private String encounterId;

    // ── Classification ──
    private ActionTier tier;
    private String clinicalCategory;
    private String alertCode;                   // CID-01 through CID-17

    // ── Clinical Content ──
    private String title;
    private String body;
    private List<String> recommendedActions;
    private Map<String, Object> clinicalContext;
    private Map<String, Double> mlPredictions;

    // ── Source Provenance ──
    private String sourceModule;
    private String triggerEventId;
    private String correlationId;
    private List<String> contributingSources;   // ["CDS:NEWS2=13", "ML:sepsis=0.72"]

    // ── Lifecycle ──
    private AlertState state;
    private long createdAt;
    private Long acknowledgedAt;
    private Long actionedAt;
    private Long resolvedAt;
    private Long escalatedAt;
    private String acknowledgedBy;
    private String actionDescription;

    // ── SLA ──
    private long slaDeadlineMs;
    private int escalationLevel;
    private String assignedTo;

    // ── Notification Tracking ──
    private List<NotificationRecord> notificationHistory;
}
```

### 4.3 SLA Escalation via Flink Timers

```java
private void registerEscalationTimer(ClinicalAlert alert, Context ctx) {
    long escalationTime = switch (alert.getTier()) {
        case HALT -> alert.getCreatedAt() + (30 * 60 * 1000);
        case PAUSE -> alert.getCreatedAt() + (24 * 60 * 60 * 1000);
        case SOFT_FLAG -> -1;
        default -> -1;
    };
    if (escalationTime > 0) {
        ctx.timerService().registerProcessingTimeTimer(escalationTime);
    }
}

@Override
public void onTimer(long timestamp, OnTimerContext ctx,
    Collector<ClinicalAction> out) throws Exception {

    ClinicalAlert alert = activeAlertState.value();
    if (alert == null || alert.getState() != AlertState.ACTIVE) return;

    alert.setState(AlertState.ESCALATED);
    alert.setEscalatedAt(timestamp);
    alert.setEscalationLevel(alert.getEscalationLevel() + 1);

    String escalateTo = switch (alert.getEscalationLevel()) {
        case 1 -> "CARE_COORDINATOR";
        case 2 -> "CLINICAL_SUPERVISOR";
        default -> "DEPARTMENT_HEAD";
    };
    alert.setAssignedTo(escalateTo);

    out.collect(ClinicalAction.escalation(alert, escalateTo));

    // Register next escalation
    long nextEscalation = switch (alert.getTier()) {
        case HALT -> timestamp + (90 * 60 * 1000);
        case PAUSE -> timestamp + (48 * 60 * 60 * 1000);
        default -> -1;
    };
    if (nextEscalation > 0 && alert.getEscalationLevel() < 3) {
        ctx.timerService().registerProcessingTimeTimer(nextEscalation);
    }
}
```

### 4.4 Auto-Resolution

When a triggering condition resolves, the alert auto-resolves:

```java
private void checkAutoResolution(CDSEvent event, ClinicalAlert activeAlert,
    Collector<ClinicalAction> out) {

    if (activeAlert == null || activeAlert.getState() != AlertState.ACTIVE) return;

    boolean resolved = switch (activeAlert.getClinicalCategory()) {
        case "HYPERKALEMIA" -> {
            double k = safeLabValue(event.getPatientState().getRecentLabs(), "2823-3");
            yield !Double.isNaN(k) && k < 5.0;
        }
        case "SEPSIS" -> event.getPatientState().getNews2Score() < 5
            && event.getPatientState().getQsofaScore() == 0;
        case "AKI" -> {
            double cr = safeLabValue(event.getPatientState().getRecentLabs(), "2160-0");
            yield !Double.isNaN(cr) && cr < 1.5;
        }
        case "ANTICOAGULATION" -> {
            double inr = firstAvailableLab(event.getPatientState().getRecentLabs(),
                "34714-6", "6301-6");
            yield !Double.isNaN(inr) && inr < 4.0;
        }
        default -> false;
    };

    if (resolved) {
        activeAlert.setState(AlertState.AUTO_RESOLVED);
        activeAlert.setResolvedAt(System.currentTimeMillis());
        out.collect(ClinicalAction.autoResolved(activeAlert));
    }
}
```

---

## 5. Notification Routing

### 5.1 Channel Selection

```java
public class NotificationRouter {
    public static List<NotificationChannel> getChannels(ClinicalAlert alert) {
        return switch (alert.getTier()) {
            case HALT -> List.of(
                NotificationChannel.SMS,
                NotificationChannel.FCM_PUSH,
                NotificationChannel.PHONE_FALLBACK
            );
            case PAUSE -> List.of(
                NotificationChannel.FCM_PUSH,
                NotificationChannel.EMAIL
            );
            case SOFT_FLAG -> List.of(NotificationChannel.DASHBOARD_ONLY);
            default -> List.of(NotificationChannel.DASHBOARD_ONLY);
        };
    }
}
```

### 5.2 Notification Output Contract

Module 6 does NOT send notifications directly. It emits structured requests to `clinical-notifications.v1`. A separate Notification Service handles delivery. Same principle as FHIR — no external HTTP calls inside Flink operators.

```java
public class NotificationRequest implements Serializable {
    private String notificationId;
    private String alertId;
    private String patientId;
    private String recipientId;
    private String recipientRole;
    private NotificationChannel channel;
    private ActionTier tier;
    private String title;
    private String body;
    private Map<String, String> data;
    private long createdAt;
    private long expiresAt;
    private int priority;
    private String deepLink;
    private boolean requiresAcknowledgment;
}
```

### 5.3 HALT SMS Format

SMS must work offline. Keep under 160 characters:

```java
private static String formatHaltSMS(ClinicalAlert alert) {
    return String.format("HALT: Pt %s %s. Ack in app. [ID:%s]",
        alert.getPatientSummary(),
        alert.getClinicalSummary(),
        alert.getAlertId().substring(0, 4));
}
// Example: "HALT: Pt Kavitha Iyer (50F) K+ 6.5 NEWS2=13. Ack in app. [ID:A7F3]"
```

---

## 6. Multi-Sink Distribution

### 6.1 Output Destinations

| Destination | Data | Guarantee | Failure |
|------------|------|-----------|---------|
| **Kafka: clinical-notifications.v1** | Notification requests | EXACTLY_ONCE | Never drop alerts |
| **Kafka: clinical-audit.v1** | Audit trail (7-year retention) | EXACTLY_ONCE | Never drop audits |
| **Kafka: clinical-actions.v1** | Dashboard actions | AT_LEAST_ONCE | Duplicates acceptable |
| **Kafka: fhir-writeback.v1** | FHIR write requests | EXACTLY_ONCE | External service retries |
| **Kafka: patient-graph-updates.v1** | Neo4j graph updates | AT_LEAST_ONCE | Idempotent upserts |
| **Kafka: alert-state-updates.v1** | PostgreSQL alert state | EXACTLY_ONCE | Idempotent upserts |

### 6.2 Side-Output Architecture

```java
private static final OutputTag<NotificationRequest> NOTIFICATION_TAG =
    new OutputTag<>("notifications", TypeInformation.of(NotificationRequest.class));
private static final OutputTag<AuditRecord> AUDIT_TAG =
    new OutputTag<>("audit", TypeInformation.of(AuditRecord.class));
private static final OutputTag<FhirWriteRequest> FHIR_TAG =
    new OutputTag<>("fhir-writeback", TypeInformation.of(FhirWriteRequest.class));
private static final OutputTag<PatientGraphUpdate> GRAPH_TAG =
    new OutputTag<>("graph-update", TypeInformation.of(PatientGraphUpdate.class));
private static final OutputTag<AlertStateUpdate> ALERT_STATE_TAG =
    new OutputTag<>("alert-state", TypeInformation.of(AlertStateUpdate.class));
```

---

## 7. FHIR Write-back

Module 6 emits structured write requests — a separate FHIR Writer Service handles delivery.

```java
public class FhirWriteRequest implements Serializable {
    private String requestId;
    private String patientId;
    private FhirResourceType resourceType;  // OBSERVATION, RISK_ASSESSMENT, DETECTED_ISSUE,
                                             // CLINICAL_IMPRESSION, FLAG, COMMUNICATION_REQUEST
    private String fhirResourceJson;
    private WritePriority priority;          // CRITICAL, NORMAL, LOW
    private long createdAt;
    private int maxRetries;
}
```

ML Predictions → FHIR RiskAssessment, CDS Recommendations → FHIR ClinicalImpression, Safety Alerts → FHIR DetectedIssue, Patient Flags → FHIR Flag.

---

## 8. Audit Trail

Every clinical decision, alert, notification, and action must be auditable. Healthcare regulations require 7-year retention.

```java
public class AuditRecord implements Serializable {
    private String auditId;
    private long timestamp;
    private AuditEventType eventType;
    private String eventDescription;
    private String patientId;
    private String encounterId;
    private String practitionerId;
    private String sourceModule;
    private ActionTier tier;
    private String clinicalCategory;
    private Map<String, Object> clinicalData;
    private String correlationId;
    private String triggerEventId;
    private String alertId;
    private String modelVersion;
    private String kbVersion;
    private Map<String, Object> inputSnapshot;
}
```

Kafka topic config for audit: `retention.ms=220752000000` (7 years), `min.insync.replicas=2`, `compression.type=zstd`.

---

## 9. Input Stream Unification

Unify four input schemas into a single `ClinicalEvent` wrapper:

```java
public class ClinicalEvent implements Serializable {
    private String patientId;
    private long eventTime;
    private ClinicalEventSource source;
    private CDSEvent cdsEvent;           // only one of these
    private PatternEvent patternEvent;   // is non-null
    private MLPrediction prediction;     // per event

    // Convenience accessors across all source types
    public int getNews2Score() { ... }
    public int getQsofaScore() { ... }
    public boolean hasActiveAlert(String type, String severity) { ... }
    public boolean hasPattern(String type) { ... }
    public boolean hasPrediction(String category) { ... }
}
```

Stream unification:
```java
DataStream<ClinicalEvent> unified = cdsStream
    .union(patternStream, predictionStream, highRiskStream)
    .keyBy(ClinicalEvent::getPatientId);
```

---

## 10. Patient Alert State

```java
public class PatientAlertState implements Serializable {
    private String patientId;
    private Map<String, ClinicalAlert> activeAlerts;
    private Map<String, Long> recentAlertTimestamps;
    private int totalNotificationsSent;
    private long lastNotificationTime;
    private int alertsInLast24Hours;
    private long alertWindowStart;
}
```

**Alert Fatigue Protection:** Cap at 10 alerts/24hr per patient. If triggered, log a meta-alert for clinical engineering review — don't raise the cap.

```java
private static final int MAX_ALERTS_PER_24H = 10;

private boolean checkAlertFatigue(PatientAlertState state) {
    long now = System.currentTimeMillis();
    if (now - state.getAlertWindowStart() > 24 * 60 * 60 * 1000) {
        state.setAlertsInLast24Hours(0);
        state.setAlertWindowStart(now);
    }
    if (state.getAlertsInLast24Hours() >= MAX_ALERTS_PER_24H) {
        LOG.warn("Alert fatigue threshold for patient {} — {} alerts in 24h. Suppressing.",
            state.getPatientId(), state.getAlertsInLast24Hours());
        return true;
    }
    state.setAlertsInLast24Hours(state.getAlertsInLast24Hours() + 1);
    return false;
}
```

State TTL: 7-day with `OnReadAndWrite` + `NeverReturnExpired`, same as Modules 3 and 5.

---

## 11. Acknowledgment Feedback Loop

Physician acknowledgments flow back via `alert-acknowledgments.v1`:

```java
// KeyedCoProcessFunction: ClinicalEvent + AlertAcknowledgment

@Override
public void processElement2(AlertAcknowledgment ack, Context ctx,
    Collector<ClinicalAction> out) throws Exception {

    PatientAlertState state = alertState.value();
    if (state == null) return;

    ClinicalAlert alert = state.getActiveAlerts().get(ack.getClinicalCategory());
    if (alert == null || !alert.getAlertId().equals(ack.getAlertId())) return;

    switch (ack.getAction()) {
        case ACKNOWLEDGE -> {
            alert.setState(AlertState.ACKNOWLEDGED);
            alert.setAcknowledgedAt(ack.getTimestamp());
            alert.setAcknowledgedBy(ack.getPractitionerId());
        }
        case ACTION_TAKEN -> {
            alert.setState(AlertState.ACTIONED);
            alert.setActionedAt(ack.getTimestamp());
            alert.setActionDescription(ack.getActionDescription());
        }
        case DISMISS -> {
            alert.setState(AlertState.RESOLVED);
            alert.setResolvedAt(ack.getTimestamp());
            state.getActiveAlerts().remove(ack.getClinicalCategory());
        }
    }

    alertState.update(state);
    ctx.output(AUDIT_TAG, AuditRecord.alertAcknowledged(alert, ack));
}
```

---

## 12. Testing Strategy

| Test Class | Count | Coverage |
|-----------|-------|----------|
| `Module6ActionClassifierTest` | ~15 | HALT/PAUSE/SOFT_FLAG/ROUTINE across all clinical scenarios |
| `Module6DeduplicationTest` | ~8 | Cross-module dedup, escalation bypass, window expiry |
| `Module6AlertLifecycleTest` | ~10 | State transitions, auto-resolution, escalation timers |
| `Module6NotificationRoutingTest` | ~6 | Channel selection by tier, SMS formatting |
| `Module6AuditTrailTest` | ~5 | Record completeness, provenance fields |
| `Module6AlertFatigueTest` | ~4 | 24hr cap, window reset, suppression |
| `Module6FhirWritebackTest` | ~5 | RiskAssessment FHIR JSON, resource type mapping |
| `Module6RealDataIntegrationTest` | ~5 | Production E2E data → correct classification |

**Write first:**
```java
@Test
void sepsisPatient_producesHaltAlert_withCorrectNotifications() {
    // NEWS2=13, qSOFA=2, SEPSIS_PATTERN HIGH (from passing E2E scenario)
    // Assert: tier = HALT, channels = SMS+FCM, SLA = 30min
}
```

---

## 13. File Structure

```
backend/shared-infrastructure/flink-processing/src/
├── main/java/com/cardiofit/flink/
│   ├── operators/Module6_ClinicalActionEngine.java
│   ├── operators/Module6ActionClassifier.java
│   ├── operators/Module6CrossModuleDedup.java
│   ├── models/ClinicalAlert.java
│   ├── models/ClinicalAction.java
│   ├── models/ClinicalEvent.java
│   ├── models/NotificationRequest.java
│   ├── models/AuditRecord.java
│   ├── models/FhirWriteRequest.java
│   ├── models/PatientGraphUpdate.java
│   ├── models/AlertStateUpdate.java
│   ├── models/PatientAlertState.java
│   ├── routing/NotificationRouter.java
│   └── lifecycle/AlertLifecycleManager.java
├── test/java/com/cardiofit/flink/
│   ├── operators/Module6ActionClassifierTest.java
│   ├── operators/Module6DeduplicationTest.java
│   ├── operators/Module6AlertLifecycleTest.java
│   ├── operators/Module6NotificationRoutingTest.java
│   ├── operators/Module6AuditTrailTest.java
│   ├── operators/Module6AlertFatigueTest.java
│   ├── operators/Module6FhirWritebackTest.java
│   ├── operators/Module6RealDataIntegrationTest.java
│   └── builders/Module6TestBuilder.java
```

---

## 14. Implementation Order

| Step | Task | Depends On |
|------|------|-----------|
| 1 | Data models: ClinicalEvent, ClinicalAlert, ClinicalAction, NotificationRequest, AuditRecord | Nothing |
| 2 | `ClinicalActionClassifier` — static HALT/PAUSE/SOFT_FLAG logic | Step 1 |
| 3 | `Module6RealDataIntegrationTest` — validate with production E2E data | Step 2 |
| 4 | `Module6TestBuilder` — test data factory | Step 1 |
| 5 | `Module6ActionClassifierTest` — full classification tests | Steps 2, 4 |
| 6 | `CrossModuleDeduplicator` + tests | Step 1 |
| 7 | `AlertLifecycleManager` + tests | Step 1 |
| 8 | `NotificationRouter` + tests | Step 1 |
| 9 | `Module6_ClinicalActionEngine` — main operator | Steps 2, 6, 7, 8 |
| 10 | FHIR write-back, audit trail, graph update side outputs | Step 9 |
| 11 | Acknowledgment feedback loop (KeyedCoProcessFunction) | Step 9 |
| 12 | E2E integration with Modules 3+4+5 | Step 9, Module 5 E2E passing |

**Step 3 is non-negotiable before Step 9.** Use the actual CDS/pattern events from your passing E2E scenarios to validate classification. The sepsis patient (NEWS2=13, qSOFA=2) must produce HALT. The AKI patient (elevatedCreatinine, hyperkalemia) must produce HALT or PAUSE. The controls must produce ROUTINE.

---

## 15. Lessons from Modules 1–5 (Apply to Module 6)

1. **No external HTTP calls inside Flink operators.** Module 6 emits write requests to Kafka. Separate services handle FHIR writes, notifications, Neo4j updates.

2. **Silent deserialization across four input topics.** Each deserializer must validate `patientId != null`. One bad deserializer silently dropping events breaks the entire alert lifecycle.

3. **Risk indicator flags may be missing.** Module 6's classifier must use null-safe accessors for every risk flag and activeAlert check.

4. **Lab-only emergencies are invisible to vitals-based scoring.** The classifier must check lab-derived alerts (AKI, anticoagulation, hyperkalemia) independently of NEWS2/qSOFA. A patient with NEWS2=0 and K+ 6.5 is a HALT.

5. **Cross-module dedup is different from within-module dedup.** Module 4 deduplicates within pattern types. Module 6 deduplicates across modules — same clinical situation from three modules = one alert.

6. **Floating-point thresholds need epsilon tolerance.** ML thresholds (sepsis >= 0.60) use `>= 0.60 - 1e-9` for IEEE 754 boundary handling.

7. **Alert fatigue is a clinical safety issue.** 50 alerts per physician per shift = all alerts ignored. The 10/24hr cap is a safety mechanism. If it triggers, review the alerting logic — don't raise the cap.

8. **Audit trail is non-negotiable for healthcare.** Every clinical decision needs: what happened, what data triggered it, which module produced it, which model version was used, and what the patient state was at the time. 7-year retention. Tamper-evident. This is regulatory, not optional.
