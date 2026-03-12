package models

// FlagNodeDefinition represents a Wave 3.5 FLAG_NODE YAML (e.g. EW-09).
// Unlike NodeDefinition (Bayesian inference), flag nodes use simple
// co-occurrence condition evaluation to generate clinical flags.
// They are loaded from the modifiers/ directory, not nodes/.
type FlagNodeDefinition struct {
	NodeID       string `yaml:"node_id" json:"node_id"`
	Version      string `yaml:"version" json:"version"`
	Name         string `yaml:"name" json:"name"`
	Description  string `yaml:"description" json:"description"`
	TriggerEvent string `yaml:"trigger_event" json:"trigger_event"` // e.g. "BP_STATUS_UPDATE"
	Type         string `yaml:"type" json:"type"`                   // "FLAG_NODE"

	Flags          []FlagCondition      `yaml:"flags" json:"flags"`
	ReservedFields []ReservedDataField  `yaml:"reserved_data_fields,omitempty" json:"reserved_data_fields,omitempty"`
}

// FlagCondition represents a single clinical flag with co-occurrence conditions.
type FlagCondition struct {
	FlagID      string          `yaml:"flag_id" json:"flag_id"`
	Description string          `yaml:"description" json:"description"`
	Conditions  []FlagPredicate `yaml:"conditions" json:"conditions"`
	Action      string          `yaml:"action" json:"action"`     // FLAG_FOR_REVIEW, SPECIALIST_REFERRAL
	Urgency     string          `yaml:"urgency" json:"urgency"`   // 24h, 48h, 72h
	NoteEN      string          `yaml:"note_en" json:"note_en"`
	NoteHI      string          `yaml:"note_hi" json:"note_hi"`
}

// FlagPredicate is a single condition within a flag's co-occurrence check.
type FlagPredicate struct {
	Field    string      `yaml:"field" json:"field"`       // e.g. "symptom_exertional_dyspnoea"
	Operator string      `yaml:"operator" json:"operator"` // eq, in, gte, lte, gt, lt
	Value    interface{} `yaml:"value" json:"value"`        // bool, string, number, or []string for "in"
}

// ReservedDataField documents future KB-20 data dependencies not yet available.
type ReservedDataField struct {
	Field       string `yaml:"field" json:"field"`
	Source      string `yaml:"source" json:"source"`
	Description string `yaml:"description" json:"description"`
}

// FlagNodeResult represents the output of evaluating a FLAG_NODE.
type FlagNodeResult struct {
	NodeID       string       `json:"node_id"`
	TriggerEvent string       `json:"trigger_event"`
	FiredFlags   []FiredFlag  `json:"fired_flags"`
}

// FiredFlag is a single flag that has been triggered by condition evaluation.
type FiredFlag struct {
	FlagID  string `json:"flag_id"`
	Action  string `json:"action"`
	Urgency string `json:"urgency"`
	NoteEN  string `json:"note_en"`
	NoteHI  string `json:"note_hi"`
}
