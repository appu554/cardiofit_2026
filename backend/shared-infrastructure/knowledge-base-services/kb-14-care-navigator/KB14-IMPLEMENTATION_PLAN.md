# KB-14 Care Navigator & Tasking Engine - Implementation Plan

## Executive Summary

KB-14 is the **action orchestration layer** that converts clinical intelligence from KB-3 (Temporal Service), KB-9 (Care Gaps Service), and KB-12 (Order Sets & Care Plans) into assigned, tracked, and escalated tasks. It ensures that every clinical alert results in coordinated action with clear ownership and accountability.

> **Core Principle**: "Every alert must result in an assigned task with a deadline, an owner, and an escalation path."

---

## Implementation Requirements

| Requirement | Selection |
|-------------|-----------|
| **Language** | Go 1.22+ |
| **Framework** | Gin Web Framework |
| **Database** | PostgreSQL (GORM) |
| **Caching** | Redis |
| **Port** | 8091 |
| **Deployment** | Docker + Docker Compose |
| **MVP Scope** | All three sources (KB-3, KB-9, KB-12) |
| **Notifications** | Mock/Stub (log-based) |
| **FHIR Support** | Full FHIR R4 Task resources |

---

## Directory Structure

```
kb-14-care-navigator/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
├── internal/
│   ├── api/
│   │   ├── server.go                  # Gin server & routing setup
│   │   ├── handlers.go                # Task CRUD handlers
│   │   ├── worklist_handlers.go       # Worklist endpoints
│   │   ├── assignment_handlers.go     # Assignment endpoints
│   │   ├── analytics_handlers.go      # Analytics endpoints
│   │   ├── fhir_handlers.go           # FHIR Task resource handlers
│   │   ├── sync_handlers.go           # KB sync endpoints
│   │   └── middleware.go              # Auth, logging, recovery
│   ├── cache/
│   │   └── redis.go                   # Redis caching layer
│   ├── clients/
│   │   ├── kb3_client.go              # KB-3 Temporal client
│   │   ├── kb9_client.go              # KB-9 Care Gaps client
│   │   └── kb12_client.go             # KB-12 Order Sets client
│   ├── config/
│   │   └── config.go                  # Viper configuration
│   ├── database/
│   │   ├── connection.go              # GORM PostgreSQL connection
│   │   └── repository.go              # Task/Team repositories
│   ├── models/
│   │   ├── task.go                    # Task, TaskStatus, TaskType
│   │   ├── assignment.go              # Assignment, TeamMember
│   │   ├── escalation.go              # Escalation, EscalationLevel
│   │   ├── worklist.go                # WorklistItem, Filters
│   │   ├── team.go                    # Team, TeamMember
│   │   ├── notification.go            # NotificationRequest (stub)
│   │   └── fhir.go                    # FHIR R4 Task resource
│   ├── services/
│   │   ├── task_service.go            # Core task CRUD
│   │   ├── task_factory.go            # Task creation from sources
│   │   ├── assignment_engine.go       # Assignment logic
│   │   ├── escalation_engine.go       # Escalation processing
│   │   ├── worklist_service.go        # Worklist generation
│   │   ├── notification_service.go    # Stub notification service
│   │   └── analytics_service.go       # Metrics & analytics
│   ├── workers/
│   │   ├── escalation_worker.go       # Background escalation checker
│   │   ├── kb3_sync_worker.go         # KB-3 sync worker
│   │   ├── kb9_sync_worker.go         # KB-9 sync worker
│   │   └── kb12_sync_worker.go        # KB-12 sync worker
│   └── fhir/
│       └── task_mapper.go             # FHIR Task <-> KB-14 Task
├── migrations/
│   ├── 001_create_tasks.sql
│   ├── 002_create_teams.sql
│   └── 003_create_escalations.sql
├── test/
│   ├── task_test.go
│   ├── factory_test.go
│   └── integration_test.go
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── IMPLEMENTATION_PLAN.md
└── README.md
```

---

## Implementation Phases

### Phase 1: Foundation (Day 1)

**Goal**: Establish project structure and core infrastructure

| File | Description |
|------|-------------|
| `go.mod` | Dependencies: gin, gorm, uuid, go-redis, viper, prometheus |
| `cmd/server/main.go` | Entry point with graceful shutdown |
| `internal/config/config.go` | Viper-based configuration management |
| `internal/database/connection.go` | PostgreSQL + GORM setup with auto-migration |

**Dependencies**:
```go
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/google/uuid v1.6.0
    github.com/redis/go-redis/v9 v9.4.0
    github.com/sirupsen/logrus v1.9.3
    github.com/spf13/viper v1.18.2
    github.com/prometheus/client_golang v1.18.0
    gorm.io/driver/postgres v1.5.6
    gorm.io/gorm v1.25.7
)
```

---

### Phase 2: Data Models (Day 1-2)

**Goal**: Define all domain entities

| File | Models |
|------|--------|
| `internal/models/task.go` | Task, TaskStatus, TaskType, TaskPriority, TaskSource, TaskAction, TaskNote |
| `internal/models/team.go` | Team, TeamMember |
| `internal/models/assignment.go` | AssignmentSuggestion |
| `internal/models/escalation.go` | Escalation, EscalationLevel, EscalationThresholds |
| `internal/models/worklist.go` | WorklistItem, WorklistFilters |
| `internal/models/notification.go` | NotificationRequest, NotificationChannel |
| `internal/models/fhir.go` | FHIRTask, FHIRReference, FHIRPeriod, FHIRIdentifier |

**Task Status Flow**:
```
CREATED → ASSIGNED → IN_PROGRESS → COMPLETED → VERIFIED
    ↓         ↓           ↓
 DECLINED  BLOCKED    ESCALATED → CANCELLED
```

**Task Types by Category**:

| Category | Task Types | Default Role | SLA |
|----------|-----------|--------------|-----|
| **Clinical** | CRITICAL_LAB_REVIEW | Physician | 1 hr |
| | MEDICATION_REVIEW | Pharmacist | 4 hr |
| | ABNORMAL_RESULT | Ordering MD | 24 hr |
| | THERAPEUTIC_CHANGE | Physician | 48 hr |
| | CARE_PLAN_REVIEW | PCP | 7 days |
| | ACUTE_PROTOCOL_DEADLINE | Attending | 1 hr |
| **Care Coordination** | CARE_GAP_CLOSURE | Care Coordinator | 30 days |
| | MONITORING_OVERDUE | Care Coordinator | 3 days |
| | TRANSITION_FOLLOWUP | Transition Coordinator | 7 days |
| | ANNUAL_WELLNESS | Nurse | 30 days |
| | CHRONIC_CARE_MGMT | Care Manager | 30 days |
| **Patient Outreach** | APPOINTMENT_REMIND | Scheduler | 3 days |
| | MISSED_APPOINTMENT | Outreach Specialist | 2 days |
| | SCREENING_OUTREACH | Outreach Specialist | 14 days |
| | MEDICATION_REFILL | Outreach Specialist | 7 days |
| **Administrative** | PRIOR_AUTH_NEEDED | Auth Specialist | 3 days |
| | REFERRAL_PROCESSING | Referral Coordinator | 5 days |

---

### Phase 3: Database Layer (Day 2)

**Goal**: Create database schema and repositories

| File | Description |
|------|-------------|
| `migrations/001_create_tasks.sql` | Tasks table with JSONB columns, indexes |
| `migrations/002_create_teams.sql` | Teams and team_members tables |
| `migrations/003_create_escalations.sql` | Escalations table |
| `internal/database/repository.go` | TaskRepository, TeamRepository, EscalationRepository |

**Key Table: tasks**
```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id VARCHAR(50) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'CREATED',
    priority VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
    source VARCHAR(30) NOT NULL,
    source_id VARCHAR(100),
    patient_id VARCHAR(50) NOT NULL,
    encounter_id VARCHAR(50),
    title VARCHAR(200) NOT NULL,
    description TEXT,
    instructions TEXT,
    clinical_note TEXT,
    assigned_to UUID,
    assigned_role VARCHAR(50),
    team_id UUID,
    due_date TIMESTAMPTZ,
    sla_minutes INTEGER DEFAULT 0,
    escalation_level INTEGER DEFAULT 0,
    completed_by UUID,
    completed_at TIMESTAMPTZ,
    verified_by UUID,
    verified_at TIMESTAMPTZ,
    outcome VARCHAR(50),
    actions JSONB DEFAULT '[]',
    notes JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ
);
```

---

### Phase 4: Integration Clients (Day 2-3)

**Goal**: Connect to KB-3, KB-9, KB-12 services

| File | Service | Port | Key Endpoints |
|------|---------|------|---------------|
| `internal/clients/kb3_client.go` | KB-3 Temporal | 8087 | `/v1/alerts/overdue`, `/v1/alerts/process` |
| `internal/clients/kb9_client.go` | KB-9 Care Gaps | 8089 | `/api/v1/care-gaps`, `/api/v1/measures` |
| `internal/clients/kb12_client.go` | KB-12 Order Sets | 8090 | `/api/v1/careplans/patient/{id}`, `/api/v1/ordersets` |

**Integration Mapping**:

| Source | Input | Task Type Created |
|--------|-------|-------------------|
| KB-3 | Overdue constraint | MONITORING_OVERDUE |
| KB-3 | Protocol deadline approaching | ACUTE_PROTOCOL_DEADLINE |
| KB-9 | HbA1c care gap | CARE_GAP_CLOSURE |
| KB-9 | Screening gap | SCREENING_OUTREACH |
| KB-12 | Care plan activity due | CARE_PLAN_REVIEW |
| KB-12 | Medication refill needed | MEDICATION_REFILL |

---

### Phase 5: Core Services (Day 3-4)

**Goal**: Implement business logic engines

| File | Responsibility |
|------|----------------|
| `internal/services/task_service.go` | CRUD operations, lifecycle management |
| `internal/services/task_factory.go` | Create tasks from KB-3/KB-9/KB-12 sources |
| `internal/services/assignment_engine.go` | Role-based routing, workload balancing, panel attribution |
| `internal/services/escalation_engine.go` | 4-level escalation with configurable thresholds |
| `internal/services/worklist_service.go` | Worklist generation with filtering/sorting |
| `internal/services/notification_service.go` | Stub notification (log-based) |
| `internal/services/analytics_service.go` | Dashboard metrics, SLA compliance, trends |

**Escalation Thresholds**:

| Level | Standard Tasks (% SLA) | Critical Tasks (% SLA) | Action |
|-------|------------------------|------------------------|--------|
| 1 WARNING | 50% | 25% | Notify supervisor |
| 2 URGENT | 75% | 50% | Escalate to manager |
| 3 CRITICAL | 100% | 75% | Deadline reached notification |
| 4 EXECUTIVE | 125% | 100% | Executive escalation |

---

### Phase 6: API Handlers (Day 4-5)

**Goal**: Expose all REST endpoints

| File | Endpoints |
|------|-----------|
| `internal/api/server.go` | Router setup, middleware registration |
| `internal/api/handlers.go` | Task CRUD, lifecycle operations |
| `internal/api/worklist_handlers.go` | User/team/patient worklists |
| `internal/api/assignment_handlers.go` | Suggest, bulk-assign, workload |
| `internal/api/analytics_handlers.go` | Dashboard, SLA, trends |
| `internal/api/fhir_handlers.go` | FHIR R4 Task search/read |
| `internal/api/sync_handlers.go` | Manual KB sync triggers |

**Complete API Reference**:

```
# Task Management
POST   /api/v1/tasks                     # Create task
GET    /api/v1/tasks/{id}                # Get task
PATCH  /api/v1/tasks/{id}                # Update task
POST   /api/v1/tasks/{id}/assign         # Assign task
POST   /api/v1/tasks/{id}/start          # Start task
POST   /api/v1/tasks/{id}/complete       # Complete task
POST   /api/v1/tasks/{id}/escalate       # Escalate task
POST   /api/v1/tasks/{id}/add-note       # Add note

# Task Creation from Sources
POST   /api/v1/tasks/from-care-gap       # Create from KB-9 care gap
POST   /api/v1/tasks/from-temporal-alert # Create from KB-3 alert
POST   /api/v1/tasks/from-care-plan      # Create from KB-12 activity
POST   /api/v1/tasks/from-protocol       # Create from KB-12 protocol

# Worklists
GET    /api/v1/worklist?userId={id}      # User worklist
GET    /api/v1/worklist/team/{teamId}    # Team worklist
GET    /api/v1/worklist/patient/{id}     # Patient tasks
GET    /api/v1/worklist/overdue          # All overdue tasks
GET    /api/v1/worklist/urgent           # All urgent tasks
GET    /api/v1/worklist/unassigned       # Unassigned tasks

# Assignment
GET    /api/v1/assignment/suggest?taskId={id}   # Suggest assignees
POST   /api/v1/assignment/bulk-assign           # Bulk assign
GET    /api/v1/assignment/workload?memberId={id} # Get workload

# Analytics
GET    /api/v1/analytics/dashboard       # Dashboard metrics
GET    /api/v1/analytics/sla?days={n}    # SLA compliance
GET    /api/v1/analytics/trends?days={n} # Task trends
GET    /api/v1/analytics/care-gaps?days={n} # Care gap analytics

# FHIR
GET    /fhir/Task                        # Search FHIR Tasks
GET    /fhir/Task/{id}                   # Get FHIR Task

# Sync
POST   /api/v1/sync/kb3                  # Sync from KB-3
POST   /api/v1/sync/kb9                  # Sync from KB-9
POST   /api/v1/sync/kb12                 # Sync from KB-12
POST   /api/v1/sync/all                  # Sync from all sources

# Health
GET    /health                           # Service health
GET    /ready                            # Readiness probe
GET    /live                             # Liveness probe
GET    /metrics                          # Prometheus metrics
```

---

### Phase 7: Background Workers (Day 5)

**Goal**: Implement automated processes

| File | Interval | Responsibility |
|------|----------|----------------|
| `internal/workers/escalation_worker.go` | 60 seconds | Check SLA thresholds, escalate tasks |
| `internal/workers/kb3_sync_worker.go` | 5 minutes | Poll KB-3 for overdue alerts |
| `internal/workers/kb9_sync_worker.go` | 5 minutes | Sync care gaps from KB-9 |
| `internal/workers/kb12_sync_worker.go` | 5 minutes | Sync care plan activities from KB-12 |

---

### Phase 8: FHIR & Caching (Day 5-6)

**Goal**: Implement FHIR interoperability and performance optimization

| File | Description |
|------|-------------|
| `internal/fhir/task_mapper.go` | Bidirectional mapping: KB-14 Task <-> FHIR R4 Task |
| `internal/cache/redis.go` | Task caching, worklist caching, TTL management |

**FHIR Task Status Mapping**:

| KB-14 Status | FHIR Status |
|--------------|-------------|
| CREATED | requested |
| ASSIGNED | accepted |
| IN_PROGRESS | in-progress |
| COMPLETED | completed |
| VERIFIED | completed |
| CANCELLED | cancelled |
| DECLINED | rejected |

---

### Phase 9: Docker & Infrastructure (Day 6)

**Goal**: Containerize and configure deployment

| File | Description |
|------|-------------|
| `Dockerfile` | Multi-stage Alpine build, non-root user, health check |
| `docker-compose.yml` | KB-14 + PostgreSQL (5438) + Redis (6386) |
| `Makefile` | Build, test, run, docker commands |

**Docker Compose Services**:

| Service | Port (Host:Container) | Purpose |
|---------|----------------------|---------|
| kb-14-care-navigator | 8091:8091 | Main application |
| kb-14-postgres | 5438:5432 | PostgreSQL database |
| kb-14-redis | 6386:6379 | Redis cache |

---

### Phase 10: Testing (Day 6+)

**Goal**: Validate implementation

| File | Scope |
|------|-------|
| `test/task_test.go` | Task service unit tests |
| `test/factory_test.go` | Task factory tests |
| `test/integration_test.go` | Full API integration tests |

---

## Environment Variables

```bash
# Server
PORT=8091
ENVIRONMENT=development
LOG_LEVEL=info

# Database
DATABASE_URL=postgres://kb14user:kb14password@localhost:5438/kb_care_navigator?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6386/0

# KB Service Clients
KB3_TEMPORAL_URL=http://localhost:8087
KB9_CARE_GAPS_URL=http://localhost:8089
KB12_ORDER_SETS_URL=http://localhost:8090

# Workers
ESCALATION_CHECK_INTERVAL=60
SYNC_INTERVAL_MINUTES=5
WORKERS_ENABLED=true
```

---

## Metrics (Prometheus)

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

---

## Timeline Summary

| Day | Phase | Deliverables |
|-----|-------|--------------|
| 1 | Foundation | Project structure, go.mod, main.go, config |
| 1-2 | Models | All domain models (Task, Team, Escalation, FHIR) |
| 2 | Database | Migrations, repositories |
| 2-3 | Clients | KB-3, KB-9, KB-12 HTTP clients |
| 3-4 | Services | Core business logic engines |
| 4-5 | Handlers | All REST API endpoints |
| 5 | Workers | Background escalation and sync workers |
| 5-6 | FHIR/Cache | FHIR mapping, Redis caching |
| 6 | Docker | Dockerfile, docker-compose, Makefile |
| 6+ | Testing | Unit and integration tests |

**Estimated Total Files**: ~40 files
**Estimated Lines of Code**: ~4,000-5,000

---

## Success Criteria

- [ ] All API endpoints functional and documented
- [ ] Tasks created from KB-3, KB-9, KB-12 sources
- [ ] Escalation engine working with 4 levels
- [ ] Assignment engine with role-based routing
- [ ] FHIR R4 Task resources exposed
- [ ] Background workers running reliably
- [ ] Docker deployment working
- [ ] Health checks passing
- [ ] Unit test coverage > 70%
