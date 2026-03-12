# KB-0 Governance Dashboard

A Next.js 14 dashboard for managing the Canonical Fact Store governance workflow. This is the pharmacist-facing UI for reviewing, approving, and managing clinical knowledge facts.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Governance Dashboard (Next.js)                   │
│                           Port: 3001                                │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ REST API
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    KB-0 Governance Platform (Go)                    │
│                           Port: 8080                                │
│  /api/v2/governance/queue    - Review queue operations              │
│  /api/v2/governance/facts    - Fact CRUD + review actions           │
│  /api/v2/governance/metrics  - Dashboard metrics                    │
│  /api/v2/governance/executor - Background watcher control           │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ SQL
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│              Canonical Fact Store (PostgreSQL Shared DB)            │
│  clinical_facts, governance_audit_log, governance_decisions         │
└─────────────────────────────────────────────────────────────────────┘
```

## Features

### Dashboard Overview
- Real-time governance metrics (pending reviews, SLA compliance, etc.)
- Queue summary by priority level
- Recent activity feed (21 CFR Part 11 compliant audit trail)
- Executor status and control

### Review Queue
- Filterable queue by priority, SLA status, fact type
- Search by drug name, RxCUI, or reviewer
- Sortable columns (priority, confidence, SLA due date)
- Pagination support

### Fact Detail View
- Complete clinical content display
- Confidence score and evidence level visualization
- Source authority and metadata
- Conflict detection and resolution panel
- Full audit history with digital signatures

### Review Actions
- Approve/Reject/Escalate workflow
- Required reason and optional clinical justification
- Override support (Emergency, Institutional, Clinical Judgment)
- 21 CFR Part 11 compliant recording

## Getting Started

### Prerequisites
- Node.js 18+
- KB-0 Governance Platform running on port 8080

### Installation

```bash
cd vaidshala/clinical-applications/ui/governance-dashboard

# Install dependencies
npm install

# Copy environment file
cp .env.example .env.local

# Start development server
npm run dev
```

The dashboard will be available at http://localhost:3001

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_KB0_API_URL` | KB-0 API URL for client-side requests | `http://localhost:8080/api/v2/governance` |
| `KB0_API_URL` | KB-0 API URL for server-side rewrites | `http://localhost:8080` |

## Project Structure

```
governance-dashboard/
├── app/                      # Next.js App Router pages
│   ├── page.tsx             # Dashboard home
│   ├── queue/page.tsx       # Review queue
│   ├── facts/[id]/page.tsx  # Fact detail view
│   ├── layout.tsx           # Root layout with sidebar
│   └── providers.tsx        # React Query provider
├── components/
│   ├── dashboard/           # Dashboard-specific components
│   │   ├── MetricsCards.tsx
│   │   ├── QueueSummary.tsx
│   │   ├── RecentActivity.tsx
│   │   ├── SlaOverview.tsx
│   │   ├── ExecutorStatus.tsx
│   │   ├── Sidebar.tsx
│   │   └── Header.tsx
│   ├── queue/               # Queue page components
│   │   ├── QueueTable.tsx
│   │   └── QueueFilters.tsx
│   └── facts/               # Fact detail components
│       ├── FactHeader.tsx
│       ├── FactContent.tsx
│       ├── FactMetadata.tsx
│       ├── ConflictPanel.tsx
│       ├── ReviewActions.tsx
│       └── AuditHistory.tsx
├── lib/
│   ├── api.ts               # KB-0 API client
│   └── utils.ts             # Utility functions
├── types/
│   └── governance.ts        # TypeScript type definitions
├── styles/
│   └── globals.css          # Tailwind + custom styles
└── hooks/                   # Custom React hooks
```

## Governance Workflow

### Fact Lifecycle
```
DRAFT → PENDING_REVIEW → APPROVED → ACTIVE
                      ↘ REJECTED
                      ↘ ESCALATED (to CMO)
```

### Priority Levels & SLAs
| Priority | SLA Target | Color |
|----------|------------|-------|
| CRITICAL | 24 hours | Red |
| HIGH | 48 hours | Orange |
| STANDARD | 7 days | Blue |
| LOW | 14 days | Gray |

### Confidence-Based Routing
| Confidence | Decision |
|------------|----------|
| ≥ 95% | Auto-approve eligible |
| 65-95% | Requires manual review |
| < 65% | Rejected |

### Authority Priority (Conflict Resolution)
```
ONC (1) > FDA (2) > USP (3) > ... > OHDSI (21)
```

## Scripts

```bash
npm run dev          # Start development server (port 3001)
npm run build        # Build for production
npm run start        # Start production server
npm run lint         # Run ESLint
npm run type-check   # Run TypeScript type checking
```

## Tech Stack

- **Framework**: Next.js 14 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **State**: React Query (TanStack Query v5)
- **Icons**: Lucide React
- **Date Handling**: date-fns

## Related Documentation

- [KB1 Phase 2 Governance Complete](../../../../../../claudedocs/KB1_PHASE2_GOVERNANCE_COMPLETE.md)
- [KB-0 Governance Platform](../../../../../../backend/shared-infrastructure/knowledge-base-services/kb-0-governance-platform/)
