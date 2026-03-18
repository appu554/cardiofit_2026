package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) activateProtocol(c *gin.Context) {
	patientID := c.Param("id")
	var req struct {
		ProtocolID string `json:"protocol_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state, err := s.protocolService.ActivateProtocol(patientID, req.ProtocolID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": state})
}

func (s *Server) getActiveProtocols(c *gin.Context) {
	patientID := c.Param("id")
	protocols, err := s.protocolService.GetActiveProtocols(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": protocols})
}

func (s *Server) transitionProtocolPhase(c *gin.Context) {
	patientID := c.Param("id")
	protocolID := c.Param("protocol_id")
	var req struct {
		NextPhase string `json:"next_phase" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state, err := s.protocolService.TransitionPhase(patientID, protocolID, req.NextPhase)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": state})
}
