# Clinical Knowledge Core - Governance

> **"This repository is your clinical constitution. Changes here are rare, reviewed, and signed."**
> — CTO/CMO Joint Statement

## Purpose

This repository contains **all clinical truth** for the Clinical Reasoning Platform:
- CQL logic libraries
- Terminology definitions and value sets
- Clinical calculators
- Guideline implementations
- Regional adaptations

**This repository contains NO:**
- Application code
- User interface code
- AI/ML models
- Patient data (PHI)

---

## Non-Negotiable Rules

### Rule 1: Tier-0.5 is BUILD-TIME AUTHORITATIVE ONLY

**🔴 CRITICAL: Runtime SHALL NEVER expand or mutate terminology.**

Tier-0.5 stores:
- CodeSystem definitions (snapshotted, version-pinned)
- ValueSet definitions (intensional, not expanded)
- Region overlay mappings

Expanded value sets belong **ONLY** in `/build/valueset-expansion/` outputs.

```
✅ CORRECT: Runtime loads pre-expanded sets from signed artifacts
❌ WRONG:   Runtime queries terminology server to expand value sets
```

**Rationale:** Runtime terminology mutation creates:
- Non-deterministic execution
- Audit trail breaks
- Regional inconsistency
- Regulatory violations

---

### Rule 2: Graph-Native Subsumption Over JSON Explosion

**Class membership (e.g., "is ACE inhibitor") SHALL be resolved via graph subsumption, not by enumerating children in JSON unless legally required.**

```
✅ CORRECT: ValueSet defines root concept + "descendants-of" relationship
           Runtime resolves membership via graph query

❌ WRONG:   ValueSet enumerates 50,000 child codes in JSON array
```

**Exceptions (legal requirements):**
- CMS eCQM submissions requiring explicit code lists
- Regulatory audits requiring static snapshots

When enumeration is required, document with:
```json
{
  "enumeration_reason": "CMS_SUBMISSION_2025",
  "generated_from": "snomed:372568004 (ACE inhibitor)",
  "generated_date": "2024-12-12",
  "code_count": 47523
}
```

---

### Rule 3: Guideline Provenance is Mandatory

**Every file in `tier-4-guidelines/` MUST have provenance metadata.**

Required structure:
```
tier-4-guidelines/
└── {source}/
    └── {guideline}/
        ├── logic.cql           # Clinical logic
        ├── evidence.md         # Source documentation
        ├── version.lock        # Version pinning
        └── adaptation.md       # Local modifications (if any)
```

**version.lock format:**
```yaml
guideline_id: "WHO-ANC-2020"
source_url: "https://www.who.int/publications/i/item/9789240020306"
source_version: "2020-06-17"
downloaded_date: "2024-12-01"
sha256: "abc123..."
adaptations:
  - type: "threshold"
    original: "BMI > 30"
    adapted: "BMI > 25"
    rationale: "Asian population cutoff per WHO 2004"
    approved_by: "CMO"
    approved_date: "2024-12-10"
```

---

### Rule 4: No Silent Threshold Changes

**Any change to a clinical threshold requires:**
1. Evidence citation
2. Clinical reviewer approval
3. Explicit documentation in `adaptation.md`
4. Version bump (see Versioning)

---

### Rule 5: Tier 0-1 Grammar Lock

**Tier 0 and Tier 1 libraries define the GRAMMAR of clinical logic, not clinical meaning.**

They MUST NOT contain:
- Disease names used as logic (e.g., `define "HasDiabetes"`)
- Clinical thresholds (e.g., `HbA1c > 9%`, `BP > 140/90`)
- Regional adaptations (e.g., Asian BMI cutoffs)
- Drug knowledge (e.g., Metformin contraindications)
- Guideline semantics (e.g., ADA, WHO, CMS decision logic)

**Allowed in Tier 0-1:**
- FHIR resource traversal (reading `Observation.interpretation`, `Encounter.class`)
- Date/time interval operations
- Null-safe access patterns
- Status code filtering
- LOINC/SNOMED code constants for filtering (not interpretation)

**Any change to Tier 0 or Tier 1:**
- Requires **MAJOR** version bump
- Requires CTO + CMO approval with 48h review
- Requires full regression test of all downstream tiers

```
✅ CORRECT: define function IsHigh(obs): obs.interpretation.coding.code = 'H'
            (Reading FHIR data)

❌ WRONG:   define "HighHbA1c": HbA1cResult > 9.0
            (Clinical threshold - belongs in Tier 3+)
```

---

### Rule 6: Explicit Version Pinning

**All CQL library includes MUST specify explicit versions.**

```cql
// ❌ WRONG - Ambiguous, non-deterministic
include FHIRHelpers called FHIRHelpers

// ✅ CORRECT - Explicit, auditable
include FHIRHelpers version '4.0.1' called FHIRHelpers
```

**Rationale:** Version pinning ensures:
- Reproducible builds across environments
- Clear audit trails for regulatory compliance
- Prevention of silent dependency drift

---

## Condition Coverage Index Rule (HARD GATE)

**Before authoring ANY new clinical logic:**

1. The condition MUST exist in `coverage-index.yaml`
2. Coverage status MUST be `not_covered` or `partially_covered`
3. Existing authoritative sources MUST be documented
4. `authoring_allowed` MUST be set to `true` (requires CMO approval)
5. Clinical owner MUST be assigned
6. Gap rationale MUST justify custom authoring vs. importing existing guidelines

**Pull requests violating this rule will NOT be reviewed.**

This single rule prevents 90% of future clinical debt.

---

## Change Control

### Tier-Based Versioning (REQUIRED)

| Change Type | Version Bump | Review Required |
|-------------|--------------|-----------------|
| Tier-0 or Tier-0.5 | **MAJOR** | CTO + CMO + 48h review |
| Tier-1 or Tier-2 | **MINOR** | Clinical + Technical |
| Tier-3 | **MINOR** | Clinical + Technical |
| Tier-4 or Tier-5 | **PATCH** or **MINOR** | Clinical |
| Build/CI only | **PATCH** | Technical |

### Pull Request Requirements

**All PRs to `main` require:**

| Requirement | Tier 0-2 | Tier 3-5 | Build/CI |
|-------------|----------|----------|----------|
| Clinical Reviewer | ✅ Required | ✅ Required | ❌ |
| Technical Reviewer | ✅ Required | ✅ Required | ✅ Required |
| Evidence Link | ✅ Required | ✅ Required | ❌ |
| Test Coverage | ✅ ≥80% | ✅ ≥70% | ✅ ≥60% |
| CMO Sign-off | ✅ Required | ⚠️ For thresholds | ❌ |

---

**Document Version:** 1.1.0
**Effective Date:** 2024-12-13
**Approved By:** CTO, CMO

### Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2024-12-12 | Initial release |
| 1.1.0 | 2024-12-13 | Added Rule 5 (Grammar Lock) and Rule 6 (Version Pinning) |
