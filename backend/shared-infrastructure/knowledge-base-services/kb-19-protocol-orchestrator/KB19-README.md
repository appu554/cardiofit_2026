# KB-19 Protocol Orchestrator

**The Decision Synthesis Brain** - Clinical Protocol Orchestration Service

## Overview

KB-19 is the central orchestration layer for clinical protocol arbitration. It is **NOT another protocol calculator** - it is the decision synthesis engine that:

- Consumes clinical truth from Vaidshala CQL Engine
- Orchestrates KB-3 (temporal), KB-8 (calculators), KB-12 (ordersets), KB-14 (governance)
- Performs **arbitration** when multiple protocols conflict
- Produces evidence-backed recommendations with ACC/AHA Class grading
- Generates audit trails for regulatory compliance (FDA 21 CFR Part 11)

## Architecture Position

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         KNOWLEDGE LAYER (Tier 4 CQL)                        │
│                    "The Truth" - What is clinically true?                   │
│                           (Vaidshala CQL Engine)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                         EXECUTION LAYER (Tier 6 Go)                         │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    KB-19 Protocol Orchestrator                       │   │
│  │              "The Brain" - Decision Synthesis Engine                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│         │              │              │              │                      │
│         ▼              ▼              ▼              ▼                      │
│     ┌───────┐     ┌───────┐     ┌────────┐    ┌────────┐                   │
│     │ KB-3  │     │ KB-8  │     │ KB-12  │    │ KB-14  │                   │
│     │Temporal│    │ Calc  │     │OrderSet│    │Govern  │                   │
│     └───────┘     └───────┘     └────────┘    └────────┘                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Design Principles

1. **KB-19 is STATELESS** - No protocol logic lives here
2. **KB-19 DELEGATES truth** - CQL Engine owns clinical truth
3. **KB-19 OWNS synthesis** - Arbitration is the unique value
4. **KB-19 EXPLAINS itself** - Every decision has narrative + evidence
5. **KB-19 BINDS execution** - But never executes directly

## 8-Step Arbitration Pipeline

1. **Collect** - Gather candidate protocols based on CQL truth flags
2. **Filter** - Remove contraindicated or inapplicable protocols
3. **Conflict** - Identify conflicts between applicable protocols
4. **Priority** - Apply priority hierarchy (Emergency > Acute > Chronic)
5. **Safety** - Apply safety gatekeepers (ICU, pregnancy, renal, etc.)
6. **Grade** - Assign ACC/AHA recommendation class (I, IIa, IIb, III)
7. **Narrative** - Generate human-readable explanation
8. **Bind** - Bind to execution services (KB-3, KB-12, KB-14)

## API Endpoints

### Health & Status
```
GET  /health              - Health check
GET  /ready               - Readiness check (includes dependent services)
GET  /metrics             - Prometheus metrics
```

### Protocol Orchestration
```
POST /api/v1/execute      - Execute full protocol arbitration
POST /api/v1/evaluate     - Evaluate single protocol
```

### Protocol Management
```
GET  /api/v1/protocols    - List available protocols
GET  /api/v1/protocols/:id - Get protocol details
```

### Decision History
```
GET  /api/v1/decisions/:patientId - Get decisions for patient
GET  /api/v1/bundle/:id   - Get recommendation bundle by ID
```

### Conflict Matrix
```
GET  /api/v1/conflicts    - List known conflict rules
GET  /api/v1/conflicts/:protocolId - Get conflicts for protocol
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| PORT | 8099 | Server port |
| ENVIRONMENT | development | Environment name |
| LOG_LEVEL | info | Log level (debug, info, warn, error) |
| DB_HOST | localhost | PostgreSQL host |
| DB_PORT | 5432 | PostgreSQL port |
| DB_NAME | kb_protocol_orchestrator | Database name |
| KB3_URL | http://localhost:8087 | KB-3 Guidelines URL |
| KB8_URL | http://localhost:8088 | KB-8 Calculator URL |
| KB12_URL | http://localhost:8092 | KB-12 OrderSets URL |
| KB14_URL | http://localhost:8094 | KB-14 Governance URL |
| VAIDSHALA_CQL_URL | http://localhost:9000 | Vaidshala CQL Engine URL |

## Running Locally

```bash
# Install dependencies
go mod download

# Run the service
go run cmd/server/main.go

# Or build and run
go build -o kb-19 ./cmd/server
./kb-19
```

## Docker

```bash
# Build
docker build -t kb-19-protocol-orchestrator .

# Run
docker run -p 8099:8099 \
  -e DB_HOST=host.docker.internal \
  -e VAIDSHALA_CQL_URL=http://host.docker.internal:9000 \
  kb-19-protocol-orchestrator
```

## Example Request

```bash
curl -X POST http://localhost:8099/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "encounter_id": "550e8400-e29b-41d4-a716-446655440001",
    "context": {
      "cql_truth_flags": {
        "HasSepsis": true,
        "HasHFrEF": true,
        "HasAKI": false
      },
      "calculator_scores": {
        "SOFA": 8,
        "CHA2DS2VASc": 4
      }
    }
  }'
```

## Example Response

The response is a `RecommendationBundle` containing:

- Arbitrated decisions with evidence envelopes
- Conflict resolutions with clinical rationale
- Safety gates applied
- Human-readable narrative summary
- Processing metrics

## Domain Models

| Model | Purpose |
|-------|---------|
| PatientContext | Complete clinical snapshot for arbitration input |
| ProtocolDescriptor | Protocol metadata (triggers, contraindications, priority) |
| ProtocolEvaluation | Per-protocol applicability assessment |
| EvidenceEnvelope | Legal protection with inference chain and checksums |
| ArbitratedDecision | Final decision with evidence and safety flags |
| RecommendationBundle | Complete output with all decisions and narrative |

## Dependencies

- **Vaidshala CQL Engine** (required) - Clinical truth evaluation
- **KB-3 Guidelines** - Temporal binding (scheduling, deadlines)
- **KB-8 Calculator** - Risk scores (CHA2DS2-VASc, SOFA, etc.)
- **KB-12 OrderSets** - Order set activation
- **KB-14 Governance** - Task escalation and audit
- **PostgreSQL** - Decision audit storage
- **Redis** (optional) - Caching

## License

Proprietary - CardioFit Clinical Synthesis Hub
