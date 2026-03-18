package unit

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

func TestRunSimulation_90Days(t *testing.T) {
	initial := models.SimState{IS: 0.4, VF: 0.6, HGO: 0.5, MM: 0.5, VR: 0.5, RR: 0.8}
	intervention := models.Intervention{
		Type: "LIFESTYLE", Code: "EX001", Description: "Brisk walking 30min/day",
		ISEffect: 0.002, VFEffect: -0.001, MMEffect: 0.001,
	}

	projected := services.RunSimulation(initial, intervention, 90)
	if len(projected) != 4 { // day 0, 30, 60, 90
		t.Fatalf("expected 4 time points, got %d", len(projected))
	}
	last := projected[len(projected)-1]
	if last.State.IS <= initial.IS {
		t.Errorf("IS should improve: initial=%f final=%f", initial.IS, last.State.IS)
	}
}

func TestComputeBiomarkers(t *testing.T) {
	state := models.SimState{IS: 0.6, VF: 0.4, HGO: 0.4, MM: 0.6, VR: 0.4, RR: 0.8}
	bio := services.ComputeBiomarkers(state)
	if bio.FBG <= 0 || bio.SBP <= 0 {
		t.Errorf("biomarkers should be positive: FBG=%f SBP=%f", bio.FBG, bio.SBP)
	}
}
