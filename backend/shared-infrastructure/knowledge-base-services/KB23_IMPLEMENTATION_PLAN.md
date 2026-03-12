# KB-23 Decision Cards Engine — Implementation Plan

**Service**: Decision Cards Engine (Clinical Decision Card Generation & MCU Gate Management)
**Technology**: Go 1.22 / Gin / GORM / Redis
**Port**: 8134 | **PostgreSQL**: 5438 | **Redis**: 6387
**Pattern Source**: KB-22 HPI Engine (adapted)
**Specifications**: KB-23 Pre-Implementation Specification + Final Pre-Implementation Review + Supplementary Addendum + Three-Channel Safety Architecture

---

## Architecture Overview

KB-23 is the **decision synthesis layer** of the Vaidshala clinical engine. It receives a DifferentialSnapshot from KB-22, maps it to pre-authored CardTemplates, generates DecisionCards with MCU_GATE signals, and provides the enriched gate response that V-MCU consumes on every titration cycle. It also manages the treatment perturbation log and hypoglycaemia fast-path — two safety-critical input channels that bypass the standard KB-22 inference pipeline.

```
                        ┌─────────────────────────────────────────────────┐
                        │              KB-23 Decision Cards Engine         │
                        │                                                 │
  KB-22 (snapshot) ────►│  ┌──────────────────┐  ┌────────────────────┐  │──► V-MCU (MCU_GATE + enriched)
  KB-20 (clinical) ────►│  │  TemplateSelector │  │  CardBuilder       │  │──► KB-19 (events)
  KB-21 (adherence) ───►│  │  + ConfidenceTier │  │  + RecommComposer  │  │──► WhatsApp (patient cards)
  V-MCU (perturb.) ────►│  └────────┬─────────┘  └─────────┬──────────┘  │
  CGM/Gluco (hypo) ────►│           │  template_match       │             │
                        │           └──────────┬─────────────┘             │
                        │                      │                           │
                        │  ┌───────────────────▼───────────────────────┐  │
                        │  │    MCU Gate Manager                        │  │
                        │  │    (hysteresis + lifecycle + re-entry)     │  │
                        │  └───────────────────────────────────────────┘  │
                        └─────────────────────────────────────────────────┘
```

**Three safety-critical inputs**:
1. **KB-22 DifferentialSnapshot** — standard diagnostic inference path (Channel A)
2. **POST /safety/hypoglycaemia-alert** — device/V-MCU fast-path bypassing KB-22
3. **POST /perturbations** — V-MCU dose-change log for observation bias dampening

---

## Findings Register (All RED + AMBER — Cumulative Across 3 Documents)

### RED Findings (11 total — architectural constraints, non-retrofittable)

| ID | Finding | Source | Implementation Impact |
|----|---------|--------|----------------------|
| V-01 | FIRM threshold → node-configurable | Final Review | `confidence_tier.go`: per-template thresholds, `firm_medication_change` default 0.82 |
| V-04 | SAFETY_INSTRUCTION rec type | Final Review | `card_recommendation.go`: new type + patient_safety_instructions field on DecisionCard |
| V-06 | Stress hyperglycaemia MCU_GATE | Final Review | `templates/CT_ACUTE_ILLNESS.yaml` + cross-node differential |
| V-07 | HbA1c distortion templates | Final Review | `templates/CT_HBA1C_*.yaml` + KB-20 clinical-events query |
| V-08 | Hypoglycaemia fast-path | Final Review | `card_handlers.go`: POST /safety/hypoglycaemia-alert endpoint |
| N-01 | MCU_GATE hysteresis | Final Review | `mcu_gate_cache.go`: asymmetric upgrade/downgrade rules |
| N-02 | V-MCU integrator freeze contract | Final Review | `kb19_publisher.go`: MCU_GATE_CHANGED event payload |
| N-03 | Post-HALT re-entry protocol | Final Review | `card_lifecycle.go`: re_entry_protocol flag computation |
| A-01 | TreatmentPerturbation tracking | Addendum | `treatment_perturbation.go` + POST/GET endpoints + Redis cache |
| A-03 | V-MCU dose cooldown contract | Addendum | V-MCU action (no KB-23 code — spec reference only) |
| SA-01–06 | Three-channel safety architecture | Three-Channel | V-MCU action (KB-23 provides Channel A via MCU_GATE) |

### AMBER Findings (8 total — implement during build)

| ID | Finding | Source | Phase |
|----|---------|--------|-------|
| V-02 | Composite card temporal decay (PENDING_REAFFIRMATION) | Final Review | Phase 4 |
| V-03 | Regional language schema (text_local, locale_code) | Final Review | Phase 1 (schema) |
| V-05 | MEDICATION_REVIEW rec type | Final Review | Phase 2 |
| V-09 | Secondary differentials on DecisionCard | Final Review | Phase 3 |
| N-04 | SAFETY_INSTRUCTION authoring standard | Final Review | Template authoring |
| N-05 | dose_adjustment_notes contract (MODIFY requires non-null) | Final Review | Phase 2 |
| A-02 | Posterior confidence decay (Kalman-like) | Addendum | Phase 3 |
| A-04/A-05 | KB-21 adherence gain factor + observation reliability | Addendum | Phase 3 |

---

## Directory Structure

```
kb-23-decision-cards/
├── main.go                                # Application entry point
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── framework.yaml                         # KB framework metadata
├── README.md
├── templates/                             # Mounted volume; CardTemplate YAML definitions
│   ├── p01_chest_pain/
│   │   ├── CT_ACS_PROBABLE.yaml
│   │   ├── CT_ACS_FIRM.yaml
│   │   └── CT_STABLE_ANGINA.yaml
│   ├── p02_dyspnea/
│   │   ├── CT_HF_ACUTE.yaml
│   │   ├── CT_HF_CHRONIC.yaml
│   │   └── CT_PNEUMONIA.yaml
│   ├── p05_palpitations/
│   │   └── CT_ARRHYTHMIA.yaml
│   ├── cross_node/
│   │   ├── CT_ACUTE_ILLNESS_STRESS_HYPERGLYCAEMIA.yaml    # V-06
│   │   ├── CT_HBA1C_UNRELIABLE_HAEMOGLOBINOPATHY.yaml     # V-07
│   │   ├── CT_HBA1C_UNRELIABLE_RECENT_TRANSFUSION.yaml    # V-07
│   │   └── CT_HYPOGLYCAEMIA_DEVICE_DETECTED.yaml          # V-08
│   └── vocabulary/
│       └── dose_adjustment_tokens.yaml    # N-05: controlled vocabulary
├── migrations/
│   ├── 001_initial_schema.sql             # 7 tables + indexes
│   └── 002_treatment_perturbations.sql    # A-01: perturbation table
└── internal/
    ├── config/
    │   └── config.go                      # Environment-based configuration (Viper)
    ├── database/
    │   └── connection.go                  # GORM PostgreSQL + AutoMigrate
    ├── cache/
    │   └── redis.go                       # Redis client: MCU_GATE, perturbations, adherence
    ├── metrics/
    │   └── collector.go                   # Prometheus metrics (card generation, gate transitions)
    ├── models/
    │   ├── decision_card.go               # DecisionCard GORM model (core entity)
    │   ├── card_recommendation.go         # 9 recommendation types including SAFETY_INSTRUCTION
    │   ├── card_template.go               # CardTemplate (parsed from YAML)
    │   ├── summary_fragment.go            # SummaryFragment with locale support (V-03)
    │   ├── confidence_tier.go             # FIRM/PROBABLE/POSSIBLE/UNCERTAIN + thresholds
    │   ├── mcu_gate.go                    # MCU_GATE enum + gate history model
    │   ├── composite_card.go              # 72-hour card synthesis model
    │   ├── treatment_perturbation.go      # A-01: TreatmentPerturbation GORM model
    │   ├── hypoglycaemia_alert.go         # V-08: fast-path alert model
    │   └── events.go                      # KB-19 event types: MCU_GATE_CHANGED, SAFETY_ALERT, etc.
    ├── services/
    │   ├── template_loader.go             # YAML parse + validation + hot-reload
    │   │                                  # Validates: SAFETY_INSTRUCTION fields, dose_adjustment_notes vocab
    │   │                                  # N-04: patient_advocate_reviewed_by enforcement
    │   ├── template_selector.go           # Maps DifferentialSnapshot → CardTemplate(s)
    │   │                                  # V-09: secondary_differentials auto-include logic
    │   ├── confidence_tier_service.go     # V-01: per-template threshold computation
    │   │                                  # firm_posterior + firm_medication_change separation
    │   ├── card_builder.go                # Assembles DecisionCard from template + snapshot + context
    │   │                                  # N-05: dose_adjustment_notes required when mcu_gate=MODIFY
    │   │                                  # A-02: confidence_tier_decayed application
    │   │                                  # A-05: observation_reliability computation
    │   ├── recommendation_composer.go     # Generates typed recommendations per confidence tier
    │   │                                  # V-04: SAFETY_INSTRUCTION bypass logic
    │   │                                  # V-05: MEDICATION_REVIEW at PROBABLE tier
    │   ├── mcu_gate_manager.go            # Core MCU_GATE decision table evaluation
    │   │                                  # V-06: stress hyperglycaemia gate rules
    │   ├── mcu_gate_cache.go              # N-01: gate_history[] + asymmetric hysteresis
    │   │                                  # CheckHysteresis() for downgrade eligibility
    │   ├── card_lifecycle.go              # Card state machine: ACTIVE → SUPERSEDED → ARCHIVED
    │   │                                  # V-02: PENDING_REAFFIRMATION flagging (>48h PAUSE)
    │   │                                  # N-03: ComputeReentryRequired() on HALT lift
    │   ├── composite_card_service.go      # 72-hour synthesis with most-restrictive MCU_GATE
    │   │                                  # Recurrence detection (3+ occurrences → urgency upgrade)
    │   ├── hypoglycaemia_handler.go       # V-08: fast-path card generation (bypasses KB-22)
    │   │                                  # Severity routing: SEVERE→HALT, MODERATE→PAUSE, MILD→MODIFY
    │   ├── perturbation_service.go        # A-01: store, cache, query active perturbations
    │   │                                  # Redis cache with TTL = EffectWindowEnd
    │   ├── kb20_client.go                 # Fetch patient stratum, labs, clinical events (V-07)
    │   │                                  # GET /patient/:id/clinical-events (transfusion check)
    │   ├── kb21_client.go                 # A-04: adherence score by medication class
    │   │                                  # Cache adherence per patient (TTL: 6h)
    │   ├── behavioral_gap_handler.go       # KB-21 G-01: BEHAVIORAL_GAP/DISCORDANT card generation
    │   │                                  # POST /safety/behavioral-gap-alert handler
    │   │                                  # BEHAVIORAL_GAP → MODIFY gate + LIFESTYLE rec
    │   │                                  # DISCORDANT → SAFE gate + MEDICATION_REVIEW rec
    │   ├── kb19_publisher.go              # Publish events to KB-19 via POST /api/v1/events:
    │   │                                  # MCU_GATE_CHANGED (with re_entry fields: N-02, N-03)
    │   │                                  # SAFETY_ALERT, UNACKNOWLEDGED_URGENT_CARD
    │   │                                  # MCU_GATE_REAFFIRMATION_NEEDED (V-02)
    │   │                                  # NOTE: DATA_ANOMALY_DETECTED originates from V-MCU, not KB-23 (SA-05)
    │   └── fragment_loader.go             # SummaryFragment loader with locale fallback chain
    │                                      # N-04: reading_level_validated enforcement
    └── api/
        ├── server.go                      # Gin HTTP server + middleware + graceful shutdown
        ├── routes.go                      # Route registration
        ├── card_handlers.go               # POST /decision-cards (inbound from KB-22 push)
        │                                  # GET /cards/:id
        │                                  # V-08: POST /safety/hypoglycaemia-alert (fast-path)
        │                                  # KB-21 G-01: POST /safety/behavioral-gap-alert
        ├── patient_handlers.go            # GET /patients/:id/mcu-gate (enriched response for V-MCU)
        │                                  # GET /patients/:id/active-cards
        │                                  # A-04: adherence_gain_factor in response
        │                                  # A-05: observation_reliability in response
        ├── gate_handlers.go               # POST /cards/:id/mcu-gate-resume (clinician action)
        │                                  # V-02: PENDING_REAFFIRMATION handling
        ├── perturbation_handlers.go       # A-01: POST /perturbations (from V-MCU)
        │                                  # GET /perturbations/:patient_id/active (for KB-22)
        ├── infra_handlers.go              # GET /health, /readiness, /metrics
        │                                  # POST /internal/templates/reload (hot-reload)
        └── middleware.go                  # JWT auth, request logging, correlation IDs
```

---

## Data Models (7 Tables + 1 Addendum Table)

### 1. DecisionCard (decision_cards) — Core Entity

| Field | Type | Notes |
|-------|------|-------|
| card_id | UUID PK | |
| patient_id | UUID indexed | FK to KB-20 |
| session_id | UUID nullable indexed | FK to KB-22 (null for fast-path cards) |
| snapshot_id | UUID nullable | DifferentialSnapshot that triggered this card |
| template_id | string indexed | CT_ACS_PROBABLE etc. |
| node_id | string indexed | P01_CHEST_PAIN etc. |
| primary_differential_id | string | Top diagnosis from snapshot |
| primary_posterior | float64 | Posterior probability |
| diagnostic_confidence_tier | enum | FIRM / PROBABLE / POSSIBLE / UNCERTAIN |
| confidence_tier_decayed | bool | A-02: true when decay applied |
| confidence_tier_decay_reason | string nullable | A-02: e.g., 'PERTURBATION:INSULIN+2u' |
| mcu_gate | enum | SAFE / MODIFY / PAUSE / HALT |
| mcu_gate_rationale | text | Clinical rationale for gate |
| dose_adjustment_notes | text nullable | N-05: required when mcu_gate=MODIFY |
| observation_reliability | enum | HIGH / MODERATE / LOW (A-05) |
| secondary_differentials | JSONB nullable | V-09: array of secondary differential entries |
| clinician_summary | text | Structured clinician-facing summary |
| patient_summary_en | text | Patient-facing English summary |
| patient_summary_hi | text | Patient-facing Hindi summary |
| patient_summary_local | text nullable | V-03: regional language |
| patient_safety_instructions | JSONB nullable | V-04: SAFETY_INSTRUCTION entries |
| locale_code | string nullable | V-03: BCP-47 code |
| safety_tier | enum | IMMEDIATE / URGENT / ROUTINE |
| recurrence_count | int | 3+ triggers urgency upgrade |
| card_source | enum | KB22_SESSION / HYPOGLYCAEMIA_FAST_PATH / PERTURBATION_DECAY |
| status | enum | ACTIVE / SUPERSEDED / PENDING_REAFFIRMATION / ARCHIVED |
| pending_reaffirmation | bool | V-02: PAUSE > 48h without acknowledgement |
| re_entry_protocol | bool | N-03: true when HALT lifts after > 96h |
| created_at | timestamptz | |
| updated_at | timestamptz | |
| superseded_at | timestamptz nullable | |
| superseded_by | UUID nullable | |

### 2. CardRecommendation (card_recommendations)

| Field | Type | Notes |
|-------|------|-------|
| recommendation_id | UUID PK | |
| card_id | UUID FK indexed | |
| rec_type | enum | INVESTIGATION / REFERRAL / MONITORING / MEDICATION_HOLD / MEDICATION_MODIFY / MEDICATION_CONTINUE / LIFESTYLE / SAFETY_INSTRUCTION / MEDICATION_REVIEW |
| urgency | enum | IMMEDIATE / URGENT / ROUTINE / SCHEDULED |
| target | string | ECHO / TROPONIN / CARDIOLOGY / etc. |
| action_text_en | text | |
| action_text_hi | text | |
| rationale_en | text | |
| guideline_ref | string | ACC_AHA_ACS_2023_S2.1 etc. |
| confidence_tier_required | enum | Minimum tier for this rec type |
| bypasses_confidence_gate | bool | V-04: true for SAFETY_INSTRUCTION |
| trigger_condition_en | text nullable | V-04: SAFETY_INSTRUCTION only |
| trigger_condition_hi | text nullable | V-04: SAFETY_INSTRUCTION only |
| from_secondary_differential | bool | V-09: auto-included from secondary |
| conflict_flag | bool | V-09: medication conflict from secondary |
| sort_order | int | Display ordering |
| created_at | timestamptz | |

### 3. CardTemplate (card_templates) — In-memory from YAML, reference table for audit

| Field | Type | Notes |
|-------|------|-------|
| template_id | string PK | CT_ACS_PROBABLE etc. |
| node_id | string indexed | P01_CHEST_PAIN |
| differential_id | string | ACS_STEMI etc. |
| template_version | string | Semantic version |
| content_sha256 | string | Integrity hash |
| confidence_thresholds | JSONB | V-01: per-template override |
| mcu_gate_default | enum | Template-level gate default |
| recommendations_count | int | |
| has_safety_instructions | bool | V-04 |
| requires_dose_adjustment_notes | bool | N-05 |
| clinical_reviewer | string | Named reviewer |
| approved_at | timestamptz | |
| loaded_at | timestamptz | Last hot-reload |

### 4. SummaryFragment (summary_fragments) — In-memory from YAML

| Field | Type | Notes |
|-------|------|-------|
| fragment_id | string PK | |
| template_id | string FK | |
| fragment_type | enum | CLINICIAN / PATIENT / SAFETY_INSTRUCTION |
| text_en | text | |
| text_hi | text | |
| text_local | text nullable | V-03: regional language |
| locale_code | string nullable | V-03: BCP-47 |
| patient_advocate_reviewed_by | string nullable | N-04: required for SAFETY_INSTRUCTION |
| reading_level_validated | bool | N-04: required for SAFETY_INSTRUCTION |
| guideline_ref | string nullable | |
| version | string | |

### 5. MCUGateHistory (mcu_gate_history) — N-01 Hysteresis Tracking

| Field | Type | Notes |
|-------|------|-------|
| history_id | UUID PK | |
| patient_id | UUID indexed | |
| card_id | UUID FK | Card that triggered this gate |
| gate_value | enum | SAFE / MODIFY / PAUSE / HALT |
| previous_gate | enum nullable | Gate before this transition |
| session_id | UUID nullable | KB-22 session (null for fast-path) |
| transition_reason | text | |
| clinician_resume_by | string nullable | Named clinician for HALT/PAUSE resume |
| clinician_resume_reason | text nullable | |
| re_entry_protocol | bool | N-03 |
| halt_duration_hours | float64 nullable | N-03: for re-entry computation |
| acknowledged_at | timestamptz nullable | V-02: clinician acknowledgement |
| created_at | timestamptz indexed | |

### 6. CompositeCardSignal (composite_card_signals) — 72-hour Synthesis

| Field | Type | Notes |
|-------|------|-------|
| composite_id | UUID PK | |
| patient_id | UUID indexed | |
| card_ids | UUID[] | Contributing cards in 72h window |
| most_restrictive_gate | enum | Most restrictive MCU_GATE |
| recurrence_count | int | Symptom pattern count |
| urgency_upgraded | bool | 3+ recurrences |
| synthesis_summary_en | text | |
| synthesis_summary_hi | text | |
| window_start | timestamptz | |
| window_end | timestamptz | |
| created_at | timestamptz | |

### 7. HypoglycaemiaAlert (hypoglycaemia_alerts) — V-08 Fast-Path

| Field | Type | Notes |
|-------|------|-------|
| alert_id | UUID PK | |
| patient_id | UUID indexed | |
| source | enum | CGM / GLUCOMETER / VMCU_DETECTED / VMCU_PREDICTED / PATIENT_REPORT / KB21_BEHAVIORAL |
| glucose_mmol_l | float64 | |
| duration_minutes | int nullable | CGM only |
| severity | enum | MILD / MODERATE / SEVERE |
| predicted_at_hours | float64 nullable | B-05: for VMCU_PREDICTED source |
| halt_source | enum | MEASURED / PREDICTED (B-05) |
| generated_card_id | UUID nullable | FK to DecisionCard created |
| event_timestamp | timestamptz | |
| processed_at | timestamptz | |

### 8. TreatmentPerturbation (treatment_perturbations) — A-01

| Field | Type | Notes |
|-------|------|-------|
| perturbation_id | UUID PK | |
| patient_id | UUID indexed | |
| intervention_type | enum | INSULIN_INCREASE / INSULIN_DECREASE / DRUG_HOLD / DRUG_START / DOSE_ADJUST |
| dose_delta | float64 | e.g., +2.0 units |
| baseline_dose | float64 | |
| effect_window_start | timestamptz indexed | |
| effect_window_end | timestamptz indexed | |
| affected_observables | text[] | ["FBG", "PPBG", "HBA1C"] |
| stability_factor | float64 | LR dampening: 0.3–0.7 |
| created_at | timestamptz | |

---

## API Endpoints

### Card Generation & Management
| Method | Path | Description | Priority |
|--------|------|-------------|----------|
| POST | `/api/v1/decision-cards` | Receive HPI_COMPLETE from KB-22 and generate DecisionCard (KB-22 pushes here) | Phase 2 |
| GET | `/api/v1/cards/:id` | Get DecisionCard by ID | Phase 2 |
| GET | `/api/v1/patients/:id/active-cards` | Active cards for patient (with PENDING_REAFFIRMATION) | Phase 2 |
| POST | `/api/v1/cards/:id/mcu-gate-resume` | Clinician resumes PAUSE/HALT gate | Phase 4 |

### MCU Gate (V-MCU Integration — Channel A)
| Method | Path | Description | Priority |
|--------|------|-------------|----------|
| GET | `/api/v1/patients/:id/mcu-gate` | Enriched MCU_GATE response (< 5ms from Redis) | Phase 2 |

**Enriched response payload**:
```json
{
  "mcu_gate": "MODIFY",
  "dose_adjustment_notes": "HBA1C_CORRECTION: use fructosamine as titration signal",
  "adherence_gain_factor": 0.75,
  "adherence_score_source": "KB21_INSULIN_CLASS",
  "observation_reliability": "MODERATE",
  "active_perturbation_count": 1,
  "re_entry_protocol": false,
  "gate_card_id": "uuid"
}
```

### Safety Fast-Paths
| Method | Path | Description | Priority |
|--------|------|-------------|----------|
| POST | `/api/v1/safety/hypoglycaemia-alert` | V-08: Device/V-MCU/KB-21 hypoglycaemia fast-path | Phase 2 (minimal handler; hysteresis wrapping in Phase 3) |
| POST | `/api/v1/safety/behavioral-gap-alert` | KB-21 G-01: BEHAVIORAL_GAP/DISCORDANT alert from KB-21 | Phase 2 (minimal handler; KB-21 client already live) |

### Treatment Perturbation (A-01)
| Method | Path | Description | Priority |
|--------|------|-------------|----------|
| POST | `/api/v1/perturbations` | V-MCU publishes dose change + effect window | Phase 3 |
| GET | `/api/v1/perturbations/:patient_id/active` | KB-22 queries active dampening (< 3ms) | Phase 3 |

### Infrastructure
| Method | Path | Description | Priority |
|--------|------|-------------|----------|
| GET | `/health` | Health check | Phase 1 |
| GET | `/readiness` | Readiness probe | Phase 1 |
| GET | `/metrics` | Prometheus metrics | Phase 1 |
| POST | `/internal/templates/reload` | Hot-reload CardTemplate YAMLs | Phase 1 |

---

## Cross-KB Dependencies & Contracts

### KB-23 Consumes (Inbound)

| Source | Contract | Data | Phase |
|--------|----------|------|-------|
| KB-22 | KB-22 pushes HPI_COMPLETE to POST /api/v1/decision-cards | ranked_differentials[], safety_flags[], session_id | Phase 2 |
| KB-20 | GET /patient/:id/summary-context | stratum, medications, labs, eGFR | Phase 2 |
| KB-20 | GET /patient/:id/clinical-events | Transfusion events (V-07), weight, vitals | Phase 3 |
| KB-21 | GET /patient/:id/adherence (by medication class) | adherence_score, insulin_adherence | Phase 3 |
| V-MCU | POST /perturbations | TreatmentPerturbation record | Phase 3 |
| V-MCU/CGM/KB-20 | POST /safety/hypoglycaemia-alert | Glucose, severity, source, duration | Phase 2 (minimal handler) |
| KB-21 | POST /safety/behavioral-gap-alert (G-01) | TreatmentResponseClass, adherence, HbA1c delta | Phase 2 (KB-21 client already live) |
| KB-21 | POST /safety/hypoglycaemia-alert with source=KB21_BEHAVIORAL (G-03) | Behavioral hypo risk | Phase 2 (KB-21 client already live) |

### KB-23 Publishes (Outbound)

| Target | Contract | Data | Phase |
|--------|----------|------|-------|
| KB-19 | POST /api/v1/events: MCU_GATE_CHANGED | gate, re_entry_protocol, halt_duration_hours, re_entry_phase1/2_hours | Phase 2 |
| KB-19 | POST /api/v1/events: SAFETY_ALERT | IMMEDIATE safety flags (< 2s) | Phase 2 |
| KB-19 | POST /api/v1/events: UNACKNOWLEDGED_URGENT_CARD | Cards not acknowledged within timeout | Phase 4 |
| KB-19 | POST /api/v1/events: MCU_GATE_REAFFIRMATION_NEEDED | V-02: PAUSE > 48h | Phase 4 |
| V-MCU | GET /patients/:id/mcu-gate (Redis-cached, < 5ms) | Enriched gate response — V-MCU reads on every titration cycle. No Kafka needed; <30s propagation via Redis cache invalidation + rewrite | Phase 2 |
| KB-22 | GET /perturbations/:patient_id/active | Active perturbation windows + stability_factor | Phase 3 |

---

## Implementation Phases

### Phase 1 — Foundation & Schema (Week 1–2)

**Goal**: Service skeleton, database, configuration, template loading infrastructure.

| Task | Files | Finding |
|------|-------|---------|
| Scaffold Go module with Gin, GORM, Redis, Zap, Prometheus | `main.go`, `go.mod` | — |
| Config loader (Viper) with all env vars | `internal/config/config.go` | — |
| PostgreSQL connection + AutoMigrate for all 8 tables | `internal/database/connection.go` | — |
| Redis client with JSON serialization | `internal/cache/redis.go` | — |
| Prometheus metrics collector | `internal/metrics/collector.go` | — |
| All GORM models (8 tables) with correct types + indexes | `internal/models/*.go` | All |
| Confidence tier enum with threshold model | `internal/models/confidence_tier.go` | V-01 |
| 9 recommendation types enum (including SAFETY_INSTRUCTION, MEDICATION_REVIEW) | `internal/models/card_recommendation.go` | V-04, V-05 |
| SummaryFragment model with locale fields | `internal/models/summary_fragment.go` | V-03 |
| DecisionCard model with ALL fields (incl. A-02, A-05 additions) | `internal/models/decision_card.go` | V-03, V-04, N-05, A-02, A-05 |
| TreatmentPerturbation model | `internal/models/treatment_perturbation.go` | A-01 |
| HypoglycaemiaAlert model | `internal/models/hypoglycaemia_alert.go` | V-08 |
| MCUGateHistory model | `internal/models/mcu_gate.go` | N-01 |
| SQL migrations (001 + 002) | `migrations/*.sql` | — |
| TemplateLoader: YAML parsing, schema validation, hot-reload | `internal/services/template_loader.go` | N-04 |
| TemplateLoader validation: SAFETY_INSTRUCTION requires trigger_condition + action_text | `internal/services/template_loader.go` | V-04 |
| TemplateLoader validation: dose_adjustment_notes against controlled vocabulary | `internal/services/template_loader.go` | N-05 |
| FragmentLoader with locale fallback chain (text_local → text_hi → text_en) | `internal/services/fragment_loader.go` | V-03 |
| Health, readiness, metrics endpoints | `internal/api/infra_handlers.go` | — |
| Gin server + middleware + graceful shutdown | `internal/api/server.go` | — |
| Dockerfile + docker-compose.yml | root | — |

**Deliverable**: Service starts, connects to DB/Redis, loads templates, passes health check.

### Phase 2 — Core Card Generation & MCU Gate (Week 3–5)

**Goal**: End-to-end card generation from KB-22 snapshot to V-MCU enriched gate response.

| Task | Files | Finding |
|------|-------|---------|
| ConfidenceTierService: compute tier from posterior + per-template thresholds | `internal/services/confidence_tier_service.go` | V-01 |
| ConfidenceTierService: separate firm_medication_change threshold (default 0.82) | `internal/services/confidence_tier_service.go` | V-01 |
| TemplateSelector: map DifferentialSnapshot → best-matching CardTemplate | `internal/services/template_selector.go` | — |
| MCUGateManager: evaluate MCU_GATE decision table from template + differential | `internal/services/mcu_gate_manager.go` | — |
| MCUGateManager: stress hyperglycaemia rows (V-06) | `internal/services/mcu_gate_manager.go` | V-06 |
| RecommendationComposer: generate typed recommendations per confidence tier | `internal/services/recommendation_composer.go` | — |
| RecommendationComposer: SAFETY_INSTRUCTION bypass + patient_safety_instructions field | `internal/services/recommendation_composer.go` | V-04 |
| RecommendationComposer: MEDICATION_REVIEW permitted at PROBABLE tier | `internal/services/recommendation_composer.go` | V-05 |
| RecommendationComposer: firm_medication_change gate for MEDICATION_HOLD/MODIFY | `internal/services/recommendation_composer.go` | V-01 |
| CardBuilder: assemble DecisionCard from template + snapshot + context + recommendations | `internal/services/card_builder.go` | — |
| CardBuilder: dose_adjustment_notes required when mcu_gate=MODIFY | `internal/services/card_builder.go` | N-05 |
| KB20Client: fetch patient summary-context (stratum, labs, medications) | `internal/services/kb20_client.go` | — |
| KB19Publisher: publish MCU_GATE_CHANGED with re_entry fields | `internal/services/kb19_publisher.go` | N-02, N-03 |
| KB19Publisher: publish SAFETY_ALERT for IMMEDIATE safety flags (< 2s SLA) | `internal/services/kb19_publisher.go` | — |
| MCUGateCache: Redis cache for per-patient MCU_GATE | `internal/services/mcu_gate_cache.go` | — |
| POST /decision-cards handler (inbound from KB-22 HPI_COMPLETE push) | `internal/api/card_handlers.go` | — |
| GET /cards/:id handler | `internal/api/card_handlers.go` | — |
| GET /patients/:id/active-cards handler | `internal/api/patient_handlers.go` | — |
| GET /patients/:id/mcu-gate handler (enriched response, < 5ms from Redis) | `internal/api/patient_handlers.go` | — |
| HypoglycaemiaHandler (minimal): POST /safety/hypoglycaemia-alert → gate write (SEVERE→HALT, MODERATE→PAUSE, MILD→MODIFY) | `internal/services/hypoglycaemia_handler.go` | V-08 |
| POST /safety/hypoglycaemia-alert endpoint (V-MCU, CGM, KB-21 sources) | `internal/api/card_handlers.go` | V-08 |
| BehavioralGapHandler (minimal): POST /safety/behavioral-gap-alert → gate write (BEHAVIORAL_GAP→MODIFY, DISCORDANT→SAFE) | `internal/services/behavioral_gap_handler.go` | KB-21 G-01 |
| POST /safety/behavioral-gap-alert endpoint | `internal/api/card_handlers.go` | KB-21 G-01 |

**Deliverable**: KB-22 snapshot → DecisionCard → MCU_GATE cached in Redis → V-MCU reads enriched response. KB-21 safety fast-path endpoints live (closes G-01/G-03 gaps on deploy).

### Phase 3 — Hysteresis, Perturbation Tracking & Full Enrichment (Week 6–8)

**Goal**: Hysteresis prevents gate oscillation. Perturbation tracking prevents observation bias. Safety handlers enriched with full template matching and source-aware logic.

| Task | Files | Finding |
|------|-------|---------|
| MCUGateCache: gate_history[] tracking per patient | `internal/services/mcu_gate_cache.go` | N-01 |
| MCUGateCache: asymmetric hysteresis (upgrades immediate, downgrades require 2+ sessions / 72h) | `internal/services/mcu_gate_cache.go` | N-01 |
| HypoglycaemiaHandler: add hysteresis-aware downgrade logic | `internal/services/hypoglycaemia_handler.go` | N-01, V-08 |
| HypoglycaemiaHandler: VMCU_PREDICTED source support (B-05) | `internal/services/hypoglycaemia_handler.go` | B-05 |
| HypoglycaemiaHandler: CT_HYPOGLYCAEMIA_DEVICE_DETECTED template matching | `templates/cross_node/` | V-08 |
| BehavioralGapHandler: full LIFESTYLE + MEDICATION_REVIEW recommendation composition | `internal/services/behavioral_gap_handler.go` | KB-21 G-01 |
| PerturbationService: store TreatmentPerturbation in DB | `internal/services/perturbation_service.go` | A-01 |
| PerturbationService: Redis cache active perturbations (TTL = EffectWindowEnd) | `internal/services/perturbation_service.go` | A-01 |
| PerturbationService: GET /perturbations/:patient_id/active (< 3ms) | `internal/services/perturbation_service.go` | A-01 |
| POST /perturbations endpoint | `internal/api/perturbation_handlers.go` | A-01 |
| GET /perturbations/:patient_id/active endpoint | `internal/api/perturbation_handlers.go` | A-01 |
| CardBuilder: confidence_tier_decayed on perturbation receipt for FIRM/PROBABLE cards | `internal/services/card_builder.go` | A-02 |
| KB21Client: fetch adherence score by medication class | `internal/services/kb21_client.go` | A-04 |
| KB21Client: cache adherence per patient (TTL: 6h) | `internal/services/kb21_client.go` | A-04 |
| Patient handler: adherence_gain_factor in enriched mcu-gate response | `internal/api/patient_handlers.go` | A-04 |
| Patient handler: observation_reliability in enriched mcu-gate response | `internal/api/patient_handlers.go` | A-05 |
| KB20Client: GET /patient/:id/clinical-events (transfusion, weight) | `internal/services/kb20_client.go` | V-07 |
| TemplateSelector: secondary_differentials[] auto-include (INVESTIGATION/MONITORING only) | `internal/services/template_selector.go` | V-09 |
| TemplateSelector: MCU_GATE from secondary takes most-restrictive | `internal/services/template_selector.go` | V-09 |
| CardBuilder: CONFLICT_FLAG for medication changes from secondary differential | `internal/services/card_builder.go` | V-09 |

**Deliverable**: All three input channels operational. Hysteresis prevents gate oscillation. Perturbation dampening prevents observation bias. V-MCU receives complete enriched gate.

### Phase 4 — Card Lifecycle, Composite Synthesis & Re-Entry (Week 9–10)

**Goal**: Full card lifecycle management, 72-hour composite synthesis, clinician gate management.

| Task | Files | Finding |
|------|-------|---------|
| CardLifecycle: ACTIVE → SUPERSEDED → ARCHIVED state machine | `internal/services/card_lifecycle.go` | — |
| CardLifecycle: PENDING_REAFFIRMATION when PAUSE > 48h | `internal/services/card_lifecycle.go` | V-02 |
| CardLifecycle: ComputeReentryRequired() when HALT lifts after > 96h | `internal/services/card_lifecycle.go` | N-03 |
| CardLifecycle: emit MCU_GATE_REAFFIRMATION_NEEDED to KB-19 | `internal/services/card_lifecycle.go` | V-02 |
| POST /cards/:id/mcu-gate-resume: clinician gate resume with named reason + audit | `internal/api/gate_handlers.go` | V-02, N-01 |
| POST /cards/:id/mcu-gate-resume: HALT resume requires named clinician + reason | `internal/api/gate_handlers.go` | V-02 |
| CompositeCardService: 72-hour window synthesis | `internal/services/composite_card_service.go` | — |
| CompositeCardService: most-restrictive MCU_GATE across window | `internal/services/composite_card_service.go` | — |
| CompositeCardService: recurrence detection (3+ → urgency upgrade) | `internal/services/composite_card_service.go` | — |
| KB19Publisher: MCU_GATE_CHANGED with re_entry_protocol fields | `internal/services/kb19_publisher.go` | N-03 |
| KB19Publisher: UNACKNOWLEDGED_URGENT_CARD event | `internal/services/kb19_publisher.go` | — |
| Background job: scan for PAUSE > 48h without acknowledgement | `internal/services/card_lifecycle.go` | V-02 |

**Deliverable**: Full card lifecycle with clinician interaction. Composite synthesis prevents conflicting signals. Re-entry protocol signals V-MCU correctly.

### Phase 5 — Clinical Template Authoring & Integration Testing (Week 11–12)

**Goal**: All 11 CardTemplates authored and validated. End-to-end integration tests.

| Task | Files | Finding |
|------|-------|---------|
| Author CT_ACS_PROBABLE, CT_ACS_FIRM templates (P1) | `templates/p01_chest_pain/` | — |
| Author CT_HF_ACUTE, CT_HF_CHRONIC templates (P2) | `templates/p02_dyspnea/` | — |
| Author CT_ACUTE_ILLNESS_STRESS_HYPERGLYCAEMIA (cross-node) | `templates/cross_node/` | V-06 |
| Author CT_HBA1C_UNRELIABLE_HAEMOGLOBINOPATHY | `templates/cross_node/` | V-07 |
| Author CT_HBA1C_UNRELIABLE_RECENT_TRANSFUSION | `templates/cross_node/` | V-07 |
| Author CT_HYPOGLYCAEMIA_DEVICE_DETECTED (device source variants) | `templates/cross_node/` | V-08 |
| Define dose_adjustment_notes controlled vocabulary | `templates/vocabulary/` | N-05 |
| Author SAFETY_INSTRUCTION fragments for P1, P2 nodes | `templates/` | V-04, N-04 |
| Confidence threshold values per node: P1 (0.65), P2 (0.75/0.82), P5 (0.88) | `templates/` | V-01 |
| Integration test: KB-22 snapshot → card generation → MCU_GATE → V-MCU query | `tests/integration/` | — |
| Integration test: hypoglycaemia fast-path → HALT card → KB-19 event | `tests/integration/` | V-08 |
| Integration test: perturbation → dampening → KB-22 query | `tests/integration/` | A-01 |
| Integration test: hysteresis prevents PAUSE→SAFE→PAUSE oscillation | `tests/integration/` | N-01 |
| Integration test: HALT > 96h → re_entry_protocol=true in event | `tests/integration/` | N-03 |
| Load test: GET /patients/:id/mcu-gate < 5ms P95 | `tests/performance/` | — |
| Load test: GET /perturbations/:patient_id/active < 3ms P95 | `tests/performance/` | A-01 |

**Deliverable**: All templates validated. End-to-end tests green. Performance targets met.

### Phase 6 — Hardening, Monitoring & Documentation (Week 13–14)

**Goal**: Production-ready with full observability, error handling, and documentation.

| Task | Files |
|------|-------|
| Prometheus dashboards: card generation rate, gate transitions, fast-path alerts | `monitoring/` |
| Structured logging with correlation IDs across all handlers | All |
| Error handling: KB-20/KB-21/KB-22 unavailability graceful degradation | All clients |
| Redis failover: fallback to DB for MCU_GATE if Redis unavailable | `internal/cache/redis.go` |
| Rate limiting on POST endpoints | `internal/api/middleware.go` |
| OpenAPI specification for all endpoints | `docs/openapi.yaml` |
| README with startup, configuration, and API guide | `README.md` |
| Docker health check integration | `Dockerfile` |

---

## Configuration (Environment Variables)

```bash
# Service
PORT=8134
ENVIRONMENT=development
LOG_LEVEL=info

# Database
DATABASE_URL=postgresql://kb23_user:kb_password@localhost:5438/kb23_decision_cards
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10

# Redis
REDIS_URL=redis://localhost:6387/0
REDIS_MCU_GATE_TTL=3600         # 1 hour default, refreshed on gate change
REDIS_PERTURBATION_TTL=0        # Dynamic: EffectWindowEnd per record
REDIS_ADHERENCE_TTL=21600       # 6 hours

# Template Loading
TEMPLATES_DIR=/app/templates
TEMPLATES_HOT_RELOAD=true

# Cross-KB Service URLs
KB19_URL=http://localhost:8099   # Protocol Orchestrator (port confirmed from KB-19 config.go)
KB20_URL=http://localhost:8131   # Patient Profile
KB21_URL=http://localhost:8133   # Behavioural Intelligence
KB22_URL=http://localhost:8132   # HPI Engine

# Confidence Thresholds (defaults — overridden per template)
DEFAULT_FIRM_POSTERIOR=0.75
DEFAULT_FIRM_MEDICATION_CHANGE=0.82
DEFAULT_PROBABLE_POSTERIOR=0.60
DEFAULT_POSSIBLE_POSTERIOR=0.40

# Safety
HYPOGLYCAEMIA_SEVERE_THRESHOLD=3.0      # mmol/L
HYPOGLYCAEMIA_MODERATE_THRESHOLD=3.9    # mmol/L
SAFETY_ALERT_PUBLISH_TIMEOUT=2s         # KB-19 SAFETY_ALERT < 2s SLA

# Monitoring
METRICS_ENABLED=true
```

---

## Redis Cache Keys

| Key Pattern | Value | TTL | Purpose |
|-------------|-------|-----|---------|
| `mcu_gate:{patient_id}` | Enriched MCU_GATE response JSON | 1h (refreshed on change) | V-MCU reads < 5ms |
| `gate_history:{patient_id}` | Array of {gate, timestamp, session_id} | 30d | N-01: hysteresis evaluation |
| `perturbation:active:{patient_id}` | Array of active TreatmentPerturbation | Dynamic (EffectWindowEnd) | KB-22 reads < 3ms |
| `adherence:{patient_id}` | {insulin_adherence, gain_factor, source} | 6h | A-04: enriched response |
| `template:{template_id}` | Parsed CardTemplate | Until reload | Template cache |

---

## Critical Path & Pre-Implementation Gate Checklist

### Blocking Dependencies (Must Be Resolved Before Phase Start)

| # | Gate Item | Owner | Blocks | Status |
|---|-----------|-------|--------|--------|
| 1 | V-MCU team briefed on instability analysis (Section 5 of Final Review) and all 11 design commitments | V-MCU team | V-MCU design | PENDING |
| 2 | KB-19 extends event_handlers.go switch statement to accept MCU_GATE_CHANGED (currently only HPI_COMPLETE + SAFETY_ALERT). HPIEvent struct needs gate fields (gate, re_entry_protocol, halt_duration_hours) | KB-19 team | Phase 2 | PENDING |
| 2b | KB-22 docker-compose.yml port fixes: KB19_URL 8129→8099, KB20_URL 8130→8131, KB23_URL 8133→8134 (pre-existing bugs, not caused by KB-23) | KB-22 team | Phase 1 | PENDING |
| 3 | KB-20 confirms GET /patient/:id/clinical-events exposes TRANSFUSION events | KB-20 team | Phase 3 | PENDING |
| 4 | KB-21 confirms adherence-by-medication-class endpoint | KB-21 team | Phase 3 | PENDING |
| 5 | Clinical team delivers minimum 11 CardTemplate YAMLs with V-06, V-07, V-08 additions | Clinical team | Phase 5 | PENDING |
| 6 | SAFETY_INSTRUCTION fragments reviewed by patient advocate + Hindi non-clinician reviewer | Clinical + advocate | Phase 5 | PENDING |
| 7 | dose_adjustment_notes controlled vocabulary finalised (clinical + engineering) | Clinical + KB-23 | Phase 2 | PENDING |
| 8 | confidence_thresholds per-node values agreed: P1(0.65), P2(0.75/0.82), P5(0.88) | Clinical + KB-23 | Phase 2 | PENDING |
| 9 | V-MCU confirms POST /safety/hypoglycaemia-alert as valid input source | V-MCU team | Phase 2 | PENDING |

### Parallel Work Streams

```
KB-23 Phase 1-4 ────────────────────────────────────► (no V-MCU dependency)
Clinical Template Authoring ─────────────────────────► (parallel, feeds Phase 5)
V-MCU Pre-Implementation Spec ──────────────────────► (parallel, consumes KB-23 contracts)
KB-22 Perturbation Integration ─────────────────────► (depends on KB-23 Phase 3)
```

KB-23 implementation can proceed through Phase 4 with NO dependency on V-MCU. V-MCU spec can be written in parallel consuming the contracts defined here.

---

## V-MCU Design Commitments Reference (All 11)

KB-23 specifies contracts that V-MCU must honour. These are NOT KB-23 code — they are V-MCU pre-implementation requirements traced here for completeness.

| # | Commitment | Source |
|---|-----------|--------|
| 1 | Integrator freeze() / resume() — freeze on PAUSE/HALT, resume from frozen value not zero | Final Review N-02 |
| 2 | Rate limiter post-resume — 50% max delta for ceil(pause_hours/24) cycles | Final Review N-02 |
| 3 | 3-phase re-entry protocol on re_entry_protocol=true | Final Review N-03 |
| 4 | MCU_GATE subscription via KB-19 events. No polling. < 30s propagation | Final Review N-02 |
| 5 | No autonomous gate override — cannot override PAUSE/HALT from internal readings | Final Review S5 |
| 6 | Dose cooldown — 48h basal / 6h rapid-acting minimum between changes | Addendum A-03 |
| 7 | Control gain modulation — gain_factor = f(adherence) from KB-23 enriched response | Addendum A-04 |
| 8 | MetabolicPhysiologyEngine (KB-24) internal module — MetabolicState, ISF, mechanism, dawn, predictive hypo | Addendum B-01–B-05 |
| 9 | Safety Arbiter — 1oo3 veto, synchronous, internal, before every dose output | Three-Channel SA-01 |
| 10 | PhysiologySafetyMonitor — separate from MetabolicPhysiologyEngine. Raw inputs only. Build-time import constraint | Three-Channel SA-02 |
| 11 | ProtocolGuard — pre-compiled rules from protocol_rules.yaml. No runtime network calls | Three-Channel SA-03 |

---

## Architectural Decisions Log

### AD-01: KB-22 → KB-23 Data Flow is Push, Not Pull

KB-22's `outcome_publisher.go` already implements `POST /api/v1/decision-cards` to KB-23 on session completion. KB-23 does NOT reach back into KB-22. There is no `kb22_client.go`. The inbound reception is handled by `card_handlers.go`.

**Evidence**: [outcome_publisher.go:67-68](backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/outcome_publisher.go#L67-L68)

### AD-02: V-MCU Gate Transport — Redis Cache, Not Kafka

V-MCU reads the enriched gate from KB-23's Redis cache via `GET /patients/:id/mcu-gate` on every titration cycle. KB-23 invalidates and rewrites the cache on every gate change. The <30s propagation commitment (V-MCU Design Commitment #4) is met by cache write latency (~1ms). No Kafka topic is needed for this path.

KB-23 separately publishes MCU_GATE_CHANGED to KB-19 via `POST /api/v1/events` for protocol arbitration and audit. KB-19 does NOT forward events to V-MCU.

**Evidence**: KB-19 has `POST /api/v1/events` ([event_handlers.go:57](backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/internal/api/event_handlers.go#L57)), but only for logging and arbitration triggering — not for relay to V-MCU.

### AD-03: KB-21 Behavioral Gap Fast-Path (from KB-21 G-01/G-03)

KB-21's `safety_client.go` already calls two KB-23 endpoints:
1. `POST /safety/behavioral-gap-alert` — for BEHAVIORAL_GAP and DISCORDANT treatment response classifications
2. `POST /safety/hypoglycaemia-alert` with `source=KB21_BEHAVIORAL` — for behavioral hypo risk

These contracts were defined in the KB-21 implementation (findings G-01 and G-03), not in the three KB-23 specification documents. They are verified in existing KB-21 code and must be honoured. Because KB-21's `safety_client.go` is already live and calling these endpoints, both handlers are scheduled as **Phase 2 minimal handlers** — the moment KB-23 deploys, KB-21's G-01 and G-03 safety gaps close automatically.

**Evidence**: [safety_client.go:53-83](backend/shared-infrastructure/knowledge-base-services/kb-21-behavioral-intelligence/internal/services/safety_client.go#L53-L83)

### AD-04: DATA_ANOMALY_DETECTED Originates from V-MCU, Not KB-23

SA-05 in the Three-Channel Safety Architecture specifies that DATA_ANOMALY_DETECTED events originate from V-MCU's PhysiologySafetyMonitor (Channel B) when it issues HOLD_DATA. V-MCU sends the re-validation request to KB-20 and notifies KB-19. KB-23 is not in that path.

### AD-05: Port Discrepancies — KB-22 Docker Compose

KB-22's `docker-compose.yml` has **three** incorrect upstream port references:

| Variable | docker-compose value | Actual port | Source |
|----------|---------------------|-------------|--------|
| KB19_URL | 8129 | **8099** | [KB-19 config.go:136](backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/internal/config/config.go#L136) |
| KB20_URL | 8130 | **8131** | [KB-20 config.go:66](backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/config/config.go#L66) |
| KB23_URL | 8133 | **8134** | KB-23 Implementation Plan (this document) |

KB-22's `config.go` defaults also differ (KB19_URL defaults to 8129 in config.go:70). These are **pre-existing bugs** in KB-22, not caused by KB-23. They should be fixed as a separate KB-22 maintenance task (Gate Checklist item 2b). KB-23 declares port 8134 as authoritative.

**Evidence**: [KB-22 docker-compose.yml:85-90](backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/docker-compose.yml#L85-L90)

### AD-06: KB-19 Event Handler Requires Code Extension for MCU_GATE_CHANGED

KB-19's `event_handlers.go` currently accepts only `HPI_COMPLETE` and `SAFETY_ALERT` event types (hard switch at line 71). Any other `event_type` returns HTTP 400 `unknown_event_type`. The `HPIEvent` struct also lacks MCU gate fields.

KB-23 publishes `MCU_GATE_CHANGED` events to KB-19 `POST /api/v1/events` in Phase 2. This requires the KB-19 team to:
1. Add `"MCU_GATE_CHANGED"` case to the event type switch
2. Extend `HPIEvent` struct with gate fields: `gate`, `re_entry_protocol`, `halt_duration_hours`, `re_entry_phase1_hours`, `re_entry_phase2_hours`

This is a **blocking dependency** for KB-23 Phase 2 (Gate Checklist item 2).

**Evidence**: [event_handlers.go:71-93](backend/shared-infrastructure/knowledge-base-services/kb-19-protocol-orchestrator/internal/api/event_handlers.go#L71-L93)

---

## Final Scorecard

| Metric | Value |
|--------|-------|
| Total findings (RED + AMBER) | 19 (11R + 8A) |
| Recommendation types | 9 |
| Fast-path endpoints | 3 (hypoglycaemia-alert + behavioral-gap-alert + perturbations) |
| CardTemplates required | 11 (minimum) |
| GORM tables | 8 |
| Redis cache keys | 5 patterns |
| API endpoints | 13 |
| Cross-KB dependencies | 6 (KB-19, KB-20, KB-21, KB-22, V-MCU, CGM) |
| V-MCU design commitments | 11 |
| Implementation phases | 6 (14 weeks) |
| Port | 8134 |
| PostgreSQL port | 5438 |
| Redis DB | 6387 |
