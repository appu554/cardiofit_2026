package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/dlq"
)

// handleDLQList returns DLQ entries with optional status/error_class/source_type filters.
func (s *Server) handleDLQList(c *gin.Context) {
	filter := dlq.ListFilter{
		Limit:  50,
		Offset: 0,
	}

	if st := c.Query("status"); st != "" {
		status := dlq.DLQStatus(st)
		filter.Status = &status
	}
	if ec := c.Query("error_class"); ec != "" {
		errClass := dlq.ErrorClass(ec)
		filter.ErrorClass = &errClass
	}
	if src := c.Query("source_type"); src != "" {
		filter.SourceType = &src
	}

	entries, err := s.dlqResolver.List(c.Request.Context(), filter)
	if err != nil {
		s.logger.Error("failed to list DLQ entries", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list DLQ entries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entries": entries, "count": len(entries)})
}

// handleDLQGet returns a single DLQ entry by ID.
func (s *Server) handleDLQGet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid DLQ entry ID"})
		return
	}

	entry, err := s.dlqResolver.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entry)
}

// handleDLQDiscard marks a DLQ entry as discarded (will not be retried).
func (s *Server) handleDLQDiscard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid DLQ entry ID"})
		return
	}

	if err := s.dlqResolver.Discard(c.Request.Context(), id); err != nil {
		s.logger.Error("failed to discard DLQ entry", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "discarded"})
}

// handleDLQCount returns DLQ entry counts grouped by status.
func (s *Server) handleDLQCount(c *gin.Context) {
	counts, err := s.dlqResolver.Count(c.Request.Context())
	if err != nil {
		s.logger.Error("failed to count DLQ entries", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count DLQ entries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"counts": counts})
}
