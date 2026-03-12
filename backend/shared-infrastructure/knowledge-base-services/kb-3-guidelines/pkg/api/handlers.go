// Package api provides REST API handlers for KB-3 Guidelines Service
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb-3-guidelines/pkg/models"
	"github.com/cardiofit/kb-3-guidelines/pkg/protocols"
	"github.com/cardiofit/kb-3-guidelines/pkg/temporal"
)

// Handler provides HTTP handlers for the API
type Handler struct {
	registry   *protocols.ProtocolRegistry
	pathway    *temporal.PathwayEngine
	scheduling *temporal.SchedulingEngine
}

// NewHandler creates a new API handler
func NewHandler() *Handler {
	return &Handler{
		registry:   protocols.GetRegistry(),
		pathway:    temporal.GetPathwayEngine(),
		scheduling: temporal.GetSchedulingEngine(),
	}
}

// ===== Health & Status Handlers =====

// Health checks service health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "kb-3-guidelines",
		"version": "1.0.0",
		"time":    time.Now().UTC(),
	})
}

// Metrics returns service metrics
func (h *Handler) Metrics(c *gin.Context) {
	summary := h.registry.GetProtocolSummary()

	c.JSON(http.StatusOK, gin.H{
		"protocols": summary,
		"active_pathways": len(h.pathway.GetActivePathways()),
		"overdue_alerts":  len(h.pathway.GetOverdueAlerts()),
	})
}

// Version returns service version info
func (h *Handler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":     "kb-3-guidelines",
		"version":     "1.0.0",
		"api_version": "v1",
		"build_date":  "2024-01-01",
	})
}

// ===== Protocol Handlers =====

// ListProtocols returns all available protocols
func (h *Handler) ListProtocols(c *gin.Context) {
	summary := h.registry.GetProtocolSummary()
	c.JSON(http.StatusOK, summary)
}

// ListAcuteProtocols returns all acute protocol definitions
func (h *Handler) ListAcuteProtocols(c *gin.Context) {
	protocols := h.registry.ListAcuteProtocols()
	c.JSON(http.StatusOK, protocols)
}

// ListChronicSchedules returns all chronic schedule definitions
func (h *Handler) ListChronicSchedules(c *gin.Context) {
	schedules := h.registry.ListChronicSchedules()
	c.JSON(http.StatusOK, schedules)
}

// ListPreventiveSchedules returns all preventive schedule definitions
func (h *Handler) ListPreventiveSchedules(c *gin.Context) {
	schedules := h.registry.ListPreventiveSchedules()
	c.JSON(http.StatusOK, schedules)
}

// GetProtocol returns a specific protocol by type and ID
func (h *Handler) GetProtocol(c *gin.Context) {
	protocolType := c.Param("type")
	protocolID := c.Param("id")

	switch protocolType {
	case "acute":
		protocol, err := h.registry.GetAcuteProtocol(protocolID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, protocol)

	case "chronic":
		schedule, err := h.registry.GetChronicSchedule(protocolID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, schedule)

	case "preventive":
		schedule, err := h.registry.GetPreventiveSchedule(protocolID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, schedule)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid protocol type"})
	}
}

// SearchProtocols searches for protocols
func (h *Handler) SearchProtocols(c *gin.Context) {
	query := c.Query("q")
	protocolType := c.Query("type")

	results := h.registry.SearchProtocols(query, protocolType)
	c.JSON(http.StatusOK, results)
}

// GetProtocolsByCondition returns protocols for a condition
func (h *Handler) GetProtocolsByCondition(c *gin.Context) {
	condition := c.Param("condition")
	results := h.registry.GetProtocolsByCondition(condition)
	c.JSON(http.StatusOK, results)
}

// ===== Pathway Handlers =====

// StartPathway initiates a new pathway for a patient
func (h *Handler) StartPathway(c *gin.Context) {
	var req models.StartPathwayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the protocol
	protocol, err := h.registry.GetAcuteProtocol(req.PathwayID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "protocol not found"})
		return
	}

	// Start the pathway
	instance, err := h.pathway.StartPathway(protocol, req.PatientID, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, instance)
}

// GetPathwayStatus returns the current status of a pathway
func (h *Handler) GetPathwayStatus(c *gin.Context) {
	instanceID := c.Param("id")

	instance, err := h.pathway.GetPathwayStatus(instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, instance)
}

// GetPendingActions returns pending actions for a pathway
func (h *Handler) GetPendingActions(c *gin.Context) {
	instanceID := c.Param("id")

	actions, err := h.pathway.GetPendingActions(instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, actions)
}

// GetOverdueActions returns overdue actions for a pathway
func (h *Handler) GetOverdueActions(c *gin.Context) {
	instanceID := c.Param("id")

	actions, err := h.pathway.GetOverdueActions(instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, actions)
}

// EvaluateConstraints evaluates time constraints for a pathway
func (h *Handler) EvaluateConstraints(c *gin.Context) {
	instanceID := c.Param("id")

	evaluations, err := h.pathway.EvaluateConstraints(instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, evaluations)
}

// GetPathwayAudit returns the audit log for a pathway
func (h *Handler) GetPathwayAudit(c *gin.Context) {
	instanceID := c.Param("id")

	audit, err := h.pathway.GetPathwayAudit(instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, audit)
}

// AdvanceStage advances the pathway to the next stage
func (h *Handler) AdvanceStage(c *gin.Context) {
	instanceID := c.Param("id")

	if err := h.pathway.AdvanceStage(instanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "advanced"})
}

// CompleteAction marks an action as completed
func (h *Handler) CompleteAction(c *gin.Context) {
	instanceID := c.Param("id")

	var req models.CompleteActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.pathway.CompleteAction(instanceID, req.ActionID, req.CompletedBy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}

// SuspendPathway suspends a pathway
func (h *Handler) SuspendPathway(c *gin.Context) {
	instanceID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.pathway.SuspendPathway(instanceID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "suspended"})
}

// ResumePathway resumes a suspended pathway
func (h *Handler) ResumePathway(c *gin.Context) {
	instanceID := c.Param("id")

	if err := h.pathway.ResumePathway(instanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "resumed"})
}

// CancelPathway cancels a pathway
func (h *Handler) CancelPathway(c *gin.Context) {
	instanceID := c.Param("id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := h.pathway.CancelPathway(instanceID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

// ===== Patient Handlers =====

// GetPatientPathways returns all pathways for a patient
func (h *Handler) GetPatientPathways(c *gin.Context) {
	patientID := c.Param("id")
	pathways := h.pathway.GetPatientPathways(patientID)
	c.JSON(http.StatusOK, pathways)
}

// GetPatientSchedule returns the schedule for a patient
func (h *Handler) GetPatientSchedule(c *gin.Context) {
	patientID := c.Param("id")

	schedule, err := h.scheduling.GetPatientSchedule(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// GetScheduleSummary returns a schedule summary for a patient
func (h *Handler) GetScheduleSummary(c *gin.Context) {
	patientID := c.Param("id")
	summary := h.scheduling.GetScheduleSummary(patientID)
	c.JSON(http.StatusOK, summary)
}

// GetPatientOverdue returns overdue items for a patient
func (h *Handler) GetPatientOverdue(c *gin.Context) {
	patientID := c.Param("id")

	items, err := h.scheduling.GetOverdueItems(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetPatientUpcoming returns upcoming items for a patient
func (h *Handler) GetPatientUpcoming(c *gin.Context) {
	patientID := c.Param("id")

	days := 7 // Default to 7 days
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	items, err := h.scheduling.GetUpcoming(patientID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// ExportPatientData exports all KB-3 data for a patient (pathways, schedules, audit logs)
func (h *Handler) ExportPatientData(c *gin.Context) {
	patientID := c.Param("id")

	// Get all pathways for patient
	pathways := h.pathway.GetPatientPathways(patientID)

	// Get schedule and summary
	schedule, _ := h.scheduling.GetPatientSchedule(patientID)
	summary := h.scheduling.GetScheduleSummary(patientID)
	overdue, _ := h.scheduling.GetOverdueItems(patientID)

	// Build export response
	export := gin.H{
		"patient_id":  patientID,
		"exported_at": time.Now().UTC(),
		"version":     "3.0.0",
		"data": gin.H{
			"pathways": gin.H{
				"active":    pathways,
				"count":     len(pathways),
			},
			"schedule": gin.H{
				"items":   schedule,
				"summary": summary,
				"overdue": overdue,
			},
		},
		"metadata": gin.H{
			"format":  "json",
			"service": "kb-3-guidelines",
		},
	}

	// Set content disposition for download if requested
	if c.Query("download") == "true" {
		c.Header("Content-Disposition", "attachment; filename=patient-"+patientID+"-kb3-export.json")
	}

	c.JSON(http.StatusOK, export)
}

// StartProtocolForPatient starts a protocol for a patient
func (h *Handler) StartProtocolForPatient(c *gin.Context) {
	patientID := c.Param("id")

	var req struct {
		ProtocolID   string                 `json:"protocol_id" binding:"required"`
		ProtocolType string                 `json:"protocol_type" binding:"required"`
		Context      map[string]interface{} `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.ProtocolType {
	case "acute":
		protocol, err := h.registry.GetAcuteProtocol(req.ProtocolID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "protocol not found"})
			return
		}

		instance, err := h.pathway.StartPathway(protocol, patientID, req.Context)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, instance)

	case "chronic":
		schedule, err := h.registry.GetChronicSchedule(req.ProtocolID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}

		items, err := h.scheduling.ApplyChronicSchedule(patientID, schedule, time.Now())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"scheduled_items": items})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid protocol type"})
	}
}

// ===== Scheduling Handlers =====

// GetSchedule returns schedule for a patient
func (h *Handler) GetSchedule(c *gin.Context) {
	patientID := c.Param("patientId")

	schedule, err := h.scheduling.GetPatientSchedule(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// GetSchedulePending returns pending scheduled items
func (h *Handler) GetSchedulePending(c *gin.Context) {
	patientID := c.Param("patientId")

	items, err := h.scheduling.GetPendingItems(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// AddScheduledItem adds a new scheduled item
func (h *Handler) AddScheduledItem(c *gin.Context) {
	patientID := c.Param("patientId")

	var req models.AddScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.scheduling.AddScheduledItem(patientID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// CompleteScheduledItem marks a scheduled item as completed
func (h *Handler) CompleteScheduledItem(c *gin.Context) {
	patientID := c.Param("patientId")

	var req struct {
		ItemID string `json:"item_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.scheduling.CompleteItem(patientID, req.ItemID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}

// ===== Temporal Handlers =====

// EvaluateTemporalRelation evaluates a temporal relationship
func (h *Handler) EvaluateTemporalRelation(c *gin.Context) {
	var req temporal.TemporalRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	target := temporal.NewInterval(req.TargetStart, req.TargetEnd)
	reference := temporal.NewInterval(req.ReferenceStart, req.ReferenceEnd)

	var offset time.Duration
	if req.Offset != nil {
		if parsed, err := time.ParseDuration(*req.Offset); err == nil {
			offset = parsed
		}
	}

	result := temporal.EvaluateTemporalRelation(target, reference, req.Operator, offset)

	response := temporal.TemporalRelationResponse{
		Result:    result,
		Operator:  req.Operator,
		Target:    target,
		Reference: reference,
	}
	if offset > 0 {
		response.Offset = &offset
	}

	c.JSON(http.StatusOK, response)
}

// CalculateNextOccurrence calculates the next occurrence of a recurrence pattern
func (h *Handler) CalculateNextOccurrence(c *gin.Context) {
	var req temporal.NextOccurrenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pattern := models.RecurrencePattern{
		Frequency: models.Frequency(req.Recurrence.Frequency),
		Interval:  req.Recurrence.Interval,
	}

	next := pattern.CalculateNextOccurrence(req.FromTime)

	c.JSON(http.StatusOK, temporal.NextOccurrenceResponse{
		NextOccurrence: next,
		FromTime:       req.FromTime,
		Frequency:      req.Recurrence.Frequency,
		Interval:       req.Recurrence.Interval,
	})
}

// ValidateConstraintTiming validates if an action meets its time constraint
func (h *Handler) ValidateConstraintTiming(c *gin.Context) {
	var req temporal.ValidateConstraintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := temporal.ValidateConstraintTiming(
		req.ActionTime,
		req.ReferenceTime,
		req.Deadline,
		req.GracePeriod,
	)

	c.JSON(http.StatusOK, result)
}

// ===== Alert Handlers =====

// ProcessAlerts processes and returns current alerts
func (h *Handler) ProcessAlerts(c *gin.Context) {
	// Get pathway overdue alerts
	pathwayAlerts := h.pathway.GetOverdueAlerts()

	// Get scheduling overdue items
	overdueItems := h.scheduling.GetAllOverdueItems()

	c.JSON(http.StatusOK, gin.H{
		"pathway_alerts":  pathwayAlerts,
		"scheduling_alerts": overdueItems,
		"processed_at":    time.Now().UTC(),
	})
}

// GetAllOverdue returns all overdue items across the system
func (h *Handler) GetAllOverdue(c *gin.Context) {
	pathwayAlerts := h.pathway.GetOverdueAlerts()
	overdueItems := h.scheduling.GetAllOverdueItems()

	c.JSON(http.StatusOK, gin.H{
		"pathway_overdue":    pathwayAlerts,
		"scheduling_overdue": overdueItems,
		"total_count":        len(pathwayAlerts) + len(overdueItems),
	})
}

// ===== Batch Handlers =====

// BatchStartProtocols starts multiple protocols
func (h *Handler) BatchStartProtocols(c *gin.Context) {
	var req struct {
		Requests []models.StartPathwayRequest `json:"requests" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var results []gin.H
	for _, r := range req.Requests {
		protocol, err := h.registry.GetAcuteProtocol(r.PathwayID)
		if err != nil {
			results = append(results, gin.H{
				"patient_id": r.PatientID,
				"pathway_id": r.PathwayID,
				"error":      "protocol not found",
			})
			continue
		}

		instance, err := h.pathway.StartPathway(protocol, r.PatientID, r.Context)
		if err != nil {
			results = append(results, gin.H{
				"patient_id": r.PatientID,
				"pathway_id": r.PathwayID,
				"error":      err.Error(),
			})
			continue
		}

		results = append(results, gin.H{
			"patient_id":  r.PatientID,
			"pathway_id":  r.PathwayID,
			"instance_id": instance.InstanceID,
			"status":      "started",
		})
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// ===== Governance Handlers (Placeholder - to be integrated with database) =====

// GetGuidelines returns all guidelines
func (h *Handler) GetGuidelines(c *gin.Context) {
	// This would be populated from the database
	c.JSON(http.StatusOK, []models.Guideline{})
}

// GetGuideline returns a specific guideline
func (h *Handler) GetGuideline(c *gin.Context) {
	guidelineID := c.Param("id")
	c.JSON(http.StatusNotFound, gin.H{"error": "guideline not found", "id": guidelineID})
}

// ResolveConflict resolves a conflict between guidelines
func (h *Handler) ResolveConflict(c *gin.Context) {
	var req models.ResolveConflictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Placeholder - would implement full conflict resolution logic
	resolution := models.Resolution{
		Applicable: true,
		RuleUsed:   "safety_first",
		Rationale:  "Applied safety-first conflict resolution",
	}

	c.JSON(http.StatusOK, resolution)
}

// GetSafetyOverrides returns all active safety overrides
func (h *Handler) GetSafetyOverrides(c *gin.Context) {
	// This would be populated from the database
	c.JSON(http.StatusOK, []models.SafetyOverride{})
}

// CreateSafetyOverride creates a new safety override
func (h *Handler) CreateSafetyOverride(c *gin.Context) {
	var override models.SafetyOverride
	if err := c.ShouldBindJSON(&override); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	override.OverrideID = uuid.New().String()
	override.Active = true

	c.JSON(http.StatusCreated, override)
}

// CreateVersion creates a new guideline version
func (h *Handler) CreateVersion(c *gin.Context) {
	var req models.CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	version := models.GuidelineVersion{
		VersionID:   uuid.New().String(),
		GuidelineID: req.GuidelineID,
		ChangeType:  req.ChangeType,
		Changes:     req.Changes,
		Status:      models.VersionDraft,
		CreatedBy:   req.RequestorID,
		CreatedAt:   time.Now(),
	}

	c.JSON(http.StatusCreated, version)
}

// ProcessApproval processes an approval for a version
func (h *Handler) ProcessApproval(c *gin.Context) {
	versionID := c.Param("id")

	var req models.ProcessApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"version_id": versionID,
		"status":     req.Status,
		"message":    "Approval processed",
	})
}
