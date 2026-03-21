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

	fhir := s.Router.Group("/fhir")
	{
		// Patient CRUD
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

		// $operations - Enrollment
		fhir.POST("/Patient/$enroll", s.stubHandler("Enroll Patient"))
		fhir.POST("/Patient/:id/$verify-otp", s.stubHandler("Verify OTP"))
		fhir.POST("/Patient/:id/$link-abha", s.stubHandler("Link ABHA"))

		// $operations - Safety
		fhir.POST("/Patient/:id/$evaluate-safety", s.stubHandler("Evaluate Safety"))
		fhir.POST("/Encounter/:id/$fill-slot", s.stubHandler("Fill Slot"))

		// $operations - Review
		fhir.POST("/Encounter/:id/$submit-review", s.stubHandler("Submit Review"))
		fhir.POST("/Encounter/:id/$approve", s.stubHandler("Approve"))
		fhir.POST("/Encounter/:id/$request-clarification", s.stubHandler("Request Clarification"))
		fhir.POST("/Encounter/:id/$escalate", s.stubHandler("Escalate"))

		// $operations - Check-in
		fhir.POST("/Patient/:id/$checkin", s.stubHandler("Start Checkin"))
		fhir.POST("/Encounter/:id/$checkin-slot", s.stubHandler("Fill Checkin Slot"))

		// $operations - Co-enrollee
		fhir.POST("/Patient/:id/$register-co-enrollee", s.stubHandler("Register Co-enrollee"))
	}
}

func (s *Server) stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"status":   "not_implemented",
			"endpoint": name,
			"message":  "This endpoint will be implemented in Phase 3",
		})
	}
}
