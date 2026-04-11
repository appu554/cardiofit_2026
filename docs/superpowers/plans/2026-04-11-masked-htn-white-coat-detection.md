# Masked Hypertension & White-Coat Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a clinic-home BP discordance classifier (6 phenotypes) with cross-domain risk amplification and decision card generation for the CardioFit platform.

**Architecture:** KB-26 owns the core classification engine (ClassifyBPContext) that compares clinic vs home BP means against market-configurable thresholds, applying cross-domain amplification (diabetes, CKD, morning surge, engagement bias, medication timing). KB-23 consumes the classification to generate 8 decision card types and integrates with the existing four-pillar evaluator. Market YAML configs provide threshold overrides for India and Australia.

**Tech Stack:** Go 1.22+ (Gin, GORM), PostgreSQL 15, YAML market configs

---

## File Structure

### KB-26 (Metabolic Digital Twin)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-26-metabolic-digital-twin/internal/models/bp_context.go` | BPContextPhenotype enum, BPContextClassification struct, BPContextHistory GORM model |
| Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier.go` | ClassifyBPContext() — core engine with threshold comparison, cross-domain amplification, selection bias, medication timing |
| Create | `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier_test.go` | 14 test cases covering all phenotypes, amplifications, edge cases |
| Create | `kb-26-metabolic-digital-twin/migrations/006_bp_context.sql` | bp_context_history table for progression tracking |

### KB-23 (Decision Cards)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `kb-23-decision-cards/internal/models/bp_context.go` | Local mirror of BPContextClassification (KB-23 owns its models) |
| Create | `kb-23-decision-cards/internal/services/masked_htn_cards.go` | EvaluateMaskedHTNCards() — 8 card types with urgency logic |
| Create | `kb-23-decision-cards/internal/services/masked_htn_cards_test.go` | 7 test cases for card generation |
| Modify | `kb-23-decision-cards/internal/services/four_pillar_evaluator.go` | Add BPContext field to FourPillarInput, masked HTN logic in medication pillar |

### Market Configs
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `market-configs/shared/bp_context_thresholds.yaml` | Base thresholds, data requirements, amplification rules, selection bias config |
| Create | `market-configs/india/bp_context_overrides.yaml` | Wider WCE threshold (20mmHg), sodium correlation, device validation |
| Create | `market-configs/australia/bp_context_overrides.yaml` | ABPM MBS 11607, indigenous overrides, Heart Foundation alignment |

---

## Task 1: BP Context Model (KB-26)

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/models/bp_context.go`

- [ ] **Step 1: Create the BP context model file**

```go
// kb-26-metabolic-digital-twin/internal/models/bp_context.go
package models

import "time"

// BPContextPhenotype classifies the patient's clinic-home BP relationship.
type BPContextPhenotype string

const (
	PhenotypeSustainedHTN          BPContextPhenotype = "SUSTAINED_HTN"
	PhenotypeWhiteCoatHTN          BPContextPhenotype = "WHITE_COAT_HTN"
	PhenotypeMaskedHTN             BPContextPhenotype = "MASKED_HTN"
	PhenotypeSustainedNormotension BPContextPhenotype = "SUSTAINED_NORMOTENSION"
	PhenotypeMaskedUncontrolled    BPContextPhenotype = "MASKED_UNCONTROLLED"
	PhenotypeWhiteCoatUncontrolled BPContextPhenotype = "WHITE_COAT_UNCONTROLLED"
	PhenotypeInsufficientData      BPContextPhenotype = "INSUFFICIENT_DATA"
)

// BPContextClassification is the full output of the clinic-home discordance analysis.
type BPContextClassification struct {
	PatientID  string             `json:"patient_id"`
	Phenotype  BPContextPhenotype `json:"phenotype"`
	ComputedAt time.Time          `json:"computed_at"`

	// Clinic BP summary
	ClinicSBPMean        float64 `json:"clinic_sbp_mean"`
	ClinicDBPMean        float64 `json:"clinic_dbp_mean"`
	ClinicReadingCount   int     `json:"clinic_reading_count"`
	ClinicAboveThreshold bool    `json:"clinic_above_threshold"`

	// Home BP summary
	HomeSBPMean        float64 `json:"home_sbp_mean"`
	HomeDBPMean        float64 `json:"home_dbp_mean"`
	HomeReadingCount   int     `json:"home_reading_count"`
	HomeDaysWithData   int     `json:"home_days_with_data"`
	HomeAboveThreshold bool    `json:"home_above_threshold"`

	// Discordance metrics
	ClinicHomeGapSBP float64 `json:"clinic_home_gap_sbp"`
	ClinicHomeGapDBP float64 `json:"clinic_home_gap_dbp"`
	WhiteCoatEffect  float64 `json:"white_coat_effect_mmhg"`

	// Data quality
	SufficientClinic bool   `json:"sufficient_clinic"`
	SufficientHome   bool   `json:"sufficient_home"`
	Confidence       string `json:"confidence"`
	ClinicWindow     string `json:"clinic_window,omitempty"`
	HomeWindow       string `json:"home_window,omitempty"`

	// Cross-domain risk amplification
	IsDiabetic            bool   `json:"is_diabetic"`
	DiabetesAmplification bool   `json:"diabetes_amplification"`
	HasCKD                bool   `json:"has_ckd"`
	CKDAmplification      bool   `json:"ckd_amplification"`
	EngagementPhenotype   string `json:"engagement_phenotype,omitempty"`
	SelectionBiasRisk     bool   `json:"selection_bias_risk"`
	MorningSurgeCompound  bool   `json:"morning_surge_compound"`

	// Treatment context
	OnAntihypertensives        bool   `json:"on_antihypertensives"`
	MedicationTimingHypothesis string `json:"medication_timing_hypothesis,omitempty"`

	// Thresholds used (market-specific)
	ClinicSBPThreshold float64 `json:"clinic_sbp_threshold"`
	ClinicDBPThreshold float64 `json:"clinic_dbp_threshold"`
	HomeSBPThreshold   float64 `json:"home_sbp_threshold"`
	HomeDBPThreshold   float64 `json:"home_dbp_threshold"`
}

// BPContextHistory stores classification snapshots for progression tracking.
type BPContextHistory struct {
	ID            string             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID     string             `gorm:"size:100;index;not null" json:"patient_id"`
	SnapshotDate  time.Time          `gorm:"index;not null" json:"snapshot_date"`
	Phenotype     BPContextPhenotype `gorm:"size:30;not null" json:"phenotype"`
	ClinicSBPMean float64            `json:"clinic_sbp_mean"`
	HomeSBPMean   float64            `json:"home_sbp_mean"`
	GapSBP        float64            `json:"gap_sbp"`
	Confidence    string             `gorm:"size:10" json:"confidence"`
	CreatedAt     time.Time          `json:"created_at"`
}
```

- [ ] **Step 2: Verify file compiles**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./internal/models/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/models/bp_context.go
git commit -m "feat(kb26): BP context phenotype model — 7 phenotypes + history table struct"
```

---

## Task 2: Market Configuration YAMLs

**Files:**
- Create: `market-configs/shared/bp_context_thresholds.yaml`
- Create: `market-configs/india/bp_context_overrides.yaml`
- Create: `market-configs/australia/bp_context_overrides.yaml`

- [ ] **Step 1: Create shared BP context thresholds**

```yaml
# market-configs/shared/bp_context_thresholds.yaml
# BP context classification thresholds for masked HTN / white-coat detection.
# Sources: ESH 2023, ISH 2020, AHA/ACC 2017

# Classification thresholds (mmHg)
thresholds:
  clinic:
    sbp_elevated: 140
    dbp_elevated: 90
    sbp_elevated_dm: 130
    dbp_elevated_dm: 80
  home:
    sbp_elevated: 135
    dbp_elevated: 85
  abpm_daytime:
    sbp_elevated: 135
    dbp_elevated: 85
  abpm_nighttime:
    sbp_elevated: 120
    dbp_elevated: 70
  abpm_24hr:
    sbp_elevated: 130
    dbp_elevated: 80

# Minimum data requirements for valid classification
data_requirements:
  clinic:
    min_readings: 2
    max_age_days: 90
    separate_visits_preferred: true
  home:
    min_readings: 12
    min_days: 4
    max_age_days: 14
    discard_first_day: true
    morning_evening_preferred: true

# White-coat effect magnitude thresholds
white_coat_effect:
  clinically_significant: 15
  severe: 30

# Cross-domain risk amplification
amplification:
  diabetes:
    masked_htn_risk_multiplier: 3.2
    urgency_override: "IMMEDIATE"
  ckd:
    masked_htn_egfr_decline_per_10mmhg: 15
    urgency_override: "URGENT"
  morning_surge:
    compound_threshold: 20
    urgency_override: "IMMEDIATE"

# WCH progression monitoring
wch_progression:
  recheck_interval_months: 6
  progression_rate_pct_per_year: 3

# Selection bias detection
selection_bias:
  engagement_phenotypes_at_risk:
    - "MEASUREMENT_AVOIDANT"
    - "CRISIS_ONLY_MEASURER"
  min_home_readings_for_confidence: 20
  flag_if_readings_below: 12

# Source classification
source_mapping:
  clinic: ["CLINIC", "OFFICE", "HOSPITAL"]
  home: ["HOME_CUFF", "HOME_WRIST", "PATIENT_REPORTED"]
  community: ["COMMUNITY_HEALTH_WORKER", "PHARMACY"]
  abpm: ["ABPM_24HR"]
```

- [ ] **Step 2: Create India overrides**

```yaml
# market-configs/india/bp_context_overrides.yaml
# Sources: ISH 2020 (India endorsed), RSSDI 2023

thresholds_override:
  clinic:
    sbp_elevated_dm: 130
    dbp_elevated_dm: 80

white_coat_effect_override:
  clinically_significant: 20
  rationale: "Indian clinic environments generate higher anxiety responses (crowded OPDs, long waits)"

# Device validation concern
device_validation:
  warn_on_wrist_cuff: true
  warn_on_unvalidated: true
  recommended_devices: ["Omron HEM-7120", "Omron HEM-7130", "Dr. Morepen BP-02"]
  message: "Ensure upper-arm oscillometric cuff is validated per ESH/ISH protocol"

# Salt-sensitivity correlation
sodium_correlation:
  enabled: true
  sodium_threshold_mg: 2000
  bp_elevation_expected_per_1000mg: 5
```

- [ ] **Step 3: Create Australia overrides**

```yaml
# market-configs/australia/bp_context_overrides.yaml
# Sources: Heart Foundation 2022, RACGP 2024

abpm_recommendation:
  mbs_item_code: "11607"
  recommend_when: ["MASKED_HTN_PROBABLE", "WHITE_COAT_HTN_PROBABLE"]
  message: "Confirm with 24-hour ABPM (Medicare Item 11607) for definitive classification"

indigenous_overrides:
  source_mapping_override:
    community_health_worker: "HOME"
  screening_interval_months: 3

heart_foundation_alignment:
  recommend_hbpm_for_treated: true
  classification_thresholds: "ISH_2020"
```

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add market-configs/shared/bp_context_thresholds.yaml
git add market-configs/india/bp_context_overrides.yaml
git add market-configs/australia/bp_context_overrides.yaml
git commit -m "feat: BP context market configs — shared thresholds + India/Australia overrides"
```

---

## Task 3: BP Context Classifier Tests (KB-26)

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier_test.go`

Note: Tests use standard library `testing` (no testify — not in go.mod).

- [ ] **Step 1: Write the full test file**

```go
// kb-26-metabolic-digital-twin/internal/services/bp_context_classifier_test.go
package services

import (
	"testing"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// === CORE PHENOTYPE CLASSIFICATION ===

func TestClassifyBPContext_SustainedHTN(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 155, DBP: 95, Source: "CLINIC"},
			{SBP: 150, DBP: 92, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 142, 88),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeSustainedHTN {
		t.Errorf("expected SUSTAINED_HTN, got %s", result.Phenotype)
	}
	if !result.ClinicAboveThreshold {
		t.Error("expected ClinicAboveThreshold = true")
	}
	if !result.HomeAboveThreshold {
		t.Error("expected HomeAboveThreshold = true")
	}
}

func TestClassifyBPContext_MaskedHTN(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 128, DBP: 78, Source: "CLINIC"},
			{SBP: 132, DBP: 82, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 148, 92),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}
	if result.ClinicAboveThreshold {
		t.Error("expected ClinicAboveThreshold = false")
	}
	if !result.HomeAboveThreshold {
		t.Error("expected HomeAboveThreshold = true")
	}
	if result.ClinicHomeGapSBP >= 0 {
		t.Errorf("expected negative gap (home > clinic), got %.1f", result.ClinicHomeGapSBP)
	}
}

func TestClassifyBPContext_WhiteCoatHTN(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 158, DBP: 96, Source: "CLINIC"},
			{SBP: 152, DBP: 94, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 125, 78),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeWhiteCoatHTN {
		t.Errorf("expected WHITE_COAT_HTN, got %s", result.Phenotype)
	}
	if result.WhiteCoatEffect < 15 {
		t.Errorf("expected WCE >= 15, got %.1f", result.WhiteCoatEffect)
	}
}

func TestClassifyBPContext_SustainedNormotension(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 122, DBP: 76, Source: "CLINIC"},
			{SBP: 118, DBP: 74, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 120, 75),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeSustainedNormotension {
		t.Errorf("expected SUSTAINED_NORMOTENSION, got %s", result.Phenotype)
	}
}

// === TREATED PATIENT PHENOTYPES ===

func TestClassifyBPContext_MaskedUncontrolled(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 132, DBP: 80, Source: "CLINIC"},
			{SBP: 128, DBP: 78, Source: "CLINIC"},
		},
		HomeReadings:        generateHomeReadings(14, 145, 90),
		OnAntihypertensives: true,
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeMaskedUncontrolled {
		t.Errorf("expected MASKED_UNCONTROLLED, got %s", result.Phenotype)
	}
}

func TestClassifyBPContext_WhiteCoatUncontrolled(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 148, DBP: 92, Source: "CLINIC"},
			{SBP: 145, DBP: 90, Source: "CLINIC"},
		},
		HomeReadings:        generateHomeReadings(14, 128, 80),
		OnAntihypertensives: true,
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeWhiteCoatUncontrolled {
		t.Errorf("expected WHITE_COAT_UNCONTROLLED, got %s", result.Phenotype)
	}
}

// === CROSS-DOMAIN AMPLIFICATION ===

func TestClassifyBPContext_MaskedHTN_DiabeticAmplification(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 125, DBP: 76, Source: "CLINIC"},
			{SBP: 128, DBP: 78, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 142, 88),
		IsDiabetic:   true,
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}
	if !result.DiabetesAmplification {
		t.Error("expected DiabetesAmplification = true")
	}
}

func TestClassifyBPContext_MaskedHTN_CKDAmplification(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 130, DBP: 80, Source: "CLINIC"},
			{SBP: 132, DBP: 78, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 144, 90),
		HasCKD:       true,
		EGFR:         42,
	}

	result := ClassifyBPContext(input)
	if !result.CKDAmplification {
		t.Error("expected CKDAmplification = true")
	}
}

func TestClassifyBPContext_MaskedHTN_MorningSurgeCompound(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 128, DBP: 78, Source: "CLINIC"},
			{SBP: 130, DBP: 80, Source: "CLINIC"},
		},
		HomeReadings:      generateHomeReadings(14, 140, 88),
		MorningSurge7dAvg: 28,
	}

	result := ClassifyBPContext(input)
	if !result.MorningSurgeCompound {
		t.Error("expected MorningSurgeCompound = true")
	}
}

// === SELECTION BIAS DETECTION ===

func TestClassifyBPContext_SelectionBias_MeasurementAvoidant(t *testing.T) {
	// 13 readings (just above minimum) from avoidant patient
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 128, DBP: 78, Source: "CLINIC"},
			{SBP: 130, DBP: 80, Source: "CLINIC"},
		},
		HomeReadings:        generateHomeReadings(13, 155, 95),
		EngagementPhenotype: "MEASUREMENT_AVOIDANT",
	}

	result := ClassifyBPContext(input)
	if !result.SelectionBiasRisk {
		t.Error("expected SelectionBiasRisk = true")
	}
	if result.Confidence != "LOW" {
		t.Errorf("expected LOW confidence, got %s", result.Confidence)
	}
}

// === INSUFFICIENT DATA ===

func TestClassifyBPContext_InsufficientHome(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 145, DBP: 92, Source: "CLINIC"},
			{SBP: 148, DBP: 90, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(2, 138, 86),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeInsufficientData {
		t.Errorf("expected INSUFFICIENT_DATA, got %s", result.Phenotype)
	}
	if result.SufficientHome {
		t.Error("expected SufficientHome = false")
	}
}

func TestClassifyBPContext_InsufficientClinic(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{},
		HomeReadings:   generateHomeReadings(14, 138, 86),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeInsufficientData {
		t.Errorf("expected INSUFFICIENT_DATA, got %s", result.Phenotype)
	}
	if result.SufficientClinic {
		t.Error("expected SufficientClinic = false")
	}
}

// === MEDICATION TIMING HYPOTHESIS ===

func TestClassifyBPContext_MedicationTimingHypothesis(t *testing.T) {
	now := time.Now()
	var allHome []BPReading
	for i := 0; i < 7; i++ {
		allHome = append(allHome, BPReading{
			SBP: 148, DBP: 92, Source: "HOME_CUFF", TimeContext: "MORNING",
			Timestamp: now.Add(time.Duration(-i*24) * time.Hour),
		})
		allHome = append(allHome, BPReading{
			SBP: 125, DBP: 78, Source: "HOME_CUFF", TimeContext: "EVENING",
			Timestamp: now.Add(time.Duration(-i*24+12) * time.Hour),
		})
	}

	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 128, DBP: 80, Source: "CLINIC"},
			{SBP: 130, DBP: 78, Source: "CLINIC"},
		},
		HomeReadings:        allHome,
		OnAntihypertensives: true,
	}

	result := ClassifyBPContext(input)
	if result.MedicationTimingHypothesis == "" {
		t.Error("expected non-empty MedicationTimingHypothesis")
	}
}

// === RAJESH KUMAR INTEGRATION ===

func TestClassifyBPContext_RajeshKumar(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 170, DBP: 104, Source: "CLINIC"},
			{SBP: 168, DBP: 100, Source: "CLINIC"},
		},
		HomeReadings:        generateHomeReadings(14, 158, 96),
		IsDiabetic:          true,
		HasCKD:              true,
		EGFR:                42,
		OnAntihypertensives: true,
		MorningSurge7dAvg:   28,
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeSustainedHTN {
		t.Errorf("expected SUSTAINED_HTN, got %s", result.Phenotype)
	}
	if !result.DiabetesAmplification {
		t.Error("expected DiabetesAmplification = true")
	}
	if !result.CKDAmplification {
		t.Error("expected CKDAmplification = true")
	}
	if !result.MorningSurgeCompound {
		t.Error("expected MorningSurgeCompound = true")
	}
}

// === HELPER ===

func generateHomeReadings(count int, avgSBP, avgDBP float64) []BPReading {
	readings := make([]BPReading, count)
	now := time.Now()
	for i := 0; i < count; i++ {
		readings[i] = BPReading{
			SBP:       avgSBP + float64(i%3-1)*3,
			DBP:       avgDBP + float64(i%3-1)*2,
			Source:    "HOME_CUFF",
			Timestamp: now.Add(time.Duration(-i*12) * time.Hour),
		}
	}
	return readings
}
```

- [ ] **Step 2: Verify tests fail (classifier not yet implemented)**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestClassifyBPContext" -v 2>&1 | head -5`
Expected: compilation error — `ClassifyBPContext` undefined

- [ ] **Step 3: Commit failing tests**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/bp_context_classifier_test.go
git commit -m "test(kb26): failing BP context classifier tests — 14 cases for 6 phenotypes"
```

---

## Task 4: BP Context Classifier Implementation (KB-26)

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_classifier.go`

- [ ] **Step 1: Implement the full classifier**

```go
// kb-26-metabolic-digital-twin/internal/services/bp_context_classifier.go
package services

import (
	"fmt"
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// Default thresholds — overridden by market config at runtime.
const (
	defaultClinicSBP          = 140.0
	defaultClinicDBP          = 90.0
	defaultClinicSBP_DM       = 130.0
	defaultClinicDBP_DM       = 80.0
	defaultHomeSBP            = 135.0
	defaultHomeDBP            = 85.0
	minClinicReadings         = 2
	minHomeReadings           = 12
	minHomeDays               = 4
	significantWCE            = 15.0
	morningSurgeCompoundLimit = 20.0
	minHomeForConfidence      = 20
)

// BPReading represents a single blood pressure measurement.
type BPReading struct {
	SBP         float64   `json:"sbp"`
	DBP         float64   `json:"dbp"`
	Source      string    `json:"source"`
	TimeContext string    `json:"time_context,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// BPContextInput is the input to the clinic-home BP classifier.
type BPContextInput struct {
	ClinicReadings      []BPReading
	HomeReadings        []BPReading
	OnAntihypertensives bool
	IsDiabetic          bool
	HasCKD              bool
	EGFR                float64
	EngagementPhenotype string
	MorningSurge7dAvg   float64
}

// ClassifyBPContext performs the clinic-home BP discordance analysis.
func ClassifyBPContext(input BPContextInput) models.BPContextClassification {
	result := models.BPContextClassification{
		ComputedAt:          time.Now(),
		OnAntihypertensives: input.OnAntihypertensives,
		IsDiabetic:          input.IsDiabetic,
		HasCKD:              input.HasCKD,
	}

	// Set thresholds (diabetic patients use stricter clinic thresholds per ISH 2020)
	clinicSBPThresh := defaultClinicSBP
	clinicDBPThresh := defaultClinicDBP
	if input.IsDiabetic {
		clinicSBPThresh = defaultClinicSBP_DM
		clinicDBPThresh = defaultClinicDBP_DM
	}
	result.ClinicSBPThreshold = clinicSBPThresh
	result.ClinicDBPThreshold = clinicDBPThresh
	result.HomeSBPThreshold = defaultHomeSBP
	result.HomeDBPThreshold = defaultHomeDBP

	// Check data sufficiency
	result.SufficientClinic = len(input.ClinicReadings) >= minClinicReadings
	homeDistinctDays := countDistinctDays(input.HomeReadings)
	result.SufficientHome = len(input.HomeReadings) >= minHomeReadings && homeDistinctDays >= minHomeDays
	result.ClinicReadingCount = len(input.ClinicReadings)
	result.HomeReadingCount = len(input.HomeReadings)
	result.HomeDaysWithData = homeDistinctDays

	if !result.SufficientClinic || !result.SufficientHome {
		result.Phenotype = models.PhenotypeInsufficientData
		result.Confidence = "LOW"
		if len(input.ClinicReadings) > 0 {
			result.ClinicSBPMean, result.ClinicDBPMean = computeBPMeans(input.ClinicReadings)
		}
		if len(input.HomeReadings) > 0 {
			result.HomeSBPMean, result.HomeDBPMean = computeBPMeans(input.HomeReadings)
		}
		return result
	}

	// Compute means
	result.ClinicSBPMean, result.ClinicDBPMean = computeBPMeans(input.ClinicReadings)
	result.HomeSBPMean, result.HomeDBPMean = computeBPMeans(input.HomeReadings)

	// Compute discordance: positive = clinic higher, negative = home higher
	result.ClinicHomeGapSBP = math.Round((result.ClinicSBPMean-result.HomeSBPMean)*10) / 10
	result.ClinicHomeGapDBP = math.Round((result.ClinicDBPMean-result.HomeDBPMean)*10) / 10
	result.WhiteCoatEffect = math.Max(0, result.ClinicHomeGapSBP)

	// Classify against thresholds
	result.ClinicAboveThreshold = result.ClinicSBPMean >= clinicSBPThresh || result.ClinicDBPMean >= clinicDBPThresh
	result.HomeAboveThreshold = result.HomeSBPMean >= defaultHomeSBP || result.HomeDBPMean >= defaultHomeDBP

	switch {
	case result.ClinicAboveThreshold && result.HomeAboveThreshold:
		result.Phenotype = models.PhenotypeSustainedHTN
	case result.ClinicAboveThreshold && !result.HomeAboveThreshold:
		if input.OnAntihypertensives {
			result.Phenotype = models.PhenotypeWhiteCoatUncontrolled
		} else {
			result.Phenotype = models.PhenotypeWhiteCoatHTN
		}
	case !result.ClinicAboveThreshold && result.HomeAboveThreshold:
		if input.OnAntihypertensives {
			result.Phenotype = models.PhenotypeMaskedUncontrolled
		} else {
			result.Phenotype = models.PhenotypeMaskedHTN
		}
	default:
		result.Phenotype = models.PhenotypeSustainedNormotension
	}

	// Cross-domain amplification — applies to masked AND sustained HTN
	isMasked := result.Phenotype == models.PhenotypeMaskedHTN ||
		result.Phenotype == models.PhenotypeMaskedUncontrolled
	isElevated := isMasked || result.Phenotype == models.PhenotypeSustainedHTN

	if isElevated && input.IsDiabetic {
		result.DiabetesAmplification = true
	}
	if isElevated && input.HasCKD {
		result.CKDAmplification = true
	}
	if isElevated && input.MorningSurge7dAvg > morningSurgeCompoundLimit {
		result.MorningSurgeCompound = true
	}

	// Selection bias detection
	biasRiskPhenotypes := map[string]bool{
		"MEASUREMENT_AVOIDANT": true,
		"CRISIS_ONLY_MEASURER": true,
	}
	if biasRiskPhenotypes[input.EngagementPhenotype] && len(input.HomeReadings) < minHomeForConfidence {
		result.SelectionBiasRisk = true
	}

	// Confidence assessment
	result.Confidence = assessBPConfidence(result, input)

	// Medication timing hypothesis
	if input.OnAntihypertensives && isElevated {
		result.MedicationTimingHypothesis = detectMedicationTimingPattern(input.HomeReadings)
	}

	result.EngagementPhenotype = input.EngagementPhenotype
	return result
}

func computeBPMeans(readings []BPReading) (float64, float64) {
	var sumSBP, sumDBP float64
	for _, r := range readings {
		sumSBP += r.SBP
		sumDBP += r.DBP
	}
	n := float64(len(readings))
	return math.Round(sumSBP/n*10) / 10, math.Round(sumDBP/n*10) / 10
}

func countDistinctDays(readings []BPReading) int {
	days := make(map[string]bool)
	for _, r := range readings {
		days[r.Timestamp.Format("2006-01-02")] = true
	}
	return len(days)
}

func assessBPConfidence(result models.BPContextClassification, input BPContextInput) string {
	if result.SelectionBiasRisk {
		return "LOW"
	}
	if len(input.ClinicReadings) >= 3 && len(input.HomeReadings) >= minHomeForConfidence &&
		countDistinctDays(input.HomeReadings) >= 7 {
		return "HIGH"
	}
	if result.SufficientClinic && result.SufficientHome {
		return "MODERATE"
	}
	return "LOW"
}

func detectMedicationTimingPattern(homeReadings []BPReading) string {
	var morningSBPs, eveningSBPs []float64
	for _, r := range homeReadings {
		switch r.TimeContext {
		case "MORNING":
			morningSBPs = append(morningSBPs, r.SBP)
		case "EVENING":
			eveningSBPs = append(eveningSBPs, r.SBP)
		}
	}

	if len(morningSBPs) < 3 || len(eveningSBPs) < 3 {
		return ""
	}

	morningMean := meanFloat(morningSBPs)
	eveningMean := meanFloat(eveningSBPs)

	if morningMean-eveningMean > 15 {
		return fmt.Sprintf("Morning BP (mean %.0f) significantly higher than evening (mean %.0f) — "+
			"suggests medication wearing off overnight. Consider evening dosing or longer-acting formulation.",
			morningMean, eveningMean)
	}
	if eveningMean-morningMean > 15 {
		return fmt.Sprintf("Evening BP (mean %.0f) higher than morning (mean %.0f) — "+
			"investigate afternoon/evening BP triggers: dietary sodium, stress, medication timing.",
			eveningMean, morningMean)
	}
	return ""
}

func meanFloat(vals []float64) float64 {
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}
```

- [ ] **Step 2: Run all classifier tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestClassifyBPContext" -v`
Expected: All 14 tests PASS

- [ ] **Step 3: Commit passing classifier**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/bp_context_classifier.go
git commit -m "feat(kb26): BP context classifier — 6 phenotypes with cross-domain amplification

Classifies SUSTAINED_HTN, WHITE_COAT_HTN, MASKED_HTN, SUSTAINED_NORMOTENSION,
MASKED_UNCONTROLLED, WHITE_COAT_UNCONTROLLED from clinic-home BP comparison.
Cross-domain: DM (3.2x TOD), CKD (15%/10mmHg eGFR decline), morning surge
compound. Selection bias detection. Medication timing hypothesis."
```

---

## Task 5: Database Migration (KB-26)

**Files:**
- Create: `kb-26-metabolic-digital-twin/migrations/006_bp_context.sql`

- [ ] **Step 1: Create the migration file**

```sql
-- kb-26-metabolic-digital-twin/migrations/006_bp_context.sql
-- BP context classification history for phenotype progression tracking.

CREATE TABLE IF NOT EXISTS bp_context_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id VARCHAR(100) NOT NULL,
    snapshot_date DATE NOT NULL,
    phenotype VARCHAR(30) NOT NULL,
    clinic_sbp_mean DECIMAL(5,1),
    home_sbp_mean DECIMAL(5,1),
    gap_sbp DECIMAL(5,1),
    confidence VARCHAR(10),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(patient_id, snapshot_date)
);

CREATE INDEX idx_bpc_patient ON bp_context_history(patient_id, snapshot_date DESC);
```

- [ ] **Step 2: Commit migration**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/migrations/006_bp_context.sql
git commit -m "feat(kb26): migration 006 — bp_context_history table for phenotype tracking"
```

---

## Task 6: BP Context Model (KB-23 Local Mirror)

**Files:**
- Create: `kb-23-decision-cards/internal/models/bp_context.go`

KB-23 owns its own models. This mirrors the KB-26 struct fields that card generation needs.

- [ ] **Step 1: Create the KB-23 local model**

```go
// kb-23-decision-cards/internal/models/bp_context.go
package models

// BPContextPhenotype classifies the patient's clinic-home BP relationship.
type BPContextPhenotype string

const (
	PhenotypeSustainedHTN          BPContextPhenotype = "SUSTAINED_HTN"
	PhenotypeWhiteCoatHTN          BPContextPhenotype = "WHITE_COAT_HTN"
	PhenotypeMaskedHTN             BPContextPhenotype = "MASKED_HTN"
	PhenotypeSustainedNormotension BPContextPhenotype = "SUSTAINED_NORMOTENSION"
	PhenotypeMaskedUncontrolled    BPContextPhenotype = "MASKED_UNCONTROLLED"
	PhenotypeWhiteCoatUncontrolled BPContextPhenotype = "WHITE_COAT_UNCONTROLLED"
	PhenotypeInsufficientData      BPContextPhenotype = "INSUFFICIENT_DATA"
)

// BPContextClassification is the clinic-home discordance analysis result,
// consumed by card generation and four-pillar evaluation.
type BPContextClassification struct {
	PatientID  string             `json:"patient_id"`
	Phenotype  BPContextPhenotype `json:"phenotype"`

	// BP summaries
	ClinicSBPMean      float64 `json:"clinic_sbp_mean"`
	ClinicDBPMean      float64 `json:"clinic_dbp_mean"`
	HomeSBPMean        float64 `json:"home_sbp_mean"`
	HomeDBPMean        float64 `json:"home_dbp_mean"`
	HomeReadingCount   int     `json:"home_reading_count"`

	// Discordance
	ClinicHomeGapSBP float64 `json:"clinic_home_gap_sbp"`
	WhiteCoatEffect  float64 `json:"white_coat_effect_mmhg"`
	Confidence       string  `json:"confidence"`

	// Cross-domain amplification
	IsDiabetic            bool `json:"is_diabetic"`
	DiabetesAmplification bool `json:"diabetes_amplification"`
	HasCKD                bool `json:"has_ckd"`
	CKDAmplification      bool `json:"ckd_amplification"`
	SelectionBiasRisk     bool `json:"selection_bias_risk"`
	MorningSurgeCompound  bool `json:"morning_surge_compound"`

	// Treatment context
	OnAntihypertensives        bool   `json:"on_antihypertensives"`
	EngagementPhenotype        string `json:"engagement_phenotype,omitempty"`
	MedicationTimingHypothesis string `json:"medication_timing_hypothesis,omitempty"`
}
```

- [ ] **Step 2: Verify file compiles**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./internal/models/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-23-decision-cards/internal/models/bp_context.go
git commit -m "feat(kb23): BP context model — local mirror for card generation"
```

---

## Task 7: Masked HTN Card Tests (KB-23)

**Files:**
- Create: `kb-23-decision-cards/internal/services/masked_htn_cards_test.go`

- [ ] **Step 1: Write the full card test file**

```go
// kb-23-decision-cards/internal/services/masked_htn_cards_test.go
package services

import (
	"strings"
	"testing"

	"kb-23-decision-cards/internal/models"
)

func TestMaskedHTNCards_MaskedHTN_Diabetic_Immediate(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:             models.PhenotypeMaskedHTN,
		ClinicSBPMean:         128,
		ClinicDBPMean:         78,
		HomeSBPMean:           148,
		HomeDBPMean:           92,
		ClinicHomeGapSBP:      -20,
		DiabetesAmplification: true,
		IsDiabetic:            true,
		Confidence:            "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "MASKED_HYPERTENSION" {
			found = true
			if c.Urgency != "IMMEDIATE" {
				t.Errorf("expected IMMEDIATE urgency for DM amplification, got %s", c.Urgency)
			}
			if !strings.Contains(c.Rationale, "3.2") {
				t.Error("expected rationale to mention 3.2x risk multiplier")
			}
		}
	}
	if !found {
		t.Error("expected MASKED_HYPERTENSION card")
	}
}

func TestMaskedHTNCards_WhiteCoatHTN_AvoidOvertreatment(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:       models.PhenotypeWhiteCoatHTN,
		ClinicSBPMean:   155,
		ClinicDBPMean:   96,
		HomeSBPMean:     125,
		HomeDBPMean:     78,
		ClinicHomeGapSBP: 30,
		WhiteCoatEffect: 30,
		Confidence:      "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "WHITE_COAT_HYPERTENSION" {
			found = true
			if c.Urgency != "ROUTINE" {
				t.Errorf("expected ROUTINE urgency, got %s", c.Urgency)
			}
			if !strings.Contains(c.Rationale, "overtreatment") {
				t.Error("expected rationale to mention overtreatment")
			}
		}
	}
	if !found {
		t.Error("expected WHITE_COAT_HYPERTENSION card")
	}
}

func TestMaskedHTNCards_MUCH_TreatedPatient(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:           models.PhenotypeMaskedUncontrolled,
		ClinicSBPMean:       130,
		ClinicDBPMean:       80,
		HomeSBPMean:         150,
		HomeDBPMean:         92,
		OnAntihypertensives: true,
		CKDAmplification:    true,
		Confidence:          "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "MASKED_UNCONTROLLED" {
			found = true
			if c.Urgency != "URGENT" {
				t.Errorf("expected URGENT urgency, got %s", c.Urgency)
			}
			if !strings.Contains(c.Rationale, "controlled in clinic but not at home") {
				t.Error("expected rationale to mention 'controlled in clinic but not at home'")
			}
		}
	}
	if !found {
		t.Error("expected MASKED_UNCONTROLLED card")
	}
}

func TestMaskedHTNCards_CompoundRisk_MH_MorningSurge(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:            models.PhenotypeMaskedHTN,
		ClinicSBPMean:        128,
		HomeSBPMean:          142,
		MorningSurgeCompound: true,
		Confidence:           "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "MASKED_HTN_MORNING_SURGE_COMPOUND" {
			found = true
			if c.Urgency != "IMMEDIATE" {
				t.Errorf("expected IMMEDIATE urgency, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected MASKED_HTN_MORNING_SURGE_COMPOUND card")
	}
}

func TestMaskedHTNCards_SelectionBias_FlagsUncertainty(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:           models.PhenotypeMaskedHTN,
		SelectionBiasRisk:   true,
		Confidence:          "LOW",
		HomeReadingCount:    8,
		EngagementPhenotype: "MEASUREMENT_AVOIDANT",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "SELECTION_BIAS_WARNING" {
			found = true
			if !strings.Contains(strings.ToLower(c.Rationale), "avoidant") ||
				!strings.Contains(strings.ToLower(c.Rationale), "measurement") {
				t.Error("expected rationale to reference measurement avoidant behaviour")
			}
		}
	}
	if !found {
		t.Error("expected SELECTION_BIAS_WARNING card")
	}
}

func TestMaskedHTNCards_MedicationTiming(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:                  models.PhenotypeMaskedUncontrolled,
		OnAntihypertensives:        true,
		MedicationTimingHypothesis: "Morning BP significantly higher than evening — consider evening dosing",
		Confidence:                 "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "MEDICATION_TIMING" {
			found = true
			if !strings.Contains(c.Rationale, "evening dosing") {
				t.Error("expected rationale to mention evening dosing")
			}
		}
	}
	if !found {
		t.Error("expected MEDICATION_TIMING card")
	}
}

func TestMaskedHTNCards_Normotensive_NoUrgentCards(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:  models.PhenotypeSustainedNormotension,
		Confidence: "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	for _, c := range cards {
		if c.Urgency == "IMMEDIATE" || c.Urgency == "URGENT" {
			t.Errorf("normotensive patient should not receive urgent cards, got %s (%s)", c.Urgency, c.CardType)
		}
	}
}
```

- [ ] **Step 2: Verify tests fail (cards not yet implemented)**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestMaskedHTNCards" -v 2>&1 | head -5`
Expected: compilation error — `EvaluateMaskedHTNCards` undefined

- [ ] **Step 3: Commit failing tests**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-23-decision-cards/internal/services/masked_htn_cards_test.go
git commit -m "test(kb23): failing masked HTN card tests — 7 cases for 8 card types"
```

---

## Task 8: Masked HTN Card Implementation (KB-23)

**Files:**
- Create: `kb-23-decision-cards/internal/services/masked_htn_cards.go`

- [ ] **Step 1: Implement the card evaluator**

```go
// kb-23-decision-cards/internal/services/masked_htn_cards.go
package services

import (
	"fmt"

	"kb-23-decision-cards/internal/models"
)

// MaskedHTNCard is a decision card produced by BP context analysis.
type MaskedHTNCard struct {
	CardType  string   `json:"card_type"`
	Urgency   string   `json:"urgency"`
	Title     string   `json:"title"`
	Rationale string   `json:"rationale"`
	Actions   []string `json:"actions"`
}

// EvaluateMaskedHTNCards generates decision cards from a BP context classification.
func EvaluateMaskedHTNCards(classification *models.BPContextClassification) []MaskedHTNCard {
	if classification == nil {
		return nil
	}

	var cards []MaskedHTNCard

	switch classification.Phenotype {
	case models.PhenotypeMaskedHTN, models.PhenotypeMaskedUncontrolled:
		cards = append(cards, buildMaskedHTNCards(classification)...)

	case models.PhenotypeWhiteCoatHTN, models.PhenotypeWhiteCoatUncontrolled:
		cards = append(cards, buildWhiteCoatCards(classification)...)

	case models.PhenotypeSustainedNormotension:
		return nil

	case models.PhenotypeSustainedHTN:
		if classification.MorningSurgeCompound {
			cards = append(cards, MaskedHTNCard{
				CardType: "SUSTAINED_HTN_MORNING_SURGE",
				Urgency:  "URGENT",
				Title:    "Sustained Hypertension with Abnormal Morning Surge",
				Rationale: fmt.Sprintf("Both clinic (%.0f) and home (%.0f) BP elevated with morning surge compound risk. "+
					"Morning surge >20 mmHg in sustained HTN significantly increases stroke risk (Kario 2019).",
					classification.ClinicSBPMean, classification.HomeSBPMean),
				Actions: []string{
					"Consider bedtime dosing of antihypertensive",
					"Evaluate for sleep apnea (strong association with morning surge)",
					"Add long-acting ARB or CCB for 24-hour coverage",
				},
			})
		}

	case models.PhenotypeInsufficientData:
		return nil
	}

	// Selection bias warning (appended to any classification)
	if classification.SelectionBiasRisk {
		cards = append(cards, MaskedHTNCard{
			CardType: "SELECTION_BIAS_WARNING",
			Urgency:  "ROUTINE",
			Title:    "Home BP Data May Reflect Measurement Selection Bias",
			Rationale: fmt.Sprintf("Patient phenotype: %s with only %d home readings. "+
				"Patients who measure only when symptomatic (MEASUREMENT_AVOIDANT) produce inflated home BP averages. "+
				"Classification confidence is LOW — increase measurement frequency before acting on masked HTN diagnosis.",
				classification.EngagementPhenotype, classification.HomeReadingCount),
			Actions: []string{
				"Encourage regular measurement schedule (morning + evening daily)",
				"Consider 24-hour ABPM for definitive classification",
				"Address measurement avoidance — may indicate health anxiety",
			},
		})
	}

	// Medication timing hypothesis (appended if present)
	if classification.MedicationTimingHypothesis != "" {
		cards = append(cards, MaskedHTNCard{
			CardType:  "MEDICATION_TIMING",
			Urgency:   "ROUTINE",
			Title:     "Medication Timing May Explain BP Pattern",
			Rationale: classification.MedicationTimingHypothesis,
			Actions: []string{
				"Review current medication timing with patient",
				"Consider chronotherapy: evening dosing for morning surge",
				"If already evening-dosed, consider longer-acting formulation",
			},
		})
	}

	return cards
}

func buildMaskedHTNCards(c *models.BPContextClassification) []MaskedHTNCard {
	var cards []MaskedHTNCard

	urgency := "URGENT"
	title := "Masked Hypertension Detected"
	rationale := fmt.Sprintf(
		"Clinic BP appears normal (%.0f/%.0f) but home BP is elevated (%.0f/%.0f). "+
			"Clinic-home gap: %.0f mmHg. ",
		c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean, c.HomeDBPMean, c.ClinicHomeGapSBP)

	cardType := "MASKED_HYPERTENSION"
	if c.Phenotype == models.PhenotypeMaskedUncontrolled {
		cardType = "MASKED_UNCONTROLLED"
		title = "Masked Uncontrolled Hypertension — Therapy Appears Effective But Isn't"
		rationale += "This patient's BP appears controlled in clinic but not at home — " +
			"current therapy is insufficient despite clinic reassurance. "
	}

	if c.DiabetesAmplification {
		urgency = "IMMEDIATE"
		rationale += fmt.Sprintf("DIABETES AMPLIFICATION: Masked HTN in diabetic patients carries "+
			"3.2x higher target organ damage risk (Leitao 2015). ")
	}

	if c.CKDAmplification {
		if urgency != "IMMEDIATE" {
			urgency = "URGENT"
		}
		rationale += fmt.Sprintf("CKD AMPLIFICATION: Masked HTN accelerates eGFR decline "+
			"~15%% faster per 10 mmHg masked SBP elevation (Agarwal 2017). ")
	}

	actions := []string{
		"Initiate or intensify antihypertensive therapy based on HOME BP, not clinic BP",
		"Target home BP <135/85 (or <130/80 if diabetic)",
	}
	if c.Phenotype == models.PhenotypeMaskedUncontrolled {
		actions = append(actions,
			"Current therapy creates false confidence — clinic visits do not reflect true BP burden",
			"Consider adding a second/third antihypertensive class",
		)
	}
	if c.DiabetesAmplification || c.CKDAmplification {
		actions = append(actions,
			"Prefer ACEi/ARB (renoprotective) + SGLT2i (cardiorenal benefit)",
			"Screen for target organ damage: echocardiogram, urine ACR, retinal exam",
		)
	}

	cards = append(cards, MaskedHTNCard{
		CardType:  cardType,
		Urgency:   urgency,
		Title:     title,
		Rationale: rationale,
		Actions:   actions,
	})

	// Compound risk card for morning surge
	if c.MorningSurgeCompound {
		cards = append(cards, MaskedHTNCard{
			CardType: "MASKED_HTN_MORNING_SURGE_COMPOUND",
			Urgency:  "IMMEDIATE",
			Title:    "Masked Hypertension + Abnormal Morning Surge — Compound Stroke Risk",
			Rationale: "Masked hypertension combined with morning surge >20 mmHg creates multiplicatively " +
				"elevated stroke risk (Kario 2019, JACC). The patient's BP is normal in the clinic " +
				"but dangerously elevated each morning before the day's first dose.",
			Actions: []string{
				"Urgent: add or adjust bedtime antihypertensive",
				"Evaluate for obstructive sleep apnea (strongest modifiable morning surge cause)",
				"Consider 24-hour ABPM for full circadian profiling",
				"Schedule follow-up within 1-2 weeks to assess response",
			},
		})
	}

	return cards
}

func buildWhiteCoatCards(c *models.BPContextClassification) []MaskedHTNCard {
	cardType := "WHITE_COAT_HYPERTENSION"
	title := "White-Coat Hypertension — Consider Avoiding Medication"
	if c.Phenotype == models.PhenotypeWhiteCoatUncontrolled {
		cardType = "WHITE_COAT_UNCONTROLLED"
		title = "White-Coat Effect — Clinic BP Elevated but Home BP Controlled"
	}

	rationale := fmt.Sprintf(
		"Clinic BP elevated (%.0f/%.0f) but home BP normal (%.0f/%.0f). "+
			"White-coat effect: %.0f mmHg. ",
		c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean, c.HomeDBPMean, c.WhiteCoatEffect)

	if c.WhiteCoatEffect >= 30 {
		rationale += "Severe white-coat effect (>=30 mmHg) — investigate clinic anxiety or pain. "
	}

	rationale += "Risk of overtreatment: initiating or intensifying therapy based on clinic BP alone " +
		"may cause unnecessary side effects without cardiovascular benefit. "

	actions := []string{
		"Do NOT intensify antihypertensive therapy based solely on clinic readings",
		"Continue home BP monitoring — recheck in 6 months",
		"Lifestyle modifications remain appropriate (exercise, salt reduction, weight management)",
		"Monitor for progression to sustained HTN (~3% per year — Mancia 2006)",
	}

	if c.Phenotype == models.PhenotypeWhiteCoatUncontrolled {
		actions = append([]string{
			"Consider REDUCING antihypertensive dose if patient symptomatic (dizziness, fatigue)",
		}, actions...)
	}

	return []MaskedHTNCard{{
		CardType:  cardType,
		Urgency:   "ROUTINE",
		Title:     title,
		Rationale: rationale,
		Actions:   actions,
	}}
}
```

- [ ] **Step 2: Run card tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestMaskedHTNCards" -v`
Expected: All 7 tests PASS

- [ ] **Step 3: Commit passing cards**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-23-decision-cards/internal/services/masked_htn_cards.go
git commit -m "feat(kb23): masked HTN card evaluator — 8 card types with urgency logic

MASKED_HYPERTENSION (IMMEDIATE for DM), WHITE_COAT_HTN (ROUTINE),
MASKED_UNCONTROLLED (URGENT), MORNING_SURGE_COMPOUND (IMMEDIATE),
SELECTION_BIAS_WARNING, MEDICATION_TIMING, SUSTAINED_HTN_MORNING_SURGE,
WHITE_COAT_UNCONTROLLED."
```

---

## Task 9: Four-Pillar Integration (KB-23)

**Files:**
- Modify: `kb-23-decision-cards/internal/services/four_pillar_evaluator.go`

- [ ] **Step 1: Add BPContext to FourPillarInput**

In `four_pillar_evaluator.go`, add a new field to `FourPillarInput` after the `InertiaReport` field (line 52):

```go
	InertiaReport   *models.PatientInertiaReport   `json:"inertia_report,omitempty"`
	BPContext       *models.BPContextClassification `json:"bp_context,omitempty"`
```

- [ ] **Step 2: Add masked HTN logic to evaluateMedicationPillar**

In `evaluateMedicationPillar`, add the BP context check after the therapeutic inertia block (after line 157, before the "Guideline adherence check" comment at line 159):

```go
	// Masked hypertension: clinic BP looks fine but home BP is elevated
	if input.BPContext != nil {
		switch input.BPContext.Phenotype {
		case models.PhenotypeMaskedHTN, models.PhenotypeMaskedUncontrolled:
			status := PillarGap
			if input.BPContext.DiabetesAmplification || input.BPContext.MorningSurgeCompound {
				status = PillarUrgentGap
			}
			p.Status = status
			p.Reason = "masked hypertension — home BP elevated despite normal clinic readings"
			p.Actions = []string{
				"treat based on HOME BP targets, not clinic readings",
			}
			if input.BPContext.DiabetesAmplification {
				p.Actions = append(p.Actions, "DM + masked HTN: 3.2x target organ damage risk — immediate action")
			}
			return p

		case models.PhenotypeWhiteCoatHTN, models.PhenotypeWhiteCoatUncontrolled:
			p.Status = PillarGap
			p.Reason = "white-coat effect — do not intensify based on clinic BP alone"
			p.Actions = []string{
				"continue home monitoring; lifestyle intervention appropriate",
			}
			return p
		}
	}
```

- [ ] **Step 3: Run existing four-pillar tests to verify no regression**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run "TestEvaluateFourPillars" -v`
Expected: All existing tests PASS (BPContext is nil in existing tests, so new code path is not hit)

- [ ] **Step 4: Run ALL KB-23 tests for regression check**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -v -count=1`
Expected: All tests PASS

- [ ] **Step 5: Commit four-pillar integration**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-23-decision-cards/internal/services/four_pillar_evaluator.go
git commit -m "feat(kb23): four-pillar integration — masked HTN in medication pillar

MH/MUCH -> GAP (URGENT_GAP if DM or morning surge). WCH/WCUH -> GAP
(avoid overtreatment). Nil-safe: existing callers unaffected."
```

---

## Task 10: Full Regression + Final Commit

- [ ] **Step 1: Run KB-26 full test suite**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./... -v -count=1`
Expected: All tests PASS, no regressions

- [ ] **Step 2: Run KB-23 full test suite**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./... -v -count=1`
Expected: All tests PASS, no regressions

- [ ] **Step 3: Verify all new files are committed**

Run: `cd /Volumes/Vaidshala/cardiofit && git status`
Expected: clean working tree (all files committed in prior tasks)

- [ ] **Step 4: Verify commit history**

Run: `cd /Volumes/Vaidshala/cardiofit && git log --oneline -8`
Expected: 7-8 commits from this implementation (model, configs, tests, classifier, migration, KB-23 model, cards, four-pillar)
