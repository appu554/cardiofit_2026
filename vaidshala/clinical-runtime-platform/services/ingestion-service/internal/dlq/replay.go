package dlq

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ReplayHandler handles the DLQ replay endpoint.
// POST /fhir/OperationOutcome/:id/$replay
type ReplayHandler struct {
	publisher Publisher
	logger    *zap.Logger
}

// NewReplayHandler creates a new ReplayHandler.
func NewReplayHandler(publisher Publisher, logger *zap.Logger) *ReplayHandler {
	return &ReplayHandler{publisher: publisher, logger: logger}
}

// HandleReplay replays a single DLQ message by its ID.
func (h *ReplayHandler) HandleReplay(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid DLQ entry ID",
		})
		return
	}

	err = h.publisher.MarkReplayed(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to replay DLQ entry",
			zap.String("id", idStr),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to replay DLQ entry: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "replayed",
		"dlq_id":  id.String(),
		"message": "DLQ entry marked as replayed and queued for reprocessing",
	})
}

// HandleListPending lists all pending DLQ entries.
// GET /fhir/OperationOutcome?category=dlq
func (h *ReplayHandler) HandleListPending(c *gin.Context) {
	entries := h.publisher.ListPending(c.Request.Context())

	fhirEntries := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		fhirEntries = append(fhirEntries, gin.H{
			"resourceType": "OperationOutcome",
			"id":           e.ID.String(),
			"issue": []gin.H{
				{
					"severity":    "error",
					"code":        "processing",
					"diagnostics": e.ErrorMessage,
					"details": gin.H{
						"text": string(e.ErrorClass),
					},
				},
			},
			"extension": []gin.H{
				{
					"url":         "source_type",
					"valueString": e.SourceType,
				},
				{
					"url":         "source_id",
					"valueString": e.SourceID,
				},
				{
					"url":         "created_at",
					"valueString": e.CreatedAt.String(),
				},
				{
					"url":          "retry_count",
					"valueInteger": e.RetryCount,
				},
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        len(fhirEntries),
		"entry":        fhirEntries,
	})
}
