package models

import "time"

// ---------------------------------------------------------------------------
// InertiaDomain — clinical domain where therapeutic inertia is evaluated
// ---------------------------------------------------------------------------

// InertiaDomain identifies the clinical domain of inertia evaluation.
type InertiaDomain string

const (
	DomainGlycaemic   InertiaDomain = "GLYCAEMIC"
	DomainHemodynamic InertiaDomain = "HEMODYNAMIC"
	DomainRenal       InertiaDomain = "RENAL"
	DomainLipid       InertiaDomain = "LIPID"
)

// ---------------------------------------------------------------------------
// InertiaPattern — the specific pattern of therapeutic inertia detected
// ---------------------------------------------------------------------------

// InertiaPattern classifies the type of inertia detected.
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

// ---------------------------------------------------------------------------
// InertiaSeverity — Khunti-bracket severity classification
// ---------------------------------------------------------------------------

// InertiaSeverity classifies duration-based severity of therapeutic inertia.
type InertiaSeverity string

const (
	SeverityMild     InertiaSeverity = "MILD"
	SeverityModerate InertiaSeverity = "MODERATE"
	SeveritySevere   InertiaSeverity = "SEVERE"
	SeverityCritical InertiaSeverity = "CRITICAL"
)

// ---------------------------------------------------------------------------
// InertiaVerdict — single domain inertia evaluation result
// ---------------------------------------------------------------------------

// InertiaVerdict captures the full evaluation of therapeutic inertia for one
// domain/pattern combination, including clinical context and next steps.
type InertiaVerdict struct {
	Domain              InertiaDomain  `json:"domain"`
	Pattern             InertiaPattern `json:"pattern"`
	Detected            bool           `json:"detected"`
	InertiaDurationDays int            `json:"inertia_duration_days"`

	// Target vs actual
	TargetValue  float64 `json:"target_value"`
	CurrentValue float64 `json:"current_value"`

	// Evidence window
	FirstExceedanceDate time.Time `json:"first_exceedance_date"`
	ConsecutiveReadings int       `json:"consecutive_readings"`
	DataSource          string    `json:"data_source"`

	// Intervention history
	LastInterventionDate  *time.Time `json:"last_intervention_date,omitempty"`
	LastInterventionType  string     `json:"last_intervention_type,omitempty"`
	DaysSinceIntervention int        `json:"days_since_intervention"`

	// Medication context
	CurrentMedications []string `json:"current_medications"`
	AtMaxDose          bool     `json:"at_max_dose"`
	NextStepInPathway  string   `json:"next_step_in_pathway"`

	// Market-specific barriers
	CostBarrierLikely   bool `json:"cost_barrier_likely"`
	PBSAuthorityRequired bool `json:"pbs_authority_required"`

	// Classification
	Severity         InertiaSeverity `json:"severity"`
	RiskAccumulation string          `json:"risk_accumulation"`
	GuidelineReference string        `json:"guideline_reference"`
}

// ---------------------------------------------------------------------------
// PatientInertiaReport — aggregate inertia assessment across all domains
// ---------------------------------------------------------------------------

// PatientInertiaReport is the top-level inertia report for a patient,
// aggregating verdicts across all evaluated domains.
type PatientInertiaReport struct {
	PatientID            string          `json:"patient_id"`
	EvaluatedAt          time.Time       `json:"evaluated_at"`
	Verdicts             []InertiaVerdict `json:"verdicts"`
	HasAnyInertia        bool            `json:"has_any_inertia"`
	HasDualDomainInertia bool            `json:"has_dual_domain_inertia"`
	MostSevere           *InertiaVerdict `json:"most_severe,omitempty"`
	OverallUrgency       string          `json:"overall_urgency"`
	InertiaScore         float64         `json:"inertia_score"`
}

// ---------------------------------------------------------------------------
// DomainTargetStatus — whether a patient is at target for a given domain
// ---------------------------------------------------------------------------

// DomainTargetStatus tracks the current vs target state for a single domain,
// providing evidence for inertia detection.
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

// ---------------------------------------------------------------------------
// InterventionTimeline — medication change history by domain
// ---------------------------------------------------------------------------

// InterventionTimeline tracks recent medication changes per domain to
// distinguish true inertia from active titration.
type InterventionTimeline struct {
	PatientID              string                       `json:"patient_id"`
	ByDomain               map[InertiaDomain]LatestAction `json:"by_domain"`
	AnyChangeInLast12Weeks bool                         `json:"any_change_in_last_12_weeks"`
	TotalActiveInterventions int                        `json:"total_active_interventions"`
}

// LatestAction describes the most recent medication action in a domain.
type LatestAction struct {
	InterventionID   string    `json:"intervention_id"`
	InterventionType string    `json:"intervention_type"`
	DrugClass        string    `json:"drug_class"`
	DrugName         string    `json:"drug_name"`
	DoseMg           float64   `json:"dose_mg"`
	ActionDate       time.Time `json:"action_date"`
	DaysSince        int       `json:"days_since"`
}
