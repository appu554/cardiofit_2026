#!/usr/bin/env python3
"""
Test FHIR access using patient service credentials.

This script tests Google Cloud Healthcare FHIR API access to verify:
1. Credentials are valid
2. Healthcare API is enabled
3. FHIR store exists and is accessible
4. We can read/write FHIR resources

Based on patient service implementation patterns.
"""

import sys
import os
import json
from google.auth.transport.requests import Request
from google.oauth2 import service_account
import requests

# Terminal colors
GREEN = '\033[0;32m'
RED = '\033[0;31m'
YELLOW = '\033[1;33m'
BLUE = '\033[0;34m'
BOLD = '\033[1m'
NC = '\033[0m'

def print_header(text):
    print(f"\n{BLUE}{'━' * 80}{NC}")
    print(f"{BOLD}{text}{NC}")
    print(f"{BLUE}{'━' * 80}{NC}\n")

def print_pass(text):
    print(f"  {GREEN}✓{NC} {text}")

def print_fail(text):
    print(f"  {RED}✗{NC} {text}")

def print_warn(text):
    print(f"  {YELLOW}⊘{NC} {text}")

def print_info(text):
    print(f"  {BLUE}ℹ{NC} {text}")

# Configuration from patient service
CREDENTIALS_PATH = "/Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/credentials/google-credentials.json"

# Read credentials to get project ID
def get_project_config():
    """Extract configuration from credentials file."""
    try:
        with open(CREDENTIALS_PATH, 'r') as f:
            creds = json.load(f)

        project_id = creds.get('project_id', '')
        print_info(f"Project ID from credentials: {project_id}")

        # Default FHIR store configuration (may need adjustment)
        configs = [
            {
                "name": "Config 1 (cardiofit-ehr project)",
                "project_id": "cardiofit-ehr",
                "location": "us-central1",
                "dataset_id": "cardiofit-fhir-dataset",
                "fhir_store_id": "cardiofit-fhir-store"
            },
            {
                "name": f"Config 2 ({project_id} project)",
                "project_id": project_id,
                "location": "us-central1",
                "dataset_id": "clinical-synthesis-hub",
                "fhir_store_id": "fhir-store"
            },
            {
                "name": f"Config 3 ({project_id} alt dataset)",
                "project_id": project_id,
                "location": "us-central1",
                "dataset_id": "cardiofit-fhir-dataset",
                "fhir_store_id": "cardiofit-fhir-store"
            }
        ]

        return configs

    except Exception as e:
        print_fail(f"Cannot read credentials: {e}")
        return None

class FHIRTester:
    """Test FHIR API access."""

    def __init__(self, config):
        self.config = config
        self.project_id = config['project_id']
        self.location = config['location']
        self.dataset_id = config['dataset_id']
        self.fhir_store_id = config['fhir_store_id']

        self.fhir_base_url = (
            f"https://healthcare.googleapis.com/v1/projects/{self.project_id}"
            f"/locations/{self.location}/datasets/{self.dataset_id}"
            f"/fhirStores/{self.fhir_store_id}/fhir"
        )

        self.access_token = None

    def get_access_token(self):
        """Get OAuth2 access token."""
        try:
            credentials = service_account.Credentials.from_service_account_file(
                CREDENTIALS_PATH,
                scopes=['https://www.googleapis.com/auth/cloud-healthcare']
            )
            credentials.refresh(Request())
            self.access_token = credentials.token
            return True
        except Exception as e:
            print_fail(f"Failed to get access token: {e}")
            return False

    def test_fhir_store_access(self):
        """Test if FHIR store is accessible."""
        if not self.access_token:
            return False

        # Try to get FHIR capability statement (metadata)
        url = f"{self.fhir_base_url}/metadata"
        headers = {'Authorization': f'Bearer {self.access_token}'}

        try:
            response = requests.get(url, headers=headers, timeout=10)

            if response.status_code == 200:
                data = response.json()
                print_pass(f"FHIR store accessible")
                print_info(f"FHIR Version: {data.get('fhirVersion', 'Unknown')}")
                print_info(f"Store URL: {self.fhir_base_url}")
                return True
            elif response.status_code == 403:
                print_fail(f"Permission denied - Healthcare API may not be enabled")
                print_info(f"Enable at: https://console.cloud.google.com/apis/library/healthcare.googleapis.com?project={self.project_id}")
                return False
            elif response.status_code == 404:
                print_fail(f"FHIR store not found")
                print_info(f"Dataset: {self.dataset_id}, Store: {self.fhir_store_id}")
                return False
            else:
                print_fail(f"HTTP {response.status_code}: {response.text[:200]}")
                return False

        except Exception as e:
            print_fail(f"Error accessing FHIR store: {e}")
            return False

    def test_read_patient(self, patient_id="PAT-ROHAN-001"):
        """Test reading a patient resource."""
        if not self.access_token:
            return False

        url = f"{self.fhir_base_url}/Patient/{patient_id}"
        headers = {'Authorization': f'Bearer {self.access_token}'}

        try:
            response = requests.get(url, headers=headers, timeout=10)

            if response.status_code == 200:
                data = response.json()
                name = data.get('name', [{}])[0]
                given = ' '.join(name.get('given', []))
                family = name.get('family', '')
                print_pass(f"Patient found: {given} {family}")
                return True
            elif response.status_code == 404:
                print_warn(f"Patient {patient_id} not found (needs to be loaded)")
                return False
            else:
                print_fail(f"HTTP {response.status_code}")
                return False

        except Exception as e:
            print_fail(f"Error reading patient: {e}")
            return False

    def test_create_test_patient(self):
        """Test creating a test patient."""
        if not self.access_token:
            return False

        test_patient_id = "TEST-PATIENT-001"

        # Check if test patient already exists
        url_check = f"{self.fhir_base_url}/Patient/{test_patient_id}"
        headers = {'Authorization': f'Bearer {self.access_token}', 'Content-Type': 'application/fhir+json'}

        try:
            response = requests.get(url_check, headers=headers, timeout=10)
            if response.status_code == 200:
                print_pass(f"Test patient already exists")
                return True
        except:
            pass

        # Create test patient
        test_patient = {
            "resourceType": "Patient",
            "id": test_patient_id,
            "identifier": [{
                "system": "https://test.cardiofit.com",
                "value": "TEST-001"
            }],
            "name": [{
                "use": "official",
                "family": "Test",
                "given": ["FHIR"]
            }],
            "gender": "other",
            "birthDate": "2025-01-01"
        }

        url = f"{self.fhir_base_url}/Patient/{test_patient_id}"

        try:
            response = requests.put(url, json=test_patient, headers=headers, timeout=10)

            if response.status_code in [200, 201]:
                print_pass(f"Created test patient successfully")
                return True
            else:
                print_fail(f"Failed to create patient: HTTP {response.status_code}")
                print_info(f"Response: {response.text[:200]}")
                return False

        except Exception as e:
            print_fail(f"Error creating patient: {e}")
            return False

    def test_search_patients(self):
        """Test searching for patients."""
        if not self.access_token:
            return False

        url = f"{self.fhir_base_url}/Patient?_count=5"
        headers = {'Authorization': f'Bearer {self.access_token}'}

        try:
            response = requests.get(url, headers=headers, timeout=10)

            if response.status_code == 200:
                data = response.json()
                total = data.get('total', 0)
                entries = len(data.get('entry', []))
                print_pass(f"Search successful: {entries} patients returned (total: {total})")
                return True
            else:
                print_fail(f"Search failed: HTTP {response.status_code}")
                return False

        except Exception as e:
            print_fail(f"Error searching: {e}")
            return False

def test_configuration(config):
    """Test a specific configuration."""
    print_header(f"Testing: {config['name']}")

    print_info(f"Project: {config['project_id']}")
    print_info(f"Location: {config['location']}")
    print_info(f"Dataset: {config['dataset_id']}")
    print_info(f"FHIR Store: {config['fhir_store_id']}")
    print()

    tester = FHIRTester(config)

    # Test 1: Get access token
    print("1. Authentication")
    if not tester.get_access_token():
        return False
    print_pass("Access token obtained")
    print()

    # Test 2: Access FHIR store
    print("2. FHIR Store Access")
    if not tester.test_fhir_store_access():
        return False
    print()

    # Test 3: Search for patients
    print("3. Search Patients")
    tester.test_search_patients()
    print()

    # Test 4: Read Rohan Sharma patient
    print("4. Read Rohan Sharma Patient")
    tester.test_read_patient("PAT-ROHAN-001")
    print()

    # Test 5: Create test patient
    print("5. Create Test Patient")
    tester.test_create_test_patient()
    print()

    return True

def main():
    """Main test workflow."""
    print(f"\n{BOLD}{'=' * 80}{NC}")
    print(f"{BOLD}Google Cloud Healthcare FHIR API Access Test{NC}")
    print(f"{BOLD}Using Patient Service Credentials{NC}")
    print(f"{BOLD}{'=' * 80}{NC}")

    # Check credentials file exists
    if not os.path.exists(CREDENTIALS_PATH):
        print_fail(f"Credentials file not found: {CREDENTIALS_PATH}")
        return 1

    print_pass(f"Credentials file found")

    # Get possible configurations
    configs = get_project_config()
    if not configs:
        return 1

    print_info(f"Testing {len(configs)} possible configurations...")
    print()

    # Test each configuration
    successful_config = None
    for config in configs:
        if test_configuration(config):
            successful_config = config
            break

    # Summary
    print_header("Summary")

    if successful_config:
        print(f"{GREEN}{'╔' + '═' * 78 + '╗'}{NC}")
        print(f"{GREEN}║{' ' * 20}✓ FHIR ACCESS WORKING{' ' * 28}║{NC}")
        print(f"{GREEN}{'╚' + '═' * 78 + '╝'}{NC}")
        print()
        print(f"{BOLD}Working Configuration:{NC}")
        print(f"  Project: {successful_config['project_id']}")
        print(f"  Dataset: {successful_config['dataset_id']}")
        print(f"  FHIR Store: {successful_config['fhir_store_id']}")
        print()
        print(f"{BOLD}Next Steps:{NC}")
        print(f"  1. Update flink.properties with these values")
        print(f"  2. Load Rohan Sharma data using GOOGLE_FHIR_UI_GUIDE.md")
        print(f"  3. Run: ./test-rohan-enrichment.sh")
        print()
        return 0
    else:
        print(f"{RED}{'╔' + '═' * 78 + '╗'}{NC}")
        print(f"{RED}║{' ' * 20}✗ FHIR ACCESS FAILED{' ' * 29}║{NC}")
        print(f"{RED}{'╚' + '═' * 78 + '╝'}{NC}")
        print()
        print(f"{YELLOW}Possible Issues:{NC}")
        print(f"  1. Healthcare API not enabled - Enable at:")
        print(f"     https://console.cloud.google.com/apis/library/healthcare.googleapis.com")
        print(f"  2. FHIR store doesn't exist - Create at:")
        print(f"     https://console.cloud.google.com/healthcare/browser")
        print(f"  3. Service account lacks permissions - Add 'Healthcare FHIR Resource Editor' role")
        print()
        return 1

if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print(f"\n\n{YELLOW}⊘ Test interrupted by user{NC}\n")
        sys.exit(1)
    except Exception as e:
        print(f"\n{RED}✗ Unexpected error: {e}{NC}\n")
        import traceback
        traceback.print_exc()
        sys.exit(1)
