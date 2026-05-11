package api

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/ethics/consent_extension"
	"github.com/cardiofit/shared/v2_substrate/ethics/vulnerability"

	"github.com/cardiofit/kb32/internal/capacity"
	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/reasoning"
)

// fakeCapacitySource is a controllable test double for capacity.CapacitySource.
// It mirrors the fake used in internal/capacity but lives in the api package
// so the pipeline-level tests stay self-contained.
type fakeCapacitySource struct {
	assessment    vulnerability.Assessment
	assessmentErr error
	consent       *consent_extension.RestrictivePracticeConsent
	consentErr    error
}

func (f *fakeCapacitySource) AssessmentFor(_ context.Context, _ uuid.UUID) (vulnerability.Assessment, error) {
	return f.assessment, f.assessmentErr
}

func (f *fakeCapacitySource) RestrictivePracticeConsentFor(_ context.Context, _ uuid.UUID,
	_ consent_extension.PracticeType) (*consent_extension.RestrictivePracticeConsent, error) {
	return f.consent, f.consentErr
}

// buildPassingPipelineNoGate constructs a pipeline that passes Stages 1–6 in
// the no-capacity-gate baseline. Used as the regression smoke fixture so a
// nil gate behaves identically to pre-Phase-3-Task-3 callers.
func buildPassingPipelineNoGate(t *testing.T) *Pipeline {
	t.Helper()
	snapClient := &fakeSubstrateClient{snap: standardPassingSnap()}
	reasoningSrc := &fakeReasoningSource{result: &reasoning.EvaluateRuleResult{
		Triggered: true, Type: "MONITOR", Urgency: "red",
	}}
	appSrc := &fakeAppropriatenessSource{assessment: standardPassingAssessment()}
	assembler := kb32ctx.NewAssembler(snapClient)
	chain := reasoning.NewChainBuilder(reasoningSrc)
	return NewPipeline(assembler, chain, appSrc, nil)
}

// TestPipeline_NoCapacityGate_RegressionSmoke verifies that constructing a
// Pipeline without WithCapacityGate behaves identically to before — the
// recommendation drafts cleanly with a non-empty ContentHash.
func TestPipeline_NoCapacityGate_RegressionSmoke(t *testing.T) {
	pipeline := buildPassingPipelineNoGate(t)

	result, err := pipeline.Run(context.Background(), "TestRule", uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}
	if result.HoldReason != "" {
		t.Errorf("expected gate-free pipeline to draft cleanly; HoldReason=%q", result.HoldReason)
	}
	if result.ContentHash == "" {
		t.Error("expected non-empty ContentHash on drafted recommendation")
	}
}

// TestPipeline_CapacityGate_HoldsOnSDMRequired verifies that when the gate
// returns ErrSDMRequired, the pipeline holds with the appropriate HoldReason
// and never reaches Stage 5 (ContentHash remains empty).
func TestPipeline_CapacityGate_HoldsOnSDMRequired(t *testing.T) {
	pipeline := buildPassingPipelineNoGate(t)

	src := &fakeCapacitySource{
		assessment: vulnerability.Assessment{
			CognitiveCapacity: vulnerability.CapacityUncertain,
			SDMRequired:       false,
		},
	}
	pipeline.WithCapacityGate(capacity.NewGate(src))

	result, err := pipeline.Run(context.Background(), "TestRule", uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}
	if !strings.Contains(result.HoldReason, "capacity/consent hold") {
		t.Errorf("expected HoldReason to contain 'capacity/consent hold'; got %q", result.HoldReason)
	}
	if !strings.Contains(result.HoldReason, "SDM workflow required") {
		t.Errorf("expected HoldReason to contain 'SDM workflow required'; got %q", result.HoldReason)
	}
	if result.ContentHash != "" {
		t.Errorf("expected ContentHash empty on Stage 3.5 hold; got %q", result.ContentHash)
	}
}

// TestPipeline_CapacityGate_ProceedsWhenGatePasses verifies that a gate
// returning nil lets the pipeline continue through Stages 4–6 to a non-empty
// ContentHash.
func TestPipeline_CapacityGate_ProceedsWhenGatePasses(t *testing.T) {
	pipeline := buildPassingPipelineNoGate(t)

	src := &fakeCapacitySource{
		assessment: vulnerability.Assessment{
			CognitiveCapacity: vulnerability.CapacityIntact,
		},
	}
	pipeline.WithCapacityGate(capacity.NewGate(src))

	result, err := pipeline.Run(context.Background(), "TestRule", uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}
	if result.HoldReason != "" {
		t.Errorf("expected gate to pass; HoldReason=%q", result.HoldReason)
	}
	if result.ContentHash == "" {
		t.Error("expected non-empty ContentHash when gate passes")
	}
}
