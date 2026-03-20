# Clinical Signal Capture Layer — Design Specification

**Date**: 2026-03-20
**Status**: Draft
**Scope**: 22 signal types, 3 build phases, Kafka-native event mesh architecture
**Services affected**: KB-20, KB-21, KB-22, KB-23, KB-25, KB-26, KB-27 (new), V-MCU

---

## 1. Problem Statement

The Vaidshala Tier-1 observation pipeline currently handles 3 signal types (glucose, BP, weight). The clinical engines (KB-20 trajectories, KB-22 HPI, KB-26 digital twin, V-MCU titration) are built and waiting for a comprehensive signal feed. The Clinical Signal Capture Layer extends the pipeline to 22 signal types across 5 clinical domains, unifying structured and unstructured patient data collection through a Kafka-native event mesh.

### What exists today
- KB-20: Trajectory engines (eGFR, glucose, BP, ACR), transactional outbox with 38 event types, in-memory subscriber delivery only
- KB-21: Per-class adherence scoring, nudge engine, dietary signal model (carb category only)
- KB-22: Signal ingestion handlers (observation, checkin, twin-state), Bayesian HPI engine, Kafka publisher for HPI events
- KB-23: SignalCardBuilder mapping PM/MD signals to card templates
- KB-25: Neo4j causal chain graph (887 nodes, ~5230 edges), HTTP-only
- KB-26: 3 HTTP webhook endpoints for observation/checkin/med-change, MRI scoring
- Kafka: 3-broker Docker cluster with 68 pre-created topics, Schema Registry, used by KB-22/KB-24/KB-17

### What's missing
- Unified signal type registry (S1-S22) across all services
- Kafka bridging from KB-20 outbox to downstream consumers
- Kafka consumers in KB-26, KB-21, KB-25, KB-23
- Activity scoring, protein tracking, South Asian threshold alerting
- PREVENT/ASCVD risk score computation
- NLU extraction layer for patient free-text (Phase 3)
- Symptom lifecycle tracking (report → resolution)

---

## 2. Architecture: Kafka-Native Event Mesh

### Design decision
Every signal flows through Kafka as the single source of truth. Each KB service joins a consumer group for the topics it needs. This replaces the current HTTP webhook pattern as the primary transport (webhooks kept as dev/test fallback).

### Data flow

```
App/Device → HTTP → KB-20 (validate, persist, outbox)
                         │
                    OutboxRelay (1s poll)
                         │
                    ┌─────┴──────┐
                    │            │
        clinical.observations.v1  clinical.priority-events.v1
                    │            │
         ┌──────┬──┴──┬─────┐   ├──── KB-22 (HPI evaluation)
         │      │     │     │   ├──── KB-23 (priority cards)
       KB-22  KB-26 KB-21 KB-25 └──── KB-26 (twin state)
         │      │     │
         └──┬───┘     │
            │         │
       KB-23 cards  BEHAVIORAL_GAP alerts
       (via existing HTTP signal publisher)

Patient free-text → HTTP → KB-27 NLU (extract, confidence gate)
                              │
                         confidence ≥ 0.7?
                        yes/        \no
                       │              │
                  KB-20 signal    Return clarification
                  endpoint       question to app (buttons)
```

### Why Kafka-native over HTTP webhooks
- **Per-patient ordering**: Kafka partitions by `patient_id` — all signals for one patient land in the same partition, guaranteeing order
- **Independent consumption**: Each service processes at its own pace via consumer groups
- **Replay capability**: If KB-26 goes down, it catches up from Kafka offset on restart
- **Existing investment**: 3-broker cluster, Schema Registry, topic initializer already running in Docker
- **Proven pattern**: KB-22 and KB-24 already publish to Kafka via segmentio/kafka-go

---

## 3. Signal Type Registry

### Location
`backend/shared-infrastructure/knowledge-base-services/shared/signals/` — a shared Go module imported by all KB services.

### Signal types

| ID | Constant | Domain | Description | Priority auto-flag |
|----|----------|--------|-------------|-------------------|
| S1 | `SignalFBG` | Glycaemic | Fasting blood glucose | <4.0 mmol/L or >20.0 mmol/L |
| S2 | `SignalPPBG` | Glycaemic | Postprandial glucose | <4.0 mmol/L |
| S3 | `SignalHbA1c` | Glycaemic | Glycated haemoglobin | — |
| S4 | `SignalMealLog` | Glycaemic | Carb/protein/fat intake | — |
| S5 | `SignalGlucoseCV` | Glycaemic | Glucose variability (CV%) | CV >36% |
| S6 | `SignalHypoEvent` | Glycaemic | Hypoglycaemia event | Always priority |
| S7 | `SignalSBP` | Hemodynamic | Systolic BP | >180 or <90 mmHg |
| S8 | `SignalDBP` | Hemodynamic | Diastolic BP | — |
| S9 | `SignalHR` | Hemodynamic | Heart rate | <40 or >150 bpm |
| S10 | `SignalOrthostatic` | Hemodynamic | Orthostatic drop | Always priority |
| S11 | `SignalCreatinine` | Renal | Creatinine / eGFR | eGFR <15 |
| S12 | `SignalACR` | Renal | Albumin-creatinine ratio | >300 mg/mmol |
| S13 | `SignalPotassium` | Renal | Potassium | >5.5 or <3.0 mEq/L |
| S14 | `SignalWeight` | Metabolic | Body weight | — |
| S15 | `SignalWaist` | Metabolic | Waist circumference | — |
| S16 | `SignalActivity` | Metabolic | Steps / exercise | — |
| S17 | `SignalLipidPanel` | Metabolic | TC, HDL, LDL, TG | — |
| S18 | `SignalSymptom` | Patient-Reported | Symptom report | Severity=SEVERE |
| S19 | `SignalAdverseEvent` | Patient-Reported | Drug adverse event | Always priority |
| S20 | `SignalAdherence` | Patient-Reported | Medication adherence | — |
| S21 | `SignalResolution` | Patient-Reported | Symptom resolution follow-up | Status=WORSE |
| S22 | `SignalHospitalisation` | Patient-Reported | Hospitalisation event | Always priority |

### Canonical event envelope

```go
type ClinicalSignalEnvelope struct {
    EventID     uuid.UUID       `json:"event_id"`
    PatientID   string          `json:"patient_id"`
    SignalType  SignalType      `json:"signal_type"`
    Priority    bool            `json:"priority"`
    Timestamp   time.Time       `json:"measured_at"`
    Source      SignalSource    `json:"source"`       // APP_MANUAL, BLE_DEVICE, FHIR_SYNC, NLU_EXTRACTION
    Confidence  float64         `json:"confidence"`   // 1.0 for structured, 0.0-1.0 for NLU
    LOINCCode   string          `json:"loinc_code,omitempty"`
    Payload     json.RawMessage `json:"payload"`      // Signal-specific data
    CreatedAt   time.Time       `json:"created_at"`
}
```

### Topic routing

```go
func (e *ClinicalSignalEnvelope) KafkaTopic() string {
    if e.Priority {
        return "clinical.priority-events.v1"
    }
    return "clinical.observations.v1"
}
```

### Validation

Each signal type has a `SignalValidator` with:
- Plausibility ranges (reusing KB-20's existing ranges from `lab_validator.go`)
- New ranges for S4 (carbs 0-500g, protein 0-200g), S15 (waist 40-200cm), S16 (steps 0-50000, duration 0-480min)
- Return status: `ACCEPTED`, `FLAGGED`, or `REJECTED`
- South Asian thresholds for waist: men ≥90cm, women ≥80cm flagged as elevated

---

## 4. Kafka Topology

### Topics

| Topic | Partitions | Replication | Retention | Partition key | Purpose |
|-------|-----------|-------------|-----------|---------------|---------|
| `clinical.observations.v1` | 12 | 3 | 7 days | `patient_id` | Standard signal distribution |
| `clinical.priority-events.v1` | 6 | 3 | 30 days | `patient_id` | Adverse events, hypo, hospitalisation |

### Consumer groups

| Group ID | Service | Topics consumed |
|----------|---------|----------------|
| `kb22-signal-consumer` | KB-22 HPI Engine | observations, priority-events |
| `kb26-twin-consumer` | KB-26 Metabolic Digital Twin | observations, priority-events |
| `kb21-behavioral-consumer` | KB-21 Behavioral Intelligence | observations |
| `kb25-lifestyle-consumer` | KB-25 Lifestyle Knowledge Graph | observations |
| `kb23-priority-consumer` | KB-23 Decision Cards | priority-events only |

### Client library
Standardize on `segmentio/kafka-go` (already used by KB-22, KB-24). Writer config: `RequiredAcks=RequireAll`, `Compression=Snappy`, `BatchSize=100`, `BatchTimeout=10ms`.

---

## 5. KB-20 Outbox Relay

### Design
KB-20 already has a transactional outbox (`event_outbox` table) with a 1-second background poller. The relay bridges this to Kafka without changing existing data persistence logic.

### Components

```
KafkaOutboxRelay struct {
    db          *gorm.DB
    writer      *kafka.Writer       // segmentio/kafka-go
    eventMapper EventToSignalMapper // maps KB-20 EventType → ClinicalSignalEnvelope
    pollInterval time.Duration      // 1 second
    batchSize    int                // 50 events per poll
}
```

### Event type mapping

| KB-20 EventType | Signal Type | Priority |
|----------------|-------------|----------|
| `EventLabResult` (LabType=FBG) | `SignalFBG` | if value <4.0 mmol/L |
| `EventLabResult` (LabType=SBP) | `SignalSBP` | if value >180 |
| `EventLabResult` (LabType=DBP) | `SignalDBP` | no |
| `EventLabResult` (LabType=HBA1C) | `SignalHbA1c` | no |
| `EventLabResult` (LabType=CREATININE) | `SignalCreatinine` | if eGFR <15 |
| `EventLabResult` (LabType=ACR) | `SignalACR` | if value >300 |
| `EventLabResult` (LabType=POTASSIUM) | `SignalPotassium` | if >5.5 or <3.0 |
| `EventLabResult` (LabType=TOTAL_CHOLESTEROL) | `SignalLipidPanel` | no |
| `EventLabResult` (LabType=HDL) | `SignalLipidPanel` | no |
| `EventBPSevereAlert` | `SignalSBP` | yes |
| `EventBPUrgencyAlert` | `SignalSBP` | yes |
| `EventOrthostaticAlert` | `SignalOrthostatic` | yes |
| `EventGlucoseTrajectoryChange` | `SignalGlucoseCV` | if CV >36% |

### Idempotency
Each outbox row has a UUID → becomes `envelope.EventID`. Consumers deduplicate on `event_id` via idempotent upsert.

### Graceful degradation
If Kafka is unreachable, rows stay in `event_outbox` with `published_at = NULL`. The relay retries on next poll cycle. Existing in-memory subscribers continue independently (dual delivery during migration).

---

## 6. New KB-20 Signal Endpoints

For signals that don't come through the existing lab/vital path:

| Endpoint | Signal | Payload |
|----------|--------|---------|
| `POST /api/v1/patient/{id}/signals/meal` | S4 | carb_estimate, protein_g, fat_g, meal_type, regional_variant_id |
| `POST /api/v1/patient/{id}/signals/activity` | S16 | steps, exercise_type, duration_min, source (manual/device_sync) |
| `POST /api/v1/patient/{id}/signals/waist` | S15 | value_cm |
| `POST /api/v1/patient/{id}/signals/adherence` | S20 | drug_class, status (taken/missed), reason, timestamp |
| `POST /api/v1/patient/{id}/signals/symptom` | S18 | symptom, onset, severity, temporal_relation, frequency |
| `POST /api/v1/patient/{id}/signals/adverse-event` | S19 | symptom, suspected_drug, onset_relative, severity |
| `POST /api/v1/patient/{id}/signals/resolution` | S21 | original_event_id, status (RESOLVED/BETTER/SAME/WORSE) |
| `POST /api/v1/patient/{id}/signals/hospitalisation` | S22 | reason, admission_date, discharge_date, related_condition |

All endpoints: validate payload → persist to signal-specific table → write to `event_outbox` → relay publishes to Kafka.

---

## 7. Per-Service Consumer Wiring

### KB-26 (Metabolic Digital Twin) — Primary signal consumer

| Signal | Action |
|--------|--------|
| S1 FBG | Update `twin_state.fasting_glucose` (Tier 1) |
| S2 PPBG | Update `twin_state.postprandial_glucose` (Tier 1) |
| S3 HbA1c | Update `twin_state.hba1c` (Tier 1), re-derive Tier 2 |
| S7 SBP | Update `twin_state.sbp` (Tier 1) |
| S8 DBP | Update `twin_state.dbp` (Tier 1) |
| S9 HR | Update `twin_state.resting_hr` (Tier 1) |
| S11 Creatinine | Update `twin_state.creatinine` (Tier 1), derive eGFR (Tier 2) |
| S12 ACR | Update `twin_state.acr` (Tier 1) |
| S13 K+ | Update `twin_state.potassium` (Tier 1) |
| S14 Weight | Update `twin_state.weight_kg` (Tier 1), derive BMI (Tier 2) |
| S15 Waist | Update `twin_state.waist_cm` (Tier 1) |
| S16 Activity | Update `twin_state.daily_steps` (Tier 1) |
| S17 Lipids | Update `twin_state.tc/hdl/ldl/tg` (Tier 1) |
| S4 Meal Log | Update checkin state (diet quality) |
| S20 Adherence | Update `twin_state.compliance_score` (Tier 1) |
| S6 Hypo Event | Flag in twin state, trigger MRI recompute |
| S19 Adverse Event | Flag, pause affected drug simulation |
| S22 Hospitalisation | Flag, suspend MRI trending |

After any Tier 1 update: Tier 2 derivation → Tier 3 estimation → MRI recompute → publish MRI delta to KB-22/KB-23 via existing signal publisher.

### KB-22 (HPI Engine) — Clinical signal evaluator

| Signal | Action |
|--------|--------|
| S1-S17 | Route to existing `handleObservation()` — PM/MD node evaluation |
| S18 Symptom | Initiate HPI session for symptom cluster, run Bayesian differential |
| S19 Adverse Event | Adjust Bayesian prior via KB-24 CMApplicator ADR profile match. If HARD_BLOCK → immediate KB-23 card |
| S21 Resolution | Update differential. If drug-associated, track resolution timeline. If persistent >14 days → CDI +1 |
| S22 Hospitalisation | Suspend active HPI sessions, trigger clinical review escalation |

### KB-21 (Behavioral Intelligence) — Adherence & lifestyle consumer

| Signal | Action |
|--------|--------|
| S20 Adherence | `RecomputeAdherence()` per drug class. If <0.40 → publish `BEHAVIORAL_GAP` |
| S4 Meal Log | `recordDietarySignal()`. Compute protein intake vs M3-PRP target. If 7-day deficit → publish `PROTEIN_RESTORATION` |
| S16 Activity | New `ActivityScorer`: daily/weekly score, age-adjusted thresholds, M3-VFRP exercise adherence |
| S6 Hypo Event | HypoRiskService — update exercise risk factor |
| S14 Weight | Track weight trajectory for M3-VFRP |
| S15 Waist | South Asian threshold alerting (men ≥90cm, women ≥80cm) |

**New KB-21 components**:
- `ActivityScorer` service — age-adjusted thresholds: <4000 steps (age <65), <2500 (65-75), <1500 (>75)
- `ProteinTracker` — 7-day rolling protein intake vs target (0.8-1.2 g/kg)
- `BEHAVIORAL_GAP` alert publisher — fires at adherence <0.40

### KB-25 (Lifestyle Knowledge Graph) — Causal attribution consumer

| Signal | Action |
|--------|--------|
| S4 Meal Log | Match food items to Neo4j food nodes, traverse causal chains, compute attribution scores → POST to KB-26 `/calibrate` |
| S16 Activity | Match exercise to Neo4j exercise nodes, traverse causal chains → attribution scores |
| S14 Weight | Update body composition context |
| S15 Waist | Update metabolic risk context |

### KB-23 (Decision Cards) — Priority-only consumer

| Signal | Action |
|--------|--------|
| S6 Hypo Event | Generate HYPO card immediately |
| S19 Adverse Event | Generate ADR_REVIEW card. If HARD_BLOCK → PHYSICIAN_ALERT card |
| S22 Hospitalisation | Generate CLINICAL_REVIEW card, pause active protocol cards |

KB-23 only subscribes to `clinical.priority-events.v1`. For standard signals, it receives card requests from KB-22's existing signal publisher.

### V-MCU — Snapshot-based (not a Kafka consumer)

V-MCU continues to pull patient state from KB-20 snapshots. The change: KB-20 now includes KB-21 adherence score in the snapshot. V-MCU gain factor modulation: adherence <0.40 → suppress titration.

---

## 8. KB-27 Clinical NLU Service (Phase 3)

### Service identity

```yaml
name: kb-27-clinical-nlu
port: 8138
language: Go
database: kb27_clinical_nlu (PostgreSQL)
cache: Redis (rate limiting, session cache)
llm: Claude API (extraction only)
```

### Dual-path architecture

- **Path A (structured)**: App presents buttons/selectors → patient taps → structured payload goes directly to KB-20 signal endpoints. No NLU.
- **Path B (free-text fallback)**: Patient types text → KB-27 extracts entities → confidence gating → if confident, publishes structured envelope to KB-20 → same pipeline as Path A.

### Extraction contract

```go
type ExtractionRequest struct {
    PatientID   string `json:"patient_id"`
    FreeText    string `json:"free_text"`
    Language    string `json:"language"`
    SignalHint  string `json:"signal_hint"`         // symptom|adverse_event|resolution|hospitalisation
    ActiveMeds  []Med  `json:"active_medications"`  // from KB-20
}

type ExtractionResult struct {
    SignalType    SignalType        `json:"signal_type"`
    Entities      []ExtractedEntity `json:"entities"`
    MinConfidence float64           `json:"min_confidence"`
    NeedsClarity  bool             `json:"needs_clarification"`
    Clarification *ClarifyQuestion  `json:"clarification,omitempty"`
}

type ExtractedEntity struct {
    Name       string  `json:"name"`       // symptom, drug, onset, severity, temporal_relation
    Value      string  `json:"value"`
    Confidence float64 `json:"confidence"`
}
```

### Signal-specific extraction schemas

**S18 Symptom Report**: symptom, onset (duration), severity (MILD/MODERATE/SEVERE), temporal_relation (AFTER_MEDICATION/AFTER_STANDING/AT_REST/AFTER_EATING/OTHER), frequency (CONSTANT/INTERMITTENT/EPISODIC), suspected_trigger

**S19 Adverse Event**: symptom, suspected_drug (resolved against active_medications), onset_relative_to_drug, severity, temporal_relation (AFTER_START/AFTER_DOSE_CHANGE/AFTER_MISSED_DOSE)

**S21 Symptom Resolution**: original_symptom_event_id, status (RESOLVED/BETTER/SAME/WORSE), notes

**S22 Hospitalisation**: reason, admission_date, discharge_date, related_condition

### Confidence gating

- Threshold: 0.7 per entity
- If any entity below threshold → return `ClarifyQuestion` with multiple-choice options
- Clarification questions are **templated per language** (Hindi, English, regional), not LLM-generated
- Max 2 clarification rounds → `MANUAL_REVIEW_REQUIRED` fallback with clinical team notification
- Structured clarification answer merged directly with extraction (no second LLM call)

### Safety boundaries (hard constraints)

1. **Extraction only**: LLM system prompt explicitly forbids diagnosis, recommendation, interpretation
2. **Confidence gating**: No entity below 0.7 enters the clinical pipeline without structured clarification
3. **Never safety-critical**: KB-27 output goes to KB-20 (validation) → Kafka → KB-22 (Bayesian reasoning). NLU output never directly triggers Channel B safety rules, dose changes, or protocol actions
4. **Audit trail**: Every extraction logged — input text, LLM response, entities, confidence scores, clarification rounds, final output. PostgreSQL `extraction_logs` table
5. **Rate limiting**: Max 10 extractions per patient per hour (Redis-backed)

### Symptom resolution scheduler

Background job (in KB-22 or KB-20):
- 7 days and 14 days after S18/S19, push follow-up prompt via KB-21 nudge engine
- Patient response: structured buttons (RESOLVED/BETTER/SAME/WORSE)
- If "Worse" → auto-trigger new KB-22 HPI session for that symptom cluster

---

## 9. Build Phases

### Phase 1 — Core 5 Signals (S1 FBG, S7/S8 BP, S14 Weight, S3 HbA1c, S20 Adherence)

| Step | Description | Depends on |
|------|-------------|-----------|
| 1.1 | Create `shared/signals/` module — types, envelope, validators for Phase 1 signals | — |
| 1.2 | Add `clinical.observations.v1` and `clinical.priority-events.v1` to topics-config.yaml | — |
| 1.3 | KB-20 `KafkaOutboxRelay` — poll outbox → map to envelope → publish to Kafka | 1.1, 1.2 |
| 1.4 | KB-26 Kafka consumer — replace HTTP webhooks, route to `ProcessObservation()` | 1.1, 1.2 |
| 1.5 | KB-22 Kafka consumer — route to existing `handleObservation()` | 1.1, 1.2 |
| 1.6 | KB-21 Kafka consumer — S20 adherence only, route to `RecomputeAdherence()` | 1.1, 1.2 |
| 1.7 | Integration validation — end-to-end signal flow verification | 1.3-1.6 |

**Milestone**: FBG reading entered in KB-20 → Kafka → KB-26 twin update + KB-22 evaluation → KB-23 card if abnormal.

### Phase 2 — 12 Structured Signals (S2, S4-S6, S9-S13, S15-S17)

| Step | Description | Depends on |
|------|-------------|-----------|
| 2.1 | Expand signal registry — add S2, S4-S6, S9-S13, S15-S17 with validation schemas | Phase 1 |
| 2.2 | KB-20 new signal endpoints — meal, activity, waist | 2.1 |
| 2.3 | KB-21 new services — ActivityScorer, ProteinTracker, BEHAVIORAL_GAP alert | 2.1 |
| 2.4 | KB-26 consumer expansion — handle all 17 structured signals, PREVENT score | 2.1 |
| 2.5 | KB-25 Kafka consumer — meal/activity causal attribution | 2.1 |
| 2.6 | KB-23 priority card templates — HYPO, PROTEIN_RESTORATION, ORTHOSTATIC, HIGH_K | 2.1 |
| 2.7 | PREVENT score integration in KB-26 | 2.4 |
| 2.8 | V-MCU adherence wiring — gain factor modulation via KB-20 snapshot | 2.3 |

**Milestone**: All 17 structured signals flowing. Activity and meal logs drive KB-25 attribution. PREVENT recomputes on lab changes.

### Phase 3 — 5 Unstructured Signals (S18, S19, S21, S22 + NLU)

| Step | Description | Depends on |
|------|-------------|-----------|
| 3.1 | KB-27 service scaffold — Go service, PostgreSQL, Redis, Claude API client | Phase 2 |
| 3.2 | Extraction engine — `/api/v1/nlu/extract`, language detection, signal templates | 3.1 |
| 3.3 | Confidence gating & clarification loop — threshold 0.7, max 2 rounds | 3.2 |
| 3.4 | KB-20 structured Path A endpoints — symptom, adverse-event, resolution, hospitalisation | Phase 2 |
| 3.5 | KB-22 HPI session integration — S18/S19/S21/S22 consumer handlers | 3.4 |
| 3.6 | Symptom resolution scheduler — 7/14 day follow-up via KB-21 nudge | 3.5 |
| 3.7 | Audit & safety validation — verify NLU never triggers safety rules directly | 3.1-3.6 |

**Milestone**: Patient free-text → KB-27 extraction → confidence check → KB-20 → Kafka → KB-22 HPI session → KB-23 card.

---

## 10. Migration Strategy

No big-bang cutover. Each service runs in **dual mode** — HTTP webhooks AND Kafka consumer — controlled by feature flag:

```
Phase 1: kb26.kafka.enabled=true, kb22.kafka.enabled=true
Phase 2: kb25.kafka.enabled=true, kb21.kafka.expanded=true, kb23.kafka.enabled=true
Phase 3: HTTP webhooks deprecated (kept for dev/test only)
```

### Rollback
If a Kafka consumer has issues, flip the feature flag to disable it. The HTTP webhook path continues to work. The outbox relay can be paused without data loss (events accumulate in outbox table).

---

## 11. Non-Goals

- **Not a chatbot**: The NLU layer is extraction-only, not conversational AI
- **Not real-time streaming to V-MCU**: V-MCU continues to pull KB-20 snapshots, not consume Kafka directly
- **Not replacing the stream services pipeline**: The existing device-data Kafka pipeline (raw-device-data → validated-device-data) remains separate. This spec covers clinical observation signals, not raw device telemetry
- **Not adding FCM/APNs push delivery**: Push notification infrastructure is out of scope — KB-21's nudge engine logic exists but the delivery provider integration is separate work
