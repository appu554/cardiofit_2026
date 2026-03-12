// Package policy provides governance policy evaluation for clinical facts.
// This implements the "policy inside KB-0" approach where governance decisions
// are made by pure functions operating on clinical facts from the Canonical Fact Store.
package policy

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// FACT TYPES (Mirrors clinical_facts table)
// =============================================================================

// FactType represents the type of clinical fact.
type FactType string

const (
	FactTypeOrganImpairment    FactType = "ORGAN_IMPAIRMENT"
	FactTypeSafetySignal       FactType = "SAFETY_SIGNAL"
	FactTypeReproductiveSafety FactType = "REPRODUCTIVE_SAFETY"
	FactTypeInteraction        FactType = "INTERACTION"
	FactTypeFormulary          FactType = "FORMULARY"
	FactTypeLabReference       FactType = "LAB_REFERENCE"
)

// FactStatus represents the lifecycle state of a clinical fact.
type FactStatus string

const (
	FactStatusDraft      FactStatus = "DRAFT"
	FactStatusApproved   FactStatus = "APPROVED"
	FactStatusActive     FactStatus = "ACTIVE"
	FactStatusSuperseded FactStatus = "SUPERSEDED"
	FactStatusDeprecated FactStatus = "DEPRECATED"
)

// SourceType represents how the fact was extracted.
type SourceType string

const (
	SourceTypeLLM     SourceType = "LLM"
	SourceTypeAPISync SourceType = "API_SYNC"
	SourceTypeETL     SourceType = "ETL"
	SourceTypeManual  SourceType = "MANUAL"
)

// ConfidenceBand represents confidence categorization.
type ConfidenceBand string

const (
	ConfidenceHigh   ConfidenceBand = "HIGH"
	ConfidenceMedium ConfidenceBand = "MEDIUM"
	ConfidenceLow    ConfidenceBand = "LOW"
)

// ReviewPriority represents the urgency of review.
type ReviewPriority string

const (
	ReviewPriorityCritical ReviewPriority = "CRITICAL"
	ReviewPriorityHigh     ReviewPriority = "HIGH"
	ReviewPriorityStandard ReviewPriority = "STANDARD"
	ReviewPriorityLow      ReviewPriority = "LOW"
)

// GovernanceDecision represents the outcome of policy evaluation.
type GovernanceDecision string

const (
	DecisionAutoApproved  GovernanceDecision = "AUTO_APPROVED"
	DecisionApproved      GovernanceDecision = "APPROVED"
	DecisionRejected      GovernanceDecision = "REJECTED"
	DecisionSuperseded    GovernanceDecision = "SUPERSEDED"
	DecisionEscalated     GovernanceDecision = "ESCALATED"
	DecisionPendingReview GovernanceDecision = "PENDING_REVIEW"
)

// =============================================================================
// CLINICAL FACT (From Shared DB)
// =============================================================================

// ClinicalFact represents a fact from the Canonical Fact Store (clinical_facts table).
type ClinicalFact struct {
	// Identity
	FactID   uuid.UUID `json:"factId" db:"fact_id"`
	FactType FactType  `json:"factType" db:"fact_type"`

	// Drug Reference
	RxCUI    string `json:"rxcui" db:"rxcui"`
	DrugName string `json:"drugName" db:"drug_name"`

	// Drug Composition (from source_documents)
	GenericName  *string  `json:"genericName,omitempty" db:"generic_name"`
	Manufacturer *string  `json:"manufacturer,omitempty" db:"manufacturer"`
	NDCCodes     []string `json:"ndcCodes,omitempty" db:"ndc_codes"`
	ATCCodes     []string `json:"atcCodes,omitempty" db:"atc_codes"`

	// Scope
	Scope      string  `json:"scope" db:"scope"`
	ClassRxCUI *string `json:"classRxcui,omitempty" db:"class_rxcui"`
	ClassName  *string `json:"className,omitempty" db:"class_name"`

	// Content (type-specific JSONB)
	Content map[string]interface{} `json:"content" db:"content"`

	// Provenance
	SourceType        SourceType `json:"sourceType" db:"source_type"`
	SourceID          string     `json:"sourceId" db:"source_id"`
	SourceVersion     *string    `json:"sourceVersion,omitempty" db:"source_version"`
	ExtractionMethod  string     `json:"extractionMethod" db:"extraction_method"`
	AuthorityPriority int        `json:"authorityPriority" db:"authority_priority"`

	// Evidence (populated from derived_facts)
	EvidenceSpans   []string   `json:"evidenceSpans,omitempty"`
	SourceSectionID *uuid.UUID `json:"sourceSectionId,omitempty"`

	// Confidence
	ConfidenceScore   *float64       `json:"confidenceScore,omitempty" db:"confidence_score"`
	ConfidenceBand    ConfidenceBand `json:"confidenceBand" db:"confidence_band"`
	ConfidenceSignals map[string]interface{} `json:"confidenceSignals,omitempty" db:"confidence_signals"`

	// Lifecycle
	Status        FactStatus  `json:"status" db:"status"`
	EffectiveFrom time.Time   `json:"effectiveFrom" db:"effective_from"`
	EffectiveTo   *time.Time  `json:"effectiveTo,omitempty" db:"effective_to"`
	SupersededBy  *uuid.UUID  `json:"supersededBy,omitempty" db:"superseded_by"`
	Version       int         `json:"version" db:"version"`

	// Governance (added by Phase 2 migration)
	ReviewPriority       *ReviewPriority     `json:"reviewPriority,omitempty" db:"review_priority"`
	AssignedReviewer     *string             `json:"assignedReviewer,omitempty" db:"assigned_reviewer"`
	AssignedAt           *time.Time          `json:"assignedAt,omitempty" db:"assigned_at"`
	ReviewDueAt          *time.Time          `json:"reviewDueAt,omitempty" db:"review_due_at"`
	GovernanceDecision   *GovernanceDecision `json:"governanceDecision,omitempty" db:"governance_decision"`
	DecisionReason       *string             `json:"decisionReason,omitempty" db:"decision_reason"`
	DecisionAt           *time.Time          `json:"decisionAt,omitempty" db:"decision_at"`
	DecisionBy           *string             `json:"decisionBy,omitempty" db:"decision_by"`
	HasConflict          bool                `json:"hasConflict" db:"has_conflict"`
	ConflictWithFactIDs  []uuid.UUID         `json:"conflictWithFactIds,omitempty" db:"conflict_with_fact_ids"`
	ConflictResolutionNotes *string          `json:"conflictResolutionNotes,omitempty" db:"conflict_resolution_notes"`

	// Audit
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	CreatedBy string    `json:"createdBy" db:"created_by"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// =============================================================================
// POLICY CONFIGURATION
// =============================================================================

// PolicyConfig holds thresholds and settings for policy evaluation.
type PolicyConfig struct {
	// Activation thresholds
	AutoApproveThreshold   float64 `json:"autoApproveThreshold"`   // >= this = AUTO_APPROVE
	RequireReviewThreshold float64 `json:"requireReviewThreshold"` // >= this < auto = REQUIRE_REVIEW
	RejectThreshold        float64 `json:"rejectThreshold"`        // < this = REJECT

	// Fact type overrides (some types always require review)
	AlwaysRequireReviewTypes []FactType `json:"alwaysRequireReviewTypes"`

	// Source type overrides
	AlwaysRequireReviewSources []SourceType `json:"alwaysRequireReviewSources"`

	// Conflict resolution
	EnableAuthorityPriority bool `json:"enableAuthorityPriority"`
	EnableRecencyTiebreak   bool `json:"enableRecencyTiebreak"`
}

// DefaultPolicyConfig returns the default policy configuration.
func DefaultPolicyConfig() PolicyConfig {
	return PolicyConfig{
		AutoApproveThreshold:   2.0, // Disabled: all facts require human pharmacist review
		RequireReviewThreshold: 0.65,
		RejectThreshold:        0.65,
		AlwaysRequireReviewTypes: []FactType{
			FactTypeSafetySignal,
		},
		AlwaysRequireReviewSources: []SourceType{
			SourceTypeLLM,
		},
		EnableAuthorityPriority: true,
		EnableRecencyTiebreak:   true,
	}
}

// =============================================================================
// POLICY DECISION TYPES
// =============================================================================

// ActivationDecision represents the result of activation policy evaluation.
type ActivationDecision struct {
	Outcome          GovernanceDecision `json:"outcome"`
	Reason           string             `json:"reason"`
	ConfidenceScore  float64            `json:"confidenceScore"`
	ThresholdApplied float64            `json:"thresholdApplied"`
	ReviewPriority   ReviewPriority     `json:"reviewPriority,omitempty"`
	RequiresReview   bool               `json:"requiresReview"`
	EvaluatedAt      time.Time          `json:"evaluatedAt"`
}

// ConflictDecision represents the result of conflict resolution.
type ConflictDecision struct {
	HasConflict          bool                   `json:"hasConflict"`
	ConflictingFactIDs   []uuid.UUID            `json:"conflictingFactIds,omitempty"`
	WinnerFactID         *uuid.UUID             `json:"winnerFactId,omitempty"`
	ResolutionStrategy   string                 `json:"resolutionStrategy"` // AUTHORITY_PRIORITY, RECENCY, MANUAL
	Reason               string                 `json:"reason"`
	RequiresManualReview bool                   `json:"requiresManualReview"`
	Details              map[string]interface{} `json:"details,omitempty"`
	EvaluatedAt          time.Time              `json:"evaluatedAt"`
}

// OverrideDecision represents the result of override policy evaluation.
type OverrideDecision struct {
	Allowed     bool                   `json:"allowed"`
	Reason      string                 `json:"reason"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
	ExpiresAt   *time.Time             `json:"expiresAt,omitempty"`
	RequiredRole string                `json:"requiredRole,omitempty"`
	EvaluatedAt time.Time              `json:"evaluatedAt"`
}

// StabilityDecision represents the result of stability policy evaluation.
type StabilityDecision struct {
	IsStable         bool      `json:"isStable"`
	MinActiveHours   int       `json:"minActiveHours"`
	CurrentActiveHours float64 `json:"currentActiveHours"`
	Reason           string    `json:"reason"`
	CanSupersede     bool      `json:"canSupersede"`
	EvaluatedAt      time.Time `json:"evaluatedAt"`
}

// =============================================================================
// AUTHORITY PRIORITY
// =============================================================================

// AuthorityInfo represents information about a source authority.
type AuthorityInfo struct {
	Code        string `json:"code" db:"authority_code"`
	Name        string `json:"name" db:"authority_name"`
	Priority    int    `json:"priority" db:"priority"`
	Jurisdiction string `json:"jurisdiction,omitempty" db:"jurisdiction"`
	TrustLevel  string `json:"trustLevel" db:"trust_level"`
}

// DefaultAuthorityPriorities returns the default authority priority map.
// Lower number = higher priority (ONC = 1, FDA = 2, OHDSI = 21).
func DefaultAuthorityPriorities() map[string]int {
	return map[string]int{
		"ONC":      1,  // Office of National Coordinator (Constitutional DDI)
		"FDA":      2,  // Food and Drug Administration
		"USP":      3,  // United States Pharmacopeia
		"NICE":     4,  // UK NICE Guidelines
		"TGA":      5,  // Australian TGA
		"CDSCO":    6,  // Indian CDSCO
		"EMA":      7,  // European Medicines Agency
		"DRUGBANK": 10, // DrugBank
		"RXNORM":   11, // RxNorm
		"OHDSI":    21, // OHDSI (research-grade)
	}
}

// =============================================================================
// QUEUE ITEM (For UI display)
// =============================================================================

// QueueItem represents a fact in the governance review queue.
type QueueItem struct {
	FactID              uuid.UUID          `json:"factId" db:"fact_id"`
	FactType            FactType           `json:"factType" db:"fact_type"`
	RxCUI               string             `json:"rxcui" db:"rxcui"`
	DrugName            string             `json:"drugName" db:"drug_name"`
	Scope               string             `json:"scope" db:"scope"`
	Content             map[string]interface{} `json:"content" db:"content"`
	SourceType          SourceType         `json:"sourceType" db:"source_type"`
	SourceID            string             `json:"sourceId" db:"source_id"`
	ConfidenceScore     *float64           `json:"confidenceScore,omitempty" db:"confidence_score"`
	ConfidenceBand      ConfidenceBand     `json:"confidenceBand" db:"confidence_band"`
	Status              FactStatus         `json:"status" db:"status"`
	ReviewPriority      *ReviewPriority    `json:"reviewPriority,omitempty" db:"review_priority"`
	AssignedReviewer    *string            `json:"assignedReviewer,omitempty" db:"assigned_reviewer"`
	ReviewDueAt         *time.Time         `json:"reviewDueAt,omitempty" db:"review_due_at"`
	HasConflict         bool               `json:"hasConflict" db:"has_conflict"`
	ConflictWithFactIDs []uuid.UUID        `json:"conflictWithFactIds,omitempty" db:"conflict_with_fact_ids"`
	AuthorityPriority   int                `json:"authorityPriority" db:"authority_priority"`
	CreatedAt           time.Time          `json:"createdAt" db:"created_at"`

	// Computed fields from view
	PriorityRank  int      `json:"priorityRank" db:"priority_rank"`
	DaysUntilDue  *float64 `json:"daysUntilDue,omitempty" db:"days_until_due"`
	SLAStatus     string   `json:"slaStatus" db:"sla_status"`
}

// =============================================================================
// REVIEW REQUEST (From UI)
// =============================================================================

// ReviewRequest represents a review action from the UI.
type ReviewRequest struct {
	FactID       uuid.UUID          `json:"factId"`
	Decision     GovernanceDecision `json:"decision"`
	Reason       string             `json:"reason"`
	ReviewerID   string             `json:"reviewerId"`
	ReviewerName string             `json:"reviewerName"`
	Credentials  string             `json:"credentials,omitempty"` // PharmD, MD, etc.
	IPAddress    string             `json:"ipAddress,omitempty"`
	SessionID    string             `json:"sessionId,omitempty"`
}

// =============================================================================
// GOVERNANCE AUDIT EVENT
// =============================================================================

// AuditEvent represents an event for the governance audit log.
type AuditEvent struct {
	EventType       string                 `json:"eventType"`
	FactID          uuid.UUID              `json:"factId"`
	PreviousState   string                 `json:"previousState,omitempty"`
	NewState        string                 `json:"newState"`
	ActorType       string                 `json:"actorType"` // SYSTEM, PHARMACIST, PHYSICIAN, ADMIN
	ActorID         string                 `json:"actorId"`
	ActorName       string                 `json:"actorName,omitempty"`
	ActorCredentials string                `json:"actorCredentials,omitempty"`
	Details         map[string]interface{} `json:"details"`
	IPAddress       string                 `json:"ipAddress,omitempty"`
	SessionID       string                 `json:"sessionId,omitempty"`
}

// =============================================================================
// CONFLICT GROUP (For conflict listing)
// =============================================================================

// ConflictGroup represents a group of conflicting facts for a drug/type.
type ConflictGroup struct {
	GroupID            string           `json:"groupId"`
	DrugRxCUI          string           `json:"drugRxcui"`
	DrugName           string           `json:"drugName"`
	FactType           string           `json:"factType"`
	Facts              []*ClinicalFact  `json:"facts"`
	ResolutionStrategy string           `json:"resolutionStrategy"` // AUTHORITY_PRIORITY, RECENCY, MANUAL
	SuggestedWinner    *uuid.UUID       `json:"suggestedWinner,omitempty"`
	ResolutionReason   *string          `json:"resolutionReason,omitempty"`
}

// =============================================================================
// AUDIT LOG ENTRY (For audit listing)
// =============================================================================

// AuditLogEntry represents an entry from the governance_audit_log table.
type AuditLogEntry struct {
	ID            string                 `json:"id"`
	EventType     string                 `json:"eventType" db:"event_type"`
	FactID        string                 `json:"factId" db:"fact_id"`
	PreviousState string                 `json:"previousState,omitempty" db:"previous_state"`
	NewState      string                 `json:"newState,omitempty" db:"new_state"`
	ActorType     string                 `json:"actorType" db:"actor_type"`
	ActorID       string                 `json:"actorId" db:"actor_id"`
	ActorName     string                 `json:"actorName" db:"actor_name"`
	ActorRole     string                 `json:"actorRole,omitempty" db:"actor_role"`
	Reason        string                 `json:"reason,omitempty" db:"reason"`
	Details       map[string]interface{} `json:"metadata,omitempty" db:"details"`
	IPAddress     string                 `json:"ipAddress,omitempty" db:"ip_address"`
	SessionID     string                 `json:"sessionId,omitempty" db:"session_id"`
	Signature     string                 `json:"signature" db:"event_signature"`
	CreatedAt     time.Time              `json:"createdAt" db:"event_timestamp"`
}
