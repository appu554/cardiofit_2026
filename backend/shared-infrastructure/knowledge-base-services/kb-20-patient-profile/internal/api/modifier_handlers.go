package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getModifierRegistry(c *gin.Context) {
	nodeID := c.Param("node_id")

	modifiers, err := s.cmRegistry.GetRegistryForNode(nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": modifiers})
}
