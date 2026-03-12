// Package arbitration implements the Truth Arbitration Engine for Phase 3d.
// It resolves conflicts between Rules, Authorities, and Lab Interpretations
// using a deterministic precedence lattice.
//
// Precedence Hierarchy (highest to lowest):
//   1. REGULATORY - FDA Black Box, REMS (Trust: 1.00)
//   2. AUTHORITY  - CPIC, CredibleMeds, LactMed (Trust: 1.00)
//   3. LAB        - KB-16 interpretations (Trust: 0.95)
//   4. RULE       - Phase 3b.5 canonical rules (Trust: 0.90)
//   5. LOCAL      - Hospital policies (Trust: 0.80)
package arbitration

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// SOURCE TYPE ENUM
// =============================================================================

// SourceType represents the type of truth source in the precedence hierarchy.
type SourceType string

const (
	// SourceRegulatory represents FDA Black Box warnings, REMS, and hard contraindications.
	// Trust level: 1.00 (immutable). Always wins in conflict resolution.
	SourceRegulatory SourceType = "REGULATORY"

	// SourceAuthority represents curated expert guidelines (CPIC, CredibleMeds, LactMed).
	// Trust level: 1.00 (definitive). Beats rules and local policies.
	SourceAuthority SourceType = "AUTHORITY"

	// SourceLab represents KB-16 lab interpretations with patient context.
	// Trust level: 0.95 (contextual). Validates rules in real-time.
	SourceLab SourceType = "LAB"

	// SourceRule represents Phase 3b.5 canonical rules extracted from SPL/SmPC.
	// Trust level: 0.90 (deterministic). Can be overridden by authorities.
	SourceRule SourceType = "RULE"

	// SourceLocal represents hospital/institution-specific policy overrides.
	// Trust level: 0.80 (site-specific). Can override rules but not authorities.
	SourceLocal SourceType = "LOCAL"
)

// Precedence returns the precedence level (lower = higher priority).
func (s SourceType) Precedence() int {
	switch s {
	case SourceRegulatory:
		return 1
	case SourceAuthority:
		return 2
	case SourceLab:
		return 3
	case SourceRule:
		return 4
	case SourceLocal:
		return 5
	default:
		return 99
	}
}

// TrustLevel returns the default trust level for this source type.
func (s SourceType) TrustLevel() float64 {
	switch s {
	case SourceRegulatory:
		return 1.00
	case SourceAuthority:
		return 1.00
	case SourceLab:
		return 0.95
	case SourceRule:
		return 0.90
	case SourceLocal:
		return 0.80
	default:
		return 0.50
	}
}

// =============================================================================
// DECISION TYPE ENUM
// =============================================================================

// DecisionType represents the outcome of truth arbitration.
type DecisionType string

const (
	// DecisionAccept means all sources agree or no conflicts exist. Proceed safely.
	DecisionAccept DecisionType = "ACCEPT"

	// DecisionBlock means a hard constraint was violated. Cannot proceed.
	DecisionBlock DecisionType = "BLOCK"

	// DecisionOverride means a soft conflict exists. Can proceed with acknowledgment.
	DecisionOverride DecisionType = "OVERRIDE"

	// DecisionDefer means insufficient data exists. Need more information.
	DecisionDefer DecisionType = "DEFER"

	// DecisionEscalate means complex conflict requires human review.
	DecisionEscalate DecisionType = "ESCALATE"
)

// Severity returns the severity level of this decision (higher = more severe).
func (d DecisionType) Severity() int {
	switch d {
	case DecisionBlock:
		return 5
	case DecisionEscalate:
		return 4
	case DecisionDefer:
		return 3
	case DecisionOverride:
		return 2
	case DecisionAccept:
		return 1
	default:
		return 0
	}
}

// RequiresAction returns true if this decision requires user action.
func (d DecisionType) RequiresAction() bool {
	return d != DecisionAccept
}

// =============================================================================
// CONFLICT TYPE ENUM
// =============================================================================

// ConflictType represents the type of conflict detected between sources.
type ConflictType string

const (
	// ConflictRuleVsAuthority occurs when SPL rule disagrees with CPIC/CredibleMeds.
	// Example: SPL "avoid" vs CPIC "contraindicated"
	ConflictRuleVsAuthority ConflictType = "RULE_VS_AUTHORITY"

	// ConflictRuleVsLab occurs when a rule is triggered by lab values.
	// Example: Rule CrCl < 30, Lab eGFR = 28
	ConflictRuleVsLab ConflictType = "RULE_VS_LAB"

	// ConflictAuthorityVsLab occurs when authority threshold differs from lab context.
	// Example: CPIC eGFR < 30 vs Lab showing normal for pregnancy
	ConflictAuthorityVsLab ConflictType = "AUTHORITY_VS_LAB"

	// ConflictAuthorityVsAuthority occurs when two authorities disagree.
	// Example: CPIC vs CredibleMeds on same drug
	ConflictAuthorityVsAuthority ConflictType = "AUTHORITY_VS_AUTHORITY"

	// ConflictRuleVsRule occurs when multiple rules have different thresholds.
	// Example: Two SPLs with different CrCl cutoffs
	ConflictRuleVsRule ConflictType = "RULE_VS_RULE"

	// ConflictLocalVsAny occurs when hospital policy overrides a guideline.
	// Example: Hospital allows drug that guidelines say to avoid
	ConflictLocalVsAny ConflictType = "LOCAL_VS_ANY"
)

// Severity returns the default severity of this conflict type.
func (c ConflictType) Severity() string {
	switch c {
	case ConflictAuthorityVsLab:
		return "CRITICAL"
	case ConflictRuleVsLab, ConflictAuthorityVsAuthority:
		return "HIGH"
	case ConflictRuleVsAuthority, ConflictLocalVsAny:
		return "MEDIUM"
	case ConflictRuleVsRule:
		return "LOW"
	default:
		return "MEDIUM"
	}
}

// =============================================================================
// AUTHORITY LEVEL ENUM
// =============================================================================

// AuthorityLevel represents the evidence hierarchy for authority facts.
type AuthorityLevel string

const (
	// AuthorityDefinitive represents highest evidence (CPIC 1A, FDA contraindication).
	AuthorityDefinitive AuthorityLevel = "DEFINITIVE"

	// AuthorityPrimary represents strong evidence (CPIC 1B, major guidelines).
	AuthorityPrimary AuthorityLevel = "PRIMARY"

	// AuthoritySecondary represents moderate evidence (expert consensus).
	AuthoritySecondary AuthorityLevel = "SECONDARY"

	// AuthorityTertiary represents limited evidence (case reports, local practice).
	AuthorityTertiary AuthorityLevel = "TERTIARY"
)

// Priority returns the priority level (lower = higher priority).
func (a AuthorityLevel) Priority() int {
	switch a {
	case AuthorityDefinitive:
		return 1
	case AuthorityPrimary:
		return 2
	case AuthoritySecondary:
		return 3
	case AuthorityTertiary:
		return 4
	default:
		return 99
	}
}

// =============================================================================
// CLINICAL EFFECT ENUM
// =============================================================================

// ClinicalEffect represents the clinical action associated with an assertion.
type ClinicalEffect string

const (
	// EffectContraindicated means the drug must not be used.
	EffectContraindicated ClinicalEffect = "CONTRAINDICATED"

	// EffectAvoid means the drug should not be used unless no alternatives exist.
	EffectAvoid ClinicalEffect = "AVOID"

	// EffectCaution means the drug can be used with enhanced monitoring.
	EffectCaution ClinicalEffect = "CAUTION"

	// EffectReduceDose means dose adjustment is required.
	EffectReduceDose ClinicalEffect = "REDUCE_DOSE"

	// EffectMonitor means enhanced monitoring is required.
	EffectMonitor ClinicalEffect = "MONITOR"

	// EffectAllow means the drug is safe to proceed.
	EffectAllow ClinicalEffect = "ALLOW"

	// EffectNoEffect means there is no clinical impact.
	EffectNoEffect ClinicalEffect = "NO_EFFECT"
)

// RestrictivenessScore returns how restrictive this effect is (lower = more restrictive).
// Used for P7: more restrictive effect wins ties.
func (e ClinicalEffect) RestrictivenessScore() int {
	switch e {
	case EffectContraindicated:
		return 1
	case EffectAvoid:
		return 2
	case EffectCaution:
		return 3
	case EffectReduceDose:
		return 4
	case EffectMonitor:
		return 5
	case EffectAllow:
		return 6
	case EffectNoEffect:
		return 7
	default:
		return 10
	}
}

// IsRestrictive returns true if this effect restricts drug use.
func (e ClinicalEffect) IsRestrictive() bool {
	return e.RestrictivenessScore() <= 3
}

// MoreRestrictiveThan returns true if this effect is more restrictive than other.
func (e ClinicalEffect) MoreRestrictiveThan(other ClinicalEffect) bool {
	return e.RestrictivenessScore() < other.RestrictivenessScore()
}

// =============================================================================
// PRECEDENCE RULE
// =============================================================================

// PrecedenceRule represents a P1-P7 rule for conflict resolution.
type PrecedenceRule struct {
	ID               uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey"`
	RuleCode         string      `json:"rule_code" gorm:"uniqueIndex;not null"` // P1, P2, P3...
	RuleName         string      `json:"rule_name" gorm:"not null"`
	Description      string      `json:"description" gorm:"not null"`
	Priority         int         `json:"priority" gorm:"not null"` // Lower = higher priority
	SourceA          *SourceType `json:"source_a,omitempty"`
	SourceB          *SourceType `json:"source_b,omitempty"`
	Winner           *SourceType `json:"winner,omitempty"`
	SpecialCondition *string     `json:"special_condition,omitempty"`
	Rationale        string      `json:"rationale" gorm:"not null"`
	IsActive         bool        `json:"is_active" gorm:"default:true"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// TableName returns the database table name.
func (PrecedenceRule) TableName() string {
	return "precedence_rules"
}

// =============================================================================
// CONFLICT STRUCT
// =============================================================================

// Conflict represents a detected conflict between two sources.
type Conflict struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	ArbitrationID     uuid.UUID      `json:"arbitration_id" gorm:"type:uuid;not null"`
	Type              ConflictType   `json:"type" gorm:"not null"`
	SourceAType       SourceType     `json:"source_a_type" gorm:"not null"`
	SourceAID         *uuid.UUID     `json:"source_a_id,omitempty" gorm:"type:uuid"`
	SourceAAssertion  string         `json:"source_a_assertion" gorm:"not null"`
	SourceAEffect     ClinicalEffect `json:"source_a_effect,omitempty"`
	SourceBType       SourceType     `json:"source_b_type" gorm:"not null"`
	SourceBID         *uuid.UUID     `json:"source_b_id,omitempty" gorm:"type:uuid"`
	SourceBAssertion  string         `json:"source_b_assertion" gorm:"not null"`
	SourceBEffect     ClinicalEffect `json:"source_b_effect,omitempty"`
	ResolutionWinner  *SourceType    `json:"resolution_winner,omitempty"`
	ResolutionRule    string         `json:"resolution_rule,omitempty"` // P1, P2, P3...
	ResolutionRationale string       `json:"resolution_rationale,omitempty"`
	Severity          string         `json:"severity" gorm:"default:MEDIUM"`
	DetectedAt        time.Time      `json:"detected_at"`

	// P2 Authority Hierarchy Metadata - populated for AUTHORITY_VS_AUTHORITY conflicts
	// These fields enable proper comparison of authority levels per P2 rule
	SourceAAuthorityLevel *AuthorityLevel `json:"source_a_authority_level,omitempty" gorm:"type:varchar(20)"`
	SourceBAuthorityLevel *AuthorityLevel `json:"source_b_authority_level,omitempty" gorm:"type:varchar(20)"`

	// P5 Provenance Metadata - for tie-breaking by consensus strength
	SourceAProvenanceCount *int `json:"source_a_provenance_count,omitempty"`
	SourceBProvenanceCount *int `json:"source_b_provenance_count,omitempty"`
}

// TableName returns the database table name.
func (Conflict) TableName() string {
	return "conflicts_detected"
}

// Resolution contains the result of resolving a conflict.
type Resolution struct {
	Winner    SourceType `json:"winner"`
	Rule      string     `json:"rule"`      // P1, P2, etc.
	Rationale string     `json:"rationale"`
}

// =============================================================================
// AUDIT ENTRY
// =============================================================================

// AuditEntry represents a single step in the arbitration audit trail.
type AuditEntry struct {
	ID            uuid.UUID              `json:"id" gorm:"type:uuid;primaryKey"`
	ArbitrationID uuid.UUID              `json:"arbitration_id" gorm:"type:uuid;not null"`
	StepNumber    int                    `json:"step_number" gorm:"not null"`
	StepName      string                 `json:"step_name" gorm:"not null"`
	StepDescription string               `json:"step_description,omitempty"`
	Inputs        map[string]interface{} `json:"inputs,omitempty" gorm:"type:jsonb"`
	Outputs       map[string]interface{} `json:"outputs,omitempty" gorm:"type:jsonb"`
	DurationMs    int                    `json:"duration_ms,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// TableName returns the database table name.
func (AuditEntry) TableName() string {
	return "arbitration_audit_entries"
}
