# Vaidshala Architecture V3 Gap Analysis

## Document Purpose
This document analyzes the **current implementation** against the **proposed 7-Tier Architecture** from `vaidshala-architecture-v3.docx` to identify gaps before refactoring.

---

## 1. Executive Summary

| Metric | Status |
|--------|--------|
| **Overall Compliance** | 90% Correct |
| **Critical Gap** | Medication Advisor (8095) acting as Judge + Jury + Clerk |
| **Required Action** | Move enforcement/commit to KB-19 Protocol Orchestrator |
| **Risk Level** | HIGH (not court-defensible in current state) |

---

## 2. Current Implementation Inventory

### 2.1 Medication Advisor Engine (Port 8095)

**File:** `vaidshala/clinical-runtime-platform/services/medication-advisor-engine/advisor/engine.go`

#### Current Responsibilities (Lines 382-506)

| Responsibility | Method | Lines | Status per V3 |
|---------------|--------|-------|---------------|
| Build clinical snapshot | `buildClinicalData()` | 387 | ✅ CORRECT |
| Create evidence envelope | `evidenceManager.CreateEnvelope()` | 412-422 | ⚠️ SHOULD BE KB-19 |
| Execute workflow | `workflowEngine.Execute()` | 425-433 | ✅ CORRECT |
| Score/rank proposals | `scoringEngine.RankProposals()` | 436 | ✅ CORRECT |
| **Process excluded drugs** | `processExcludedDrugs()` | 439 | ⚠️ Returns HardBlocks - WRONG |
| **Process DDI hard blocks** | `processDDIHardBlocks()` | 444-445 | ❌ WRONG - Should return risk scores only |
| **Process Lab hard blocks** | `processLabHardBlocks()` | 451-453 | ❌ WRONG - Should return risk scores only |
| **Generate governance events** | `generateGovernanceEvents()` | 466 | ❌ WRONG - Should be KB-19 |
| **Generate lab safety tasks** | `generateLabSafetyTasks()` | 470-471 | ❌ WRONG - Should be KB-19/KB-14 |
| **Determine disposition** | `determineDisposition()` | 474 | ❌ WRONG - Should be KB-19 |
| **Build audit trail** | `buildAuditTrail()` | 482-491 | ❌ WRONG - Should be KB-19/KB-18 |

#### Hard Block Generation (Lines 893-1574) - SHOULD NOT EXIST HERE

```go
// These functions should return RISK SCORES, not HARD BLOCKS:

func processExcludedDrugs()     // Line 893  - Generates HardBlock structs ❌
func processDDIHardBlocks()     // Line 1195 - Generates HardBlock structs ❌
func processLabHardBlocks()     // Line 1442 - Generates HardBlock structs ❌
func generateGovernanceEvents() // Line 566  - Creates governance audit ❌
func generateLabSafetyTasks()   // Line 672  - Creates KB-14 tasks ❌
```

#### DDI Hard Stop Rules (Lines 1087-1178)

Currently embedded in Medication Advisor:
- MAOI + SSRI → Serotonin Syndrome
- Warfarin + NSAIDs → Major bleeding
- Methotrexate + Trimethoprim → Bone marrow suppression
- Potassium-sparing + ACE inhibitors → Hyperkalemia
- Cisapride + QT prolonging → Torsades de Pointes
- Linezolid + SSRIs → Serotonin Syndrome
- Ergotamine + CYP3A4 inhibitors → Ergotism
- Clozapine + Carbamazepine → Agranulocytosis
- Simvastatin + Gemfibrozil → Rhabdomyolysis
- Pimozide + Macrolides → QT prolongation

**V3 Requirement:** These should return severity scores to KB-19, not HardBlock decisions.

#### Lab-Drug Hard Stop Rules (Lines 1314-1432)

Currently embedded in Medication Advisor:
- K+ >5.5 + ACE/ARB → Hyperkalemia
- eGFR <30 + Metformin → Lactic acidosis
- INR >4.0 + Warfarin → Bleeding
- Creatinine >2.0 + Aminoglycosides → Nephrotoxicity
- Na+ <135 + Lithium → Lithium toxicity
- K+ <3.5 + Digoxin → Digoxin toxicity
- Platelets <50K + Anticoagulants → Bleeding
- ALT >120 + Statins → Hepatotoxicity
- Hemoglobin <7.0 + Anticoagulants → Active bleeding

**V3 Requirement:** These should return contraindication flags with severity, not HardBlock decisions.

---

### 2.2 KB-19 Protocol Orchestrator (Port 8119)

**File:** `backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/`

#### Current API Endpoints

| Endpoint | Method | Purpose | V3 Status |
|----------|--------|---------|-----------|
| `/health` | GET | Health check | ✅ EXISTS |
| `/ready` | GET | Readiness check | ✅ EXISTS |
| `/api/v1/execute` | POST | Protocol arbitration | ✅ EXISTS |
| `/api/v1/evaluate` | POST | Single protocol evaluation | ✅ EXISTS |
| `/api/v1/protocols` | GET | List protocols | ✅ EXISTS |
| `/api/v1/protocols/:id` | GET | Get protocol details | ✅ EXISTS |
| `/api/v1/decisions/:patientId` | GET | Get patient decisions | ✅ EXISTS |
| `/api/v1/bundle/:id` | GET | Get decision bundle | ✅ EXISTS |
| `/api/v1/conflicts` | GET | List conflicts | ✅ EXISTS |
| **`/api/v1/transaction/validate`** | POST | Validate medication | ❌ MISSING |
| **`/api/v1/transaction/commit`** | POST | Commit with audit | ❌ MISSING |
| **`/api/v1/transaction/override`** | POST | Override with identity | ❌ MISSING |
| **`/api/v1/transaction/{id}/audit`** | GET | Get audit trail | ❌ MISSING |

#### Current KB-19 Components

| Component | File | Status |
|-----------|------|--------|
| ArbitrationEngine | `internal/arbitration/engine.go` | ✅ EXISTS |
| SafetyGatekeeper | `internal/arbitration/safety_gatekeeper.go` | ✅ EXISTS |
| ConflictDetector | `internal/arbitration/conflict_detector.go` | ✅ EXISTS |
| PriorityResolver | `internal/arbitration/priority_resolver.go` | ✅ EXISTS |
| NarrativeGenerator | `internal/arbitration/narrative_generator.go` | ✅ EXISTS |
| VaidshalaCLient | `internal/clients/vaidshala_client.go` | ✅ EXISTS |
| KB14GovernanceClient | `internal/clients/kb14_governance_client.go` | ✅ EXISTS |
| **MedicationAdvisorClient** | N/A | ❌ MISSING |
| **TransactionManager** | N/A | ❌ MISSING |
| **OverrideWorkflow** | N/A | ❌ MISSING |
| **AuditFinalizer** | N/A | ❌ MISSING |

---

### 2.3 Vaidshala Service (Port 8090)

**File:** `vaidshala/clinical-runtime-platform/cmd/clinical-runtime/main.go`

#### Current Status: ✅ CORRECT PER V3

| Responsibility | Status | Notes |
|---------------|--------|-------|
| CQL Engine orchestration | ✅ CORRECT | Runs CQL → Measure → Medication |
| Session management | ✅ CORRECT | 3-phase workflow |
| KnowledgeSnapshot building | ✅ CORRECT | Uses factory wiring |
| KB service calls | ✅ CORRECT | Via snapshotBuilder |

**V3 Statement:** "Vaidshala (8090) stays exactly where it is - it is NOT a transaction engine"

---

## 3. Gap Analysis Matrix

### 3.1 Responsibilities Mapping

| Responsibility | Current Location | V3 Required Location | Gap Status |
|---------------|------------------|---------------------|------------|
| Risk calculation | Med-Advisor (8095) | Med-Advisor (8095) | ✅ NO GAP |
| DDI severity scoring | Med-Advisor (8095) | Med-Advisor (8095) | ✅ NO GAP |
| Lab contraindication flags | Med-Advisor (8095) | Med-Advisor (8095) | ✅ NO GAP |
| Dosing recommendations | Med-Advisor (8095) | Med-Advisor (8095) | ✅ NO GAP |
| Pregnancy/lactation risk | Med-Advisor (8095) | Med-Advisor (8095) | ✅ NO GAP |
| **Hard block enforcement** | Med-Advisor (8095) | KB-19 (8119) | ❌ GAP |
| **Override workflow** | Med-Advisor (8095) | KB-19 (8119) | ❌ GAP |
| **Audit finalization** | Med-Advisor (8095) | KB-19 (8119) | ❌ GAP |
| **Jurisdiction rules** | Med-Advisor (8095) | KB-19 (8119) | ❌ GAP |
| **Commit to EHR** | Med-Advisor (8095) | KB-19 (8119) | ❌ GAP |
| **Governance events** | Med-Advisor (8095) | KB-19 (8119) | ❌ GAP |
| **Identity binding** | Missing | KB-19 + KB-18 | ❌ GAP |

### 3.2 Data Flow Gap

#### Current Flow (WRONG):
```
EHR/CPOE → Med-Advisor (8095) → [BLOCKS HERE] → Response
                    ↓
           Creates HardBlocks
           Creates GovernanceEvents
           Creates Tasks
           Determines Disposition
           Finalizes Audit
```

#### V3 Required Flow (CORRECT):
```
EHR/CPOE → KB-19 (8119) → Med-Advisor (8095) → Risk Graph → KB-19 (8119)
                                                                ↓
                                                    [VALIDATES HERE]
                                                    [COMMITS HERE]
                                                    [AUDIT HERE]
```

---

## 4. Code-Level Gap Analysis

### 4.1 Medication Advisor Engine - Functions to Refactor

| Function | Current Return | V3 Required Return | Action |
|----------|---------------|-------------------|--------|
| `processExcludedDrugs()` | `[]HardBlock, []ExcludedDrugInfo` | `[]RiskScore, []ExcludedDrugInfo` | REFACTOR |
| `processDDIHardBlocks()` | `[]HardBlock` | `[]DDIRiskScore` | REFACTOR |
| `processLabHardBlocks()` | `[]HardBlock` | `[]LabContraindicationFlag` | REFACTOR |
| `generateGovernanceEvents()` | `[]GovernanceEvent` | REMOVE | DELETE |
| `generateLabSafetyTasks()` | `[]GeneratedTask` | REMOVE | DELETE |
| `determineDisposition()` | `DispositionCode` | REMOVE | DELETE |
| `buildAuditTrail()` | `AuditTrailSummary` | REMOVE | DELETE |

### 4.2 Medication Advisor Engine - New Response Structure

**Current CalculateResponse (WRONG):**
```go
type CalculateResponse struct {
    SnapshotID       uuid.UUID
    EnvelopeID       uuid.UUID
    Proposals        []MedicationProposal
    HardBlocks       []HardBlock          // ❌ WRONG - enforcement
    ExcludedDrugs    []ExcludedDrugInfo
    GeneratedTasks   []GeneratedTask      // ❌ WRONG - should be KB-14
    GovernanceEvents []GovernanceEvent    // ❌ WRONG - should be KB-19
    Disposition      DispositionCode      // ❌ WRONG - should be KB-19
    AuditTrail       AuditTrailSummary    // ❌ WRONG - should be KB-18
    ExecutionTimeMs  int64
    KBVersions       map[string]string
}
```

**V3 Required Response (CORRECT):**
```go
type CalculateResponse struct {
    SnapshotID       uuid.UUID
    Proposals        []MedicationProposal
    RiskProfile      RiskProfile           // NEW - comprehensive risk
    DDIRisks         []DDIRiskScore        // NEW - severity scores only
    LabRisks         []LabContraindicationFlag // NEW - flags only
    PregnancyRisks   []PregnancySafetyFlag // NEW - flags only
    ExcludedDrugs    []ExcludedDrugInfo    // Keep but no HardBlock
    QualityScores    map[string]float64    // NEW - scoring breakdown
    ExecutionTimeMs  int64
    KBVersions       map[string]string
    // NO HardBlocks
    // NO Disposition
    // NO GovernanceEvents
    // NO AuditTrail
    // NO GeneratedTasks
}
```

### 4.3 KB-19 Protocol Orchestrator - Functions to Add

| Function | Purpose | Priority |
|----------|---------|----------|
| `TransactionValidate()` | Validate medication against rules | P0 |
| `TransactionCommit()` | Commit with full audit trail | P0 |
| `ProcessOverride()` | Handle override with identity binding | P0 |
| `EnforceHardBlocks()` | Take risk scores → make block decisions | P0 |
| `GenerateGovernanceEvents()` | Create Tier-7 audit events | P0 |
| `FinalizeAudit()` | Complete audit with KB-18 | P0 |
| `CallMedicationAdvisor()` | Get risk profile from 8095 | P0 |
| `ApplyJurisdictionRules()` | Hospital/region policy | P1 |
| `CreateEHRCommit()` | FHIR MedicationRequest | P1 |

### 4.4 KB-19 - New Transaction API

```go
// POST /api/v1/transaction/validate
type TransactionValidateRequest struct {
    PatientID        uuid.UUID              `json:"patient_id"`
    EncounterID      uuid.UUID              `json:"encounter_id"`
    ProposedMedication ProposedMedication   `json:"proposed_medication"`
    ProviderID       string                 `json:"provider_id"`
    RequestedBy      string                 `json:"requested_by"`
}

type TransactionValidateResponse struct {
    TransactionID    uuid.UUID              `json:"transaction_id"`
    Allowed          bool                   `json:"allowed"`
    HardBlocks       []HardBlock            `json:"hard_blocks,omitempty"`
    Warnings         []Warning              `json:"warnings,omitempty"`
    RiskProfile      RiskProfile            `json:"risk_profile"`
    RequiresOverride bool                   `json:"requires_override"`
    OverrideOptions  []OverrideOption       `json:"override_options,omitempty"`
}

// POST /api/v1/transaction/commit
type TransactionCommitRequest struct {
    TransactionID    uuid.UUID              `json:"transaction_id"`
    ProviderID       string                 `json:"provider_id"`
    Acknowledged     bool                   `json:"acknowledged"`
    Overrides        []OverrideDecision     `json:"overrides,omitempty"`
}

type TransactionCommitResponse struct {
    CommitID         uuid.UUID              `json:"commit_id"`
    Status           string                 `json:"status"`
    FHIRResourceID   string                 `json:"fhir_resource_id"`
    AuditTrailID     uuid.UUID              `json:"audit_trail_id"`
    GovernanceEvents []GovernanceEvent      `json:"governance_events"`
}

// POST /api/v1/transaction/override
type TransactionOverrideRequest struct {
    TransactionID    uuid.UUID              `json:"transaction_id"`
    ProviderID       string                 `json:"provider_id"`
    OverrideType     string                 `json:"override_type"`
    Reason           string                 `json:"reason"`
    AcknowledgmentText string              `json:"acknowledgment_text"`
}

type TransactionOverrideResponse struct {
    OverrideID       uuid.UUID              `json:"override_id"`
    Status           string                 `json:"status"`
    IdentityBound    bool                   `json:"identity_bound"`
    AuditRecorded    bool                   `json:"audit_recorded"`
}
```

---

## 5. Verification Checklist

### 5.1 Pre-Refactor Verification

- [ ] **KB-19 is running:** `curl http://localhost:8119/health`
- [ ] **Med-Advisor is running:** `curl http://localhost:8095/health`
- [ ] **Vaidshala is running:** `curl http://localhost:8090/health`
- [ ] **KB-18 Governance exists:** Check for KB-18 service
- [ ] **All KB services healthy:** KB-1, KB-4, KB-5, KB-7, KB-8

### 5.2 Current Behavior Tests

```bash
# Test 1: Med-Advisor currently returns HardBlocks (SHOULD FAIL after refactor)
curl -X POST http://localhost:8095/api/v1/advisor/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "patient_context": {
      "age": 65,
      "medications": [{"code": "6809", "display": "Metformin"}],
      "lab_results": [{"code": "33914-3", "value": 25, "unit": "mL/min/1.73m2"}]
    }
  }' | jq '.hard_blocks'
# Current: Returns hard_blocks array
# After V3: Should NOT have hard_blocks field

# Test 2: KB-19 transaction API (SHOULD WORK after refactor)
curl -X POST http://localhost:8119/api/v1/transaction/validate \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "proposed_medication": {"rxnorm": "6809", "name": "Metformin"}
  }'
# Current: 404 Not Found
# After V3: Returns TransactionValidateResponse
```

---

## 6. Implementation Plan

### Phase 1: Add KB-19 Transaction API (Week 1)

| Task | Priority | Est. Hours |
|------|----------|------------|
| Create `TransactionValidateRequest/Response` contracts | P0 | 2h |
| Implement `handleTransactionValidate()` handler | P0 | 4h |
| Create `MedicationAdvisorClient` to call 8095 | P0 | 3h |
| Implement `TransactionCommit()` with KB-18 integration | P0 | 4h |
| Implement `TransactionOverride()` with identity binding | P0 | 4h |
| Add transaction audit trail endpoints | P0 | 3h |
| **Total Phase 1** | | **20h** |

### Phase 2: Refactor Medication Advisor (Week 2)

| Task | Priority | Est. Hours |
|------|----------|------------|
| Create new `RiskProfile` response structure | P0 | 2h |
| Refactor `processDDIHardBlocks()` → `calculateDDIRisks()` | P0 | 3h |
| Refactor `processLabHardBlocks()` → `calculateLabRisks()` | P0 | 3h |
| Remove `generateGovernanceEvents()` | P0 | 1h |
| Remove `generateLabSafetyTasks()` | P0 | 1h |
| Remove `determineDisposition()` | P0 | 1h |
| Remove `buildAuditTrail()` | P0 | 1h |
| Update `CalculateResponse` structure | P0 | 2h |
| Update all tests | P0 | 4h |
| **Total Phase 2** | | **18h** |

### Phase 3: Integration Testing (Week 3)

| Task | Priority | Est. Hours |
|------|----------|------------|
| End-to-end flow test: CPOE → KB-19 → Med-Advisor → KB-19 | P0 | 4h |
| Override workflow test with identity binding | P0 | 3h |
| Audit trail verification | P0 | 2h |
| Court-defensibility documentation | P1 | 3h |
| Performance testing | P1 | 2h |
| **Total Phase 3** | | **14h** |

---

## 7. Success Criteria

### 7.1 Technical Success

- [ ] KB-19 `/api/v1/transaction/validate` returns block decisions
- [ ] KB-19 `/api/v1/transaction/commit` creates FHIR resources
- [ ] KB-19 `/api/v1/transaction/override` binds identity
- [ ] Med-Advisor returns ONLY risk scores (no HardBlocks)
- [ ] All governance events created by KB-19
- [ ] All audit trails finalized by KB-18

### 7.2 Regulatory Success

- [ ] Override decision traceable to specific provider
- [ ] Block decision traceable to KB-19 (not Med-Advisor)
- [ ] Audit trail shows: "KB-19 blocked, Dr. X overrode, KB-18 recorded"
- [ ] Court-defensible: Single authority (KB-19) for commit decisions

### 7.3 Architecture Compliance

- [ ] Med-Advisor = Risk Computer only
- [ ] KB-19 = Transaction Authority
- [ ] KB-18 = Governance & Identity
- [ ] Vaidshala (8090) = Unchanged
- [ ] ICU Intelligence (8120) = Unchanged

---

## 8. Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing integrations | HIGH | Feature flag for gradual rollout |
| KB-19 becomes single point of failure | MEDIUM | Circuit breaker, fallback mode |
| Performance degradation (extra hop) | LOW | Parallel calls, caching |
| Override abuse | HIGH | KB-18 identity binding, audit alerts |

---

## 9. Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Architect | | | |
| CTO | | | |
| CMO | | | |
| Security | | | |

---

## Appendix A: File References

### Medication Advisor Engine
- Main: `vaidshala/clinical-runtime-platform/services/medication-advisor-engine/advisor/engine.go`
- Server: `vaidshala/clinical-runtime-platform/services/medication-advisor-engine/cmd/server/main.go`
- Workflow: `vaidshala/clinical-runtime-platform/services/medication-advisor-engine/advisor/workflow.go`
- Scoring: `vaidshala/clinical-runtime-platform/services/medication-advisor-engine/advisor/scoring.go`

### KB-19 Protocol Orchestrator
- Handlers: `backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/internal/api/handlers.go`
- Contracts: `backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/pkg/contracts/api_contracts.go`
- Engine: `backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/internal/arbitration/engine.go`
- Safety: `backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/internal/arbitration/safety_gatekeeper.go`

### Vaidshala Service
- Main: `vaidshala/clinical-runtime-platform/cmd/clinical-runtime/main.go`

### Architecture Document
- Source: `backend/vaidshala-architecture-v3.docx`
