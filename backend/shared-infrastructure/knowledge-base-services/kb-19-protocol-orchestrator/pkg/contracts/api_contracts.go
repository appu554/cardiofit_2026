// Package contracts provides API request/response contracts for KB-19.
// These contracts define the interface between KB-19 and its consumers.
package contracts

import (
	"time"

	"github.com/google/uuid"

	"kb-19-protocol-orchestrator/internal/models"
)

// ExecuteRequest is the request to execute protocol arbitration.
type ExecuteRequest struct {
	PatientID   uuid.UUID `json:"patient_id" binding:"required"`
	EncounterID uuid.UUID `json:"encounter_id" binding:"required"`

	// Optional: Pre-fetched clinical context
	// If not provided, KB-19 will fetch from Vaidshala
	CQLTruthFlags    map[string]bool    `json:"cql_truth_flags,omitempty"`
	CalculatorScores map[string]float64 `json:"calculator_scores,omitempty"`

	// Optional: Specific protocols to evaluate
	// If empty, all applicable protocols are evaluated
	ProtocolIDs []string `json:"protocol_ids,omitempty"`

	// Optional: Clinical context override
	ClinicalContext *ClinicalContextOverride `json:"clinical_context,omitempty"`

	// Request metadata
	RequestedBy string `json:"requested_by,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
}

// ClinicalContextOverride allows callers to provide specific clinical context.
type ClinicalContextOverride struct {
	IsICU           bool    `json:"is_icu"`
	IsPregnant      bool    `json:"is_pregnant"`
	HasAKI          bool    `json:"has_aki"`
	AKIStage        int     `json:"aki_stage"`
	EGFR            float64 `json:"egfr"`
	ShockState      string  `json:"shock_state"`
	BleedingRisk    string  `json:"bleeding_risk"`
	VentilatorMode  string  `json:"ventilator_mode,omitempty"`
}

// ExecuteResponse is the response from protocol execution.
type ExecuteResponse struct {
	BundleID          uuid.UUID                      `json:"bundle_id"`
	PatientID         uuid.UUID                      `json:"patient_id"`
	Timestamp         time.Time                      `json:"timestamp"`
	Status            string                         `json:"status"`
	Decisions         []DecisionSummary              `json:"decisions"`
	NarrativeSummary  string                         `json:"narrative_summary"`
	ExecutiveSummary  *ExecutiveSummary              `json:"executive_summary,omitempty"`
	ConflictsResolved []ConflictSummary              `json:"conflicts_resolved,omitempty"`
	SafetyGates       []SafetyGateSummary            `json:"safety_gates,omitempty"`
	ProcessingTimeMs  int64                          `json:"processing_time_ms"`
	Alerts            []Alert                        `json:"alerts,omitempty"`
}

// DecisionSummary is a summary of an arbitrated decision.
type DecisionSummary struct {
	ID                  uuid.UUID         `json:"id"`
	DecisionType        string            `json:"decision_type"`
	Target              string            `json:"target"`
	TargetCode          string            `json:"target_code,omitempty"`
	Rationale           string            `json:"rationale"`
	Urgency             string            `json:"urgency"`
	RecommendationClass string            `json:"recommendation_class"`
	EvidenceLevel       string            `json:"evidence_level"`
	SourceProtocol      string            `json:"source_protocol"`
	SafetyFlags         []SafetyFlagSummary `json:"safety_flags,omitempty"`
	Monitoring          []MonitoringItem  `json:"monitoring,omitempty"`
}

// SafetyFlagSummary is a summary of a safety flag.
type SafetyFlagSummary struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Reason   string `json:"reason"`
	Source   string `json:"source"`
}

// MonitoringItem is a monitoring requirement.
type MonitoringItem struct {
	Parameter    string   `json:"parameter"`
	Frequency    string   `json:"frequency"`
	Target       string   `json:"target,omitempty"`
	AlertIfAbove float64  `json:"alert_if_above,omitempty"`
	AlertIfBelow float64  `json:"alert_if_below,omitempty"`
}

// ExecutiveSummary provides a high-level summary for clinicians.
type ExecutiveSummary struct {
	TotalDecisions   int      `json:"total_decisions"`
	DoActions        int      `json:"do_actions"`
	AvoidActions     int      `json:"avoid_actions"`
	DelayedActions   int      `json:"delayed_actions"`
	SafetyBlocks     int      `json:"safety_blocks"`
	CriticalAlerts   int      `json:"critical_alerts"`
	HighestUrgency   string   `json:"highest_urgency"`
	KeyFindings      []string `json:"key_findings"`
}

// ConflictSummary is a summary of a resolved conflict.
type ConflictSummary struct {
	ProtocolA      string  `json:"protocol_a"`
	ProtocolB      string  `json:"protocol_b"`
	ConflictType   string  `json:"conflict_type"`
	Winner         string  `json:"winner"`
	LoserOutcome   string  `json:"loser_outcome"`
	Explanation    string  `json:"explanation"`
}

// SafetyGateSummary is a summary of a safety gate check.
type SafetyGateSummary struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	Triggered bool   `json:"triggered"`
	Result    string `json:"result"`
	Details   string `json:"details,omitempty"`
}

// Alert represents a clinical alert from the arbitration process.
type Alert struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	DecisionRef uuid.UUID `json:"decision_ref,omitempty"`
	RequiresAck bool      `json:"requires_ack"`
}

// EvaluateRequest is the request to evaluate a specific protocol.
type EvaluateRequest struct {
	PatientID   uuid.UUID `json:"patient_id" binding:"required"`
	EncounterID uuid.UUID `json:"encounter_id" binding:"required"`
	ProtocolID  string    `json:"protocol_id" binding:"required"`

	// Optional: Pre-fetched clinical context
	CQLTruthFlags    map[string]bool    `json:"cql_truth_flags,omitempty"`
	CalculatorScores map[string]float64 `json:"calculator_scores,omitempty"`
}

// EvaluateResponse is the response from protocol evaluation.
type EvaluateResponse struct {
	ProtocolID              string               `json:"protocol_id"`
	ProtocolName            string               `json:"protocol_name"`
	IsApplicable            bool                 `json:"is_applicable"`
	ApplicabilityReason     string               `json:"applicability_reason"`
	Contraindicated         bool                 `json:"contraindicated"`
	ContraindicationReasons []string             `json:"contraindication_reasons,omitempty"`
	RecommendedActions      []RecommendedAction  `json:"recommended_actions,omitempty"`
	RiskScoreImpact         float64              `json:"risk_score_impact,omitempty"`
	CQLFactsUsed            []string             `json:"cql_facts_used"`
	CalculatorsUsed         map[string]float64   `json:"calculators_used,omitempty"`
	EvaluatedAt             time.Time            `json:"evaluated_at"`
}

// RecommendedAction is an action recommended by a protocol.
type RecommendedAction struct {
	ActionType          string `json:"action_type"`
	Target              string `json:"target"`
	Dosing              string `json:"dosing,omitempty"`
	Timing              string `json:"timing,omitempty"`
	Urgency             string `json:"urgency"`
	RecommendationClass string `json:"recommendation_class"`
	EvidenceLevel       string `json:"evidence_level"`
	Rationale           string `json:"rationale"`
}

// ProtocolListResponse is the response for listing available protocols.
type ProtocolListResponse struct {
	Protocols []ProtocolSummary `json:"protocols"`
	Total     int               `json:"total"`
}

// ProtocolSummary is a summary of an available protocol.
type ProtocolSummary struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Category            string   `json:"category"`
	PriorityClass       int      `json:"priority_class"`
	GuidelineSource     string   `json:"guideline_source"`
	GuidelineVersion    string   `json:"guideline_version"`
	TriggerCriteria     []string `json:"trigger_criteria"`
	RequiredCalculators []string `json:"required_calculators"`
	IsActive            bool     `json:"is_active"`
}

// DecisionHistoryRequest is the request for decision history.
type DecisionHistoryRequest struct {
	PatientID uuid.UUID `json:"patient_id"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
	Since     time.Time `json:"since,omitempty"`
}

// DecisionHistoryResponse is the response for decision history.
type DecisionHistoryResponse struct {
	Decisions []DecisionSummary `json:"decisions"`
	Total     int               `json:"total"`
	HasMore   bool              `json:"has_more"`
}

// HealthResponse is the health check response.
type HealthResponse struct {
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	Uptime      string            `json:"uptime"`
	Dependencies map[string]string `json:"dependencies"`
}

// ReadyResponse is the readiness check response.
type ReadyResponse struct {
	Ready       bool              `json:"ready"`
	Checks      map[string]bool   `json:"checks"`
	Message     string            `json:"message,omitempty"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// ToDecisionSummary converts an ArbitratedDecision to a DecisionSummary.
func ToDecisionSummary(d *models.ArbitratedDecision) DecisionSummary {
	summary := DecisionSummary{
		ID:                  d.ID,
		DecisionType:        string(d.DecisionType),
		Target:              d.Target,
		TargetCode:          d.TargetRxNorm,
		Rationale:           d.Rationale,
		Urgency:             string(d.Urgency),
		RecommendationClass: string(d.Evidence.RecommendationClass),
		EvidenceLevel:       string(d.Evidence.EvidenceLevel),
		SourceProtocol:      d.SourceProtocol,
	}

	// Convert safety flags
	for _, flag := range d.SafetyFlags {
		summary.SafetyFlags = append(summary.SafetyFlags, SafetyFlagSummary{
			Type:     string(flag.Type),
			Severity: flag.Severity,
			Reason:   flag.Reason,
			Source:   flag.Source,
		})
	}

	// Convert monitoring items
	for _, m := range d.MonitoringPlan {
		item := MonitoringItem{
			Parameter: m.Parameter,
			Frequency: m.Frequency,
		}
		if m.AlertIfAbove != nil {
			item.AlertIfAbove = *m.AlertIfAbove
		}
		if m.AlertIfBelow != nil {
			item.AlertIfBelow = *m.AlertIfBelow
		}
		summary.Monitoring = append(summary.Monitoring, item)
	}

	return summary
}

// ToConflictSummary converts a ConflictResolution to a ConflictSummary.
func ToConflictSummary(c *models.ConflictResolution) ConflictSummary {
	return ConflictSummary{
		ProtocolA:    c.ProtocolA,
		ProtocolB:    c.ProtocolB,
		ConflictType: string(c.ConflictType),
		Winner:       c.Winner,
		LoserOutcome: string(c.LoserOutcome),
		Explanation:  c.Explanation,
	}
}

// ToSafetyGateSummary converts a SafetyGate to a SafetyGateSummary.
func ToSafetyGateSummary(g *models.SafetyGate) SafetyGateSummary {
	return SafetyGateSummary{
		Name:      g.Name,
		Source:    g.Source,
		Triggered: g.Triggered,
		Result:    g.Result,
		Details:   g.Details,
	}
}
