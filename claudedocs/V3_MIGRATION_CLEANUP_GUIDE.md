# V3 Architecture Migration: Cleanup Guide

## Overview

This document lists the code that should be deleted from **Medication Advisor Engine** (8095) after V3 architecture is fully tested. The code has been MOVED (not copied) to **KB-19 Protocol Orchestrator** (8119).

**V3 Architecture Pattern:**
- Med-Advisor (8095) = Judge (calculates risks ONLY)
- KB-19 (8119) = Transaction Authority (makes block decisions, handles governance)

## Phase 3 Status: COMPLETE ✅

### Files Created in KB-19

| KB-19 File | Purpose | Status |
|------------|---------|--------|
| `internal/transaction/types.go` | Transaction types, HardBlock, GovernanceEvent | ✅ Created (Phase 1) |
| `internal/transaction/rules/ddi_rules.go` | DDI evaluation rules | ✅ Created (Phase 1) |
| `internal/transaction/rules/lab_rules.go` | Lab contraindication rules | ✅ Created (Phase 1) |
| `internal/transaction/conflict_detector.go` | Clinical snapshot conflict detection | ✅ Created (Phase 1) |
| `pkg/contracts/transaction_contracts.go` | API contracts for transactions | ✅ Created (Phase 1) |
| `internal/transaction/validator.go` | Transaction validation logic | ✅ Created (Phase 2) |
| `internal/transaction/committer.go` | Governance event generation, audit trail | ✅ Created (Phase 2) |
| `internal/transaction/manager.go` | Transaction state machine | ✅ Created (Phase 2) |
| `internal/api/transaction_handlers.go` | HTTP handlers for transaction routes | ✅ Created (Phase 3) |
| `internal/clients/medadvisor_client.go` | Client to call Med-Advisor /risk-profile | ✅ Created (Phase 3) |

### Files Modified in Med-Advisor

| Med-Advisor File | Change | Status |
|------------------|--------|--------|
| `advisor/risk_profile.go` | NEW: V3 RiskProfile endpoint (risk-only) | ✅ Created |
| `cmd/server/main.go` | Added /risk-profile route | ✅ Modified |

---

## Code to Delete from Med-Advisor (After V3 Testing)

### 1. Governance Functions (MOVED to `committer.go`)

**File:** `advisor/engine.go`

| Function | Lines | New Location |
|----------|-------|--------------|
| `generateGovernanceEvents()` | 566-606 | `kb-19/internal/transaction/committer.go` |
| `determineGovernanceEventType()` | 608-629 | `kb-19/internal/transaction/committer.go` |
| `computeEventHash()` | 631-668 | `kb-19/internal/transaction/committer.go` |
| `generateLabSafetyTasks()` | 672-744 | `kb-19/internal/transaction/committer.go` |
| `buildAuditTrail()` | 783-852 | `kb-19/internal/transaction/committer.go` |

### 2. Block Processing Functions (MOVED to `validator.go`)

**File:** `advisor/engine.go`

| Function | Lines | New Location |
|----------|-------|--------------|
| `processExcludedDrugs()` | 893-950 | `kb-19/internal/transaction/validator.go` |
| `determineDisposition()` | 747-780 | `kb-19/internal/transaction/validator.go` |
| `isHardBlockSeverity()` | 952-970 | `kb-19/internal/transaction/validator.go` |
| `mapToHardBlockType()` | 972-990 | `kb-19/internal/transaction/validator.go` |
| `isPregnancyCode()` | 992-1010 | `kb-19/internal/transaction/validator.go` |
| `isPregnancyRelatedBlock()` | 1012-1030 | `kb-19/internal/transaction/validator.go` |
| `extractFDACategory()` | 1032-1050 | `kb-19/internal/transaction/validator.go` |
| `generateAckText()` | 1052-1065 | `kb-19/internal/transaction/validator.go` |

### 3. DDI Rules (MOVED to `ddi_rules.go`)

**File:** `advisor/engine.go`

| Function | Lines | New Location |
|----------|-------|--------------|
| `processDDIHardBlocks()` | ~440-550 | `kb-19/internal/transaction/rules/ddi_rules.go` |
| DDI severity constants | Various | `kb-19/internal/transaction/rules/ddi_rules.go` |
| DDI rule definitions | Various | `kb-19/internal/transaction/rules/ddi_rules.go` |

### 4. Lab Rules (MOVED to `lab_rules.go`)

**File:** `advisor/engine.go`

| Function | Lines | New Location |
|----------|-------|--------------|
| `processLabHardBlocks()` | ~450-560 | `kb-19/internal/transaction/rules/lab_rules.go` |
| Lab threshold constants | Various | `kb-19/internal/transaction/rules/lab_rules.go` |
| Lab contraindication rules | Various | `kb-19/internal/transaction/rules/lab_rules.go` |

---

## What to Keep in Med-Advisor

### KEEP: Risk Calculation Logic

| Component | Reason |
|-----------|--------|
| `WorkflowOrchestrator` | KB orchestration for risk gathering |
| `ProposalScoringEngine` | Scoring candidates |
| `ConflictDetector` | Detecting conflicts (used by RiskProfile) |
| `RiskProfile()` method | V3 API endpoint |
| All KB client calls | Risk calculation requires KB data |

### KEEP: Response Types (Used by RiskProfile)

| Type | Reason |
|------|--------|
| `MedicationRisk` | Part of RiskProfileResponse |
| `DDIRisk` | Part of RiskProfileResponse |
| `LabRisk` | Part of RiskProfileResponse |
| `DoseRecommendation` | Part of RiskProfileResponse |

---

## Migration Verification Checklist

### Before Deleting Code from Med-Advisor

- [ ] KB-19 transaction routes are working (`/api/v1/transactions/*`)
- [ ] KB-19 can call Med-Advisor `/api/v1/risk-profile` successfully
- [ ] Integration test: Create → Validate → Commit workflow works end-to-end
- [ ] Governance events are being generated correctly by KB-19
- [ ] Audit trail hash chain is computed correctly
- [ ] Lab safety tasks (KB-14) are being generated
- [ ] Override workflow with identity binding works
- [ ] Performance testing passes (< 200ms p95 latency)

### After Deleting Code from Med-Advisor

- [ ] Med-Advisor still builds successfully
- [ ] RiskProfile endpoint still works
- [ ] Calculate endpoint continues to work (for backwards compatibility)
- [ ] No broken references in tests

---

## Backwards Compatibility

### Deprecation Timeline

1. **Phase 1 (Now):** Both `/calculate` and `/risk-profile` available
2. **Phase 2 (After Testing):** `/calculate` marked deprecated in docs
3. **Phase 3 (After 3 months):** `/calculate` returns deprecation warning header
4. **Phase 4 (After 6 months):** `/calculate` can be removed

### Header for Deprecated Endpoints

```go
c.Header("X-Deprecation-Warning", "This endpoint is deprecated. Use KB-19 transaction API instead.")
c.Header("X-Sunset-Date", "2026-07-01")
```

---

## V3 API Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           V3 ARCHITECTURE FLOW                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   1. Client Request                                                         │
│   ┌─────────────┐                                                           │
│   │  Provider   │ → POST /api/v1/transactions/create                       │
│   │  (EHR/UI)   │                                                           │
│   └──────┬──────┘                                                           │
│          │                                                                  │
│   2. KB-19 Transaction Authority (8119)                                    │
│   ┌──────▼──────────────────────────────────────────────────────────┐      │
│   │                                                                  │      │
│   │  a) Call Med-Advisor /risk-profile                              │      │
│   │     └──► GET risk calculations (DDI, Lab, Allergy)              │      │
│   │                                                                  │      │
│   │  b) Evaluate blocks using moved logic                           │      │
│   │     └──► EvaluateDDIBlocks()                                    │      │
│   │     └──► EvaluateLabBlocks()                                    │      │
│   │     └──► EvaluateExcludedDrugs()                                │      │
│   │                                                                  │      │
│   │  c) Determine disposition                                       │      │
│   │     └──► DetermineDisposition()                                 │      │
│   │                                                                  │      │
│   │  d) Return transaction with hard blocks                         │      │
│   │                                                                  │      │
│   └──────┬──────────────────────────────────────────────────────────┘      │
│          │                                                                  │
│   3. Override Flow (if hard blocks)                                        │
│   ┌──────▼──────┐                                                           │
│   │  Provider   │ → POST /api/v1/transactions/{id}/override               │
│   │  Acks Block │   (with identity binding via KB-18)                      │
│   └──────┬──────┘                                                           │
│          │                                                                  │
│   4. Commit Flow                                                           │
│   ┌──────▼──────────────────────────────────────────────────────────┐      │
│   │  KB-19 Commit                                                    │      │
│   │                                                                  │      │
│   │  a) Generate governance events                                  │      │
│   │     └──► GenerateGovernanceEvents() with hash chain            │      │
│   │                                                                  │      │
│   │  b) Generate KB-14 tasks                                        │      │
│   │     └──► GenerateLabSafetyTasks()                              │      │
│   │                                                                  │      │
│   │  c) Build audit trail                                           │      │
│   │     └──► BuildAuditTrail() Tier-7 compliant                    │      │
│   │                                                                  │      │
│   │  d) Return committed transaction                                │      │
│   │                                                                  │      │
│   └──────────────────────────────────────────────────────────────────┘      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Contact

For questions about this migration, contact the Vaidshala platform team.

**Migration Start Date:** 2026-01-15
**Expected Completion:** 2026-02-15 (Phase 4 integration testing)
