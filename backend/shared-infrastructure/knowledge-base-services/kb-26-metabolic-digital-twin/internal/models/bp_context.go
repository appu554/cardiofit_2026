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
//
// Phenotype is the *stable* phenotype after the stability engine has had its
// say. RawPhenotype (Phase 5 P5-1) is the raw classifier output before the
// engine intervened — when Phenotype != RawPhenotype, the engine damped a
// transition. Storing both lets clinicians and downstream logic see "the
// algorithm thinks X but we're holding at Y," and lets the engine compute a
// raw-disagreement rate to override the dwell when warranted (see
// stability.Policy.MaxDwellOverrideRate). Older snapshots written before
// Phase 5 P5-1 carry an empty RawPhenotype.
type BPContextHistory struct {
	ID            string             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID     string             `gorm:"size:100;not null;uniqueIndex:idx_bp_ctx_patient_date,priority:1" json:"patient_id"`
	SnapshotDate  time.Time          `gorm:"not null;uniqueIndex:idx_bp_ctx_patient_date,priority:2" json:"snapshot_date"`
	Phenotype     BPContextPhenotype `gorm:"size:30;not null" json:"phenotype"`
	RawPhenotype  BPContextPhenotype `gorm:"size:30" json:"raw_phenotype,omitempty"`
	ClinicSBPMean float64            `json:"clinic_sbp_mean"`
	HomeSBPMean   float64            `json:"home_sbp_mean"`
	GapSBP        float64            `json:"gap_sbp"`
	Confidence    string             `gorm:"size:10" json:"confidence"`
	CreatedAt     time.Time          `json:"created_at"`
}
