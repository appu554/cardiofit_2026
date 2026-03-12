# Phase 3d: Truth Arbitration Engine Implementation Plan

> "When truths collide, precedence decides."
> The final arbiter before clinical action.

## Executive Summary

Phase 3d implements the **Truth Arbitration Engine** - a deterministic conflict resolution system that reconciles disagreements between:
- **Phase 3b.5**: Canonical Rules (extracted from FDA SPL/SmPC)
- **Phase 3b.6**: Lab Interpretations (KB-16 context-aware ranges)
- **Authority Facts**: CPIC, CredibleMeds, LactMed guidelines

### Decision Outcomes
| Decision | Meaning | Clinical Action |
|----------|---------|-----------------|
| **ACCEPT** | All sources agree or no conflicts | Proceed |
| **BLOCK** | Hard constraint violated | Cannot proceed |
| **OVERRIDE** | Soft conflict, can proceed with acknowledgment | Warning + documentation |
| **DEFER** | Insufficient data | Request more information |
| **ESCALATE** | Complex conflict | Route to expert review |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    TRUTH ARBITRATION ENGINE                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │ REGULATORY   │  │ AUTHORITY    │  │ LAB          │           │
│  │ BLOCKS       │  │ FACTS        │  │ INTERP       │           │
│  │ (FDA BBW)    │  │ (CPIC)       │  │ (KB-16)      │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘           │
│         │                 │                 │                    │
│         └────────────┬────┴────────────────┘                    │
│                      ▼                                           │
│         ┌────────────────────────┐                              │
│         │   CONFLICT DETECTOR    │                              │
│         │  - Pairwise comparison │                              │
│         │  - Pattern matching    │                              │
│         └───────────┬────────────┘                              │
│                     ▼                                            │
│         ┌────────────────────────┐                              │
│         │   PRECEDENCE ENGINE    │                              │
│         │  - P1-P7 rule ladder   │                              │
│         │  - Resolution matrix   │                              │
│         └───────────┬────────────┘                              │
│                     ▼                                            │
│         ┌────────────────────────┐                              │
│         │  DECISION SYNTHESIZER  │                              │
│         │  - Final verdict       │                              │
│         │  - Confidence score    │                              │
│         │  - Audit trail         │                              │
│         └───────────┬────────────┘                              │
│                     ▼                                            │
│  ┌──────────────────────────────────────────────────────┐       │
│  │              ArbitrationDecision                      │       │
│  │  - Decision: ACCEPT|BLOCK|OVERRIDE|DEFER|ESCALATE   │       │
│  │  - WinningSource: REGULATORY|AUTHORITY|LAB|RULE     │       │
│  │  - ConflictsFound: []Conflict                        │       │
│  │  - AuditTrail: []AuditEntry                         │       │
│  └──────────────────────────────────────────────────────┘       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Precedence Hierarchy

```
PRECEDENCE LATTICE (Highest to Lowest)
═══════════════════════════════════════

     ┌─────────────────────┐
     │   1. REGULATORY     │  FDA Black Box, REMS, Contraindication
     │   Trust: 1.00       │  ALWAYS WINS (P1)
     └──────────┬──────────┘
                ▼
     ┌─────────────────────┐
     │   2. AUTHORITY      │  CPIC, CredibleMeds, LactMed
     │   Trust: 1.00       │  Curated expert consensus
     └──────────┬──────────┘
                ▼
     ┌─────────────────────┐
     │   3. LAB_INTERP     │  KB-16 critical/abnormal
     │   Trust: 0.95       │  Real-time patient data
     └──────────┬──────────┘
                ▼
     ┌─────────────────────┐
     │   4. CANONICAL_RULE │  Phase 3b.5 extracted rules
     │   Trust: 0.90       │  Deterministic extraction
     └──────────┬──────────┘
                ▼
     ┌─────────────────────┐
     │   5. LOCAL_POLICY   │  Hospital formulary overrides
     │   Trust: 0.80       │  Site-specific (cannot override authorities)
     └─────────────────────┘
```

---

## Precedence Rules (P1-P7)

| Rule | Description | Winner | Rationale |
|------|-------------|--------|-----------|
| **P1** | REGULATORY_BLOCK always wins | REGULATORY | Legal requirement |
| **P2** | DEFINITIVE authority > PRIMARY authority | Higher level | Evidence hierarchy |
| **P3** | AUTHORITY_FACT > CANONICAL_RULE (same drug) | AUTHORITY | Curated > extracted |
| **P4** | LAB critical + RULE triggered = ESCALATE | ESCALATE | Real-time validation |
| **P5** | More provenance sources > fewer sources | Higher count | Consensus strength |
| **P6** | LOCAL_POLICY can override rules, NOT authorities | Conditional | Site autonomy + safety |
| **P7** | More restrictive action wins ties | Stricter | Fail-safe default |

---

## Conflict Types

| Conflict Type | Example | Frequency | Severity |
|---------------|---------|-----------|----------|
| `RULE_VS_AUTHORITY` | SPL "avoid" vs CPIC "contraindicated" | Common | MEDIUM |
| `RULE_VS_LAB` | Rule: CrCl < 30, Lab: eGFR = 28 | Common | HIGH |
| `AUTHORITY_VS_LAB` | CPIC: eGFR < 30, Lab: normal for pregnancy | Rare | CRITICAL |
| `AUTHORITY_VS_AUTHORITY` | CPIC vs CredibleMeds on same drug | Rare | HIGH |
| `RULE_VS_RULE` | Two SPLs with different thresholds | Common | LOW |
| `LOCAL_VS_ANY` | Hospital policy overrides guideline | Common | MEDIUM |

---

## Implementation Tasks

### Task 1: Database Migration (Day 1-2)

**File**: `migrations/004_truth_arbitration.sql`

```sql
-- Tables to create:
-- 1. arbitration_decisions - Decision audit log
-- 2. conflicts_detected - Conflict records
-- 3. precedence_rules - Configurable P1-P7 rules
-- 4. authority_facts - CPIC, CredibleMeds facts cache
-- 5. regulatory_blocks - FDA BBW cache
```

### Task 2: Core Type Definitions (Day 2-3)

**File**: `pkg/arbitration/types.go`

```go
type SourceType string
const (
    SourceRegulatory   SourceType = "REGULATORY"
    SourceAuthority    SourceType = "AUTHORITY"
    SourceLab          SourceType = "LAB"
    SourceRule         SourceType = "RULE"
    SourceLocal        SourceType = "LOCAL"
)

type DecisionType string
const (
    DecisionAccept   DecisionType = "ACCEPT"
    DecisionBlock    DecisionType = "BLOCK"
    DecisionOverride DecisionType = "OVERRIDE"
    DecisionDefer    DecisionType = "DEFER"
    DecisionEscalate DecisionType = "ESCALATE"
)

type ConflictType string
const (
    ConflictRuleVsAuthority      ConflictType = "RULE_VS_AUTHORITY"
    ConflictRuleVsLab            ConflictType = "RULE_VS_LAB"
    ConflictAuthorityVsLab       ConflictType = "AUTHORITY_VS_LAB"
    ConflictAuthorityVsAuthority ConflictType = "AUTHORITY_VS_AUTHORITY"
    ConflictRuleVsRule           ConflictType = "RULE_VS_RULE"
    ConflictLocalVsAny           ConflictType = "LOCAL_VS_ANY"
)
```

### Task 3: Input/Output Schemas (Day 3-4)

**File**: `pkg/arbitration/schemas.go`

- `ArbitrationInput` - All assertions for a clinical decision
- `ArbitrationDecision` - Final verdict with audit trail
- `CanonicalRuleAssertion` - Phase 3b.5 rules
- `AuthorityFactAssertion` - CPIC/CredibleMeds/LactMed
- `LabInterpretationAssertion` - KB-16 results
- `RegulatoryBlockAssertion` - FDA Black Box
- `LocalPolicyAssertion` - Hospital overrides

### Task 4: Conflict Detector (Day 4-6)

**File**: `pkg/arbitration/conflict_detector.go`

```go
type ConflictDetector struct {
    // Detect all pairwise conflicts between assertions
}

func (cd *ConflictDetector) DetectConflicts(evaluated *EvaluatedAssertions) []Conflict {
    // Compare each pair of assertions
    // Identify conflicts based on effect disagreement
    // Classify conflict type
}
```

### Task 5: Precedence Engine (Day 6-8)

**File**: `pkg/arbitration/precedence_engine.go`

```go
type PrecedenceEngine struct {
    rules []PrecedenceRule // P1-P7
}

func (pe *PrecedenceEngine) ResolveConflict(conflict *Conflict) *Resolution {
    // Apply P1-P7 rules in order
    // Return winner and rule applied
}

// Resolution Matrix implementation
func (pe *PrecedenceEngine) GetWinner(sourceA, sourceB SourceType) SourceType {
    // Use conflict resolution matrix
}
```

### Task 6: Decision Synthesizer (Day 8-10)

**File**: `pkg/arbitration/decision_synthesizer.go`

```go
type DecisionSynthesizer struct {
    precedenceEngine *PrecedenceEngine
}

func (ds *DecisionSynthesizer) Synthesize(
    conflicts []Conflict,
    evaluated *EvaluatedAssertions,
) *ArbitrationDecision {
    // Determine final decision from resolved conflicts
    // Calculate confidence
    // Generate audit trail
}
```

### Task 7: Main Arbitration Engine (Day 10-12)

**File**: `pkg/arbitration/engine.go`

```go
type ArbitrationEngine struct {
    conflictDetector    *ConflictDetector
    precedenceEngine    *PrecedenceEngine
    decisionSynthesizer *DecisionSynthesizer
}

func (e *ArbitrationEngine) Arbitrate(input *ArbitrationInput) (*ArbitrationDecision, error) {
    // STEP 1: Check regulatory blocks (P1)
    // STEP 2: Evaluate all assertions
    // STEP 3: Detect conflicts
    // STEP 4: No conflicts = ACCEPT
    // STEP 5: Resolve conflicts using precedence rules
    // STEP 6: Determine final decision
}
```

### Task 8: Unit Tests (Day 12-14)

**File**: `tests/arbitration_test.go`

- Test P1-P7 precedence rules
- Test all conflict types
- Test Metformin + renal impairment scenario
- Test edge cases (no conflicts, multiple conflicts)
- Test confidence calculation

---

## Test Scenarios

### Scenario 1: Metformin + Renal Impairment

```
Patient: 68yo female, eGFR = 28
Intent: PRESCRIBE Metformin 500mg BID

Inputs:
- FDA SPL: IF CrCl < 30 THEN Avoid (0.95)
- CPIC: eGFR < 30 = Contraindicated (1.00)
- KB-16: eGFR = 28 (CRITICAL)
- Hospital: Allow if eGFR 25-30 with monitoring (0.80)

Conflicts:
- C1: SPL vs CPIC → CPIC wins (P3)
- C2: CPIC vs Hospital → CPIC wins (P6)
- C3: Lab validates rule → ESCALATE (P4)

Decision: BLOCK
Rationale: CPIC DEFINITIVE contraindication cannot be overridden
```

### Scenario 2: Warfarin + Genetic Variant

```
Patient: 55yo male, CYP2C9 *1/*3
Intent: PRESCRIBE Warfarin 5mg daily

Inputs:
- CPIC: CYP2C9 *1/*3 = Reduce dose 25-50%
- SPL Rule: Standard dosing
- No lab conflicts

Conflicts:
- C1: CPIC vs SPL → CPIC wins (P3)

Decision: OVERRIDE
Rationale: Can proceed with CPIC-guided dosing adjustment
```

### Scenario 3: No Conflicts

```
Patient: 30yo female, all labs normal
Intent: PRESCRIBE Amoxicillin 500mg TID

Inputs:
- SPL Rule: No renal adjustment needed
- No authority conflicts
- Labs normal

Conflicts: None

Decision: ACCEPT
Confidence: 0.95
```

---

## File Structure

```
kb-16-lab-interpretation/
├── migrations/
│   └── 004_truth_arbitration.sql       # NEW
├── pkg/
│   └── arbitration/                    # NEW PACKAGE
│       ├── types.go                    # Source, Decision, Conflict types
│       ├── schemas.go                  # Input/Output schemas
│       ├── conflict_detector.go        # Pairwise conflict detection
│       ├── precedence_engine.go        # P1-P7 rule ladder
│       ├── decision_synthesizer.go     # Final decision synthesis
│       └── engine.go                   # Main arbitration engine
├── tests/
│   ├── arbitration_test.go             # NEW
│   └── arbitration_scenarios_test.go   # NEW
```

---

## Integration Points

### Input Sources

| Source | Origin | Interface |
|--------|--------|-----------|
| Canonical Rules | Phase 3b.5 `DraftRule` | KB-5 Drug Interactions service |
| Authority Facts | CPIC, CredibleMeds | External API / cached table |
| Lab Interpretations | KB-16 Phase 3b.6 | `ContextualInterpretationEngine` |
| Regulatory Blocks | FDA SPL extraction | KB-1 Drug Rules service |
| Local Policies | Hospital config | Local database table |

### Output Consumers

| Consumer | Usage |
|----------|-------|
| CDS Hooks | Provide ACCEPT/BLOCK/OVERRIDE decision |
| Alert System | Generate clinical alerts for BLOCK/ESCALATE |
| Audit Log | Record all arbitration decisions |
| UI Dashboard | Display decision rationale to clinicians |

---

## Success Criteria

1. **P1-P7 Rules**: All precedence rules correctly implemented and tested
2. **Conflict Detection**: All 6 conflict types identified correctly
3. **Decision Accuracy**: Metformin scenario produces correct BLOCK decision
4. **Audit Trail**: Complete provenance for every decision
5. **Performance**: < 100ms per arbitration call
6. **Test Coverage**: > 90% for arbitration package

---

## Timeline

| Day | Task | Deliverable |
|-----|------|-------------|
| 1-2 | Migration 004 | Database schema |
| 2-3 | Type definitions | `types.go` |
| 3-4 | Input/Output schemas | `schemas.go` |
| 4-6 | Conflict detector | `conflict_detector.go` |
| 6-8 | Precedence engine | `precedence_engine.go` |
| 8-10 | Decision synthesizer | `decision_synthesizer.go` |
| 10-12 | Main engine | `engine.go` |
| 12-14 | Tests | Full test coverage |

---

## References

- Phase 3b.5: Canonical Rule Engine (`/kb-5-drug-interactions/`)
- Phase 3b.6: KB-16 Lab Reference Ranges (`migrations/003_conditional_reference_ranges.sql`)
- CPIC Guidelines: https://cpicpgx.org/guidelines/
- CredibleMeds: https://crediblemeds.org/
- FDA SPL: https://dailymed.nlm.nih.gov/
