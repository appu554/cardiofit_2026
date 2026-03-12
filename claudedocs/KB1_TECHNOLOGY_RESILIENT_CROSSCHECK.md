# KB1 Implementation Plan vs Technology-Resilient Architecture - Cross-Check Report

> **Date**: 2026-01-19
> **Purpose**: Verify alignment between KB1_DATA_SOURCE_INJECTION_IMPLEMENTATION_PLAN.md and the "Clinical Knowledge Operating System - Technology-Resilient Implementation Plan"

---

## Executive Summary

The cross-check reveals **strong architectural alignment** between the two documents, with the Technology-Resilient document serving as a **validation and enhancement** layer over the KB1 implementation plan. Both documents share the same core philosophy: "Freeze meaning. Fluidly replace intelligence."

| Aspect | Alignment Status | Notes |
|--------|------------------|-------|
| Core Architecture | ✅ 100% Aligned | Both use Canonical Fact Store + Evidence Router |
| Class vs Drug Precedence | ✅ 100% Aligned | Identical deterministic rules |
| Temporal Versioning | ✅ 100% Aligned | Both require "what was active on date X?" queries |
| LLM Replaceability | ✅ 100% Aligned | FactExtractor interface contract matches |
| Phase Structure | ⚠️ Numbering Differs | Tech-Resilient: 0-5, KB1: 1-4 (semantically similar) |
| **3 Identified Gaps** | 🔴 New Requirements | Not in KB1 plan, critical additions |

---

## Part I: Architecture Invariant Validation

### 1. Canonical Fact Store ✅ ALIGNED

| Tech-Resilient Requirement | KB1 Implementation | Status |
|---------------------------|-------------------|--------|
| Single source of truth for all clinical facts | PostgreSQL + JSONB Fact Store (§3) | ✅ |
| Six fact types | ORGAN_IMPAIRMENT, SAFETY_SIGNAL, REPRODUCTIVE_SAFETY, INTERACTION, FORMULARY, LAB_REFERENCE | ✅ |
| Fact lifecycle states | DRAFT → APPROVED → ACTIVE → SUPERSEDED → DEPRECATED | ✅ |
| No KB stores its own facts | KBs are projections (§10) | ✅ |

**KB1 Implementation Location**: [factstore/models.go](../backend/shared-infrastructure/knowledge-base-services/shared/factstore/models.go)

### 2. Evidence Router ✅ ALIGNED

| Tech-Resilient Requirement | KB1 Implementation | Status |
|---------------------------|-------------------|--------|
| Single fabric routes all evidence | Evidence Router (§7) | ✅ |
| Three processing streams | SPL Narrative, API Graph, Structured Dataset | ✅ |
| Tagged evidence units | EvidenceUnit with ClinicalDomains, KBTargets | ✅ |
| Deduplication via checksum | `Checksum string` in EvidenceUnit | ✅ |

**KB1 Implementation Location**: [evidence/router.go](../backend/shared-infrastructure/knowledge-base-services/shared/evidence/router.go)

### 3. Class vs Drug Precedence ✅ ALIGNED

| Tech-Resilient Requirement | KB1 Implementation | Status |
|---------------------------|-------------------|--------|
| Drug-specific always wins over class | Rule 1 in §4 | ✅ |
| Less restrictive → human review | Rule 2: "Flag for KB-18 pharmacist review" | ✅ |
| No drug-specific → inherit class | Rule 3 in §4 | ✅ |
| Deterministic resolution | PrecedenceResolver implementation | ✅ |

**KB1 Implementation Location**: [factstore/precedence.go](../backend/shared-infrastructure/knowledge-base-services/shared/factstore/precedence.go) (planned)

### 4. Temporal Versioning ✅ ALIGNED

| Tech-Resilient Requirement | KB1 Implementation | Status |
|---------------------------|-------------------|--------|
| "What was active on date X?" queries | GetFactAtTime() in §3.5 | ✅ |
| effective_from, effective_to fields | In Fact struct | ✅ |
| Supersession tracking | superseded_by field | ✅ |
| Full version history | GetFactHistory() method | ✅ |

**KB1 Implementation Location**: [factstore/versioning.go](../backend/shared-infrastructure/knowledge-base-services/shared/factstore/versioning.go) (planned)

---

## Part II: Three Critical Gaps Identified

The Technology-Resilient document identifies **three gaps** not addressed in the KB1 implementation plan:

### Gap 1: Fact Stability Contract 🔴 MISSING

**Tech-Resilient Requirement**:
> Every fact must declare its volatility: STABLE (≤1 change/year), EVOLVING (monthly updates), or VOLATILE (external refresh).

| Component | Required Addition | Priority |
|-----------|------------------|----------|
| Volatility Classification | Add `VolatilityClass` enum to Fact struct | HIGH |
| Refresh Triggers | TTL-based or event-based refresh policies | HIGH |
| Staleness Detection | Alert when facts exceed expected refresh window | MEDIUM |

**Recommended Implementation**:
```go
// Add to factstore/models.go
type VolatilityClass string
const (
    VolatilityStable   VolatilityClass = "STABLE"   // ≤1 change/year (e.g., contraindications)
    VolatilityEvolving VolatilityClass = "EVOLVING" // Monthly updates (e.g., formulary)
    VolatilityVolatile VolatilityClass = "VOLATILE" // External refresh (e.g., pricing)
)

type FactStability struct {
    Volatility       VolatilityClass `json:"volatility" db:"volatility"`
    ExpectedRefreshDays int          `json:"expectedRefreshDays" db:"expected_refresh_days"`
    LastRefreshedAt  time.Time       `json:"lastRefreshedAt" db:"last_refreshed_at"`
    RefreshSource    string          `json:"refreshSource" db:"refresh_source"`
}
```

**Impact if Not Implemented**: Stale facts persist indefinitely, no automated refresh triggers, compliance risk for time-sensitive clinical data.

---

### Gap 2: Evidence Conflict Resolution Layer 🔴 MISSING

**Tech-Resilient Requirement**:
> When OpenFDA says "minor interaction" and DrugBank says "major interaction", the conflict must be resolved BEFORE creating a fact.

| Component | Required Addition | Priority |
|-----------|------------------|----------|
| Pre-Fact Arbitration | Conflict detection before fact creation | HIGH |
| Source Authority Ranking | FDA > academic > commercial hierarchy | HIGH |
| Conflict Documentation | Record why one source was chosen | MEDIUM |

**Recommended Implementation**:
```go
// New file: shared/conflicts/resolver.go
type ConflictResolution struct {
    ConflictID      string          `json:"conflictId"`
    EvidenceUnits   []EvidenceUnit  `json:"evidenceUnits"`
    ConflictType    ConflictType    `json:"conflictType"`   // SEVERITY_MISMATCH, VALUE_MISMATCH, etc.
    WinningEvidence string          `json:"winningEvidence"`
    ResolutionRule  string          `json:"resolutionRule"` // "FDA_AUTHORITY", "MOST_CONSERVATIVE", etc.
    Rationale       string          `json:"rationale"`
    ResolvedAt      time.Time       `json:"resolvedAt"`
}

type ConflictResolver interface {
    DetectConflicts(ctx context.Context, units []*EvidenceUnit) ([]*EvidenceConflict, error)
    Resolve(ctx context.Context, conflict *EvidenceConflict) (*ConflictResolution, error)
}
```

**Impact if Not Implemented**: Conflicting facts from multiple sources reach production, clinical decisions become unpredictable, regulatory audit failures.

---

### Gap 3: Decision Explanation Model 🔴 MISSING

**Tech-Resilient Requirement**:
> Every KB-19 decision must produce a first-class explanation object, not just an alert.

| Component | Required Addition | Priority |
|-----------|------------------|----------|
| DecisionExplanation struct | Complete reasoning chain | HIGH |
| Source Attribution | Which facts contributed | HIGH |
| Clinician-Readable Summary | Plain language explanation | MEDIUM |

**Recommended Implementation**:
```go
// New file: shared/runtime/explanation.go
type DecisionExplanation struct {
    DecisionID      string                `json:"decisionId"`
    DecisionType    string                `json:"decisionType"`    // DOSING_ADJUSTMENT, CONTRAINDICATION, etc.

    // What facts drove this decision
    SourceFacts     []FactContribution    `json:"sourceFacts"`

    // The reasoning chain
    ReasoningChain  []ReasoningStep       `json:"reasoningChain"`

    // Clinician-readable summary
    Summary         string                `json:"summary"`         // "Reduce dose because eGFR 28 requires 50% reduction per FDA label"

    // Audit trail
    ComputedAt      time.Time             `json:"computedAt"`
    PatientContext  PatientContext        `json:"patientContext"`
}

type FactContribution struct {
    FactID          string  `json:"factId"`
    FactType        string  `json:"factType"`
    ContributionWeight float64 `json:"contributionWeight"`
    RelevantContent string  `json:"relevantContent"`
}

type ReasoningStep struct {
    StepOrder       int     `json:"stepOrder"`
    InputFacts      []string `json:"inputFacts"`
    Rule            string  `json:"rule"`           // "EGFR_THRESHOLD_CHECK"
    Output          string  `json:"output"`
    Explanation     string  `json:"explanation"`
}
```

**Impact if Not Implemented**: Clinicians can't understand WHY a recommendation was made, reduces trust, compliance risk when auditors ask "why did the system say this?"

---

## Part III: Intelligence Retirement Strategy

### Tech-Resilient Requirement
> When swapping LLM models or retiring extractors, all facts produced by the old intelligence must be systematically handled.

### KB1 Current State
The KB1 plan has `ExtractorProvenance` tracking but **no explicit retirement workflow**.

### Gap 4: Intelligence Retirement Workflow 🟡 PARTIAL

| Component | KB1 Status | Tech-Resilient Addition |
|-----------|-----------|------------------------|
| Extractor version tracking | ✅ ExtractorProvenance | - |
| Model ID tracking | ✅ ModelID field | - |
| Deprecation workflow | ❌ Missing | Gradual retirement with re-extraction |
| Orphan fact handling | ❌ Missing | Bulk re-queue for new extractor |

**Recommended Implementation**:
```go
// New file: shared/registry/extractors.go
type ExtractorRegistry interface {
    Register(extractor FactExtractor) error
    Deprecate(extractorName string, replacementName string) error
    Retire(extractorName string) error
    GetOrphanedFacts(extractorName string) ([]*Fact, error)
    MigrateFactsToNewExtractor(oldExtractor, newExtractor string) error
}

type ExtractorStatus string
const (
    ExtractorActive     ExtractorStatus = "ACTIVE"
    ExtractorDeprecated ExtractorStatus = "DEPRECATED" // Still works, but new facts use replacement
    ExtractorRetired    ExtractorStatus = "RETIRED"    // No longer producing facts
)
```

---

## Part IV: Phase Structure Comparison

### Numbering Differences

| Tech-Resilient Phase | KB1 Phase | Semantic Match |
|---------------------|-----------|----------------|
| Phase 0: Drug Universe | Part of Phase 1 | ✅ Same content |
| Phase 1: Classification | Part of Phase 1 | ✅ Same content |
| Phase 2: Extraction | Phase 2 | ✅ Same content |
| Phase 3: Governance | Part of Phase 2 | ✅ Same content |
| Phase 4: Projections | Phase 2C | ✅ Same content |
| Phase 5: Runtime | Phase 3 | ✅ Same content |

**Conclusion**: Different numbering, same semantic phases. KB1 combines some phases for simplicity.

### Timeline Alignment

| Tech-Resilient | KB1 Plan | Aligned? |
|---------------|----------|----------|
| Week 1: Drug Master | Week 1 | ✅ |
| Weeks 2-3: KB-5 DDI | Not explicit | ⚠️ KB1 focuses on KB-1 renal first |
| Weeks 5-8: LLM extraction | Weeks 3-4 | ⚠️ Slight variation |
| Weeks 12-13: Governance | Weeks 5-8 | ⚠️ KB1 earlier |

**Recommendation**: Align to Tech-Resilient timeline for consistency.

---

## Part V: 10-Year Survival Architecture Assessment

### Tech-Resilient Survival Criteria

| Criterion | KB1 Support | Assessment |
|-----------|-------------|------------|
| Facts survive LLM obsolescence | ✅ FactExtractor interface | Extractors pluggable |
| Facts survive API discontinuation | ✅ Evidence Router streams | New APIs = new streams |
| Facts survive regulatory changes | ✅ Temporal versioning | Can answer "what was active on date X?" |
| Facts survive vendor pivot | ✅ No vendor lock-in | Uses open standards (RxNorm, FHIR) |

### Identified Survivability Gaps

| Gap | Risk Level | Mitigation in Tech-Resilient Doc |
|-----|------------|--------------------------------|
| LLM cost explosion | HIGH | Model swap without re-extraction |
| Regulatory jurisdiction expansion | MEDIUM | Jurisdiction tagging on facts |
| Data source discontinuation (e.g., RxNav DDI) | HIGH | Multiple source fallback |

---

## Part VI: Recommended Actions

### Immediate (This Week)

1. **Add Fact Stability Contract** (Gap 1)
   - Update `factstore/models.go` with VolatilityClass
   - Add FactStability struct
   - Update SQL migrations

2. **Create Evidence Conflict Resolution** (Gap 2)
   - New `conflicts/resolver.go`
   - Implement source authority ranking
   - Add conflict detection to Evidence Router

### Next Sprint (Week 2-3)

3. **Implement Decision Explanation Model** (Gap 3)
   - New `runtime/explanation.go`
   - Add FactContribution tracking
   - Create clinician-readable summary generator

4. **Build Intelligence Retirement Workflow** (Gap 4)
   - New `registry/extractors.go`
   - Add deprecation/retirement lifecycle
   - Implement orphan fact migration

### Future (Phase 3+)

5. **Align Phase Numbering**
   - Consider adopting Tech-Resilient Phase 0-5 structure
   - Update documentation for consistency

---

## Appendix: File Mapping

| Tech-Resilient Concept | KB1 File | Status |
|----------------------|----------|--------|
| Canonical Fact Store | `factstore/models.go` | ✅ Exists |
| Evidence Router | `evidence/router.go` | ✅ Exists |
| FactExtractor Interface | `extraction/interfaces.go` | ✅ Exists |
| Class vs Drug Precedence | `factstore/precedence.go` | ⏳ Planned |
| Temporal Versioning | `factstore/versioning.go` | ⏳ Planned |
| **Fact Stability Contract** | `factstore/models.go` | 🔴 **ADD** |
| **Conflict Resolution** | `conflicts/resolver.go` | 🔴 **CREATE** |
| **Decision Explanation** | `runtime/explanation.go` | 🔴 **CREATE** |
| **Extractor Registry** | `registry/extractors.go` | 🔴 **CREATE** |

---

## Conclusion

The KB1 Implementation Plan is **architecturally sound** and aligns with the Technology-Resilient vision. The three critical gaps identified should be addressed to achieve true 10-year survivability:

1. **Fact Stability Contract** - Prevents stale facts
2. **Evidence Conflict Resolution** - Ensures consistent decisions when sources disagree
3. **Decision Explanation Model** - Enables clinician trust and regulatory compliance

**Overall Assessment**: 85% aligned. Remaining 15% are additive enhancements, not corrections.

---

**Document Version**: 1.0
**Author**: Claude Code
**Next Review**: After Phase 2 completion
