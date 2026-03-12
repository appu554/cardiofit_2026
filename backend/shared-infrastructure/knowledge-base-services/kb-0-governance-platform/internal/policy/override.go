// Package policy provides governance policy evaluation for clinical facts.
package policy

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// OVERRIDE POLICY
// =============================================================================
// Determines when and how clinical facts can be overridden.
//
// Override Types:
//   - EMERGENCY: CMO/attending physician override for urgent clinical need (4h max)
//   - INSTITUTIONAL: Hospital-specific policy override (requires P&T approval)
//   - CLINICAL_JUDGMENT: Documented clinical rationale for specific patient
//
// Required Documentation:
//   - Override reason
//   - Clinical justification
//   - Expected duration
//   - Responsible clinician with credentials
//
// Audit Requirements (21 CFR Part 11):
//   - All overrides must be logged
//   - Override expiration enforced
//   - Override acknowledgments tracked
// =============================================================================

// OverrideType represents the type of override being requested.
type OverrideType string

const (
	OverrideTypeEmergency         OverrideType = "EMERGENCY"
	OverrideTypeInstitutional     OverrideType = "INSTITUTIONAL"
	OverrideTypeClinicalJudgment  OverrideType = "CLINICAL_JUDGMENT"
)

// OverrideRequest represents a request to override a clinical fact.
type OverrideRequest struct {
	FactID            uuid.UUID    `json:"factId"`
	OverrideType      OverrideType `json:"overrideType"`
	Reason            string       `json:"reason"`
	ClinicalRationale string       `json:"clinicalRationale"`
	RequestedDuration time.Duration `json:"requestedDuration,omitempty"`

	// Requestor information
	RequestorID       string `json:"requestorId"`
	RequestorName     string `json:"requestorName"`
	RequestorRole     string `json:"requestorRole"`
	RequestorCredentials string `json:"requestorCredentials"` // MD, PharmD, etc.

	// Context
	PatientID         string `json:"patientId,omitempty"` // For patient-specific overrides
	EncounterID       string `json:"encounterId,omitempty"`
}

// OverrideConstraints defines the limits for different override types.
type OverrideConstraints struct {
	MaxDuration        time.Duration `json:"maxDuration"`
	RequiredRole       string        `json:"requiredRole"`
	RequiresPatientID  bool          `json:"requiresPatientId"`
	RequiresApproval   bool          `json:"requiresApproval"`
	ApproverRole       string        `json:"approverRole,omitempty"`
}

// DefaultOverrideConstraints returns the default constraints for each override type.
func DefaultOverrideConstraints() map[OverrideType]OverrideConstraints {
	return map[OverrideType]OverrideConstraints{
		OverrideTypeEmergency: {
			MaxDuration:       4 * time.Hour,
			RequiredRole:      "physician",
			RequiresPatientID: true,
			RequiresApproval:  false, // Immediate, but CMO notified
		},
		OverrideTypeInstitutional: {
			MaxDuration:       0, // No limit (until revoked)
			RequiredRole:      "pharmacist",
			RequiresPatientID: false,
			RequiresApproval:  true,
			ApproverRole:      "pt_chair", // P&T Committee approval
		},
		OverrideTypeClinicalJudgment: {
			MaxDuration:       24 * time.Hour,
			RequiredRole:      "physician",
			RequiresPatientID: true,
			RequiresApproval:  false,
		},
	}
}

// EvaluateOverride determines if an override request is allowed.
// This is a pure function - no side effects, no database writes.
func EvaluateOverride(request *OverrideRequest, fact *ClinicalFact) OverrideDecision {
	now := time.Now()
	constraints := DefaultOverrideConstraints()[request.OverrideType]

	// Rule 1: Check if fact type can be overridden
	if !isOverridable(fact.FactType) {
		return OverrideDecision{
			Allowed:     false,
			Reason:      "This fact type cannot be overridden - it represents absolute contraindications",
			EvaluatedAt: now,
		}
	}

	// Rule 2: Check requestor role
	if !hasRequiredRole(request.RequestorRole, constraints.RequiredRole) {
		return OverrideDecision{
			Allowed:      false,
			Reason:       "Requestor does not have the required role for this override type",
			RequiredRole: constraints.RequiredRole,
			EvaluatedAt:  now,
		}
	}

	// Rule 3: Check patient ID requirement
	if constraints.RequiresPatientID && request.PatientID == "" {
		return OverrideDecision{
			Allowed:     false,
			Reason:      "Patient ID is required for this override type",
			EvaluatedAt: now,
		}
	}

	// Rule 4: Check clinical rationale
	if request.ClinicalRationale == "" {
		return OverrideDecision{
			Allowed:     false,
			Reason:      "Clinical rationale is required for all overrides",
			EvaluatedAt: now,
		}
	}

	// Rule 5: Check credentials for high-risk overrides
	if request.OverrideType == OverrideTypeEmergency && request.RequestorCredentials == "" {
		return OverrideDecision{
			Allowed:     false,
			Reason:      "Professional credentials must be documented for emergency overrides",
			EvaluatedAt: now,
		}
	}

	// Rule 6: Calculate expiration
	var expiresAt *time.Time
	if constraints.MaxDuration > 0 {
		duration := constraints.MaxDuration
		if request.RequestedDuration > 0 && request.RequestedDuration < constraints.MaxDuration {
			duration = request.RequestedDuration
		}
		expiry := now.Add(duration)
		expiresAt = &expiry
	}

	// Override allowed
	return OverrideDecision{
		Allowed: true,
		Reason:  "Override request meets all requirements",
		Constraints: map[string]interface{}{
			"maxDuration":      constraints.MaxDuration.String(),
			"requiresApproval": constraints.RequiresApproval,
			"approverRole":     constraints.ApproverRole,
		},
		ExpiresAt:    expiresAt,
		RequiredRole: constraints.RequiredRole,
		EvaluatedAt:  now,
	}
}

// =============================================================================
// STABILITY POLICY
// =============================================================================
// Determines if a clinical fact has been active long enough to be superseded.
// This prevents rapid flip-flopping of clinical rules.
// =============================================================================

// EvaluateStability determines if a fact is stable enough to be superseded.
func EvaluateStability(fact *ClinicalFact, minActiveHours int) StabilityDecision {
	now := time.Now()

	// Fact must be ACTIVE to have stability
	if fact.Status != FactStatusActive {
		return StabilityDecision{
			IsStable:           false,
			MinActiveHours:     minActiveHours,
			CurrentActiveHours: 0,
			Reason:             "Fact is not currently active",
			CanSupersede:       true, // Non-active facts can be superseded anytime
			EvaluatedAt:        now,
		}
	}

	// Calculate how long fact has been active
	activeHours := now.Sub(fact.EffectiveFrom).Hours()

	if activeHours < float64(minActiveHours) {
		return StabilityDecision{
			IsStable:           false,
			MinActiveHours:     minActiveHours,
			CurrentActiveHours: activeHours,
			Reason:             "Fact has not been active long enough for stability check",
			CanSupersede:       false,
			EvaluatedAt:        now,
		}
	}

	return StabilityDecision{
		IsStable:           true,
		MinActiveHours:     minActiveHours,
		CurrentActiveHours: activeHours,
		Reason:             "Fact has been stable and can be safely superseded",
		CanSupersede:       true,
		EvaluatedAt:        now,
	}
}

// DefaultMinStabilityHours returns the minimum active hours for different fact types.
func DefaultMinStabilityHours() map[FactType]int {
	return map[FactType]int{
		FactTypeSafetySignal:       168, // 7 days - safety rules need long stability
		FactTypeInteraction:        72,  // 3 days - DDI rules need moderate stability
		FactTypeOrganImpairment:    48,  // 2 days
		FactTypeReproductiveSafety: 168, // 7 days
		FactTypeFormulary:          24,  // 1 day - formulary can change faster
		FactTypeLabReference:       24,  // 1 day - lab ranges are relatively stable
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// isOverridable returns true if the fact type can be overridden.
// Some absolute contraindications should never be overridden.
func isOverridable(factType FactType) bool {
	// All current fact types are overridable with proper justification
	// In the future, we might have absolute contraindications that cannot be overridden
	return true
}

// hasRequiredRole checks if the requestor has the required role.
func hasRequiredRole(requestorRole, requiredRole string) bool {
	// Simple role hierarchy
	roleHierarchy := map[string]int{
		"system":     100,
		"cmo":        90,
		"director":   80,
		"pt_chair":   75,
		"physician":  70,
		"pharmacist": 60,
		"nurse":      50,
		"tech":       40,
	}

	requestorLevel := roleHierarchy[requestorRole]
	requiredLevel := roleHierarchy[requiredRole]

	// Equal or higher level is allowed
	return requestorLevel >= requiredLevel
}

// =============================================================================
// OVERRIDE ACKNOWLEDGMENT
// =============================================================================

// OverrideAcknowledgment represents acknowledgment of an active override.
type OverrideAcknowledgment struct {
	OverrideID     uuid.UUID `json:"overrideId"`
	AcknowledgedBy string    `json:"acknowledgedBy"`
	AcknowledgedAt time.Time `json:"acknowledgedAt"`
	Notes          string    `json:"notes,omitempty"`
}

// ValidateAcknowledgment checks if an acknowledgment is valid.
func ValidateAcknowledgment(ack *OverrideAcknowledgment) error {
	if ack.OverrideID == uuid.Nil {
		return ErrInvalidOverrideID
	}
	if ack.AcknowledgedBy == "" {
		return ErrMissingAcknowledger
	}
	return nil
}

// Custom errors for override operations
type OverrideError string

func (e OverrideError) Error() string { return string(e) }

const (
	ErrInvalidOverrideID    OverrideError = "invalid override ID"
	ErrMissingAcknowledger  OverrideError = "acknowledger is required"
	ErrOverrideExpired      OverrideError = "override has expired"
	ErrOverrideNotFound     OverrideError = "override not found"
)
