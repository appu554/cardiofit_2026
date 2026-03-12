# KB1 Data Source Injection - Implementation Crosscheck Report

**Date**: 2026-01-19
**Scope**: Compare Implementation Plan vs Actual Implementation vs User Feedback Gaps

---

## Executive Summary

This report crosschecks the implementation plan against what was actually built and identifies the remaining gaps from user feedback.

### Quick Status Overview

| Component | Plan Section | Implementation Status | Notes |
|-----------|--------------|----------------------|-------|
| Drug Master Table (Layer 0) | §2 | ✅ **COMPLETE** | `drugmaster/models.go` |
| Canonical Fact Store | §3 | ✅ **COMPLETE** | `factstore/models.go`, SQL migrations |
| Class vs Drug Precedence | §4 | ⚠️ **PARTIAL** | Scope field present, resolution logic missing |
| Evidence Router | §7 | ✅ **COMPLETE** | `evidence/router.go` |
| FactExtractor Interface | §8 | ✅ **COMPLETE** | `extraction/interfaces.go` |
| Confidence-Driven Governance | §9 | ✅ **COMPLETE** | `governance/engine.go` |
| KB Projection Definitions | §10 | ❌ **NOT IMPLEMENTED** | Gap identified |
| KB-19 Runtime Arbitration | §11 | ❌ **NOT IMPLEMENTED** | Gap identified |
| Evolution Guardrails | §12 | ❌ **NOT IMPLEMENTED** | Gap identified |
| DI Container | §16 | ✅ **COMPLETE** | `di/container.go`, `di/providers.go` |
| Data Source Interfaces | §15 | ✅ **COMPLETE** | `datasources/interfaces.go` |
| RxNav Client | §15 | ✅ **COMPLETE** | `datasources/rxnav/client.go` |
| Redis Cache | §15 | ✅ **COMPLETE** | `datasources/cache/redis.go` |
| Renal Classification | N/A | ✅ **COMPLETE** | `classification/renal/classifier.go` |

---

## Part 1: What's Been Implemented (✅ NAILED)

### 1.1 Drug Master Table (Layer 0)

**File**: [drugmaster/models.go](../backend/shared-infrastructure/knowledge-base-services/shared/drugmaster/models.go)

| Plan Requirement | Implemented | Evidence |
|------------------|-------------|----------|
| RxCUI as primary key | ✅ | `RxCUI string` field |
| Drug names (canonical, generic, brand) | ✅ | `DrugName`, `GenericName`, `BrandNames []string` |
| RxNorm TTY classification | ✅ | `RxNormTTY` enum (IN, MIN, SCDC, SCD, SBD, GPCK, BPCK) |
| ATC codes | ✅ | `ATCCodes []string` |
| Cross-references (NDCs, SPL, SNOMED) | ✅ | All present |
| Drug status (ACTIVE, RETIRED, REMAPPED) | ✅ | `DrugStatus` enum |
| Repository interface | ✅ | Complete CRUD + clinical queries |
| Renal/Hepatic relevance tagging | ✅ | `RenalRelevance`, `HepaticRelevance` enums |

### 1.2 Canonical Fact Store

**File**: [factstore/models.go](../backend/shared-infrastructure/knowledge-base-services/shared/factstore/models.go)

| Plan Requirement | Implemented | Evidence |
|------------------|-------------|----------|
| Six Fact Types | ✅ | `ORGAN_IMPAIRMENT`, `SAFETY_SIGNAL`, `REPRODUCTIVE_SAFETY`, `INTERACTION`, `FORMULARY`, `LAB_REFERENCE` |
| Fact Lifecycle States | ✅ | `DRAFT → APPROVED → ACTIVE → SUPERSEDED → DEPRECATED` |
| Scope (Drug vs Class) | ✅ | `FactScope` enum |
| Confidence Model | ✅ | `FactConfidence` with Score, Band, Signals, SourceDiversity |
| Temporal Versioning | ✅ | `EffectiveFrom`, `EffectiveTo`, `SupersededBy`, `Version` |
| Provenance Tracking | ✅ | `ExtractorID`, `ExtractorVersion`, `SourceURL`, `EvidenceID` |
| Jurisdiction Support | ✅ | `Jurisdictions []string`, `RegulatoryBody` |
| Content Type Schemas | ✅ | All 6 content types defined |

### 1.3 Evidence Router

**File**: [evidence/router.go](../backend/shared-infrastructure/knowledge-base-services/shared/evidence/router.go)

| Plan Requirement | Implemented | Evidence |
|------------------|-------------|----------|
| Single factory for all evidence | ✅ | `Router` struct with stream registry |
| Pluggable processing streams | ✅ | `ProcessingStream` interface |
| Deduplication | ✅ | `DeduplicationEnabled`, checksum tracking |
| Batch processing | ✅ | `RouteAll()`, `RouteBatch()` |
| Concurrency control | ✅ | `MaxConcurrency` with semaphore |
| Retry logic | ✅ | `MaxRetries`, `RetryDelayMs` |
| Metrics | ✅ | `RouterMetrics` struct |

### 1.4 FactExtractor Interface

**File**: [extraction/interfaces.go](../backend/shared-infrastructure/knowledge-base-services/shared/extraction/interfaces.go)

| Plan Requirement | Implemented | Evidence |
|------------------|-------------|----------|
| Universal extractor interface | ✅ | `FactExtractor` interface |
| Extraction result with facts | ✅ | `ExtractionResult` struct |
| Confidence signals | ✅ | `ConfidenceSignal` in extraction |
| Extractor metadata | ✅ | `ExtractorMetadata` struct |
| Source verification | ✅ | `SourceVerifier` interface |

### 1.5 Confidence-Driven Auto-Governance

**File**: [governance/engine.go](../backend/shared-infrastructure/knowledge-base-services/shared/governance/engine.go)

| Plan Requirement | Implemented | Evidence |
|------------------|-------------|----------|
| ≥0.85 auto-approve | ✅ | `AutoApproveThreshold: 0.85` |
| 0.65-0.84 queue for review | ✅ | `ReviewThreshold: 0.65` |
| <0.65 auto-reject | ✅ | Below threshold → `DEPRECATED` |
| Review queue | ✅ | `ReviewQueueItem`, `GetReviewQueue()` |
| Manual approval/rejection | ✅ | `ApproveFactManually()`, `RejectFactManually()` |
| Escalation support | ✅ | `EscalateFact()`, `MaxEscalations` |
| Processing loop | ✅ | `processLoop()` with `ProcessInterval` |
| Metrics | ✅ | `EngineMetrics`, `QueueStats` |

### 1.6 DI Container & Data Sources

**Files**:
- [di/container.go](../backend/shared-infrastructure/knowledge-base-services/shared/di/container.go)
- [di/providers.go](../backend/shared-infrastructure/knowledge-base-services/shared/di/providers.go)
- [datasources/interfaces.go](../backend/shared-infrastructure/knowledge-base-services/shared/datasources/interfaces.go)
- [datasources/rxnav/client.go](../backend/shared-infrastructure/knowledge-base-services/shared/datasources/rxnav/client.go)
- [datasources/cache/redis.go](../backend/shared-infrastructure/knowledge-base-services/shared/datasources/cache/redis.go)

| Plan Requirement | Implemented | Evidence |
|------------------|-------------|----------|
| DI Container | ✅ | `Container` with lifecycle management |
| Factory providers | ✅ | `Provider` interface |
| Lazy initialization | ✅ | Factory pattern |
| Data source interfaces | ✅ | `RxNavClient`, `DailyMedClient`, `LLMClient`, `Cache` |
| RxNav client | ✅ | Full implementation with rate limiting, caching |
| Redis cache | ✅ | `RedisCache` + `MemoryCache` fallback |

---

## Part 2: Gaps Identified (❌ NOT IMPLEMENTED)

### 2.1 Gap: Fact Stability Contract (User Feedback Gap 1)

**What's Missing**: Stability metadata to define how volatile or stable a fact is.

```go
// REQUIRED ADDITION to Fact struct:
type FactStability struct {
    ExpectedHalfLife         time.Duration `json:"expectedHalfLife"`         // Expected time until obsolete
    ChangeTriggers           []string      `json:"changeTriggers"`           // What would cause update
    AutoRevalidationInterval int           `json:"autoRevalidationDays"`     // Days between revalidation
    StabilityClass           string        `json:"stabilityClass"`           // IMMUTABLE, SLOW_EVOLVING, VOLATILE
}
```

**Why Important**: Enables scheduled re-extraction and prevents stale facts from persisting indefinitely.

---

### 2.2 Gap: Evidence Conflict Resolution Layer (User Feedback Gap 2)

**What's Missing**: Pre-fact arbitration when multiple sources disagree.

**Required Implementation**:
```go
// shared/conflicts/resolver.go
type ConflictResolver interface {
    // DetectConflicts identifies disagreements between evidence units
    DetectConflicts(ctx context.Context, evidences []*evidence.EvidenceUnit) ([]Conflict, error)

    // Resolve determines the winning evidence and creates resolution record
    Resolve(ctx context.Context, conflict Conflict) (*Resolution, error)
}

type Conflict struct {
    ConflictID      string
    EvidenceA       *evidence.EvidenceUnit
    EvidenceB       *evidence.EvidenceUnit
    ConflictType    string  // VALUE_MISMATCH, THRESHOLD_OVERLAP, CONTRADICTION
    Field           string  // Which field disagrees
    Impact          string  // Clinical impact assessment
}

type Resolution struct {
    WinningEvidence string
    Method          string  // RECENCY, SOURCE_PRIORITY, CONSERVATIVE, HUMAN_REVIEW
    Rationale       string
    AuditTrail      []string
}
```

**Why Important**: Prevents conflicting facts from reaching production. Currently, if DailyMed says "reduce 50%" and DrugBank says "reduce 25%", both could become active facts.

---

### 2.3 Gap: KB-19 Decision Explanation Model (User Feedback Gap 3)

**What's Missing**: First-class explanation objects for runtime decisions.

**Required Implementation**:
```go
// shared/runtime/explanation.go
type DecisionExplanation struct {
    // The decision that was made
    Decision     string `json:"decision"`      // BLOCK, WARN, ADJUST, ALLOW

    // Facts that contributed
    FactsUsed    []FactContribution `json:"factsUsed"`

    // The reasoning chain
    ReasoningChain []ReasoningStep `json:"reasoningChain"`

    // Counterfactuals
    Counterfactuals []Counterfactual `json:"counterfactuals"`
}

type FactContribution struct {
    FactID       string  `json:"factId"`
    FactType     string  `json:"factType"`
    KB           string  `json:"kb"`
    Weight       float64 `json:"weight"`       // How much this fact influenced decision
    WasDecisive  bool    `json:"wasDecisive"`  // Would decision change without this?
}

type Counterfactual struct {
    Condition    string `json:"condition"`     // "If eGFR were 35 instead of 28"
    AlternateDecision string `json:"alternateDecision"`
}
```

**Why Important**: Clinicians need to understand WHY a decision was made, not just WHAT was decided.

---

### 2.4 Gap: Intelligence Retirement Strategy / Version Registry (User Feedback Gap 4)

**What's Missing**: Formal retirement pathway for extractors.

**Required Implementation**:
```go
// shared/registry/extractors.go
type ExtractorRegistry struct {
    extractors map[string]*ExtractorVersion
}

type ExtractorVersion struct {
    ExtractorID        string    `json:"extractorId"`
    Version            string    `json:"version"`
    Status             string    `json:"status"`  // ACTIVE, DEPRECATED, RETIRED

    // Retirement tracking
    DeprecatedAt       *time.Time `json:"deprecatedAt,omitempty"`
    RetireAt           *time.Time `json:"retireAt,omitempty"`
    ReplacedBy         string     `json:"replacedBy,omitempty"`

    // Stats
    FactsProduced      int64      `json:"factsProduced"`
    FactsStillActive   int64      `json:"factsStillActive"`
    LastUsed           time.Time  `json:"lastUsed"`

    // Migration
    MigrationPolicy    string     `json:"migrationPolicy"` // RE_EXTRACT, GRANDFATHER, MANUAL_REVIEW
}

// DEPRECATION_GRACE_PERIOD = 90 days
// RETIREMENT_REQUIRES: all facts migrated or superseded
```

**Why Important**: When you upgrade from GPT-4 to Claude-5, you need a formal way to retire the old extractor without invalidating its facts.

---

### 2.5 Gap: KB Projection Definitions (Plan Section 10)

**What's Missing**: The entire projection layer that makes KBs read-models over the Fact Store.

**Required Implementation**:
```go
// shared/projections/definitions.go
package projections

type ProjectionDefinition struct {
    KBID         string
    Name         string
    FactTypes    []factstore.FactType
    Filters      []Filter
    Transformers []Transformer
}

type Filter struct {
    Field    string
    Operator string
    Value    interface{}
}

type Transformer struct {
    Name   string
    Config map[string]interface{}
}

// Standard projections
var (
    KB1Projection = ProjectionDefinition{
        KBID:      "KB-1",
        Name:      "Renal Dosing Rules",
        FactTypes: []factstore.FactType{factstore.FactTypeOrganImpairment},
        Filters: []Filter{
            {Field: "content.organSystem", Operator: "=", Value: "RENAL"},
            {Field: "status", Operator: "=", Value: "ACTIVE"},
        },
    }
    // ... KB4, KB5, KB6, KB16 projections
)
```

**Why Important**: Without projections, each KB stores its own copy of facts, violating the anti-redundancy principle.

---

### 2.6 Gap: KB-19 Runtime Arbitration (Plan Section 11)

**What's Missing**: The decision engine that reasons over facts at runtime.

**Required Implementation**:
```go
// shared/runtime/arbitration.go
package runtime

type ArbitrationEngine struct {
    factAggregator *FactAggregator
    decisionRules  []DecisionRule
}

type ArbitrationRequest struct {
    PatientState    PatientState    `json:"patientState"`
    Order           MedicationOrder `json:"order"`
    ClinicalContext ClinicalContext `json:"clinicalContext"`
}

type ArbitrationResponse struct {
    Decision       Decision        `json:"decision"`      // BLOCK, WARN, ADJUST, ALLOW
    Reasoning      []ReasoningStep `json:"reasoning"`
    Alternatives   []Alternative   `json:"alternatives,omitempty"`
    OverridePolicy OverridePolicy  `json:"overridePolicy"`
}

type Decision string
const (
    DecisionBlock  Decision = "BLOCK"
    DecisionWarn   Decision = "WARN"
    DecisionAdjust Decision = "ADJUST"
    DecisionAllow  Decision = "ALLOW"
)

// RULE: CONTRAINDICATION > ADJUSTMENT > MONITORING > ALLOW
```

**Why Important**: KB-19 is the "Judge" that makes runtime clinical decisions. Without it, facts have no runtime application.

---

### 2.7 Gap: Evolution Guardrails (Plan Section 12)

**What's Missing**: LLM policy configuration and swap protection mechanisms.

**Required Implementation**:
```go
// shared/policy/llm_policy.go
type LLMPolicy struct {
    // Deployment restrictions
    AllowedContexts    []string // "batch_offline", "summarization"
    ProhibitedContexts []string // "runtime_decision", "bedside_recommendation"

    // Audit requirements
    PromptVersioning   bool
    OutputValidation   bool
    HumanSamplingRate  float64  // 5% = 0.05

    // Swap protection
    ModelRegistry      map[string]ModelConfig
}

type ModelConfig struct {
    ModelID          string
    Provider         string
    Status           string  // ACTIVE, DEPRECATED, RETIRED
    SwappableFor     []string // What models can replace this
    ValidationSuite  string  // Test suite ID to run on swap
}
```

**Why Important**: Protects against LLM vendor lock-in and ensures model swaps don't break production.

---

## Part 3: Implementation Completeness Matrix

### Phase 1 (Foundation) - **100% COMPLETE**

| Component | Status | File |
|-----------|--------|------|
| go.mod | ✅ | `shared/go.mod` |
| Drug Master Table | ✅ | `drugmaster/models.go` |
| Fact Store Models | ✅ | `factstore/models.go` |
| SQL Migrations | ✅ | `factstore/migrations/001_canonical_factstore.sql` |
| Data Source Interfaces | ✅ | `datasources/interfaces.go` |
| RxNav Client | ✅ | `datasources/rxnav/client.go` |
| Redis Cache | ✅ | `datasources/cache/redis.go` |
| Evidence Models | ✅ | `evidence/models.go` |
| Evidence Router | ✅ | `evidence/router.go` |
| Extractor Interfaces | ✅ | `extraction/interfaces.go` |
| DI Container | ✅ | `di/container.go` |
| DI Providers | ✅ | `di/providers.go` |
| Governance Engine | ✅ | `governance/engine.go` |
| Renal Classifier | ✅ | `classification/renal/classifier.go` |

### Phase 2 (Anti-Redundancy) - **50% COMPLETE**

| Component | Status | Required |
|-----------|--------|----------|
| Fact Stability Contract | ❌ | Add to `factstore/models.go` |
| Evidence Conflict Resolution | ❌ | New `conflicts/resolver.go` |
| Decision Explanation Model | ❌ | New `runtime/explanation.go` |
| Intelligence Retirement | ❌ | New `registry/extractors.go` |
| KB Projections | ❌ | New `projections/definitions.go` |

### Phase 3 (Runtime) - **0% COMPLETE**

| Component | Status | Required |
|-----------|--------|----------|
| KB-19 Arbitration Engine | ❌ | New `runtime/arbitration.go` |
| Decision Rules Engine | ❌ | New `runtime/rules.go` |
| Patient State Model | ❌ | New `runtime/patient.go` |
| Override Policy | ❌ | New `runtime/override.go` |

### Phase 4 (Evolution Protection) - **0% COMPLETE**

| Component | Status | Required |
|-----------|--------|----------|
| LLM Policy Configuration | ❌ | New `policy/llm_policy.go` |
| Model Registry | ❌ | New `policy/model_registry.go` |
| Temporal Queries | ❌ | Add to `factstore/repository.go` |
| Jurisdiction Queries | ❌ | Add to `factstore/repository.go` |

---

## Part 4: Recommended Implementation Order

Based on the gaps identified and their dependencies:

### Phase 2A: Fact Stability & Conflict Resolution (Week 1)
1. Add `FactStability` to `factstore/models.go`
2. Create `conflicts/resolver.go`
3. Update SQL migrations for stability metadata

### Phase 2B: Intelligence Registry (Week 2)
1. Create `registry/extractors.go`
2. Add extractor lifecycle management
3. Implement deprecation/retirement workflow

### Phase 2C: KB Projections (Week 3)
1. Create `projections/definitions.go`
2. Implement projection executors
3. Wire KB-1 through KB-16 projections

### Phase 3: Runtime Layer (Weeks 4-5)
1. Create `runtime/arbitration.go`
2. Implement decision rules engine
3. Create `runtime/explanation.go`
4. Build KB-19 integration

### Phase 4: Evolution Protection (Week 6)
1. Create `policy/llm_policy.go`
2. Implement model swap validation
3. Add temporal queries to repository

---

## Part 5: Files Inventory

### Existing Files (14 total)

```
shared/
├── go.mod
├── classification/
│   └── renal/
│       └── classifier.go
├── datasources/
│   ├── interfaces.go
│   ├── cache/
│   │   └── redis.go
│   └── rxnav/
│       └── client.go
├── di/
│   ├── container.go
│   └── providers.go
├── drugmaster/
│   └── models.go
├── evidence/
│   ├── models.go
│   └── router.go
├── extraction/
│   └── interfaces.go
├── factstore/
│   ├── models.go
│   └── migrations/
│       └── 001_canonical_factstore.sql
└── governance/
    └── engine.go
```

### Required New Files (10 total)

```
shared/
├── conflicts/
│   └── resolver.go          # Evidence conflict resolution
├── projections/
│   └── definitions.go       # KB projection definitions
├── registry/
│   └── extractors.go        # Extractor version registry
├── runtime/
│   ├── arbitration.go       # KB-19 decision engine
│   ├── explanation.go       # Decision explanation model
│   ├── patient.go           # Patient state model
│   ├── rules.go             # Decision rules engine
│   └── override.go          # Override policy
└── policy/
    ├── llm_policy.go        # LLM deployment policy
    └── model_registry.go    # Model swap registry
```

---

## Conclusion

**Phase 1 is 100% complete** - all foundational infrastructure is in place.

**Remaining work** focuses on:
1. **Stability & Conflict Resolution** - Preventing stale/conflicting facts
2. **Intelligence Lifecycle** - Managing extractor retirement
3. **KB Projections** - Eliminating redundancy across KBs
4. **Runtime Layer** - Enabling KB-19 clinical decisions
5. **Evolution Protection** - Ensuring system survives LLM changes

The architecture is sound. The remaining gaps are about **completeness**, not **correctness**.
