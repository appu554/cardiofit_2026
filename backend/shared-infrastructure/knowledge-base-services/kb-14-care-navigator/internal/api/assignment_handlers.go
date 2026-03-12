// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-14-care-navigator/internal/models"
)

// SuggestAssignees suggests the best assignees for a task
func (s *Server) SuggestAssignees(c *gin.Context) {
	taskIDStr := c.Query("taskId")
	if taskIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "taskId is required",
		})
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid task ID format",
		})
		return
	}

	// Get the task
	task, err := s.taskService.GetByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "task not found",
		})
		return
	}

	// Build assignment criteria
	criteria := &models.AssignmentCriteria{
		TaskType:      task.Type,
		TaskPriority:  task.Priority,
		PatientID:     task.PatientID,
		RequiredRole:  task.AssignedRole,
		PreferredTeam: task.TeamID,
	}

	// Get suggestions
	suggestions, err := s.assignmentEngine.SuggestAssignees(c.Request.Context(), criteria)
	if err != nil {
		s.log.WithError(err).Error("Failed to suggest assignees")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"task_id":     taskID,
		"suggestions": suggestions,
	})
}

// BulkAssign assigns multiple tasks to a single assignee
func (s *Server) BulkAssign(c *gin.Context) {
	var req models.BulkAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if len(req.TaskIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "at least one task ID is required",
		})
		return
	}

	response, err := s.assignmentEngine.BulkAssign(c.Request.Context(), &req)
	if err != nil {
		s.log.WithError(err).Error("Failed to bulk assign tasks")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetWorkload retrieves workload information for a team member
func (s *Server) GetWorkload(c *gin.Context) {
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid member ID format",
		})
		return
	}

	workload, err := s.assignmentEngine.GetWorkload(c.Request.Context(), memberID)
	if err != nil {
		s.log.WithError(err).Error("Failed to get workload")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    workload,
	})
}
