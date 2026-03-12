# KB-10 Clinical Rules Engine

## Overview

KB-10 is the Clinical Rules Engine for the Clinical Decision Support System. It provides configurable business rules execution without code changes, enabling healthcare organizations to customize clinical decision support to their specific needs.

## Key Capabilities

| Capability | Description |
|------------|-------------|
| **YAML-Driven Rules** | Define rules in YAML without code changes |
| **Hot Reload** | Reload rules via SIGHUP without service restart |
| **Rule Versioning** | Track rule versions and support A/B testing |
| **Conflict Detection** | Detect and resolve conflicting rules |
| **Priority Hierarchy** | Process rules in priority order |
| **Audit Trail** | Complete audit logging of all rule executions |

## Rule Types

| Type | Code | Description |
|------|------|-------------|
| Alert | `ALERT` | Generates notifications for specific conditions |
| Inference | `INFERENCE` | Derives new facts from existing data |
| Validation | `VALIDATION` | Validates actions before execution |
| Escalation | `ESCALATION` | Triggers escalation workflows |
| Suppression | `SUPPRESSION` | Suppresses other rules/alerts |
| Derivation | `DERIVATION` | Calculates derived values |
| Recommendation | `RECOMMENDATION` | Suggests clinical actions |
| Conflict | `CONFLICT` | Detects protocol conflicts |

## Architecture

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

## API Endpoints

### Rule Evaluation

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/evaluate` | POST | Evaluate all rules for patient context |
| `/api/v1/evaluate/rules` | POST | Evaluate specific rules by ID |
| `/api/v1/evaluate/type/:type` | POST | Evaluate rules by type |
| `/api/v1/evaluate/category/:category` | POST | Evaluate rules by category |

### Rule Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/rules` | GET | List all rules |
| `/api/v1/rules/:id` | GET | Get rule by ID |
| `/api/v1/rules` | POST | Create new rule |
| `/api/v1/rules/:id` | PUT | Update rule |
| `/api/v1/rules/:id` | DELETE | Delete rule |
| `/api/v1/rules/reload` | POST | Hot-reload rules from disk |
| `/api/v1/rules/stats` | GET | Get rule store statistics |

### Alerts

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/alerts` | GET | List all alerts |
| `/api/v1/alerts/:id` | GET | Get alert by ID |
| `/api/v1/alerts/:id/acknowledge` | POST | Acknowledge alert |
| `/api/v1/alerts/patient/:patientId` | GET | Get patient alerts |

## Rule Definition Example

```yaml
# rules/safety/critical-alerts.yaml
type: rules
version: "1.0.0"

rules:
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
        message: "CRITICAL: Potassium {{.Context.Labs.potassium.value}} mEq/L"
        priority: STAT

      - type: ESCALATE
        parameters:
          level: PHYSICIAN
          urgency: STAT
        recipients:
          - attending_physician
        channel: PAGER

    evidence:
      level: HIGH
      source: "AHA Guidelines"

    tags:
      - electrolyte
      - critical
      - cardiac-risk
```

## Condition Operators

| Operator | Code | Description |
|----------|------|-------------|
| Equals | `EQ` | Field equals value |
| Not Equals | `NEQ` | Field not equals value |
| Greater Than | `GT` | Field greater than value |
| Greater or Equal | `GTE` | Field greater than or equal |
| Less Than | `LT` | Field less than value |
| Less or Equal | `LTE` | Field less than or equal |
| Contains | `CONTAINS` | String contains substring |
| In | `IN` | Value in list |
| Between | `BETWEEN` | Value between min and max |
| Exists | `EXISTS` | Field exists |
| Is Null | `IS_NULL` | Field is null |
| Matches | `MATCHES` | Regex pattern match |
| Age GT | `AGE_GT` | Age greater than years |
| Within Days | `WITHIN_DAYS` | Date within N days |

## Integration Points

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        KB-10 INTEGRATION MAP                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  CONSUMERS                        │  PROVIDERS                             │
│  ═══════════════                  │  ═══════════════                       │
│                                   │                                         │
│  KB-19 Protocol Orchestrator ────▶│  Vaidshala CQL Engine                  │
│    • Conflict arbitration rules   │    • CQL expression evaluation         │
│                                   │                                         │
│  KB-4 Patient Safety ────────────▶│  KB-8 Calculator Service               │
│    • Safety threshold rules       │    • Risk score calculations           │
│                                   │                                         │
│  KB-16 Lab Interpretation ───────▶│  KB-7 Terminology Service              │
│    • Critical value rules         │    • Code lookups                       │
│                                   │                                         │
│  KB-18 Governance Engine ────────▶│  PostgreSQL                            │
│    • Approval workflow rules      │    • Audit storage                      │
│                                   │                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
kb-10-rules-engine/
├── cmd/server/
│   └── main.go                     # Entry point
├── internal/
│   ├── api/
│   │   └── server.go               # HTTP API server (769 lines)
│   ├── config/
│   │   └── config.go               # Configuration management
│   ├── database/
│   │   └── postgres.go             # PostgreSQL operations
│   ├── engine/
│   │   ├── engine.go               # Core rule engine
│   │   ├── evaluator.go            # Condition evaluator
│   │   ├── executor.go             # Action executor
│   │   └── cache.go                # Evaluation cache
│   ├── loader/
│   │   └── yaml_loader.go          # YAML rule loader
│   ├── metrics/
│   │   └── metrics.go              # Metrics collection
│   └── models/
│       ├── rule.go                 # Rule domain models
│       └── store.go                # Rule store
├── rules/
│   ├── safety/
│   │   ├── critical-alerts.yaml    # Critical lab/vital alerts
│   │   └── medication-validation.yaml
│   ├── clinical/
│   │   ├── inference-rules.yaml    # Clinical inference rules
│   │   └── escalation-rules.yaml   # Escalation pathways
│   └── governance/
│       └── governance-rules.yaml   # Approval workflows
├── cql/
│   └── tier-6-application/
│       ├── ClinicalRulesEngine-1.0.0.cql
│       ├── AlertRules-1.0.0.cql
│       └── EscalationRules-1.0.0.cql
├── tests/
│   └── engine_test.go              # Unit tests
├── go.mod
├── Dockerfile
└── README.md
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KB10_PORT` | `8100` | HTTP server port |
| `KB10_RULES_PATH` | `./rules` | Path to rules directory |
| `KB10_LOG_LEVEL` | `info` | Log level |
| `KB10_DB_HOST` | `localhost` | PostgreSQL host |
| `KB10_DB_PORT` | `5432` | PostgreSQL port |
| `KB10_DB_NAME` | `kb10_rules` | Database name |
| `KB10_DB_USER` | `postgres` | Database user |
| `KB10_DB_PASSWORD` | | Database password |
| `KB10_ENABLE_CACHING` | `true` | Enable evaluation caching |
| `KB10_CACHE_TTL` | `5m` | Cache TTL |
| `VAIDSHALA_URL` | `http://localhost:8096` | Vaidshala CQL Engine URL |

## Quick Start

```bash
# Build
docker build -t kb-10-rules-engine .

# Run
docker run -d \
  -p 8100:8100 \
  -e KB10_DB_HOST=postgres \
  -e KB10_DB_PASSWORD=secret \
  -v $(pwd)/rules:/app/rules \
  kb-10-rules-engine

# Test evaluation
curl -X POST http://localhost:8100/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-001",
    "labs": {
      "potassium": {"value": 6.8}
    }
  }'
```

## Hot Reload

Rules can be reloaded without restart:

```bash
# Via SIGHUP
docker kill -s HUP <container_id>

# Via API
curl -X POST http://localhost:8100/api/v1/rules/reload
```

## Implementation Status

| Component | Status | Lines |
|-----------|--------|-------|
| Core Engine | ✅ Complete | ~2,500 |
| YAML Loader | ✅ Complete | ~500 |
| API Server | ✅ Complete | ~800 |
| Database | ✅ Complete | ~500 |
| Models | ✅ Complete | ~800 |
| CQL Libraries | ✅ Complete | ~600 |
| Sample Rules | ✅ Complete | ~1,500 |
| Tests | ✅ Complete | ~400 |
| **Total** | **✅ Complete** | **~7,600** |

## Related KBs

- **KB-4**: Patient Safety - Consumes safety threshold rules
- **KB-16**: Lab Interpretation - Consumes critical value rules
- **KB-18**: Governance Engine - Consumes approval workflow rules
- **KB-19**: Protocol Orchestrator - Consumes conflict arbitration rules

## License

Proprietary - Vaidshala Clinical Platform
