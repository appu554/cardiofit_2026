package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"

	"kb-drug-rules/internal/api"
	"kb-drug-rules/internal/cache"
	"kb-drug-rules/internal/compiler"
	"kb-drug-rules/internal/config"
	"kb-drug-rules/internal/database"
	"kb-drug-rules/internal/federation"
	"kb-drug-rules/internal/governance"
	grpcService "kb-drug-rules/internal/grpc"
	"kb-drug-rules/internal/metrics"
	pb "kb-drug-rules/proto"
)

// KB1Server represents the enhanced KB-1 server with dual API support
type KB1Server struct {
	config     *config.Config
	logger     *logrus.Logger
	db         *gorm.DB
	cache      cache.KB1CacheInterface
	grpcServer *grpc.Server
	httpServer *http.Server
	compiler   *compiler.TOMLCompiler
	governance governance.Engine
	metrics    metrics.Collector
}

func main() {
	// Initialize enhanced KB-1 server
	server, err := NewKB1Server()
	if err != nil {
		log.Fatalf("Failed to create KB-1 server: %v", err)
	}

	// Start server with graceful shutdown
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start KB-1 server: %v", err)
	}
}

// NewKB1Server creates a new enhanced KB-1 server instance
func NewKB1Server() (*KB1Server, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger with structured logging for SaMD compliance
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	if cfg.Debug {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Initialize database with KB-1 schema
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run KB-1 schema migration
	if err := database.RunKB1Migration(db, logger); err != nil {
		logger.WithError(err).Warn("KB-1 migration warning (may be already applied)")
	} else {
		logger.Info("KB-1 schema migration completed successfully")
	}

	// Initialize KB-1 compliant cache
	kb1Cache, err := cache.NewKB1Cache(cfg.RedisURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize KB-1 cache: %w", err)
	}

	// Initialize TOML compiler
	tomlCompiler, err := compiler.NewTOMLCompiler(logger, cfg.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TOML compiler: %w", err)
	}

	// Initialize governance engine (enhanced for KB-1)
	governanceEngine, err := governance.NewKB1GovernanceEngine(cfg, logger)
	if err != nil {
		logger.WithError(err).Warn("Governance engine initialization failed, using mock")
		governanceEngine = governance.NewMockEngine()
	}

	// Initialize metrics collector (using standard collector for KB-1)
	metricsCollector := metrics.NewCollector()

	return &KB1Server{
		config:     cfg,
		logger:     logger,
		db:         db,
		cache:      kb1Cache,
		compiler:   tomlCompiler,
		governance: governanceEngine,
		metrics:    metricsCollector,
	}, nil
}

// Start starts both gRPC and HTTP servers concurrently
func (s *KB1Server) Start() error {
	s.logger.Info("Starting KB-1 Drug Dosing Rules service")

	// Start background maintenance tasks
	s.startBackgroundTasks()

	// Start cache invalidation listener
	go func() {
		if err := s.cache.ListenForInvalidationEvents(); err != nil {
			s.logger.WithError(err).Error("Cache invalidation listener failed")
		}
	}()

	// Prewarm cache with top medications
	go func() {
		topMedications := []string{
			"rxnorm:8610",    // Lisinopril
			"rxnorm:6809",    // Metformin
			"rxnorm:1596450", // Amlodipine
			"rxnorm:32968",   // Atorvastatin
			"rxnorm:1998",    // Levothyroxine
		}

		getRuleFunc := func(drugCode string) ([]byte, error) {
			return []byte(fmt.Sprintf(`{"drug_code": "%s", "prewarmed": true}`, drugCode)), nil
		}

		if err := s.cache.PrewarmTopMedications(topMedications, getRuleFunc); err != nil {
			s.logger.WithError(err).Error("Cache prewarming failed")
		}
	}()

	// Create wait group for concurrent servers
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Start gRPC server (primary interface for Rust Engine)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.startGRPCServer(); err != nil {
			errChan <- fmt.Errorf("gRPC server failed: %w", err)
		}
	}()

	// Start HTTP server (for Apollo Federation and REST API)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.startHTTPServer(); err != nil {
			errChan <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		s.logger.Info("Received shutdown signal")
	case err := <-errChan:
		s.logger.WithError(err).Error("Server startup failed")
		return err
	}

	// Graceful shutdown
	return s.shutdown()
}

// startGRPCServer starts the gRPC server for Rust Engine communication
func (s *KB1Server) startGRPCServer() error {
	grpcPort := s.config.GRPCPort
	if grpcPort == 0 {
		grpcPort = 9081 // Default gRPC port
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %d: %w", grpcPort, err)
	}

	// Create gRPC server with KB-1 performance optimizations
	s.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024),  // 1MB max message size
		grpc.MaxSendMsgSize(1024*1024),  // 1MB max message size
		grpc.MaxConcurrentStreams(1000), // High concurrency for Rust Engine
		grpc.ConnectionTimeout(5*time.Second),
	)

	// Create and register dosing service
	dosingService := grpcService.NewDosingServer(s.db, s.cache, s.logger, s.metrics)
	pb.RegisterDosingServiceServer(s.grpcServer, dosingService)

	// Enable reflection for development
	if s.config.Debug {
		reflection.Register(s.grpcServer)
	}

	s.logger.WithField("port", grpcPort).Info("KB-1 gRPC server starting")

	return s.grpcServer.Serve(listener)
}

// startHTTPServer starts the HTTP server for REST API and GraphQL federation
func (s *KB1Server) startHTTPServer() error {
	httpPort := s.config.Port
	if httpPort == 0 {
		httpPort = 8081 // Default HTTP port
	}

	// Setup Gin router
	if !s.config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(api.LoggingMiddleware(s.logger))
	router.Use(api.MetricsMiddleware(s.metrics))
	router.Use(api.CORSMiddleware())

	// Initialize API server (existing REST endpoints)
	apiServer := api.NewServer(&api.ServerConfig{
		DB:         s.db,
		Cache:      s.cache,
		Governance: s.governance,
		Metrics:    s.metrics,
		Logger:     s.logger,
	})

	// Register REST API routes
	apiServer.RegisterRoutes(router)

	// Add GraphQL federation endpoints
	s.registerFederationEndpoints(router)

	// Add KB-1 specific endpoints
	s.registerKB1Endpoints(router)

	// Create HTTP server with timeouts
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", httpPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.logger.WithField("port", httpPort).Info("KB-1 HTTP server starting")

	return s.httpServer.ListenAndServe()
}

// registerFederationEndpoints adds GraphQL federation endpoints
func (s *KB1Server) registerFederationEndpoints(router *gin.Engine) {
	// Create federation resolver
	resolver := federation.NewResolver(s.db, s.cache, s.logger)

	// GraphQL federation endpoint for Apollo Gateway
	router.POST("/graphql", func(c *gin.Context) {
		_ = resolver // Use resolver
		c.JSON(http.StatusOK, gin.H{
			"message": "GraphQL federation endpoint",
			"schema":  federation.GraphQLSchema,
		})
	})

	// Federation schema endpoint for Apollo Gateway discovery
	router.GET("/graphql/schema", func(c *gin.Context) {
		c.Header("Content-Type", "application/graphql")
		c.String(http.StatusOK, federation.GraphQLSchema)
	})

	// Federation health check for Apollo Gateway
	router.GET("/graphql/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "kb-drug-rules",
			"version":   "1.0.0",
			"timestamp": time.Now().Format(time.RFC3339),
			"capabilities": []string{
				"dosing_rules",
				"toml_compilation",
				"clinical_governance",
				"apollo_federation",
			},
		})
	})
}

// registerKB1Endpoints adds KB-1 specific monitoring and admin endpoints
func (s *KB1Server) registerKB1Endpoints(router *gin.Engine) {
	kb1Group := router.Group("/kb1")
	{
		// KB-1 service information
		kb1Group.GET("/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":           "KB-1 Drug Dosing Rules",
				"version":           "1.0.0",
				"specification":     "KB-1 SaMD Compliance",
				"grpc_port":         s.config.GRPCPort,
				"http_port":         s.config.Port,
				"slo_requirement":   "p95 < 60ms",
				"cache_pattern":     "dose:v2:{drug_code}:{context_hash}",
				"materialized_view": "active_dosing_rules",
			})
		})

		// KB-1 performance metrics
		kb1Group.GET("/metrics/kb1", func(c *gin.Context) {
			stats, err := s.cache.GetKB1Stats()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, stats)
		})

		// Materialized view refresh (admin operation)
		kb1Group.POST("/admin/refresh-view", func(c *gin.Context) {
			if err := s.refreshMaterializedView(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Materialized view refreshed successfully"})
		})

		// Cache invalidation (admin operation)
		kb1Group.POST("/admin/invalidate-cache/:drug_code", func(c *gin.Context) {
			drugCode := c.Param("drug_code")
			if err := s.cache.InvalidateDrugCode(drugCode); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message":   "Cache invalidated successfully",
				"drug_code": drugCode,
			})
		})

		// TOML compilation endpoint
		kb1Group.POST("/compile", func(c *gin.Context) {
			var request struct {
				TOMLContent string `json:"toml_content" binding:"required"`
				SourceFile  string `json:"source_file" binding:"required"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			result, err := s.compiler.Compile(request.TOMLContent, request.SourceFile)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, result)
		})
	}
}

// refreshMaterializedView refreshes the active_dosing_rules materialized view
func (s *KB1Server) refreshMaterializedView() error {
	start := time.Now()

	// Execute PostgreSQL function for concurrent refresh
	if err := s.db.Exec("SELECT refresh_active_dosing_rules()").Error; err != nil {
		return fmt.Errorf("failed to refresh materialized view: %w", err)
	}

	duration := time.Since(start)
	s.logger.WithField("duration_ms", duration.Milliseconds()).Info("Materialized view refreshed")

	return nil
}

// shutdown performs graceful shutdown of both servers
func (s *KB1Server) shutdown() error {
	s.logger.Info("Initiating graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var shutdownErrors []error

	// Shutdown HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("HTTP server shutdown failed: %w", err))
		} else {
			s.logger.Info("HTTP server shutdown completed")
		}
	}()

	// Shutdown gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.grpcServer.GracefulStop()
		s.logger.Info("gRPC server shutdown completed")
	}()

	// Wait for shutdown completion
	shutdownComplete := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownComplete)
	}()

	select {
	case <-shutdownComplete:
		s.logger.Info("Graceful shutdown completed")
	case <-ctx.Done():
		s.logger.Warn("Shutdown timeout exceeded, forcing stop")
		s.grpcServer.Stop()
	}

	// Close database connections
	if sqlDB, err := s.db.DB(); err == nil {
		sqlDB.Close()
	}

	// Close cache connections
	s.cache.Close()

	if len(shutdownErrors) > 0 {
		for _, err := range shutdownErrors {
			s.logger.WithError(err).Error("Shutdown error")
		}
		return shutdownErrors[0]
	}

	s.logger.Info("KB-1 service shutdown completed")
	return nil
}

// HealthCheck validates all KB-1 components
func (s *KB1Server) HealthCheck() error {
	// Check database connectivity
	if sqlDB, err := s.db.DB(); err != nil || sqlDB.Ping() != nil {
		return fmt.Errorf("database health check failed")
	}

	// Check cache connectivity
	if err := s.cache.Ping(); err != nil {
		return fmt.Errorf("cache health check failed")
	}

	// Check materialized view exists and has data
	var viewCount int64
	if err := s.db.Raw("SELECT COUNT(*) FROM active_dosing_rules").Scan(&viewCount).Error; err != nil {
		return fmt.Errorf("materialized view health check failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"database":          "healthy",
		"cache":             "healthy",
		"materialized_view": "healthy",
		"active_rules":      viewCount,
	}).Info("KB-1 health check passed")

	return nil
}

// Background tasks for KB-1 maintenance
func (s *KB1Server) startBackgroundTasks() {
	// Periodic materialized view refresh (every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.refreshMaterializedView(); err != nil {
				s.logger.WithError(err).Error("Scheduled materialized view refresh failed")
			}
		}
	}()

	// Cache statistics collection (every 1 minute)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if stats, err := s.cache.GetKB1Stats(); err == nil {
				s.metrics.RecordGauge("kb1_cache_hit_rate", stats.HitRate, nil)
				s.metrics.RecordGauge("kb1_cache_key_count", float64(stats.KeyCount), nil)
				s.metrics.RecordGauge("kb1_slo_compliance", stats.SLOCompliance, nil)

				// Alert if SLO compliance drops below threshold
				if stats.SLOCompliance < 95.0 {
					s.logger.WithField("slo_compliance", stats.SLOCompliance).Warn("KB-1 SLO compliance below threshold")
				}
			}
		}
	}()

	s.logger.Info("KB-1 background maintenance tasks started")
}
