# KB-7 Terminology Service - Quick Reference

## Service URLs

| Environment | Base URL |
|-------------|----------|
| Local | `http://localhost:8087` |
| Docker | `http://kb7-terminology:8087` |

---

## Essential Commands

### 1. First-Time Setup
```bash
# Start the service
cd kb-7-terminology
go build -o kb7-server ./cmd/server && ./kb7-server

# Seed value sets (run once)
curl -X POST http://localhost:8087/v1/rules/seed
```

### 2. Health Check
```bash
curl http://localhost:8087/health | jq '.status'
# Expected: "healthy"
```

---

## Most Used APIs

### Validate a Code in a Value Set (THREE-CHECK PIPELINE)
```bash
curl -X POST http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/validate \
  -H "Content-Type: application/json" \
  -d '{"code": "448417001", "system": "http://snomed.info/sct"}'
```

**Response Fields:**
| Field | Description |
|-------|-------------|
| `valid` | true/false - Is code valid? |
| `match_type` | `exact`, `subsumption`, or `none` |
| `matched_code` | The code that matched (for subsumption) |
| `pipeline` | Detailed audit trail of all 3 steps |

---

### Find All Value Sets for a Code (Reverse Lookup)
```bash
curl -X POST http://localhost:8087/v1/rules/classify \
  -H "Content-Type: application/json" \
  -d '{"code": "91302008", "system": "http://snomed.info/sct"}'
```

---

### Test IS-A Relationship (Subsumption)
```bash
curl -X POST http://localhost:8087/v1/subsumption/test \
  -H "Content-Type: application/json" \
  -d '{
    "code_a": "448417001",
    "code_b": "91302008",
    "system": "http://snomed.info/sct"
  }'
```

**Result:** `subsumes: true` means code_a IS-A code_b

---

### List All Value Sets
```bash
curl "http://localhost:8087/v1/rules/valuesets?limit=50"
```

---

### Expand a Value Set
```bash
curl -X POST http://localhost:8087/v1/rules/valuesets/SepsisDiagnosis/expand \
  -H "Content-Type: application/json" -d '{}'
```

---

### Get Ancestors of a Concept
```bash
curl -X POST http://localhost:8087/v1/subsumption/ancestors \
  -H "Content-Type: application/json" \
  -d '{"code": "448417001", "system": "http://snomed.info/sct", "max_depth": 5}'
```

---

## CDSS (Clinical Decision Support) APIs

The CDSS pipeline enables patient-level evaluation: **FHIR Resources → Facts → Rules → Alerts**

### Full Patient Evaluation (Main Endpoint)
```bash
curl -X POST http://localhost:8087/v1/cdss/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-123",
    "conditions": [
      {"code": {"coding": [{"system": "http://snomed.info/sct", "code": "91302008"}]},
       "clinicalStatus": {"coding": [{"code": "active"}]}}
    ],
    "observations": [
      {"code": {"coding": [{"system": "http://loinc.org", "code": "2524-7"}]},
       "valueQuantity": {"value": 3.5, "unit": "mmol/L"}}
    ],
    "options": {"evaluate_rules": true, "generate_alerts": true}
  }'
```

**Response Summary:**
| Field | Description |
|-------|-------------|
| `alerts` | Generated clinical alerts with recommendations |
| `rules_fired` | Number of compound rules triggered |
| `pipeline_used` | `THREE-CHECK` or `TWO-CHECK` |
| `execution_time_ms` | Total evaluation time |

---

### Quick Single-Code Validation
```bash
curl -X POST http://localhost:8087/v1/cdss/validate \
  -H "Content-Type: application/json" \
  -d '{"code": "91302008", "system": "http://snomed.info/sct"}'
```

---

### CDSS Health Check
```bash
curl http://localhost:8087/v1/cdss/health
```

---

### List Clinical Domains
```bash
curl http://localhost:8087/v1/cdss/domains
```

---

### List Clinical Indicators (with filtering)
```bash
# All indicators
curl http://localhost:8087/v1/cdss/indicators

# Filter by domain
curl "http://localhost:8087/v1/cdss/indicators?domain=sepsis"
```

---

### Build Facts Only (without evaluation)
```bash
curl -X POST http://localhost:8087/v1/cdss/facts/build \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-123",
    "conditions": [...],
    "observations": [...]
  }'
```

---

## CDSS API Endpoints Summary

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/v1/cdss/evaluate` | **Main** - Full CDSS pipeline |
| `POST` | `/v1/cdss/facts/build` | Extract facts from FHIR |
| `POST` | `/v1/cdss/evaluate/facts` | Evaluate pre-built facts |
| `POST` | `/v1/cdss/alerts/generate` | Generate alerts from results |
| `POST` | `/v1/cdss/validate` | Quick single-code check |
| `GET` | `/v1/cdss/health` | CDSS health status |
| `GET` | `/v1/cdss/domains` | List clinical domains |
| `GET` | `/v1/cdss/indicators` | List clinical indicators |

---

## Sample LOINC Codes for Lab Thresholds

| Code | Display | Unit | Example Threshold |
|------|---------|------|-------------------|
| `2524-7` | Lactate | mmol/L | > 2.0 (Sepsis) |
| `2160-0` | Creatinine | mg/dL | > 2.0 (AKI) |
| `2339-0` | Glucose | mg/dL | < 70 (Hypoglycemia) |
| `30934-4` | BNP | pg/mL | > 400 (Heart Failure) |
| `2708-6` | SpO2 | % | < 90% (Hypoxia) |

---

## Regional Queries (Australian Data)

Add the `X-Region: au` header for Australian terminology:

```bash
curl -H "X-Region: au" \
     -X POST http://localhost:8087/v1/subsumption/test \
     -H "Content-Type: application/json" \
     -d '{
       "code_a": "22973011000036107",
       "code_b": "414984009",
       "system": "http://snomed.info/sct"
     }'
```

---

## Value Set Identifiers

### Clinical Protocols
| Value Set | Description |
|-----------|-------------|
| `SepsisDiagnosis` | Sepsis-related SNOMED codes |
| `AUSepsisConditions` | Australian Sepsis Pathway |
| `AcuteRenalFailure` | Renal failure codes |
| `AUAKIConditions` | Australian AKI Pathway |

### FHIR Standard
| Value Set | Description |
|-----------|-------------|
| `AdministrativeGender` | male, female, other, unknown |
| `MaritalStatus` | Marital status codes |
| `AddressUse` | home, work, temp, billing |
| `IdentifierUse` | usual, official, temp, secondary |

### Medications
| Value Set | Description |
|-----------|-------------|
| `ACEInhibitors` | ACE inhibitor drugs |
| `NSAIDs` | NSAID medications |
| `Anticoagulants` | Anticoagulant drugs |

---

## THREE-CHECK PIPELINE Explained

```
Step 1: EXPANSION
  └── Load all codes from value set definition (PostgreSQL)
  └── ~1ms (cached after first call)

Step 2: EXACT MATCH
  └── Check if input code is directly in the expanded set
  └── ~0.01ms (in-memory lookup)

Step 3: SUBSUMPTION (if no exact match)
  └── Check if input code IS-A any code in the set
  └── Uses Neo4j ELK hierarchy (pre-computed)
  └── ~2-5ms
```

---

## Sample SNOMED Codes for Testing

| Code | Display | Use Case |
|------|---------|----------|
| `91302008` | Sepsis (disorder) | Exact match in SepsisDiagnosis |
| `448417001` | Streptococcal sepsis | Subsumption match (IS-A Sepsis) |
| `414984009` | Product containing oxycodone | Drug parent concept |
| `22973011000036107` | Oxycodone 80mg tablet | Australian AMT drug |

---

## Error Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| `connection refused` | Service not running | Start kb7-server |
| `value set not found` | Not seeded | Run `/v1/rules/seed` |
| `subsumption disabled` | Neo4j not connected | Check Neo4j AU container |
| `GraphDB unhealthy` | GraphDB not running | Start GraphDB container |

---

## Port Reference

| Service | Port |
|---------|------|
| KB-7 API | 8087 |
| Neo4j AU HTTP | 7475 |
| Neo4j AU Bolt | 7688 |
| GraphDB | 7200 |
| PostgreSQL | 5433 |
| Redis | 6380 |

---

## Quick Docker Commands

```bash
# Start all infrastructure
docker compose -f docker-compose.neo4j-au.yml up -d

# Check Neo4j
curl http://localhost:7475

# Check GraphDB
curl http://localhost:7200/rest/repositories
```

---

## Postman Collection

Import the Postman collection from:
```
kb-7-terminology/postman/KB7_Terminology_API.postman_collection.json
```

Set variables:
- `baseUrl`: `http://localhost:8087`
- `region`: `au` (or `us`, `in`)
