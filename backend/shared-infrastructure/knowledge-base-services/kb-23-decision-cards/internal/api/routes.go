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

		// Gate Management (Phase 4)
		v1.POST("/cards/:id/mcu-gate-resume", s.handleMCUGateResume)
	}
}
