# Phenotype Evaluation API

## Overview

The Phenotype Evaluation API provides comprehensive clinical phenotyping capabilities using CEL (Common Expression Language) based rules. This endpoint can process up to 1,000 patients in batch mode with sub-100ms performance.

## Endpoint Details

```
POST /v1/phenotypes/evaluate
```

**Performance SLA**: 100ms p95 for batch of 100 patients  
**Rate Limit**: 1,000 requests/hour (individual), 10,000/hour (service)  
**Authentication**: Required (JWT Bearer token)

## Request Format

### Request Headers

```http
Authorization: Bearer <jwt_token>
Content-Type: application/json
X-Client-ID: <client_identifier>
X-Request-ID: <unique_request_id>
X-Debug: false  # Set to true for debug information
```

### Request Body

```json
{
  "patients": [
    {
      "id": "patient_12345",
      "age": 65,
      "gender": "male",
      "weight": {
        "value": 80.5,
        "unit": "kg"
      },
      "height": {
        "value": 175,
        "unit": "cm"
      },
      "conditions": [
        "diabetes_type_2",
        "hypertension",
        "coronary_artery_disease"
      ],
      "medications": [
        {
          "name": "metformin",
          "dosage": "1000mg",
          "frequency": "twice_daily",
          "start_date": "2024-06-01"
        },
        {
          "name": "lisinopril",
          "dosage": "10mg",
          "frequency": "once_daily",
          "start_date": "2024-03-15"
        }
      ],
      "labs": {
        "hba1c": {
          "value": 8.2,
          "unit": "%",
          "date": "2025-01-10"
        },
        "total_cholesterol": {
          "value": 240,
          "unit": "mg/dL",
          "date": "2025-01-10"
        },
        "ldl_cholesterol": {
          "value": 160,
          "unit": "mg/dL",
          "date": "2025-01-10"
        },
        "creatinine": {
          "value": 1.2,
          "unit": "mg/dL",
          "date": "2025-01-10"
        },
        "egfr": {
          "value": 58,
          "unit": "mL/min/1.73m²",
          "date": "2025-01-10"
        }
      },
      "vitals": {
        "systolic_bp": 145,
        "diastolic_bp": 90,
        "heart_rate": 78,
        "temperature": 98.6,
        "date": "2025-01-15"
      },
      "allergies": [
        {
          "substance": "penicillin",
          "reaction": "rash",
          "severity": "moderate"
        }
      ],
      "family_history": [
        {
          "condition": "myocardial_infarction",
          "relationship": "father",
          "age_at_diagnosis": 55
        }
      ]
    }
  ],
  "phenotype_filters": [
    "cardiovascular",
    "diabetes",
    "medication"
  ],
  "include_explanation": false,
  "batch_size": 100,
  "detail_level": "standard"
}
```

### Request Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `patients` | Array | Yes | Array of patient data objects (max 1,000) |
| `phenotype_filters` | Array | No | Filter to specific phenotype categories |
| `include_explanation` | Boolean | No | Include detailed reasoning (default: false) |
| `batch_size` | Integer | No | Processing batch size (default: 100, max: 1,000) |
| `detail_level` | String | No | Detail level: "minimal", "standard", "comprehensive" |

## Response Format

### Success Response

```json
{
  "status": "success",
  "data": {
    "results": [
      {
        "patient_id": "patient_12345",
        "phenotypes": [
          {
            "id": "high_cardiovascular_risk",
            "name": "High Cardiovascular Risk",
            "category": "cardiovascular",
            "positive": true,
            "confidence": 0.95,
            "severity": "moderate",
            "contributing_factors": [
              "age >= 65",
              "diabetes_type_2_present",
              "hypertension_present",
              "ldl_cholesterol > 100"
            ],
            "rule_evaluation": {
              "rule_id": "cv_risk_001",
              "cel_expression": "age >= 65 && (has_condition('diabetes') || has_condition('hypertension')) && lab_value('ldl_cholesterol') > 100",
              "evaluation_result": true,
              "intermediate_values": {
                "age_check": true,
                "condition_check": true,
                "ldl_check": true
              }
            }
          },
          {
            "id": "diabetes_poor_control",
            "name": "Diabetes with Poor Glycemic Control",
            "category": "diabetes",
            "positive": true,
            "confidence": 0.98,
            "severity": "high",
            "contributing_factors": [
              "hba1c > 8.0",
              "diabetes_type_2_present",
              "current_antidiabetic_therapy"
            ],
            "rule_evaluation": {
              "rule_id": "dm_control_001",
              "cel_expression": "has_condition('diabetes_type_2') && lab_value('hba1c') > 8.0",
              "evaluation_result": true,
              "intermediate_values": {
                "diabetes_check": true,
                "hba1c_check": true
              }
            }
          },
          {
            "id": "ckd_stage_3",
            "name": "Chronic Kidney Disease Stage 3",
            "category": "renal",
            "positive": true,
            "confidence": 0.92,
            "severity": "moderate",
            "contributing_factors": [
              "egfr_30_to_59",
              "diabetes_present",
              "hypertension_present"
            ],
            "rule_evaluation": {
              "rule_id": "ckd_stage_001",
              "cel_expression": "lab_value('egfr') >= 30 && lab_value('egfr') < 60 && (has_condition('diabetes') || has_condition('hypertension'))",
              "evaluation_result": true,
              "intermediate_values": {
                "egfr_range_check": true,
                "risk_factor_check": true
              }
            }
          }
        ],
        "processing_metadata": {
          "evaluation_time_ms": 15,
          "rules_evaluated": 12,
          "cache_hits": 8,
          "phenotypes_positive": 3,
          "confidence_average": 0.95
        }
      }
    ],
    "summary": {
      "total_patients": 1,
      "total_phenotypes_evaluated": 12,
      "positive_phenotypes": 3,
      "average_processing_time_ms": 15,
      "cache_hit_rate": 0.67
    }
  },
  "metadata": {
    "request_id": "req_123456789",
    "timestamp": "2025-01-15T10:30:00Z",
    "processing_time_ms": 25,
    "version": "1.0.0",
    "batch_id": "batch_987654321"
  }
}
```

### Error Response

```json
{
  "status": "error",
  "errors": [
    {
      "code": "INVALID_PATIENT_DATA",
      "message": "Patient age is required for phenotype evaluation",
      "field": "patients[0].age",
      "severity": "error"
    },
    {
      "code": "MISSING_REQUIRED_LABS",
      "message": "HbA1c value required for diabetes phenotype evaluation",
      "field": "patients[0].labs.hba1c",
      "severity": "warning"
    }
  ],
  "metadata": {
    "request_id": "req_123456789",
    "timestamp": "2025-01-15T10:30:00Z",
    "processing_time_ms": 5
  }
}
```

## Phenotype Categories

### Cardiovascular Phenotypes

| Phenotype ID | Name | Description | Key Factors |
|--------------|------|-------------|-------------|
| `high_cardiovascular_risk` | High Cardiovascular Risk | Elevated 10-year CV risk | Age, diabetes, hypertension, cholesterol |
| `heart_failure_risk` | Heart Failure Risk | Risk for developing HF | EF, BNP, prior MI, diabetes |
| `coronary_artery_disease` | Coronary Artery Disease | Known or suspected CAD | Prior MI, stents, bypass, symptoms |
| `atrial_fibrillation_risk` | Atrial Fibrillation Risk | Risk for developing AF | Age, heart failure, hypertension |

### Diabetes Phenotypes

| Phenotype ID | Name | Description | Key Factors |
|--------------|------|-------------|-------------|
| `diabetes_poor_control` | Poor Glycemic Control | HbA1c > 8% or frequent highs | HbA1c, glucose patterns, medication adherence |
| `diabetes_complications` | Diabetic Complications | Micro/macrovascular complications | Retinopathy, nephropathy, neuropathy |
| `insulin_resistance` | Insulin Resistance | Evidence of insulin resistance | BMI, HOMA-IR, metabolic syndrome |
| `hypoglycemia_risk` | Hypoglycemia Risk | Risk for severe hypoglycemia | Medications, prior episodes, renal function |

### Medication Risk Phenotypes

| Phenotype ID | Name | Description | Key Factors |
|--------------|------|-------------|-------------|
| `polypharmacy_risk` | Polypharmacy Risk | >5 medications or interactions | Medication count, interaction potential |
| `medication_adherence_risk` | Poor Adherence Risk | Factors affecting adherence | Complexity, cost, side effects |
| `drug_allergy_risk` | Drug Allergy Risk | High risk for allergic reactions | Prior reactions, cross-reactivity |
| `renal_dosing_required` | Renal Dose Adjustment | Medications requiring adjustment | eGFR, nephrotoxic medications |

## Clinical Decision Logic

### CEL Rule Examples

#### High Cardiovascular Risk

```cel
// Primary cardiovascular risk assessment
age >= 65 && 
(has_condition('diabetes_type_2') || has_condition('diabetes_type_1')) &&
(has_condition('hypertension') || vitals.systolic_bp > 140) &&
(lab_value('ldl_cholesterol') > 100 || lab_value('total_cholesterol') > 200)

// Enhanced with additional factors
|| (age >= 50 && 
    family_history_includes('coronary_artery_disease') && 
    (smoking_status == 'current' || lab_value('ldl_cholesterol') > 160))

// Risk modifiers
&& !has_condition('end_stage_renal_disease')  // Exclude ESRD patients
&& !has_medication_class('statin')  // Not already on optimal therapy
```

#### Diabetes Poor Control

```cel
has_condition('diabetes_type_2') &&
(
  lab_value('hba1c') > 8.0 ||
  (lab_value('hba1c') > 7.5 && age < 65) ||
  (glucose_readings_avg_30d > 180 && glucose_readings_count > 10)
) &&
// Exclude if recent medication changes
!medication_changed_within_days(90) &&
// Exclude if hospitalized recently
!hospitalized_within_days(30)
```

#### CKD Stage Classification

```cel
// Stage 3A CKD
lab_value('egfr') >= 45 && lab_value('egfr') < 60 &&
(has_condition('diabetes') || has_condition('hypertension') || 
 lab_value('urine_protein') > 300) &&
// Confirmed on two occasions
egfr_trend_declining_6_months()

// Stage 3B CKD  
|| (lab_value('egfr') >= 30 && lab_value('egfr') < 45 &&
    has_supporting_evidence_ckd())
```

### Rule Evaluation Process

1. **Patient Data Validation**: Verify required data elements
2. **Rule Selection**: Identify applicable phenotype rules
3. **CEL Evaluation**: Execute CEL expressions against patient data
4. **Confidence Calculation**: Assess rule confidence based on data quality
5. **Result Compilation**: Aggregate positive phenotypes with contributing factors

### Confidence Scoring

Confidence scores are calculated based on:

- **Data Completeness**: 0.1-0.3 weight
- **Data Recency**: 0.1-0.2 weight  
- **Rule Specificity**: 0.3-0.5 weight
- **Clinical Context**: 0.1-0.3 weight

```go
confidence = (data_completeness * 0.3) + 
             (data_recency * 0.2) + 
             (rule_specificity * 0.4) + 
             (clinical_context * 0.1)
```

## Usage Examples

### Basic Phenotype Evaluation

```bash
curl -X POST http://localhost:8088/v1/phenotypes/evaluate \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patients": [{
      "id": "patient_001",
      "age": 68,
      "gender": "female",
      "conditions": ["diabetes_type_2", "hypertension"],
      "labs": {
        "hba1c": {"value": 8.5, "unit": "%", "date": "2025-01-10"},
        "ldl_cholesterol": {"value": 180, "unit": "mg/dL", "date": "2025-01-10"}
      },
      "vitals": {
        "systolic_bp": 155,
        "diastolic_bp": 95,
        "date": "2025-01-15"
      }
    }],
    "phenotype_filters": ["cardiovascular", "diabetes"],
    "include_explanation": true
  }'
```

### Batch Processing Example

```bash
curl -X POST http://localhost:8088/v1/phenotypes/evaluate \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d @batch_patients.json
```

### Filtered Evaluation

```bash
curl -X POST http://localhost:8088/v1/phenotypes/evaluate \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patients": [...],
    "phenotype_filters": ["cardiovascular"],
    "detail_level": "comprehensive"
  }'
```

## Performance Optimization

### Request Optimization

- **Batch Size**: Optimal batch size is 100-500 patients
- **Data Minimization**: Include only necessary patient data fields
- **Filtering**: Use `phenotype_filters` to limit evaluation scope
- **Detail Level**: Use "minimal" for basic phenotype identification

### Response Caching

Results are cached based on:
- Patient data hash
- Phenotype rule versions  
- Request parameters

Cache TTL: 1 hour for standard requests, 5 minutes for debug requests

### Performance Monitoring

```bash
# Monitor endpoint performance
curl -w "@curl-format.txt" -s -o /dev/null \
  -X POST http://localhost:8088/v1/phenotypes/evaluate \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d @test_request.json
```

## Integration Examples

### Python Integration

```python
import requests
import json

def evaluate_phenotypes(patients, token):
    url = "http://localhost:8088/v1/phenotypes/evaluate"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "patients": patients,
        "phenotype_filters": ["cardiovascular", "diabetes"],
        "include_explanation": False
    }
    
    response = requests.post(url, headers=headers, json=payload)
    
    if response.status_code == 200:
        return response.json()
    else:
        response.raise_for_status()

# Usage
patients = [{"id": "p001", "age": 65, ...}]
result = evaluate_phenotypes(patients, jwt_token)
```

### Go Integration

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type PhenotypeRequest struct {
    Patients         []Patient `json:"patients"`
    PhenotypeFilters []string  `json:"phenotype_filters,omitempty"`
    IncludeExplanation bool    `json:"include_explanation"`
}

func evaluatePhenotypes(req PhenotypeRequest, token string) (*PhenotypeResponse, error) {
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }
    
    httpReq, err := http.NewRequest("POST", "http://localhost:8088/v1/phenotypes/evaluate", 
        bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("Authorization", "Bearer "+token)
    httpReq.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result PhenotypeResponse
    err = json.NewDecoder(resp.Body).Decode(&result)
    return &result, err
}
```

## Validation and Testing

### Test Data Sets

The service includes comprehensive test data sets:

- **Basic Cases**: Standard phenotype scenarios
- **Edge Cases**: Boundary conditions and missing data
- **Complex Cases**: Multiple comorbidities and interactions
- **Performance Cases**: Large batch processing scenarios

### Validation Tools

```bash
# Validate phenotype rules
curl -X POST http://localhost:8088/v1/validation/phenotypes \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"phenotype_id": "high_cardiovascular_risk"}'

# Test with sample data
curl -X POST http://localhost:8088/v1/phenotypes/evaluate \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d @test-data/sample-patients.json
```

---

**Last Updated**: 2025-01-15  
**API Version**: 1.0.0  
**Next Review**: 2025-04-15