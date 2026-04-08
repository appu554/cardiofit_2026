// Package models defines the domain types for the KB-24 Safety Constraint Engine.
// These types are intentionally decoupled from KB-22 models to allow independent evolution.
package models

import "time"

// SafetyLevel represents the severity classification of a safety flag.
type SafetyLevel string

const (
	SafetyImmediate SafetyLevel = "IMMEDIATE" // requires action within 30 minutes
	SafetyUrgent    SafetyLevel = "URGENT"    // requires action within 4 hours
	SafetyWarn      SafetyLevel = "WARN"      // flag for next appointment
)

// SafetyFlag records a single fired safety trigger evaluation result.
type SafetyFlag struct {
	FlagID            string      `json:"flag_id"`
	Severity          SafetyLevel `json:"severity"`
	RecommendedAction string      `json:"recommended_action"`
	FiredAt           time.Time   `json:"fired_at"`
}

// SafetyTriggerDef defines a safety condition loaded from node YAML definitions.
// Supports BOOLEAN conditions (Q001=YES AND Q003=YES) and COMPOSITE_SCORE triggers.
type SafetyTriggerDef struct {
	ID        string `yaml:"id" json:"id"`
	Type      string `yaml:"type,omitempty" json:"type,omitempty"`           // BOOLEAN (default) or COMPOSITE_SCORE
	RuleType  string `yaml:"rule_type,omitempty" json:"rule_type,omitempty"` // HARD_STOP or SOFT_FLAG (intake context)
	Condition string `yaml:"condition" json:"condition"`                     // Boolean expression: 'Q001=YES AND Q003=YES'
	Severity  string `yaml:"severity" json:"severity"`                       // IMMEDIATE | URGENT | WARN
	Action    string `yaml:"action" json:"action"`

	// COMPOSITE_SCORE fields: weighted question-answer scoring.
	// Keys are "QUESTION_ID=ANSWER_VALUE" pairs, values are weights.
	Weights   map[string]float64 `yaml:"weights,omitempty" json:"weights,omitempty"`
	Threshold float64            `yaml:"threshold,omitempty" json:"threshold,omitempty"`
}

// NodeDefinition is a minimal projection of the KB-22 node YAML, containing
// only the fields required by the Safety Constraint Engine.
type NodeDefinition struct {
	NodeID         string              `yaml:"node_id" json:"node_id"`
	SafetyTriggers []SafetyTriggerDef  `yaml:"safety_triggers" json:"safety_triggers"`
}
