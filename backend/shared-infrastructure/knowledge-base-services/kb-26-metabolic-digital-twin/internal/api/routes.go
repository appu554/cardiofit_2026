package api

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/gin-gonic/gin"
)

func (s *Server) setupRoutes() {
	// Infrastructure endpoints
	s.Router.GET("/health", s.healthCheck)
	s.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1
	v1 := s.Router.Group("/api/v1/kb26")
	{
		// Twin state CRUD
		v1.GET("/twin/:patientId", s.getTwin)
		v1.GET("/twin/:patientId/history", s.getTwinHistory)

		// Sync / re-derive
		v1.POST("/sync/:patientId", s.syncTwin)

		// Simulation
		v1.POST("/simulate", s.simulate)
		v1.POST("/simulate-comparison", s.simulateComparison)

		// Calibration
		v1.POST("/calibrate", s.calibrate)

		// Confidence
		v1.GET("/twin/:patientId/confidence", s.getTwinConfidence)

		// Webhooks
		v1.POST("/events/observation", s.webhookObservation)
		v1.POST("/events/checkin", s.webhookCheckin)
		v1.POST("/events/med-change", s.webhookMedChange)
	}
}
