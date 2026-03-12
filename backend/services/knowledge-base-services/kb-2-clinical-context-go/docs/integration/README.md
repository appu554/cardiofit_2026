# Integration Documentation

## Overview

This directory contains comprehensive integration documentation for the KB-2 Clinical Context service, providing guidance for integrating with Apollo Federation, Evidence Envelope audit systems, knowledge base services, and Flow2 orchestrator within the Clinical Synthesis Hub ecosystem.

## Integration Architecture

The KB-2 Clinical Context service operates as a **core intelligence hub** within the microservices architecture:

```
Clinical Synthesis Hub Integration Landscape
├── Frontend (Angular) 
│   └── Apollo GraphQL Client → Apollo Federation Gateway
├── API Gateway Layer
│   ├── Apollo Federation Server (GraphQL Gateway)
│   ├── REST API Gateway (Load Balancer)
│   └── Authentication Service Integration
├── Core Services Layer
│   ├── KB-2 Clinical Context (This Service) ←→ Flow2 Orchestrator
│   ├── KB-Drug-Rules Service ←→ KB-2 Context Assembly
│   ├── KB-Guideline-Evidence ←→ KB-2 Risk Assessment
│   └── Safety Gateway Platform ←→ KB-2 Safety Validation
├── Supporting Services Layer
│   ├── Patient Service → KB-2 Patient Data
│   ├── Observation Service → KB-2 Clinical Data
│   ├── Clinical Reasoning Service ←→ KB-2 Intelligence
│   └── Evidence Envelope Service ← KB-2 Audit Trail
└── Data Layer
    ├── MongoDB (Patient/Clinical Data)
    ├── Redis (Caching/Sessions)
    ├── Neo4j (Knowledge Graph)
    └── PostgreSQL (Knowledge Base Rules)
```

## Integration Patterns

### 1. Apollo Federation Integration
- **Schema Federation**: Type extensions and resolvers
- **Query Optimization**: Intelligent batching and caching
- **Type Safety**: Strong typing with GraphQL schemas
- **Real-time Updates**: Subscription support for clinical events

### 2. Evidence Envelope Audit Integration
- **Audit Trail Generation**: Complete decision audit logging
- **Regulatory Compliance**: HIPAA and clinical decision support tracking
- **Evidence Chain**: Clinical reasoning evidence documentation
- **Performance Monitoring**: Decision quality and outcome tracking

### 3. Knowledge Base Service Coordination
- **Rule Synchronization**: Coordinated rule updates across KB services
- **Data Consistency**: Shared clinical knowledge and reference data
- **Performance Optimization**: Distributed caching and query coordination
- **Version Management**: Coordinated versioning and deployment strategies

### 4. Flow2 Orchestrator Integration
- **Clinical Workflow Orchestration**: Multi-step clinical decision processes
- **Context Assembly**: Comprehensive patient intelligence aggregation
- **Decision Support Delivery**: Real-time clinical decision support
- **Outcome Tracking**: Clinical decision effectiveness monitoring

## Document Organization

### [Apollo Federation Guide](./apollo-federation.md)
Comprehensive integration guide for Apollo Federation including schema design, resolvers implementation, and performance optimization.

**Contents:**
- GraphQL schema federation and type extensions
- Resolver implementation with batching optimization
- Caching strategies for federated queries
- Real-time subscriptions and updates
- Federation gateway configuration
- Performance monitoring and optimization

### [Evidence Envelope Integration](./evidence-envelope.md)
Complete integration framework for Evidence Envelope audit system ensuring regulatory compliance and clinical decision tracking.

**Contents:**
- Audit trail generation and formatting
- Clinical decision evidence documentation
- Regulatory compliance tracking
- Performance impact assessment
- Audit query and reporting interfaces
- Data retention and archival procedures

### [Knowledge Base Coordination](./knowledge-base-coordination.md)
Framework for coordinating with other knowledge base services including rule synchronization, data consistency, and performance optimization.

**Contents:**
- Inter-service communication patterns
- Rule synchronization and version management
- Shared data consistency strategies
- Distributed caching coordination
- Performance optimization techniques
- Error handling and recovery procedures

### [Flow2 Orchestrator Integration](./flow2-integration.md)
Detailed integration guide for Flow2 orchestrator including clinical workflow coordination, context assembly, and decision support delivery.

**Contents:**
- Clinical workflow integration patterns
- Context assembly and aggregation strategies
- Real-time decision support delivery
- Performance optimization and caching
- Error handling and fallback procedures
- Monitoring and quality assurance

### [Clinical Workflow Integration](./clinical-workflow.md)
Comprehensive guide for integrating KB-2 services into clinical workflows including EHR integration, CDS Hooks, and SMART on FHIR applications.

**Contents:**
- EHR integration patterns and standards
- CDS Hooks implementation and optimization
- SMART on FHIR application integration
- HL7 FHIR resource mapping and transformation
- Clinical decision point integration
- Workflow performance optimization

## Integration Endpoints

### Apollo Federation Schema Extensions

#### Patient Type Extension
```graphql
extend type Patient @key(fields: "id") {
  id: ID! @external
  clinicalContext: ClinicalContext
  phenotypes(categories: [PhenotypeCategory!]): [Phenotype!]!
  riskAssessments(categories: [RiskCategory!]): [RiskAssessment!]!
  treatmentPreferences(conditions: [String!]): [TreatmentPreference!]!
}
```

#### Clinical Context Type
```graphql
type ClinicalContext {
  patientId: ID!
  timestamp: DateTime!
  phenotypes: [Phenotype!]!
  riskAssessments: [RiskAssessment!]!
  treatmentPreferences: [TreatmentPreference!]!
  processingMetadata: ProcessingMetadata!
}

type Phenotype {
  id: String!
  name: String!
  category: PhenotypeCategory!
  positive: Boolean!
  confidence: Float!
  severity: Severity!
  contributingFactors: [String!]!
  ruleEvaluation: RuleEvaluation
}

type RiskAssessment {
  category: RiskCategory!
  score: Float!
  riskLevel: RiskLevel!
  tenYearRisk: Float
  factors: [RiskFactor!]!
  model: RiskModel!
}
```

### REST API Integration Points

#### Core Clinical Intelligence Endpoints
```http
POST /v1/phenotypes/evaluate
POST /v1/phenotypes/explain  
POST /v1/risk/assess
POST /v1/treatment/preferences
POST /v1/context/assemble
```

#### Integration Support Endpoints
```http
GET /v1/integration/schema          # GraphQL schema for federation
GET /v1/integration/health          # Integration health status
POST /v1/integration/validate       # Integration configuration validation
GET /v1/integration/metrics         # Integration performance metrics
```

### Service Discovery and Health Checks

#### Service Registration
```yaml
service_registration:
  name: "kb2-clinical-context"
  version: "1.0.0"
  tags: ["clinical", "phenotyping", "risk-assessment", "treatment-preferences"]
  
  endpoints:
    graphql: "http://kb2-service:8088/graphql"
    rest: "http://kb2-service:8088/v1"
    health: "http://kb2-service:8088/health"
    metrics: "http://kb2-service:8088/metrics"
  
  capabilities:
    - "phenotype_evaluation"
    - "risk_assessment"  
    - "treatment_recommendations"
    - "clinical_context_assembly"
  
  dependencies:
    - "mongodb"
    - "redis"
    - "patient-service"
    - "observation-service"
```

#### Health Check Integration
```json
{
  "status": "healthy",
  "timestamp": "2025-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "7d 14h 23m",
  "dependencies": {
    "mongodb": {
      "status": "healthy",
      "response_time_ms": 5,
      "connection_pool": "8/20 active"
    },
    "redis": {
      "status": "healthy", 
      "response_time_ms": 1,
      "memory_usage": "512MB / 2GB"
    },
    "patient_service": {
      "status": "healthy",
      "response_time_ms": 15,
      "last_check": "2025-01-15T10:29:45Z"
    }
  },
  "performance_metrics": {
    "requests_per_second": 1250,
    "average_response_time_ms": 35,
    "error_rate": 0.001,
    "cache_hit_rate": 0.96
  }
}
```

## Integration Patterns and Best Practices

### 1. Asynchronous Communication

#### Event-Driven Integration
```yaml
event_patterns:
  clinical_context_updated:
    trigger: "Context assembly completion"
    payload: "Patient ID, context data, metadata"
    consumers: ["flow2-orchestrator", "evidence-envelope", "clinical-reasoning"]
    
  phenotype_evaluation_completed:
    trigger: "Phenotype evaluation completion"
    payload: "Patient ID, phenotypes, confidence scores"
    consumers: ["safety-gateway", "evidence-envelope"]
    
  risk_assessment_completed:
    trigger: "Risk assessment completion"
    payload: "Patient ID, risk scores, risk levels"
    consumers: ["flow2-orchestrator", "clinical-reasoning"]
```

#### Message Queue Integration
```go
// Example event publishing
type ContextUpdatedEvent struct {
    PatientID string `json:"patient_id"`
    Context   ClinicalContext `json:"context"`
    Timestamp time.Time `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata"`
}

func (s *ContextService) publishContextUpdated(ctx context.Context, patientID string, context ClinicalContext) error {
    event := ContextUpdatedEvent{
        PatientID: patientID,
        Context:   context,
        Timestamp: time.Now(),
        Metadata: map[string]interface{}{
            "service": "kb2-clinical-context",
            "version": s.version,
        },
    }
    
    return s.eventBus.Publish(ctx, "clinical_context_updated", event)
}
```

### 2. Synchronous API Integration

#### Request/Response Patterns
```go
// Flow2 Integration Client
type Flow2Client struct {
    baseURL    string
    httpClient *http.Client
    auth       AuthProvider
}

func (c *Flow2Client) RequestContextAssembly(ctx context.Context, req *ContextRequest) (*ContextResponse, error) {
    // Prepare request with authentication
    httpReq, err := c.prepareRequest(ctx, "POST", "/v1/context/request", req)
    if err != nil {
        return nil, err
    }
    
    // Execute request with timeout and retries
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    // Process response
    var result ContextResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}
```

#### Circuit Breaker Pattern
```go
// Circuit breaker for external service calls
type ServiceIntegration struct {
    circuitBreaker *gobreaker.CircuitBreaker
    client         *http.Client
}

func (s *ServiceIntegration) CallExternalService(ctx context.Context, req *Request) (*Response, error) {
    result, err := s.circuitBreaker.Execute(func() (interface{}, error) {
        return s.client.Do(req)
    })
    
    if err != nil {
        // Handle circuit breaker errors
        return nil, fmt.Errorf("service call failed: %w", err)
    }
    
    return result.(*Response), nil
}
```

### 3. Data Consistency Patterns

#### Eventual Consistency
```go
// Saga pattern for distributed transactions
type ClinicalContextSaga struct {
    steps []SagaStep
    compensations []CompensationStep
}

func (s *ClinicalContextSaga) Execute(ctx context.Context, data *ContextData) error {
    completed := 0
    
    // Execute saga steps
    for i, step := range s.steps {
        if err := step.Execute(ctx, data); err != nil {
            // Compensate completed steps
            s.compensate(ctx, completed)
            return fmt.Errorf("saga step %d failed: %w", i, err)
        }
        completed++
    }
    
    return nil
}
```

#### Optimistic Locking
```go
// Version-based optimistic locking
type ClinicalContext struct {
    PatientID string `json:"patient_id" bson:"patient_id"`
    Version   int64  `json:"version" bson:"version"`
    Data      ContextData `json:"data" bson:"data"`
    UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func (r *ContextRepository) UpdateContext(ctx context.Context, patientID string, updater func(*ClinicalContext) error) error {
    session, err := r.client.StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)
    
    return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
        // Read current version
        var current ClinicalContext
        err := r.collection.FindOne(sc, bson.M{"patient_id": patientID}).Decode(&current)
        if err != nil {
            return err
        }
        
        // Apply updates
        updated := current
        if err := updater(&updated); err != nil {
            return err
        }
        updated.Version++
        updated.UpdatedAt = time.Now()
        
        // Conditional update with version check
        result, err := r.collection.ReplaceOne(sc, 
            bson.M{"patient_id": patientID, "version": current.Version},
            updated)
        if err != nil {
            return err
        }
        
        if result.ModifiedCount == 0 {
            return errors.New("concurrent modification detected")
        }
        
        return nil
    })
}
```

## Performance Optimization

### 1. Caching Strategies

#### Distributed Caching
```go
// Multi-level caching strategy
type CacheManager struct {
    local  *cache.Cache       // In-memory L1 cache
    redis  *redis.Client      // Distributed L2 cache
    config CacheConfig
}

func (c *CacheManager) GetClinicalContext(ctx context.Context, patientID string) (*ClinicalContext, error) {
    key := fmt.Sprintf("clinical_context:%s", patientID)
    
    // L1 cache (local memory)
    if value, found := c.local.Get(key); found {
        return value.(*ClinicalContext), nil
    }
    
    // L2 cache (Redis)
    result, err := c.redis.Get(ctx, key).Result()
    if err == nil {
        var context ClinicalContext
        if err := json.Unmarshal([]byte(result), &context); err == nil {
            // Populate L1 cache
            c.local.Set(key, &context, c.config.LocalTTL)
            return &context, nil
        }
    }
    
    // Cache miss - fetch from source
    return nil, cache.ErrCacheMiss
}
```

#### Cache Invalidation
```go
// Event-driven cache invalidation
func (c *CacheManager) InvalidatePatientContext(ctx context.Context, patientID string) error {
    key := fmt.Sprintf("clinical_context:%s", patientID)
    
    // Invalidate local cache
    c.local.Delete(key)
    
    // Invalidate distributed cache
    if err := c.redis.Del(ctx, key).Err(); err != nil {
        return fmt.Errorf("failed to invalidate Redis cache: %w", err)
    }
    
    // Notify other service instances
    event := CacheInvalidationEvent{
        PatientID: patientID,
        Timestamp: time.Now(),
    }
    
    return c.eventBus.Publish(ctx, "cache_invalidation", event)
}
```

### 2. Request Batching

#### GraphQL DataLoader Pattern
```go
// DataLoader for batching database requests
type PhenotypeLoader struct {
    batch     []*PhenotypeBatchItem
    batchSize int
    wait      time.Duration
    mu        sync.Mutex
}

func (l *PhenotypeLoader) Load(ctx context.Context, patientID string) (*[]Phenotype, error) {
    return l.loadThunk(patientID)()
}

func (l *PhenotypeLoader) loadThunk(patientID string) func() (*[]Phenotype, error) {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    // Add to batch
    item := &PhenotypeBatchItem{
        PatientID: patientID,
        Result:    make(chan *PhenotypeResult, 1),
    }
    l.batch = append(l.batch, item)
    
    // Trigger batch processing if batch is full
    if len(l.batch) >= l.batchSize {
        go l.processBatch()
    }
    
    return func() (*[]Phenotype, error) {
        result := <-item.Result
        return result.Phenotypes, result.Error
    }
}
```

### 3. Database Optimization

#### Connection Pooling
```go
// MongoDB connection pool configuration
func NewMongoClient(uri string) (*mongo.Client, error) {
    clientOptions := options.Client().
        ApplyURI(uri).
        SetMaxPoolSize(20).
        SetMinPoolSize(5).
        SetMaxConnIdleTime(30 * time.Second).
        SetServerSelectionTimeout(5 * time.Second)
    
    client, err := mongo.Connect(context.Background(), clientOptions)
    if err != nil {
        return nil, err
    }
    
    // Test connection
    if err := client.Ping(context.Background(), nil); err != nil {
        return nil, err
    }
    
    return client, nil
}
```

#### Query Optimization
```go
// Optimized MongoDB queries with proper indexing
func (r *PhenotypeRepository) GetPhenotypesForPatients(ctx context.Context, patientIDs []string) (map[string][]Phenotype, error) {
    // Use $in operator for batch lookup
    filter := bson.M{"patient_id": bson.M{"$in": patientIDs}}
    
    // Project only needed fields
    projection := bson.M{
        "patient_id": 1,
        "phenotypes": 1,
        "timestamp": 1,
    }
    
    cursor, err := r.collection.Find(ctx, filter, options.Find().SetProjection(projection))
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    // Group results by patient ID
    results := make(map[string][]Phenotype)
    for cursor.Next(ctx) {
        var doc PhenotypeDocument
        if err := cursor.Decode(&doc); err != nil {
            continue
        }
        results[doc.PatientID] = doc.Phenotypes
    }
    
    return results, nil
}
```

## Security Integration

### 1. Authentication and Authorization

#### JWT Token Validation
```go
// JWT middleware for API authentication
func JWTMiddleware(secret []byte) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        tokenString := extractToken(c.GetHeader("Authorization"))
        if tokenString == "" {
            c.JSON(401, gin.H{"error": "Authorization token required"})
            c.Abort()
            return
        }
        
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return secret, nil
        })
        
        if err != nil || !token.Valid {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
        
        // Extract claims
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            c.Set("user_id", claims["user_id"])
            c.Set("roles", claims["roles"])
            c.Set("permissions", claims["permissions"])
        }
        
        c.Next()
    })
}
```

#### Role-Based Access Control
```go
// RBAC authorization middleware
func RequirePermission(permission string) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        permissions, exists := c.Get("permissions")
        if !exists {
            c.JSON(403, gin.H{"error": "No permissions found"})
            c.Abort()
            return
        }
        
        permissionList, ok := permissions.([]interface{})
        if !ok {
            c.JSON(403, gin.H{"error": "Invalid permissions format"})
            c.Abort()
            return
        }
        
        // Check if required permission exists
        hasPermission := false
        for _, p := range permissionList {
            if p.(string) == permission {
                hasPermission = true
                break
            }
        }
        
        if !hasPermission {
            c.JSON(403, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }
        
        c.Next()
    })
}
```

### 2. Data Encryption

#### Encryption in Transit
```go
// TLS configuration for secure communication
func NewTLSConfig() *tls.Config {
    return &tls.Config{
        MinVersion:               tls.VersionTLS12,
        CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
        PreferServerCipherSuites: true,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    }
}
```

#### Field-Level Encryption
```go
// Sensitive data encryption
type EncryptionService struct {
    key []byte
}

func (e *EncryptionService) EncryptPII(data string) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

## Monitoring and Observability

### 1. Distributed Tracing

#### OpenTelemetry Integration
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (s *PhenotypeService) EvaluatePhenotypes(ctx context.Context, request *EvaluationRequest) (*EvaluationResponse, error) {
    tracer := otel.Tracer("kb2-clinical-context")
    ctx, span := tracer.Start(ctx, "phenotype_evaluation")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(
        attribute.String("patient.id", request.PatientID),
        attribute.Int("phenotypes.count", len(request.Phenotypes)),
    )
    
    // Nested span for database operation
    ctx, dbSpan := tracer.Start(ctx, "database_query")
    patientData, err := s.patientRepo.GetPatient(ctx, request.PatientID)
    dbSpan.End()
    
    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    // Continue with phenotype evaluation...
    return s.evaluatePhenotypesImpl(ctx, request, patientData)
}
```

### 2. Metrics Collection

#### Prometheus Metrics
```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    phenotypeEvaluations = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kb2_phenotype_evaluations_total",
            Help: "Total number of phenotype evaluations",
        },
        []string{"phenotype", "result", "confidence_level"},
    )
    
    evaluationDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "kb2_evaluation_duration_seconds",
            Help:    "Duration of phenotype evaluations",
            Buckets: prometheus.DefBuckets,
        },
        []string{"phenotype", "batch_size"},
    )
)

func (s *PhenotypeService) recordMetrics(phenotype string, result bool, confidence float64, duration time.Duration, batchSize int) {
    confidenceLevel := "low"
    if confidence > 0.7 {
        confidenceLevel = "medium"
    }
    if confidence > 0.9 {
        confidenceLevel = "high"
    }
    
    phenotypeEvaluations.WithLabelValues(
        phenotype, 
        strconv.FormatBool(result), 
        confidenceLevel,
    ).Inc()
    
    evaluationDuration.WithLabelValues(
        phenotype, 
        strconv.Itoa(batchSize),
    ).Observe(duration.Seconds())
}
```

## Error Handling and Resilience

### 1. Retry Logic

#### Exponential Backoff
```go
// Retry configuration
type RetryConfig struct {
    MaxRetries  int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
    Jitter      bool
}

func (c *ServiceClient) CallWithRetry(ctx context.Context, operation func() error) error {
    var lastErr error
    
    for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
        if err := operation(); err == nil {
            return nil
        } else {
            lastErr = err
            
            // Don't retry on certain errors
            if isNonRetryableError(err) {
                return err
            }
            
            if attempt < c.retryConfig.MaxRetries {
                delay := c.calculateDelay(attempt)
                select {
                case <-time.After(delay):
                    continue
                case <-ctx.Done():
                    return ctx.Err()
                }
            }
        }
    }
    
    return fmt.Errorf("operation failed after %d attempts: %w", c.retryConfig.MaxRetries+1, lastErr)
}
```

### 2. Graceful Degradation

#### Fallback Strategies
```go
// Fallback service implementation
type ClinicalContextService struct {
    primary   ContextProvider
    fallback  ContextProvider
    circuit   *CircuitBreaker
}

func (s *ClinicalContextService) AssembleClinicalContext(ctx context.Context, patientID string) (*ClinicalContext, error) {
    // Try primary service
    if s.circuit.State() != CircuitBreakerOpen {
        context, err := s.primary.GetContext(ctx, patientID)
        if err == nil {
            s.circuit.RecordSuccess()
            return context, nil
        }
        
        s.circuit.RecordFailure()
        
        // If circuit is now open, log the transition
        if s.circuit.State() == CircuitBreakerOpen {
            log.Warn("Circuit breaker opened for primary context service")
        }
    }
    
    // Fallback to cached or simplified context
    log.Info("Using fallback context provider")
    return s.fallback.GetContext(ctx, patientID)
}
```

---

**Integration Oversight**: Platform Architecture Team + Clinical Informatics  
**Last Updated**: 2025-01-15  
**Next Review**: 2025-04-15  
**Integration Support**: integration-support@cardiofit.health