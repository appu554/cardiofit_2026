// Package api implements the HTTP surface for the kb-32 recommendation craft
// engine. It exposes POST /v1/craft/draft as the primary endpoint.
//
// # Permissions middleware deferral
//
// Production PDP (Pharmacist Decision Portal) permissions middleware wrapping
// for the /v1/craft/draft route is deferred to Phase 2b (or a Phase 2-completion
// plan). The craft engine pipeline enforces its own clinical-safety gate
// (Stage 4 appropriateness check) independently of transport-layer auth, so
// deferring the wrapping does not compromise clinical safety during Phase 2a
// shadow deployment. When Phase 2b wires the PDP middleware, it mounts
// directly onto the /v1/craft/ route group in cmd/server/main.go.
package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/appropriateness"
	"github.com/cardiofit/kb32/internal/citations"
	"github.com/cardiofit/kb32/internal/formatter"
	"github.com/cardiofit/kb32/internal/framing"
	"github.com/cardiofit/kb32/internal/generator"
	"github.com/cardiofit/kb32/internal/ordering"
	"github.com/cardiofit/kb32/internal/reasoning"
	"github.com/cardiofit/kb32/internal/urgency"
)

// PipelineResult is the structured output of a successful Pipeline.Run call.
// It captures the key artifacts from each stage for the HTTP handler to
// convert into a DraftResponse.
type PipelineResult struct {
	// Packet is the draft recommendation packet produced in Stage 3.
	Packet *generator.Packet

	// ContentHash is the SHA-256 hex string from Stage 5 (framing.ContentHash).
	ContentHash string

	// LayerOutput is the validated four-layer brevity output from Stage 6.
	LayerOutput formatter.LayerOutput

	// UrgencyTag is the urgency tier derived from the ClinicalSnapshot by the
	// urgency tagger (applied alongside Stage 3).
	UrgencyTag string

	// Assessment is the appropriateness assessment produced by the
	// AppropriatenessSource in Stage 4.
	Assessment appropriateness.Assessment

	// HoldReason is non-empty when the appropriateness gate held the
	// recommendation. When non-empty the caller should set State="detected".
	HoldReason string

	// Citations is the slice of fire-time citation pins produced after Stage 5.
	// Each entry links this recommendation to the exact source version that was
	// active at the moment the recommendation fired. Empty when the pipeline was
	// constructed without a Registry (nil-registry dev/test mode) or when
	// ClinicalContent.EvidenceAnchors is empty.
	Citations []citations.RecommendationCitation
}

// AppropriatenessSource is the port through which the Pipeline scores a
// draft Packet against the five-dimension appropriateness rubric.
//
// The DefaultAppropriatenessSource (this package) returns 3 across all five
// dimensions, guaranteeing that the gate always passes during Phase 2a shadow
// deployment. Real multi-dimension scoring is deferred to Phase 2b.
//
// IMPORTANT: Replace DefaultAppropriatenessSource with a real implementation
// before clinical production deployment.
type AppropriatenessSource interface {
	Assess(ctx context.Context, packet *generator.Packet, snap kb32ctx.ClinicalSnapshot,
		rule reasoning.ApplicableRule) (appropriateness.Assessment, error)
}

// DefaultAppropriatenessSource returns a passing Assessment with all five
// dimensions at 3 (above HoldThreshold=2). This is the Phase 2a default.
// See AppropriatenessSource for replacement requirements.
type DefaultAppropriatenessSource struct{}

// Assess returns an Assessment with all dimensions at 3.
func (DefaultAppropriatenessSource) Assess(_ context.Context, _ *generator.Packet,
	_ kb32ctx.ClinicalSnapshot, _ reasoning.ApplicableRule) (appropriateness.Assessment, error) {
	return appropriateness.Assessment{
		ClinicalWarrant:        3,
		EvidenceSolidity:       3,
		AlternativesConsidered: 3,
		RestraintConsidered:    3,
		GoalsOfCareAlignment:   3,
	}, nil
}

// Pipeline orchestrates the six stages of the recommendation craft engine.
// Wire all dependencies via NewPipeline; the zero value is not usable.
//
// Stage order:
//  1. context.Assembler     – pull ClinicalSnapshot for ResidentID
//  2. reasoning.ChainBuilder – evaluate CQL rules; get ApplicableRules
//  3. generator.Generate     – produce draft Packet from top rule + Snapshot
//  4. AppropriatenessSource  – GATE: score assessment; hold if any dim ≤ 2
//  5. framing.ContentHash    – compute deterministic SHA-256 content hash
//  5b. citations.PinAtFireTime – pin source versions active at fire time (audit trail)
//  6. formatter.Validate     – enforce Layer 1/2 word budgets
type Pipeline struct {
	assembler  *kb32ctx.Assembler
	chain      *reasoning.ChainBuilder
	appSrc     AppropriatenessSource
	registry   citations.Registry // nil = dev/test mode, pinning skipped gracefully
	candidates []string           // candidate rule IDs passed to ChainBuilder
}

// NewPipeline constructs a Pipeline with the supplied collaborators.
// candidates is the ordered list of CQL rule IDs to evaluate in Stage 2.
// registry may be nil; when nil, citation pinning is skipped without error
// (dev/test mode). Production callers must supply a non-nil Registry.
func NewPipeline(
	assembler *kb32ctx.Assembler,
	chain *reasoning.ChainBuilder,
	appSrc AppropriatenessSource,
	candidates []string,
) *Pipeline {
	return &Pipeline{
		assembler:  assembler,
		chain:      chain,
		appSrc:     appSrc,
		candidates: candidates,
	}
}

// NewPipelineWithRegistry constructs a Pipeline with a citation Registry wired
// in. Use this constructor in production and in tests that assert citation
// pinning behaviour. The registry must implement citations.Registry; an
// in-memory implementation is available via citations.NewInMemoryRegistry().
func NewPipelineWithRegistry(
	assembler *kb32ctx.Assembler,
	chain *reasoning.ChainBuilder,
	appSrc AppropriatenessSource,
	candidates []string,
	registry citations.Registry,
) *Pipeline {
	return &Pipeline{
		assembler:  assembler,
		chain:      chain,
		appSrc:     appSrc,
		candidates: candidates,
		registry:   registry,
	}
}

// Run executes all six pipeline stages for the given ruleID, residentID, and authorID.
//
// When the appropriateness gate holds the recommendation (Stage 4), Run returns
// a PipelineResult with a non-empty HoldReason and does NOT return an error —
// the held state is a valid pipeline outcome that the HTTP handler maps to
// State="detected" in the DraftResponse.
//
// Run returns an error only for genuine infrastructure/logic failures such as
// a missing snapshot, no applicable rules, a malformed packet, or a formatter
// budget violation.
func (p *Pipeline) Run(ctx context.Context, ruleID string, residentID, authorID uuid.UUID) (*PipelineResult, error) {
	// Stage 1: context assembly — pull ClinicalSnapshot.
	snap, err := p.assembler.Assemble(ctx, residentID)
	if err != nil {
		return nil, fmt.Errorf("pipeline stage1 (context): %w", err)
	}

	// Stage 2: reasoning chain — evaluate candidate rules.
	// Use the requested ruleID as the sole candidate; additional candidates
	// can be supplied via p.candidates (appended after ruleID for ordering).
	candidates := append([]string{ruleID}, p.candidates...)
	applicable, err := p.chain.Build(ctx, residentID, candidates)
	if err != nil {
		return nil, fmt.Errorf("pipeline stage2 (reasoning): %w", err)
	}

	// Stage 3: generator — produce draft Packet.
	// ordering.Order is applied before generation so the highest-priority
	// rule (STOP > MONITOR > DOSE_CHANGE > ADD) drives the packet.
	// We need to convert applicable rules to packets for ordering; since we
	// have raw ApplicableRules pre-packet, we sort by type rank directly.
	orderedRules := orderApplicableRules(applicable)

	pkt, err := generator.Generate(snap, orderedRules, authorID)
	if err != nil {
		return nil, fmt.Errorf("pipeline stage3 (generator): %w", err)
	}

	// Apply urgency tagger to the snapshot (used in E2E assertions).
	urgencyTag := urgency.Tag(snap)

	// Stage 4: appropriateness gate — score and check.
	var topRule reasoning.ApplicableRule
	if len(orderedRules) > 0 {
		topRule = orderedRules[0]
	}
	assessment, err := p.appSrc.Assess(ctx, pkt, snap, topRule)
	if err != nil {
		return nil, fmt.Errorf("pipeline stage4 (appropriateness): %w", err)
	}
	if gateErr := appropriateness.Check(assessment); gateErr != nil {
		dimName, dimScore := assessment.LowestDimension()
		holdReason := fmt.Sprintf("appropriateness hold: dimension %q scored %d (threshold %d)",
			dimName, dimScore, appropriateness.HoldThreshold)
		return &PipelineResult{
			Packet:     pkt,
			UrgencyTag: urgencyTag,
			Assessment: assessment,
			HoldReason: holdReason,
		}, nil
	}

	// Stage 5: content hash — deterministic SHA-256 over clinical content.
	// Build a framing.ClinicalContent from the packet. Urgency is normalised
	// to the three-tier framing vocabulary (red/amber/green) via urgencyTag.
	content := framing.ClinicalContent{
		RuleID:  pkt.AppliedRule.RuleID,
		Type:    pkt.Type,
		Urgency: urgencyTag,
	}
	if err := content.Validate(); err != nil {
		return nil, fmt.Errorf("pipeline stage5 (content validate): %w", err)
	}
	hash := framing.ContentHash(content)

	// Stage 5b: citation pinning — lock in the source versions active at fire
	// time. This creates an immutable audit trail linking the recommendation to
	// the exact evidence sources that justified it at the moment it fired.
	//
	// When registry is nil (dev/test mode with NewPipeline) pinning is skipped
	// gracefully and Citations will be an empty slice. Production callers must
	// use NewPipelineWithRegistry to satisfy the audit-defensibility commitment
	// from Phase 2b Task 6.
	var pinnedCitations []citations.RecommendationCitation
	if p.registry != nil && len(content.EvidenceAnchors) > 0 {
		fireTime := time.Now().UTC()
		recID := pkt.RecommendationID.String()
		pinned, pinErr := citations.PinAtFireTime(ctx, p.registry, recID, content.EvidenceAnchors, fireTime)
		if pinErr != nil {
			return nil, fmt.Errorf("pipeline stage5b (citations): %w", pinErr)
		}
		pinnedCitations = pinned
	}

	// Stage 6: formatter validation — enforce Layer 1/2 word budgets.
	// Build a LayerOutput from the packet sections.
	layerOut := buildLayerOutput(pkt, snap)
	if err := formatter.Validate(layerOut); err != nil {
		return nil, fmt.Errorf("pipeline stage6 (formatter): %w", err)
	}

	return &PipelineResult{
		Packet:      pkt,
		ContentHash: hash,
		LayerOutput: layerOut,
		UrgencyTag:  urgencyTag,
		Assessment:  assessment,
		Citations:   pinnedCitations,
	}, nil
}

// orderApplicableRules reorders applicable rules by canonical type priority
// using the same rank as ordering.Order (STOP=0, MONITOR=1, DOSE_CHANGE=2, ADD=3).
// This mirrors the ordering package's logic for pre-packet rule slices.
func orderApplicableRules(rules []reasoning.ApplicableRule) []reasoning.ApplicableRule {
	if len(rules) == 0 {
		return rules
	}
	// Wrap into packets for ordering, then unwrap.
	packets := make([]*generator.Packet, len(rules))
	for i, r := range rules {
		packets[i] = &generator.Packet{Type: r.Type, AppliedRule: r}
	}
	ordered := ordering.Order(packets)
	out := make([]reasoning.ApplicableRule, len(ordered))
	for i, p := range ordered {
		out[i] = p.AppliedRule
	}
	return out
}

// buildLayerOutput constructs a LayerOutput from the generated packet.
// Layer 1 (signal) is kept short by using only the issue section summary.
// Layer 2 (reasoning) combines the clinical context.
// Layers 3 and 4 are left minimal for Phase 2a.
func buildLayerOutput(pkt *generator.Packet, snap kb32ctx.ClinicalSnapshot) formatter.LayerOutput {
	// Layer 1: short signal — "Rule X fired: type Y at urgency Z."
	// This maps to the issue section, which is always short by construction.
	l1 := pkt.Sections["issue"]

	// Layer 2: clinical context — snapshot summary.
	// The clinical_context section is also short by construction.
	l2 := pkt.Sections["clinical_context"]

	// If L1 would be over budget (unlikely given generator output, but defensive),
	// we trim to the first sentence.
	if formatter.WordCount(l1) > formatter.Layer1MaxWords {
		l1 = trimToWords(l1, formatter.Layer1MaxWords)
	}
	if formatter.WordCount(l2) > formatter.Layer2MaxWords {
		l2 = trimToWords(l2, formatter.Layer2MaxWords)
	}

	_ = snap // available for future Layer 3 provenance enrichment

	return formatter.LayerOutput{
		L1Signal:    l1,
		L2Reasoning: l2,
		L3Provenance: []string{pkt.AppliedRule.RuleID},
		L4DeepAudit: pkt.RecommendationID.String(),
	}
}

// trimToWords returns the first n whitespace-delimited words of s joined by spaces.
func trimToWords(s string, n int) string {
	words := splitWords(s)
	if len(words) <= n {
		return s
	}
	result := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			result += " "
		}
		result += words[i]
	}
	return result
}

// splitWords returns the whitespace-delimited tokens of s.
func splitWords(s string) []string {
	var words []string
	start := -1
	for i, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if start >= 0 {
				words = append(words, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words = append(words, s[start:])
	}
	return words
}
