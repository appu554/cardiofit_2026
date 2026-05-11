// FailedInterventionRecord substrate shape — Failed Intervention History
// per S2 v1.0 Part 8 + CAPE Guidelines v1.1 §4.3 (lines 627–660).
//
// SOURCE OF TRUTH: shared/v2_substrate/failed_interventions/types.go
// (FailedInterventionRecord). The pin test in this package enforces
// structural parity; update the local copy + this SOURCE OF TRUTH
// comment in lock-step when the canonical type drifts.
//
// KNOWN GAP (Step 4 Task B):
//
//	The canonical Store writes ResidentID=uuid.Nil today because the
//	OverrideReason payload lacks a resident-id field. The S2 panel
//	surfaces this gap explicitly via a "FIR retrieval incomplete —
//	kb-32 RecommendationID→ResidentID resolver pending" badge until
//	kb-32 ships the JOIN-resolver extension. The s2-aggregator does
//	NOT pretend FIR retrieval is wired in Phase 1.
package substrate_types

import (
	"time"

	"github.com/google/uuid"
)

// FailedInterventionRecord mirrors the canonical
// failed_interventions.FailedInterventionRecord. Field names and order
// match the canonical type 1:1; the pin test asserts this.
//
// Phase 1 commitment: SubstrateRefs are attached at the aggregation
// layer (failed_history.go) — the substrate-level shape is intentionally
// free of rendering concerns.
type FailedInterventionRecord struct {
	ResidentID        uuid.UUID
	InterventionType  string
	AttemptDate       time.Time
	Outcome           string
	DocumentedReason  string
	RetryEligibleDate time.Time
	DocumentedBy      uuid.UUID
}

// Outcome constants mirror the canonical failed_interventions package's
// closed vocabulary. Values match byte-for-byte.
const (
	// OutcomeReversedDueToBPSDRecurrence — intervention reversed because
	// BPSD returned after the change.
	OutcomeReversedDueToBPSDRecurrence = "reversed_due_to_BPSD_recurrence"

	// OutcomeReversedDueToFamilyRequest — reversed at family/decision-maker
	// request.
	OutcomeReversedDueToFamilyRequest = "reversed_due_to_family_request"

	// OutcomeReversedDueToClinicalDecline — reversed because the resident's
	// clinical status declined after the change.
	OutcomeReversedDueToClinicalDecline = "reversed_due_to_clinical_decline"

	// OutcomeReversedDueToFrailty — reversed because the frailty profile
	// made continuation inadvisable.
	OutcomeReversedDueToFrailty = "reversed_due_to_frailty"

	// OutcomeGoalsOfCareAligned — reversed because goals-of-care alignment
	// indicated the original regimen was clinically appropriate.
	OutcomeGoalsOfCareAligned = "goals_of_care_aligned"
)
