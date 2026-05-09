// Package api provides the HTTP handlers for the ethics-monitoring service.
// Phase 3 Task 1 exposes only /healthz; future tasks will add EBA finding
// query endpoints.
//
// VisibilityClass: AD
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Version is the service version emitted in /healthz responses. Bumped per
// Phase 3 task as behaviour evolves.
const Version = "0.1.0-phase-3-task-1"

// JobCounter is implemented by the cron orchestrator. The api package depends
// on this minimal interface rather than the concrete *cron.Orchestrator to
// keep imports acyclic and tests trivial.
type JobCounter interface {
	JobCount() int
}

// Handler bundles HTTP handlers with their collaborators.
type Handler struct {
	Orch JobCounter
}

// NewHandler constructs a Handler.
func NewHandler(orch JobCounter) *Handler {
	return &Handler{Orch: orch}
}

// Healthz returns 200 with status, version, and registered job count.
func (h *Handler) Healthz(c *gin.Context) {
	jobs := 0
	if h.Orch != nil {
		jobs = h.Orch.JobCount()
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": Version,
		"jobs":    jobs,
	})
}

// Register attaches the /healthz route to r.
func (h *Handler) Register(r *gin.Engine) {
	r.GET("/healthz", h.Healthz)
}
