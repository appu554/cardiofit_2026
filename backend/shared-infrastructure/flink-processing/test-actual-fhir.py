#!/usr/bin/env python3
"""
Test FHIR access with actual cardiofit-905a8 project configuration.
"""

import sys
import json
import requests
from google.auth.transport.requests import Request
from google.oauth2 import service_account

# Terminal colors
GREEN = '\033[0;32m'
RED = '\033[0;31m'
YELLOW = '\033[1;33m'
BLUE = '\033[0;34m'
BOLD = '\033[1m'
NC = '\033[0m'

CREDENTIALS_PATH = "/Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/credentials/google-credentials.json"

# Actual configuration from user
# projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store
PROJECT_ID = "cardiofit-905a8"
LOCATION = "asia-south1"
DATASET_ID = "clinical-synthesis-hub"
FHIR_STORE_ID = "fhir-store"

FHIR_BASE_URL = (
    f"https://healthcare.googleapis.com/v1/projects/{PROJECT_ID}"
    f"/locations/{LOCATION}/datasets/{DATASET_ID}/fhirStores/{FHIR_STORE_ID}/fhir"
)

def get_access_token():
    """Get OAuth2 access token."""
    credentials = service_account.Credentials.from_service_account_file(
        CREDENTIALS_PATH,
        scopes=['https://www.googleapis.com/auth/cloud-healthcare']
    )
    credentials.refresh(Request())
    return credentials.token

def main():
    print(f"\n{BOLD}Testing FHIR Access: cardiofit-905a8/cardiofit/fhir-store{NC}\n")

    # Get token
    print("1. Getting access token...")
    token = get_access_token()
    print(f"   {GREEN}✓{NC} Token obtained\n")

    headers = {'Authorization': f'Bearer {token}'}

    # Test metadata
    print("2. Testing FHIR store access...")
    response = requests.get(f"{FHIR_BASE_URL}/metadata", headers=headers, timeout=10)
    if response.status_code == 200:
        data = response.json()
        print(f"   {GREEN}✓{NC} FHIR store accessible")
        print(f"   {BLUE}ℹ{NC} FHIR Version: {data.get('fhirVersion', 'Unknown')}\n")
    else:
        print(f"   {RED}✗{NC} Failed: HTTP {response.status_code}")
        print(f"   {response.text[:200]}\n")
        return 1

    # Search existing patients
    print("3. Searching for existing patients...")
    response = requests.get(f"{FHIR_BASE_URL}/Patient?_count=10", headers=headers, timeout=10)
    if response.status_code == 200:
        data = response.json()
        total = data.get('total', 0)
        entries = data.get('entry', [])
        print(f"   {GREEN}✓{NC} Found {len(entries)} patients (total: {total})")

        if entries:
            print(f"\n   Existing patients:")
            for entry in entries[:5]:
                patient = entry.get('resource', {})
                patient_id = patient.get('id', 'Unknown')
                name = patient.get('name', [{}])[0]
                given = ' '.join(name.get('given', []))
                family = name.get('family', '')
                print(f"     • {patient_id}: {given} {family}")
        print()
    else:
        print(f"   {RED}✗{NC} Search failed: HTTP {response.status_code}\n")

    # Check for Rohan Sharma
    print("4. Checking for Rohan Sharma (PAT-ROHAN-001)...")
    response = requests.get(f"{FHIR_BASE_URL}/Patient/PAT-ROHAN-001", headers=headers, timeout=10)
    if response.status_code == 200:
        data = response.json()
        name = data.get('name', [{}])[0]
        given = ' '.join(name.get('given', []))
        family = name.get('family', '')
        print(f"   {GREEN}✓{NC} Patient found: {given} {family}")
        print(f"   {BLUE}ℹ{NC} Birth Date: {data.get('birthDate', 'Unknown')}")
        print(f"   {BLUE}ℹ{NC} Gender: {data.get('gender', 'Unknown')}\n")
    elif response.status_code == 404:
        print(f"   {YELLOW}⊘{NC} Patient PAT-ROHAN-001 not found - needs to be loaded\n")
    else:
        print(f"   {RED}✗{NC} Failed: HTTP {response.status_code}\n")

    # Check for observations
    print("5. Checking for Rohan's observations...")
    obs_ids = ["obs-bp-20251009", "obs-hba1c-20250915", "obs-lipid-20250915"]
    found_obs = 0

    for obs_id in obs_ids:
        response = requests.get(f"{FHIR_BASE_URL}/Observation/{obs_id}", headers=headers, timeout=10)
        if response.status_code == 200:
            data = response.json()
            code = data.get('code', {}).get('coding', [{}])[0].get('display', 'Unknown')
            print(f"   {GREEN}✓{NC} {obs_id}: {code}")
            found_obs += 1
        elif response.status_code == 404:
            print(f"   {YELLOW}⊘{NC} {obs_id}: Not found")

    print()

    # Summary
    print(f"{BOLD}{'=' * 80}{NC}")
    print(f"{BOLD}Summary:{NC}\n")
    print(f"  FHIR Store URL: {FHIR_BASE_URL}")
    print(f"  Project: {PROJECT_ID}")
    print(f"  Dataset: {DATASET_ID}")
    print(f"  FHIR Store: {FHIR_STORE_ID}\n")

    if response.status_code == 200 and found_obs > 0:
        print(f"{GREEN}✓ FHIR store is operational with Rohan's data{NC}\n")
        print(f"{BOLD}Ready for Module 2 testing!{NC}")
        print(f"Run: ./test-rohan-enrichment.sh\n")
        return 0
    else:
        print(f"{YELLOW}⊘ FHIR store accessible but Rohan's data not loaded{NC}\n")
        print(f"{BOLD}Next Steps:{NC}")
        print(f"  1. Follow GOOGLE_FHIR_UI_GUIDE.md to load Rohan's data")
        print(f"  2. Use this FHIR store URL in the UI:")
        print(f"     {FHIR_BASE_URL}\n")
        return 0

if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as e:
        print(f"\n{RED}✗ Error: {e}{NC}\n")
        import traceback
        traceback.print_exc()
        sys.exit(1)
