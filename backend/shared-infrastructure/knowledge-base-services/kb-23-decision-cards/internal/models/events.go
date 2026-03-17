package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// KB19Event is the payload published to KB-19 via POST /api/v1/events.
type KB19Event struct {
	EventType   EventType  `json:"event_type"`
	PatientID   uuid.UUID  `json:"patient_id"`
	SessionID   *uuid.UUID `json:"session_id,omitempty"`
	CardID      uuid.UUID  `json:"card_id"`

	// MCU_GATE_CHANGED fields
	Gate                MCUGate  `json:"gate,omitempty"`
	PreviousGate        *MCUGate `json:"previous_gate,omitempty"`
	ReEntryProtocol     bool     `json:"re_entry_protocol,omitempty"`
	HaltDurationHours   float64  `json:"halt_duration_hours,omitempty"`
	ReEntryPhase1Hours  float64  `json:"re_entry_phase1_hours,omitempty"`
	ReEntryPhase2Hours  float64  `json:"re_entry_phase2_hours,omitempty"`
	DoseAdjustmentNotes string   `json:"dose_adjustment_notes,omitempty"`

	// SAFETY_ALERT fields
	FlagID            string `json:"flag_id,omitempty"`
	Severity          string `json:"severity,omitempty"`
	RecommendedAction string `json:"recommended_action,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// SafetyAlertRequest is the inbound payload for POST /safety/hypoglycaemia-alert
// and POST /safety/behavioral-gap-alert from KB-21.
type SafetyAlertRequest struct {
	PatientID uuid.UUID `json:"patient_id" binding:"required"`
	Source    string    `json:"source" binding:"required"`
	AlertType string   `json:"alert_type"`
	GateType  string   `json:"gate_type"`
	Severity  string   `json:"severity"`
	Timestamp time.Time `json:"timestamp"`

	// Hypoglycaemia fields
	GlucoseMmolL     float64 `json:"glucose_mmol_l,omitempty"`
	DurationMinutes  int     `json:"duration_minutes,omitempty"`
	PredictedAtHours float64 `json:"predicted_at_hours,omitempty"`

	// Behavioral gap fields (KB-21 G-01)
	TreatmentResponseClass string  `json:"treatment_response_class,omitempty"`
	MeanAdherenceScore     float64 `json:"mean_adherence_score,omitempty"`
	HbA1cDelta             float64 `json:"hba1c_delta,omitempty"`
	DoseAdjustmentNotes    string  `json:"dose_adjustment_notes,omitempty"`

	// Hypo risk fields (KB-21 G-03)
	RiskFactors         []string `json:"risk_factors,omitempty"`
	RiskLevel           string   `json:"risk_level,omitempty"`
	AffectedMedications []string `json:"affected_medications,omitempty"`
}

// HPICompleteEvent is the inbound payload from KB-22 when an HPI session
// reaches convergence or completes.
type HPICompleteEvent struct {
	EventType           string              `json:"event_type"`
	PatientID           uuid.UUID           `json:"patient_id" binding:"required"`
	SessionID           uuid.UUID           `json:"session_id" binding:"required"`
	NodeID              string              `json:"node_id"`
	StratumLabel        string              `json:"stratum_label"`
	TopDiagnosis        string              `json:"top_diagnosis"`
	TopPosterior        float64             `json:"top_posterior"`
	RankedDifferentials []DifferentialEntry  `json:"ranked_differentials"`
	SafetyFlags         []SafetyFlagEntry   `json:"safety_flags"`

	// G5: HARD_BLOCK contraindications from KB-22 context modifiers.
	// Each block triggers a SAFETY_INSTRUCTION recommendation in the decision card.
	MedicationBlocks []MedicationBlock `json:"medication_blocks,omitempty"`

	// CTL Panel 4: Reasoning chain passed through from KB-22
	ReasoningChain json.RawMessage `json:"reasoning_chain,omitempty"`

	ConvergenceReached  bool                `json:"convergence_reached"`
	CompletedAt         *time.Time          `json:"completed_at"`
}

// MedicationBlock represents a HARD_BLOCK contraindication from KB-22 context modifiers.
// Used by card_builder to inject SAFETY_INSTRUCTION recommendations.
type MedicationBlock struct {
	ModifierID       string `json:"modifier_id"`
	BlockedTreatment string `json:"blocked_treatment"`
	Reason           string `json:"reason,omitempty"`
	DrugClass        string `json:"drug_class,omitempty"`
}

// DifferentialEntry represents a single ranked differential diagnosis.
type DifferentialEntry struct {
	DifferentialID string  `json:"differential_id"`
	Posterior      float64 `json:"posterior"`
}

// SafetyFlagEntry represents a safety flag raised during an HPI session.
type SafetyFlagEntry struct {
	FlagID            string `json:"flag_id"`
	Severity          string `json:"severity"`
	RecommendedAction string `json:"recommended_action"`
}
