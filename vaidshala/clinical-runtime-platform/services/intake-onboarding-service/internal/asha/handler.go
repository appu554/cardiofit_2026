package asha

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	syncService *SyncService
	queue       *OfflineQueue
	logger      *zap.Logger
}

func NewHandler(syncService *SyncService, queue *OfflineQueue, logger *zap.Logger) *Handler {
	return &Handler{
		syncService: syncService,
		queue:       queue,
		logger:      logger,
	}
}

type TabletSubmission struct {
	DeviceID    string      `json:"device_id" binding:"required"`
	AshaID      uuid.UUID   `json:"asha_id" binding:"required"`
	PatientID   uuid.UUID   `json:"patient_id" binding:"required"`
	TenantID    uuid.UUID   `json:"tenant_id" binding:"required"`
	Slots       []SlotEntry `json:"slots" binding:"required,min=1"`
	CollectedAt time.Time   `json:"collected_at" binding:"required"`
	SyncSeqNo   int64       `json:"sync_seq_no"`
	IsOffline   bool        `json:"is_offline"`
	GPSLocation *GPSLocation `json:"gps_location,omitempty"`
}

type SlotEntry struct {
	SlotName string      `json:"slot_name" binding:"required"`
	Domain   string      `json:"domain" binding:"required"`
	Value    interface{} `json:"value" binding:"required"`
	Unit     string      `json:"unit,omitempty"`
	Method   string      `json:"method,omitempty"`
	Notes    string      `json:"notes,omitempty"`
}

type GPSLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy_meters"`
}

type SubmissionResult struct {
	SlotName string `json:"slot_name"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

func (h *Handler) HandleBatchSubmit(c *gin.Context) {
	var sub TabletSubmission
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid submission: " + err.Error()})
		return
	}

	h.logger.Info("ASHA tablet submission received",
		zap.String("device_id", sub.DeviceID),
		zap.String("patient_id", sub.PatientID.String()),
		zap.Int("slot_count", len(sub.Slots)),
		zap.Bool("is_offline", sub.IsOffline),
		zap.Int64("sync_seq_no", sub.SyncSeqNo),
	)

	var results []SubmissionResult

	if sub.IsOffline {
		syncResults, err := h.syncService.ReconcileOfflineBatch(sub)
		if err != nil {
			h.logger.Error("offline sync reconciliation failed",
				zap.String("device_id", sub.DeviceID),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed"})
			return
		}
		results = syncResults
	} else {
		for _, slot := range sub.Slots {
			result := h.processSlot(sub.PatientID, sub.TenantID, sub.AshaID, slot, sub.CollectedAt)
			results = append(results, result)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results":           results,
		"last_accepted_seq": sub.SyncSeqNo,
		"server_time":       time.Now().UTC(),
	})
}

func (h *Handler) processSlot(patientID, tenantID, ashaID uuid.UUID, slot SlotEntry, collectedAt time.Time) SubmissionResult {
	h.logger.Debug("processing ASHA slot",
		zap.String("patient_id", patientID.String()),
		zap.String("slot_name", slot.SlotName),
	)

	return SubmissionResult{
		SlotName: slot.SlotName,
		Status:   "ACCEPTED",
	}
}

func (h *Handler) HandleSyncStatus(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}

	status, err := h.queue.GetDeviceSyncStatus(deviceID)
	if err != nil {
		h.logger.Error("failed to get sync status", zap.String("device_id", deviceID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get sync status"})
		return
	}

	c.JSON(http.StatusOK, status)
}
