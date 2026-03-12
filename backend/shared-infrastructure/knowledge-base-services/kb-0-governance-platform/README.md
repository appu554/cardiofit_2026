# KB-0: Unified Clinical Knowledge Governance Platform

Hospital-grade governance infrastructure for all 19 CardioFit Knowledge Bases.

## Overview

KB-0 provides shared infrastructure for:
- **Ingestion Pipeline Framework** - Common adapters for FDA, TGA, CMS, SNOMED, RxNorm
- **Approval Workflow Engine** - Three templates covering all risk levels
- **Unified Audit System** - Immutable compliance logging
- **Cross-KB Dashboard Support** - Aggregated metrics and reporting

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         KB-0: GOVERNANCE PLATFORM                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                    WORKFLOW ENGINE                                   │  │
│   ├─────────────────────────────────────────────────────────────────────┤  │
│   │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │  │
│   │   │  Template A │  │  Template B │  │  Template C │               │  │
│   │   │  High-Risk  │  │  Med-Risk   │  │  Low-Risk   │               │  │
│   │   │  Clinical   │  │  Quality    │  │  Infra      │               │  │
│   │   └─────────────┘  └─────────────┘  └─────────────┘               │  │
│   │                                                                     │  │
│   │   DRAFT → REVIEW → APPROVED → ACTIVE → RETIRED                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                    AUDIT & COMPLIANCE                                │  │
│   │   • Immutable PostgreSQL audit log (trigger-protected)              │  │
│   │   • Cross-KB compliance reporting                                   │  │
│   │   • Regulatory export (FDA, TGA, CMS formats)                       │  │
│   │   • 10+ year retention                                              │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## KB Coverage

### High-Risk Clinical (Template A: CLINICAL_HIGH)
| KB | Service | Reviewer | Approver |
|----|---------|----------|----------|
| KB-1 | Drug Dosing | 2× Pharmacist | CMO |
| KB-4 | Patient Safety | 2× Pharmacist | CMO |
| KB-5 | Drug Interactions | 2× Pharmacist | CMO |
| KB-12 | Order Sets | Physician + Pharmacist | CMO |
| KB-19 | Protocol Orchestrator | Specialist + Pharmacist | CMO |

### Medium-Risk Quality (Template B: QUALITY_MED)
| KB | Service | Reviewer | Approver |
|----|---------|----------|----------|
| KB-6 | Formulary | Pharmacist | P&T Chair |
| KB-8 | Calculators | Physician | Clinical Lead |
| KB-9 | Care Gaps | Quality Analyst | Quality Director |
| KB-13 | Quality Measures | Quality Analyst | Quality Director |
| KB-15 | Evidence Engine | Physician | Clinical Lead |
| KB-16 | Lab Interpretation | Pathologist | Lab Director |

### Low-Risk Infrastructure (Template C: INFRA_LOW)
| KB | Service | Validation | Approver |
|----|---------|------------|----------|
| KB-7 | Terminology | Auto + Spot | Terminology Manager |
| KB-2 | Clinical Context | Auto | Tech Lead |
| KB-3 | Temporal Logic | Auto | Tech Lead |
| KB-10 | Rules Engine | Auto | Tech Lead |
| KB-11 | Population Health | Auto | Analytics Lead |
| KB-14 | Care Navigator | Auto | Ops Lead |
| KB-17 | Population Registry | Auto | Analytics Lead |
| KB-18 | Governance Engine | Auto | Compliance |

## Quick Start

### Option A: Docker Deployment (Recommended)

The fastest way to run KB-0 with all dependencies:

```bash
# Build and start all services (KB-0, PostgreSQL, Redis)
make run

# Check health status
make health

# View logs
make logs

# Stop all services
make stop
```

**Services Started:**
| Service | Port | Description |
|---------|------|-------------|
| KB-0 Governance | 8080 | Main governance API |
| PostgreSQL | 5500 | Governance database |
| Redis | 6381 | Caching layer |

### Option B: Local Development

#### 1. Initialize Database

```bash
# Create KB-0 database
createdb kb0_governance

# Run schema
psql -d kb0_governance -f migrations/001_create_schema.sql
```

#### 2. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit with your values
export KB0_DATABASE_HOST=localhost
export KB0_DATABASE_PORT=5500
export KB0_DATABASE_NAME=kb0_governance
export KB0_DATABASE_USER=kb0_user
export KB0_DATABASE_PASSWORD=kb0_secure_password
export KB0_REDIS_URL=redis://localhost:6381
export KB0_PORT=8080
export KB1_URL=http://localhost:8081
```

#### 3. Start Service

```bash
# Using make (recommended)
make dev

# Or directly with go run
go run cmd/server/main.go
```

## Docker Infrastructure

### Files Created
- `Dockerfile` - Multi-stage Go 1.22 production build
- `docker-compose.yml` - Full service stack with PostgreSQL and Redis
- `Makefile` - Development and deployment commands
- `.env.example` - Environment configuration template
- `migrations/001_create_schema.sql` - Database schema

### Makefile Commands

```bash
make help          # Show all available commands
make build         # Build Docker image
make run           # Start all services
make stop          # Stop all services
make logs          # View KB-0 logs
make health        # Check service health
make clean         # Remove containers and volumes
make dev           # Run locally with go run
make test          # Run unit tests
make test-kb1      # Test KB-1 integration
make test-workflow # Test full governance workflow
make db-shell      # Connect to PostgreSQL
```

## KB-1 Drug Rules Integration

KB-0 acts as the governance orchestrator for KB-1 Drug Rules Service. The integration allows centralized governance control over drug dosing rules.

### Integration Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    KB-0 GOVERNANCE                               │
│                    (Port 8080)                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   /api/v1/kb1/pending      → GET pending drugs from KB-1        │
│   /api/v1/kb1/drugs/:id/review  → Submit pharmacist review      │
│   /api/v1/kb1/drugs/:id/approve → Submit CMO approval           │
│                                                                 │
│         ↓ Calls KB-1 Admin API ↓                                │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                    KB-1 DRUG RULES                              │
│                    (Port 8081)                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   /v1/admin/pending        ← Returns DRAFT/REVIEWED drugs       │
│   /v1/admin/review/:id     ← DRAFT → REVIEWED                   │
│   /v1/admin/approve/:id    ← REVIEWED → ACTIVE                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### KB-1 Integration Endpoints

| Method | KB-0 Endpoint | KB-1 Target | Description |
|--------|---------------|-------------|-------------|
| GET | `/api/v1/kb1/pending` | `/v1/admin/pending` | Get pending drugs needing review |
| POST | `/api/v1/kb1/drugs/:id/review` | `/v1/admin/review/:id` | Submit pharmacist review |
| POST | `/api/v1/kb1/drugs/:id/approve` | `/v1/admin/approve/:id` | Submit CMO approval |

### Governance Workflow: DRAFT → REVIEWED → ACTIVE

```
DRUG SUBMITTED (DRAFT)
        │
        ▼
┌───────────────────┐
│ PHARMACIST REVIEW │  POST /api/v1/kb1/drugs/:id/review
│   - Dosing check  │  Body: {reviewed_by, review_notes, checklist}
│   - Renal adjust  │
│   - Interactions  │
└─────────┬─────────┘
          │
          ▼
   STATUS: REVIEWED
          │
          ▼
┌───────────────────┐
│   CMO APPROVAL    │  POST /api/v1/kb1/drugs/:id/approve
│   - Final sign-off│  Body: {approved_by, review_notes, is_high_risk}
│   - Risk level    │
└─────────┬─────────┘
          │
          ▼
   STATUS: ACTIVE (Available for clinical use)
```

### Verified Workflow Example

The following workflow was tested with **Heparin Sodium**:

```bash
# 1. Get pending drugs from KB-1
curl http://localhost:8080/api/v1/kb1/pending

# 2. Submit pharmacist review
curl -X POST http://localhost:8080/api/v1/kb1/drugs/5d737c7f-347a-4aa2-8991-1a5140aa8a20/review \
  -H "Content-Type: application/json" \
  -d '{
    "reviewed_by": "Dr. Clinical Pharmacist",
    "review_notes": "All dosing verified against FDA label",
    "checklist": {
      "dosing_verified": true,
      "renal_verified": true,
      "hepatic_verified": true,
      "interactions_verified": true,
      "safety_verified": true
    }
  }'

# 3. Submit CMO approval
curl -X POST http://localhost:8080/api/v1/kb1/drugs/5d737c7f-347a-4aa2-8991-1a5140aa8a20/approve \
  -H "Content-Type: application/json" \
  -d '{
    "approved_by": "Dr. CMO",
    "review_notes": "Approved for clinical use",
    "is_high_risk": true
  }'
```

**Result**: Heparin Sodium governance status changed from `DRAFT` → `REVIEWED` → `ACTIVE`

### Go Client Integration

```go
import (
    kb0client "kb-0-governance-platform/pkg/client"
    kb0models "kb-0-governance-platform/internal/models"
)

// Create client
client := kb0client.NewClient(kb0client.Config{
    BaseURL: "http://localhost:8080",
    KB:      kb0models.KB1,
})

// Submit review
result, err := client.SubmitReview(ctx, &kb0client.ReviewRequest{
    ItemID:       "kb1:warfarin:us:2025.1",
    ReviewerID:   "pharmacist-001",
    ReviewerName: "Dr. Smith",
    ReviewerRole: "pharmacist",
    Notes:        "Dosing verified against FDA label",
})
```

## API Endpoints

### Workflow Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/workflow/review` | Submit pharmacist review |
| POST | `/api/v1/workflow/approve` | Approve item |
| POST | `/api/v1/workflow/reject` | Reject item |
| POST | `/api/v1/workflow/activate/{id}` | Activate approved item |

### Query Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/items/pending-review?kb=KB-1` | Pending review queue |
| GET | `/api/v1/items/pending-approval?kb=KB-1` | Pending approval queue |
| GET | `/api/v1/items/active?kb=KB-1` | Active items |
| GET | `/api/v1/metrics/{kb}` | KB governance metrics |
| GET | `/api/v1/metrics/all` | Cross-KB metrics |

### Audit Operations
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/audit/{item_id}` | Item audit trail |
| GET | `/api/v1/audit/export?kb=KB-1&since=2025-01-01` | Regulatory export |

## Directory Structure

```
kb-0-governance-platform/
├── cmd/server/          # Main application
├── internal/
│   ├── api/             # HTTP handlers
│   ├── audit/           # Audit logging
│   ├── database/        # PostgreSQL persistence
│   ├── models/          # Universal types
│   └── workflow/        # State machine engine
├── pkg/
│   ├── adapters/        # Ingestion adapters (FDA, TGA, etc.)
│   └── client/          # Go client for other KBs
├── sql/                 # Database schema
└── config/              # Configuration
```

## Workflow Templates

### Template A: CLINICAL_HIGH (High-Risk)
```
DRAFT → PRIMARY_REVIEW → SECONDARY_REVIEW → CMO_APPROVAL → ACTIVE
                    ↓                    ↓
                 REVISE              REJECTED
```
- 24h review SLA, 48h approval SLA
- Dual pharmacist/physician review required
- CMO attestation required for activation

### Template B: QUALITY_MED (Medium-Risk)
```
DRAFT → REVIEW → DIRECTOR_APPROVAL → ACTIVE
              ↓
           REJECTED
```
- 24h SLA for both stages
- Single specialist review
- Director approval

### Template C: INFRA_LOW (Low-Risk)
```
DRAFT → AUTO_VALIDATION → LEAD_APPROVAL → ACTIVE
                      ↓
                   REJECTED
```
- 1h validation SLA, 24h approval SLA
- Automated schema/regression testing
- Tech lead approval

## Database Schema

Key tables:
- `knowledge_items` - Universal items across all KBs
- `actors` - Users with role-based KB access
- `reviews` - Pharmacist/specialist reviews
- `approvals` - CMO/director approvals
- `audit_log` - **IMMUTABLE** audit trail
- `emergency_overrides` - Time-limited CMO overrides

## Migration from KB-1

KB-1's existing governance implementation can gradually migrate to KB-0:

1. **Phase 1**: KB-0 runs alongside KB-1's internal governance
2. **Phase 2**: KB-1 uses KB-0 client for new items
3. **Phase 3**: Migrate existing KB-1 items to KB-0
4. **Phase 4**: Remove KB-1 internal governance code

## Cost Savings

| Metric | Without KB-0 | With KB-0 | Savings |
|--------|-------------|-----------|---------|
| Development | 43 weeks | 16 weeks | **63%** |
| Code Duplication | ~70% | ~5% | **65%** |
| Dashboards | 8+ separate | 1 unified | **7 dashboards** |

## License

Internal use only - CardioFit Clinical Synthesis Hub
