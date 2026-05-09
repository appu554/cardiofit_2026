// Package api — explain_handlers.go implements the Layer 4 deep-audit endpoint
// GET /v1/explain/:decision_id, returning the full audit trail for a single
// algorithmic decision (Ethical Architecture Guidelines Principle 6 / §13.2
// reviewability).
//
// Permission middleware (AD auditor class restriction) is intentionally NOT
// mounted here. The route is reachable to any caller in Phase 3; AD-class
// gating is deferred to Phase 2-completion / Phase 4 once the PDP wrapper is
// extended to cover the explain surface alongside /v1/craft.
package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/citations"
	"github.com/cardiofit/shared/v2_substrate/ethics/decision_metadata"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// EvidenceTraceLinker is a minimal port for fetching nodes linked to a
// decision in the evidence-trace graph. Implementations may walk forward,
// backward, or both — combining their union into a single slice. Returning
// an empty slice is a valid no-op while the production traversal wiring is
// pending.
//
// The substrate package shared/v2_substrate/evidence_trace exposes
// TraceForward / TraceBackward primitives keyed on EdgeStore; the production
// adapter that combines them and seeds the start node from a decision's
// audit-trace ref will land alongside the explain endpoint's full Phase 4
// productionisation.
type EvidenceTraceLinker interface {
	LinkedNodes(ctx context.Context, decisionID uuid.UUID, depth int) ([]uuid.UUID, error)
}

// NoOpEvidenceTraceLinker returns no linked nodes. It is the default
// implementation injected when no real EdgeStore-backed adapter is available
// (Phase 3 ship state). Phase 4 wires the real one.
type NoOpEvidenceTraceLinker struct{}

// LinkedNodes always returns (nil, nil).
func (NoOpEvidenceTraceLinker) LinkedNodes(_ context.Context, _ uuid.UUID, _ int) ([]uuid.UUID, error) {
	return nil, nil
}

// ExplainResponse is the JSON body returned by GET /v1/explain/:decision_id.
//
// Note on Citations: the v2_substrate decision_metadata.Metadata struct does
// not currently carry a recommendation-ID linkage (see recorder.go — fields
// are DecisionID, Component, DecisionType, AffectedSubjectID/Class,
// PrinciplesImplicated, ERMReviewed, ERMOutcome, ContestationEnabled,
// AuditTraceRef, Timestamp). Because citations.Registry.ListCitations is keyed
// on recommendation-ID rather than decision-ID, this endpoint cannot resolve
// citations from a decision_id alone with the current substrate. The field is
// returned as an empty slice; a future substrate change that adds a
// RecommendationID field on Metadata will let us wire the lookup without
// changing the response shape.
type ExplainResponse struct {
	DecisionID  uuid.UUID                          `json:"decision_id"`
	Metadata    *decision_metadata.Metadata        `json:"metadata"`
	EthicsLog   []ethics_log.Entry                 `json:"ethics_log"`
	Citations   []citations.RecommendationCitation `json:"citations"`
	LinkedTrace []uuid.UUID                        `json:"linked_evidence_trace_nodes"`
}

// ExplainHandler serves the Layer 4 audit trail for a single decision.
type ExplainHandler struct {
	metadata    decision_metadata.Store
	log         ethics_log.Store
	citationReg citations.Registry
	linker      EvidenceTraceLinker
}

// NewExplainHandler constructs an ExplainHandler. If linker is nil, a
// NoOpEvidenceTraceLinker is substituted so the response always includes a
// valid (possibly empty) linked-nodes slice.
func NewExplainHandler(
	md decision_metadata.Store,
	log ethics_log.Store,
	reg citations.Registry,
	linker EvidenceTraceLinker,
) *ExplainHandler {
	if linker == nil {
		linker = NoOpEvidenceTraceLinker{}
	}
	return &ExplainHandler{metadata: md, log: log, citationReg: reg, linker: linker}
}

// HandleExplain serves GET /v1/explain/:decision_id.
//
// Response codes:
//   - 400 bad_decision_id   — :decision_id is not a parseable UUID.
//   - 404 decision_not_found — no Metadata record exists for the decision.
//   - 200                   — full audit trail (metadata + ethics log + citations + linked trace nodes).
//
// Errors from downstream stores (ethics log, citation registry, evidence-trace
// linker) degrade gracefully to empty slices rather than 500, so a partial
// audit view is preferred over an opaque server error.
func (h *ExplainHandler) HandleExplain(c *gin.Context) {
	idStr := c.Param("decision_id")
	decisionID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_decision_id"})
		return
	}

	ctx := c.Request.Context()

	md, err := h.metadata.Get(ctx, decisionID)
	if err != nil || md == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "decision_not_found"})
		return
	}

	// Ethics log entries — degrade to empty slice on error.
	entries := []ethics_log.Entry{}
	if h.log != nil {
		if e, err := ethics_log.NewQuerier(h.log).ByDecision(ctx, decisionID); err == nil && e != nil {
			entries = e
		}
	}

	// Citations — registry lookup is keyed on recommendation-ID, but the
	// current decision_metadata.Metadata struct does not carry a
	// recommendation-ID. Until the substrate is extended, return [].
	// See ExplainResponse type doc-comment for the full deferral note.
	cites := []citations.RecommendationCitation{}

	// Linked evidence-trace nodes — depth=5 per Phase 3 plan.
	linked := []uuid.UUID{}
	if h.linker != nil {
		if nodes, err := h.linker.LinkedNodes(ctx, decisionID, 5); err == nil && nodes != nil {
			linked = nodes
		}
	}

	c.JSON(http.StatusOK, ExplainResponse{
		DecisionID:  decisionID,
		Metadata:    md,
		EthicsLog:   entries,
		Citations:   cites,
		LinkedTrace: linked,
	})
}
