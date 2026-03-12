# KB-14: Care Navigator & Tasking Engine

**Clinical Knowledge Platform - Task Orchestration Service**

KB-14 is the action orchestration layer that converts clinical intelligence from KB-3 (Temporal Service), KB-9 (Care Gaps Service), and KB-12 (Order Sets & Care Plans) into assigned, tracked, and escalated tasks. It ensures that every clinical alert results in coordinated action with clear ownership and accountability.

## Core Principle

> "Every alert must result in an assigned task with a deadline, an owner, and an escalation path."

Without KB-14, clinical intelligence becomes noise. With KB-14, intelligence becomes coordinated care.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         KB-14 Care Navigator                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                  │
│  │    KB-3      │    │    KB-9      │    │   KB-12      │                  │
│  │  Temporal    │    │  Care Gaps   │    │ Order Sets   │                  │
│  │   Alerts     │    │              │    │ Care Plans   │                  │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘                  │
│         │                   │                   │                           │
│         ▼                   ▼                   ▼                           │
│  ┌──────────────────────────────────────────────────────────────────┐      │
│  │                    Task Generation Engine                         │      │
│  │  • Alert Processor  • Gap Processor  • Activity Processor        │      │
│  │  • Deduplication    • Priority       • Dependency                │      │
│  │  • Enrichment       • Mapping        • Sequencing                │      │
│  └──────────────────────────────────────────────────────────────────┘      │
│                                 │                                           │
│                                 ▼                                           │
│  ┌──────────────────────────────────────────────────────────────────┐      │
│  │                      Assignment Engine                            │      │
│  │  • Role-Based Routing      • Workload Balancing                  │      │
│  │  • Panel Attribution       • Skill Matching                       │      │
│  │  • Shift Awareness         • Language/Location                   │      │
│  └──────────────────────────────────────────────────────────────────┘      │
│                                 │                                           │
│                                 ▼                                           │
│  ┌──────────────────────────────────────────────────────────────────┐      │
│  │                    Task Lifecycle Engine                          │      │
│  │                                                                    │      │
│  │  CREATED → ASSIGNED → IN_PROGRESS → COMPLETED → VERIFIED         │      │
│  │      ↓         ↓           ↓                                      │      │
│  │  DECLINED  BLOCKED    ESCALATED                                   │      │
│  │                                                                    │      │
│  └──────────────────────────────────────────────────────────────────┘      │
│                                 │                                           │
│         ┌───────────────────────┼───────────────────────┐                  │
│         ▼                       ▼                       ▼                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                  │
│  │ Escalation   │    │ Notification │    │  Analytics   │                  │
│  │   Engine     │    │   Engine     │    │   Engine     │                  │
│  └──────────────┘    └──────────────┘    └──────────────┘                  │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│  Output Layer                                                                │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐              │
│  │  Worklists │ │ FHIR Task  │ │  Metrics   │ │ EHR Write  │              │
│  │            │ │ Resources  │ │  Export    │ │   Back     │              │
│  └────────────┘ └────────────┘ └────────────┘ └────────────┘              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Task Type Taxonomy

### Clinical Tasks (Licensed Clinician Required)

| Task Type | Source | Default Role | SLA | Priority |
|-----------|--------|--------------|-----|----------|
| CRITICAL_LAB_REVIEW | KB-5 | Physician | 1 hr | CRITICAL |
| MEDICATION_REVIEW | KB-4 | Pharmacist | 4 hr | HIGH |
| ABNORMAL_RESULT | KB-5 | Ordering MD | 24 hr | HIGH |
| THERAPEUTIC_CHANGE | KB-4/KB-3 | Physician | 48 hr | MEDIUM |
| CARE_PLAN_REVIEW | KB-12 | PCP | 7 days | MEDIUM |
| ACUTE_PROTOCOL_DEADLINE | KB-12/KB-3 | Attending | 1 hr | CRITICAL |

### Care Coordination Tasks

| Task Type | Source | Default Role | SLA | Priority |
|-----------|--------|--------------|-----|----------|
| CARE_GAP_CLOSURE | KB-9 | Care Coordinator | 30 days | MEDIUM |
| MONITORING_OVERDUE | KB-3 | Care Coordinator | 3 days | HIGH |
| TRANSITION_FOLLOWUP | KB-3/KB-12 | Transition Coordinator | 7 days | HIGH |
| ANNUAL_WELLNESS | KB-9 | Nurse | 30 days | LOW |
| CHRONIC_CARE_MGMT | KB-12 | Care Manager | 30 days | MEDIUM |

### Patient Outreach Tasks

| Task Type | Source | Default Role | SLA | Priority |
|-----------|--------|--------------|-----|----------|
| APPOINTMENT_REMIND | KB-3 | Scheduler | 3 days | LOW |
| MISSED_APPOINTMENT | KB-3 | Outreach Specialist | 2 days | MEDIUM |
| SCREENING_OUTREACH | KB-9 | Outreach Specialist | 14 days | MEDIUM |
| MEDICATION_REFILL | KB-3 | Outreach Specialist | 7 days | LOW |

### Administrative Tasks

| Task Type | Source | Default Role | SLA | Priority |
|-----------|--------|--------------|-----|----------|
| PRIOR_AUTH_NEEDED | KB-12 | Auth Specialist | 3 days | HIGH |
| REFERRAL_PROCESSING | KB-12 | Referral Coordinator | 5 days | MEDIUM |

## Escalation Engine

### Four-Level Escalation Model

```
Level 1: WARNING        @ 50% SLA elapsed  → Notify supervisor
Level 2: URGENT         @ 75% SLA elapsed  → Escalate to manager
Level 3: CRITICAL       @ 100% SLA elapsed → Deadline reached
Level 4: EXECUTIVE      @ 125% SLA elapsed → Executive notification
```

### Critical Task Escalation (Faster)

```
Level 1: @ 25% SLA elapsed  → Notify attending
Level 2: @ 50% SLA elapsed  → Escalate to supervisor
Level 3: @ 75% SLA elapsed  → Administrator notification
Level 4: @ 100% SLA elapsed → Executive escalation
```

## Integration Flows

### Flow 1: KB-9 Care Gap → Task

```
KB-9 Care Gap (HbA1c overdue)
    │
    ▼
POST /api/v1/tasks/from-care-gap
    │
    ▼
Task Created (CARE_GAP_CLOSURE)
    │
    ├── Priority: HIGH
    ├── SLA: 30 days
    ├── Actions: [Contact patient, Order lab]
    └── Auto-close: When HbA1c result received
    │
    ▼
Assignment Engine → Care Coordinator for patient's PCP
    │
    ▼
Notification → Task assigned notification
```

### Flow 2: KB-3 Temporal Alert → Task

```
KB-3 Alert (Warfarin INR overdue 14 days)
    │
    ▼
POST /api/v1/tasks/from-temporal-alert
    │
    ▼
Task Created (MONITORING_OVERDUE)
    │
    ├── Priority: HIGH (elevated due to anticoagulant)
    ├── SLA: 3 days (urgent)
    ├── Clinical Note: "Patient on warfarin - bleeding/clotting risk"
    └── Auto-close: When INR result received
    │
    ▼
Assignment Engine → Care Coordinator
    │
    ▼
Escalation: If not started in 24 hours → Escalate to MD
```

### Flow 3: KB-12 Acute Protocol → Urgent Task

```
KB-12/KB-3 Sepsis Protocol (Antibiotics deadline approaching)
    │
    ▼
POST /api/v1/tasks/from-protocol
    │
    ▼
Task Created (ACUTE_PROTOCOL_DEADLINE)
    │
    ├── Priority: CRITICAL
    ├── SLA: 15 minutes remaining
    ├── Notification: Pager + In-App + Overhead page
    └── Immediate escalation if not acknowledged
    │
    ▼
Assignment → Attending Physician + Charge Nurse
```

## API Reference

### Task Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tasks` | Create a task |
| GET | `/api/v1/tasks/{id}` | Get task details |
| PATCH | `/api/v1/tasks/{id}` | Update task |
| POST | `/api/v1/tasks/{id}/assign` | Assign task |
| POST | `/api/v1/tasks/{id}/start` | Start task |
| POST | `/api/v1/tasks/{id}/complete` | Complete task |
| POST | `/api/v1/tasks/{id}/escalate` | Escalate task |
| POST | `/api/v1/tasks/{id}/add-note` | Add note |

### Task Creation from Sources

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/tasks/from-care-gap` | Create from KB-9 care gap |
| POST | `/api/v1/tasks/from-temporal-alert` | Create from KB-3 alert |
| POST | `/api/v1/tasks/from-care-plan` | Create from KB-12 activity |
| POST | `/api/v1/tasks/from-protocol` | Create from KB-12 protocol |

### Worklists

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/worklist?userId={id}` | User worklist |
| GET | `/api/v1/worklist/team/{teamId}` | Team worklist |
| GET | `/api/v1/worklist/patient/{patientId}` | Patient tasks |
| GET | `/api/v1/worklist/overdue` | All overdue tasks |
| GET | `/api/v1/worklist/urgent` | All urgent tasks |
| GET | `/api/v1/worklist/unassigned` | Unassigned tasks |

### Assignment

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/assignment/suggest?taskId={id}` | Suggest assignees |
| POST | `/api/v1/assignment/bulk-assign` | Bulk assign tasks |
| GET | `/api/v1/assignment/workload?memberId={id}` | Get workload |

### Analytics

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/analytics/dashboard` | Dashboard metrics |
| GET | `/api/v1/analytics/sla?days={n}` | SLA compliance |
| GET | `/api/v1/analytics/trends?days={n}` | Task trends |
| GET | `/api/v1/analytics/care-gaps?days={n}` | Care gap analytics |

### FHIR Integration

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/fhir/Task` | Search FHIR Tasks |
| GET | `/fhir/Task/{id}` | Get FHIR Task |

### Sync/Integration

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/sync/kb3` | Sync from KB-3 |
| POST | `/api/v1/sync/kb9` | Sync from KB-9 |
| POST | `/api/v1/sync/kb12` | Sync from KB-12 |
| POST | `/api/v1/sync/all` | Sync from all sources |

## Notification Channels

| Priority | Channels |
|----------|----------|
| CRITICAL | Pager, SMS, In-App, Email |
| HIGH | Push, In-App, Email |
| NORMAL | In-App, Email |
| LOW | In-App only |

## Metrics & Analytics

### Operational Metrics (Prometheus)

```
kb14_tasks_created_total{type, source}
kb14_tasks_completed_total{type, outcome}
kb14_tasks_escalated_total{level, reason}
kb14_tasks_overdue_total
kb14_tasks_active_count
kb14_task_completion_time_seconds{type}
kb14_sla_compliance_rate
kb14_assignment_queue_size{assignee}
```

### Quality Metrics

- **Task Performance**: Avg time to assignment, first action, completion
- **SLA Compliance**: By task type, priority, team
- **Care Gap Closure**: Rate, avg days to close, by measure type
- **Team Performance**: Tasks per member, completion rate, workload distribution

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP server port | 8087 |
| KB3_SERVICE_URL | KB-3 Temporal Service URL | http://localhost:8083 |
| KB9_SERVICE_URL | KB-9 Care Gaps Service URL | http://localhost:8081 |
| KB12_SERVICE_URL | KB-12 Order Sets Service URL | http://localhost:8086 |

## Running the Service

### Local Development

```bash
go run ./cmd/server
```

### Docker

```bash
docker build -t kb14-care-navigator .
docker run -p 8087:8087 \
  -e KB3_SERVICE_URL=http://kb3:8083 \
  -e KB9_SERVICE_URL=http://kb9:8081 \
  -e KB12_SERVICE_URL=http://kb12:8086 \
  kb14-care-navigator
```

### Health Check

```bash
curl http://localhost:8087/health
```

## Value Proposition

### Without KB-14

- **KB-9**: "Patient has HbA1c gap" → Information with no action
- **KB-3**: "INR is overdue" → Alert that may be missed
- **KB-12**: "Annual eye exam due" → Care plan item untracked
- Multiple systems, no coordination, dropped balls

### With KB-14

- **KB-9 → KB-14**: "Task assigned to Mary, due in 30 days, patient contacted"
- **KB-3 → KB-14**: "Urgent task to care coordinator, escalated to MD at 3 days"
- **KB-12 → KB-14**: "Task created, reminder sent, appointment scheduled"
- Single worklist, clear ownership, tracked completion, automatic escalation

### Key Outcomes

✅ Every clinical alert becomes an assigned task  
✅ Clear ownership and accountability  
✅ Automated escalation prevents dropped balls  
✅ Workload balancing prevents burnout  
✅ Metrics enable continuous improvement  
✅ Patients receive timely care interventions  

## Project Structure

```
kb14-care-navigator/
├── cmd/server/
│   └── main.go              # HTTP server, routes, middleware
├── pkg/
│   ├── tasks/
│   │   ├── types.go         # Task models, enums, requests
│   │   ├── service.go       # Core task service
│   │   └── factory.go       # Task creation from sources
│   ├── assignment/
│   │   └── engine.go        # Assignment & routing logic
│   ├── escalation/
│   │   └── engine.go        # Escalation processing
│   ├── notification/
│   │   └── service.go       # Multi-channel notifications
│   ├── worklist/
│   │   └── service.go       # Worklist generation
│   ├── teams/
│   │   └── service.go       # Care team management
│   ├── integration/
│   │   └── clients.go       # KB-3, KB-9, KB-12 clients
│   ├── analytics/
│   │   └── service.go       # Analytics & reporting
│   └── fhir/
│       └── task_resource.go # FHIR R4 Task mapping
├── test/
│   ├── task_test.go         # Task service tests
│   └── factory_test.go      # Factory tests
├── Dockerfile
├── go.mod
└── README.md
```

## Dependencies

- Go 1.21+
- github.com/google/uuid

## Related Services

- **KB-3**: Temporal Service - Monitoring schedules, alerts
- **KB-9**: Care Gaps Service - Quality measure gaps
- **KB-12**: Order Sets & Care Plans - Care plan activities, protocols
- **KB-5**: Lab Interpretation - Critical lab alerts
- **KB-4**: Medication Advisor - Medication review triggers

---

**KB-14 is the bridge from clinical intelligence to coordinated action.**
