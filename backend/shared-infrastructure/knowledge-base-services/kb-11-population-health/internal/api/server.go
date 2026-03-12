// Package api provides the HTTP API for KB-11 Population Health Engine.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/analytics"
	"github.com/cardiofit/kb-11-population-health/internal/api/handlers"
	"github.com/cardiofit/kb-11-population-health/internal/api/middleware"
	"github.com/cardiofit/kb-11-population-health/internal/cohort"
	"github.com/cardiofit/kb-11-population-health/internal/config"
	"github.com/cardiofit/kb-11-population-health/internal/database"
	"github.com/cardiofit/kb-11-population-health/internal/projection"
	"github.com/cardiofit/kb-11-population-health/internal/risk"
)

// Server represents the HTTP API server.
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	config     *config.Config
	logger     *logrus.Entry
	startTime  time.Time
}

// NewServer creates a new API server.
func NewServer(
	cfg *config.Config,
	db *database.DB,
	projService *projection.Service,
	riskEngine *risk.Engine,
	cohortService *cohort.Service,
	analyticsEngine *analytics.Engine,
	logger *logrus.Entry,
) *Server {
	// Set Gin mode
	if cfg.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Apply middleware
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())

	// Create handlers
	healthHandler := handlers.NewHealthHandler(db, cfg, logger)
	projectionHandler := handlers.NewProjectionHandler(projService, logger)
	metricsHandler := handlers.NewMetricsHandler(projService, logger)
	syncHandler := handlers.NewSyncHandler(projService, logger)
	riskHandler := handlers.NewRiskHandler(riskEngine, logger)
	cohortHandler := handlers.NewCohortHandler(cohortService, logger)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsEngine, logger)

	// Register routes
	registerRoutes(router, cfg, healthHandler, projectionHandler, metricsHandler, syncHandler, riskHandler, cohortHandler, analyticsHandler)

	server := &Server{
		router: router,
		config: cfg,
		logger: logger.WithField("component", "server"),
	}

	return server
}

// registerRoutes sets up all API routes.
func registerRoutes(
	router *gin.Engine,
	cfg *config.Config,
	health *handlers.HealthHandler,
	projection *handlers.ProjectionHandler,
	metrics *handlers.MetricsHandler,
	sync *handlers.SyncHandler,
	riskHandler *handlers.RiskHandler,
	cohortHandler *handlers.CohortHandler,
	analyticsHandler *handlers.AnalyticsHandler,
) {
	// Health and readiness endpoints
	router.GET("/health", health.Health)
	router.GET("/ready", health.Ready)
	router.GET("/live", health.Live)

	// Prometheus metrics
	if cfg.Metrics.Enabled {
		router.GET(cfg.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Patient projections (READ-ONLY views of synced data)
		patients := v1.Group("/patients")
		{
			patients.GET("", projection.QueryPatients)
			patients.GET("/:fhir_id", projection.GetPatient)
			patients.GET("/:fhir_id/risk", projection.GetPatientRisk)
			patients.GET("/:fhir_id/risk/history", projection.GetPatientRiskHistory)
		}

		// Attribution management (KB-11 OWNS this)
		attribution := v1.Group("/attribution")
		{
			attribution.PUT("/:fhir_id", projection.UpdateAttribution)
			attribution.POST("/batch", projection.BatchUpdateAttribution)
		}

		// Population metrics (CORE PURPOSE of KB-11)
		metricsGroup := v1.Group("/metrics")
		{
			metricsGroup.GET("/population", metrics.GetPopulationMetrics)
			metricsGroup.GET("/risk-distribution", metrics.GetRiskDistribution)
		}

		// Risk calculation (GOVERNED by KB-18)
		riskGroup := v1.Group("/risk")
		{
			riskGroup.GET("/models", riskHandler.ListModels)
			riskGroup.GET("/models/:name", riskHandler.GetModel)
			riskGroup.POST("/calculate", riskHandler.CalculateRisk)
			riskGroup.POST("/calculate/batch", riskHandler.BatchCalculateRisk)
			riskGroup.GET("/patients/:fhir_id", riskHandler.GetPatientRiskAssessments)
			riskGroup.POST("/verify", riskHandler.VerifyDeterminism)
		}

		// Cohort management (OWNED by KB-11)
		cohorts := v1.Group("/cohorts")
		{
			// CRUD operations
			cohorts.GET("", cohortHandler.ListCohorts)
			cohorts.GET("/:id", cohortHandler.GetCohort)
			cohorts.PATCH("/:id", cohortHandler.UpdateCohort)
			cohorts.DELETE("/:id", cohortHandler.DeleteCohort)

			// Create cohorts by type
			cohorts.POST("/static", cohortHandler.CreateStaticCohort)
			cohorts.POST("/dynamic", cohortHandler.CreateDynamicCohort)
			cohorts.POST("/snapshot", cohortHandler.CreateSnapshotCohort)

			// Predefined cohort creation
			predefined := cohorts.Group("/predefined")
			{
				predefined.POST("/high-risk", cohortHandler.CreateHighRiskCohort)
				predefined.POST("/rising-risk", cohortHandler.CreateRisingRiskCohort)
				predefined.POST("/care-gap", cohortHandler.CreateCareGapCohort)
				predefined.POST("/pcp", cohortHandler.CreatePCPCohort)
				predefined.POST("/practice", cohortHandler.CreatePracticeCohort)
			}

			// Membership management
			cohorts.GET("/:id/members", cohortHandler.GetMembers)
			cohorts.POST("/:id/members", cohortHandler.AddMember)
			cohorts.DELETE("/:id/members/:patientId", cohortHandler.RemoveMember)
			cohorts.GET("/:id/members/:patientId", cohortHandler.CheckMembership)

			// Refresh and analytics
			cohorts.POST("/:id/refresh", cohortHandler.RefreshCohort)
			cohorts.GET("/:id/stats", cohortHandler.GetCohortStats)
			cohorts.GET("/compare", cohortHandler.CompareCohorts)
		}

		// Sync management (READ-ONLY sync from upstream)
		syncGroup := v1.Group("/sync")
		{
			syncGroup.GET("/status", sync.GetAllSyncStatus)
			syncGroup.GET("/status/:source", sync.GetSyncStatus)
			syncGroup.POST("/fhir", sync.TriggerFHIRSync)
			syncGroup.POST("/kb17", sync.TriggerKB17Sync)
		}

		// Population Analytics (CORE PURPOSE of KB-11)
		analyticsGroup := v1.Group("/analytics")
		{
			// Population-level analytics
			analyticsGroup.GET("/population/snapshot", analyticsHandler.GetPopulationSnapshot)
			analyticsGroup.GET("/risk/stratification", analyticsHandler.GetRiskStratificationReport)

			// Provider analytics
			analyticsGroup.GET("/providers/:provider_id", analyticsHandler.GetProviderAnalytics)
			analyticsGroup.GET("/practices/:practice_id", analyticsHandler.GetPracticeAnalytics)

			// Dashboards
			dashboards := analyticsGroup.Group("/dashboard")
			{
				dashboards.GET("/executive", analyticsHandler.GetExecutiveDashboard)
				dashboards.GET("/care-manager", analyticsHandler.GetCareManagerDashboard)
			}

			// Comparisons
			compare := analyticsGroup.Group("/compare")
			{
				compare.GET("/providers", analyticsHandler.CompareProviders)
				compare.GET("/practices", analyticsHandler.ComparePractices)
			}

			// Phase D: Custom Query & Advanced Analytics
			analyticsGroup.POST("/query", analyticsHandler.ExecuteCustomQuery)
			analyticsGroup.GET("/trends", analyticsHandler.GetTrendAnalysis)
			analyticsGroup.GET("/utilization", analyticsHandler.GetUtilizationReport)
		}
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.startTime = time.Now()

	addr := fmt.Sprintf(":%d", s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	s.logger.WithFields(logrus.Fields{
		"port":        s.config.Server.Port,
		"environment": s.config.Server.Environment,
	}).Info("Starting KB-11 Population Health Engine API server")

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

// Uptime returns the server uptime.
func (s *Server) Uptime() time.Duration {
	return time.Since(s.startTime)
}
