package models

import "time"

// SafetyActionType defines the type of safety action to take
type SafetyActionType string

const (
	ActionContraindicate    SafetyActionType = "contraindicate"
	ActionModifyDose        SafetyActionType = "modify_dose"
	ActionRequireMonitoring SafetyActionType = "require_monitoring"
	ActionSubstituteTherapy SafetyActionType = "substitute_therapy"
	ActionManualReview      SafetyActionType = "manual_review"
)

// SafetyOverride represents a safety override rule
type SafetyOverride struct {
	OverrideID         string                  `json:"override_id"`
	Name               string                  `json:"name"`
	Description        string                  `json:"description"`
	TriggerConditions  SafetyTriggerConditions `json:"trigger_conditions"`
	OverrideAction     SafetyAction            `json:"override_action"`
	Priority           int                     `json:"priority"`
	Active             bool                    `json:"active"`
	AffectedGuidelines []string                `json:"affected_guidelines"`
	EffectiveDate      time.Time               `json:"effective_date"`
	ExpiryDate         *time.Time              `json:"expiry_date,omitempty"`
	RequiresSignature  bool                    `json:"requires_signature"`
	CreatedBy          string                  `json:"created_by"`
	ClinicalRationale  string                  `json:"clinical_rationale"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

// SafetyTriggerConditions defines when a safety override applies
type SafetyTriggerConditions struct {
	Pregnancy                *bool                   `json:"pregnancy,omitempty"`
	Pediatric                *bool                   `json:"pediatric,omitempty"`
	Geriatric                *bool                   `json:"geriatric,omitempty"`
	Conditions               []string                `json:"conditions,omitempty"`
	Medications              []string                `json:"medications,omitempty"`
	LabThresholds            map[string]LabThreshold `json:"lab_thresholds,omitempty"`
	AllergyContraindications []string                `json:"allergy_contraindications,omitempty"`
	SeverityThreshold        string                  `json:"severity_threshold,omitempty"`
	ClinicalContext          []string                `json:"clinical_context,omitempty"`
}

// LabThreshold defines a lab value threshold for triggering
type LabThreshold struct {
	Operator string  `json:"operator"` // >, >=, <, <=, =
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
}

// SafetyAction defines what action to take when triggered
type SafetyAction struct {
	ActionType                 SafetyActionType `json:"action_type"`
	Description                string           `json:"description"`
	Parameters                 map[string]any   `json:"parameters,omitempty"`
	MonitoringRequirements     []string         `json:"monitoring_requirements,omitempty"`
	AlternativeRecommendations []string         `json:"alternative_recommendations,omitempty"`
	EscalationRequired         bool             `json:"escalation_required,omitempty"`
}

// SafetyAssessment result from evaluating patient safety
type SafetyAssessment struct {
	PatientID               string                   `json:"patient_id"`
	SafetyScore             int                      `json:"safety_score"` // 0-100
	RiskLevel               string                   `json:"risk_level"`   // low, moderate, high, critical
	RiskFactors             []string                 `json:"risk_factors"`
	Contraindications       []Contraindication       `json:"contraindications"`
	Warnings                []Warning                `json:"warnings"`
	RequiredMonitoring      []string                 `json:"required_monitoring"`
	OverrideRecommendations []OverrideRecommendation `json:"override_recommendations"`
	AssessedAt              time.Time                `json:"assessed_at"`
}

// Contraindication represents a contraindicated action
type Contraindication struct {
	GuidelineID string `json:"guideline_id"`
	Reason      string `json:"reason"`
	Severity    string `json:"severity"`
	OverrideID  string `json:"override_id,omitempty"`
}

// Warning represents a clinical warning
type Warning struct {
	Type        string `json:"type"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	ActionItems []string `json:"action_items,omitempty"`
}

// OverrideRecommendation provides alternative recommendations
type OverrideRecommendation struct {
	Original    string   `json:"original"`
	Alternative string   `json:"alternative"`
	Rationale   string   `json:"rationale"`
	Evidence    string   `json:"evidence,omitempty"`
}

// SafetyOverrideLog for audit trail
type SafetyOverrideLog struct {
	LogID          string                 `json:"log_id"`
	OverrideID     string                 `json:"override_id"`
	PatientID      string                 `json:"patient_id"`
	PatientContext map[string]interface{} `json:"patient_context"`
	ActionTaken    string                 `json:"action_taken"`
	Rationale      string                 `json:"rationale"`
	AppliedBy      string                 `json:"applied_by"`
	AppliedAt      time.Time              `json:"applied_at"`
	Signature      string                 `json:"signature,omitempty"`
}

// CalculateSafetyScore computes safety score from assessment
// Formula: 100 - (25 × contraindications) - (10 × warnings) - (5 × risk factors)
func CalculateSafetyScore(contraindications, warnings, riskFactors int) int {
	score := 100
	score -= contraindications * 25
	score -= warnings * 10
	score -= riskFactors * 5
	if score < 0 {
		score = 0
	}
	return score
}

// GetRiskLevel returns risk level based on safety score
func GetRiskLevel(score int) string {
	switch {
	case score >= 80:
		return "low"
	case score >= 60:
		return "moderate"
	case score >= 40:
		return "high"
	default:
		return "critical"
	}
}
