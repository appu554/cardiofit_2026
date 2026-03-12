// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-14-care-navigator/internal/models"
)

// GetWorklist retrieves a worklist with filters
func (s *Server) GetWorklist(c *gin.Context) {
	var filters models.WorklistFilters

	// Parse query parameters
	if userID := c.Query("userId"); userID != "" {
		uid, err := uuid.Parse(userID)
		if err == nil {
			filters.UserID = &uid
		}
	}

	if teamID := c.Query("teamId"); teamID != "" {
		tid, err := uuid.Parse(teamID)
		if err == nil {
			filters.TeamID = &tid
		}
	}

	if patientID := c.Query("patientId"); patientID != "" {
		filters.PatientID = patientID
	}

	// Parse status filter (comma-separated)
	if statuses := c.QueryArray("status"); len(statuses) > 0 {
		for _, s := range statuses {
			filters.Statuses = append(filters.Statuses, models.TaskStatus(s))
		}
	}

	// Parse priority filter
	if priorities := c.QueryArray("priority"); len(priorities) > 0 {
		for _, p := range priorities {
			filters.Priorities = append(filters.Priorities, models.TaskPriority(p))
		}
	}

	// Parse type filter
	if types := c.QueryArray("type"); len(types) > 0 {
		for _, t := range types {
			filters.Types = append(filters.Types, models.TaskType(t))
		}
	}

	// Parse source filter
	if sources := c.QueryArray("source"); len(sources) > 0 {
		for _, src := range sources {
			filters.Sources = append(filters.Sources, models.TaskSource(src))
		}
	}

	// Boolean filters
	if overdue := c.Query("overdue"); overdue == "true" {
		filters.Overdue = true
	}

	if unassigned := c.Query("unassigned"); unassigned == "true" {
		filters.Unassigned = true
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	filters.Page = page
	filters.PageSize = pageSize

	// Sorting
	filters.SortBy = c.DefaultQuery("sortBy", "due_date")
	filters.SortOrder = c.DefaultQuery("sortOrder", "asc")

	// Get worklist
	response, err := s.worklistService.GetWorklist(c.Request.Context(), filters)
	if err != nil {
		s.log.WithError(err).Error("Failed to get worklist")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUserWorklist retrieves a worklist for a specific user
func (s *Server) GetUserWorklist(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid user ID format",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	response, err := s.worklistService.GetUserWorklist(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		s.log.WithError(err).Error("Failed to get user worklist")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetTeamWorklist retrieves a worklist for a team
func (s *Server) GetTeamWorklist(c *gin.Context) {
	teamIDStr := c.Param("teamId")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid team ID format",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	response, err := s.worklistService.GetTeamWorklist(c.Request.Context(), teamID, page, pageSize)
	if err != nil {
		s.log.WithError(err).Error("Failed to get team worklist")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPatientWorklist retrieves all tasks for a patient
func (s *Server) GetPatientWorklist(c *gin.Context) {
	patientID := c.Param("patientId")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "patient ID is required",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	response, err := s.worklistService.GetPatientWorklist(c.Request.Context(), patientID, page, pageSize)
	if err != nil {
		s.log.WithError(err).Error("Failed to get patient worklist")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetOverdueWorklist retrieves all overdue tasks
func (s *Server) GetOverdueWorklist(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	response, err := s.worklistService.GetOverdueWorklist(c.Request.Context(), page, pageSize)
	if err != nil {
		s.log.WithError(err).Error("Failed to get overdue worklist")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUrgentWorklist retrieves urgent tasks (CRITICAL and HIGH priority)
func (s *Server) GetUrgentWorklist(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	response, err := s.worklistService.GetUrgentWorklist(c.Request.Context(), page, pageSize)
	if err != nil {
		s.log.WithError(err).Error("Failed to get urgent worklist")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUnassignedWorklist retrieves unassigned tasks
func (s *Server) GetUnassignedWorklist(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	response, err := s.worklistService.GetUnassignedWorklist(c.Request.Context(), page, pageSize)
	if err != nil {
		s.log.WithError(err).Error("Failed to get unassigned worklist")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetWorklistSummary retrieves a summary of the worklist
func (s *Server) GetWorklistSummary(c *gin.Context) {
	var filters models.WorklistFilters

	// Parse user/team filters
	if userID := c.Query("userId"); userID != "" {
		uid, err := uuid.Parse(userID)
		if err == nil {
			filters.UserID = &uid
		}
	}

	if teamID := c.Query("teamId"); teamID != "" {
		tid, err := uuid.Parse(teamID)
		if err == nil {
			filters.TeamID = &tid
		}
	}

	summary, err := s.worklistService.GetWorklistSummary(c.Request.Context(), filters)
	if err != nil {
		s.log.WithError(err).Error("Failed to get worklist summary")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}
