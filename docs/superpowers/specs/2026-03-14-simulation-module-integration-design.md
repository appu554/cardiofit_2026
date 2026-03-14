# Simulation Module Integration Design

**Date**: 2026-03-14
**Status**: Approved
**Scope**: Full 3-phase integration (A: Type Alignment, B: Safety Scenarios, C: Physiology Trajectories)
**Approach**: Bridge Adapter Pattern
**Location**: `vaidshala/simulation/`
**Reference**: `Vaidshala_Simulation_Implementation_Guidelines.docx`

---

## Summary

Integrate the standalone V-MCU simulation module (16 Go files, 3,004 lines across 6 packages) into the CardioFit monorepo at `vaidshala/simulation/`. A bridge adapter layer converts between simulation types and production V-MCU types, enabling the 13 safety scenarios and 5 physiology engines to validate the production engine without modifying either codebase. The existing production `engines/vmcu/simulation/` package is preserved — the bridge connects both worlds.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Module location | `vaidshala/simulation/` | Top-level peer to `clinical-runtime-platform/` |
| Integration pattern | Bridge Adapter | Guidelines doc recommends "thin adapter function". Preserves clinically validated scenarios unchanged. |
| Existing production simulation | Keep both, bridge them | Production `simulation/` package has tighter type integration. Standalone has clinically grounded scenarios + physiology engines. Bridge connects them. |
| CI | GitHub Actions (new) | No CI exists currently. Simulation becomes the first CI gate. |
| Coefficients | Configurable YAML | Population-specific profiles (default + South Asian). Extract all tunable parameters. |
| Timeline | All phases production-ready | No phased rollout — A, B, and C all built to production quality. |

---

## 1. Module Structure

```
vaidshala/simulation/
├── go.mod                          # module vaidshala/simulation (go 1.23.0+)
├── go.sum
│
├── bridge/                         # Integration layer (sim ↔ production)
│   ├── type_mapper.go              # RawPatientData, TitrationContext conversion
│   ├── result_mapper.go            # TitrationCycleResult conversion
│   ├── engine_adapter.go           # wraps production VMCUEngine.RunCycle()
│   ├── unit_converter.go           # mmol/L ↔ mg/dL (pass-through for now)
│   └── bridge_test.go              # round-trip conversion tests
│
├── pkg/                            # Standalone simulation (from /Downloads/simulation/)
│   ├── types/types.go              # Simulation's own types (unchanged)
│   ├── patient/archetypes.go       # 11 single-cycle archetypes used across 13 scenarios (10 original + SeasonalHyponatraemia)
│   ├── harness/
│   │   ├── channel_b.go            # 18 rules (reference implementation)
│   │   ├── channel_c.go            # 9 rules (reference implementation)
│   │   ├── vmcu_engine.go          # Simulation's own engine (for comparison)
│   │   └── multicycle.go           # 90-day trajectory runner
│   ├── scenarios/
│   │   ├── registry.go             # Scenario registry with tags
│   │   ├── simulation_test.go      # 13 scenarios against SIMULATION engine
│   │   └── production_test.go      # 13 scenarios against PRODUCTION engine (via bridge)
│   └── physiology/                 # Layer 2 — 5 engines (migrated from v2 delivery)
│       ├── body_composition.go     # Engine 1: visceral fat, insulin sensitivity (from body_engine.go)
│       ├── glucose.go              # Engine 2: FBG, PPBG, HbA1c, beta cell (from glucose_engine.go)
│       ├── hemodynamic.go          # Engine 3: SBP/DBP equilibrium-seeking (from hemodynamic_engine.go)
│       ├── renal.go                # Engine 4: eGFR, creatinine, K+, Na+, ACR (from renal_engine.go)
│       ├── observation.go          # Engine 5: Gaussian noise generator
│       ├── state.go                # PhysiologyState vector (~40 variables)
│       ├── config.go               # LoadPopulationConfig() from YAML
│       ├── archetypes.go           # 4 trajectory archetypes
│       └── trajectory_test.go      # 7 trajectory tests + untreated control
│
├── config/
│   ├── default.yaml                # Default population coefficients
│   └── south_asian.yaml            # South Asian phenotype overrides
│
├── ci/
│   └── simulation-gate.yml         # GitHub Actions CI workflow
│
└── cmd/
    └── main.go                     # CLI runner (see Section 11 for CLI specification)
```

## 2. Bridge Type Mapping

### Type Divergence Table

| Aspect | Simulation | Production | Bridge Action |
|--------|-----------|------------|---------------|
| GateSignal | `int` (CLEAR=0..HALT=4) | `string` ("CLEAR".."HALT") | Enum ↔ string lookup table |
| Lab values (glucose, K+, Cr, etc.) | `float64` | `*float64` | Value → pointer; nil → zero-value with flag |
| SBP, DBP, HeartRate | `int` | `*float64` | `int` → `float64` → pointer wrapping |
| Glucose units | mmol/L | mmol/L | Pass-through |
| Creatinine units | µmol/L | µmol/L | Pass-through |
| Timestamps | `time.Time` | `*time.Time` | Value → pointer |
| Channel B result | `string` rule fired | `ChannelBResult` struct | Extract `.RuleFired` field |
| Channel C result | `string` rule fired | `ChannelCResult` struct | Extract `.RuleID` field |
| Arbiter input | `ArbiterInput{3 GateSignal(int)}` | `ArbiterInput{3 GateSignal(string)}` | Convert each gate signal |
| Arbiter output `AllChannels` | `map[Channel]GateSignal` | `ArbiterInput` struct | Map ↔ struct field extraction |
| Arbiter output `DominantChannel` | `Channel` (string alias) | `string` | Type alias ↔ bare string |

### Round-Trip Safety

The bridge is bidirectional and round-trip safe:

```
sim_input → ToProductionInput() → production RunCycle() → ToSimulationResult() → sim assertions
```

`bridge_test.go` verifies: `ToSimulationInput(ToProductionInput(x)) == x` for every archetype. If the round-trip ever breaks, bridge tests fail before any scenario can give a false pass.

### Unit Converter

`unit_converter.go` provides boundary functions that are currently pass-through:

```go
func GlucoseToProduction(simValue float64) float64   // mmol/L → mmol/L
func CreatinineToProduction(simValue float64) float64 // µmol/L → µmol/L
```

Conversion factors documented as constants: `mmol/L = mg/dL ÷ 18.0` (glucose), `µmol/L = mg/dL × 88.4` (creatinine). If units ever diverge, one file changes.

## 3. Phase A — Type Alignment

### Compile-Time Guarantees

The bridge imports both `pkg/types` (simulation) and `engines/vmcu` (production). Type drift is caught at build time — if either side renames a field or changes a type, the bridge won't compile.

### Test-Time Guarantees (`bridge_test.go`)

1. **Field coverage test**: Every simulation `RawPatientData` field has a mapping to production. Adding a field without a mapping fails the test.
2. **Round-trip test**: For each of 11 archetypes, convert sim→production→sim and assert equality.
3. **GateSignal equivalence test**: All 5 values map correctly in both directions. Ordering invariant preserved.
4. **Arbiter compatibility test**: Run `Arbitrate()` on both simulation and production arbiters for all 125 combinations. Results must be identical.

## 4. Phase B — Safety Scenario Integration

### Dual Test Strategy

| Test File | Engine | Purpose |
|-----------|--------|---------|
| `simulation_test.go` | `harness.VMCUEngine` | Baseline — proves scenarios are correct |
| `production_test.go` | `bridge.ProductionEngine` | Validation — proves production V-MCU matches |

Both share same archetypes and expected outcomes. 11 archetypes are used across 13 scenarios (not 1:1 — Scenario 11 uses 2 archetypes in sequence for freeze/resume, Scenario 12 uses no archetypes for the arbiter sweep, Scenario 13 adds a new seasonal hyponatraemia archetype). Discrepancy = production bug or bridge bug, never test bug.

### Engine Adapter

`bridge/engine_adapter.go`:

```go
type ProductionEngine struct {
    engine *vmcu.VMCUEngine
}

func NewProductionEngine() *ProductionEngine
func (pe *ProductionEngine) RunCycle(input types.TitrationCycleInput) types.TitrationCycleResult
```

Production `VMCUEngine` constructed with no infrastructure: `safetyCache: nil`, `asyncTracer: nil`, `holdResponder: nil`, `eventPublisher: nil`. Zero network calls.

**ProtocolRulesPath**: Production's `NewVMCUEngine()` calls `channel_c.LoadRules(cfg.ProtocolRulesPath)` which reads `protocol_rules.yaml` from disk. The bridge resolves this path using two strategies:

1. **`//go:embed` in test fixture** (preferred): Copy `protocol_rules.yaml` into `bridge/testdata/protocol_rules.yaml` and embed it at build time. This eliminates working-directory sensitivity and makes the simulation fully self-contained for CI. The embedded copy must be refreshed when the production YAML changes — the CI gate verifies `sha256(bridge/testdata/protocol_rules.yaml) == sha256(../clinical-runtime-platform/engines/vmcu/protocol_rules.yaml)` and blocks merge on mismatch.

2. **Explicit path parameter** (fallback): `NewProductionEngine(opts ...EngineOption)` accepts `WithProtocolRulesPath(path string)` for cases where the caller needs to point to a specific YAML file. CI workflow sets this explicitly via environment variable `SIMULATION_PROTOCOL_RULES_PATH`.

This is the ONE file I/O dependency — all other inputs are in-memory.

### Production's Extra Rules

Production has 25 Channel B + expanded Channel C (vs simulation's 18+9). Extra rules (DA-01..DA-08, B-19, PG-15, PG-16) don't interfere with the original 12 scenarios because:

- DA rules: Archetypes have valid data with recent timestamps → evaluate CLEAR
- B-19: Now covered by new Scenario 13 (seasonal hyponatraemia archetype)
- PG-15/PG-16: Require specific boolean flags not set by archetypes

### Scenario 3 (RAAS Tolerance) — Special Handling

Bridge must map BOTH:
- `sim.RawPatientData.CreatinineRiseExplained` → `prod.RawPatientData.CreatinineRiseExplained`
- AND → `prod.TitrationContext.RAASCreatinineTolerant`

Both must be `true` for PG-14 downgrade to fire from both channels.

**Rule ID divergence**: The simulation fires rule `B-04+PG-14` for the RAAS tolerance downgrade. Production uses rule ID `B-03-RAAS-SUPPRESSED` for the Channel B suppression. The bridge's `result_mapper.go` must normalize these IDs. The `production_test.go` expected rule attribution for Scenario 3 must use production's rule ID (`B-03-RAAS-SUPPRESSED`), not the simulation's (`B-04+PG-14`). This is the ONE scenario where expected rule attribution differs between `simulation_test.go` and `production_test.go`.

### Rule ID Mapping Table

The bridge must include an exhaustive `RuleIDNormalization` map that is tested at build time. If a production rule ID is missing from this map, the bridge must panic (not silently pass). This table must be verified during implementation by comparing every `checkBxx()` function in production's `monitor.go` against every `ruleXxx()` function in the simulation's `channel_b.go`.

| Simulation Rule ID | Production Rule ID | Clinical Condition | Verified Match |
|---|---|---|---|
| B-01 | B-01 | Glucose <3.9 mmol/L (hypoglycaemia) | Gate + threshold |
| B-02 | B-07 | Glucose declining + recent dose + <5.5 | Gate + logic |
| B-03 | B-04 | K+ <3.0 or >6.0 (extreme electrolyte) | Gate + thresholds |
| B-04 | B-03 | Creatinine 48h delta >26 µmol/L (AKI) | Gate + delta calc |
| B-04+PG-14 | B-03-RAAS-SUPPRESSED | RAAS tolerance downgrade | **DIVERGENT** — only entry where rule attribution differs |
| B-05 | B-06 | Weight 72h delta >2.5 kg | Gate + delta calc |
| B-06 | DA-01..DA-05 | Physiologically impossible values | Production splits into 5 DA rules |
| B-07 | B-08 | eGFR <15 (CKD Stage 5) | Gate + threshold |
| B-08 | B-05 | SBP <90 (hypotension) | Gate + threshold |
| B-09 | B-09 | eGFR 15-29 (CKD Stage 4) | Gate + threshold |
| B-10 (DA-06) | DA-06 | Stale K+ >14 days | Gate + staleness |
| B-11 (DA-07) | DA-07 | Stale creatinine >30 days | Gate + staleness |
| B-12 | B-12 | J-curve eGFR-stratified BP floor | Gate + CKD stage map |
| B-13 | B-13 | HR <45 resting (severe bradycardia) | Gate + threshold |
| B-14 | B-14 | HR <55 + beta-blocker + recent dose | Gate + conditions |
| B-15 | B-15 | HR >120 resting (tachycardia) | Gate + threshold |
| B-16 | B-16 | Irregular HR (possible AF) | Gate + KB22 trigger |
| B-17 | B-17 | Na+ <132 + thiazide (severe hyponatraemia) | Gate + threshold |
| B-18 | B-18 | Na+ 132-135 + thiazide (mild hyponatraemia) | Gate + threshold |
| — | B-10 | eGFR slope <-5 mL/min/year (rapid decline) | Production-only, no sim equivalent |
| — | B-11 | Beta-blocker + glucose <4.5 (raised threshold) | Production-only, no sim equivalent |
| — | B-19 | Na+ <135 + SUMMER + thiazide (seasonal) | Production-only — covered by new Scenario 13 |
| — | DA-08 | On RAAS + creatinine >14d stale | Production-only, no sim equivalent |
| PG-01..PG-08 | PG-01..PG-08 | Same IDs | Verified match |
| PG-14 | PG-14 | RAAS creatinine tolerance | Same ID (but see B-04+PG-14 above) |
| — | PG-09..PG-13, PG-15, PG-16 | HTN composite conditions | Production-only, no sim equivalent |

**NOTE**: Production-only rules (marked "—" in simulation column) have no simulation equivalent. These rules are tested through production's own test suite at `engines/vmcu/`, not through the simulation bridge. The bridge's `RuleIDNormalization` map must include them with a `PRODUCTION_ONLY` marker so the bridge doesn't panic on unknown IDs — it simply passes them through without asserting rule attribution.

### Scenario 13 (NEW) — Seasonal Hyponatraemia (B-19)

Added to cover production's B-19 rule, which has zero test coverage from the original 12 scenarios. This is clinically important for the Indian deployment — summer amplifies thiazide-induced electrolyte losses.

**Archetype: `SeasonalHyponatraemia`**
- `SodiumCurrent: 134 mmol/L` (below 135 threshold)
- `ThiazideActive: true`
- `Season: "SUMMER"`
- All other labs normal

**Expected**: `PAUSE` (B-19). Not HALT — Na+ is 134 (above the 132 severe threshold for B-17). This tests the seasonal modifier that only exists in production's Channel B, not in the simulation's 18-rule set. Therefore Scenario 13 only runs in `production_test.go`, not `simulation_test.go` (the simulation engine lacks B-19).

This brings production Channel B coverage from 18/25 to 19/25 rules tested via simulation scenarios.

### Production Extra Steps Impact Analysis

Production's `RunCycle()` has post-arbiter steps that the simulation engine does not. Each has been verified per-scenario:

| Production Step | Impact on 12 Scenarios | Reason |
|---|---|---|
| BP-velocity modulation (`classifyBPStatus`) | No impact. SBP values in archetypes (102, 135, etc.) fall between 90-140, classified as "" (no category). Neutral multiplier. | Only fires at SBP extremes (<90 or >140) |
| Metabolic engine (KB-24) | No impact. Bridge sets `MetabolicInput: nil`, which skips KB-24 enrichment. | Optional enrichment, not safety-critical |
| Deprescribing suppression | No impact. No archetype has `DeprescribingContext.Active = true`. | Only affects active deprescribing plans |
| Cooldown tracker | Bridge must set `LastDoseChangeTime` to >48h ago so cooldown doesn't block dose. | Scenario 9 (GreenTrajectory) would fail if cooldown blocks the expected dose application |

### Discrepancy Investigation Order

1. **Bridge mapping error** (most likely) → fix adapter
2. **Production engine bug** → fix engine
3. **Simulation expectation wrong** (least likely) → only if clinical reasoning disproven

**Never modify expected outcomes to make tests pass.**

## 5. Phase C — Physiological Trajectory Integration

### Closed-Loop Architecture

```
Day N:
  BodyComposition.Step()  →  insulinSensitivity
  Glucose.Step()          →  FBG, PPBG, HbA1c       (uses insulinSensitivity)
  Hemodynamic.Step()      →  SBP, DBP, HR            (uses visceralFatIndex)
  Renal.Step()            →  eGFR, Cr, K+, Na+       (uses SBP, FBG)
  Observation.Generate()  →  noisy RawPatientData
                               │
                    bridge.ProductionEngine.RunCycle()
                               │
                    Apply dose delta to physiology medication profile
                               │
                    Day N+1: repeat
```

Engine execution order is fixed: Body → Glucose → Hemodynamic → Renal. Each depends on the previous engine's output.

**NOTE**: The 5 physiology engines are **migrated and adapted from the v2 delivery** (`glucose_engine.go`, `hemodynamic_engine.go`, `renal_engine.go`, `body_engine.go`, `simulator.go`). The v2 engines are the authoritative implementation — already built, tuned to published trial data, and passing 7 trajectory tests. The existing `harness/multicycle.go` from v1 was a skeleton with simplified linear models; v2 replaced it with full equilibrium-seeking models (~40 state variables). Migration involves renaming files to match this spec's conventions and wiring the bridge adapter for closed-loop integration with the production `RunCycle()`.

### Configurable Coefficients (`config/default.yaml`)

```yaml
population: "default"
version: "1.0"

body_composition:
  visceral_fat_insulin_threshold: 1.4
  muscle_sensitivity_weight: 0.3
  sglt2i_calorie_loss_kcal: 250
  glp1ra_appetite_reduction_pct: 0.15

glucose:
  equilibrium_drift_rate: 0.10
  beta_cell_decline_rate: 0.001
  glucotoxicity_threshold_mmol: 10.0    # 180 mg/dL ÷ 18 = 10.0 mmol/L
  glucotoxicity_multiplier: 2.5
  carb_baseline_g: 250
  ppbg_spike_coefficient: 0.15

hemodynamic:
  sbp_drift_rate: 0.05
  acei_arb_effect_mmhg: -8
  thiazide_effect_mmhg: -10
  ccb_effect_mmhg: -7
  beta_blocker_effect_mmhg: -5
  sglt2i_bp_effect_mmhg: -4

renal:
  natural_egfr_decline_per_year: 1.0
  acei_arb_protection_pct: 0.35
  sglt2i_protection_pct: 0.30
  glp1ra_protection_pct: 0.20
  uncontrolled_sbp_threshold: 140
  high_glucose_threshold_mmol: 10.0

# All noise values in the same units as the engines (mmol/L, mmHg, µmol/L, kg)
observation_noise:
  glucose_stddev_mmol: 0.28             # 5.0 mg/dL ÷ 18 ≈ 0.28 mmol/L
  bp_stddev_mmhg: 3.0
  potassium_stddev_mmol: 0.1
  creatinine_stddev_umol: 5.0
  weight_stddev_kg: 0.3

simulation:
  random_seed: 42
  total_days: 90
  cycles_per_day: 1

autonomy:
  single_step_pct: 0.20
  cumulative_pct: 0.50
```

South Asian override (`config/south_asian.yaml`):

```yaml
population: "south_asian"
extends: "default"

body_composition:
  visceral_fat_insulin_threshold: 1.2

glucose:
  carb_baseline_g: 350
  ppbg_spike_coefficient: 0.18
```

### Trajectory Validation Criteria

| Archetype | Key Assertions |
|-----------|---------------|
| VisceralObesePatient | FBG declines (158→141+), HbA1c improves, SBP declines |
| CKDProgressorPatient | eGFR decline rate ≤0.7 (vs 1.3 untreated). ACR improving |
| ElderlyFrailPatient | FBG stays 125-140. Zero HALT from hypoglycaemia |
| GoodResponderPatient | FBG drops significantly, HbA1c→6.5, deprescribing trajectory |
| Untreated control | FBG rising, HbA1c rising, eGFR declining |

Statistical assertions for noise: HALT rate <8%, fixed seed=42, range assertions not exact values.

### Autonomy Limit Validation

- `abs(finalDose - initialApprovedDose) / initialApprovedDose ≤ 0.50` for all archetypes
- No single-step delta exceeds ±20%
- Post-HALT resume applies 50% dampening

## 6. Error Handling & Edge Cases

### Bridge Failure Modes

| Failure Mode | Detection | Response |
|---|---|---|
| Nil pointer (missing lab) | Bridge sets `*float64` nil when `DataAvailable=false` | Archetypes set explicit values; nil only for intentional missing data |
| GateSignal mapping miss | Exhaustive switch + `default: panic()` | Go exhaustive switch linter catches at build |
| Extra production fields | `ToSimulationResult()` extracts only needed fields | Production-only features ignored |
| Missing production fields | Bridge sets defaults: `MetabolicInput: nil`, `GainFactor: 1.0` | Defaults documented in `engine_adapter.go` |

### Observation Noise & Determinism

- Fixed seed per `config/default.yaml` (`random_seed: 42`)
- Per-archetype RNG — tests can run `t.Parallel()`
- Statistical assertions where noise makes exact values impossible

### Timestamp Sensitivity

- Timestamps constructed relative to fixed `simTime`, not `time.Now()`
- `bridge.WithSimulatedTime(t)` option for controlled clock
- Scenario 4: K+ timestamp = `simTime - 16 days` (always exactly 16 days stale)
- Scenario 11: range assertions (5-6 cycles, not exactly 5)

### Safety Invariants (Always True)

Checked by `validateInvariants(result)` after every cycle in both Layer 1 and Layer 2:

1. HALT → `DoseApplied=false`, `DoseDelta=0.0`
2. `FinalGate >= max(ChannelA, ChannelB, ChannelC)`
3. Any channel HALT → final HALT
4. All channels CLEAR → final CLEAR
5. Single-step delta ≤ ±20% of last approved dose
6. Cumulative drift ≤ ±50% from physician-approved dose
7. Post-resume first dose ≤ 50% of normal
8. Every `RunCycle()` produces exactly one `SafetyTrace`

## 7. CI Gate

### GitHub Actions Workflow (`ci/simulation-gate.yml`)

Triggers on PRs touching `vaidshala/clinical-runtime-platform/engines/vmcu/**` or `vaidshala/simulation/**`.

Steps:
1. Checkout + Go 1.23.0+ setup (must match production's `go.mod` directive)
2. Bridge compile check: `go build ./bridge/...`
3. Bridge round-trip tests: `go test ./bridge/ -v`
4. Layer 1 simulation scenarios: `go test ./pkg/scenarios/ -run TestSimulation -v -count=1`
5. Layer 1 production scenarios: `go test ./pkg/scenarios/ -run TestProduction -v -count=1`
6. Arbiter exhaustive sweep: `go test ./pkg/scenarios/ -run TestArbiter -v -count=1`
7. Layer 2 trajectory tests: `go test ./pkg/physiology/ -v -count=1 -timeout=300s`
8. Performance benchmark: `go test ./bridge/ -bench=BenchmarkRunCycle -benchmem`

### Failure Policy

| Step | Failure Action |
|------|---------------|
| Bridge compile | Block merge |
| Bridge round-trip | Block merge |
| Any scenario (simulation) | Block merge |
| Any scenario (production) | Block merge |
| Arbiter <125/125 | Block merge |
| Trajectory assertions | Block merge |
| Benchmark regression >3x baseline | Warn (not block) |

No exceptions. No skipped tests. `-count=1` disables test caching.

### Scenario Coverage Tracking

Registry `Tags` field maps scenarios to rules. CI post-step warns (not blocks) if a production rule has no tagged scenario. New rules get a grace period for scenario authoring.

## 8. FactStore Phase Interaction

| Phase | Production Change | Simulation Impact | Action |
|---|---|---|---|
| Phase 0 | KB-20 projection endpoints | None | Optional divergence test (separate from CI gate) |
| Phase 1 | RunCycle auto-fetch if RawLabs==nil | None — bridge always provides explicit RawLabs | No change |
| Phase 2 | KB-20 HTN composite booleans | None — bridge sets TitrationContext booleans explicitly | No change |

**Hard rule**: Simulation never imports KB-20, KB-22, KB-23, or any KB service. Zero network, zero database, pure-function testing.

## 9. Production Dependency

The `go.mod` uses a `replace` directive:

```
module vaidshala/simulation

go 1.23.0

require vaidshala/clinical-runtime-platform v0.0.0

replace vaidshala/clinical-runtime-platform => ../clinical-runtime-platform
```

**NOTE**: Module path is `vaidshala/simulation` (no `github.com/` prefix), matching the production module's path convention (`vaidshala/clinical-runtime-platform`). Go version must be `1.23.0` or later to match production's `go.mod` directive — the simulation cannot import the production module with a lower Go version.

Simulation always tests against the local branch version. No published versions, no version skew.

## 10. When to Add New Scenarios

Add a scenario when:
- New Channel B or Channel C rule added
- Clinical edge case discovered during pilot
- Arbiter logic changes
- Integrator freeze/resume modified
- Autonomy limits updated

Pattern: define archetype → define expected gate → assert gate + rule attribution → tag with rule IDs.

## 11. CLI Specification (`cmd/main.go`)

The CLI provides quick local validation before pushing:

```bash
# Default: run all 13 scenarios against SIMULATION engine
go run cmd/main.go

# --production: run all 13 scenarios against PRODUCTION engine (via bridge)
go run cmd/main.go --production

# --trajectory: additionally run 90-day trajectory tests (slower)
go run cmd/main.go --production --trajectory
```

**Default mode**: Runs 12 scenarios (1-12) against the simulation's own `harness.VMCUEngine`. Prints a summary table with columns: `#`, `SCENARIO`, `GATE`, `DOSE (YES/NO)`, `DELTA`, `B-RULE`, `C-RULE`, `STATUS (PASS/FAIL)`. Appends arbiter sweep result (125/125). Exits 0 if all pass, 1 if any fail.

**--production mode**: Same output format, but runs all 13 scenarios (1-13, including Scenario 13 which is production-only) via `bridge.ProductionEngine`. Requires `protocol_rules.yaml` to be resolvable (see Section 4).

**--trajectory mode**: After scenarios, runs the 4 trajectory archetypes for 90 days each. Prints per-archetype summary: initial/final FBG, HbA1c, eGFR, dose, HALT count, and pass/fail against trajectory assertions.
