# KB-1: Drug Dosing Rules Service

**Comprehensive Drug Dosing Calculation and Validation**

## Why Drug Dosing Service is Critical

| Area | Impact |
|------|--------|
| **Patient Safety** | Prevent dosing errors - leading cause of ADEs |
| **Special Populations** | Pediatric, geriatric, renal, hepatic adjustments |
| **High-Alert Drugs** | Extra validation for dangerous medications |
| **Clinical Workflow** | Automated dose calculations save time |

## Features

### Dosing Methods

| Method | Description | Use Case |
|--------|-------------|----------|
| **Fixed** | Standard doses | Most oral medications |
| **Weight-Based** | mg/kg calculation | Antibiotics, anticoagulants |
| **BSA-Based** | mg/m² calculation | Chemotherapy |
| **Age-Based** | Pediatric/geriatric | Special populations |
| **Renal-Adjusted** | CrCl/eGFR-based | Renally cleared drugs |
| **Hepatic-Adjusted** | Child-Pugh based | Hepatically cleared drugs |
| **Titration** | Step-up schedules | Metoprolol, lisinopril |

### Patient Parameter Calculations

| Parameter | Formula | Use |
|-----------|---------|-----|
| **BSA** | Mosteller | Chemotherapy dosing |
| **IBW** | Devine | Aminoglycosides |
| **AdjBW** | IBW + 0.4(ABW-IBW) | Obese patients |
| **CrCl** | Cockcroft-Gault | Renal dose adjustment |
| **eGFR** | CKD-EPI 2021 | Renal staging |

### Safety Features

- **Max Single Dose** validation
- **Max Daily Dose** validation
- **High-Alert Drug** warnings
- **Narrow Therapeutic Index** alerts
- **Black Box Warning** flags
- **Monitoring Requirements**

## API Endpoints

### Dose Calculation
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/calculate` | Full dose calculation |
| POST | `/api/v1/calculate/weight-based` | Weight-based calculation |
| POST | `/api/v1/calculate/bsa-based` | BSA-based calculation |
| POST | `/api/v1/calculate/pediatric` | Pediatric dosing |
| POST | `/api/v1/calculate/renal` | Renal-adjusted dose |

### Dose Validation
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/validate` | Validate proposed dose |
| GET | `/api/v1/validate/max-dose?rxnorm=` | Get max dose limits |

### Patient Parameters
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/patient/bsa` | Calculate BSA |
| POST | `/api/v1/patient/ibw` | Calculate IBW |
| POST | `/api/v1/patient/crcl` | Calculate CrCl |
| POST | `/api/v1/patient/egfr` | Calculate eGFR |

### Dosing Rules
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/rules` | List all rules |
| GET | `/api/v1/rules/{rxnorm}` | Get specific rule |
| GET | `/api/v1/rules/search?q=` | Search rules |

### Adjustments
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/adjustments/renal?rxnorm=` | Renal dosing info |
| GET | `/api/v1/adjustments/hepatic?rxnorm=` | Hepatic dosing info |
| GET | `/api/v1/adjustments/age?rxnorm=` | Age-based info |

### High-Alert
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/high-alert/check?rxnorm=` | Check high-alert status |

## Built-in Drug Rules

### Diabetes
| Drug | Method | Max Daily | Special |
|------|--------|-----------|---------|
| Metformin | Fixed | 2000mg | Renal adjustment |
| Empagliflozin | Fixed | 25mg | eGFR threshold |
| Liraglutide | Titration | 1.8mg | Black box thyroid |
| Insulin Glargine | Weight-based | 100 units | High-alert |

### Cardiovascular
| Drug | Method | Max Daily | Special |
|------|--------|-----------|---------|
| Lisinopril | Fixed/Titration | 40mg | Renal, start low |
| Losartan | Fixed | 100mg | Hepatic adjustment |
| Metoprolol | Titration | 400mg | HF titration |
| Carvedilol | Titration | 100mg | Take with food |
| Atorvastatin | Fixed | 80mg | Hepatic caution |
| Furosemide | Fixed | 600mg | Higher in CKD |
| Spironolactone | Fixed | 400mg | K+ monitoring |

### Anticoagulants
| Drug | Method | Special |
|------|--------|---------|
| Warfarin | Fixed | High-alert, narrow TI |
| Apixaban | Fixed | Renal criteria |
| Enoxaparin | Weight-based | 1mg/kg, high-alert |
| Heparin | Weight-based | High-alert, narrow TI |

### Antibiotics
| Drug | Method | Special |
|------|--------|---------|
| Amoxicillin | Fixed/Weight | Pediatric dosing |
| Vancomycin | Weight-based | Narrow TI, levels |
| Gentamicin | Weight-based | High-alert, levels |
| Ciprofloxacin | Fixed | Black box tendon |

### Pain
| Drug | Method | Special |
|------|--------|---------|
| Acetaminophen | Fixed | Max 4g/day (3g in elderly) |
| Ibuprofen | Fixed | Beers list, renal caution |
| Morphine | Fixed | High-alert, black box |
| Oxycodone | Fixed | High-alert, black box |

## Example: Calculate Dose

```bash
curl -X POST http://localhost:8081/api/v1/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "rxnormCode": "11124",
    "age": 65,
    "gender": "M",
    "weightKg": 80,
    "heightCm": 178,
    "serumCreatinine": 1.2,
    "eGFR": 58
  }'
```

Response:
```json
{
  "success": true,
  "drugName": "Vancomycin",
  "recommendedDose": 1200,
  "doseUnit": "mg",
  "frequency": "Q12H",
  "dosingMethod": "WEIGHT_BASED",
  "calculationBasis": "Weight: 80.0 kg",
  "renalAdjustment": {
    "applied": true,
    "eGFR": 58,
    "notes": "Consider Q24H dosing"
  }
}
```

## Example: Validate Dose

```bash
curl -X POST http://localhost:8081/api/v1/validate \
  -H "Content-Type: application/json" \
  -d '{
    "rxnormCode": "7052",
    "proposedDose": 50,
    "frequency": "Q4H"
  }'
```

## File Structure

```
kb1-drug-dosing-service/
├── cmd/server/main.go              (750 lines)   HTTP server
├── pkg/dosing/
│   ├── service.go                  (1100 lines)  Core types, calculations
│   └── rules.go                    (900 lines)   24 built-in drug rules
├── test/service_test.go            (300 lines)
├── Dockerfile
├── go.mod
└── README.md
```

## Integration Points

### Upstream
- **KB-7 Terminology**: RxNorm code validation

### Downstream
- **Medication Advisor Engine**: Dose calculations
- **CPOE (KB-12)**: Dose validation at order entry
- **Pharmacy**: Dispensing verification
- **Clinical Documentation**: Dose recording

## Quick Start

```bash
# Docker
docker build -t kb1-drug-dosing-service .
docker run -p 8081:8081 kb1-drug-dosing-service

# Local
go run cmd/server/main.go
```

## License

Proprietary - Healthcare Platform
