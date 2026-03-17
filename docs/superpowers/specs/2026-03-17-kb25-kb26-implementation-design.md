# KB-25 Lifestyle Knowledge Graph + KB-26 Metabolic Digital Twin — Implementation Design

**Date**: 2026-03-17
**Strategy**: Sequential (KB-25 first, then KB-26)
**Timeline**: ~19 weeks
**Spec Sources**: `KB-25_Lifestyle_Knowledge_Graph_Specification.docx`, `KB-26_Metabolic_Digital_Twin_Specification.docx`

---

## 1. Overview

Two new Go microservices following existing KB service conventions (Gin/zap/Prometheus), built sequentially because KB-26 depends on KB-25 for causal chain data, attribution queries, and effect sizes. KB-25 uses the Neo4j Go driver (no GORM — graph store). KB-26 uses GORM/PostgreSQL (matching KB-20/21 patterns).

**Port Registry**: Both services must be registered in `ports.yaml` during Phase 1 scaffolding.

| Service | Port | Primary Store | Purpose |
|---------|------|---------------|---------|
| KB-25 Lifestyle Knowledge Graph | 8136 | Neo4j 5.x + Redis | Causal reasoning for food/exercise interventions with drug-equivalent rigor |
| KB-26 Metabolic Digital Twin | 8137 | PostgreSQL + Redis | Persisted derived physiological state, coupled simulation, Bayesian calibration |

**Key architectural distinction**: KB-25 introduces Neo4j (a first for the platform) because causal chain traversals with variable-depth paths are naturally graph problems. KB-26 stays pure PostgreSQL for time-series twin state.

---

## 2. KB-25: Lifestyle Knowledge Graph

### 2.1 Service Identity (from spec §1.2)

- **Service Name**: KB-25: Lifestyle Knowledge Graph (LKG)
- **Port**: 8136
- **Graph Database**: Neo4j 5.x (database name: `lkg`)
- **API Runtime**: Go 1.22+
- **FHIR Resources**: NutritionOrder, ActivityDefinition, CarePlan, Goal
- **Primary Consumers**: KB-20, KB-21, KB-23, KB-24/V-MCU, Tier-1
- **Data Sources**: IFCT-2017, ACSM Exercise Guidelines, ADA MNT, ICMR-NIN 2024

### 2.2 Directory Structure

```
kb-25-lifestyle-knowledge-graph/
├── main.go
├── go.mod / go.sum
├── Dockerfile
├── docker-compose.yml
├── migrations/
│   └── 001_neo4j_schema.cypher           # Constraints + indexes (spec §7.1)
├── seed/
│   ├── ifct2017_foods.cypher             # 528 Food nodes
│   ├── exercises.cypher                  # 124 Exercise nodes
│   ├── nutrients.cypher                  # 67 Nutrient nodes (ICMR RDAs)
│   ├── physprocesses.cypher              # 83 PhysProcess nodes
│   ├── clinvars.cypher                   # 22 ClinVar nodes (KB-20 mapping)
│   ├── drugclasses.cypher                # 18 DrugClass cross-refs
│   ├── patientctx.cypher                # 45 PatientCtx modifier nodes
│   ├── causal_edges.cypher              # STIMULATES/IMPROVES/REDUCES with EffectDescriptors
│   ├── safety_rules.cypher              # LS-01..LS-14 contraindication edges
│   └── interactions.cypher              # Lifestyle-drug interaction edges
├── internal/
│   ├── config/
│   │   └── config.go                     # Neo4j, Redis, KB-20/21/1/4 URLs
│   ├── api/
│   │   ├── server.go                     # Gin server + middleware
│   │   ├── routes.go                     # 10 endpoints (spec §11.1)
│   │   ├── comparator_handlers.go        # /compare-interventions, /project-combined
│   │   ├── recommendation_handlers.go    # /recommend-lifestyle, /food-search
│   │   ├── safety_handlers.go            # /check-safety
│   │   ├── attribution_handlers.go       # /attribute-outcome
│   │   └── query_handlers.go             # /diet-quality, /exercise-rx, /causal-chain
│   ├── graph/
│   │   ├── client.go                     # Neo4j bolt driver wrapper
│   │   ├── queries.go                    # Cypher query templates (spec §7.2)
│   │   └── models.go                     # Go structs for graph nodes/edges
│   ├── models/
│   │   ├── effect_descriptor.go          # EffectDescriptor struct (spec §2.2.1)
│   │   ├── causal_chain.go               # CausalChain + ChainComponent
│   │   ├── safety_rules.go               # LSRule, ContraRef, InteractionEntry
│   │   ├── food.go                       # Food node struct (IFCT properties)
│   │   ├── exercise.go                   # Exercise node + safety tiers
│   │   └── comparison.go                 # ComparisonResult, InterventionOption
│   ├── services/
│   │   ├── chain_traversal.go            # Graph traversal: Food/Exercise → ClinVar
│   │   ├── effect_modifier.go            # ComputeModifiedEffect (spec §4.2)
│   │   ├── comparator_engine.go          # Intervention comparison (spec §6)
│   │   ├── safety_engine.go              # LS-01..LS-14 + drug interaction check
│   │   ├── attribution_engine.go         # Lab delta decomposition
│   │   ├── exercise_rx.go               # GenerateExerciseRx (spec §10.1)
│   │   └── diet_quality.go              # Diet quality scoring (0-100)
│   ├── clients/
│   │   ├── kb20_client.go               # Patient state snapshots
│   │   ├── kb21_client.go               # Adherence data
│   │   ├── kb1_client.go                # Drug effect data
│   │   └── kb4_client.go                # Drug safety cross-reference
│   ├── cache/
│   │   └── redis.go                     # Chains: 1hr TTL, patient: 5min TTL
│   └── metrics/
│       └── collector.go                  # Prometheus metrics
└── tests/
    ├── integration/
    │   ├── chain_traversal_test.go
    │   ├── comparator_test.go
    │   └── safety_test.go
    └── unit/
        ├── effect_modifier_test.go
        └── diet_quality_test.go
```

### 2.3 API Endpoints (spec §11.1)

| Endpoint | Method | Purpose | Consumer |
|----------|--------|---------|----------|
| `/api/v1/kb25/compare-interventions` | POST | Compare lifestyle vs pharmacological options | KB-23, V-MCU |
| `/api/v1/kb25/recommend-lifestyle` | POST | Patient-specific lifestyle recs with cultural context | KB-21, Tier-1 |
| `/api/v1/kb25/check-safety` | POST | Validate exercise/diet against contraindications | V-MCU, KB-20 |
| `/api/v1/kb25/attribute-outcome` | POST | Decompose lab change into lifestyle vs drug contribution | KB-20 |
| `/api/v1/kb25/project-combined` | POST | Project combined lifestyle + proposed med change | V-MCU |
| `/api/v1/kb25/diet-quality/{patientId}` | GET | Current diet quality score (0-100) | KB-23 |
| `/api/v1/kb25/exercise-rx/{patientId}` | GET | Current exercise prescription + compliance | KB-23 |
| `/api/v1/kb25/food-search` | GET | Search food DB by name/region/diet type/nutrient | KB-21 |
| `/api/v1/kb25/causal-chain/{target}` | GET | Retrieve all causal chains to a clinical variable | Internal/audit |
| `/api/v1/kb25/health` | GET | Service health check | Infrastructure |
| `/metrics` | GET | Prometheus metrics | Infrastructure |

### 2.4 Graph Scale (spec §13.1)

| Entity | Count |
|--------|-------|
| Food nodes | 528 |
| Exercise nodes | 124 |
| Nutrient nodes | 67 |
| PhysProcess nodes | 83 |
| ClinVar nodes | 22 |
| DrugClass nodes | 18 |
| PatientCtx nodes | 45 |
| **Total nodes** | **887** |
| **Total edges** | **~5,230** |

### 2.5 Key Data Structures

#### EffectDescriptor (spec §2.2.1)
```go
type EffectDescriptor struct {
    EffectSize         float64
    EffectUnit         string
    ConfidenceInterval [2]float64
    DoseResponse       DoseResponseCurve
    OnsetDays          int
    PeakEffectDays     int
    SteadyStateDays    int
    EvidenceGrade      string            // A|B|C|D
    SourcePMIDs        []string
    EffectModifiers    []ModifierRef
    Contraindications  []ContraRef
}
```

#### Evidence Grading (spec §2.3)
- **Grade A**: 2+ RCTs or meta-analysis, N>500 — primary option
- **Grade B**: 1 RCT N>100 or 3+ observational — moderate confidence
- **Grade C**: Observational or mechanistic — physician review required
- **Grade D**: Expert consensus — never auto-applied, always physician-gated

**Rule**: Lifestyle intervention auto-selected over medication escalation ONLY if Grade A or B AND clinical urgency LOW or MODERATE.

#### Decision Rules (spec §6.2)
- HbA1c 6.5-7.5%, no urgency → LIFESTYLE FIRST for 90 days
- HbA1c 7.5-9.0% → LIFESTYLE + MEDICATION together
- HbA1c > 9.0% → MEDICATION PRIMARY, lifestyle adjunct only
- Prediabetes (5.7-6.4%) → LIFESTYLE ONLY
- BP 130-150/80-95 → LIFESTYLE FIRST for 90 days
- BP > 160/100 → MEDICATION PRIMARY

### 2.6 Safety Layer

#### Hard-Stops (LS-01 through LS-14)

| Rule | Condition | Blocked Intervention |
|------|-----------|---------------------|
| LS-01 | eGFR < 30 (CKD 4-5) | Protein > 0.6 g/kg/day |
| LS-02 | SBP > 180 or DBP > 110 | Vigorous exercise (MET > 6) |
| LS-03 | FBG < 70 in last 7d | Exercise >30min without carb adjustment |
| LS-04 | SU/insulin + exercise | Exercise without hypo warning |
| LS-05 | SGLT2i + vigorous exercise | Without hydration plan |
| LS-06 | Proliferative retinopathy | Resistance training, Valsalva |
| LS-07 | Peripheral neuropathy | High-impact foot exercises |
| LS-08 | Pregnancy + T2DM/GDM | Caloric deficit diet |
| LS-09 | Hyperkalemia (K+ > 5.5) | High-potassium foods |
| LS-10 | Recent cardiac event (30d) | Any exercise prescription |
| LS-11 | HbA1c > 13% | Lifestyle-only without medication |
| LS-12 | BMR < 1200 kcal | Caloric deficit diet |
| LS-13 | Gastroparesis | High-fiber diet > 25g/day |
| LS-14 | Eating disorder history | Calorie counting/restrictive diets |

### 2.7 Integration Contracts (spec §8)

| Contract | Direction | Purpose |
|----------|-----------|---------|
| Lifestyle Attribution | KB-20 → KB-25 → KB-20 | Decompose lab deltas into lifestyle vs drug fractions |
| Patient State Snapshot | KB-20 → KB-25 | Current state for modifier application |
| Lifestyle Projection | KB-25 → KB-20 | Projected FBG/HbA1c at 30/60/90d |
| Cultural Recommendations | KB-21 → KB-25 → KB-21 | Region/diet-specific food suggestions |
| Adherence Adjustment | KB-21 → KB-25 → KB-21 | Adherence-weighted projections |
| Decision Card Data | KB-25 → KB-23 | Comparison panels, diet quality, safety alerts |
| Lifestyle Ceiling Check | V-MCU → KB-25 | Untapped lifestyle potential before med escalation |
| Drug-Exercise Interaction | V-MCU → KB-25 | Exercise interaction with current meds |
| Combined Projection | V-MCU → KB-25 | Lifestyle + proposed med change projection |

### 2.8 Docker Compose

```yaml
services:
  kb-25-lkg:
    build: .
    container_name: kb-25-lkg
    restart: unless-stopped
    ports:
      - "8136:8136"
    environment:
      PORT: "8136"
      NEO4J_URI: "bolt://kb25-neo4j:7687"
      NEO4J_DATABASE: "lkg"
      NEO4J_USER: "neo4j"
      NEO4J_PASSWORD: "kb25_lkg_password"
      REDIS_URL: "redis://kb25-redis:6379"
      KB20_URL: "http://kb-20-patient-profile:8131"
      KB21_URL: "http://kb-21-behavioral-intelligence:8133"
      KB1_URL: "http://kb-1-drug-rules:8081"
      KB4_URL: "http://kb-4-patient-safety:8088"
      CACHE_TTL_CHAINS: "3600"
      CACHE_TTL_PATIENT: "300"
    depends_on:
      kb25-neo4j:
        condition: service_healthy
      kb25-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--spider", "http://localhost:8136/health"]
      interval: 30s
      timeout: 5s
      start_period: 30s
      retries: 3
    networks:
      - kb25-network
      - kb-network

  kb25-neo4j:
    image: neo4j:5-community
    container_name: kb25-neo4j
    environment:
      NEO4J_AUTH: "neo4j/kb25_lkg_password"
      NEO4J_PLUGINS: '["apoc"]'
      NEO4J_server_default__database: "lkg"
    ports:
      - "7476:7474"    # Avoids conflict with shared infra Neo4j (7474) and dedicated (7475)
      - "7689:7687"    # Avoids conflict with shared infra Neo4j (7687) and dedicated (7688)
    volumes:
      - kb25_neo4j_data:/data
      - ./migrations:/migrations:ro
      - ./seed:/seed:ro
    healthcheck:
      test: ["CMD-SHELL", "cypher-shell -u neo4j -p kb25_lkg_password 'RETURN 1'"]
      interval: 30s
      timeout: 10s
      start_period: 30s
      retries: 5
    networks:
      - kb25-network

  kb25-redis:
    image: redis:7-alpine
    container_name: kb25-redis
    ports:
      - "6389:6379"    # Avoids conflict with KB-22 (6386), KB-23 (6388)
    command: redis-server --appendonly yes
    volumes:
      - kb25_redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      start_period: 10s
      retries: 3
    networks:
      - kb25-network

volumes:
  kb25_neo4j_data:
  kb25_redis_data:

networks:
  kb25-network:
    driver: bridge
  kb-network:
    external: true
```

### 2.9 Dependencies (go.mod)

```
github.com/neo4j/neo4j-go-driver/v5    # Neo4j Bolt driver (NEW to platform)
github.com/gin-gonic/gin v1.9.1
github.com/redis/go-redis/v9 v9.5.1
github.com/google/uuid v1.6.0
github.com/prometheus/client_golang
go.uber.org/zap v1.27.0
```

---

## 3. KB-26: Metabolic Digital Twin

### 3.1 Service Identity (from spec §3.2)

- **Service Name**: KB-26: Metabolic Digital Twin (MDT)
- **Port**: 8137
- **Database**: PostgreSQL (time-series twin state) + Redis (current state cache)
- **API Runtime**: Go 1.22+
- **Primary Consumers**: KB-25, KB-23, KB-20, M3-PRP/VFRP, Tier-1
- **Data Sources**: KB-20 (raw observations), KB-21 (behavioral signals), KB-25 (causal attribution), V-MCU events
- **Update Trigger**: Event-driven (new KB-20 observation, KB-21 check-in, KB-25 attribution, V-MCU med change)

### 3.2 What KB-26 Does NOT Duplicate (spec §1.1)

| Capability | Lives In | KB-26 Duplicates? |
|------------|----------|-------------------|
| Raw observation storage | KB-20 | NO — consumes KB-20 |
| Trajectory classification (GREEN/YELLOW/RED) | KB-20 | NO — KB-20's job |
| Dosing-specific metabolic state (ISF, dose cooldown) | KB-24/V-MCU | NO — V-MCU-private |
| Population-level causal chains | KB-25 | NO — KB-25's knowledge model |
| Behavioral signals (adherence, meal quality) | KB-21 | NO — consumes KB-21 |

### 3.3 The Three Genuine Gaps KB-26 Fills

1. **Persisted Derived State**: Translates raw labs into physiological understanding, tracked as time-series
2. **Coupled Forward Simulation**: Models feedback loops (VF↓ → IS↑ → insulin demand↓ → weight gain↓ → VF↓)
3. **Bayesian Patient-Specific Calibration**: Adjusts KB-25 population effects to individual observed responses

### 3.4 Directory Structure

```
kb-26-metabolic-digital-twin/
├── main.go
├── go.mod / go.sum
├── Dockerfile
├── docker-compose.yml
├── migrations/
│   └── 001_schema.sql
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── api/
│   │   ├── server.go
│   │   ├── routes.go                     # 7 endpoints (spec §8.1)
│   │   ├── twin_handlers.go
│   │   ├── simulation_handlers.go
│   │   └── calibration_handlers.go
│   ├── database/
│   │   └── connection.go                 # GORM Postgres
│   ├── models/
│   │   ├── twin_state.go                 # MetabolicTwinState (spec §4.1)
│   │   ├── estimated_variable.go         # EstimatedVariable with confidence
│   │   ├── calibrated_effect.go          # CalibratedEffect (Bayesian posterior)
│   │   ├── simulation.go                 # SimState, ProjectedState, Intervention
│   │   └── events.go
│   ├── services/
│   │   ├── twin_updater.go              # Event-driven updates from KB-20/21
│   │   ├── tier2_deriver.go             # 6 Tier 2 derivation formulas (spec §5.1)
│   │   ├── tier3_estimator.go           # 5 Tier 3 estimation methods (spec §5.2)
│   │   ├── simulation_engine.go         # Coupled forward simulation (spec §6)
│   │   ├── coupling_equations.go        # 6 coupling functions (spec §6.3)
│   │   ├── biomarker_computer.go        # FBG/PPBG/SBP/Waist/eGFR/HbA1c from SimState
│   │   ├── bayesian_calibrator.go       # 6-step calibration pipeline (spec §7)
│   │   └── confidence_analyzer.go       # What measurements would improve estimates
│   ├── clients/
│   │   ├── kb20_client.go
│   │   ├── kb21_client.go
│   │   └── kb25_client.go
│   ├── cache/
│   │   └── redis.go
│   └── metrics/
│       └── collector.go
└── tests/
    ├── integration/
    │   └── twin_update_test.go
    └── unit/
        ├── tier2_deriver_test.go
        ├── simulation_engine_test.go
        └── bayesian_calibrator_test.go
```

### 3.5 API Endpoints (spec §8.1)

| Endpoint | Method | Purpose | Consumer |
|----------|--------|---------|----------|
| `/api/v1/kb26/twin/{patientId}` | GET | Current MetabolicTwinState (all tiers + confidence) | KB-23, KB-25 |
| `/api/v1/kb26/twin/{patientId}/history` | GET | Time-series of twin state updates | KB-23, analytics |
| `/api/v1/kb26/simulate` | POST | Coupled forward simulation for proposed intervention | KB-25, M3-PRP/VFRP |
| `/api/v1/kb26/simulate-comparison` | POST | Multi-intervention simultaneous simulation | KB-23, V-MCU |
| `/api/v1/kb26/calibrate` | POST | Trigger Bayesian calibration with new observation | KB-25, internal |
| `/api/v1/kb26/twin/{patientId}/confidence` | GET | Confidence analysis + recommended measurements | KB-23, physician UI |
| `/api/v1/kb26/health` | GET | Service health check | Infrastructure |
| `/metrics` | GET | Prometheus metrics | Infrastructure |

### 3.6 Twin State Variable Tiers (spec §4.1)

#### Tier 1: Directly Measured (HIGH confidence)
From KB-20 observations. Full clinical use.
- FBG7dMean, FBG14dTrend, PPBG7dMean, HbA1c, SBP14dMean, DBP14dMean
- eGFR, WaistCm, WeightKg, BMI, DailySteps7dMean, RestingHR

#### Tier 2: Reliably Derived (MODERATE-HIGH confidence)
Computed from Tier 1 via validated formulas. Clinical use with "derived" tag.

| Variable | Formula | Source |
|----------|---------|--------|
| VisceralFatProxy | 0.5×norm(waist) + 0.3×norm(waist/height) + 0.2×norm(TG/HDL) | waist, height, lipids |
| RenalTrajectory | eGFR slope via linear regression (12mo) | eGFR history |
| MAP | DBP + (SBP-DBP)/3 | BP readings |
| GlycemicVariability | FBG CV% over 14 days | FBG values |
| DawnPhenomenon | FBG > PPBG in ≥3/5 paired readings AND FBG > 130 | FBG/PPBG pairs |
| TrigHDLRatio | TG/HDL (<2.0 low, 2.0-3.5 moderate, >3.5 high risk) | Lipid panel |

#### Tier 3: Estimated (LOW-MODERATE confidence)
Advisory use only. Cannot drive medication decisions. Displayed with "estimated" flag + confidence.

| Variable | Method | Initial Confidence |
|----------|--------|-------------------|
| InsulinSensitivity | HOMA-IR (if insulin available) or trajectory-based fallback | 0.30-0.75 |
| HepaticGlucoseOutput | Dawn phenomenon + FBG/PPBG ratio classification | 0.50-0.80 |
| MuscleMassProxy | Weight + protein + exercise + grip composite | 0.25-0.70 |
| BetaCellFunction | Medication response classification | 0.30-0.70 |
| SympatheticTone | Resting HR + BP variability binary classification | 0.40-0.55 |

### 3.7 Coupled Simulation Variables (spec §6.1)

| Symbol | Variable | Primary Coupling |
|--------|----------|-----------------|
| IS | Insulin Sensitivity | IS↑ → FBG↓, IS↑ → insulin demand↓ → VF↓ (feedback loop) |
| VF | Visceral Fat | VF↑ → IS↓, VF↑ → HGO↑, VF↑ → BP↑ |
| HGO | Hepatic Glucose Output | HGO↑ → FBG↑ (fasting), HGO↑ → insulin demand↑ |
| MM | Muscle Mass | MM↑ → IS↑, MM↑ → glucose disposal↑, MM↑ → BMR↑ → VF↓ |
| VR | Vascular Resistance | VR↑ → SBP↑, VR↑ → cardiac risk↑ |
| RR | Renal Reserve | RR↓ → drug clearance↓, RR↓ → protein cap↓, RR↓ → K+ risk↑ |

### 3.8 Bayesian Calibration Pipeline (spec §7.1)

1. **Observation Window**: 14-day window with single intervention change (from KB-20)
2. **Attribution Query**: KB-25 `/attribute-outcome` returns intervention-specific delta
3. **Prior**: KB-25 population effect → Normal(μ=population_effect, σ=CI_width/3.92)
4. **Likelihood**: Observed patient effect as likelihood function
5. **Posterior Update**: Bayes theorem; with 1 observation stays near prior, 5+ converges
6. **Store**: `CalibratedEffect{PopulationEffect, PatientEffect, Observations, Confidence}`

**Critical note**: 12-week "observe-only" burn-in after deployment. Calibrations computed but not used for clinical decisions until validated.

### 3.9 Docker Compose

```yaml
services:
  kb-26-mdt:
    build: .
    container_name: kb-26-mdt
    restart: unless-stopped
    ports:
      - "8137:8137"
    environment:
      PORT: "8137"
      DATABASE_URL: "postgres://kb_user:kb26_password@kb26-postgres:5432/kb26_mdt?sslmode=disable"
      REDIS_URL: "redis://kb26-redis:6379"
      KB20_URL: "http://kb-20-patient-profile:8131"
      KB21_URL: "http://kb-21-behavioral-intelligence:8133"
      KB25_URL: "http://kb-25-lkg:8136"
    depends_on:
      kb26-postgres:
        condition: service_healthy
      kb26-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--spider", "http://localhost:8137/health"]
      interval: 30s
      timeout: 5s
      start_period: 30s
      retries: 3
    networks:
      - kb26-network
      - kb-network

  kb26-postgres:
    image: postgres:15-alpine
    container_name: kb26-postgres
    environment:
      POSTGRES_USER: kb_user
      POSTGRES_PASSWORD: kb26_password
      POSTGRES_DB: kb26_mdt
    ports:
      - "5440:5432"    # Avoids conflict with KB-22 (5437), KB-23 (5439)
    volumes:
      - kb26_pgdata:/var/lib/postgresql/data
      - ./migrations/001_schema.sql:/docker-entrypoint-initdb.d/001_schema.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kb_user -d kb26_mdt"]
      interval: 10s
      timeout: 5s
      start_period: 10s
      retries: 5
    networks:
      - kb26-network

  kb26-redis:
    image: redis:7-alpine
    container_name: kb26-redis
    ports:
      - "6391:6379"    # Avoids conflict with KB-22 (6386), KB-23 (6388), KB-25 (6389)
    command: redis-server --appendonly yes
    volumes:
      - kb26_redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      start_period: 10s
      retries: 3
    networks:
      - kb26-network

volumes:
  kb26_pgdata:
  kb26_redis_data:

networks:
  kb26-network:
    driver: bridge
  kb-network:
    external: true
```

### 3.10 Event Transport for Twin Updates

KB-26 receives updates via **HTTP webhook POST** from source services (matching KB-20/21 event outbox pattern):

- **KB-20 → KB-26**: KB-20 publishes observation events to its transactional outbox. KB-26 registers as a webhook subscriber at KB-20 startup. On new lab/vital/anthropometry observation, KB-20 POSTs to `POST /api/v1/kb26/events/observation`.
- **KB-21 → KB-26**: KB-21 publishes check-in events. KB-26 subscribes via `POST /api/v1/kb26/events/checkin`.
- **KB-25 → KB-26**: After attribution computation, KB-25 calls `POST /api/v1/kb26/calibrate` directly (already specified).
- **V-MCU → KB-26**: Medication change events via `POST /api/v1/kb26/events/med-change`.

**Fallback**: If webhook delivery fails, KB-26 has a polling endpoint `GET /api/v1/kb26/sync/{patientId}` that fetches latest state from KB-20/21 on demand.

### 3.11 Database Schema (migrations/001_schema.sql)

```sql
-- Twin state snapshots (time-series)
CREATE TABLE twin_states (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL,
    state_version   INT NOT NULL,
    update_source   VARCHAR(50) NOT NULL,  -- KB20_OBSERVATION | KB21_CHECKIN | KB25_ATTRIBUTION | VMCU_MED_CHANGE
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Tier 1: Directly Measured
    fbg_7d_mean         FLOAT,
    fbg_14d_trend       VARCHAR(20),    -- IMPROVING | STABLE | WORSENING
    ppbg_7d_mean        FLOAT,
    hba1c               FLOAT,
    hba1c_date          TIMESTAMPTZ,
    sbp_14d_mean        FLOAT,
    dbp_14d_mean        FLOAT,
    egfr                FLOAT,
    egfr_date           TIMESTAMPTZ,
    waist_cm            FLOAT,
    weight_kg           FLOAT,
    bmi                 FLOAT,
    daily_steps_7d_mean FLOAT,
    resting_hr          FLOAT,

    -- Tier 2: Reliably Derived
    visceral_fat_proxy  FLOAT,
    visceral_fat_trend  VARCHAR(20),
    renal_slope         FLOAT,
    renal_classification VARCHAR(30),   -- STABLE | DECLINING | RAPIDLY_DECLINING | INSUFFICIENT_DATA
    map_value           FLOAT,
    glycemic_variability FLOAT,
    dawn_phenomenon     BOOLEAN,
    protein_adequacy    FLOAT,
    diet_quality_score  FLOAT,
    exercise_compliance FLOAT,
    trig_hdl_ratio      FLOAT,

    -- Tier 3: Estimated (JSONB for flexibility — each has value, classification, confidence, method)
    insulin_sensitivity   JSONB,
    hepatic_glucose_output JSONB,
    muscle_mass_proxy     JSONB,
    beta_cell_function    JSONB,
    sympathetic_tone      JSONB,

    UNIQUE(patient_id, state_version)
);
CREATE INDEX idx_twin_states_patient ON twin_states(patient_id, updated_at DESC);

-- Calibrated effects per patient (Bayesian posteriors)
CREATE TABLE calibrated_effects (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id        UUID NOT NULL,
    kb25_edge_type    VARCHAR(50) NOT NULL,
    intervention_code VARCHAR(50) NOT NULL,
    target_variable   VARCHAR(50) NOT NULL,
    population_effect FLOAT NOT NULL,
    patient_effect    FLOAT NOT NULL,
    observations      INT NOT NULL DEFAULT 0,
    confidence        FLOAT NOT NULL DEFAULT 0.0,
    prior_mean        FLOAT,
    prior_sd          FLOAT,
    posterior_mean    FLOAT,
    posterior_sd      FLOAT,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(patient_id, intervention_code, target_variable)
);
CREATE INDEX idx_calibrated_patient ON calibrated_effects(patient_id);

-- Simulation run history (audit trail)
CREATE TABLE simulation_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL,
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    intervention    JSONB NOT NULL,
    projection_days INT NOT NULL,
    results         JSONB NOT NULL,         -- array of ProjectedState
    twin_state_id   UUID REFERENCES twin_states(id),
    requested_by    VARCHAR(50)             -- consumer service name
);
CREATE INDEX idx_simulation_patient ON simulation_runs(patient_id, requested_at DESC);
```

### 3.12 Dependencies (go.mod)

```
github.com/gin-gonic/gin v1.9.1
gorm.io/gorm v1.25.8
gorm.io/driver/postgres v1.5.7
github.com/redis/go-redis/v9 v9.5.1
github.com/google/uuid v1.6.0
github.com/prometheus/client_golang
go.uber.org/zap v1.27.0
gonum.org/v1/gonum                      # Numerical computation for simulation + Bayesian
```

---

## 4. Implementation Timeline (Sequential)

### KB-25 Phases (Weeks 1-7)

| Phase | Week | Deliverable | Key Files |
|-------|------|-------------|-----------|
| **1: Scaffold + Neo4j Schema** | 1 | Service boots, connects to Neo4j, /health works | main.go, config.go, server.go, graph/client.go, migrations/ |
| **2: Ontology Seed Data** | 1-2 | 887 nodes loaded, queryable | seed/*.cypher |
| **3: Causal Edges** | 2-3 | ~5,230 edges with EffectDescriptors, chain traversal working | models/effect_descriptor.go, services/chain_traversal.go, GET /causal-chain |
| **4: Effect Modifier Engine** | 3 | Patient-specific effect computation | services/effect_modifier.go, clients/kb20_client.go |
| **5: Safety Engine** | 3-4 | LS-01..LS-14 + drug interactions, POST /check-safety | services/safety_engine.go, clients/kb1_client.go, kb4_client.go |
| **6: Comparator Engine** | 4-5 | Full lifestyle vs pharma comparison, POST /compare-interventions | services/comparator_engine.go |
| **7: Recommendation + Attribution APIs** | 5-6 | All 10 endpoints operational | services/attribution_engine.go, exercise_rx.go, diet_quality.go |
| **8: Integration + Caching** | 6-7 | Redis caching, graceful degradation, Prometheus → **KB-25 DONE** | cache/redis.go, metrics/collector.go |

### KB-26 Phases (Weeks 8-19)

| Phase | Week | Deliverable | Key Files |
|-------|------|-------------|-----------|
| **1: Twin State (Tier 1+2)** | 8-10 | Twin populates from KB-20, Tier 2 derivations, GET /twin | models/twin_state.go, services/tier2_deriver.go, twin_updater.go |
| **2: Tier 3 Estimation** | 10-12 | 5 estimation modules with confidence, GET /confidence | services/tier3_estimator.go, confidence_analyzer.go |
| **3: Coupled Simulation** | 12-15 | What-if simulation, POST /simulate | services/simulation_engine.go, coupling_equations.go, biomarker_computer.go |
| **4: Bayesian Calibration** | 15-17 | Patient-specific calibration, POST /calibrate | services/bayesian_calibrator.go |
| **5: Consumer Integration** | 17-19 | KB-23/KB-25/M3/Tier-1 enrichment → **KB-26 DONE** | clients/kb25_client.go, all handler updates |

### Critical Path Notes

- **KB-25 Phase 3** (causal edges) is the bottleneck — every edge requires clinical evidence review
- **KB-26 Phase 4** (Bayesian calibration) requires real patient data — plan 12-week observe-only burn-in
- KB-26 Phase 3 (simulation) has a hard dependency on KB-25 being operational for causal chain data

---

## 5. Infrastructure Summary

| Service | Port | Postgres Port | Redis Port | Neo4j Ports | Docker Network |
|---------|------|---------------|------------|-------------|----------------|
| KB-25 | 8136 | — | 6389 | 7476/7689 | kb25-network + kb-network (external) |
| KB-26 | 8137 | 5440 | 6391 | — | kb26-network + kb-network (external) |

Both services follow existing KB conventions:
- Multi-stage Alpine Dockerfile with non-root user (UID 1001)
- `/health` endpoint + Docker HEALTHCHECK
- `/metrics` Prometheus endpoint
- Structured zap logging
- Graceful degradation when dependencies unavailable
- Shared `kb-network` (external) for cross-KB container-to-container communication via service names

---

## 6. Performance Targets (from specs)

### KB-25 (spec §13.2)
- Chain traversal: < 50ms (cached), < 200ms (uncached)
- Compare-interventions: < 500ms (full comparison with 3+ options)
- Safety check: < 100ms
- Food search: < 50ms

### KB-26
- Twin state GET: < 50ms (cached), < 200ms (recompute)
- Forward simulation (90 days): < 1s
- Simulate-comparison (3 options × 90 days): < 3s
- Bayesian calibration update: < 500ms
