# KB-12: Order Sets & Care Plans Service

**Clinical Order Set Management with KB-3 Temporal Logic Integration**

## Overview

KB-12 provides comprehensive clinical order set and care plan management with deep integration to KB-3 Temporal Logic for time-bound protocol enforcement. This service enables:

- **Order Set Templates**: Reusable, evidence-based order sets for admissions, procedures, and protocols
- **Care Plan Management**: Longitudinal chronic disease management plans with goals and milestones
- **Temporal Orchestration**: Time-critical deadline enforcement (SEP-1, Stroke, STEMI)
- **Smart Defaults**: Patient-specific order customization (weight-based dosing, renal adjustments)
- **FHIR R4 Output**: Standards-compliant resource generation

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    KB-12 Order Sets & Care Plans Service                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │  Order Set      │  │   Care Plan     │  │    Protocol     │             │
│  │  Engine         │  │   Engine        │  │    Engine       │             │
│  │                 │  │                 │  │                 │             │
│  │ • Templates     │  │ • Goals         │  │ • Acute (SEP-1) │             │
│  │ • Activation    │  │ • Activities    │  │ • Chronic (DM)  │             │
│  │ • Sequencing    │  │ • Milestones    │  │ • Perioperative │             │
│  │ • Dependencies  │  │ • Progress      │  │ • Transitions   │             │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘             │
│           │                    │                    │                       │
│  ┌────────▼────────────────────▼────────────────────▼────────┐             │
│  │              Temporal Orchestration Layer                  │             │
│  │         (Deep Integration with KB-3 Temporal Logic)        │             │
│  │                                                            │             │
│  │  • Sequence Resolution    • Dependency Graph               │             │
│  │  • Timing Constraints     • State Machine Execution        │             │
│  │  • Recurrence Patterns    • Deadline Monitoring            │             │
│  └────────────────────────────────────────────────────────────┘             │
│                              │                                              │
│  ┌───────────────────────────▼───────────────────────────────┐             │
│  │                    Output Generation                       │             │
│  │  • FHIR MedicationRequest  • FHIR ServiceRequest          │             │
│  │  • FHIR CarePlan           • FHIR Task                    │             │
│  │  • CDS Hooks Cards         • Alert/Reminder               │             │
│  └───────────────────────────────────────────────────────────┘             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              ┌─────────┐    ┌─────────┐    ┌─────────┐
              │  KB-3   │    │  KB-6   │    │  KB-1   │
              │Temporal │    │Formulary│    │ Dosing  │
              │ Logic   │    │         │    │ Rules   │
              └─────────┘    └─────────┘    └─────────┘
```

## Order Set Categories

| Category | Count | Description |
|----------|-------|-------------|
| **ADMISSION** | 15 | Condition-specific admission orders (CHF, COPD, Pneumonia, DKA, Stroke, MI, Sepsis, etc.) |
| **PROCEDURE** | 10 | Pre/post procedure orders (Pre-op, Post-op, Colonoscopy, Cardiac Cath, etc.) |
| **ACUTE_PROTOCOL** | 8 | Time-critical bundles (Sepsis SEP-1, Stroke, STEMI, DKA) |
| **CHRONIC_CARE** | 12 | Longitudinal care plans (Diabetes, CKD, HTN, CHF, COPD, Anticoagulation) |
| **PERIOPERATIVE** | 6 | Surgical orders (Pre-op, Post-op, VTE prophylaxis) |
| **TRANSITION** | 5 | Transitions of care (Discharge, SNF, Home Health, Hospice, Rehab) |
| **EMERGENCY** | 4 | Emergency protocols (Code Blue, RRT, MTP, Malignant Hyperthermia) |

## Key Temporal Constraints (KB-3 Integration)

| Protocol | Constraint | Deadline | Clinical Significance |
|----------|------------|----------|----------------------|
| **Sepsis (SEP-1)** | Antibiotics | ≤1 hour | 7-8% mortality increase per hour delay |
| **Sepsis (SEP-1)** | Fluid resuscitation | ≤3 hours | 30 mL/kg if hypotensive/lactate ≥4 |
| **Sepsis (SEP-1)** | Repeat lactate | ≤6 hours | If initial lactate >2 mmol/L |
| **Stroke** | Door-to-CT | ≤25 min | Rapid imaging for tPA decision |
| **Stroke** | Door-to-Needle | ≤60 min | tPA administration |
| **Stroke** | tPA Window | ≤4.5 hours | From symptom onset |
| **STEMI** | Door-to-ECG | ≤10 min | Rapid diagnosis |
| **STEMI** | Door-to-Balloon | ≤90 min | PCI timeframe |
| **DKA** | K+ before insulin | Required | Fatal hypokalemia risk if K+ <3.3 |
| **DKA** | Insulin overlap | 2 hours | SC insulin before stopping IV |
| **Surgery** | Prophylactic antibiotic | ≤60 min | Before incision (SCIP) |
| **MTP** | TXA administration | ≤3 hours | From injury onset (CRASH-2) |

## API Endpoints

### Templates

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/templates` | List all templates (filter by category, specialty) |
| GET | `/api/v1/templates/{id}` | Get template details |
| GET | `/api/v1/templates/search?q=` | Search templates by keyword/condition |
| GET | `/api/v1/templates/categories` | List template categories |

### Order Set Activation

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/activate` | Activate order set for patient |
| GET | `/api/v1/instances` | List active order set instances |
| GET | `/api/v1/instances/{id}` | Get instance details |
| POST | `/api/v1/instances/update` | Update order status |

### Care Plans

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/careplans` | List patient care plans |
| GET | `/api/v1/careplans/{id}` | Get care plan details |
| POST | `/api/v1/careplans/create` | Create new care plan |
| POST | `/api/v1/careplans/progress` | Update activity progress |

### Temporal Constraints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/constraints` | Get constraints for instance |
| POST | `/api/v1/constraints/evaluate` | Evaluate constraint status |
| GET | `/api/v1/alerts` | Get active deadline alerts |

### FHIR Output

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/fhir/bundle` | Generate FHIR Bundle |
| GET | `/api/v1/fhir/plandefinition` | Get PlanDefinition resource |
| GET | `/api/v1/fhir/careplan` | Get FHIR CarePlan resource |

### Health & Metrics

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/health/live` | Liveness probe |
| GET | `/health/ready` | Readiness probe |
| GET | `/metrics` | Service metrics |

## Quick Start

### Running with Docker

```bash
# Build the image
docker build -t kb12-ordersets-service .

# Run the container
docker run -p 8086:8086 kb12-ordersets-service
```

### Running Locally

```bash
# Install dependencies
go mod download

# Run the service
go run cmd/server/main.go

# Run tests
go test ./test/...
```

### Example: Activate Sepsis Order Set

```bash
curl -X POST http://localhost:8086/api/v1/activate \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "os-adm-sepsis-001",
    "patientId": "patient-123",
    "encounterId": "encounter-456",
    "activatedBy": "dr-smith"
  }'
```

Response includes:
- Activated order set instance
- SEP-1 bundle temporal constraints
- Deadline alerts
- FHIR Bundle for integration

### Example: Get Temporal Constraints

```bash
curl http://localhost:8086/api/v1/constraints?instanceId=osi_abc12345
```

Response shows:
- 1-hour antibiotic deadline status
- 3-hour fluid resuscitation status
- 6-hour repeat lactate status
- Time remaining and alert levels

## Clinical Content

### Admission Order Sets (Detailed)

**CHF Exacerbation** (os-adm-chf-001)
- Admission/Monitoring: Telemetry, daily weight, I/O, continuous pulse ox
- Diet: 2g sodium, fluid restriction 1.5-2L
- Labs: BMP daily, BNP, troponin, LFTs, TSH
- Imaging: CXR, Echo if not recent
- Medications: IV diuretics, GDMT continuation, electrolyte replacement
- Temporal: Daily weight by 8am, BNP repeat day 3

**Sepsis/Septic Shock** (os-adm-sepsis-001)
- 1-Hour Bundle: Lactate, blood cultures, broad-spectrum antibiotics
- 3-Hour Bundle: 30 mL/kg crystalloid if hypotension/lactate ≥4
- 6-Hour Bundle: Repeat lactate, vasopressors if needed
- Antibiotic options: Pip-tazo, vancomycin, ceftriaxone, meropenem

**DKA** (os-adm-dka-001)
- Fluids: NS bolus, then maintenance with dextrose when BG <250
- Insulin: Regular insulin drip 0.1 units/kg/hr (hold if K+ <3.3)
- Labs: POC glucose q1h, BMP q2-4h, beta-hydroxybutyrate
- Potassium: Protocol-based replacement
- Transition: 2-hour overlap with SC insulin

### Chronic Care Plans (Detailed)

**Type 2 Diabetes** (cp-chronic-dm-001)
- Monitoring: HbA1c q3-6mo, annual UACR, annual eye/foot exam
- Medications: Metformin, SGLT2i if HF/CKD, GLP-1 RA if ASCVD/weight
- CV risk: Statin, BP control, aspirin consideration
- Education: DSMES, MNT referral
- Immunizations: Flu, pneumococcal, HepB, COVID

**CKD** (cp-chronic-ckd-001)
- Monitoring by stage: G3a annual, G3b q6mo, G4-5 q3mo
- Medications: ACEi/ARB, SGLT2i, finerenone
- CKD-MBD: Ca/Phos/PTH monitoring, vitamin D, phosphate binders
- Anemia: Iron studies, ESA if indicated
- Modality planning: Nephrology referral, transplant eval, access planning

## Temporal Logic Features

### Constraint Types

1. **DEADLINE**: Must complete by specific time (e.g., antibiotics within 1 hour)
2. **INTERVAL**: Recurring at regular intervals (e.g., BMP q4h)
3. **SEQUENCE**: Must occur in specific order (e.g., cultures before antibiotics)

### Constraint Status Tracking

- **PENDING**: Not yet due
- **APPROACHING**: Within alert threshold
- **DUE**: At deadline
- **OVERDUE**: Past deadline
- **MET**: Successfully completed
- **MISSED**: Deadline passed without completion
- **WAIVED**: Clinician override with documentation

### Alert Thresholds

Configurable alerts at multiple time points before deadline:
- INFO: 30 minutes before
- WARNING: 15 minutes before
- CRITICAL: At deadline or overdue

## Integration Points

### Upstream (Consumers)

- **EHR/CPOE Systems**: Order activation and submission
- **CDS Hooks**: Real-time decision support
- **Population Health**: Care gap identification

### Downstream (Dependencies)

- **KB-3 Temporal Logic**: Constraint definitions and evaluation
- **KB-1 Drug Dosing**: Weight-based and renal dose calculations
- **KB-6 Formulary**: Medication availability and alternatives
- **KB-7 Terminology**: Code mapping and validation

## CPOE/EHR Integration

KB-12 provides comprehensive integration with CPOE systems and EHR workflows.

### CPOE Integration Features

| Feature | Description |
|---------|-------------|
| **Order Submission** | Submit orders to CPOE with validation and safety checks |
| **Draft Sessions** | Support for order drafting before submission |
| **Co-Signature Workflow** | Nurse/resident order co-signature management |
| **Safety Alerts** | Allergy, interaction, duplicate, and dose range checks |
| **Override Management** | Track and audit alert overrides with reasons |
| **HL7v2 Output** | Generate ORM^O01 order messages |
| **FHIR R4 Output** | Generate MedicationRequest, ServiceRequest resources |

### EHR Workflow Integration

| Feature | Description |
|---------|-------------|
| **Workflow Events** | Publish/subscribe to clinical workflow events |
| **Worklists** | Pending orders, co-signs, deadlines, critical alerts |
| **Task Management** | Create, assign, complete, escalate clinical tasks |
| **Notifications** | Multi-channel notifications (in-app, pager, SMS, email) |
| **Real-time Sync** | Bidirectional order status synchronization |

### CDS Hooks Integration

Implements HL7 CDS Hooks specification for clinical decision support:

| Hook | Service ID | Description |
|------|------------|-------------|
| `order-select` | kb12-order-select | Suggest order sets based on selections |
| `order-sign` | kb12-order-sign | Validate orders against temporal constraints |
| `patient-view` | kb12-patient-view | Display care plan tasks and deadlines |
| `encounter-start` | kb12-encounter-start | Suggest admission order sets |
| `encounter-discharge` | kb12-encounter-discharge | Check discharge readiness |

### Safety Check Types

| Alert Type | Severity | Description |
|------------|----------|-------------|
| ALLERGY | CRITICAL | Drug allergy/sensitivity detected |
| DRUG_INTERACTION | VARIES | Drug-drug interaction identified |
| DUPLICATE | WARNING | Duplicate therapy detected |
| DOSE_RANGE | WARNING/CRITICAL | Dose outside recommended range |
| RENAL_DOSE | WARNING | Dose adjustment needed for renal function |
| CONTRAINDICATION | CRITICAL | Absolute contraindication |
| FORMULARY | INFO | Non-formulary medication |
| MAX_DOSE | CRITICAL | Maximum dose exceeded |
| HIGH_ALERT | WARNING | High-alert medication |

## File Structure

```
kb12-ordersets-careplans/
├── cmd/server/main.go                    (1,085 lines)  HTTP server with 25+ endpoints
├── pkg/ordersets/
│   ├── service.go                        (1,682 lines)  Core service & types
│   ├── admission_ordersets.go            (1,733 lines)  15 admission order sets
│   ├── chronic_careplans.go              (1,041 lines)  12 chronic care plans
│   ├── procedure_ordersets.go            (1,055 lines)  10 procedure order sets
│   └── template_loader.go                (537 lines)    Template loading & emergency protocols
├── pkg/cpoe/
│   ├── cpoe_integration.go               (1,205 lines)  CPOE order submission & management
│   ├── integration.go                    (1,277 lines)  EHR integration layer
│   └── handlers.go                       (730 lines)    HTTP handlers for CPOE endpoints
├── pkg/workflow/
│   └── ehr_workflow.go                   (984 lines)    EHR workflow & task management
├── pkg/cdshooks/
│   └── cds_hooks.go                      (908 lines)    CDS Hooks implementation
├── test/service_test.go                  (608 lines)    Comprehensive tests
├── Dockerfile                            (74 lines)     Multi-stage build
├── go.mod                                (5 lines)
└── README.md                                            This file
```

**Total: ~12,900 lines Go code**

## Evidence Base

This service implements guidelines from:

- **Sepsis**: Surviving Sepsis Campaign 2021, CMS SEP-1 Measure
- **Heart Failure**: 2022 AHA/ACC/HFSA HF Guideline
- **Diabetes**: ADA Standards of Care 2024
- **CKD**: KDIGO 2024 CKD Guideline
- **Stroke**: AHA/ASA Stroke Guidelines
- **STEMI**: ACC/AHA STEMI Guideline
- **Surgery**: SCIP Guidelines, ERAS Protocols
- **Anticoagulation**: CHEST Guidelines, ACC/AHA AFib Guideline

## License

Proprietary - Healthcare Platform

## Version History

- **1.0.0** (2024-01): Initial release with 45 order sets and care plans
