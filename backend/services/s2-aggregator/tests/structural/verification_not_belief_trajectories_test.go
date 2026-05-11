// Package structural holds cross-cutting tests that enforce the v1.0
// Part 17 critical invariants. These tests live outside any internal/
// package so they exercise the public interfaces only.
package structural

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TestEveryTrajectoryHasSubstrateRef enforces the v1.0 Part 17 critical
// test (lines 1298–1313) at the trajectory layer: every Trajectory in a
// representative built []Trajectory must carry at least one SubstrateRef.
//
// This is the verification-not-belief invariant structurally. Failing
// this test fails Principle 2 of v1.0 (every claim verifiable through
// the substrate it references).
func TestEveryTrajectoryHasSubstrateRef(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	// Seed a representative substrate: full eGFR series, sparse weight,
	// no DBI/ACB/CFS data, and PRN administrations across all three
	// classes. This produces a Trajectory list covering all rendering
	// paths (full data, sparse, no observation, PRN).
	client := aggregation.NewInMemorySubstrateClient().WithObservations(
		mkObs(rid, "egfr", 52, asOf.AddDate(-1, 0, 0)),
		mkObs(rid, "egfr", 45, asOf.AddDate(0, -6, 0)),
		mkObs(rid, "egfr", 40, asOf.AddDate(0, -1, 0)),
		mkObs(rid, "weight", 70, asOf.AddDate(0, -2, 0)),
	).WithAdministrations(
		substrate_types.PRNAdministration{ResidentID: rid, Class: substrate_types.PRNBenzodiazepine, AdministeredAt: asOf.AddDate(0, 0, -10)},
		substrate_types.PRNAdministration{ResidentID: rid, Class: substrate_types.PRNAntipsychotic, AdministeredAt: asOf.AddDate(0, 0, -5)},
	)

	trs, err := aggregation.BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories error: %v", err)
	}
	if len(trs) == 0 {
		t.Fatal("expected non-empty trajectory set")
	}

	for _, tr := range trs {
		if len(tr.SubstrateRefs) == 0 {
			t.Errorf(
				"trajectory %q has no SubstrateRef — violates verification-not-belief discipline (v1.0 Principle 2; Part 17 critical test)",
				tr.Parameter,
			)
		}
	}
}

func mkObs(rid uuid.UUID, param string, v float64, at time.Time) substrate_types.Observation {
	return substrate_types.Observation{
		ID:         uuid.New(),
		ResidentID: rid,
		Parameter:  param,
		Value:      v,
		Unit:       "test",
		ObservedAt: at,
		Source:     "kb-20",
		Confidence: "high",
	}
}
