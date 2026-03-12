# Manual FHIR Data Loading Guide for Rohan Sharma

This guide provides step-by-step instructions for manually loading the Rohan Sharma synthetic data into Google Cloud Healthcare FHIR store.

## Prerequisites

### 1. Enable Google Cloud Healthcare API

```bash
# Set your project ID
export PROJECT_ID="cardiofit-ehr"

# Enable Healthcare API
gcloud services enable healthcare.googleapis.com --project=$PROJECT_ID

# Verify it's enabled
gcloud services list --enabled --project=$PROJECT_ID | grep healthcare
```

### 2. Verify Credentials and Permissions

```bash
# Check current authentication
gcloud auth list

# If not authenticated, login
gcloud auth login

# Set application default credentials
gcloud auth application-default login

# Verify service account has permissions
gcloud projects get-iam-policy $PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:*"

# Add Healthcare permissions if needed
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:YOUR_SERVICE_ACCOUNT@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/healthcare.fhirResourceEditor"
```

### 3. Create Healthcare Dataset and FHIR Store (if not exists)

```bash
# Set variables
export LOCATION="us-central1"
export DATASET_ID="cardiofit-fhir-dataset"
export FHIR_STORE_ID="cardiofit-fhir-store"

# Create dataset
gcloud healthcare datasets create $DATASET_ID \
  --location=$LOCATION \
  --project=$PROJECT_ID

# Create FHIR store
gcloud healthcare fhir-stores create $FHIR_STORE_ID \
  --dataset=$DATASET_ID \
  --location=$LOCATION \
  --version=R4 \
  --project=$PROJECT_ID

# Verify creation
gcloud healthcare fhir-stores describe $FHIR_STORE_ID \
  --dataset=$DATASET_ID \
  --location=$LOCATION \
  --project=$PROJECT_ID
```

## Option 1: Load via REST API (Recommended)

### Setup Environment

```bash
# Get access token
export ACCESS_TOKEN=$(gcloud auth application-default print-access-token)

# Set FHIR store URL
export FHIR_BASE_URL="https://healthcare.googleapis.com/v1/projects/$PROJECT_ID/locations/$LOCATION/datasets/$DATASET_ID/fhirStores/$FHIR_STORE_ID/fhir"

# Test connection
curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$FHIR_BASE_URL/Patient?_count=1" | jq .
```

### Load Resources Step-by-Step

#### 1. Patient Resource

```bash
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Patient/PAT-ROHAN-001" \
  -d '{
    "resourceType": "Patient",
    "id": "PAT-ROHAN-001",
    "identifier": [{"system": "https://ayuehr.in/patients", "value": "ROHAN-001"}],
    "name": [{"use": "official", "family": "Sharma", "given": ["Rohan"]}],
    "gender": "male",
    "birthDate": "1983-05-15",
    "address": [{
      "line": ["JP Nagar"],
      "city": "Bengaluru",
      "state": "Karnataka",
      "postalCode": "560078",
      "country": "IN"
    }]
  }'
```

#### 2. Blood Pressure Observation

```bash
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Observation/obs-bp-20251009" \
  -d '{
    "resourceType": "Observation",
    "id": "obs-bp-20251009",
    "status": "final",
    "category": [{"coding": [{"code": "vital-signs"}]}],
    "code": {
      "coding": [{
        "system": "http://loinc.org",
        "code": "85354-9",
        "display": "Blood pressure panel"
      }]
    },
    "subject": {"reference": "Patient/PAT-ROHAN-001"},
    "effectiveDateTime": "2025-10-09T10:05:00Z",
    "component": [
      {
        "code": {"coding": [{"code": "8480-6", "display": "Systolic BP"}]},
        "valueQuantity": {"value": 150, "unit": "mmHg"}
      },
      {
        "code": {"coding": [{"code": "8462-4", "display": "Diastolic BP"}]},
        "valueQuantity": {"value": 96, "unit": "mmHg"}
      }
    ]
  }'
```

#### 3. HbA1c Observation

```bash
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Observation/obs-hba1c-20250915" \
  -d '{
    "resourceType": "Observation",
    "id": "obs-hba1c-20250915",
    "status": "final",
    "category": [{"coding": [{"code": "laboratory"}]}],
    "code": {
      "coding": [{
        "system": "http://loinc.org",
        "code": "4548-4",
        "display": "Hemoglobin A1c"
      }]
    },
    "subject": {"reference": "Patient/PAT-ROHAN-001"},
    "effectiveDateTime": "2025-09-15T08:00:00Z",
    "valueQuantity": {"value": 6.3, "unit": "%"}
  }'
```

#### 4. Lipid Panel

```bash
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Observation/obs-lipid-20250915" \
  -d '{
    "resourceType": "Observation",
    "id": "obs-lipid-20250915",
    "status": "final",
    "category": [{"coding": [{"code": "laboratory"}]}],
    "code": {
      "coding": [{
        "system": "http://loinc.org",
        "code": "24331-1",
        "display": "Lipid panel"
      }]
    },
    "subject": {"reference": "Patient/PAT-ROHAN-001"},
    "effectiveDateTime": "2025-09-15T08:00:00Z",
    "component": [
      {
        "code": {"coding": [{"code": "2085-9", "display": "HDL Cholesterol"}]},
        "valueQuantity": {"value": 38, "unit": "mg/dL"}
      },
      {
        "code": {"coding": [{"code": "13457-7", "display": "LDL Cholesterol"}]},
        "valueQuantity": {"value": 155, "unit": "mg/dL"}
      },
      {
        "code": {"coding": [{"code": "2571-8", "display": "Triglycerides"}]},
        "valueQuantity": {"value": 180, "unit": "mg/dL"}
      }
    ]
  }'
```

#### 5. Anthropometric Data (BMI & Waist)

```bash
# Waist circumference
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Observation/obs-waist-20251009" \
  -d '{
    "resourceType": "Observation",
    "id": "obs-waist-20251009",
    "status": "final",
    "code": {
      "coding": [{
        "system": "http://loinc.org",
        "code": "8280-0",
        "display": "Waist circumference"
      }]
    },
    "subject": {"reference": "Patient/PAT-ROHAN-001"},
    "effectiveDateTime": "2025-10-09T10:06:00Z",
    "valueQuantity": {"value": 95, "unit": "cm"}
  }'

# BMI
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Observation/obs-bmi-20251009" \
  -d '{
    "resourceType": "Observation",
    "id": "obs-bmi-20251009",
    "status": "final",
    "code": {
      "coding": [{
        "system": "http://loinc.org",
        "code": "39156-5",
        "display": "Body Mass Index"
      }]
    },
    "subject": {"reference": "Patient/PAT-ROHAN-001"},
    "effectiveDateTime": "2025-10-09T10:07:00Z",
    "valueQuantity": {"value": 29.1, "unit": "kg/m2"}
  }'
```

#### 6. Conditions

```bash
# Hypertension
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Condition/cond-hypertension" \
  -d '{
    "resourceType": "Condition",
    "id": "cond-hypertension",
    "clinicalStatus": {"coding": [{"code": "active"}]},
    "code": {
      "coding": [{
        "system": "http://snomed.info/sct",
        "code": "38341003",
        "display": "Hypertensive disorder"
      }]
    },
    "subject": {"reference": "Patient/PAT-ROHAN-001"},
    "onsetDateTime": "2023-06-10T00:00:00Z"
  }'

# Prediabetes
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Condition/cond-prediabetes" \
  -d '{
    "resourceType": "Condition",
    "id": "cond-prediabetes",
    "clinicalStatus": {"coding": [{"code": "active"}]},
    "code": {
      "coding": [{
        "system": "http://snomed.info/sct",
        "code": "15777000",
        "display": "Prediabetes"
      }]
    },
    "subject": {"reference": "Patient/PAT-ROHAN-001"},
    "onsetDateTime": "2024-03-10T00:00:00Z"
  }'
```

#### 7. Medication

```bash
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/MedicationRequest/medreq-1" \
  -d '{
    "resourceType": "MedicationRequest",
    "id": "medreq-1",
    "status": "active",
    "intent": "order",
    "medicationCodeableConcept": {
      "coding": [{
        "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
        "code": "860975",
        "display": "Telmisartan 40 mg Tablet"
      }]
    },
    "subject": {"reference": "Patient/PAT-ROHAN-001"},
    "authoredOn": "2025-09-20T09:00:00Z",
    "dosageInstruction": [{"text": "Take one tablet once daily in the morning"}]
  }'
```

#### 8. Family History

```bash
curl -X PUT \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/FamilyMemberHistory/family-hist-1" \
  -d '{
    "resourceType": "FamilyMemberHistory",
    "id": "family-hist-1",
    "status": "completed",
    "patient": {"reference": "Patient/PAT-ROHAN-001"},
    "relationship": {"coding": [{"code": "FTH", "display": "Father"}]},
    "condition": [{
      "code": {
        "coding": [{
          "system": "http://snomed.info/sct",
          "code": "22298006",
          "display": "Myocardial infarction"
        }]
      },
      "onsetString": "Father at age 52"
    }]
  }'
```

### Verify All Resources Loaded

```bash
# Count resources by type
echo "Patient count:"
curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$FHIR_BASE_URL/Patient?_count=1000" | jq '.total'

echo "Observation count:"
curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$FHIR_BASE_URL/Observation?_count=1000" | jq '.total'

echo "Condition count:"
curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$FHIR_BASE_URL/Condition?_count=1000" | jq '.total'

# Get Rohan Sharma's complete record
curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$FHIR_BASE_URL/Patient/PAT-ROHAN-001/\$everything" | jq .
```

## Option 2: Use Local FHIR Server (Alternative)

If Google Cloud setup is problematic, use a local HAPI FHIR server for testing:

### Start HAPI FHIR Server

```bash
docker run -d -p 8082:8080 --name hapi-fhir \
  hapiproject/hapi:latest

# Wait for startup
sleep 30

# Test connection
curl http://localhost:8082/fhir/metadata | jq '.fhirVersion'
```

### Update Module 2 Configuration

Edit `flink.properties` to use local FHIR server:

```properties
# Comment out Google Cloud FHIR
# google.cloud.fhir.base.url=https://healthcare.googleapis.com/...

# Use local FHIR server
fhir.base.url=http://localhost:8082/fhir
fhir.auth.type=none
```

### Load Data to Local FHIR

```bash
# Set local FHIR URL
export FHIR_BASE_URL="http://localhost:8082/fhir"

# Run the same curl commands as above but without Authorization header
# Example:
curl -X PUT \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL/Patient/PAT-ROHAN-001" \
  -d '{ /* patient resource */ }'
```

## Option 3: Batch Upload Script

Create a batch upload script for all resources:

```bash
#!/bin/bash
# load-fhir-batch.sh

export ACCESS_TOKEN=$(gcloud auth application-default print-access-token)
export FHIR_BASE_URL="https://healthcare.googleapis.com/v1/projects/cardiofit-ehr/locations/us-central1/datasets/cardiofit-fhir-dataset/fhirStores/cardiofit-fhir-store/fhir"

# Bundle all resources
curl -X POST \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/fhir+json" \
  "$FHIR_BASE_URL" \
  -d @rohan-sharma-bundle.json
```

## Troubleshooting

### 403 Permission Denied

```bash
# Check project billing
gcloud billing projects describe $PROJECT_ID

# Enable billing if needed
gcloud billing projects link $PROJECT_ID \
  --billing-account=BILLING_ACCOUNT_ID

# Verify Healthcare API is enabled
gcloud services list --enabled --project=$PROJECT_ID | grep healthcare

# Check IAM permissions
gcloud projects get-iam-policy $PROJECT_ID
```

### 404 Resource Not Found

```bash
# Verify FHIR store exists
gcloud healthcare fhir-stores list \
  --dataset=$DATASET_ID \
  --location=$LOCATION \
  --project=$PROJECT_ID
```

### Authentication Issues

```bash
# Refresh credentials
gcloud auth application-default login

# Set quota project
gcloud auth application-default set-quota-project $PROJECT_ID

# Use service account key directly
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/credentials/google-credentials.json"
```

## Verification Checklist

After loading, verify:

- [ ] Patient PAT-ROHAN-001 exists
- [ ] Blood pressure observation (150/96 mmHg)
- [ ] HbA1c result (6.3%)
- [ ] Lipid panel (HDL 38, LDL 155, TG 180)
- [ ] BMI (29.1) and waist (95 cm)
- [ ] Hypertension condition
- [ ] Prediabetes condition
- [ ] Telmisartan medication
- [ ] Father's MI family history

## Next Steps

Once FHIR data is loaded:

1. Verify Module 2 configuration points to correct FHIR store
2. Test Module 2 enrichment: `./test-rohan-enrichment.sh`
3. Check enriched output includes both FHIR and Neo4j data
4. Monitor Flink logs for any FHIR API errors

## Quick Reference

```bash
# Get access token
gcloud auth application-default print-access-token

# Query patient
curl -H "Authorization: Bearer $(gcloud auth application-default print-access-token)" \
  "https://healthcare.googleapis.com/v1/projects/cardiofit-ehr/locations/us-central1/datasets/cardiofit-fhir-dataset/fhirStores/cardiofit-fhir-store/fhir/Patient/PAT-ROHAN-001"

# Delete resource (if needed)
curl -X DELETE \
  -H "Authorization: Bearer $(gcloud auth application-default print-access-token)" \
  "https://healthcare.googleapis.com/v1/projects/cardiofit-ehr/locations/us-central1/datasets/cardiofit-fhir-dataset/fhirStores/cardiofit-fhir-store/fhir/Patient/PAT-ROHAN-001"
```
