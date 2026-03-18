package models

// DeteriorationNodeDefinition represents a parsed MD node YAML (Layer 3).
type DeteriorationNodeDefinition struct {
	NodeID                string                 `yaml:"node_id" json:"node_id"`
	Version               string                 `yaml:"version" json:"version"`
	Type                  string                 `yaml:"type" json:"type"` // always "DETERIORATION"
	TitleEN               string                 `yaml:"title_en" json:"title_en"`
	TitleHI               string                 `yaml:"title_hi" json:"title_hi"`
	StateVariable         string                 `yaml:"state_variable,omitempty" json:"state_variable,omitempty"`
	StateVariableLabel    string                 `yaml:"state_variable_label,omitempty" json:"state_variable_label,omitempty"`
	TriggerOn             []TriggerDef           `yaml:"trigger_on" json:"trigger_on"`
	RequiredInputs        []RequiredInput        `yaml:"required_inputs" json:"required_inputs"`
	AggregatedInputs      []AggregatedInputDef   `yaml:"aggregated_inputs,omitempty" json:"aggregated_inputs,omitempty"`
	ComputedFields        []ComputedFieldDef     `yaml:"computed_fields,omitempty" json:"computed_fields,omitempty"`
	ComputedFieldVariants []ComputedFieldVariant `yaml:"computed_field_variants,omitempty" json:"computed_field_variants,omitempty"`
	ContributingSignals   []string               `yaml:"contributing_signals,omitempty" json:"contributing_signals,omitempty"`
	Trajectory            *TrajectoryConfig      `yaml:"trajectory,omitempty" json:"trajectory,omitempty"`
	Thresholds            []ThresholdDef         `yaml:"thresholds" json:"thresholds"`
	Projections           []ProjectionDef        `yaml:"projections,omitempty" json:"projections,omitempty"`
	InsufficientData      InsufficientDataPolicy `yaml:"insufficient_data" json:"insufficient_data"`
}

// TriggerDef defines what events cause this MD node to be evaluated.
type TriggerDef struct {
	Event string `yaml:"event" json:"event"` // e.g., "OBSERVATION:FBG", "SIGNAL:PM-04", "PROTOCOL:M3-PRP:ADHERENCE"
}

// TrajectoryConfig defines how trajectory computation is performed.
type TrajectoryConfig struct {
	Method        string `yaml:"method" json:"method"` // LINEAR_REGRESSION
	WindowDays    int    `yaml:"window_days" json:"window_days"`
	MinDataPoints int    `yaml:"min_data_points" json:"min_data_points"`
	RateUnit      string `yaml:"rate_unit" json:"rate_unit"`
	DataSource    string `yaml:"data_source" json:"data_source"`
}

// ThresholdDef defines a deterioration threshold evaluated top-to-bottom (first match wins).
type ThresholdDef struct {
	Signal            string              `yaml:"signal" json:"signal"`
	Condition         string              `yaml:"condition" json:"condition"`
	Severity          string              `yaml:"severity" json:"severity"`
	Trajectory        string              `yaml:"trajectory" json:"trajectory"`
	MCUGateSuggestion string              `yaml:"mcu_gate_suggestion" json:"mcu_gate_suggestion"`
	CardTemplate      string              `yaml:"card_template,omitempty" json:"card_template,omitempty"`
	Actions           []RecommendedAction `yaml:"actions,omitempty" json:"actions,omitempty"`
}

// ProjectionDef defines a forward-looking projection for a state variable.
type ProjectionDef struct {
	Name               string  `yaml:"name" json:"name"`
	Variable           string  `yaml:"variable" json:"variable"`
	Threshold          float64 `yaml:"threshold" json:"threshold"`
	Method             string  `yaml:"method" json:"method"` // LINEAR_EXTRAPOLATION
	ConfidenceRequired float64 `yaml:"confidence_required" json:"confidence_required"`
}

// ComputedFieldVariant defines a conditional formula variant for adaptive weight computation.
// Used by MD-04 which has different weight formulas depending on which PM inputs are available.
// DeteriorationNodeEngine evaluates conditions top-to-bottom, first match wins.
type ComputedFieldVariant struct {
	Condition string `yaml:"condition" json:"condition"`
	Name      string `yaml:"name" json:"name"`
	Formula   string `yaml:"formula" json:"formula"`
}
