# Flow2 Engines Integration in Medication Service V2

## Overview

This document explains how the Flow2 Go Engine and Flow2 Rust Engine work together within the Medication Service V2 architecture to implement the complete Calculate > Validate > Commit workflow for medication intelligence.

## Architecture Integration

### Engine Coordination Pattern

```
┌─────────────────────────────────────────────────────────────────┐
│                    Medication Service V2                        │
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐  │
│  │  Flow2 Go       │    │  Flow2 Rust     │    │ Knowledge   │  │
│  │  Engine         │◄──►│  Engine         │    │ Bases       │  │
│  │  (Port 8085)    │    │  (Port 8095)    │    │ (Multiple)  │  │
│  │                 │    │                 │    │             │  │
│  │ • Orchestration │    │ • Calculations  │    │ • Rules     │  │
│  │ • Context Mgmt  │    │ • Dose Safety   │    │ • Evidence  │  │
│  │ • API Gateway   │    │ • FFI Interface │    │ • Protocols │  │
│  └─────────────────┘    └─────────────────┘    └─────────────┘  │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              Main Go Service (Port 8005)                    │ │
│  │         • Recipe Resolution • Context Gateway               │ │
│  │         • 4-Phase Orchestration • External APIs            │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## 4-Phase Implementation with Flow2 Engines

### Phase 1: Ingestion & Intent Resolution
**Primary Engine**: Flow2 Go Engine
**Support**: Main Go Service

```go
// Main Go Service initiates
func (s *MedicationService) ProcessRequest(req *MedicationRequest) {
    // Phase 1: Intent Resolution
    intent := s.recipeResolver.ResolveIntent(req)

    // Delegate to Flow2 Go Engine for orchestration
    result := s.flow2GoEngine.ProcessPhases(intent)
}

// Flow2 Go Engine orchestrates
func (e *Flow2GoEngine) ProcessPhases(intent IntentManifest) {
    // Phase 1 processing in Go engine
    envelope := e.createEvidenceEnvelope(intent)

    // Continue to Phase 2...
}
```

### Phase 2: Context Assembly
**Primary Engine**: Flow2 Go Engine
**Support**: Context Gateway, Knowledge Bases

```go
func (e *Flow2GoEngine) AssembleContext(intent IntentManifest) {
    var wg sync.WaitGroup

    // Parallel context fetching
    wg.Add(3)

    // Branch 1: Patient snapshot via Context Gateway
    go func() {
        defer wg.Done()
        snapshot := e.contextGateway.CreateSnapshot(intent.RecipeID)
        e.enrichedContext.Snapshot = snapshot
    }()

    // Branch 2: Clinical knowledge via Knowledge Bases
    go func() {
        defer wg.Done()
        kbData := e.queryKnowledgeBases(intent.RequiredKBs)
        e.enrichedContext.KnowledgeData = kbData
    }()

    // Branch 3: Phenotype evaluation (in-memory)
    go func() {
        defer wg.Done()
        phenotype := e.evaluatePhenotype(intent.ClinicalData)
        e.enrichedContext.Phenotype = phenotype
    }()

    wg.Wait()
}
```

### Phase 3: Clinical Intelligence
**Orchestration**: Flow2 Go Engine
**Computation**: Flow2 Rust Engine via FFI
**Knowledge**: Knowledge Bases

#### Phase 3a: Candidate Generation (Go Engine)
```go
func (e *Flow2GoEngine) GenerateCandidates(context EnrichedContext) []Candidate {
    // Safety vetting via Knowledge Bases
    candidates := e.queryFormulary(context.TherapyOptions)

    // Parallel safety checks
    var wg sync.WaitGroup
    safetyChannel := make(chan SafetyResult, len(candidates))

    for _, candidate := range candidates {
        wg.Add(1)
        go func(c Candidate) {
            defer wg.Done()
            safety := e.performSafetyCheck(c, context.Snapshot)
            safetyChannel <- SafetyResult{Candidate: c, Safe: safety}
        }(candidate)
    }

    wg.Wait()
    close(safetyChannel)

    // Filter to safe candidates only
    var safeCandidates []Candidate
    for result := range safetyChannel {
        if result.Safe {
            safeCandidates = append(safeCandidates, result.Candidate)
        }
    }

    return safeCandidates
}
```

#### Phase 3b: Dose Calculation (Rust Engine via FFI)
```go
// Go Engine calls Rust Engine
func (e *Flow2GoEngine) CalculateDoses(candidates []Candidate) []DoseCalculation {
    var calculations []DoseCalculation

    for _, candidate := range candidates {
        // FFI call to Rust engine
        rustRequest := &RustCalculationRequest{
            Medication: candidate,
            PatientData: e.enrichedContext.Snapshot.PatientData,
            DosingRules: e.enrichedContext.KnowledgeData.DosingRules,
        }

        // Call Rust engine via HTTP/gRPC
        rustResult := e.rustEngine.CalculateDose(rustRequest)

        calculations = append(calculations, DoseCalculation{
            Candidate: candidate,
            Dose: rustResult.CalculatedDose,
            Adjustments: rustResult.Adjustments,
            SafetyLimits: rustResult.SafetyValidation,
        })
    }

    return calculations
}
```

```rust
// Rust Engine implementation
impl DoseCalculationEngine {
    pub fn calculate_dose(&self, request: &CalculationRequest) -> CalculationResult {
        // High-performance parallel processing
        let base_dose = self.calculate_base_dose(&request.medication, &request.patient);

        // Parallel adjustments
        let adjustments = rayon::par_iter([
            self.renal_adjustment(&request.patient.labs),
            self.hepatic_adjustment(&request.patient.labs),
            self.age_adjustment(request.patient.age),
            self.weight_adjustment(request.patient.weight),
        ]).collect::<Vec<_>>();

        // Apply safety constraints
        let final_dose = self.apply_safety_limits(base_dose, &adjustments);

        CalculationResult {
            calculated_dose: final_dose,
            adjustments,
            safety_validation: self.validate_constraints(&final_dose),
            confidence_score: self.calculate_confidence(&request, &final_dose),
        }
    }
}
```

#### Phase 3c: Scoring & Ranking (Go Engine)
```go
func (e *Flow2GoEngine) ScoreAndRank(calculations []DoseCalculation) []RankedProposal {
    // Multi-factor scoring
    var scoredProposals []ScoredProposal

    for _, calc := range calculations {
        scores := ProposalScores{
            GuidelineAdherence: e.scoreGuidelineAdherence(calc),
            SafetyProfile:      e.scoreSafetyProfile(calc),
            PatientFactors:     e.scorePatientFactors(calc),
            FormularyPreference: e.scoreFormularyPreference(calc),
            CostEffectiveness:  e.scoreCostEffectiveness(calc),
            AdherenceLikelihood: e.scoreAdherence(calc),
        }

        totalScore := e.calculateWeightedScore(scores)

        scoredProposals = append(scoredProposals, ScoredProposal{
            Calculation: calc,
            Scores: scores,
            TotalScore: totalScore,
        })
    }

    // Sort by total score (descending)
    sort.Slice(scoredProposals, func(i, j int) bool {
        return scoredProposals[i].TotalScore > scoredProposals[j].TotalScore
    })

    // Return top 3-5 proposals
    topCount := min(5, len(scoredProposals))
    return scoredProposals[:topCount]
}
```

### Phase 4: Proposal Generation
**Primary Engine**: Flow2 Go Engine
**Support**: Global Outbox Service

```go
func (e *Flow2GoEngine) GenerateProposals(ranked []RankedProposal) ProposalSet {
    proposalSet := ProposalSet{
        ID: generateProposalSetID(),
        SnapshotReference: e.enrichedContext.Snapshot.ID, // CRITICAL for validation
        Timestamp: time.Now(),
        EvidenceEnvelope: e.evidenceEnvelope,
    }

    for _, ranked := range ranked {
        proposal := MedicationProposal{
            Medication: ranked.Calculation.Candidate,
            Dose: ranked.Calculation.Dose,
            Rationale: e.generateRationale(ranked),
            MonitoringPlan: e.generateMonitoringPlan(ranked),
            Alternatives: e.generateAlternatives(ranked),
            Confidence: ranked.TotalScore,
        }

        proposalSet.Proposals = append(proposalSet.Proposals, proposal)
    }

    // Persist with WORM pattern
    e.persistProposalSet(proposalSet)

    // Publish events via Global Outbox
    e.outboxService.PublishProposalGenerated(proposalSet)

    return proposalSet
}
```

## Inter-Engine Communication

### Flow2 Go ↔ Flow2 Rust Communication

#### HTTP API Pattern
```go
// Go Engine HTTP client
type RustEngineClient struct {
    baseURL string
    client  *http.Client
    timeout time.Duration
}

func (c *RustEngineClient) CalculateDose(request *CalculationRequest) (*CalculationResult, error) {
    jsonBody, _ := json.Marshal(request)

    resp, err := c.client.Post(
        c.baseURL+"/api/v1/calculate-dose",
        "application/json",
        bytes.NewBuffer(jsonBody),
    )

    if err != nil {
        return nil, fmt.Errorf("rust engine call failed: %w", err)
    }

    var result CalculationResult
    json.NewDecoder(resp.Body).Decode(&result)

    return &result, nil
}
```

```rust
// Rust Engine HTTP server
#[tokio::main]
async fn main() {
    let app = Router::new()
        .route("/api/v1/calculate-dose", post(calculate_dose_handler))
        .route("/health", get(health_check));

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8095").await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

async fn calculate_dose_handler(
    Json(request): Json<CalculationRequest>
) -> Result<Json<CalculationResult>, StatusCode> {
    let engine = DoseCalculationEngine::new();

    match engine.calculate_dose(&request) {
        Ok(result) => Ok(Json(result)),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}
```

### Performance Optimization

#### Connection Pooling
```go
// Optimized client with connection pooling
func NewRustEngineClient(baseURL string) *RustEngineClient {
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     30 * time.Second,
    }

    return &RustEngineClient{
        baseURL: baseURL,
        client: &http.Client{
            Transport: transport,
            Timeout:   30 * time.Second,
        },
    }
}
```

#### Batch Processing
```go
// Batch multiple calculations for efficiency
func (c *RustEngineClient) CalculateDosesBatch(requests []CalculationRequest) ([]CalculationResult, error) {
    batchRequest := BatchCalculationRequest{
        Calculations: requests,
        ProcessingMode: "parallel",
    }

    // Single HTTP call for multiple calculations
    return c.processBatch(batchRequest)
}
```

## Error Handling & Resilience

### Circuit Breaker Pattern
```go
type CircuitBreaker struct {
    maxFailures int
    timeout     time.Duration
    state       CircuitState
    failures    int
    lastFailure time.Time
}

func (e *Flow2GoEngine) CallRustEngineWithCircuitBreaker(request *CalculationRequest) (*CalculationResult, error) {
    if e.circuitBreaker.IsOpen() {
        // Fall back to cached calculations or simplified logic
        return e.fallbackCalculation(request), nil
    }

    result, err := e.rustEngine.CalculateDose(request)

    if err != nil {
        e.circuitBreaker.RecordFailure()
        return e.fallbackCalculation(request), err
    }

    e.circuitBreaker.RecordSuccess()
    return result, nil
}
```

### Graceful Degradation
```go
func (e *Flow2GoEngine) fallbackCalculation(request *CalculationRequest) *CalculationResult {
    // Simple calculation when Rust engine unavailable
    simpleDose := e.calculateSimpleDose(request.Medication, request.PatientData)

    return &CalculationResult{
        CalculatedDose: simpleDose,
        Confidence: 0.7, // Lower confidence for fallback
        DegradedMode: true,
        Adjustments: []string{"Simplified calculation - verify manually"},
    }
}
```

## Monitoring & Observability

### Distributed Tracing
```go
func (e *Flow2GoEngine) ProcessWithTracing(ctx context.Context, request *MedicationRequest) {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("service", "flow2-go-engine"),
        attribute.String("operation", "medication-processing"),
        attribute.String("patient_id", request.PatientID),
    )

    // Phase 1
    phase1Span := e.tracer.Start(ctx, "phase1-intent-resolution")
    intent := e.resolveIntent(request)
    phase1Span.End()

    // Phase 2
    phase2Span := e.tracer.Start(ctx, "phase2-context-assembly")
    context := e.assembleContext(intent)
    phase2Span.End()

    // Phase 3 with Rust engine tracing
    phase3Span := e.tracer.Start(ctx, "phase3-clinical-intelligence")
    candidates := e.generateCandidates(context)

    // Trace Rust engine calls
    rustSpan := e.tracer.Start(ctx, "rust-engine-calculation")
    calculations := e.callRustEngine(candidates)
    rustSpan.SetAttributes(
        attribute.Int("candidate_count", len(candidates)),
        attribute.String("rust_engine_url", e.rustEngineURL),
    )
    rustSpan.End()

    ranked := e.scoreAndRank(calculations)
    phase3Span.End()

    // Phase 4
    phase4Span := e.tracer.Start(ctx, "phase4-proposal-generation")
    proposals := e.generateProposals(ranked)
    phase4Span.End()
}
```

### Metrics Collection
```go
// Prometheus metrics
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "medication_request_duration_seconds",
            Help: "Time spent processing medication requests",
        },
        []string{"phase", "engine"},
    )

    rustEngineLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "rust_engine_call_duration_seconds",
            Help: "Latency of calls to Rust engine",
        },
        []string{"operation"},
    )
)

func (e *Flow2GoEngine) recordMetrics(phase string, duration time.Duration) {
    requestDuration.WithLabelValues(phase, "go-engine").Observe(duration.Seconds())
}
```

## Configuration Management

### Engine Configuration
```yaml
# config/flow2-engines.yaml
flow2_go_engine:
  port: 8085
  rust_engine_url: "http://localhost:8095"
  timeout: "30s"
  circuit_breaker:
    max_failures: 5
    timeout: "60s"

flow2_rust_engine:
  port: 8095
  worker_threads: 4
  max_concurrent_calculations: 100
  calculation_timeout: "10s"

integration:
  batch_size: 10
  max_batch_wait: "100ms"
  connection_pool_size: 50
```

This integration ensures that the Flow2 engines work together seamlessly to provide high-performance, clinically safe medication intelligence while maintaining the flexibility to optimize each engine for its specific responsibilities.