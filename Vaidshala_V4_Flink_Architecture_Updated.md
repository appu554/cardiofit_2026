# VAIDSHALA V4 — Flink Processing Architecture

## Updated: March 2026 | Post-E2E Validation | 11 Modules • 32 Kafka Topics

### Dual-Domain Processing + Enhancement Integration + Clinical Signal Capture

---

## 1. Architecture Principles (Validated by E2E Testing)

### 1.1 Seven Hard-Won Engineering Rules

These rules emerged from E2E testing of Modules 1–5 and are **non-negotiable** for all modules.

**Rule 1 — Validate after deserialization, not before.** Jackson with `FAIL_ON_UNKNOWN_PROPERTIES = false` silently produces objects with all-null fields. Every module entry point must check `patientId != null` and route nulls to DLQ. This bug silently dropped 100% of Module 3's input before it was caught.

**Rule 2 — Field names are contracts.** Production uses `heartrate` (lowercase-no-separator). Module 4 tests used `heart_rate` (snake_case). Module 2 internally used `HeartRate` (camelCase). Any module that reads vitals must normalize keys at entry through a shared alias map. The same applies to lab LOINC codes — multiple LOINC variants map to the same clinical concept.

**Rule 3 — Null lab values cause NPEs through auto-unboxing.** `LabResult.getValue()` can return `null` even when the `LabResult` object exists. Every numeric comparison must triple-null-check: `labs != null → labs.get(code) != null → getValue() != null`. This crashed Module 2 in production.

**Rule 4 — No synchronous external calls on the critical path.** FHIR lookups inside Flink operators caused cascading restart loops. Use `AsyncDataStream` with 2-second timeout, patient-level state cache (15-min TTL), and circuit breaker (5 failures → 60s open). When FHIR is down, proceed with `hasFhirData: false` — downstream modules degrade gracefully.

**Rule 5 — Lab-only emergencies are invisible to vitals-based scoring.** NEWS2 and qSOFA use zero lab values. AKI (creatinine 3.2, K+ 6.1) and anticoagulation crises (INR 6.0) produce NEWS2=0. Any logic that gates on NEWS2/qSOFA (cooldown, priority, alerting) must also check lab-derived risk indicators.

**Rule 6 — Risk indicators have two sources, and Module 3 can overwrite Module 2.** Module 2 sets medication-derived flags (`onAnticoagulation`, `onVasopressors`). Module 3's `PatientContextAggregator` rebuilds risk indicators from scratch, potentially overwriting them. Merge, don't replace. The `PatientContextAggregator` must also set vital-sign boolean indicators (`tachycardia`, `hypoxia`, `tachypnea`, `hypotension`, `fever`) from `latestVitals` — without these, the multi-component acuity score is suppressed (2.5 instead of 7.0+ for a NEWS2=10 patient).

**Rule 7 — Floating-point boundary comparisons need epsilon tolerance.** `0.4 + 0.3 + 0.1 = 0.7999999999999999` in IEEE 754. Clinical thresholds must use `>= threshold - 1e-9`, not exact `>=`. This affects CEP pattern matching, risk classification, and ML prediction thresholds.

### 1.2 Data Flow Invariant

```
Every event in the pipeline carries:
  patientId     — non-null, validated at Module 1 entry
  correlationId — assigned at Module 1, propagated through all modules
  eventTime     — source timestamp, monotonically increasing per patient
  processingTime — wall clock at each module (for latency measurement)
```

---

## 2. Complete Module Registry (11 Modules)

### 2.1 Core Pipeline — Modules 1–6 (E2E Validated)

| Module | Name | Operator Pattern | Status | Tests |
|--------|------|-----------------|--------|-------|
| **1/1b** | Ingestion & Gateway | `FlatMapFunction` | **E2E PASS** | 18 |
| **2** | Enhanced Context Assembly | `RichAsyncFunction` + `KeyedProcessFunction` | **E2E PASS** | — |
| **3** | Comprehensive CDS (8-Phase) | `KeyedBroadcastProcessFunction` | **E2E PASS (8/8)** | — |
| **4** | Pattern Detection (CEP + Analytics) | `ProcessFunction` + CEP + `WindowFunction` | **E2E PASS** | 96 |
| **5** | ML Inference Engine | `KeyedCoProcessFunction` (CDS + Patterns) | **Implementation** | 36 |
| **6** | Clinical Action & Distribution | `KeyedProcessFunction` + 6 async sinks | **Designed** | — |

### 2.2 Domain-Specific Modules — Modules 7–11

| Module | Name | Operator Pattern | Status | Kafka Output |
|--------|------|-----------------|--------|-------------|
| **7** | BP Variability Engine | Sliding window (7d + 30d) | **Partial** (Java on disk) | `flink.bp-variability-metrics` |
| **8** | Comorbidity Interaction Detector | Event-driven CEP | **Partial** (Java on disk) | `alerts.comorbidity-interactions` |
| **9** | Engagement Monitor | Rolling 14d window | **Not created** | `flink.engagement-signals` |
| **10/10b** | Meal Response Correlator + Aggregator | Session window (2h/4h) + 7d tumbling | **Not created** | `flink.meal-response`, `flink.meal-patterns` |
| **11** | Intervention Window Monitor | Event-driven timers | **Not created** | `clinical.intervention-window-signals` |

---

## 3. Module Specifications

### Module 1/1b — Ingestion & Gateway

**Input:** 15 Kafka topics (6 from Module 1 + 9 from Module 1b)
**Output:** `enriched-patient-events-v1` (keyed by `patientId`)
**DLQ:** `dlq.processing-errors.v1`

Processing steps: validation (timestamp sanity ±1h future/30d past), sanitization (lowercase/underscore normalization), canonicalization (numeric string parsing), data tier tagging (TIER_1_CGM / TIER_2_HYBRID / TIER_3_SMBG), deduplication (content hash), output routing.

**KB-7 Integration (Phase 3):** Add Step 6 — Clinical Concept Resolution. Before publishing to Kafka, resolve LOINC codes into canonical clinical concepts (`2160-0` → `CREATININE`, `30934-4` → `BNP`) using KB-7's LOINC alias map loaded at startup. This eliminates LOINC awareness from all downstream modules. Add `clinicalConcept`, `conceptGroup`, and `abnormalityFlag` fields to the canonical lab event.

### Module 2 — Enhanced Context Assembly

**Input:** `enriched-patient-events-v1`
**Output:** `enriched-patient-events-v1` (re-published with enrichment), `comprehensive-cds-events.v1` (via unified pipeline)
**Side outputs:** `simple-alerts.v1`, `protocol-triggers.v1`

Architecture: Two-phase enrichment — FHIR (demographics, medications, conditions) then Neo4j (cohort, graph relationships). Clinical scoring: NEWS2, qSOFA, metabolic acuity.

**Three protection layers (from E2E):**
1. Timeout + fallback: 2s FHIR timeout, proceed with `hasFhirData: false`
2. Patient-level FHIR cache: `FhirCacheFunction` with 15-min TTL in Flink keyed state
3. Circuit breaker: 5 consecutive failures → 60s open → half-open probe → auto-recover

**Critical fix applied:** `ClinicalIntelligenceEvaluator.java:175` — null-check on `wbc.getValue()` before auto-unboxing. Scan all `.getValue()` comparisons for same pattern.

**Serialization contract:** Output uses camelCase (`patientId`, `eventType`). Module 3 expects camelCase. The `createUnifiedPipeline` path must be used, NOT `createEnhancedPipeline` (which outputs snake_case and silently nullifies all downstream fields).

### Module 3 — Comprehensive CDS Engine (8-Phase)

**Input:** `enriched-patient-events-v1` (from Module 2, camelCase)
**Output:** `comprehensive-cds-events.v1`
**Broadcast inputs:** KB-3 protocols, KB-4 dosing, KB-5 interactions, KB-7 terminology (via CDC topics)
**State:** `ValueState<PatientContextState>` per patient, 7-day TTL

Operator: `KeyedBroadcastProcessFunction<String, EnrichedPatientContext, KBUpdateEvent, CDSEvent>`

8 Phases: Protocol matching → Clinical scoring → Diagnostics → Lab ordering → Medication analysis → Safety checking → Evidence composition → Alert generation.

**PatientContextAggregator fixes (from E2E):**
- Vital-sign boolean indicators: `tachycardia` (HR > 100), `hypotension` (SBP < 90), `tachypnea` (RR > 22), `hypoxia` (SpO2 < 92), `fever` (temp ≥ 38.3) — computed from `latestVitals` using multi-key lookup for naming variants
- Anticoagulation detection: `checkAnticoagulationRisk()` scanning active medications for warfarin, heparin, DOACs
- INR LOINC: checking both `34714-6` and `6301-6`
- BNP LOINC: `30934-4` added to `checkLabAbnormalities()`
- Risk indicator merging: medication-derived flags from Module 2 preserved, not overwritten

**Lab-based alerting (new):**
- `evaluateRenalRisk()`: AKI staging (KDIGO), hyperkalemia, nephrotoxic medication interaction
- `evaluateDrugLabRisk()`: supratherapeutic INR, bleeding risk (low Hgb + anticoagulation), thrombocytopenia
- Active alerts stored in `riskIndicators.activeAlerts` map

### Module 4 — Pattern Detection

**Input:** `comprehensive-cds-events.v1`
**Output:** `clinical-patterns.v1`
**State:** Pattern deduplication state per patient, 5-min window

Architecture: 3-layer detection system.
- Layer 1: Instant state assessment (every event)
- Layer 2: Flink CEP temporal patterns (6 patterns)
- Layer 3: Windowed analytics (TrendAnalysis, AnomalyDetection, ProtocolMonitoring)

**Clinical significance scoring:** `Module4ClinicalScoring` extracted for testability. Formula: `min(1.0, news2_weight + qsofa_weight + acuity/10 * 0.2)` where NEWS2 weights are stepped (0→0, 1-4→0.15, 5-6→0.35, 7-9→0.40, 10+→0.50) and qSOFA weights are (0→0, 1→0.15, 2+→0.30).

**CEP thresholds:** baseline (significance < 0.3) → warning (≥ 0.6) → critical (≥ 0.8). Floating-point boundary at critical: `0.7999999999999999` — use epsilon tolerance.

**Deduplication with severity escalation:** `PatternDeduplicationFunction` allows CRITICAL to pass through even within the 5-min dedup window if the previous pattern was HIGH. Tagged with `SEVERITY_ESCALATION`.

**Window functions use:** `TrendAnalysis` uses `apply()` (WindowFunction), `AnomalyDetection` uses `process()` (ProcessWindowFunction — pass `null` for Context in tests). `TrendAnalysis` uses index-based regression (not timestamp-based), so division-by-zero is impossible. `AnomalyDetection` requires ≥ 5 events. `ProtocolMonitoring` checks `guidelineRecommendations.size() > 0`, NOT admission→assessment→intervention sequences.

**Test suite:** 96 tests covering CDS→Semantic conversion (38), CEP patterns (17), post-CEP processing (16), windowed analytics (20), integration trajectories (5).

### Module 5 — ML Inference Engine

**Input 1 (primary):** `comprehensive-cds-events.v1` — triggers inference
**Input 2 (secondary):** `clinical-patterns.v1` — buffers into patient state
**Output:** `ml-predictions.v1`
**Side outputs:** `high-risk-predictions.v1`, `prediction-audit.v1`

Operator: `KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>`

**Why KeyedCoProcessFunction:** Module 4's 80→354 amplification means each CDS event generates ~4-5 pattern events. Running inference on every pattern event = 4-5x wasted compute. CDS events trigger inference; pattern events update state only. Exception: CRITICAL + SEVERITY_ESCALATION patterns bypass cooldown.

**Feature extraction:** 55-element float array.
- [0-5] Vital signs (normalized 0-1, production keys: `heartrate`, `systolicbloodpressure`)
- [6-8] Clinical scores (NEWS2, qSOFA, acuity)
- [9-10] Event context (log-scaled count, hours since admission)
- [11-30] Temporal ring buffers (NEWS2 history × 10, acuity history × 10)
- [31-34] Pattern features (count, deterioration count, max severity, escalation flag)
- [35-44] Risk indicator flags (10 booleans as 0/1, with safe extraction for unreliable flags)
- [45-49] Lab-derived features (normalized, -1.0 sentinel for missing)
- [50-54] Active alert features (binary presence + max severity)

**Lab-aware inference cooldown:**
- Stable (NEWS2 < 5): 30s cooldown
- Moderate (NEWS2 5-6 OR elevated lactate/leukocytosis): 10s cooldown
- High risk (NEWS2 ≥ 7 OR qSOFA ≥ 2 OR lab-critical): No cooldown
- Lab-critical: `hyperkalemia`, `severelyElevatedLactate`, `elevatedCreatinine`, `thrombocytopenia`

**Five prediction categories:** Readmission (30d), Sepsis (pre-SIRS), Deterioration (6-12h), Fall, Mortality. ONNX Runtime with 1 thread per model. Graceful degradation when models not loaded.

**Calibration:** Platt scaling per category. Sepsis thresholds intentionally lower (HIGH at 0.35 vs deterioration HIGH at 0.45) — false-negative sepsis is fatal.

**LOINC-to-name mapping in state update:** Production `recentLabs` is keyed by LOINC code. `updateStateFromCDS` must map `2524-7` → `lactate`, `2160-0` → `creatinine`, etc. before feature extraction.

### Module 6 — Clinical Action & Distribution Engine

**Inputs:** `comprehensive-cds-events.v1`, `clinical-patterns.v1`, `ml-predictions.v1`, `high-risk-predictions.v1`
**Outputs:** 6 external destinations + `clinical-alerts-realtime.v1` (WebSocket gateway)

Operator: `KeyedProcessFunction` with unified `ClinicalActionEvent` wrapper and 6 async sinks.

**Six destinations with independent circuit breakers:**

| Destination | Latency | Delivery | Failure Mode |
|-------------|---------|----------|-------------|
| FHIR Store | < 2s | At-least-once + idempotent PUT | Retry → DLQ |
| KB-14 Care Navigator | < 1s | At-least-once | Retry → direct WebSocket fallback |
| Elasticsearch | < 5s | At-least-once | Retry → skip |
| WebSocket/SSE | < 500ms | Best-effort | Drop (UI polls) |
| Neo4j Patient Graph | < 3s | At-least-once | Retry |
| Audit Store | < 5s | Exactly-once | **Block pipeline** |

**Priority classification:**
- STAT (< 30s): CRITICAL patterns, NEWS2 ≥ 10, qSOFA ≥ 2 → all 6 sinks
- URGENT (< 5min): HIGH patterns, NEWS2 ≥ 7, lab-critical → KB-14 + WebSocket + FHIR + ES
- ROUTINE (< 1h): NEWS2 5-6 → FHIR (batched) + ES + Neo4j
- SCHEDULED: LOW/MODERATE → ES + Neo4j only

**Lab-based priority override:** AKI (elevatedCreatinine + hyperkalemia) upgrades from SCHEDULED to URGENT. Drug-lab (elevatedINR + onAnticoagulation) upgrades to URGENT.

**Alert fatigue prevention:** 10 alerts/patient/hour cap. 5-minute STAT cooldown (same condition). STAT never suppressed even at cap.

**Action-level deduplication:** 10-minute window per patient + task type. Severity escalation passes through.

### Module 7 — BP Variability Engine

**Input:** `ingestion.vitals`, `ingestion.clinic-bp`
**Output:** `flink.bp-variability-metrics` (main), `ingestion.safety-critical` (crisis side-output)
**Windows:** 7d + 30d sliding

Computes: ARV (Average Real Variability), surge detection (SBP increase > 30mmHg in < 1hr), dipping classification (nocturnal BP reduction pattern), crisis bypass (SBP > 180 or DBP > 120 → immediate safety-critical output).

**Consumers:** KB-20 (state update), KB-26 (MHRI hemodynamic component), KB-23 (card generation), M4 RunCycle.

### Module 8 — Comorbidity Interaction Detector (CID)

**Input:** `ENRICHED_PATIENT_EVENTS` (meds + labs + vitals)
**Output:** `alerts.comorbidity-interactions` (PAUSE/SOFT_FLAG), `ingestion.safety-critical` (HALT side-output)

17 CID rules across 3 severity levels:
- 5 HALT rules (life-threatening): hypo masking (insulin/SU + beta-blocker + glucose < 60), euglycemic DKA (SGLT2i + nausea), etc.
- 5 PAUSE rules: GLP1RA + SU + FBG < 70, ACEi/ARB + K+ > 5.5, etc.
- 7 SOFT_FLAG rules: informational interactions

**Consumers:** KB-23 (safety cards), KB-24 (rule evaluation), notification-service (HALT alerts).

### Module 9 — Engagement Monitor

**Input:** All 6 signal types from `ingestion.*` topics
**Output:** `flink.engagement-signals` (daily scores), `alerts.engagement-drop` (threshold breach)
**Window:** Rolling 14d

6-signal composite scoring: medication adherence, BP measurement frequency, meal logging, appointment attendance, app usage, goal completion.

**Enhancement 6 integration — Early Relapse Prediction:**
Add 5 trajectory features (7-day rolling slopes): steps, meal quality, response latency, check-in completeness, protein adherence. Composite relapse risk score (weighted, initially expert-derived → data-derived after 200 patients). Output: `alerts.relapse-risk` (new topic).

BCE integration: MODERATE relapse risk → Recovery motivation phase, reduced targets, empathetic messaging. HIGH → KB-23 Decision Card + physician outreach.

**SPAN Micro-Engagement integration:**
Module 9 feeds SPAN timing signals — engagement composite informs the optimal micro-moment selection for BCE v2's SPAN delivery. SPAN (Specific, Proximal, Actionable, Nudge) micro-engagements are 5-15 second interactions delivered at behaviorally optimal moments identified by the RL timing bandit. Module 9 provides the engagement state that SPAN uses to determine intervention intensity:
- HIGH engagement → SPAN delivers growth-oriented micro-challenges
- MODERATE engagement → SPAN delivers maintenance nudges
- LOW engagement → SPAN delivers re-engagement micro-commitments (T-01)
- DECLINING trajectory → SPAN shifts to streak-protection and empathetic framing

### Module 10/10b — Meal Response Correlator + Aggregator

**Module 10 Input:** `ingestion.vitals`, `ingestion.patient-reported`, `ingestion.labs`
**Module 10 Output:** `flink.meal-response` (real-time pairs)
**Windows:** 2h (glucose-meal session), 4h (sodium-BP session)

**Module 10b Input:** `flink.meal-response`
**Module 10b Output:** `flink.meal-patterns` (weekly summary)
**Window:** 7d tumbling

Creates glucose-meal pairs (stimulus: carbs, response: glucose delta, peak, latency) and sodium-BP pairs. Weekly aggregation: worst/best foods, average daily carb impact, sodium sensitivity estimation.

**Consumers:** KB-26 (twin calibration), KB-25 (lifestyle graph food→outcome edges), KB-21 (dietary patterns), phenotype-clustering (salt_sensitivity_beta).

### Module 11 — Intervention Window Monitor

**Input:** `clinical.intervention-events` (from KB-23 physician APPROVE/MODIFY)
**Output:** `clinical.intervention-window-signals` (WINDOW_OPENED / WINDOW_CLOSED / WINDOW_MIDPOINT / EXPIRED)
**Timers:** 14d (lifestyle), 28d (medication)

On physician approval of intervention → opens observation window → emits MIDPOINT for coaching nudge → on CLOSED, triggers IOR generator to assemble OutcomeRecord (baseline vs current MHRI/HbA1c/SBP/eGFR deltas).

**Consumers:** IOR generator (batch), KB-20 (active_interventions[]), coaching engine (midpoint nudge).

---

## 4. Enhancement Integration Map

### 4.1 Enhancement → Module Impact Matrix

| Enhancement | Module 2 | Module 3 | Module 5 | Module 6 | Module 9 | New Topics |
|-------------|----------|----------|----------|----------|----------|------------|
| E1: Sleep Quality | — | KB-26 coupling via CDS | Sleep features in ML | Alert routing | Sleep signal tracking | — |
| E2: Mental Health (PHQ-2) | — | Safety gate | Distress attenuation factor | Decision Card routing | — | — |
| E3: Metabolic Mapping Phase | — | Protocol selection input | Profile-specific features | MAP Decision Card | — | — |
| E4: Patient-Facing Visualization | — | — | — | Tier-1 message formatting | — | — |
| E5: Complication Risk Scoring | — | — | Risk score features | Risk Decision Cards | — | — |
| E6: Relapse Prediction | — | — | Relapse risk features | Pre-emptive intervention | **Core implementation** | `alerts.relapse-risk` |
| E7: Caregiver Channel | — | — | — | Message routing (A/B/C classification) | Caregiver engagement | — |
| SPAN Micro-Engagement | — | — | — | Real-time delivery | **Timing signals** | — |

### 4.2 Enhancement 1 — Sleep Quality (KB-26 Extension)

**Data flow:** Tier-1 check-in (biweekly) → KB-20 state → KB-26 twin state (`SleepDuration7dMean`, `SleepQualityScore`, `SleepAdequacy`, `SleepTrend`).

**Coupling equations (KB-26 simulation):**
- Sleep → Insulin Sensitivity: coefficient -0.04/unit decline
- Sleep → Visceral Fat: coefficient +0.015/unit decline
- Sleep → Vascular Resistance: coefficient +0.02/unit decline
- Sleep → Hepatic Glucose Output: coefficient +0.02/unit decline

**Module 5 impact:** Add sleep adequacy as a feature in the base vector. When available (Tier 2 derived), use as input alongside vitals and labs for deterioration and readmission predictions.

**Safety rule LS-16:** Sleep < 4hr for ≥ 3 consecutive check-ins → physician review flag.

### 4.3 Enhancement 2 — Mental Health Screening (PHQ-2)

**Data flow:** 4-weekly PHQ-2 in Tier-1 check-in → KB-20 (`PHQ2Score`, `MentalHealthFlag`) → KB-26 (`PsychologicalDistress`: LOW/MODERATE/HIGH).

**Twin state modulation:** Attenuation factor on intervention effects — 0.7x for HIGH distress, 0.85x for MODERATE. Cortisol pathway coupling to IS and VF (shared mechanism with sleep).

**Module 5 impact:** `PsychologicalDistress` level as a categorical feature. Modulates prediction confidence — HIGH distress patients have less predictable intervention response.

**Module 6 impact:** PHYSICIAN_REVIEW_REQUIRED (PHQ-2 score 6) → KB-23 Decision Card via URGENT routing. BCE shifts to supportive-only mode.

**Safety boundary:** Platform screens, flags, adapts, and refers. It NEVER treats depression.

### 4.4 Enhancement 3 — Metabolic Mapping Phase (M3-MAP)

**Protocol phase:** Phase -1, 14-21 days pre-protocol observation.

**Mechanism profiles:**
1. IS Deficit Dominant → PRP + VFRP (40-50% of patients)
2. HGO Dominant → VFRP + Metformin recommendation (skip PRP Phase 1)
3. Beta Cell Deficit → Medication-first (lifestyle as adjunct)
4. Mixed → PRP + VFRP with early Day 42 review

**Module 5 impact:** Mechanism profile as a categorical feature. Profile-specific prediction models (an HGO-dominant patient's deterioration trajectory differs from an IS-deficit patient).

### 4.5 Enhancement 4 — Patient-Facing Simulation Visualization

**Data flow:** KB-26 `POST /simulate-comparison` → Tier-1 message formatting → WhatsApp delivery.

Delivered at Day 28 and Day 63. Three scenarios: current trajectory, single best intervention, full program. Uses KB-25 attribution for personalized result explanation.

**Safety boundary:** NEVER show complication risk probabilities directly to patients. Tier 1 and 2 variables only.

### 4.6 Enhancement 5 — Complication Risk Scoring

**Models:** UKPDS v2 (CVD), KFRE (renal), Framingham/WHO-ISH (ASCVD).

**KB-26 API:** `GET /twin/{patientId}/risk-profile` → 5y/10y CVD risk, 2y/5y renal risk, composite 0-100, `risk_delta` since baseline.

**Module 5 impact:** Risk scores as features for deterioration and mortality prediction models.

**Module 6 impact:** Risk scores on Day 28 and Day 63 Decision Cards. Risk-reduction framing for escalation decisions.

**Market shim:** India (UKPDS + KFRE + WHO/ISH), Australia (Absolute CVD Risk + KFRE + Framingham).

### 4.7 Enhancement 6 — Early Relapse Prediction (Module 9 Extension)

**Five trajectory features:** steps slope, meal quality slope, response latency slope, check-in completeness slope, protein adherence slope (all 7-day rolling).

**Composite score:** Expert weights initially (steps 0.30, latency 0.25, meal 0.20, check-in 0.15, protein 0.10) → data-derived after 200 patients.

**New topic:** `alerts.relapse-risk` (partitions: 4, retention: 90d).

**BCE integration:** MODERATE → Recovery phase, reduced targets, empathetic messaging. HIGH → KB-23 Decision Card + outreach call.

### 4.8 Enhancement 7 — Caregiver Communication Channel

**Message classification:**
- Category A (shareable): achievements, streaks, weekly summaries → caregiver
- Category B (patient-only): check-ins, lab results, medication reminders → patient only
- Category C (action-oriented): dietary preparation guidance, family activities → caregiver

**Consent:** Optional, revocable, stored in KB-20 with timestamp and history.

**Design rules:** Never surveillance language. Max 2-3 messages/week. Include in celebration, never in failure. Dietary prescription translation (KB-25 → regional cooking guidance).

### 4.9 SPAN Micro-Engagement (BCE v2 Addition)

**Definition:** SPAN = Specific, Proximal, Actionable, Nudge. 5-15 second micro-interactions delivered at behaviorally optimal moments.

**Architecture:** SPAN is NOT a new module. It's a BCE v2 delivery mode that consumes:
- Module 9 engagement signals (engagement state → intervention intensity)
- KB-21 behavioral phenotype (phenotype → technique selection)
- RL timing bandit (optimal delivery window per patient)

**SPAN delivery triggers:**
- Post-meal window (15-30 min after meal log): "Quick protein check — did your lunch include dal or egg?"
- Pre-activity window (based on learned activity pattern): "Perfect time for a 10-minute walk"
- Morning context window: "Start your day with a glass of water before tea"
- Streak maintenance: "Day 15 of walking! Just 10 minutes keeps it going"

**Module 9 feeds SPAN:**
- Engagement composite score determines SPAN frequency (HIGH engagement → 1-2/day, LOW → max 1 every 2 days)
- Response latency trend determines SPAN timing (shift delivery window if patient responds faster at different times)
- Relapse risk score determines SPAN intensity (pre-relapse → streak protection, micro-commitments)

**Fatigue prevention:** SPAN messages count toward the daily message cap. Never deliver SPAN and a standard coaching message within 2 hours. SPAN pauses automatically during LOW engagement periods lasting > 3 days (avoid compounding disengagement).

---

## 5. Kafka Topic Registry (32 Topics)

### 5.1 Input Topics (15)

| Topic | Source | Module Consumer |
|-------|--------|----------------|
| `vital-signs-events-v1` | EHR/devices | Module 1 |
| `lab-result-events-v1` | Lab systems | Module 1 |
| `ingestion.labs` | Ingestion service | Module 1b |
| `ingestion.vitals` | Ingestion service | Module 1b |
| `ingestion.device-data` | Ingestion service | Module 1b |
| `ingestion.patient-reported` | Ingestion service | Module 1b |
| `ingestion.wearable-aggregates` | CGM Aggregation | Module 1b |
| `ingestion.cgm-raw` | BLE relay | Module 1b |
| `ingestion.abdm-records` | ABDM integration | Module 1b |
| `ingestion.medications` | Ingestion service | Module 1b |
| `ingestion.observations` | Ingestion service | Module 1b |
| `ingestion.clinic-bp` | Clinic systems | Module 7 |
| `intake.checkin-events` | Tier-1 check-in | Trajectory Analysis |
| `intake.safety-alerts` | Intake service | Deterioration Detection |
| `clinical.intervention-events` | KB-23 physician actions | Module 11 |

### 5.2 Inter-Module Topics (10)

| Topic | Producer | Consumer |
|-------|----------|----------|
| `enriched-patient-events-v1` | Module 1/1b, Module 2 | Module 2, Module 3, Module 8 |
| `comprehensive-cds-events.v1` | Module 3 | Module 4, Module 5, Module 6 |
| `clinical-patterns.v1` | Module 4 | Module 5, Module 6 |
| `ml-predictions.v1` | Module 5 | Module 6 |
| `high-risk-predictions.v1` | Module 5 (side output) | Module 6 |
| `prediction-audit.v1` | Module 5 (side output) | Audit store |
| `flink.meal-response` | Module 10 | Module 10b, KB-26, KB-25 |
| `flink.meal-patterns` | Module 10b | KB-21, KB-26, phenotype-clustering |
| `flink.trajectory-signals` | Trajectory Analysis | M4 RunCycle |
| `flink.mri-triggers` | MRI Trigger | KB-26 |

### 5.3 Alert & Action Topics (5)

| Topic | Producer | Consumer |
|-------|----------|----------|
| `ingestion.safety-critical` | Module 7 (crisis), Module 8 (HALT) | notification-service, KB-23 |
| `alerts.comorbidity-interactions` | Module 8 | KB-23, KB-24, notification-service |
| `alerts.engagement-drop` | Module 9 | notification-service, KB-21 |
| `alerts.relapse-risk` | Module 9 (Enhancement 6) | KB-23, BCE |
| `clinical-alerts-realtime.v1` | Module 6 | WebSocket gateway |

### 5.4 State & Analytics Topics (2)

| Topic | Producer | Consumer |
|-------|----------|----------|
| `flink.engagement-signals` | Module 9 | KB-20, KB-26, KB-21 |
| `flink.bp-variability-metrics` | Module 7 | KB-20, KB-26, KB-23, M4 RunCycle |
| `clinical.intervention-window-signals` | Module 11 | IOR generator, KB-20, coaching engine |
| `clinical-audit.v1` | Module 6 | Audit database |
| `clinical.decision-cards` | KB-23 | API gateway, notification-service |

---

## 6. Cross-Cutting Architecture

### 6.1 Serialization Contract

All inter-module Kafka messages use Jackson JSON with camelCase field names. `FAIL_ON_UNKNOWN_PROPERTIES = false` is set globally — making null-field validation at entry points mandatory.

Lab payloads must use flat structure: `{"labName": "creatinine", "value": 3.2, "unit": "mg/dL"}` — NOT nested `{"results": {"creatinine": 3.2}}`.

### 6.2 State TTL Strategy

| Module | State Type | TTL | Rationale |
|--------|-----------|-----|-----------|
| Module 3 | PatientContextState | 7 days | Acute/subacute monitoring |
| Module 4 | Dedup state | 5 minutes | Pattern suppression window |
| Module 5 | PatientMLState | 7 days | Temporal features (ring buffers) |
| Module 6 | PatientActionState | 24 hours | Action dedup + batching |
| Module 7 | BP readings | 30 days | Monthly variability calculation |
| Module 9 | Engagement signals | 14 days | Rolling engagement window |

All use `OnReadAndWrite` + `NeverReturnExpired`. First event after state expiry produces `contextDepth: INITIAL` metadata.

### 6.3 Checkpointing

| Module | Interval | Backend | Retained |
|--------|---------|---------|----------|
| Module 1/1b | 60s | RocksDB | 3 |
| Module 2 | 30s | RocksDB | 3 |
| Module 3 | 15s | RocksDB | 3 |
| Module 4 | 10s | RocksDB | 3 |
| Module 5 | 15s | RocksDB | 3 |
| Module 6 | 30s | RocksDB | 3 |
| Modules 7-11 | 30s | RocksDB | 3 |

Set `state.checkpoints.num-retained: 3` globally. Configure log rotation (100MB max, 5 files) to prevent disk exhaustion during restart loops.

### 6.4 Consumer Group Naming

Pattern: `vaidshala-v4-{module-name}-{function}`

Examples:
- `vaidshala-v4-module1-ingestion`
- `vaidshala-v4-module3-cds-engine`
- `vaidshala-v4-module5-ml-inference`
- `vaidshala-v4-module6-action-distribution`

Never share consumer groups between modules. Module 5's dual-input must use a single consumer group for the `connect()` operator — Flink manages this internally.

### 6.5 Metrics Standard

Every module emits:
- `events_in` — counter, by source topic
- `events_out` — counter, by output topic/sink
- `events_rejected` — counter, by error type
- `processing_latency_ms` — histogram, p50/p95/p99
- `state_size_bytes` — gauge (RocksDB)
- `error_rate` — counter
- Module-specific gauges (models_loaded, circuit_breaker_state, etc.)

---

## 7. Implementation Priority

### Phase 1 — Complete (Modules 1–4 E2E validated)
- All core pipeline modules built and tested
- 8/8 clinical scenarios passing
- 96 Module 4 tests + supplementary tests

### Phase 2 — In Progress (Module 5)
- KeyedCoProcessFunction with lab-aware cooldown
- 36 new tests covering all 5 E2E gaps
- ONNX inference with graceful degradation

### Phase 3 — Next (Module 6 + Modules 7-8 verification)
- Module 6: Clinical Action & Distribution Engine
- Module 7: Verify BP Variability Engine completeness
- Module 8: Verify 17 CID rules and tests

### Phase 4 — Build (Modules 9-11)
- Module 9: Engagement Monitor + Enhancement 6 (relapse prediction) + SPAN signals
- Module 10/10b: Meal Response Correlator + Aggregator
- Module 11: Intervention Window Monitor

### Phase 5 — Enhancement Integration
- Phase A (weeks 1-3): Enhancement 1 (Sleep) + Enhancement 2 (Mental Health) — shared cortisol pathway
- Phase B (weeks 3-6): Enhancement 5 (Complication Risk) + Enhancement 6 (Relapse) — independent, parallelizable
- Phase C (weeks 6-9): Enhancement 3 (Metabolic Mapping Phase) — depends on KB-26
- Phase D (weeks 9-12): Enhancement 4 (Patient Visualization) + Enhancement 7 (Caregiver) + SPAN — presentation layer

---

*Document version: 2.0 | Updated: 31 March 2026 | Post-E2E validation of Modules 1-4, Module 5 in implementation*
