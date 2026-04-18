package models

import (
	"time"

	"github.com/google/uuid"
)

// TemporalClassification describes the temporal pattern of a deviation.
type TemporalClassification string

const (
	TemporalSpike       TemporalClassification = "SPIKE"       // single reading, needs confirmation
	TemporalTrend       TemporalClassification = "TREND"        // 3+ readings same direction
	TemporalPersistence TemporalClassification = "PERSISTENCE"  // sustained >24h or 5+ readings
)

// AcuteEventType classifies the type of acute deterioration.
type AcuteEventType string

const (
	AcuteKidneyInjury            AcuteEventType = "ACUTE_KIDNEY_INJURY"
	AcuteFluidOverload           AcuteEventType = "FLUID_OVERLOAD"
	AcuteHypertensiveEmergency   AcuteEventType = "HYPERTENSIVE_EMERGENCY"
	AcuteSevereHypoglycaemia     AcuteEventType = "SEVERE_HYPOGLYCAEMIA"
	AcuteSevereHyperglycaemia    AcuteEventType = "SEVERE_HYPERGLYCAEMIA"
	AcuteCompoundCardiorenal     AcuteEventType = "COMPOUND_CARDIORENAL"
	AcuteCompoundInfection       AcuteEventType = "COMPOUND_INFECTION_CASCADE"
	AcuteMedicationCrisis        AcuteEventType = "MEDICATION_INDUCED_CRISIS"
	AcuteMeasurementGapDeviation AcuteEventType = "MEASUREMENT_GAP_DEVIATION"
)

// AcuteSeverity classifies how dangerous the acute event is.
type AcuteSeverity string

const (
	SeverityCritical AcuteSeverity = "CRITICAL"
	SeverityHigh     AcuteSeverity = "HIGH"
	SeverityModerate AcuteSeverity = "MODERATE"
)

// AcuteEvent is persisted for every detected acute-on-chronic deterioration.
type AcuteEvent struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID          string     `gorm:"size:100;index:idx_acute_patient,priority:1;not null" json:"patient_id"`
	DetectedAt         time.Time  `gorm:"index:idx_acute_patient,priority:2,sort:desc;not null" json:"detected_at"`
	EventType          string     `gorm:"size:40;not null" json:"event_type"`
	Severity           string     `gorm:"size:10;not null" json:"severity"`
	VitalSignType      string     `gorm:"size:20" json:"vital_sign_type"`
	CurrentValue       float64    `json:"current_value"`
	BaselineMedian     float64    `json:"baseline_median"`
	DeviationPercent   float64    `json:"deviation_percent"`
	DeviationAbsolute  float64    `json:"deviation_absolute"`
	Direction          string     `gorm:"size:20" json:"direction"`
	CompoundPattern    string     `gorm:"size:40" json:"compound_pattern,omitempty"`
	MedicationContext  string     `gorm:"type:text" json:"medication_context,omitempty"`
	ConfounderContext  string     `gorm:"size:100" json:"confounder_context,omitempty"`
	GapAmplified       bool       `gorm:"default:false" json:"gap_amplified"`
	ConfounderDampened     bool       `gorm:"default:false" json:"confounder_dampened"`
	TemporalClassification string    `gorm:"size:15" json:"temporal_classification,omitempty"`
	EscalationTier         string    `gorm:"size:20" json:"escalation_tier"`
	SuggestedAction    string     `gorm:"type:text" json:"suggested_action"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
	ResolutionType     string     `gorm:"size:20" json:"resolution_type,omitempty"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (AcuteEvent) TableName() string { return "acute_events" }

// PatientBaselineSnapshot stores the rolling baseline for one vital sign.
type PatientBaselineSnapshot struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID      string    `gorm:"size:100;uniqueIndex:uq_baseline_patient_vital,priority:1;not null" json:"patient_id"`
	VitalSignType  string    `gorm:"size:20;uniqueIndex:uq_baseline_patient_vital,priority:2;not null" json:"vital_sign_type"`
	BaselineMedian float64   `json:"baseline_median"`
	BaselineMAD    float64   `json:"baseline_mad"`
	ReadingCount   int       `json:"reading_count"`
	Confidence     string    `gorm:"size:10" json:"confidence"`
	LookbackDays   int       `json:"lookback_days"`
	UsualMeasurementHour int       `json:"usual_measurement_hour"`
	ComputedAt           time.Time `gorm:"not null" json:"computed_at"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PatientBaselineSnapshot) TableName() string { return "patient_baselines" }

// ValidationState classifies the validation status of a deviation reading.
type ValidationState string

const (
	ValidationConfirmed              ValidationState = "CONFIRMED"
	ValidationUnconfirmed            ValidationState = "UNCONFIRMED"
	ValidationAwaitingConfirmation   ValidationState = "AWAITING_CONFIRMATION"
	ValidationUnconfirmedCritical    ValidationState = "UNCONFIRMED_CRITICAL"
	ValidationNotApplicable          ValidationState = "NOT_APPLICABLE"
	ValidationRefuted                ValidationState = "REFUTED"
	ValidationExpired                ValidationState = "EXPIRED_UNCONFIRMED"
)

// PendingValidation tracks a weight reading awaiting confirmation.
type PendingValidation struct {
	ID                  string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID           string    `gorm:"size:100;index;not null" json:"patient_id"`
	VitalSignType       string    `gorm:"size:20;not null" json:"vital_sign_type"`
	OriginalValue       float64   `json:"original_value"`
	OriginalDeviation   float64   `json:"original_deviation"`
	OriginalReadingTime time.Time `json:"original_reading_time"`
	ExpiresAt           time.Time `gorm:"index;not null" json:"expires_at"`
	ConfirmationValue   *float64  `json:"confirmation_value,omitempty"`
	ValidationOutcome   string    `gorm:"size:30" json:"validation_outcome,omitempty"`
	CreatedAt           time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PendingValidation) TableName() string { return "pending_validations" }

// DeviationResult is the output of a single vital sign deviation check.
type DeviationResult struct {
	VitalSignType        string  `json:"vital_sign_type"`
	CurrentValue         float64 `json:"current_value"`
	BaselineMedian       float64 `json:"baseline_median"`
	BaselineMAD          float64 `json:"baseline_mad"`
	DeviationAbsolute    float64 `json:"deviation_absolute"`
	DeviationPercent     float64 `json:"deviation_percent"`
	Direction            string  `json:"direction"`
	ClinicalSignificance string  `json:"clinical_significance"`
	GapAmplified         bool    `json:"gap_amplified"`
	ConfounderDampened     bool    `json:"confounder_dampened"`
	RawSeverity            string  `json:"raw_severity,omitempty"`
	EffectiveSeverity      string  `json:"effective_severity,omitempty"`
	TemporalClassification string  `json:"temporal_classification,omitempty"`
	ValidationState        string  `json:"validation_state,omitempty"`
	ValidationReason     string  `json:"validation_reason,omitempty"`
}

// CompoundPatternMatch describes a multi-vital-sign syndrome detection.
type CompoundPatternMatch struct {
	PatternName         string            `json:"pattern_name"`
	MatchedDeviations   []DeviationResult `json:"matched_deviations"`
	PatternConfidence   string            `json:"pattern_confidence"`
	ClinicalSyndrome    string            `json:"clinical_syndrome"`
	RecommendedResponse string            `json:"recommended_response"`
	CompoundSeverity    string            `json:"compound_severity"`
	ClinicianLabel      string            `json:"clinician_label"`
	PatientLabel        string            `json:"patient_label"`
}
