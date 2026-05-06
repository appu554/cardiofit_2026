package api

// Identity matching REST handlers (Wave 1R.3) — Layer 2 doc §3.3.
//
// Routes are mounted by main.go onto a separate /v2/identity group so
// the IdentityMatcher ships non-breakingly alongside the existing
// /api/v1 + /v2 substrate routes:
//
//   v2idHandlers := api.NewIdentityHandlers(idStore)
//   v2idHandlers.RegisterRoutes(httpServer.Router.Group("/v2/identity"))
//
// Endpoints:
//   POST /v2/identity/match               — IncomingIdentifier -> MatchResult
//   GET  /v2/identity/review-queue        — list queue entries (paginated)
//   POST /v2/identity/review/:id/resolve  — promote a queue entry to mapping

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/interfaces"

	"kb-patient-profile/internal/storage"
)

// IdentityHandlers serves the v2 identity-matching REST endpoints.
type IdentityHandlers struct {
	store *storage.IdentityStore
}

// NewIdentityHandlers constructs a handler set bound to the given store.
func NewIdentityHandlers(store *storage.IdentityStore) *IdentityHandlers {
	return &IdentityHandlers{store: store}
}

// RegisterRoutes wires the identity endpoints onto the given router
// group. Caller mounts the group at "/v2/identity".
func (h *IdentityHandlers) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/match", h.matchIdentity)
	g.GET("/review-queue", h.listReviewQueue)
	g.POST("/review/:id/resolve", h.resolveReview)
}

// matchResponse is the JSON wire format returned by POST /match. It
// surfaces both the matcher's MatchResult and the audit cross-refs
// that the service layer wrote alongside it.
type matchResponse struct {
	Match                identity.MatchResult `json:"match"`
	EvidenceTraceNodeRef uuid.UUID            `json:"evidence_trace_node_ref"`
	ReviewQueueEntryID   *uuid.UUID           `json:"review_queue_entry_id,omitempty"`
}

func (h *IdentityHandlers) matchIdentity(c *gin.Context) {
	var in identity.IncomingIdentifier
	if err := c.BindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.store.MatchAndPersist(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, matchResponse{
		Match:                res.Match,
		EvidenceTraceNodeRef: res.EvidenceTraceNodeRef,
		ReviewQueueEntryID:   res.ReviewQueueEntryID,
	})
}

func (h *IdentityHandlers) listReviewQueue(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	if status != "" && status != "pending" && status != "resolved" && status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be one of pending|resolved|rejected (or empty for all)"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit <= 0 || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be in (0, 1000]"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}
	out, err := h.store.ListIdentityReviewQueue(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if out == nil {
		out = []interfaces.IdentityReviewQueueEntry{}
	}
	c.JSON(http.StatusOK, out)
}

// resolveRequest is the body of POST /review/:id/resolve. resolved_resident_ref
// must be a non-zero UUID — rejection of the queue entry uses a separate
// path (not exposed here yet) so the handler keeps a single happy
// success-or-error contract.
type resolveRequest struct {
	ResolvedResidentRef uuid.UUID `json:"resolved_resident_ref"`
	ResolvedBy          uuid.UUID `json:"resolved_by"`
	ResolutionNote      string    `json:"resolution_note,omitempty"`
}

// resolveResponse mirrors the IdentityStore.ResolveReview return shape.
type resolveResponse struct {
	Entry    *interfaces.IdentityReviewQueueEntry `json:"entry"`
	Rerouted int                                  `json:"rerouted_mappings"`
}

func (h *IdentityHandlers) resolveReview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req resolveRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ResolvedResidentRef == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resolved_resident_ref must be a non-zero UUID"})
		return
	}
	if req.ResolvedBy == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resolved_by must be a non-zero UUID"})
		return
	}
	entry, rerouted, err := h.store.ResolveReview(c.Request.Context(), id, req.ResolvedResidentRef, req.ResolvedBy, req.ResolutionNote)
	if err != nil {
		if errors.Is(err, interfaces.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resolveResponse{Entry: entry, Rerouted: rerouted})
}
