#!/usr/bin/env python3
"""
Test CAE via HTTPS endpoint
"""

import json
import logging
import requests
import time
from datetime import datetime

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_cae_https():
    """Test CAE via HTTPS endpoint"""
    
    logger.info("🚀 Testing CAE via HTTPS endpoint")
    logger.info("=" * 80)
    
    # Endpoint URL - adjust as needed
    url = "http://localhost:8000/api/clinical-assertions"
    
    # Use the elderly cardiovascular patient data from comprehensive tests
    payload = {
        "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
        "correlation_id": "test_correlation_123",
        "medication_ids": ["warfarin", "aspirin", "lisinopril", "metoprolol", "metformin", "atorvastatin"],
        "condition_ids": ["atrial_fibrillation", "hypertension", "diabetes", "coronary_artery_disease"],
        "patient_context": {
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "demographics": {
                "age": 67,
                "gender": "male",
                "weight_kg": 78.5
            },
            "allergy_ids": [],
            "metadata": {},
            "context_version": "",
            "assembly_time": "0001-01-01T00:00:00Z"
        },
        "priority": "PRIORITY_STANDARD",
        "reasoner_types": ["interaction", "contraindication", "duplicate_therapy", "dosing"]
    }
    
    headers = {
        'Content-Type': 'application/json'
    }
    
    try:
        logger.info(f"Sending request to {url}...")
        logger.info(f"Request payload: {json.dumps(payload, indent=2)}")
        
        start_time = time.time()
        response = requests.post(url, headers=headers, json=payload, timeout=30)
        elapsed_time = time.time() - start_time
        
        logger.info(f"Response received in {elapsed_time:.2f} seconds")
        logger.info(f"Status code: {response.status_code}")
        
        if response.status_code == 200:
            response_data = response.json()
            logger.info(f"Response data: {json.dumps(response_data, indent=2)}")
            
            # Check for assertions in the response
            assertions = response_data.get("assertions", [])
            logger.info(f"Received {len(assertions)} clinical assertions")
            
            for i, assertion in enumerate(assertions):
                logger.info(f"Assertion {i+1}: {assertion.get('assertion_type')} - {assertion.get('description')}")
            
            logger.info("✅ HTTPS test completed successfully")
            return True
        else:
            logger.error(f"Error response: {response.text}")
            logger.error("❌ HTTPS test failed")
            return False
            
    except Exception as e:
        logger.error(f"❌ Exception during HTTPS test: {e}")
        return False

if __name__ == "__main__":
    test_cae_https()
