# KB1 Phase 2: Canonical Fact Store Governance - COMPLETE

**Date**: 2026-01-23
**Status**: ✅ COMPLETE
**Reference**: KB1 Data Source Injection Implementation Plan - Phase 2

## Summary

Phase 2 implements governance workflows for the Canonical Fact Store (Shared DB). KB-0 acts as the governance platform that watches for new DRAFT facts and processes them through policy evaluation, automatic approval, or assignment for human review.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    ANGULAR UI (Pharmacist Dashboard)                │
│                         ↓         ↑                                 │
│                   REST API v2     │                                 │
└───────────────────────┬───────────┴─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────────┐
│                      KB-0 GOVERNANCE PLATFORM                       │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    Governance Executor                        │   │
│  │  ┌──────────────┐  ┌───────────────┐  ┌─────────────────┐   │   │
│  │  │ Queue Watcher │  │ Policy Engine │  │  Fact Store    │   │   │
│  │  │  (Polling)    │→│  (Decisions)  │→│  (Persistence) │   │   │
│  │  └──────────────┘  └───────────────┘  └─────────────────┘   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                              │                                       │
│  ┌──────────────────────────▼───────────────────────────────────┐   │
│  │                   internal/policy/                            │   │
│  │  ┌────────────┐  ┌───────────┐  ┌───────────┐  ┌──────────┐  │   │
│  │  │ activation │  │ conflict  │  │ override  │  │  engine  │  │   │
│  │  │  policy    │  │  policy   │  │  policy   │  │ (coord.) │  │   │
│  │  └────────────┘  └───────────┘  └───────────┘  └──────────┘  │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────────┐
│                CANONICAL FACT STORE (Shared DB)                     │
│  ┌──────────────────┐  ┌────────────────────┐  ┌────────────────┐  │
│  │  clinical_facts  │  │ v_governance_queue │  │ governance_    │  │
│  │  (+ governance   │  │    (poll view)     │  │ audit_log      │  │
│  │   columns)       │  │                    │  │                │  │
│  └──────────────────┘  └────────────────────┘  └────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

## Files Created/Modified

### Database Migration
- `shared/migrations/007_phase2_governance.sql` - **NEW**
  - Governance columns on `clinical_facts` table
  - `v_governance_queue` view for KB-0 polling
  - `governance_decisions` table
  - `governance_audit_log` table (21 CFR Part 11 compliant)
  - `authority_priorities` table (ONC > FDA > OHDSI hierarchy)
  - Helper functions: `calculate_review_priority()`, `calculate_review_due_date()`, `log_governance_event()`
  - Triggers for auto-assignment and audit logging

### KB-0 Policy Package (`internal/policy/`)
- `types.go` - **NEW**: Policy types for clinical facts
  - `ClinicalFact` struct (mirrors clinical_facts table)
  - `PolicyConfig` with thresholds
  - Decision types: `ActivationDecision`, `ConflictDecision`, `OverrideDecision`, `StabilityDecision`
  - Queue types: `QueueItem`, `ReviewRequest`, `AuditEvent`

- `activation.go` - **NEW**: Activation policy
  - Confidence-based routing: ≥0.95 = AUTO_APPROVE, 0.65-0.85 = REQUIRE_REVIEW, <0.65 = REJECT
  - Fact type overrides (SAFETY_SIGNAL always requires review)
  - Source type overrides (LLM extractions require review)
  - Conflict detection integration

- `conflict.go` - **NEW**: Conflict resolution policy
  - Authority priority resolution: ONC (1) > FDA (2) > OHDSI (21)
  - Recency tiebreaker for same authority
  - Multi-way conflict resolution
  - Manual review fallback for unresolvable conflicts

- `override.go` - **NEW**: Override and stability policies
  - Override types: EMERGENCY (4h), INSTITUTIONAL, CLINICAL_JUDGMENT (24h)
  - Role requirements and validation
  - Stability policy for supersession (min active hours per fact type)

- `engine.go` - **NEW**: Policy evaluation engine
  - `EvaluateFact()` - main entry point for governance decisions
  - `BatchEvaluateFacts()` - batch processing
  - Statistics and metrics computation

### KB-0 Database Layer (`internal/database/`)
- `fact_store.go` - **NEW**: Fact store for Canonical Fact Store
  - Queue operations: `GetGovernanceQueue()`, `GetQueueByPriority()`, `GetQueueByReviewer()`
  - Fact operations: `GetFact()`, `GetFactsByDrug()`, `ActivateFact()`, `SupersedeFact()`
  - Governance operations: `UpdateGovernanceDecision()`, `AssignReviewer()`, `MarkConflict()`
  - Audit operations: `LogGovernanceEvent()`, `RecordDecision()`
  - Metrics: `GetFactMetrics()`

### KB-0 Governance Layer (`internal/governance/`)
- `executor.go` - **NEW**: Governance workflow executor
  - Background watcher with configurable poll interval
  - Batch processing of pending facts
  - `ProcessFact()` - single fact governance workflow
  - `ApproveReview()`, `RejectReview()`, `EscalateReview()` - manual review actions
  - Automatic conflict resolution and supersession

### KB-0 API Layer (`internal/api/`)
- `fact_handlers.go` - **NEW**: REST API for Angular UI
  - Queue endpoints: `GET /queue`, `GET /queue/priority/{priority}`, `GET /queue/reviewer/{id}`
  - Fact endpoints: `GET /facts/{id}`, `GET /facts/{id}/conflicts`, `GET /facts/{id}/history`
  - Review actions: `POST /facts/{id}/approve`, `POST /facts/{id}/reject`, `POST /facts/{id}/escalate`
  - Dashboard: `GET /metrics`, `GET /dashboard`
  - Executor control: `POST /executor/start`, `POST /executor/stop`, `GET /executor/status`

### Go Module
- `go.mod` - **MODIFIED**: Added dependencies
  - `github.com/google/uuid v1.6.0`
  - `github.com/lib/pq v1.10.9`

## Policy Decision Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                         NEW FACT (DRAFT)                            │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 1. ACTIVATION POLICY                                                │
│    ├─ Is fact type SAFETY_SIGNAL? → PENDING_REVIEW (CRITICAL)      │
│    ├─ Is source LLM with confidence < 0.95? → PENDING_REVIEW       │
│    ├─ Has conflicts? → PENDING_REVIEW                              │
│    ├─ Confidence ≥ 0.95? → AUTO_APPROVE                            │
│    ├─ Confidence ≥ 0.65? → PENDING_REVIEW                          │
│    └─ Confidence < 0.65? → REJECT                                  │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 2. CONFLICT POLICY                                                  │
│    ├─ Same drug + type + different source?                         │
│    │   ├─ Different authority priority → Winner by priority        │
│    │   ├─ Same priority → Winner by recency                        │
│    │   └─ Cannot resolve → MANUAL review required                  │
│    └─ Loser facts → Mark SUPERSEDED                                │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│ 3. EXECUTE DECISION                                                 │
│    ├─ AUTO_APPROVE → Activate fact, supersede conflicts            │
│    ├─ PENDING_REVIEW → Assign priority + SLA, wait for human       │
│    └─ REJECT → Mark rejected, log reason                           │
└─────────────────────────────────────────────────────────────────────┘
```

## API Endpoints (v2)

### Queue Operations
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v2/governance/queue` | Get pending review queue |
| GET | `/api/v2/governance/queue/priority/{priority}` | Get queue by priority |
| GET | `/api/v2/governance/queue/reviewer/{id}` | Get reviewer's assigned queue |

### Fact Operations
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v2/governance/facts/{id}` | Get fact details |
| GET | `/api/v2/governance/facts/{id}/conflicts` | Get conflicting facts |
| GET | `/api/v2/governance/facts/{id}/history` | Get audit history |

### Review Actions
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v2/governance/facts/{id}/approve` | Approve fact |
| POST | `/api/v2/governance/facts/{id}/reject` | Reject fact |
| POST | `/api/v2/governance/facts/{id}/escalate` | Escalate to CMO |
| POST | `/api/v2/governance/facts/{id}/assign` | Assign reviewer |

### Dashboard
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v2/governance/metrics` | Get governance metrics |
| GET | `/api/v2/governance/dashboard` | Get full dashboard data |

### Executor Control
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v2/governance/executor/start` | Start background watcher |
| POST | `/api/v2/governance/executor/stop` | Stop background watcher |
| GET | `/api/v2/governance/executor/status` | Get executor status |
| POST | `/api/v2/governance/executor/process/{id}` | Manually process single fact |

## Configuration

```go
// PolicyConfig thresholds
AutoApproveThreshold:   0.95  // ≥ 0.95 = auto-approve
RequireReviewThreshold: 0.65  // ≥ 0.65 = require review
RejectThreshold:        0.65  // < 0.65 = reject

// SLA targets by priority
CRITICAL: 24 hours
HIGH:     48 hours
STANDARD: 7 days
LOW:      14 days

// Authority priorities (lower = higher priority)
ONC:      1   // Constitutional DDI rules
FDA:      2   // Drug safety authority
USP:      3   // Pharmacopeia standards
NICE:     4   // UK guidelines
TGA:      5   // Australian regulator
CDSCO:    6   // Indian regulator
EMA:      7   // European regulator
DRUGBANK: 10  // Drug database
RXNORM:   11  // Terminology
OHDSI:    21  // Research-grade DDI
```

## Next Steps (Phase 3)

1. **Angular UI Components**
   - Review queue dashboard
   - Fact detail view with conflict visualization
   - Approval/rejection workflow forms
   - Metrics charts and SLA tracking

2. **Integration Testing**
   - End-to-end governance workflow tests
   - Conflict resolution scenario tests
   - SLA breach notification tests

3. **Production Hardening**
   - Connection pooling for database
   - Retry logic for transient failures
   - Metrics and monitoring (Prometheus)
   - Rate limiting for API endpoints

## Verification

**KB-0 governance workflow is live.**

The Phase 2 implementation provides:
- ✅ Policy logic inside KB-0 (no separate KB-18 service needed for MVP)
- ✅ Canonical Fact Store governance columns and views
- ✅ Confidence-based activation policy
- ✅ Authority priority conflict resolution (ONC > FDA > OHDSI)
- ✅ REST API for Angular UI
- ✅ Background executor for automatic processing
- ✅ 21 CFR Part 11 compliant audit logging
