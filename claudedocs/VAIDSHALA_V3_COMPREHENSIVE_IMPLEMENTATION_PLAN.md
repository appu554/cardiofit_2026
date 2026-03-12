# Vaidshala V3 Comprehensive Implementation Plan

**Date**: 2026-03-06
**Scope**: All gaps identified across three cross-check rounds (11 agents) against the full CardioFit codebase
**Documents Reviewed**:
1. Vaidshala_V3_Integration_Architecture_Updated.docx (March 2026 revision)
2. Cross-check synthesis analysis
3. V3 Metabolic Correction Loop Final Architecture Proposal (Parts 1-7)

---

## Priority Classification

| Priority | Definition | SLA |
|----------|-----------|-----|
| **P0** | Runtime failures, silent data loss, safety-critical bugs | Fix before any integration testing |
| **P1** | Missing integration contracts, broken cross-KB wiring | Fix before V-MCU clinical pilot |
| **P2** | New specification requirements from Final Architecture Proposal | Implement before clinical validation |
| **P3** | Engineering quality, observability, future scalability | Implement before production deployment |

---

## PHASE 0: Port Registry & Collision Resolution

**Rationale**: Four independent port allocation schemes exist across the project (CLAUDE.md KB services, Apollo Federation, docker-compose files, medication-service-v2). Two runtime collisions cause silent failures.

### Task 0.1: Create Canonical Port Registry

**File to create**: `backend/shared-infrastructure/PORT_REGISTRY.md`

Establish a single source of truth for all service ports. The registry must be referenced by every docker-compose, config.go, and CLAUDE.md.

**Canonical port assignments** (resolving all conflicts):

| Service | Port | Notes |
|---------|------|-------|
| KB-1 Drug Rules | 8081 | Unchanged |
| KB-2 Clinical Context | 8086 | Unchanged |
| KB-3 Guidelines | 8087 | Unchanged |
| KB-4 Patient Safety | 8088 | Unchanged |
| KB-5 Drug Interactions | 8089 | Unchanged |
| KB-6 Formulary | 8091 | Unchanged |
| KB-7 Terminology | 8092 | Unchanged |
| KB-8 Calculator | 8093 | Unchanged |
| KB-9 Care Gaps | 8094 | Unchanged |
| KB-10 Rules Engine | 8095 | Unchanged |
| KB-11 Population Health | 8096 | Unchanged |
| KB-12 Ordersets/CarePlans | 8097 | Unchanged |
| KB-13 Quality Measures | 8098 | Unchanged |
| KB-14 Care Navigator | 8099 | Unchanged |
| KB-16 Lab Interpretation | 8100 | Unchanged |
| KB-17 Population Registry | 8101 | Unchanged |
| KB-18 Governance Engine | 8102 | Unchanged |
| KB-19 Protocol Orchestrator | 8103 | **CHANGED from 8099/8097/8129** |
| KB-20 Patient Profile | 8131 | Unchanged |
| KB-21 Behavioral Intelligence | 8133 | **CHANGED from 8093** (was colliding with KB-8) |
| KB-22 HPI Engine | 8132 | Unchanged |
| KB-23 Decision Cards | 8134 | Unchanged |
| V-MCU Engine | 8140 | New assignment |

### Task 0.2: Fix KB-21 Port Collision (P0)

**Problem**: KB-21 docker-compose.yml binds to port 8093, which is KB-8 Calculator's port. Both services fail silently when co-deployed.

**Files to change**:
- [`kb-21-behavioral-intelligence/docker-compose.yml`](backend/shared-infrastructure/knowledge-base-services/kb-21-behavioral-intelligence/docker-compose.yml) — Lines 10, 12: Change `8093` → `8133`
- `kb-21-behavioral-intelligence/internal/config/config.go` — Default port: Change `8093` → `8133`
- `kb-21-behavioral-intelligence/cmd/server/main.go` — If port is hardcoded

**Downstream references to update**:
- [`kb-22-hpi-engine/internal/config/config.go:67`](backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/config/config.go#L67) — KB21URL already defaults to `8093`, must change to `8133`
- KB-23 config if it references KB-21
- KB-21 docker-compose.yml line 19: `KB20_PATIENT_PROFILE_URL` uses `8095` (should be `8131`)
- KB-21 docker-compose.yml line 21: `KB19_ORCHESTRATOR_URL` uses `8097` (should be `8103`)

### Task 0.3: Fix KB-5 Port in KB-22 Config (P0)

**Problem**: [`kb-22-hpi-engine/internal/config/config.go:69`](backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/config/config.go#L69) defaults KB5_URL to port `8089`. Port 8089 IS actually KB-5's port per the KB CLAUDE.md — so this is **correct** in the KB port scheme. However, verify that KB-5's actual `main.go` binds to 8089.

**Action**: Verify KB-5 binding port matches 8089. If it does, this item is resolved. If not, align all references.

### Task 0.4: Fix KB-19 Port in KB-22 Config (P0)

**Problem**: [`kb-22-hpi-engine/internal/config/config.go:70`](backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/config/config.go#L70) defaults KB19_URL to port `8099`. KB-19's actual port needs verification and alignment with the canonical registry (Task 0.1).

**Files to change**:
- `kb-22-hpi-engine/internal/config/config.go:70` — Update KB19URL default to match canonical port
- `kb-23-decision-cards/internal/config/config.go` — Same for KB-23's KB19 reference
- KB-21 docker-compose.yml line 21 — Already flagged in Task 0.2

### Task 0.5: Propagate Port Registry to All docker-compose Files

**Files to audit and update**:
- `kb-21-behavioral-intelligence/docker-compose.yml` (3 wrong cross-KB URLs)
- `kb-22-hpi-engine/docker-compose.yml` (verify all cross-KB URLs)
- `kb-23-decision-cards/docker-compose.yml` (verify all cross-KB URLs)
- Root `docker-compose.yml` files
- Apollo Federation config (if any KB-20+ references exist)

---

## PHASE 1: Cross-KB Wiring Fixes

### ~~Task 1.1: Fix KB-22 outcome_publisher.go /events Endpoint (P0)~~

**STATUS: RESOLVED** — KB-19 [server.go:151](backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/internal/api/server.go#L151) registers `POST /api/v1/events` as an event ingestion endpoint alongside `/api/v1/execute`. The `/events` endpoint accepts HPI_COMPLETE, SAFETY_ALERT, and MCU_GATE_CHANGED events. KB-22's `outcome_publisher.go` and KB-23's `kb19_publisher.go` are calling the correct endpoint.

### ~~Task 1.2: Fix KB-23 kb19_publisher.go /events Endpoint (P0)~~

**STATUS: RESOLVED** — Same as Task 1.1. KB-23's [kb19_publisher.go:90](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/kb19_publisher.go#L90) correctly calls `/api/v1/events`.

### Task 1.3: Add LAB_RESULT Event Type to KB-20 (P1)

**Problem**: KB-20's [`events.go`](backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/events.go) defines only 4 event types (STRATUM_CHANGE, SAFETY_ALERT, MEDICATION_THRESHOLD_CROSSED, MEDICATION_CHANGE). No LAB_RESULT event exists. When [`fhir_sync_worker.go:160`](backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/fhir/fhir_sync_worker.go#L160) writes a lab result via `w.db.Create(lab)`, no event is published. V-MCU and KB-23 cannot react to new lab data in real-time.

**Changes required**:

1. **`kb-20-patient-profile/internal/models/events.go`** — Add:
   ```go
   EventLabResult = "LAB_RESULT"
   ```
   Add `LabResultPayload` struct with fields: `LabType`, `Value`, `Unit`, `Timestamp`, `SourceSystem`.

2. **`kb-20-patient-profile/internal/fhir/fhir_sync_worker.go`** — After line 160 (`w.db.Create(lab)`), publish a LAB_RESULT event via the event bus.

3. **`kb-20-patient-profile/internal/services/event_publisher.go`** (or equivalent) — Add handler for LAB_RESULT events.

### Task 1.4: Wire KB-20 eGFR Trajectory to V-MCU/KB-23 (P1)

**Problem**: KB-20's [`egfr_engine.go:113-160`](backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/egfr_engine.go#L113) implements OLS linear regression for eGFR trajectory (`ClassifyTrajectory()`), but this slope data is not consumed by V-MCU Channel B or KB-23 MCU_GATE computation.

**Changes required**:

1. **KB-20 API**: Expose eGFR trajectory slope via existing patient stratum endpoint or a new `GET /api/v1/patients/:id/egfr-trajectory` endpoint returning:
   ```json
   {
     "slope_ml_min_per_year": -4.2,
     "classification": "RAPID_DECLINE",
     "r_squared": 0.87,
     "data_points": 6,
     "window_months": 12
   }
   ```

2. **V-MCU Channel B**: Add a new rule (B-10 or modify B-08/B-09) that considers eGFR slope alongside absolute eGFR for HALT/PAUSE decisions. A rapid decline (>5 mL/min/year) should trigger PAUSE even if absolute eGFR is above the B-08/B-09 thresholds.

3. **KB-23 MCU_GATE**: Optionally incorporate eGFR trajectory into MCU_GATE computation as an additional risk signal.

### Task 1.5: Add AG-01 TreatmentPerturbation Fetch to KB-22 (P1)

**Problem**: [`session_context_provider.go`](backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/session_context_provider.go) runs only 3 parallel goroutines (KB-20 stratum, KB-21 adherence-weights, KB-21 answer-reliability). The architecture spec requires a 4th goroutine fetching KB-23 `GET /api/v1/perturbations/:patient_id/active` to get TreatmentPerturbation data (observation dampening factors).

**Changes required**:

1. **`kb-22-hpi-engine/internal/services/session_context_provider.go`** — Add goroutine 4:
   ```go
   // Goroutine 4: KB-23 active perturbations (AG-01)
   g.Go(func() error {
       url := fmt.Sprintf("%s/api/v1/perturbations/%s/active", p.config.KB23URL, patientID.String())
       // fetch and set result.TreatmentPerturbations
       return nil
   })
   ```

2. **`kb-22-hpi-engine/internal/models/session_context.go`** — Add `TreatmentPerturbations` field to the session context struct.

3. **`kb-22-hpi-engine/internal/services/bayesian_engine.go`** — Modify `Update()` (around line 94) to accept and apply a `stability_factor` parameter derived from TreatmentPerturbation data. This dampens observation weight when a medication change is within the perturbation window.

### Task 1.6: Add AG-02 Answer Reliability Endpoint to KB-21 (P1)

**Problem**: KB-22's [`session_context_provider.go:212`](backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/session_context_provider.go#L212) calls `GET /api/v1/patient/:id/answer-reliability` on KB-21, but KB-21's `handlers.go`/`server.go` do not register this endpoint. The call silently falls back to reliability=1.0 (no adjustment).

**Changes required**:

1. **`kb-21-behavioral-intelligence/internal/api/server.go`** — Register route:
   ```go
   v1.GET("/patient/:id/answer-reliability", s.handleAnswerReliability)
   ```

2. **`kb-21-behavioral-intelligence/internal/api/handlers.go`** — Implement `handleAnswerReliability()`:
   - Query patient's historical answer consistency from KB-21's behavioral model
   - Return a reliability score (0.0–1.0) based on contradiction frequency, response latency patterns, and recency
   - Default to 1.0 for patients with insufficient behavioral data

3. **Unit tests**: Add test coverage for the new endpoint.

### Task 1.7: Wire Adherence-Tier Gain Mapping (P1)

**Problem**: KB-21 computes adherence tiers (HIGH/MEDIUM/LOW) per G-05, but these tiers are not mapped to gain factors used by KB-22's Bayesian engine. The spec requires:
- HIGH → gain 1.0 (full observation weight)
- MEDIUM → gain 0.7
- LOW → gain 0.4

**Changes required**:

1. **KB-22 session_context_provider.go** — After fetching KB-21 adherence weights, map tier labels to numeric gain factors.
2. **KB-22 bayesian_engine.go** — Apply adherence gain factor as a multiplier on observation likelihood in the `Update()` method.

---

## PHASE 2: Safety-Critical Fixes

### Task 2.1: V-MCU Channel B Data Absence Handling (P0)

**Problem**: Go zero-values for unset float64 fields (0.0) cause false HALTs. When `GlucoseCurrent` is 0.0 (meaning "no data"), rule B-01 fires (0.0 < 3.9 threshold) and incorrectly produces HALT. Similarly, `PotassiumCurrent` at 0.0 triggers B-04 (0.0 < 3.0).

**Root cause**: [`channel_b/monitor.go`](vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor.go) reads `RawPatientData` struct fields that use Go's zero-value (0.0) for missing data instead of distinguishing "absent" from "measured zero".

**Changes required**:

1. **`vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/types.go`** (or equivalent) — Change `RawPatientData` fields from `float64` to `*float64` (pointer types) so nil represents "absent":
   ```go
   type RawPatientData struct {
       GlucoseCurrent    *float64
       CreatinineCurrent *float64
       PotassiumCurrent  *float64
       SBPCurrent        *float64
       WeightKgCurrent   *float64
       EGFRCurrent       *float64
       HbA1cCurrent      *float64
       // Delta fields remain float64 (computed, never absent)
       // ...
       // Staleness timestamps for each measurement
       GlucoseTimestamp    *time.Time
       CreatinineTimestamp *time.Time
       PotassiumTimestamp  *time.Time
       SBPTimestamp        *time.Time
   }
   ```

2. **`channel_b/monitor.go` Evaluate()** — Before each rule check, verify the relevant pointer is non-nil. If nil, skip the rule and log the absence. When ALL safety-critical labs are absent, return `HOLD_DATA` (not CLEAR, not HALT):
   ```go
   // In checkB01():
   if data.GlucoseCurrent == nil {
       return nil // skip — no glucose data available
   }
   if *data.GlucoseCurrent < m.cfg.GlucoseHaltThreshold {
       return &PhysioResult{Gate: HALT, Rule: "B-01", ...}
   }
   ```

3. **Add staleness checks** — If a measurement exists but its timestamp is older than a configurable threshold (e.g., glucose > 4h, creatinine > 48h), treat it as stale and apply HOLD_DATA for that measurement:
   ```go
   func (m *PhysiologySafetyMonitor) isStale(ts *time.Time, maxAge time.Duration) bool {
       if ts == nil { return true }
       return time.Since(*ts) > maxAge
   }
   ```

4. **Update all callers** of `RawPatientData` to use pointer semantics when populating the struct.

5. **Tests**: Add test cases for:
   - All fields nil → HOLD_DATA
   - Only glucose nil, others present → evaluate remaining rules
   - Stale glucose (>4h) → HOLD_DATA for glucose-dependent rules
   - Measured glucose of 0.0 (impossible clinically, but handle gracefully)

### Task 2.2: V-MCU Data Staleness Configuration (P1)

**File to create**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/staleness_config.go`

Define configurable staleness thresholds per lab type:

| Lab | Max Staleness | Rationale |
|-----|--------------|-----------|
| Glucose | 4 hours | Rapid physiological change |
| Potassium | 12 hours | Moderate variability |
| Creatinine | 48 hours | Slow-moving marker |
| eGFR | 7 days | Derived, slow trend |
| HbA1c | 90 days | 3-month average marker |
| SBP | 4 hours | Rapid physiological change |
| Weight | 72 hours | Matches B-06 delta window |

---

## PHASE 3: New Specification Requirements

These requirements come from Part 2 of the Final Architecture Proposal.

### Task 3.1: DEPRESCRIBING_MODE for V-MCU (P2)

**Problem**: V-MCU's titration engine only handles dose escalation. No endpoint exists for clinician-initiated medication reduction (deprescribing). The Final Architecture Proposal requires a `DEPRESCRIBING_MODE` that:
- Allows controlled dose reduction with safety monitoring
- Applies a different set of Channel B thresholds during deprescribing (wider bounds)
- Tracks deprescribing rationale and outcome

**Changes required**:

1. **`vaidshala/clinical-runtime-platform/engines/vmcu/titration/deprescribing.go`** (new file):
   - `DeprescribingPlan` struct with target dose, step-down rate, monitoring schedule
   - `StartDeprescribing()` — initiates deprescribing with clinician rationale
   - `StepDown()` — executes next dose reduction step
   - `AbortDeprescribing()` — cancels and returns to previous stable dose
   - Safety: if Channel B fires HALT during deprescribing, freeze at current reduced dose (don't revert to original)

2. **`channel_b/monitor.go`** — Add `DeprescribingMode bool` to `PhysioConfig` or `RawPatientData`. When true, widen glucose thresholds:
   - B-01 HALT: 3.9 → 3.3 mmol/L (more permissive during controlled reduction)
   - B-02 PAUSE: 4.5 → 3.9 mmol/L

3. **V-MCU API**: Add `POST /api/v1/patients/:id/deprescribe` endpoint.

4. **SafetyTrace**: Record deprescribing context in trace entries.

### Task 3.2: Cross-Session Plausibility Arbitration for KB-20 (P2)

**Problem**: KB-20's [`lab_validator.go`](backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_validator.go) only validates individual lab observations (single-observation range checks). No temporal consistency validation exists across sessions.

**Example failure**: Patient's eGFR goes from 45 → 90 → 30 in three consecutive days — each individually valid, but the trajectory is physiologically implausible.

**Changes required**:

1. **`kb-20-patient-profile/internal/services/plausibility_engine.go`** (new file):
   ```go
   type PlausibilityEngine struct {
       db  *database.Database
       log *zap.Logger
   }

   // CheckPlausibility validates a new lab value against the patient's
   // historical trajectory. Returns a PlausibilityResult with confidence
   // and suggested action (ACCEPT, FLAG_REVIEW, REJECT_RETEST).
   func (e *PlausibilityEngine) CheckPlausibility(
       ctx context.Context,
       patientID uuid.UUID,
       labType string,
       newValue float64,
       timestamp time.Time,
   ) (*PlausibilityResult, error)
   ```

2. **Plausibility rules**:
   - **Rate-of-change limits**: eGFR cannot change >15 mL/min/1.73m² per day, creatinine cannot change >50% in 24h (unless dialysis)
   - **Direction consistency**: If 3+ consecutive values trend in one direction, a sudden reversal >2 standard deviations is flagged
   - **Physiological bounds**: Some combinations are impossible (e.g., eGFR=90 with creatinine=500)

3. **Integration**: Call `PlausibilityEngine.CheckPlausibility()` from `fhir_sync_worker.go` before `w.db.Create(lab)`. If result is `FLAG_REVIEW`, still store but mark with `plausibility_flag` column and publish a `PLAUSIBILITY_FLAG` event.

### Task 3.3: KB-23 SLA Miss Behavior (P2)

**Problem**: KB-23's [`card_lifecycle.go`](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_lifecycle.go) manages card state transitions but has no SLA miss scanning. When a Decision Card remains in ACTIVE state past its SLA deadline without physician acknowledgement, nothing happens.

**Spec requirement**: Cards exceeding SLA should:
1. Escalate to supervisor/team lead
2. Publish `SLA_BREACH` event to KB-19
3. Auto-elevate card priority

**Changes required**:

1. **`kb-23-decision-cards/internal/services/sla_scanner.go`** (new file):
   ```go
   type SLAScanner struct {
       db       *database.Database
       kb19     *KB19Publisher
       log      *zap.Logger
       interval time.Duration // scan frequency (default 1 minute)
   }

   // Start begins the background SLA scanning loop.
   func (s *SLAScanner) Start(ctx context.Context)

   // scanOverdueCards finds ACTIVE cards past their SLA deadline.
   func (s *SLAScanner) scanOverdueCards(ctx context.Context) error
   ```

2. **`kb-23-decision-cards/internal/models/decision_card.go`** — Add fields:
   ```go
   SLADeadline   *time.Time `json:"sla_deadline" gorm:"column:sla_deadline"`
   SLABreached   bool       `json:"sla_breached" gorm:"column:sla_breached"`
   SLABreachedAt *time.Time `json:"sla_breached_at" gorm:"column:sla_breached_at"`
   EscalatedTo   string     `json:"escalated_to,omitempty" gorm:"column:escalated_to"`
   ```

3. **SLA configuration by card severity**:
   - HALT cards: 15-minute SLA
   - PAUSE cards: 1-hour SLA
   - MODIFY cards: 4-hour SLA

4. **Publish SLA_BREACH event** to KB-19 via KB19Publisher when SLA is breached.

### Task 3.4: V-MCU Autonomy Limits (P2)

**Problem**: The spec defines 11 design commitments for V-MCU, including that V-MCU must never autonomously exceed certain dose limits without physician confirmation. These limits need explicit enforcement.

**Changes required**:

1. **`vaidshala/clinical-runtime-platform/engines/vmcu/autonomy/limits.go`** (new file):
   ```go
   type AutonomyLimits struct {
       MaxSingleStepPct    float64 // max dose change per cycle (default 20%)
       MaxCumulativePct    float64 // max cumulative change without confirmation (default 50%)
       MaxAbsoluteDoseMg   map[string]float64 // per-drug-class absolute ceilings
       RequireConfirmation bool    // if true, pause and wait for physician ack
   }

   // CheckLimit validates a proposed dose change against autonomy constraints.
   // Returns (allowed bool, reason string).
   func (l *AutonomyLimits) CheckLimit(
       currentDose, proposedDose float64,
       drugClass string,
       cumulativeChangePct float64,
   ) (bool, string)
   ```

2. **Integration**: Call `CheckLimit()` from `vmcu_engine.go` RunCycle() after dose computation but before applying the change. If limit exceeded, freeze at current dose and publish a `PHYSICIAN_CONFIRMATION_REQUIRED` event.

3. **SafetyTrace**: Record autonomy limit checks in every trace entry.

### Task 3.5: Simulation Harness with 8 Synthetic Scenarios (P2)

**Problem**: The Final Architecture Proposal requires 8 synthetic 90-day patient trajectories exercising edge cases before clinical pilot. The existing `vmcu_integration_test.go` has 13 scenario tests but they are unit-level, not the required 90-day trajectory simulations.

**File to create**: `vaidshala/clinical-runtime-platform/engines/vmcu/simulation/`

**Required scenarios**:

| # | Scenario | Key Edge Cases |
|---|----------|---------------|
| 1 | Stable diabetic, gradual improvement | Normal operation, dose reduction trigger |
| 2 | Acute kidney injury during titration | eGFR crash → HALT → freeze → slow recovery |
| 3 | Hypoglycaemia cluster (3 events in 7 days) | B-01 repeated → extended cooldown |
| 4 | Missing data for 72 hours | Staleness → HOLD_DATA → data returns → re-entry |
| 5 | Cross-session plausibility failure | Implausible lab → FLAG_REVIEW → retest confirms |
| 6 | Concurrent deprescribing | SGLT2i reduction while maintaining metformin |
| 7 | Channel A/B/C disagreement | B=PAUSE, C=CLEAR, A=MODIFY → arbiter selects PAUSE |
| 8 | Autonomy limit breach | Cumulative change >50% → physician confirmation wait |

**Structure**:
```
simulation/
├── harness.go          # Test harness engine (time-stepping, state management)
├── patient_factory.go  # Synthetic patient generator
├── scenario_1_test.go  # Stable improvement
├── scenario_2_test.go  # AKI during titration
├── scenario_3_test.go  # Hypoglycaemia cluster
├── scenario_4_test.go  # Missing data
├── scenario_5_test.go  # Plausibility failure
├── scenario_6_test.go  # Deprescribing
├── scenario_7_test.go  # Channel disagreement
├── scenario_8_test.go  # Autonomy limit
└── assertions.go       # Custom assertion helpers for safety trace validation
```

Each scenario must:
- Run as a Go test (`go test -v -run TestScenario1 ./simulation/...`)
- Simulate 90 days of 6-hour V-MCU cycles (360 cycles)
- Validate SafetyTrace completeness at every cycle
- Assert correct Channel B/C/A gate transitions
- Verify dose output stays within autonomy limits

---

## PHASE 4: Kafka & Event Infrastructure

### Task 4.1: Provision Missing Kafka Topics (P3)

**Problem**: The existing `topics-config.yaml` only defines device pipeline topics. No topics exist for:
- `mcu-gate-changed.v1` (KB-23 → KB-19)
- `behavioral-gap.v1` (KB-21 → KB-23)
- `outcome-correlation-changed.v1` (KB-20 → KB-21)
- `lab-result.v1` (KB-20 → V-MCU/KB-23) — distinct from device pipeline `lab-result-events.v1`
- `sla-breach.v1` (KB-23 → notification system)
- `plausibility-flag.v1` (KB-20 → review queue)

**Decision**: For v1.0, continue using HTTP POST for inter-KB communication (already working). Kafka topics are for observability/audit replays and asynchronous fanout. Implement Kafka publishing as a **secondary** channel alongside existing HTTP.

**Changes required**:

1. **`backend/shared-infrastructure/kafka/topics-config.yaml`** — Add topic definitions with appropriate partition counts and retention.

2. **Each publishing service** — Add optional Kafka publisher alongside HTTP POST. Controlled by env var `KAFKA_ENABLED=true|false` (default false for v1.0).

### Task 4.2: Apollo Federation KB-20+ Schema Integration (P3)

**Problem**: Apollo Federation only federates KB-1 through KB-7. KB-20, KB-21, KB-22, KB-23 have no GraphQL schemas and are not federated.

**Changes required** (lower priority — these KBs communicate via REST, not GraphQL):

1. **Create federated GraphQL schemas** for KB-20, KB-22, KB-23 exposing read-only query types.
2. **Update `apollo-federation/supergraph.yaml`** to include new service URLs.
3. **Regenerate supergraph schema**.

This is P3 because the KBs already communicate effectively via REST. Federation is for frontend dashboard queries only.

---

## PHASE 5: Documentation & Governance

### Task 5.1: Update All CLAUDE.md Files (P3)

After implementing all changes, update:
- Root [`CLAUDE.md`](CLAUDE.md) — Add KB-19 through KB-23 ports, V-MCU reference
- [`knowledge-base-services/CLAUDE.md`](backend/shared-infrastructure/knowledge-base-services/CLAUDE.md) — Add KB-19+ service descriptions and ports
- Create `vaidshala/CLAUDE.md` if not present — Document V-MCU architecture and commands

### Task 5.2: Create Integration Test Suite (P3)

**File to create**: `backend/shared-infrastructure/knowledge-base-services/integration_tests/v3_integration_test.go`

End-to-end test validating the full chain:
1. KB-20 receives lab result → publishes LAB_RESULT event
2. KB-22 receives HPI completion → fetches KB-20 stratum + KB-21 adherence + KB-23 perturbations
3. KB-22 publishes HPI_COMPLETE to KB-19 via `/api/v1/events`
4. KB-23 receives behavioral gap → computes MCU_GATE → publishes to KB-19
5. V-MCU runs cycle → Channel B evaluates → Channel C evaluates → Arbiter produces gate → dose computed

---

## Build Sequence & Dependencies

```
PHASE 0 (Port Registry)     ─── no dependencies, do first
    ↓
PHASE 1 (Wiring Fixes)      ─── depends on correct ports from Phase 0
    ├── Task 1.3 (KB-20 LAB_RESULT)     independent
    ├── Task 1.4 (eGFR trajectory)      depends on 1.3
    ├── Task 1.5 (AG-01 perturbation)   independent
    ├── Task 1.6 (AG-02 reliability)    independent
    └── Task 1.7 (adherence gain)       independent
    ↓
PHASE 2 (Safety Fixes)      ─── can start in parallel with Phase 1
    ├── Task 2.1 (data absence)         CRITICAL PATH — do first
    └── Task 2.2 (staleness config)     depends on 2.1
    ↓
PHASE 3 (New Specs)          ─── depends on Phases 1+2
    ├── Task 3.1 (DEPRESCRIBING)        depends on 2.1 (pointer types)
    ├── Task 3.2 (plausibility)         depends on 1.3 (LAB_RESULT)
    ├── Task 3.3 (SLA miss)             independent
    ├── Task 3.4 (autonomy limits)      independent
    └── Task 3.5 (simulation)           depends on ALL of 3.1–3.4
    ↓
PHASE 4 (Infrastructure)    ─── can start after Phase 1
    ├── Task 4.1 (Kafka topics)         independent
    └── Task 4.2 (Apollo Federation)    independent
    ↓
PHASE 5 (Documentation)     ─── after all other phases
```

---

## Corrected Priority Register

Items from the Final Architecture Proposal cross-checked against actual codebase:

| ID | Item | Original Priority | Corrected Priority | Status |
|----|------|------------------|--------------------|--------|
| ~~B1~~ | ~~outcome_publisher /events→/execute~~ | ~~P0~~ | ~~N/A~~ | **RESOLVED**: `/api/v1/events` is correct — KB-19 registers both endpoints |
| ~~B2~~ | ~~YAML label/label_en~~ | ~~P0~~ | ~~N/A~~ | **RESOLVED**: Semantic naming only, not functional |
| B3 | KB-21 port collision (8093) | P0 | **P0** | Task 0.2 |
| B4 | KB-22 KB5_URL port confusion | P0 | **P0 (verify)** | Task 0.3 — needs verification |
| B5 | V-MCU data absence false HALTs | P0 | **P0** | Task 2.1 |
| G1 | KB-20 LAB_RESULT event missing | P1 | **P1** | Task 1.3 |
| G2 | eGFR trajectory not wired | P1 | **P1** | Task 1.4 |
| G3 | AG-01 TreatmentPerturbation | P1 | **P1** | Task 1.5 |
| G4 | AG-02 answer-reliability endpoint | P1 | **P1** | Task 1.6 |
| G5 | Adherence-tier gain mapping | P1 | **P1** | Task 1.7 |
| S1 | DEPRESCRIBING_MODE | P2 | **P2** | Task 3.1 |
| S2 | Cross-session plausibility | P2 | **P2** | Task 3.2 |
| S3 | SLA miss behavior | P2 | **P2** | Task 3.3 |
| S4 | V-MCU autonomy limits | P2 | **P2** | Task 3.4 |
| S5 | Simulation harness (8 scenarios) | P2 | **P2** | Task 3.5 |
| E1 | Kafka topic provisioning | P3 | **P3** | Task 4.1 |
| E2 | Apollo Federation KB-20+ | P3 | **P3** | Task 4.2 |
| E3 | Documentation updates | P3 | **P3** | Task 5.1 |
| E4 | Integration test suite | P3 | **P3** | Task 5.2 |

---

## Success Criteria

### Pre-Integration Testing (After Phases 0-2)
- [ ] All KB services start without port collisions
- [ ] KB-22 → KB-19 `/api/v1/events` delivers HPI_COMPLETE successfully
- [ ] KB-23 → KB-19 `/api/v1/events` delivers MCU_GATE_CHANGED successfully
- [ ] KB-20 publishes LAB_RESULT events on new lab data
- [ ] V-MCU Channel B handles nil glucose without false HALT
- [ ] V-MCU Channel B handles stale data with HOLD_DATA

### Pre-Clinical Pilot (After Phase 3)
- [ ] DEPRESCRIBING_MODE allows controlled dose reduction
- [ ] Cross-session plausibility catches physiologically impossible lab sequences
- [ ] KB-23 SLA scanner detects and escalates overdue cards
- [ ] V-MCU autonomy limits prevent >50% cumulative dose change without confirmation
- [ ] All 8 simulation scenarios pass with correct SafetyTrace entries

### Production Readiness (After Phases 4-5)
- [ ] Kafka topics provisioned for audit replay
- [ ] All CLAUDE.md files reflect current architecture
- [ ] Integration test suite validates full KB-20→KB-22→KB-23→KB-19→V-MCU chain
