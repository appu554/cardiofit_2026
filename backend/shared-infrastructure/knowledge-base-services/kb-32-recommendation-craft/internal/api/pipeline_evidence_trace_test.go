package api

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/ethics/vulnerability"

	"github.com/cardiofit/kb32/internal/appropriateness"
	"github.com/cardiofit/kb32/internal/capacity"
	"github.com/cardiofit/kb32/internal/lifecycle"
	"github.com/cardiofit/kb32/internal/reasoning"
)

// These tests close the Phase 2-completion Task 4 commitment: the Pipeline
// must emit a Stage 7 DraftedTransitionEntry on every successful
// detected → drafted transition, and MUST NOT emit when a hold path returns
// early (capacity gate hold, appropriateness gate hold).

// captureEmitter is a controllable EvidenceTraceEmitter test double scoped to
// the pipeline package so we can assert end-to-end pipeline behaviour without
// dragging the lifecycle test fixtures across packages.
type captureEmitter struct {
	calls   int
	last    lifecycle.DraftedTransitionEntry
	failErr error
}

func (c *captureEmitter) EmitDraftedTransition(_ context.Context, entry lifecycle.DraftedTransitionEntry) error {
	c.calls++
	c.last = entry
	return c.failErr
}

// TestPipeline_EvidenceTrace_EmittedOnSuccess verifies that the success-only
// path of Pipeline.Run invokes the EvidenceTraceEmitter exactly once and
// populates the entry with the rule ID, urgency, and assessment that drove
// the recommendation.
func TestPipeline_EvidenceTrace_EmittedOnSuccess(t *testing.T) {
	pipeline := buildPassingPipelineNoGate(t)
	emitter := &captureEmitter{}
	pipeline.WithEvidenceTracer(emitter)

	authorID := uuid.New()
	result, err := pipeline.Run(context.Background(), "TestRule", uuid.New(), authorID)
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}
	if result.HoldReason != "" {
		t.Fatalf("expected success path; got HoldReason=%q", result.HoldReason)
	}
	if emitter.calls != 1 {
		t.Fatalf("expected exactly 1 emission on success; got %d", emitter.calls)
	}
	if emitter.last.RuleID != "TestRule" {
		t.Errorf("emitted RuleID: got %q want %q", emitter.last.RuleID, "TestRule")
	}
	if emitter.last.AuthorID != authorID {
		t.Errorf("emitted AuthorID: got %s want %s", emitter.last.AuthorID, authorID)
	}
	if emitter.last.ContentHash == "" {
		t.Error("expected non-empty ContentHash in emitted entry")
	}
	if emitter.last.Urgency == "" {
		t.Error("expected non-empty Urgency in emitted entry")
	}
	if emitter.last.FiredAt.IsZero() {
		t.Error("expected non-zero FiredAt in emitted entry")
	}
}

// TestPipeline_EvidenceTrace_NotEmittedOnCapacityHold verifies that when
// Stage 3.5 holds the recommendation, the success-only Stage 7 emission is
// skipped — the recommendation is still in `detected` state and must not be
// recorded as drafted in the audit ledger.
func TestPipeline_EvidenceTrace_NotEmittedOnCapacityHold(t *testing.T) {
	pipeline := buildPassingPipelineNoGate(t)

	// Force a capacity hold via the same fixture used in
	// TestPipeline_CapacityGate_HoldsOnSDMRequired.
	src := &fakeCapacitySource{
		assessment: vulnerability.Assessment{
			CognitiveCapacity: vulnerability.CapacityUncertain,
			SDMRequired:       false,
		},
	}
	pipeline.WithCapacityGate(capacity.NewGate(src))

	emitter := &captureEmitter{}
	pipeline.WithEvidenceTracer(emitter)

	result, err := pipeline.Run(context.Background(), "TestRule", uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}
	if result.HoldReason == "" {
		t.Fatal("expected capacity hold; got empty HoldReason")
	}
	if emitter.calls != 0 {
		t.Errorf("expected zero emissions on hold path; got %d", emitter.calls)
	}
}

// TestPipeline_EvidenceTrace_NotEmittedOnAppropriatenessHold verifies that
// when Stage 4 holds the recommendation (any dimension ≤ HoldThreshold), the
// Stage 7 emission is skipped.
func TestPipeline_EvidenceTrace_NotEmittedOnAppropriatenessHold(t *testing.T) {
	// Build a pipeline whose appropriateness source returns a holding score.
	snapClient := &fakeSubstrateClient{snap: standardPassingSnap()}
	reasoningSrc := &fakeReasoningSource{result: &reasoning.EvaluateRuleResult{
		Triggered: true, Type: "MONITOR", Urgency: "red",
	}}
	holdAssessment := appropriateness.Assessment{
		ClinicalWarrant:        1, // ≤ HoldThreshold → hold
		EvidenceSolidity:       3,
		AlternativesConsidered: 3,
		RestraintConsidered:    3,
		GoalsOfCareAlignment:   3,
	}
	appSrc := &fakeAppropriatenessSource{assessment: holdAssessment}
	pipeline := buildTestPipeline(snapClient, reasoningSrc, appSrc)

	emitter := &captureEmitter{}
	pipeline.WithEvidenceTracer(emitter)

	result, err := pipeline.Run(context.Background(), "TestRule", uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}
	if result.HoldReason == "" {
		t.Fatal("expected appropriateness hold; got empty HoldReason")
	}
	if emitter.calls != 0 {
		t.Errorf("expected zero emissions on appropriateness hold; got %d", emitter.calls)
	}
}

// TestPipeline_EvidenceTrace_EmitterFailureFailsRun verifies the fail-hard
// contract: a Stage 7 emission error must surface as a pipeline error, not
// be swallowed. A best-effort audit trail is not an audit trail.
func TestPipeline_EvidenceTrace_EmitterFailureFailsRun(t *testing.T) {
	pipeline := buildPassingPipelineNoGate(t)
	boom := errors.New("synthetic ledger failure")
	emitter := &captureEmitter{failErr: boom}
	pipeline.WithEvidenceTracer(emitter)

	_, err := pipeline.Run(context.Background(), "TestRule", uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected emitter failure to fail pipeline.Run; got nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("expected wrapped boom error; got %v", err)
	}
}

// TestPipeline_NoEvidenceTracerIsNoop verifies backward compatibility: a
// Pipeline constructed without WithEvidenceTracer still drafts cleanly.
func TestPipeline_NoEvidenceTracerIsNoop(t *testing.T) {
	pipeline := buildPassingPipelineNoGate(t)

	result, err := pipeline.Run(context.Background(), "TestRule", uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("pipeline.Run with nil tracer: %v", err)
	}
	if result.HoldReason != "" {
		t.Errorf("expected success; got HoldReason=%q", result.HoldReason)
	}
}
