// Package api provides the HTTP API server for KB-19 Protocol Orchestrator.
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/adapters"
	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/clients"
	"kb-19-protocol-orchestrator/internal/config"
	"kb-19-protocol-orchestrator/internal/transaction"
)

// Server represents the HTTP API server for KB-19.
type Server struct {
	cfg                *config.Config
	log                *logrus.Entry
	router             *gin.Engine
	httpServer         *http.Server
	engine             *arbitration.Engine
	transactionManager *transaction.Manager
	transactionHandler *TransactionHandler
}

// NewServer creates a new API server instance.
func NewServer(cfg *config.Config, log *logrus.Entry) (*Server, error) {
	// Set Gin mode based on environment
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(loggingMiddleware(log))
	router.Use(corsMiddleware())

	// Create arbitration engine
	engine, err := arbitration.NewEngine(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create arbitration engine: %w", err)
	}

	// Create transaction manager (V3 Transaction Authority)
	txnStore := transaction.NewInMemoryStore() // Use in-memory store for now
	txnManager := transaction.NewManager(txnStore)

	// Wire KB-5 DDI client for drug-drug interaction checking
	if cfg.KBServices.KB5URL != "" {
		kb5Client := clients.NewKB5DDIClient(
			cfg.KBServices.KB5URL,
			cfg.KBServices.Timeout,
			log.WithField("component", "kb5-client"),
		)
		// Create adapter that implements KB5DDIChecker interface
		kb5Adapter := clients.NewKB5DDICheckerAdapter(kb5Client)
		txnManager.SetKB5Client(kb5Adapter)
		log.WithField("kb5_url", cfg.KBServices.KB5URL).Info("KB-5 DDI client configured")
	} else {
		log.Warn("KB5_URL not configured - using local DDI rules only")
	}

	// Wire Med-Advisor client for V3 risk profile workflow (preferred path)
	if cfg.KBServices.MedicationAdvisorURL != "" {
		medAdvisorClient := clients.NewMedicationAdvisorClient(
			cfg.KBServices.MedicationAdvisorURL,
			cfg.KBServices.Timeout,
			log.WithField("component", "med-advisor-client"),
		)
		// Create adapter that implements RiskProfileProvider interface
		riskProvider := adapters.NewMedAdvisorRiskProvider(medAdvisorClient)
		txnManager.SetRiskProvider(riskProvider)
		log.WithField("med_advisor_url", cfg.KBServices.MedicationAdvisorURL).Info("Med-Advisor V3 risk provider configured")
	} else {
		log.Warn("MEDICATION_ADVISOR_URL not configured - using KB-5 direct DDI mode only")
	}

	txnHandler := NewTransactionHandler(txnManager)

	server := &Server{
		cfg:                cfg,
		log:                log,
		router:             router,
		engine:             engine,
		transactionManager: txnManager,
		transactionHandler: txnHandler,
	}

	// Register routes
	server.registerRoutes()

	// Create HTTP server
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return server, nil
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Router returns the gin router for testing purposes.
func (s *Server) Router() *gin.Engine {
	return s.router
}

// registerRoutes sets up all API routes.
func (s *Server) registerRoutes() {
	// Health endpoints
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/ready", s.handleReady)
	s.router.GET("/metrics", s.handleMetrics)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Protocol orchestration
		v1.POST("/execute", s.handleExecute)
		v1.POST("/evaluate", s.handleEvaluate)

		// Protocol management
		v1.GET("/protocols", s.handleListProtocols)
		v1.GET("/protocols/:id", s.handleGetProtocol)

		// Decision history
		v1.GET("/decisions/:patientId", s.handleGetDecisions)
		v1.GET("/bundle/:id", s.handleGetBundle)

		// Conflict matrix
		v1.GET("/conflicts", s.handleListConflicts)
		v1.GET("/conflicts/:protocolId", s.handleGetConflictsForProtocol)

		// Event ingestion: KB-22 (HPI_COMPLETE, SAFETY_ALERT), KB-23 (MCU_GATE_CHANGED)
		v1.POST("/events", s.handleIngestEvent)

		// V3 Transaction Authority routes (MOVED FROM medication-advisor-engine)
		// These routes implement the Calculate → Validate → Commit workflow
		s.transactionHandler.RegisterTransactionRoutes(v1)
	}
}

// loggingMiddleware provides request logging.
func loggingMiddleware(log *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		log.WithFields(logrus.Fields{
			"status":   statusCode,
			"method":   c.Request.Method,
			"path":     path,
			"latency":  latency.String(),
			"clientIP": c.ClientIP(),
		}).Info("Request completed")
	}
}

// corsMiddleware provides CORS headers.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
