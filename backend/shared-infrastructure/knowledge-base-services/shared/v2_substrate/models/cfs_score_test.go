package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCFSScoreJSONRoundTrip(t *testing.T) {
	in := CFSScore{
		ID:                uuid.New(),
		ResidentRef:       uuid.New(),
		AssessedAt:        time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC),
		AssessorRoleRef:   uuid.New(),
		InstrumentVersion: "v2.0",
		Score:             7,
		Rationale:         "Severe frailty — bed-bound 12 weeks post-stroke",
		CreatedAt:         time.Now().UTC(),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out CFSScore
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Score != in.Score {
		t.Errorf("Score drift: got %d want %d", out.Score, in.Score)
	}
	if out.InstrumentVersion != in.InstrumentVersion {
		t.Errorf("InstrumentVersion drift: got %q want %q", out.InstrumentVersion, in.InstrumentVersion)
	}
	if out.Rationale != in.Rationale {
		t.Errorf("Rationale drift: got %q want %q", out.Rationale, in.Rationale)
	}
	if out.AssessedAt != in.AssessedAt {
		t.Errorf("AssessedAt drift")
	}
}

func TestCFSScoreShouldHintCareIntensityReview(t *testing.T) {
	cases := []struct {
		score int
		want  bool
	}{
		{1, false},
		{4, false},
		{6, false},
		{7, true},
		{8, true},
		{9, true},
	}
	for _, tc := range cases {
		if got := CFSScoreShouldHintCareIntensityReview(tc.score); got != tc.want {
			t.Errorf("CFSScoreShouldHintCareIntensityReview(%d) = %v; want %v", tc.score, got, tc.want)
		}
	}
}
