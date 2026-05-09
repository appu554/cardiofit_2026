// Package reasoning_test exercises the ChainBuilder.
package reasoning_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cardiofit/kb32/internal/reasoning"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// stubbedSource — test double for ReasoningSource
// ---------------------------------------------------------------------------

// stubbedSource maps ruleIDs to pre-configured outcomes (result or error).
// Unrecognised rule IDs return ErrCQLPlaceholderResponse (simulating a
// placeholder response from the Phase 0.5 engine).
type stubbedSource struct {
	results map[string]*reasoning.EvaluateRuleResult
	errs    map[string]error
}

func newStubbedSource() *stubbedSource {
	return &stubbedSource{
		results: make(map[string]*reasoning.EvaluateRuleResult),
		errs:    make(map[string]error),
	}
}

func (s *stubbedSource) withResult(ruleID string, r *reasoning.EvaluateRuleResult) *stubbedSource {
	s.results[ruleID] = r
	return s
}

func (s *stubbedSource) withError(ruleID string, err error) *stubbedSource {
	s.errs[ruleID] = err
	return s
}

func (s *stubbedSource) EvaluateRule(_ context.Context, ruleID string, _ uuid.UUID) (*reasoning.EvaluateRuleResult, error) {
	if err, ok := s.errs[ruleID]; ok {
		return nil, err
	}
	if r, ok := s.results[ruleID]; ok {
		return r, nil
	}
	// Default: placeholder response (non-configured rules skip cleanly).
	return nil, reasoning.ErrCQLPlaceholderResponse
}

// ---------------------------------------------------------------------------
// ChainBuilder tests
// ---------------------------------------------------------------------------

// TestChainBuilder_HappyPath — one rule fires → one ApplicableRule returned.
func TestChainBuilder_HappyPath(t *testing.T) {
	residentID := uuid.New()
	src := newStubbedSource().withResult("EGFR-001", &reasoning.EvaluateRuleResult{
		RuleID:    "EGFR-001",
		Triggered: true,
		Type:      "STOP",
		Urgency:   "HIGH",
	})

	cb := reasoning.NewChainBuilder(src)
	rules, err := cb.Build(context.Background(), residentID, []string{"EGFR-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].RuleID != "EGFR-001" {
		t.Errorf("RuleID: got %q want %q", rules[0].RuleID, "EGFR-001")
	}
	if rules[0].Type != "STOP" {
		t.Errorf("Type: got %q want %q", rules[0].Type, "STOP")
	}
	if rules[0].Urgency != "HIGH" {
		t.Errorf("Urgency: got %q want %q", rules[0].Urgency, "HIGH")
	}
}

// TestChainBuilder_PlaceholderSkipped — placeholder response causes skip, not error.
func TestChainBuilder_PlaceholderSkipped(t *testing.T) {
	residentID := uuid.New()
	src := newStubbedSource() // all rules return placeholder by default

	cb := reasoning.NewChainBuilder(src)
	rules, err := cb.Build(context.Background(), residentID, []string{"PLACEHOLDER-001"})
	if err != nil {
		t.Fatalf("unexpected error (placeholder should be skipped): %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules (placeholder skipped), got %d", len(rules))
	}
}

// TestChainBuilder_NotTriggeredExcluded — a real (non-placeholder) rule with
// triggered=false must NOT appear in the output.
func TestChainBuilder_NotTriggeredExcluded(t *testing.T) {
	residentID := uuid.New()
	src := newStubbedSource().withResult("DBI-001", &reasoning.EvaluateRuleResult{
		RuleID:    "DBI-001",
		Triggered: false,
		Type:      "MONITOR",
		Urgency:   "LOW",
	})

	cb := reasoning.NewChainBuilder(src)
	rules, err := cb.Build(context.Background(), residentID, []string{"DBI-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules (not triggered), got %d", len(rules))
	}
}

// TestChainBuilder_PropagatesNonPlaceholderError — a real error (not placeholder)
// is returned from Build.
func TestChainBuilder_PropagatesNonPlaceholderError(t *testing.T) {
	residentID := uuid.New()
	sentinel := errors.New("network failure")
	src := newStubbedSource().withError("NET-ERR-001", sentinel)

	cb := reasoning.NewChainBuilder(src)
	_, err := cb.Build(context.Background(), residentID, []string{"NET-ERR-001"})
	if err == nil {
		t.Fatal("expected error to be propagated, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got: %v", err)
	}
}

// TestChainBuilder_ContextCancellation — a pre-cancelled context causes Build
// to return immediately without processing candidates.
func TestChainBuilder_ContextCancellation(t *testing.T) {
	residentID := uuid.New()
	src := newStubbedSource().withResult("EGFR-001", &reasoning.EvaluateRuleResult{
		RuleID:    "EGFR-001",
		Triggered: true,
		Type:      "STOP",
		Urgency:   "HIGH",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before Build runs

	cb := reasoning.NewChainBuilder(src)
	_, err := cb.Build(ctx, residentID, []string{"EGFR-001"})
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// TestChainBuilder_NoCandidates — empty candidates slice returns empty output, no error.
func TestChainBuilder_NoCandidates(t *testing.T) {
	src := newStubbedSource()
	cb := reasoning.NewChainBuilder(src)
	rules, err := cb.Build(context.Background(), uuid.New(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

// TestChainBuilder_PartialPlaceholderMixed verifies the critical mixed scenario:
//   - rule A: real, triggered=true  → included
//   - rule B: placeholder           → skipped (no error)
//   - rule C: real, triggered=false → excluded
//
// Output must contain exactly rule A.
func TestChainBuilder_PartialPlaceholderMixed(t *testing.T) {
	residentID := uuid.New()
	src := newStubbedSource().
		withResult("RULE-A", &reasoning.EvaluateRuleResult{
			RuleID:    "RULE-A",
			Triggered: true,
			Type:      "STOP",
			Urgency:   "HIGH",
		}).
		// RULE-B is not configured → defaults to ErrCQLPlaceholderResponse.
		withResult("RULE-C", &reasoning.EvaluateRuleResult{
			RuleID:    "RULE-C",
			Triggered: false,
			Type:      "MONITOR",
			Urgency:   "ROUTINE",
		})

	cb := reasoning.NewChainBuilder(src)
	rules, err := cb.Build(context.Background(), residentID, []string{"RULE-A", "RULE-B", "RULE-C"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d: %v", len(rules), rules)
	}
	if rules[0].RuleID != "RULE-A" {
		t.Errorf("expected RULE-A, got %q", rules[0].RuleID)
	}
}
