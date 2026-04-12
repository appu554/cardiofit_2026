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
	PatientID string             `json:"patient_id"`
	Phenotype BPContextPhenotype `json:"phenotype"`

	// BP summaries
	ClinicSBPMean    float64 `json:"clinic_sbp_mean"`
	ClinicDBPMean    float64 `json:"clinic_dbp_mean"`
	HomeSBPMean      float64 `json:"home_sbp_mean"`
	HomeDBPMean      float64 `json:"home_dbp_mean"`
	HomeReadingCount int     `json:"home_reading_count"`

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
