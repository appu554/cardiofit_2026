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

	// FHIR CRUD (passthrough to FHIR Store -- Phase 4 for full implementation)
	fhir := s.Router.Group("/fhir")
	{
		// Patient
		fhir.POST("/Patient", s.stubHandler("Create Patient"))
		fhir.GET("/Patient/:id", s.stubHandler("Read Patient"))
		fhir.PUT("/Patient/:id", s.stubHandler("Update Patient"))
		fhir.GET("/Patient", s.stubHandler("Search Patient"))

		// Observation
		fhir.POST("/Observation", s.stubHandler("Create Observation"))
		fhir.GET("/Observation", s.stubHandler("Search Observation"))

		// Encounter
		fhir.POST("/Encounter", s.stubHandler("Create Encounter"))
		fhir.PUT("/Encounter/:id", s.stubHandler("Update Encounter"))
		fhir.GET("/Encounter/:id", s.stubHandler("Read Encounter"))

		// Other resources
		fhir.POST("/MedicationStatement", s.stubHandler("Create MedicationStatement"))
		fhir.GET("/MedicationStatement", s.stubHandler("Search MedicationStatement"))
		fhir.GET("/DetectedIssue", s.stubHandler("Search DetectedIssue"))
		fhir.POST("/Condition", s.stubHandler("Create Condition"))
		fhir.GET("/Condition", s.stubHandler("Search Condition"))
		fhir.POST("", s.stubHandler("FHIR Transaction Bundle"))

		// -- LIVE $operations (Phase 3 -- wired to real handlers) --

		// Enrollment
		fhir.POST("/Patient/$enroll", s.appHandler.HandleEnroll)

		// Safety engine
		fhir.POST("/Patient/:id/$evaluate-safety", s.appHandler.HandleEvaluateSafety)
		fhir.POST("/Encounter/:id/$fill-slot", s.appHandler.HandleFillSlot)

		// -- STUB $operations (Phase 4-5) --
		fhir.POST("/Patient/:id/$verify-otp", s.stubHandler("Verify OTP"))
		fhir.POST("/Patient/:id/$link-abha", s.stubHandler("Link ABHA"))

		// -- LIVE Review operations (Phase 5) --
		fhir.POST("/Encounter/:id/$submit-review", s.reviewHandler.HandleSubmitReview)
		fhir.POST("/ReviewEntry/:id/$approve", s.reviewHandler.HandleApprove)
		fhir.POST("/ReviewEntry/:id/$request-clarification", s.reviewHandler.HandleRequestClarification)
		fhir.POST("/ReviewEntry/:id/$escalate", s.reviewHandler.HandleEscalate)

		// -- LIVE Check-in operations (Phase 5) --
		fhir.POST("/Patient/:id/$checkin", s.checkinHandler.HandleStartCheckin)
		fhir.POST("/CheckinSession/:id/$checkin-slot", s.checkinHandler.HandleFillCheckinSlot)

		// Co-enrollee
		fhir.POST("/Patient/:id/$register-co-enrollee", s.stubHandler("Register Co-enrollee"))
	}

	// -- Phase 4: Channel Adapters --

	// WhatsApp Business API webhook
	webhook := s.Router.Group("/webhook")
	{
		webhook.GET("/whatsapp", s.whatsappHandler.HandleVerification)
		webhook.POST("/whatsapp", s.whatsappHandler.HandleIncoming)
	}

	// ASHA tablet channel
	channel := s.Router.Group("/channel")
	{
		channel.POST("/asha/submit", s.ashaHandler.HandleBatchSubmit)
		channel.GET("/asha/sync/:deviceId", s.ashaHandler.HandleSyncStatus)
	}
}

func (s *Server) stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"status":   "not_implemented",
			"endpoint": name,
			"message":  "This endpoint will be implemented in Phase 4-5",
		})
	}
}
