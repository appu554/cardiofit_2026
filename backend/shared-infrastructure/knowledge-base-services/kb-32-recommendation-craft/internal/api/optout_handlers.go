package api

// optout_handlers.go — POST/DELETE /v1/framing/optout/{gp_id}
//
// Prescriber framing opt-out HTTP endpoint (Phase 2-completion Task 6).
// Backs the migration-047 prescriber_framing_optout substrate via
// framing.OptOutStore.
//
// Permission middleware: NOT mounted on these routes. Phase 2-completion
// Task 7 is the follow-up that wraps /v1/craft/* with the shared
// permissions.Middleware; that work may extend to /v1/framing/* later.
// Until then, the routes are reachable to any authenticated caller and
// ingress filtering is the only access control.

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/framing"
)

// OptOutRequest is the JSON body for POST /v1/framing/optout/{gp_id}.
// The body is optional; if absent or unparseable, the handler proceeds
// with an empty reason (opt-out reasons are not required).
type OptOutRequest struct {
	// Reason is the optional free-text justification supplied by the GP.
	Reason string `json:"reason"`
}

// OptOutResponse is the JSON body returned by POST /v1/framing/optout/{gp_id}
// on 201 Created.
type OptOutResponse struct {
	// GPID echoes the path parameter.
	GPID string `json:"gp_id"`

	// Reason echoes the persisted reason (may be empty).
	Reason string `json:"reason,omitempty"`

	// OptedOutAt is the server-assigned timestamp of the opt-out action,
	// formatted as RFC3339 UTC. It is the wall-clock at which the handler
	// returned, not necessarily the database write time.
	OptedOutAt string `json:"opted_out_at"`
}

// OptOutHandler serves the prescriber framing opt-out endpoints.
type OptOutHandler struct {
	store framing.OptOutStore
}

// NewOptOutHandler constructs an OptOutHandler backed by the given store.
func NewOptOutHandler(store framing.OptOutStore) *OptOutHandler {
	return &OptOutHandler{store: store}
}

// HandleRegister serves POST /v1/framing/optout/:gp_id.
//
// Response codes:
//   - 201 Created       — opt-out registered (idempotent for re-register)
//   - 400 Bad Request   — :gp_id is not a parseable UUID
//   - 500 Internal      — substrate write failure (sanitised body; no SQL)
//
// The request body is OPTIONAL. A missing or malformed body is tolerated;
// reason simply defaults to empty.
func (h *OptOutHandler) HandleRegister(c *gin.Context) {
	gpID, err := uuid.Parse(c.Param("gp_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_gp_id"})
		return
	}

	// Body is optional — ShouldBindJSON failure is intentionally swallowed.
	// reason defaults to "" when the body is absent or unparseable.
	var req OptOutRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.store.RegisterOptOut(c.Request.Context(), gpID, req.Reason); err != nil {
		log.Printf("optout: RegisterOptOut failed for gp_id=%s: %v", gpID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "optout_register_failed"})
		return
	}

	c.JSON(http.StatusCreated, OptOutResponse{
		GPID:       gpID.String(),
		Reason:     req.Reason,
		OptedOutAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleRevoke serves DELETE /v1/framing/optout/:gp_id.
//
// Response codes:
//   - 204 No Content    — opt-out revoked, OR no active opt-out existed
//                         (idempotent no-op — the application semantic is
//                         "ensure this GP is not opted out" rather than
//                         "delete a specific record")
//   - 400 Bad Request   — :gp_id is not a parseable UUID
//   - 500 Internal      — substrate write failure
func (h *OptOutHandler) HandleRevoke(c *gin.Context) {
	gpID, err := uuid.Parse(c.Param("gp_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_gp_id"})
		return
	}

	if err := h.store.RevokeOptOut(c.Request.Context(), gpID); err != nil {
		log.Printf("optout: RevokeOptOut failed for gp_id=%s: %v", gpID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "optout_revoke_failed"})
		return
	}

	c.Status(http.StatusNoContent)
}
