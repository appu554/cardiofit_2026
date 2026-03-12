# KB-20: Patient Profile & Contextual State Engine

Central patient state service for the CardioFit Clinical Synthesis Hub. Stores demographics, lab values, medication state, computes CKD staging/eGFR, activates strata, manages context modifier registries, and stores ADR profiles with completeness grading.

Consumed by KB-22 (HPI Engine) for stratum-conditional medication overlay in the Bayesian differential engine. Resolves HPI gaps A01, A02, B01, B03-design, B04-design, D01, D06.

## Quick Start

```bash
# Start dependencies + service
docker-compose up -d

# Verify health
curl http://localhost:8131/health

# Build from source
go build -o bin/kb-20-patient-profile .
```

## Architecture

```
                        ┌────────────────────────────────┐
  Pipeline ──POST──►    │  KB-20 Patient Profile Engine   │
  SPLGuard ──POST──►    │                                 │
                        │  Gin HTTP (port 8131)           │
  KB-22 HPI ──GET──►    │  ├─ PatientService              │
  KB-1 Rules ─GET──►    │  ├─ LabService (F-05 validation) │
  KB-19 Proto ────►     │  ├─ MedicationService (F-01 FDC) │
                        │  ├─ StratumEngine (F-03 alerts)  │
                        │  ├─ CMRegistry                   │
                        │  ├─ ADRService                   │
                        │  └─ PipelineService              │
                        └───────┬──────────┬───────────────┘
                                │          │
                         PostgreSQL    Redis
                         (port 5436)  (port 6385)
```

## RED Findings (Pre-Implementation Review)

| Finding | Severity | Implementation |
|---------|----------|----------------|
| **F-01** FDC Decomposition | RED | `fdc_components` field in MedicationState; India-specific drug classes; `EffectiveDrugClasses()` decomposes FDCs for CM evaluation |
| **F-03** Intra-stratum alerts | RED | `MEDICATION_THRESHOLD_CROSSED` event on eGFR crossing 60/45/30/15; `ckd_substage` in stratum response for G3a vs G3b visibility |
| **F-05** Lab validation | RED | Plausibility ranges per lab type; `ACCEPTED`/`FLAGGED`/`REJECTED` status; SBP > DBP cross-validation |

## REST API

All endpoints under `/api/v1` unless noted.

### Infrastructure
| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness probe (DB + cache check) |
| GET | `/readiness` | Readiness probe (DB only) |
| GET | `/metrics` | Prometheus metrics |

### Patient
| Method | Path | Description |
|--------|------|-------------|
| POST | `/patient` | Create patient profile |
| GET | `/patient/:id/profile` | Full patient state (profile + labs + meds + eGFR) |
| PUT | `/patient/:id` | Update demographics |

### Labs
| Method | Path | Description |
|--------|------|-------------|
| POST | `/patient/:id/labs` | Add lab value (F-05 validation; 422 on rejection) |
| GET | `/patient/:id/labs` | Get lab history (`?lab_type=` filter) |
| GET | `/patient/:id/labs/egfr` | eGFR trajectory with trend classification |

### Medications
| Method | Path | Description |
|--------|------|-------------|
| POST | `/patient/:id/medications` | Add medication (FDC decomposition F-01) |
| PUT | `/patient/:id/medications/:med_id` | Update dose/frequency/status |
| GET | `/patient/:id/medications` | Active medication list |

### Stratum & Modifiers
| Method | Path | Description |
|--------|------|-------------|
| GET | `/patient/:id/stratum/:node_id` | Stratum query (ckd_substage F-03) |
| GET | `/modifiers/registry/:node_id` | CM registry for HPI node |

### ADR Profiles
| Method | Path | Description |
|--------|------|-------------|
| GET | `/adr/profiles/:drug_class` | ADR profiles (`?include_stubs=true`) |

### Pipeline Batch Write
| Method | Path | Description |
|--------|------|-------------|
| POST | `/pipeline/modifiers` | Batch write context modifiers |
| POST | `/pipeline/adr-profiles` | Batch write ADR profiles (merge strategy) |

## Data Models

- **PatientProfile** — Demographics, disease history, derived comorbidities, CKD status
- **LabEntry** — Lab values with plausibility validation (F-05)
- **MedicationState** — Active medications with FDC decomposition (F-01)
- **ContextModifier** — Per-node CM registry with completeness grading (FULL/PARTIAL/STUB)
- **AdverseReactionProfile** — ADR profiles mirroring Python schema, auto-computed completeness

## Clinical Logic

### eGFR Computation (CKD-EPI 2021)
Race-free equation auto-derived from creatinine values:
```
eGFR = 142 x min(Scr/k, 1)^a x max(Scr/k, 1)^-1.200 x 0.9938^age [x 1.012 if female]
```

### CKD Staging
| Stage | eGFR Range |
|-------|-----------|
| G1 | >= 90 |
| G2 | 60-89 |
| G3a | 45-59 |
| G3b | 30-44 |
| G4 | 15-29 |
| G5 | < 15 |

Auto-confirmation requires 2 eGFR readings < 60, >= 90 days apart (KDIGO criteria).

### Medication Threshold Crossings (F-03)
Events fire when eGFR crosses boundaries regardless of stratum change:
- **60**: Metformin — monitor renal function
- **45**: Metformin — cap dose 1500mg
- **30**: Metformin — reduce 500-1000mg; SGLT2i — efficacy reduced
- **15**: Metformin — discontinue

### Completeness Grading
- **FULL** (1.0x magnitude): drug + reaction + onset + mechanism + CM rule + confidence >= 0.70
- **PARTIAL** (0.7x magnitude): drug + reaction + (onset OR mechanism)
- **STUB** (excluded from clinical use): minimal data

## Events Published

| Event | Trigger |
|-------|---------|
| `STRATUM_CHANGE` | Patient stratum label changes |
| `SAFETY_ALERT` | Clinically significant safety event |
| `MEDICATION_THRESHOLD_CROSSED` | eGFR crosses 60/45/30/15 boundary (F-03) |
| `MEDICATION_CHANGE` | Medication added, updated, or discontinued |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8131 | HTTP server port |
| `DATABASE_URL` | `postgres://kb20_user:kb20_password@localhost:5436/kb_service_20?sslmode=disable` | PostgreSQL connection |
| `REDIS_URL` | `redis://localhost:6385` | Redis cache |
| `ENVIRONMENT` | development | Environment (development/production) |
| `DB_MAX_CONNECTIONS` | 25 | Connection pool size |

## Docker Services

| Service | Port | Image |
|---------|------|-------|
| kb20-postgres | 5436 | postgres:15-alpine |
| kb20-redis | 6385 | redis:7-alpine |
| kb20-service | 8131 | Custom Go build |

## Python Schema Alignment

Go models align with the existing Python schema at `shared/extraction/schemas/kb20_contextual.py`:
- `AdverseReactionProfile` mirrors Python's `AdverseReactionProfile` Pydantic model
- `ContextModifier` mirrors Python's `ContextualModifierFact`
- Completeness grading logic matches Python's `compute_completeness_grade()`
