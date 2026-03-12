# Medication Service V2 Implementation Guide

Comprehensive guide for implementing the Go/Rust Medication Service V2 with Recipe & Snapshot architecture.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Initial Setup](#initial-setup)
- [Component Migration](#component-migration)
- [Service Implementation](#service-implementation)
- [Database Configuration](#database-configuration)
- [Testing Strategy](#testing-strategy)
- [Performance Optimization](#performance-optimization)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements
```bash
# Go Development
go version >= 1.21
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Rust Development  
rustc --version >= 1.70
cargo --version >= 1.70

# Database Systems
postgresql >= 15.0
redis >= 7.0

# Container Platform
docker >= 24.0
docker-compose >= 2.0

# Additional Tools
git >= 2.30
make >= 4.0
```

### Development Environment Setup
```bash
# Install Go dependencies
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install Rust toolchain
rustup component add rustfmt clippy
cargo install cargo-audit cargo-tarpaulin

# Install protocol buffer compiler
# Ubuntu/Debian
sudo apt install protobuf-compiler
# macOS
brew install protobuf
# Windows
# Download from https://github.com/protocolbuffers/protobuf/releases
```

## Initial Setup

### 1. Create Service Directory Structure

```bash
cd backend/services
mkdir medication-service-v2
cd medication-service-v2

# Create Go project structure
mkdir -p {cmd/medication-server,internal/{domain,application,infrastructure,interfaces},pkg,configs,deployments,tests,docs}

# Initialize Go module
go mod init medication-service-v2
```

### 2. Setup Component Directories

```bash
# Create component directories
mkdir -p {flow2-go-engine-v2,flow2-rust-engine-v2,knowledge-bases-v2/{kb-drug-rules,kb-guideline-evidence}}

# Create shared directories  
mkdir -p {scripts,migrations,monitoring}
```

### 3. Initialize Configuration Files

Create `configs/service.yaml`:
```yaml
service:
  name: medication-service-v2
  version: "1.0.0"
  port: 8005
  environment: development

database:
  host: localhost
  port: 5434
  name: medication_v2
  user: medication_user
  password: medication_pass
  ssl_mode: disable
  max_connections: 100
  max_idle_connections: 10

redis:
  url: redis://localhost:6381
  max_connections: 20
  timeout: 5s

recipe_resolver:
  cache_enabled: true
  cache_ttl: 10m
  default_recipe_ttl: 1h
  max_conditional_depth: 5

clinical_engine:
  rust_engine_url: http://localhost:8095
  timeout: 30s
  max_retries: 3
  circuit_breaker:
    failure_threshold: 5
    recovery_timeout: 30s

context_gateway:
  base_url: http://localhost:8020
  timeout: 15s
  max_retries: 2

knowledge_bases:
  drug_rules_url: http://localhost:8086
  guidelines_url: http://localhost:8089
  apollo_federation_url: http://localhost:4000/graphql
  cache_ttl: 5m

monitoring:
  metrics_enabled: true
  tracing_enabled: true
  log_level: info
  health_check_interval: 30s
```

## Component Migration

### 1. Copy Existing Components

```bash
# Copy Flow2 engines with version suffix
cp -r ../medication-service/flow2-go-engine ./flow2-go-engine-v2
cp -r ../medication-service/flow2-rust-engine ./flow2-rust-engine-v2

# Copy knowledge bases
cp -r ../medication-service/knowledge-bases/kb-drug-rules ./knowledge-bases-v2/kb-drug-rules
cp -r ../medication-service/knowledge-bases/kb-guideline-evidence ./knowledge-bases-v2/kb-guideline-evidence
```

### 2. Update Port Configurations

```bash
# Update Go engine ports (8080 -> 8085)
find flow2-go-engine-v2 -name "*.yaml" -o -name "*.toml" -o -name "*.json" | \
  xargs sed -i 's/:8080/:8085/g'

# Update Rust engine ports (8090 -> 8095)  
find flow2-rust-engine-v2 -name "*.toml" -o -name "*.yaml" | \
  xargs sed -i 's/:8090/:8095/g'

# Update knowledge base ports
find knowledge-bases-v2/kb-drug-rules -name "*.toml" | \
  xargs sed -i 's/:8081/:8086/g'
  
find knowledge-bases-v2/kb-guideline-evidence -name "*.toml" | \
  xargs sed -i 's/:8084/:8089/g'
```

### 3. Update Service Names and IDs

```bash
# Update service identifiers to avoid conflicts
find . -name "*.toml" -o -name "*.yaml" -o -name "*.json" | \
  xargs sed -i 's/medication-service/medication-service-v2/g'
  
find . -name "*.toml" -o -name "*.yaml" -o -name "*.json" | \
  xargs sed -i 's/flow2-go-engine/flow2-go-engine-v2/g'
  
find . -name "*.toml" -o -name "*.yaml" -o -name "*.json" | \
  xargs sed -i 's/flow2-rust-engine/flow2-rust-engine-v2/g'
```

### 4. Update Database Connections

Create separate database configurations:
```bash
# Update database names and ports
find . -name "*.toml" -o -name "*.yaml" | \
  xargs sed -i 's/medication_db/medication_v2_db/g'
  
find . -name "*.toml" -o -name "*.yaml" | \
  xargs sed -i 's/:5432/:5434/g'  # PostgreSQL port
  
find . -name "*.toml" -o -name "*.yaml" | \
  xargs sed -i 's/:6379/:6381/g'  # Redis port
```

## Service Implementation

### 1. Go Main Service Implementation

Create `cmd/medication-server/main.go`:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"medication-service-v2/internal/application"
	"medication-service-v2/internal/infrastructure"
	"medication-service-v2/internal/interfaces/grpc_server"
	"medication-service-v2/internal/interfaces/http_server"
)

func main() {
	// Load configuration
	config, err := infrastructure.LoadConfig("configs/service.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize dependencies
	deps, err := initializeDependencies(config)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}
	defer deps.Cleanup()

	// Initialize services
	medicationService := application.NewMedicationService(
		deps.RecipeResolver,
		deps.ContextGateway,
		deps.RustEngine,
		deps.ApolloClient,
		deps.CacheManager,
		deps.EventPublisher,
	)

	// Start gRPC server
	grpcServer := grpc.NewServer()
	grpc_server.RegisterMedicationServiceServer(grpcServer, medicationService)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Service.Port+1))
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	go func() {
		log.Printf("Starting gRPC server on port %d", config.Service.Port+1)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Start HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Setup HTTP routes
	http_server.SetupRoutes(router, medicationService, config)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Service.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Starting HTTP server on port %d", config.Service.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	grpcServer.GracefulStop()
	log.Println("Servers shut down successfully")
}

type Dependencies struct {
	RecipeResolver  *application.RecipeResolver
	ContextGateway  *infrastructure.ContextGatewayClient
	RustEngine      *infrastructure.RustEngineClient
	ApolloClient    *infrastructure.ApolloFederationClient
	CacheManager    *infrastructure.CacheManager
	EventPublisher  *infrastructure.EventPublisher
	DatabaseManager *infrastructure.DatabaseManager
}

func (d *Dependencies) Cleanup() {
	if d.DatabaseManager != nil {
		d.DatabaseManager.Close()
	}
	if d.CacheManager != nil {
		d.CacheManager.Close()
	}
}

func initializeDependencies(config *infrastructure.Config) (*Dependencies, error) {
	// Database connection
	dbManager, err := infrastructure.NewDatabaseManager(config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Redis cache
	cacheManager, err := infrastructure.NewCacheManager(config.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// External clients
	contextGateway := infrastructure.NewContextGatewayClient(config.ContextGateway)
	rustEngine := infrastructure.NewRustEngineClient(config.ClinicalEngine)
	apolloClient := infrastructure.NewApolloFederationClient(config.KnowledgeBases)

	// Event publisher
	eventPublisher, err := infrastructure.NewEventPublisher(config.Database, config.Monitoring)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event publisher: %w", err)
	}

	// Recipe resolver
	recipeResolver := application.NewRecipeResolver(cacheManager, config.RecipeResolver)

	return &Dependencies{
		RecipeResolver:  recipeResolver,
		ContextGateway:  contextGateway,
		RustEngine:      rustEngine,
		ApolloClient:    apolloClient,
		CacheManager:    cacheManager,
		EventPublisher:  eventPublisher,
		DatabaseManager: dbManager,
	}, nil
}
```

### 2. Recipe Resolver Implementation

Create `internal/application/recipe_resolver.go`:
```go
package application

import (
	"context"
	"fmt"
	"time"

	"medication-service-v2/internal/domain"
	"medication-service-v2/internal/infrastructure"
)

type RecipeResolver struct {
	cache               *infrastructure.CacheManager
	recipeDefinitions   map[string]*domain.RecipeTemplate
	conditionalEngine   *ConditionalRuleEngine
	config              *RecipeResolverConfig
}

type RecipeResolverConfig struct {
	CacheEnabled       bool          `yaml:"cache_enabled"`
	CacheTTL          time.Duration `yaml:"cache_ttl"`
	DefaultRecipeTTL  time.Duration `yaml:"default_recipe_ttl"`
	MaxConditionalDepth int         `yaml:"max_conditional_depth"`
}

func NewRecipeResolver(cache *infrastructure.CacheManager, config *RecipeResolverConfig) *RecipeResolver {
	resolver := &RecipeResolver{
		cache:             cache,
		recipeDefinitions: make(map[string]*domain.RecipeTemplate),
		conditionalEngine: NewConditionalRuleEngine(config.MaxConditionalDepth),
		config:           config,
	}

	// Load recipe definitions
	if err := resolver.loadRecipeDefinitions(); err != nil {
		log.Printf("Warning: Failed to load recipe definitions: %v", err)
	}

	return resolver
}

func (r *RecipeResolver) ResolveWorkflowRecipe(
	ctx context.Context,
	protocolID string,
	contextNeeds *domain.ContextNeeds,
	patientCharacteristics *domain.PatientCharacteristics,
) (*domain.WorkflowRecipe, error) {
	// Check cache first
	if r.config.CacheEnabled {
		cacheKey := fmt.Sprintf("recipe:%s:%s", protocolID, contextNeeds.Hash())
		if cached, err := r.cache.Get(ctx, cacheKey); err == nil {
			var recipe domain.WorkflowRecipe
			if err := json.Unmarshal([]byte(cached), &recipe); err == nil {
				return &recipe, nil
			}
		}
	}

	// Get recipe template
	template, exists := r.recipeDefinitions[protocolID]
	if !exists {
		return nil, domain.ErrRecipeNotFound
	}

	// Build recipe
	recipe := &domain.WorkflowRecipe{
		RecipeID:   fmt.Sprintf("%s_%d", protocolID, time.Now().Unix()),
		ProtocolID: protocolID,
		Version:    template.Version,
		TTLSeconds: int64(r.config.DefaultRecipeTTL.Seconds()),
	}

	// Merge base requirements
	baseFields := []string{}
	baseFields = append(baseFields, r.getCalculationFields(protocolID)...)
	baseFields = append(baseFields, r.getSafetyFields(protocolID)...)
	baseFields = append(baseFields, r.getAuditFields()...)

	// Apply conditional fields
	conditionalFields, err := r.conditionalEngine.ResolveConditionalFields(
		contextNeeds,
		template.ConditionalRules,
		patientCharacteristics,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve conditional fields: %w", err)
	}

	// Merge and deduplicate fields
	recipe.RequiredFields = r.mergeAndDeduplicate(baseFields, conditionalFields)
	recipe.FreshnessRequirements = template.FreshnessRules
	recipe.AllowLiveFetch = false // Default to false for safety
	recipe.AllowedLiveFields = []string{}

	// Cache result
	if r.config.CacheEnabled {
		cacheKey := fmt.Sprintf("recipe:%s:%s", protocolID, contextNeeds.Hash())
		if recipeJSON, err := json.Marshal(recipe); err == nil {
			r.cache.Set(ctx, cacheKey, string(recipeJSON), r.config.CacheTTL)
		}
	}

	return recipe, nil
}

func (r *RecipeResolver) getCalculationFields(protocolID string) []string {
	// Protocol-specific calculation fields
	baseFields := []string{
		"demographics.weight_kg",
		"demographics.height_cm", 
		"demographics.age_years",
		"demographics.gender",
	}

	switch protocolID {
	case "hypertension-standard":
		return append(baseFields, []string{
			"vitals.blood_pressure",
			"labs.creatinine",
			"labs.potassium",
		}...)
	case "diabetes-management":
		return append(baseFields, []string{
			"labs.hba1c",
			"labs.glucose",
			"vitals.weight_history",
		}...)
	default:
		return baseFields
	}
}

func (r *RecipeResolver) getSafetyFields(protocolID string) []string {
	return []string{
		"allergies.drug_allergies",
		"conditions.current_conditions",
		"medications.current_medications",
		"conditions.contraindications",
	}
}

func (r *RecipeResolver) getAuditFields() []string {
	return []string{
		"encounter.provider_id",
		"encounter.encounter_id",
		"encounter.timestamp",
		"patient.patient_id",
	}
}

func (r *RecipeResolver) mergeAndDeduplicate(fieldLists ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, fields := range fieldLists {
		for _, field := range fields {
			if !seen[field] {
				seen[field] = true
				result = append(result, field)
			}
		}
	}

	return result
}

func (r *RecipeResolver) loadRecipeDefinitions() error {
	// Load recipe definitions from configuration files
	// This would typically load from YAML/JSON files or database
	
	r.recipeDefinitions["hypertension-standard"] = &domain.RecipeTemplate{
		ProtocolID:  "hypertension-standard",
		Version:     "1.2",
		Name:        "Standard Hypertension Management",
		Description: "Evidence-based hypertension treatment protocol",
		ConditionalRules: []*domain.ConditionalRule{
			{
				Condition:    "age < 18",
				AddFields:    []string{"vitals.growth_charts", "demographics.parent_consent"},
				Description:  "Pediatric requirements",
			},
			{
				Condition:    "pregnancy_status == true",
				AddFields:    []string{"obstetrics.pregnancy_stage", "medications.pregnancy_safe"},
				Description:  "Pregnancy considerations",
			},
		},
		FreshnessRules: map[string]time.Duration{
			"vitals":      24 * time.Hour,
			"labs":        7 * 24 * time.Hour,
			"medications": 24 * time.Hour,
			"allergies":   30 * 24 * time.Hour,
		},
	}

	return nil
}
```

### 3. Context Gateway Client Implementation

Create `internal/infrastructure/context_gateway_client.go`:
```go
package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"medication-service-v2/internal/domain"
)

type ContextGatewayClient struct {
	baseURL    string
	httpClient *http.Client
	config     *ContextGatewayConfig
}

type ContextGatewayConfig struct {
	BaseURL    string        `yaml:"base_url"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int          `yaml:"max_retries"`
}

func NewContextGatewayClient(config *ContextGatewayConfig) *ContextGatewayClient {
	return &ContextGatewayClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}
}

func (c *ContextGatewayClient) CreateSnapshot(
	ctx context.Context,
	recipe *domain.WorkflowRecipe,
	patientID string,
) (*domain.ClinicalSnapshot, error) {
	payload := map[string]interface{}{
		"patient_id":              patientID,
		"required_fields":         recipe.RequiredFields,
		"freshness_requirements":  recipe.FreshnessRequirements,
		"ttl_seconds":            recipe.TTLSeconds,
		"allow_live_fetch":       recipe.AllowLiveFetch,
		"allowed_live_fields":    recipe.AllowedLiveFields,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/snapshots", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	var lastErr error

	// Retry logic
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		resp, lastErr = c.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode < 500 {
			break
		}
		
		if resp != nil {
			resp.Body.Close()
		}

		if attempt < c.config.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to create snapshot after %d retries: %w", c.config.MaxRetries+1, lastErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snapshot creation failed with status %d", resp.StatusCode)
	}

	var snapshotResponse struct {
		Snapshot *domain.ClinicalSnapshot `json:"snapshot"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&snapshotResponse); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot response: %w", err)
	}

	return snapshotResponse.Snapshot, nil
}

func (c *ContextGatewayClient) GetSnapshot(ctx context.Context, snapshotID string) (*domain.ClinicalSnapshot, error) {
	url := fmt.Sprintf("%s/v1/snapshots/%s", c.baseURL, snapshotID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrSnapshotNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get snapshot: status %d", resp.StatusCode)
	}

	var snapshot domain.ClinicalSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot: %w", err)
	}

	return &snapshot, nil
}

func (c *ContextGatewayClient) ValidateSnapshot(ctx context.Context, snapshotID string) (*domain.SnapshotValidation, error) {
	url := fmt.Sprintf("%s/v1/snapshots/%s/validate", c.baseURL, snapshotID)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create validation request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snapshot validation failed: status %d", resp.StatusCode)
	}

	var validation domain.SnapshotValidation
	if err := json.NewDecoder(resp.Body).Decode(&validation); err != nil {
		return nil, fmt.Errorf("failed to decode validation response: %w", err)
	}

	return &validation, nil
}
```

## Database Configuration

### 1. Database Migration Setup

Create `migrations/001_initial_schema.sql`:
```sql
-- Medication Service V2 Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Medications table
CREATE TABLE medications (
    medication_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rxnorm_code VARCHAR(20) NOT NULL,
    brand_name VARCHAR(255),
    generic_name VARCHAR(255) NOT NULL,
    therapeutic_class VARCHAR(100),
    pharmacologic_class VARCHAR(100),
    dea_schedule INTEGER,
    is_high_alert BOOLEAN DEFAULT FALSE,
    is_controlled BOOLEAN DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uk_rxnorm UNIQUE (rxnorm_code)
);

-- Indexes for medications
CREATE INDEX idx_medications_generic_name ON medications(generic_name);
CREATE INDEX idx_medications_therapeutic_class ON medications(therapeutic_class);
CREATE INDEX idx_medications_search ON medications USING gin(
    to_tsvector('english', brand_name || ' ' || generic_name)
);

-- Proposals table (two-phase support)
CREATE TABLE medication_proposals (
    proposal_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PROPOSED',
    recipe_id VARCHAR(100) NOT NULL,
    snapshot_id UUID,
    proposals JSONB NOT NULL,
    snapshot_reference JSONB,
    evidence_envelope JSONB,
    processing_time_ms INTEGER,
    proposed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    proposed_by VARCHAR(100),
    committed_at TIMESTAMP WITH TIME ZONE,
    committed_by VARCHAR(100),
    commit_context JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT chk_status CHECK (status IN ('PROPOSED', 'COMMITTED', 'CANCELLED', 'EXPIRED'))
);

-- Indexes for proposals
CREATE INDEX idx_proposals_patient ON medication_proposals(patient_id);
CREATE INDEX idx_proposals_status ON medication_proposals(status);
CREATE INDEX idx_proposals_proposed_at ON medication_proposals(proposed_at);
CREATE INDEX idx_proposals_snapshot ON medication_proposals(snapshot_id);

-- Recipe templates table
CREATE TABLE recipe_templates (
    protocol_id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(20) NOT NULL,
    description TEXT,
    template_data JSONB NOT NULL,
    conditional_rules JSONB,
    freshness_rules JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Clinical snapshots cache table  
CREATE TABLE clinical_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    patient_id VARCHAR(100) NOT NULL,
    recipe_id VARCHAR(100) NOT NULL,
    data JSONB NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    signature TEXT,
    included_fields JSONB NOT NULL,
    metadata JSONB,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uk_checksum UNIQUE (checksum)
);

-- Indexes for snapshots
CREATE INDEX idx_snapshots_patient ON clinical_snapshots(patient_id);
CREATE INDEX idx_snapshots_recipe ON clinical_snapshots(recipe_id);
CREATE INDEX idx_snapshots_expires ON clinical_snapshots(expires_at);

-- Outbox events table for reliable event publishing
CREATE TABLE outbox_events (
    event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER DEFAULT 0,
    last_retry_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT
);

-- Indexes for outbox
CREATE INDEX idx_outbox_unpublished ON outbox_events(created_at) 
    WHERE published_at IS NULL;
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_type, aggregate_id);

-- Performance metrics table
CREATE TABLE performance_metrics (
    metric_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value FLOAT NOT NULL,
    dimensions JSONB,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for metrics
CREATE INDEX idx_metrics_name_time ON performance_metrics(metric_name, recorded_at);

-- Audit log table
CREATE TABLE audit_log (
    audit_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    actor VARCHAR(100) NOT NULL,
    changes JSONB,
    context JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for audit log
CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_actor ON audit_log(actor);
CREATE INDEX idx_audit_created ON audit_log(created_at);
```

### 2. Database Connection Setup

Create `internal/infrastructure/database.go`:
```go
package infrastructure

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DatabaseManager struct {
	db     *sqlx.DB
	config *DatabaseConfig
}

type DatabaseConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Name           string `yaml:"name"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	SSLMode        string `yaml:"ssl_mode"`
	MaxConnections int    `yaml:"max_connections"`
	MaxIdle        int    `yaml:"max_idle_connections"`
}

func NewDatabaseManager(config *DatabaseConfig) (*DatabaseManager, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Name, config.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxConnections)
	db.SetMaxIdleConns(config.MaxIdle)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	manager := &DatabaseManager{
		db:     db,
		config: config,
	}

	// Run migrations
	if err := manager.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return manager, nil
}

func (dm *DatabaseManager) runMigrations() error {
	driver, err := postgres.WithInstance(dm.db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", 
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (dm *DatabaseManager) GetDB() *sqlx.DB {
	return dm.db
}

func (dm *DatabaseManager) Close() error {
	return dm.db.Close()
}

func (dm *DatabaseManager) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return dm.db.PingContext(ctx)
}
```

## Testing Strategy

### 1. Unit Test Setup

Create `internal/application/medication_service_test.go`:
```go
package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"medication-service-v2/internal/application"
	"medication-service-v2/internal/domain"
	"medication-service-v2/tests/mocks"
)

func TestMedicationService_Phase1IngestAndResolve(t *testing.T) {
	tests := []struct {
		name        string
		request     *domain.MedicationRequest
		mockSetup   func(*mocks.MockRecipeResolver)
		expected    *domain.PhaseResult
		expectedErr string
	}{
		{
			name: "successful recipe resolution",
			request: &domain.MedicationRequest{
				PatientID:   "patient-123",
				Indication:  "hypertension",
				ClinicalContext: map[string]interface{}{
					"age":    45,
					"weight": 70.5,
				},
			},
			mockSetup: func(m *mocks.MockRecipeResolver) {
				m.On("ResolveWorkflowRecipe", 
					mock.Anything, 
					"hypertension-standard",
					mock.AnythingOfType("*domain.ContextNeeds"),
					mock.AnythingOfType("*domain.PatientCharacteristics"),
				).Return(&domain.WorkflowRecipe{
					RecipeID:   "hypertension-standard_123456",
					ProtocolID: "hypertension-standard",
					RequiredFields: []string{
						"demographics.weight_kg",
						"demographics.age_years",
						"vitals.blood_pressure",
					},
					TTLSeconds: 3600,
				}, nil)
			},
			expected: &domain.PhaseResult{
				Manifest: &domain.IntentManifest{
					ProtocolID: "hypertension-standard",
					TherapyOptions: []string{"ACE_inhibitor", "ARB", "diuretic"},
				},
				Recipe: &domain.WorkflowRecipe{
					RecipeID:   "hypertension-standard_123456",
					ProtocolID: "hypertension-standard",
				},
			},
		},
		{
			name: "invalid patient context",
			request: &domain.MedicationRequest{
				PatientID:  "patient-123",
				Indication: "hypertension",
				// Missing clinical context
			},
			mockSetup:   func(m *mocks.MockRecipeResolver) {},
			expected:    nil,
			expectedErr: "invalid patient context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRecipeResolver := &mocks.MockRecipeResolver{}
			mockContextGateway := &mocks.MockContextGatewayClient{}
			mockRustEngine := &mocks.MockRustEngineClient{}
			mockApolloClient := &mocks.MockApolloFederationClient{}
			mockCacheManager := &mocks.MockCacheManager{}
			mockEventPublisher := &mocks.MockEventPublisher{}

			tt.mockSetup(mockRecipeResolver)

			// Create service
			service := application.NewMedicationService(
				mockRecipeResolver,
				mockContextGateway,
				mockRustEngine,
				mockApolloClient,
				mockCacheManager,
				mockEventPublisher,
			)

			// Execute
			ctx := context.Background()
			result, err := service.Phase1IngestAndResolve(ctx, tt.request)

			// Assert
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Recipe.ProtocolID, result.Recipe.ProtocolID)
			}

			// Verify mocks
			mockRecipeResolver.AssertExpectations(t)
		})
	}
}

func TestMedicationService_CompleteWorkflow(t *testing.T) {
	// Integration test for complete 4-phase workflow
	service := setupTestService(t)
	
	ctx := context.Background()
	request := &domain.MedicationRequest{
		PatientID:   "patient-123",
		Indication:  "hypertension",
		ClinicalContext: map[string]interface{}{
			"age_years": 45,
			"weight_kg": 70.5,
		},
		Preferences: &domain.MedicationPreferences{
			Route:     "PO",
			Frequency: "daily",
		},
	}

	// Phase 1: Recipe Resolution
	start := time.Now()
	phase1Result, err := service.Phase1IngestAndResolve(ctx, request)
	require.NoError(t, err)
	assert.NotNil(t, phase1Result.Recipe)
	
	// Phase 2: Snapshot Creation
	snapshot, err := service.Phase2AssembleContext(ctx, phase1Result.Recipe, request.PatientID)
	require.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.NotEmpty(t, snapshot.ID)

	// Phase 3: Clinical Intelligence
	proposals, err := service.Phase3ClinicalIntelligence(ctx, phase1Result.Manifest, snapshot)
	require.NoError(t, err)
	assert.Greater(t, len(proposals), 0)

	// Phase 4: Proposal Generation  
	finalProposal, err := service.Phase4GenerateProposal(ctx, proposals, snapshot, phase1Result.Manifest)
	require.NoError(t, err)
	assert.NotNil(t, finalProposal)
	assert.Equal(t, snapshot.ID, finalProposal.SnapshotReference.SnapshotID)

	// Verify performance target (<250ms)
	duration := time.Since(start)
	assert.Less(t, duration, 250*time.Millisecond, "Workflow should complete within 250ms")
}

func setupTestService(t *testing.T) *application.MedicationService {
	// Setup test dependencies with real implementations or mocks as needed
	// This would typically use test containers for integration tests
	return nil // Implementation depends on test infrastructure
}
```

### 2. Integration Test Setup

Create `tests/integration/api_test.go`:
```go
package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"

	"medication-service-v2/internal/application"
	"medication-service-v2/internal/infrastructure"
	"medication-service-v2/internal/interfaces/http_server"
)

type IntegrationTestSuite struct {
	router     *gin.Engine
	service    *application.MedicationService
	pgContainer testcontainers.Container
	redisContainer testcontainers.Container
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase("medication_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	require.NoError(t, err)

	// Start Redis container
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7"),
	)
	require.NoError(t, err)

	// Get connection details
	pgHost, err := pgContainer.Host(ctx)
	require.NoError(t, err)
	pgPort, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	redisHost, err := redisContainer.Host(ctx)
	require.NoError(t, err)
	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	// Setup configuration
	config := &infrastructure.Config{
		Database: infrastructure.DatabaseConfig{
			Host:     pgHost,
			Port:     pgPort.Int(),
			Name:     "medication_test",
			User:     "test",
			Password: "test",
			SSLMode:  "disable",
		},
		Redis: infrastructure.RedisConfig{
			URL: fmt.Sprintf("redis://%s:%s", redisHost, redisPort.Port()),
		},
	}

	// Initialize service
	deps, err := initializeDependencies(config)
	require.NoError(t, err)

	service := application.NewMedicationService(deps...)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	http_server.SetupRoutes(router, service, config)

	return &IntegrationTestSuite{
		router:         router,
		service:        service,
		pgContainer:    pgContainer,
		redisContainer: redisContainer,
	}
}

func (suite *IntegrationTestSuite) cleanup() {
	ctx := context.Background()
	suite.pgContainer.Terminate(ctx)
	suite.redisContainer.Terminate(ctx)
}

func TestMedicationProposalAPI(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	tests := []struct {
		name           string
		request        map[string]interface{}
		expectedStatus int
		assertions     func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful medication proposal",
			request: map[string]interface{}{
				"patient_id":  "patient-123",
				"indication":  "hypertension",
				"clinical_context": map[string]interface{}{
					"weight_kg": 70.5,
					"age_years": 45,
					"gender":    "M",
				},
				"preferences": map[string]interface{}{
					"route":     "PO",
					"frequency": "daily",
				},
			},
			expectedStatus: http.StatusOK,
			assertions: func(t *testing.T, response map[string]interface{}) {
				assert.NotEmpty(t, response["proposal_id"])
				assert.Equal(t, "PROPOSED", response["status"])
				assert.NotEmpty(t, response["proposals"])
				
				processingTime := response["processing_time_ms"].(float64)
				assert.Less(t, processingTime, 250.0, "Should meet performance target")
				
				proposals := response["proposals"].([]interface{})
				assert.Greater(t, len(proposals), 0, "Should have at least one proposal")
				
				firstProposal := proposals[0].(map[string]interface{})
				assert.Equal(t, float64(1), firstProposal["rank"])
				assert.NotEmpty(t, firstProposal["medication"])
				assert.NotEmpty(t, firstProposal["dose"])
			},
		},
		{
			name: "invalid patient context",
			request: map[string]interface{}{
				"patient_id": "patient-123",
				"indication": "hypertension",
				// Missing clinical_context
			},
			expectedStatus: http.StatusBadRequest,
			assertions: func(t *testing.T, response map[string]interface{}) {
				assert.NotEmpty(t, response["error"])
				errorInfo := response["error"].(map[string]interface{})
				assert.Equal(t, "INVALID_PATIENT_CONTEXT", errorInfo["code"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/v1/medications/propose", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")

			// Execute request
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Assert status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			// Run custom assertions
			tt.assertions(t, response)
		})
	}
}

func TestHealthEndpoints(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.cleanup()

	tests := []struct {
		endpoint       string
		expectedStatus int
		assertions     func(*testing.T, map[string]interface{})
	}{
		{
			endpoint:       "/health/live",
			expectedStatus: http.StatusOK,
			assertions: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "healthy", response["status"])
				assert.Equal(t, "medication-service-v2", response["service"])
			},
		},
		{
			endpoint:       "/health/ready",
			expectedStatus: http.StatusOK,
			assertions: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "ready", response["status"])
				
				deps := response["dependencies"].(map[string]interface{})
				assert.Equal(t, "healthy", deps["database"])
				assert.Equal(t, "healthy", deps["redis"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()
			
			suite.router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			tt.assertions(t, response)
		})
	}
}
```

## Performance Optimization

### 1. Caching Strategy Implementation

Create `internal/infrastructure/cache_manager.go`:
```go
package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type CacheManager struct {
	client *redis.Client
	config *CacheConfig
}

type CacheConfig struct {
	URL            string                    `yaml:"url"`
	MaxConnections int                      `yaml:"max_connections"`
	Timeout        time.Duration            `yaml:"timeout"`
	TTLs          map[string]time.Duration `yaml:"ttls"`
}

func NewCacheManager(config *CacheConfig) (*CacheManager, error) {
	opt, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	opt.PoolSize = config.MaxConnections
	opt.ReadTimeout = config.Timeout
	opt.WriteTimeout = config.Timeout

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheManager{
		client: client,
		config: config,
	}, nil
}

func (c *CacheManager) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *CacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	return c.client.Set(ctx, key, valueJSON, ttl).Err()
}

func (c *CacheManager) GetOrSet(
	ctx context.Context,
	key string,
	fetcher func() (interface{}, error),
	ttlType string,
) (interface{}, error) {
	// Try to get from cache first
	cached, err := c.client.Get(ctx, key).Result()
	if err == nil {
		var result interface{}
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
	}

	// Cache miss - fetch the data
	result, err := fetcher()
	if err != nil {
		return nil, err
	}

	// Cache the result
	ttl := c.getTTL(ttlType)
	resultJSON, err := json.Marshal(result)
	if err == nil {
		// Don't fail the request if caching fails
		c.client.Set(ctx, key, resultJSON, ttl)
	}

	return result, nil
}

func (c *CacheManager) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *CacheManager) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}

	return nil
}

func (c *CacheManager) getTTL(ttlType string) time.Duration {
	if ttl, exists := c.config.TTLs[ttlType]; exists {
		return ttl
	}
	return 5 * time.Minute // default TTL
}

func (c *CacheManager) Close() error {
	return c.client.Close()
}

func (c *CacheManager) HealthCheck(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Cache invalidation utilities
func (c *CacheManager) InvalidateMedicationCache(ctx context.Context, medicationID string) error {
	patterns := []string{
		fmt.Sprintf("medication:%s:*", medicationID),
		fmt.Sprintf("formulary:*:%s", medicationID),
		"search:*", // Invalidate all search results
	}

	for _, pattern := range patterns {
		if err := c.DeletePattern(ctx, pattern); err != nil {
			return fmt.Errorf("failed to invalidate pattern %s: %w", pattern, err)
		}
	}

	return nil
}

func (c *CacheManager) InvalidatePatientCache(ctx context.Context, patientID string) error {
	patterns := []string{
		fmt.Sprintf("patient:%s:*", patientID),
		fmt.Sprintf("proposal:*:%s", patientID),
		fmt.Sprintf("snapshot:*:%s", patientID),
	}

	for _, pattern := range patterns {
		if err := c.DeletePattern(ctx, pattern); err != nil {
			return fmt.Errorf("failed to invalidate pattern %s: %w", pattern, err)
		}
	}

	return nil
}
```

### 2. Connection Pooling and Resource Management

Create `internal/infrastructure/resource_pool.go`:
```go
package infrastructure

import (
	"context"
	"sync"
	"time"
)

type ResourcePool struct {
	resources chan interface{}
	factory   func() (interface{}, error)
	cleanup   func(interface{}) error
	maxSize   int
	created   int
	mu        sync.RWMutex
}

func NewResourcePool(
	maxSize int,
	factory func() (interface{}, error),
	cleanup func(interface{}) error,
) *ResourcePool {
	return &ResourcePool{
		resources: make(chan interface{}, maxSize),
		factory:   factory,
		cleanup:   cleanup,
		maxSize:   maxSize,
	}
}

func (p *ResourcePool) Get(ctx context.Context) (interface{}, error) {
	select {
	case resource := <-p.resources:
		return resource, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// No resource available, create new one if under limit
		p.mu.Lock()
		if p.created < p.maxSize {
			p.created++
			p.mu.Unlock()
			
			resource, err := p.factory()
			if err != nil {
				p.mu.Lock()
				p.created--
				p.mu.Unlock()
				return nil, err
			}
			return resource, nil
		}
		p.mu.Unlock()

		// Wait for resource to become available
		select {
		case resource := <-p.resources:
			return resource, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (p *ResourcePool) Put(resource interface{}) {
	select {
	case p.resources <- resource:
		// Resource returned to pool
	default:
		// Pool is full, cleanup resource
		if p.cleanup != nil {
			p.cleanup(resource)
		}
		p.mu.Lock()
		p.created--
		p.mu.Unlock()
	}
}

func (p *ResourcePool) Close() error {
	close(p.resources)
	
	// Cleanup all resources in pool
	for resource := range p.resources {
		if p.cleanup != nil {
			p.cleanup(resource)
		}
	}
	
	return nil
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Port Conflicts
```bash
# Check for port conflicts
netstat -tulpn | grep -E ':(8005|8085|8095|8086|8089)'

# If ports are in use, update configuration or stop conflicting services
sudo systemctl stop existing-service
```

#### 2. Database Connection Issues
```bash
# Test database connectivity
psql -h localhost -p 5434 -U medication_user -d medication_v2 -c "SELECT 1;"

# Check database logs
docker logs medication-service-v2-postgres-1

# Verify migration status
docker exec -it medication-service-v2-postgres-1 psql -U medication_user -d medication_v2 -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 5;"
```

#### 3. Redis Connection Issues
```bash
# Test Redis connectivity
redis-cli -h localhost -p 6381 ping

# Check Redis logs
docker logs medication-service-v2-redis-1

# Monitor Redis performance
redis-cli -h localhost -p 6381 --latency-history -i 1
```

#### 4. Service Health Issues
```bash
# Check service health
curl http://localhost:8005/health/ready

# Check dependency health
curl http://localhost:8005/health/deps

# View service logs
make logs
```

#### 5. Performance Issues
```bash
# Check metrics endpoint
curl http://localhost:8005/metrics | grep medication_v2

# Monitor resource usage
docker stats

# Check database performance
docker exec -it medication-service-v2-postgres-1 psql -U medication_user -d medication_v2 -c "SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

### Debug Configuration

Create `configs/debug.yaml`:
```yaml
service:
  port: 8005
  log_level: debug
  enable_pprof: true
  pprof_port: 6060

database:
  log_queries: true
  slow_query_threshold: 100ms

redis:
  log_commands: true

monitoring:
  enable_debug_endpoints: true
  enable_request_logging: true
  log_response_times: true

clinical_engine:
  log_calculations: true
  debug_mode: true
```

### Logging Configuration

Create `internal/infrastructure/logger.go`:
```go
package infrastructure

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func NewLogger(config *LogConfig) *Logger {
	logger := logrus.New()
	
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	return &Logger{Logger: logger}
}

func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.Logger.WithContext(ctx)
	
	// Add trace ID from context if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		entry = entry.WithField("trace_id", traceID)
	}
	
	// Add patient ID from context if available
	if patientID := ctx.Value("patient_id"); patientID != nil {
		entry = entry.WithField("patient_id", patientID)
	}

	return entry
}

func (l *Logger) LogMedicationProposal(ctx context.Context, proposalID, patientID string, duration time.Duration) {
	l.WithContext(ctx).WithFields(logrus.Fields{
		"proposal_id":     proposalID,
		"patient_id":      patientID,
		"duration_ms":     duration.Milliseconds(),
		"operation":       "medication_proposal",
		"service":         "medication-service-v2",
	}).Info("Medication proposal completed")
}

func (l *Logger) LogRecipeResolution(ctx context.Context, protocolID, recipeID string, duration time.Duration) {
	l.WithContext(ctx).WithFields(logrus.Fields{
		"protocol_id":     protocolID,
		"recipe_id":       recipeID,
		"duration_ms":     duration.Milliseconds(),
		"operation":       "recipe_resolution",
		"service":         "medication-service-v2",
	}).Info("Recipe resolution completed")
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}
```

This implementation guide provides a comprehensive roadmap for implementing the Go/Rust Medication Service V2 while avoiding conflicts with the existing Python service. The guide covers all aspects from initial setup through troubleshooting, ensuring a smooth transition to the high-performance architecture.