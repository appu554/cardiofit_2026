package failed_interventions

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsRetryEligible(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		eligibleAt time.Time
		want    bool
	}{
		{"future eligibility — not retry-eligible", now.Add(24 * time.Hour), false},
		{"past eligibility — retry-eligible", now.Add(-24 * time.Hour), true},
		// Boundary: RetryEligibleDate == now → veto no longer active → IS retry-eligible.
		// Chosen semantic: non-strict (RetryEligibleDate <= now ⇒ eligible).
		// Mirrors CAPE Guidelines line 645's strict After() check on the veto side.
		{"boundary: exactly now — IS retry-eligible (non-strict, <= now)", now, true},
		// One nanosecond into the future — still vetoed.
		{"boundary: now+1ns future — NOT retry-eligible", now.Add(time.Nanosecond), false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := FailedInterventionRecord{RetryEligibleDate: tc.eligibleAt}
			if got := r.IsRetryEligible(now); got != tc.want {
				t.Errorf("IsRetryEligible(%v): got %v want %v", tc.eligibleAt, got, tc.want)
			}
		})
	}
}

func TestIsVetoActive(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	rid := uuid.New()
	doc := uuid.New()

	active := FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "antipsychotic_deprescribing",
		RetryEligibleDate: now.Add(30 * 24 * time.Hour),
		DocumentedBy:      doc,
	}
	expired := FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "benzodiazepine_deprescribing",
		RetryEligibleDate: now.Add(-30 * 24 * time.Hour),
		DocumentedBy:      doc,
	}
	otherType := FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "dose_reduction",
		RetryEligibleDate: now.Add(30 * 24 * time.Hour),
		DocumentedBy:      doc,
	}

	cases := []struct {
		name             string
		records          []FailedInterventionRecord
		interventionType string
		want             bool
	}{
		{"no records — no veto", nil, "antipsychotic_deprescribing", false},
		{"empty intervention type — no match", []FailedInterventionRecord{active}, "", false},
		{"active match", []FailedInterventionRecord{active}, "antipsychotic_deprescribing", true},
		{"case-insensitive match", []FailedInterventionRecord{active}, "Antipsychotic_Deprescribing", true},
		{"expired record — no veto", []FailedInterventionRecord{expired}, "benzodiazepine_deprescribing", false},
		{"wrong type — no veto", []FailedInterventionRecord{otherType}, "antipsychotic_deprescribing", false},
		{"OR semantics: one active among many → veto", []FailedInterventionRecord{expired, otherType, active}, "antipsychotic_deprescribing", true},
		{"all expired — no veto", []FailedInterventionRecord{expired, expired}, "benzodiazepine_deprescribing", false},
		{"record with empty intervention type ignored", []FailedInterventionRecord{{InterventionType: "", RetryEligibleDate: now.Add(time.Hour)}}, "antipsychotic_deprescribing", false},
		// Boundary: RetryEligibleDate exactly == now is NOT active (strict After).
		{"boundary: eligibility exactly now — not active", []FailedInterventionRecord{{InterventionType: "x", RetryEligibleDate: now}}, "x", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsVetoActive(tc.records, tc.interventionType, now); got != tc.want {
				t.Errorf("IsVetoActive(%q): got %v want %v", tc.interventionType, got, tc.want)
			}
		})
	}
}

func TestIsRetryEligibleAndIsVetoActiveAreNegations(t *testing.T) {
	// Single-record consistency check: IsRetryEligible == !IsVetoActive
	// when the record matches and is the only one in the slice.
	t.Parallel()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	for _, delta := range []time.Duration{-time.Hour, 0, time.Hour, time.Nanosecond, -time.Nanosecond} {
		r := FailedInterventionRecord{
			InterventionType:  "x",
			RetryEligibleDate: now.Add(delta),
		}
		eligible := r.IsRetryEligible(now)
		veto := IsVetoActive([]FailedInterventionRecord{r}, "x", now)
		if eligible == veto {
			t.Errorf("delta=%v: IsRetryEligible=%v IsVetoActive=%v — should be negations", delta, eligible, veto)
		}
	}
}
