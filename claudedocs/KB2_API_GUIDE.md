# KB-2 Clinical Context Service API Guide

## Overview

KB-2 is the **Clinical Context Service** that transforms raw FHIR data into an enriched clinical execution context. It's split into two distinct phases:

| Component | Purpose | Port |
|-----------|---------|------|
| **KB-2A** | Data Assembly - Parse FHIR Bundle → PatientContext (NO intelligence) | 8082 |
| **KB-2B** | Intelligence Enrichment - Add phenotypes, risk scores, care gaps | 8082 |

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                         KB-2 ARCHITECTURE                                       │
├────────────────────────────────────────────────────────────────────────────────┤
│                                                                                │
│   FHIR Bundle ───▶ KB-2A (Assembly) ───▶ KnowledgeSnapshotBuilder ───▶ KB-2B   │
│                          │                      │           │          │       │
│                          ▼                      ▼           ▼          ▼       │
│                    PatientContext            KB-7        KB-8    Enriched      │
│                    (data only)           (Terminology) (Calcs)   Context       │
│                                                                                │
└────────────────────────────────────────────────────────────────────────────────┘
```

---

## Architecture Principles (CTO/CMO Spec)

### Critical Design Rules

1. **KB-2A: Pure Data Assembly**
   - Parses FHIR Bundle into normalized `PatientContext`
   - **NEVER** performs calculations or adds intelligence
   - Output is raw structured data only

2. **KnowledgeSnapshotBuilder: Pre-Computation**
   - Queries KB-7 (Terminology) and KB-8 (Calculators) using base PatientContext
   - Builds frozen `KnowledgeSnapshot` with pre-computed answers
   - Engines see **ANSWERS**, not questions

3. **KB-2B: Intelligence Enrichment**
   - Adds phenotypes, risk profiles, clinical flags, care gaps
   - Operates on PatientContext + KnowledgeSnapshot
   - Returns enriched `PatientContext`

4. **Frozen Contract**
   - Final `ClinicalExecutionContext` is **IMMUTABLE**
   - Engines receive pre-computed answers
   - **NO KB calls at execution time**

---

## KB-2A: Data Assembly API

### Purpose
Transform raw FHIR data into a normalized `PatientContext` structure without any clinical intelligence.

### Endpoint

```
POST /api/v1/context/build
```

### Request Body

```json
{
  "patient_id": "patient-123",
  "fhir_bundle": {
    "resourceType": "Bundle",
    "type": "collection",
    "entry": [
      {
        "resource": {
          "resourceType": "Patient",
          "id": "patient-123",
          "gender": "female",
          "birthDate": "1956-03-15"
        }
      },
      {
        "resource": {
          "resourceType": "Condition",
          "clinicalStatus": {"coding": [{"code": "active"}]},
          "code": {
            "coding": [
              {
                "system": "http://snomed.info/sct",
                "code": "49436004",
                "display": "Atrial fibrillation"
              }
            ]
          }
        }
      },
      {
        "resource": {
          "resourceType": "Observation",
          "code": {
            "coding": [
              {
                "system": "http://loinc.org",
                "code": "2160-0",
                "display": "Serum Creatinine"
              }
            ]
          },
          "valueQuantity": {
            "value": 1.4,
            "unit": "mg/dL"
          }
        }
      }
    ]
  }
}
```

### Response

```json
{
  "success": true,
  "data": {
    "context_id": "ctx-8e8850d8-dc62-4167-a1b2-3c4d5e6f7890",
    "patient_id": "patient-123",
    "processed_at": "2025-12-17T05:37:42Z",
    "cache_hit": false,
    "phenotypes": []
  },
  "meta": {
    "phenotypes_detected": 0,
    "cache_hit": false,
    "processing_time_ms": 45
  }
}
```

### What KB-2A Extracts

| FHIR Resource | PatientContext Field | Description |
|---------------|---------------------|-------------|
| Patient | `demographics` | Age, gender, birth date, region |
| Condition (active) | `activeConditions` | Normalized condition codes |
| MedicationRequest | `activeMedications` | Active prescriptions |
| Observation (lab) | `recentLabResults` | Lab values (last 90 days) |
| Observation (vital) | `recentVitalSigns` | Vitals (last 30 days) |
| Encounter | `recentEncounters` | Clinical encounters |
| AllergyIntolerance | `allergies` | Active allergies |

### PatientContext Output Structure

```go
type PatientContext struct {
    // KB-2A: Raw Assembly (NO intelligence)
    Demographics      PatientDemographics   `json:"demographics"`
    ActiveConditions  []ClinicalCondition   `json:"activeConditions"`
    ActiveMedications []Medication          `json:"activeMedications"`
    RecentLabResults  []LabResult           `json:"recentLabResults"`
    RecentVitalSigns  []VitalSign           `json:"recentVitalSigns"`
    RecentEncounters  []Encounter           `json:"recentEncounters"`
    Allergies         []Allergy             `json:"allergies"`

    // KB-2B: Intelligence (added later)
    RiskProfile       RiskProfile           `json:"riskProfile"`
    ClinicalSummary   ClinicalSummary       `json:"clinicalSummary"`
}
```

---

## KnowledgeSnapshotBuilder: Orchestration

### Purpose
Query KB-7 (Terminology) and KB-8 (Calculators) to pre-compute all knowledge base answers.

### Flow

```
PatientContext (from KB-2A)
         │
         ▼
┌─────────────────────────────┐
│  KnowledgeSnapshotBuilder   │
│         .Build()            │
└─────────────────────────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌────────┐
│  KB-7  │ │  KB-8  │
│Terminol│ │ Calcs  │
└────────┘ └────────┘
    │         │
    ▼         ▼
┌─────────────────────────────┐
│     KnowledgeSnapshot       │
│   (frozen pre-answers)      │
└─────────────────────────────┘
```

### KB-7 Queries (Terminology)

| Query | Purpose | Example |
|-------|---------|---------|
| `CheckMembership` | Is code in ValueSet? | Is 73211009 in "Diabetes" ValueSet? |
| `ExpandValueSet` | Get all codes in ValueSet | All codes in "ACE Inhibitor" |
| `ResolveCode` | Get display name | 73211009 → "Diabetes mellitus" |
| `GetRelevantValueSets` | ValueSets for conditions | Patient's relevant ValueSets |

### KB-8 Queries (Calculators)

| Calculator | When Computed | Input Required |
|------------|---------------|----------------|
| **eGFR** | Always | Creatinine, Age, Sex |
| **ASCVD** | Ages 40-79 | TC, HDL, SBP, DM, Smoker |
| **CHA₂DS₂-VASc** | AFib patients | Age, Sex, CHF/HTN/DM/Stroke/Vascular |
| **HAS-BLED** | AFib patients | Renal/Liver, HTN, Bleeding Hx, Age |
| **BMI** | Always | Weight, Height |
| **SOFA** | ICU patients | PaO2/FiO2, Platelets, Bilirubin, MAP, GCS, Cr |
| **qSOFA** | Sepsis screening | RR, SBP, Altered mentation |

### KnowledgeSnapshot Structure

```go
type KnowledgeSnapshot struct {
    // KB-7: Terminology
    Terminology TerminologySnapshot `json:"terminology"`

    // KB-8: Calculators
    Calculators CalculatorSnapshot `json:"calculators"`

    // KB-4: Safety
    Safety SafetySnapshot `json:"safety"`

    // KB-5: Drug Interactions
    Interactions InteractionSnapshot `json:"interactions"`

    // KB-6: Formulary
    Formulary FormularySnapshot `json:"formulary"`

    // KB-1: Dosing
    Dosing DosingSnapshot `json:"dosing"`

    // Metadata
    SnapshotTimestamp time.Time         `json:"snapshotTimestamp"`
    KBVersions        map[string]string `json:"kbVersions"`
}
```

---

## KB-2B: Intelligence Enrichment API

### Purpose
Enrich PatientContext with phenotypes, risk scores, clinical flags, and care gaps.

### Endpoints

#### 1. Phenotype Detection

```
POST /api/v1/phenotypes/detect
```

**Request:**
```json
{
  "patient_id": "patient-123",
  "conditions": [
    {"system": "http://snomed.info/sct", "code": "49436004"},
    {"system": "http://snomed.info/sct", "code": "73211009"}
  ],
  "medications": [],
  "observations": [
    {"code": "2160-0", "value": 1.4, "unit": "mg/dL"}
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "phenotypes": [
      {
        "phenotype_id": "afib_high_stroke_risk",
        "name": "AFib with High Stroke Risk",
        "confidence": 0.95,
        "evidence": ["CHA₂DS₂-VASc ≥ 2", "Age ≥ 65"]
      },
      {
        "phenotype_id": "ckd_stage_3",
        "name": "CKD Stage 3",
        "confidence": 0.92,
        "evidence": ["eGFR 30-59"]
      }
    ],
    "total_phenotypes": 2,
    "processing_time_ms": 12
  }
}
```

#### 2. Risk Assessment

```
POST /api/v1/risk/assess
```

**Request:**
```json
{
  "patient_id": "patient-123",
  "risk_types": ["cardiovascular", "stroke", "bleeding", "fall"]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "risk_scores": {
      "stroke_risk": {
        "name": "CHA₂DS₂-VASc",
        "value": 4,
        "category": "high",
        "annual_risk": "4.8%"
      },
      "bleeding_risk": {
        "name": "HAS-BLED",
        "value": 3,
        "category": "high",
        "annual_risk": "3.74%"
      }
    },
    "confidence_score": 0.87,
    "recommendations": [
      {
        "type": "anticoagulation",
        "priority": "high",
        "description": "Start oral anticoagulation (DOAC preferred)"
      }
    ]
  }
}
```

#### 3. Care Gaps Identification

```
GET /api/v1/care-gaps/:patient_id?include_resolved=false&timeframe_days=90
```

**Response:**
```json
{
  "success": true,
  "data": {
    "patient_id": "patient-123",
    "care_gaps": [
      {
        "measure_id": "AFib-Anticoag",
        "description": "Patient with AFib and CHA₂DS₂-VASc ≥2 not on anticoagulation",
        "priority": "high",
        "recommended_action": "Consider starting oral anticoagulant",
        "evidence": {
          "cha2ds2_vasc": 4,
          "current_anticoagulant": null
        }
      }
    ],
    "total_gaps": 1,
    "priority": "high"
  }
}
```

---

## Complete Flow Example

### Step 1: KB-2A Assembly

```bash
curl -X POST http://localhost:8082/api/v1/context/build \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "patient-afib-123",
    "fhir_bundle": {
      "resourceType": "Bundle",
      "entry": [
        {"resource": {"resourceType": "Patient", "gender": "female", "birthDate": "1956-03-15"}},
        {"resource": {"resourceType": "Condition", "code": {"coding": [{"system": "http://snomed.info/sct", "code": "49436004"}]}}},
        {"resource": {"resourceType": "Observation", "code": {"coding": [{"system": "http://loinc.org", "code": "2160-0"}]}, "valueQuantity": {"value": 1.4}}}
      ]
    }
  }'
```

**Output:** Base PatientContext with conditions, labs (NO risk scores yet)

### Step 2: KB-7 Terminology (via KnowledgeSnapshotBuilder)

```bash
# Check if code is in Diabetes ValueSet
curl -X POST http://localhost:8087/v1/rules/classify \
  -H "Content-Type: application/json" \
  -d '{"code": "73211009", "system": "http://snomed.info/sct"}'
```

**Output:** ValueSet memberships for the code

### Step 3: KB-8 Calculators (via KnowledgeSnapshotBuilder)

```bash
# Calculate eGFR
curl -X POST http://localhost:8093/api/v1/calculate/egfr \
  -H "Content-Type: application/json" \
  -d '{"serumCreatinine": 1.4, "ageYears": 68, "sex": "female"}'
```

**Output:**
```json
{
  "value": 41,
  "unit": "mL/min/1.73m²",
  "ckdStage": "G3b",
  "requiresRenalDoseAdjustment": true
}
```

```bash
# Calculate CHA₂DS₂-VASc
curl -X POST http://localhost:8093/api/v1/calculate/cha2ds2vasc \
  -H "Content-Type: application/json" \
  -d '{
    "ageYears": 68,
    "sex": "female",
    "hasHypertension": true,
    "hasDiabetes": true
  }'
```

**Output:**
```json
{
  "total": 4,
  "annualStrokeRisk": "4.8%",
  "anticoagulationRecommended": true
}
```

### Step 4: KB-2B Enrichment

```bash
curl -X POST http://localhost:8082/api/v1/risk/assess \
  -H "Content-Type: application/json" \
  -d '{"patient_id": "patient-afib-123", "risk_types": ["stroke", "bleeding"]}'
```

**Output:** Enriched PatientContext with risk scores, clinical flags, care gaps

---

## Go Client Usage

### Using KB7HTTPClient

```go
package main

import (
    "context"
    "fmt"
    "vaidshala/clinical-runtime-platform/clients"
    "vaidshala/clinical-runtime-platform/contracts"
)

func main() {
    ctx := context.Background()

    // Initialize KB-7 client
    kb7Client := clients.NewKB7HTTPClient("http://localhost:8087")

    // Check if code is in a ValueSet
    code := contracts.ClinicalCode{
        System: "http://snomed.info/sct",
        Code:   "73211009",
    }

    memberships, err := kb7Client.CheckMembership(ctx, code, []string{"Diabetes"})
    if err != nil {
        panic(err)
    }

    fmt.Printf("Code %s is in ValueSets: %v\n", code.Code, memberships)

    // Expand a ValueSet
    codes, err := kb7Client.ExpandValueSet(ctx, "Diabetes")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Diabetes ValueSet has %d codes\n", len(codes))
}
```

### Using KB8HTTPClient

```go
package main

import (
    "context"
    "fmt"
    "time"
    "vaidshala/clinical-runtime-platform/clients"
    "vaidshala/clinical-runtime-platform/contracts"
)

func main() {
    ctx := context.Background()

    // Initialize KB-8 client
    kb8Client := clients.NewKB8HTTPClient("http://localhost:8093")

    // Create PatientContext
    birthDate := time.Date(1956, 3, 15, 0, 0, 0, 0, time.UTC)
    patient := &contracts.PatientContext{
        Demographics: contracts.PatientDemographics{
            Gender:    "female",
            BirthDate: &birthDate,
        },
        RecentLabResults: []contracts.LabResult{
            {
                Code:  contracts.ClinicalCode{Code: "2160-0"}, // Creatinine
                Value: &contracts.Quantity{Value: 1.4, Unit: "mg/dL"},
            },
        },
        ActiveConditions: []contracts.ClinicalCondition{
            {Code: contracts.ClinicalCode{Code: "49436004"}}, // AFib
            {Code: contracts.ClinicalCode{Code: "38341003"}}, // HTN
        },
    }

    // Calculate eGFR
    egfr, err := kb8Client.CalculateEGFR(ctx, patient)
    if err != nil {
        panic(err)
    }
    fmt.Printf("eGFR: %.1f %s (Stage: %s)\n", egfr.Value, egfr.Unit, egfr.Category)

    // Calculate CHA₂DS₂-VASc (for AFib patient)
    cha2ds2, err := kb8Client.CalculateCHA2DS2VASc(ctx, patient)
    if err != nil {
        panic(err)
    }
    fmt.Printf("CHA₂DS₂-VASc: %.0f (Category: %s)\n", cha2ds2.Value, cha2ds2.Category)
}
```

### Using KnowledgeSnapshotBuilder

```go
package main

import (
    "context"
    "fmt"
    "vaidshala/clinical-runtime-platform/builders"
    "vaidshala/clinical-runtime-platform/clients"
)

func main() {
    ctx := context.Background()

    // Initialize clients
    kb7Client := clients.NewKB7HTTPClient("http://localhost:8087")
    kb8Client := clients.NewKB8HTTPClient("http://localhost:8093")

    // Create builder with default config
    config := builders.DefaultKnowledgeSnapshotConfig()
    builder := builders.NewKnowledgeSnapshotBuilder(
        kb7Client, // KB-7 Terminology
        kb8Client, // KB-8 Calculators
        nil,       // KB-4 Safety (optional)
        nil,       // KB-5 Interactions (optional)
        nil,       // KB-6 Formulary (optional)
        nil,       // KB-1 Dosing (optional)
        nil,       // KB-11 CDI (optional)
        config,
    )

    // Build KnowledgeSnapshot from PatientContext
    snapshot, err := builder.Build(ctx, patient)
    if err != nil {
        panic(err)
    }

    // Access pre-computed answers
    fmt.Printf("eGFR: %.1f\n", snapshot.Calculators.EGFR.Value)
    fmt.Printf("CHA₂DS₂-VASc: %.0f\n", snapshot.Calculators.CHA2DS2VASc.Value)
    fmt.Printf("Is Diabetic: %v\n", snapshot.Terminology.ValueSetMemberships["is_diabetic"])
}
```

---

## Service Ports Reference

| Service | Port | Purpose |
|---------|------|---------|
| KB-2 (Clinical Context) | 8082 | Data assembly + Intelligence enrichment |
| KB-7 (Terminology) | 8087 | ValueSet expansion, code validation |
| KB-8 (Calculators) | 8093 | Clinical calculators (eGFR, ASCVD, etc.) |
| KB-4 (Patient Safety) | 8088 | Allergies, contraindications |
| KB-5 (Drug Interactions) | 8089 | Drug-drug interactions |
| KB-6 (Formulary) | 8091 | Formulary status, PBS/NLEM |
| KB-1 (Drug Rules) | 8081 | Dose adjustments |

---

## Running the E2E Test

To see the complete KB-2A → KB-2B flow with real API calls:

```bash
cd vaidshala/clinical-runtime-platform
go test -v ./tests/integration/ -run TestFullExecutionContextFlow
```

This test demonstrates:
1. FHIR Bundle parsing (KB-2A)
2. KB-7 terminology queries
3. KB-8 calculator queries (eGFR, CHA₂DS₂-VASc, HAS-BLED)
4. KnowledgeSnapshot assembly
5. KB-2B intelligence enrichment
6. Final ClinicalExecutionContext output

---

## Key Takeaways

1. **Separation of Concerns**: KB-2A handles data, KB-2B handles intelligence
2. **Pre-Computation**: All KB queries happen at build time, not execution time
3. **Frozen Contract**: Engines receive immutable context with pre-computed answers
4. **Audit Trail**: Build order matters - snapshot uses BASE context for reproducibility
5. **Parallel Queries**: KnowledgeSnapshotBuilder queries KB-7 and KB-8 in parallel for performance
