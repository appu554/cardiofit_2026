// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-14-care-navigator/internal/models"
)

// CreateTeam creates a new team
func (s *Server) CreateTeam(c *gin.Context) {
	var req struct {
		TeamID            string     `json:"team_id" binding:"required"`
		Name              string     `json:"name" binding:"required"`
		Type              string     `json:"type" binding:"required"`
		ManagerID         *uuid.UUID `json:"manager_id,omitempty"`
		PanelPCPs         []string   `json:"panel_pcps,omitempty"`
		MaxTasksPerMember int        `json:"max_tasks_per_member,omitempty"`
		AutoAssign        *bool      `json:"auto_assign,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set defaults
	autoAssign := true
	if req.AutoAssign != nil {
		autoAssign = *req.AutoAssign
	}
	maxTasks := req.MaxTasksPerMember
	if maxTasks <= 0 {
		maxTasks = 20
	}

	team := &models.Team{
		TeamID:            req.TeamID,
		Name:              req.Name,
		Type:              req.Type,
		ManagerID:         req.ManagerID,
		PanelPCPs:         req.PanelPCPs,
		MaxTasksPerMember: maxTasks,
		AutoAssign:        autoAssign,
		Active:            true,
	}

	if err := s.teamRepo.CreateTeam(c.Request.Context(), team); err != nil {
		s.log.WithError(err).Error("Failed to create team")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    team,
	})
}

// GetTeam retrieves a team by ID
func (s *Server) GetTeam(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid team ID format",
		})
		return
	}

	team, err := s.teamRepo.GetTeamByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "team not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    team,
	})
}

// ListTeams retrieves all teams
func (s *Server) ListTeams(c *gin.Context) {
	teams, err := s.teamRepo.ListTeams(c.Request.Context())
	if err != nil {
		s.log.WithError(err).Error("Failed to list teams")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    teams,
		"total":   len(teams),
	})
}

// UpdateTeam updates a team
func (s *Server) UpdateTeam(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid team ID format",
		})
		return
	}

	var req struct {
		Name              *string    `json:"name"`
		Type              *string    `json:"type"`
		ManagerID         *uuid.UUID `json:"manager_id"`
		PanelPCPs         []string   `json:"panel_pcps"`
		MaxTasksPerMember *int       `json:"max_tasks_per_member"`
		AutoAssign        *bool      `json:"auto_assign"`
		Active            *bool      `json:"active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get existing team
	team, err := s.teamRepo.GetTeamByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "team not found",
		})
		return
	}

	// Apply updates
	if req.Name != nil {
		team.Name = *req.Name
	}
	if req.Type != nil {
		team.Type = *req.Type
	}
	if req.ManagerID != nil {
		team.ManagerID = req.ManagerID
	}
	if req.PanelPCPs != nil {
		team.PanelPCPs = req.PanelPCPs
	}
	if req.MaxTasksPerMember != nil {
		team.MaxTasksPerMember = *req.MaxTasksPerMember
	}
	if req.AutoAssign != nil {
		team.AutoAssign = *req.AutoAssign
	}
	if req.Active != nil {
		team.Active = *req.Active
	}

	if err := s.teamRepo.UpdateTeam(c.Request.Context(), team); err != nil {
		s.log.WithError(err).Error("Failed to update team")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    team,
	})
}

// DeleteTeam deletes a team
func (s *Server) DeleteTeam(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid team ID format",
		})
		return
	}

	if err := s.teamRepo.DeleteTeam(c.Request.Context(), id); err != nil {
		s.log.WithError(err).Error("Failed to delete team")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "team deleted",
	})
}

// AddTeamMember adds a member to a team
func (s *Server) AddTeamMember(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid team ID format",
		})
		return
	}

	var req struct {
		MemberID     string     `json:"member_id" binding:"required"`
		UserID       string     `json:"user_id" binding:"required"`
		Name         string     `json:"name" binding:"required"`
		Email        string     `json:"email"`
		Phone        string     `json:"phone"`
		Role         string     `json:"role" binding:"required"`
		Skills       []string   `json:"skills"`
		Languages    []string   `json:"languages"`
		MaxTasks     int        `json:"max_tasks"`
		SupervisorID *uuid.UUID `json:"supervisor_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set default max tasks
	if req.MaxTasks <= 0 {
		req.MaxTasks = 20
	}

	member := &models.TeamMember{
		MemberID:     req.MemberID,
		UserID:       req.UserID,
		TeamID:       teamID,
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Role:         req.Role,
		Skills:       req.Skills,
		Languages:    req.Languages,
		MaxTasks:     req.MaxTasks,
		SupervisorID: req.SupervisorID,
		Active:       true,
	}

	if err := s.teamRepo.CreateMember(c.Request.Context(), member); err != nil {
		s.log.WithError(err).Error("Failed to add team member")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    member,
	})
}

// ListTeamMembers lists all members of a team
func (s *Server) ListTeamMembers(c *gin.Context) {
	teamIDStr := c.Param("id")
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid team ID format",
		})
		return
	}

	members, err := s.teamRepo.GetMembersByTeam(c.Request.Context(), teamID)
	if err != nil {
		s.log.WithError(err).Error("Failed to list team members")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
		"total":   len(members),
	})
}

// RemoveTeamMember removes a member from a team
func (s *Server) RemoveTeamMember(c *gin.Context) {
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid member ID format",
		})
		return
	}

	if err := s.teamRepo.DeleteMember(c.Request.Context(), memberID); err != nil {
		s.log.WithError(err).Error("Failed to remove team member")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "member removed",
	})
}
