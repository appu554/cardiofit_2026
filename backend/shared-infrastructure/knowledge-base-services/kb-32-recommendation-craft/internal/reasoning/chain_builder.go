// Package reasoning implements Stage 2 of the six-stage rendering pipeline.
// See hapi_client.go for the package-level doc comment.
package reasoning

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// ApplicableRule
// ---------------------------------------------------------------------------

// ApplicableRule represents a rule that fired (Triggered=true) for a given
// resident and is carried into Stage 3 (generator) for packet construction.
type ApplicableRule struct {
	// RuleID is the canonical rule identifier from kb-cql-runtime.
	RuleID string

	// Type is the recommendation type (e.g. "STOP", "MONITOR", "DOSE_CHANGE").
	Type string

	// Urgency is the urgency tier (e.g. "HIGH", "ROUTINE").
	Urgency string
}

// ---------------------------------------------------------------------------
// ReasoningSource interface
// ---------------------------------------------------------------------------

// ReasoningSource is the port through which ChainBuilder invokes CQL rule
// evaluation. HAPIClient implements this interface for production use; tests
// use in-process stubs.
type ReasoningSource interface {
	EvaluateRule(ctx context.Context, ruleID string, residentID uuid.UUID) (*EvaluateRuleResult, error)
}

// ---------------------------------------------------------------------------
// ChainBuilder
// ---------------------------------------------------------------------------

// ChainBuilder walks a slice of candidate rule IDs, evaluates each via a
// ReasoningSource, and returns the subset that fired (Triggered=true).
//
// Placeholder responses (ErrCQLPlaceholderResponse) are silently skipped,
// supporting Phase 0.5 deployment while the CQF-FHIR-CR engine wiring is
// deferred. Any other error is propagated immediately.
type ChainBuilder struct {
	src ReasoningSource
}

// NewChainBuilder constructs a ChainBuilder backed by the given ReasoningSource.
func NewChainBuilder(src ReasoningSource) *ChainBuilder {
	return &ChainBuilder{src: src}
}

// Build evaluates each rule in candidates for the given residentID and returns
// the ApplicableRule slice for rules that triggered.
//
// Behaviour by result:
//   - ErrCQLPlaceholderResponse  → continue (non-firing; Phase 0.5 deferral)
//   - other error                → propagate immediately
//   - result.Triggered == false  → exclude from output
//   - result.Triggered == true   → append ApplicableRule to output
//
// Build checks ctx.Err() before entering the candidate loop. Context
// cancellation inside the loop is surfaced by the ReasoningSource's
// EvaluateRule call.
func (cb *ChainBuilder) Build(ctx context.Context, residentID uuid.UUID, candidates []string) ([]ApplicableRule, error) {
	// Honour pre-cancelled contexts before any I/O.
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("reasoning: context cancelled before chain build: %w", err)
	}

	var out []ApplicableRule
	for _, ruleID := range candidates {
		result, err := cb.src.EvaluateRule(ctx, ruleID, residentID)
		if err != nil {
			// Phase 0.5 placeholder: engine is pending; skip gracefully.
			if errors.Is(err, ErrCQLPlaceholderResponse) {
				continue
			}
			return nil, fmt.Errorf("reasoning: evaluate rule %q: %w", ruleID, err)
		}
		if result.Triggered {
			out = append(out, ApplicableRule{
				RuleID:  result.RuleID,
				Type:    result.Type,
				Urgency: result.Urgency,
			})
		}
	}
	return out, nil
}
