# KB-14 Care Navigator: Tier-7 Governance Certification Checklist

**Service**: KB-14 Care Navigator & Tasking Engine
**Version**: 1.0.0
**Certification Date**: _________________
**Certified By**: _________________
**Review Cycle**: Quarterly

---

## Executive Summary

This certification checklist validates that KB-14 Care Navigator meets **Tier-7 Governance** requirements for clinical task orchestration. Tier-7 compliance ensures:

- ✅ **Court-Proof Traceability**: Every action has an immutable audit trail
- ✅ **No Orphan Intelligence**: Every alert must result in a task OR documented disposition
- ✅ **Reason Code Enforcement**: Status changes require standardized reason codes
- ✅ **Escalation Accountability**: SLA breaches are tracked and escalated systematically
- ✅ **Compliance Scoring**: Real-time governance metrics and risk assessment

---

## Section 1: Immutable Audit Ledger

### 1.1 Hash Chain Integrity
| Requirement | Status | Evidence |
|-------------|--------|----------|
| Each audit record contains SHA-256 hash of content | ☐ | `task_audit_log.record_hash` |
| Each record links to previous record's hash | ☐ | `task_audit_log.previous_hash` |
| Genesis record has empty previous_hash | ☐ | First record verification |
| Hash chain verification endpoint exists | ☐ | `GET /governance/audit/verify/{taskId}` |
| Database triggers prevent UPDATE on audit table | ☐ | `audit_log_immutable_update` trigger |
| Database triggers prevent DELETE on audit table | ☐ | `audit_log_immutable_delete` trigger |

### 1.2 Audit Event Completeness
| Event Type | Captured | Event Category |
|------------|----------|----------------|
| CREATED | ☐ | LIFECYCLE |
| ASSIGNED | ☐ | ASSIGNMENT |
| REASSIGNED | ☐ | ASSIGNMENT |
| STARTED | ☐ | LIFECYCLE |
| COMPLETED | ☐ | LIFECYCLE |
| VERIFIED | ☐ | LIFECYCLE |
| DECLINED | ☐ | LIFECYCLE |
| CANCELLED | ☐ | LIFECYCLE |
| ESCALATED | ☐ | ESCALATION |
| ESCALATION_ACKNOWLEDGED | ☐ | ESCALATION |
| ESCALATION_RESOLVED | ☐ | ESCALATION |
| PRIORITY_CHANGED | ☐ | MODIFICATION |
| DUE_DATE_CHANGED | ☐ | MODIFICATION |
| NOTE_ADDED | ☐ | MODIFICATION |
| SLA_WARNING | ☐ | GOVERNANCE |
| SLA_BREACH | ☐ | GOVERNANCE |

### 1.3 Actor Attribution
| Requirement | Status | Evidence |
|-------------|--------|----------|
| Actor ID captured for all events | ☐ | `task_audit_log.actor_id` |
| Actor type classification (USER/SYSTEM/WORKER/INTEGRATION) | ☐ | `task_audit_log.actor_type` |
| Actor name recorded | ☐ | `task_audit_log.actor_name` |
| Actor role recorded | ☐ | `task_audit_log.actor_role` |
| IP address captured for user actions | ☐ | `task_audit_log.ip_address` |
| Session ID tracked | ☐ | `task_audit_log.session_id` |

---

## Section 2: No Orphan Intelligence Rule

### 2.1 Intelligence Tracking
| Requirement | Status | Evidence |
|-------------|--------|----------|
| All KB-3 temporal alerts tracked | ☐ | `intelligence_tracking.source_service = 'KB3_TEMPORAL'` |
| All KB-9 care gaps tracked | ☐ | `intelligence_tracking.source_service = 'KB9_CARE_GAPS'` |
| All KB-12 activities tracked | ☐ | `intelligence_tracking.source_service = 'KB12_ORDER_SETS'` |
| Source ID preserved (unique constraint) | ☐ | `UNIQUE(source_service, source_id)` |
| Full intelligence snapshot stored | ☐ | `intelligence_tracking.intelligence_snapshot` JSONB |

### 2.2 Intelligence Disposition
| Requirement | Status | Evidence |
|-------------|--------|----------|
| Intelligence must become TASK_CREATED OR DECLINED | ☐ | Status constraint validation |
| Declined intelligence requires disposition_code | ☐ | `intelligence_tracking.disposition_code` |
| Declined intelligence requires disposition_reason | ☐ | `intelligence_tracking.disposition_reason` |
| Disposition recorded with actor ID | ☐ | `intelligence_tracking.disposition_by` |
| Disposition timestamp captured | ☐ | `intelligence_tracking.disposition_at` |
| Orphaned intelligence detection (timeout) | ☐ | `FindOrphanedIntelligence()` method |
| Governance event for orphaned intelligence | ☐ | `INTELLIGENCE_GAP` event type |

### 2.3 Intelligence Accountability Dashboard
| Metric | Available | Endpoint |
|--------|-----------|----------|
| Intelligence by source service | ☐ | `GET /governance/intelligence/accountability` |
| Tasks created count | ☐ | `accountability.tasks_created` |
| Dispositioned count | ☐ | `accountability.dispositioned` |
| Pending count | ☐ | `accountability.pending` |

---

## Section 3: Reason Code Enforcement

### 3.1 Reason Code Categories
| Category | Purpose | Codes Loaded |
|----------|---------|--------------|
| ACCEPTANCE | Task creation justification | ☐ |
| REJECTION | Decline/rejection reasons | ☐ |
| ESCALATION | Escalation justification | ☐ |
| COMPLETION | Completion documentation | ☐ |
| CANCELLATION | Cancellation reasons | ☐ |

### 3.2 Pre-loaded Reason Codes
| Code | Category | Requires Justification | Requires Supervisor |
|------|----------|------------------------|---------------------|
| CLINICAL_NECESSITY | ACCEPTANCE | No | No |
| PROTOCOL_REQUIREMENT | ACCEPTANCE | No | No |
| PATIENT_REQUEST | ACCEPTANCE | No | No |
| PREVENTIVE_CARE | ACCEPTANCE | No | No |
| REGULATORY_COMPLIANCE | ACCEPTANCE | No | No |
| NOT_CLINICALLY_RELEVANT | REJECTION | **Yes** | No |
| DUPLICATE_INTELLIGENCE | REJECTION | No | No |
| PATIENT_DECLINED | REJECTION | **Yes** | No |
| CONTRAINDICATED | REJECTION | **Yes** | **Yes** |
| ALTERNATE_PATH | REJECTION | **Yes** | No |
| OUTSIDE_SCOPE | REJECTION | **Yes** | No |
| SLA_BREACH | ESCALATION | No | No |
| CLINICAL_URGENCY | ESCALATION | No | No |
| RESOURCE_UNAVAILABLE | ESCALATION | **Yes** | No |
| EXPERTISE_REQUIRED | ESCALATION | **Yes** | No |
| MANUAL_ESCALATION | ESCALATION | **Yes** | No |
| RESOLVED | COMPLETION | No | No |
| RESOLVED_WITH_FOLLOWUP | COMPLETION | No | No |
| PARTIALLY_RESOLVED | COMPLETION | **Yes** | No |
| REFERRED | COMPLETION | No | No |
| NO_LONGER_APPLICABLE | CANCELLATION | **Yes** | No |
| PATIENT_TRANSFERRED | CANCELLATION | No | No |
| PATIENT_DECEASED | CANCELLATION | No | No |
| SUPERSEDED | CANCELLATION | No | No |
| ERROR_CREATED | CANCELLATION | **Yes** | **Yes** |

### 3.3 Reason Code Validation
| Requirement | Status | Evidence |
|-------------|--------|----------|
| API validates reason codes before status change | ☐ | `ValidateReasonCode()` method |
| Justification required when `requires_justification = true` | ☐ | `DeclineWithAudit()` validation |
| Supervisor approval flow when `requires_supervisor_approval = true` | ☐ | Approval workflow |
| Reason code validation endpoint | ☐ | `GET /governance/reason-codes/validate/{code}` |

---

## Section 4: Escalation Accountability

### 4.1 Escalation Levels
| Level | Standard SLA % | Critical SLA % | Action |
|-------|---------------|----------------|--------|
| 1 - WARNING | 50% | 25% | Notify assignee |
| 2 - URGENT | 75% | 50% | Escalate to supervisor |
| 3 - CRITICAL | 100% | 75% | Deadline reached |
| 4 - EXECUTIVE | 125% | 100% | Executive notification |

### 4.2 Escalation Processing
| Requirement | Status | Evidence |
|-------------|--------|----------|
| Automatic escalation check (every 60s) | ☐ | `escalation_worker.go` |
| Escalation creates audit event | ☐ | `ESCALATED` audit event |
| Escalation creates governance event | ☐ | `ESCALATION_ALERT` governance event |
| Escalation notification sent | ☐ | Notification service integration |
| Escalation acknowledgement tracking | ☐ | `escalation.acknowledged_at` |
| Escalation resolution tracking | ☐ | `escalation.resolved_at` |

### 4.3 SLA Governance Events
| Requirement | Status | Evidence |
|-------------|--------|----------|
| SLA_WARNING governance event at warning threshold | ☐ | `PublishSLAWarning()` |
| SLA_BREACH governance event at breach threshold | ☐ | `PublishSLABreach()` |
| Risk score calculation for SLA events | ☐ | `governance_events.risk_score` |
| Action deadline set for SLA breaches | ☐ | `governance_events.action_deadline` |

---

## Section 5: Governance Events & Compliance

### 5.1 Governance Event Types
| Event Type | Severity | Captured |
|------------|----------|----------|
| COMPLIANCE_CHECK | INFO/WARNING | ☐ |
| AUDIT_REQUIRED | WARNING | ☐ |
| POLICY_VIOLATION | CRITICAL | ☐ |
| SLA_BREACH | WARNING/CRITICAL/ALERT | ☐ |
| ESCALATION_ALERT | WARNING/CRITICAL | ☐ |
| INTELLIGENCE_GAP | WARNING | ☐ |

### 5.2 Governance Event Resolution
| Requirement | Status | Evidence |
|-------------|--------|----------|
| Events requiring action flagged | ☐ | `governance_events.requires_action` |
| Action deadline tracked | ☐ | `governance_events.action_deadline` |
| Resolution workflow | ☐ | `POST /governance/events/{id}/resolve` |
| Resolution notes captured | ☐ | `governance_events.resolution_notes` |
| Resolver attribution | ☐ | `governance_events.resolved_by` |

### 5.3 Compliance Dashboard
| Metric | Available | Endpoint |
|--------|-----------|----------|
| Events by type and severity | ☐ | `GET /governance/dashboard` |
| Resolved vs pending events | ☐ | Dashboard data |
| Average compliance score | ☐ | `governance_events.compliance_score` |
| Average risk score | ☐ | `governance_events.risk_score` |

---

## Section 6: Compliance Scoring

### 6.1 Compliance Score Components
| Component | Weight | Calculation |
|-----------|--------|-------------|
| SLA Compliance | 40% | 100% - (SLA breaches / total events) |
| Escalation Resolution Rate | 30% | Resolved events / total events |
| Intelligence Gap Score | 30% | 100% - (pending / total intelligence) |

### 6.2 Risk Level Classification
| Risk Level | Score Range | Action Required |
|------------|-------------|-----------------|
| LOW | 85-100% | Routine monitoring |
| MEDIUM | 70-84% | Weekly review |
| HIGH | 0-69% | Immediate action |

### 6.3 Compliance Endpoints
| Endpoint | Available | Response |
|----------|-----------|----------|
| `GET /governance/compliance-score` | ☐ | Overall score with breakdown |
| `GET /governance/dashboard?days=N` | ☐ | Historical dashboard data |

---

## Section 7: API Governance Endpoints

### 7.1 Audit Trail Endpoints
| Endpoint | Method | Status |
|----------|--------|--------|
| `/governance/audit/task/{id}` | GET | ☐ |
| `/governance/audit/patient/{id}` | GET | ☐ |
| `/governance/audit/actor/{id}` | GET | ☐ |
| `/governance/audit/search` | GET | ☐ |
| `/governance/audit/summary/{taskId}` | GET | ☐ |
| `/governance/audit/verify/{taskId}` | GET | ☐ |

### 7.2 Governance Events Endpoints
| Endpoint | Method | Status |
|----------|--------|--------|
| `/governance/events` | GET | ☐ |
| `/governance/events/{id}` | GET | ☐ |
| `/governance/events/unresolved` | GET | ☐ |
| `/governance/events/requiring-action` | GET | ☐ |
| `/governance/events/{id}/resolve` | POST | ☐ |

### 7.3 Reason Codes Endpoints
| Endpoint | Method | Status |
|----------|--------|--------|
| `/governance/reason-codes` | GET | ☐ |
| `/governance/reason-codes/{category}` | GET | ☐ |
| `/governance/reason-codes/validate/{code}` | GET | ☐ |

### 7.4 Intelligence Tracking Endpoints
| Endpoint | Method | Status |
|----------|--------|--------|
| `/governance/intelligence/accountability` | GET | ☐ |
| `/governance/intelligence/{id}/disposition` | POST | ☐ |

### 7.5 Dashboard & Compliance Endpoints
| Endpoint | Method | Status |
|----------|--------|--------|
| `/governance/dashboard` | GET | ☐ |
| `/governance/compliance-score` | GET | ☐ |

---

## Section 8: Database Schema Compliance

### 8.1 Required Tables
| Table | Purpose | Status |
|-------|---------|--------|
| `task_audit_log` | Immutable audit ledger | ☐ |
| `governance_events` | Governance event tracking | ☐ |
| `reason_codes` | Standardized reason codes | ☐ |
| `intelligence_tracking` | Intelligence accountability | ☐ |

### 8.2 Required Indexes
| Index | Purpose | Status |
|-------|---------|--------|
| `idx_audit_log_task_id` | Task-based audit lookup | ☐ |
| `idx_audit_log_patient_id` | Patient-based audit lookup | ☐ |
| `idx_audit_log_event_timestamp` | Time-based audit queries | ☐ |
| `idx_audit_log_hash_chain` | Hash chain verification | ☐ |
| `idx_governance_events_unresolved` | Unresolved event queries | ☐ |
| `idx_intelligence_unprocessed` | Orphan detection | ☐ |

### 8.3 Required Triggers
| Trigger | Purpose | Status |
|---------|---------|--------|
| `audit_log_immutable_update` | Prevent UPDATE on audit log | ☐ |
| `audit_log_immutable_delete` | Prevent DELETE on audit log | ☐ |

### 8.4 Required Views
| View | Purpose | Status |
|------|---------|--------|
| `v_task_audit_summary` | Audit trail summary per task | ☐ |
| `v_governance_dashboard` | 30-day governance dashboard | ☐ |
| `v_intelligence_accountability` | 7-day intelligence accountability | ☐ |

---

## Section 9: Integration Testing

### 9.1 Audit Trail Tests
| Test Case | Status |
|-----------|--------|
| Create task → audit event created with hash | ☐ |
| Assign task → audit event with previous/new assignee | ☐ |
| Complete task → audit event with reason code | ☐ |
| Decline task → audit event requires reason code | ☐ |
| Cancel task → audit event with cancellation reason | ☐ |
| Escalate task → audit event with escalation level | ☐ |
| Hash chain verification passes for valid chain | ☐ |
| Hash chain verification fails for tampered record | ☐ |

### 9.2 Intelligence Tracking Tests
| Test Case | Status |
|-----------|--------|
| KB-3 alert creates intelligence tracking record | ☐ |
| KB-9 care gap creates intelligence tracking record | ☐ |
| KB-12 activity creates intelligence tracking record | ☐ |
| Intelligence linked to task on task creation | ☐ |
| Intelligence disposition requires valid reason code | ☐ |
| Orphaned intelligence detected after timeout | ☐ |
| Governance event created for orphaned intelligence | ☐ |

### 9.3 Governance Event Tests
| Test Case | Status |
|-----------|--------|
| SLA warning creates governance event | ☐ |
| SLA breach creates governance event with deadline | ☐ |
| Policy violation creates critical governance event | ☐ |
| Governance event resolution workflow | ☐ |
| Unresolved events query returns pending events | ☐ |
| Events requiring action flagged correctly | ☐ |

### 9.4 Compliance Scoring Tests
| Test Case | Status |
|-----------|--------|
| Compliance score calculates correctly | ☐ |
| SLA component weighted at 40% | ☐ |
| Escalation component weighted at 30% | ☐ |
| Intelligence component weighted at 30% | ☐ |
| Risk level classification correct | ☐ |

---

## Section 10: Certification Sign-off

### Pre-Certification Review
- [ ] All database migrations applied successfully
- [ ] All indexes created and verified
- [ ] All triggers active and tested
- [ ] All API endpoints responding correctly
- [ ] Integration tests passing
- [ ] Performance benchmarks met (< 100ms audit write)
- [ ] Hash chain integrity verified
- [ ] Reason codes loaded and active

### Certification Approval

| Role | Name | Signature | Date |
|------|------|-----------|------|
| Technical Lead | | | |
| Clinical Informatics | | | |
| Compliance Officer | | | |
| Security Officer | | | |

### Certification Statement

> I certify that KB-14 Care Navigator meets all Tier-7 Governance requirements as specified in this checklist. The system maintains court-proof audit trails, enforces no-orphan intelligence rules, requires reason code documentation for status changes, and provides comprehensive compliance scoring.

---

## Appendix A: File Inventory

| File | Purpose |
|------|---------|
| `migrations/004_create_audit_log.sql` | Audit ledger schema |
| `internal/models/governance.go` | Governance domain models |
| `internal/database/audit_repository.go` | Audit repository layer |
| `internal/services/governance_service.go` | Governance business logic |
| `internal/api/governance_handlers.go` | Governance API endpoints |
| `internal/models/task.go` | Task model with governance fields |
| `internal/services/task_service.go` | Task service with audit integration |

---

## Appendix B: Glossary

| Term | Definition |
|------|------------|
| **Immutable Audit Ledger** | Append-only audit log with cryptographic hash chain preventing tampering |
| **Hash Chain** | Each record links to previous via SHA-256 hash, creating tamper-evident sequence |
| **Intelligence** | Clinical alerts/gaps from KB-3, KB-9, KB-12 requiring action |
| **Orphan Intelligence** | Intelligence that hasn't been converted to task or dispositioned within timeout |
| **Reason Code** | Standardized code documenting justification for status changes |
| **Governance Event** | High-level compliance event requiring tracking and potential action |
| **Compliance Score** | Weighted aggregate score indicating overall governance health |
| **Tier-7** | Highest governance tier requiring full audit, accountability, and compliance |

---

**Document Version**: 1.0.0
**Last Updated**: 2025-12-27
**Next Review**: 2026-03-27
