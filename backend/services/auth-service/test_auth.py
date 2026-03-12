import requests
import json
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def test_auth_service_health():
    """Test the health endpoint of the auth service."""
    try:
        response = requests.get("http://localhost:8001/health")
        logger.info(f"Health check status code: {response.status_code}")
        
        if response.status_code == 200:
            logger.info("✅ Auth service health check passed")
            return True
        else:
            logger.error(f"❌ Auth service health check failed: {response.text}")
            return False
    except requests.RequestException as e:
        logger.error(f"❌ Auth service health check failed: {str(e)}")
        return False

def main():
    """Main function to run the tests."""
    logger.info("Starting Auth Service tests")
    
    # Test health endpoint
    if not test_auth_service_health():
        logger.error("Auth service is not running or not responding")
        return
    
    logger.info("Auth service tests completed")

if __name__ == "__main__":
    main()
