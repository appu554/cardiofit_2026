# CardioFit KB-0 Governance Platform: Achievement Summary

**Project**: Clinical Synthesis Hub -- CardioFit
**Component**: KB-0 Governance Platform (End-to-End)
**Period**: January 2026
**Status**: Deployed to Google Cloud

---

## Table of Contents

1. [Executive Overview](#executive-overview)
2. [SPL Pipeline (Data Ingestion)](#1-spl-pipeline-data-ingestion)
3. [KB-0 Governance Platform (Go Backend)](#2-kb-0-governance-platform-go-backend)
4. [PostgreSQL Canonical Fact Store](#3-postgresql-canonical-fact-store)
5. [Governance Dashboard (Next.js UI)](#4-governance-dashboard-nextjs-ui)
6. [Auth0 Integration](#5-auth0-integration)
7. [Google Cloud Deployment](#6-google-cloud-deployment)
8. [Architecture Diagram](#architecture-diagram)
9. [Endpoint Reference](#endpoint-reference)
10. [Port Allocation](#port-allocation)

---

## Executive Overview

The KB-0 Governance Platform is the foundational governance layer of the CardioFit Clinical Synthesis Hub. It provides a complete pipeline for ingesting FDA drug safety data, deriving clinical facts with confidence scoring, routing those facts through a pharmacist review workflow, and promoting approved facts into a production-ready canonical fact store. The platform spans five subsystems built across four technology stacks (Python, Go, PostgreSQL, Next.js/TypeScript) and is deployed to Google Cloud.

**Key metrics at a glance:**

| Metric | Value |
|---|---|
| Derived facts ingested | 242 |
| Clinical facts approved | 1 |
| PostgreSQL tables | 33 |
| Database size | ~2.1 GB |
| Go service image size | 13 MB (compressed) |
| Dashboard pages | 7 |
| Auth0 roles supported | 3 (Admin, Pharmacist, Viewer) |
| Cloud VM | e2-medium (2 vCPU, 4 GB RAM) |

---

## 1. SPL Pipeline (Data Ingestion)

The DailyMed SPL pipeline fetches Structured Product Labeling XML data from the FDA, extracts clinical safety signals, and writes derived facts into the canonical fact store.

### What Was Built

- **DailyMed SPL fetcher** that retrieves FDA drug label XML documents for target drugs.
- **Clinical signal extraction** using MedDRA terminology to identify adverse reactions, warnings, and contraindications from SPL XML.
- **RxNorm mapping** that associates extracted terms with standardized RxCUI drug codes for unambiguous drug identification.
- **Confidence scoring engine** that computes a numeric confidence score and assigns a confidence band (HIGH, MEDIUM, LOW) to each extracted fact.
- **Fact writer** that inserts derived facts into the `derived_facts` table in the PostgreSQL canonical fact store.
- **Audit trail** via `ingestion_run` records created for each pipeline execution.

### Drugs Processed

| Drug | RxCUI |
|---|---|
| Metformin | 6809 |
| Lisinopril | 29046 |
| Atorvastatin | 83367 |

### Fact Schema

Each derived fact includes:

| Field | Description |
|---|---|
| `factType` | SAFETY_SIGNAL, CONTRAINDICATION, ADVERSE_REACTION, WARNING |
| `severity` | Clinical severity level of the signal |
| `meddraCode` | MedDRA preferred term code |
| `rxcui` | RxNorm Concept Unique Identifier |
| `confidenceScore` | Numeric score (0.0 -- 1.0) |
| `confidenceBand` | HIGH, MEDIUM, or LOW |
| `recommendation` | Suggested clinical action |
| `sourceReference` | DailyMed SPL document identifier |

---

## 2. KB-0 Governance Platform (Go Backend)

A Go REST API service that implements the governance review workflow, approval gates, conflict detection, and SLA tracking.

### What Was Built

- **Go HTTP service** running on port 8080 with `/api/v2/governance/*` endpoints.
- **Dual-table architecture**: `derived_facts` (pipeline output, pending review) and `clinical_facts` (approved, production-ready).
- **Review workflow** with state transitions: `PENDING_REVIEW` to `APPROVED`, `REJECTED`, or `ESCALATED`.
- **Full audit trail** recording every governance decision with reviewer ID, timestamp, and reason.
- **SLA tracking** with review deadlines, priority ranking, and compliance percentage calculations.
- **Approval gate system** requiring a minimum reference count before a fact can be approved.
- **Conflict detection** for overlapping or contradictory facts.
- **Dashboard metrics endpoint** providing real-time governance KPIs.
- **Cloud deployment** to Google Cloud (GCE VM at `34.46.243.149:8080`).

### Key Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/api/v2/governance/queue` | Review queue with SLA tracking and priority ranking |
| POST | `/api/v2/governance/facts/:id/approve` | Approve a derived fact (applies approval gate checks) |
| POST | `/api/v2/governance/facts/:id/reject` | Reject a fact with structured reason codes |
| POST | `/api/v2/governance/facts/:id/escalate` | Escalate a fact for senior review |
| GET | `/api/v2/governance/dashboard` | Real-time governance KPI metrics |
| GET | `/api/v2/governance/conflicts` | Detected fact conflicts |
| GET | `/api/v2/governance/audit` | Governance audit history |

---

## 3. PostgreSQL Canonical Fact Store

PostgreSQL 15 (Alpine) serving as the single source of truth for all clinical facts across their lifecycle.

### What Was Built

- **Database**: `canonical_facts` with 33 tables.
- **Container**: `kb-fact-store`, originally running locally on port 5433, now deployed to Google Cloud VM.
- **Schema migration tracking** via `schema_migrations` and `schema_version_registry` tables.

### Key Tables

| Table | Purpose | Record Count |
|---|---|---|
| `derived_facts` | Pipeline output, pending review | 242 |
| `clinical_facts` | Approved, production-ready facts | 1 |
| `fact_reviews` | Individual review decisions | -- |
| `governance_audit_log` | Full audit trail of all actions | -- |
| `governance_decisions` | Decision records with reasons | -- |
| `drug_master` | Canonical drug reference data | -- |
| `drug_rules` | Clinical drug rules | -- |
| `interaction_matrix` | Drug-drug interaction mappings | -- |
| `ddi_constitutional_rules` | DDI constitutional safety rules | -- |
| `formulary_coverage` | Formulary coverage data | -- |
| `lab_reference_ranges` | Laboratory reference ranges | -- |
| `loinc_reference_ranges` | LOINC-coded lab ranges | -- |

### Database Specifications

| Property | Value |
|---|---|
| Engine | PostgreSQL 15 (Alpine) |
| Database name | `canonical_facts` |
| Total tables | 33 |
| Database size | ~2.1 GB |
| Compressed dump size | 94 MB |
| Local port | 5433 |
| Cloud endpoint | `34.46.243.149:5433` |

---

## 4. Governance Dashboard (Next.js UI)

A Next.js 14 application providing a pharmacist-facing governance workflow interface with real-time metrics and role-based access.

### What Was Built

- **Framework**: Next.js 14, TypeScript, Tailwind CSS, React Query.
- **Port**: 3001 (local development).
- **Sidebar navigation** with role-based user info display.
- **Header** with real-time governance metrics (pending count, SLA compliance percentage).

### Pages

| Page | Purpose |
|---|---|
| Dashboard | KPI overview with governance metrics |
| Review Queue | Pharmacist workflow with sortable, filterable table of pending facts |
| Active Facts | View of approved, production-ready clinical facts |
| Conflicts | Detected contradictory or overlapping facts |
| Audit History | Complete trail of all governance decisions |
| Executor | Fact execution and deployment controls |
| Settings | Platform configuration |

### Review Queue Features

- Sortable and filterable table of pending derived facts.
- SLA status indicators: `ON_TRACK`, `AT_RISK`, `BREACHED`.
- Fact Detail view displaying clinical content, references, and confidence scoring.
- Decision Controls panel with three actions:
  - **Approve**: Validates approval gate requirements before promotion.
  - **Reject**: Requires structured rejection reason codes per 21 CFR Part 11.
  - **Escalate**: Routes fact to senior reviewer.

### Rejection Reason Codes (21 CFR Part 11 Compliant)

| Code | Description |
|---|---|
| `INSUFFICIENT_EVIDENCE` | Not enough supporting references |
| `CLINICAL_INACCURACY` | Fact content is clinically incorrect |
| `OUTDATED_INFORMATION` | Source data is no longer current |

### Role-Based Access Control

| Role | Permissions |
|---|---|
| ADMIN | Full access to all features and settings |
| PHARMACIST | Approve, reject, and escalate facts |
| VIEWER | Read-only access (Lock icon displayed on restricted actions) |

---

## 5. Auth0 Integration

Auth0 Universal Login providing authentication, session management, and role-based authorization across the dashboard.

### What Was Built

- **SDK**: `@auth0/nextjs-auth0` v4.
- **Auth0 tenant**: `dev-hfw6wda5wtf8l13c.au.auth0.com`.
- **Middleware** protecting all routes; unauthenticated users are redirected to Auth0 login.
- **Client-side `Auth0Provider`** wrapping the application for session management.
- **`useAuth` hook** providing user info, role checking, and permission flags.
- **Role extraction** from JWT custom claims (namespace: `https://cardiofit.com/roles`).
- **User display** in Sidebar (avatar, name, role) and Header.
- **Audit integration**: `ReviewActions` uses Auth0 `sub` claim as `reviewerId` for the governance audit trail.
- **Logout**: `/auth/logout` route.

### Auth0 Configuration

| Property | Value |
|---|---|
| SDK | `@auth0/nextjs-auth0` v4 |
| Tenant domain | `dev-hfw6wda5wtf8l13c.au.auth0.com` |
| JWT claims namespace | `https://cardiofit.com/roles` |
| Login route | Auth0 Universal Login (redirect) |
| Logout route | `/auth/logout` |

---

## 6. Google Cloud Deployment

The PostgreSQL fact store and KB-0 Go service are deployed to a Google Compute Engine VM.

### What Was Built

- **GCE VM** `cardiofit-kb0` in `us-central1-a`.
- **Machine type**: `e2-medium` (2 vCPU, 4 GB RAM, 30 GB disk).
- **OS**: Container-Optimized OS (`cos-stable`) with Docker pre-installed.
- **PostgreSQL container** (`kb-fact-store`) on port 5433 with full database restored from local dump.
- **KB-0 Go service container** (`kb0-service`) on port 8080, linked to the PostgreSQL container.
- **Firewall rule** `allow-kb0` opening TCP ports 8080 and 5433.
- **Docker image**: Multi-stage Go build producing a 13 MB compressed image, stored in Artifact Registry.
- **Local environment** (pipeline `.env`, dashboard `.env.local`) updated to point to cloud endpoints.

### Cloud Infrastructure

| Property | Value |
|---|---|
| GCP project | `project-2bbef9ac-174b-4b59-8fe` |
| GCP account | `nishanthumbi@gmail.com` |
| VM name | `cardiofit-kb0` |
| Zone | `us-central1-a` |
| Machine type | `e2-medium` |
| vCPU / RAM / Disk | 2 / 4 GB / 30 GB |
| OS | Container-Optimized OS (cos-stable) |
| External IP | `34.46.243.149` |
| Firewall rule | `allow-kb0` (TCP 8080, 5433) |

---

## Architecture Diagram

```
Local Development Machine                        GCE VM (34.46.243.149)
┌─────────────────────────────────┐         ┌──────────────────────────────────┐
│                                 │         │                                  │
│  ┌───────────────────────────┐  │         │  ┌────────────────────────────┐  │
│  │ SPL Pipeline              │  │         │  │ PostgreSQL 15 :5433        │  │
│  │ (DailyMed SPL Fetcher)    │──┼─writes──┼─▶│ Database: canonical_facts  │  │
│  │  - XML parsing            │  │         │  │  - derived_facts (242)     │  │
│  │  - MedDRA extraction      │  │         │  │  - clinical_facts (1)      │  │
│  │  - RxNorm mapping         │  │         │  │  - 33 tables total         │  │
│  │  - Confidence scoring     │  │         │  └────────────┬───────────────┘  │
│  └───────────────────────────┘  │         │               │                  │
│                                 │         │               │ linked           │
│  ┌───────────────────────────┐  │         │               │                  │
│  │ Governance Dashboard      │  │         │  ┌────────────▼───────────────┐  │
│  │ (Next.js 14 + Auth0)     │  │         │  │ KB-0 Go Service :8080      │  │
│  │ :3001                     │──┼─ API ───┼─▶│ /api/v2/governance/*       │  │
│  │  - Review Queue           │  │  calls  │  │  - Approval workflow       │  │
│  │  - Dashboard KPIs         │  │         │  │  - SLA tracking            │  │
│  │  - Audit History          │  │         │  │  - Conflict detection      │  │
│  │  - Role-based UI          │  │         │  │  - Audit logging           │  │
│  └──────────┬────────────────┘  │         │  └────────────────────────────┘  │
│             │                   │         │                                  │
│  ┌──────────▼────────────────┐  │         └──────────────────────────────────┘
│  │ Auth0 Universal Login     │  │
│  │ (dev-hfw6wda5wtf8l13c)    │  │
│  │  - JWT with role claims   │  │
│  │  - Session management     │  │
│  └───────────────────────────┘  │
│                                 │
└─────────────────────────────────┘
```

### Data Flow

```
DailyMed (FDA)
      │
      ▼
SPL Pipeline ──▶ derived_facts (PENDING_REVIEW)
                        │
                        ▼
              KB-0 Go Service (Governance API)
                        │
               ┌────────┼────────┐
               ▼        ▼        ▼
           APPROVED  REJECTED  ESCALATED
               │
               ▼
        clinical_facts (Production-Ready)
```

---

## Endpoint Reference

### KB-0 Governance API (`34.46.243.149:8080`)

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v2/governance/queue` | Review queue with SLA tracking |
| GET | `/api/v2/governance/dashboard` | Real-time governance KPIs |
| GET | `/api/v2/governance/conflicts` | Detected fact conflicts |
| GET | `/api/v2/governance/audit` | Governance audit history |
| POST | `/api/v2/governance/facts/:id/approve` | Approve a derived fact |
| POST | `/api/v2/governance/facts/:id/reject` | Reject with reason code |
| POST | `/api/v2/governance/facts/:id/escalate` | Escalate for senior review |

### Auth0 Routes (Dashboard)

| Route | Description |
|---|---|
| `/auth/login` | Redirect to Auth0 Universal Login |
| `/auth/logout` | End session and redirect |
| `/auth/callback` | Auth0 callback handler |

---

## Port Allocation

| Service | Port | Host |
|---|---|---|
| Governance Dashboard (Next.js) | 3001 | localhost |
| KB-0 Go Service | 8080 | 34.46.243.149 |
| PostgreSQL (canonical_facts) | 5433 | 34.46.243.149 |

---

## Technology Stack Summary

| Layer | Technology | Version |
|---|---|---|
| Data Ingestion | Python | 3.11+ |
| Backend API | Go | 1.21+ |
| Database | PostgreSQL | 15 (Alpine) |
| Frontend | Next.js | 14 |
| UI Styling | Tailwind CSS | 3.x |
| State Management | React Query | -- |
| Authentication | Auth0 (`@auth0/nextjs-auth0`) | v4 |
| Containerization | Docker | -- |
| Cloud Platform | Google Cloud (GCE) | -- |
| Container OS | Container-Optimized OS (cos-stable) | -- |

---

*Document generated: January 2026*
*Platform: CardioFit Clinical Synthesis Hub -- KB-0 Governance Platform*
