package models

import "time"

// PAIScore is the composite Patient Acuity Index output.
type PAIScore struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID       string    `gorm:"size:100;index;not null" json:"patient_id"`
	ComputedAt      time.Time `gorm:"not null" json:"computed_at"`
	Score           float64   `json:"score"`
	Trend           string    `json:"trend"`
	TrendSlopePerHr float64   `json:"trend_slope_per_hr"`
	Tier            string    `json:"tier"`
	VelocityScore   float64   `json:"velocity_score"`
	ProximityScore  float64   `json:"proximity_score"`
	BehavioralScore float64   `json:"behavioral_score"`
	ContextScore    float64   `json:"context_score"`
	AttentionScore  float64   `json:"attention_score"`

	DominantDimension    string  `json:"dominant_dimension"`
	DominantContribution float64 `json:"dominant_contribution"`

	PrimaryReason      string `json:"primary_reason"`
	SuggestedAction    string `json:"suggested_action"`
	SuggestedTimeframe string `json:"suggested_timeframe"`
	EscalationTier     string `json:"escalation_tier"`

	PreviousScore     *float64 `json:"previous_score,omitempty"`
	ScoreDelta        float64  `json:"score_delta"`
	SignificantChange bool     `json:"significant_change"`

	TriggerEvent  string `json:"trigger_event"`
	InputSources  int    `json:"input_sources"`
	DataFreshness string `json:"data_freshness"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (PAIScore) TableName() string { return "pai_scores" }

// PAIDimensionInput carries all inputs needed for PAI computation.
type PAIDimensionInput struct {
	PatientID string

	// Velocity inputs (from domain decomposition + MHRI trajectory)
	MHRICompositeSlope    *float64
	GlucoseDomainSlope    *float64
	CardioDomainSlope     *float64
	BodyCompDomainSlope   *float64
	BehavioralDomainSlope *float64
	SecondDerivative      *string // ACCELERATING_DECLINE, DECELERATING_DECLINE, etc.
	ConcordantDeterioration bool
	DomainsDeterioriating   int

	// Proximity inputs
	CurrentEGFR       *float64
	CurrentHbA1c      *float64
	CurrentSBP        *float64
	CurrentDBP        *float64
	CurrentPotassium  *float64
	CurrentTBRL2Pct   *float64
	CurrentTIR        *float64
	CurrentWeight     *float64
	PreviousWeight72h *float64

	// Behavioral inputs
	EngagementComposite    *float64
	EngagementStatus       string
	DaysSinceLastBPReading int
	DaysSinceLastGlucose   int
	AvgReadingsPerWeek     float64
	CurrentReadingsPerWeek float64
	MeasurementFreqDrop    float64

	// Clinical context inputs
	CKMStage                 string
	HasRecentHospitalization bool
	DaysSinceDischarge       *int
	IsAcutelyIll             bool
	HasRecentHypo            bool
	ActiveSteroidCourse      bool
	IsPostDischarge30d       bool
	MedicationCount          int
	Age                      int
	HFType                   string
	NYHAClass                string

	// Attention gap inputs
	DaysSinceLastClinician   int
	DaysSinceLastCardAck     int
	DaysSinceLastMedChange   int
	HasUnacknowledgedCards   bool
	UnacknowledgedCardCount  int
	OldestUnacknowledgedDays int

	// Confounder context (from V4-8)
	ActiveConfounderScore float64
	SeasonalWindow        bool
}

// PAITier classifies the urgency level.
type PAITier string

const (
	TierCritical PAITier = "CRITICAL"
	TierHigh     PAITier = "HIGH"
	TierModerate PAITier = "MODERATE"
	TierLow      PAITier = "LOW"
	TierMinimal  PAITier = "MINIMAL"
)

// PAIHistory stores PAI snapshots for trend analysis.
type PAIHistory struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID   string    `gorm:"size:100;index;not null" json:"patient_id"`
	Score       float64   `json:"score"`
	Tier        string    `gorm:"size:10" json:"tier"`
	VelocityS   float64   `json:"velocity_score"`
	ProximityS  float64   `json:"proximity_score"`
	BehavioralS float64   `json:"behavioral_score"`
	ContextS    float64   `json:"context_score"`
	AttentionS  float64   `json:"attention_score"`
	TriggerEvt  string    `gorm:"size:50" json:"trigger_event"`
	ComputedAt  time.Time `gorm:"index;not null" json:"computed_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (PAIHistory) TableName() string { return "pai_history" }

// PAIChangeEvent is published to KB-19 when PAI changes significantly.
type PAIChangeEvent struct {
	PatientID       string  `json:"patient_id"`
	NewScore        float64 `json:"new_score"`
	PreviousScore   float64 `json:"previous_score"`
	NewTier         string  `json:"new_tier"`
	PreviousTier    string  `json:"previous_tier"`
	DominantReason  string  `json:"dominant_reason"`
	SuggestedAction string  `json:"suggested_action"`
	Timeframe       string  `json:"timeframe"`
}
