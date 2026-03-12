package contextrouter

import (
	"time"
)

// =============================================================================
// Context Router Decision Model
// =============================================================================
// This file defines the decision types and structures for the Context Router.
// The Context Router is STRICTLY DOWNSTREAM of Class Expansion.
//
// Execution Contract:
//   EXPANSION answers: "CAN this interaction exist?" (semantic)
//   CONTEXT answers:   "DOES it matter for this patient now?" (clinical)
//
// Golden Rule: Class Expansion NEVER checks LOINC. Context Router ALWAYS does.
// =============================================================================

// DecisionType represents the alert action determined by context evaluation
type DecisionType string

const (
	// DecisionBlock - Absolute contraindication, cannot proceed
	// Used for CRITICAL interactions that are always dangerous
	DecisionBlock DecisionType = "BLOCK"

	// DecisionInterrupt - Requires clinician acknowledgment before proceeding
	// Used when context thresholds are exceeded or context unavailable for required rules
	DecisionInterrupt DecisionType = "INTERRUPT"

	// DecisionInformational - Display information but allow to proceed
	// Used for WARNING-level interactions or when context shows safe values
	DecisionInformational DecisionType = "INFORMATIONAL"

	// DecisionSuppressed - Do not display, context indicates low risk
	// Used when context values are within safe ranges
	DecisionSuppressed DecisionType = "SUPPRESSED"

	// DecisionNeedsContext - Cannot evaluate, required context is missing
	// Used when context_required=true but LOINC value not provided
	DecisionNeedsContext DecisionType = "NEEDS_CONTEXT"
)

// EvaluationTier controls runtime evaluation priority (from OHDSI expansion service)
type EvaluationTier string

const (
	TierONCHigh   EvaluationTier = "TIER_0_ONC_HIGH"  // Always evaluate (ONC constitutional)
	TierSevere    EvaluationTier = "TIER_1_SEVERE"    // Severe/Contraindicated
	TierModerate  EvaluationTier = "TIER_2_MODERATE"  // Lazy-evaluated
	TierMechanism EvaluationTier = "TIER_3_MECHANISM" // Mechanism-only signals
)

// InteractionDirection indicates which drug is affected
type InteractionDirection string

const (
	DirectionBidirectional  InteractionDirection = "BIDIRECTIONAL"
	DirectionAffectsTrigger InteractionDirection = "AFFECTS_TRIGGER"
	DirectionAffectsTarget  InteractionDirection = "AFFECTS_TARGET"
)

// LabValue represents a single lab measurement with metadata
type LabValue struct {
	Value     float64   `json:"value"`
	Unit      string    `json:"unit,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// PatientContext contains the clinical context for a patient
// Labs are keyed by LOINC code for deterministic lookup
type PatientContext struct {
	PatientID       string              `json:"patient_id"`
	Labs            map[string]LabValue `json:"labs"`                     // LOINC code -> value with metadata
	Age             *int                `json:"age,omitempty"`
	Weight          *float64            `json:"weight_kg,omitempty"`
	RenalFunction   *float64            `json:"egfr,omitempty"`           // eGFR mL/min/1.73m2
	HepaticFunction *string             `json:"hepatic_class,omitempty"`  // Child-Pugh A/B/C
	Allergies       []string            `json:"allergies,omitempty"`      // Known drug allergies
	ActiveConditions []string           `json:"active_conditions,omitempty"` // ICD-10 codes
}

// DDIProjection represents an expanded drug-drug interaction from Class Expansion
// This is the INPUT to Context Router (output of OHDSI expansion)
type DDIProjection struct {
	RuleID               int                  `json:"rule_id"`
	DrugAConceptID       int64                `json:"drug_a_concept_id"`
	DrugAName            string               `json:"drug_a_name"`
	DrugAClassName       string               `json:"drug_a_class_name,omitempty"`
	DrugBConceptID       int64                `json:"drug_b_concept_id"`
	DrugBName            string               `json:"drug_b_name"`
	DrugBClassName       string               `json:"drug_b_class_name,omitempty"`
	RiskLevel            string               `json:"risk_level"` // CRITICAL, HIGH, WARNING, MODERATE
	AlertMessage         string               `json:"alert_message"`
	RuleAuthority        string               `json:"rule_authority"`
	RuleVersion          string               `json:"rule_version"`

	// Context metadata (from constitutional rules)
	ContextRequired      bool                 `json:"context_required"`
	ContextLOINCID       *string              `json:"context_loinc_id,omitempty"`
	ContextLOINCName     *string              `json:"context_loinc_name,omitempty"`
	ContextThreshold     *float64             `json:"context_threshold,omitempty"`
	ContextOperator      *string              `json:"context_operator,omitempty"` // <, >, <=, >=, =

	// Tiering & Directionality
	EvaluationTier       EvaluationTier       `json:"evaluation_tier"`
	InteractionDirection InteractionDirection `json:"interaction_direction"`
	LazyEvaluate         bool                 `json:"lazy_evaluate"`
	AffectedDrugRole     string               `json:"affected_drug_role,omitempty"`
}

// DDIDecision is the OUTPUT of Context Router
// Contains the reasoned decision with full audit trail
type DDIDecision struct {
	// Identity
	RuleID         int    `json:"rule_id"`
	DrugAConceptID int64  `json:"drug_a_concept_id"`
	DrugAName      string `json:"drug_a_name"`
	DrugBConceptID int64  `json:"drug_b_concept_id"`
	DrugBName      string `json:"drug_b_name"`

	// Decision
	Decision       DecisionType `json:"decision"`
	RiskLevel      string       `json:"risk_level"`
	AlertMessage   string       `json:"alert_message"`

	// Reasoning (for audit trail)
	Reason            string   `json:"reason"`
	ContextEvaluated  bool     `json:"context_evaluated"`
	ContextLOINCID    *string  `json:"context_loinc_id,omitempty"`
	ContextValue      *float64 `json:"context_value,omitempty"`
	ContextThreshold  *float64 `json:"context_threshold,omitempty"`
	ContextOperator   *string  `json:"context_operator,omitempty"`
	ThresholdExceeded *bool    `json:"threshold_exceeded,omitempty"`

	// Threshold Source (v2.0 Clinical Decision Limits)
	// Tracks whether threshold came from authoritative clinical guidelines
	// or fallback to projection-defined values
	//   - "AUTHORITATIVE": KDIGO, AHA/ACC, CPIC, CredibleMeds (near-zero false positive)
	//   - "PROJECTION_FALLBACK": DDI rule threshold (legacy behavior)
	//   - "": Context not evaluated
	LimitSource     string  `json:"limit_source,omitempty"`
	LimitAuthority  string  `json:"limit_authority,omitempty"` // e.g., "KDIGO 2024", "AHA/ACC"

	// Governance
	RuleAuthority  string         `json:"rule_authority"`
	EvaluationTier EvaluationTier `json:"evaluation_tier"`

	// Timestamps
	EvaluatedAt    time.Time `json:"evaluated_at"`
}

// ContextRouterRequest is the API request to evaluate DDIs with context
type ContextRouterRequest struct {
	PatientID      string              `json:"patient_id"`
	DrugConceptIDs []int64             `json:"drug_concept_ids"`
	Labs           map[string]LabValue `json:"labs"` // LOINC code -> value
}

// ContextRouterResponse is the API response with contextualized decisions
type ContextRouterResponse struct {
	PatientID        string        `json:"patient_id"`
	TotalProjections int           `json:"total_projections"`
	Decisions        []DDIDecision `json:"decisions"`

	// Summary counts by decision type
	BlockCount         int `json:"block_count"`
	InterruptCount     int `json:"interrupt_count"`
	InformationalCount int `json:"informational_count"`
	SuppressedCount    int `json:"suppressed_count"`
	NeedsContextCount  int `json:"needs_context_count"`

	// Performance
	EvaluationDurationMs int64 `json:"evaluation_duration_ms"`
}

// =============================================================================
// Helper Methods
// =============================================================================

// IsCritical returns true if the projection is CRITICAL risk level
func (p *DDIProjection) IsCritical() bool {
	return p.RiskLevel == "CRITICAL"
}

// IsONCConstitutional returns true if the rule is from ONC authority
func (p *DDIProjection) IsONCConstitutional() bool {
	return p.EvaluationTier == TierONCHigh
}

// RequiresContext returns true if context evaluation is required
func (p *DDIProjection) RequiresContext() bool {
	return p.ContextRequired && p.ContextLOINCID != nil
}

// HasContextMetadata returns true if the projection has context metadata
func (p *DDIProjection) HasContextMetadata() bool {
	return p.ContextLOINCID != nil && p.ContextThreshold != nil && p.ContextOperator != nil
}

// IsActionable returns true if the decision requires clinician action
func (d *DDIDecision) IsActionable() bool {
	return d.Decision == DecisionBlock || d.Decision == DecisionInterrupt || d.Decision == DecisionNeedsContext
}

// Priority returns the sort priority (lower = more urgent)
func (d *DDIDecision) Priority() int {
	switch d.Decision {
	case DecisionBlock:
		return 1
	case DecisionInterrupt:
		return 2
	case DecisionNeedsContext:
		return 3
	case DecisionInformational:
		return 4
	case DecisionSuppressed:
		return 5
	default:
		return 99
	}
}

// ShouldDisplay returns true if the decision should be shown to clinician
func (d *DDIDecision) ShouldDisplay() bool {
	return d.Decision != DecisionSuppressed
}

// GetLabValue safely retrieves a lab value from patient context
func (ctx *PatientContext) GetLabValue(loincCode string) (float64, bool) {
	if ctx.Labs == nil {
		return 0, false
	}
	lab, exists := ctx.Labs[loincCode]
	if !exists {
		return 0, false
	}
	return lab.Value, true
}

// HasLab checks if a specific LOINC code exists in patient context
func (ctx *PatientContext) HasLab(loincCode string) bool {
	if ctx.Labs == nil {
		return false
	}
	_, exists := ctx.Labs[loincCode]
	return exists
}
