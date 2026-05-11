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
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/citations"
	"github.com/cardiofit/shared/v2_substrate/ethics/decision_metadata"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// LinkedTraceDepth is the maximum traversal depth requested from the
// EvidenceTraceLinker. Set per Phase 3 (tightened) plan Task 4.
const LinkedTraceDepth = 5

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

// TODO(phase-2-completion / phase-4): mount with AD-class auditor permission
// middleware before pilot. Until then, the route is authenticated only by the
// kb-32 service-level auth (no auditor-class scope check).
//
// HandleExplain serves GET /v1/explain/:decision_id.
//
// Response codes:
//   - 400 bad_decision_id        — :decision_id is not a parseable UUID.
//   - 404 decision_not_found     — no Metadata record exists for the decision.
//   - 500 metadata_lookup_failed — substrate failure during metadata.Get;
//     distinguishes a real server error from a legitimate "never existed"
//     so auditors aren't misled when the DB is temporarily unavailable.
//   - 200                        — full audit trail (metadata + ethics log + citations + linked trace nodes).
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
	if err != nil {
		log.Printf("explain: metadata.Get failed for decision_id=%s: %v", decisionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "metadata_lookup_failed"})
		return
	}
	if md == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "decision_not_found"})
		return
	}

	// Ethics log entries — degrade to empty slice on error, but log
	// server-side so operators can detect substrate problems even when the
	// caller-facing response stays 200 with a partial payload.
	entries := []ethics_log.Entry{}
	if h.log != nil {
		e, err := ethics_log.NewQuerier(h.log).ByDecision(ctx, decisionID)
		if err != nil {
			log.Printf("explain: ethics_log.ByDecision failed for decision_id=%s: %v", decisionID, err)
			entries = []ethics_log.Entry{}
		} else if e != nil {
			entries = e
		}
	}

	// Citations — registry lookup is keyed on recommendation-ID, but the
	// current decision_metadata.Metadata struct does not carry a
	// recommendation-ID. Until the substrate is extended, return [].
	// See ExplainResponse type doc-comment for the full deferral note.
	//
	// The log line below is staged here so it activates the moment a future
	// Metadata.RecommendationID field unblocks the ListCitations call:
	//
	//   cites, err := h.citationReg.ListCitations(ctx, md.RecommendationID)
	//   if err != nil {
	//       log.Printf("explain: citations.ListCitations failed for decision_id=%s: %v", decisionID, err)
	//       cites = []citations.RecommendationCitation{}
	//   }
	cites := []citations.RecommendationCitation{}

	// Linked evidence-trace nodes — depth per Phase 3 plan.
	linked := []uuid.UUID{}
	if h.linker != nil {
		nodes, err := h.linker.LinkedNodes(ctx, decisionID, LinkedTraceDepth)
		if err != nil {
			log.Printf("explain: linker.LinkedNodes failed for decision_id=%s: %v", decisionID, err)
			linked = []uuid.UUID{}
		} else if nodes != nil {
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
