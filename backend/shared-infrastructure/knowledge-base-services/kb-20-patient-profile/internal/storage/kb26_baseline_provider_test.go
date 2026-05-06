package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/delta"
)

func TestInMemoryBaselineProvider_FetchExistingBaseline(t *testing.T) {
	p := NewInMemoryBaselineProvider()
	rid := uuid.New()
	p.Seed(rid, "8480-6", delta.Baseline{
		BaselineValue: 130.0, StdDev: 8.0, SampleSize: 30, ComputedAt: time.Now(),
	})
	bl, err := p.FetchBaseline(context.Background(), rid, "8480-6")
	if err != nil {
		t.Fatalf("FetchBaseline: %v", err)
	}
	if bl.BaselineValue != 130.0 || bl.StdDev != 8.0 {
		t.Errorf("baseline drift: got %+v", bl)
	}
}

func TestInMemoryBaselineProvider_MissingReturnsErrNoBaseline(t *testing.T) {
	p := NewInMemoryBaselineProvider()
	_, err := p.FetchBaseline(context.Background(), uuid.New(), "8480-6")
	if !errors.Is(err, delta.ErrNoBaseline) {
		t.Errorf("expected ErrNoBaseline, got %v", err)
	}
}
