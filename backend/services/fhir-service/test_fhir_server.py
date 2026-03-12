import requests
import json
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
BASE_URL = "http://localhost:8004"
TEST_TOKEN = "test_token"

# Headers
headers = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {TEST_TOKEN}"
}

def test_health():
    """Test the health endpoint"""
    try:
        response = requests.get(f"{BASE_URL}/health")
        logger.info(f"Health check response: {response.status_code} - {response.text}")
        return response.status_code == 200
    except Exception as e:
        logger.error(f"Error checking health: {str(e)}")
        return False

def test_api_endpoints():
    """Test various API endpoints to see which ones work"""
    endpoints = [
        "/api/fhir/Patient",
        "/fhir/Patient",
        "/api/Patient",
        "/Patient"
    ]
    
    for endpoint in endpoints:
        try:
            url = f"{BASE_URL}{endpoint}"
            logger.info(f"Testing endpoint: {url}")
            response = requests.get(url, headers=headers)
            logger.info(f"Response: {response.status_code} - {response.text[:100]}...")
        except Exception as e:
            logger.error(f"Error testing {url}: {str(e)}")

def test_create_medication():
    """Test creating a medication"""
    endpoints = [
        "/api/fhir/Medication",
        "/fhir/Medication",
        "/api/Medication",
        "/Medication"
    ]
    
    payload = {
        "resourceType": "Medication",
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
    
    for endpoint in endpoints:
        try:
            url = f"{BASE_URL}{endpoint}"
            logger.info(f"Testing POST to endpoint: {url}")
            logger.info(f"Payload: {json.dumps(payload, indent=2)}")
            response = requests.post(url, headers=headers, json=payload)
            logger.info(f"Response: {response.status_code} - {response.text[:100]}...")
        except Exception as e:
            logger.error(f"Error testing POST to {url}: {str(e)}")

if __name__ == "__main__":
    logger.info("Starting FHIR server tests...")
    
    # Test health endpoint
    if test_health():
        logger.info("Health check passed!")
    else:
        logger.error("Health check failed!")
    
    # Test API endpoints
    logger.info("\nTesting API endpoints...")
    test_api_endpoints()
    
    # Test creating a medication
    logger.info("\nTesting medication creation...")
    test_create_medication()
