# KB-3 Guidelines Service - API Usage Guide

**Version**: 3.0.0
**Base URL**: `http://localhost:8083`
**Content-Type**: `application/json`

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Health & Status](#health--status)
4. [Protocol Management](#protocol-management)
5. [Pathway Operations](#pathway-operations)
6. [Patient Operations](#patient-operations)
7. [Scheduling Operations](#scheduling-operations)
8. [Temporal Operations](#temporal-operations)
9. [Alert Management](#alert-management)
10. [Batch Operations](#batch-operations)
11. [Governance](#governance)
12. [Common Patterns](#common-patterns)

---

## Overview

KB-3 provides **temporal logic and clinical pathway management** for healthcare applications:

```
┌─────────────────────────────────────────────────────────────────┐
│                    KB-3 GUIDELINES SERVICE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐    ┌──────────────────┐                   │
│  │  PATHWAY ENGINE  │    │ SCHEDULING ENGINE │                   │
│  │  (Acute Care)    │    │ (Chronic/Prevent) │                   │
│  │                  │    │                   │                   │
│  │  • Sepsis        │    │  • Diabetes       │                   │
│  │  • Stroke        │    │  • Heart Failure  │                   │
│  │  • STEMI         │    │  • Preventive     │                   │
│  │  • DKA           │    │  • Immunizations  │                   │
│  └────────┬─────────┘    └────────┬──────────┘                   │
│           │                       │                              │
│           └───────────┬───────────┘                              │
│                       ▼                                          │
│           ┌───────────────────────┐                              │
│           │   TEMPORAL OPERATORS  │                              │
│           │  (Allen's Interval    │                              │
│           │   Algebra - 13 ops)   │                              │
│           └───────────────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
```

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Protocol** | A clinical guideline template (e.g., SEPSIS-SEP1-2021) |
| **Pathway** | An active instance of a protocol for a specific patient |
| **Action** | A task within a pathway with deadlines and constraints |
| **Constraint Status** | PENDING, MET, APPROACHING, OVERDUE, MISSED |
| **Schedule** | Recurring care items (labs, appointments, screenings) |

---

## Quick Start

### 1. Check Service Health
```bash
curl http://localhost:8083/health
```

### 2. List Available Protocols
```bash
curl http://localhost:8083/v1/protocols
```

### 3. Start a Pathway for a Patient
```bash
curl -X POST http://localhost:8083/v1/pathways/start \
  -H "Content-Type: application/json" \
  -d '{
    "pathway_id": "SEPSIS-SEP1-2021",
    "patient_id": "patient-001",
    "context": {"severity": "moderate", "source": "ED"}
  }'
```

### 4. Check Pathway Status
```bash
curl http://localhost:8083/v1/pathways/{instance_id}
```

---

## Health & Status

### GET /health
Check service health status.

```bash
curl http://localhost:8083/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "kb-3-guidelines",
  "version": "3.0.0",
  "components": {
    "database": "connected",
    "neo4j": "connected",
    "redis": "connected"
  }
}
```

### GET /metrics
Get performance metrics.

```bash
curl http://localhost:8083/metrics
```

**Response:**
```json
{
  "uptime_seconds": 3600,
  "requests_total": 1523,
  "active_pathways": 45,
  "scheduled_items": 230,
  "memory_mb": 128
}
```

### GET /version
Get service version information.

```bash
curl http://localhost:8083/version
```

**Response:**
```json
{
  "version": "3.0.0",
  "build_date": "2025-12-20",
  "go_version": "1.22"
}
```

---

## Protocol Management

### GET /v1/protocols
List all available clinical protocols.

```bash
curl http://localhost:8083/v1/protocols
```

**Response:**
```json
{
  "protocols": {
    "acute": [
      {
        "protocol_id": "SEPSIS-SEP1-2021",
        "name": "Sepsis Bundle - CMS SEP-1",
        "type": "acute",
        "guideline_source": "Surviving Sepsis Campaign 2021",
        "stages": ["recognition", "3h_bundle", "6h_bundle"]
      }
    ],
    "chronic": [...],
    "preventive": [...]
  },
  "summary": {
    "total": 17,
    "acute_count": 6,
    "chronic_count": 6,
    "preventive_count": 5
  }
}
```

### GET /v1/protocols/acute
List acute care protocols only.

```bash
curl http://localhost:8083/v1/protocols/acute
```

**Available Acute Protocols:**
| Protocol ID | Name | Use Case |
|-------------|------|----------|
| SEPSIS-SEP1-2021 | Sepsis Bundle | Suspected sepsis in ED |
| STROKE-AHA-2019 | Acute Ischemic Stroke | Stroke alerts |
| STEMI-ACC-2013 | STEMI Protocol | Cardiac emergencies |
| DKA-ADA-2024 | Diabetic Ketoacidosis | DKA management |
| TRAUMA-ATLS-10 | Trauma Protocol | Trauma cases |
| PE-ESC-2019 | Pulmonary Embolism | PE management |

### GET /v1/protocols/chronic
List chronic disease management schedules.

```bash
curl http://localhost:8083/v1/protocols/chronic
```

**Available Chronic Schedules:**
| Schedule ID | Name | Key Monitoring |
|-------------|------|----------------|
| DIABETES-ADA-2024 | Diabetes Management | HbA1c q3-6mo, annual eye/foot |
| HF-ACCAHA-2022 | Heart Failure | 7-day post-DC, K+ monitoring |
| CKD-KDIGO-2024 | CKD Management | eGFR by stage |
| ANTICOAG-CHEST | Anticoagulation | INR monitoring |
| COPD-GOLD-2024 | COPD Management | CAT score, spirometry |
| HTN-ACCAHA-2017 | Hypertension | BP monitoring |

### GET /v1/protocols/preventive
List preventive care schedules.

```bash
curl http://localhost:8083/v1/protocols/preventive
```

### GET /v1/protocols/{type}/{id}
Get detailed protocol information.

```bash
curl http://localhost:8083/v1/protocols/acute/SEPSIS-SEP1-2021
```

**Response:**
```json
{
  "protocol_id": "SEPSIS-SEP1-2021",
  "name": "Sepsis Bundle - CMS SEP-1",
  "type": "acute",
  "guideline_source": "Surviving Sepsis Campaign 2021",
  "stages": [
    {
      "stage_id": "recognition",
      "name": "Sepsis Recognition",
      "order": 1,
      "actions": [
        {
          "action_id": "screen",
          "name": "Sepsis Screening",
          "type": "assessment",
          "deadline_offset": "0s",
          "required": true
        },
        {
          "action_id": "lactate_initial",
          "name": "Initial Lactate",
          "type": "lab",
          "deadline_offset": "30m",
          "required": true
        }
      ]
    },
    {
      "stage_id": "3h_bundle",
      "name": "3-Hour Bundle",
      "actions": [...]
    }
  ],
  "constraints": [
    {
      "action": "antibiotics",
      "deadline": "1h",
      "grace_period": "15m",
      "severity": "critical"
    }
  ]
}
```

### GET /v1/protocols/search?q={keyword}
Search protocols by keyword.

```bash
curl "http://localhost:8083/v1/protocols/search?q=diabetes"
```

### GET /v1/protocols/condition/{condition}
Get protocols for a specific condition.

```bash
curl http://localhost:8083/v1/protocols/condition/sepsis
```

---

## Pathway Operations

Pathways are **active instances** of protocols for specific patients.

### POST /v1/pathways/start
Start a new pathway for a patient.

```bash
curl -X POST http://localhost:8083/v1/pathways/start \
  -H "Content-Type: application/json" \
  -d '{
    "pathway_id": "SEPSIS-SEP1-2021",
    "patient_id": "patient-001",
    "context": {
      "severity": "severe",
      "source": "ED",
      "initial_lactate": 4.2,
      "hypotensive": true
    }
  }'
```

**Response:**
```json
{
  "instance_id": "abc123-def456-ghi789",
  "pathway_id": "SEPSIS-SEP1-2021",
  "patient_id": "patient-001",
  "current_stage": "recognition",
  "status": "active",
  "started_at": "2025-12-20T10:30:00Z",
  "actions": [
    {
      "action_id": "abc123-recognition-screen",
      "name": "Sepsis Screening",
      "type": "assessment",
      "status": "PENDING",
      "deadline": "2025-12-20T10:30:00Z",
      "grace_period": "15m"
    },
    {
      "action_id": "abc123-recognition-lactate_initial",
      "name": "Initial Lactate",
      "status": "PENDING",
      "deadline": "2025-12-20T11:00:00Z"
    }
  ],
  "audit_log": [
    {
      "timestamp": "2025-12-20T10:30:00Z",
      "action": "pathway_started",
      "details": {"protocol_id": "SEPSIS-SEP1-2021"}
    }
  ]
}
```

### GET /v1/pathways/{id}
Get full pathway status.

```bash
curl http://localhost:8083/v1/pathways/abc123-def456-ghi789
```

### GET /v1/pathways/{id}/pending
Get pending actions for a pathway.

```bash
curl http://localhost:8083/v1/pathways/abc123-def456-ghi789/pending
```

**Response:**
```json
{
  "instance_id": "abc123-def456-ghi789",
  "pending_actions": [
    {
      "action_id": "abc123-3h_bundle-antibiotics",
      "name": "Broad-spectrum Antibiotics",
      "status": "APPROACHING",
      "deadline": "2025-12-20T11:30:00Z",
      "time_remaining": "45m",
      "priority": "critical"
    }
  ],
  "count": 1
}
```

### GET /v1/pathways/{id}/overdue
Get overdue actions.

```bash
curl http://localhost:8083/v1/pathways/abc123-def456-ghi789/overdue
```

**Response:**
```json
{
  "instance_id": "abc123-def456-ghi789",
  "overdue_actions": [
    {
      "action_id": "abc123-recognition-screen",
      "name": "Sepsis Screening",
      "status": "OVERDUE",
      "deadline": "2025-12-20T10:30:00Z",
      "overdue_by": "15m",
      "in_grace_period": true
    }
  ],
  "count": 1
}
```

### GET /v1/pathways/{id}/constraints
Evaluate all constraints for a pathway.

```bash
curl http://localhost:8083/v1/pathways/abc123-def456-ghi789/constraints
```

**Response:**
```json
{
  "instance_id": "abc123-def456-ghi789",
  "evaluations": [
    {
      "action_id": "abc123-3h_bundle-antibiotics",
      "action_name": "Broad-spectrum Antibiotics",
      "status": "APPROACHING",
      "deadline": "2025-12-20T11:30:00Z",
      "time_remaining": "45m",
      "grace_period": "15m",
      "severity": "critical"
    },
    {
      "action_id": "abc123-recognition-screen",
      "action_name": "Sepsis Screening",
      "status": "MET",
      "completed_at": "2025-12-20T10:25:00Z"
    }
  ],
  "summary": {
    "total": 8,
    "met": 2,
    "pending": 4,
    "approaching": 1,
    "overdue": 1,
    "missed": 0
  }
}
```

### POST /v1/pathways/{id}/complete-action
Mark an action as completed.

```bash
curl -X POST http://localhost:8083/v1/pathways/abc123-def456-ghi789/complete-action \
  -H "Content-Type: application/json" \
  -d '{
    "action_id": "abc123-3h_bundle-antibiotics",
    "completed_by": "Dr. Smith",
    "notes": "Piperacillin/tazobactam administered IV"
  }'
```

**Response:**
```json
{
  "success": true,
  "action": {
    "action_id": "abc123-3h_bundle-antibiotics",
    "name": "Broad-spectrum Antibiotics",
    "status": "MET",
    "completed_at": "2025-12-20T11:15:00Z",
    "completed_by": "Dr. Smith",
    "on_time": true
  },
  "pathway_status": "active",
  "current_stage": "3h_bundle"
}
```

### POST /v1/pathways/{id}/advance
Manually advance to the next stage.

```bash
curl -X POST http://localhost:8083/v1/pathways/abc123-def456-ghi789/advance \
  -H "Content-Type: application/json" \
  -d '{
    "actor": "Dr. Smith",
    "reason": "All recognition stage actions completed"
  }'
```

### GET /v1/pathways/{id}/audit
Get full audit log for a pathway.

```bash
curl http://localhost:8083/v1/pathways/abc123-def456-ghi789/audit
```

**Response:**
```json
{
  "instance_id": "abc123-def456-ghi789",
  "audit_log": [
    {
      "entry_id": "audit-001",
      "timestamp": "2025-12-20T10:30:00Z",
      "action": "pathway_started",
      "actor": "system",
      "details": {"protocol_id": "SEPSIS-SEP1-2021"}
    },
    {
      "entry_id": "audit-002",
      "timestamp": "2025-12-20T10:45:00Z",
      "action": "action_completed",
      "actor": "Dr. Smith",
      "details": {"action_id": "abc123-recognition-screen"}
    },
    {
      "entry_id": "audit-003",
      "timestamp": "2025-12-20T11:00:00Z",
      "action": "stage_advanced",
      "actor": "system",
      "details": {"from": "recognition", "to": "3h_bundle"}
    }
  ]
}
```

### POST /v1/pathways/{id}/suspend
Suspend an active pathway.

```bash
curl -X POST http://localhost:8083/v1/pathways/abc123-def456-ghi789/suspend \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Patient transferred to another facility",
    "actor": "Dr. Johnson"
  }'
```

### POST /v1/pathways/{id}/resume
Resume a suspended pathway.

```bash
curl -X POST http://localhost:8083/v1/pathways/abc123-def456-ghi789/resume \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Patient returned, continuing care",
    "actor": "Dr. Johnson"
  }'
```

### POST /v1/pathways/{id}/cancel
Cancel a pathway.

```bash
curl -X POST http://localhost:8083/v1/pathways/abc123-def456-ghi789/cancel \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Diagnosis changed - not sepsis",
    "actor": "Dr. Smith"
  }'
```

---

## Patient Operations

### GET /v1/patients/{id}/pathways
Get all active pathways for a patient.

```bash
curl http://localhost:8083/v1/patients/patient-001/pathways
```

**Response:**
```json
{
  "patient_id": "patient-001",
  "pathways": [
    {
      "instance_id": "abc123",
      "pathway_id": "SEPSIS-SEP1-2021",
      "status": "active",
      "current_stage": "3h_bundle",
      "started_at": "2025-12-20T10:30:00Z"
    }
  ],
  "count": 1
}
```

### GET /v1/patients/{id}/schedule
Get scheduled items for a patient.

```bash
curl http://localhost:8083/v1/patients/patient-001/schedule
```

**Response:**
```json
{
  "patient_id": "patient-001",
  "items": [
    {
      "item_id": "sched-001",
      "type": "lab",
      "name": "HbA1c Test",
      "due_date": "2025-12-25T10:00:00Z",
      "priority": 2,
      "is_recurring": true,
      "recurrence": {
        "frequency": "monthly",
        "interval": 3
      },
      "status": "pending"
    },
    {
      "item_id": "sched-002",
      "type": "exam",
      "name": "Dilated Eye Exam",
      "due_date": "2026-03-15T09:00:00Z",
      "priority": 2,
      "is_recurring": true,
      "recurrence": {
        "frequency": "yearly",
        "interval": 1
      },
      "status": "pending"
    }
  ]
}
```

### GET /v1/patients/{id}/schedule-summary
Get schedule statistics for a patient.

```bash
curl http://localhost:8083/v1/patients/patient-001/schedule-summary
```

**Response:**
```json
{
  "patient_id": "patient-001",
  "total_items": 12,
  "pending_items": 8,
  "overdue_items": 2,
  "completed_items": 2,
  "upcoming_in_week": 3,
  "upcoming_in_month": 7
}
```

### GET /v1/patients/{id}/overdue
Get overdue scheduled items.

```bash
curl http://localhost:8083/v1/patients/patient-001/overdue
```

### GET /v1/patients/{id}/upcoming
Get upcoming scheduled items.

```bash
curl "http://localhost:8083/v1/patients/patient-001/upcoming?days=30"
```

### GET /v1/patients/{id}/export
Export all patient data (pathways, schedules, audit logs).

```bash
curl http://localhost:8083/v1/patients/patient-001/export
```

**With download header:**
```bash
curl "http://localhost:8083/v1/patients/patient-001/export?download=true" \
  -o patient-001-export.json
```

**Response:**
```json
{
  "patient_id": "patient-001",
  "exported_at": "2025-12-20T15:30:00Z",
  "version": "3.0.0",
  "data": {
    "pathways": {
      "active": [...],
      "count": 2
    },
    "schedule": {
      "items": [...],
      "summary": {...},
      "overdue": [...]
    }
  },
  "metadata": {
    "format": "json",
    "service": "kb-3-guidelines"
  }
}
```

### POST /v1/patients/{id}/start-protocol
Start a protocol for a specific patient.

```bash
curl -X POST http://localhost:8083/v1/patients/patient-001/start-protocol \
  -H "Content-Type: application/json" \
  -d '{
    "protocol_id": "DIABETES-ADA-2024",
    "context": {
      "diagnosis_date": "2025-01-15",
      "type": "Type 2",
      "initial_hba1c": 8.5
    }
  }'
```

---

## Scheduling Operations

### GET /v1/schedule/{patientId}
Get patient schedule (alias for /v1/patients/{id}/schedule).

```bash
curl http://localhost:8083/v1/schedule/patient-001
```

### GET /v1/schedule/{patientId}/pending
Get pending scheduled items.

```bash
curl http://localhost:8083/v1/schedule/patient-001/pending
```

### POST /v1/schedule/{patientId}/add
Add a new scheduled item.

```bash
curl -X POST http://localhost:8083/v1/schedule/patient-001/add \
  -H "Content-Type: application/json" \
  -d '{
    "type": "lab",
    "name": "Lipid Panel",
    "due_date": "2025-12-30T08:00:00Z",
    "priority": 2,
    "is_recurring": true,
    "recurrence": {
      "frequency": "yearly",
      "interval": 1
    }
  }'
```

**Response:**
```json
{
  "item_id": "sched-new-001",
  "patient_id": "patient-001",
  "type": "lab",
  "name": "Lipid Panel",
  "due_date": "2025-12-30T08:00:00Z",
  "status": "pending",
  "created_at": "2025-12-20T15:30:00Z"
}
```

**Recurrence Patterns:**
| Frequency | Interval | Example |
|-----------|----------|---------|
| daily | 1 | Every day |
| weekly | 2 | Every 2 weeks |
| monthly | 3 | Every 3 months (quarterly) |
| yearly | 1 | Annually |

### POST /v1/schedule/{patientId}/complete
Mark a scheduled item as complete.

```bash
curl -X POST http://localhost:8083/v1/schedule/patient-001/complete \
  -H "Content-Type: application/json" \
  -d '{
    "item_id": "sched-001",
    "completed_by": "Lab Tech",
    "result": "HbA1c: 7.2%"
  }'
```

**Response (with next occurrence for recurring items):**
```json
{
  "success": true,
  "completed_item": {
    "item_id": "sched-001",
    "name": "HbA1c Test",
    "completed_at": "2025-12-20T10:30:00Z"
  },
  "next_occurrence": {
    "item_id": "sched-001-next",
    "due_date": "2026-03-20T10:00:00Z",
    "status": "pending"
  }
}
```

---

## Temporal Operations

KB-3 implements **Allen's Interval Algebra** for temporal reasoning.

### POST /v1/temporal/evaluate
Evaluate a temporal relation between two intervals.

```bash
curl -X POST http://localhost:8083/v1/temporal/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "target": {
      "start": "2025-12-20T10:00:00Z",
      "end": "2025-12-20T11:00:00Z"
    },
    "reference": {
      "start": "2025-12-20T11:00:00Z",
      "end": "2025-12-20T12:00:00Z"
    },
    "operator": "meets"
  }'
```

**Response:**
```json
{
  "result": true,
  "operator": "meets",
  "explanation": "Target interval ends exactly when reference interval starts"
}
```

**Available Operators:**

| Operator | Description | Example |
|----------|-------------|---------|
| `before` | Target ends before reference starts | Lactate must be drawn BEFORE antibiotics |
| `after` | Target starts after reference ends | Follow-up AFTER initial treatment |
| `meets` | Target ends when reference starts | Blood culture MEETS antibiotic administration |
| `overlaps` | Intervals share time period | Observation periods overlap |
| `during` | Target contained within reference | Action must occur DURING shift |
| `contains` | Target contains reference | Monitoring window CONTAINS treatment |
| `starts` | Both start at same time | Dual therapy STARTS together |
| `ends` | Both end at same time | Concurrent treatments END together |
| `equals` | Intervals are identical | Same time window |
| `within` | Target within offset of reference | Within 1 hour of admission |
| `within_before` | Target within offset before reference | 30min before procedure |
| `within_after` | Target within offset after reference | 2h after medication |
| `same_as` | Equivalent intervals | Alias for equals |

### POST /v1/temporal/next-occurrence
Calculate the next occurrence based on a recurrence pattern.

```bash
curl -X POST http://localhost:8083/v1/temporal/next-occurrence \
  -H "Content-Type: application/json" \
  -d '{
    "from": "2025-12-20T10:00:00Z",
    "recurrence": {
      "frequency": "monthly",
      "interval": 3
    }
  }'
```

**Response:**
```json
{
  "next_occurrence": "2026-03-20T10:00:00Z",
  "pattern": {
    "frequency": "monthly",
    "interval": 3
  }
}
```

### POST /v1/temporal/validate-constraint
Validate if a timing constraint is satisfied.

```bash
curl -X POST http://localhost:8083/v1/temporal/validate-constraint \
  -H "Content-Type: application/json" \
  -d '{
    "action_time": "2025-12-20T11:15:00Z",
    "reference_time": "2025-12-20T10:30:00Z",
    "constraint": {
      "operator": "within_after",
      "offset": 3600000000000
    }
  }'
```

**Note:** Offset is in nanoseconds (1 hour = 3,600,000,000,000 ns)

**Response:**
```json
{
  "valid": true,
  "constraint_status": "MET",
  "elapsed": "45m",
  "allowed": "1h",
  "margin": "15m"
}
```

---

## Alert Management

### POST /v1/alerts/process
Process all pending alerts across pathways.

```bash
curl -X POST http://localhost:8083/v1/alerts/process
```

**Response:**
```json
{
  "processed_at": "2025-12-20T15:30:00Z",
  "alerts_generated": [
    {
      "alert_id": "alert-001",
      "type": "constraint_approaching",
      "severity": "warning",
      "pathway_id": "abc123",
      "action_id": "abc123-3h_bundle-antibiotics",
      "message": "Antibiotic administration deadline approaching in 15 minutes",
      "deadline": "2025-12-20T15:45:00Z"
    },
    {
      "alert_id": "alert-002",
      "type": "constraint_overdue",
      "severity": "critical",
      "pathway_id": "def456",
      "action_id": "def456-recognition-lactate",
      "message": "Initial lactate is 10 minutes overdue",
      "deadline": "2025-12-20T15:20:00Z"
    }
  ],
  "summary": {
    "total_processed": 45,
    "alerts_generated": 2,
    "critical": 1,
    "warning": 1
  }
}
```

### GET /v1/alerts/overdue
Get all overdue items across all patients.

```bash
curl http://localhost:8083/v1/alerts/overdue
```

**Response:**
```json
{
  "overdue_items": [
    {
      "patient_id": "patient-001",
      "type": "pathway_action",
      "item_id": "abc123-recognition-lactate",
      "name": "Initial Lactate",
      "deadline": "2025-12-20T11:00:00Z",
      "overdue_by": "30m",
      "severity": "critical"
    },
    {
      "patient_id": "patient-002",
      "type": "scheduled_item",
      "item_id": "sched-005",
      "name": "INR Check",
      "deadline": "2025-12-19T10:00:00Z",
      "overdue_by": "1d5h",
      "severity": "moderate"
    }
  ],
  "count": 2
}
```

---

## Batch Operations

### POST /v1/batch/start-protocols
Start multiple protocols at once.

```bash
curl -X POST http://localhost:8083/v1/batch/start-protocols \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {
        "pathway_id": "DIABETES-ADA-2024",
        "patient_id": "patient-001",
        "context": {"type": "Type 2"}
      },
      {
        "pathway_id": "HTN-ACCAHA-2017",
        "patient_id": "patient-001",
        "context": {"stage": "2"}
      },
      {
        "pathway_id": "DIABETES-ADA-2024",
        "patient_id": "patient-002",
        "context": {"type": "Type 1"}
      }
    ]
  }'
```

**Response:**
```json
{
  "results": [
    {
      "patient_id": "patient-001",
      "pathway_id": "DIABETES-ADA-2024",
      "instance_id": "batch-001",
      "status": "started"
    },
    {
      "patient_id": "patient-001",
      "pathway_id": "HTN-ACCAHA-2017",
      "instance_id": "batch-002",
      "status": "started"
    },
    {
      "patient_id": "patient-002",
      "pathway_id": "DIABETES-ADA-2024",
      "instance_id": "batch-003",
      "status": "started"
    }
  ],
  "summary": {
    "total": 3,
    "successful": 3,
    "failed": 0
  }
}
```

---

## Governance

### GET /v1/guidelines
List all clinical guidelines.

```bash
curl http://localhost:8083/v1/guidelines
```

### GET /v1/guidelines/{id}
Get specific guideline details.

```bash
curl http://localhost:8083/v1/guidelines/GL-SEPSIS-2021
```

### POST /v1/conflicts/resolve
Resolve a guideline conflict.

```bash
curl -X POST http://localhost:8083/v1/conflicts/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "conflict_id": "conflict-001",
    "context": {
      "patient_id": "patient-001",
      "age": 65,
      "conditions": ["diabetes", "CKD"]
    }
  }'
```

### GET /v1/safety-overrides
List active safety overrides.

```bash
curl http://localhost:8083/v1/safety-overrides
```

### POST /v1/safety-overrides
Create a safety override.

```bash
curl -X POST http://localhost:8083/v1/safety-overrides \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pregnancy Override",
    "trigger_conditions": {
      "pregnancy": true
    },
    "override_action": {
      "type": "contraindicate",
      "affected_medications": ["warfarin", "methotrexate"]
    }
  }'
```

### POST /v1/versions
Create a new guideline version.

```bash
curl -X POST http://localhost:8083/v1/versions \
  -H "Content-Type: application/json" \
  -d '{
    "guideline_id": "GL-DIABETES-2024",
    "version": "2024.2",
    "changes": "Updated HbA1c targets for elderly patients"
  }'
```

### POST /v1/versions/{id}/approve
Process version approval.

```bash
curl -X POST http://localhost:8083/v1/versions/ver-001/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approver": "Dr. Medical Director",
    "decision": "approved",
    "comments": "Reviewed and approved for implementation"
  }'
```

---

## Common Patterns

### Pattern 1: ED Sepsis Workflow

```bash
# 1. Patient arrives with suspected sepsis
INSTANCE=$(curl -s -X POST http://localhost:8083/v1/pathways/start \
  -H "Content-Type: application/json" \
  -d '{
    "pathway_id": "SEPSIS-SEP1-2021",
    "patient_id": "ED-2025-001",
    "context": {"source": "ED", "chief_complaint": "fever, confusion"}
  }' | jq -r '.instance_id')

# 2. Complete sepsis screening
curl -X POST "http://localhost:8083/v1/pathways/$INSTANCE/complete-action" \
  -H "Content-Type: application/json" \
  -d '{"action_id": "'$INSTANCE'-recognition-screen", "completed_by": "RN Jones"}'

# 3. Check what's pending
curl "http://localhost:8083/v1/pathways/$INSTANCE/pending"

# 4. Monitor constraints
curl "http://localhost:8083/v1/pathways/$INSTANCE/constraints"
```

### Pattern 2: Chronic Disease Onboarding

```bash
# 1. Start diabetes management for new patient
curl -X POST http://localhost:8083/v1/patients/patient-new/start-protocol \
  -H "Content-Type: application/json" \
  -d '{
    "protocol_id": "DIABETES-ADA-2024",
    "context": {
      "diagnosis_date": "2025-12-01",
      "type": "Type 2",
      "initial_hba1c": 9.2
    }
  }'

# 2. View generated schedule
curl http://localhost:8083/v1/patients/patient-new/schedule

# 3. Get schedule summary
curl http://localhost:8083/v1/patients/patient-new/schedule-summary
```

### Pattern 3: Compliance Monitoring

```bash
# 1. Get all overdue items across organization
curl http://localhost:8083/v1/alerts/overdue

# 2. Process alerts to generate notifications
curl -X POST http://localhost:8083/v1/alerts/process

# 3. Export patient data for compliance review
curl "http://localhost:8083/v1/patients/patient-001/export?download=true" \
  -o compliance-review.json
```

### Pattern 4: Multi-Patient Batch Operations

```bash
# Start preventive care for multiple patients
curl -X POST http://localhost:8083/v1/batch/start-protocols \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"pathway_id": "ADULT-USPSTF", "patient_id": "p1", "context": {"age": 55, "sex": "M"}},
      {"pathway_id": "ADULT-USPSTF", "patient_id": "p2", "context": {"age": 62, "sex": "F"}},
      {"pathway_id": "CANCER-SCREENING", "patient_id": "p2", "context": {"age": 62, "sex": "F"}}
    ]
  }'
```

---

## Error Handling

All endpoints return standard error responses:

```json
{
  "error": "pathway_not_found",
  "message": "No pathway found with instance ID: xyz123",
  "status": 404
}
```

| Status Code | Meaning |
|-------------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request (invalid input) |
| 404 | Not Found |
| 409 | Conflict (e.g., pathway already exists) |
| 500 | Internal Server Error |

---

## Rate Limits

| Endpoint Category | Rate Limit |
|-------------------|------------|
| Health/Status | Unlimited |
| Read Operations | 1000/min |
| Write Operations | 100/min |
| Batch Operations | 10/min |

---

## Support

- **Service Health**: `GET /health`
- **API Version**: `GET /version`
- **Documentation**: This guide
- **Cross-Check Report**: `KB3_IMPLEMENTATION_CROSSCHECK.md`
