import requests
import json
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
BASE_URL = "http://localhost:8018/api"

# Try different FHIR URLs
FHIR_URLS = [
    "http://localhost:8004/api",  # Standard API path
    "http://localhost:8004",      # Root path
    "http://127.0.0.1:8004/api",  # Using IP instead of localhost
    "http://127.0.0.1:8004"       # IP with root path
]
FHIR_URL = FHIR_URLS[0]  # Default to the first URL

TEST_TOKEN = "test_token"

# Headers
headers = {
    "Content-Type": "application/json",
    "Authorization": f"Bearer {TEST_TOKEN}"
}

def test_health():
    """Test the health endpoint"""
    try:
        response = requests.get(f"http://localhost:8018/health")
        logger.info(f"Health check response: {response.status_code} - {response.text}")
        return response.status_code == 200
    except Exception as e:
        logger.error(f"Error checking health: {str(e)}")
        return False

def test_fhir_health():
    """Test the FHIR server health endpoint by trying different URLs"""
    global FHIR_URL

    for url in FHIR_URLS:
        try:
            # Try with /health endpoint
            health_url = f"{url}/health"
            logger.info(f"Attempting to connect to FHIR server at {health_url}")
            response = requests.get(health_url, timeout=5)  # 5 second timeout

            if response.status_code == 200:
                logger.info(f"FHIR health check response: {response.status_code} - {response.text}")
                FHIR_URL = url  # Set the working URL as the default
                return True
            else:
                logger.warning(f"FHIR health check failed with status {response.status_code} at {health_url}")

            # Try without /health endpoint
            logger.info(f"Attempting to connect to FHIR server at {url}")
            response = requests.get(url, timeout=5)  # 5 second timeout

            if response.status_code == 200:
                logger.info(f"FHIR server response: {response.status_code} - {response.text[:100]}...")
                FHIR_URL = url  # Set the working URL as the default
                return True
            else:
                logger.warning(f"FHIR server check failed with status {response.status_code} at {url}")

        except requests.exceptions.Timeout:
            logger.warning(f"Timeout connecting to FHIR server at {url}")
        except requests.exceptions.ConnectionError:
            logger.warning(f"Connection error connecting to FHIR server at {url}")
        except Exception as e:
            logger.warning(f"Error checking FHIR health at {url}: {str(e)}")

    # If we get here, all URLs failed
    logger.error("All FHIR server URLs failed to connect")
    return False

def test_create_medication():
    """Test creating a medication with detailed logging"""
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

    logger.info(f"Sending POST request to {BASE_URL}/medications")
    logger.info(f"Headers: {headers}")
    logger.info(f"Payload: {json.dumps(payload, indent=2)}")

    try:
        # First, try without following redirects
        response = requests.post(
            f"{BASE_URL}/medications",
            headers=headers,
            json=payload,
            allow_redirects=False
        )

        logger.info(f"Initial response (no redirect): {response.status_code} - {response.text}")

        if response.status_code == 307:
            redirect_url = response.headers.get('Location')
            logger.info(f"Redirect URL: {redirect_url}")

            # Try the redirect URL directly
            response = requests.post(
                redirect_url,
                headers=headers,
                json=payload
            )

            logger.info(f"Redirect response: {response.status_code} - {response.text}")

        # Now try with following redirects
        response = requests.post(
            f"{BASE_URL}/medications",
            headers=headers,
            json=payload,
            allow_redirects=True
        )

        logger.info(f"Response with redirect: {response.status_code} - {response.text}")

        if response.status_code == 201:
            data = response.json()
            logger.info(f"Created medication with ID: {data.get('id')}")
            return data.get('id')
        else:
            logger.error(f"Failed to create medication: {response.text}")
            return None
    except Exception as e:
        logger.error(f"Exception during medication creation: {str(e)}")
        return None

def test_create_medication_with_status():
    """Test creating a medication with status field"""
    payload = {
        "status": "active",
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

    logger.info(f"Sending POST request to {BASE_URL}/medications with status")
    logger.info(f"Headers: {headers}")
    logger.info(f"Payload: {json.dumps(payload, indent=2)}")

    try:
        response = requests.post(
            f"{BASE_URL}/medications/",
            headers=headers,
            json=payload
        )

        logger.info(f"Response: {response.status_code} - {response.text}")

        if response.status_code == 201:
            data = response.json()
            logger.info(f"Created medication with ID: {data.get('id')}")
            return data.get('id')
        else:
            logger.error(f"Failed to create medication: {response.text}")
            return None
    except Exception as e:
        logger.error(f"Exception during medication creation: {str(e)}")
        return None

if __name__ == "__main__":
    logger.info("Starting debug tests...")

    # Test health endpoint
    if test_health():
        logger.info("Health check passed!")
    else:
        logger.error("Health check failed!")
        logger.info("Exiting tests since medication service is not healthy.")
        exit(1)

    logger.info("\nTesting FHIR server connection...")
    # Test FHIR health endpoint
    fhir_healthy = test_fhir_health()
    if fhir_healthy:
        logger.info("FHIR health check passed!")
    else:
        logger.error("FHIR health check failed! Make sure the FHIR server is running.")
        logger.info("Continuing with tests anyway, but they will likely fail...")

    logger.info("\nTesting medication creation...")
    # Test creating a medication
    try:
        medication_id = test_create_medication()
        if medication_id:
            logger.info(f"Test passed! Created medication with ID: {medication_id}")
        else:
            logger.info("\nTrying alternative approach with status field...")
            medication_id = test_create_medication_with_status()
            if medication_id:
                logger.info(f"Alternative test passed! Created medication with ID: {medication_id}")
            else:
                logger.error("All tests failed!")
    except Exception as e:
        logger.error(f"Unexpected error during tests: {str(e)}")

    logger.info("\nDebug tests completed.")
