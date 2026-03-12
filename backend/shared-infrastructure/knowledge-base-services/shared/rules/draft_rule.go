// Package rules provides canonical rule representation for clinical decision support.
//
// Phase 3b.5.1: DraftRule Contract
// Key Principle: Every extracted rule needs a canonical, computable representation
// with full provenance and semantic fingerprint for deduplication.
//
// Schema: {Condition + Action + Provenance + Fingerprint}
// Invariant: Every rule is traceable back to its regulatory source
package rules

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/types"
)

// =============================================================================
// DRAFT RULE - CANONICAL REPRESENTATION
// =============================================================================

// DraftRule represents a canonical, computable clinical rule
// Invariant: Every rule has a semantic fingerprint and full provenance
type DraftRule struct {
	RuleID uuid.UUID `json:"rule_id" db:"rule_id"`
	Domain string    `json:"domain" db:"domain"`       // KB-1, KB-4, KB-5
	RuleType RuleType `json:"rule_type" db:"rule_type"` // DOSING, CONTRAINDICATION, INTERACTION

	// Computable IF/THEN structure
	Condition Condition `json:"condition" db:"condition"`
	Action    Action    `json:"action" db:"action"`

	// Full lineage
	Provenance Provenance `json:"provenance" db:"provenance"`

	// Semantic deduplication
	SemanticFingerprint Fingerprint `json:"semantic_fingerprint" db:"semantic_fingerprint"`

	// Governance
	GovernanceStatus GovernanceStatus `json:"governance_status" db:"governance_status"`
	ReviewedBy       *string          `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt       *time.Time       `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ReviewNotes      string           `json:"review_notes,omitempty" db:"review_notes"`

	// Lifecycle
	IsActive     bool       `json:"is_active" db:"is_active"`
	SupersededBy *uuid.UUID `json:"superseded_by,omitempty" db:"superseded_by"`
	Supersedes   *uuid.UUID `json:"supersedes,omitempty" db:"supersedes"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// =============================================================================
// RULE TYPE DEFINITIONS
// =============================================================================

// RuleType classifies the clinical rule category
type RuleType string

const (
	// RuleTypeDosing indicates dose adjustment rules (renal, hepatic, age, weight)
	RuleTypeDosing RuleType = "DOSING"

	// RuleTypeContraindication indicates conditions where drug should not be used
	RuleTypeContraindication RuleType = "CONTRAINDICATION"

	// RuleTypeInteraction indicates drug-drug or drug-condition interactions
	RuleTypeInteraction RuleType = "INTERACTION"

	// RuleTypeMonitoring indicates monitoring requirements
	RuleTypeMonitoring RuleType = "MONITORING"

	// RuleTypeWarning indicates safety warnings
	RuleTypeWarning RuleType = "WARNING"

	// RuleTypePrecaution indicates use with caution scenarios
	RuleTypePrecaution RuleType = "PRECAUTION"
)

// =============================================================================
// CONDITION - THE "IF" PART (Type aliases to types package)
// =============================================================================

// Condition represents the IF part of the rule (alias to types.Condition)
type Condition = types.Condition

// Operator defines the comparison type (alias to types.Operator)
type Operator = types.Operator

// Operator constants - aliases to types package
const (
	OpLessThan       = types.OpLessThan
	OpGreaterThan    = types.OpGreaterThan
	OpLessOrEqual    = types.OpLessOrEqual
	OpGreaterOrEqual = types.OpGreaterOrEqual
	OpBetween        = types.OpBetween
	OpEquals         = types.OpEquals
	OpNotEquals      = types.OpNotEquals
	OpIn             = types.OpIn
)

// =============================================================================
// ACTION - THE "THEN" PART (Type aliases to types package)
// =============================================================================

// Action represents the THEN part of the rule (alias to types.Action)
type Action = types.Action

// Effect defines the clinical action type (alias to types.Effect)
type Effect = types.Effect

// Effect constants - aliases to types package
const (
	EffectContraindicated = types.EffectContraindicated
	EffectDoseAdjust      = types.EffectDoseAdjust
	EffectAvoid           = types.EffectAvoid
	EffectMonitor         = types.EffectMonitor
	EffectUseWithCaution  = types.EffectUseWithCaution
	EffectNoChange        = types.EffectNoChange
	EffectHold            = types.EffectHold
	EffectDiscontinue     = types.EffectDiscontinue
)

// Severity indicates the clinical importance of the action (alias to types.Severity)
type Severity = types.Severity

// Severity constants - aliases to types package
const (
	SeverityCritical = types.SeverityCritical
	SeverityHigh     = types.SeverityHigh
	SeverityModerate = types.SeverityModerate
	SeverityLow      = types.SeverityLow
	SeverityInfo     = types.SeverityInfo
)

// DoseAdjustment contains specific dosing modifications (alias to types.DoseAdjustment)
type DoseAdjustment = types.DoseAdjustment

// AdjustmentType defines how the dose is modified (alias to types.AdjustmentType)
type AdjustmentType = types.AdjustmentType

// AdjustmentType constants - aliases to types package
const (
	AdjustmentPercentage = types.AdjustmentPercentage
	AdjustmentAbsolute   = types.AdjustmentAbsolute
	AdjustmentInterval   = types.AdjustmentInterval
	AdjustmentMaxDose    = types.AdjustmentMaxDose
	AdjustmentFrequency  = types.AdjustmentFrequency
)

// =============================================================================
// PROVENANCE - COMPLETE SOURCE LINEAGE
// =============================================================================

// Provenance tracks the complete source lineage for audit trail
type Provenance struct {
	SourceDocumentID uuid.UUID  `json:"source_document_id"`
	SourceSectionID  *uuid.UUID `json:"source_section_id,omitempty"`
	SourceType       string     `json:"source_type"`        // FDA_SPL, CPIC, CREDIBLEMEDS, etc.
	DocumentID       string     `json:"document_id"`        // SetID for SPL
	VersionNumber    string     `json:"version_number"`     // Document version
	SectionCode      string     `json:"section_code"`       // LOINC code
	SectionName      string     `json:"section_name"`       // Human-readable section name
	TableID          string     `json:"table_id,omitempty"` // Source table ID if from table
	ExtractionMethod string     `json:"extraction_method"`  // TABLE_PARSE, REGEX_PARSE, AUTHORITY
	EvidenceSpan     string     `json:"evidence_span"`      // Quoted source text
	Confidence       float64    `json:"confidence"`         // 0.0 - 1.0
	ExtractedAt      time.Time  `json:"extracted_at"`
}

// =============================================================================
// FINGERPRINT - SEMANTIC DEDUPLICATION
// =============================================================================

// Fingerprint provides semantic deduplication
type Fingerprint struct {
	Hash      string    `json:"hash"`       // SHA256 of canonical JSON
	Version   int       `json:"version"`    // Schema version for hash compatibility
	CreatedAt time.Time `json:"created_at"`
}

// =============================================================================
// GOVERNANCE STATUS
// =============================================================================

// GovernanceStatus tracks the rule lifecycle
type GovernanceStatus string

const (
	GovernanceDraft      GovernanceStatus = "DRAFT"
	GovernanceReview     GovernanceStatus = "PENDING_REVIEW"
	GovernanceApproved   GovernanceStatus = "APPROVED"
	GovernanceRejected   GovernanceStatus = "REJECTED"
	GovernanceActive     GovernanceStatus = "ACTIVE"
	GovernanceSuperseded GovernanceStatus = "SUPERSEDED"
	GovernanceRetired    GovernanceStatus = "RETIRED"
)

// =============================================================================
// FINGERPRINT COMPUTATION
// =============================================================================

// canonicalForm contains the fields used for fingerprinting
// Only domain, condition, and action determine semantic equivalence
type canonicalForm struct {
	Domain    string    `json:"domain"`
	RuleType  RuleType  `json:"rule_type"`
	Condition Condition `json:"condition"`
	Action    Action    `json:"action"`
}

// ComputeFingerprint generates a semantic fingerprint for deduplication
// Two rules with the same fingerprint are semantically equivalent
func (r *DraftRule) ComputeFingerprint() Fingerprint {
	canonical := canonicalForm{
		Domain:    r.Domain,
		RuleType:  r.RuleType,
		Condition: r.Condition,
		Action:    r.Action,
	}

	// Deterministic JSON marshaling
	jsonBytes, _ := json.Marshal(canonical)
	hash := sha256.Sum256(jsonBytes)

	return Fingerprint{
		Hash:      fmt.Sprintf("%x", hash),
		Version:   1,
		CreatedAt: time.Now(),
	}
}

// =============================================================================
// FACTORY METHODS
// =============================================================================

// NewDraftRule creates a new DraftRule with required fields
func NewDraftRule(domain string, ruleType RuleType, condition Condition, action Action, provenance Provenance) *DraftRule {
	now := time.Now()

	rule := &DraftRule{
		RuleID:           uuid.New(),
		Domain:           domain,
		RuleType:         ruleType,
		Condition:        condition,
		Action:           action,
		Provenance:       provenance,
		GovernanceStatus: GovernanceDraft,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Compute fingerprint
	rule.SemanticFingerprint = rule.ComputeFingerprint()

	return rule
}

// NewDosingRule creates a dosing-specific rule
func NewDosingRule(domain string, condition Condition, adjustment DoseAdjustment, message string, provenance Provenance) *DraftRule {
	action := Action{
		Effect:     EffectDoseAdjust,
		Adjustment: &adjustment,
		Message:    message,
		Severity:   SeverityHigh,
	}

	return NewDraftRule(domain, RuleTypeDosing, condition, action, provenance)
}

// NewContraindicationRule creates a contraindication rule
func NewContraindicationRule(domain string, condition Condition, message string, provenance Provenance) *DraftRule {
	action := Action{
		Effect:   EffectContraindicated,
		Message:  message,
		Severity: SeverityCritical,
	}

	return NewDraftRule(domain, RuleTypeContraindication, condition, action, provenance)
}

// =============================================================================
// SERIALIZATION
// =============================================================================

// ToJSON serializes the rule to JSON
func (r *DraftRule) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// ToJSONPretty returns formatted JSON for debugging
func (r *DraftRule) ToJSONPretty() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// FromJSON deserializes a rule from JSON
func FromJSON(data []byte) (*DraftRule, error) {
	var r DraftRule
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// =============================================================================
// INTERFACE ACCESSORS - For use by fingerprint_registry to avoid import cycles
// =============================================================================

// GetRuleID returns the rule ID
func (r *DraftRule) GetRuleID() uuid.UUID {
	return r.RuleID
}

// GetDomain returns the domain
func (r *DraftRule) GetDomain() string {
	return r.Domain
}

// GetRuleType returns the rule type as string
func (r *DraftRule) GetRuleType() string {
	return string(r.RuleType)
}

// GetFingerprintHash returns the semantic fingerprint hash
func (r *DraftRule) GetFingerprintHash() string {
	return r.SemanticFingerprint.Hash
}

// GetFingerprintVersion returns the semantic fingerprint version
func (r *DraftRule) GetFingerprintVersion() int {
	return r.SemanticFingerprint.Version
}
