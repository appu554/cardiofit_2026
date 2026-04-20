package services

import (
	"fmt"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ResolutionResult is the outcome of processing a worklist action.
type ResolutionResult struct {
	UpdatedItem *models.WorklistItem
	Feedback    *models.WorklistFeedback
	Error       error
}

// HandleWorklistAction processes a clinician action on a worklist item and
// returns the updated item, optional feedback, or an error.
func HandleWorklistAction(item *models.WorklistItem, req models.WorklistActionRequest) ResolutionResult {
	switch req.ActionCode {

	case "ACKNOWLEDGE":
		item.ResolutionState = models.ResolutionResolved
		return ResolutionResult{UpdatedItem: item}

	case "CALL_PATIENT", "RECHECK_VITALS", "VISIT_TODAY", "TELECONSULT", "MEDICATION_REVIEW":
		item.ResolutionState = models.ResolutionInProgress
		return ResolutionResult{UpdatedItem: item}

	case "DEFER":
		item.ResolutionState = models.ResolutionDeferred
		return ResolutionResult{UpdatedItem: item}

	case "DISMISS":
		item.ResolutionState = models.ResolutionResolved
		feedback := &models.WorklistFeedback{
			PatientID:    req.PatientID,
			ClinicianID:  req.ClinicianID,
			FeedbackType: "NOT_USEFUL",
			Reason:       req.Notes,
			SubmittedAt:  time.Now(),
		}
		return ResolutionResult{UpdatedItem: item, Feedback: feedback}

	case "ESCALATE_TO_GP", "REFERRAL", "CALL_ANM":
		item.ResolutionState = models.ResolutionHandedOff
		return ResolutionResult{UpdatedItem: item}

	default:
		return ResolutionResult{
			Error: fmt.Errorf("unknown action code: %s", req.ActionCode),
		}
	}
}
