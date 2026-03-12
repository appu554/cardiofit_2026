# Vaidshala - Clinical Knowledge Architecture

> **वैद्यशाला** (Vaidshala) - Sanskrit for "House of Healers/Physicians"

## Overview

Vaidshala is a **tiered clinical knowledge architecture** that separates clinical truth (knowledge) from execution (runtime) and application (products). This structure enables:

- **Clinical governance** - All medical logic is reviewed, versioned, and signed
- **Regional adaptation** - Support for India (IN) and Australia (AU) with different terminologies
- **Deterministic execution** - Same inputs always produce same outputs
- **Product innovation** - Applications built on top of stable clinical foundations

## Architecture: The Dependency Lattice

This is **not a linear stack** - it's a **compiler + runtime system**:

```
AUTHORING TIME (Global, non-PHI)
│
│  CQL / ValueSets / Calculators
│
▼
BUILD TIME (CI, signed artifacts)
│
│  CQL → ELM
│  ValueSet expansion
│  Manifest signing
│
▼
RUNTIME (Per-country, PHI-safe)
│
│  FHIR data + Redis ValueSets
│  Deterministic execution
│
▼
APPLICATION ENGINES
(CDS, Medication, AI Scribe, CDI)
```

## Repository Structure

### 1. `clinical-knowledge-core/`
**The Clinical Constitution** - Changes here are rare, reviewed, and signed.

Contains:
- Tier 0: FHIR Foundation (ModelInfo, FHIRHelpers)
- Tier 0.5: Terminology (SNOMED, ICD-10, LOINC, RxNorm, ValueSets)
- Tier 1: Primitives (interval helpers, encounter helpers)
- Tier 2: CQM Infrastructure (quality measure patterns)
- Tier 3: Domain Commons (eGFR, SOFA, dose calculators)
- Tier 4: Guidelines (CMS eCQM, WHO, ICMR, RACGP)
- Tier 5: Regional Adapters (India/Australia thresholds, drug availability)

### 2. `clinical-runtime-platform/`
**The Deterministic Engine** - Executes clinical logic without modification.

Contains:
- FHIR Server (HAPI FHIR)
- CQL Executor (cqf-ruler wrapper)
- Terminology Cache (Redis-backed)
- Audit Service (evidence envelopes)
- Infrastructure (K8s, Terraform, observability)

### 3. `clinical-applications/`
**Product Differentiation** - Where innovation happens.

Contains:
- CDS Hooks service
- Medication Advisor
- Conditions Advisor
- Order Set Recommender
- AI Scribe Validator
- CDI Advisor

## Tier Dependency Model

```
Tier 6  → depends on T5, T4, T3, T2, T1, T0, T0.5
Tier 5  → depends on       T4, T3, T2, T1, T0, T0.5
Tier 4  → depends on           T3, T2, T1, T0, T0.5
Tier 3  → depends on               T2, T1, T0, T0.5
Tier 2  → depends on                   T1, T0, T0.5
Tier 1  → depends on                       T0, T0.5
Tier 0  → depends on                           T0.5
Tier 0.5 → foundational (no dependencies above)
```

**Key Insight**: If Tier 0.5 (Terminology) changes, EVERY tier must adjust.

## Control Plane vs Data Plane

### Control Plane (Global, NO PHI)
- Git repositories
- CI/CD pipelines
- CQL source code
- Compiled ELM
- ValueSet definitions
- Signed manifests

### Data Plane (Per Country: IN, AU)
- FHIR store
- Redis (expanded value sets)
- Runtime CQL execution
- Audit logs
- KMS (regional keys)

## Getting Started

1. **For Clinical Authors**: Work in `clinical-knowledge-core/`
2. **For Platform Engineers**: Work in `clinical-runtime-platform/`
3. **For Application Developers**: Work in `clinical-applications/`

## Governance

All changes to `clinical-knowledge-core/` require:
- 1 clinical reviewer (physician or pharmacist)
- 1 informatics reviewer (engineer)
- Evidence source linked
- No silent threshold changes

See [clinical-knowledge-core/GOVERNANCE.md](clinical-knowledge-core/GOVERNANCE.md) for details.

## Regional Support

- **India (IN)**: NLEM drugs, ICD-10-WHO, ICMR guidelines, BMI ≥23 thresholds
- **Australia (AU)**: PBS/AMT drugs, ICD-10-AM, RACGP guidelines, standard thresholds

## License

Proprietary - CardioFit Clinical Synthesis Hub
