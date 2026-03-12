// Context Gateway Go Service - Clinical Snapshot & Recipe Management
// Implements the Context Gateway from Implementation Requirements documentation
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"context-gateway-go/internal/api"
	"context-gateway-go/internal/services"
	"context-gateway-go/internal/storage"
	pb "context-gateway-go/proto"
)

const (
	defaultGRPCPort = ":8017"
	defaultHTTPPort = ":8117"
	defaultRedisAddr = "localhost:6379"
	defaultMongoURI = "mongodb://localhost:27017"
	defaultDBName = "clinical_context_go"
)

type Config struct {
	GRPCPort    string
	HTTPPort    string
	RedisAddr   string
	MongoURI    string
	DBName      string
	Environment string
}

func main() {
	// Parse command line flags
	config := parseFlags()
	
	// Initialize logging
	setupLogging(config.Environment)
	
	log.Printf("🚀 Starting Context Gateway Go Service")
	log.Printf("   Environment: %s", config.Environment)
	log.Printf("   gRPC Port: %s", config.GRPCPort)
	log.Printf("   HTTP Port: %s", config.HTTPPort)
	log.Printf("   Redis: %s", config.RedisAddr)
	log.Printf("   MongoDB: %s", config.MongoURI)
	
	// Initialize storage layer
	snapshotStore, err := storage.NewSnapshotStore(config.RedisAddr, config.MongoURI, config.DBName)
	if err != nil {
		log.Fatalf("❌ Failed to initialize snapshot store: %v", err)
	}
	defer func() {
		if err := snapshotStore.Close(); err != nil {
			log.Printf("⚠️ Error closing snapshot store: %v", err)
		}
	}()
	
	// Initialize services
	recipeService := services.NewRecipeService()
	dataSourceRegistry := services.NewDataSourceRegistry()
	
	// Initialize Context Gateway service
	contextGatewayService := services.NewContextGatewayService(
		snapshotStore,
		recipeService,
		dataSourceRegistry,
	)
	
	// Start gRPC server
	grpcServer := setupGRPCServer(contextGatewayService)
	grpcListener, err := net.Listen("tcp", config.GRPCPort)
	if err != nil {
		log.Fatalf("❌ Failed to listen on gRPC port %s: %v", config.GRPCPort, err)
	}
	
	// Start HTTP server for metrics and health checks
	httpServer := setupHTTPServer(config.HTTPPort, contextGatewayService, snapshotStore)
	
	// Start servers in goroutines
	go func() {
		log.Printf("✅ Context Gateway gRPC server listening on %s", config.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("❌ gRPC server failed: %v", err)
		}
	}()
	
	go func() {
		log.Printf("✅ Context Gateway HTTP server listening on %s", config.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ HTTP server failed: %v", err)
		}
	}()
	
	// Service is now ready
	log.Printf("🎉 Context Gateway Go Service is ready!")
	log.Printf("   📊 Health Check: http://localhost%s/health", config.HTTPPort)
	log.Printf("   📈 Metrics: http://localhost%s/metrics", config.HTTPPort)
	log.Printf("   🔧 Status: http://localhost%s/status", config.HTTPPort)
	
	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Printf("🛑 Shutting down Context Gateway Go Service...")
	
	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ HTTP server shutdown error: %v", err)
	}
	
	// Shutdown gRPC server
	grpcServer.GracefulStop()
	
	log.Printf("✅ Context Gateway Go Service stopped gracefully")
}

func parseFlags() Config {
	config := Config{
		GRPCPort:    defaultGRPCPort,
		HTTPPort:    defaultHTTPPort,
		RedisAddr:   defaultRedisAddr,
		MongoURI:    defaultMongoURI,
		DBName:      defaultDBName,
		Environment: "development",
	}
	
	flag.StringVar(&config.GRPCPort, "grpc-port", config.GRPCPort, "gRPC server port")
	flag.StringVar(&config.HTTPPort, "http-port", config.HTTPPort, "HTTP server port")
	flag.StringVar(&config.RedisAddr, "redis-addr", config.RedisAddr, "Redis address")
	flag.StringVar(&config.MongoURI, "mongo-uri", config.MongoURI, "MongoDB URI")
	flag.StringVar(&config.DBName, "db-name", config.DBName, "Database name")
	flag.StringVar(&config.Environment, "env", config.Environment, "Environment (development, production)")
	
	flag.Parse()
	
	// Override with environment variables if present
	if val := os.Getenv("GRPC_PORT"); val != "" {
		config.GRPCPort = ":" + val
	}
	if val := os.Getenv("HTTP_PORT"); val != "" {
		config.HTTPPort = ":" + val
	}
	if val := os.Getenv("REDIS_ADDR"); val != "" {
		config.RedisAddr = val
	}
	if val := os.Getenv("MONGO_URI"); val != "" {
		config.MongoURI = val
	}
	if val := os.Getenv("DB_NAME"); val != "" {
		config.DBName = val
	}
	if val := os.Getenv("ENVIRONMENT"); val != "" {
		config.Environment = val
	}
	
	return config
}

func setupLogging(environment string) {
	if environment == "production" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}

func setupGRPCServer(contextGatewayService *services.ContextGatewayService) *grpc.Server {
	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB max receive message size
		grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB max send message size
	}
	
	grpcServer := grpc.NewServer(opts...)
	
	// Register Context Gateway service
	pb.RegisterContextGatewayServer(grpcServer, contextGatewayService)
	
	// Register health service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("context_gateway", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	
	// Enable reflection for development (disable in production)
	reflection.Register(grpcServer)
	
	return grpcServer
}

func setupHTTPServer(port string, contextGatewayService *services.ContextGatewayService, snapshotStore *storage.SnapshotStore) *http.Server {
	router := mux.NewRouter()
	
	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Check service health
		_, err := contextGatewayService.GetServiceHealth(r.Context(), &pb.HealthRequest{
			IncludeDependencies: true,
		})
		
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"%s","timestamp":"%s"}`, 
				err.Error(), time.Now().UTC().Format(time.RFC3339))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"context-gateway-go","timestamp":"%s"}`, 
			time.Now().UTC().Format(time.RFC3339))
	}).Methods("GET")
	
	// Readiness check endpoint
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Simple readiness check - ensure storage is accessible
		_, err := snapshotStore.GetStats(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"ready":false,"error":"%s","timestamp":"%s"}`, 
				err.Error(), time.Now().UTC().Format(time.RFC3339))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"ready":true,"service":"context-gateway-go","timestamp":"%s"}`, 
			time.Now().UTC().Format(time.RFC3339))
	}).Methods("GET")
	
	// Status endpoint with detailed service information
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Get comprehensive service health
		healthResp, err := contextGatewayService.GetServiceHealth(r.Context(), &pb.HealthRequest{
			IncludeDependencies: true,
		})
		
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status":"error","error":"%s","timestamp":"%s"}`, 
				err.Error(), time.Now().UTC().Format(time.RFC3339))
			return
		}
		
		// Get metrics
		metricsResp, _ := contextGatewayService.GetMetrics(r.Context(), &pb.MetricsRequest{})
		
		status := map[string]interface{}{
			"service":      "context-gateway-go",
			"version":      healthResp.Version,
			"status":       healthResp.Status.String(),
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
			"dependencies": healthResp.Dependencies,
			"cache_stats":  healthResp.CacheStats,
		}
		
		if metricsResp != nil {
			status["metrics"] = metricsResp.Metrics.AsMap()
		}
		
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(status); err != nil {
			log.Printf("Error encoding status response: %v", err)
		}
	}).Methods("GET")
	
	// Metrics endpoint (Prometheus format)
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	
	// Add federation endpoints for Apollo Federation integration
	api.SetupFederationRoutes(router, contextGatewayService)
	
	// Service information endpoint
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := map[string]interface{}{
			"service":     "Context Gateway Go Service",
			"version":     "1.0.0",
			"description": "Clinical Context Gateway implementing Recipe & Snapshot Management",
			"architecture": map[string]interface{}{
				"language":     "Go",
				"storage":      "Dual-layer (Redis + MongoDB)",
				"communication": "gRPC",
				"caching":      "Multi-layer intelligent cache",
			},
			"endpoints": map[string]interface{}{
				"grpc":    defaultGRPCPort,
				"health":  "/health",
				"ready":   "/ready",
				"status":  "/status",
				"metrics": "/metrics",
			},
			"capabilities": []string{
				"Clinical Snapshot Management",
				"Recipe-based Context Assembly",
				"Dual-layer Storage (Hot/Cold)",
				"Live Field Fetching with Governance",
				"Cryptographic Integrity Verification",
				"Clinical Audit Trail",
				"Performance Metrics Collection",
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		if err := json.NewEncoder(w).Encode(info); err != nil {
			log.Printf("Error encoding info response: %v", err)
		}
	}).Methods("GET")
	
	return &http.Server{
		Addr:         port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}