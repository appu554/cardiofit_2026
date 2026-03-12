# Phase 3b.6: KB-16 Lab Reference Ranges Ingestion - Implementation Plan

> **Analysis Date**: 2026-01-26
> **Specification Source**: `shared/Phase3b6_KB16_Lab_Reference_Ingestion.docx`
> **Implementation Location**: `kb-16-lab-interpretation/`
> **Estimated Effort**: 12.5 days (6 FTE with parallel work)
> **Status**: 🟡 **PLANNING COMPLETE - READY FOR IMPLEMENTATION**

---

## Executive Summary

Phase 3b.6 transforms KB-16 from a basic lab interpretation service into a **Context-Aware Clinical Reference Engine**. The key insight is that lab values must be interpreted based on PATIENT CONTEXT (pregnancy, CKD stage, age, etc.), not just standard reference ranges.

### The Problem
```
Traditional: Hemoglobin 12-16 g/dL for ALL patients
Reality:     Hemoglobin 10.5-14.0 g/dL for pregnant women (T3) → hemodilution
             Using standard range = DANGEROUS false normals/abnormals
```

### The Solution
**Conditional Reference Ranges** that automatically select the correct range based on:
- **Age**: Neonate, Pediatric, Adult, Geriatric
- **Gender**: Male, Female
- **Pregnancy**: Trimester 1, 2, 3, Postpartum
- **Lactation**: Breastfeeding status
- **Renal Function**: CKD Stage 1-5, Dialysis
- **Gestational Age**: For neonatal bilirubin (Bhutani nomogram)

---

## Gap Analysis: Current vs. Required

### ✅ Already Exists in KB-16

| Component | File | Status |
|-----------|------|--------|
| Basic reference database | `pkg/reference/database.go` | ✅ 40+ tests, age/sex ranges |
| Authority registry | `pkg/reference/authorities.go` | ✅ 30+ authorities (CLSI, ACOG, ATA, KDIGO) |
| ICMR India-specific ranges | `pkg/reference/icmr_ranges.go` | ✅ Complete with dual interpretation |
| Interpretation engine | `pkg/interpretation/engine.go` | ✅ Context-aware comments |
| Type definitions | `pkg/types/types.go` | ✅ 650+ lines |
| Database schema | `migrations/001_initial_schema.sql` | ✅ Core tables |
| Context-aware comments | `pkg/interpretation/engine.go` | ✅ Pregnancy, CKD, Dialysis, etc. |

### 🔴 Missing (Phase 3b.6 Scope)

| Component | Gap Description | Priority |
|-----------|-----------------|----------|
| `ConditionalReferenceRange` struct | Full condition-based range selection | P0 |
| `RangeConditions` schema | Pregnancy/trimester, CKD stage, gestational age | P0 |
| Range Selection Algorithm | Specificity scoring, most-specific-wins | P0 |
| `conditional_reference_ranges` table | Database migration 024 | P0 |
| `lab_tests` table | Centralized LOINC test definitions | P0 |
| Pregnancy-specific ranges | TSH, Cr, Hgb, Plt, Fibrinogen by trimester | P0 |
| CKD-adjusted ranges | K, Phos, PTH by CKD stage | P0 |
| Neonatal bilirubin nomogram | Bhutani curves with hour-of-life | P0 |
| Range ingestion from authorities | ACOG, ATA, KDIGO, AAP structured tables | P1 |

---

## Architecture Design

### 3b.6.1 ConditionalReferenceRange Schema

**File**: `kb-16-lab-interpretation/pkg/reference/conditional_ranges.go`

```go
// ConditionalReferenceRange represents a reference range with conditions
type ConditionalReferenceRange struct {
    ID              uuid.UUID       `json:"id" gorm:"type:uuid;primary_key"`
    LabTestID       uuid.UUID       `json:"lab_test_id" gorm:"type:uuid;not null"`

    // CONDITIONS (all must match for this range to apply)
    Conditions      RangeConditions `json:"conditions" gorm:"embedded"`

    // Reference values
    LowNormal       *float64        `json:"low_normal,omitempty"`
    HighNormal      *float64        `json:"high_normal,omitempty"`
    CriticalLow     *float64        `json:"critical_low,omitempty"`
    CriticalHigh    *float64        `json:"critical_high,omitempty"`
    PanicLow        *float64        `json:"panic_low,omitempty"`
    PanicHigh       *float64        `json:"panic_high,omitempty"`

    // Interpretation hints
    InterpretationNote string       `json:"interpretation_note,omitempty"`
    ClinicalAction     string       `json:"clinical_action,omitempty"`

    // Governance
    Authority          string       `json:"authority" gorm:"not null"`
    AuthorityRef       string       `json:"authority_ref" gorm:"not null"`
    AuthorityVersion   string       `json:"authority_version,omitempty"`
    EffectiveDate      time.Time    `json:"effective_date" gorm:"not null"`
    ExpirationDate     *time.Time   `json:"expiration_date,omitempty"`

    // Priority (higher = more specific, wins ties)
    SpecificityScore   int          `json:"specificity_score" gorm:"default:0"`

    CreatedAt          time.Time    `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt          time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}
```

### 3b.6.2 RangeConditions Schema

```go
// RangeConditions defines all patient context variables for range selection
type RangeConditions struct {
    // Demographics
    Gender          *string   `json:"gender,omitempty" gorm:"column:gender"`           // M, F, null=any
    AgeMinYears     *float64  `json:"age_min_years,omitempty" gorm:"column:age_min_years"`
    AgeMaxYears     *float64  `json:"age_max_years,omitempty" gorm:"column:age_max_years"`
    AgeMinDays      *int      `json:"age_min_days,omitempty" gorm:"column:age_min_days"`     // For neonates
    AgeMaxDays      *int      `json:"age_max_days,omitempty" gorm:"column:age_max_days"`

    // Pregnancy/Lactation
    IsPregnant      *bool     `json:"is_pregnant,omitempty" gorm:"column:is_pregnant"`
    Trimester       *int      `json:"trimester,omitempty" gorm:"column:trimester"`           // 1, 2, 3
    IsPostpartum    *bool     `json:"is_postpartum,omitempty" gorm:"column:is_postpartum"`
    PostpartumWeeks *int      `json:"postpartum_weeks,omitempty" gorm:"column:postpartum_weeks"`
    IsLactating     *bool     `json:"is_lactating,omitempty" gorm:"column:is_lactating"`

    // Neonatal (for bilirubin nomograms)
    GestationalAgeWeeksMin *int `json:"ga_weeks_min,omitempty" gorm:"column:gestational_age_weeks_min"`
    GestationalAgeWeeksMax *int `json:"ga_weeks_max,omitempty" gorm:"column:gestational_age_weeks_max"`
    HoursOfLifeMin  *int      `json:"hours_of_life_min,omitempty" gorm:"column:hours_of_life_min"`
    HoursOfLifeMax  *int      `json:"hours_of_life_max,omitempty" gorm:"column:hours_of_life_max"`

    // Renal status
    CKDStage        *int      `json:"ckd_stage,omitempty" gorm:"column:ckd_stage"`           // 1-5
    IsOnDialysis    *bool     `json:"is_on_dialysis,omitempty" gorm:"column:is_on_dialysis"`
    EGFRMin         *float64  `json:"egfr_min,omitempty" gorm:"column:egfr_min"`
    EGFRMax         *float64  `json:"egfr_max,omitempty" gorm:"column:egfr_max"`
}
```

### 3b.6.3 Range Selection Algorithm

```go
// SelectRange chooses the most specific matching range for a patient
func (e *ReferenceEngine) SelectRange(ctx context.Context,
    loincCode string, patient *PatientContext) (*ConditionalReferenceRange, error) {

    // 1. Get all ranges for this LOINC code
    ranges, err := e.getRangesForLOINC(ctx, loincCode)
    if err != nil {
        return nil, err
    }

    // 2. Filter to ranges where ALL conditions match patient
    var matching []*ConditionalReferenceRange
    for _, r := range ranges {
        if e.conditionsMatch(&r.Conditions, patient) {
            matching = append(matching, r)
        }
    }

    // 3. If no match, return default (conditions all null)
    if len(matching) == 0 {
        return e.getDefaultRange(ctx, loincCode)
    }

    // 4. If multiple match, select MOST SPECIFIC (highest SpecificityScore)
    sort.Slice(matching, func(i, j int) bool {
        return matching[i].SpecificityScore > matching[j].SpecificityScore
    })

    return matching[0], nil
}

// conditionsMatch checks if all non-null conditions match the patient
func (e *ReferenceEngine) conditionsMatch(cond *RangeConditions, patient *PatientContext) bool {
    // Each non-null condition MUST match
    if cond.Gender != nil && *cond.Gender != patient.Sex {
        return false
    }
    if cond.AgeMinYears != nil && float64(patient.Age) < *cond.AgeMinYears {
        return false
    }
    if cond.AgeMaxYears != nil && float64(patient.Age) >= *cond.AgeMaxYears {
        return false
    }
    if cond.IsPregnant != nil && *cond.IsPregnant != patient.IsPregnant {
        return false
    }
    if cond.Trimester != nil && *cond.Trimester != patient.Trimester {
        return false
    }
    if cond.CKDStage != nil && *cond.CKDStage != patient.CKDStage {
        return false
    }
    if cond.IsOnDialysis != nil && *cond.IsOnDialysis != patient.IsOnDialysis {
        return false
    }
    // ... additional condition checks
    return true
}
```

---

## Database Migration

### Migration 003: Conditional Reference Ranges

**File**: `kb-16-lab-interpretation/migrations/003_conditional_reference_ranges.sql`

```sql
-- Migration: 003_conditional_reference_ranges.sql
-- Phase 3b.6: Context-Aware Lab Reference Ranges

-- =============================================================================
-- LAB TESTS TABLE (Centralized LOINC definitions)
-- =============================================================================

CREATE TABLE IF NOT EXISTS lab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loinc_code VARCHAR(20) NOT NULL UNIQUE,
    test_name VARCHAR(200) NOT NULL,
    short_name VARCHAR(50),
    unit VARCHAR(50) NOT NULL,
    specimen_type VARCHAR(50),           -- blood, urine, csf
    method VARCHAR(100),                 -- enzymatic, colorimetric, etc.
    category VARCHAR(50),                -- Chemistry, Hematology, Coagulation
    decimal_places INT DEFAULT 2,
    trending_enabled BOOLEAN DEFAULT TRUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- CONDITIONAL REFERENCE RANGES TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS conditional_reference_ranges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lab_test_id UUID NOT NULL REFERENCES lab_tests(id),

    -- CONDITIONS (null = any)
    gender VARCHAR(1),                    -- M, F, null
    age_min_years DECIMAL(5,2),
    age_max_years DECIMAL(5,2),
    age_min_days INTEGER,                 -- For neonates
    age_max_days INTEGER,

    -- Pregnancy/Lactation
    is_pregnant BOOLEAN,
    trimester INTEGER,                    -- 1, 2, 3
    is_postpartum BOOLEAN,
    postpartum_weeks INTEGER,
    is_lactating BOOLEAN,

    -- Neonatal (for bilirubin nomograms)
    gestational_age_weeks_min INTEGER,
    gestational_age_weeks_max INTEGER,
    hours_of_life_min INTEGER,
    hours_of_life_max INTEGER,

    -- Renal status
    ckd_stage INTEGER,                    -- 1-5
    is_on_dialysis BOOLEAN,
    egfr_min DECIMAL(6,2),
    egfr_max DECIMAL(6,2),

    -- REFERENCE VALUES
    low_normal DECIMAL(10,4),
    high_normal DECIMAL(10,4),
    critical_low DECIMAL(10,4),
    critical_high DECIMAL(10,4),
    panic_low DECIMAL(10,4),
    panic_high DECIMAL(10,4),

    -- Interpretation hints
    interpretation_note TEXT,
    clinical_action TEXT,

    -- GOVERNANCE
    authority VARCHAR(50) NOT NULL,       -- CLSI, ACOG, ATA, KDIGO
    authority_reference TEXT NOT NULL,    -- Specific document
    authority_version VARCHAR(50),
    effective_date DATE NOT NULL,
    expiration_date DATE,

    -- Priority (higher = more specific)
    specificity_score INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- NEONATAL BILIRUBIN THRESHOLDS (Bhutani Nomogram)
-- =============================================================================

CREATE TABLE IF NOT EXISTS neonatal_bilirubin_thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Risk stratification
    gestational_age_weeks_min INTEGER NOT NULL,
    gestational_age_weeks_max INTEGER NOT NULL,
    risk_category VARCHAR(20) NOT NULL,  -- LOW, MEDIUM, HIGH

    -- Hour-of-life thresholds (for interpolation)
    hour_of_life INTEGER NOT NULL,
    photo_threshold DECIMAL(5,2) NOT NULL,     -- Start phototherapy
    exchange_threshold DECIMAL(5,2),           -- Consider exchange transfusion

    -- Governance
    authority VARCHAR(50) DEFAULT 'AAP',
    authority_reference TEXT DEFAULT 'AAP Clinical Practice Guideline 2022',

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(gestational_age_weeks_min, gestational_age_weeks_max, risk_category, hour_of_life)
);

-- =============================================================================
-- INDEXES
-- =============================================================================

CREATE INDEX idx_crr_lab_test ON conditional_reference_ranges(lab_test_id);
CREATE INDEX idx_crr_pregnancy ON conditional_reference_ranges(is_pregnant, trimester);
CREATE INDEX idx_crr_ckd ON conditional_reference_ranges(ckd_stage);
CREATE INDEX idx_crr_dialysis ON conditional_reference_ranges(is_on_dialysis);
CREATE INDEX idx_crr_age ON conditional_reference_ranges(age_min_years, age_max_years);
CREATE INDEX idx_crr_neonatal ON conditional_reference_ranges(hours_of_life_min, gestational_age_weeks_min);
CREATE INDEX idx_crr_authority ON conditional_reference_ranges(authority);
CREATE INDEX idx_crr_specificity ON conditional_reference_ranges(specificity_score DESC);

CREATE INDEX idx_bili_ga ON neonatal_bilirubin_thresholds(gestational_age_weeks_min, gestational_age_weeks_max);
CREATE INDEX idx_bili_hour ON neonatal_bilirubin_thresholds(hour_of_life);
CREATE INDEX idx_bili_risk ON neonatal_bilirubin_thresholds(risk_category);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE lab_tests IS 'Centralized LOINC-based lab test definitions';
COMMENT ON TABLE conditional_reference_ranges IS 'Context-aware reference ranges with conditions';
COMMENT ON TABLE neonatal_bilirubin_thresholds IS 'Bhutani nomogram for neonatal jaundice';
COMMENT ON COLUMN conditional_reference_ranges.specificity_score IS 'Higher score = more specific, wins selection ties';
COMMENT ON COLUMN conditional_reference_ranges.trimester IS 'Pregnancy trimester: 1, 2, or 3';
COMMENT ON COLUMN conditional_reference_ranges.ckd_stage IS 'CKD stage: 1-5, per KDIGO classification';
```

---

## Implementation Tasks

| # | Task | Deliverable | Days | Priority |
|---|------|-------------|------|----------|
| **1** | **Database Migration** | `003_conditional_reference_ranges.sql` | 1 | P0 |
| **2** | **Go Structs & Models** | `pkg/reference/conditional_ranges.go` | 1 | P0 |
| **3** | **Pregnancy Ranges Ingestion** | TSH, Cr, Hgb, Plt, Fibrinogen by trimester | 2 | P0 |
| **4** | **Renal Function Ranges** | K, Phos, PTH by CKD stage | 1 | P0 |
| **5** | **Age-Specific Ranges** | Neonate, Pediatric, Adult, Geriatric | 1.5 | P0 |
| **6** | **Gender-Specific Ranges** | Hgb, Cr, Ferritin, Hormones | 0.5 | P1 |
| **7** | **Neonatal Bilirubin Nomogram** | Bhutani curves by GA/risk/hour | 1 | P0 |
| **8** | **Critical Values by Context** | Panic + Critical thresholds | 1 | P0 |
| **9** | **Range Selection Engine** | `pkg/reference/range_selector.go` | 1.5 | P0 |
| **10** | **Integration with Interpreter** | Update `engine.go` to use conditional ranges | 1 | P0 |
| **11** | **India-Specific Ranges (ICMR)** | Enhance existing with conditions | 0.5 | P1 |
| **12** | **Unit + Integration Tests** | >85% coverage | 2 | P1 |

**Total Estimated Effort**: 14 days (can be parallelized to ~8 days with 2 developers)

---

## Condition-Specific Reference Ranges Data

### Pregnancy-Specific Ranges (ACOG, ATA)

| Test | Unit | T1 (1-13w) | T2 (14-27w) | T3 (28-40w) | Authority |
|------|------|------------|-------------|-------------|-----------|
| Hemoglobin | g/dL | 11.0-14.0 | 10.5-14.0 | 10.5-14.0 | WHO, ACOG |
| Platelets | k/µL | >150 | >100 | >100 | ACOG |
| Creatinine | mg/dL | 0.4-0.7 | 0.4-0.8 | 0.4-0.9 | ACOG |
| TSH | mIU/L | 0.1-2.5 | 0.2-3.0 | 0.3-3.0 | ATA 2017 |
| Fibrinogen | mg/dL | 300-500 | 350-550 | 400-600 | ACOG |
| Uric Acid | mg/dL | 2.0-4.5 | 2.5-5.0 | 2.5-5.5 | ACOG |
| AST/ALT | U/L | 10-35 | 10-35 | 10-35 | AASLD |

**Clinical Alert**: AST/ALT ≥2× ULN in pregnancy → HELLP syndrome evaluation

### Renal Function-Adjusted Ranges (KDIGO)

| Test | Unit | CKD 1-2 (eGFR ≥60) | CKD 3-4 (eGFR 15-59) | CKD 5 / Dialysis |
|------|------|---------------------|----------------------|------------------|
| Potassium | mEq/L | 3.5-5.0 | 3.5-5.5 | 3.5-6.0 |
| Phosphate | mg/dL | 2.5-4.5 | 2.5-4.5 | 3.5-5.5 |
| Hgb Target | g/dL | 12-16 (F), 14-18 (M) | 10-12 | 10-11.5 |
| PTH | pg/mL | 15-65 | 35-150 | 150-600 |

### Neonatal Bilirubin Thresholds (AAP 2022)

| Hours of Life | Low Risk (≥38w) | Medium Risk (35-37w) | High Risk (<35w) |
|---------------|-----------------|----------------------|------------------|
| 24h | 12 mg/dL | 10 mg/dL | 8 mg/dL |
| 48h | 15 mg/dL | 13 mg/dL | 11 mg/dL |
| 72h | 18 mg/dL | 16 mg/dL | 14 mg/dL |
| 96h+ | 20 mg/dL | 18 mg/dL | 15 mg/dL |

---

## File Structure

```
kb-16-lab-interpretation/
├── pkg/
│   ├── reference/
│   │   ├── database.go              # Existing - basic test definitions
│   │   ├── authorities.go           # Existing - authority registry
│   │   ├── icmr_ranges.go           # Existing - India-specific
│   │   ├── conditional_ranges.go    # NEW - structs & models
│   │   ├── range_selector.go        # NEW - selection algorithm
│   │   ├── pregnancy_ranges.go      # NEW - ACOG/ATA ranges
│   │   ├── renal_ranges.go          # NEW - KDIGO ranges
│   │   └── neonatal_bilirubin.go    # NEW - Bhutani nomogram
│   ├── interpretation/
│   │   └── engine.go                # UPDATE - use conditional ranges
│   ├── types/
│   │   ├── types.go                 # Existing
│   │   └── patient_context.go       # NEW - enhanced patient context
│   └── store/
│       └── conditional_store.go     # NEW - database operations
├── migrations/
│   ├── 001_initial_schema.sql       # Existing
│   ├── 002_clinical_decision_limits.sql  # Existing
│   └── 003_conditional_reference_ranges.sql  # NEW
└── tests/
    ├── conditional_ranges_test.go   # NEW
    ├── range_selector_test.go       # NEW
    └── pregnancy_ranges_test.go     # NEW
```

---

## Exit Criteria

| # | Criterion | Validation Method |
|---|-----------|-------------------|
| 1 | Conditional reference range schema deployed | Migration runs successfully |
| 2 | Pregnancy-specific ranges for all critical tests | TSH, Cr, Hgb, Plt, Fibrinogen populated |
| 3 | CKD-stage-specific targets | K, Phos, PTH, Hgb by CKD 1-5 + dialysis |
| 4 | Age-stratified ranges | Neonate, pediatric, adult, geriatric present |
| 5 | Neonatal bilirubin nomogram | Hour-of-life interpolation working |
| 6 | Range selection engine | Correctly picks most-specific matching range |
| 7 | All ranges have governance | Authority, reference, effective date populated |
| 8 | >85% test coverage | `go test -cover` shows ≥85% |
| 9 | **NO LLM used** | All ranges from structured guideline tables |

---

## Integration Points

### KB-2 Clinical Context Integration
```go
// Patient context from KB-2 provides:
type PatientContext struct {
    PatientID    string
    Age          int
    Sex          string           // M, F
    IsPregnant   bool
    Trimester    int              // 1, 2, 3
    CKDStage     int              // 1-5
    IsOnDialysis bool
    GestationalAge int            // weeks (for neonates)
    HoursOfLife  int              // for neonatal bilirubin
    Conditions   []Condition      // ICD-10/SNOMED codes
    Medications  []Medication     // RxNorm codes
}
```

### KB-14 Care Navigator Integration
- Critical values → KB-14 task creation
- SLA tracking for acknowledgment

---

## Authority Source Mapping

| Authority | Scope | Documents to Ingest |
|-----------|-------|---------------------|
| **CLSI C28** | Reference interval methodology | C28-A3c (2024) |
| **ACOG** | Pregnancy ranges | Practice Bulletins |
| **ATA** | Thyroid in pregnancy | 2017 Guidelines |
| **KDIGO** | CKD targets | 2012 + 2024 Updates |
| **AAP** | Neonatal bilirubin | 2022 Hyperbilirubinemia Guideline |
| **CAP** | Critical values | Critical Value Notification |
| **ICMR** | India-specific | Already in `icmr_ranges.go` |

---

## Key Design Decisions

1. **Specificity Scoring**: Higher score = more specific. Pregnancy T3 female (score=3) wins over generic female (score=1).

2. **Null = Any**: Null conditions match any patient. Only non-null conditions must match.

3. **Most-Specific Wins**: When multiple ranges match, highest specificity_score wins.

4. **Authority Hierarchy**: CLSI methodology → Specialty guidelines (ACOG, KDIGO) → Regional (ICMR).

5. **No LLM**: All ranges come from structured guideline tables - purely deterministic ingestion.

---

*Phase 3b.6 Implementation Plan Complete*
*Generated: 2026-01-26*
