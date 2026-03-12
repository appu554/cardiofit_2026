# Vaidshala V3 Implementation Plan

## Document Control

| Field | Value |
|-------|-------|
| **Version** | 2.0 |
| **Date** | January 15, 2026 |
| **Status** | READY FOR IMPLEMENTATION |
| **Based On** | vaidshala-architecture-v3.docx |
| **Approach** | **MOVE (not CREATE)** - Migrate existing code from Med-Advisor to KB-19 |
| **Total Effort** | ~40 hours (5 dev days) |
| **Timeline** | 2 weeks |

---

## 0. Migration Map (MOVE, Not CREATE)

### Critical Understanding
**We are NOT writing new code from scratch.** We are **MOVING** existing, working code from Medication Advisor (8095) to KB-19 (8119).

### Code to MOVE: Medication Advisor → KB-19

| Function/Type | Source File | Source Lines | Destination | Notes |
|--------------|-------------|--------------|-------------|-------|
| **HardBlock struct** | `engine.go` | 318-330 | `transaction/types.go` | Direct copy |
| **GovernanceEvent struct** | `engine.go` | 291-310 | `transaction/types.go` | Direct copy |
| **GovernanceEventType** | `engine.go` | 273-288 | `transaction/types.go` | Direct copy |
| **DispositionCode** | `engine.go` | 236-268 | `transaction/types.go` | Direct copy |
| **AuditTrailSummary** | `engine.go` | 195-221 | `transaction/types.go` | Direct copy |
| **processExcludedDrugs()** | `engine.go` | 893-950 | `transaction/validator.go` | Rename to `evaluateHardBlocks()` |
| **processDDIHardBlocks()** | `engine.go` | 1195-1230 | `transaction/validator.go` | Rename to `evaluateDDIBlocks()` |
| **processLabHardBlocks()** | `engine.go` | 1442-1492 | `transaction/validator.go` | Rename to `evaluateLabBlocks()` |
| **generateGovernanceEvents()** | `engine.go` | 566-606 | `transaction/committer.go` | Direct copy |
| **generateLabSafetyTasks()** | `engine.go` | 672-744 | `transaction/committer.go` | Direct copy |
| **determineDisposition()** | `engine.go` | 747-780 | `transaction/validator.go` | Direct copy |
| **buildAuditTrail()** | `engine.go` | 783-852 | `transaction/committer.go` | Direct copy |
| **computeEventHash()** | `engine.go` | 631-668 | `transaction/committer.go` | Direct copy |
| **DDIHardStopPair struct** | `engine.go` | 1072-1083 | `transaction/rules/ddi_rules.go` | Direct copy |
| **knownDDIHardStops** | `engine.go` | 1087-1177 | `transaction/rules/ddi_rules.go` | Direct copy |
| **drugClassMemberships** | `engine.go` | 1181-1192 | `transaction/rules/ddi_rules.go` | Direct copy |
| **LabDrugHardStop struct** | `engine.go` | 1315-1328 | `transaction/rules/lab_rules.go` | Direct copy |
| **knownLabDrugHardStops** | `engine.go` | 1332-1432 | `transaction/rules/lab_rules.go` | Direct copy |
| **labDrugClassMemberships** | `engine.go` | 1435-1439 | `transaction/rules/lab_rules.go` | Direct copy |
| **ConflictDetector** | `conflicts.go` | entire file | `transaction/conflict_detector.go` | Direct copy |
| **Validate()** | `engine.go` | 1599-1653 | `transaction/validator.go` | Refactor to use transaction |
| **Commit()** | `engine.go` | 1686-1734 | `transaction/committer.go` | Refactor to use transaction |

### Code to KEEP in Medication Advisor (Risk Computer)

| Function/Type | File | Lines | Why Keep |
|--------------|------|-------|----------|
| **WorkflowOrchestrator** | `workflow.go` | entire file | Core risk calculation logic |
| **KB client calls (Phase 1-4)** | `workflow.go` | 184-502 | Drug selection, safety checks, dosing, scoring |
| **MedicationCandidate** | `workflow.go` | 53-59 | Risk output type |
| **ExcludedDrug** | `workflow.go` | 62-68 | Risk output type |
| **QualityFactors** | `engine.go` | 366-373 | Risk scores |
| **ProposalScoringEngine** | `scoring.go` | entire file | Risk ranking |
| **SnapshotManager** | `snapshot/manager.go` | entire file | Clinical context snapshots |
| **EvidenceEnvelopeManager** | `evidence/envelope.go` | entire file | Evidence chain building |

### Code to DELETE from Medication Advisor (After Move)

| Function | File | Lines | Reason |
|----------|------|-------|--------|
| `processExcludedDrugs()` | `engine.go` | 893-950 | Moved to KB-19 |
| `processDDIHardBlocks()` | `engine.go` | 1195-1230 | Moved to KB-19 |
| `processLabHardBlocks()` | `engine.go` | 1442-1492 | Moved to KB-19 |
| `generateGovernanceEvents()` | `engine.go` | 566-606 | Moved to KB-19 |
| `generateLabSafetyTasks()` | `engine.go` | 672-744 | Moved to KB-19 |
| `determineDisposition()` | `engine.go` | 747-780 | Moved to KB-19 |
| `buildAuditTrail()` | `engine.go` | 783-852 | Moved to KB-19 |
| `Validate()` | `engine.go` | 1599-1653 | Moved to KB-19 |
| `Commit()` | `engine.go` | 1686-1734 | Moved to KB-19 |
| `ConflictDetector` | `conflicts.go` | entire file | Moved to KB-19 |
| All DDI/Lab rule definitions | `engine.go` | 1072-1439 | Moved to KB-19 |

---

## 1. Implementation Overview

### 1.1 The Core Problem

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CURRENT STATE (WRONG)                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Medication Advisor Engine (8095) is acting as:                            │
│                                                                             │
│   ┌─────────┐   ┌─────────┐   ┌─────────┐                                  │
│   │  JUDGE  │ + │  JURY   │ + │  CLERK  │  = NOT COURT-DEFENSIBLE          │
│   │(decides)│   │(blocks) │   │(audits) │                                  │
│   └─────────┘   └─────────┘   └─────────┘                                  │
│                                                                             │
│   Question: "If a blocked medication is overridden, who allowed it?"        │
│   Answer: "Port 8095? CQL? UI?" ← THIS IS THE PROBLEM                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         TARGET STATE (V3 CORRECT)                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌──────────────────┐        ┌──────────────────┐                         │
│   │ Med-Advisor 8095 │        │    KB-19 8119    │                         │
│   │                  │        │                  │                         │
│   │  RISK COMPUTER   │   →    │ TRANSACTION      │                         │
│   │  (calculates)    │        │ AUTHORITY        │                         │
│   │                  │        │ (decides+commits)│                         │
│   └──────────────────┘        └──────────────────┘                         │
│                                                                             │
│   Question: "Who allowed the override?"                                     │
│   Answer: "KB-19, with Dr. Smith's identity bound via KB-18"               │
│           ← THIS IS COURT-DEFENSIBLE                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Implementation Phases (Revised for MOVE Approach)

| Phase | Name | Duration | Effort |
|-------|------|----------|--------|
| **Phase 1** | Move Types & Rules to KB-19 | Day 1-2 | 12h |
| **Phase 2** | Move Logic Functions to KB-19 | Day 3-4 | 16h |
| **Phase 3** | Cleanup Med-Advisor + Wire Routes | Day 5 | 8h |
| **Phase 4** | Integration Testing | Day 6 | 4h |
| **Total** | | 2 weeks | 40h |

---

## 2. Phase 1: Move Types & Rules to KB-19

### 2.1 Objective

**MOVE** all type definitions and rule data from Medication Advisor `engine.go` to KB-19 `transaction/` package.

### 2.2 Target File Structure (After Migration)

```
backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/
├── internal/
│   ├── transaction/
│   │   ├── types.go                # MOVE FROM engine.go: HardBlock, GovernanceEvent, etc.
│   │   ├── manager.go              # NEW: Transaction state machine (minimal new code)
│   │   ├── validator.go            # MOVE FROM engine.go: processExcludedDrugs, processDDI, processLab
│   │   ├── committer.go            # MOVE FROM engine.go: generateGovernanceEvents, buildAuditTrail
│   │   ├── conflict_detector.go    # MOVE FROM conflicts.go: entire file
│   │   └── rules/
│   │       ├── ddi_rules.go        # MOVE FROM engine.go: DDIHardStopPair, knownDDIHardStops
│   │       └── lab_rules.go        # MOVE FROM engine.go: LabDrugHardStop, knownLabDrugHardStops
│   ├── clients/
│   │   └── medication_advisor_client.go  # NEW: Call Med-Advisor /risk-profile (small)
│   └── api/
│       └── transaction_handlers.go       # NEW: Wire moved functions to HTTP endpoints
└── pkg/
    └── contracts/
        └── transaction_contracts.go      # NEW: API request/response contracts
```

### 2.3 Task Breakdown

#### Task 1.1: Move Types to KB-19 (3h)

**Source:** `medication-advisor-engine/advisor/engine.go`
**Destination:** `kb-19-protocol-orchestrator/internal/transaction/types.go`

**Copy these types DIRECTLY from engine.go:**

| Type | Source Lines | Action |
|------|--------------|--------|
| `DispositionCode` + constants | 236-268 | Direct copy |
| `GovernanceEventType` + constants | 273-288 | Direct copy |
| `GovernanceEvent` | 291-310 | Direct copy |
| `HardBlock` | 318-342 | Direct copy |
| `ExcludedDrugInfo` | 332-341 | Direct copy |
| `AuditTrailSummary` | 195-221 | Direct copy |
| `GeneratedTask` | 223-234 | Direct copy |

#### Task 1.2: Move DDI Rules to KB-19 (2h)

**Source:** `medication-advisor-engine/advisor/engine.go`
**Destination:** `kb-19-protocol-orchestrator/internal/transaction/rules/ddi_rules.go`

**Copy these DIRECTLY from engine.go:**

| Item | Source Lines | Action |
|------|--------------|--------|
| `DDIHardStopPair` struct | 1072-1083 | Direct copy |
| `knownDDIHardStops` slice | 1087-1177 | Direct copy |
| `drugClassMemberships` map | 1181-1192 | Direct copy |
| `checkDDIPair()` | 1233-1264 | Direct copy |
| `isInDrugClass()` | 1266-1281 | Direct copy |
| `generateDDIAckText()` | 1283-1289 | Direct copy |

#### Task 1.3: Move Lab Rules to KB-19 (2h)

**Source:** `medication-advisor-engine/advisor/engine.go`
**Destination:** `kb-19-protocol-orchestrator/internal/transaction/rules/lab_rules.go`

**Copy these DIRECTLY from engine.go:**

| Item | Source Lines | Action |
|------|--------------|--------|
| `LabDrugHardStop` struct | 1315-1328 | Direct copy |
| `knownLabDrugHardStops` slice | 1332-1432 | Direct copy |
| `labDrugClassMemberships` map | 1435-1439 | Direct copy |
| `labRuleApplies()` | 1494-1518 | Direct copy |
| `findLabValue()` | 1520-1528 | Direct copy |
| `isLabViolation()` | 1530-1539 | Direct copy |
| `generateLabAckText()` | 1541-1547 | Direct copy |
| `float64Ptr()` | 1549-1551 | Direct copy |
| `getLabValueAsFloat64()` | 1554-1574 | Direct copy |

#### Task 1.4: Move ConflictDetector to KB-19 (2h)

**Source:** `medication-advisor-engine/advisor/conflicts.go` (entire file)
**Destination:** `kb-19-protocol-orchestrator/internal/transaction/conflict_detector.go`

**Action:** Copy entire file with package rename from `advisor` to `transaction`

#### Task 1.5: Create Transaction Contracts (3h)

**File:** `pkg/contracts/transaction_contracts.go`

```go
package contracts

import (
    "time"
    "github.com/google/uuid"
)

// ============================================================================
// TRANSACTION VALIDATE
// ============================================================================

// TransactionValidateRequest - Input for medication validation
type TransactionValidateRequest struct {
    PatientID          uuid.UUID          `json:"patient_id" binding:"required"`
    EncounterID        uuid.UUID          `json:"encounter_id" binding:"required"`
    ProposedMedication ProposedMedication `json:"proposed_medication" binding:"required"`
    ProviderID         string             `json:"provider_id" binding:"required"`
    RequestedBy        string             `json:"requested_by" binding:"required"`
    ClinicalContext    *ClinicalContext   `json:"clinical_context,omitempty"`
}

// ProposedMedication - The medication being ordered
type ProposedMedication struct {
    RxNormCode   string   `json:"rxnorm_code" binding:"required"`
    DrugName     string   `json:"drug_name" binding:"required"`
    DoseValue    float64  `json:"dose_value,omitempty"`
    DoseUnit     string   `json:"dose_unit,omitempty"`
    Route        string   `json:"route,omitempty"`
    Frequency    string   `json:"frequency,omitempty"`
    Duration     string   `json:"duration,omitempty"`
    Indication   string   `json:"indication,omitempty"`
}

// ClinicalContext - Patient clinical state
type ClinicalContext struct {
    Age              int                `json:"age"`
    Gender           string             `json:"gender"`
    WeightKg         float64            `json:"weight_kg"`
    EGFR             float64            `json:"egfr"`
    IsPregnant       bool               `json:"is_pregnant"`
    IsLactating      bool               `json:"is_lactating"`
    Conditions       []string           `json:"conditions"`       // ICD-10 codes
    CurrentMeds      []string           `json:"current_meds"`     // RxNorm codes
    Allergies        []string           `json:"allergies"`        // RxNorm codes
    RecentLabs       map[string]float64 `json:"recent_labs"`      // LOINC → value
}

// TransactionValidateResponse - Output from validation
type TransactionValidateResponse struct {
    TransactionID    uuid.UUID           `json:"transaction_id"`
    Status           ValidationStatus    `json:"status"`
    Allowed          bool                `json:"allowed"`
    HardBlocks       []HardBlock         `json:"hard_blocks,omitempty"`
    SoftWarnings     []SoftWarning       `json:"soft_warnings,omitempty"`
    RiskProfile      *RiskProfile        `json:"risk_profile"`
    RequiresOverride bool                `json:"requires_override"`
    OverrideOptions  []OverrideOption    `json:"override_options,omitempty"`
    ExpiresAt        time.Time           `json:"expires_at"`
    NextStep         string              `json:"next_step"`
}

type ValidationStatus string

const (
    ValidationStatusApproved       ValidationStatus = "APPROVED"
    ValidationStatusBlockedHard    ValidationStatus = "BLOCKED_HARD"
    ValidationStatusBlockedSoft    ValidationStatus = "BLOCKED_SOFT"
    ValidationStatusRequiresReview ValidationStatus = "REQUIRES_REVIEW"
)

// HardBlock - Non-overrideable block (or requires special override)
type HardBlock struct {
    ID           uuid.UUID `json:"id"`
    Type         string    `json:"type"`          // DDI, LAB_CONTRAINDICATION, PREGNANCY, etc.
    Severity     string    `json:"severity"`      // CRITICAL, HIGH
    Reason       string    `json:"reason"`
    Source       string    `json:"source"`        // KB-4, KB-5, KB-16
    SourceCode   string    `json:"source_code"`   // RxNorm, LOINC
    Overrideable bool      `json:"overrideable"`
    OverrideType string    `json:"override_type,omitempty"` // ATTENDING_APPROVAL, etc.
}

// SoftWarning - Warning that can be acknowledged
type SoftWarning struct {
    ID       uuid.UUID `json:"id"`
    Type     string    `json:"type"`
    Severity string    `json:"severity"` // MEDIUM, LOW
    Message  string    `json:"message"`
    Source   string    `json:"source"`
}

// RiskProfile - Comprehensive risk assessment from Med-Advisor
type RiskProfile struct {
    OverallRisk      string             `json:"overall_risk"` // HIGH, MEDIUM, LOW
    DDIRisks         []DDIRisk          `json:"ddi_risks"`
    LabRisks         []LabRisk          `json:"lab_risks"`
    PregnancyRisk    *PregnancyRisk     `json:"pregnancy_risk,omitempty"`
    RenalRisk        *RenalRisk         `json:"renal_risk,omitempty"`
    HepaticRisk      *HepaticRisk       `json:"hepatic_risk,omitempty"`
    GeriatricRisk    *GeriatricRisk     `json:"geriatric_risk,omitempty"`
    QualityScores    map[string]float64 `json:"quality_scores"`
}

type DDIRisk struct {
    InteractingDrug string  `json:"interacting_drug"`
    InteractingCode string  `json:"interacting_code"`
    Severity        string  `json:"severity"`        // CRITICAL, MAJOR, MODERATE, MINOR
    Mechanism       string  `json:"mechanism"`
    ClinicalEffect  string  `json:"clinical_effect"`
    ManagementNote  string  `json:"management_note"`
    Source          string  `json:"source"`          // KB-5
}

type LabRisk struct {
    LabCode         string  `json:"lab_code"`        // LOINC
    LabName         string  `json:"lab_name"`
    CurrentValue    float64 `json:"current_value"`
    Unit            string  `json:"unit"`
    RiskThreshold   float64 `json:"risk_threshold"`
    RiskType        string  `json:"risk_type"`       // CONTRAINDICATION, DOSE_ADJUSTMENT
    ClinicalConcern string  `json:"clinical_concern"`
    Source          string  `json:"source"`          // KB-16
}

type PregnancyRisk struct {
    Category        string `json:"category"`         // A, B, C, D, X
    RiskLevel       string `json:"risk_level"`
    FDAWarning      string `json:"fda_warning"`
    Recommendation  string `json:"recommendation"`
}

type RenalRisk struct {
    EGFR            float64 `json:"egfr"`
    CKDStage        string  `json:"ckd_stage"`
    DoseAdjustment  string  `json:"dose_adjustment"`
    Contraindicated bool    `json:"contraindicated"`
}

type HepaticRisk struct {
    ChildPughScore  string `json:"child_pugh_score"`
    DoseAdjustment  string `json:"dose_adjustment"`
    Contraindicated bool   `json:"contraindicated"`
}

type GeriatricRisk struct {
    BeersListMatch  bool   `json:"beers_list_match"`
    BeersCategory   string `json:"beers_category"`
    ACBScore        int    `json:"acb_score"`
    FallRisk        string `json:"fall_risk"`
    Recommendation  string `json:"recommendation"`
}

// OverrideOption - Available override types
type OverrideOption struct {
    Type             string   `json:"type"`
    DisplayName      string   `json:"display_name"`
    RequiresReason   bool     `json:"requires_reason"`
    RequiresApprover bool     `json:"requires_approver"`
    AllowedApprovers []string `json:"allowed_approvers,omitempty"`
}

// ============================================================================
// TRANSACTION COMMIT
// ============================================================================

// TransactionCommitRequest - Input for committing a transaction
type TransactionCommitRequest struct {
    TransactionID   uuid.UUID          `json:"transaction_id" binding:"required"`
    ProviderID      string             `json:"provider_id" binding:"required"`
    Acknowledged    bool               `json:"acknowledged" binding:"required"`
    Overrides       []OverrideDecision `json:"overrides,omitempty"`
    ClinicalNotes   string             `json:"clinical_notes,omitempty"`
}

// OverrideDecision - Provider's override decision
type OverrideDecision struct {
    BlockID      uuid.UUID `json:"block_id"`
    OverrideType string    `json:"override_type"`
    Reason       string    `json:"reason"`
    ApproverID   string    `json:"approver_id,omitempty"`
}

// TransactionCommitResponse - Output from commit
type TransactionCommitResponse struct {
    CommitID         uuid.UUID          `json:"commit_id"`
    TransactionID    uuid.UUID          `json:"transaction_id"`
    Status           CommitStatus       `json:"status"`
    FHIRResourceID   string             `json:"fhir_resource_id,omitempty"`
    FHIRResourceType string             `json:"fhir_resource_type,omitempty"`
    GovernanceEvents []GovernanceEvent  `json:"governance_events"`
    AuditTrailID     uuid.UUID          `json:"audit_trail_id"`
    CommittedAt      time.Time          `json:"committed_at"`
    CommittedBy      string             `json:"committed_by"`
}

type CommitStatus string

const (
    CommitStatusSuccess  CommitStatus = "SUCCESS"
    CommitStatusFailed   CommitStatus = "FAILED"
    CommitStatusPending  CommitStatus = "PENDING"
)

// GovernanceEvent - Tier-7 audit event
type GovernanceEvent struct {
    ID            uuid.UUID `json:"id"`
    EventType     string    `json:"event_type"`
    Timestamp     time.Time `json:"timestamp"`
    ActorID       string    `json:"actor_id"`
    ActorType     string    `json:"actor_type"`
    Action        string    `json:"action"`
    Target        string    `json:"target"`
    Outcome       string    `json:"outcome"`
    EvidenceHash  string    `json:"evidence_hash"`
    ImmutableHash string    `json:"immutable_hash"`
}

// ============================================================================
// TRANSACTION OVERRIDE
// ============================================================================

// TransactionOverrideRequest - Input for override workflow
type TransactionOverrideRequest struct {
    TransactionID     uuid.UUID `json:"transaction_id" binding:"required"`
    BlockID           uuid.UUID `json:"block_id" binding:"required"`
    ProviderID        string    `json:"provider_id" binding:"required"`
    OverrideType      string    `json:"override_type" binding:"required"`
    Reason            string    `json:"reason" binding:"required"`
    AcknowledgmentText string   `json:"acknowledgment_text" binding:"required"`
    ApproverID        string    `json:"approver_id,omitempty"`
}

// TransactionOverrideResponse - Output from override
type TransactionOverrideResponse struct {
    OverrideID      uuid.UUID `json:"override_id"`
    TransactionID   uuid.UUID `json:"transaction_id"`
    Status          string    `json:"status"`
    IdentityBound   bool      `json:"identity_bound"`
    AuditRecorded   bool      `json:"audit_recorded"`
    KB18RecordID    string    `json:"kb18_record_id"`
    NextStep        string    `json:"next_step"`
}

// ============================================================================
// AUDIT TRAIL
// ============================================================================

// TransactionAuditResponse - Complete audit trail
type TransactionAuditResponse struct {
    TransactionID    uuid.UUID           `json:"transaction_id"`
    PatientID        uuid.UUID           `json:"patient_id"`
    Timeline         []AuditEvent        `json:"timeline"`
    FinalOutcome     string              `json:"final_outcome"`
    ProviderChain    []ProviderAction    `json:"provider_chain"`
    EvidenceHashes   []string            `json:"evidence_hashes"`
    CourtDefensible  bool                `json:"court_defensible"`
    ComplianceFlags  []string            `json:"compliance_flags"`
}

type AuditEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    EventType   string    `json:"event_type"`
    Actor       string    `json:"actor"`
    Action      string    `json:"action"`
    Details     string    `json:"details"`
    Hash        string    `json:"hash"`
}

type ProviderAction struct {
    ProviderID   string    `json:"provider_id"`
    ProviderName string    `json:"provider_name"`
    Role         string    `json:"role"`
    Action       string    `json:"action"`
    Timestamp    time.Time `json:"timestamp"`
    Signature    string    `json:"signature,omitempty"`
}
```

**Acceptance Criteria:**
- [ ] All types compile without errors
- [ ] JSON tags match API specification
- [ ] Validation bindings are correct

---

#### Task 1.2: Create Medication Advisor Client (3h)

**File:** `internal/clients/medication_advisor_client.go`

```go
package clients

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/google/uuid"
)

// MedicationAdvisorClient calls Med-Advisor (8095) for risk profiles
type MedicationAdvisorClient struct {
    baseURL    string
    httpClient *http.Client
}

// NewMedicationAdvisorClient creates a new client
func NewMedicationAdvisorClient(baseURL string) *MedicationAdvisorClient {
    return &MedicationAdvisorClient{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// RiskProfileRequest - What we send to Med-Advisor
type RiskProfileRequest struct {
    PatientID          uuid.UUID          `json:"patient_id"`
    ProposedMedication ProposedMedication `json:"proposed_medication"`
    PatientContext     PatientContext     `json:"patient_context"`
}

type ProposedMedication struct {
    RxNormCode string  `json:"rxnorm_code"`
    DrugName   string  `json:"drug_name"`
    DoseValue  float64 `json:"dose_value,omitempty"`
    DoseUnit   string  `json:"dose_unit,omitempty"`
}

type PatientContext struct {
    Age          int                `json:"age"`
    Gender       string             `json:"gender"`
    WeightKg     float64            `json:"weight_kg"`
    EGFR         float64            `json:"egfr"`
    IsPregnant   bool               `json:"is_pregnant"`
    IsLactating  bool               `json:"is_lactating"`
    Conditions   []string           `json:"conditions"`
    CurrentMeds  []string           `json:"current_meds"`
    Allergies    []string           `json:"allergies"`
    RecentLabs   map[string]float64 `json:"recent_labs"`
}

// RiskProfileResponse - What Med-Advisor returns (V3 format - NO HardBlocks)
type RiskProfileResponse struct {
    SnapshotID      uuid.UUID          `json:"snapshot_id"`
    OverallRisk     string             `json:"overall_risk"`
    DDIRisks        []DDIRiskScore     `json:"ddi_risks"`
    LabRisks        []LabRiskScore     `json:"lab_risks"`
    PregnancyRisk   *PregnancyRiskInfo `json:"pregnancy_risk,omitempty"`
    RenalRisk       *RenalRiskInfo     `json:"renal_risk,omitempty"`
    HepaticRisk     *HepaticRiskInfo   `json:"hepatic_risk,omitempty"`
    GeriatricRisk   *GeriatricRiskInfo `json:"geriatric_risk,omitempty"`
    DosingAdvice    *DosingAdvice      `json:"dosing_advice,omitempty"`
    QualityScores   map[string]float64 `json:"quality_scores"`
    KBVersions      map[string]string  `json:"kb_versions"`
    ExecutionTimeMs int64              `json:"execution_time_ms"`
}

type DDIRiskScore struct {
    InteractingDrug string `json:"interacting_drug"`
    InteractingCode string `json:"interacting_code"`
    Severity        string `json:"severity"`
    SeverityScore   int    `json:"severity_score"` // 1-10
    Mechanism       string `json:"mechanism"`
    ClinicalEffect  string `json:"clinical_effect"`
    ManagementNote  string `json:"management_note"`
    Source          string `json:"source"`
}

type LabRiskScore struct {
    LabCode         string  `json:"lab_code"`
    LabName         string  `json:"lab_name"`
    CurrentValue    float64 `json:"current_value"`
    Unit            string  `json:"unit"`
    RiskThreshold   float64 `json:"risk_threshold"`
    RiskScore       int     `json:"risk_score"` // 1-10
    RiskType        string  `json:"risk_type"`
    ClinicalConcern string  `json:"clinical_concern"`
}

type PregnancyRiskInfo struct {
    Category       string `json:"category"`
    RiskScore      int    `json:"risk_score"`
    FDAWarning     string `json:"fda_warning"`
    Recommendation string `json:"recommendation"`
}

type RenalRiskInfo struct {
    EGFR            float64 `json:"egfr"`
    CKDStage        string  `json:"ckd_stage"`
    RiskScore       int     `json:"risk_score"`
    DoseAdjustment  string  `json:"dose_adjustment"`
    Contraindicated bool    `json:"contraindicated"`
}

type HepaticRiskInfo struct {
    ChildPughScore  string `json:"child_pugh_score"`
    RiskScore       int    `json:"risk_score"`
    DoseAdjustment  string `json:"dose_adjustment"`
    Contraindicated bool   `json:"contraindicated"`
}

type GeriatricRiskInfo struct {
    BeersListMatch bool   `json:"beers_list_match"`
    BeersCategory  string `json:"beers_category"`
    ACBScore       int    `json:"acb_score"`
    FallRiskScore  int    `json:"fall_risk_score"`
    Recommendation string `json:"recommendation"`
}

type DosingAdvice struct {
    RecommendedDose string `json:"recommended_dose"`
    MaxDose         string `json:"max_dose"`
    Frequency       string `json:"frequency"`
    AdjustmentNote  string `json:"adjustment_note"`
}

// GetRiskProfile calls Med-Advisor to get comprehensive risk profile
func (c *MedicationAdvisorClient) GetRiskProfile(ctx context.Context, req *RiskProfileRequest) (*RiskProfileResponse, error) {
    url := fmt.Sprintf("%s/api/v1/advisor/risk-profile", c.baseURL)

    // TODO: Implement HTTP POST call
    // This is the NEW endpoint that Med-Advisor will expose after refactor

    return nil, fmt.Errorf("not implemented - Med-Advisor refactor required first")
}

// Health checks Med-Advisor health
func (c *MedicationAdvisorClient) Health(ctx context.Context) error {
    url := fmt.Sprintf("%s/health", c.baseURL)

    resp, err := c.httpClient.Get(url)
    if err != nil {
        return fmt.Errorf("medication advisor health check failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("medication advisor unhealthy: status %d", resp.StatusCode)
    }

    return nil
}
```

**Acceptance Criteria:**
- [ ] Client can call Med-Advisor `/health`
- [ ] Client has typed request/response for risk profile
- [ ] Error handling is comprehensive

---

#### Task 1.3: Create Transaction Manager (4h)

**File:** `internal/transaction/manager.go`

```go
package transaction

import (
    "context"
    "sync"
    "time"

    "github.com/google/uuid"

    "kb-19-protocol-orchestrator/pkg/contracts"
)

// TransactionState represents the state of a transaction
type TransactionState string

const (
    StateCreated     TransactionState = "CREATED"
    StateValidating  TransactionState = "VALIDATING"
    StateValidated   TransactionState = "VALIDATED"
    StateBlocked     TransactionState = "BLOCKED"
    StateOverriding  TransactionState = "OVERRIDING"
    StateCommitting  TransactionState = "COMMITTING"
    StateCommitted   TransactionState = "COMMITTED"
    StateFailed      TransactionState = "FAILED"
    StateExpired     TransactionState = "EXPIRED"
)

// Transaction represents a medication transaction
type Transaction struct {
    ID                uuid.UUID
    PatientID         uuid.UUID
    EncounterID       uuid.UUID
    ProposedMedication contracts.ProposedMedication
    ProviderID        string
    State             TransactionState
    RiskProfile       *contracts.RiskProfile
    HardBlocks        []contracts.HardBlock
    SoftWarnings      []contracts.SoftWarning
    Overrides         []contracts.OverrideDecision
    CreatedAt         time.Time
    UpdatedAt         time.Time
    ExpiresAt         time.Time
    CommitResult      *contracts.TransactionCommitResponse
    AuditEvents       []AuditEvent
}

// AuditEvent for transaction history
type AuditEvent struct {
    Timestamp time.Time
    EventType string
    Actor     string
    Details   string
}

// Manager manages transaction lifecycle
type Manager struct {
    mu           sync.RWMutex
    transactions map[uuid.UUID]*Transaction
    ttl          time.Duration
}

// NewManager creates a new transaction manager
func NewManager(ttl time.Duration) *Manager {
    m := &Manager{
        transactions: make(map[uuid.UUID]*Transaction),
        ttl:          ttl,
    }
    go m.cleanup()
    return m
}

// Create creates a new transaction
func (m *Manager) Create(ctx context.Context, req *contracts.TransactionValidateRequest) (*Transaction, error) {
    txn := &Transaction{
        ID:                 uuid.New(),
        PatientID:          req.PatientID,
        EncounterID:        req.EncounterID,
        ProposedMedication: req.ProposedMedication,
        ProviderID:         req.ProviderID,
        State:              StateCreated,
        CreatedAt:          time.Now(),
        UpdatedAt:          time.Now(),
        ExpiresAt:          time.Now().Add(m.ttl),
        AuditEvents:        make([]AuditEvent, 0),
    }

    txn.addAuditEvent("TRANSACTION_CREATED", req.RequestedBy, "Transaction initiated")

    m.mu.Lock()
    m.transactions[txn.ID] = txn
    m.mu.Unlock()

    return txn, nil
}

// Get retrieves a transaction by ID
func (m *Manager) Get(ctx context.Context, id uuid.UUID) (*Transaction, error) {
    m.mu.RLock()
    txn, ok := m.transactions[id]
    m.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("transaction not found: %s", id)
    }

    if time.Now().After(txn.ExpiresAt) && txn.State != StateCommitted {
        txn.State = StateExpired
        return nil, fmt.Errorf("transaction expired: %s", id)
    }

    return txn, nil
}

// UpdateState updates transaction state
func (m *Manager) UpdateState(ctx context.Context, id uuid.UUID, state TransactionState, actor string, details string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    txn, ok := m.transactions[id]
    if !ok {
        return fmt.Errorf("transaction not found: %s", id)
    }

    txn.State = state
    txn.UpdatedAt = time.Now()
    txn.addAuditEvent("STATE_CHANGED", actor, fmt.Sprintf("%s: %s", state, details))

    return nil
}

// SetRiskProfile sets the risk profile from Med-Advisor
func (m *Manager) SetRiskProfile(ctx context.Context, id uuid.UUID, profile *contracts.RiskProfile) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    txn, ok := m.transactions[id]
    if !ok {
        return fmt.Errorf("transaction not found: %s", id)
    }

    txn.RiskProfile = profile
    txn.UpdatedAt = time.Now()

    return nil
}

// SetHardBlocks sets the hard blocks (determined by KB-19, not Med-Advisor)
func (m *Manager) SetHardBlocks(ctx context.Context, id uuid.UUID, blocks []contracts.HardBlock) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    txn, ok := m.transactions[id]
    if !ok {
        return fmt.Errorf("transaction not found: %s", id)
    }

    txn.HardBlocks = blocks
    txn.UpdatedAt = time.Now()

    if len(blocks) > 0 {
        txn.State = StateBlocked
        txn.addAuditEvent("BLOCKED", "KB-19", fmt.Sprintf("%d hard blocks applied", len(blocks)))
    }

    return nil
}

// AddOverride adds an override decision
func (m *Manager) AddOverride(ctx context.Context, id uuid.UUID, override contracts.OverrideDecision) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    txn, ok := m.transactions[id]
    if !ok {
        return fmt.Errorf("transaction not found: %s", id)
    }

    txn.Overrides = append(txn.Overrides, override)
    txn.UpdatedAt = time.Now()
    txn.addAuditEvent("OVERRIDE_ADDED", override.ApproverID, override.Reason)

    return nil
}

// helper to add audit event
func (t *Transaction) addAuditEvent(eventType, actor, details string) {
    t.AuditEvents = append(t.AuditEvents, AuditEvent{
        Timestamp: time.Now(),
        EventType: eventType,
        Actor:     actor,
        Details:   details,
    })
}

// cleanup removes expired transactions
func (m *Manager) cleanup() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        m.mu.Lock()
        now := time.Now()
        for id, txn := range m.transactions {
            if now.Sub(txn.CreatedAt) > time.Hour && txn.State != StateCommitted {
                delete(m.transactions, id)
            }
        }
        m.mu.Unlock()
    }
}
```

**Acceptance Criteria:**
- [ ] Transaction state machine works correctly
- [ ] Audit events are recorded
- [ ] Expiration logic works

---

#### Task 1.4: Create Transaction Validator (4h)

**File:** `internal/transaction/validator.go`

```go
package transaction

import (
    "context"
    "fmt"

    "github.com/google/uuid"

    "kb-19-protocol-orchestrator/internal/clients"
    "kb-19-protocol-orchestrator/pkg/contracts"
)

// Validator validates medication transactions
type Validator struct {
    medAdvisorClient *clients.MedicationAdvisorClient
    // Add KB-18 client for governance rules
    // Add hospital policy rules
}

// NewValidator creates a new validator
func NewValidator(medAdvisorClient *clients.MedicationAdvisorClient) *Validator {
    return &Validator{
        medAdvisorClient: medAdvisorClient,
    }
}

// Validate performs comprehensive validation
func (v *Validator) Validate(ctx context.Context, txn *Transaction) (*contracts.TransactionValidateResponse, error) {
    // Step 1: Get risk profile from Med-Advisor
    riskProfile, err := v.getRiskProfile(ctx, txn)
    if err != nil {
        return nil, fmt.Errorf("failed to get risk profile: %w", err)
    }

    // Step 2: Convert risk scores to hard blocks (KB-19 DECISION)
    hardBlocks := v.evaluateHardBlocks(riskProfile)

    // Step 3: Extract soft warnings
    softWarnings := v.extractSoftWarnings(riskProfile)

    // Step 4: Determine override options
    overrideOptions := v.determineOverrideOptions(hardBlocks)

    // Step 5: Determine final status
    status := v.determineStatus(hardBlocks, softWarnings)

    return &contracts.TransactionValidateResponse{
        TransactionID:    txn.ID,
        Status:           status,
        Allowed:          status == contracts.ValidationStatusApproved,
        HardBlocks:       hardBlocks,
        SoftWarnings:     softWarnings,
        RiskProfile:      riskProfile,
        RequiresOverride: len(hardBlocks) > 0 && v.anyOverrideable(hardBlocks),
        OverrideOptions:  overrideOptions,
        ExpiresAt:        txn.ExpiresAt,
        NextStep:         v.determineNextStep(status, hardBlocks),
    }, nil
}

// getRiskProfile calls Med-Advisor for risk assessment
func (v *Validator) getRiskProfile(ctx context.Context, txn *Transaction) (*contracts.RiskProfile, error) {
    // TODO: Call Med-Advisor when refactored
    // For now, return mock profile
    return &contracts.RiskProfile{
        OverallRisk:   "MEDIUM",
        DDIRisks:      []contracts.DDIRisk{},
        LabRisks:      []contracts.LabRisk{},
        QualityScores: map[string]float64{"safety": 0.8, "efficacy": 0.9},
    }, nil
}

// evaluateHardBlocks converts risk scores to hard block decisions
// THIS IS WHERE KB-19 MAKES THE ENFORCEMENT DECISION
func (v *Validator) evaluateHardBlocks(profile *contracts.RiskProfile) []contracts.HardBlock {
    blocks := make([]contracts.HardBlock, 0)

    // DDI Hard Blocks - CRITICAL severity = hard block
    for _, ddi := range profile.DDIRisks {
        if ddi.Severity == "CRITICAL" {
            blocks = append(blocks, contracts.HardBlock{
                ID:           uuid.New(),
                Type:         "DDI",
                Severity:     "CRITICAL",
                Reason:       fmt.Sprintf("%s: %s", ddi.ClinicalEffect, ddi.ManagementNote),
                Source:       ddi.Source,
                SourceCode:   ddi.InteractingCode,
                Overrideable: false, // CRITICAL DDIs are NOT overrideable
            })
        } else if ddi.Severity == "MAJOR" {
            blocks = append(blocks, contracts.HardBlock{
                ID:           uuid.New(),
                Type:         "DDI",
                Severity:     "HIGH",
                Reason:       fmt.Sprintf("%s: %s", ddi.ClinicalEffect, ddi.ManagementNote),
                Source:       ddi.Source,
                SourceCode:   ddi.InteractingCode,
                Overrideable: true,
                OverrideType: "ATTENDING_APPROVAL",
            })
        }
    }

    // Lab Hard Blocks - Contraindication = hard block
    for _, lab := range profile.LabRisks {
        if lab.RiskType == "CONTRAINDICATION" {
            blocks = append(blocks, contracts.HardBlock{
                ID:           uuid.New(),
                Type:         "LAB_CONTRAINDICATION",
                Severity:     "CRITICAL",
                Reason:       lab.ClinicalConcern,
                Source:       lab.Source,
                SourceCode:   lab.LabCode,
                Overrideable: false,
            })
        }
    }

    // Pregnancy Hard Blocks - Category X = hard block
    if profile.PregnancyRisk != nil && profile.PregnancyRisk.Category == "X" {
        blocks = append(blocks, contracts.HardBlock{
            ID:           uuid.New(),
            Type:         "PREGNANCY",
            Severity:     "CRITICAL",
            Reason:       profile.PregnancyRisk.FDAWarning,
            Source:       "KB-4",
            Overrideable: false,
        })
    }

    return blocks
}

// extractSoftWarnings extracts warnings that can be acknowledged
func (v *Validator) extractSoftWarnings(profile *contracts.RiskProfile) []contracts.SoftWarning {
    warnings := make([]contracts.SoftWarning, 0)

    // Moderate DDIs = soft warning
    for _, ddi := range profile.DDIRisks {
        if ddi.Severity == "MODERATE" {
            warnings = append(warnings, contracts.SoftWarning{
                ID:       uuid.New(),
                Type:     "DDI",
                Severity: "MEDIUM",
                Message:  fmt.Sprintf("%s: %s", ddi.ClinicalEffect, ddi.ManagementNote),
                Source:   ddi.Source,
            })
        }
    }

    // Geriatric warnings
    if profile.GeriatricRisk != nil && profile.GeriatricRisk.BeersListMatch {
        warnings = append(warnings, contracts.SoftWarning{
            ID:       uuid.New(),
            Type:     "BEERS_CRITERIA",
            Severity: "MEDIUM",
            Message:  profile.GeriatricRisk.Recommendation,
            Source:   "KB-4",
        })
    }

    return warnings
}

func (v *Validator) determineOverrideOptions(blocks []contracts.HardBlock) []contracts.OverrideOption {
    options := make([]contracts.OverrideOption, 0)

    for _, block := range blocks {
        if block.Overrideable {
            options = append(options, contracts.OverrideOption{
                Type:             block.OverrideType,
                DisplayName:      v.getOverrideDisplayName(block.OverrideType),
                RequiresReason:   true,
                RequiresApprover: block.OverrideType == "ATTENDING_APPROVAL",
            })
        }
    }

    return options
}

func (v *Validator) getOverrideDisplayName(overrideType string) string {
    switch overrideType {
    case "ATTENDING_APPROVAL":
        return "Attending Physician Approval"
    case "PHARMACIST_REVIEW":
        return "Pharmacist Review"
    case "BENEFIT_OUTWEIGHS_RISK":
        return "Benefit Outweighs Risk Acknowledgment"
    default:
        return overrideType
    }
}

func (v *Validator) determineStatus(blocks []contracts.HardBlock, warnings []contracts.SoftWarning) contracts.ValidationStatus {
    hasNonOverrideable := false
    for _, block := range blocks {
        if !block.Overrideable {
            hasNonOverrideable = true
            break
        }
    }

    if hasNonOverrideable {
        return contracts.ValidationStatusBlockedHard
    }

    if len(blocks) > 0 {
        return contracts.ValidationStatusBlockedSoft
    }

    if len(warnings) > 0 {
        return contracts.ValidationStatusRequiresReview
    }

    return contracts.ValidationStatusApproved
}

func (v *Validator) anyOverrideable(blocks []contracts.HardBlock) bool {
    for _, block := range blocks {
        if block.Overrideable {
            return true
        }
    }
    return false
}

func (v *Validator) determineNextStep(status contracts.ValidationStatus, blocks []contracts.HardBlock) string {
    switch status {
    case contracts.ValidationStatusApproved:
        return "POST /api/v1/transaction/commit"
    case contracts.ValidationStatusBlockedHard:
        return "Cannot proceed - non-overrideable block"
    case contracts.ValidationStatusBlockedSoft:
        return "POST /api/v1/transaction/override"
    case contracts.ValidationStatusRequiresReview:
        return "POST /api/v1/transaction/commit with acknowledged=true"
    default:
        return "Unknown"
    }
}
```

**Acceptance Criteria:**
- [ ] Risk scores converted to hard blocks correctly
- [ ] Override options determined correctly
- [ ] Status determination logic is correct

---

#### Task 1.5: Create Transaction Handlers (4h)

**File:** `internal/api/transaction_handlers.go`

```go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "kb-19-protocol-orchestrator/internal/transaction"
    "kb-19-protocol-orchestrator/pkg/contracts"
)

// handleTransactionValidate handles POST /api/v1/transaction/validate
func (s *Server) handleTransactionValidate(c *gin.Context) {
    var req contracts.TransactionValidateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": err.Error(),
        })
        return
    }

    // Create transaction
    txn, err := s.txnManager.Create(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "transaction_creation_failed",
            "message": err.Error(),
        })
        return
    }

    // Update state
    s.txnManager.UpdateState(c.Request.Context(), txn.ID, transaction.StateValidating, req.RequestedBy, "Validation started")

    // Validate
    resp, err := s.validator.Validate(c.Request.Context(), txn)
    if err != nil {
        s.txnManager.UpdateState(c.Request.Context(), txn.ID, transaction.StateFailed, "system", err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "validation_failed",
            "message": err.Error(),
        })
        return
    }

    // Update transaction with results
    s.txnManager.SetRiskProfile(c.Request.Context(), txn.ID, resp.RiskProfile)
    s.txnManager.SetHardBlocks(c.Request.Context(), txn.ID, resp.HardBlocks)

    // Update state
    if resp.Allowed {
        s.txnManager.UpdateState(c.Request.Context(), txn.ID, transaction.StateValidated, req.RequestedBy, "Validation approved")
    } else {
        s.txnManager.UpdateState(c.Request.Context(), txn.ID, transaction.StateBlocked, "KB-19", "Validation blocked")
    }

    c.JSON(http.StatusOK, resp)
}

// handleTransactionCommit handles POST /api/v1/transaction/commit
func (s *Server) handleTransactionCommit(c *gin.Context) {
    var req contracts.TransactionCommitRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": err.Error(),
        })
        return
    }

    // Get transaction
    txn, err := s.txnManager.Get(c.Request.Context(), req.TransactionID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error":   "transaction_not_found",
            "message": err.Error(),
        })
        return
    }

    // Validate state
    if txn.State != transaction.StateValidated && txn.State != transaction.StateOverriding {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_state",
            "message": "Transaction must be validated before commit",
            "state":   txn.State,
        })
        return
    }

    // Update state
    s.txnManager.UpdateState(c.Request.Context(), txn.ID, transaction.StateCommitting, req.ProviderID, "Commit started")

    // Commit
    resp, err := s.committer.Commit(c.Request.Context(), txn, &req)
    if err != nil {
        s.txnManager.UpdateState(c.Request.Context(), txn.ID, transaction.StateFailed, "system", err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "commit_failed",
            "message": err.Error(),
        })
        return
    }

    // Update state
    s.txnManager.UpdateState(c.Request.Context(), txn.ID, transaction.StateCommitted, req.ProviderID, "Commit successful")

    c.JSON(http.StatusOK, resp)
}

// handleTransactionOverride handles POST /api/v1/transaction/override
func (s *Server) handleTransactionOverride(c *gin.Context) {
    var req contracts.TransactionOverrideRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_request",
            "message": err.Error(),
        })
        return
    }

    // Get transaction
    txn, err := s.txnManager.Get(c.Request.Context(), req.TransactionID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error":   "transaction_not_found",
            "message": err.Error(),
        })
        return
    }

    // Validate state
    if txn.State != transaction.StateBlocked {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_state",
            "message": "Transaction must be blocked to override",
            "state":   txn.State,
        })
        return
    }

    // Find the block being overridden
    var targetBlock *contracts.HardBlock
    for _, block := range txn.HardBlocks {
        if block.ID == req.BlockID {
            targetBlock = &block
            break
        }
    }

    if targetBlock == nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error":   "block_not_found",
            "message": "Block ID not found in transaction",
        })
        return
    }

    if !targetBlock.Overrideable {
        c.JSON(http.StatusForbidden, gin.H{
            "error":   "not_overrideable",
            "message": "This block cannot be overridden",
        })
        return
    }

    // Process override with identity binding
    resp, err := s.overrideProcessor.Process(c.Request.Context(), txn, targetBlock, &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "override_failed",
            "message": err.Error(),
        })
        return
    }

    // Add override to transaction
    s.txnManager.AddOverride(c.Request.Context(), txn.ID, contracts.OverrideDecision{
        BlockID:      req.BlockID,
        OverrideType: req.OverrideType,
        Reason:       req.Reason,
        ApproverID:   req.ApproverID,
    })

    // Update state
    s.txnManager.UpdateState(c.Request.Context(), txn.ID, transaction.StateValidated, req.ProviderID, "Override approved")

    c.JSON(http.StatusOK, resp)
}

// handleTransactionAudit handles GET /api/v1/transaction/:id/audit
func (s *Server) handleTransactionAudit(c *gin.Context) {
    txnIDStr := c.Param("id")
    txnID, err := uuid.Parse(txnIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid_transaction_id",
            "message": "Transaction ID must be a valid UUID",
        })
        return
    }

    // Get transaction
    txn, err := s.txnManager.Get(c.Request.Context(), txnID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error":   "transaction_not_found",
            "message": err.Error(),
        })
        return
    }

    // Build audit response
    resp := s.buildAuditResponse(txn)

    c.JSON(http.StatusOK, resp)
}

func (s *Server) buildAuditResponse(txn *transaction.Transaction) *contracts.TransactionAuditResponse {
    timeline := make([]contracts.AuditEvent, len(txn.AuditEvents))
    for i, e := range txn.AuditEvents {
        timeline[i] = contracts.AuditEvent{
            Timestamp: e.Timestamp,
            EventType: e.EventType,
            Actor:     e.Actor,
            Details:   e.Details,
        }
    }

    return &contracts.TransactionAuditResponse{
        TransactionID:   txn.ID,
        PatientID:       txn.PatientID,
        Timeline:        timeline,
        FinalOutcome:    string(txn.State),
        CourtDefensible: txn.State == transaction.StateCommitted,
    }
}
```

**Acceptance Criteria:**
- [ ] All 4 endpoints work
- [ ] State machine transitions correctly
- [ ] Audit trail is complete

---

#### Task 1.6: Wire Up Transaction Routes (3h)

**File:** Update `internal/api/server.go`

```go
// Add to Server struct
type Server struct {
    // ... existing fields
    txnManager       *transaction.Manager
    validator        *transaction.Validator
    committer        *transaction.Committer
    overrideProcessor *transaction.OverrideProcessor
}

// Add to setupRoutes
func (s *Server) setupRoutes() {
    // ... existing routes

    // Transaction API (V3)
    txn := s.router.Group("/api/v1/transaction")
    {
        txn.POST("/validate", s.handleTransactionValidate)
        txn.POST("/commit", s.handleTransactionCommit)
        txn.POST("/override", s.handleTransactionOverride)
        txn.GET("/:id/audit", s.handleTransactionAudit)
    }
}
```

---

## 3. Phase 2: Move Logic Functions to KB-19

### 3.1 Objective

**MOVE** all hard block processing, governance, and audit functions from Medication Advisor to KB-19.

### 3.2 Task Breakdown

#### Task 2.1: Move Block Processing Functions (4h)

**Source:** `medication-advisor-engine/advisor/engine.go`
**Destination:** `kb-19-protocol-orchestrator/internal/transaction/validator.go`

**Functions to MOVE:**

| Function | Source Lines | New Name | Notes |
|----------|--------------|----------|-------|
| `processExcludedDrugs()` | 893-950 | `evaluateExcludedDrugs()` | Add `txn *Transaction` param |
| `processDDIHardBlocks()` | 1195-1230 | `evaluateDDIBlocks()` | Uses moved DDI rules |
| `processLabHardBlocks()` | 1442-1492 | `evaluateLabBlocks()` | Uses moved Lab rules |
| `determineDisposition()` | 747-780 | `determineDisposition()` | Direct copy |

**Modification needed:** Change receiver from `(e *MedicationAdvisorEngine)` to `(v *Validator)` and update dependencies.

#### Task 2.2: Move Governance Functions (4h)

**Source:** `medication-advisor-engine/advisor/engine.go`
**Destination:** `kb-19-protocol-orchestrator/internal/transaction/committer.go`

**Functions to MOVE:**

| Function | Source Lines | Notes |
|----------|--------------|-------|
| `generateGovernanceEvents()` | 566-606 | Direct copy |
| `generateLabSafetyTasks()` | 672-744 | Direct copy |
| `computeEventHash()` | 631-668 | Direct copy |
| `buildAuditTrail()` | 783-852 | Direct copy |

**Modification needed:** Change receiver from `(e *MedicationAdvisorEngine)` to `(c *Committer)`.

#### Task 2.3: Move Validate/Commit Functions (4h)

**Source:** `medication-advisor-engine/advisor/engine.go`
**Destination:** `kb-19-protocol-orchestrator/internal/transaction/`

**Functions to MOVE:**

| Function | Source Lines | Destination | Notes |
|----------|--------------|-------------|-------|
| `Validate()` | 1599-1653 | `validator.go` | Refactor to call moved functions |
| `Commit()` | 1686-1734 | `committer.go` | Refactor to call moved functions |

#### Task 2.4: Create Transaction Manager (4h)

**File:** `kb-19-protocol-orchestrator/internal/transaction/manager.go`

**This is the ONLY significant new code.** Transaction state machine to coordinate the moved functions:

```go
package transaction

// Transaction state machine
type TransactionState string

const (
    StateCreated     TransactionState = "CREATED"
    StateValidating  TransactionState = "VALIDATING"
    StateValidated   TransactionState = "VALIDATED"
    StateBlocked     TransactionState = "BLOCKED"
    StateOverriding  TransactionState = "OVERRIDING"
    StateCommitting  TransactionState = "COMMITTING"
    StateCommitted   TransactionState = "COMMITTED"
    StateFailed      TransactionState = "FAILED"
)

// Transaction holds the state for a medication transaction
// This wraps the moved functions from Med-Advisor
type Transaction struct {
    ID                uuid.UUID
    PatientID         uuid.UUID
    EncounterID       uuid.UUID
    ProposedMedication ProposedMedication
    ProviderID        string
    State             TransactionState

    // These are populated by moved functions
    HardBlocks        []HardBlock       // From evaluateDDIBlocks(), evaluateLabBlocks()
    GovernanceEvents  []GovernanceEvent // From generateGovernanceEvents()
    AuditTrail        *AuditTrailSummary // From buildAuditTrail()
    Disposition       DispositionCode   // From determineDisposition()

    CreatedAt         time.Time
    ExpiresAt         time.Time
}

// Manager coordinates the moved functions
type Manager struct {
    validator  *Validator   // Has moved evaluateDDI/Lab/Excluded functions
    committer  *Committer   // Has moved governance/audit functions
}

func (m *Manager) Validate(ctx context.Context, txn *Transaction) error {
    // Call moved functions in sequence
    txn.HardBlocks = append(txn.HardBlocks, m.validator.evaluateDDIBlocks(...)...)
    txn.HardBlocks = append(txn.HardBlocks, m.validator.evaluateLabBlocks(...)...)
    txn.HardBlocks = append(txn.HardBlocks, m.validator.evaluateExcludedDrugs(...)...)
    txn.Disposition = m.validator.determineDisposition(txn.HardBlocks)
    return nil
}

func (m *Manager) Commit(ctx context.Context, txn *Transaction) error {
    // Call moved functions
    txn.GovernanceEvents = m.committer.generateGovernanceEvents(...)
    txn.AuditTrail = m.committer.buildAuditTrail(...)
    return nil
}
```

---

## 4. Phase 3: Cleanup Med-Advisor + Wire Routes

### 4.1 Objective

**DELETE** moved code from Medication Advisor and **ADD** new `/risk-profile` endpoint. Wire transaction routes in KB-19.

### 4.2 Task Breakdown

#### Task 3.1: Delete Moved Functions from Med-Advisor (3h)

**File:** `medication-advisor-engine/advisor/engine.go`

**Delete these functions (now in KB-19):**

| Function | Lines | Status |
|----------|-------|--------|
| `processExcludedDrugs()` | 893-950 | DELETE |
| `processDDIHardBlocks()` | 1195-1230 | DELETE |
| `processLabHardBlocks()` | 1442-1492 | DELETE |
| `generateGovernanceEvents()` | 566-606 | DELETE |
| `generateLabSafetyTasks()` | 672-744 | DELETE |
| `determineDisposition()` | 747-780 | DELETE |
| `buildAuditTrail()` | 783-852 | DELETE |
| `computeEventHash()` | 631-668 | DELETE |
| `Validate()` | 1599-1653 | DELETE |
| `Commit()` | 1686-1734 | DELETE |
| All DDI/Lab rule definitions | 1072-1439 | DELETE |

**Delete entire file:**
- `conflicts.go` - Moved to KB-19

**Delete types (now in KB-19):**
- `HardBlock`, `GovernanceEvent`, `DispositionCode`, etc.

#### Task 3.2: Add /risk-profile Endpoint to Med-Advisor (2h)

**File:** `medication-advisor-engine/cmd/server/main.go`

**Keep existing `/calculate` for backward compatibility. Add new V3 endpoint:**

```go
// V3 endpoint - returns ONLY risk scores, NO hard blocks
v1.POST("/risk-profile", handleRiskProfile(engine))

func handleRiskProfile(engine *advisor.MedicationAdvisorEngine) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Returns DDIRisks, LabRisks, PregnancyRisk, etc.
        // Does NOT return HardBlocks, Disposition, GovernanceEvents
    }
}
```

#### Task 3.3: Wire Transaction Routes in KB-19 (3h)

**File:** `kb-19-protocol-orchestrator/internal/api/server.go`

**Add transaction routes that call the moved functions:**

```go
// Transaction API (V3) - uses moved functions from Med-Advisor
txn := s.router.Group("/api/v1/transaction")
{
    txn.POST("/validate", s.handleTransactionValidate)   // Calls evaluateDDI/Lab/Excluded
    txn.POST("/commit", s.handleTransactionCommit)       // Calls generateGovernance/buildAudit
    txn.POST("/override", s.handleTransactionOverride)   // New override workflow
    txn.GET("/:id/audit", s.handleTransactionAudit)      // Return audit trail
}
```

---

## 5. Phase 4: Integration & Testing

### 5.1 End-to-End Flow Test

```bash
#!/bin/bash
# test-v3-flow.sh

# Step 1: Validate medication
VALIDATE_RESPONSE=$(curl -s -X POST http://localhost:8119/api/v1/transaction/validate \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "encounter_id": "550e8400-e29b-41d4-a716-446655440001",
    "proposed_medication": {
      "rxnorm_code": "6809",
      "drug_name": "Metformin",
      "dose_value": 500,
      "dose_unit": "mg"
    },
    "provider_id": "DR001",
    "requested_by": "Dr. Smith",
    "clinical_context": {
      "age": 65,
      "egfr": 25,
      "is_pregnant": false,
      "current_meds": ["197381"],
      "recent_labs": {"33914-3": 25}
    }
  }')

echo "Validate Response:"
echo $VALIDATE_RESPONSE | jq .

# Extract transaction ID
TXN_ID=$(echo $VALIDATE_RESPONSE | jq -r '.transaction_id')
STATUS=$(echo $VALIDATE_RESPONSE | jq -r '.status')

# Step 2: If blocked, try override (if overrideable)
if [ "$STATUS" == "BLOCKED_SOFT" ]; then
  BLOCK_ID=$(echo $VALIDATE_RESPONSE | jq -r '.hard_blocks[0].id')

  OVERRIDE_RESPONSE=$(curl -s -X POST http://localhost:8119/api/v1/transaction/override \
    -H "Content-Type: application/json" \
    -d "{
      \"transaction_id\": \"$TXN_ID\",
      \"block_id\": \"$BLOCK_ID\",
      \"provider_id\": \"DR001\",
      \"override_type\": \"BENEFIT_OUTWEIGHS_RISK\",
      \"reason\": \"Patient has been stable on this medication\",
      \"acknowledgment_text\": \"I acknowledge the risk\"
    }")

  echo "Override Response:"
  echo $OVERRIDE_RESPONSE | jq .
fi

# Step 3: Commit
COMMIT_RESPONSE=$(curl -s -X POST http://localhost:8119/api/v1/transaction/commit \
  -H "Content-Type: application/json" \
  -d "{
    \"transaction_id\": \"$TXN_ID\",
    \"provider_id\": \"DR001\",
    \"acknowledged\": true
  }")

echo "Commit Response:"
echo $COMMIT_RESPONSE | jq .

# Step 4: Get audit trail
AUDIT_RESPONSE=$(curl -s http://localhost:8119/api/v1/transaction/$TXN_ID/audit)

echo "Audit Trail:"
echo $AUDIT_RESPONSE | jq .
```

### 5.2 Success Criteria Checklist

```markdown
## Technical Success
- [ ] KB-19 `/api/v1/transaction/validate` returns block decisions
- [ ] KB-19 `/api/v1/transaction/commit` creates FHIR resources
- [ ] KB-19 `/api/v1/transaction/override` binds identity
- [ ] Med-Advisor `/api/v1/advisor/risk-profile` returns ONLY risk scores
- [ ] Med-Advisor response has NO HardBlocks field
- [ ] Med-Advisor response has NO Disposition field
- [ ] Med-Advisor response has NO GovernanceEvents field
- [ ] All governance events created by KB-19
- [ ] All audit trails finalized by KB-18

## Regulatory Success
- [ ] Override decision traceable to specific provider
- [ ] Block decision traceable to KB-19 (not Med-Advisor)
- [ ] Audit trail shows: "KB-19 blocked, Dr. X overrode, KB-18 recorded"
- [ ] Court-defensible: Single authority (KB-19) for commit decisions

## Architecture Compliance
- [ ] Med-Advisor = Risk Computer only
- [ ] KB-19 = Transaction Authority
- [ ] KB-18 = Governance & Identity
- [ ] Vaidshala (8090) = Unchanged
- [ ] ICU Intelligence (8120) = Unchanged
```

---

## 6. Timeline & Milestones (MOVE Approach)

```
Week 1: Move Code to KB-19
├── Day 1: Phase 1 - Move Types (HardBlock, GovernanceEvent, etc.)
├── Day 1: Phase 1 - Move DDI Rules (knownDDIHardStops, checkDDIPair)
├── Day 2: Phase 1 - Move Lab Rules (knownLabDrugHardStops, labRuleApplies)
├── Day 2: Phase 1 - Move ConflictDetector (entire conflicts.go)
├── Day 3: Phase 2 - Move processDDIHardBlocks → evaluateDDIBlocks
├── Day 3: Phase 2 - Move processLabHardBlocks → evaluateLabBlocks
├── Day 4: Phase 2 - Move governance functions (buildAuditTrail, etc.)
└── Day 4: Phase 2 - Create Transaction Manager (only new code)

Week 2: Cleanup & Test
├── Day 1: Phase 3 - Delete moved code from Med-Advisor engine.go
├── Day 1: Phase 3 - Delete conflicts.go from Med-Advisor
├── Day 2: Phase 3 - Add /risk-profile endpoint to Med-Advisor
├── Day 2: Phase 3 - Wire transaction routes in KB-19
├── Day 3: Phase 4 - End-to-end flow testing
├── Day 4: Phase 4 - Override workflow testing
├── Day 5: Phase 4 - Audit trail verification + documentation
```

**Key Difference from Previous Plan:**
- **Old:** Write new code from scratch (~52h)
- **New:** Move existing code + minimal new glue code (~40h)
- **Savings:** ~12h (no need to rewrite battle-tested logic)

---

## 7. Rollout Strategy

### 7.1 Feature Flag Approach

```go
// config.go
type Config struct {
    UseV3Architecture bool `env:"USE_V3_ARCHITECTURE" default:"false"`
}

// In KB-19
if config.UseV3Architecture {
    // Call Med-Advisor /risk-profile (V3)
    // Make block decisions in KB-19
} else {
    // Call Med-Advisor /calculate (legacy)
    // Pass through HardBlocks from Med-Advisor
}
```

### 7.2 Rollout Phases

| Phase | Flag Value | Behavior |
|-------|------------|----------|
| **Phase A** | `false` | Legacy mode - Med-Advisor makes decisions |
| **Phase B** | `true` (10% traffic) | V3 mode for 10% of requests |
| **Phase C** | `true` (50% traffic) | V3 mode for 50% of requests |
| **Phase D** | `true` (100% traffic) | Full V3 mode |
| **Phase E** | Remove flag | Legacy code removed |

---

## 8. Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking existing EHR integrations | Feature flag + gradual rollout |
| KB-19 single point of failure | Circuit breaker + fallback to legacy |
| Performance degradation | Caching + parallel KB calls |
| Override abuse | KB-18 identity binding + audit alerts |
| Data migration | No migration needed - new transactions only |

---

## 9. Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Lead Developer | | | |
| Architect | | | |
| QA Lead | | | |
| CTO | | | |
| CMO | | | |

---

## 10. Appendix: Quick Reference

### New KB-19 Endpoints (V3)

```
POST /api/v1/transaction/validate   - Validate medication
POST /api/v1/transaction/commit     - Commit with audit
POST /api/v1/transaction/override   - Override with identity
GET  /api/v1/transaction/:id/audit  - Get audit trail
```

### New Med-Advisor Endpoint (V3)

```
POST /api/v1/advisor/risk-profile   - Get risk scores ONLY
```

### Removed from Med-Advisor Response

```diff
- HardBlocks       []HardBlock
- Disposition      DispositionCode
- GovernanceEvents []GovernanceEvent
- AuditTrail       AuditTrailSummary
- GeneratedTasks   []GeneratedTask
```

### Key Files

| Component | File |
|-----------|------|
| KB-19 Transaction API | `kb-19-protocol-orchestrator/internal/api/transaction_handlers.go` |
| KB-19 Transaction Manager | `kb-19-protocol-orchestrator/internal/transaction/manager.go` |
| KB-19 Validator | `kb-19-protocol-orchestrator/internal/transaction/validator.go` |
| Med-Advisor Risk Profile | `medication-advisor-engine/advisor/risk_profile.go` |
| Med-Advisor Engine | `medication-advisor-engine/advisor/engine.go` |
