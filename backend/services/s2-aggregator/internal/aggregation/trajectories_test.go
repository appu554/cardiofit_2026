package aggregation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

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

// TestBuildTrajectories_FullData_eGFR exercises the happy-path: 4 eGFR
// observations spanning >1 year produce a populated trajectory with
// velocity, baseline, and threshold flag when value crosses CKD boundary.
func TestBuildTrajectories_FullData_eGFR(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	client := NewInMemorySubstrateClient().WithObservations(
		mkObs(rid, "egfr", 52, asOf.AddDate(-1, 0, 0)),
		mkObs(rid, "egfr", 48, asOf.AddDate(0, -6, 0)),
		mkObs(rid, "egfr", 44, asOf.AddDate(0, -4, 0)),
		mkObs(rid, "egfr", 28, asOf.AddDate(0, 0, -15)), // crosses <30
	)

	trs, err := BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories error: %v", err)
	}

	var egfr *Trajectory
	for i := range trs {
		if trs[i].Parameter == "egfr" {
			egfr = &trs[i]
			break
		}
	}
	if egfr == nil {
		t.Fatal("eGFR trajectory missing")
	}
	if egfr.SparseDataFlag {
		t.Error("eGFR with 4 obs should not be sparse")
	}
	if egfr.CurrentValue == nil || *egfr.CurrentValue != 28 {
		t.Errorf("CurrentValue: got %+v want 28", egfr.CurrentValue)
	}
	if egfr.Velocity == nil {
		t.Error("Velocity should be computed with 4 obs")
	}
	if len(egfr.ThresholdFlags) == 0 || egfr.ThresholdFlags[0].Kind != "egfr_below_30" {
		t.Errorf("expected egfr_below_30 flag; got %+v", egfr.ThresholdFlags)
	}
	if len(egfr.SubstrateRefs) == 0 {
		t.Error("eGFR must carry SubstrateRefs (verification-not-belief)")
	}
}

// TestBuildTrajectories_SparseData_TwoObs verifies v1.0 Part 5.3:
// 2 observations → SparseDataFlag=true, no velocity.
func TestBuildTrajectories_SparseData_TwoObs(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	client := NewInMemorySubstrateClient().WithObservations(
		mkObs(rid, "weight", 72, asOf.AddDate(0, -2, 0)),
		mkObs(rid, "weight", 68, asOf.AddDate(0, 0, -5)),
	)

	trs, err := BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, tr := range trs {
		if tr.Parameter == "weight" {
			if !tr.SparseDataFlag {
				t.Error("2-obs weight should be sparse")
			}
			if tr.Velocity != nil {
				t.Error("velocity must be nil for <3 obs")
			}
			if tr.CurrentValue == nil || *tr.CurrentValue != 68 {
				t.Errorf("CurrentValue mismatch")
			}
			return
		}
	}
	t.Fatal("weight trajectory not found")
}

// TestBuildTrajectories_NoObs verifies the "no observation on record"
// case: trajectory still rendered with SparseDataFlag and a SubstrateRef
// pointing at the resident (verification-not-belief structural check
// applies even to absences).
func TestBuildTrajectories_NoObs(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	client := NewInMemorySubstrateClient()

	trs, err := BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, tr := range trs {
		if !tr.SparseDataFlag {
			t.Errorf("%s: expected sparse on empty substrate", tr.Parameter)
		}
		if len(tr.SubstrateRefs) == 0 {
			t.Errorf("%s: no SubstrateRefs — violates verification-not-belief", tr.Parameter)
		}
	}
}

// TestBuildTrajectories_ThresholdFlag_ACB_DBI_CFS verifies the placeholder
// thresholds fire correctly.
func TestBuildTrajectories_ThresholdFlag_ACB_DBI_CFS(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	client := NewInMemorySubstrateClient().WithObservations(
		mkObs(rid, "acb", 4, asOf),
		mkObs(rid, "dbi", 1.2, asOf),
		mkObs(rid, "cfs", 7, asOf),
	)
	trs, err := BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	wanted := map[string]string{
		"acb": "acb_elevated_placeholder",
		"dbi": "dbi_elevated_placeholder",
		"cfs": "cfs_severely_frail",
	}
	for _, tr := range trs {
		want, ok := wanted[tr.Parameter]
		if !ok {
			continue
		}
		if len(tr.ThresholdFlags) == 0 || tr.ThresholdFlags[0].Kind != want {
			t.Errorf("%s: want flag %q got %+v", tr.Parameter, want, tr.ThresholdFlags)
		}
	}
}

// TestBuildTrajectories_PRNVelocity_Severity verifies the PRN trajectory
// surfaces a severity flag when recent administrations exceed baseline.
func TestBuildTrajectories_PRNVelocity_Severity(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	client := NewInMemorySubstrateClient()
	// Baseline window (now-120d, now-30d]: 3 administrations → 1/30d avg.
	for i := 0; i < 3; i++ {
		client.WithAdministrations(substrate_types.PRNAdministration{
			ResidentID:     rid,
			Class:          substrate_types.PRNBenzodiazepine,
			AdministeredAt: asOf.AddDate(0, 0, -90+i*15),
		})
	}
	// Recent window: 5 administrations → ratio 5.0, severity 5.
	for i := 0; i < 5; i++ {
		client.WithAdministrations(substrate_types.PRNAdministration{
			ResidentID:     rid,
			Class:          substrate_types.PRNBenzodiazepine,
			AdministeredAt: asOf.AddDate(0, 0, -25+i*5),
		})
	}

	trs, err := BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, tr := range trs {
		if tr.Parameter == "prn_velocity_benzodiazepine" {
			if len(tr.ThresholdFlags) == 0 {
				t.Error("expected severity flag on PRN benzodiazepine trajectory")
			}
			if len(tr.SubstrateRefs) == 0 {
				t.Error("PRN trajectory missing SubstrateRef")
			}
			return
		}
	}
	t.Fatal("PRN benzodiazepine trajectory missing")
}

// TestComputeMultiParameterCompositions_AnticholinergicRenal verifies the
// first canonical example: ACB elevated AND eGFR <45.
func TestComputeMultiParameterCompositions_AnticholinergicRenal(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	client := NewInMemorySubstrateClient().WithObservations(
		mkObs(rid, "acb", 4, asOf),
		mkObs(rid, "egfr", 38, asOf),
	)
	trs, _ := BuildTrajectories(context.Background(), client, rid, asOf)
	comps := ComputeMultiParameterCompositions(trs)
	if len(comps) == 0 {
		t.Fatal("expected anticholinergic+renal composition")
	}
	found := false
	for _, c := range comps {
		if c.CompositionLabel == "anticholinergic burden + renal decline" {
			found = true
			if len(c.SubstrateRefs) == 0 {
				t.Error("composition missing SubstrateRefs")
			}
		}
	}
	if !found {
		t.Errorf("composition label not present; got %+v", comps)
	}
}

// TestComputeMultiParameterCompositions_FrailtyPRN verifies the second
// canonical example: CFS ≥6 AND PRN severity ≥3.
func TestComputeMultiParameterCompositions_FrailtyPRN(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	client := NewInMemorySubstrateClient().WithObservations(
		mkObs(rid, "cfs", 7, asOf),
	)
	// Seed PRN escalation severity ≥3 (ratio >1.5).
	for i := 0; i < 2; i++ {
		client.WithAdministrations(substrate_types.PRNAdministration{
			ResidentID:     rid,
			Class:          substrate_types.PRNAntipsychotic,
			AdministeredAt: asOf.AddDate(0, 0, -90+i*30),
		})
	}
	for i := 0; i < 4; i++ {
		client.WithAdministrations(substrate_types.PRNAdministration{
			ResidentID:     rid,
			Class:          substrate_types.PRNAntipsychotic,
			AdministeredAt: asOf.AddDate(0, 0, -20+i*4),
		})
	}

	trs, _ := BuildTrajectories(context.Background(), client, rid, asOf)
	comps := ComputeMultiParameterCompositions(trs)
	found := false
	for _, c := range comps {
		if c.CompositionLabel == "severe frailty + escalating antipsychotic use" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected frailty+antipsychotic composition; got %+v", comps)
	}
}

// TestBuildTrajectories_EveryTrajectoryHasSubstrateRef is the structural
// verification-not-belief check at the package level. The cross-cutting
// version lives in tests/structural/.
func TestBuildTrajectories_EveryTrajectoryHasSubstrateRef(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	client := NewInMemorySubstrateClient().WithObservations(
		mkObs(rid, "egfr", 50, asOf.AddDate(-1, 0, 0)),
		mkObs(rid, "egfr", 45, asOf.AddDate(0, -6, 0)),
		mkObs(rid, "egfr", 40, asOf.AddDate(0, -1, 0)),
	)
	trs, err := BuildTrajectories(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for _, tr := range trs {
		if len(tr.SubstrateRefs) == 0 {
			t.Errorf("trajectory %q has no SubstrateRef — violates verification-not-belief", tr.Parameter)
		}
	}
}

// TestBuildTrajectories_NilClient guards the contract.
func TestBuildTrajectories_NilClient(t *testing.T) {
	_, err := BuildTrajectories(context.Background(), nil, uuid.New(), time.Now())
	if err == nil {
		t.Fatal("expected error on nil client")
	}
}
