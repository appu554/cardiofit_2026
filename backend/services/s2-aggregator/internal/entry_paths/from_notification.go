package entry_paths

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FromNotification builds EntryPathMetadata for v1.0 Part 3.3 entry: an
// in-app or email notification dispatched the pharmacist to S2. The
// notification context surfaces in the notification context band per
// v1.0 Part 4.1 Component 3.
func FromNotification(
	ctx context.Context,
	pharmacistID, residentID uuid.UUID,
	notif NotificationContext,
) (EntryPathMetadata, error) {
	if pharmacistID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("pharmacist_id is zero"))
	}
	if residentID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("resident_id is zero"))
	}
	if notif.NotificationID == uuid.Nil {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("notification_id is zero"))
	}
	if strings.TrimSpace(notif.ReasonText) == "" {
		return EntryPathMetadata{}, errors.Join(ErrInvalidEntryPathInput, errors.New("notification reason_text is empty"))
	}
	if notif.DispatchedAt.IsZero() {
		notif.DispatchedAt = time.Now().UTC()
	}
	return EntryPathMetadata{
		TriggeredAt:  time.Now().UTC(),
		PharmacistID: pharmacistID,
		ResidentID:   residentID,
		Path:         EntryPathNotification,
		Context:      notif,
	}, nil
}
