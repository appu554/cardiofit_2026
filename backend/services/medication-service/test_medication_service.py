import requests
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
BASE_URL = "http://localhost:8018/api"
TEST_TOKEN = "test_token"

# Headers
headers = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {TEST_TOKEN}"
}

def test_health_check():
    """Test the health check endpoint"""
    try:
        response = requests.get(f"http://localhost:8018/health")
        logger.info(f"Health check response: {response.status_code} - {response.text}")
        if response.status_code == 200:
            data = response.json()
            if data.get("status") == "healthy":
                logger.info("✅ Health check passed")
                return True
        logger.error("❌ Health check failed")
        return False
    except Exception as e:
        logger.error(f"Error checking health: {str(e)}")
        return False

def test_create_medication():
    """Test creating a medication"""
    payload = {
        "code": {
            "coding": [
                {
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                    "code": "1049502",
                    "display": "Acetaminophen 325 MG Oral Tablet"
                }
            ],
            "text": "Acetaminophen 325 MG Oral Tablet"
        }
    }

    try:
        logger.info(f"Sending POST request to {BASE_URL}/medications/")
        logger.info(f"Headers: {headers}")
        logger.info(f"Payload: {json.dumps(payload, indent=2)}")

        response = requests.post(f"{BASE_URL}/medications/", headers=headers, json=payload)
        logger.info(f"Response: {response.status_code} - {response.text}")

        if response.status_code == 201:
            data = response.json()
            medication_id = data.get("id")
            logger.info(f"✅ Created medication with ID: {medication_id}")
            return medication_id
        else:
            logger.error(f"❌ Failed to create medication: {response.status_code} - {response.text}")
            return None
    except Exception as e:
        logger.error(f"Error creating medication: {str(e)}")
        return None

def test_get_medication(medication_id):
    """Test getting a medication by ID"""
    response = requests.get(f"{BASE_URL}/medications/{medication_id}", headers=headers)
    assert response.status_code == 200
    data = response.json()
    assert data["resourceType"] == "Medication"
    assert data["id"] == medication_id
    print(f"✅ Retrieved medication with ID: {medication_id}")

def test_update_medication(medication_id):
    """Test updating a medication"""
    payload = {
        "status": "active",
        "code": {
            "coding": [
                {
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                    "code": "1049502",
                    "display": "Acetaminophen 325 MG Oral Tablet [Updated]"
                }
            ],
            "text": "Acetaminophen 325 MG Oral Tablet [Updated]"
        }
    }

    response = requests.put(f"{BASE_URL}/medications/{medication_id}", headers=headers, json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["resourceType"] == "Medication"
    assert data["id"] == medication_id
    assert data["code"]["text"] == "Acetaminophen 325 MG Oral Tablet [Updated]"
    print(f"✅ Updated medication with ID: {medication_id}")

def test_create_medication_request(patient_id="123"):
    """Test creating a medication request"""
    payload = {
        "status": "active",
        "intent": "order",
        "medicationCodeableConcept": {
            "coding": [
                {
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                    "code": "1049502",
                    "display": "Acetaminophen 325 MG Oral Tablet"
                }
            ],
            "text": "Acetaminophen 325 MG Oral Tablet"
        },
        "subject": {
            "reference": f"Patient/{patient_id}"
        },
        "authoredOn": datetime.now().isoformat(),
        "dosageInstruction": [
            {
                "text": "Take 1 tablet by mouth every 4-6 hours as needed for pain",
                "timing": {
                    "code": {
                        "text": "Every 4-6 hours as needed"
                    }
                }
            }
        ]
    }

    response = requests.post(f"{BASE_URL}/medication-requests", headers=headers, json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["resourceType"] == "MedicationRequest"
    assert data["status"] == "active"
    assert data["intent"] == "order"
    assert data["subject"]["reference"] == f"Patient/{patient_id}"

    medication_request_id = data["id"]
    print(f"✅ Created medication request with ID: {medication_request_id}")
    return medication_request_id

def test_get_patient_medication_requests(patient_id="123"):
    """Test getting medication requests for a patient"""
    response = requests.get(f"{BASE_URL}/medication-requests/patient/{patient_id}", headers=headers)
    assert response.status_code == 200
    data = response.json()
    assert isinstance(data, list)
    if len(data) > 0:
        assert data[0]["resourceType"] == "MedicationRequest"
        assert data[0]["subject"]["reference"] == f"Patient/{patient_id}"
    print(f"✅ Retrieved medication requests for patient: {patient_id}")

def test_create_medication_administration(patient_id="123", medication_request_id=None):
    """Test creating a medication administration"""
    payload = {
        "status": "completed",
        "medicationCodeableConcept": {
            "coding": [
                {
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                    "code": "1049502",
                    "display": "Acetaminophen 325 MG Oral Tablet"
                }
            ],
            "text": "Acetaminophen 325 MG Oral Tablet"
        },
        "subject": {
            "reference": f"Patient/{patient_id}"
        },
        "effectiveDateTime": datetime.now().isoformat(),
        "performer": [
            {
                "actor": {
                    "reference": "Practitioner/456",
                    "display": "Dr. Jane Smith"
                }
            }
        ],
        "dosage": {
            "text": "1 tablet",
            "dose": {
                "value": 1,
                "unit": "tablet"
            }
        }
    }

    if medication_request_id:
        payload["request"] = {
            "reference": f"MedicationRequest/{medication_request_id}"
        }

    response = requests.post(f"{BASE_URL}/medication-administrations", headers=headers, json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["resourceType"] == "MedicationAdministration"
    assert data["status"] == "completed"
    assert data["subject"]["reference"] == f"Patient/{patient_id}"

    medication_administration_id = data["id"]
    print(f"✅ Created medication administration with ID: {medication_administration_id}")
    return medication_administration_id

def test_process_hl7_message():
    """Test processing an HL7 RDE message"""
    # Sample RDE message
    hl7_message = """MSH|^~\\&|SENDING_APPLICATION|SENDING_FACILITY|RECEIVING_APPLICATION|RECEIVING_FACILITY|20230615080000||RDE^O11|MSGID123|P|2.5.1|
PID|||123^^^MRN||DOE^JOHN||19700101|M||
ORC|NW|ORDER123||||||20230615080000|||DOCTOR^JOHN^A|
RXE||1049502^Acetaminophen 325 MG Oral Tablet^RXNORM|1|TAB|Q4-6H PRN||||||
RXR|PO||
TQ1|||Q4-6H PRN|||20230615080000|||"""

    payload = {
        "message": hl7_message
    }

    response = requests.post(f"{BASE_URL}/hl7/rde", headers=headers, json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "success"
    assert data["message_type"] == "RDE"
    assert "resources_created" in data
    print("✅ Processed HL7 RDE message")

def run_tests():
    """Run all tests"""
    logger.info("Starting medication service tests...")

    try:
        # Basic health check
        if not test_health_check():
            logger.error("Health check failed, skipping other tests")
            return

        logger.info("\nRunning tests with authentication...")

        # Medication tests
        medication_id = test_create_medication()
        if not medication_id:
            logger.error("Failed to create medication, skipping other tests")
            return

        logger.info("\nAll tests completed successfully!")

    except Exception as e:
        logger.error(f"\nTest failed: {str(e)} ❌")
        raise

if __name__ == "__main__":
    run_tests()
