#!/usr/bin/env python3
"""
Check Data Sources for Module 2
Verifies that patient/encounter services and databases are accessible
"""

import requests
import json
import sys

# Color codes
GREEN = '\033[92m'
YELLOW = '\033[93m'
RED = '\033[91m'
BLUE = '\033[94m'
BOLD = '\033[1m'
RESET = '\033[0m'

def print_header(text):
    print(f"\n{BOLD}{BLUE}{'='*70}{RESET}")
    print(f"{BOLD}{BLUE}{text}{RESET}")
    print(f"{BOLD}{BLUE}{'='*70}{RESET}\n")

def print_success(text):
    print(f"{GREEN}✅ {text}{RESET}")

def print_error(text):
    print(f"{RED}❌ {text}{RESET}")

def print_warning(text):
    print(f"{YELLOW}⚠️  {text}{RESET}")

def print_info(text):
    print(f"{BLUE}ℹ️  {text}{RESET}")

def check_service(name, url, port):
    """Check if a service is accessible"""
    try:
        response = requests.get(url, timeout=5)
        if response.status_code < 400:
            print_success(f"{name} is RUNNING on port {port}")
            return True
        else:
            print_error(f"{name} returned status {response.status_code}")
            return False
    except requests.exceptions.ConnectionError:
        print_error(f"{name} is NOT accessible at {url}")
        print_info(f"  Start with: cd backend/services/{name.lower().replace(' ', '-')} && python run_service.py")
        return False
    except requests.exceptions.Timeout:
        print_warning(f"{name} timed out - may be slow")
        return False
    except Exception as e:
        print_error(f"{name} check failed: {e}")
        return False

def check_patient_data(patient_id="P12345"):
    """Check if patient data exists"""
    url = f"http://localhost:8003/patients/{patient_id}"
    try:
        response = requests.get(url, timeout=5)
        if response.status_code == 200:
            patient = response.json()
            print_success(f"Patient {patient_id} found in database")
            print(f"  Name: {patient.get('firstName', 'N/A')} {patient.get('lastName', 'N/A')}")
            print(f"  MRN: {patient.get('mrn', 'N/A')}")
            return True
        elif response.status_code == 404:
            print_warning(f"Patient {patient_id} NOT found")
            print_info("  Add test patient with: POST http://localhost:8003/patients")
            return False
        else:
            print_error(f"Unexpected response: {response.status_code}")
            return False
    except Exception as e:
        print_error(f"Cannot check patient data: {e}")
        return False

def show_data_source_summary():
    """Show summary of all data sources"""
    print_header("Data Sources for Module 2")

    print(f"{BOLD}Available Data Sources:{RESET}\n")

    sources = {
        "Patient Service": {
            "url": "http://localhost:8003/health",
            "port": 8003,
            "provides": "Patient demographics, MRN, contact info",
            "database": "MongoDB or Google Healthcare API"
        },
        "Encounter Service": {
            "url": "http://localhost:8010/health",
            "port": 8010,
            "provides": "Hospital visits, department, care team",
            "database": "MongoDB/PostgreSQL"
        },
        "Observation Service": {
            "url": "http://localhost:8010/health",
            "port": 8010,
            "provides": "Clinical observations, vital signs",
            "database": "MongoDB"
        }
    }

    results = {}
    for name, info in sources.items():
        print(f"{BOLD}{name}:{RESET}")
        print(f"  Provides: {info['provides']}")
        print(f"  Database: {info['database']}")
        print(f"  Status: ", end="")

        results[name] = check_service(name, info['url'], info['port'])
        print()

    return results

def test_patient_lookup_flow():
    """Test the complete patient lookup flow"""
    print_header("Testing Patient Lookup Flow")

    print("This simulates what Module 2 will do:")
    print("1. Extract patientId from event")
    print("2. Call patient service API")
    print("3. Parse patient data")
    print("4. Add to enriched event\n")

    patient_id = "P12345"

    print(f"Step 1: Extract patientId from event")
    print(f"  patientId = '{patient_id}' ✓\n")

    print(f"Step 2: Call patient service API")
    url = f"http://localhost:8003/patients/{patient_id}"
    print(f"  URL: {url}")

    try:
        response = requests.get(url, timeout=5)
        print(f"  Status: {response.status_code}")

        if response.status_code == 200:
            print_success("API call successful\n")

            print(f"Step 3: Parse patient data")
            patient = response.json()
            print(f"  Patient data received:")
            print(f"    patientId: {patient.get('patientId', 'N/A')}")
            print(f"    firstName: {patient.get('firstName', 'N/A')}")
            print(f"    lastName: {patient.get('lastName', 'N/A')}")
            print(f"    dateOfBirth: {patient.get('dateOfBirth', 'N/A')}")
            print(f"    gender: {patient.get('gender', 'N/A')}")
            print(f"    mrn: {patient.get('mrn', 'N/A')}\n")

            print(f"Step 4: Add to enriched event")
            enriched = {
                "eventId": "abc-123",
                "patientId": patient_id,
                "eventType": "vital_signs",
                "timestamp": 1759305006359,
                "payload": {"heart_rate": 78},
                "patient": {  # ← Module 2 adds this
                    "firstName": patient.get('firstName'),
                    "lastName": patient.get('lastName'),
                    "dateOfBirth": patient.get('dateOfBirth'),
                    "gender": patient.get('gender'),
                    "mrn": patient.get('mrn')
                }
            }

            print("  Enriched event:")
            print(json.dumps(enriched, indent=2))
            print()
            print_success("Patient lookup flow SUCCESSFUL!")
            return True

        elif response.status_code == 404:
            print_error("Patient not found in database\n")
            print_info("Add test patient data first:")
            print("  1. Start patient-service: cd backend/services/patient-service && python run_service.py")
            print("  2. Add patient: POST http://localhost:8003/patients with patient data")
            return False
        else:
            print_error(f"Unexpected status code: {response.status_code}")
            return False

    except requests.exceptions.ConnectionError:
        print_error("Cannot connect to patient service\n")
        print_info("Start patient service:")
        print("  cd backend/services/patient-service && python run_service.py")
        return False
    except Exception as e:
        print_error(f"Error: {e}")
        return False

def create_test_patient():
    """Create a test patient for Module 2 testing"""
    print_header("Creating Test Patient")

    test_patient = {
        "patientId": "P12345",
        "firstName": "John",
        "lastName": "Doe",
        "dateOfBirth": "1980-05-15",
        "gender": "male",
        "mrn": "MRN-67890",
        "address": {
            "street": "123 Main St",
            "city": "Boston",
            "state": "MA",
            "zip": "02101"
        },
        "contactInfo": {
            "phone": "+1-555-0123",
            "email": "john.doe@example.com"
        }
    }

    print("Patient data to create:")
    print(json.dumps(test_patient, indent=2))
    print()

    url = "http://localhost:8003/patients"

    try:
        response = requests.post(url, json=test_patient, timeout=5)

        if response.status_code in [200, 201]:
            print_success("Test patient created successfully!")
            return True
        elif response.status_code == 409:
            print_warning("Patient already exists")
            return True
        else:
            print_error(f"Failed to create patient: {response.status_code}")
            print(f"Response: {response.text}")
            return False

    except requests.exceptions.ConnectionError:
        print_error("Cannot connect to patient service")
        print_info("Start patient service first:")
        print("  cd backend/services/patient-service && python run_service.py")
        return False
    except Exception as e:
        print_error(f"Error: {e}")
        return False

def main():
    """Main function"""
    print(f"\n{BOLD}{BLUE}╔════════════════════════════════════════════════════════════╗{RESET}")
    print(f"{BOLD}{BLUE}║     Module 2 Data Source Checker                          ║{RESET}")
    print(f"{BOLD}{BLUE}╚════════════════════════════════════════════════════════════╝{RESET}\n")

    if len(sys.argv) > 1:
        command = sys.argv[1].lower()

        if command == "services":
            show_data_source_summary()
        elif command == "test":
            test_patient_lookup_flow()
        elif command == "create":
            create_test_patient()
        else:
            print("Usage:")
            print(f"  {sys.argv[0]} services  # Check all services")
            print(f"  {sys.argv[0]} test      # Test patient lookup")
            print(f"  {sys.argv[0]} create    # Create test patient")
    else:
        # Run all checks
        results = show_data_source_summary()

        if results.get("Patient Service"):
            check_patient_data("P12345")
            print()
            test_patient_lookup_flow()
        else:
            print()
            print_warning("Patient service is not running")
            print_info("Module 2 requires patient service to be running")
            print()
            print("Next steps:")
            print("  1. Start patient service:")
            print("     cd backend/services/patient-service && python run_service.py")
            print("  2. Create test patient:")
            print(f"     python3 {sys.argv[0]} create")
            print("  3. Test lookup flow:")
            print(f"     python3 {sys.argv[0]} test")

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print(f"\n\n{YELLOW}Interrupted by user{RESET}")
        sys.exit(0)
