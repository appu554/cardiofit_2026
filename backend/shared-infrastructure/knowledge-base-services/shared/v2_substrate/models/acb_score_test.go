package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestACBScoreJSONRoundTrip(t *testing.T) {
	medRef1 := uuid.New()
	in := ACBScore{
		ID:                uuid.New(),
		ResidentRef:       uuid.New(),
		ComputedAt:        time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC),
		Score:             6,
		ComputationInputs: []uuid.UUID{medRef1},
		UnknownDrugs:      []string{"experimental-x"},
		CreatedAt:         time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out ACBScore
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Score != in.Score {
		t.Errorf("Score drift")
	}
	if len(out.ComputationInputs) != 1 || out.ComputationInputs[0] != medRef1 {
		t.Errorf("ComputationInputs drift")
	}
	if len(out.UnknownDrugs) != 1 || out.UnknownDrugs[0] != "experimental-x" {
		t.Errorf("UnknownDrugs drift")
	}
}
