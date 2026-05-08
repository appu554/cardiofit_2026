// Package dashboards provides the pharmacist self-visibility dashboard surfaces.
//
// This file implements Surface 5: My CPD Progression.
// Auto-tagged CPD activities from clinical work; pharmacist confirms each before
// it counts toward AHPRA-required hours. Confirmed hours are aggregated by
// activity category. Reflective entries may link to individual activities.
//
// VisibilityClass: WO for activity log (employer can see compliance status);
// POA for any linked reflective content (see Task 1 ReflectiveEntry).
package dashboards

import (
	"context"

	"github.com/google/uuid"
)

// CPDActivity represents a single continuing professional development activity
// that has been auto-tagged from clinical work. The pharmacist must confirm
// each activity before it contributes to AHPRA-required hours.
//
// VisibilityClass: WO for activity log (employer can see compliance status);
// POA for any reflective content linked via SourceRef.
type CPDActivity struct {
	ID            uuid.UUID
	AHPRACategory string  // e.g. "clinical_review", "education", "audit_feedback"
	Hours         float64
	Status        string // "pending_confirmation" | "confirmed" | "rejected"
	SourceRef     string // recommendation/case ID that generated this activity
}

// CPDView is the pharmacist's read-only view of their CPD progression.
//
// VisibilityClass: WO for the activity log; employer can see compliance status.
// Any linked reflective content is POA (pharmacist-only access, per Task 1).
//
// ConfirmedHours maps AHPRA category keys to the total confirmed hours in that
// category. It is always non-nil (even when empty) so callers can distinguish
// "no confirmed activities" from "data unavailable".
type CPDView struct {
	// ConfirmedHours maps each AHPRA category to the sum of confirmed hours.
	// Never nil; an empty map means no confirmed activities yet.
	ConfirmedHours map[string]float64

	// PendingConfirmation is the count of activities awaiting pharmacist review.
	PendingConfirmation int
}

// CPDSource is the data-access interface backing CPD.
//
// Implementations must:
//   - Respect context cancellation.
//   - Return activities for the specified pharmacist only.
type CPDSource interface {
	// Activities returns all CPD activities for the given pharmacist, including
	// confirmed, pending, and rejected records.
	Activities(ctx context.Context, pharmacistID uuid.UUID) ([]CPDActivity, error)
}

// CPD implements Surface 5 — My CPD Progression.
//
// Construct with NewCPD; call For to obtain the CPDView for a specific pharmacist.
type CPD struct{ src CPDSource }

// NewCPD returns a CPD backed by the given CPDSource.
func NewCPD(s CPDSource) *CPD { return &CPD{src: s} }

// For returns the CPDView for the given pharmacist.
//
// Walk logic:
//   - "confirmed"             → add Hours to ConfirmedHours[AHPRACategory]
//   - "pending_confirmation"  → increment PendingConfirmation
//   - "rejected" (and any other status) → ignored entirely
//
// ConfirmedHours is always non-nil in the returned view.
func (c *CPD) For(ctx context.Context, pharmacistID uuid.UUID) (CPDView, error) {
	acts, err := c.src.Activities(ctx, pharmacistID)
	if err != nil {
		return CPDView{}, err
	}
	v := CPDView{ConfirmedHours: map[string]float64{}}
	for _, a := range acts {
		switch a.Status {
		case "confirmed":
			v.ConfirmedHours[a.AHPRACategory] += a.Hours
		case "pending_confirmation":
			v.PendingConfirmation++
		// "rejected" and any unknown status are intentionally ignored
		}
	}
	return v, nil
}

// IsValidCPDStatus reports whether s is one of the three recognised CPD activity
// statuses: "pending_confirmation", "confirmed", or "rejected".
// Comparison is case-sensitive; unknown or empty strings return false.
func IsValidCPDStatus(s string) bool {
	switch s {
	case "pending_confirmation", "confirmed", "rejected":
		return true
	default:
		return false
	}
}
