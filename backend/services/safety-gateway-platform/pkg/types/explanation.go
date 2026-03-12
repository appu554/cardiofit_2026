package types

import "time"

// ExplanationLevel defines the level of detail in explanations
type ExplanationLevel string

const (
	ExplanationLevelBasic    ExplanationLevel = "basic"
	ExplanationLevelDetailed ExplanationLevel = "detailed"
	ExplanationLevelExpert   ExplanationLevel = "expert"
)

// Explanation represents a structured explanation of safety decisions
type Explanation struct {
	Level       ExplanationLevel      `json:"level"`
	Summary     string                `json:"summary"`
	Details     []ExplanationDetail   `json:"details"`
	Confidence  float64               `json:"confidence"`
	Evidence    []Evidence            `json:"evidence"`
	Visuals     []ExplanationVisual   `json:"visuals,omitempty"`
	Actionable  []ActionableGuidance  `json:"actionable"`
	GeneratedAt time.Time             `json:"generated_at"`
}

// ExplanationDetail represents a detailed explanation component
type ExplanationDetail struct {
	Category           string  `json:"category"`
	Severity           string  `json:"severity"`
	Description        string  `json:"description"`
	ClinicalRationale  string  `json:"clinical_rationale"`
	Confidence         float64 `json:"confidence"`
	EngineSource       string  `json:"engine_source"`
	RecommendedAction  string  `json:"recommended_action,omitempty"`
}

// Evidence represents supporting evidence for safety decisions
type Evidence struct {
	Type        string                 `json:"type"`
	Source      string                 `json:"source"`
	Description string                 `json:"description"`
	Strength    string                 `json:"strength"`
	URL         string                 `json:"url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExplanationVisual represents visual aids for explanations
type ExplanationVisual struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Data        string `json:"data"` // Base64 encoded or URL
	Format      string `json:"format"`
}

// ActionableGuidance represents actionable guidance for clinicians
type ActionableGuidance struct {
	Action      string   `json:"action"`
	Priority    string   `json:"priority"`
	Steps       []string `json:"steps"`
	Monitoring  []string `json:"monitoring"`
	Timeline    string   `json:"timeline,omitempty"`
	Responsible string   `json:"responsible,omitempty"`
}

// OverrideLevel defines the required authorization level for overrides
type OverrideLevel string

const (
	OverrideLevelResident   OverrideLevel = "resident"
	OverrideLevelAttending  OverrideLevel = "attending"
	OverrideLevelPharmacist OverrideLevel = "pharmacist"
	OverrideLevelChief      OverrideLevel = "chief"
)

// OverrideToken represents a token for overriding unsafe decisions
type OverrideToken struct {
	TokenID         string           `json:"token_id"`
	RequestID       string           `json:"request_id"`
	PatientID       string           `json:"patient_id"`
	DecisionSummary *DecisionSummary `json:"decision_summary"`
	RequiredLevel   OverrideLevel    `json:"required_level"`
	ExpiresAt       time.Time        `json:"expires_at"`
	ContextHash     string           `json:"context_hash"`
	CreatedAt       time.Time        `json:"created_at"`
	Signature       string           `json:"signature"`
}

// DecisionSummary represents a summary of the safety decision
type DecisionSummary struct {
	Status             SafetyStatus `json:"status"`
	CriticalViolations []string     `json:"critical_violations"`
	EnginesFailed      []string     `json:"engines_failed"`
	RiskScore          float64      `json:"risk_score"`
	Explanation        string       `json:"explanation"`
}

// OverrideValidation represents the result of override validation
type OverrideValidation struct {
	Valid       bool           `json:"valid"`
	Reason      string         `json:"reason,omitempty"`
	Token       *OverrideToken `json:"token,omitempty"`
	ClinicianID string         `json:"clinician_id,omitempty"`
	ValidatedAt time.Time      `json:"validated_at"`
}
