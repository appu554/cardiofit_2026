# Workflow Engine Go Service - Full Ecosystem Testing Guide

## 🎯 Overview

This document provides comprehensive testing documentation for the **Workflow Engine Go Service** within the Clinical Synthesis Hub CardioFit platform. The service implements an Advanced 3-Phase Pattern (Calculate → Validate → Commit) with real-time UI interaction capabilities and requires a complex ecosystem of external services for complete functionality testing.

## 🏗️ Architecture & Dependencies

### Core Service Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend UI (React/Vue)                 │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│          Apollo Federation Gateway (GraphQL)               │
│                    Port: 4000                              │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│           Workflow Engine Go Service                       │
│                    Port: 8020                              │
│     ┌─────────────┬─────────────┬─────────────┐            │
│     │  Calculate  │  Validate   │   Commit    │            │
│     │    Phase    │    Phase    │   Phase     │            │
│     └─────────────┴─────────────┴─────────────┘            │
└─────┬───────────┬─────────────┬─────────────┬──────────────┘
      │           │             │             │
      ▼           ▼             ▼             ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│Flow2 Go  │ │Safety    │ │Context   │ │Medication│
│Engine    │ │Gateway   │ │Gateway   │ │Service   │
│Port: 8080│ │Port: 8018│ │Port: 8016│ │Port: 8004│
└──────────┘ └──────────┘ └──────────┘ └──────────┘
```

## 📋 Required External Services

### Infrastructure Services (Critical Foundation)

| Service | Port | Location | Purpose | Status |
|---------|------|----------|---------|--------|
| **PostgreSQL** | 5432 | docker-compose | Workflow state persistence, audit trails | ✅ Ready |
| **Redis** | 6379 | docker-compose | Session state, real-time notifications | ✅ Ready |
| **Apollo Federation** | 4000 | `@apollo-federation/` | GraphQL gateway, UI coordination | ✅ Ready |

### Clinical Processing Engines (Core Workflow)

| Service | Port | Location | Purpose | Status |
|---------|------|----------|---------|--------|
| **Flow2 Go Engine** | 8080 | `@backend/services/medication-service/flow2-go-engine/` | Calculate Phase - medication intelligence | ❌ **BLOCKED** |
| **Flow2 Rust Engine** | 8090 | `@backend/services/medication-service/flow2-rust-engine/` | High-performance clinical rules | ✅ Ready |
| **Safety Gateway** | 8018 | `@backend/services/safety-gateway-platform/` | Validate Phase - clinical safety | ✅ Ready |
| **Context Gateway** | 8016 | `@backend/services/context-gateway-go/` | Clinical context assembly | ✅ Ready |

### Clinical Data Services (Supporting)

| Service | Port | Location | Purpose | Status |
|---------|------|----------|---------|--------|
| **Medication Service V2** | 8004 | `@backend/services/medication-service-v2/` | FHIR medication resources | ✅ Ready |
| **Auth Service** | 8001 | `@backend/services/auth-service/` | JWT authentication | ✅ Ready |
| **Patient Service** | 8003 | `@backend/services/patient-service/` | Patient demographics | ✅ Ready |

## 🚨 Critical Blockers

### 1. Flow2 Go Engine - Duplicate Type Definitions
**Error**: Multiple model files contain conflicting type definitions
```bash
internal/models/orb_models.go:88:6: ExecutionSummary redeclared
internal/models/phase1_models.go:159:6: PatientPreferences redeclared
internal/models/rust_models.go:38:6: ClinicalContext redeclared
internal/models/rust_models.go:187:6: too many errors
```

**Impact**: Blocks Calculate Phase functionality - **CRITICAL BLOCKER**

### 2. Workflow Engine - Constructor Mismatches
**Errors**:
```bash
NewUICoordinator: not enough arguments (missing apolloGatewayURL)
NewCommitOrchestrator: not enough arguments (missing 4 required parameters)
NewBatchProcessor: too many arguments in call
UICoordinator missing methods: RegisterWorkflowState, UpdateWorkflowPhase, SendNotification
```

**Impact**: Prevents service startup - **CRITICAL BLOCKER**

## 🚀 Full Ecosystem Testing Execution Plan

### Phase 1: Critical Compilation Fixes 🔥

#### 1.1 Fix Flow2 Go Engine Duplicate Types
```bash
cd backend/services/medication-service/flow2-go-engine

# Identify all duplicate types
find internal/models/ -name "*.go" -exec grep -l "type.*struct" {} \;

# Strategy:
# 1. Consolidate ExecutionSummary, PatientPreferences, ClinicalContext
# 2. Create canonical definitions in single model file
# 3. Update imports across affected packages
# 4. Test: go build -o bin/flow2-go-engine ./cmd/server
```

#### 1.2 Fix Workflow Engine Constructor Issues
```bash
cd backend/services/workflow-engine-go-service

# Fix strategic orchestrator constructor calls:
# - Add apolloGatewayURL parameter to NewUICoordinator
# - Add missing parameters to NewCommitOrchestrator
# - Fix NewBatchProcessor signature
# - Implement missing UICoordinator methods

# Test: go build -o bin/workflow-engine ./cmd/server/main.go
```

### Phase 2: Infrastructure Services Startup 🏗️

#### 2.1 Database & Cache Layer
```bash
# Start PostgreSQL and Redis
docker-compose up -d postgres redis

# Verify connectivity
psql -h localhost -p 5432 -U workflow_user -d workflow_engine -c "SELECT 1;"
redis-cli ping

# Expected: Both return success responses
```

#### 2.2 Apollo Federation Gateway
```bash
cd apollo-federation
npm install
npm run start:ui  # UI-enabled workflow gateway

# Verify: http://localhost:4000/graphql shows GraphQL playground
```

### Phase 3: Service Startup Sequence ⚡

#### 3.1 Foundation Services
```bash
# Authentication Service (Port 8001)
cd backend/services/auth-service
pip install -r requirements.txt
python run_service.py &

# Patient Service (Port 8003)
cd backend/services/patient-service
python run_service.py &

# Health Check
curl http://localhost:8001/health
curl http://localhost:8003/health
```

#### 3.2 Context & Data Services
```bash
# Context Gateway Go (Port 8016)
cd backend/services/context-gateway-go
go run . &

# Medication Service V2 (Port 8004)
cd backend/services/medication-service-v2
python run_service.py &

# Health Check
curl http://localhost:8016/health
curl http://localhost:8004/health
```

#### 3.3 Processing Engines (After Fixes)
```bash
# Flow2 Go Engine (Port 8080) - After compilation fix
cd backend/services/medication-service/flow2-go-engine
python3 run.py --dev &

# Flow2 Rust Engine (Port 8090)
cd backend/services/medication-service/flow2-rust-engine
cargo run &

# Safety Gateway Platform (Port 8018)
cd backend/services/safety-gateway-platform
go run . &

# Health Check
curl http://localhost:8080/health
curl http://localhost:8090/health
curl http://localhost:8018/health
```

#### 3.4 Target Workflow Engine (After Fixes)
```bash
cd backend/services/workflow-engine-go-service
go run cmd/server/main.go &

# Health Check
curl http://localhost:8020/health
```

### Phase 4: End-to-End Workflow Testing 🎯

#### 4.1 Health Check Cascade
```bash
#!/bin/bash
# health-check-all.sh

services=(
  "http://localhost:4000/graphql"     # Apollo Federation
  "http://localhost:8001/health"      # Auth Service
  "http://localhost:8003/health"      # Patient Service
  "http://localhost:8004/health"      # Medication Service V2
  "http://localhost:8016/health"      # Context Gateway
  "http://localhost:8018/health"      # Safety Gateway
  "http://localhost:8080/health"      # Flow2 Go Engine
  "http://localhost:8090/health"      # Flow2 Rust Engine
  "http://localhost:8020/health"      # Workflow Engine
)

echo "🔍 Testing all service health endpoints..."
for service in "${services[@]}"; do
  echo -n "Testing $service ... "
  if curl -f -s $service > /dev/null; then
    echo "✅ OK"
  else
    echo "❌ FAILED"
  fi
done
```

#### 4.2 Complete 3-Phase Workflow Test
```graphql
# Test via Apollo Federation GraphQL
# URL: http://localhost:4000/graphql

mutation ExecuteWorkflow($input: WorkflowExecutionInput!) {
  executeWorkflow(input: $input) {
    workflowId
    phases {
      phase
      status
      duration
      results {
        calculateResults {
          medicationIntelligence
          doseOptimization
          clinicalRecommendations
        }
        validateResults {
          safetyStatus
          safetyAlerts
          clinicalSignificance
        }
        commitResults {
          commitStatus
          auditTrail
          idempotencyToken
        }
      }
    }
    uiCoordination {
      realTimeUpdates
      notificationsSent
      sessionState
    }
  }
}
```

**Test Variables**:
```json
{
  "input": {
    "patientId": "patient-123",
    "medicationOrder": {
      "medicationId": "medication-456",
      "dosage": "10mg",
      "frequency": "twice daily",
      "route": "oral"
    },
    "clinicalContext": {
      "conditions": ["hypertension"],
      "allergies": [],
      "currentMedications": []
    }
  }
}
```

#### 4.3 Real-Time UI Coordination Test
```javascript
// Test WebSocket connections for real-time updates
const ws = new WebSocket('ws://localhost:4000/subscriptions');

// Subscribe to workflow updates
const subscription = `
  subscription WorkflowUpdates($workflowId: ID!) {
    workflowStatusUpdated(workflowId: $workflowId) {
      workflowId
      currentPhase
      status
      progress
      realTimeNotifications
    }
  }
`;

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'start',
    payload: {
      query: subscription,
      variables: { workflowId: 'workflow-123' }
    }
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Real-time workflow update:', data);
};
```

#### 4.4 Error Handling & Rollback Testing
```bash
# Test rollback mechanism (5-minute window)
curl -X POST http://localhost:8020/api/v1/orchestration/rollback \
  -H "Content-Type: application/json" \
  -d '{
    "workflowId": "workflow-123",
    "rollbackToken": "token-456",
    "reason": "Clinical override required"
  }'

# Expected: {"success": true, "rollbackCompleted": true}

# Test idempotency protection
curl -X POST http://localhost:8020/api/v1/orchestration/commit \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: duplicate-request-123" \
  -d '{"workflowId": "workflow-123"}'

# Expected: Same response for duplicate requests
```

## 📊 Performance Targets

Based on configuration in `internal/config/config.go`:

| Phase | Target Time | Measurement | Status |
|-------|-------------|-------------|---------|
| **Calculate** | 175ms | TBD | ⏳ |
| **Validate** | 100ms | TBD | ⏳ |
| **Commit** | 50ms | TBD | ⏳ |
| **Total** | **325ms** | **TBD** | **⏳** |

### Performance Testing Commands
```bash
# Load testing with ApacheBench
ab -n 100 -c 10 -H "Content-Type: application/json" \
   -p workflow-test-payload.json \
   http://localhost:8020/api/v1/orchestration/execute

# Monitor real-time performance
curl http://localhost:8020/metrics | grep workflow_execution_duration
```

## 🎯 Success Criteria Checklist

### ✅ Compilation Success
- [ ] Flow2 Go Engine builds without errors
- [ ] Workflow Engine Go Service builds without errors
- [ ] All external services start successfully

### ✅ Service Integration
- [ ] All health endpoints respond successfully
- [ ] Apollo Federation can communicate with all subgraphs
- [ ] Database connections and migrations complete
- [ ] Redis pub/sub messaging functional

### ✅ End-to-End Workflow
- [ ] Complete Calculate → Validate → Commit workflow execution
- [ ] Real-time UI coordination through Apollo Federation
- [ ] Clinical safety validation with actual safety gateway
- [ ] Rollback and idempotency mechanisms functional
- [ ] Performance targets met (≤325ms total execution)

### ✅ Production Readiness
- [ ] Comprehensive error handling and recovery
- [ ] Monitoring and observability integration
- [ ] Security validation (JWT, HTTPS, data protection)
- [ ] Load testing with realistic clinical scenarios

## 🛠️ Troubleshooting Guide

### Common Issues

#### "Connection refused" errors
```bash
# Check if service is running
netstat -tlnp | grep :8080

# Check service logs
docker-compose logs -f workflow-engine
```

#### GraphQL Federation errors
```bash
# Check subgraph registration
curl http://localhost:4000/.well-known/apollo/server-health

# Verify schema composition
npm run generate-supergraph:ui
```

#### Database connection issues
```bash
# Check PostgreSQL connectivity
docker-compose exec postgres psql -U workflow_user -d workflow_engine -c "\dt"

# Check migration status
go run cmd/migrate/main.go status
```

### Recovery Procedures

#### Service startup failure
1. Check all prerequisite services are running
2. Verify environment variables and configuration
3. Check port conflicts: `lsof -i :PORT_NUMBER`
4. Review service logs for specific error messages

#### Performance degradation
1. Check database connection pool utilization
2. Monitor Redis memory usage and eviction
3. Review external service response times
4. Check for resource contention (CPU, memory)

## 📝 Test Execution Log

### Execution Date: ___________
### Tester: ___________

#### Phase 1: Compilation Fixes
- [ ] Flow2 Go Engine compilation: ___________
- [ ] Workflow Engine compilation: ___________

#### Phase 2: Infrastructure
- [ ] PostgreSQL startup: ___________
- [ ] Redis startup: ___________
- [ ] Apollo Federation startup: ___________

#### Phase 3: Service Startup
- [ ] Auth Service: ___________
- [ ] Patient Service: ___________
- [ ] Context Gateway: ___________
- [ ] Medication Service: ___________
- [ ] Flow2 Go Engine: ___________
- [ ] Flow2 Rust Engine: ___________
- [ ] Safety Gateway: ___________
- [ ] Workflow Engine: ___________

#### Phase 4: End-to-End Testing
- [ ] Health check cascade: ___________
- [ ] 3-Phase workflow execution: ___________
- [ ] Real-time UI coordination: ___________
- [ ] Error handling & rollback: ___________
- [ ] Performance validation: ___________

**Overall Test Result**: ___________

**Notes**:
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________

---

## 📞 Support & Resources

- **Architecture Documentation**: `README.md`
- **API Documentation**: `docs/api.md`
- **Configuration Guide**: `internal/config/config.go`
- **Troubleshooting**: This document - Troubleshooting section
- **Performance Monitoring**: `http://localhost:9090/metrics` (Prometheus)

This comprehensive testing approach ensures the workflow engine service is validated against its complete production environment, exposing real integration issues and performance characteristics that mock-based testing would miss.