package drill_through

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

type fakeFetcher struct {
	rows map[uuid.UUID]substrate_types.Observation
}

func (f *fakeFetcher) GetObservationByID(_ context.Context, id uuid.UUID) (substrate_types.Observation, error) {
	o, ok := f.rows[id]
	if !ok {
		return substrate_types.Observation{}, errors.New("not found")
	}
	return o, nil
}

func TestGetSubstrateObservation_HappyPath(t *testing.T) {
	id := uuid.New()
	o := substrate_types.Observation{
		ID:         id,
		ResidentID: uuid.New(),
		Parameter:  "egfr",
		Value:      41,
		Unit:       "mL/min/1.73m²",
		ObservedAt: time.Date(2026, 4, 15, 14, 23, 0, 0, time.UTC),
		Source:     "pathology_lab",
		Confidence: "high",
	}
	f := &fakeFetcher{rows: map[uuid.UUID]substrate_types.Observation{id: o}}
	ref := aggregation.SubstrateRef{Source: "kb-20", ID: id, Description: "egfr=41"}
	backTrail := []aggregation.SubstrateRef{{Source: "trajectory", ID: uuid.New(), Description: "eGFR claim"}}

	got, err := GetSubstrateObservation(context.Background(), f, ref, backTrail)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Observation.ID != id {
		t.Errorf("wrong observation id")
	}
	if got.SubstrateConfidence != "high" {
		t.Errorf("confidence: %q", got.SubstrateConfidence)
	}
	if len(got.ClaimBackTrail) != 1 {
		t.Errorf("back trail not preserved")
	}
}

func TestGetSubstrateObservation_NilFetcher(t *testing.T) {
	_, err := GetSubstrateObservation(context.Background(), nil, aggregation.SubstrateRef{ID: uuid.New()}, nil)
	if err == nil {
		t.Fatal("expected error on nil fetcher")
	}
}

func TestGetSubstrateObservation_EmptyRef(t *testing.T) {
	_, err := GetSubstrateObservation(context.Background(), &fakeFetcher{}, aggregation.SubstrateRef{}, nil)
	if err == nil {
		t.Fatal("expected error on empty ref")
	}
}

func TestGetSubstrateObservation_DefaultConfidence(t *testing.T) {
	id := uuid.New()
	f := &fakeFetcher{rows: map[uuid.UUID]substrate_types.Observation{
		id: {ID: id, Confidence: ""},
	}}
	got, err := GetSubstrateObservation(context.Background(), f, aggregation.SubstrateRef{ID: id}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.SubstrateConfidence != "high" {
		t.Errorf("default confidence should be 'high'; got %q", got.SubstrateConfidence)
	}
}
