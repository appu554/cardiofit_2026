package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"kb-formulary/internal/cache"
	"kb-formulary/internal/config"
	"kb-formulary/internal/database"
	"kb-formulary/internal/grpc"
	"kb-formulary/internal/repository"
	"kb-formulary/internal/server"
	"kb-formulary/internal/services"
)

func main() {
	log.Println("Starting KB-6 Formulary Management Service...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	log.Println("Connecting to PostgreSQL database...")
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis cache
	log.Println("Connecting to Redis cache...")
	cache, err := cache.NewRedisManager(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	// Initialize Elasticsearch if enabled
	var es *database.ElasticsearchConnection
	if cfg.Elasticsearch.Enabled {
		log.Println("Connecting to Elasticsearch...")
		// Convert config struct to match database package structure
		esConfig := database.ElasticsearchConfig{
			Addresses: cfg.Elasticsearch.Addresses,
			Username:  cfg.Elasticsearch.Username,
			Password:  cfg.Elasticsearch.Password,
			CloudID:   cfg.Elasticsearch.CloudID,
			APIKey:    cfg.Elasticsearch.APIKey,
			Enabled:   cfg.Elasticsearch.Enabled,
		}
		es, err = database.NewElasticsearchConnection(esConfig)
		if err != nil {
			log.Printf("Warning: Failed to connect to Elasticsearch: %v", err)
		} else {
			defer es.Close()
			
			// Initialize Elasticsearch indices
			ctx := context.Background()
			if err := es.InitializeIndices(ctx); err != nil {
				log.Printf("Warning: Failed to initialize Elasticsearch indices: %v", err)
			} else {
				log.Println("Elasticsearch indices initialized successfully")
			}
		}
	}

	// Initialize Event Emitter for Cross-Service Signals (Enhancement #2)
	log.Println("Initializing Event Emitter for cross-service signals...")
	eventEmitterConfig := services.DefaultEventEmitterConfig()
	eventEmitter := services.NewEventEmitter(cache, eventEmitterConfig)
	defer eventEmitter.Close()

	// Initialize business services
	log.Println("Initializing business services...")
	formularyService := services.NewFormularyService(db, cache, es)
	inventoryService := services.NewInventoryService(db, cache)

	// Initialize PA/ST/QL repositories
	log.Println("Initializing PA/ST/QL repositories...")
	paRepo := repository.NewPARepository(db.DB())
	stRepo := repository.NewSTRepository(db.DB())
	qlRepo := repository.NewQLRepository(db.DB())

	// Initialize PA/ST/QL services with Event Emitter
	log.Println("Initializing PA/ST/QL services...")
	paService := services.NewPAService(paRepo)
	stService := services.NewSTService(stRepo)
	qlService := services.NewQLService(qlRepo)

	// Register event emitter with services for cross-service signaling
	paService.SetEventEmitter(eventEmitter)
	stService.SetEventEmitter(eventEmitter)
	qlService.SetEventEmitter(eventEmitter)

	// Initialize gRPC server
	log.Println("Initializing gRPC server...")
	grpcServer := grpc.NewServer(cfg, formularyService, inventoryService)

	// Initialize HTTP server for REST API
	log.Println("Initializing HTTP server...")
	httpServer := server.NewHTTPServer(cfg, formularyService, inventoryService, paService, stService, qlService)

	// Perform comprehensive health checks
	log.Println("Performing comprehensive health checks...")
	healthCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var healthWg sync.WaitGroup
	healthResults := make(map[string]error)
	healthMutex := sync.Mutex{}

	// Database health check
	healthWg.Add(1)
	go func() {
		defer healthWg.Done()
		err := db.HealthCheck()
		healthMutex.Lock()
		healthResults["database"] = err
		healthMutex.Unlock()
	}()

	// Redis health check
	healthWg.Add(1)
	go func() {
		defer healthWg.Done()
		err := cache.Ping()
		healthMutex.Lock()
		healthResults["redis"] = err
		healthMutex.Unlock()
	}()

	// Elasticsearch health check (if enabled)
	if es != nil {
		healthWg.Add(1)
		go func() {
			defer healthWg.Done()
			err := es.HealthCheck(healthCtx)
			healthMutex.Lock()
			healthResults["elasticsearch"] = err
			healthMutex.Unlock()
		}()
	}

	// Wait for health checks to complete
	healthWg.Wait()

	// Report health check results
	healthPassed := true
	for service, err := range healthResults {
		if err != nil {
			log.Printf("Warning: %s health check failed: %v", service, err)
			if service == "database" {
				log.Fatalf("Critical: Database health check failed, cannot continue")
			}
		} else {
			log.Printf("%s health check passed", service)
		}
	}

	if healthPassed {
		log.Println("All critical health checks passed")
	}

	// Display service information
	displayServiceInfo(cfg)

	// Start both servers in goroutines
	serverErrChan := make(chan error, 2)
	
	// Start gRPC server
	go func() {
		log.Printf("Starting gRPC server on port %s...", cfg.Server.Port)
		if err := grpcServer.Start(); err != nil {
			serverErrChan <- fmt.Errorf("gRPC server failed: %w", err)
		}
	}()
	
	// Start HTTP server
	go func() {
		httpPort := getHTTPPort(cfg.Server.Port)
		log.Printf("Starting HTTP REST API server on port %s...", httpPort)
		if err := httpServer.Start(); err != nil {
			serverErrChan <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case <-quit:
		log.Println("Received shutdown signal...")
	case err := <-serverErrChan:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown
	log.Println("Shutting down KB-6 Formulary Management Service...")
	
	// Stop both servers
	log.Println("Stopping gRPC server...")
	grpcServer.Stop()
	
	log.Println("Stopping HTTP server...")
	if err := httpServer.Stop(); err != nil {
		log.Printf("Error stopping HTTP server: %v", err)
	}
	
	// Close connections
	log.Println("Closing database connections...")
	if err := db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
	
	log.Println("Closing cache connections...")
	if err := cache.Close(); err != nil {
		log.Printf("Error closing cache: %v", err)
	}

	if es != nil {
		log.Println("Closing Elasticsearch connections...")
		if err := es.Close(); err != nil {
			log.Printf("Error closing Elasticsearch: %v", err)
		}
	}

	log.Println("KB-6 Formulary Management Service stopped successfully")
}

func displayServiceInfo(cfg *config.Config) {
	fmt.Printf(`
========================================
KB-6 Formulary Management Service
========================================
Service: kb-6-formulary
gRPC Port: %s
HTTP Port: %s
Version: 1.0.0
Environment: %s
Dataset Version: kb6.formulary.2025Q3.v1
========================================

🏥 Core Capabilities:
✓ Formulary coverage checking (gRPC & REST)
✓ Insurance plan management
✓ Real-time stock inventory tracking
✓ Demand prediction and forecasting
✓ Cost optimization analysis
✓ Therapeutic alternatives discovery

🔐 Prior Authorization (PA):
✓ Clinical criteria evaluation (DIAGNOSIS, LAB, PRIOR_THERAPY, AGE)
✓ PA submission and status tracking
✓ Auto-approval for eligible requests
✓ Criteria-based approval workflows

💊 Step Therapy (ST):
✓ Multi-step therapy validation
✓ Drug history verification
✓ Override request processing
✓ Exception handling for contraindications

📏 Quantity Limits (QL):
✓ Per-fill and days supply validation
✓ Annual fill limit tracking
✓ Override request processing
✓ Suggested compliant quantities

🛠️ Infrastructure:
- Database: PostgreSQL (port %s)
- Cache: Redis (DB %d, port %s)
- Search: Elasticsearch (%s)
- Metrics: Prometheus (/metrics)
- Health: gRPC health service

📊 Performance Targets:
- Coverage Check: p95 < 25ms (cached)
- PA Check: p95 < 50ms
- ST Validation: p95 < 40ms
- QL Check: p95 < 30ms
- Stock Query: p95 < 40ms
- Search: p95 < 200ms
- Cache Hit Rate: >95%% formulary, >85%% stock

🔌 Integration Points:
- ScoringEngine (gRPC primary consumer)
- Flow2 Orchestrator (workflow integration)
- Safety Gateway (event publishing)
- KB-7 Terminology (code normalization)
- Unified ETL Pipeline (data ingestion)

========================================
Service ready for ScoringEngine integration
========================================
`,
		cfg.Server.Port,
		getHTTPPort(cfg.Server.Port),
		cfg.Server.Environment,
		cfg.Database.Port,
		cfg.Redis.Database,
		cfg.Redis.Address,
		getElasticsearchStatus(cfg.Elasticsearch.Enabled),
	)
}

func getElasticsearchStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

// getHTTPPort calculates HTTP port from gRPC port (gRPC port + 1)
func getHTTPPort(grpcPort string) string {
	// For KB-6, if gRPC is on 8086, HTTP will be on 8087
	switch grpcPort {
	case "8086":
		return "8087"
	default:
		// Fallback logic - just add 1 to the last digit if possible
		if len(grpcPort) > 0 {
			lastChar := grpcPort[len(grpcPort)-1]
			if lastChar >= '0' && lastChar < '9' {
				return grpcPort[:len(grpcPort)-1] + string(lastChar+1)
			}
		}
		return "8087" // default fallback
	}
}