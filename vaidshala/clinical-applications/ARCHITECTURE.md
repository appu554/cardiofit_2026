# Clinical Applications Architecture

## Design Principles

### 1. Applications Don't Own Clinical Logic
```
┌─────────────────────────────────────────────────────────────┐
│                        APPLICATION                           │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  ❌ NO clinical thresholds                          │    │
│  │  ❌ NO drug dosing calculations                      │    │
│  │  ❌ NO diagnosis criteria                            │    │
│  │  ❌ NO value set definitions                         │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  ✓ Call runtime APIs                                │    │
│  │  ✓ Render explanations                              │    │
│  │  ✓ Handle user interactions                         │    │
│  │  ✓ Manage workflows                                 │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 2. API-First Integration
All clinical operations go through the runtime platform APIs.

### 3. Evidence-Driven UI
Every recommendation shows its evidence trail.

## Application Patterns

### CDS Hooks Pattern
```
EHR Event → CDS Hook Service → Runtime API → Cards Response → EHR
```

### Advisor Pattern
```
User Request → Context Builder → Runtime API → Recommendation → UI
```

### Validator Pattern
```
Document → Validation Request → Runtime API → Issues/Suggestions → UI
```

## Request Flow

```
┌─────────────────┐
│   User Action   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│  Application    │────▶│ Patient Context │
│   Controller    │     │    Builder      │
└────────┬────────┘     └─────────────────┘
         │
         ▼
┌─────────────────┐
│  Runtime API    │
│   Invocation    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│  Response       │────▶│ Evidence        │
│  Processing     │     │ Extraction      │
└────────┬────────┘     └─────────────────┘
         │
         ▼
┌─────────────────┐
│  UI Rendering   │
└─────────────────┘
```

## Component Structure

### Each Application Has
```
apps/{app-name}/
├── src/
│   ├── controllers/     # Request handlers
│   ├── services/        # Business logic
│   ├── clients/         # Runtime API clients
│   ├── models/          # Data models
│   └── views/           # Response renderers
├── tests/
├── config/
└── README.md
```

### Shared Components
```
orchestration/
├── patient-context-builder/
│   ├── assembler.ts     # Builds patient context
│   └── cache.ts         # Context caching
└── rule-invocation-engine/
    ├── invoker.ts       # Calls runtime APIs
    └── retry.ts         # Retry logic

contracts/
├── cql-response-schema.json
└── evidence-envelope.schema.json

ui/
├── clinician/
│   ├── recommendation-card/
│   ├── evidence-panel/
│   └── alert-banner/
└── admin/
    ├── configuration/
    └── audit-viewer/
```

## CDS Hooks Implementation

### Supported Hooks
| Hook | Trigger | Use Case |
|------|---------|----------|
| `patient-view` | Chart opened | Background alerts |
| `order-select` | Order selected | Interaction checks |
| `order-sign` | Order signed | Final validation |
| `encounter-start` | Visit begins | Care reminders |
| `encounter-discharge` | Discharge | Reconciliation |

### Response Cards
```json
{
  "cards": [
    {
      "uuid": "card-uuid",
      "summary": "Drug interaction detected",
      "indicator": "critical",
      "source": {
        "label": "Vaidshala Clinical Engine",
        "url": "https://vaidshala.internal/evidence/123"
      },
      "suggestions": [
        {
          "label": "Change to alternative",
          "uuid": "suggestion-uuid",
          "actions": [...]
        }
      ],
      "overrideReasons": [
        {
          "code": "benefit-outweighs-risk",
          "display": "Clinical benefit outweighs risk"
        }
      ]
    }
  ]
}
```

## Evidence Display Pattern

Every recommendation includes traceable evidence:

```
┌─────────────────────────────────────────────────────────────┐
│  ⚠️ ALERT: Consider dose reduction                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Recommendation: Reduce metformin to 500mg BID              │
│                                                              │
│  Evidence:                                                   │
│  ├── Patient eGFR: 42 mL/min (CKD Stage 3b)                │
│  ├── Guideline: KDIGO CKD 2024, Section 4.2                │
│  ├── Rule: RenalDoseAdjust v1.2.0                          │
│  └── Executed: 2024-01-15T10:30:00Z                        │
│                                                              │
│  [Accept] [Override with reason] [Dismiss]                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Integration Points

### With CardioFit Services
```
clinical-applications/
        │
        ├──▶ Flow2 Go Engine (scoring/ranking)
        ├──▶ Flow2 Rust Engine (rule evaluation)
        ├──▶ Safety Gateway (clinical validation)
        ├──▶ Clinical Reasoning Service (ML inference)
        └──▶ KB Services (terminology, guidelines)
```

### API Contracts
All integrations use well-defined contracts:
- OpenAPI for REST APIs
- GraphQL schemas for federation
- JSON Schema for data validation
