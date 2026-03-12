# ONC → MED-RT → OHDSI → LOINC Execution Contract

## Overview

This document defines the formal **Execution Contract** for Drug-Drug Interaction (DDI) processing in the CardioFit Clinical Synthesis Hub. The contract specifies the 4-layer pipeline from authoritative ONC rules to contextualized clinical decisions.

---

## Execution Pipeline

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│    ONC                MED-RT              OHDSI               LOINC         │
│  (Authority)        (Taxonomy)         (Expansion)         (Context)       │
│                                                                             │
│  ┌─────────┐       ┌─────────┐       ┌─────────────┐      ┌─────────────┐  │
│  │25 Rules │──────▶│Drug Class│─────▶│DDIProjection│─────▶│DDIDecision  │  │
│  │         │       │Hierarchy │       │   (2,527+)  │      │(Actionable) │  │
│  └─────────┘       └─────────┘       └─────────────┘      └─────────────┘  │
│                                                                             │
│  "WHAT rules      "HOW drugs        "CAN this         "DOES it matter     │
│   exist?"          are grouped?"     interaction       for THIS patient   │
│                                      exist?"           NOW?"              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Layer 1: ONC (Authority Layer)

**Purpose**: Define the authoritative constitutional rules for drug-drug interactions.

### Location
- Database: `canonical_facts.ddi_constitutional_rules`
- Service: None (static data loaded at initialization)

### Data Structure
```sql
CREATE TABLE ddi_constitutional_rules (
    rule_id SERIAL PRIMARY KEY,
    trigger_class_name VARCHAR(255) NOT NULL,    -- e.g., "Anticoagulants"
    trigger_concept_id BIGINT NOT NULL,          -- ATC/OHDSI concept ID
    target_class_name VARCHAR(255) NOT NULL,     -- e.g., "NSAIDs"
    target_concept_id BIGINT NOT NULL,           -- ATC/OHDSI concept ID
    risk_level VARCHAR(50) NOT NULL,             -- CRITICAL, HIGH, WARNING, MODERATE
    description TEXT NOT NULL,                   -- Clinical alert message

    -- LOINC Context Metadata
    context_loinc_id VARCHAR(20),                -- e.g., "6301-6" (INR)
    context_loinc_name VARCHAR(255),             -- e.g., "INR"
    context_threshold_val DECIMAL(10,2),         -- e.g., 3.0
    context_logic_operator VARCHAR(10),          -- <, >, <=, >=, =
    context_required BOOLEAN DEFAULT FALSE,

    -- Tiering & Governance
    evaluation_tier evaluation_tier_enum,        -- TIER_0_ONC_HIGH, TIER_1_SEVERE, etc.
    interaction_direction direction_enum,        -- BIDIRECTIONAL, AFFECTS_TRIGGER, AFFECTS_TARGET
    lazy_evaluate BOOLEAN DEFAULT FALSE,

    -- Provenance
    rule_authority VARCHAR(100) DEFAULT 'ONC',
    rule_version VARCHAR(50),
    active BOOLEAN DEFAULT TRUE
);
```

### Rule Count
- **25 ONC High-Priority Rules** (immutable constitutional set)

### Invariants
- ❌ ONC rules are NEVER modified at runtime
- ❌ ONC rules NEVER check LOINC values (that's Context Router's job)
- ✅ ONC rules specify LOINC metadata for downstream evaluation
- ✅ ONC TIER_0 rules can NEVER be suppressed without explicit override

---

## Layer 2: MED-RT (Taxonomy Layer)

**Purpose**: Provide drug class hierarchies and therapeutic relationships.

### Source
- NLM Medication Reference Terminology (MED-RT)
- Integrated into OHDSI vocabulary via concept relationships

### Drug Class Hierarchy
```
MED-RT Relationship Types:
├── has_ingredient           Drug → Ingredient
├── has_mechanism            Drug → Mechanism
├── has_physiologic_effect   Drug → Effect
├── has_therapeutic_class    Drug → Class (ATC)
├── may_treat                Drug → Condition
├── may_prevent              Drug → Condition
└── may_diagnose             Drug → Condition
```

### Integration Point
- MED-RT classes are mapped to OHDSI `concept_id` values
- ATC codes provide standardized therapeutic classification
- Used by ONC rules to specify drug classes (e.g., ATC C01AA05 = Digoxin class)

---

## Layer 3: OHDSI (Expansion Layer)

**Purpose**: Expand class-based rules to specific drug pairs using standardized vocabulary.

### Location
- Database: `canonical_facts.ohdsi_concept`, `ohdsi_concept_relationship`
- Service: `kb-5-drug-interactions/internal/services/ohdsi_expansion_service.go`

### Expansion Query
```sql
-- Get all drugs belonging to a class
SELECT cr.concept_id_1 AS drug_concept_id
FROM ohdsi_concept_relationship cr
JOIN ohdsi_concept c ON cr.concept_id_1 = c.concept_id
WHERE cr.concept_id_2 = :class_concept_id
  AND cr.relationship_id = 'Drug has drug class'
  AND c.standard_concept = 'S';
```

### Output: DDIProjection
```go
type DDIProjection struct {
    RuleID               int                  // From ONC rule
    DrugAConceptID       int64                // Expanded drug A
    DrugAName            string
    DrugBConceptID       int64                // Expanded drug B
    DrugBName            string
    RiskLevel            string               // CRITICAL, HIGH, WARNING, MODERATE
    AlertMessage         string               // Clinical description

    // LOINC Context Metadata (passed through from ONC rule)
    ContextRequired      bool
    ContextLOINCID       *string              // e.g., "6301-6"
    ContextThreshold     *float64             // e.g., 3.0
    ContextOperator      *string              // e.g., ">"

    // Tiering
    EvaluationTier       EvaluationTier       // TIER_0_ONC_HIGH, etc.
    LazyEvaluate         bool
}
```

### Expansion Statistics (v1.0-spine)
| Metric | Value |
|--------|-------|
| ONC Rules | 25 |
| Expanded Drug Pairs | 2,527 |
| Average Expansion Factor | 101× |

### Invariants
- ❌ Expansion service NEVER checks LOINC values
- ❌ Expansion service NEVER makes clinical decisions
- ✅ Expansion service produces semantic drug pairs with LOINC metadata embedded
- ✅ Expansion service respects class membership from OHDSI vocabulary

---

## Layer 4: LOINC (Context Layer)

**Purpose**: Evaluate patient-specific clinical context against thresholds.

### Location
- Service: `shared-infrastructure/orchestration/context_router/`
- Files: `context_router.go`, `loinc_evaluator.go`, `decision_model.go`

### Input: PatientContext
```go
type PatientContext struct {
    PatientID       string
    Labs            map[string]LabValue  // LOINC code → value
    Age             *int
    Weight          *float64
    RenalFunction   *float64             // eGFR
    HepaticFunction *string              // Child-Pugh class
}

type LabValue struct {
    Value     float64
    Unit      string
    Timestamp time.Time
}
```

### Threshold Evaluation
```go
// Pure deterministic evaluation: value ⨯ operator ⨯ threshold
func EvaluateThreshold(loincCode string, value float64, threshold float64, operator string) ThresholdResult
```

### Output: DDIDecision
```go
type DDIDecision struct {
    RuleID           int
    DrugAConceptID   int64
    DrugBConceptID   int64

    Decision         DecisionType  // BLOCK, INTERRUPT, INFORMATIONAL, SUPPRESSED, NEEDS_CONTEXT
    RiskLevel        string
    AlertMessage     string

    // Audit Trail
    Reason           string
    ContextEvaluated bool
    ContextLOINCID   *string
    ContextValue     *float64
    ThresholdExceeded *bool

    EvaluatedAt      time.Time
}
```

### Decision Type Matrix

| Risk Level | Context Missing | Threshold Exceeded | Threshold Safe |
|------------|-----------------|-------------------|----------------|
| **CRITICAL** | BLOCK | BLOCK | BLOCK |
| **HIGH (ONC)** | INTERRUPT | INTERRUPT | INFORMATIONAL |
| **HIGH (non-ONC)** | NEEDS_CONTEXT | INTERRUPT | INFORMATIONAL |
| **WARNING** | NEEDS_CONTEXT | INTERRUPT | SUPPRESSED |
| **MODERATE** | NEEDS_CONTEXT | INFORMATIONAL | SUPPRESSED |

### Invariants
- ✅ Context Router ALWAYS evaluates LOINC when `context_required=true`
- ✅ Context Router provides full audit trail for every decision
- ✅ TIER_0 (ONC Constitutional) rules are NEVER suppressed in strict mode
- ✅ Missing context for required rules fails safe (INTERRUPT or NEEDS_CONTEXT)
- ❌ Context Router NEVER modifies projections (read-only consumer)
- ❌ Context Router NEVER makes network calls to external systems

---

## Golden Rules (System Invariants)

| # | Rule | Enforcement |
|---|------|-------------|
| 1 | **Class Expansion NEVER checks LOINC** | OHDSI service has no LOINC logic |
| 2 | **Context Router ALWAYS checks LOINC (when required)** | `RequiresContext()` check in policy |
| 3 | **TIER_0 rules cannot be suppressed** | `StrictONCMode` config flag |
| 4 | **Expansion answers CAN, Context answers DOES** | Architectural separation |
| 5 | **All decisions have audit trail** | `DDIDecision.Reason` mandatory |
| 6 | **Projections are immutable after expansion** | Context Router is read-only |

---

## Common LOINC Codes for DDI Context

| LOINC Code | Name | Common Use |
|------------|------|------------|
| 6301-6 | INR | Warfarin + NSAID/Antiplatelet interactions |
| 2823-3 | Potassium | Digoxin + Diuretic, ACE + K-sparing |
| 33914-3 | eGFR | Renal-cleared drugs (Metformin, Lithium) |
| 10535-3 | Digoxin Level | Digoxin toxicity monitoring |
| 8634-8 | QTc Interval | QT-prolonging drug combinations |
| 2160-0 | Creatinine | Renal function proxy |
| 1742-6 | ALT | Hepatotoxic drug combinations |
| 777-3 | Platelets | Antiplatelet/anticoagulant bleeding risk |

---

## Example Flow: Warfarin + Ibuprofen

```
Step 1: ONC Rule #1 (Authority)
├── Trigger: Anticoagulants (ATC B01AA)
├── Target: NSAIDs (ATC M01A)
├── Risk: HIGH
├── Context: INR (LOINC 6301-6), Threshold > 3.0
└── Tier: TIER_0_ONC_HIGH

Step 2: OHDSI Expansion (Semantic)
├── Anticoagulants → [Warfarin, Heparin, Apixaban, ...]
├── NSAIDs → [Ibuprofen, Naproxen, Aspirin, ...]
├── Cross-product: 127 drug pairs
└── Output: DDIProjection{DrugA: Warfarin, DrugB: Ibuprofen, ...}

Step 3: Context Router (Clinical)
├── Patient Labs: INR = 2.5
├── Threshold Check: 2.5 > 3.0? → FALSE
├── ONC Strict Mode: Cannot suppress
└── Output: DDIDecision{Decision: INFORMATIONAL, Reason: "INR within safe range"}

Alternative Step 3: INR = 4.0
├── Threshold Check: 4.0 > 3.0? → TRUE
└── Output: DDIDecision{Decision: INTERRUPT, Reason: "INR exceeds threshold"}
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| v1.0-spine | 2026-01-21 | Initial 25 ONC rules, OHDSI expansion, 2,527 pairs |
| v1.1-context | 2026-01-22 | Context Router in shared orchestration layer |

---

## Related Documents

- [KB-5 Drug Interactions Service](../knowledge-base-services/kb-5-drug-interactions/README.md)
- [Context Router Implementation](./orchestration/context_router/)
- [OHDSI Vocabulary Setup](../knowledge-base-services/shared/extraction/etl/)
