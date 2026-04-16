package api

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"kb-patient-profile/internal/models"
)

// uuidPattern matches standard UUID format (accepts both FHIR UUIDs and ABHA IDs).
// ABHA IDs like "91-1001-2001-3001" do NOT match this pattern.
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// resolveFHIRPatientID is a middleware that resolves FHIR UUIDs to ABHA patient_ids.
// If the :id param is a UUID, it looks up the patient by fhir_patient_id and replaces
// the param with the ABHA patient_id. This lets KB-22/KB-23/KB-26 call KB-20 with
// the FHIR UUID they naturally have, while KB-20 internally uses ABHA IDs.
func (s *Server) resolveFHIRPatientID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id == "" || !uuidPattern.MatchString(id) {
			c.Next()
			return
		}

		// UUID detected — resolve to ABHA patient_id
		var profile models.PatientProfile
		if err := s.db.DB.Where("fhir_patient_id = ? AND active = true", id).First(&profile).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "patient not found for FHIR ID " + id})
			c.Abort()
			return
		}

		// Replace :id param while preserving all other params (node_id, med_id, etc.)
		newParams := make(gin.Params, 0, len(c.Params))
		for _, p := range c.Params {
			if p.Key == "id" {
				newParams = append(newParams, gin.Param{Key: "id", Value: profile.PatientID})
			} else {
				newParams = append(newParams, p)
			}
		}
		c.Params = newParams
		c.Next()
	}
}

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
		// Patient profile — FHIR UUID resolution middleware applied to all :id routes
		patient := v1.Group("/patient")
		patient.Use(s.resolveFHIRPatientID())
		{
			patient.POST("", s.createPatient)
			patient.GET("/:id/profile", s.getProfile)
			patient.PUT("/:id", s.updatePatient)

			// V4-7: stability-aware phenotype cluster update.
			// The Python clustering pipeline PATCHes the raw assignment;
			// the handler routes it through StabilityEngine.Evaluate
			// before writing the stable cluster to PatientProfile.
			patient.PATCH("/:id/phenotype-cluster", s.patchPhenotypeCluster)

			// Labs
			patient.POST("/:id/labs", s.addLab)
			patient.GET("/:id/labs", s.getLabs)
			patient.GET("/:id/labs/egfr", s.getEGFRHistory)
			patient.GET("/:id/bp-readings", s.listBPReadings)

			// Medications
			patient.POST("/:id/medications", s.addMedication)
			patient.PUT("/:id/medications/:med_id", s.updateMedication)
			patient.GET("/:id/medications", s.getMedications)

			// Stratum
			patient.GET("/:id/stratum/:node_id", s.getStratum)

			// FactStore projections (Phase 0)
			patient.GET("/:id/channel-b-inputs", s.getChannelBInputs)
			patient.GET("/:id/channel-c-inputs", s.getChannelCInputs)
			patient.GET("/:id/personalized-targets", s.getPersonalizedTargets)
			patient.DELETE("/:id/projections/cache", s.invalidateProjectionCache)

			// M3 Protocol lifecycle
			patient.POST("/:id/protocols", s.activateProtocol)
			patient.GET("/:id/protocols", s.getActiveProtocols)
			patient.PUT("/:id/protocols/:protocol_id/transition", s.transitionProtocolPhase)

			// Engagement season (Patient Engagement Loop)
			patient.GET("/:id/engagement-season", s.getEngagementSeason)

			// Phase 7 P7-C: renal snapshot for KB-23 decision-card generation.
			// getRenalStatus has been defined since Phase 6 but was never
			// wired into the router — activating it now that KB-23's
			// RenalAnticipatoryOrchestrator needs per-patient eGFR +
			// slope + active-med data in one round trip.
			patient.GET("/:id/renal-status", s.getRenalStatus)

			// Phase 7 P7-D: intervention timeline for KB-23's inertia
			// input assembler. Returns the latest clinical action per
			// therapeutic-inertia domain (GLYCAEMIC/HEMODYNAMIC/RENAL/
			// LIPID) so the detector can compute grace-period windows
			// against real medication events rather than hardcoded nulls.
			patient.GET("/:id/intervention-timeline", s.getInterventionTimeline)

			// Phase 8 P8-1: CRITICAL PATH summary context endpoint.
			// KB-23's KB20Client.FetchSummaryContext has been calling
			// this URL since Phase 6 but no handler existed — every
			// card-generation code path (P7-A renal reactive, P7-B
			// CKM 4c, P7-D inertia weekly, P7-E CGM TIR override)
			// silently 404'd and produced nothing for real patients.
			// This route is what converts the entire Phase 7 stack
			// from "shipped code" to "shipped clinical effect."
			patient.GET("/:id/summary-context", s.getSummaryContext)

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

		// Phase 7 P7-C: list endpoint for the renal-active patient
		// population (patients with at least one active medication in
		// the renal-sensitive drug class set). Consumed by KB-23's
		// monthly RenalAnticipatoryBatch to enumerate candidates.
		v1.GET("/patients/renal-active", s.listRenalActivePatients)

		// Phase 9 P9-B: list endpoint for patients who were actively
		// monitoring home BP and stopped (>=7 readings in a 28-day
		// window ending 14 days ago + 0 readings in the last 14 days).
		// Consumed by KB-23's MonitoringEngagementBatch weekly.
		v1.GET("/patients/monitoring-lapsed", s.listMonitoringLapsedPatients)

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

		// Lab thresholds for Flink stream enrichment and V-MCU Channel B safety
		thresholds := v1.Group("/thresholds")
		{
			thresholds.GET("/labs", s.getLabThresholds)
		}
	}
}
