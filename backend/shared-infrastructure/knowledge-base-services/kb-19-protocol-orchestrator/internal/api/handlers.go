// Package api provides the HTTP API server for KB-19 Protocol Orchestrator.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-19-protocol-orchestrator/internal/models"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

// handleHealth handles GET /health
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Service:   "kb-19-protocol-orchestrator",
		Version:   "1.0.0",
		Timestamp: time.Now(),
	})
}

// ReadyResponse represents the readiness check response.
type ReadyResponse struct {
	Ready       bool              `json:"ready"`
	Services    map[string]string `json:"services"`
	Timestamp   time.Time         `json:"timestamp"`
}

// handleReady handles GET /ready
func (s *Server) handleReady(c *gin.Context) {
	// Check all dependent services
	services := map[string]string{
		"vaidshala_cql":   "unknown",
		"kb3_temporal":    "unknown",
		"kb8_calculator":  "unknown",
		"kb12_orderset":   "unknown",
		"kb14_governance": "unknown",
	}

	// TODO: Actually check service health
	// For now, assume ready
	for k := range services {
		services[k] = "up"
	}

	c.JSON(http.StatusOK, ReadyResponse{
		Ready:     true,
		Services:  services,
		Timestamp: time.Now(),
	})
}

// handleMetrics handles GET /metrics (Prometheus format)
func (s *Server) handleMetrics(c *gin.Context) {
	// TODO: Implement Prometheus metrics
	c.String(http.StatusOK, "# KB-19 Protocol Orchestrator Metrics\n")
}

// ExecuteRequest represents the request body for protocol execution.
type ExecuteRequest struct {
	PatientID    string                 `json:"patient_id" binding:"required"`
	EncounterID  string                 `json:"encounter_id" binding:"required"`
	Context      map[string]interface{} `json:"context"`
	ProtocolIDs  []string               `json:"protocol_ids"` // Optional: specific protocols to evaluate
	Options      ExecuteOptions         `json:"options"`
}

// ExecuteOptions provides execution options.
type ExecuteOptions struct {
	IncludeNarrative bool `json:"include_narrative"`
	StrictMode       bool `json:"strict_mode"`
	MaxProtocols     int  `json:"max_protocols"`
}

// handleExecute handles POST /api/v1/execute
// This is the main entry point for protocol arbitration.
func (s *Server) handleExecute(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Parse UUIDs
	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_patient_id",
			"message": "patient_id must be a valid UUID",
		})
		return
	}

	encounterID, err := uuid.Parse(req.EncounterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_encounter_id",
			"message": "encounter_id must be a valid UUID",
		})
		return
	}

	// Execute arbitration
	bundle, err := s.engine.Execute(c.Request.Context(), patientID, encounterID, req.Context)
	if err != nil {
		s.log.WithError(err).Error("Arbitration execution failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "execution_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, bundle)
}

// EvaluateRequest represents the request body for single protocol evaluation.
type EvaluateRequest struct {
	PatientID   string                 `json:"patient_id" binding:"required"`
	EncounterID string                 `json:"encounter_id" binding:"required"`
	ProtocolID  string                 `json:"protocol_id" binding:"required"`
	Context     map[string]interface{} `json:"context"`
}

// handleEvaluate handles POST /api/v1/evaluate
// Evaluates a single protocol against patient context.
func (s *Server) handleEvaluate(c *gin.Context) {
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_patient_id",
			"message": "patient_id must be a valid UUID",
		})
		return
	}

	encounterID, err := uuid.Parse(req.EncounterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_encounter_id",
			"message": "encounter_id must be a valid UUID",
		})
		return
	}

	// Evaluate single protocol
	evaluation, err := s.engine.EvaluateProtocol(c.Request.Context(), patientID, encounterID, req.ProtocolID, req.Context)
	if err != nil {
		s.log.WithError(err).Error("Protocol evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "evaluation_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, evaluation)
}

// handleListProtocols handles GET /api/v1/protocols
func (s *Server) handleListProtocols(c *gin.Context) {
	// Get query parameters for filtering
	category := c.Query("category")
	setting := c.Query("setting")

	protocols := s.engine.ListProtocols(category, setting)
	c.JSON(http.StatusOK, gin.H{
		"protocols": protocols,
		"count":     len(protocols),
	})
}

// handleGetProtocol handles GET /api/v1/protocols/:id
func (s *Server) handleGetProtocol(c *gin.Context) {
	protocolID := c.Param("id")

	protocol, err := s.engine.GetProtocol(protocolID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "protocol_not_found",
			"message": "Protocol not found: " + protocolID,
		})
		return
	}

	c.JSON(http.StatusOK, protocol)
}

// handleGetDecisions handles GET /api/v1/decisions/:patientId
func (s *Server) handleGetDecisions(c *gin.Context) {
	patientIDStr := c.Param("patientId")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_patient_id",
			"message": "patient_id must be a valid UUID",
		})
		return
	}

	// Get query parameters
	limitStr := c.DefaultQuery("limit", "10")
	// TODO: Parse limit and use it

	decisions, err := s.engine.GetDecisionsForPatient(c.Request.Context(), patientID)
	if err != nil {
		s.log.WithError(err).Error("Failed to get decisions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "query_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"decisions":  decisions,
		"count":      len(decisions),
		"limit":      limitStr,
	})
}

// handleGetBundle handles GET /api/v1/bundle/:id
func (s *Server) handleGetBundle(c *gin.Context) {
	bundleIDStr := c.Param("id")
	bundleID, err := uuid.Parse(bundleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_bundle_id",
			"message": "bundle_id must be a valid UUID",
		})
		return
	}

	bundle, err := s.engine.GetBundle(c.Request.Context(), bundleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "bundle_not_found",
			"message": "Bundle not found: " + bundleIDStr,
		})
		return
	}

	c.JSON(http.StatusOK, bundle)
}

// handleListConflicts handles GET /api/v1/conflicts
func (s *Server) handleListConflicts(c *gin.Context) {
	conflicts := models.PredefinedConflicts

	c.JSON(http.StatusOK, gin.H{
		"conflicts": conflicts,
		"count":     len(conflicts),
	})
}

// handleGetConflictsForProtocol handles GET /api/v1/conflicts/:protocolId
func (s *Server) handleGetConflictsForProtocol(c *gin.Context) {
	protocolID := c.Param("protocolId")

	conflicts := models.GetConflictsForProtocol(protocolID)

	c.JSON(http.StatusOK, gin.H{
		"protocol_id": protocolID,
		"conflicts":   conflicts,
		"count":       len(conflicts),
	})
}
