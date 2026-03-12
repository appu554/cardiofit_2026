# Medication Advisor Engine

> **Tier 6 Application Engine** - Clinical Decision Support for Medication Selection

## Overview

The Medication Advisor Engine is a **Tier 6 Application Engine** in the Vaidshala Clinical Knowledge Architecture. It provides intelligent medication recommendations through a **Calculate → Validate → Commit** workflow with full audit trail compliance.

### Position in Tier Architecture

```
Tier 0   │ FHIR Foundation (ModelInfo, FHIRHelpers)
Tier 0.5 │ Terminology (SNOMED, ICD-10, LOINC, RxNorm)
Tier 1   │ Primitives (interval helpers, utilities)
Tier 2   │ CQM Infrastructure (quality measure patterns)
Tier 3   │ Domain Commons (eGFR, SOFA, dose calculators)
Tier 4   │ Guidelines (CMS eCQM, WHO, ICMR, RACGP)
Tier 5   │ Regional Adapters (India/Australia)
─────────┼──────────────────────────────────────────────
Tier 6   │ APPLICATION ENGINES ◄── YOU ARE HERE
         │   ├── CQL Engine (Clinical Truth)
         │   ├── Measure Engine (Care Gaps)
         │   └── Medication Advisor Engine (Drug Recommendations)
```

## Purpose

**Question it Answers**: "Given this patient context, what medication should be prescribed?"

Unlike CQL Engine (truth evaluation) or Measure Engine (care gap detection), the Medication Advisor Engine:
- **Recommends** specific medications with dosages
- **Validates** against contraindications and interactions
- **Commits** prescriptions with full audit trail
- **Explains** why medications were chosen or excluded

## Industry Standards Compliance

### HL7 CDS Hooks
| Requirement | Implementation |
|-------------|----------------|
| Service Discovery | `GET /cds-services` |
| Hook Invocation | `POST /cds-services/medication-advisor` |
| Card Response | Ranked proposals with links |
| Feedback Loop | Evidence Envelope capture |

### FDA SaMD Class IIa
| Requirement | Implementation |
|-------------|----------------|
| Traceability | InferenceChain + SHA256 checksum |
| Reproducibility | Immutable snapshots with 30-min TTL |
| Audit Trail | EvidenceEnvelope finalization |
| Risk Management | Hard/Soft conflict classification |

### ISO 13485
| Requirement | Implementation |
|-------------|----------------|
| Design Controls | 4-phase workflow |
| Risk Analysis | Safety Gateway integration |
| Validation | Comprehensive test suite |
| Post-Market Surveillance | Override capture for learning |

## Architecture

### Calculate → Validate → Commit Workflow

```
┌────────────────────────────────────────────────────────────────────┐
│                     CALCULATE PHASE                                 │
│                                                                    │
│  Input: PatientContext + ClinicalQuestion                          │
│                                                                    │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐         │
│  │ Recipe       │ →  │ KB           │ →  │ Proposal     │         │
│  │ Resolution   │    │ Orchestration│    │ Generation   │         │
│  │ (Phase 1)    │    │ (Phase 2-3)  │    │ (Phase 4)    │         │
│  └──────────────┘    └──────────────┘    └──────────────┘         │
│                                                                    │
│  Output: Ranked MedicationProposal[] + SnapshotID                  │
└────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────────┐
│                     VALIDATE PHASE                                  │
│                                                                    │
│  Input: SnapshotID + Selected ProposalID                           │
│                                                                    │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐         │
│  │ Snapshot     │ →  │ Conflict     │ →  │ Recommendation│         │
│  │ Retrieval    │    │ Detection    │    │ Generation   │         │
│  └──────────────┘    └──────────────┘    └──────────────┘         │
│                                                                    │
│  Output: ValidationResult (proceed | warn | abort)                 │
└────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────────┐
│                      COMMIT PHASE                                   │
│                                                                    │
│  Input: SnapshotID + ProposalID + ProviderOverrides                │
│                                                                    │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐         │
│  │ Final        │ →  │ FHIR         │ →  │ Evidence     │         │
│  │ Validation   │    │ Generation   │    │ Finalization │         │
│  └──────────────┘    └──────────────┘    └──────────────┘         │
│                                                                    │
│  Output: FHIR MedicationRequest + EvidenceEnvelope                 │
└────────────────────────────────────────────────────────────────────┘
```

### Knowledge Base Integration

| KB Service | Port | Purpose | Phase |
|------------|------|---------|-------|
| KB-1 Dosing | 8081 | Dosage calculations | Phase 3 |
| KB-2 Interactions | 8089 | Drug-drug interactions | Phase 2 |
| KB-3 Guidelines | 8087 | Clinical guidelines | Phase 2 |
| KB-4 Safety | 8088 | Patient safety rules | Phase 2 |
| KB-5 Monitoring | TBD | Lab monitoring requirements | Phase 3 |
| KB-6 Efficacy | TBD | Drug efficacy data | Phase 4 |

### Weighted Scoring (QualityFactor)

| Factor | Weight | Description |
|--------|--------|-------------|
| Guideline | 30% | Alignment with clinical guidelines |
| Safety | 25% | Contraindication/interaction score |
| Efficacy | 20% | Expected treatment effectiveness |
| Interaction | 15% | Drug-drug interaction severity |
| Monitoring | 10% | Required lab monitoring burden |

## API Reference

### Calculate Endpoint
```http
POST /api/v1/advisor/calculate
Content-Type: application/json

{
  "patientId": "patient-123",
  "clinicalQuestion": {
    "text": "Add SGLT2i for T2DM with CKD",
    "intent": "ADD_MEDICATION",
    "targetDrugClass": "SGLT2i"
  },
  "patientContext": {
    "age": 72,
    "sex": "male",
    "conditions": [{"code": "44054006", "display": "Type 2 Diabetes"}],
    "computedScores": {"egfr": 42.5, "ckdStage": "G3b"}
  }
}
```

**Response**:
```json
{
  "snapshotId": "snap-abc123",
  "evidenceEnvelopeId": "env-xyz789",
  "proposals": [
    {
      "rank": 1,
      "medication": {"code": "1545149", "display": "Dapagliflozin 10mg"},
      "dosage": "10mg once daily",
      "qualityScore": 0.89,
      "qualityFactors": {"guideline": 0.95, "safety": 0.85, "efficacy": 0.90}
    }
  ]
}
```

### Validate Endpoint
```http
POST /api/v1/advisor/validate
Content-Type: application/json

{
  "snapshotId": "snap-abc123",
  "proposalId": "prop-001"
}
```

**Response**:
```json
{
  "valid": true,
  "conflicts": [],
  "recommendation": "proceed",
  "hardConflicts": 0,
  "softConflicts": 0
}
```

### Commit Endpoint
```http
POST /api/v1/advisor/commit
Content-Type: application/json

{
  "snapshotId": "snap-abc123",
  "proposalId": "prop-001",
  "providerId": "dr-smith",
  "overrides": []
}
```

**Response**:
```json
{
  "medicationRequestId": "MedicationRequest/mr-12345",
  "evidenceEnvelopeFinalized": true,
  "auditRecordId": "audit-67890"
}
```

### Explain Endpoint
```http
POST /api/v1/advisor/explain
Content-Type: application/json

{
  "envelopeId": "env-xyz789",
  "question": "whyExcluded",
  "medicationCode": "197361"
}
```

**Response**:
```json
{
  "question": "whyExcluded",
  "answer": "Metformin excluded due to eGFR < 30 (patient eGFR: 42.5)",
  "inferenceChain": [
    {"step": 1, "source": "KB-4", "rule": "metformin-renal-contraindication"}
  ],
  "confidenceScore": 0.95
}
```

### CDS Hooks Endpoints
```http
GET /cds-services
POST /cds-services/medication-advisor
```

## Configuration

### Environment Variables
```bash
# Server
PORT=8095
ENVIRONMENT=production

# Knowledge Bases
KB1_DOSING_URL=http://kb-drug-rules:8081
KB2_INTERACTIONS_URL=http://kb-drug-interactions:8089
KB3_GUIDELINES_URL=http://kb-guidelines:8087
KB4_SAFETY_URL=http://kb-patient-safety:8088

# Snapshot
SNAPSHOT_TTL_MINUTES=30
SNAPSHOT_STORAGE=redis

# Evidence
EVIDENCE_CHECKSUM_ALGORITHM=sha256
```

## Quick Start

### Build
```bash
cd vaidshala/clinical-runtime-platform/services/medication-advisor-engine
go build -o medication-advisor ./cmd/server
```

### Run
```bash
./medication-advisor
# Server starts on :8095
```

### Test
```bash
go test ./...
```

### Docker
```bash
docker build -t medication-advisor-engine .
docker run -p 8095:8095 medication-advisor-engine
```

## Project Structure

```
medication-advisor-engine/
├── README.md            # This documentation
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── Makefile             # Build commands
├── Dockerfile           # Container build
│
├── advisor/             # Main engine logic
│   ├── engine.go        # MedicationAdvisorEngine struct (NEW)
│   ├── workflow.go      # 4-phase workflow orchestration (COPIED)
│   ├── scoring.go       # QualityFactor weighted scoring (COPIED)
│   ├── conflicts.go     # Hard/Soft conflict classification (NEW)
│   └── explain.go       # Explain API implementation (NEW)
│
├── snapshot/            # Snapshot management
│   ├── types.go         # SnapshotType constants (COPIED)
│   ├── models.go        # Snapshot data models (COPIED)
│   └── manager.go       # Snapshot lifecycle (COPIED)
│
├── evidence/            # Audit trail
│   ├── envelope.go      # EvidenceEnvelopeManager (COPIED)
│   └── inference_chain.go # InferenceChain tracking (NEW)
│
├── recipe/              # Recipe resolution
│   └── resolver.go      # Declarative data requirements (COPIED)
│
├── kbclients/           # Knowledge Base clients
│   └── clients.go       # KB-1 through KB-6 integration (COPIED)
│
├── fhir/                # FHIR output
│   └── medication_request.go # FHIR R4 MedicationRequest (NEW)
│
├── cmd/server/          # HTTP server
│   ├── main.go          # Entry point (NEW)
│   └── cds_hooks.go     # HL7 CDS Hooks handlers (NEW)
│
└── test/                # Tests
    └── engine_test.go   # Comprehensive test suite (NEW)
```

## Code Provenance

This engine is built from existing proven Go code from `medication-service-v2`:

| Component | Source File | Lines | Status |
|-----------|-------------|-------|--------|
| Snapshot Types | `snapshot.go` | 602 | COPIED |
| Snapshot Models | `snapshot_models.go` | 287 | COPIED |
| Snapshot Manager | `snapshot_orchestrator.go` | 595 | COPIED |
| Evidence Envelope | `evidence_envelope.go` | 376 | COPIED |
| Recipe Resolver | `recipe_resolver_service.go` | 592 | COPIED |
| KB Clients | `knowledge_base_integration_service.go` | 752 | COPIED |
| Workflow | `workflow_orchestrator_service.go` | 973 | COPIED |
| Scoring | `proposal_generation_service.go` | 800 | COPIED |
| **TOTAL COPIED** | | **~4,977** | 88% |
| Main Engine | `advisor/engine.go` | ~300 | NEW |
| Explain API | `advisor/explain.go` | ~200 | NEW |
| Conflicts | `advisor/conflicts.go` | ~100 | NEW |
| InferenceChain | `evidence/inference_chain.go` | ~200 | NEW |
| FHIR Output | `fhir/medication_request.go` | ~150 | NEW |
| CDS Hooks | `cmd/server/cds_hooks.go` | ~150 | NEW |
| Tests | `test/engine_test.go` | ~500 | NEW |
| **TOTAL NEW** | | **~1,600** | 12% |

## Conflict Classification

### Hard Conflicts (Abort)
- **Labs**: Critical lab values changed since snapshot
- **Conditions**: New contraindicated conditions diagnosed
- **Allergies**: New allergy reported to proposed drug class

### Soft Conflicts (Warn)
- **Demographics**: Weight/age updated (may affect dosing)
- **Medications**: Non-interacting medication added
- **Minor labs**: Non-critical lab value changes

## Dependencies

- Go 1.21+
- Redis (for snapshot storage)
- KB Services (KB-1 through KB-6)

## Related Services

| Service | Relationship |
|---------|--------------|
| CQL Engine | Consumes clinical facts |
| Measure Engine | Identifies care gaps triggering recommendations |
| Evidence Envelope Service | Stores finalized audit trails |
| FHIR Server | Stores MedicationRequest resources |

## Standalone Architecture

This engine is **self-contained** with zero external Go/Python service dependencies at runtime:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│            MEDICATION-ADVISOR-ENGINE (Standalone Go Service)                │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │  advisor/engine.go - Main Orchestrator                                │  │
│  │                                                                       │  │
│  │  Calculate() → Validate() → Commit() → Explain()                      │  │
│  │      │             │            │           │                         │  │
│  │      ▼             ▼            ▼           ▼                         │  │
│  │  ┌─────────────────────────────────────────────────────────────┐     │  │
│  │  │  SELF-CONTAINED COMPONENTS (No External Go/Python deps)     │     │  │
│  │  │                                                             │     │  │
│  │  │  snapshot/     → types.go, models.go, manager.go            │     │  │
│  │  │  evidence/     → envelope.go, inference_chain.go            │     │  │
│  │  │  recipe/       → resolver.go                                │     │  │
│  │  │  kbclients/    → clients.go (HTTP calls to KB services)     │     │  │
│  │  │  advisor/      → workflow.go, scoring.go, conflicts.go      │     │  │
│  │  │  fhir/         → medication_request.go                      │     │  │
│  │  └─────────────────────────────────────────────────────────────┘     │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │  cmd/server/main.go - HTTP Server + CDS Hooks                         │  │
│  │                                                                       │  │
│  │  POST /api/v1/advisor/calculate   ─┐                                  │  │
│  │  POST /api/v1/advisor/validate    ─┼─► REST API                       │  │
│  │  POST /api/v1/advisor/commit      ─┤                                  │  │
│  │  POST /api/v1/advisor/explain     ─┘                                  │  │
│  │  GET  /cds-services               ─── HL7 CDS Hooks Discovery         │  │
│  │  POST /cds-services/medication-advisor ─ Hook Invocation              │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

## License

Proprietary - CardioFit Clinical Synthesis Hub
