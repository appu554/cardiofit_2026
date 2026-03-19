package models

// ClinicalSignalEvent is the inbound event from KB-22 SignalPublisher.
// Mirrors KB-22's event model for deserialization.
// Also accepts MRI_DETERIORATION events published directly by KB-26.
type ClinicalSignalEvent struct {
	EventID             string                `json:"event_id"`
	PatientID           string                `json:"patient_id"`
	NodeID              string                `json:"node_id"`
	StratumLabel        string                `json:"stratum_label"`
	SignalType          string                `json:"signal_type"` // MONITORING_CLASSIFICATION, DETERIORATION_SIGNAL, or MRI_DETERIORATION
	EvaluatedAt         string                `json:"evaluated_at"`
	Classification      *ClassificationResult `json:"classification,omitempty"`
	DeteriorationSignal *DeteriorationResult  `json:"deterioration_signal,omitempty"`
	MCUGateSuggestion   *string               `json:"mcu_gate_suggestion,omitempty"`
	SafetyFlags         []SignalSafetyFlag    `json:"safety_flags,omitempty"`
	ContributingSignals map[string]float64    `json:"contributing_signals,omitempty"`
	ProjectedThreshold  *ThresholdProjection  `json:"projected_threshold,omitempty"`

	// MRI_DETERIORATION fields — top-level fields sent by KB-26 MRIEventPublisher.
	// Populated when SignalType == "MRI_DETERIORATION".
	MRICategory  string  `json:"category,omitempty"`
	MRISeverity  string  `json:"severity,omitempty"`
	MRIScore     float64 `json:"score,omitempty"`
	MRITopDriver string  `json:"top_driver,omitempty"`
	MRITrend     string  `json:"trend,omitempty"`
}

// ClassificationResult holds a monitoring classification from a PM node.
type ClassificationResult struct {
	Category        string             `json:"category"`
	DataSufficiency string             `json:"data_sufficiency"`
	ComputedFields  map[string]float64 `json:"computed_fields,omitempty"`
}

// DeteriorationResult holds a deterioration signal from an MD node.
type DeteriorationResult struct {
	Signal               string            `json:"signal"`
	Severity             string            `json:"severity"`
	Trajectory           string            `json:"trajectory"`
	RateOfChange         float64           `json:"rate_of_change"`
	TrajectoryConfidence float64           `json:"trajectory_confidence"`
	MCUGateSuggestion    string            `json:"mcu_gate_suggestion"`
	Actions              []RecommendedAction `json:"actions,omitempty"`
}

// RecommendedAction is a suggested action from a deterioration signal.
type RecommendedAction struct {
	Action   string `json:"action"`
	Priority string `json:"priority"`
}

// SignalSafetyFlag is a safety flag raised during signal evaluation.
type SignalSafetyFlag struct {
	FlagID    string `json:"flag_id"`
	Severity  string `json:"severity"`
	Action    string `json:"action"`
	Condition string `json:"condition"`
}

// ThresholdProjection projects when a monitored value will cross a threshold.
type ThresholdProjection struct {
	ThresholdName  string  `json:"threshold_name"`
	CurrentValue   float64 `json:"current_value"`
	ThresholdValue float64 `json:"threshold_value"`
	ProjectedDate  string  `json:"projected_date"`
	Confidence     float64 `json:"confidence"`
}
