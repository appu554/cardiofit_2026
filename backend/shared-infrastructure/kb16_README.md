# KB-16: Lab Interpretation & Trending Service

**Clinical Knowledge Platform - Laboratory Result Intelligence**

KB-16 provides clinical interpretation, trending analysis, and visualization for laboratory results. It transforms raw lab values into clinically actionable intelligence with context-aware interpretation, patient-specific baselines, and panel-level pattern recognition.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                   KB-16 Lab Interpretation & Trending                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    INTERPRETATION ENGINE                             │    │
│  │  • Critical Value Detection      • Abnormal Classification          │    │
│  │  • Delta Check Logic             • Panic Value Alerts               │    │
│  │  • Age/Sex Reference Ranges      • Clinical Significance Scoring    │    │
│  │  • Interpretive Comments         • Recommendations                  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    TRENDING & ANALYSIS ENGINE                        │    │
│  │  • Multi-Window Trending (7d, 30d, 90d, 1yr)                        │    │
│  │  • Trajectory Detection (improving/stable/worsening/volatile)       │    │
│  │  • Rate of Change Calculation     • Predictive Trending             │    │
│  │  • Statistical Analysis           • Linear Regression               │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    BASELINE TRACKER                                  │    │
│  │  • Patient-Specific Baselines     • Standard Deviation Bands        │    │
│  │  • Baseline Deviation Alerts      • Stable Period Detection         │    │
│  │  • Manual & Calculated Baselines  • Outlier Exclusion               │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    PANEL MANAGEMENT                                  │    │
│  │  • Panel Definitions (BMP, CMP, CBC, LFT, Lipid, Thyroid...)       │    │
│  │  • Panel Assembly & Completeness  • Calculated Values               │    │
│  │  • Pattern Detection              • Panel-Level Interpretation      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    REVIEW & WORKFLOW                                 │    │
│  │  • Acknowledgment Tracking        • Critical Value Queue            │    │
│  │  • Review Status Management       • Action Tracking                 │    │
│  │  • KB-14 Task Integration         • Compliance Metrics              │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    VISUALIZATION & EXPORT                            │    │
│  │  • Chart-Ready Data Structures    • Sparkline Data                  │    │
│  │  • FHIR R4 Observation/Report     • Comparison Views                │    │
│  │  • Multi-Test Charts              • Panel Visualizations            │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Features

### 1. Clinical Interpretation

| Capability | Description |
|------------|-------------|
| **Critical Values** | Immediate detection of panic/critical values with alerts |
| **Abnormal Classification** | LOW, HIGH, CRITICAL_LOW, CRITICAL_HIGH, PANIC |
| **Delta Checking** | Configurable thresholds for significant changes |
| **Age/Sex Ranges** | Reference ranges adjusted for demographics |
| **Clinical Comments** | Auto-generated interpretive comments |
| **Recommendations** | Context-aware clinical recommendations |

### 2. Trending Analysis

| Window | Use Case |
|--------|----------|
| 7 days | Acute changes, hospital course |
| 30 days | Recent trends, medication effects |
| 90 days | Chronic condition monitoring |
| 1 year | Long-term disease progression |

**Trajectory Detection:**
- IMPROVING - Moving toward normal range
- WORSENING - Moving away from normal range
- STABLE - Within consistent range
- VOLATILE - Significant variability

### 3. Panel Intelligence

| Panel | Components | Patterns Detected |
|-------|------------|-------------------|
| BMP | Na, K, Cl, CO2, BUN, Cr, Glu, Ca | Anion gap, AKI, electrolyte disorders |
| CMP | BMP + LFTs + Protein | Hepatocellular vs cholestatic injury |
| CBC | WBC, RBC, Hgb, Hct, Plt, MCV | Pancytopenia, anemia classification |
| LFT | AST, ALT, ALP, Bilirubin, Albumin | Liver injury patterns |
| Lipid | TC, TG, HDL, LDL | Cardiovascular risk |
| Renal | Cr, BUN, eGFR, UACR | CKD staging, AKI |

### 4. Baseline Tracking

- **Calculated Baselines**: Derived from stable historical values
- **Manual Baselines**: Clinician-set reference points
- **Deviation Alerts**: Notification when results deviate > 2 SD from baseline
- **Outlier Exclusion**: IQR-based outlier removal for baseline calculation

### 5. Review Workflow

```
Result Received
      │
      ▼
┌──────────────┐
│   PENDING    │ ◄─── Critical results auto-queued
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ ACKNOWLEDGED │ ◄─── Provider sees result (timestamp captured)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   REVIEWED   │ ◄─── Clinical review with notes
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  ACTIONED    │ ◄─── Follow-up action taken (KB-14 task created)
└──────────────┘
```

## Critical Value Rules

### Panic Values (Immediate Notification)

| Test | Panic Low | Panic High |
|------|-----------|------------|
| Potassium | < 2.5 mEq/L | > 6.5 mEq/L |
| Sodium | < 120 mEq/L | > 160 mEq/L |
| Glucose | < 40 mg/dL | > 500 mg/dL |
| Hemoglobin | < 5.0 g/dL | - |
| Platelets | < 20,000 /uL | - |
| INR | - | > 8.0 |
| Lactate | - | > 7.0 mmol/L |

### Critical Values (30-Minute Notification)

| Test | Critical Low | Critical High |
|------|--------------|---------------|
| Potassium | < 3.0 mEq/L | > 6.0 mEq/L |
| Sodium | < 125 mEq/L | > 155 mEq/L |
| Glucose | < 50 mg/dL | > 400 mg/dL |
| Hemoglobin | < 7.0 g/dL | > 20.0 g/dL |
| Platelets | < 50,000 /uL | > 1,000,000 /uL |
| Troponin | - | > 0.04 ng/mL |
| Creatinine | - | > 10.0 mg/dL |

### Delta Check Thresholds

| Test | Threshold | Time Window |
|------|-----------|-------------|
| Hemoglobin | > 2.0 g/dL decrease | 24 hours |
| Creatinine | > 50% increase | 48 hours |
| Platelets | > 50% decrease | 24 hours |
| Potassium | > 1.0 mEq/L change | 24 hours |
| Sodium | > 8 mEq/L change | 24 hours |

## API Reference

### Results Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/results` | Store a lab result |
| GET | `/api/v1/results/{id}` | Get result by ID |
| POST | `/api/v1/results/batch` | Store multiple results |
| GET | `/api/v1/patients/{id}/results` | Get patient results |
| GET | `/api/v1/patients/{id}/results/{code}` | Get patient results for specific test |

### Interpretation

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/interpret` | Interpret a single result |
| POST | `/api/v1/interpret/batch` | Interpret multiple results |

### Trending

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/trending/{patientId}/{code}` | Get trend for a test |
| GET | `/api/v1/trending/{patientId}/{code}/multi` | Multi-window trend |
| GET | `/api/v1/trending/{patientId}` | All trends for patient |

### Baselines

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/baselines/{patientId}` | Get patient baselines |
| GET | `/api/v1/baselines/{patientId}/{code}` | Get baseline for test |
| POST | `/api/v1/baselines/{patientId}/{code}` | Set manual baseline |
| POST | `/api/v1/baselines/{patientId}/{code}/calculate` | Calculate baseline |

### Panels

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/panels` | List panel definitions |
| GET | `/api/v1/panels/{type}` | Get panel definition |
| POST | `/api/v1/panels/{patientId}/assemble/{type}` | Assemble panel from results |
| GET | `/api/v1/panels/{patientId}/detect` | Detect available panels |

### Review Workflow

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/review/pending` | Get pending reviews |
| GET | `/api/v1/review/critical` | Get critical value queue |
| POST | `/api/v1/review/acknowledge` | Acknowledge result |
| POST | `/api/v1/review/complete` | Complete review |
| GET | `/api/v1/review/stats` | Review statistics |

### Visualization

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/charts/{patientId}/{code}` | Get chart data |
| GET | `/api/v1/sparklines/{patientId}/{code}` | Get sparkline data |
| GET | `/api/v1/dashboard/{patientId}` | Get dashboard data |

### FHIR

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/fhir/Observation` | Search observations |
| GET | `/fhir/Observation/{id}` | Get observation by ID |
| GET | `/fhir/DiagnosticReport/{patientId}/{panelType}` | Get diagnostic report |

### Reference Data

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/reference/tests` | List all test definitions |
| GET | `/api/v1/reference/tests/{code}` | Get test definition |

## Integration Points

### KB-2 Patient Context Service

```
KB-2 (Patient Context)
  │
  │  GET /api/v1/patients/{id}/labs
  │
  ▼
KB-16 (Lab Interpretation)
  │
  │  Interpret, trend, analyze
  │
  ▼
Interpreted Results
```

### KB-14 Care Navigator

```
KB-16 Critical Result Detected
  │
  │  POST /api/v1/tasks/from-lab-result
  │
  ▼
KB-14 Task Created
  │
  │  CRITICAL_LAB_REVIEW task
  │  Assigned to ordering provider
  │  1-hour SLA
  │
  ▼
Provider Review → Action
```

## Example Interpretation Response

```json
{
  "result": {
    "id": "result-123",
    "code": "2823-3",
    "name": "Potassium",
    "valueNumeric": 6.2,
    "unit": "mEq/L",
    "interpretation": {
      "flag": "CRITICAL_HIGH",
      "severity": "HIGH",
      "isCritical": true,
      "isPanic": false,
      "requiresAction": true,
      "deviationPercent": 24.0,
      "deviationDirection": "above",
      "deltaCheck": {
        "previousValue": 4.8,
        "percentChange": 29.2,
        "exceedsThreshold": true,
        "alert": "Significant change: 29.2% (4.8 → 6.2) in 18.5 hours"
      },
      "clinicalComment": "Critical high K at 6.2 mEq/L - notify provider",
      "recommendations": [
        "Notify ordering provider within 30 minutes",
        "Obtain ECG to evaluate for hyperkalemia changes",
        "Review medications for potassium-sparing drugs"
      ]
    }
  }
}
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | HTTP server port | 8088 |
| KB2_SERVICE_URL | KB-2 Patient Context URL | http://localhost:8082 |
| KB14_SERVICE_URL | KB-14 Care Navigator URL | http://localhost:8087 |

## Running the Service

### Local Development

```bash
go run ./cmd/server
```

### Docker

```bash
docker build -t kb16-lab-interpretation .
docker run -p 8088:8088 \
  -e KB2_SERVICE_URL=http://kb2:8082 \
  -e KB14_SERVICE_URL=http://kb14:8087 \
  kb16-lab-interpretation
```

### Health Check

```bash
curl http://localhost:8088/health
```

## Project Structure

```
kb16-lab-interpretation/
├── cmd/server/
│   └── main.go              # HTTP server with 40+ endpoints
├── pkg/
│   ├── types/
│   │   └── types.go         # Core types and models
│   ├── reference/
│   │   └── database.go      # Test definitions, reference ranges
│   ├── interpretation/
│   │   └── engine.go        # Clinical interpretation engine
│   ├── trending/
│   │   └── engine.go        # Time-series trending analysis
│   ├── baseline/
│   │   └── tracker.go       # Patient-specific baselines
│   ├── panels/
│   │   └── manager.go       # Panel assembly and interpretation
│   ├── review/
│   │   └── service.go       # Review workflow management
│   ├── visualization/
│   │   ├── exporter.go      # Chart data generation
│   │   └── service.go       # Visualization service
│   ├── fhir/
│   │   └── observation.go   # FHIR R4 mapping
│   ├── store/
│   │   └── result_store.go  # In-memory result storage
│   └── integration/
│       └── clients.go       # KB-2, KB-14 clients
├── Dockerfile
├── go.mod
└── README.md
```

## Test Coverage

The service includes comprehensive tests for:
- Interpretation engine (critical values, delta checks)
- Trending calculations (statistics, trajectory detection)
- Baseline tracking (calculation, deviation detection)
- Panel assembly and interpretation
- FHIR mapping (to/from)

## Reference Database

The service includes 40+ pre-configured lab tests with:
- LOINC codes
- Reference ranges (default, age-based, sex-based)
- Critical/panic values
- Delta check thresholds
- Panel memberships
- Trending configurations

### Supported Test Categories

- **Chemistry**: Electrolytes, metabolic panel components
- **Hematology**: CBC components, indices
- **Coagulation**: PT, INR, PTT
- **Cardiac**: Troponin, BNP
- **Lipids**: Cholesterol, triglycerides
- **Thyroid**: TSH, T4
- **Renal**: Creatinine, BUN, eGFR
- **Inflammatory**: CRP, ESR

## Value Proposition

### Without KB-16

- Raw lab values without clinical context
- Manual identification of critical values
- No trending or trajectory analysis
- No patient-specific baselines
- Disconnected review workflows

### With KB-16

- **Immediate Intelligence**: Critical values flagged instantly
- **Clinical Context**: Interpretive comments and recommendations
- **Trend Analysis**: See patterns across multiple time windows
- **Patient Baselines**: Compare to individual's normal, not just population
- **Panel Patterns**: Recognize clinical syndromes (AKI, hepatitis patterns)
- **Review Tracking**: Ensure every critical result is acknowledged
- **Task Integration**: Critical results create KB-14 tasks automatically

---

**KB-16 transforms lab results from data points into clinical intelligence.**
