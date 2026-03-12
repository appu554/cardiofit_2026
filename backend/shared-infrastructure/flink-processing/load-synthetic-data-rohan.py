#!/usr/bin/env python3
"""
Load Rohan Sharma synthetic data into Google Cloud Healthcare FHIR store.

This script loads the comprehensive synthetic dataset for testing Module 2's
full enrichment pipeline including:
- Patient demographics
- Vital signs (BP, BMI, waist circumference)
- Lab results (HbA1c, lipid panel)
- Conditions (hypertension, prediabetes)
- Medications (Telmisartan)
- Family history (father's MI)
- Lifestyle questionnaire

Usage:
    python3 load-synthetic-data-rohan.py
"""

import os
import sys
import json
import requests
from google.auth.transport.requests import Request
from google.oauth2 import service_account

# Add parent directory to path for shared modules
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../..')))

# Google Cloud Healthcare API configuration
PROJECT_ID = os.environ.get('GOOGLE_CLOUD_PROJECT', 'cardiofit-ehr')
LOCATION = os.environ.get('GOOGLE_CLOUD_LOCATION', 'us-central1')
DATASET_ID = os.environ.get('GOOGLE_CLOUD_DATASET', 'cardiofit-fhir-dataset')
FHIR_STORE_ID = os.environ.get('GOOGLE_CLOUD_FHIR_STORE', 'cardiofit-fhir-store')
CREDENTIALS_PATH = os.environ.get('GOOGLE_APPLICATION_CREDENTIALS',
                                   '../../../services/patient-service/credentials/google-credentials.json')

# FHIR store base URL
FHIR_BASE_URL = (
    f"https://healthcare.googleapis.com/v1/projects/{PROJECT_ID}"
    f"/locations/{LOCATION}/datasets/{DATASET_ID}/fhirStores/{FHIR_STORE_ID}/fhir"
)


def get_access_token():
    """Get OAuth2 access token for Google Cloud Healthcare API."""
    credentials = service_account.Credentials.from_service_account_file(
        CREDENTIALS_PATH,
        scopes=['https://www.googleapis.com/auth/cloud-healthcare']
    )
    credentials.refresh(Request())
    return credentials.token


def create_or_update_resource(resource_type, resource_id, resource_data):
    """Create or update a FHIR resource using PUT (update or create)."""
    url = f"{FHIR_BASE_URL}/{resource_type}/{resource_id}"
    token = get_access_token()

    headers = {
        'Authorization': f'Bearer {token}',
        'Content-Type': 'application/fhir+json'
    }

    response = requests.put(url, json=resource_data, headers=headers)

    if response.status_code in [200, 201]:
        print(f"✅ Successfully created/updated {resource_type}/{resource_id}")
        return response.json()
    else:
        print(f"❌ Failed to create {resource_type}/{resource_id}: {response.status_code}")
        print(f"   Response: {response.text}")
        return None


def load_rohan_sharma_data():
    """Load all Rohan Sharma synthetic data into FHIR store."""

    print("=" * 80)
    print("Loading Rohan Sharma Synthetic Data into FHIR Store")
    print("=" * 80)
    print(f"FHIR Store: {FHIR_BASE_URL}")
    print()

    # 1. Patient Resource
    print("📋 1/7 Loading Patient...")
    patient = {
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
    }
    create_or_update_resource("Patient", "PAT-ROHAN-001", patient)

    # 2. Blood Pressure Observation
    print("\n📋 2/7 Loading Blood Pressure...")
    bp_obs = {
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
    }
    create_or_update_resource("Observation", "obs-bp-20251009", bp_obs)

    # 3. HbA1c Observation
    print("\n📋 3/7 Loading HbA1c...")
    hba1c_obs = {
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
    }
    create_or_update_resource("Observation", "obs-hba1c-20250915", hba1c_obs)

    # 4. Lipid Panel Observation
    print("\n📋 4/7 Loading Lipid Panel...")
    lipid_obs = {
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
    }
    create_or_update_resource("Observation", "obs-lipid-20250915", lipid_obs)

    # 5. Waist Circumference & BMI
    print("\n📋 5/7 Loading Anthropometric Data...")
    waist_obs = {
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
    }
    create_or_update_resource("Observation", "obs-waist-20251009", waist_obs)

    bmi_obs = {
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
    }
    create_or_update_resource("Observation", "obs-bmi-20251009", bmi_obs)

    # 6. Conditions
    print("\n📋 6/7 Loading Conditions...")
    htn_condition = {
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
    }
    create_or_update_resource("Condition", "cond-hypertension", htn_condition)

    prediabetes_condition = {
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
    }
    create_or_update_resource("Condition", "cond-prediabetes", prediabetes_condition)

    # 7. Medication & Family History
    print("\n📋 7/7 Loading Medication & Family History...")
    medication = {
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
    }
    create_or_update_resource("MedicationRequest", "medreq-1", medication)

    family_history = {
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
    }
    create_or_update_resource("FamilyMemberHistory", "family-hist-1", family_history)

    # 8. Lifestyle Questionnaire
    lifestyle_questionnaire = {
        "resourceType": "QuestionnaireResponse",
        "id": "lifestyle-20251009",
        "subject": {"reference": "Patient/PAT-ROHAN-001"},
        "authored": "2025-10-09T10:15:00Z",
        "item": [
            {
                "linkId": "diet",
                "text": "Daily fruit/veg intake",
                "answer": [{"valueString": "2 servings/day"}]
            },
            {
                "linkId": "physical-activity",
                "text": "Weekly exercise frequency",
                "answer": [{"valueString": "1 session/week"}]
            },
            {
                "linkId": "stress",
                "text": "Stress level (self-rated)",
                "answer": [{"valueString": "High"}]
            },
            {
                "linkId": "sleep",
                "text": "Average sleep per night",
                "answer": [{"valueDecimal": 5.5}]
            }
        ]
    }
    create_or_update_resource("QuestionnaireResponse", "lifestyle-20251009", lifestyle_questionnaire)

    print("\n" + "=" * 80)
    print("✅ FHIR Data Load Complete!")
    print("=" * 80)
    print("\n📊 Summary:")
    print("  - Patient: Rohan Sharma (PAT-ROHAN-001)")
    print("  - Observations: BP 150/96, HbA1c 6.3%, Lipids, BMI 29.1, Waist 95cm")
    print("  - Conditions: Hypertension, Prediabetes")
    print("  - Medications: Telmisartan 40mg")
    print("  - Family History: Father's MI at age 52")
    print("  - Lifestyle: High stress, low activity, poor diet")
    print("\n🔍 Next Steps:")
    print("  1. Load Neo4j graph data: python3 load-neo4j-rohan.py")
    print("  2. Send test event to Kafka: ./test-rohan-enrichment.sh")
    print("  3. Verify Module 2 enrichment in Flink UI: http://localhost:8081")
    print()


if __name__ == "__main__":
    try:
        load_rohan_sharma_data()
    except Exception as e:
        print(f"\n❌ Error loading FHIR data: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
