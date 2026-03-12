# KB-3 Temporal Logic & Clinical Pathways Service

A comprehensive temporal reasoning engine for clinical decision support, implementing evidence-based protocols with time-bound constraints, clinical pathway state machines, and intelligent scheduling.

## Overview

KB-3 addresses **Gap 3: Temporal Logic & Workflow Orchestration** in the clinical knowledge base architecture by providing:

- **CQL-Compatible Temporal Operators** - Allen's Interval Algebra implementation for clinical reasoning
- **Clinical Pathway State Machines** - Stage-based protocol execution with entry/exit conditions
- **Time-Bound Protocol Enforcement** - Deadline tracking with alert escalation
- **Chronic Disease Scheduling** - Guideline-based recurrence patterns
- **Preventive Care Management** - Age/sex-appropriate screening schedules

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    KB-3 Temporal Service                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Temporal   │  │   Pathway   │  │      Scheduling         │  │
│  │  Operators  │  │   Engine    │  │       Engine            │  │
│  │             │  │             │  │                         │  │
│  │ - before    │  │ - stages    │  │ - recurrence patterns   │  │
│  │ - after     │  │ - actions   │  │ - due date tracking     │  │
│  │ - within    │  │ - triggers  │  │ - alert management      │  │
│  │ - during    │  │ - status    │  │ - compliance metrics    │  │
│  │ - overlaps  │  │             │  │                         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                     Protocol Library                            │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────────┐   │
│  │    Acute      │  │    Chronic    │  │    Preventive     │   │
│  │               │  │               │  │                   │   │
│  │ - Sepsis      │  │ - Diabetes    │  │ - Prenatal        │   │
│  │ - Stroke      │  │ - Heart Fail  │  │ - Well Child      │   │
│  │ - STEMI       │  │ - CKD         │  │ - Adult Prev      │   │
│  │ - DKA         │  │ - Anticoag    │  │ - Cancer Screen   │   │
│  │ - Trauma      │  │ - COPD        │  │ - Immunizations   │   │
│  │ - PE          │  │ - HTN         │  │                   │   │
│  └───────────────┘  └───────────────┘  └───────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Clinical Protocols Implemented

### Acute Care Protocols

| Protocol | Guideline Source | Key Temporal Constraints |
|----------|-----------------|-------------------------|
| Sepsis | Surviving Sepsis Campaign 2021, CMS SEP-1 | Antibiotics ≤1h, 3h/6h bundles |
| Stroke | AHA/ASA 2019 | Door-to-CT ≤25min, Door-to-needle ≤60min, tPA ≤4.5h |
| STEMI | ACC/AHA 2013 | ECG ≤10min, Door-to-balloon ≤90min |
| DKA | ADA 2024 | K+ check before insulin, 2h overlap on transition |
| Trauma | ATLS 10th Edition | TXA ≤3h from injury |
| PE | ESC 2019 | Anticoagulation ≤1h of diagnosis |

### Chronic Disease Schedules

| Schedule | Guideline Source | Key Monitoring |
|----------|-----------------|----------------|
| Diabetes | ADA Standards 2024 | HbA1c q3-6mo, annual eye/foot/nephropathy |
| Heart Failure | ACC/AHA/HFSA 2022 | 7-day post-DC follow-up, K+ 3-7d after RAAS |
| CKD | KDIGO 2024 | eGFR q3-12mo by stage, nephrology referral |
| Anticoagulation | CHEST Guidelines | INR ≤4 weeks, recheck 3-7d after dose change |
| COPD | GOLD 2024 | CAT score quarterly, annual spirometry |
| Hypertension | ACC/AHA 2017 | Monthly until at goal, then q3-6mo |

### Preventive Care Schedules

| Schedule | Target Population | Key Components |
|----------|------------------|----------------|
| Prenatal | Pregnant women | Visit schedule 8→40w, GBS 36w, GCT 24-28w |
| Well Child | Birth to 21 years | EPSDT schedule, developmental screening, immunizations |
| Adult Preventive | Adults 18+ | USPSTF recommendations by age/sex |
| Cancer Screening | Age-appropriate | Mammography, colonoscopy, cervical, lung (high-risk) |
| Immunizations | All ages | ACIP schedule, catch-up, travel vaccines |

## API Endpoints

### Health & Status
```
GET  /health              - Service health check
GET  /metrics             - Performance metrics
GET  /version             - Service version info
```

### Protocol Management
```
GET  /protocols           - List all protocols
GET  /protocols/acute     - List acute protocols
GET  /protocols/chronic   - List chronic schedules
GET  /protocols/preventive - List preventive schedules
GET  /protocols/{type}/{id} - Get specific protocol
```

### Pathway Operations
```
POST /pathways/start      - Start a pathway instance
GET  /pathways/{id}       - Get pathway status
GET  /pathways/{id}/pending - Get pending actions
GET  /pathways/{id}/overdue - Get overdue actions
GET  /pathways/{id}/constraints - Evaluate constraints
GET  /pathways/{id}/audit - Get audit log
POST /pathways/{id}/advance - Advance to next stage
POST /pathways/{id}/complete-action - Complete an action
```

### Patient Operations
```
GET  /patients/{id}/pathways - Get patient's active pathways
GET  /patients/{id}/schedule - Get patient's schedule
GET  /patients/{id}/schedule-summary - Get schedule summary
GET  /patients/{id}/overdue - Get overdue items
GET  /patients/{id}/upcoming?days=30 - Get upcoming items
GET  /patients/{id}/export - Export patient data
POST /patients/{id}/start-protocol - Start protocol for patient
```

### Scheduling Operations
```
GET  /schedule/{patientId} - Get patient schedule
GET  /schedule/{patientId}/pending - Get pending items
POST /schedule/{patientId}/add - Add scheduled item
POST /schedule/{patientId}/complete - Complete scheduled item
```

### Temporal Operations
```
POST /temporal/evaluate - Evaluate temporal relation
POST /temporal/next-occurrence - Calculate next occurrence
POST /temporal/validate-constraint - Validate constraint timing
```

### Alert Management
```
POST /alerts/process - Process all pending alerts
GET  /alerts/overdue - Get all overdue items
```

### Batch Operations
```
POST /batch/start-protocols - Start multiple protocols
```

## Quick Start

### Docker

```bash
# Build
docker build -t kb3-temporal-service .

# Run
docker run -p 8083:8083 kb3-temporal-service
```

### From Source

```bash
# Build
go build -o kb3-temporal-service ./cmd/server

# Run
./kb3-temporal-service

# Or with custom port
PORT=9000 ./kb3-temporal-service
```

### Test

```bash
go test ./test/... -v
```

## Usage Examples

### Start Sepsis Protocol
```bash
curl -X POST http://localhost:8083/pathways/start \
  -H "Content-Type: application/json" \
  -d '{
    "pathway_id": "SEPSIS-SEP1-2021",
    "patient_id": "patient-123",
    "context": {
      "lactate": 3.5,
      "sepsis_source": "pneumonia"
    }
  }'
```

### Check Pathway Status
```bash
curl http://localhost:8083/pathways/INST-xxxxx/status
```

### Get Overdue Actions
```bash
curl http://localhost:8083/pathways/INST-xxxxx/overdue
```

### Start Diabetes Management
```bash
curl -X POST http://localhost:8083/patients/patient-456/start-protocol \
  -H "Content-Type: application/json" \
  -d '{
    "protocol_id": "DIABETES-ADA-2024",
    "protocol_type": "chronic",
    "context": {
      "hba1c": 7.8,
      "diagnosis_date": "2023-01-15"
    }
  }'
```

### Add Scheduled Item
```bash
curl -X POST http://localhost:8083/schedule/patient-456/add \
  -H "Content-Type: application/json" \
  -d '{
    "type": "lab",
    "name": "HbA1c Check",
    "due_date": "2024-03-15T09:00:00Z",
    "priority": 2,
    "is_recurring": true,
    "recurrence": {
      "frequency": "monthly",
      "interval": 3
    }
  }'
```

### Calculate Next Occurrence
```bash
curl -X POST http://localhost:8083/temporal/next-occurrence \
  -H "Content-Type: application/json" \
  -d '{
    "from_time": "2024-01-01T00:00:00Z",
    "recurrence": {
      "frequency": "monthly",
      "interval": 3
    }
  }'
```

## Constraint Evaluation Status

Constraints are evaluated with the following statuses:

| Status | Description |
|--------|-------------|
| `PENDING` | Action not yet due |
| `MET` | Constraint satisfied within deadline |
| `APPROACHING` | Within alert threshold of deadline |
| `OVERDUE` | Past deadline but within grace period |
| `MISSED` | Past deadline and grace period |
| `NOT_APPLICABLE` | Constraint does not apply to this context |

## Temporal Operators (CQL-Compatible)

| Operator | Description |
|----------|-------------|
| `before` | Target occurs before reference |
| `after` | Target occurs after reference |
| `same_as` | Target and reference are equivalent |
| `meets` | Target ends exactly when reference starts |
| `overlaps` | Intervals share some time period |
| `within` | Target is within offset of reference |
| `within_before` | Target is within offset before reference |
| `within_after` | Target is within offset after reference |
| `during` | Target interval contained within reference |
| `contains` | Target interval contains reference |
| `starts` | Both start at same time |
| `ends` | Both end at same time |
| `equals` | Intervals are identical |

## Project Structure

```
kb3-temporal-service/
├── cmd/
│   └── server/
│       └── main.go           # HTTP server and routes
├── pkg/
│   └── temporal/
│       ├── types.go          # Core temporal types and operators
│       ├── pathway.go        # Clinical pathway engine
│       ├── acute_protocols.go # Acute care protocols
│       ├── chronic_schedules.go # Chronic disease schedules
│       ├── preventive_schedules.go # Preventive care schedules
│       ├── scheduling.go     # Scheduling engine
│       └── service.go        # Main service orchestration
├── test/
│   └── service_test.go       # Comprehensive test suite
├── Dockerfile
├── go.mod
└── README.md
```

## Performance Targets

| Metric | Target |
|--------|--------|
| Pathway Start | < 10ms |
| Constraint Evaluation | < 5ms |
| Schedule Query | < 5ms |
| P95 Latency | < 50ms |

## Integration with Other KBs

KB-3 integrates with:
- **KB-1**: Clinical rules reference temporal constraints
- **KB-2**: Drug interactions have temporal components
- **KB-4**: Diagnostic criteria use temporal patterns
- **KB-7**: Documentation templates with time-based fields
- **GO**: Orchestrator manages multi-KB temporal workflows

## License

Proprietary - Clinical Knowledge Base System

## Version

3.0.0 - Production Release
