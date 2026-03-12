# KB-16 Lab Interpretation & Trending Service

**Port:** 8095 | **Version:** 1.0.0

KB-16 transforms raw laboratory results into clinically actionable intelligence with context-aware interpretation, patient-specific baselines, multi-window trending, and panel-level pattern recognition.

## Quick Start

```bash
# Start the service
cd kb-16-lab-interpretation
go run ./cmd/server

# Or with Docker
docker build -t kb-16:latest .
docker run -p 8095:8095 kb-16:latest
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8095` | HTTP server port |
| `ENVIRONMENT` | `development` | Environment (development/staging/production) |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `REDIS_URL` | - | Redis connection string |
| `KB2_SERVICE_URL` | `http://localhost:8086` | KB-2 Clinical Context service URL |
| `KB14_SERVICE_URL` | `http://localhost:8093` | KB-14 Care Navigator service URL |

## API Endpoints

### Health & Metrics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/metrics` | Prometheus metrics |

### Lab Results

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/results` | Store a lab result |
| GET | `/api/v1/results/:id` | Get result by ID |
| POST | `/api/v1/results/batch` | Store multiple results |
| GET | `/api/v1/patients/:patientId/results` | Get patient results |
| GET | `/api/v1/patients/:patientId/results/:code` | Get results by test code |

### Interpretation

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/interpret` | Interpret a lab result |
| POST | `/api/v1/interpret/batch` | Interpret multiple results |

### Trending

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/trending/:patientId/:code` | Get trend for a test |
| GET | `/api/v1/trending/:patientId/:code/multi` | Get multi-window trend |
| GET | `/api/v1/trending/:patientId` | Get all trends for patient |

### Baselines

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/baselines/:patientId` | Get patient baselines |
| GET | `/api/v1/baselines/:patientId/:code` | Get baseline for test |
| POST | `/api/v1/baselines/:patientId/:code` | Set manual baseline |
| POST | `/api/v1/baselines/:patientId/:code/calculate` | Calculate baseline |

### Panels

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/panels` | List panel definitions |
| GET | `/api/v1/panels/:type` | Get panel definition |
| POST | `/api/v1/panels/:patientId/assemble/:type` | Assemble panel from results |
| GET | `/api/v1/panels/:patientId/detect` | Detect available panels |

### Review Workflow

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/review/pending` | Get pending reviews |
| GET | `/api/v1/review/critical` | Get critical value queue |
| POST | `/api/v1/review/acknowledge` | Acknowledge result |
| POST | `/api/v1/review/complete` | Complete review |
| GET | `/api/v1/review/stats` | Review statistics |

### Visualization

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/charts/:patientId/:code` | Get chart data |
| GET | `/api/v1/sparklines/:patientId/:code` | Get sparkline data |
| GET | `/api/v1/dashboard/:patientId` | Get dashboard data |

### Reference Data

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/reference/tests` | List test definitions |
| GET | `/api/v1/reference/tests/:code` | Get test definition |

### FHIR R4

| Method | Path | Description |
|--------|------|-------------|
| GET | `/fhir/Observation` | Search FHIR Observations |
| GET | `/fhir/Observation/:id` | Get FHIR Observation |
| GET | `/fhir/DiagnosticReport/:patientId/:panelType` | Get FHIR DiagnosticReport |

## Example Usage

### Store a Lab Result

```bash
curl -X POST http://localhost:8095/api/v1/results \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-123",
    "code": "2823-3",
    "name": "Potassium",
    "value_numeric": 6.2,
    "unit": "mEq/L",
    "collected_at": "2025-01-15T10:30:00Z"
  }'
```

### Interpret a Result

```bash
curl -X POST http://localhost:8095/api/v1/interpret \
  -H "Content-Type: application/json" \
  -d '{
    "result": {
      "patient_id": "patient-123",
      "code": "2823-3",
      "name": "Potassium",
      "value_numeric": 6.2,
      "unit": "mEq/L",
      "collected_at": "2025-01-15T10:30:00Z"
    },
    "patient_context": {
      "age": 65,
      "sex": "male"
    }
  }'
```

### Get Trend Analysis

```bash
curl "http://localhost:8095/api/v1/trending/patient-123/2823-3?window=30d"
```

### Assemble a Panel

```bash
curl -X POST "http://localhost:8095/api/v1/panels/patient-123/assemble/BMP?lookback_days=7"
```

## Supported Tests (40+ LOINC Codes)

### Chemistry
- **Sodium** (2951-2): Normal 136-145 mEq/L
- **Potassium** (2823-3): Normal 3.5-5.0 mEq/L
- **Chloride** (2075-0): Normal 98-106 mEq/L
- **CO2** (1963-8): Normal 22-29 mEq/L
- **BUN** (3094-0): Normal 7-20 mg/dL
- **Creatinine** (2160-0): Normal 0.7-1.3 mg/dL
- **Glucose** (2345-7): Normal 70-100 mg/dL
- **Calcium** (17861-6): Normal 8.5-10.5 mg/dL
- **Magnesium** (19123-9): Normal 1.7-2.2 mg/dL
- **Phosphorus** (2777-1): Normal 2.5-4.5 mg/dL

### Hematology
- **WBC** (6690-2): Normal 4.5-11.0 K/uL
- **RBC** (789-8): Normal 4.7-6.1 M/uL (male), 4.2-5.4 M/uL (female)
- **Hemoglobin** (718-7): Normal 14-18 g/dL (male), 12-16 g/dL (female)
- **Hematocrit** (4544-3): Normal 40-54% (male), 36-48% (female)
- **Platelets** (777-3): Normal 150-400 K/uL
- **MCV** (787-2): Normal 80-100 fL
- **MCH** (785-6): Normal 27-31 pg
- **MCHC** (786-4): Normal 32-36 g/dL

### Coagulation
- **PT** (5902-2): Normal 11-15 seconds
- **INR** (34714-6): Normal 0.9-1.1
- **PTT** (5945-1): Normal 25-35 seconds

### Cardiac
- **Troponin I** (49563-0): Normal <0.04 ng/mL
- **BNP** (30934-4): Normal <100 pg/mL
- **CK-MB** (13969-1): Normal <5 ng/mL

### Lipids
- **Total Cholesterol** (2093-3): Normal <200 mg/dL
- **Triglycerides** (2571-8): Normal <150 mg/dL
- **HDL** (2085-9): Normal >40 mg/dL
- **LDL** (13457-7): Normal <100 mg/dL

### Liver Function
- **AST** (1920-8): Normal 10-40 U/L
- **ALT** (1742-6): Normal 7-56 U/L
- **ALP** (6768-6): Normal 44-147 U/L
- **Total Bilirubin** (1975-2): Normal 0.1-1.2 mg/dL
- **Albumin** (1751-7): Normal 3.5-5.0 g/dL

### Thyroid
- **TSH** (3016-3): Normal 0.4-4.0 mIU/L
- **Free T4** (3024-7): Normal 0.8-1.8 ng/dL
- **T3** (3053-6): Normal 80-200 ng/dL

### Inflammatory
- **CRP** (1988-5): Normal <1.0 mg/L
- **ESR** (30341-2): Normal 0-20 mm/hr
- **Procalcitonin** (75241-0): Normal <0.25 ng/mL
- **Lactate** (2524-7): Normal 0.5-2.0 mmol/L

## Panel Types

| Type | Name | Components |
|------|------|------------|
| BMP | Basic Metabolic Panel | Na, K, Cl, CO2, BUN, Cr, Glucose, Ca |
| CMP | Comprehensive Metabolic Panel | BMP + AST, ALT, ALP, Bili, Albumin, TP |
| CBC | Complete Blood Count | WBC, RBC, Hgb, Hct, Plt, MCV, MCH, MCHC |
| LFT | Liver Function Tests | AST, ALT, ALP, Bili, Albumin |
| LIPID | Lipid Panel | TC, TG, HDL, LDL |
| RENAL | Renal Function Panel | Cr, BUN, eGFR, Microalbumin |

## Critical/Panic Values

These values trigger immediate alerts and KB-14 task creation:

| Test | Panic Low | Panic High |
|------|-----------|------------|
| Potassium | <2.5 mEq/L | >6.5 mEq/L |
| Sodium | <120 mEq/L | >160 mEq/L |
| Glucose | <40 mg/dL | >500 mg/dL |
| Hemoglobin | <5 g/dL | - |
| Platelets | <20 K/uL | - |
| INR | - | >8.0 |
| Lactate | - | >7.0 mmol/L |

## Delta Checking

Significant changes from previous results trigger alerts:

| Test | Threshold | Window |
|------|-----------|--------|
| Hemoglobin | >2 g/dL decrease | 24 hours |
| Creatinine | >50% increase | 48 hours |
| Platelets | >50% decrease | 24 hours |
| Potassium | >1.0 mEq/L change | 24 hours |
| Sodium | >8 mEq/L change | 24 hours |

## Integration

### KB-2 Clinical Context
KB-16 calls KB-2 to get patient context (age, sex, conditions, medications) for context-aware interpretation.

### KB-14 Care Navigator
KB-16 creates tasks in KB-14 for:
- Panic/critical lab values (30-min SLA)
- Significant delta changes (2-hour SLA)

## Database Schema

The service uses PostgreSQL with the following tables:
- `lab_results` - Stored lab results
- `interpretations` - Result interpretations
- `patient_baselines` - Patient-specific baselines
- `result_reviews` - Review workflow tracking
- `audit_log` - Audit trail

## Metrics

Prometheus metrics available at `/metrics`:
- `kb16_http_requests_total` - HTTP request counter
- `kb16_http_request_duration_seconds` - Request latency histogram
- `kb16_critical_values_total` - Critical value counter
- `kb16_panic_values_total` - Panic value counter
- `kb16_interpretations_total` - Interpretation counter
