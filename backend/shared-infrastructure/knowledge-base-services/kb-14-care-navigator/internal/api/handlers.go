// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/models"
)

// CreateTask creates a new task
func (s *Server) CreateTask(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	task, err := s.taskService.Create(c.Request.Context(), &req)
	if err != nil {
		s.log.WithError(err).Error("Failed to create task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// GetTask retrieves a task by ID
func (s *Server) GetTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	task, err := s.taskService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "task not found",
		})
		return
	}

	c.JSON(http.StatusOK, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// ListTasks retrieves tasks with optional filters
func (s *Server) ListTasks(c *gin.Context) {
	var filters models.WorklistFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set defaults
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PageSize <= 0 || filters.PageSize > 100 {
		filters.PageSize = 20
	}

	response, err := s.worklistService.GetWorklist(c.Request.Context(), filters)
	if err != nil {
		s.log.WithError(err).Error("Failed to list tasks")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateTask updates a task
func (s *Server) UpdateTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	task, err := s.taskService.Update(c.Request.Context(), id, &req)
	if err != nil {
		s.log.WithError(err).Error("Failed to update task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// DeleteTask deletes a task
func (s *Server) DeleteTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	if _, err := s.taskService.Cancel(c.Request.Context(), id, "Deleted via API"); err != nil {
		s.log.WithError(err).Error("Failed to delete task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "task deleted",
	})
}

// AssignTask assigns a task to a team member
func (s *Server) AssignTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	var req models.AssignTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	task, err := s.taskService.Assign(c.Request.Context(), id, &req)
	if err != nil {
		s.log.WithError(err).Error("Failed to assign task")

		// Return 400 for validation errors (inactive member, no capacity, not found)
		if errors.Is(err, database.ErrInactiveMember) ||
			errors.Is(err, database.ErrNoCapacity) ||
			errors.Is(err, database.ErrNotFound) ||
			strings.Contains(err.Error(), "cannot assign") ||
			strings.Contains(err.Error(), "assignee not found") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// StartTask starts work on a task
func (s *Server) StartTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	task, err := s.taskService.Start(c.Request.Context(), id, userID)
	if err != nil {
		s.log.WithError(err).Error("Failed to start task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// CompleteTask completes a task
func (s *Server) CompleteTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	var req models.CompleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	task, err := s.taskService.Complete(c.Request.Context(), id, userID, &req)
	if err != nil {
		s.log.WithError(err).Error("Failed to complete task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// CancelTask cancels a task
func (s *Server) CancelTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	task, err := s.taskService.Cancel(c.Request.Context(), id, req.Reason)
	if err != nil {
		s.log.WithError(err).Error("Failed to cancel task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// EscalateTask manually escalates a task
func (s *Server) EscalateTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get task
	task, err := s.taskService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "task not found",
		})
		return
	}

	// Escalate
	escalation, err := s.escalationEngine.EscalateTask(c.Request.Context(), task, req.Reason)
	if err != nil {
		s.log.WithError(err).Error("Failed to escalate task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"escalation": escalation,
	})
}

// AddNote adds a note to a task
func (s *Server) AddNote(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	var req models.AddNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	task, err := s.taskService.AddNote(c.Request.Context(), id, &req)
	if err != nil {
		s.log.WithError(err).Error("Failed to add note")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// CreateFromCareGap creates a task from a KB-9 care gap
func (s *Server) CreateFromCareGap(c *gin.Context) {
	var gap struct {
		GapID          string                 `json:"gap_id" binding:"required"`
		PatientID      string                 `json:"patient_id" binding:"required"`
		GapType        string                 `json:"gap_type" binding:"required"`
		GapCategory    string                 `json:"gap_category"`
		Title          string                 `json:"title" binding:"required"`
		Description    string                 `json:"description"`
		DueDate        string                 `json:"due_date"`
		Interventions  []map[string]string    `json:"interventions"`
		Metadata       map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&gap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Convert to CareGap struct
	careGap := &models.CareGap{
		GapID:       gap.GapID,
		PatientID:   gap.PatientID,
		GapType:     gap.GapType,
		GapCategory: gap.GapCategory,
		Title:       gap.Title,
		Description: gap.Description,
	}

	task, err := s.taskFactory.CreateFromCareGapModel(c.Request.Context(), careGap)
	if err != nil {
		s.log.WithError(err).Error("Failed to create task from care gap")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// CreateFromTemporalAlert creates a task from a KB-3 temporal alert
func (s *Server) CreateFromTemporalAlert(c *gin.Context) {
	var alert struct {
		AlertID      string `json:"alert_id" binding:"required"`
		PatientID    string `json:"patient_id" binding:"required"`
		EncounterID  string `json:"encounter_id"`
		ProtocolID   string `json:"protocol_id"`
		ProtocolName string `json:"protocol_name"`
		Action       string `json:"action" binding:"required"`
		Severity     string `json:"severity" binding:"required"`
		Deadline     string `json:"deadline"`
		Description  string `json:"description"`
	}

	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Convert to TemporalAlert struct
	temporalAlert := &models.TemporalAlert{
		AlertID:      alert.AlertID,
		PatientID:    alert.PatientID,
		EncounterID:  alert.EncounterID,
		ProtocolID:   alert.ProtocolID,
		ProtocolName: alert.ProtocolName,
		Action:       alert.Action,
		Severity:     alert.Severity,
		Description:  alert.Description,
	}

	task, err := s.taskFactory.CreateFromTemporalAlertModel(c.Request.Context(), temporalAlert)
	if err != nil {
		s.log.WithError(err).Error("Failed to create task from temporal alert")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// CreateFromCarePlan creates a task from a KB-12 care plan activity
func (s *Server) CreateFromCarePlan(c *gin.Context) {
	var activity struct {
		ActivityID  string `json:"activity_id" binding:"required"`
		CarePlanID  string `json:"care_plan_id" binding:"required"`
		PatientID   string `json:"patient_id" binding:"required"`
		Type        string `json:"type" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Status      string `json:"status"`
		DueDate     string `json:"due_date"`
	}

	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Convert to CarePlanActivity struct
	carePlanActivity := &models.CarePlanActivity{
		ActivityID:  activity.ActivityID,
		CarePlanID:  activity.CarePlanID,
		PatientID:   activity.PatientID,
		Type:        activity.Type,
		Title:       activity.Title,
		Description: activity.Description,
		Status:      activity.Status,
	}

	task, err := s.taskFactory.CreateFromCarePlanActivityModel(c.Request.Context(), carePlanActivity)
	if err != nil {
		s.log.WithError(err).Error("Failed to create task from care plan")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}

// CreateFromProtocol creates a task from a KB-12 protocol step
func (s *Server) CreateFromProtocol(c *gin.Context) {
	var protocol struct {
		StepID       string `json:"step_id" binding:"required"`
		ProtocolID   string `json:"protocol_id" binding:"required"`
		ProtocolName string `json:"protocol_name"`
		PatientID    string `json:"patient_id" binding:"required"`
		EncounterID  string `json:"encounter_id"`
		StepType     string `json:"step_type" binding:"required"`
		Title        string `json:"title" binding:"required"`
		Description  string `json:"description"`
		DueDate      string `json:"due_date"`
	}

	if err := c.ShouldBindJSON(&protocol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Convert to ProtocolStep struct
	protocolStep := &models.ProtocolStep{
		StepID:       protocol.StepID,
		ProtocolID:   protocol.ProtocolID,
		ProtocolName: protocol.ProtocolName,
		PatientID:    protocol.PatientID,
		EncounterID:  protocol.EncounterID,
		StepType:     protocol.StepType,
		Title:        protocol.Title,
		Description:  protocol.Description,
	}

	task, err := s.taskFactory.CreateFromProtocolStep(c.Request.Context(), protocolStep)
	if err != nil {
		s.log.WithError(err).Error("Failed to create task from protocol")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.TaskResponse{
		Success: true,
		Data:    task,
	})
}
