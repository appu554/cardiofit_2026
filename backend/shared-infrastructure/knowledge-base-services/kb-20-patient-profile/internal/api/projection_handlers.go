package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getChannelBInputs serves the FactStore Channel B projection for V-MCU.
// GET /api/v1/patient/:id/channel-b-inputs
func (s *Server) getChannelBInputs(c *gin.Context) {
	patientID := c.Param("id")

	projection, err := s.projectionService.GetChannelBProjection(patientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": projection})
}

// getChannelCInputs serves the FactStore Channel C projection for V-MCU.
// GET /api/v1/patient/:id/channel-c-inputs
func (s *Server) getChannelCInputs(c *gin.Context) {
	patientID := c.Param("id")

	projection, err := s.projectionService.GetChannelCProjection(patientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": projection})
}

// invalidateProjectionCache busts both Channel B and C projection caches.
// DELETE /api/v1/patient/:id/projections/cache
func (s *Server) invalidateProjectionCache(c *gin.Context) {
	patientID := c.Param("id")

	s.projectionService.InvalidateProjectionCache(patientID)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "projection cache invalidated"})
}

// getLOINCRegistry returns the KB-7 verified LOINC code mappings used by FactStore.
// GET /api/v1/loinc/registry
func (s *Server) getLOINCRegistry(c *gin.Context) {
	if s.loincRegistry == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LOINC registry not initialized"})
		return
	}

	mappings := s.loincRegistry.AllMappings()
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"ready":    s.loincRegistry.IsReady(),
		"summary":  s.loincRegistry.VerificationSummary(),
		"mappings": mappings,
	})
}
