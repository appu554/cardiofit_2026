# Rust Clinical Engine Integration

This document describes the integration of the high-performance Rust Clinical Engine with the Go-based Medication Service V2, providing microsecond-level clinical calculations and intelligence operations.

## Overview

The Rust Clinical Engine integration provides:
- **Sub-millisecond response times** for critical clinical calculations
- **Drug interaction analysis** with comprehensive safety checks
- **Dosage calculations** with patient-specific parameters
- **Safety validation** including contraindications and allergies
- **Clinical rule evaluation** with evidence-based recommendations
- **Performance monitoring** with circuit breaker patterns
- **Error handling** and fallback mechanisms

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Medication Service V2 (Go)              │
├─────────────────────────────────────────────────────────────┤
│ Phase 3: Clinical Intelligence & Rule Evaluation          │
│                                                            │
│ ┌─────────────────┐    ┌──────────────────────────────┐   │
│ │ Clinical        │    │ Clinical Calculation Service │   │
│ │ Intelligence    │────│ • Drug Interactions          │   │
│ │ Service         │    │ • Dosage Calculations        │   │
│ │                 │    │ • Safety Validation          │   │
│ │                 │    │ • Rule Evaluation            │   │
│ └─────────────────┘    └──────────────────────────────┘   │
│                                     │                      │
│ ┌─────────────────────────────────────────────────────┐    │
│ │ Rust Clinical Engine Client                         │    │
│ │ • HTTP/gRPC Communication                          │    │
│ │ • Circuit Breaker Pattern                          │    │
│ │ • Performance Monitoring                           │    │
│ │ • Error Handling & Retries                         │    │
│ └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                                │
                                │ HTTP/gRPC
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                 Rust Clinical Engine                        │
├─────────────────────────────────────────────────────────────┤
│ High-Performance Clinical Computing                        │
│                                                            │
│ • Drug Interaction Detection (microsecond analysis)       │
│ • Dosage Calculation Algorithms                           │
│ • Safety Validation Rules                                 │
│ • Clinical Rule Evaluation Engine                         │
│ • Evidence-Based Recommendations                          │
│                                                            │
│ Port: 8090 (default)                                      │
└─────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. Rust Clinical Engine Client
**File**: `internal/infrastructure/clients/rust_clinical_engine_client.go`

High-performance HTTP client for communicating with the Rust engine:

```go
// Example: Drug Interaction Analysis
request := &clients.DrugInteractionRequest{
    RequestID:       "req-123",
    PatientID:       "patient-456",
    Medications:     medicationList,
    ClinicalContext: patientContext,
    AnalysisDepth:   "comprehensive",
    Priority:        "high",
}

response, err := rustClient.AnalyzeDrugInteractions(ctx, request)
```

**Features:**
- Automatic retries with exponential backoff
- Circuit breaker pattern for resilience
- Performance monitoring and metrics
- Request/response validation
- Comprehensive error handling

### 2. Clinical Calculation Service
**File**: `internal/application/services/clinical_calculation_service.go`

Orchestrates high-performance clinical calculations:

```go
// Example: Phase 3 Clinical Intelligence Processing
phase3Request := &Phase3ClinicalIntelligenceRequest{
    WorkflowID:          workflowID,
    PatientID:           patientID,
    SnapshotData:        snapshotData,
    RequestedOperations: operations,
    Priority:            "high",
}

response, err := clinicalCalcService.ProcessPhase3Intelligence(ctx, phase3Request)
```

**Features:**
- Parallel operation execution
- Performance target enforcement
- Result caching and optimization
- Quality score calculation
- Comprehensive metrics collection

### 3. Performance Monitoring Service
**File**: `internal/application/services/rust_engine_monitoring_service.go`

Monitors Rust engine performance and health:

```go
// Example: Recording Operation Metrics
monitoringService.RecordOperation(
    "drug_interactions", 
    duration, 
    success, 
    performanceTarget,
)
```

**Features:**
- Real-time performance tracking
- Health check caching
- Circuit breaker management
- Performance alert generation
- Comprehensive metrics collection

### 4. Configuration Management
**File**: `config/rust_engine_config.go`

Flexible configuration with environment-specific overrides:

```go
// Load configuration
config, err := config.LoadRustEngineConfiguration()

// Environment-specific optimizations
targets := config.GetOptimalPerformanceTargets("production")
```

## Performance Targets

| Operation Type | Development | Staging | Production |
|---------------|-------------|---------|------------|
| Drug Interactions | 100ms | 75ms | 25ms |
| Dosage Calculation | 75ms | 50ms | 15ms |
| Safety Validation | 150ms | 100ms | 40ms |
| Rule Evaluation | 100ms | 75ms | 20ms |

## Usage Examples

### 1. Basic Drug Interaction Analysis

```go
package main

import (
    "context"
    "time"
    
    "medication-service-v2/internal/infrastructure/clients"
)

func analyzeDrugInteractions(ctx context.Context) error {
    // Initialize Rust engine client
    config := &clients.RustClinicalEngineConfig{
        BaseURL:    "http://localhost:8090",
        Timeout:    5 * time.Second,
        MaxRetries: 3,
    }
    
    rustClient := clients.NewRustClinicalEngineClient(config, logger)
    
    // Prepare medication list
    medications := []clients.MedicationForAnalysis{
        {
            MedicationCode: "warfarin",
            Name:          "Warfarin",
            Dose:          "5mg",
            Route:         "oral",
            Frequency:     "daily",
            IsActive:      true,
        },
        {
            MedicationCode: "aspirin",
            Name:          "Aspirin",
            Dose:          "81mg",
            Route:         "oral",
            Frequency:     "daily",
            IsActive:      true,
        },
    }
    
    // Analyze drug interactions
    request := &clients.DrugInteractionRequest{
        RequestID:       "interaction-001",
        PatientID:       "patient-123",
        Medications:     medications,
        ClinicalContext: patientContext,
        AnalysisDepth:   "comprehensive",
        Priority:        "high",
    }
    
    response, err := rustClient.AnalyzeDrugInteractions(ctx, request)
    if err != nil {
        return fmt.Errorf("drug interaction analysis failed: %w", err)
    }
    
    // Process results
    for _, interaction := range response.Interactions {
        log.Printf("Interaction found: %s - %s (Severity: %s)", 
            interaction.Medication1Name, 
            interaction.Medication2Name, 
            interaction.Severity)
    }
    
    return nil
}
```

### 2. Dosage Calculation with Patient Parameters

```go
func calculateDosage(ctx context.Context) error {
    // Patient-specific parameters
    request := &clients.DosageCalculationRequest{
        RequestID:      "dosage-001",
        PatientID:      "patient-123",
        MedicationCode: "metformin",
        PatientWeight:  75.0,
        PatientAge:     45,
        KidneyFunction: &clients.KidneyFunctionData{
            CreatinineLevel:     1.0,
            CreatinineClearance: 90.0,
            GFR:                90.0,
            Stage:              "normal",
        },
        ClinicalContext: patientContext,
        CalculationType: "standard",
    }
    
    response, err := rustClient.CalculateDosage(ctx, request)
    if err != nil {
        return fmt.Errorf("dosage calculation failed: %w", err)
    }
    
    // Process dosage recommendation
    if response.RecommendedDosage != nil {
        log.Printf("Recommended dosage: %.1f %s %s", 
            response.RecommendedDosage.Amount,
            response.RecommendedDosage.Unit,
            response.RecommendedDosage.Frequency)
    }
    
    return nil
}
```

### 3. Comprehensive Safety Validation

```go
func validateSafety(ctx context.Context) error {
    // Proposed medication
    proposedMedication := &clients.MedicationForAnalysis{
        MedicationCode: "simvastatin",
        Name:          "Simvastatin",
        Dose:          "40mg",
        Route:         "oral",
        Frequency:     "daily",
        IsActive:      true,
    }
    
    // Patient allergies
    allergies := []clients.AllergyInfo{
        {
            AllergenCode: "penicillin",
            AllergenName: "Penicillin",
            Severity:     "severe",
            Reaction:     "anaphylaxis",
        },
    }
    
    request := &clients.SafetyValidationRequest{
        RequestID:          "safety-001",
        PatientID:          "patient-123",
        ProposedMedication: proposedMedication,
        CurrentMedications: currentMedicationList,
        Allergies:          allergies,
        Conditions:         patientConditions,
        ClinicalContext:    patientContext,
        ValidationLevel:    "comprehensive",
    }
    
    response, err := rustClient.ValidateSafety(ctx, request)
    if err != nil {
        return fmt.Errorf("safety validation failed: %w", err)
    }
    
    // Process safety results
    log.Printf("Safety validation result: %s (Risk Score: %.2f)", 
        response.ValidationResult, 
        response.OverallRiskScore)
    
    for _, alert := range response.SafetyAlerts {
        log.Printf("Safety Alert: %s - %s", alert.Severity, alert.Description)
    }
    
    return nil
}
```

### 4. Clinical Rule Evaluation

```go
func evaluateRules(ctx context.Context) error {
    request := &clients.ClinicalRuleEvaluationRequest{
        RequestID:         "rules-001",
        PatientID:         "patient-123",
        RuleSet:           "drug_rules",
        EvaluationContext: patientClinicalContext,
        RuleFilters:       []string{"safety", "efficacy", "interactions"},
        Priority:          "high",
    }
    
    response, err := rustClient.EvaluateRules(ctx, request)
    if err != nil {
        return fmt.Errorf("rule evaluation failed: %w", err)
    }
    
    // Process evaluated rules
    log.Printf("Rules evaluated: %d, Overall score: %.2f", 
        len(response.EvaluatedRules), 
        response.OverallScore)
    
    for _, rule := range response.EvaluatedRules {
        if rule.Triggered {
            log.Printf("Rule triggered: %s (Score: %.2f)", 
                rule.RuleName, 
                rule.Score)
        }
    }
    
    return nil
}
```

## Configuration

### Environment Variables

```bash
# Basic connection settings
export RUST_ENGINE_BASE_URL="http://localhost:8090"
export RUST_ENGINE_TIMEOUT="5s"
export RUST_ENGINE_MAX_RETRIES=3

# Performance settings
export RUST_ENGINE_MAX_CONCURRENT_REQUESTS=100
export RUST_ENGINE_ENABLE_CACHING=true

# Performance targets
export RUST_ENGINE_PERFORMANCE_DRUG_INTERACTION_ANALYSIS="50ms"
export RUST_ENGINE_PERFORMANCE_DOSAGE_CALCULATION="30ms"
export RUST_ENGINE_PERFORMANCE_SAFETY_VALIDATION="75ms"
export RUST_ENGINE_PERFORMANCE_RULE_EVALUATION="40ms"

# Circuit breaker settings
export RUST_ENGINE_CIRCUIT_BREAKER_ENABLED=true
export RUST_ENGINE_CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
export RUST_ENGINE_CIRCUIT_BREAKER_TIMEOUT="30s"
```

### Configuration File (rust_engine_config.yaml)

```yaml
# See config/rust_engine_config.yaml for complete configuration
base_url: "http://localhost:8090"
timeout: "5s"
max_retries: 3

performance_targets:
  drug_interaction_analysis: "50ms"
  dosage_calculation: "30ms"
  safety_validation: "75ms"
  rule_evaluation: "40ms"

circuit_breaker:
  enabled: true
  failure_threshold: 5
  timeout: "30s"
```

## Integration with 4-Phase Workflow

The Rust Clinical Engine is integrated into **Phase 3: Clinical Intelligence & Rule Evaluation** of the 4-phase workflow:

```go
// Phase 3 integration example
func ProcessWorkflowPhase3(ctx context.Context, workflowID uuid.UUID) error {
    // Initialize services
    clinicalIntelligenceService := services.NewClinicalIntelligenceService(
        clinicalCalculationService,
        rustEngineClient,
        knowledgeBaseClients,
        auditService,
        metricsService,
        cacheService,
        config,
        logger,
    )
    
    // Process using high-performance Rust engine
    request := &services.ClinicalIntelligenceRequest{
        WorkflowID:    workflowID,
        PatientID:     patientID,
        SnapshotData:  snapshotData,
        ClinicalParams: clinicalParams,
        RequestedBy:   userID,
        RequestedAt:   time.Now(),
    }
    
    result, err := clinicalIntelligenceService.ProcessPhase3IntelligenceWithRustEngine(ctx, request)
    if err != nil {
        return fmt.Errorf("Phase 3 processing failed: %w", err)
    }
    
    // Validate performance targets were met
    if result.ProcessingMetrics.TotalProcessingTime > 250*time.Millisecond {
        log.Warn("Phase 3 performance target missed")
    }
    
    return nil
}
```

## Performance Monitoring

### Metrics Collection

The system automatically collects comprehensive performance metrics:

```go
// Get operation metrics
metrics := monitoringService.GetOperationMetrics()
for operationType, metric := range metrics {
    log.Printf("%s: Avg=%v, P95=%v, Success Rate=%.2f%%",
        operationType,
        metric.AverageResponseTime,
        metric.P95ResponseTime,
        float64(metric.SuccessfulRequests)/float64(metric.TotalRequests)*100)
}
```

### Performance Alerts

Automatic alerts for performance degradation:

```go
// Check for performance alerts
alerts := monitoringService.GetPerformanceAlerts()
for _, alert := range alerts {
    if !alert.Acknowledged {
        log.Printf("Performance Alert: %s - %s (Severity: %s)",
            alert.OperationType,
            alert.Message,
            alert.Severity)
    }
}
```

### Health Monitoring

Continuous health monitoring with caching:

```go
// Check engine health (cached for efficiency)
isHealthy, healthDetails := monitoringService.IsHealthy(ctx)
if !isHealthy {
    log.Warn("Rust engine is unhealthy", "details", healthDetails)
}
```

## Error Handling and Resilience

### Circuit Breaker Pattern

Automatic circuit breaker protection:

```go
// Check if operations can be executed
if !monitoringService.CanExecuteOperation() {
    return fmt.Errorf("circuit breaker is open - Rust engine unavailable")
}

// Execute operation with circuit breaker protection
result, err := rustClient.AnalyzeDrugInteractions(ctx, request)
```

### Fallback Mechanisms

When the Rust engine is unavailable:

```go
func ProcessWithFallback(ctx context.Context, request *ClinicalRequest) (*ClinicalResult, error) {
    // Try Rust engine first
    if monitoringService.CanExecuteOperation() {
        result, err := processWithRustEngine(ctx, request)
        if err == nil {
            return result, nil
        }
        log.Warn("Rust engine failed, falling back to legacy processing", "error", err)
    }
    
    // Fallback to legacy processing
    return processWithLegacyEngine(ctx, request)
}
```

## Testing

### Unit Tests

```bash
# Run Rust engine integration tests
go test ./internal/infrastructure/clients -v -run TestRustClinicalEngine

# Run clinical calculation service tests
go test ./internal/application/services -v -run TestClinicalCalculationService
```

### Integration Tests

```bash
# Start Rust engine (for integration tests)
cd ../flow2-rust-engine
cargo run --release

# Run integration tests
cd ../medication-service-v2
go test ./tests/integration -v -run TestRustEngineIntegration
```

### Performance Tests

```bash
# Run performance benchmarks
go test ./internal/infrastructure/clients -bench=BenchmarkRustEngine -benchmem

# Load testing with realistic payloads
go test ./tests/performance -v -run TestRustEngineLoad
```

## Deployment Considerations

### Production Deployment

1. **Resource Requirements**:
   - Rust Engine: 2 CPU cores, 4GB RAM minimum
   - Network: Low-latency connection between services
   - Storage: Minimal (stateless operation)

2. **Scaling Strategy**:
   - Horizontal scaling: Multiple Rust engine instances
   - Load balancing: Round-robin with health checks
   - Auto-scaling: Based on request rate and response times

3. **Monitoring Setup**:
   - Prometheus metrics export
   - Grafana dashboards for visualization
   - AlertManager for critical alerts

### Health Checks

```yaml
# Kubernetes health check example
livenessProbe:
  httpGet:
    path: /health
    port: 8090
  initialDelaySeconds: 30
  periodSeconds: 10
  
readinessProbe:
  httpGet:
    path: /health
    port: 8090
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Troubleshooting

### Common Issues

1. **Connection Timeouts**:
   ```bash
   # Check network connectivity
   curl http://rust-engine:8090/health
   
   # Verify configuration
   grep base_url config/rust_engine_config.yaml
   ```

2. **Performance Degradation**:
   ```go
   // Check performance metrics
   metrics := monitoringService.GetOperationMetrics()
   
   // Review performance alerts
   alerts := monitoringService.GetPerformanceAlerts()
   ```

3. **Circuit Breaker Open**:
   ```go
   // Check circuit breaker status
   status := monitoringService.GetCircuitBreakerStatus()
   
   // Wait for recovery or manual reset
   ```

### Debug Logging

Enable detailed logging for troubleshooting:

```yaml
# In rust_engine_config.yaml
logging:
  level: "debug"
  enable_request_logging: true
  enable_performance_logging: true
```

## Future Enhancements

1. **gRPC Communication**: Migrate to gRPC for even better performance
2. **Streaming APIs**: Support for real-time streaming calculations
3. **ML Integration**: Machine learning model integration
4. **Advanced Caching**: Distributed caching with Redis
5. **Batch Processing**: Support for bulk operations

## Support

For issues with the Rust Clinical Engine integration:

1. Check the logs: `tail -f logs/rust-engine-integration.log`
2. Verify configuration: Review `config/rust_engine_config.yaml`
3. Test connectivity: Use provided health check endpoints
4. Monitor performance: Review metrics and alerts
5. Consult documentation: This file and inline code comments

The Rust Clinical Engine integration provides the high-performance clinical intelligence needed for Phase 3 operations, ensuring sub-millisecond response times for critical clinical decision support.