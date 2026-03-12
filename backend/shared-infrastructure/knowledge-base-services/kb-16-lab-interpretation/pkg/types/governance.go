// Package types defines governance models for KB-16 Lab Interpretation
// Implements the v2.0 Enhanced Specification for audit defensibility and clinical authority tracking
package types

import (
	"time"
)

// =============================================================================
// LAB TEST GOVERNANCE SCHEMA (v2.0)
// =============================================================================

// LabTestGovernance provides complete governance metadata for a lab test
// This ensures audit defensibility, regulatory compliance, and clinical traceability
type LabTestGovernance struct {
	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 1: REGULATORY / PROFESSIONAL AUTHORITY
	// ═══════════════════════════════════════════════════════════════════════════

	// Reference Range Authority - Where do normal ranges come from?
	ReferenceRangeSource    string `json:"referenceRangeSource" yaml:"referenceRangeSource"`       // Authority ID (e.g., "CLSI.C28", "Tietz")
	ReferenceRangeReference string `json:"referenceRangeReference" yaml:"referenceRangeReference"` // Full citation
	ReferenceRangeMethod    string `json:"referenceRangeMethod" yaml:"referenceRangeMethod"`       // Method (e.g., "95th percentile healthy population")

	// Critical Value Authority - Where do critical thresholds come from?
	CriticalValueSource     string `json:"criticalValueSource" yaml:"criticalValueSource"`         // Authority ID (e.g., "CAP.Critical")
	CriticalValueReference  string `json:"criticalValueReference" yaml:"criticalValueReference"`   // Full citation
	NotificationRequirement string `json:"notificationRequirement" yaml:"notificationRequirement"` // Timing requirement (e.g., "30 minutes")

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 2: CLINICAL GUIDELINE AUTHORITY (Test-Specific)
	// ═══════════════════════════════════════════════════════════════════════════

	ClinicalGuidelineSource string `json:"clinicalGuidelineSource,omitempty" yaml:"clinicalGuidelineSource,omitempty"` // Guideline authority (e.g., "KDIGO.CKD", "ESC.NSTEACS")
	ClinicalGuidelineRef    string `json:"clinicalGuidelineRef,omitempty" yaml:"clinicalGuidelineRef,omitempty"`       // Full guideline citation
	InterpretationMethod    string `json:"interpretationMethod,omitempty" yaml:"interpretationMethod,omitempty"`       // Interpretation algorithm (e.g., "ESC 0/1h Protocol")

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 3: ASSAY-SPECIFIC OVERRIDES (CRITICAL for hs-Troponin, hormones)
	// ═══════════════════════════════════════════════════════════════════════════

	// AssayDependency indicates whether thresholds depend on the specific assay/platform
	// Values: "ASSAY_SPECIFIC" (e.g., hs-Troponin), "STANDARDIZED" (e.g., potassium), "METHOD_DEPENDENT"
	AssayDependency string `json:"assayDependency" yaml:"assayDependency"`

	// AssaySpecificThresholds contains manufacturer-specific cutoffs
	// Required when AssayDependency = "ASSAY_SPECIFIC"
	AssaySpecificThresholds []AssayThreshold `json:"assaySpecificThresholds,omitempty" yaml:"assaySpecificThresholds,omitempty"`

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYER 4: LOCAL POLICY ALLOWANCE (Hospital-specific overrides)
	// ═══════════════════════════════════════════════════════════════════════════

	// LocalPolicyAllowed indicates whether hospital can override default thresholds
	// Some tests (e.g., hs-Troponin) CANNOT be overridden due to assay-specific validation
	LocalPolicyAllowed bool `json:"localPolicyAllowed" yaml:"localPolicyAllowed"`

	// LocalPolicyScope defines what can be overridden
	// Values: "REFERENCE_RANGE", "CRITICAL_VALUE", "BOTH", "NONE"
	LocalPolicyScope string `json:"localPolicyScope,omitempty" yaml:"localPolicyScope,omitempty"`

	// LocalPolicyJustification explains why local override is or isn't allowed
	LocalPolicyJustification string `json:"localPolicyJustification,omitempty" yaml:"localPolicyJustification,omitempty"`

	// ═══════════════════════════════════════════════════════════════════════════
	// METADATA & AUDIT TRAIL
	// ═══════════════════════════════════════════════════════════════════════════

	// Delta Threshold Authority
	DeltaThresholdSource    string `json:"deltaThresholdSource,omitempty" yaml:"deltaThresholdSource,omitempty"`
	DeltaThresholdReference string `json:"deltaThresholdReference,omitempty" yaml:"deltaThresholdReference,omitempty"`

	// Quality Metadata
	EvidenceLevel string `json:"evidenceLevel" yaml:"evidenceLevel"` // HIGH, MODERATE, LOW, VERY_LOW
	Jurisdiction  string `json:"jurisdiction" yaml:"jurisdiction"`   // GLOBAL, US, EU, IN, AU, etc.

	// Review/Audit Metadata
	LastReviewed  string `json:"lastReviewed" yaml:"lastReviewed"`   // ISO date
	ReviewedBy    string `json:"reviewedBy" yaml:"reviewedBy"`       // Lab Director credentials
	Version       string `json:"version" yaml:"version"`             // Semantic version
	EffectiveDate string `json:"effectiveDate" yaml:"effectiveDate"` // ISO date
}

// AssayThreshold contains manufacturer-specific thresholds for an assay
// Required for assay-dependent tests like hs-Troponin, some hormones
type AssayThreshold struct {
	// Manufacturer Information
	Manufacturer string `json:"manufacturer" yaml:"manufacturer"` // "Roche", "Abbott", "Siemens", "Beckman"
	Platform     string `json:"platform" yaml:"platform"`         // "Cobas e801", "Architect i2000", "Atellica IM"
	AssayName    string `json:"assayName" yaml:"assayName"`       // "Elecsys hs-TnT", "ARCHITECT STAT High Sensitive Troponin-I"

	// Thresholds
	ReferenceHigh  *float64 `json:"referenceHigh,omitempty" yaml:"referenceHigh,omitempty"`   // Upper reference limit
	ReferenceLow   *float64 `json:"referenceLow,omitempty" yaml:"referenceLow,omitempty"`     // Lower reference limit
	URL99thPercent *float64 `json:"url99thPercent,omitempty" yaml:"url99thPercent,omitempty"` // 99th percentile URL (for troponin)

	// Sex-Specific Thresholds (optional)
	MaleURL99thPercent   *float64 `json:"maleUrl99thPercent,omitempty" yaml:"maleUrl99thPercent,omitempty"`
	FemaleURL99thPercent *float64 `json:"femaleUrl99thPercent,omitempty" yaml:"femaleUrl99thPercent,omitempty"`

	// Regulatory Clearance
	FDACleared       bool   `json:"fdaCleared" yaml:"fdaCleared"`                                 // FDA 510(k) cleared
	FDAClearanceNum  string `json:"fdaClearanceNum,omitempty" yaml:"fdaClearanceNum,omitempty"`   // e.g., "K173327"
	CEMarked         bool   `json:"ceMarked,omitempty" yaml:"ceMarked,omitempty"`                 // CE marked for EU
	PackageInsertRef string `json:"packageInsertRef,omitempty" yaml:"packageInsertRef,omitempty"` // Package insert reference

	// Effective dates
	EffectiveDate string `json:"effectiveDate" yaml:"effectiveDate"`
	ExpiresDate   string `json:"expiresDate,omitempty" yaml:"expiresDate,omitempty"` // When superseded by newer version
}

// =============================================================================
// AUTHORITY REGISTRY
// =============================================================================

// Authority represents a registered clinical/regulatory authority
type Authority struct {
	ID            string   `json:"id" yaml:"id"`                       // Unique identifier (e.g., "CLSI.C28")
	Name          string   `json:"name" yaml:"name"`                   // Full name
	Layer         string   `json:"layer" yaml:"layer"`                 // REGULATORY, SCIENTIFIC, CLINICAL
	Jurisdiction  string   `json:"jurisdiction" yaml:"jurisdiction"`   // GLOBAL, US, EU, IN, AU
	AuthorityType string   `json:"authorityType" yaml:"authorityType"` // PRIMARY, SECONDARY
	URL           string   `json:"url,omitempty" yaml:"url,omitempty"` // Reference URL
	Description   string   `json:"description" yaml:"description"`     // What this authority covers
	UseFor        []string `json:"useFor" yaml:"useFor"`               // ["reference_ranges", "critical_values", etc.]
}

// =============================================================================
// ASSAY DEPENDENCY CONSTANTS
// =============================================================================

// Assay dependency types
const (
	AssayDependencyStandardized   = "STANDARDIZED"    // Thresholds consistent across platforms (e.g., potassium)
	AssayDependencyAssaySpecific  = "ASSAY_SPECIFIC"  // Thresholds vary by manufacturer (e.g., hs-Troponin)
	AssayDependencyMethodDependent = "METHOD_DEPENDENT" // Thresholds vary by analytical method
)

// Local policy scope values
const (
	LocalPolicyScopeNone           = "NONE"
	LocalPolicyScopeReferenceRange = "REFERENCE_RANGE"
	LocalPolicyScopeCriticalValue  = "CRITICAL_VALUE"
	LocalPolicyScopeBoth           = "BOTH"
)

// Evidence level constants (GRADE)
const (
	EvidenceLevelHigh     = "HIGH"
	EvidenceLevelModerate = "MODERATE"
	EvidenceLevelLow      = "LOW"
	EvidenceLevelVeryLow  = "VERY_LOW"
)

// Jurisdiction constants
const (
	JurisdictionGlobal = "GLOBAL"
	JurisdictionUS     = "US"
	JurisdictionEU     = "EU"
	JurisdictionIN     = "IN" // India
	JurisdictionAU     = "AU" // Australia
	JurisdictionUK     = "UK"
)

// Authority layer constants
const (
	AuthorityLayerRegulatory = "REGULATORY"
	AuthorityLayerScientific = "SCIENTIFIC"
	AuthorityLayerClinical   = "CLINICAL"
)

// =============================================================================
// GOVERNANCE VALIDATION
// =============================================================================

// GovernanceValidationResult contains validation results for governance metadata
type GovernanceValidationResult struct {
	IsValid       bool     `json:"isValid"`
	Errors        []string `json:"errors,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	MissingFields []string `json:"missingFields,omitempty"`
}

// ValidateGovernance checks if governance metadata meets requirements
func (g *LabTestGovernance) Validate() *GovernanceValidationResult {
	result := &GovernanceValidationResult{IsValid: true}

	// Required fields
	if g.ReferenceRangeSource == "" {
		result.Errors = append(result.Errors, "referenceRangeSource is required")
		result.MissingFields = append(result.MissingFields, "referenceRangeSource")
		result.IsValid = false
	}

	if g.EvidenceLevel == "" {
		result.Errors = append(result.Errors, "evidenceLevel is required")
		result.MissingFields = append(result.MissingFields, "evidenceLevel")
		result.IsValid = false
	}

	if g.Version == "" {
		result.Errors = append(result.Errors, "version is required")
		result.MissingFields = append(result.MissingFields, "version")
		result.IsValid = false
	}

	// Assay-specific validation
	if g.AssayDependency == AssayDependencyAssaySpecific && len(g.AssaySpecificThresholds) == 0 {
		result.Errors = append(result.Errors, "assaySpecificThresholds required when assayDependency is ASSAY_SPECIFIC")
		result.IsValid = false
	}

	// Local policy validation
	if !g.LocalPolicyAllowed && g.LocalPolicyScope != "" && g.LocalPolicyScope != LocalPolicyScopeNone {
		result.Warnings = append(result.Warnings, "localPolicyScope set but localPolicyAllowed is false")
	}

	// Warnings for optional but recommended fields
	if g.LastReviewed == "" {
		result.Warnings = append(result.Warnings, "lastReviewed not set - governance may be stale")
	}

	if g.ReviewedBy == "" {
		result.Warnings = append(result.Warnings, "reviewedBy not set - no audit trail for review")
	}

	return result
}

// IsAssaySpecific returns true if this test requires assay-specific thresholds
func (g *LabTestGovernance) IsAssaySpecific() bool {
	return g.AssayDependency == AssayDependencyAssaySpecific
}

// GetAssayThreshold returns the threshold for a specific manufacturer/platform
func (g *LabTestGovernance) GetAssayThreshold(manufacturer, platform string) *AssayThreshold {
	for i := range g.AssaySpecificThresholds {
		t := &g.AssaySpecificThresholds[i]
		if t.Manufacturer == manufacturer && (platform == "" || t.Platform == platform) {
			return t
		}
	}
	return nil
}

// CanOverride returns true if local policy can override the specified scope
func (g *LabTestGovernance) CanOverride(scope string) bool {
	if !g.LocalPolicyAllowed {
		return false
	}
	switch g.LocalPolicyScope {
	case LocalPolicyScopeBoth:
		return true
	case LocalPolicyScopeReferenceRange:
		return scope == LocalPolicyScopeReferenceRange
	case LocalPolicyScopeCriticalValue:
		return scope == LocalPolicyScopeCriticalValue
	default:
		return false
	}
}

// =============================================================================
// GOVERNANCE AUDIT EVENTS
// =============================================================================

// GovernanceAuditEvent represents an audit trail entry for governance changes
type GovernanceAuditEvent struct {
	ID            string    `json:"id"`
	TestCode      string    `json:"testCode"`
	EventType     string    `json:"eventType"` // CREATED, UPDATED, REVIEWED, OVERRIDE_APPLIED
	PreviousValue string    `json:"previousValue,omitempty"`
	NewValue      string    `json:"newValue,omitempty"`
	ChangedBy     string    `json:"changedBy"`
	Reason        string    `json:"reason,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// Governance audit event types
const (
	GovernanceEventCreated         = "CREATED"
	GovernanceEventUpdated         = "UPDATED"
	GovernanceEventReviewed        = "REVIEWED"
	GovernanceEventOverrideApplied = "OVERRIDE_APPLIED"
	GovernanceEventAssayAdded      = "ASSAY_THRESHOLD_ADDED"
	GovernanceEventAssayUpdated    = "ASSAY_THRESHOLD_UPDATED"
)
