# Vaidshala Architecture

## The Mental Model

> **This is a compiler + runtime system, not an app stack.**

The Vaidshala architecture treats clinical knowledge like source code:
- **Authored** by clinical experts
- **Compiled** into executable artifacts
- **Executed** deterministically at runtime
- **Consumed** by applications

## Tier Structure

```
               ┌───────────────────────────────┐
               │   Tier 6 — App Engines        │
               │   (Medication, CDS, CDI, AI)  │
               └───────────────▲───────────────┘
                               │ uses
               ┌───────────────┴───────────────┐
               │   Tier 5 — Regional Shims     │
               │   (India, Australia adapters) │
               └───────────────▲───────────────┘
                               │ maps
               ┌───────────────┴───────────────┐
               │   Tier 4 — Measures/Guidelines│
               │   (CMS, WHO, specialty CPGs)  │
               └───────────────▲───────────────┘
                               │ imports
               ┌───────────────┴───────────────┐
               │   Tier 3 — Domain Commons     │
               │   (Calculators, patterns)     │
               └───────────────▲───────────────┘
                               │ depends on
               ┌───────────────┴───────────────┐
               │   Tier 2 — CQM Infrastructure │
               └───────────────▲───────────────┘
                               │ uses
               ┌───────────────┴───────────────┐
               │   Tier 1 — Primitives         │
               └───────────────▲───────────────┘
                               │ uses
               ┌───────────────┴───────────────┐
               │   Tier 0 — FHIR Foundation    │
               └───────────────▲───────────────┘
                               │ depends on
               ┌───────────────┴───────────────┐
               │  Tier 0.5 — Terminology Layer │
               └───────────────────────────────┘
```

## Tier Responsibilities

### Tier 0: FHIR Foundation
- **Provides**: Grammar of clinical logic
- **Contains**: ModelInfo, FHIRHelpers, choice types
- **Does NOT provide**: Codes, value sets, clinical concepts

### Tier 0.5: Terminology Layer
- **Provides**: Dictionary of the system
- **Contains**: SNOMED CT, ICD-10, LOINC, RxNorm, AMT, NLEM
- **Critical**: Without terminology, all logic returns empty sets

### Tier 1: Primitives
- **Provides**: Utility functions
- **Contains**: Interval helpers, encounter helpers, medication helpers
- **Examples**: `Condition.onset.toInterval()`, `MostRecentObservation()`

### Tier 2: CQM Infrastructure
- **Provides**: Quality measure patterns
- **Contains**: CQMCommon, CumulativeMedicationDuration, population criteria
- **Purpose**: Reusable patterns for quality measurement

### Tier 3: Domain Commons
- **Provides**: Clinical calculators
- **Contains**: eGFR, SOFA, qSOFA, ASCVD, HbA1c calculators
- **Purpose**: Domain-specific computations

### Tier 4: Guidelines
- **Provides**: Clinical rules
- **Contains**: CMS eCQMs, WHO guidelines, ICMR, RACGP
- **Purpose**: Evidence-based clinical recommendations

### Tier 5: Regional Adapters
- **Provides**: Localization
- **Contains**: India/Australia specific thresholds, drug availability
- **Examples**: BMI ≥23 (India), PBS formulary (Australia)

### Tier 6: Application Engines
- **Provides**: End-user functionality
- **Contains**: CDS Hooks, Medication Advisor, AI Scribe Validator
- **Purpose**: Product-level features

## Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     AUTHORING TIME                               │
├─────────────────────────────────────────────────────────────────┤
│  clinical-knowledge-core/                                        │
│  ├── CQL source files                                           │
│  ├── ValueSet definitions (JSON/XML)                            │
│  └── Calculator implementations                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      BUILD TIME (CI/CD)                          │
├─────────────────────────────────────────────────────────────────┤
│  1. CQL → ELM compilation                                        │
│  2. ValueSet expansion                                           │
│  3. Manifest generation                                          │
│  4. Digital signing (Ed25519)                                    │
│  5. Artifact publishing                                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      RUNTIME (Per Region)                        │
├─────────────────────────────────────────────────────────────────┤
│  clinical-runtime-platform/                                      │
│  ├── Pull signed artifacts                                       │
│  ├── Load ValueSets into Redis                                   │
│  ├── Execute CQL against FHIR data                              │
│  └── Generate evidence envelopes                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      APPLICATION LAYER                           │
├─────────────────────────────────────────────────────────────────┤
│  clinical-applications/                                          │
│  ├── Invoke runtime APIs                                         │
│  ├── Render explanations                                         │
│  ├── Handle user interactions                                    │
│  └── Never encode medical truth                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Key Principles

### 1. Separation of Concerns
- **Knowledge** (clinical-knowledge-core): What is clinically true
- **Execution** (clinical-runtime-platform): How to run the logic
- **Presentation** (clinical-applications): How to show results

### 2. Immutability
- Published artifacts are never modified
- New versions are published, old versions archived
- Rollback = deploy previous version

### 3. Determinism
- Same inputs always produce same outputs
- No hidden state or side effects
- Auditable and reproducible

### 4. Regional Isolation
- Each region has its own data plane
- PHI never leaves regional boundaries
- Terminology and thresholds adapted per region

## Integration with CardioFit

Vaidshala consumes and is consumed by existing CardioFit services:

```
┌─────────────────────────────────────────────────────────────────┐
│                     CARDIOFIT PLATFORM                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐      ┌──────────────────────────────────┐ │
│  │ KB-7 Terminology │ ───▶ │ vaidshala/tier-0.5-terminology   │ │
│  └──────────────────┘      └──────────────────────────────────┘ │
│                                                                  │
│  ┌──────────────────┐      ┌──────────────────────────────────┐ │
│  │ KB-3 Guidelines  │ ───▶ │ vaidshala/tier-4-guidelines      │ │
│  └──────────────────┘      └──────────────────────────────────┘ │
│                                                                  │
│  ┌──────────────────┐      ┌──────────────────────────────────┐ │
│  │ Flow2 Engines    │ ◀─── │ clinical-applications/apps/      │ │
│  └──────────────────┘      └──────────────────────────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Security Model

### Artifact Signing
- All published artifacts are signed with Ed25519
- Runtime verifies signatures before loading
- Unauthorized modifications rejected

### Audit Trail
- Every clinical decision generates evidence envelope
- Envelopes link to: input data, logic version, output, timestamp
- Immutable storage for compliance

### Regional Compliance
- HIPAA (US), DISHA (India), Privacy Act (Australia)
- Data residency enforced at infrastructure level
- Encryption at rest and in transit
