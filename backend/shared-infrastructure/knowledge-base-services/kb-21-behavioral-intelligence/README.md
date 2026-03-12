# KB-21 Behavioral Intelligence Service

**Port**: 8093 | **Language**: Go 1.22 | **Framework**: Gin | **Database**: PostgreSQL + Redis

The Behavioral Intelligence Service is the behavioral-clinical interface of the Vaidshala v3 architecture. It tracks patient medication adherence, engagement patterns, and behavioral phenotypes — then feeds this behavioral state into the correction loop via V-MCU integration.

## Architectural Position

```
Patient (WhatsApp/SMS/IVR)
    ↓ InteractionEvent
KB-21 (Behavioral Intelligence)
    ├── AdherenceState ──────→ V-MCU (loop_trust_score + per_class_adherence gates titration)
    ├── EngagementProfile ───→ V-MCU (phenotype informs control authority)
    ├── AdherenceWeights ────→ KB-22 (scales drug-ADR CM magnitudes)
    ├── BEHAVIORAL_GAP ──────→ KB-23 (MODIFY gate, two-hop fast-path) [G-01]
    ├── DISCORDANT ──────────→ KB-23 (SAFE gate, medication review) [G-01]
    ├── HYPO_RISK_ELEVATED ──→ KB-23 (PAUSE gate, fast-path) [G-03]
    │                     └──→ KB-19 (secondary event bus notification)
    ├── OutcomeCorrelation ──→ V-MCU (pharmacological vs behavioral differential)
    └── ← LAB_RESULT ────────── KB-20 (clinical outcome feedback)
```

## Pre-Implementation Review Findings Implemented

This service incorporates all 11 findings from the KB-21 Pre-Implementation Final Review (March 2026):

| Finding | Severity | Description | Implementation |
|---------|----------|-------------|----------------|
| F-01 | RED | Loop Trust Calibration | `loop_trust_score` on EngagementProfile |
| F-02 | RED | KB-24 Rejected → V-MCU Assembler Pattern | No KB-24; V-MCU assembles locally |
| F-03 | RED | Hypoglycemia Safety Loop | `HYPO_RISK_ELEVATED` event via HypoRiskService |
| F-04 | RED | Behavioral-Clinical Feedback | `OutcomeCorrelation` entity + CorrelationService |
| F-05 | AMBER | Meal Adherence (Circle 1) | `evening_meal_confirmed`, `fasting_today` on InteractionEvent |
| F-06 | RED | Adherence → Diagnostic Reliability | `/adherence-weights` endpoint for KB-22 |
| F-07 | AMBER | FDC-Linked Adherence | Single AdherenceState for FDC, projected to components |
| F-08 | AMBER | Titration Window Mismatch | `adherence_score_7d` (7-day) alongside 30-day score |
| F-09 | AMBER | Device Change Detection | `device_change_suspected` on EngagementProfile |
| F-10 | GREEN | Privacy / DPDPA | `consent_for_festival_adapt`, `retention_policy_months` |
| F-11 | GREEN | Aggregate Analytics | `CohortSnapshot` entity + analytics endpoints |

## Post-Implementation Gap Fixes (G-01 through G-04)

| Gap | Severity | Problem | Fix |
|-----|----------|---------|-----|
| G-01 | RED | BEHAVIORAL_GAP has no path to KB-23 — classification sits in DB with nothing acting on it | CorrelationService calls SafetyClient.AlertBehavioralGap() directly after saving OutcomeCorrelation. KB-23 generates MODIFY-gate DecisionCard. |
| G-02 | RED | LoopTrustResponse returns single aggregated adherence — V-MCU cannot compute per-class gain_factor | LoopTrustResponse now includes `per_class_adherence` map keyed by drug class with Score7d, Score30d, Trend, DataQuality, IsFDC, Source. |
| G-03 | RED | HYPO_RISK only published to event bus (4-hop path) — too slow for safety | HypoRiskService calls SafetyClient.AlertHypoRisk() directly to KB-23 (2-hop fast-path). PAUSE gate (not HALT — behavioral risk is probabilistic). Event bus publish retained as secondary notification. |
| G-04 | RED | New patients with no WhatsApp interaction get adherence=0 → V-MCU bottoms out gain_factor at 0.25 | PRE_GATEWAY_DEFAULT_ADHERENCE=0.70 returned when no interaction data exists. Response includes `adherence_source: "DEFAULT_PRE_GATEWAY"` flag. |

### KB-23 Safety Fast-Path (G-01, G-03)

KB-21 communicates directly with KB-23 via `SafetyClient` for two safety-critical scenarios:

1. **BEHAVIORAL_GAP** (G-01): CorrelationService detects adherence↓ + outcome↓ → calls KB-23 with MODIFY gate + `dose_adjustment_notes: "BEHAVIORAL_GAP"`. Prevents V-MCU from escalating insulin on non-adherent patients.

2. **HYPO_RISK_ELEVATED** (G-03): HypoRiskService detects meal skip + insulin, erratic adherence + insulin, or fasting + SU → calls KB-23 with PAUSE gate. KB-19 notified as secondary event.

This is a two-hop path (KB-21 → KB-23) vs the previous four-hop path (KB-21 → event bus → KB-19 → KB-4).

## Key Concepts

### Loop Trust Score (Finding F-01)

The composite trust input for V-MCU's correction loop control authority:

```
loop_trust_score = adherence_score × data_quality_weight × phenotype_weight × temporal_stability
```

| Component | Values |
|-----------|--------|
| data_quality_weight | HIGH=1.0, MODERATE=0.75, LOW=0.50 |
| phenotype_weight | CHAMPION=1.0, STEADY=0.90, SPORADIC=0.65, DECLINING=0.40, DORMANT=0.10, CHURNED=0.0 |
| temporal_stability | STABLE/IMPROVING=1.0, DECLINING=0.70, CRITICAL=0.40 |

**V-MCU consumes** the `loop_trust_score` and applies its own thresholds (KB-21 provides informational recommendations: AUTO ≥0.75, ASSISTED ≥0.55, CONFIRM ≥0.35, DISABLED <0.35).

### Behavioral Phenotypes

| Phenotype | Criteria | V-MCU Implication |
|-----------|----------|-------------------|
| CHAMPION | adherence ≥ 0.90, stable/improving | Full auto-titration eligible |
| STEADY | adherence 0.70–0.89, stable | Standard monitoring |
| SPORADIC | adherence 0.50–0.69, erratic | Enhanced monitoring required |
| DECLINING | any level, downward trend | Behavioral intervention first |
| DORMANT | no interaction 14+ days | Re-engagement before titration |
| CHURNED | no interaction 30+ days | Loop disabled |

### Three-Loop Correction Architecture (Post-Review)

1. **Pharmacological Loop** (KB-20 ↔ V-MCU): FBG high → dose increase → FBG improves. KB-21 provides `loop_trust_score` to gate dose changes.
2. **Behavioral Loop** (KB-21 ↔ Nudge Engine ↔ Patient): Adherence declining → barrier detected → targeted nudge → adherence improves.
3. **Outcome Loop** (KB-21 OutcomeCorrelation): Connects both loops. CONCORDANT = celebrate. DISCORDANT = escalate pharmacologically. BEHAVIORAL_GAP = fix behavior first.

### Adherence-Adjusted CM Weights (Finding F-06)

KB-22 queries `GET /api/v1/patient/{id}/adherence-weights` to scale drug-ADR context modifier magnitudes:

```
adjusted_magnitude = base_magnitude × min(1.0, adherence_score / 0.70)
```

This prevents drug-ADR CMs from dominating differentials when the patient is not actually taking the drug.

## API Reference

### Infrastructure

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check with database + cache status |
| GET | `/metrics` | Prometheus metrics |

### Patient Interactions

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/patient/{id}/interaction` | Record patient interaction event |

### Adherence

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/patient/{id}/adherence` | Get adherence state (all drug classes) |
| POST | `/api/v1/patient/{id}/adherence/recompute` | Force adherence recomputation |
| GET | `/api/v1/patient/{id}/adherence-weights` | Adherence weights for KB-22 (F-06) |

### Engagement

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/patient/{id}/engagement` | Get engagement profile + phenotype |
| POST | `/api/v1/patient/{id}/engagement/recompute` | Force engagement recomputation |
| GET | `/api/v1/patient/{id}/loop-trust` | Loop trust score for V-MCU (F-01) |

### Outcome Correlation (Finding F-04)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/patient/{id}/outcome-correlation` | Latest OutcomeCorrelation |
| GET | `/api/v1/patient/{id}/outcome-correlation/history` | Full correlation history |

### Hypoglycemia Risk (Finding F-03)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/patient/{id}/hypo-risk` | Evaluate behavioral hypo risk factors |

### Event Webhooks (Dev Mode)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/webhooks/lab-result` | Receive KB-20 LAB_RESULT events |

### Analytics (Finding F-11)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/analytics/phenotype-distribution` | Current phenotype distribution |
| GET | `/api/v1/analytics/question-effectiveness` | Question telemetry rankings |
| GET | `/api/v1/analytics/cohort` | Weekly cohort snapshots |

## Events Published

| Event | Consumer | Trigger |
|-------|----------|---------|
| `HYPO_RISK_ELEVATED` | KB-19, KB-4 | Meal skip + insulin, erratic adherence + insulin, fasting + SU |
| `ADHERENCE_CHANGED` | V-MCU, KB-23 | Adherence score change beyond threshold |
| `PHENOTYPE_CHANGED` | V-MCU, KB-23 | Patient phenotype transition |

## Events Consumed

| Event | Source | Action |
|-------|--------|--------|
| `LAB_RESULT` | KB-20 | Triggers OutcomeCorrelation recomputation |
| `MEDICATION_CHANGED` | KB-20 | Triggers adherence state reconciliation |

## Quick Start

### Local Development

```bash
# Start infrastructure
docker-compose up -d kb21-postgres kb21-redis

# Set environment
export DATABASE_URL="postgres://kb21_user:kb21_pass@localhost:5434/kb_behavioral_intelligence"
export REDIS_URL="redis://localhost:6381/21"
export ENVIRONMENT=development

# Run service
go run main.go
```

### Docker

```bash
docker-compose up -d
```

### Verify

```bash
curl http://localhost:8093/health
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8093 | HTTP server port |
| ENVIRONMENT | development | development or production |
| DATABASE_URL | postgres://...5433/kb_behavioral_intelligence | PostgreSQL DSN |
| REDIS_URL | redis://localhost:6380/21 | Redis cache URL |
| KB20_PATIENT_PROFILE_URL | http://localhost:8095 | KB-20 service URL |
| KB4_PATIENT_SAFETY_URL | http://localhost:8088 | KB-4 service URL |
| KB23_SAFETY_URL | http://localhost:8098 | KB-23 direct fast-path for safety alerts (G-01/G-03) |
| PRE_GATEWAY_DEFAULT_ADHERENCE | 0.70 | Default adherence when no WhatsApp data exists (G-04) |
| EVENT_BUS_ENABLED | false | Enable Kafka event bus |
| PHENOTYPE_EVAL_INTERVAL_HOURS | 24 | Phenotype re-evaluation frequency |
| OUTCOME_CORRELATION_MIN_EVENTS | 5 | Minimum events for correlation |
| NUDGE_MAX_PER_DAY | 3 | Maximum nudges per patient per day |

## Database Migrations

```
migrations/
├── 001_initial_schema.sql          — Core tables: interaction_events, adherence_states,
│                                     engagement_profiles, question_telemetry,
│                                     nudge_records, dietary_signals, barrier_detections
├── 002_outcome_correlation.sql     — OutcomeCorrelation entity (Finding F-04)
└── 003_cohort_analytics.sql        — CohortSnapshot for population analytics (Finding F-11)
```

## KB-20 Contract

| Direction | Data | Purpose |
|-----------|------|---------|
| KB-21 → KB-20 | `loop_trust_score` via V-MCU query | Gates correction loop authority |
| KB-20 → KB-21 | `LAB_RESULT` events (HbA1c, FBG) | Triggers OutcomeCorrelation |
| KB-20 → KB-21 | `MEDICATION_CHANGED` events | FDC reconciliation |
| KB-21 → KB-22 | `adherence_weights` | Scales drug-ADR CM activation |
| KB-21 → KB-19 | `HYPO_RISK_ELEVATED` | Behavioral safety signals |

## Implementation Priority (From Review)

| Week | Deliverable | Review Findings Included |
|------|-------------|-------------------------|
| 3 | InteractionEvent + AdherenceState + KB-20 sync | F-07 (FDC), F-08 (7d score), F-05 (dietary) |
| 4 | EngagementProfile + loop_trust_score | F-01 (trust), F-03 (hypo risk event) |
| 5 | Phenotyping + QuestionTelemetry + pata-nahi | F-06 (adherence-weights endpoint) |
| 6–7 | Nudge engine + barrier detection | — |
| 7–8 | OutcomeCorrelation | F-04 (most complex addition) |
| 8–9 | Decay prediction + festival calendar | F-09 (device change) |
| 9–10+ | Advanced nudges + CohortAnalytics | F-11 (cohort), F-10 (privacy) |

**Total estimated effort**: 10–12 weeks (vs original 8–10; +2 weeks from review findings).
