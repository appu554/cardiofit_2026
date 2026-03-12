# Medication Microservice Tech Stack Migration Plan
## From Python to Go + Rust Multi-Service Architecture

### 🎯 Executive Summary

This migration plan transforms our medication service from a single Python microservice into a **high-performance multi-service architecture** using Go and Rust, while preserving the existing Python service for zero-downtime migration.

### 🏗️ Target Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    API Gateway (Kong)                           │
│                 SMART on FHIR + Rate Limiting                  │
└─────────────────────────┬───────────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────────┐
│              Medication Service Orchestrator (Go)               │
│           • Request routing & load balancing                    │
│           • Business logic coordination                         │
│           • GraphQL Federation gateway                          │
│           • Circuit breaker & fallback logic                   │
└─────┬─────────────────┬─────────────────┬─────────────────┬─────┘
      │                 │                 │                 │
┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐
│ Python    │    │ Go        │    │ Rust      │    │ Rust      │
│ Legacy    │    │ Business  │    │ Calc      │    │ Safety    │
│ Service   │    │ Logic     │    │ Engine    │    │ Engine    │
│           │    │ Service   │    │           │    │           │
│ • FHIR    │    │ • Recipe  │    │ • Dose    │    │ • Drug    │
│ • GraphQL │    │ • Context │    │ • PK/PD   │    │ • Allergy │
│ • Recipes │    │ • Rules   │    │ • ML      │    │ • Rules   │
└───────────┘    └───────────┘    └───────────┘    └───────────┘
      │                 │                 │                 │
      └─────────────────┼─────────────────┼─────────────────┘
                        │                 │
              ┌─────────▼─────────┐      │
              │ Event Bus (Kafka) │      │
              │ • Commands        │      │
              │ • Events          │      │
              │ • Audit Trail     │      │
              └───────────────────┘      │
                        │                │
              ┌─────────▼─────────┐      │
              │ Data Layer        │      │
              │ • PostgreSQL 16   │      │
              │ • Redis 7         │      │
              │ • DragonflyDB     │      │
              │ • ClickHouse      │      │
              │ • Neo4j           │      │
              └───────────────────┘      │
                        │                │
              ┌─────────▼─────────┐      │
              │ ML Platform       │      │
              │ • PyTorch         │      │
              │ • Kubeflow        │      │
              │ • Feature Store   │      │
              └───────────────────┘      │
                                        │
              ┌─────────────────────────▼┐
              │ Observability Stack      │
              │ • Prometheus + Grafana   │
              │ • Jaeger Tracing        │
              │ • Vector + ClickHouse   │
              └─────────────────────────┘
```

### 🚀 Migration Strategy: Strangler Fig Pattern

**Phase 1: Preserve & Extend** (Months 1-2)
- Keep existing Python service running
- Build Go orchestrator as traffic router
- Implement comprehensive monitoring
- Zero business disruption

**Phase 2: High-Performance Engines** (Months 2-4)
- Build Rust calculation engine
- Build Rust safety validation engine
- Implement Go business logic service
- Gradual traffic migration

**Phase 3: Event-Driven Architecture** (Months 4-6)
- Implement Kafka event streaming
- Add Apache Flink for real-time processing
- Build comprehensive audit trails
- Advanced analytics with ClickHouse

**Phase 4: Production Hardening** (Months 6-8)
- Service mesh with Istio
- Advanced security with OPA
- Multi-region deployment
- Complete Python service retirement

## 📋 Detailed Service Specifications

### 1. Medication Orchestrator Service (Go)

**Technology Stack:**
```go
// Core dependencies
github.com/gin-gonic/gin v1.9.1
github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0
github.com/99designs/gqlgen v0.17.40
github.com/confluentinc/confluent-kafka-go v2.3.0
github.com/redis/go-redis/v9 v9.3.0
github.com/jackc/pgx/v5 v5.5.0
```

**Responsibilities:**
- Request routing and load balancing
- GraphQL Federation coordination
- Circuit breaker implementation
- Business workflow orchestration
- Event publishing to Kafka

**Key Features:**
```go
type MedicationOrchestrator struct {
    pythonClient   *PythonServiceClient
    goClient       *GoBusinessLogicClient
    rustCalcClient *RustCalculationClient
    rustSafetyClient *RustSafetyClient
    eventBus       *kafka.Producer
    circuitBreaker *CircuitBreaker
}

func (o *MedicationOrchestrator) ProcessMedicationRequest(ctx context.Context, req *MedicationRequest) (*MedicationResponse, error) {
    // Route to appropriate service based on request type and load
    switch req.Type {
    case "dose_calculation":
        if o.rustCalcClient.IsHealthy() {
            return o.rustCalcClient.Calculate(ctx, req)
        }
        return o.pythonClient.Calculate(ctx, req) // Fallback
    case "safety_validation":
        return o.rustSafetyClient.Validate(ctx, req)
    default:
        return o.goClient.Process(ctx, req)
    }
}
```

### 2. Go Business Logic Service

**Technology Stack:**
```go
// Business logic dependencies
github.com/gin-gonic/gin v1.9.1
gorm.io/gorm v1.25.5
github.com/shopspring/decimal v1.3.1
github.com/google/uuid v1.4.0
```

**Responsibilities:**
- Clinical recipe execution
- Context aggregation and processing
- FHIR resource management
- Business rule validation
- Workflow state management

**Service Structure:**
```
medication-business-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── entities/
│   │   │   ├── medication.go
│   │   │   ├── prescription.go
│   │   │   └── clinical_context.go
│   │   ├── repositories/
│   │   │   ├── medication_repo.go
│   │   │   └── prescription_repo.go
│   │   └── services/
│   │       ├── recipe_engine.go
│   │       ├── context_service.go
│   │       └── workflow_service.go
│   ├── infrastructure/
│   │   ├── database/
│   │   │   ├── postgres.go
│   │   │   └── migrations/
│   │   ├── cache/
│   │   │   └── redis.go
│   │   └── messaging/
│   │       └── kafka.go
│   ├── interfaces/
│   │   ├── http/
│   │   │   ├── handlers/
│   │   │   └── middleware/
│   │   ├── grpc/
│   │   │   └── server.go
│   │   └── graphql/
│   │       └── resolvers.go
│   └── application/
│       ├── commands/
│       ├── queries/
│       └── services/
├── api/
│   ├── proto/
│   │   └── medication.proto
│   └── graphql/
│       └── schema.graphql
├── configs/
├── scripts/
└── tests/
```

### 3. Rust Calculation Engine

**Technology Stack:**
```toml
[dependencies]
tokio = { version = "1.35", features = ["full"] }
axum = "0.7"
sqlx = { version = "0.7", features = ["postgres", "runtime-tokio-rustls"] }
tonic = "0.12"
serde = { version = "1.0", features = ["derive"] }
redis = { version = "0.24", features = ["tokio-comp"] }
decimal = "2.1"
```

**Responsibilities:**
- Ultra-fast dose calculations (<1ms)
- Pharmacokinetic/pharmacodynamic modeling
- Complex mathematical operations
- ML model inference
- Real-time optimization algorithms

**Performance Targets:**
- **Latency**: <1ms P99 for dose calculations
- **Throughput**: >100,000 calculations/second
- **Memory**: Zero-copy operations where possible
- **Concurrency**: Handle 10,000+ concurrent requests

### 4. Rust Safety Engine

**Technology Stack:**
```toml
[dependencies]
tokio = { version = "1.35", features = ["full"] }
axum = "0.7"
tonic = "0.12"
neo4j = "0.7"
rayon = "1.8"  # Parallel processing
petgraph = "0.6"  # Graph algorithms
```

**Responsibilities:**
- Drug interaction detection
- Allergy validation
- Contraindication checking
- Clinical rule evaluation
- Safety score calculation

**Key Features:**
```rust
#[derive(Debug, Clone)]
pub struct SafetyEngine {
    interaction_graph: Arc<InteractionGraph>,
    allergy_matcher: Arc<AllergyMatcher>,
    rule_engine: Arc<ClinicalRuleEngine>,
}

impl SafetyEngine {
    pub async fn validate_prescription(&self, prescription: &Prescription) -> SafetyResult {
        let tasks = vec![
            self.check_drug_interactions(prescription),
            self.validate_allergies(prescription),
            self.check_contraindications(prescription),
        ];
        
        // Parallel safety validation
        let results = futures::future::join_all(tasks).await;
        SafetyResult::aggregate(results)
    }
}
```

## 🛠️ Implementation Roadmap

### Phase 1: Foundation Setup (Weeks 1-8)

**Week 1-2: Infrastructure Setup**
```bash
# Create new service directories
mkdir -p backend/services/medication-orchestrator-go
mkdir -p backend/services/medication-business-go  
mkdir -p backend/services/medication-calc-rust
mkdir -p backend/services/medication-safety-rust

# Initialize Go modules
cd backend/services/medication-orchestrator-go
go mod init medication-orchestrator
go get github.com/gin-gonic/gin@v1.9.1

cd ../medication-business-go
go mod init medication-business
go get gorm.io/gorm@v1.25.5
```

**Week 3-4: Go Orchestrator Development**
- Implement basic HTTP server with Gin
- Add gRPC client connections to all services
- Implement circuit breaker pattern
- Add comprehensive logging and metrics

**Week 5-6: Go Business Logic Service**
- Port clinical recipe engine from Python
- Implement FHIR resource management
- Add database layer with GORM
- Create GraphQL resolvers

**Week 7-8: Rust Calculation Engine**
- Implement basic dose calculation algorithms
- Add gRPC server with Tonic
- Implement Redis caching layer
- Performance benchmarking and optimization

### Phase 2: Core Services (Weeks 9-16)

**Week 9-10: Rust Safety Engine**
- Implement drug interaction detection
- Add allergy validation logic
- Create clinical rule evaluation engine
- Integration with Neo4j graph database

**Week 11-12: Service Integration**
- Connect all services via gRPC
- Implement service discovery
- Add health checks and monitoring
- Create comprehensive test suites

**Week 13-14: Event Streaming**
- Implement Kafka producers/consumers
- Add event sourcing for audit trails
- Create event replay capabilities
- Implement exactly-once semantics

**Week 15-16: Performance Optimization**
- Load testing and benchmarking
- Memory optimization
- Connection pooling
- Caching strategy refinement

### Phase 3: Advanced Features (Weeks 17-24)

**Week 17-18: Analytics Integration**
- ClickHouse setup for time-series data
- Real-time analytics dashboards
- ML model integration
- Predictive analytics

**Week 19-20: Service Mesh**
- Istio deployment
- mTLS configuration
- Traffic management
- Security policies

**Week 21-22: Monitoring & Observability**
- Prometheus metrics
- Jaeger distributed tracing
- Grafana dashboards
- Alert management

**Week 23-24: Production Hardening**
- Security scanning
- Performance testing
- Disaster recovery
- Documentation

## 🔄 Migration Strategy

### Traffic Routing Strategy

```go
// Gradual traffic migration
type TrafficRouter struct {
    pythonWeight int  // Start: 100%
    goWeight     int  // Start: 0%
    rustWeight   int  // Start: 0%
}

func (r *TrafficRouter) RouteRequest(req *Request) ServiceClient {
    switch {
    case req.IsCalculationHeavy() && r.rustWeight > 0:
        return r.rustCalcClient
    case req.IsBusinessLogic() && r.goWeight > 0:
        return r.goBusinessClient
    default:
        return r.pythonLegacyClient
    }
}
```

### Rollback Strategy

```yaml
# Kubernetes deployment with instant rollback
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: medication-orchestrator
spec:
  strategy:
    canary:
      steps:
      - setWeight: 10
      - pause: {duration: 10m}
      - setWeight: 50
      - pause: {duration: 10m}
      - setWeight: 100
      analysis:
        templates:
        - templateName: error-rate
        args:
        - name: service-name
          value: medication-orchestrator
```

## 📊 Performance Expectations

### Current Python Service vs New Architecture

| Metric | Python Current | Go Service | Rust Engine | Improvement |
|--------|---------------|------------|-------------|-------------|
| **Latency P99** | 500ms | 50ms | 1ms | 500x |
| **Throughput** | 1K req/s | 50K req/s | 100K req/s | 100x |
| **Memory Usage** | 512MB | 128MB | 32MB | 16x |
| **CPU Usage** | 80% | 20% | 5% | 16x |
| **Startup Time** | 30s | 5s | 1s | 30x |

### Cost Analysis

**Infrastructure Costs (Monthly)**
- Current Python: $5,000
- New Multi-Service: $3,000 (40% reduction)
- Performance gains: 100x throughput
- **ROI**: 300% improvement in cost/performance

## 🔒 Risk Mitigation

### Technical Risks
1. **Service Complexity**: Mitigated by comprehensive monitoring
2. **Network Latency**: Mitigated by service mesh optimization
3. **Data Consistency**: Mitigated by event sourcing and SAGA pattern
4. **Learning Curve**: Mitigated by gradual team training

### Business Risks
1. **Downtime**: Zero-downtime migration with traffic routing
2. **Data Loss**: Comprehensive backup and event replay
3. **Performance Regression**: Extensive load testing
4. **Compliance**: Maintain all existing audit trails

## 🎯 Success Metrics

### Technical KPIs
- **Latency**: <10ms P99 for all operations
- **Throughput**: >50K requests/second
- **Availability**: 99.99% uptime
- **Error Rate**: <0.01%

### Business KPIs
- **Cost Reduction**: 40% infrastructure savings
- **Development Velocity**: 50% faster feature delivery
- **Clinical Safety**: Zero safety incidents
- **Compliance**: 100% audit trail coverage

## 🚀 Getting Started

### Immediate Next Steps

1. **Create Go Orchestrator** - Start with basic traffic routing
2. **Set up Monitoring** - Establish baseline metrics
3. **Build Rust PoC** - Validate performance assumptions
4. **Team Training** - Begin Go and Rust education
5. **Infrastructure Setup** - Deploy Kafka, Redis, PostgreSQL

## 💻 Implementation Guides

### Go Orchestrator Service Implementation

**Directory Structure:**
```
medication-orchestrator-go/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handlers/
│   │   ├── medication.go
│   │   ├── health.go
│   │   └── metrics.go
│   ├── clients/
│   │   ├── python_client.go
│   │   ├── go_client.go
│   │   ├── rust_calc_client.go
│   │   └── rust_safety_client.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── logging.go
│   │   └── circuit_breaker.go
│   └── models/
│       └── medication.go
├── api/
│   └── proto/
│       └── medication.proto
├── docker/
│   └── Dockerfile
├── k8s/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── configmap.yaml
└── go.mod
```

**Main Server Implementation:**
```go
// cmd/server/main.go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "medication-orchestrator/internal/config"
    "medication-orchestrator/internal/handlers"
    "medication-orchestrator/internal/middleware"
)

func main() {
    cfg := config.Load()

    // Initialize clients
    clients := initializeClients(cfg)

    // Setup Gin router
    r := gin.New()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    r.Use(middleware.CORS())
    r.Use(middleware.Auth())
    r.Use(middleware.CircuitBreaker())

    // Health check
    r.GET("/health", handlers.HealthCheck)
    r.GET("/metrics", handlers.Metrics)

    // Medication routes
    medicationHandler := handlers.NewMedicationHandler(clients)
    v1 := r.Group("/api/v1")
    {
        v1.POST("/medications/calculate-dose", medicationHandler.CalculateDose)
        v1.POST("/medications/validate-safety", medicationHandler.ValidateSafety)
        v1.POST("/medications/process-recipe", medicationHandler.ProcessRecipe)
        v1.GET("/medications/patient/:id", medicationHandler.GetPatientMedications)
    }

    // GraphQL endpoint
    r.POST("/graphql", handlers.GraphQLHandler)

    // Start server
    srv := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: r,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed to start: %v", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }
}
```

### Rust Calculation Engine Implementation

**Cargo.toml:**
```toml
[package]
name = "medication-calc-engine"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1.35", features = ["full"] }
axum = "0.7"
tower = "0.4"
tower-http = { version = "0.5", features = ["cors", "trace"] }
sqlx = { version = "0.7", features = ["postgres", "runtime-tokio-rustls", "uuid", "chrono"] }
tonic = "0.12"
prost = "0.12"
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
redis = { version = "0.24", features = ["tokio-comp"] }
decimal = "2.1"
uuid = { version = "1.6", features = ["v4", "serde"] }
chrono = { version = "0.4", features = ["serde"] }
tracing = "0.1"
tracing-subscriber = "0.3"
anyhow = "1.0"
thiserror = "1.0"

[build-dependencies]
tonic-build = "0.12"
```

**Main Service Implementation:**
```rust
// src/main.rs
use axum::{
    extract::State,
    http::StatusCode,
    response::Json,
    routing::{get, post},
    Router,
};
use std::sync::Arc;
use tokio::net::TcpListener;
use tower_http::{cors::CorsLayer, trace::TraceLayer};
use tracing::{info, instrument};

mod calculation;
mod cache;
mod database;
mod models;
mod error;

use calculation::DoseCalculationService;
use cache::CacheService;
use database::DatabaseService;
use models::{DoseRequest, DoseResponse, HealthResponse};

#[derive(Clone)]
pub struct AppState {
    calculation_service: Arc<DoseCalculationService>,
    cache_service: Arc<CacheService>,
    database_service: Arc<DatabaseService>,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::init();

    // Initialize services
    let database_service = Arc::new(DatabaseService::new().await?);
    let cache_service = Arc::new(CacheService::new().await?);
    let calculation_service = Arc::new(DoseCalculationService::new(
        database_service.clone(),
        cache_service.clone(),
    ));

    let state = AppState {
        calculation_service,
        cache_service,
        database_service,
    };

    // Build router
    let app = Router::new()
        .route("/health", get(health_check))
        .route("/calculate-dose", post(calculate_dose))
        .route("/calculate-pk", post(calculate_pharmacokinetics))
        .route("/optimize-dose", post(optimize_dose))
        .layer(CorsLayer::permissive())
        .layer(TraceLayer::new_for_http())
        .with_state(state);

    // Start server
    let listener = TcpListener::bind("0.0.0.0:8080").await?;
    info!("Medication Calculation Engine listening on {}", listener.local_addr()?);

    axum::serve(listener, app).await?;

    Ok(())
}

#[instrument]
async fn health_check() -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "healthy".to_string(),
        service: "medication-calc-engine".to_string(),
        version: env!("CARGO_PKG_VERSION").to_string(),
    })
}

#[instrument(skip(state))]
async fn calculate_dose(
    State(state): State<AppState>,
    Json(request): Json<DoseRequest>,
) -> Result<Json<DoseResponse>, StatusCode> {
    match state.calculation_service.calculate_dose(request).await {
        Ok(response) => Ok(Json(response)),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}
```

**High-Performance Dose Calculation:**
```rust
// src/calculation/mod.rs
use crate::models::{DoseRequest, DoseResponse, CalculationType};
use crate::cache::CacheService;
use crate::database::DatabaseService;
use decimal::Decimal;
use std::sync::Arc;
use tokio::time::Instant;
use tracing::{info, warn, instrument};

pub struct DoseCalculationService {
    database: Arc<DatabaseService>,
    cache: Arc<CacheService>,
}

impl DoseCalculationService {
    pub fn new(database: Arc<DatabaseService>, cache: Arc<CacheService>) -> Self {
        Self { database, cache }
    }

    #[instrument(skip(self))]
    pub async fn calculate_dose(&self, request: DoseRequest) -> anyhow::Result<DoseResponse> {
        let start = Instant::now();

        // Check cache first
        let cache_key = format!("dose:{}:{}:{}",
            request.patient_id,
            request.medication_code,
            request.calculation_type
        );

        if let Ok(Some(cached)) = self.cache.get::<DoseResponse>(&cache_key).await {
            info!("Cache hit for dose calculation");
            return Ok(cached);
        }

        // Get patient context
        let patient_context = self.database
            .get_patient_context(&request.patient_id)
            .await?;

        // Perform calculation based on type
        let dose = match request.calculation_type {
            CalculationType::WeightBased => {
                self.calculate_weight_based_dose(&request, &patient_context).await?
            },
            CalculationType::BsaBased => {
                self.calculate_bsa_based_dose(&request, &patient_context).await?
            },
            CalculationType::RenalAdjusted => {
                self.calculate_renal_adjusted_dose(&request, &patient_context).await?
            },
            CalculationType::PharmacokineticsGuided => {
                self.calculate_pk_guided_dose(&request, &patient_context).await?
            },
        };

        let response = DoseResponse {
            patient_id: request.patient_id,
            medication_code: request.medication_code,
            calculated_dose: dose,
            calculation_method: request.calculation_type,
            calculation_time_ms: start.elapsed().as_millis() as u64,
            confidence_score: self.calculate_confidence_score(&request, &dose).await?,
            clinical_notes: self.generate_clinical_notes(&request, &dose).await?,
        };

        // Cache the result
        let _ = self.cache.set(&cache_key, &response, 3600).await;

        let duration = start.elapsed();
        if duration.as_millis() > 10 {
            warn!("Slow dose calculation: {:?}", duration);
        }

        info!("Dose calculation completed in {:?}", duration);
        Ok(response)
    }

    async fn calculate_weight_based_dose(
        &self,
        request: &DoseRequest,
        patient_context: &PatientContext,
    ) -> anyhow::Result<Decimal> {
        let weight = patient_context.weight_kg
            .ok_or_else(|| anyhow::anyhow!("Patient weight not available"))?;

        let dose_per_kg = request.dosing_parameters
            .get("dose_per_kg")
            .and_then(|v| v.parse::<Decimal>().ok())
            .ok_or_else(|| anyhow::anyhow!("Dose per kg not specified"))?;

        let base_dose = weight * dose_per_kg;

        // Apply adjustments
        let adjusted_dose = self.apply_clinical_adjustments(
            base_dose,
            patient_context,
            &request.medication_code,
        ).await?;

        Ok(adjusted_dose)
    }

    async fn apply_clinical_adjustments(
        &self,
        base_dose: Decimal,
        patient_context: &PatientContext,
        medication_code: &str,
    ) -> anyhow::Result<Decimal> {
        let mut adjusted_dose = base_dose;

        // Renal adjustment
        if let Some(creatinine_clearance) = patient_context.creatinine_clearance {
            if creatinine_clearance < Decimal::from(60) {
                let adjustment_factor = self.database
                    .get_renal_adjustment_factor(medication_code, creatinine_clearance)
                    .await?;
                adjusted_dose *= adjustment_factor;
            }
        }

        // Hepatic adjustment
        if let Some(liver_function) = &patient_context.liver_function {
            if liver_function != "normal" {
                let adjustment_factor = self.database
                    .get_hepatic_adjustment_factor(medication_code, liver_function)
                    .await?;
                adjusted_dose *= adjustment_factor;
            }
        }

        // Age-based adjustment
        if let Some(age) = patient_context.age_years {
            if age >= 65 {
                adjusted_dose *= Decimal::from_str("0.8")?; // 20% reduction for elderly
            }
        }

        Ok(adjusted_dose)
    }
}
```

## 🚀 Deployment Configurations

### Kubernetes Deployment Manifests

**Go Orchestrator Deployment:**
```yaml
# k8s/medication-orchestrator-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: medication-orchestrator
  labels:
    app: medication-orchestrator
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: medication-orchestrator
  template:
    metadata:
      labels:
        app: medication-orchestrator
        version: v1
    spec:
      containers:
      - name: orchestrator
        image: clinical-platform/medication-orchestrator:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: url
        - name: KAFKA_BROKERS
          value: "kafka:9092"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: medication-orchestrator
spec:
  selector:
    app: medication-orchestrator
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

**Rust Calculation Engine Deployment:**
```yaml
# k8s/medication-calc-rust-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: medication-calc-rust
  labels:
    app: medication-calc-rust
    version: v1
spec:
  replicas: 5
  selector:
    matchLabels:
      app: medication-calc-rust
  template:
    metadata:
      labels:
        app: medication-calc-rust
        version: v1
    spec:
      containers:
      - name: calc-engine
        image: clinical-platform/medication-calc-rust:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis-secret
              key: url
        - name: RUST_LOG
          value: "info"
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 2
          periodSeconds: 3
---
apiVersion: v1
kind: Service
metadata:
  name: medication-calc-rust
spec:
  selector:
    app: medication-calc-rust
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

### Docker Configurations

**Go Orchestrator Dockerfile:**
```dockerfile
# docker/Dockerfile.orchestrator
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs

EXPOSE 8080
CMD ["./main"]
```

**Rust Calculation Engine Dockerfile:**
```dockerfile
# docker/Dockerfile.calc-rust
FROM rust:1.75 AS builder

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release
RUN rm src/main.rs

COPY src ./src
RUN touch src/main.rs
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/target/release/medication-calc-engine .

EXPOSE 8080
CMD ["./medication-calc-engine"]
```

### Monitoring and Observability

**Prometheus Configuration:**
```yaml
# monitoring/prometheus-config.yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "medication_rules.yml"

scrape_configs:
  - job_name: 'medication-orchestrator'
    static_configs:
      - targets: ['medication-orchestrator:8080']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'medication-calc-rust'
    static_configs:
      - targets: ['medication-calc-rust:8080']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'medication-safety-rust'
    static_configs:
      - targets: ['medication-safety-rust:8080']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'medication-business-go'
    static_configs:
      - targets: ['medication-business-go:8080']
    metrics_path: /metrics
    scrape_interval: 5s
```

**Grafana Dashboard Configuration:**
```json
{
  "dashboard": {
    "title": "Medication Service Performance",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{job=~\"medication.*\"}[5m])",
            "legendFormat": "{{job}} - {{method}}"
          }
        ]
      },
      {
        "title": "Response Time P99",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{job=~\"medication.*\"}[5m]))",
            "legendFormat": "{{job}} P99"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{job=~\"medication.*\",status=~\"5..\"}[5m])",
            "legendFormat": "{{job}} Errors"
          }
        ]
      },
      {
        "title": "Dose Calculation Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(dose_calculation_duration_seconds_bucket[5m]))",
            "legendFormat": "Dose Calc P95"
          }
        ]
      }
    ]
  }
}
```

## 🔧 Development Workflow

### Local Development Setup

**Docker Compose for Local Development:**
```yaml
# docker-compose.dev.yml
version: '3.8'
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: medication_db
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  dragonfly:
    image: docker.dragonflydb.io/dragonflydb/dragonfly
    ports:
      - "6380:6379"

  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    ports:
      - "9092:9092"
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000

  clickhouse:
    image: clickhouse/clickhouse-server:latest
    ports:
      - "8123:8123"
      - "9000:9000"

  neo4j:
    image: neo4j:5
    environment:
      NEO4J_AUTH: neo4j/password
    ports:
      - "7474:7474"
      - "7687:7687"

volumes:
  postgres_data:
```

### Testing Strategy

**Integration Test Example:**
```go
// tests/integration_test.go
package tests

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestDoseCalculationFlow(t *testing.T) {
    // Setup test server
    server := setupTestServer()
    defer server.Close()

    // Test data
    request := DoseCalculationRequest{
        PatientID:        "test-patient-123",
        MedicationCode:   "acetaminophen",
        CalculationType:  "weight_based",
        DosingParameters: map[string]string{
            "dose_per_kg": "10",
        },
    }

    // Make request
    body, _ := json.Marshal(request)
    resp, err := http.Post(server.URL+"/api/v1/medications/calculate-dose",
        "application/json", bytes.NewBuffer(body))

    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Verify response
    var response DoseCalculationResponse
    json.NewDecoder(resp.Body).Decode(&response)

    assert.Equal(t, request.PatientID, response.PatientID)
    assert.Greater(t, response.CalculatedDose, 0.0)
    assert.Less(t, response.CalculationTimeMs, int64(100)) // Sub-100ms target
}

func TestServiceFailover(t *testing.T) {
    // Test that orchestrator falls back to Python service
    // when Rust service is unavailable

    // Simulate Rust service failure
    rustService.Shutdown()

    // Make request - should fallback to Python
    resp := makeCalculationRequest(t)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Verify fallback was used
    assert.Contains(t, resp.Header.Get("X-Service-Used"), "python")
}
```

**Performance Benchmark:**
```rust
// benches/dose_calculation_bench.rs
use criterion::{black_box, criterion_group, criterion_main, Criterion};
use medication_calc_engine::calculation::DoseCalculationService;

fn bench_dose_calculation(c: &mut Criterion) {
    let rt = tokio::runtime::Runtime::new().unwrap();
    let service = rt.block_on(async {
        DoseCalculationService::new_for_testing().await
    });

    c.bench_function("weight_based_calculation", |b| {
        b.to_async(&rt).iter(|| async {
            let request = create_test_request();
            black_box(service.calculate_dose(request).await.unwrap())
        })
    });
}

criterion_group!(benches, bench_dose_calculation);
criterion_main!(benches);
```

## 📋 Migration Checklist

### Pre-Migration Checklist
- [ ] Team training on Go and Rust completed
- [ ] Development environment setup
- [ ] CI/CD pipelines configured
- [ ] Monitoring and alerting setup
- [ ] Database migration scripts prepared
- [ ] Comprehensive test suite created
- [ ] Performance benchmarks established
- [ ] Rollback procedures documented

### Migration Execution Checklist
- [ ] Deploy Go orchestrator in shadow mode
- [ ] Deploy Rust calculation engine
- [ ] Deploy Rust safety engine
- [ ] Configure traffic routing (0% new services)
- [ ] Run parallel testing for 1 week
- [ ] Gradually increase traffic to new services
- [ ] Monitor performance and error rates
- [ ] Complete traffic migration
- [ ] Retire Python service (keep as backup)

### Post-Migration Checklist
- [ ] Performance targets achieved
- [ ] All tests passing
- [ ] Monitoring dashboards updated
- [ ] Documentation updated
- [ ] Team training on new architecture
- [ ] Incident response procedures updated
- [ ] Backup and recovery tested

Would you like me to start implementing any specific component of this migration plan?
