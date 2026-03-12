# KB-8 Calculator Service: Atomic Pattern Architecture

## The Architectural Split

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CLINICAL KNOWLEDGE PLATFORM                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                    STREAM (FAST) - KB-8 CALCULATORS                   │  │
│  │                         Pattern: ATOMIC                               │  │
│  │                                                                       │  │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────┐   │  │
│  │  │   Flink     │    │    Go KB-8  │    │     CQL Engine          │   │  │
│  │  │  Streaming  │───▶│   Service   │───▶│  (Pure Math, No I/O)    │   │  │
│  │  │             │    │             │    │                         │   │  │
│  │  │  Receives:  │    │  Receives:  │    │  Receives:              │   │  │
│  │  │  Lab ADT    │    │  HTTP/gRPC  │    │  Parameters only        │   │  │
│  │  │  Vitals     │    │             │    │                         │   │  │
│  │  └─────────────┘    └─────────────┘    │  Returns:               │   │  │
│  │        │                   │           │  Scores + Tuples        │   │  │
│  │        │                   │           │                         │   │  │
│  │        │   Parameters      │           │  No FHIR queries        │   │  │
│  │        │   (values)        │           │  No database calls      │   │  │
│  │        ▼                   ▼           │  Millisecond latency    │   │  │
│  │  ┌─────────────────────────────────────┴─────────────────────────┐   │  │
│  │  │                    ClinicalCalculatorsCommon.cql               │   │  │
│  │  │                         Version 2.0.000                        │   │  │
│  │  │                                                                │   │  │
│  │  │  parameter "Creatinine" Decimal                                │   │  │
│  │  │  parameter "Age" Integer                                       │   │  │
│  │  │  parameter "Sex" String                                        │   │  │
│  │  │                                                                │   │  │
│  │  │  define "eGFR CKD-EPI 2021":                                   │   │  │
│  │  │    142.0 * Power(Min(Creatinine/0.9, 1.0), -0.302) * ...       │   │  │
│  │  └────────────────────────────────────────────────────────────────┘   │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                    SERVER (DEEP) - KB-9 CARE GAPS                     │  │
│  │                       Pattern: QUERY-BASED                            │  │
│  │                                                                       │  │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────┐   │  │
│  │  │   HAPI      │    │  CMS eCQM   │    │     CQL Engine          │   │  │
│  │  │   FHIR      │◀───│  Libraries  │◀───│  (Query-Based)          │   │  │
│  │  │   Server    │    │             │    │                         │   │  │
│  │  │             │    │  CMS122     │    │  Contains:              │   │  │
│  │  │  Patient    │    │  CMS165     │    │  [Observation: ...]     │   │  │
│  │  │  Repository │    │  CMS130     │    │  [Condition: ...]       │   │  │
│  │  └─────────────┘    └─────────────┘    │                         │   │  │
│  │        ▲                               │  Queries FHIR           │   │  │
│  │        │                               │  for longitudinal data  │   │  │
│  │        │    FHIR Queries               │                         │   │  │
│  │        └───────────────────────────────┴─────────────────────────┘   │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Data Flow: Phase 2 Context Assembly

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PHASE 2: CONTEXT ASSEMBLY                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Step 1: Context Integration Service fetches raw data                       │
│  ─────────────────────────────────────────────────────                      │
│                                                                             │
│  ┌─────────────────┐         ┌─────────────────┐                           │
│  │ Context Gateway │────────▶│   FHIR Server   │                           │
│  └─────────────────┘         └─────────────────┘                           │
│          │                            │                                     │
│          │  Fetches:                  │                                     │
│          │  - Patient demographics    │                                     │
│          │  - Recent labs             │                                     │
│          │  - Vital signs             │                                     │
│          │  - Active conditions       │                                     │
│          ▼                            │                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Raw Patient Data                              │   │
│  │  {                                                                   │   │
│  │    "creatinine": 1.8,                                                │   │
│  │    "age": 72,                                                        │   │
│  │    "sex": "male",                                                    │   │
│  │    "systolicBP": 95,                                                 │   │
│  │    "respiratoryRate": 24,                                            │   │
│  │    "gcs": 14,                                                        │   │
│  │    "hasCHF": true,                                                   │   │
│  │    "hasHypertension": true,                                          │   │
│  │    ...                                                               │   │
│  │  }                                                                   │   │
│  └──────────────────────────────────────┬──────────────────────────────┘   │
│                                         │                                   │
│  Step 2: Pass values to KB-8 (Atomic)   │                                   │
│  ─────────────────────────────────────  │                                   │
│                                         ▼                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        KB-8 Calculator Service                       │   │
│  │                                                                      │   │
│  │  POST /api/v1/calculate/batch                                        │   │
│  │  {                                                                   │   │
│  │    "parameters": {                                                   │   │
│  │      "Creatinine": 1.8,                                              │   │
│  │      "Age": 72,                                                      │   │
│  │      "Sex": "male",                                                  │   │
│  │      "SystolicBP": 95,                                               │   │
│  │      "RespiratoryRate": 24,                                          │   │
│  │      "GCS": 14,                                                      │   │
│  │      "HasCHF": true,                                                 │   │
│  │      "HasHypertension": true                                         │   │
│  │    }                                                                 │   │
│  │  }                                                                   │   │
│  │                                                                      │   │
│  │  ┌──────────────────────────────────────────────────────────────┐   │   │
│  │  │              CQL Evaluation (Pure Math)                       │   │   │
│  │  │                                                               │   │   │
│  │  │  "eGFR CKD-EPI 2021" → 42.5                                   │   │   │
│  │  │  "CKD Stage" → "G3b"                                          │   │   │
│  │  │  "Requires Renal Dose Adjustment" → true                      │   │   │
│  │  │  "qSOFA Total Score" → 2                                      │   │   │
│  │  │  "qSOFA Positive" → true                                      │   │   │
│  │  │  "CHA2DS2-VASc Total Score" → 4                               │   │   │
│  │  │  "Anticoagulation Recommended" → true                         │   │   │
│  │  │                                                               │   │   │
│  │  │  Latency: ~5ms (no I/O, just math)                            │   │   │
│  │  └──────────────────────────────────────────────────────────────┘   │   │
│  └──────────────────────────────────────┬──────────────────────────────┘   │
│                                         │                                   │
│  Step 3: Attach to CompleteContextPayload                                   │
│  ────────────────────────────────────────                                   │
│                                         ▼                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    CompleteContextPayload                            │   │
│  │  {                                                                   │   │
│  │    "patientId": "patient-123",                                       │   │
│  │    "rawData": { ... },                // Original FHIR data          │   │
│  │    "computedScores": {                // KB-8 enrichment             │   │
│  │      "egfr": 42.5,                                                   │   │
│  │      "ckdStage": "G3b",                                              │   │
│  │      "requiresRenalDoseAdjustment": true,                            │   │
│  │      "qsofaTotal": 2,                                                │   │
│  │      "qsofaPositive": true,                                          │   │
│  │      "cha2ds2vascTotal": 4,                                          │   │
│  │      "anticoagulationRecommended": true                              │   │
│  │    }                                                                 │   │
│  │  }                                                                   │   │
│  └──────────────────────────────────────┬──────────────────────────────┘   │
│                                         │                                   │
│                                         ▼                                   │
│                              To Phase 3 (Intelligence)                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Data Flow: Phase 3b Dose Calculation

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     PHASE 3B: DOSE CALCULATION                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Rust DoseCalculationEngine                        │   │
│  │                                                                      │   │
│  │  fn calculate_dose(context: &CompleteContextPayload, drug: &Drug) {  │   │
│  │                                                                      │   │
│  │      // eGFR already calculated in Phase 2 - just READ it            │   │
│  │      let egfr = context.computed_scores.egfr;  // 42.5               │   │
│  │      let needs_adjustment = context.computed_scores                  │   │
│  │                                   .requires_renal_dose_adjustment;   │   │
│  │                                                                      │   │
│  │      // NO CQL call here - value is pre-computed                     │   │
│  │      if needs_adjustment {                                           │   │
│  │          apply_renal_adjustment(&drug.dosing_rule, egfr)             │   │
│  │      }                                                               │   │
│  │  }                                                                   │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  KEY INSIGHT:                                                               │
│  ───────────                                                                │
│  Phase 3b does NOT call KB-8.                                               │
│  eGFR was calculated ONCE in Phase 2.                                       │
│  Phase 3b CONSUMES the pre-computed value.                                  │
│  "Calculate once, use everywhere."                                          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Data Flow: Flink Streaming (Real-time)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      FLINK STREAMING (REAL-TIME)                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │  Lab ADT    │    │   Flink     │    │    CQL      │    │   Alert     │  │
│  │  Stream     │───▶│   Window    │───▶│   Engine    │───▶│   Service   │  │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘  │
│                           │                   │                             │
│  New Creatinine           │                   │                             │
│  Result: 2.4 mg/dL        │                   │                             │
│                           ▼                   │                             │
│                    ┌─────────────────┐        │                             │
│                    │ Enrichment Join │        │                             │
│                    │                 │        │                             │
│                    │ Patient lookup: │        │                             │
│                    │ Age: 72         │        │                             │
│                    │ Sex: male       │        │                             │
│                    └────────┬────────┘        │                             │
│                             │                 │                             │
│                             ▼                 │                             │
│                    Parameters:                │                             │
│                    {                          │                             │
│                      Creatinine: 2.4,         │                             │
│                      Age: 72,                 │                             │
│                      Sex: "male"              │                             │
│                    }                          │                             │
│                             │                 │                             │
│                             │                 ▼                             │
│                             │    ┌─────────────────────────────────────┐   │
│                             │    │  CQL Atomic Evaluation               │   │
│                             └───▶│                                      │   │
│                                  │  Input: Creatinine=2.4, Age=72,     │   │
│                                  │         Sex="male"                   │   │
│                                  │                                      │   │
│                                  │  Output:                             │   │
│                                  │    eGFR: 31.2                        │   │
│                                  │    CKD Stage: G3b                    │   │
│                                  │    Requires Adjustment: true         │   │
│                                  │                                      │   │
│                                  │  Latency: ~2ms                       │   │
│                                  └──────────────────────┬──────────────┘   │
│                                                         │                   │
│                                                         ▼                   │
│                                              ┌─────────────────────┐        │
│                                              │  Alert Evaluation   │        │
│                                              │                     │        │
│                                              │  eGFR dropped from  │        │
│                                              │  42.5 → 31.2        │        │
│                                              │  (26% decline)      │        │
│                                              │                     │        │
│                                              │  TRIGGER AKI ALERT  │        │
│                                              └─────────────────────┘        │
│                                                                             │
│  KEY INSIGHT:                                                               │
│  ───────────                                                                │
│  Flink handles data gathering (windowing, enrichment).                      │
│  CQL handles pure calculation (eGFR formula).                               │
│  Separation enables millisecond-latency real-time alerts.                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Composability Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          COMPOSABILITY MODEL                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  WITHIN CQL (same library):                                                 │
│  ────────────────────────                                                   │
│                                                                             │
│  define "BMI":                                                              │
│    WeightKg / Power(HeightCm / 100.0, 2)                                    │
│                                                                             │
│  define "Obesity Adjustment for SOFA":                                      │
│    if "BMI" > 40 then                         // References BMI definition  │
│      "SOFA Total Score" + 1                   // Hypothetical adjustment    │
│    else                                                                     │
│      "SOFA Total Score"                                                     │
│                                                                             │
│  → CQL engine evaluates "BMI" ONCE, caches result                           │
│  → All definitions referencing "BMI" use cached value                       │
│  → No recalculation                                                         │
│                                                                             │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  ACROSS PHASES (via CompleteContextPayload):                                │
│  ───────────────────────────────────────────                                │
│                                                                             │
│  Phase 2:                                                                   │
│    KB-8 calculates eGFR = 42.5                                              │
│    Attaches to payload: computedScores.egfr = 42.5                          │
│                                                                             │
│  Phase 3a (Safety):                                                         │
│    Go CandidateBuilder reads payload.computedScores.egfr                    │
│    Uses for contraindication check (e.g., Metformin + eGFR < 30)            │
│    NO CQL call - value already computed                                     │
│                                                                             │
│  Phase 3b (Dosing):                                                         │
│    Rust DoseEngine reads payload.computedScores.egfr                        │
│    Uses for renal dose adjustment                                           │
│    NO CQL call - value already computed                                     │
│                                                                             │
│  Phase 3c (Scoring):                                                        │
│    Go ScoringEngine reads payload.computedScores.egfr                       │
│    Uses as ranking factor (prefer drugs not requiring adjustment)           │
│    NO CQL call - value already computed                                     │
│                                                                             │
│  → "Calculate once in Phase 2, consume everywhere downstream"               │
│  → eGFR is a FIRST-CLASS DATA TYPE in the payload                           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Summary: The Split Brain

| Engine | Pattern | Data Source | Latency | Use Case |
|--------|---------|-------------|---------|----------|
| **KB-8 Calculators** | ATOMIC | Caller provides values | ~5ms | Real-time scores, Flink streams |
| **KB-9 Care Gaps** | QUERY-BASED | CQL queries FHIR | ~200ms | Longitudinal measures, eCQM |

**KB-8 CQL contains:**
```cql
parameter "Creatinine" Decimal
parameter "Age" Integer
define "eGFR": 142.0 * Power(Min(Creatinine/0.9, 1.0), -0.302) * ...
```

**KB-9 CQL contains:**
```cql
define "Last Colonoscopy":
  Last([Procedure: "Colonoscopy"] P where P.status = 'completed' sort by performed)
```

Different patterns for different engines. Same CQL language. Clean separation.
