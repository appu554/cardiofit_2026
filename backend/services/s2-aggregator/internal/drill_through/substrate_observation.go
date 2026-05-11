// Package drill_through implements the v1.0 Part 10 drill-through pattern:
// every claim in S2 is one click from the underlying substrate observation,
// trajectory history, or negative-evidence search record. The handlers
// here are read-only by design.
package drill_through

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// SubstrateObservation is the rich view returned by GetSubstrateObservation
// per v1.0 Part 10.1. The raw observation row plus substrate confidence
// plus a back-trail to the originating claim are bundled so the rendering
// layer can show the full v1.0 Part 10.1 layout (value, observed-at,
// source, specimen collection, reference range, confidence, linked
// clinical context).
//
// Drill-through depth (v1.0 Part 10.2) is supported by leaving ClinicalContext
// as a free-form slice — Task 8 will wire pathology-lab audit log /
// specimen-collection-record / pharmacist-note lookups.
type SubstrateObservation struct {
	Ref                aggregation.SubstrateRef
	Observation        substrate_types.Observation
	SubstrateConfidence string // "high" | "moderate" | "low"
	ClinicalContext    []string // free-form context lines per v1.0 Part 10.1
	ClaimBackTrail     []aggregation.SubstrateRef // the "one-click-back-to-claim" trail per v1.0 Part 10.1
}

// ObservationFetcher is the minimal port for fetching an observation by
// ID. Production wires this to a kb-20 reader; tests use the in-memory
// fake from aggregation.NewInMemorySubstrateClient.
type ObservationFetcher interface {
	GetObservationByID(ctx context.Context, id uuid.UUID) (substrate_types.Observation, error)
}

// GetSubstrateObservation returns the substrate observation behind a
// SubstrateRef. backTrail is the chain of claims that led to this
// drill-through (typically a single claim, but nested drill-throughs
// can accumulate).
//
// The function is intentionally narrow: it does NOT load pathology lab
// audit logs or specimen records (those are v1.0 Part 10.2 deeper
// drill-through and are TODO at Task 8 production wiring).
func GetSubstrateObservation(
	ctx context.Context,
	fetcher ObservationFetcher,
	ref aggregation.SubstrateRef,
	backTrail []aggregation.SubstrateRef,
) (SubstrateObservation, error) {
	if fetcher == nil {
		return SubstrateObservation{}, fmt.Errorf("GetSubstrateObservation: nil fetcher")
	}
	if ref.ID == uuid.Nil {
		return SubstrateObservation{}, fmt.Errorf("GetSubstrateObservation: empty ref")
	}

	obs, err := fetcher.GetObservationByID(ctx, ref.ID)
	if err != nil {
		return SubstrateObservation{}, fmt.Errorf("GetSubstrateObservation(%s): %w", ref.ID, err)
	}

	conf := obs.Confidence
	if conf == "" {
		conf = "high" // default per v1.0 Part 10.3 — explicit downgrade required
	}

	return SubstrateObservation{
		Ref:                 ref,
		Observation:         obs,
		SubstrateConfidence: conf,
		ClinicalContext: []string{
			// TODO(Task 8): wire pathology lab integration audit log,
			// specimen collection record, and concurrent-observation
			// context per v1.0 Part 10.1 lines 810–815.
		},
		ClaimBackTrail: append([]aggregation.SubstrateRef{}, backTrail...),
	}, nil
}

// AssessedAtRange is a convenience type for callers building the
// substrate observation view; matches v1.0 Part 10.1 line "Observed:"
// + "Specimen collection:" pairing.
type AssessedAtRange struct {
	Observed         time.Time
	SpecimenCollected *time.Time
}
