// Package trace implements the SafetyTrace audit system (SA-04).
//
// Every titration cycle produces one SafetyTrace record.
// Records are APPEND-ONLY — no UPDATE or DELETE operations.
// 10-year retention for DISHA compliance.
package trace

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// SafetyTrace is an append-only audit record for one titration cycle.
type SafetyTrace struct {
	TraceID        string    `json:"trace_id"`
	PatientID      string    `json:"patient_id"`
	CycleTimestamp time.Time `json:"cycle_timestamp"`

	MCUGate          string `json:"mcu_gate"`
	MCUGateCardID    string `json:"mcu_gate_card_id,omitempty"`
	MCUGateRationale string `json:"mcu_gate_rationale,omitempty"`

	PhysioGate      string          `json:"physio_gate"`
	PhysioRuleFired string          `json:"physio_rule_fired,omitempty"`
	PhysioRawValues json.RawMessage `json:"physio_raw_values,omitempty"`

	ProtocolGate     string `json:"protocol_gate"`
	ProtocolRuleID   string `json:"protocol_rule_id,omitempty"`
	ProtocolRuleVsn  string `json:"protocol_rule_vsn"`
	ProtocolGuideRef string `json:"protocol_guide_ref,omitempty"`

	FinalGate        string `json:"final_gate"`
	DominantChannel  string `json:"dominant_channel,omitempty"`
	ArbiterRationale string `json:"arbiter_rationale,omitempty"`

	DoseApplied *float64 `json:"dose_applied,omitempty"`
	DoseDelta   *float64 `json:"dose_delta,omitempty"`
	BlockedBy   string   `json:"blocked_by,omitempty"`

	ObservationReliability string  `json:"observation_reliability,omitempty"`
	GainFactor             float64 `json:"gain_factor"`

	// ── HTN co-management audit fields (Wave 1) ──
	RAASCausalSuppression bool    `json:"raas_causal_suppression,omitempty"` // true if B-03 was downgraded by PG-14
	AppliedSBPLowerLimit  *float64 `json:"applied_sbp_lower_limit,omitempty"` // J-curve floor used for B-12
	SourceEGFRForThreshold *float64 `json:"source_egfr_for_threshold,omitempty"` // eGFR that determined the J-curve floor
	BPPatternApplied      string   `json:"bp_pattern_applied,omitempty"`       // BPPattern in effect during this cycle
	HTNProtocolRulesFired []string `json:"htn_protocol_rules_fired,omitempty"` // PG-08..PG-14 rules that evaluated true
}

// TraceWriter writes SafetyTrace records.
type TraceWriter struct {
	traces []SafetyTrace
}

// NewTraceWriter creates a new trace writer.
func NewTraceWriter() *TraceWriter {
	return &TraceWriter{}
}

// Record creates a SafetyTrace from a completed titration cycle.
func (w *TraceWriter) Record(result *vt.TitrationCycleResult) SafetyTrace {
	rawVals, _ := json.Marshal(result.ChannelB.RawValues)

	st := SafetyTrace{
		TraceID:        generateTraceID(),
		PatientID:      result.PatientID,
		CycleTimestamp: result.Timestamp,

		MCUGate:          string(result.ChannelA.Gate),
		MCUGateCardID:    result.ChannelA.CardID,
		MCUGateRationale: result.ChannelA.Rationale,

		PhysioGate:      string(result.ChannelB.Gate),
		PhysioRuleFired: result.ChannelB.RuleFired,
		PhysioRawValues: rawVals,

		ProtocolGate:     string(result.ChannelC.Gate),
		ProtocolRuleID:   result.ChannelC.RuleID,
		ProtocolRuleVsn:  result.ChannelC.RuleVersion,
		ProtocolGuideRef: result.ChannelC.GuidelineRef,

		FinalGate:        string(result.Arbiter.FinalGate),
		DominantChannel:  result.Arbiter.DominantChannel,
		ArbiterRationale: result.Arbiter.RationaleCode,

		DoseApplied: result.DoseApplied,
		DoseDelta:   result.DoseDelta,
		BlockedBy:   result.BlockedBy,

		ObservationReliability: result.ChannelA.ObsReliability,
		GainFactor:             result.ChannelA.GainFactor,
	}

	w.traces = append(w.traces, st)
	return st
}

// Flush returns all pending traces and clears the buffer.
func (w *TraceWriter) Flush() []SafetyTrace {
	out := w.traces
	w.traces = nil
	return out
}

// PendingCount returns the number of un-flushed traces.
func (w *TraceWriter) PendingCount() int {
	return len(w.traces)
}

func generateTraceID() string {
	return fmt.Sprintf("%s-%s", time.Now().Format("20060102T150405.000"), randomSuffix())
}

func randomSuffix() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
