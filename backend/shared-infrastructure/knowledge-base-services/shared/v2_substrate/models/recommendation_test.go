package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRecommendationJSONRoundTrip(t *testing.T) {
	medUse := uuid.New()
	in := Recommendation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		AuthorID:   uuid.New(),
		State:      RecommendationStateDrafted,
		Type:       RecommendationTypeStop,
		Urgency:    RecommendationUrgencyAmber,
		Title:      "Cease oxybutynin",
		ClinicalContent: ClinicalContent{
			Issue:           "Anticholinergic burden contributing to fall risk",
			ClinicalContext: "87yo female, eGFR 32, recent fall, ACB 4",
			Rationale:       "DBI 0.8 attributable; alternatives reviewed",
			EvidenceRefs:    []string{"ADG-2025-Rec-42", "Beers-2023-OAB"},
			ProposedPlan:    "Cease oxybutynin 5mg BD; monitor for urinary retention 14 days",
			MonitoringPlan:  "Voiding diary 14 days; falls reassessment at 30 days",
		},
		MedicineUseRefs: []uuid.UUID{medUse},
		ConsentRequired: false,
		ReviewDueAt:     nil,
		SubmittedAt:     nil,
		CreatedAt:       time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:       time.Now().UTC().Truncate(time.Microsecond),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Recommendation
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID != in.ID || out.State != in.State || out.Type != in.Type {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
	if out.ClinicalContent.Issue != in.ClinicalContent.Issue {
		t.Errorf("clinical content lost in round trip")
	}
	if len(out.MedicineUseRefs) != 1 || out.MedicineUseRefs[0] != medUse {
		t.Errorf("medicine use refs lost: %v", out.MedicineUseRefs)
	}
}

func TestIsValidRecommendationState(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{RecommendationStateDetected, true},
		{RecommendationStateDeferred, true},
		{RecommendationStateClosed, true},
		{"bogus", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsValidRecommendationState(c.s); got != c.want {
			t.Errorf("IsValidRecommendationState(%q)=%v want %v", c.s, got, c.want)
		}
	}
}

func TestRecommendationTransitionMatrix(t *testing.T) {
	type tc struct {
		from, to string
		want     bool
	}
	cases := []tc{
		// Happy path
		{RecommendationStateDetected, RecommendationStateDrafted, true},
		{RecommendationStateDrafted, RecommendationStateSubmitted, true},
		{RecommendationStateSubmitted, RecommendationStateViewed, true},
		{RecommendationStateViewed, RecommendationStateDecided, true},
		{RecommendationStateDecided, RecommendationStateImplemented, true},
		{RecommendationStateImplemented, RecommendationStateMonitoringActive, true},
		{RecommendationStateMonitoringActive, RecommendationStateOutcomeRecorded, true},
		{RecommendationStateOutcomeRecorded, RecommendationStateClosed, true},

		// Deferred branches
		{RecommendationStateSubmitted, RecommendationStateDeferred, true},
		{RecommendationStateViewed, RecommendationStateDeferred, true},
		{RecommendationStateDeferred, RecommendationStateSubmitted, true},
		{RecommendationStateDeferred, RecommendationStateClosed, true},

		// Direct-to-closed escapes
		{RecommendationStateDetected, RecommendationStateClosed, true},
		{RecommendationStateDrafted, RecommendationStateClosed, true},
		{RecommendationStateSubmitted, RecommendationStateClosed, true},
		{RecommendationStateViewed, RecommendationStateClosed, true},
		{RecommendationStateDecided, RecommendationStateClosed, true},

		// Skip-monitoring branch (Implemented → OutcomeRecorded directly)
		{RecommendationStateImplemented, RecommendationStateOutcomeRecorded, true},

		// Forbidden: terminal
		{RecommendationStateClosed, RecommendationStateDrafted, false},
		{RecommendationStateClosed, RecommendationStateSubmitted, false},

		// Forbidden: skipping decided
		{RecommendationStateViewed, RecommendationStateImplemented, false},
		{RecommendationStateSubmitted, RecommendationStateDecided, false},

		// Forbidden: backwards
		{RecommendationStateDecided, RecommendationStateSubmitted, false},
		{RecommendationStateMonitoringActive, RecommendationStateImplemented, false},

		// Forbidden: bogus
		{"bogus", RecommendationStateDrafted, false},
		{RecommendationStateDrafted, "bogus", false},
	}
	for _, c := range cases {
		if got := IsValidTransition(c.from, c.to); got != c.want {
			t.Errorf("IsValidTransition(%q, %q) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestIsValidRecommendationType(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{RecommendationTypeStop, true},
		{RecommendationTypeMonitor, true},
		{RecommendationTypeDoseChange, true},
		{RecommendationTypeAdd, true},
		{"bogus", false},
	}
	for _, c := range cases {
		if got := IsValidRecommendationType(c.s); got != c.want {
			t.Errorf("IsValidRecommendationType(%q)=%v want %v", c.s, got, c.want)
		}
	}
}

func TestIsValidRecommendationUrgency(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{RecommendationUrgencyRed, true},
		{RecommendationUrgencyAmber, true},
		{RecommendationUrgencyGreen, true},
		{"bogus", false},
	}
	for _, c := range cases {
		if got := IsValidRecommendationUrgency(c.s); got != c.want {
			t.Errorf("IsValidRecommendationUrgency(%q)=%v want %v", c.s, got, c.want)
		}
	}
}

// TestRecommendation_RejectedWithdrawnSupersededAreValid asserts that the three
// new terminal states are accepted by IsValidRecommendationState and that all
// 13 canonical states (Guidelines Part 3) are present.
func TestRecommendation_RejectedWithdrawnSupersededAreValid(t *testing.T) {
	all13 := []string{
		RecommendationStateDetected,
		RecommendationStateDrafted,
		RecommendationStateSubmitted,
		RecommendationStateViewed,
		RecommendationStateDeferred,
		RecommendationStateDecided,
		RecommendationStateImplemented,
		RecommendationStateMonitoringActive,
		RecommendationStateOutcomeRecorded,
		RecommendationStateClosed,
		RecommendationStateRejected,
		RecommendationStateWithdrawn,
		RecommendationStateSuperseded,
	}
	if len(all13) != 13 {
		t.Fatalf("expected 13 states in the canonical set, got %d", len(all13))
	}
	for _, s := range all13 {
		if !IsValidRecommendationState(s) {
			t.Errorf("IsValidRecommendationState(%q) = false, want true", s)
		}
	}
	// Sanity: bogus still rejected.
	if IsValidRecommendationState("bogus") {
		t.Error("IsValidRecommendationState(\"bogus\") = true, want false")
	}
}

// TestRecommendation_RejectedFromDecided asserts that decided → rejected is
// permitted (GP declines a pharmacist proposal outright).
func TestRecommendation_RejectedFromDecided(t *testing.T) {
	if !IsValidTransition(RecommendationStateDecided, RecommendationStateRejected) {
		t.Errorf("IsValidTransition(decided, rejected) = false, want true")
	}
}

// TestRecommendation_DraftedToWithdrawn asserts that drafted → withdrawn is
// permitted (author pulls the recommendation before submission).
func TestRecommendation_DraftedToWithdrawn(t *testing.T) {
	if !IsValidTransition(RecommendationStateDrafted, RecommendationStateWithdrawn) {
		t.Errorf("IsValidTransition(drafted, withdrawn) = false, want true")
	}
}

// TestRecommendation_ImplementedToSuperseded asserts that implemented →
// superseded is permitted (a newer guideline supersedes the action).
func TestRecommendation_ImplementedToSuperseded(t *testing.T) {
	if !IsValidTransition(RecommendationStateImplemented, RecommendationStateSuperseded) {
		t.Errorf("IsValidTransition(implemented, superseded) = false, want true")
	}
}

// TestRecommendation_MonitoringActiveToSuperseded asserts that
// monitoring-active → superseded is permitted.
func TestRecommendation_MonitoringActiveToSuperseded(t *testing.T) {
	if !IsValidTransition(RecommendationStateMonitoringActive, RecommendationStateSuperseded) {
		t.Errorf("IsValidTransition(monitoring-active, superseded) = false, want true")
	}
}

// TestRecommendation_RejectedIsTerminal asserts that no transitions are
// permitted out of rejected.
func TestRecommendation_RejectedIsTerminal(t *testing.T) {
	allStates := []string{
		RecommendationStateDetected, RecommendationStateDrafted,
		RecommendationStateSubmitted, RecommendationStateViewed,
		RecommendationStateDeferred, RecommendationStateDecided,
		RecommendationStateImplemented, RecommendationStateMonitoringActive,
		RecommendationStateOutcomeRecorded, RecommendationStateClosed,
		RecommendationStateRejected, RecommendationStateWithdrawn,
		RecommendationStateSuperseded,
	}
	for _, to := range allStates {
		if IsValidTransition(RecommendationStateRejected, to) {
			t.Errorf("IsValidTransition(rejected, %q) = true, want false — rejected is terminal", to)
		}
	}
}

// TestRecommendation_SupersededIsTerminal asserts that no transitions are
// permitted out of superseded.
func TestRecommendation_SupersededIsTerminal(t *testing.T) {
	allStates := []string{
		RecommendationStateDetected, RecommendationStateDrafted,
		RecommendationStateSubmitted, RecommendationStateViewed,
		RecommendationStateDeferred, RecommendationStateDecided,
		RecommendationStateImplemented, RecommendationStateMonitoringActive,
		RecommendationStateOutcomeRecorded, RecommendationStateClosed,
		RecommendationStateRejected, RecommendationStateWithdrawn,
		RecommendationStateSuperseded,
	}
	for _, to := range allStates {
		if IsValidTransition(RecommendationStateSuperseded, to) {
			t.Errorf("IsValidTransition(superseded, %q) = true, want false — superseded is terminal", to)
		}
	}
}

// TestRecommendation_WithdrawnIsTerminal asserts that no transitions are
// permitted out of withdrawn.
func TestRecommendation_WithdrawnIsTerminal(t *testing.T) {
	allStates := []string{
		RecommendationStateDetected, RecommendationStateDrafted,
		RecommendationStateSubmitted, RecommendationStateViewed,
		RecommendationStateDeferred, RecommendationStateDecided,
		RecommendationStateImplemented, RecommendationStateMonitoringActive,
		RecommendationStateOutcomeRecorded, RecommendationStateClosed,
		RecommendationStateRejected, RecommendationStateWithdrawn,
		RecommendationStateSuperseded,
	}
	for _, to := range allStates {
		if IsValidTransition(RecommendationStateWithdrawn, to) {
			t.Errorf("IsValidTransition(withdrawn, %q) = true, want false — withdrawn is terminal", to)
		}
	}
}
