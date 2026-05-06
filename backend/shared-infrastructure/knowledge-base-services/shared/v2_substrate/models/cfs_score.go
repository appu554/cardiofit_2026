// Package models — CFSScore is the Clinical Frailty Scale (Rockwood) capture
// entity introduced by Wave 2.6 of the Layer 2 substrate plan
// (Layer 2 doc §2.4 / §2.6).
//
// CFS is a clinician-entered score on a 1-9 scale (Rockwood 2020 revision):
//
//	1  Very fit
//	2  Well
//	3  Managing well
//	4  Living with very mild frailty
//	5  Living with mild frailty
//	6  Living with moderate frailty
//	7  Living with severe frailty
//	8  Living with very severe frailty
//	9  Terminally ill
//
// The substrate captures CFS as an append-only score history per Resident.
// CFS≥7 surfaces a worklist hint suggesting a care-intensity review; the
// substrate never auto-transitions care intensity from a CFS value (Layer 2
// doc §2.4 explicitly preserves clinician judgement).
//
// Canonical storage: kb-20-patient-profile (cfs_scores table, migration 018).
// The latest row by AssessedAt per ResidentRef is the current score (queried
// via the cfs_current view).
//
// FHIR boundary: not mapped in MVP — CFS is a Vaidshala-internal informational
// score per the plan (no FHIR mapping required for Wave 2.6 acceptance).
package models

import (
	"time"

	"github.com/google/uuid"
)

// CFSScore captures one Clinical Frailty Scale assessment for a Resident.
// Score is constrained to [1, 9] per the Rockwood scale; the validator
// rejects out-of-range values, and the cfs_scores.score CHECK constraint
// is the storage-level backstop.
type CFSScore struct {
	ID                uuid.UUID `json:"id"`
	ResidentRef       uuid.UUID `json:"resident_ref"`
	AssessedAt        time.Time `json:"assessed_at"`
	AssessorRoleRef   uuid.UUID `json:"assessor_role_ref"`
	InstrumentVersion string    `json:"instrument_version"` // e.g. "v2.0" (Rockwood 2020)
	Score             int       `json:"score"`              // 1-9 inclusive
	Rationale         string    `json:"rationale,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// CFSCareIntensityReviewThreshold is the CFS value at or above which a
// care-intensity review hint is surfaced. Layer 2 doc §2.4 (line 540-547)
// flags CFS≥7 ("severe frailty") as the trigger for considering a
// palliative tag — the substrate does not auto-transition; it surfaces
// the hint for clinician review.
const CFSCareIntensityReviewThreshold = 7

// CFSScoreShouldHintCareIntensityReview reports whether a score warrants
// surfacing a care-intensity review hint to the clinician worklist. Used by
// the storage layer's hint-emission path; pure logic so it can be unit-tested
// without DB.
func CFSScoreShouldHintCareIntensityReview(score int) bool {
	return score >= CFSCareIntensityReviewThreshold
}
