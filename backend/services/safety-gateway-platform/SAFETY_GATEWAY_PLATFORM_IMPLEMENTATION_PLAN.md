# **Safety Gateway Platform: Comprehensive Implementation Plan**

## **Executive Summary**

The Safety Gateway Platform is a dedicated microservice that extracts orchestration logic from individual clinical services, creating a centralized, high-performance safety conductor that coordinates multiple clinical reasoning engines while maintaining strict safety guarantees.

## **Architecture Overview**

### **Core Principle: Platform as Conductor**
```
The Safety Gateway Platform orchestrates safety engines but contains ZERO clinical logic.
It is the conductor, not the musician.
```

### **Service Architecture: Single Process with Pluggable Engines**

> **CRITICAL ARCHITECTURAL PRINCIPLE**: For sub-200ms performance and atomic decision-making, all engines run as **in-process modules** within the same Go binary, not as separate networked microservices.

```
┌─────────────────────────────────────────────────────────────┐
│         Safety Gateway Platform (Single Go Binary)          │
│                     (Port 8030)                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Ingress   │  │   Context   │  │  Override   │        │
│  │ Validator   │  │  Assembly   │  │   Token     │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
│                                                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │            Orchestration Engine                     │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │   │
│  │  │   Request   │  │   Engine    │  │  Response   │  │   │
│  │  │   Router    │  │  Registry   │  │ Aggregator  │  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │   │
│  └─────────────────────────────────────────────────────┘   │
│                            │                               │
│                            ▼ (In-Memory Function Calls)    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Pluggable Engine Layer                 │   │
│  │ ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────┐ │   │
│  │ │    CAE    │ │ Protocol  │ │Constraint │ │GraphDB│ │   │
│  │ │  Engine   │ │  Engine   │ │Validator  │ │Assert │ │   │
│  │ │ (Tier 1)  │ │ (Tier 2)  │ │ (Tier 1)  │ │(Tier2)│ │   │
│  │ └───────────┘ └───────────┘ └───────────┘ └───────┘ │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ Explanation │  │    Audit    │  │ Performance │        │
│  │   Engine    │  │   Logger    │  │  Monitor    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

**Key Architectural Benefits:**
- **Performance**: Sub-millisecond engine calls (vs 5-15ms network calls)
- **Atomicity**: Shared memory space ensures consistent `ClinicalContext`
- **Simplicity**: Single binary deployment, no service mesh complexity
- **Reliability**: No network failures between engines

## **Core Components Design**

### **1. Ingress Validator**
**Purpose**: First line of defense - validate and sanitize all incoming requests

```go
type IngressValidator struct {
    maxRequestSize   int64
    rateLimiter     *RateLimiter
    schemaValidator *SchemaValidator
}

func (v *IngressValidator) ValidateRequest(req *SafetyRequest) error {
    // Size validation
    if req.Size() > v.maxRequestSize {
        return ErrRequestTooLarge
    }
    
    // Rate limiting
    if !v.rateLimiter.Allow(req.ClientID) {
        return ErrRateLimitExceeded
    }
    
    // Schema validation
    if err := v.schemaValidator.Validate(req); err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }
    
    return nil
}
```

### **2. Context Assembly Service**
**Purpose**: Optimized patient data fetching and context building

```go
type ContextAssemblyService struct {
    fhirClient    *FHIRClient
    graphClient   *GraphDBClient
    cacheManager  *CacheManager
    contextBuilder *ContextBuilder
}

type ClinicalContext struct {
    PatientID        string                 `json:"patient_id"`
    Demographics     *PatientDemographics   `json:"demographics"`
    ActiveMedications []Medication          `json:"active_medications"`
    Allergies        []Allergy             `json:"allergies"`
    Conditions       []Condition           `json:"conditions"`
    RecentVitals     []VitalSign           `json:"recent_vitals"`
    LabResults       []LabResult           `json:"lab_results"`
    ContextVersion   string                `json:"context_version"`
    AssemblyTime     time.Time             `json:"assembly_time"`
}

func (c *ContextAssemblyService) AssembleContext(patientID string) (*ClinicalContext, error) {
    // Check cache first
    if cached := c.cacheManager.Get(patientID); cached != nil {
        return cached, nil
    }
    
    // Parallel data fetching
    var wg sync.WaitGroup
    var demographics *PatientDemographics
    var medications []Medication
    var allergies []Allergy
    
    wg.Add(3)
    
    go func() {
        defer wg.Done()
        demographics, _ = c.fhirClient.GetPatientDemographics(patientID)
    }()
    
    go func() {
        defer wg.Done()
        medications, _ = c.fhirClient.GetActiveMedications(patientID)
    }()
    
    go func() {
        defer wg.Done()
        allergies, _ = c.fhirClient.GetAllergies(patientID)
    }()
    
    wg.Wait()
    
    // Build context
    context := c.contextBuilder.Build(patientID, demographics, medications, allergies)
    
    // Cache with TTL
    c.cacheManager.Set(patientID, context, 5*time.Minute)
    
    return context, nil
}
```

### **3. Engine Registry (In-Process Pluggable Engines)**
**Purpose**: Registration and management of in-process clinical reasoning engines

**Pluggable Engine Interface:**
```go
type SafetyEngine interface {
    // Core engine methods
    ID() string
    Name() string
    Capabilities() []string

    // Safety evaluation
    Evaluate(ctx context.Context, req *SafetyRequest, clinicalContext *ClinicalContext) (*EngineResult, error)

    // Health and lifecycle
    HealthCheck() error
    Initialize(config EngineConfig) error
    Shutdown() error
}

type EngineRegistry struct {
    engines     map[string]*EngineInfo
    mutex       sync.RWMutex
}

type EngineInfo struct {
    ID           string              `json:"id"`
    Name         string              `json:"name"`
    Instance     SafetyEngine        `json:"-"` // In-process instance
    Capabilities []string            `json:"capabilities"`
    Tier         CriticalityTier     `json:"tier"`
    Priority     int                 `json:"priority"`
    Timeout      time.Duration       `json:"timeout"`
    Status       EngineStatus        `json:"status"`
    LastCheck    time.Time           `json:"last_check"`
}

type CriticalityTier int

const (
    TierVetoCritical CriticalityTier = 1  // CAE, Allergy Check - failure = UNSAFE
    TierAdvisory     CriticalityTier = 2  // Formulary, Duplicate - failure = WARNING
)

**Supported In-Process Engines:**
- **CAE Engine** (Tier 1) - Drug interactions, contraindications
- **Allergy Engine** (Tier 1) - Allergy contraindications
- **Protocol Engine** (Tier 2) - Clinical guidelines compliance
- **Constraint Validator** (Tier 1) - Hard safety constraints
- **GraphDB Asserter** (Tier 2) - Pattern-based insights

```go
func (r *EngineRegistry) RegisterEngine(engine SafetyEngine, tier CriticalityTier, priority int) error {
    if err := engine.HealthCheck(); err != nil {
        return fmt.Errorf("engine %s failed health check: %w", engine.ID(), err)
    }

    info := &EngineInfo{
        ID:           engine.ID(),
        Name:         engine.Name(),
        Instance:     engine,
        Capabilities: engine.Capabilities(),
        Tier:         tier,
        Priority:     priority,
        Timeout:      r.getTimeoutForTier(tier),
        Status:       EngineStatusHealthy,
        LastCheck:    time.Now(),
    }

    r.engines[engine.ID()] = info
    return nil
}

func (r *EngineRegistry) GetEnginesForRequest(req *SafetyRequest) []*EngineInfo {
    r.mutex.RLock()
    defer r.mutex.RUnlock()

    var selectedEngines []*EngineInfo

    for _, engine := range r.engines {
        if engine.Status == EngineStatusHealthy &&
           r.engineSupportsRequest(engine, req) {
            selectedEngines = append(selectedEngines, engine)
        }
    }

    // Sort by tier first (Tier 1 critical), then priority
    sort.Slice(selectedEngines, func(i, j int) bool {
        if selectedEngines[i].Tier != selectedEngines[j].Tier {
            return selectedEngines[i].Tier < selectedEngines[j].Tier
        }
        return selectedEngines[i].Priority > selectedEngines[j].Priority
    })

    return selectedEngines
}
```

### **4. Orchestration Engine**
**Purpose**: Core coordination logic with deterministic execution

```go
type OrchestrationEngine struct {
    registry        *EngineRegistry
    contextService  *ContextAssemblyService
    responseBuilder *ResponseBuilder
    circuitBreaker  *CircuitBreaker
}

func (o *OrchestrationEngine) ProcessSafetyRequest(req *SafetyRequest) (*SafetyResponse, error) {
    startTime := time.Now()
    
    // Assemble clinical context
    context, err := o.contextService.AssembleContext(req.PatientID)
    if err != nil {
        return nil, fmt.Errorf("context assembly failed: %w", err)
    }
    
    // Get applicable engines
    engines := o.registry.GetEnginesForRequest(req)
    if len(engines) == 0 {
        return nil, ErrNoEnginesAvailable
    }
    
    // Execute engines in parallel with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
    defer cancel()
    
    results := o.executeEnginesParallel(ctx, engines, req, context)
    
    // Aggregate results
    response := o.responseBuilder.AggregateResults(req, results)
    
    // Add timing information
    response.ProcessingTime = time.Since(startTime)
    
    return response, nil
}

func (o *OrchestrationEngine) executeEnginesParallel(ctx context.Context, engines []*EngineInfo, req *SafetyRequest, clinicalContext *ClinicalContext) []EngineResult {
    resultsChan := make(chan EngineResult, len(engines))

    for _, engine := range engines {
        go func(eng *EngineInfo) {
            // In-process function call - sub-millisecond latency
            result := o.executeEngineInProcess(ctx, eng, req, clinicalContext)
            resultsChan <- result
        }(engine)
    }

    var results []EngineResult
    for i := 0; i < len(engines); i++ {
        select {
        case result := <-resultsChan:
            results = append(results, result)
        case <-ctx.Done():
            // Timeout - return what we have
            break
        }
    }

    return results
}

func (o *OrchestrationEngine) executeEngineInProcess(ctx context.Context, engineInfo *EngineInfo, req *SafetyRequest, clinicalContext *ClinicalContext) EngineResult {
    startTime := time.Now()

    // Create engine-specific context with timeout
    engineCtx, cancel := context.WithTimeout(ctx, engineInfo.Timeout)
    defer cancel()

    // Execute engine (in-process function call)
    result, err := engineInfo.Instance.Evaluate(engineCtx, req, clinicalContext)

    duration := time.Since(startTime)

    if err != nil {
        // Handle failure based on criticality tier
        if engineInfo.Tier == TierVetoCritical {
            return EngineResult{
                EngineID:  engineInfo.ID,
                Status:    "UNSAFE", // Fail closed for critical engines
                Error:     err.Error(),
                Duration:  duration,
                Tier:      engineInfo.Tier,
            }
        } else {
            return EngineResult{
                EngineID:  engineInfo.ID,
                Status:    "WARNING", // Degraded for advisory engines
                Error:     err.Error(),
                Duration:  duration,
                Tier:      engineInfo.Tier,
            }
        }
    }

    result.Duration = duration
    result.Tier = engineInfo.Tier
    return *result
}
```

## **Criticality Tiers & Failure Handling**

### **Engine Criticality Tiers**
```go
const (
    TierVetoCritical CriticalityTier = 1  // Failure = UNSAFE (fail closed)
    TierAdvisory     CriticalityTier = 2  // Failure = WARNING (degraded)
)
```

**Tier 1 (Veto-Critical) Engines:**
- **CAE Engine**: Drug interactions, contraindications
- **Allergy Engine**: Allergy contraindications
- **Constraint Validator**: Hard safety constraints

**Tier 2 (Advisory) Engines:**
- **Protocol Engine**: Clinical guidelines compliance
- **GraphDB Asserter**: Pattern-based insights
- **Formulary Engine**: Drug availability checks

**Aggregation Rules by Tier:**
```
ANY Tier 1 engine returns UNSAFE     → Final: UNSAFE
ANY Tier 1 engine fails/timeouts     → Final: UNSAFE (fail closed)
ALL Tier 1 engines return SAFE       → Proceed to Tier 2 evaluation
Tier 2 engine failures               → WARNING (degraded check)
```

## **Response Time Budget (200ms Total) - In-Process Optimized**

```
┌─────────────────────────────────────────────────────────┐
│            Response Time Allocation (In-Process)        │
├─────────────────────────────────────────────────────────┤
│  Context Assembly:      20ms  (10%)                    │
│  ├── Cache Check:        2ms                           │
│  ├── FHIR Queries:      15ms  (parallel)              │
│  └── Context Build:      3ms                           │
│                                                         │
│  Engine Execution:     160ms  (80%) ← IMPROVED         │
│  ├── Function Calls:   <1ms   (in-process)            │
│  ├── Engine Processing: 155ms (parallel)              │
│  └── Context Switching:  4ms  (goroutine overhead)    │
│                                                         │
│  Result Aggregation:    10ms  (5%)                     │
│  ├── Tier-based Rules:   5ms                          │
│  ├── Explanation Build:  3ms                           │
│  └── Response Format:    2ms                           │
│                                                         │
│  Audit Logging:         5ms   (2.5%) - Async          │
│  Network Overhead:      5ms   (2.5%)                   │
└─────────────────────────────────────────────────────────┘

Performance Gains from In-Process Architecture:
✓ Eliminated 5-15ms network latency per engine call
✓ Shared memory space - no serialization overhead
✓ Sub-millisecond function call latency
✓ 10ms additional budget for engine processing
```

## **Failure Handling Strategy**

### **Circuit Breaker Pattern**
```go
type CircuitBreaker struct {
    failureThreshold int
    resetTimeout     time.Duration
    state           CircuitState
    failures        int
    lastFailTime    time.Time
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    if cb.state == CircuitStateOpen {
        if time.Since(cb.lastFailTime) > cb.resetTimeout {
            cb.state = CircuitStateHalfOpen
        } else {
            return ErrCircuitBreakerOpen
        }
    }
    
    err := fn()
    
    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()
        
        if cb.failures >= cb.failureThreshold {
            cb.state = CircuitStateOpen
        }
        
        return err
    }
    
    // Success - reset circuit breaker
    cb.failures = 0
    cb.state = CircuitStateClosed
    return nil
}
```

### **Fail-Closed Principle**
```
Engine Failure Scenarios:
├── Engine Timeout → Mark as UNSAFE (conservative)
├── Engine Crash → Mark as UNSAFE (conservative)  
├── Network Error → Mark as UNSAFE (conservative)
├── Invalid Response → Mark as UNSAFE (conservative)
└── Unknown Error → Mark as UNSAFE (conservative)

Aggregation Rules:
├── ANY engine returns UNSAFE → Final: UNSAFE
├── ALL engines return SAFE → Final: SAFE
├── Mixed results → Apply consensus rules
└── No engines respond → Final: MANUAL_REVIEW
```

## **Implementation Phases**

### **Phase 1: Core Platform (Week 1-2)**
- [ ] Basic service structure and gRPC server
- [ ] Ingress validator with request validation
- [ ] Engine registry with health monitoring
- [ ] Basic orchestration engine
- [ ] Response aggregation logic

### **Phase 2: Context Service (Week 3)**
- [ ] Context assembly service
- [ ] Multi-source data fetching (FHIR, GraphDB)
- [ ] Intelligent caching with TTL
- [ ] Context versioning and immutability

### **Phase 3: Safety Features (Week 4)**
- [ ] Circuit breaker implementation
- [ ] Timeout enforcement
- [ ] Fail-closed behavior
- [ ] Override token system

### **Phase 4: Observability (Week 5)**
- [ ] Structured logging
- [ ] Metrics collection
- [ ] Distributed tracing
- [ ] Audit trail

### **Phase 5: Advanced Features (Week 6-7)**
- [ ] Explanation engine
- [ ] Performance optimization
- [ ] Load balancing
- [ ] Configuration management

### **Phase 6: Testing & Validation (Week 8)**
- [ ] Chaos engineering tests
- [ ] Performance benchmarking
- [ ] Clinical workflow validation
- [ ] Security testing

## **Service Configuration**

```yaml
# safety-gateway-platform/config.yaml
service:
  name: "safety-gateway-platform"
  port: 8030
  version: "1.0.0"

performance:
  max_concurrent_requests: 1000
  request_timeout_ms: 200
  context_assembly_timeout_ms: 20
  engine_execution_timeout_ms: 150

engines:
  cae_service:
    endpoint: "localhost:8027"
    timeout_ms: 100
    priority: 10
    capabilities: ["drug_interaction", "contraindication", "dosing"]
  
  protocol_engine:
    endpoint: "localhost:8031"
    timeout_ms: 80
    priority: 8
    capabilities: ["clinical_protocol", "guideline_compliance"]

circuit_breaker:
  failure_threshold: 5
  reset_timeout_seconds: 30
  half_open_max_calls: 3

caching:
  context_ttl_minutes: 5
  max_cache_size_mb: 100
  eviction_policy: "lru"

observability:
  log_level: "info"
  metrics_enabled: true
  tracing_enabled: true
  audit_log_async: true
```

## **Next Steps**

1. **Create service structure** in `backend/services/safety-gateway-platform/`
2. **Extract orchestration logic** from current CAE service
3. **Implement core components** following the design above
4. **Update CAE service** to register as an engine
5. **Test integration** with existing clinical services
6. **Deploy and monitor** the new architecture

This architecture creates a robust, scalable, and maintainable safety platform that can serve all clinical services while maintaining the highest safety standards.

## **Override Token System Design**

### **Decision Token Architecture**
```go
type OverrideToken struct {
    TokenID          string                 `json:"token_id"`
    RequestID        string                 `json:"request_id"`
    PatientID        string                 `json:"patient_id"`
    DecisionSummary  *DecisionSummary       `json:"decision_summary"`
    RequiredLevel    OverrideLevel          `json:"required_level"`
    ExpiresAt        time.Time              `json:"expires_at"`
    ContextHash      string                 `json:"context_hash"`
    CreatedAt        time.Time              `json:"created_at"`
    Signature        string                 `json:"signature"`
}

type DecisionSummary struct {
    Status              SafetyStatus           `json:"status"`
    CriticalViolations  []string              `json:"critical_violations"`
    EnginesFailed       []string              `json:"engines_failed"`
    RiskScore           float64               `json:"risk_score"`
    Explanation         string                `json:"explanation"`
}

type OverrideLevel string

const (
    OverrideLevelResident    OverrideLevel = "resident"
    OverrideLevelAttending   OverrideLevel = "attending"
    OverrideLevelPharmacist  OverrideLevel = "pharmacist"
    OverrideLevelChief       OverrideLevel = "chief"
)
```

### **Override Service Implementation**
```go
type OverrideService struct {
    tokenStore    TokenStore
    cryptoService *CryptoService
    auditLogger   *AuditLogger
    rbacService   *RBACService
}

func (o *OverrideService) GenerateOverrideToken(req *SafetyRequest, unsafeResponse *SafetyResponse) (*OverrideToken, error) {
    // Determine required override level based on risk
    requiredLevel := o.determineRequiredOverrideLevel(unsafeResponse)

    // Create token
    token := &OverrideToken{
        TokenID:     generateUUID(),
        RequestID:   req.RequestID,
        PatientID:   req.PatientID,
        DecisionSummary: &DecisionSummary{
            Status:             unsafeResponse.Status,
            CriticalViolations: unsafeResponse.CriticalViolations,
            EnginesFailed:      unsafeResponse.EnginesFailed,
            RiskScore:          unsafeResponse.RiskScore,
            Explanation:        unsafeResponse.Explanation,
        },
        RequiredLevel: requiredLevel,
        ExpiresAt:     time.Now().Add(5 * time.Minute),
        ContextHash:   o.computeContextHash(req),
        CreatedAt:     time.Now(),
    }

    // Sign token
    signature, err := o.cryptoService.SignToken(token)
    if err != nil {
        return nil, err
    }
    token.Signature = signature

    // Store token
    if err := o.tokenStore.Store(token); err != nil {
        return nil, err
    }

    // Audit log
    o.auditLogger.LogTokenGeneration(token)

    return token, nil
}

func (o *OverrideService) ValidateOverride(tokenID string, clinicianID string, reason string) (*OverrideValidation, error) {
    // Retrieve token
    token, err := o.tokenStore.Get(tokenID)
    if err != nil {
        return nil, err
    }

    // Check expiration
    if time.Now().After(token.ExpiresAt) {
        return &OverrideValidation{Valid: false, Reason: "Token expired"}, nil
    }

    // Verify signature
    if !o.cryptoService.VerifyToken(token) {
        return &OverrideValidation{Valid: false, Reason: "Invalid token signature"}, nil
    }

    // Check clinician authorization
    clinicianLevel, err := o.rbacService.GetClinicianOverrideLevel(clinicianID)
    if err != nil {
        return nil, err
    }

    if !o.hasRequiredLevel(clinicianLevel, token.RequiredLevel) {
        return &OverrideValidation{
            Valid:  false,
            Reason: fmt.Sprintf("Insufficient override level. Required: %s, Has: %s", token.RequiredLevel, clinicianLevel),
        }, nil
    }

    // Log override
    o.auditLogger.LogOverrideAttempt(token, clinicianID, reason, true)

    return &OverrideValidation{Valid: true, Token: token}, nil
}
```

## **Structured Explanation Engine**

### **Explanation Architecture**
```go
type ExplanationEngine struct {
    templateEngine *TemplateEngine
    nlgService     *NLGService
    evidenceDB     *EvidenceDatabase
}

type Explanation struct {
    Level        ExplanationLevel       `json:"level"`
    Summary      string                `json:"summary"`
    Details      []ExplanationDetail   `json:"details"`
    Confidence   float64               `json:"confidence"`
    Evidence     []Evidence            `json:"evidence"`
    Visuals      []ExplanationVisual   `json:"visuals,omitempty"`
    Actionable   []ActionableGuidance  `json:"actionable"`
}

type ExplanationDetail struct {
    Category    string  `json:"category"`
    Severity    string  `json:"severity"`
    Description string  `json:"description"`
    Clinical    string  `json:"clinical_rationale"`
    Confidence  float64 `json:"confidence"`
}

type ActionableGuidance struct {
    Action      string   `json:"action"`
    Priority    string   `json:"priority"`
    Steps       []string `json:"steps"`
    Monitoring  []string `json:"monitoring"`
}

func (e *ExplanationEngine) GenerateExplanation(req *SafetyRequest, results []EngineResult) (*Explanation, error) {
    // Group results by concern type
    groupedResults := e.groupByConcern(results)

    // Generate summary
    summary := e.generateSummary(groupedResults)

    // Build detailed explanations
    details := e.buildDetailedExplanations(groupedResults)

    // Add evidence
    evidence := e.gatherEvidence(groupedResults)

    // Generate actionable guidance
    actionable := e.generateActionableGuidance(groupedResults, req)

    // Calculate overall confidence
    confidence := e.calculateOverallConfidence(results)

    return &Explanation{
        Level:      ExplanationLevelDetailed,
        Summary:    summary,
        Details:    details,
        Confidence: confidence,
        Evidence:   evidence,
        Actionable: actionable,
    }, nil
}

func (e *ExplanationEngine) generateSummary(groupedResults map[string][]EngineResult) string {
    var summaryParts []string

    for category, results := range groupedResults {
        if len(results) == 0 {
            continue
        }

        // Count by severity
        severityCounts := make(map[string]int)
        for _, result := range results {
            severityCounts[result.Severity]++
        }

        // Generate category summary
        categoryPart := e.generateCategorySummary(category, severityCounts)
        summaryParts = append(summaryParts, categoryPart)
    }

    return strings.Join(summaryParts, ". ")
}
```

## **Performance Optimization Strategies**

### **Multi-Level Caching Architecture**
```go
type CacheManager struct {
    l1Cache *sync.Map          // In-memory, ultra-fast
    l2Cache *RedisClient       // Redis, fast
    l3Cache *DatabaseClient    // Database, persistent
}

func (c *CacheManager) Get(key string) (interface{}, bool) {
    // L1 Cache (in-memory)
    if value, ok := c.l1Cache.Load(key); ok {
        return value, true
    }

    // L2 Cache (Redis)
    if value, err := c.l2Cache.Get(key); err == nil {
        // Promote to L1
        c.l1Cache.Store(key, value)
        return value, true
    }

    // L3 Cache (Database)
    if value, err := c.l3Cache.Get(key); err == nil {
        // Promote to L2 and L1
        c.l2Cache.Set(key, value, 5*time.Minute)
        c.l1Cache.Store(key, value)
        return value, true
    }

    return nil, false
}

func (c *CacheManager) Set(key string, value interface{}, ttl time.Duration) {
    // Store in all levels
    c.l1Cache.Store(key, value)
    c.l2Cache.Set(key, value, ttl)
    c.l3Cache.Set(key, value, ttl)
}
```

### **Request Batching and Optimization**
```go
type RequestBatcher struct {
    batchSize    int
    batchTimeout time.Duration
    pending      map[string][]*SafetyRequest
    mutex        sync.Mutex
}

func (b *RequestBatcher) BatchRequest(req *SafetyRequest) <-chan *SafetyResponse {
    b.mutex.Lock()
    defer b.mutex.Unlock()

    // Group by patient for batching efficiency
    patientKey := req.PatientID

    responseChan := make(chan *SafetyResponse, 1)

    // Add to batch
    if b.pending[patientKey] == nil {
        b.pending[patientKey] = make([]*SafetyRequest, 0)
    }

    req.ResponseChan = responseChan
    b.pending[patientKey] = append(b.pending[patientKey], req)

    // Check if batch is ready
    if len(b.pending[patientKey]) >= b.batchSize {
        go b.processBatch(patientKey)
    } else {
        // Set timeout for partial batch
        go b.timeoutBatch(patientKey)
    }

    return responseChan
}
```

## **Comprehensive Observability**

### **Structured Audit Logging**
```go
type AuditLogger struct {
    logger     *zap.Logger
    asyncQueue chan *AuditEvent
    storage    AuditStorage
}

type AuditEvent struct {
    EventID       string                 `json:"event_id"`
    Timestamp     time.Time              `json:"timestamp"`
    EventType     string                 `json:"event_type"`
    RequestID     string                 `json:"request_id"`
    PatientID     string                 `json:"patient_id_hash"` // Hashed for privacy
    ClinicianID   string                 `json:"clinician_id_hash"`
    ActionType    string                 `json:"action_type"`
    Decision      string                 `json:"decision"`
    Duration      time.Duration          `json:"duration_ms"`
    EngineResults []EngineAuditResult    `json:"engine_results"`
    Context       map[string]interface{} `json:"context"`
    Compliance    ComplianceInfo         `json:"compliance"`
}

func (a *AuditLogger) LogSafetyDecision(req *SafetyRequest, resp *SafetyResponse, duration time.Duration) {
    event := &AuditEvent{
        EventID:     generateUUID(),
        Timestamp:   time.Now(),
        EventType:   "safety_decision",
        RequestID:   req.RequestID,
        PatientID:   hashPatientID(req.PatientID),
        ActionType:  req.ActionType,
        Decision:    string(resp.Status),
        Duration:    duration,
        EngineResults: a.convertEngineResults(resp.EngineResults),
        Context: map[string]interface{}{
            "medication_count": len(req.MedicationIDs),
            "condition_count":  len(req.ConditionIDs),
            "risk_score":      resp.RiskScore,
        },
        Compliance: ComplianceInfo{
            HIPAA:    true,
            SOX:      false,
            GDPR:     true,
            Retention: "7_years",
        },
    }

    // Async logging to avoid blocking
    select {
    case a.asyncQueue <- event:
    default:
        // Queue full - log synchronously as fallback
        a.logger.Error("Audit queue full, logging synchronously", zap.Any("event", event))
    }
}
```

### **Real-time Metrics Dashboard**
```go
type MetricsCollector struct {
    registry prometheus.Registerer

    // Request metrics
    requestsTotal     *prometheus.CounterVec
    requestDuration   *prometheus.HistogramVec
    activeRequests    prometheus.Gauge

    // Engine metrics
    engineHealth      *prometheus.GaugeVec
    engineLatency     *prometheus.HistogramVec
    engineErrors      *prometheus.CounterVec

    // Business metrics
    unsafeDecisions   *prometheus.CounterVec
    overrideTokens    *prometheus.CounterVec
    contextCacheHits  *prometheus.CounterVec
}

func (m *MetricsCollector) RecordRequest(status string, duration time.Duration) {
    m.requestsTotal.WithLabelValues(status).Inc()
    m.requestDuration.WithLabelValues(status).Observe(duration.Seconds())
}

func (m *MetricsCollector) RecordEngineResult(engineID string, status string, duration time.Duration) {
    m.engineLatency.WithLabelValues(engineID, status).Observe(duration.Seconds())

    if status == "error" {
        m.engineErrors.WithLabelValues(engineID).Inc()
    }
}
```

## **Integration Strategy with Existing Services**

### **Migration Path from Current CAE (Refined)**
```
Phase 1: Dark Launch Parallel Deployment
├── Deploy Safety Gateway Platform alongside existing CAE
├── Extract CAE reasoning logic as first in-process engine
├── Route 100% of traffic through SGP in "shadow mode"
├── Log SGP decisions without affecting live responses
├── Compare SGP vs CAE decisions for validation
└── Achieve 99.9% decision consistency before proceeding

Phase 2: Gradual Traffic Migration
├── Route 10% of live traffic through SGP
├── Monitor performance metrics (latency, accuracy)
├── Increase to 50% traffic if metrics are stable
├── Add Allergy Engine and Constraint Validator (Tier 1)
└── Validate Tier-based failure handling

Phase 3: Full Migration & Engine Expansion
├── Route 100% of traffic through SGP
├── Decommission old CAE orchestration logic
├── Add Tier 2 engines (Protocol, GraphDB Asserter)
├── Implement engine-specific optimizations
└── Achieve sub-150ms average response time

Phase 4: Advanced Features Rollout
├── Deploy Override Token system
├── Implement structured explanation engine
├── Add comprehensive observability dashboard
├── Enable real-time engine health monitoring
└── Full production readiness certification
```

**Dark Launch Benefits:**
- **Zero Risk**: No impact on live clinical decisions
- **Full Validation**: Compare every decision at production scale
- **Performance Baseline**: Establish latency and throughput metrics
- **Confidence Building**: Prove system reliability before go-live

### **Service Communication Patterns**
```go
// GraphQL Federation Integration
type SafetyGatewayResolver struct {
    gateway *SafetyGatewayClient
}

func (r *SafetyGatewayResolver) ValidateMedicationOrder(ctx context.Context, args struct {
    PatientID     string   `json:"patientId"`
    MedicationIDs []string `json:"medicationIds"`
    Priority      string   `json:"priority"`
}) (*SafetyValidationResult, error) {

    request := &SafetyRequest{
        RequestID:     generateRequestID(),
        PatientID:     args.PatientID,
        MedicationIDs: args.MedicationIDs,
        ActionType:    "medication_order",
        Priority:      args.Priority,
        Source:        "graphql_federation",
    }

    response, err := r.gateway.ValidateRequest(ctx, request)
    if err != nil {
        return nil, err
    }

    return &SafetyValidationResult{
        Status:      response.Status,
        RiskScore:   response.RiskScore,
        Explanation: response.Explanation,
        OverrideToken: response.OverrideToken,
    }, nil
}

// gRPC Service Integration
func (s *MedicationService) PrescribeMedication(ctx context.Context, req *PrescriptionRequest) (*PrescriptionResponse, error) {
    // Validate with Safety Gateway before processing
    safetyReq := &SafetyRequest{
        PatientID:     req.PatientID,
        MedicationIDs: []string{req.MedicationID},
        ActionType:    "prescription",
        Priority:      "high",
    }

    safetyResp, err := s.safetyGateway.ValidateRequest(ctx, safetyReq)
    if err != nil {
        return nil, fmt.Errorf("safety validation failed: %w", err)
    }

    if safetyResp.Status == "UNSAFE" {
        return &PrescriptionResponse{
            Status:        "REQUIRES_OVERRIDE",
            OverrideToken: safetyResp.OverrideToken,
            Explanation:   safetyResp.Explanation,
        }, nil
    }

    // Proceed with prescription
    return s.processPrescription(req)
}
```

## **Comprehensive Testing Strategy**

### **Chaos Engineering Test Suite**
```go
type ChaosTestSuite struct {
    gateway     *SafetyGatewayPlatform
    engines     []*MockEngine
    loadGen     *LoadGenerator
    monitor     *TestMonitor
}

func (c *ChaosTestSuite) TestEngineFailures() {
    tests := []ChaosTest{
        {
            Name: "Random Engine Timeouts",
            Scenario: func() {
                // Randomly timeout 30% of engine calls
                for _, engine := range c.engines {
                    if rand.Float32() < 0.3 {
                        engine.SetTimeout(0) // Immediate timeout
                    }
                }
            },
            Assertions: []Assertion{
                AssertResponseTime(200 * time.Millisecond),
                AssertNoServiceCrash(),
                AssertFailClosedBehavior(),
            },
        },
        {
            Name: "Memory Pressure",
            Scenario: func() {
                // Consume 90% of available memory
                c.loadGen.GenerateMemoryPressure(0.9)
            },
            Assertions: []Assertion{
                AssertGracefulDegradation(),
                AssertNoMemoryLeaks(),
                AssertServiceRecovery(),
            },
        },
        {
            Name: "Network Partitions",
            Scenario: func() {
                // Simulate network partition to 50% of engines
                c.simulateNetworkPartition(0.5)
            },
            Assertions: []Assertion{
                AssertFallbackBehavior(),
                AssertPartialResults(),
                AssertAuditIntegrity(),
            },
        },
    }

    for _, test := range tests {
        c.runChaosTest(test)
    }
}

func (c *ChaosTestSuite) TestPerformanceUnderLoad() {
    loadProfiles := []LoadProfile{
        {Name: "Baseline", RPS: 100, Duration: 5 * time.Minute},
        {Name: "Peak", RPS: 1000, Duration: 2 * time.Minute},
        {Name: "Stress", RPS: 2000, Duration: 1 * time.Minute},
        {Name: "Soak", RPS: 200, Duration: 30 * time.Minute},
    }

    for _, profile := range loadProfiles {
        results := c.loadGen.RunLoadTest(profile)

        // Assertions
        assert.True(c.t, results.P99Latency < 200*time.Millisecond)
        assert.True(c.t, results.ErrorRate < 0.01) // <1% error rate
        assert.True(c.t, results.MemoryGrowth < 0.1) // <10% memory growth
    }
}
```

### **Clinical Workflow Validation**
```go
type ClinicalWorkflowTest struct {
    gateway    *SafetyGatewayPlatform
    fhirClient *FHIRTestClient
    scenarios  []ClinicalScenario
}

type ClinicalScenario struct {
    Name        string
    PatientData *TestPatient
    Action      ClinicalAction
    Expected    ExpectedOutcome
}

func (c *ClinicalWorkflowTest) TestHighRiskScenarios() {
    scenarios := []ClinicalScenario{
        {
            Name: "Warfarin + NSAIDs Interaction",
            PatientData: &TestPatient{
                Age: 75,
                CurrentMeds: []string{"warfarin", "aspirin"},
                Conditions: []string{"atrial_fibrillation", "hypertension"},
                Allergies: []string{},
            },
            Action: ClinicalAction{
                Type: "prescribe",
                Medication: "ibuprofen",
                Dose: "400mg",
            },
            Expected: ExpectedOutcome{
                Status: "UNSAFE",
                Violations: []string{"major_drug_interaction"},
                RequiresOverride: true,
                OverrideLevel: "attending",
            },
        },
        {
            Name: "Penicillin Allergy Check",
            PatientData: &TestPatient{
                Age: 45,
                CurrentMeds: []string{},
                Conditions: []string{"pneumonia"},
                Allergies: []string{"penicillin"},
            },
            Action: ClinicalAction{
                Type: "prescribe",
                Medication: "amoxicillin",
                Dose: "500mg",
            },
            Expected: ExpectedOutcome{
                Status: "UNSAFE",
                Violations: []string{"allergy_contraindication"},
                RequiresOverride: true,
                OverrideLevel: "pharmacist",
            },
        },
    }

    for _, scenario := range scenarios {
        c.runClinicalScenario(scenario)
    }
}
```

## **Deployment and Operations**

### **Docker Configuration**
```dockerfile
# Dockerfile for Safety Gateway Platform
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o safety-gateway ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/safety-gateway .
COPY --from=builder /app/config.yaml .

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ./safety-gateway health-check || exit 1

EXPOSE 8030
CMD ["./safety-gateway", "serve"]
```

### **Kubernetes Deployment**
```yaml
# k8s/safety-gateway-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: safety-gateway-platform
  labels:
    app: safety-gateway-platform
spec:
  replicas: 3
  selector:
    matchLabels:
      app: safety-gateway-platform
  template:
    metadata:
      labels:
        app: safety-gateway-platform
    spec:
      containers:
      - name: safety-gateway
        image: clinical-hub/safety-gateway:latest
        ports:
        - containerPort: 8030
        env:
        - name: REDIS_URL
          value: "redis://redis-service:6379"
        - name: POSTGRES_URL
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: postgres-url
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
            port: 8030
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8030
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: safety-gateway-service
spec:
  selector:
    app: safety-gateway-platform
  ports:
  - protocol: TCP
    port: 8030
    targetPort: 8030
  type: ClusterIP
```

### **Monitoring and Alerting**
```yaml
# monitoring/alerts.yaml
groups:
- name: safety-gateway-alerts
  rules:
  - alert: SafetyGatewayHighLatency
    expr: histogram_quantile(0.95, rate(safety_gateway_request_duration_seconds_bucket[5m])) > 0.2
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "Safety Gateway high latency"
      description: "95th percentile latency is {{ $value }}s"

  - alert: SafetyGatewayHighErrorRate
    expr: rate(safety_gateway_requests_total{status="error"}[5m]) / rate(safety_gateway_requests_total[5m]) > 0.01
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Safety Gateway high error rate"
      description: "Error rate is {{ $value | humanizePercentage }}"

  - alert: SafetyGatewayEngineDown
    expr: safety_gateway_engine_health{status="healthy"} == 0
    for: 30s
    labels:
      severity: critical
    annotations:
      summary: "Safety Gateway engine is down"
      description: "Engine {{ $labels.engine_id }} is not healthy"

  - alert: SafetyGatewayUnsafeDecisionSpike
    expr: increase(safety_gateway_decisions_total{status="unsafe"}[5m]) > 10
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "Spike in unsafe decisions"
      description: "{{ $value }} unsafe decisions in the last 5 minutes"
```

## **Security and Compliance**

### **HIPAA Compliance Measures**
```go
type ComplianceManager struct {
    encryptionService *EncryptionService
    auditLogger      *ComplianceAuditLogger
    accessControl    *AccessControlManager
}

func (c *ComplianceManager) ProcessRequest(req *SafetyRequest) (*SafetyRequest, error) {
    // Encrypt PII fields
    encryptedReq := &SafetyRequest{
        RequestID:     req.RequestID,
        PatientID:     c.encryptionService.EncryptPII(req.PatientID),
        MedicationIDs: req.MedicationIDs, // Non-PII
        ActionType:    req.ActionType,
        Priority:      req.Priority,
        Timestamp:     req.Timestamp,
    }

    // Log access
    c.auditLogger.LogPIIAccess(req.PatientID, req.ClinicianID, "safety_validation")

    return encryptedReq, nil
}

func (c *ComplianceManager) SanitizeLogsForStorage(event *AuditEvent) *AuditEvent {
    sanitized := *event

    // Hash patient ID
    sanitized.PatientID = c.hashPII(event.PatientID)

    // Remove sensitive context
    sanitized.Context = c.removeSensitiveFields(event.Context)

    // Ensure retention policy
    sanitized.RetentionPolicy = "7_years_then_delete"

    return &sanitized
}
```

This comprehensive implementation plan provides a complete blueprint for extracting the orchestration logic into a dedicated Safety Gateway Platform microservice, ensuring high performance, reliability, clinical safety standards, and regulatory compliance.
