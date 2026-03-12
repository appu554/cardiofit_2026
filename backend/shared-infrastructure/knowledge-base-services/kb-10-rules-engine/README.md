# KB-10 Clinical Rules Engine

A high-performance, YAML-driven clinical rules engine for the CardioFit Clinical Decision Support System.

## Overview

KB-10 provides configurable business rules execution without code changes, enabling healthcare organizations to customize clinical decision support to their specific needs.

## Key Features

| Feature | Description |
|---------|-------------|
| **YAML-Driven Rules** | Define rules in YAML without code changes |
| **Hot Reload** | Reload rules via SIGHUP or API without service restart |
| **Rule Versioning** | Track rule versions and support A/B testing |
| **Conflict Detection** | Detect and resolve conflicting rules |
| **Priority Hierarchy** | Process rules in priority order |
| **Audit Trail** | Complete audit logging of all rule executions |
| **20+ Operators** | Comprehensive condition evaluation operators |
| **Caching** | Intelligent result caching for performance |

## Quick Start

### Using Docker Compose

```bash
# Start all services
docker-compose up -d

# Check health
curl http://localhost:8100/health

# Evaluate rules
curl -X POST http://localhost:8100/api/v1/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-001",
    "labs": {
      "potassium": {"value": 6.8, "unit": "mEq/L"}
    }
  }'
```

### Local Development

```bash
# Build
go build -o bin/kb-10-rules-engine ./cmd/server

# Run with environment variables
KB10_PORT=8100 \
KB10_DB_HOST=localhost \
KB10_DB_PORT=5433 \
KB10_DB_PASSWORD=password \
./bin/kb-10-rules-engine
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

## Condition Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `EQ` | Equals | `status EQ "active"` |
| `NEQ` | Not equals | `status NEQ "discharged"` |
| `GT/GTE` | Greater than | `potassium GTE 6.5` |
| `LT/LTE` | Less than | `bp_systolic LT 90` |
| `CONTAINS` | String contains | `diagnosis CONTAINS "sepsis"` |
| `IN` | Value in list | `med_class IN ["opioid", "benzo"]` |
| `BETWEEN` | Range check | `age BETWEEN [65, 85]` |
| `EXISTS` | Field exists | `troponin EXISTS` |
| `MATCHES` | Regex pattern | `icd10 MATCHES "^I[0-9]{2}"` |
| `AGE_GT` | Age greater than | `patient.dob AGE_GT 65` |
| `WITHIN_DAYS` | Date within N days | `hba1c.date WITHIN_DAYS 90` |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KB10_PORT` | `8100` | HTTP server port |
| `KB10_RULES_PATH` | `./rules` | Path to rules directory |
| `KB10_LOG_LEVEL` | `info` | Log level |
| `KB10_DB_HOST` | `localhost` | PostgreSQL host |
| `KB10_DB_PORT` | `5433` | PostgreSQL port |
| `KB10_DB_NAME` | `kb10_rules` | Database name |
| `KB10_DB_USER` | `postgres` | Database user |
| `KB10_DB_PASSWORD` | | Database password |
| `KB10_ENABLE_CACHING` | `true` | Enable evaluation caching |
| `KB10_CACHE_TTL` | `5m` | Cache TTL |
| `VAIDSHALA_URL` | `http://localhost:8096` | Vaidshala CQL Engine URL |

## Hot Reload

Rules can be reloaded without restart:

```bash
# Via SIGHUP signal
docker kill -s HUP kb-10-rules-engine

# Via API
curl -X POST http://localhost:8100/api/v1/rules/reload
```

## Directory Structure

```
kb-10-rules-engine/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── api/server.go            # HTTP API server
│   ├── config/config.go         # Configuration
│   ├── database/postgres.go     # PostgreSQL operations
│   ├── engine/
│   │   ├── engine.go            # Core rule engine
│   │   ├── evaluator.go         # Condition evaluator
│   │   ├── executor.go          # Action executor
│   │   └── cache.go             # Evaluation cache
│   ├── loader/yaml_loader.go    # YAML rule loader
│   ├── metrics/metrics.go       # Prometheus metrics
│   └── models/                  # Domain models
├── rules/                       # YAML rule definitions
│   ├── safety/
│   ├── clinical/
│   └── governance/
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Integration Points

**Consumers (call KB-10):**
- KB-19 Protocol Orchestrator - conflict arbitration rules
- KB-4 Patient Safety - safety threshold rules
- KB-16 Lab Interpretation - critical value rules
- KB-18 Governance Engine - approval workflow rules

**Providers (KB-10 calls):**
- Vaidshala CQL Engine - CQL expression evaluation
- KB-8 Calculator Service - risk score calculations
- KB-7 Terminology Service - code lookups

## License

Proprietary - Vaidshala Clinical Platform
