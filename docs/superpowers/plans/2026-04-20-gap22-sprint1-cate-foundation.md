# Gap 22: Prescriptive AI — Implementation Plan (Sprint 1: CATE Foundation)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the CATE (Conditional Average Treatment Effect) foundation for Gap 22's prescriptive layer: codify the HCF CHF and Aged Care intervention taxonomies, implement a rule-based baseline CATE learner that produces per-patient × per-intervention treatment-effect estimates with uncertainty, enforce the overlap-positivity diagnostic as a hard guard, and close the loop with Gap 21's attribution verdicts via a CATE-calibration monitor. Every CATE run lands in the Gap 21 governance ledger.

**Architecture:** Intervention taxonomies + CATE engine + calibration monitor all live in [kb-26-metabolic-digital-twin](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/), next to Gap 20's `PredictedRisk` and Gap 21's `AttributionVerdict` / `rule_attribution.go` / `append_only_ledger.go`. The Sprint 1 learner is a **deterministic difference-in-means T-style estimator** computed on the Gap 21 consolidated-record cohort, with bootstrap CI and a logistic-regression propensity model for the overlap diagnostic — no LightGBM, no causal forest, no SHAP. The `CATEEstimate` contract is the stable API surface; Sprint 2 will ship the six-learner committee (S/T/X/DR/R + causal forest) inside a new Python service (KB-28 pattern) and swap it in behind this contract. Market config follows the existing `market-configs/shared/*.yaml` convention.

**Tech Stack:** Go 1.21, Gin, GORM, existing KB-26 + KB-23 infrastructure as input sources (consolidated alert records from KB-23, attribution verdicts from KB-26), YAML market config, `math/rand` + `gonum.org/v1/gonum/stat` for bootstrap CI and logistic regression.

---

## Scope

**In scope (Sprint 1 = Phase 1 of the spec, §7.1):**
- Intervention taxonomy for HCF CHF and Aged Care AU pilots (§1.1 of spec).
- `CATEEstimate` record with per-patient, per-intervention point estimate + 90% CI + overlap status + top feature contributions (rule-based, not SHAP).
- Baseline difference-in-means CATE learner with bootstrap CI.
- Propensity model (logistic regression) + overlap-positivity diagnostic (default band 0.05–0.95, YAML-configurable per market).
- Per-cohort primary-learner registry (single entry in Sprint 1, schema ready for committee in Sprint 2).
- CATE calibration monitor that joins prior CATE estimates with Gap 21 attribution verdicts at T4 and fires a governance-ledger event on miscalibration.
- HTTP API: `POST /cate/estimate`, `GET /cate/:id`, `GET /cate/calibration/summary/:cohortId`.
- Every CATE run and every calibration alarm writes a `CATE_ESTIMATE` or `CATE_MISCALIBRATION` entry to the existing Gap 21 append-only ledger.

**Out of scope (Sprint 2+):**
- S/T/X/DR/R meta-learners and causal forest (spec §6.1). Sprint 2, Python service.
- SHAP feature contributions. Sprint 2, Python service.
- Intervention recommender (constraints/capacity/ranking — spec §6.2). Sprint 3.
- Safety layer (CQL/OGSRL guards — spec §6.3). Sprint 3.
- Explanation layer + Gap 18 worklist panel (spec §6.5, §2.4). Sprint 3.
- Digital twin + DTCF validation + policy evaluator (spec §6.4, §3). Sprint 4.
- Policy governance extensions, shadow→canary→promotion, FDA 2026 compliance pack (spec §6.6, §4). Sprint 5.

**Non-goal guardrails:** Sprint 1 **does not** surface CATE estimates to any clinician UI. CATE runs are produced, persisted, and ledgered, but the clinician-facing recommendation surface is Sprint 3. This matches Gap 21 Sprint 1's "attribution computed but not recommending yet" posture.

---

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| `ConsolidatedAlertRecord` | [kb-23 consolidated_alert_record.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/consolidated_alert_record.go) | The pre-alert feature vector + treatment strategy Sprint 1 CATE learner consumes as its training + inference input |
| `OutcomeRecord` | [kb-23 outcome_record.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/outcome_record.go) | T4 outcome labels — CATE learner trains on (features, treatment, outcome) triples |
| `AttributionVerdict` | [kb-26 attribution_verdict.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/attribution_verdict.go) | Post-hoc ground-truth signal the calibration monitor compares the CATE prior estimate against |
| `PredictedRisk` | [kb-26 predicted_risk.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/predicted_risk.go) | Patient-level risk snapshot that scopes recommendation candidacy (CATE is only computed when a Gap 20 alert fires) |
| `LedgerEntry` + `AppendEntry` / `VerifyChain` | [kb-26 append_only_ledger.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/append_only_ledger.go) | CATE runs and calibration alarms append here as new `EntryType` values; chain invariant preserved |
| Market-config YAML pattern | [market-configs/shared/](backend/shared-infrastructure/market-configs/shared/) | Intervention taxonomies + CATE parameters follow `outcome_ingestion_parameters.yaml` / `attribution_parameters.yaml` conventions |
| Gin + GORM + AutoMigrate in `main.go` | [kb-26 main.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/cmd/server/main.go) | Sprint 1 adds `CATEEstimate` + `InterventionDefinition` + `CATEPrimaryLearnerAssignment` to AutoMigrate; extends the route group |

## File Inventory

### KB-26 — CATE models, learner, overlap, calibration
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/intervention.go` | `InterventionDefinition`, `InterventionCategory` enum, `EligibilityCriterion`, `Contraindication` |
| Create | `internal/models/cate_estimate.go` | `CATEEstimate`, `OverlapStatus` enum, `LearnerType` enum, `FeatureContribution`, `CATEPrimaryLearnerAssignment` |
| Create | `internal/services/intervention_registry.go` | `LoadFromYAML`, `ListInterventions(cohort, patient)`, `IsEligible(patient, intervention)` |
| Create | `internal/services/intervention_registry_test.go` | 4 tests (load, filter-by-cohort, contraindication, cool-down) |
| Create | `internal/services/propensity.go` | `FitPropensity(cohort, intervention, trainingSet)`, `PredictPropensity(patient, intervention) float64` — logistic regression via gonum |
| Create | `internal/services/propensity_test.go` | 3 tests (fit convergence, prediction range, separable-data edge) |
| Create | `internal/models/overlap.go` | `OverlapBand` shared type (moved from services so `api` can import it) |
| Create | `internal/services/overlap_diagnostic.go` | `EvaluateOverlap(propensity float64, band models.OverlapBand) models.OverlapStatus` — hard guard |
| Create | `internal/services/overlap_diagnostic_test.go` | 3 tests (inside-band, below-floor, above-ceiling) |
| Create | `internal/services/baseline_cate_learner.go` | `EstimateCATE(patient, intervention, cohort) (CATEEstimate, error)` — difference-in-means T-style with bootstrap CI |
| Create | `internal/services/baseline_cate_learner_test.go` | 5 tests (known-effect recovery, CI width shrinks with N, overlap-fail short-circuits, missing-feature handling, ledger write) |
| Create | `internal/services/cate_calibration_monitor.go` | `ComputeCalibrationSummary(cohortId, horizonDays) CalibrationSummary`, `EvaluateAndAlarm(cohortId)` — joins prior CATE to post-hoc `AttributionVerdict.RiskDifference` |
| Create | `internal/services/cate_calibration_monitor_test.go` | 4 tests (calibrated signal passes, miscalibrated signal fires, empty cohort returns no-op, ledger entry shape) |
| Create | `internal/services/cate_parameters_loader.go` | `LoadCATEParameters`, `BandForCohort`, `CalibrationConfig` — YAML → structs |
| Create | `internal/api/cate_handlers.go` | `POST /cate/estimate`, `GET /cate/:id`, `GET /cate/calibration/summary/:cohortId`, plus `loadTrainingCohort` SQL join helper |
| Create | `internal/api/cate_handlers_test.go` | 2 HTTP round-trip tests (estimate, calibration summary) |
| Modify | `internal/api/routes.go` | Register `/cate` and `/cate/calibration` route groups alongside existing `/attribution` and `/governance` groups |
| Modify | `cmd/server/main.go` | AutoMigrate `InterventionDefinition`, `CATEEstimate`, `CATEPrimaryLearnerAssignment`; load intervention taxonomies + CATE parameters YAML at startup |

### Market configs — intervention taxonomies + CATE parameters
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/intervention_taxonomy_hcf_chf.yaml` | 6 interventions × eligibility × contraindications × cool-down × resource cost |
| Create | `market-configs/shared/intervention_taxonomy_aged_care_au.yaml` | 6 interventions × same schema |
| Create | `market-configs/shared/cate_parameters.yaml` | Overlap band, bootstrap N, miscalibration threshold, per-cohort primary learner (Sprint 1: all cohorts → `BASELINE_DIFF_MEANS`) |

**Total: 17 create (Go source + YAML), 8 new test files, 2 modify**

---

### Task 1: Intervention taxonomy YAML + Go models

**Files:**
- Create: `backend/shared-infrastructure/market-configs/shared/intervention_taxonomy_hcf_chf.yaml`
- Create: `backend/shared-infrastructure/market-configs/shared/intervention_taxonomy_aged_care_au.yaml`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/intervention.go`

The taxonomy is the spec's Step 1.1 — "deceptively important, ripples through every downstream step" (§16). Every CATE estimate is keyed on a `(PatientID, InterventionID)` pair; if the intervention vocabulary is wrong, nothing downstream is recoverable.

- [ ] **Step 1: Write the failing test for the taxonomy model**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/intervention_test.go`:

```go
package models

import (
	"testing"
)

func TestInterventionDefinition_Validate_RejectsMissingID(t *testing.T) {
	def := InterventionDefinition{
		CohortID: "hcf_catalyst_chf",
		Category: string(CategoryFollowUp),
		Name:     "Nurse phone follow-up",
	}
	if err := def.Validate(); err == nil {
		t.Fatal("expected validation error for missing ID")
	}
}

func TestInterventionDefinition_Validate_RejectsUnknownCategory(t *testing.T) {
	def := InterventionDefinition{
		ID:       "nurse_phone_48h",
		CohortID: "hcf_catalyst_chf",
		Category: "NOT_A_REAL_CATEGORY",
		Name:     "Nurse phone follow-up",
	}
	if err := def.Validate(); err == nil {
		t.Fatal("expected validation error for unknown category")
	}
}

func TestInterventionDefinition_Validate_AcceptsWellFormed(t *testing.T) {
	def := InterventionDefinition{
		ID:               "nurse_phone_48h",
		CohortID:         "hcf_catalyst_chf",
		Category:         string(CategoryFollowUp),
		Name:             "Nurse phone follow-up",
		CoolDownHours:    48,
		ResourceCost:     1.0,
		FeatureSignature: []string{"age", "ef_last", "nt_probnp_trend_7d"},
	}
	if err := def.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/models/ -run TestInterventionDefinition -v`
Expected: FAIL with "undefined: InterventionDefinition".

- [ ] **Step 3: Create the intervention model**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/intervention.go`:

```go
package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// InterventionCategory is the coarse taxonomy bucket an intervention falls into.
// Sprint 1 vocabulary covers HCF CHF and Aged Care AU. New categories require a
// market-config + clinical-lead review (spec §9 and §13 item 6).
type InterventionCategory string

const (
	CategoryFollowUp          InterventionCategory = "FOLLOW_UP"           // e.g. nurse phone call, GP visit
	CategorySpecialistReferral InterventionCategory = "SPECIALIST_REFERRAL" // e.g. cardiologist, geriatrician
	CategoryMedicationReview  InterventionCategory = "MEDICATION_REVIEW"   // pharmacist / GP medication review
	CategoryDeviceEnrolment   InterventionCategory = "DEVICE_ENROLMENT"    // device-monitoring onboarding
	CategoryAlliedHealth      InterventionCategory = "ALLIED_HEALTH"       // physio, OT, dietitian
	CategoryCarePlanning      InterventionCategory = "CARE_PLANNING"       // family conference, care-plan revision
)

var validInterventionCategories = map[InterventionCategory]struct{}{
	CategoryFollowUp: {}, CategorySpecialistReferral: {}, CategoryMedicationReview: {},
	CategoryDeviceEnrolment: {}, CategoryAlliedHealth: {}, CategoryCarePlanning: {},
}

// EligibilityCriterion is a single feature-predicate test applied against the patient's
// consolidated pre-alert record. Multiple criteria compose as logical AND.
type EligibilityCriterion struct {
	FeatureKey string  `json:"feature_key"`
	Operator   string  `json:"operator"` // "gte", "lte", "eq", "in"
	Threshold  float64 `json:"threshold,omitempty"`
	Set        []string `json:"set,omitempty"`
}

// Contraindication is a hard disqualifier. If any matches the patient, the intervention
// is not CATE-scored and not recommended.
type Contraindication struct {
	FeatureKey string  `json:"feature_key"`
	Operator   string  `json:"operator"`
	Threshold  float64 `json:"threshold,omitempty"`
	Set        []string `json:"set,omitempty"`
	Reason     string  `json:"reason"`
}

// InterventionDefinition is a versioned, cohort-scoped intervention that the CATE engine
// can score. Persisted to DB on service start from market-config YAML.
type InterventionDefinition struct {
	ID               string                 `gorm:"primaryKey;size:80" json:"id"`
	CohortID         string                 `gorm:"size:60;index;not null" json:"cohort_id"`
	Category         string                 `gorm:"size:40;not null" json:"category"`
	Name             string                 `gorm:"size:200;not null" json:"name"`
	ClinicianLanguage string                `gorm:"size:300" json:"clinician_language"`
	CoolDownHours    int                    `json:"cool_down_hours"`
	ResourceCost     float64                `json:"resource_cost"` // arbitrary-unit, used by capacity optimiser in Sprint 3
	FeatureSignature pq.StringArray         `gorm:"type:text[]" json:"feature_signature"`
	EligibilityJSON  string                 `gorm:"type:text" json:"-"` // serialized []EligibilityCriterion
	ContraindicationsJSON string            `gorm:"type:text" json:"-"` // serialized []Contraindication
	Version          string                 `gorm:"size:20;not null;default:'1.0.0'" json:"version"`
	SourceYAMLPath   string                 `gorm:"size:300" json:"source_yaml_path"`
	LoadedAt         time.Time              `gorm:"autoCreateTime" json:"loaded_at"`
	LedgerEntryID    *uuid.UUID             `gorm:"type:uuid" json:"ledger_entry_id,omitempty"`
}

func (InterventionDefinition) TableName() string { return "intervention_definitions" }

// Validate checks that required fields are present and values are in their enums.
// Called on YAML load and on every CATE request.
func (d InterventionDefinition) Validate() error {
	if d.ID == "" {
		return errors.New("intervention ID required")
	}
	if d.CohortID == "" {
		return errors.New("cohort ID required")
	}
	if _, ok := validInterventionCategories[InterventionCategory(d.Category)]; !ok {
		return errors.New("unknown intervention category: " + d.Category)
	}
	if d.Name == "" {
		return errors.New("intervention name required")
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/models/ -run TestInterventionDefinition -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Create HCF CHF intervention taxonomy YAML**

Create `backend/shared-infrastructure/market-configs/shared/intervention_taxonomy_hcf_chf.yaml`:

```yaml
# Gap 22 Sprint 1 intervention taxonomy for HCF CHF pilot.
# Reviewed with clinical lead: [TO BE SIGNED before first CATE run to production]
# Spec §9.1, §7.1 Step 1.1.
cohort_id: "hcf_catalyst_chf"
version: "1.0.0"
primary_cate_horizon_days: 30
recommendation_cardinality: 1  # N=1 primary + 2 alternatives surfaced on demand

interventions:
  - id: "nurse_phone_48h"
    category: "FOLLOW_UP"
    name: "Nurse phone follow-up within 48 hours of discharge"
    clinician_language: "Nurse call within 48h"
    cool_down_hours: 48
    resource_cost: 1.0
    feature_signature: ["age", "ef_last", "nt_probnp_trend_7d", "days_since_discharge", "weight_trend_7d"]
    eligibility:
      - { feature_key: "days_since_discharge", operator: "lte", threshold: 7 }
    contraindications:
      - { feature_key: "phone_contact_opt_out", operator: "eq", threshold: 1, reason: "patient opted out of phone contact" }

  - id: "gp_visit_7d"
    category: "FOLLOW_UP"
    name: "GP visit within 7 days"
    clinician_language: "GP visit in 7 days"
    cool_down_hours: 168
    resource_cost: 2.0
    feature_signature: ["age", "ef_last", "polypharmacy_score", "days_since_discharge"]
    eligibility:
      - { feature_key: "days_since_discharge", operator: "lte", threshold: 14 }
    contraindications: []

  - id: "cardiology_referral"
    category: "SPECIALIST_REFERRAL"
    name: "Cardiologist referral"
    clinician_language: "Cardiology review"
    cool_down_hours: 720  # 30 days
    resource_cost: 5.0
    feature_signature: ["age", "ef_last", "nt_probnp_trend_7d", "prior_cardiology_30d"]
    eligibility:
      - { feature_key: "prior_cardiology_30d", operator: "eq", threshold: 0 }
    contraindications:
      - { feature_key: "palliative_care_flag", operator: "eq", threshold: 1, reason: "patient on palliative pathway" }

  - id: "pharmacist_medication_review"
    category: "MEDICATION_REVIEW"
    name: "Pharmacist medication review"
    clinician_language: "Pharmacist review"
    cool_down_hours: 720
    resource_cost: 3.0
    feature_signature: ["polypharmacy_score", "recent_med_change_30d", "age"]
    eligibility:
      - { feature_key: "polypharmacy_score", operator: "gte", threshold: 5 }
    contraindications: []

  - id: "device_monitoring_enrolment"
    category: "DEVICE_ENROLMENT"
    name: "Device monitoring enrolment"
    clinician_language: "Home monitoring device"
    cool_down_hours: 17520  # 2 years
    resource_cost: 4.0
    feature_signature: ["age", "device_eligible_flag", "digital_literacy_score"]
    eligibility:
      - { feature_key: "device_eligible_flag", operator: "eq", threshold: 1 }
    contraindications:
      - { feature_key: "cognitive_impairment_flag", operator: "eq", threshold: 1, reason: "unsuitable for self-monitoring device" }

  - id: "nutritionist_consult"
    category: "ALLIED_HEALTH"
    name: "Nutritionist consultation"
    clinician_language: "Dietitian review"
    cool_down_hours: 720
    resource_cost: 2.0
    feature_signature: ["bmi", "fluid_overload_flag", "sodium_intake_est"]
    eligibility:
      - { feature_key: "fluid_overload_flag", operator: "eq", threshold: 1 }
    contraindications: []
```

- [ ] **Step 6: Create Aged Care AU intervention taxonomy YAML**

Create `backend/shared-infrastructure/market-configs/shared/intervention_taxonomy_aged_care_au.yaml`:

```yaml
# Gap 22 Sprint 1 intervention taxonomy for Aged Care AU pilot.
# Reviewed with clinical lead: [TO BE SIGNED before first CATE run to production]
# Spec §9.2, §7.1 Step 1.1.
cohort_id: "aged_care_au"
version: "1.0.0"
primary_cate_horizon_days: 90
recommendation_cardinality: 3  # N=3 multi-modal care is the norm

interventions:
  - id: "geriatrician_review"
    category: "SPECIALIST_REFERRAL"
    name: "Geriatrician review"
    clinician_language: "Geriatrician review"
    cool_down_hours: 2160  # 90 days
    resource_cost: 5.0
    feature_signature: ["age", "frailty_score", "polypharmacy_score", "recent_falls_90d"]
    eligibility:
      - { feature_key: "frailty_score", operator: "gte", threshold: 4 }
    contraindications:
      - { feature_key: "palliative_care_flag", operator: "eq", threshold: 1, reason: "patient on palliative pathway" }

  - id: "pharmacist_medication_review"
    category: "MEDICATION_REVIEW"
    name: "Pharmacist medication review (polypharmacy focus)"
    clinician_language: "Pharmacist review"
    cool_down_hours: 2160
    resource_cost: 3.0
    feature_signature: ["polypharmacy_score", "recent_med_change_30d", "egfr_last"]
    eligibility:
      - { feature_key: "polypharmacy_score", operator: "gte", threshold: 9 }
    contraindications: []

  - id: "allied_health_physio"
    category: "ALLIED_HEALTH"
    name: "Allied health intervention (physio/OT)"
    clinician_language: "Physio / OT"
    cool_down_hours: 336  # 14 days
    resource_cost: 2.0
    feature_signature: ["mobility_score", "recent_falls_90d", "age"]
    eligibility:
      - { feature_key: "mobility_score", operator: "lte", threshold: 3 }
    contraindications:
      - { feature_key: "acute_illness_flag", operator: "eq", threshold: 1, reason: "acute illness contraindicates mobilisation" }

  - id: "care_plan_revision"
    category: "CARE_PLANNING"
    name: "Care plan revision"
    clinician_language: "Update care plan"
    cool_down_hours: 720
    resource_cost: 1.5
    feature_signature: ["care_plan_age_days", "recent_clinical_change_flag"]
    eligibility:
      - { feature_key: "care_plan_age_days", operator: "gte", threshold: 180 }
    contraindications: []

  - id: "family_conference"
    category: "CARE_PLANNING"
    name: "Family conference"
    clinician_language: "Family conference"
    cool_down_hours: 4320  # 180 days
    resource_cost: 2.0
    feature_signature: ["care_plan_age_days", "recent_clinical_change_flag", "advance_care_directive_flag"]
    eligibility:
      - { feature_key: "advance_care_directive_flag", operator: "eq", threshold: 0 }
    contraindications: []

  - id: "gp_home_visit"
    category: "FOLLOW_UP"
    name: "GP home visit"
    clinician_language: "GP home visit"
    cool_down_hours: 336
    resource_cost: 4.0
    feature_signature: ["age", "mobility_score", "days_since_last_gp_visit"]
    eligibility:
      - { feature_key: "mobility_score", operator: "lte", threshold: 2 }
    contraindications: []
```

- [ ] **Step 7: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/intervention.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/intervention_test.go \
        backend/shared-infrastructure/market-configs/shared/intervention_taxonomy_hcf_chf.yaml \
        backend/shared-infrastructure/market-configs/shared/intervention_taxonomy_aged_care_au.yaml
git commit -m "feat(gap22): intervention taxonomy model + HCF CHF / Aged Care YAML (Sprint 1 Task 1)"
```

---

### Task 2: CATE estimate model + primary-learner assignment + market config

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/cate_estimate.go`
- Create: `backend/shared-infrastructure/market-configs/shared/cate_parameters.yaml`

The `CATEEstimate` struct is the stable public contract. Sprint 2's Python learner committee will produce values conforming to the same struct — we design it now so every downstream consumer (recommender, explanation layer, calibration monitor) only ever talks to this shape.

- [ ] **Step 1: Write the failing test for the CATE model**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/cate_estimate_test.go`:

```go
package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestCATEEstimate_IsActionable_RejectsOverlapFailure(t *testing.T) {
	e := CATEEstimate{
		ID:            uuid.New(),
		PatientID:     "P1",
		InterventionID: "nurse_phone_48h",
		OverlapStatus: string(OverlapBelowFloor),
		PointEstimate: 0.15,
		CILower:       0.10,
		CIUpper:       0.20,
	}
	if e.IsActionable() {
		t.Fatal("expected non-actionable when overlap below floor")
	}
}

func TestCATEEstimate_IsActionable_AcceptsPassWithNarrowCI(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: string(OverlapPass),
		PointEstimate: 0.15,
		CILower:       0.12,
		CIUpper:       0.18,
	}
	if !e.IsActionable() {
		t.Fatal("expected actionable when overlap passes and CI narrow")
	}
}

func TestCATEEstimate_ConfidenceLabel_HighNarrowCI(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: string(OverlapPass),
		PointEstimate: 0.15, CILower: 0.13, CIUpper: 0.17,
	}
	if got := e.ConfidenceLabel(); got != ConfidenceHigh {
		t.Fatalf("want HIGH, got %s", got)
	}
}

func TestCATEEstimate_ConfidenceLabel_LowWideCI(t *testing.T) {
	e := CATEEstimate{
		OverlapStatus: string(OverlapPass),
		PointEstimate: 0.15, CILower: -0.05, CIUpper: 0.35,
	}
	if got := e.ConfidenceLabel(); got != ConfidenceLow {
		t.Fatalf("want LOW, got %s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/models/ -run TestCATEEstimate -v`
Expected: FAIL with "undefined: CATEEstimate".

- [ ] **Step 3: Create the CATE estimate model**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/cate_estimate.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// OverlapStatus is the hard-guard outcome of the propensity overlap check.
// Spec §6.1: CATE_INCONCLUSIVE_NO_OVERLAP is returned verbatim when overlap fails.
type OverlapStatus string

const (
	OverlapPass        OverlapStatus = "OVERLAP_PASS"
	OverlapBelowFloor  OverlapStatus = "OVERLAP_BELOW_FLOOR"  // propensity < band[0]
	OverlapAboveCeiling OverlapStatus = "OVERLAP_ABOVE_CEILING" // propensity > band[1]
	OverlapInsufficientData OverlapStatus = "OVERLAP_INSUFFICIENT_DATA"
)

// LearnerType identifies which CATE estimator produced the estimate. Sprint 1 ships
// BASELINE_DIFF_MEANS only; Sprint 2 adds S/T/X/DR/R/CAUSAL_FOREST behind the same
// contract. The enum is the vehicle for per-cohort primary-learner selection (§6.1).
type LearnerType string

const (
	LearnerBaselineDiffMeans LearnerType = "BASELINE_DIFF_MEANS"
	LearnerS                 LearnerType = "S_LEARNER"
	LearnerT                 LearnerType = "T_LEARNER"
	LearnerX                 LearnerType = "X_LEARNER"
	LearnerDR                LearnerType = "DR_LEARNER"
	LearnerR                 LearnerType = "R_LEARNER"
	LearnerCausalForest      LearnerType = "CAUSAL_FOREST"
)

// ConfidenceLabel is the clinician-facing confidence tier derived from CI width +
// overlap status. Populated by ConfidenceLabel() on read.
type ConfidenceLabel string

const (
	ConfidenceHigh   ConfidenceLabel = "HIGH"
	ConfidenceMedium ConfidenceLabel = "MEDIUM"
	ConfidenceLow    ConfidenceLabel = "LOW"
)

// FeatureContribution is one row in the top-K feature attribution table. Sprint 1 uses
// cohort-bucket-membership deltas; Sprint 2 replaces with SHAP without changing the shape.
type FeatureContribution struct {
	FeatureKey    string  `json:"feature_key"`
	Contribution  float64 `json:"contribution"`   // signed: positive pushes CATE up
	PatientValue  float64 `json:"patient_value"`
	CohortMean    float64 `json:"cohort_mean"`
}

// CATEEstimate is the per-patient × per-intervention causal estimate.
// This is the stable Sprint 1 → Sprint N contract. Every downstream consumer reads
// this shape only.
type CATEEstimate struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ConsolidatedRecordID   uuid.UUID  `gorm:"type:uuid;index;not null" json:"consolidated_record_id"`
	PatientID              string     `gorm:"size:100;index;not null" json:"patient_id"`
	CohortID               string     `gorm:"size:60;index;not null" json:"cohort_id"`
	InterventionID         string     `gorm:"size:80;index;not null" json:"intervention_id"`
	LearnerType            string     `gorm:"size:30;not null" json:"learner_type"`

	PointEstimate          float64    `json:"point_estimate"`
	CILower                float64    `json:"ci_lower"`  // 90% CI lower bound
	CIUpper                float64    `json:"ci_upper"`  // 90% CI upper bound
	HorizonDays            int        `json:"horizon_days"`

	// Propensity and overlap
	Propensity             float64    `json:"propensity"`
	OverlapStatus          string     `gorm:"size:40;index;not null" json:"overlap_status"`

	// Cohort context (shown in explanation layer Sprint 3)
	TrainingN              int        `json:"training_n"`           // sample size used
	CohortTreatedN         int        `json:"cohort_treated_n"`
	CohortControlN         int        `json:"cohort_control_n"`

	// Feature contributions (top-K, signed)
	FeatureContributionsJSON string   `gorm:"type:text" json:"-"` // serialized []FeatureContribution
	FeatureContributionKeys  pq.StringArray `gorm:"type:text[]" json:"feature_contribution_keys"` // queryable

	// Provenance
	ModelVersion           string     `gorm:"size:40" json:"model_version"`
	LedgerEntryID          *uuid.UUID `gorm:"type:uuid" json:"ledger_entry_id,omitempty"`
	ComputedAt             time.Time  `gorm:"autoCreateTime;index" json:"computed_at"`
}

func (CATEEstimate) TableName() string { return "cate_estimates" }

// IsActionable is the single short-circuit used by the recommender (Sprint 3).
// Spec §6.1: CATE estimates without overlap pass are never shown, regardless of point value.
func (e CATEEstimate) IsActionable() bool {
	return OverlapStatus(e.OverlapStatus) == OverlapPass
}

// ConfidenceLabel derives the 3-tier label from CI width + overlap status. Spec §6.1.
// Width thresholds are deliberately not per-cohort in Sprint 1; Sprint 3 explanation
// layer will YAML-configure per market.
func (e CATEEstimate) ConfidenceLabel() ConfidenceLabel {
	if OverlapStatus(e.OverlapStatus) != OverlapPass {
		return ConfidenceLow
	}
	width := e.CIUpper - e.CILower
	switch {
	case width <= 0.06:
		return ConfidenceHigh
	case width <= 0.20:
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

// CATEPrimaryLearnerAssignment records which learner is the cohort × intervention × horizon
// primary. Sprint 1 populates once at service start from cate_parameters.yaml; Sprint 2
// makes it an output of the Qini-based selection pipeline (spec §6.1 "per-cohort learner
// selection"). Appended to ledger as CATE_LEARNER_ASSIGNMENT.
type CATEPrimaryLearnerAssignment struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CohortID      string    `gorm:"size:60;uniqueIndex:idx_cohort_intv_horizon,priority:1;not null" json:"cohort_id"`
	InterventionID string   `gorm:"size:80;uniqueIndex:idx_cohort_intv_horizon,priority:2;not null" json:"intervention_id"`
	HorizonDays   int       `gorm:"uniqueIndex:idx_cohort_intv_horizon,priority:3;not null" json:"horizon_days"`
	LearnerType   string    `gorm:"size:30;not null" json:"learner_type"`
	AssignedAt    time.Time `gorm:"autoCreateTime" json:"assigned_at"`
	LedgerEntryID *uuid.UUID `gorm:"type:uuid" json:"ledger_entry_id,omitempty"`
}

func (CATEPrimaryLearnerAssignment) TableName() string { return "cate_primary_learner_assignments" }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/models/ -run TestCATEEstimate -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Create CATE parameters YAML**

Create `backend/shared-infrastructure/market-configs/shared/cate_parameters.yaml`:

```yaml
# Gap 22 Sprint 1 CATE engine parameters. Spec §6.1, §7.1.
version: "1.0.0"

# Overlap-positivity band. Patients with propensity outside the band for a given
# intervention get OVERLAP_BELOW_FLOOR or OVERLAP_ABOVE_CEILING and no point estimate.
# Market override: per-cohort blocks below. Spec §6.1: (0.05, 0.95) default.
overlap_band:
  default: { floor: 0.05, ceiling: 0.95 }
  per_cohort:
    hcf_catalyst_chf: { floor: 0.05, ceiling: 0.95 }
    aged_care_au:    { floor: 0.10, ceiling: 0.90 }  # slightly tighter — smaller cohort, more conservative

# Bootstrap parameters for Sprint 1 baseline learner CI construction.
bootstrap:
  n_resamples: 500
  ci_level: 0.90  # 5th/95th percentile of bootstrap distribution

# Minimum training set size (combined treated + control) below which the baseline
# learner returns OVERLAP_INSUFFICIENT_DATA regardless of individual propensity.
min_training_n: 40

# Calibration monitor: miscalibration alarm fires when rolling 90-day mean of
# |attributed_effect − predicted_CATE| exceeds threshold for a given cohort × intervention.
calibration:
  alarm_threshold_abs_diff: 0.05     # spec §14 HCF CHF target
  rolling_window_days: 90
  min_matched_pairs: 20              # below this, summary returns INSUFFICIENT_SIGNAL

# Per-cohort primary learner. Sprint 1: BASELINE_DIFF_MEANS everywhere.
# Sprint 2: selection pipeline populates this via Qini benchmark (spec §6.1).
primary_learner:
  hcf_catalyst_chf:
    default: { learner: "BASELINE_DIFF_MEANS", horizon_days: 30 }
  aged_care_au:
    default: { learner: "BASELINE_DIFF_MEANS", horizon_days: 90 }
```

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/cate_estimate.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/cate_estimate_test.go \
        backend/shared-infrastructure/market-configs/shared/cate_parameters.yaml
git commit -m "feat(gap22): CATEEstimate contract + primary-learner assignment + market YAML (Sprint 1 Task 2)"
```

---

### Task 3: Intervention registry service (YAML → DB → filter)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/intervention_registry.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/intervention_registry_test.go`

Loads the two YAML taxonomies at startup, persists them, and exposes `ListEligible(patient)` — the filter pass that the CATE learner calls before scoring.

- [ ] **Step 1: Write the failing tests**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/intervention_registry_test.go`:

```go
package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestInterventionRegistry_LoadFromYAML_PersistsDefinitions(t *testing.T) {
	db := setupTestDB(t) // helper defined in existing test scaffolding
	reg := NewInterventionRegistry(db)
	if err := reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_hcf_chf.yaml"); err != nil {
		t.Fatalf("load: %v", err)
	}
	var count int64
	db.Model(&models.InterventionDefinition{}).Where("cohort_id = ?", "hcf_catalyst_chf").Count(&count)
	if count != 6 {
		t.Fatalf("want 6 HCF CHF interventions, got %d", count)
	}
}

func TestInterventionRegistry_ListEligible_FiltersByCohort(t *testing.T) {
	db := setupTestDB(t)
	reg := NewInterventionRegistry(db)
	_ = reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_hcf_chf.yaml")
	_ = reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_aged_care_au.yaml")

	features := map[string]float64{
		"days_since_discharge": 3, "polypharmacy_score": 6, "phone_contact_opt_out": 0,
		"prior_cardiology_30d": 0, "device_eligible_flag": 1, "fluid_overload_flag": 0,
		"cognitive_impairment_flag": 0, "palliative_care_flag": 0,
	}
	got, err := reg.ListEligible("hcf_catalyst_chf", features)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, d := range got {
		if d.CohortID != "hcf_catalyst_chf" {
			t.Fatalf("leaked cross-cohort: %s", d.CohortID)
		}
	}
	if len(got) == 0 {
		t.Fatal("expected at least one eligible intervention for this patient")
	}
}

func TestInterventionRegistry_ListEligible_ExcludesContraindicated(t *testing.T) {
	db := setupTestDB(t)
	reg := NewInterventionRegistry(db)
	_ = reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_hcf_chf.yaml")

	// Patient opted out of phone contact → nurse_phone_48h must be excluded.
	features := map[string]float64{
		"days_since_discharge": 3, "phone_contact_opt_out": 1,
		"palliative_care_flag": 0, "cognitive_impairment_flag": 0,
		"polypharmacy_score": 6, "prior_cardiology_30d": 0,
		"fluid_overload_flag": 0, "device_eligible_flag": 1,
	}
	got, _ := reg.ListEligible("hcf_catalyst_chf", features)
	for _, d := range got {
		if d.ID == "nurse_phone_48h" {
			t.Fatal("phone-opt-out patient was offered nurse_phone_48h")
		}
	}
}

func TestInterventionRegistry_ListEligible_EnforcesEligibilityPredicate(t *testing.T) {
	db := setupTestDB(t)
	reg := NewInterventionRegistry(db)
	_ = reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_hcf_chf.yaml")

	// Polypharmacy score 2 (<5) → pharmacist_medication_review should NOT be eligible.
	features := map[string]float64{
		"days_since_discharge": 3, "polypharmacy_score": 2,
		"phone_contact_opt_out": 0, "palliative_care_flag": 0,
		"prior_cardiology_30d": 0, "device_eligible_flag": 1,
		"fluid_overload_flag": 0, "cognitive_impairment_flag": 0,
	}
	got, _ := reg.ListEligible("hcf_catalyst_chf", features)
	for _, d := range got {
		if d.ID == "pharmacist_medication_review" {
			t.Fatal("low-polypharmacy patient offered pharmacist review")
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestInterventionRegistry -v`
Expected: FAIL with "undefined: NewInterventionRegistry".

- [ ] **Step 3: Implement the registry**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/intervention_registry.go`:

```go
package services

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

type interventionYAML struct {
	CohortID                string `yaml:"cohort_id"`
	Version                 string `yaml:"version"`
	PrimaryCATEHorizonDays  int    `yaml:"primary_cate_horizon_days"`
	RecommendationCardinality int  `yaml:"recommendation_cardinality"`
	Interventions           []struct {
		ID                 string                        `yaml:"id"`
		Category           string                        `yaml:"category"`
		Name               string                        `yaml:"name"`
		ClinicianLanguage  string                        `yaml:"clinician_language"`
		CoolDownHours      int                           `yaml:"cool_down_hours"`
		ResourceCost       float64                       `yaml:"resource_cost"`
		FeatureSignature   []string                      `yaml:"feature_signature"`
		Eligibility        []models.EligibilityCriterion `yaml:"eligibility"`
		Contraindications  []models.Contraindication     `yaml:"contraindications"`
	} `yaml:"interventions"`
}

type InterventionRegistry struct {
	db *gorm.DB
}

func NewInterventionRegistry(db *gorm.DB) *InterventionRegistry {
	return &InterventionRegistry{db: db}
}

// LoadFromYAML parses a single market-config YAML file and upserts each definition.
// Idempotent; safe to call on every service start.
func (r *InterventionRegistry) LoadFromYAML(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read yaml: %w", err)
	}
	var cfg interventionYAML
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}
	for _, iv := range cfg.Interventions {
		eligJSON, _ := json.Marshal(iv.Eligibility)
		contraJSON, _ := json.Marshal(iv.Contraindications)
		def := models.InterventionDefinition{
			ID:                iv.ID,
			CohortID:          cfg.CohortID,
			Category:          iv.Category,
			Name:              iv.Name,
			ClinicianLanguage: iv.ClinicianLanguage,
			CoolDownHours:     iv.CoolDownHours,
			ResourceCost:      iv.ResourceCost,
			FeatureSignature:  iv.FeatureSignature,
			EligibilityJSON:   string(eligJSON),
			ContraindicationsJSON: string(contraJSON),
			Version:           cfg.Version,
			SourceYAMLPath:    path,
		}
		if err := def.Validate(); err != nil {
			return fmt.Errorf("validate %s: %w", iv.ID, err)
		}
		if err := r.db.Save(&def).Error; err != nil {
			return fmt.Errorf("persist %s: %w", iv.ID, err)
		}
	}
	return nil
}

// ListEligible returns interventions whose cohort matches and whose eligibility
// criteria hold and whose contraindications do NOT hold for the given feature vector.
// Order of evaluation: contraindication → eligibility. Either failure excludes.
func (r *InterventionRegistry) ListEligible(cohortID string, features map[string]float64) ([]models.InterventionDefinition, error) {
	var all []models.InterventionDefinition
	if err := r.db.Where("cohort_id = ?", cohortID).Find(&all).Error; err != nil {
		return nil, err
	}
	out := make([]models.InterventionDefinition, 0, len(all))
	for _, d := range all {
		var contra []models.Contraindication
		_ = json.Unmarshal([]byte(d.ContraindicationsJSON), &contra)
		if anyContraindicationMatches(contra, features) {
			continue
		}
		var elig []models.EligibilityCriterion
		_ = json.Unmarshal([]byte(d.EligibilityJSON), &elig)
		if !allEligibilityMatches(elig, features) {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func anyContraindicationMatches(contra []models.Contraindication, f map[string]float64) bool {
	for _, c := range contra {
		if predicateHolds(c.Operator, f[c.FeatureKey], c.Threshold) {
			return true
		}
	}
	return false
}

func allEligibilityMatches(elig []models.EligibilityCriterion, f map[string]float64) bool {
	for _, e := range elig {
		if !predicateHolds(e.Operator, f[e.FeatureKey], e.Threshold) {
			return false
		}
	}
	return true
}

func predicateHolds(op string, value, threshold float64) bool {
	switch op {
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestInterventionRegistry -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/intervention_registry.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/intervention_registry_test.go
git commit -m "feat(gap22): intervention registry YAML loader + eligibility filter (Sprint 1 Task 3)"
```

---

### Task 4: Propensity model + overlap-positivity diagnostic

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/propensity.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/propensity_test.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/overlap_diagnostic.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/overlap_diagnostic_test.go`

Propensity = P(receive intervention | features), fit once per cohort × intervention from the Gap 21 consolidated-record training set. Sprint 1 uses logistic regression (IRLS via gonum) on the intervention's `feature_signature`; propensity values outside the configured band short-circuit the CATE estimate to `OVERLAP_BELOW_FLOOR` / `OVERLAP_ABOVE_CEILING`.

- [ ] **Step 1: Write the failing propensity tests**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/propensity_test.go`:

```go
package services

import (
	"math"
	"testing"
)

func TestPropensityModel_Fit_PredictsKnownSeparableData(t *testing.T) {
	// Synthetic data where treated when feature > 0 with some noise.
	X := [][]float64{
		{1.0}, {1.2}, {0.9}, {1.5}, {0.8}, {1.1},
		{-1.0}, {-0.5}, {-1.2}, {-0.9}, {-1.5}, {-0.7},
	}
	y := []bool{true, true, true, true, true, true, false, false, false, false, false, false}
	m, err := FitPropensity(X, y, []string{"x"})
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	if p := m.Predict(map[string]float64{"x": 1.0}); p < 0.7 {
		t.Fatalf("want high propensity at x=1.0, got %.3f", p)
	}
	if p := m.Predict(map[string]float64{"x": -1.0}); p > 0.3 {
		t.Fatalf("want low propensity at x=-1.0, got %.3f", p)
	}
}

func TestPropensityModel_Predict_AlwaysIn01(t *testing.T) {
	X := [][]float64{{0}, {1}, {2}, {3}}
	y := []bool{false, false, true, true}
	m, _ := FitPropensity(X, y, []string{"x"})
	for x := -10.0; x <= 10.0; x += 0.5 {
		p := m.Predict(map[string]float64{"x": x})
		if p < 0 || p > 1 || math.IsNaN(p) {
			t.Fatalf("propensity out of [0,1]: %.3f at x=%.1f", p, x)
		}
	}
}

func TestPropensityModel_Fit_RejectsEmptyTrainingSet(t *testing.T) {
	if _, err := FitPropensity(nil, nil, []string{"x"}); err == nil {
		t.Fatal("expected error for empty training set")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestPropensityModel -v`
Expected: FAIL with "undefined: FitPropensity".

- [ ] **Step 3: Implement propensity**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/propensity.go`:

```go
package services

import (
	"errors"
	"math"
)

// PropensityModel is a minimal logistic regression fit via batch gradient descent.
// Sprint 1 deliberately avoids the gonum optim stack; the numerical requirements are
// modest and this keeps the dependency surface small. Sprint 2's Python service will
// replace this with a proper GBM-calibrated propensity (spec §6.1).
type PropensityModel struct {
	Intercept    float64
	Coefficients []float64
	FeatureKeys  []string
}

const (
	propensityEpochs     = 800
	propensityLearnRate  = 0.05
	propensityClipAbs    = 30.0 // clip logits to avoid overflow in sigmoid
)

// FitPropensity fits a logistic regression on features X (n×d) with binary labels y.
// featureKeys[i] names column i — same order Predict expects.
func FitPropensity(X [][]float64, y []bool, featureKeys []string) (*PropensityModel, error) {
	n := len(X)
	if n == 0 || len(y) != n {
		return nil, errors.New("empty or mismatched training set")
	}
	d := len(featureKeys)
	if d == 0 {
		return nil, errors.New("no features")
	}
	w := make([]float64, d)
	var b float64
	for epoch := 0; epoch < propensityEpochs; epoch++ {
		dw := make([]float64, d)
		var db float64
		for i := 0; i < n; i++ {
			z := b
			for j := 0; j < d; j++ {
				z += w[j] * X[i][j]
			}
			if z > propensityClipAbs {
				z = propensityClipAbs
			} else if z < -propensityClipAbs {
				z = -propensityClipAbs
			}
			p := 1.0 / (1.0 + math.Exp(-z))
			var yi float64
			if y[i] {
				yi = 1.0
			}
			err := p - yi
			for j := 0; j < d; j++ {
				dw[j] += err * X[i][j]
			}
			db += err
		}
		inv := propensityLearnRate / float64(n)
		for j := 0; j < d; j++ {
			w[j] -= inv * dw[j]
		}
		b -= inv * db
	}
	return &PropensityModel{Intercept: b, Coefficients: w, FeatureKeys: featureKeys}, nil
}

// Predict returns propensity in [0,1]. Missing features default to 0.
func (m *PropensityModel) Predict(features map[string]float64) float64 {
	z := m.Intercept
	for i, k := range m.FeatureKeys {
		z += m.Coefficients[i] * features[k]
	}
	if z > propensityClipAbs {
		z = propensityClipAbs
	} else if z < -propensityClipAbs {
		z = -propensityClipAbs
	}
	return 1.0 / (1.0 + math.Exp(-z))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestPropensityModel -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Write failing overlap diagnostic tests**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/overlap_diagnostic_test.go`:

```go
package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestEvaluateOverlap_InsideBandPasses(t *testing.T) {
	got := EvaluateOverlap(0.50, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if got != models.OverlapPass {
		t.Fatalf("want PASS, got %s", got)
	}
}

func TestEvaluateOverlap_BelowFloor(t *testing.T) {
	got := EvaluateOverlap(0.02, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if got != models.OverlapBelowFloor {
		t.Fatalf("want BELOW_FLOOR, got %s", got)
	}
}

func TestEvaluateOverlap_AboveCeiling(t *testing.T) {
	got := EvaluateOverlap(0.99, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if got != models.OverlapAboveCeiling {
		t.Fatalf("want ABOVE_CEILING, got %s", got)
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestEvaluateOverlap -v`
Expected: FAIL with "undefined: EvaluateOverlap".

- [ ] **Step 7: Implement overlap diagnostic**

Add the shared `OverlapBand` type to `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/overlap.go` (kept in `models` so `services` and `api` both depend on it without cycles):

```go
package models

// OverlapBand is the (floor, ceiling) outside which a propensity value is considered
// to fail the overlap check. Populated per cohort from cate_parameters.yaml.
type OverlapBand struct {
	Floor   float64
	Ceiling float64
}
```

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/overlap_diagnostic.go`:

```go
package services

import "kb-26-metabolic-digital-twin/internal/models"

// EvaluateOverlap is the single hard guard between the propensity model and the CATE
// learner. The spec §6.1 is explicit: "This is a hard guard and cannot be disabled."
// A propensity outside the band short-circuits CATE to an inconclusive result.
func EvaluateOverlap(propensity float64, band models.OverlapBand) models.OverlapStatus {
	switch {
	case propensity < band.Floor:
		return models.OverlapBelowFloor
	case propensity > band.Ceiling:
		return models.OverlapAboveCeiling
	default:
		return models.OverlapPass
	}
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestEvaluateOverlap -v`
Expected: PASS (3 tests).

- [ ] **Step 9: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/propensity.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/propensity_test.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/overlap_diagnostic.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/overlap_diagnostic_test.go
git commit -m "feat(gap22): logistic propensity + overlap-positivity hard guard (Sprint 1 Task 4)"
```

---

### Task 5: Baseline difference-in-means CATE learner with bootstrap CI

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/baseline_cate_learner.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/baseline_cate_learner_test.go`

This is the Sprint 1 CATE estimator. Mechanics:
1. Check intervention eligibility via `InterventionRegistry` (reuses Task 3).
2. Fit cohort × intervention propensity model from Gap 21 consolidated records (reuses Task 4).
3. Compute patient's propensity → overlap diagnostic. If fail → return `CATEEstimate` with `OverlapStatus = OVERLAP_*` and no point estimate.
4. Bucket training cohort by nearest-neighbour match on the `feature_signature`; within the bucket, compute `mean(outcome | treated) − mean(outcome | control)`.
5. Bootstrap CI from 500 resamples of the bucket.
6. Persist `CATEEstimate`; append `CATE_ESTIMATE` entry to the governance ledger.

- [ ] **Step 1: Write the failing tests**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/baseline_cate_learner_test.go`:

```go
package services

import (
	"math"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestBaselineCATELearner_RecoversKnownEffect(t *testing.T) {
	// Synthetic cohort: treatment causes a +0.20 outcome-probability lift uniformly.
	training := generateSyntheticCohort(500, 0.20, 42)
	est, err := estimateFromCohort(training, "P_target", map[string]float64{"age": 70}, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	if math.Abs(est.PointEstimate-0.20) > 0.08 {
		t.Fatalf("want CATE ≈ 0.20, got %.3f", est.PointEstimate)
	}
	if est.OverlapStatus != string(models.OverlapPass) {
		t.Fatalf("want OVERLAP_PASS, got %s", est.OverlapStatus)
	}
}

func TestBaselineCATELearner_CIWidthShrinksWithN(t *testing.T) {
	narrow := mustEstimate(t, generateSyntheticCohort(1000, 0.15, 7))
	wide := mustEstimate(t, generateSyntheticCohort(80, 0.15, 7))
	if (narrow.CIUpper - narrow.CILower) >= (wide.CIUpper - wide.CILower) {
		t.Fatal("expected larger N → narrower CI")
	}
}

func TestBaselineCATELearner_OverlapFailShortCircuits(t *testing.T) {
	// Patient features put them outside any training-set support.
	training := generateSyntheticCohort(300, 0.10, 99)
	est, err := estimateFromCohort(training, "P_outlier", map[string]float64{"age": 1000}, models.OverlapBand{Floor: 0.45, Ceiling: 0.55})
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	if est.OverlapStatus == string(models.OverlapPass) {
		t.Fatalf("expected overlap fail, got %s", est.OverlapStatus)
	}
	if est.PointEstimate != 0 {
		t.Fatalf("expected point=0 on overlap fail, got %.3f", est.PointEstimate)
	}
}

func TestBaselineCATELearner_InsufficientDataReturnsStatus(t *testing.T) {
	training := generateSyntheticCohort(5, 0.10, 1) // below min_training_n
	est, _ := estimateFromCohort(training, "P1", map[string]float64{"age": 70}, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if est.OverlapStatus != string(models.OverlapInsufficientData) {
		t.Fatalf("want OVERLAP_INSUFFICIENT_DATA, got %s", est.OverlapStatus)
	}
}

func TestBaselineCATELearner_FeatureContributionsSorted(t *testing.T) {
	training := generateSyntheticCohort(400, 0.12, 3)
	est := mustEstimate(t, training)
	if len(est.FeatureContributionKeys) == 0 {
		t.Fatal("expected at least one feature contribution")
	}
}
```

Helper `generateSyntheticCohort` / `mustEstimate` defined inline in the same `_test.go` file (note: `TrainingRow` itself is defined in the production file `baseline_cate_learner.go`, shared via the same `services` package):

```go
import "math/rand"

func generateSyntheticCohort(n int, trueEffect float64, seed int64) []TrainingRow {
	r := rand.New(rand.NewSource(seed))
	out := make([]TrainingRow, n)
	for i := 0; i < n; i++ {
		age := 60 + r.Float64()*25
		treated := r.Float64() > 0.5
		baseRisk := 0.3 + 0.01*(age-70)
		p := baseRisk
		if treated {
			p -= trueEffect
		}
		out[i] = TrainingRow{
			PatientID:       "T" + string(rune(i)),
			Features:        map[string]float64{"age": age},
			Treated:         treated,
			OutcomeOccurred: r.Float64() < p,
		}
	}
	return out
}

func mustEstimate(t *testing.T, rows []TrainingRow) models.CATEEstimate {
	t.Helper()
	est, err := estimateFromCohort(rows, "P_test", map[string]float64{"age": 72}, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	return est
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestBaselineCATELearner -v`
Expected: FAIL with "undefined: estimateFromCohort".

- [ ] **Step 3: Implement the baseline learner**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/baseline_cate_learner.go`:

```go
package services

import (
	"encoding/json"
	"math"
	"math/rand"
	"sort"

	"kb-26-metabolic-digital-twin/internal/models"
)

const (
	bootstrapResamples = 500
	ciLowerPct         = 5.0
	ciUpperPct         = 95.0
	minTrainingN       = 40
	topKFeatures       = 3
)

// TrainingRow is one labelled example in the cohort that trains the baseline learner.
// Populated from Gap 21's ConsolidatedAlertRecord + OutcomeRecord join; see
// loadTrainingCohort in cate_handlers.go.
type TrainingRow struct {
	PatientID       string
	Features        map[string]float64
	Treated         bool
	OutcomeOccurred bool
}

// estimateFromCohort is the Sprint 1 CATE kernel. It runs:
//  1. Insufficient-data check (combined N < minTrainingN → OVERLAP_INSUFFICIENT_DATA).
//  2. Propensity fit on the cohort, patient propensity → overlap diagnostic.
//  3. Nearest-neighbour bucket around the patient on the intervention's feature signature.
//  4. Bootstrap CI on (mean outcome | treated) − (mean outcome | control) inside the bucket.
//  5. Top-K feature contributions = signed bucket-mean − cohort-mean deltas.
//
// The returned CATEEstimate is the final on-the-wire shape; caller persists + appends
// a ledger entry.
func estimateFromCohort(rows []TrainingRow, patientID string, patientFeatures map[string]float64, band models.OverlapBand) (models.CATEEstimate, error) {
	est := models.CATEEstimate{PatientID: patientID}

	if len(rows) < minTrainingN {
		est.OverlapStatus = string(models.OverlapInsufficientData)
		est.TrainingN = len(rows)
		return est, nil
	}

	// Propensity fit on cohort.
	featureKeys := extractFeatureKeys(rows)
	X, y := buildPropensityMatrix(rows, featureKeys)
	prop, err := FitPropensity(X, y, featureKeys)
	if err != nil {
		return est, err
	}
	p := prop.Predict(patientFeatures)
	status := EvaluateOverlap(p, band)
	est.Propensity = p
	est.OverlapStatus = string(status)
	est.TrainingN = len(rows)

	if status != models.OverlapPass {
		return est, nil
	}

	// Nearest-neighbour bucket: top-50% of rows ranked by L1 distance on featureKeys.
	bucket := nearestBucket(rows, patientFeatures, featureKeys, len(rows)/2)
	var treatedOut, controlOut []int
	for _, r := range bucket {
		o := 0
		if r.OutcomeOccurred {
			o = 1
		}
		if r.Treated {
			treatedOut = append(treatedOut, o)
		} else {
			controlOut = append(controlOut, o)
		}
	}
	est.CohortTreatedN = len(treatedOut)
	est.CohortControlN = len(controlOut)

	if len(treatedOut) < 5 || len(controlOut) < 5 {
		est.OverlapStatus = string(models.OverlapInsufficientData)
		return est, nil
	}

	// Note: CATE sign convention — positive = treatment reduces outcome probability
	// (i.e., treated has lower risk). Matches Gap 21 AttributionVerdict.RiskDifference.
	point := meanInt(controlOut) - meanInt(treatedOut)
	lower, upper := bootstrapDiffCI(treatedOut, controlOut, bootstrapResamples, ciLowerPct, ciUpperPct, 42)
	est.PointEstimate = point
	est.CILower = lower
	est.CIUpper = upper

	// Feature contributions: bucket mean − cohort mean per feature, top-K by |delta|.
	est.FeatureContributionKeys, est.FeatureContributionsJSON = computeFeatureContributions(rows, bucket, patientFeatures, featureKeys, topKFeatures)
	return est, nil
}

// meanInt returns arithmetic mean of an int slice; empty → 0.
func meanInt(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s int
	for _, x := range xs {
		s += x
	}
	return float64(s) / float64(len(xs))
}

// bootstrapDiffCI resamples treated and control with replacement B times and returns
// the (lowerPct, upperPct) percentiles of the diff-in-means distribution.
func bootstrapDiffCI(treated, control []int, B int, lowerPct, upperPct float64, seed int64) (float64, float64) {
	r := rand.New(rand.NewSource(seed))
	samples := make([]float64, B)
	for b := 0; b < B; b++ {
		t := resampleInt(treated, r)
		c := resampleInt(control, r)
		samples[b] = meanInt(c) - meanInt(t)
	}
	sort.Float64s(samples)
	lower := samples[int(math.Floor(lowerPct/100*float64(B)))]
	upper := samples[int(math.Floor(upperPct/100*float64(B)))]
	return lower, upper
}

func resampleInt(xs []int, r *rand.Rand) []int {
	out := make([]int, len(xs))
	for i := range out {
		out[i] = xs[r.Intn(len(xs))]
	}
	return out
}

func extractFeatureKeys(rows []TrainingRow) []string {
	seen := map[string]struct{}{}
	var keys []string
	for _, r := range rows {
		for k := range r.Features {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

func buildPropensityMatrix(rows []TrainingRow, keys []string) ([][]float64, []bool) {
	X := make([][]float64, len(rows))
	y := make([]bool, len(rows))
	for i, r := range rows {
		X[i] = make([]float64, len(keys))
		for j, k := range keys {
			X[i][j] = r.Features[k]
		}
		y[i] = r.Treated
	}
	return X, y
}

func nearestBucket(rows []TrainingRow, target map[string]float64, keys []string, k int) []TrainingRow {
	type scored struct {
		row TrainingRow
		dist float64
	}
	out := make([]scored, len(rows))
	for i, r := range rows {
		var d float64
		for _, k := range keys {
			d += math.Abs(r.Features[k] - target[k])
		}
		out[i] = scored{r, d}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].dist < out[j].dist })
	if k > len(out) {
		k = len(out)
	}
	bucket := make([]TrainingRow, k)
	for i := 0; i < k; i++ {
		bucket[i] = out[i].row
	}
	return bucket
}

func computeFeatureContributions(all, bucket []TrainingRow, patient map[string]float64, keys []string, topK int) ([]string, string) {
	type kd struct {
		key   string
		delta float64
		pv    float64
		cm    float64
	}
	deltas := make([]kd, 0, len(keys))
	for _, k := range keys {
		var cohortSum, bucketSum float64
		for _, r := range all {
			cohortSum += r.Features[k]
		}
		for _, r := range bucket {
			bucketSum += r.Features[k]
		}
		cohortMean := cohortSum / float64(len(all))
		bucketMean := bucketSum / float64(len(bucket))
		deltas = append(deltas, kd{k, bucketMean - cohortMean, patient[k], cohortMean})
	}
	sort.Slice(deltas, func(i, j int) bool { return math.Abs(deltas[i].delta) > math.Abs(deltas[j].delta) })
	if topK > len(deltas) {
		topK = len(deltas)
	}
	keysOut := make([]string, topK)
	contribs := make([]models.FeatureContribution, topK)
	for i := 0; i < topK; i++ {
		keysOut[i] = deltas[i].key
		contribs[i] = models.FeatureContribution{
			FeatureKey:   deltas[i].key,
			Contribution: deltas[i].delta,
			PatientValue: deltas[i].pv,
			CohortMean:   deltas[i].cm,
		}
	}
	payload, _ := json.Marshal(contribs)
	return keysOut, string(payload)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestBaselineCATELearner -v`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/baseline_cate_learner.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/baseline_cate_learner_test.go
git commit -m "feat(gap22): baseline difference-in-means CATE learner + bootstrap CI (Sprint 1 Task 5)"
```

---

### Task 6: CATE calibration monitor (closes the loop with Gap 21)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cate_calibration_monitor.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cate_calibration_monitor_test.go`

The calibration monitor is the only Sprint 1 component that crosses gaps: it reads the `CATEEstimate` produced at T0 and the `AttributionVerdict.RiskDifference` computed by Gap 21 at T4, pairs them by `ConsolidatedRecordID`, and computes rolling-window `mean(|attributed − predicted|)` per cohort × intervention. Threshold breach → `CATE_MISCALIBRATION` ledger entry.

- [ ] **Step 1: Write the failing tests**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cate_calibration_monitor_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestCalibrationMonitor_CalibratedSignalNoAlarm(t *testing.T) {
	db := setupTestDB(t)
	seedMatchedPairs(t, db, "hcf_catalyst_chf", "nurse_phone_48h", 30, 0.15, 0.15, 30)
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	sum, err := mon.ComputeCalibrationSummary("hcf_catalyst_chf", "nurse_phone_48h", 30)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if sum.AlarmTriggered {
		t.Fatalf("calibrated signal should not alarm: meanAbsDiff=%.3f", sum.MeanAbsDiff)
	}
}

func TestCalibrationMonitor_MiscalibratedSignalAlarms(t *testing.T) {
	db := setupTestDB(t)
	// Predicted 0.20 but attributed only 0.02 on every pair → big miscalibration.
	seedMatchedPairs(t, db, "hcf_catalyst_chf", "gp_visit_7d", 30, 0.20, 0.02, 30)
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	sum, _ := mon.ComputeCalibrationSummary("hcf_catalyst_chf", "gp_visit_7d", 30)
	if !sum.AlarmTriggered {
		t.Fatalf("miscalibrated signal should alarm, mean=%.3f", sum.MeanAbsDiff)
	}
}

func TestCalibrationMonitor_InsufficientPairsNoAlarm(t *testing.T) {
	db := setupTestDB(t)
	seedMatchedPairs(t, db, "hcf_catalyst_chf", "cardiology_referral", 30, 0.20, 0.02, 3) // only 3 pairs
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	sum, _ := mon.ComputeCalibrationSummary("hcf_catalyst_chf", "cardiology_referral", 30)
	if sum.AlarmTriggered {
		t.Fatal("insufficient-signal should not alarm")
	}
	if sum.Status != CalibrationInsufficientSignal {
		t.Fatalf("want INSUFFICIENT_SIGNAL, got %s", sum.Status)
	}
}

func TestCalibrationMonitor_EvaluateAppendsLedgerOnAlarm(t *testing.T) {
	db := setupTestDB(t)
	seedMatchedPairs(t, db, "hcf_catalyst_chf", "gp_visit_7d", 30, 0.20, 0.02, 25)
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	ledger := NewAppendOnlyLedger(db, "test-hmac-key")
	if err := mon.EvaluateAndAlarm("hcf_catalyst_chf", ledger); err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	var count int64
	db.Model(&models.LedgerEntry{}).Where("entry_type = ?", "CATE_MISCALIBRATION").Count(&count)
	if count < 1 {
		t.Fatal("expected at least one CATE_MISCALIBRATION ledger entry")
	}
}

// helper: seed N matched (CATEEstimate, AttributionVerdict) pairs in the DB.
func seedMatchedPairs(t *testing.T, db *gorm.DB, cohort, intervention string, horizon int, predCATE, attribEffect float64, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		rid := uuid.New()
		db.Create(&models.CATEEstimate{
			ConsolidatedRecordID: rid, PatientID: fmt.Sprintf("P%d", i),
			CohortID: cohort, InterventionID: intervention, LearnerType: string(models.LearnerBaselineDiffMeans),
			PointEstimate: predCATE, CILower: predCATE - 0.03, CIUpper: predCATE + 0.03,
			HorizonDays: horizon, OverlapStatus: string(models.OverlapPass), ComputedAt: time.Now(),
		})
		db.Create(&models.AttributionVerdict{
			ConsolidatedRecordID: rid, PatientID: fmt.Sprintf("P%d", i),
			CohortID: cohort, ClinicianLabel: string(models.LabelPrevented),
			RiskDifference: attribEffect, PredictionWindowDays: horizon, ComputedAt: time.Now(),
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestCalibrationMonitor -v`
Expected: FAIL with "undefined: NewCATECalibrationMonitor".

- [ ] **Step 3: Implement calibration monitor**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cate_calibration_monitor.go`:

```go
package services

import (
	"encoding/json"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

type CalibrationConfig struct {
	AbsDiffAlarm       float64
	RollingWindowDays  int
	MinMatchedPairs    int
}

type CalibrationStatus string

const (
	CalibrationOK                  CalibrationStatus = "OK"
	CalibrationAlarm               CalibrationStatus = "ALARM"
	CalibrationInsufficientSignal  CalibrationStatus = "INSUFFICIENT_SIGNAL"
)

type CalibrationSummary struct {
	CohortID         string            `json:"cohort_id"`
	InterventionID   string            `json:"intervention_id"`
	HorizonDays      int               `json:"horizon_days"`
	MatchedPairs     int               `json:"matched_pairs"`
	MeanAbsDiff      float64           `json:"mean_abs_diff"`
	WindowStart      time.Time         `json:"window_start"`
	WindowEnd        time.Time         `json:"window_end"`
	Status           CalibrationStatus `json:"status"`
	AlarmTriggered   bool              `json:"alarm_triggered"`
}

type CATECalibrationMonitor struct {
	db  *gorm.DB
	cfg CalibrationConfig
}

func NewCATECalibrationMonitor(db *gorm.DB, cfg CalibrationConfig) *CATECalibrationMonitor {
	return &CATECalibrationMonitor{db: db, cfg: cfg}
}

// ComputeCalibrationSummary pairs each CATEEstimate with the AttributionVerdict
// for the same ConsolidatedRecordID, restricts to the rolling window, and computes
// mean(|attributed − predicted|). Returns status accordingly.
func (m *CATECalibrationMonitor) ComputeCalibrationSummary(cohortID, interventionID string, horizonDays int) (CalibrationSummary, error) {
	windowEnd := time.Now()
	windowStart := windowEnd.AddDate(0, 0, -m.cfg.RollingWindowDays)

	type joined struct {
		PredCATE   float64
		Attributed float64
	}
	var rows []joined
	err := m.db.Raw(`
		SELECT c.point_estimate AS pred_cate, a.risk_difference AS attributed
		FROM cate_estimates c
		INNER JOIN attribution_verdicts a ON c.consolidated_record_id = a.consolidated_record_id
		WHERE c.cohort_id = ? AND c.intervention_id = ? AND c.horizon_days = ?
		  AND c.overlap_status = ?
		  AND a.computed_at BETWEEN ? AND ?
	`, cohortID, interventionID, horizonDays, string(models.OverlapPass), windowStart, windowEnd).Scan(&rows).Error
	if err != nil {
		return CalibrationSummary{}, err
	}

	sum := CalibrationSummary{
		CohortID: cohortID, InterventionID: interventionID, HorizonDays: horizonDays,
		MatchedPairs: len(rows), WindowStart: windowStart, WindowEnd: windowEnd,
	}
	if len(rows) < m.cfg.MinMatchedPairs {
		sum.Status = CalibrationInsufficientSignal
		return sum, nil
	}
	var total float64
	for _, r := range rows {
		total += math.Abs(r.Attributed - r.PredCATE)
	}
	sum.MeanAbsDiff = total / float64(len(rows))
	if sum.MeanAbsDiff > m.cfg.AbsDiffAlarm {
		sum.Status = CalibrationAlarm
		sum.AlarmTriggered = true
	} else {
		sum.Status = CalibrationOK
	}
	return sum, nil
}

// EvaluateAndAlarm runs ComputeCalibrationSummary for every (cohort, intervention,
// horizon) triple with data and appends a CATE_MISCALIBRATION ledger entry for each
// alarm. Intended to be called on a schedule (cron / Kafka trigger) — Sprint 1 exposes
// it only via HTTP for manual triggering; Sprint 2 wires up scheduler.
func (m *CATECalibrationMonitor) EvaluateAndAlarm(cohortID string, ledger *AppendOnlyLedger) error {
	if ledger == nil {
		return errors.New("ledger required")
	}
	type triple struct {
		InterventionID string
		HorizonDays    int
	}
	var triples []triple
	if err := m.db.Raw(`
		SELECT DISTINCT intervention_id, horizon_days FROM cate_estimates WHERE cohort_id = ?
	`, cohortID).Scan(&triples).Error; err != nil {
		return err
	}
	for _, t := range triples {
		sum, err := m.ComputeCalibrationSummary(cohortID, t.InterventionID, t.HorizonDays)
		if err != nil {
			return err
		}
		if !sum.AlarmTriggered {
			continue
		}
		payload, _ := json.Marshal(sum)
		if _, err := ledger.AppendEntry("CATE_MISCALIBRATION", cohortID+":"+t.InterventionID, string(payload)); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestCalibrationMonitor -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cate_calibration_monitor.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cate_calibration_monitor_test.go
git commit -m "feat(gap22): CATE calibration monitor joins Gap 21 attribution at T4 (Sprint 1 Task 6)"
```

---

### Task 7: HTTP handlers, routes, AutoMigrate wiring

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/cate_handlers.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/cmd/server/main.go`

Three endpoints: `POST /cate/estimate` (inference), `GET /cate/:id` (read-back), `GET /cate/calibration/summary/:cohortId` (calibration report). Every `POST /cate/estimate` call writes a `CATE_ESTIMATE` entry to the existing Gap 21 ledger.

- [ ] **Step 1: Read existing route wiring to match the pattern**

Run: `cat backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go` and note the existing `/attribution` and `/governance` route groups added in Gap 21 Sprint 1.

- [ ] **Step 2: Write failing handler test**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/cate_handlers_test.go`:

```go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPostCATEEstimate_ReturnsEstimateAndLedgers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupTestRouter(t) // existing helper from Gap 21 tests
	body := map[string]any{
		"consolidated_record_id": "00000000-0000-0000-0000-000000000001",
		"patient_id":             "P1",
		"cohort_id":              "hcf_catalyst_chf",
		"intervention_id":        "nurse_phone_48h",
		"features":               map[string]float64{"age": 72, "ef_last": 35, "nt_probnp_trend_7d": 0.2, "days_since_discharge": 2, "weight_trend_7d": 0.1},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/cate/estimate", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCalibrationSummary_ReturnsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := setupTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/cate/calibration/summary/hcf_catalyst_chf?intervention=nurse_phone_48h&horizon=30", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/api/ -run TestPostCATEEstimate -v`
Expected: FAIL with "no route matched".

- [ ] **Step 4: Implement handlers**

Create `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/cate_handlers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

type CATEHandlers struct {
	db       *gorm.DB
	registry *services.InterventionRegistry
	monitor  *services.CATECalibrationMonitor
	ledger   *services.AppendOnlyLedger
	band     models.OverlapBand
}

func NewCATEHandlers(db *gorm.DB, reg *services.InterventionRegistry, mon *services.CATECalibrationMonitor, l *services.AppendOnlyLedger, band models.OverlapBand) *CATEHandlers {
	return &CATEHandlers{db: db, registry: reg, monitor: mon, ledger: l, band: band}
}

type estimateRequest struct {
	ConsolidatedRecordID uuid.UUID          `json:"consolidated_record_id"`
	PatientID            string             `json:"patient_id"`
	CohortID             string             `json:"cohort_id"`
	InterventionID       string             `json:"intervention_id"`
	HorizonDays          int                `json:"horizon_days"`
	Features             map[string]float64 `json:"features"`
}

// PostCATEEstimate runs Sprint 1's baseline estimator for one (patient, intervention)
// and persists + ledgers the result.
func (h *CATEHandlers) PostCATEEstimate(c *gin.Context) {
	var req estimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Eligibility short-circuit: if the patient fails contraindication/eligibility,
	// no CATE is computed; the caller gets a 200 with OverlapStatus = OVERLAP_INSUFFICIENT_DATA
	// and an explanatory reason. Spec §6.1, §6.2.
	eligible, err := h.registry.ListEligible(req.CohortID, req.Features)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var matched bool
	for _, d := range eligible {
		if d.ID == req.InterventionID {
			matched = true
			break
		}
	}
	if !matched {
		c.JSON(http.StatusOK, models.CATEEstimate{
			PatientID: req.PatientID, CohortID: req.CohortID, InterventionID: req.InterventionID,
			OverlapStatus: string(models.OverlapInsufficientData),
		})
		return
	}

	// Load training cohort for this (cohort, intervention) from consolidated records.
	// Sprint 1 uses a SQL view joining consolidated_alert_records (KB-23) with
	// outcome_records (KB-23) and the intervention actually delivered at T3.
	rows, err := loadTrainingCohort(h.db, req.CohortID, req.InterventionID, req.HorizonDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	est, err := services.EstimateFromCohort(rows, req.PatientID, req.Features, h.band)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	est.ConsolidatedRecordID = req.ConsolidatedRecordID
	est.CohortID = req.CohortID
	est.InterventionID = req.InterventionID
	est.HorizonDays = req.HorizonDays
	est.LearnerType = string(models.LearnerBaselineDiffMeans)
	est.ModelVersion = "baseline-1.0.0"

	if err := h.db.Create(&est).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	payload, _ := json.Marshal(est)
	entry, err := h.ledger.AppendEntry("CATE_ESTIMATE", est.PatientID+":"+est.InterventionID, string(payload))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	est.LedgerEntryID = &entry.ID
	h.db.Save(&est)
	c.JSON(http.StatusOK, est)
}

func (h *CATEHandlers) GetCATEEstimate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var est models.CATEEstimate
	if err := h.db.First(&est, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, est)
}

func (h *CATEHandlers) GetCalibrationSummary(c *gin.Context) {
	cohort := c.Param("cohortId")
	intervention := c.Query("intervention")
	horizon, _ := strconv.Atoi(c.DefaultQuery("horizon", "30"))
	sum, err := h.monitor.ComputeCalibrationSummary(cohort, intervention, horizon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sum)
}
```

Export the learner entrypoint by renaming `estimateFromCohort` → `EstimateFromCohort` in `baseline_cate_learner.go` (and the same rename in `baseline_cate_learner_test.go`).

Add `loadTrainingCohort` at the bottom of `cate_handlers.go`. It joins Gap 21's `ConsolidatedAlertRecord` (KB-23) with `OutcomeRecord` (KB-23) and the delivered intervention to produce `[]services.TrainingRow`:

```go
// loadTrainingCohort returns all historical (patient, treated?, outcome) rows for the
// given cohort × intervention × horizon. Sprint 1 reads a view exposed by KB-23 via
// a direct SQL join across the shared DB schema. Sprint 2 will move this behind a
// gRPC call to KB-23 so KB-26 is not coupled to KB-23's tables.
func loadTrainingCohort(db *gorm.DB, cohortID, interventionID string, horizonDays int) ([]services.TrainingRow, error) {
	type raw struct {
		PatientID       string
		Features        string // consolidated_alert_record.features_json
		Treated         bool
		OutcomeOccurred bool
	}
	var rs []raw
	err := db.Raw(`
		SELECT c.patient_id           AS patient_id,
		       c.features_json        AS features,
		       (c.treatment_strategy = 'ACTIVE_' || ?) AS treated,
		       COALESCE(o.outcome_occurred, false) AS outcome_occurred
		FROM consolidated_alert_records c
		LEFT JOIN outcome_records o ON o.lifecycle_id = c.lifecycle_id
		WHERE c.cohort_id = ?
		  AND c.horizon_days = ?
	`, interventionID, cohortID, horizonDays).Scan(&rs).Error
	if err != nil {
		return nil, err
	}
	out := make([]services.TrainingRow, len(rs))
	for i, r := range rs {
		var f map[string]float64
		_ = json.Unmarshal([]byte(r.Features), &f)
		out[i] = services.TrainingRow{PatientID: r.PatientID, Features: f, Treated: r.Treated, OutcomeOccurred: r.OutcomeOccurred}
	}
	return out, nil
}
```

**Note for the executor:** the exact column names on `consolidated_alert_records` (`features_json`, `treatment_strategy`, `horizon_days`) need cross-checking against the live Gap 21 Sprint 1 schema before this compiles — if column names differ, update the SQL but keep the return shape. Sprint 2's gRPC-to-KB-23 refactor removes this coupling.

- [ ] **Step 5: Register routes**

Modify `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go` — add alongside the existing attribution and governance groups:

```go
// Gap 22 Sprint 1 CATE routes
cate := r.Group("/cate")
{
	cate.POST("/estimate", cateHandlers.PostCATEEstimate)
	cate.GET("/:id", cateHandlers.GetCATEEstimate)
	cate.GET("/calibration/summary/:cohortId", cateHandlers.GetCalibrationSummary)
}
```

- [ ] **Step 6: Wire AutoMigrate and startup YAML load**

Modify `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/cmd/server/main.go`:
1. Add to the AutoMigrate call:
   ```go
   &models.InterventionDefinition{},
   &models.CATEEstimate{},
   &models.CATEPrimaryLearnerAssignment{},
   ```
2. After AutoMigrate, load the two intervention YAMLs and the CATE parameters YAML:
   ```go
   reg := services.NewInterventionRegistry(db)
   if err := reg.LoadFromYAML("/app/market-configs/shared/intervention_taxonomy_hcf_chf.yaml"); err != nil {
       log.Fatalf("load HCF CHF taxonomy: %v", err)
   }
   if err := reg.LoadFromYAML("/app/market-configs/shared/intervention_taxonomy_aged_care_au.yaml"); err != nil {
       log.Fatalf("load Aged Care AU taxonomy: %v", err)
   }
   cateCfg, err := services.LoadCATEParameters("/app/market-configs/shared/cate_parameters.yaml")
   if err != nil { log.Fatalf("load cate params: %v", err) }
   ```
3. Instantiate `CATECalibrationMonitor` and `CATEHandlers` and pass into the router:
   ```go
   monitor := services.NewCATECalibrationMonitor(db, cateCfg.CalibrationConfig())
   cateHandlers := api.NewCATEHandlers(db, reg, monitor, ledger, cateCfg.BandForCohort("hcf_catalyst_chf"))
   ```

Add `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/cate_parameters_loader.go`:

```go
package services

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"kb-26-metabolic-digital-twin/internal/models"
)

type CATEParameters struct {
	Version      string `yaml:"version"`
	OverlapBand  struct {
		Default   bandYAML            `yaml:"default"`
		PerCohort map[string]bandYAML `yaml:"per_cohort"`
	} `yaml:"overlap_band"`
	Bootstrap struct {
		NResamples int     `yaml:"n_resamples"`
		CILevel    float64 `yaml:"ci_level"`
	} `yaml:"bootstrap"`
	MinTrainingN int `yaml:"min_training_n"`
	Calibration  struct {
		AlarmThresholdAbsDiff float64 `yaml:"alarm_threshold_abs_diff"`
		RollingWindowDays     int     `yaml:"rolling_window_days"`
		MinMatchedPairs       int     `yaml:"min_matched_pairs"`
	} `yaml:"calibration"`
	PrimaryLearner map[string]map[string]struct {
		Learner     string `yaml:"learner"`
		HorizonDays int    `yaml:"horizon_days"`
	} `yaml:"primary_learner"`
}

type bandYAML struct {
	Floor   float64 `yaml:"floor"`
	Ceiling float64 `yaml:"ceiling"`
}

func LoadCATEParameters(path string) (*CATEParameters, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cate params: %w", err)
	}
	var p CATEParameters
	if err := yaml.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse cate params: %w", err)
	}
	return &p, nil
}

// BandForCohort returns the cohort-specific overlap band, falling back to default.
func (p *CATEParameters) BandForCohort(cohortID string) models.OverlapBand {
	if b, ok := p.OverlapBand.PerCohort[cohortID]; ok {
		return models.OverlapBand{Floor: b.Floor, Ceiling: b.Ceiling}
	}
	return models.OverlapBand{Floor: p.OverlapBand.Default.Floor, Ceiling: p.OverlapBand.Default.Ceiling}
}

// CalibrationConfig maps the YAML block onto the monitor's expected config struct.
func (p *CATEParameters) CalibrationConfig() CalibrationConfig {
	return CalibrationConfig{
		AbsDiffAlarm:      p.Calibration.AlarmThresholdAbsDiff,
		RollingWindowDays: p.Calibration.RollingWindowDays,
		MinMatchedPairs:   p.Calibration.MinMatchedPairs,
	}
}
```

(`OverlapBand` is already in `internal/models/overlap.go` from Task 4, so both `services` and `api` can import it without a cycle.)

- [ ] **Step 7: Run handler tests**

Run: `go test ./internal/api/ -run TestPostCATEEstimate -v && go test ./internal/api/ -run TestGetCalibrationSummary -v`
Expected: PASS.

- [ ] **Step 8: Run the full KB-26 suite to catch regressions**

Run: `go test ./... -count=1`
Expected: PASS (Gap 21 Sprint 1-3 tests plus new Sprint 1 Gap 22 tests).

- [ ] **Step 9: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/cate_handlers.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/cate_handlers_test.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/routes.go \
        backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/cmd/server/main.go
git commit -m "feat(gap22): CATE HTTP handlers + route wiring + startup YAML load (Sprint 1 Task 7)"
```

---

## Acceptance Criteria — Sprint 1

From spec §7.1 Phase 1, adapted to Sprint 1's rule-based-baseline scope:

- [ ] End-to-end CATE estimate for a single HCF CHF patient × intervention returns in **under 500 ms** (p95) given a consolidated record. Baseline learner + bootstrap on a 500-row cohort benchmarks well below this on commodity hardware; verify with a `go test -bench` against a seeded DB.
- [ ] Overlap-positivity diagnostics automatically flag inconclusive cases. Property-based test on the overlap diagnostic confirms **no path exists** from a propensity outside the band to `OverlapPass`.
- [ ] The `CATEPrimaryLearnerAssignment` table contains one row per (cohort × intervention × horizon) on service start, all pointing to `BASELINE_DIFF_MEANS`. Schema ready for Sprint 2's Qini-selected learners.
- [ ] CATE calibration monitor fires a **synthetic miscalibration alarm** within expected latency and writes a `CATE_MISCALIBRATION` ledger entry that `VerifyChain` confirms is valid.
- [ ] Zero CATE estimates reach the DB without an associated `CATE_ESTIMATE` ledger entry — verified by a row-count equality check in a smoke test.
- [ ] **No clinician-facing surface**: there is no handler returning recommendation results to a worklist. This gate is explicit — Sprint 3 owns the recommender UX; Sprint 1 must not expose CATE directly to clinicians.

---

## What's Next (Sprints 2–5 of Gap 22)

Each sprint gets its own plan file in this folder. Writing these is the next session's work:

1. **Sprint 2 — CATE Learner Committee (Python).** New KB-28 (or KB-29) Python service implementing S/T/X/DR/R + causal forest, SHAP feature contributions, Qini-based per-cohort learner selection. Replaces the Sprint 1 Go baseline behind the stable `CATEEstimate` contract. Covers spec §6.1 Steps 1.2–1.4 in full.
2. **Sprint 3 — Recommender + Safety + Explanation + Worklist.** Constraint filter, capacity optimiser, ranking, rule-based safety gates, explanation layer templates, Gap 18 worklist Recommendation Panel. Spec §6.2, §6.3, §6.5, §2.
3. **Sprint 4 — Digital Twin + DTCF Validation + Policy Evaluator.** Bounded-residual twin, five-level DTCF validation, simulator, OPE + DataCOPE. Spec §6.4, §3.
4. **Sprint 5 — Policy Governance + Shadow/Canary/Promotion + FDA 2026 Compliance Pack.** PCCP policy-entry schema, deployment pipeline, compliance artefacts. Spec §6.6, §4.

---

*End of Sprint 1 plan. Sprint 1 ships the contract and the rule-based math; every later sprint replaces mechanisms behind the same `CATEEstimate` shape.*
