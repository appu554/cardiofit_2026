package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAKPSScoreJSONRoundTrip(t *testing.T) {
	in := AKPSScore{
		ID:                uuid.New(),
		ResidentRef:       uuid.New(),
		AssessedAt:        time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC),
		AssessorRoleRef:   uuid.New(),
		InstrumentVersion: "abernethy_2005",
		Score:             40,
		Rationale:         "In bed >50% of day, requires assistance with ADLs",
		CreatedAt:         time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out AKPSScore
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Score != in.Score {
		t.Errorf("Score drift")
	}
	if out.InstrumentVersion != in.InstrumentVersion {
		t.Errorf("InstrumentVersion drift")
	}
}

func TestAKPSScoreShouldHintCareIntensityReview(t *testing.T) {
	cases := []struct {
		score int
		want  bool
	}{
		{0, true},
		{20, true},
		{40, true},
		{50, false},
		{80, false},
		{100, false},
	}
	for _, tc := range cases {
		if got := AKPSScoreShouldHintCareIntensityReview(tc.score); got != tc.want {
			t.Errorf("AKPSScoreShouldHintCareIntensityReview(%d) = %v; want %v", tc.score, got, tc.want)
		}
	}
}
