# KB-22 Three-Layer Node Taxonomy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 15 new clinical nodes (9 PM physiological monitoring + 6 MD metabolic deterioration) to KB-22 HPI Engine, with event publishing to KB-23 and Kafka, KB-20/KB-26 data resolution, and internal PM→MD cascade.

**Architecture:** New code lives in isolated files within KB-22's existing `internal/` structure. A `SignalHandlerGroup` sub-router encapsulates all PM/MD dependencies. PM/MD YAML definitions go in new `monitoring/` and `deterioration/` directories. KB-23 gets a new `POST /api/v1/clinical-signals` endpoint with `SignalCardBuilder`. No existing KB-22 code is modified except two additive changes to `config.go` and `server.go`.

**Tech Stack:** Go 1.22, Gin, GORM, PostgreSQL, Redis, segmentio/kafka-go, gopkg.in/yaml.v3

**Spec:** `docs/superpowers/specs/2026-03-18-kb22-three-layer-node-taxonomy-design.md`

---

## File Structure

### KB-22 New Files

| File | Responsibility |
|------|---------------|
| `internal/models/monitoring_node.go` | `MonitoringNodeDefinition`, `ClassificationDef`, `RequiredInputDef`, `ComputedFieldDef`, `InsufficientDataPolicy`, `CheckinPromptDef` |
| `internal/models/deterioration_node.go` | `DeteriorationNodeDefinition`, `ThresholdDef`, `TrajectoryConfig`, `ProjectionDef` |
| `internal/models/clinical_signal_event.go` | `ClinicalSignalEvent`, `SignalType`, `ClassificationResult`, `DeteriorationResult`, `ThresholdProjection`, `RecommendedAction`, `MonitoringDataPoint`, `SignalSafetyFlag` |
| `internal/models/data_input.go` | `RequiredInput`, `AggregatedInputDef`, `ResolvedData`, `DataSufficiency`, `TwinStateView`, `EstimatedValue`, `TimeSeriesPoint` |
| `internal/services/expression_evaluator.go` | Recursive-descent parser for restricted condition/formula grammar |
| `internal/services/expression_evaluator_test.go` | Parser tests: arithmetic, comparison, AND/OR/NOT, field substitution, rejection of disallowed constructs |
| `internal/services/monitoring_node_loader.go` | `MonitoringNodeLoader` — parses `monitoring/*.yaml`, validates schema |
| `internal/services/monitoring_node_loader_test.go` | Loader validation: valid YAML, missing fields, bad conditions |
| `internal/services/deterioration_node_loader.go` | `DeteriorationNodeLoader` — parses `deterioration/*.yaml`, validates schema + DAG acyclicity |
| `internal/services/deterioration_node_loader_test.go` | Loader validation + DAG cycle detection |
| `internal/services/data_resolver.go` | `DataResolverImpl` — fetches from KB-20, KB-26, cache; JSONB unwrapping; staleness check |
| `internal/services/data_resolver_test.go` | Multi-source resolution, JSONB unwrap, insufficiency, staleness |
| `internal/services/kb26_client.go` | `KB26Client` — HTTP client for KB-26 twin state + history API |
| `internal/services/kb26_client_test.go` | Twin state fetch, history extraction, error handling |
| `internal/services/monitoring_engine.go` | `MonitoringNodeEngine` — evaluates PM nodes: resolve → compute → classify → safety → emit |
| `internal/services/monitoring_engine_test.go` | Classification logic per PM node category |
| `internal/services/trajectory_computer.go` | `TrajectoryComputer` — linear regression on time series, slope + confidence |
| `internal/services/trajectory_computer_test.go` | Regression accuracy, min data points, projection |
| `internal/services/deterioration_engine.go` | `DeteriorationNodeEngine` — evaluates MD nodes: resolve → trajectory → threshold → project → emit |
| `internal/services/deterioration_engine_test.go` | Trajectory + threshold evaluation |
| `internal/services/signal_cascade.go` | `SignalCascade` — two-pass PM→MD + MD→MD-06 coordinator |
| `internal/services/signal_cascade_test.go` | Cascade ordering, non-fatal failures, MD-06 second pass |
| `internal/services/signal_publisher.go` | `SignalPublisher` — POST to KB-23 + Kafka topic with retry |
| `internal/services/signal_publisher_test.go` | KB-23 call, Kafka publish, retry logic |
| `internal/api/signal_handler_group.go` | `SignalHandlerGroup` — sub-router DI container + route registration |
| `internal/api/event_ingestion_handlers.go` | `handleObservation`, `handleTwinStateUpdate`, `handleCheckinResponse` |
| `internal/api/signal_query_handlers.go` | `handleGetPatientSignals`, `handleGetSignalHistory`, `handleGetDeteriorationSummary` |
| `monitoring/` | 9 PM node YAML files |
| `deterioration/` | 6 MD node YAML files |
| `migrations/006_monitoring_deterioration.sql` | 3 new tables: `clinical_signals`, `clinical_signals_latest`, `signal_evaluation_log` |

### KB-22 Modified Files

| File | Change |
|------|--------|
| `internal/config/config.go` | Add ~8 new env vars (KB26_URL, timeouts, dirs, debounce, Kafka topic, staleness) |
| `internal/api/server.go` | Add `SignalHandlerGroup` field + init call + route registration (3 lines) |

### KB-23 New Files (cross-service)

| File | Responsibility |
|------|---------------|
| `internal/models/clinical_signal_event.go` | Copy of ClinicalSignalEvent types for deserialization |
| `internal/services/signal_card_builder.go` | `SignalCardBuilder.Build()` — template selection, confidence derivation, gate eval |
| `internal/services/signal_card_builder_test.go` | Template resolution, confidence mapping, HALT handling |
| `internal/api/signal_handlers.go` | `handleClinicalSignal` — POST /api/v1/clinical-signals |

### KB-23 Modified Files

| File | Change |
|------|--------|
| `internal/models/enums.go` | Add `CardSourceClinicalSignal = "CLINICAL_SIGNAL"` |
| `internal/api/routes.go` | Add `v1.POST("/clinical-signals", ...)` |
| `internal/api/server.go` | Add `SignalCardBuilder` field + init |
| `internal/services/hysteresis_engine.go` | Add HALT gate rules (5 rules from spec Section 4.4) |

---

## Task Breakdown

### Task 1: Database Migration

**Files:**
- Create: `kb-22-hpi-engine/migrations/006_monitoring_deterioration.sql`

This is the foundation — all engine code will persist to these tables.

- [ ] **Step 1: Write the migration SQL**

```sql
-- migrations/006_monitoring_deterioration.sql
-- KB-22 Three-Layer Node Taxonomy: clinical signal storage

CREATE TABLE IF NOT EXISTS clinical_signals (
    signal_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    node_version    VARCHAR(20) NOT NULL,
    signal_type     VARCHAR(30) NOT NULL,
    stratum_label   VARCHAR(100),

    -- PM node fields
    classification_category  VARCHAR(50),
    classification_value     DOUBLE PRECISION,
    classification_unit      VARCHAR(20),
    data_sufficiency         VARCHAR(20),

    -- MD node fields
    deterioration_signal     VARCHAR(80),
    severity                 VARCHAR(20),
    trajectory               VARCHAR(20),
    rate_of_change           DOUBLE PRECISION,
    state_variable           VARCHAR(10),

    -- Projection
    projected_threshold_name  VARCHAR(80),
    projected_threshold_date  TIMESTAMPTZ,
    projection_confidence     DOUBLE PRECISION,

    -- Shared
    resolved_data            JSONB,
    contributing_signals     JSONB,
    safety_flags             JSONB,
    mcu_gate_suggestion      VARCHAR(10),
    published_to_kb23        BOOLEAN DEFAULT FALSE,

    evaluated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signals_patient_node ON clinical_signals(patient_id, node_id, evaluated_at DESC);
CREATE INDEX idx_signals_patient_type ON clinical_signals(patient_id, signal_type, evaluated_at DESC);
CREATE INDEX idx_signals_severity ON clinical_signals(severity) WHERE severity IN ('SEVERE', 'CRITICAL');
CREATE INDEX idx_signals_unpublished ON clinical_signals(published_to_kb23) WHERE published_to_kb23 = FALSE;

CREATE TABLE IF NOT EXISTS clinical_signals_latest (
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    signal_id       UUID NOT NULL REFERENCES clinical_signals(signal_id),
    evaluated_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (patient_id, node_id)
);

CREATE TABLE IF NOT EXISTS signal_evaluation_log (
    patient_id      UUID NOT NULL,
    node_id         VARCHAR(50) NOT NULL,
    last_evaluated  TIMESTAMPTZ NOT NULL,
    last_trigger    VARCHAR(100),
    PRIMARY KEY (patient_id, node_id)
);
```

- [ ] **Step 2: Verify migration applies cleanly**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && psql "$DATABASE_URL" -f migrations/006_monitoring_deterioration.sql`
Expected: Three tables created, no errors. If DB not running, verify SQL syntax with `psql --echo-errors`.

- [ ] **Step 3: Commit**

```bash
git add kb-22-hpi-engine/migrations/006_monitoring_deterioration.sql
git commit -m "feat(kb-22): add migration 006 for clinical signals tables"
```

---

### Task 2: Core Model Types

**Files:**
- Create: `kb-22-hpi-engine/internal/models/clinical_signal_event.go`
- Create: `kb-22-hpi-engine/internal/models/data_input.go`
- Create: `kb-22-hpi-engine/internal/models/monitoring_node.go`
- Create: `kb-22-hpi-engine/internal/models/deterioration_node.go`

All engine code depends on these types. No business logic — pure data structures.

- [ ] **Step 1: Write ClinicalSignalEvent model**

Create `internal/models/clinical_signal_event.go` with the types from spec Section 3.1 and 3.3:
- `ClinicalSignalEvent` struct (header + Layer 2 + Layer 3 + shared fields)
- `SignalType` enum: `MONITORING_CLASSIFICATION`, `DETERIORATION_SIGNAL`
- `ClassificationResult`, `DeteriorationResult`, `ThresholdProjection`
- `RecommendedAction`, `MonitoringDataPoint`, `SignalSafetyFlag`

Reference spec lines 52-158 for exact field definitions. Use `json` tags. Use `string` for IDs (wire format), not `uuid.UUID`. Use `*Type` with `omitempty` for optional fields.

- [ ] **Step 2: Write DataInput model**

Create `internal/models/data_input.go` with types from spec Section 8.4 and 5.4:
- `RequiredInput` struct: `Field`, `Source`, `Unit`, `MinObservations`, `LookbackDays`, `Optional`, `Description`
- `AggregatedInputDef` struct: `Field string`, `Source string`, `LookbackDays int`, `Aggregation string` (MEAN/STDEV/COUNT/MAX/MIN/CV/RAW), `Optional bool`, `Description string`. This is used by PM-04 through PM-09 which need statistical summaries over time-series data (e.g., 30-day mean of daily steps, CV of glucose readings). DataResolver pre-computes aggregates and places scalar results into `ResolvedData.Fields`.
- `ResolvedData` struct: `Fields map[string]float64`, `TimeSeries map[string][]TimeSeriesPoint` (raw series for TrajectoryComputer/regression), `FieldTimestamps`, `Sufficiency`, `MissingFields`, `Sources`
- `DataSufficiency` enum: `SUFFICIENT`, `PARTIAL`, `INSUFFICIENT`
- `TwinStateView` struct with flattened values (IS, HGO, MM as `EstimatedValue`; VF as float64; VR, RR as `EstimatedValue`)
- `EstimatedValue` struct: `Value float64`, `Confidence float64`
- `TimeSeriesPoint` struct: `Timestamp time.Time`, `Value float64`

**Design note (reviewer Issue 6/12):** Several PM nodes require array/time-series data:
- PM-04: `ppbg_values` array for excursion calculation
- PM-05: glucose readings array → `stdev()` and `mean()` for CV computation
- PM-06: FBG history → linear regression for slope
- PM-07: 30-day step data → mean
- PM-08: 30-day RMSSD data → mean
- PM-09: sleep quality responses → `normalize()` composite

The `AggregatedInputDef` allows YAML to declare what aggregation to apply. DataResolver fetches the time series, computes the aggregate, and stores the scalar result in `Fields`. For PM-06 which needs raw series for regression, DataResolver also stores the raw series in `TimeSeries` for `TrajectoryComputer` consumption.

- [ ] **Step 3: Write MonitoringNodeDefinition model**

Create `internal/models/monitoring_node.go` with types from spec Section 7.1:
- `MonitoringNodeDefinition`: `NodeID`, `Version`, `Type` (always "MONITORING"), `TitleEN`, `TitleHI`, `RequiredInputs []RequiredInput`, `AggregatedInputs []AggregatedInputDef` (for PM-04 through PM-09 which need pre-computed aggregates from time-series data), `ComputedFields []ComputedFieldDef`, `Classifications []ClassificationDef`, `InsufficientData InsufficientDataPolicy`, `SafetyTriggers []MonitoringSafetyTrigger`, `CascadeTo []string`, `CheckinPrompts []CheckinPromptDef`
- `ComputedFieldDef`: `Name string`, `Formula string`
- `ClassificationDef`: `Category`, `Condition`, `Severity`, `MCUGateSuggestion`, `CardTemplate`
- `InsufficientDataPolicy`: `Action` (SKIP/FLAG_FOR_REVIEW), `NoteEN`, `Fallback`
- `MonitoringSafetyTrigger`: `ID`, `Condition`, `Severity`, `Action`
- `CheckinPromptDef`: `PromptID`, `TextEN`, `TextHI`, `ResponseType`, `MapsTo`

- [ ] **Step 4: Write DeteriorationNodeDefinition model**

Create `internal/models/deterioration_node.go` with types from spec Section 7.2:
- `DeteriorationNodeDefinition`: `NodeID`, `Version`, `Type` (always "DETERIORATION"), `TitleEN`, `TitleHI`, `StateVariable`, `StateVariableLabel`, `TriggerOn []TriggerDef`, `RequiredInputs []RequiredInput`, `AggregatedInputs []AggregatedInputDef`, `ComputedFields []ComputedFieldDef` (for MD-04/MD-05 composite scores with adaptive weights), `ContributingSignals []string`, `Trajectory TrajectoryConfig`, `Thresholds []ThresholdDef`, `Projections []ProjectionDef`, `InsufficientData InsufficientDataPolicy`
- `TriggerDef`: `Event string` (e.g., "OBSERVATION:FBG", "SIGNAL:PM-04", "PROTOCOL:M3-PRP:ADHERENCE")
- `TrajectoryConfig`: `Method` (LINEAR_REGRESSION), `WindowDays`, `MinDataPoints`, `RateUnit`, `DataSource`
- `ThresholdDef`: `Signal`, `Condition`, `Severity`, `Trajectory`, `MCUGateSuggestion`, `CardTemplate`, `Actions []RecommendedAction`
- `ProjectionDef`: `Name`, `Variable`, `Threshold float64`, `Method`, `ConfidenceRequired float64`

- [ ] **Step 5: Verify build compiles**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 6: Commit**

```bash
git add kb-22-hpi-engine/internal/models/clinical_signal_event.go \
       kb-22-hpi-engine/internal/models/data_input.go \
       kb-22-hpi-engine/internal/models/monitoring_node.go \
       kb-22-hpi-engine/internal/models/deterioration_node.go
git commit -m "feat(kb-22): add model types for clinical signals, data input, PM/MD node definitions"
```

---

### Task 3: Expression Evaluator

**Files:**
- Create: `kb-22-hpi-engine/internal/services/expression_evaluator.go`
- Test: `kb-22-hpi-engine/internal/services/expression_evaluator_test.go`

The expression evaluator is used by both PM and MD engines to evaluate YAML `condition` and `formula` strings. Build it first so engines can depend on it.

- [ ] **Step 1: Write failing tests for expression evaluator**

Create `expression_evaluator_test.go` with test cases:
```go
func TestExpressionEvaluator_Arithmetic(t *testing.T) {
    // "(sbp_nocturnal_mean - sbp_daytime_mean) / sbp_daytime_mean"
    // fields: {sbp_nocturnal_mean: 130, sbp_daytime_mean: 140}
    // expected: (130 - 140) / 140 = -0.0714...
}

func TestExpressionEvaluator_Comparison(t *testing.T) {
    // "dipping_ratio > 0.0" with dipping_ratio = -0.05 → false
    // "dipping_ratio > 0.0" with dipping_ratio = 0.05 → true
}

func TestExpressionEvaluator_LogicalAND(t *testing.T) {
    // "rate_of_change < -0.08 AND twin_state.IS < 0.30"
    // fields: {rate_of_change: -0.10, "twin_state.IS": 0.25} → true
}

func TestExpressionEvaluator_LogicalOR(t *testing.T) {
    // "a > 1 OR b > 1" with a=0.5, b=2.0 → true
}

func TestExpressionEvaluator_RejectsDisallowed(t *testing.T) {
    // "os.Exit(1)" → error (non-whitelisted function)
    // "import fmt" → error
}

func TestExpressionEvaluator_FieldSubstitution(t *testing.T) {
    // "sbp_home_mean - bp_target_sbp"
    // fields: {sbp_home_mean: 150, bp_target_sbp: 130} → 20.0
}

func TestExpressionEvaluator_BuiltinNormalize(t *testing.T) {
    // "normalize(hba1c, 6, 12)" with hba1c=9 → (9-6)/(12-6) = 0.5
    // Clamps to [0, 1]: normalize(hba1c, 6, 12) with hba1c=13 → 1.0
}

func TestExpressionEvaluator_BuiltinAbs(t *testing.T) {
    // "abs(slope)" with slope=-0.05 → 0.05
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go test ./internal/services/ -run TestExpressionEvaluator -v`
Expected: FAIL — functions not defined

- [ ] **Step 3: Implement expression evaluator**

Create `expression_evaluator.go` with a recursive-descent parser:
- `ExpressionEvaluator` struct
- `EvaluateNumeric(expr string, fields map[string]float64) (float64, error)` — returns numeric result
- `EvaluateBool(expr string, fields map[string]float64) (bool, error)` — returns boolean result
- Tokenizer: splits on whitespace, operators (`+`, `-`, `*`, `/`, `(`, `)`, `AND`, `OR`, `NOT`, `>`, `>=`, `<`, `<=`, `==`, `!=`)
- Parser: `parseOr → parseAnd → parseNot → parseComparison → parseAddSub → parseMulDiv → parseUnary → parsePrimary`
- `parsePrimary`: numeric literal, string literal, field name lookup in `fields` map, or **whitelisted function call**
- **Built-in function whitelist** (reviewer Issue 4): `normalize(value, min, max)` → clamped `(value-min)/(max-min)` in [0,1]; `abs(value)` → absolute value. These are the only functions allowed — all others are rejected. Function detection: if token is `word(`, check whitelist; if not in whitelist, return parse error.
- Validation: reject non-whitelisted function calls and keywords (`import`, `func`, `go`, `return`)

**Design note (reviewer Issue 9):** Boolean check-in inputs (PM-02 symptoms) are resolved as 1.0/0.0 by DataResolver. PM-02's formula uses pure arithmetic `symptom_headache + symptom_dizziness + symptom_chest_pain + symptom_fatigue + symptom_oedema` instead of `sum()`. No array functions needed in the evaluator — DataResolver pre-computes all aggregates via `AggregatedInputDef`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go test ./internal/services/ -run TestExpressionEvaluator -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/internal/services/expression_evaluator.go \
       kb-22-hpi-engine/internal/services/expression_evaluator_test.go
git commit -m "feat(kb-22): add restricted expression evaluator for PM/MD node conditions"
```

---

### Task 4: Configuration Extensions

**Files:**
- Modify: `kb-22-hpi-engine/internal/config/config.go`

- [ ] **Step 1: Add new config fields**

Add to `Config` struct (after existing fields, before closing brace):
```go
// KB-26 Metabolic Digital Twin
KB26URL string

// PM/MD Node directories
MonitoringNodesDir    string
DeteriorationNodesDir string

// Signal evaluation
KB26TimeoutMS             int
KB20ObservationTimeoutMS  int
SignalDebounceTTLSec      int
SignalPublisherRetryCount int
SignalPublisherRetryDelaySec int
KafkaSignalTopic          string
KB26StalenessDays         int
```

Add to `Load()` function (after existing assignments):
```go
KB26URL:                    envOrDefault("KB26_URL", "http://localhost:8137"),
MonitoringNodesDir:         envOrDefault("MONITORING_NODES_DIR", "./monitoring"),
DeteriorationNodesDir:      envOrDefault("DETERIORATION_NODES_DIR", "./deterioration"),
KB26TimeoutMS:              envIntOrDefault("KB26_TIMEOUT_MS", 5000),
KB20ObservationTimeoutMS:   envIntOrDefault("KB20_OBSERVATION_TIMEOUT_MS", 10000),
SignalDebounceTTLSec:       envIntOrDefault("SIGNAL_DEBOUNCE_TTL_SEC", 300),
SignalPublisherRetryCount:     envIntOrDefault("SIGNAL_PUBLISHER_RETRY_COUNT", 3),
SignalPublisherRetryDelaySec:  envIntOrDefault("SIGNAL_PUBLISHER_RETRY_DELAY_SEC", 30),
KafkaSignalTopic:           envOrDefault("KAFKA_SIGNAL_TOPIC", "clinical.signal.events"),
KB26StalenessDays:          envIntOrDefault("KB26_STALENESS_DAYS", 21),
```

Add timeout helper methods:
```go
func (c *Config) KB26Timeout() time.Duration {
    return time.Duration(c.KB26TimeoutMS) * time.Millisecond
}
func (c *Config) KB20ObservationTimeout() time.Duration {
    return time.Duration(c.KB20ObservationTimeoutMS) * time.Millisecond
}
func (c *Config) KB26StalenessThreshold() time.Duration {
    return time.Duration(c.KB26StalenessDays) * 24 * time.Hour
}
```

- [ ] **Step 2: Verify build compiles and existing tests pass**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go build ./... && go test ./internal/config/ -v`
Expected: BUILD SUCCESS, existing tests PASS

- [ ] **Step 3: Commit**

```bash
git add kb-22-hpi-engine/internal/config/config.go
git commit -m "feat(kb-22): add KB-26, PM/MD node config for three-layer taxonomy"
```

---

### Task 5: KB-26 Client

**Files:**
- Create: `kb-22-hpi-engine/internal/services/kb26_client.go`
- Test: `kb-22-hpi-engine/internal/services/kb26_client_test.go`

The DataResolver depends on this client to fetch twin state and variable history.

- [ ] **Step 1: Write failing tests for KB-26 client**

Test cases:
- `TestKB26Client_GetTwinState`: Mock HTTP server returns KB-26 JSON with JSONB fields → verify `TwinStateView` has unwrapped float values
- `TestKB26Client_GetTwinState_NilTier3`: Tier 3 fields are null → `EstimatedValue` has zero Value and zero Confidence
- `TestKB26Client_GetVariableHistory`: Mock returns 10 snapshots → extract IS time series → verify 10 `TimeSeriesPoint` values
- `TestKB26Client_GetTwinState_Timeout`: Mock delays 10s → verify context deadline error
- `TestKB26Client_VRFallback`: VascularResistance field is null → verify VR derived from MAP: `VR = MAP / 80.0` (simplified proxy)
- `TestKB26Client_RRFallback`: RenalReserve field is null → verify RR derived from eGFR: `RR = eGFR / 120.0` (normalized)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestKB26Client -v`
Expected: FAIL

- [ ] **Step 3: Implement KB-26 client**

Create `kb26_client.go`:
```go
type KB26Client struct {
    baseURL    string
    httpClient *http.Client
    log        *zap.Logger
}

func NewKB26Client(baseURL string, timeout time.Duration, log *zap.Logger) *KB26Client

func (c *KB26Client) GetTwinState(ctx context.Context, patientID string) (*models.TwinStateView, error)
// GET /api/v1/kb26/twin/{patientID}
// Deserializes JSONB EstimatedVariable fields into EstimatedValue
// Applies VR/RR fallback derivation when KB-26 fields are null (spec Section 5.3)
// Sets LastUpdated from response's UpdatedAt

func (c *KB26Client) GetVariableHistory(ctx context.Context, patientID, variable string, days int) ([]models.TimeSeriesPoint, error)
// GET /api/v1/kb26/twin/{patientID}/history?limit={days}
// Extracts specific variable from each snapshot into time series

// Helper: extractEstimated parses JSONB → EstimatedValue
func extractEstimated(raw json.RawMessage) models.EstimatedValue

// Helper: extractEstimatedOrDerive handles VR/RR fallback
func extractEstimatedOrDerive(resp *twinStateResponse, variable string) models.EstimatedValue
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestKB26Client -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/internal/services/kb26_client.go \
       kb-22-hpi-engine/internal/services/kb26_client_test.go
git commit -m "feat(kb-22): add KB-26 client with JSONB unwrapping and VR/RR fallback"
```

---

### Task 6: Data Resolver

**Files:**
- Create: `kb-22-hpi-engine/internal/services/data_resolver.go`
- Test: `kb-22-hpi-engine/internal/services/data_resolver_test.go`

Central data-fetching abstraction used by both PM and MD engines.

- [ ] **Step 1: Write failing tests for DataResolver**

Test cases:
- `TestDataResolver_KB20Source`: RequiredInput with `source: KB-20` → calls KB-20 client → returns field in `ResolvedData.Fields`
- `TestDataResolver_KB26Source`: RequiredInput with `source: KB-26` → calls KB-26 client → unwraps JSONB → returns field
- `TestDataResolver_MissingRequiredField`: Required field not returned → `Sufficiency = INSUFFICIENT`, field in `MissingFields`
- `TestDataResolver_MissingOptionalField`: Optional field not returned → `Sufficiency = SUFFICIENT`, field in `MissingFields`
- `TestDataResolver_Staleness`: KB-26 twin state `LastUpdated` is 30 days ago → `Sufficiency` downgraded from SUFFICIENT to PARTIAL
- `TestDataResolver_MultipleSources`: Mix of KB-20 and KB-26 fields → all resolved correctly
- `TestDataResolver_FieldTimestamps`: Resolved data includes timestamps per field
- `TestDataResolver_AggregatedInput_Mean`: AggregatedInputDef with `aggregation: MEAN`, 30 daily step values → `Fields["daily_steps_30d_mean"]` = arithmetic mean
- `TestDataResolver_AggregatedInput_CV`: AggregatedInputDef with `aggregation: CV`, 14 glucose readings → `Fields["glucose_cv"]` = stdev/mean*100
- `TestDataResolver_AggregatedInput_RAW`: AggregatedInputDef with `aggregation: RAW` → raw series stored in `TimeSeries["fbg_values"]` for TrajectoryComputer
- `TestDataResolver_BooleanConversion`: TIER1_CHECKIN boolean → resolved as 1.0/0.0 in Fields

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestDataResolver -v`
Expected: FAIL

- [ ] **Step 3: Implement DataResolver**

Create `data_resolver.go`:
```go
type DataResolver interface {
    Resolve(ctx context.Context, patientID string, inputs []models.RequiredInput, aggInputs []models.AggregatedInputDef) (*models.ResolvedData, error)
}

type DataResolverImpl struct {
    kb20BaseURL        string
    kb20Client         *http.Client
    kb26Client         *KB26Client
    cache              *cache.CacheClient
    stalenessThreshold time.Duration
    log                *zap.Logger
}

func NewDataResolver(cfg *config.Config, kb26Client *KB26Client, cacheClient *cache.CacheClient, log *zap.Logger) *DataResolverImpl

func (r *DataResolverImpl) Resolve(ctx context.Context, patientID string, inputs []models.RequiredInput, aggInputs []models.AggregatedInputDef) (*models.ResolvedData, error)
// 1. Group inputs by source (KB-20, KB-26, DEVICE, TIER1_CHECKIN)
// 2. Fetch KB-20 labs/observations (GET /api/v1/patient/{id}/labs?type={type}&days={lookback})
// 3. Fetch KB-26 twin state via kb26Client.GetTwinState()
// 4. Check staleness of twin state (spec Section 5.5)
// 5. For each input, map to resolved field or add to MissingFields
// 6. **Process AggregatedInputDefs (reviewer Issue 6/12)**:
//    a. For each aggInput, fetch time-series data from source (KB-20 history or KB-26 history)
//    b. Apply aggregation: MEAN→arithmetic mean, STDEV→sample stdev, CV→stdev/mean*100,
//       COUNT→len, MAX→max, MIN→min, RAW→store in TimeSeries (for TrajectoryComputer)
//    c. Store scalar result in Fields[aggInput.Field]; store raw series in TimeSeries[aggInput.Field] if RAW
//    d. Boolean check-in inputs resolved as 1.0 (true) / 0.0 (false)
// 7. Compute DataSufficiency: INSUFFICIENT if any non-optional required field missing,
//    PARTIAL if optional fields missing or stale twin state, SUFFICIENT otherwise
// 8. Record source used per field in Sources map
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestDataResolver -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/internal/services/data_resolver.go \
       kb-22-hpi-engine/internal/services/data_resolver_test.go
git commit -m "feat(kb-22): add DataResolver with KB-20/KB-26 multi-source resolution"
```

---

### Task 7: Trajectory Computer

**Files:**
- Create: `kb-22-hpi-engine/internal/services/trajectory_computer.go`
- Test: `kb-22-hpi-engine/internal/services/trajectory_computer_test.go`

Used by MD engine for linear regression on variable history.

- [ ] **Step 1: Write failing tests**

Test cases:
- `TestTrajectoryComputer_LinearRegression`: 10 equally spaced points with known slope → verify computed slope matches within 0.001
- `TestTrajectoryComputer_MinDataPoints`: 3 points but min_data_points=5 → returns error
- `TestTrajectoryComputer_Projection`: slope=-0.05/month, current=0.35, threshold=0.20 → projected date ~3 months from now
- `TestTrajectoryComputer_StableTrajectory`: Flat data → slope near 0.0
- `TestTrajectoryComputer_Confidence`: Returns R² as confidence measure

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestTrajectoryComputer -v`
Expected: FAIL

- [ ] **Step 3: Implement trajectory computer**

```go
type TrajectoryComputer struct {
    log *zap.Logger
}

type TrajectoryResult struct {
    Slope      float64   // rate of change in rate_unit
    Intercept  float64
    RSquared   float64   // goodness of fit (0-1)
    DataPoints int
}

func (tc *TrajectoryComputer) Compute(series []models.TimeSeriesPoint, cfg models.TrajectoryConfig) (*TrajectoryResult, error)
// 1. Check len(series) >= cfg.MinDataPoints
// 2. Convert timestamps to numeric x (days from first point)
// 3. Simple linear regression: slope = Σ(xi-x̄)(yi-ȳ) / Σ(xi-x̄)²
// 4. Convert slope to rate_unit (e.g., per_month = slope * 30)
// 5. Compute R² for confidence

func (tc *TrajectoryComputer) Project(current float64, slope float64, threshold float64) (*time.Time, error)
// Linear extrapolation: days_to_threshold = (threshold - current) / slope
// Only valid if slope is moving toward threshold
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestTrajectoryComputer -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/internal/services/trajectory_computer.go \
       kb-22-hpi-engine/internal/services/trajectory_computer_test.go
git commit -m "feat(kb-22): add TrajectoryComputer with linear regression and projection"
```

---

### Task 8: Node Loaders (Monitoring + Deterioration)

**Files:**
- Create: `kb-22-hpi-engine/internal/services/monitoring_node_loader.go`
- Create: `kb-22-hpi-engine/internal/services/monitoring_node_loader_test.go`
- Create: `kb-22-hpi-engine/internal/services/deterioration_node_loader.go`
- Create: `kb-22-hpi-engine/internal/services/deterioration_node_loader_test.go`

Separate from existing `NodeLoader` — these parse `monitoring/*.yaml` and `deterioration/*.yaml`.

- [ ] **Step 1: Write failing tests for MonitoringNodeLoader**

Test cases:
- `TestMonitoringNodeLoader_ValidYAML`: Parse a minimal PM node YAML → verify all fields populated
- `TestMonitoringNodeLoader_MissingNodeID`: YAML without node_id → error
- `TestMonitoringNodeLoader_InvalidCondition`: Classification condition with function call → error (uses ExpressionEvaluator validation)
- `TestMonitoringNodeLoader_EmptyClassifications`: No classifications → error
- `TestMonitoringNodeLoader_TypeMustBeMonitoring`: `type: BAYESIAN` → error
- `TestMonitoringNodeLoader_HotReload`: Load, modify file, Reload → see updated definition

Use temporary directories with test YAML files.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestMonitoringNodeLoader -v`
Expected: FAIL

- [ ] **Step 3: Implement MonitoringNodeLoader**

```go
type MonitoringNodeLoader struct {
    dir   string
    log   *zap.Logger
    mu    sync.RWMutex
    nodes map[string]*models.MonitoringNodeDefinition
}

func NewMonitoringNodeLoader(dir string, log *zap.Logger) *MonitoringNodeLoader
func (l *MonitoringNodeLoader) Load() error
// Reads all *.yaml from dir, parses, validates:
// - node_id required, type must be "MONITORING"
// - at least one classification
// - all condition expressions parse without error (dry-run ExpressionEvaluator)
// - cascade_to references are strings (validated at cascade build time)
func (l *MonitoringNodeLoader) Reload() error
func (l *MonitoringNodeLoader) Get(nodeID string) *models.MonitoringNodeDefinition
func (l *MonitoringNodeLoader) All() map[string]*models.MonitoringNodeDefinition
```

- [ ] **Step 4: Run monitoring loader tests**

Run: `go test ./internal/services/ -run TestMonitoringNodeLoader -v`
Expected: ALL PASS

- [ ] **Step 5: Write failing tests for DeteriorationNodeLoader**

Test cases:
- `TestDeteriorationNodeLoader_ValidYAML`: Parse MD node YAML → verify fields
- `TestDeteriorationNodeLoader_TypeMustBeDeterioration`: Wrong type → error
- `TestDeteriorationNodeLoader_DAGValidation`: MD-06 has `contributing_signals: [MD-01, MD-02]` and MD-01 has `cascade_to: [MD-06]` → valid. But if MD-01 cascades to MD-02 AND MD-02 cascades to MD-01 → error (cycle detected)
- `TestDeteriorationNodeLoader_MissingTrajectory`: MD node without trajectory config → error

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestDeteriorationNodeLoader -v`
Expected: FAIL

- [ ] **Step 7: Implement DeteriorationNodeLoader**

```go
type DeteriorationNodeLoader struct {
    dir   string
    log   *zap.Logger
    mu    sync.RWMutex
    nodes map[string]*models.DeteriorationNodeDefinition
}

func (l *DeteriorationNodeLoader) Load() error
// Same pattern as MonitoringNodeLoader + DAG acyclicity check:
// Build adjacency graph from contributing_signals, detect cycles with DFS
func (l *DeteriorationNodeLoader) validateDAG() error
```

- [ ] **Step 8: Run deterioration loader tests**

Run: `go test ./internal/services/ -run TestDeteriorationNodeLoader -v`
Expected: ALL PASS

- [ ] **Step 9: Commit**

```bash
git add kb-22-hpi-engine/internal/services/monitoring_node_loader.go \
       kb-22-hpi-engine/internal/services/monitoring_node_loader_test.go \
       kb-22-hpi-engine/internal/services/deterioration_node_loader.go \
       kb-22-hpi-engine/internal/services/deterioration_node_loader_test.go
git commit -m "feat(kb-22): add MonitoringNodeLoader and DeteriorationNodeLoader with DAG validation"
```

---

### Task 9: Monitoring Engine

**Files:**
- Create: `kb-22-hpi-engine/internal/services/monitoring_engine.go`
- Test: `kb-22-hpi-engine/internal/services/monitoring_engine_test.go`

The core PM node evaluation logic.

- [ ] **Step 1: Write failing tests**

Test cases (use mock DataResolver):
- `TestMonitoringEngine_ClassifyNormalDipper`: dipping_ratio=-0.15 → NORMAL_DIPPER, severity=NONE, gate=SAFE
- `TestMonitoringEngine_ClassifyReverseDipper`: dipping_ratio=0.05 → REVERSE_DIPPER, severity=CRITICAL, gate=PAUSE
- `TestMonitoringEngine_InsufficientData_Skip`: DataResolver returns INSUFFICIENT → no event emitted, returns nil
- `TestMonitoringEngine_InsufficientData_FlagForReview`: DataResolver returns INSUFFICIENT with FLAG_FOR_REVIEW policy → event emitted with data_sufficiency=INSUFFICIENT
- `TestMonitoringEngine_ComputedFields`: sbp=130, daytime=140 → dipping_ratio=-0.071
- `TestMonitoringEngine_SafetyTrigger`: sbp_nocturnal>160 → safety flag attached to event
- `TestMonitoringEngine_FirstMatchWins`: Multiple classifications, first matching one is used
- `TestMonitoringEngine_BuildsClinicalSignalEvent`: Verify event has correct SignalType, NodeID, Classification fields

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestMonitoringEngine -v`
Expected: FAIL

- [ ] **Step 3: Implement MonitoringNodeEngine**

```go
type MonitoringNodeEngine struct {
    loader     *MonitoringNodeLoader
    resolver   DataResolver
    evaluator  *ExpressionEvaluator
    trajectory *TrajectoryComputer  // reviewer Issue 5: PM-06 needs linear regression for fbg_slope
    db         *gorm.DB
    log        *zap.Logger
    metrics    *metrics.Collector
}

func (e *MonitoringNodeEngine) Evaluate(ctx context.Context, nodeID, patientID, stratumLabel string) (*models.ClinicalSignalEvent, error)
// Flow from spec Section 8.5:
// 1. loader.Get(nodeID) — return error if not found
// 2. resolver.Resolve(patientID, node.RequiredInputs, node.AggregatedInputs)
// 3. Check DataSufficiency — if INSUFFICIENT, apply node.InsufficientData policy
// 3b. If resolved.TimeSeries has entries (e.g., PM-06 fbg_values), run trajectory.Compute()
//     and add rate_of_change/slope to fields map (reviewer Issue 5: PM-06 regression)
// 4. Evaluate computed_fields via evaluator.EvaluateNumeric(), add to fields map
// 5. Iterate classifications top-to-bottom, evaluator.EvaluateBool() on each condition
//    First match wins → set classification result
// 6. Evaluate safety_triggers via evaluator.EvaluateBool()
// 7. Build ClinicalSignalEvent with SignalType=MONITORING_CLASSIFICATION
// 8. Persist to clinical_signals table + upsert clinical_signals_latest
// 9. Return event (caller handles publishing + cascade)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestMonitoringEngine -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/internal/services/monitoring_engine.go \
       kb-22-hpi-engine/internal/services/monitoring_engine_test.go
git commit -m "feat(kb-22): add MonitoringNodeEngine with classification and safety evaluation"
```

---

### Task 10: Deterioration Engine

**Files:**
- Create: `kb-22-hpi-engine/internal/services/deterioration_engine.go`
- Test: `kb-22-hpi-engine/internal/services/deterioration_engine_test.go`

The core MD node evaluation logic.

- [ ] **Step 1: Write failing tests**

Test cases (use mock DataResolver + TrajectoryComputer):
- `TestDeteriorationEngine_CriticalDecline`: IS slope=-0.10, IS=0.25 → IS_CRITICAL_DECLINE, CRITICAL, PAUSE
- `TestDeteriorationEngine_StableTrajectory`: IS slope=0.01 → IS_STABLE, NONE, SAFE, no card template
- `TestDeteriorationEngine_Projection`: IS=0.35, slope=-0.05 → projected threshold crossing at 0.20 → ~3 months
- `TestDeteriorationEngine_InsufficientHistory`: Only 3 data points, min=5 → apply USE_SNAPSHOT fallback
- `TestDeteriorationEngine_MD04CompositeScore`: Multiple PM signal inputs → weighted composite → threshold check
- `TestDeteriorationEngine_MD06HaltGate`: cv_risk_score=2.8 → CV_RISK_CRITICAL, gate=HALT
- `TestDeteriorationEngine_ContributingSignals`: Event includes list of PM nodes that contributed

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestDeteriorationEngine -v`
Expected: FAIL

- [ ] **Step 3: Implement DeteriorationNodeEngine**

```go
type DeteriorationNodeEngine struct {
    loader     *DeteriorationNodeLoader
    resolver   DataResolver
    trajectory *TrajectoryComputer
    kb26Client *KB26Client
    evaluator  *ExpressionEvaluator
    db         *gorm.DB
    log        *zap.Logger
    metrics    *metrics.Collector
}

func (e *DeteriorationNodeEngine) Evaluate(ctx context.Context, nodeID, patientID, stratumLabel string, cascadeCtx *CascadeContext) (*models.ClinicalSignalEvent, error)
// Flow from spec Section 8.6:
// 1. loader.Get(nodeID)
// 2. resolver.Resolve(patientID, node.RequiredInputs) — includes KB-26 twin state
// 3. Check DataSufficiency — if INSUFFICIENT, apply policy (FLAG_FOR_REVIEW or USE_SNAPSHOT)
// 4. If node has trajectory config:
//    a. kb26Client.GetVariableHistory(patientID, node.StateVariable, node.Trajectory.WindowDays)
//    b. trajectory.Compute(series, node.Trajectory)
//    c. Add rate_of_change to fields map
// 5. Evaluate thresholds top-to-bottom (first match wins)
// 6. Compute projections if configured
// 7. Build ClinicalSignalEvent with SignalType=DETERIORATION_SIGNAL
// 8. Persist to clinical_signals + upsert clinical_signals_latest
// 9. Return event

// CascadeContext carries PM classification results for composite MD nodes (MD-04, MD-05, MD-06)
type CascadeContext struct {
    PMSignals map[string]float64  // node_id → severity score (0=NONE, 1=MILD, 2=MODERATE, 3=CRITICAL)
    MDSignals map[string]float64  // node_id → severity score (for MD→MD-06 pass)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestDeteriorationEngine -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/internal/services/deterioration_engine.go \
       kb-22-hpi-engine/internal/services/deterioration_engine_test.go
git commit -m "feat(kb-22): add DeteriorationNodeEngine with trajectory computation and projections"
```

---

### Task 11: Signal Cascade

**Files:**
- Create: `kb-22-hpi-engine/internal/services/signal_cascade.go`
- Test: `kb-22-hpi-engine/internal/services/signal_cascade_test.go`

Two-pass PM→MD + MD→MD-06 coordinator.

- [ ] **Step 1: Write failing tests**

Test cases:
- `TestSignalCascade_PMToMD`: PM-04 cascades to [MD-01, MD-05] → both evaluated
- `TestSignalCascade_MDToMD06`: PM-01 → MD-02 (fires) → MD-06 evaluates in pass 2
- `TestSignalCascade_NonFatalFailure`: MD-01 evaluation fails → MD-05 still evaluated, error logged
- `TestSignalCascade_NoCascade`: PM node with empty cascade_to → no MD nodes evaluated
- `TestSignalCascade_MD06NotTriggeredIfNoContributor`: PM-08 → MD-04 only, MD-04 doesn't feed MD-06 (wait — it does in some configs, so test that MD-06 is only triggered when a contributing MD node fires)
- `TestSignalCascade_BuildsCascadeContext`: PM severity scores passed to MD engine

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestSignalCascade -v`
Expected: FAIL

- [ ] **Step 3: Implement SignalCascade**

```go
type SignalCascade struct {
    pmToMD      map[string][]string // built from PM node cascade_to
    mdToMD06    map[string]bool     // built from MD-06 contributing_signals
    deterEngine *DeteriorationNodeEngine
    log         *zap.Logger
}

func NewSignalCascade(monLoader *MonitoringNodeLoader, deterLoader *DeteriorationNodeLoader, deterEngine *DeteriorationNodeEngine, log *zap.Logger) *SignalCascade
// Builds pmToMD from all PM nodes' cascade_to fields
// Builds mdToMD06 from MD-06's contributing_signals

func (sc *SignalCascade) Trigger(ctx context.Context, sourceNodeID, patientID, stratumLabel string, classificationSeverity float64) []*models.ClinicalSignalEvent
// Two-pass evaluation from spec Section 8.7:
// Pass 1: For each MD in pmToMD[sourceNodeID], evaluate with CascadeContext
// Pass 2: If any Pass 1 result is in mdToMD06, evaluate MD-06
// Cascade failures are non-fatal (logged, not returned as errors)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestSignalCascade -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/internal/services/signal_cascade.go \
       kb-22-hpi-engine/internal/services/signal_cascade_test.go
git commit -m "feat(kb-22): add SignalCascade with two-pass PM→MD→MD-06 evaluation"
```

---

### Task 12: Signal Publisher

**Files:**
- Create: `kb-22-hpi-engine/internal/services/signal_publisher.go`
- Test: `kb-22-hpi-engine/internal/services/signal_publisher_test.go`

Publishes ClinicalSignalEvent to KB-23 HTTP endpoint + Kafka topic.

- [ ] **Step 1: Write failing tests**

Test cases:
- `TestSignalPublisher_PublishToKB23`: Mock KB-23 returns 201 → event marked published_to_kb23=true
- `TestSignalPublisher_PublishToKafka`: Mock Kafka → event published to clinical.signal.events topic
- `TestSignalPublisher_KB23Returns204`: No card needed → no error, just log
- `TestSignalPublisher_KB23Retry`: First call fails, second succeeds → published after retry
- `TestSignalPublisher_KB23FailAfterRetries`: All retries fail → published_to_kb23 stays false, logged, no panic

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestSignalPublisher -v`
Expected: FAIL

- [ ] **Step 3: Implement SignalPublisher**

```go
type SignalPublisher struct {
    kb23URL    string
    kb23Client *http.Client
    kafka      KafkaPublisher
    kafkaTopic string
    retryCount int
    retryDelay time.Duration
    db         *gorm.DB
    log        *zap.Logger
}

func (p *SignalPublisher) Publish(ctx context.Context, event *models.ClinicalSignalEvent) error
// 1. POST to KB-23 /api/v1/clinical-signals with retry
// 2. On success (201/204), update clinical_signals.published_to_kb23 = true
// 3. Publish to Kafka topic (fire-and-forget with logging on error)
// 4. Return nil even if KB-23 fails (async retry can pick up unpublished later)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run TestSignalPublisher -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/internal/services/signal_publisher.go \
       kb-22-hpi-engine/internal/services/signal_publisher_test.go
git commit -m "feat(kb-22): add SignalPublisher with KB-23 + Kafka dual publish and retry"
```

---

### Task 13: API Handlers + SignalHandlerGroup

**Files:**
- Create: `kb-22-hpi-engine/internal/api/signal_handler_group.go`
- Create: `kb-22-hpi-engine/internal/api/event_ingestion_handlers.go`
- Create: `kb-22-hpi-engine/internal/api/signal_query_handlers.go`
- Modify: `kb-22-hpi-engine/internal/api/server.go` (3 additive lines)

- [ ] **Step 1: Implement SignalHandlerGroup**

Create `signal_handler_group.go`:
```go
type SignalHandlerGroup struct {
    monitoringLoader    *services.MonitoringNodeLoader
    deteriorationLoader *services.DeteriorationNodeLoader
    monitoringEngine    *services.MonitoringNodeEngine
    deteriorationEngine *services.DeteriorationNodeEngine
    cascade             *services.SignalCascade
    resolver            services.DataResolver
    publisher           *services.SignalPublisher
    cache               *cache.CacheClient
    db                  *database.Database
    cfg                 *config.Config
    log                 *zap.Logger
}

func NewSignalHandlerGroup(cfg *config.Config, db *database.Database, cacheClient *cache.CacheClient, kafkaPublisher services.KafkaPublisher, log *zap.Logger, metricsCollector *metrics.Collector) *SignalHandlerGroup
// 1. Create KB26Client
// 2. Create DataResolver
// 3. Create ExpressionEvaluator
// 4. Create TrajectoryComputer
// 5. Create MonitoringNodeLoader + Load()
// 6. Create DeteriorationNodeLoader + Load()
// 7. Create MonitoringNodeEngine
// 8. Create DeteriorationNodeEngine
// 9. Create SignalCascade
// 10. Create SignalPublisher

func (g *SignalHandlerGroup) RegisterRoutes(router *gin.RouterGroup)
// Events:
//   POST /events/observation
//   POST /events/twin-state-update
//   POST /events/checkin-response
// Queries:
//   GET /patients/:id/signals
//   GET /patients/:id/signals/:nodeId
//   GET /patients/:id/deterioration-summary
// Node management:
//   GET /nodes/monitoring
//   GET /nodes/monitoring/:nodeId
//   GET /nodes/deterioration
//   GET /nodes/deterioration/:nodeId
```

- [ ] **Step 2: Implement event ingestion handlers**

Create `event_ingestion_handlers.go`:
```go
func (g *SignalHandlerGroup) handleObservation(c *gin.Context)
// Parse ObservationEvent → 202 Accepted → background goroutine:
//   1. Redis debounce check: "eval:{patientID}:{nodeID}" with TTL
//      **Safety bypass (reviewer Issue 10):** On debounce cache hit, still check if observation
//      value exceeds any matching node's safety_trigger condition. If safety trigger fires,
//      proceed with full evaluation despite debounce (safety always evaluates).
//   2. Find matching PM nodes (by observation code → required_inputs mapping)
//   3. For each matching PM node: engine.Evaluate() → cascade.Trigger() → publisher.Publish()
//   4. Find matching MD nodes (by trigger_on containing "OBSERVATION:{code}")
//   5. For each matching MD node: engine.Evaluate() → publisher.Publish()

func (g *SignalHandlerGroup) handleTwinStateUpdate(c *gin.Context)
// Parse TwinStateUpdateEvent → 202 Accepted → background goroutine:
//   Find MD nodes with trigger_on containing "TWIN_STATE_UPDATE"
//   Evaluate each → publish

func (g *SignalHandlerGroup) handleCheckinResponse(c *gin.Context)
// Parse CheckinResponseEvent → 202 Accepted → background goroutine:
//   Find PM nodes with TIER1_CHECKIN inputs matching the prompt_id
//   Evaluate → cascade → publish
```

- [ ] **Step 3: Implement signal query handlers**

Create `signal_query_handlers.go`:
```go
func (g *SignalHandlerGroup) handleGetPatientSignals(c *gin.Context)
// SELECT * FROM clinical_signals_latest WHERE patient_id = ? JOIN clinical_signals

func (g *SignalHandlerGroup) handleGetSignalHistory(c *gin.Context)
// SELECT * FROM clinical_signals WHERE patient_id = ? AND node_id = ? ORDER BY evaluated_at DESC LIMIT ?

func (g *SignalHandlerGroup) handleGetDeteriorationSummary(c *gin.Context)
// SELECT * FROM clinical_signals_latest WHERE patient_id = ? AND signal_type = 'DETERIORATION_SIGNAL' JOIN clinical_signals

func (g *SignalHandlerGroup) handleListMonitoringNodes(c *gin.Context)
func (g *SignalHandlerGroup) handleGetMonitoringNode(c *gin.Context)
func (g *SignalHandlerGroup) handleListDeteriorationNodes(c *gin.Context)
func (g *SignalHandlerGroup) handleGetDeteriorationNode(c *gin.Context)
```

- [ ] **Step 4: Write handler unit tests (reviewer Issue 8)**

Create `kb-22-hpi-engine/internal/api/signal_handlers_test.go`:
- `TestHandleObservation_Returns202`: Valid JSON → 202 Accepted
- `TestHandleObservation_InvalidJSON_Returns400`: Malformed body → 400
- `TestHandleGetPatientSignals_Returns200`: Mock DB with 2 signals → 200 with array
- `TestHandleGetSignalHistory_Returns200`: Mock DB → 200 with ordered results
- `TestHandleGetDeteriorationSummary_Returns200`: Mock DB with MD signals → 200

Run: `go test ./internal/api/ -run TestHandle -v`
Expected: FAIL (handlers not yet wired)

- [ ] **Step 5: Wire into server.go**

Add to `Server` struct:
```go
SignalGroup *SignalHandlerGroup
```

Add to `InitServices()` (after existing service init, before SessionService creation):
```go
s.SignalGroup = NewSignalHandlerGroup(s.Config, s.DB, s.Cache, s.KafkaPublisher, s.Log, s.Metrics)
```

Add to `RegisterRoutes()` (after existing v1 block):
```go
s.SignalGroup.RegisterRoutes(v1)
```

Extend `reloadNodesHandler` to also reload PM/MD loaders:
```go
if s.SignalGroup != nil {
    if err := s.SignalGroup.monitoringLoader.Reload(); err != nil {
        s.Log.Warn("failed to reload monitoring nodes", zap.Error(err))
    }
    if err := s.SignalGroup.deteriorationLoader.Reload(); err != nil {
        s.Log.Warn("failed to reload deterioration nodes", zap.Error(err))
    }
}
```

- [ ] **Step 6: Verify build compiles**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 7: Run handler tests + verify existing tests still pass**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go test ./... -count=1`
Expected: ALL tests PASS (new handler tests + backward compatibility)

- [ ] **Step 8: Commit**

```bash
git add kb-22-hpi-engine/internal/api/signal_handler_group.go \
       kb-22-hpi-engine/internal/api/event_ingestion_handlers.go \
       kb-22-hpi-engine/internal/api/signal_query_handlers.go \
       kb-22-hpi-engine/internal/api/signal_handlers_test.go \
       kb-22-hpi-engine/internal/api/server.go
git commit -m "feat(kb-22): add SignalHandlerGroup, event ingestion, signal queries, wire into server"
```

---

### Task 14: PM Node YAML Definitions (9 files)

**Files:**
- Create: `kb-22-hpi-engine/monitoring/pm_01_home_bp.yaml`
- Create: `kb-22-hpi-engine/monitoring/pm_02_daily_symptom_checkin.yaml`
- Create: `kb-22-hpi-engine/monitoring/pm_03_nocturnal_bp_dipping.yaml`
- Create: `kb-22-hpi-engine/monitoring/pm_04_postprandial_glucose.yaml`
- Create: `kb-22-hpi-engine/monitoring/pm_05_glycemic_variability.yaml`
- Create: `kb-22-hpi-engine/monitoring/pm_06_fbg_trend.yaml`
- Create: `kb-22-hpi-engine/monitoring/pm_07_exercise_capacity.yaml`
- Create: `kb-22-hpi-engine/monitoring/pm_08_hrv_pattern.yaml`
- Create: `kb-22-hpi-engine/monitoring/pm_09_sleep_quality.yaml`

All clinical values come directly from spec Section 11.1.

- [ ] **Step 1: Create monitoring directory**

Run: `mkdir -p backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/monitoring`

- [ ] **Step 2: Write PM-01 through PM-03 YAML files**

Write `pm_01_home_bp.yaml`, `pm_02_daily_symptom_checkin.yaml`, `pm_03_nocturnal_bp_dipping.yaml` following the schema from spec Section 7.1 with clinical values from Section 11.1.

Each file must include: `node_id`, `version: "1.0.0"`, `type: MONITORING`, `title_en`, `title_hi`, `required_inputs`, `computed_fields`, `classifications` (ordered most severe first), `insufficient_data`, `safety_triggers`, `cascade_to`.

PM-03 example is fully specified in spec lines 433-497. Use it as the template for all PM nodes.

- [ ] **Step 3: Write PM-04 through PM-06 YAML files**

`pm_04_postprandial_glucose.yaml`, `pm_05_glycemic_variability.yaml`, `pm_06_fbg_trend.yaml`

Clinical thresholds from spec:
- PM-04: SEVERE_EXCURSION (>80), HIGH (50-80), MODERATE (30-50), NORMAL (<30)
- PM-05: HIGHLY_VARIABLE (cv>36), MODERATELY_VARIABLE (20-36), STABLE (<20)
- PM-06: RAPIDLY_RISING (slope>5/mo), GRADUALLY_RISING (2-5), STABLE (|slope|<2), IMPROVING (<-2)

- [ ] **Step 4: Write PM-07 through PM-09 YAML files**

`pm_07_exercise_capacity.yaml`, `pm_08_hrv_pattern.yaml`, `pm_09_sleep_quality.yaml`

PM-08 has `insufficient_data: action: SKIP` (many patients lack wearable HRV).
PM-09 has `insufficient_data: action: SKIP` (weekly check-in may not be available).
PM-09 computed field uses the whitelisted `normalize()` builtin (reviewer Issue 4): `sleep_score = 0.40 * normalize(sleep_difficulty, 1, 5) + 0.35 * (1 - normalize(sleep_duration_hrs, 4, 9)) + 0.25 * sleep_disruptions`

- [ ] **Step 5: Verify all PM YAMLs load**

Run: Write a quick test or use the MonitoringNodeLoader:
```bash
cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine
go test ./internal/services/ -run TestMonitoringNodeLoader_LoadAll -v
```
(Add a test that loads from the actual `monitoring/` directory)

- [ ] **Step 6: Commit**

```bash
git add kb-22-hpi-engine/monitoring/
git commit -m "feat(kb-22): add 9 PM node YAML definitions (Layer 2 physiological monitoring)"
```

---

### Task 15: MD Node YAML Definitions (6 files)

**Files:**
- Create: `kb-22-hpi-engine/deterioration/md_01_insulin_resistance.yaml`
- Create: `kb-22-hpi-engine/deterioration/md_02_vascular_compliance.yaml`
- Create: `kb-22-hpi-engine/deterioration/md_03_renal_function.yaml`
- Create: `kb-22-hpi-engine/deterioration/md_04_autonomic_dysfunction.yaml`
- Create: `kb-22-hpi-engine/deterioration/md_05_glycemic_control.yaml`
- Create: `kb-22-hpi-engine/deterioration/md_06_cardiovascular_risk.yaml`

All clinical values come from spec Section 11.2.

- [ ] **Step 1: Create deterioration directory**

Run: `mkdir -p backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/deterioration`

- [ ] **Step 2: Write MD-01 through MD-03 YAML files**

Each file follows the schema from spec Section 7.2 with values from Section 11.2.

MD-01: `state_variable: IS`, thresholds from spec line 1003, `trigger_on` includes `PROTOCOL:M3-PRP:ADHERENCE`
MD-02: `state_variable: VR`, fallback derivation note
MD-03: `state_variable: RR`, uses pre-computed RenalSlope (no additional regression)

- [ ] **Step 3: Write MD-04 through MD-06 YAML files**

MD-04: Composite score with adaptive weights (spec lines 1027-1030). No state_variable — computed from 6 contributing factors. Uses `computed_fields` with conditional weight redistribution (reviewer Issue 3). The YAML defines 4 formula variants; DeteriorationNodeEngine checks data_sufficiency per contributing PM input and selects the appropriate variant:
  - Full (PM-08 + PM-09 available): `autonomic_score = 0.30*pm03_severity + 0.15*pm08_severity + 0.20*pm07_severity + 0.15*pm09_severity + 0.10*orthostatic_delta + 0.10*m3_vfrp_activity`
  - PM-08 unavailable: `autonomic_score = 0.30*pm03_severity + 0.30*pm09_severity + 0.20*pm07_severity + 0.10*orthostatic_delta + 0.10*m3_vfrp_activity`
  - PM-09 unavailable: `autonomic_score = 0.30*pm03_severity + 0.30*pm08_severity + 0.20*pm07_severity + 0.10*orthostatic_delta + 0.10*m3_vfrp_activity`
  - Both unavailable: `autonomic_score = 0.40*pm03_severity + 0.30*pm07_severity + 0.15*orthostatic_delta + 0.15*m3_vfrp_activity`
  - Express as `computed_field_variants` list in YAML — each variant has a `condition` (e.g., `pm08_available AND pm09_available`) and a `formula`. DeteriorationNodeEngine evaluates conditions top-to-bottom, first match wins.
MD-05: `state_variable: HGO`, composite with HbA1c normalization. Uses `computed_fields` with `normalize()` (whitelisted builtin) and PM cascade severity scores from `CascadeContext.PMSignals` (0=NONE, 1=MILD, 2=MODERATE, 3=CRITICAL): `glycemic_composite = 0.35*normalize(hba1c,6,12) + 0.30*normalize(fbg,90,250) + 0.20*pm04 + 0.15*pm05` — where `pm04` and `pm05` are severity scores injected into the fields map by DeteriorationNodeEngine from CascadeContext, matching the spec exactly.
MD-06: `state_variable: CV_RISK` (composite), HALT gate for CRITICAL. `trigger_on` includes `PROTOCOL:M3-VFRP:ACTIVITY`. Contributing signals: MD-01, MD-02, MD-03.

- [ ] **Step 4: Verify all MD YAMLs load**

Run: `go test ./internal/services/ -run TestDeteriorationNodeLoader_LoadAll -v`

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/deterioration/
git commit -m "feat(kb-22): add 6 MD node YAML definitions (Layer 3 metabolic deterioration)"
```

---

### Task 16: KB-23 Signal Endpoint + SignalCardBuilder

**Files:**
- Create: `kb-23-decision-cards/internal/models/clinical_signal_event.go`
- Create: `kb-23-decision-cards/internal/services/signal_card_builder.go`
- Create: `kb-23-decision-cards/internal/services/signal_card_builder_test.go`
- Create: `kb-23-decision-cards/internal/api/signal_handlers.go`
- Modify: `kb-23-decision-cards/internal/models/enums.go`
- Modify: `kb-23-decision-cards/internal/api/routes.go`
- Modify: `kb-23-decision-cards/internal/api/server.go`
- Modify: `kb-23-decision-cards/internal/services/hysteresis_engine.go`

This is a cross-service change. KB-22's SignalPublisher calls this endpoint.

- [ ] **Step 1: Add ClinicalSignalEvent types to KB-23 models**

Create `kb-23-decision-cards/internal/models/clinical_signal_event.go` — copy the same types from KB-22's `clinical_signal_event.go`. These are the deserialization targets for POST /api/v1/clinical-signals.

- [ ] **Step 2: Add CLINICAL_SIGNAL to CardSource enum**

In `kb-23-decision-cards/internal/models/enums.go`, add:
```go
CardSourceClinicalSignal CardSource = "CLINICAL_SIGNAL"
```

- [ ] **Step 3: Write failing tests for SignalCardBuilder**

Test cases:
- `TestSignalCardBuilder_TemplateResolution`: PM-03 REVERSE_DIPPER → resolves to "dc-pm03-reverse-dipper-v1"
- `TestSignalCardBuilder_PM09Template`: PM-09 SEVERELY_DISRUPTED → resolves to "dc-pm09-severely-disrupted-v1" (**reviewer Issue 2**: spec Section 15.3 lists templates through PM-08 only; PM-09 templates must also be created)
- `TestSignalCardBuilder_NoTemplate`: PM-01 NORMAL → returns nil (204 No Content)
- `TestSignalCardBuilder_ConfidenceTier`: SUFFICIENT + CRITICAL → FIRM; PARTIAL + MODERATE → POSSIBLE
- `TestSignalCardBuilder_HALTGate`: MD-06 CV_RISK_CRITICAL with mcu_gate_suggestion=HALT → card has MCUGate=HALT, pending_reaffirmation=true
- `TestSignalCardBuilder_HysteresisApplied`: Recent SAFE gate, now PAUSE suggested → PAUSE allowed (upgrade immediate)
- `TestSignalCardBuilder_CardSource`: Built card has CardSource=CLINICAL_SIGNAL

- [ ] **Step 4: Run tests to verify they fail**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/services/ -run TestSignalCardBuilder -v`
Expected: FAIL

- [ ] **Step 5: Implement SignalCardBuilder**

Create `kb-23-decision-cards/internal/services/signal_card_builder.go`:
```go
type SignalCardBuilder struct {
    templateLoader *TemplateLoader
    gateManager    *MCUGateManager
    hysteresis     *HysteresisEngine
    kb20Client     *KB20Client
    kb19Publisher  *KB19Publisher
    db             *database.Database
    log            *zap.Logger
}

func (b *SignalCardBuilder) Build(ctx context.Context, event *models.ClinicalSignalEvent) (*models.DecisionCard, error)
// Spec Section 4.3:
// 1. resolveTemplate(event) — match by node_id + classification/deterioration signal
// 2. deriveConfidenceTier(event) — SUFFICIENT+CRITICAL=FIRM, PARTIAL=POSSIBLE, etc.
// 3. evaluateGate(event, tier) — use event.MCUGateSuggestion, apply hysteresis
// 4. KB-20 enrichment (best-effort)
// 5. Build DecisionCard with CardSource=CLINICAL_SIGNAL
// 6. Persist + publish gate change
```

- [ ] **Step 6: Extend HysteresisEngine with HALT rules**

In `kb-23-decision-cards/internal/services/hysteresis_engine.go`, add HALT handling per spec Section 4.4:

```go
// In the gate transition evaluation method:
// 1. HALT can only be set by MD-06 or physician action
// 2. HALT cannot be auto-downgraded by timer
// 3. HALT → PAUSE requires explicit physician resume
// 4. HALT → MODIFY/SAFE not allowed (must step through PAUSE)
// 5. HALT triggers pending_reaffirmation = true
```

Add a check: if `suggestedGate == HALT && source != "MD-06" && source != "PHYSICIAN"`, reject with warning log.
Add a check: if `currentGate == HALT && suggestedGate != HALT`, return HALT (block downgrade unless physician resume).

- [ ] **Step 7: Implement signal handler**

Create `kb-23-decision-cards/internal/api/signal_handlers.go`:
```go
func (s *Server) handleClinicalSignal(c *gin.Context) {
    var event models.ClinicalSignalEvent
    if err := c.ShouldBindJSON(&event); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    card, err := s.signalCardBuilder.Build(c.Request.Context(), &event)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    if card == nil {
        c.Status(204)
        return
    }
    c.JSON(201, card)
}
```

- [ ] **Step 8: Wire into KB-23 routes and server**

In `routes.go`, add: `v1.POST("/clinical-signals", s.handleClinicalSignal)`

In `server.go`, add `signalCardBuilder *services.SignalCardBuilder` field and init in server setup.

- [ ] **Step 9: Run tests to verify they pass**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./... -v`
Expected: ALL PASS (new + existing)

- [ ] **Step 10: Commit**

```bash
git add kb-23-decision-cards/internal/models/clinical_signal_event.go \
       kb-23-decision-cards/internal/models/enums.go \
       kb-23-decision-cards/internal/services/signal_card_builder.go \
       kb-23-decision-cards/internal/services/signal_card_builder_test.go \
       kb-23-decision-cards/internal/services/hysteresis_engine.go \
       kb-23-decision-cards/internal/api/signal_handlers.go \
       kb-23-decision-cards/internal/api/routes.go \
       kb-23-decision-cards/internal/api/server.go
git commit -m "feat(kb-23): add clinical signal endpoint with SignalCardBuilder and HALT gate rules"
```

---

### Task 17: Docker + Integration Wiring

**Files:**
- Modify: `kb-22-hpi-engine/docker-compose.yml`
- Modify: `kb-22-hpi-engine/Dockerfile`

- [ ] **Step 1: Update Dockerfile to copy new directories**

Add after existing `COPY nodes/ ./nodes/`:
```dockerfile
COPY monitoring/ ./monitoring/
COPY deterioration/ ./deterioration/
```

- [ ] **Step 2: Update docker-compose.yml with KB-26 env vars**

Add to kb22-service environment:
```yaml
KB26_URL: "http://kb26-service:8137"
KB26_TIMEOUT_MS: "5000"
KB20_OBSERVATION_TIMEOUT_MS: "10000"
MONITORING_NODES_DIR: "/app/monitoring"
DETERIORATION_NODES_DIR: "/app/deterioration"
SIGNAL_DEBOUNCE_TTL_SEC: "300"
SIGNAL_PUBLISHER_RETRY_COUNT: "3"
SIGNAL_PUBLISHER_RETRY_DELAY_SEC: "30"
KAFKA_SIGNAL_TOPIC: "clinical.signal.events"
KB26_STALENESS_DAYS: "21"
```

- [ ] **Step 3: Verify Docker build**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && docker build -t kb22-test .`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add kb-22-hpi-engine/Dockerfile kb-22-hpi-engine/docker-compose.yml
git commit -m "feat(kb-22): update Docker config for PM/MD node directories and KB-26 integration"
```

---

### Task 18: Integration Tests

**Files:**
- Create: `kb-22-hpi-engine/tests/integration/signal_integration_test.go`

- [ ] **Step 1: Write integration tests**

Test scenarios from spec Section 14:
- **Full PM→MD cascade**: Simulate observation event (FBG=200) → PM-04 classifies HIGH_EXCURSION → cascade triggers MD-01 + MD-05 → ClinicalSignalEvents published
- **Twin state update → MD evaluation**: Simulate twin state update → MD-01 through MD-06 evaluate
- **Debounce**: Send same observation twice within 5 minutes → only one evaluation
- **Insufficient data graceful degradation**: PM-08 with no HRV data → SKIP, no event
- **Backward compatibility**: Run all existing KB-22 test suites after changes

- [ ] **Step 2: Run integration tests**

Run: `go test -tags=integration ./tests/integration/ -run TestSignal -v -timeout 60s`
Expected: ALL PASS (may need Docker services running)

- [ ] **Step 3: Run full test suite for backward compatibility**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go test ./... -count=1`
Expected: ALL tests pass including existing wave tests

- [ ] **Step 4: Commit**

```bash
git add kb-22-hpi-engine/tests/integration/signal_integration_test.go
git commit -m "test(kb-22): add integration tests for PM→MD cascade, debounce, and graceful degradation"
```

---

### Task 19: Golden Dataset Calibration (reviewer Issue 1)

**Files:**
- Create: `kb-22-hpi-engine/calibration/pm_golden_dataset.json`
- Create: `kb-22-hpi-engine/calibration/md_golden_dataset.json`
- Create: `kb-22-hpi-engine/tests/calibration/golden_dataset_test.go`

The spec (Section 14) requires golden datasets for validating clinical threshold correctness. These are known input→output pairs that verify PM/MD engines produce the correct classifications for given clinical data.

- [ ] **Step 1: Create PM golden dataset**

Create `calibration/pm_golden_dataset.json` with test vectors:
```json
[
  {
    "test_id": "PM01_SEVERELY_ABOVE",
    "node_id": "PM-01",
    "inputs": {"sbp_home_mean": 165, "dbp_home_mean": 102, "bp_target_sbp": 130, "bp_target_dbp": 80},
    "expected_classification": "SEVERELY_ABOVE",
    "expected_severity": "CRITICAL",
    "expected_gate": "PAUSE"
  },
  {
    "test_id": "PM03_REVERSE_DIPPER",
    "node_id": "PM-03",
    "inputs": {"sbp_nocturnal_mean": 145, "sbp_daytime_mean": 140},
    "expected_classification": "REVERSE_DIPPER",
    "expected_severity": "CRITICAL",
    "expected_gate": "PAUSE"
  }
]
```
Include at least 2 test vectors per PM node (one normal, one abnormal classification). Total: ~20 vectors.

- [ ] **Step 2: Create MD golden dataset**

Create `calibration/md_golden_dataset.json` with test vectors covering known KB-26 state trajectories:
```json
[
  {
    "test_id": "MD01_CRITICAL_DECLINE",
    "node_id": "MD-01",
    "twin_state": {"IS": 0.25, "IS_confidence": 0.85},
    "history": [{"days_ago": 90, "value": 0.45}, {"days_ago": 60, "value": 0.38}, {"days_ago": 30, "value": 0.30}, {"days_ago": 0, "value": 0.25}],
    "expected_signal": "IS_CRITICAL_DECLINE",
    "expected_severity": "CRITICAL",
    "expected_gate": "PAUSE"
  }
]
```
Include at least 2 vectors per MD node. Total: ~14 vectors.

- [ ] **Step 3: Write golden dataset test**

Create `tests/calibration/golden_dataset_test.go`:
```go
func TestPMGoldenDataset(t *testing.T) {
    // Load calibration/pm_golden_dataset.json
    // For each vector: create mock DataResolver returning vector.inputs
    // Run MonitoringNodeEngine.Evaluate()
    // Assert classification, severity, gate match expected
}

func TestMDGoldenDataset(t *testing.T) {
    // Load calibration/md_golden_dataset.json
    // For each vector: create mock KB-26 client returning twin_state + history
    // Run DeteriorationNodeEngine.Evaluate()
    // Assert signal, severity, gate match expected
}
```

- [ ] **Step 4: Run golden dataset tests**

Run: `go test ./tests/calibration/ -v`
Expected: ALL PASS — clinical thresholds produce correct classifications

- [ ] **Step 5: Commit**

```bash
git add kb-22-hpi-engine/calibration/ kb-22-hpi-engine/tests/calibration/
git commit -m "test(kb-22): add PM/MD golden dataset calibration tests for clinical threshold validation"
```

---

### Task 20: Final Validation + Backward Compatibility Check

> **Scope note (reviewer Issue 11):** KB-26 Kafka consumer for MRI recomputation (subscribing to `clinical.signal.events`) is **out of scope** for this implementation. KB-22 publishes to the topic; KB-26 consumer will be addressed in a separate KB-26-focused plan.

- [ ] **Step 1: Run full KB-22 test suite**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go test ./... -count=1 -v 2>&1 | tail -50`
Expected: ALL PASS. If any existing test fails, the change MUST be fixed before merging.

- [ ] **Step 2: Run full KB-23 test suite**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./... -count=1 -v 2>&1 | tail -50`
Expected: ALL PASS.

- [ ] **Step 3: Build both services**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine && go build ./... && cd ../kb-23-decision-cards && go build ./...`
Expected: Both build successfully.

- [ ] **Step 4: Verify health endpoints respond**

If Docker is available:
```bash
make run-kb-docker
curl http://localhost:8132/health  # KB-22
curl http://localhost:8134/health  # KB-23
```
Expected: Both return `{"status": "healthy"}` with nodes_loaded count including new PM/MD nodes.

- [ ] **Step 5: Final commit with summary**

```bash
git add -A
git status  # verify no stray files
git commit -m "feat(kb-22): complete three-layer node taxonomy implementation

Layer 2: 9 PM physiological monitoring nodes (BP, glucose, HRV, sleep, exercise)
Layer 3: 6 MD metabolic deterioration nodes (insulin resistance, vascular, renal, autonomic, glycemic, CV risk)
Engine: DataResolver, ExpressionEvaluator, TrajectoryComputer, SignalCascade
Integration: KB-23 clinical-signals endpoint with HALT gate rules
All existing tests pass unchanged."
```
