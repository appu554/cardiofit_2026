# KB-1 Drug Dosing Rules Service

## Overview

KB-1 is a clinical drug dosing calculation service that provides evidence-based dosing recommendations with automatic adjustments for renal, hepatic, and age-related factors.

**Port:** 8081
**Container:** `kb-drug-rules`
**Drugs:** 37 medications across 7 categories

## Quick Start

### Running the Service

```bash
# Docker (recommended)
docker run -d --name kb-drug-rules -p 8081:8081 kb-drug-rules:latest

# Or build and run locally
go build -o bin/kb-1-drug-rules ./cmd/server
PORT=8081 ./bin/kb-1-drug-rules
```

### Health Check

```bash
curl http://localhost:8081/health
```

---

## API Endpoints

### 1. Drug Rules

#### List All Drug Rules
```bash
GET /v1/rules
```

**Response:**
```json
{
  "query": "",
  "count": 37,
  "results": [
    {
      "rxnorm_code": "6809",
      "drug_name": "Metformin",
      "drug_class": "Biguanide",
      "category": "diabetes",
      "dosing_method": "FIXED"
    }
  ]
}
```

#### Get Single Drug Rule
```bash
GET /v1/rules/:rxnorm
```

**Example:**
```bash
curl http://localhost:8081/v1/rules/6809
```

#### Search Drug Rules
```bash
GET /v1/rules/search?category=cardiovascular
GET /v1/rules/search?name=metformin
GET /v1/rules/search?class=ACE%20Inhibitor
```

---

### 2. Dose Calculation

#### General Dose Calculation
```bash
POST /v1/calculate
```

**Request:**
```json
{
  "rxnorm_code": "6809",
  "patient": {
    "weight_kg": 70,
    "height_cm": 175,
    "age": 55,
    "gender": "M",
    "serum_creatinine": 1.0
  },
  "indication": "type2_diabetes"
}
```

**Response:**
```json
{
  "success": true,
  "drug_name": "Metformin",
  "recommended_dose": 500,
  "unit": "mg",
  "frequency": "BID",
  "route": "oral",
  "dosing_method": "FIXED",
  "dose_range": { "min": 500, "max": 1000 },
  "calculated_parameters": {
    "bsa": 1.84,
    "ibw": 70.5,
    "bmi": 22.9,
    "crcl": 82.6,
    "egfr": 88.9,
    "ckd_stage": "G2"
  },
  "monitoring": ["Renal function", "B12 levels annually"]
}
```

#### Renal-Adjusted Dose
```bash
POST /v1/calculate/renal
```

**Request:**
```json
{
  "rxnorm_code": "6809",
  "patient": {
    "weight_kg": 70,
    "height_cm": 175,
    "age": 65,
    "gender": "M",
    "serum_creatinine": 2.5
  },
  "egfr": 28
}
```

**Response (CKD Stage 4):**
```json
{
  "success": true,
  "drug_name": "Metformin",
  "original_dose": 500,
  "adjusted_dose": 0,
  "egfr": 27.8,
  "ckd_stage": "G4",
  "contraindicated": true,
  "recommendation": "Contraindicated in eGFR <30"
}
```

#### Geriatric Dose
```bash
POST /v1/calculate/geriatric
```

**Request:**
```json
{
  "rxnorm_code": "7052",
  "patient": {
    "weight_kg": 60,
    "height_cm": 165,
    "age": 78,
    "gender": "F"
  }
}
```

**Response:**
```json
{
  "success": true,
  "drug_name": "Morphine",
  "recommended_dose": 7.5,
  "unit": "mg",
  "adjustment_notes": "Start at 50% dose in elderly"
}
```

#### Weight-Based Dose
```bash
POST /v1/calculate/weight-based
```

#### BSA-Based Dose
```bash
POST /v1/calculate/bsa-based
```

#### Pediatric Dose
```bash
POST /v1/calculate/pediatric
```

#### Hepatic-Adjusted Dose
```bash
POST /v1/calculate/hepatic
```

---

### 3. Patient Parameter Calculations

#### Calculate BSA (Body Surface Area)
```bash
POST /v1/patient/bsa
```

**Request:**
```json
{
  "weight_kg": 70,
  "height_cm": 175
}
```

**Response:**
```json
{
  "bsa": 1.84,
  "formula": "Mosteller",
  "height_cm": 175,
  "weight_kg": 70
}
```

#### Calculate eGFR
```bash
POST /v1/patient/egfr
```

**Request:**
```json
{
  "age": 55,
  "gender": "M",
  "serum_creatinine": 1.2
}
```

**Response:**
```json
{
  "egfr": 71.4,
  "formula": "CKD-EPI 2021",
  "ckd_stage": "G2",
  "ckd_description": "Mildly decreased kidney function"
}
```

#### Calculate CrCl (Creatinine Clearance)
```bash
POST /v1/patient/crcl
```

#### Calculate IBW (Ideal Body Weight)
```bash
POST /v1/patient/ibw
```

---

### 4. Dose Validation

#### Validate Dose Against Max Limits
```bash
POST /v1/validate/dose
```

**Request:**
```json
{
  "rxnorm_code": "161",
  "dose": 1500,
  "frequency": "Q6H"
}
```

#### Get Max Dose Information
```bash
GET /v1/validate/max-dose?rxnorm=161&dose=5000
```

**Response:**
```json
{
  "rxnorm_code": "161",
  "drug_name": "Acetaminophen",
  "max_single_dose": 1000,
  "max_daily_dose": 4000,
  "unit": "mg"
}
```

---

### 5. High-Alert Medication Check

```bash
GET /v1/high-alert/check?rxnorm=11289
```

**Response:**
```json
{
  "rxnorm_code": "11289",
  "drug_name": "Warfarin",
  "is_high_alert": true,
  "is_narrow_ti": true,
  "has_black_box_warning": false,
  "is_beers_list": false
}
```

---

### 6. Adjustment Information

#### Renal Adjustment Guidelines
```bash
GET /v1/adjustments/renal?rxnorm=6809
```

#### Hepatic Adjustment Guidelines
```bash
GET /v1/adjustments/hepatic?rxnorm=52175
```

#### Age-Based Adjustment Guidelines
```bash
GET /v1/adjustments/age?rxnorm=7052
```

---

## Drug Categories

| Category | Count | Examples |
|----------|-------|----------|
| **Diabetes** | 4 | Metformin, Empagliflozin, Liraglutide, Insulin Glargine |
| **Cardiovascular** | 15 | Lisinopril, Ramipril, Losartan, Amlodipine, Metoprolol, Atorvastatin |
| **Anticoagulant** | 7 | Warfarin, Apixaban, Rivaroxaban, Dabigatran, Enoxaparin, Heparin |
| **Antibiotic** | 4 | Amoxicillin, Vancomycin, Gentamicin, Ciprofloxacin |
| **Pain** | 4 | Acetaminophen, Ibuprofen, Morphine, Oxycodone |
| **Gastrointestinal** | 2 | Pantoprazole, Omeprazole |
| **Antihistamine** | 1 | Diphenhydramine |

---

## Clinical Formulas Used

| Calculation | Formula | Reference |
|-------------|---------|-----------|
| **BSA** | Mosteller: √(height × weight / 3600) | Mosteller RD, 1987 |
| **eGFR** | CKD-EPI 2021 (race-free) | NEJM 2021 |
| **CrCl** | Cockcroft-Gault | Nephron 1976 |
| **IBW** | Devine formula | Devine BJ, 1974 |

---

## CKD Staging

| Stage | eGFR Range | Description |
|-------|------------|-------------|
| G1 | ≥90 | Normal or high |
| G2 | 60-89 | Mildly decreased |
| G3a | 45-59 | Mildly to moderately decreased |
| G3b | 30-44 | Moderately to severely decreased |
| G4 | 15-29 | Severely decreased |
| G5 | <15 | Kidney failure |

---

## Safety Features

### High-Alert Medications
Flagged drugs requiring extra verification:
- Warfarin, Heparin, Enoxaparin
- Apixaban, Rivaroxaban, Dabigatran, Edoxaban
- Morphine, Oxycodone
- Insulin Glargine

### Narrow Therapeutic Index
Drugs with small margin between therapeutic and toxic doses:
- Warfarin
- Digoxin (if added)

### Beers Criteria (Geriatric)
Medications to avoid in elderly:
- Diphenhydramine (anticholinergic)
- Pantoprazole, Omeprazole (>8 weeks)

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8081 | Server port |
| `ENVIRONMENT` | development | development/staging/production |
| `LOG_LEVEL` | info | debug/info/warn/error |

---

## Docker Commands

```bash
# Build image
docker build -t kb-drug-rules:latest .

# Run container
docker run -d --name kb-drug-rules -p 8081:8081 kb-drug-rules:latest

# View logs
docker logs kb-drug-rules

# Stop container
docker stop kb-drug-rules

# Remove container
docker rm kb-drug-rules
```

---

## Integration Example

### Calculate Dose for CKD Patient

```bash
# Step 1: Calculate eGFR
curl -s -X POST http://localhost:8081/v1/patient/egfr \
  -H "Content-Type: application/json" \
  -d '{"age": 72, "gender": "F", "serum_creatinine": 1.8}'

# Step 2: Get renal-adjusted dose
curl -s -X POST http://localhost:8081/v1/calculate/renal \
  -H "Content-Type: application/json" \
  -d '{
    "rxnorm_code": "1364430",
    "patient": {"weight_kg": 55, "height_cm": 160, "age": 72, "gender": "F"},
    "egfr": 35
  }'
```

### Check High-Alert Status Before Dispensing

```bash
curl -s "http://localhost:8081/v1/high-alert/check?rxnorm=11289" | jq .
```

---

## Common RxNorm Codes

| Drug | RxNorm | Category |
|------|--------|----------|
| Metformin | 6809 | Diabetes |
| Lisinopril | 29046 | Cardiovascular |
| Losartan | 52175 | Cardiovascular |
| Amlodipine | 17767 | Cardiovascular |
| Atorvastatin | 83367 | Cardiovascular |
| Warfarin | 11289 | Anticoagulant |
| Apixaban | 1364430 | Anticoagulant |
| Vancomycin | 11124 | Antibiotic |
| Morphine | 7052 | Pain |
| Acetaminophen | 161 | Pain |
