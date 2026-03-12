# CMS eCQM Compilation Notes

**Date**: 2025-12-14
**Compiler**: cqframework/cql-translation-service v3.22.0 (Docker)
**Status**: First Compilation Attempt - Expected Failures

---

## Executive Summary

All 12 CQL files failed compilation as **expected**. This is correct per CTO/CMO directive.

```
╔═══════════════════════════════════════════════════════════════════════════════╗
║  COMPILATION RESULTS                                                          ║
╠═══════════════════════════════════════════════════════════════════════════════╣
║  Total Files:          12                                                     ║
║  Successfully Compiled: 0                                                     ║
║  Failed:               12                                                     ║
║                                                                               ║
║  Error Categories:                                                            ║
║    ├─ Expected Dependency Errors:  10 unique libraries missing                ║
║    ├─ Real Grammar Errors:          3 files with syntax issues                ║
║    └─ Version Mismatches:           2 (FHIRHelpers, QICore)                   ║
╚═══════════════════════════════════════════════════════════════════════════════╝
```

---

## Error Classification

### Category 1: Expected Dependency Errors ✅

**These are expected.** CMS eCQM measures require vendor libraries we haven't imported yet.

| Missing Library | Required Version | Required By | Action |
|-----------------|-----------------|-------------|--------|
| FHIRHelpers | 4.4.000 | CMS122, CMS134, CMS165, CMS2 | Import from CQF |
| QICoreCommon | 2.1.000 | CMS122, CMS134 | Import from CQF |
| SupplementalDataElements | 3.5.000 | CMS122, CMS134, CMS165, CMS2 | Import from CMS |
| Status | 1.8.000 | CMS122 | Import from CQF |
| CQMCommon | 2.2.000 | CMS134 | Import from CQF |
| CumulativeMedicationDuration | 4.1.000 | CMS122 | Import from CQF |
| AdultOutpatientEncounters | 4.11.000 | CMS122 | Import from CQF |
| AdvancedIllnessandFrailty | 1.16.000 | CMS122 | Import from CQF |
| Hospice | 6.12.000 | CMS122, CMS134 | Import from CQF |
| PalliativeCare | 1.11.000 | CMS122, CMS134 | Import from CQF |

**Example Error Message:**
```xml
<annotation message="Could not load source for library FHIRHelpers, version 4.4.000."
            errorType="include" errorSeverity="error"
            targetIncludeLibraryId="FHIRHelpers"
            targetIncludeLibraryVersionId="4.4.000"/>
```

**Resolution**: Import these as vendor code into `tier-2-cqm-infra/vendor/`

---

### Category 2: Real Grammar Errors ⚠️

**These need fixing.** Our custom CQL files have syntax issues.

#### 2.1 FHIRHelpers.cql (Tier 0)

**Issue**: The `value.value` accessor pattern is not valid CQL grammar.

```
Lines 30, 33, 36, 39, 42, 45, 48... (89+ occurrences)
Message: "Syntax error at ."
```

**Current Code (Invalid)**:
```cql
define function ToString(value FHIR.uuid): System.String
  value.value
```

**Correct CQL Pattern**:
```cql
define function ToString(value FHIR.uuid):
  if value is null then null
  else value.value as String
```

**Root Cause**: Our FHIRHelpers was created as a simplified template, not the official HL7 version.

**Resolution**: Replace with official FHIRHelpers 4.4.000 from eCQI as vendor code.

---

#### 2.2 IntervalHelpers.cql (Tier 1)

**Multiple Issues**:

| Line | Error | Issue |
|------|-------|-------|
| 41 | "Syntax error at codesystem" | Misplaced codesystem declaration |
| 126 | "Syntax error at year" | Invalid date component syntax |
| 133 | "Could not find type: quarter" | 'quarter' is not a valid CQL date precision |
| 318-326 | "Syntax error at days" | Invalid duration arithmetic |
| 410 | "Circular reference" | Self-referential Range function |

**Example - Invalid Quarter Syntax**:
```cql
// Our code (line 133) - INVALID
define function QuarterOfYear(value DateTime): Integer
  quarter from value    // 'quarter' is not a CQL keyword!
```

**Resolution**:
- Remove `quarter` functions (not supported in CQL 1.5)
- Fix date arithmetic to use `day`, `month`, `year` only
- Refactor Range function to avoid circular reference

---

#### 2.3 Other Tier 1-2 Files

Files with include dependency failures (cascade from above):
- MedicationHelpers.cql - Depends on FHIRHelpers
- EncounterHelpers.cql - Depends on FHIRHelpers
- ObservationHelpers.cql - Depends on FHIRHelpers
- CQMCommon.cql - Depends on Tier 1 libraries
- CommonClinicalCodes.cql - Needs codesystem declarations
- ClinicalObservationHelpers.cql - Cascading failures

---

### Category 3: Version Mismatches ⚡

**Architecture Decision Required**

| Component | Our Version | CMS Required | Impact |
|-----------|-------------|--------------|--------|
| FHIRHelpers | 4.0.1 | 4.4.000 | Major - API differences |
| FHIR Model | FHIR 4.0.1 | QICore 4.1.1 | Major - QI-Core extends FHIR |

**Analysis**:

CMS eCQM measures use:
```cql
using QICore version '4.1.1'
include FHIRHelpers version '4.4.000' called FHIRHelpers
```

Our code uses:
```cql
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1' called FHIRHelpers
```

**QI-Core vs FHIR**:
- QI-Core is a FHIR R4 **profile** (subset + extensions)
- CMS measures target QI-Core, not raw FHIR
- We need QI-Core ModelInfo for CMS measure compilation

**Resolution Options**:
1. **Option A**: Import QI-Core ModelInfo and use QI-Core profiles (recommended for CMS)
2. **Option B**: Maintain separate Tier-4 paths for QI-Core and raw FHIR measures
3. **Option C**: Adapt CMS measures to raw FHIR (violates vendor code policy)

---

## Dependency Graph

```
CMS122-DiabetesHbA1c.cql
├── FHIRHelpers 4.4.000 ❌ (missing)
├── QICoreCommon 2.1.000 ❌ (missing)
├── SupplementalDataElements 3.5.000 ❌ (missing)
├── Status 1.8.000 ❌ (missing)
├── CumulativeMedicationDuration 4.1.000 ❌ (missing)
├── AdultOutpatientEncounters 4.11.000 ❌ (missing)
├── AdvancedIllnessandFrailty 1.16.000 ❌ (missing)
├── Hospice 6.12.000 ❌ (missing)
└── PalliativeCare 1.11.000 ❌ (missing)

CMS134-DiabeticNephropathy.cql
├── FHIRHelpers 4.4.000 ❌ (missing)
├── SupplementalDataElements 3.5.000 ❌ (missing)
├── CQMCommon 2.2.000 ❌ (missing)
├── Hospice 6.12.000 ❌ (missing)
├── PalliativeCare 1.11.000 ❌ (missing)
└── QICoreCommon 2.1.000 ❌ (missing)
```

---

## Next Steps (CTO/CMO Gate)

### Immediate Actions

1. **Fix Real Grammar Errors**
   - [ ] Replace FHIRHelpers.cql with official HL7 version 4.4.000
   - [ ] Fix IntervalHelpers.cql 'quarter' and circular reference issues
   - [ ] Remove unsupported CQL syntax

2. **Import Vendor Libraries**
   - [ ] Download FHIRHelpers 4.4.000 from [CQF GitHub](https://github.com/cqframework/cqf-content)
   - [ ] Download QICoreCommon 2.1.000
   - [ ] Download SupplementalDataElements 3.5.000
   - [ ] Download remaining 7 vendor libraries

3. **Resolve Version Strategy**
   - [ ] CTO/CMO decision on QI-Core vs raw FHIR approach
   - [ ] Document version pinning policy

### Validation Criteria

Re-run compilation when:
- All 10 vendor libraries imported to `tier-2-cqm-infra/vendor/`
- FHIRHelpers.cql replaced with official version
- IntervalHelpers.cql grammar errors fixed

**Expected Result**: CMS measures compile successfully with vendor dependencies.

---

## Appendix: Raw Error Samples

### CMS122 Errors (First 10)
```
1. Could not load source for library FHIRHelpers, version 4.4.000
2. Could not load source for library QICoreCommon, version 2.1.000
3. Could not load source for library SupplementalDataElements, version 3.5.000
4. Could not load source for library Status, version 1.8.000
5. Could not load source for library CumulativeMedicationDuration, version 4.1.000
6. Could not load source for library AdultOutpatientEncounters, version 4.11.000
7. Could not load source for library AdvancedIllnessandFrailty, version 1.16.000
8. Could not load source for library Hospice, version 6.12.000
9. Could not load source for library PalliativeCare, version 1.11.000
10. Could not resolve identifier SDE in the current library (cascade)
```

### FHIRHelpers Syntax Errors (Pattern)
```
Line 30:  Syntax error at .
Line 33:  Syntax error at .
Line 36:  Syntax error at .
... (89 occurrences - all value.value accessors)
```

### IntervalHelpers Semantic Errors
```
Line 133: Could not find type for model: null and name: quarter
Line 318: Internal translator error (days arithmetic)
Line 410: Circular reference in Range function
```

---

**Document Version**: 1.0
**Prepared For**: CTO/CMO Review
**Classification**: Technical - Build Infrastructure
