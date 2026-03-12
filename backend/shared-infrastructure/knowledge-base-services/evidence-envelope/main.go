package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"evidence-envelope/internal/models"
	"evidence-envelope/internal/services"
	
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "evidence_envelope_request_duration_seconds",
			Help: "Duration of Evidence Envelope requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	
	totalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "evidence_envelope_requests_total",
			Help: "Total number of Evidence Envelope requests",
		},
		[]string{"method", "endpoint", "status"},
	)
)

func init() {
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(totalRequests)
}

type Config struct {
	Port        string
	DatabaseURL string
	Environment string
}

func loadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8088"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/evidence_envelope?sslmode=disable"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GraphQL Schema for Evidence Envelope
const schema = `
	scalar Time
	scalar JSON

	type EvidenceTransaction {
		id: String!
		transactionId: String!
		userId: String
		sessionId: String
		sourceService: String!
		targetService: String
		operationType: String!
		graphqlOperation: String
		requestPayload: JSON
		responsePayload: JSON
		httpStatus: Int
		processingTimeMs: Int
		timestamp: Time!
		correlationId: String
		traceId: String
		spanId: String
	}

	type ClinicalDecision {
		id: String!
		transactionId: String!
		decisionId: String!
		patientId: String
		decisionType: String!
		knowledgeSource: String!
		inputData: JSON!
		decisionOutcome: JSON!
		confidenceScore: Float
		evidenceSources: JSON
		overriddenBy: String
		overrideReason: String
		createdAt: Time!
		expiresAt: Time
	}

	input TransactionInput {
		userId: String
		sessionId: String
		sourceService: String!
		targetService: String
		operationType: String!
		graphqlOperation: String
		requestPayload: JSON
		correlationId: String
	}

	input ClinicalDecisionInput {
		transactionId: String!
		decisionId: String!
		patientId: String
		decisionType: String!
		knowledgeSource: String!
		inputData: JSON!
		decisionOutcome: JSON!
		confidenceScore: Float
		evidenceSources: JSON
		expiresAt: Time
	}

	input AuditQueryInput {
		userId: String
		service: String
		operationType: String
		patientId: String
		startTime: Time
		endTime: Time
		limit: Int!
		offset: Int!
	}

	type TransactionResponse {
		transactionId: String!
		createdAt: Time!
	}

	type Query {
		transaction(transactionId: String!): EvidenceTransaction
		auditTrail(query: AuditQueryInput!): [EvidenceTransaction!]!
		clinicalDecisionHistory(patientId: String!, limit: Int!): [ClinicalDecision!]!
		health: String!
	}

	type Mutation {
		createTransaction(input: TransactionInput!): TransactionResponse!
		completeTransaction(transactionId: String!, responsePayload: JSON, httpStatus: Int!, processingTimeMs: Int!): Boolean!
		recordClinicalDecision(input: ClinicalDecisionInput!): Boolean!
	}
`

type Resolver struct {
	evidenceService *services.EvidenceService
}

func (r *Resolver) Transaction(args struct{ TransactionID string }) (*models.EvidenceTransaction, error) {
	return r.evidenceService.GetTransactionAuditTrail(context.Background(), args.TransactionID)
}

func (r *Resolver) AuditTrail(args struct{ Query models.AuditQuery }) ([]models.EvidenceTransaction, error) {
	return r.evidenceService.QueryAuditTrail(context.Background(), args.Query)
}

func (r *Resolver) ClinicalDecisionHistory(args struct{ PatientID string; Limit int32 }) ([]models.ClinicalDecision, error) {
	return r.evidenceService.GetClinicalDecisionHistory(context.Background(), args.PatientID, int(args.Limit))
}

func (r *Resolver) Health() string {
	return "Evidence Envelope service is healthy"
}

func (r *Resolver) CreateTransaction(args struct{ Input models.TransactionRequest }) (*models.TransactionResponse, error) {
	return r.evidenceService.CreateTransaction(context.Background(), args.Input)
}

func (r *Resolver) CompleteTransaction(args struct{ 
	TransactionID string; 
	ResponsePayload *string; 
	HTTPStatus int32; 
	ProcessingTimeMS int32 
}) (bool, error) {
	var payload []byte
	if args.ResponsePayload != nil {
		payload = []byte(*args.ResponsePayload)
	}
	
	err := r.evidenceService.CompleteTransaction(
		context.Background(), 
		args.TransactionID, 
		payload, 
		int(args.HTTPStatus), 
		int(args.ProcessingTimeMS),
	)
	return err == nil, err
}

func (r *Resolver) RecordClinicalDecision(args struct{ Input models.ClinicalDecision }) (bool, error) {
	err := r.evidenceService.RecordClinicalDecision(context.Background(), args.Input)
	return err == nil, err
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create database driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not run migrations: %w", err)
	}

	return nil
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := loadConfig()
	
	logger.Info("Starting Evidence Envelope service",
		zap.String("port", cfg.Port),
		zap.String("environment", cfg.Environment))

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize services
	evidenceService := services.NewEvidenceService(db, logger)
	
	// Initialize GraphQL
	resolver := &Resolver{evidenceService: evidenceService}
	graphqlSchema := graphql.MustParseSchema(schema, resolver)
	
	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	
	// Metrics middleware
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", c.Writer.Status())
		
		requestDuration.WithLabelValues(c.Request.Method, c.FullPath(), status).Observe(duration)
		totalRequests.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
	})

	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "evidence-envelope"})
	})

	// GraphQL endpoints
	router.POST("/graphql", gin.WrapH(&relay.Handler{Schema: graphqlSchema}))
	router.POST("/api/federation", gin.WrapH(&relay.Handler{Schema: graphqlSchema}))
	
	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Service information
	fmt.Printf(`
========================================
Evidence Envelope Service
========================================
Service: evidence-envelope
Port: %s
Version: 1.0.0
Environment: %s
========================================

Features:
- Transaction audit trails
- Data lineage tracking  
- Clinical decision logging
- Knowledge base versioning
- Performance metrics
- GraphQL Federation support

Database: PostgreSQL
Metrics: Prometheus
Tracing: OpenTelemetry compatible

========================================
`, cfg.Port, cfg.Environment)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	logger.Info("Evidence Envelope service started successfully",
		zap.String("address", "http://localhost:"+cfg.Port))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Evidence Envelope service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	logger.Info("Evidence Envelope service stopped")
}