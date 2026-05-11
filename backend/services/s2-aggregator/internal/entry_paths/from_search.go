package entry_paths

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FromSearch builds EntryPathMetadata for v1.0 Part 3.2 entry: pharmacist
// searches for a resident by name, room, or other identifier. No CAPE
// context band renders for this entry path — the entry is not triage-driven.
func FromSearch(
	ctx context.Context,
	pharmacistID, residentID uuid.UUID,
	query string,
) (EntryPathMetadata, error) {
	if pharmacistID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("pharmacist_id is zero"))
	}
	if residentID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("resident_id is zero"))
	}
	if strings.TrimSpace(query) == "" {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("search query is empty"))
	}
	now := time.Now().UTC()
	return EntryPathMetadata{
		TriggeredAt:  now,
		PharmacistID: pharmacistID,
		ResidentID:   residentID,
		Path:         EntryPathSearch,
		Context: SearchContext{
			Query:     query,
			MatchedAt: now,
		},
	}, nil
}
