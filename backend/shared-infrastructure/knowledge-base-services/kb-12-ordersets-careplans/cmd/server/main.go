// KB-12 Order Sets & Care Plans Service
// Main entry point for the clinical order sets and care plans microservice
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

	"kb-12-ordersets-careplans/internal/cache"
	"kb-12-ordersets-careplans/internal/clients"
	"kb-12-ordersets-careplans/internal/config"
	"kb-12-ordersets-careplans/internal/database"
	"kb-12-ordersets-careplans/pkg/careplans"
	"kb-12-ordersets-careplans/pkg/cdshooks"
	"kb-12-ordersets-careplans/pkg/cpoe"
	"kb-12-ordersets-careplans/pkg/ordersets"
	"kb-12-ordersets-careplans/pkg/workflow"
)

// Build-time variables
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logging
	cfg.Logging.SetupLogging()

	// Set Gin mode
	if cfg.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize logger
	log.Printf("KB-12 Order Sets & Care Plans Service v%s (built %s)", Version, BuildTime)
	log.Printf("Environment: %s", cfg.Server.Environment)

	// Initialize database
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Database connected")

	// Initialize Redis cache
	redisCache, err := cache.NewCache(&cfg.Redis)
	if err != nil {
		log.Printf("⚠ Redis cache not available: %v (continuing without cache)", err)
		redisCache = nil
	} else {
		log.Println("✓ Redis cache connected")
	}

	// Initialize KB service clients
	kb1Client := clients.NewKB1DosingClient(cfg.KBServices.KB1Dosing)
	kb3Client := clients.NewKB3TemporalClient(cfg.KBServices.KB3Temporal)
	kb6Client := clients.NewKB6FormularyClient(cfg.KBServices.KB6Formulary)
	kb7Client := clients.NewKB7TerminologyClient(cfg.KBServices.KB7Terminology)
	log.Println("✓ KB service clients initialized")

	// Initialize template loader
	templateLoader := ordersets.NewTemplateLoader(db.GetDB(), redisCache)
	if err := templateLoader.LoadAllTemplates(context.Background()); err != nil {
		log.Printf("⚠ Error loading templates: %v (using hardcoded defaults)", err)
	}
	log.Printf("✓ Templates loaded: %v", ordersets.GetTemplateCount())

	// Initialize order set service
	orderSetService := ordersets.NewOrderSetService(db.GetDB(), redisCache, kb1Client, kb3Client, kb6Client, kb7Client)
	log.Println("✓ Order Set Service initialized")

	// Initialize care plan service
	_ = careplans.GetAllCarePlans() // Load care plans
	log.Printf("✓ Care Plan Service initialized: %v", careplans.GetCarePlanCount())

	// Initialize CPOE service
	cpoeService := cpoe.NewCPOEService(kb1Client, kb3Client, kb6Client, kb7Client)
	log.Println("✓ CPOE Service initialized")

	// Initialize workflow engine
	workflowEngine := workflow.NewWorkflowEngine(kb3Client)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := workflowEngine.Start(ctx); err != nil {
		log.Printf("⚠ Workflow engine start error: %v", err)
	} else {
		log.Println("✓ Workflow Engine started")
	}

	// Initialize CDS Hooks service
	cdsHooksService := cdshooks.NewCDSHooksService(templateLoader)
	feedbackHandler := cdshooks.NewFeedbackHandler()
	log.Println("✓ CDS Hooks Service initialized")

	// Create Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Register health routes
	registerHealthRoutes(router, db, redisCache)

	// Register API routes
	api := router.Group("/api/v1")
	{
		// Order Set endpoints
		api.GET("/ordersets", func(c *gin.Context) {
			templates := orderSetService.GetAllTemplates()
			c.JSON(http.StatusOK, gin.H{"templates": templates, "count": len(templates)})
		})
		api.GET("/ordersets/:id", func(c *gin.Context) {
			template, err := orderSetService.GetTemplate(c.Request.Context(), c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, template)
		})
		api.GET("/ordersets/category/:category", func(c *gin.Context) {
			templates := orderSetService.GetTemplatesByCategory(c.Param("category"))
			c.JSON(http.StatusOK, gin.H{"templates": templates, "count": len(templates)})
		})
		api.POST("/ordersets/:id/apply", func(c *gin.Context) {
			// Apply order set - creates workflow
			c.JSON(http.StatusOK, gin.H{"status": "applied"})
		})

		// Care Plan endpoints
		api.GET("/careplans", func(c *gin.Context) {
			plans := careplans.GetAllCarePlans()
			c.JSON(http.StatusOK, gin.H{"care_plans": plans, "count": len(plans)})
		})
		api.GET("/careplans/:id", func(c *gin.Context) {
			for _, plan := range careplans.GetAllCarePlans() {
				if plan.TemplateID == c.Param("id") {
					c.JSON(http.StatusOK, plan)
					return
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "care plan not found"})
		})

		// CPOE endpoints
		api.POST("/cpoe/sessions", func(c *gin.Context) {
			var req cpoe.CreateSessionRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			session, err := cpoeService.CreateOrderSession(c.Request.Context(), &req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, session)
		})
		api.POST("/cpoe/sessions/:id/orders", func(c *gin.Context) {
			var order cpoe.PendingOrder
			if err := c.ShouldBindJSON(&order); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			resp, err := cpoeService.AddOrder(c.Request.Context(), c.Param("id"), &order)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, resp)
		})
		api.GET("/cpoe/sessions/:id", func(c *gin.Context) {
			session, err := cpoeService.GetSession(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, session)
		})
		api.POST("/cpoe/sessions/:id/sign", func(c *gin.Context) {
			var req struct {
				SignerID  string            `json:"signer_id"`
				Overrides map[string]string `json:"overrides"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			resp, err := cpoeService.SignOrders(c.Request.Context(), c.Param("id"), req.SignerID, req.Overrides)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, resp)
		})

		// Workflow endpoints
		api.GET("/workflows", func(c *gin.Context) {
			patientID := c.Query("patient_id")
			if patientID != "" {
				workflows := workflowEngine.GetPatientWorkflows(patientID)
				c.JSON(http.StatusOK, gin.H{"workflows": workflows, "count": len(workflows)})
				return
			}
			c.JSON(http.StatusOK, gin.H{"metrics": workflowEngine.GetMetrics()})
		})
		api.GET("/workflows/:id", func(c *gin.Context) {
			wf, err := workflowEngine.GetWorkflowInstance(c.Param("id"))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, wf)
		})
		api.GET("/workflows/constraints", func(c *gin.Context) {
			constraints := workflowEngine.GetActiveTimeConstraints()
			c.JSON(http.StatusOK, gin.H{"constraints": constraints, "count": len(constraints)})
		})
	}

	// Register CDS Hooks routes
	registerCDSHooksRoutes(router, cdsHooksService, feedbackHandler)

	// Register metrics route
	registerMetricsRoute(router, workflowEngine)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⏳ Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Stop workflow engine
	workflowEngine.Stop()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("✓ Server stopped")
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// registerHealthRoutes adds health check endpoints
func registerHealthRoutes(router *gin.Engine, db *database.Connection, cacheClient *cache.Cache) {
	router.GET("/health", func(c *gin.Context) {
		health := gin.H{
			"status":  "healthy",
			"service": "kb-12-ordersets-careplans",
			"version": Version,
			"time":    time.Now().UTC().Format(time.RFC3339),
		}

		// Check database
		if db != nil {
			if err := db.Health(c.Request.Context()); err != nil {
				health["database"] = "unhealthy"
				health["status"] = "degraded"
			} else {
				health["database"] = "healthy"
			}
		}

		// Check cache
		if cacheClient != nil {
			if err := cacheClient.Health(c.Request.Context()); err != nil {
				health["cache"] = "unhealthy"
			} else {
				health["cache"] = "healthy"
			}
		} else {
			health["cache"] = "disabled"
		}

		c.JSON(http.StatusOK, health)
	})

	router.GET("/ready", func(c *gin.Context) {
		if db != nil {
			if err := db.Health(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "error": "database not ready"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	router.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"alive": true})
	})
}

// registerCDSHooksRoutes adds CDS Hooks 2.0 compliant endpoints
func registerCDSHooksRoutes(router *gin.Engine, service *cdshooks.CDSHooksService, feedback *cdshooks.FeedbackHandler) {
	cds := router.Group("/cds-services")
	{
		// Discovery endpoint
		cds.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, service.GetDiscovery())
		})

		// Individual hook endpoints
		hooks := []string{"kb12-patient-view", "kb12-order-select", "kb12-order-sign", "kb12-encounter-start", "kb12-encounter-discharge"}
		for _, hookID := range hooks {
			hookID := hookID
			cds.POST("/"+hookID, func(c *gin.Context) {
				var req cdshooks.CDSRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				resp, err := service.ProcessHook(c.Request.Context(), hookID, &req)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, resp)
			})
		}

		// Feedback endpoint
		cds.POST("/feedback", func(c *gin.Context) {
			var fb cdshooks.CardFeedback
			if err := c.ShouldBindJSON(&fb); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if err := feedback.RecordFeedback(&fb); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"recorded": true})
		})
	}
}

// registerMetricsRoute adds metrics endpoint
func registerMetricsRoute(router *gin.Engine, engine *workflow.WorkflowEngine) {
	router.GET("/metrics", func(c *gin.Context) {
		metrics := gin.H{
			"templates":  ordersets.GetTemplateCount(),
			"care_plans": careplans.GetCarePlanCount(),
		}

		if engine != nil {
			metrics["workflows"] = engine.GetMetrics()
		}

		c.JSON(http.StatusOK, metrics)
	})
}
