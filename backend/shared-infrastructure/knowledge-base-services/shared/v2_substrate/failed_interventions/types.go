// Package failed_interventions implements the Failed Intervention History
// substrate — CAPE Layer 4 veto pattern (CAPE Guidelines v1.1 §4.3, lines
// 627–660).
//
// VisibilityClass: PDP (Pharmacist-Default-Private) — resident clinical record.
//
// A FailedInterventionRecord documents that a clinical intervention was
// attempted and subsequently reversed (e.g. an antipsychotic deprescribing
// was tried and the resident's BPSD returned, prompting reinstatement). CAPE
// Layer 4 reads these records as veto factors against re-attempting the
// same intervention class within the retry-eligibility window (typically
// 12 months from the attempt date).
//
// Records are written by the kb-32 override-capture flow when the override's
// outcome matches a documented-reversal vocabulary (see Outcome* constants).
// No separate entry workflow exists — the audit trail lives on the override
// store, and this substrate is a derived index optimised for CAPE's
// IsVetoActive lookup. See CAPE Guidelines v1.1 line 659.
package failed_interventions

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// FailedInterventionRecord documents a clinical intervention that was attempted
// and later reversed. CAPE Layer 4 reads these as veto factors against
// re-attempting the same intervention class within the retry-eligibility window.
//
// Spec: CAPE_Implementation_Guidelines_v1_1.md lines 632–640 (verbatim Go
// struct). DocumentedBy is the pharmacist UUID who captured the reversal
// (PharmacistID in the spec).
type FailedInterventionRecord struct {
	// ResidentID is the resident whose intervention failed. Required.
	ResidentID uuid.UUID

	// InterventionType is the canonical class of intervention, e.g.
	// "antipsychotic_deprescribing", "benzodiazepine_deprescribing".
	// Match is case-insensitive in IsVetoActive. The canonical vocabulary
	// is produced by ClassifyInterventionType (classifier.go).
	InterventionType string

	// AttemptDate is when the intervention was originally tried.
	AttemptDate time.Time

	// Outcome describes how the intervention failed. Should be one of the
	// Outcome* constants below for CAPE recognition, but free-form clinical
	// text is allowed and will be persisted verbatim.
	Outcome string

	// DocumentedReason is free-form audit-trail rationale captured at the
	// reversal moment (typically the pharmacist's narrative from the
	// override-capture flow).
	DocumentedReason string

	// RetryEligibleDate is the wall-clock time after which CAPE will no
	// longer treat this record as an active veto. Typically AttemptDate +
	// DefaultRetryWindow (12 months), but callers may set custom windows.
	RetryEligibleDate time.Time

	// DocumentedBy is the PharmacistID (UUID) of the clinician who recorded
	// the reversal. Required for audit traceability.
	DocumentedBy uuid.UUID
}

// DefaultRetryWindow is the standard retry-eligibility window applied after a
// failed intervention (12 months, per CAPE Guidelines line 638).
const DefaultRetryWindow = 365 * 24 * time.Hour

// Outcome constants — the failure-outcome vocabulary CAPE recognises.
// Anything outside this set is allowed (free-form clinical text), but
// these are the values that classify cleanly for veto reasoning.
const (
	// OutcomeReversedDueToBPSDRecurrence — intervention reversed because
	// Behavioural and Psychological Symptoms of Dementia (BPSD) returned.
	// Canonical example in CAPE Guidelines line 636.
	OutcomeReversedDueToBPSDRecurrence = "reversed_due_to_BPSD_recurrence"

	// OutcomeReversedDueToFamilyRequest — intervention reversed at the
	// family / decision-maker's documented request (CAPE Guidelines line 636).
	OutcomeReversedDueToFamilyRequest = "reversed_due_to_family_request"

	// OutcomeReversedDueToClinicalDecline — intervention reversed because
	// the resident's clinical status declined after the change.
	OutcomeReversedDueToClinicalDecline = "reversed_due_to_clinical_decline"

	// OutcomeReversedDueToFrailty — intervention reversed because the
	// resident's frailty profile made continuation inadvisable.
	OutcomeReversedDueToFrailty = "reversed_due_to_frailty"

	// OutcomeGoalsOfCareAligned — intervention reversed because goals-of-care
	// alignment indicated the original regimen was clinically appropriate
	// (CAPE Guidelines line 425; mirrors override taxonomy
	// goals_of_care_aligned ACOP extension).
	OutcomeGoalsOfCareAligned = "goals_of_care_aligned"
)

// IsRetryEligible reports whether enough time has passed since the failure
// for a re-attempt to be considered. The boundary semantic is non-strict
// (RetryEligibleDate <= now ⇒ eligible): at the exact instant
// RetryEligibleDate == now the record is NO LONGER an active veto and
// retry IS eligible.
//
// Rationale: CAPE Guidelines line 645 specifies the active-veto condition
// as `RetryEligibleDate.After(time.Now())`. time.Time.After is strict —
// at Equal it returns false — so Equal means "not in the future" ⇒ no
// veto ⇒ retry eligible. Symmetrically, IsVetoActive below uses the
// same strict After check. Equivalence:
//   IsRetryEligible(now) == !IsVetoActive([...], type, now)
// for a single matching record.
func (r FailedInterventionRecord) IsRetryEligible(now time.Time) bool {
	return !r.RetryEligibleDate.After(now)
}

// IsVetoActive reports whether any record in `records` blocks the named
// intervention type at `now`. A record blocks when:
//   - InterventionType matches `interventionType` case-insensitively
//   - The empty-string interventionType never matches any record (defensive
//     against caller bugs that pass an unclassified rule ID through)
//   - RetryEligibleDate is strictly after `now` (i.e., the retry window has
//     not yet elapsed)
//
// Multi-record semantics: OR — any single blocking record makes the
// veto active. Returns true on the first match for efficiency.
//
// Spec: CAPE Guidelines lines 642–650 (IsVetoActive function, line-for-line
// equivalent semantics).
func IsVetoActive(records []FailedInterventionRecord, interventionType string, now time.Time) bool {
	if interventionType == "" {
		return false
	}
	target := strings.ToLower(interventionType)
	for _, r := range records {
		if r.InterventionType == "" {
			continue
		}
		if !strings.EqualFold(r.InterventionType, target) {
			continue
		}
		if r.RetryEligibleDate.After(now) {
			return true
		}
	}
	return false
}
