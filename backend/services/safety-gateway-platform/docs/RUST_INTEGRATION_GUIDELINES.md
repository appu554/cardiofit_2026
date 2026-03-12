# Rust Integration Guidelines
## Development Standards for Protocol Engine Implementation

### Executive Summary
This document provides comprehensive guidelines for implementing Rust components within the Go-based Safety Gateway Platform. These guidelines ensure consistent, safe, and maintainable hybrid Go-Rust development practices.

---

## Code Organization Standards

### Directory Structure
```
safety-gateway-platform/
├── cmd/
│   └── protocol-engine/
│       └── main.go                    # Go main entry point
├── internal/
│   ├── protocol/
│   │   ├── engine.go                  # Go orchestration layer
│   │   ├── rust_bridge.go             # FFI interface
│   │   └── types.go                   # Shared type definitions
│   └── engines/
│       └── rust_engines/              # Rust implementation
│           ├── Cargo.toml
│           ├── build.rs               # Build script for C bindings
│           ├── protocol_engine.h      # C header for FFI
│           └── src/
│               ├── lib.rs             # Rust library entry point
│               ├── ffi/               # FFI interface implementations
│               ├── protocol/          # Protocol engine core
│               ├── rules/             # Rule evaluation engine
│               ├── temporal/          # Temporal constraint engine
│               └── state/             # State machine implementation
├── docs/                              # Documentation
├── migrations/                        # Database migrations
├── protocols/                         # Protocol definitions
└── tests/
    ├── integration/                   # Cross-language integration tests
    └── benchmarks/                    # Performance benchmarks
```

### Naming Conventions

#### Rust Naming Standards
```rust
// Module names: snake_case
pub mod protocol_engine;
pub mod rule_evaluator;

// Struct names: PascalCase
pub struct ProtocolEngine {
    rule_engine: RuleEngine,
    state_manager: StateManager,
}

// Function names: snake_case
pub fn evaluate_protocol(&self, request: &EvaluationRequest) -> Result<EvaluationResult, ProtocolError> {
    // Implementation
}

// Constants: SCREAMING_SNAKE_CASE
pub const MAX_RULE_DEPTH: usize = 10;
pub const DEFAULT_TIMEOUT_MS: u64 = 5000;

// Error types: descriptive with Error suffix
#[derive(Debug, thiserror::Error)]
pub enum ProtocolEngineError {
    #[error("Rule evaluation failed: {0}")]
    RuleEvaluationError(String),
    
    #[error("Invalid protocol: {protocol_id}")]
    InvalidProtocol { protocol_id: String },
}
```

#### Go Naming Standards (for Rust integration)
```go
// FFI interface types: prefixed with 'Rust'
type RustProtocolEngine struct {
    engine unsafe.Pointer
    config *RustEngineConfig
}

// FFI wrapper functions: descriptive names
func (r *RustProtocolEngine) EvaluateProtocol(request *EvaluationRequest) (*EvaluationResult, error) {
    // Implementation
}

// Error types: consistent with Rust counterparts
type ProtocolEngineError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Source  string `json:"source"` // "rust" or "go"
}
```

---

## FFI Interface Standards

### Memory Management Rules

#### Rule 1: Ownership Transfer
```rust
// Rust: Always return owned data for FFI
#[no_mangle]
pub extern "C" fn rust_evaluate_protocol(
    engine: *mut ProtocolEngine,
    request_json: *const c_char,
) -> *mut RustResult {
    // Transfer ownership to C caller
    Box::into_raw(Box::new(result))
}

// Corresponding free function (mandatory)
#[no_mangle]
pub extern "C" fn rust_free_result(result: *mut RustResult) {
    if !result.is_null() {
        unsafe { Box::from_raw(result) }; // Automatic cleanup
    }
}
```

```go
// Go: Always free Rust-allocated memory
func (r *RustProtocolEngine) EvaluateProtocol(request *EvaluationRequest) (*EvaluationResult, error) {
    // Convert to C string
    cRequest := C.CString(requestJson)
    defer C.free(unsafe.Pointer(cRequest)) // Go allocation - Go cleanup
    
    // Call Rust function
    cResult := C.rust_evaluate_protocol(r.engine, cRequest)
    defer C.rust_free_result(cResult) // Rust allocation - Rust cleanup
    
    // Process result...
}
```

#### Rule 2: Error Handling Across FFI
```rust
// Rust: Use Result types internally, convert to error codes for FFI
#[repr(C)]
pub struct RustResult {
    success: bool,
    error_code: i32,
    data: *mut c_char,
    error_message: *mut c_char,
}

#[no_mangle]
pub extern "C" fn rust_evaluate_protocol(
    engine: *mut ProtocolEngine,
    request: *const c_char,
) -> RustResult {
    match internal_evaluate(engine, request) {
        Ok(result) => RustResult {
            success: true,
            error_code: 0,
            data: string_to_c_char(serde_json::to_string(&result).unwrap()),
            error_message: std::ptr::null_mut(),
        },
        Err(e) => RustResult {
            success: false,
            error_code: e.error_code(),
            data: std::ptr::null_mut(),
            error_message: string_to_c_char(e.to_string()),
        },
    }
}
```

```go
// Go: Check error codes and handle appropriately
func (r *RustProtocolEngine) EvaluateProtocol(request *EvaluationRequest) (*EvaluationResult, error) {
    cResult := C.rust_evaluate_protocol(r.engine, cRequest)
    defer r.freeRustResult(cResult)
    
    if !cResult.success {
        errorMsg := C.GoString(cResult.error_message)
        return nil, &ProtocolEngineError{
            Code:    int(cResult.error_code),
            Message: errorMsg,
            Source:  "rust",
        }
    }
    
    // Process successful result...
}
```

#### Rule 3: Thread Safety
```rust
// Rust: Use thread-safe types for shared state
pub struct ProtocolEngine {
    rules: Arc<RwLock<HashMap<String, CompiledRule>>>,
    state_cache: Arc<Mutex<LruCache<String, ProtocolState>>>,
}

// Thread-safe evaluation
impl ProtocolEngine {
    pub fn evaluate(&self, request: &EvaluationRequest) -> Result<EvaluationResult, ProtocolError> {
        // Read locks for concurrent access
        let rules = self.rules.read().unwrap();
        let mut cache = self.state_cache.lock().unwrap();
        
        // Evaluation logic...
    }
}
```

```go
// Go: Protect FFI calls with mutexes if needed
type RustProtocolEngine struct {
    engine unsafe.Pointer
    mutex  sync.RWMutex    // Protect engine access if Rust engine isn't thread-safe
}

func (r *RustProtocolEngine) EvaluateProtocol(request *EvaluationRequest) (*EvaluationResult, error) {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    
    // FFI call with protection
    return r.evaluateUnsafe(request)
}
```

---

## Performance Guidelines

### Optimization Strategies

#### 1. Memory Allocation Minimization
```rust
// Use object pools for frequent allocations
pub struct EvaluationContextPool {
    contexts: Mutex<Vec<EvaluationContext>>,
    total_allocated: AtomicUsize,
}

impl EvaluationContextPool {
    pub fn acquire(&self) -> PooledContext {
        let mut contexts = self.contexts.lock().unwrap();
        match contexts.pop() {
            Some(context) => PooledContext::Reused(context),
            None => {
                self.total_allocated.fetch_add(1, Ordering::Relaxed);
                PooledContext::New(EvaluationContext::new())
            }
        }
    }
}

// RAII wrapper for automatic return to pool
pub struct PooledContext {
    context: Option<EvaluationContext>,
    pool: Arc<EvaluationContextPool>,
}

impl Drop for PooledContext {
    fn drop(&mut self) {
        if let Some(context) = self.context.take() {
            let mut contexts = self.pool.contexts.lock().unwrap();
            contexts.push(context.reset()); // Reset and return to pool
        }
    }
}
```

#### 2. Batch Processing
```rust
// Process multiple requests in single FFI call
#[no_mangle]
pub extern "C" fn rust_evaluate_batch(
    engine: *mut ProtocolEngine,
    requests_json: *const c_char,
    count: usize,
) -> *mut RustBatchResult {
    let requests: Vec<EvaluationRequest> = serde_json::from_str(requests_str).unwrap();
    
    // Parallel processing with Rayon
    let results: Vec<EvaluationResult> = requests
        .par_iter()
        .map(|request| engine.evaluate(request))
        .collect::<Result<Vec<_>, _>>()
        .unwrap();
    
    Box::into_raw(Box::new(RustBatchResult::new(results)))
}
```

```go
// Go: Use batch processing for better performance
func (r *RustProtocolEngine) EvaluateBatch(requests []*EvaluationRequest) ([]*EvaluationResult, error) {
    if len(requests) == 1 {
        // Single request - use regular path
        result, err := r.EvaluateProtocol(requests[0])
        return []*EvaluationResult{result}, err
    }
    
    // Batch processing for multiple requests
    requestsJson, err := json.Marshal(requests)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal batch requests: %w", err)
    }
    
    cRequests := C.CString(string(requestsJson))
    defer C.free(unsafe.Pointer(cRequests))
    
    cResult := C.rust_evaluate_batch(r.engine, cRequests, C.size_t(len(requests)))
    defer C.rust_free_batch_result(cResult)
    
    // Process batch results...
}
```

#### 3. Compilation Optimizations
```toml
# Cargo.toml - Optimized for clinical performance
[profile.release]
opt-level = 3                    # Maximum optimization
lto = "fat"                     # Fat LTO for better cross-crate optimization
codegen-units = 1               # Single codegen unit
panic = "abort"                 # Smaller binary, faster panics
overflow-checks = false         # Remove overflow checks in release

[profile.release.package."*"]
opt-level = 3                   # Optimize all dependencies

# Target-specific optimizations
[target.x86_64-unknown-linux-gnu]
rustflags = ["-C", "target-cpu=native", "-C", "target-feature=+avx2"]
```

---

## Error Handling Standards

### Error Type Hierarchy
```rust
// Base error type for all protocol engine errors
#[derive(Debug, thiserror::Error)]
pub enum ProtocolEngineError {
    #[error("Configuration error: {0}")]
    Configuration(#[from] ConfigError),
    
    #[error("Rule evaluation error: {0}")]
    RuleEvaluation(#[from] RuleError),
    
    #[error("State management error: {0}")]
    StateManagement(#[from] StateError),
    
    #[error("Temporal constraint error: {0}")]
    TemporalConstraint(#[from] TemporalError),
    
    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),
    
    #[error("FFI error: {message}")]
    FFI { message: String },
}

// Specific error types
#[derive(Debug, thiserror::Error)]
pub enum RuleError {
    #[error("Rule not found: {rule_id}")]
    RuleNotFound { rule_id: String },
    
    #[error("Rule compilation failed: {rule_id}, reason: {reason}")]
    CompilationFailed { rule_id: String, reason: String },
    
    #[error("Rule evaluation timeout: {rule_id}, timeout: {timeout_ms}ms")]
    EvaluationTimeout { rule_id: String, timeout_ms: u64 },
}

// Convert to FFI-compatible error codes
impl From<ProtocolEngineError> for i32 {
    fn from(error: ProtocolEngineError) -> i32 {
        match error {
            ProtocolEngineError::Configuration(_) => -2001,
            ProtocolEngineError::RuleEvaluation(RuleError::RuleNotFound { .. }) => -2002,
            ProtocolEngineError::RuleEvaluation(RuleError::CompilationFailed { .. }) => -2003,
            ProtocolEngineError::RuleEvaluation(RuleError::EvaluationTimeout { .. }) => -2004,
            ProtocolEngineError::StateManagement(_) => -2005,
            ProtocolEngineError::TemporalConstraint(_) => -2006,
            ProtocolEngineError::Serialization(_) => -2007,
            ProtocolEngineError::FFI { .. } => -2008,
        }
    }
}
```

### Go Error Mapping
```go
// Error mapping from Rust to Go
var rustErrorMap = map[int]string{
    -2001: "Configuration error",
    -2002: "Rule not found",
    -2003: "Rule compilation failed",
    -2004: "Rule evaluation timeout",
    -2005: "State management error",
    -2006: "Temporal constraint error", 
    -2007: "Serialization error",
    -2008: "FFI error",
}

type ProtocolEngineError struct {
    Code      int                    `json:"code"`
    Message   string                 `json:"message"`
    Source    string                 `json:"source"`
    Details   map[string]interface{} `json:"details,omitempty"`
    RequestID string                 `json:"request_id,omitempty"`
}

func (e *ProtocolEngineError) Error() string {
    return fmt.Sprintf("[%s:%d] %s", e.Source, e.Code, e.Message)
}

// Convert Rust error to Go error
func convertRustError(errorCode int, errorMessage string, requestID string) error {
    category, exists := rustErrorMap[errorCode]
    if !exists {
        category = "Unknown error"
    }
    
    return &ProtocolEngineError{
        Code:      errorCode,
        Message:   fmt.Sprintf("%s: %s", category, errorMessage),
        Source:    "rust",
        RequestID: requestID,
    }
}
```

---

## Testing Standards

### Unit Testing Guidelines

#### Rust Testing Standards
```rust
// Use property-based testing for critical algorithms
use proptest::prelude::*;

proptest! {
    #[test]
    fn rule_evaluation_deterministic(
        rule in arbitrary_rule(),
        context in arbitrary_evaluation_context(),
    ) {
        let engine = RuleEngine::new();
        let result1 = engine.evaluate(&rule, &context).unwrap();
        let result2 = engine.evaluate(&rule, &context).unwrap();
        prop_assert_eq!(result1, result2, "Rule evaluation must be deterministic");
    }
    
    #[test]
    fn temporal_constraints_monotonic(
        constraint in arbitrary_temporal_constraint(),
        base_time in 0u64..1_000_000_000u64,
        delta in 1u64..86400u64, // 1 second to 1 day
    ) {
        let engine = TemporalConstraintEngine::new();
        let context1 = create_context_at_time(base_time);
        let context2 = create_context_at_time(base_time + delta);
        
        let result1 = engine.evaluate_constraint(&constraint, &context1);
        let result2 = engine.evaluate_constraint(&constraint, &context2);
        
        // If constraint was satisfied at base_time and is time-bounded,
        // it should eventually become unsatisfied
        if matches!(constraint, TemporalConstraint::Window { .. }) {
            // Time monotonicity check
        }
    }
}

// Standard unit tests with comprehensive edge cases
#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_sepsis_bundle_state_transitions() {
        let mut state_machine = ProtocolStateMachine::sepsis_bundle();
        
        // Test initial state
        assert_eq!(state_machine.current_state(), "INACTIVE");
        
        // Test valid transition
        let event = ProtocolEvent::SepsisCriteriaMet {
            temperature: 38.5,
            lactate: 2.5,
        };
        
        let result = state_machine.transition(&event).unwrap();
        assert_eq!(result.to_state, "RECOGNITION");
        assert_eq!(state_machine.current_state(), "RECOGNITION");
        
        // Test invalid transition
        let invalid_event = ProtocolEvent::AntibioticsAdministered;
        let result = state_machine.transition(&invalid_event);
        assert!(result.is_err());
        assert_eq!(state_machine.current_state(), "RECOGNITION"); // State unchanged
    }
    
    #[test]
    fn test_rule_engine_performance() {
        let engine = RuleEngine::new();
        let rules = create_complex_rule_set(100);
        let context = create_large_evaluation_context();
        
        let start = std::time::Instant::now();
        let results = engine.evaluate_parallel(&rules, &context);
        let duration = start.elapsed();
        
        // Performance assertions
        assert_eq!(results.len(), 100);
        assert!(duration.as_millis() < 100, "Rule evaluation too slow: {}ms", duration.as_millis());
        
        // All results should be valid
        assert!(results.iter().all(|r| r.is_ok()));
    }
}
```

#### Go Integration Testing
```go
func TestRustIntegrationSafety(t *testing.T) {
    engine, err := NewRustProtocolEngine(testConfig())
    require.NoError(t, err)
    defer engine.Close()
    
    // Test memory safety with invalid inputs
    tests := []struct {
        name    string
        request *EvaluationRequest
        wantErr bool
    }{
        {
            name:    "nil request",
            request: nil,
            wantErr: true,
        },
        {
            name: "empty protocol ID",
            request: &EvaluationRequest{
                ProtocolID: "",
                RequestID:  "test-001",
            },
            wantErr: true,
        },
        {
            name: "malformed JSON in patient context",
            request: &EvaluationRequest{
                ProtocolID: "test-protocol",
                RequestID:  "test-001",
                // Intentionally malformed data to test error handling
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := engine.EvaluateProtocol(tt.request)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}

// Memory leak detection
func TestMemoryLeakPrevention(t *testing.T) {
    engine, err := NewRustProtocolEngine(testConfig())
    require.NoError(t, err)
    defer engine.Close()
    
    // Force garbage collection and measure initial memory
    runtime.GC()
    runtime.GC()
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)
    
    // Perform many evaluations to stress test memory management
    const iterations = 10000
    for i := 0; i < iterations; i++ {
        request := &EvaluationRequest{
            RequestID:  fmt.Sprintf("test-%d", i),
            ProtocolID: "test-protocol",
            PatientContext: PatientContext{
                PatientID: fmt.Sprintf("patient-%d", i),
                // Add other test data
            },
        }
        
        result, err := engine.EvaluateProtocol(request)
        require.NoError(t, err)
        require.NotNil(t, result)
        
        // Verify result is properly structured
        assert.NotEmpty(t, result.RequestID)
        assert.NotEmpty(t, result.Decision)
    }
    
    // Force garbage collection and measure final memory
    runtime.GC()
    runtime.GC()
    var m2 runtime.MemStats
    runtime.ReadMemStats(&m2)
    
    // Memory growth should be bounded (less than 50MB for 10k operations)
    memGrowth := int64(m2.HeapAlloc - m1.HeapAlloc)
    maxAllowedGrowth := int64(50 * 1024 * 1024) // 50MB
    
    assert.Less(t, memGrowth, maxAllowedGrowth, 
        "Memory growth too large: %d bytes (%.2f MB) for %d operations", 
        memGrowth, float64(memGrowth)/(1024*1024), iterations)
}
```

### Benchmark Testing
```rust
// Criterion-based performance benchmarks
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};

fn benchmark_rule_evaluation_scaling(c: &mut Criterion) {
    let engine = RuleEngine::new();
    
    let mut group = c.benchmark_group("rule_evaluation_scaling");
    
    for rule_count in [10, 50, 100, 500, 1000].iter() {
        group.throughput(Throughput::Elements(*rule_count as u64));
        group.bench_with_input(
            BenchmarkId::new("parallel", rule_count),
            rule_count,
            |b, &rule_count| {
                let rules = create_rule_set(rule_count);
                let context = create_test_context();
                b.iter(|| engine.evaluate_parallel(&rules, &context));
            },
        );
        
        group.bench_with_input(
            BenchmarkId::new("sequential", rule_count),
            rule_count,
            |b, &rule_count| {
                let rules = create_rule_set(rule_count);
                let context = create_test_context();
                b.iter(|| engine.evaluate_sequential(&rules, &context));
            },
        );
    }
    
    group.finish();
}

fn benchmark_ffi_overhead(c: &mut Criterion) {
    let engine = create_test_engine();
    let request = create_standard_request();
    
    c.bench_function("ffi_call_overhead", |b| {
        b.iter(|| {
            // Measure pure FFI call overhead
            let json = serde_json::to_string(&request).unwrap();
            let c_str = std::ffi::CString::new(json).unwrap();
            let result = unsafe { rust_evaluate_protocol(engine.as_ptr(), c_str.as_ptr()) };
            unsafe { rust_free_result(result) };
        });
    });
    
    c.bench_function("native_call", |b| {
        b.iter(|| {
            // Measure native Rust call for comparison
            engine.evaluate(&request).unwrap();
        });
    });
}

criterion_group!(
    benches, 
    benchmark_rule_evaluation_scaling,
    benchmark_ffi_overhead
);
criterion_main!(benches);
```

---

## Monitoring and Observability

### Metrics Collection
```rust
// Prometheus metrics in Rust
use prometheus::{Counter, Histogram, Gauge, register_counter, register_histogram, register_gauge};
use once_cell::sync::Lazy;

static PROTOCOL_EVALUATIONS: Lazy<Counter> = Lazy::new(|| {
    register_counter!("protocol_evaluations_total", "Total protocol evaluations")
        .expect("Failed to register counter")
});

static EVALUATION_DURATION: Lazy<Histogram> = Lazy::new(|| {
    register_histogram!(
        "protocol_evaluation_duration_seconds",
        "Protocol evaluation duration",
        vec![0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1.0]
    ).expect("Failed to register histogram")
});

static ACTIVE_PROTOCOL_STATES: Lazy<Gauge> = Lazy::new(|| {
    register_gauge!("protocol_active_states", "Number of active protocol states")
        .expect("Failed to register gauge")
});

// Instrument evaluation functions
impl ProtocolEngine {
    pub fn evaluate(&self, request: &EvaluationRequest) -> Result<EvaluationResult, ProtocolEngineError> {
        let timer = EVALUATION_DURATION.start_timer();
        PROTOCOL_EVALUATIONS.inc();
        
        let result = self.evaluate_internal(request);
        
        timer.observe_duration();
        
        // Record result-specific metrics
        match &result {
            Ok(eval_result) => {
                PROTOCOL_EVALUATIONS.with_label_values(&["success"]).inc();
                self.record_decision_metrics(&eval_result.decision);
            }
            Err(error) => {
                PROTOCOL_EVALUATIONS.with_label_values(&["error"]).inc();
                self.record_error_metrics(error);
            }
        }
        
        result
    }
}
```

### Structured Logging
```rust
// Structured logging with slog
use slog::{info, warn, error, debug, Logger};

pub struct InstrumentedProtocolEngine {
    engine: ProtocolEngine,
    logger: Logger,
}

impl InstrumentedProtocolEngine {
    pub fn evaluate(&self, request: &EvaluationRequest) -> Result<EvaluationResult, ProtocolEngineError> {
        debug!(self.logger, "Starting protocol evaluation";
               "request_id" => &request.request_id,
               "protocol_id" => &request.protocol_id,
               "patient_id" => &request.patient_context.patient_id);
        
        let start = std::time::Instant::now();
        let result = self.engine.evaluate(request);
        let duration = start.elapsed();
        
        match &result {
            Ok(eval_result) => {
                info!(self.logger, "Protocol evaluation completed";
                      "request_id" => &request.request_id,
                      "decision" => format!("{:?}", eval_result.decision),
                      "duration_ms" => duration.as_millis() as u64,
                      "triggered_protocols" => eval_result.triggered_protocols.len());
            }
            Err(error) => {
                error!(self.logger, "Protocol evaluation failed";
                       "request_id" => &request.request_id,
                       "error" => format!("{}", error),
                       "duration_ms" => duration.as_millis() as u64);
            }
        }
        
        result
    }
}
```

### Distributed Tracing
```rust
use opentelemetry::{global, trace::{TraceId, Span, Tracer}};

impl ProtocolEngine {
    pub fn evaluate_with_tracing(&self, request: &EvaluationRequest) -> Result<EvaluationResult, ProtocolEngineError> {
        let tracer = global::tracer("protocol-engine");
        let mut span = tracer.start("protocol_evaluation");
        
        // Add span attributes
        span.set_attribute(opentelemetry::KeyValue::new("protocol.request_id", request.request_id.clone()));
        span.set_attribute(opentelemetry::KeyValue::new("protocol.protocol_id", request.protocol_id.clone()));
        span.set_attribute(opentelemetry::KeyValue::new("protocol.patient_id", request.patient_context.patient_id.clone()));
        
        let result = self.evaluate_internal(request);
        
        // Record result in span
        match &result {
            Ok(eval_result) => {
                span.set_attribute(opentelemetry::KeyValue::new("protocol.decision", format!("{:?}", eval_result.decision)));
                span.set_attribute(opentelemetry::KeyValue::new("protocol.triggered_count", eval_result.triggered_protocols.len() as i64));
                span.set_status(opentelemetry::trace::Status::Ok);
            }
            Err(error) => {
                span.set_attribute(opentelemetry::KeyValue::new("protocol.error", error.to_string()));
                span.set_status(opentelemetry::trace::Status::error(error.to_string()));
            }
        }
        
        result
    }
}
```

---

## Security Guidelines

### Memory Safety Rules
1. **No unsafe code outside FFI boundaries**
2. **All FFI functions must validate pointer parameters**  
3. **Use `std::ptr::null()` checks for all C pointers**
4. **Implement proper Drop traits for all FFI resources**

### Data Validation
```rust
// Input validation for all external data
pub fn validate_evaluation_request(request: &EvaluationRequest) -> Result<(), ValidationError> {
    // Required field validation
    if request.request_id.is_empty() {
        return Err(ValidationError::MissingField("request_id"));
    }
    
    if request.protocol_id.is_empty() {
        return Err(ValidationError::MissingField("protocol_id"));
    }
    
    // Format validation
    if !is_valid_uuid(&request.request_id) {
        return Err(ValidationError::InvalidFormat {
            field: "request_id",
            expected: "UUID",
        });
    }
    
    // Business logic validation
    if !is_valid_protocol_id(&request.protocol_id) {
        return Err(ValidationError::InvalidValue {
            field: "protocol_id",
            reason: "Unknown protocol",
        });
    }
    
    // Patient context validation
    validate_patient_context(&request.patient_context)?;
    
    Ok(())
}
```

### Audit Trail Requirements
```rust
// Comprehensive audit logging for all operations
#[derive(Debug, Serialize)]
pub struct ProtocolAuditEvent {
    pub timestamp: SystemTime,
    pub request_id: String,
    pub operation: String,
    pub user_id: Option<String>,
    pub patient_id: String,
    pub protocol_id: String,
    pub decision: String,
    pub reasoning: Vec<String>,
    pub overrides: Vec<Override>,
    pub digital_signature: String,
}

impl ProtocolEngine {
    pub fn evaluate_with_audit(&self, request: &EvaluationRequest, user_context: &UserContext) -> Result<EvaluationResult, ProtocolEngineError> {
        let result = self.evaluate(request)?;
        
        // Create audit event
        let audit_event = ProtocolAuditEvent {
            timestamp: SystemTime::now(),
            request_id: request.request_id.clone(),
            operation: "protocol_evaluation".to_string(),
            user_id: user_context.user_id.clone(),
            patient_id: request.patient_context.patient_id.clone(),
            protocol_id: request.protocol_id.clone(),
            decision: format!("{:?}", result.decision),
            reasoning: result.reasoning.clone(),
            overrides: result.overrides.clone(),
            digital_signature: self.sign_result(&result)?,
        };
        
        // Log audit event
        self.audit_logger.log_event(&audit_event)?;
        
        Ok(result)
    }
}
```

These comprehensive guidelines ensure that the hybrid Go-Rust implementation maintains the highest standards of safety, performance, and maintainability required for clinical healthcare applications.