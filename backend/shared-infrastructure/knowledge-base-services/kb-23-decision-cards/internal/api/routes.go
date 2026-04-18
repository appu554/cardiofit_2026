package api

// RegisterRoutes sets up all KB-23 API routes.
func (s *Server) RegisterRoutes() {
	// Infrastructure
	s.Router.GET("/health", s.handleHealth)
	s.Router.GET("/readiness", s.handleReadiness)
	s.Router.GET("/metrics", s.handleMetrics)
	s.Router.POST("/internal/templates/reload", s.handleTemplateReload)

	// API v1
	v1 := s.Router.Group("/api/v1")
	{
		// Card Generation & Management (Phase 2)
		v1.POST("/decision-cards", s.handleGenerateCard)
		v1.GET("/cards/:id", s.handleGetCard)

		// Patient endpoints (Phase 2)
		v1.GET("/patients/:id/active-cards", s.handleGetActiveCards)
		v1.GET("/patients/:id/mcu-gate", s.handleGetMCUGate)

		// Safety Fast-Paths (Phase 2 minimal handlers)
		v1.POST("/safety/hypoglycaemia-alert", s.handleHypoglycaemiaAlert)
		v1.POST("/safety/behavioral-gap-alert", s.handleBehavioralGapAlert)

		// Treatment Perturbation (Phase 3)
		v1.POST("/perturbations", s.handleCreatePerturbation)
		v1.GET("/perturbations/:patient_id/active", s.handleGetActivePerturbations)

		// Configuration endpoints (consumed by Flink)
		v1.GET("/config/risk-scoring", s.getRiskScoringConfig)

		// Clinical Signal Processing (KB-22 SignalPublisher)
		v1.POST("/clinical-signals", s.handleClinicalSignal)

		// Gate Management (Phase 4)
		v1.POST("/cards/:id/mcu-gate-resume", s.handleMCUGateResume)

		// Composite Card Synthesis trigger (Phase 4 P9)
		// Called by KB-26 after each BP context classification to fold
		// active cards (masked HTN, medication timing, selection bias,
		// etc.) into a single CompositeCardSignal.
		v1.POST("/composite-cards/synthesize/:patientId", s.handleCompositeSynthesize)

		// Phase 10 Gap 10: explainability endpoint — returns the
		// full evidence trail for a decision card. Clinicians can
		// ask "why did the system recommend this?" and get a
		// structured answer covering template selection, confidence
		// tier, MCU gate rationale, safety checks, and patient state.
		v1.GET("/cards/:id/explainability", s.handleGetExplainability)

		// Escalation Protocol Engine (Gap 15)
		escalation := v1.Group("/escalation")
		{
			escalation.POST("/:id/acknowledge", s.acknowledgeEscalation)
			escalation.POST("/:id/action", s.recordEscalationAction)
			escalation.GET("/patient/:patientId", s.getPatientEscalations)
			escalation.GET("/metrics", s.getEscalationMetrics)
		}
		// Clinician preferences
		v1.POST("/clinician/:clinicianId/preferences", s.upsertClinicianPreferences)

		// Gap 18: Clinician Worklist
		worklist := v1.Group("/worklist")
		{
			worklist.GET("", s.getWorklist)
			worklist.POST("/action", s.handleWorklistAction)
			worklist.POST("/feedback", s.recordWorklistFeedback)
		}

		// Gap 19: Time-to-Response Tracking
		tracking := v1.Group("/tracking")
		{
			tracking.GET("/detection/:id", s.getDetectionLifecycle)
			tracking.GET("/patient/:patientId", s.getPatientLifecycles)
		}
		responseMetrics := v1.Group("/metrics")
		{
			responseMetrics.GET("/clinician/:clinicianId", s.getClinicianMetrics)
			responseMetrics.GET("/system", s.getSystemMetrics)
			responseMetrics.GET("/pilot", s.getPilotMetrics)
		}
	}
}
