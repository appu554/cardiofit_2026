# Protocol Engine Rust Implementation Guide
## Safety Gateway Platform Enhancement with Rust Performance Core

### Executive Summary
This documentation provides a Rust-enhanced implementation strategy for the Protocol Engine, leveraging the existing Rust engine infrastructure within the Safety Gateway Platform. The approach combines Go orchestration with Rust performance cores for clinical rule evaluation and temporal constraint processing.

---

## Rust Integration Architecture

### Hybrid Go-Rust Architecture
```
┌─────────────────────────────────────────────────────────┐
│                 Go Orchestration Layer                  │
│  ┌─────────────────┐  ┌──────────────────────────────┐ │
│  │ Protocol Engine │  │    State Manager             │ │
│  │   (Go)          │  │      (Go)                    │ │
│  └─────────┬───────┘  └──────────────┬───────────────┘ │
│            │                         │                 │
│  ┌─────────▼───────┐  ┌──────────────▼───────────────┐ │
│  │   FFI Bridge    │  │     GraphQL Resolvers        │ │
│  │     (Go)        │  │         (Go)                 │ │
│  └─────────┬───────┘  └──────────────────────────────┘ │
└───────────┬─────────────────────────────────────────────┘
            │
┌───────────▼─────────────────────────────────────────────┐
│                Rust Performance Core                    │
│  ┌─────────────────┐  ┌──────────────────────────────┐ │
│  │  Rule Engine    │  │  Temporal Constraint Engine │ │
│  │    (Rust)       │  │          (Rust)             │ │
│  └─────────────────┘  └──────────────────────────────┘ │
│  ┌─────────────────┐  ┌──────────────────────────────┐ │
│  │ Protocol Parser │  │    Clinical Algorithm       │ │
│  │    (Rust)       │  │        (Rust)               │ │
│  └─────────────────┘  └──────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Why Hybrid Go-Rust Approach?

**Go Strengths (Orchestration Layer)**:
- Excellent for HTTP services, GraphQL, and gRPC
- Strong concurrency model for I/O operations
- Established integration with existing Go Safety Gateway
- Better for database connections and API handling

**Rust Strengths (Performance Core)**:
- Zero-cost abstractions for clinical algorithms
- Memory safety for mission-critical healthcare logic  
- Exceptional performance for rule evaluation (5-10x faster than Go)
- Pattern matching ideal for protocol state machines
- Existing flow2-rust-engine integration patterns

---

## Phase 1: Rust Core Components (Enhanced - Weeks 1-2)

### 1.1 Protocol Engine Rust Core
**Duration**: 4 days
**Dependencies**: Existing rust_engines infrastructure

#### Rust Protocol Engine Core Structure
```rust
// src/protocol/mod.rs
pub mod engine;
pub mod state_machine;
pub mod temporal;
pub mod rules;
pub mod types;

// Core Protocol Engine
pub struct ProtocolEngine {
    rule_engine: RuleEngine,
    state_manager: StateMachineManager,
    temporal_engine: TemporalConstraintEngine,
    knowledge_cache: Arc<RwLock<KnowledgeCache>>,
}

impl ProtocolEngine {
    pub fn new(config: ProtocolEngineConfig) -> Result<Self, ProtocolEngineError> {
        Ok(Self {
            rule_engine: RuleEngine::new(&config.rule_config)?,
            state_manager: StateMachineManager::new(&config.state_config)?,
            temporal_engine: TemporalConstraintEngine::new(&config.temporal_config)?,
            knowledge_cache: Arc::new(RwLock::new(KnowledgeCache::new())),
        })
    }

    pub fn evaluate_protocol(
        &self,
        request: &ProtocolEvaluationRequest,
    ) -> Result<ProtocolEvaluationResult, ProtocolEngineError> {
        // High-performance protocol evaluation
        let rules = self.load_applicable_rules(&request.protocol_id, &request.snapshot_id)?;
        let context = self.build_evaluation_context(request)?;
        
        // Parallel rule evaluation using Rayon
        let rule_results: Vec<RuleResult> = rules
            .par_iter()
            .map(|rule| self.rule_engine.evaluate(rule, &context))
            .collect::<Result<Vec<_>, _>>()?;
            
        self.aggregate_results(rule_results, &context)
    }
}
```

#### Protocol Types & Models
```rust
// src/protocol/types.rs
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolEvaluationRequest {
    pub request_id: String,
    pub snapshot_id: String,
    pub protocol_id: String,
    pub patient_context: PatientContext,
    pub proposed_action: ClinicalAction,
    pub urgency: Urgency,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolEvaluationResult {
    pub request_id: String,
    pub snapshot_id: String,
    pub decision: ProtocolDecision,
    pub triggered_protocols: Vec<TriggeredProtocol>,
    pub constraints: Constraints,
    pub recommendations: Vec<Recommendation>,
    pub approval_required: bool,
    pub evaluation_time_ms: u64,
    pub provenance: EvaluationProvenance,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ProtocolDecision {
    Accept,
    Reject { reason: String },
    Warning { message: String },
    RequireApproval { criteria: ApprovalCriteria },
    RecommendAlternative { alternatives: Vec<Alternative> },
    Delay { until: DateTime<Utc>, reason: String },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Constraints {
    pub hard: Vec<HardConstraint>,
    pub soft: Vec<SoftConstraint>,
    pub temporal: Vec<TemporalConstraint>,
}
```

### 1.2 High-Performance Rule Engine
**Duration**: 3 days

```rust
// src/protocol/rules.rs
use rayon::prelude::*;

pub struct RuleEngine {
    compiled_rules: HashMap<String, CompiledRule>,
    expression_evaluator: ExpressionEvaluator,
}

impl RuleEngine {
    pub fn evaluate(&self, rule: &Rule, context: &EvaluationContext) -> Result<RuleResult, RuleError> {
        // Leverage pattern matching for rule evaluation
        match &rule.rule_type {
            RuleType::Simple(condition) => {
                self.evaluate_simple_condition(condition, context)
            }
            RuleType::Complex(conditions) => {
                self.evaluate_complex_conditions(conditions, context)
            }
            RuleType::Temporal(temporal_rule) => {
                self.evaluate_temporal_rule(temporal_rule, context)
            }
            RuleType::StateMachine(state_rule) => {
                self.evaluate_state_rule(state_rule, context)
            }
        }
    }

    fn evaluate_complex_conditions(
        &self,
        conditions: &[Condition],
        context: &EvaluationContext,
    ) -> Result<RuleResult, RuleError> {
        // Parallel evaluation for independent conditions
        let results: Vec<ConditionResult> = conditions
            .par_iter()
            .map(|condition| self.evaluate_condition(condition, context))
            .collect::<Result<Vec<_>, _>>()?;

        // Combine results based on logical operators
        self.combine_condition_results(results, &conditions)
    }
}

// Compiled rule for performance
#[derive(Debug)]
pub struct CompiledRule {
    pub id: String,
    pub bytecode: Vec<Instruction>,
    pub metadata: RuleMetadata,
}

// Instruction set for rule evaluation VM
#[derive(Debug)]
pub enum Instruction {
    LoadPatientValue(PatientField),
    LoadMedicationValue(MedicationField),
    LoadConstant(Value),
    Compare(Operator),
    LogicalAnd,
    LogicalOr,
    Jump(usize),
    Return(ProtocolDecision),
}
```

### 1.3 Temporal Constraint Engine
**Duration**: 3 days

```rust
// src/protocol/temporal.rs
pub struct TemporalConstraintEngine {
    time_windows: HashMap<String, TimeWindow>,
    snapshot_time_cache: LruCache<String, DateTime<Utc>>,
}

impl TemporalConstraintEngine {
    pub fn evaluate_temporal_constraint(
        &self,
        constraint: &TemporalConstraint,
        context: &EvaluationContext,
    ) -> Result<TemporalResult, TemporalError> {
        let snapshot_time = self.get_snapshot_time(&context.snapshot_id)?;
        let current_time = Utc::now();
        
        match constraint.constraint_type {
            TemporalConstraintType::TimeWindow { start, end } => {
                self.evaluate_time_window(snapshot_time, current_time, start, end)
            }
            TemporalConstraintType::MaxAge { duration } => {
                self.evaluate_max_age(snapshot_time, current_time, duration)
            }
            TemporalConstraintType::Sequence { events } => {
                self.evaluate_sequence(events, context)
            }
            TemporalConstraintType::Periodic { interval, count } => {
                self.evaluate_periodic(interval, count, context)
            }
        }
    }

    fn evaluate_time_window(
        &self,
        snapshot_time: DateTime<Utc>,
        current_time: DateTime<Utc>,
        start: Duration,
        end: Duration,
    ) -> Result<TemporalResult, TemporalError> {
        let window_start = snapshot_time + start;
        let window_end = snapshot_time + end;
        
        if current_time >= window_start && current_time <= window_end {
            Ok(TemporalResult::WithinWindow)
        } else if current_time < window_start {
            Ok(TemporalResult::TooEarly {
                remaining: window_start - current_time,
            })
        } else {
            Ok(TemporalResult::TooLate {
                exceeded: current_time - window_end,
            })
        }
    }
}
```

---

## Phase 2: State Machine & Protocol Management (Weeks 3-4)

### 2.1 Protocol State Machine (Rust Implementation)
```rust
// src/protocol/state_machine.rs
use std::collections::HashMap;
use serde::{Deserialize, Serialize};

pub struct StateMachineManager {
    state_machines: HashMap<String, ProtocolStateMachine>,
    transition_validators: Vec<TransitionValidator>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolStateMachine {
    pub protocol_id: String,
    pub states: HashMap<String, ProtocolState>,
    pub transitions: Vec<StateTransition>,
    pub initial_state: String,
}

impl ProtocolStateMachine {
    pub fn transition(
        &self,
        current_state: &str,
        event: &ProtocolEvent,
        context: &EvaluationContext,
    ) -> Result<StateTransitionResult, StateTransitionError> {
        // Find valid transitions from current state
        let valid_transitions: Vec<&StateTransition> = self
            .transitions
            .iter()
            .filter(|t| t.from_state == current_state)
            .filter(|t| self.can_transition(t, event, context))
            .collect();

        match valid_transitions.len() {
            0 => Err(StateTransitionError::NoValidTransition {
                from: current_state.to_string(),
                event: event.clone(),
            }),
            1 => {
                let transition = valid_transitions[0];
                Ok(StateTransitionResult {
                    from_state: current_state.to_string(),
                    to_state: transition.to_state.clone(),
                    transition_id: transition.id.clone(),
                    timestamp: Utc::now(),
                    evidence: self.capture_transition_evidence(context),
                })
            }
            _ => Err(StateTransitionError::AmbiguousTransition {
                from: current_state.to_string(),
                valid_transitions: valid_transitions.into_iter().map(|t| t.id.clone()).collect(),
            }),
        }
    }

    // Example: Sepsis Bundle State Machine
    pub fn sepsis_bundle_state_machine() -> Self {
        ProtocolStateMachine {
            protocol_id: "sepsis-bundle".to_string(),
            states: HashMap::from([
                ("INACTIVE".to_string(), ProtocolState::new("INACTIVE", false)),
                ("RECOGNITION".to_string(), ProtocolState::new("RECOGNITION", true)),
                ("INITIAL_RESUSCITATION".to_string(), ProtocolState::new("INITIAL_RESUSCITATION", true)),
                ("ANTIBIOTIC_ADMINISTRATION".to_string(), ProtocolState::new("ANTIBIOTIC_ADMINISTRATION", true)),
                ("SOURCE_CONTROL".to_string(), ProtocolState::new("SOURCE_CONTROL", true)),
                ("MAINTENANCE".to_string(), ProtocolState::new("MAINTENANCE", false)),
            ]),
            transitions: vec![
                StateTransition {
                    id: "trigger_sepsis".to_string(),
                    from_state: "INACTIVE".to_string(),
                    to_state: "RECOGNITION".to_string(),
                    trigger: TransitionTrigger::Condition("sepsis_criteria_met".to_string()),
                    actions: vec![Action::LogEvent("sepsis_recognized".to_string())],
                },
                StateTransition {
                    id: "begin_resuscitation".to_string(),
                    from_state: "RECOGNITION".to_string(),
                    to_state: "INITIAL_RESUSCITATION".to_string(),
                    trigger: TransitionTrigger::Event("begin_treatment".to_string()),
                    actions: vec![
                        Action::StartTimer("resuscitation_window".to_string()),
                        Action::SetFlag("fluid_therapy_indicated".to_string()),
                    ],
                },
                // ... more transitions
            ],
            initial_state: "INACTIVE".to_string(),
        }
    }
}
```

### 2.2 Go-Rust FFI Bridge Enhancement
```go
// internal/protocol/rust_bridge.go
package protocol

/*
#cgo LDFLAGS: -L../engines/rust_engines/target/release -lrust_protocol_engine
#include "../engines/rust_engines/protocol_engine.h"
*/
import "C"
import (
    "encoding/json"
    "fmt"
    "unsafe"
)

type RustProtocolEngine struct {
    engine *C.ProtocolEngine
}

func NewRustProtocolEngine(config ProtocolEngineConfig) (*RustProtocolEngine, error) {
    configJson, err := json.Marshal(config)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal config: %w", err)
    }

    cConfig := C.CString(string(configJson))
    defer C.free(unsafe.Pointer(cConfig))

    cEngine := C.protocol_engine_new(cConfig)
    if cEngine == nil {
        return nil, fmt.Errorf("failed to create rust protocol engine")
    }

    return &RustProtocolEngine{engine: cEngine}, nil
}

func (r *RustProtocolEngine) EvaluateProtocol(request *ProtocolEvaluationRequest) (*ProtocolEvaluationResult, error) {
    requestJson, err := json.Marshal(request)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    cRequest := C.CString(string(requestJson))
    defer C.free(unsafe.Pointer(cRequest))

    cResult := C.protocol_engine_evaluate(r.engine, cRequest)
    defer C.free_protocol_result(cResult)

    if cResult == nil {
        return nil, fmt.Errorf("protocol evaluation failed")
    }

    resultJson := C.GoString(cResult.json_data)
    
    var result ProtocolEvaluationResult
    if err := json.Unmarshal([]byte(resultJson), &result); err != nil {
        return nil, fmt.Errorf("failed to unmarshal result: %w", err)
    }

    return &result, nil
}

func (r *RustProtocolEngine) Close() error {
    if r.engine != nil {
        C.protocol_engine_free(r.engine)
        r.engine = nil
    }
    return nil
}
```

### 2.3 Rust FFI Interface
```rust
// src/ffi/protocol.rs
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use crate::protocol::ProtocolEngine;

#[no_mangle]
pub extern "C" fn protocol_engine_new(config_json: *const c_char) -> *mut ProtocolEngine {
    let config_str = unsafe {
        match CStr::from_ptr(config_json).to_str() {
            Ok(s) => s,
            Err(_) => return std::ptr::null_mut(),
        }
    };

    let config = match serde_json::from_str(config_str) {
        Ok(c) => c,
        Err(_) => return std::ptr::null_mut(),
    };

    match ProtocolEngine::new(config) {
        Ok(engine) => Box::into_raw(Box::new(engine)),
        Err(_) => std::ptr::null_mut(),
    }
}

#[no_mangle]
pub extern "C" fn protocol_engine_evaluate(
    engine: *mut ProtocolEngine,
    request_json: *const c_char,
) -> *mut ProtocolResult {
    let engine = unsafe { &*engine };
    
    let request_str = unsafe {
        match CStr::from_ptr(request_json).to_str() {
            Ok(s) => s,
            Err(_) => return std::ptr::null_mut(),
        }
    };

    let request = match serde_json::from_str(request_str) {
        Ok(r) => r,
        Err(_) => return std::ptr::null_mut(),
    };

    match engine.evaluate_protocol(&request) {
        Ok(result) => {
            let json_data = match serde_json::to_string(&result) {
                Ok(json) => json,
                Err(_) => return std::ptr::null_mut(),
            };
            
            let c_json = match CString::new(json_data) {
                Ok(s) => s.into_raw(),
                Err(_) => return std::ptr::null_mut(),
            };
            
            Box::into_raw(Box::new(ProtocolResult { json_data: c_json }))
        }
        Err(_) => std::ptr::null_mut(),
    }
}

#[repr(C)]
pub struct ProtocolResult {
    json_data: *mut c_char,
}

#[no_mangle]
pub extern "C" fn free_protocol_result(result: *mut ProtocolResult) {
    if !result.is_null() {
        unsafe {
            let result = Box::from_raw(result);
            if !result.json_data.is_null() {
                let _ = CString::from_raw(result.json_data);
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn protocol_engine_free(engine: *mut ProtocolEngine) {
    if !engine.is_null() {
        unsafe { Box::from_raw(engine) };
    }
}
```

---

## Phase 3: Enhanced Integration & Performance (Weeks 5-6)

### 3.1 Go Orchestration Layer
```go
// internal/protocol/engine.go
package protocol

import (
    "context"
    "fmt"
    "time"
    
    "github.com/cardiofit/safety-gateway/internal/shared"
)

// ProtocolEngine orchestrates protocol evaluation using Rust core
type ProtocolEngine struct {
    rustEngine      *RustProtocolEngine
    stateManager    *ProtocolStateManager
    snapshotManager shared.SnapshotManager
    contextGateway  shared.ContextGatewayClient
    auditService    shared.AuditService
    approvalEngine  *PolicyApprovalEngine
    config          *ProtocolEngineConfig
}

func NewProtocolEngine(config *ProtocolEngineConfig, deps *Dependencies) (*ProtocolEngine, error) {
    rustEngine, err := NewRustProtocolEngine(*config)
    if err != nil {
        return nil, fmt.Errorf("failed to create rust engine: %w", err)
    }

    return &ProtocolEngine{
        rustEngine:      rustEngine,
        stateManager:    NewProtocolStateManager(deps.DB, deps.Cache),
        snapshotManager: deps.SnapshotManager,
        contextGateway:  deps.ContextGateway,
        auditService:    deps.AuditService,
        approvalEngine:  NewPolicyApprovalEngine(deps),
        config:          config,
    }, nil
}

func (pe *ProtocolEngine) Evaluate(
    ctx context.Context,
    proposedAction *ClinicalAction,
    patientContext *PatientContext,
    snapshotID string,
) (*ProtocolResult, error) {
    start := time.Now()
    
    // Step 1: Validate snapshot (Go orchestration)
    snapshot, err := pe.snapshotManager.GetSnapshot(ctx, snapshotID)
    if err != nil {
        return nil, fmt.Errorf("invalid snapshot: %w", err)
    }

    // Step 2: Build evaluation request
    request := &ProtocolEvaluationRequest{
        RequestID:       generateRequestID(),
        SnapshotID:      snapshotID,
        ProtocolID:      pe.determineProtocolID(proposedAction, patientContext),
        PatientContext:  *patientContext,
        ProposedAction:  *proposedAction,
        Urgency:         pe.determineUrgency(ctx),
    }

    // Step 3: Delegate to Rust engine for high-performance evaluation
    rustResult, err := pe.rustEngine.EvaluateProtocol(request)
    if err != nil {
        return nil, fmt.Errorf("rust evaluation failed: %w", err)
    }

    // Step 4: Handle stateful protocols (Go orchestration)
    if pe.requiresStateManagement(rustResult) {
        stateResult, err := pe.handleStateTransition(ctx, rustResult, request)
        if err != nil {
            return nil, fmt.Errorf("state transition failed: %w", err)
        }
        rustResult = pe.mergeStateResult(rustResult, stateResult)
    }

    // Step 5: Handle approvals if required (Go orchestration)
    if rustResult.ApprovalRequired {
        approvalRequest, err := pe.approvalEngine.InitiateApproval(ctx, rustResult, request)
        if err != nil {
            return nil, fmt.Errorf("approval initiation failed: %w", err)
        }
        rustResult.ApprovalRequestID = &approvalRequest.ID
    }

    // Step 6: Audit trail (Go orchestration)
    auditEvent := &AuditEvent{
        Type:           "PROTOCOL_EVALUATED",
        RequestID:      request.RequestID,
        SnapshotID:     snapshotID,
        Duration:       time.Since(start),
        Result:         rustResult,
        DigitalSignature: pe.auditService.Sign(rustResult),
    }
    
    if err := pe.auditService.LogEvent(ctx, auditEvent); err != nil {
        // Log but don't fail - audit is important but not critical
        pe.config.Logger.Errorf("failed to log audit event: %v", err)
    }

    return &ProtocolResult{
        RequestID:        request.RequestID,
        SnapshotID:       snapshotID,
        Decision:         rustResult.Decision,
        TriggeredProtocols: rustResult.TriggeredProtocols,
        Constraints:      rustResult.Constraints,
        Recommendations:  rustResult.Recommendations,
        ApprovalRequired: rustResult.ApprovalRequired,
        ApprovalRequestID: rustResult.ApprovalRequestID,
        EvaluationTime:   time.Since(start),
        Signature:        auditEvent.DigitalSignature,
        Provenance:       rustResult.Provenance,
    }, nil
}
```

### 3.2 Performance Optimizations
```rust
// src/protocol/optimizations.rs
use dashmap::DashMap;
use rayon::prelude::*;

pub struct PerformanceOptimizer {
    rule_cache: DashMap<String, CompiledRule>,
    hot_protocols: Arc<RwLock<HashSet<String>>>,
    metrics: Arc<Mutex<PerformanceMetrics>>,
}

impl PerformanceOptimizer {
    pub fn precompile_protocols(&self, protocol_ids: &[String]) -> Result<(), OptimizationError> {
        // Parallel protocol compilation
        protocol_ids.par_iter().try_for_each(|protocol_id| {
            let protocol = self.load_protocol(protocol_id)?;
            let compiled = self.compile_protocol(protocol)?;
            self.rule_cache.insert(protocol_id.clone(), compiled);
            Ok::<(), OptimizationError>(())
        })?;
        
        Ok(())
    }

    pub fn enable_jit_compilation(&self) -> Result<(), OptimizationError> {
        // Just-in-time compilation for frequently used protocols
        let hot_protocols = self.hot_protocols.read().unwrap();
        for protocol_id in hot_protocols.iter() {
            if !self.rule_cache.contains_key(protocol_id) {
                self.compile_protocol_async(protocol_id.clone());
            }
        }
        Ok(())
    }
}

// Performance metrics collection
#[derive(Debug)]
pub struct PerformanceMetrics {
    pub evaluations_per_second: f64,
    pub average_latency_ms: f64,
    pub cache_hit_rate: f64,
    pub memory_usage_mb: f64,
    pub compilation_time_ms: f64,
}
```

---

## Phase 4: Production Deployment & Monitoring (Weeks 7-8)

### 4.1 Comprehensive Testing with Rust Components
```rust
// tests/protocol_integration_tests.rs
use tokio_test;

#[tokio::test]
async fn test_sepsis_bundle_full_workflow() {
    let engine = create_test_protocol_engine().await;
    let snapshot = create_sepsis_test_snapshot().await;
    
    // Test sepsis recognition trigger
    let patient_context = PatientContext {
        temperature: Some(38.5),
        heart_rate: Some(110),
        lactate: Some(2.5),
        ..Default::default()
    };
    
    let action = ClinicalAction {
        action_type: "order_labs".to_string(),
        labs: vec!["blood_culture".to_string(), "lactate".to_string()],
        ..Default::default()
    };
    
    let result = engine.evaluate_protocol(&ProtocolEvaluationRequest {
        request_id: "test-sepsis-001".to_string(),
        snapshot_id: snapshot.id.clone(),
        protocol_id: "sepsis-bundle".to_string(),
        patient_context,
        proposed_action: action,
        urgency: Urgency::Stat,
    }).unwrap();
    
    // Verify sepsis protocol triggered
    assert_eq!(result.decision, ProtocolDecision::Accept);
    assert!(result.triggered_protocols.iter().any(|p| p.id == "sepsis-bundle"));
    
    // Verify state transition
    let state_transition = result.triggered_protocols[0].state_transition.as_ref().unwrap();
    assert_eq!(state_transition.from_state, "INACTIVE");
    assert_eq!(state_transition.to_state, "RECOGNITION");
}

#[tokio::test] 
async fn test_protocol_performance_benchmarks() {
    let engine = create_test_protocol_engine().await;
    let test_cases = generate_test_cases(1000);
    
    let start = std::time::Instant::now();
    
    // Parallel evaluation benchmark
    let results: Vec<_> = test_cases
        .par_iter()
        .map(|test_case| engine.evaluate_protocol(test_case))
        .collect();
    
    let duration = start.elapsed();
    let evaluations_per_second = 1000.0 / duration.as_secs_f64();
    
    // Assert performance targets
    assert!(evaluations_per_second > 1000.0, "Target: >1000 evaluations/second, actual: {}", evaluations_per_second);
    assert!(duration.as_millis() < 1000, "Target: <1000ms total, actual: {}ms", duration.as_millis());
    
    // Verify all evaluations succeeded
    assert!(results.iter().all(|r| r.is_ok()));
}
```

### 4.2 Production Monitoring & Observability
```rust
// src/observability/metrics.rs
use prometheus::{Counter, Histogram, Gauge};
use once_cell::sync::Lazy;

static PROTOCOL_EVALUATIONS: Lazy<Counter> = Lazy::new(|| {
    Counter::new("protocol_evaluations_total", "Total protocol evaluations")
        .expect("Failed to create counter")
});

static EVALUATION_DURATION: Lazy<Histogram> = Lazy::new(|| {
    Histogram::with_opts(
        prometheus::HistogramOpts::new(
            "protocol_evaluation_duration_seconds",
            "Protocol evaluation duration"
        ).buckets(vec![0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1.0])
    ).expect("Failed to create histogram")
});

static ACTIVE_PROTOCOLS: Lazy<Gauge> = Lazy::new(|| {
    Gauge::new("protocol_active_states", "Number of active protocol states")
        .expect("Failed to create gauge")
});

pub fn record_evaluation_metrics(result: &ProtocolEvaluationResult, duration: std::time::Duration) {
    PROTOCOL_EVALUATIONS.inc();
    EVALUATION_DURATION.observe(duration.as_secs_f64());
    
    // Record decision type metrics
    let decision_label = match &result.decision {
        ProtocolDecision::Accept => "accept",
        ProtocolDecision::Reject { .. } => "reject",
        ProtocolDecision::Warning { .. } => "warning",
        ProtocolDecision::RequireApproval { .. } => "require_approval",
        _ => "other",
    };
    
    PROTOCOL_EVALUATIONS.with_label_values(&[decision_label]).inc();
}
```

### 4.3 Kubernetes Deployment with Rust Components
```yaml
# k8s/protocol-engine-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: protocol-engine
  namespace: safety-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: protocol-engine
  template:
    metadata:
      labels:
        app: protocol-engine
        version: v1.0.0-rust
    spec:
      containers:
      - name: protocol-engine
        image: safety-gateway/protocol-engine-rust:1.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8081  
          name: metrics
        - containerPort: 8082
          name: grpc
        env:
        - name: RUST_LOG
          value: "info"
        - name: PROTOCOL_ENGINE_CONFIG
          value: "/config/protocol-engine.toml"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: protocol-engine-secret
              key: database-url
        resources:
          requests:
            memory: "256Mi"    # Rust uses less memory
            cpu: "250m"        # Rust is more CPU efficient
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config-volume
          mountPath: /config
        - name: protocols-volume
          mountPath: /protocols
      volumes:
      - name: config-volume
        configMap:
          name: protocol-engine-config
      - name: protocols-volume
        configMap:
          name: protocol-definitions
```

---

## Performance Benefits & Benchmarks

### Expected Performance Improvements

| Metric | Go-Only Implementation | Hybrid Go-Rust | Improvement |
|--------|------------------------|-----------------|-------------|
| **Evaluation Latency** | ~50ms | ~10ms | 5x faster |
| **Memory Usage** | ~512MB | ~256MB | 2x reduction |
| **Throughput** | ~500 req/sec | ~2500 req/sec | 5x increase |
| **CPU Efficiency** | 100% (baseline) | ~40% | 2.5x efficiency |
| **Rule Compilation** | Runtime | Compile-time | 10x faster |

### Memory Safety & Clinical Safety Benefits

1. **Zero Buffer Overflows**: Rust's memory safety eliminates a class of bugs critical in healthcare
2. **Panic-Free Operation**: Rust's Result/Option types ensure graceful error handling
3. **Data Race Prevention**: Rust's ownership system prevents concurrent access issues
4. **Predictable Performance**: No garbage collection pauses during critical evaluations

---

## Migration Strategy

### Phase 1: Hybrid Implementation
- Keep Go orchestration layer for API handling and database operations
- Implement performance-critical components (rule evaluation, temporal constraints) in Rust
- Use FFI bridge for seamless integration

### Phase 2: Gradual Rust Adoption  
- Monitor performance gains and stability
- Migrate additional components to Rust as confidence builds
- Maintain Go components for complex integration logic

### Phase 3: Full Optimization
- Optimize hot paths with Rust implementations
- Consider async Rust for I/O-bound operations
- Profile and optimize based on production metrics

---

## Risk Mitigation & Considerations

### Technical Risks
- **FFI Overhead**: Minimize boundary crossings, batch operations
- **Memory Management**: Careful handling of allocated memory across FFI boundary
- **Error Propagation**: Robust error handling across language boundaries

### Operational Risks  
- **Deployment Complexity**: Both Rust and Go binaries must be deployed together
- **Debugging**: Mixed-language debugging requires specialized tools
- **Team Skills**: Rust learning curve for Go developers

### Mitigation Strategies
- **Comprehensive Testing**: Extensive integration tests for FFI boundary
- **Gradual Rollout**: Deploy hybrid system in shadow mode first
- **Fallback Mechanism**: Keep Go-only implementation as backup
- **Training Program**: Rust training for development team

This hybrid Go-Rust approach leverages the strengths of both languages while maintaining the existing Safety Gateway Platform architecture and ensuring optimal performance for clinical decision support systems.