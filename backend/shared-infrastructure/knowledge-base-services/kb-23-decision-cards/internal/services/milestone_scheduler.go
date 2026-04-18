package services

import "time"

// TransitionInfo carries the transition data KB-23 needs (mirrors KB-20's CareTransition).
type TransitionInfo struct {
	TransitionID     string
	PatientID        string
	DischargeDate    time.Time
	FacilityType     string
	PrimaryDiagnosis string
	WindowDays       int
}

// ScheduledMilestone is a milestone to be created.
type ScheduledMilestone struct {
	MilestoneType string
	ScheduledFor  time.Time
	Required      bool
}

// MilestoneAssessment is the result of evaluating a milestone.
type MilestoneAssessment struct {
	MilestoneType    string
	Findings         string
	CardTier         string // ROUTINE, URGENT, IMMEDIATE
	SuggestedActions string
}

// Standard milestone offsets from discharge.
const (
	MilestoneMedReconciliation48H = "MEDICATION_RECONCILIATION_48H"
	MilestoneFirstFollowup7D      = "FIRST_FOLLOWUP_7D"
	MilestoneMidpointReview14D    = "MIDPOINT_REVIEW_14D"
	MilestoneExitAssessment30D    = "EXIT_ASSESSMENT_30D"
	MilestoneEngagementCheck72H   = "ENGAGEMENT_CHECK_72H"
	MilestoneMedSupplyCheck       = "MEDICATION_SUPPLY_CHECK"
)

// ScheduleMilestones creates the milestone schedule for a transition.
// Always produces 4 required milestones at standard offsets. Conditionally
// adds ENGAGEMENT_CHECK_72H and MEDICATION_SUPPLY_CHECK based on flags.
func ScheduleMilestones(info TransitionInfo, includeEngagementCheck, includeSupplyCheck bool) []ScheduledMilestone {
	discharge := info.DischargeDate

	milestones := []ScheduledMilestone{
		{
			MilestoneType: MilestoneMedReconciliation48H,
			ScheduledFor:  discharge.Add(48 * time.Hour),
			Required:      true,
		},
		{
			MilestoneType: MilestoneFirstFollowup7D,
			ScheduledFor:  discharge.Add(168 * time.Hour),
			Required:      true,
		},
		{
			MilestoneType: MilestoneMidpointReview14D,
			ScheduledFor:  discharge.Add(336 * time.Hour),
			Required:      true,
		},
		{
			MilestoneType: MilestoneExitAssessment30D,
			ScheduledFor:  discharge.Add(720 * time.Hour),
			Required:      true,
		},
	}

	if includeEngagementCheck {
		milestones = append(milestones, ScheduledMilestone{
			MilestoneType: MilestoneEngagementCheck72H,
			ScheduledFor:  discharge.Add(72 * time.Hour),
			Required:      false,
		})
	}

	if includeSupplyCheck {
		milestones = append(milestones, ScheduledMilestone{
			MilestoneType: MilestoneMedSupplyCheck,
			ScheduledFor:  discharge.Add(24 * time.Hour),
			Required:      false,
		})
	}

	return milestones
}

// AssessMilestone48h evaluates the 48-hour medication reconciliation milestone.
// Maps reconciliation outcome to card tier:
//   - CLEAN                        → ROUTINE
//   - DISCREPANCIES_CLINICIAN_REVIEW → URGENT
//   - HIGH_RISK_URGENT             → IMMEDIATE
//   - UNCLEAR_INSUFFICIENT_DATA    → URGENT
func AssessMilestone48h(reconciliationOutcome string) MilestoneAssessment {
	var tier, findings, actions string

	switch reconciliationOutcome {
	case "CLEAN":
		tier = "ROUTINE"
		findings = "Medication reconciliation completed with no discrepancies"
		actions = "Continue standard monitoring schedule"
	case "DISCREPANCIES_CLINICIAN_REVIEW":
		tier = "URGENT"
		findings = "Medication discrepancies detected requiring clinician review"
		actions = "Schedule clinician review within 24 hours; flag discrepancies in patient record"
	case "HIGH_RISK_URGENT":
		tier = "IMMEDIATE"
		findings = "High-risk medication issues identified requiring immediate attention"
		actions = "Immediate clinician notification; hold affected medications pending review"
	case "UNCLEAR_INSUFFICIENT_DATA":
		tier = "URGENT"
		findings = "Insufficient data to complete medication reconciliation"
		actions = "Request updated medication list from patient/facility; schedule follow-up within 24 hours"
	default:
		tier = "URGENT"
		findings = "Unrecognised reconciliation outcome: " + reconciliationOutcome
		actions = "Escalate to clinical pharmacist for manual review"
	}

	return MilestoneAssessment{
		MilestoneType:    MilestoneMedReconciliation48H,
		Findings:         findings,
		CardTier:         tier,
		SuggestedActions: actions,
	}
}
