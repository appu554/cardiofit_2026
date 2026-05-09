package portability

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// fakeCarrier is a no-op Carrier used in transition tests.
type fakeCarrier struct{}

func (f *fakeCarrier) MovePOA(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error      { return nil }
func (f *fakeCarrier) MovePortfolio(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error { return nil }
func (f *fakeCarrier) MoveOwnPFA(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error    { return nil }

// TestTransition_PreservesPOAandPDPandOwnPFA — verbatim from plan spec.
// Per Guidelines §10: reflective entries (POA), portfolio (PDP), and own PFA
// must travel with the pharmacist; active recommendations stay with prior deployment.
func TestTransition_PreservesPOAandPDPandOwnPFA(t *testing.T) {
	pharm := uuid.New()
	priorEmp := uuid.New()
	newEmp := uuid.New()
	carrier := &fakeCarrier{}
	h := NewHandler(carrier)

	plan, err := h.Initiate(context.Background(), pharm, priorEmp, &newEmp)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if !plan.PreservesReflectiveEntries || !plan.PreservesPortfolio || !plan.PreservesOwnPFA {
		t.Errorf("plan must preserve POA + portfolio + own PFA; got %+v", plan)
	}
	if plan.PreservesActiveRecommendations {
		t.Errorf("active recommendations must stay with prior employer")
	}
}

// TestTransition_FreeTierReversionWhenNoNewEmployer — verbatim from plan spec.
// When no new employer is given the account must revert to the free tier.
func TestTransition_FreeTierReversionWhenNoNewEmployer(t *testing.T) {
	h := NewHandler(&fakeCarrier{})
	plan, _ := h.Initiate(context.Background(), uuid.New(), uuid.New(), nil)
	if !plan.RevertsToFreeTier {
		t.Errorf("expected free-tier reversion when new employer is nil")
	}
}

// errCarrier is a Carrier that returns a configurable error for the first
// carrier method called.
type errCarrier struct {
	failOn string // "poa" | "portfolio" | "pfa"
}

func (e *errCarrier) MovePOA(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	if e.failOn == "poa" {
		return errors.New("poa: move failed")
	}
	return nil
}
func (e *errCarrier) MovePortfolio(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	if e.failOn == "portfolio" {
		return errors.New("portfolio: move failed")
	}
	return nil
}
func (e *errCarrier) MoveOwnPFA(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	if e.failOn == "pfa" {
		return errors.New("pfa: move failed")
	}
	return nil
}

// TestTransition_PropagatesCarrierError — augmentation.
// If any of MovePOA/MovePortfolio/MoveOwnPFA returns an error, Initiate must
// surface it rather than silently continuing.
func TestTransition_PropagatesCarrierError(t *testing.T) {
	cases := []struct {
		name   string
		failOn string
	}{
		{"MovePOA fails", "poa"},
		{"MovePortfolio fails", "portfolio"},
		{"MoveOwnPFA fails", "pfa"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(&errCarrier{failOn: tc.failOn})
			_, err := h.Initiate(context.Background(), uuid.New(), uuid.New(), nil)
			if err == nil {
				t.Errorf("expected error when %s, got nil", tc.name)
			}
		})
	}
}

// TestTransition_ContextCancellation — augmentation.
// A cancelled context must be detected immediately at the top of Initiate,
// before any carrier calls.
func TestTransition_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := NewHandler(&fakeCarrier{})
	_, err := h.Initiate(ctx, uuid.New(), uuid.New(), nil)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
