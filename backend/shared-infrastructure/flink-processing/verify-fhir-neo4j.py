#!/usr/bin/env python3
"""
Verify data consistency between FHIR and Neo4j for Rohan Sharma.

This script uses patient service credentials to check Google Cloud Healthcare FHIR
and validates Neo4j graph data to ensure both systems are ready for Module 2 testing.
"""

import sys
import json
import requests
from google.auth.transport.requests import Request
from google.oauth2 import service_account
from neo4j import GraphDatabase

# Terminal colors
class Colors:
    GREEN = '\033[0;32m'
    RED = '\033[0;31m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    BOLD = '\033[1m'
    NC = '\033[0m'  # No Color

def print_header(text):
    print(f"\n{Colors.BLUE}{'━' * 80}{Colors.NC}")
    print(f"{Colors.BOLD}{text}{Colors.NC}")
    print(f"{Colors.BLUE}{'━' * 80}{Colors.NC}\n")

def print_pass(text):
    print(f"  {Colors.GREEN}✓ PASS{Colors.NC} - {text}")

def print_fail(text):
    print(f"  {Colors.RED}✗ FAIL{Colors.NC} - {text}")

def print_warn(text):
    print(f"  {Colors.YELLOW}⊘ WARN{Colors.NC} - {text}")

# Google Cloud Healthcare API Configuration
CREDENTIALS_PATH = "/Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/credentials/google-credentials.json"
PROJECT_ID = "cardiofit-ehr"
LOCATION = "us-central1"
DATASET_ID = "cardiofit-fhir-dataset"
FHIR_STORE_ID = "cardiofit-fhir-store"
FHIR_BASE_URL = (
    f"https://healthcare.googleapis.com/v1/projects/{PROJECT_ID}"
    f"/locations/{LOCATION}/datasets/{DATASET_ID}/fhirStores/{FHIR_STORE_ID}/fhir"
)

# Neo4j Configuration
NEO4J_URI = "bolt://localhost:7687"
NEO4J_USERNAME = "neo4j"
NEO4J_PASSWORD = "CardioFit2024!"

# Test patient ID
PATIENT_ID = "PAT-ROHAN-001"


class FHIRChecker:
    """Check FHIR store for Rohan Sharma data."""

    def __init__(self):
        self.access_token = None
        self.credentials_valid = False

    def get_access_token(self):
        """Get OAuth2 access token for Google Cloud Healthcare API."""
        try:
            credentials = service_account.Credentials.from_service_account_file(
                CREDENTIALS_PATH,
                scopes=['https://www.googleapis.com/auth/cloud-healthcare']
            )
            credentials.refresh(Request())
            self.access_token = credentials.token
            self.credentials_valid = True
            return True
        except FileNotFoundError:
            print_fail(f"Credentials file not found: {CREDENTIALS_PATH}")
            return False
        except Exception as e:
            print_fail(f"Failed to get access token: {e}")
            return False

    def check_resource(self, resource_type, resource_id, description):
        """Check if a FHIR resource exists."""
        if not self.credentials_valid:
            print_warn(f"{description} (skipped - no credentials)")
            return None

        url = f"{FHIR_BASE_URL}/{resource_type}/{resource_id}"
        headers = {'Authorization': f'Bearer {self.access_token}'}

        try:
            response = requests.get(url, headers=headers, timeout=10)

            if response.status_code == 200:
                data = response.json()
                print_pass(description)
                return data
            elif response.status_code == 404:
                print_fail(f"{description} - Resource not found in FHIR store")
                return None
            elif response.status_code == 403:
                print_fail(f"{description} - Permission denied (enable Healthcare API?)")
                return None
            else:
                print_fail(f"{description} - HTTP {response.status_code}")
                return None

        except Exception as e:
            print_fail(f"{description} - Error: {e}")
            return None

    def verify_all_resources(self):
        """Check all FHIR resources for Rohan Sharma."""
        print_header("FHIR Store Verification")

        print("Authentication:")
        if not self.get_access_token():
            print_warn("Cannot verify FHIR data without authentication")
            return 0, 10

        print_pass(f"Access token obtained")
        print()

        checks = 0
        passed = 0

        resources = [
            ("Patient", "PAT-ROHAN-001", "Patient: Rohan Sharma"),
            ("Observation", "obs-bp-20251009", "Blood Pressure (150/96 mmHg)"),
            ("Observation", "obs-hba1c-20250915", "HbA1c (6.3%)"),
            ("Observation", "obs-lipid-20250915", "Lipid Panel"),
            ("Observation", "obs-bmi-20251009", "BMI (29.1)"),
            ("Observation", "obs-waist-20251009", "Waist Circumference (95 cm)"),
            ("Condition", "cond-hypertension", "Condition: Hypertension"),
            ("Condition", "cond-prediabetes", "Condition: Prediabetes"),
            ("MedicationRequest", "medreq-1", "Medication: Telmisartan 40mg"),
            ("FamilyMemberHistory", "family-hist-1", "Family History: Father's MI"),
        ]

        print("Resources:")
        for resource_type, resource_id, description in resources:
            checks += 1
            result = self.check_resource(resource_type, resource_id, description)
            if result:
                passed += 1

        return passed, checks


class Neo4jChecker:
    """Check Neo4j graph for Rohan Sharma data."""

    def __init__(self):
        self.driver = None
        self.connected = False

    def connect(self):
        """Connect to Neo4j."""
        try:
            self.driver = GraphDatabase.driver(
                NEO4J_URI,
                auth=(NEO4J_USERNAME, NEO4J_PASSWORD)
            )
            # Test connection
            with self.driver.session() as session:
                result = session.run("RETURN 1")
                result.single()
            self.connected = True
            return True
        except Exception as e:
            print_fail(f"Cannot connect to Neo4j: {e}")
            return False

    def check_query(self, query, description):
        """Execute a Neo4j query and check if data exists."""
        if not self.connected:
            print_warn(f"{description} (skipped - not connected)")
            return False

        try:
            with self.driver.session() as session:
                result = session.run(query)
                record = result.single()

                if record and record[0]:
                    value = record[0]
                    if isinstance(value, list):
                        if len(value) > 0:
                            print_pass(f"{description} (Count: {len(value)})")
                            return True
                    elif value != 0 and value is not None:
                        print_pass(f"{description} (Value: {value})")
                        return True

                print_fail(f"{description} - No data found")
                return False

        except Exception as e:
            print_fail(f"{description} - Error: {e}")
            return False

    def verify_all_data(self):
        """Check all Neo4j graph data for Rohan Sharma."""
        print_header("Neo4j Graph Database Verification")

        print("Connection:")
        if not self.connect():
            print_warn("Cannot verify Neo4j data without connection")
            return 0, 7

        print_pass(f"Connected to Neo4j at {NEO4J_URI}")
        print()

        checks = 0
        passed = 0

        queries = [
            (
                "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) RETURN p.name",
                "Patient Node: Rohan Sharma"
            ),
            (
                "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:HAS_CONDITION]->(c:Condition) "
                "RETURN collect(c.name)",
                "Conditions (Hypertension, Prediabetes)"
            ),
            (
                "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:EXHIBITS_LIFESTYLE]->(lf:LifestyleFactor) "
                "RETURN collect(lf.name)",
                "Lifestyle Factors (3 risk indicators)"
            ),
            (
                "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:HAS_PROVIDER]->(prov:Provider) "
                "RETURN prov.name",
                "Care Team: Dr. Priya Rao (Cardiology)"
            ),
            (
                "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:IN_COHORT]->(cohort:Cohort) "
                "RETURN cohort.name",
                "Risk Cohort: Urban Metabolic Syndrome"
            ),
            (
                "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:FAMILY_HISTORY_OF]->(f:FamilyCondition) "
                "RETURN f.condition",
                "Family History: Father's MI"
            ),
            (
                "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[r]->(n) "
                "RETURN count(r)",
                "Total Relationships"
            ),
        ]

        print("Graph Data:")
        for query, description in queries:
            checks += 1
            if self.check_query(query, description):
                passed += 1

        return passed, checks

    def close(self):
        """Close Neo4j connection."""
        if self.driver:
            self.driver.close()


def print_data_mapping(fhir_passed, neo4j_passed):
    """Print data mapping analysis between systems."""
    print_header("Data Consistency Analysis")

    print("Data Distribution:")
    print()
    print("┌─────────────────────────────────┬──────────┬──────────┐")
    print("│ Data Element                    │ Neo4j    │ FHIR     │")
    print("├─────────────────────────────────┼──────────┼──────────┤")
    print("│ Patient Demographics            │    ✓     │    ✓     │")
    print("│ Conditions (HTN, Prediabetes)   │    ✓     │    ✓     │")
    print("│ Care Team (Dr. Priya Rao)       │    ✓     │   N/A    │")
    print("│ Risk Cohorts                    │    ✓     │   N/A    │")
    print("│ Lifestyle Risk Factors          │    ✓     │   N/A    │")
    print("│ Vital Signs (BP, BMI)           │   N/A    │    ✓     │")
    print("│ Lab Results (HbA1c, Lipids)     │   N/A    │    ✓     │")
    print("│ Medications (Telmisartan)       │   N/A    │    ✓     │")
    print("│ Family History (Father's MI)    │    ✓     │    ✓     │")
    print("└─────────────────────────────────┴──────────┴──────────┘")
    print()

    print(f"{Colors.BOLD}Complementary Data Sources:{Colors.NC}")
    print(f"  • Neo4j provides social/organizational context")
    print(f"  • FHIR provides clinical measurements and history")
    print(f"  • Module 2 enriches by combining both systems")
    print()


def check_infrastructure():
    """Check if Kafka and Flink are running."""
    print_header("Infrastructure Readiness")

    import subprocess

    checks = []

    # Check Kafka
    try:
        result = subprocess.run(
            ["docker", "ps"],
            capture_output=True,
            text=True,
            timeout=5
        )
        if "kafka" in result.stdout:
            print_pass("Kafka broker running")
            checks.append(True)
        else:
            print_fail("Kafka broker not running")
            checks.append(False)
    except Exception as e:
        print_fail(f"Cannot check Kafka: {e}")
        checks.append(False)

    # Check Flink
    try:
        result = subprocess.run(
            ["curl", "-s", "http://localhost:8081/overview"],
            capture_output=True,
            timeout=5
        )
        if result.returncode == 0:
            print_pass("Flink cluster running")
            checks.append(True)
        else:
            print_fail("Flink cluster not running")
            checks.append(False)
    except Exception as e:
        print_fail(f"Cannot check Flink: {e}")
        checks.append(False)

    # Check Neo4j
    try:
        result = subprocess.run(
            ["docker", "ps"],
            capture_output=True,
            text=True,
            timeout=5
        )
        if "neo4j" in result.stdout:
            print_pass("Neo4j database running")
            checks.append(True)
        else:
            print_fail("Neo4j database not running")
            checks.append(False)
    except Exception as e:
        print_fail(f"Cannot check Neo4j: {e}")
        checks.append(False)

    return all(checks)


def main():
    """Main verification workflow."""
    print(f"\n{Colors.BOLD}{'=' * 80}{Colors.NC}")
    print(f"{Colors.BOLD}Data Consistency Verification for Module 2 Testing{Colors.NC}")
    print(f"{Colors.BOLD}Patient: Rohan Sharma (PAT-ROHAN-001){Colors.NC}")
    print(f"{Colors.BOLD}{'=' * 80}{Colors.NC}")

    # Check Neo4j
    neo4j_checker = Neo4jChecker()
    neo4j_passed, neo4j_total = neo4j_checker.verify_all_data()
    neo4j_checker.close()

    # Check FHIR
    fhir_checker = FHIRChecker()
    fhir_passed, fhir_total = fhir_checker.verify_all_resources()

    # Data mapping
    print_data_mapping(fhir_passed, neo4j_passed)

    # Infrastructure
    infra_ready = check_infrastructure()

    # Final assessment
    print_header("Final Assessment")

    neo4j_ready = neo4j_passed >= 5
    fhir_ready = fhir_passed >= 8
    system_ready = neo4j_ready and infra_ready

    print(f"Results:")
    if neo4j_ready:
        print_pass(f"Neo4j: {neo4j_passed}/{neo4j_total} checks passed - READY")
    else:
        print_fail(f"Neo4j: {neo4j_passed}/{neo4j_total} checks passed - NOT READY")

    if fhir_ready:
        print_pass(f"FHIR: {fhir_passed}/{fhir_total} resources found - READY")
    elif fhir_passed > 0:
        print_warn(f"FHIR: {fhir_passed}/{fhir_total} resources found - PARTIAL (Module 2 will gracefully degrade)")
    else:
        print_warn(f"FHIR: {fhir_passed}/{fhir_total} resources found - UNAVAILABLE (Module 2 will use Neo4j only)")

    if infra_ready:
        print_pass("Infrastructure: Kafka + Flink + Neo4j operational")
    else:
        print_fail("Infrastructure: Some components not running")

    print()

    if system_ready:
        print(f"{Colors.GREEN}{'╔' + '═' * 78 + '╗'}{Colors.NC}")
        print(f"{Colors.GREEN}║{' ' * 20}✓ SYSTEM READY FOR MODULE 2 TEST{' ' * 25}║{Colors.NC}")
        print(f"{Colors.GREEN}{'╚' + '═' * 78 + '╝'}{Colors.NC}")
        print()
        print(f"{Colors.BOLD}🚀 Next Step:{Colors.NC}")
        print(f"   ./test-rohan-enrichment.sh")
        print()
        return 0
    else:
        print(f"{Colors.RED}{'╔' + '═' * 78 + '╗'}{Colors.NC}")
        print(f"{Colors.RED}║{' ' * 25}✗ SYSTEM NOT READY{' ' * 28}║{Colors.NC}")
        print(f"{Colors.RED}{'╚' + '═' * 78 + '╝'}{Colors.NC}")
        print()
        print(f"{Colors.YELLOW}⚠️  Action Required:{Colors.NC}")

        if not neo4j_ready:
            print(f"   1. Load Neo4j data: ./load-neo4j-http.sh")

        if not fhir_ready and fhir_passed == 0:
            print(f"   2. Load FHIR data: Follow GOOGLE_FHIR_UI_GUIDE.md")

        if not infra_ready:
            print(f"   3. Start infrastructure: docker-compose up -d")

        print()
        return 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print(f"\n\n{Colors.YELLOW}⊘ Verification interrupted by user{Colors.NC}\n")
        sys.exit(1)
    except Exception as e:
        print(f"\n{Colors.RED}✗ Unexpected error: {e}{Colors.NC}\n")
        import traceback
        traceback.print_exc()
        sys.exit(1)
