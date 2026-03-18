# KB-22 Three-Layer Node Taxonomy Design

**Date**: 2026-03-18
**Status**: Draft
**Service**: KB-22 HPI Engine (port 8132)
**Scope**: Add 15 new clinical nodes (9 PM + 6 MD) to KB-22 with new engine types

## 1. Problem Statement

KB-22 currently has 26 Layer 1 symptom presentation nodes (Bayesian differential diagnosis). These detect disease that has already declared itself clinically. The system needs two additional layers:

- **Layer 2 (Physiological Monitoring)**: 9 PM nodes that classify structured physiological data (BP patterns, glucose trends, HRV, exercise capacity, sleep quality) — detects changes weeks before symptoms.
- **Layer 3 (Metabolic Deterioration)**: 6 MD nodes that compute deterioration trajectory signals from KB-20 longitudinal data and KB-26 twin state — detects metabolic shifts months before physiological changes become clinically obvious.

Together, the three layers shift KB-22 from reactive disease detection to predictive metabolic intelligence.

### 1.1 PM Node Numbering: Relationship to Conceptual Specification

The conceptual KB-22 Clinical Node Taxonomy document defined PM-01 through PM-08 with different assignments (PM-01 Protein Intake, PM-02 Activity Score, etc.). This engineering spec renumbers PM nodes based on engineering readiness and coupling to the MD deterioration cascade. The conceptual spec's PM-01 (Protein Intake) and PM-02 (Activity Score) are already operational via M3-PRP and M3-VFRP protocols — they are not re-implemented as KB-22 PM nodes.

**M3 protocol signals enter the cascade as external trigger events**, not as PM nodes:
- `PROTOCOL:M3-PRP:ADHERENCE` — protein intake adherence signal from M3-PRP, consumed by MD-01 (Insulin Resistance) and MD-04 (Autonomic Dysfunction) as a contributing lifestyle factor
- `PROTOCOL:M3-VFRP:ACTIVITY` — activity/exercise signal from M3-VFRP, consumed by MD-04 (Autonomic Dysfunction) and MD-06 (Cardiovascular Risk)

These are listed in the `trigger_on` fields of the relevant MD node YAML definitions (see Section 11.2).

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

### 3.1 Type Alignment with Existing Codebase

The event uses its own type definitions (not reusing KB-22's `HPICompleteEvent` or KB-23's `models.HPICompleteEvent` directly) because Layer 2/3 nodes do not produce Bayesian differentials. When Layer 1 eventually migrates to this event (Section 13), an explicit adapter maps `HPICompleteEvent` fields.

**Actual codebase types for reference:**
- KB-22 `HPICompleteEvent` (`internal/models/events.go`): `SessionID uuid.UUID`, `TopPosterior float64` (non-pointer), `ConvergenceReached bool` (non-pointer)
- KB-23 `HPICompleteEvent` (`internal/models/events.go`): `SessionID uuid.UUID`, `TopPosterior float64`, `RankedDifferentials []DifferentialEntry{DifferentialID, Posterior}`
- KB-22 uses `SafetyFlagSummary`; KB-23 uses `SafetyFlagEntry` — different structs

The `ClinicalSignalEvent` defines its own canonical types to avoid coupling to either service's internal models.

```go
type ClinicalSignalEvent struct {
    // Header
    EventID        string          `json:"event_id"`         // UUID string
    EventType      string          `json:"event_type"`       // "CLINICAL_SIGNAL"
    SignalType     SignalType       `json:"signal_type"`      // discriminator
    PatientID      string          `json:"patient_id"`       // UUID string (not uuid.UUID — wire format)
    NodeID         string          `json:"node_id"`
    NodeVersion    string          `json:"node_version"`
    StratumLabel   string          `json:"stratum_label"`
    EmittedAt      time.Time       `json:"emitted_at"`

    // Layer 2: Monitoring Classification (PM nodes)
    Classification      *ClassificationResult  `json:"classification,omitempty"`
    MonitoringData      []MonitoringDataPoint  `json:"monitoring_data,omitempty"`
    TrendDirection      *string                `json:"trend_direction,omitempty"`

    // Layer 3: Deterioration Signal (MD nodes)
    DeteriorationSignal *DeteriorationResult   `json:"deterioration_signal,omitempty"`
    ProjectedThreshold  *ThresholdProjection   `json:"projected_threshold,omitempty"`
    ContributingSignals []string               `json:"contributing_signals,omitempty"`

    // Shared: Safety + Actions
    SafetyFlags         []SignalSafetyFlag     `json:"safety_flags,omitempty"`
    RecommendedActions  []RecommendedAction    `json:"recommended_actions,omitempty"`
    AcuityCategory      *string                `json:"acuity_category,omitempty"`
    MCUGateSuggestion   *string                `json:"mcu_gate_suggestion,omitempty"`
}

type SignalType string
const (
    SignalMonitoringClassification SignalType = "MONITORING_CLASSIFICATION"
    SignalDeteriorationSignal      SignalType = "DETERIORATION_SIGNAL"
)

// SignalSafetyFlag — own type, not reusing KB-22's SafetyFlagSummary or KB-23's SafetyFlagEntry
type SignalSafetyFlag struct {
    FlagID     string `json:"flag_id"`
    Severity   string `json:"severity"`    // IMMEDIATE | URGENT | WARN
    Action     string `json:"action"`
    Condition  string `json:"condition"`   // for audit trail
}
```

**Note**: `SignalBayesianDifferential` is intentionally omitted from the `SignalType` enum. Layer 1 continues using `HPICompleteEvent` via the existing `POST /api/v1/decision-cards` endpoint. Migration to `ClinicalSignalEvent` is deferred (Section 16).

### 3.2 MRI (Metabolic Risk Index) Integration

The Metabolic Risk Index (KB-26 computed module) consumes KB-22 Layer 2/3 outputs to recompute domain risk scores. PM node classifications (e.g., REVERSE_DIPPER, SEVERE_EXCURSION) are not twin state updates — they are clinical signals that should trigger MRI recomputation.

**Approach**: KB-26 subscribes to the `clinical.signal.events` Kafka topic. When a relevant PM/MD signal arrives, KB-26's MRI module recomputes the affected domain score. KB-22 does not call KB-26 directly for MRI — KB-26 decides which signals affect which MRI domains.

```
KB-22 PM/MD evaluation
  → SignalPublisher
    → KB-23 (POST /api/v1/clinical-signals)     — card generation
    → Kafka topic: clinical.signal.events        — event bus
        → KB-26 MRI module (consumer)            — risk score recomputation
        → KB-19 (consumer, if needed)            — orchestration
```

This keeps KB-22 unaware of MRI internals. KB-26's Kafka consumer filters by `signal_type` and `node_id` to decide which signals affect which MRI domain (e.g., PM-03 REVERSE_DIPPER → cardiovascular domain, PM-05 HIGHLY_VARIABLE → glycemic domain).

### 3.3 Sub-types

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
    StateVariable string  `json:"state_variable"`  // logical name: IS|VF|HGO|MM|VR|RR
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

type MonitoringDataPoint struct {
    Field     string    `json:"field"`
    Value     float64   `json:"value"`
    Unit      string    `json:"unit"`
    Timestamp time.Time `json:"timestamp"`
    Source    string    `json:"source"`
}
```

## 4. KB-23 Integration: Signal Handler Design

### 4.1 Current KB-23 Architecture

KB-23's card generation pipeline (`kb-23-decision-cards/internal/api/card_handlers.go`) is tightly coupled to `HPICompleteEvent`:

```
POST /api/v1/decision-cards
  → handleGenerateCard()
    → models.HPICompleteEvent (deserialized)
    → TemplateSelector.SelectBest(event.TopDiagnosis, event.NodeID)
    → CardBuilder.Build(template, event, patientCtx)
      → ConfidenceTierService (uses event.TopPosterior)
      → MCUGateManager (uses confidence tier + template gate_rules)
      → HysteresisEngine (checks gate oscillation)
```

PM/MD signals cannot use this path because they have no `TopDiagnosis`, `TopPosterior`, or `RankedDifferentials`.

### 4.2 New KB-23 Endpoint

```
POST /api/v1/clinical-signals
  Body: ClinicalSignalEvent
  Response: 201 Created (DecisionCard) | 204 No Content (no card needed)
```

### 4.3 Signal Card Builder (new KB-23 component)

A new `SignalCardBuilder` handles `ClinicalSignalEvent`, separate from the existing `CardBuilder`:

```go
// kb-23-decision-cards/internal/services/signal_card_builder.go

type SignalCardBuilder struct {
    templateLoader *TemplateLoader
    gateManager    *MCUGateManager       // reused from existing
    hysteresis     *HysteresisEngine     // reused from existing
    kb20Client     *KB20Client           // reused from existing
    kb19Publisher  *KB19Publisher         // reused from existing
}

func (b *SignalCardBuilder) Build(ctx context.Context, event *ClinicalSignalEvent) (*DecisionCard, error) {
    // 1. Template selection by node_id + classification/signal
    //    New template directories: templates/monitoring/, templates/deterioration/
    //    Template key: "{node_id}:{classification_category}" or "{node_id}:{deterioration_signal}"
    templateID := b.resolveTemplate(event)
    if templateID == "" {
        return nil, nil  // 204 No Content — no card for this classification (e.g., NORMAL_DIPPER)
    }

    // 2. Confidence tier: for PM/MD signals, derive from data_sufficiency + trajectory confidence
    //    SUFFICIENT data + CRITICAL severity → FIRM
    //    PARTIAL data + any severity → POSSIBLE
    //    Maps to existing ConfidenceTier enum
    tier := b.deriveConfidenceTier(event)

    // 3. MCU gate evaluation: use event.MCUGateSuggestion as input, apply hysteresis
    //    Reuses existing HysteresisEngine — prevents gate oscillation
    //    HALT gate (from MD-06) requires explicit clinician acknowledgment
    gate := b.evaluateGate(ctx, event, tier)

    // 4. KB-20 enrichment (best-effort, same as existing path)
    patientCtx, _ := b.kb20Client.FetchSummaryContext(ctx, event.PatientID)

    // 5. Build card with signal-specific fields
    card := &DecisionCard{
        CardSource:    "CLINICAL_SIGNAL",    // new CardSource enum value
        NodeID:        event.NodeID,
        PatientID:     event.PatientID,
        MCUGate:       gate,
        SafetyTier:    b.deriveSafetyTier(event.SafetyFlags),
        // ... map remaining fields from event
    }

    // 6. Persist + publish gate change to KB-19
    return card, nil
}
```

**HALT gate handling**: KB-23's `HysteresisEngine` currently handles SAFE/MODIFY/PAUSE transitions. MD-06 is the first source of HALT suggestions. The engine must be extended with a HALT rule: HALT always requires clinician reaffirmation (`pending_reaffirmation = true`), and cannot be auto-downgraded by hysteresis. This is consistent with existing PAUSE behavior but stricter.

## 5. KB-26 Twin State Integration: Actual API Mapping

### 5.1 Actual KB-26 TwinState Structure

KB-26's `TwinState` (`kb-26-metabolic-digital-twin/internal/models/twin_state.go`) stores Tier 3 estimates as JSONB `EstimatedVariable` structs:

```go
// Actual KB-26 types
type TwinState struct {
    // Tier 1: Direct measurements
    FBG7dMean, PPBG7dMean, HbA1c       *float64
    SBP14dMean, DBP14dMean              *float64
    EGFR                                *float64
    WaistCm, WeightKg, BMI             *float64
    DailySteps7dMean, RestingHR        *float64

    // Tier 2: Reliably derived
    VisceralFatProxy    float64         // 0-1
    VisceralFatTrend    string          // IMPROVING|STABLE|WORSENING
    RenalSlope          float64         // mL/min/1.73m²/year
    RenalClassification string          // G1-G5
    MAPValue            float64
    GlycemicVariability float64         // CV%

    // Tier 3: Estimated (JSONB)
    InsulinSensitivity   datatypes.JSON  // EstimatedVariable{Value, Classification, Confidence, Method}
    HepaticGlucoseOutput datatypes.JSON
    MuscleMassProxy      datatypes.JSON
    BetaCellFunction     datatypes.JSON
    SympatheticTone      datatypes.JSON
}

type EstimatedVariable struct {
    Value          float64 `json:"value"`
    Classification string  `json:"classification"`
    Confidence     float64 `json:"confidence"`
    Method         string  `json:"method"`
}
```

### 5.2 Variable Mapping: Spec Names → Actual KB-26 Fields

| Spec Variable | KB-26 Actual Field | Access Pattern | Notes |
|---|---|---|---|
| `IS` | `InsulinSensitivity` (JSONB) | Deserialize → `.Value` | Direct mapping |
| `VF` | `VisceralFatProxy` (float64) | Direct Tier 2 field | Already a plain float |
| `HGO` | `HepaticGlucoseOutput` (JSONB) | Deserialize → `.Value` | Direct mapping |
| `MM` | `MuscleMassProxy` (JSONB) | Deserialize → `.Value` | Direct mapping |
| **`VR`** | **Does not exist** | **Must be derived** | See Section 5.3 |
| **`RR`** | **Does not exist** | **Must be derived** | See Section 5.3 |

### 5.3 Missing KB-26 Variables: VR and RR

KB-26's ODE engine (`coupling_equations.go`) models VR and RR as SimState fields, but `TwinState` (the persisted API-facing model) does not expose them. Two options:

**Option A (Recommended): Extend KB-26 TwinState** — Add `VascularResistance` and `RenalReserve` as JSONB `EstimatedVariable` fields to `TwinState`. This is a KB-26 migration + derivation logic addition. Pre-requisite for MD-02 and MD-03.

**Option B: Derive in KB-22** — MD-02 derives vascular resistance from `SBP14dMean`, `DBP14dMean`, `MAPValue` (all in Tier 2). MD-03 derives renal trajectory from `EGFR`, `RenalSlope`, `RenalClassification` (all in Tier 2). No KB-26 changes needed but less accurate than the ODE-derived values.

**Decision**: Option A for accuracy, with Option B as graceful fallback when KB-26 fields are not yet populated. The `DataResolver` checks for the JSONB field; if null, falls back to Tier 2 derivation.

### 5.4 DataResolver KB-26 Integration

```go
// kb26_client.go — unwraps EstimatedVariable JSONB for MD node consumption

func (c *KB26Client) GetTwinState(ctx context.Context, patientID string) (*TwinStateView, error) {
    // GET /api/v1/kb26/twin/{patientID}
    resp := c.httpGet(ctx, fmt.Sprintf("/api/v1/kb26/twin/%s", patientID))

    // Deserialize JSONB EstimatedVariable fields into plain values
    return &TwinStateView{
        IS:           extractEstimated(resp.InsulinSensitivity),  // {Value, Confidence}
        VF:           resp.VisceralFatProxy,                       // already float
        VFTrend:      resp.VisceralFatTrend,
        HGO:          extractEstimated(resp.HepaticGlucoseOutput),
        MM:           extractEstimated(resp.MuscleMassProxy),
        VR:           extractEstimatedOrDerive(resp, "VR"),        // KB-26 field or Tier 2 fallback
        RR:           extractEstimatedOrDerive(resp, "RR"),        // KB-26 field or Tier 2 fallback
        RenalSlope:   resp.RenalSlope,
        EGFR:         resp.EGFR,
        GlycemicVar:  resp.GlycemicVariability,
        DailySteps:   resp.DailySteps7dMean,
        RestingHR:    resp.RestingHR,
    }, nil
}

// TwinStateView — flattened view for engine consumption
type TwinStateView struct {
    IS, HGO, MM     EstimatedValue   // {Value float64, Confidence float64}
    VF               float64
    VFTrend          string
    VR, RR           EstimatedValue   // from KB-26 JSONB or Tier 2 derivation
    RenalSlope       float64
    EGFR             *float64
    GlycemicVar      float64
    DailySteps       *float64
    RestingHR        *float64
    LastUpdated      time.Time        // from TwinState.UpdatedAt — staleness check
}
```

### 5.5 Twin State Staleness Handling

If KB-26's twin state hasn't been updated recently, derived values no longer reflect the patient's metabolic state. The DataResolver enforces a staleness check:

```go
const DefaultStalenessTreshold = 21 * 24 * time.Hour  // 21 days, configurable via KB26_STALENESS_DAYS

func (r *DataResolverImpl) checkStaleness(view *TwinStateView, resolved *ResolvedData) {
    age := time.Since(view.LastUpdated)
    if age > r.stalenessThreshold {
        // Downgrade to PARTIAL — MD nodes will still evaluate but with reduced confidence
        if resolved.Sufficiency == DataSufficient {
            resolved.Sufficiency = DataPartial
        }
        resolved.MissingFields = append(resolved.MissingFields,
            fmt.Sprintf("KB-26 twin state stale (last updated %d days ago)", int(age.Hours()/24)))
    }
}
```

This aligns with the Strategic Review's missing data degradation rules. MD nodes receiving PARTIAL sufficiency still evaluate (using the stale snapshot) but the `data_sufficiency` field in the emitted `ClinicalSignalEvent` signals to KB-23 that confidence should be reduced.
```

### 5.5 Trajectory Data: Twin State History

MD nodes need time-series data for trajectory computation. KB-26 exposes `GET /api/v1/kb26/twin/{patientId}/history?limit=N` which returns full `TwinState` snapshots.

**Approach**: `DataResolver` calls `/history?limit=90` (one per day over 90 days) and extracts the specific variable into a time series in-memory. This is bandwidth-inefficient but correct. A future KB-26 endpoint (`GET /api/v1/kb26/twin/{patientId}/variable-history?variable=IS&days=90`) can optimize this — but it is NOT a prerequisite for initial implementation.

```go
func (c *KB26Client) GetVariableHistory(ctx context.Context, patientID, variable string, days int) ([]TimeSeriesPoint, error) {
    // GET /api/v1/kb26/twin/{patientId}/history?limit={days}
    snapshots := c.getHistory(ctx, patientID, days)

    // Extract specific variable from each snapshot
    var series []TimeSeriesPoint
    for _, s := range snapshots {
        val := extractVariable(s, variable)  // e.g., extract IS from InsulinSensitivity JSONB
        if val != nil {
            series = append(series, TimeSeriesPoint{Timestamp: s.UpdatedAt, Value: *val})
        }
    }
    return series, nil
}
```

## 6. KB-20 Data Source Gap Analysis

### 6.1 Fields That Exist in KB-20

| PM Node Input | KB-20 Actual Field | Location |
|---|---|---|
| `sbp_home_mean` (PM-01) | `SBP7dMean` via BP trajectory | `lab_tracker.go` |
| `dbp_home_mean` (PM-01) | Derivable from BP readings | `lab_tracker.go` |
| `fbg_7d_mean` (PM-04, PM-06) | `LabTypeFBG` entries | `lab_tracker.go:38` |
| `hba1c` (MD-05) | `LabTypeHbA1c` entries | `lab_tracker.go:39` |
| `egfr` (MD-03) | `EGFR` + trajectory | `PatientProfile` |

### 6.2 Fields That Need New KB-20 Lab Types

| PM Node Input | Required Change | Priority |
|---|---|---|
| `sbp_nocturnal_mean`, `sbp_daytime_mean` (PM-03) | Add nocturnal/daytime BP tagging to lab entries. Alt: compute from timestamped BP readings (night = 22:00-06:00) | HIGH (PM-03 depends on this) |
| `ppbg_values` (PM-04) | Add `LabTypePPBG` constant + storage. FBG exists but PPBG does not | HIGH (PM-04, PM-05 depend on this) |
| `glucose_all_values` (PM-05) | Query combining FBG + PPBG lab entries | Derives from above |
| `homa_ir` (MD-01) | Add `LabTypeHOMA_IR` or compute from fasting insulin + FBG | LOW (optional input) |

### 6.3 Fields That Come From KB-26 (Not KB-20)

| PM Node Input | KB-26 Field |
|---|---|
| `daily_steps_7d_mean` (PM-07) | `TwinState.DailySteps7dMean` |
| `resting_hr_7d_mean` (PM-08) | `TwinState.RestingHR` |

### 6.4 Fields That Require New Data Sources

| PM Node Input | Source | Notes |
|---|---|---|
| `rmssd_7d_mean`, `rmssd_30d_mean` (PM-08) | Wearable device integration | PM-08 has `insufficient_data: SKIP` — graceful when unavailable |
| `exercise_tolerance_self` (PM-07) | Tier-1 check-in | Falls back to step count only when unavailable |
| All `TIER1_CHECKIN` sources (PM-02 symptoms) | Tier-1 adapter (not built) | DataResolver resolves from API/manual input today |

**DataResolver implementation**: When a `required_input` references a field that doesn't exist yet, DataResolver returns it as a missing field in `ResolvedData.MissingFields`. The engine checks `DataSufficiency` — if the field is non-optional and missing, the node evaluates as INSUFFICIENT and applies its `insufficient_data` policy.

## 7. YAML Node Schemas

### 7.1 MonitoringNodeDefinition (Layer 2)

```yaml
node_id: PM-03
version: "1.0.0"
type: MONITORING
title_en: "Nocturnal BP Dipping Classification"
title_hi: "Raat ka BP Dipping Vargikaran"

required_inputs:
  - field: sbp_nocturnal_mean
    source: KB-20
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

classifications:
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
  - category: EXTREME_DIPPER
    condition: "dipping_ratio < -0.20"
    severity: MODERATE
    mcu_gate_suggestion: MODIFY
    card_template: "dc-pm03-extreme-dipper-v1"
  - category: NORMAL_DIPPER
    condition: "dipping_ratio >= -0.20 AND dipping_ratio < -0.10"
    severity: NONE
    mcu_gate_suggestion: SAFE
    card_template: null

insufficient_data:
  action: SKIP
  note_en: "Insufficient BP readings for dipping classification"

safety_triggers:
  - id: PM03_ST01
    condition: "sbp_nocturnal_mean > 160"
    severity: URGENT
    action: "Nocturnal hypertension. Review evening antihypertensive timing."

cascade_to:
  - MD-02
  - MD-04

checkin_prompts:
  - prompt_id: PM03_CK01
    text_en: "Did you wear your BP monitor while sleeping last night?"
    text_hi: "Kya aapne raat mein sote samay BP monitor pehna tha?"
    response_type: BOOLEAN
    maps_to: nocturnal_monitoring_compliance
```

### 7.2 DeteriorationNodeDefinition (Layer 3)

```yaml
node_id: MD-01
version: "1.0.0"
type: DETERIORATION
title_en: "Insulin Resistance Trajectory Monitor"
title_hi: "Insulin Pratirodh Pravrutti Monitor"

# Logical name (mapped to KB-26 field by DataResolver — see Section 5.2)
state_variable: IS
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
    description: "Current insulin sensitivity estimate (JSONB EstimatedVariable.Value)"
  - field: twin_state.IS_confidence
    source: KB-26
    description: "Confidence in IS estimate (JSONB EstimatedVariable.Confidence)"
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
  method: LINEAR_REGRESSION
  window_days: 90
  min_data_points: 5
  rate_unit: "per_month"
  data_source: KB-26_HISTORY    # uses GET /twin/{id}/history, extracts IS time series

thresholds:
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
  - signal: IS_MODERATE_DECLINE
    condition: "rate_of_change < -0.04"
    severity: MODERATE
    trajectory: DECELERATING
    mcu_gate_suggestion: MODIFY
    card_template: "dc-md01-moderate-decline-v1"
    actions:
      - type: LIFESTYLE
        description: "Intensify structured exercise prescription"
        urgency: 72H
  - signal: IS_STABLE
    condition: "rate_of_change >= -0.04 AND rate_of_change <= 0.04"
    severity: NONE
    trajectory: STABLE
    mcu_gate_suggestion: SAFE
    card_template: null
  - signal: IS_IMPROVING
    condition: "rate_of_change > 0.04"
    severity: NONE
    trajectory: IMPROVING
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

### 7.3 Expression Evaluator: Restricted Grammar

Condition strings and computed field formulas use a restricted expression language:

```
Allowed operators: + - * / ( ) AND OR NOT > >= < <= == !=
Allowed operands:  field names (from required_inputs + computed_fields), numeric literals, string literals
Not allowed:       function calls, assignments, loops, imports, system calls
```

Implemented as a simple recursive-descent parser, not `eval()`. Since expressions come from YAML files on disk (not user input), injection risk is low, but the restricted grammar prevents accidental complexity in clinical rules.

## 8. Engine Architecture

### 8.1 New Files

```
internal/
├── models/
│   ├── monitoring_node.go          # MonitoringNodeDefinition struct
│   ├── deterioration_node.go       # DeteriorationNodeDefinition struct
│   ├── clinical_signal_event.go    # ClinicalSignalEvent + sub-types
│   └── data_input.go              # RequiredInput, ResolvedData, TwinStateView
│
├── services/
│   ├── monitoring_node_loader.go   # MonitoringNodeLoader (separate from NodeLoader)
│   ├── deterioration_node_loader.go # DeteriorationNodeLoader (separate from NodeLoader)
│   ├── monitoring_engine.go        # MonitoringNodeEngine
│   ├── deterioration_engine.go     # DeteriorationNodeEngine
│   ├── signal_cascade.go           # PM→MD + MD→MD cascade coordinator
│   ├── data_resolver.go            # Abstract data fetching from KB-20/KB-26/cache
│   ├── expression_evaluator.go     # Restricted expression parser + evaluator
│   ├── trajectory_computer.go      # Linear regression / Bayesian slope
│   ├── signal_publisher.go         # ClinicalSignalEvent → KB-23 + Kafka
│   └── kb26_client.go              # KB-26 twin state + history API client
│
├── api/
│   ├── event_ingestion_handlers.go # Webhooks for KB-20/KB-26 events
│   ├── signal_query_handlers.go    # Query latest signals per patient
│   └── signal_handler_group.go     # Sub-router for PM/MD endpoints (isolates DI)
│
monitoring/                         # PM node YAML files (9 files)
deterioration/                      # MD node YAML files (6 files)
```

### 8.2 Modified Files

| File | Change | Risk |
|---|---|---|
| `internal/api/server.go` | Add `SignalHandlerGroup` sub-router registration | Low — additive |
| `internal/config/config.go` | Add KB26_URL, new dir paths, debounce + timeout config | Low — additive |

**Note**: The existing `NodeLoader` (`internal/services/node_loader.go`) is NOT modified. PM and MD nodes get their own separate loaders (`MonitoringNodeLoader`, `DeteriorationNodeLoader`) to avoid touching the Bayesian validation logic. This addresses review finding I-1.

### 8.3 Server DI: Sub-Router Pattern

To manage DI complexity (review finding I-4), new engines are grouped into a `SignalHandlerGroup`:

```go
// signal_handler_group.go — isolated DI for PM/MD engines
type SignalHandlerGroup struct {
    monitoringLoader    *MonitoringNodeLoader
    deteriorationLoader *DeteriorationNodeLoader
    monitoringEngine    *MonitoringNodeEngine
    deteriorationEngine *DeteriorationNodeEngine
    cascade             *SignalCascade
    dataResolver        *DataResolver
    signalPublisher     *SignalPublisher
    db                  *gorm.DB
    cache               *redis.Client
}

func NewSignalHandlerGroup(db *gorm.DB, cache *redis.Client, config *Config) *SignalHandlerGroup {
    // Initialize all PM/MD dependencies in isolation
    // Shares db, cache, kafka from main Server
}

func (g *SignalHandlerGroup) RegisterRoutes(router *gin.RouterGroup) {
    // /api/v1/events/*
    // /api/v1/patients/:id/signals*
    // /api/v1/nodes/monitoring*
    // /api/v1/nodes/deterioration*
}
```

The main `Server.InitServices()` creates `SignalHandlerGroup` and calls `RegisterRoutes()` — one line added to existing code.

### 8.4 Core Interfaces

```go
type DataResolver interface {
    Resolve(ctx context.Context, patientID string, inputs []RequiredInput) (*ResolvedData, error)
}

type ResolvedData struct {
    Fields          map[string]float64
    FieldTimestamps map[string]time.Time
    Sufficiency     DataSufficiency
    MissingFields   []string
    Sources         map[string]string    // field → actual source used
}

type DataSufficiency string
const (
    DataSufficient   DataSufficiency = "SUFFICIENT"
    DataPartial      DataSufficiency = "PARTIAL"      // some optional fields missing
    DataInsufficient DataSufficiency = "INSUFFICIENT"  // required fields missing
)
```

### 8.5 MonitoringNodeEngine Flow

1. Receive trigger (KB-20 observation event or API call)
2. `MonitoringNodeLoader.Get(nodeID)`
3. `DataResolver.Resolve(patientID, node.RequiredInputs)` — fetches from KB-20/KB-26/cache
4. Check `DataSufficiency` — if INSUFFICIENT, apply `node.insufficient_data` policy
5. Evaluate `computed_fields` via `ExpressionEvaluator`
6. Evaluate `classifications` top-to-bottom, first match wins
7. Evaluate `safety_triggers` (reuses condition expression parser)
8. Build `ClinicalSignalEvent` with `SignalType = MONITORING_CLASSIFICATION`
9. Publish to KB-23 + Kafka via `SignalPublisher`
10. `SignalCascade.Trigger(nodeID, patientID, classificationResult)` — invoke subscribed MD nodes

### 8.6 DeteriorationNodeEngine Flow

1. Receive trigger (KB-20/KB-26 event OR internal cascade from PM node)
2. `DeteriorationNodeLoader.Get(nodeID)`
3. `DataResolver.Resolve(patientID, node.RequiredInputs)` — includes KB-26 twin state (JSONB unwrapped)
4. Check `DataSufficiency` — if INSUFFICIENT, apply policy (FLAG_FOR_REVIEW / USE_SNAPSHOT)
5. `TrajectoryComputer.Compute()` — fetch variable history from KB-26, run linear regression
6. Evaluate `thresholds` top-to-bottom, first match wins
7. Compute `projections` if configured
8. Build `ClinicalSignalEvent` with `SignalType = DETERIORATION_SIGNAL`
9. Publish to KB-23 + Kafka via `SignalPublisher`

### 8.7 Signal Cascade Coordinator

Two-pass evaluation:

```go
type SignalCascade struct {
    pmToMD    map[string][]string    // built from PM node cascade_to fields
    mdToMD06  map[string]bool        // built from MD-06 contributing_signals
    deterEngine *DeteriorationNodeEngine
}

func (sc *SignalCascade) Trigger(ctx, sourceNodeID, patientID, classification) []*ClinicalSignalEvent {
    // Pass 1: PM → MD (MD-01 through MD-05)
    pass1Results := map[string]*ClinicalSignalEvent{}
    for _, mdNodeID := range sc.pmToMD[sourceNodeID] {
        event, err := sc.deterEngine.Evaluate(ctx, mdNodeID, patientID, cascadeCtx)
        if err != nil { continue }  // cascade failures non-fatal
        if event != nil { pass1Results[mdNodeID] = event }
    }

    // Pass 2: MD → MD-06 (if any Pass 1 node feeds MD-06)
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

### 8.8 Debounce Strategy

Per-patient per-node Redis cache with 5-minute TTL:

```
Key:   "eval:{patientID}:{nodeID}"
TTL:   SIGNAL_DEBOUNCE_TTL_SEC (default 300)

On event:
  1. Check cache → if exists → SKIP (unless safety trigger condition met)
  2. If miss → evaluate → set cache key
```

### 8.9 Concurrency and Request Lifecycle

Event ingestion endpoints return `202 Accepted` immediately. Evaluation runs in a background goroutine:

```go
func (h *EventIngestionHandler) handleObservation(c *gin.Context) {
    var event ObservationEvent
    if err := c.ShouldBindJSON(&event); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    c.Status(202)  // return immediately

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        h.evaluateMatchingNodes(ctx, event)  // PM eval → cascade → MD eval → publish
    }()
}
```

**Worst-case latency estimate**: A single observation triggers PM-01 + PM-03 → MD-02, MD-04, MD-06 = 5 evaluations. Each evaluation involves 1 DataResolver call (~100ms with cache) + expression eval (~1ms) + DB write (~10ms). Total: ~600ms. Well within the 30-second goroutine timeout.

## 9. API Endpoints

### 9.1 Event Ingestion

```
POST /api/v1/events/observation
  Body: { event_type, patient_id, observation_code, value, unit, timestamp, source }
  Response: 202 Accepted
  Async: route to matching PM/MD nodes → evaluate → cascade → publish

POST /api/v1/events/twin-state-update
  Body: { event_type, patient_id, state_version, changed_variables, update_source, timestamp }
  Response: 202 Accepted
  Async: route to MD nodes with matching trigger_on → evaluate → publish

POST /api/v1/events/checkin-response
  Body: { event_type, patient_id, prompt_id, response_value, response_type, timestamp }
  Response: 202 Accepted
  Future: Tier-1 adapter sends check-in answers here
```

### 9.2 Signal Queries

```
GET /api/v1/patients/:id/signals
  Latest signal per node for patient (from clinical_signals_latest)

GET /api/v1/patients/:id/signals/:nodeId?limit=10
  Signal history for specific node

GET /api/v1/patients/:id/deterioration-summary
  All MD node latest results (6 signals + projections)
```

### 9.3 Node Management

```
GET  /api/v1/nodes/monitoring
GET  /api/v1/nodes/monitoring/:nodeId
GET  /api/v1/nodes/deterioration
GET  /api/v1/nodes/deterioration/:nodeId
POST /internal/nodes/reload           # extended to reload PM/MD YAML
```

## 10. Database Changes

### Migration: `006_monitoring_deterioration.sql`

Depends on migrations 001-005 being applied. No other migration 006 should be created concurrently on this branch.

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

## 11. Complete 14-Node Inventory

### 11.1 Layer 2: Physiological Monitoring Nodes

#### PM-01: Home BP Monitoring
- **Purpose**: Classify home BP readings against stratum-specific targets
- **Inputs**: `sbp_home_mean` (KB-20 `SBP7dMean`, 7d), `dbp_home_mean` (KB-20, 7d), `bp_target_sbp` (KB-20 stratum)
- **Computed**: `sbp_delta = sbp_home_mean - bp_target_sbp`
- **Classifications**: SEVERELY_ABOVE (delta>30, CRITICAL/PAUSE) → ABOVE_TARGET (delta>10, MODERATE/MODIFY) → AT_TARGET (|delta|<=10, NONE/SAFE) → BELOW_TARGET (delta<-10, MODERATE/MODIFY) → HYPOTENSIVE (sbp<90, CRITICAL/PAUSE)
- **Safety**: SBP > 180 → IMMEDIATE
- **Cascade**: MD-02, MD-06

#### PM-02: Daily Symptom Check-in
- **Purpose**: Structured daily screening mapped to Layer 1 P-nodes
- **Inputs**: `symptom_headache`, `symptom_dizziness`, `symptom_chest_pain`, `symptom_fatigue`, `symptom_oedema` (all TIER1_CHECKIN, BOOLEAN)
- **Computed**: `symptom_count = sum(all booleans)`
- **Classifications**: MULTI_SYMPTOM (count>=3, MODERATE/MODIFY) → SINGLE_SYMPTOM (count 1-2, MILD/SAFE) → ASYMPTOMATIC (0, NONE/SAFE)
- **Safety**: `symptom_chest_pain = true` → IMMEDIATE (auto-trigger P01 HPI session)
- **Cascade**: MD-04, MD-06

#### PM-03: Nocturnal BP Dipping Pattern
- **Purpose**: Classify nocturnal dipping from ABPM or home night readings
- **Inputs**: `sbp_nocturnal_mean` (KB-20 timestamped BP, 14d, night=22:00-06:00), `sbp_daytime_mean` (KB-20, 14d)
- **KB-20 gap**: Requires nocturnal/daytime BP tagging or timestamp-based derivation (see Section 6.2)
- **Computed**: `dipping_ratio = (nocturnal - daytime) / daytime`
- **Classifications**: REVERSE_DIPPER (ratio>0, CRITICAL/PAUSE) → NON_DIPPER (ratio>=-0.10, MODERATE/MODIFY) → EXTREME_DIPPER (ratio<-0.20, MODERATE/MODIFY) → NORMAL_DIPPER (-0.20 to -0.10, NONE/SAFE)
- **Safety**: `sbp_nocturnal_mean > 160` → URGENT
- **Cascade**: MD-02, MD-04

#### PM-04: Postprandial Blood Glucose Pattern
- **Purpose**: Classify PPBG excursion patterns
- **Inputs**: `ppbg_values` (KB-20, 14d, array)
- **KB-20 gap**: Requires new `LabTypePPBG` constant (see Section 6.2)
- **Computed**: `ppbg_mean`, `excursion = ppbg_mean - fbg_7d_mean`
- **Classifications**: SEVERE_EXCURSION (excursion>80, CRITICAL/PAUSE) → HIGH_EXCURSION (50-80, MODERATE/MODIFY) → MODERATE_EXCURSION (30-50, MILD/SAFE) → NORMAL_EXCURSION (<30, NONE/SAFE)
- **Safety**: Any single `ppbg > 300` → URGENT
- **Cascade**: MD-01, MD-05

#### PM-05: Glycemic Variability
- **Purpose**: CV% of glucose readings to detect instability
- **Inputs**: `glucose_all_values` (KB-20, 14d, combined FBG+PPBG query)
- **Computed**: `glucose_cv = stdev / mean * 100`
- **Classifications**: HIGHLY_VARIABLE (cv>36, MODERATE/MODIFY) → MODERATELY_VARIABLE (20-36, MILD/SAFE) → STABLE (cv<20, NONE/SAFE)
- **Safety**: `glucose_cv > 50` → URGENT
- **Cascade**: MD-01, MD-05
- **Evidence**: 36% CV threshold from International Consensus on CGM (Danne et al. 2017)

#### PM-06: FBG Trend Computation
- **Purpose**: FBG trajectory — purely algorithmic, no patient interaction
- **Inputs**: `fbg_values` (KB-20 `LabTypeFBG`, 90d array with timestamps)
- **Computed**: `fbg_slope` (linear regression, mg/dL per month)
- **Classifications**: RAPIDLY_RISING (slope>5/month, MODERATE/MODIFY) → GRADUALLY_RISING (2-5, MILD/SAFE) → STABLE (|slope|<2, NONE/SAFE) → IMPROVING (slope<-2, NONE/SAFE)
- **Safety**: `fbg_current_mean > 250` → URGENT
- **Cascade**: MD-01, MD-05

#### PM-07: Exercise Capacity / Muscle Function
- **Purpose**: Track functional capacity via step count trends
- **Inputs**: `daily_steps_7d_mean` (KB-26 `TwinState.DailySteps7dMean`), `daily_steps_30d_mean` (KB-26 history), `exercise_tolerance_self` (TIER1_CHECKIN, LIKERT_5, optional)
- **Computed**: `steps_trend = (7d_mean - 30d_mean) / 30d_mean`
- **Classifications**: DECLINING_RAPIDLY (trend<-0.25, MODERATE/MODIFY) → DECLINING (-0.25 to -0.10, MILD/SAFE) → STABLE (|trend|<0.10, NONE/SAFE) → IMPROVING (trend>0.10, NONE/SAFE)
- **Safety**: `steps_7d < 500 AND exercise_tolerance <= 2` → URGENT
- **Cascade**: MD-04, MD-06

#### PM-08: Heart Rate Variability Pattern
- **Purpose**: HRV trend from wearable data as autonomic health proxy
- **Inputs**: `rmssd_7d_mean` (DEVICE), `rmssd_30d_mean` (DEVICE), `resting_hr_7d_mean` (KB-26 `TwinState.RestingHR`)
- **Data gap**: RMSSD requires wearable device integration (not yet built)
- **Computed**: `hrv_trend = (7d - 30d) / 30d`
- **Classifications**: SEVERELY_DEPRESSED (rmssd<15 AND hr>90, MODERATE/MODIFY) → DECLINING (trend<-0.20, MILD/SAFE) → STABLE (|trend|<0.20, NONE/SAFE) → IMPROVING (trend>0.20, NONE/SAFE)
- **Safety**: None
- **Cascade**: MD-04
- **Insufficient data**: SKIP (many patients lack wearable HRV)

#### PM-09: Sleep Quality Screen
- **Purpose**: Weekly 3-question sleep quality assessment as autonomic health proxy (replaces HRV when wearable data unavailable)
- **Inputs**: `sleep_difficulty` (TIER1_CHECKIN, LIKERT_5), `sleep_duration_hrs` (TIER1_CHECKIN, numeric), `sleep_disruptions` (TIER1_CHECKIN, BOOLEAN — "Did you wake up more than twice last night?")
- **Computed**: `sleep_score = 0.40 * normalize(sleep_difficulty, 1, 5) + 0.35 * (1 - normalize(sleep_duration_hrs, 4, 9)) + 0.25 * sleep_disruptions` (where normalize maps to 0-1, higher = worse)
- **Classifications**: SEVERELY_DISRUPTED (sleep_score > 0.75, MODERATE/MODIFY) → POOR_QUALITY (0.50-0.75, MILD/SAFE) → ADEQUATE (0.25-0.50, NONE/SAFE) → GOOD (sleep_score < 0.25, NONE/SAFE)
- **Safety**: None (trend indicator, not acute)
- **Cascade**: MD-04 (Autonomic Dysfunction)
- **Checkin**: 3 weekly questions: "How difficult was it to fall asleep this week?" (Likert 1-5), "How many hours of sleep did you get last night?" (numeric), "Did you wake up more than twice last night?" (Y/N)
- **Evidence**: Sleep quality correlates with BP dipping (PM-03), autonomic dysfunction (MD-04), and glycemic variability (PM-05). The 2025 digital biomarker review (van den Brink et al., SAGE Journals) identifies sleep as a window into cardiometabolic health. PM-09 provides the autonomic proxy signal that MD-04 needs when PM-08 HRV data is unavailable.
- **Min observations**: 1 weekly check-in
- **Insufficient data**: SKIP (falls back to PM-08 HRV if available, or MD-04 evaluates with remaining signals)

### 11.2 Layer 3: Metabolic Deterioration Nodes

#### MD-01: Insulin Resistance Trajectory
- **KB-26 field**: `InsulinSensitivity` (JSONB EstimatedVariable → extract `.Value`)
- **Trigger**: OBSERVATION:FBG, OBSERVATION:PPBG, TWIN_STATE_UPDATE, SIGNAL:PM-04/05/06, PROTOCOL:M3-PRP:ADHERENCE
- **Trajectory**: Linear regression on IS history (KB-26 `/history`) over 90d
- **Thresholds**: IS_CRITICAL_DECLINE (rate<-0.08 AND IS<0.30, CRITICAL/PAUSE) → IS_MODERATE_DECLINE (rate<-0.04, MODERATE/MODIFY) → IS_STABLE (|rate|<=0.04, SAFE) → IS_IMPROVING (rate>0.04, SAFE)
- **Projection**: When will IS cross 0.20?
- **Contributing**: PM-04, PM-05, PM-06

#### MD-02: Vascular Compliance Decline
- **KB-26 field**: `VascularResistance` (JSONB, requires KB-26 extension — see Section 5.3)
- **Fallback**: Derive from `SBP14dMean`, `DBP14dMean`, `MAPValue` (all KB-26 Tier 2)
- **Trigger**: OBSERVATION:SBP, OBSERVATION:DBP, TWIN_STATE_UPDATE, SIGNAL:PM-01/03
- **Trajectory**: Linear regression on VR over 90d
- **Thresholds**: VR_CRITICAL_RISE (rate>0.08 AND VR>0.70, CRITICAL/PAUSE) → VR_MODERATE_RISE (rate>0.04, MODERATE/MODIFY) → VR_STABLE (|rate|<=0.04, SAFE) → VR_IMPROVING (rate<-0.04, SAFE)
- **Contributing**: PM-01, PM-03

#### MD-03: Renal Function Trajectory
- **KB-26 field**: `RenalReserve` (JSONB, requires KB-26 extension — see Section 5.3)
- **Fallback**: Use `EGFR` + `RenalSlope` + `RenalClassification` (all KB-26 Tier 2)
- **Trigger**: OBSERVATION:EGFR, OBSERVATION:CREATININE, TWIN_STATE_UPDATE
- **Trajectory**: Uses KB-26 pre-computed `RenalSlope` directly (no additional regression needed)
- **Thresholds**: RR_RAPID_DECLINE (slope<-5/yr, CRITICAL/PAUSE) → RR_PROGRESSIVE_DECLINE (-5 to -3, MODERATE/MODIFY) → RR_SLOW_DECLINE (-3 to -1, MILD/SAFE) → RR_STABLE (|slope|<1, SAFE)
- **Projection**: When will eGFR cross next CKD stage boundary?
- **Evidence**: KDIGO 2024 rapid decline = >5 mL/min/1.73m²/year

#### MD-04: Autonomic Dysfunction Progression
- **KB-26 field**: None — composite signal from PM nodes
- **Trigger**: SIGNAL:PM-03, SIGNAL:PM-07, SIGNAL:PM-08, SIGNAL:PM-09, TWIN_STATE_UPDATE, PROTOCOL:M3-VFRP:ACTIVITY
- **Computed**: `autonomic_score = 0.30*pm03 + 0.15*pm08 + 0.20*pm07 + 0.15*pm09 + 0.10*orthostatic + 0.10*m3_vfrp_activity`
  - When PM-08 (HRV) unavailable: redistribute weight to PM-09 (sleep): `0.30*pm03 + 0.30*pm09 + 0.20*pm07 + 0.10*orthostatic + 0.10*m3_vfrp`
  - When PM-09 (sleep) unavailable: redistribute to PM-08: `0.30*pm03 + 0.30*pm08 + 0.20*pm07 + 0.10*orthostatic + 0.10*m3_vfrp`
  - When both PM-08 and PM-09 unavailable: `0.40*pm03 + 0.30*pm07 + 0.15*orthostatic + 0.15*m3_vfrp`
- **Thresholds**: AUTONOMIC_SEVERE (>=2.5, CRITICAL/PAUSE) → AUTONOMIC_MODERATE (1.5-2.5, MODERATE/MODIFY) → AUTONOMIC_MILD (0.5-1.5, MILD/SAFE) → AUTONOMIC_NORMAL (<0.5, SAFE)
- **Contributing**: PM-03, PM-07, PM-08, PM-09, M3-VFRP

#### MD-05: Glycemic Control Deterioration
- **KB-26 field**: `HepaticGlucoseOutput` (JSONB EstimatedVariable → extract `.Value`)
- **Trigger**: OBSERVATION:FBG, OBSERVATION:PPBG, OBSERVATION:HBA1C, TWIN_STATE_UPDATE, SIGNAL:PM-04/05/06
- **Computed**: `glycemic_composite = 0.35*normalize(hba1c,6,12) + 0.30*normalize(fbg,90,250) + 0.20*pm04 + 0.15*pm05`
- **Thresholds**: GLYCEMIC_CRITICAL (composite>0.80 AND accelerating, CRITICAL/PAUSE) → GLYCEMIC_WORSENING (0.60-0.80, MODERATE/MODIFY) → GLYCEMIC_SUBOPTIMAL (0.40-0.60, MILD/SAFE) → GLYCEMIC_CONTROLLED (<0.40, SAFE)
- **Projection**: When will HbA1c cross 7.0%, 8.0%, 9.0%?
- **Contributing**: PM-04, PM-05, PM-06

#### MD-06: Cardiovascular Risk Emergence
- **KB-26 fields**: `VisceralFatProxy` (Tier 2 float), `InsulinSensitivity` (JSONB), `VascularResistance` (JSONB or fallback)
- **Trigger**: SIGNAL:MD-01/02/03, SIGNAL:PM-01/02, TWIN_STATE_UPDATE, PROTOCOL:M3-VFRP:ACTIVITY
- **Computed**: `cv_risk_score = 0.25*md01 + 0.25*md02 + 0.20*md03 + 0.15*pm01 + 0.15*(VF*3)`
- **Thresholds**: CV_RISK_CRITICAL (>=2.5, CRITICAL/HALT) → CV_RISK_HIGH (1.8-2.5, MODERATE/PAUSE) → CV_RISK_ELEVATED (1.0-1.8, MILD/MODIFY) → CV_RISK_LOW (<1.0, SAFE)
- **Note**: Only node that can suggest HALT. KB-23 hysteresis must handle HALT (always requires clinician reaffirmation, cannot auto-downgrade). Second-order cascade (MD→MD).
- **Contributing**: MD-01, MD-02, MD-03, PM-01, PM-02

## 12. Cascade Dependency Graph

```
Layer 2 (PM)               Layer 3 (MD)             External Protocols
────────────               ──────────────           ──────────────────
PM-01 (Home BP)      ──→   MD-02 (Vascular)    ──┐
                     ──→   MD-06 (CV Risk)    ◄──┤
PM-02 (Symptoms)     ──→   MD-04 (Autonomic)    │
                     ──→   MD-06               ◄──┤
PM-03 (Dipping)      ──→   MD-02               ──┤
                     ──→   MD-04                  │
PM-04 (PPBG)         ──→   MD-01 (Insulin Res)──┤  M3-PRP ──→ MD-01
                     ──→   MD-05 (Glycemic)   ──┤
PM-05 (Glyc Var)     ──→   MD-01              ──┤
                     ──→   MD-05              ──┤
PM-06 (FBG Trend)    ──→   MD-01              ──┤
                     ──→   MD-05              ──┤
PM-07 (Exercise)     ──→   MD-04                │
PM-08 (HRV)          ──→   MD-04                │  M3-VFRP ──→ MD-04
PM-09 (Sleep)        ──→   MD-04                │  M3-VFRP ──→ MD-06
                                                │
                     MD-01 ──→ MD-06 ◄──────────┘
                     MD-02 ──→ MD-06
                     MD-03 ──→ MD-06
```

Two-pass cascade: Pass 1 evaluates PM→MD (MD-01 through MD-05). Pass 2 evaluates MD→MD-06. M3 protocol signals enter as external triggers alongside KB-20 observations — they do not pass through PM nodes.

## 13. Configuration

New environment variables:

```
KB26_URL=http://localhost:8137
KB26_TIMEOUT_MS=5000
KB20_OBSERVATION_TIMEOUT_MS=10000          # longer than default 40ms for lookback queries
MONITORING_NODES_DIR=/app/monitoring
DETERIORATION_NODES_DIR=/app/deterioration
SIGNAL_DEBOUNCE_TTL_SEC=300
SIGNAL_PUBLISHER_RETRY_COUNT=3
SIGNAL_PUBLISHER_RETRY_DELAY_SEC=30
KAFKA_SIGNAL_TOPIC=clinical.signal.events  # new Kafka topic
KB26_STALENESS_DAYS=21                     # twin state staleness threshold
```

Note: KB-20's existing timeout of 40ms (`KB20TimeoutMS` in config.go) is insufficient for DataResolver lookback queries (90 days of observations). The new `KB20_OBSERVATION_TIMEOUT_MS` provides a separate, longer timeout for PM/MD data resolution.

## 14. Testing Strategy

### Unit Tests
- `monitoring_engine_test.go`: Classification logic for each PM node category
- `deterioration_engine_test.go`: Trajectory computation + threshold evaluation
- `signal_cascade_test.go`: Two-pass cascade ordering, non-fatal failure handling
- `data_resolver_test.go`: Multi-source resolution, KB-26 JSONB unwrapping, insufficiency detection
- `expression_evaluator_test.go`: Condition parsing, computed fields, restricted grammar enforcement
- `trajectory_computer_test.go`: Linear regression accuracy, min data point enforcement
- `monitoring_node_loader_test.go`: YAML parsing + validation
- `deterioration_node_loader_test.go`: YAML parsing + validation, cascade DAG validation

### Integration Tests
- Full PM→MD cascade: PM-04 classification → MD-01 evaluation → ClinicalSignalEvent published
- KB-26 twin state update → MD-01 through MD-06 evaluation
- Debounce: rapid events → single evaluation
- Insufficient data: graceful degradation per node policy

### Backward Compatibility Requirement
All existing KB-22 unit tests (`*_test.go` in `internal/services/` and `internal/api/`) must pass without modification after these changes. The new code is additive — if any existing test breaks, the change is rejected.

### Golden Dataset
- `calibration/pm_golden_dataset.json`: Known BP readings → expected PM-01/PM-03 classifications
- `calibration/md_golden_dataset.json`: Known KB-26 state trajectories → expected MD signals

## 15. KB-23 Changes Required

### 15.1 New Endpoint

```
POST /api/v1/clinical-signals
  Body: ClinicalSignalEvent
  Response: 201 Created (DecisionCard) | 204 No Content (no card needed)
```

### 15.2 Signal Card Builder

New `SignalCardBuilder` service (separate from existing `CardBuilder`):

1. **Template selection**: by `node_id` + `classification_category` or `deterioration_signal`. Templates in `templates/monitoring/` and `templates/deterioration/`.
2. **Confidence tier derivation**: maps `data_sufficiency` + `severity` to existing `ConfidenceTier` enum (FIRM/PROBABLE/POSSIBLE/UNCERTAIN).
3. **MCU gate evaluation**: uses `mcu_gate_suggestion` from event, applies existing `HysteresisEngine` for oscillation prevention.
4. **HALT gate rule**: HALT (from MD-06) always sets `pending_reaffirmation = true`. Cannot be auto-downgraded by hysteresis. Requires clinician resume via existing `POST /api/v1/cards/:id/mcu-gate-resume`.
5. **KB-20 enrichment**: best-effort patient context, same as existing path.
6. **Card persistence + KB-19 publishing**: reuses existing `CardLifecycle` and `KB19Publisher`.

### 4.4 HALT Gate Rules for HysteresisEngine Extension

HALT is not a "more severe PAUSE" — it means stop all automated medication adjustments until a physician explicitly resumes. The existing `HysteresisEngine` handles SAFE→MODIFY→PAUSE transitions with cooldown timers. HALT requires different semantics:

1. **Source restriction**: HALT can only be set by MD-06 (CV Risk Emergence) or by manual physician action. PM nodes and MD-01 through MD-05 can suggest at most PAUSE — never HALT.
2. **No automatic downgrade**: HALT cannot be downgraded by hysteresis timer expiry. Only explicit physician action via `POST /api/v1/cards/:id/mcu-gate-resume` can transition from HALT to a lower gate.
3. **V-MCU enforcement**: While HALT is active, every `GET /patients/:id/mcu-gate` query returns `HALT`. This is consistent with V-MCU's Three-Channel Safety Architecture (SA-01) where `final_gate = MostRestrictive(MCU_GATE, PHYSIO_GATE, PROTOCOL_GATE)`.
4. **Mandatory clinician card**: HALT triggers an URGENT KB-23 Decision Card with `pending_reaffirmation = true` that cannot be auto-acknowledged. The SLA scanner monitors unacknowledged HALT cards.
5. **Gate transition rules**:
   - `* → HALT`: Only from MD-06 signal or physician manual action
   - `HALT → PAUSE`: Only via physician resume action
   - `HALT → MODIFY/SAFE`: Not allowed — must step through PAUSE first

### 15.3 New Template Categories

- `templates/monitoring/dc-pm01-*.yaml` through `dc-pm08-*.yaml`
- `templates/deterioration/dc-md01-*.yaml` through `dc-md06-*.yaml`

Template authoring is a separate clinical task — this spec defines the event contract.

## 16. Migration Path for Existing HPICompleteEvent

The existing `OutcomePublisher` continues publishing `HPICompleteEvent` to KB-23's `POST /api/v1/decision-cards`. No breaking changes.

Future Layer 1 migration (optional, deferred):
1. Add adapter: `HPICompleteEvent` → `ClinicalSignalEvent` with `SignalType = BAYESIAN_DIFFERENTIAL`
2. Map types: `uuid.UUID` → string, `SafetyFlagSummary` → `SignalSafetyFlag`, `float64` → `*float64`
3. Switch OutcomePublisher target to `/api/v1/clinical-signals`
4. KB-23 routes by `signal_type`

## 17. Pre-requisites in Other Services

| Service | Change Required | Priority | Blocks |
|---|---|---|---|
| KB-26 | Add `VascularResistance`, `RenalReserve` JSONB fields to TwinState + derivation from SimState | HIGH | MD-02, MD-03 (without this, they use Tier 2 fallback) |
| KB-20 | Add `LabTypePPBG` constant + storage | HIGH | PM-04, PM-05 |
| KB-20 | Add timestamped BP entry support (for nocturnal/daytime derivation) | MEDIUM | PM-03 (can derive from timestamps if entries have time) |
| KB-23 | Add `POST /api/v1/clinical-signals` + `SignalCardBuilder` | HIGH | All PM/MD card generation |
| KB-23 | Add HALT gate rules to HysteresisEngine (Section 4.4) | HIGH | MD-06 HALT suggestion |
| KB-23 | Add `CLINICAL_SIGNAL` to `CardSource` enum (`enums.go`) | LOW | Signal-generated cards |
| KB-23 | Author monitoring + deterioration card templates | MEDIUM | Card content (engine works without, just no cards generated) |
| KB-26 | Add Kafka consumer for `clinical.signal.events` → MRI recomputation (Section 3.2) | MEDIUM | MRI domain scores won't reflect PM/MD signals until built |

## 18. Risk Assessment

| Risk | Likelihood | Mitigation |
|---|---|---|
| KB-26 VR/RR fields not added in time | Medium | DataResolver Tier 2 fallback (derive from SBP/DBP/MAP for VR, EGFR/RenalSlope for RR) |
| KB-20 PPBG lab type not added in time | Medium | PM-04/PM-05 evaluate as INSUFFICIENT, apply SKIP policy. Other nodes unaffected. |
| KB-26 history endpoint too slow for 90d queries | Low | Cache fetched history in Redis (TTL = debounce period). Future: request KB-26 variable-specific history endpoint. |
| Debounce too aggressive (misses rapid deterioration) | Low | Safety trigger bypass: skip debounce if any safety condition matches |
| Cascade loop | None | MD-06 has no `cascade_to` — acyclic by design. Loader validates DAG at startup. |
| Clinical threshold values wrong | Medium | All thresholds in YAML, hot-reloadable. Golden dataset calibration. Clinical team review. |
| PM nodes generate noise from insufficient data | Medium | `insufficient_data` policy per node. `data_sufficiency` in every signal. |
| Existing Layer 1 regression | Low | Backward compatibility requirement: all existing tests must pass unchanged. |
