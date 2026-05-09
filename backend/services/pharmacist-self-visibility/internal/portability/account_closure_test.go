package portability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeCloser is a configurable Closer for account closure tests.
type fakeCloser struct {
	exportRef   string
	anonymised  int
	adCount     int
	exportErr   error
	anonymiseErr error
	adCountErr  error
}

func (f *fakeCloser) ExportPharmacistData(_ context.Context, _ uuid.UUID) (string, error) {
	return f.exportRef, f.exportErr
}

func (f *fakeCloser) AnonymiseAggregateContributions(_ context.Context, _ uuid.UUID) (int, error) {
	return f.anonymised, f.anonymiseErr
}

func (f *fakeCloser) CountADRecords(_ context.Context, _ uuid.UUID) (int, error) {
	return f.adCount, f.adCountErr
}

// TestClose_HappyPath verifies that Close populates all AccountClosure fields
// correctly and that RetentionUntil is ClosedAt + retentionYears.
func TestClose_HappyPath(t *testing.T) {
	pharmacistID := uuid.New()
	fc := &fakeCloser{
		exportRef:  "s3://exports/pharmacist-archive.zip",
		anonymised: 5,
		adCount:    12,
	}
	h := NewCloseHandler(fc)
	before := time.Now().UTC()

	closure, err := h.Close(context.Background(), pharmacistID, 7)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if closure.ID == (uuid.UUID{}) {
		t.Error("ID must be set")
	}
	if closure.PharmacistID != pharmacistID {
		t.Errorf("PharmacistID = %v, want %v", closure.PharmacistID, pharmacistID)
	}
	if closure.ExportRef != "s3://exports/pharmacist-archive.zip" {
		t.Errorf("ExportRef = %q", closure.ExportRef)
	}
	if closure.AnonymisedAggregateContributions != 5 {
		t.Errorf("AnonymisedAggregateContributions = %d, want 5", closure.AnonymisedAggregateContributions)
	}
	if closure.RetainedADRecords != 12 {
		t.Errorf("RetainedADRecords = %d, want 12", closure.RetainedADRecords)
	}
	if closure.ClosedAt.Before(before) {
		t.Errorf("ClosedAt %v is before test start %v", closure.ClosedAt, before)
	}

	// RetentionUntil must be exactly ClosedAt + 7 years (within tolerance).
	expectedRetention := closure.ClosedAt.AddDate(7, 0, 0)
	diff := closure.RetentionUntil.Sub(expectedRetention)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("RetentionUntil = %v, want ~%v (ClosedAt+7y)", closure.RetentionUntil, expectedRetention)
	}
}

// TestClose_PropagatesExportError verifies that an error from ExportPharmacistData
// is returned by Close without further processing.
func TestClose_PropagatesExportError(t *testing.T) {
	exportErr := errors.New("export: storage unavailable")
	fc := &fakeCloser{exportErr: exportErr}
	h := NewCloseHandler(fc)

	_, err := h.Close(context.Background(), uuid.New(), 7)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, exportErr) {
		t.Errorf("got %v, want wrapped %v", err, exportErr)
	}
}

// TestClose_PropagatesAnonymiseError verifies that an error from
// AnonymiseAggregateContributions is returned by Close.
func TestClose_PropagatesAnonymiseError(t *testing.T) {
	anonymiseErr := errors.New("anonymise: db timeout")
	fc := &fakeCloser{
		exportRef:    "s3://ok",
		anonymiseErr: anonymiseErr,
	}
	h := NewCloseHandler(fc)

	_, err := h.Close(context.Background(), uuid.New(), 7)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, anonymiseErr) {
		t.Errorf("got %v, want wrapped %v", err, anonymiseErr)
	}
}

// TestClose_ContextCancellation verifies that a cancelled context is detected
// at the top of Close before any Closer calls are made.
func TestClose_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	h := NewCloseHandler(&fakeCloser{exportRef: "s3://ok"})
	_, err := h.Close(ctx, uuid.New(), 7)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
