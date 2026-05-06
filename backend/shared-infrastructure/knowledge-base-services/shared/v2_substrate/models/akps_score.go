// Package models — AKPSScore is the Australia-modified Karnofsky Performance
// Status capture entity introduced by Wave 2.6 of the Layer 2 substrate plan
// (Layer 2 doc §2.4 / §2.6).
//
// AKPS is a clinician-entered functional-status score on a 0-100 scale in
// 10-point increments (Abernethy et al. 2005). Lower scores indicate higher
// dependency / palliative status; AKPS≤40 surfaces a worklist hint suggesting
// a care-intensity review (Layer 2 doc §2.4 line 540-547).
//
// Canonical storage: kb-20-patient-profile (akps_scores table, migration 018).
// The latest row by AssessedAt per ResidentRef is the current score (queried
// via the akps_current view).
//
// FHIR boundary: not mapped in MVP — AKPS is a Vaidshala-internal informational
// score per the plan.
package models

import (
	"time"

	"github.com/google/uuid"
)

// AKPSScore captures one Australia-modified Karnofsky Performance Status
// assessment for a Resident. Score is 0-100 in 10-point increments; the
// validator rejects out-of-range and non-multiple-of-10 values, and the
// akps_scores.score CHECK constraint is the storage-level backstop.
type AKPSScore struct {
	ID                uuid.UUID `json:"id"`
	ResidentRef       uuid.UUID `json:"resident_ref"`
	AssessedAt        time.Time `json:"assessed_at"`
	AssessorRoleRef   uuid.UUID `json:"assessor_role_ref"`
	InstrumentVersion string    `json:"instrument_version"` // e.g. "abernethy_2005"
	Score             int       `json:"score"`              // 0-100, multiples of 10
	Rationale         string    `json:"rationale,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// AKPSCareIntensityReviewThreshold is the AKPS value at or below which a
// care-intensity review hint is surfaced. Layer 2 doc §2.4 (line 540-547)
// flags AKPS≤40 as the trigger for considering a palliative tag — the
// substrate does not auto-transition; it surfaces the hint for clinician
// review.
const AKPSCareIntensityReviewThreshold = 40

// AKPSScoreShouldHintCareIntensityReview reports whether a score warrants
// surfacing a care-intensity review hint to the clinician worklist. Used by
// the storage layer's hint-emission path; pure logic so it can be unit-tested
// without DB.
func AKPSScoreShouldHintCareIntensityReview(score int) bool {
	return score <= AKPSCareIntensityReviewThreshold
}
