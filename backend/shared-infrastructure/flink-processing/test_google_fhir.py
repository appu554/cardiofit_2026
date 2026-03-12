#!/usr/bin/env python3
"""
Test Google Cloud Healthcare FHIR API
Fetches patient data to verify AsyncPatientEnricher can access FHIR resources.

Usage:
    python test_google_fhir.py [patient_id]

Example:
    python test_google_fhir.py 905a60cb-8241-418f-b29b-5b020e851392
"""

import sys
import json
import requests
from google.auth import default
from google.auth.transport.requests import Request

# Configuration (matching Flink configuration)
PROJECT_ID = "cardiofit-905a8"
LOCATION = "asia-south1"
DATASET = "clinical-synthesis-hub"
FHIR_STORE = "fhir-store"
BASE_URL = f"https://healthcare.googleapis.com/v1/projects/{PROJECT_ID}/locations/{LOCATION}/datasets/{DATASET}/fhirStores/{FHIR_STORE}/fhir"

def get_access_token():
    """Get Google Cloud access token using application default credentials."""
    try:
        credentials, project = default(scopes=['https://www.googleapis.com/auth/cloud-healthcare'])
        credentials.refresh(Request())
        return credentials.token
    except Exception as e:
        print(f"❌ Failed to get access token: {e}")
        print("\n💡 Try running: gcloud auth application-default login")
        sys.exit(1)

def fetch_patient(patient_id, access_token):
    """Fetch patient resource from FHIR API."""
    url = f"{BASE_URL}/Patient/{patient_id}"
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/fhir+json"
    }

    print(f"📋 Fetching Patient/{patient_id}...")
    print(f"URL: {url}\n")

    try:
        response = requests.get(url, headers=headers, timeout=5)

        if response.status_code == 200:
            patient = response.json()
            print("✅ Patient found (HTTP 200)")
            print("\nPatient Details:")
            print(f"  ID: {patient.get('id')}")
            print(f"  Resource Type: {patient.get('resourceType')}")

            if patient.get('name'):
                name = patient['name'][0]
                given = ' '.join(name.get('given', []))
                family = name.get('family', '')
                print(f"  Name: {given} {family}")

            print(f"  Gender: {patient.get('gender', 'N/A')}")
            print(f"  Birth Date: {patient.get('birthDate', 'N/A')}")

            if patient.get('identifier'):
                print(f"  Identifiers: {len(patient['identifier'])} identifier(s)")

            return patient

        elif response.status_code == 404:
            print("❌ Patient not found (HTTP 404)")
            print("\n💡 This patient ID does not exist in the FHIR store.")
            print("   AsyncPatientEnricher will return empty snapshot (expected behavior).")
            return None

        else:
            print(f"⚠️  Unexpected status: HTTP {response.status_code}")
            print(f"Response: {response.text}")
            return None

    except requests.exceptions.Timeout:
        print("⚠️  Request timeout (5 seconds)")
        print("   FHIR API is slow or unreachable")
        return None
    except Exception as e:
        print(f"❌ Error fetching patient: {e}")
        return None

def fetch_medications(patient_id, access_token):
    """Fetch medication statements for patient."""
    url = f"{BASE_URL}/MedicationStatement"
    params = {"subject": f"Patient/{patient_id}"}
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/fhir+json"
    }

    print(f"\n📋 Fetching MedicationStatement for Patient/{patient_id}...")

    try:
        response = requests.get(url, headers=headers, params=params, timeout=5)

        if response.status_code == 200:
            bundle = response.json()
            total = bundle.get('total', 0)
            print(f"✅ Medications found: {total}")

            if total > 0 and bundle.get('entry'):
                print("\nMedications:")
                for i, entry in enumerate(bundle['entry'][:5], 1):
                    resource = entry.get('resource', {})
                    med_code = resource.get('medicationCodeableConcept', {})
                    med_name = med_code.get('text', 'Unknown')
                    status = resource.get('status', 'Unknown')

                    dosage_text = "No dosage info"
                    if resource.get('dosage') and len(resource['dosage']) > 0:
                        dosage_text = resource['dosage'][0].get('text', 'No dosage info')

                    print(f"  {i}. {med_name}")
                    print(f"     Status: {status}")
                    print(f"     Dosage: {dosage_text}")

                if total > 5:
                    print(f"  ... and {total - 5} more")

            return bundle
        else:
            print(f"⚠️  Medications query status: HTTP {response.status_code}")
            return None

    except Exception as e:
        print(f"❌ Error fetching medications: {e}")
        return None

def fetch_conditions(patient_id, access_token):
    """Fetch conditions for patient."""
    url = f"{BASE_URL}/Condition"
    params = {"subject": f"Patient/{patient_id}"}
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/fhir+json"
    }

    print(f"\n📋 Fetching Condition for Patient/{patient_id}...")

    try:
        response = requests.get(url, headers=headers, params=params, timeout=5)

        if response.status_code == 200:
            bundle = response.json()
            total = bundle.get('total', 0)
            print(f"✅ Conditions found: {total}")

            if total > 0 and bundle.get('entry'):
                print("\nConditions:")
                for i, entry in enumerate(bundle['entry'][:5], 1):
                    resource = entry.get('resource', {})
                    code = resource.get('code', {})
                    condition_name = code.get('text', 'Unknown condition')

                    clinical_status = "Unknown"
                    if resource.get('clinicalStatus'):
                        clinical_status = resource['clinicalStatus'].get('coding', [{}])[0].get('code', 'Unknown')

                    print(f"  {i}. {condition_name}")
                    print(f"     Clinical Status: {clinical_status}")

                if total > 5:
                    print(f"  ... and {total - 5} more")

            return bundle
        else:
            print(f"⚠️  Conditions query status: HTTP {response.status_code}")
            return None

    except Exception as e:
        print(f"❌ Error fetching conditions: {e}")
        return None

def list_patients(access_token, count=5):
    """List available patients in FHIR store."""
    url = f"{BASE_URL}/Patient"
    params = {"_count": count}
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/fhir+json"
    }

    print(f"\n📋 Listing {count} patients from FHIR store...")

    try:
        response = requests.get(url, headers=headers, params=params, timeout=5)

        if response.status_code == 200:
            bundle = response.json()
            total = bundle.get('total', 0)
            print(f"✅ Total patients in store: {total}")

            if bundle.get('entry'):
                print(f"\nFirst {len(bundle['entry'])} patients:")
                for i, entry in enumerate(bundle['entry'], 1):
                    resource = entry.get('resource', {})
                    patient_id = resource.get('id', 'Unknown')

                    name_str = "Unknown"
                    if resource.get('name'):
                        name = resource['name'][0]
                        given = ' '.join(name.get('given', []))
                        family = name.get('family', '')
                        name_str = f"{given} {family}".strip()

                    print(f"  {i}. ID: {patient_id}")
                    print(f"     Name: {name_str}")
                    print(f"     Gender: {resource.get('gender', 'N/A')}")

                print("\n💡 You can test with any of these patient IDs")
                return bundle
            else:
                print("⚠️  No patients found in FHIR store")
                return None
        else:
            print(f"⚠️  Patient list query status: HTTP {response.status_code}")
            return None

    except Exception as e:
        print(f"❌ Error listing patients: {e}")
        return None

def main():
    """Main test function."""
    patient_id = sys.argv[1] if len(sys.argv) > 1 else "905a60cb-8241-418f-b29b-5b020e851392"

    print("=" * 60)
    print("Google Cloud Healthcare FHIR API Test")
    print("=" * 60)
    print(f"Project: {PROJECT_ID}")
    print(f"Location: {LOCATION}")
    print(f"Dataset: {DATASET}")
    print(f"FHIR Store: {FHIR_STORE}")
    print(f"Base URL: {BASE_URL}")
    print(f"Patient ID: {patient_id}")
    print("=" * 60)
    print()

    # Get access token
    print("🔑 Getting access token...")
    access_token = get_access_token()
    print("✅ Access token obtained\n")

    # Test 1: Fetch specific patient
    patient = fetch_patient(patient_id, access_token)

    if patient:
        # Test 2: Fetch medications
        medications = fetch_medications(patient_id, access_token)

        # Test 3: Fetch conditions
        conditions = fetch_conditions(patient_id, access_token)

        # Summary
        print("\n" + "=" * 60)
        print("Summary")
        print("=" * 60)
        print(f"Patient: ✅ Found")
        print(f"Medications: {medications.get('total', 0) if medications else 0}")
        print(f"Conditions: {conditions.get('total', 0) if conditions else 0}")
        print("=" * 60)

        print("\n✅ This patient can be used for testing AsyncPatientEnricher!")
        print("   Send an event with this patient ID to see FHIR enrichment.")
    else:
        # Patient not found - list available patients
        print("\n" + "=" * 60)
        print("Patient not found - Listing available patients...")
        print("=" * 60)
        list_patients(access_token, count=10)

        print("\n" + "=" * 60)
        print("Next Steps")
        print("=" * 60)
        print("1. Use one of the patient IDs listed above for testing")
        print("2. OR create a new patient in the FHIR store")
        print("3. Then send an event with that patient ID")
        print("=" * 60)

if __name__ == "__main__":
    main()
