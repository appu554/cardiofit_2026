package entry_paths

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidEntryPathInput is returned by entry-path handlers when their
// inputs do not satisfy the path's preconditions (zero UUIDs, missing
// signals, empty queries, etc.).
var ErrInvalidEntryPathInput = errors.New("invalid entry-path input")

// FromWorklist builds EntryPathMetadata for the highest-frequency entry
// path per v1.0 Part 3.1: pharmacist arrives from S1 CAPE worklist.
//
// The WorklistContext carries the primary signals that drove kb-33's
// prioritisation so the CAPE context band in v1.0 Part 4.1 Component 2
// can render them without re-derivation (v1.0 Part 3.1 lines 254–262 +
// Addendum Part 4.8 carry-through commitment).
//
// TODO(kb-33 Step 5 integration): once kb-33-triage-engine ships, the
// CAPE context shape will gain dimension score breakdowns, instability
// chronology references, and substrate IDs. This handler will then
// validate those richer fields in addition to PrimarySignals.
func FromWorklist(
	ctx context.Context,
	pharmacistID, residentID uuid.UUID,
	capeCtx WorklistContext,
) (EntryPathMetadata, error) {
	if pharmacistID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("pharmacist_id is zero"))
	}
	if residentID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("resident_id is zero"))
	}
	if len(capeCtx.PrimarySignals) == 0 {
		// A worklist entry without signals is nonsense — the whole point
		// of v1.0 §3.1 is to carry signals through. Reject early.
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("worklist entry has no primary signals"))
	}
	if capeCtx.TriagedAt.IsZero() {
		capeCtx.TriagedAt = time.Now().UTC()
	}
	return EntryPathMetadata{
		TriggeredAt:  time.Now().UTC(),
		PharmacistID: pharmacistID,
		ResidentID:   residentID,
		Path:         EntryPathWorklist,
		Context:      capeCtx,
	}, nil
}
