package models

type LSRule struct {
	Code        string `json:"code"`
	Condition   string `json:"condition"`
	Blocked     string `json:"blocked"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type InteractionEntry struct {
	LifestyleCode string `json:"lifestyle_code"`
	DrugClassCode string `json:"drug_class_code"`
	Interaction   string `json:"interaction"`
	Severity      string `json:"severity"`
	Action        string `json:"action"`
	Description   string `json:"description"`
}

type SafetyCheckRequest struct {
	PatientID     string   `json:"patient_id" binding:"required"`
	Interventions []string `json:"interventions" binding:"required"`
	Medications   []string `json:"medications,omitempty"`
}

type SafetyCheckResult struct {
	Safe         bool               `json:"safe"`
	Violations   []SafetyViolation  `json:"violations,omitempty"`
	Warnings     []SafetyViolation  `json:"warnings,omitempty"`
	Interactions []InteractionEntry `json:"interactions,omitempty"`
}

type SafetyViolation struct {
	RuleCode    string `json:"rule_code"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Blocked     string `json:"blocked"`
}
