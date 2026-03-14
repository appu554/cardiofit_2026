// Package models — request and response types for the SCE REST API.
package models

import "github.com/google/uuid"

// EvaluateRequest is the payload for POST /api/v1/evaluate.
// Contains a single answer submission with session context for safety evaluation.
type EvaluateRequest struct {
	SessionID  uuid.UUID       `json:"session_id" binding:"required"`
	NodeID     string          `json:"node_id" binding:"required"`
	QuestionID string          `json:"question_id" binding:"required"`
	Answer     string          `json:"answer" binding:"required"`
	FiredCMs   map[string]bool `json:"fired_cms,omitempty"`
}

// EvaluateResponse is the result of a safety evaluation.
// When Clear is true, no safety triggers fired. When false, Flags contains
// the fired triggers and EscalationRequired indicates whether KB-19 should
// override the Bayesian engine's response.
type EvaluateResponse struct {
	Clear              bool         `json:"clear"`
	Flags              []SafetyFlag `json:"flags,omitempty"`
	EscalationRequired bool         `json:"escalation_required"`
	ReasonCode         string       `json:"reason_code,omitempty"`
}

// ClearSessionResponse is the result of POST /api/v1/sessions/:id/clear.
type ClearSessionResponse struct {
	Status string `json:"status"`
}

// HealthResponse is the result of GET /health.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}
