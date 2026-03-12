# Google Cloud Healthcare FHIR Store - UI Loading Guide

Complete guide for loading Rohan Sharma synthetic data through the Google Cloud Console UI.

## Step 1: Access FHIR Store in Google Cloud Console

1. Navigate to: https://console.cloud.google.com/healthcare
2. Select your project: **`cardiofit-ehr`**
3. Click on **"Browser"** in the left menu
4. Navigate to:
   - Location: **`us-central1`**
   - Dataset: **`cardiofit-fhir-dataset`**
   - FHIR Store: **`cardiofit-fhir-store`**

## Step 2: Access FHIR Resource Browser

1. Click on the FHIR store name: **`cardiofit-fhir-store`**
2. Click the **"VIEW RESOURCES"** button or **"FHIR VIEWER"** tab
3. You should see the FHIR resource management interface

## Step 3: Create Patient Resource

### Method A: Through "Create Resource" Button

1. Click **"+ CREATE RESOURCE"** button
2. Select resource type: **`Patient`**
3. Copy and paste this JSON:

```json
{
  "resourceType": "Patient",
  "id": "PAT-ROHAN-001",
  "identifier": [
    {
      "system": "https://ayuehr.in/patients",
      "value": "ROHAN-001"
    }
  ],
  "name": [
    {
      "use": "official",
      "family": "Sharma",
      "given": ["Rohan"]
    }
  ],
  "gender": "male",
  "birthDate": "1983-05-15",
  "address": [
    {
      "line": ["JP Nagar"],
      "city": "Bengaluru",
      "state": "Karnataka",
      "postalCode": "560078",
      "country": "IN"
    }
  ]
}
```

4. Click **"CREATE"**
5. Verify creation success ✅

### Method B: Through REST API Console

1. Click **"REST API Console"** or **"Try this API"**
2. Select `projects.locations.datasets.fhirStores.fhir.create`
3. Enter:
   - **parent**: `projects/cardiofit-ehr/locations/us-central1/datasets/cardiofit-fhir-dataset/fhirStores/cardiofit-fhir-store`
   - **type**: `Patient`
   - **Request body**: (paste JSON above)

## Step 4: Create Observations

### 4.1 Blood Pressure Observation

Click **"+ CREATE RESOURCE"** → Select **`Observation`** → Paste:

```json
{
  "resourceType": "Observation",
  "id": "obs-bp-20251009",
  "status": "final",
  "category": [
    {
      "coding": [
        {
          "code": "vital-signs"
        }
      ]
    }
  ],
  "code": {
    "coding": [
      {
        "system": "http://loinc.org",
        "code": "85354-9",
        "display": "Blood pressure panel"
      }
    ]
  },
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "effectiveDateTime": "2025-10-09T10:05:00Z",
  "component": [
    {
      "code": {
        "coding": [
          {
            "code": "8480-6",
            "display": "Systolic BP"
          }
        ]
      },
      "valueQuantity": {
        "value": 150,
        "unit": "mmHg"
      }
    },
    {
      "code": {
        "coding": [
          {
            "code": "8462-4",
            "display": "Diastolic BP"
          }
        ]
      },
      "valueQuantity": {
        "value": 96,
        "unit": "mmHg"
      }
    }
  ]
}
```

### 4.2 HbA1c Lab Result

Click **"+ CREATE RESOURCE"** → Select **`Observation`** → Paste:

```json
{
  "resourceType": "Observation",
  "id": "obs-hba1c-20250915",
  "status": "final",
  "category": [
    {
      "coding": [
        {
          "code": "laboratory"
        }
      ]
    }
  ],
  "code": {
    "coding": [
      {
        "system": "http://loinc.org",
        "code": "4548-4",
        "display": "Hemoglobin A1c"
      }
    ]
  },
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "effectiveDateTime": "2025-09-15T08:00:00Z",
  "valueQuantity": {
    "value": 6.3,
    "unit": "%"
  }
}
```

### 4.3 Lipid Panel

Click **"+ CREATE RESOURCE"** → Select **`Observation`** → Paste:

```json
{
  "resourceType": "Observation",
  "id": "obs-lipid-20250915",
  "status": "final",
  "category": [
    {
      "coding": [
        {
          "code": "laboratory"
        }
      ]
    }
  ],
  "code": {
    "coding": [
      {
        "system": "http://loinc.org",
        "code": "24331-1",
        "display": "Lipid panel"
      }
    ]
  },
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "effectiveDateTime": "2025-09-15T08:00:00Z",
  "component": [
    {
      "code": {
        "coding": [
          {
            "code": "2085-9",
            "display": "HDL Cholesterol"
          }
        ]
      },
      "valueQuantity": {
        "value": 38,
        "unit": "mg/dL"
      }
    },
    {
      "code": {
        "coding": [
          {
            "code": "13457-7",
            "display": "LDL Cholesterol"
          }
        ]
      },
      "valueQuantity": {
        "value": 155,
        "unit": "mg/dL"
      }
    },
    {
      "code": {
        "coding": [
          {
            "code": "2571-8",
            "display": "Triglycerides"
          }
        ]
      },
      "valueQuantity": {
        "value": 180,
        "unit": "mg/dL"
      }
    }
  ]
}
```

### 4.4 Body Mass Index (BMI)

Click **"+ CREATE RESOURCE"** → Select **`Observation`** → Paste:

```json
{
  "resourceType": "Observation",
  "id": "obs-bmi-20251009",
  "status": "final",
  "code": {
    "coding": [
      {
        "system": "http://loinc.org",
        "code": "39156-5",
        "display": "Body Mass Index"
      }
    ]
  },
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "effectiveDateTime": "2025-10-09T10:07:00Z",
  "valueQuantity": {
    "value": 29.1,
    "unit": "kg/m2"
  }
}
```

### 4.5 Waist Circumference

Click **"+ CREATE RESOURCE"** → Select **`Observation`** → Paste:

```json
{
  "resourceType": "Observation",
  "id": "obs-waist-20251009",
  "status": "final",
  "code": {
    "coding": [
      {
        "system": "http://loinc.org",
        "code": "8280-0",
        "display": "Waist circumference"
      }
    ]
  },
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "effectiveDateTime": "2025-10-09T10:06:00Z",
  "valueQuantity": {
    "value": 95,
    "unit": "cm"
  }
}
```

## Step 5: Create Condition Resources

### 5.1 Hypertension

Click **"+ CREATE RESOURCE"** → Select **`Condition`** → Paste:

```json
{
  "resourceType": "Condition",
  "id": "cond-hypertension",
  "clinicalStatus": {
    "coding": [
      {
        "code": "active"
      }
    ]
  },
  "code": {
    "coding": [
      {
        "system": "http://snomed.info/sct",
        "code": "38341003",
        "display": "Hypertensive disorder"
      }
    ]
  },
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "onsetDateTime": "2023-06-10T00:00:00Z"
}
```

### 5.2 Prediabetes

Click **"+ CREATE RESOURCE"** → Select **`Condition`** → Paste:

```json
{
  "resourceType": "Condition",
  "id": "cond-prediabetes",
  "clinicalStatus": {
    "coding": [
      {
        "code": "active"
      }
    ]
  },
  "code": {
    "coding": [
      {
        "system": "http://snomed.info/sct",
        "code": "15777000",
        "display": "Prediabetes"
      }
    ]
  },
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "onsetDateTime": "2024-03-10T00:00:00Z"
}
```

## Step 6: Create Medication Request

Click **"+ CREATE RESOURCE"** → Select **`MedicationRequest`** → Paste:

```json
{
  "resourceType": "MedicationRequest",
  "id": "medreq-1",
  "status": "active",
  "intent": "order",
  "medicationCodeableConcept": {
    "coding": [
      {
        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
        "code": "860975",
        "display": "Telmisartan 40 mg Tablet"
      }
    ]
  },
  "subject": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "authoredOn": "2025-09-20T09:00:00Z",
  "dosageInstruction": [
    {
      "text": "Take one tablet once daily in the morning"
    }
  ]
}
```

## Step 7: Create Family Member History

Click **"+ CREATE RESOURCE"** → Select **`FamilyMemberHistory`** → Paste:

```json
{
  "resourceType": "FamilyMemberHistory",
  "id": "family-hist-1",
  "status": "completed",
  "patient": {
    "reference": "Patient/PAT-ROHAN-001"
  },
  "relationship": {
    "coding": [
      {
        "code": "FTH",
        "display": "Father"
      }
    ]
  },
  "condition": [
    {
      "code": {
        "coding": [
          {
            "system": "http://snomed.info/sct",
            "code": "22298006",
            "display": "Myocardial infarction"
          }
        ]
      },
      "onsetString": "Father at age 52"
    }
  ]
}
```

## Step 8: Verify All Resources

### In the FHIR Viewer:

1. Click **"Search"** or filter by:
   - Resource Type: **All**
   - Patient: **PAT-ROHAN-001**

2. You should see:
   - ✅ 1 Patient
   - ✅ 5 Observations (BP, HbA1c, Lipids, BMI, Waist)
   - ✅ 2 Conditions (Hypertension, Prediabetes)
   - ✅ 1 MedicationRequest
   - ✅ 1 FamilyMemberHistory
   - **Total: 10 resources**

### Using Search Queries:

Search for patient's complete record:
- Query: `Patient/PAT-ROHAN-001/$everything`
- This should return all 10 resources in a Bundle

## Quick Copy-Paste Checklist

Use this checklist while creating resources:

```
[ ] Patient: PAT-ROHAN-001 (Rohan Sharma, Male, DOB: 1983-05-15)
[ ] Observation: BP 150/96 mmHg (obs-bp-20251009)
[ ] Observation: HbA1c 6.3% (obs-hba1c-20250915)
[ ] Observation: Lipid Panel - HDL 38, LDL 155, TG 180 (obs-lipid-20250915)
[ ] Observation: BMI 29.1 kg/m2 (obs-bmi-20251009)
[ ] Observation: Waist 95 cm (obs-waist-20251009)
[ ] Condition: Hypertension (cond-hypertension)
[ ] Condition: Prediabetes (cond-prediabetes)
[ ] MedicationRequest: Telmisartan 40mg (medreq-1)
[ ] FamilyMemberHistory: Father's MI at 52 (family-hist-1)
```

## Troubleshooting

### Can't Find "Create Resource" Button?

- Make sure you're in the FHIR store detail view
- Look for tabs: **Overview**, **Resources**, **Import**, **Export**
- Click the **Resources** tab
- The **"+ CREATE RESOURCE"** button should be at the top

### Validation Errors?

Common issues:
- **Missing required field**: Check that `resourceType` and `id` are present
- **Invalid reference**: Ensure `Patient/PAT-ROHAN-001` exists before creating references
- **Date format**: Use ISO 8601 format: `YYYY-MM-DDTHH:MM:SSZ`

### Permission Denied?

You need the role:
- **Healthcare FHIR Resource Editor** or
- **Healthcare Dataset Administrator**

To add permissions:
1. Go to IAM & Admin → IAM
2. Find your user account
3. Click **Edit** (pencil icon)
4. Click **Add Another Role**
5. Select **Healthcare FHIR Resource Editor**
6. Click **Save**

## Alternative: Bulk Upload via Import

If creating one-by-one is tedious:

1. Save all JSONs to a single NDJSON file
2. Upload to Google Cloud Storage
3. Use FHIR Import feature:
   - FHIR Store → **Import** tab
   - Source: Cloud Storage bucket
   - File: `rohan-sharma-resources.ndjson`
   - Click **Import**

NDJSON format (one resource per line):
```
{"resourceType":"Patient","id":"PAT-ROHAN-001",...}
{"resourceType":"Observation","id":"obs-bp-20251009",...}
{"resourceType":"Observation","id":"obs-hba1c-20250915",...}
...
```

## Next Steps After Loading

1. **Verify in UI**: All 10 resources visible in FHIR viewer
2. **Test API Access**: Run Module 2's FHIR client test
3. **Run Enrichment Test**: Execute `./test-rohan-enrichment.sh`
4. **Check Logs**: Monitor Flink for FHIR API success messages

## Quick Reference Links

- **Google Cloud Console Healthcare**: https://console.cloud.google.com/healthcare
- **FHIR R4 Specification**: http://hl7.org/fhir/R4/
- **LOINC Code Search**: https://loinc.org/search/
- **SNOMED CT Browser**: https://browser.ihtsdotools.org/

---

**Pro Tip**: Keep this guide open in a browser tab while you create resources in the Google Cloud Console in another tab for easy copy-pasting!
