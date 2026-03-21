package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) setupRoutes() {
	s.Router.GET("/healthz", s.handleHealthz)
	s.Router.GET("/readyz", s.handleReadyz)
	s.Router.GET("/startupz", s.handleStartupz)
	s.Router.GET("/metrics", s.prometheusHandler())

	fhir := s.Router.Group("/fhir")
	{
		fhir.POST("", s.stubHandler("FHIR Transaction Bundle"))
		fhir.POST("/Observation", s.stubHandler("FHIR Observation"))
		fhir.POST("/DiagnosticReport", s.stubHandler("FHIR DiagnosticReport"))
		fhir.POST("/MedicationStatement", s.stubHandler("FHIR MedicationStatement"))
	}

	ingest := s.Router.Group("/ingest")
	{
		ingest.POST("/ehr/hl7v2", s.stubHandler("HL7v2 ingest"))
		ingest.POST("/ehr/fhir", s.stubHandler("FHIR passthrough"))
		ingest.POST("/labs/:labId", s.stubHandler("Lab ingest"))
		ingest.POST("/devices", s.stubHandler("Device ingest"))
		ingest.POST("/wearables/:provider", s.stubHandler("Wearable ingest"))
		ingest.POST("/abdm/data-push", s.stubHandler("ABDM data push"))
	}

	internal := s.Router.Group("/internal")
	{
		internal.POST("/hpi", s.stubHandler("HPI slot data from Intake"))
	}

	s.Router.GET("/$source-status", s.stubHandler("Source status"))
}

func (s *Server) stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"status":   "not_implemented",
			"endpoint": name,
			"message":  "This endpoint will be implemented in Phase 2",
		})
	}
}
