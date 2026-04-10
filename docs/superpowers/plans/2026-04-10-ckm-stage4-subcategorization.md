# CKM Stage 4 Subcategorization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change CKM staging from integer 0-4 to string with 4a/4b/4c substages per AHA Ndumele et al. 2023, adding HF medication gating, mandatory medication checker, and MHRI substage-aware domain weighting — so a HFrEF patient never receives pioglitazone and a HFpEF patient gets SGLT2i as the only mandatory disease-modifying therapy.

**Architecture:** KB-20 owns the CKM stage model, migration (int→string), and classifier. KB-23 consumes substage for HF medication gating (contraindications), mandatory medication gap detection, and urgency escalation. KB-26 adjusts MHRI domain weights and score ceiling by substage/NYHA class. Market YAML configs define substage-specific mandatory/recommended/contraindicated medications for India and Australia.

**Tech Stack:** Go 1.21 (Gin/GORM/stdlib testing), PostgreSQL 15 (JSONB for substage metadata), YAML market configs.

**Dependency:** This plan runs on `feature/v4-clinical-gaps` branch after the clinical gaps implementation (renal gating, CGM analytics, therapeutic inertia). It modifies `conflict_detector.go` and `urgency_calculator.go` from that work.

---

## File Inventory

### KB-20 Patient Profile (6 files)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-20-patient-profile/internal/models/ckm_stage_v2.go` | CKMStageValue string type, SubstageMetadata JSONB, HFType, ASCVDEvent, SubclinicalMarker |
| Create | `kb-20-patient-profile/internal/services/ckm_classifier.go` | ClassifyCKMStage() with full 0-4c staging per Ndumele 2023 |
| Create | `kb-20-patient-profile/internal/services/ckm_classifier_test.go` | 11 test cases covering all stages + Asian BMI + rheumatic |
| Create | `kb-20-patient-profile/migrations/005_ckm_substage.sql` | Add ckm_stage_v2 VARCHAR(5), ckm_substage_metadata JSONB, migration from int |
| Modify | `kb-20-patient-profile/internal/models/patient_profile.go:71` | Add CKMStageV2 string + SubstageMetadata fields alongside existing int |
| Modify | `kb-20-patient-profile/internal/models/ckm_stage.go` | Add deprecation comment pointing to ckm_stage_v2.go |

### KB-23 Decision Cards (9 files)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-23-decision-cards/internal/services/ckm_stage4_pathways.go` | Load pathway YAMLs, query mandatory/recommended/contraindicated meds by substage |
| Create | `kb-23-decision-cards/internal/services/ckm_stage4_pathways_test.go` | Test pathway loading + querying |
| Create | `kb-23-decision-cards/internal/services/hf_medication_gate.go` | HFMedicationGate — block pioglitazone/saxagliptin/non-DHP CCB in 4c |
| Create | `kb-23-decision-cards/internal/services/hf_medication_gate_test.go` | 5 tests: pioglitazone blocked, saxagliptin blocked, non-DHP CCB blocked, metformin allowed, non-4c allowed |
| Create | `kb-23-decision-cards/internal/services/mandatory_med_checker.go` | MandatoryMedChecker — detect missing mandatory meds per substage |
| Create | `kb-23-decision-cards/internal/services/mandatory_med_checker_test.go` | 4 tests: 4a missing statin, 4b all present, 4c-HFrEF missing pillars, 4c-HFpEF only SGLT2i |
| Modify | `kb-23-decision-cards/internal/services/conflict_detector.go` | Add HF medication gate to DetectAllConflicts |
| Modify | `kb-23-decision-cards/internal/services/urgency_calculator.go` | Substage escalation for 4c |

### KB-26 Metabolic Digital Twin (2 files)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-26-metabolic-digital-twin/internal/services/ckm_mhri_adjustment.go` | MHRISubstageAdjustment — domain weight rebalancing + NYHA ceiling |
| Create | `kb-26-metabolic-digital-twin/internal/services/ckm_mhri_adjustment_test.go` | 4 tests: 4a prevention, 4c-HFrEF cardio dominant, 4c-HFpEF equal, Stage 2 default |

### Market Configs (3 files)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `market-configs/shared/ckm_stage4_pathways.yaml` | Mandatory/recommended/contraindicated meds per 4a/4b/4c substage |
| Create | `market-configs/india/ckm_stage4_overrides.yaml` | CSI 2023 preferences, ARNI cost, rheumatic HF pathway |
| Create | `market-configs/australia/ckm_stage4_overrides.yaml` | PBS authority codes for Entresto/ticagrelor, indigenous screening |

**Total: 20 files (14 create, 6 modify) across 18 steps in 3 phases.**

All KB service paths are relative to `backend/shared-infrastructure/knowledge-base-services/`. Market config paths are relative to `backend/shared-infrastructure/`.

---

## Phase K1: CKM Stage V2 Data Model + Migration + Classifier (Steps 1–5)

---

- [ ] **Step 1: Create CKM Stage V2 model**

Create `kb-20-patient-profile/internal/models/ckm_stage_v2.go`:

```go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// CKMStageValue represents the full CKM stage with substage.
// Valid values: "0", "1", "2", "3", "4a", "4b", "4c"
type CKMStageValue string

const (
	CKMStageV2_0  CKMStageValue = "0"
	CKMStageV2_1  CKMStageValue = "1"
	CKMStageV2_2  CKMStageValue = "2"
	CKMStageV2_3  CKMStageValue = "3"
	CKMStageV2_4a CKMStageValue = "4a"
	CKMStageV2_4b CKMStageValue = "4b"
	CKMStageV2_4c CKMStageValue = "4c"
)

// IsStage4 returns true for any Stage 4 substage.
func (s CKMStageValue) IsStage4() bool {
	return s == CKMStageV2_4a || s == CKMStageV2_4b || s == CKMStageV2_4c
}

// NumericStage returns the integer portion (0-4) for backward compatibility.
func (s CKMStageValue) NumericStage() int {
	switch s {
	case CKMStageV2_0:
		return 0
	case CKMStageV2_1:
		return 1
	case CKMStageV2_2:
		return 2
	case CKMStageV2_3:
		return 3
	case CKMStageV2_4a, CKMStageV2_4b, CKMStageV2_4c:
		return 4
	default:
		return -1
	}
}

// IsValid checks if the stage value is one of the defined constants.
func (s CKMStageValue) IsValid() bool {
	return s.NumericStage() >= 0
}

// HFType represents the heart failure classification for Stage 4c.
type HFType string

const (
	HFTypeReduced       HFType = "HFrEF"  // EF <= 40%
	HFTypeMildlyReduced HFType = "HFmrEF" // EF 41-49%
	HFTypePreserved     HFType = "HFpEF"  // EF >= 50%
	HFTypeUnclassified  HFType = ""
)

// ClassifyHFType determines HF subtype from LVEF percentage.
func ClassifyHFType(lvefPct float64) HFType {
	switch {
	case lvefPct <= 40:
		return HFTypeReduced
	case lvefPct <= 49:
		return HFTypeMildlyReduced
	default:
		return HFTypePreserved
	}
}

// SubstageMetadata carries the clinical detail behind Stage 4 classification.
type SubstageMetadata struct {
	// Stage 4c — Heart Failure
	HFClassification HFType   `json:"hf_type,omitempty"`
	LVEFPercent      *float64 `json:"lvef_pct,omitempty"`
	NYHAClass        string   `json:"nyha_class,omitempty"`
	NTproBNP         *float64 `json:"nt_probnp,omitempty"`
	BNP              *float64 `json:"bnp,omitempty"`
	HFEtiology       string   `json:"hf_etiology,omitempty"`

	// Stage 4b — Clinical ASCVD
	ASCVDEvents         []ASCVDEvent `json:"ascvd_events,omitempty"`
	MostRecentEventDate *time.Time   `json:"most_recent_event_date,omitempty"`
	OnAntiplatelet      bool         `json:"on_antiplatelet,omitempty"`

	// Stage 4a — Subclinical CVD
	SubclinicalMarkers []SubclinicalMarker `json:"subclinical_markers,omitempty"`
	CACScore           *float64            `json:"cac_score,omitempty"`
	CIMTPercentile     *int                `json:"cimt_percentile,omitempty"`
	HasLVH             bool                `json:"has_lvh,omitempty"`

	// Staging metadata
	StagingDate       time.Time `json:"staging_date"`
	StagingSource     string    `json:"staging_source,omitempty"`
	ReviewNeeded      bool      `json:"review_needed,omitempty"`
	RheumaticEtiology bool      `json:"rheumatic_etiology,omitempty"`
}

type ASCVDEvent struct {
	EventType string    `json:"event_type"` // MI, STROKE, TIA, PAD, PCI, CABG, SIGNIFICANT_CAD
	EventDate time.Time `json:"event_date"`
	Details   string    `json:"details,omitempty"`
}

type SubclinicalMarker struct {
	MarkerType string    `json:"marker_type"` // CAC, CIMT, LVH, NT_PROBNP_ELEVATED, SUBCLINICAL_ATHEROSCLEROSIS
	Value      string    `json:"value"`
	Date       time.Time `json:"date"`
}

// Scan implements sql.Scanner for GORM JSONB deserialization.
func (m *SubstageMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("SubstageMetadata.Scan: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// Value implements driver.Valuer for GORM JSONB serialization.
func (m SubstageMetadata) Value() (driver.Value, error) {
	if m.StagingDate.IsZero() && m.HFClassification == "" &&
		len(m.ASCVDEvents) == 0 && len(m.SubclinicalMarkers) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// CKMStageResult is the full staging output.
type CKMStageResult struct {
	Stage            CKMStageValue    `json:"stage"`
	PreviousStage    CKMStageValue    `json:"previous_stage,omitempty"`
	StageChanged     bool             `json:"stage_changed"`
	Metadata         SubstageMetadata `json:"metadata"`
	StagingRationale string           `json:"staging_rationale"`
}
```

---

- [ ] **Step 2: Create migration SQL**

Create `kb-20-patient-profile/migrations/005_ckm_substage.sql`:

```sql
-- CKM Stage 4 subcategorization migration.
-- Changes ckm_stage from integer 0-4 to ckm_stage_v2 VARCHAR(5) with 4a/4b/4c.
-- Existing Stage 4 patients default to "4a" (safest: triggers preventive pathway).

-- Step 1: Add new columns
ALTER TABLE patient_profiles
    ADD COLUMN IF NOT EXISTS ckm_stage_v2 VARCHAR(5),
    ADD COLUMN IF NOT EXISTS ckm_substage_metadata JSONB,
    ADD COLUMN IF NOT EXISTS ckm_substage_review_needed BOOLEAN DEFAULT FALSE;

-- Step 2: Migrate existing integer ckm_stage to string
UPDATE patient_profiles SET ckm_stage_v2 = CASE
    WHEN ckm_stage = 0 THEN '0'
    WHEN ckm_stage = 1 THEN '1'
    WHEN ckm_stage = 2 THEN '2'
    WHEN ckm_stage = 3 THEN '3'
    WHEN ckm_stage = 4 THEN '4a'
    ELSE '0'
END
WHERE ckm_stage_v2 IS NULL;

-- Step 3: Flag existing Stage 4 patients for clinician review
UPDATE patient_profiles
    SET ckm_substage_review_needed = TRUE,
        ckm_substage_metadata = jsonb_build_object(
            'review_needed', true,
            'staging_source', 'MIGRATION',
            'staging_date', NOW()
        )
WHERE ckm_stage = 4;

-- Step 4: Create indexes
CREATE INDEX IF NOT EXISTS idx_pp_ckm_v2 ON patient_profiles(ckm_stage_v2);
CREATE INDEX IF NOT EXISTS idx_pp_ckm_review ON patient_profiles(ckm_substage_review_needed)
    WHERE ckm_substage_review_needed = TRUE;

-- Step 5: Add check constraint
ALTER TABLE patient_profiles
    ADD CONSTRAINT chk_ckm_stage_v2
    CHECK (ckm_stage_v2 IN ('0', '1', '2', '3', '4a', '4b', '4c'));
```

---

- [ ] **Step 3: Add V2 fields to PatientProfile model**

Modify `kb-20-patient-profile/internal/models/patient_profile.go` — add after the existing `CKMStage int` field (line 71):

```go
	CKMStageV2          string            `gorm:"column:ckm_stage_v2;type:varchar(5)" json:"ckm_stage_v2"`
	CKMSubstageMetadata *SubstageMetadata `gorm:"column:ckm_substage_metadata;type:jsonb" json:"ckm_substage_metadata,omitempty"`
	CKMSubstageReviewNeeded bool          `gorm:"column:ckm_substage_review_needed;default:false" json:"ckm_substage_review_needed"`
```

Also add deprecation comment to the old field:

```go
	// Deprecated: Use CKMStageV2 (string "0"-"4c") instead. Kept for backward compat.
	CKMStage       int      `gorm:"default:0" json:"ckm_stage"`
```

And modify `kb-20-patient-profile/internal/models/ckm_stage.go` — add at top after package line:

```go
// Deprecated: This file contains the legacy integer-based CKM staging.
// New code should use CKMStageValue from ckm_stage_v2.go and
// ClassifyCKMStage from services/ckm_classifier.go.
```

---

- [ ] **Step 4: Write CKM classifier tests**

Create `kb-20-patient-profile/internal/services/ckm_classifier_test.go`:

```go
package services

import (
	"testing"
	"time"

	"kb-patient-profile/internal/models"
)

func floatP(f float64) *float64 { return &f }
func intP(i int) *int           { return &i }

func TestClassifyCKM_Stage0_NoRiskFactors(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 30, BMI: 22.0, EGFR: 95,
	})
	if result.Stage != models.CKMStageV2_0 {
		t.Errorf("expected Stage 0, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage1_Adiposity(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 35, BMI: 27.0, EGFR: 90,
	})
	if result.Stage != models.CKMStageV2_1 {
		t.Errorf("expected Stage 1, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage1_AsianBMI(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 35, BMI: 24.0, EGFR: 90, AsianBMICutoffs: true,
	})
	if result.Stage != models.CKMStageV2_1 {
		t.Errorf("expected Stage 1 with Asian BMI cutoff 23, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage2_Diabetes(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 50, BMI: 28.0, EGFR: 75, HasDiabetes: true,
	})
	if result.Stage != models.CKMStageV2_2 {
		t.Errorf("expected Stage 2, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage2_CKD(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 60, BMI: 24.0, EGFR: 48, ACR: floatP(45.0),
	})
	if result.Stage != models.CKMStageV2_2 {
		t.Errorf("expected Stage 2 with eGFR <60, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage3_HighRisk(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 55, BMI: 30.0, EGFR: 65, HasDiabetes: true, HasHTN: true,
		PREVENTScore: floatP(15.0),
	})
	if result.Stage != models.CKMStageV2_3 {
		t.Errorf("expected Stage 3, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage4a_SubclinicalCAC(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 58, BMI: 29.0, EGFR: 60, HasDiabetes: true, HasHTN: true,
		CACScore: floatP(350.0),
	})
	if result.Stage != models.CKMStageV2_4a {
		t.Errorf("expected Stage 4a, got %s", result.Stage)
	}
	if result.Metadata.CACScore == nil || *result.Metadata.CACScore != 350.0 {
		t.Error("expected CACScore=350 in metadata")
	}
}

func TestClassifyCKM_Stage4b_PriorMI(t *testing.T) {
	miDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 62, BMI: 27.0, EGFR: 55, HasDiabetes: true,
		ASCVDEvents: []models.ASCVDEvent{
			{EventType: "MI", EventDate: miDate, Details: "STEMI anterior"},
		},
	})
	if result.Stage != models.CKMStageV2_4b {
		t.Errorf("expected Stage 4b, got %s", result.Stage)
	}
	if len(result.Metadata.ASCVDEvents) != 1 {
		t.Error("expected 1 ASCVD event in metadata")
	}
}

func TestClassifyCKM_Stage4c_HFrEF(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 58, BMI: 30.0, EGFR: 45,
		HasHeartFailure: true, LVEF: floatP(35.0), NYHAClass: "II",
	})
	if result.Stage != models.CKMStageV2_4c {
		t.Errorf("expected Stage 4c, got %s", result.Stage)
	}
	if result.Metadata.HFClassification != models.HFTypeReduced {
		t.Errorf("expected HFrEF, got %s", result.Metadata.HFClassification)
	}
}

func TestClassifyCKM_Stage4c_HFpEF(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 68, BMI: 35.0, EGFR: 50, HasDiabetes: true,
		HasHeartFailure: true, LVEF: floatP(55.0), NYHAClass: "III",
	})
	if result.Stage != models.CKMStageV2_4c {
		t.Errorf("expected Stage 4c, got %s", result.Stage)
	}
	if result.Metadata.HFClassification != models.HFTypePreserved {
		t.Errorf("expected HFpEF, got %s", result.Metadata.HFClassification)
	}
}

func TestClassifyCKM_Stage4c_Rheumatic(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 32, BMI: 22.0, EGFR: 80,
		HasHeartFailure: true, LVEF: floatP(45.0), NYHAClass: "II",
		HFEtiology: "RHEUMATIC", RheumaticEtiology: true,
	})
	if result.Stage != models.CKMStageV2_4c {
		t.Errorf("expected Stage 4c, got %s", result.Stage)
	}
	if !result.Metadata.RheumaticEtiology {
		t.Error("expected RheumaticEtiology=true")
	}
}

func TestClassifyCKM_Hierarchy_4c_Trumps_4b(t *testing.T) {
	// Patient with both HF and prior MI → 4c wins (higher severity)
	miDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 65, BMI: 28.0, EGFR: 40,
		HasHeartFailure: true, LVEF: floatP(30.0), NYHAClass: "III",
		ASCVDEvents: []models.ASCVDEvent{{EventType: "MI", EventDate: miDate}},
	})
	if result.Stage != models.CKMStageV2_4c {
		t.Errorf("expected 4c to trump 4b, got %s", result.Stage)
	}
}
```

---

- [ ] **Step 5: Implement CKM classifier**

Create `kb-20-patient-profile/internal/services/ckm_classifier.go`:

```go
package services

import (
	"fmt"
	"time"

	"kb-patient-profile/internal/models"
)

// CKMClassifierInput holds all data needed for CKM staging.
type CKMClassifierInput struct {
	Age         int
	BMI         float64
	WaistCm     float64
	Sex         string
	HasDiabetes    bool
	HasPrediabetes bool
	HasHTN         bool
	HasDyslipidemia bool
	HasMetSyndrome  bool
	HbA1c          *float64
	EGFR           float64
	ACR            *float64
	ASCVD10YearRisk *float64
	PREVENTScore    *float64
	CACScore       *float64
	CIMTPercentile *int
	HasLVH         bool
	NTproBNP       *float64
	HasSubclinicalAtherosclerosis bool
	ASCVDEvents    []models.ASCVDEvent
	HasHeartFailure bool
	LVEF           *float64
	NYHAClass      string
	HFEtiology     string
	AsianBMICutoffs bool
	RheumaticEtiology bool
}

// ClassifyCKMStage computes the full CKM stage with substage.
// Implements Ndumele et al. 2023 (Circulation) staging algorithm.
// Hierarchy: 4c > 4b > 4a > 3 > 2 > 1 > 0.
func ClassifyCKMStage(input CKMClassifierInput) models.CKMStageResult {
	result := models.CKMStageResult{
		Metadata: models.SubstageMetadata{
			StagingDate:   time.Now(),
			StagingSource: "ALGORITHM",
		},
	}

	// Stage 4c: Heart failure — highest, check first
	if input.HasHeartFailure {
		result.Stage = models.CKMStageV2_4c
		result.StagingRationale = "Heart failure in CKM context"

		if input.LVEF != nil {
			hfType := models.ClassifyHFType(*input.LVEF)
			result.Metadata.HFClassification = hfType
			result.Metadata.LVEFPercent = input.LVEF
			result.StagingRationale += " — " + string(hfType)
		}
		result.Metadata.NYHAClass = input.NYHAClass
		result.Metadata.NTproBNP = input.NTproBNP
		result.Metadata.HFEtiology = input.HFEtiology
		result.Metadata.RheumaticEtiology = input.RheumaticEtiology
		return result
	}

	// Stage 4b: Clinical ASCVD
	if len(input.ASCVDEvents) > 0 {
		result.Stage = models.CKMStageV2_4b
		result.Metadata.ASCVDEvents = input.ASCVDEvents

		mostRecent := input.ASCVDEvents[0].EventDate
		for _, e := range input.ASCVDEvents[1:] {
			if e.EventDate.After(mostRecent) {
				mostRecent = e.EventDate
			}
		}
		result.Metadata.MostRecentEventDate = &mostRecent
		result.StagingRationale = "Clinical ASCVD history: " + input.ASCVDEvents[0].EventType
		return result
	}

	// Stage 4a: Subclinical CVD
	hasSubclinical := false
	var markers []models.SubclinicalMarker

	if input.CACScore != nil && *input.CACScore > 0 {
		hasSubclinical = true
		result.Metadata.CACScore = input.CACScore
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "CAC", Value: fmt.Sprintf("%.0f", *input.CACScore), Date: time.Now(),
		})
	}
	if input.CIMTPercentile != nil && *input.CIMTPercentile > 75 {
		hasSubclinical = true
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "CIMT", Value: fmt.Sprintf(">p%d", *input.CIMTPercentile), Date: time.Now(),
		})
	}
	if input.HasLVH {
		hasSubclinical = true
		result.Metadata.HasLVH = true
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "LVH", Value: "present", Date: time.Now(),
		})
	}
	if input.NTproBNP != nil && *input.NTproBNP > 125 {
		hasSubclinical = true
		result.Metadata.NTproBNP = input.NTproBNP
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "NT_PROBNP_ELEVATED",
			Value:      fmt.Sprintf("%.0f pg/mL", *input.NTproBNP),
			Date:       time.Now(),
		})
	}
	if input.HasSubclinicalAtherosclerosis {
		hasSubclinical = true
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "SUBCLINICAL_ATHEROSCLEROSIS", Value: "present", Date: time.Now(),
		})
	}

	if hasSubclinical {
		result.Stage = models.CKMStageV2_4a
		result.Metadata.SubclinicalMarkers = markers
		result.StagingRationale = fmt.Sprintf("Subclinical CVD: %d marker(s) detected", len(markers))
		return result
	}

	// Stage 3: High predicted risk
	highRisk := false
	if input.PREVENTScore != nil && *input.PREVENTScore >= 10.0 {
		highRisk = true
	} else if input.ASCVD10YearRisk != nil && *input.ASCVD10YearRisk >= 7.5 {
		highRisk = true
	}
	if input.EGFR < 30 || (input.ACR != nil && *input.ACR >= 300) {
		highRisk = true
	}

	if highRisk && (input.HasDiabetes || input.HasHTN || input.HasDyslipidemia) {
		result.Stage = models.CKMStageV2_3
		result.StagingRationale = "High predicted ASCVD risk with metabolic risk factors"
		return result
	}

	// Stage 2: Metabolic risk factors or CKD
	hasMetabolic := input.HasDiabetes || input.HasHTN || input.HasDyslipidemia || input.HasMetSyndrome
	hasCKD := input.EGFR < 60 || (input.ACR != nil && *input.ACR >= 30)

	if hasMetabolic || hasCKD {
		result.Stage = models.CKMStageV2_2
		result.StagingRationale = "Metabolic risk factors and/or CKD present"
		return result
	}

	// Stage 1: Excess adiposity
	bmiOverweight := 25.0
	if input.AsianBMICutoffs {
		bmiOverweight = 23.0
	}
	if input.BMI >= bmiOverweight {
		result.Stage = models.CKMStageV2_1
		result.StagingRationale = "Excess adiposity without metabolic derangement"
		return result
	}

	// Stage 0: No CKM risk factors
	result.Stage = models.CKMStageV2_0
	result.StagingRationale = "No CKM risk factors identified"
	return result
}
```

---

- [ ] **Step 6: Run classifier tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/services/ -run "TestClassifyCKM" -v -count=1
```

Expected: All 12 tests PASS.

---

- [ ] **Step 7: Commit Phase K1**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/ckm_stage_v2.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/ckm_stage.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/patient_profile.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ckm_classifier.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/ckm_classifier_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/migrations/005_ckm_substage.sql
git commit -m "feat(kb20): CKM Stage V2 — 4a/4b/4c substages per Ndumele 2023

CKMStageValue string type (0/1/2/3/4a/4b/4c), SubstageMetadata JSONB
(HF type, ASCVD events, subclinical markers), ClassifyCKMStage() per
Ndumele 2023 hierarchy: 4c>4b>4a>3>2>1>0. Asian BMI cutoffs, rheumatic
etiology flag. Migration: existing Stage 4 -> 4a with review_needed."
```

---

## Phase K2: Substage-Aware Decision Cards — KB-23 (Steps 8–14)

---

- [ ] **Step 8: Create Stage 4 pathway definitions YAML**

Create `backend/shared-infrastructure/market-configs/shared/ckm_stage4_pathways.yaml`:

```yaml
# Mandatory and recommended medications by CKM Stage 4 substage.
# Sources: AHA/ACC/HFSA 2022, ESC 2023, ADA SOC 2025, KDIGO 2024

stage_4a:
  label: "Subclinical CVD in CKM"
  strategy: "AGGRESSIVE_PREVENTION"
  mandatory_medications:
    - class: "STATIN"
      intensity: "HIGH"
      rationale: "All Stage 4a require high-intensity statin (AHA/ACC 2018)"
    - class: "ACEi_OR_ARB"
      condition: "IF_DIABETIC_OR_CKD"
      rationale: "Renoprotective + CV risk reduction"
  recommended_medications:
    - class: "SGLT2i"
      condition: "IF_EGFR_GE_20"
      rationale: "Cardiorenal protection (EMPA-REG, DAPA-CKD)"
    - class: "GLP1_RA"
      condition: "IF_OBESE_OR_ASCVD_HIGH"
      rationale: "CV benefit + weight management (LEADER, SUSTAIN-6)"
  contraindicated: []
  target_adjustments:
    sbp_target: 130
    ldl_target: 70

stage_4b:
  label: "Clinical ASCVD in CKM"
  strategy: "SECONDARY_PREVENTION"
  mandatory_medications:
    - class: "STATIN"
      intensity: "HIGH"
      rationale: "Secondary prevention (AHA/ACC 2018)"
    - class: "ANTIPLATELET"
      rationale: "Aspirin +/- P2Y12 inhibitor post-ACS"
    - class: "ACEi_OR_ARB"
      rationale: "Post-MI cardioprotection + renoprotection"
    - class: "BETA_BLOCKER"
      condition: "IF_POST_MI"
      rationale: "Post-MI mortality reduction (CAPRICORN)"
  recommended_medications:
    - class: "SGLT2i"
      rationale: "CV risk reduction (EMPA-REG)"
      priority: "HIGH"
    - class: "GLP1_RA_CV_PROVEN"
      rationale: "Liraglutide/semaglutide/dulaglutide — proven CV benefit"
    - class: "EZETIMIBE"
      condition: "IF_LDL_ABOVE_70_ON_MAX_STATIN"
      rationale: "IMPROVE-IT post-ACS"
  contraindicated: []
  target_adjustments:
    sbp_target: 130
    ldl_target: 55

stage_4c:
  label: "Heart Failure in CKM"
  strategy: "HF_GUIDELINE_DIRECTED"

  hfref:
    label: "HFrEF — Four Pillars"
    mandatory_medications:
      - class: "ARNI_OR_ACEi_ARB"
        preferred: "SACUBITRIL_VALSARTAN"
        rationale: "PARADIGM-HF: ARNI preferred; ACEi/ARB if ARNI not tolerated"
      - class: "BETA_BLOCKER_HF"
        rationale: "CIBIS-II, MERIT-HF, COPERNICUS — mortality reduction"
        note: "NOT atenolol"
      - class: "MRA"
        rationale: "RALES, EMPHASIS-HF"
        condition: "IF_K_BELOW_5_AND_EGFR_GE_30"
      - class: "SGLT2i"
        rationale: "DAPA-HF, EMPEROR-Reduced"
    contraindicated:
      - class: "PIOGLITAZONE"
        reason: "Fluid retention exacerbates HF (FDA black box)"
      - class: "SAXAGLIPTIN"
        reason: "SAVOR-TIMI 53: increased HF hospitalization"
      - class: "ALOGLIPTIN"
        reason: "EXAMINE: potential HF risk signal"
      - class: "NON_DHP_CCB"
        reason: "Negative inotropic effect in HFrEF"

  hfmref:
    label: "HFmrEF — Emerging Evidence"
    mandatory_medications:
      - class: "SGLT2i"
        rationale: "DELIVER + EMPEROR-Preserved subgroup"
      - class: "ACEi_OR_ARB"
        rationale: "Reasonable from HFrEF extrapolation"
    contraindicated:
      - class: "PIOGLITAZONE"
        reason: "Fluid retention"

  hfpef:
    label: "HFpEF — Limited Disease-Modifying"
    mandatory_medications:
      - class: "SGLT2i"
        rationale: "EMPEROR-Preserved, DELIVER — ONLY proven disease-modifying therapy"
    recommended_medications:
      - class: "GLP1_RA"
        condition: "IF_OBESE"
        rationale: "STEP-HFpEF: semaglutide improved symptoms in obese HFpEF"
      - class: "MRA"
        condition: "IF_HOSPITALIZED_RECENTLY"
        rationale: "TOPCAT Americas subgroup"
    contraindicated:
      - class: "PIOGLITAZONE"
        reason: "Fluid retention"
```

---

- [ ] **Step 9: Create India and Australia Stage 4 overrides**

Create `backend/shared-infrastructure/market-configs/india/ckm_stage4_overrides.yaml`:

```yaml
# Sources: CSI 2023, RSSDI 2023

stage_4b_overrides:
  antiplatelet_preference:
    first_line: "ASPIRIN_PLUS_CLOPIDOGREL"
    alternative: "ASPIRIN_PLUS_TICAGRELOR"
    cost_note: "Clopidogrel Rs30-80/month vs ticagrelor Rs1500-2500/month"
  statin_preference:
    first_line: "ATORVASTATIN_GENERIC"
    max_dose_commonly_available: 80

stage_4c_overrides:
  arni:
    availability: "MODERATE"
    cost: "Rs2500-4000/month"
    alternative_if_unavailable: "ENALAPRIL_OR_RAMIPRIL"
    cost_note: "If ARNI unaffordable, use ACEi at maximum tolerated dose"
  rheumatic_hf:
    enabled: true
    additional_medications:
      - class: "WARFARIN"
        condition: "IF_MITRAL_STENOSIS_OR_AF"
        monitoring: "INR 2.0-3.0, monthly"
      - class: "PENICILLIN_PROPHYLAXIS"
        condition: "IF_AGE_UNDER_40_OR_RECURRENT_RF"
    referral: "Cardiothoracic surgery evaluation"
  sglt2i_hf:
    availability: "HIGH"
    cost_note: "Rs800-1500/month — affordable for most urban patients"

young_onset_cvd:
  enabled: true
  age_threshold: 45
  additional_screening: ["FAMILIAL_HYPERCHOLESTEROLAEMIA", "LIPOPROTEIN_A"]
```

Create `backend/shared-infrastructure/market-configs/australia/ckm_stage4_overrides.yaml`:

```yaml
# Sources: RACGP 2024, NHF 2023, PBS Schedule

stage_4b_overrides:
  pbs_authority:
    ticagrelor: "PBS_ITEM_10400"
    prasugrel: "PBS_ITEM_10401"
    eplerenone: "PBS_ITEM_9483"

stage_4c_overrides:
  pbs_authority:
    sacubitril_valsartan: "PBS_ITEM_11626"
    note: "Requires specialist initiation for HFrEF EF<=40%"
  indigenous_screening:
    echo_from_age: 5
    rheumatic_screening: true
    source: "NACCHO_2023, RHDAustralia"

nvdpa_integration:
  enabled: true
  use_absolute_cvd_risk: true
  tool: "AUSTRALIAN_ABSOLUTE_CVD_RISK_CALCULATOR"
  threshold_for_stage3: 10
```

---

- [ ] **Step 10: Write failing test for HF medication gate**

Create `kb-23-decision-cards/internal/services/hf_medication_gate_test.go`:

```go
package services

import (
	"testing"

	"kb-23-decision-cards/internal/models"
)

// We need CKMStageValue and HFType from KB-20's model.
// Since KB-23 can't import KB-20, we define local aliases that
// mirror the values. In production, these come via HTTP/gRPC.
// For testing, we use string constants directly.

func TestHFGate_Pioglitazone_Blocked_4c_HFrEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, reason := gate.CheckContraindication("PIOGLITAZONE", "4c", "HFrEF")
	if !blocked {
		t.Error("pioglitazone should be blocked in 4c-HFrEF")
	}
	if reason == "" {
		t.Error("expected reason for pioglitazone block")
	}
}

func TestHFGate_Saxagliptin_Blocked_4c_HFrEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("SAXAGLIPTIN", "4c", "HFrEF")
	if !blocked {
		t.Error("saxagliptin should be blocked in 4c-HFrEF")
	}
}

func TestHFGate_NonDHP_CCB_Blocked_4c_HFrEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("NON_DHP_CCB", "4c", "HFrEF")
	if !blocked {
		t.Error("non-DHP CCB should be blocked in 4c-HFrEF")
	}
}

func TestHFGate_Metformin_Allowed_4c(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("METFORMIN", "4c", "HFrEF")
	if blocked {
		t.Error("metformin should NOT be blocked in HF")
	}
}

func TestHFGate_Pioglitazone_Allowed_Non4c(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("PIOGLITAZONE", "4b", "")
	if blocked {
		t.Error("pioglitazone should be allowed in non-4c stages")
	}
}

func TestHFGate_Pioglitazone_Blocked_4c_HFpEF(t *testing.T) {
	gate := NewHFMedicationGate()
	blocked, _ := gate.CheckContraindication("PIOGLITAZONE", "4c", "HFpEF")
	if !blocked {
		t.Error("pioglitazone should be blocked in ALL HF types including HFpEF")
	}
}
```

---

- [ ] **Step 11: Implement HF medication gate**

Create `kb-23-decision-cards/internal/services/hf_medication_gate.go`:

```go
package services

import "fmt"

// HFContraindication defines a drug class that is contraindicated in HF.
type HFContraindication struct {
	DrugClass   string   // e.g., "PIOGLITAZONE"
	HFTypes     []string // empty = ALL HF types; ["HFrEF"] = only HFrEF
	Reason      string
	SourceTrial string
}

// HFMedicationGate checks drug contraindications specific to heart failure.
type HFMedicationGate struct {
	contraindications []HFContraindication
}

func NewHFMedicationGate() *HFMedicationGate {
	return &HFMedicationGate{
		contraindications: []HFContraindication{
			{
				DrugClass:   "PIOGLITAZONE",
				HFTypes:     nil, // ALL HF types
				Reason:      "Fluid retention exacerbates heart failure",
				SourceTrial: "FDA_BLACK_BOX",
			},
			{
				DrugClass:   "SAXAGLIPTIN",
				HFTypes:     nil, // ALL HF types — conservative
				Reason:      "Increased HF hospitalization risk",
				SourceTrial: "SAVOR-TIMI_53",
			},
			{
				DrugClass:   "ALOGLIPTIN",
				HFTypes:     nil,
				Reason:      "Potential HF risk signal",
				SourceTrial: "EXAMINE",
			},
			{
				DrugClass:   "NON_DHP_CCB",
				HFTypes:     []string{"HFrEF"}, // only HFrEF
				Reason:      "Negative inotropic effect",
				SourceTrial: "AHA_ACC_HFSA_2022",
			},
		},
	}
}

// CheckContraindication returns (blocked, reason) if the drug is contraindicated
// for the patient's CKM substage and HF type. Only blocks for Stage 4c.
func (g *HFMedicationGate) CheckContraindication(drugClass, ckmStage, hfType string) (bool, string) {
	if ckmStage != "4c" {
		return false, ""
	}

	for _, ci := range g.contraindications {
		if ci.DrugClass != drugClass {
			continue
		}

		// If HFTypes is empty, applies to ALL HF types
		if len(ci.HFTypes) == 0 {
			return true, fmt.Sprintf("CONTRAINDICATED in heart failure: %s (%s)", ci.Reason, ci.SourceTrial)
		}

		// Check if patient's HF type is in the contraindicated list
		for _, ht := range ci.HFTypes {
			if ht == hfType {
				return true, fmt.Sprintf("CONTRAINDICATED in %s: %s (%s)", hfType, ci.Reason, ci.SourceTrial)
			}
		}
	}

	return false, ""
}
```

---

- [ ] **Step 12: Write failing test for mandatory medication checker**

Create `kb-23-decision-cards/internal/services/mandatory_med_checker_test.go`:

```go
package services

import (
	"testing"
)

func TestMandatoryMeds_4a_MissingStatin(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"METFORMIN", "ACEi"}

	gaps := checker.CheckMandatory("4a", "", activeMeds)

	hasStatinGap := false
	for _, g := range gaps {
		if g.MissingClass == "STATIN" {
			hasStatinGap = true
			if g.Urgency != "URGENT" {
				t.Errorf("expected URGENT for missing statin in 4a, got %s", g.Urgency)
			}
		}
	}
	if !hasStatinGap {
		t.Error("should flag missing statin for Stage 4a")
	}
}

func TestMandatoryMeds_4b_AllPresent(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"STATIN", "ASPIRIN", "ACEi", "BETA_BLOCKER", "SGLT2i"}

	gaps := checker.CheckMandatory("4b", "", activeMeds)

	for _, g := range gaps {
		if g.Urgency == "IMMEDIATE" {
			t.Errorf("unexpected IMMEDIATE gap when all mandatory present: %s", g.MissingClass)
		}
	}
}

func TestMandatoryMeds_4c_HFrEF_MissingPillars(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"ACEi", "METFORMIN"} // missing: BB, MRA, SGLT2i

	gaps := checker.CheckMandatory("4c", "HFrEF", activeMeds)

	missingClasses := map[string]bool{}
	for _, g := range gaps {
		missingClasses[g.MissingClass] = true
	}
	if !missingClasses["SGLT2i"] {
		t.Error("should flag missing SGLT2i for HFrEF")
	}
	if !missingClasses["BETA_BLOCKER_HF"] {
		t.Error("should flag missing beta-blocker for HFrEF")
	}
	if !missingClasses["MRA"] {
		t.Error("should flag missing MRA for HFrEF")
	}
}

func TestMandatoryMeds_4c_HFpEF_OnlySGLT2i(t *testing.T) {
	checker := NewMandatoryMedChecker()
	activeMeds := []string{"ACEi", "THIAZIDE", "METFORMIN"}

	gaps := checker.CheckMandatory("4c", "HFpEF", activeMeds)

	hasSGLT2iGap := false
	for _, g := range gaps {
		if g.MissingClass == "SGLT2i" {
			hasSGLT2iGap = true
		}
	}
	if !hasSGLT2iGap {
		t.Error("should flag missing SGLT2i — only mandatory disease-modifying therapy for HFpEF")
	}
}
```

---

- [ ] **Step 13: Implement mandatory medication checker**

Create `kb-23-decision-cards/internal/services/mandatory_med_checker.go`:

```go
package services

// MandatoryMedGap represents a medication that should be present but isn't.
type MandatoryMedGap struct {
	MissingClass string `json:"missing_class"`
	Rationale    string `json:"rationale"`
	Urgency      string `json:"urgency"`
	Alternative  string `json:"alternative,omitempty"`
	SourceTrial  string `json:"source_trial"`
}

type MandatoryMedChecker struct{}

func NewMandatoryMedChecker() *MandatoryMedChecker {
	return &MandatoryMedChecker{}
}

func (c *MandatoryMedChecker) CheckMandatory(
	ckmStage string,
	hfType string,
	activeMedClasses []string,
) []MandatoryMedGap {
	activeSet := make(map[string]bool)
	for _, mc := range activeMedClasses {
		activeSet[mc] = true
	}

	// Normalize: ACEi or ARB counts as RAS blockade
	hasRAS := activeSet["ACEi"] || activeSet["ARB"] || activeSet["ARNI"] || activeSet["SACUBITRIL_VALSARTAN"]
	hasAntiplatelet := activeSet["ASPIRIN"] || activeSet["CLOPIDOGREL"] || activeSet["TICAGRELOR"] || activeSet["PRASUGREL"]
	hasBB := activeSet["BETA_BLOCKER"] || activeSet["BETA_BLOCKER_HF"] || activeSet["CARVEDILOL"] || activeSet["BISOPROLOL"]

	var gaps []MandatoryMedGap

	switch ckmStage {
	case "4a":
		if !activeSet["STATIN"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "STATIN",
				Rationale:    "All Stage 4a require high-intensity statin for subclinical CVD",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}

	case "4b":
		if !activeSet["STATIN"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "STATIN",
				Rationale:    "Secondary prevention requires high-intensity statin",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}
		if !hasAntiplatelet {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ANTIPLATELET",
				Rationale:    "Post-ASCVD requires antiplatelet therapy",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}
		if !hasRAS {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ACEi_OR_ARB",
				Rationale:    "Post-MI cardioprotection + renoprotection",
				Urgency:      "URGENT",
				SourceTrial:  "HOPE, EUROPA",
			})
		}

	case "4c":
		gaps = append(gaps, c.checkHFMandatory(hfType, activeSet, hasRAS, hasBB)...)
	}

	return gaps
}

func (c *MandatoryMedChecker) checkHFMandatory(
	hfType string,
	activeSet map[string]bool,
	hasRAS bool,
	hasBB bool,
) []MandatoryMedGap {
	var gaps []MandatoryMedGap

	switch hfType {
	case "HFrEF":
		// Four pillars: ARNI/ACEi/ARB, BB, MRA, SGLT2i
		if !hasRAS {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ARNI_OR_ACEi_ARB",
				Rationale:    "PARADIGM-HF: ARNI preferred; ACEi/ARB if not tolerated",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "PARADIGM-HF",
			})
		}
		if !hasBB {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "BETA_BLOCKER_HF",
				Rationale:    "HFrEF mortality reduction — carvedilol/bisoprolol/metoprolol succinate",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "CIBIS-II, MERIT-HF, COPERNICUS",
			})
		}
		if !activeSet["MRA"] && !activeSet["SPIRONOLACTONE"] && !activeSet["EPLERENONE"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "MRA",
				Rationale:    "HFrEF mortality + hospitalization reduction",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "RALES, EMPHASIS-HF",
			})
		}
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "HFrEF mortality + hospitalization reduction",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "DAPA-HF, EMPEROR-Reduced",
			})
		}

	case "HFmrEF":
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "Benefit across EF spectrum (DELIVER subgroup)",
				Urgency:      "URGENT",
				SourceTrial:  "DELIVER, EMPEROR-Preserved",
			})
		}
		if !hasRAS {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ACEi_OR_ARB",
				Rationale:    "Reasonable from HFrEF extrapolation",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_HFSA_2022",
			})
		}

	case "HFpEF":
		// ONLY mandatory: SGLT2i
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "ONLY proven disease-modifying therapy for HFpEF (EMPEROR-Preserved, DELIVER)",
				Urgency:      "URGENT",
				SourceTrial:  "EMPEROR-Preserved, DELIVER",
			})
		}

	default:
		// Unknown HF type — at minimum, SGLT2i
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "SGLT2i beneficial across HF spectrum",
				Urgency:      "URGENT",
				SourceTrial:  "DAPA-HF, EMPEROR-Preserved",
			})
		}
	}

	return gaps
}
```

---

- [ ] **Step 14: Run Phase K2 tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestHFGate|TestMandatoryMeds" -v -count=1
```

Expected: All 10 tests PASS (6 HF gate + 4 mandatory med).

---

- [ ] **Step 15: Integrate HF gate into conflict detector**

Modify `kb-23-decision-cards/internal/services/conflict_detector.go` — add HF medication gating to `DetectAllConflicts`. Add after the stale eGFR check:

```go
// Add field to EnrichedConflictReport:
	HFContraindications []HFBlockResult `json:"hf_contraindications,omitempty"`

// Add type:
type HFBlockResult struct {
	DrugClass string `json:"drug_class"`
	Reason    string `json:"reason"`
}

// In DetectAllConflicts, after stale eGFR logic, add:
	// HF medication gating (Stage 4c)
	if ckmStage == "4c" {
		hfGate := NewHFMedicationGate()
		for _, m := range meds {
			blocked, reason := hfGate.CheckContraindication(m.DrugClass, ckmStage, hfType)
			if blocked {
				report.HasSafetyBlock = true
				report.HFContraindications = append(report.HFContraindications, HFBlockResult{
					DrugClass: m.DrugClass,
					Reason:    reason,
				})
				report.BlockedDrugClasses = append(report.BlockedDrugClasses, m.DrugClass)
			}
		}
	}
```

Note: `DetectAllConflicts` needs two new parameters: `ckmStage string` and `hfType string`. Update the function signature and all callers (renal_integration_test.go) to pass `""` for backward compatibility.

---

- [ ] **Step 16: Commit Phase K2**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/market-configs/shared/ckm_stage4_pathways.yaml
git add backend/shared-infrastructure/market-configs/india/ckm_stage4_overrides.yaml
git add backend/shared-infrastructure/market-configs/australia/ckm_stage4_overrides.yaml
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/hf_medication_gate.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/hf_medication_gate_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/mandatory_med_checker.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/mandatory_med_checker_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/conflict_detector.go
git commit -m "feat(kb23): CKM substage-aware cards — HF gate + mandatory med checker

HFMedicationGate blocks pioglitazone (all HF), saxagliptin (SAVOR-TIMI
53), alogliptin (EXAMINE), non-DHP CCB (HFrEF only). MandatoryMedChecker:
4a=statin, 4b=statin+antiplatelet+ACEi/ARB, 4c-HFrEF=four pillars
(ARNI+BB+MRA+SGLT2i), 4c-HFpEF=SGLT2i only. Stage 4 pathway YAMLs
with India (clopidogrel preference, ARNI cost) and Australia (PBS codes,
indigenous rheumatic screening)."
```

---

## Phase K3: MHRI Substage Adjustment + Full Regression (Steps 17–18)

---

- [ ] **Step 17: Write failing test for MHRI substage adjustment**

Create `kb-26-metabolic-digital-twin/internal/services/ckm_mhri_adjustment_test.go`:

```go
package services

import (
	"testing"
)

func TestMHRI_Stage4a_PreventiveWeighting(t *testing.T) {
	adjustment := ComputeCKMSubstageAdjustment("4a", "", "")
	if adjustment.CardioDomainWeight <= 0.25 {
		t.Errorf("4a should increase cardio weight above default 0.25, got %.2f",
			adjustment.CardioDomainWeight)
	}
}

func TestMHRI_Stage4c_HFrEF_Adjustment(t *testing.T) {
	adjustment := ComputeCKMSubstageAdjustment("4c", "HFrEF", "III")
	if adjustment.CardioDomainWeight < 0.35 {
		t.Errorf("4c-HFrEF should have cardio weight >= 0.35, got %.2f",
			adjustment.CardioDomainWeight)
	}
	if adjustment.ScoreCeiling >= 60 {
		t.Errorf("NYHA III should cap score below 60, got %.1f", adjustment.ScoreCeiling)
	}
}

func TestMHRI_Stage4c_HFpEF_Adjustment(t *testing.T) {
	adjustment := ComputeCKMSubstageAdjustment("4c", "HFpEF", "II")
	if adjustment.BehavioralDomainWeight <= 0.15 {
		t.Errorf("4c-HFpEF should increase behavioral weight above 0.15, got %.2f",
			adjustment.BehavioralDomainWeight)
	}
}

func TestMHRI_Stage2_NoAdjustment(t *testing.T) {
	adjustment := ComputeCKMSubstageAdjustment("2", "", "")
	if adjustment.GlucoseDomainWeight != 0.35 {
		t.Errorf("Stage 2 should have default glucose weight 0.35, got %.2f",
			adjustment.GlucoseDomainWeight)
	}
	if adjustment.CardioDomainWeight != 0.25 {
		t.Errorf("Stage 2 should have default cardio weight 0.25, got %.2f",
			adjustment.CardioDomainWeight)
	}
	if adjustment.ScoreCeiling != 100.0 {
		t.Errorf("Stage 2 should have no ceiling (100), got %.1f", adjustment.ScoreCeiling)
	}
}
```

---

- [ ] **Step 18: Implement MHRI substage adjustment**

Create `kb-26-metabolic-digital-twin/internal/services/ckm_mhri_adjustment.go`:

```go
package services

// MHRISubstageAdjustment modifies MHRI domain weights and score interpretation
// based on CKM substage. Default weights: glucose 35%, cardio 25%, body_comp 25%, behavioral 15%.
type MHRISubstageAdjustment struct {
	GlucoseDomainWeight    float64 `json:"glucose_domain_weight"`
	CardioDomainWeight     float64 `json:"cardio_domain_weight"`
	BodyCompDomainWeight   float64 `json:"body_comp_domain_weight"`
	BehavioralDomainWeight float64 `json:"behavioral_domain_weight"`
	ScoreCeiling           float64 `json:"score_ceiling"`
	InterpretationNote     string  `json:"interpretation_note"`
}

// ComputeCKMSubstageAdjustment returns adjusted MHRI weights for a CKM substage.
// Uses string parameters to avoid cross-module import from KB-20.
func ComputeCKMSubstageAdjustment(ckmStage, hfType, nyhaClass string) MHRISubstageAdjustment {
	adj := MHRISubstageAdjustment{
		GlucoseDomainWeight:    0.35,
		CardioDomainWeight:     0.25,
		BodyCompDomainWeight:   0.25,
		BehavioralDomainWeight: 0.15,
		ScoreCeiling:           100.0,
	}

	switch ckmStage {
	case "4a":
		adj.CardioDomainWeight = 0.30
		adj.GlucoseDomainWeight = 0.30
		adj.BodyCompDomainWeight = 0.25
		adj.BehavioralDomainWeight = 0.15
		adj.InterpretationNote = "Stage 4a: cardio domain weighted for subclinical CVD monitoring"

	case "4b":
		adj.CardioDomainWeight = 0.35
		adj.GlucoseDomainWeight = 0.30
		adj.BodyCompDomainWeight = 0.20
		adj.BehavioralDomainWeight = 0.15
		adj.ScoreCeiling = 85.0
		adj.InterpretationNote = "Stage 4b: secondary prevention; ceiling 85 due to residual ASCVD risk"

	case "4c":
		switch hfType {
		case "HFrEF":
			adj.CardioDomainWeight = 0.40
			adj.GlucoseDomainWeight = 0.25
			adj.BodyCompDomainWeight = 0.20
			adj.BehavioralDomainWeight = 0.15
			adj.InterpretationNote = "Stage 4c HFrEF: cardio dominant — track EF, NT-proBNP"
		case "HFmrEF":
			adj.CardioDomainWeight = 0.35
			adj.GlucoseDomainWeight = 0.25
			adj.BodyCompDomainWeight = 0.20
			adj.BehavioralDomainWeight = 0.20
			adj.InterpretationNote = "Stage 4c HFmrEF: balanced cardio + behavioral"
		case "HFpEF":
			adj.CardioDomainWeight = 0.25
			adj.GlucoseDomainWeight = 0.25
			adj.BodyCompDomainWeight = 0.25
			adj.BehavioralDomainWeight = 0.25
			adj.InterpretationNote = "Stage 4c HFpEF: equal weighting — obesity, exercise, comorbidity"
		default:
			adj.ScoreCeiling = 70.0
			adj.InterpretationNote = "Stage 4c: HF subtype unknown — conservative ceiling"
			return adj
		}

		// NYHA class ceiling
		switch nyhaClass {
		case "IV":
			adj.ScoreCeiling = 30.0
		case "III":
			adj.ScoreCeiling = 50.0
		case "II":
			adj.ScoreCeiling = 70.0
		case "I":
			adj.ScoreCeiling = 85.0
		}
	}

	return adj
}
```

---

- [ ] **Step 19: Run Phase K3 tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestMHRI_Stage" -v -count=1
```

Expected: All 4 tests PASS.

---

- [ ] **Step 20: Full regression across all 3 services**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./... -count=1

cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./... -count=1

cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./... -count=1
```

Expected: All tests PASS, zero regressions.

---

- [ ] **Step 21: Final commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/ckm_mhri_adjustment.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/ckm_mhri_adjustment_test.go
git commit -m "feat(kb26): MHRI substage-aware domain weighting

Stage 4a: cardio weight 30% (subclinical CVD prevention focus).
Stage 4b: cardio weight 35%, score ceiling 85 (residual post-ASCVD risk).
Stage 4c-HFrEF: cardio weight 40% (EF/NT-proBNP dominant).
Stage 4c-HFpEF: equal 25% all domains (phenotypic syndrome).
NYHA class ceilings: IV->30, III->50, II->70, I->85."
```

---

## Summary

| Phase | Steps | Key Deliverables |
|-------|-------|-----------------|
| K1: Data Model + Classifier | 1–7 | CKMStageValue string (0-4c), SubstageMetadata JSONB, ClassifyCKMStage() per Ndumele 2023, migration with review_needed flag |
| K2: Substage Cards | 8–16 | HFMedicationGate (pioglitazone/saxagliptin/non-DHP CCB), MandatoryMedChecker (4a→statin, 4b→secondary prevention, 4c-HFrEF→four pillars, 4c-HFpEF→SGLT2i only), pathway YAMLs |
| K3: MHRI + Regression | 17–21 | Substage-aware MHRI domain weights, NYHA class ceiling, full cross-service regression |

**Total: 21 steps, 20 files (14 create, 6 modify), 3 phases.**

All steps follow TDD: write test → verify fail → implement → verify pass → commit.
