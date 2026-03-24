# Ingestion Service: Outbox Integration + Flink Pipeline + KB Threshold Centralization

**Date:** 2026-03-23
**Status:** Draft
**Branch:** `feature/kb25-kb26-implementation`
**Approach:** Modified Approach A — Outbox SDK with ingestion-specific topics

---

## 1. Overview

Replace the ingestion service's direct dual-write pattern (inline FHIR Store + Kafka in HTTP handlers) with the platform-wide Global Outbox SDK for atomic event publishing. Wire the ingestion events through the existing Flink analytical pipeline. Centralize all hardcoded clinical thresholds in Flink by connecting to KB services via async HTTP + BroadcastState hot-swap.

| Component | Current State | Target State |
|-----------|--------------|-------------|
| Kafka publishing | Direct `segmentio/kafka-go` writes in `producer.go` | Outbox SDK `SaveAndPublish()` / `SaveAndPublishBatch()` |
| FHIR Store writes | Synchronous in HTTP handler (kept) | Synchronous in HTTP handler (kept — no change) |
| Flink consumption | Does not consume `ingestion.*` topics | Module1b IngestionCanonicalizer reads `ingestion.*` topics |
| Flink FHIR writes | FHIRRouter writes all events to FHIR | Routing operator checks `sourceSystem=="ingestion-service"` → sets `sendToFHIR=false` (already in FHIR) |
| Clinical thresholds | 50+ hardcoded values in Flink Java | Live from KB-4, KB-20, KB-1, KB-23 via BroadcastState |

---

## 2. Architecture

### 2.1 Data Flow — Routine Observation

```
Flutter App / Lab Webhook / Wearable Sync / WhatsApp NLU / ABDM HIE
  │
  │ HTTP POST
  ▼
API Gateway (JWT validation, tenant check, OpenTelemetry trace ID)
  │
  ▼
Ingestion Service (port 8140)
  ├─ Stage 1: Adapter (parse source-specific format)
  ├─ Stage 2: Normalize (units) + Validate (plausibility, critical value detection, quality scoring)
  ├─ Stage 3: FHIR Map + Store (SYNCHRONOUS — returns fhir_resource_id)
  ├─ Stage 4: Outbox SDK Write (ATOMIC with business DB — SaveAndPublish)
  └─ HTTP 200/201 → caller (~295ms total)

  [ASYNCHRONOUS — patient never waits beyond this point]

  outbox_events_ingestion_service (PostgreSQL, status=pending)
  │
  │ Central Publisher polls every 2s, priority-ordered
  ▼
Kafka: ingestion.{labs|vitals|device-data|patient-reported|...}
  │
  ▼
Flink Module1b (IngestionCanonicalizer) → Module2 (Enrichment)
  → Module3 (CDS) → Module4 (Pattern Detection / CEP)
  → Module5 (ML Scoring) → Module6 (Egress Routing)
  │
  ├─ FHIRRouter: SKIP (routing operator sets sendToFHIR=false for sourceSystem=ingestion-service)
  ├─ CriticalAlertRouter → prod.ehr.alerts.critical (if applicable)
  ├─ AuditRouter → prod.ehr.audit.logs
  ├─ AnalyticsRouter → prod.ehr.analytics.events
  └─ GraphRouter → prod.ehr.graph.mutations

  Flink writes DERIVED intelligence to FHIR:
    RiskAssessment, DetectedIssue, CarePlan (NOT raw Observations)

  KB-20 consumes enriched topic → INSERT ... ON CONFLICT (source_event_id) DO NOTHING
```

### 2.2 Data Flow — Critical Value (K+ 6.8 mEq/L)

Same as routine until Stage 2. Validator sets `CRITICAL_VALUE` flag. Stage 4 uses `SaveAndPublishBatch()` for dual-publish in ONE PostgreSQL transaction:

```
SaveAndPublishBatch(ctx, []EventRequest{
  {EventType: "observation.created", Topic: "ingestion.labs",            MedicalContext: "critical", Priority: 1, EventData: canonicalObs},
  {EventType: "observation.created", Topic: "ingestion.safety-critical", MedicalContext: "critical", Priority: 1, EventData: canonicalObs},
}, nil) // nil businessLogic — FHIR write already completed in Stage 3
```

Both rows: same `event_data` payload (canonical observation + fhir_resource_id), different `topic` and `event_type`. Central Publisher processes critical events FIRST in every poll cycle. Circuit breaker: critical events ALWAYS pass, even when circuit is OPEN.

Safety path: `ingestion.safety-critical` → KB-22 consumer (100ms polling) → deterioration detection → KB-23 Decision Card → physician push notification. Latency: ~2.5s.

### 2.3 Critical Value Dual-Publish Triggers

| Condition | Threshold | Source Topic | Safety Topic |
|-----------|-----------|-------------|-------------|
| Hyperkalemia | K+ > 5.5 mEq/L | `ingestion.labs` | `ingestion.safety-critical` |
| Hypokalemia | K+ < 3.0 mEq/L | `ingestion.labs` | `ingestion.safety-critical` |
| Severe hyperglycemia | FBG > 300 mg/dL | source varies | `ingestion.safety-critical` |
| Severe hypoglycemia | FBG < 54 mg/dL | source varies | `ingestion.safety-critical` |
| eGFR critical | eGFR < 15 mL/min | `ingestion.labs` | `ingestion.safety-critical` |
| Hypertensive crisis | BP > 180/120 mmHg | `ingestion.vitals` | `ingestion.safety-critical` |

Routine observations: single `SaveAndPublish()` to source topic only.

### 2.4 FHIR Write Separation

| Writer | FHIR Resource Types | When | Why |
|--------|-------------------|------|-----|
| **Ingestion Service** (synchronous) | `Observation` | Before outbox write, in HTTP handler | Patient app needs immediate confirmation; ABDM compliance; FHIR resource ID in Kafka payload |
| **Flink** (asynchronous) | `RiskAssessment`, `DetectedIssue`, `CarePlan` | After trend analysis, pattern detection, ML scoring | Derived clinical intelligence — what the raw facts mean |

FHIRRouter skip mechanism:

The Flink pipeline model hierarchy is: `CanonicalEvent` → Module2 enrichment → `EnrichedClinicalEvent` → `TransactionalMultiSinkRouterV2_OptionC` → `RoutedEnrichedEvent` (with `RoutingDecision`) → FHIRRouter. The `CanonicalEvent` has no routing field — routing decisions are made by `TransactionalMultiSinkRouterV2_OptionC` which inspects the enriched event and creates a `RoutingDecision`.

To prevent duplicate FHIR writes, the routing operator's `shouldPersistToFHIR()` checks `sourceSystem`:

```java
// In TransactionalMultiSinkRouterV2_OptionC.shouldPersistToFHIR():
if ("ingestion-service".equals(event.getSourceSystem())) {
    return false; // Raw Observation already in FHIR from ingestion Stage 3
}
// ... existing logic for EHR pipeline events unchanged ...
```

Module1b sets `sourceSystem = "ingestion-service"` on every `CanonicalEvent` it produces. This propagates through enrichment to `EnrichedClinicalEvent.getSourceSystem()`, where the routing operator reads it.

The FHIRRouter itself remains unchanged:
```java
// FHIRRouter (UNCHANGED):
.filter(event -> event.getRouting() != null && event.getRouting().isSendToFHIR())
```

Similarly, critical value routing for ingestion events is handled in `isCriticalEvent()`:
```java
// In TransactionalMultiSinkRouterV2_OptionC.isCriticalEvent():
if ("ingestion-service".equals(event.getSourceSystem())) {
    List<?> flags = (List<?>) event.getPayload().get("flags");
    if (flags != null && flags.contains("CRITICAL_VALUE")) return true;
}
```

---

## 3. Outbox Integration Details

### 3.1 Outbox SDK Configuration

```go
outboxConfig := &outboxsdk.ClientConfig{
    ServiceName:          "ingestion-service",
    DatabaseURL:          "postgresql://ingestion_user:***@localhost:5433/ingestion_service",
    OutboxServiceGRPCURL: "localhost:50052",
    DefaultTopic:         "ingestion.observations",
    DefaultPriority:      5,
    DefaultMedicalContext: "routine",
}
```

The SDK auto-creates `outbox_events_ingestion_service` table with 8 indexes. Central Publisher auto-discovers it via service registry scan.

### 3.2 Event Envelope Format

Outbox SDK envelope is generic (platform-level). Clinical fields inside `event_data` JSONB:

```json
{
  "id": "outbox-generated-uuid",
  "service_name": "ingestion-service",
  "event_type": "observation.created",
  "topic": "ingestion.labs",
  "correlation_id": "otel-trace-uuid",
  "priority": 5,
  "medical_context": "routine",
  "metadata": {
    "patient_id": "venkatesh-uuid",
    "source": "thyrocare",
    "loinc": "2160-0"
  },
  "event_data": {
    "event_id": "ingestion-generated-uuid",
    "patient_id": "venkatesh-uuid",
    "tenant_id": "cardiofit-demo",
    "channel_type": "corporate",
    "observation_type": "serum_creatinine",
    "loinc_code": "2160-0",
    "value": 1.4,
    "unit": "mg/dL",
    "effective_at": "2026-03-23T10:30:00Z",
    "source_type": "diagnostic_lab",
    "source_name": "thyrocare",
    "quality_score": 0.92,
    "flags": [],
    "device_context": null,
    "clinical_context": {
      "ordering_provider": null,
      "specimen_type": "serum",
      "reference_range": "0.6-1.2 mg/dL"
    },
    "fhir_resource_id": "Observation/abc123",
    "abdm_context": null
  },
  "status": "pending",
  "created_at": "2026-03-23T10:30:05Z"
}
```

**Envelope design rationale:**
- `metadata` JSONB: lightweight indexed fields for operational tooling (DLQ dashboard, monitoring). Denormalized copies, not authoritative.
- `event_data` JSONB: complete canonical observation. Source of truth for downstream consumers (KB-20, KB-22, KB-26, Flink).
- `patient_id` appears in both: intentional duplication. DLQ dashboard needs fast patient lookup without deserializing `event_data`.
- Outbox envelope is NOT extended with clinical columns. Per-service schema bloat defeats the shared SDK. Central Publisher never inspects `event_data`.

### 3.3 Topic Architecture

| Topic | Partitions | Retention | Medical Context | Purpose |
|-------|-----------|-----------|----------------|---------|
| `ingestion.labs` | 12 | 90 days | routine or critical | Authoritative lab results (Thyrocare, Redcliffe, etc.) |
| `ingestion.vitals` | 8 | 30 days | routine or critical | BP, HR, weight, temperature |
| `ingestion.device-data` | 8 | 30 days | routine | BLE glucometers, BP monitors |
| `ingestion.patient-reported` | 8 | 30 days | routine | App check-in, WhatsApp NLU |
| `ingestion.wearable-aggregates` | 4 | 14 days | background | Health Connect, Apple Health aggregates |
| `ingestion.cgm-raw` | 4 | 7 days (compacted) | background | Ultrahuman M1 CGM raw readings (288/day) |
| `ingestion.abdm-records` | 4 | 180 days | routine | ABDM HIE records (regulatory retention) |
| `ingestion.medications` | 8 | 90 days | routine | Medication updates (from ObsMedications type) |
| `ingestion.observations` | 8 | 30 days | routine | General fallback for unclassified observation types |
| `ingestion.safety-critical` | 4 | 90 days | critical | Critical values from any source |
| `kb.clinical-thresholds.changes` | 1 | 7 days (compacted) | — | KB threshold hot-swap for Flink |

### 3.4 Medical Circuit Breaker Behavior (Morning Burst)

During 6-8 AM IST peak (40-60% of daily traffic in 2 hours):

| Medical Context | Circuit CLOSED | Circuit OPEN (queue > 1000) |
|----------------|---------------|---------------------------|
| **critical** (K+ 6.8, BP 190/125) | Processed | **Always processed** |
| **urgent** (abnormal labs) | Processed | **Always processed** |
| **routine** (normal FBG, BP) | Processed | Deferred (retry in 5 min) |
| **background** (CGM, steps, sleep) | Processed | Dropped first (retry after recovery) |

Patient already received HTTP 200 — no UX impact from deferred/dropped events. CGM loss during overload is clinically acceptable (aggregates recomputed from full window on recovery).

**Note on `urgent` context:** The `medical_context` field in the outbox SDK supports four values: `critical`, `urgent`, `routine`, `background`. Ingestion events use three: `critical` (dual-published critical values), `routine` (normal labs, vitals, patient-reported, ABDM), and `background` (CGM raw, wearable aggregates). The `urgent` context is reserved for future use — e.g., abnormal-but-not-critical lab results (K+ 5.2 — elevated but below 5.5 threshold) where the ingestion service wants priority processing without triggering the safety-critical dual-publish. Until implemented, `urgent` events behave identically to `critical` in the circuit breaker (always processed).

### 3.5 Dead Letter Queue (DLQ)

Failed outbox events (after max retries) move to `status=dead_letter` in the outbox table. The Global Outbox Service's DLQ dashboard provides visibility. Additionally, ingestion-specific DLQ topics exist for consumer-side failures:

| DLQ Topic | Source Topic | Purpose |
|-----------|-------------|---------|
| `dlq.ingestion.labs.v1` | `ingestion.labs` | Consumer processing failures (malformed payload, KB-20 insert failure) |
| `dlq.ingestion.vitals.v1` | `ingestion.vitals` | Consumer processing failures |
| `dlq.ingestion.safety-critical.v1` | `ingestion.safety-critical` | Critical value consumer failures (requires immediate investigation) |

DLQ entries are queryable via the existing ingestion service admin endpoints (`/admin/dlq/$count`, `/admin/dlq?status=PENDING`).

### 3.5 Latency Tiers

| Tier | Path | Latency | What |
|------|------|---------|------|
| Patient UX | HTTP handler → FHIR Store → outbox → HTTP 200 | ~295ms | Patient sees confirmation |
| Safety-critical | Outbox → publisher → `ingestion.safety-critical` → KB-22 (100ms poll) | ~2.5s | Physician alert for critical values |
| Analytical pipeline | Outbox → publisher → Kafka → Flink full pipeline → KB-20 | ~3-5s | Trend analysis, risk scoring, pattern detection |

### 3.6 Failure Modes

| Failure | Impact | Recovery |
|---------|--------|----------|
| FHIR Store down | HTTP handler returns 500 — no outbox write, no Kafka event | Caller retries. FHIR resource idempotent on retry. |
| PostgreSQL down | Outbox write fails — HTTP handler returns 500 | Caller retries. FHIR resource may be orphaned (see below). |
| Kafka down | Outbox events stay pending — Central Publisher retries with exponential backoff | Events accumulate in outbox table. Published when Kafka recovers. At-least-once guaranteed. |
| Central Publisher down | Events stay pending in outbox table indefinitely | Restart publisher. It resumes polling from `status=pending`. No data loss. |
| Flink down | Kafka events accumulate. Consumer lag grows. | Flink restarts from checkpoint. Replays from committed offsets. Dedup via `source_event_id`. |
| KB services down (all) | Flink uses stale cache (24h) then hardcoded defaults | Zero Flink failure. Thresholds degrade gracefully. |

**Orphaned FHIR resource scenario:** If FHIR Store write succeeds (Stage 3) but PostgreSQL transaction fails (Stage 4), an Observation exists in FHIR with no corresponding Kafka event. Downstream systems (KB-20, Flink) never learn about it. Mitigation strategy:
- **Primary:** Caller retries the full request. FHIR write is idempotent (same resource ID → upsert). The retry succeeds at Stage 4, publishing the outbox event. The orphan is resolved.
- **Lab webhooks (fire-and-forget):** Lab adapters (Thyrocare, Redcliffe) use webhook delivery with retry. If the ingestion service returns 500, the lab's webhook system retries (typically 3 attempts over 15 minutes). This covers the most likely fire-and-forget source.
- **Reconciliation (Phase 4):** A scheduled job queries FHIR Store for Observations created in the last 24h and cross-references against `outbox_events_ingestion_service` (published) and KB-20's `patient_observations`. Any FHIR Observation without a matching `source_event_id` in KB-20 is re-published to the outbox. This catches the rare case where both the original request and all retries failed at Stage 4. Not needed for MVP — the retry-based mitigation covers >99.9% of cases.

---

## 4. Flink Integration

### 4.1 Module1b — IngestionCanonicalizer (new, ~200 lines)

Reads from all `ingestion.*` topics. Transforms Global Outbox envelope to Flink's internal canonical format. Outputs to Module2 input topic (same junction as existing Module1).

```java
public class Module1b_IngestionCanonicalizer {
    // DEPLOYMENT: Separate Flink job (not co-deployed with Module1)
    // CONSUMER GROUP: "flink-module1b-ingestion" (independent from Module1's group)
    // PARALLELISM: 4 (matches highest ingestion topic partition count / 3)
    // CHECKPOINTING: 60 seconds (aligned with other Flink jobs)
    //
    // KafkaSource (multi-topic subscription):
    //   ingestion.labs, ingestion.vitals, ingestion.device-data,
    //   ingestion.patient-reported, ingestion.wearable-aggregates,
    //   ingestion.cgm-raw, ingestion.abdm-records, ingestion.medications,
    //   ingestion.observations, ingestion.safety-critical
    //
    // Transform: OutboxEnvelope.event_data → CanonicalEvent
    //   - Extract patient_id, observation_type, loinc_code, value, unit
    //   - Map source_type to Flink's SourceClassification enum
    //   - Preserve quality_score, flags, fhir_resource_id
    //   - Key by patient_id
    //
    // Output: Same topic/format as Module1 output → feeds into Module2
    //
    // FAILURE ISOLATION: Module1b crash does not affect Module1 (EHR pipeline).
    // Module1 continues processing prod.ehr.* topics independently.
}
```

### 4.2 CriticalAlertRouter Enhancement

Add `ingestion.safety-critical` as an additional input source to the existing CriticalAlertRouter:

```java
// Existing: reads from prod.ehr.events.enriched.routing
// Added: also reads from ingestion.safety-critical
// Routes to: prod.ehr.alerts.critical
```

### 4.3 Existing Flink Analytical Capabilities (no changes needed)

All of these process ingestion events automatically once Module1b feeds them into Module2:

| Capability | Module | Window | Output |
|-----------|--------|--------|--------|
| Creatinine trajectory + KDIGO AKI staging | LabTrendAnalyzer | 48h sliding, 1h slide | LabTrendAlert |
| Glucose variability (CV >36%) | LabTrendAnalyzer | 24h sliding, 1h slide | LabTrendAlert |
| Vital variability (HR/BP/RR/Temp/SpO2 CV) | VitalVariabilityAnalyzer | 4h sliding, 30m slide | VitalVariabilityAlert |
| NEWS2 early warning score | NEWS2Calculator | 4h tumbling | NEWS2Alert |
| MEWS early warning score | MEWSCalculator | 4h tumbling | MEWSAlert |
| Daily risk score (3-factor composite) | RiskScoreCalculator | 24h tumbling | DailyRiskScore |
| Sepsis pattern (CEP) | Module4 | 2-6h event-time | PatternEvent |
| Respiratory distress (CEP) | Module4 | event-time | PatternEvent |
| Cardiac event (CEP) | Module4 | event-time | PatternEvent |
| SIRS criteria (CEP) | Module4 | event-time | PatternEvent |
| Vital deterioration (CEP) | Module4 | event-time | PatternEvent |
| Medication adherence (CEP) | Module4 | 24-72h | PatternEvent |
| Alert prioritization (5D scoring, P0-P4) | AlertPrioritizer | per-alert | PrioritizedAlert |
| Alert dedup + composition (30-min suppression) | Module6 | per-alert state | ComposedAlert |
| Time-series aggregation | TimeSeriesAggregator | 1-min tumbling | VitalMetric |
| ML inference (sepsis, cardiac, respiratory) | Module5 | per-event | MLPrediction |

**SmartAlertGenerator suppression state:** Alert suppression in `SmartAlertGenerator.java` is currently **DISABLED** due to a cross-patient safety bug — static shared `alertHistory` caused Patient B's critical alerts to be suppressed when Patient A triggered the same alert type within the suppression window. All alerts fire unconditionally. With Module1b adding ingestion events to Flink, alert volume will increase. The per-patient fix (Flink `KeyedState`-based suppression) should be implemented before or alongside PR3. Until then, `Module6_AlertComposition` (which uses proper keyed state) provides the only dedup layer.

---

## 5. KB Threshold Centralization

### 5.1 Problem

Flink hardcodes 50+ clinical thresholds (vital ranges, lab critical values, KDIGO staging, NEWS2/MEWS scoring, risk weights, alert severity scores, high-risk medication list). KB services (KB-4, KB-20, KB-1, KB-23) own these thresholds but expose no query API that Flink can call. V-MCU Channel B independently hardcodes overlapping thresholds with subtle mismatches (K+ 5.5 vs 6.0).

### 5.2 Solution: BroadcastState + Async HTTP + Three-Tier Fallback

```
Flink Job Startup
  │ Blocking HTTP GET to KB-4, KB-20, KB-1, KB-23 (10s timeout)
  │ Load thresholds into Caffeine L1 cache (5-min TTL)
  │ If any KB service unreachable → use hardcoded defaults (zero regression)
  ▼
Runtime (continuous)
  │
  ├─ Every 5 minutes: ClinicalThresholdService refreshes from KB services
  │   └─ Update Caffeine cache → publish to BroadcastState
  │
  └─ Kafka topic "kb.clinical-thresholds.changes" (CDC from KB services)
      └─ BroadcastState<String, ClinicalThresholdSet> updates all operators
         └─ LabTrendAnalyzer, SmartAlertGenerator, NEWS2Calculator,
            MEWSCalculator, RiskScoreCalculator, AlertPrioritizer
            all read thresholds from BroadcastState
```

Three-tier fallback (zero-downtime guarantee):

| Tier | Source | TTL | When Used |
|------|--------|-----|-----------|
| 1 | Live KB service response (Caffeine L1) | 5 min | Normal operation |
| 2 | Stale cache (Caffeine L2) | 24 hours | KB service unreachable |
| 3 | Hardcoded defaults (compiled into JAR) | Infinite | KB down >24h, cold start |

### 5.3 KB Service → Flink Threshold Mapping

#### KB-4 (Patient Safety, port 8088)

**New endpoint: `GET /v1/thresholds/vitals`**

```json
{
  "heart_rate": {
    "bradycardia_severe": 40, "bradycardia_moderate": 50,
    "normal_low": 60, "normal_high": 100,
    "tachycardia_moderate": 110, "tachycardia_severe": 120
  },
  "systolic_bp": {
    "hypotension_severe": 70, "hypotension": 90,
    "normal_high": 140, "stage2_htn": 140, "crisis": 180
  },
  "diastolic_bp": {
    "normal_high": 90, "crisis": 120
  },
  "spo2": { "critical": 90, "low": 92, "normal_low": 95 },
  "respiratory_rate": {
    "critical_low": 8, "normal_low": 12, "normal_high": 20, "critical_high": 30
  },
  "temperature": {
    "hypothermia": 35.0, "normal_low": 36.1, "normal_high": 37.8, "high_fever": 39.5
  },
  "version": "2026-03-23T00:00:00Z"
}
```

**New endpoint: `GET /v1/thresholds/early-warning-scores`**

```json
{
  "news2": {
    "respiratory_rate": [
      {"min": 0, "max": 8, "points": 3},
      {"min": 9, "max": 11, "points": 1},
      {"min": 12, "max": 20, "points": 0},
      {"min": 21, "max": 24, "points": 2},
      {"min": 25, "max": 999, "points": 3}
    ],
    "spo2_scale1": [
      {"min": 0, "max": 91, "points": 3},
      {"min": 92, "max": 93, "points": 2},
      {"min": 94, "max": 95, "points": 1},
      {"min": 96, "max": 100, "points": 0}
    ],
    "spo2_scale2": [
      {"min": 0, "max": 92, "points": 3},
      {"min": 93, "max": 94, "points": 2},
      {"min": 95, "max": 96, "points": 1},
      {"min": 97, "max": 100, "points": 3}
    ],
    "systolic_bp": [
      {"min": 0, "max": 90, "points": 3},
      {"min": 91, "max": 100, "points": 2},
      {"min": 101, "max": 110, "points": 1},
      {"min": 111, "max": 219, "points": 0},
      {"min": 220, "max": 999, "points": 3}
    ],
    "heart_rate": [
      {"min": 0, "max": 40, "points": 3},
      {"min": 41, "max": 50, "points": 1},
      {"min": 51, "max": 90, "points": 0},
      {"min": 91, "max": 110, "points": 1},
      {"min": 111, "max": 130, "points": 2},
      {"min": 131, "max": 999, "points": 3}
    ],
    "temperature": [
      {"min": 0, "max": 35.0, "points": 3},
      {"min": 35.1, "max": 36.0, "points": 1},
      {"min": 36.1, "max": 38.0, "points": 0},
      {"min": 38.1, "max": 39.0, "points": 1},
      {"min": 39.1, "max": 99, "points": 2}
    ],
    "consciousness": {"alert": 0, "voice": 3, "pain": 3, "unresponsive": 3},
    "supplemental_o2": {"on_oxygen": 2},
    "thresholds": {"critical": 7, "high": 5, "low_medium_single3": true}
  },
  "mews": {
    "respiratory_rate": [
      {"min": 0, "max": 8, "points": 2},
      {"min": 9, "max": 14, "points": 0},
      {"min": 15, "max": 20, "points": 1},
      {"min": 21, "max": 29, "points": 2},
      {"min": 30, "max": 999, "points": 3}
    ],
    "heart_rate": [
      {"min": 0, "max": 40, "points": 2},
      {"min": 41, "max": 50, "points": 1},
      {"min": 51, "max": 100, "points": 0},
      {"min": 101, "max": 110, "points": 1},
      {"min": 111, "max": 129, "points": 2},
      {"min": 130, "max": 999, "points": 3}
    ],
    "systolic_bp": [
      {"min": 0, "max": 70, "points": 3},
      {"min": 71, "max": 80, "points": 2},
      {"min": 81, "max": 100, "points": 1},
      {"min": 101, "max": 199, "points": 0},
      {"min": 200, "max": 999, "points": 2}
    ],
    "temperature": [
      {"min": 0, "max": 35.0, "points": 2},
      {"min": 35.1, "max": 38.4, "points": 0},
      {"min": 38.5, "max": 99, "points": 2}
    ],
    "consciousness": {"alert": 0, "voice": 1, "pain": 2, "unresponsive": 3},
    "thresholds": {"critical": 5, "high": 3}
  },
  "version": "2026-03-23T00:00:00Z"
}
```

#### KB-20 (Patient Profile, port 8131)

**New endpoint: `GET /api/v1/thresholds/labs`**

Exposes the existing `lab_validator.go` plausibility ranges plus KDIGO/clinical thresholds:

```json
{
  "creatinine": {
    "plausible_range": [0.2, 20.0],
    "normal_range": [0.6, 1.2],
    "aki_stage1_delta_48h": 0.3,
    "aki_stage1_pct_increase": 50,
    "aki_stage2_multiplier": 2.0,
    "aki_stage3_multiplier": 3.0,
    "aki_stage3_absolute": 4.0,
    "worsening_slope": 0.1
  },
  "potassium": {
    "plausible_range": [1.5, 9.0],
    "normal_range": [3.5, 5.0],
    "alert_low": 3.0,
    "alert_high": 5.5,
    "halt_low": 3.0,
    "halt_high": 6.0
  },
  "glucose": {
    "plausible_range": [30, 600],
    "normal_fasting": [70, 100],
    "hypo": 70,
    "severe_hypo": 54,
    "severe_hyper": 300,
    "critical_high": 400,
    "cv_threshold": 36.0
  },
  "egfr": {
    "plausible_range": [0, 200],
    "halt": 15,
    "pause": 30,
    "ckd_stage3a": 45,
    "ckd_stage3b": 30
  },
  "hba1c": {
    "plausible_range": [3.0, 18.0],
    "normal_high": 5.7,
    "prediabetic": 6.5
  },
  "lactate": {
    "normal_high": 2.0,
    "critical": 4.0
  },
  "troponin": {
    "normal_high": 0.04,
    "critical": 0.5
  },
  "wbc": {
    "critical_low": 4.0,
    "critical_high": 15.0
  },
  "version": "2026-03-23T00:00:00Z"
}
```

**Threshold context distinction (K+ example):**
- `alert_high: 5.5` — lab critical value notification threshold (Flink alerts physician, ingestion service flags CRITICAL_VALUE)
- `halt_high: 6.0` — medication safety HALT threshold (V-MCU Channel B stops dose titration)
- Both from same KB-20 endpoint. Different consumers use different fields. Single source of truth.

#### KB-1 (Drug Rules, port 8081)

**New endpoint: `GET /v1/high-risk/categories`**

```json
{
  "high_risk_categories": [
    "anticoagulants", "insulin", "opioids",
    "chemotherapy", "antiarrhythmics"
  ],
  "high_risk_drugs": [
    {"rxnorm": "855332", "name": "warfarin", "category": "anticoagulants"},
    {"rxnorm": "311040", "name": "insulin_glargine", "category": "insulin"},
    {"rxnorm": "197696", "name": "morphine", "category": "opioids"}
  ],
  "version": "2026-03-23T00:00:00Z"
}
```

#### KB-23 (Decision Cards, port 8134)

**New endpoint: `GET /api/v1/config/risk-scoring`**

```json
{
  "daily_risk_weights": {
    "vital_stability": 0.40,
    "lab_abnormality": 0.35,
    "medication_complexity": 0.25
  },
  "risk_levels": [
    {"name": "LOW", "min": 0, "max": 24, "action": "routine monitoring"},
    {"name": "MODERATE", "min": 25, "max": 49, "action": "enhanced monitoring"},
    {"name": "HIGH", "min": 50, "max": 74, "action": "frequent assessment"},
    {"name": "CRITICAL", "min": 75, "max": 100, "action": "ICU-level monitoring"}
  ],
  "alert_severity_scores": {
    "CARDIAC_ARREST": 10, "RESPIRATORY_FAILURE": 10,
    "SEVERE_SEPTIC_SHOCK": 10,
    "SEPSIS_LIKELY": 9, "RESPIRATORY_DISTRESS": 9,
    "AKI_STAGE3": 8, "SEVERE_HYPOTENSION": 8, "SPO2_CRITICAL": 8,
    "VITAL_THRESHOLD_BREACH_HIGH": 7,
    "VITAL_THRESHOLD_BREACH_WARNING": 5,
    "MEDICATION_ALERT": 3
  },
  "time_sensitivity_scores": {
    "CARDIAC_ARREST": 5, "SEVERE_RESPIRATORY_DISTRESS": 5,
    "SEPSIS_LIKELY": 4, "NEWS2_GTE_7": 3,
    "SEPSIS_PATTERN": 3, "RESPIRATORY_DISTRESS": 3,
    "WARNING_SEVERITY": 2, "LAB_ABNORMALITY": 2,
    "MEDICATION_ALERT": 1
  },
  "patient_vulnerability": {
    "age_75_plus": 2, "age_65_plus": 1,
    "chronic_conditions_3_plus": 2, "chronic_conditions_1_plus": 1,
    "high_risk_conditions": ["diabetes", "heart_failure", "ckd"],
    "high_risk_condition_bonus": 1,
    "news2_gte_5_baseline": 1
  },
  "version": "2026-03-23T00:00:00Z"
}
```

### 5.4 ClinicalThresholdService (new Flink class)

```java
// Reuses existing patterns:
// - AsyncHttpClient (Netty-based, from GoogleFHIRClient used by AsyncPatientEnricher — NOT GoogleFHIRStoreSink which is synchronous)
// - Caffeine cache with L1 fresh + L2 stale (from GoogleFHIRClient)
// - Resilience4j circuit breaker (from AsyncPatientEnricher)
// - BroadcastState (from Module3_ComprehensiveCDS_WithCDC)

public class ClinicalThresholdService {

    // HTTP clients (non-blocking, async)
    private final AsyncHttpClient httpClient;

    // Caffeine caches
    private final Cache<String, ClinicalThresholdSet> freshCache;   // 5-min TTL
    private final Cache<String, ClinicalThresholdSet> staleCache;   // 24-hour TTL

    // Circuit breakers (per KB service)
    private final CircuitBreaker kb4CircuitBreaker;
    private final CircuitBreaker kb20CircuitBreaker;
    private final CircuitBreaker kb1CircuitBreaker;
    private final CircuitBreaker kb23CircuitBreaker;

    // Hardcoded defaults (Tier 3 fallback — same values as today)
    private static final ClinicalThresholdSet DEFAULTS = ClinicalThresholdSet.hardcodedDefaults();

    // Called at job startup (blocking, 10s timeout)
    public ClinicalThresholdSet loadInitialThresholds() { ... }

    // Called every 5 minutes (async, non-blocking)
    public CompletableFuture<ClinicalThresholdSet> refreshThresholds() { ... }

    // Three-tier resolution
    public ClinicalThresholdSet getThresholds() {
        ClinicalThresholdSet fresh = freshCache.getIfPresent("thresholds");
        if (fresh != null) return fresh;                        // Tier 1
        ClinicalThresholdSet stale = staleCache.getIfPresent("thresholds");
        if (stale != null) return stale;                        // Tier 2
        return DEFAULTS;                                         // Tier 3
    }
}
```

### 5.5 BroadcastState Wiring

```java
// Kafka source for threshold change events
KafkaSource<ClinicalThresholdSet> thresholdSource = KafkaSource
    .<ClinicalThresholdSet>builder()
    .setBootstrapServers(kafkaBrokers)
    .setTopics("kb.clinical-thresholds.changes")
    .setGroupId("flink-threshold-consumer")
    .setDeserializer(new ClinicalThresholdDeserializer())
    .build();

DataStream<ClinicalThresholdSet> thresholdStream = env.fromSource(
    thresholdSource, WatermarkStrategy.noWatermarks(), "ThresholdSource");

// Broadcast to all operators
MapStateDescriptor<String, ClinicalThresholdSet> THRESHOLD_STATE =
    new MapStateDescriptor<>("clinical-thresholds",
        Types.STRING, Types.POJO(ClinicalThresholdSet.class));

BroadcastStream<ClinicalThresholdSet> broadcastThresholds =
    thresholdStream.broadcast(THRESHOLD_STATE);

// Each analytics operator joins with broadcast
mainStream
    .keyBy(event -> event.getPatientId())
    .connect(broadcastThresholds)
    .process(new ThresholdAwareLabTrendAnalyzer());
```

### 5.6 V-MCU Channel B Alignment

After centralization, both Flink and V-MCU read from KB-20:

| Threshold | KB-20 Field | Flink Uses | V-MCU Channel B Uses |
|-----------|-------------|-----------|---------------------|
| K+ notification | `potassium.alert_high: 5.5` | SmartAlertGenerator, critical value flag | — |
| K+ dose HALT | `potassium.halt_high: 6.0` | — | Rule B-04 |
| Glucose hypo | `glucose.hypo: 70` | LabTrendAnalyzer | Rule B-01 (3.9 mmol/L = 70 mg/dL) |
| eGFR HALT | `egfr.halt: 15` | RiskScoreCalculator | Rule B-08 |
| eGFR PAUSE | `egfr.pause: 30` | RiskScoreCalculator | Rule B-09 |
| SBP hypotension | `systolic_bp.hypotension_severe: 70` | — | Rule B-05 (SBP <90) |
| SBP hypertensive crisis | `systolic_bp.crisis: 180` | SmartAlertGenerator | — |

Flink covers hypertension alerting. V-MCU covers hypotension dose safety. Complementary, not duplicative. Single KB-20 source for both.

---

## 6. New KB Endpoints Summary

| KB Service | Endpoint | Method | Purpose |
|-----------|----------|--------|---------|
| KB-4 (8088) | `/v1/thresholds/vitals` | GET | Vital sign thresholds for SmartAlertGenerator |
| KB-4 (8088) | `/v1/thresholds/early-warning-scores` | GET | NEWS2 + MEWS scoring parameters |
| KB-20 (8131) | `/api/v1/thresholds/labs` | GET | Lab thresholds (KDIGO, K+, glucose, eGFR, etc.) |
| KB-1 (8081) | `/v1/high-risk/categories` | GET | High-risk medication categories + drug list |
| KB-23 (8134) | `/api/v1/config/risk-scoring` | GET | Risk weights, level cutoffs, alert severity scores |

All endpoints: read-only, cacheable, versioned (`version` field for change detection).

**Threshold change propagation — two mechanisms:**
1. **Primary (MVP):** ClinicalThresholdService polls KB endpoints every 5 minutes. Compares `version` field. If changed, updates Caffeine cache and publishes to BroadcastState directly (in-process, no Kafka needed). This works immediately with the new GET endpoints — no additional KB service code required.
2. **Future enhancement:** KB services publish to `kb.clinical-thresholds.changes` Kafka topic when thresholds change (e.g., after admin updates a guideline). Flink consumes via BroadcastState for sub-second propagation. This requires adding a Kafka producer to each KB service's admin/update handlers — scoped as a follow-up after the core centralization is stable.

PR4-PR7 implement only the GET endpoints (mechanism 1). The Kafka publishing (mechanism 2) is a future PR.

---

## 7. Implementation Sequence

| PR | What | Dependencies | Parallelizable |
|----|------|-------------|---------------|
| **PR0** | Create all Kafka topics (`ingestion.*` + `kb.clinical-thresholds.changes`) with partition counts and retention policies | — | Yes |
| **PR1** | Ingestion service: Add Outbox SDK `ClientConfig`, replace `producer.go` with `SaveAndPublish()` / `SaveAndPublishBatch()`, keep FHIR writes in handlers, map router output to `ingestion.*` topics with `medical_context` | PR0 | No |
| **PR2** | Ingestion service: Fix wearable gap — wire Health Connect, Ultrahuman, Apple Health adapters through router → outbox → `ingestion.wearable-aggregates` | PR1 | No |
| **PR3** | Flink: Create Module1b IngestionCanonicalizer + add FHIRRouter skip condition + add CriticalAlertRouter `ingestion.safety-critical` subscription | PR1 | Yes (with PR2) |
| **PR4** | KB-4: Add `GET /v1/thresholds/vitals` + `GET /v1/thresholds/early-warning-scores` | — | Yes |
| **PR5** | KB-20: Add `GET /api/v1/thresholds/labs` (expose existing `lab_validator.go` ranges + KDIGO + halt/alert contexts) | — | Yes |
| **PR6** | KB-1: Add `GET /v1/high-risk/categories` | — | Yes |
| **PR7** | KB-23: Add `GET /api/v1/config/risk-scoring` | — | Yes |
| **PR8** | Flink: `ClinicalThresholdService` + BroadcastState wiring + refactor all hardcoded thresholds to read from BroadcastState with three-tier fallback | PR3, PR4-PR7 | No |

**Parallel execution plan:**

```
Phase 1 (parallel): PR0 + PR4 + PR5 + PR6 + PR7
Phase 2 (sequential): PR1 (depends on PR0)
Phase 3 (parallel): PR2 + PR3 (both depend on PR1; PR3 also parallelizable with PR2)
Phase 4 (sequential): PR8 (depends on PR3 + PR4-PR7)
```

---

## 8. Testing Strategy

### 8.1 Outbox Integration Tests

| Test | What It Verifies |
|------|-----------------|
| POST `/fhir/Observation` with routine glucose → check `outbox_events_ingestion_service` table has 1 row with `topic=ingestion.labs`, `medical_context=routine` | Single-publish routing |
| POST `/fhir/Observation` with K+ 6.8 → check outbox table has 2 rows (labs + safety-critical), both `medical_context=critical`, both `priority=1` | Dual-publish for critical values |
| POST `/devices` → check outbox row has `topic=ingestion.device-data`, `fhir_resource_id` is populated | Device adapter + FHIR write before outbox |
| POST `/wearables/ultrahuman` → check outbox row has `topic=ingestion.wearable-aggregates`, `medical_context=background` | Wearable gap fix verification |
| Kill Kafka → POST observation → verify HTTP 200, outbox row exists with `status=pending` → restart Kafka → verify row becomes `status=published` | Outbox retry guarantee |
| Verify `event_data` contains all canonical observation fields (patient_id, loinc, value, unit, quality_score, flags, fhir_resource_id) | Envelope completeness |

### 8.2 Flink Integration Tests

| Test | What It Verifies |
|------|-----------------|
| Publish outbox envelope to `ingestion.labs` → verify Module1b transforms to CanonicalEvent with correct fields | IngestionCanonicalizer |
| Publish observation with `source=ingestion-service`, `type=observation.created` → verify FHIRRouter does NOT write to FHIR | Skip condition |
| Publish critical K+ to `ingestion.safety-critical` → verify CriticalAlertRouter forwards to `prod.ehr.alerts.critical` | Critical alert routing |
| Publish 10 glucose readings for same patient → verify LabTrendAnalyzer computes CV and slope | Trend analysis with ingestion data |

### 8.3 KB Threshold Centralization Tests

| Test | What It Verifies |
|------|-----------------|
| Start Flink with all KB services up → verify thresholds loaded from KB services (not defaults) | Tier 1 loading |
| Start Flink with KB-4 down → verify stale cache used, then defaults after 24h | Tier 2 + Tier 3 fallback |
| Publish threshold change to `kb.clinical-thresholds.changes` → verify BroadcastState updates within 5s | Hot-swap |
| Change glucose CV threshold from 36% to 33% via KB-20 → verify LabTrendAnalyzer uses 33% on next window | End-to-end threshold propagation |
| Verify K+ alert_high=5.5 used by Flink SmartAlertGenerator, halt_high=6.0 used by V-MCU Channel B, both from KB-20 | Threshold context separation |

---

## 9. Migration Path

### Phase 1: Outbox Wiring (PR0-PR2)
- Ingestion service uses outbox SDK. Direct Kafka producer removed.
- FHIR Store writes remain synchronous.
- Existing ingestion Kafka consumers (if any) updated to read outbox envelope format.

### Phase 2: Flink Connection (PR3)
- Module1b reads `ingestion.*` topics. Ingestion events flow through full Flink pipeline.
- FHIRRouter skip condition prevents duplicate FHIR writes.
- CriticalAlertRouter subscribes to `ingestion.safety-critical`.

### Phase 3: KB Centralization (PR4-PR8)
- KB services expose threshold query endpoints.
- Flink refactored: hardcoded constants → BroadcastState reads.
- Three-tier fallback ensures zero regression.

### Phase 4: Future (not in scope)
- Debezium CDC on outbox table for sub-100ms latency (replaces 2s polling).
- V-MCU Channel B refactored to query KB-20 for thresholds (currently embedded constants).
- NEWS2/MEWS CQL libraries in Vaidshala clinical-knowledge-core (replaces KB-4 JSON).

---

## 10. Known Constraints

1. **2-second outbox polling latency** — acceptable for ingestion (lab results are minutes-to-hours old on arrival). Phase 4 Debezium CDC reduces to <100ms.
2. **Wearable data as "background" priority** — dropped first during circuit breaker OPEN. Clinically acceptable: CGM aggregates recomputed from full window on recovery.
3. **KB threshold refresh every 5 minutes** — clinical guidelines change at most quarterly. 5-min cache TTL is conservative.
4. **Module1b is a new Flink job** — adds one job to manage. But it's ~200 lines, stateless, and follows the proven Module1 pattern.
5. **No gRPC for threshold endpoints** — KB services expose HTTP REST only. Async HTTP client with Caffeine cache provides adequate performance (~5ms cached, ~50ms uncached).
6. **Outbox table auto-creation** — The SDK auto-creates the outbox table during client initialization (via the internal `createOutboxTable()` call inside `initialize()`), which runs CREATE TABLE IF NOT EXISTS. The ingestion service's PostgreSQL instance (port 5433, `ingestion_service` DB) must be accessible with DDL permissions for the `ingestion_user` role. For manual migrations, use the public `MigrationTool.CreateOutboxTable()` API. Verify before PR1.
7. **Idempotency on caller retry** — If a caller retries after a timeout (FHIR write succeeded but HTTP response was lost), the ingestion service generates a new `event_id`. To prevent duplicate outbox events, the FHIR `fhir_resource_id` should be used as a dedup key: `INSERT INTO outbox_events_ingestion_service ... ON CONFLICT ON CONSTRAINT uq_fhir_resource_id DO NOTHING`. Requires adding a unique partial index on `(event_data->>'fhir_resource_id') WHERE status != 'published'` to the outbox table.
8. **`kb.clinical-thresholds.changes` topic** — Created in PR0 but initially unused (no producer). ClinicalThresholdService uses 5-minute polling (mechanism 1) for MVP. The topic is available for future KB service CDC publishers (mechanism 2).
