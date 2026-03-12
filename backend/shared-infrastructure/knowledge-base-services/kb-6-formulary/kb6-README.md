# KB-6: Formulary & Coverage Service

**Drug Formulary Management, Coverage Determination, and Prior Authorization**

## Why Formulary Service is Critical

| Consumer | Need |
|----------|------|
| **Medication Advisor** | Real-time formulary status during prescribing |
| **CPOE** | Coverage checks before order submission |
| **Pharmacy** | Generic substitution, quantity limits |
| **Revenue Cycle** | Prior auth status, coverage determination |
| **Patient Portal** | Cost estimates, alternatives |

## Features

### Tier Management

| Tier | Name | Typical Copay | Description |
|------|------|---------------|-------------|
| **Tier 0** | Preventive | $0 | Preventive medications |
| **Tier 1** | Generic | $5-15 | Generic medications |
| **Tier 2** | Preferred Brand | $35-50 | Preferred brand medications |
| **Tier 3** | Non-Preferred | $75-100 | Non-preferred brands |
| **Tier 4** | Specialty | 25% coinsurance | Specialty medications |

### Prior Authorization

- **Clinical Criteria Evaluation**: Diagnosis, labs, prior therapy
- **Step Therapy Integration**: Required drug trials
- **Electronic PA (ePA)**: CoverMyMeds integration ready
- **Approval Tracking**: Duration, renewal, expiration

### Step Therapy

- **Multi-Step Requirements**: Sequential drug trials
- **Override Criteria**: Medical necessity, contraindications
- **Prior Therapy Verification**: Claims history integration

### Generic Substitution

- **AB-Rated Equivalents**: FDA Orange Book ratings
- **Cost Savings Calculation**: Brand vs generic comparison
- **DAW Code Support**: Dispense As Written handling

### Quantity Limits

- **Per-Fill Limits**: Maximum quantity per dispensing
- **Days Supply Limits**: Maximum days supply
- **Daily Dose Limits**: Max quantity per day
- **Override Support**: Clinical justification

## API Endpoints

### Formulary Lookup
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/lookup?rxnorm=` | Lookup drug by RxNorm |
| POST | `/api/v1/lookup/batch` | Batch lookup |
| GET | `/api/v1/search?q=` | Search by name/class |

### Coverage Determination
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/coverage/check` | Full coverage check |
| POST | `/api/v1/coverage/benefits` | Real-time benefits |

### Tiers & Copays
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/tiers` | List all tiers |
| GET | `/api/v1/tiers/copay?rxnorm=` | Estimate copay |

### Prior Authorization
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/pa/requirements?rxnorm=` | Get PA requirements |
| GET | `/api/v1/pa/check?rxnorm=` | Check if PA required |
| POST | `/api/v1/pa/submit` | Submit PA request |
| GET | `/api/v1/pa/status?id=` | Check PA status |

### Step Therapy
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/steptherapy/check` | Check ST requirements |
| GET | `/api/v1/steptherapy/requirements?rxnorm=` | Get ST rules |

### Quantity Limits
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/quantitylimit/check?rxnorm=` | Check quantity limit |

### Generic Substitution
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/generic/substitution?brandRxnorm=` | Get generic equivalent |
| GET | `/api/v1/generic/alternatives?rxnorm=` | Get all generics |

### Alternatives
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/alternatives?rxnorm=` | Get therapeutic alternatives |

## Built-in Drug Entries

### Diabetes Medications
| Drug | Tier | PA | ST | Notes |
|------|------|----|----|-------|
| Metformin | Generic | No | No | First-line therapy |
| Empagliflozin (Jardiance) | Preferred | No | Yes | Requires metformin first |
| Canagliflozin (Invokana) | Non-Preferred | Yes | Yes | PA if Jardiance tried |
| Semaglutide (Ozempic) | Specialty | Yes | Yes | HbA1c > 7%, prior trials |

### Cardiovascular
| Drug | Tier | PA | ST | Notes |
|------|------|----|----|-------|
| Lisinopril | Generic | No | No | First-line ACEi |
| Losartan | Generic | No | No | First-line ARB |
| Metoprolol ER | Generic | No | No | Beta blocker |
| Atorvastatin | Generic | No | No | First-line statin |
| Rosuvastatin | Preferred | No | No | Alternative statin |

### Anticoagulants
| Drug | Tier | PA | ST | Notes |
|------|------|----|----|-------|
| Warfarin | Generic | No | No | Vitamin K antagonist |
| Apixaban (Eliquis) | Preferred | No | No | Preferred DOAC |
| Rivaroxaban (Xarelto) | Non-Preferred | No | No | Non-preferred DOAC |

## PA Clinical Criteria Examples

### Semaglutide (Ozempic)
```json
{
  "criteria": [
    {"type": "DIAGNOSIS", "required": ["E11"], "description": "Type 2 Diabetes"},
    {"type": "LAB", "test": "HbA1c", "operator": ">", "value": 7.0},
    {"type": "PRIOR_THERAPY", "drugs": ["Metformin"], "duration": 90},
    {"type": "PRIOR_THERAPY", "drugs": ["SGLT2i"], "duration": 60}
  ],
  "approvalDuration": 365,
  "renewalAllowed": true
}
```

### Oxycodone
```json
{
  "criteria": [
    {"type": "DIAGNOSIS", "required": ["G89", "M54", "C00-C96"]},
    {"type": "PRIOR_THERAPY", "drugs": ["Acetaminophen", "NSAIDs"], "duration": 7},
    {"type": "AGE", "minAge": 18}
  ],
  "approvalDuration": 30,
  "requiredDocs": ["Pain assessment", "PDMP check", "Opioid agreement"]
}
```

## Step Therapy Examples

### GLP-1 Agonists (Ozempic)
```
Step 1: Metformin x 90 days
Step 2: SGLT2 inhibitor x 60 days
Then: GLP-1 agonist approved
```

### Non-Preferred DOAC (Xarelto)
```
Step 1: Apixaban (Eliquis) x 30 days
Then: Rivaroxaban approved
Override: Adverse reaction, drug interaction
```

## Quick Start

### Running with Docker

```bash
docker build -t kb6-formulary-service .
docker run -p 8085:8085 kb6-formulary-service
```

### Running Locally

```bash
go mod download
go run cmd/server/main.go
go test ./test/...
```

### Example: Coverage Check

```bash
curl -X POST http://localhost:8085/api/v1/coverage/check \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "planId": "plan-456",
    "rxnormCode": "1991302",
    "quantity": 4,
    "daysSupply": 28,
    "diagnoses": ["E11.9"],
    "patientAge": 55
  }'
```

### Example: PA Submission

```bash
curl -X POST http://localhost:8085/api/v1/pa/submit \
  -H "Content-Type: application/json" \
  -d '{
    "patientId": "patient-123",
    "providerId": "dr-456",
    "rxnormCode": "1991302",
    "quantity": 4,
    "daysSupply": 28,
    "diagnoses": ["E11.9"],
    "clinicalNotes": "HbA1c 8.2%, failed metformin and empagliflozin",
    "urgencyLevel": "STANDARD"
  }'
```

## File Structure

```
kb6-formulary-service/
├── cmd/server/main.go              (700 lines)   HTTP server with 25+ endpoints
├── pkg/formulary/
│   ├── service.go                  (900 lines)   Core service, coverage logic
│   ├── data.go                     (400 lines)   Built-in drug entries
│   └── requirements.go             (350 lines)   PA, ST, generic substitution
├── test/service_test.go            (300 lines)   Comprehensive tests
├── Dockerfile                      (50 lines)    Multi-stage build
├── go.mod                          (5 lines)
└── README.md                                     This file
```

## Integration Points

### Upstream
- **PBM Systems**: Real-time benefits, PA submission
- **Claims Systems**: Prior therapy verification
- **Pharmacy Networks**: Generic substitution rules

### Downstream
- **KB-1 Drug Dosing**: Formulary status for dosing
- **Medication Advisor**: Coverage in recommendations
- **CPOE (KB-12)**: Pre-order coverage checks
- **Patient Portal**: Cost transparency

## License

Proprietary - Healthcare Platform
