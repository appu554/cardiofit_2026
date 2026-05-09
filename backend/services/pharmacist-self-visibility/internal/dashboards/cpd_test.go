package dashboards

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// fakeCPDSrc — test double for CPDSource
// ---------------------------------------------------------------------------

type fakeCPDSrc struct {
	autoTagged []CPDActivity
	err        error
}

func (f *fakeCPDSrc) Activities(_ context.Context, _ uuid.UUID) ([]CPDActivity, error) {
	return f.autoTagged, f.err
}

// ---------------------------------------------------------------------------
// Plan verbatim test (Task 8)
// ---------------------------------------------------------------------------

func TestCPD_AutoTagPendingConfirmation(t *testing.T) {
	pharm := uuid.New()
	src := &fakeCPDSrc{autoTagged: []CPDActivity{
		{ID: uuid.New(), AHPRACategory: "clinical_review", Hours: 1.5, Status: "pending_confirmation"},
		{ID: uuid.New(), AHPRACategory: "education", Hours: 2.0, Status: "confirmed"},
	}}
	d := NewCPD(src)
	view, _ := d.For(context.Background(), pharm)
	if view.PendingConfirmation != 1 || view.ConfirmedHours["education"] != 2.0 {
		t.Errorf("got pending=%d confirmedEducation=%v", view.PendingConfirmation, view.ConfirmedHours["education"])
	}
}

// ---------------------------------------------------------------------------
// Augmentation 2: IsValidCPDStatus helper
// ---------------------------------------------------------------------------

func TestIsValidCPDStatus(t *testing.T) {
	valid := []string{"pending_confirmation", "confirmed", "rejected"}
	for _, s := range valid {
		if !IsValidCPDStatus(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	invalid := []string{"", "unknown", "CONFIRMED", "Pending_Confirmation"}
	for _, s := range invalid {
		if IsValidCPDStatus(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

// ---------------------------------------------------------------------------
// Augmentation 3: rejected status excluded from both buckets
// ---------------------------------------------------------------------------

func TestCPD_RejectedStatusExcludedFromBoth(t *testing.T) {
	src := &fakeCPDSrc{autoTagged: []CPDActivity{
		{ID: uuid.New(), AHPRACategory: "audit_feedback", Hours: 3.0, Status: "rejected"},
	}}
	d := NewCPD(src)
	view, err := d.For(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.PendingConfirmation != 0 {
		t.Errorf("rejected activity must not increment PendingConfirmation; got %d", view.PendingConfirmation)
	}
	if total := view.ConfirmedHours["audit_feedback"]; total != 0 {
		t.Errorf("rejected activity must not contribute to ConfirmedHours; got %v", total)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 4: multiple categories aggregate correctly
// ---------------------------------------------------------------------------

func TestCPD_MultipleCategoriesAggregateCorrectly(t *testing.T) {
	src := &fakeCPDSrc{autoTagged: []CPDActivity{
		{ID: uuid.New(), AHPRACategory: "clinical_review", Hours: 1.0, Status: "confirmed"},
		{ID: uuid.New(), AHPRACategory: "clinical_review", Hours: 0.5, Status: "confirmed"},
		{ID: uuid.New(), AHPRACategory: "education", Hours: 2.0, Status: "confirmed"},
		{ID: uuid.New(), AHPRACategory: "audit_feedback", Hours: 1.5, Status: "confirmed"},
		// A pending and a rejected that must not leak into any category total
		{ID: uuid.New(), AHPRACategory: "education", Hours: 99.0, Status: "pending_confirmation"},
		{ID: uuid.New(), AHPRACategory: "clinical_review", Hours: 99.0, Status: "rejected"},
	}}
	d := NewCPD(src)
	view, err := d.For(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := view.ConfirmedHours["clinical_review"]; got != 1.5 {
		t.Errorf("clinical_review: want 1.5, got %v", got)
	}
	if got := view.ConfirmedHours["education"]; got != 2.0 {
		t.Errorf("education: want 2.0, got %v", got)
	}
	if got := view.ConfirmedHours["audit_feedback"]; got != 1.5 {
		t.Errorf("audit_feedback: want 1.5, got %v", got)
	}
	// Pending count: exactly 1
	if view.PendingConfirmation != 1 {
		t.Errorf("PendingConfirmation: want 1, got %d", view.PendingConfirmation)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 5a: source error propagates
// ---------------------------------------------------------------------------

func TestCPD_PropagatesSourceError(t *testing.T) {
	sentinel := errors.New("source unavailable")
	src := &fakeCPDSrc{err: sentinel}
	d := NewCPD(src)
	_, err := d.For(context.Background(), uuid.New())
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error; got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 5b: context cancellation
// ---------------------------------------------------------------------------

type cancelCPDSrc struct{}

func (c *cancelCPDSrc) Activities(ctx context.Context, _ uuid.UUID) ([]CPDActivity, error) {
	return nil, ctx.Err()
}

func TestCPD_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	d := NewCPD(&cancelCPDSrc{})
	_, err := d.For(ctx, uuid.New())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled; got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 6: empty activities → non-nil ConfirmedHours map, zero pending
// ---------------------------------------------------------------------------

func TestCPD_EmptyReturnsZeroView(t *testing.T) {
	src := &fakeCPDSrc{autoTagged: []CPDActivity{}}
	d := NewCPD(src)
	view, err := d.For(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ConfirmedHours == nil {
		t.Error("ConfirmedHours must be non-nil even when no activities exist (distinguishable from 'data unavailable')")
	}
	if view.PendingConfirmation != 0 {
		t.Errorf("PendingConfirmation: want 0, got %d", view.PendingConfirmation)
	}
	if len(view.ConfirmedHours) != 0 {
		t.Errorf("ConfirmedHours should be empty map; got %v", view.ConfirmedHours)
	}
}
