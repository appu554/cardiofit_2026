package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cardiofit/kb7-query-router/internal/cache"
	"github.com/cardiofit/kb7-query-router/internal/config"
	"github.com/cardiofit/kb7-query-router/internal/elasticsearch"
	"github.com/cardiofit/kb7-query-router/internal/graphdb"
	"github.com/cardiofit/kb7-query-router/internal/postgres"
	"github.com/cardiofit/kb7-query-router/internal/router"
	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Initialize OpenTelemetry
	tp, err := initTracer()
	if err != nil {
		logger.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Errorf("Error shutting down tracer: %v", err)
		}
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize clients
	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		logger.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer redisClient.Close()

	postgresClient, err := postgres.NewClient(cfg.PostgresURL)
	if err != nil {
		logger.Fatalf("Failed to initialize PostgreSQL client: %v", err)
	}
	defer postgresClient.Close()

	graphDBClient, err := graphdb.NewClient(cfg.GraphDBEndpoint)
	if err != nil {
		logger.Fatalf("Failed to initialize GraphDB client: %v", err)
	}

	// Initialize Elasticsearch client
	esConfig := es.Config{
		Addresses: []string{cfg.ElasticsearchURL},
		// Configure authentication if needed
		// Username: cfg.ElasticsearchUsername,
		// Password: cfg.ElasticsearchPassword,
	}

	elasticsearchClient, err := elasticsearch.NewClient(esConfig, cfg.ElasticsearchIndex, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize Elasticsearch client: %v", err)
	}

	// Initialize query router
	queryRouter := router.NewHybridQueryRouter(
		postgresClient,
		graphDBClient,
		elasticsearchClient,
		redisClient,
		logger,
	)

	// Setup HTTP server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("kb7-query-router"))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		status := queryRouter.HealthCheck()
		if status.Healthy {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	})

	// Readiness check endpoint
	r.GET("/ready", func(c *gin.Context) {
		status := queryRouter.ReadinessCheck()
		if status.Ready {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	})

	// API routes
	v1 := r.Group("/api/v1")
	{
		// Concept lookup - PostgreSQL route
		v1.GET("/concepts/:system/:code", queryRouter.HandleConceptLookup)

		// Subsumption query - GraphDB route
		v1.GET("/concepts/:system/:code/subconcepts", queryRouter.HandleSubconceptQuery)

		// Cross-terminology mapping - PostgreSQL route
		v1.GET("/mappings/:fromSystem/:fromCode/:toSystem", queryRouter.HandleMappingQuery)

		// Drug interactions - GraphDB route
		v1.POST("/interactions", queryRouter.HandleDrugInteractions)

		// Concept relationships - Hybrid route
		v1.GET("/concepts/:system/:code/relationships", queryRouter.HandleRelationshipQuery)

		// Text search - PostgreSQL route (legacy)
		v1.GET("/search", queryRouter.HandleTextSearch)

		// Advanced search - Elasticsearch routes
		v1.GET("/search/advanced", queryRouter.HandleAdvancedSearch)
		v1.POST("/search/advanced", queryRouter.HandleAdvancedSearch)

		// Autocomplete - Elasticsearch routes
		v1.GET("/search/autocomplete", queryRouter.HandleAutocomplete)
		v1.POST("/search/autocomplete", queryRouter.HandleAutocomplete)

		// Query metrics
		v1.GET("/metrics", queryRouter.HandleMetrics)
	}

	// Start metrics server
	go func() {
		metricsRouter := gin.New()
		metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))
		logger.Infof("Starting metrics server on port %s", cfg.MetricsPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.MetricsPort), metricsRouter); err != nil {
			logger.Errorf("Metrics server error: %v", err)
		}
	}()

	// Start main server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.APIPort),
		Handler: r,
	}

	go func() {
		logger.Infof("Starting KB7 Query Router on port %s", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server startup failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exiting")
}

func initTracer() (*trace.TracerProvider, error) {
	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(os.Getenv("JAEGER_ENDPOINT")),
	))
	if err != nil {
		return nil, err
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	return tp, nil
}