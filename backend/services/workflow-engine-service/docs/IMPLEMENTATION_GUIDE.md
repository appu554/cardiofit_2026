# Workflow Engine Service - Implementation Guide

## Overview

This comprehensive implementation guide provides step-by-step instructions for converting the Python-based workflow engine service to either Go or Rust. Both implementations maintain the strategic orchestration architecture while delivering enterprise-grade performance for clinical workflow management.

## Architecture Decision Matrix

| Aspect | Python (Current) | Go Implementation | Rust Implementation |
|--------|------------------|-------------------|---------------------|
| **Performance** | Moderate (interpreted) | High (compiled, GC) | Highest (compiled, no GC) |
| **Memory Usage** | High (100-150MB) | Medium (30-60MB) | Lowest (20-40MB) |
| **Development Speed** | Fast | Moderate | Slower (learning curve) |
| **Type Safety** | Runtime (with typing) | Compile-time | Compile-time + ownership |
| **Concurrency** | asyncio (GIL limitations) | Goroutines (excellent) | async/await (zero-cost) |
| **Error Handling** | Exceptions | Explicit error returns | Result<T, E> types |
| **Ecosystem** | Very mature | Mature | Growing rapidly |
| **Learning Curve** | Low | Moderate | High |
| **Deployment** | Multi-file + dependencies | Single binary | Single binary |
| **Clinical Safety** | Good with discipline | Good with types | Excellent with ownership |

## Recommended Choice by Use Case

### Choose Go When:
- **Team Familiarity**: Team has C/Java background
- **Rapid Development**: Need faster time-to-market
- **Microservices**: Building distributed service architecture
- **Network Programming**: Heavy focus on HTTP/gRPC services
- **Operational Simplicity**: Single binary deployment preferred

### Choose Rust When:
- **Maximum Performance**: Sub-millisecond latency requirements
- **Safety Critical**: Patient safety is paramount concern  
- **High Concurrency**: Thousands of concurrent workflows
- **Long-term Maintenance**: Building for 10+ year lifecycle
- **Resource Constraints**: Running in memory-limited environments

## Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2)

#### Go Implementation
```bash
# Project initialization
mkdir workflow-engine-go
cd workflow-engine-go
go mod init github.com/clinical-synthesis-hub/workflow-engine

# Create directory structure
mkdir -p {cmd/server,internal/{config,domain,repository,service,handler,middleware,orchestration,clinical,monitoring,auth,database},pkg/{fhir,camunda,safety,medication},api/{proto,graphql,openapi},migrations,configs,deployments}

# Core dependencies
go get github.com/gin-gonic/gin
go get github.com/kelseyhightower/envconfig
go get github.com/jmoiron/sqlx
go get github.com/lib/pq
go get go.uber.org/zap
go get go.opentelemetry.io/otel
```

#### Rust Implementation  
```bash
# Project initialization
cargo new workflow-engine-rust --bin
cd workflow-engine-rust

# Create workspace structure  
mkdir -p {src/{config,domain,infrastructure,application,presentation,utils},migrations,proto,configs}

# Add dependencies to Cargo.toml
cargo add tokio --features full
cargo add axum
cargo add serde --features derive
cargo add sqlx --features runtime-tokio-rustls,postgres,uuid,chrono,json
cargo add tracing
cargo add config
```

### Phase 2: Core Domain (Weeks 3-4)

#### Domain Models Implementation

**Go Domain Models:**
```go
// internal/domain/workflow.go
package domain

import (
    "time"
    "database/sql/driver"
    "encoding/json"
)

type WorkflowStatus string

const (
    WorkflowStatusPending    WorkflowStatus = "pending"
    WorkflowStatusRunning    WorkflowStatus = "running"
    WorkflowStatusCompleted  WorkflowStatus = "completed"
    WorkflowStatusFailed     WorkflowStatus = "failed"
    WorkflowStatusCancelled  WorkflowStatus = "cancelled"
)

type WorkflowDefinition struct {
    ID          string                 `json:"id" db:"id"`
    Name        string                 `json:"name" db:"name"`
    Version     string                 `json:"version" db:"version"`
    Description string                 `json:"description" db:"description"`
    BPMNData    string                 `json:"bpmn_data" db:"bpmn_data"`
    Variables   map[string]interface{} `json:"variables" db:"variables"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
    Active      bool                   `json:"active" db:"active"`
}

func (w WorkflowDefinition) Value() (driver.Value, error) {
    return json.Marshal(w)
}

func (w *WorkflowDefinition) Scan(value interface{}) error {
    if value == nil {
        return nil
    }
    return json.Unmarshal(value.([]byte), w)
}

type WorkflowInstance struct {
    ID               string                 `json:"id" db:"id"`
    DefinitionID     string                 `json:"definition_id" db:"definition_id"`
    PatientID        string                 `json:"patient_id" db:"patient_id"`
    Status           WorkflowStatus         `json:"status" db:"status"`
    Variables        map[string]interface{} `json:"variables" db:"variables"`
    StartTime        time.Time              `json:"start_time" db:"start_time"`
    EndTime          *time.Time             `json:"end_time,omitempty" db:"end_time"`
    CorrelationID    string                 `json:"correlation_id" db:"correlation_id"`
    SnapshotChain    *SnapshotChainTracker  `json:"snapshot_chain,omitempty" db:"snapshot_chain"`
    CreatedAt        time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}
```

**Rust Domain Models:**
```rust
// src/domain/workflow.rs
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::Type;
use std::collections::HashMap;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, Type)]
#[sqlx(type_name = "workflow_status", rename_all = "lowercase")]
pub enum WorkflowStatus {
    Pending,
    Running,
    Completed,
    Failed,
    Cancelled,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkflowDefinition {
    pub id: String,
    pub name: String,
    pub version: String,
    pub description: String,
    pub bpmn_data: String,
    pub variables: HashMap<String, serde_json::Value>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub active: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkflowInstance {
    pub id: String,
    pub definition_id: String,
    pub patient_id: String,
    pub status: WorkflowStatus,
    pub variables: HashMap<String, serde_json::Value>,
    pub start_time: DateTime<Utc>,
    pub end_time: Option<DateTime<Utc>>,
    pub correlation_id: String,
    pub snapshot_chain: Option<crate::domain::snapshot::SnapshotChainTracker>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl WorkflowInstance {
    pub fn new(
        definition_id: String,
        patient_id: String,
        correlation_id: String,
        variables: HashMap<String, serde_json::Value>,
    ) -> Self {
        let now = Utc::now();
        
        Self {
            id: format!("workflow_{}", Uuid::new_v4().simple()),
            definition_id,
            patient_id,
            status: WorkflowStatus::Pending,
            variables,
            start_time: now,
            end_time: None,
            correlation_id,
            snapshot_chain: None,
            created_at: now,
            updated_at: now,
        }
    }
    
    pub fn start(&mut self) {
        self.status = WorkflowStatus::Running;
        self.start_time = Utc::now();
        self.updated_at = Utc::now();
    }
    
    pub fn complete(&mut self) {
        self.status = WorkflowStatus::Completed;
        self.end_time = Some(Utc::now());
        self.updated_at = Utc::now();
    }
    
    pub fn fail(&mut self) {
        self.status = WorkflowStatus::Failed;
        self.end_time = Some(Utc::now());
        self.updated_at = Utc::now();
    }
    
    pub fn duration(&self) -> Option<chrono::Duration> {
        self.end_time.map(|end| end - self.start_time)
    }
    
    pub fn is_terminal(&self) -> bool {
        matches!(
            self.status,
            WorkflowStatus::Completed | WorkflowStatus::Failed | WorkflowStatus::Cancelled
        )
    }
}
```

### Phase 3: Database Layer (Weeks 5-6)

#### Database Schema Migration

**SQL Schema (Common to both):**
```sql
-- migrations/001_initial_schema.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Workflow definitions
CREATE TABLE workflow_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR NOT NULL,
    version VARCHAR NOT NULL,
    description TEXT,
    bpmn_data TEXT NOT NULL,
    variables JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    active BOOLEAN DEFAULT true,
    UNIQUE(name, version)
);

-- Workflow instances
CREATE TYPE workflow_status AS ENUM ('pending', 'running', 'completed', 'failed', 'cancelled');

CREATE TABLE workflow_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    definition_id UUID REFERENCES workflow_definitions(id),
    patient_id VARCHAR NOT NULL,
    status workflow_status DEFAULT 'pending',
    variables JSONB DEFAULT '{}',
    start_time TIMESTAMPTZ DEFAULT NOW(),
    end_time TIMESTAMPTZ,
    correlation_id VARCHAR NOT NULL,
    snapshot_chain JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Snapshot tracking
CREATE TYPE snapshot_status AS ENUM ('created', 'active', 'expired', 'archived', 'corrupted');
CREATE TYPE workflow_phase AS ENUM ('calculate', 'validate', 'commit', 'override');

CREATE TABLE snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    snapshot_id VARCHAR UNIQUE NOT NULL,
    checksum VARCHAR NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    status snapshot_status DEFAULT 'created',
    phase_created workflow_phase NOT NULL,
    patient_id VARCHAR NOT NULL,
    context_version VARCHAR NOT NULL,
    metadata JSONB DEFAULT '{}',
    data JSONB NOT NULL
);

-- Workflow tasks
CREATE TYPE task_status AS ENUM ('created', 'assigned', 'in_progress', 'completed', 'cancelled');

CREATE TABLE workflow_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id),
    task_definition_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    assignee_id VARCHAR,
    status task_status DEFAULT 'created',
    variables JSONB DEFAULT '{}',
    due_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Workflow events (audit trail)
CREATE TABLE workflow_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id),
    task_id UUID REFERENCES workflow_tasks(id),
    event_type VARCHAR NOT NULL,
    event_data JSONB DEFAULT '{}',
    user_id VARCHAR,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Performance and monitoring
CREATE TABLE workflow_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id),
    metric_name VARCHAR NOT NULL,
    metric_value NUMERIC NOT NULL,
    recorded_at TIMESTAMPTZ DEFAULT NOW(),
    correlation_id VARCHAR
);

-- Indexes for performance
CREATE INDEX idx_workflow_instances_patient_id ON workflow_instances(patient_id);
CREATE INDEX idx_workflow_instances_status ON workflow_instances(status);
CREATE INDEX idx_workflow_instances_correlation_id ON workflow_instances(correlation_id);
CREATE INDEX idx_snapshots_snapshot_id ON snapshots(snapshot_id);
CREATE INDEX idx_snapshots_patient_id ON snapshots(patient_id);
CREATE INDEX idx_snapshots_expires_at ON snapshots(expires_at);
CREATE INDEX idx_workflow_tasks_assignee_id ON workflow_tasks(assignee_id);
CREATE INDEX idx_workflow_tasks_status ON workflow_tasks(status);
CREATE INDEX idx_workflow_events_workflow_instance_id ON workflow_events(workflow_instance_id);
CREATE INDEX idx_workflow_metrics_correlation_id ON workflow_metrics(correlation_id);
```

#### Go Database Layer
```go
// internal/repository/workflow_repository.go
package repository

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    
    "github.com/jmoiron/sqlx"
    "github.com/clinical-synthesis-hub/workflow-engine/internal/domain"
)

type WorkflowRepository interface {
    CreateDefinition(ctx context.Context, def *domain.WorkflowDefinition) error
    GetDefinition(ctx context.Context, id string) (*domain.WorkflowDefinition, error)
    CreateInstance(ctx context.Context, instance *domain.WorkflowInstance) error
    GetInstance(ctx context.Context, id string) (*domain.WorkflowInstance, error)
    UpdateInstance(ctx context.Context, instance *domain.WorkflowInstance) error
    GetInstancesByPatient(ctx context.Context, patientID string) ([]*domain.WorkflowInstance, error)
}

type workflowRepository struct {
    db *sqlx.DB
}

func NewWorkflowRepository(db *sqlx.DB) WorkflowRepository {
    return &workflowRepository{db: db}
}

func (r *workflowRepository) CreateDefinition(ctx context.Context, def *domain.WorkflowDefinition) error {
    query := `
        INSERT INTO workflow_definitions 
        (id, name, version, description, bpmn_data, variables, active)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
    
    variables, err := json.Marshal(def.Variables)
    if err != nil {
        return fmt.Errorf("failed to marshal variables: %w", err)
    }
    
    _, err = r.db.ExecContext(ctx, query,
        def.ID, def.Name, def.Version, def.Description,
        def.BPMNData, variables, def.Active)
    
    return err
}

func (r *workflowRepository) GetDefinition(ctx context.Context, id string) (*domain.WorkflowDefinition, error) {
    query := `
        SELECT id, name, version, description, bpmn_data, variables, 
               created_at, updated_at, active
        FROM workflow_definitions 
        WHERE id = $1
    `
    
    var def domain.WorkflowDefinition
    var variables []byte
    
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &def.ID, &def.Name, &def.Version, &def.Description,
        &def.BPMNData, &variables, &def.CreatedAt, &def.UpdatedAt, &def.Active)
    
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("workflow definition not found: %s", id)
        }
        return nil, err
    }
    
    if err := json.Unmarshal(variables, &def.Variables); err != nil {
        return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
    }
    
    return &def, nil
}

func (r *workflowRepository) CreateInstance(ctx context.Context, instance *domain.WorkflowInstance) error {
    query := `
        INSERT INTO workflow_instances 
        (id, definition_id, patient_id, status, variables, start_time, 
         correlation_id, snapshot_chain)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
    
    variables, err := json.Marshal(instance.Variables)
    if err != nil {
        return fmt.Errorf("failed to marshal variables: %w", err)
    }
    
    var snapshotChain []byte
    if instance.SnapshotChain != nil {
        snapshotChain, err = json.Marshal(instance.SnapshotChain)
        if err != nil {
            return fmt.Errorf("failed to marshal snapshot chain: %w", err)
        }
    }
    
    _, err = r.db.ExecContext(ctx, query,
        instance.ID, instance.DefinitionID, instance.PatientID,
        instance.Status, variables, instance.StartTime,
        instance.CorrelationID, snapshotChain)
    
    return err
}

// Additional methods...
```

#### Rust Database Layer
```rust
// src/infrastructure/database/workflow_repository.rs
use crate::domain::workflow::{WorkflowDefinition, WorkflowInstance, WorkflowStatus};
use crate::utils::error::{DatabaseError, Result};

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use sqlx::{PgPool, Row};
use std::collections::HashMap;
use uuid::Uuid;

#[async_trait]
pub trait WorkflowRepository: Send + Sync {
    async fn create_definition(&self, definition: &WorkflowDefinition) -> Result<()>;
    async fn get_definition(&self, id: &str) -> Result<Option<WorkflowDefinition>>;
    async fn create_instance(&self, instance: &WorkflowInstance) -> Result<()>;
    async fn get_instance(&self, id: &str) -> Result<Option<WorkflowInstance>>;
    async fn update_instance(&self, instance: &WorkflowInstance) -> Result<()>;
    async fn get_instances_by_patient(&self, patient_id: &str) -> Result<Vec<WorkflowInstance>>;
}

pub struct PostgresWorkflowRepository {
    pool: PgPool,
}

impl PostgresWorkflowRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl WorkflowRepository for PostgresWorkflowRepository {
    #[tracing::instrument(skip(self, definition))]
    async fn create_definition(&self, definition: &WorkflowDefinition) -> Result<()> {
        let query = r#"
            INSERT INTO workflow_definitions 
            (id, name, version, description, bpmn_data, variables, active)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
        "#;
        
        let variables = serde_json::to_value(&definition.variables)
            .map_err(|e| DatabaseError::SerializationError(e.to_string()))?;
        
        sqlx::query(query)
            .bind(&definition.id)
            .bind(&definition.name)
            .bind(&definition.version)
            .bind(&definition.description)
            .bind(&definition.bpmn_data)
            .bind(&variables)
            .bind(definition.active)
            .execute(&self.pool)
            .await
            .map_err(|e| DatabaseError::QueryError(e.to_string()))?;
        
        Ok(())
    }
    
    #[tracing::instrument(skip(self))]
    async fn get_definition(&self, id: &str) -> Result<Option<WorkflowDefinition>> {
        let query = r#"
            SELECT id, name, version, description, bpmn_data, variables, 
                   created_at, updated_at, active
            FROM workflow_definitions 
            WHERE id = $1
        "#;
        
        let row = sqlx::query(query)
            .bind(id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DatabaseError::QueryError(e.to_string()))?;
        
        match row {
            Some(row) => {
                let variables: serde_json::Value = row.get("variables");
                let variables: HashMap<String, serde_json::Value> = serde_json::from_value(variables)
                    .map_err(|e| DatabaseError::DeserializationError(e.to_string()))?;
                
                let definition = WorkflowDefinition {
                    id: row.get("id"),
                    name: row.get("name"),
                    version: row.get("version"),
                    description: row.get("description"),
                    bpmn_data: row.get("bpmn_data"),
                    variables,
                    created_at: row.get("created_at"),
                    updated_at: row.get("updated_at"),
                    active: row.get("active"),
                };
                
                Ok(Some(definition))
            }
            None => Ok(None),
        }
    }
    
    #[tracing::instrument(skip(self, instance))]
    async fn create_instance(&self, instance: &WorkflowInstance) -> Result<()> {
        let query = r#"
            INSERT INTO workflow_instances 
            (id, definition_id, patient_id, status, variables, start_time, 
             correlation_id, snapshot_chain)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        "#;
        
        let variables = serde_json::to_value(&instance.variables)
            .map_err(|e| DatabaseError::SerializationError(e.to_string()))?;
            
        let snapshot_chain = instance.snapshot_chain.as_ref()
            .map(|chain| serde_json::to_value(chain))
            .transpose()
            .map_err(|e| DatabaseError::SerializationError(e.to_string()))?;
        
        sqlx::query(query)
            .bind(&instance.id)
            .bind(&instance.definition_id)
            .bind(&instance.patient_id)
            .bind(&instance.status)
            .bind(&variables)
            .bind(instance.start_time)
            .bind(&instance.correlation_id)
            .bind(&snapshot_chain)
            .execute(&self.pool)
            .await
            .map_err(|e| DatabaseError::QueryError(e.to_string()))?;
        
        Ok(())
    }
    
    #[tracing::instrument(skip(self))]
    async fn get_instance(&self, id: &str) -> Result<Option<WorkflowInstance>> {
        let query = r#"
            SELECT id, definition_id, patient_id, status, variables, 
                   start_time, end_time, correlation_id, snapshot_chain,
                   created_at, updated_at
            FROM workflow_instances 
            WHERE id = $1
        "#;
        
        let row = sqlx::query(query)
            .bind(id)
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DatabaseError::QueryError(e.to_string()))?;
        
        match row {
            Some(row) => {
                let variables: serde_json::Value = row.get("variables");
                let variables: HashMap<String, serde_json::Value> = serde_json::from_value(variables)
                    .map_err(|e| DatabaseError::DeserializationError(e.to_string()))?;
                
                let snapshot_chain: Option<serde_json::Value> = row.get("snapshot_chain");
                let snapshot_chain = snapshot_chain
                    .map(|chain| serde_json::from_value(chain))
                    .transpose()
                    .map_err(|e| DatabaseError::DeserializationError(e.to_string()))?;
                
                let instance = WorkflowInstance {
                    id: row.get("id"),
                    definition_id: row.get("definition_id"),
                    patient_id: row.get("patient_id"),
                    status: row.get("status"),
                    variables,
                    start_time: row.get("start_time"),
                    end_time: row.get("end_time"),
                    correlation_id: row.get("correlation_id"),
                    snapshot_chain,
                    created_at: row.get("created_at"),
                    updated_at: row.get("updated_at"),
                };
                
                Ok(Some(instance))
            }
            None => Ok(None),
        }
    }
    
    // Additional methods...
}
```

### Phase 4: API Layer (Weeks 7-8)

#### Go HTTP Handlers
```go
// internal/handler/workflow_handler.go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/clinical-synthesis-hub/workflow-engine/internal/service"
    "go.uber.org/zap"
)

type WorkflowHandler struct {
    orchestrationService service.OrchestrationService
    logger              *zap.Logger
}

func NewWorkflowHandler(
    orchestrationService service.OrchestrationService,
    logger *zap.Logger,
) *WorkflowHandler {
    return &WorkflowHandler{
        orchestrationService: orchestrationService,
        logger:              logger,
    }
}

func (h *WorkflowHandler) OrchestrateMedicationRequest(c *gin.Context) {
    var request service.CalculateRequest
    
    if err := c.ShouldBindJSON(&request); err != nil {
        h.logger.Error("Invalid request body", zap.Error(err))
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request body",
            "details": err.Error(),
        })
        return
    }
    
    result, err := h.orchestrationService.OrchestrateMedicationRequest(c.Request.Context(), &request)
    if err != nil {
        h.logger.Error("Orchestration failed", 
            zap.String("correlation_id", request.CorrelationID),
            zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Orchestration failed",
            "correlation_id": request.CorrelationID,
        })
        return
    }
    
    c.JSON(http.StatusOK, result)
}

func (h *WorkflowHandler) HealthCheck(c *gin.Context) {
    health, err := h.orchestrationService.HealthCheck(c.Request.Context())
    if err != nil {
        h.logger.Error("Health check failed", zap.Error(err))
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "status": "unhealthy",
            "error": err.Error(),
        })
        return
    }
    
    c.JSON(http.StatusOK, health)
}
```

#### Rust HTTP Handlers
```rust
// src/presentation/rest/handlers.rs
use crate::application::orchestration::OrchestrationService;
use crate::domain::orchestration::CalculateRequest;
use crate::utils::error::Result;

use axum::{
    extract::State,
    http::StatusCode,
    response::Json,
    Extension,
};
use serde_json::Value;
use std::sync::Arc;
use tracing::{error, info};

pub async fn orchestrate_medication_request(
    State(orchestration_service): State<Arc<dyn OrchestrationService>>,
    Json(request): Json<CalculateRequest>,
) -> Result<Json<Value>, StatusCode> {
    info!(
        correlation_id = %request.correlation_id,
        patient_id = %request.patient_id,
        "Received medication orchestration request"
    );
    
    match orchestration_service.orchestrate_medication_request(request).await {
        Ok(result) => Ok(Json(serde_json::to_value(result).unwrap())),
        Err(e) => {
            error!(error = %e, "Orchestration failed");
            Err(StatusCode::INTERNAL_SERVER_ERROR)
        }
    }
}

pub async fn health_check(
    State(orchestration_service): State<Arc<dyn OrchestrationService>>,
) -> Result<Json<Value>, StatusCode> {
    match orchestration_service.health_check().await {
        Ok(health) => Ok(Json(serde_json::to_value(health).unwrap())),
        Err(e) => {
            error!(error = %e, "Health check failed");
            Err(StatusCode::SERVICE_UNAVAILABLE)
        }
    }
}
```

## Performance Benchmarks

### Expected Performance Improvements

| Metric | Python (Baseline) | Go (Improvement) | Rust (Improvement) |
|--------|-------------------|------------------|-------------------|
| **Request Latency (p50)** | 45ms | 15ms (-67%) | 8ms (-82%) |
| **Request Latency (p99)** | 180ms | 60ms (-67%) | 25ms (-86%) |
| **Memory Usage** | 120MB | 45MB (-62%) | 28MB (-77%) |
| **CPU Usage** | 100% (baseline) | 60% (-40%) | 35% (-65%) |
| **Cold Start Time** | 5.2s | 0.8s (-85%) | 0.3s (-94%) |
| **Throughput (req/s)** | 850 | 2,400 (+182%) | 4,200 (+394%) |

### Load Testing Configuration

**Go Load Test:**
```bash
# Install hey for load testing
go install github.com/rakyll/hey@latest

# Test orchestration endpoint
hey -n 10000 -c 100 -H "Content-Type: application/json" \
    -d '{"patient_id":"test-123","correlation_id":"load-test","medication_request":{},"clinical_intent":{},"provider_context":{}}' \
    http://localhost:8017/api/v1/orchestrate/medication
```

**Rust Load Test:**
```bash
# Install wrk for high-performance load testing
sudo apt install wrk

# Create test payload
cat > payload.json << EOF
{"patient_id":"test-123","correlation_id":"load-test","medication_request":{},"clinical_intent":{},"provider_context":{}}
EOF

# Run load test
wrk -t12 -c400 -d30s -s script.lua http://localhost:8017/api/v1/orchestrate/medication
```

## Production Deployment

### Go Deployment

**Dockerfile:**
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o workflow-engine cmd/server/main.go

# Production stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/workflow-engine .
COPY configs/ configs/

EXPOSE 8017
CMD ["./workflow-engine"]
```

### Rust Deployment

**Dockerfile:**
```dockerfile
# Build stage
FROM rust:1.75-slim AS builder

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release
RUN rm src/main.rs

COPY src/ src/
RUN touch src/main.rs
RUN cargo build --release

# Production stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/target/release/workflow-engine-rust .
COPY configs/ configs/

EXPOSE 8017
CMD ["./workflow-engine-rust"]
```

## Monitoring and Observability

### Metrics Collection

**Go with Prometheus:**
```go
// internal/monitoring/metrics.go
package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    OrchestrationDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "workflow_orchestration_duration_seconds",
            Help: "Duration of workflow orchestration requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"status", "phase"},
    )
    
    OrchestrationCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "workflow_orchestration_requests_total",
            Help: "Total number of orchestration requests",
        },
        []string{"status"},
    )
    
    ActiveWorkflows = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "workflow_active_instances_total",
            Help: "Number of currently active workflow instances",
        },
    )
)
```

**Rust with OpenTelemetry:**
```rust
// src/infrastructure/monitoring/metrics.rs
use opentelemetry::{
    global,
    metrics::{Counter, Histogram, UpDownCounter},
    KeyValue,
};

#[derive(Clone)]
pub struct WorkflowMetrics {
    pub orchestration_duration: Histogram<f64>,
    pub orchestration_counter: Counter<u64>,
    pub active_workflows: UpDownCounter<i64>,
}

impl WorkflowMetrics {
    pub fn new() -> Self {
        let meter = global::meter("workflow-engine");
        
        Self {
            orchestration_duration: meter
                .f64_histogram("workflow_orchestration_duration")
                .with_description("Duration of workflow orchestration requests")
                .init(),
            orchestration_counter: meter
                .u64_counter("workflow_orchestration_requests_total")
                .with_description("Total number of orchestration requests")
                .init(),
            active_workflows: meter
                .i64_up_down_counter("workflow_active_instances_total")
                .with_description("Number of currently active workflow instances")
                .init(),
        }
    }
    
    pub fn record_orchestration(&self, duration: f64, status: &str, phase: &str) {
        self.orchestration_duration.record(duration, &[
            KeyValue::new("status", status.to_string()),
            KeyValue::new("phase", phase.to_string()),
        ]);
        
        self.orchestration_counter.add(1, &[
            KeyValue::new("status", status.to_string()),
        ]);
    }
}
```

This comprehensive implementation guide provides the foundation for successfully converting the Python workflow engine service to either Go or Rust while maintaining clinical safety, regulatory compliance, and enterprise-grade performance characteristics.