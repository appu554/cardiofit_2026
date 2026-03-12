# V-MCU Three-Channel Safety Architecture — Implementation Plan

**Source**: `Vaidshala_Three_Channel_Safety_Architecture.docx` (SA-01 through SA-06)
**Scope**: Channel B, Channel C, Safety Arbiter, SafetyTrace, HOLD_DATA, Latency Budget
**Prerequisite**: Channel A (KB-23 MCU_GATE) is complete. No KB-23 changes required.
**Language**: Go (matches KB-20 through KB-23 pattern)
**Port**: 8135 (next in sequence after KB-23 on 8134)

---

## Architecture Overview

```
V-MCU receives three independent gate signals before every dose output:

  Channel A: MCU_GATE       ← KB-23 Redis cache (GET /patients/:id/mcu-gate, <5ms)
  Channel B: PHYSIO_GATE    ← PhysiologySafetyMonitor (in-process, raw labs, <10ms)
  Channel C: PROTOCOL_GATE  ← ProtocolGuard (in-process, compiled rules, <2ms)
                │
                ▼
         Safety Arbiter (pure function, 1oo3 veto, <1ms)
                │
                ▼
         SafetyTrace (async PostgreSQL write, <5ms non-blocking)
                │
                ▼
         Titration Algorithm (only if FinalGate permits)
```

**Total synchronous overhead**: < 18ms per titration cycle.

---

## Phase 0: Project Scaffold & Infrastructure

**Duration**: 1 session | **Dependencies**: None

### Tasks

#### 0.1 Create V-MCU service directory

```
backend/shared-infrastructure/knowledge-base-services/v-mcu/
├── cmd/server/main.go
├── internal/
│   ├── api/
│   │   ├── server.go
│   │   ├── routes.go
│   │   ├── health_handlers.go
│   │   └── titration_handlers.go
│   ├── arbiter/                    ← Safety Arbiter (SA-01)
│   ├── cache/                      ← Local safety data cache
│   ├── channel_a/                  ← KB-23 MCU_GATE client
│   ├── channel_b/                  ← PhysiologySafetyMonitor (SA-02)
│   ├── channel_c/                  ← ProtocolGuard (SA-03)
│   ├── config/
│   ├── database/
│   ├── metrics/
│   ├── models/
│   └── titration/                  ← Titration algorithm (post-arbiter)
├── protocol_rules.yaml             ← Channel C compiled rules
├── migrations/
│   └── 001_initial_schema.sql
├── tests/
├── go.mod
├── go.sum
├── Dockerfile
└── docker-compose.yml
```

#### 0.2 Initialize go.mod

```
module v-mcu

go 1.22

require (
    github.com/gin-gonic/gin
    github.com/google/uuid
    go.uber.org/zap
    gorm.io/gorm
    gorm.io/driver/postgres
    github.com/redis/go-redis/v9
    github.com/prometheus/client_golang
    github.com/shopspring/decimal
    gopkg.in/yaml.v3
)
```

#### 0.3 Config — environment-based, matching KB-23 pattern

```go
// internal/config/config.go
type Config struct {
    Port        string
    Environment string

    // Database (SafetyTrace + dose history)
    DatabaseURL      string
    DBMaxConnections int

    // Redis (local safety cache)
    RedisURL      string
    RedisPassword string

    // Cross-KB URLs
    KB19URL string // Protocol Orchestrator (events)
    KB20URL string // Patient Profile (raw labs)
    KB23URL string // Decision Cards (MCU_GATE)

    // Cross-KB Timeouts
    KB20TimeoutMS int // 200ms default
    KB23TimeoutMS int // 100ms default (Redis-backed)

    // Channel B thresholds (SA-02)
    GlucoseHaltThreshold   float64 // 3.9 mmol/L
    GlucosePauseThreshold  float64 // 4.5 mmol/L
    CreatinineDeltaHalt    float64 // 26 µmol/L in 48h
    PotassiumLowHalt       float64 // 3.0 mEq/L
    PotassiumHighHalt      float64 // 6.0 mEq/L
    SBPHaltThreshold       float64 // 90 mmHg
    WeightDeltaPause       float64 // 2.5 kg in 72h
    GlucoseTrendThreshold  float64 // 5.5 mmol/L for declining trend

    // Channel B data anomaly thresholds (SA-05)
    EGFRDeltaHoldData       float64 // 40% in 48h
    CreatinineDeltaHoldData float64 // 100% in 48h
    GlucoseFloorHoldData    float64 // 1.0 mmol/L
    HbA1cDeltaHoldData      float64 // 2.0% in 30d
    PotassiumCeilingHold    float64 // 8.0 mEq/L

    // Channel C
    ProtocolRulesPath string // path to protocol_rules.yaml

    // Titration
    DoseCooldownBasalHours int     // 48h (A-03)
    DoseCooldownRapidHours int     // 6h (A-03)
    MaxDoseDeltaPercent    float64 // 20% (PG-05)

    // Cache refresh
    SafetyCacheRefreshMinutes int // 60 min default

    // SafetyTrace
    TraceRetentionYears int // 10 (DISHA compliance)

    MetricsEnabled bool
}
```

#### 0.4 Database migration — 001_initial_schema.sql

```sql
-- V-MCU initial schema: SafetyTrace + dose history

CREATE TABLE safety_traces (
    trace_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id          UUID NOT NULL,
    cycle_timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Channel A (Diagnostic)
    mcu_gate            VARCHAR(10) NOT NULL,
    mcu_gate_card_id    UUID,
    mcu_gate_rationale  TEXT,

    -- Channel B (Physiological)
    physio_gate         VARCHAR(10) NOT NULL,
    physio_rule_fired   VARCHAR(100),
    physio_raw_values   JSONB,

    -- Channel C (Protocol)
    protocol_gate       VARCHAR(10) NOT NULL,
    protocol_rule_id    VARCHAR(20),
    protocol_rule_vsn   VARCHAR(64),
    protocol_guide_ref  VARCHAR(100),

    -- Arbiter output
    final_gate          VARCHAR(10) NOT NULL,
    dominant_channel    VARCHAR(5),
    arbiter_rationale   TEXT,

    -- Titration output
    dose_applied        DECIMAL(10,4),
    dose_delta          DECIMAL(10,4),
    blocked_by          VARCHAR(50),

    -- Enrichment
    observation_reliability VARCHAR(10),
    gain_factor            DECIMAL(5,3) DEFAULT 1.0
);

CREATE INDEX idx_safety_traces_patient ON safety_traces(patient_id);
CREATE INDEX idx_safety_traces_timestamp ON safety_traces(cycle_timestamp);
CREATE INDEX idx_safety_traces_final_gate ON safety_traces(final_gate);

CREATE TABLE dose_history (
    dose_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL,
    medication_class VARCHAR(50) NOT NULL,
    dose_before     DECIMAL(10,2),
    dose_after      DECIMAL(10,2),
    dose_delta_pct  DECIMAL(5,2),
    trace_id        UUID REFERENCES safety_traces(trace_id),
    applied_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dose_history_patient ON dose_history(patient_id, applied_at DESC);
```

---

## Phase 1: Gate Signal Models & Channel A Client

**Duration**: 1 session | **Dependencies**: Phase 0

### Tasks

#### 1.1 Gate signal enum — 5-state hierarchy

```go
// internal/models/gate_signal.go
package models

type GateSignal string

const (
    GateClear    GateSignal = "CLEAR"     // no objection
    GateModify   GateSignal = "MODIFY"    // change permitted with constraint
    GatePause    GateSignal = "PAUSE"     // hold current dose
    GateHalt     GateSignal = "HALT"      // full stop; clinical escalation
    GateHoldData GateSignal = "HOLD_DATA" // data anomaly; defer action
)

// Level returns severity for comparison. HALT > HOLD_DATA > PAUSE > MODIFY > CLEAR.
func (g GateSignal) Level() int {
    switch g {
    case GateHalt:     return 4
    case GateHoldData: return 3
    case GatePause:    return 2
    case GateModify:   return 1
    case GateClear:    return 0
    default:           return 0
    }
}
```

**Design note**: HOLD_DATA sits between PAUSE and HALT. It defers one cycle (like PAUSE) but triggers a KB-20 re-validation request (unlike PAUSE). HALT is reserved for confirmed clinical danger.

#### 1.2 Arbiter I/O models

```go
// internal/models/arbiter.go
type ArbiterInput struct {
    MCUGate      GateSignal `json:"mcu_gate"`
    PhysioGate   GateSignal `json:"physio_gate"`
    ProtocolGate GateSignal `json:"protocol_gate"`
}

type ArbiterOutput struct {
    FinalGate       GateSignal   `json:"final_gate"`
    DominantChannel string       `json:"dominant_channel"` // "A" | "B" | "C" | "NONE"
    AllChannels     ArbiterInput `json:"all_channels"`
    RationaleCode   string       `json:"rationale_code"`
}
```

#### 1.3 Channel A client — reads KB-23 enriched MCU_GATE

```go
// internal/channel_a/kb23_client.go
// Reads from KB-23 Redis-backed endpoint: GET /patients/:id/mcu-gate
// Returns: MCUGate (SAFE→CLEAR mapping), CardID, dose_adjustment_notes
// Timeout: 100ms (Redis-backed, target <5ms)
// Fallback on timeout: GatePause (safety-first degradation)
```

**Mapping**: KB-23 uses SAFE/MODIFY/PAUSE/HALT. V-MCU maps SAFE→CLEAR for the 5-state hierarchy.

#### 1.4 SafetyTrace model + GORM auto-migration

```go
// internal/models/safety_trace.go
// Maps directly to safety_traces table.
// Append-only — no UPDATE or DELETE operations.
```

---

## Phase 2: Channel B — PhysiologySafetyMonitor (SA-02)

**Duration**: 2 sessions | **Dependencies**: Phase 1
**Critical constraint**: Channel B must NOT import any Channel A or titration package.

### Tasks

#### 2.1 Raw data input struct — no MetabolicState

```go
// internal/channel_b/raw_inputs.go
package channel_b

// RawPatientData contains only unprocessed values from KB-20.
// CONSTRAINT: This struct must never contain MetabolicState,
// HyperglycaemiaMechanism, ISF, or any KB-22/KB-23 derived field.
type RawPatientData struct {
    // Current lab values (raw from KB-20)
    GlucoseCurrent    float64   // mmol/L
    GlucoseTimestamp  time.Time
    CreatinineCurrent float64   // µmol/L
    PotassiumCurrent  float64   // mEq/L
    SBPCurrent        float64   // mmHg
    WeightKgCurrent   float64
    EGFRCurrent       float64   // mL/min/1.73m²
    HbA1cCurrent      float64   // %

    // Historical values (for delta computation)
    Creatinine48hAgo  *float64
    EGFRPrior48h      *float64
    HbA1cPrior30d     *float64
    Weight72hAgo      *float64

    // Glucose trend (last 3 readings)
    GlucoseReadings   []TimestampedValue

    // Dose context (from V-MCU internal history, not KB-23)
    RecentDoseIncrease bool
}

type TimestampedValue struct {
    Value     float64
    Timestamp time.Time
}
```

#### 2.2 PhysiologySafetyMonitor — threshold-based rules

```go
// internal/channel_b/monitor.go

// Evaluate runs all Channel B rules against raw inputs.
// Returns PHYSIO_GATE signal + which rule fired + anomaly flags.
// Rule evaluation order:
//   1. Data anomaly checks (SA-05) → HOLD_DATA
//   2. Critical thresholds → HALT
//   3. Warning thresholds → PAUSE
//   4. Default → CLEAR
func (m *PhysiologySafetyMonitor) Evaluate(data *RawPatientData) PhysioResult
```

Complete rule set from the architecture document:

| # | Condition | Gate | Rationale |
|---|-----------|------|-----------|
| B-01 | glucose < 3.9 mmol/L | HALT | Active hypoglycaemia |
| B-02 | glucose < 4.5 mmol/L | PAUSE | Near-hypoglycaemia |
| B-03 | creatinine 48h delta > 26 µmol/L | HALT | KDIGO AKI Stage 1 |
| B-04 | potassium < 3.0 OR > 6.0 mEq/L | HALT | Cardiac arrhythmia risk |
| B-05 | SBP < 90 mmHg | HALT | Haemodynamic instability |
| B-06 | weight 72h delta > 2.5 kg | PAUSE | Fluid overload signal |
| B-07 | 3 consecutive declining glucose + current < 5.5 + recent dose increase | PAUSE | Glucose trajectory concern |
| DA-01 | eGFR change > 40% in 48h | HOLD_DATA | Likely lab error or AKI |
| DA-02 | creatinine change > 100% in 48h (no clinical event) | HOLD_DATA | Suspected lab error |
| DA-03 | glucose < 1.0 mmol/L | HOLD_DATA | Instrument calibration error |
| DA-04 | HbA1c change > 2.0% in 30d | HOLD_DATA | Biologically impossible |
| DA-05 | potassium > 8.0 mEq/L | HOLD_DATA then confirm | Extreme value — confirm first |

#### 2.3 KB-20 raw lab fetcher

```go
// internal/channel_b/kb20_fetcher.go
// Fetches raw lab values from KB-20: GET /patient/:id/labs?types=...
// Data is cached locally in V-MCU's safety cache (refreshed hourly + on events).
// During rule evaluation, NO network calls — reads from local cache only (SA-06).
```

Uses KB-20's existing endpoints:
- `GET /patient/:id/labs` — returns `[]LabEntry` with `LabType`, `Value`, `MeasuredAt`
- Lab types already defined in KB-20: `CREATININE`, `EGFR`, `FBG`, `HBA1C`, `SBP`, `POTASSIUM`

#### 2.4 Build-time import constraint enforcement

```go
// tests/import_constraint_test.go
// Verifies that channel_b package does NOT import:
//   - v-mcu/internal/titration
//   - v-mcu/internal/channel_a
//   - Any kb-22 or kb-23 package
// Uses go/parser to scan import declarations.
// This test MUST run in CI — failure blocks deployment.
```

#### 2.5 Unit tests — all 12 rules

Each rule gets a dedicated test case with boundary values:
- B-01: glucose at 3.89 (HALT), 3.90 (HALT boundary), 3.91 (not B-01)
- B-04: potassium at 2.99 (HALT), 3.01 (CLEAR), 5.99 (CLEAR), 6.01 (HALT)
- DA-01: eGFR from 52 → 28 in 48h (46% drop → HOLD_DATA)
- DA-04: HbA1c from 7.0 → 9.1 in 30d (2.1% → HOLD_DATA)

---

## Phase 3: Channel C — ProtocolGuard (SA-03)

**Duration**: 1.5 sessions | **Dependencies**: Phase 1
**Critical constraint**: Rules pre-compiled at startup. Zero network calls during evaluation.

### Tasks

#### 3.1 protocol_rules.yaml — static clinical rules

```yaml
# protocol_rules.yaml — loaded at V-MCU startup, versioned with deployment
version: "1.0.0"

rules:
  - rule_id: PG-01
    description: "Metformin absolute contraindication at eGFR < 30"
    guideline_ref: "KDIGO_CKD_2024_S4.3"
    condition:
      field: egfr
      operator: lt
      value: 30
      medication_active: METFORMIN
    gate: HALT

  - rule_id: PG-02
    description: "SGLT2i efficacy threshold at eGFR < 45"
    guideline_ref: "SGLT2I_EFFICACY_THRESHOLD"
    condition:
      field: egfr
      operator: lt
      value: 45
      medication_active: SGLT2I
    gate: PAUSE

  - rule_id: PG-03
    description: "AKI detected — hold all nephrotoxic agents"
    guideline_ref: "KDIGO_AKI_2012"
    condition:
      field: aki_detected
      operator: eq
      value: true
    gate: HALT

  - rule_id: PG-04
    description: "Never increase insulin into active hypoglycaemia"
    guideline_ref: "ADA_HYPO_MANAGEMENT_2024"
    condition:
      field: active_hypoglycaemia
      operator: eq
      value: true
      action_type: insulin_increase
    gate: HALT

  - rule_id: PG-05
    description: "Maximum 20% dose change per cycle"
    guideline_ref: "ALGORITHMIC_DRIFT_PROTECTION"
    condition:
      field: dose_delta_percent
      operator: gt
      value: 20
    gate: PAUSE

  - rule_id: PG-07
    description: "Post-hypoglycaemia dose increase prohibition (7d window)"
    guideline_ref: "POST_HYPO_SAFETY_WINDOW"
    condition:
      field: hypoglycaemia_within_7d
      operator: eq
      value: true
      action_type: dose_increase
    gate: HALT
```

**Note**: PG-06 (therapeutic futility) is excluded from v1 pending clinical team decision on the HbA1c improvement metric. The document offers three options; none has been selected yet.

#### 3.2 ProtocolGuard — compiled rule evaluator

```go
// internal/channel_c/guard.go
type ProtocolGuard struct {
    rules       []CompiledRule
    rulesHash   string          // SHA-256 of protocol_rules.yaml for SafetyTrace
    log         *zap.Logger
}

// LoadRules parses protocol_rules.yaml at startup.
// Called once during V-MCU initialization — not at runtime.
func LoadRules(path string) (*ProtocolGuard, error)

// Evaluate checks all rules against the current patient + proposed action.
// Returns PROTOCOL_GATE signal + which rule fired.
// MUST NOT make any network calls — all data comes from PatientContext.
func (g *ProtocolGuard) Evaluate(ctx *TitrationContext) ProtocolResult
```

#### 3.3 TitrationContext — input to Channel C

```go
// internal/channel_c/context.go
type TitrationContext struct {
    // From local safety cache (KB-20 raw data, refreshed hourly)
    EGFR                  float64
    ActiveMedications     []string // drug class list
    AKIDetected           bool     // derived from Channel B creatinine delta
    ActiveHypoglycaemia   bool     // derived from Channel B glucose rules
    HypoglycaemiaWithin7d bool     // from dose_history + KB-23 cards

    // From proposed titration action
    ProposedAction        string   // "dose_increase" | "dose_decrease" | "dose_hold"
    DoseDeltaPercent      float64  // absolute % change proposed
}
```

#### 3.4 Rule version hash in SafetyTrace

Every SafetyTrace record includes `protocol_rule_vsn` = SHA-256 of the `protocol_rules.yaml` file loaded at startup. This enables post-hoc audit: given any SafetyTrace, you can verify exactly which rule set was in effect when the decision was made.

#### 3.5 Unit tests — all 6 rules + edge cases

- PG-01: eGFR=29 + Metformin active → HALT; eGFR=31 + Metformin → CLEAR
- PG-02: eGFR=44 + SGLT2I active → PAUSE; eGFR=46 → CLEAR
- PG-05: dose delta 21% → PAUSE; 19% → CLEAR
- PG-04 + PG-07 interaction: both HALT, but different clinical contexts

---

## Phase 4: Safety Arbiter (SA-01)

**Duration**: 0.5 session | **Dependencies**: Phases 1-3
**Critical constraint**: Pure function. No I/O. No external dependencies. < 1ms.

### Tasks

#### 4.1 Arbitrate function — 1oo3 veto with severity hierarchy

```go
// internal/arbiter/arbiter.go
package arbiter

import "v-mcu/internal/models"

// Arbitrate applies the 1oo3 veto rule: the most restrictive gate wins.
// Severity: HALT > HOLD_DATA > PAUSE > MODIFY > CLEAR
// This function has NO side effects, NO I/O, NO external calls.
func Arbitrate(input models.ArbiterInput) models.ArbiterOutput {
    signals := []models.GateSignal{input.MCUGate, input.PhysioGate, input.ProtocolGate}
    final := mostRestrictive(signals)
    dominant := dominantChannel(input, final)
    rationale := buildRationale(input, final, dominant)

    return models.ArbiterOutput{
        FinalGate:       final,
        DominantChannel: dominant,
        AllChannels:     input,
        RationaleCode:   rationale,
    }
}

// mostRestrictive picks the signal with the highest severity level.
func mostRestrictive(signals []models.GateSignal) models.GateSignal {
    max := models.GateClear
    for _, s := range signals {
        if s.Level() > max.Level() {
            max = s
        }
    }
    return max
}

// dominantChannel identifies which channel drove the final decision.
// When multiple channels agree on the same level, reports the first match
// in priority order: B (physiology) > C (protocol) > A (diagnostic).
// Safety channels take attribution priority over diagnostic.
func dominantChannel(input models.ArbiterInput, final models.GateSignal) string {
    if input.PhysioGate == final   { return "B" }
    if input.ProtocolGate == final { return "C" }
    if input.MCUGate == final      { return "A" }
    return "NONE"
}
```

**Design rationale**: When all three channels return CLEAR, `dominantChannel` returns "NONE" — no channel drove a restrictive decision. When two or more channels agree on the restrictive signal, attribution goes to Channel B first (physiological safety is the most fundamental), then C, then A. This isn't functionally important (the gate is the same either way) but matters for SafetyTrace audit readability.

#### 4.2 No-bypass structural guarantee

The titration function signature enforces that Arbitrate() must be called:

```go
// internal/titration/engine.go

// ComputeDose is the ONLY function that produces a dose output.
// It REQUIRES an ArbiterOutput — there is no code path to call it
// without first calling arbiter.Arbitrate().
func (e *TitrationEngine) ComputeDose(
    arbiterResult models.ArbiterOutput,
    metabolicState *MetabolicState,
    currentDose float64,
) (*DoseResult, error) {
    // If arbiter blocked, return nil dose
    if arbiterResult.FinalGate.Level() >= models.GatePause.Level() {
        return &DoseResult{Blocked: true, BlockedBy: arbiterResult.DominantChannel}, nil
    }
    // ... titration logic only reachable when gate is CLEAR or MODIFY
}
```

This is a code-structure guarantee, not a runtime check. The function cannot be called without an `ArbiterOutput` value because Go's type system requires it.

#### 4.3 Unit tests — all gate combinations

Test matrix: 5 states x 3 channels = 125 combinations. Key cases:
- All CLEAR → FinalGate=CLEAR, Dominant=NONE
- A=CLEAR, B=HALT, C=CLEAR → FinalGate=HALT, Dominant=B
- A=MODIFY, B=CLEAR, C=PAUSE → FinalGate=PAUSE, Dominant=C
- A=HALT, B=HALT, C=HALT → FinalGate=HALT, Dominant=B (physiology takes attribution)
- A=HOLD_DATA, B=PAUSE, C=CLEAR → FinalGate=HOLD_DATA, Dominant=A
- B=HOLD_DATA, C=HALT → FinalGate=HALT, Dominant=C (HALT > HOLD_DATA)

---

## Phase 5: Local Safety Cache & Event Integration

**Duration**: 1.5 sessions | **Dependencies**: Phases 2-3

### Tasks

#### 5.1 Safety data cache — read-through with scheduled refresh

```go
// internal/cache/safety_cache.go
// Local in-memory cache of KB-20 labs, KB-23 perturbations, and MCU_GATE.
// Refreshed every 60 minutes AND on KB-19 events.
//
// During titration cycle: Channel B and C read from this cache ONLY.
// Network calls happen during refresh, never during evaluation.
//
// Cache entries:
//   - Raw labs per patient (from KB-20)
//   - Active perturbation windows (from KB-23)
//   - Current MCU_GATE per patient (from KB-23 Redis)
//   - Dose history per patient (from local DB)
```

#### 5.2 KB-19 event subscription

V-MCU subscribes to KB-19 events for cache invalidation:
- `MCU_GATE_CHANGED` → refresh Channel A cache for that patient
- `LAB_UPDATED` → refresh Channel B cache for that patient
- `PERTURBATION_CREATED` → refresh perturbation window
- `DATA_ANOMALY_RESOLVED` → clear HOLD_DATA flag

Per spec: event subscription, not polling. < 30s propagation.

#### 5.3 HOLD_DATA response flow (SA-05)

When Channel B returns HOLD_DATA:
1. V-MCU logs anomalous value to SafetyTrace
2. V-MCU sends KB-20 re-validation: `POST /patient/:id/labs/:lab_id/flag-anomaly`
3. V-MCU notifies KB-19: `DATA_ANOMALY_DETECTED` event
4. This titration cycle is deferred (no dose change)
5. Next cycle: if re-validated as correct → proceed; if confirmed error → KB-20 marks as ANOMALOUS

**New KB-20 endpoint required**: `POST /patient/:id/labs/:lab_id/flag-anomaly`
- Sets `ValidationStatus = 'FLAGGED'` with `FlagReason = 'ANOMALY_FLAGGED_BY_VMCU'`
- Triggers clinical review notification via KB-19
- Excludes entry from subsequent queries until clinician confirms/rejects

---

## Phase 6: SafetyTrace Audit System (SA-04)

**Duration**: 1 session | **Dependencies**: Phases 4-5

### Tasks

#### 6.1 SafetyTrace writer — async, non-blocking

```go
// internal/trace/writer.go
// Writes one SafetyTrace record per titration cycle.
// Async PostgreSQL insert — does NOT block the titration cycle.
// Target: < 5ms non-blocking write via buffered channel + goroutine.
//
// IMMUTABILITY: No UPDATE or DELETE operations on safety_traces.
// Append-only audit log for DISHA compliance.
```

#### 6.2 SafetyTrace fields populated from all phases

| Field | Source |
|-------|--------|
| mcu_gate, mcu_gate_card_id, mcu_gate_rationale | Channel A (KB-23 cache) |
| physio_gate, physio_rule_fired, physio_raw_values | Channel B (PhysiologySafetyMonitor) |
| protocol_gate, protocol_rule_id, protocol_rule_vsn, protocol_guide_ref | Channel C (ProtocolGuard) |
| final_gate, dominant_channel, arbiter_rationale | Safety Arbiter output |
| dose_applied, dose_delta, blocked_by | Titration engine output (null when blocked) |
| observation_reliability, gain_factor | KB-23 enriched gate response |

#### 6.3 Data retention policy

```go
// internal/trace/retention.go
// SafetyTrace records retained for 10 years (DISHA compliance).
// Records beyond retention window: archived to cold storage, not deleted.
// Configurable via TRACE_RETENTION_YEARS env var (default: 10).
```

#### 6.4 Trace query endpoints (for clinical audit)

```
GET /patients/:id/safety-traces              → paginated trace history
GET /patients/:id/safety-traces?gate=HALT    → filtered by gate outcome
GET /safety-traces/:trace_id                 → single trace detail
```

---

## Phase 7: Integration & Latency Validation (SA-06)

**Duration**: 1 session | **Dependencies**: All previous phases

### Tasks

#### 7.1 Full titration cycle integration test

End-to-end test that:
1. Seeds KB-20 with raw lab data for a test patient
2. Seeds KB-23 with an MCU_GATE=MODIFY for that patient
3. Populates the local safety cache
4. Triggers a titration cycle
5. Asserts: Channel A returns MODIFY, Channel B returns CLEAR, Channel C returns CLEAR
6. Asserts: Arbiter returns MODIFY, Dominant=A
7. Asserts: SafetyTrace record written with all fields populated
8. Asserts: Dose output reflects MODIFY constraints

#### 7.2 Latency budget validation

```go
// tests/latency_test.go
// Measures wall-clock time for each phase:
//   Channel A read:    assert < 5ms
//   Channel B eval:    assert < 10ms
//   Channel C eval:    assert < 2ms
//   Arbiter:           assert < 1ms
//   Total synchronous: assert < 18ms
//   SafetyTrace write: assert < 5ms (non-blocking, measured separately)
```

#### 7.3 Safety scenario tests

| Scenario | Expected Outcome |
|----------|-----------------|
| Normal patient, all clear | FinalGate=CLEAR, dose applied |
| Glucose 3.5 mmol/L | Channel B HALT, dose blocked |
| eGFR 25 + Metformin | Channel C HALT (PG-01), dose blocked |
| KB-23 returns PAUSE (acute illness) | Channel A PAUSE, dose blocked |
| eGFR dropped 45% in 48h | Channel B HOLD_DATA, KB-20 flagged, dose deferred |
| Potassium 2.8 + proposed insulin increase | Channel B HALT (B-04) + Channel C HALT (PG-04), Dominant=B |
| All three channels HALT simultaneously | FinalGate=HALT, Dominant=B, all channels logged in SafetyTrace |

#### 7.4 Prometheus metrics

```
vmcu_titration_cycles_total{final_gate}        — counter
vmcu_channel_a_latency_ms                      — histogram
vmcu_channel_b_latency_ms                      — histogram
vmcu_channel_c_latency_ms                      — histogram
vmcu_arbiter_latency_ms                        — histogram
vmcu_safety_trace_write_latency_ms             — histogram
vmcu_gate_blocked_total{channel,gate}          — counter
vmcu_hold_data_triggered_total{rule}           — counter
vmcu_cache_refresh_total{source}               — counter
vmcu_cache_age_seconds{source}                 — gauge
```

---

## Phase 8: V-MCU Design Commitments 1-8 (Titration Engine)

**Duration**: 3 sessions | **Dependencies**: Phases 0-7
**Note**: These are the 8 pre-existing commitments from the Final Review and Supplementary Addendum.

### Tasks (summary — each is its own sub-phase)

| # | Commitment | Implementation |
|---|-----------|---------------|
| 1 | Integrator freeze()/resume() | `internal/titration/integrator.go` — freeze on PAUSE/HALT, resume from frozen value |
| 2 | Rate limiter post-resume | 50% max dose delta for ceil(pause_hours/24) cycles after resume |
| 3 | 3-phase re-entry protocol | `internal/titration/reentry.go` — monitoring → conservative → normal |
| 4 | MCU_GATE subscription | KB-19 event: `MCU_GATE_CHANGED` → cache invalidation (done in Phase 5) |
| 5 | No autonomous gate override | Code structure: V-MCU cannot call dose computation without Arbiter (done in Phase 4) |
| 6 | Dose cooldown | 48h basal / 6h rapid-acting minimum between changes |
| 7 | Control gain modulation | `dose_delta *= gain_factor` from KB-23 adherence enrichment |
| 8 | MetabolicPhysiologyEngine (KB-24) | `internal/titration/metabolic_engine.go` — MetabolicState, ISF, mechanism, dawn phenomenon |

**Important**: Commitment 8 (MetabolicPhysiologyEngine / KB-24) is the optimisation module. It must be in a **separate Go package** from Channel B's PhysiologySafetyMonitor. The build-time import constraint (Phase 2.4) ensures Channel B never reads MetabolicState.

---

## Dependency Graph

```
Phase 0: Scaffold ─────────────────────────────────────────┐
    │                                                       │
Phase 1: Models + Channel A Client                         │
    │         │         │                                   │
Phase 2:   Phase 3:   Phase 4:                             │
Channel B  Channel C  Arbiter                              │
    │         │         │                                   │
    └────┬────┘         │                                   │
         │              │                                   │
Phase 5: Cache + Events │                                   │
         │              │                                   │
         └──────┬───────┘                                   │
                │                                           │
         Phase 6: SafetyTrace                               │
                │                                           │
         Phase 7: Integration + Latency                     │
                │                                           │
         Phase 8: Titration Engine (Commitments 1-8)        │
                                                            │
    All phases share infrastructure from ────────────────────┘
```

Phases 2, 3, and 4 can be developed **in parallel** — they have no dependencies on each other.

---

## Cross-KB Integration Points

### V-MCU reads from:

| Service | Endpoint | Purpose | Frequency |
|---------|----------|---------|-----------|
| KB-20 | `GET /patient/:id/labs` | Raw lab values for Channel B | Cache refresh (hourly + event) |
| KB-20 | `GET /patient/:id/medications` | Active medications for Channel C | Cache refresh (hourly + event) |
| KB-23 | `GET /patients/:id/mcu-gate` | Channel A gate signal | Cache refresh (hourly + event) |
| KB-23 | `GET /perturbations/active` | Active perturbation windows | Cache refresh (hourly + event) |

### V-MCU writes to:

| Service | Endpoint | Purpose | Frequency |
|---------|----------|---------|-----------|
| KB-20 | `POST /patient/:id/labs/:lab_id/flag-anomaly` | HOLD_DATA anomaly flagging (SA-05) | On HOLD_DATA only |
| KB-19 | Event: `TITRATION_COMPLETED` | Titration outcome for orchestration | Every cycle |
| KB-19 | Event: `DATA_ANOMALY_DETECTED` | HOLD_DATA notification | On HOLD_DATA only |
| KB-23 | `POST /perturbations` | Treatment perturbation from dose changes | On dose change |

### New endpoints required in existing services:

| Service | New Endpoint | Spec Reference |
|---------|-------------|----------------|
| KB-20 | `POST /patient/:id/labs/:lab_id/flag-anomaly` | SA-05 |

---

## Open Items (Require Decisions Before Implementation)

| # | Item | Owner | Blocking Phase |
|---|------|-------|---------------|
| 1 | **PG-06 therapeutic futility rule** — what metric for "measurable HbA1c improvement"? Options: (a) 12 cycles / 3 months, (b) 14-day CGM/FBG trend, (c) defer to v2 | Clinical team | Phase 3 (non-blocking — PG-06 excluded from v1) |
| 2 | **KB-24 MetabolicPhysiologyEngine spec** — full pre-implementation spec for ISF, mechanism classification, dawn phenomenon | Architecture team | Phase 8 |
| 3 | **V-MCU port assignment** — confirm 8135 or other | DevOps | Phase 0 |
| 4 | **KB-19 event schema** — confirm event type names for TITRATION_COMPLETED, DATA_ANOMALY_DETECTED | KB-19 team | Phase 5 |

---

## Estimated Implementation Effort

| Phase | Sessions | Parallelizable |
|-------|----------|---------------|
| 0: Scaffold | 1 | — |
| 1: Models + Channel A | 1 | — |
| 2: Channel B | 2 | Yes (with 3, 4) |
| 3: Channel C | 1.5 | Yes (with 2, 4) |
| 4: Arbiter | 0.5 | Yes (with 2, 3) |
| 5: Cache + Events | 1.5 | — |
| 6: SafetyTrace | 1 | — |
| 7: Integration | 1 | — |
| 8: Titration Engine | 3 | — |
| **Total** | **~12.5 sessions** | Phases 2-4 parallel saves ~2 sessions |

Critical path: 0 → 1 → {2,3,4} → 5 → 6 → 7 → 8 = ~10.5 sessions sequential.

---

## Verification Checklist (Pre-Staging)

- [ ] Channel B package has zero imports from Channel A, titration, or KB-22/KB-23 packages
- [ ] `go mod graph` confirms no transitive dependency between channel_b and titration
- [ ] All 12 Channel B rules have boundary-value unit tests
- [ ] All 6 Channel C rules have boundary-value unit tests
- [ ] Arbiter has exhaustive gate combination tests (125 cases)
- [ ] No execution path exists to ComputeDose() without Arbitrate() (code review)
- [ ] SafetyTrace is append-only — no UPDATE/DELETE in any migration or code
- [ ] SafetyTrace includes protocol_rule_vsn hash on every record
- [ ] Latency budget met: < 18ms total synchronous, < 5ms trace write
- [ ] HOLD_DATA triggers KB-20 flag-anomaly endpoint
- [ ] KB-19 event subscription confirmed working (< 30s propagation)
- [ ] `protocol_rules.yaml` version hash matches deployment artifact
- [ ] PG-06 placeholder documented but NOT in v1 rule set
- [ ] All Prometheus metrics emitting correctly
- [ ] 10-year retention policy configured in SafetyTrace writer
