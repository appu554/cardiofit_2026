// Package decisions defines the governance decision interfaces for KB-18.
//
// EXECUTION CONTRACT (Phase 2: Governance Before Intelligence):
//
//	KB-18 decides WHAT should happen (policy decisions)
//	KB-0 decides HOW and WHEN it happens (workflow execution)
//
// KEY INVARIANTS:
//   - KB-18 NEVER mutates state
//   - KB-0 NEVER invents policy
//   - All decisions are deterministic and fully auditable
//   - Same input ALWAYS produces same output
package decisions

import (
	"context"
	"time"
)

// =============================================================================
// FACT LIFECYCLE STATE MACHINE
// =============================================================================
//
// State transitions governed by KB-18 policy decisions:
//
//	DRAFT → (ActivationDecision) → AUTO_ACTIVATE → ACTIVE
//	                             → REQUIRE_REVIEW → PENDING_REVIEW
//	                             → REJECT → REJECTED
//
//	PENDING_REVIEW → (ReviewDecision) → APPROVE → APPROVED
//	                                  → REVISE → DRAFT
//	                                  → REJECT → REJECTED
//
//	APPROVED → (ActivationDecision) → ACTIVATE → ACTIVE
//
//	ACTIVE → (StabilityDecision) → STALE → requires refresh
//	                             → UNSAFE → requires immediate review
//
//	ACTIVE → RETIRE → RETIRED (via supersession or expiration)

// ActivationAction represents the policy decision for fact activation
type ActivationAction string

const (
	// ActionAutoActivate - High confidence fact, can be activated immediately
	ActionAutoActivate ActivationAction = "AUTO_ACTIVATE"
	// ActionRequireReview - Medium confidence, requires human review before activation
	ActionRequireReview ActivationAction = "REQUIRE_REVIEW"
	// ActionReject - Low confidence or policy violation, reject the fact
	ActionReject ActivationAction = "REJECT"
)

// ActivationDecision is the KB-18 answer to: "Should this fact activate?"
// This is a PURE POLICY DECISION - KB-18 never executes, only decides.
type ActivationDecision struct {
	// Action is the policy decision
	Action ActivationAction `json:"action"`

	// Reason provides human-readable explanation for audit
	Reason string `json:"reason"`

	// ConfidenceThreshold that triggered this decision
	ConfidenceThreshold float64 `json:"confidence_threshold,omitempty"`

	// ActualConfidence of the fact being evaluated
	ActualConfidence float64 `json:"actual_confidence,omitempty"`

	// ReviewQueue specifies which queue if REQUIRE_REVIEW
	// e.g., "pharmacist", "specialist", "cmo"
	ReviewQueue string `json:"review_queue,omitempty"`

	// RequiresDualReview indicates if two independent reviews are needed
	RequiresDualReview bool `json:"requires_dual_review,omitempty"`

	// PolicyReference is the governance policy that mandated this decision
	PolicyReference string `json:"policy_reference,omitempty"`

	// RiskFactors that influenced this decision
	RiskFactors []RiskFactor `json:"risk_factors,omitempty"`
}

// RiskFactor represents a single risk indicator
type RiskFactor struct {
	Type        string  `json:"type"`        // HIGH_ALERT, NARROW_THERAPEUTIC, BLACK_BOX, etc.
	Description string  `json:"description"` // Human-readable description
	Impact      string  `json:"impact"`      // How this factor affected the decision
	Weight      float64 `json:"weight"`      // Contribution to overall risk score
}

// =============================================================================
// EVIDENCE CONFLICT RESOLUTION STATE MACHINE
// =============================================================================
//
// When multiple sources provide conflicting facts:
//
//	CONFLICT_DETECTED → (ConflictResolutionDecision) → AUTO_RESOLVE → resolved by authority hierarchy
//	                                                 → HUMAN_REQUIRED → escalate to reviewer
//
// Authority Hierarchy (from Phase 1):
//
//	ONC Constitutional (authority=1) > FDA (authority=2) > ... > OHDSI (authority=21)

// ConflictResolutionAction represents how to resolve evidence conflicts
type ConflictResolutionAction string

const (
	// ResolutionAutoResolve - Conflict can be resolved by authority hierarchy
	ResolutionAutoResolve ConflictResolutionAction = "AUTO_RESOLVE"
	// ResolutionHumanRequired - Conflict too complex, needs human judgment
	ResolutionHumanRequired ConflictResolutionAction = "HUMAN_REQUIRED"
)

// ConflictResolutionDecision is the KB-18 answer to: "How to resolve this conflict?"
type ConflictResolutionDecision struct {
	// Action is the policy decision
	Action ConflictResolutionAction `json:"action"`

	// Reason provides explanation for audit
	Reason string `json:"reason"`

	// WinningSourceID is the source to prefer (if AUTO_RESOLVE)
	WinningSourceID string `json:"winning_source_id,omitempty"`

	// WinningAuthority is the authority of the winning source
	WinningAuthority string `json:"winning_authority,omitempty"`

	// AuthorityRank of the winning source (lower = higher priority)
	AuthorityRank int `json:"authority_rank,omitempty"`

	// ConflictingFacts lists all facts that were in conflict
	ConflictingFacts []ConflictingFact `json:"conflicting_facts"`

	// ReviewQueue if HUMAN_REQUIRED
	ReviewQueue string `json:"review_queue,omitempty"`

	// EscalationLevel indicates urgency of human review
	EscalationLevel string `json:"escalation_level,omitempty"` // ROUTINE, URGENT, CRITICAL

	// PolicyReference is the conflict resolution policy applied
	PolicyReference string `json:"policy_reference,omitempty"`
}

// ConflictingFact represents a single fact in a conflict
type ConflictingFact struct {
	FactID      string `json:"fact_id"`
	SourceID    string `json:"source_id"`
	Authority   string `json:"authority"`
	AuthorityRank int  `json:"authority_rank"`
	Value       string `json:"value"`       // The conflicting value
	ExtractedAt string `json:"extracted_at"`
}

// =============================================================================
// STABILITY (STALENESS) STATE MACHINE
// =============================================================================
//
// Active facts can become stale or unsafe:
//
//	ACTIVE → (StabilityDecision) → SAFE (no action needed)
//	                             → STALE (flag for refresh, still usable)
//	                             → UNSAFE (immediate review required, may disable)
//
// Staleness factors:
//   - Source document updated but fact not re-extracted
//   - Time since last verification exceeds threshold
//   - Dependent facts have changed
//   - External authority issued correction

// StabilityStatus represents the staleness state of a fact
type StabilityStatus string

const (
	// StatusSafe - Fact is current and valid
	StatusSafe StabilityStatus = "SAFE"
	// StatusStale - Fact may need refresh but still usable
	StatusStale StabilityStatus = "STALE"
	// StatusUnsafe - Fact requires immediate review, may be disabled
	StatusUnsafe StabilityStatus = "UNSAFE"
)

// StabilityDecision is the KB-18 answer to: "Is this fact still valid?"
type StabilityDecision struct {
	// Status is the staleness assessment
	Status StabilityStatus `json:"status"`

	// Reason provides explanation for audit
	Reason string `json:"reason"`

	// StalenessFactor indicates what triggered staleness
	StalenessFactor string `json:"staleness_factor,omitempty"`

	// DaysSinceVerification is days since last human or source verification
	DaysSinceVerification int `json:"days_since_verification,omitempty"`

	// SourceDocumentChanged indicates if underlying source was updated
	SourceDocumentChanged bool `json:"source_document_changed,omitempty"`

	// SourceDocumentVersion is the current source document version
	SourceDocumentVersion string `json:"source_document_version,omitempty"`

	// FactDocumentVersion is the version the fact was extracted from
	FactDocumentVersion string `json:"fact_document_version,omitempty"`

	// RecommendedAction for KB-0 to execute
	RecommendedAction StabilityAction `json:"recommended_action"`

	// RefreshDeadline is deadline by which fact must be refreshed (if STALE)
	RefreshDeadline *time.Time `json:"refresh_deadline,omitempty"`

	// DisableUntilRefresh indicates if fact should be disabled while stale
	DisableUntilRefresh bool `json:"disable_until_refresh,omitempty"`

	// PolicyReference is the staleness policy applied
	PolicyReference string `json:"policy_reference,omitempty"`
}

// StabilityAction is the recommended action for staleness
type StabilityAction string

const (
	// StabilityNoAction - No action needed
	StabilityNoAction StabilityAction = "NO_ACTION"
	// StabilityScheduleRefresh - Schedule background refresh
	StabilityScheduleRefresh StabilityAction = "SCHEDULE_REFRESH"
	// StabilityImmediateReview - Require immediate human review
	StabilityImmediateReview StabilityAction = "IMMEDIATE_REVIEW"
	// StabilityDisable - Disable fact until refreshed
	StabilityDisable StabilityAction = "DISABLE"
)

// =============================================================================
// OVERRIDE DECISION STATE MACHINE
// =============================================================================
//
// When a clinician needs to override a governance decision:
//
//	OVERRIDE_REQUESTED → (OverrideDecision) → SINGLE_SIGNATURE → one approver
//	                                        → DUAL_SIGNATURE → two approvers
//	                                        → NOT_PERMITTED → override denied
//
// Override requirements depend on:
//   - Risk level of the item being overridden
//   - Override reason category
//   - Clinician role and credentials
//   - Recent override patterns

// OverridePermission represents whether override is allowed
type OverridePermission string

const (
	// PermitSingleSignature - Override allowed with one approver
	PermitSingleSignature OverridePermission = "SINGLE_SIGNATURE"
	// PermitDualSignature - Override requires two independent approvers
	PermitDualSignature OverridePermission = "DUAL_SIGNATURE"
	// PermitNotAllowed - Override is not permitted
	PermitNotAllowed OverridePermission = "NOT_PERMITTED"
)

// OverrideDecision is the KB-18 answer to: "What signatures are needed?"
type OverrideDecision struct {
	// Permission is the policy decision
	Permission OverridePermission `json:"permission"`

	// Reason provides explanation for audit
	Reason string `json:"reason"`

	// RequiredApprovers lists roles that can approve
	RequiredApprovers []string `json:"required_approvers,omitempty"`

	// RequiredCredentials lists required professional credentials
	RequiredCredentials []string `json:"required_credentials,omitempty"`

	// MinApproverLevel is minimum seniority for single approver
	MinApproverLevel string `json:"min_approver_level,omitempty"`

	// ExpirationHours is how long the override remains valid
	ExpirationHours int `json:"expiration_hours,omitempty"`

	// RequiresDocumentation indicates clinical justification is required
	RequiresDocumentation bool `json:"requires_documentation,omitempty"`

	// RequiredAttestations are mandatory acknowledgments
	RequiredAttestations []string `json:"required_attestations,omitempty"`

	// PatternAlert indicates if override pattern is concerning
	PatternAlert *OverridePatternAlert `json:"pattern_alert,omitempty"`

	// PolicyReference is the override policy applied
	PolicyReference string `json:"policy_reference,omitempty"`
}

// OverridePatternAlert warns about concerning override patterns
type OverridePatternAlert struct {
	AlertType     string `json:"alert_type"`     // FREQUENCY, CONSISTENCY, CATEGORY
	Description   string `json:"description"`
	OverrideCount int    `json:"override_count"` // Recent override count
	TimePeriod    string `json:"time_period"`    // e.g., "24h", "7d"
	Recommendation string `json:"recommendation"`
}

// =============================================================================
// GOVERNANCE POLICY ENGINE INTERFACE
// =============================================================================

// GovernanceDecisionEngine is the interface KB-0 uses to query KB-18 for decisions.
// KB-18 implements this interface. KB-0 calls it and executes the results.
//
// INVARIANTS:
//   - All methods are pure functions (no side effects)
//   - Same input ALWAYS produces same output
//   - All decisions are fully auditable
type GovernanceDecisionEngine interface {
	// DecideActivation answers: "Should this fact activate?"
	DecideActivation(ctx context.Context, req *ActivationRequest) (*ActivationDecision, error)

	// DecideConflictResolution answers: "How to resolve this conflict?"
	DecideConflictResolution(ctx context.Context, req *ConflictResolutionRequest) (*ConflictResolutionDecision, error)

	// DecideStability answers: "Is this fact still valid?"
	DecideStability(ctx context.Context, req *StabilityRequest) (*StabilityDecision, error)

	// DecideOverride answers: "What signatures are needed?"
	DecideOverride(ctx context.Context, req *OverrideRequest) (*OverrideDecision, error)
}

// =============================================================================
// REQUEST TYPES
// =============================================================================

// ActivationRequest is the input for activation decisions
type ActivationRequest struct {
	// FactID is the unique identifier of the fact
	FactID string `json:"fact_id"`

	// FactType is the type of fact (ORGAN_IMPAIRMENT, INTERACTION, etc.)
	FactType string `json:"fact_type"`

	// Confidence is the extraction confidence score (0.0 to 1.0)
	Confidence float64 `json:"confidence"`

	// ConfidenceSignals are the individual signals that contributed
	ConfidenceSignals []ConfidenceSignal `json:"confidence_signals,omitempty"`

	// SourceAuthority is the authoritative source
	SourceAuthority string `json:"source_authority"`

	// RiskLevel of the fact content
	RiskLevel string `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL

	// RiskFlags are additional risk indicators
	RiskFlags FactRiskFlags `json:"risk_flags,omitempty"`

	// KBID is the target Knowledge Base
	KBID string `json:"kb_id"`

	// ExtractionMethod indicates how the fact was extracted
	ExtractionMethod string `json:"extraction_method"` // LLM, API_SYNC, ETL

	// RequestorID is who is requesting activation
	RequestorID string `json:"requestor_id,omitempty"`
}

// ConfidenceSignal represents one factor in confidence calculation
type ConfidenceSignal struct {
	Name    string  `json:"name"`
	Present bool    `json:"present"`
	Weight  float64 `json:"weight"`
	Value   string  `json:"value,omitempty"`
}

// FactRiskFlags are risk indicators for a fact
type FactRiskFlags struct {
	HighAlertDrug       bool `json:"high_alert_drug,omitempty"`
	NarrowTherapeutic   bool `json:"narrow_therapeutic,omitempty"`
	BlackBoxWarning     bool `json:"black_box_warning,omitempty"`
	ControlledSubstance bool `json:"controlled_substance,omitempty"`
	Chemotherapy        bool `json:"chemotherapy,omitempty"`
	Pediatric           bool `json:"pediatric,omitempty"`
	Pregnancy           bool `json:"pregnancy,omitempty"`
	Geriatric           bool `json:"geriatric,omitempty"`
}

// ConflictResolutionRequest is the input for conflict resolution decisions
type ConflictResolutionRequest struct {
	// ConflictID is the unique identifier for this conflict
	ConflictID string `json:"conflict_id"`

	// ConflictType describes the nature of the conflict
	ConflictType string `json:"conflict_type"` // VALUE, THRESHOLD, CONTRAINDICATION, etc.

	// ConflictingFacts are the facts in conflict
	ConflictingFacts []ConflictingFactInput `json:"conflicting_facts"`

	// KBID is the Knowledge Base context
	KBID string `json:"kb_id"`

	// DrugRxCUI is the drug involved (if applicable)
	DrugRxCUI string `json:"drug_rxcui,omitempty"`

	// ClinicalContext is additional context for resolution
	ClinicalContext string `json:"clinical_context,omitempty"`
}

// ConflictingFactInput is input data for a conflicting fact
type ConflictingFactInput struct {
	FactID         string  `json:"fact_id"`
	SourceID       string  `json:"source_id"`
	Authority      string  `json:"authority"`
	Value          string  `json:"value"`
	Confidence     float64 `json:"confidence"`
	ExtractedAt    string  `json:"extracted_at"`
	SourceVersion  string  `json:"source_version,omitempty"`
}

// StabilityRequest is the input for stability assessment
type StabilityRequest struct {
	// FactID is the unique identifier of the fact
	FactID string `json:"fact_id"`

	// FactType is the type of fact
	FactType string `json:"fact_type"`

	// LastVerifiedAt is when the fact was last verified
	LastVerifiedAt time.Time `json:"last_verified_at"`

	// SourceDocumentVersion is the current source document version
	SourceDocumentVersion string `json:"source_document_version"`

	// FactDocumentVersion is the version the fact was extracted from
	FactDocumentVersion string `json:"fact_document_version"`

	// KBID is the Knowledge Base context
	KBID string `json:"kb_id"`

	// RiskLevel of the fact
	RiskLevel string `json:"risk_level"`

	// HasDependentFacts indicates if other facts depend on this
	HasDependentFacts bool `json:"has_dependent_facts,omitempty"`

	// DependentFactCount is the number of dependent facts
	DependentFactCount int `json:"dependent_fact_count,omitempty"`
}

// OverrideRequest is the input for override decisions
type OverrideRequest struct {
	// ItemID is the item being overridden
	ItemID string `json:"item_id"`

	// ItemType is the type of item
	ItemType string `json:"item_type"`

	// KBID is the Knowledge Base context
	KBID string `json:"kb_id"`

	// RiskLevel of the item
	RiskLevel string `json:"risk_level"`

	// OverrideReason is the clinical justification category
	OverrideReason string `json:"override_reason"`

	// RequestorID is who is requesting the override
	RequestorID string `json:"requestor_id"`

	// RequestorRole is the role of the requestor
	RequestorRole string `json:"requestor_role"`

	// RequestorCredentials are the professional credentials
	RequestorCredentials []string `json:"requestor_credentials,omitempty"`

	// RecentOverrideCount is overrides by this user in last 24h
	RecentOverrideCount int `json:"recent_override_count,omitempty"`

	// RecentOverrideCountWeekly is overrides in last 7 days
	RecentOverrideCountWeekly int `json:"recent_override_count_weekly,omitempty"`

	// PatientContext provides patient-specific context
	PatientContext *PatientContext `json:"patient_context,omitempty"`
}

// PatientContext provides patient information for override decisions
type PatientContext struct {
	PatientID   string   `json:"patient_id,omitempty"`
	Age         int      `json:"age,omitempty"`
	IsPregnant  bool     `json:"is_pregnant,omitempty"`
	IsPediatric bool     `json:"is_pediatric,omitempty"`
	IsGeriatric bool     `json:"is_geriatric,omitempty"`
	RenalStage  string   `json:"renal_stage,omitempty"`
	HepaticClass string  `json:"hepatic_class,omitempty"`
	Diagnoses   []string `json:"diagnoses,omitempty"`
}
