# KB-14: Care Navigator & Tasking Engine

**Clinical Knowledge Platform - Task Orchestration Service**

KB-14 is the action orchestration layer that converts clinical intelligence from KB-3 (Temporal Service), KB-9 (Care Gaps Service), and KB-12 (Order Sets & Care Plans) into assigned, tracked, and escalated tasks. It ensures that every clinical alert results in coordinated action with clear ownership and accountability.

## Core Principle

> "Every alert must result in an assigned task with a deadline, an owner, and an escalation path."

Without KB-14, clinical intelligence becomes noise. With KB-14, intelligence becomes coordinated care.

---

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

---

## Features

### Task Generation
- **Multi-Source Ingestion**: Creates tasks from KB-3 alerts, KB-9 care gaps, and KB-12 care plan activities
- **Intelligent Deduplication**: Prevents duplicate tasks for the same clinical event
- **Priority Mapping**: Automatically assigns priority based on clinical urgency
- **SLA Assignment**: Sets deadlines based on task type and clinical requirements

### Assignment Engine
- **Role-Based Routing**: Routes tasks to appropriate clinical roles
- **Workload Balancing**: Distributes tasks evenly across team members
- **Panel Attribution**: Assigns care coordinator based on patient's PCP
- **Skill Matching**: Matches tasks to team members with required skills

### Escalation Engine
- **4-Level Escalation**: WARNING → URGENT → CRITICAL → EXECUTIVE
- **Configurable Thresholds**: Standard (50/75/100/125%) vs Critical (25/50/75/100%)
- **Automatic Notifications**: Alerts supervisors and managers at each level
- **Audit Trail**: Complete history of escalation events

### FHIR Interoperability
- **FHIR R4 Task Resources**: Full support for FHIR Task search and read
- **Standard Compliance**: Maps KB-14 task lifecycle to FHIR task status
- **EHR Integration**: Ready for integration with FHIR-enabled EHR systems

---

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

---

## Escalation Model

### Four-Level Escalation (Standard Tasks)

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

---

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

---

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

---

## Notification Channels

| Priority | Channels |
|----------|----------|
| CRITICAL | Pager, SMS, In-App, Email |
| HIGH | Push, In-App, Email |
| NORMAL | In-App, Email |
| LOW | In-App only |

*Note: Current implementation uses stub/mock notifications (log-based). Real notification channels will be integrated in Phase 2.*

---

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

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP server port | 8091 |
| ENVIRONMENT | development/production | development |
| LOG_LEVEL | Logging level | info |
| DATABASE_URL | PostgreSQL connection string | (required) |
| REDIS_URL | Redis connection string | (required) |
| KB3_TEMPORAL_URL | KB-3 Temporal Service URL | http://localhost:8087 |
| KB9_CARE_GAPS_URL | KB-9 Care Gaps Service URL | http://localhost:8089 |
| KB12_ORDER_SETS_URL | KB-12 Order Sets Service URL | http://localhost:8090 |
| ESCALATION_CHECK_INTERVAL | Escalation check interval (seconds) | 60 |
| SYNC_INTERVAL_MINUTES | KB sync interval (minutes) | 5 |
| WORKERS_ENABLED | Enable background workers | true |

---

## Running the Service

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- Redis 7+
- Docker & Docker Compose (for containerized deployment)

### Local Development

```bash
# Install dependencies
go mod download

# Run migrations
go run ./cmd/migrate

# Start the service
go run ./cmd/server
```

### Docker

```bash
# Build and start all services
docker-compose up -d

# Check logs
docker-compose logs -f kb-14-care-navigator

# Stop all services
docker-compose down
```

### Using Makefile

```bash
# Show available commands
make help

# Build the binary
make build

# Run tests
make test

# Start with Docker
make docker-up

# Check health
make health
```

### Health Check

```bash
curl http://localhost:8091/health
```

Response:
```json
{
  "status": "healthy",
  "service": "kb-14-care-navigator",
  "version": "1.0.0",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "kb3": "ok",
    "kb9": "ok",
    "kb12": "ok"
  }
}
```

---

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

---

## Project Structure

```
kb-14-care-navigator/
├── cmd/server/
│   └── main.go              # HTTP server, routes, middleware
├── internal/
│   ├── api/                 # HTTP handlers
│   ├── cache/               # Redis caching
│   ├── clients/             # KB-3, KB-9, KB-12 clients
│   ├── config/              # Configuration
│   ├── database/            # PostgreSQL + GORM
│   ├── fhir/                # FHIR R4 mapping
│   ├── models/              # Domain models
│   ├── services/            # Business logic
│   └── workers/             # Background workers
├── migrations/              # SQL migrations
├── test/                    # Tests
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── IMPLEMENTATION_PLAN.md
└── README.md
```

---

## Dependencies

- Go 1.22+
- github.com/gin-gonic/gin
- github.com/google/uuid
- github.com/redis/go-redis/v9
- github.com/sirupsen/logrus
- github.com/spf13/viper
- github.com/prometheus/client_golang
- gorm.io/gorm
- gorm.io/driver/postgres

---

## Related Services

| Service | Port | Description |
|---------|------|-------------|
| **KB-3** | 8087 | Temporal Service - Monitoring schedules, alerts |
| **KB-9** | 8089 | Care Gaps Service - Quality measure gaps |
| **KB-12** | 8090 | Order Sets & Care Plans - Care plan activities, protocols |
| **KB-5** | 8085 | Lab Interpretation - Critical lab alerts |
| **KB-4** | 8084 | Medication Advisor - Medication review triggers |

---

## License

Proprietary - CardioFit Clinical Synthesis Hub

---

**KB-14 is the bridge from clinical intelligence to coordinated action.**
