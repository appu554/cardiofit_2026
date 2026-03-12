# KB-5: Drug-Drug Interactions Service

**Comprehensive Drug Interaction Checking and Clinical Safety Platform**

## Service Configuration

| Setting | Value |
|---------|-------|
| **HTTP Port** | 8085 |
| **gRPC Port** | 8086 |
| **Service Name** | kb-5-drug-interactions |
| **Version** | 2.0.0 (Enhanced) |

## Why DDI Service is Critical

| Area | Impact |
|------|--------|
| **Patient Safety** | DDIs cause ~125,000 deaths/year in US |
| **Prescribing** | Real-time alerts prevent errors |
| **Medication Advisor** | Screen recommendations for interactions |
| **Alert Fatigue** | Tiered severity reduces noise |

## Features

### Interaction Types

| Type | Description | Example |
|------|-------------|---------|
| **Drug-Drug** | Two drugs interact | Warfarin + Ibuprofen |
| **Drug-Food** | Food affects drug | Grapefruit + Statins |
| **Drug-Disease** | Drug contraindicated in disease | NSAIDs + CKD |
| **Drug-Allergy** | Cross-reactivity | Penicillin allergy + Cephalosporins |
| **Duplicate Therapy** | Same class duplication | Two SSRIs |
| **Pharmacogenomic** | PGx/CYP metabolism | CYP2D6 poor metabolizer + Codeine |
| **Drug Class Patterns** | Multi-drug patterns | Triple Whammy (ACE/ARB + Diuretic + NSAID) |

### Severity Levels

| Level | Action | Suppressible |
|-------|--------|--------------|
| **CONTRAINDICATED** | Block prescribing | No |
| **MAJOR** | Avoid if possible, monitor closely | No |
| **MODERATE** | Monitor for effects | Yes |
| **MINOR** | Awareness only | Yes |

### Mechanisms

| Type | Examples |
|------|----------|
| **Pharmacokinetic** | CYP inhibition, P-gp transport, renal excretion |
| **Pharmacodynamic** | QT prolongation, bleeding, serotonin syndrome |

## API Endpoints

### Core Interaction Checking

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/interactions/check` | Comprehensive interaction check |
| POST | `/api/v1/interactions/batch-check` | Check medication list |
| GET | `/api/v1/interactions/quick-check?drug1=&drug2=` | Quick check two drugs |
| GET | `/api/v1/interactions/:interaction_id` | Get interaction details |
| GET | `/api/v1/interactions/search` | Search interactions |
| GET | `/api/v1/interactions/statistics` | Interaction statistics |

### Enhanced Endpoints (Phase 2)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/interactions/comprehensive` | Full analysis (DDI, PGx, food, class) |
| POST | `/api/v1/interactions/food` | Food/alcohol/herbal interactions |
| POST | `/api/v1/interactions/class` | Drug class pattern matching |

### CYP/Pharmacogenomic Analysis (Phase 2)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/cyp/profile/:drug_code` | Drug's CYP enzyme profile |
| GET | `/api/v1/cyp/interactions/:enzyme` | Get drugs affecting enzyme |

### Drug-Disease Contraindications (Phase 3)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/contraindications/disease` | Check drug-disease contraindications |
| GET | `/api/v1/contraindications/drug/:drug_code/diseases` | Diseases contraindicated for drug |
| GET | `/api/v1/contraindications/disease/:disease_code/drugs` | Drugs contraindicated for disease |

### Allergy Cross-Reactivity (Phase 3)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/allergy/check` | Check allergy cross-reactivity risk |
| GET | `/api/v1/allergy/cross-reactivity/:allergen` | Get cross-reactive drugs |
| GET | `/api/v1/allergy/common-patterns` | Common cross-reactivity patterns |

### Duplicate Therapy Detection (Phase 3)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/duplicates/check` | Check for therapeutic duplication |
| GET | `/api/v1/duplicates/classes/:drug_code` | Drug's therapeutic classes |
| GET | `/api/v1/duplicates/common-classes` | Common duplicate therapy classes |

### Drug Information

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/drugs/:drug_code/interactions` | Get drug's interactions |
| GET | `/api/v1/drugs/:drug_code/synonyms` | Get drug synonyms |
| GET | `/api/v1/drugs/:drug_code/alternatives` | Get safer alternatives |

### Patient History

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/patients/:patient_id/interactions/history` | Patient interaction history |

### Alert Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| PUT | `/api/v1/alerts/:alert_id/override` | Override alert with reason |

### Administration

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Service health check |
| GET | `/metrics` | Prometheus metrics |
| GET | `/api/v1/admin/stats` | System statistics |
| POST | `/api/v1/admin/cache/clear` | Clear cache (type: interactions, synonyms, rules, all) |
| GET | `/api/v1/admin/database/health` | Database health |
| POST | `/api/v1/admin/rules/reload` | Reload interaction rules |
| GET | `/api/v1/admin/analytics` | Analytics dashboard |

## Built-in Clinical Knowledge

### High-Priority DDIs

| Drug Pair | Severity | Mechanism | Clinical Effect |
|-----------|----------|-----------|-----------------|
| **Warfarin + NSAIDs** | MAJOR | PD-Bleeding | GI hemorrhage |
| **Warfarin + Amiodarone** | MAJOR | CYP2C9 inhibition | INR ↑30-50% |
| **Warfarin + Fluconazole** | MAJOR | CYP2C9 inhibition | INR doubles |
| **Simvastatin + Amiodarone** | CONTRAINDICATED | CYP3A4 | Rhabdomyolysis |
| **Opioid + Benzodiazepine** | CONTRAINDICATED | CNS depression | Respiratory failure |
| **ACE + ARB** | CONTRAINDICATED | Dual RAAS | Hyperkalemia, AKI |
| **ACE + Spironolactone** | MAJOR | Hyperkalemia | Cardiac arrhythmia |
| **SSRI + MAOIs** | CONTRAINDICATED | Serotonergic | Serotonin syndrome |
| **SSRI + Tramadol** | MAJOR | Serotonergic | Serotonin syndrome |
| **Methotrexate + TMP-SMX** | CONTRAINDICATED | Antifolate | Pancytopenia |
| **Digoxin + Amiodarone** | MAJOR | P-gp inhibition | Digoxin toxicity |
| **QT drugs + QT drugs** | MAJOR | QT prolongation | Torsades de pointes |

### Drug-Disease Contraindications (Phase 3)

| Drug | Disease | Severity | Rationale |
|------|---------|----------|-----------|
| NSAIDs | CKD | MAJOR | Worsens renal function |
| NSAIDs | Heart Failure | MAJOR | Fluid retention |
| NSAIDs | GI Bleed history | CONTRAINDICATED | Bleeding risk |
| Metformin | CKD Stage 4-5 | CONTRAINDICATED | Lactic acidosis |
| Beta blockers | Asthma | MAJOR | Bronchospasm |
| ACE inhibitors | Pregnancy | CONTRAINDICATED | Fetal toxicity |
| Fluoroquinolones | Myasthenia Gravis | MAJOR | Exacerbation |
| Opioids | Respiratory Failure | CONTRAINDICATED | Respiratory depression |

### Allergy Cross-Reactivity (Phase 3)

| Allergen | Cross-Reactive Drug | Rate | Severity |
|----------|---------------------|------|----------|
| Penicillin | 1st-gen Cephalosporins | ~2% | Moderate |
| Penicillin | 3rd-gen Cephalosporins | <1% | Minor |
| Penicillin | Carbapenems | ~1% | Moderate |
| Aspirin | NSAIDs (AERD patients) | 20-25% | Major |
| ACE inhibitor (angioedema) | Other ACE inhibitors | 95% | Major |
| Sulfonamide antibiotics | Thiazides | <1% | Minor |

### Duplicate Therapy Classes (Phase 3)

| ATC Code | Class | Severity | Max Drugs |
|----------|-------|----------|-----------|
| M01A | NSAIDs | MAJOR | 1 |
| C07 | Beta-blockers | MAJOR | 1 |
| N05BA | Benzodiazepines | MAJOR | 1 |
| N06AB | SSRIs | MAJOR | 1 |
| C09AA | ACE Inhibitors | MAJOR | 1 |
| C10AA | Statins | MAJOR | 1 |
| A10BB | Sulfonylureas | MAJOR | 1 |

## Example: Check Interactions

```bash
curl -X POST http://localhost:8085/api/v1/interactions/check \
  -H "Content-Type: application/json" \
  -d '{
    "drug_codes": ["RXCUI:5640", "RXCUI:11289"],
    "include_food_interactions": true,
    "include_disease_check": true,
    "patient_diagnoses": ["N18.3"]
  }'
```

Response:
```json
{
  "has_interactions": true,
  "total_interactions": 2,
  "major_count": 2,
  "drug_drug_interactions": [
    {
      "drug1": "Warfarin",
      "drug2": "Ibuprofen",
      "severity": "major",
      "clinical_effect": "Increased bleeding risk",
      "management": "Avoid. Use acetaminophen instead."
    }
  ],
  "drug_disease_interactions": [
    {
      "drug": "Ibuprofen",
      "disease": "CKD",
      "severity": "major",
      "management": "Avoid if possible"
    }
  ],
  "requires_action": true,
  "block_prescribing": false
}
```

## File Structure

```
kb-5-drug-interactions/
├── main.go                           Entry point, server initialization
├── internal/
│   ├── api/
│   │   ├── server.go                 HTTP server and routes
│   │   └── handlers.go               Request handlers
│   ├── services/
│   │   ├── interaction_service.go    Core DDI service
│   │   ├── pgx_engine.go             Pharmacogenomic engine (Phase 2)
│   │   ├── class_interaction_engine.go Drug class patterns (Phase 2)
│   │   ├── modifier_engine.go        Food/alcohol/herbal (Phase 2)
│   │   ├── drug_disease_engine.go    Drug-disease contraindications (Phase 3)
│   │   ├── allergy_engine.go         Allergy cross-reactivity (Phase 3)
│   │   └── duplicate_therapy_engine.go Duplicate therapy detection (Phase 3)
│   ├── models/                       Data models
│   ├── database/                     Database connection
│   ├── cache/                        Redis caching
│   ├── metrics/                      Prometheus metrics
│   └── config/                       Configuration
├── migrations/
│   ├── 001_initial_schema.sql
│   ├── 002_enhanced_schema.sql       Phase 2 tables
│   ├── 003_phase3_clinical_safety.sql Phase 3 tables
│   └── 004_seed_phase3_clinical_data.sql Seed data
├── api/                              gRPC protobuf definitions
├── Dockerfile
├── go.mod
└── kb5-README.md
```

## Integration Points

### Upstream
- **KB-7 Terminology**: RxNorm code validation, SNOMED-CT mappings
- **KB-2 Patient Context**: Patient diagnoses, allergies, medications

### Downstream
- **Medication Advisor Engine**: Screen recommendations
- **CPOE (KB-12)**: Pre-order interaction check
- **Pharmacy**: Dispensing verification
- **Alerts Engine**: Trigger clinical alerts

## Quick Start

```bash
# Docker
docker build -t kb5-drug-interactions .
docker run -p 8085:8085 -p 8086:8086 kb5-drug-interactions

# Local
go run main.go
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8085 | HTTP server port |
| `GRPC_PORT` | 8086 | gRPC server port |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `REDIS_URL` | - | Redis connection string |
| `REDIS_HOT_CACHE_URL` | - | Hot cache Redis URL |
| `REDIS_WARM_CACHE_URL` | - | Warm cache Redis URL |
| `ENVIRONMENT` | development | Environment (development/production) |
| `LOG_LEVEL` | info | Logging level |

## Alert Fatigue Management

The service supports tiered alerting to reduce alert fatigue:

1. **Non-suppressible alerts**: CONTRAINDICATED, MAJOR (require action)
2. **Suppressible alerts**: MODERATE, MINOR (can be configured)
3. **Override reasons**: Document clinical justification
4. **Context-aware**: Adjust based on patient factors

## Enhanced Features (v2.0)

- Pharmacogenomic (PGx) interaction detection
- Drug class pattern matching (Triple Whammy, etc.)
- Food/alcohol/herbal interaction analysis
- Drug-disease contraindication analysis
- Allergy cross-reactivity detection
- Duplicate therapy detection
- Hot/warm cache optimization (50k/200k entries)
- Evidence Envelope compliance
- Institutional override system
- Real-time confidence scoring
- Clinical alert synthesis
- Parallel engine processing

## License

Proprietary - Healthcare Platform
