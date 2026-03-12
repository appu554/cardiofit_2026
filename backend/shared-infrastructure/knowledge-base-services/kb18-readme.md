# KB-18 Governance Engine

**Clinical Governance Enforcement Platform**

A deterministic, auditable, legally-defensible governance engine that answers four critical questions for every clinical decision:

1. **What dose SHOULD this patient be receiving?**
2. **Is the prescribed dose SAFE?**
3. **Is the institution following program rules?**
4. **Who is accountable if the patient is harmed?**

## Overview

KB-18 provides real-time clinical governance enforcement with:

- **Reproducible decisions** - Same input always produces same output
- **Transparent logic** - Every decision fully explainable
- **Evidence trails** - Immutable audit records with cryptographic hashes
- **Risk classification** - Severity from INFO to FATAL
- **Deterministic behavior** - No ambiguity in enforcement
- **Structured artifacts** - Machine-readable, court-ready documentation

## Enforcement Levels

| Level | Behavior | Override |
|-------|----------|----------|
| `IGNORE` | Log only, no action | N/A |
| `NOTIFY` | Notify but allow | N/A |
| `WARN_ACKNOWLEDGE` | Warn, require acknowledgment | Yes |
| `HARD_BLOCK` | Block, no override possible | No |
| `HARD_BLOCK_WITH_OVERRIDE` | Block, governance can override | Yes |
| `MANDATORY_ESCALATION` | Block + immediate escalation | No |

## Pre-Configured Programs

### Maternal Safety
| Program | Description | Key Rules |
|---------|-------------|-----------|
| `MATERNAL_MEDICATION` | Medication safety in pregnancy | Teratogenic blocks, Category X enforcement |
| `PREECLAMPSIA_PROTOCOL` | Severe preeclampsia management | Magnesium requirement, BP targets |
| `MAGNESIUM_PROTOCOL` | Magnesium sulfate safety | Toxicity monitoring, renal adjustment |
| `GESTATIONAL_DM` | Gestational diabetes | Insulin dosing, glucose targets |

### Opioid Stewardship
| Program | Description | Key Rules |
|---------|-------------|-----------|
| `OPIOID_STEWARDSHIP` | Overall opioid safety | MME limits, PDMP, interactions |
| `OPIOID_NAIVE` | Naive patient safety | ER opioid block, starting dose limits |
| `OPIOID_MAT` | Medication-assisted treatment | Buprenorphine, methadone protocols |

### Anticoagulation
| Program | Description | Key Rules |
|---------|-------------|-----------|
| `ANTICOAGULATION` | General anticoagulant safety | Duplication, renal dosing |
| `WARFARIN_MANAGEMENT` | Warfarin-specific | INR monitoring, interactions |
| `DOAC_MANAGEMENT` | DOAC-specific | Renal adjustment, reversal |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    GOVERNANCE REQUEST                        │
│  (Medication Order / Protocol Check / Compliance Audit)     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   PROGRAM MATCHER                            │
│  • Registry membership (from KB-17)                          │
│  • Active diagnoses                                          │
│  • Current medications                                       │
│  • Demographics (age, sex, pregnancy)                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    RULE ENGINE                               │
│  • Condition evaluation (AND/OR logic)                       │
│  • Multi-type conditions (lab, vital, diagnosis, etc.)      │
│  • Priority-based rule ordering                              │
│  • Severity + enforcement determination                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  GOVERNANCE RESPONSE                         │
│  ┌─────────────┐ ┌──────────────┐ ┌────────────────────┐   │
│  │ Violations  │ │Recommendations│ │   Evidence Trail   │   │
│  │ + Severity  │ │ + Dose Adj   │ │   (Immutable)      │   │
│  └─────────────┘ └──────────────┘ └────────────────────┘   │
│  ┌─────────────┐ ┌──────────────┐ ┌────────────────────┐   │
│  │  Outcome    │ │   Required   │ │   Accountable      │   │
│  │  Decision   │ │   Actions    │ │   Parties          │   │
│  └─────────────┘ └──────────────┘ └────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Decision Flow Example

```
1. Medication Order: Methotrexate 15mg weekly
   Patient: 28yo pregnant female, 12 weeks gestation

2. Program Match:
   ✓ MATERNAL_MEDICATION (pregnancy + medication)

3. Rule Evaluation:
   Rule MAT-001: Teratogenic Medication Block
   - Condition: drugClass IN [METHOTREXATE, ...]  ✓ MET
   - Condition: isPregnant = true                  ✓ MET
   - All conditions met = VIOLATION

4. Outcome:
   - Enforcement: HARD_BLOCK
   - Severity: FATAL
   - Override: NOT ALLOWED
   - Action: Block order, escalate to pharmacy supervisor

5. Evidence Trail:
   - Patient snapshot captured
   - Rule evaluation recorded
   - Decision hash generated
   - Immutable record stored
```

## API Endpoints

### Core Evaluation
```
POST /api/v1/evaluate              - General governance evaluation
POST /api/v1/evaluate/medication   - Medication order evaluation
POST /api/v1/evaluate/protocol     - Protocol compliance check
```

### Programs
```
GET  /api/v1/programs              - List all programs
GET  /api/v1/programs/{code}       - Get program details
```

### Overrides
```
GET  /api/v1/overrides             - List overrides
POST /api/v1/overrides/request     - Request override
POST /api/v1/overrides/approve     - Approve override
POST /api/v1/overrides/deny        - Deny override
```

### Acknowledgments
```
GET  /api/v1/acknowledgments       - List acknowledgments
POST /api/v1/acknowledge           - Record acknowledgment
```

### Escalations
```
GET  /api/v1/escalations           - List escalations
POST /api/v1/escalations/resolve   - Resolve escalation
```

### Analytics
```
GET  /api/v1/stats                 - Engine statistics
GET  /api/v1/audit/pattern         - Override pattern analysis
```

## Request/Response Examples

### Medication Evaluation Request
```json
{
  "patientId": "P001",
  "patientContext": {
    "patientId": "P001",
    "age": 28,
    "sex": "F",
    "isPregnant": true,
    "gestationalAge": 12,
    "registryMemberships": [
      {"registryCode": "PREGNANCY", "status": "ACTIVE"}
    ]
  },
  "order": {
    "medicationCode": "MTX",
    "medicationName": "Methotrexate",
    "drugClass": "METHOTREXATE",
    "dose": 15,
    "doseUnit": "mg",
    "frequency": "weekly",
    "route": "PO"
  },
  "requestorId": "DR001",
  "requestorRole": "PHYSICIAN",
  "facilityId": "HOSP001"
}
```

### Blocked Response
```json
{
  "requestId": "med-eval-20231215...",
  "outcome": "BLOCKED",
  "isApproved": false,
  "hasViolations": true,
  "highestSeverity": "FATAL",
  "violations": [
    {
      "id": "viol-...",
      "ruleId": "MAT-001",
      "ruleName": "Teratogenic Medication Block",
      "category": "CONTRAINDICATION",
      "severity": "FATAL",
      "enforcementLevel": "HARD_BLOCK",
      "description": "Methotrexate is Category X and absolutely contraindicated in pregnancy",
      "clinicalRisk": "Teratogenic effects with proven fetal harm",
      "canOverride": false
    }
  ],
  "recommendations": [
    {
      "type": "alternative",
      "title": "Consider Alternative",
      "description": "Consult rheumatology for pregnancy-safe alternatives"
    }
  ],
  "accountableParties": [
    {"role": "PRESCRIBER", "accountability": "Order verification"},
    {"role": "PHARMACIST", "accountability": "Dispensing safety check"}
  ],
  "nextSteps": [
    "Order cannot proceed - modify order or select alternative"
  ],
  "evidenceTrail": {
    "trailId": "trail-...",
    "hash": "abc123...",
    "isImmutable": true
  }
}
```

### Override Request
```json
{
  "violationId": "viol-...",
  "requestId": "REQ001",
  "userId": "DR001",
  "userRole": "ONCOLOGIST",
  "userName": "Dr. Smith",
  "reason": "Terminal cancer, comfort care",
  "clinicalJustification": "High-dose opioids required for end-of-life pain control",
  "riskAccepted": true
}
```

## Integration Points

### Upstream (Consumes From)
- **KB-17 Population Registry**: Patient registry memberships for program eligibility
- **KB-2 Patient Context**: Demographics, diagnoses, medications, labs
- **KB-8 Risk Scores**: Risk stratification for high-risk patient identification
- **EHR/CPOE**: Medication orders for real-time evaluation

### Downstream (Produces To)
- **KB-14 Care Navigator**: Tasks for acknowledgment, escalation follow-up
- **KB-15 Patient Engagement**: Notifications for patient safety alerts
- **Audit Systems**: Immutable evidence trails for compliance/legal
- **Analytics**: Override patterns, violation trends, compliance metrics

## Evidence Trail Structure

Every governance decision produces an immutable evidence trail:

```json
{
  "trailId": "trail-20231215...",
  "timestamp": "2023-12-15T10:30:00Z",
  "patientSnapshot": { ... },
  "programsEvaluated": ["MATERNAL_MEDICATION"],
  "rulesApplied": [
    {
      "ruleId": "MAT-001",
      "ruleName": "Teratogenic Medication Block",
      "wasEvaluated": true,
      "wasTriggered": true,
      "inputData": {...},
      "outputDecision": "violation=true"
    }
  ],
  "finalDecision": "BLOCKED",
  "decisionRationale": "Blocked due to 1 violation(s). Highest severity: FATAL",
  "requestedBy": "DR001",
  "evaluatedBy": "KB18-GOV-ENGINE-v1.0",
  "hash": "sha256:abc123...",
  "isImmutable": true
}
```

## Running the Service

### Local Development
```bash
cd kb18-governance-engine
go run cmd/server/main.go
```

### Docker
```bash
docker build -t kb18-governance-engine .
docker run -p 8018:8018 kb18-governance-engine
```

### Environment Variables
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8018` |
| `LOG_LEVEL` | Logging level | `INFO` |
| `AUDIT_ENABLED` | Enable audit logging | `true` |

## Testing

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package tests
go test ./pkg/engine/... -v
go test ./pkg/override/... -v
```

## File Structure

```
kb18-governance-engine/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── pkg/
│   ├── types/
│   │   └── types.go             # Type definitions
│   ├── programs/
│   │   └── definitions.go       # Governance programs
│   ├── engine/
│   │   ├── governance_engine.go # Core evaluation logic
│   │   └── governance_engine_test.go
│   ├── override/
│   │   ├── override_store.go    # Override/acknowledgment handling
│   │   └── override_store_test.go
│   └── server/
│       └── server.go            # HTTP API
├── go.mod
└── README.md
```

## Accountability Chain

Each program defines who is accountable at each escalation level:

```
Level 1: PRESCRIBER (Ordering Physician)
    ↓ escalate if unresolved
Level 2: PHARMACIST (Clinical Pharmacist)
    ↓ escalate if unresolved
Level 3: ATTENDING_PHYSICIAN / SPECIALIST
    ↓ escalate if unresolved
Level 4: DEPARTMENT_CHIEF / MEDICAL_DIRECTOR
    ↓ escalate if unresolved
Level 5: CHIEF_MEDICAL_OFFICER
```

## Override Pattern Monitoring

The system automatically monitors for suspicious override patterns:

- **>5 overrides in 24 hours**: Flag for review
- **>20 overrides in 7 days**: Investigation trigger
- **High denial rate**: Pattern analysis
- **Single-user high frequency**: Audit trigger

```json
{
  "userId": "DR001",
  "last24Hours": 8,
  "last7Days": 25,
  "approvedCount": 20,
  "deniedCount": 5,
  "flaggedForReview": true,
  "flagReason": "Excessive overrides in 24 hours"
}
```

## Clinical Rationale

Every rule includes clinical rationale and evidence:

- **Evidence Level**: A (Strong), B (Moderate), C (Weak), D (Limited), Expert
- **References**: Guidelines, studies, FDA labels
- **Clinical Rationale**: Plain-language explanation of risk

## Compliance & Legal

KB-18 is designed to withstand regulatory scrutiny:

- ✅ Complete audit trails for every decision
- ✅ Immutable evidence with cryptographic hashes
- ✅ Clear accountability chains
- ✅ Override justification requirements
- ✅ Pattern monitoring for abuse detection
- ✅ Structured artifacts for legal discovery

## Roadmap

- [ ] Database persistence for evidence trails
- [ ] Real-time Kafka event consumption
- [ ] Multi-facility governance policies
- [ ] Custom program builder UI
- [ ] Integration with KB-14 task creation
- [ ] National Command Center dashboard
- [ ] TGA/CDSCO certification preparation
