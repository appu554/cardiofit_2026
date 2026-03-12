# Vaidshala CQL Architecture Implementation Plan

**Version**: 1.5 (Phase 0 Implementation Order + File Structure)
**Date**: January 2026
**Scope**: Close 58% CQL gap + 60% KB client gap
**Duration**: 6.5 Weeks (32.5 working days) - Added Phase 0
**CTO/CMO Grade**: A+ / Ready for Execution

---

## 🔴 CRITICAL ARCHITECTURE PRINCIPLE (CTO/CMO Final Directive)

> **"CQL explains. KB-19 recommends. ICU decides."**

This is the **authority hierarchy** that must be respected at all times:

| Layer | Responsibility | Can ICU Override? |
|-------|---------------|-------------------|
| CQL (Tier 0-6a) | Evaluate clinical truth | ✅ Yes |
| KB-19 (Guidelines) | Recommend actions | ✅ Yes |
| KB-18 (Governance) | Route approvals | ✅ Yes |
| KB-14 (Navigator) | Execute workflows | ✅ Yes |
| **ICU Intelligence** | **Assert dominance** | ❌ Only reality |

### The Vaidshala Standard (What Makes This Architecture Correct)

1. **Snapshot Principle**: "CQL does not need a terminology ENGINE at runtime; it needs a terminology ANSWER."
2. **Runtime Principle**: Workflow services (KB-18, KB-19) are *consumers*, not providers.
3. **Dominance Principle**: ICU Intelligence can veto *everything except reality*.

---

## Executive Summary

### Current State
| Metric | Current | Target | Gap |
|--------|---------|--------|-----|
| CQL Tiers | 5/12 (42%) | 12/12 (100%) | 7 tiers |
| KB Clients | 8/20 (40%) | 20/20 (100%) | 12 clients |
| CQL Lines | 7,628 | ~18,000 | ~10,400 |
| ICU Dominance Engine | ❌ Missing | ✅ Required | **PHASE 0 BLOCKER** |

---

## CRITICAL ARCHITECTURE: Snapshot vs Runtime KB Separation

### Architecture Principle (CTO/CMO Directive)
> "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."

**Two categories of KB clients with different integration patterns:**

### Category A: SNAPSHOT KBs (Data Layer - KnowledgeSnapshotBuilder)
Pre-computed at build time → Frozen into `KnowledgeSnapshot` → CQL evaluates against frozen data

| KB | Service | Status | Integration |
|----|---------|--------|-------------|
| KB-1 | Drug Rules | ✅ Exists | `DosingSnapshot` |
| KB-4 | Patient Safety | ✅ Exists | `SafetySnapshot` |
| KB-5 | Drug Interactions | ✅ Exists | `InteractionSnapshot` |
| KB-6 | Formulary | ✅ Exists | `FormularySnapshot` |
| KB-7 | Terminology | ✅ Exists | `TerminologySnapshot` |
| KB-8 | Calculator | ✅ Exists | `CalculatorSnapshot` |
| KB-11 | CDI | ✅ Exists | `CDIFacts` |
| **KB-16** | **Lab Interpretation** | 🟡 **ADD** | `LabInterpretationSnapshot` |

### Category B: RUNTIME KBs (Workflow Layer - RuntimeClients)
Called during execution → Workflow/action capabilities → NOT in snapshot builder

| KB | Service | Status | Purpose |
|----|---------|--------|---------|
| KB-3 | Guidelines | ❌ Missing | Fetch guideline recommendations |
| KB-9 | Care Gaps | ❌ Missing | Close gaps, get gap status |
| KB-10 | Rules Engine | ❌ Missing | Evaluate rules at runtime |
| KB-12 | Order Sets | ❌ Missing | Instantiate order templates |
| KB-13 | Quality Measures | ❌ Missing | Evaluate measures |
| KB-14 | Care Navigator | ❌ Missing | Execute workflows |
| **KB-15** | **Evidence Engine** | ❌ **Missing** | **Fetch evidence, GRADE grading** |
| KB-17 | Population Registry | ❌ Missing | Register patients, get cohorts |
| KB-18 | Governance | ❌ Missing | Approval workflows |
| KB-19 | Protocol Orchestrator | ❌ Missing | Execute protocols |

### Category C: NON-KB (Tier 7 Safety Layer)
ICU Intelligence → Safety veto in Go → Deterministic, non-overrideable

| Component | Status | Purpose |
|-----------|--------|---------|
| ICU Intelligence | ✅ Exists | HARD VETO - blocks unsafe actions before RuntimeClients |

### Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        BUILD TIME (Snapshot Builder)                             │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   PatientContext ───────────────────────────────────────────────────────────►   │
│         │                                                                        │
│         ▼                                                                        │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                    KNOWLEDGE SNAPSHOT BUILDER                            │   │
│   │                                                                          │   │
│   │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │   │
│   │  │  KB-1    │  │  KB-4    │  │  KB-5    │  │  KB-6    │  │  KB-7    │  │   │
│   │  │ Dosing   │  │ Safety   │  │   DDI    │  │Formulary │  │Terminol. │  │   │
│   │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  │   │
│   │       │             │             │             │             │         │   │
│   │  ┌──────────┐  ┌──────────┐  ┌──────────┐                              │   │
│   │  │  KB-8    │  │  KB-11   │  │  KB-16   │  ◄── NEW: Add to builder    │   │
│   │  │Calculator│  │   CDI    │  │Lab Interp│                              │   │
│   │  └────┬─────┘  └────┬─────┘  └────┬─────┘                              │   │
│   │       │             │             │                                     │   │
│   │       └─────────────┴─────────────┴─────────────────────────────────────┘   │
│   │                               │                                              │
│   │                               ▼                                              │
│   │              ┌────────────────────────────────┐                              │
│   │              │   FROZEN KNOWLEDGE SNAPSHOT    │                              │
│   │              │ (Immutable, Deterministic)     │                              │
│   │              └────────────────────────────────┘                              │
│   └─────────────────────────────────────────────────────────────────────────────┘
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        EXECUTION TIME (CQL Engine)                               │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   KnowledgeSnapshot + FHIR Bundle ───────────────────────────────────────────►  │
│         │                                                                        │
│         ▼                                                                        │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                         CQL ENGINE (Tier 0-6a)                           │   │
│   │                                                                          │   │
│   │  "Given this patient context, does this clinical fact hold?"            │   │
│   │                                                                          │   │
│   │  OUTPUTS:                                                                │   │
│   │  • Clinical classifications (risk levels, care gaps, etc.)              │   │
│   │  • Governance classifications (approval levels)                          │   │
│   │  • Alert classifications (priority levels)                               │   │
│   └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          │ CQL Classification Outputs
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│            ICU INTELLIGENCE (Tier 7 - Go) - MANDATORY VETO CHECK                │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   CQL Outputs ──────► ICU SAFETY VETO ──────► Pass/Reject                       │
│                             │                                                    │
│                             │ HARD STOP: Deterministic, Non-overrideable        │
│                             │ Any vetoed action is BLOCKED before workflow      │
│                             │                                                    │
│                             ▼                                                    │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │  ICUIntelligenceClient.Evaluate(action, safetyFacts) → VetoResult       │   │
│   │                                                                          │   │
│   │  if VetoResult.Vetoed {                                                  │   │
│   │      audit.RecordVeto(vetoResult) // Immutable audit trail              │   │
│   │      return BLOCKED // No workflow execution                            │   │
│   │  }                                                                       │   │
│   └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          │ ONLY if NOT vetoed
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        WORKFLOW TIME (Runtime Clients)                           │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   CQL Outputs (vetted) ─────────────────────────────────────────────────────►   │
│         │                                                                        │
│         ▼                                                                        │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                      RUNTIME CLIENTS (Workflow Layer)                    │   │
│   │                                                                          │   │
│   │  Called AFTER ICU veto check for ACTIONS (not data):                    │   │
│   │                                                                          │   │
│   │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │   │
│   │  │  KB-14   │  │  KB-18   │  │  KB-19   │  │  KB-10   │  │  KB-9    │  │   │
│   │  │Navigator │  │Governance│  │ Protocol │  │  Rules   │  │Care Gaps │  │   │
│   │  │(workflow)│  │(approval)│  │(orchestr)│  │ (eval)   │  │ (close)  │  │   │
│   │  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │   │
│   │                                                                          │   │
│   │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │   │
│   │  │  KB-13   │  │  KB-17   │  │  KB-12   │  │  KB-3    │  │  KB-15   │  │   │
│   │  │ Quality  │  │ Pop Reg  │  │Order Sets│  │Guidelines│  │Evidence  │  │   │
│   │  │(evaluate)│  │(register)│  │(instant) │  │ (fetch)  │  │ (GRADE)  │  │   │
│   │  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │   │
│   └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

### Implementation Timeline (CORRECTED with Phase 0)

```
╔═══════════════════════════════════════════════════════════════════════════════╗
║  🔴 PHASE 0 (Day 1-3): ICU DOMINANCE ENGINE - MUST COMPLETE FIRST            ║
╠═══════════════════════════════════════════════════════════════════════════════╣
║  • ICU State Machine (5 states only)                                         ║
║  • Veto Contract Interface                                                   ║
║  • Context Gate (IsInICU || IsCodeActive)                                    ║
║  • Authority Boundary Lock                                                   ║
╚═══════════════════════════════════════════════════════════════════════════════╝

PHASE 1 (Week 1-2): Foundation Layer
├── Tier 3: Domain Commons (5 CQL files)
├── KB-16 Client [SNAPSHOT] ◄── Add to KnowledgeSnapshotBuilder
└── Update KnowledgeSnapshotBuilder with LabInterpretationSnapshot

PHASE 2 (Week 2-3): Integration Layer
├── Tier 1.5: Clinical Utilities (2 CQL files)
├── Tier-6a: Classification ONLY (Go boundary enforcement)
├── KB-10 Client [RUNTIME] - Rules Engine
├── KB-18 Client [RUNTIME] - Governance
└── Create RuntimeClients structure

PHASE 3 (Week 3-4): Orchestration Layer
├── Tier 6a: Orchestration CQL (5 CQL files) ◄── Classification only!
├── Tier-6b: Execution logic (Go)            ◄── Branching/workflow
├── KB-9 Client [RUNTIME] - Care Gaps
├── KB-13 Client [RUNTIME] - Quality Measures
├── KB-14 Client [RUNTIME] - Care Navigator
└── KB-17 Client [RUNTIME] - Population Registry

PHASE 4 (Week 5-6): Regional & Guidelines
├── Tier 5: Regional Adapters (4 CQL files)
├── Tier 4b: Clinical Guidelines (5 CQL files)
├── KB-3 Client [RUNTIME] - Guidelines
├── KB-12 Client [RUNTIME] - Order Sets
└── KB-19 Client [RUNTIME] - Protocol Orchestrator (CONSTRAINED SCOPE)
```

---

## 🔴 Phase 0: ICU Dominance Engine (Day 1-3) - MANDATORY FIRST

### Why Phase 0 Comes First

> **"Before writing any more CQL, finalize the ICU Dominance State Machine."**
> — CTO/CMO Directive

The ICU Dominance Engine **defines authority boundaries** that inform:
- How KB-19 must behave (recommendations only, no veto authority)
- What CQL is allowed to decide (truth, not action)
- When ICU dominance is asserted (context gate)

### 0.1 ICU Dominance State Machine (6 States)

**Location**: `vaidshala/clinical-runtime-platform/icu/dominance_engine.go`

```go
// Package icu implements the ICU Intelligence Dominance Engine.
// This is NOT a KB - it is a state-dominance engine that can veto everything else.
//
// ARCHITECTURE CRITICAL:
// - ICU asserts dominance, it does not "recommend"
// - ICU can override KB-19, KB-18, KB-14, and CQL outputs
// - ICU cannot override reality
package icu

// DominanceState represents the 6 possible ICU dominance states.
// Using string type for better logging, debugging, and serialization.
type DominanceState string

const (
	// StateNone - No ICU dominance, normal workflow proceeds
	StateNone DominanceState = "NONE"

	// StateShock - Hemodynamic instability dominates all decisions
	// Triggers: MAP <65, Lactate >4, Vasopressor requirement
	StateShock DominanceState = "SHOCK"

	// StateHypoxia - Respiratory failure dominates all decisions
	// Triggers: SpO2 <88%, P/F ratio <100, FiO2 >0.6
	StateHypoxia DominanceState = "HYPOXIA"

	// StateActiveBleed - Active hemorrhage dominates all decisions
	// Triggers: Hgb drop >2g/dL/6h, Active transfusion, Surgical bleeding
	StateActiveBleed DominanceState = "ACTIVE_BLEED"

	// StateLowOutputFailure - Cardiogenic/distributive failure dominates
	// Triggers: CI <2.0, ScvO2 <60%, Inotrope escalation
	StateLowOutputFailure DominanceState = "LOW_OUTPUT_FAILURE"

	// StateNeurologicCollapse - CNS crisis dominates all decisions
	// Triggers: GCS <8, Active seizure, ICP >20, Herniation signs
	StateNeurologicCollapse DominanceState = "NEUROLOGIC_COLLAPSE"
)

// DominanceResult represents the outcome of an ICU dominance evaluation
type DominanceResult struct {
	// CurrentState is the active dominance state
	CurrentState DominanceState

	// Vetoed indicates if the proposed action is blocked
	Vetoed bool

	// VetoReason explains why the action was blocked
	VetoReason string

	// TriggeringRule identifies which safety rule triggered the veto
	TriggeringRule string

	// OverriddenRecommendations lists what KB-19 recommendations are ignored
	OverriddenRecommendations []string
}

// SafetyFacts contains all clinical facts needed for dominance classification.
// These facts are gathered from real-time monitoring and FHIR resources.
type SafetyFacts struct {
	// ─── Context Flags ───────────────────────────────────────────────────────
	IsInICU               bool    // Patient currently in ICU
	IsCodeActive          bool    // Code Blue/ACLS in progress
	IsCriticallyUnstable  bool    // Derived from vital trends

	// ─── Neurologic (Priority 1) ─────────────────────────────────────────────
	GCS                   int     // Glasgow Coma Scale (3-15)
	HasActiveSeizure      bool    // Witnessed seizure activity
	ICP                   float64 // Intracranial pressure (mmHg)
	HasHerniationSigns    bool    // Cushing's triad, blown pupil

	// ─── Hemodynamic (Priority 2) ────────────────────────────────────────────
	MAP                   float64 // Mean arterial pressure (mmHg)
	Lactate               float64 // Serum lactate (mmol/L)
	OnVasopressors        bool    // Any vasopressor infusion active
	HasSepticShock        bool    // Sepsis-3 criteria met

	// ─── Respiratory (Priority 3) ────────────────────────────────────────────
	SpO2                  float64 // Oxygen saturation (%)
	PFRatio               float64 // PaO2/FiO2 ratio
	FiO2                  float64 // Fraction of inspired oxygen (0.0-1.0)
	OnMechanicalVent      bool    // Currently intubated/ventilated

	// ─── Bleeding (Priority 4) ───────────────────────────────────────────────
	HgbDrop6h             float64 // Hemoglobin drop in last 6 hours (g/dL)
	HasActiveTransfusion  bool    // PRBC transfusion in progress
	HasSurgicalBleeding   bool    // Post-op or traumatic bleeding
	INR                   float64 // International Normalized Ratio
	HasActiveBleeding     bool    // Clinically evident bleeding

	// ─── Cardiac Output (Priority 5) ─────────────────────────────────────────
	CardiacIndex          float64 // Cardiac index (L/min/m²)
	ScvO2                 float64 // Central venous oxygen saturation (%)
	OnInotropeEscalation  bool    // Inotrope dose increasing
	HasAKI                bool    // Acute kidney injury (KDIGO stage ≥2)
	HasALF                bool    // Acute liver failure
}

// DominanceEngine evaluates ICU state dominance
type DominanceEngine struct {
	// State machine configuration
	config *DominanceConfig
}

// Evaluate checks if ICU dominance should veto an action
// CRITICAL: This is called BEFORE any RuntimeClient KB
func (e *DominanceEngine) Evaluate(ctx context.Context, action ProposedAction, facts SafetyFacts) (*DominanceResult, error) {
	// ═══════════════════════════════════════════════════════════════════════
	// CONTEXT GATE: Only assert dominance in ICU/Code contexts
	// ═══════════════════════════════════════════════════════════════════════
	if !facts.IsInICU && !facts.IsCodeActive {
		// Not in ICU context - pass unless absolute contraindication
		return e.evaluateAbsoluteContraindicationsOnly(ctx, action, facts)
	}

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 1: Classify dominance state (explicit classifier)
	// ═══════════════════════════════════════════════════════════════════════
	state := e.ClassifyDominanceState(facts)

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 2: Evaluate action against dominance state
	// ═══════════════════════════════════════════════════════════════════════
	switch state {
	case StateShock:
		return e.evaluateShockDominance(ctx, action, facts)
	case StateHypoxia:
		return e.evaluateHypoxiaDominance(ctx, action, facts)
	case StateActiveBleed:
		return e.evaluateActiveBleedDominance(ctx, action, facts)
	case StateLowOutputFailure:
		return e.evaluateLowOutputDominance(ctx, action, facts)
	case StateNeurologicCollapse:
		return e.evaluateNeurologicDominance(ctx, action, facts)
	default:
		return &DominanceResult{CurrentState: StateNone, Vetoed: false}, nil
	}
}

// ClassifyDominanceState is the EXPLICIT State Classifier.
//
// ═══════════════════════════════════════════════════════════════════════════════
// PRIORITY ORDER (highest to lowest):
//   1. NEUROLOGIC_COLLAPSE - Brain death/herniation trumps everything
//   2. SHOCK              - Hemodynamic instability is next
//   3. HYPOXIA            - Respiratory failure follows
//   4. ACTIVE_BLEED       - Hemorrhage control
//   5. LOW_OUTPUT_FAILURE - Cardiac output failure
//   6. NONE               - Normal state, no dominance
//
// CLINICAL RATIONALE:
// - Neurologic collapse can cause ALL other states (code blue, herniation)
// - Shock kills faster than hypoxia (minutes vs hours)
// - Hypoxia compounds all other states rapidly
// - Active bleeding must be addressed before optimizing cardiac output
// ═══════════════════════════════════════════════════════════════════════════════
func (e *DominanceEngine) ClassifyDominanceState(facts SafetyFacts) DominanceState {

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 1: NEUROLOGIC_COLLAPSE
	// GCS <8, Active seizure, ICP >20, Herniation signs
	// ─────────────────────────────────────────────────────────────────────────
	if facts.GCS < 8 || facts.HasActiveSeizure || facts.ICP > 20 || facts.HasHerniationSigns {
		return StateNeurologicCollapse
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 2: SHOCK
	// MAP <65, Lactate >4, Vasopressor requirement, Septic shock
	// ─────────────────────────────────────────────────────────────────────────
	if facts.MAP < 65 || facts.Lactate > 4.0 || facts.OnVasopressors || facts.HasSepticShock {
		return StateShock
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 3: HYPOXIA
	// SpO2 <88%, P/F ratio <100, FiO2 >0.6, Mechanical ventilation with distress
	// ─────────────────────────────────────────────────────────────────────────
	if facts.SpO2 < 88 || facts.PFRatio < 100 || facts.FiO2 > 0.6 {
		return StateHypoxia
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 4: ACTIVE_BLEED
	// Hgb drop >2g/dL/6h, Active transfusion, Surgical bleeding, Critical INR
	// ─────────────────────────────────────────────────────────────────────────
	if facts.HgbDrop6h > 2.0 || facts.HasActiveTransfusion || facts.HasSurgicalBleeding ||
	   (facts.INR > 4.0 && facts.HasActiveBleeding) {
		return StateActiveBleed
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 5: LOW_OUTPUT_FAILURE
	// CI <2.0, ScvO2 <60%, Inotrope escalation, Combined AKI + ALF
	// ─────────────────────────────────────────────────────────────────────────
	if facts.CardiacIndex < 2.0 || facts.ScvO2 < 60 || facts.OnInotropeEscalation ||
	   (facts.HasAKI && facts.HasALF) {
		return StateLowOutputFailure
	}

	// ─────────────────────────────────────────────────────────────────────────
	// PRIORITY 6: NONE - No dominance state active
	// ─────────────────────────────────────────────────────────────────────────
	return StateNone
}
```

### 0.2 Veto Contract Interface

**Location**: `vaidshala/clinical-runtime-platform/contracts/veto_contract.go`

```go
// VetoContract defines the authority boundary between ICU Intelligence and KB-19
//
// This contract MUST be locked before any KB-19 implementation
type VetoContract interface {
	// CanICUVeto returns true if ICU has authority to veto this action type
	CanICUVeto(actionType ActionType) bool

	// CanKB19Recommend returns true if KB-19 can make recommendations for this state
	// KB-19 can ALWAYS recommend, but ICU can ALWAYS veto
	CanKB19Recommend(state DominanceState) bool

	// MustDeferToICU returns true if KB-19 must defer to ICU for this action
	MustDeferToICU(action ProposedAction, state DominanceState) bool
}

// DefaultVetoContract implements the standard authority boundary
type DefaultVetoContract struct{}

func (c *DefaultVetoContract) CanICUVeto(actionType ActionType) bool {
	// ICU can veto EVERYTHING except documentation
	return actionType != ActionTypeDocumentation
}

func (c *DefaultVetoContract) CanKB19Recommend(state DominanceState) bool {
	// KB-19 can always recommend - but recommendations may be overridden
	return true
}

func (c *DefaultVetoContract) MustDeferToICU(action ProposedAction, state DominanceState) bool {
	// In dominant states, high-risk actions MUST defer to ICU
	if state == StateNormal {
		return false
	}
	return action.RiskLevel == RiskLevelHigh || action.RiskLevel == RiskLevelCritical
}
```

### 0.3 Context Gate Pattern

**CRITICAL**: ICU dominance should only assert in ICU/Code contexts:

```go
// ContextGate determines if ICU dominance evaluation is required
func (e *DominanceEngine) requiresDominanceEvaluation(facts SafetyFacts) bool {
	// Gate 1: Patient location
	if facts.IsInICU {
		return true
	}

	// Gate 2: Active emergency code
	if facts.IsCodeActive {
		return true
	}

	// Gate 3: Critical vitals regardless of location
	if facts.IsCriticallyUnstable() {
		return true
	}

	// Default: Standard workflow, no ICU dominance needed
	return false
}
```

### 0.4 Phase 0 Deliverables

**Directory Structure**:
```
vaidshala/clinical-runtime-platform/
├── icu/
│   ├── dominance_state.go    # 6 DominanceState constants + Priority()
│   ├── safety_facts.go       # SafetyFacts input struct (25 fields)
│   ├── dominance_engine.go   # ClassifyDominanceState + Evaluate()
│   └── dominance_test.go     # Golden path + edge case tests
└── contracts/
    └── veto_contract.go      # VetoContract interface (authority boundary)
```

| Deliverable | Type | Location | Lines | Purpose |
|-------------|------|----------|-------|---------|
| dominance_state.go | Go | icu/ | ~120 | 6 states + Priority() + helpers |
| safety_facts.go | Go | icu/ | ~80 | Clinical facts input struct |
| dominance_engine.go | Go | icu/ | ~250 | ClassifyDominanceState + context gate |
| veto_contract.go | Go | contracts/ | ~100 | Authority boundary interface |
| dominance_test.go | Go | icu/ | ~200 | Test coverage for all 6 states |

**Total Phase 0 Lines**: ~750 lines
**Duration**: 3 days

> **⚠️ BLOCKING**: Phases 1-4 cannot begin until Phase 0 is complete and the veto contract is locked.

### 0.5 Implementation Order

```
Step 1: dominance_state.go     ← State definitions (foundation)
Step 2: safety_facts.go        ← Input struct for classifier
Step 3: veto_contract.go       ← Authority boundary interface
Step 4: dominance_engine.go    ← Classifier + Evaluate (uses 1-3)
Step 5: dominance_test.go      ← Verify all states classify correctly
```

---

## Phase 1: Foundation Layer (Week 1-2)

### Objective
Create the clinical semantic foundation that unblocks ICU Intelligence and enables KB-3, KB-4, KB-5, KB-9, KB-19 integration.

### 1.1 Tier 3: Domain Commons CQL

#### 1.1.1 SafetyCommon.cql (CRITICAL - Unblocks ICU Intelligence)

**Location**: `vaidshala/clinical-knowledge-core/tier-3-domain-commons/safety/SafetyCommon.cql`

**Specification**:
```cql
/*
 * SafetyCommon CQL Library
 * ============================================================
 * Tier 3 Domain Commons - Clinical Safety Patterns
 *
 * Purpose: Shared safety definitions consumed by:
 *   - ICU Intelligence (Tier 7)
 *   - KB-4 Patient Safety
 *   - KB-5 Drug Interactions
 *
 * Governance: Tier-3 review required (Clinical + Technical + Safety)
 * ============================================================
 */
library SafetyCommon version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations
include MedicationHelpers version '1.0.0' called Medications

// ============================================================
// VALUE SET REFERENCES (KB-7 Terminology)
// ============================================================

// High-Alert Medications (ISMP categories)
valueset "High Alert Medications": 'http://cardiofit.io/fhir/ValueSet/high-alert-medications'
valueset "Anticoagulants": 'http://cardiofit.io/fhir/ValueSet/anticoagulants'
valueset "Opioids": 'http://cardiofit.io/fhir/ValueSet/opioids'
valueset "Insulins": 'http://cardiofit.io/fhir/ValueSet/insulins'
valueset "Chemotherapy Agents": 'http://cardiofit.io/fhir/ValueSet/chemotherapy-agents'
valueset "Sedatives": 'http://cardiofit.io/fhir/ValueSet/sedatives'
valueset "Neuromuscular Blockers": 'http://cardiofit.io/fhir/ValueSet/neuromuscular-blockers'

// Allergen Classes
valueset "Penicillin Allergens": 'http://cardiofit.io/fhir/ValueSet/penicillin-allergens'
valueset "Sulfa Allergens": 'http://cardiofit.io/fhir/ValueSet/sulfa-allergens'
valueset "NSAID Allergens": 'http://cardiofit.io/fhir/ValueSet/nsaid-allergens'
valueset "Contrast Allergens": 'http://cardiofit.io/fhir/ValueSet/contrast-allergens'

// Critical Conditions
valueset "Anaphylaxis Conditions": 'http://cardiofit.io/fhir/ValueSet/anaphylaxis-conditions'
valueset "Severe Bleeding Conditions": 'http://cardiofit.io/fhir/ValueSet/severe-bleeding'
valueset "Acute Kidney Injury": 'http://cardiofit.io/fhir/ValueSet/aki-conditions'
valueset "Acute Liver Failure": 'http://cardiofit.io/fhir/ValueSet/alf-conditions'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// HIGH-ALERT MEDICATION DETECTION
// ============================================================

/*
 * Check if patient has any active high-alert medications
 * Returns: List of active high-alert medication requests
 */
define "Active High Alert Medications":
  [MedicationRequest: "High Alert Medications"] MR
    where Medications.IsMedicationActive(MR)

define "Has Active High Alert Medication":
  exists "Active High Alert Medications"

/*
 * Check for specific high-alert categories
 */
define "Active Anticoagulants":
  [MedicationRequest: "Anticoagulants"] MR
    where Medications.IsMedicationActive(MR)

define "Active Opioids":
  [MedicationRequest: "Opioids"] MR
    where Medications.IsMedicationActive(MR)

define "Active Insulins":
  [MedicationRequest: "Insulins"] MR
    where Medications.IsMedicationActive(MR)

define "Active Sedatives":
  [MedicationRequest: "Sedatives"] MR
    where Medications.IsMedicationActive(MR)

define "Active Neuromuscular Blockers":
  [MedicationRequest: "Neuromuscular Blockers"] MR
    where Medications.IsMedicationActive(MR)

define "On Multiple High Alert Categories":
  (if exists "Active Anticoagulants" then 1 else 0) +
  (if exists "Active Opioids" then 1 else 0) +
  (if exists "Active Insulins" then 1 else 0) +
  (if exists "Active Sedatives" then 1 else 0) > 1

// ============================================================
// ALLERGY DETECTION
// ============================================================

/*
 * Active confirmed allergies (not refuted/entered-in-error)
 */
define "Active Allergies":
  [AllergyIntolerance] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { system: FHIR.uri { value: 'http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical' }, code: FHIR.code { value: 'active' } } }
    })
    and A.verificationStatus !~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { system: FHIR.uri { value: 'http://terminology.hl7.org/CodeSystem/allergyintolerance-verification' }, code: FHIR.code { value: 'entered-in-error' } } }
    })

/*
 * Check for specific allergen class allergies
 */
define "Has Penicillin Allergy":
  exists ([AllergyIntolerance: "Penicillin Allergens"] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    }))

define "Has Sulfa Allergy":
  exists ([AllergyIntolerance: "Sulfa Allergens"] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    }))

define "Has NSAID Allergy":
  exists ([AllergyIntolerance: "NSAID Allergens"] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    }))

define "Has Contrast Allergy":
  exists ([AllergyIntolerance: "Contrast Allergens"] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    }))

/*
 * Severe allergy history (anaphylaxis)
 */
define "History of Anaphylaxis":
  exists [Condition: "Anaphylaxis Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })
    or C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'resolved' } } }
    })

// ============================================================
// CRITICAL CONDITION DETECTION (ICU Intelligence Input)
// ============================================================

/*
 * Active severe bleeding - CRITICAL for anticoagulant safety
 */
define "Has Active Severe Bleeding":
  exists [Condition: "Severe Bleeding Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Acute Kidney Injury - CRITICAL for nephrotoxic drug safety
 */
define "Has Acute Kidney Injury":
  exists [Condition: "Acute Kidney Injury"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Acute Liver Failure - CRITICAL for hepatotoxic drug safety
 */
define "Has Acute Liver Failure":
  exists [Condition: "Acute Liver Failure"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// SAFETY RISK SCORING (Classification - NOT enforcement)
// ============================================================

/*
 * Overall Safety Risk Level
 * OUTPUT: Integer score for classification
 * NOTE: This CLASSIFIES risk. KB-4/ICU ENFORCES actions.
 */
define "Safety Risk Score":
  (if "Has Active High Alert Medication" then 2 else 0) +
  (if "On Multiple High Alert Categories" then 3 else 0) +
  (if "History of Anaphylaxis" then 5 else 0) +
  (if "Has Active Severe Bleeding" then 5 else 0) +
  (if "Has Acute Kidney Injury" then 4 else 0) +
  (if "Has Acute Liver Failure" then 4 else 0)

define "Safety Risk Level":
  case
    when "Safety Risk Score" >= 10 then 'CRITICAL'
    when "Safety Risk Score" >= 5 then 'HIGH'
    when "Safety Risk Score" >= 2 then 'MODERATE'
    else 'LOW'
  end

// ============================================================
// DRUG-CONDITION CONTRAINDICATION FLAGS
// ============================================================

/*
 * Anticoagulant Contraindication Flags
 * OUTPUT: Boolean flags for ICU Intelligence consumption
 */
define "Anticoagulant Contraindicated Due To Bleeding":
  "Has Active Severe Bleeding" and exists "Active Anticoagulants"

define "NSAID Contraindicated Due To AKI":
  "Has Acute Kidney Injury"

define "Hepatotoxic Drug Contraindicated":
  "Has Acute Liver Failure"

// ============================================================
// POLYPHARMACY DETECTION
// ============================================================

/*
 * Count of active medications
 */
define "Active Medication Count":
  Count([MedicationRequest] MR where Medications.IsMedicationActive(MR))

define "Has Polypharmacy":
  "Active Medication Count" >= 5

define "Has Extreme Polypharmacy":
  "Active Medication Count" >= 10

define "Polypharmacy Risk Level":
  case
    when "Active Medication Count" >= 15 then 'CRITICAL'
    when "Active Medication Count" >= 10 then 'HIGH'
    when "Active Medication Count" >= 5 then 'MODERATE'
    else 'LOW'
  end
```

**Dependencies**:
- Tier 0: FHIRHelpers
- Tier 1: IntervalHelpers, ObservationHelpers, MedicationHelpers
- KB-7: ValueSet expansions for all referenced value sets

**Testing Requirements**:
1. Unit tests with mock FHIR bundles containing high-alert medications
2. Test allergy detection with various clinical statuses
3. Validate risk score calculation across scenarios
4. Test polypharmacy detection thresholds

**Estimated Lines**: ~350 lines

---

#### 1.1.2 CardiovascularCommon.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-3-domain-commons/cardiometabolic/CardiovascularCommon.cql`

**Specification**:
```cql
/*
 * CardiovascularCommon CQL Library
 * ============================================================
 * Tier 3 Domain Commons - Cardiovascular Patterns
 *
 * Purpose: Shared cardiovascular definitions consumed by:
 *   - KB-3 Guidelines
 *   - KB-9 Care Gaps
 *   - KB-19 Protocol Orchestrator
 *   - CMS Quality Measures
 *
 * Governance: Tier-3 review required (Clinical + Technical)
 * ============================================================
 */
library CardiovascularCommon version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations

// ============================================================
// VALUE SET REFERENCES
// ============================================================

// Conditions
valueset "Hypertension": 'http://cardiofit.io/fhir/ValueSet/hypertension'
valueset "Heart Failure": 'http://cardiofit.io/fhir/ValueSet/heart-failure'
valueset "Coronary Artery Disease": 'http://cardiofit.io/fhir/ValueSet/cad'
valueset "Atrial Fibrillation": 'http://cardiofit.io/fhir/ValueSet/atrial-fibrillation'
valueset "Stroke or TIA": 'http://cardiofit.io/fhir/ValueSet/stroke-tia'
valueset "Peripheral Arterial Disease": 'http://cardiofit.io/fhir/ValueSet/pad'
valueset "Dyslipidemia": 'http://cardiofit.io/fhir/ValueSet/dyslipidemia'

// LOINC Codes
valueset "Systolic Blood Pressure": 'http://cardiofit.io/fhir/ValueSet/systolic-bp'
valueset "Diastolic Blood Pressure": 'http://cardiofit.io/fhir/ValueSet/diastolic-bp'
valueset "LDL Cholesterol": 'http://cardiofit.io/fhir/ValueSet/ldl-cholesterol'
valueset "Total Cholesterol": 'http://cardiofit.io/fhir/ValueSet/total-cholesterol'
valueset "HDL Cholesterol": 'http://cardiofit.io/fhir/ValueSet/hdl-cholesterol'
valueset "Triglycerides": 'http://cardiofit.io/fhir/ValueSet/triglycerides'
valueset "BNP Labs": 'http://cardiofit.io/fhir/ValueSet/bnp'
valueset "Troponin Labs": 'http://cardiofit.io/fhir/ValueSet/troponin'

// Medications
valueset "Statins": 'http://cardiofit.io/fhir/ValueSet/statins'
valueset "ACE Inhibitors": 'http://cardiofit.io/fhir/ValueSet/ace-inhibitors'
valueset "ARBs": 'http://cardiofit.io/fhir/ValueSet/arbs'
valueset "Beta Blockers": 'http://cardiofit.io/fhir/ValueSet/beta-blockers'
valueset "Antiplatelet Agents": 'http://cardiofit.io/fhir/ValueSet/antiplatelet'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// CONDITION DETECTION
// ============================================================

define "Has Hypertension":
  exists [Condition: "Hypertension"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Heart Failure":
  exists [Condition: "Heart Failure"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Coronary Artery Disease":
  exists [Condition: "Coronary Artery Disease"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Atrial Fibrillation":
  exists [Condition: "Atrial Fibrillation"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has History of Stroke or TIA":
  exists [Condition: "Stroke or TIA"]

define "Has Peripheral Arterial Disease":
  exists [Condition: "Peripheral Arterial Disease"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Dyslipidemia":
  exists [Condition: "Dyslipidemia"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has ASCVD":
  "Has Coronary Artery Disease"
    or "Has History of Stroke or TIA"
    or "Has Peripheral Arterial Disease"

// ============================================================
// BLOOD PRESSURE MEASUREMENTS
// ============================================================

define "All Systolic BP Readings":
  [Observation: "Systolic Blood Pressure"] O
    where O.status in { 'final', 'amended', 'corrected' }

define "Most Recent Systolic BP":
  Observations.MostRecentObservation("All Systolic BP Readings")

define "Most Recent Systolic BP Value":
  Observations.ObservationQuantityValue("Most Recent Systolic BP")

define "All Diastolic BP Readings":
  [Observation: "Diastolic Blood Pressure"] O
    where O.status in { 'final', 'amended', 'corrected' }

define "Most Recent Diastolic BP":
  Observations.MostRecentObservation("All Diastolic BP Readings")

define "Most Recent Diastolic BP Value":
  Observations.ObservationQuantityValue("Most Recent Diastolic BP")

/*
 * Blood Pressure Classification (ACC/AHA 2017)
 */
define "BP Classification":
  case
    when "Most Recent Systolic BP Value" >= 180 or "Most Recent Diastolic BP Value" >= 120
      then 'Hypertensive Crisis'
    when "Most Recent Systolic BP Value" >= 140 or "Most Recent Diastolic BP Value" >= 90
      then 'Stage 2 Hypertension'
    when "Most Recent Systolic BP Value" >= 130 or "Most Recent Diastolic BP Value" >= 80
      then 'Stage 1 Hypertension'
    when "Most Recent Systolic BP Value" >= 120
      then 'Elevated'
    when "Most Recent Systolic BP Value" < 120 and "Most Recent Diastolic BP Value" < 80
      then 'Normal'
    else 'Unknown'
  end

define "Has Uncontrolled Hypertension":
  "Has Hypertension" and (
    "Most Recent Systolic BP Value" >= 140 or "Most Recent Diastolic BP Value" >= 90
  )

// ============================================================
// LIPID PANEL
// ============================================================

define "Most Recent LDL":
  Observations.MostRecentObservation(
    [Observation: "LDL Cholesterol"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent LDL Value":
  Observations.ObservationQuantityValue("Most Recent LDL")

define "Most Recent Total Cholesterol":
  Observations.MostRecentObservation(
    [Observation: "Total Cholesterol"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent HDL":
  Observations.MostRecentObservation(
    [Observation: "HDL Cholesterol"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Triglycerides":
  Observations.MostRecentObservation(
    [Observation: "Triglycerides"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

/*
 * LDL Goal Assessment based on risk
 */
define "LDL At Goal for ASCVD":
  "Most Recent LDL Value" < 70

define "LDL At Goal for High Risk":
  "Most Recent LDL Value" < 100

define "Needs LDL Reduction":
  case
    when "Has ASCVD" then not "LDL At Goal for ASCVD"
    when "Has Dyslipidemia" then not "LDL At Goal for High Risk"
    else "Most Recent LDL Value" >= 190
  end

// ============================================================
// CARDIAC BIOMARKERS
// ============================================================

define "Most Recent BNP":
  Observations.MostRecentObservation(
    [Observation: "BNP Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent BNP Value":
  Observations.ObservationQuantityValue("Most Recent BNP")

define "Has Elevated BNP":
  "Most Recent BNP Value" > 100

define "Most Recent Troponin":
  Observations.MostRecentObservation(
    [Observation: "Troponin Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Has Elevated Troponin":
  Observations.ObservationQuantityValue("Most Recent Troponin") > 0.04

// ============================================================
// MEDICATION THERAPY ASSESSMENT
// ============================================================

define "On Statin Therapy":
  exists [MedicationRequest: "Statins"] MR
    where MR.status = 'active'

define "On ACE Inhibitor or ARB":
  exists [MedicationRequest: "ACE Inhibitors"] MR
    where MR.status = 'active'
  or exists [MedicationRequest: "ARBs"] MR
    where MR.status = 'active'

define "On Beta Blocker":
  exists [MedicationRequest: "Beta Blockers"] MR
    where MR.status = 'active'

define "On Antiplatelet Therapy":
  exists [MedicationRequest: "Antiplatelet Agents"] MR
    where MR.status = 'active'

// ============================================================
// CARE GAPS (for KB-9)
// ============================================================

define "Needs Statin for ASCVD":
  "Has ASCVD" and not "On Statin Therapy"

define "Needs ACE/ARB for Heart Failure":
  "Has Heart Failure" and not "On ACE Inhibitor or ARB"

define "Needs Beta Blocker for Heart Failure":
  "Has Heart Failure" and not "On Beta Blocker"

define "Needs Antiplatelet for ASCVD":
  "Has ASCVD" and not "On Antiplatelet Therapy"

// ============================================================
// RISK STRATIFICATION (Classification for KB-11/KB-17)
// ============================================================

define "Cardiovascular Risk Factors":
  (if "Has Hypertension" then 1 else 0) +
  (if "Has Dyslipidemia" then 1 else 0) +
  (if Intervals.AgeInYearsAt(Today()) >= 65 then 1 else 0) +
  (if "Has Coronary Artery Disease" then 2 else 0) +
  (if "Has Heart Failure" then 2 else 0) +
  (if "Has Atrial Fibrillation" then 1 else 0) +
  (if "Has History of Stroke or TIA" then 2 else 0)

define "Cardiovascular Risk Level":
  case
    when "Cardiovascular Risk Factors" >= 5 then 'VERY HIGH'
    when "Cardiovascular Risk Factors" >= 3 then 'HIGH'
    when "Cardiovascular Risk Factors" >= 1 then 'MODERATE'
    else 'LOW'
  end
```

**Estimated Lines**: ~400 lines

---

#### 1.1.3 DiabetesCommon.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-3-domain-commons/cardiometabolic/DiabetesCommon.cql`

**Key Definitions**:
- `"Has Type 1 Diabetes"`, `"Has Type 2 Diabetes"`, `"Has Gestational Diabetes"`
- `"Most Recent HbA1c Value"`, `"HbA1c At Goal"` (individualized targets)
- `"Most Recent Fasting Glucose"`, `"Has Hypoglycemia History"`
- `"On Metformin"`, `"On Insulin"`, `"On SGLT2 Inhibitor"`, `"On GLP1 Agonist"`
- `"Diabetes Risk Level"` (for KB-11/KB-17 consumption)
- Care gaps: `"Needs HbA1c Test"`, `"Needs Eye Exam"`, `"Needs Foot Exam"`

**Estimated Lines**: ~350 lines

---

#### 1.1.4 RenalCommon.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-3-domain-commons/cardiometabolic/RenalCommon.cql`

**Key Definitions**:
- `"Has Chronic Kidney Disease"`, `"CKD Stage"` (1-5 based on eGFR)
- `"Most Recent eGFR Value"`, `"Most Recent Serum Creatinine"`
- `"Most Recent Urine Albumin Creatinine Ratio"` (UACR)
- `"Has Dialysis Dependence"`, `"Has Kidney Transplant"`
- `"Nephrotoxic Drug Risk Level"` (for KB-4/KB-5)
- `"Needs Nephrology Referral"` (eGFR < 30 or rapidly declining)

**Estimated Lines**: ~300 lines

---

#### 1.1.5 MaternalHealthCommon.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-3-domain-commons/maternal/MaternalHealthCommon.cql`

**Key Definitions**:
- `"Is Currently Pregnant"`, `"Pregnancy Trimester"`
- `"Has Gestational Hypertension"`, `"Has Preeclampsia"`
- `"Has Gestational Diabetes"`
- `"Pregnancy Category X Medications Active"` (CRITICAL for KB-4)
- `"Needs Prenatal Screening"` (care gaps)
- `"Maternal Risk Level"` (for KB-4 safety scoring)

**Estimated Lines**: ~250 lines

---

### 1.2 KB Client Implementation

#### 1.2.1 kb11_http_client.go (Population Health)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb11_http_client.go`

**Interface Definition**:
```go
// Package clients provides HTTP clients for KB services.
//
// KB11HTTPClient implements the KB11Client interface for KB-11 Population Health Service.
// It provides population segmentation, risk stratification, and cohort management.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// This client is used by KnowledgeSnapshotBuilder to populate PopulationSnapshot.
// Population-level analytics are pre-computed - engines work with frozen cohort data.
//
// Connects to: http://localhost:8089 (Docker: kb11-population-health)
package clients

import (
	"context"
	"vaidshala/clinical-runtime-platform/contracts"
)

// KB11Client defines the interface for Population Health service interactions.
type KB11Client interface {
	// GetPatientRiskStrata returns risk stratification for a patient
	GetPatientRiskStrata(ctx context.Context, patientID string) (*contracts.RiskStratification, error)

	// GetCohortMembers returns patients matching cohort criteria
	GetCohortMembers(ctx context.Context, cohortID string, limit int) ([]string, error)

	// GetPopulationMetrics returns aggregate metrics for a population
	GetPopulationMetrics(ctx context.Context, populationID string) (*contracts.PopulationMetrics, error)

	// GetRiskProjection returns projected risk trajectory for a patient
	GetRiskProjection(ctx context.Context, patientID string, horizonDays int) (*contracts.RiskProjection, error)

	// GetInterventionCandidates returns patients who would benefit from intervention
	GetInterventionCandidates(ctx context.Context, interventionType string, limit int) ([]contracts.InterventionCandidate, error)
}

// KB11HTTPClient implements KB11Client via HTTP REST calls.
type KB11HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// Request/Response types for KB-11 API
type kb11RiskStrataRequest struct {
	PatientID string `json:"patient_id"`
}

type kb11RiskStrataResponse struct {
	PatientID       string              `json:"patient_id"`
	OverallRisk     string              `json:"overall_risk"` // "low", "moderate", "high", "very_high"
	RiskScore       float64             `json:"risk_score"`
	DomainRisks     map[string]float64  `json:"domain_risks"` // cardiovascular, metabolic, etc.
	RiskFactors     []string            `json:"risk_factors"`
	LastUpdated     time.Time           `json:"last_updated"`
}

type kb11CohortRequest struct {
	CohortID string `json:"cohort_id"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

type kb11CohortResponse struct {
	CohortID     string   `json:"cohort_id"`
	CohortName   string   `json:"cohort_name"`
	PatientIDs   []string `json:"patient_ids"`
	TotalCount   int      `json:"total_count"`
}

type kb11PopulationMetricsResponse struct {
	PopulationID    string            `json:"population_id"`
	TotalPatients   int               `json:"total_patients"`
	RiskDistribution map[string]int   `json:"risk_distribution"`
	TopConditions   []ConditionCount  `json:"top_conditions"`
	TopMedications  []MedicationCount `json:"top_medications"`
	AverageAge      float64           `json:"average_age"`
	LastUpdated     time.Time         `json:"last_updated"`
}

type kb11RiskProjectionRequest struct {
	PatientID   string `json:"patient_id"`
	HorizonDays int    `json:"horizon_days"`
}

type kb11RiskProjectionResponse struct {
	PatientID         string    `json:"patient_id"`
	CurrentRisk       float64   `json:"current_risk"`
	ProjectedRisk     float64   `json:"projected_risk"`
	Trajectory        string    `json:"trajectory"` // "improving", "stable", "worsening"
	KeyDrivers        []string  `json:"key_drivers"`
	RecommendedActions []string `json:"recommended_actions"`
}

type kb11InterventionCandidatesRequest struct {
	InterventionType string `json:"intervention_type"`
	Limit            int    `json:"limit"`
}

type kb11InterventionCandidate struct {
	PatientID        string  `json:"patient_id"`
	PriorityScore    float64 `json:"priority_score"`
	ExpectedBenefit  string  `json:"expected_benefit"`
	Barriers         []string `json:"barriers"`
}
```

**Implementation Methods**:
- `NewKB11HTTPClient(baseURL string) *KB11HTTPClient`
- `NewKB11HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB11HTTPClient`
- `GetPatientRiskStrata(ctx, patientID) (*RiskStratification, error)`
- `GetCohortMembers(ctx, cohortID, limit) ([]string, error)`
- `GetPopulationMetrics(ctx, populationID) (*PopulationMetrics, error)`
- `GetRiskProjection(ctx, patientID, horizonDays) (*RiskProjection, error)`
- `GetInterventionCandidates(ctx, interventionType, limit) ([]InterventionCandidate, error)`
- `HealthCheck(ctx) error`

**Estimated Lines**: ~450 lines

---

#### 1.2.2 kb17_http_client.go (Population Registry)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb17_http_client.go`

**Interface Definition**:
```go
// KB17Client defines the interface for Population Registry service interactions.
type KB17Client interface {
	// RegisterPatient adds/updates patient in population registry
	RegisterPatient(ctx context.Context, patient contracts.PatientRegistration) error

	// GetRegisteredPatients returns patients matching criteria
	GetRegisteredPatients(ctx context.Context, criteria contracts.RegistryCriteria) ([]contracts.RegisteredPatient, error)

	// GetPatientPrograms returns programs a patient is enrolled in
	GetPatientPrograms(ctx context.Context, patientID string) ([]contracts.ProgramEnrollment, error)

	// GetProgramEligibility determines if patient is eligible for program
	GetProgramEligibility(ctx context.Context, patientID string, programID string) (*contracts.EligibilityResult, error)

	// GetPatientTimeline returns longitudinal patient data
	GetPatientTimeline(ctx context.Context, patientID string, startDate, endDate time.Time) (*contracts.PatientTimeline, error)
}
```

**Estimated Lines**: ~400 lines

---

### 1.3 KB-16 Snapshot Integration (Lab Interpretation)

**CRITICAL**: KB-16 is a SNAPSHOT KB - must be added to KnowledgeSnapshotBuilder

#### 1.3.1 kb16_http_client.go (Lab Interpretation - SNAPSHOT)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb16_http_client.go`

**Architecture Note**: KB-16 provides reference ranges and lab interpretations that CQL consumes.
The snapshot pattern means lab reference data is pre-computed at build time.

**Interface Definition**:
```go
// Package clients provides HTTP clients for KB services.
//
// KB16HTTPClient implements the KB16Client interface for KB-16 Lab Interpretation.
// It provides reference ranges, critical value thresholds, and panel interpretations.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// This client IS PART of KnowledgeSnapshotBuilder (Category A).
// Lab reference ranges are pre-computed - CQL evaluates against frozen lab data.
//
// Connects to: http://localhost:8096 (Docker: kb16-lab-interpretation)
package clients

import (
	"context"
	"time"
)

// KB16Client defines the interface for Lab Interpretation service interactions.
// SNAPSHOT KB - used at build time to produce LabInterpretationSnapshot.
type KB16Client interface {
	// InterpretLabResult interprets a single lab result
	InterpretLabResult(ctx context.Context, labResult contracts.LabResult, patientContext contracts.PatientDemographics) (*contracts.LabInterpretation, error)

	// GetReferenceRange returns age/sex appropriate reference range
	GetReferenceRange(ctx context.Context, loincCode string, patientContext contracts.PatientDemographics) (*contracts.ReferenceRange, error)

	// InterpretLabPanel interprets a panel of related labs
	InterpretLabPanel(ctx context.Context, panelType string, labs []contracts.LabResult, patientContext contracts.PatientDemographics) (*contracts.PanelInterpretation, error)

	// GetCriticalValues returns critical value thresholds
	GetCriticalValues(ctx context.Context, loincCode string) (*contracts.CriticalValues, error)

	// GetReferenceRangesForPatient returns all applicable reference ranges
	// Used at snapshot build time to pre-compute all lab thresholds
	GetReferenceRangesForPatient(ctx context.Context, patientContext contracts.PatientDemographics, loincCodes []string) (map[string]*contracts.ReferenceRange, error)
}
```

**Estimated Lines**: ~400 lines

---

#### 1.3.2 Update KnowledgeSnapshotBuilder

**Location**: `vaidshala/clinical-runtime-platform/builders/knowledge_snapshot.go`

**Required Changes**:
```go
// Add KB-16 client to KnowledgeSnapshotBuilder struct
type KnowledgeSnapshotBuilder struct {
	// Existing clients...
	kb7Client     KB7Client
	kb7FHIRClient KB7FHIRClient
	kb8Client     KB8Client
	kb4Client     KB4Client
	kb5Client     KB5Client
	kb6Client     KB6Client
	kb1Client     KB1Client
	kb11Client    KB11Client
	kb16Client    KB16Client // ◄── ADD: Lab Interpretation
	// ...
}

// Add LabInterpretationSnapshot to ClinicalExecutionContext
type LabInterpretationSnapshot struct {
	// Pre-computed reference ranges keyed by LOINC code
	ReferenceRanges map[string]*contracts.ReferenceRange

	// Critical value thresholds
	CriticalValues map[string]*contracts.CriticalValues

	// Lab interpretations for recent results
	Interpretations map[string]*contracts.LabInterpretation

	// Panel interpretations (e.g., BMP, CMP, Lipid)
	PanelInterpretations map[string]*contracts.PanelInterpretation

	// Metadata
	BuildTimestamp time.Time
	PatientID      string
}
```

---

### 1.4 Phase 1 Deliverables Summary

| Deliverable | Type | Location | Lines | Status | Category |
|-------------|------|----------|-------|--------|----------|
| SafetyCommon.cql | CQL | tier-3-domain-commons/safety/ | ~350 | Required | CQL |
| CardiovascularCommon.cql | CQL | tier-3-domain-commons/cardiometabolic/ | ~400 | Required | CQL |
| DiabetesCommon.cql | CQL | tier-3-domain-commons/cardiometabolic/ | ~350 | Required | CQL |
| RenalCommon.cql | CQL | tier-3-domain-commons/cardiometabolic/ | ~300 | Required | CQL |
| MaternalHealthCommon.cql | CQL | tier-3-domain-commons/maternal/ | ~250 | Required | CQL |
| kb11_http_client.go | Go | clients/ | ~450 | Required | SNAPSHOT |
| kb17_http_client.go | Go | clients/ | ~400 | Required | SNAPSHOT |
| **kb16_http_client.go** | **Go** | **clients/** | **~400** | **Required** | **SNAPSHOT** |
| **KnowledgeSnapshotBuilder update** | **Go** | **builders/** | **~100** | **Required** | **SNAPSHOT** |

**Total Phase 1 Estimated Lines**: ~3,000 lines

---

## Phase 2: Integration Layer (Week 2-3)

### Objective
Create clinical utility wrappers and critical governance/rules clients to enable KB-8 calculator consumption in CQL and connect governance workflow.

### 2.1 Tier 1.5: Clinical Utilities CQL

#### 2.1.1 ClinicalCalculators.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-1.5-utilities/ClinicalCalculators.cql`

**Purpose**: Wrap KB-8 Calculator outputs for CQL consumption

**Specification**:
```cql
/*
 * ClinicalCalculators CQL Library
 * ============================================================
 * Tier 1.5 Clinical Utilities - Calculator Integration
 *
 * Purpose: Provides CQL interfaces to KB-8 Calculator outputs
 *          These are WRAPPERS around pre-computed values, NOT calculations
 *
 * ARCHITECTURE NOTE:
 * KB-8 Calculator performs the actual calculations at snapshot build time.
 * This library provides CQL definitions that consume those pre-computed values
 * from the ClinicalExecutionContext.CalculatorSnapshot.
 *
 * Governance: Tier-1.5 review (Clinical + Technical)
 * ============================================================
 */
library ClinicalCalculators version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations

// ============================================================
// VALUE SET REFERENCES
// ============================================================

valueset "Serum Creatinine": 'http://cardiofit.io/fhir/ValueSet/serum-creatinine'
valueset "Body Weight": 'http://cardiofit.io/fhir/ValueSet/body-weight'
valueset "Body Height": 'http://cardiofit.io/fhir/ValueSet/body-height'
valueset "Bilirubin Total": 'http://cardiofit.io/fhir/ValueSet/bilirubin-total'
valueset "INR": 'http://cardiofit.io/fhir/ValueSet/inr'
valueset "Albumin": 'http://cardiofit.io/fhir/ValueSet/albumin'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// RAW OBSERVATION VALUES (Input to Calculators)
// ============================================================

define "Most Recent Serum Creatinine":
  Observations.MostRecentObservation(
    [Observation: "Serum Creatinine"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Serum Creatinine Value":
  Observations.ObservationQuantityValue("Most Recent Serum Creatinine")

define "Most Recent Weight":
  Observations.MostRecentObservation(
    [Observation: "Body Weight"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Weight Kg":
  Observations.ObservationQuantityValue("Most Recent Weight")

define "Most Recent Height":
  Observations.MostRecentObservation(
    [Observation: "Body Height"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Height Cm":
  Observations.ObservationQuantityValue("Most Recent Height")

// ============================================================
// ESTIMATED GFR (eGFR) - From KB-8 Snapshot
// ============================================================

/*
 * NOTE: eGFR calculation uses CKD-EPI 2021 formula which requires:
 * - Serum creatinine
 * - Age
 * - Sex (no longer race-adjusted per 2021 guidelines)
 *
 * The actual calculation is performed by KB-8 Calculator Service.
 * This definition provides the CQL interface to consume that value.
 *
 * CQL cannot perform the CKD-EPI formula directly (requires natural log).
 * The snapshot contains pre-computed eGFR from KB-8.
 */

/*
 * eGFR Value from Snapshot (pre-computed by KB-8)
 * Returns: Decimal representing mL/min/1.73m²
 *
 * USAGE: Access via ClinicalExecutionContext.CalculatorSnapshot.EGFR
 * This define serves as documentation and type definition.
 */
define "eGFR Value":
  // Actual value comes from CalculatorSnapshot.EGFR
  // This placeholder returns null - runtime injects actual value
  null as Decimal

/*
 * CKD Stage Classification based on eGFR
 * This CAN be computed in CQL once we have the eGFR value
 */
define "CKD Stage from eGFR":
  case
    when "eGFR Value" is null then null
    when "eGFR Value" >= 90 then 'Stage 1'
    when "eGFR Value" >= 60 then 'Stage 2'
    when "eGFR Value" >= 45 then 'Stage 3a'
    when "eGFR Value" >= 30 then 'Stage 3b'
    when "eGFR Value" >= 15 then 'Stage 4'
    else 'Stage 5'
  end

define "Has Severe Renal Impairment":
  "eGFR Value" < 30

define "Has Moderate Renal Impairment":
  "eGFR Value" >= 30 and "eGFR Value" < 60

define "Has Mild Renal Impairment":
  "eGFR Value" >= 60 and "eGFR Value" < 90

// ============================================================
// BODY MASS INDEX (BMI) - Can be computed in CQL
// ============================================================

/*
 * BMI can be calculated directly in CQL: weight(kg) / height(m)²
 */
define "BMI Value":
  if "Most Recent Weight Kg" is null or "Most Recent Height Cm" is null then null
  else "Most Recent Weight Kg" / (("Most Recent Height Cm" / 100) * ("Most Recent Height Cm" / 100))

define "BMI Category":
  case
    when "BMI Value" is null then 'Unknown'
    when "BMI Value" < 18.5 then 'Underweight'
    when "BMI Value" < 25 then 'Normal'
    when "BMI Value" < 30 then 'Overweight'
    when "BMI Value" < 35 then 'Obese Class I'
    when "BMI Value" < 40 then 'Obese Class II'
    else 'Obese Class III'
  end

/*
 * Asian BMI thresholds (WHO Asia-Pacific)
 * Used for Tier 5 India adapter
 */
define "BMI Category Asian":
  case
    when "BMI Value" is null then 'Unknown'
    when "BMI Value" < 18.5 then 'Underweight'
    when "BMI Value" < 23 then 'Normal'
    when "BMI Value" < 25 then 'Overweight'
    when "BMI Value" < 30 then 'Obese Class I'
    else 'Obese Class II+'
  end

// ============================================================
// BODY SURFACE AREA (BSA) - From KB-8 Snapshot
// ============================================================

/*
 * BSA uses Mosteller formula: sqrt((height_cm * weight_kg) / 3600)
 * Requires square root which CQL doesn't natively support.
 * Pre-computed by KB-8.
 */
define "BSA Value":
  // From CalculatorSnapshot.BSA
  null as Decimal

// ============================================================
// CHILD-PUGH SCORE - From KB-8 Snapshot
// ============================================================

/*
 * Child-Pugh Score components:
 * - Total Bilirubin
 * - Serum Albumin
 * - INR
 * - Ascites (clinical assessment)
 * - Hepatic Encephalopathy (clinical assessment)
 *
 * Pre-computed by KB-8 based on available lab values + clinical inputs.
 */
define "Child-Pugh Score":
  // From CalculatorSnapshot.ChildPughScore
  null as Integer

define "Child-Pugh Class":
  case
    when "Child-Pugh Score" is null then null
    when "Child-Pugh Score" <= 6 then 'A'
    when "Child-Pugh Score" <= 9 then 'B'
    else 'C'
  end

define "Has Severe Hepatic Impairment":
  "Child-Pugh Class" = 'C'

define "Has Moderate Hepatic Impairment":
  "Child-Pugh Class" = 'B'

// ============================================================
// MELD SCORE - From KB-8 Snapshot
// ============================================================

/*
 * MELD Score = 3.78×ln(bilirubin) + 11.2×ln(INR) + 9.57×ln(creatinine) + 6.43
 * Requires natural logarithm - pre-computed by KB-8.
 */
define "MELD Score":
  // From CalculatorSnapshot.MELDScore
  null as Integer

define "MELD Mortality Risk":
  case
    when "MELD Score" is null then 'Unknown'
    when "MELD Score" < 10 then 'Low (< 2%)'
    when "MELD Score" < 20 then 'Moderate (6%)'
    when "MELD Score" < 30 then 'High (20%)'
    when "MELD Score" < 40 then 'Very High (53%)'
    else 'Severe (71%)'
  end

// ============================================================
// SOFA SCORE (ICU) - From KB-8 Snapshot
// ============================================================

/*
 * Sequential Organ Failure Assessment (SOFA)
 * Components: Respiration, Coagulation, Liver, Cardiovascular, CNS, Renal
 * Pre-computed by KB-8 with clinical inputs.
 */
define "SOFA Score":
  // From CalculatorSnapshot.SOFAScore
  null as Integer

define "Has Organ Dysfunction":
  "SOFA Score" >= 2

define "SOFA Mortality Risk":
  case
    when "SOFA Score" is null then 'Unknown'
    when "SOFA Score" < 2 then '< 10%'
    when "SOFA Score" < 4 then '< 10%'
    when "SOFA Score" < 6 then '15-20%'
    when "SOFA Score" < 8 then '40-50%'
    when "SOFA Score" < 10 then '50-60%'
    else '> 80%'
  end

// ============================================================
// qSOFA SCORE - Can be computed in CQL
// ============================================================

/*
 * Quick SOFA can be computed in CQL (simple criteria count)
 * Components:
 * - Respiratory rate ≥ 22
 * - Altered mentation (GCS < 15)
 * - Systolic BP ≤ 100
 */
define "Respiratory Rate High":
  Observations.ObservationQuantityValue(
    Observations.MostRecentObservation([Observation: code in {'9279-1'}])
  ) >= 22

define "Systolic BP Low":
  Observations.ObservationQuantityValue(
    Observations.MostRecentObservation([Observation: code in {'8480-6'}])
  ) <= 100

define "GCS Below Normal":
  Observations.ObservationQuantityValue(
    Observations.MostRecentObservation([Observation: code in {'9269-2'}])
  ) < 15

define "qSOFA Score":
  (if "Respiratory Rate High" then 1 else 0) +
  (if "Systolic BP Low" then 1 else 0) +
  (if "GCS Below Normal" then 1 else 0)

define "qSOFA Positive":
  "qSOFA Score" >= 2

// ============================================================
// ASCVD RISK SCORE - From KB-8 Snapshot
// ============================================================

/*
 * 10-year ASCVD Risk (Pooled Cohort Equations)
 * Requires exponential calculations - pre-computed by KB-8.
 */
define "ASCVD 10 Year Risk":
  // From CalculatorSnapshot.ASCVD10YearRisk
  null as Decimal

define "ASCVD Risk Category":
  case
    when "ASCVD 10 Year Risk" is null then 'Unknown'
    when "ASCVD 10 Year Risk" < 5.0 then 'Low'
    when "ASCVD 10 Year Risk" < 7.5 then 'Borderline'
    when "ASCVD 10 Year Risk" < 20.0 then 'Intermediate'
    else 'High'
  end

define "Statin Recommended by ASCVD Risk":
  "ASCVD 10 Year Risk" >= 7.5

// ============================================================
// CHA₂DS₂-VASc SCORE - Can be partially computed in CQL
// ============================================================

/*
 * CHA₂DS₂-VASc for AFib stroke risk
 * Some components require clinical assessment (prior stroke/TIA)
 * Full calculation from KB-8.
 */
define "CHA2DS2VASc Score":
  // From CalculatorSnapshot.CHA2DS2VAScScore
  null as Integer

define "Anticoagulation Recommended by CHA2DS2VASc":
  "CHA2DS2VASc Score" >= 2

define "Annual Stroke Risk CHA2DS2VASc":
  case
    when "CHA2DS2VASc Score" = 0 then 0.0
    when "CHA2DS2VASc Score" = 1 then 1.3
    when "CHA2DS2VASc Score" = 2 then 2.2
    when "CHA2DS2VASc Score" = 3 then 3.2
    when "CHA2DS2VASc Score" = 4 then 4.0
    when "CHA2DS2VASc Score" = 5 then 6.7
    when "CHA2DS2VASc Score" = 6 then 9.8
    when "CHA2DS2VASc Score" = 7 then 9.6
    when "CHA2DS2VASc Score" = 8 then 6.7
    when "CHA2DS2VASc Score" = 9 then 15.2
    else null
  end

// ============================================================
// HAS-BLED SCORE - From KB-8 Snapshot
// ============================================================

define "HAS-BLED Score":
  // From CalculatorSnapshot.HASBLEDScore
  null as Integer

define "High Bleeding Risk":
  "HAS-BLED Score" >= 3
```

**Estimated Lines**: ~400 lines

---

#### 2.1.2 LabReferenceRanges.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-1.5-utilities/LabReferenceRanges.cql`

**Purpose**: Integrate KB-16 Lab Interpretation outputs

**Key Definitions**:
- Age-stratified reference ranges (pediatric, adult, geriatric)
- Gender-stratified reference ranges
- Critical value flags (`"Is Critical High"`, `"Is Critical Low"`)
- Pregnancy-adjusted ranges
- `"Lab Abnormality Level"` (normal, low, high, critical low, critical high)

**Estimated Lines**: ~350 lines

---

### 2.2 KB Client Implementation

#### 2.2.1 kb10_http_client.go (Rules Engine)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb10_http_client.go`

**Interface Definition**:
```go
// KB10Client defines the interface for Rules Engine service interactions.
type KB10Client interface {
	// EvaluateRules executes rules for a clinical context
	EvaluateRules(ctx context.Context, ruleSetID string, facts map[string]interface{}) (*contracts.RuleEvaluationResult, error)

	// GetActiveAlerts returns currently active alerts for a patient
	GetActiveAlerts(ctx context.Context, patientID string) ([]contracts.ClinicalAlert, error)

	// AcknowledgeAlert marks an alert as acknowledged
	AcknowledgeAlert(ctx context.Context, alertID string, acknowledgerID string, notes string) error

	// GetRuleDefinitions returns available rule sets
	GetRuleDefinitions(ctx context.Context) ([]contracts.RuleSetDefinition, error)
}
```

**Estimated Lines**: ~350 lines

---

#### 2.2.2 kb18_http_client.go (Governance Engine)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb18_http_client.go`

**Interface Definition**:
```go
// KB18Client defines the interface for Governance Engine service interactions.
// CRITICAL: This client RECEIVES classifications from CQL and ENFORCES approval workflows.
type KB18Client interface {
	// ClassifyApprovalLevel determines approval requirements for an action
	// INPUT: CQL classification output from GovernanceClassifier.cql
	ClassifyApprovalLevel(ctx context.Context, action contracts.ClinicalAction, classification contracts.GovernanceClassification) (*contracts.ApprovalRequirement, error)

	// SubmitForApproval creates an approval request
	SubmitForApproval(ctx context.Context, request contracts.ApprovalRequest) (*contracts.ApprovalSubmission, error)

	// GetPendingApprovals returns approvals awaiting decision
	GetPendingApprovals(ctx context.Context, approverID string) ([]contracts.PendingApproval, error)

	// RecordDecision records an approval decision with audit trail
	RecordDecision(ctx context.Context, approvalID string, decision contracts.ApprovalDecision) (*contracts.AuditRecord, error)

	// GetGovernancePolicy returns current governance policy
	GetGovernancePolicy(ctx context.Context, policyID string) (*contracts.GovernancePolicy, error)
}
```

**Estimated Lines**: ~450 lines

---

#### 2.2.3 RuntimeClients Structure (CRITICAL ARCHITECTURE)

**Location**: `vaidshala/clinical-runtime-platform/clients/runtime_clients.go`

**Architecture Note**: This structure holds clients for Runtime KBs (Category B) that are called
during workflow execution, NOT during snapshot build time.

**Implementation**:
```go
// Package clients provides HTTP clients for KB services.
//
// RuntimeClients holds clients for KB services that are called during execution.
// These are WORKFLOW clients, not SNAPSHOT clients.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// Category B KBs provide ACTIONS and WORKFLOWS at execution time.
// They do NOT provide data to CQL - they consume CQL outputs.
//
// Workflow Pattern:
// 1. CQL evaluates against frozen snapshot → produces classifications
// 2. RuntimeClients consume classifications → trigger workflows
// 3. Example: CQL says "LEVEL_5 approval needed" → KB-18 routes to Medical Director
package clients

import (
	"net/http"
	"time"
)

// RuntimeClients holds all Runtime KB client instances.
// These clients are called DURING workflow execution (not snapshot build).
type RuntimeClients struct {
	// ICU Intelligence (Tier 7 - MANDATORY VETO CHECK)
	// MUST be called BEFORE any workflow KB!
	ICU *ICUIntelligenceClient // Safety veto layer (Go, not a KB)

	// Governance & Workflow
	KB3  *KB3HTTPClient  // Guidelines (workflow recommendations)
	KB10 *KB10HTTPClient // Rules Engine (execute rules)
	KB18 *KB18HTTPClient // Governance (approval workflows)
	KB19 *KB19HTTPClient // Protocol Orchestration (execute protocols)

	// Care Management
	KB9  *KB9HTTPClient  // Care Gaps (workflow triggers)
	KB12 *KB12HTTPClient // OrderSets/CarePlans (execution)
	KB13 *KB13HTTPClient // Quality Measures (workflow reporting)
	KB14 *KB14HTTPClient // Care Navigator (workflow navigation)

	// Evidence & Registry
	KB15 *KB15HTTPClient // Evidence Engine (GRADE grading) ◄── ADDED per CTO review
	KB17 *KB17HTTPClient // Population Registry (cohort management)
}

// ICUIntelligenceClient is NOT a KB client - it's the Tier 7 safety veto layer.
// ARCHITECTURE NOTE: This must be called BEFORE any RuntimeClient KB.
type ICUIntelligenceClient interface {
	// Evaluate checks if proposed action should be vetoed
	// Returns VetoResult with Vetoed=true if action is unsafe
	Evaluate(ctx context.Context, action contracts.ProposedAction, facts contracts.SafetyFacts) (*contracts.VetoResult, error)
}

// RuntimeClientConfig holds configuration for Runtime KB clients.
type RuntimeClientConfig struct {
	// Base URLs for Runtime KB services
	KB3BaseURL  string // Guidelines (default: http://localhost:8087)
	KB9BaseURL  string // Care Gaps (default: http://localhost:8089)
	KB10BaseURL string // Rules Engine (default: http://localhost:8090)
	KB12BaseURL string // OrderSets (default: http://localhost:8092)
	KB13BaseURL string // Quality (default: http://localhost:8093)
	KB14BaseURL string // Care Navigator (default: http://localhost:8094)
	KB15BaseURL string // Evidence Engine (default: http://localhost:8095) ◄── ADDED
	KB17BaseURL string // Population Registry (default: http://localhost:8097) ◄── ADDED
	KB18BaseURL string // Governance (default: http://localhost:8098)
	KB19BaseURL string // Protocol (default: http://localhost:8099)

	// HTTP client settings
	Timeout time.Duration // Default: 30s
}

// DefaultRuntimeClientConfig returns default configuration.
func DefaultRuntimeClientConfig() RuntimeClientConfig {
	return RuntimeClientConfig{
		KB3BaseURL:  "http://localhost:8087",
		KB9BaseURL:  "http://localhost:8089",
		KB10BaseURL: "http://localhost:8090",
		KB12BaseURL: "http://localhost:8092",
		KB13BaseURL: "http://localhost:8093",
		KB14BaseURL: "http://localhost:8094",
		KB15BaseURL: "http://localhost:8095",
		KB17BaseURL: "http://localhost:8097",
		KB18BaseURL: "http://localhost:8098",
		KB19BaseURL: "http://localhost:8099",
		Timeout:     30 * time.Second,
	}
}

// NewRuntimeClients creates all Runtime KB clients from configuration.
func NewRuntimeClients(config RuntimeClientConfig, icuClient *ICUIntelligenceClient) *RuntimeClients {
	httpClient := &http.Client{Timeout: config.Timeout}

	return &RuntimeClients{
		ICU:  icuClient, // Injected - required dependency
		KB3:  NewKB3HTTPClientWithHTTP(config.KB3BaseURL, httpClient),
		KB9:  NewKB9HTTPClientWithHTTP(config.KB9BaseURL, httpClient),
		KB10: NewKB10HTTPClientWithHTTP(config.KB10BaseURL, httpClient),
		KB12: NewKB12HTTPClientWithHTTP(config.KB12BaseURL, httpClient),
		KB13: NewKB13HTTPClientWithHTTP(config.KB13BaseURL, httpClient),
		KB14: NewKB14HTTPClientWithHTTP(config.KB14BaseURL, httpClient),
		KB15: NewKB15HTTPClientWithHTTP(config.KB15BaseURL, httpClient),
		KB17: NewKB17HTTPClientWithHTTP(config.KB17BaseURL, httpClient),
		KB18: NewKB18HTTPClientWithHTTP(config.KB18BaseURL, httpClient),
		KB19: NewKB19HTTPClientWithHTTP(config.KB19BaseURL, httpClient),
	}
}
```

**Usage Pattern (CORRECTED - ICU Veto First)**:
```go
// CORRECT workflow execution pattern - ICU veto BEFORE any KB call
func ExecuteClinicalDecision(ctx context.Context, decision CQLOutput) error {
	runtime := NewRuntimeClients(DefaultRuntimeClientConfig(), icuClient)

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 1: ICU INTELLIGENCE VETO CHECK (MANDATORY - CANNOT BE SKIPPED)
	// ═══════════════════════════════════════════════════════════════════════
	vetoResult, err := runtime.ICU.Evaluate(ctx, decision.Action, decision.SafetyFacts)
	if err != nil {
		return fmt.Errorf("ICU veto check failed: %w", err)
	}
	if vetoResult.Vetoed {
		// HARD STOP - Action is blocked, record immutable audit trail
		audit.RecordVeto(ctx, audit.VetoRecord{
			Action:     decision.Action,
			Reason:     vetoResult.Reason,
			SafetyRule: vetoResult.TriggeringRule,
			Timestamp:  time.Now(),
		})
		return &VetoError{
			Reason: vetoResult.Reason,
			Rule:   vetoResult.TriggeringRule,
		}
	}

	// ═══════════════════════════════════════════════════════════════════════
	// STEP 2: GOVERNANCE CHECK (Only if not vetoed)
	// ═══════════════════════════════════════════════════════════════════════
	if decision.GovernanceClassification.RequiresApproval {
		approval, err := runtime.KB18.SubmitForApproval(ctx, ApprovalRequest{
			Level:          decision.ApprovalLevel,
			Action:         decision.Action,
			Classification: decision.GovernanceClassification,
		})
	return err
}
```

**Estimated Lines**: ~450 lines (updated to include ICU + KB-15/KB-17)

---

#### 2.2.4 kb15_http_client.go (Evidence Engine - RUNTIME)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb15_http_client.go`

**Architecture Note**: KB-15 is a RUNTIME KB - called during workflow to fetch evidence for recommendations.
Provides GRADE grading for KB-19 protocol recommendations.

**Interface Definition**:
```go
// KB15Client defines the interface for Evidence Engine service interactions.
// RUNTIME KB - provides evidence at recommendation time, NOT pre-computed.
type KB15Client interface {
	// SearchEvidence searches for clinical evidence based on query criteria
	SearchEvidence(ctx context.Context, query contracts.EvidenceQuery) (*contracts.EvidenceResult, error)

	// GetEvidenceGrading returns GRADE assessment for evidence
	// (High, Moderate, Low, Very Low)
	GetEvidenceGrading(ctx context.Context, evidenceID string) (*contracts.GRADEAssessment, error)

	// GetEvidenceEnvelope returns pre-packaged evidence for protocol decision node
	// Used by KB-19 to attach evidence to recommendations
	GetEvidenceEnvelope(ctx context.Context, protocolID string, decisionNodeID string) (*contracts.EvidenceEnvelope, error)

	// GenerateCitation generates formatted citation for evidence
	GenerateCitation(ctx context.Context, evidenceID string, style string) (string, error)

	// GetSystematicReviews returns relevant systematic reviews for condition
	GetSystematicReviews(ctx context.Context, conditionCode string) ([]contracts.SystematicReview, error)
}
```

**Estimated Lines**: ~350 lines

---

### 2.3 Phase 2 Deliverables Summary

| Deliverable | Type | Location | Lines | Status | Category |
|-------------|------|----------|-------|--------|----------|
| ClinicalCalculators.cql | CQL | tier-1.5-utilities/ | ~400 | Required | CQL |
| LabReferenceRanges.cql | CQL | tier-1.5-utilities/ | ~350 | Required | CQL |
| kb10_http_client.go | Go | clients/ | ~350 | Required | RUNTIME |
| kb18_http_client.go | Go | clients/ | ~450 | Required | RUNTIME |
| **kb15_http_client.go** | **Go** | **clients/** | **~350** | **Required** | **RUNTIME** |
| **runtime_clients.go** | **Go** | **clients/** | **~450** | **Required** | **RUNTIME** |

**Total Phase 2 Estimated Lines**: ~2,350 lines

> **Notes**:
> - KB-16 moved to Phase 1 (SNAPSHOT category)
> - KB-15 added per CTO/CMO review (Evidence Engine for GRADE grading)
> - RuntimeClients now includes ICU Intelligence integration point

---

## Phase 3: Orchestration Layer (Week 3-4)

### Objective
Create the classification layer for governance, alerts, and protocol orchestration.

### 3.1 Tier 6a: Orchestration CQL

#### 3.1.1 GovernanceClassifier.cql (CRITICAL)

**Location**: `vaidshala/clinical-knowledge-core/tier-6a-orchestration/GovernanceClassifier.cql`

**Purpose**: CLASSIFY approval requirements - KB-18 ENFORCES

**Specification**:
```cql
/*
 * GovernanceClassifier CQL Library
 * ============================================================
 * Tier 6a Orchestration - Governance Classification
 *
 * PURPOSE: CLASSIFICATION ONLY - NOT ENFORCEMENT
 * This library determines WHAT level of approval an action requires.
 * KB-18 Governance Engine ENFORCES the actual approval workflow.
 *
 * ARCHITECTURE CRITICAL:
 * - CQL OUTPUT: "This action requires Medical Director approval"
 * - CQL DOES NOT: Actually route to Medical Director (that's KB-18)
 *
 * Governance: Tier-6a review (Clinical + Technical + Compliance)
 * ============================================================
 */
library GovernanceClassifier version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include SafetyCommon version '1.0.0' called Safety
include ClinicalCalculators version '1.0.0' called Calculators

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// APPROVAL LEVEL CODES
// ============================================================

/*
 * Approval Levels (consumed by KB-18):
 * LEVEL_1: Auto-approved (no human required)
 * LEVEL_2: RN/Pharmacist approval
 * LEVEL_3: Attending Physician approval
 * LEVEL_4: Department Head approval
 * LEVEL_5: Medical Director approval (IRB for research)
 * LEVEL_6: C-Suite/Board approval (institutional risk)
 */

// ============================================================
// MEDICATION ORDER CLASSIFICATION
// ============================================================

/*
 * Classify medication order approval requirements
 * OUTPUT: Approval level string for KB-18 consumption
 */
define "Medication Approval Level":
  case
    // LEVEL 6: Institutional risk medications
    when "Is Experimental Drug" then 'LEVEL_6'
    when "Is Compassionate Use" then 'LEVEL_5'

    // LEVEL 5: Medical Director review
    when "Is Controlled Substance Schedule II" and "Exceeds Standard Duration" then 'LEVEL_5'
    when "Is High Cost Medication" and "No Prior Authorization" then 'LEVEL_5'

    // LEVEL 4: Department Head
    when "Is Off Label Use" and "No Evidence Support" then 'LEVEL_4'
    when "Exceeds Formulary Limits" then 'LEVEL_4'

    // LEVEL 3: Attending Physician
    when Safety."Has Active High Alert Medication" then 'LEVEL_3'
    when "Has Drug Interaction Warning" then 'LEVEL_3'
    when "Requires Renal Dose Adjustment" and not "Adjustment Applied" then 'LEVEL_3'

    // LEVEL 2: RN/Pharmacist
    when "Is PRN Medication" and "First Administration" then 'LEVEL_2'
    when "Requires Double Check" then 'LEVEL_2'

    // LEVEL 1: Auto-approved
    else 'LEVEL_1'
  end

/*
 * Supporting classifications
 */
define "Is Experimental Drug":
  exists [MedicationRequest] MR
    where MR.category ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'experimental' } } }
    })

define "Is Compassionate Use":
  exists [MedicationRequest] MR
    where MR.category ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'compassionate-use' } } }
    })

define "Is Controlled Substance Schedule II":
  exists [MedicationRequest] MR
    where MR.medication.text ~ 'Schedule II'

define "Exceeds Standard Duration":
  // Check if prescription duration exceeds standard limits
  false // Placeholder - needs actual duration check

define "Is High Cost Medication":
  // From KB-6 Formulary - cost > threshold
  false // Placeholder

define "No Prior Authorization":
  // PA status from KB-6
  false // Placeholder

define "Is Off Label Use":
  // Indication not matching approved uses
  false // Placeholder

define "No Evidence Support":
  // KB-3 evidence level check
  false // Placeholder

define "Exceeds Formulary Limits":
  // KB-6 formulary restrictions
  false // Placeholder

define "Has Drug Interaction Warning":
  // KB-5 interaction check
  false // Placeholder

define "Requires Renal Dose Adjustment":
  Calculators."Has Severe Renal Impairment" or Calculators."Has Moderate Renal Impairment"

define "Adjustment Applied":
  // Check if dose was already adjusted
  false // Placeholder

define "Is PRN Medication":
  exists [MedicationRequest] MR
    where MR.dosageInstruction.asNeeded is not null

define "First Administration":
  // Check administration history
  false // Placeholder

define "Requires Double Check":
  Safety."Has Active High Alert Medication"

// ============================================================
// PROCEDURE ORDER CLASSIFICATION
// ============================================================

define "Procedure Approval Level":
  case
    when "Is Invasive Procedure" and "High Risk Patient" then 'LEVEL_4'
    when "Is Sedation Required" then 'LEVEL_3'
    when "Is Radiation Exposure" then 'LEVEL_3'
    when "Is Contrast Required" and Safety."Has Contrast Allergy" then 'LEVEL_4'
    else 'LEVEL_2'
  end

define "Is Invasive Procedure":
  // Procedure category check
  false // Placeholder

define "High Risk Patient":
  Safety."Safety Risk Level" in { 'HIGH', 'CRITICAL' }

define "Is Sedation Required":
  // Procedure sedation flag
  false // Placeholder

define "Is Radiation Exposure":
  // Imaging modality check
  false // Placeholder

define "Is Contrast Required":
  // Contrast administration flag
  false // Placeholder

// ============================================================
// DISCHARGE CLASSIFICATION
// ============================================================

define "Discharge Approval Level":
  case
    when "Against Medical Advice" then 'LEVEL_4'
    when "High Risk Discharge" then 'LEVEL_3'
    when "Incomplete Follow Up Plan" then 'LEVEL_3'
    else 'LEVEL_2'
  end

define "Against Medical Advice":
  // AMA discharge flag
  false // Placeholder

define "High Risk Discharge":
  Safety."Safety Risk Level" = 'CRITICAL'

define "Incomplete Follow Up Plan":
  // Follow up orders check
  false // Placeholder

// ============================================================
// OVERRIDE CLASSIFICATION
// ============================================================

/*
 * When a clinical decision support alert is overridden
 */
define "Override Approval Level":
  case
    when Safety."Safety Risk Level" = 'CRITICAL' then 'LEVEL_5'
    when Safety."Safety Risk Level" = 'HIGH' then 'LEVEL_4'
    when Safety."Safety Risk Level" = 'MODERATE' then 'LEVEL_3'
    else 'LEVEL_2'
  end

// ============================================================
// GOVERNANCE METADATA OUTPUT
// ============================================================

/*
 * Complete governance classification for KB-18 consumption
 */
define "Governance Classification Output":
  Tuple {
    medicationLevel: "Medication Approval Level",
    procedureLevel: "Procedure Approval Level",
    dischargeLevel: "Discharge Approval Level",
    overrideLevel: "Override Approval Level",
    safetyRiskLevel: Safety."Safety Risk Level",
    requiresAudit: Safety."Safety Risk Level" in { 'HIGH', 'CRITICAL' },
    requiresWitness: Safety."On Multiple High Alert Categories"
  }
```

**Estimated Lines**: ~350 lines

---

#### 3.1.2 PopulationStratifier.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-6a-orchestration/PopulationStratifier.cql`

**Purpose**: Risk stratification for KB-11/KB-17 population health

**Key Definitions**:
- `"Overall Risk Stratum"` (low, moderate, high, very high)
- `"Cardiovascular Risk Stratum"`, `"Metabolic Risk Stratum"`, `"Safety Risk Stratum"`
- `"Intervention Priority Score"` (numeric for ranking)
- `"Eligible for Care Management"`, `"Eligible for Disease Management"`
- `"Rising Risk Indicator"` (trending toward higher risk)

**Estimated Lines**: ~300 lines

---

#### 3.1.3 AlertClassifier.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-6a-orchestration/AlertClassifier.cql`

**Purpose**: Alert priority classification for KB-10

**Key Definitions**:
- `"Alert Priority Level"` (informational, warning, urgent, critical)
- `"Alert Category"` (safety, interaction, gap, reminder)
- `"Requires Immediate Action"` (boolean for critical alerts)
- `"Alert Suppression Eligible"` (can be acknowledged without action)
- `"Alert Escalation Path"` (who receives escalated alerts)

**Estimated Lines**: ~250 lines

---

#### 3.1.4 CDSSOrchestrator.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-6a-orchestration/CDSSOrchestrator.cql`

**Purpose**: Protocol selection and conflict resolution for KB-19

**Key Definitions**:
- `"Applicable Protocols"` (list of protocols matching patient context)
- `"Protocol Conflicts"` (detected conflicts between protocols)
- `"Protocol Priority Ranking"` (ordered by applicability/evidence)
- `"Protocol Contraindications"` (protocols to exclude)
- `"Recommended Protocol"` (top-ranked applicable protocol)

**Estimated Lines**: ~400 lines

---

#### 3.1.5 ConflictArbitrator.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-6a-orchestration/ConflictArbitrator.cql`

**Purpose**: Multi-protocol conflict handling

**Key Definitions**:
- `"Conflicting Recommendations"` (detected conflicts)
- `"Arbitration Rule Applied"` (which rule resolved conflict)
- `"Conservative Recommendation"` (safest option)
- `"Aggressive Recommendation"` (most beneficial option)
- `"Arbiter Output"` (final recommendation with rationale)

**Estimated Lines**: ~300 lines

---

### 3.2 KB Client Implementation

#### 3.2.1 kb9_http_client.go (Care Gaps)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb9_http_client.go`

**Interface Definition**:
```go
// KB9Client defines the interface for Care Gaps service interactions.
type KB9Client interface {
	// GetActiveGaps returns open care gaps for a patient
	GetActiveGaps(ctx context.Context, patientID string) ([]contracts.CareGap, error)

	// GetMeasureStatus returns quality measure status for a patient
	GetMeasureStatus(ctx context.Context, patientID string, measureID string) (*contracts.MeasureStatus, error)

	// GetDueInterventions returns interventions due for a patient
	GetDueInterventions(ctx context.Context, patientID string) ([]contracts.DueIntervention, error)

	// CloseGap marks a care gap as closed
	CloseGap(ctx context.Context, gapID string, closureReason string, evidence contracts.GapEvidence) error

	// GetPopulationGaps returns aggregate gap analysis for a population
	GetPopulationGaps(ctx context.Context, populationID string) (*contracts.PopulationGapSummary, error)
}
```

**Estimated Lines**: ~400 lines

---

#### 3.2.2 kb13_http_client.go (Quality Measures)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb13_http_client.go`

**Interface Definition**:
```go
// KB13Client defines the interface for Quality Measures service interactions.
type KB13Client interface {
	// EvaluateMeasure evaluates a quality measure for a patient
	EvaluateMeasure(ctx context.Context, patientID string, measureID string) (*contracts.MeasureResult, error)

	// GetMeasureDefinition returns measure specification
	GetMeasureDefinition(ctx context.Context, measureID string) (*contracts.MeasureDefinition, error)

	// GetPerformanceRate returns aggregate performance for a measure
	GetPerformanceRate(ctx context.Context, populationID string, measureID string) (*contracts.PerformanceRate, error)

	// GetMeasureReport generates a measure report for reporting period
	GetMeasureReport(ctx context.Context, measureID string, reportingPeriod contracts.DateRange) (*contracts.MeasureReport, error)

	// BatchEvaluate evaluates multiple measures for a patient
	BatchEvaluate(ctx context.Context, patientID string, measureIDs []string) ([]contracts.MeasureResult, error)
}
```

**Estimated Lines**: ~450 lines

---

#### 3.2.3 kb14_http_client.go (Care Navigator)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb14_http_client.go`

**Interface Definition**:
```go
// KB14Client defines the interface for Care Navigator service interactions.
type KB14Client interface {
	// GetActiveWorkflows returns active workflows for a patient
	GetActiveWorkflows(ctx context.Context, patientID string) ([]contracts.Workflow, error)

	// StartWorkflow initiates a new care workflow
	StartWorkflow(ctx context.Context, patientID string, workflowType string, params map[string]interface{}) (*contracts.WorkflowInstance, error)

	// AdvanceWorkflow moves workflow to next step
	AdvanceWorkflow(ctx context.Context, workflowID string, stepOutput map[string]interface{}) (*contracts.WorkflowStep, error)

	// GetWorkflowStatus returns current workflow status
	GetWorkflowStatus(ctx context.Context, workflowID string) (*contracts.WorkflowStatus, error)

	// GetPendingTasks returns tasks awaiting action
	GetPendingTasks(ctx context.Context, assigneeID string) ([]contracts.WorkflowTask, error)

	// CompleteTask marks a task as completed
	CompleteTask(ctx context.Context, taskID string, outcome contracts.TaskOutcome) error
}
```

**Estimated Lines**: ~500 lines

---

### 3.3 Phase 3 Deliverables Summary

| Deliverable | Type | Location | Lines | Status |
|-------------|------|----------|-------|--------|
| GovernanceClassifier.cql | CQL | tier-6a-orchestration/ | ~350 | Required |
| PopulationStratifier.cql | CQL | tier-6a-orchestration/ | ~300 | Required |
| AlertClassifier.cql | CQL | tier-6a-orchestration/ | ~250 | Required |
| CDSSOrchestrator.cql | CQL | tier-6a-orchestration/ | ~400 | Required |
| ConflictArbitrator.cql | CQL | tier-6a-orchestration/ | ~300 | Required |
| kb9_http_client.go | Go | clients/ | ~400 | Required |
| kb13_http_client.go | Go | clients/ | ~450 | Required |
| kb14_http_client.go | Go | clients/ | ~500 | Required |

**Total Phase 3 Estimated Lines**: ~2,950 lines

---

## Phase 4: Regional & Guidelines (Week 5-6)

### Objective
Complete regional adapters and clinical guidelines to enable multi-region deployment and protocol-driven care.

### 4.1 Tier 5: Regional Adapters CQL

#### 4.1.1 IndiaThresholdAdapter.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-5-regional-adapters/IN/IndiaThresholdAdapter.cql`

**Key Definitions**:
- `"BMI Obesity Threshold India"` (≥23 vs ≥30 for Western)
- `"Diabetes Fasting Glucose Threshold India"` (126 mg/dL)
- `"Hypertension Stage 1 Threshold India"` (140/90 mmHg)
- `"Pediatric Age Threshold India"` (< 18 years)
- All thresholds sourced from ICMR guidelines

**Estimated Lines**: ~200 lines

---

#### 4.1.2 IndiaDrugAdapter.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-5-regional-adapters/IN/IndiaDrugAdapter.cql`

**Key Definitions**:
- `"Is NLEM Drug"` (National List of Essential Medicines)
- `"Is Scheduled Drug India"` (H1, X, etc.)
- `"Requires Prescription India"` (Schedule H)
- `"Available in India"` (market availability)

**Estimated Lines**: ~150 lines

---

#### 4.1.3 AustraliaThresholdAdapter.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-5-regional-adapters/AU/AustraliaThresholdAdapter.cql`

**Key Definitions**:
- `"PBS Eligible"` (Pharmaceutical Benefits Scheme)
- `"TGA Approved"` (Therapeutic Goods Administration)
- `"RACGP Guideline Threshold"` (Australian GP guidelines)
- All thresholds sourced from RACGP/TGA

**Estimated Lines**: ~200 lines

---

#### 4.1.4 AustraliaDrugAdapter.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-5-regional-adapters/AU/AustraliaDrugAdapter.cql`

**Key Definitions**:
- `"Is PBS Listed"` (PBS formulary membership)
- `"PBS Restriction Code"` (authority required, etc.)
- `"Is TGA Registered"` (market authorization)
- `"Requires Authority Script"` (authority prescription)

**Estimated Lines**: ~150 lines

---

### 4.2 Tier 4b: Clinical Guidelines CQL

#### 4.2.1 SepsisGuidelines.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-4-guidelines/clinical/SepsisGuidelines.cql`

**Purpose**: Surviving Sepsis Campaign 2021 implementation

**Key Definitions**:
- `"Meets Sepsis Criteria"` (qSOFA ≥ 2 + suspected infection)
- `"Meets Septic Shock Criteria"` (sepsis + vasopressors + lactate > 2)
- `"Hour-1 Bundle Applicable"` (initial resuscitation)
- `"Antimicrobial Delay Risk"` (time since sepsis onset)
- `"Fluid Resuscitation Target"` (30 mL/kg crystalloid)
- `"Vasopressor Indication"` (MAP < 65 despite fluids)

**Estimated Lines**: ~400 lines

---

#### 4.2.2 VTEProphylaxisGuidelines.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-4-guidelines/clinical/VTEProphylaxisGuidelines.cql`

**Purpose**: VTE prevention per CHEST guidelines

**Key Definitions**:
- `"VTE Risk Score"` (Padua/Caprini score)
- `"Bleeding Risk Score"` (IMPROVE score)
- `"Pharmacologic Prophylaxis Indicated"`
- `"Mechanical Prophylaxis Indicated"`
- `"Contraindication to Anticoagulation"`

**Estimated Lines**: ~350 lines

---

#### 4.2.3 HFGDMTGuidelines.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-4-guidelines/clinical/HFGDMTGuidelines.cql`

**Purpose**: Heart Failure GDMT per ACC/AHA 2022

**Key Definitions**:
- `"HF Stage"` (A, B, C, D)
- `"LVEF Category"` (HFrEF, HFmrEF, HFpEF)
- `"GDMT Medication Recommendations"`
- `"Target Dose Achievement"`
- `"Quadruple Therapy Eligible"`

**Estimated Lines**: ~400 lines

---

#### 4.2.4 T2DMGuidelines.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-4-guidelines/clinical/T2DMGuidelines.cql`

**Purpose**: Type 2 Diabetes per ADA Standards 2024

**Key Definitions**:
- `"Glycemic Target"` (individualized A1C goal)
- `"Metformin First Line Applicable"`
- `"SGLT2i/GLP1 Indicated for ASCVD"`
- `"Insulin Initiation Criteria"`
- `"Hypoglycemia Risk Level"`

**Estimated Lines**: ~400 lines

---

#### 4.2.5 AKIGuidelines.cql

**Location**: `vaidshala/clinical-knowledge-core/tier-4-guidelines/clinical/AKIGuidelines.cql`

**Purpose**: AKI per KDIGO 2012

**Key Definitions**:
- `"AKI Stage"` (1, 2, 3 per KDIGO)
- `"AKI Risk Factors"`
- `"Nephrotoxin Avoidance List"`
- `"Fluid Management Target"`
- `"RRT Indication"`

**Estimated Lines**: ~350 lines

---

### 4.3 KB Client Implementation

#### 4.3.1 kb3_http_client.go (Guidelines)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb3_http_client.go`

**Interface Definition**:
```go
// KB3Client defines the interface for Guidelines service interactions.
type KB3Client interface {
	// GetApplicableGuidelines returns guidelines matching patient context
	GetApplicableGuidelines(ctx context.Context, patientContext contracts.PatientContext) ([]contracts.Guideline, error)

	// GetGuidelineRecommendations returns specific recommendations from a guideline
	GetGuidelineRecommendations(ctx context.Context, guidelineID string, patientContext contracts.PatientContext) ([]contracts.Recommendation, error)

	// GetEvidenceLevel returns evidence supporting a recommendation
	GetEvidenceLevel(ctx context.Context, recommendationID string) (*contracts.EvidenceLevel, error)

	// GetGuidelineMetadata returns guideline source and version info
	GetGuidelineMetadata(ctx context.Context, guidelineID string) (*contracts.GuidelineMetadata, error)
}
```

**Estimated Lines**: ~400 lines

---

#### 4.3.2 kb12_http_client.go (Order Sets)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb12_http_client.go`

**Interface Definition**:
```go
// KB12Client defines the interface for Order Sets/Care Plans service interactions.
type KB12Client interface {
	// GetApplicableOrderSets returns order sets matching clinical context
	GetApplicableOrderSets(ctx context.Context, diagnosis string, setting string) ([]contracts.OrderSet, error)

	// GetOrderSetDetails returns full order set specification
	GetOrderSetDetails(ctx context.Context, orderSetID string) (*contracts.OrderSetDetail, error)

	// InstantiateOrderSet creates patient-specific orders from template
	InstantiateOrderSet(ctx context.Context, orderSetID string, patientID string, customizations map[string]interface{}) (*contracts.InstantiatedOrderSet, error)

	// GetCarePlanTemplates returns available care plan templates
	GetCarePlanTemplates(ctx context.Context, conditionCodes []string) ([]contracts.CarePlanTemplate, error)
}
```

**Estimated Lines**: ~400 lines

---

#### 4.3.3 kb19_http_client.go (Protocol Orchestrator)

**Location**: `vaidshala/clinical-runtime-platform/clients/kb19_http_client.go`

### 🔴 KB-19 SCOPE CONSTRAINTS (CTO/CMO Directive)

> **KB-19 MUST BE: "Guideline reasoning + explanation."**
>
> **KB-19 MUST NOT BE:**
> - A real-time dominance engine (that's ICU Intelligence)
> - A safety veto system (that's ICU Intelligence)
> - A crisis override mechanism (that's ICU Intelligence)
> - A workflow engine (that's KB-14)

**Authority Boundary**:
| KB-19 CAN | KB-19 CANNOT |
|-----------|--------------|
| Recommend medications | Veto ICU decisions |
| Suggest protocols | Override safety rules |
| Explain reasoning | Execute workflows directly |
| Attach evidence (KB-15) | Make real-time risk decisions |

**Interface Definition**:
```go
// KB19Client defines the interface for Protocol Orchestrator service interactions.
//
// ARCHITECTURE CONSTRAINT (CTO/CMO):
// KB-19 provides RECOMMENDATIONS and EXPLANATIONS only.
// It cannot veto, override, or make safety decisions.
// ICU Intelligence has authority over all KB-19 recommendations.
//
// KB-19 output is ALWAYS subject to:
// 1. ICU Dominance check (can be overridden)
// 2. Governance approval (KB-18)
// 3. Workflow execution (KB-14)
type KB19Client interface {
	// SelectProtocol chooses optimal protocol for clinical situation
	// OUTPUT: Recommendation (not execution)
	SelectProtocol(ctx context.Context, clinicalContext contracts.ClinicalContext) (*contracts.ProtocolSelection, error)

	// ExecuteProtocolStep executes next step in active protocol
	// CONSTRAINT: Must be called AFTER ICU veto check
	ExecuteProtocolStep(ctx context.Context, protocolInstanceID string) (*contracts.StepResult, error)

	// GetProtocolStatus returns current protocol execution status
	GetProtocolStatus(ctx context.Context, protocolInstanceID string) (*contracts.ProtocolStatus, error)

	// ResolveConflict arbitrates between conflicting protocol recommendations
	// NOTE: ICU conflicts are NOT resolved here - ICU always wins
	ResolveConflict(ctx context.Context, conflicts []contracts.ProtocolConflict) (*contracts.ConflictResolution, error)

	// GetProtocolLibrary returns available protocols
	GetProtocolLibrary(ctx context.Context, category string) ([]contracts.ProtocolSummary, error)

	// GetRecommendationWithEvidence returns recommendation with KB-15 evidence attached
	// This is the PRIMARY use case for KB-19
	GetRecommendationWithEvidence(ctx context.Context, clinicalContext contracts.ClinicalContext) (*contracts.EvidenceBasedRecommendation, error)
}
```

**KB-19 Output Structure** (must include ICU deferral flag):
```go
type ProtocolSelection struct {
	RecommendedProtocol    string
	Rationale              string
	EvidenceGrade          string   // From KB-15

	// REQUIRED: ICU authority acknowledgment
	DeferToICUIfDominant   bool     // Always true for high-risk actions
	CanBeOverriddenByICU   bool     // Always true
}
```

**Estimated Lines**: ~500 lines

---

### 4.4 Phase 4 Deliverables Summary

| Deliverable | Type | Location | Lines | Status |
|-------------|------|----------|-------|--------|
| IndiaThresholdAdapter.cql | CQL | tier-5-regional-adapters/IN/ | ~200 | Required |
| IndiaDrugAdapter.cql | CQL | tier-5-regional-adapters/IN/ | ~150 | Required |
| AustraliaThresholdAdapter.cql | CQL | tier-5-regional-adapters/AU/ | ~200 | Required |
| AustraliaDrugAdapter.cql | CQL | tier-5-regional-adapters/AU/ | ~150 | Required |
| SepsisGuidelines.cql | CQL | tier-4-guidelines/clinical/ | ~400 | Required |
| VTEProphylaxisGuidelines.cql | CQL | tier-4-guidelines/clinical/ | ~350 | Required |
| HFGDMTGuidelines.cql | CQL | tier-4-guidelines/clinical/ | ~400 | Required |
| T2DMGuidelines.cql | CQL | tier-4-guidelines/clinical/ | ~400 | Required |
| AKIGuidelines.cql | CQL | tier-4-guidelines/clinical/ | ~350 | Required |
| kb3_http_client.go | Go | clients/ | ~400 | Required |
| kb12_http_client.go | Go | clients/ | ~400 | Required |
| kb19_http_client.go | Go | clients/ | ~500 | Required |

**Total Phase 4 Estimated Lines**: ~3,900 lines

---

## Dependency Graph

```
                           ┌─────────────────────────────────────────┐
                           │         TIER 0: FHIRHelpers             │
                           │              (COMPLETE)                 │
                           └─────────────────┬───────────────────────┘
                                             │
                           ┌─────────────────┴───────────────────────┐
                           │         TIER 1: Primitives              │
                           │    Interval, Observation, Medication    │
                           │              (COMPLETE)                 │
                           └─────────────────┬───────────────────────┘
                                             │
              ┌──────────────────────────────┼──────────────────────────────┐
              │                              │                              │
              ▼                              ▼                              ▼
┌─────────────────────────┐    ┌─────────────────────────┐    ┌─────────────────────────┐
│   TIER 1.5: Utilities   │    │   TIER 2: CQMCommon     │    │   TIER 3: Domain        │
│   ClinicalCalculators   │    │      (COMPLETE)         │    │   SafetyCommon          │
│   LabReferenceRanges    │    │                         │    │   CardiovascularCommon  │
│     (PHASE 2)           │    │                         │    │   DiabetesCommon        │
└───────────┬─────────────┘    └───────────┬─────────────┘    │   RenalCommon           │
            │                              │                   │     (PHASE 1)           │
            │                              │                   └───────────┬─────────────┘
            │                              │                               │
            │              ┌───────────────┴───────────────┐               │
            │              │                               │               │
            ▼              ▼                               ▼               ▼
┌───────────────────────────────────────────────────────────────────────────────────┐
│                         TIER 4a: Quality Measures                                  │
│                    CMS122, CMS134, CMS165, CMS2 (PARTIAL)                         │
└───────────────────────────────────────────┬───────────────────────────────────────┘
                                            │
              ┌─────────────────────────────┼─────────────────────────────┐
              │                             │                             │
              ▼                             ▼                             ▼
┌─────────────────────────┐   ┌─────────────────────────┐   ┌─────────────────────────┐
│   TIER 4b: Guidelines   │   │   TIER 5: Regional      │   │   TIER 6a: Orchestration│
│   SepsisGuidelines      │   │   IndiaAdapter          │   │   GovernanceClassifier  │
│   VTEGuidelines         │   │   AustraliaAdapter      │   │   PopulationStratifier  │
│   HFGDMTGuidelines      │   │     (PHASE 4)           │   │   AlertClassifier       │
│     (PHASE 4)           │   │                         │   │   CDSSOrchestrator      │
└───────────┬─────────────┘   └───────────┬─────────────┘   │     (PHASE 3)           │
            │                             │                 └───────────┬─────────────┘
            │                             │                             │
            └─────────────────────────────┼─────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                          TIER 6b: Governance (Go)                                   │
│                    workflow.go, scoring.go, conflicts.go (PARTIAL)                  │
│                      + kb14_http_client, kb18_http_client (PHASE 2-3)              │
└─────────────────────────────────────────┬───────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                          TIER 7: ICU Intelligence (Go)                              │
│                    icu_safety_rules.go, icu_clinical_state.go                       │
│                              (COMPLETE)                                             │
└─────────────────────────────────────────────────────────────────────────────────────┘

KB CLIENT DEPENDENCIES (Updated with Category Labels):
═══════════════════════════════════════════════════════

CATEGORY A - SNAPSHOT KBs (KnowledgeSnapshotBuilder):
─────────────────────────────────────────────────────
KB-1  (Drug Rules)    ──[SNAPSHOT]──► DosingSnapshot
KB-4  (Patient Safety)──[SNAPSHOT]──► SafetySnapshot
KB-5  (Drug Interact) ──[SNAPSHOT]──► InteractionSnapshot
KB-6  (Formulary)     ──[SNAPSHOT]──► FormularySnapshot
KB-7  (Terminology)   ──[SNAPSHOT]──► TerminologySnapshot
KB-8  (Calculator)    ──[SNAPSHOT]──► CalculatorSnapshot
KB-11 (Population)    ──[SNAPSHOT]──► PopulationSnapshot → Tier 6a
KB-16 (Lab Interp)    ──[SNAPSHOT]──► LabInterpretationSnapshot → Tier 1.5
KB-17 (Registry)      ──[SNAPSHOT]──► RegistrySnapshot → Tier 3

CATEGORY B - RUNTIME KBs (RuntimeClients):
──────────────────────────────────────────
KB-3  (Guidelines)    ──[RUNTIME]──► Consumes CQL output → Tier 4b
KB-9  (Care Gaps)     ──[RUNTIME]──► Consumes CQL output → Tier 2/4a
KB-10 (Rules Engine)  ──[RUNTIME]──► Consumes CQL output → Tier 6a
KB-12 (OrderSets)     ──[RUNTIME]──► Consumes CQL output → Tier 4b
KB-13 (Quality)       ──[RUNTIME]──► Consumes CQL output → Tier 2/4a
KB-14 (Navigator)     ──[RUNTIME]──► Consumes CQL output → Tier 6b
KB-15 (Evidence)      ──[RUNTIME]──► Fetch evidence, GRADE → KB-19 ◄── ADDED
KB-18 (Governance)    ──[RUNTIME]──► Consumes CQL output → Tier 6a
KB-19 (Protocol)      ──[RUNTIME]──► Consumes CQL output → Tier 6a

CATEGORY C - NON-KB (Tier 7 Safety Layer):
──────────────────────────────────────────
ICU Intelligence      ──[VETO]────► MANDATORY check BEFORE RuntimeClients

DATA FLOW PATTERN:
─────────────────
[SNAPSHOT KBs] → KnowledgeSnapshotBuilder → ClinicalExecutionContext
                                                    │
                                                    ▼
                                         ┌──────────────────┐
                                         │  CQL Evaluation  │
                                         │  (Frozen Data)   │
                                         └────────┬─────────┘
                                                  │ Classifications
                                                  ▼
                                         ┌──────────────────┐
                                         │ RuntimeClients   │
                                         │ (Workflow Layer) │
                                         └──────────────────┘
                                                  │ Actions
                                                  ▼
                                         [RUNTIME KBs Execute Workflows]
```

---

## Critical Path Analysis

```
CRITICAL PATH (Blocking Dependencies):
══════════════════════════════════════

Week 1-2: FOUNDATION [SNAPSHOT Layer] (Blocks everything)
├── SafetyCommon.cql ────► Blocks ICU Intelligence semantic layer
├── CardiovascularCommon.cql ────► Blocks KB-3, KB-9, KB-19
├── kb11_http_client.go [SNAPSHOT] ────► Blocks Population roadmap
├── kb17_http_client.go [SNAPSHOT] ────► Dependency for kb11
├── kb16_http_client.go [SNAPSHOT] ────► Blocks LabInterpretationSnapshot ◄── NEW
└── KnowledgeSnapshotBuilder update ────► Required for KB-16 integration ◄── NEW

Week 2-3: INTEGRATION [RUNTIME Layer] (Blocks orchestration)
├── ClinicalCalculators.cql ────► Blocks KB-8 CQL consumption
├── LabReferenceRanges.cql ────► Consumes KB-16 snapshot
├── kb18_http_client.go [RUNTIME] ────► Blocks governance workflow
├── kb10_http_client.go [RUNTIME] ────► Blocks alert routing
└── runtime_clients.go ────► RuntimeClients structure ◄── NEW

Week 3-4: ORCHESTRATION [RUNTIME Layer] (Blocks protocol execution)
├── GovernanceClassifier.cql ────► Required input for KB-18
├── CDSSOrchestrator.cql ────► Required for KB-19
├── kb14_http_client.go [RUNTIME] ────► Blocks workflow execution
├── kb9_http_client.go [RUNTIME] ────► Blocks care gap workflows
└── kb13_http_client.go [RUNTIME] ────► Blocks quality measure workflows

Week 5-6: REGIONAL & GUIDELINES [RUNTIME Layer] (Blocks multi-region)
├── IndiaThresholdAdapter.cql ────► Blocks India deployment
├── AustraliaThresholdAdapter.cql ────► Blocks AU deployment
├── SepsisGuidelines.cql ────► Blocks protocol library
├── kb3_http_client.go [RUNTIME] ────► Guidelines workflow
├── kb12_http_client.go [RUNTIME] ────► OrderSets workflow
└── kb19_http_client.go [RUNTIME] ────► Protocol orchestration

PARALLEL TRACKS (Can run concurrently):
═══════════════════════════════════════
Track A: CQL Development (Clinical team)
Track B: SNAPSHOT KB Clients (Engineering team - Week 1-2)
Track C: RUNTIME KB Clients (Engineering team - Week 2-4)
Track D: Testing & Validation (QA team)
```

---

## Testing Requirements

### Unit Testing

| Component | Test Framework | Coverage Target |
|-----------|----------------|-----------------|
| CQL Libraries | CQL Testing Framework | 80% |
| KB Clients | Go testing + testify | 85% |
| Integration | End-to-end tests | 70% |

### CQL Testing Strategy

```yaml
test_structure:
  tier_3_tests:
    - SafetyCommon_test.cql
    - CardiovascularCommon_test.cql
    - DiabetesCommon_test.cql
    - RenalCommon_test.cql
    - MaternalHealthCommon_test.cql

  tier_6a_tests:
    - GovernanceClassifier_test.cql
    - PopulationStratifier_test.cql
    - AlertClassifier_test.cql

test_data:
  - FHIR bundles with known conditions
  - Edge cases for threshold detection
  - Multi-condition patient scenarios
  - Regional threshold variations
```

### KB Client Testing Strategy

```go
// Example test structure for kb11_http_client_test.go
func TestKB11HTTPClient_GetPatientRiskStrata(t *testing.T) {
    // Setup mock server
    // Test successful response
    // Test error handling
    // Test timeout handling
}

func TestKB11HTTPClient_Integration(t *testing.T) {
    // Skip if KB-11 service not running
    // Test actual API calls
    // Verify response parsing
}
```

---

## Validation Criteria

### Phase Gate Criteria

| Phase | Entry Criteria | Exit Criteria |
|-------|----------------|---------------|
| Phase 1 | Tier 0,1,2 complete | All Tier 3 CQL compiles, KB-11/17 clients pass health checks |
| Phase 2 | Phase 1 complete | Tier 1.5 CQL compiles, KB-10/16/18 clients pass health checks |
| Phase 3 | Phase 2 complete | All Tier 6a CQL compiles, KB-9/13/14 clients operational |
| Phase 4 | Phase 3 complete | Regional adapters validated, Guidelines compile, KB-3/12/19 operational |

### Acceptance Criteria

```yaml
cql_acceptance:
  - Compiles without errors via CQL-to-ELM translator
  - All includes resolve correctly
  - ValueSet references validated against KB-7
  - Unit tests pass with > 80% coverage
  - Clinical review sign-off obtained

kb_client_acceptance:
  - Health check endpoint responds 200 OK
  - All interface methods implemented
  - Error handling follows patterns in existing clients
  - Response parsing validated against API contracts
  - Integration tests pass against running KB service
  - Timeout handling verified
```

---

## Resource Allocation

### Team Structure

| Role | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
|------|---------|---------|---------|---------|
| CQL Developer | 2 FTE | 1 FTE | 2 FTE | 2 FTE |
| Go Developer | 1 FTE | 2 FTE | 2 FTE | 2 FTE |
| Clinical SME | 1 FTE | 0.5 FTE | 1 FTE | 1.5 FTE |
| QA Engineer | 0.5 FTE | 1 FTE | 1 FTE | 1 FTE |

### Estimated Effort

| Phase | CQL Lines | Go Lines | Total Lines | Person-Days |
|-------|-----------|----------|-------------|-------------|
| Phase 1 | 1,650 | 850 | 2,500 | 10 |
| Phase 2 | 750 | 1,200 | 1,950 | 8 |
| Phase 3 | 1,600 | 1,350 | 2,950 | 12 |
| Phase 4 | 2,600 | 1,300 | 3,900 | 15 |
| **Total** | **6,600** | **4,700** | **11,300** | **45** |

---

## Risk Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| KB service API changes | High | Medium | Lock API contracts before development |
| CQL compilation issues | Medium | Low | Use CQL translator docker container for consistency |
| Clinical validation delays | High | Medium | Engage clinical SME early, parallel review |
| Integration failures | High | Medium | Mock servers for unit tests, staged integration |
| Regional threshold disputes | Medium | Medium | Document sources (ICMR, RACGP), get sign-off |

---

## Summary

### Implementation Totals

| Metric | Count |
|--------|-------|
| New CQL Files | 21 |
| New CQL Lines | ~6,600 |
| New Go Files | 12 |
| New Go Lines | ~4,700 |
| Total New Code | ~11,300 lines |
| Duration | 6 weeks |
| Person-Days | 45 |

### Post-Implementation State

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| CQL Tiers | 5/12 (42%) | 12/12 (100%) | +7 tiers |
| KB Clients | 8/20 (40%) | 20/20 (100%) | +12 clients |
| CQL Lines | 7,628 | ~14,200 | +6,600 |
| Go Client Lines | 3,835 | ~8,500 | +4,700 |

### Key Success Factors

1. **Parallel Execution**: CQL and Go development can proceed in parallel
2. **Clear Dependencies**: Phase gates ensure foundational work complete before dependent work
3. **Clinical Engagement**: Early SME involvement prevents rework
4. **Testing First**: Mock servers and test data ready before implementation
5. **Incremental Delivery**: Each phase delivers usable functionality

---

*Document generated: January 2026*
*Next review: Weekly during implementation*
