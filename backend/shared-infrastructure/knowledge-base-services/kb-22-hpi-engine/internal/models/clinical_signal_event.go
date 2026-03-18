package models

import "time"

// ClinicalSignalEvent is emitted by PM (Layer 2) and MD (Layer 3) nodes after evaluation.
// It carries both the classification/deterioration result and any safety flags or
// recommended actions that downstream consumers (KB-23, V-MCU) should act on.
type ClinicalSignalEvent struct {
	// Header
	EventID      string     `json:"event_id"`
	EventType    string     `json:"event_type"`    // "CLINICAL_SIGNAL"
	SignalType   SignalType `json:"signal_type"`
	PatientID    string     `json:"patient_id"`
	NodeID       string     `json:"node_id"`
	NodeVersion  string     `json:"node_version"`
	StratumLabel string     `json:"stratum_label"`
	EmittedAt    time.Time  `json:"emitted_at"`

	// Layer 2: Monitoring Classification (PM nodes)
	Classification *ClassificationResult `json:"classification,omitempty"`
	MonitoringData []MonitoringDataPoint `json:"monitoring_data,omitempty"`
	TrendDirection *string               `json:"trend_direction,omitempty"`

	// Layer 3: Deterioration Signal (MD nodes)
	DeteriorationSignal *DeteriorationResult `json:"deterioration_signal,omitempty"`
	ProjectedThreshold  *ThresholdProjection `json:"projected_threshold,omitempty"`
	ContributingSignals []string             `json:"contributing_signals,omitempty"`

	// Shared: Safety + Actions
	SafetyFlags        []SignalSafetyFlag  `json:"safety_flags,omitempty"`
	RecommendedActions []RecommendedAction `json:"recommended_actions,omitempty"`
	AcuityCategory     *string             `json:"acuity_category,omitempty"`
	MCUGateSuggestion  *string             `json:"mcu_gate_suggestion,omitempty"`
}

// SignalType distinguishes PM (monitoring classification) from MD (deterioration signal) events.
type SignalType string

const (
	SignalMonitoringClassification SignalType = "MONITORING_CLASSIFICATION"
	SignalDeteriorationSignal      SignalType = "DETERIORATION_SIGNAL"
)

// SignalSafetyFlag is a safety condition raised by a PM or MD node.
// Severity maps to V-MCU gate logic: IMMEDIATE triggers hard veto, URGENT triggers soft veto, WARN is informational.
type SignalSafetyFlag struct {
	FlagID    string `json:"flag_id"`
	Severity  string `json:"severity"`   // IMMEDIATE | URGENT | WARN
	Action    string `json:"action"`
	Condition string `json:"condition"`
}

// ClassificationResult holds the outcome of a PM node evaluation.
type ClassificationResult struct {
	Category        string  `json:"category"`
	Value           float64 `json:"value"`
	Unit            string  `json:"unit"`
	Threshold       string  `json:"threshold"`
	Confidence      float64 `json:"confidence"`
	DataSufficiency string  `json:"data_sufficiency"`
}

// DeteriorationResult holds the outcome of an MD node threshold evaluation.
type DeteriorationResult struct {
	Signal        string  `json:"signal"`
	Severity      string  `json:"severity"`
	Trajectory    string  `json:"trajectory"`
	RateOfChange  float64 `json:"rate_of_change"`
	StateVariable string  `json:"state_variable"`
}

// ThresholdProjection carries a forward-looking estimate of when a state variable
// will cross a clinically meaningful threshold.
type ThresholdProjection struct {
	ThresholdName  string    `json:"threshold_name"`
	CurrentValue   float64   `json:"current_value"`
	ThresholdValue float64   `json:"threshold_value"`
	ProjectedDate  time.Time `json:"projected_date"`
	Confidence     float64   `json:"confidence"`
}

// RecommendedAction is a suggested clinical or system action produced by a PM or MD node.
type RecommendedAction struct {
	ActionID     string `json:"action_id"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	Urgency      string `json:"urgency"`
	CardTemplate string `json:"card_template"`
}

// MonitoringDataPoint is a single resolved scalar used during PM node evaluation.
// Included in the event so downstream consumers can audit what data was used.
type MonitoringDataPoint struct {
	Field     string    `json:"field"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}
