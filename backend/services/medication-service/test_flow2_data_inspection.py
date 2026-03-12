"""
Flow 2 Data Inspection Test

This test shows exactly what data is being retrieved for each data point
in the Flow 2 context assembly process.
"""

import requests
import json
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Test patient ID
PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def inspect_flow2_data():
    """
    Inspect the detailed data returned by Flow 2 context assembly
    """
    try:
        logger.info("🔍 Inspecting Flow 2 Data Retrieval")
        logger.info("=" * 80)
        
        # Context Service GraphQL endpoint
        context_url = "http://localhost:8016/graphql"
        
        # GraphQL query to get detailed context data
        query = """
        query GetDetailedContext($patientId: String!, $recipeName: String!) {
            assembleContext(patientId: $patientId, recipeName: $recipeName) {
                contextId
                patientId
                recipeName
                status
                completenessScore
                assembledAt
                assembledData
                safetyFlags {
                    flagType
                    severity
                    message
                    dataPoint
                    details
                }
                sourceMetadata {
                    sourceType
                    sourceEndpoint
                    retrievedAt
                    completeness
                    responseTimeMs
                    cacheHit
                    errorMessage
                }
                cacheInfo {
                    cacheHit
                    cacheKey
                    ttlSeconds
                }
            }
        }
        """
        
        variables = {
            "patientId": PATIENT_ID,
            "recipeName": "medication_safety_base_context_v2"
        }
        
        payload = {
            "query": query,
            "variables": variables
        }
        
        logger.info(f"📡 Making GraphQL request to: {context_url}")
        
        response = requests.post(
            context_url,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "errors" in data:
                logger.error("❌ GraphQL Errors:")
                for error in data["errors"]:
                    logger.error(f"   - {error}")
                return None
            
            context_data = data.get("data", {}).get("assembleContext", {})
            
            logger.info("=" * 80)
            logger.info("📊 DETAILED DATA INSPECTION")
            logger.info("=" * 80)
            
            # Show assembled data in detail
            assembled_data = context_data.get("assembledData", {})
            
            for data_point_name, data_content in assembled_data.items():
                logger.info(f"🔹 {data_point_name.upper()}:")
                logger.info(f"   Type: {type(data_content)}")
                
                if isinstance(data_content, dict):
                    logger.info(f"   Keys: {list(data_content.keys())}")
                    
                    # Show detailed content for each data point
                    if data_point_name == "patient_demographics":
                        logger.info("   📋 DEMOGRAPHICS DETAILS:")
                        for key, value in data_content.items():
                            if key in ["name", "gender", "birthDate", "age", "weight", "height"]:
                                logger.info(f"      {key}: {value}")
                    
                    elif data_point_name == "patient_allergies":
                        logger.info("   🚨 ALLERGIES DETAILS:")
                        for key, value in data_content.items():
                            logger.info(f"      {key}: {value}")
                        
                        # Show allergy list if present
                        allergies = data_content.get("allergies", [])
                        if isinstance(allergies, list):
                            logger.info(f"      Allergy Count: {len(allergies)}")
                            for i, allergy in enumerate(allergies[:3]):  # Show first 3
                                logger.info(f"      Allergy {i+1}: {allergy}")
                    
                    elif data_point_name == "current_medications":
                        logger.info("   💊 MEDICATIONS DETAILS:")
                        for key, value in data_content.items():
                            if key == "medication_requests" and isinstance(value, list):
                                logger.info(f"      {key}: {len(value)} medications")
                                for i, med in enumerate(value[:2]):  # Show first 2
                                    if isinstance(med, dict):
                                        med_name = "Unknown"
                                        if "medicationCodeableConcept" in med:
                                            coding = med["medicationCodeableConcept"].get("coding", [])
                                            if coding and isinstance(coding, list):
                                                med_name = coding[0].get("display", "Unknown")
                                        logger.info(f"         Med {i+1}: {med_name}")
                                        logger.info(f"         Status: {med.get('status', 'Unknown')}")
                            else:
                                logger.info(f"      {key}: {value}")
                    
                    else:
                        # Show first few key-value pairs for other data points
                        shown = 0
                        for key, value in data_content.items():
                            if shown < 5:
                                logger.info(f"      {key}: {value}")
                                shown += 1
                            else:
                                logger.info(f"      ... ({len(data_content) - shown} more fields)")
                                break
                
                elif isinstance(data_content, list):
                    logger.info(f"   Length: {len(data_content)}")
                    if len(data_content) > 0:
                        logger.info(f"   Sample: {data_content[0]}")
                else:
                    logger.info(f"   Value: {data_content}")
                
                logger.info("")
            
            # Show safety flags in detail
            safety_flags = context_data.get("safetyFlags", [])
            logger.info("🚨 SAFETY FLAGS DETAILS:")
            for flag in safety_flags:
                if isinstance(flag, dict):
                    flag_type = flag.get("flagType", "Unknown")
                    severity = flag.get("severity", "Unknown")
                    message = flag.get("message", "No message")
                    data_point = flag.get("dataPoint", "Unknown")
                    
                    logger.info(f"   🔸 {severity}: {flag_type}")
                    logger.info(f"      Message: {message}")
                    logger.info(f"      Data Point: {data_point}")
                    
                    details = flag.get("details")
                    if details and isinstance(details, dict):
                        logger.info(f"      Details: {details}")
                    logger.info("")
            
            return context_data
        else:
            logger.error(f"❌ GraphQL request failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except Exception as e:
        logger.error(f"❌ Error inspecting Flow 2 data: {str(e)}")
        return None

def test_individual_services():
    """
    Test individual services to see what data they return
    """
    logger.info("=" * 80)
    logger.info("🔧 INDIVIDUAL SERVICE TESTING")
    logger.info("=" * 80)
    
    # Test Medication Service for allergies (using public endpoint)
    logger.info("🧪 Testing Medication Service - Allergies (Public Endpoint):")
    try:
        allergy_url = f"http://localhost:8009/api/public/allergies/patient/{PATIENT_ID}"
        response = requests.get(allergy_url, timeout=10)
        logger.info(f"   Status: {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            logger.info(f"   Count: {data.get('count', 'Unknown')}")
            logger.info(f"   Source: {data.get('source', 'Unknown')}")
            logger.info(f"   Note: {data.get('note', 'No note')}")

            allergies = data.get('allergies', [])
            if isinstance(allergies, list) and len(allergies) > 0:
                logger.info(f"   Allergies: {len(allergies)} found")
                for i, allergy in enumerate(allergies[:2]):
                    if isinstance(allergy, dict):
                        allergen = "Unknown"
                        if "code" in allergy:
                            code = allergy["code"]
                            if isinstance(code, dict):
                                coding = code.get("coding", [])
                                if coding and isinstance(coding, list):
                                    allergen = coding[0].get("display", "Unknown")
                                elif "text" in code:
                                    allergen = code["text"]
                        logger.info(f"      Allergy {i+1}: {allergen}")
            else:
                logger.info("   No allergies found")
        else:
            logger.info(f"   Error: {response.text}")
    except Exception as e:
        logger.error(f"   Exception: {e}")
    
    logger.info("")
    
    # Test Medication Service for medications
    logger.info("🧪 Testing Medication Service - Medications:")
    try:
        med_url = f"http://localhost:8009/api/public/medication-requests/patient/{PATIENT_ID}"
        response = requests.get(med_url, timeout=10)
        logger.info(f"   Status: {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            logger.info(f"   Count: {data.get('count', 'Unknown')}")
            med_requests = data.get('medication_requests', [])
            logger.info(f"   Medications: {len(med_requests)} found")
            for i, med in enumerate(med_requests[:2]):
                if isinstance(med, dict):
                    status = med.get('status', 'Unknown')
                    logger.info(f"      Med {i+1}: Status={status}")
        else:
            logger.info(f"   Error: {response.text}")
    except Exception as e:
        logger.error(f"   Exception: {e}")
    
    logger.info("")
    
    # Test Observation Service for vital signs
    logger.info("🧪 Testing Observation Service - Vital Signs:")
    try:
        obs_url = "http://localhost:8007/api/federation"
        query = """
        query GetVitalSigns($patientId: String!) {
            observations(patientId: $patientId, count: 10) {
                id
                code { text }
                valueQuantity { value unit }
            }
        }
        """
        payload = {
            "query": query,
            "variables": {"patientId": PATIENT_ID}
        }
        response = requests.post(obs_url, json=payload, timeout=10)
        logger.info(f"   Status: {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            observations = data.get("data", {}).get("observations", [])
            logger.info(f"   Observations: {len(observations)} found")
            for obs in observations[:3]:
                code_text = obs.get("code", {}).get("text", "Unknown")
                value_qty = obs.get("valueQuantity", {})
                if value_qty:
                    value = value_qty.get("value", "Unknown")
                    unit = value_qty.get("unit", "")
                    logger.info(f"      {code_text}: {value} {unit}")
        else:
            logger.info(f"   Error: {response.text}")
    except Exception as e:
        logger.error(f"   Exception: {e}")

def main():
    """Main test function"""
    logger.info("🚀 Flow 2 Data Inspection Test")
    logger.info("🎯 Detailed analysis of what data is being retrieved")
    logger.info("=" * 80)
    
    try:
        # Step 1: Inspect Flow 2 context data
        context_data = inspect_flow2_data()
        
        # Step 2: Test individual services
        test_individual_services()
        
        logger.info("=" * 80)
        logger.info("🎉 DATA INSPECTION COMPLETE")
        logger.info("=" * 80)
        
        if context_data:
            completeness = context_data.get("completenessScore", 0)
            status = context_data.get("status", "Unknown")
            logger.info(f"✅ Flow 2 Status: {status}")
            logger.info(f"✅ Completeness: {completeness}%")
            return 0
        else:
            logger.error("❌ Could not retrieve context data")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
