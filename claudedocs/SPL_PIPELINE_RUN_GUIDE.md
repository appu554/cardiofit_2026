# SPL FactStore Pipeline — Run Guide

> **Last Updated**: January 30, 2026

## Quick Start

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared

# Build the CLI
go build -o bin/spl-pipeline ./cmd/spl-pipeline

# Run for a single drug (uses cloud DB)
./bin/spl-pipeline --drug metformin \
  --db-url "postgres://kb_admin:kb_secure_password_2024@34.46.243.149:5433/canonical_facts?sslmode=disable"

# Run all 10 initial drugs
./bin/spl-pipeline --all-initial \
  --db-url "postgres://kb_admin:kb_secure_password_2024@34.46.243.149:5433/canonical_facts?sslmode=disable"

# Dry-run (no DB writes)
./bin/spl-pipeline --drug metformin --dry-run
```

---

## Environment Variables

### PostgreSQL (Cloud — GCE VM)

| Variable | Value |
|----------|-------|
| `KB0_DATABASE_HOST` | `34.46.243.149` |
| `KB0_DATABASE_PORT` | `5433` |
| `KB0_DATABASE_NAME` | `canonical_facts` |
| `KB0_DATABASE_USER` | `kb_admin` |
| `KB0_DATABASE_PASSWORD` | `kb_secure_password_2024` |
| **Full DB URL** | `postgres://kb_admin:kb_secure_password_2024@34.46.243.149:5433/canonical_facts?sslmode=disable` |

### PostgreSQL (Local — if running locally)

| Variable | Value |
|----------|-------|
| `KB0_DATABASE_HOST` | `localhost` |
| `KB0_DATABASE_PORT` | `5433` |
| All others | Same as above |

### KB-0 Governance API

| Variable | Value |
|----------|-------|
| `KB0_PORT` | `8080` |
| `KB0_API_URL` (cloud) | `http://34.46.243.149:8080` |
| `KB0_API_URL` (local) | `http://localhost:8080` |
| `KB0_REDIS_URL` | `redis://localhost:6380` |
| `KB1_URL` | `http://localhost:8081` |

### Auth0 (Governance Dashboard)

| Variable | Value |
|----------|-------|
| `AUTH0_DOMAIN` | `dev-hfw6wda5wtf8l13c.au.auth0.com` |
| `AUTH0_CLIENT_ID` | `PDIhzsjttlz4W94efli4H0vWYOUbgVjT` |
| `AUTH0_CLIENT_SECRET` | `ZT0JZzXY-_Me5hdzmYq4hK4eI7bR2kXEaKTjbtX9f6PXB1Mlkxma_HqbtRrgfWKH` |
| `AUTH0_SECRET` | `1fd8de07522f367f922d8023f8229cc242aff7b03bc76857f755fa9320ff7f29` |
| `AUTH0_AUDIENCE` | `https://kb0-governance-api` |
| `APP_BASE_URL` | `http://localhost:3001` |

### Google Cloud

| Variable | Value |
|----------|-------|
| **Project** | `project-2bbef9ac-174b-4b59-8fe` |
| **Account** | `nishanthumbi@gmail.com` |
| **VM Name** | `cardiofit-kb0` |
| **Zone** | `us-central1-a` |
| **External IP** | `34.46.243.149` |

---

## Pipeline Phases (A → I)

The SPL pipeline executes a 9-phase sequence:

| Phase | Name | Description |
|-------|------|-------------|
| **A** | Verify Spine | Check DB connectivity, schema migrations |
| **B** | Select Scope | Choose drugs by name, RxCUI, batch, or all-initial |
| **C** | SPL Acquisition | Fetch FDA drug labels from DailyMed API (free, no API key) |
| **D** | LOINC Routing | Route SPL sections by LOINC code to target KBs |
| **E** | Table Extraction | Parse structured tables (GFR dosing, DDI, adverse events) |
| **F** | DraftFact Creation | Generate canonical facts with confidence scores |
| **G** | Governance | Auto-approve or queue for review based on thresholds |
| **H** | KB Projection | Project approved facts to downstream KB services |
| **I** | Report | Summary of ingestion run with audit trail |

### Fact Types Generated

| Type | Description |
|------|-------------|
| `ORGAN_IMPAIRMENT` | Renal/hepatic dosing adjustments |
| `SAFETY_SIGNAL` | Black box warnings, contraindications |
| `REPRODUCTIVE_SAFETY` | Pregnancy, lactation, teratogenicity |
| `INTERACTION` | Drug-drug, drug-food, drug-lab |
| `FORMULARY` | Coverage tiers, prior auth |
| `LAB_REFERENCE` | Reference ranges, critical values |

---

## CLI Flags

```
--drug <name>        Run for a single drug (e.g., metformin)
--rxcui <code>       Run for a specific RxCUI
--all-initial        Run for all 10 initial drugs
--db-url <url>       PostgreSQL connection string
--dry-run            No database writes
--skip-llm           Skip LLM-based extraction
--skip-projection    Skip KB projection phase
--json               Output as JSON
```

### Initial 10 Drugs

metformin (6809), lisinopril (29046), atorvastatin (83367), amlodipine (17767), omeprazole (7646), levothyroxine (10582), simvastatin (36567), losartan (52175), albuterol (435), gabapentin (25480)

---

## Running the Full Stack

### 1. Start Cloud Services (already running)

```bash
# Verify cloud PostgreSQL
docker run --rm -e PGPASSWORD=kb_secure_password_2024 postgres:15-alpine \
  psql -h 34.46.243.149 -p 5433 -U kb_admin -d canonical_facts -c "SELECT count(*) FROM derived_facts;"

# Verify cloud KB-0
curl -s http://34.46.243.149:8080/health
```

### 2. Run the SPL Pipeline

```bash
cd backend/shared-infrastructure/knowledge-base-services/shared
go build -o bin/spl-pipeline ./cmd/spl-pipeline
./bin/spl-pipeline --drug metformin \
  --db-url "postgres://kb_admin:kb_secure_password_2024@34.46.243.149:5433/canonical_facts?sslmode=disable"
```

### 3. Start the Dashboard

```bash
cd vaidshala/clinical-applications/ui/governance-dashboard
npm run dev -- -p 3001
# Open http://localhost:3001 → Auth0 login → Review Queue
```

### 4. Review Facts in Dashboard

1. Log in via Auth0 (ADMIN or PHARMACIST role)
2. Navigate to **Review Queue**
3. Click a pending fact to see clinical detail
4. **Approve** / **Reject** / **Escalate** with audit reason
5. Approved facts move from `derived_facts` → `clinical_facts`

---

## GCloud VM Management

```bash
# SSH into VM
gcloud compute ssh cardiofit-kb0 --zone=us-central1-a --project=project-2bbef9ac-174b-4b59-8fe

# Check containers on VM
gcloud compute ssh cardiofit-kb0 --zone=us-central1-a --project=project-2bbef9ac-174b-4b59-8fe \
  --command="docker ps"

# View KB-0 logs
gcloud compute ssh cardiofit-kb0 --zone=us-central1-a --project=project-2bbef9ac-174b-4b59-8fe \
  --command="docker logs kb0-service --tail 50"

# Stop/Start VM (to save costs)
gcloud compute instances stop cardiofit-kb0 --zone=us-central1-a --project=project-2bbef9ac-174b-4b59-8fe
gcloud compute instances start cardiofit-kb0 --zone=us-central1-a --project=project-2bbef9ac-174b-4b59-8fe
# NOTE: External IP may change after restart — check with:
gcloud compute instances describe cardiofit-kb0 --zone=us-central1-a --project=project-2bbef9ac-174b-4b59-8fe \
  --format='value(networkInterfaces[0].accessConfigs[0].natIP)'
```

---

## Key File Locations

| File | Purpose |
|------|---------|
| `backend/shared-infrastructure/knowledge-base-services/shared/cmd/spl-pipeline/main.go` | Pipeline entry point |
| `backend/shared-infrastructure/knowledge-base-services/shared/factstore/pipeline.go` | Pipeline orchestration |
| `backend/shared-infrastructure/knowledge-base-services/shared/factstore/repository.go` | DB read/write ops |
| `backend/shared-infrastructure/knowledge-base-services/shared/datasources/dailymed/fetcher.go` | DailyMed SPL fetcher |
| `backend/shared-infrastructure/knowledge-base-services/kb-0-governance-platform/.env` | KB-0 env config |
| `vaidshala/clinical-applications/ui/governance-dashboard/.env.local` | Dashboard env config |
