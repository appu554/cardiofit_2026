# Developer Onboarding Guide - Medication Service V2

Welcome to the Medication Service V2 development team! This guide will help you get up to speed with the Go/Rust architecture and clinical workflow patterns.

## Table of Contents
- [Project Overview](#project-overview)
- [Development Environment Setup](#development-environment-setup)
- [Architecture Deep Dive](#architecture-deep-dive)
- [Development Workflow](#development-workflow)
- [Testing Guidelines](#testing-guidelines)
- [Code Review Process](#code-review-process)
- [Clinical Domain Knowledge](#clinical-domain-knowledge)
- [Debugging & Troubleshooting](#debugging--troubleshooting)
- [Performance Considerations](#performance-considerations)
- [Getting Help](#getting-help)

## Project Overview

### What is Medication Service V2?

Medication Service V2 is a high-performance rewrite of the clinical medication management system, implementing the Recipe & Snapshot architecture pattern for enhanced safety and performance.

**Key Improvements over V1 (Python):**
- **Performance**: <250ms end-to-end processing (vs ~800ms in V1)
- **Safety**: Immutable clinical snapshots prevent data inconsistencies
- **Scalability**: Go concurrency handles 10x more concurrent requests
- **Reliability**: Rust's memory safety eliminates clinical calculation errors

### Architecture at a Glance

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Go Main Service │────│ Rust Clinical   │────│ Knowledge Bases │
│ • Orchestration │    │ Engine          │    │ • Drug Rules    │
│ • Recipe Logic  │    │ • Calculations  │    │ • Guidelines    │
│ • API Layer     │    │ • Safety Checks │    │ • Evidence      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

**4-Phase Workflow:**
1. **Ingestion & Recipe Resolution**: Determine what clinical data is needed
2. **Context Assembly**: Create immutable snapshot of patient data
3. **Clinical Intelligence**: Perform calculations and generate options
4. **Proposal Generation**: Create final medication recommendations

## Development Environment Setup

### 1. Prerequisites Installation

**Go Development Environment:**
```bash
# Install Go 1.21+
curl -LO https://golang.org/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install Go tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**Rust Development Environment:**
```bash
# Install Rust 1.70+
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Install Rust tools
rustup component add rustfmt clippy
cargo install cargo-audit cargo-tarpaulin cargo-watch
```

**Database & Tools:**
```bash
# PostgreSQL
sudo apt install postgresql-15 postgresql-client-15

# Redis
sudo apt install redis-server redis-tools

# Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Protocol Buffers
sudo apt install protobuf-compiler
```

### 2. Repository Setup

```bash
# Clone the repository
git clone https://github.com/clinical-platform/cardiofit.git
cd cardiofit/backend/services/medication-service-v2

# Set up Go workspace
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Initialize Go modules
go mod download

# Install pre-commit hooks
pre-commit install
```

### 3. Local Development Services

```bash
# Start infrastructure services
make dev-infrastructure

# This starts:
# - PostgreSQL on port 5434
# - Redis on port 6381  
# - Jaeger on port 16686
# - Prometheus on port 9090
```

### 4. IDE Configuration

**VS Code Extensions:**
- Go (official Google extension)
- Rust Analyzer
- REST Client
- Docker
- PostgreSQL

**IntelliJ/GoLand:**
- Go plugin
- Rust plugin
- Database Tools
- Docker plugin

## Architecture Deep Dive

### 1. Directory Structure

```
medication-service-v2/
├── cmd/medication-server/          # Main application entry point
├── internal/
│   ├── domain/                     # Domain models and business logic
│   │   ├── medication.go           # Core medication entities
│   │   ├── recipe.go              # Recipe and workflow models
│   │   └── snapshot.go            # Clinical snapshot models
│   ├── application/                # Application services (use cases)
│   │   ├── medication_service.go   # Main service orchestrator
│   │   ├── recipe_resolver.go     # Recipe resolution logic
│   │   └── clinical_intelligence.go # Clinical decision logic
│   ├── infrastructure/             # External integrations
│   │   ├── context_gateway.go     # Context Gateway client
│   │   ├── cache_manager.go       # Redis cache management
│   │   └── database.go            # PostgreSQL connections
│   └── interfaces/                 # API layer
│       ├── http/                  # REST API handlers
│       └── grpc/                  # gRPC service handlers
├── pkg/                           # Shared packages
├── flow2-go-engine-v2/           # Clinical orchestration engine (Go)
├── flow2-rust-engine-v2/         # High-performance calculations (Rust)
├── knowledge-bases-v2/           # Clinical knowledge services
├── configs/                      # Configuration files
├── migrations/                   # Database migrations
├── tests/                        # Test files
└── docs/                         # Documentation
```

### 2. Key Concepts

**Recipe & Snapshot Pattern:**
```go
// Recipe defines what clinical data is needed
type WorkflowRecipe struct {
    RecipeID         string            `json:"recipe_id"`
    RequiredFields   []string          `json:"required_fields"`
    FreshnessReqs    map[string]time.Duration `json:"freshness_requirements"`
    TTLSeconds       int64             `json:"ttl_seconds"`
}

// Snapshot is an immutable view of patient data
type ClinicalSnapshot struct {
    ID           string                 `json:"id"`
    PatientID    string                 `json:"patient_id"`
    Data         map[string]interface{} `json:"data"`
    Checksum     string                 `json:"checksum"`
    CreatedAt    time.Time              `json:"created_at"`
    ExpiresAt    time.Time              `json:"expires_at"`
    Signature    string                 `json:"signature"`
}
```

**4-Phase Workflow Implementation:**
```go
func (s *MedicationService) ProcessMedicationRequest(
    ctx context.Context, 
    request *MedicationRequest,
) (*MedicationProposal, error) {
    // Phase 1: Recipe Resolution
    phase1Result, err := s.Phase1IngestAndResolve(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("phase 1 failed: %w", err)
    }
    
    // Phase 2: Snapshot Creation
    snapshot, err := s.Phase2AssembleContext(ctx, phase1Result.Recipe, request.PatientID)
    if err != nil {
        return nil, fmt.Errorf("phase 2 failed: %w", err)
    }
    
    // Phase 3: Clinical Intelligence
    proposals, err := s.Phase3ClinicalIntelligence(ctx, phase1Result.Manifest, snapshot)
    if err != nil {
        return nil, fmt.Errorf("phase 3 failed: %w", err)
    }
    
    // Phase 4: Final Proposal
    return s.Phase4GenerateProposal(ctx, proposals, snapshot, phase1Result.Manifest)
}
```

## Development Workflow

### 1. Feature Development Process

**Step 1: Create Feature Branch**
```bash
git checkout -b feature/enhance-recipe-resolution
```

**Step 2: Implement Feature (TDD Approach)**
```bash
# 1. Write failing tests
make test  # Should fail

# 2. Write minimal implementation
make test  # Should pass

# 3. Refactor and optimize
make test  # Should still pass
```

**Step 3: Run Full Test Suite**
```bash
# Unit tests
make test-unit

# Integration tests  
make test-integration

# Performance tests
make test-performance

# All tests
make test-all
```

**Step 4: Code Quality Checks**
```bash
# Go linting and formatting
make lint-go
make format-go

# Rust linting and formatting
make lint-rust
make format-rust

# Security audit
make audit

# All quality checks
make quality-check
```

### 2. Local Development Commands

```bash
# Development server with hot reload
make dev

# Run specific service components
make run-go-engine      # Flow2 Go engine
make run-rust-engine    # Rust clinical engine
make run-knowledge-bases # KB services

# Database operations
make db-migrate         # Run pending migrations
make db-reset          # Reset database
make db-seed           # Load test data

# Debugging
make debug             # Start with debugger
make logs              # View service logs
make metrics           # View metrics dashboard
```

### 3. Working with Multiple Services

Since the medication service is composed of multiple components, you'll often need to work across Go, Rust, and knowledge base services:

**Terminal 1 - Main Go Service:**
```bash
make run-go-service
# Runs on port 8005
```

**Terminal 2 - Rust Clinical Engine:**
```bash
cd flow2-rust-engine-v2
cargo watch -x run
# Runs on port 8095
```

**Terminal 3 - Knowledge Bases:**
```bash
make run-knowledge-bases
# KB Drug Rules: port 8086
# KB Guidelines: port 8089
```

**Terminal 4 - Infrastructure:**
```bash
make dev-infrastructure
# PostgreSQL: 5434, Redis: 6381
```

## Testing Guidelines

### 1. Testing Philosophy

We follow the testing pyramid approach:

```
        /\
       /  \    E2E Tests (10%)
      /____\   Integration Tests (20%)
     /      \  Unit Tests (70%)
    /________\
```

**Unit Tests**: Test individual functions and methods in isolation
**Integration Tests**: Test service interactions and database operations
**E2E Tests**: Test complete workflows through the API

### 2. Writing Unit Tests

**Go Unit Test Example:**
```go
// internal/application/recipe_resolver_test.go
func TestRecipeResolver_ResolveWorkflowRecipe(t *testing.T) {
    tests := []struct {
        name           string
        protocolID     string
        contextNeeds   *domain.ContextNeeds
        expected       *domain.WorkflowRecipe
        expectedErr    string
    }{
        {
            name:       "successful hypertension recipe resolution",
            protocolID: "hypertension-standard",
            contextNeeds: &domain.ContextNeeds{
                CalculationFields: []string{"weight", "age"},
                SafetyFields:     []string{"allergies"},
            },
            expected: &domain.WorkflowRecipe{
                ProtocolID: "hypertension-standard",
                RequiredFields: []string{
                    "demographics.weight_kg",
                    "demographics.age_years", 
                    "allergies.drug_allergies",
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resolver := setupTestRecipeResolver(t)
            
            result, err := resolver.ResolveWorkflowRecipe(
                context.Background(),
                tt.protocolID,
                tt.contextNeeds,
                &domain.PatientCharacteristics{},
            )
            
            if tt.expectedErr != "" {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedErr)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.expected.ProtocolID, result.ProtocolID)
                assert.ElementsMatch(t, tt.expected.RequiredFields, result.RequiredFields)
            }
        })
    }
}
```

**Rust Unit Test Example:**
```rust
// flow2-rust-engine-v2/src/calculations.rs
#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_weight_based_dose_calculation() {
        let calculator = WeightBasedCalculator::new();
        let medication = create_test_medication("lisinopril");
        let patient_data = create_test_patient_data(70.5); // 70.5kg
        
        let result = calculator.calculate_dose(&medication, &patient_data).await;
        
        assert!(result.is_ok());
        let dose = result.unwrap();
        assert_eq!(dose.calculated_dose, 10.0); // 70.5kg * 0.142mg/kg ≈ 10mg
        assert_eq!(dose.rounded_dose, 10.0);
        assert_eq!(dose.calculation_method, "weight_based");
    }

    #[test]
    fn test_renal_dose_adjustment() {
        let adjuster = RenalDoseAdjuster::new();
        let base_dose = 20.0;
        let creatinine_clearance = 45.0; // Moderate renal impairment
        
        let adjusted = adjuster.adjust_dose(base_dose, creatinine_clearance);
        
        assert_eq!(adjusted.dose, 10.0); // 50% reduction for CrCl 30-60
        assert_eq!(adjusted.adjustment_factor, 0.5);
        assert!(adjusted.reason.contains("renal"));
    }
}
```

### 3. Integration Testing

**Database Integration Test:**
```go
// tests/integration/database_test.go
func TestMedicationRepository_Integration(t *testing.T) {
    db := setupTestDatabase(t)
    defer teardownTestDatabase(t, db)
    
    repo := infrastructure.NewMedicationRepository(db)
    
    // Create test medication
    medication := &domain.Medication{
        RxNormCode:  "123456",
        GenericName: "lisinopril",
        BrandName:   "Prinivil",
    }
    
    // Save medication
    err := repo.SaveMedication(context.Background(), medication)
    require.NoError(t, err)
    assert.NotEmpty(t, medication.ID)
    
    // Retrieve medication
    retrieved, err := repo.GetMedicationByRxNorm(context.Background(), "123456")
    require.NoError(t, err)
    assert.Equal(t, medication.GenericName, retrieved.GenericName)
}
```

### 4. Performance Testing

**Load Test Example:**
```go
// tests/performance/load_test.go
func TestMedicationProposal_LoadTest(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    service := setupTestService(t)
    
    // Test parameters
    concurrency := 50
    requestsPerWorker := 20
    targetLatency := 250 * time.Millisecond
    
    var wg sync.WaitGroup
    latencies := make(chan time.Duration, concurrency*requestsPerWorker)
    
    // Start worker goroutines
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            for j := 0; j < requestsPerWorker; j++ {
                start := time.Now()
                
                request := createTestRequest()
                _, err := service.ProcessMedicationRequest(context.Background(), request)
                require.NoError(t, err)
                
                latencies <- time.Since(start)
            }
        }()
    }
    
    wg.Wait()
    close(latencies)
    
    // Analyze results
    var totalLatency time.Duration
    count := 0
    var maxLatency time.Duration
    
    for latency := range latencies {
        totalLatency += latency
        count++
        if latency > maxLatency {
            maxLatency = latency
        }
    }
    
    avgLatency := totalLatency / time.Duration(count)
    
    t.Logf("Load test results:")
    t.Logf("- Requests: %d", count)
    t.Logf("- Average latency: %v", avgLatency)
    t.Logf("- Max latency: %v", maxLatency)
    
    // Assert performance targets
    assert.Less(t, avgLatency, targetLatency, "Average latency should be under 250ms")
    assert.Less(t, maxLatency, 2*targetLatency, "Max latency should be under 500ms")
}
```

## Code Review Process

### 1. Pre-Review Checklist

Before submitting a PR, ensure you've completed:

- [ ] All tests pass (`make test-all`)
- [ ] Code is properly formatted (`make format`)
- [ ] Linting passes (`make lint`)
- [ ] Security audit passes (`make audit`)
- [ ] Documentation updated (if needed)
- [ ] Performance impact assessed
- [ ] Breaking changes documented

### 2. Code Review Guidelines

**For Reviewers:**

**Focus Areas:**
1. **Clinical Safety**: Ensure calculations are correct and safe
2. **Performance**: Check for potential bottlenecks
3. **Error Handling**: Verify proper error propagation
4. **Testing**: Ensure adequate test coverage
5. **Security**: Look for potential vulnerabilities

**Review Questions:**
- Does this code handle patient data safely?
- Are clinical calculations mathematically correct?
- Could this change affect system performance?
- Are error conditions properly handled?
- Is the code well-tested?

**For Authors:**

**PR Description Should Include:**
- Clear description of changes
- Clinical rationale (if applicable)
- Performance impact assessment
- Testing approach
- Deployment considerations

### 3. Clinical Code Review

When reviewing clinical logic, pay special attention to:

**Dose Calculations:**
```go
// ❌ Dangerous - no bounds checking
func calculateDose(weightKg float64, dosePerKg float64) float64 {
    return weightKg * dosePerKg
}

// ✅ Safe - includes validation and bounds
func calculateDose(weightKg float64, dosePerKg float64) (float64, error) {
    if weightKg <= 0 || weightKg > 500 {
        return 0, fmt.Errorf("invalid weight: %f kg", weightKg)
    }
    if dosePerKg <= 0 || dosePerKg > 100 {
        return 0, fmt.Errorf("invalid dose per kg: %f", dosePerKg)
    }
    
    dose := weightKg * dosePerKg
    
    // Apply maximum dose constraints
    if dose > 100 { // Example max dose
        return 100, nil
    }
    
    return dose, nil
}
```

**Data Validation:**
```go
// ✅ Always validate clinical data
func validatePatientData(data *PatientData) error {
    if data.WeightKg <= 0 || data.WeightKg > 500 {
        return ErrInvalidWeight
    }
    if data.AgeYears < 0 || data.AgeYears > 120 {
        return ErrInvalidAge
    }
    // Additional validations...
    return nil
}
```

## Clinical Domain Knowledge

### 1. Key Clinical Concepts

**Medication Dosing Methods:**

1. **Fixed Dose**: Same dose for all patients (e.g., aspirin 81mg daily)
2. **Weight-Based**: Dose calculated per kg body weight (e.g., 0.1mg/kg)
3. **BSA-Based**: Dose calculated per m² body surface area (chemotherapy)
4. **AUC-Based**: Dose based on drug exposure area under curve

**Important Considerations:**

- **Renal Adjustment**: Reduce dose for kidney impairment
- **Hepatic Adjustment**: Reduce dose for liver disease
- **Age-Based**: Different dosing for pediatric/geriatric
- **Drug Interactions**: Medications affecting each other

### 2. Clinical Validation Rules

```go
// Example clinical validation
func validateMedicationProposal(proposal *MedicationProposal, patient *PatientData) error {
    // Age-based contraindications
    if patient.AgeYears < 18 && proposal.Medication.IsAdultOnly {
        return ErrPediatricContraindication
    }
    
    // Allergy checking
    for _, allergy := range patient.Allergies {
        if proposal.Medication.Contains(allergy.Drug) {
            return ErrAllergyContraindication
        }
    }
    
    // Dose range validation
    if proposal.Dose < proposal.Medication.MinDose || 
       proposal.Dose > proposal.Medication.MaxDose {
        return ErrDoseOutOfRange
    }
    
    return nil
}
```

### 3. Common Clinical Workflows

**Hypertension Management Example:**
1. Check current blood pressure readings
2. Review current medications for interactions
3. Consider patient factors (age, kidney function, diabetes)
4. Select appropriate medication class (ACE-I, ARB, diuretic, CCB)
5. Calculate appropriate starting dose
6. Set monitoring requirements

**Antibiotic Selection Example:**
1. Identify infection type and location
2. Consider local resistance patterns
3. Review patient allergies
4. Check kidney/liver function for dose adjustment
5. Select appropriate antibiotic and duration

## Debugging & Troubleshooting

### 1. Common Issues and Solutions

**Issue: Service Won't Start**
```bash
# Check logs
make logs

# Common causes:
# - Database connection failure
# - Port conflicts
# - Missing environment variables

# Solutions:
# 1. Verify database is running
docker ps | grep postgres-v2

# 2. Check port availability
netstat -tulpn | grep :8005

# 3. Validate configuration
make validate-config
```

**Issue: High Latency**
```bash
# Check performance metrics
curl http://localhost:8005/metrics | grep duration

# Profile the application
go tool pprof http://localhost:8005/debug/pprof/profile

# Check database performance
make db-analyze-slow-queries
```

**Issue: Clinical Calculation Errors**
```go
// Add debug logging to calculations
func (c *DoseCalculator) Calculate(ctx context.Context, req *CalculationRequest) (*DoseResult, error) {
    log := logger.FromContext(ctx)
    log.Debug("Starting dose calculation",
        zap.String("medication", req.Medication.Name),
        zap.Float64("weight", req.PatientWeight),
        zap.String("method", req.Method))
    
    result, err := c.performCalculation(req)
    
    log.Debug("Calculation completed",
        zap.Float64("calculated_dose", result.Dose),
        zap.Error(err))
    
    return result, err
}
```

### 2. Debugging Tools

**Go Debugging:**
```bash
# Delve debugger
dlv debug ./cmd/medication-server

# Profile CPU usage
go tool pprof http://localhost:8005/debug/pprof/profile

# Profile memory usage
go tool pprof http://localhost:8005/debug/pprof/heap

# Trace execution
go tool trace trace.out
```

**Rust Debugging:**
```bash
# Enable debug logging in Rust
RUST_LOG=debug cargo run

# Use rust-gdb for debugging
rust-gdb target/debug/clinical-engine

# Memory debugging with Valgrind
valgrind --tool=memcheck ./target/debug/clinical-engine
```

### 3. Monitoring and Alerting

**Key Metrics to Monitor:**
- Request latency (95th percentile < 250ms)
- Error rate (< 0.1%)
- Throughput (> 100 RPS)
- Database connection pool usage
- Memory usage
- CPU utilization

**Setting Up Alerts:**
```yaml
# prometheus-alerts.yml
- alert: MedicationServiceHighLatency
  expr: histogram_quantile(0.95, rate(medication_v2_request_duration_seconds_bucket[5m])) > 0.25
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Medication Service V2 high latency"
    description: "95th percentile latency is {{ $value }} seconds"
```

## Performance Considerations

### 1. Go Performance Best Practices

**Efficient Memory Usage:**
```go
// ❌ Creates garbage
func processPatients(patients []Patient) []Result {
    results := []Result{}
    for _, patient := range patients {
        result := processPatient(patient)
        results = append(results, result)
    }
    return results
}

// ✅ Pre-allocate slice
func processPatients(patients []Patient) []Result {
    results := make([]Result, 0, len(patients))
    for _, patient := range patients {
        result := processPatient(patient)
        results = append(results, result)
    }
    return results
}
```

**Effective Goroutine Usage:**
```go
// ✅ Bounded concurrency for clinical calculations
func (s *ClinicalService) ProcessBatch(ctx context.Context, requests []Request) ([]Response, error) {
    const maxConcurrency = 10
    semaphore := make(chan struct{}, maxConcurrency)
    
    var wg sync.WaitGroup
    responses := make([]Response, len(requests))
    
    for i, req := range requests {
        wg.Add(1)
        go func(index int, request Request) {
            defer wg.Done()
            
            semaphore <- struct{}{} // Acquire
            defer func() { <-semaphore }() // Release
            
            responses[index] = s.processRequest(ctx, request)
        }(i, req)
    }
    
    wg.Wait()
    return responses, nil
}
```

### 2. Rust Performance Optimization

**Zero-Copy String Handling:**
```rust
// ❌ Unnecessary allocations
fn process_medication_name(name: String) -> String {
    name.to_uppercase().replace(" ", "_")
}

// ✅ Use string slices where possible
fn process_medication_name(name: &str) -> String {
    name.to_uppercase().replace(" ", "_")
}
```

**Efficient Data Structures:**
```rust
use fnv::FnvHashMap; // Faster than std::HashMap for small keys

// ✅ Use appropriate data structures
fn build_drug_lookup() -> FnvHashMap<&'static str, DrugInfo> {
    let mut lookup = FnvHashMap::default();
    // populate...
    lookup
}
```

### 3. Database Performance

**Efficient Queries:**
```sql
-- ❌ Inefficient query
SELECT * FROM medications 
WHERE generic_name LIKE '%insulin%'
ORDER BY created_at DESC;

-- ✅ Optimized with proper indexing
SELECT medication_id, generic_name, brand_name 
FROM medications 
WHERE generic_name_search @@ to_tsquery('insulin')
ORDER BY ts_rank(generic_name_search, to_tsquery('insulin')) DESC
LIMIT 20;
```

**Connection Pooling:**
```go
// ✅ Proper connection pool configuration
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(20)
db.SetConnMaxLifetime(time.Hour)
```

## Getting Help

### 1. Internal Resources

**Team Contacts:**
- **Architecture Questions**: @senior-architect
- **Clinical Domain**: @clinical-lead
- **Go/Performance**: @go-expert
- **Rust/Calculations**: @rust-expert
- **DevOps/Deployment**: @devops-lead

**Documentation:**
- [Architecture Decision Records](./adrs/)
- [Clinical Protocols](./clinical-protocols/)
- [Performance Benchmarks](./benchmarks/)
- [Troubleshooting Guide](./troubleshooting.md)

### 2. External Resources

**Go Development:**
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [High Performance Go Workshop](https://dave.cheney.net/high-performance-go-workshop)

**Rust Development:**
- [The Rust Programming Language](https://doc.rust-lang.org/book/)
- [Rust Performance Book](https://nnethercote.github.io/perf-book/)
- [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/)

**Clinical Computing:**
- [FHIR Specification](https://www.hl7.org/fhir/)
- [RxNorm Database](https://www.nlm.nih.gov/research/umls/rxnorm/)
- [Clinical Decision Support](https://www.ahrq.gov/cds/index.html)

### 3. Development Support

**Getting Unstuck:**
1. Check the troubleshooting guide first
2. Search existing issues and PRs
3. Ask in the team chat channel
4. Schedule a pair programming session
5. Create a GitHub issue with details

**Code Review Support:**
- Tag relevant experts based on the change type
- Include clinical context in your PR description
- Don't hesitate to ask for architecture guidance
- Request performance review for critical paths

**Learning Opportunities:**
- Weekly architecture discussions (Fridays 2pm)
- Monthly clinical domain training
- Quarterly performance optimization workshops
- Annual clinical computing conference

Welcome to the team! The medication service is critical healthcare infrastructure, and your contributions help ensure safe, effective clinical decision-making. Don't hesitate to ask questions - we're here to help you succeed.