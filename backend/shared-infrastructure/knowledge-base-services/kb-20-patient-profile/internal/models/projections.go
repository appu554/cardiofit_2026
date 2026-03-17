package models

import "time"

// ────────────────────────────────────────────────────────────────────────────
// FactStore Phase 0: Channel B / Channel C projection DTOs
//
// These structs are the typed contract between KB-20 (data owner) and V-MCU
// (data consumer). The JSON field names match channel_b.RawPatientData and
// channel_c.TitrationContext exactly — the V-MCU runtime deserialises these
// directly into its input structs. No import dependency on V-MCU packages.
// ────────────────────────────────────────────────────────────────────────────

// ChannelBProjection is the typed view KB-20 serves to populate
// channel_b.RawPatientData for V-MCU Channel B evaluation.
type ChannelBProjection struct {
	PatientID string `json:"patient_id"`

	// ── Current lab values (nil = not available) ──
	GlucoseCurrent    *float64   `json:"glucose_current"`
	GlucoseTimestamp  *time.Time `json:"glucose_timestamp"`
	CreatinineCurrent *float64   `json:"creatinine_current"`
	PotassiumCurrent  *float64   `json:"potassium_current"`
	SBPCurrent        *float64   `json:"sbp_current"`
	DBPCurrent        *float64   `json:"dbp_current"`
	WeightKgCurrent   *float64   `json:"weight_kg_current"`
	EGFRCurrent       *float64   `json:"egfr_current"`
	EGFRSlope         *float64   `json:"egfr_slope"`
	HbA1cCurrent      *float64   `json:"hba1c_current"`
	SodiumCurrent     *float64   `json:"sodium_current"`
	HeartRateCurrent  *float64   `json:"heart_rate_current"`

	// ── Historical values (for delta computation) ──
	Creatinine48hAgo *float64 `json:"creatinine_48h_ago"`
	EGFRPrior48h     *float64 `json:"egfr_prior_48h"`
	HbA1cPrior30d    *float64 `json:"hba1c_prior_30d"`
	Weight72hAgo     *float64 `json:"weight_72h_ago"`

	// ── Staleness timestamps (nil = never measured → HOLD_DATA) ──
	Staleness StalenessInfo `json:"staleness"`

	// ── Medication flags ──
	OnRAASAgent                   bool `json:"on_raas_agent"`
	BetaBlockerActive             bool `json:"beta_blocker_active"`
	BetaBlockerDoseChangeIn7d     bool `json:"beta_blocker_dose_change_in_7d"`
	BetaBlockerPerturbationActive bool `json:"beta_blocker_perturbation_active"`
	ThiazideActive                bool `json:"thiazide_active"`

	// ── RAAS creatinine tolerance (PG-14 → B-03 suppression) ──
	CreatinineRiseExplained bool `json:"creatinine_rise_explained"`

	// ── BP context ──
	BPPattern              string  `json:"bp_pattern"`
	MeasurementUncertainty float64 `json:"measurement_uncertainty"`
	SBPLowerLimit          *float64 `json:"sbp_lower_limit"` // J-curve eGFR-stratified floor

	// ── Heart rate context ──
	HRRegularity       string `json:"hr_regularity"`        // REGULAR | IRREGULAR | UNKNOWN
	HRContext          string `json:"hr_context"`            // RESTING | POST_ACTIVITY | STANDING | SUPINE
	HeartRateConfirmed bool   `json:"heart_rate_confirmed"`

	// ── Patient context ──
	Season   string `json:"season"`
	CKDStage string `json:"ckd_stage"`

	// ── Glucose source (CGM | POINT_OF_CARE | LAB | PATIENT_REPORTED) ──
	GlucoseSource string `json:"glucose_source,omitempty"`

	// ── Recent glucose readings (last 3, most recent first) ──
	GlucoseReadings []TimestampedLabValue `json:"glucose_readings"`

	// ── Active treatment perturbations ──
	ActivePerturbations []PerturbationWindow `json:"active_perturbations"`

	// ── FBG Trajectory (Sprint 1 + Track 3) ──
	GlucoseTrajectory     string  `json:"glucose_trajectory,omitempty"`  // STABLE | RISING | RAPID_RISING | DECLINING | IMPROVING
	GlucoseCV             float64 `json:"glucose_cv_pct,omitempty"`     // coefficient of variation %
	GlucoseHighVariability bool   `json:"glucose_high_variability"`     // true if CV > 36% (B-20)

	// ── Perturbation suppression (Track 3) ──
	PerturbationSuppressed bool    `json:"perturbation_suppressed"`
	SuppressionMode        string  `json:"suppression_mode,omitempty"`    // FULL | DAMPENED | TAGGED | NONE
	DominantPerturbation   string  `json:"dominant_perturbation,omitempty"`
	PerturbationGainFactor float64 `json:"perturbation_gain_factor"`     // 0.0 (FULL), 0.5 (DAMPENED), 1.0 (NONE)

	// ── Projection metadata ──
	ProjectedAt time.Time `json:"projected_at"`
}

// ChannelCProjection is the typed view KB-20 serves to populate the
// KB-20-owned fields of channel_c.TitrationContext. Fields that are
// V-MCU-internal (ProposedAction, DoseDeltaPercent, AKIDetected,
// ActiveHypoglycaemia, HypoglycaemiaWithin7d) are NOT included — the
// V-MCU orchestrator fills those from its own state.
type ChannelCProjection struct {
	PatientID string `json:"patient_id"`

	// ── Core data ──
	EGFR              float64  `json:"egfr"`
	ActiveMedications []string `json:"active_medications"` // drug class list

	// ── HTN composite booleans (PG-08..PG-16) ──
	// Pre-computed by KB-20 from labs + medications + trajectory.
	ACEiARBHyperKDecliningEGFR   bool `json:"acei_arb_hyperk_declining_egfr"`    // PG-08
	BetaBlockerInsulinActive     bool `json:"beta_blocker_insulin_active"`       // PG-09
	ResistantHTNDetected         bool `json:"resistant_htn_detected"`            // PG-10
	ThiazideHyponatraemia        bool `json:"thiazide_hyponatraemia"`           // PG-11
	MRAHyperKLowEGFR             bool `json:"mra_hyperk_low_egfr"`              // PG-12
	CCBExcessiveResponse         bool `json:"ccb_excessive_response"`           // PG-13
	RAASCreatinineTolerant       bool `json:"raas_creatinine_tolerant"`         // PG-14
	ACEiInducedCoughProbability  float64 `json:"acei_induced_cough_probability"` // PG-15 (from KB-22 posterior)
	AFConfirmedNoAnticoagulation bool `json:"af_confirmed_no_anticoagulation"`  // PG-16

	// ── Numeric thresholds for rule evaluation ──
	PotassiumCurrent  float64 `json:"potassium_current"`
	SBPCurrent        float64 `json:"sbp_current"`
	SodiumCurrent     float64 `json:"sodium_current"`
	CreatinineRisePct float64 `json:"creatinine_rise_pct"`

	// ── CKD deprescribing guard (AD-09) ──
	CKDStage4DeprescribingBlocked bool `json:"ckd_stage4_deprescribing_blocked"`

	// ── PREVENT risk stratification (Track 2) ──
	PREVENTRiskTier     string  `json:"prevent_risk_tier,omitempty"`      // LOW | BORDERLINE | INTERMEDIATE | HIGH
	PREVENTSBPTarget    float64 `json:"prevent_sbp_target,omitempty"`     // 120 or 130 mmHg
	PREVENT10yrCVD      float64 `json:"prevent_10yr_cvd,omitempty"`       // 0.0-1.0
	PREVENT10yrASCVD    float64 `json:"prevent_10yr_ascvd,omitempty"`     // 0.0-1.0
	PREVENT10yrHF       float64 `json:"prevent_10yr_hf,omitempty"`        // 0.0-1.0
	PREVENTModelUsed    string  `json:"prevent_model_used,omitempty"`     // BASE | HBA1C | UACR | FULL
	OnStatin            bool    `json:"on_statin"`                        // for PG-22 statin gap detection

	// ── Projection metadata ──
	ProjectedAt time.Time `json:"projected_at"`
}

// StalenessInfo carries per-lab-type staleness data for DA-06, DA-07, DA-08 rules.
// Keyed by lab type (EGFR, HBA1C, CREATININE, POTASSIUM).
// nil LastMeasuredAt means "never measured" — Channel B treats this as HOLD_DATA.
type StalenessInfo struct {
	Labs map[string]LabStaleness `json:"labs"`
}

// LabStaleness holds staleness data for a single lab type.
type LabStaleness struct {
	LastMeasuredAt *time.Time `json:"last_measured_at"`
	StaleDays      int        `json:"stale_days"`  // days since last measurement (0 if never measured)
	IsStale        bool       `json:"is_stale"`     // true if exceeds staleness threshold for this lab type
}

// Staleness thresholds (days) per lab type — from KDIGO/ADA guidelines.
const (
	StalenessThresholdEGFR       = 90  // DA-06: eGFR stale after 90 days
	StalenessThresholdHbA1c      = 90  // DA-07: HbA1c stale after 90 days
	StalenessThresholdCreatinine = 14  // DA-08: Creatinine stale after 14 days
	StalenessThresholdPotassium  = 14  // DA-08: Potassium stale after 14 days
)

// TimestampedLabValue pairs a lab measurement with its timestamp.
type TimestampedLabValue struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// PerturbationWindow represents an active treatment perturbation for Channel B dampening.
type PerturbationWindow struct {
	DrugClass       string    `json:"drug_class"`
	ChangeType      string    `json:"change_type"`       // INITIATION | UPTITRATION | SWITCH
	ChangedAt       time.Time `json:"changed_at"`
	WindowExpiresAt time.Time `json:"window_expires_at"` // pharmacodynamic window end
	ExpectedEffect  string    `json:"expected_effect"`   // e.g., "GLUCOSE_DROP", "BP_DROP"
}
