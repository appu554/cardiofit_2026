# Clinical Applications

> **Product Differentiation Layer** - Where innovation happens.

## Purpose

This repository contains the application engines that consume clinical knowledge:
- CDS Hooks service
- Medication Advisor
- Conditions Advisor
- Order Set Recommender
- AI Scribe Validator
- CDI Advisor

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  APPLICATION LAYER                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐    │
│  │  CDS Hooks   │   │  Medication  │   │  Conditions  │    │
│  │              │   │   Advisor    │   │   Advisor    │    │
│  └──────────────┘   └──────────────┘   └──────────────┘    │
│                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐    │
│  │  Order Set   │   │  AI Scribe   │   │    CDI       │    │
│  │ Recommender  │   │  Validator   │   │   Advisor    │    │
│  └──────────────┘   └──────────────┘   └──────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│               RUNTIME PLATFORM (APIs)                        │
│  • CQL Execution  • Terminology  • FHIR Data                │
└─────────────────────────────────────────────────────────────┘
```

## Applications

### CDS Hooks (`apps/cds-hooks/`)
HL7 CDS Hooks service for EHR integration.
- `patient-view` hook
- `order-select` hook
- `order-sign` hook

### Medication Advisor (`apps/medication-advisor/`)
Intelligent medication recommendations.
- Drug-drug interactions
- Dose optimization
- Therapeutic alternatives
- Formulary checking

### Conditions Advisor (`apps/conditions-advisor/`)
Condition management recommendations.
- Risk assessment
- Screening recommendations
- Care gap identification

### Order Set Recommender (`apps/order-set-recommender/`)
Clinical order set suggestions.
- Diagnosis-based recommendations
- Protocol adherence
- Evidence-based ordering

### AI Scribe Validator (`apps/ai-scribe-validator/`)
Clinical documentation validation.
- Note completeness
- Terminology accuracy
- Compliance checking

### CDI Advisor (`apps/cdi-advisor/`)
Clinical Documentation Improvement.
- Coding optimization
- Query generation
- DRG impact analysis

## Key Principle

> **Applications NEVER encode medical truth.**

All clinical logic comes from `clinical-knowledge-core` via the runtime platform. Applications only:
- Invoke runtime APIs
- Render explanations
- Handle user interactions
- Manage workflows

## Contracts

API contracts are defined in `contracts/`:
- `cql-response-schema.json` - Response format from CQL execution
- `evidence-envelope.schema.json` - Audit trail format

## UI Components

Reusable UI components in `ui/`:
- `clinician/` - Clinician-facing components
- `admin/` - Administrative components

## Development

```bash
# Install dependencies
npm install

# Run tests
npm test

# Start development server
npm run dev
```

## Integration with CardioFit

These applications integrate with existing CardioFit services:
- Flow2 Go Engine (scoring)
- Flow2 Rust Engine (rules)
- Safety Gateway (validation)
- Clinical Reasoning Service (ML)

## License

Proprietary - CardioFit Clinical Synthesis Hub
