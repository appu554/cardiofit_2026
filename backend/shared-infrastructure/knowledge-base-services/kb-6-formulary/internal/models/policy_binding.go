package models

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// POLICY BINDING TYPES
// =============================================================================

// BindingLevel defines the hierarchical level of policy enforcement
type BindingLevel string

const (
	BindingLevelFederal     BindingLevel = "FEDERAL"       // National regulations (e.g., FDA, CMS)
	BindingLevelState       BindingLevel = "STATE"         // State-level regulations
	BindingLevelPayer       BindingLevel = "PAYER"         // Insurance company policies
	BindingLevelPBM         BindingLevel = "PBM"           // Pharmacy Benefit Manager rules
	BindingLevelHospital    BindingLevel = "HOSPITAL"      // Hospital/health system governance
	BindingLevelNetwork     BindingLevel = "NETWORK"       // Provider network requirements
	BindingLevelFormulary   BindingLevel = "FORMULARY"     // Formulary committee decisions
	BindingLevelInstitution BindingLevel = "INSTITUTION"   // Institutional policies
)

// ComplianceEnforceMode defines how policy compliance is enforced
type ComplianceEnforceMode string

const (
	ComplianceEnforceHardBlock     ComplianceEnforceMode = "HARD_BLOCK"      // Cannot proceed without compliance
	ComplianceEnforceSoftBlock     ComplianceEnforceMode = "SOFT_BLOCK"      // Warning with override capability
	ComplianceEnforceWarn          ComplianceEnforceMode = "WARN"            // Warning only, no block
	ComplianceEnforceNotify        ComplianceEnforceMode = "NOTIFY"          // Notify stakeholders, no user impact
	ComplianceEnforceAudit         ComplianceEnforceMode = "AUDIT"           // Log only for compliance reporting
	ComplianceEnforceAdvisory      ComplianceEnforceMode = "ADVISORY"        // Educational, non-blocking
)

// JurisdictionType defines the type of jurisdiction
type JurisdictionType string

const (
	JurisdictionUS         JurisdictionType = "US"           // United States
	JurisdictionUSState    JurisdictionType = "US_STATE"     // US State (requires state_code)
	JurisdictionIndia      JurisdictionType = "INDIA"        // India (NLEM, CDSCO)
	JurisdictionAustralia  JurisdictionType = "AUSTRALIA"    // Australia (PBS, TGA)
	JurisdictionUK         JurisdictionType = "UK"           // United Kingdom (NHS, NICE)
	JurisdictionEU         JurisdictionType = "EU"           // European Union
	JurisdictionCanada     JurisdictionType = "CANADA"       // Canada (Health Canada)
	JurisdictionGlobal     JurisdictionType = "GLOBAL"       // Global (WHO, ICH)
)

// PolicyType defines the type of policy being applied
type PolicyType string

const (
	PolicyTypePriorAuth       PolicyType = "PRIOR_AUTHORIZATION"
	PolicyTypeStepTherapy     PolicyType = "STEP_THERAPY"
	PolicyTypeQuantityLimit   PolicyType = "QUANTITY_LIMIT"
	PolicyTypeFormulary       PolicyType = "FORMULARY"
	PolicyTypeGenericSubst    PolicyType = "GENERIC_SUBSTITUTION"
	PolicyTypeDrugInteraction PolicyType = "DRUG_INTERACTION"
	PolicyTypeSafety          PolicyType = "SAFETY"
	PolicyTypeReimbursement   PolicyType = "REIMBURSEMENT"
)

// =============================================================================
// POLICY BINDING MODELS
// =============================================================================

// Jurisdiction represents the geographic/legal jurisdiction for policy
type Jurisdiction struct {
	Type      JurisdictionType `json:"type"`
	Code      string           `json:"code,omitempty"`       // State/region code (e.g., "CA", "NSW")
	Name      string           `json:"name,omitempty"`       // Human-readable name
	Authority string           `json:"authority,omitempty"`  // Regulatory authority (e.g., "FDA", "TGA", "NLEM")
}

// PolicyReference represents a reference to a specific policy document
type PolicyReference struct {
	ID              string    `json:"id"`                         // Unique policy identifier
	Name            string    `json:"name"`                       // Policy name
	Version         string    `json:"version"`                    // Policy version
	EffectiveDate   time.Time `json:"effective_date"`             // When policy became effective
	ExpirationDate  *time.Time `json:"expiration_date,omitempty"` // When policy expires
	DocumentURL     string    `json:"document_url,omitempty"`     // Link to policy document
	SectionRef      string    `json:"section_ref,omitempty"`      // Specific section/clause reference
	LastReviewDate  *time.Time `json:"last_review_date,omitempty"`
}

// PayerProgram represents the payer program context
type PayerProgram struct {
	PayerID        string   `json:"payer_id"`                   // Insurance payer identifier
	ProgramID      string   `json:"program_id"`                 // Specific program identifier
	ProgramName    string   `json:"program_name"`               // Program name (e.g., "Medicare Part D", "Commercial PPO")
	ProgramType    string   `json:"program_type"`               // Type (Medicare, Medicaid, Commercial, Exchange)
	PlanID         string   `json:"plan_id,omitempty"`          // Specific plan within program
	ContractID     string   `json:"contract_id,omitempty"`      // Payer contract reference
	BenefitPhase   string   `json:"benefit_phase,omitempty"`    // Medicare phases: deductible, initial, gap, catastrophic
}

// PolicyBinding represents the complete policy binding context for PA/ST/QL evaluations
type PolicyBinding struct {
	ID                    uuid.UUID             `json:"id"`

	// Program Context
	PayerProgram          *PayerProgram         `json:"payer_program,omitempty"`

	// Policy Reference
	PolicyReference       PolicyReference       `json:"policy_reference"`
	PolicyType            PolicyType            `json:"policy_type"`

	// Jurisdiction
	Jurisdiction          Jurisdiction          `json:"jurisdiction"`

	// Binding Enforcement
	BindingLevel          BindingLevel          `json:"binding_level"`
	ComplianceEnforceMode ComplianceEnforceMode `json:"compliance_enforce_mode"`

	// Governance Context
	GovernanceRuleID      *string               `json:"governance_rule_id,omitempty"`      // Reference to Tier-7 Governance Engine rule
	ComplianceCategory    string                `json:"compliance_category,omitempty"`     // Category for compliance reporting
	AuditRequired         bool                  `json:"audit_required"`                    // Whether audit trail is mandatory

	// Hierarchy & Precedence
	ParentBindingID       *uuid.UUID            `json:"parent_binding_id,omitempty"`       // Parent binding for hierarchical policies
	Precedence            int                   `json:"precedence"`                        // Priority when multiple bindings apply (higher = more priority)

	// Override Configuration
	OverrideAllowed       bool                  `json:"override_allowed"`                  // Whether binding can be overridden
	OverrideApprovalLevel string                `json:"override_approval_level,omitempty"` // Required approval level for override

	// Metadata
	CreatedAt             time.Time             `json:"created_at"`
	UpdatedAt             time.Time             `json:"updated_at"`
}

// =============================================================================
// POLICY VIOLATION MODELS
// =============================================================================

// PolicyViolation represents a detected policy violation
type PolicyViolation struct {
	ID                uuid.UUID             `json:"id"`
	Binding           PolicyBinding         `json:"binding"`
	ViolationType     string                `json:"violation_type"`       // Type of violation
	ViolationCode     string                `json:"violation_code"`       // Machine-readable code
	Message           string                `json:"message"`              // Human-readable message
	Severity          string                `json:"severity"`             // critical, high, medium, low
	EnforcementAction ComplianceEnforceMode `json:"enforcement_action"`   // Action taken
	RequiresOverride  bool                  `json:"requires_override"`    // Whether override is needed to proceed
	AuditLogID        *string               `json:"audit_log_id,omitempty"` // Reference to audit entry
	DetectedAt        time.Time             `json:"detected_at"`
}

// =============================================================================
// REGIONAL POLICY TEMPLATES
// =============================================================================

// PredefinedPolicyBindings provides templates for common jurisdictions
var PredefinedPolicyBindings = map[string]PolicyBinding{
	// India NLEM (National List of Essential Medicines)
	"INDIA_NLEM": {
		Jurisdiction: Jurisdiction{
			Type:      JurisdictionIndia,
			Name:      "India",
			Authority: "NLEM",
		},
		BindingLevel:          BindingLevelFederal,
		ComplianceEnforceMode: ComplianceEnforceHardBlock,
		ComplianceCategory:    "NLEM_COMPLIANCE",
		AuditRequired:         true,
	},
	// Australia PBS (Pharmaceutical Benefits Scheme)
	"AUSTRALIA_PBS": {
		Jurisdiction: Jurisdiction{
			Type:      JurisdictionAustralia,
			Name:      "Australia",
			Authority: "PBS",
		},
		BindingLevel:          BindingLevelFederal,
		ComplianceEnforceMode: ComplianceEnforceSoftBlock,
		ComplianceCategory:    "PBS_COMPLIANCE",
		AuditRequired:         true,
	},
	// US Medicare Part D
	"US_MEDICARE_PART_D": {
		Jurisdiction: Jurisdiction{
			Type:      JurisdictionUS,
			Name:      "United States",
			Authority: "CMS",
		},
		BindingLevel:          BindingLevelFederal,
		ComplianceEnforceMode: ComplianceEnforceHardBlock,
		ComplianceCategory:    "MEDICARE_PART_D",
		AuditRequired:         true,
	},
	// Hospital Governance
	"HOSPITAL_GOVERNANCE": {
		Jurisdiction: Jurisdiction{
			Type:      JurisdictionUS,
			Name:      "Institutional",
			Authority: "Hospital P&T Committee",
		},
		BindingLevel:          BindingLevelHospital,
		ComplianceEnforceMode: ComplianceEnforceWarn,
		ComplianceCategory:    "HOSPITAL_FORMULARY",
		AuditRequired:         false,
	},
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewPolicyBinding creates a new policy binding with defaults
func NewPolicyBinding(policyType PolicyType, jurisdiction Jurisdiction, level BindingLevel) PolicyBinding {
	return PolicyBinding{
		ID:                    uuid.New(),
		PolicyType:            policyType,
		Jurisdiction:          jurisdiction,
		BindingLevel:          level,
		ComplianceEnforceMode: ComplianceEnforceWarn,
		AuditRequired:         false,
		Precedence:            1,
		OverrideAllowed:       true,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}

// GetPolicyBindingTemplate retrieves a predefined template
func GetPolicyBindingTemplate(templateName string) (*PolicyBinding, bool) {
	binding, ok := PredefinedPolicyBindings[templateName]
	if !ok {
		return nil, false
	}
	// Create a copy to avoid modifying the template
	copy := binding
	copy.ID = uuid.New()
	copy.CreatedAt = time.Now()
	copy.UpdatedAt = time.Now()
	return &copy, true
}

// FormatViolationMessage creates a formatted policy violation message
func FormatViolationMessage(binding PolicyBinding, violationType string) string {
	prefix := ""
	switch binding.Jurisdiction.Type {
	case JurisdictionIndia:
		prefix = "India"
	case JurisdictionAustralia:
		prefix = "Australia"
	case JurisdictionUS:
		prefix = "US"
	case JurisdictionUK:
		prefix = "UK"
	default:
		prefix = string(binding.Jurisdiction.Type)
	}

	authority := binding.Jurisdiction.Authority
	if authority == "" {
		authority = "Policy"
	}

	return prefix + " " + authority + " " + violationType
}
