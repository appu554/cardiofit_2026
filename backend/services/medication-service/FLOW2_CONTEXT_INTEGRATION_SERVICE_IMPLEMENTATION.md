# Flow 2: Context Integration Service Implementation Guide

## Overview

This document provides the detailed implementation plan for the **Context Integration Service** module within the Go Orchestrator. This service executes **Phase 2: Context Assembly** in the new Flow 2 architecture, transforming from sequential processing to high-performance parallel clinical intelligence.

## 🎯 Architectural Evolution

### Current Flow (Sequential)
```
Phase 1: ORB Engine → Intent Manifest
Phase 2: Context Service → Clinical Context  
Phase 3: Rust Engine → Medication Proposal
Phase 4: Response Assembly → Final Response
```

### New Flow (Parallel Clinical Intelligence)
```
Phase 1: ORB Engine → Intent Manifest
Phase 2: Context Integration Service → CompleteContextPayload
Phase 3: [Calculation Engine || Clinical Rules Engine || Formulary Intelligence] → Multiple Results
Phase 4: Recommendation Engine → Ranked Proposals → Final Assembly
```

## 🏗️ Context Integration Service Architecture

### Role & Responsibility
- **Location**: Module within Go Orchestrator (not a separate microservice)
- **Purpose**: Execute Phase 2 - Master orchestrator for all data fetching
- **Input**: Intent Manifest from Phase 1 (ORB Engine)
- **Output**: CompleteContextPayload for Phase 3 (Clinical Intelligence Engine)

### Architectural Placement
```
┌─────────────────────────────────────────────────────────────┐
│                    Go Orchestrator                          │
│  ┌─────────────┐    ┌─────────────────────┐    ┌─────────┐  │
│  │ Phase 1:    │───▶│ Phase 2: Context    │───▶│ Phase 3:│  │
│  │ ORB Engine  │    │ Integration Service │    │ Clinical│  │
│  │             │    │                     │    │ Intel.  │  │
│  └─────────────┘    └─────────────────────┘    └─────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────────────────────┐
        │           Downstream Dependencies                   │
        │  ┌─────────────────────┐  ┌─────────────────────┐   │
        │  │ Your Existing       │  │ 7 New Knowledge     │   │
        │  │ Context Gateway     │  │ Base Services       │   │
        │  │ Service             │  │                     │   │
        │  └─────────────────────┘  └─────────────────────┘   │
        └─────────────────────────────────────────────────────┘
```

## 📊 Enhanced Data Structures

### Core Data Payloads

```go
// CompleteContextPayload - The master payload for Phase 3
type CompleteContextPayload struct {
    Patient     PatientContext      `json:"patient"`
    Knowledge   KnowledgeContext    `json:"knowledge"`
    Metadata    ContextMetadata     `json:"metadata"`
    Provenance  map[string]string   `json:"provenance"`
    CacheInfo   CacheInformation    `json:"cache_info"`
}

// PatientContext - All patient-specific clinical data
type PatientContext struct {
    Demographics    PatientDemographics `json:"demographics"`
    ActiveMedications []Medication      `json:"active_medications"`
    Allergies       []Allergy          `json:"allergies"`
    Conditions      []Condition        `json:"conditions"`
    LabResults      []LabResult        `json:"lab_results"`
    VitalSigns      []VitalSign        `json:"vital_signs"`
}

// KnowledgeContext - All clinical knowledge data
type KnowledgeContext struct {
    DrugInteractions    []DrugInteraction   `json:"drug_interactions"`
    FormularyInfo       []FormularyEntry    `json:"formulary_info"`
    ClinicalGuidelines  []Guideline         `json:"clinical_guidelines"`
    DosageReferences    []DosageReference   `json:"dosage_references"`
    SafetyAlerts        []SafetyAlert       `json:"safety_alerts"`
    MonitoringProtocols []MonitoringRule    `json:"monitoring_protocols"`
    EvidenceBase        []EvidenceEntry     `json:"evidence_base"`
}

// ContextMetadata - Processing and quality information
type ContextMetadata struct {
    RequestID           string              `json:"request_id"`
    PatientID           string              `json:"patient_id"`
    AssemblyTimeMs      int64               `json:"assembly_time_ms"`
    DataCompleteness    float64             `json:"data_completeness"`
    QualityScore        float64             `json:"quality_score"`
    DataSources         []string            `json:"data_sources"`
    RetrievalErrors     []RetrievalError    `json:"retrieval_errors"`
}
```

## 🚀 Implementation Logic

### Core Assembly Function

```go
// AssembleContext - Main function for Context Integration Service
func (s *ContextIntegrationService) AssembleContext(
    ctx context.Context, 
    manifest *orb.IntentManifest,
) (*CompleteContextPayload, error) {
    
    startTime := time.Now()
    
    // ENHANCEMENT 1: AGGRESSIVE L3 REDIS CACHING
    cacheKey := s.generateCacheKey(manifest.PatientID, manifest.ContextRecipeID)
    if cachedPayload := s.checkL3Cache(ctx, cacheKey); cachedPayload != nil {
        s.logger.Info("CACHE HIT: Returning complete context from L3 Redis cache")
        return cachedPayload, nil
    }
    
    // ENHANCEMENT 2: MASSIVE PARALLELISM
    var g errgroup.Group
    var patientData PatientContext
    var knowledgeData KnowledgeContext
    var retrievalErrors []RetrievalError
    
    // Goroutine 1: Patient Data via Context Gateway Service
    g.Go(func() error {
        s.logger.WithFields(logrus.Fields{
            "context_recipe_id": manifest.ContextRecipeID,
            "data_requirements": len(manifest.DataRequirements),
        }).Info("Fetching patient data from Context Gateway")
        
        pData, err := s.contextGatewayClient.FetchPatientData(ctx, manifest)
        if err != nil {
            retrievalErrors = append(retrievalErrors, RetrievalError{
                Source: "context_gateway",
                Error:  err.Error(),
                Impact: "high",
            })
            return fmt.Errorf("Context Gateway failed: %w", err)
        }
        patientData = pData
        return nil
    })
    
    // Goroutine 2: Knowledge Base Data (Parallel KB Queries)
    g.Go(func() error {
        s.logger.WithFields(logrus.Fields{
            "kb_hints": manifest.KBHints,
            "recipe_id": manifest.RecipeID,
        }).Info("Fetching knowledge data from KB services")
        
        // ENHANCEMENT 3: MANIFEST-DRIVEN KB QUERIES
        kData, err := s.kbClient.FetchKnowledgeData(ctx, manifest.KBHints)
        if err != nil {
            // Partial KB failure is acceptable - flag but continue
            retrievalErrors = append(retrievalErrors, RetrievalError{
                Source: "knowledge_bases",
                Error:  err.Error(),
                Impact: "medium",
            })
            s.logger.Warn("Knowledge Base fetch returned partial data", "error", err)
        }
        knowledgeData = kData
        return nil // Don't halt entire process for KB failures
    })
    
    // Wait for all parallel operations
    if err := g.Wait(); err != nil {
        s.logger.Error("Failed to assemble context", "error", err)
        
        // ENHANCEMENT 4: STALE-WHILE-REVALIDATE
        if stalePayload := s.getStaleFromCache(ctx, cacheKey); stalePayload != nil {
            s.logger.Warn("SERVING STALE: Primary fetch failed, returning stale data")
            return stalePayload, nil
        }
        return nil, err
    }
    
    // Assemble final payload
    completePayload := s.assembleCompletePayload(
        patientData, knowledgeData, manifest, retrievalErrors, startTime,
    )
    
    // Cache for future requests
    s.cacheL3(ctx, cacheKey, completePayload, 5*time.Minute)
    
    s.logger.WithFields(logrus.Fields{
        "assembly_time_ms": completePayload.Metadata.AssemblyTimeMs,
        "data_completeness": completePayload.Metadata.QualityScore,
        "cache_status": "fresh",
    }).Info("Context assembly completed successfully")
    
    return completePayload, nil
}
```

## 🔧 Key Enhancements

### 1. Massive Parallelism
- **Before**: Sequential calls (Context Service → Wait → KB Services → Wait)
- **After**: Parallel execution using `errgroup.Group`
- **Benefit**: Latency = max(slowest_dependency) instead of sum(all_dependencies)

### 2. Aggressive L3 Redis Caching
- **Implementation**: Complete payload cached in Redis
- **TTL**: 5 minutes for clinical data
- **Benefit**: Sub-millisecond response for cached requests

### 3. Enhanced Resilience
- **Circuit Breakers**: Wrap all network calls
- **Stale-while-revalidate**: Serve cached data if fresh fetch fails
- **Partial Failure Handling**: Continue processing if non-critical KB fails

### 4. Manifest-Driven KB Queries
- **Smart Targeting**: Only query relevant KBs based on Intent Manifest hints
- **Example**: TB case queries resistance profiles, Heparin case skips them
- **Benefit**: Reduced network traffic and improved efficiency

## 📈 Performance Expectations

### Current Performance
```
Context Service: ~200ms
Total Phase 2: ~200ms
```

### New Performance
```
Patient Data (Context Gateway): ~200ms
Knowledge Data (7 KBs in parallel): ~150ms
Total Phase 2: max(200ms, 150ms) = ~200ms
```

### Cache Performance
```
L3 Redis Hit: <5ms
Cache Hit Ratio: 60-80% expected
Effective Average: ~60ms
```

## 🎯 Integration Points

### Input: Intent Manifest
```go
type IntentManifest struct {
    RequestID           string   `json:"request_id"`
    PatientID           string   `json:"patient_id"`
    RecipeID            string   `json:"recipe_id"`
    ContextRecipeID     string   `json:"context_recipe_id"`
    DataRequirements    []string `json:"data_requirements"`
    KBHints             []string `json:"kb_hints"`
    Priority            string   `json:"priority"`
    ClinicalRationale   string   `json:"clinical_rationale"`
}
```

### Output: CompleteContextPayload
- Ready for Phase 3 parallel processing
- Contains all data needed for Calculation, Safety, and Formulary engines
- Includes quality metrics and provenance information

## 💻 Go Implementation Structure

### File Organization
```
flow2-go-engine/internal/
├── context/
│   ├── integration_service.go      # Main Context Integration Service
│   ├── cache_manager.go           # L3 Redis caching implementation
│   ├── kb_client.go              # Knowledge Base services client
│   └── models.go                 # Enhanced data structures
├── clients/
│   ├── context_gateway_client.go  # Enhanced Context Gateway client
│   └── interfaces.go             # Updated interfaces
└── services/
    └── circuit_breaker.go        # Circuit breaker implementation
```

### Context Integration Service Interface
```go
// ContextIntegrationService interface
type ContextIntegrationService interface {
    // Main assembly method
    AssembleContext(ctx context.Context, manifest *orb.IntentManifest) (*CompleteContextPayload, error)

    // Cache management
    InvalidateCache(patientID string) error
    GetCacheStats() CacheStatistics

    // Health and monitoring
    HealthCheck() error
    GetMetrics() IntegrationMetrics
}

// Supporting interfaces
type KnowledgeBaseClient interface {
    FetchKnowledgeData(ctx context.Context, kbHints []string) (KnowledgeContext, error)
    GetAvailableKBs() []string
    HealthCheck() map[string]bool
}

type CacheManager interface {
    Get(ctx context.Context, key string) (*CompleteContextPayload, error)
    Set(ctx context.Context, key string, payload *CompleteContextPayload, ttl time.Duration) error
    GetStale(ctx context.Context, key string) (*CompleteContextPayload, error)
    InvalidatePattern(pattern string) error
}
```

### Enhanced Context Gateway Client
```go
// Enhanced client for your existing Context Gateway Service
type ContextGatewayClient struct {
    httpClient    *http.Client
    baseURL       string
    circuitBreaker *CircuitBreaker
    logger        *logrus.Logger
}

func (c *ContextGatewayClient) FetchPatientData(
    ctx context.Context,
    manifest *orb.IntentManifest,
) (PatientContext, error) {

    request := ContextGatewayRequest{
        PatientID:        manifest.PatientID,
        RecipeID:         manifest.ContextRecipeID,
        DataRequirements: manifest.DataRequirements,
        Priority:         manifest.Priority,
    }

    // Use circuit breaker for resilience
    result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
        return c.makeHTTPRequest(ctx, request)
    })

    if err != nil {
        return PatientContext{}, err
    }

    return result.(PatientContext), nil
}
```

## 🔄 Implementation Phases

### Phase 1: Foundation (Week 1)
- [ ] Create enhanced data structures (`models.go`)
- [ ] Implement basic Context Integration Service skeleton
- [ ] Set up L3 Redis caching infrastructure
- [ ] Create Knowledge Base client interfaces

### Phase 2: Core Implementation (Week 2)
- [ ] Implement parallel data fetching with `errgroup`
- [ ] Build Context Gateway client with circuit breakers
- [ ] Add stale-while-revalidate caching logic
- [ ] Implement manifest-driven KB query optimization

### Phase 3: Integration & Testing (Week 3)
- [ ] Integration testing with existing Context Gateway Service
- [ ] Performance benchmarking and optimization
- [ ] Error handling and resilience testing
- [ ] Monitoring and metrics implementation

### Phase 4: Production Readiness (Week 4)
- [ ] Load testing and capacity planning
- [ ] Documentation and runbooks
- [ ] Deployment automation
- [ ] Monitoring dashboards and alerts

## 🎯 Success Metrics

### Performance Targets
- **Cache Hit Ratio**: >70%
- **P95 Latency**: <300ms (fresh data)
- **P95 Latency**: <10ms (cached data)
- **Availability**: >99.9%

### Quality Targets
- **Data Completeness**: >95%
- **Error Rate**: <0.1%
- **Circuit Breaker Activation**: <1% of requests

This Context Integration Service forms the foundation for the new parallel clinical intelligence architecture, enabling the system to generate multiple ranked medication proposals with the same latency as the current single proposal system.
