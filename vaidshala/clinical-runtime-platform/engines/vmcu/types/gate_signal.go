// Package types defines shared V-MCU types used across all sub-packages.
// This package breaks import cycles: both arbiter and vmcu_engine import types,
// but neither imports each other directly.
package types

import "time"

// GateSignal represents the 5-state safety gate hierarchy.
// Severity: HALT > HOLD_DATA > PAUSE > MODIFY > CLEAR
type GateSignal string

const (
	GateClear    GateSignal = "CLEAR"
	GateModify   GateSignal = "MODIFY"
	GatePause    GateSignal = "PAUSE"
	GateHoldData GateSignal = "HOLD_DATA"
	GateHalt     GateSignal = "HALT"
)

// Level returns the severity rank for comparison.
func (g GateSignal) Level() int {
	switch g {
	case GateHalt:
		return 4
	case GateHoldData:
		return 3
	case GatePause:
		return 2
	case GateModify:
		return 1
	case GateClear:
		return 0
	default:
		return 0
	}
}

// IsBlocking returns true if this gate prevents dose changes.
func (g GateSignal) IsBlocking() bool {
	return g.Level() >= GatePause.Level()
}

// String implements fmt.Stringer.
func (g GateSignal) String() string {
	return string(g)
}

// ChannelAResult holds the output from Channel A (KB-23 MCU_GATE).
type ChannelAResult struct {
	Gate              GateSignal `json:"gate"`
	CardID            string     `json:"card_id,omitempty"`
	Rationale         string     `json:"rationale,omitempty"`
	DoseAdjustNotes   string     `json:"dose_adjustment_notes,omitempty"`
	GainFactor        float64    `json:"gain_factor"`
	ObsReliability    string     `json:"observation_reliability,omitempty"`
	PerturbationCount int        `json:"perturbation_count"`
}

// ChannelBResult holds the output from Channel B (PhysiologySafetyMonitor).
type ChannelBResult struct {
	Gate       GateSignal         `json:"gate"`
	RuleFired  string             `json:"rule_fired,omitempty"`
	RawValues  map[string]float64 `json:"raw_values,omitempty"`
	IsAnomaly  bool               `json:"is_anomaly"`
	AnomalyLab string             `json:"anomaly_lab,omitempty"`
}

// ChannelCResult holds the output from Channel C (ProtocolGuard).
type ChannelCResult struct {
	Gate         GateSignal `json:"gate"`
	RuleID       string     `json:"rule_id,omitempty"`
	RuleVersion  string     `json:"rule_version"`
	GuidelineRef string     `json:"guideline_ref,omitempty"`
}

// ArbiterInput collects all three channel signals for arbitration.
type ArbiterInput struct {
	MCUGate      GateSignal `json:"mcu_gate"`
	PhysioGate   GateSignal `json:"physio_gate"`
	ProtocolGate GateSignal `json:"protocol_gate"`
}

// ArbiterOutput is the result of the 1oo3 veto arbitration.
type ArbiterOutput struct {
	FinalGate       GateSignal   `json:"final_gate"`
	DominantChannel string       `json:"dominant_channel"`
	AllChannels     ArbiterInput `json:"all_channels"`
	RationaleCode   string       `json:"rationale_code"`
}

// TitrationCycleResult is the complete output of one V-MCU cycle.
type TitrationCycleResult struct {
	PatientID string    `json:"patient_id"`
	Timestamp time.Time `json:"timestamp"`

	ChannelA ChannelAResult `json:"channel_a"`
	ChannelB ChannelBResult `json:"channel_b"`
	ChannelC ChannelCResult `json:"channel_c"`

	Arbiter ArbiterOutput `json:"arbiter"`

	DoseApplied *float64 `json:"dose_applied,omitempty"`
	DoseDelta   *float64 `json:"dose_delta,omitempty"`
	BlockedBy   string   `json:"blocked_by,omitempty"`
}
