package labs

import (
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	adapters map[string]LabAdapter
	mu       sync.RWMutex
	logger   *zap.Logger
}

func NewHandler(logger *zap.Logger, adapters ...LabAdapter) *Handler {
	h := &Handler{
		adapters: make(map[string]LabAdapter),
		logger:   logger,
	}
	for _, a := range adapters {
		h.adapters[a.LabID()] = a
	}
	return h
}

func (h *Handler) HandleLabWebhook(c *gin.Context) {
	labID := c.Param("labId")

	h.mu.RLock()
	adapter, exists := h.adapters[labID]
	h.mu.RUnlock()

	if !exists {
		h.logger.Warn("unknown lab ID", zap.String("lab_id", labID))
		c.JSON(http.StatusNotFound, gin.H{
			"error":      "unknown lab: " + labID,
			"known_labs": h.knownLabIDs(),
		})
		return
	}

	apiKey := c.GetHeader("X-API-Key")
	if !adapter.ValidateWebhookAuth(apiKey) {
		h.logger.Warn("lab webhook auth failed", zap.String("lab_id", labID))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	observations, err := adapter.Parse(c.Request.Context(), body)
	if err != nil {
		h.logger.Error("lab parsing failed",
			zap.String("lab_id", labID),
			zap.Error(err),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "parse failed: " + err.Error(),
		})
		return
	}

	h.logger.Info("lab results parsed",
		zap.String("lab_id", labID),
		zap.Int("observation_count", len(observations)),
	)

	c.JSON(http.StatusAccepted, gin.H{
		"status":            "accepted",
		"observation_count": len(observations),
	})
}

func (h *Handler) knownLabIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.adapters))
	for id := range h.adapters {
		ids = append(ids, id)
	}
	return ids
}
