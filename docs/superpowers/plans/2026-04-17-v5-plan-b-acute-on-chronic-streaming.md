# V5 Plan B: Acute-on-Chronic Streaming Detection (Gaps 16+22)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Flink streaming detector that catches acute deterioration against stable baselines in near-real-time — eGFR acute drops, glucose crises, BP crises, CGM TBR spikes, HF weight gain, and measurement cessation — and publishes SAFETY/IMMEDIATE escalation events within minutes of the triggering data point.

**Architecture:** A new Flink operator (`AcuteOnChronicDetector`) maintains per-patient keyed state: 30-day rolling baselines for eGFR, glucose, BP, weight, CGM TBR, and measurement frequency. On every new reading, it compares against the baseline and fires an alert event when deviation exceeds clinically significant thresholds. Alert events flow to KB-14 via Kafka → KB-23 card generation → KB-14 task creation. This is the "event-driven compute layer" (Gap 22) — continuous stateful stream processing that fires on every data point.

**Tech Stack:** Java 17 + Flink 2.1 for streaming operators. Kafka for input (clinical data) and output (alert events). Go for KB-23 alert card templates and KB-14 task creation. Market-config YAML for threshold definitions.

---

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| Module3_CGMStreamJob (14d sliding window) | Flink `operators/` | Pattern for keyed state + event-time windows |
| Module7 BP variability (if exists) | Flink `operators/` | Pattern for per-patient vital sign streaming |
| Kafka topic `clinical.priority-events.v1` | KB-23 consumer | Alert events can publish here for card generation |
| PrioritySignalHandler | KB-23 `internal/services/` | Already handles reactive eGFR events — acute-on-chronic events follow same pattern |
| SafetyEvent audit trail | KB-20 `internal/models/` | Acute events should be recorded here |

## File Inventory

### Flink (Stream Processing) — Acute-on-Chronic Operator
| Action | File | Responsibility |
|---|---|---|
| Create | `flink-processing/.../operators/AcuteOnChronicDetector.java` | Main Flink operator: per-patient keyed state, rolling baselines, deviation detection |
| Create | `flink-processing/.../operators/AcuteOnChronicAlert.java` | Alert event POJO (patientID, alertType, currentValue, baselineValue, deviationPct, severity) |
| Create | `flink-processing/.../operators/PatientBaseline.java` | Per-patient rolling baseline state (30-day medians for each metric) |
| Create | `flink-processing/.../operators/AcuteThresholds.java` | Threshold definitions for each metric type |
| Create | `flink-processing/src/test/.../AcuteOnChronicDetectorTest.java` | 8 tests |

### KB-23 (Decision Cards) — Acute Alert Cards
| Action | File | Responsibility |
|---|---|---|
| Create | `kb-23-decision-cards/templates/acute/egfr_acute_drop.yaml` | SAFETY card: eGFR acute drop >25% |
| Create | `kb-23-decision-cards/templates/acute/glucose_crisis.yaml` | SAFETY card: BG >400 or <40 |
| Create | `kb-23-decision-cards/templates/acute/bp_crisis.yaml` | SAFETY card: SBP >180 or DBP >120 |
| Create | `kb-23-decision-cards/templates/acute/cgm_tbr_spike.yaml` | IMMEDIATE card: TBR L2 >5% acute |
| Create | `kb-23-decision-cards/templates/acute/hf_weight_gain.yaml` | URGENT card: >2kg/72h for HF patients |
| Create | `kb-23-decision-cards/templates/acute/measurement_cessation.yaml` | URGENT card: daily measurer stopped >72h |
| Create | `kb-23-decision-cards/internal/services/acute_alert_handler.go` | Kafka consumer for acute alert events |
| Create | `kb-23-decision-cards/internal/services/acute_alert_handler_test.go` | 6 tests |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/acute_thresholds.yaml` | Threshold definitions for each acute metric |

**Total: 16 files (16 create)**

---

### Task 1: Acute threshold config + alert models

**Files:**
- Create: `market-configs/shared/acute_thresholds.yaml`
- Create: `flink-processing/.../operators/AcuteOnChronicAlert.java`
- Create: `flink-processing/.../operators/PatientBaseline.java`
- Create: `flink-processing/.../operators/AcuteThresholds.java`

- [ ] **Step 1:** Create `acute_thresholds.yaml`:
```yaml
# Acute-on-chronic deterioration thresholds.
# Each metric: baseline_window_days, deviation_threshold, severity, description.

metrics:
  egfr_acute_drop:
    baseline_window_days: 30
    deviation_pct: 25          # >25% decline from 30-day median
    severity: SAFETY
    description: "Possible AKI — eGFR dropped >25% from baseline"
    
  glucose_high_crisis:
    absolute_threshold: 400    # mg/dL
    severity: SAFETY
    description: "Glucose crisis >400 — possible DKA or HHS"
    
  glucose_low_crisis:
    absolute_threshold: 40     # mg/dL
    severity: SAFETY
    description: "Severe hypoglycaemia <40 — immediate intervention"

  sbp_crisis:
    absolute_threshold: 180
    severity: SAFETY
    description: "Hypertensive urgency — SBP ≥180"
    
  dbp_crisis:
    absolute_threshold: 120
    severity: SAFETY
    description: "Hypertensive urgency — DBP ≥120"

  cgm_tbr_spike:
    baseline_window_days: 30
    absolute_threshold: 5.0    # TBR L2 % in 24h window
    baseline_max: 1.0          # only fire if 30d baseline TBR <1%
    severity: IMMEDIATE
    description: "CGM severe hypo spike — TBR jumped from <1% to >5%"

  hf_weight_gain:
    window_hours: 72
    gain_threshold_kg: 2.0
    severity: URGENT
    requires_context: "CKM_4C"
    description: "Acute weight gain >2kg/72h in HF patient — possible decompensation"

  measurement_cessation:
    baseline_window_days: 30
    cessation_hours: 72
    min_baseline_frequency: 0.8  # at least 0.8 readings/day average to qualify
    severity: URGENT
    description: "Daily measurer stopped for >72 hours"
```

- [ ] **Step 2:** Create Java POJOs for AcuteOnChronicAlert (alertType, patientID, metricType, currentValue, baselineValue, deviationPct, severity, timestamp, description) and PatientBaseline (per-metric rolling buffer with 30-day median computation).

- [ ] **Step 3:** Verify compilation. Commit: `feat(flink): acute-on-chronic models + threshold config (V5 Plan B Task 1)`

---

### Task 2: Flink AcuteOnChronicDetector operator

**Files:**
- Create: `flink-processing/.../operators/AcuteOnChronicDetector.java`
- Create: `flink-processing/src/test/.../AcuteOnChronicDetectorTest.java`

- [ ] **Step 1:** Write 8 failing tests:
1. `testEGFRAcuteDrop_25Percent_FiresAlert` — baseline eGFR 60, new reading 42 (30% drop) → SAFETY alert
2. `testEGFRAcuteDrop_10Percent_NoAlert` — baseline 60, new 55 (8% drop) → no alert
3. `testGlucoseHighCrisis_Above400` — reading 420 → SAFETY alert regardless of baseline
4. `testGlucoseLowCrisis_Below40` — reading 35 → SAFETY alert
5. `testSBPCrisis_Above180` — reading 195 → SAFETY alert
6. `testCGMTBRSpike_From1To6` — 30d baseline TBR 0.8%, new 24h TBR 6.2% → IMMEDIATE alert
7. `testHFWeightGain_2kgIn72h` — weight 88→90.5 in 72h for CKM 4c → URGENT alert
8. `testMeasurementCessation_72h` — patient averaging 1.2 readings/day, no reading for 73h → URGENT alert

- [ ] **Step 2:** Implement `AcuteOnChronicDetector extends KeyedProcessFunction<String, ClinicalReading, AcuteOnChronicAlert>` with:
- `ValueState<PatientBaseline>` for per-patient keyed state
- `processElement()`: update baseline rolling buffer, check each threshold, emit alerts
- Baseline: circular buffer of last 30 days readings, compute median on demand
- Threshold checks: deviation-based (eGFR, CGM TBR) and absolute (glucose, BP)
- Context-aware: HF weight gain only fires when patient state includes CKM 4c

- [ ] **Step 3:** Run tests — all 8 pass.

- [ ] **Step 4:** Commit: `feat(flink): acute-on-chronic streaming detector — 8 alert types (V5 Plan B Task 2)`

---

### Task 3: KB-23 acute alert card templates

**Files:**
- Create: 6 YAML templates in `kb-23-decision-cards/templates/acute/`

- [ ] **Step 1:** Create all 6 templates following the existing renal_contraindication.yaml pattern. Each template has: template_id (`dc-acute-{type}-v1`), node_id CROSS_NODE, card_source STREAMING_DETECTOR, mcu_gate_default (HALT for SAFETY, MODIFY for IMMEDIATE, SAFE for URGENT), clinician + patient fragments with {{.CurrentValue}}, {{.BaselineValue}}, {{.DeviationPct}} placeholders.

- [ ] **Step 2:** Verify YAML parses. Commit: `feat(kb23): acute-on-chronic card templates (V5 Plan B Task 3)`

---

### Task 4: KB-23 acute alert Kafka consumer + card generation

**Files:**
- Create: `kb-23-decision-cards/internal/services/acute_alert_handler.go`
- Create: `kb-23-decision-cards/internal/services/acute_alert_handler_test.go`

- [ ] **Step 1:** Write 6 tests:
1. `TestAcuteHandler_EGFRDrop_ProducesSafetyCard` — EGFR_ACUTE_DROP alert → card with HALT gate
2. `TestAcuteHandler_GlucoseCrisis_ProducesSafetyCard` — GLUCOSE_HIGH_CRISIS → HALT gate
3. `TestAcuteHandler_BPCrisis_ProducesSafetyCard` — SBP_CRISIS → HALT gate
4. `TestAcuteHandler_CGMSpike_ProducesImmediateCard` — CGM_TBR_SPIKE → MODIFY gate
5. `TestAcuteHandler_WeightGain_ProducesUrgentCard` — HF_WEIGHT_GAIN → SAFE gate
6. `TestAcuteHandler_Cessation_ProducesUrgentCard` — MEASUREMENT_CESSATION → SAFE gate

- [ ] **Step 2:** Implement `AcuteAlertHandler` — Kafka consumer for `clinical.acute-alerts.v1` topic. On each alert: load template by alert type, render fragments, persist DecisionCard, call notifyFHIR.

- [ ] **Step 3:** Run tests. Commit: `feat(kb23): acute alert handler — streaming cards from Flink (V5 Plan B Task 4)`

---

### Task 5: Flink job wiring + integration test

- [ ] **Step 1:** Wire AcuteOnChronicDetector into the Flink job graph: KafkaSource (clinical readings) → keyBy(patientId) → AcuteOnChronicDetector → KafkaSink (clinical.acute-alerts.v1).

- [ ] **Step 2:** Full test sweep across Flink, KB-23.

- [ ] **Step 3:** Commit: `feat: complete V5 acute-on-chronic streaming detection`

---

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + config | 4 files | 1-2 hours |
| Task 2: Flink detector | 2 files, 8 tests | 3-4 hours |
| Task 3: Card templates | 6 YAML files | 1 hour |
| Task 4: KB-23 consumer | 2 files, 6 tests | 2-3 hours |
| Task 5: Wiring + integration | 2 files modified | 1-2 hours |
| **Total** | **~16 files, ~14 tests** | **~8-12 hours** |
