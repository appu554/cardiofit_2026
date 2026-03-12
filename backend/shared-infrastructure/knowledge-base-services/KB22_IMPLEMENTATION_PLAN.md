# KB-22 HPI Engine — Implementation Plan

**Service**: History of Present Illness Engine
**Technology**: Go 1.22 / Gin / GORM / Redis
**Port**: 8132 | **PostgreSQL**: 5437 | **Redis**: 6386
**Pattern Source**: KB-5 Drug Interactions (adapted)
**Specifications**: KB-22 Pre-Implementation Specification + Final Pre-Implementation Review

---

## Architecture Overview

KB-22 is the **reasoning core** of the Vaidshala clinical engine. It receives a patient presentation, conducts an entropy-maximising sequential interview across P1-P26 HPI nodes, and produces a calibrated differential diagnosis with safety flags.

```
                        ┌─────────────────────────────────────────┐
                        │              KB-22 HPI Engine           │
                        │                                         │
  KB-20 (stratum) ─────►│  ┌──────────────┐  ┌───────────────┐   │───► KB-23 (Decision Cards)
  KB-21 (adherence) ───►│  │   Bayesian    │  │    Safety     │   │───► KB-19 (Protocol Orch.)
  KB-21 (reliability) ─►│  │   Engine      │  │    Engine     │   │
  KB-1  (guidelines) ──►│  │  (goroutine)  │  │  (goroutine)  │   │
                        │  └──────┬───────┘  └───────┬───────┘   │
                        │         │   answer_event    │           │
                        │         └───────┬───────────┘           │
                        │                 │                       │
                        │  ┌──────────────▼──────────────────┐   │
                        │  │    Question Orchestrator         │   │───► KB-21 (telemetry write)
                        │  │    (entropy maximisation)        │   │
                        │  └─────────────────────────────────┘   │
                        └─────────────────────────────────────────┘
```

**Unique characteristic**: Bayesian Engine and Safety Engine run on **separate goroutines** with independent `recover()` blocks. A panic in inference cannot silence safety alerts.

---

## Findings Register (All RED + AMBER)

### RED Findings (8 total — architectural constraints, non-retrofittable)

| ID | Finding | Implementation Impact |
|----|---------|----------------------|
| F-01 | Log-odds CM composition (additive, not multiplicative) | `bayesian_engine.go`: `log_odds_state map[string]float64` as internal state |
| F-02 | Safety Engine on parallel goroutine with own `recover()` | `safety_engine.go`: independent goroutine, buffered channel fan-out |
| F-03 | Adherence-adjusted CM scaling from KB-21 | `cm_applicator.go`: `adjusted_mag = base_mag * min(1.0, adherence/0.70)` |
| F-04 | Pata-nahi neutral update (log-odds delta = 0.0) | `bayesian_engine.go`: PATA_NAHI answer applies 0.0, tracked separately |
| F-05 | Multi-stratum node support with per-stratum priors | `node.go` + `stratum_resolver.go`: per-stratum prior tables in YAML |
| R-01 | Dual-criterion termination (posterior + gap) | `bayesian_engine.go`: `convergence_logic: BOTH\|EITHER\|POSTERIOR_ONLY` |
| R-02 | Symptom cluster dampening (correlated evidence) | `bayesian_engine.go`: `cluster_answered` map, dampening factor |
| R-03 | Answer reliability weighting from KB-21 | `session_context_provider.go`: `LR_effective = LR ^ reliability_modifier` |

### AMBER Findings (4 total — implement during build)

| ID | Finding | Phase |
|----|---------|-------|
| F-06 | Stratum-specific calibration isolation | Phase 5 |
| F-07 | Cross-node safety trigger registry | Phase 6 |
| F-08 | Session resumability (suspend/resume) | Phase 4 |
| R-07 | LR source provenance in node YAML | Phase 1 (schema) |

### NEW Findings from Final Review (4 total)

| ID | Finding | Severity | Phase |
|----|---------|----------|-------|
| N-01 | KB-1 guideline prior injection | RED | Phase 3 |
| N-02 | KB-9 medication safety gate on safety flag | RED | Phase 5 |
| N-03 | India-specific population LR gap (clinical authoring) | AMBER | Pre-YAML |
| N-04 | KB-21 answer-reliability API contract (cross-KB) | AMBER | KB-21 dependency |

---

## Directory Structure

```
kb-22-hpi-engine/
├── main.go
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── framework.yaml
├── README.md
├── nodes/                              # Mounted volume; P1-P26 YAML node definitions
│   ├── p01_chest_pain.yaml
│   ├── p02_dyspnea.yaml
│   ├── cross_node_triggers.yaml        # F-07: global safety triggers
│   └── p03...p26.yaml
├── migrations/
│   └── 001_initial_schema.sql          # 6 tables + indexes
└── internal/
    ├── config/
    │   └── config.go                   # Environment-based configuration
    ├── database/
    │   └── connection.go               # GORM PostgreSQL + AutoMigrate
    ├── cache/
    │   └── redis.go                    # Redis client with JSON serialization
    ├── metrics/
    │   └── collector.go                # Prometheus metrics
    ├── models/
    │   ├── session.go                  # HPISession + SessionStatus enum
    │   ├── answer.go                   # SessionAnswer (append-only log)
    │   ├── differential.go             # DifferentialEntry, DifferentialSnapshot
    │   ├── safety.go                   # SafetyFlag, SafetyLevel
    │   ├── node.go                     # NodeDefinition (parsed from YAML)
    │   ├── calibration.go              # CalibrationRecord, AdjudicationFeedback
    │   └── events.go                   # HPI_COMPLETE, SAFETY_ALERT, QUESTION_ANSWERED
    ├── services/
    │   ├── session_service.go          # Session CRUD + state machine
    │   ├── bayesian_engine.go          # F-01: log-odds + CM composition + LR accumulation
    │   │                               # R-01: dual-criterion termination
    │   │                               # R-02: cluster dampening
    │   │                               # R-03: reliability weighting
    │   ├── safety_engine.go            # F-02: parallel goroutine + trigger evaluation
    │   ├── question_orchestrator.go    # A01: entropy maximisation + F-04 pata-nahi
    │   │                               # R-05: minimum inclusion guard
    │   ├── node_loader.go              # YAML parse + validation + hot-reload
    │   │                               # R-06: BOOLEAN/COMPOSITE_SCORE type validation
    │   │                               # R-07: lr_source/population_reference warnings
    │   ├── session_context_provider.go # 3-goroutine parallel fetch:
    │   │                               #   KB-20 stratum + KB-21 adherence + KB-21 reliability
    │   ├── cm_applicator.go            # F-01 + F-03: log-delta summation with adherence
    │   ├── guideline_client.go         # N-01: KB-1 prior adjustments + management summaries
    │   ├── medication_safety_provider.go # N-02: KB-9 contraindication check on safety flag
    │   ├── cross_node_safety.go        # F-07: global cross-node trigger registry
    │   ├── telemetry_writer.go         # Async POST to KB-21 /question-telemetry
    │   ├── outcome_publisher.go        # HPI_COMPLETE to KB-23 + KB-19 with retry
    │   └── calibration_manager.go      # Concordance tracking + LR estimation
    └── api/
        ├── server.go                   # Gin HTTP server + middleware
        ├── routes.go                   # Route registration
        ├── session_handlers.go         # POST/GET sessions, answers, suspend/resume
        ├── answer_handlers.go          # POST answers with fan-out
        ├── node_handlers.go            # GET nodes, POST reload
        └── calibration_handlers.go     # Feedback, status, golden import
```

---

## Data Models (6 Tables)

### 1. HPISession (hpi_sessions)
| Field | Type | Notes |
|-------|------|-------|
| session_id | UUID PK | |
| patient_id | UUID indexed | FK to KB-20 |
| node_id | string indexed | P01_CHEST_PAIN etc. |
| stratum_label | string | DM_ONLY / DM_HTN / DM_HTN_CKD |
| ckd_substage | string nullable | G1-G5 from KB-20 |
| status | enum | INITIALISING / ACTIVE / SUSPENDED / SAFETY_ESCALATED / COMPLETED / ABANDONED / STRATUM_DRIFTED |
| log_odds_state | JSONB | map[differential_id -> float64] |
| cm_log_deltas_applied | JSONB | map[cm_id -> float64] audit trail |
| cluster_answered | JSONB | map[cluster_name -> int] for R-02 |
| reliability_modifier | float64 | From KB-21 answer-reliability (R-03) |
| guideline_prior_refs | text[] | KB-1 guideline references (N-01) |
| questions_asked | int | Excluding PATA_NAHI |
| questions_pata_nahi | int | Separate counter |
| safety_flags | JSONB | SafetyFlag IDs activated |
| current_question_id | string nullable | Awaiting response |
| substage_drifted | bool | R-04: set on resume if CKD changed |
| started_at | timestamptz | |
| last_activity_at | timestamptz indexed | Session expiry basis |
| completed_at | timestamptz nullable | |
| outcome_published | bool | HPI_COMPLETE sent successfully |

### 2. SessionAnswer (session_answers) — append-only
| Field | Type | Notes |
|-------|------|-------|
| answer_id | UUID PK | |
| session_id | UUID FK indexed | |
| question_id | string | P02_Q007 etc. |
| answer_value | string | YES / NO / PATA_NAHI / numeric |
| lr_applied | JSONB | map[diff_id -> float64] log(LR) applied |
| information_gain_observed | float64 | H_before - H_after |
| was_pata_nahi | bool indexed | Fast filter |
| answer_latency_ms | int | WhatsApp round-trip |
| answered_at | timestamptz indexed | |

### 3. DifferentialSnapshot (differential_snapshots)
| Field | Type | Notes |
|-------|------|-------|
| snapshot_id | UUID PK | |
| session_id | UUID FK unique | One per completed session |
| ranked_differentials | JSONB | []DifferentialEntry sorted desc |
| safety_flags | JSONB | All fired SafetyFlag records |
| top_diagnosis | string | Convenience: ranked[0].id |
| top_posterior | float64 | ranked[0].posterior |
| convergence_reached | bool | |
| questions_to_convergence | int nullable | |
| guideline_prior_refs | text[] | N-01: KB-1 refs for audit |
| clinician_adjudication | string nullable | POST /calibration/feedback |
| concordant | bool nullable | top_diagnosis == adjudication |

### 4. SafetyFlag (safety_flags)
| Field | Type | Notes |
|-------|------|-------|
| flag_id | string | SF_ACS_POSSIBLE etc. |
| session_id | UUID FK indexed | |
| severity | enum | IMMEDIATE / URGENT / WARN |
| trigger_expression | string | Audit trail |
| differential_context | JSONB | Top-3 at time of flag |
| recommended_action | string | Plain-language for KB-23 |
| medication_safety_context | JSONB nullable | N-02: KB-9 enrichment |
| published_to_kb19 | bool | IMMEDIATE fast-path |
| fired_at | timestamptz indexed | |

### 5. CalibrationRecord (calibration_records)
| Field | Type | Notes |
|-------|------|-------|
| record_id | UUID PK | |
| snapshot_id | UUID FK indexed | |
| node_id | string indexed | |
| stratum_label | string indexed | F-06: per-stratum queries |
| ckd_substage | string nullable indexed | |
| confirmed_diagnosis | string | Clinician adjudication |
| engine_top_1 | string | KB-22 top differential |
| engine_top_3 | text[] | KB-22 top-3 |
| concordant_top1 | bool indexed | |
| concordant_top3 | bool indexed | |
| question_answers | JSONB | Full answer sequence for LR estimation |
| adjudicated_at | timestamptz indexed | |

### 6. CrossNodeTrigger (cross_node_triggers) — F-07
| Field | Type | Notes |
|-------|------|-------|
| trigger_id | string PK | SF_CHEST_PAIN_ACUTE etc. |
| condition | string | Boolean expression |
| severity | enum | IMMEDIATE / URGENT / WARN |
| recommended_action | string | |
| active | bool | |

---

## REST API (18 endpoints)

### Infrastructure
| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Liveness: DB + Redis + KB-20 reachability |
| GET | /readiness | Readiness: DB only |
| GET | /metrics | Prometheus metrics |
| POST | /internal/nodes/reload | Hot-reload node YAMLs |

### Sessions
| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/sessions | Create session: `{patient_id, node_id}` -> session_id + first_question |
| GET | /api/v1/sessions/:id | Full session state |
| POST | /api/v1/sessions/:id/answers | Submit answer -> next_question + top_3 + safety_flags |
| POST | /api/v1/sessions/:id/suspend | Mark SUSPENDED (WhatsApp drop) |
| POST | /api/v1/sessions/:id/resume | Resume with R-04 stale-stratum detection |
| POST | /api/v1/sessions/:id/complete | Force-complete (clinician override) |

### Differentials, Safety & Nodes
| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/sessions/:id/differential | Current ranked differential with CM breakdown |
| GET | /api/v1/sessions/:id/safety | All safety flags with severity + actions |
| GET | /api/v1/snapshots/:session_id | Completed session DifferentialSnapshot |
| GET | /api/v1/nodes | List loaded nodes with calibration status |
| GET | /api/v1/nodes/:node_id | Full node definition |

### Calibration
| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/calibration/feedback | Submit adjudication: `{snapshot_id, confirmed_diagnosis}` |
| GET | /api/v1/calibration/status/:node_id | Concordance stats (?stratum=, ?ckd_substage=) |
| POST | /api/v1/calibration/import-golden | Bulk import golden dataset (ADMIN scope) |

---

## Core Clinical Logic

### Bayesian Engine Equations
```
Initialisation:  lo_d = log(prior_d / (1 - prior_d))        # per-stratum from node YAML
KB-1 injection:  lo_d += guideline_adjustment_d               # N-01 (if available)
CM application:  lo_d += SUM_i(delta_i_d)                     # F-01 additive
   delta_i_d = log(1 + adj_mag) if INCREASE_PRIOR
             = log(1 - adj_mag) if DECREASE_PRIOR
   adj_mag = base_mag * min(1.0, adherence_score / 0.70)      # F-03

Answer update:
   YES  -> lo_d += log(LR+_d) * reliability_modifier          # R-03
   NO   -> lo_d += log(LR-_d) * reliability_modifier
   PATA -> lo_d += 0.0                                         # F-04

Cluster dampening (R-02):
   If cluster_answered[cluster] >= 1:
     lr_delta *= cluster_dampening ^ cluster_answered[cluster]

Dual-criterion termination (R-01):
   BOTH:          top_posterior >= threshold AND (top - second) >= gap
   EITHER:        top_posterior >= threshold OR  (top - second) >= gap
   POSTERIOR_ONLY: top_posterior >= threshold

Output:  p_d = 1 / (1 + exp(-lo_d))                           # sigmoid on read path
```

### Session Lifecycle (9-step init, 9-step per-answer)
```
INIT:
1. Validate node_id (NodeLoader.Get)
2. Create HPISession (status=INITIALISING)
3. Parallel 3-goroutine fetch:
   A: KB-20 /patient/:id/stratum/:node_id (40ms deadline, REQUIRED)
   B: KB-21 /patient/:id/adherence-weights (40ms, optional)
   C: KB-21 /patient/:id/answer-reliability (40ms, optional)
4. Snapshot stratum + ckd_substage
5. Load per-stratum priors -> log-odds vector (F-05)
6. Query KB-1 for guideline prior adjustments (N-01, optional)
7. Apply CMs with adherence scaling (F-01 + F-03)
8. Start SafetyEngine goroutine (F-02)
9. Select first question (mandatory-first)
10. status -> ACTIVE

ANSWER:
1. Validate answer (YES/NO/PATA_NAHI/numeric), question_id match
2. Fan out to BayesianEngine channel + SafetyEngine channel
3. Bayesian update with reliability weighting + cluster dampening
4. Safety evaluation (5ms timeout, non-blocking)
5. Compute information_gain_observed
6. Async telemetry write to KB-21
7. Check 5 termination conditions
8. If safety flag URGENT/IMMEDIATE: query KB-9 (N-02, 30ms, non-blocking)
9. Select next question (entropy maximisation)
10. Respond: next_question + top_3 + safety_flags (target <= 50ms)
```

---

## KB Dependency Contracts

| KB | Direction | Endpoint | Timing | Failure Mode |
|----|-----------|----------|--------|-------------|
| KB-20 (8131) | READ | GET /patient/:id/stratum/:node_id | Session start (required, 40ms) | 503 — session not created |
| KB-21 (8133) | READ | GET /patient/:id/adherence-weights | Session start (parallel, 40ms) | Timeout -> scale=1.0 |
| KB-21 (8133) | READ | GET /patient/:id/answer-reliability | Session start (parallel, 40ms) | Timeout -> reliability=1.0 |
| KB-1 | READ | GET /guidelines/prior-adjustments | Session start (optional) | Unavailable -> node YAML priors only |
| KB-1 | READ | GET /guidelines/management-summary | On safety flag (optional) | Flag fires without guideline text |
| KB-9 | READ | GET /medication-advisor/contraindications | On URGENT/IMMEDIATE flag (30ms) | Flag fires without med enrichment |
| KB-21 (8133) | WRITE | POST /patient/:id/question-telemetry | Per-answer (async) | Logged, retried, non-blocking |
| KB-23 (8134) | WRITE | POST /decision-cards | Session completion (sync, retry) | 3x retry at 30s intervals |
| KB-19 (8129) | WRITE | HPI_COMPLETE event | Session completion (sync, retry) | Same retry as KB-23 |
| KB-19 (8129) | WRITE | SAFETY_ALERT event | On IMMEDIATE flag (fast-path) | 5s retry interval |

---

## Implementation Phases

### Phase 1 — Infrastructure (Week 1)

**Files**: `main.go`, `config.go`, `connection.go`, `redis.go`, `collector.go`, `server.go`, `routes.go`, `node_loader.go`, all `models/*.go`

**Deliverables**:
- [ ] KB-5 pattern reuse: main.go init sequence (config -> DB -> Redis -> metrics -> nodes -> server)
- [ ] `config.go`: PORT=8132, DATABASE_URL, REDIS_URL, KB20_URL, KB21_URL, KB1_URL, KB9_URL, KB19_URL, KB23_URL, NODES_DIR, timeouts
- [ ] `connection.go`: GORM PostgreSQL on port 5437, AutoMigrate 6 tables
- [ ] `redis.go`: Prefixes `kb22:session:` (24h), `kb22:node:` (1h), `kb22:diff:` (10m), `kb22:cal:` (30m)
- [ ] `collector.go`: SessionsStarted, QuestionsAsked, SafetyFlagsRaised, DifferentialConverged, PatanahiRate, CalibrationConcordance
- [ ] All 7 model files with GORM tags and JSONB custom types
- [ ] `node_loader.go`: YAML parse + startup validation (all Section 6.2 rules) + hot-reload endpoint
- [ ] R-07 schema: `lr_source`, `lr_evidence_class`, `population_reference` fields — WARNING on missing (not error in Circle 1)
- [ ] R-06 schema: `type: BOOLEAN` (default) / `COMPOSITE_SCORE` (rejected at startup in Circle 1)
- [ ] `migrations/001_initial_schema.sql`: DDL for all 6 tables + indexes
- [ ] Health + readiness probes
- [ ] `go.mod`: gin, gorm, go-redis, uuid, zap, prometheus, yaml.v3

**Verification**: `go build` + `go vet` exit 0

---

### Phase 2 — Bayesian + Safety Core (Week 2-3)

**Files**: `bayesian_engine.go`, `safety_engine.go`, `cm_applicator.go`, `session_context_provider.go`

**Deliverables**:
- [ ] `bayesian_engine.go`:
  - Log-odds internal state (`map[string]float64`) — F-01
  - `InitPriors(node, stratum)`: per-stratum prior -> log-odds — F-05
  - `Update(q_id, answer, reliability_modifier)`: log(LR) * reliability — R-03
  - PATA_NAHI -> delta 0.0 — F-04
  - Cluster dampening: `cluster_answered` tracking, `lr_delta *= dampening^n` — R-02
  - `CheckConvergence()`: dual-criterion (posterior + gap + convergence_logic) — R-01
  - `GetPosteriors()`: sigmoid on read path only, normalised
- [ ] `safety_engine.go`:
  - `Start()` launches goroutine with `recover()` — F-02
  - Buffered channel receives answer events
  - Boolean trigger evaluation from node YAML expressions
  - SafetyFlag creation with differential context
  - IMMEDIATE triggers published to KB-19 immediately
  - Independent of BayesianEngine state
- [ ] `cm_applicator.go`:
  - `Apply(active_modifiers, adherence_weights)` — F-01 + F-03
  - `adjusted_mag = base_mag * min(1.0, adherence/0.70)`
  - Log-delta summation: `log(1 + adj_mag)` or `log(1 - adj_mag)`
- [ ] `session_context_provider.go`:
  - 3 parallel goroutines: KB-20 stratum + KB-21 adherence + KB-21 reliability
  - 40ms deadline each with `context.WithTimeout`
  - KB-20 required; KB-21 endpoints optional (defaults on timeout)

**Unit Tests** (critical — these validate RED findings):
- [ ] Test 2: Log-odds composition (3 CMs, verify additive != multiplicative)
- [ ] Test 3: Pata-nahi neutrality (5 PATA_NAHI -> posterior unchanged)
- [ ] Test 4: Safety goroutine isolation (inject panic in Bayesian, verify safety continues) — **must be CI test**
- [ ] Test 5: Adherence scaling (mag=0.4, adherence=0.35 -> adjusted=0.20)
- [ ] Test 6: Multi-stratum priors (DM_ONLY vs DM_HTN_CKD log-odds difference)
- [ ] Cluster dampening: 2nd YES in same cluster applies dampened LR
- [ ] Dual-criterion termination: BOTH, EITHER, POSTERIOR_ONLY modes
- [ ] Reliability weighting: `LR^0.70` for DECLINING phenotype patient

---

### Phase 3 — Question Orchestration + KB-1 Integration (Week 3-4)

**Files**: `question_orchestrator.go`, `guideline_client.go`

**Deliverables**:
- [ ] `question_orchestrator.go`:
  - Mandatory-first ordering (safety questions precede entropy-ranked)
  - Entropy maximisation: `IG(q) = H_current - SUM_a p(a|q) * H(posterior|q=a)` — Gap A01
  - One-step lookahead (50ms constraint)
  - Branch condition evaluator: stratum, ckd_substage, prior answers
  - Tie-breaking: `minimum_inclusion_guard` questions preferred — R-05
  - Fallback tie-break: lowest pata-nahi rate from KB-21
  - `minimum_inclusion_guard` auto-injected on safety trigger component questions at startup
- [ ] `guideline_client.go`:
  - `GetPriorAdjustments(node_id, stratum, ckd_substage)` from KB-1 — N-01
  - `GetManagementSummary(flag_id, stratum)` from KB-1 — N-01
  - Graceful degradation: KB-1 unavailable -> node YAML priors only
  - `guideline_prior_refs` stored on session and snapshot

**Unit Tests**:
- [ ] Test 7: Entropy maximisation (3-differential, known LRs, verify Q* has highest IG)
- [ ] Test 8: Branch condition evaluation (DM_HTN_CKD G3b includes Q007; DM_ONLY excludes)
- [ ] Minimum inclusion guard tie-breaking
- [ ] KB-1 unavailable -> graceful degradation

---

### Phase 4 — Session Lifecycle + API (Week 4-5)

**Files**: `session_service.go`, `session_handlers.go`, `answer_handlers.go`, `node_handlers.go`

**Deliverables**:
- [ ] `session_service.go`:
  - Full state machine: INITIALISING -> ACTIVE -> COMPLETED/ABANDONED/SAFETY_ESCALATED/STRATUM_DRIFTED
  - 9-step session initialisation sequence
  - 9-step per-answer processing with fan-out
  - 5 termination conditions (IMMEDIATE safety, convergence, max_questions, dominant safety, timeout)
- [ ] Session handlers:
  - POST /sessions (create with parallel KB fetch)
  - GET /sessions/:id (full state)
  - POST /sessions/:id/suspend — F-08
  - POST /sessions/:id/resume — F-08 + R-04 (stale-stratum detection, re-query KB-20)
  - POST /sessions/:id/complete (clinician override)
- [ ] Answer handlers:
  - POST /sessions/:id/answers (validate, fan-out, respond <= 50ms)
  - 409 on question_id mismatch
- [ ] Node handlers:
  - GET /nodes (list with calibration status)
  - GET /nodes/:node_id (full definition)
- [ ] R-04: On resume, re-query KB-20. If stratum changed -> STRATUM_DRIFTED + KB-19 event. If ckd_substage changed -> recompute priors.
- [ ] Nightly job: SUSPENDED > 24h -> ABANDONED

**Integration Tests**:
- [ ] Test 9: Full P1 synthetic trace (65M, DM+HTN, metformin+SGLT2i, 8 ACS-pattern answers -> top ACS > 0.75)
- [ ] Test 10: KB-21 timeout degradation (mock timeout, session proceeds with defaults)
- [ ] Session suspend + resume lifecycle
- [ ] Question_id mismatch -> 409

---

### Phase 5 — Telemetry + Publication + Calibration (Week 5-6)

**Files**: `telemetry_writer.go`, `outcome_publisher.go`, `calibration_manager.go`, `calibration_handlers.go`, `medication_safety_provider.go`

**Deliverables**:
- [ ] `telemetry_writer.go`:
  - Async goroutine POST to KB-21 /question-telemetry
  - Fields: question_id, node_id, stratum, IG_observed, was_pata_nahi, answer_latency_ms
  - Max 3 retries, 30s interval. Non-blocking.
- [ ] `outcome_publisher.go`:
  - HPI_COMPLETE to KB-23 (DifferentialSnapshot payload) — sync with retry
  - HPI_COMPLETE to KB-19 (event payload) — sync with retry
  - SAFETY_ALERT to KB-19 (IMMEDIATE fast-path, 5s retry)
- [ ] `medication_safety_provider.go` — N-02:
  - On URGENT/IMMEDIATE flag: GET KB-9 /medication-advisor/contraindications
  - 30ms timeout, non-blocking
  - Response appended to SafetyFlag.medication_safety_context
- [ ] `calibration_manager.go`:
  - POST /calibration/feedback -> CalibrationRecord creation
  - GET /calibration/status/:node_id -> concordance stats
  - F-06: ?stratum= and ?ckd_substage= query parameters
  - Tier 1: synthetic traces, expert panel concordance
  - Tier 2: LR blending (N >= 20) with conjugate prior
  - Tier 3: Golden dataset import
  - POST /calibration/import-golden (ADMIN scope)

---

### Phase 6 — Cross-Node Safety + Docker (Week 6)

**Files**: `cross_node_safety.go`, `Dockerfile`, `docker-compose.yml`, `framework.yaml`, `README.md`

**Deliverables**:
- [ ] `cross_node_safety.go` — F-07:
  - Load `cross_node_triggers.yaml` alongside node-specific triggers
  - Global triggers: SF_CHEST_PAIN_ACUTE, SF_STROKE_SCREEN
  - Evaluated regardless of active node
- [ ] `Dockerfile`:
  - Multi-stage alpine build (builder + runtime)
  - Non-root user (UID 1001)
  - EXPOSE 8132
  - HEALTHCHECK /health
  - Copy /nodes and /migrations
- [ ] `docker-compose.yml`:
  - kb22-postgres (port 5437, postgres:15-alpine)
  - kb22-redis (port 6386, redis:7-alpine)
  - kb22-service (port 8132, mounts /nodes volume)
  - Health checks on all services
  - Depends_on with service_healthy conditions
- [ ] `framework.yaml`:
  - 50ms SLA on answer path
  - Integration map with KB-20, KB-21, KB-1, KB-9, KB-23, KB-19
  - 6 KB dependency contracts documented

---

## Configuration (Environment Variables)

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8132 | HTTP server port |
| DATABASE_URL | postgres://kb22_user:kb22_password@localhost:5437/kb_service_22?sslmode=disable | PostgreSQL |
| REDIS_URL | redis://localhost:6386 | Redis cache |
| KB20_URL | http://localhost:8131 | KB-20 Patient Profile |
| KB21_URL | http://localhost:8133 | KB-21 Behavioral Intelligence |
| KB1_URL | http://localhost:8081 | KB-1 Drug Rules / Guidelines |
| KB9_URL | http://localhost:8091 | KB-9 Medication Advisor |
| KB19_URL | http://localhost:8129 | KB-19 Protocol Orchestrator |
| KB23_URL | http://localhost:8134 | KB-23 Decision Cards |
| NODES_DIR | /app/nodes | P1-P26 YAML directory |
| KB20_TIMEOUT_MS | 40 | KB-20 query deadline |
| KB21_TIMEOUT_MS | 40 | KB-21 query deadline |
| KB9_TIMEOUT_MS | 30 | KB-9 query deadline |
| SESSION_TTL_HOURS | 24 | Redis + suspend expiry |
| ENVIRONMENT | development | development / production |
| DB_MAX_CONNECTIONS | 25 | Connection pool |

---

## Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| kb22_sessions_started_total | counter | Sessions created |
| kb22_questions_asked_total | counter | Questions answered |
| kb22_safety_flags_raised_total | counter | Safety flags fired (by severity) |
| kb22_differential_converged_total | counter | Sessions reaching convergence |
| kb22_patanahi_rate | gauge | Rolling pata-nahi rate per node |
| kb22_calibration_concordance | gauge | Top-1 concordance per node/stratum |
| kb22_answer_latency_ms | histogram | Answer processing time |
| kb22_session_init_latency_ms | histogram | Session init time (incl. KB fetches) |
| kb22_kb20_fetch_duration_ms | histogram | KB-20 query latency |
| kb22_kb21_fetch_duration_ms | histogram | KB-21 query latency |
| kb22_entropy_computation_ms | histogram | Question ordering compute time |

---

## Verification Tests

| # | Test | Expected | Validates |
|---|------|----------|-----------|
| 1 | `go build` + `go vet` | Exit 0, 0 warnings | Pre-condition |
| 2 | Log-odds composition | 3 CMs: log-delta sum = 0.559; multiplicative = 0.006 | F-01 |
| 3 | Pata-nahi neutrality | 5 PATA_NAHI -> posterior unchanged; then YES shifts correctly | F-04 |
| 4 | Safety goroutine isolation | Inject Bayesian panic; safety alert still published to KB-19 | F-02 (CI test) |
| 5 | Adherence scaling | mag=0.4, adherence=0.35 -> adjusted=0.20 | F-03 |
| 6 | Multi-stratum priors | DM_ONLY vs DM_HTN_CKD differ by log(0.22/0.78)-log(0.06/0.94) | F-05 |
| 7 | Entropy maximisation | Known LR table: Q* has highest IG | A01 |
| 8 | Branch conditions | DM_HTN_CKD G3b includes Q007; DM_ONLY excludes | F-05+branch |
| 9 | Full P1 synthetic trace | 65M DM+HTN, 8 ACS answers -> top ACS > 0.75 | End-to-end |
| 10 | KB-21 timeout | Session proceeds with scale=1.0, reliability=1.0 | F-03 degradation |

---

## Implementation Gate Checklist (Pre-Build)

| # | Gate Item | Owner | Status |
|---|-----------|-------|--------|
| 1 | P1 + P2 node YAMLs with updated schema (cluster, convergence_logic, population_reference) | Clinical team | PENDING |
| 2 | KB-21 delivers GET /patient/:id/answer-reliability (N-04) | KB-21 team | PENDING |
| 3 | KB-1 confirms /guidelines/prior-adjustments and /management-summary contracts | KB-1 team | CONFIRM |
| 4 | KB-9 confirms /medication-advisor/contraindications contract | KB-9 team | CONFIRM |
| 5 | Clinical review of India-specific differentials (N-03) | Clinical team | PENDING |
| 6 | KB-20 confirms drug_class on CONCOMITANT_DRUG modifiers | KB-20 team | CONFIRM |
| 7 | Dual-criterion thresholds agreed per node | Clinical + KB-22 | PENDING |
| 8 | Safety goroutine test (Test 4) confirmed as CI test | KB-22 engineering | PENDING |

---

## Implementation Scorecard

| Metric | Count | Coverage |
|--------|-------|----------|
| Source files | ~33 | 100% |
| Go packages | 8 | 100% |
| GORM models | 6 | 100% |
| Service files | 14 | 100% (incl. guideline_client, medication_safety_provider, cross_node_safety) |
| REST endpoints | 18 | 100% |
| RED findings | 8 | 100% (F-01 through F-05 + R-01, R-02, R-03) |
| AMBER findings | 4 | 100% (F-06, F-07, F-08, R-07) |
| NEW findings | 4 | 100% (N-01 through N-04) |
| Gap register items | 7 | 100% (A01, A02, B01, B03, B04, D01, D06) |
| KB contracts | 6 | 100% (KB-20, KB-21x2, KB-1, KB-9, KB-23, KB-19) |
| Calibration tiers | 3 | 100% |
| Node YAML schema | Defined + validated | 100% |
| Verification tests | 10 | 100% |

---

## Critical Path

1. **Phase 1-2 can begin immediately** — no dependency on node content
2. **KB-21 must deliver answer-reliability endpoint** before Phase 2 R-03 testing
3. **P1 + P2 YAMLs required** before Phase 4 integration testing
4. **Phase 2 is highest-risk**: log-odds engine + safety goroutine + all RED findings
5. **Test 9 (P1 synthetic trace)** is the minimum clinical gate before staging
6. **50ms answer latency SLA** must be load-tested at Phase 4 completion
