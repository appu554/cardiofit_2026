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

		// Perturbation analysis
		v1.POST("/perturbation", s.perturbationAnalysis)

		// Confidence
		v1.GET("/twin/:patientId/confidence", s.getTwinConfidence)

		// Webhooks
		v1.POST("/events/observation", s.webhookObservation)
		v1.POST("/events/checkin", s.webhookCheckin)
		v1.POST("/events/med-change", s.webhookMedChange)

		// MRI (Metabolic Risk Index)
		v1.GET("/mri/:patientId", s.getMRI)
		v1.GET("/mri/:patientId/history", s.getMRIHistory)
		v1.GET("/mri/:patientId/decomposition", s.getMRIDecomposition)
		v1.GET("/mri/:patientId/domain-trajectory", s.getDomainTrajectory)
		v1.POST("/mri/simulate", s.simulateMRI)

		// BP context classification (masked HTN / white-coat / true HTN)
		v1.POST("/bp-context/:patientId", s.classifyBPContext)

		// Phase 7 P7-D: per-domain target status for KB-23's inertia
		// input assembler. Stateless compute — caller supplies HbA1c /
		// SBP / eGFR in the POST body, KB-26 returns DomainTargetStatusResult
		// for GLYCAEMIC, HEMODYNAMIC, and RENAL domains.
		v1.POST("/target-status/:patientId", s.getTargetStatus)

		// Phase 7 P7-E Milestone 2: latest CGM period report for a
		// patient. KB-23's inertia assembler uses this to populate the
		// CGM_TIR branch of the glycaemic domain inertia input.
		v1.GET("/cgm-latest/:patientId", s.getCGMLatest)

		// PREVENT (10-year CVD risk)
		v1.GET("/prevent/:patientId", s.getPREVENT)

		// Relapse detection (Patient Engagement Loop)
		relapse := v1.Group("/relapse/:patientId")
		{
			relapse.GET("/nadir", s.getNadir)
			relapse.POST("/check", s.checkRelapse)
			relapse.GET("/history", s.getRelapseHistory)
		}
	}
}
