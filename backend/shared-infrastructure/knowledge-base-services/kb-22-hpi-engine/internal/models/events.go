package models

import (
	"time"

	"github.com/google/uuid"
)

// Event types published by KB-22.
const (
	EventHPIComplete    = "HPI_COMPLETE"
	EventSafetyAlert    = "SAFETY_ALERT"
	EventQuestionAnswered = "QUESTION_ANSWERED"
	EventStratumDrifted = "STRATUM_DRIFTED"
)

// HPICompleteEvent is published to KB-23 and KB-19 on session completion.
type HPICompleteEvent struct {
	EventType   string    `json:"event_type"`
	PatientID   uuid.UUID `json:"patient_id"`
	SessionID   uuid.UUID `json:"session_id"`
	NodeID      string    `json:"node_id"`
	StratumLabel string   `json:"stratum_label"`

	TopDiagnosis  string  `json:"top_diagnosis"`
	TopPosterior  float64 `json:"top_posterior"`

	// Top-5 with posteriors for KB-23 Decision Card
	RankedDifferentials []DifferentialEntry `json:"ranked_differentials"`

	// IMMEDIATE + URGENT only for KB-19
	SafetyFlags []SafetyFlagSummary `json:"safety_flags,omitempty"`

	// CM contributions for explainability
	CMLogDeltasApplied map[string]float64 `json:"cm_log_deltas_applied,omitempty"`

	// N-01: guideline references
	GuidelinePriorRefs []string `json:"guideline_prior_refs,omitempty"`

	// CTL Panel 4: Reasoning chain from Bayesian update loop
	ReasoningChain []ReasoningStep `json:"reasoning_chain,omitempty"`

	ConvergenceReached bool `json:"convergence_reached"`

	CompletedAt time.Time `json:"completed_at"`
}

// SafetyAlertEvent is published to KB-19 immediately on IMMEDIATE trigger.
// Does NOT wait for session completion.
type SafetyAlertEvent struct {
	EventType   string    `json:"event_type"`
	PatientID   uuid.UUID `json:"patient_id"`
	SessionID   uuid.UUID `json:"session_id"`
	FlagID      string    `json:"flag_id"`
	Severity    string    `json:"severity"`
	RecommendedAction string `json:"recommended_action"`

	// N-02: KB-5 medication safety context (if available)
	MedicationSafetyContext interface{} `json:"medication_safety_context,omitempty"`

	FiredAt time.Time `json:"fired_at"`
}

// SafetyFlagSummary is a compact representation for event payloads.
type SafetyFlagSummary struct {
	FlagID            string `json:"flag_id"`
	Severity          string `json:"severity"`
	RecommendedAction string `json:"recommended_action"`
}

// QuestionTelemetry is written async to KB-21 per answered question.
type QuestionTelemetry struct {
	PatientID   uuid.UUID `json:"patient_id"`
	SessionID   uuid.UUID `json:"session_id"`
	QuestionID  string    `json:"question_id"`
	NodeID      string    `json:"node_id"`
	StratumLabel string   `json:"stratum_label"`

	InformationGainObserved float64 `json:"information_gain_observed"`
	WasPataNahi             bool    `json:"was_pata_nahi"`
	AnswerLatencyMS         int     `json:"answer_latency_ms"`

	AnsweredAt time.Time `json:"answered_at"`
}

// StratumDriftEvent is published to KB-19 when stratum changes on session resume (R-04).
type StratumDriftEvent struct {
	EventType      string    `json:"event_type"`
	PatientID      uuid.UUID `json:"patient_id"`
	SessionID      uuid.UUID `json:"session_id"`
	OldStratum     string    `json:"old_stratum"`
	NewStratum     string    `json:"new_stratum"`
	OldCKDSubstage *string   `json:"old_ckd_substage,omitempty"`
	NewCKDSubstage *string   `json:"new_ckd_substage,omitempty"`
	DetectedAt     time.Time `json:"detected_at"`
}

// AnswerResponse is returned to the caller after submitting an answer.
type AnswerResponse struct {
	SessionID        uuid.UUID           `json:"session_id"`
	Status           SessionStatus       `json:"status"`
	NextQuestion     *QuestionResponse   `json:"next_question,omitempty"`
	TopDifferentials []DifferentialEntry  `json:"top_differentials"`
	SafetyFlags      []SafetyFlagSummary `json:"safety_flags,omitempty"`
	// G16: termination reason when session ends due to cascade protocol.
	// Values: CONVERGED, MAX_QUESTIONS, PARTIAL_ASSESSMENT, SAFETY_ESCALATED
	TerminationReason string `json:"termination_reason,omitempty"`
	// G17: contradiction events detected on this answer
	Contradictions []ContradictionEvent `json:"contradictions,omitempty"`
	// G13: node transition events triggered on this answer
	Transitions []TransitionEvent `json:"transitions,omitempty"`
}

// QuestionResponse is the next question to present to the patient.
type QuestionResponse struct {
	QuestionID string `json:"question_id"`
	TextEN     string `json:"text_en"`
	TextHI     string `json:"text_hi"`
	Mandatory  bool   `json:"mandatory"`
	// G16: true when the question is rephrased via alt_prompt due to pata-nahi cascade.
	IsRephrase bool `json:"is_rephrase,omitempty"`
	// G16: true when cascade has entered binary-only mode (3+ consecutive pata-nahi).
	BinaryOnly bool `json:"binary_only,omitempty"`
}

// SessionResponse is the full session state for GET /sessions/:id.
type SessionResponse struct {
	SessionID       uuid.UUID           `json:"session_id"`
	PatientID       uuid.UUID           `json:"patient_id"`
	NodeID          string              `json:"node_id"`
	StratumLabel    string              `json:"stratum_label"`
	Status          SessionStatus       `json:"status"`
	QuestionsAsked  int                 `json:"questions_asked"`
	CurrentQuestion *QuestionResponse   `json:"current_question,omitempty"`
	TopDifferentials []DifferentialEntry `json:"top_differentials"`
	SafetyFlags     []SafetyFlagSummary `json:"safety_flags,omitempty"`
	StartedAt       time.Time           `json:"started_at"`
	LastActivityAt  time.Time           `json:"last_activity_at"`
}

// CreateSessionRequest is the request body for POST /sessions.
type CreateSessionRequest struct {
	PatientID uuid.UUID `json:"patient_id" binding:"required"`
	NodeID    string    `json:"node_id" binding:"required"`
}

// SubmitAnswerRequest is the request body for POST /sessions/:id/answers.
type SubmitAnswerRequest struct {
	QuestionID  string `json:"question_id" binding:"required"`
	AnswerValue string `json:"answer_value" binding:"required"`
	LatencyMS   int    `json:"latency_ms,omitempty"`
}
