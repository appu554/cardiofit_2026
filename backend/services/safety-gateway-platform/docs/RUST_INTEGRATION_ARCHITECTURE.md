# Rust Integration Architecture for Protocol Engine
## Technical Architecture Documentation

### Overview
This document details the technical architecture for integrating Rust performance cores into the Go-based Safety Gateway Platform, specifically for the Protocol Engine implementation.

---

## Architectural Principles

### Hybrid Language Strategy
The Protocol Engine uses a **hybrid Go-Rust architecture** that leverages the strengths of both languages:

- **Go**: HTTP services, GraphQL federation, database operations, orchestration
- **Rust**: High-performance clinical algorithms, rule evaluation, temporal constraints

### Language Boundary Design
```
┌─────────────────────────────────────────────────────────┐
│                    Go Application Layer                  │
│  ┌─────────────────┐  ┌──────────────────────────────┐ │
│  │  HTTP Handlers  │  │    GraphQL Resolvers         │ │
│  │     (Go)        │  │        (Go)                  │ │
│  └─────────────────┘  └──────────────────────────────┘ │
│                           │                             │
│  ┌─────────────────────────▼─────────────────────────┐  │
│  │        Protocol Engine Orchestrator               │  │
│  │                  (Go)                             │  │
│  └─────────────────────┬───────────────────────────┬─┘  │
└───────────────────────┬┼───────────────────────────┼────┘
                        ││                           │
┌───────────────────────▼▼───────────────────────────▼────┐
│                    FFI Bridge Layer                     │
│  ┌─────────────────┐           ┌──────────────────────┐ │
│  │   C Bindings    │           │   Memory Management  │ │
│  │     (Go/C)      │           │       (Go/C)         │ │
│  └─────────────────┘           └──────────────────────┘ │
└───────────────────────┬───────────────────────────────┬─┘
                        │                               │
┌───────────────────────▼───────────────────────────────▼─┐
│                 Rust Performance Core                   │
│  ┌─────────────────┐  ┌──────────────────────────────┐ │
│  │  Rule Engine    │  │  Temporal Constraint Engine │ │
│  │    (Rust)       │  │          (Rust)             │ │
│  └─────────────────┘  └──────────────────────────────┘ │
│  ┌─────────────────┐  ┌──────────────────────────────┐ │
│  │ Protocol Parser │  │    State Machine Engine     │ │
│  │    (Rust)       │  │         (Rust)              │ │
│  └─────────────────┘  └──────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

## Component Architecture

### 1. Go Orchestration Layer

#### Protocol Engine Controller
```go
// Manages high-level protocol evaluation workflow
type ProtocolEngine struct {
    rustCore        *RustProtocolCore
    stateManager    *StateManager
    snapshotClient  SnapshotClient
    approvalEngine  *ApprovalEngine
    auditLogger     AuditLogger
}

// Orchestrates evaluation with error handling and state management
func (pe *ProtocolEngine) Evaluate(ctx context.Context, req *EvaluationRequest) (*EvaluationResult, error) {
    // 1. Validate request and snapshot
    // 2. Prepare evaluation context  
    // 3. Delegate to Rust core for performance-critical evaluation
    // 4. Handle state transitions and approvals
    // 5. Audit and return results
}
```

#### Benefits of Go Orchestration
- **Error Handling**: Go's explicit error handling for orchestration logic
- **Concurrency**: Goroutines for I/O operations and service coordination
- **Integration**: Native HTTP, gRPC, and database client libraries
- **Observability**: Structured logging, metrics, and distributed tracing

### 2. FFI Bridge Layer

#### Memory Management Strategy
```go
// Safe memory management for FFI operations
type RustProtocolCore struct {
    engine unsafe.Pointer
    mutex  sync.RWMutex
}

func (rpc *RustProtocolCore) Evaluate(request []byte) ([]byte, error) {
    rpc.mutex.RLock()
    defer rpc.mutex.RUnlock()
    
    // Allocate C string for request
    cRequest := C.CString(string(request))
    defer C.free(unsafe.Pointer(cRequest))
    
    // Call Rust function through FFI
    cResult := C.rust_evaluate_protocol(rpc.engine, cRequest)
    defer C.rust_free_result(cResult)
    
    // Convert C result back to Go
    result := C.GoString(cResult.data)
    return []byte(result), nil
}
```

#### FFI Safety Patterns
- **RAII Pattern**: Automatic cleanup using defer statements
- **Error Propagation**: Convert Rust errors to Go errors safely
- **Memory Safety**: No memory leaks or dangling pointers
- **Thread Safety**: Mutex protection for shared FFI resources

### 3. Rust Performance Core

#### High-Performance Rule Engine
```rust
// Optimized for clinical rule evaluation
pub struct RuleEngine {
    compiled_rules: DashMap<String, CompiledRule>,
    expression_cache: LruCache<String, CompiledExpression>,
    thread_pool: ThreadPool,
}

impl RuleEngine {
    pub fn evaluate_parallel(&self, rules: &[Rule], context: &EvaluationContext) -> Vec<RuleResult> {
        // Use Rayon for data parallelism
        rules.par_iter()
            .map(|rule| self.evaluate_single(rule, context))
            .collect()
    }
    
    fn evaluate_single(&self, rule: &Rule, context: &EvaluationContext) -> RuleResult {
        match &rule.rule_type {
            RuleType::Simple(condition) => self.evaluate_condition(condition, context),
            RuleType::Complex(logic) => self.evaluate_complex_logic(logic, context),
            RuleType::Temporal(temporal) => self.evaluate_temporal_constraint(temporal, context),
        }
    }
}
```

#### State Machine Implementation
```rust
// Memory-efficient protocol state machines
pub struct ProtocolStateMachine {
    states: FxHashMap<StateId, ProtocolState>,
    transitions: Vec<StateTransition>,
    current_state: StateId,
}

impl ProtocolStateMachine {
    pub fn transition(&mut self, event: &ProtocolEvent) -> Result<TransitionResult, StateError> {
        // Pattern matching for efficient state transitions
        match self.find_valid_transition(event) {
            Some(transition) => {
                let old_state = self.current_state;
                self.current_state = transition.target_state;
                Ok(TransitionResult {
                    from: old_state,
                    to: transition.target_state,
                    timestamp: SystemTime::now(),
                })
            }
            None => Err(StateError::InvalidTransition {
                current_state: self.current_state,
                event: event.clone(),
            }),
        }
    }
}
```

#### Temporal Constraint Engine
```rust
// High-precision temporal constraint evaluation
pub struct TemporalConstraintEngine {
    time_windows: FxHashMap<String, TimeWindow>,
    constraint_cache: LruCache<String, CompiledConstraint>,
}

impl TemporalConstraintEngine {
    pub fn evaluate_constraint(&self, constraint: &TemporalConstraint, context: &EvaluationContext) -> TemporalResult {
        let snapshot_time = context.snapshot_time;
        let current_time = SystemTime::now();
        
        match constraint {
            TemporalConstraint::Window { start, end } => {
                self.evaluate_time_window(snapshot_time, current_time, *start, *end)
            }
            TemporalConstraint::Sequence { events, max_duration } => {
                self.evaluate_event_sequence(events, *max_duration, context)
            }
            TemporalConstraint::Periodic { interval, count } => {
                self.evaluate_periodic_constraint(*interval, *count, context)
            }
        }
    }
}
```

---

## Data Flow Architecture

### Request Processing Pipeline
```
1. HTTP Request → Go HTTP Handler
2. Request Validation → Go Middleware  
3. Snapshot Resolution → Go Service Client
4. Context Building → Go Orchestrator
5. Rule Evaluation → Rust Performance Core (FFI)
6. Result Processing → Go Orchestrator
7. State Management → Go State Manager
8. Approval Handling → Go Approval Engine
9. Audit Logging → Go Audit Service
10. HTTP Response → Go HTTP Handler
```

### Memory Management Flow
```rust
// Rust side: Zero-copy where possible
#[no_mangle]
pub extern "C" fn rust_evaluate_protocol(
    engine: *mut ProtocolEngine,
    request_json: *const c_char,
) -> *mut RustResult {
    // 1. Deserialize request (allocation)
    let request: EvaluationRequest = serde_json::from_str(request_str)?;
    
    // 2. Evaluate using borrowed references (zero-copy)
    let result = engine.evaluate(&request)?;
    
    // 3. Serialize result (allocation)  
    let json_result = serde_json::to_string(&result)?;
    
    // 4. Return owned C string (caller must free)
    Box::into_raw(Box::new(RustResult::new(json_result)))
}
```

```go
// Go side: RAII cleanup pattern
func (pe *ProtocolEngine) evaluateRust(request *EvaluationRequest) (*EvaluationResult, error) {
    // 1. Serialize to JSON (Go allocation)
    jsonData, err := json.Marshal(request)
    if err != nil {
        return nil, err
    }
    
    // 2. Create C string (C allocation)
    cRequest := C.CString(string(jsonData))
    defer C.free(unsafe.Pointer(cRequest)) // Auto cleanup
    
    // 3. Call Rust function (Rust allocation)
    cResult := C.rust_evaluate_protocol(pe.rustEngine, cRequest)
    defer C.rust_free_result(cResult) // Auto cleanup
    
    // 4. Convert back to Go (Go allocation)
    resultJson := C.GoString(cResult.data)
    var result EvaluationResult
    err = json.Unmarshal([]byte(resultJson), &result)
    return &result, err
}
```

---

## Performance Optimizations

### 1. Compilation Strategies

#### Rust Compile-Time Optimizations
```toml
# Cargo.toml - Release profile optimization
[profile.release]
opt-level = 3              # Maximum optimization
lto = true                 # Link-time optimization  
codegen-units = 1          # Single codegen unit for better optimization
panic = "abort"            # Smaller binary size
overflow-checks = false    # Remove integer overflow checks in release

[profile.release.package."*"]
opt-level = 3              # Optimize dependencies too
```

#### Protocol Pre-compilation
```rust
// Build-time protocol compilation
pub struct ProtocolCompiler {
    rule_compiler: RuleCompiler,
    state_machine_compiler: StateMachineCompiler,
}

impl ProtocolCompiler {
    pub fn compile_protocol(&self, protocol: &Protocol) -> CompiledProtocol {
        CompiledProtocol {
            id: protocol.id.clone(),
            compiled_rules: self.compile_rules(&protocol.rules),
            compiled_state_machine: self.compile_state_machine(&protocol.state_machine),
            bytecode: self.generate_bytecode(&protocol),
        }
    }
}
```

### 2. Runtime Optimizations

#### Memory Pool Allocation
```rust
use std::sync::Arc;
use parking_lot::RwLock;

pub struct MemoryPool {
    evaluation_contexts: Vec<EvaluationContext>,
    rule_results: Vec<RuleResult>,
    available_contexts: Arc<RwLock<Vec<usize>>>,
}

impl MemoryPool {
    pub fn borrow_context(&self) -> PooledContext {
        let mut available = self.available_contexts.write();
        match available.pop() {
            Some(index) => PooledContext::new(index, &self.evaluation_contexts[index]),
            None => PooledContext::new_allocated(), // Fallback allocation
        }
    }
}
```

#### CPU Cache Optimization
```rust
// Data layout optimized for cache efficiency
#[repr(C)]
pub struct CacheOptimizedRule {
    // Hot data (frequently accessed) - first cache line
    rule_type: u8,           // 1 byte
    priority: u8,            // 1 byte  
    enabled: bool,           // 1 byte
    _padding: [u8; 5],       // Align to 8 bytes
    condition_ptr: *const CompiledCondition, // 8 bytes
    
    // Cold data (less frequently accessed) - second cache line  
    metadata: RuleMetadata,  // Variable size
    description: String,     // Variable size
}
```

### 3. Concurrency Optimizations

#### Lock-Free Data Structures
```rust
use crossbeam::atomic::AtomicCell;
use dashmap::DashMap;

pub struct ConcurrentRuleCache {
    rules: DashMap<String, CompiledRule>,
    hit_count: AtomicCell<u64>,
    miss_count: AtomicCell<u64>,
}

impl ConcurrentRuleCache {
    pub fn get_rule(&self, rule_id: &str) -> Option<CompiledRule> {
        match self.rules.get(rule_id) {
            Some(rule) => {
                self.hit_count.fetch_add(1);
                Some(rule.clone())
            }
            None => {
                self.miss_count.fetch_add(1);
                None
            }
        }
    }
}
```

---

## Error Handling Strategy

### 1. Rust Error Types
```rust
#[derive(Debug, thiserror::Error)]
pub enum ProtocolEngineError {
    #[error("Rule evaluation failed: {0}")]
    RuleEvaluationError(#[from] RuleError),
    
    #[error("State transition failed: {0}")]
    StateTransitionError(#[from] StateError),
    
    #[error("Temporal constraint violation: {0}")]
    TemporalConstraintError(#[from] TemporalError),
    
    #[error("Protocol not found: {protocol_id}")]
    ProtocolNotFound { protocol_id: String },
    
    #[error("Snapshot invalid: {snapshot_id}")]
    InvalidSnapshot { snapshot_id: String },
}

// Convert to C-compatible error codes
impl From<ProtocolEngineError> for i32 {
    fn from(error: ProtocolEngineError) -> i32 {
        match error {
            ProtocolEngineError::RuleEvaluationError(_) => -1001,
            ProtocolEngineError::StateTransitionError(_) => -1002,
            ProtocolEngineError::TemporalConstraintError(_) => -1003,
            ProtocolEngineError::ProtocolNotFound { .. } => -1004,
            ProtocolEngineError::InvalidSnapshot { .. } => -1005,
        }
    }
}
```

### 2. Go Error Handling
```go
// Structured error types for Go layer
type ProtocolEngineError struct {
    Code      int    `json:"code"`
    Message   string `json:"message"`
    RequestID string `json:"request_id"`
    Details   map[string]interface{} `json:"details,omitempty"`
}

func (e ProtocolEngineError) Error() string {
    return fmt.Sprintf("protocol engine error (code: %d): %s", e.Code, e.Message)
}

// Convert FFI error codes to Go errors
func convertRustError(errorCode int, requestID string) error {
    switch errorCode {
    case -1001:
        return ProtocolEngineError{
            Code:      errorCode,
            Message:   "Rule evaluation failed",
            RequestID: requestID,
        }
    case -1002:
        return ProtocolEngineError{
            Code:      errorCode,
            Message:   "State transition failed", 
            RequestID: requestID,
        }
    default:
        return ProtocolEngineError{
            Code:      errorCode,
            Message:   "Unknown protocol engine error",
            RequestID: requestID,
        }
    }
}
```

---

## Testing Strategy

### 1. Unit Testing
```rust
// Rust unit tests with property-based testing
use proptest::prelude::*;

proptest! {
    #[test]
    fn rule_evaluation_is_deterministic(
        rule in arbitrary_rule(),
        context in arbitrary_context()
    ) {
        let engine = RuleEngine::new();
        let result1 = engine.evaluate(&rule, &context);
        let result2 = engine.evaluate(&rule, &context);
        prop_assert_eq!(result1, result2);
    }
}

#[test]
fn temporal_constraint_edge_cases() {
    let engine = TemporalConstraintEngine::new();
    let constraint = TemporalConstraint::Window {
        start: Duration::from_secs(0),
        end: Duration::from_secs(3600), // 1 hour window
    };
    
    // Test exactly at boundary
    let context = create_test_context_at_time(Duration::from_secs(3600));
    let result = engine.evaluate_constraint(&constraint, &context);
    assert_eq!(result, TemporalResult::WithinWindow);
}
```

### 2. Integration Testing  
```go
// Go integration tests with FFI boundary testing
func TestRustFFIIntegration(t *testing.T) {
    engine, err := NewRustProtocolEngine(testConfig())
    require.NoError(t, err)
    defer engine.Close()
    
    // Test serialization/deserialization across FFI boundary
    request := &EvaluationRequest{
        RequestID:  "test-001",
        SnapshotID: "snapshot-001", 
        ProtocolID: "sepsis-bundle",
        // ... other fields
    }
    
    result, err := engine.Evaluate(request)
    require.NoError(t, err)
    require.NotNil(t, result)
    
    // Verify result structure
    assert.Equal(t, request.RequestID, result.RequestID)
    assert.NotEmpty(t, result.Decision)
}

// Memory leak detection
func TestMemoryLeakPrevention(t *testing.T) {
    engine, err := NewRustProtocolEngine(testConfig())
    require.NoError(t, err)
    defer engine.Close()
    
    // Measure initial memory
    var m1 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    // Perform many evaluations
    for i := 0; i < 10000; i++ {
        request := createTestRequest(i)
        _, err := engine.Evaluate(request)
        require.NoError(t, err)
    }
    
    // Measure final memory
    var m2 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    // Memory growth should be bounded
    memoryGrowth := int64(m2.HeapAlloc - m1.HeapAlloc)
    assert.Less(t, memoryGrowth, int64(100*1024*1024), "Memory growth exceeds 100MB")
}
```

### 3. Performance Testing
```rust
// Criterion-based performance benchmarks
use criterion::{criterion_group, criterion_main, Criterion};

fn benchmark_rule_evaluation(c: &mut Criterion) {
    let engine = RuleEngine::new();
    let rule = create_complex_rule();
    let context = create_test_context();
    
    c.bench_function("complex_rule_evaluation", |b| {
        b.iter(|| engine.evaluate(criterion::black_box(&rule), criterion::black_box(&context)))
    });
}

fn benchmark_parallel_evaluation(c: &mut Criterion) {
    let engine = RuleEngine::new();
    let rules = create_rule_set(100);
    let context = create_test_context();
    
    c.bench_function("parallel_rule_evaluation_100", |b| {
        b.iter(|| engine.evaluate_parallel(criterion::black_box(&rules), criterion::black_box(&context)))
    });
}

criterion_group!(benches, benchmark_rule_evaluation, benchmark_parallel_evaluation);
criterion_main!(benches);
```

---

## Deployment Architecture

### 1. Build System
```makefile
# Makefile for hybrid Go-Rust builds
.PHONY: build-rust build-go build-all clean

RUST_TARGET=x86_64-unknown-linux-gnu
GO_TARGET=linux/amd64

build-rust:
	cd internal/engines/rust_engines && \
	cargo build --release --target $(RUST_TARGET)

build-go: build-rust
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
	go build -o bin/protocol-engine \
	-ldflags="-linkmode external -extldflags '-static'" \
	cmd/protocol-engine/main.go

build-all: build-go
	docker build -t protocol-engine:latest .

clean:
	cargo clean -p rust_engines
	rm -rf bin/
```

### 2. Docker Configuration
```dockerfile
# Multi-stage build for efficient Rust-Go binary
FROM rust:1.75-alpine AS rust-builder
WORKDIR /build
COPY internal/engines/rust_engines/ .
RUN apk add --no-cache musl-dev && \
    cargo build --release --target x86_64-unknown-linux-musl

FROM golang:1.21-alpine AS go-builder  
WORKDIR /build
COPY --from=rust-builder /build/target/x86_64-unknown-linux-musl/release/librust_protocol_engine.a ./lib/
COPY . .
RUN apk add --no-cache gcc musl-dev && \
    CGO_ENABLED=1 go build -o protocol-engine \
    -ldflags="-linkmode external -extldflags '-static'" \
    cmd/protocol-engine/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=go-builder /build/protocol-engine /usr/local/bin/
EXPOSE 8080 8081 8082
CMD ["protocol-engine"]
```

### 3. Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: protocol-engine
spec:
  replicas: 3
  selector:
    matchLabels:
      app: protocol-engine
  template:
    metadata:
      labels:
        app: protocol-engine
    spec:
      containers:
      - name: protocol-engine
        image: protocol-engine:latest
        resources:
          requests:
            memory: "128Mi"  # Rust efficiency
            cpu: "100m"
          limits:
            memory: "512Mi" 
            cpu: "500m"
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /health/live 
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 20
```

This hybrid architecture provides the optimal balance of Go's excellent ecosystem integration with Rust's performance and safety guarantees, specifically designed for the critical healthcare applications of the Protocol Engine.