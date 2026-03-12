# KB-1 Drug Dosing Rules Service - Complete Documentation

**Status**: ✅ Fully Operational
**Date**: 2025-12-23
**Port**: 8081 (HTTP) | 9081 (gRPC) | 5481 (PostgreSQL) | 6382 (Redis)

---

## Overview

KB-1 is a Go-based Drug Dosing Rules service providing clinical dose calculations with safety validation:

1. **Dose Calculation** - Fixed, weight-based, BSA-based, titration methods
2. **Renal Adjustment** - CKD staging with automatic dose modification
3. **Pediatric Dosing** - Age/weight-based pediatric calculations
4. **Patient Parameters** - BSA, IBW, CrCl, eGFR calculators
5. **Dose Validation** - Safety alerts, high-alert flags, black box warnings
6. **Multi-Region Support** - US, EU, CA, AU, IN regulatory compliance

### Drug Coverage

**23 drugs across 5 therapeutic categories:**

| Category | Drugs | Count |
|----------|-------|-------|
| **Diabetes** | Metformin, Empagliflozin, Liraglutide, Insulin Glargine | 4 |
| **Cardiovascular** | Lisinopril, Losartan, Atorvastatin, Furosemide, Carvedilol, Metoprolol, Spironolactone | 7 |
| **Anticoagulants** | Warfarin, Enoxaparin, Heparin, Apixaban | 4 |
| **Antibiotics** | Vancomycin, Gentamicin, Ciprofloxacin, Amoxicillin | 4 |
| **Pain** | Acetaminophen, Ibuprofen, Oxycodone, Morphine | 4 |

---

## Quick Start

### Start the Service

```bash
cd backend/services/knowledge-base-services/kb-drug-rules

# Start all containers (postgres, redis, app)
docker-compose -f docker-compose.kb1.yml up -d

# Verify health
curl http://localhost:8081/health
```

### Stop the Service

```bash
docker-compose -f docker-compose.kb1.yml down        # Stop containers
docker-compose -f docker-compose.kb1.yml down -v     # Stop + remove data
```

### View Logs

```bash
docker-compose -f docker-compose.kb1.yml logs -f kb-drug-rules
```

---

## API Endpoints

### Health & Admin

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Service health status |
| `/metrics` | GET | Prometheus metrics |
| `/v1/rules` | GET | List all drug rules |
| `/v1/items/:rxnorm_code` | GET | Get specific drug rule |

### Dose Calculation

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/calculate` | POST | Comprehensive dose calculation |
| `/v1/calculate/weight-based` | POST | Weight-based dosing (mg/kg) |
| `/v1/calculate/bsa-based` | POST | BSA-based dosing (mg/m²) |
| `/v1/calculate/pediatric` | POST | Pediatric dose calculation |
| `/v1/calculate/renal` | POST | Renal-adjusted dosing |

### Patient Parameters

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/patient/bsa` | POST | Body Surface Area (Mosteller/DuBois) |
| `/v1/patient/ibw` | POST | Ideal Body Weight (Devine formula) |
| `/v1/patient/crcl` | POST | Creatinine Clearance (Cockcroft-Gault) |
| `/v1/patient/egfr` | POST | eGFR (CKD-EPI 2021) |

### Dose Validation

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/validate/dose` | POST | Validate proposed dose with safety alerts |

### Governance-Enhanced Endpoints (Tier-7 Compliance)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/governance/validate` | POST | Validate dose with governance severity mapping |
| `/v1/governance/calculate` | POST | Calculate dose with full provenance trail |
| `/v1/governance/severities` | GET | List all governance severity levels |
| `/v1/governance/provenance/:rxnorm` | GET | Get evidence provenance for a drug |

---

## Test Examples

### 1. Health Check

```bash
curl http://localhost:8081/health
```

**Response:**
```json
{
  "service": "kb-drug-rules",
  "status": "healthy",
  "timestamp": "2025-12-23T10:44:51Z",
  "version": "1.0.0"
}
```

### 2. List All Drug Rules

```bash
curl http://localhost:8081/v1/rules
```

**Response:**
```json
{
  "rules": [
    {
      "rxnorm_code": "6809",
      "drug_name": "Metformin",
      "therapeutic_class": "Biguanide Antidiabetic",
      "dosing_method": "FIXED",
      "is_high_alert": false,
      "has_black_box": false,
      "is_narrow_ti": false
    }
  ],
  "total_rules": 23
}
```

### 3. Get Specific Drug Rule

```bash
curl http://localhost:8081/v1/items/6809
```

**Response:**
```json
{
  "rxnorm_code": "6809",
  "drug_name": "Metformin",
  "dosing_method": "FIXED",
  "default_dose": 500,
  "dose_unit": "mg",
  "frequency": "BID",
  "max_single_dose": 1000,
  "max_daily_dose": 2000,
  "renal_adjustments": [
    {"egfr_min": 30, "egfr_max": 45, "dose_multiplier": 0.5, "max_dose_cap": 1000}
  ],
  "monitoring_required": ["HbA1c", "Renal function", "Vitamin B12"],
  "warnings": ["Hold before contrast procedures"]
}
```

### 4. Weight-Based Dose Calculation (Enoxaparin)

```bash
curl -X POST http://localhost:8081/v1/calculate/weight-based \
  -H "Content-Type: application/json" \
  -d '{
    "rxnorm_code": "67108",
    "weight_kg": 80
  }'
```

**Response:**
```json
{
  "success": true,
  "drug_name": "Enoxaparin",
  "rxnorm_code": "67108",
  "calculated_dose": 80,
  "dose_unit": "mg",
  "dose_per_kg": 1,
  "weight_kg": 80,
  "frequency": "Q12H",
  "max_daily_dose": 300,
  "calculation_basis": "Weight-based: 1.00 mg/kg × 80.0 kg"
}
```

### 5. Renal-Adjusted Dosing (Vancomycin)

```bash
curl -X POST http://localhost:8081/v1/calculate/renal \
  -H "Content-Type: application/json" \
  -d '{
    "rxnorm_code": "11124",
    "age": 70,
    "gender": "male",
    "weight_kg": 75,
    "serum_creatinine": 2.0
  }'
```

**Response:**
```json
{
  "success": true,
  "drug_name": "Vancomycin",
  "original_dose": 1050,
  "adjusted_dose": 787.5,
  "dose_unit": "mg",
  "ckd_stage": "G3b",
  "egfr": 35.2,
  "dose_multiplier": 0.75,
  "adjustment_reason": "CKD Stage G3b: Reduce dose by 25%",
  "monitoring": ["Trough levels", "Renal function"]
}
```

### 6. Pediatric Dosing (Amoxicillin)

```bash
curl -X POST http://localhost:8081/v1/calculate/pediatric \
  -H "Content-Type: application/json" \
  -d '{
    "rxnorm_code": "723",
    "age": 8,
    "weight_kg": 25,
    "indication": "otitis_media"
  }'
```

**Response:**
```json
{
  "success": true,
  "drug_name": "Amoxicillin",
  "rxnorm_code": "723",
  "calculated_dose": 0,
  "dose_unit": "mg",
  "frequency": "TID",
  "patient_age": 8,
  "patient_weight_kg": 25,
  "calculation_basis": "Fixed dose with pediatric adjustment",
  "warnings": ["Pediatric: 25-50 mg/kg/day divided TID"]
}
```

### 7. BSA Calculation

```bash
curl -X POST http://localhost:8081/v1/patient/bsa \
  -H "Content-Type: application/json" \
  -d '{
    "height_cm": 175,
    "weight_kg": 80
  }'
```

**Response:**
```json
{
  "success": true,
  "bsa_m2": 1.97,
  "height_cm": 175,
  "weight_kg": 80,
  "formula": "Mosteller: √[(Height × Weight) / 3600]",
  "reference": "Mosteller RD. N Engl J Med 1987;317:1098"
}
```

### 8. eGFR Calculation

```bash
curl -X POST http://localhost:8081/v1/patient/egfr \
  -H "Content-Type: application/json" \
  -d '{
    "age": 65,
    "serum_creatinine": 1.2,
    "gender": "male"
  }'
```

**Response:**
```json
{
  "success": true,
  "egfr_ml_min": 67.1,
  "ckd_stage": "G2",
  "ckd_description": "Mildly decreased kidney function",
  "age": 65,
  "gender": "male",
  "serum_creatinine": 1.2,
  "formula": "CKD-EPI 2021 (race-free)",
  "reference": "Inker LA, et al. N Engl J Med 2021;385:1737-1749"
}
```

### 9. Creatinine Clearance (Cockcroft-Gault)

```bash
curl -X POST http://localhost:8081/v1/patient/crcl \
  -H "Content-Type: application/json" \
  -d '{
    "age": 70,
    "weight_kg": 75,
    "serum_creatinine": 1.5,
    "gender": "male"
  }'
```

**Response:**
```json
{
  "success": true,
  "crcl_ml_min": 48.6,
  "age": 70,
  "weight_kg": 75,
  "serum_creatinine": 1.5,
  "gender": "male",
  "formula": "Cockcroft-Gault: [(140 - Age) × Weight] / [72 × SCr] (× 0.85 if female)",
  "reference": "Cockcroft DW, Gault MH. Nephron 1976;16:31-41"
}
```

### 10. Ideal Body Weight

```bash
curl -X POST http://localhost:8081/v1/patient/ibw \
  -H "Content-Type: application/json" \
  -d '{
    "height_cm": 175,
    "gender": "male"
  }'
```

**Response:**
```json
{
  "success": true,
  "ibw_kg": 70.5,
  "height_cm": 175,
  "gender": "male",
  "formula": "Devine: Male: 50 + 2.3×(height_in - 60), Female: 45.5 + 2.3×(height_in - 60)",
  "reference": "Devine BJ. Drug Intell Clin Pharm 1974;8:650-655"
}
```

### 11. Dose Validation with Safety Alerts (Warfarin)

```bash
curl -X POST http://localhost:8081/v1/validate/dose \
  -H "Content-Type: application/json" \
  -d '{
    "rxnorm_code": "11289",
    "proposed_dose": 15,
    "frequency": "daily",
    "age": 75
  }'
```

**Response:**
```json
{
  "valid": true,
  "proposed_dose": 15,
  "max_allowed_dose": 15,
  "min_allowed_dose": 1,
  "warnings": [
    "Proposed dose exceeds typical single dose of 10 mg",
    "Elderly often require lower doses"
  ],
  "safety_alerts": [
    "HIGH-ALERT medication - requires independent double-check",
    "NARROW THERAPEUTIC INDEX - monitor levels closely"
  ]
}
```

### 12. Dose Validation - Overdose Detection (Acetaminophen)

```bash
curl -X POST http://localhost:8081/v1/validate/dose \
  -H "Content-Type: application/json" \
  -d '{
    "rxnorm_code": "161",
    "proposed_dose": 5000,
    "frequency": "daily",
    "age": 45
  }'
```

**Response:**
```json
{
  "valid": false,
  "proposed_dose": 5000,
  "max_allowed_dose": 4000,
  "min_allowed_dose": 325,
  "warnings": [
    "Proposed dose exceeds typical single dose of 1000 mg"
  ],
  "errors": [
    "Proposed dose exceeds maximum daily dose of 4000 mg"
  ]
}
```

### 13. Comprehensive Dose Calculation (Metformin with Renal Monitoring)

```bash
curl -X POST http://localhost:8081/v1/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "rxnorm_code": "6809",
    "age": 65,
    "gender": "male",
    "weight_kg": 85,
    "height_cm": 178,
    "serum_creatinine": 1.4,
    "indication": "type_2_diabetes"
  }'
```

**Response:**
```json
{
  "success": true,
  "drug_name": "Metformin",
  "rxnorm_code": "6809",
  "recommended_dose": 500,
  "dose_unit": "mg",
  "frequency": "BID",
  "dosing_method": "FIXED",
  "calculation_basis": "Fixed dose: 500.00 mg",
  "renal_adjustment": {
    "applied": true,
    "reason": "eGFR 55.8 mL/min/1.73m² (range 45-60)",
    "multiplier": 1,
    "max_dose_cap": 2000,
    "notes": "Monitor renal function, hold for contrast"
  },
  "patient_parameters": {
    "bsa": 2.05,
    "ibw": 73.2,
    "adj_bw": 77.9,
    "crcl": 63.2,
    "egfr": 55.8,
    "bmi": 26.8,
    "is_obese": false,
    "is_pediatric": false,
    "is_geriatric": true
  },
  "monitoring_required": ["HbA1c", "Renal function", "Vitamin B12"]
}
```

---

## Drug Rules Reference

### Complete Drug List with RxNorm Codes

| Drug Name | RxNorm | Class | Dosing Method | High-Alert | Black Box | Narrow TI |
|-----------|--------|-------|---------------|------------|-----------|-----------|
| Acetaminophen | 161 | Analgesic | FIXED | ❌ | ❌ | ❌ |
| Amoxicillin | 723 | Aminopenicillin | FIXED | ❌ | ❌ | ❌ |
| Apixaban | 1364430 | Factor Xa Inhibitor | FIXED | ✅ | ❌ | ❌ |
| Atorvastatin | 83367 | HMG-CoA Reductase Inhibitor | FIXED | ❌ | ❌ | ❌ |
| Carvedilol | 20352 | Beta/Alpha Blocker | TITRATION | ❌ | ❌ | ❌ |
| Ciprofloxacin | 2551 | Fluoroquinolone | FIXED | ❌ | ✅ | ❌ |
| Empagliflozin | 1545653 | SGLT2 Inhibitor | FIXED | ❌ | ❌ | ❌ |
| Enoxaparin | 67108 | LMWH | WEIGHT_BASED | ✅ | ❌ | ❌ |
| Furosemide | 4603 | Loop Diuretic | FIXED | ❌ | ❌ | ❌ |
| Gentamicin | 3058 | Aminoglycoside | WEIGHT_BASED | ✅ | ❌ | ✅ |
| Heparin | 5224 | UFH | WEIGHT_BASED | ✅ | ❌ | ✅ |
| Ibuprofen | 5640 | NSAID | FIXED | ❌ | ❌ | ❌ |
| Insulin Glargine | 261551 | Long-Acting Insulin | WEIGHT_BASED | ✅ | ❌ | ❌ |
| Liraglutide | 475968 | GLP-1 Agonist | TITRATION | ❌ | ✅ | ❌ |
| Lisinopril | 8610 | ACE Inhibitor | FIXED | ❌ | ❌ | ❌ |
| Losartan | 52175 | ARB | FIXED | ❌ | ❌ | ❌ |
| Metformin | 6809 | Biguanide | FIXED | ❌ | ❌ | ❌ |
| Metoprolol Succinate | 866924 | Beta-1 Blocker | TITRATION | ❌ | ❌ | ❌ |
| Morphine Sulfate | 7052 | Opioid | FIXED | ✅ | ✅ | ❌ |
| Oxycodone | 7804 | Opioid | FIXED | ✅ | ✅ | ❌ |
| Spironolactone | 9997 | Aldosterone Antagonist | FIXED | ❌ | ❌ | ❌ |
| Vancomycin | 11124 | Glycopeptide | WEIGHT_BASED | ❌ | ❌ | ✅ |
| Warfarin | 11289 | Vitamin K Antagonist | FIXED | ✅ | ❌ | ✅ |

### Safety Flags Explained

| Flag | Meaning | Clinical Action |
|------|---------|-----------------|
| **High-Alert** | High risk of significant harm if used in error | Independent double-check required |
| **Black Box** | FDA black box warning | Review specific warnings before prescribing |
| **Narrow TI** | Narrow therapeutic index | Monitor drug levels closely |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        KB-1 Service                             │
│                     (Go + Gin Framework)                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   HTTP API  │  │  gRPC API   │  │    Prometheus Metrics   │  │
│  │   :8081     │  │   :9081     │  │        /metrics         │  │
│  └──────┬──────┘  └──────┬──────┘  └─────────────────────────┘  │
│         │                │                                       │
│  ┌──────▼────────────────▼──────┐                               │
│  │      Calculation Engine      │                               │
│  │  - Fixed Dose                │                               │
│  │  - Weight-Based (mg/kg)      │                               │
│  │  - BSA-Based (mg/m²)         │                               │
│  │  - Titration                 │                               │
│  │  - Renal Adjustment          │                               │
│  │  - Pediatric                 │                               │
│  └──────┬───────────────────────┘                               │
│         │                                                        │
│  ┌──────▼──────┐    ┌───────────────┐    ┌──────────────────┐   │
│  │ Drug Rules  │    │ Safety Engine │    │ Patient Params   │   │
│  │ (23 drugs)  │    │ - High Alert  │    │ - BSA (Mosteller)│   │
│  │             │    │ - Black Box   │    │ - IBW (Devine)   │   │
│  │             │    │ - Narrow TI   │    │ - CrCl (C-G)     │   │
│  │             │    │ - Dose Limits │    │ - eGFR (CKD-EPI) │   │
│  └─────────────┘    └───────────────┘    └──────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│                     Data Layer                                   │
│  ┌─────────────────┐           ┌─────────────────┐              │
│  │   PostgreSQL    │           │      Redis      │              │
│  │     :5481       │           │      :6382      │              │
│  │  - Drug Rules   │           │  - Cache Layer  │              │
│  │  - Audit Logs   │           │  - Prewarm Top5 │              │
│  └─────────────────┘           └─────────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8081 | HTTP server port |
| `GRPC_PORT` | 9081 | gRPC server port |
| `DEBUG` | false | Enable debug logging |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `REDIS_URL` | - | Redis connection string |
| `CACHE_TTL` | 3600 | Cache TTL in seconds |
| `CACHE_MAX_SIZE` | 10000 | Maximum cache entries |
| `DEFAULT_REGION` | US | Default regulatory region |
| `SUPPORTED_REGIONS` | US,EU,CA,AU,IN | Comma-separated regions |
| `REQUIRE_APPROVAL` | false | Require rule approval |
| `REQUIRE_SIGNATURE` | false | Require digital signatures |
| `METRICS_ENABLED` | true | Enable Prometheus metrics |
| `METRICS_PATH` | /metrics | Metrics endpoint path |

---

## Docker Compose Configuration

```yaml
# docker-compose.kb1.yml
services:
  kb-drug-rules:
    build: .
    container_name: kb1-drug-rules
    ports:
      - "8081:8081"
    environment:
      - DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@kb1-postgres:5432/kb_drug_rules
      - REDIS_URL=redis://kb1-redis:6379/0
    depends_on:
      kb1-postgres:
        condition: service_healthy
      kb1-redis:
        condition: service_healthy

  kb1-postgres:
    image: postgres:15-alpine
    container_name: kb1-postgres
    ports:
      - "5481:5432"
    environment:
      - POSTGRES_USER=kb_drug_rules_user
      - POSTGRES_PASSWORD=kb_password
      - POSTGRES_DB=kb_drug_rules

  kb1-redis:
    image: redis:7-alpine
    container_name: kb1-redis
    ports:
      - "6382:6379"
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker-compose -f docker-compose.kb1.yml logs kb-drug-rules

# Common issues:
# 1. Database not ready - wait for postgres healthcheck
# 2. Port conflict - check if 8081 is in use: lsof -i:8081
# 3. Config error - check environment variables
```

### Database Connection Failed

```bash
# Test database connectivity
docker exec kb1-postgres pg_isready -U kb_drug_rules_user -d kb_drug_rules

# Check credentials match docker-compose environment
```

### Rules Not Loading

```bash
# Check rule count
curl http://localhost:8081/v1/rules | jq '.total_rules'

# Should return 23 - if 0, check database migrations
docker logs kb1-drug-rules | grep -i migration
```

---

## Integration with Other KB Services

KB-1 can be integrated with:

| Service | Integration Point | Use Case |
|---------|-------------------|----------|
| **KB-3 Guidelines** | Clinical protocols | Protocol-based dosing recommendations |
| **KB-4 Patient Safety** | Safety alerts | Cross-service safety validation |
| **KB-5 Drug Interactions** | DDI checking | Combined dose + interaction checking |
| **KB-7 Terminology** | Code lookups | RxNorm/SNOMED code validation |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-12-23 | Initial release with 23 drugs |

---

## Support

For issues or questions:
- Check service logs: `docker-compose logs -f kb-drug-rules`
- Verify health: `curl http://localhost:8081/health`
- Review this documentation for API examples
