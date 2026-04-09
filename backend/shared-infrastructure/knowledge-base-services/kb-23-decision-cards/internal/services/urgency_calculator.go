package services

import "kb-23-decision-cards/internal/models"

// ---------------------------------------------------------------------------
// Urgency constants
// ---------------------------------------------------------------------------

const (
	UrgencyImmediate = "IMMEDIATE"
	UrgencyUrgent    = "URGENT"
	UrgencyRoutine   = "ROUTINE"
	UrgencyScheduled = "SCHEDULED"
)

// ---------------------------------------------------------------------------
// CalculateDualDomainUrgency — urgency with renal safety override
// ---------------------------------------------------------------------------

// CalculateDualDomainUrgency determines overall card urgency by combining
// dual-domain state, four-pillar gap counts, and renal gating safety.
//
// Priority order (first match wins):
//  1. Renal CONTRAINDICATED in any medication → IMMEDIATE
//  2. UrgentCount >= 2 → IMMEDIATE
//  3. UrgentCount == 1 → URGENT
//  4. Dual-domain "GU-HU" (glucose uncontrolled, HbA1c uncontrolled) → URGENT
//  5. "GU-HC" or "GC-HU" (one domain uncontrolled) → ROUTINE
//  6. Default → SCHEDULED
func CalculateDualDomainUrgency(
	dualDomainState string,
	fourPillar FourPillarResult,
	renalGating *models.PatientGatingReport,
) string {
	// Renal contraindication always escalates to IMMEDIATE
	if renalGating != nil && renalGating.HasContraindicated {
		return UrgencyImmediate
	}

	// Multiple urgent pillar gaps → IMMEDIATE
	if fourPillar.UrgentCount >= 2 {
		return UrgencyImmediate
	}

	// Single urgent pillar gap → URGENT
	if fourPillar.UrgentCount == 1 {
		return UrgencyUrgent
	}

	// Dual-domain state classification
	switch dualDomainState {
	case "GU-HU":
		return UrgencyUrgent
	case "GU-HC", "GC-HU":
		return UrgencyRoutine
	default:
		return UrgencyScheduled
	}
}
