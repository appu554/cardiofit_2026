package portability

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// AccountClosure records a pharmacist's full account closure event.
// Per Self-Visibility Guidelines §10.5: pharmacist-controlled data is exported
// to the pharmacist, AD-class records are retained for regulatory compliance,
// and aggregate contributions are anonymised in place.
//
// VisibilityClass: pharmacist-controlled — the closure record belongs to the
// pharmacist; AD-class retained records are held for regulatory purposes only.
type AccountClosure struct {
	ID                               uuid.UUID
	PharmacistID                     uuid.UUID
	ExportRef                        string    // pointer to the exported archive (e.g. S3 URI / file path)
	RetainedADRecords                int       // count of AD-class records retained for regulatory compliance
	AnonymisedAggregateContributions int       // count of anonymisation actions taken on aggregate data
	ClosedAt                         time.Time
	RetentionUntil                   time.Time // regulatory retention horizon for the AD records
}

// Closer executes the three operations required for account closure.
type Closer interface {
	// ExportPharmacistData bundles all pharmacist-controlled data and returns a
	// reference (e.g. an S3 URI or file path) to the exported archive.
	ExportPharmacistData(ctx context.Context, pharmacistID uuid.UUID) (string, error)

	// AnonymiseAggregateContributions removes the pharmacist's identity from any
	// aggregate data they contributed to. Returns the count of actions taken.
	AnonymiseAggregateContributions(ctx context.Context, pharmacistID uuid.UUID) (int, error)

	// CountADRecords returns the number of audit-defensible records that must be
	// retained per regulatory requirements, even after account closure.
	CountADRecords(ctx context.Context, pharmacistID uuid.UUID) (int, error)
}

// ErrCloseFailed is returned when the account closure operation cannot complete.
var ErrCloseFailed = errors.New("portability: account closure failed")

// CloseHandler orchestrates account closures per Guidelines §10.5.
type CloseHandler struct{ c Closer }

// NewCloseHandler returns a CloseHandler backed by the given Closer.
func NewCloseHandler(c Closer) *CloseHandler { return &CloseHandler{c: c} }

// Close performs the full account closure sequence for the given pharmacist:
//  1. Export all pharmacist-controlled data.
//  2. Anonymise aggregate contributions.
//  3. Record the count of AD-class records retained under regulatory requirements.
//
// retentionYears sets the regulatory retention horizon for AD-class records.
func (h *CloseHandler) Close(ctx context.Context, pharmacistID uuid.UUID, retentionYears int) (AccountClosure, error) {
	if err := ctx.Err(); err != nil {
		return AccountClosure{}, err
	}

	exportRef, err := h.c.ExportPharmacistData(ctx, pharmacistID)
	if err != nil {
		return AccountClosure{}, err
	}

	anonymised, err := h.c.AnonymiseAggregateContributions(ctx, pharmacistID)
	if err != nil {
		return AccountClosure{}, err
	}

	adCount, err := h.c.CountADRecords(ctx, pharmacistID)
	if err != nil {
		return AccountClosure{}, err
	}

	now := time.Now().UTC()
	return AccountClosure{
		ID:                               uuid.New(),
		PharmacistID:                     pharmacistID,
		ExportRef:                        exportRef,
		RetainedADRecords:                adCount,
		AnonymisedAggregateContributions: anonymised,
		ClosedAt:                         now,
		RetentionUntil:                   now.AddDate(retentionYears, 0, 0),
	}, nil
}
