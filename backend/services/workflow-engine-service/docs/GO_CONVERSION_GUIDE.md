# Workflow Engine Service - Go Conversion Guide

## Overview

This guide provides detailed documentation for converting the Python-based workflow engine service to Go. The conversion maintains the core architecture while leveraging Go's performance characteristics and type safety for enterprise-grade clinical workflow orchestration.

## Architecture Analysis

### Current Python Structure
```
app/
├── core/              # Configuration and settings
├── db/                # Database models and connections
├── models/            # SQLAlchemy models  
├── orchestration/     # Strategic orchestration layer
├── clinical/          # Clinical workflow implementations
├── gql_schema/        # GraphQL Federation schemas
├── security/          # Authentication middleware
├── monitoring/        # Observability components
└── main.py           # FastAPI application
```

### Proposed Go Structure
```
cmd/
├── server/           # Main application entry point
└── migrate/          # Database migration utility

internal/
├── config/           # Configuration management
├── domain/           # Business domain models
├── repository/       # Data access layer
├── service/          # Business logic services
├── handler/          # HTTP handlers (REST + GraphQL)
├── middleware/       # HTTP middleware stack
├── orchestration/    # Strategic orchestration
├── clinical/         # Clinical workflow engines
├── monitoring/       # Metrics and tracing
├── auth/             # Authentication components
└── database/         # Database connection management

pkg/
├── fhir/             # FHIR client libraries
├── camunda/          # Camunda integration
├── safety/           # Safety Gateway client
└── medication/       # Medication service client

api/
├── proto/            # gRPC protobuf definitions
├── graphql/          # GraphQL schema definitions
└── openapi/          # OpenAPI specifications

migrations/           # Database schema migrations
configs/              # Configuration files
deployments/          # Deployment configurations
```

## Core Components Conversion

### 1. Configuration Management

**Python Implementation:**
```python
# app/core/config.py
class Settings(BaseSettings):
    SERVICE_NAME: str = "workflow-engine-service"
    SERVICE_PORT: int = 8017
    DATABASE_URL: str
    GOOGLE_CLOUD_PROJECT: str
```

**Go Implementation:**
```go
// internal/config/config.go
package config

import (
    "github.com/kelseyhightower/envconfig"
    "time"
)

type Config struct {
    // Service Configuration
    ServiceName string `envconfig:"SERVICE_NAME" default:"workflow-engine-service"`
    ServicePort int    `envconfig:"SERVICE_PORT" default:"8017"`
    Debug       bool   `envconfig:"DEBUG" default:"true"`
    
    // Database Configuration  
    DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
    
    // Google Cloud Healthcare API
    UseGoogleHealthcareAPI bool   `envconfig:"USE_GOOGLE_HEALTHCARE_API" default:"true"`
    GoogleCloudProject     string `envconfig:"GOOGLE_CLOUD_PROJECT" required:"true"`
    GoogleCloudLocation    string `envconfig:"GOOGLE_CLOUD_LOCATION" default:"asia-south1"`
    GoogleCloudDataset     string `envconfig:"GOOGLE_CLOUD_DATASET" default:"clinical-synthesis-hub"`
    
    // External Services
    AuthServiceURL       string `envconfig:"AUTH_SERVICE_URL" default:"http://localhost:8001"`
    MedicationServiceURL string `envconfig:"MEDICATION_SERVICE_URL" default:"http://localhost:8004"`
    SafetyGatewayURL     string `envconfig:"SAFETY_GATEWAY_URL" default:"http://localhost:8018"`
    Flow2GoURL           string `envconfig:"FLOW2_GO_URL" default:"http://localhost:8080"`
    Flow2RustURL         string `envconfig:"FLOW2_RUST_URL" default:"http://localhost:8090"`
    
    // Workflow Engine Configuration
    WorkflowExecutionTimeout time.Duration `envconfig:"WORKFLOW_EXECUTION_TIMEOUT" default:"1h"`
    TaskAssignmentTimeout    time.Duration `envconfig:"TASK_ASSIGNMENT_TIMEOUT" default:"24h"`
    
    // Monitoring
    PrometheusEnabled bool   `envconfig:"PROMETHEUS_ENABLED" default:"true"`
    JaegerEndpoint    string `envconfig:"JAEGER_ENDPOINT" default:"http://localhost:14268/api/traces"`
    
    // Feature Flags
    MockMode        bool `envconfig:"WORKFLOW_MOCK_MODE" default:"false"`
    EnableWebhooks  bool `envconfig:"WORKFLOW_ENABLE_WEBHOOKS" default:"true"`
}

func Load() (*Config, error) {
    var config Config
    err := envconfig.Process("", &config)
    if err != nil {
        return nil, fmt.Errorf("failed to load configuration: %w", err)
    }
    return &config, nil
}
```

### 2. Domain Models

**Python Implementation:**
```python
# app/orchestration/interfaces.py
@dataclass
class SnapshotReference:
    snapshot_id: str
    checksum: str
    created_at: datetime
    expires_at: datetime
    status: SnapshotStatus
```

**Go Implementation:**
```go
// internal/domain/snapshot.go
package domain

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "time"
)

type SnapshotStatus string

const (
    SnapshotStatusCreated   SnapshotStatus = "created"
    SnapshotStatusActive    SnapshotStatus = "active"
    SnapshotStatusExpired   SnapshotStatus = "expired"
    SnapshotStatusArchived  SnapshotStatus = "archived"
    SnapshotStatusCorrupted SnapshotStatus = "corrupted"
)

type WorkflowPhase string

const (
    WorkflowPhaseCalculate WorkflowPhase = "calculate"
    WorkflowPhaseValidate  WorkflowPhase = "validate" 
    WorkflowPhaseCommit    WorkflowPhase = "commit"
    WorkflowPhaseOverride  WorkflowPhase = "override"
)

type SnapshotReference struct {
    SnapshotID     string            `json:"snapshot_id" db:"snapshot_id"`
    Checksum       string            `json:"checksum" db:"checksum"`
    CreatedAt      time.Time         `json:"created_at" db:"created_at"`
    ExpiresAt      time.Time         `json:"expires_at" db:"expires_at"`
    Status         SnapshotStatus    `json:"status" db:"status"`
    PhaseCreated   WorkflowPhase     `json:"phase_created" db:"phase_created"`
    PatientID      string            `json:"patient_id" db:"patient_id"`
    ContextVersion string            `json:"context_version" db:"context_version"`
    Metadata       map[string]any    `json:"metadata" db:"metadata"`
}

func (s *SnapshotReference) IsValid() bool {
    now := time.Now().UTC()
    return s.Status == SnapshotStatusActive && s.ExpiresAt.After(now)
}

func (s *SnapshotReference) ValidateIntegrity(data map[string]any) bool {
    dataBytes, err := json.Marshal(data)
    if err != nil {
        return false
    }
    
    hash := sha256.Sum256(dataBytes)
    calculatedChecksum := fmt.Sprintf("%x", hash)
    
    return calculatedChecksum == s.Checksum
}

type SnapshotChainTracker struct {
    WorkflowID        string             `json:"workflow_id" db:"workflow_id"`
    CalculateSnapshot *SnapshotReference `json:"calculate_snapshot" db:"calculate_snapshot"`
    ValidateSnapshot  *SnapshotReference `json:"validate_snapshot" db:"validate_snapshot"`
    CommitSnapshot    *SnapshotReference `json:"commit_snapshot" db:"commit_snapshot"`
    OverrideSnapshot  *SnapshotReference `json:"override_snapshot" db:"override_snapshot"`
    ChainCreatedAt    time.Time          `json:"chain_created_at" db:"chain_created_at"`
}

func (s *SnapshotChainTracker) AddPhaseSnapshot(phase WorkflowPhase, snapshot *SnapshotReference) {
    switch phase {
    case WorkflowPhaseCalculate:
        s.CalculateSnapshot = snapshot
    case WorkflowPhaseValidate:
        s.ValidateSnapshot = snapshot
    case WorkflowPhaseCommit:
        s.CommitSnapshot = snapshot
    case WorkflowPhaseOverride:
        s.OverrideSnapshot = snapshot
    }
}

func (s *SnapshotChainTracker) ValidateChainConsistency() bool {
    var snapshots []*SnapshotReference
    
    if s.CalculateSnapshot != nil {
        snapshots = append(snapshots, s.CalculateSnapshot)
    }
    if s.ValidateSnapshot != nil {
        snapshots = append(snapshots, s.ValidateSnapshot)
    }
    if s.CommitSnapshot != nil {
        snapshots = append(snapshots, s.CommitSnapshot)
    }
    
    if len(snapshots) < 2 {
        return true // Single or no snapshots are consistent
    }
    
    base := snapshots[0]
    for _, snapshot := range snapshots[1:] {
        if snapshot.PatientID != base.PatientID || 
           snapshot.ContextVersion != base.ContextVersion {
            return false
        }
    }
    
    return true
}
```

### 3. Strategic Orchestrator

**Python Implementation:**
```python
# app/orchestration/strategic_orchestrator.py
class StrategicOrchestrator:
    async def orchestrate_medication_request(
        self, request: CalculateRequest
    ) -> Dict[str, Any]:
        # Calculate > Validate > Commit pattern
```

**Go Implementation:**
```go
// internal/orchestration/strategic_orchestrator.go
package orchestration

import (
    "context"
    "fmt"
    "time"
    
    "github.com/clinical-synthesis-hub/workflow-engine/internal/domain"
    "github.com/clinical-synthesis-hub/workflow-engine/internal/service"
    "github.com/clinical-synthesis-hub/workflow-engine/pkg/safety"
    "github.com/clinical-synthesis-hub/workflow-engine/pkg/medication"
    
    "go.uber.org/zap"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

type OrchestrationResult string

const (
    OrchestrationResultSuccess OrchestrationResult = "success"
    OrchestrationResultWarning OrchestrationResult = "warning"
    OrchestrationResultFailure OrchestrationResult = "failure"
    OrchestrationResultBlocked OrchestrationResult = "blocked"
)

type CalculateRequest struct {
    PatientID         string         `json:"patient_id"`
    MedicationRequest map[string]any `json:"medication_request"`
    ClinicalIntent    map[string]any `json:"clinical_intent"`
    ProviderContext   map[string]any `json:"provider_context"`
    CorrelationID     string         `json:"correlation_id"`
    Urgency           string         `json:"urgency,omitempty"`
}

type CalculateResponse struct {
    ProposalSetID     string                 `json:"proposal_set_id"`
    SnapshotID        string                 `json:"snapshot_id"`
    RankedProposals   []map[string]any       `json:"ranked_proposals"`
    ClinicalEvidence  map[string]any         `json:"clinical_evidence"`
    MonitoringPlan    map[string]any         `json:"monitoring_plan"`
    KBVersions        map[string]string      `json:"kb_versions"`
    ExecutionTimeMs   float64                `json:"execution_time_ms"`
    Result            OrchestrationResult    `json:"result"`
}

type ValidateRequest struct {
    ProposalSetID           string                 `json:"proposal_set_id"`
    SnapshotID              string                 `json:"snapshot_id"`
    SelectedProposals       []map[string]any       `json:"selected_proposals"`
    ValidationRequirements  map[string]any         `json:"validation_requirements"`
    CorrelationID           string                 `json:"correlation_id"`
}

type ValidateResponse struct {
    ValidationID        string                 `json:"validation_id"`
    Verdict             string                 `json:"verdict"` // SAFE, WARNING, UNSAFE
    Findings            []map[string]any       `json:"findings"`
    OverrideTokens      []string               `json:"override_tokens,omitempty"`
    ApprovalRequirements map[string]any        `json:"approval_requirements,omitempty"`
    Result              OrchestrationResult    `json:"result"`
}

type CommitRequest struct {
    ProposalSetID    string         `json:"proposal_set_id"`
    ValidationID     string         `json:"validation_id"`
    SelectedProposal map[string]any `json:"selected_proposal"`
    ProviderDecision map[string]any `json:"provider_decision"`
    CorrelationID    string         `json:"correlation_id"`
}

type CommitResponse struct {
    MedicationOrderID        string              `json:"medication_order_id"`
    PersistenceStatus        string              `json:"persistence_status"`
    EventPublicationStatus   string              `json:"event_publication_status"`
    AuditTrailID             string              `json:"audit_trail_id"`
    Result                   OrchestrationResult `json:"result"`
}

type StrategicOrchestrator struct {
    logger              *zap.Logger
    tracer              trace.Tracer
    flow2GoClient       service.Flow2GoClient
    flow2RustClient     service.Flow2RustClient
    safetyClient        safety.Client
    medicationClient    medication.Client
    snapshotService     service.SnapshotService
    
    // Performance targets in milliseconds
    performanceTargets struct {
        calculateMs int64
        validateMs  int64
        commitMs    int64
        totalMs     int64
    }
}

func NewStrategicOrchestrator(
    logger *zap.Logger,
    flow2GoClient service.Flow2GoClient,
    flow2RustClient service.Flow2RustClient,
    safetyClient safety.Client,
    medicationClient medication.Client,
    snapshotService service.SnapshotService,
) *StrategicOrchestrator {
    tracer := otel.Tracer("strategic-orchestrator")
    
    orchestrator := &StrategicOrchestrator{
        logger:           logger,
        tracer:           tracer,
        flow2GoClient:    flow2GoClient,
        flow2RustClient:  flow2RustClient,
        safetyClient:     safetyClient,
        medicationClient: medicationClient,
        snapshotService:  snapshotService,
    }
    
    // Performance targets from data flow documentation
    orchestrator.performanceTargets.calculateMs = 175
    orchestrator.performanceTargets.validateMs = 100
    orchestrator.performanceTargets.commitMs = 50
    orchestrator.performanceTargets.totalMs = 325
    
    return orchestrator
}

func (o *StrategicOrchestrator) OrchestrateMedicationRequest(
    ctx context.Context,
    request *CalculateRequest,
) (map[string]any, error) {
    ctx, span := o.tracer.Start(ctx, "orchestrate_medication_request")
    defer span.End()
    
    orchestrationStart := time.Now()
    correlationID := request.CorrelationID
    
    o.logger.Info("Starting medication orchestration",
        zap.String("correlation_id", correlationID),
        zap.String("patient_id", request.PatientID))
    
    // Initialize snapshot chain tracker
    snapshotChain := &domain.SnapshotChainTracker{
        WorkflowID:     correlationID,
        ChainCreatedAt: time.Now().UTC(),
    }
    
    // STEP 1: CALCULATE - Generate medication proposals
    o.logger.Info("Starting CALCULATE step", zap.String("correlation_id", correlationID))
    calculateResponse, err := o.executeCalculateStep(ctx, request)
    if err != nil {
        return o.createErrorResponse("CALCULATE_FAILED", err.Error(), correlationID), nil
    }
    
    if calculateResponse.Result != OrchestrationResultSuccess {
        return o.createErrorResponse("CALCULATE_FAILED", 
            fmt.Sprintf("Calculate step failed: %s", calculateResponse.Result), correlationID), nil
    }
    
    // Update snapshot chain
    calculateSnapshot, err := o.snapshotService.GetSnapshot(ctx, calculateResponse.SnapshotID)
    if err != nil {
        return o.createErrorResponse("SNAPSHOT_ERROR", err.Error(), correlationID), nil
    }
    snapshotChain.AddPhaseSnapshot(domain.WorkflowPhaseCalculate, calculateSnapshot)
    
    // STEP 2: VALIDATE - Comprehensive safety validation
    o.logger.Info("Starting VALIDATE step", zap.String("correlation_id", correlationID))
    validateRequest := &ValidateRequest{
        ProposalSetID:     calculateResponse.ProposalSetID,
        SnapshotID:        calculateResponse.SnapshotID,
        SelectedProposals: calculateResponse.RankedProposals[:min(3, len(calculateResponse.RankedProposals))],
        ValidationRequirements: map[string]any{
            "cae_engine":              true,
            "protocol_engine":         true,
            "comprehensive_validation": true,
        },
        CorrelationID: correlationID,
    }
    
    validateResponse, err := o.executeValidateStep(ctx, validateRequest)
    if err != nil {
        return o.createErrorResponse("VALIDATE_FAILED", err.Error(), correlationID), nil
    }
    
    // Update snapshot chain
    validateSnapshot, err := o.snapshotService.GetSnapshot(ctx, calculateResponse.SnapshotID)
    if err != nil {
        return o.createErrorResponse("SNAPSHOT_ERROR", err.Error(), correlationID), nil
    }
    snapshotChain.AddPhaseSnapshot(domain.WorkflowPhaseValidate, validateSnapshot)
    
    // Validate snapshot chain consistency
    if !snapshotChain.ValidateChainConsistency() {
        return o.createErrorResponse("SNAPSHOT_CONSISTENCY_ERROR", 
            "Snapshot consistency validation failed", correlationID), nil
    }
    
    // STEP 3: COMMIT - Conditional based on validation result
    switch validateResponse.Verdict {
    case "SAFE":
        o.logger.Info("Starting COMMIT step", zap.String("correlation_id", correlationID))
        commitRequest := &CommitRequest{
            ProposalSetID:    calculateResponse.ProposalSetID,
            ValidationID:     validateResponse.ValidationID,
            SelectedProposal: calculateResponse.RankedProposals[0],
            ProviderDecision: map[string]any{"auto_selected": true},
            CorrelationID:    correlationID,
        }
        
        commitResponse, err := o.executeCommitStep(ctx, commitRequest)
        if err != nil {
            return o.createErrorResponse("COMMIT_FAILED", err.Error(), correlationID), nil
        }
        
        // Success path
        totalTime := time.Since(orchestrationStart).Milliseconds()
        
        response := map[string]any{
            "status":             "SUCCESS",
            "correlation_id":     correlationID,
            "medication_order_id": commitResponse.MedicationOrderID,
            "calculation": map[string]any{
                "proposal_set_id":    calculateResponse.ProposalSetID,
                "snapshot_id":        calculateResponse.SnapshotID,
                "execution_time_ms":  calculateResponse.ExecutionTimeMs,
            },
            "validation": map[string]any{
                "validation_id": validateResponse.ValidationID,
                "verdict":       validateResponse.Verdict,
            },
            "commitment": map[string]any{
                "order_id":       commitResponse.MedicationOrderID,
                "audit_trail_id": commitResponse.AuditTrailID,
            },
            "performance": map[string]any{
                "total_time_ms": totalTime,
                "meets_target":  totalTime <= o.performanceTargets.totalMs,
            },
        }
        
        return response, nil
        
    case "WARNING":
        // Return to provider with warning and override options
        return map[string]any{
            "status":              "REQUIRES_PROVIDER_DECISION",
            "correlation_id":      correlationID,
            "validation_findings": validateResponse.Findings,
            "override_tokens":     validateResponse.OverrideTokens,
            "proposals":           calculateResponse.RankedProposals,
            "snapshot_id":         calculateResponse.SnapshotID,
        }, nil
        
    default: // UNSAFE
        // Block and suggest alternatives
        alternatives, err := o.generateAlternatives(ctx, calculateResponse.SnapshotID, validateResponse.Findings)
        if err != nil {
            o.logger.Warn("Failed to generate alternatives", zap.Error(err))
            alternatives = []map[string]any{}
        }
        
        return map[string]any{
            "status":                  "BLOCKED_UNSAFE",
            "correlation_id":          correlationID,
            "blocking_findings":       validateResponse.Findings,
            "alternative_approaches":  alternatives,
        }, nil
    }
}

func (o *StrategicOrchestrator) executeCalculateStep(
    ctx context.Context,
    request *CalculateRequest,
) (*CalculateResponse, error) {
    ctx, span := o.tracer.Start(ctx, "execute_calculate_step")
    defer span.End()
    
    calculateStart := time.Now()
    
    // Route to Flow2 Go Engine using Recipe Snapshot Architecture
    flow2Request := service.Flow2ExecuteRequest{
        PatientID:         request.PatientID,
        Medication:        request.MedicationRequest,
        ClinicalIntent:    request.ClinicalIntent,
        ProviderContext:   request.ProviderContext,
        ExecutionMode:     "snapshot_optimized",
        CorrelationID:     request.CorrelationID,
    }
    
    flow2Result, err := o.flow2GoClient.ExecuteAdvanced(ctx, &flow2Request)
    if err != nil {
        o.logger.Error("Calculate step failed", zap.Error(err))
        return &CalculateResponse{
            Result: OrchestrationResultFailure,
        }, err
    }
    
    executionTime := float64(time.Since(calculateStart).Nanoseconds()) / 1e6 // Convert to milliseconds
    
    return &CalculateResponse{
        ProposalSetID:     flow2Result.ProposalSetID,
        SnapshotID:        flow2Result.SnapshotID,
        RankedProposals:   flow2Result.RankedProposals,
        ClinicalEvidence:  flow2Result.ClinicalEvidence,
        MonitoringPlan:    flow2Result.MonitoringPlan,
        KBVersions:        flow2Result.KBVersions,
        ExecutionTimeMs:   executionTime,
        Result:            OrchestrationResultSuccess,
    }, nil
}

func (o *StrategicOrchestrator) executeValidateStep(
    ctx context.Context,
    request *ValidateRequest,
) (*ValidateResponse, error) {
    ctx, span := o.tracer.Start(ctx, "execute_validate_step")
    defer span.End()
    
    o.logger.Info("Executing VALIDATE step via Safety Gateway",
        zap.String("correlation_id", request.CorrelationID))
    
    // Create Safety Gateway validation request
    safetyRequest := &safety.ValidationRequest{
        ProposalSetID:           request.ProposalSetID,
        SnapshotID:              request.SnapshotID,
        Proposals:               request.SelectedProposals,
        PatientContext:          map[string]any{}, // Will be populated from snapshot
        ValidationRequirements:  request.ValidationRequirements,
        CorrelationID:           request.CorrelationID,
    }
    
    // Execute comprehensive validation via Safety Gateway client
    safetyResponse, err := o.safetyClient.ComprehensiveValidation(ctx, safetyRequest)
    if err != nil {
        o.logger.Error("Validate step failed", zap.Error(err))
        return &ValidateResponse{
            ValidationID: "",
            Verdict:      "UNSAFE",
            Findings: []map[string]any{{
                "error":    fmt.Sprintf("Safety Gateway validation failed: %s", err.Error()),
                "severity": "CRITICAL",
                "category": "SYSTEM_ERROR",
            }},
            Result: OrchestrationResultFailure,
        }, err
    }
    
    // Convert Safety Gateway response to orchestrator format
    if safetyResponse.Verdict == safety.VerdictError {
        return &ValidateResponse{
            ValidationID: safetyResponse.ValidationID,
            Verdict:      "UNSAFE",
            Findings:     []map[string]any{{"error": "Safety Gateway validation error"}},
            Result:       OrchestrationResultFailure,
        }, nil
    }
    
    // Convert findings to map format
    findings := make([]map[string]any, len(safetyResponse.Findings))
    for i, finding := range safetyResponse.Findings {
        findings[i] = map[string]any{
            "finding_id":            finding.FindingID,
            "severity":              finding.Severity,
            "category":              finding.Category,
            "description":           finding.Description,
            "clinical_significance": finding.ClinicalSignificance,
            "recommendation":        finding.Recommendation,
            "confidence_score":      finding.ConfidenceScore,
        }
    }
    
    return &ValidateResponse{
        ValidationID:         safetyResponse.ValidationID,
        Verdict:              string(safetyResponse.Verdict),
        Findings:             findings,
        OverrideTokens:       safetyResponse.OverrideTokens,
        ApprovalRequirements: safetyResponse.OverrideRequirements,
        Result:               OrchestrationResultSuccess,
    }, nil
}

func (o *StrategicOrchestrator) executeCommitStep(
    ctx context.Context,
    request *CommitRequest,
) (*CommitResponse, error) {
    ctx, span := o.tracer.Start(ctx, "execute_commit_step")
    defer span.End()
    
    // Route to Medication Service for persistence
    medicationRequest := &medication.CommitRequest{
        ProposalSetID:    request.ProposalSetID,
        ValidationID:     request.ValidationID,
        SelectedProposal: request.SelectedProposal,
        ProviderDecision: request.ProviderDecision,
        CorrelationID:    request.CorrelationID,
    }
    
    medicationResult, err := o.medicationClient.Commit(ctx, medicationRequest)
    if err != nil {
        o.logger.Error("Commit step failed", zap.Error(err))
        return &CommitResponse{
            Result: OrchestrationResultFailure,
        }, err
    }
    
    return &CommitResponse{
        MedicationOrderID:      medicationResult.MedicationOrderID,
        PersistenceStatus:      medicationResult.PersistenceStatus,
        EventPublicationStatus: medicationResult.EventPublicationStatus,
        AuditTrailID:           medicationResult.AuditTrailID,
        Result:                 OrchestrationResultSuccess,
    }, nil
}

func (o *StrategicOrchestrator) generateAlternatives(
    ctx context.Context,
    snapshotID string,
    blockingFindings []map[string]any,
) ([]map[string]any, error) {
    ctx, span := o.tracer.Start(ctx, "generate_alternatives")
    defer span.End()
    
    alternatives, err := o.flow2GoClient.GenerateAlternatives(ctx, &service.AlternativesRequest{
        SnapshotID:       snapshotID,
        BlockingFindings: blockingFindings,
    })
    
    if err != nil {
        o.logger.Warn("Alternative generation failed", zap.Error(err))
        return []map[string]any{}, err
    }
    
    return alternatives.Alternatives, nil
}

func (o *StrategicOrchestrator) createErrorResponse(
    errorCode, errorMessage, correlationID string,
) map[string]any {
    return map[string]any{
        "status":        "ERROR",
        "error_code":    errorCode,
        "error_message": errorMessage,
        "correlation_id": correlationID,
        "timestamp":     time.Now().UTC().Format(time.RFC3339),
    }
}

func (o *StrategicOrchestrator) HealthCheck(ctx context.Context) (map[string]any, error) {
    ctx, span := o.tracer.Start(ctx, "health_check")
    defer span.End()
    
    servicesStatus := map[string]string{}
    
    // Check Flow2 Go Engine
    if err := o.flow2GoClient.HealthCheck(ctx); err != nil {
        servicesStatus["flow2_go"] = "unavailable"
    } else {
        servicesStatus["flow2_go"] = "healthy"
    }
    
    // Check Safety Gateway
    if err := o.safetyClient.HealthCheck(ctx); err != nil {
        servicesStatus["safety_gateway"] = "unavailable"
    } else {
        servicesStatus["safety_gateway"] = "healthy"
    }
    
    // Check Medication Service
    if err := o.medicationClient.HealthCheck(ctx); err != nil {
        servicesStatus["medication_service"] = "unavailable"
    } else {
        servicesStatus["medication_service"] = "healthy"
    }
    
    overallHealthy := true
    for _, status := range servicesStatus {
        if status != "healthy" {
            overallHealthy = false
            break
        }
    }
    
    status := "healthy"
    if !overallHealthy {
        status = "degraded"
    }
    
    return map[string]any{
        "status":                status,
        "services":              servicesStatus,
        "orchestration_pattern": "Calculate > Validate > Commit",
        "performance_targets": map[string]int64{
            "calculate_ms": o.performanceTargets.calculateMs,
            "validate_ms":  o.performanceTargets.validateMs,
            "commit_ms":    o.performanceTargets.commitMs,
            "total_ms":     o.performanceTargets.totalMs,
        },
    }, nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

## Key Go Advantages

**★ Insight ─────────────────────────────────────**
Go provides several architectural benefits for the workflow engine:
1. **Compile-time Safety**: Strong typing prevents runtime errors common in dynamic languages
2. **Concurrency Primitives**: Goroutines and channels enable efficient parallel processing of workflow steps  
3. **Memory Efficiency**: Lower memory footprint critical for clinical systems processing high volumes
**─────────────────────────────────────────────────**

### Performance Benefits

1. **Faster Startup Time**: Go binaries start significantly faster than Python applications
2. **Lower Memory Usage**: Approximately 60-80% less memory usage compared to Python
3. **Better Concurrency**: Native goroutines handle thousands of concurrent workflows efficiently
4. **Predictable Latency**: Go's garbage collector provides more predictable latency characteristics

### Type Safety Benefits

1. **Compile-time Validation**: Catch errors before deployment
2. **Interface-based Design**: Clear contracts between components
3. **Null Safety**: Explicit pointer handling prevents null reference errors
4. **Schema Validation**: Struct tags enable automatic validation

### Infrastructure Benefits

1. **Single Binary Deployment**: No dependency management issues
2. **Cross-platform Compilation**: Build for any target architecture
3. **Container Efficiency**: Smaller Docker images and faster cold starts
4. **Observability**: Built-in support for metrics, tracing, and profiling

## Migration Strategy

### Phase 1: Core Services (Weeks 1-2)
- Configuration management
- Domain models and interfaces
- Database layer with SQLX
- Basic HTTP server setup

### Phase 2: Business Logic (Weeks 3-4)
- Strategic orchestrator conversion
- Snapshot management service
- Client libraries for external services
- Error handling and logging

### Phase 3: API Layer (Weeks 5-6)
- REST API handlers
- GraphQL Federation setup
- Authentication middleware
- Request validation

### Phase 4: Integration (Weeks 7-8)
- External service clients
- Database migrations
- Monitoring and metrics
- End-to-end testing

## Testing Strategy

Go provides excellent testing capabilities:

```go
func TestStrategicOrchestrator_OrchestrateMedicationRequest(t *testing.T) {
    tests := []struct {
        name           string
        request        *CalculateRequest
        mockFlow2      func(*service.MockFlow2GoClient)
        mockSafety     func(*safety.MockClient)
        mockMedication func(*medication.MockClient)
        wantStatus     string
        wantErr        bool
    }{
        {
            name: "successful_orchestration",
            request: &CalculateRequest{
                PatientID:     "patient-123",
                CorrelationID: "corr-456",
            },
            mockFlow2: func(m *service.MockFlow2GoClient) {
                m.EXPECT().ExecuteAdvanced(gomock.Any(), gomock.Any()).
                    Return(&service.Flow2ExecuteResponse{
                        ProposalSetID: "prop-789",
                        SnapshotID:    "snap-101112",
                        RankedProposals: []map[string]any{
                            {"medication": "aspirin", "dose": "81mg"},
                        },
                    }, nil)
            },
            wantStatus: "SUCCESS",
            wantErr:    false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

This Go conversion maintains the strategic orchestration architecture while providing enterprise-grade performance, type safety, and maintainability for clinical workflow management.