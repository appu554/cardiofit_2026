package models

// MonitoringNodeDefinition represents a parsed PM node YAML (Layer 2).
type MonitoringNodeDefinition struct {
	NodeID           string                    `yaml:"node_id" json:"node_id"`
	Version          string                    `yaml:"version" json:"version"`
	Type             string                    `yaml:"type" json:"type"` // always "MONITORING"
	TitleEN          string                    `yaml:"title_en" json:"title_en"`
	TitleHI          string                    `yaml:"title_hi" json:"title_hi"`
	RequiredInputs   []RequiredInput           `yaml:"required_inputs" json:"required_inputs"`
	AggregatedInputs []AggregatedInputDef      `yaml:"aggregated_inputs,omitempty" json:"aggregated_inputs,omitempty"`
	ComputedFields   []ComputedFieldDef        `yaml:"computed_fields,omitempty" json:"computed_fields,omitempty"`
	Classifications  []ClassificationDef       `yaml:"classifications" json:"classifications"`
	InsufficientData InsufficientDataPolicy    `yaml:"insufficient_data" json:"insufficient_data"`
	SafetyTriggers   []MonitoringSafetyTrigger `yaml:"safety_triggers,omitempty" json:"safety_triggers,omitempty"`
	CascadeTo        []string                  `yaml:"cascade_to,omitempty" json:"cascade_to,omitempty"`
	CheckinPrompts   []CheckinPromptDef        `yaml:"checkin_prompts,omitempty" json:"checkin_prompts,omitempty"`
}

// ComputedFieldDef defines a derived field calculated from resolved inputs.
type ComputedFieldDef struct {
	Name    string `yaml:"name" json:"name"`
	Formula string `yaml:"formula" json:"formula"`
}

// ClassificationDef defines a classification rule evaluated top-to-bottom (first match wins).
type ClassificationDef struct {
	Category          string `yaml:"category" json:"category"`
	Condition         string `yaml:"condition" json:"condition"`
	Severity          string `yaml:"severity" json:"severity"`
	MCUGateSuggestion string `yaml:"mcu_gate_suggestion" json:"mcu_gate_suggestion"`
	CardTemplate      string `yaml:"card_template,omitempty" json:"card_template,omitempty"`
}

// InsufficientDataPolicy defines behavior when required data is missing.
type InsufficientDataPolicy struct {
	Action   string `yaml:"action" json:"action"` // SKIP | FLAG_FOR_REVIEW
	NoteEN   string `yaml:"note_en,omitempty" json:"note_en,omitempty"`
	Fallback string `yaml:"fallback,omitempty" json:"fallback,omitempty"`
}

// MonitoringSafetyTrigger defines a safety condition that always evaluates (even during debounce).
type MonitoringSafetyTrigger struct {
	ID        string `yaml:"id" json:"id"`
	Condition string `yaml:"condition" json:"condition"`
	Severity  string `yaml:"severity" json:"severity"`
	Action    string `yaml:"action" json:"action"`
}

// CheckinPromptDef defines a Tier-1 check-in prompt for patient self-reporting.
type CheckinPromptDef struct {
	PromptID     string `yaml:"prompt_id" json:"prompt_id"`
	TextEN       string `yaml:"text_en" json:"text_en"`
	TextHI       string `yaml:"text_hi" json:"text_hi"`
	ResponseType string `yaml:"response_type" json:"response_type"` // BOOLEAN | SCALE | NUMERIC
	MapsTo       string `yaml:"maps_to" json:"maps_to"`             // field name in required_inputs
}
