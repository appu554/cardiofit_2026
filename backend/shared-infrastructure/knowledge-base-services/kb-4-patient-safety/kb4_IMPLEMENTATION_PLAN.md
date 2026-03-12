# KB-4 Patient Safety Service Completion Plan

## Executive Summary

Complete the KB-4 Patient Safety Service by externalizing remaining hardcoded data, expanding drug coverage, implementing full Beers Criteria (Tables 1-5), adding STOPP/START criteria for international support, and populating AU/IN jurisdictions.

## Current State Analysis

### What's Already Implemented ✅
- **Governance Model**: Full ClinicalGovernance struct with provenance tracking
- **Jurisdiction Awareness**: US → Global → Flat fallback hierarchy in loader.go
- **YAML Externalization**: 8 knowledge categories loaded from YAML (282 entries)
- **Version Control**: EffectiveDate, ReviewDate, ApprovalStatus tracking
- **10 Safety Dimensions**: All dimensions defined in types.go

### Remaining Gaps 🔴
| Gap | Current State | Target State |
|-----|---------------|--------------|
| Dose Limits | 15 entries in data.go only | YAML with governance |
| Age Limits | 11 entries in data.go only | YAML with governance |
| Beers Criteria | Table 1 only (57 entries) | All 5 tables (200+ entries) |
| STOPP/START | Not implemented | Full v3 implementation |
| AU/IN Jurisdictions | Empty directories | Populated with local content |
| Drug Coverage | ~282 entries | 500+ entries |

---

## Phase 1: Externalize Dose Limits and Age Limits

### 1.1 Files to Create

| File | Location | Entries |
|------|----------|---------|
| `dose_limits.yaml` | `knowledge/global/dose_limits/` | 15+ |
| `age_limits.yaml` | `knowledge/global/age_limits/` | 11+ |

### 1.2 Types to Add (types.go)

```go
// DoseLimitEntry represents a governed dose limit
type DoseLimitEntry struct {
    RxNorm        string             `yaml:"rxnorm" json:"rxnorm"`
    DrugName      string             `yaml:"drugName" json:"drugName"`
    MaxDailyDose  float64            `yaml:"maxDailyDose" json:"maxDailyDose"`
    MaxSingleDose float64            `yaml:"maxSingleDose" json:"maxSingleDose"`
    Unit          string             `yaml:"unit" json:"unit"`
    Governance    ClinicalGovernance `yaml:"governance" json:"governance"`
    Notes         string             `yaml:"notes,omitempty" json:"notes,omitempty"`
}

// AgeLimitEntry represents a governed age restriction
type AgeLimitEntry struct {
    RxNorm     string             `yaml:"rxnorm" json:"rxnorm"`
    DrugName   string             `yaml:"drugName" json:"drugName"`
    MinimumAge int                `yaml:"minimumAge" json:"minimumAge"`
    MaximumAge int                `yaml:"maximumAge,omitempty" json:"maximumAge,omitempty"`
    Governance ClinicalGovernance `yaml:"governance" json:"governance"`
    Reason     string             `yaml:"reason" json:"reason"`
}
```

### 1.3 Loader Functions to Add (loader.go)

- `loadDoseLimits()` - Load dose_limits.yaml files
- `loadAgeLimits()` - Load age_limits.yaml files

### 1.4 Checker Updates (checker.go)

- Update `GetDoseLimit()` to check governed knowledge first
- Update age limit checking to use governed knowledge first

---

## Phase 2: Expand Drug Coverage

### Priority Additions by Category

| Category | Current | Target | Priority Sources |
|----------|---------|--------|------------------|
| Black Box | 52 | 150+ | FDA DailyMed |
| Pregnancy | 32 | 100+ | FDA PLLR Labels |
| Lactation | 28 | 80+ | NIH LactMed |
| High Alert | 44 | 100+ | ISMP 2024 List |
| Beers Table 1 | 57 | 100+ | AGS 2023 |
| Lab Monitoring | 35 | 80+ | FDA Labels |
| Contraindications | 20 | 60+ | FDA Labels |
| Anticholinergic | 25 | 60+ | ACB Scale |

---

## Phase 3: Beers Criteria Tables 2-5

### 3.1 New Types Required

| Type | Purpose | Source |
|------|---------|--------|
| `BeersConditionEntry` | Table 2: Disease-specific PIMs | AGS 2023 |
| `BeersCautionEntry` | Table 3: Use with Caution | AGS 2023 |
| `BeersDrugInteractionEntry` | Table 4: Drug-Drug Interactions | AGS 2023 |
| `BeersRenalEntry` | Table 5: Renal Adjustments | AGS 2023 |

### 3.2 New YAML Files

| File | Location | Entries |
|------|----------|---------|
| `beers_table2_conditions.yaml` | `knowledge/us/beers/` | 40+ |
| `beers_table3_caution.yaml` | `knowledge/us/beers/` | 15+ |
| `beers_table4_interactions.yaml` | `knowledge/us/beers/` | 10+ |
| `beers_table5_renal.yaml` | `knowledge/us/beers/` | 20+ |

---

## Phase 4: STOPP/START Criteria

### 4.1 New Types

| Type | Purpose | Source |
|------|---------|--------|
| `StoppEntry` | Potentially Inappropriate Prescriptions | STOPP/START v3 2023 |
| `StartEntry` | Prescribing Omissions | STOPP/START v3 2023 |

### 4.2 New YAML Files

| File | Location | Entries |
|------|----------|---------|
| `stopp_v3.yaml` | `knowledge/global/stopp_start/` | 80+ |
| `start_v3.yaml` | `knowledge/global/stopp_start/` | 35+ |

---

## Phase 5: Populate AU/IN Jurisdictions

### 5.1 Australian Content

| File | Source | Entries |
|------|--------|---------|
| `tga_blackbox.yaml` | TGA Product Information | 50+ |
| `tga_pregnancy.yaml` | TGA Pregnancy Categories | 80+ |
| `apinchs.yaml` | APINCHS High-Risk Meds | 40+ |

### 5.2 Indian Content

| File | Source | Entries |
|------|--------|---------|
| `cdsco_warnings.yaml` | CDSCO Safety Alerts | 40+ |
| `banned_combinations.yaml` | CDSCO Banned FDCs | 350+ |
| `nlem_2022.yaml` | National Essential Medicines | 300+ |

---

## Implementation Timeline

| Phase | Duration | Priority | Status |
|-------|----------|----------|--------|
| Phase 1: Dose/Age Limits | 3-4 days | CRITICAL | ✅ Complete |
| Phase 2: Drug Coverage | 5-7 days | HIGH | ⏳ Pending |
| Phase 3: Beers Tables 2-5 | 4-5 days | HIGH | ⏳ Pending |
| Phase 4: STOPP/START | 3-4 days | MEDIUM | ⏳ Pending |
| Phase 5: AU/IN Content | 5-7 days | MEDIUM | ⏳ Pending |

## Phase 1 Completion Summary (2025-01-14)

### Files Created
- `knowledge/global/dose_limits/dose_limits.yaml` - 15 dose limit entries with governance
- `knowledge/global/age_limits/age_limits.yaml` - 12 age limit entries with governance

### Files Modified
- `pkg/safety/loader.go`:
  - Added `DoseLimitsFile` and `AgeLimitsFile` struct types
  - Added `loadDoseLimits()` and `loadAgeLimits()` functions
  - Added `GetDoseLimit()` and `GetAgeLimit()` query methods to KnowledgeStore
  - Updated `LoadAll()` to load dose and age limits
  - Updated `GetStats()` to include dose_limits and age_limits counts

- `pkg/safety/checker.go`:
  - Updated `GetDoseLimit()` to use governed knowledge first
  - Added `GetAgeLimit()` public method with governed knowledge support
  - Updated `checkDoseLimits()` to use governed YAML → hardcoded fallback
  - Updated `checkAgeLimits()` to use governed YAML → hardcoded fallback

### Build Status
- ✅ `go build ./...` - Success
- ⚠️ No test files in pkg/safety (tests recommended for Phase 2)

---

## Success Criteria

1. ✅ Zero safety knowledge hardcoded without YAML fallback
2. ✅ 500+ unique drug entries across all categories
3. ✅ All 5 AGS 2023 Beers tables implemented
4. ✅ STOPP/START v3 fully implemented
5. ✅ AU and IN directories populated
6. ✅ >90% test coverage on new code
7. ✅ 100% governance metadata completeness

---

## Verification

```bash
# Build and test
cd backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety
go build ./...
go test ./...

# Health check
make health

# Integration test
curl http://localhost:8088/health
```
