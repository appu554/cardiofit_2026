package api

import (
	"net/http"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *Server) getTwinConfidence(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	twin, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin not found", "NOT_FOUND", nil)
		return
	}

	analysis := services.AnalyzeConfidence(twin)
	sendSuccess(c, analysis, nil)
}

func (s *Server) webhookObservation(c *gin.Context) {
	var event models.ObservationEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		sendError(c, http.StatusBadRequest, "invalid event", "INVALID_EVENT", nil)
		return
	}
	s.logger.Info("observation event received",
		zap.String("patient", event.PatientID),
		zap.String("code", event.Code),
	)
	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
}

func (s *Server) webhookCheckin(c *gin.Context) {
	var event models.CheckinEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		sendError(c, http.StatusBadRequest, "invalid event", "INVALID_EVENT", nil)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
}

func (s *Server) webhookMedChange(c *gin.Context) {
	var event models.MedChangeEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		sendError(c, http.StatusBadRequest, "invalid event", "INVALID_EVENT", nil)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
}
