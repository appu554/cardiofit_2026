package governance

import "time"

// GovernanceSeverity represents the escalation tier for clinical decisions
type GovernanceSeverity string

const (
	// SeverityNotifyOnly - Information only, no action required
	SeverityNotifyOnly GovernanceSeverity = "NOTIFY_ONLY"

	// SeverityCounselingRequired - Patient counseling must be documented
	SeverityCounselingRequired GovernanceSeverity = "COUNSELING_REQUIRED"

	// SeverityOverrideWithDocumentation - Can override with documented clinical rationale
	SeverityOverrideWithDocumentation GovernanceSeverity = "OVERRIDE_WITH_DOCUMENTATION"

	// SeverityOverrideWithSupervisor - Requires supervisor approval to override
	SeverityOverrideWithSupervisor GovernanceSeverity = "OVERRIDE_WITH_SUPERVISOR"

	// SeverityHardBlock - Cannot be overridden, absolute contraindication
	SeverityHardBlock GovernanceSeverity = "HARD_BLOCK"

	// SeverityMandatoryEscalation - Must escalate to clinical review board
	SeverityMandatoryEscalation GovernanceSeverity = "MANDATORY_ESCALATION"
)

// GovernanceAction defines the required action for each severity level
type GovernanceAction struct {
	Severity            GovernanceSeverity `json:"severity"`
	DisplayName         string             `json:"display_name"`
	Description         string             `json:"description"`
	RequiresOverride    bool               `json:"requires_override"`
	OverrideAllowed     bool               `json:"override_allowed"`
	RequiresSupervisor  bool               `json:"requires_supervisor"`
	RequiresEscalation  bool               `json:"requires_escalation"`
	DocumentationNeeded bool               `json:"documentation_needed"`
	AuditRequired       bool               `json:"audit_required"`
}

// GovernanceActions maps severity levels to their required actions
var GovernanceActions = map[GovernanceSeverity]GovernanceAction{
	SeverityNotifyOnly: {
		Severity:            SeverityNotifyOnly,
		DisplayName:         "Notify Only",
		Description:         "Information provided for awareness. No action required.",
		RequiresOverride:    false,
		OverrideAllowed:     true,
		RequiresSupervisor:  false,
		RequiresEscalation:  false,
		DocumentationNeeded: false,
		AuditRequired:       false,
	},
	SeverityCounselingRequired: {
		Severity:            SeverityCounselingRequired,
		DisplayName:         "Counseling Required",
		Description:         "Patient counseling must be provided and documented.",
		RequiresOverride:    false,
		OverrideAllowed:     true,
		RequiresSupervisor:  false,
		RequiresEscalation:  false,
		DocumentationNeeded: true,
		AuditRequired:       true,
	},
	SeverityOverrideWithDocumentation: {
		Severity:            SeverityOverrideWithDocumentation,
		DisplayName:         "Override Allowed with Documentation",
		Description:         "Can proceed with documented clinical rationale.",
		RequiresOverride:    true,
		OverrideAllowed:     true,
		RequiresSupervisor:  false,
		RequiresEscalation:  false,
		DocumentationNeeded: true,
		AuditRequired:       true,
	},
	SeverityOverrideWithSupervisor: {
		Severity:            SeverityOverrideWithSupervisor,
		DisplayName:         "Override Requires Supervisor",
		Description:         "Supervisor approval required to proceed.",
		RequiresOverride:    true,
		OverrideAllowed:     true,
		RequiresSupervisor:  true,
		RequiresEscalation:  false,
		DocumentationNeeded: true,
		AuditRequired:       true,
	},
	SeverityHardBlock: {
		Severity:            SeverityHardBlock,
		DisplayName:         "Hard Block - No Override",
		Description:         "Absolute contraindication. Cannot be overridden.",
		RequiresOverride:    false,
		OverrideAllowed:     false,
		RequiresSupervisor:  false,
		RequiresEscalation:  true,
		DocumentationNeeded: true,
		AuditRequired:       true,
	},
	SeverityMandatoryEscalation: {
		Severity:            SeverityMandatoryEscalation,
		DisplayName:         "Mandatory Escalation",
		Description:         "Must be reviewed by clinical review board.",
		RequiresOverride:    true,
		OverrideAllowed:     true,
		RequiresSupervisor:  true,
		RequiresEscalation:  true,
		DocumentationNeeded: true,
		AuditRequired:       true,
	},
}

// EvidenceProvenance provides audit trail and regulatory compliance data
type EvidenceProvenance struct {
	// Clinical reference source (e.g., "UpToDate 2024", "Lexicomp", "FDA Label")
	ClinicalReferenceSource string `json:"clinical_reference_source"`

	// Calculation method version (e.g., "CKD-EPI 2021", "Cockcroft-Gault 1976")
	CalculationMethodVersion string `json:"calculation_method_version"`

	// Dataset version for drug rules
	DatasetVersion string `json:"dataset_version"`

	// Governance binding - regulatory framework (e.g., "FDA", "TGA", "EMA")
	GovernanceBinding string `json:"governance_binding"`

	// Whether secondary validation is required
	RequiresSecondaryValidation bool `json:"requires_secondary_validation"`

	// Timestamp of rule evaluation
	EvaluatedAt time.Time `json:"evaluated_at"`

	// Rule version that was applied
	RuleVersion string `json:"rule_version"`

	// Evidence level (e.g., "Level 1A", "Level 2B", "Expert Consensus")
	EvidenceLevel string `json:"evidence_level,omitempty"`

	// Reference URLs or DOIs
	References []string `json:"references,omitempty"`
}

// GovernanceResult combines severity mapping with evidence provenance
type GovernanceResult struct {
	// Original alert/warning type
	OriginalType string `json:"original_type"`

	// Original message
	OriginalMessage string `json:"original_message"`

	// Mapped governance severity
	Severity GovernanceSeverity `json:"severity"`

	// Required actions based on severity
	Action GovernanceAction `json:"action"`

	// Evidence provenance for audit trail
	Provenance EvidenceProvenance `json:"provenance"`

	// Override code if applicable
	OverrideCode string `json:"override_code,omitempty"`

	// Override reason options
	OverrideReasonOptions []string `json:"override_reason_options,omitempty"`
}

// GovernanceEnhancedResponse wraps any response with governance metadata
type GovernanceEnhancedResponse struct {
	// Original response data
	Data interface{} `json:"data"`

	// Governance results for all alerts/warnings
	Governance []GovernanceResult `json:"governance"`

	// Overall highest severity
	HighestSeverity GovernanceSeverity `json:"highest_severity"`

	// Whether the action can proceed
	CanProceed bool `json:"can_proceed"`

	// Required steps before proceeding
	RequiredSteps []string `json:"required_steps,omitempty"`

	// Evidence provenance for the calculation
	Provenance EvidenceProvenance `json:"provenance"`
}
