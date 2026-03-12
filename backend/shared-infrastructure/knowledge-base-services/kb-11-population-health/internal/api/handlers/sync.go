// Package handlers provides HTTP request handlers for the KB-11 API.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/models"
	"github.com/cardiofit/kb-11-population-health/internal/projection"
)

// SyncHandler handles data synchronization endpoints.
// IMPORTANT: All sync operations are READ-ONLY from upstream sources.
// KB-11 CONSUMES data, it does NOT write to FHIR Store or KB-17.
type SyncHandler struct {
	service *projection.Service
	logger  *logrus.Entry
}

// NewSyncHandler creates a new sync handler.
func NewSyncHandler(service *projection.Service, logger *logrus.Entry) *SyncHandler {
	return &SyncHandler{
		service: service,
		logger:  logger.WithField("handler", "sync"),
	}
}

// GetAllSyncStatus handles GET /v1/sync/status - get sync status for all sources.
func (h *SyncHandler) GetAllSyncStatus(c *gin.Context) {
	statuses, err := h.service.GetAllSyncStatus(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sync status")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get sync status",
			"STATUS_ERROR",
			err.Error(),
		))
		return
	}

	responses := make([]*models.SyncStatusResponse, len(statuses))
	for i, s := range statuses {
		responses[i] = models.FromSyncStatusRecord(s)
	}

	c.JSON(http.StatusOK, gin.H{
		"sync_statuses": responses,
	})
}

// GetSyncStatus handles GET /v1/sync/status/:source - get sync status for a source.
func (h *SyncHandler) GetSyncStatus(c *gin.Context) {
	sourceStr := c.Param("source")
	source := models.SyncSource(sourceStr)

	if !source.IsValid() {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid sync source",
			"INVALID_SOURCE",
			"Valid sources: FHIR, KB17, KB13",
		))
		return
	}

	status, err := h.service.GetSyncStatus(c.Request.Context(), source)
	if err != nil {
		h.logger.WithError(err).WithField("source", source).Error("Failed to get sync status")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get sync status",
			"STATUS_ERROR",
			err.Error(),
		))
		return
	}

	if status == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			"Sync status not found",
			"NOT_FOUND",
			"",
		))
		return
	}

	c.JSON(http.StatusOK, models.FromSyncStatusRecord(status))
}

// TriggerFHIRSync handles POST /v1/sync/fhir - trigger FHIR sync.
// IMPORTANT: This is a READ-ONLY sync operation from FHIR Store.
func (h *SyncHandler) TriggerFHIRSync(c *gin.Context) {
	var req models.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		req = models.SyncRequest{
			Source:     models.SyncSourceFHIR,
			FullSync:   false,
			MaxRecords: 1000,
		}
	}

	// Check if sync is already in progress
	if h.service.IsSyncActive(models.SyncSourceFHIR) {
		c.JSON(http.StatusConflict, models.NewErrorResponse(
			"FHIR sync already in progress",
			"SYNC_IN_PROGRESS",
			"",
		))
		return
	}

	// Start sync in background
	go func() {
		result, err := h.service.SyncFromFHIR(c.Request.Context(), req.FullSync, req.MaxRecords)
		if err != nil {
			h.logger.WithError(err).Error("FHIR sync failed")
		} else {
			h.logger.WithFields(logrus.Fields{
				"records_synced": result.RecordsSynced,
				"duration":       result.Duration.String(),
			}).Info("FHIR sync completed")
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "FHIR sync started",
		"source":      "FHIR",
		"full_sync":   req.FullSync,
		"max_records": req.MaxRecords,
	})
}

// TriggerKB17Sync handles POST /v1/sync/kb17 - trigger KB-17 sync.
// IMPORTANT: This is a READ-ONLY sync operation from KB-17 Registry.
func (h *SyncHandler) TriggerKB17Sync(c *gin.Context) {
	var req models.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		req = models.SyncRequest{
			Source:     models.SyncSourceKB17,
			FullSync:   false,
			MaxRecords: 1000,
		}
	}

	// Check if sync is already in progress
	if h.service.IsSyncActive(models.SyncSourceKB17) {
		c.JSON(http.StatusConflict, models.NewErrorResponse(
			"KB-17 sync already in progress",
			"SYNC_IN_PROGRESS",
			"",
		))
		return
	}

	// Start sync in background
	go func() {
		result, err := h.service.SyncFromKB17(c.Request.Context(), req.FullSync, req.MaxRecords)
		if err != nil {
			h.logger.WithError(err).Error("KB-17 sync failed")
		} else {
			h.logger.WithFields(logrus.Fields{
				"records_synced": result.RecordsSynced,
				"duration":       result.Duration.String(),
			}).Info("KB-17 sync completed")
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "KB-17 sync started",
		"source":      "KB17",
		"full_sync":   req.FullSync,
		"max_records": req.MaxRecords,
	})
}
