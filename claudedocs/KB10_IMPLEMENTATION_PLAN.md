# KB-10 Clinical Rules Engine - Implementation Plan

## Executive Summary

KB-10 is the **Clinical Rules Engine** for the CardioFit CDSS platform. It provides YAML-driven configurable business rules execution without code changes, enabling healthcare organizations to customize clinical decision support to their specific needs.

**Target Port**: 8100 (as specified in kb10-readme.md)
**Estimated LOC**: ~7,600 lines
**Implementation Time**: 5-7 days
**Reference Implementation**: KB-1 (cleanest pattern), KB-8 (calculator patterns)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         KB-10 RULES ENGINE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        YAML RULE LOADER                               │ │
│  │  • Load rules from /rules directory                                   │ │
│  │  • Hot-reload on SIGHUP                                               │ │
│  │  • Validate rule syntax and conflicts                                 │ │
│  └────────────────────────────────────┬──────────────────────────────────┘ │
│                                       │                                     │
│                                       ▼                                     │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                         RULE STORE                                    │ │
│  │  • In-memory rule storage                                             │ │
│  │  • Indexed by type, category, severity, tags                         │ │
│  │  • Priority-sorted for evaluation                                     │ │
│  └────────────────────────────────────┬──────────────────────────────────┘ │
│                                       │                                     │
│                                       ▼                                     │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                      EVALUATION ENGINE                                │ │
│  │  • Condition evaluator (20+ operators)                               │ │
│  │  • CQL expression support via Vaidshala                              │ │
│  │  • Action executor                                                    │ │
│  │  • Result caching                                                     │ │
│  └────────────────────────────────────┬──────────────────────────────────┘ │
│                                       │                                     │
│                                       ▼                                     │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                      AUDIT DATABASE                                   │ │
│  │  • PostgreSQL storage                                                 │ │
│  │  • Rule execution history                                             │ │
│  │  • Alert management                                                   │ │
│  │  • Rule statistics                                                    │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Foundation (Day 1-2)

### 1.1 Directory Structure Creation

```
kb-10-rules-engine/
├── cmd/
│   └── server/
│       └── main.go                     # Entry point (~80 lines)
├── internal/
│   ├── api/
│   │   └── server.go                   # HTTP API server (~800 lines)
│   ├── config/
│   │   └── config.go                   # Configuration management (~150 lines)
│   ├── database/
│   │   └── postgres.go                 # PostgreSQL operations (~500 lines)
│   ├── engine/
│   │   ├── engine.go                   # Core rule engine (~600 lines)
│   │   ├── evaluator.go                # Condition evaluator (~800 lines)
│   │   ├── executor.go                 # Action executor (~400 lines)
│   │   └── cache.go                    # Evaluation cache (~200 lines)
│   ├── loader/
│   │   └── yaml_loader.go              # YAML rule loader (~500 lines)
│   ├── metrics/
│   │   └── metrics.go                  # Metrics collection (~200 lines)
│   └── models/
│       ├── rule.go                     # Rule domain models (~600 lines)
│       └── store.go                    # Rule store (~400 lines)
├── rules/
│   ├── safety/
│   │   ├── critical-alerts.yaml        # Critical lab/vital alerts
│   │   └── medication-validation.yaml  # Medication safety rules
│   ├── clinical/
│   │   ├── inference-rules.yaml        # Clinical inference rules
│   │   └── escalation-rules.yaml       # Escalation pathways
│   └── governance/
│       └── governance-rules.yaml       # Approval workflows
├── cql/
│   └── tier-6-application/
│       ├── ClinicalRulesEngine-1.0.0.cql
│       ├── AlertRules-1.0.0.cql
│       └── EscalationRules-1.0.0.cql
├── tests/
│   ├── unit/
│   │   ├── engine_test.go
│   │   ├── evaluator_test.go
│   │   └── loader_test.go
│   ├── integration/
│   │   └── api_test.go
│   └── clinical/
│       └── scenarios_test.go
├── migrations/
│   ├── 001_create_rules_table.sql
│   ├── 002_create_alerts_table.sql
│   └── 003_create_audit_table.sql
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── README.md
```

### 1.2 Core Models (`internal/models/rule.go`)

```go
// Rule Types
const (
    RuleTypeAlert          = "ALERT"
    RuleTypeInference      = "INFERENCE"
    RuleTypeValidation     = "VALIDATION"
    RuleTypeEscalation     = "ESCALATION"
    RuleTypeSuppression    = "SUPPRESSION"
    RuleTypeDerivation     = "DERIVATION"
    RuleTypeRecommendation = "RECOMMENDATION"
    RuleTypeConflict       = "CONFLICT"
)

// Condition Operators
const (
    OperatorEQ         = "EQ"
    OperatorNEQ        = "NEQ"
    OperatorGT         = "GT"
    OperatorGTE        = "GTE"
    OperatorLT         = "LT"
    OperatorLTE        = "LTE"
    OperatorCONTAINS   = "CONTAINS"
    OperatorIN         = "IN"
    OperatorBETWEEN    = "BETWEEN"
    OperatorEXISTS     = "EXISTS"
    OperatorIS_NULL    = "IS_NULL"
    OperatorMATCHES    = "MATCHES"
    OperatorAGE_GT     = "AGE_GT"
    OperatorWITHIN_DAYS = "WITHIN_DAYS"
)

// Core Structs
type Rule struct {
    ID             string       `yaml:"id" json:"id"`
    Name           string       `yaml:"name" json:"name"`
    Description    string       `yaml:"description" json:"description"`
    Type           string       `yaml:"type" json:"type"`
    Category       string       `yaml:"category" json:"category"`
    Severity       string       `yaml:"severity" json:"severity"`
    Status         string       `yaml:"status" json:"status"`
    Priority       int          `yaml:"priority" json:"priority"`
    Version        string       `yaml:"version" json:"version"`
    Conditions     []Condition  `yaml:"conditions" json:"conditions"`
    ConditionLogic string       `yaml:"condition_logic" json:"condition_logic"`
    Actions        []Action     `yaml:"actions" json:"actions"`
    Evidence       Evidence     `yaml:"evidence" json:"evidence"`
    Tags           []string     `yaml:"tags" json:"tags"`
    CreatedAt      time.Time    `json:"created_at"`
    UpdatedAt      time.Time    `json:"updated_at"`
}

type Condition struct {
    Field      string      `yaml:"field" json:"field"`
    Operator   string      `yaml:"operator" json:"operator"`
    Value      interface{} `yaml:"value" json:"value"`
    Unit       string      `yaml:"unit,omitempty" json:"unit,omitempty"`
    CQLExpr    string      `yaml:"cql_expression,omitempty" json:"cql_expression,omitempty"`
}

type Action struct {
    Type       string            `yaml:"type" json:"type"`
    Message    string            `yaml:"message,omitempty" json:"message,omitempty"`
    Priority   string            `yaml:"priority,omitempty" json:"priority,omitempty"`
    Parameters map[string]string `yaml:"parameters,omitempty" json:"parameters,omitempty"`
    Recipients []string          `yaml:"recipients,omitempty" json:"recipients,omitempty"`
    Channel    string            `yaml:"channel,omitempty" json:"channel,omitempty"`
}

type Evidence struct {
    Level  string `yaml:"level" json:"level"`
    Source string `yaml:"source" json:"source"`
}
```

### 1.3 Configuration (`internal/config/config.go`)

```go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    Rules    RulesConfig    `mapstructure:"rules"`
    Vaidshala VaidshalaConfig `mapstructure:"vaidshala"`
    Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
    Port         int           `mapstructure:"port" default:"8100"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"30s"`
    WriteTimeout time.Duration `mapstructure:"write_timeout" default:"30s"`
}

type RulesConfig struct {
    Path          string        `mapstructure:"path" default:"./rules"`
    EnableCaching bool          `mapstructure:"enable_caching" default:"true"`
    CacheTTL      time.Duration `mapstructure:"cache_ttl" default:"5m"`
    WatchInterval time.Duration `mapstructure:"watch_interval" default:"30s"`
}

// Environment Variables:
// KB10_PORT=8100
// KB10_RULES_PATH=./rules
// KB10_LOG_LEVEL=info
// KB10_DB_HOST=localhost
// KB10_DB_PORT=5433
// KB10_DB_NAME=kb10_rules
// KB10_DB_USER=postgres
// KB10_DB_PASSWORD=password
// KB10_ENABLE_CACHING=true
// KB10_CACHE_TTL=5m
// VAIDSHALA_URL=http://localhost:8096
```

---

## Phase 2: Rule Engine Core (Day 2-3)

### 2.1 YAML Rule Loader (`internal/loader/yaml_loader.go`)

**Key Responsibilities:**
- Load rules from filesystem (recursive directory scan)
- Parse YAML syntax with validation
- Handle hot-reload via SIGHUP signal
- Detect and report rule conflicts
- Version tracking for A/B testing

```go
type YAMLLoader struct {
    rulesPath string
    store     *models.RuleStore
    validator *RuleValidator
    logger    *logrus.Logger
}

func (l *YAMLLoader) LoadRules() error {
    // 1. Walk rules directory
    // 2. Parse each YAML file
    // 3. Validate rule syntax
    // 4. Check for conflicts
    // 5. Add to store with priority ordering
}

func (l *YAMLLoader) SetupHotReload() {
    // Listen for SIGHUP
    // Reload rules without restart
    // Log reload events
}
```

### 2.2 Rule Store (`internal/models/store.go`)

**Key Features:**
- In-memory storage with concurrent access
- Multiple indexes (by type, category, severity, tags)
- Priority-sorted retrieval
- Statistics and metrics

```go
type RuleStore struct {
    rules      map[string]*Rule           // ID -> Rule
    byType     map[string][]*Rule         // Type -> Rules
    byCategory map[string][]*Rule         // Category -> Rules
    bySeverity map[string][]*Rule         // Severity -> Rules
    byTag      map[string][]*Rule         // Tag -> Rules
    mu         sync.RWMutex
}

func (s *RuleStore) GetByType(ruleType string) []*Rule
func (s *RuleStore) GetByCategory(category string) []*Rule
func (s *RuleStore) GetBySeverity(severity string) []*Rule
func (s *RuleStore) GetByTags(tags []string) []*Rule
func (s *RuleStore) GetStats() *StoreStats
```

### 2.3 Condition Evaluator (`internal/engine/evaluator.go`)

**20+ Operators Implementation:**

| Operator | Implementation | Example |
|----------|----------------|---------|
| `EQ` | Direct equality | `labs.potassium.value == 6.5` |
| `NEQ` | Not equal | `patient.status != "discharged"` |
| `GT/GTE` | Numeric comparison | `labs.creatinine.value >= 2.0` |
| `LT/LTE` | Numeric comparison | `vitals.bp_systolic < 90` |
| `CONTAINS` | Substring match | `diagnosis.code CONTAINS "I50"` |
| `IN` | List membership | `medication.class IN ["opioid", "benzo"]` |
| `BETWEEN` | Range check | `age BETWEEN 65 AND 85` |
| `EXISTS` | Field presence | `labs.troponin EXISTS` |
| `IS_NULL` | Null check | `medication.end_date IS_NULL` |
| `MATCHES` | Regex pattern | `icd10 MATCHES "^I[0-9]{2}"` |
| `AGE_GT` | Age calculation | `patient.dob AGE_GT 65` |
| `WITHIN_DAYS` | Temporal check | `labs.hba1c.date WITHIN_DAYS 90` |

```go
type ConditionEvaluator struct {
    vaidshalaClient *VaidshalaClient  // For CQL expressions
    logger          *logrus.Logger
}

func (e *ConditionEvaluator) Evaluate(condition *Condition, context *EvaluationContext) (bool, error) {
    switch condition.Operator {
    case OperatorEQ:
        return e.evaluateEquals(condition, context)
    case OperatorGTE:
        return e.evaluateGreaterOrEqual(condition, context)
    case OperatorCONTAINS:
        return e.evaluateContains(condition, context)
    case OperatorMATCHES:
        return e.evaluateRegex(condition, context)
    case OperatorWITHIN_DAYS:
        return e.evaluateWithinDays(condition, context)
    // ... 15 more operators
    }
}

func (e *ConditionEvaluator) EvaluateCQL(expression string, context *EvaluationContext) (bool, error) {
    // Call Vaidshala CQL Engine
}
```

### 2.4 Core Engine (`internal/engine/engine.go`)

```go
type RulesEngine struct {
    store       *models.RuleStore
    evaluator   *ConditionEvaluator
    executor    *ActionExecutor
    cache       *EvaluationCache
    db          *database.PostgresDB
    logger      *logrus.Logger
}

type EvaluationContext struct {
    PatientID   string                 `json:"patient_id"`
    EncounterID string                 `json:"encounter_id,omitempty"`
    Labs        map[string]interface{} `json:"labs,omitempty"`
    Vitals      map[string]interface{} `json:"vitals,omitempty"`
    Medications []interface{}          `json:"medications,omitempty"`
    Conditions  []interface{}          `json:"conditions,omitempty"`
    Patient     map[string]interface{} `json:"patient,omitempty"`
    Timestamp   time.Time              `json:"timestamp"`
}

type EvaluationResult struct {
    RuleID      string        `json:"rule_id"`
    RuleName    string        `json:"rule_name"`
    Triggered   bool          `json:"triggered"`
    Severity    string        `json:"severity,omitempty"`
    Actions     []ActionResult `json:"actions,omitempty"`
    ExecutedAt  time.Time     `json:"executed_at"`
    CacheHit    bool          `json:"cache_hit"`
}

func (e *RulesEngine) Evaluate(ctx context.Context, evalCtx *EvaluationContext) ([]*EvaluationResult, error) {
    // 1. Check cache
    // 2. Get applicable rules (by context)
    // 3. Sort by priority
    // 4. Evaluate conditions (AND/OR logic)
    // 5. Execute actions for triggered rules
    // 6. Store audit trail
    // 7. Cache results
    // 8. Return results
}

func (e *RulesEngine) EvaluateSpecific(ctx context.Context, ruleIDs []string, evalCtx *EvaluationContext) ([]*EvaluationResult, error)
func (e *RulesEngine) EvaluateByType(ctx context.Context, ruleType string, evalCtx *EvaluationContext) ([]*EvaluationResult, error)
func (e *RulesEngine) EvaluateByCategory(ctx context.Context, category string, evalCtx *EvaluationContext) ([]*EvaluationResult, error)
```

---

## Phase 3: API Layer (Day 3-4)

### 3.1 API Server (`internal/api/server.go`)

**Endpoints:**

#### Rule Evaluation
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/evaluate` | POST | Evaluate all rules for patient context |
| `/api/v1/evaluate/rules` | POST | Evaluate specific rules by ID |
| `/api/v1/evaluate/type/:type` | POST | Evaluate rules by type |
| `/api/v1/evaluate/category/:category` | POST | Evaluate rules by category |

#### Rule Management
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/rules` | GET | List all rules |
| `/api/v1/rules/:id` | GET | Get rule by ID |
| `/api/v1/rules` | POST | Create new rule |
| `/api/v1/rules/:id` | PUT | Update rule |
| `/api/v1/rules/:id` | DELETE | Delete rule |
| `/api/v1/rules/reload` | POST | Hot-reload rules from disk |
| `/api/v1/rules/stats` | GET | Get rule store statistics |

#### Alerts
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/alerts` | GET | List all alerts |
| `/api/v1/alerts/:id` | GET | Get alert by ID |
| `/api/v1/alerts/:id/acknowledge` | POST | Acknowledge alert |
| `/api/v1/alerts/patient/:patientId` | GET | Get patient alerts |

#### Health & Metrics
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/ready` | GET | Readiness check |
| `/metrics` | GET | Prometheus metrics |

```go
type Server struct {
    engine  *engine.RulesEngine
    config  *config.Config
    db      *database.PostgresDB
    logger  *logrus.Logger
    router  *gin.Engine
}

func NewServer(cfg *config.Config, engine *engine.RulesEngine, db *database.PostgresDB) *Server

func (s *Server) setupRoutes() {
    // Middleware
    s.router.Use(gin.Recovery())
    s.router.Use(s.requestIDMiddleware())
    s.router.Use(s.loggingMiddleware())

    // Health endpoints
    s.router.GET("/health", s.healthHandler)
    s.router.GET("/ready", s.readyHandler)
    s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

    // API v1
    v1 := s.router.Group("/api/v1")
    {
        // Evaluation
        v1.POST("/evaluate", s.evaluateHandler)
        v1.POST("/evaluate/rules", s.evaluateSpecificHandler)
        v1.POST("/evaluate/type/:type", s.evaluateByTypeHandler)
        v1.POST("/evaluate/category/:category", s.evaluateByCategoryHandler)

        // Rules management
        v1.GET("/rules", s.listRulesHandler)
        v1.GET("/rules/:id", s.getRuleHandler)
        v1.POST("/rules", s.createRuleHandler)
        v1.PUT("/rules/:id", s.updateRuleHandler)
        v1.DELETE("/rules/:id", s.deleteRuleHandler)
        v1.POST("/rules/reload", s.reloadRulesHandler)
        v1.GET("/rules/stats", s.ruleStatsHandler)

        // Alerts
        v1.GET("/alerts", s.listAlertsHandler)
        v1.GET("/alerts/:id", s.getAlertHandler)
        v1.POST("/alerts/:id/acknowledge", s.acknowledgeAlertHandler)
        v1.GET("/alerts/patient/:patientId", s.patientAlertsHandler)
    }
}
```

### 3.2 Request/Response Models

```go
// Evaluation Request
type EvaluateRequest struct {
    PatientID   string                 `json:"patient_id" binding:"required"`
    EncounterID string                 `json:"encounter_id,omitempty"`
    Labs        map[string]LabValue    `json:"labs,omitempty"`
    Vitals      map[string]VitalSign   `json:"vitals,omitempty"`
    Medications []MedicationContext    `json:"medications,omitempty"`
    Conditions  []ConditionContext     `json:"conditions,omitempty"`
    Patient     PatientContext         `json:"patient,omitempty"`
}

type LabValue struct {
    Value float64   `json:"value"`
    Unit  string    `json:"unit,omitempty"`
    Date  time.Time `json:"date,omitempty"`
}

// Evaluation Response
type EvaluateResponse struct {
    PatientID     string              `json:"patient_id"`
    RulesEvaluated int               `json:"rules_evaluated"`
    RulesTriggered int               `json:"rules_triggered"`
    Results       []*EvaluationResult `json:"results"`
    ExecutionTime float64            `json:"execution_time_ms"`
    Timestamp     time.Time          `json:"timestamp"`
}

// Alert Model
type Alert struct {
    ID           string    `json:"id"`
    RuleID       string    `json:"rule_id"`
    PatientID    string    `json:"patient_id"`
    Severity     string    `json:"severity"`
    Message      string    `json:"message"`
    Status       string    `json:"status"` // active, acknowledged, resolved
    AcknowledgedBy string  `json:"acknowledged_by,omitempty"`
    AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

---

## Phase 4: Database & Persistence (Day 4-5)

### 4.1 PostgreSQL Schema (`migrations/`)

```sql
-- 001_create_rules_table.sql
CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    category VARCHAR(100) NOT NULL,
    severity VARCHAR(50),
    status VARCHAR(50) DEFAULT 'ACTIVE',
    priority INTEGER DEFAULT 100,
    version VARCHAR(50),
    conditions JSONB NOT NULL,
    condition_logic VARCHAR(10) DEFAULT 'AND',
    actions JSONB NOT NULL,
    evidence JSONB,
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_rules_type ON rules(type);
CREATE INDEX idx_rules_category ON rules(category);
CREATE INDEX idx_rules_severity ON rules(severity);
CREATE INDEX idx_rules_status ON rules(status);
CREATE INDEX idx_rules_tags ON rules USING GIN(tags);

-- 002_create_alerts_table.sql
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id VARCHAR(100) NOT NULL REFERENCES rules(id),
    patient_id VARCHAR(100) NOT NULL,
    encounter_id VARCHAR(100),
    severity VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    context JSONB,
    status VARCHAR(50) DEFAULT 'active',
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alerts_patient ON alerts(patient_id);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_created ON alerts(created_at);

-- 003_create_audit_table.sql
CREATE TABLE IF NOT EXISTS rule_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id VARCHAR(100) NOT NULL,
    patient_id VARCHAR(100) NOT NULL,
    encounter_id VARCHAR(100),
    triggered BOOLEAN NOT NULL,
    context JSONB,
    result JSONB,
    execution_time_ms FLOAT,
    cache_hit BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_executions_rule ON rule_executions(rule_id);
CREATE INDEX idx_executions_patient ON rule_executions(patient_id);
CREATE INDEX idx_executions_created ON rule_executions(created_at);

-- Partitioning for performance (optional)
-- CREATE TABLE rule_executions_2024 PARTITION OF rule_executions
--     FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');
```

### 4.2 Database Operations (`internal/database/postgres.go`)

```go
type PostgresDB struct {
    pool   *pgxpool.Pool
    logger *logrus.Logger
}

// Rules Operations
func (db *PostgresDB) SaveRule(ctx context.Context, rule *models.Rule) error
func (db *PostgresDB) GetRule(ctx context.Context, id string) (*models.Rule, error)
func (db *PostgresDB) ListRules(ctx context.Context, filters *RuleFilters) ([]*models.Rule, error)
func (db *PostgresDB) UpdateRule(ctx context.Context, rule *models.Rule) error
func (db *PostgresDB) DeleteRule(ctx context.Context, id string) error

// Alerts Operations
func (db *PostgresDB) CreateAlert(ctx context.Context, alert *models.Alert) error
func (db *PostgresDB) GetAlert(ctx context.Context, id string) (*models.Alert, error)
func (db *PostgresDB) ListAlerts(ctx context.Context, filters *AlertFilters) ([]*models.Alert, error)
func (db *PostgresDB) AcknowledgeAlert(ctx context.Context, id, acknowledgedBy string) error
func (db *PostgresDB) GetPatientAlerts(ctx context.Context, patientID string) ([]*models.Alert, error)

// Audit Operations
func (db *PostgresDB) RecordExecution(ctx context.Context, execution *models.RuleExecution) error
func (db *PostgresDB) GetExecutionStats(ctx context.Context, ruleID string) (*models.ExecutionStats, error)
```

---

## Phase 5: Sample Rules & CQL (Day 5-6)

### 5.1 Critical Alerts Rule (`rules/safety/critical-alerts.yaml`)

```yaml
type: rules
version: "1.0.0"
description: Critical laboratory and vital sign alerts

rules:
  # Critical Hyperkalemia
  - id: ALERT-LAB-K-CRITICAL-HIGH
    name: Critical Hyperkalemia Alert
    description: Alerts when potassium level is critically elevated
    type: ALERT
    category: SAFETY
    severity: CRITICAL
    status: ACTIVE
    priority: 1

    conditions:
      - field: labs.potassium.value
        operator: GTE
        value: 6.5
        unit: mEq/L

    condition_logic: AND

    actions:
      - type: ALERT
        message: "CRITICAL: Potassium {{.Context.Labs.potassium.value}} mEq/L - Risk of cardiac arrhythmia"
        priority: STAT

      - type: ESCALATE
        parameters:
          level: PHYSICIAN
          urgency: STAT
        recipients:
          - attending_physician
          - charge_nurse
        channel: PAGER

    evidence:
      level: HIGH
      source: "AHA Guidelines 2023"

    tags:
      - electrolyte
      - critical
      - cardiac-risk

  # Critical Hypoglycemia
  - id: ALERT-LAB-GLUCOSE-CRITICAL-LOW
    name: Critical Hypoglycemia Alert
    description: Alerts when glucose is critically low
    type: ALERT
    category: SAFETY
    severity: CRITICAL
    status: ACTIVE
    priority: 1

    conditions:
      - field: labs.glucose.value
        operator: LT
        value: 50
        unit: mg/dL

    actions:
      - type: ALERT
        message: "CRITICAL: Glucose {{.Context.Labs.glucose.value}} mg/dL - Severe hypoglycemia"
        priority: STAT

      - type: ESCALATE
        parameters:
          level: RAPID_RESPONSE
        channel: OVERHEAD_PAGE

    evidence:
      level: HIGH
      source: "ADA Standards of Care 2024"

    tags:
      - glucose
      - critical
      - diabetic-emergency

  # Severe Hypotension
  - id: ALERT-VITAL-BP-CRITICAL-LOW
    name: Critical Hypotension Alert
    description: Alerts when blood pressure is critically low
    type: ALERT
    category: SAFETY
    severity: CRITICAL
    status: ACTIVE
    priority: 1

    conditions:
      - field: vitals.bp_systolic
        operator: LT
        value: 80

    actions:
      - type: ALERT
        message: "CRITICAL: SBP {{.Context.Vitals.bp_systolic}} mmHg - Severe hypotension"
        priority: STAT

      - type: ESCALATE
        parameters:
          level: RAPID_RESPONSE

    evidence:
      level: HIGH
      source: "Surviving Sepsis Campaign"

    tags:
      - vital-sign
      - critical
      - shock
```

### 5.2 Clinical Inference Rules (`rules/clinical/inference-rules.yaml`)

```yaml
type: rules
version: "1.0.0"
description: Clinical inference and derivation rules

rules:
  # Sepsis Inference
  - id: INFERENCE-SEPSIS-SUSPECTED
    name: Suspected Sepsis Inference
    description: Infer sepsis when SIRS criteria met with infection
    type: INFERENCE
    category: CLINICAL
    severity: HIGH
    status: ACTIVE
    priority: 10

    conditions:
      - field: vitals.temperature
        operator: GT
        value: 38.3
      - field: vitals.heart_rate
        operator: GT
        value: 90
      - field: labs.wbc.value
        operator: GT
        value: 12000
      - field: conditions
        operator: CONTAINS
        value: "infection"

    condition_logic: "((1 AND 2) OR 3) AND 4"

    actions:
      - type: INFERENCE
        message: "Suspected Sepsis - SIRS criteria met with infection source"
        parameters:
          inferred_condition: sepsis
          confidence: 0.85

      - type: RECOMMENDATION
        message: "Consider sepsis workup and lactate measurement"
        parameters:
          protocol: SEPSIS_BUNDLE

    evidence:
      level: MODERATE
      source: "Surviving Sepsis Campaign 2021"

    tags:
      - sepsis
      - sirs
      - inference

  # AKI Staging
  - id: DERIVATION-AKI-STAGE
    name: AKI Stage Derivation
    description: Calculate AKI stage based on creatinine change
    type: DERIVATION
    category: CLINICAL
    severity: MODERATE
    status: ACTIVE
    priority: 20

    conditions:
      - field: labs.creatinine.value
        operator: EXISTS
      - field: labs.creatinine_baseline.value
        operator: EXISTS

    actions:
      - type: DERIVATION
        parameters:
          calculation: |
            ratio = current_creatinine / baseline_creatinine
            if ratio >= 3.0: return "Stage 3"
            if ratio >= 2.0: return "Stage 2"
            if ratio >= 1.5: return "Stage 1"
            return "No AKI"
          output_field: aki_stage

    evidence:
      level: HIGH
      source: "KDIGO Guidelines 2012"

    tags:
      - aki
      - renal
      - derivation
```

### 5.3 CQL Library (`cql/tier-6-application/AlertRules-1.0.0.cql`)

```cql
library AlertRules version '1.0.0'

using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'

valueset "Potassium Lab": 'http://cardiofit.org/fhir/ValueSet/potassium-labs'
valueset "Glucose Lab": 'http://cardiofit.org/fhir/ValueSet/glucose-labs'

context Patient

// Critical Hyperkalemia Detection
define "Critical Hyperkalemia":
  exists(
    [Observation: "Potassium Lab"] O
    where O.status = 'final'
      and O.value as Quantity >= 6.5 'mEq/L'
      and O.effective.toInterval() during "Measurement Period"
  )

// Critical Hypoglycemia Detection
define "Critical Hypoglycemia":
  exists(
    [Observation: "Glucose Lab"] O
    where O.status = 'final'
      and O.value as Quantity < 50 'mg/dL'
      and O.effective.toInterval() during "Measurement Period"
  )

// Sepsis Screening
define "SIRS Criteria Met":
  Count({
    exists([Observation] O where O.code ~ 'TEMP' and O.value > 38.3 'Cel'),
    exists([Observation] O where O.code ~ 'HR' and O.value > 90 '/min'),
    exists([Observation] O where O.code ~ 'RR' and O.value > 20 '/min'),
    exists([Observation] O where O.code ~ 'WBC' and (O.value > 12000 '10*3/uL' or O.value < 4000 '10*3/uL'))
  } C where C = true) >= 2
```

---

## Phase 6: Testing & Deployment (Day 6-7)

### 6.1 Unit Tests (`tests/unit/evaluator_test.go`)

```go
func TestConditionEvaluator_NumericOperators(t *testing.T) {
    evaluator := NewConditionEvaluator(nil, logrus.New())

    tests := []struct {
        name      string
        condition Condition
        context   *EvaluationContext
        expected  bool
    }{
        {
            name: "GTE operator - true",
            condition: Condition{
                Field:    "labs.potassium.value",
                Operator: "GTE",
                Value:    6.5,
            },
            context: &EvaluationContext{
                Labs: map[string]interface{}{
                    "potassium": map[string]interface{}{"value": 6.8},
                },
            },
            expected: true,
        },
        {
            name: "GTE operator - false",
            condition: Condition{
                Field:    "labs.potassium.value",
                Operator: "GTE",
                Value:    6.5,
            },
            context: &EvaluationContext{
                Labs: map[string]interface{}{
                    "potassium": map[string]interface{}{"value": 5.0},
                },
            },
            expected: false,
        },
        // ... more test cases for all 20+ operators
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := evaluator.Evaluate(&tt.condition, tt.context)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestRulesEngine_CriticalAlert(t *testing.T) {
    // Test critical hyperkalemia triggers proper alert
    engine := setupTestEngine(t)

    ctx := &EvaluationContext{
        PatientID: "patient-001",
        Labs: map[string]interface{}{
            "potassium": map[string]interface{}{
                "value": 6.8,
                "unit":  "mEq/L",
            },
        },
    }

    results, err := engine.Evaluate(context.Background(), ctx)
    assert.NoError(t, err)
    assert.True(t, len(results) > 0)

    // Verify critical alert triggered
    var criticalFound bool
    for _, r := range results {
        if r.RuleID == "ALERT-LAB-K-CRITICAL-HIGH" && r.Triggered {
            criticalFound = true
            assert.Equal(t, "CRITICAL", r.Severity)
        }
    }
    assert.True(t, criticalFound, "Critical hyperkalemia alert should trigger")
}
```

### 6.2 Clinical Scenario Tests (`tests/clinical/scenarios_test.go`)

```go
func TestClinicalScenario_SepsisPatient(t *testing.T) {
    engine := setupTestEngine(t)

    // Simulated sepsis patient
    ctx := &EvaluationContext{
        PatientID: "sepsis-patient-001",
        Labs: map[string]interface{}{
            "wbc":      map[string]interface{}{"value": 15000},
            "lactate":  map[string]interface{}{"value": 4.2},
            "creatinine": map[string]interface{}{"value": 2.1},
        },
        Vitals: map[string]interface{}{
            "temperature": 38.9,
            "heart_rate":  112,
            "bp_systolic": 85,
        },
        Conditions: []interface{}{
            map[string]interface{}{"code": "pneumonia", "status": "active"},
        },
    }

    results, err := engine.Evaluate(context.Background(), ctx)
    assert.NoError(t, err)

    // Expected alerts and inferences
    expectedRules := map[string]bool{
        "ALERT-VITAL-BP-CRITICAL-LOW": true,
        "INFERENCE-SEPSIS-SUSPECTED":  true,
    }

    for _, r := range results {
        if r.Triggered {
            if _, expected := expectedRules[r.RuleID]; expected {
                delete(expectedRules, r.RuleID)
            }
        }
    }

    assert.Empty(t, expectedRules, "Expected rules not triggered: %v", expectedRules)
}
```

### 6.3 Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o kb-10-rules-engine ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/kb-10-rules-engine .
COPY --from=builder /app/rules ./rules

# Create non-root user
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8100

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8100/health || exit 1

ENTRYPOINT ["./kb-10-rules-engine"]
```

### 6.4 Docker Compose Integration

```yaml
# Add to docker-compose.kb-only.yml
  kb-10-rules-engine:
    build:
      context: ./kb-10-rules-engine
      dockerfile: Dockerfile
    container_name: kb-10-rules-engine
    ports:
      - "8100:8100"
    environment:
      - KB10_PORT=8100
      - KB10_RULES_PATH=/app/rules
      - KB10_LOG_LEVEL=info
      - KB10_DB_HOST=kb-postgres
      - KB10_DB_PORT=5432
      - KB10_DB_NAME=kb10_rules
      - KB10_DB_USER=postgres
      - KB10_DB_PASSWORD=${POSTGRES_PASSWORD:-password}
      - KB10_REDIS_HOST=kb-redis
      - KB10_REDIS_PORT=6379
      - KB10_ENABLE_CACHING=true
      - KB10_CACHE_TTL=5m
      - VAIDSHALA_URL=http://vaidshala:8096
    volumes:
      - ./kb-10-rules-engine/rules:/app/rules:ro
    depends_on:
      kb-postgres:
        condition: service_healthy
      kb-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8100/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - kb-network
    restart: unless-stopped
```

---

## Integration Points

### Consumers (Services that call KB-10)

| Service | Integration | Purpose |
|---------|-------------|---------|
| **KB-19 Protocol Orchestrator** | `/api/v1/evaluate/type/CONFLICT` | Conflict arbitration rules |
| **KB-4 Patient Safety** | `/api/v1/evaluate/category/SAFETY` | Safety threshold rules |
| **KB-16 Lab Interpretation** | `/api/v1/evaluate/type/ALERT` | Critical value rules |
| **KB-18 Governance Engine** | `/api/v1/evaluate/type/VALIDATION` | Approval workflow rules |

### Providers (Services KB-10 calls)

| Service | Integration | Purpose |
|---------|-------------|---------|
| **Vaidshala CQL Engine** | HTTP `POST /cql/evaluate` | CQL expression evaluation |
| **KB-8 Calculator Service** | HTTP `GET /v1/calculate/:type` | Risk score calculations |
| **KB-7 Terminology Service** | HTTP `GET /v1/codes/:system/:code` | Code lookups |

---

## Implementation Checklist

### Phase 1: Foundation ✅
- [ ] Create directory structure
- [ ] Implement `go.mod` with dependencies
- [ ] Create `internal/models/rule.go` with all types
- [ ] Create `internal/config/config.go` with Viper
- [ ] Create `cmd/server/main.go` entry point

### Phase 2: Engine Core ✅
- [ ] Implement `internal/loader/yaml_loader.go`
- [ ] Implement `internal/models/store.go`
- [ ] Implement `internal/engine/evaluator.go` (20+ operators)
- [ ] Implement `internal/engine/engine.go`
- [ ] Implement `internal/engine/executor.go`
- [ ] Implement `internal/engine/cache.go`

### Phase 3: API Layer ✅
- [ ] Implement `internal/api/server.go` with all endpoints
- [ ] Add request validation middleware
- [ ] Add error handling middleware
- [ ] Add logging and metrics middleware

### Phase 4: Database ✅
- [ ] Create migration files
- [ ] Implement `internal/database/postgres.go`
- [ ] Add audit trail functionality
- [ ] Implement alert management

### Phase 5: Rules & CQL ✅
- [ ] Create `rules/safety/critical-alerts.yaml`
- [ ] Create `rules/clinical/inference-rules.yaml`
- [ ] Create `rules/governance/governance-rules.yaml`
- [ ] Create CQL libraries

### Phase 6: Testing & Deployment ✅
- [ ] Write unit tests (80%+ coverage)
- [ ] Write integration tests
- [ ] Write clinical scenario tests
- [ ] Create Dockerfile
- [ ] Update docker-compose
- [ ] Update Makefile

---

## Metrics & Monitoring

### Prometheus Metrics

```go
var (
    rulesEvaluated = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kb10_rules_evaluated_total",
            Help: "Total number of rules evaluated",
        },
        []string{"rule_type", "category"},
    )

    rulesTriggered = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kb10_rules_triggered_total",
            Help: "Total number of rules triggered",
        },
        []string{"rule_id", "severity"},
    )

    evaluationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "kb10_evaluation_duration_seconds",
            Help:    "Time taken to evaluate rules",
            Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
        },
        []string{"endpoint"},
    )

    cacheHitRate = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "kb10_cache_hit_rate",
            Help: "Cache hit rate for rule evaluations",
        },
        []string{"cache_type"},
    )

    activeAlerts = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "kb10_active_alerts",
            Help: "Number of active (unacknowledged) alerts",
        },
    )
)
```

---

## Estimated Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| Phase 1: Foundation | Day 1-2 | Directory structure, models, config |
| Phase 2: Engine Core | Day 2-3 | YAML loader, evaluator, engine |
| Phase 3: API Layer | Day 3-4 | HTTP server, all endpoints |
| Phase 4: Database | Day 4-5 | PostgreSQL, migrations, audit |
| Phase 5: Rules & CQL | Day 5-6 | Sample rules, CQL libraries |
| Phase 6: Testing | Day 6-7 | Tests, Docker, deployment |

**Total: 5-7 days** for complete implementation with ~7,600 lines of code.

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| CQL integration complexity | Start with native operators, add CQL later |
| Performance with many rules | Implement caching early, use priority-based short-circuit |
| Rule conflicts | Build conflict detection into loader |
| Hot-reload stability | Use copy-on-write for rule store updates |
| Database bottleneck | Batch audit writes, use async patterns |

---

## Next Steps

1. **Create directory structure** using the template above
2. **Copy KB-1 patterns** as starting point for boilerplate
3. **Implement core models** first (types are foundation)
4. **Build evaluator** with test-driven development
5. **Add API endpoints** incrementally
6. **Write tests** alongside implementation
7. **Deploy to local Docker** environment for integration testing
