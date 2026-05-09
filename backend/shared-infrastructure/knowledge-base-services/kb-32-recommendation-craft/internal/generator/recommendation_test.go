// Package generator_test exercises the recommendation generator (Stage 3).
package generator_test

import (
	"strings"
	"testing"
	"time"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/generator"
	"github.com/cardiofit/kb32/internal/reasoning"
	"github.com/cardiofit/kb32/internal/template"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func freshSnapshot() kb32ctx.ClinicalSnapshot {
	return kb32ctx.ClinicalSnapshot{
		ResidentID:         uuid.New(),
		EGFR:               52.4,
		DBI:                1.1,
		ACB:                3,
		CFS:                6,
		CareIntensity:      "active",
		RecentFall72h:      true,
		RecentAdmission72h: false,
		AssessedAt:         time.Now(),
	}
}

func oneRule(ruleType, urgency string) []reasoning.ApplicableRule {
	return []reasoning.ApplicableRule{
		{RuleID: "TEST-001", Type: ruleType, Urgency: urgency},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestGenerate_HappyPath verifies that a valid snapshot + 1 rule produces a
// non-nil Packet with all 7 sections present and template-valid.
func TestGenerate_HappyPath(t *testing.T) {
	snap := freshSnapshot()
	rules := oneRule("STOP", "HIGH")
	authorID := uuid.New()

	pkt, err := generator.Generate(snap, rules, authorID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkt == nil {
		t.Fatal("expected non-nil Packet")
	}

	// All 7 sections must be present and template-valid.
	if err := template.Enforce(pkt.Sections); err != nil {
		t.Errorf("Enforce failed on generated packet: %v", err)
	}

	// Core fields.
	if pkt.RecommendationID == uuid.Nil {
		t.Error("RecommendationID must be set")
	}
	if pkt.AuthorID != authorID {
		t.Errorf("AuthorID: got %v want %v", pkt.AuthorID, authorID)
	}
	if pkt.Type != "STOP" {
		t.Errorf("Type: got %q want %q", pkt.Type, "STOP")
	}
	if pkt.AppliedRule.RuleID != "TEST-001" {
		t.Errorf("AppliedRule.RuleID: got %q want %q", pkt.AppliedRule.RuleID, "TEST-001")
	}
	if pkt.SnapshotRef != snap.ResidentID {
		t.Errorf("SnapshotRef: got %v want %v", pkt.SnapshotRef, snap.ResidentID)
	}
}

// TestGenerate_NoRulesError verifies that an empty rules slice returns
// ErrNoApplicableRules.
func TestGenerate_NoRulesError(t *testing.T) {
	snap := freshSnapshot()
	_, err := generator.Generate(snap, nil, uuid.New())
	if err == nil {
		t.Fatal("expected ErrNoApplicableRules, got nil")
	}
	if err != generator.ErrNoApplicableRules {
		t.Errorf("expected ErrNoApplicableRules, got: %v", err)
	}
}

// TestGenerate_EmptyRulesSliceError verifies that an explicitly empty (non-nil)
// slice also returns ErrNoApplicableRules.
func TestGenerate_EmptyRulesSliceError(t *testing.T) {
	snap := freshSnapshot()
	_, err := generator.Generate(snap, []reasoning.ApplicableRule{}, uuid.New())
	if err == nil {
		t.Fatal("expected ErrNoApplicableRules, got nil")
	}
	if err != generator.ErrNoApplicableRules {
		t.Errorf("expected ErrNoApplicableRules, got: %v", err)
	}
}

// TestGenerate_FirstRuleWins verifies that when two rules are provided, the
// generated packet uses the first one.
func TestGenerate_FirstRuleWins(t *testing.T) {
	snap := freshSnapshot()
	rules := []reasoning.ApplicableRule{
		{RuleID: "FIRST-001", Type: "STOP", Urgency: "HIGH"},
		{RuleID: "SECOND-002", Type: "MONITOR", Urgency: "ROUTINE"},
	}

	pkt, err := generator.Generate(snap, rules, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkt.Type != "STOP" {
		t.Errorf("Type: got %q want %q (first rule should win)", pkt.Type, "STOP")
	}
	if pkt.AppliedRule.RuleID != "FIRST-001" {
		t.Errorf("AppliedRule.RuleID: got %q want %q", pkt.AppliedRule.RuleID, "FIRST-001")
	}
}

// TestGenerate_TemplateEnforcedAtConstruction proves the generator's output is
// template-conformant by verifying that manually corrupting a section causes
// Enforce to fail — confirming the original packet had valid sections.
func TestGenerate_TemplateEnforcedAtConstruction(t *testing.T) {
	snap := freshSnapshot()
	rules := oneRule("MONITOR", "ROUTINE")

	pkt, err := generator.Generate(snap, rules, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First verify the original packet passes.
	if err := template.Enforce(pkt.Sections); err != nil {
		t.Fatalf("original packet should be template-valid: %v", err)
	}

	// Now corrupt one section to empty string.
	pkt.Sections[template.SectionRationale] = ""

	// Enforce must now fail.
	if err := template.Enforce(pkt.Sections); err == nil {
		t.Error("expected Enforce to fail after corrupting a section, got nil")
	}
}

// TestGenerate_SectionsContainSnapshotData verifies that snapshot clinical
// values (eGFR, DBI, CFS, CareIntensity) appear in the ClinicalContext section.
func TestGenerate_SectionsContainSnapshotData(t *testing.T) {
	snap := kb32ctx.ClinicalSnapshot{
		ResidentID:    uuid.New(),
		EGFR:          48.7,
		DBI:           2.3,
		ACB:           4,
		CFS:           7,
		CareIntensity: "palliative",
		AssessedAt:    time.Now(),
	}
	rules := oneRule("DOSE_CHANGE", "HIGH")

	pkt, err := generator.Generate(snap, rules, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := pkt.Sections[template.SectionClinicalCtx]
	checks := []string{"48.7", "2.3", "7", "palliative"}
	for _, want := range checks {
		if !strings.Contains(ctx, want) {
			t.Errorf("ClinicalContext missing %q\ngot: %s", want, ctx)
		}
	}
}

// TestGenerate_UrgencySectionSet verifies that the urgency section in the packet
// matches the ApplicableRule's Urgency value.
func TestGenerate_UrgencySectionSet(t *testing.T) {
	snap := freshSnapshot()
	rules := oneRule("ADD", "ROUTINE")

	pkt, err := generator.Generate(snap, rules, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	urgencySection := pkt.Sections[template.SectionUrgency]
	if !strings.Contains(urgencySection, "ROUTINE") {
		t.Errorf("urgency section should contain %q, got: %q", "ROUTINE", urgencySection)
	}
}

// ---------------------------------------------------------------------------
// IsValidPacketType tests
// ---------------------------------------------------------------------------

func TestIsValidPacketType_Valid(t *testing.T) {
	valid := []string{"STOP", "MONITOR", "DOSE_CHANGE", "ADD"}
	for _, v := range valid {
		if !generator.IsValidPacketType(v) {
			t.Errorf("expected IsValidPacketType(%q) = true", v)
		}
	}
}

func TestIsValidPacketType_Invalid(t *testing.T) {
	invalid := []string{"", "stop", "Stop", "UNKNOWN", "HALT", "REVIEW"}
	for _, v := range invalid {
		if generator.IsValidPacketType(v) {
			t.Errorf("expected IsValidPacketType(%q) = false", v)
		}
	}
}
