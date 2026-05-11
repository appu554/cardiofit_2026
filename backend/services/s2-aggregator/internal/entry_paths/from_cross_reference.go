package entry_paths

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"time"
)

// FromCrossReference builds EntryPathMetadata for v1.0 Part 3.4 entry:
// the pharmacist navigated to this resident from another resident's
// chart (e.g., a medication-class cross-reference, a family member, or
// facility cohort review). Triggers comparative mode in later rendering.
func FromCrossReference(
	ctx context.Context,
	pharmacistID, residentID uuid.UUID,
	xref CrossReferenceContext,
) (EntryPathMetadata, error) {
	if pharmacistID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("pharmacist_id is zero"))
	}
	if residentID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("resident_id is zero"))
	}
	if xref.OriginResidentID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("origin_resident_id is zero"))
	}
	if xref.OriginResidentID == residentID {
		// Cross-referencing a resident to themselves is a nonsense state
		// — almost certainly a caller bug.
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("origin_resident_id equals target resident_id"))
	}
	reason := strings.TrimSpace(xref.ReasonCode)
	if reason == "" {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("reason_code is empty"))
	}
	if _, ok := ValidCrossReferenceReasons[reason]; !ok {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("reason_code not in canonical vocabulary: "+reason))
	}
	xref.ReasonCode = reason
	return EntryPathMetadata{
		TriggeredAt:  time.Now().UTC(),
		PharmacistID: pharmacistID,
		ResidentID:   residentID,
		Path:         EntryPathCrossReference,
		Context:      xref,
	}, nil
}
