# KB-4: Patient Safety Service

**Comprehensive Medication Safety Checking**

## Why Patient Safety Service is Critical

| Area | Impact |
|------|--------|
| **Medication Errors** | 1.5 million people affected annually in US |
| **Black Box Warnings** | FDA's strongest warning - legal requirement |
| **Special Populations** | Pediatric, geriatric, pregnancy, lactation |
| **High-Alert Drugs** | Extra validation for dangerous medications |

## Features

### Safety Check Types

| Type | Description | Severity |
|------|-------------|----------|
| **Black Box Warning** | FDA's strongest warning | HIGH |
| **Contraindication** | Absolute/relative contraindications | CRITICAL/HIGH |
| **Age Limit** | Minimum/maximum age restrictions | CRITICAL |
| **Dose Limit** | Max single/daily/cumulative | HIGH |
| **Pregnancy** | FDA category, teratogenicity | CRITICAL |
| **Lactation** | Milk transfer, infant effects | CRITICAL/HIGH |
| **High-Alert** | ISMP high-alert medications | MODERATE |
| **Beers Criteria** | Geriatric inappropriate drugs | MODERATE |
| **Anticholinergic** | ACB score calculation | MODERATE |
| **Lab Required** | Required monitoring | LOW |

### Severity Levels

| Level | Action | Override |
|-------|--------|----------|
| **CRITICAL** | Block prescribing | No |
| **HIGH** | Require acknowledgment | Limited |
| **MODERATE** | Caution advised | Yes |
| **LOW** | Informational | Yes |

## API Endpoints

### Main Safety Check
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/check` | Comprehensive safety check |
| POST | `/api/v1/check/comprehensive` | Full check with all options |

### Black Box Warnings
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/blackbox?rxnorm=` | Get black box warning |
| GET | `/api/v1/blackbox/list` | List all black box drugs |
| GET | `/api/v1/blackbox/search?category=` | Search by risk category |

### Contraindications
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/contraindications?rxnorm=` | Get contraindications |
| POST | `/api/v1/contraindications/check` | Check against patient |

### Dose Limits
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/limits/dose?rxnorm=` | Get dose limits |
| GET | `/api/v1/limits/age?rxnorm=` | Get age limits |
| POST | `/api/v1/limits/validate` | Validate proposed dose |

### Pregnancy/Lactation
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/pregnancy?rxnorm=` | Pregnancy safety info |
| GET | `/api/v1/lactation?rxnorm=` | Lactation safety info |

### High-Alert Medications
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/high-alert?rxnorm=` | Check high-alert status |
| GET | `/api/v1/high-alert/list` | List all high-alert drugs |

### Beers Criteria
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/beers?rxnorm=` | Get Beers criteria info |
| POST | `/api/v1/beers/check` | Check medication list |

### Anticholinergic Burden
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/anticholinergic?rxnorm=` | Get ACB score |
| POST | `/api/v1/anticholinergic/burden` | Calculate total burden |

### Lab Requirements
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/labs?rxnorm=` | Get required labs |

## Built-in Safety Data

### Black Box Warnings (9)

| Drug | Risk Categories | REMS |
|------|-----------------|------|
| **Oxycodone** | Addiction, Respiratory Depression, Neonatal Withdrawal | No |
| **Morphine** | Addiction, Respiratory Depression | No |
| **Ciprofloxacin** | Tendon Rupture, Neuropathy, CNS Effects | No |
| **Sertraline** | Suicidality (<25 years) | No |
| **Liraglutide** | Thyroid C-Cell Tumors | No |
| **Clozapine** | Neutropenia, Myocarditis | **Yes** |
| **Isotretinoin** | Teratogenicity | **Yes (iPLEDGE)** |
| **Warfarin** | Bleeding | No |
| **Methotrexate** | Teratogenicity, Bone Marrow | No |

### Pregnancy Category X (Contraindicated)

| Drug | Teratogenic Effects |
|------|---------------------|
| **Isotretinoin** | Craniofacial, CNS, Cardiac defects |
| **Warfarin** | Warfarin embryopathy |
| **Methotrexate** | Aminopterin syndrome |
| **ACE Inhibitors** | Renal dysgenesis, oligohydramnios |

### High-Alert Medications (8)

| Drug | Category | Requirements |
|------|----------|--------------|
| Warfarin | Anticoagulants | Double-check, INR |
| Enoxaparin | Anticoagulants | Double-check, Weight-based |
| Heparin | Anticoagulants | Double-check, Smart pump |
| Insulin Glargine | Insulin | Double-check, No IV |
| Insulin Regular | Insulin | Double-check, Smart pump |
| Morphine | Opioids | Double-check, Smart pump |
| Oxycodone | Opioids | PMP check, Naloxone |
| Potassium Chloride IV | Electrolytes | Double-check, Smart pump |

### Beers Criteria (Elderly PIMs)

| Drug | Category | Concern |
|------|----------|---------|
| Diphenhydramine | AVOID | Highly anticholinergic |
| Alprazolam | AVOID | Fall risk, cognitive impairment |
| Ibuprofen | AVOID | GI bleeding, AKI |
| Oxybutynin | AVOID | Cognitive decline |
| Glyburide | AVOID | Prolonged hypoglycemia |

### Anticholinergic Burden Scores

| Score | Risk Level | Drugs |
|-------|------------|-------|
| **3 (High)** | Significant cognitive risk | Diphenhydramine, Oxybutynin, Amitriptyline |
| **2 (Moderate)** | Monitor for effects | Cyclobenzaprine |
| **1 (Low)** | Minimal concern | Furosemide, Metoprolol |

## Example: Comprehensive Safety Check

```bash
curl -X POST http://localhost:8083/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{
    "drug": {
      "rxnormCode": "7804",
      "drugName": "Oxycodone"
    },
    "proposedDose": 20,
    "doseUnit": "mg",
    "frequency": "Q4H",
    "patient": {
      "ageYears": 72,
      "gender": "F",
      "isPregnant": false,
      "diagnoses": [
        {"code": "G47.0", "display": "Insomnia"}
      ]
    }
  }'
```

Response:
```json
{
  "safe": false,
  "requiresAction": true,
  "blockPrescribing": false,
  "criticalAlerts": 0,
  "highAlerts": 2,
  "totalAlerts": 3,
  "alerts": [
    {
      "type": "BLACK_BOX_WARNING",
      "severity": "HIGH",
      "title": "Addiction, Abuse, and Misuse",
      "requiresAcknowledgment": true
    },
    {
      "type": "DOSE_LIMIT",
      "severity": "HIGH",
      "title": "Exceeds Maximum Single Dose",
      "message": "Proposed 20mg exceeds geriatric max 15mg"
    },
    {
      "type": "BEERS_CRITERIA",
      "severity": "MODERATE",
      "title": "Beers Criteria: Opioids",
      "message": "Increased fall risk in elderly"
    }
  ],
  "isHighAlertDrug": true
}
```

## File Structure

```
kb4-patient-safety-service/
├── cmd/server/main.go              (850 lines)   HTTP server
├── pkg/safety/
│   ├── service.go                  (850 lines)   Core types, checking logic
│   └── data.go                     (650 lines)   Built-in safety data
├── Dockerfile
├── go.mod
└── README.md
```

## Integration Points

### Upstream
- **KB-7 Terminology**: Drug identification
- **KB-2 Patient Context**: Patient diagnoses, allergies

### Downstream
- **Medication Advisor Engine**: Safety screening
- **CPOE (KB-12)**: Pre-order safety check
- **KB-5 DDI Service**: Complementary safety
- **Pharmacy**: Dispensing verification

## Quick Start

```bash
# Docker
docker build -t kb4-patient-safety-service .
docker run -p 8083:8083 kb4-patient-safety-service

# Local
go run cmd/server/main.go
```

## Clinical Decision Support Integration

The service supports tiered alerting:

1. **CRITICAL**: Block prescribing (pregnancy category X, absolute contraindications)
2. **HIGH**: Require acknowledgment (black box, relative contraindications)
3. **MODERATE**: Caution advised (Beers, high-alert)
4. **LOW**: Informational (lab requirements)

## License

Proprietary - Healthcare Platform
