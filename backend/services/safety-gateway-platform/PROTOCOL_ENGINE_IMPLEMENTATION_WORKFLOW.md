# Protocol Engine Implementation Workflow
## Safety Gateway Platform Enhancement

### Executive Summary
This workflow provides a comprehensive implementation strategy for adding the Protocol Engine to the Safety Gateway Platform, based on the design documents:
- `04_9.1 Protocol Engine - Comprehensive Design & Implementation Guide.txt`
- `04_9.2 Protocol Engine - Enhanced Snapshot-Driven Architecture_.txt`

The Protocol Engine will enforce clinical protocols, care pathways, and institutional policies, complementing the existing Clinical Assertion Engine (CAE) with stateful protocol tracking and temporal constraints.

---

## Phase 1: Foundation & Core Engine (Weeks 1-2)

### 1.1 Project Structure & Dependencies
**Duration**: 2 days
**Dependencies**: None
**Deliverables**:
- [ ] Create Go module structure for Protocol Engine
- [ ] Set up dependency injection framework integration
- [ ] Configure database schema for protocols and states
- [ ] Implement basic configuration management

```go
// Target structure:
safety-gateway-platform/
├── internal/protocol/
│   ├── engine.go
│   ├── repository.go
│   ├── models/
│   └── config/
├── migrations/
├── protocols/
└── cmd/protocol-engine/
```

### 1.2 Core Protocol Engine
**Duration**: 4 days
**Dependencies**: 1.1
**Deliverables**:
- [ ] Implement `ProtocolEngine` main struct with snapshot integration
- [ ] Create `ProtocolEvaluationContext` with snapshot awareness
- [ ] Build basic protocol loading and caching mechanisms
- [ ] Implement hard/soft constraint evaluation logic

### 1.3 Snapshot Manager Integration
**Duration**: 3 days
**Dependencies**: 1.2
**Deliverables**:
- [ ] Integrate with existing `SnapshotManager` service
- [ ] Implement snapshot-aware protocol version resolution
- [ ] Add protocol versions to snapshot manifest schema
- [ ] Create snapshot validation for protocol compatibility

### 1.4 Basic GraphQL Schema
**Duration**: 1 day
**Dependencies**: 1.2
**Deliverables**:
- [ ] Design GraphQL schema for protocol evaluation results
- [ ] Implement basic GraphQL resolvers
- [ ] Configure Apollo Federation schema extension

**Phase 1 Success Criteria**:
- ✅ Basic protocol evaluation with snapshot integration
- ✅ Hard/soft constraint enforcement
- ✅ GraphQL API endpoints functional
- ✅ Unit test coverage >80%

---

## Phase 2: State Management & Temporal Engine (Weeks 3-4)

### 2.1 Protocol State Manager
**Duration**: 5 days
**Dependencies**: Phase 1
**Deliverables**:
- [ ] Implement `ProtocolStateManager` with snapshot continuity
- [ ] Create protocol state persistence layer (PostgreSQL/Redis)
- [ ] Build state transition validation logic
- [ ] Implement state snapshot lineage tracking

### 2.2 Temporal Constraint Engine
**Duration**: 4 days
**Dependencies**: 2.1
**Deliverables**:
- [ ] Implement `TemporalConstraintEngine` with snapshot time reference
- [ ] Create time-window validation for medication administration
- [ ] Build perioperative protocol temporal logic
- [ ] Implement snapshot age validation for temporal decisions

### 2.3 Stateful Protocol Implementation
**Duration**: 1 day
**Dependencies**: 2.1, 2.2
**Deliverables**:
- [ ] Implement sample stateful protocols (Sepsis Bundle, VTE Prophylaxis)
- [ ] Create protocol state machine definitions
- [ ] Build state transition event handling

**Phase 2 Success Criteria**:
- ✅ Stateful protocol tracking across snapshots
- ✅ Temporal constraint validation functional
- ✅ Protocol state persistence and recovery
- ✅ Integration tests passing

---

## Phase 3: CAE Integration & Event Publishing (Weeks 5-6)

### 3.1 CAE Coordination
**Duration**: 4 days
**Dependencies**: Phase 2
**Deliverables**:
- [ ] Implement `CoordinatedSafetyEvaluation` with shared snapshots
- [ ] Create conflict resolution logic between CAE and Protocol Engine
- [ ] Build precedence rules for safety vs protocol decisions
- [ ] Implement bidirectional communication patterns

### 3.2 Event Publishing Infrastructure
**Duration**: 3 days
**Dependencies**: 3.1
**Deliverables**:
- [ ] Implement outbox pattern for reliable event publishing
- [ ] Create Kafka event schemas for protocol evaluations
- [ ] Build event publishing with transactional guarantees
- [ ] Implement event replay and failure recovery

### 3.3 Approval Workflow Foundation
**Duration**: 3 days
**Dependencies**: 3.1
**Deliverables**:
- [ ] Implement `PolicyApprovalEngine` basic structure
- [ ] Create approval request management
- [ ] Build role-based approval routing
- [ ] Implement basic override patterns

**Phase 3 Success Criteria**:
- ✅ Protocol Engine + CAE coordinated evaluation
- ✅ Reliable event publishing to workflow platform
- ✅ Basic approval workflows functional
- ✅ End-to-end integration tests passing

---

## Phase 4: Production Readiness (Weeks 7-8)

### 4.1 Comprehensive Testing Framework
**Duration**: 4 days
**Dependencies**: Phase 3
**Deliverables**:
- [ ] Implement protocol test harness with YAML definitions
- [ ] Create integration test suite for full workflows
- [ ] Build performance test scenarios
- [ ] Implement compliance and audit testing

### 4.2 Performance Optimization
**Duration**: 2 days
**Dependencies**: 4.1
**Deliverables**:
- [ ] Implement protocol precompilation and caching
- [ ] Optimize database queries and connection pooling
- [ ] Add performance monitoring and metrics
- [ ] Implement circuit breakers and rate limiting

### 4.3 Production Deployment
**Duration**: 4 days
**Dependencies**: 4.2
**Deliverables**:
- [ ] Create Kubernetes deployment manifests
- [ ] Implement health checks and readiness probes
- [ ] Configure monitoring, logging, and alerting
- [ ] Execute shadow mode deployment strategy

**Phase 4 Success Criteria**:
- ✅ Production-ready deployment with monitoring
- ✅ Performance benchmarks met
- ✅ Shadow mode validation successful
- ✅ Full test coverage and documentation complete

---

## Technical Architecture

### Core Components
```go
type ProtocolEngine struct {
    protocolRepository    *ProtocolRepository
    stateManager         *ProtocolStateManager
    ruleEngine          *ClinicalRuleEngine
    temporalEngine      *TemporalConstraintEngine
    approvalEngine      *PolicyApprovalEngine
    snapshotManager     *SnapshotManager
    contextGateway      *ContextGatewayClient
    auditService        *AuditService
}

type ProtocolEvaluationContext struct {
    snapshot           *Snapshot
    kbVersions        map[string]string
    protocolVersions  map[string]string
    proposedAction    *ClinicalAction
    patientContext    *PatientContext
}
```

### Database Schema Extensions
```sql
-- Protocol definitions
CREATE TABLE protocols (
    id VARCHAR(255) PRIMARY KEY,
    version VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    category protocol_category NOT NULL,
    definition JSONB NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Protocol states
CREATE TABLE protocol_states (
    id UUID PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    protocol_id VARCHAR(255) NOT NULL,
    current_state VARCHAR(100) NOT NULL,
    snapshot_id VARCHAR(255) NOT NULL,
    state_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Approval requests
CREATE TABLE approval_requests (
    id UUID PRIMARY KEY,
    protocol_result_id UUID NOT NULL,
    status approval_status NOT NULL DEFAULT 'PENDING',
    approvers TEXT[] NOT NULL,
    decisions JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Integration Points

#### 1. Snapshot Manager Integration
```go
func (pe *ProtocolEngine) evaluate(
    proposedAction *ClinicalAction,
    patientContext *PatientContext,
    snapshotId string,
) (*ProtocolResult, error) {
    snapshot, err := pe.snapshotManager.GetSnapshot(snapshotId)
    if err != nil {
        return nil, fmt.Errorf("invalid snapshot: %w", err)
    }
    
    versions, err := pe.contextGateway.ResolveVersions(snapshotId)
    if err != nil {
        return nil, fmt.Errorf("version resolution failed: %w", err)
    }
    
    return pe.evaluateWithContext(proposedAction, patientContext, snapshot, versions)
}
```

#### 2. CAE Coordination
```go
type CombinedSafetyResult struct {
    CAEResult      *cae.EvaluationResult    `json:"cae_result"`
    ProtocolResult *protocol.ProtocolResult `json:"protocol_result"`
    Decision       SafetyDecision           `json:"decision"`
    Conflicts      []DecisionConflict       `json:"conflicts,omitempty"`
}

func (cs *CoordinatedSafetyEvaluation) Evaluate(
    proposal *MedicationProposal,
    snapshotId string,
) (*CombinedSafetyResult, error) {
    caeResult, protocolResult := await Promise.All([
        cs.cae.Evaluate(proposal.Action, proposal.PatientContext, snapshotId),
        cs.protocolEngine.Evaluate(proposal.Action, proposal.PatientContext, snapshotId),
    ])
    
    return cs.mergeResults(caeResult, protocolResult, snapshotId)
}
```

#### 3. Event Publishing
```go
type ProtocolEvaluatedEvent struct {
    EventType string `json:"event_type"`
    EventID   string `json:"event_id"`
    Payload   struct {
        ProposalID          string   `json:"proposal_id"`
        SnapshotID          string   `json:"snapshot_id"`
        Decision            string   `json:"decision"`
        TriggeredProtocols  []string `json:"triggered_protocols"`
        RequiresApproval    bool     `json:"requires_approval"`
        ApprovalRequestID   string   `json:"approval_request_id,omitempty"`
    } `json:"payload"`
    Metadata EventMetadata `json:"metadata"`
}
```

---

## Testing Strategy

### Unit Testing
```go
func TestProtocolEngine_EvaluateWithSnapshot(t *testing.T) {
    engine := setupTestEngine(t)
    snapshot := createTestSnapshot(t, "test-snapshot-id")
    
    result, err := engine.Evaluate(testAction, testContext, snapshot.ID)
    
    assert.NoError(t, err)
    assert.Equal(t, protocol.ACCEPT, result.Decision)
    assert.Equal(t, snapshot.ID, result.SnapshotID)
}
```

### Integration Testing
```yaml
# test_cases.yaml
test_suite: "protocol-engine-integration"
snapshot_id: "integration-test-2025-09-09"

test_cases:
  - id: "sepsis-bundle-recognition"
    description: "Validate sepsis protocol triggers correctly"
    given:
      patient:
        temperature: 38.5
        lactate: 2.5
      proposed_action:
        type: "order_labs"
    then:
      decision: "ACCEPT"
      triggered_protocols: ["sepsis-bundle"]
      state_transition: "RECOGNITION -> INITIAL_RESUSCITATION"
```

### Performance Testing
- Target: <100ms protocol evaluation latency
- Load: 1000 concurrent evaluations
- Memory: <512MB heap usage
- Database: <50ms query response time

---

## Deployment Strategy

### Shadow Mode (Week 8)
1. Deploy Protocol Engine alongside existing services
2. Run evaluations in parallel with current system
3. Log decisions but don't enforce constraints
4. Compare results and validate accuracy

### Soft Enforcement (Week 9-10)
1. Enable soft constraints only (warnings, not blocks)
2. Allow universal overrides for all decisions
3. Monitor adoption and feedback
4. Fine-tune protocol definitions

### Full Enforcement (Week 11-12)
1. Enable hard constraints for critical protocols
2. Restrict overrides to authorized roles
3. Full audit trail and compliance reporting
4. Monitor system stability and performance

---

## Risk Mitigation

### Technical Risks
- **Database Performance**: Implement connection pooling, query optimization
- **Snapshot Compatibility**: Version validation, backward compatibility
- **State Consistency**: Transactional updates, conflict resolution
- **Memory Usage**: Protocol caching limits, garbage collection tuning

### Operational Risks
- **Deployment Issues**: Blue-green deployment, automated rollback
- **Data Migration**: Incremental migration, data validation
- **Performance Impact**: Load testing, circuit breakers
- **Clinical Impact**: Shadow mode validation, gradual rollout

### Compliance Risks
- **Audit Requirements**: Comprehensive logging, digital signatures
- **Data Privacy**: HIPAA compliance, data retention policies
- **Regulatory Changes**: Version control, approval workflows

---

## Success Metrics

### Technical Metrics
- **Latency**: <100ms 95th percentile evaluation time
- **Throughput**: >1000 evaluations/second
- **Availability**: 99.9% uptime SLA
- **Error Rate**: <0.1% evaluation failures

### Business Metrics
- **Protocol Compliance**: >95% adherence to clinical pathways
- **Override Rate**: <5% of decisions overridden
- **Time to Decision**: <30 seconds for approval workflows
- **Audit Completeness**: 100% decision traceability

### Quality Metrics
- **Test Coverage**: >90% code coverage
- **Documentation**: 100% API documentation
- **Security**: Zero critical vulnerabilities
- **Performance**: All benchmarks met

This comprehensive workflow ensures the Protocol Engine integrates seamlessly with the existing Safety Gateway Platform while maintaining the highest standards of clinical safety, system reliability, and regulatory compliance.