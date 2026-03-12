// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/cache"
	"kb-14-care-navigator/internal/clients"
	"kb-14-care-navigator/internal/config"
	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/fhir"
	"kb-14-care-navigator/internal/services"
)

// Server represents the HTTP server
type Server struct {
	config    *config.Config
	router    *gin.Engine
	server    *http.Server
	log       *logrus.Entry

	// Repositories
	taskRepo       *database.TaskRepository
	teamRepo       *database.TeamRepository
	escalationRepo *database.EscalationRepository

	// Services
	taskService         *services.TaskService
	taskFactory         *services.TaskFactory
	assignmentEngine    *services.AssignmentEngine
	escalationEngine    *services.EscalationEngine
	worklistService     *services.WorklistService
	analyticsService    *services.AnalyticsService
	notificationService *services.NotificationService

	// Clients
	kb3Client *clients.KB3Client
	kb9Client *clients.KB9Client
	kb12Client *clients.KB12Client

	// Cache
	redisCache *cache.RedisCache

	// FHIR
	fhirMapper *fhir.TaskMapper
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config) (*Server, error) {
	log := logrus.WithField("component", "api-server")

	// Set Gin mode
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	db, err := database.NewConnection(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize repositories
	taskRepo := database.NewTaskRepository(db, log)
	teamRepo := database.NewTeamRepository(db, log)
	escalationRepo := database.NewEscalationRepository(db, log)
	auditRepo := database.NewAuditRepository(db.DB)
	governanceRepo := database.NewGovernanceRepository(db.DB)
	reasonCodeRepo := database.NewReasonCodeRepository(db.DB)
	intelligenceRepo := database.NewIntelligenceRepository(db.DB)

	// Initialize KB clients
	kb3Client := clients.NewKB3Client(cfg.KBServices.KB3Temporal)
	kb9Client := clients.NewKB9Client(cfg.KBServices.KB9CareGaps)
	kb12Client := clients.NewKB12Client(cfg.KBServices.KB12OrderSets)

	// Initialize notification service (stub)
	notificationService := services.NewNotificationService(log)

	// Initialize governance service
	governanceService := services.NewGovernanceService(auditRepo, governanceRepo, reasonCodeRepo, intelligenceRepo, log)

	// Initialize services
	taskService := services.NewTaskService(taskRepo, teamRepo, escalationRepo, governanceService, log)
	taskFactory := services.NewTaskFactory(taskService, kb3Client, kb9Client, kb12Client, log)
	assignmentEngine := services.NewAssignmentEngine(taskRepo, teamRepo, log)
	escalationEngine := services.NewEscalationEngine(taskRepo, teamRepo, escalationRepo, notificationService, log)
	worklistService := services.NewWorklistService(taskRepo, teamRepo, log)
	analyticsService := services.NewAnalyticsService(taskRepo, teamRepo, escalationRepo, log)

	// Initialize Redis cache (optional)
	var redisCache *cache.RedisCache
	if cfg.Redis.URL != "" {
		redisCache, err = cache.NewRedisCache(cfg.Redis)
		if err != nil {
			log.WithError(err).Warn("Failed to connect to Redis, caching disabled")
		}
	}

	// Initialize FHIR mapper
	baseURL := fmt.Sprintf("http://localhost:%s", cfg.Server.Port)
	fhirMapper := fhir.NewTaskMapper(baseURL)

	// Create server
	s := &Server{
		config:              cfg,
		log:                 log,
		taskRepo:            taskRepo,
		teamRepo:            teamRepo,
		escalationRepo:      escalationRepo,
		taskService:         taskService,
		taskFactory:         taskFactory,
		assignmentEngine:    assignmentEngine,
		escalationEngine:    escalationEngine,
		worklistService:     worklistService,
		analyticsService:    analyticsService,
		notificationService: notificationService,
		kb3Client:           kb3Client,
		kb9Client:           kb9Client,
		kb12Client:          kb12Client,
		redisCache:          redisCache,
		fhirMapper:          fhirMapper,
	}

	// Setup router
	s.setupRouter()

	return s, nil
}

// setupRouter configures all routes
func (s *Server) setupRouter() {
	router := gin.New()

	// Apply middleware
	router.Use(gin.Recovery())
	router.Use(RequestLogger(s.log))
	router.Use(CORSMiddleware())
	router.Use(RequestIDMiddleware())

	// Health endpoints
	router.GET("/health", s.healthCheck)
	router.GET("/ready", s.readinessCheck)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Task routes
		tasks := v1.Group("/tasks")
		{
			tasks.POST("", s.CreateTask)
			tasks.GET("", s.ListTasks)
			tasks.GET("/:id", s.GetTask)
			tasks.PATCH("/:id", s.UpdateTask)
			tasks.DELETE("/:id", s.DeleteTask)

			// Task actions
			tasks.POST("/:id/assign", s.AssignTask)
			tasks.POST("/:id/start", s.StartTask)
			tasks.POST("/:id/complete", s.CompleteTask)
			tasks.POST("/:id/cancel", s.CancelTask)
			tasks.POST("/:id/escalate", s.EscalateTask)
			tasks.POST("/:id/add-note", s.AddNote)

			// Task creation from sources
			tasks.POST("/from-care-gap", s.CreateFromCareGap)
			tasks.POST("/from-temporal-alert", s.CreateFromTemporalAlert)
			tasks.POST("/from-care-plan", s.CreateFromCarePlan)
			tasks.POST("/from-protocol", s.CreateFromProtocol)
		}

		// Worklist routes
		worklist := v1.Group("/worklist")
		{
			worklist.GET("", s.GetWorklist)
			worklist.GET("/user/:userId", s.GetUserWorklist)
			worklist.GET("/team/:teamId", s.GetTeamWorklist)
			worklist.GET("/patient/:patientId", s.GetPatientWorklist)
			worklist.GET("/overdue", s.GetOverdueWorklist)
			worklist.GET("/urgent", s.GetUrgentWorklist)
			worklist.GET("/unassigned", s.GetUnassignedWorklist)
			worklist.GET("/summary", s.GetWorklistSummary)
		}

		// Assignment routes
		assignment := v1.Group("/assignment")
		{
			assignment.GET("/suggest", s.SuggestAssignees)
			assignment.POST("/bulk-assign", s.BulkAssign)
			assignment.GET("/workload/:memberId", s.GetWorkload)
		}

		// Analytics routes
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/dashboard", s.GetDashboardMetrics)
			analytics.GET("/sla", s.GetSLAMetrics)
			analytics.GET("/trends", s.GetTrendMetrics)
			analytics.GET("/care-gaps", s.GetCareGapAnalytics)
		}

		// Sync routes
		sync := v1.Group("/sync")
		{
			sync.POST("/kb3", s.SyncKB3)
			sync.POST("/kb9", s.SyncKB9)
			sync.POST("/kb12", s.SyncKB12)
			sync.POST("/all", s.SyncAll)
		}

		// Team routes
		teams := v1.Group("/teams")
		{
			teams.POST("", s.CreateTeam)
			teams.GET("", s.ListTeams)
			teams.GET("/:id", s.GetTeam)
			teams.PATCH("/:id", s.UpdateTeam)
			teams.DELETE("/:id", s.DeleteTeam)

			// Team members
			teams.POST("/:id/members", s.AddTeamMember)
			teams.GET("/:id/members", s.ListTeamMembers)
			teams.DELETE("/:id/members/:memberId", s.RemoveTeamMember)
		}

		// Escalation routes
		escalations := v1.Group("/escalations")
		{
			escalations.GET("", s.ListEscalations)
			escalations.GET("/:id", s.GetEscalation)
			escalations.POST("/:id/acknowledge", s.AcknowledgeEscalation)
			escalations.POST("/:id/resolve", s.ResolveEscalation)
		}
	}

	// FHIR routes
	fhirGroup := router.Group("/fhir")
	{
		fhirGroup.GET("/Task", s.SearchFHIRTasks)
		fhirGroup.GET("/Task/:id", s.GetFHIRTask)
		fhirGroup.POST("/Task", s.CreateFHIRTask)
		fhirGroup.PUT("/Task/:id", s.UpdateFHIRTask)
	}

	s.router = router
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.config.Server.Port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.log.WithField("port", s.config.Server.Port).Info("Starting KB-14 Care Navigator server")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down server...")

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	}

	if s.redisCache != nil {
		if err := s.redisCache.Close(); err != nil {
			s.log.WithError(err).Warn("Failed to close Redis connection")
		}
	}

	s.log.Info("Server shutdown complete")
	return nil
}

// GetEscalationEngine returns the escalation engine for workers
func (s *Server) GetEscalationEngine() *services.EscalationEngine {
	return s.escalationEngine
}

// GetTaskFactory returns the task factory for workers
func (s *Server) GetTaskFactory() *services.TaskFactory {
	return s.taskFactory
}

// GetKB3Client returns the KB-3 client for workers
func (s *Server) GetKB3Client() *clients.KB3Client {
	return s.kb3Client
}

// GetKB9Client returns the KB-9 client for workers
func (s *Server) GetKB9Client() *clients.KB9Client {
	return s.kb9Client
}

// GetKB12Client returns the KB-12 client for workers
func (s *Server) GetKB12Client() *clients.KB12Client {
	return s.kb12Client
}

// GetTaskRepo returns the task repository for workers
func (s *Server) GetTaskRepo() *database.TaskRepository {
	return s.taskRepo
}

// GetEscalationRepo returns the escalation repository for workers
func (s *Server) GetEscalationRepo() *database.EscalationRepository {
	return s.escalationRepo
}

// GetRedisCache returns the Redis cache for workers
func (s *Server) GetRedisCache() *cache.RedisCache {
	return s.redisCache
}

// healthCheck returns health status
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "kb-14-care-navigator",
		"version": "1.0.0",
	})
}

// readinessCheck checks if the service is ready
func (s *Server) readinessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check database
	if err := s.taskRepo.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database connection failed",
		})
		return
	}

	// Check Redis if enabled
	if s.redisCache != nil {
		if err := s.redisCache.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "redis connection failed",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"database": "connected",
		"redis":    s.redisCache != nil,
	})
}
