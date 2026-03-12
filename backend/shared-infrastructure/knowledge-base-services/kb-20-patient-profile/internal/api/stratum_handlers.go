package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getStratum(c *gin.Context) {
	patientID := c.Param("id")
	nodeID := c.Param("node_id")

	response, err := s.stratumEngine.GetStratum(patientID, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}
