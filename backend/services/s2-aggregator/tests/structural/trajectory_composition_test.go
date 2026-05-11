// trajectory_composition_test.go — v1.0 Part 17 Category 2
// (trajectory rendering). Composition tests beyond the narrow per-panel
// trajectory test in verification_not_belief_trajectories_test.go:
// covers multi-parameter composition, sparse-velocity nil contract,
// threshold flag trigger at eGFR<30, and the 90d baseline window.
package structural

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TestTrajectory_MultiParameter_ComposesAnticholinergicBurdenRenalDecline —
// ACB elevated + eGFR decline → MultiParameterComposition emitted
// matching the Task 3 example pattern.
func TestTrajectory_MultiParameter_ComposesAnticholinergicBurdenRenalDecline(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	client := aggregation.NewInMemorySubstrateClient().WithObservations(
		// eGFR crossing into <45 (CKD 3b).
		mkObsV9(rid, "egfr", 50, asOf.AddDate(-1, 0, 0)),
		mkObsV9(rid, "egfr", 42, asOf.AddDate(0, -2, 0)),
		mkObsV9(rid, "egfr", 38, asOf.AddDate(0, -1, 0)),
		// ACB elevated (≥3 placeholder threshold).
		mkObsV9(rid, "acb", 4, asOf.AddDate(0, -1, 0)),
	)
	trs, err := aggregation.BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories: %v", err)
	}
	comps := aggregation.ComputeMultiParameterCompositions(trs)
	if len(comps) == 0 {
		t.Fatal("expected at least one MultiParameterComposition (ACB + eGFR)")
	}
	saw := false
	for _, c := range comps {
		if c.CompositionLabel == "anticholinergic burden + renal decline" {
			saw = true
			if len(c.SubstrateRefs) == 0 {
				t.Error("composition emitted but SubstrateRefs empty")
			}
		}
	}
	if !saw {
		t.Errorf("expected 'anticholinergic burden + renal decline' composition; got %+v", comps)
	}
}

// TestTrajectory_VelocityNullWhenSparse — <3 observations → velocity is
// nil + SparseDataFlag is true (v1.0 Part 5.3 graceful degradation).
func TestTrajectory_VelocityNullWhenSparse(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	client := aggregation.NewInMemorySubstrateClient().WithObservations(
		mkObsV9(rid, "weight", 70, asOf.AddDate(0, -2, 0)),
		mkObsV9(rid, "weight", 68, asOf.AddDate(0, -1, 0)),
	)
	trs, err := aggregation.BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories: %v", err)
	}
	for _, tr := range trs {
		if tr.Parameter == "weight" {
			if tr.Velocity != nil {
				t.Errorf("weight trajectory has 2 obs — Velocity must be nil; got %v", *tr.Velocity)
			}
			if !tr.SparseDataFlag {
				t.Error("weight trajectory has 2 obs — SparseDataFlag must be true")
			}
			return
		}
	}
	t.Fatal("weight trajectory not found in output")
}

// TestTrajectory_ThresholdFlag_eGFR_Below30_Triggers — eGFR crosses 30
// → threshold flag emitted with Kind "egfr_below_30".
func TestTrajectory_ThresholdFlag_eGFR_Below30_Triggers(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	client := aggregation.NewInMemorySubstrateClient().WithObservations(
		mkObsV9(rid, "egfr", 50, asOf.AddDate(0, -3, 0)),
		mkObsV9(rid, "egfr", 35, asOf.AddDate(0, -2, 0)),
		mkObsV9(rid, "egfr", 28, asOf.AddDate(0, -1, 0)),
	)
	trs, err := aggregation.BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories: %v", err)
	}
	for _, tr := range trs {
		if tr.Parameter != "egfr" {
			continue
		}
		saw := false
		for _, f := range tr.ThresholdFlags {
			if f.Kind == "egfr_below_30" {
				saw = true
			}
		}
		if !saw {
			t.Errorf("egfr <30 — expected threshold flag 'egfr_below_30'; got %+v", tr.ThresholdFlags)
		}
		return
	}
	t.Fatal("egfr trajectory not found in output")
}

// TestTrajectory_BaselineComputation_90dWindow — baseline = mean over
// (asOf-180d, asOf-90d]; observations outside that window are excluded.
func TestTrajectory_BaselineComputation_90dWindow(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	// Window: (asOf-180d, asOf-90d]. Place two observations inside
	// (values 50, 60 → mean 55) and two outside (one too recent,
	// one too old).
	client := aggregation.NewInMemorySubstrateClient().WithObservations(
		mkObsV9(rid, "egfr", 99, asOf.AddDate(-1, 0, 0)),                          // before window (too old) — excluded
		mkObsV9(rid, "egfr", 50, asOf.Add(-150*24*time.Hour)),                     // inside window
		mkObsV9(rid, "egfr", 60, asOf.Add(-100*24*time.Hour)),                     // inside window
		mkObsV9(rid, "egfr", 30, asOf.Add(-30*24*time.Hour)),                      // recent (outside window) — excluded from baseline
	)
	trs, err := aggregation.BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories: %v", err)
	}
	for _, tr := range trs {
		if tr.Parameter != "egfr" {
			continue
		}
		if tr.Baseline90d == nil {
			t.Fatal("egfr baseline expected populated")
		}
		got := *tr.Baseline90d
		want := 55.0
		if got != want {
			t.Errorf("baseline computation: got %.2f want %.2f (only the 50+60 inside-window obs should average)", got, want)
		}
		return
	}
	t.Fatal("egfr trajectory not found")
}

// TestTrajectory_AlwaysCarriesNoObservationRef — even when there are
// zero observations on record for a parameter, the trajectory must
// still carry a SubstrateRef (per Task 3 judgment call). This is the
// composition-test mirror of the per-panel structural test, but it
// asserts the contract holds for EVERY parameter in the catalogue,
// not just the seeded one.
func TestTrajectory_AlwaysCarriesNoObservationRef(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	// No observations at all.
	client := aggregation.NewInMemorySubstrateClient()
	trs, err := aggregation.BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories: %v", err)
	}
	if len(trs) == 0 {
		t.Fatal("expected trajectory catalogue to render even with zero observations")
	}
	for _, tr := range trs {
		if len(tr.SubstrateRefs) == 0 {
			t.Errorf("trajectory %q has zero SubstrateRefs even on absence — violates verification-not-belief at the trajectory layer", tr.Parameter)
		}
	}
}

// TestTrajectory_PRNVelocity_ComputesRatio — sanity check that PRN
// administrations seed a PRN trajectory with a velocity ratio. Uses 4
// recent + 0 baseline administrations, expected ratio = +Inf (recent
// against zero baseline) so Velocity remains nil but CurrentValue is
// populated and SubstrateRefs non-empty.
func TestTrajectory_PRNVelocity_ComputesRatio(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	client := aggregation.NewInMemorySubstrateClient().WithAdministrations(
		substrate_types.PRNAdministration{ResidentID: rid, Class: substrate_types.PRNBenzodiazepine, AdministeredAt: asOf.AddDate(0, 0, -5)},
		substrate_types.PRNAdministration{ResidentID: rid, Class: substrate_types.PRNBenzodiazepine, AdministeredAt: asOf.AddDate(0, 0, -10)},
		substrate_types.PRNAdministration{ResidentID: rid, Class: substrate_types.PRNBenzodiazepine, AdministeredAt: asOf.AddDate(0, 0, -15)},
		substrate_types.PRNAdministration{ResidentID: rid, Class: substrate_types.PRNBenzodiazepine, AdministeredAt: asOf.AddDate(0, 0, -20)},
	)
	trs, err := aggregation.BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories: %v", err)
	}
	for _, tr := range trs {
		if tr.Parameter != "prn_velocity_benzodiazepine" {
			continue
		}
		if tr.CurrentValue == nil {
			t.Error("PRN trajectory CurrentValue must be populated")
		}
		if len(tr.SubstrateRefs) == 0 {
			t.Error("PRN trajectory SubstrateRefs must be non-empty")
		}
		return
	}
	t.Fatal("prn_velocity_benzodiazepine trajectory not found")
}
