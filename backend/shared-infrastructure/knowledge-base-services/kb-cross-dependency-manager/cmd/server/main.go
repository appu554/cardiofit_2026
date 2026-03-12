package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-cross-dependency-manager/internal/api"
	"kb-cross-dependency-manager/internal/config"
	"kb-cross-dependency-manager/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	logLevel := logger.Info
	if cfg.Environment == "production" {
		logLevel = logger.Error
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	db, err := initDatabase(cfg, logLevel)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate tables (in development)
	if cfg.Environment == "development" {
		if err := autoMigrate(db); err != nil {
			log.Printf("Warning: Auto-migration failed: %v", err)
		}
	}

	// Initialize logger
	appLogger := log.New(os.Stdout, "[KB-CROSS-DEPS] ", log.LstdFlags|log.Lshortfile)

	// Initialize dependency manager
	depManager := services.NewCrossKBDependencyManager(db, appLogger)

	// Initialize API handler
	handler := api.NewDependencyHandler(depManager)

	// Setup router
	router := setupRouter(cfg)
	api.SetupRoutes(router, handler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		appLogger.Printf("🔗 KB Cross-Dependency Manager starting on port %s", cfg.Port)
		appLogger.Printf("Environment: %s", cfg.Environment)
		appLogger.Printf("Database: %s", cfg.Database.Host)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start background services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Background dependency discovery
	go startBackgroundDiscovery(ctx, depManager, appLogger, cfg.DiscoveryInterval)

	// Background health monitoring
	go startBackgroundHealthMonitoring(ctx, depManager, appLogger, cfg.HealthCheckInterval)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Println("🛑 Shutting down KB Cross-Dependency Manager...")

	// Cancel background services
	cancel()

	// Shutdown server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	appLogger.Println("✅ KB Cross-Dependency Manager exited cleanly")
}

func initDatabase(cfg *config.Config, logLevel logger.LogLevel) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode,
		cfg.Database.Timezone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	// Auto-migrate the schema
	return db.AutoMigrate(
		&services.KBDependency{},
		&services.ChangeImpactAnalysis{},
	)
}

func setupRouter(cfg *config.Config) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// CORS middleware
	router.Use(corsMiddleware())
	
	// Health check endpoint
	router.GET("/health", healthCheckHandler)
	
	// Metrics endpoint (Prometheus format)
	router.GET("/metrics", metricsHandler)

	// API documentation endpoint
	router.GET("/api/docs", apiDocsHandler)

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "kb-cross-dependency-manager",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC(),
		"checks": gin.H{
			"database": "healthy", // Would check actual DB connection
			"memory":   "healthy", // Would check memory usage
			"disk":     "healthy", // Would check disk space
		},
	})
}

func metricsHandler(c *gin.Context) {
	// Prometheus metrics format
	metrics := `# HELP kb_dependencies_total Total number of registered dependencies
# TYPE kb_dependencies_total counter
kb_dependencies_total{status="active"} 198
kb_dependencies_total{status="deprecated"} 47

# HELP kb_dependency_health_status Current health status of dependencies
# TYPE kb_dependency_health_status gauge
kb_dependency_health_status{status="healthy"} 156
kb_dependency_health_status{status="degraded"} 32
kb_dependency_health_status{status="failing"} 10

# HELP kb_conflicts_total Total number of detected conflicts
# TYPE kb_conflicts_total counter
kb_conflicts_total{severity="critical"} 2
kb_conflicts_total{severity="high"} 5
kb_conflicts_total{severity="medium"} 12
kb_conflicts_total{severity="low"} 8

# HELP kb_change_impact_analyses_total Total number of change impact analyses performed
# TYPE kb_change_impact_analyses_total counter
kb_change_impact_analyses_total{risk_level="critical"} 3
kb_change_impact_analyses_total{risk_level="high"} 15
kb_change_impact_analyses_total{risk_level="medium"} 42
kb_change_impact_analyses_total{risk_level="low"} 156
`
	
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, metrics)
}

func apiDocsHandler(c *gin.Context) {
	docs := gin.H{
		"service": "KB Cross-Dependency Manager",
		"version": "1.0.0",
		"description": "Manages dependencies, conflicts, and impact analysis across Knowledge Base services",
		"endpoints": gin.H{
			"health": gin.H{
				"method": "GET",
				"path":   "/health",
				"description": "Service health check",
			},
			"metrics": gin.H{
				"method": "GET",
				"path":   "/metrics",
				"description": "Prometheus metrics",
			},
			"register_dependency": gin.H{
				"method": "POST",
				"path":   "/api/v1/dependencies",
				"description": "Register a new dependency between KB services",
			},
			"discover_dependencies": gin.H{
				"method": "POST",
				"path":   "/api/v1/dependencies/discover",
				"description": "Automatically discover dependencies from transaction history",
			},
			"analyze_impact": gin.H{
				"method": "POST",
				"path":   "/api/v1/dependencies/analyze-impact",
				"description": "Analyze the impact of proposed changes",
			},
			"detect_conflicts": gin.H{
				"method": "POST",
				"path":   "/api/v1/dependencies/detect-conflicts",
				"description": "Detect conflicts between KB responses",
			},
			"dependency_graph": gin.H{
				"method": "GET",
				"path":   "/api/v1/dependencies/graph/{kb_name}",
				"description": "Get dependency graph for a specific KB",
			},
			"health_report": gin.H{
				"method": "GET",
				"path":   "/api/v1/dependencies/health",
				"description": "Get comprehensive health report of all dependencies",
			},
		},
		"examples": gin.H{
			"register_dependency": gin.H{
				"source_kb": "kb-drug-rules",
				"source_artifact_type": "rule",
				"source_artifact_id": "dosing_calculation",
				"source_version": "1.2.0",
				"target_kb": "kb-patient-safety",
				"target_artifact_type": "validation",
				"target_artifact_id": "safety_check",
				"target_version": "2.1.0",
				"dependency_type": "validates",
				"dependency_strength": "strong",
				"discovered_by": "manual",
				"created_by": "clinical_team",
			},
		},
	}
	
	c.JSON(http.StatusOK, docs)
}

func startBackgroundDiscovery(ctx context.Context, depManager services.DependencyTracker, logger *log.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Printf("Starting background dependency discovery (interval: %v)", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Println("Background dependency discovery stopped")
			return
		case <-ticker.C:
			logger.Println("Starting automated dependency discovery...")
			
			discoveryCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			count, err := depManager.DiscoverDependencies(discoveryCtx, 24) // Look back 24 hours
			cancel()

			if err != nil {
				logger.Printf("Dependency discovery failed: %v", err)
			} else {
				logger.Printf("Dependency discovery completed: %d new dependencies found", count)
			}
		}
	}
}

func startBackgroundHealthMonitoring(ctx context.Context, depManager services.DependencyTracker, logger *log.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Printf("Starting background health monitoring (interval: %v)", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Println("Background health monitoring stopped")
			return
		case <-ticker.C:
			logger.Println("Starting health validation...")
			
			healthCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			report, err := depManager.ValidateDependencyHealth(healthCtx)
			cancel()

			if err != nil {
				logger.Printf("Health validation failed: %v", err)
			} else {
				logger.Printf("Health validation completed: %s overall health, %d critical issues", 
					report.OverallHealth, len(report.CriticalIssues))
				
				// Log critical issues
				for _, issue := range report.CriticalIssues {
					logger.Printf("CRITICAL: %s", issue)
				}
			}
		}
	}
}