package models

import "time"

// ConfounderCategory classifies the type of confounder.
type ConfounderCategory string

const (
	ConfounderMedication    ConfounderCategory = "MEDICATION"
	ConfounderAdherence     ConfounderCategory = "ADHERENCE"
	ConfounderSeasonal      ConfounderCategory = "SEASONAL"
	ConfounderReligiousFast ConfounderCategory = "RELIGIOUS_FASTING"
	ConfounderFestivalDiet  ConfounderCategory = "FESTIVAL_DIETARY"
	ConfounderAcuteIllness  ConfounderCategory = "ACUTE_ILLNESS"
	ConfounderIatrogenic    ConfounderCategory = "IATROGENIC"
	ConfounderLifestyle     ConfounderCategory = "LIFESTYLE"
	ConfounderEnvironmental ConfounderCategory = "ENVIRONMENTAL"
)

// ConfounderFactor represents a single active confounder during an outcome window.
type ConfounderFactor struct {
	Category          ConfounderCategory `json:"category"`
	Name              string             `json:"name"`
	Weight            float64            `json:"weight"`
	AffectedOutcomes  []string           `json:"affected_outcomes"`
	ExpectedDirection string             `json:"expected_direction"`
	ExpectedMagnitude string             `json:"expected_magnitude"`
	WindowStart       time.Time          `json:"window_start"`
	WindowEnd         time.Time          `json:"window_end"`
	OverlapDays       int                `json:"overlap_days"`
	OverlapPct        float64            `json:"overlap_pct"`
	Source            string             `json:"source"`
	Confidence        string             `json:"confidence"`
}

// EnhancedConfounderResult is the full output of the enhanced confounder scorer.
type EnhancedConfounderResult struct {
	CompositeScore        float64            `json:"composite_score"`
	ConfidenceLevel       string             `json:"confidence_level"`
	MedicationScore       float64            `json:"medication_score"`
	CalendarScore         float64            `json:"calendar_score"`
	ClinicalEventScore    float64            `json:"clinical_event_score"`
	LifestyleScore        float64            `json:"lifestyle_score"`
	ActiveFactors         []ConfounderFactor `json:"active_factors"`
	FactorCount           int                `json:"factor_count"`
	Narrative             string             `json:"narrative"`
	ShouldDefer           bool               `json:"should_defer"`
	DeferReasonCode       string             `json:"defer_reason_code,omitempty"`
	SuggestedRecheckWeeks int                `json:"suggested_recheck_weeks,omitempty"`
}

// CalendarEvent represents a seasonal/religious/cultural event in the confounder calendar.
type CalendarEvent struct {
	Name              string   `yaml:"name" json:"name"`
	Category          string   `yaml:"category" json:"category"`
	RecurrenceType    string   `yaml:"recurrence_type" json:"recurrence_type"`
	GregorianApprox   []int    `yaml:"gregorian_approx_month" json:"gregorian_approx_month,omitempty"`
	DurationDays      int      `yaml:"duration_days" json:"duration_days"`
	AffectedOutcomes  []string `yaml:"affected_outcomes" json:"affected_outcomes"`
	ExpectedDirection string   `yaml:"expected_direction" json:"expected_direction"`
	ExpectedMagnitude string   `yaml:"expected_magnitude" json:"expected_magnitude"`
	BaseWeight        float64  `yaml:"base_weight" json:"base_weight"`
	PostEventWashout  int      `yaml:"post_event_washout_days" json:"post_event_washout_days"`
	AppliesTo         string   `yaml:"applies_to" json:"applies_to"`
	Notes             string   `yaml:"notes" json:"notes,omitempty"`
}

// ClinicalEventConfounder represents an acute clinical event that confounds outcomes.
type ClinicalEventConfounder struct {
	EventType         string   `json:"event_type"`
	DetectionMethod   string   `json:"detection_method"`
	AffectedOutcomes  []string `json:"affected_outcomes"`
	ExpectedDirection string   `json:"expected_direction"`
	Weight            float64  `json:"weight"`
	WashoutDays       int      `json:"washout_days"`
}

// OutcomeConfounderAnnotation extends an outcome record with confounder detail.
type OutcomeConfounderAnnotation struct {
	OutcomeID          string                   `json:"outcome_id"`
	ConfounderResult   EnhancedConfounderResult `json:"confounder_result"`
	AdjustedConfidence string                   `json:"adjusted_confidence"`
	OriginalDelta      float64                  `json:"original_delta"`
}
