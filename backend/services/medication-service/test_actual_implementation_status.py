"""
Test Actual Implementation Status
Check what's really implemented vs what's planned
"""

import requests
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_basic_endpoints():
    """Test basic endpoints that should be working"""
    logger.info("🔍 Testing Basic Endpoints")
    
    base_url = "http://localhost:8009"
    
    # Test 1: Health check
    try:
        response = requests.get(f"{base_url}/health", timeout=5)
        logger.info(f"   Health Check: {response.status_code}")
    except Exception as e:
        logger.error(f"   Health Check Failed: {e}")
    
    # Test 2: Flow 2 health check
    try:
        response = requests.get(f"{base_url}/api/flow2/medication-safety/health", timeout=5)
        logger.info(f"   Flow 2 Health: {response.status_code}")
    except Exception as e:
        logger.error(f"   Flow 2 Health Failed: {e}")
    
    # Test 3: Workflow proposals endpoint
    try:
        test_payload = {
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "medication_code": "313782",
            "medication_name": "Acetaminophen",
            "dosage": "500mg",
            "frequency": "every 6 hours",
            "duration": "5 days",
            "route": "oral",
            "indication": "pain management",
            "provider_id": "test-provider",
            "encounter_id": "test-encounter",
            "priority": "routine"
        }
        
        response = requests.post(
            f"{base_url}/api/proposals/medication",
            json=test_payload,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        logger.info(f"   Workflow Proposals: {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            logger.info(f"   Proposal ID: {data.get('proposal_id', 'Unknown')}")
        elif response.status_code == 401:
            logger.info("   Workflow Proposals: Needs authentication (expected)")
        else:
            logger.info(f"   Response: {response.text[:200]}")
    except Exception as e:
        logger.error(f"   Workflow Proposals Failed: {e}")

def test_flow2_validation():
    """Test Flow 2 validation endpoint"""
    logger.info("🔍 Testing Flow 2 Validation")
    
    base_url = "http://localhost:8009"
    
    try:
        test_payload = {
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "medication": {
                "code": "313782",
                "name": "Acetaminophen",
                "dosage": "500mg",
                "frequency": "every 6 hours",
                "route": "oral"
            },
            "indication": "pain management",
            "prescriber": {
                "provider_id": "test-provider"
            },
            "urgency": "routine"
        }
        
        response = requests.post(
            f"{base_url}/api/flow2/medication-safety/validate",
            json=test_payload,
            headers={"Content-Type": "application/json"},
            timeout=15
        )
        logger.info(f"   Flow 2 Validation: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            logger.info(f"   Validation Status: {data.get('status', 'Unknown')}")
            logger.info(f"   Context Quality: {data.get('context_quality', 'Unknown')}")
            logger.info(f"   Recommendations: {len(data.get('recommendations', []))}")
        elif response.status_code == 401:
            logger.info("   Flow 2 Validation: Needs authentication (expected)")
        else:
            logger.info(f"   Response: {response.text[:200]}")
            
    except Exception as e:
        logger.error(f"   Flow 2 Validation Failed: {e}")

def test_advanced_services_import():
    """Test if advanced services can be imported"""
    logger.info("🔍 Testing Advanced Services Import")
    
    services_to_test = [
        "app.domain.services.dose_calculation_service",
        "app.domain.services.advanced_pharmacokinetics_service", 
        "app.domain.services.formulary_management_service",
        "app.domain.services.pharmacogenomics_service",
        "app.domain.services.therapeutic_drug_monitoring_service",
        "app.application.services.medication_proposal_service"
    ]
    
    for service in services_to_test:
        try:
            __import__(service)
            logger.info(f"   ✅ {service}: Available")
        except ImportError as e:
            logger.error(f"   ❌ {service}: Not available - {e}")
        except Exception as e:
            logger.warning(f"   ⚠️ {service}: Import issue - {e}")

def main():
    """Main test function"""
    logger.info("🚀 Actual Implementation Status Check")
    logger.info("🎯 Testing what's really working vs what's planned")
    logger.info("=" * 70)
    
    try:
        # Test basic endpoints
        test_basic_endpoints()
        logger.info("")
        
        # Test Flow 2 validation
        test_flow2_validation()
        logger.info("")
        
        # Test advanced services import
        test_advanced_services_import()
        
        logger.info("=" * 70)
        logger.info("🎉 Implementation Status Check Complete")
        logger.info("✅ Check logs above to see what's actually working")
        
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1
    
    return 0

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
