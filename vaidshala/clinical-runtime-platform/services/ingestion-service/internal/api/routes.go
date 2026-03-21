package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) setupRoutes() {
	// Infrastructure
	s.Router.GET("/healthz", s.handleHealthz)
	s.Router.GET("/readyz", s.handleReadyz)
	s.Router.GET("/startupz", s.handleStartupz)
	s.Router.GET("/metrics", s.prometheusHandler())

	// FHIR-compliant inbound -- Phase 2 real handlers
	fhirGroup := s.Router.Group("/fhir")
	{
		fhirGroup.POST("", s.stubHandler("FHIR Transaction Bundle"))
		fhirGroup.POST("/Observation", s.handleFHIRObservation)
		fhirGroup.POST("/DiagnosticReport", s.stubHandler("FHIR DiagnosticReport"))
		fhirGroup.POST("/MedicationStatement", s.stubHandler("FHIR MedicationStatement"))

		// DLQ endpoints
		fhirGroup.GET("/OperationOutcome", s.dlqReplay.HandleListPending)
		fhirGroup.POST("/OperationOutcome/:id/$replay", s.dlqReplay.HandleReplay)
	}

	// Source-specific receivers
	ingest := s.Router.Group("/ingest")
	{
		ingest.POST("/ehr/hl7v2", s.stubHandler("HL7v2 ingest"))                   // Phase 4
		ingest.POST("/ehr/fhir", s.stubHandler("FHIR passthrough"))                // Phase 4
		ingest.POST("/labs/:labId", s.stubHandler("Lab ingest"))                    // Phase 4
		ingest.POST("/devices", s.handleDeviceIngest)                               // Phase 2
		ingest.POST("/app-checkin", s.handleAppCheckin)                             // Phase 2
		ingest.POST("/whatsapp", s.handleWhatsAppIngest)                            // Phase 2
		ingest.POST("/wearables/:provider", s.stubHandler("Wearable ingest"))       // Phase 4
		ingest.POST("/abdm/data-push", s.stubHandler("ABDM data push"))            // Phase 4
	}

	// Internal (service-to-service)
	internal := s.Router.Group("/internal")
	{
		internal.POST("/hpi", s.handleHPIIngest) // Phase 2
	}

	// Admin/Dashboard
	s.Router.GET("/$source-status", s.stubHandler("Source status"))
}

// stubHandler returns a 501 Not Implemented response with the endpoint name.
// These stubs are replaced with real handlers in later phases.
func (s *Server) stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"status":   "not_implemented",
			"endpoint": name,
			"message":  "This endpoint will be implemented in a future phase",
		})
	}
}
