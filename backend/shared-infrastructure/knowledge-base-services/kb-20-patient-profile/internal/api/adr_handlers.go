package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getADRProfiles(c *gin.Context) {
	drugClass := c.Param("drug_class")
	includeStubs := c.Query("include_stubs") == "true"

	profiles, err := s.adrService.GetByDrugClass(drugClass, includeStubs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": profiles})
}
