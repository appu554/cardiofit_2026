package api

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// setupRoutes registers all KB-20 REST API endpoints.
func (s *Server) setupRoutes() {
	// Infrastructure endpoints
	s.Router.GET("/health", s.healthHandler)
	s.Router.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	s.Router.GET("/readiness", s.readinessHandler)

	v1 := s.Router.Group("/api/v1")
	{
		// Patient profile
		patient := v1.Group("/patient")
		{
			patient.POST("", s.createPatient)
			patient.GET("/:id/profile", s.getProfile)
			patient.PUT("/:id", s.updatePatient)

			// Labs
			patient.POST("/:id/labs", s.addLab)
			patient.GET("/:id/labs", s.getLabs)
			patient.GET("/:id/labs/egfr", s.getEGFRHistory)

			// Medications
			patient.POST("/:id/medications", s.addMedication)
			patient.PUT("/:id/medications/:med_id", s.updateMedication)
			patient.GET("/:id/medications", s.getMedications)

			// Stratum
			patient.GET("/:id/stratum/:node_id", s.getStratum)

			// FactStore projections (Phase 0)
			patient.GET("/:id/channel-b-inputs", s.getChannelBInputs)
			patient.GET("/:id/channel-c-inputs", s.getChannelCInputs)
			patient.DELETE("/:id/projections/cache", s.invalidateProjectionCache)

			// M3 Protocol lifecycle
			patient.POST("/:id/protocols", s.activateProtocol)
			patient.GET("/:id/protocols", s.getActiveProtocols)
			patient.PUT("/:id/protocols/:protocol_id/transition", s.transitionProtocolPhase)

			// Engagement season (Patient Engagement Loop)
			patient.GET("/:id/engagement-season", s.getEngagementSeason)

			// Signals — patient-reported signal ingestion (S4, S15, S16, S18-S22)
			signals := patient.Group("/:id/signals")
			{
				signals.POST("/meal", s.submitMealSignal)
				signals.POST("/activity", s.submitActivitySignal)
				signals.POST("/waist", s.submitWaistSignal)
				signals.POST("/adherence", s.submitAdherenceSignal)
				signals.POST("/symptom", s.submitSymptomSignal)
				signals.POST("/adverse-event", s.submitAdverseEventSignal)
				signals.POST("/resolution", s.submitResolutionSignal)
				signals.POST("/hospitalisation", s.submitHospitalisationSignal)
			}
		}

		// Context modifier registry
		modifiers := v1.Group("/modifiers")
		{
			modifiers.GET("/registry/:node_id", s.getModifierRegistry)
		}

		// ADR profiles
		adr := v1.Group("/adr")
		{
			adr.GET("/profiles/:drug_class", s.getADRProfiles)
		}

		// Pipeline batch write
		pipeline := v1.Group("/pipeline")
		{
			pipeline.POST("/modifiers", s.batchWriteModifiers)
			pipeline.POST("/adr-profiles", s.batchWriteADRProfiles)
		}

		// LOINC registry (KB-7 verified codes)
		loinc := v1.Group("/loinc")
		{
			loinc.GET("/registry", s.getLOINCRegistry)
		}
	}
}
