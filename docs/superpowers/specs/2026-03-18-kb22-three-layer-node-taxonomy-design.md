# KB-22 Three-Layer Node Taxonomy Design

**Date**: 2026-03-18
**Status**: Draft
**Service**: KB-22 HPI Engine (port 8132)
**Scope**: Add 12 new clinical nodes (8 PM + 6 MD) to KB-22 with new engine types

## 1. Problem Statement

KB-22 currently has 26 Layer 1 symptom presentation nodes (Bayesian differential diagnosis). These detect disease that has already declared itself clinically. The system needs two additional layers:

- **Layer 2 (Physiological Monitoring)**: 8 PM nodes that classify structured physiological data (BP patterns, glucose trends, HRV, exercise capacity) — detects changes weeks before symptoms.
- **Layer 3 (Metabolic Deterioration)**: 6 MD nodes that compute deterioration trajectory signals from KB-20 longitudinal data and KB-26 twin state — detects metabolic shifts months before physiological changes become clinically obvious.

Together, the three layers shift KB-22 from reactive disease detection to predictive metabolic intelligence.

## 2. Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| PM node data source | Abstract `DataResolver` interface | Source-agnostic: works via KB-20/API today, Tier-1 WhatsApp adapter later |
| KB-23 integration | Unified `ClinicalSignalEvent` with `signal_type` discriminator | One ingestion path for all node layers, honest schema |
| MD node execution model | Event-driven (reactive) on KB-20/KB-26 changes | Immediate detection of deterioration; debounce prevents chattiness |
| PM → MD cascading | Internal within KB-22 (same request cycle) | Fast, atomic; function call, not network hop |
| Overall architecture | Unified Node Engine (Approach A) | Single binary, existing infra reuse, minimal ops overhead |

## 3. Unified ClinicalSignalEvent

All three node layers emit this event. KB-23 consumes it via a single new handler.

```go
type ClinicalSignalEvent struct {
    // Header
    EventID        string          `json:"event_id"`
    EventType      string          `json:"event_type"`       // "CLINICAL_SIGNAL"
    SignalType     SignalType       `json:"signal_type"`      // discriminator
    PatientID      string          `json:"patient_id"`
    NodeID         string          `json:"node_id"`
    NodeVersion    string          `json:"node_version"`
    StratumLabel   string          `json:"stratum_label"`
    EmittedAt      time.Time       `json:"emitted_at"`

    // Layer 1: Bayesian Differential (existing HPICompleteEvent fields)
    SessionID           *string                `json:"session_id,omitempty"`
    TopDiagnosis        *string                `json:"top_diagnosis,omitempty"`
    TopPosterior        *float64               `json:"top_posterior,omitempty"`
    RankedDifferentials []DifferentialEntry     `json:"ranked_differentials,omitempty"`
    ConvergenceReached  *bool                  `json:"convergence_reached,omitempty"`
    ReasoningChain      json.RawMessage        `json:"reasoning_chain,omitempty"`
    MedicationBlocks    []MedicationBlock      `json:"medication_blocks,omitempty"`

    // Layer 2: Monitoring Classification (PM nodes)
    Classification      *ClassificationResult  `json:"classification,omitempty"`
    MonitoringData      []MonitoringDataPoint  `json:"monitoring_data,omitempty"`
    TrendDirection      *string                `json:"trend_direction,omitempty"`

    // Layer 3: Deterioration Signal (MD nodes)
    DeteriorationSignal *DeteriorationResult   `json:"deterioration_signal,omitempty"`
    ProjectedThreshold  *ThresholdProjection   `json:"projected_threshold,omitempty"`
    ContributingSignals []string               `json:"contributing_signals,omitempty"`

    // Shared
    SafetyFlags         []SafetyFlagEntry      `json:"safety_flags,omitempty"`
    RecommendedActions  []RecommendedAction    `json:"recommended_actions,omitempty"`
    AcuityCategory      *string                `json:"acuity_category,omitempty"`
    MCUGateSuggestion   *string                `json:"mcu_gate_suggestion,omitempty"`
}

type SignalType string
const (
    SignalBayesianDifferential     SignalType = "BAYESIAN_DIFFERENTIAL"
    SignalMonitoringClassification SignalType = "MONITORING_CLASSIFICATION"
    SignalDeteriorationSignal      SignalType = "DETERIORATION_SIGNAL"
)
```

### Sub-types

```go
type ClassificationResult struct {
    Category        string  `json:"category"`
    Value           float64 `json:"value"`
    Unit            string  `json:"unit"`
    Threshold       string  `json:"threshold"`
    Confidence      float64 `json:"confidence"`
    DataSufficiency string  `json:"data_sufficiency"`
}

type DeteriorationResult struct {
    Signal        string  `json:"signal"`
    Severity      string  `json:"severity"`
    Trajectory    string  `json:"trajectory"`
    RateOfChange  float64 `json:"rate_of_change"`
    StateVariable string  `json:"state_variable"`
}

type ThresholdProjection struct {
    ThresholdName  string    `json:"threshold_name"`
    CurrentValue   float64   `json:"current_value"`
    ThresholdValue float64   `json:"threshold_value"`
    ProjectedDate  time.Time `json:"projected_date"`
    Confidence     float64   `json:"confidence"`
}

type RecommendedAction struct {
    ActionID     string `json:"action_id"`
    Type         string `json:"type"`
    Description  string `json:"description"`
    Urgency      string `json:"urgency"`
    CardTemplate string `json:"card_template"`
}
```

KB-23 receives this via `POST /api/v1/clinical-signals` (new endpoint in KB-23). The `MCUGateSuggestion` field is advisory — KB-23 remains the gate authority with hysteresis filtering (N-01).

## 4. YAML Node Schemas

### 4.1 MonitoringNodeDefinition (Layer 2)

```yaml
node_id: PM-03                            # PM-01 through PM-08
version: "1.0.0"
type: MONITORING                          # discriminator for NodeLoader
title_en: "Nocturnal BP Dipping Classification"
title_hi: "Raat ka BP Dipping Vargikaran"

required_inputs:
  - field: sbp_nocturnal_mean
    source: KB-20                         # KB-20 | TIER1_CHECKIN | DEVICE | MANUAL
    unit: mmHg
    min_observations: 3
    lookback_days: 14
  - field: sbp_daytime_mean
    source: KB-20
    unit: mmHg
    min_observations: 3
    lookback_days: 14

computed_fields:
  - name: dipping_ratio
    formula: "(sbp_nocturnal_mean - sbp_daytime_mean) / sbp_daytime_mean"

classifications:                          # top-to-bottom, first match wins
  - category: REVERSE_DIPPER
    condition: "dipping_ratio > 0.0"
    severity: CRITICAL
    mcu_gate_suggestion: PAUSE
    card_template: "dc-pm03-reverse-dipper-v1"
  - category: NON_DIPPER
    condition: "dipping_ratio >= -0.10"
    severity: MODERATE
    mcu_gate_suggestion: MODIFY
    card_template: "dc-pm03-non-dipper-v1"
  - category: NORMAL_DIPPER
    condition: "dipping_ratio >= -0.20 AND dipping_ratio < -0.10"
    severity: NONE
    mcu_gate_suggestion: SAFE
    card_template: null

insufficient_data:
  action: SKIP                            # SKIP | FLAG_FOR_REVIEW | USE_LAST_KNOWN
  note_en: "Insufficient BP readings for dipping classification"

safety_triggers:
  - id: PM03_ST01
    condition: "sbp_nocturnal_mean > 160"
    severity: URGENT
    action: "Nocturnal hypertension. Review evening antihypertensive timing."

cascade_to:
  - MD-02
  - MD-04

checkin_prompts:                          # for future Tier-1 adapter
  - prompt_id: PM03_CK01
    text_en: "Did you wear your BP monitor while sleeping last night?"
    text_hi: "Kya aapne raat mein sote samay BP monitor pehna tha?"
    response_type: BOOLEAN
    maps_to: nocturnal_monitoring_compliance
```

### 4.2 DeteriorationNodeDefinition (Layer 3)

```yaml
node_id: MD-01                            # MD-01 through MD-06
version: "1.0.0"
type: DETERIORATION                       # discriminator for NodeLoader
title_en: "Insulin Resistance Trajectory Monitor"
title_hi: "Insulin Pratirodh Pravrutti Monitor"

state_variable: IS                        # KB-26 ODE variable: IS|VF|HGO|MM|VR|RR
state_variable_label: "Insulin Sensitivity"

trigger_on:
  - event: "OBSERVATION:FBG"
  - event: "OBSERVATION:PPBG"
  - event: "TWIN_STATE_UPDATE"
  - event: "SIGNAL:PM-04"
  - event: "SIGNAL:PM-05"
  - event: "SIGNAL:PM-06"

required_inputs:
  - field: twin_state.IS
    source: KB-26
    description: "Current insulin sensitivity estimate"
  - field: twin_state.IS_confidence
    source: KB-26
  - field: fbg_7d_mean
    source: KB-20
    unit: mg/dL
    lookback_days: 7
  - field: homa_ir
    source: KB-20
    unit: index
    optional: true

contributing_signals:
  - PM-04
  - PM-05
  - PM-06

trajectory:
  method: LINEAR_REGRESSION               # LINEAR_REGRESSION | EXPONENTIAL_DECAY | BAYESIAN_SLOPE
  window_days: 90
  min_data_points: 5
  rate_unit: "per_month"

thresholds:                               # top-to-bottom, first match wins
  - signal: IS_CRITICAL_DECLINE
    condition: "rate_of_change < -0.08 AND twin_state.IS < 0.30"
    severity: CRITICAL
    trajectory: ACCELERATING
    mcu_gate_suggestion: PAUSE
    card_template: "dc-md01-critical-decline-v1"
    actions:
      - type: INVESTIGATION
        description: "Fasting insulin + C-peptide to confirm progression"
        urgency: 24H
      - type: MEDICATION_REVIEW
        description: "Consider adding/uptitrating insulin sensitizer"
        urgency: 48H
  - signal: IS_STABLE
    condition: "rate_of_change >= -0.04 AND rate_of_change <= 0.04"
    severity: NONE
    trajectory: STABLE
    mcu_gate_suggestion: SAFE
    card_template: null

projections:
  - name: IS_CRITICAL_THRESHOLD
    variable: twin_state.IS
    threshold: 0.20
    method: LINEAR_EXTRAPOLATION
    confidence_required: 0.60

insufficient_data:
  action: FLAG_FOR_REVIEW
  note_en: "Insufficient longitudinal data for trajectory computation"
  fallback: USE_SNAPSHOT
```

## 5. Engine Architecture

### 5.1 New Files

```
internal/
├── models/
│   ├── monitoring_node.go          # MonitoringNodeDefinition struct
│   ├── deterioration_node.go       # DeteriorationNodeDefinition struct
│   ├── clinical_signal_event.go    # ClinicalSignalEvent + sub-types
│   └── data_input.go              # DataRequest, DataResponse
│
├── services/
│   ├── monitoring_engine.go        # MonitoringNodeEngine
│   ├── deterioration_engine.go     # DeteriorationNodeEngine
│   ├── signal_cascade.go           # PM→MD + MD→MD cascade coordinator
│   ├── data_resolver.go            # Abstract data fetching from KB-20/KB-26/cache
│   ├── expression_evaluator.go     # Condition strings + computed_fields
│   ├── trajectory_computer.go      # Linear regression / Bayesian slope
│   ├── signal_publisher.go         # ClinicalSignalEvent → KB-23 + Kafka
│   └── kb26_client.go              # KB-26 twin state API client
│
├── api/
│   ├── event_ingestion_handlers.go # Webhooks for KB-20/KB-26 events
│   └── signal_query_handlers.go    # Query latest signals per patient

monitoring/                         # PM node YAML files
deterioration/                      # MD node YAML files
```

### 5.2 Modified Files

| File | Change |
|---|---|
| `internal/services/node_loader.go` | Add MONITORING/DETERIORATION type dispatch, load from new dirs |
| `internal/api/server.go` | Register new route groups, wire new engines into DI |
| `internal/config/config.go` | Add KB26_URL, new dir paths, debounce config |

No changes to existing Layer 1 code (Bayesian engine, session service, safety engine, question orchestrator).

### 5.3 Core Interfaces

```go
// DataResolver — abstract data fetching (key abstraction for source-agnostic PM nodes)
type DataResolver interface {
    Resolve(ctx context.Context, patientID string, inputs []RequiredInput) (*ResolvedData, error)
}

type ResolvedData struct {
    Fields          map[string]float64
    FieldTimestamps map[string]time.Time
    Sufficiency     DataSufficiency      // SUFFICIENT | PARTIAL | INSUFFICIENT
    MissingFields   []string
    Sources         map[string]string
}
```

### 5.4 MonitoringNodeEngine Flow

1. Receive trigger (KB-20 observation event or API call)
2. `NodeLoader.GetMonitoringNode(nodeID)`
3. `DataResolver.Resolve(patientID, node.RequiredInputs)` — fetches from KB-20/KB-26/cache
4. Check `DataSufficiency` — if INSUFFICIENT, apply `node.insufficient_data` policy
5. Evaluate `computed_fields` via `ExpressionEvaluator`
6. Evaluate `classifications` top-to-bottom, first match wins
7. Evaluate `safety_triggers` in parallel (reuses Layer 1 safety trigger schema)
8. Build `ClinicalSignalEvent` with `SignalType = MONITORING_CLASSIFICATION`
9. Publish to KB-23 + Kafka via `SignalPublisher`
10. `SignalCascade.Trigger(nodeID, patientID, classificationResult)` — invoke subscribed MD nodes

### 5.5 DeteriorationNodeEngine Flow

1. Receive trigger (KB-20/KB-26 event OR internal cascade from PM node)
2. `NodeLoader.GetDeteriorationNode(nodeID)`
3. `DataResolver.Resolve(patientID, node.RequiredInputs)` — includes KB-26 twin state
4. Check `DataSufficiency` — if INSUFFICIENT, apply policy (FLAG_FOR_REVIEW / USE_SNAPSHOT)
5. `TrajectoryComputer.Compute(observations, node.Trajectory)` — linear regression, rate per month
6. Evaluate `thresholds` top-to-bottom, first match wins
7. Compute `projections` if configured (linear extrapolation to clinical thresholds)
8. Build `ClinicalSignalEvent` with `SignalType = DETERIORATION_SIGNAL`
9. Publish to KB-23 + Kafka via `SignalPublisher`

### 5.6 Signal Cascade Coordinator

Two-pass evaluation:
- **Pass 1**: PM → MD (evaluate MD-01 through MD-05 that subscribe to the triggering PM node)
- **Pass 2**: MD → MD-06 (if any MD node from Pass 1 is in MD-06's `contributing_signals`)

```go
type SignalCascade struct {
    pmToMD    map[string][]string   // PM nodeID → []MD nodeIDs
    mdToMD06  map[string]bool       // MD nodeIDs that feed MD-06
    deterEngine *DeteriorationNodeEngine
}

func (sc *SignalCascade) Trigger(ctx, sourceNodeID, patientID, classification) []*ClinicalSignalEvent {
    // Pass 1: PM → MD
    pass1Results := map[string]*ClinicalSignalEvent{}
    for _, mdNodeID := range sc.pmToMD[sourceNodeID] {
        event, err := sc.deterEngine.Evaluate(ctx, mdNodeID, patientID, cascadeCtx)
        if err != nil { continue }  // cascade failures non-fatal
        if event != nil { pass1Results[mdNodeID] = event }
    }

    // Pass 2: MD → MD-06 (if any Pass 1 result feeds MD-06)
    needsMD06 := false
    for mdID := range pass1Results {
        if sc.mdToMD06[mdID] { needsMD06 = true; break }
    }
    if needsMD06 {
        event, _ := sc.deterEngine.Evaluate(ctx, "MD-06", patientID, md06CascadeCtx)
        if event != nil { pass1Results["MD-06"] = event }
    }

    return collect(pass1Results)
}
```

### 5.7 Debounce Strategy

Per-patient per-node Redis cache with 5-minute TTL:

```
Key:   "eval:{patientID}:{nodeID}"
Value: last evaluation timestamp
TTL:   300 seconds (configurable via SIGNAL_DEBOUNCE_TTL_SEC)

On event:
  1. Check cache → if exists → SKIP (unless safety trigger condition met)
  2. If miss → evaluate → set cache key
```

## 6. API Endpoints

### 6.1 Event Ingestion (webhook receivers)

```
POST /api/v1/events/observation
  Body: { event_type, patient_id, observation_code, value, unit, timestamp, source }
  Response: 202 Accepted
  Logic: Route to matching PM/MD nodes → evaluate → cascade → publish

POST /api/v1/events/twin-state-update
  Body: { event_type, patient_id, state_version, changed_variables, update_source, timestamp }
  Response: 202 Accepted
  Logic: Route to MD nodes with matching trigger_on → evaluate → publish

POST /api/v1/events/checkin-response
  Body: { event_type, patient_id, prompt_id, response_value, response_type, timestamp }
  Response: 202 Accepted
  Logic: Future Tier-1 adapter endpoint. Routes to PM nodes by prompt_id mapping.
```

### 6.2 Signal Queries

```
GET /api/v1/patients/:id/signals
  Response: Latest ClinicalSignalEvent per node for patient
  Source: clinical_signals_latest table

GET /api/v1/patients/:id/signals/:nodeId
  Params: ?limit=10
  Response: Signal history for specific node
  Source: clinical_signals table, ordered by evaluated_at DESC

GET /api/v1/patients/:id/deterioration-summary
  Response: All MD node latest results in single view (6 deterioration signals + projections)
  Source: clinical_signals_latest WHERE signal_type = DETERIORATION_SIGNAL
```

### 6.3 Node Management (extend existing)

```
GET  /api/v1/nodes/monitoring              # List loaded PM node definitions
GET  /api/v1/nodes/monitoring/:nodeId      # Get specific PM node definition
GET  /api/v1/nodes/deterioration           # List loaded MD node definitions
GET  /api/v1/nodes/deterioration/:nodeId   # Get specific MD node definition
POST /internal/nodes/reload                # EXISTING — extended to reload PM/MD YAML
```

## 7. Database Changes

### Migration: `006_monitoring_deterioration.sql`

```sql
CREATE TABLE clinical_signals (
    signal_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    node_version    VARCHAR(20) NOT NULL,
    signal_type     VARCHAR(30) NOT NULL,
    stratum_label   VARCHAR(100),

    -- PM node fields
    classification_category  VARCHAR(50),
    classification_value     DOUBLE PRECISION,
    classification_unit      VARCHAR(20),
    data_sufficiency         VARCHAR(20),

    -- MD node fields
    deterioration_signal     VARCHAR(80),
    severity                 VARCHAR(20),
    trajectory               VARCHAR(20),
    rate_of_change           DOUBLE PRECISION,
    state_variable           VARCHAR(10),

    -- Projection
    projected_threshold_name  VARCHAR(80),
    projected_threshold_date  TIMESTAMPTZ,
    projection_confidence     DOUBLE PRECISION,

    -- Shared
    resolved_data            JSONB,
    contributing_signals     JSONB,
    safety_flags             JSONB,
    mcu_gate_suggestion      VARCHAR(10),
    published_to_kb23        BOOLEAN DEFAULT FALSE,

    evaluated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signals_patient_node ON clinical_signals(patient_id, node_id, evaluated_at DESC);
CREATE INDEX idx_signals_patient_type ON clinical_signals(patient_id, signal_type, evaluated_at DESC);
CREATE INDEX idx_signals_severity ON clinical_signals(severity) WHERE severity IN ('SEVERE', 'CRITICAL');
CREATE INDEX idx_signals_unpublished ON clinical_signals(published_to_kb23) WHERE published_to_kb23 = FALSE;

CREATE TABLE clinical_signals_latest (
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    signal_id       UUID NOT NULL REFERENCES clinical_signals(signal_id),
    evaluated_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (patient_id, node_id)
);

CREATE TABLE signal_evaluation_log (
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    last_evaluated  TIMESTAMPTZ NOT NULL,
    last_trigger    VARCHAR(100),
    PRIMARY KEY (patient_id, node_id)
);
```

## 8. Complete 12-Node Inventory

### 8.1 Layer 2: Physiological Monitoring Nodes

#### PM-01: Home BP Monitoring
- **Purpose**: Classify home BP readings against stratum-specific targets
- **Inputs**: `sbp_home_mean` (KB-20, 7d), `dbp_home_mean` (KB-20, 7d), `bp_target_sbp` (KB-20 stratum)
- **Computed**: `sbp_delta = sbp_home_mean - bp_target_sbp`
- **Classifications**: SEVERELY_ABOVE (delta>30, CRITICAL/PAUSE) → ABOVE_TARGET (delta>10, MODERATE/MODIFY) → AT_TARGET (|delta|<=10, NONE/SAFE) → BELOW_TARGET (delta<-10, MODERATE/MODIFY) → HYPOTENSIVE (sbp<90, CRITICAL/PAUSE)
- **Safety**: SBP > 180 → IMMEDIATE
- **Cascade**: MD-02, MD-06
- **Checkin**: "What was your BP reading this morning?" → structured SBP/DBP entry

#### PM-02: Daily Symptom Check-in
- **Purpose**: Structured daily screening mapped to Layer 1 P-nodes
- **Inputs**: `symptom_headache`, `symptom_dizziness`, `symptom_chest_pain`, `symptom_fatigue`, `symptom_oedema` (all TIER1_CHECKIN, BOOLEAN)
- **Computed**: `symptom_count = sum(all booleans)`
- **Classifications**: MULTI_SYMPTOM (count>=3, MODERATE/MODIFY) → SINGLE_SYMPTOM (count 1-2, MILD/SAFE) → ASYMPTOMATIC (0, NONE/SAFE)
- **Safety**: `symptom_chest_pain = true` → IMMEDIATE (auto-trigger P01 HPI session)
- **Cascade**: MD-04, MD-06
- **Checkin**: "How are you feeling today? Any of these?" → multi-select buttons

#### PM-03: Nocturnal BP Dipping Pattern
- **Purpose**: Classify nocturnal dipping from ABPM or home night readings
- **Inputs**: `sbp_nocturnal_mean` (KB-20, 14d), `sbp_daytime_mean` (KB-20, 14d)
- **Computed**: `dipping_ratio = (nocturnal - daytime) / daytime`
- **Classifications**: REVERSE_DIPPER (ratio>0, CRITICAL/PAUSE) → NON_DIPPER (ratio>=-0.10, MODERATE/MODIFY) → EXTREME_DIPPER (ratio<-0.20, MODERATE/MODIFY) → NORMAL_DIPPER (-0.20 to -0.10, NONE/SAFE)
- **Safety**: `sbp_nocturnal_mean > 160` → URGENT
- **Cascade**: MD-02, MD-04
- **Min observations**: 3 nocturnal + 3 daytime over 14d

#### PM-04: Postprandial Blood Glucose Pattern
- **Purpose**: Classify PPBG excursion patterns and post-meal spikes
- **Inputs**: `ppbg_values` (KB-20, 14d, array), `fbg_7d_mean` (KB-20)
- **Computed**: `ppbg_mean`, `ppbg_cv`, `excursion = ppbg_mean - fbg_7d_mean`
- **Classifications**: SEVERE_EXCURSION (excursion>80, CRITICAL/PAUSE) → HIGH_EXCURSION (50-80, MODERATE/MODIFY) → MODERATE_EXCURSION (30-50, MILD/SAFE) → NORMAL_EXCURSION (<30, NONE/SAFE)
- **Safety**: Any single `ppbg > 300` → URGENT
- **Cascade**: MD-01, MD-05
- **Checkin**: "What was your sugar reading after lunch today?" → numeric entry

#### PM-05: Glycemic Variability
- **Purpose**: CV% of glucose readings to detect instability
- **Inputs**: `glucose_all_values` (KB-20, 14d, array of all FBG+PPBG)
- **Computed**: `glucose_cv = stdev / mean * 100`
- **Classifications**: HIGHLY_VARIABLE (cv>36, MODERATE/MODIFY) → MODERATELY_VARIABLE (20-36, MILD/SAFE) → STABLE (cv<20, NONE/SAFE)
- **Safety**: `glucose_cv > 50` → URGENT
- **Cascade**: MD-01, MD-05
- **Min observations**: 6 readings over 14d
- **Evidence**: 36% threshold from International Consensus on CGM (Danne et al. 2017)

#### PM-06: FBG Trend Computation
- **Purpose**: FBG trajectory — purely algorithmic, no patient interaction
- **Inputs**: `fbg_values` (KB-20, 90d, array with timestamps)
- **Computed**: `fbg_slope` (linear regression, mg/dL per month), `fbg_current_mean` (7d), `fbg_prior_mean` (30-90d)
- **Classifications**: RAPIDLY_RISING (slope>5/month, MODERATE/MODIFY) → GRADUALLY_RISING (2-5, MILD/SAFE) → STABLE (|slope|<2, NONE/SAFE) → IMPROVING (slope<-2, NONE/SAFE)
- **Safety**: `fbg_current_mean > 250` → URGENT
- **Cascade**: MD-01, MD-05
- **Min observations**: 5 FBG readings over 90d
- **No checkin prompts**: entirely computed from KB-20

#### PM-07: Exercise Capacity / Muscle Function
- **Purpose**: Track functional capacity via step count trends and self-reported tolerance
- **Inputs**: `daily_steps_7d_mean` (KB-20/DEVICE), `daily_steps_30d_mean` (KB-20/DEVICE), `exercise_tolerance_self` (TIER1_CHECKIN, LIKERT_5)
- **Computed**: `steps_trend = (7d_mean - 30d_mean) / 30d_mean`
- **Classifications**: DECLINING_RAPIDLY (trend<-0.25, MODERATE/MODIFY) → DECLINING (-0.25 to -0.10, MILD/SAFE) → STABLE (|trend|<0.10, NONE/SAFE) → IMPROVING (trend>0.10, NONE/SAFE)
- **Safety**: `steps_7d < 500 AND exercise_tolerance <= 2` → URGENT
- **Cascade**: MD-04, MD-06
- **Checkin**: "How would you rate your energy for physical activity this week?" → Likert 1-5

#### PM-08: Heart Rate Variability Pattern
- **Purpose**: HRV trend from wearable data as autonomic health proxy
- **Inputs**: `rmssd_7d_mean` (DEVICE, ms), `rmssd_30d_mean` (DEVICE, ms), `resting_hr_7d_mean` (KB-20/DEVICE)
- **Computed**: `hrv_trend = (7d - 30d) / 30d`, `hr_hrv_ratio = resting_hr / rmssd`
- **Classifications**: SEVERELY_DEPRESSED (rmssd<15 AND hr>90, MODERATE/MODIFY) → DECLINING (trend<-0.20, MILD/SAFE) → STABLE (|trend|<0.20, NONE/SAFE) → IMPROVING (trend>0.20, NONE/SAFE)
- **Safety**: None (trend indicator, not acute)
- **Cascade**: MD-04
- **Insufficient data**: SKIP (many patients lack wearable HRV)
- **Min observations**: 5 days of RMSSD

### 8.2 Layer 3: Metabolic Deterioration Nodes

#### MD-01: Insulin Resistance Trajectory
- **KB-26 variable**: IS (Insulin Sensitivity)
- **Trigger**: OBSERVATION:FBG, OBSERVATION:PPBG, OBSERVATION:HOMA_IR, TWIN_STATE_UPDATE, SIGNAL:PM-04, SIGNAL:PM-05, SIGNAL:PM-06
- **Trajectory**: Linear regression on IS over 90d, rate per month
- **Thresholds**: IS_CRITICAL_DECLINE (rate<-0.08 AND IS<0.30, CRITICAL/PAUSE — fasting insulin 24H, med review 48H) → IS_MODERATE_DECLINE (rate<-0.04, MODERATE/MODIFY — exercise 72H, increase monitoring) → IS_STABLE (|rate|<=0.04, SAFE) → IS_IMPROVING (rate>0.04, SAFE)
- **Projection**: When will IS cross 0.20? Linear extrapolation, confidence >= 0.60
- **Contributing**: PM-04, PM-05, PM-06

#### MD-02: Vascular Compliance Decline
- **KB-26 variable**: VR (Vascular Resistance)
- **Trigger**: OBSERVATION:SBP, OBSERVATION:DBP, TWIN_STATE_UPDATE, SIGNAL:PM-01, SIGNAL:PM-03
- **Computed**: `pulse_pressure = sbp - dbp`
- **Trajectory**: Linear regression on VR over 90d
- **Thresholds**: VR_CRITICAL_RISE (rate>0.08 AND VR>0.70, CRITICAL/PAUSE — nephro/cardio review 24H) → VR_MODERATE_RISE (rate>0.04, MODERATE/MODIFY — antihypertensive review 48H) → VR_STABLE (|rate|<=0.04, SAFE) → VR_IMPROVING (rate<-0.04, SAFE)
- **Projection**: When will VR cross 0.80?
- **Contributing**: PM-01, PM-03

#### MD-03: Renal Function Trajectory
- **KB-26 variable**: RR (Renal Reserve)
- **Trigger**: OBSERVATION:EGFR, OBSERVATION:CREATININE, OBSERVATION:ACR, TWIN_STATE_UPDATE
- **Trajectory**: Uses KB-26 pre-computed renal_slope (mL/min/1.73m²/year)
- **Thresholds**: RR_RAPID_DECLINE (slope<-5/yr, CRITICAL/PAUSE — nephrology 24H, hold nephrotoxics IMMEDIATE) → RR_PROGRESSIVE_DECLINE (-5 to -3, MODERATE/MODIFY — optimize RAAS 48H, monthly eGFR) → RR_SLOW_DECLINE (-3 to -1, MILD/SAFE — quarterly eGFR) → RR_STABLE (|slope|<1, SAFE)
- **Projection**: When will eGFR cross next CKD stage (45→G3b, 30→G4)?
- **Evidence**: KDIGO 2024 defines rapid decline as >5 mL/min/1.73m²/year
- **Contributing**: None from PM — direct KB-20/KB-26

#### MD-04: Autonomic Dysfunction Progression
- **KB-26 variable**: None directly — composite signal
- **Trigger**: SIGNAL:PM-03, SIGNAL:PM-07, SIGNAL:PM-08, TWIN_STATE_UPDATE
- **Computed**: `autonomic_score = 0.35*pm03 + 0.30*pm08 + 0.20*pm07 + 0.15*orthostatic` (severity → numeric: NONE=0, MILD=1, MODERATE=2, CRITICAL=3)
- **Thresholds**: AUTONOMIC_SEVERE (score>=2.5, CRITICAL/PAUSE — tilt-table 48H, review cardioactive meds 24H) → AUTONOMIC_MODERATE (1.5-2.5, MODERATE/MODIFY — orthostatic BP protocol 72H) → AUTONOMIC_MILD (0.5-1.5, MILD/SAFE) → AUTONOMIC_NORMAL (<0.5, SAFE)
- **Contributing**: PM-03 (dipping), PM-07 (exercise), PM-08 (HRV)
- **Note**: Most cascade-dependent MD node

#### MD-05: Glycemic Control Deterioration
- **KB-26 variable**: HGO (Hepatic Glucose Output)
- **Trigger**: OBSERVATION:FBG, OBSERVATION:PPBG, OBSERVATION:HBA1C, TWIN_STATE_UPDATE, SIGNAL:PM-04, SIGNAL:PM-05, SIGNAL:PM-06
- **Computed**: `glycemic_composite = 0.35*normalize(hba1c,6,12) + 0.30*normalize(fbg,90,250) + 0.20*pm04_severity + 0.15*pm05_severity`
- **Trajectory**: Linear regression on composite over 90d
- **Thresholds**: GLYCEMIC_CRITICAL (composite>0.80 AND accelerating, CRITICAL/PAUSE — urgent med intensification 24H) → GLYCEMIC_WORSENING (0.60-0.80 OR rate>0.03/month, MODERATE/MODIFY — adherence review 48H, dietary counseling 72H) → GLYCEMIC_SUBOPTIMAL (0.40-0.60, MILD/SAFE) → GLYCEMIC_CONTROLLED (<0.40, SAFE)
- **Projection**: When will HbA1c cross 7.0%, 8.0%, 9.0%? Uses KB-26 biomarker equations.
- **Contributing**: PM-04 (PPBG), PM-05 (glycemic variability), PM-06 (FBG trend)

#### MD-06: Cardiovascular Risk Emergence
- **KB-26 variables**: VR + IS + VF (composite)
- **Trigger**: SIGNAL:MD-01, SIGNAL:MD-02, SIGNAL:MD-03, SIGNAL:PM-01, SIGNAL:PM-02, TWIN_STATE_UPDATE
- **Computed**: `cv_risk_score = 0.25*md01 + 0.25*md02 + 0.20*md03 + 0.15*pm01 + 0.15*(VF*3)`
- **Thresholds**: CV_RISK_CRITICAL (score>=2.5, CRITICAL/HALT — comprehensive CV assessment IMMEDIATE) → CV_RISK_HIGH (1.8-2.5, MODERATE/PAUSE — cardiology 48H) → CV_RISK_ELEVATED (1.0-1.8, MILD/MODIFY) → CV_RISK_LOW (<1.0, SAFE)
- **Contributing**: MD-01, MD-02, MD-03, PM-01, PM-02
- **Note**: Only node that can suggest HALT gate. Second-order cascade (MD→MD).

## 9. Cascade Dependency Graph

```
Layer 2 (PM)              Layer 3 (MD)
────────────              ──────────────
PM-01 (Home BP)     ──→   MD-02 (Vascular)    ──┐
                    ──→   MD-06 (CV Risk)    ◄──┤
PM-02 (Symptoms)    ──→   MD-04 (Autonomic)    │
                    ──→   MD-06               ◄──┤
PM-03 (Dipping)     ──→   MD-02               ──┤
                    ──→   MD-04                  │
PM-04 (PPBG)        ──→   MD-01 (Insulin Res)──┤
                    ──→   MD-05 (Glycemic)   ──┤
PM-05 (Glyc Var)    ──→   MD-01              ──┤
                    ──→   MD-05              ──┤
PM-06 (FBG Trend)   ──→   MD-01              ──┤
                    ──→   MD-05              ──┤
PM-07 (Exercise)    ──→   MD-04                │
PM-08 (HRV)         ──→   MD-04                │
                                               │
                    MD-01 ──→ MD-06 ◄──────────┘
                    MD-02 ──→ MD-06
                    MD-03 ──→ MD-06
```

Two-pass cascade: Pass 1 evaluates PM→MD (MD-01 through MD-05). Pass 2 evaluates MD→MD-06 if any Pass 1 node is in MD-06's contributing_signals.

## 10. Configuration

New environment variables in `config.go`:

```
KB26_URL=http://localhost:8137
KB26_TIMEOUT_MS=5000
MONITORING_NODES_DIR=/app/monitoring
DETERIORATION_NODES_DIR=/app/deterioration
SIGNAL_DEBOUNCE_TTL_SEC=300
SIGNAL_PUBLISHER_RETRY_COUNT=3
SIGNAL_PUBLISHER_RETRY_DELAY_SEC=30
```

## 11. Testing Strategy

### Unit Tests
- `monitoring_engine_test.go`: Classification logic for each PM node category
- `deterioration_engine_test.go`: Trajectory computation + threshold evaluation
- `signal_cascade_test.go`: Two-pass cascade ordering, non-fatal failure handling
- `data_resolver_test.go`: Multi-source resolution, insufficiency detection
- `expression_evaluator_test.go`: Condition parsing, computed fields
- `trajectory_computer_test.go`: Linear regression accuracy, min data point enforcement

### Integration Tests
- Full PM→MD cascade: PM-04 classification → MD-01 evaluation → ClinicalSignalEvent published
- KB-26 twin state update → MD-01 through MD-06 evaluation
- Debounce: rapid events → single evaluation
- Insufficient data: graceful degradation per node policy

### Golden Dataset
- `calibration/pm_golden_dataset.json`: Known BP readings → expected PM-01/PM-03 classifications
- `calibration/md_golden_dataset.json`: Known KB-26 state trajectories → expected MD signals

## 12. KB-23 Changes Required

KB-23 needs a new endpoint to accept `ClinicalSignalEvent`:

```
POST /api/v1/clinical-signals
  Body: ClinicalSignalEvent
  Response: 201 Created (DecisionCard) or 204 No Content (no card needed)
```

New template categories needed:
- `templates/monitoring/` — PM node card templates (e.g., `dc-pm03-reverse-dipper-v1.yaml`)
- `templates/deterioration/` — MD node card templates (e.g., `dc-md01-critical-decline-v1.yaml`)

Template authoring is a separate task — this spec defines the event contract that templates consume.

## 13. Migration Path for Existing HPICompleteEvent

The existing `OutcomePublisher` continues publishing `HPICompleteEvent` to KB-23's existing `POST /api/v1/decision-cards` endpoint. Migration to `ClinicalSignalEvent` for Layer 1 is optional and deferred — no breaking changes.

When ready, Layer 1 migration is:
1. Map HPICompleteEvent fields into ClinicalSignalEvent with `SignalType = BAYESIAN_DIFFERENTIAL`
2. Switch OutcomePublisher to publish to `/api/v1/clinical-signals`
3. KB-23 routes by `signal_type` to existing card builder vs new signal handler

## 14. Risk Assessment

| Risk | Likelihood | Mitigation |
|---|---|---|
| KB-26 API not stable enough for MD nodes | Medium | DataResolver graceful degradation; MD nodes fall back to KB-20-only trajectory |
| Debounce too aggressive (misses rapid deterioration) | Low | Safety trigger bypass: skip debounce if any safety condition matches |
| Cascade loop (MD-06 triggers something that re-triggers MD-06) | None | MD-06 has no `cascade_to` — it's a terminal node. Cascade is acyclic by design. |
| Clinical threshold values wrong | Medium | All thresholds in YAML, hot-reloadable. Golden dataset calibration. Clinical team review. |
| PM nodes generate noise from insufficient data | Medium | `insufficient_data` policy per node. `data_sufficiency` field in every signal for downstream filtering. |
