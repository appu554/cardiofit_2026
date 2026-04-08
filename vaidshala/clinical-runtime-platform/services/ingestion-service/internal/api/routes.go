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

	// Source-specific receivers (mounted at root — gateway strips /api/v1/ingest prefix)
	s.Router.POST("/ehr/hl7v2", s.ehrHandler.HandleHL7v2)                   // Phase 4 (MLLP stub)
	s.Router.POST("/ehr/fhir", s.ehrHandler.HandleFHIRPassthrough)          // Phase 4
	s.Router.POST("/labs/:labId", s.labHandler.HandleLabWebhook)            // Phase 4
	s.Router.POST("/devices", s.handleDeviceIngest)                         // Phase 2
	s.Router.POST("/app-checkin", s.handleAppCheckin)                       // Phase 2
	s.Router.POST("/whatsapp", s.handleWhatsAppIngest)                      // Phase 2
	s.Router.POST("/wearables/:provider", s.handleWearableIngest) // Phase 5 — pipeline-integrated
	if s.abdmHandler != nil {
		s.Router.POST("/abdm/data-push", s.abdmHandler.HandleDataPush)  // Phase 4
	} else {
		s.Router.POST("/abdm/data-push", s.stubHandler("ABDM data push — configure X25519 keys")) // Phase 4
	}


	// Admin/Dashboard
	s.Router.GET("/$source-status", s.stubHandler("Source status"))

	// Phase 5: DLQ resolver admin endpoints
	admin := s.Router.Group("/admin")
	{
		admin.GET("/dlq", s.handleDLQList)
		admin.GET("/dlq/:id", s.handleDLQGet)
		admin.POST("/dlq/:id/$discard", s.handleDLQDiscard)
		admin.GET("/dlq/$count", s.handleDLQCount)
	}
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
