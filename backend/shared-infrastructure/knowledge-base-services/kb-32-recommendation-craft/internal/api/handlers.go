package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DraftRequest is the request body for POST /v1/craft/draft.
// All three fields are required; absent or malformed UUIDs produce 400.
type DraftRequest struct {
	// RuleID is the CQL rule identifier to evaluate and drive the packet.
	RuleID string `json:"rule_id" binding:"required"`

	// ResidentID is the UUID of the resident for whom the recommendation
	// is being drafted.
	ResidentID string `json:"resident_id" binding:"required"`

	// AuthorID is the UUID of the pharmacist (or system actor) initiating
	// the draft. Recorded in the Packet and propagated to the EvidenceTrace.
	AuthorID string `json:"author_id" binding:"required"`
}

// DraftResponse is the JSON response from POST /v1/craft/draft.
type DraftResponse struct {
	// RecommendationID is the UUID of the generated draft packet.
	RecommendationID string `json:"recommendation_id"`

	// State is either "drafted" (gate passed) or "detected" (gate held).
	State string `json:"state"`

	// ContentHash is the SHA-256 hex content hash from Stage 5.
	// Empty when State="detected" (gate held before hashing).
	ContentHash string `json:"content_hash,omitempty"`

	// HoldReason is set when State="detected" and describes which
	// appropriateness dimension triggered the hold.
	HoldReason string `json:"hold_reason,omitempty"`

	// UrgencyTag is the urgency tier derived from the ClinicalSnapshot
	// ("red", "amber", or "green"). Always present.
	UrgencyTag string `json:"urgency_tag"`
}

// Handler holds the dependencies for the /v1/craft/ route group.
type Handler struct {
	pipeline *Pipeline
}

// NewHandler constructs a Handler with the given Pipeline.
func NewHandler(pipeline *Pipeline) *Handler {
	return &Handler{pipeline: pipeline}
}

// HandleDraft is the thin Gin handler for POST /v1/craft/draft.
// It parses and validates the request, delegates to Pipeline.Run, and maps
// the PipelineResult to a DraftResponse.
//
// HTTP status codes:
//   - 200: gate passed → State="drafted"
//   - 200: gate held   → State="detected" (a held recommendation is a valid outcome)
//   - 400: malformed JSON or missing/invalid UUID fields
//   - 500: pipeline infrastructure error (substrate unavailable, no applicable rules, etc.)
func (h *Handler) HandleDraft(c *gin.Context) {
	var req DraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	residentID, err := uuid.Parse(req.ResidentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resident_id: " + err.Error()})
		return
	}

	authorID, err := uuid.Parse(req.AuthorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "author_id: " + err.Error()})
		return
	}

	result, err := h.pipeline.Run(c.Request.Context(), req.RuleID, residentID, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := DraftResponse{
		RecommendationID: result.Packet.RecommendationID.String(),
		UrgencyTag:       result.UrgencyTag,
	}

	if result.HoldReason != "" {
		resp.State = "detected"
		resp.HoldReason = result.HoldReason
	} else {
		resp.State = "drafted"
		resp.ContentHash = result.ContentHash
	}

	c.JSON(http.StatusOK, resp)
}
