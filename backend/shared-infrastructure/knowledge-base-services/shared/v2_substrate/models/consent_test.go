package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConsentJSONRoundTrip(t *testing.T) {
	validUntil := time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Microsecond)
	in := Consent{
		ID:            uuid.New(),
		ResidentID:    uuid.New(),
		Class:         ConsentClassPsychotropic,
		State:         ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "substitute_decision_maker",
		Conditions:    "valid only for risperidone <0.5mg BD",
		ScopeNotes:    "covers BPSD recommendations through 2026-12",
		ValidFrom:     time.Now().UTC().Truncate(time.Microsecond),
		ValidUntil:    &validUntil,
		CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:     time.Now().UTC().Truncate(time.Microsecond),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Consent
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID != in.ID || out.State != in.State || out.Class != in.Class {
		t.Errorf("scalars mismatched: got %+v want %+v", out, in)
	}
	if out.GrantedByID != in.GrantedByID || out.GrantedByRole != in.GrantedByRole {
		t.Errorf("grantor info lost in round trip")
	}
	if out.Conditions != in.Conditions || out.ScopeNotes != in.ScopeNotes {
		t.Errorf("conditions/scope lost in round trip")
	}
	if out.ValidUntil == nil || !out.ValidUntil.Equal(*in.ValidUntil) {
		t.Errorf("ValidUntil lost in round trip")
	}
}

func TestConsentRoundTripNullableValidUntil(t *testing.T) {
	// Open-ended consent (no expiry) — ValidUntil nil
	in := Consent{
		ID:            uuid.New(),
		ResidentID:    uuid.New(),
		Class:         ConsentClassGeneralMedication,
		State:         ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "resident_self",
		ValidFrom:     time.Now().UTC().Truncate(time.Microsecond),
		ValidUntil:    nil,
		CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:     time.Now().UTC().Truncate(time.Microsecond),
	}
	raw, _ := json.Marshal(in)
	var out Consent
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ValidUntil != nil {
		t.Errorf("expected nil ValidUntil; got %v", *out.ValidUntil)
	}
}

func TestConsentTransitionMatrix(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		// Happy path
		{ConsentStateRequested, ConsentStateDiscussed, true},
		{ConsentStateRequested, ConsentStateRefused, true}, // declined before discussion
		{ConsentStateDiscussed, ConsentStateGranted, true},
		{ConsentStateDiscussed, ConsentStateGrantedWithConditions, true},
		{ConsentStateDiscussed, ConsentStateRefused, true},
		{ConsentStateGranted, ConsentStateActive, true},
		{ConsentStateGrantedWithConditions, ConsentStateActive, true},
		{ConsentStateActive, ConsentStateUnderReview, true},
		{ConsentStateActive, ConsentStateWithdrawn, true},
		{ConsentStateActive, ConsentStateExpired, true},
		{ConsentStateUnderReview, ConsentStateActive, true},
		{ConsentStateUnderReview, ConsentStateWithdrawn, true},

		// Forbidden — terminal states
		{ConsentStateRefused, ConsentStateActive, false},
		{ConsentStateRefused, ConsentStateGranted, false},
		{ConsentStateExpired, ConsentStateActive, false},
		{ConsentStateWithdrawn, ConsentStateActive, false},

		// Forbidden — skipping discussed
		{ConsentStateRequested, ConsentStateGranted, false},
		{ConsentStateRequested, ConsentStateActive, false},

		// Forbidden — skipping active
		{ConsentStateGranted, ConsentStateUnderReview, false},
		{ConsentStateGrantedWithConditions, ConsentStateWithdrawn, false},

		// Forbidden — bogus
		{"bogus", ConsentStateActive, false},
		{ConsentStateActive, "bogus", false},
	}
	for _, c := range cases {
		if got := IsValidConsentTransition(c.from, c.to); got != c.want {
			t.Errorf("IsValidConsentTransition(%q,%q)=%v want %v",
				c.from, c.to, got, c.want)
		}
	}
}

func TestIsValidConsentState(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{ConsentStateRequested, true},
		{ConsentStateDiscussed, true},
		{ConsentStateActive, true},
		{ConsentStateExpired, true},
		{"bogus", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsValidConsentState(c.s); got != c.want {
			t.Errorf("IsValidConsentState(%q)=%v want %v", c.s, got, c.want)
		}
	}
}

func TestIsValidConsentClass(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{ConsentClassPsychotropic, true},
		{ConsentClassRestrictivePractice, true},
		{ConsentClassChemoTherapy, true},
		{ConsentClassEndOfLifeMedication, true},
		{ConsentClassGeneralMedication, true},
		{"bogus", false},
	}
	for _, c := range cases {
		if got := IsValidConsentClass(c.s); got != c.want {
			t.Errorf("IsValidConsentClass(%q)=%v want %v", c.s, got, c.want)
		}
	}
}
