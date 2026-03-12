# MedicationEngine - Runtime Safety Engine

> **Runtime Layer** - Deterministic Safety Executor for Medication Orders

## Overview

The `MedicationEngine` is a **Runtime Layer engine** in the Clinical Runtime Platform. It is a thin, deterministic, stateless engine that consumes the frozen `ClinicalExecutionContext` and produces safety alerts and constraint recommendations.

> *"The MedicationEngine is a deterministic safety executor, not a decision maker. It consumes frozen clinical truths and accountability facts, enforces constraints, and deliberately avoids therapy strategy, which belongs to the Advisor Engine."*

### Position in Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        RUNTIME LAYER (This Engine)                       │
│                                                                         │
│  ┌──────────────┐   ┌──────────────────┐   ┌──────────────────┐        │
│  │  CQL Engine  │   │  Measure Engine  │   │ MedicationEngine │ ◄──────│
│  │  (Truths)    │   │  (Care Gaps)     │   │ (Safety Rails)   │        │
│  └──────────────┘   └──────────────────┘   └──────────────────┘        │
│          │                  │                       │                   │
│          └──────────────────┴───────────────────────┘                   │
│                              │                                          │
│                              ▼                                          │
│                   Frozen ClinicalExecutionContext                       │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               │ (External - NOT Runtime)
                               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    TIER 6: MEDICATION ADVISOR ENGINE                     │
│                    (Proposals, Rankings, Strategy)                       │
└─────────────────────────────────────────────────────────────────────────┘
```

### Engine Layer Responsibilities

| Engine | Question Answered |
|--------|-------------------|
| CQL Engine | *What is true about this patient?* |
| Measure Engine | *Are we meeting standards of care?* |
| **MedicationEngine** | *Is this order safe, legal, and dose-correct given what is true?* |

---

## Engine Contract

### Core Principles

```go
// ENGINE CONTRACT:
// 1. Engines receive ClinicalExecutionContext - they NEVER call KBs directly
// 2. Engines are STATELESS - all data comes from context
// 3. Engines return EngineResult - recommendations, alerts, measures
// 4. Engines are DETERMINISTIC - same context = same result
```

### Interface

```go
type MedicationEngine struct {
    config MedicationEngineConfig
}

func (e *MedicationEngine) Name() string
func (e *MedicationEngine) Evaluate(
    ctx context.Context,
    execCtx *contracts.ClinicalExecutionContext,
) (*contracts.EngineResult, error)
```

### What This Engine Does

| Capability | Status |
|------------|--------|
| Safety alerts (DDI, contraindications) | ✅ YES |
| Dose adjustment recommendations | ✅ YES |
| Formulary compliance checks | ✅ YES |
| Regional rules (NLEM/PBS) | ✅ YES |
| Allergy/pregnancy warnings | ✅ YES |

### What This Engine Does NOT Do

| Capability | Status | Where It Belongs |
|------------|--------|------------------|
| Drug proposals | ❌ NO | Tier 6 Advisor Engine |
| Drug ranking | ❌ NO | Tier 6 Advisor Engine |
| Therapy strategy | ❌ NO | Tier 6 Advisor Engine |
| Care gap decisions | ❌ NO | Measure Engine |
| External KB calls | ❌ NO | Snapshot Builder |

---

## Configuration

### MedicationEngineConfig

```go
type MedicationEngineConfig struct {
    CheckInteractions      bool   // Enable DDI checking (KB-5)
    CheckContraindications bool   // Enable drug-condition checking (KB-4)
    CheckDosing            bool   // Enable dose adjustment recommendations (KB-1)
    CheckFormulary         bool   // Enable formulary status recommendations (KB-6)
    Region                 string // Regional rules: "AU", "IN"
}
```

### Default Configuration

```go
func DefaultMedicationEngineConfig() MedicationEngineConfig {
    return MedicationEngineConfig{
        CheckInteractions:      true,
        CheckContraindications: true,
        CheckDosing:            true,
        CheckFormulary:         true,
        Region:                 "AU",
    }
}
```

### Usage

```go
// Default configuration
engine := NewMedicationEngine(DefaultMedicationEngineConfig())

// Custom configuration (India region, no formulary checks)
engine := NewMedicationEngine(MedicationEngineConfig{
    CheckInteractions:      true,
    CheckContraindications: true,
    CheckDosing:            true,
    CheckFormulary:         false,
    Region:                 "IN",
})
```

---

## Data Flow

### Input: ClinicalExecutionContext (Frozen)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    ClinicalExecutionContext (Frozen)                     │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  Knowledge Snapshot (Pre-computed by Snapshot Builder)            │  │
│  │                                                                   │  │
│  │  Interactions (KB-5)     → CurrentDDIs, HasCriticalInteraction   │  │
│  │  Safety (KB-4)           → Contraindications, Allergies, Pregnancy│  │
│  │  Dosing (KB-1)           → Renal, Hepatic, Age-based adjustments │  │
│  │  Formulary (KB-6)        → Status, Alternatives, Prior Auth      │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  Patient Context                                                  │  │
│  │  ActiveMedications, Conditions, Labs, Vitals                      │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  Runtime Metadata                                                 │  │
│  │  Region, RequestID, TenantID                                      │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Output: EngineResult

```go
type EngineResult struct {
    EngineName      string            // "medication-advisor"
    Success         bool              // Execution status
    Recommendations []Recommendation  // Dose adjustments, substitutions
    Alerts          []Alert           // DDI warnings, contraindications
    ExecutionTimeMs int64             // Performance metric
    EvidenceLinks   []string          // Audit trail links
}
```

---

## Five Safety Checks

### 1. Drug-Drug Interactions (KB-5)

**Source**: `execCtx.Knowledge.Interactions`

| Field | Purpose |
|-------|---------|
| `CurrentDDIs` | List of active medication interactions |
| `HasCriticalInteraction` | Quick flag for critical alerts |
| `SeverityMax` | Maximum severity level present |

**Output**:
- **Alerts**: For severe/critical interactions
- **Recommendations**: Review actions for all interactions

```go
// Alert example
Alert{
    ID:          "DDI-a1b2c3d4",
    Severity:    "critical",
    Category:    "medication-safety",
    Title:       "Drug Interaction: Warfarin + Aspirin",
    Description: "Increased bleeding risk when combined",
}

// Recommendation example
Recommendation{
    ID:          "REC-DDI-e5f6g7h8",
    Type:        "medication-review",
    Title:       "Review interaction: Warfarin + Aspirin",
    Priority:    "high",
    Source:      "medication-advisor/ddi-checker",
    Actions:     [{Type: "review", Description: "Consider dose adjustment"}],
}
```

---

### 2. Drug-Condition Contraindications (KB-4)

**Source**: `execCtx.Knowledge.Safety.Contraindications`

**Output**:
- **Alerts**: For all contraindications
- **Recommendations**: Discontinuation suggestions

```go
// Alert example
Alert{
    ID:          "CI-i9j0k1l2",
    Severity:    "high",
    Category:    "medication-safety",
    Title:       "Contraindicated: Metformin with Severe Renal Impairment",
    Description: "eGFR < 30 contraindicates metformin use",
}

// Recommendation example
Recommendation{
    ID:          "REC-CI-m3n4o5p6",
    Type:        "medication-discontinuation",
    Title:       "Consider discontinuing Metformin",
    Priority:    "high",
    Source:      "medication-advisor/contraindication-checker",
}
```

---

### 3. Dose Adjustments (KB-1)

**Source**: `execCtx.Knowledge.Dosing`

| Adjustment Type | Map Key |
|-----------------|---------|
| Renal | `RenalAdjustments` |
| Hepatic | `HepaticAdjustments` |
| Age-based | `AgeBasedAdjustments` |

**Output**:
- **Recommendations**: Modify-order actions with adjustment percentages
- **Alerts**: When renal/hepatic adjustment flags are set

```go
// Recommendation example
Recommendation{
    ID:          "REC-RENAL-q7r8s9t0",
    Type:        "dose-adjustment",
    Title:       "Renal dose adjustment needed: Gentamicin",
    Description: "Reduce dose for eGFR < 60",
    Priority:    "medium",
    Source:      "medication-advisor/renal-dosing (eGFR threshold: 60)",
    Actions:     [{Type: "modify-order", Description: "Adjust dose by 50%"}],
}

// Alert example
Alert{
    ID:          "RENAL-ADJ-u1v2w3x4",
    Severity:    "moderate",
    Category:    "dosing",
    Title:       "Renal Dose Adjustments Required",
    Description: "Patient has reduced kidney function. Review all renally-cleared medications.",
}
```

---

### 4. Formulary Status (KB-6)

**Source**: `execCtx.Knowledge.Formulary`

| Field | Purpose |
|-------|---------|
| `MedicationStatus` | Preferred/non-preferred/excluded status |
| `GenericAlternatives` | Alternative medications |
| `PriorAuthRequired` | Medications requiring prior authorization |
| `NLEMAvailability` | India National List of Essential Medicines |
| `PBSAvailability` | Australia Pharmaceutical Benefits Scheme |

**Regional Logic**:
```go
if region == "IN" {
    // Check NLEM availability
}
if region == "AU" {
    // Check PBS availability
}
```

**Output**:
- **Recommendations**: Formulary substitutions, NLEM/PBS alternatives
- **Alerts**: Prior authorization requirements

---

### 5. Safety Alerts (KB-4)

**Source**: `execCtx.Knowledge.Safety`

| Field | Purpose |
|-------|---------|
| `SafetyAlerts` | Pre-computed safety alerts (passthrough) |
| `PregnancyStatus` | Pregnancy/lactation status |
| `ActiveAllergies` | Patient allergies with criticality |

**Output**:
- **Alerts**: Pregnancy warnings, lactation warnings, high-risk allergies

```go
// Pregnancy alert
Alert{
    ID:          "PREG-y5z6a7b8",
    Severity:    "high",
    Category:    "medication-safety",
    Title:       "Pregnancy Alert - Review All Medications",
    Description: "Patient is pregnant (Trimester 2, ~24 weeks). Review all medications for pregnancy safety categories.",
}

// Allergy alert
Alert{
    ID:          "ALLERGY-c9d0e1f2",
    Severity:    "high",
    Category:    "allergy",
    Title:       "High-Risk Allergy: Penicillin",
    Description: "Patient has high-criticality allergy to Penicillin (drug). Reactions: [anaphylaxis]",
}
```

---

## Severity Mapping

### Alert Severity Mapping

| Input Severity | Output Severity |
|----------------|-----------------|
| `critical` | `critical` |
| `severe`, `high` | `high` |
| `moderate`, `medium` | `moderate` |
| default | `low` |

### Priority Mapping

| Input Severity | Output Priority |
|----------------|-----------------|
| `critical`, `severe` | `high` |
| `moderate` | `medium` |
| default | `low` |

---

## Regional Support

### Australia (AU)

| Feature | Implementation |
|---------|----------------|
| PBS Availability | Checks `execCtx.Knowledge.Formulary.PBSAvailability` |
| Alert Type | `formulary-substitution` |
| Message | "Not on PBS. Patient may face higher out-of-pocket costs." |

### India (IN)

| Feature | Implementation |
|---------|----------------|
| NLEM Availability | Checks `execCtx.Knowledge.Formulary.NLEMAvailability` |
| Alert Type | `formulary-substitution` |
| Message | "Not on NLEM. Consider NLEM alternatives if available." |

---

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "vaidshala/clinical-runtime-platform/contracts"
    "vaidshala/clinical-runtime-platform/engines"
)

func main() {
    // Create engine with default config
    engine := engines.NewMedicationEngine(engines.DefaultMedicationEngineConfig())

    // Get frozen context from orchestrator
    execCtx := getExecutionContext() // From snapshot builder

    // Evaluate
    result, err := engine.Evaluate(context.Background(), execCtx)
    if err != nil {
        log.Fatal(err)
    }

    // Process alerts
    for _, alert := range result.Alerts {
        fmt.Printf("[%s] %s: %s\n", alert.Severity, alert.Title, alert.Description)
    }

    // Process recommendations
    for _, rec := range result.Recommendations {
        fmt.Printf("[%s] %s: %s\n", rec.Priority, rec.Title, rec.Description)
    }
}
```

### Integration with Orchestrator

```go
// In orchestrator - engines run in sequence
func (o *Orchestrator) Execute(ctx context.Context, execCtx *contracts.ClinicalExecutionContext) (*OrchestratorResult, error) {

    // 1. CQL Engine - determine truths
    cqlResult, _ := o.cqlEngine.Evaluate(ctx, execCtx)

    // 2. Measure Engine - determine care gaps (uses CQL facts)
    o.measureEngine.SetClinicalFacts(cqlResult.ClinicalFacts)
    measureResult, _ := o.measureEngine.Evaluate(ctx, execCtx)

    // 3. Medication Engine - determine safety constraints
    medResult, _ := o.medicationEngine.Evaluate(ctx, execCtx)

    // Combine results
    return &OrchestratorResult{
        ClinicalFacts:    cqlResult.ClinicalFacts,
        MeasureResults:   measureResult.MeasureResults,
        Alerts:           medResult.Alerts,
        Recommendations:  medResult.Recommendations,
    }, nil
}
```

---

## CTO/CMO Compliance Checklist

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Stateless engine | ✅ PASS | Only holds `MedicationEngineConfig` - no mutable state |
| Uses KnowledgeSnapshot only | ✅ PASS | All data from `execCtx.Knowledge.*` snapshots |
| No external KB calls | ✅ PASS | Zero HTTP/gRPC calls - pure function |
| Deterministic | ✅ PASS | Same context = same result (always) |
| Produces alerts/constraints | ✅ PASS | Returns `Alerts` + `Recommendations` |
| Does NOT generate proposals | ✅ PASS | No proposal generation logic |
| Does NOT rank drugs | ✅ PASS | No ranking/scoring algorithm |
| Does NOT run workflows | ✅ PASS | Single synchronous evaluation |

---

## Architecture Boundaries

### What MUST NOT Happen

- MedicationEngine must NOT call MeasureEngine
- MedicationEngine must NOT interpret measure logic
- MedicationEngine must NOT branch on CMS IDs
- MedicationEngine must NOT decide "care gaps"
- MedicationEngine must NOT make HTTP/gRPC calls

### What IS Correct

- Measure results are materialized upstream into ClinicalExecutionContext
- MedicationEngine sees renal impairment, bleeding risk, etc. as **facts**, not measures
- The engine consumes facts without knowing their measure origin

---

## Related Components

| Component | Relationship |
|-----------|--------------|
| [CQL Engine](cql_engine.go) | Sibling - determines clinical truths |
| [Measure Engine](measure_engine.go) | Sibling - determines care gaps |
| [Execution Context](../contracts/execution_context.go) | Input contract |
| [Snapshot Builder](../builders/knowledge_snapshot.go) | Builds the frozen context |
| [Medication Advisor Engine](../services/medication-advisor-engine/) | Tier 6 - therapy strategy (external) |

---

## File Reference

**Source**: `vaidshala/clinical-runtime-platform/engines/medication_engine.go`

**Lines**: 462

**Dependencies**:
- `github.com/google/uuid` - Alert ID generation
- `vaidshala/clinical-runtime-platform/contracts` - Execution context and result types

---

## License

Proprietary - CardioFit Clinical Synthesis Hub
