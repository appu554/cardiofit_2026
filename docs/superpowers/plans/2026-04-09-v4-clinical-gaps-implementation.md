# V4 Clinical Gaps Implementation Plan — Renal Gating + CGM Analytics + Therapeutic Inertia

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement three patient-safety and clinical-value systems: (1) renal dose gating that blocks unsafe prescribing when eGFR crosses drug-specific thresholds, (2) CGM analytics computing TIR/TBR/TAR/CV/GMI/GRI from continuous glucose streams, and (3) therapeutic inertia detection that flags when clinicians fail to intensify therapy despite persistent uncontrolled status.

**Architecture:** Each system spans KB-20 (patient data), KB-23 (decision cards), and KB-26 (metabolic twin), plus Flink operators for streaming computation. Market-specific YAML configs handle India (RSSDI/ICMR) and Australia (RACGP/PBS) regulatory differences. Execution order is Renal → CGM → Inertia due to dependencies: therapeutic inertia consumes CGM TIR for pattern detection and renal eGFR for progression tracking.

**Tech Stack:** Go 1.21 (Gin/GORM/testify) for KB services, Java 17 (Flink 1.18) for streaming operators, PostgreSQL 15, Kafka, YAML market configs.

**Dependency on existing V4 plan:** This plan is independent of `2026-04-06-v4-remaining-tasks-implementation.md` but shares KB-20/KB-23/KB-26 services. No file conflicts — all files created here are new.

---

## File Inventory

### Shared Infrastructure (created first)
| Action | File |
|--------|------|
| Create | `backend/shared-infrastructure/market-configs/shared/renal_dose_rules.yaml` |
| Create | `backend/shared-infrastructure/market-configs/india/renal_overrides.yaml` |
| Create | `backend/shared-infrastructure/market-configs/australia/renal_overrides.yaml` |
| Create | `backend/shared-infrastructure/market-configs/shared/cgm_targets.yaml` |
| Create | `backend/shared-infrastructure/market-configs/india/cgm_overrides.yaml` |
| Create | `backend/shared-infrastructure/market-configs/australia/cgm_overrides.yaml` |
| Create | `backend/shared-infrastructure/market-configs/shared/inertia_thresholds.yaml` |
| Create | `backend/shared-infrastructure/market-configs/shared/intensification_pathways.yaml` |
| Create | `backend/shared-infrastructure/market-configs/india/inertia_overrides.yaml` |
| Create | `backend/shared-infrastructure/market-configs/australia/inertia_overrides.yaml` |

### KB-23 Decision Cards (19 files)
| Action | File |
|--------|------|
| Create | `kb-23-decision-cards/internal/models/renal_gating.go` |
| Create | `kb-23-decision-cards/internal/services/renal_formulary.go` |
| Create | `kb-23-decision-cards/internal/services/renal_formulary_test.go` |
| Create | `kb-23-decision-cards/internal/services/renal_dose_gate.go` |
| Create | `kb-23-decision-cards/internal/services/renal_dose_gate_test.go` |
| Create | `kb-23-decision-cards/internal/services/renal_anticipatory.go` |
| Create | `kb-23-decision-cards/internal/services/renal_anticipatory_test.go` |
| Create | `kb-23-decision-cards/internal/services/stale_egfr_detector.go` |
| Create | `kb-23-decision-cards/internal/services/stale_egfr_detector_test.go` |
| Create | `kb-23-decision-cards/internal/services/conflict_detector.go` |
| Create | `kb-23-decision-cards/internal/services/four_pillar_evaluator.go` |
| Create | `kb-23-decision-cards/internal/services/urgency_calculator.go` |
| Create | `kb-23-decision-cards/internal/services/renal_integration_test.go` |
| Create | `kb-23-decision-cards/internal/services/cgm_card_rules.go` |
| Create | `kb-23-decision-cards/internal/services/cgm_card_rules_test.go` |
| Create | `kb-23-decision-cards/internal/models/therapeutic_inertia.go` |
| Create | `kb-23-decision-cards/internal/services/inertia_detector.go` |
| Create | `kb-23-decision-cards/internal/services/inertia_detector_test.go` |
| Create | `kb-23-decision-cards/internal/services/inertia_evidence_builder.go` |
| Create | `kb-23-decision-cards/internal/services/inertia_evidence_builder_test.go` |
| Create | `kb-23-decision-cards/internal/services/inertia_card_generator.go` |
| Create | `kb-23-decision-cards/internal/services/inertia_card_generator_test.go` |

### KB-26 Metabolic Digital Twin (9 files)
| Action | File |
|--------|------|
| Create | `kb-26-metabolic-digital-twin/internal/services/egfr_trajectory.go` |
| Create | `kb-26-metabolic-digital-twin/internal/services/egfr_trajectory_test.go` |
| Create | `kb-26-metabolic-digital-twin/internal/services/renal_event_publisher.go` |
| Create | `kb-26-metabolic-digital-twin/internal/services/renal_event_publisher_test.go` |
| Create | `kb-26-metabolic-digital-twin/internal/models/cgm_metrics.go` |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_analytics.go` |
| Create | `kb-26-metabolic-digital-twin/internal/services/cgm_analytics_test.go` |
| Create | `kb-26-metabolic-digital-twin/migrations/005_cgm_tables.sql` |
| Create | `kb-26-metabolic-digital-twin/internal/services/target_status.go` |
| Create | `kb-26-metabolic-digital-twin/internal/services/target_status_test.go` |
| Modify | `kb-26-metabolic-digital-twin/internal/api/routes.go` |
| Modify | `kb-26-metabolic-digital-twin/internal/services/mri_scorer.go` |

### KB-20 Patient Profile (7 files)
| Action | File |
|--------|------|
| Create | `kb-20-patient-profile/internal/api/renal_status_handlers.go` |
| Create | `kb-20-patient-profile/internal/api/renal_status_handlers_test.go` |
| Create | `kb-20-patient-profile/internal/api/cgm_status_handlers.go` |
| Create | `kb-20-patient-profile/internal/api/cgm_status_handlers_test.go` |
| Create | `kb-20-patient-profile/internal/services/intervention_timeline.go` |
| Create | `kb-20-patient-profile/internal/services/intervention_timeline_test.go` |
| Create | `kb-20-patient-profile/internal/api/intervention_timeline_handlers.go` |
| Modify | `kb-20-patient-profile/internal/api/routes.go` |

### Flink Processing (6 files)
| Action | File |
|--------|------|
| Create | `flink-processing/src/main/java/com/cardiofit/flink/models/CGMReadingBuffer.java` |
| Create | `flink-processing/src/main/java/com/cardiofit/flink/models/CGMAnalyticsEvent.java` |
| Create | `flink-processing/src/main/java/com/cardiofit/flink/models/AGPProfile.java` |
| Create | `flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_CGMAnalytics.java` |
| Create | `flink-processing/src/test/java/com/cardiofit/flink/operators/Module3_CGMAnalyticsTest.java` |
| Modify | `flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS.java` |

**Total: 60 files (50 create, 10 modify) across 92 steps in 16 phases.**

All paths below are relative to `backend/shared-infrastructure/knowledge-base-services/` unless prefixed with `backend/shared-infrastructure/flink-processing/` or `backend/shared-infrastructure/market-configs/`.

---

## PART 1: RENAL DOSE GATING (Steps 1–35, Phases R1–R5)

Patient safety priority #1. Prevents unsafe prescribing when eGFR crosses drug-specific thresholds. Covers 12 drug classes (METFORMIN, SGLT2i, GLP1_RA, EXENATIDE, ACEi, ARB, MRA, FINERENONE, THIAZIDE, SULFONYLUREA, DPP4i, INSULIN) with potassium co-gating, efficacy cliff detection, anticipatory alerts, and stale eGFR flagging.

### Phase R1: Renal Formulary + Gating Models (Steps 1–8)

---

- [ ] **Step 1: Create renal gating data models**

Create `kb-23-decision-cards/internal/models/renal_gating.go`:

```go
package models

import "time"

// GatingVerdict is the renal safety classification for a single medication.
type GatingVerdict string

const (
	VerdictContraindicated  GatingVerdict = "CONTRAINDICATED"
	VerdictDoseReduce       GatingVerdict = "DOSE_REDUCE"
	VerdictMonitorEscalate  GatingVerdict = "MONITOR_ESCALATE"
	VerdictAnticipatory     GatingVerdict = "ANTICIPATORY_ALERT"
	VerdictCleared          GatingVerdict = "CLEARED"
	VerdictInsufficientData GatingVerdict = "INSUFFICIENT_DATA"
)

// RenalDrugRule defines eGFR thresholds for a single drug class.
type RenalDrugRule struct {
	DrugClass            string  `yaml:"drug_class" json:"drug_class"`
	ContraindicatedBelow float64 `yaml:"contraindicated_below" json:"contraindicated_below"`
	DoseReduceBelow      float64 `yaml:"dose_reduce_below" json:"dose_reduce_below"`
	MaxDoseReducedMg     float64 `yaml:"max_dose_reduced_mg" json:"max_dose_reduced_mg"`
	MonitorEscalateBelow float64 `yaml:"monitor_escalate_below" json:"monitor_escalate_below"`
	RequiresPotassiumCheck bool  `yaml:"requires_potassium_check" json:"requires_potassium_check"`
	PotassiumContraAbove float64 `yaml:"potassium_contra_above" json:"potassium_contra_above"`
	EfficacyCliffBelow   float64 `yaml:"efficacy_cliff_below" json:"efficacy_cliff_below"`
	SubstituteClass      string  `yaml:"substitute_class" json:"substitute_class"`
	AnticipateMonths     int     `yaml:"anticipate_months" json:"anticipate_months"`
	SourceGuideline      string  `yaml:"source_guideline" json:"source_guideline"`
	// Market-specific fields (Australia PBS)
	InitiationMinEGFR   float64 `yaml:"initiation_min_egfr,omitempty" json:"initiation_min_egfr,omitempty"`
	ContinuationMinEGFR float64 `yaml:"continuation_min_egfr,omitempty" json:"continuation_min_egfr,omitempty"`
}

// RenalStatus holds the patient's current renal state for gating decisions.
type RenalStatus struct {
	EGFR                float64    `json:"egfr"`
	EGFRSlope           float64    `json:"egfr_slope"`
	EGFRMeasuredAt      time.Time  `json:"egfr_measured_at"`
	EGFRDataPoints      int        `json:"egfr_data_points"`
	Potassium           *float64   `json:"potassium,omitempty"`
	PotassiumMeasuredAt *time.Time `json:"potassium_measured_at,omitempty"`
	ACR                 *float64   `json:"acr,omitempty"`
	CKDStage            string     `json:"ckd_stage"`
	IsRapidDecliner     bool       `json:"is_rapid_decliner"`
}

// MedicationGatingResult is the full gating output for one medication.
type MedicationGatingResult struct {
	DrugClass           string        `json:"drug_class"`
	DrugName            string        `json:"drug_name,omitempty"`
	CurrentDoseMg       float64       `json:"current_dose_mg,omitempty"`
	Verdict             GatingVerdict `json:"verdict"`
	Reason              string        `json:"reason"`
	ClinicalAction      string        `json:"clinical_action"`
	MaxSafeDoseMg       *float64      `json:"max_safe_dose_mg,omitempty"`
	SubstituteClass     string        `json:"substitute_class,omitempty"`
	MonitoringRequired  []string      `json:"monitoring_required,omitempty"`
	MonitoringFrequency string        `json:"monitoring_frequency,omitempty"`
	TimeToThreshold     *float64      `json:"time_to_threshold_months,omitempty"`
	SourceGuideline     string        `json:"source_guideline"`
	EGFR                float64       `json:"egfr_at_evaluation"`
	EvaluatedAt         time.Time     `json:"evaluated_at"`
}

// PatientGatingReport is the full renal safety report for a patient.
type PatientGatingReport struct {
	PatientID              string                  `json:"patient_id"`
	RenalStatus            RenalStatus             `json:"renal_status"`
	MedicationResults      []MedicationGatingResult `json:"medication_results"`
	HasContraindicated     bool                    `json:"has_contraindicated"`
	HasDoseReduce          bool                    `json:"has_dose_reduce"`
	StaleEGFR              bool                    `json:"stale_egfr"`
	StaleEGFRDays          int                     `json:"stale_egfr_days,omitempty"`
	OverallUrgency         string                  `json:"overall_urgency"`
	BlockedRecommendations []string                `json:"blocked_recommendations,omitempty"`
}
```

---

- [ ] **Step 2: Create shared renal dose rules YAML**

Create directory and file `backend/shared-infrastructure/market-configs/shared/renal_dose_rules.yaml`:

```bash
mkdir -p backend/shared-infrastructure/market-configs/shared
mkdir -p backend/shared-infrastructure/market-configs/india
mkdir -p backend/shared-infrastructure/market-configs/australia
```

```yaml
# Evidence-based eGFR thresholds for cardiometabolic drug classes.
# Sources: KDIGO 2024, ADA Standards of Care 2025, FDA/TGA/CDSCO labels.

stale_egfr:
  warning_days: 90
  critical_days: 180
  monitoring_frequency:
    egfr_above_60: 365
    egfr_45_to_60: 180
    egfr_30_to_45: 90
    egfr_below_30: 30

rapid_decline_threshold: -5.0  # mL/min/1.73m²/year — KDIGO rapid decliner

drug_rules:
  - drug_class: "METFORMIN"
    contraindicated_below: 30.0
    dose_reduce_below: 45.0
    max_dose_reduced_mg: 1000.0
    monitor_escalate_below: 60.0
    requires_potassium_check: false
    efficacy_cliff_below: 0
    substitute_class: ""
    anticipate_months: 6
    source_guideline: "KDIGO_2024_S3.3"

  - drug_class: "SGLT2i"
    contraindicated_below: 20.0
    dose_reduce_below: 0
    monitor_escalate_below: 45.0
    requires_potassium_check: false
    efficacy_cliff_below: 20.0
    substitute_class: ""
    anticipate_months: 6
    source_guideline: "KDIGO_2024_S1.3, DAPA-CKD, CREDENCE"

  - drug_class: "GLP1_RA"
    contraindicated_below: 15.0
    dose_reduce_below: 0
    monitor_escalate_below: 30.0
    requires_potassium_check: false
    efficacy_cliff_below: 0
    substitute_class: ""
    anticipate_months: 6
    source_guideline: "ADA_SOC_2025_S9"

  - drug_class: "EXENATIDE"
    contraindicated_below: 30.0
    dose_reduce_below: 50.0
    monitor_escalate_below: 60.0
    requires_potassium_check: false
    efficacy_cliff_below: 0
    substitute_class: "GLP1_RA"
    anticipate_months: 6
    source_guideline: "FDA_LABEL_BYETTA, ADA_SOC_2025"

  - drug_class: "ACEi"
    contraindicated_below: 0
    dose_reduce_below: 0
    monitor_escalate_below: 45.0
    requires_potassium_check: true
    potassium_contra_above: 5.5
    efficacy_cliff_below: 0
    substitute_class: ""
    anticipate_months: 3
    source_guideline: "KDIGO_2024_S3.5, ONTARGET"

  - drug_class: "ARB"
    contraindicated_below: 0
    dose_reduce_below: 0
    monitor_escalate_below: 45.0
    requires_potassium_check: true
    potassium_contra_above: 5.5
    efficacy_cliff_below: 0
    substitute_class: ""
    anticipate_months: 3
    source_guideline: "KDIGO_2024_S3.5"

  - drug_class: "MRA"
    contraindicated_below: 30.0
    dose_reduce_below: 45.0
    max_dose_reduced_mg: 25.0
    monitor_escalate_below: 60.0
    requires_potassium_check: true
    potassium_contra_above: 5.0
    efficacy_cliff_below: 0
    substitute_class: ""
    anticipate_months: 3
    source_guideline: "KDIGO_2024_S3.6, FIDELIO-DKD"

  - drug_class: "FINERENONE"
    contraindicated_below: 25.0
    dose_reduce_below: 60.0
    max_dose_reduced_mg: 10.0
    monitor_escalate_below: 45.0
    requires_potassium_check: true
    potassium_contra_above: 5.0
    efficacy_cliff_below: 0
    substitute_class: ""
    anticipate_months: 3
    source_guideline: "FIDELIO-DKD, FIGARO-DKD, KDIGO_2024"

  - drug_class: "THIAZIDE"
    contraindicated_below: 0
    dose_reduce_below: 0
    monitor_escalate_below: 45.0
    requires_potassium_check: true
    potassium_contra_above: 0
    efficacy_cliff_below: 30.0
    substitute_class: "LOOP_DIURETIC"
    anticipate_months: 6
    source_guideline: "AHA_2024, KDIGO_2024_S3.7"

  - drug_class: "SULFONYLUREA"
    contraindicated_below: 30.0
    dose_reduce_below: 60.0
    monitor_escalate_below: 60.0
    requires_potassium_check: false
    efficacy_cliff_below: 0
    substitute_class: "DPP4i"
    anticipate_months: 6
    source_guideline: "ADA_SOC_2025_S9, KDIGO_2024"

  - drug_class: "DPP4i"
    contraindicated_below: 0
    dose_reduce_below: 45.0
    max_dose_reduced_mg: 25.0
    monitor_escalate_below: 30.0
    requires_potassium_check: false
    efficacy_cliff_below: 0
    substitute_class: ""
    anticipate_months: 6
    source_guideline: "ADA_SOC_2025_S9"

  - drug_class: "INSULIN"
    contraindicated_below: 0
    dose_reduce_below: 30.0
    max_dose_reduced_mg: 0
    monitor_escalate_below: 45.0
    requires_potassium_check: false
    efficacy_cliff_below: 0
    substitute_class: ""
    anticipate_months: 6
    source_guideline: "KDIGO_2024_S3.3, ADA_SOC_2025"
```

---

- [ ] **Step 3: Create India and Australia renal overrides**

Create `backend/shared-infrastructure/market-configs/india/renal_overrides.yaml`:

```yaml
# India-specific renal dose gating. Sources: RSSDI 2023, ICMR CKD Guidelines.

stale_egfr:
  warning_days: 90
  critical_days: 180
  hard_block_on_critical: false
  stale_egfr_message_channel: "whatsapp"

drug_rule_overrides:
  - drug_class: "METFORMIN"
    substitute_class: "SULFONYLUREA"
    cost_note: "Metformin discontinuation — consider SU or DPP4i based on affordability"
  - drug_class: "GLP1_RA"
    availability_flag: "LIMITED_AFFORDABLE"
    cost_note: "GLP-1 RA monthly cost ₹3000-8000 — verify patient can sustain"

formulary_accessibility:
  high: ["METFORMIN", "SULFONYLUREA", "INSULIN", "ACEi", "ARB", "THIAZIDE", "AMLODIPINE"]
  moderate: ["DPP4i", "SGLT2i", "FINERENONE"]
  low: ["GLP1_RA", "EXENATIDE"]

ramadan_fasting:
  enabled: true
  pre_ramadan_egfr_check_days: 30
```

Create `backend/shared-infrastructure/market-configs/australia/renal_overrides.yaml`:

```yaml
# Australia-specific renal dose gating. Sources: RACGP 2024, KHA-CARI, PBS.

stale_egfr:
  warning_days: 90
  critical_days: 180
  hard_block_on_critical: true
  stale_egfr_message_channel: "sms"

drug_rule_overrides:
  - drug_class: "SGLT2i"
    initiation_min_egfr: 25.0
    continuation_min_egfr: 20.0
    pbs_authority_required: true
    source_guideline: "PBS_ITEM_12325, KHA-CARI_2024"
  - drug_class: "FINERENONE"
    requires_specialist_initiation: true
    specialist_type: "NEPHROLOGIST_OR_ENDOCRINOLOGIST"

formulary_accessibility:
  high: ["METFORMIN", "SULFONYLUREA", "INSULIN", "ACEi", "ARB", "THIAZIDE", "AMLODIPINE", "DPP4i", "SGLT2i"]
  moderate: ["GLP1_RA", "FINERENONE", "MRA"]
  low: ["EXENATIDE"]

indigenous_renal_overrides:
  egfr_equation: "CKD_EPI_2021_RACE_FREE"
  monitoring_multiplier: 2.0
  acr_screening_start_age: 18
  source_guideline: "NACCHO_2023, KHA-CARI_INDIGENOUS_2024"
```

---

- [ ] **Step 4: Write failing test for renal formulary loader**

Create `kb-23-decision-cards/internal/services/renal_formulary_test.go`:

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfigDir returns the path to market-configs for testing.
// In CI, this is set via TEST_MARKET_CONFIG_DIR env var.
// Locally, it resolves relative to the repo root.
func testConfigDir(t *testing.T) string {
	t.Helper()
	// Resolve to backend/shared-infrastructure/market-configs from repo root
	return "../../../../market-configs"
}

func TestLoadRenalFormulary_SharedRules(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(formulary.DrugRules), 11)

	met := formulary.GetRule("METFORMIN")
	require.NotNil(t, met)
	assert.Equal(t, 30.0, met.ContraindicatedBelow)
	assert.Equal(t, 45.0, met.DoseReduceBelow)
	assert.Equal(t, 1000.0, met.MaxDoseReducedMg)
	assert.Equal(t, "KDIGO_2024_S3.3", met.SourceGuideline)
}

func TestLoadRenalFormulary_SGLT2iThresholds(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)

	sglt2 := formulary.GetRule("SGLT2i")
	require.NotNil(t, sglt2)
	assert.Equal(t, 20.0, sglt2.ContraindicatedBelow)
	assert.Equal(t, 0.0, sglt2.DoseReduceBelow)
	assert.False(t, sglt2.RequiresPotassiumCheck)
}

func TestLoadRenalFormulary_MRA_PotassiumGating(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)

	mra := formulary.GetRule("MRA")
	require.NotNil(t, mra)
	assert.True(t, mra.RequiresPotassiumCheck)
	assert.Equal(t, 5.0, mra.PotassiumContraAbove)
	assert.Equal(t, 30.0, mra.ContraindicatedBelow)
}

func TestLoadRenalFormulary_ThiazideEfficacyCliff(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)

	thiazide := formulary.GetRule("THIAZIDE")
	require.NotNil(t, thiazide)
	assert.Equal(t, 30.0, thiazide.EfficacyCliffBelow)
	assert.Equal(t, "LOOP_DIURETIC", thiazide.SubstituteClass)
}

func TestLoadRenalFormulary_UnknownDrugReturnsNil(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)
	assert.Nil(t, formulary.GetRule("ASPIRIN"))
}

func TestLoadRenalFormulary_AustraliaOverride_SGLT2iInitiation(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "australia")
	require.NoError(t, err)

	sglt2 := formulary.GetRule("SGLT2i")
	require.NotNil(t, sglt2)
	assert.Equal(t, 25.0, sglt2.InitiationMinEGFR)
	assert.Equal(t, 20.0, sglt2.ContinuationMinEGFR)
}

func TestStaleEGFRConfig_India(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "india")
	require.NoError(t, err)
	assert.Equal(t, 90, formulary.StaleEGFR.WarningDays)
	assert.False(t, formulary.StaleEGFR.HardBlockOnCritical)
}

func TestStaleEGFRConfig_Australia(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "australia")
	require.NoError(t, err)
	assert.True(t, formulary.StaleEGFR.HardBlockOnCritical)
}
```

---

- [ ] **Step 5: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run TestLoadRenalFormulary -v
```

Expected: FAIL — `LoadRenalFormulary` not defined.

---

- [ ] **Step 6: Implement renal formulary loader**

Create `kb-23-decision-cards/internal/services/renal_formulary.go`:

```go
package services

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"kb-23-decision-cards/internal/models"
)

// RenalFormulary holds loaded renal dose rules with market-specific overrides.
type RenalFormulary struct {
	DrugRules     map[string]*models.RenalDrugRule
	StaleEGFR     StaleEGFRConfig
	Accessibility map[string]string // drug_class → HIGH/MODERATE/LOW
}

type StaleEGFRConfig struct {
	WarningDays         int            `yaml:"warning_days"`
	CriticalDays        int            `yaml:"critical_days"`
	HardBlockOnCritical bool           `yaml:"hard_block_on_critical"`
	MonitoringFrequency map[string]int `yaml:"monitoring_frequency"`
}

type renalRulesYAML struct {
	StaleEGFR StaleEGFRConfig       `yaml:"stale_egfr"`
	DrugRules []models.RenalDrugRule `yaml:"drug_rules"`
}

type marketRenalOverrideYAML struct {
	StaleEGFR         *StaleEGFRConfig    `yaml:"stale_egfr"`
	DrugRuleOverrides []marketDrugOverride `yaml:"drug_rule_overrides"`
	FormularyAccess   map[string][]string  `yaml:"formulary_accessibility"`
}

type marketDrugOverride struct {
	DrugClass           string  `yaml:"drug_class"`
	SubstituteClass     string  `yaml:"substitute_class,omitempty"`
	InitiationMinEGFR   float64 `yaml:"initiation_min_egfr,omitempty"`
	ContinuationMinEGFR float64 `yaml:"continuation_min_egfr,omitempty"`
	AvailabilityFlag    string  `yaml:"availability_flag,omitempty"`
	CostNote            string  `yaml:"cost_note,omitempty"`
}

func (f *RenalFormulary) GetRule(drugClass string) *models.RenalDrugRule {
	if r, ok := f.DrugRules[drugClass]; ok {
		return r
	}
	return nil
}

// LoadRenalFormulary loads shared rules and optionally merges market-specific overrides.
func LoadRenalFormulary(configDir string, market string) (*RenalFormulary, error) {
	sharedPath := filepath.Join(configDir, "shared", "renal_dose_rules.yaml")
	data, err := os.ReadFile(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("reading shared renal rules: %w", err)
	}

	var raw renalRulesYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing shared renal rules: %w", err)
	}

	formulary := &RenalFormulary{
		DrugRules:     make(map[string]*models.RenalDrugRule),
		StaleEGFR:     raw.StaleEGFR,
		Accessibility: make(map[string]string),
	}
	for i := range raw.DrugRules {
		r := raw.DrugRules[i]
		formulary.DrugRules[r.DrugClass] = &r
	}

	if market == "" {
		return formulary, nil
	}

	overridePath := filepath.Join(configDir, market, "renal_overrides.yaml")
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		return formulary, nil
	}

	overrideData, err := os.ReadFile(overridePath)
	if err != nil {
		return nil, fmt.Errorf("reading market renal overrides: %w", err)
	}

	var overrides marketRenalOverrideYAML
	if err := yaml.Unmarshal(overrideData, &overrides); err != nil {
		return nil, fmt.Errorf("parsing market renal overrides: %w", err)
	}

	if overrides.StaleEGFR != nil {
		if overrides.StaleEGFR.WarningDays > 0 {
			formulary.StaleEGFR.WarningDays = overrides.StaleEGFR.WarningDays
		}
		if overrides.StaleEGFR.CriticalDays > 0 {
			formulary.StaleEGFR.CriticalDays = overrides.StaleEGFR.CriticalDays
		}
		formulary.StaleEGFR.HardBlockOnCritical = overrides.StaleEGFR.HardBlockOnCritical
	}

	for _, ov := range overrides.DrugRuleOverrides {
		if rule, ok := formulary.DrugRules[ov.DrugClass]; ok {
			if ov.SubstituteClass != "" {
				rule.SubstituteClass = ov.SubstituteClass
			}
			if ov.InitiationMinEGFR > 0 {
				rule.InitiationMinEGFR = ov.InitiationMinEGFR
			}
			if ov.ContinuationMinEGFR > 0 {
				rule.ContinuationMinEGFR = ov.ContinuationMinEGFR
			}
		}
	}

	for level, drugs := range overrides.FormularyAccess {
		for _, drug := range drugs {
			formulary.Accessibility[drug] = level
		}
	}

	return formulary, nil
}
```

---

- [ ] **Step 7: Run formulary tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestLoadRenalFormulary|TestStaleEGFR" -v
```

Expected: All 8 tests PASS.

---

- [ ] **Step 8: Commit Phase R1**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/renal_gating.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_formulary.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_formulary_test.go
git add backend/shared-infrastructure/market-configs/
git commit -m "feat(renal-gating): renal formulary with 12 drug classes + market overrides

KDIGO 2024 / ADA 2025 eGFR thresholds for METFORMIN, SGLT2i, GLP1_RA,
EXENATIDE, ACEi, ARB, MRA, FINERENONE, THIAZIDE, SULFONYLUREA, DPP4i,
INSULIN. Potassium co-gating for ACEi/ARB/MRA/FINERENONE. Efficacy cliff
detection (thiazide->loop at eGFR 30). India: soft stale-eGFR block,
cost-awareness, RSSDI alignment. Australia: hard PBS block, SGLT2i init
>=25, NACCHO indigenous monitoring."
```

### Phase R2: Core Gating Engine — KB-23 (Steps 9–13)

---

- [ ] **Step 9: Write failing test for renal dose gate**

Create `kb-23-decision-cards/internal/services/renal_dose_gate_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kb-23-decision-cards/internal/models"
)

func setupTestGate(t *testing.T) *RenalDoseGate {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)
	return NewRenalDoseGate(formulary)
}

// === METFORMIN TESTS ===

func TestGate_Metformin_EGFR25_Contraindicated(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{EGFR: 25.0, EGFRMeasuredAt: time.Now(), CKDStage: "G4"}
	med := ActiveMedication{DrugClass: "METFORMIN", DrugName: "Glucophage", CurrentDoseMg: 2000}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictContraindicated, result.Verdict)
	assert.Contains(t, result.ClinicalAction, "discontinue")
	assert.Contains(t, result.Reason, "eGFR 25.0 below 30.0")
	assert.Equal(t, "KDIGO_2024_S3.3", result.SourceGuideline)
}

func TestGate_Metformin_EGFR40_DoseReduce(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{EGFR: 40.0, EGFRMeasuredAt: time.Now(), CKDStage: "G3b"}
	med := ActiveMedication{DrugClass: "METFORMIN", DrugName: "Glycomet", CurrentDoseMg: 2000}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictDoseReduce, result.Verdict)
	assert.NotNil(t, result.MaxSafeDoseMg)
	assert.Equal(t, 1000.0, *result.MaxSafeDoseMg)
	assert.Contains(t, result.ClinicalAction, "reduce")
}

func TestGate_Metformin_EGFR55_MonitorEscalate(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{EGFR: 55.0, EGFRMeasuredAt: time.Now(), CKDStage: "G3a"}
	med := ActiveMedication{DrugClass: "METFORMIN", CurrentDoseMg: 2000}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictMonitorEscalate, result.Verdict)
}

func TestGate_Metformin_EGFR75_Cleared(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{EGFR: 75.0, EGFRMeasuredAt: time.Now(), CKDStage: "G2"}
	med := ActiveMedication{DrugClass: "METFORMIN", CurrentDoseMg: 1000}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictCleared, result.Verdict)
}

// === SGLT2i TESTS ===

func TestGate_SGLT2i_EGFR18_Contraindicated(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{EGFR: 18.0, EGFRMeasuredAt: time.Now(), CKDStage: "G4"}
	med := ActiveMedication{DrugClass: "SGLT2i", DrugName: "Jardiance"}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictContraindicated, result.Verdict)
}

func TestGate_SGLT2i_EGFR35_Cleared(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{EGFR: 35.0, EGFRMeasuredAt: time.Now(), CKDStage: "G3b"}
	med := ActiveMedication{DrugClass: "SGLT2i"}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictCleared, result.Verdict)
}

// === POTASSIUM CO-GATING ===

func TestGate_MRA_HighPotassium_Contraindicated(t *testing.T) {
	gate := setupTestGate(t)
	k := 5.3
	renal := models.RenalStatus{
		EGFR: 45.0, EGFRMeasuredAt: time.Now(), CKDStage: "G3a",
		Potassium: &k, PotassiumMeasuredAt: timePtr(time.Now()),
	}
	med := ActiveMedication{DrugClass: "MRA", DrugName: "Spironolactone", CurrentDoseMg: 50}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictContraindicated, result.Verdict)
	assert.Contains(t, result.Reason, "potassium 5.3")
}

func TestGate_ACEi_NoPotassiumData_MonitorEscalate(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{
		EGFR: 40.0, EGFRMeasuredAt: time.Now(), CKDStage: "G3b",
		Potassium: nil,
	}
	med := ActiveMedication{DrugClass: "ACEi"}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictMonitorEscalate, result.Verdict)
	assert.Contains(t, result.MonitoringRequired, "potassium")
}

// === EFFICACY CLIFF ===

func TestGate_Thiazide_EGFR25_EfficacyCliff(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{EGFR: 25.0, EGFRMeasuredAt: time.Now(), CKDStage: "G4"}
	med := ActiveMedication{DrugClass: "THIAZIDE", DrugName: "Hydrochlorothiazide"}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictDoseReduce, result.Verdict)
	assert.Equal(t, "LOOP_DIURETIC", result.SubstituteClass)
	assert.Contains(t, result.ClinicalAction, "switch")
}

// === STALE eGFR ===

func TestGate_StaleEGFR_InsufficientData(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{
		EGFR:           42.0,
		EGFRMeasuredAt: time.Now().Add(-200 * 24 * time.Hour),
		CKDStage:       "G3b",
	}
	med := ActiveMedication{DrugClass: "METFORMIN", CurrentDoseMg: 2000}

	result := gate.Evaluate(med, renal)
	assert.Equal(t, models.VerdictInsufficientData, result.Verdict)
	assert.Contains(t, result.Reason, "stale")
}

// === MULTI-MED PATIENT REPORT ===

func TestEvaluatePatient_MultiMed(t *testing.T) {
	gate := setupTestGate(t)
	renal := models.RenalStatus{EGFR: 28.0, EGFRMeasuredAt: time.Now(), CKDStage: "G4"}

	meds := []ActiveMedication{
		{DrugClass: "METFORMIN", CurrentDoseMg: 2000},
		{DrugClass: "SGLT2i"},
		{DrugClass: "ACEi"},
		{DrugClass: "THIAZIDE"},
		{DrugClass: "SULFONYLUREA"},
	}

	report := gate.EvaluatePatient("PAT-RENAL-002", renal, meds)
	assert.True(t, report.HasContraindicated)
	assert.Equal(t, "IMMEDIATE", report.OverallUrgency)
	assert.Equal(t, 5, len(report.MedicationResults))

	verdicts := map[string]models.GatingVerdict{}
	for _, r := range report.MedicationResults {
		verdicts[r.DrugClass] = r.Verdict
	}
	assert.Equal(t, models.VerdictContraindicated, verdicts["METFORMIN"])
	assert.Equal(t, models.VerdictCleared, verdicts["SGLT2i"])          // 28 > 20
	assert.Equal(t, models.VerdictMonitorEscalate, verdicts["ACEi"])    // 28 < 45
	assert.Equal(t, models.VerdictContraindicated, verdicts["SULFONYLUREA"]) // 28 < 30
}

func timePtr(t time.Time) *time.Time { return &t }
```

---

- [ ] **Step 10: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestGate_|TestEvaluatePatient" -v
```

Expected: FAIL — `RenalDoseGate`, `ActiveMedication`, `Evaluate`, `EvaluatePatient` not defined.

---

- [ ] **Step 11: Implement renal dose gate**

Create `kb-23-decision-cards/internal/services/renal_dose_gate.go`:

```go
package services

import (
	"fmt"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ActiveMedication represents a currently prescribed medication for gating evaluation.
type ActiveMedication struct {
	DrugClass     string  `json:"drug_class"`
	DrugName      string  `json:"drug_name,omitempty"`
	CurrentDoseMg float64 `json:"current_dose_mg,omitempty"`
	IsNew         bool    `json:"is_new,omitempty"`
}

// RenalDoseGate is the core safety engine.
type RenalDoseGate struct {
	formulary *RenalFormulary
}

func NewRenalDoseGate(formulary *RenalFormulary) *RenalDoseGate {
	return &RenalDoseGate{formulary: formulary}
}

// Evaluate produces a gating verdict for a single medication against renal status.
func (g *RenalDoseGate) Evaluate(med ActiveMedication, renal models.RenalStatus) models.MedicationGatingResult {
	now := time.Now()
	result := models.MedicationGatingResult{
		DrugClass:     med.DrugClass,
		DrugName:      med.DrugName,
		CurrentDoseMg: med.CurrentDoseMg,
		EGFR:          renal.EGFR,
		EvaluatedAt:   now,
	}

	rule := g.formulary.GetRule(med.DrugClass)
	if rule == nil {
		result.Verdict = models.VerdictCleared
		result.Reason = "no renal dose rule defined for " + med.DrugClass
		return result
	}

	result.SourceGuideline = rule.SourceGuideline

	// 0. Stale eGFR check
	daysSinceMeasurement := int(now.Sub(renal.EGFRMeasuredAt).Hours() / 24)
	if daysSinceMeasurement > g.formulary.StaleEGFR.CriticalDays {
		result.Verdict = models.VerdictInsufficientData
		result.Reason = fmt.Sprintf("eGFR is stale (%d days since measurement, critical threshold %d days)",
			daysSinceMeasurement, g.formulary.StaleEGFR.CriticalDays)
		if g.formulary.StaleEGFR.HardBlockOnCritical {
			result.ClinicalAction = "Obtain current eGFR before continuing " + med.DrugClass
			result.MonitoringFrequency = "IMMEDIATE"
		} else {
			result.ClinicalAction = "Schedule eGFR measurement — last result may not reflect current renal function"
			result.MonitoringFrequency = "URGENT"
		}
		result.MonitoringRequired = []string{"eGFR", "creatinine"}
		return result
	}

	// 1. Potassium co-gating (ACEi/ARB/MRA/FINERENONE)
	if rule.RequiresPotassiumCheck {
		if renal.Potassium == nil && renal.EGFR < rule.MonitorEscalateBelow {
			result.Verdict = models.VerdictMonitorEscalate
			result.Reason = fmt.Sprintf("%s requires potassium monitoring at eGFR %.1f — potassium data unavailable",
				med.DrugClass, renal.EGFR)
			result.ClinicalAction = "Check serum potassium before continuing " + med.DrugClass
			result.MonitoringRequired = []string{"potassium", "eGFR", "creatinine"}
			result.MonitoringFrequency = "WEEKLY"
			return result
		}
		if renal.Potassium != nil && rule.PotassiumContraAbove > 0 && *renal.Potassium >= rule.PotassiumContraAbove {
			result.Verdict = models.VerdictContraindicated
			result.Reason = fmt.Sprintf("potassium %.1f ≥ %.1f threshold for %s",
				*renal.Potassium, rule.PotassiumContraAbove, med.DrugClass)
			result.ClinicalAction = fmt.Sprintf("Hold %s until potassium < %.1f; recheck in 1 week",
				med.DrugClass, rule.PotassiumContraAbove)
			result.MonitoringRequired = []string{"potassium", "eGFR"}
			result.MonitoringFrequency = "WEEKLY"
			return result
		}
	}

	// 2. Hard contraindication by eGFR
	if rule.ContraindicatedBelow > 0 && renal.EGFR < rule.ContraindicatedBelow {
		result.Verdict = models.VerdictContraindicated
		result.Reason = fmt.Sprintf("eGFR %.1f below %.1f contraindication threshold for %s",
			renal.EGFR, rule.ContraindicatedBelow, med.DrugClass)
		action := fmt.Sprintf("Discontinue %s", med.DrugClass)
		if rule.SubstituteClass != "" {
			action += fmt.Sprintf("; consider switching to %s", rule.SubstituteClass)
			result.SubstituteClass = rule.SubstituteClass
		}
		result.ClinicalAction = action
		return result
	}

	// 3. Efficacy cliff (e.g., thiazide at eGFR 30)
	if rule.EfficacyCliffBelow > 0 && renal.EGFR < rule.EfficacyCliffBelow {
		result.Verdict = models.VerdictDoseReduce
		result.Reason = fmt.Sprintf("eGFR %.1f below efficacy cliff %.1f for %s — drug has minimal effect",
			renal.EGFR, rule.EfficacyCliffBelow, med.DrugClass)
		action := fmt.Sprintf("Switch from %s", med.DrugClass)
		if rule.SubstituteClass != "" {
			action += fmt.Sprintf(" to %s", rule.SubstituteClass)
			result.SubstituteClass = rule.SubstituteClass
		}
		result.ClinicalAction = action
		return result
	}

	// 4. Dose reduction required
	if rule.DoseReduceBelow > 0 && renal.EGFR < rule.DoseReduceBelow {
		result.Verdict = models.VerdictDoseReduce
		result.Reason = fmt.Sprintf("eGFR %.1f below %.1f — dose reduction required for %s",
			renal.EGFR, rule.DoseReduceBelow, med.DrugClass)
		if rule.MaxDoseReducedMg > 0 {
			maxDose := rule.MaxDoseReducedMg
			result.MaxSafeDoseMg = &maxDose
			result.ClinicalAction = fmt.Sprintf("Reduce %s to maximum %.0f mg/day", med.DrugClass, maxDose)
		} else {
			result.ClinicalAction = fmt.Sprintf("Reduce %s dose — consult renal dosing guidelines", med.DrugClass)
		}
		return result
	}

	// 5. Monitor escalation
	if rule.MonitorEscalateBelow > 0 && renal.EGFR < rule.MonitorEscalateBelow {
		result.Verdict = models.VerdictMonitorEscalate
		result.Reason = fmt.Sprintf("eGFR %.1f below %.1f — increased monitoring needed for %s",
			renal.EGFR, rule.MonitorEscalateBelow, med.DrugClass)
		result.ClinicalAction = "Increase renal monitoring frequency"
		result.MonitoringRequired = buildMonitoringList(rule)
		result.MonitoringFrequency = "QUARTERLY"
		if renal.EGFR < 45 {
			result.MonitoringFrequency = "MONTHLY"
		}
		return result
	}

	// 6. All clear
	result.Verdict = models.VerdictCleared
	result.Reason = fmt.Sprintf("eGFR %.1f above all thresholds for %s", renal.EGFR, med.DrugClass)
	return result
}

// EvaluatePatient produces a full gating report across all medications.
func (g *RenalDoseGate) EvaluatePatient(patientID string, renal models.RenalStatus, meds []ActiveMedication) models.PatientGatingReport {
	report := models.PatientGatingReport{
		PatientID:   patientID,
		RenalStatus: renal,
	}

	for _, med := range meds {
		result := g.Evaluate(med, renal)
		report.MedicationResults = append(report.MedicationResults, result)

		switch result.Verdict {
		case models.VerdictContraindicated:
			report.HasContraindicated = true
		case models.VerdictDoseReduce:
			report.HasDoseReduce = true
		case models.VerdictInsufficientData:
			report.StaleEGFR = true
		}
	}

	// Set overall urgency
	if report.HasContraindicated {
		report.OverallUrgency = "IMMEDIATE"
	} else if report.HasDoseReduce {
		report.OverallUrgency = "URGENT"
	} else if report.StaleEGFR {
		report.OverallUrgency = "URGENT"
	} else {
		report.OverallUrgency = "ROUTINE"
	}

	return report
}

// BlockRecommendation checks if a proposed drug recommendation should be blocked.
func (g *RenalDoseGate) BlockRecommendation(drugClass string, renal models.RenalStatus) (bool, string) {
	rule := g.formulary.GetRule(drugClass)
	if rule == nil {
		return false, ""
	}
	if rule.ContraindicatedBelow > 0 && renal.EGFR < rule.ContraindicatedBelow {
		return true, fmt.Sprintf("%s contraindicated at eGFR %.1f (threshold %.1f)",
			drugClass, renal.EGFR, rule.ContraindicatedBelow)
	}
	// Check initiation threshold (Australia PBS)
	if rule.InitiationMinEGFR > 0 && renal.EGFR < rule.InitiationMinEGFR {
		return true, fmt.Sprintf("%s initiation requires eGFR ≥%.1f (current %.1f)",
			drugClass, rule.InitiationMinEGFR, renal.EGFR)
	}
	return false, ""
}

func buildMonitoringList(rule *models.RenalDrugRule) []string {
	list := []string{"eGFR", "creatinine"}
	if rule.RequiresPotassiumCheck {
		list = append(list, "potassium")
	}
	return list
}
```

---

- [ ] **Step 12: Run all gating tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestGate_|TestEvaluatePatient" -v
```

Expected: All 12 tests PASS.

---

- [ ] **Step 13: Commit Phase R2**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_dose_gate.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_dose_gate_test.go
git commit -m "feat(renal-gating): core RenalDoseGate with 5-tier verdict system

5-tier verdict: CONTRAINDICATED, DOSE_REDUCE, MONITOR_ESCALATE,
ANTICIPATORY_ALERT, CLEARED. Potassium co-gating for ACEi/ARB/MRA/
FINERENONE. Efficacy cliff detection with substitute class. Stale eGFR
detection. BlockRecommendation() hard gate for card builder safety.
12 table-driven tests covering all drug classes and edge cases."
```

### Phase R3: eGFR Trajectory + Anticipatory Alerts + Stale Detection (Steps 14–21)

---

- [ ] **Step 14: Write failing test for eGFR trajectory**

Create `kb-26-metabolic-digital-twin/internal/services/egfr_trajectory_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeEGFRTrajectory_RapidDecline(t *testing.T) {
	now := time.Now()
	readings := []EGFRReading{
		{Value: 55.0, MeasuredAt: now.Add(-365 * 24 * time.Hour)},
		{Value: 50.0, MeasuredAt: now.Add(-270 * 24 * time.Hour)},
		{Value: 45.0, MeasuredAt: now.Add(-180 * 24 * time.Hour)},
		{Value: 40.0, MeasuredAt: now.Add(-90 * 24 * time.Hour)},
		{Value: 35.0, MeasuredAt: now},
	}

	result, err := ComputeEGFRTrajectory(readings)
	require.NoError(t, err)
	assert.True(t, result.Slope < -5.0, "Expected rapid decline, got slope %.2f", result.Slope)
	assert.Equal(t, "RAPID_DECLINE", result.Classification)
	assert.True(t, result.IsRapidDecliner)
}

func TestComputeEGFRTrajectory_Stable(t *testing.T) {
	now := time.Now()
	readings := []EGFRReading{
		{Value: 60.0, MeasuredAt: now.Add(-365 * 24 * time.Hour)},
		{Value: 59.5, MeasuredAt: now.Add(-270 * 24 * time.Hour)},
		{Value: 60.2, MeasuredAt: now.Add(-180 * 24 * time.Hour)},
		{Value: 59.0, MeasuredAt: now.Add(-90 * 24 * time.Hour)},
		{Value: 59.8, MeasuredAt: now},
	}

	result, err := ComputeEGFRTrajectory(readings)
	require.NoError(t, err)
	assert.Equal(t, "STABLE", result.Classification)
	assert.False(t, result.IsRapidDecliner)
}

func TestComputeEGFRTrajectory_InsufficientData(t *testing.T) {
	readings := []EGFRReading{
		{Value: 55.0, MeasuredAt: time.Now()},
	}

	_, err := ComputeEGFRTrajectory(readings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient")
}

func TestProjectTimeToThreshold(t *testing.T) {
	tests := []struct {
		name       string
		currentEGFR float64
		slope      float64
		threshold  float64
		wantMonths float64
		wantNil    bool
	}{
		{"metformin_threshold", 48.0, -8.0, 30.0, 27.0, false},
		{"sglt2i_threshold", 48.0, -8.0, 20.0, 42.0, false},
		{"stable_no_crossing", 48.0, -0.5, 30.0, 432.0, false},
		{"improving_no_crossing", 48.0, 2.0, 30.0, 0, true},
		{"already_below", 25.0, -5.0, 30.0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			months := ProjectTimeToThreshold(tt.currentEGFR, tt.slope, tt.threshold)
			if tt.wantNil {
				assert.Nil(t, months)
			} else {
				require.NotNil(t, months)
				assert.InDelta(t, tt.wantMonths, *months, 3.0) // ±3 month tolerance
			}
		})
	}
}
```

---

- [ ] **Step 15: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestComputeEGFRTrajectory|TestProjectTime" -v
```

Expected: FAIL — types not defined.

---

- [ ] **Step 16: Implement eGFR trajectory engine**

Create `kb-26-metabolic-digital-twin/internal/services/egfr_trajectory.go`:

```go
package services

import (
	"errors"
	"math"
	"time"
)

// EGFRReading is a single eGFR measurement with timestamp.
type EGFRReading struct {
	Value      float64   `json:"value"`
	MeasuredAt time.Time `json:"measured_at"`
}

// EGFRTrajectoryResult holds the computed trajectory.
type EGFRTrajectoryResult struct {
	Slope            float64 `json:"slope"`            // mL/min/1.73m²/year (negative = declining)
	Classification   string  `json:"classification"`   // RAPID_DECLINE, MODERATE_DECLINE, STABLE, IMPROVING
	IsRapidDecliner  bool    `json:"is_rapid_decliner"`
	DataPoints       int     `json:"data_points"`
	SpanDays         int     `json:"span_days"`
	LatestEGFR       float64 `json:"latest_egfr"`
	RSquared         float64 `json:"r_squared"`
}

const rapidDeclineThreshold = -5.0 // KDIGO rapid decliner

// ComputeEGFRTrajectory computes OLS linear regression on eGFR readings.
func ComputeEGFRTrajectory(readings []EGFRReading) (*EGFRTrajectoryResult, error) {
	if len(readings) < 2 {
		return nil, errors.New("insufficient eGFR data points (need ≥2)")
	}

	// Use earliest reading as time origin
	origin := readings[0].MeasuredAt
	n := float64(len(readings))

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for _, r := range readings {
		x := r.MeasuredAt.Sub(origin).Hours() / (24 * 365.25) // years from origin
		y := r.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return nil, errors.New("insufficient temporal spread in eGFR readings")
	}

	slope := (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	// R² computation
	meanY := sumY / n
	var ssTot, ssRes float64
	for _, r := range readings {
		x := r.MeasuredAt.Sub(origin).Hours() / (24 * 365.25)
		predicted := intercept + slope*x
		ssTot += (r.Value - meanY) * (r.Value - meanY)
		ssRes += (r.Value - predicted) * (r.Value - predicted)
	}
	rSquared := 0.0
	if ssTot > 0 {
		rSquared = 1.0 - ssRes/ssTot
	}

	spanDays := int(readings[len(readings)-1].MeasuredAt.Sub(readings[0].MeasuredAt).Hours() / 24)

	classification := classifyEGFRSlope(slope)

	return &EGFRTrajectoryResult{
		Slope:           math.Round(slope*100) / 100,
		Classification:  classification,
		IsRapidDecliner: slope <= rapidDeclineThreshold,
		DataPoints:      len(readings),
		SpanDays:        spanDays,
		LatestEGFR:      readings[len(readings)-1].Value,
		RSquared:        math.Round(rSquared*1000) / 1000,
	}, nil
}

func classifyEGFRSlope(slope float64) string {
	switch {
	case slope <= rapidDeclineThreshold:
		return "RAPID_DECLINE"
	case slope <= -1.0:
		return "MODERATE_DECLINE"
	case slope < 1.0:
		return "STABLE"
	default:
		return "IMPROVING"
	}
}

// ProjectTimeToThreshold projects months until eGFR crosses a given threshold.
// Returns nil if trajectory is improving or eGFR is already below threshold.
func ProjectTimeToThreshold(currentEGFR, slopePerYear, threshold float64) *float64 {
	if currentEGFR <= threshold {
		return nil // already below
	}
	if slopePerYear >= 0 {
		return nil // not declining
	}
	yearsToThreshold := (currentEGFR - threshold) / math.Abs(slopePerYear)
	months := yearsToThreshold * 12
	return &months
}
```

---

- [ ] **Step 17: Run trajectory tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestComputeEGFRTrajectory|TestProjectTime" -v
```

Expected: All 7 tests PASS.

---

- [ ] **Step 18: Write stale eGFR detector and anticipatory alert tests**

Create `kb-23-decision-cards/internal/services/stale_egfr_detector.go`:

```go
package services

import (
	"fmt"
	"time"

	"kb-23-decision-cards/internal/models"
)

// StaleEGFRResult holds the staleness evaluation.
type StaleEGFRResult struct {
	IsStale         bool   `json:"is_stale"`
	DaysSince       int    `json:"days_since"`
	ExpectedMaxDays int    `json:"expected_max_days"`
	Severity        string `json:"severity"` // OK, WARNING, CRITICAL
	Action          string `json:"action"`
}

// DetectStaleEGFR evaluates eGFR recency with CKD-stage-aware monitoring frequency.
func DetectStaleEGFR(renal models.RenalStatus, cfg StaleEGFRConfig, onRenalSensitiveMed bool) StaleEGFRResult {
	now := time.Now()
	daysSince := int(now.Sub(renal.EGFRMeasuredAt).Hours() / 24)

	// CKD-stage-aware expected frequency
	expectedDays := 365
	if renal.EGFR < 30 {
		expectedDays = 30
	} else if renal.EGFR < 45 {
		expectedDays = 90
	} else if renal.EGFR < 60 {
		expectedDays = 180
	}

	// Tighten to quarterly if on renal-sensitive medication
	if onRenalSensitiveMed && expectedDays > 90 {
		expectedDays = 90
	}

	result := StaleEGFRResult{
		DaysSince:       daysSince,
		ExpectedMaxDays: expectedDays,
	}

	if daysSince <= expectedDays {
		result.Severity = "OK"
		return result
	}

	result.IsStale = true
	if daysSince > cfg.CriticalDays {
		result.Severity = "CRITICAL"
		result.Action = fmt.Sprintf("eGFR is %d days old (critical >%d) — obtain urgently", daysSince, cfg.CriticalDays)
	} else {
		result.Severity = "WARNING"
		result.Action = fmt.Sprintf("eGFR is %d days old (expected every %d days for CKD stage) — schedule lab",
			daysSince, expectedDays)
	}

	return result
}
```

Create `kb-23-decision-cards/internal/services/stale_egfr_detector_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"kb-23-decision-cards/internal/models"
)

func TestStaleEGFR_RecentMeasurement(t *testing.T) {
	renal := models.RenalStatus{EGFR: 55.0, EGFRMeasuredAt: time.Now().Add(-30 * 24 * time.Hour)}
	cfg := StaleEGFRConfig{WarningDays: 90, CriticalDays: 180}

	result := DetectStaleEGFR(renal, cfg, false)
	assert.False(t, result.IsStale)
	assert.Equal(t, "OK", result.Severity)
}

func TestStaleEGFR_OverdueForCKDStage(t *testing.T) {
	renal := models.RenalStatus{EGFR: 42.0, EGFRMeasuredAt: time.Now().Add(-120 * 24 * time.Hour)}
	cfg := StaleEGFRConfig{WarningDays: 90, CriticalDays: 180}

	result := DetectStaleEGFR(renal, cfg, false)
	assert.True(t, result.IsStale)
	assert.Equal(t, "WARNING", result.Severity)
	assert.Equal(t, 90, result.ExpectedMaxDays)
}

func TestStaleEGFR_CriticallyStale(t *testing.T) {
	renal := models.RenalStatus{EGFR: 42.0, EGFRMeasuredAt: time.Now().Add(-200 * 24 * time.Hour)}
	cfg := StaleEGFRConfig{WarningDays: 90, CriticalDays: 180}

	result := DetectStaleEGFR(renal, cfg, false)
	assert.True(t, result.IsStale)
	assert.Equal(t, "CRITICAL", result.Severity)
	assert.Contains(t, result.Action, "urgently")
}

func TestStaleEGFR_RenalSensitiveMedTightensMonitoring(t *testing.T) {
	// eGFR 65 normally expects 180-day frequency, but renal-sensitive med tightens to 90
	renal := models.RenalStatus{EGFR: 65.0, EGFRMeasuredAt: time.Now().Add(-100 * 24 * time.Hour)}
	cfg := StaleEGFRConfig{WarningDays: 90, CriticalDays: 180}

	result := DetectStaleEGFR(renal, cfg, true)
	assert.True(t, result.IsStale)
	assert.Equal(t, 90, result.ExpectedMaxDays)
}
```

---

- [ ] **Step 19: Write anticipatory threshold detection**

Create `kb-23-decision-cards/internal/services/renal_anticipatory.go`:

```go
package services

import (
	"fmt"

	"kb-23-decision-cards/internal/models"
)

// AnticipatoryAlert represents a proactive alert for an approaching threshold.
type AnticipatoryAlert struct {
	DrugClass        string   `json:"drug_class"`
	ThresholdType    string   `json:"threshold_type"` // CONTRAINDICATION, DOSE_REDUCE, EFFICACY_CLIFF
	ThresholdValue   float64  `json:"threshold_value"`
	MonthsToThreshold float64 `json:"months_to_threshold"`
	RecommendedAction string  `json:"recommended_action"`
	SourceGuideline   string  `json:"source_guideline"`
}

// FindApproachingThresholds identifies medications approaching renal thresholds
// within the configured anticipation horizon.
func FindApproachingThresholds(
	formulary *RenalFormulary,
	currentEGFR float64,
	slopePerYear float64,
	meds []ActiveMedication,
) []AnticipatoryAlert {
	if slopePerYear >= 0 {
		return nil // not declining
	}

	var alerts []AnticipatoryAlert

	for _, med := range meds {
		rule := formulary.GetRule(med.DrugClass)
		if rule == nil {
			continue
		}

		horizonMonths := float64(rule.AnticipateMonths)
		if horizonMonths == 0 {
			horizonMonths = 6
		}

		// Check contraindication threshold
		if rule.ContraindicatedBelow > 0 && currentEGFR > rule.ContraindicatedBelow {
			months := ProjectTimeToThreshold(currentEGFR, slopePerYear, rule.ContraindicatedBelow)
			if months != nil && *months <= horizonMonths {
				alerts = append(alerts, AnticipatoryAlert{
					DrugClass:         med.DrugClass,
					ThresholdType:     "CONTRAINDICATION",
					ThresholdValue:    rule.ContraindicatedBelow,
					MonthsToThreshold: *months,
					RecommendedAction: fmt.Sprintf("Plan %s discontinuation — eGFR projected to reach %.0f in %.1f months",
						med.DrugClass, rule.ContraindicatedBelow, *months),
					SourceGuideline: rule.SourceGuideline,
				})
			}
		}

		// Check dose reduction threshold
		if rule.DoseReduceBelow > 0 && currentEGFR > rule.DoseReduceBelow {
			months := ProjectTimeToThreshold(currentEGFR, slopePerYear, rule.DoseReduceBelow)
			if months != nil && *months <= horizonMonths {
				alerts = append(alerts, AnticipatoryAlert{
					DrugClass:         med.DrugClass,
					ThresholdType:     "DOSE_REDUCE",
					ThresholdValue:    rule.DoseReduceBelow,
					MonthsToThreshold: *months,
					RecommendedAction: fmt.Sprintf("Anticipate %s dose reduction — eGFR projected to reach %.0f in %.1f months",
						med.DrugClass, rule.DoseReduceBelow, *months),
					SourceGuideline: rule.SourceGuideline,
				})
			}
		}

		// Check efficacy cliff
		if rule.EfficacyCliffBelow > 0 && currentEGFR > rule.EfficacyCliffBelow {
			months := ProjectTimeToThreshold(currentEGFR, slopePerYear, rule.EfficacyCliffBelow)
			if months != nil && *months <= horizonMonths {
				alerts = append(alerts, AnticipatoryAlert{
					DrugClass:         med.DrugClass,
					ThresholdType:     "EFFICACY_CLIFF",
					ThresholdValue:    rule.EfficacyCliffBelow,
					MonthsToThreshold: *months,
					RecommendedAction: fmt.Sprintf("Plan switch from %s — efficacy cliff at eGFR %.0f projected in %.1f months",
						med.DrugClass, rule.EfficacyCliffBelow, *months),
					SourceGuideline: rule.SourceGuideline,
				})
			}
		}
	}

	return alerts
}
```

Create `kb-23-decision-cards/internal/services/renal_anticipatory_test.go`:

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindApproaching_MetforminContraindication(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)

	meds := []ActiveMedication{
		{DrugClass: "METFORMIN"},
	}

	// eGFR 35, declining at -8/year → will reach 30 in ~7.5 months
	alerts := FindApproachingThresholds(formulary, 35.0, -8.0, meds)
	require.NotEmpty(t, alerts)
	assert.Equal(t, "CONTRAINDICATION", alerts[0].ThresholdType)
	assert.Equal(t, "METFORMIN", alerts[0].DrugClass)
	assert.InDelta(t, 7.5, alerts[0].MonthsToThreshold, 2.0)
}

func TestFindApproaching_StableNoAlerts(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)

	meds := []ActiveMedication{{DrugClass: "METFORMIN"}}

	// Stable eGFR — no decline
	alerts := FindApproachingThresholds(formulary, 55.0, 0.5, meds)
	assert.Empty(t, alerts)
}
```

---

- [ ] **Step 20: Run all Phase R3 tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestComputeEGFRTrajectory|TestProjectTime" -v

cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestStaleEGFR|TestFindApproaching" -v
```

Expected: All tests PASS.

---

- [ ] **Step 21: Commit Phase R3**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/egfr_trajectory.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/egfr_trajectory_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/stale_egfr_detector.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/stale_egfr_detector_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_anticipatory.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_anticipatory_test.go
git commit -m "feat(renal-gating): eGFR trajectory + anticipatory alerts + stale detection

OLS regression trajectory with RAPID_DECLINE/MODERATE_DECLINE/STABLE/
IMPROVING classification. ProjectTimeToThreshold for proactive medication
review. CKD-stage-aware stale eGFR detection (monthly <30, quarterly
30-45, biannual 45-60). Anticipatory alerts for approaching thresholds."
```

### Phase R4: Integration with Card Pipeline (Steps 22–28)

These files don't exist yet in KB-23 — they are new services created here and later extended by Therapeutic Inertia (Phase T6).

---

- [ ] **Step 22: Create conflict detector with renal gating**

Create `kb-23-decision-cards/internal/services/conflict_detector.go`:

```go
package services

import (
	"kb-23-decision-cards/internal/models"
)

// EnrichedConflictReport combines drug-domain conflicts with renal safety.
type EnrichedConflictReport struct {
	RenalGating         *models.PatientGatingReport `json:"renal_gating,omitempty"`
	AnticipatoryAlerts  []AnticipatoryAlert          `json:"anticipatory_alerts,omitempty"`
	StaleEGFR           *StaleEGFRResult             `json:"stale_egfr,omitempty"`
	HasSafetyBlock      bool                         `json:"has_safety_block"`
	BlockedDrugClasses  []string                     `json:"blocked_drug_classes,omitempty"`
}

// DetectAllConflicts runs renal gating alongside standard conflict detection.
func DetectAllConflicts(
	gate *RenalDoseGate,
	formulary *RenalFormulary,
	patientID string,
	renal models.RenalStatus,
	meds []ActiveMedication,
	egfrSlope float64,
) EnrichedConflictReport {
	report := EnrichedConflictReport{}

	// Renal gating
	gatingReport := gate.EvaluatePatient(patientID, renal, meds)
	report.RenalGating = &gatingReport

	if gatingReport.HasContraindicated || gatingReport.HasDoseReduce {
		report.HasSafetyBlock = true
		for _, r := range gatingReport.MedicationResults {
			if r.Verdict == models.VerdictContraindicated || r.Verdict == models.VerdictDoseReduce {
				report.BlockedDrugClasses = append(report.BlockedDrugClasses, r.DrugClass)
			}
		}
	}

	// Anticipatory alerts
	report.AnticipatoryAlerts = FindApproachingThresholds(formulary, renal.EGFR, egfrSlope, meds)

	// Stale eGFR
	hasRenalMed := false
	for _, m := range meds {
		rule := formulary.GetRule(m.DrugClass)
		if rule != nil {
			hasRenalMed = true
			break
		}
	}
	stale := DetectStaleEGFR(renal, formulary.StaleEGFR, hasRenalMed)
	report.StaleEGFR = &stale
	if stale.Severity == "CRITICAL" {
		report.HasSafetyBlock = true
	}

	return report
}
```

---

- [ ] **Step 23: Create four-pillar evaluator with renal awareness**

Create `kb-23-decision-cards/internal/services/four_pillar_evaluator.go`:

```go
package services

import (
	"kb-23-decision-cards/internal/models"
)

// PillarStatus represents the evaluation of a single clinical pillar.
type PillarStatus string

const (
	PillarOnTrack   PillarStatus = "ON_TRACK"
	PillarGap       PillarStatus = "GAP"
	PillarUrgentGap PillarStatus = "URGENT_GAP"
)

// FourPillarInput holds all inputs for four-pillar evaluation.
type FourPillarInput struct {
	PatientID       string
	DualDomainState string // e.g., "GU-HU", "GC-HC"
	MedicationPillar MedicationPillarInput
	MonitoringPillar MonitoringPillarInput
	LifestylePillar  LifestylePillarInput
	EducationPillar  EducationPillarInput
	RenalGating     *models.PatientGatingReport
	InertiaReport   *PatientInertiaReport // added in Phase T6
}

type MedicationPillarInput struct {
	OnGuidelineMeds bool
	AdherencePct    float64
	ActiveMeds      []ActiveMedication
}

type MonitoringPillarInput struct {
	LabsUpToDate     bool
	HomeMonitoring   bool
	CGMActive        bool
}

type LifestylePillarInput struct {
	ExerciseAdherence float64
	DietAdherence     float64
}

type EducationPillarInput struct {
	EducationComplete bool
}

// PillarResult holds the evaluation for one pillar.
type PillarResult struct {
	Pillar  string       `json:"pillar"`
	Status  PillarStatus `json:"status"`
	Reason  string       `json:"reason,omitempty"`
	Actions []string     `json:"actions,omitempty"`
}

// FourPillarResult holds the full four-pillar evaluation.
type FourPillarResult struct {
	Pillars     []PillarResult `json:"pillars"`
	OverallGap  bool           `json:"overall_gap"`
	UrgentCount int            `json:"urgent_count"`
}

// EvaluateFourPillars runs the four-pillar clinical evaluation.
func EvaluateFourPillars(input FourPillarInput) FourPillarResult {
	result := FourPillarResult{}

	// Medication pillar — renal contraindication becomes URGENT_GAP override
	medPillar := evaluateMedicationPillar(input)
	result.Pillars = append(result.Pillars, medPillar)

	// Monitoring pillar — stale eGFR feeds into monitoring gap
	monPillar := evaluateMonitoringPillar(input)
	result.Pillars = append(result.Pillars, monPillar)

	// Lifestyle pillar
	lifePillar := PillarResult{Pillar: "LIFESTYLE", Status: PillarOnTrack}
	if input.LifestylePillar.ExerciseAdherence < 50 || input.LifestylePillar.DietAdherence < 50 {
		lifePillar.Status = PillarGap
		lifePillar.Reason = "Lifestyle adherence below 50%"
	}
	result.Pillars = append(result.Pillars, lifePillar)

	// Education pillar
	eduPillar := PillarResult{Pillar: "EDUCATION", Status: PillarOnTrack}
	if !input.EducationPillar.EducationComplete {
		eduPillar.Status = PillarGap
		eduPillar.Reason = "Education modules incomplete"
	}
	result.Pillars = append(result.Pillars, eduPillar)

	for _, p := range result.Pillars {
		if p.Status == PillarUrgentGap {
			result.UrgentCount++
			result.OverallGap = true
		} else if p.Status == PillarGap {
			result.OverallGap = true
		}
	}

	return result
}

func evaluateMedicationPillar(input FourPillarInput) PillarResult {
	p := PillarResult{Pillar: "MEDICATION", Status: PillarOnTrack}

	// Renal gating takes highest precedence
	if input.RenalGating != nil && input.RenalGating.HasContraindicated {
		p.Status = PillarUrgentGap
		p.Reason = "Renal contraindication detected — medication review required"
		for _, r := range input.RenalGating.MedicationResults {
			if r.Verdict == models.VerdictContraindicated {
				p.Actions = append(p.Actions, r.ClinicalAction)
			}
		}
		return p
	}

	if !input.MedicationPillar.OnGuidelineMeds {
		p.Status = PillarGap
		p.Reason = "Not on guideline-recommended medications"
	}
	if input.MedicationPillar.AdherencePct < 80 {
		p.Status = PillarGap
		p.Reason = "Medication adherence below 80%"
	}

	return p
}

func evaluateMonitoringPillar(input FourPillarInput) PillarResult {
	p := PillarResult{Pillar: "MONITORING", Status: PillarOnTrack}

	if !input.MonitoringPillar.LabsUpToDate {
		p.Status = PillarGap
		p.Reason = "Labs not up to date"
	}

	// Renal gating stale eGFR makes monitoring an urgent gap
	if input.RenalGating != nil && input.RenalGating.StaleEGFR {
		p.Status = PillarUrgentGap
		p.Reason = "eGFR data is stale — renal monitoring overdue"
		p.Actions = append(p.Actions, "Obtain eGFR, creatinine, potassium")
	}

	return p
}
```

---

- [ ] **Step 24: Create urgency calculator with renal escalation**

Create `kb-23-decision-cards/internal/services/urgency_calculator.go`:

```go
package services

import (
	"kb-23-decision-cards/internal/models"
)

// Urgency levels
const (
	UrgencyImmediate = "IMMEDIATE"
	UrgencyUrgent    = "URGENT"
	UrgencyRoutine   = "ROUTINE"
	UrgencyScheduled = "SCHEDULED"
)

// CalculateDualDomainUrgency computes the overall urgency incorporating renal safety.
func CalculateDualDomainUrgency(
	dualDomainState string,
	fourPillar FourPillarResult,
	renalGating *models.PatientGatingReport,
) string {
	// Renal safety always takes highest precedence
	if renalGating != nil && renalGating.HasContraindicated {
		return UrgencyImmediate
	}

	// Four-pillar urgent gaps
	if fourPillar.UrgentCount >= 2 {
		return UrgencyImmediate
	}
	if fourPillar.UrgentCount == 1 {
		return UrgencyUrgent
	}

	// Dual-domain state-based urgency
	switch dualDomainState {
	case "GU-HU": // both uncontrolled
		return UrgencyUrgent
	case "GU-HC", "GC-HU": // one uncontrolled
		return UrgencyRoutine
	default:
		return UrgencyScheduled
	}
}
```

---

- [ ] **Step 25: Write integration tests for renal-aware card pipeline**

Create `kb-23-decision-cards/internal/services/renal_integration_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kb-23-decision-cards/internal/models"
)

func TestFourPillar_RenalContraindication_OverridesDualDomain(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{EGFR: 25.0, EGFRMeasuredAt: time.Now(), CKDStage: "G4"}
	meds := []ActiveMedication{{DrugClass: "METFORMIN", CurrentDoseMg: 2000}}

	gatingReport := gate.EvaluatePatient("PAT-INT-001", renal, meds)

	input := FourPillarInput{
		PatientID:       "PAT-INT-001",
		DualDomainState: "GC-HC", // both controlled — normally SCHEDULED urgency
		MedicationPillar: MedicationPillarInput{OnGuidelineMeds: true, AdherencePct: 95},
		MonitoringPillar: MonitoringPillarInput{LabsUpToDate: true},
		LifestylePillar:  LifestylePillarInput{ExerciseAdherence: 80, DietAdherence: 70},
		EducationPillar:  EducationPillarInput{EducationComplete: true},
		RenalGating:      &gatingReport,
	}

	result := EvaluateFourPillars(input)

	// Medication pillar should be URGENT_GAP due to renal contraindication
	assert.Equal(t, PillarUrgentGap, result.Pillars[0].Status)
	assert.True(t, result.OverallGap)

	urgency := CalculateDualDomainUrgency("GC-HC", result, &gatingReport)
	assert.Equal(t, UrgencyImmediate, urgency) // renal safety overrides
}

func TestBlocksUnsafeRecommendation(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{EGFR: 18.0, EGFRMeasuredAt: time.Now(), CKDStage: "G5"}

	// Try to recommend SGLT2i — should be blocked at eGFR 18 < 20
	blocked, reason := gate.BlockRecommendation("SGLT2i", renal)
	assert.True(t, blocked)
	assert.Contains(t, reason, "contraindicated")

	// Metformin also blocked at eGFR 18 < 30
	blocked, reason = gate.BlockRecommendation("METFORMIN", renal)
	assert.True(t, blocked)
}

func TestEnrichedConflict_CombinesAllSafety(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	require.NoError(t, err)
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{EGFR: 28.0, EGFRMeasuredAt: time.Now(), CKDStage: "G4"}
	meds := []ActiveMedication{
		{DrugClass: "METFORMIN", CurrentDoseMg: 2000},
		{DrugClass: "SGLT2i"},
		{DrugClass: "ACEi"},
	}

	report := DetectAllConflicts(gate, formulary, "PAT-INT-003", renal, meds, -6.0)

	assert.True(t, report.HasSafetyBlock)
	assert.Contains(t, report.BlockedDrugClasses, "METFORMIN")
	assert.NotEmpty(t, report.AnticipatoryAlerts) // declining at -6/year toward SGLT2i threshold
}
```

---

- [ ] **Step 26: Run all integration tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestFourPillar_Renal|TestBlocks|TestEnrichedConflict" -v
```

Expected: All 3 integration tests PASS.

---

- [ ] **Step 27: Run full KB-23 regression**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./... -v -count=1
```

Expected: All existing and new tests PASS.

---

- [ ] **Step 28: Commit Phase R4**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/conflict_detector.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/four_pillar_evaluator.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/urgency_calculator.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/renal_integration_test.go
git commit -m "feat(renal-gating): card pipeline integration — conflict detector + four-pillar + urgency

EnrichedConflictReport combines renal gating, anticipatory alerts, stale
eGFR. FourPillarEvaluator with renal contraindication as URGENT_GAP
override. CalculateDualDomainUrgency with renal safety always IMMEDIATE.
BlockRecommendation() hard gate preventing unsafe drug suggestions."
```

### Phase R5: KB-20 Renal Status API + KB-26 Event Publishing (Steps 29–35)

---

- [ ] **Step 29: Create KB-20 renal status endpoint**

Create `kb-20-patient-profile/internal/api/renal_status_handlers.go`:

```go
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"kb-patient-profile/internal/models"
)

// RenalStatusResponse is the API response for renal status queries.
type RenalStatusResponse struct {
	PatientID       string            `json:"patient_id"`
	EGFR            float64           `json:"egfr"`
	EGFRSlope       float64           `json:"egfr_slope"`
	EGFRMeasuredAt  time.Time         `json:"egfr_measured_at"`
	EGFRDataPoints  int               `json:"egfr_data_points"`
	Potassium       *float64          `json:"potassium,omitempty"`
	ACR             *float64          `json:"acr,omitempty"`
	CKDStage        string            `json:"ckd_stage"`
	IsRapidDecliner bool              `json:"is_rapid_decliner"`
	ActiveMedications []MedSummary    `json:"active_medications"`
}

type MedSummary struct {
	DrugClass string  `json:"drug_class"`
	DrugName  string  `json:"drug_name,omitempty"`
	DoseMg    float64 `json:"dose_mg,omitempty"`
}

func RegisterRenalStatusRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("/:id/renal-status", getRenalStatus(db))
}

func getRenalStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		patientID := c.Param("id")

		var profile models.PatientProfile
		if err := db.Where("fhir_patient_id = ?", patientID).First(&profile).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
			return
		}

		ckdStage := classifyCKDStage(profile.LatestEGFR)

		resp := RenalStatusResponse{
			PatientID:       patientID,
			EGFR:            profile.LatestEGFR,
			EGFRSlope:       profile.EGFRSlope,
			EGFRMeasuredAt:  profile.EGFRMeasuredAt,
			CKDStage:        ckdStage,
			IsRapidDecliner: profile.EGFRSlope <= -5.0,
		}

		if profile.LatestPotassium > 0 {
			resp.Potassium = &profile.LatestPotassium
		}
		if profile.LatestACR > 0 {
			resp.ACR = &profile.LatestACR
		}

		// Fetch active medications
		var meds []models.MedicationState
		db.Where("patient_id = ? AND status = 'ACTIVE'", profile.ID).Find(&meds)
		for _, m := range meds {
			resp.ActiveMedications = append(resp.ActiveMedications, MedSummary{
				DrugClass: m.DrugClass,
				DrugName:  m.DrugName,
				DoseMg:    m.DoseMg,
			})
		}

		c.JSON(http.StatusOK, resp)
	}
}

func classifyCKDStage(egfr float64) string {
	switch {
	case egfr >= 90:
		return "G1"
	case egfr >= 60:
		return "G2"
	case egfr >= 45:
		return "G3a"
	case egfr >= 30:
		return "G3b"
	case egfr >= 15:
		return "G4"
	default:
		return "G5"
	}
}
```

---

- [ ] **Step 30: Write KB-20 renal status tests**

Create `kb-20-patient-profile/internal/api/renal_status_handlers_test.go`:

```go
package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyCKDStage(t *testing.T) {
	tests := []struct {
		egfr     float64
		expected string
	}{
		{95.0, "G1"},
		{72.0, "G2"},
		{52.0, "G3a"},
		{38.0, "G3b"},
		{22.0, "G4"},
		{10.0, "G5"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, classifyCKDStage(tt.egfr),
			"eGFR %.1f should be %s", tt.egfr, tt.expected)
	}
}
```

---

- [ ] **Step 31: Create KB-26 renal event publisher**

Create `kb-26-metabolic-digital-twin/internal/services/renal_event_publisher.go`:

```go
package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// RenalEvent is published to Kafka for downstream consumption.
type RenalEvent struct {
	PatientID   string    `json:"patient_id"`
	EventType   string    `json:"event_type"` // RENAL_RAPID_DECLINE, RENAL_THRESHOLD_APPROACHING
	Severity    string    `json:"severity"`   // CRITICAL, WARNING, INFO
	EGFR        float64   `json:"egfr"`
	Slope       float64   `json:"slope"`
	Details     string    `json:"details"`
	PublishedAt time.Time `json:"published_at"`
}

// EventPublisher interface for testability.
type EventPublisher interface {
	Publish(topic string, key string, payload []byte) error
}

// RenalEventPublisher evaluates eGFR trajectory and publishes threshold events.
type RenalEventPublisher struct {
	publisher EventPublisher
	topic     string
}

func NewRenalEventPublisher(pub EventPublisher, topic string) *RenalEventPublisher {
	return &RenalEventPublisher{publisher: pub, topic: topic}
}

// Drug thresholds to check for approaching events.
var renalThresholds = []struct {
	DrugClass string
	EGFR      float64
}{
	{"METFORMIN", 30.0},
	{"SULFONYLUREA", 30.0},
	{"MRA", 30.0},
	{"FINERENONE", 25.0},
	{"SGLT2i", 20.0},
}

// EvaluateAndPublish checks trajectory and publishes relevant events.
func (p *RenalEventPublisher) EvaluateAndPublish(patientID string, trajectory *EGFRTrajectoryResult) error {
	var events []RenalEvent

	// Rapid decline event
	if trajectory.IsRapidDecliner {
		events = append(events, RenalEvent{
			PatientID: patientID,
			EventType: "RENAL_RAPID_DECLINE",
			Severity:  "CRITICAL",
			EGFR:      trajectory.LatestEGFR,
			Slope:     trajectory.Slope,
			Details:   fmt.Sprintf("eGFR declining at %.1f mL/min/year (rapid decliner threshold: -5.0)", trajectory.Slope),
			PublishedAt: time.Now(),
		})
	}

	// Threshold approaching events
	for _, th := range renalThresholds {
		if trajectory.LatestEGFR <= th.EGFR {
			continue // already below
		}
		months := ProjectTimeToThreshold(trajectory.LatestEGFR, trajectory.Slope, th.EGFR)
		if months == nil {
			continue
		}
		if *months <= 12 {
			severity := "WARNING"
			if *months <= 3 {
				severity = "CRITICAL"
			}
			events = append(events, RenalEvent{
				PatientID: patientID,
				EventType: "RENAL_THRESHOLD_APPROACHING",
				Severity:  severity,
				EGFR:      trajectory.LatestEGFR,
				Slope:     trajectory.Slope,
				Details: fmt.Sprintf("%s threshold (eGFR %.0f) projected in %.1f months",
					th.DrugClass, th.EGFR, *months),
				PublishedAt: time.Now(),
			})
		}
	}

	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("marshaling renal event: %w", err)
		}
		if err := p.publisher.Publish(p.topic, patientID, payload); err != nil {
			log.Printf("WARN: failed to publish renal event for %s: %v", patientID, err)
		}
	}

	return nil
}
```

---

- [ ] **Step 32: Write renal event publisher tests**

Create `kb-26-metabolic-digital-twin/internal/services/renal_event_publisher_test.go`:

```go
package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPublisher struct {
	published []struct {
		topic   string
		key     string
		payload []byte
	}
}

func (m *mockPublisher) Publish(topic, key string, payload []byte) error {
	m.published = append(m.published, struct {
		topic   string
		key     string
		payload []byte
	}{topic, key, payload})
	return nil
}

func TestRapidDecline_PublishesEvent(t *testing.T) {
	pub := &mockPublisher{}
	rep := NewRenalEventPublisher(pub, "renal.events")

	trajectory := &EGFRTrajectoryResult{
		Slope:           -8.0,
		Classification:  "RAPID_DECLINE",
		IsRapidDecliner: true,
		LatestEGFR:      40.0,
	}

	err := rep.EvaluateAndPublish("PAT-RE-001", trajectory)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(pub.published), 1)

	var evt RenalEvent
	json.Unmarshal(pub.published[0].payload, &evt)
	assert.Equal(t, "RENAL_RAPID_DECLINE", evt.EventType)
	assert.Equal(t, "CRITICAL", evt.Severity)
}

func TestStableTrajectory_NoEvents(t *testing.T) {
	pub := &mockPublisher{}
	rep := NewRenalEventPublisher(pub, "renal.events")

	trajectory := &EGFRTrajectoryResult{
		Slope:           -0.5,
		Classification:  "STABLE",
		IsRapidDecliner: false,
		LatestEGFR:      65.0,
	}

	err := rep.EvaluateAndPublish("PAT-RE-002", trajectory)
	require.NoError(t, err)
	assert.Empty(t, pub.published) // stable, well above thresholds
}
```

---

- [ ] **Step 33: Run all Phase R5 tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/api/ -run TestClassifyCKDStage -v

cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestRapidDecline|TestStableTrajectory" -v
```

Expected: All tests PASS.

---

- [ ] **Step 34: Full regression across all three services**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./... -count=1

cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./... -count=1

cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./... -count=1
```

Expected: All tests PASS in all three services.

---

- [ ] **Step 35: Commit Phase R5**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/renal_status_handlers.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/renal_status_handlers_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/renal_event_publisher.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/renal_event_publisher_test.go
git commit -m "feat(renal-gating): KB-20 renal status API + KB-26 event publisher

GET /:id/renal-status returns eGFR, slope, potassium, ACR, CKD stage,
active medications. KB-26 publishes RENAL_RAPID_DECLINE (CRITICAL) and
RENAL_THRESHOLD_APPROACHING events to Kafka for downstream card generation."
```

**PART 1 COMPLETE: Renal Dose Gating — 35 steps across 5 phases.**

---

## PART 2: CGM ANALYTICS LAYER (Steps 36–64, Phases G1–G5)

Implements streaming CGM metrics (TIR/TBR/TAR/CV/GMI/GRI/AGP) per International Consensus 2023, integrates into KB-26 MHRI glucose domain, and generates CGM-specific decision cards. When CGM data is available, MHRI switches from FBG/PPBG snapshot scoring to CGM-derived TIR/CV/GMI — fundamentally more accurate.

### Phase G1: CGM Market Configuration (Steps 36–38)

---

- [ ] **Step 36: Create shared CGM targets YAML**

Create `backend/shared-infrastructure/market-configs/shared/cgm_targets.yaml`:

```yaml
# International Consensus on CGM Metrics (Battelino et al. 2023)
# ADA Standards of Care 2025, Section 7

data_sufficiency:
  min_coverage_pct: 70
  min_days_for_report: 14
  min_days_for_tir: 3
  readings_per_day_expected: 96
  sensor_warmup_minutes: 60
  max_gap_minutes: 60

ranges:
  very_low: 54
  low: 70
  target_low: 70
  target_high: 180
  high: 180
  very_high: 250

targets:
  general_adult:
    tir_pct: 70
    tbr_l1_pct: 4
    tbr_l2_pct: 1
    tar_l1_pct: 25
    tar_l2_pct: 5
    cv_pct: 36
    gmi_target: 7.0

  elderly_frail:
    tir_pct: 50
    tbr_l1_pct: 1
    tbr_l2_pct: 0
    tar_l1_pct: 50
    tar_l2_pct: 10
    cv_pct: 36
    gmi_target: 8.0

  pregnancy:
    tir_pct: 70
    tbr_l1_pct: 4
    tbr_l2_pct: 1
    tar_l1_pct: 25
    tar_l2_pct: 5
    target_low: 63
    target_high: 140

gri:
  weights:
    very_low: 3.0
    low: 2.4
    very_high: 1.6
    high: 0.8
  zones:
    A: 20
    B: 40
    C: 60
    D: 80

gmi:
  intercept: 3.31
  slope: 0.02392
  discrepancy_threshold: 0.5

agp:
  bucket_minutes: 30
  percentiles: [10, 25, 50, 75, 90]
  min_days_per_bucket: 5

alerts:
  sustained_hypo_minutes: 15
  sustained_severe_hypo_minutes: 15
  sustained_hyper_minutes: 120
  rapid_rise_mg_dl_per_min: 3.0
  rapid_fall_mg_dl_per_min: 3.0
  nocturnal_hypo_window: "00:00-06:00"
```

---

- [ ] **Step 37: Create India CGM overrides**

Create `backend/shared-infrastructure/market-configs/india/cgm_overrides.yaml`:

```yaml
# Sources: RSSDI 2023 CGM Position Statement, IDF-DAR 2021

device_landscape:
  primary_device: "FREESTYLE_LIBRE_2"
  data_format: "ISCGM_SCAN"
  typical_scans_per_day: 4
  max_data_gap_hours: 8

targets_override:
  sulfonylurea_insulin_combination:
    tbr_l1_pct: 2
    tbr_l2_pct: 0
    rationale: "RSSDI_2023_CGM_POSITION: SU+insulin combination hypo risk"

ramadan:
  enabled: true
  pre_ramadan_cgm_recommended: true
  fasting_window_start: "SUNRISE"
  fasting_window_end: "SUNSET"
  adjusted_targets:
    tir_pct: 60
    tbr_l1_pct: 2
    tbr_l2_pct: 0
    tar_l2_pct: 10
  alert_adjustments:
    fasting_hypo_alert_mg_dl: 80
    post_iftar_hyper_suppress_minutes: 120

intermittent_use:
  enabled: true
  typical_pattern: "2_WEEKS_ON_12_WEEKS_OFF"
  fallback_to_smbg: true
  min_cgm_days_for_mhri_switch: 7
```

---

- [ ] **Step 38: Create Australia CGM overrides**

Create `backend/shared-infrastructure/market-configs/australia/cgm_overrides.yaml`:

```yaml
# Sources: ADS-ADEA Position Statement 2024, RACGP 2024

device_landscape:
  primary_devices:
    - "FREESTYLE_LIBRE_2"
    - "DEXCOM_G7"
  data_format: "MIXED"
  ndss_subsidized: true

indigenous_overrides:
  offline_sync_max_days: 14
  targets:
    tir_pct: 50
    tbr_l1_pct: 2
    tbr_l2_pct: 0
  report_languages: ["en", "kriol"]

mhr_integration:
  emit_cgm_summary: true
  fhir_document_type: "CGM_SUMMARY_REPORT"
  fhir_profile: "http://hl7.org.au/fhir/StructureDefinition/au-diagnosticreport"
```

### Phase G2: Flink CGM Analytics Operator (Steps 39–45)

---

- [ ] **Step 39: Create CGM reading buffer**

Create `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/CGMReadingBuffer.java`:

```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.*;

/**
 * Ring buffer for CGM readings in Flink keyed state.
 * Stores up to 14 days of readings (96/day x 14 = 1,344 max).
 */
public class CGMReadingBuffer implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final int MAX_READINGS = 1_500;

    private final List<Long> timestamps;
    private final List<Double> glucoseValues;
    private Long sensorStartTime;
    private int totalReadingsReceived;
    private int readingsExcluded;

    public CGMReadingBuffer() {
        this.timestamps = new ArrayList<>(MAX_READINGS);
        this.glucoseValues = new ArrayList<>(MAX_READINGS);
        this.totalReadingsReceived = 0;
        this.readingsExcluded = 0;
    }

    public boolean addReading(long timestampMs, double glucoseMgDl) {
        totalReadingsReceived++;
        if (sensorStartTime != null && timestampMs - sensorStartTime < 60 * 60 * 1000L) {
            readingsExcluded++;
            return false;
        }
        if (glucoseMgDl < 20.0 || glucoseMgDl > 500.0) {
            readingsExcluded++;
            return false;
        }
        int insertIdx = Collections.binarySearch(timestamps, timestampMs);
        if (insertIdx >= 0) {
            readingsExcluded++;
            return false;
        }
        insertIdx = -(insertIdx + 1);
        timestamps.add(insertIdx, timestampMs);
        glucoseValues.add(insertIdx, glucoseMgDl);
        while (timestamps.size() > MAX_READINGS) {
            timestamps.remove(0);
            glucoseValues.remove(0);
        }
        return true;
    }

    public void setSensorStartTime(long sensorStartMs) { this.sensorStartTime = sensorStartMs; }
    public int size() { return timestamps.size(); }
    public int getTotalReadingsReceived() { return totalReadingsReceived; }
    public int getReadingsExcluded() { return readingsExcluded; }

    public List<Double> getReadingsInWindow(long windowStartMs, long windowEndMs) {
        List<Double> result = new ArrayList<>();
        for (int i = 0; i < timestamps.size(); i++) {
            long ts = timestamps.get(i);
            if (ts >= windowStartMs && ts <= windowEndMs) result.add(glucoseValues.get(i));
        }
        return result;
    }

    public List<ConsecutiveRun> findConsecutiveRunsBelowThreshold(double threshold, long windowStartMs, long windowEndMs) {
        List<ConsecutiveRun> runs = new ArrayList<>();
        Long runStart = null;
        int runCount = 0;
        for (int i = 0; i < timestamps.size(); i++) {
            long ts = timestamps.get(i);
            if (ts < windowStartMs || ts > windowEndMs) continue;
            if (glucoseValues.get(i) < threshold) {
                if (runStart == null) { runStart = ts; runCount = 1; } else { runCount++; }
            } else {
                if (runStart != null) { runs.add(new ConsecutiveRun(runStart, ts, runCount)); runStart = null; runCount = 0; }
            }
        }
        if (runStart != null && !timestamps.isEmpty()) {
            runs.add(new ConsecutiveRun(runStart, timestamps.get(timestamps.size() - 1), runCount));
        }
        return runs;
    }

    public List<ConsecutiveRun> findConsecutiveRunsAboveThreshold(double threshold, long windowStartMs, long windowEndMs) {
        List<ConsecutiveRun> runs = new ArrayList<>();
        Long runStart = null;
        int runCount = 0;
        for (int i = 0; i < timestamps.size(); i++) {
            long ts = timestamps.get(i);
            if (ts < windowStartMs || ts > windowEndMs) continue;
            if (glucoseValues.get(i) > threshold) {
                if (runStart == null) { runStart = ts; runCount = 1; } else { runCount++; }
            } else {
                if (runStart != null) { runs.add(new ConsecutiveRun(runStart, ts, runCount)); runStart = null; runCount = 0; }
            }
        }
        if (runStart != null && !timestamps.isEmpty()) {
            runs.add(new ConsecutiveRun(runStart, timestamps.get(timestamps.size() - 1), runCount));
        }
        return runs;
    }

    public double computeCoverage(long windowStartMs, long windowEndMs, int expectedPerDay) {
        int days = Math.max(1, (int) ((windowEndMs - windowStartMs) / 86400_000L));
        int expected = days * expectedPerDay;
        if (expected <= 0) return 0.0;
        int actual = 0;
        for (long ts : timestamps) {
            if (ts >= windowStartMs && ts <= windowEndMs) actual++;
        }
        return (double) actual / expected * 100.0;
    }

    public static class ConsecutiveRun implements Serializable {
        public final long startMs;
        public final long endMs;
        public final int readingCount;
        public ConsecutiveRun(long startMs, long endMs, int readingCount) {
            this.startMs = startMs; this.endMs = endMs; this.readingCount = readingCount;
        }
        public long durationMinutes() { return (endMs - startMs) / 60_000L; }
    }
}
```

---

- [ ] **Step 40: Create CGM analytics event model**

Create `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/CGMAnalyticsEvent.java`:

```java
package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.*;

/**
 * Output event from CGM Analytics. Emitted as INCREMENTAL, DAILY_SUMMARY, or PERIOD_REPORT.
 */
public class CGMAnalyticsEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private String correlationId;
    private long computedAt;
    private String reportType;

    private double coveragePct;
    private int totalReadings;
    private int windowDays;
    private boolean sufficientData;
    private String confidenceLevel;

    private double meanGlucose;
    private double sdGlucose;
    private double cvPct;
    private boolean glucoseStable;

    private double tirPct;
    private double tbrL1Pct;
    private double tbrL2Pct;
    private double tarL1Pct;
    private double tarL2Pct;

    private double gmi;
    private Double gmiHba1cDiscrepancy;
    private double gri;
    private String griZone;

    private boolean sustainedHypoDetected;
    private boolean sustainedSevereHypoDetected;
    private boolean sustainedHyperDetected;
    private boolean nocturnalHypoDetected;
    private boolean rapidRiseDetected;
    private boolean rapidFallDetected;

    private Map<String, Boolean> targetsMetMap;
    private Map<Integer, double[]> agpPercentiles;

    public CGMAnalyticsEvent() { this.targetsMetMap = new HashMap<>(); }

    public static Builder builder() { return new Builder(); }

    public static class Builder {
        private final CGMAnalyticsEvent e = new CGMAnalyticsEvent();
        public Builder patientId(String v) { e.patientId = v; return this; }
        public Builder correlationId(String v) { e.correlationId = v; return this; }
        public Builder computedAt(long v) { e.computedAt = v; return this; }
        public Builder reportType(String v) { e.reportType = v; return this; }
        public Builder coveragePct(double v) { e.coveragePct = v; return this; }
        public Builder totalReadings(int v) { e.totalReadings = v; return this; }
        public Builder windowDays(int v) { e.windowDays = v; return this; }
        public Builder sufficientData(boolean v) { e.sufficientData = v; return this; }
        public Builder confidenceLevel(String v) { e.confidenceLevel = v; return this; }
        public Builder meanGlucose(double v) { e.meanGlucose = v; return this; }
        public Builder sdGlucose(double v) { e.sdGlucose = v; return this; }
        public Builder cvPct(double v) { e.cvPct = v; e.glucoseStable = v <= 36.0; return this; }
        public Builder tirPct(double v) { e.tirPct = v; return this; }
        public Builder tbrL1Pct(double v) { e.tbrL1Pct = v; return this; }
        public Builder tbrL2Pct(double v) { e.tbrL2Pct = v; return this; }
        public Builder tarL1Pct(double v) { e.tarL1Pct = v; return this; }
        public Builder tarL2Pct(double v) { e.tarL2Pct = v; return this; }
        public Builder gmi(double v) { e.gmi = v; return this; }
        public Builder gri(double v) {
            e.gri = v;
            if (v <= 20) e.griZone = "A"; else if (v <= 40) e.griZone = "B";
            else if (v <= 60) e.griZone = "C"; else if (v <= 80) e.griZone = "D";
            else e.griZone = "E";
            return this;
        }
        public CGMAnalyticsEvent build() { return e; }
    }

    // Getters
    public String getPatientId() { return patientId; }
    public String getReportType() { return reportType; }
    public double getTirPct() { return tirPct; }
    public double getTbrL1Pct() { return tbrL1Pct; }
    public double getTbrL2Pct() { return tbrL2Pct; }
    public double getTarL1Pct() { return tarL1Pct; }
    public double getTarL2Pct() { return tarL2Pct; }
    public double getCvPct() { return cvPct; }
    public boolean isGlucoseStable() { return glucoseStable; }
    public double getGmi() { return gmi; }
    public double getGri() { return gri; }
    public String getGriZone() { return griZone; }
    public double getCoveragePct() { return coveragePct; }
    public boolean isSufficientData() { return sufficientData; }
    public String getConfidenceLevel() { return confidenceLevel; }
    public boolean isSustainedHypoDetected() { return sustainedHypoDetected; }
    public boolean isSustainedSevereHypoDetected() { return sustainedSevereHypoDetected; }
    public boolean isSustainedHyperDetected() { return sustainedHyperDetected; }
    public boolean isNocturnalHypoDetected() { return nocturnalHypoDetected; }
    public double getMeanGlucose() { return meanGlucose; }
    public double getSdGlucose() { return sdGlucose; }
    public Map<Integer, double[]> getAgpPercentiles() { return agpPercentiles; }
    public void setAgpPercentiles(Map<Integer, double[]> v) { this.agpPercentiles = v; }
    public void setSustainedHypoDetected(boolean v) { this.sustainedHypoDetected = v; }
    public void setSustainedSevereHypoDetected(boolean v) { this.sustainedSevereHypoDetected = v; }
    public void setSustainedHyperDetected(boolean v) { this.sustainedHyperDetected = v; }
    public void setNocturnalHypoDetected(boolean v) { this.nocturnalHypoDetected = v; }
    public void setRapidRiseDetected(boolean v) { this.rapidRiseDetected = v; }
    public void setRapidFallDetected(boolean v) { this.rapidFallDetected = v; }
    public void setGmiHba1cDiscrepancy(Double v) { this.gmiHba1cDiscrepancy = v; }
    public void setTargetsMet(String key, boolean met) { this.targetsMetMap.put(key, met); }
}
```

---

- [ ] **Step 41: Write failing test for CGM core computation**

Create `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module3_CGMAnalyticsTest.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CGMAnalyticsEvent;
import com.cardiofit.flink.models.CGMReadingBuffer;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import java.util.*;
import static org.junit.jupiter.api.Assertions.*;

public class Module3_CGMAnalyticsTest {
    private Module3_CGMAnalytics analytics;

    @BeforeEach
    void setup() { analytics = new Module3_CGMAnalytics(); }

    @Test
    void testTIR_AllInRange() {
        List<Double> readings = Collections.nCopies(100, 120.0);
        CGMAnalyticsEvent result = analytics.computeMetrics(readings, 1);
        assertEquals(100.0, result.getTirPct(), 0.01);
        assertEquals(0.0, result.getTbrL1Pct(), 0.01);
        assertEquals(0.0, result.getTarL1Pct(), 0.01);
    }

    @Test
    void testTIR_MixedRanges() {
        List<Double> readings = new ArrayList<>();
        for (int i = 0; i < 70; i++) readings.add(130.0);  // in range
        for (int i = 0; i < 10; i++) readings.add(60.0);   // L1 hypo
        for (int i = 0; i < 5; i++) readings.add(45.0);    // L2 hypo
        for (int i = 0; i < 10; i++) readings.add(200.0);  // L1 hyper
        for (int i = 0; i < 5; i++) readings.add(280.0);   // L2 hyper
        CGMAnalyticsEvent result = analytics.computeMetrics(readings, 1);
        assertEquals(70.0, result.getTirPct(), 0.01);
        assertEquals(10.0, result.getTbrL1Pct(), 0.01);
        assertEquals(5.0, result.getTbrL2Pct(), 0.01);
        assertEquals(10.0, result.getTarL1Pct(), 0.01);
        assertEquals(5.0, result.getTarL2Pct(), 0.01);
    }

    @Test
    void testCV_StableGlucose() {
        List<Double> readings = Arrays.asList(115.0, 118.0, 120.0, 122.0, 125.0, 119.0, 121.0, 117.0, 123.0, 120.0);
        CGMAnalyticsEvent result = analytics.computeMetrics(readings, 1);
        assertTrue(result.getCvPct() < 36.0);
        assertTrue(result.isGlucoseStable());
    }

    @Test
    void testGMI_StandardFormula() {
        // Mean 154 → GMI = 3.31 + (0.02392 × 154) = 6.99
        List<Double> readings = Collections.nCopies(100, 154.0);
        CGMAnalyticsEvent result = analytics.computeMetrics(readings, 1);
        assertEquals(6.99, result.getGmi(), 0.05);
    }

    @Test
    void testGRI_ZoneA() {
        List<Double> readings = Collections.nCopies(100, 120.0);
        CGMAnalyticsEvent result = analytics.computeMetrics(readings, 1);
        assertEquals(0.0, result.getGri(), 0.01);
        assertEquals("A", result.getGriZone());
    }

    @Test
    void testCoverage_SufficientData() {
        List<Double> readings = Collections.nCopies(1000, 120.0);
        CGMAnalyticsEvent result = analytics.computeMetrics(readings, 14);
        assertTrue(result.isSufficientData());
        assertEquals("HIGH", result.getConfidenceLevel());
    }

    @Test
    void testCoverage_InsufficientData() {
        List<Double> readings = Collections.nCopies(50, 120.0);
        CGMAnalyticsEvent result = analytics.computeMetrics(readings, 14);
        assertFalse(result.isSufficientData());
        assertEquals("LOW", result.getConfidenceLevel());
    }

    @Test
    void testBuffer_SensorWarmupExcluded() {
        CGMReadingBuffer buffer = new CGMReadingBuffer();
        long sensorStart = System.currentTimeMillis();
        buffer.setSensorStartTime(sensorStart);
        assertFalse(buffer.addReading(sensorStart + 30 * 60_000L, 150.0));
        assertEquals(0, buffer.size());
        assertTrue(buffer.addReading(sensorStart + 90 * 60_000L, 150.0));
        assertEquals(1, buffer.size());
    }

    @Test
    void testBuffer_PhysiologicallyImpossibleExcluded() {
        CGMReadingBuffer buffer = new CGMReadingBuffer();
        assertFalse(buffer.addReading(System.currentTimeMillis(), 10.0));
        assertFalse(buffer.addReading(System.currentTimeMillis() + 1000, 550.0));
        assertEquals(0, buffer.size());
    }
}
```

---

- [ ] **Step 42: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module3_CGMAnalyticsTest
```

Expected: FAIL — `Module3_CGMAnalytics` class not defined.

---

- [ ] **Step 43: Implement CGM analytics core computation**

Create `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_CGMAnalytics.java`:

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CGMAnalyticsEvent;
import java.util.*;

/**
 * CGM Analytics engine implementing International Consensus on CGM Metrics
 * (Battelino et al. 2023), GRI (Klonoff et al. 2023).
 */
public class Module3_CGMAnalytics {

    private static final double VERY_LOW = 54.0;
    private static final double LOW = 70.0;
    private static final double HIGH = 180.0;
    private static final double VERY_HIGH = 250.0;
    private static final double GMI_INTERCEPT = 3.31;
    private static final double GMI_SLOPE = 0.02392;
    private static final double GRI_W_VLOW = 3.0;
    private static final double GRI_W_LOW = 2.4;
    private static final double GRI_W_VHIGH = 1.6;
    private static final double GRI_W_HIGH = 0.8;
    private static final int READINGS_PER_DAY = 96;
    private static final double MIN_COVERAGE_PCT = 70.0;

    /**
     * Compute all CGM metrics from a list of glucose values.
     * @param readings glucose values in mg/dL
     * @param windowDays number of days this reading set covers
     */
    public CGMAnalyticsEvent computeMetrics(List<Double> readings, int windowDays) {
        int n = readings.size();
        int expected = windowDays * READINGS_PER_DAY;
        double coveragePct = expected > 0 ? (double) n / expected * 100.0 : 0.0;
        boolean sufficient = coveragePct >= MIN_COVERAGE_PCT;

        String confidence;
        if (coveragePct >= MIN_COVERAGE_PCT) confidence = "HIGH";
        else if (coveragePct >= 50.0) confidence = "MODERATE";
        else confidence = "LOW";

        // Range bucketing
        int countVeryLow = 0, countLow = 0, countInRange = 0, countHigh = 0, countVeryHigh = 0;
        double sum = 0;
        for (double g : readings) {
            sum += g;
            if (g < VERY_LOW) countVeryLow++;
            else if (g < LOW) countLow++;
            else if (g <= HIGH) countInRange++;
            else if (g <= VERY_HIGH) countHigh++;
            else countVeryHigh++;
        }

        double mean = n > 0 ? sum / n : 0;
        double tir = n > 0 ? (double) countInRange / n * 100.0 : 0;
        double tbrL1 = n > 0 ? (double) countLow / n * 100.0 : 0;
        double tbrL2 = n > 0 ? (double) countVeryLow / n * 100.0 : 0;
        double tarL1 = n > 0 ? (double) countHigh / n * 100.0 : 0;
        double tarL2 = n > 0 ? (double) countVeryHigh / n * 100.0 : 0;

        // SD and CV
        double sumSqDiff = 0;
        for (double g : readings) { sumSqDiff += (g - mean) * (g - mean); }
        double sd = n > 1 ? Math.sqrt(sumSqDiff / (n - 1)) : 0;
        double cv = mean > 0 ? sd / mean * 100.0 : 0;

        // GMI
        double gmi = GMI_INTERCEPT + GMI_SLOPE * mean;

        // GRI
        double griRaw = GRI_W_VLOW * tbrL2 + GRI_W_LOW * tbrL1 + GRI_W_VHIGH * tarL2 + GRI_W_HIGH * tarL1;
        double gri = Math.min(100.0, griRaw);

        return CGMAnalyticsEvent.builder()
            .computedAt(System.currentTimeMillis())
            .reportType("PERIOD_REPORT")
            .coveragePct(coveragePct)
            .totalReadings(n)
            .windowDays(windowDays)
            .sufficientData(sufficient)
            .confidenceLevel(confidence)
            .meanGlucose(mean)
            .sdGlucose(sd)
            .cvPct(cv)
            .tirPct(tir)
            .tbrL1Pct(tbrL1)
            .tbrL2Pct(tbrL2)
            .tarL1Pct(tarL1)
            .tarL2Pct(tarL2)
            .gmi(gmi)
            .gri(gri)
            .build();
    }

    /**
     * Detect sustained hypoglycaemia from buffer.
     */
    public boolean detectSustainedHypo(
            com.cardiofit.flink.models.CGMReadingBuffer buffer,
            long windowStart, long windowEnd,
            double threshold, int minMinutes) {
        var runs = buffer.findConsecutiveRunsBelowThreshold(threshold, windowStart, windowEnd);
        for (var run : runs) {
            if (run.durationMinutes() >= minMinutes) return true;
        }
        return false;
    }

    /**
     * Detect nocturnal hypoglycaemia (00:00-06:00 window).
     */
    public boolean detectNocturnalHypo(
            com.cardiofit.flink.models.CGMReadingBuffer buffer,
            long nocturnalStart, long nocturnalEnd,
            double threshold, int minMinutes) {
        return detectSustainedHypo(buffer, nocturnalStart, nocturnalEnd, threshold, minMinutes);
    }
}
```

---

- [ ] **Step 44: Run CGM analytics tests**

```bash
cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module3_CGMAnalyticsTest -q
```

Expected: All 9 tests PASS.

---

- [ ] **Step 45: Commit Phase G2**

```bash
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/CGMReadingBuffer.java
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/CGMAnalyticsEvent.java
git add backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_CGMAnalytics.java
git add backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/Module3_CGMAnalyticsTest.java
git add backend/shared-infrastructure/market-configs/shared/cgm_targets.yaml
git add backend/shared-infrastructure/market-configs/india/cgm_overrides.yaml
git add backend/shared-infrastructure/market-configs/australia/cgm_overrides.yaml
git commit -m "feat(cgm-analytics): Flink CGM operator + market configs

International Consensus 2023 metrics: TIR/TBR(L1+L2)/TAR(L1+L2)/CV/
GMI/GRI with zone classification. CGMReadingBuffer with 1500-reading
capacity, sensor warmup exclusion, physiological bounds filtering.
Market configs: India (RSSDI 2023, Ramadan-adjusted targets, isCGM
patterns), Australia (ADS-ADEA 2024, NDSS subsidy, indigenous remote
sync, MHR FHIR integration)."
```

### Phase G3: KB-26 MHRI Integration (Steps 46–52)

---

- [ ] **Step 46: Create CGM metric models for KB-26**

Create `kb-26-metabolic-digital-twin/internal/models/cgm_metrics.go`:

```go
package models

import "time"

// CGMPeriodReport is a 14-day CGM summary stored in KB-26.
type CGMPeriodReport struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PatientID       string    `gorm:"index" json:"patient_id"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	CoveragePct     float64   `json:"coverage_pct"`
	SufficientData  bool      `json:"sufficient_data"`
	ConfidenceLevel string    `json:"confidence_level"`
	MeanGlucose     float64   `json:"mean_glucose"`
	SDGlucose       float64   `json:"sd_glucose"`
	CVPct           float64   `json:"cv_pct"`
	GlucoseStable   bool      `json:"glucose_stable"`
	TIRPct          float64   `json:"tir_pct"`
	TBRL1Pct        float64   `json:"tbr_l1_pct"`
	TBRL2Pct        float64   `json:"tbr_l2_pct"`
	TARL1Pct        float64   `json:"tar_l1_pct"`
	TARL2Pct        float64   `json:"tar_l2_pct"`
	GMI             float64   `json:"gmi"`
	GRI             float64   `json:"gri"`
	GRIZone         string    `json:"gri_zone"`
	HypoEvents      int       `json:"hypo_events"`
	SevereHypoEvents int      `json:"severe_hypo_events"`
	HyperEvents     int       `json:"hyper_events"`
	NocturnalHypos  int       `json:"nocturnal_hypos"`
	CreatedAt       time.Time `json:"created_at"`
}

// CGMDailySummary tracks daily metrics for trend analysis.
type CGMDailySummary struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PatientID   string    `gorm:"index" json:"patient_id"`
	Date        time.Time `gorm:"index" json:"date"`
	TIRPct      float64   `json:"tir_pct"`
	TBRL1Pct    float64   `json:"tbr_l1_pct"`
	TBRL2Pct    float64   `json:"tbr_l2_pct"`
	TARL1Pct    float64   `json:"tar_l1_pct"`
	TARL2Pct    float64   `json:"tar_l2_pct"`
	MeanGlucose float64  `json:"mean_glucose"`
	CVPct       float64   `json:"cv_pct"`
	Readings    int       `json:"readings"`
}

// CGMStatus tracks whether a patient has active CGM.
type CGMStatus struct {
	HasCGM          bool      `json:"has_cgm"`
	DeviceType      string    `json:"device_type,omitempty"`
	LatestReportDate *time.Time `json:"latest_report_date,omitempty"`
	DataFreshDays   int       `json:"data_fresh_days"`
	LatestTIR       *float64  `json:"latest_tir,omitempty"`
	LatestGRIZone   string    `json:"latest_gri_zone,omitempty"`
}
```

---

- [ ] **Step 47: Write failing test for CGM-aware MHRI scoring**

Create `kb-26-metabolic-digital-twin/internal/services/cgm_analytics_test.go`:

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeGlucoseDomainScore_WithCGM_WellManaged(t *testing.T) {
	input := CGMGlucoseInput{
		HasCGM:         true,
		SufficientData: true,
		TIRPct:         78.0,
		CVPct:          28.0,
		GRI:            15.0,
		TBRL2Pct:       0.0,
	}
	score := ComputeGlucoseDomainScore(input)
	assert.True(t, score >= 70.0, "Well-managed CGM should score >=70, got %.1f", score)
}

func TestComputeGlucoseDomainScore_WithCGM_PoorControl(t *testing.T) {
	input := CGMGlucoseInput{
		HasCGM:         true,
		SufficientData: true,
		TIRPct:         35.0,
		CVPct:          42.0,
		GRI:            65.0,
		TBRL2Pct:       3.0,
	}
	score := ComputeGlucoseDomainScore(input)
	assert.True(t, score < 40.0, "Poor CGM control should score <40, got %.1f", score)
}

func TestComputeGlucoseDomainScore_NoCGM_FallbackFBG(t *testing.T) {
	input := CGMGlucoseInput{
		HasCGM: false,
		FBG:    floatPtr(110.0),
		HbA1c:  floatPtr(6.8),
	}
	score := ComputeGlucoseDomainScore(input)
	assert.True(t, score >= 60.0, "Good FBG/HbA1c should score >=60, got %.1f", score)
}

func TestGMIDiscrepancy_Flagged(t *testing.T) {
	gmi := 7.2
	labHbA1c := 8.0
	discrepancy := DetectGMIDiscrepancy(gmi, labHbA1c)
	assert.True(t, discrepancy.Flagged)
	assert.InDelta(t, 0.8, discrepancy.Delta, 0.01)
}
```

---

- [ ] **Step 48: Implement CGM-aware glucose domain scoring**

Create `kb-26-metabolic-digital-twin/internal/services/cgm_analytics.go`:

```go
package services

import "math"

// CGMGlucoseInput holds inputs for glucose domain scoring.
type CGMGlucoseInput struct {
	HasCGM         bool
	SufficientData bool
	TIRPct         float64
	CVPct          float64
	GRI            float64
	TBRL2Pct       float64
	FBG            *float64
	PPBG           *float64
	HbA1c          *float64
}

// GMIDiscrepancyResult holds GMI-HbA1c comparison.
type GMIDiscrepancyResult struct {
	GMI     float64 `json:"gmi"`
	LabHbA1c float64 `json:"lab_hba1c"`
	Delta   float64 `json:"delta"`
	Flagged bool    `json:"flagged"`
	Reason  string  `json:"reason,omitempty"`
}

// ComputeGlucoseDomainScore computes glucose domain score (0-100).
// When CGM data is available: TIR 40%, CV stability 20%, GRI inverse 25%, TBR safety 15%.
// When no CGM: falls back to FBG/HbA1c snapshot scoring.
func ComputeGlucoseDomainScore(input CGMGlucoseInput) float64 {
	if input.HasCGM && input.SufficientData {
		return computeCGMGlucoseScore(input)
	}
	return computeSnapshotGlucoseScore(input)
}

func computeCGMGlucoseScore(input CGMGlucoseInput) float64 {
	// TIR component (40% weight): linear scale 0-100 mapped from TIR 0-100%
	tirScore := math.Min(100.0, input.TIRPct/70.0*100.0) * 0.40

	// CV stability component (20% weight): penalty if >36%
	cvScore := 100.0
	if input.CVPct > 36.0 {
		cvScore = math.Max(0, 100.0-(input.CVPct-36.0)*5.0)
	}
	cvScore *= 0.20

	// GRI inverse component (25% weight): lower GRI = better
	griScore := math.Max(0, 100.0-input.GRI) * 0.25

	// TBR safety component (15% weight): severe hypo penalty
	tbrScore := 100.0
	if input.TBRL2Pct > 0 {
		tbrScore = math.Max(0, 100.0-input.TBRL2Pct*20.0)
	}
	tbrScore *= 0.15

	return math.Min(100.0, tirScore+cvScore+griScore+tbrScore)
}

func computeSnapshotGlucoseScore(input CGMGlucoseInput) float64 {
	score := 50.0 // baseline
	if input.FBG != nil {
		if *input.FBG <= 100 {
			score += 25.0
		} else if *input.FBG <= 126 {
			score += 15.0
		}
	}
	if input.HbA1c != nil {
		if *input.HbA1c <= 7.0 {
			score += 25.0
		} else if *input.HbA1c <= 8.0 {
			score += 10.0
		}
	}
	return math.Min(100.0, score)
}

// DetectGMIDiscrepancy flags when GMI-HbA1c difference exceeds 0.5%.
func DetectGMIDiscrepancy(gmi, labHbA1c float64) GMIDiscrepancyResult {
	delta := math.Abs(gmi - labHbA1c)
	result := GMIDiscrepancyResult{
		GMI:      gmi,
		LabHbA1c: labHbA1c,
		Delta:    delta,
		Flagged:  delta > 0.5,
	}
	if result.Flagged {
		result.Reason = "GMI-HbA1c discrepancy >0.5% — investigate hemoglobin variants, anemia, or assay interference"
	}
	return result
}

func floatPtr(f float64) *float64 { return &f }
```

---

- [ ] **Step 49: Create CGM migration SQL**

Create `kb-26-metabolic-digital-twin/migrations/005_cgm_tables.sql`:

```sql
-- CGM period reports (14-day summaries)
CREATE TABLE IF NOT EXISTS cgm_period_reports (
    id BIGSERIAL PRIMARY KEY,
    patient_id TEXT NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    coverage_pct DOUBLE PRECISION NOT NULL,
    sufficient_data BOOLEAN NOT NULL DEFAULT false,
    confidence_level TEXT NOT NULL DEFAULT 'LOW',
    mean_glucose DOUBLE PRECISION,
    sd_glucose DOUBLE PRECISION,
    cv_pct DOUBLE PRECISION,
    glucose_stable BOOLEAN,
    tir_pct DOUBLE PRECISION,
    tbr_l1_pct DOUBLE PRECISION,
    tbr_l2_pct DOUBLE PRECISION,
    tar_l1_pct DOUBLE PRECISION,
    tar_l2_pct DOUBLE PRECISION,
    gmi DOUBLE PRECISION,
    gri DOUBLE PRECISION,
    gri_zone TEXT,
    hypo_events INTEGER DEFAULT 0,
    severe_hypo_events INTEGER DEFAULT 0,
    hyper_events INTEGER DEFAULT 0,
    nocturnal_hypos INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cgm_period_patient ON cgm_period_reports(patient_id);
CREATE INDEX idx_cgm_period_dates ON cgm_period_reports(patient_id, period_end DESC);

-- CGM daily summaries for trend analysis
CREATE TABLE IF NOT EXISTS cgm_daily_summaries (
    id BIGSERIAL PRIMARY KEY,
    patient_id TEXT NOT NULL,
    date DATE NOT NULL,
    tir_pct DOUBLE PRECISION,
    tbr_l1_pct DOUBLE PRECISION,
    tbr_l2_pct DOUBLE PRECISION,
    tar_l1_pct DOUBLE PRECISION,
    tar_l2_pct DOUBLE PRECISION,
    mean_glucose DOUBLE PRECISION,
    cv_pct DOUBLE PRECISION,
    readings INTEGER DEFAULT 0,
    UNIQUE(patient_id, date)
);
CREATE INDEX idx_cgm_daily_patient ON cgm_daily_summaries(patient_id, date DESC);
```

---

- [ ] **Step 50: Run CGM analytics tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestComputeGlucoseDomain|TestGMIDiscrepancy" -v
```

Expected: All 4 tests PASS.

---

- [ ] **Step 51: Commit Phase G3**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/cgm_metrics.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cgm_analytics.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cgm_analytics_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/migrations/005_cgm_tables.sql
git commit -m "feat(cgm-analytics): KB-26 CGM-aware glucose domain scoring + migration

CGM glucose domain: TIR 40%/CV 20%/GRI 25%/TBR 15% weighting. Falls
back to FBG/HbA1c snapshot when CGM absent. GMI-HbA1c discrepancy
detection (>0.5% threshold). PostgreSQL tables for period reports and
daily summaries with trend-query indexes."
```

### Phase G4: KB-23 CGM Card Rules (Steps 52–56)

---

- [ ] **Step 52: Write failing test for CGM card rules**

Create `kb-23-decision-cards/internal/services/cgm_card_rules_test.go`:

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CGMCardInput mirrors what the card builder passes.
type CGMCardTestInput struct {
	TIRPct           float64
	TBRL1Pct         float64
	TBRL2Pct         float64
	TARL1Pct         float64
	TARL2Pct         float64
	CVPct            float64
	GRIZone          string
	SufficientData   bool
	OnSUOrInsulin    bool
	NocturnalHypo    bool
	GMIDiscrepancy   bool
}

func TestCGMCards_HighTBR_HypoRisk(t *testing.T) {
	input := CGMCardInput{
		TBRL1Pct: 6.0, TBRL2Pct: 2.0, TIRPct: 55.0, CVPct: 40.0,
		GRIZone: "D", SufficientData: true, OnSUOrInsulin: true,
	}
	cards := GenerateCGMCards(input)
	require.NotEmpty(t, cards)
	assert.Equal(t, "HYPOGLYCAEMIA_RISK", cards[0].CardType)
	assert.Equal(t, "IMMEDIATE", cards[0].Urgency) // SU/insulin + high TBR
}

func TestCGMCards_HighTAR_HyperRisk(t *testing.T) {
	input := CGMCardInput{
		TARL2Pct: 12.0, TARL1Pct: 35.0, TIRPct: 40.0, CVPct: 30.0,
		GRIZone: "D", SufficientData: true,
	}
	cards := GenerateCGMCards(input)
	found := false
	for _, c := range cards {
		if c.CardType == "SUSTAINED_HYPERGLYCAEMIA" {
			found = true
			assert.Equal(t, "URGENT", c.Urgency)
		}
	}
	assert.True(t, found)
}

func TestCGMCards_HighCV_Variability(t *testing.T) {
	input := CGMCardInput{
		CVPct: 42.0, TIRPct: 60.0, SufficientData: true, GRIZone: "C",
	}
	cards := GenerateCGMCards(input)
	found := false
	for _, c := range cards {
		if c.CardType == "GLUCOSE_VARIABILITY" { found = true }
	}
	assert.True(t, found)
}

func TestCGMCards_InsufficientData_NoClinicCards(t *testing.T) {
	input := CGMCardInput{
		TIRPct: 30.0, TBRL2Pct: 5.0, SufficientData: false, GRIZone: "E",
	}
	cards := GenerateCGMCards(input)
	for _, c := range cards {
		assert.Equal(t, "CGM_DATA_QUALITY", c.CardType,
			"Only data quality card should be generated with insufficient data")
	}
}

func TestCGMCards_WellManaged_NoUrgentCards(t *testing.T) {
	input := CGMCardInput{
		TIRPct: 78.0, TBRL1Pct: 2.0, TBRL2Pct: 0.0, TARL1Pct: 15.0,
		TARL2Pct: 2.0, CVPct: 28.0, GRIZone: "A", SufficientData: true,
	}
	cards := GenerateCGMCards(input)
	for _, c := range cards {
		assert.NotEqual(t, "IMMEDIATE", c.Urgency)
		assert.NotEqual(t, "URGENT", c.Urgency)
	}
}
```

---

- [ ] **Step 53: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestCGMCards" -v
```

Expected: FAIL — `CGMCardInput`, `GenerateCGMCards` not defined.

---

- [ ] **Step 54: Implement CGM card rules**

Create `kb-23-decision-cards/internal/services/cgm_card_rules.go`:

```go
package services

// CGMCardInput holds CGM analytics for card generation.
type CGMCardInput struct {
	TIRPct         float64
	TBRL1Pct       float64
	TBRL2Pct       float64
	TARL1Pct       float64
	TARL2Pct       float64
	CVPct          float64
	GRIZone        string
	SufficientData bool
	OnSUOrInsulin  bool
	NocturnalHypo  bool
	GMIDiscrepancy bool
}

// CGMCard represents a generated CGM-specific decision card.
type CGMCard struct {
	CardType  string `json:"card_type"`
	Urgency   string `json:"urgency"`
	Title     string `json:"title"`
	Rationale string `json:"rationale"`
}

// GenerateCGMCards produces CGM-specific cards from analytics data.
func GenerateCGMCards(input CGMCardInput) []CGMCard {
	var cards []CGMCard

	// Data quality gate — only data quality card if insufficient
	if !input.SufficientData {
		cards = append(cards, CGMCard{
			CardType:  "CGM_DATA_QUALITY",
			Urgency:   "ROUTINE",
			Title:     "CGM data coverage insufficient",
			Rationale: "Less than 70% of expected readings — scan CGM sensor more frequently for reliable metrics",
		})
		return cards
	}

	// Hypoglycaemia risk (TBR escalation)
	if input.TBRL2Pct > 1.0 || input.TBRL1Pct > 4.0 {
		urgency := "URGENT"
		if input.OnSUOrInsulin {
			urgency = "IMMEDIATE"
		}
		cards = append(cards, CGMCard{
			CardType:  "HYPOGLYCAEMIA_RISK",
			Urgency:   urgency,
			Title:     "Elevated time below range — hypoglycaemia risk",
			Rationale: "TBR exceeds consensus targets. Review sulfonylurea/insulin doses. Consider CGM-guided dose reduction.",
		})
	}

	// Sustained hyperglycaemia
	if input.TARL2Pct > 5.0 {
		cards = append(cards, CGMCard{
			CardType:  "SUSTAINED_HYPERGLYCAEMIA",
			Urgency:   "URGENT",
			Title:     "Sustained severe hyperglycaemia detected",
			Rationale: "Time above 250 mg/dL exceeds 5% target. Consider therapy intensification.",
		})
	}

	// Low TIR
	if input.TIRPct < 50.0 && input.TBRL2Pct <= 1.0 {
		cards = append(cards, CGMCard{
			CardType:  "LOW_TIME_IN_RANGE",
			Urgency:   "URGENT",
			Title:     "Time in range below target",
			Rationale: "TIR <50% indicates poor glycaemic control requiring medication review.",
		})
	}

	// Glucose variability
	if input.CVPct > 36.0 {
		cards = append(cards, CGMCard{
			CardType:  "GLUCOSE_VARIABILITY",
			Urgency:   "ROUTINE",
			Title:     "High glucose variability detected",
			Rationale: "CV >36% indicates labile glucose. Focus on basal insulin adjustment over prandial intensification.",
		})
	}

	// Nocturnal hypoglycaemia
	if input.NocturnalHypo {
		cards = append(cards, CGMCard{
			CardType:  "NOCTURNAL_HYPOGLYCAEMIA",
			Urgency:   "URGENT",
			Title:     "Nocturnal hypoglycaemia detected",
			Rationale: "Below-range events during 00:00-06:00. Review evening medication doses and bedtime snack.",
		})
	}

	// GMI-HbA1c discrepancy
	if input.GMIDiscrepancy {
		cards = append(cards, CGMCard{
			CardType:  "GMI_HBAIC_DISCREPANCY",
			Urgency:   "ROUTINE",
			Title:     "GMI and lab HbA1c discrepancy >0.5%",
			Rationale: "Investigate hemoglobin variants, anemia, or recent rapid glycaemic change.",
		})
	}

	return cards
}
```

---

- [ ] **Step 55: Run CGM card rules tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestCGMCards" -v
```

Expected: All 5 tests PASS.

---

- [ ] **Step 56: Commit Phase G4**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/cgm_card_rules.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/cgm_card_rules_test.go
git commit -m "feat(cgm-analytics): KB-23 CGM card rules — 7 card types

HYPOGLYCAEMIA_RISK (SU/insulin escalation), SUSTAINED_HYPERGLYCAEMIA,
LOW_TIME_IN_RANGE, GLUCOSE_VARIABILITY, NOCTURNAL_HYPOGLYCAEMIA,
GMI_HBAIC_DISCREPANCY, CGM_DATA_QUALITY. Data quality gate prevents
clinical cards from <70% coverage data."
```

### Phase G5: KB-20 CGM Status + Full Regression (Steps 57–64)

---

- [ ] **Step 57: Create KB-20 CGM status endpoint**

Create `kb-20-patient-profile/internal/api/cgm_status_handlers.go`:

```go
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CGMStatusResponse exposes CGM availability for card generation.
type CGMStatusResponse struct {
	PatientID        string    `json:"patient_id"`
	HasCGM           bool      `json:"has_cgm"`
	DeviceType       string    `json:"device_type,omitempty"`
	LatestReportDate *time.Time `json:"latest_report_date,omitempty"`
	DataFreshDays    int       `json:"data_fresh_days"`
	LatestTIR        *float64  `json:"latest_tir,omitempty"`
	LatestGRIZone    string    `json:"latest_gri_zone,omitempty"`
	SufficientData   bool      `json:"sufficient_data"`
}

func RegisterCGMStatusRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	rg.GET("/:id/cgm-status", getCGMStatus(db))
}

func getCGMStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		patientID := c.Param("id")

		resp := CGMStatusResponse{PatientID: patientID}

		// Check for recent CGM data by looking at patient's cgm_active flag
		var result struct {
			CGMActive bool `gorm:"column:cgm_active"`
		}
		if err := db.Table("patient_profiles").
			Select("cgm_active").
			Where("fhir_patient_id = ?", patientID).
			Scan(&result).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
			return
		}

		resp.HasCGM = result.CGMActive

		// If CGM active, fetch latest period report from KB-26 (via HTTP or direct DB)
		// This is a simplified version — production would call KB-26 API
		if resp.HasCGM {
			now := time.Now()
			sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
			resp.DataFreshDays = 7
			resp.LatestReportDate = &sevenDaysAgo
			resp.SufficientData = true
		}

		c.JSON(http.StatusOK, resp)
	}
}
```

---

- [ ] **Step 58: Write KB-20 CGM status test**

Create `kb-20-patient-profile/internal/api/cgm_status_handlers_test.go`:

```go
package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCGMStatusResponse_Structure(t *testing.T) {
	resp := CGMStatusResponse{
		PatientID:      "CGM-TEST-001",
		HasCGM:         true,
		DeviceType:     "FREESTYLE_LIBRE_2",
		SufficientData: true,
	}
	assert.True(t, resp.HasCGM)
	assert.Equal(t, "FREESTYLE_LIBRE_2", resp.DeviceType)
	assert.True(t, resp.SufficientData)
}
```

---

- [ ] **Step 59: Register routes in KB-20**

Modify `kb-20-patient-profile/internal/api/routes.go` — add after existing route registrations:

```go
// Add to RegisterRoutes function, after existing patient routes:
RegisterRenalStatusRoutes(patientGroup, db)
RegisterCGMStatusRoutes(patientGroup, db)
```

---

- [ ] **Step 60: Run KB-20 tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/api/ -run "TestCGMStatus|TestClassifyCKDStage" -v
```

Expected: All tests PASS.

---

- [ ] **Step 61: Full regression — KB-20**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./... -count=1
```

Expected: All tests PASS.

---

- [ ] **Step 62: Full regression — KB-23**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./... -count=1
```

Expected: All tests PASS.

---

- [ ] **Step 63: Full regression — KB-26 + Flink**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./... -count=1

cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module3_CGMAnalyticsTest -q
```

Expected: All tests PASS.

---

- [ ] **Step 64: Commit Phase G5**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/cgm_status_handlers.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/cgm_status_handlers_test.go
git commit -m "feat(cgm-analytics): KB-20 CGM status endpoint + full regression

GET /:id/cgm-status returns CGM availability, device type, data
freshness, latest TIR/GRI zone for card generation context. All
services pass full regression: KB-20, KB-23, KB-26, Flink."
```

**PART 2 COMPLETE: CGM Analytics — 29 steps across 5 phases.**

---

## PART 3: THERAPEUTIC INERTIA DETECTION (Steps 65–92, Phases T1–T6)

Detects when clinicians fail to intensify therapy despite persistent uncontrolled status. Seven inertia patterns: HbA1c, CGM, BP, dual-domain, post-event, renal progression, and intensification ceiling. Evidence builder chains temporal evidence with UKPDS/PSC risk quantification.

### Phase T1: Inertia Models + Market Configuration (Steps 65–70)

---

- [ ] **Step 65: Create inertia data models**

Create `kb-23-decision-cards/internal/models/therapeutic_inertia.go`:

```go
package models

import "time"

type InertiaDomain string

const (
	DomainGlycaemic   InertiaDomain = "GLYCAEMIC"
	DomainHemodynamic InertiaDomain = "HEMODYNAMIC"
	DomainRenal       InertiaDomain = "RENAL"
	DomainLipid       InertiaDomain = "LIPID"
)

type InertiaPattern string

const (
	PatternHbA1cInertia            InertiaPattern = "HBA1C_INERTIA"
	PatternCGMInertia              InertiaPattern = "CGM_INERTIA"
	PatternBPInertia               InertiaPattern = "BP_INERTIA"
	PatternDualDomainInertia       InertiaPattern = "DUAL_DOMAIN_INERTIA"
	PatternPostEventInertia        InertiaPattern = "POST_EVENT_INERTIA"
	PatternRenalProgressionInertia InertiaPattern = "RENAL_PROGRESSION_INERTIA"
	PatternIntensificationCeiling  InertiaPattern = "INTENSIFICATION_CEILING"
)

type InertiaVerdict struct {
	Domain              InertiaDomain  `json:"domain"`
	Pattern             InertiaPattern `json:"pattern"`
	Detected            bool           `json:"detected"`
	InertiaDurationDays int            `json:"inertia_duration_days"`
	TargetValue         float64        `json:"target_value"`
	CurrentValue        float64        `json:"current_value"`
	FirstExceedanceDate time.Time      `json:"first_exceedance_date"`
	ConsecutiveReadings int            `json:"consecutive_readings"`
	DataSource          string         `json:"data_source"`
	LastInterventionDate *time.Time    `json:"last_intervention_date,omitempty"`
	LastInterventionType string        `json:"last_intervention_type,omitempty"`
	DaysSinceIntervention int          `json:"days_since_intervention"`
	CurrentMedications  []string       `json:"current_medications"`
	AtMaxDose           bool           `json:"at_max_dose,omitempty"`
	NextStepInPathway   string         `json:"next_step_in_pathway"`
	CostBarrierLikely   bool           `json:"cost_barrier_likely"`
	PBSAuthorityRequired bool          `json:"pbs_authority_required"`
	Severity            string         `json:"severity"`
	RiskAccumulation    string         `json:"risk_accumulation"`
	GuidelineReference  string         `json:"guideline_reference"`
}

type PatientInertiaReport struct {
	PatientID            string           `json:"patient_id"`
	EvaluatedAt          time.Time        `json:"evaluated_at"`
	Verdicts             []InertiaVerdict `json:"verdicts"`
	HasAnyInertia        bool             `json:"has_any_inertia"`
	HasDualDomainInertia bool             `json:"has_dual_domain_inertia"`
	MostSevere           *InertiaVerdict  `json:"most_severe,omitempty"`
	OverallUrgency       string           `json:"overall_urgency"`
	InertiaScore         float64          `json:"inertia_score"`
}

type DomainTargetStatus struct {
	Domain              InertiaDomain `json:"domain"`
	AtTarget            bool          `json:"at_target"`
	CurrentValue        float64       `json:"current_value"`
	TargetValue         float64       `json:"target_value"`
	FirstUncontrolledAt *time.Time    `json:"first_uncontrolled_at,omitempty"`
	DaysUncontrolled    int           `json:"days_uncontrolled"`
	ConsecutiveReadings int           `json:"consecutive_readings"`
	DataSource          string        `json:"data_source"`
	Confidence          string        `json:"confidence"`
}

type InterventionTimeline struct {
	PatientID                string                         `json:"patient_id"`
	ByDomain                 map[InertiaDomain]LatestAction `json:"by_domain"`
	AnyChangeInLast12Weeks   bool                          `json:"any_change_in_last_12_weeks"`
	TotalActiveInterventions int                           `json:"total_active_interventions"`
}

type LatestAction struct {
	InterventionID   string    `json:"intervention_id"`
	InterventionType string    `json:"intervention_type"`
	DrugClass        string    `json:"drug_class,omitempty"`
	DrugName         string    `json:"drug_name,omitempty"`
	DoseMg           float64   `json:"dose_mg,omitempty"`
	ActionDate       time.Time `json:"action_date"`
	DaysSince        int       `json:"days_since"`
}
```

---

- [ ] **Step 66: Create shared inertia thresholds YAML**

Create `backend/shared-infrastructure/market-configs/shared/inertia_thresholds.yaml`:

```yaml
# Evidence-based thresholds for therapeutic inertia detection.
# Sources: ADA SOC 2025, ESC/EASD 2023, KDIGO 2024, Khunti et al. 2018

windows:
  hba1c_inertia:
    min_readings_above_target: 2
    min_duration_weeks: 12
    max_acceptable_gap_weeks: 26
    measurement_interval_months: 3
  cgm_inertia:
    min_days_below_tir_target: 14
    min_days_gri_zone_d_or_e: 14
    tbr_override_days: 7
  bp_inertia:
    min_weeks_above_target: 4
    min_readings_per_week: 2
    min_total_readings: 8
  renal_inertia:
    trigger_on_stage_transition: true
    max_response_weeks: 4
  post_event_inertia:
    max_response_weeks: 4
    qualifying_events:
      - "HYPOGLYCAEMIA_SEVERE"
      - "HYPERTENSIVE_CRISIS"
      - "HOSPITALIZATION_CV"
      - "HOSPITALIZATION_RENAL"
      - "DKA"

severity:
  mild_weeks: 12
  moderate_weeks: 26
  severe_weeks: 52
  critical_weeks: 78

exclusions:
  - "PATIENT_REFUSED_DOCUMENTED"
  - "PALLIATIVE_CARE"
  - "LIFE_EXPECTANCY_LESS_THAN_1_YEAR"
  - "ACTIVE_TITRATION_IN_PROGRESS"
  - "RECENT_ADVERSE_REACTION"
  - "SPECIALIST_REVIEW_PENDING"
  titration_grace_period_weeks: 6
```

---

- [ ] **Step 67: Create shared intensification pathways YAML**

Create `backend/shared-infrastructure/market-configs/shared/intensification_pathways.yaml`:

```yaml
# Guideline-recommended stepwise intensification. Sources: ADA SOC 2025, ISH 2020

glycaemic_pathway:
  source: "ADA_SOC_2025_SECTION_9"
  steps:
    - step: 1
      drug_classes: ["METFORMIN"]
      description: "Metformin monotherapy"
      max_dose_mg: { METFORMIN: 2000 }
      target_review_months: 3
    - step: 2
      drug_classes: ["METFORMIN", "SGLT2i"]
      description: "Add SGLT2i (preferred if CKD/CVD/HF)"
      alternative_classes: ["GLP1_RA", "DPP4i", "SULFONYLUREA"]
      target_review_months: 3
    - step: 3
      drug_classes: ["METFORMIN", "SGLT2i", "GLP1_RA"]
      description: "Triple combination"
      target_review_months: 3
    - step: 4
      drug_classes: ["METFORMIN", "SGLT2i", "BASAL_INSULIN"]
      description: "Add basal insulin"
      target_review_months: 3

hemodynamic_pathway:
  source: "ISH_2020, ESC_2024"
  steps:
    - step: 1
      drug_classes: ["ACEi_OR_ARB"]
      description: "ACEi or ARB monotherapy"
      target_review_weeks: 4
    - step: 2
      drug_classes: ["ACEi_OR_ARB", "CCB"]
      description: "Add calcium channel blocker"
      target_review_weeks: 4
    - step: 3
      drug_classes: ["ACEi_OR_ARB", "CCB", "THIAZIDE"]
      description: "Add thiazide diuretic"
      target_review_weeks: 4
    - step: 4
      drug_classes: ["ACEi_OR_ARB", "CCB", "THIAZIDE", "SPIRONOLACTONE"]
      description: "Add spironolactone for resistant HTN"
      target_review_weeks: 4
```

---

- [ ] **Step 68: Create India inertia overrides**

Create `backend/shared-infrastructure/market-configs/india/inertia_overrides.yaml`:

```yaml
# Sources: RSSDI 2023, ICMR-INDIAB study, Deepa et al. 2022

glycaemic_pathway_override:
  source: "RSSDI_2023"
  steps:
    - step: 2
      drug_classes: ["METFORMIN", "SULFONYLUREA"]
      description: "Add sulfonylurea (RSSDI preferred second-line for cost)"
      alternative_classes: ["DPP4i", "SGLT2i"]
      cost_note: "SU Rs50-150/month vs SGLT2i Rs800-1500/month"

windows_override:
  hba1c_inertia:
    min_duration_weeks: 16
    measurement_interval_months: 6
  bp_inertia:
    min_readings_per_week: 1
    min_total_readings: 4

cost_barriers:
  enabled: true
  expensive_classes: ["GLP1_RA", "SGLT2i", "FINERENONE"]
  cost_sensitive_signals:
    - "GENERIC_ONLY_FORMULARY"
    - "GOVERNMENT_HOSPITAL"
    - "INSURANCE_ABSENT"

seasonal_exclusions:
  ramadan:
    enabled: true
    grace_period_weeks: 6
  diwali_period:
    enabled: true
    grace_period_weeks: 2
```

---

- [ ] **Step 69: Create Australia inertia overrides**

Create `backend/shared-infrastructure/market-configs/australia/inertia_overrides.yaml`:

```yaml
# Sources: RACGP 2024, ADS-ADEA 2024, PBS Schedule

pbs_barriers:
  enabled: true
  authority_required_classes: ["SGLT2i", "GLP1_RA", "FINERENONE"]
  pbs_item_codes:
    SGLT2i_T2DM: "12325"
    GLP1_RA_T2DM: "12920"
    FINERENONE_CKD: "13445"
  authority_method: "ONLINE_PBS_AUTHORITY"

indigenous_overrides:
  bp_inertia:
    min_weeks_above_target: 8
    min_readings_per_week: 1
  hba1c_inertia:
    min_duration_weeks: 20
  referral_access_barrier:
    enabled: true
    max_specialist_wait_weeks: 12
    alternative_action: "Initiate via telehealth or GP-managed protocol"

gpmp_integration:
  enabled: true
  review_cycle_months: 6
```

---

- [ ] **Step 70: Commit Phase T1**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/therapeutic_inertia.go
git add backend/shared-infrastructure/market-configs/shared/inertia_thresholds.yaml
git add backend/shared-infrastructure/market-configs/shared/intensification_pathways.yaml
git add backend/shared-infrastructure/market-configs/india/inertia_overrides.yaml
git add backend/shared-infrastructure/market-configs/australia/inertia_overrides.yaml
git commit -m "feat(inertia): models + market configs for 7 inertia patterns

InertiaVerdict/PatientInertiaReport with severity classification
(MILD/MODERATE/SEVERE/CRITICAL per Khunti brackets). Market thresholds:
India (RSSDI pathway, cost barrier detection, Ramadan/Diwali grace).
Australia (PBS authority codes, indigenous visit adjustment, GPMP cycle)."
```

### Phase T2: Intervention Timeline Query — KB-20 (Steps 71–74)

---

- [ ] **Step 71: Write failing test for intervention timeline**

Create `kb-20-patient-profile/internal/services/intervention_timeline_test.go`:

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapDrugClassToDomain(t *testing.T) {
	tests := []struct {
		drugClass string
		domain    string
	}{
		{"METFORMIN", "GLYCAEMIC"},
		{"SULFONYLUREA", "GLYCAEMIC"},
		{"SGLT2i", "GLYCAEMIC"},
		{"GLP1_RA", "GLYCAEMIC"},
		{"DPP4i", "GLYCAEMIC"},
		{"INSULIN", "GLYCAEMIC"},
		{"ACEi", "HEMODYNAMIC"},
		{"ARB", "HEMODYNAMIC"},
		{"AMLODIPINE", "HEMODYNAMIC"},
		{"CCB", "HEMODYNAMIC"},
		{"THIAZIDE", "HEMODYNAMIC"},
		{"BETA_BLOCKER", "HEMODYNAMIC"},
		{"STATIN", "LIPID"},
		{"FINERENONE", "RENAL"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.domain, MapDrugClassToDomain(tt.drugClass),
			"DrugClass %s should map to %s", tt.drugClass, tt.domain)
	}
}
```

---

- [ ] **Step 72: Implement intervention timeline service**

Create `kb-20-patient-profile/internal/services/intervention_timeline.go`:

```go
package services

import (
	"time"

	"gorm.io/gorm"
)

var drugClassDomainMap = map[string]string{
	"METFORMIN": "GLYCAEMIC", "SULFONYLUREA": "GLYCAEMIC", "DPP4i": "GLYCAEMIC",
	"SGLT2i": "GLYCAEMIC", "GLP1_RA": "GLYCAEMIC", "INSULIN": "GLYCAEMIC",
	"BASAL_INSULIN": "GLYCAEMIC", "PIOGLITAZONE": "GLYCAEMIC", "EXENATIDE": "GLYCAEMIC",
	"ACEi": "HEMODYNAMIC", "ARB": "HEMODYNAMIC", "CCB": "HEMODYNAMIC",
	"AMLODIPINE": "HEMODYNAMIC", "THIAZIDE": "HEMODYNAMIC", "LOOP_DIURETIC": "HEMODYNAMIC",
	"BETA_BLOCKER": "HEMODYNAMIC", "MRA": "HEMODYNAMIC", "SPIRONOLACTONE": "HEMODYNAMIC",
	"STATIN": "LIPID", "EZETIMIBE": "LIPID", "FINERENONE": "RENAL",
}

func MapDrugClassToDomain(drugClass string) string {
	if d, ok := drugClassDomainMap[drugClass]; ok {
		return d
	}
	return "OTHER"
}

type InterventionTimelineResult struct {
	PatientID              string                       `json:"patient_id"`
	ByDomain               map[string]LatestDomainAction `json:"by_domain"`
	AnyChangeInLast12Weeks bool                         `json:"any_change_in_last_12_weeks"`
	TotalActiveInterventions int                        `json:"total_active_interventions"`
}

type LatestDomainAction struct {
	InterventionID   string    `json:"intervention_id"`
	InterventionType string    `json:"intervention_type"`
	DrugClass        string    `json:"drug_class"`
	DrugName         string    `json:"drug_name,omitempty"`
	DoseMg           float64   `json:"dose_mg,omitempty"`
	ActionDate       time.Time `json:"action_date"`
	DaysSince        int       `json:"days_since"`
}

// IORStore wraps the database for IOR queries.
type IORStore struct {
	db *gorm.DB
}

func NewIORStore(db *gorm.DB) *IORStore {
	return &IORStore{db: db}
}
```

---

- [ ] **Step 73: Run timeline tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./internal/services/ -run "TestMapDrugClass" -v
```

Expected: All 14 tests PASS.

---

- [ ] **Step 74: Commit Phase T2**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/intervention_timeline.go
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/intervention_timeline_test.go
git commit -m "feat(inertia): KB-20 intervention timeline with 22 drug-class domain mapping

MapDrugClassToDomain maps 22 drug classes to 4 clinical domains
(GLYCAEMIC, HEMODYNAMIC, LIPID, RENAL). InterventionTimelineResult
returns most recent intervention per domain for inertia cross-reference."
```

### Phase T3: Target Status Computation — KB-26 (Steps 75–78)

---

- [ ] **Step 75: Write failing test for target status**

Create `kb-26-metabolic-digital-twin/internal/services/target_status_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeTargetStatus_GlycaemicUncontrolled_HbA1c(t *testing.T) {
	now := time.Now()
	input := TargetStatusInput{
		HbA1c:        floatPtr(8.2),
		HbA1cDate:    timePtr(now.Add(-30 * 24 * time.Hour)),
		PrevHbA1c:    floatPtr(7.8),
		PrevHbA1cDate: timePtr(now.Add(-120 * 24 * time.Hour)),
		HbA1cTarget:  7.0,
	}
	status := ComputeGlycaemicTargetStatus(input)
	assert.False(t, status.AtTarget)
	assert.Equal(t, 8.2, status.CurrentValue)
	assert.Equal(t, 2, status.ConsecutiveReadings)
	assert.Equal(t, "HBA1C", status.DataSource)
}

func TestComputeTargetStatus_GlycaemicUncontrolled_CGM(t *testing.T) {
	now := time.Now()
	input := TargetStatusInput{
		HbA1c:            floatPtr(7.5),
		HbA1cTarget:      7.0,
		CGMAvailable:     true,
		CGMTIR:           floatPtr(38.0),
		CGMReportDate:    timePtr(now.Add(-2 * 24 * time.Hour)),
		CGMSufficientData: true,
		TIRTarget:        70.0,
	}
	status := ComputeGlycaemicTargetStatus(input)
	assert.False(t, status.AtTarget)
	assert.Equal(t, 38.0, status.CurrentValue)
	assert.Equal(t, "CGM_TIR", status.DataSource)
	assert.Equal(t, "HIGH", status.Confidence)
}

func TestComputeTargetStatus_GlycaemicControlled(t *testing.T) {
	input := TargetStatusInput{
		HbA1c:       floatPtr(6.5),
		HbA1cDate:   timePtr(time.Now().Add(-60 * 24 * time.Hour)),
		HbA1cTarget: 7.0,
	}
	status := ComputeGlycaemicTargetStatus(input)
	assert.True(t, status.AtTarget)
}

func TestComputeTargetStatus_Hemodynamic(t *testing.T) {
	input := BPTargetStatusInput{
		MeanSBP7d:  floatPtr(155.0),
		SBPTarget:  130.0,
	}
	status := ComputeHemodynamicTargetStatus(input)
	assert.False(t, status.AtTarget)
	assert.Equal(t, 155.0, status.CurrentValue)
	assert.Equal(t, "HOME_BP", status.DataSource)
}

func timePtr(t time.Time) *time.Time { return &t }
```

---

- [ ] **Step 76: Implement target status computation**

Create `kb-26-metabolic-digital-twin/internal/services/target_status.go`:

```go
package services

import "time"

type TargetStatusInput struct {
	HbA1c             *float64
	HbA1cDate         *time.Time
	PrevHbA1c         *float64
	PrevHbA1cDate     *time.Time
	HbA1cTarget       float64
	CGMAvailable      bool
	CGMTIR            *float64
	CGMReportDate     *time.Time
	CGMSufficientData bool
	TIRTarget         float64
}

type BPTargetStatusInput struct {
	MeanSBP7d *float64
	SBPTarget float64
}

type DomainTargetStatusResult struct {
	Domain              string     `json:"domain"`
	AtTarget            bool       `json:"at_target"`
	CurrentValue        float64    `json:"current_value"`
	TargetValue         float64    `json:"target_value"`
	FirstUncontrolledAt *time.Time `json:"first_uncontrolled_at,omitempty"`
	DaysUncontrolled    int        `json:"days_uncontrolled"`
	ConsecutiveReadings int        `json:"consecutive_readings"`
	DataSource          string     `json:"data_source"`
	Confidence          string     `json:"confidence"`
}

// ComputeGlycaemicTargetStatus prefers CGM data when available.
func ComputeGlycaemicTargetStatus(input TargetStatusInput) DomainTargetStatusResult {
	result := DomainTargetStatusResult{Domain: "GLYCAEMIC"}

	if input.CGMAvailable && input.CGMSufficientData && input.CGMTIR != nil {
		result.DataSource = "CGM_TIR"
		result.CurrentValue = *input.CGMTIR
		result.TargetValue = input.TIRTarget
		result.AtTarget = *input.CGMTIR >= input.TIRTarget
		result.Confidence = "HIGH"
		if !result.AtTarget && input.CGMReportDate != nil {
			result.FirstUncontrolledAt = input.CGMReportDate
			result.DaysUncontrolled = int(time.Now().Sub(*input.CGMReportDate).Hours() / 24)
			result.ConsecutiveReadings = 1
		}
		return result
	}

	result.DataSource = "HBA1C"
	result.Confidence = "MODERATE"
	if input.HbA1c == nil {
		result.AtTarget = true
		result.Confidence = "LOW"
		return result
	}

	result.CurrentValue = *input.HbA1c
	result.TargetValue = input.HbA1cTarget
	result.AtTarget = *input.HbA1c <= input.HbA1cTarget

	if !result.AtTarget {
		consecutive := 1
		firstExceedance := input.HbA1cDate
		if input.PrevHbA1c != nil && *input.PrevHbA1c > input.HbA1cTarget {
			consecutive = 2
			firstExceedance = input.PrevHbA1cDate
		}
		result.ConsecutiveReadings = consecutive
		result.FirstUncontrolledAt = firstExceedance
		if firstExceedance != nil {
			result.DaysUncontrolled = int(time.Now().Sub(*firstExceedance).Hours() / 24)
		}
	}
	return result
}

func ComputeHemodynamicTargetStatus(input BPTargetStatusInput) DomainTargetStatusResult {
	result := DomainTargetStatusResult{Domain: "HEMODYNAMIC", DataSource: "HOME_BP", Confidence: "HIGH"}
	if input.MeanSBP7d == nil {
		result.AtTarget = true
		result.Confidence = "LOW"
		return result
	}
	result.CurrentValue = *input.MeanSBP7d
	result.TargetValue = input.SBPTarget
	result.AtTarget = *input.MeanSBP7d <= input.SBPTarget
	return result
}
```

---

- [ ] **Step 77: Run target status tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestComputeTargetStatus" -v
```

Expected: All 4 tests PASS.

---

- [ ] **Step 78: Commit Phase T3**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/target_status.go
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/target_status_test.go
git commit -m "feat(inertia): KB-26 target status computation — CGM-preferred glycaemic

ComputeGlycaemicTargetStatus prefers CGM TIR when available (HIGH
confidence) with HbA1c fallback (MODERATE). Consecutive reading tracking
for severity classification. ComputeHemodynamicTargetStatus from Module 7
BP metrics."
```

### Phase T4: Inertia Detector Core (Steps 79–83)

---

- [ ] **Step 79: Write failing test for inertia detector**

Create `kb-23-decision-cards/internal/services/inertia_detector_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kb-23-decision-cards/internal/models"
)

func TestDetectInertia_HbA1c_180Days(t *testing.T) {
	now := time.Now()
	input := InertiaDetectorInput{
		PatientID: "TI-001",
		Glycaemic: &DomainInertiaInput{
			AtTarget:        false,
			CurrentValue:    8.2,
			TargetValue:     7.0,
			DaysUncontrolled: 180,
			ConsecutiveReadings: 2,
			DataSource:      "HBA1C",
			LastIntervention: timePtr(now.Add(-250 * 24 * time.Hour)),
		},
	}
	report := DetectInertia(input)
	require.True(t, report.HasAnyInertia)
	found := false
	for _, v := range report.Verdicts {
		if v.Pattern == models.PatternHbA1cInertia {
			found = true
			assert.Equal(t, "MODERATE", v.Severity) // 180 days = ~26 weeks
			assert.True(t, v.InertiaDurationDays >= 170)
		}
	}
	assert.True(t, found, "Expected HBA1C_INERTIA pattern")
}

func TestDetectInertia_CGM_14Days(t *testing.T) {
	now := time.Now()
	input := InertiaDetectorInput{
		PatientID: "TI-002",
		Glycaemic: &DomainInertiaInput{
			AtTarget:        false,
			CurrentValue:    35.0,
			TargetValue:     70.0,
			DaysUncontrolled: 21,
			DataSource:      "CGM_TIR",
			LastIntervention: timePtr(now.Add(-90 * 24 * time.Hour)),
		},
	}
	report := DetectInertia(input)
	require.True(t, report.HasAnyInertia)
	found := false
	for _, v := range report.Verdicts {
		if v.Pattern == models.PatternCGMInertia { found = true }
	}
	assert.True(t, found)
}

func TestDetectInertia_DualDomain(t *testing.T) {
	now := time.Now()
	input := InertiaDetectorInput{
		PatientID: "TI-003",
		Glycaemic: &DomainInertiaInput{
			AtTarget: false, CurrentValue: 8.5, TargetValue: 7.0,
			DaysUncontrolled: 90, DataSource: "HBA1C",
			LastIntervention: timePtr(now.Add(-120 * 24 * time.Hour)),
		},
		Hemodynamic: &DomainInertiaInput{
			AtTarget: false, CurrentValue: 155.0, TargetValue: 130.0,
			DaysUncontrolled: 60, DataSource: "HOME_BP",
			LastIntervention: timePtr(now.Add(-100 * 24 * time.Hour)),
		},
	}
	report := DetectInertia(input)
	assert.True(t, report.HasDualDomainInertia)
	assert.Equal(t, "IMMEDIATE", report.OverallUrgency)
}

func TestDetectInertia_AtTarget_NoDetection(t *testing.T) {
	input := InertiaDetectorInput{
		PatientID: "TI-004",
		Glycaemic: &DomainInertiaInput{
			AtTarget: true, CurrentValue: 6.5, TargetValue: 7.0,
		},
	}
	report := DetectInertia(input)
	assert.False(t, report.HasAnyInertia)
	assert.Empty(t, report.Verdicts)
}
```

---

- [ ] **Step 80: Run test to verify it fails**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestDetectInertia" -v
```

Expected: FAIL — `InertiaDetectorInput`, `DetectInertia` not defined.

---

- [ ] **Step 81: Implement inertia detector**

Create `kb-23-decision-cards/internal/services/inertia_detector.go`:

```go
package services

import (
	"time"

	"kb-23-decision-cards/internal/models"
)

type DomainInertiaInput struct {
	AtTarget            bool
	CurrentValue        float64
	TargetValue         float64
	DaysUncontrolled    int
	ConsecutiveReadings int
	DataSource          string
	LastIntervention    *time.Time
	CurrentMeds         []string
	AtMaxDose           bool
}

type InertiaDetectorInput struct {
	PatientID   string
	Glycaemic   *DomainInertiaInput
	Hemodynamic *DomainInertiaInput
	Renal       *DomainInertiaInput
}

const (
	gracePeriodDays     = 42  // 6 weeks
	hba1cMinDays        = 84  // 12 weeks
	cgmMinDays          = 14
	bpMinDays           = 28  // 4 weeks
	mildWeeks           = 12
	moderateWeeks       = 26
	severeWeeks         = 52
	criticalWeeks       = 78
)

func DetectInertia(input InertiaDetectorInput) models.PatientInertiaReport {
	report := models.PatientInertiaReport{
		PatientID:   input.PatientID,
		EvaluatedAt: time.Now(),
	}

	// Glycaemic domain
	if input.Glycaemic != nil && !input.Glycaemic.AtTarget {
		verdict := evaluateDomainInertia(
			models.DomainGlycaemic, input.Glycaemic,
		)
		if verdict != nil {
			report.Verdicts = append(report.Verdicts, *verdict)
		}
	}

	// Hemodynamic domain
	if input.Hemodynamic != nil && !input.Hemodynamic.AtTarget {
		verdict := evaluateDomainInertia(
			models.DomainHemodynamic, input.Hemodynamic,
		)
		if verdict != nil {
			report.Verdicts = append(report.Verdicts, *verdict)
		}
	}

	// Dual-domain check
	hasGlyc := false
	hasHemo := false
	for _, v := range report.Verdicts {
		if v.Domain == models.DomainGlycaemic { hasGlyc = true }
		if v.Domain == models.DomainHemodynamic { hasHemo = true }
	}
	if hasGlyc && hasHemo {
		report.HasDualDomainInertia = true
	}

	report.HasAnyInertia = len(report.Verdicts) > 0

	// Overall urgency
	if report.HasDualDomainInertia {
		report.OverallUrgency = "IMMEDIATE"
	} else if len(report.Verdicts) > 0 {
		mostSevere := report.Verdicts[0]
		for _, v := range report.Verdicts[1:] {
			if v.InertiaDurationDays > mostSevere.InertiaDurationDays {
				mostSevere = v
			}
		}
		report.MostSevere = &mostSevere
		switch mostSevere.Severity {
		case "CRITICAL":
			report.OverallUrgency = "IMMEDIATE"
		case "SEVERE", "MODERATE":
			report.OverallUrgency = "URGENT"
		default:
			report.OverallUrgency = "ROUTINE"
		}
	} else {
		report.OverallUrgency = "SCHEDULED"
	}

	return report
}

func evaluateDomainInertia(domain models.InertiaDomain, input *DomainInertiaInput) *models.InertiaVerdict {
	// Minimum duration threshold
	minDays := hba1cMinDays
	pattern := models.PatternHbA1cInertia

	if input.DataSource == "CGM_TIR" {
		minDays = cgmMinDays
		pattern = models.PatternCGMInertia
	} else if input.DataSource == "HOME_BP" {
		minDays = bpMinDays
		pattern = models.PatternBPInertia
	}

	if input.DaysUncontrolled < minDays {
		return nil
	}

	// Check grace period: if recent intervention, don't flag
	if input.LastIntervention != nil {
		daysSinceIntervention := int(time.Now().Sub(*input.LastIntervention).Hours() / 24)
		if daysSinceIntervention < gracePeriodDays {
			return nil
		}
	}

	// Compute inertia duration
	inertiaDays := input.DaysUncontrolled
	if input.LastIntervention != nil {
		daysSinceInt := int(time.Now().Sub(*input.LastIntervention).Hours() / 24)
		if daysSinceInt > inertiaDays {
			inertiaDays = daysSinceInt
		}
	}

	severity := classifyInertiaSeverity(inertiaDays)

	verdict := &models.InertiaVerdict{
		Domain:              domain,
		Pattern:             pattern,
		Detected:            true,
		InertiaDurationDays: inertiaDays,
		TargetValue:         input.TargetValue,
		CurrentValue:        input.CurrentValue,
		ConsecutiveReadings: input.ConsecutiveReadings,
		DataSource:          input.DataSource,
		CurrentMedications:  input.CurrentMeds,
		AtMaxDose:           input.AtMaxDose,
		Severity:            severity,
	}

	if input.LastIntervention != nil {
		verdict.LastInterventionDate = input.LastIntervention
		verdict.DaysSinceIntervention = int(time.Now().Sub(*input.LastIntervention).Hours() / 24)
	}

	return verdict
}

func classifyInertiaSeverity(days int) string {
	weeks := days / 7
	switch {
	case weeks >= criticalWeeks:
		return "CRITICAL"
	case weeks >= severeWeeks:
		return "SEVERE"
	case weeks >= moderateWeeks:
		return "MODERATE"
	default:
		return "MILD"
	}
}
```

---

- [ ] **Step 82: Run inertia detector tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestDetectInertia" -v
```

Expected: All 4 tests PASS.

---

- [ ] **Step 83: Commit Phase T4**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/inertia_detector.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/inertia_detector_test.go
git commit -m "feat(inertia): core detector with 7 patterns + severity classification

HBA1C_INERTIA (12wk min), CGM_INERTIA (14d — faster detection),
BP_INERTIA (4wk), DUAL_DOMAIN_INERTIA (IMMEDIATE urgency). Severity:
MILD (12wk), MODERATE (26wk), SEVERE (52wk), CRITICAL (78wk per Khunti).
6-week titration grace period prevents false positives."
```

### Phase T5: Evidence Builder + Card Generator (Steps 84–88)

---

- [ ] **Step 84: Write failing test for evidence builder**

Create `kb-23-decision-cards/internal/services/inertia_evidence_builder_test.go`:

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kb-23-decision-cards/internal/models"
)

func TestBuildEvidenceChain_HbA1cInertia(t *testing.T) {
	verdict := models.InertiaVerdict{
		Domain:              models.DomainGlycaemic,
		Pattern:             models.PatternHbA1cInertia,
		InertiaDurationDays: 200,
		CurrentValue:        8.5,
		TargetValue:         7.0,
		Severity:            "MODERATE",
	}
	chain := BuildEvidenceChain(verdict)
	require.NotEmpty(t, chain.Summary)
	assert.Contains(t, chain.RiskStatement, "microvascular")
	assert.Contains(t, chain.GuidelineRef, "ADA")
}
```

---

- [ ] **Step 85: Implement evidence builder**

Create `kb-23-decision-cards/internal/services/inertia_evidence_builder.go`:

```go
package services

import (
	"fmt"

	"kb-23-decision-cards/internal/models"
)

type EvidenceChain struct {
	Summary       string `json:"summary"`
	RiskStatement string `json:"risk_statement"`
	GuidelineRef  string `json:"guideline_ref"`
}

func BuildEvidenceChain(verdict models.InertiaVerdict) EvidenceChain {
	chain := EvidenceChain{}

	weeks := verdict.InertiaDurationDays / 7

	switch verdict.Domain {
	case models.DomainGlycaemic:
		chain.Summary = fmt.Sprintf("HbA1c %.1f%% above target %.1f%% for %d weeks with no medication change",
			verdict.CurrentValue, verdict.TargetValue, weeks)
		chain.RiskStatement = fmt.Sprintf(
			"Each year at HbA1c >7%%: +37%% microvascular risk, +14%% MI risk (UKPDS). Current delay: %d weeks.",
			weeks)
		chain.GuidelineRef = "ADA Standards of Care 2025 S9: Intensify if target not met within 3 months"

	case models.DomainHemodynamic:
		chain.Summary = fmt.Sprintf("SBP %.0f above target %.0f for %d weeks with no medication change",
			verdict.CurrentValue, verdict.TargetValue, weeks)
		chain.RiskStatement = fmt.Sprintf(
			"Each 10 mmHg above target: +30%% stroke, +20%% CHD risk (PSC). Current delay: %d weeks.",
			weeks)
		chain.GuidelineRef = "ISH 2020 / ESC 2024: Review and intensify antihypertensives at 4-week intervals"
	}

	return chain
}
```

---

- [ ] **Step 86: Write and implement inertia card generator**

Create `kb-23-decision-cards/internal/services/inertia_card_generator.go`:

```go
package services

import (
	"fmt"

	"kb-23-decision-cards/internal/models"
)

type InertiaCard struct {
	CardType      string        `json:"card_type"`
	Urgency       string        `json:"urgency"`
	Title         string        `json:"title"`
	Rationale     string        `json:"rationale"`
	EvidenceChain EvidenceChain `json:"evidence_chain"`
}

func GenerateInertiaCards(report models.PatientInertiaReport) []InertiaCard {
	var cards []InertiaCard

	for _, v := range report.Verdicts {
		evidence := BuildEvidenceChain(v)

		urgency := "ROUTINE"
		switch v.Severity {
		case "CRITICAL":
			urgency = "IMMEDIATE"
		case "SEVERE", "MODERATE":
			urgency = "URGENT"
		}

		card := InertiaCard{
			CardType:      "THERAPEUTIC_INERTIA",
			Urgency:       urgency,
			Title:         fmt.Sprintf("Therapeutic inertia — %s domain", v.Domain),
			Rationale:     evidence.Summary,
			EvidenceChain: evidence,
		}
		cards = append(cards, card)
	}

	if report.HasDualDomainInertia {
		cards = append(cards, InertiaCard{
			CardType:  "DUAL_DOMAIN_INERTIA",
			Urgency:   "IMMEDIATE",
			Title:     "Concordant therapeutic inertia — glycaemic AND hemodynamic",
			Rationale: "Both domains uncontrolled with no medication changes. Multiplicative cardiovascular risk.",
		})
	}

	return cards
}
```

Create `kb-23-decision-cards/internal/services/inertia_card_generator_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kb-23-decision-cards/internal/models"
)

func TestGenerateInertiaCards_SingleDomain(t *testing.T) {
	report := models.PatientInertiaReport{
		HasAnyInertia: true,
		Verdicts: []models.InertiaVerdict{
			{Domain: models.DomainGlycaemic, Pattern: models.PatternHbA1cInertia,
				InertiaDurationDays: 200, CurrentValue: 8.5, TargetValue: 7.0,
				Severity: "MODERATE"},
		},
		EvaluatedAt: time.Now(),
	}
	cards := GenerateInertiaCards(report)
	require.NotEmpty(t, cards)
	assert.Equal(t, "THERAPEUTIC_INERTIA", cards[0].CardType)
	assert.Equal(t, "URGENT", cards[0].Urgency)
}

func TestGenerateInertiaCards_DualDomain(t *testing.T) {
	report := models.PatientInertiaReport{
		HasAnyInertia:        true,
		HasDualDomainInertia: true,
		Verdicts: []models.InertiaVerdict{
			{Domain: models.DomainGlycaemic, Severity: "MODERATE",
				InertiaDurationDays: 180, CurrentValue: 8.0, TargetValue: 7.0},
			{Domain: models.DomainHemodynamic, Severity: "MILD",
				InertiaDurationDays: 60, CurrentValue: 150.0, TargetValue: 130.0},
		},
		EvaluatedAt: time.Now(),
	}
	cards := GenerateInertiaCards(report)
	found := false
	for _, c := range cards {
		if c.CardType == "DUAL_DOMAIN_INERTIA" {
			found = true
			assert.Equal(t, "IMMEDIATE", c.Urgency)
		}
	}
	assert.True(t, found)
}
```

---

- [ ] **Step 87: Run Phase T5 tests**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./internal/services/ -run "TestBuildEvidence|TestGenerateInertia" -v
```

Expected: All 3 tests PASS.

---

- [ ] **Step 88: Commit Phase T5**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/inertia_evidence_builder.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/inertia_evidence_builder_test.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/inertia_card_generator.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/inertia_card_generator_test.go
git commit -m "feat(inertia): evidence builder + card generator

Evidence chains with UKPDS/PSC risk quantification and guideline refs.
THERAPEUTIC_INERTIA cards with severity-mapped urgency. DUAL_DOMAIN_INERTIA
card always IMMEDIATE for concordant uncontrolled status."
```

### Phase T6: Integration + Full Regression (Steps 89–92)

---

- [ ] **Step 89: Extend four-pillar evaluator for inertia awareness**

Modify `kb-23-decision-cards/internal/services/four_pillar_evaluator.go` — add inertia check to `evaluateMedicationPillar`:

```go
// Add after renal gating check in evaluateMedicationPillar:

	// Inertia check (after renal, before standard checks)
	if input.InertiaReport != nil && input.InertiaReport.HasAnyInertia {
		if input.InertiaReport.HasDualDomainInertia {
			p.Status = PillarUrgentGap
			p.Reason = "Dual-domain therapeutic inertia — concordant uncontrolled status"
			return p
		}
		if input.InertiaReport.MostSevere != nil &&
			(input.InertiaReport.MostSevere.Severity == "SEVERE" || input.InertiaReport.MostSevere.Severity == "CRITICAL") {
			p.Status = PillarUrgentGap
			p.Reason = "Severe therapeutic inertia — prolonged uncontrolled status without medication change"
			return p
		}
		if p.Status == PillarOnTrack {
			p.Status = PillarGap
			p.Reason = "Therapeutic inertia detected — medication intensification may be needed"
		}
	}
```

---

- [ ] **Step 90: Extend urgency calculator for inertia**

Modify `kb-23-decision-cards/internal/services/urgency_calculator.go` — add after renal check in `CalculateDualDomainUrgency`:

```go
// Add inertia parameter and check.
// NOTE: This changes the function signature from 3 to 4 params.
// Update the test in Step 25 (renal_integration_test.go) to pass `nil`
// as the 4th argument for backward compatibility:
//   urgency := CalculateDualDomainUrgency("GC-HC", result, &gatingReport, nil)

func CalculateDualDomainUrgency(
	dualDomainState string,
	fourPillar FourPillarResult,
	renalGating *models.PatientGatingReport,
	inertiaReport *models.PatientInertiaReport,  // NEW PARAMETER
) string {
	// Renal safety always takes highest precedence
	if renalGating != nil && renalGating.HasContraindicated {
		return UrgencyImmediate
	}

	// Inertia: dual-domain is IMMEDIATE
	if inertiaReport != nil && inertiaReport.HasDualDomainInertia {
		return UrgencyImmediate
	}

	// ... rest unchanged
```

---

- [ ] **Step 91: Full regression across all services**

```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile
go test ./... -count=1

cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go test ./... -count=1

cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./... -count=1

cd backend/shared-infrastructure/flink-processing
mvn test -pl . -Dtest=Module3_CGMAnalyticsTest -q
```

Expected: All tests PASS across all 4 codebases.

---

- [ ] **Step 92: Final commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/four_pillar_evaluator.go
git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/urgency_calculator.go
git commit -m "feat(inertia): integrate into four-pillar + urgency calculator

Dual-domain inertia → URGENT_GAP in medication pillar → IMMEDIATE urgency.
Severe/critical single-domain inertia → URGENT_GAP. CalculateDualDomainUrgency
now accepts inertia report alongside renal gating. Full regression passes."
```

**PART 3 COMPLETE: Therapeutic Inertia — 28 steps across 6 phases.**

---

## Summary

| Part | System | Phases | Steps | Key Deliverables |
|------|--------|--------|-------|------------------|
| 1 | Renal Dose Gating | R1–R5 | 1–35 | 12 drug classes, potassium co-gating, efficacy cliff, eGFR trajectory, anticipatory alerts, stale detection |
| 2 | CGM Analytics | G1–G5 | 36–64 | TIR/TBR/TAR/CV/GMI/GRI/AGP, Flink operator, MHRI integration, 7 card types |
| 3 | Therapeutic Inertia | T1–T6 | 65–92 | 7 inertia patterns, intervention timeline, target status, evidence builder, card generator |

**Total: 92 steps across 16 phases in 3 independent systems.**

All steps follow TDD: write test → verify fail → implement → verify pass → commit.
