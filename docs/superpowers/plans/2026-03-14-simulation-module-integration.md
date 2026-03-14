# Simulation Module Integration — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate the standalone V-MCU simulation harness into the CardioFit monorepo at `vaidshala/simulation/`, with a bridge adapter connecting simulation types to production V-MCU types, enabling 13 safety scenarios and 5 physiology engines to validate the production engine as a CI gate.

**Architecture:** Bridge Adapter Pattern. The standalone simulation (3,004 lines, 6 packages) is copied into `vaidshala/simulation/pkg/`. A `bridge/` package converts between simulation types (`int` GateSignal, `float64` labs) and production types (`string` GateSignal, `*float64` labs). Production V-MCU is imported via `replace` directive and called through the bridge with zero network dependencies.

**Tech Stack:** Go 1.23.0+, YAML (gopkg.in/yaml.v3) for config, GitHub Actions for CI

**Spec:** `docs/superpowers/specs/2026-03-14-simulation-module-integration-design.md`

---

## File Map

### New files to create

| File | Responsibility |
|------|---------------|
| `vaidshala/simulation/go.mod` | Module definition with replace directive |
| `vaidshala/simulation/bridge/type_mapper.go` | sim RawPatientData/TitrationContext ↔ prod types |
| `vaidshala/simulation/bridge/result_mapper.go` | prod TitrationCycleResult → sim types + RuleIDNormalization |
| `vaidshala/simulation/bridge/engine_adapter.go` | ProductionEngine wrapper around prod VMCUEngine.RunCycle() |
| `vaidshala/simulation/bridge/unit_converter.go` | Unit conversion boundary (pass-through for now) |
| `vaidshala/simulation/bridge/bridge_test.go` | Round-trip, field coverage, GateSignal, arbiter tests |
| `vaidshala/simulation/bridge/testdata/protocol_rules.yaml` | Embedded copy of production protocol rules |
| `vaidshala/simulation/pkg/scenarios/registry.go` | Scenario registry with tags |
| `vaidshala/simulation/pkg/scenarios/production_test.go` | 13 scenarios against production via bridge |
| `vaidshala/simulation/pkg/patient/seasonal_archetype.go` | SeasonalHyponatraemia archetype (Scenario 13) |
| `vaidshala/simulation/pkg/physiology/config.go` | LoadPopulationConfig() from YAML |
| `vaidshala/simulation/config/default.yaml` | Default population coefficients |
| `vaidshala/simulation/config/south_asian.yaml` | South Asian phenotype overrides |
| `.github/workflows/simulation-gate.yml` | GitHub Actions CI workflow (must be at repo root) |

### Files to copy from standalone simulation (`/Downloads/simulation/`)

| Source | Destination | Changes |
|--------|------------|---------|
| `pkg/types/types.go` | `vaidshala/simulation/pkg/types/types.go` | Update module path in imports |
| `pkg/patient/archetypes.go` | `vaidshala/simulation/pkg/patient/archetypes.go` | Update module path |
| `pkg/harness/channel_b.go` | `vaidshala/simulation/pkg/harness/channel_b.go` | Update module path |
| `pkg/harness/channel_c.go` | `vaidshala/simulation/pkg/harness/channel_c.go` | Update module path |
| `pkg/harness/vmcu_engine.go` | `vaidshala/simulation/pkg/harness/vmcu_engine.go` | Update module path |
| `pkg/harness/multicycle.go` | `vaidshala/simulation/pkg/harness/multicycle.go` | Update module path |
| `pkg/scenarios/simulation_test.go` | `vaidshala/simulation/pkg/scenarios/simulation_test.go` | Update module path |
| `cmd/main.go` | `vaidshala/simulation/cmd/main.go` | Add --production + --trajectory flags |

### Files to migrate from v2 delivery physiology engines

| Source (v2 tarball) | Destination | Changes |
|---------------------|------------|---------|
| `pkg/physiology/body_engine.go` | `vaidshala/simulation/pkg/physiology/body_composition.go` | Rename, update imports, extract config |
| `pkg/physiology/glucose_engine.go` | `vaidshala/simulation/pkg/physiology/glucose.go` | Rename, update imports, extract config |
| `pkg/physiology/hemodynamic_engine.go` | `vaidshala/simulation/pkg/physiology/hemodynamic.go` | Rename, update imports, extract config |
| `pkg/physiology/renal_engine.go` | `vaidshala/simulation/pkg/physiology/renal.go` | Rename, update imports, extract config |
| `pkg/physiology/simulator.go` | `vaidshala/simulation/pkg/physiology/observation.go` | Extract observation generator |
| `pkg/physiology/state.go` | `vaidshala/simulation/pkg/physiology/state.go` | Update imports |
| `pkg/physiology/archetypes.go` | `vaidshala/simulation/pkg/physiology/archetypes.go` | Update imports |
| `pkg/physiology/trajectory_test.go` | `vaidshala/simulation/pkg/physiology/trajectory_test.go` | Update imports |

### Production files referenced (read-only, never modified)

| File | Used by |
|------|---------|
| `vaidshala/clinical-runtime-platform/engines/vmcu/vmcu_engine.go` | bridge/engine_adapter.go |
| `vaidshala/clinical-runtime-platform/engines/vmcu/types/gate_signal.go` | bridge/type_mapper.go |
| `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/raw_inputs.go` | bridge/type_mapper.go |
| `vaidshala/clinical-runtime-platform/engines/vmcu/channel_c/protocol_guard.go` | bridge/type_mapper.go |
| `vaidshala/clinical-runtime-platform/engines/vmcu/arbiter/arbiter.go` | bridge/bridge_test.go |
| `vaidshala/clinical-runtime-platform/engines/vmcu/protocol_rules.yaml` | bridge/testdata/ (copy) |

---

## Chunk 1: Module Scaffolding & Standalone Migration

### Task 1: Create Go module and copy standalone simulation

**Files:**
- Create: `vaidshala/simulation/go.mod`
- Copy: All 8 standalone files from `/Downloads/simulation/pkg/` and `/Downloads/simulation/cmd/`

- [ ] **Step 1: Create the simulation directory structure**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
mkdir -p vaidshala/simulation/{bridge/testdata,pkg/{types,patient,harness,scenarios,physiology},config,cmd}
mkdir -p .github/workflows  # CI workflow goes at repo root
```

- [ ] **Step 2: Create go.mod**

Create `vaidshala/simulation/go.mod`:
```
module vaidshala/simulation

go 1.23.0

require (
    vaidshala/clinical-runtime-platform v0.0.0
    gopkg.in/yaml.v3 v3.0.1
)

replace vaidshala/clinical-runtime-platform => ../clinical-runtime-platform
```

Run: `cd vaidshala/simulation && go mod tidy`
Expected: go.sum generated, no errors

- [ ] **Step 3: Copy standalone simulation files**

Copy the 8 core files from `/Downloads/simulation/` to `vaidshala/simulation/`:
- `pkg/types/types.go`
- `pkg/patient/archetypes.go`
- `pkg/harness/channel_b.go`
- `pkg/harness/channel_c.go`
- `pkg/harness/vmcu_engine.go`
- `pkg/harness/multicycle.go`
- `pkg/scenarios/simulation_test.go`
- `cmd/main.go`

In each file, update the module import path from `github.com/vaidshala/simulation` to `vaidshala/simulation`.

Also add these fields to `TitrationContext` in `pkg/types/types.go` (needed by production bridge):
```go
    CKDStage           string // "3a", "3b", "4", "5" — for B-12 J-curve stratification
    OliguriaReported   bool   // overrides RAAS tolerance (B-03)
    HypoWithin7d       bool   // any hypoglycaemia event in last 7 days (PG-07)
```

Then set `CKDStage` on existing archetypes that need it:
- `JCurveCKD3b()`: add `CKDStage: "3b"` to Context
- `MetforminCKD4()`: add `CKDStage: "4"` to Context
- `SeasonalHyponatraemia()`: add `CKDStage: "2"` to Context (eGFR 65)

Also add `LastHbA1c` and `LastEGFR` to `PatientTimestamps` in `VirtualPatient`:
```go
    LastHbA1c time.Time
    LastEGFR  time.Time
```
Set these to `now` in all archetypes that provide HbA1c/eGFR values.

- [ ] **Step 4: Verify standalone simulation compiles and tests pass**

Run: `cd vaidshala/simulation && go test ./pkg/scenarios/ -v -count=1`
Expected: All 12 scenarios PASS (Scenario 13 is added later, production_test.go doesn't exist yet)

Run: `cd vaidshala/simulation && go run cmd/main.go`
Expected: Summary table with 12/12 pass, 125/125 arbiter

- [ ] **Step 5: Commit**

```bash
git add vaidshala/simulation/
git commit -m "feat: scaffold simulation module and migrate standalone harness

Copy 8 core files from standalone simulation. Update module
paths. 12 scenarios + 125 arbiter combinations passing."
```

---

### Task 2: Add SeasonalHyponatraemia archetype (Scenario 13)

**Files:**
- Create: `vaidshala/simulation/pkg/patient/seasonal_archetype.go`
- Modify: `vaidshala/simulation/pkg/scenarios/simulation_test.go`

- [ ] **Step 1: Write the archetype**

Create `vaidshala/simulation/pkg/patient/seasonal_archetype.go`:
```go
package patient

import (
    "vaidshala/simulation/pkg/types"
    "time"
)

// SeasonalHyponatraemia creates a patient with borderline sodium in summer
// on thiazide. Tests production B-19 rule (seasonal Na+ threshold).
// Na+ 134 is below the 135 seasonal threshold but above 132 severe (B-17).
// Expected: PAUSE from B-19 (production-only rule, not in simulation engine).
func SeasonalHyponatraemia() *VirtualPatient {
    now := time.Now()
    return &VirtualPatient{
        Name: "SeasonalHyponatraemia",
        RawLabs: types.RawPatientData{
            GlucoseCurrent:     7.0,   // normal
            GlucosePrevious:    7.2,
            CreatinineCurrent:  80,    // normal
            CreatininePrevious: 78,
            PotassiumCurrent:   4.2,   // normal
            EGFR:               65,    // CKD Stage 2
            SBP:                135,
            DBP:                85,
            HeartRate:          72,
            HeartRateRegularity: "REGULAR",
            Weight:             75.0,
            WeightPrevious:     75.0,
            SodiumCurrent:      134.0, // below 135 seasonal threshold
            HbA1c:              7.0,
        },
        Context: types.TitrationContext{
            CurrentDose:       10.0,
            ProposedDoseDelta: 2.0,
            ThiazideActive:    true,
            Season:            "SUMMER",
            EGFRCurrent:       65,
        },
        MCUGate:        types.CLEAR,
        AdherenceScore: 0.85,
        LoopTrustScore: 1.0,
        Timestamps: PatientTimestamps{
            LastGlucose:    now,
            LastCreatinine: now,
            LastPotassium:  now,
        },
    }
}
```

Note: The exact struct fields depend on the standalone simulation's `VirtualPatient` type. Verify field names match `pkg/patient/archetypes.go` and adapt accordingly.

- [ ] **Step 2: Verify it compiles**

Run: `cd vaidshala/simulation && go build ./pkg/patient/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add vaidshala/simulation/pkg/patient/seasonal_archetype.go
git commit -m "feat: add SeasonalHyponatraemia archetype for Scenario 13 (B-19)"
```

---

## Chunk 2: Bridge — Type Mapper & Unit Converter

### Task 3: Create unit converter (pass-through)

**Files:**
- Create: `vaidshala/simulation/bridge/unit_converter.go`
- Create: `vaidshala/simulation/bridge/unit_converter_test.go`

- [ ] **Step 1: Write the test**

Create `vaidshala/simulation/bridge/unit_converter_test.go`:
```go
package bridge

import (
    "math"
    "testing"
)

func TestGlucoseConversion_PassThrough(t *testing.T) {
    // Both sides use mmol/L — pass-through
    input := 5.5
    if got := GlucoseToProduction(input); got != input {
        t.Errorf("GlucoseToProduction(%v) = %v, want %v", input, got, input)
    }
    if got := GlucoseToSimulation(input); got != input {
        t.Errorf("GlucoseToSimulation(%v) = %v, want %v", input, got, input)
    }
}

func TestCreatinineConversion_PassThrough(t *testing.T) {
    input := 90.0
    if got := CreatinineToProduction(input); got != input {
        t.Errorf("CreatinineToProduction(%v) = %v, want %v", input, got, input)
    }
    if got := CreatinineToSimulation(input); got != input {
        t.Errorf("CreatinineToSimulation(%v) = %v, want %v", input, got, input)
    }
}

func TestConversionConstants_Documented(t *testing.T) {
    // Verify constants are correct for future use
    mgdl := 180.0
    mmol := mgdl / GlucoseMgDLToMmolL
    if math.Abs(mmol-10.0) > 0.01 {
        t.Errorf("180 mg/dL should be ~10.0 mmol/L, got %v", mmol)
    }

    crMgDL := 1.0
    crUmol := crMgDL * CreatinineMgDLToUmolL
    if math.Abs(crUmol-88.4) > 0.1 {
        t.Errorf("1.0 mg/dL creatinine should be ~88.4 µmol/L, got %v", crUmol)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestGlucose -v`
Expected: FAIL — `GlucoseToProduction` not defined

- [ ] **Step 3: Write implementation**

Create `vaidshala/simulation/bridge/unit_converter.go`:
```go
package bridge

// Conversion constants — documented for future use if units diverge.
// Currently both simulation and production use mmol/L (glucose) and µmol/L (creatinine).
const (
    GlucoseMgDLToMmolL    = 18.0  // mmol/L = mg/dL ÷ 18.0
    CreatinineMgDLToUmolL = 88.4  // µmol/L = mg/dL × 88.4
)

// GlucoseToProduction converts simulation glucose to production units.
// Currently pass-through (both use mmol/L).
func GlucoseToProduction(simValue float64) float64 { return simValue }

// GlucoseToSimulation converts production glucose to simulation units.
func GlucoseToSimulation(prodValue float64) float64 { return prodValue }

// CreatinineToProduction converts simulation creatinine to production units.
// Currently pass-through (both use µmol/L).
func CreatinineToProduction(simValue float64) float64 { return simValue }

// CreatinineToSimulation converts production creatinine to simulation units.
func CreatinineToSimulation(prodValue float64) float64 { return prodValue }
```

- [ ] **Step 4: Run tests to verify pass**

Run: `cd vaidshala/simulation && go test ./bridge/ -v`
Expected: All 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/simulation/bridge/unit_converter.go vaidshala/simulation/bridge/unit_converter_test.go
git commit -m "feat: add unit converter with pass-through and documented constants"
```

---

### Task 4: Create GateSignal mapper

**Files:**
- Create: `vaidshala/simulation/bridge/type_mapper.go`
- Modify: `vaidshala/simulation/bridge/bridge_test.go` (start building it)

- [ ] **Step 1: Write failing test for GateSignal conversion**

Create `vaidshala/simulation/bridge/bridge_test.go`:
```go
package bridge

import (
    "testing"

    simtypes "vaidshala/simulation/pkg/types"
    vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

func TestGateSignalToProduction_AllValues(t *testing.T) {
    cases := []struct {
        sim  simtypes.GateSignal
        prod vt.GateSignal
    }{
        {simtypes.CLEAR, vt.GateClear},
        {simtypes.MODIFY, vt.GateModify},
        {simtypes.PAUSE, vt.GatePause},
        {simtypes.HOLD_DATA, vt.GateHoldData},
        {simtypes.HALT, vt.GateHalt},
    }
    for _, tc := range cases {
        got := GateSignalToProduction(tc.sim)
        if got != tc.prod {
            t.Errorf("GateSignalToProduction(%d) = %q, want %q", tc.sim, got, tc.prod)
        }
    }
}

func TestGateSignalToSimulation_AllValues(t *testing.T) {
    cases := []struct {
        prod vt.GateSignal
        sim  simtypes.GateSignal
    }{
        {vt.GateClear, simtypes.CLEAR},
        {vt.GateModify, simtypes.MODIFY},
        {vt.GatePause, simtypes.PAUSE},
        {vt.GateHoldData, simtypes.HOLD_DATA},
        {vt.GateHalt, simtypes.HALT},
    }
    for _, tc := range cases {
        got := GateSignalToSimulation(tc.prod)
        if got != tc.sim {
            t.Errorf("GateSignalToSimulation(%q) = %d, want %d", tc.prod, got, tc.sim)
        }
    }
}

func TestGateSignalRoundTrip(t *testing.T) {
    for gate := simtypes.CLEAR; gate <= simtypes.HALT; gate++ {
        prod := GateSignalToProduction(gate)
        back := GateSignalToSimulation(prod)
        if back != gate {
            t.Errorf("round-trip failed: %d → %q → %d", gate, prod, back)
        }
    }
}

func TestGateSignalOrdering_Preserved(t *testing.T) {
    // HALT must be the most restrictive in both systems
    if GateSignalToProduction(simtypes.HALT) != vt.GateHalt {
        t.Fatal("HALT ordering not preserved")
    }
    if GateSignalToProduction(simtypes.CLEAR) != vt.GateClear {
        t.Fatal("CLEAR ordering not preserved")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestGateSignal -v`
Expected: FAIL — `GateSignalToProduction` not defined

- [ ] **Step 3: Write implementation**

Create `vaidshala/simulation/bridge/type_mapper.go`:
```go
package bridge

import (
    "fmt"
    "time"

    simtypes "vaidshala/simulation/pkg/types"
    vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
    cb "vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
    cc "vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
)

// gateSignalSimToProd maps simulation int-based GateSignal to production string-based GateSignal.
var gateSignalSimToProd = map[simtypes.GateSignal]vt.GateSignal{
    simtypes.CLEAR:     vt.GateClear,
    simtypes.MODIFY:    vt.GateModify,
    simtypes.PAUSE:     vt.GatePause,
    simtypes.HOLD_DATA: vt.GateHoldData,
    simtypes.HALT:      vt.GateHalt,
}

// gateSignalProdToSim is the reverse map.
var gateSignalProdToSim = map[vt.GateSignal]simtypes.GateSignal{
    vt.GateClear:    simtypes.CLEAR,
    vt.GateModify:   simtypes.MODIFY,
    vt.GatePause:    simtypes.PAUSE,
    vt.GateHoldData: simtypes.HOLD_DATA,
    vt.GateHalt:     simtypes.HALT,
}

// GateSignalToProduction converts simulation GateSignal (int) to production GateSignal (string).
// Panics on unknown values — this is intentional. An unknown gate signal is a safety bug.
func GateSignalToProduction(sim simtypes.GateSignal) vt.GateSignal {
    prod, ok := gateSignalSimToProd[sim]
    if !ok {
        panic(fmt.Sprintf("bridge: unknown simulation GateSignal: %d", sim))
    }
    return prod
}

// GateSignalToSimulation converts production GateSignal (string) to simulation GateSignal (int).
func GateSignalToSimulation(prod vt.GateSignal) simtypes.GateSignal {
    sim, ok := gateSignalProdToSim[prod]
    if !ok {
        panic(fmt.Sprintf("bridge: unknown production GateSignal: %q", prod))
    }
    return sim
}

// float64Ptr converts a float64 to *float64.
func float64Ptr(v float64) *float64 { return &v }

// timePtr converts a time.Time to *time.Time.
func timePtr(v time.Time) *time.Time { return &v }

// derefFloat64 safely dereferences *float64, returning 0 if nil.
func derefFloat64(p *float64) float64 {
    if p == nil {
        return 0
    }
    return *p
}

// intToFloat64Ptr converts int (sim SBP/DBP/HR) to *float64 (production).
func intToFloat64Ptr(v int) *float64 {
    f := float64(v)
    return &f
}
```

Note: The `time` import and helper functions (`float64Ptr`, `timePtr`, `derefFloat64`, `intToFloat64Ptr`) are used by Task 5 (RawPatientData mapper). Include them now to avoid revisiting this file.

- [ ] **Step 4: Run tests to verify pass**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestGateSignal -v`
Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/simulation/bridge/type_mapper.go vaidshala/simulation/bridge/bridge_test.go
git commit -m "feat: add GateSignal bidirectional mapper with exhaustive switch"
```

---

### Task 5: Create RawPatientData and TitrationContext mappers

**Files:**
- Modify: `vaidshala/simulation/bridge/type_mapper.go` — add `ToProductionRawLabs()`, `ToProductionContext()`
- Modify: `vaidshala/simulation/bridge/bridge_test.go` — add round-trip tests per archetype

- [ ] **Step 1: Write failing round-trip test**

Add to `bridge_test.go`:
```go
import (
    "vaidshala/simulation/pkg/patient"
)

func TestRawPatientDataRoundTrip_AllArchetypes(t *testing.T) {
    archetypes := []struct {
        name string
        fn   func() *patient.VirtualPatient
    }{
        {"ActiveHypoglycaemia", patient.ActiveHypoglycaemia},
        {"AKIMidTitration", patient.AKIMidTitration},
        {"RAASCreatinineTolerance", patient.RAASCreatinineTolerance},
        {"DataDropOut", patient.DataDropOut},
        {"NonAdherentPatient", patient.NonAdherentPatient},
        {"JCurveCKD3b", patient.JCurveCKD3b},
        {"DualRAAS", patient.DualRAAS},
        {"HyponatraemiaThiazide", patient.HyponatraemiaThiazide},
        {"GreenTrajectory", patient.GreenTrajectory},
        {"MetforminCKD4", patient.MetforminCKD4},
        {"SeasonalHyponatraemia", patient.SeasonalHyponatraemia},
    }

    for _, tc := range archetypes {
        t.Run(tc.name, func(t *testing.T) {
            vp := tc.fn()
            prodLabs := ToProductionRawLabs(&vp.RawLabs, &vp.Context, vp.Timestamps)
            backLabs := ToSimulationRawLabs(prodLabs)

            // Compare key fields (float64 tolerance for int→float64→int)
            if backLabs.GlucoseCurrent != vp.RawLabs.GlucoseCurrent {
                t.Errorf("GlucoseCurrent: got %v, want %v", backLabs.GlucoseCurrent, vp.RawLabs.GlucoseCurrent)
            }
            if backLabs.SBP != vp.RawLabs.SBP {
                t.Errorf("SBP: got %v, want %v", backLabs.SBP, vp.RawLabs.SBP)
            }
            // ... assert all fields
        })
    }
}
```

Note: The exact field comparisons depend on the simulation's `RawPatientData` struct. Compare every field from `pkg/types/types.go`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestRawPatientDataRoundTrip -v`
Expected: FAIL — `ToProductionRawLabs` not defined

- [ ] **Step 3: Write ToProductionRawLabs implementation**

Add to `vaidshala/simulation/bridge/type_mapper.go` (merge these into the existing import block):
```go
// Add to existing imports:
//    cb "vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
//    cc "vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"

// ToProductionRawLabs converts simulation RawPatientData to production RawPatientData.
// Handles: float64→*float64, int→*float64 (SBP/DBP/HR), time.Time→*time.Time.
// Also transfers fields from TitrationContext that production stores on RawPatientData
// (ThiazideActive, Season, CKDStage, OnRAASAgent, OliguriaReported).
func ToProductionRawLabs(sim *simtypes.RawPatientData, ctx *simtypes.TitrationContext, ts PatientTimestamps) *cb.RawPatientData {
    labs := &cb.RawPatientData{
        GlucoseCurrent:         float64Ptr(GlucoseToProduction(sim.GlucoseCurrent)),
        GlucoseTimestamp:       ts.LastGlucose,
        CreatinineCurrent:      float64Ptr(CreatinineToProduction(sim.CreatinineCurrent)),
        Creatinine48hAgo:       float64Ptr(CreatinineToProduction(sim.CreatininePrevious)),
        PotassiumCurrent:       float64Ptr(sim.PotassiumCurrent),
        SBPCurrent:             intToFloat64Ptr(sim.SBP),
        DBPCurrent:             intToFloat64Ptr(sim.DBP),
        HeartRateCurrent:       intToFloat64Ptr(sim.HeartRate),
        HRRegularity:           sim.HeartRateRegularity,
        WeightKgCurrent:        float64Ptr(sim.Weight),
        Weight72hAgo:           float64Ptr(sim.WeightPrevious),
        EGFRCurrent:            float64Ptr(sim.EGFR),
        HbA1cCurrent:           float64Ptr(sim.HbA1c),
        SodiumCurrent:          float64Ptr(sim.SodiumCurrent),
        BetaBlockerActive:      sim.BetaBlockerActive,
        CreatinineRiseExplained: sim.CreatinineRiseExplained,
        RecentDoseIncrease:     sim.RecentDoseIncrease,
        // Timestamps — eGFR derives from creatinine, not glucose
        CreatinineLastMeasuredAt: timePtr(ts.LastCreatinine),
        EGFRLastMeasuredAt:       timePtr(ts.LastEGFR),
        HbA1cLastMeasuredAt:      timePtr(ts.LastHbA1c),
    }

    // Fields that simulation stores on TitrationContext but production
    // stores on RawPatientData (Channel B reads these directly).
    if ctx != nil {
        labs.ThiazideActive = ctx.ThiazideActive
        labs.Season = ctx.Season
        labs.CKDStage = ctx.CKDStage
        labs.OnRAASAgent = ctx.ACEiActive || ctx.ARBActive
        labs.OliguriaReported = ctx.OliguriaReported
    }

    // Construct minimal GlucoseReadings from current + previous values.
    // Known limitation: entries 1 and 2 share the same value, so B-02
    // declining trend detection will never fire (requires strictly declining).
    // This is conservative (no false positives). No current scenario expects B-02.
    if sim.GlucoseCurrent > 0 {
        labs.GlucoseReadings = []cb.TimestampedValue{
            {Value: sim.GlucoseCurrent, Timestamp: ts.LastGlucose},
        }
        if sim.GlucosePrevious > 0 {
            labs.GlucoseReadings = append(labs.GlucoseReadings,
                cb.TimestampedValue{Value: sim.GlucosePrevious, Timestamp: ts.LastGlucose.Add(-4 * time.Hour)},
                cb.TimestampedValue{Value: sim.GlucosePrevious, Timestamp: ts.LastGlucose.Add(-8 * time.Hour)},
            )
        }
    }

    return labs
}

// ToSimulationRawLabs converts production RawPatientData back to simulation types.
func ToSimulationRawLabs(prod *cb.RawPatientData) simtypes.RawPatientData {
    return simtypes.RawPatientData{
        GlucoseCurrent:         GlucoseToSimulation(derefFloat64(prod.GlucoseCurrent)),
        CreatinineCurrent:      CreatinineToSimulation(derefFloat64(prod.CreatinineCurrent)),
        CreatininePrevious:     CreatinineToSimulation(derefFloat64(prod.Creatinine48hAgo)),
        PotassiumCurrent:       derefFloat64(prod.PotassiumCurrent),
        SBP:                    int(derefFloat64(prod.SBPCurrent)),
        DBP:                    int(derefFloat64(prod.DBPCurrent)),
        HeartRate:              int(derefFloat64(prod.HeartRateCurrent)),
        HeartRateRegularity:    prod.HRRegularity,
        Weight:                 derefFloat64(prod.WeightKgCurrent),
        WeightPrevious:         derefFloat64(prod.Weight72hAgo),
        EGFR:                   derefFloat64(prod.EGFRCurrent),
        HbA1c:                  derefFloat64(prod.HbA1cCurrent),
        SodiumCurrent:          derefFloat64(prod.SodiumCurrent),
        BetaBlockerActive:      prod.BetaBlockerActive,
        CreatinineRiseExplained: prod.CreatinineRiseExplained,
        RecentDoseIncrease:     prod.RecentDoseIncrease,
    }
}

// PatientTimestamps holds timestamps for lab measurements.
// The simulation stores these separately from RawPatientData.
type PatientTimestamps struct {
    LastGlucose    time.Time
    LastCreatinine time.Time
    LastPotassium  time.Time
    LastHbA1c      time.Time // HbA1c is a separate lab from glucose
    LastEGFR       time.Time // eGFR derived from creatinine but measured independently
}

// ToProductionContext converts simulation TitrationContext to production TitrationContext.
func ToProductionContext(sim *simtypes.TitrationContext) *cc.TitrationContext {
    return &cc.TitrationContext{
        EGFR:                   sim.EGFRCurrent,
        ActiveMedications:      medicationListFromBooleans(sim),
        AKIDetected:            false, // Set by Channel B evaluation, not input
        ActiveHypoglycaemia:    false, // Set by Channel B evaluation, not input
        HypoglycaemiaWithin7d:  sim.HypoWithin7d, // PG-07 context
        ProposedAction:         proposedActionFromDelta(sim.ProposedDoseDelta),
        DoseDeltaPercent:       (sim.ProposedDoseDelta / sim.CurrentDose) * 100,
        ACEiARBHyperKDecliningEGFR: sim.DualRAASActive,
        RAASCreatinineTolerant:     sim.RAASChangeWithin14Days,
        ThiazideHyponatraemia:     sim.ThiazideActive,
        PotassiumCurrent:          0, // Filled from Channel B
        SBPCurrent:                0, // Filled from Channel B
        SodiumCurrent:             0, // Filled from Channel B
    }
}

// NOTE: The simulation's TitrationContext needs a `HypoWithin7d bool` field added
// alongside CKDStage and OliguriaReported (see Task 1 Step 3 type extensions).

func proposedActionFromDelta(delta float64) string {
    if delta > 0 { return "dose_increase" }
    if delta < 0 { return "dose_decrease" }
    return "dose_hold"
}

func medicationListFromBooleans(ctx *simtypes.TitrationContext) []string {
    var meds []string
    if ctx.ACEiActive { meds = append(meds, "ACEi") }
    if ctx.ARBActive { meds = append(meds, "ARB") }
    if ctx.SGLT2iActive { meds = append(meds, "SGLT2i") }
    if ctx.InsulinActive { meds = append(meds, "insulin") }
    if ctx.ThiazideActive { meds = append(meds, "thiazide") }
    return meds
}
```

Note: Field names MUST be verified against the actual production struct at `channel_b/raw_inputs.go` and `channel_c/protocol_guard.go`. The code above is based on the spec's type divergence table. During implementation, read both source files and adjust field names to match exactly.

- [ ] **Step 4: Run tests to verify pass**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestRawPatientDataRoundTrip -v`
Expected: All 11 archetype round-trips PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/simulation/bridge/type_mapper.go vaidshala/simulation/bridge/bridge_test.go
git commit -m "feat: add RawPatientData and TitrationContext bidirectional mappers"
```

---

## Chunk 3: Bridge — Result Mapper, Engine Adapter & Arbiter Tests

### Task 6: Create result mapper with RuleIDNormalization

**Files:**
- Create: `vaidshala/simulation/bridge/result_mapper.go`
- Modify: `vaidshala/simulation/bridge/bridge_test.go`

- [ ] **Step 1: Write failing test for RuleIDNormalization**

Add to `bridge_test.go`:
```go
func TestRuleIDNormalization_ExhaustiveMap(t *testing.T) {
    // Every simulation rule ID must have a production mapping
    simRules := []string{
        "B-01", "B-02", "B-03", "B-04", "B-04+PG-14",
        "B-05", "B-06", "B-07", "B-08", "B-09",
        "B-10", "B-11", "B-12", "B-13", "B-14",
        "B-15", "B-16", "B-17", "B-18",
        "PG-01", "PG-02", "PG-03", "PG-04", "PG-05",
        "PG-06", "PG-07", "PG-08", "PG-14",
    }
    for _, ruleID := range simRules {
        prodID := NormalizeRuleID(ruleID, DirectionSimToProduction)
        if prodID == "" {
            t.Errorf("no production mapping for simulation rule %q", ruleID)
        }
    }
}

func TestRuleIDNormalization_Scenario3Divergence(t *testing.T) {
    prodID := NormalizeRuleID("B-04+PG-14", DirectionSimToProduction)
    if prodID != "B-03-RAAS-SUPPRESSED" {
        t.Errorf("Scenario 3 rule ID: got %q, want %q", prodID, "B-03-RAAS-SUPPRESSED")
    }
}

func TestRuleIDNormalization_ProductionOnly(t *testing.T) {
    // Production-only rules should return PRODUCTION_ONLY marker
    prodOnlyRules := []string{"B-10", "B-11", "B-19", "DA-02", "DA-03", "DA-04", "DA-05", "DA-08", "PG-09", "PG-10", "PG-11", "PG-12", "PG-13", "PG-15", "PG-16"}
    for _, ruleID := range prodOnlyRules {
        simID := NormalizeRuleID(ruleID, DirectionProdToSimulation)
        if simID != "PRODUCTION_ONLY" {
            t.Errorf("production-only rule %q: got %q, want PRODUCTION_ONLY", ruleID, simID)
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestRuleIDNormalization -v`
Expected: FAIL — `NormalizeRuleID` not defined

- [ ] **Step 3: Write implementation**

Create `vaidshala/simulation/bridge/result_mapper.go`:
```go
package bridge

import (
    "fmt"

    simtypes "vaidshala/simulation/pkg/types"
    vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
    "vaidshala/clinical-runtime-platform/engines/vmcu/trace"
)

type NormDirection int

const (
    DirectionSimToProduction NormDirection = iota
    DirectionProdToSimulation
)

// ruleIDSimToProd maps simulation rule IDs to production rule IDs.
// Exhaustive — panics on unknown IDs.
var ruleIDSimToProd = map[string]string{
    "B-01":      "B-01",
    "B-02":      "B-07",
    "B-03":      "B-04",
    "B-04":      "B-03",
    "B-04+PG-14": "B-03-RAAS-SUPPRESSED",
    "B-05":      "B-06",
    "B-06":      "DA-01", // Production splits into DA-01..DA-05
    "B-07":      "B-08",
    "B-08":      "B-05",
    "B-09":      "B-09",
    "B-10":      "DA-06",
    "B-11":      "DA-07",
    "B-12":      "B-12",
    "B-13":      "B-13",
    "B-14":      "B-14",
    "B-15":      "B-15",
    "B-16":      "B-16",
    "B-17":      "B-17",
    "B-18":      "B-18",
    "PG-01":     "PG-01",
    "PG-02":     "PG-02",
    "PG-03":     "PG-03",
    "PG-04":     "PG-04",
    "PG-05":     "PG-05",
    "PG-06":     "PG-06",
    "PG-07":     "PG-07",
    "PG-08":     "PG-08",
    "PG-14":     "PG-14",
}

// productionOnlyRules are rules that exist in production but have NO simulation equivalent.
// IMPORTANT: Do NOT add rule IDs here that appear as VALUES in ruleIDSimToProd —
// those have simulation equivalents and must go through the reverse lookup.
// e.g., DA-01 maps FROM sim B-06 → prod DA-01, so DA-01 is NOT production-only.
var productionOnlyRules = map[string]bool{
    // Production B-10 (eGFR slope rapid decline) and B-11 (beta-blocker raised glucose)
    // are DIFFERENT rules from simulation B-10/B-11 (staleness). Sim B-10→DA-06, B-11→DA-07.
    "B-10": true, "B-11": true,
    "B-19": true,
    "DA-02": true, "DA-03": true, "DA-04": true, "DA-05": true, "DA-08": true,
    "PG-09": true, "PG-10": true, "PG-11": true, "PG-12": true, "PG-13": true,
    "PG-15": true, "PG-16": true,
}

// NormalizeRuleID maps a rule ID between simulation and production.
func NormalizeRuleID(ruleID string, direction NormDirection) string {
    if direction == DirectionSimToProduction {
        prod, ok := ruleIDSimToProd[ruleID]
        if !ok {
            panic(fmt.Sprintf("bridge: unknown simulation rule ID: %q", ruleID))
        }
        return prod
    }

    // Production → simulation: reverse lookup
    if productionOnlyRules[ruleID] {
        return "PRODUCTION_ONLY"
    }
    for simID, prodID := range ruleIDSimToProd {
        if prodID == ruleID {
            return simID
        }
    }
    // Unknown production rule that isn't in production-only list
    panic(fmt.Sprintf("bridge: unknown production rule ID: %q — add to ruleIDSimToProd or productionOnlyRules", ruleID))
}

// Note: derefFloat64() is defined in type_mapper.go (same package).

// ToSimulationResult converts production TitrationCycleResult + SafetyTrace
// to simulation TitrationCycleResult.
// Production uses *float64 for DoseApplied/DoseDelta; simulation uses bool/float64.
func ToSimulationResult(prod *vt.TitrationCycleResult, safetyTrace *trace.SafetyTrace) simtypes.TitrationCycleResult {
    // Production DoseApplied is *float64 (the NEW dose value, nil = not applied).
    // Simulation DoseApplied is bool. Convert: non-nil → true.
    // Note: production sets DoseApplied to nil when blocked, and to &NewDose when applied.
    // The value can legitimately be 0.0 during deprescribing (dose reduced to zero).
    doseApplied := prod.DoseApplied != nil
    doseDelta := derefFloat64(prod.DoseDelta)

    return simtypes.TitrationCycleResult{
        FinalGate:         GateSignalToSimulation(prod.Arbiter.FinalGate),
        DominantChannel:   simtypes.Channel(prod.Arbiter.DominantChannel),
        DoseApplied:       doseApplied,
        DoseDelta:         doseDelta,
        BlockedBy:         prod.BlockedBy,
        PhysioRuleFired:   prod.ChannelB.RuleFired,
        ProtocolRuleFired: prod.ChannelC.RuleID,
        SafetyTrace: simtypes.SafetyTrace{
            MCUGate:      GateSignalToSimulation(prod.ChannelA.Gate),
            PhysioGate:   GateSignalToSimulation(prod.ChannelB.Gate),
            ProtocolGate: GateSignalToSimulation(prod.ChannelC.Gate),
            FinalGate:    GateSignalToSimulation(prod.Arbiter.FinalGate),
            DoseApplied:  doseApplied,
            DoseDelta:    doseDelta,
        },
    }
}
```

Note: Field names in `TitrationCycleResult`, `ChannelBResult`, `ChannelCResult` must be verified against production's `types/gate_signal.go`. Adjust field access paths during implementation.

- [ ] **Step 4: Run tests to verify pass**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestRuleIDNormalization -v`
Expected: All 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/simulation/bridge/result_mapper.go vaidshala/simulation/bridge/bridge_test.go
git commit -m "feat: add result mapper with exhaustive RuleIDNormalization table"
```

---

### Task 7: Create engine adapter

**Files:**
- Create: `vaidshala/simulation/bridge/engine_adapter.go`
- Copy: `vaidshala/simulation/bridge/testdata/protocol_rules.yaml`
- Modify: `vaidshala/simulation/bridge/bridge_test.go`

- [ ] **Step 1: Copy protocol_rules.yaml to testdata**

```bash
cp vaidshala/clinical-runtime-platform/engines/vmcu/protocol_rules.yaml \
   vaidshala/simulation/bridge/testdata/protocol_rules.yaml
```

- [ ] **Step 2: Write failing test for ProductionEngine construction**

Add to `bridge_test.go`:
```go
func TestNewProductionEngine_Constructs(t *testing.T) {
    engine, err := NewProductionEngine(
        WithProtocolRulesPath("testdata/protocol_rules.yaml"),
    )
    if err != nil {
        t.Fatalf("NewProductionEngine failed: %v", err)
    }
    if engine == nil {
        t.Fatal("engine is nil")
    }
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestNewProductionEngine -v`
Expected: FAIL — `NewProductionEngine` not defined

- [ ] **Step 4: Write implementation**

Create `vaidshala/simulation/bridge/engine_adapter.go`:
```go
package bridge

import (
    "fmt"
    "time"

    simtypes "vaidshala/simulation/pkg/types"
    vmcu "vaidshala/clinical-runtime-platform/engines/vmcu"
    vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
    "vaidshala/clinical-runtime-platform/engines/vmcu/trace"
)

// ProductionEngine wraps the production VMCUEngine for use by simulation scenarios.
type ProductionEngine struct {
    engine  *vmcu.VMCUEngine
    simTime time.Time
}

// EngineOption configures the ProductionEngine.
type EngineOption func(*engineConfig)

type engineConfig struct {
    protocolRulesPath string
    simTime           time.Time
}

// WithProtocolRulesPath sets the path to protocol_rules.yaml.
func WithProtocolRulesPath(path string) EngineOption {
    return func(c *engineConfig) { c.protocolRulesPath = path }
}

// WithSimulatedTime sets a fixed time for timestamp construction.
func WithSimulatedTime(t time.Time) EngineOption {
    return func(c *engineConfig) { c.simTime = t }
}

// NewProductionEngine creates a production VMCUEngine with zero infrastructure dependencies.
func NewProductionEngine(opts ...EngineOption) (*ProductionEngine, error) {
    cfg := &engineConfig{
        protocolRulesPath: "testdata/protocol_rules.yaml",
        simTime:           time.Now(),
    }
    for _, opt := range opts {
        opt(cfg)
    }

    vmcuCfg := vmcu.DefaultVMCUConfig()
    vmcuCfg.ProtocolRulesPath = cfg.protocolRulesPath
    // Zero infrastructure — nil for all optional dependencies
    engine, err := vmcu.NewVMCUEngine(vmcuCfg)
    if err != nil {
        return nil, fmt.Errorf("bridge: failed to create production engine: %w", err)
    }

    return &ProductionEngine{
        engine:  engine,
        simTime: cfg.simTime,
    }, nil
}

// RunCycle converts simulation input to production input, calls production RunCycle(),
// and converts the result back to simulation types.
func (pe *ProductionEngine) RunCycle(input simtypes.TitrationCycleInput) simtypes.TitrationCycleResult {
    ts := PatientTimestamps{
        LastGlucose:    pe.simTime,
        LastCreatinine: pe.simTime,
        LastPotassium:  pe.simTime,
        LastHbA1c:      pe.simTime,
        LastEGFR:       pe.simTime,
    }

    prodInput := vmcu.TitrationCycleInput{
        PatientID:      input.PatientID,
        ChannelAResult: vt.ChannelAResult{
            Gate:       GateSignalToProduction(input.MCUGate),
            GainFactor: 1.0,
        },
        RawLabs:          ToProductionRawLabs(input.RawLabs, input.TitrationContext, ts),
        TitrationContext: ToProductionContext(input.TitrationContext),
        CurrentDose:      input.TitrationContext.CurrentDose,
        ProposedDelta:    input.TitrationContext.ProposedDoseDelta,
        // MedClass and MetabolicInput are optional — zero-value defaults are safe
    }

    // Production RunCycle returns TWO values: (*TitrationCycleResult, *SafetyTrace)
    prodResult, safetyTrace := pe.engine.RunCycle(prodInput)
    return ToSimulationResult(prodResult, safetyTrace)
}
```

Note: The exact `VMCUEngine` constructor signature (`NewVMCUEngine`, `DefaultVMCUConfig`) must be verified against `vmcu_engine.go`. Adapt the construction pattern to match the production API.

- [ ] **Step 5: Run tests to verify pass**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestNewProductionEngine -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add vaidshala/simulation/bridge/engine_adapter.go vaidshala/simulation/bridge/testdata/
git commit -m "feat: add ProductionEngine adapter with protocol_rules.yaml embedding"
```

---

### Task 8: Add arbiter compatibility test (125 combinations)

**Files:**
- Modify: `vaidshala/simulation/bridge/bridge_test.go`

- [ ] **Step 1: Write arbiter exhaustive test**

Add to `bridge_test.go`:
```go
func TestArbiterCompatibility_125Combinations(t *testing.T) {
    gates := []simtypes.GateSignal{
        simtypes.CLEAR, simtypes.MODIFY, simtypes.PAUSE,
        simtypes.HOLD_DATA, simtypes.HALT,
    }

    passed := 0
    for _, a := range gates {
        for _, b := range gates {
            for _, c := range gates {
                // Simulation arbiter
                simInput := simtypes.ArbiterInput{
                    MCUGate:      a,
                    PhysioGate:   b,
                    ProtocolGate: c,
                }
                simResult := simtypes.Arbitrate(simInput)

                // Production arbiter (via converted types)
                prodInput := vt.ArbiterInput{
                    MCUGate:      GateSignalToProduction(a),
                    PhysioGate:   GateSignalToProduction(b),
                    ProtocolGate: GateSignalToProduction(c),
                }
                // Import and call production arbiter
                prodResult := arbiter.Arbitrate(prodInput)

                // Compare results
                expectedGate := GateSignalToProduction(simResult.FinalGate)
                if prodResult.FinalGate != expectedGate {
                    t.Errorf("Arbiter(%d,%d,%d): sim=%d→%q, prod=%q",
                        a, b, c, simResult.FinalGate, expectedGate, prodResult.FinalGate)
                } else {
                    passed++
                }
            }
        }
    }
    if passed != 125 {
        t.Fatalf("Arbiter: %d/125 passed", passed)
    }
    t.Logf("Arbiter: 125/125 combinations verified")
}
```

- [ ] **Step 2: Run test**

Run: `cd vaidshala/simulation && go test ./bridge/ -run TestArbiterCompatibility -v`
Expected: PASS — 125/125 combinations verified

- [ ] **Step 3: Commit**

```bash
git add vaidshala/simulation/bridge/bridge_test.go
git commit -m "test: verify arbiter compatibility across 125 gate combinations"
```

---

## Chunk 4: Production Scenario Tests & Scenario Registry

### Task 9: Create scenario registry

**Files:**
- Create: `vaidshala/simulation/pkg/scenarios/registry.go`

- [ ] **Step 1: Write the registry**

Create `vaidshala/simulation/pkg/scenarios/registry.go`:
```go
package scenarios

import (
    "vaidshala/simulation/pkg/types"
    "vaidshala/simulation/pkg/patient"
)

// Scenario defines a single safety test case.
type Scenario struct {
    ID        int
    Name      string
    Archetype func() *patient.VirtualPatient
    Expected  ExpectedOutcome
    Tags      []string // Rule IDs this scenario tests
    ProdOnly  bool     // true = only runs in production_test.go
}

// ExpectedOutcome defines what the scenario must produce.
type ExpectedOutcome struct {
    Gate          types.GateSignal
    DoseApplied   bool
    PhysioRule    string
    ProtocolRule  string
}

// AllScenarios returns the complete scenario registry.
func AllScenarios() []Scenario {
    return []Scenario{
        {1, "Active Hypoglycaemia", patient.ActiveHypoglycaemia, ExpectedOutcome{types.HALT, false, "B-01", "PG-04"}, []string{"B-01", "PG-04"}, false},
        {2, "AKI Mid-Titration", patient.AKIMidTitration, ExpectedOutcome{types.HALT, false, "B-04", "PG-03"}, []string{"B-04", "PG-03"}, false},
        {3, "RAAS Creatinine Tolerance", patient.RAASCreatinineTolerance, ExpectedOutcome{types.PAUSE, false, "B-04+PG-14", ""}, []string{"B-04", "PG-14"}, false},
        {4, "Data Drop-Out", patient.DataDropOut, ExpectedOutcome{types.HOLD_DATA, false, "B-10", ""}, []string{"B-10", "DA-06"}, false},
        {5, "Non-Adherent Patient", patient.NonAdherentPatient, ExpectedOutcome{types.MODIFY, false, "", ""}, []string{"MODIFY"}, false},
        {6, "J-Curve CKD3b", patient.JCurveCKD3b, ExpectedOutcome{types.PAUSE, false, "B-12", ""}, []string{"B-12"}, false},
        {7, "Dual RAAS", patient.DualRAAS, ExpectedOutcome{types.HALT, false, "", "PG-08"}, []string{"PG-08"}, false},
        {8, "Hyponatraemia + Thiazide", patient.HyponatraemiaThiazide, ExpectedOutcome{types.HALT, false, "B-17", ""}, []string{"B-17"}, false},
        {9, "GREEN Trajectory", patient.GreenTrajectory, ExpectedOutcome{types.CLEAR, true, "", ""}, []string{"CLEAR"}, false},
        {10, "Metformin CKD4", patient.MetforminCKD4, ExpectedOutcome{types.HALT, false, "", "PG-01"}, []string{"PG-01"}, false},
        // Scenarios 11 and 12 are special — handled separately in test files
        {13, "Seasonal Hyponatraemia", patient.SeasonalHyponatraemia, ExpectedOutcome{types.PAUSE, false, "B-19", ""}, []string{"B-19"}, true},
    }
}
```

Note: Scenarios 11 (IntegratorResume) and 12 (ArbiterSweep) are multi-step and don't fit the single-archetype pattern. They remain as standalone test functions in `simulation_test.go` and `production_test.go`.

- [ ] **Step 2: Verify it compiles**

Run: `cd vaidshala/simulation && go build ./pkg/scenarios/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add vaidshala/simulation/pkg/scenarios/registry.go
git commit -m "feat: add scenario registry with tags and production-only flag"
```

---

### Task 10: Create production_test.go

**Files:**
- Create: `vaidshala/simulation/pkg/scenarios/production_test.go`

- [ ] **Step 1: Write production scenario tests**

Create `vaidshala/simulation/pkg/scenarios/production_test.go`:
```go
package scenarios

import (
    "math"
    "testing"
    "time"

    "vaidshala/simulation/bridge"
    "vaidshala/simulation/pkg/patient"
    "vaidshala/simulation/pkg/types"
)

func newProdEngine(t *testing.T) *bridge.ProductionEngine {
    t.Helper()
    engine, err := bridge.NewProductionEngine(
        bridge.WithProtocolRulesPath("../../bridge/testdata/protocol_rules.yaml"),
    )
    if err != nil {
        t.Fatalf("failed to create production engine: %v", err)
    }
    return engine
}

func TestProductionScenarios(t *testing.T) {
    engine := newProdEngine(t)
    scenarios := AllScenarios()

    for _, sc := range scenarios {
        t.Run(sc.Name, func(t *testing.T) {
            vp := sc.Archetype()
            input := vp.ToTitrationInput(sc.ID)
            result := engine.RunCycle(input)

            // Gate assertion
            if result.FinalGate != sc.Expected.Gate {
                t.Errorf("gate: got %v, want %v", result.FinalGate, sc.Expected.Gate)
            }

            // Dose assertion
            if result.DoseApplied != sc.Expected.DoseApplied {
                t.Errorf("doseApplied: got %v, want %v", result.DoseApplied, sc.Expected.DoseApplied)
            }

            // Rule attribution (for Scenario 3, use production rule ID)
            expectedPhysio := sc.Expected.PhysioRule
            if sc.ID == 3 {
                expectedPhysio = "B-03-RAAS-SUPPRESSED"
            }
            if expectedPhysio != "" && result.PhysioRuleFired != expectedPhysio {
                t.Errorf("physioRule: got %q, want %q", result.PhysioRuleFired, expectedPhysio)
            }

            // Safety invariants
            validateInvariants(t, result)
        })
    }
}

func TestProductionScenario11_IntegratorResume(t *testing.T) {
    engine := newProdEngine(t)

    // Step 1: Trigger HALT with active hypoglycaemia
    hypo := patient.ActiveHypoglycaemia()
    input1 := hypo.ToTitrationInput(1)
    r1 := engine.RunCycle(input1)

    if r1.FinalGate != types.HALT {
        t.Fatalf("Setup: expected HALT from hypoglycaemia, got %v", r1.FinalGate)
    }
    if r1.DoseApplied {
        t.Fatal("Setup: dose should not be applied during HALT")
    }

    // Step 2: Verify HALT invariants
    validateInvariants(t, r1)

    // Step 3: Resume with GREEN trajectory (normal labs).
    //
    // TIME SIMULATION DESIGN NOTE:
    // The production VMCUEngine uses time.Now() internally for integrator
    // freeze/resume calculations (vmcu_engine.go). The bridge's WithSimulatedTime
    // only affects lab timestamps, NOT the engine's internal clock.
    //
    // Three approaches to test post-resume dampening:
    //   (a) If production exposes SetClockFunc() or uses a clock interface → inject mock
    //   (b) If production exposes GetIntegratorState() → manipulate FrozenSince directly
    //   (c) Accept that this test verifies freeze/resume LOGIC (not 120h duration),
    //       and rely on simulation_test.go TestScenario11 for the full dampening proof
    //
    // During implementation, check vmcu_engine.go for a clock injection point.
    // If none exists, use approach (c): verify that HALT→CLEAR transition works
    // and that the integrator correctly unfreezes, without asserting specific
    // dampening values. The standalone simulation's TestScenario11 already
    // proves the dampening math with direct integrator manipulation.

    green := patient.GreenTrajectory()
    input2 := green.ToTitrationInput(2)
    r2 := engine.RunCycle(input2)

    // Verify resume: should not remain in HALT with normal labs
    if r2.FinalGate == types.HALT {
        t.Error("Should have resumed from HALT with normal labs")
    }

    // Verify basic invariants on the resumed cycle
    validateInvariants(t, r2)

    // If the production engine supports clock injection (found during implementation),
    // add this enhanced assertion block:
    //
    // engine.SetFrozenSince(time.Now().Add(-120 * time.Hour))
    // r3 := engine.RunCycle(input2)
    // if r3.DoseApplied {
    //     normalDelta := 2.0
    //     if r3.DoseDelta > normalDelta*0.50+0.5 {
    //         t.Errorf("Post-resume delta %.2f exceeds 50%% dampening", r3.DoseDelta)
    //     }
    //     t.Logf("Post-resume dose delta: %.2f (rate-limited to 50%%)", r3.DoseDelta)
    // }

    t.Log("Freeze/resume transition verified. Full dampening math covered by simulation_test.go TestScenario11")
}

func TestProductionScenario12_ArbiterSweep(t *testing.T) {
    // Already covered by bridge_test.go TestArbiterCompatibility_125Combinations
    // This is a cross-reference — the bridge test is the authoritative arbiter test
    t.Log("Arbiter sweep covered by bridge_test.go TestArbiterCompatibility_125Combinations")
}

// validateInvariants checks all 8 safety invariants from spec Section 6.
// Invariants 5-7 require additional context (lastApprovedDose, physicianDose,
// isPostResume) passed via the optional params struct.
type InvariantContext struct {
    LastApprovedDose float64 // for invariant 5: single-step ≤ ±20%
    PhysicianDose    float64 // for invariant 6: cumulative drift ≤ ±50%
    IsPostResume     bool    // for invariant 7: first post-resume ≤ 50%
    NormalDelta      float64 // for invariant 7: what the delta would be without dampening
    HasSafetyTrace   bool    // for invariant 8: every RunCycle produces a trace
}

func validateInvariants(t *testing.T, result types.TitrationCycleResult) {
    validateInvariantsWithContext(t, result, nil)
}

func validateInvariantsWithContext(t *testing.T, result types.TitrationCycleResult, ctx *InvariantContext) {
    t.Helper()

    // Invariant 1: HALT → DoseApplied=false, DoseDelta=0
    if result.FinalGate == types.HALT {
        if result.DoseApplied {
            t.Error("INVARIANT 1 VIOLATION: DoseApplied=true during HALT")
        }
        if result.DoseDelta != 0 {
            t.Errorf("INVARIANT 1 VIOLATION: DoseDelta=%v during HALT, want 0", result.DoseDelta)
        }
    }

    // Invariant 2: FinalGate >= max(all channels)
    maxChannel := types.MostRestrictive(
        result.SafetyTrace.MCUGate,
        types.MostRestrictive(result.SafetyTrace.PhysioGate, result.SafetyTrace.ProtocolGate),
    )
    if result.FinalGate < maxChannel {
        t.Errorf("INVARIANT 2 VIOLATION: FinalGate=%v < max(channels)=%v", result.FinalGate, maxChannel)
    }

    // Invariant 3: Any channel HALT → final HALT
    if result.SafetyTrace.MCUGate == types.HALT ||
        result.SafetyTrace.PhysioGate == types.HALT ||
        result.SafetyTrace.ProtocolGate == types.HALT {
        if result.FinalGate != types.HALT {
            t.Errorf("INVARIANT 3 VIOLATION: channel HALT but FinalGate=%v", result.FinalGate)
        }
    }

    // Invariant 4: All channels CLEAR → final CLEAR
    if result.SafetyTrace.MCUGate == types.CLEAR &&
        result.SafetyTrace.PhysioGate == types.CLEAR &&
        result.SafetyTrace.ProtocolGate == types.CLEAR {
        if result.FinalGate != types.CLEAR {
            t.Errorf("INVARIANT 4 VIOLATION: all CLEAR but FinalGate=%v", result.FinalGate)
        }
    }

    // Invariants 5-8 require context — skip if not provided
    if ctx == nil {
        return
    }

    // Invariant 5: Single-step delta ≤ ±20% of last approved dose
    if ctx.LastApprovedDose > 0 && result.DoseApplied {
        maxDelta := ctx.LastApprovedDose * 0.20
        if math.Abs(result.DoseDelta) > maxDelta+0.001 { // tolerance for float
            t.Errorf("INVARIANT 5 VIOLATION: |DoseDelta|=%.3f > 20%% of lastApproved=%.3f (max=%.3f)",
                math.Abs(result.DoseDelta), ctx.LastApprovedDose, maxDelta)
        }
    }

    // Invariant 6: Cumulative drift ≤ ±50% from physician-approved dose
    if ctx.PhysicianDose > 0 && result.DoseApplied {
        currentDose := ctx.LastApprovedDose + result.DoseDelta
        maxDrift := ctx.PhysicianDose * 0.50
        drift := math.Abs(currentDose - ctx.PhysicianDose)
        if drift > maxDrift+0.001 {
            t.Errorf("INVARIANT 6 VIOLATION: cumulative drift=%.3f > 50%% of physician dose=%.3f",
                drift, ctx.PhysicianDose)
        }
    }

    // Invariant 7: Post-resume first dose ≤ 50% of normal delta
    if ctx.IsPostResume && ctx.NormalDelta > 0 && result.DoseApplied {
        if math.Abs(result.DoseDelta) > math.Abs(ctx.NormalDelta)*0.50+0.001 {
            t.Errorf("INVARIANT 7 VIOLATION: post-resume |delta|=%.3f > 50%% of normal=%.3f",
                math.Abs(result.DoseDelta), math.Abs(ctx.NormalDelta)*0.50)
        }
    }

    // Invariant 8: Every RunCycle produces exactly one SafetyTrace
    if ctx.HasSafetyTrace {
        // SafetyTrace is embedded in result — verify it has non-default values
        zeroTrace := types.SafetyTrace{}
        if result.SafetyTrace == zeroTrace {
            t.Error("INVARIANT 8 VIOLATION: SafetyTrace is empty/zero — RunCycle must always produce a trace")
        }
    }
}
```

- [ ] **Step 2: Run production tests**

Run: `cd vaidshala/simulation && go test ./pkg/scenarios/ -run TestProduction -v -count=1`
Expected: All scenarios PASS. If any fail, investigate using the discrepancy investigation order (bridge bug → engine bug → test bug).

- [ ] **Step 3: Commit**

```bash
git add vaidshala/simulation/pkg/scenarios/production_test.go
git commit -m "feat: add production scenario tests (13 scenarios via bridge adapter)"
```

---

## Chunk 5: Physiology Engine Migration & Config

### Task 11: Create population config loader

**Files:**
- Create: `vaidshala/simulation/config/default.yaml`
- Create: `vaidshala/simulation/config/south_asian.yaml`
- Create: `vaidshala/simulation/pkg/physiology/config.go`
- Create: `vaidshala/simulation/pkg/physiology/config_test.go`

- [ ] **Step 1: Write config YAML files**

Create `vaidshala/simulation/config/default.yaml` with the full content from spec Section 5 (lines 274-323).

Create `vaidshala/simulation/config/south_asian.yaml` with the override content from spec (lines 328-337).

- [ ] **Step 2: Write failing config loader test**

Create `vaidshala/simulation/pkg/physiology/config_test.go`:
```go
package physiology

import "testing"

func TestLoadPopulationConfig_Default(t *testing.T) {
    cfg, err := LoadPopulationConfig("../../config/default.yaml")
    if err != nil {
        t.Fatalf("failed to load default config: %v", err)
    }
    if cfg.Glucose.EquilibriumDriftRate != 0.10 {
        t.Errorf("glucose drift rate: got %v, want 0.10", cfg.Glucose.EquilibriumDriftRate)
    }
    if cfg.Simulation.RandomSeed != 42 {
        t.Errorf("random seed: got %v, want 42", cfg.Simulation.RandomSeed)
    }
}

func TestLoadPopulationConfig_SouthAsian(t *testing.T) {
    cfg, err := LoadPopulationConfig("../../config/default.yaml", "../../config/south_asian.yaml")
    if err != nil {
        t.Fatalf("failed to load south_asian config: %v", err)
    }
    if cfg.BodyComposition.VisceralFatInsulinThreshold != 1.2 {
        t.Errorf("VFI threshold: got %v, want 1.2", cfg.BodyComposition.VisceralFatInsulinThreshold)
    }
    if cfg.Glucose.CarbBaselineG != 350 {
        t.Errorf("carb baseline: got %v, want 350", cfg.Glucose.CarbBaselineG)
    }
    // Unmodified fields should retain defaults
    if cfg.Glucose.EquilibriumDriftRate != 0.10 {
        t.Errorf("drift rate should be default 0.10, got %v", cfg.Glucose.EquilibriumDriftRate)
    }
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd vaidshala/simulation && go test ./pkg/physiology/ -run TestLoadPopulationConfig -v`
Expected: FAIL — `LoadPopulationConfig` not defined

- [ ] **Step 4: Write implementation**

Create `vaidshala/simulation/pkg/physiology/config.go`:
```go
package physiology

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

// PopulationConfig holds all tunable physiology coefficients.
type PopulationConfig struct {
    Population string `yaml:"population"`
    Version    string `yaml:"version"`
    Extends    string `yaml:"extends,omitempty"`

    BodyComposition BodyCompositionConfig `yaml:"body_composition"`
    Glucose         GlucoseConfig         `yaml:"glucose"`
    Hemodynamic     HemodynamicConfig     `yaml:"hemodynamic"`
    Renal           RenalConfig           `yaml:"renal"`
    ObservationNoise ObservationNoiseConfig `yaml:"observation_noise"`
    Simulation      SimulationConfig      `yaml:"simulation"`
    Autonomy        AutonomyConfig        `yaml:"autonomy"`
}

type BodyCompositionConfig struct {
    VisceralFatInsulinThreshold float64 `yaml:"visceral_fat_insulin_threshold"`
    MuscleSensitivityWeight     float64 `yaml:"muscle_sensitivity_weight"`
    SGLT2iCalorieLossKcal       float64 `yaml:"sglt2i_calorie_loss_kcal"`
    GLP1RAAppetiteReductionPct  float64 `yaml:"glp1ra_appetite_reduction_pct"`
}

type GlucoseConfig struct {
    EquilibriumDriftRate      float64 `yaml:"equilibrium_drift_rate"`
    BetaCellDeclineRate       float64 `yaml:"beta_cell_decline_rate"`
    GlucotoxicityThresholdMmol float64 `yaml:"glucotoxicity_threshold_mmol"`
    GlucotoxicityMultiplier   float64 `yaml:"glucotoxicity_multiplier"`
    CarbBaselineG             float64 `yaml:"carb_baseline_g"`
    PPBGSpikeCoefficient      float64 `yaml:"ppbg_spike_coefficient"`
}

type HemodynamicConfig struct {
    SBPDriftRate          float64 `yaml:"sbp_drift_rate"`
    ACEiARBEffectMmHg     float64 `yaml:"acei_arb_effect_mmhg"`
    ThiazideEffectMmHg    float64 `yaml:"thiazide_effect_mmhg"`
    CCBEffectMmHg         float64 `yaml:"ccb_effect_mmhg"`
    BetaBlockerEffectMmHg float64 `yaml:"beta_blocker_effect_mmhg"`
    SGLT2iBPEffectMmHg    float64 `yaml:"sglt2i_bp_effect_mmhg"`
}

type RenalConfig struct {
    NaturalEGFRDeclinePerYear  float64 `yaml:"natural_egfr_decline_per_year"`
    ACEiARBProtectionPct       float64 `yaml:"acei_arb_protection_pct"`
    SGLT2iProtectionPct        float64 `yaml:"sglt2i_protection_pct"`
    GLP1RAProtectionPct        float64 `yaml:"glp1ra_protection_pct"`
    UncontrolledSBPThreshold   float64 `yaml:"uncontrolled_sbp_threshold"`
    HighGlucoseThresholdMmol   float64 `yaml:"high_glucose_threshold_mmol"`
}

type ObservationNoiseConfig struct {
    GlucoseStddevMmol    float64 `yaml:"glucose_stddev_mmol"`
    BPStddevMmHg         float64 `yaml:"bp_stddev_mmhg"`
    PotassiumStddevMmol  float64 `yaml:"potassium_stddev_mmol"`
    CreatinineStddevUmol float64 `yaml:"creatinine_stddev_umol"`
    WeightStddevKg       float64 `yaml:"weight_stddev_kg"`
}

type SimulationConfig struct {
    RandomSeed   int64   `yaml:"random_seed"`
    TotalDays    int     `yaml:"total_days"`
    CyclesPerDay int     `yaml:"cycles_per_day"`
}

type AutonomyConfig struct {
    SingleStepPct  float64 `yaml:"single_step_pct"`
    CumulativePct  float64 `yaml:"cumulative_pct"`
}

// LoadPopulationConfig loads one or more YAML files, merging overrides onto defaults.
// First file is the base, subsequent files override non-zero values.
func LoadPopulationConfig(paths ...string) (*PopulationConfig, error) {
    if len(paths) == 0 {
        return nil, fmt.Errorf("at least one config path required")
    }

    var cfg PopulationConfig

    for _, path := range paths {
        data, err := os.ReadFile(path)
        if err != nil {
            return nil, fmt.Errorf("reading %s: %w", path, err)
        }
        if err := yaml.Unmarshal(data, &cfg); err != nil {
            return nil, fmt.Errorf("parsing %s: %w", path, err)
        }
    }

    return &cfg, nil
}
```

Note: `yaml.Unmarshal` on an existing struct merges non-zero values, which gives us the `extends` behavior for free.

- [ ] **Step 5: Run tests to verify pass**

Run: `cd vaidshala/simulation && go test ./pkg/physiology/ -run TestLoadPopulationConfig -v`
Expected: Both tests PASS

- [ ] **Step 6: Commit**

```bash
git add vaidshala/simulation/config/ vaidshala/simulation/pkg/physiology/config.go vaidshala/simulation/pkg/physiology/config_test.go
git commit -m "feat: add population config loader with YAML merge support"
```

---

### Task 12: Migrate v2 physiology engines

**Files:**
- Copy + rename: 8 files from v2 tarball to `vaidshala/simulation/pkg/physiology/`

- [ ] **Step 1: Copy v2 engine files**

Copy the following from the v2 tarball (extract `vaidshala-simulation-module-v2.tar.gz` first if not already extracted):
- `body_engine.go` → `pkg/physiology/body_composition.go`
- `glucose_engine.go` → `pkg/physiology/glucose.go`
- `hemodynamic_engine.go` → `pkg/physiology/hemodynamic.go`
- `renal_engine.go` → `pkg/physiology/renal.go`
- `simulator.go` → `pkg/physiology/observation.go` (extract observation generator portion)
- `state.go` → `pkg/physiology/state.go`
- `archetypes.go` → `pkg/physiology/archetypes.go`
- `trajectory_test.go` → `pkg/physiology/trajectory_test.go`

In each file, update import paths from the v2 module path to `vaidshala/simulation`.

- [ ] **Step 2: Extract hardcoded coefficients into config**

For each engine, replace hardcoded constants with values from `PopulationConfig`. Example for glucose engine:

Before: `const equilibriumDriftRate = 0.10`
After: `engine.cfg.Glucose.EquilibriumDriftRate`

Each engine constructor should accept `*PopulationConfig` as a parameter.

- [ ] **Step 3: Verify physiology tests pass**

Run: `cd vaidshala/simulation && go test ./pkg/physiology/ -v -count=1`
Expected: All 7 trajectory tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/simulation/pkg/physiology/
git commit -m "feat: migrate v2 physiology engines with configurable coefficients"
```

---

## Chunk 6: CLI Update & CI Gate

### Task 13: Update CLI with --production and --trajectory flags

**Files:**
- Modify: `vaidshala/simulation/cmd/main.go`

- [ ] **Step 1: Add flag parsing and production mode**

Update `cmd/main.go` to add:
- `--production` flag: runs scenarios via `bridge.ProductionEngine` instead of `harness.VMCUEngine`
- `--trajectory` flag: additionally runs 4 trajectory archetypes for 90 days each
- Default mode: runs 12 scenarios (1-12) against simulation engine
- Production mode: runs 13 scenarios (1-13 including Scenario 13) against production engine

Use `flag` package for parsing. Output format:
```
┌────┬──────────────────────────────┬───────────┬──────┬────────┬────────┬────────┐
│ #  │ SCENARIO                     │ GATE      │ DOSE │ DELTA  │ B-RULE │ C-RULE │
├────┼──────────────────────────────┼───────────┼──────┼────────┼────────┼────────┤
│  1 │ Active Hypoglycaemia         │ HALT      │ NO   │  0.0   │ B-01   │ PG-04  │
│ ...│ ...                          │ ...       │ ...  │  ...   │ ...    │ ...    │
└────┴──────────────────────────────┴───────────┴──────┴────────┴────────┴────────┘
```

- [ ] **Step 2: Test both modes**

Run: `cd vaidshala/simulation && go run cmd/main.go`
Expected: 12/12 PASS, 125/125 arbiter

Run: `cd vaidshala/simulation && go run cmd/main.go --production`
Expected: 13/13 PASS, 125/125 arbiter

Run: `cd vaidshala/simulation && go run cmd/main.go --production --trajectory`
Expected: 13/13 scenarios + 4 trajectory archetypes + untreated control, all PASS

- [ ] **Step 3: Commit**

```bash
git add vaidshala/simulation/cmd/main.go
git commit -m "feat: add --production and --trajectory CLI flags"
```

---

### Task 14: Create GitHub Actions CI workflow

**Files:**
- Create: `.github/workflows/simulation-gate.yml` (GitHub Actions requires workflows at repo root)

- [ ] **Step 1: Write the workflow**

Create `.github/workflows/simulation-gate.yml`:
```yaml
name: Simulation Safety Gate

on:
  pull_request:
    paths:
      - 'vaidshala/clinical-runtime-platform/engines/vmcu/**'
      - 'vaidshala/simulation/**'
  push:
    branches: [main]
    paths:
      - 'vaidshala/clinical-runtime-platform/engines/vmcu/**'
      - 'vaidshala/simulation/**'

jobs:
  simulation-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Verify protocol_rules.yaml is current
        run: |
          cd vaidshala/simulation
          PROD_SHA=$(sha256sum ../clinical-runtime-platform/engines/vmcu/protocol_rules.yaml | cut -d' ' -f1)
          TEST_SHA=$(sha256sum bridge/testdata/protocol_rules.yaml | cut -d' ' -f1)
          if [ "$PROD_SHA" != "$TEST_SHA" ]; then
            echo "ERROR: bridge/testdata/protocol_rules.yaml is stale"
            echo "Production: $PROD_SHA"
            echo "Testdata:   $TEST_SHA"
            exit 1
          fi

      - name: Bridge compile check
        run: cd vaidshala/simulation && go build ./bridge/...

      - name: Bridge round-trip tests
        run: cd vaidshala/simulation && go test ./bridge/ -v -count=1

      - name: Layer 1 — Simulation scenarios
        run: cd vaidshala/simulation && go test ./pkg/scenarios/ -run TestSimulation -v -count=1

      - name: Layer 1 — Production scenarios
        run: cd vaidshala/simulation && go test ./pkg/scenarios/ -run TestProduction -v -count=1

      - name: Layer 1 — Arbiter sweep
        run: cd vaidshala/simulation && go test ./bridge/ -run TestArbiterCompatibility -v -count=1

      - name: Layer 2 — Trajectory tests
        run: cd vaidshala/simulation && go test ./pkg/physiology/ -v -count=1 -timeout=300s

      - name: Performance benchmark
        run: cd vaidshala/simulation && go test ./bridge/ -bench=BenchmarkRunCycle -benchmem
```

- [ ] **Step 2: Verify workflow YAML is valid**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/simulation-gate.yml'))"`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/simulation-gate.yml
git commit -m "feat: add GitHub Actions CI workflow for simulation safety gate"
```

---

### Task 15: Add benchmark test for RunCycle performance

**Files:**
- Modify: `vaidshala/simulation/bridge/bridge_test.go`

- [ ] **Step 1: Write benchmark**

Add to `bridge_test.go`:
```go
func BenchmarkRunCycle(b *testing.B) {
    engine, err := NewProductionEngine(
        WithProtocolRulesPath("testdata/protocol_rules.yaml"),
    )
    if err != nil {
        b.Fatalf("failed to create engine: %v", err)
    }

    vp := patient.GreenTrajectory()
    input := vp.ToTitrationInput(1)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        engine.RunCycle(input)
    }
}
```

- [ ] **Step 2: Run benchmark to establish baseline**

Run: `cd vaidshala/simulation && go test ./bridge/ -bench=BenchmarkRunCycle -benchmem -count=3`
Expected: Output showing ns/op, B/op, allocs/op. Record baseline for future comparison.

- [ ] **Step 3: Commit**

```bash
git add vaidshala/simulation/bridge/bridge_test.go
git commit -m "test: add RunCycle benchmark for performance regression detection"
```

---

## Chunk 7: Final Validation & Integration Verification

### Task 16: Run full test suite end-to-end

**Files:** None — verification only

- [ ] **Step 1: Run all bridge tests**

Run: `cd vaidshala/simulation && go test ./bridge/ -v -count=1`
Expected: All tests PASS (unit converter, GateSignal, round-trip, arbiter 125/125, engine construction)

- [ ] **Step 2: Run all simulation scenario tests**

Run: `cd vaidshala/simulation && go test ./pkg/scenarios/ -run TestSimulation -v -count=1`
Expected: 12/12 PASS

- [ ] **Step 3: Run all production scenario tests**

Run: `cd vaidshala/simulation && go test ./pkg/scenarios/ -run TestProduction -v -count=1`
Expected: 13/13 PASS

- [ ] **Step 4: Run trajectory tests**

Run: `cd vaidshala/simulation && go test ./pkg/physiology/ -v -count=1 -timeout=300s`
Expected: 7 trajectory tests + untreated control PASS

- [ ] **Step 5: Run CLI in all modes**

Run: `cd vaidshala/simulation && go run cmd/main.go`
Run: `cd vaidshala/simulation && go run cmd/main.go --production`
Run: `cd vaidshala/simulation && go run cmd/main.go --production --trajectory`
Expected: All three modes complete with PASS

- [ ] **Step 6: Run full go vet and build**

Run: `cd vaidshala/simulation && go vet ./... && go build ./...`
Expected: No warnings, no errors

- [ ] **Step 7: Final commit**

```bash
cd /Users/apoorvabk/Downloads/cardiofit
git add vaidshala/simulation/
git commit -m "feat: complete simulation module integration

Bridge adapter validates production V-MCU against 13 safety scenarios
+ 125 arbiter combinations + 5 physiology engine trajectories.
CI gate configured for GitHub Actions."
```
