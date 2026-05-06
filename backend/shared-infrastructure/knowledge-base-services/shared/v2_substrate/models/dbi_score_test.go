package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDBIScoreJSONRoundTrip(t *testing.T) {
	medRef1 := uuid.New()
	medRef2 := uuid.New()
	in := DBIScore{
		ID:                       uuid.New(),
		ResidentRef:              uuid.New(),
		ComputedAt:               time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC),
		Score:                    1.5,
		AnticholinergicComponent: 1.0,
		SedativeComponent:        0.5,
		ComputationInputs:        []uuid.UUID{medRef1, medRef2},
		UnknownDrugs:             []string{"some-novel-agent"},
		CreatedAt:                time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out DBIScore
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Score != in.Score {
		t.Errorf("Score drift: got %v want %v", out.Score, in.Score)
	}
	if out.AnticholinergicComponent != in.AnticholinergicComponent {
		t.Errorf("Anticholinergic drift")
	}
	if out.SedativeComponent != in.SedativeComponent {
		t.Errorf("Sedative drift")
	}
	if len(out.ComputationInputs) != 2 {
		t.Fatalf("ComputationInputs length drift: got %d", len(out.ComputationInputs))
	}
	if out.ComputationInputs[0] != medRef1 || out.ComputationInputs[1] != medRef2 {
		t.Errorf("ComputationInputs drift")
	}
	if len(out.UnknownDrugs) != 1 || out.UnknownDrugs[0] != "some-novel-agent" {
		t.Errorf("UnknownDrugs drift: %v", out.UnknownDrugs)
	}
}
