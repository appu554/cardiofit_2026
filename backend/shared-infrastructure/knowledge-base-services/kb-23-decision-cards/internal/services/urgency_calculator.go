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

// CKMSubstageInput holds optional CKM substage context for urgency escalation.
type CKMSubstageInput struct {
	CKMStage string           // "4a", "4b", "4c", or "" for non-stage-4
	HFType   string           // "HFrEF", "HFmrEF", "HFpEF" — only relevant for 4c
	MedGaps  []MandatoryMedGap // from MandatoryMedChecker
}

// CalculateDualDomainUrgency determines overall card urgency by combining
// dual-domain state, four-pillar gap counts, renal gating safety,
// therapeutic inertia assessment, and CKM substage escalation.
//
// Priority order (first match wins):
//  1. Renal CONTRAINDICATED in any medication → IMMEDIATE
//  2. Dual-domain therapeutic inertia → IMMEDIATE
//  3. Stage 4c with IMMEDIATE-urgency mandatory med gaps → IMMEDIATE
//  4. UrgentCount >= 2 → IMMEDIATE
//  5. Stage 4c with any mandatory med gaps → URGENT (escalation)
//  6. UrgentCount == 1 → URGENT
//  7. Stage 4a/4b with mandatory med gaps → URGENT (escalation)
//  8. Dual-domain "GU-HU" (glucose uncontrolled, HbA1c uncontrolled) → URGENT
//  9. "GU-HC" or "GC-HU" (one domain uncontrolled) → ROUTINE
//  10. Default → SCHEDULED
func CalculateDualDomainUrgency(
	dualDomainState string,
	fourPillar FourPillarResult,
	renalGating *models.PatientGatingReport,
	inertiaReport *models.PatientInertiaReport,
	ckmInput ...CKMSubstageInput,
) string {
	// Renal contraindication always escalates to IMMEDIATE
	if renalGating != nil && renalGating.HasContraindicated {
		return UrgencyImmediate
	}

	// Dual-domain therapeutic inertia → IMMEDIATE
	if inertiaReport != nil && inertiaReport.HasDualDomainInertia {
		return UrgencyImmediate
	}

	// Stage 4c with IMMEDIATE-urgency mandatory med gaps → IMMEDIATE
	// (e.g., HFrEF missing any of the four GDMT pillars)
	var substage *CKMSubstageInput
	if len(ckmInput) > 0 {
		substage = &ckmInput[0]
	}
	if substage != nil && substage.CKMStage == "4c" {
		for _, gap := range substage.MedGaps {
			if gap.Urgency == UrgencyImmediate {
				return UrgencyImmediate
			}
		}
	}

	// Multiple urgent pillar gaps → IMMEDIATE
	if fourPillar.UrgentCount >= 2 {
		return UrgencyImmediate
	}

	// Stage 4c with any mandatory med gaps → URGENT
	if substage != nil && substage.CKMStage == "4c" && len(substage.MedGaps) > 0 {
		return UrgencyUrgent
	}

	// Single urgent pillar gap → URGENT
	if fourPillar.UrgentCount == 1 {
		return UrgencyUrgent
	}

	// Stage 4a/4b with mandatory med gaps → URGENT
	if substage != nil && (substage.CKMStage == "4a" || substage.CKMStage == "4b") && len(substage.MedGaps) > 0 {
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
