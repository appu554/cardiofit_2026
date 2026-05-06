package reconciliation

import (
	"time"

	"github.com/google/uuid"
)

// ACOPDecision is the decision an Approving Clinical Officer Practitioner
// (or equivalent role) may record on a single reconciliation line.
type ACOPDecision string

const (
	ACOPAccept ACOPDecision = "accept"
	ACOPModify ACOPDecision = "modify"
	ACOPReject ACOPDecision = "reject"
	ACOPDefer  ACOPDecision = "defer"
)

// IsValidACOPDecision reports whether s is recognised.
func IsValidACOPDecision(s string) bool {
	switch ACOPDecision(s) {
	case ACOPAccept, ACOPModify, ACOPReject, ACOPDefer:
		return true
	}
	return false
}

// WorklistStatus enumerates the lifecycle of a reconciliation worklist.
type WorklistStatus string

const (
	WorklistPending    WorklistStatus = "pending"
	WorklistInProgress WorklistStatus = "in_progress"
	WorklistCompleted  WorklistStatus = "completed"
	WorklistAbandoned  WorklistStatus = "abandoned"
)

// IsValidWorklistStatus reports whether s is recognised.
func IsValidWorklistStatus(s string) bool {
	switch WorklistStatus(s) {
	case WorklistPending, WorklistInProgress, WorklistCompleted, WorklistAbandoned:
		return true
	}
	return false
}

// DefaultWorklistDueWindow is the default time after a hospital_discharge
// Event that the reconciliation worklist falls due. Layer 2 §3.2 step 6
// says "due within 24h"; the value is exposed as a const for callers
// that want to override per-facility policy.
const DefaultWorklistDueWindow = 24 * time.Hour

// WorklistInputs is the pure-engine output: enough information for the
// storage layer to write a reconciliation_worklist row + N
// reconciliation_decision rows in one transaction. The storage layer
// owns the UUIDs, timestamps, and FK ids — this struct carries only
// the per-row payload.
type WorklistInputs struct {
	DischargeDocumentRef uuid.UUID
	ResidentRef          uuid.UUID
	AssignedRoleRef      *uuid.UUID
	FacilityID           *uuid.UUID
	DueAt                time.Time
	Decisions            []DecisionInputs
}

// DecisionInputs is the per-row decision payload the storage layer
// inserts into reconciliation_decisions. ACOPDecision is intentionally
// empty — decisions are made by the ACOP via PATCH after the worklist
// is created.
type DecisionInputs struct {
	DiffEntry            DiffEntry
	IntentClass          IntentClass
}

// BuildWorklistInputs converts a diff slice + per-line discharge text
// supplier into the row-shape payload the storage layer needs.
//
// Only non-unchanged entries become decision rows — Unchanged lines are
// substrate-truth already and need no ACOP attention (Layer 2 §3.2
// step 6: "one decision row per non-unchanged diff").
//
// dischargeTextFor receives the diff entry and returns the per-line
// free-text the classifier should scan. Callers typically pass a
// closure over the parsed discharge document so the per-line text is
// resolved without smuggling the parsed document into this pure
// package.
func BuildWorklistInputs(
	dischargeDocumentRef, residentRef uuid.UUID,
	assignedRoleRef, facilityID *uuid.UUID,
	dischargeTime time.Time,
	dueWindow time.Duration,
	diffs []DiffEntry,
	dischargeTextFor func(DiffEntry) string,
) WorklistInputs {
	if dueWindow <= 0 {
		dueWindow = DefaultWorklistDueWindow
	}
	out := WorklistInputs{
		DischargeDocumentRef: dischargeDocumentRef,
		ResidentRef:          residentRef,
		AssignedRoleRef:      assignedRoleRef,
		FacilityID:           facilityID,
		DueAt:                dischargeTime.Add(dueWindow),
	}
	for _, d := range diffs {
		if d.Class == DiffUnchanged {
			continue
		}
		text := ""
		if dischargeTextFor != nil {
			text = dischargeTextFor(d)
		}
		// Convenience fallback so callers that don't pass a resolver
		// still get a reasonable classification from the line itself.
		if text == "" && d.DischargeLineMedicine != nil {
			text = ComposeDischargeText(d.DischargeLineMedicine)
		}
		intent := ClassifyIntent(d, text)
		out.Decisions = append(out.Decisions, DecisionInputs{
			DiffEntry:   d,
			IntentClass: intent,
		})
	}
	return out
}
