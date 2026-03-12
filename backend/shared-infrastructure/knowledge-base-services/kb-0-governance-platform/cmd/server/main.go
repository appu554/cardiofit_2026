// Package main provides the entry point for the KB-0 Governance Platform server.
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

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq" // For pq.Array in fact_store

	"kb-0-governance-platform/internal/api"
	"kb-0-governance-platform/internal/audit"
	"kb-0-governance-platform/internal/database"
	"kb-0-governance-platform/internal/governance"
	"kb-0-governance-platform/internal/workflow"
)

func main() {
	// Load configuration from environment
	config := loadConfig()

	// Connect to database
	db, err := sql.Open("pgx", config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL database")

	// Initialize components
	store := database.NewStore(db)
	factStore := database.NewFactStore(db)
	auditLogger := audit.NewLogger(db)
	workflowEngine := workflow.NewEngine(store, auditLogger, nil) // No notifier for now

	// Create governance executor for clinical facts
	executorConfig := governance.ExecutorConfig{
		PollInterval:        30 * time.Second,
		BatchSize:           50,
		MaxConcurrent:       5,
		EnableAutoProcess:   false, // Manual review mode
		EnableNotifications: false,
	}
	executor := governance.NewExecutor(factStore, nil, executorConfig) // No policy engine for now
	log.Println("Governance executor initialized")

	// Create API server with KB-1 integration
	var v1Server http.Handler
	if config.KB1URL != "" {
		log.Printf("KB-1 integration enabled: %s", config.KB1URL)
		v1Server = api.NewServerWithKB1(workflowEngine, store, auditLogger, config.KB1URL)
	} else {
		log.Println("KB-1 integration disabled (KB1_URL not set)")
		v1Server = api.NewServer(workflowEngine, store, auditLogger)
	}

	// Create v2 Fact Governance server for clinical facts
	factServer := api.NewFactServer(executor, factStore)
	log.Println("Fact governance API (v2) initialized")

	// Create Pipeline 1 review server for L2 extraction spans
	pipeline1Store := database.NewPipeline1Store(db)
	pipeline1Server := api.NewPipeline1Server(pipeline1Store)
	log.Println("Pipeline 1 review API initialized")

	// Create SPL review server for SPL FactStore Pipeline review workflows.
	// Uses the same database connection — the canonical_facts tables
	// (derived_facts, source_sections, completeness_reports, spl_sign_offs)
	// are expected to be in the same PostgreSQL database.
	splStore := database.NewSPLStore(db)
	splServer := api.NewSPLServer(splStore)
	log.Println("SPL review API initialized")

	// Combine all servers into a single handler
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route SPL review endpoints ("/api/v2/spl/" = 12 chars)
		if len(r.URL.Path) >= 12 && r.URL.Path[:12] == "/api/v2/spl/" {
			splServer.ServeHTTP(w, r)
			return
		}
		// Route Pipeline 1 endpoints
		if len(r.URL.Path) >= 18 && r.URL.Path[:18] == "/api/v2/pipeline1/" {
			pipeline1Server.ServeHTTP(w, r)
			return
		}
		// Route v2 governance endpoints to FactServer
		if len(r.URL.Path) >= 19 && r.URL.Path[:19] == "/api/v2/governance/" {
			factServer.ServeHTTP(w, r)
			return
		}
		// All other requests go to v1 server
		v1Server.ServeHTTP(w, r)
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("KB-0 Governance Platform starting on port %s", config.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Config holds server configuration.
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	KB1URL      string // KB-1 Drug Rules Service URL for governance integration
}

func loadConfig() *Config {
	return &Config{
		Port:        getEnv("KB0_PORT", "8080"),
		DatabaseURL: buildDatabaseURL(),
		RedisURL:    getEnv("KB0_REDIS_URL", "redis://localhost:6379"),
		KB1URL:      getEnv("KB1_URL", ""), // Empty default: KB-1 disabled unless explicitly configured
	}
}

func buildDatabaseURL() string {
	host := getEnv("KB0_DATABASE_HOST", "localhost")
	port := getEnv("KB0_DATABASE_PORT", "5432")
	name := getEnv("KB0_DATABASE_NAME", "kb0_governance")
	user := getEnv("KB0_DATABASE_USER", "kb0_user")
	password := getEnv("KB0_DATABASE_PASSWORD", "")

	if password != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, name)
	}
	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable",
		user, host, port, name)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
