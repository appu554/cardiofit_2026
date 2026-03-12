"""
Test Patient Observations for Demographics

This test checks what observations exist for our test patient and creates
weight/height observations if they don't exist.
"""

import requests
import json
import logging
from datetime import datetime, timedelta

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Test patient ID
PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def get_test_auth_headers():
    """
    Get authentication headers for testing
    """
    return {
        "Content-Type": "application/json",
        "Accept": "application/json",
        # Add test authentication headers that the HeaderAuthMiddleware expects
        "X-User-ID": "test-user-123",
        "X-User-Role": "doctor",
        "X-User-Roles": "doctor,admin",
        "X-User-Permissions": "observation:read,observation:write,patient:read,patient:write"
    }

def check_existing_observations():
    """
    Check what observations exist for our test patient
    """
    try:
        logger.info("🔍 Checking existing observations for patient")
        logger.info(f"   Patient ID: {PATIENT_ID}")

        # Try to get observations from Observation Service (using public endpoint)
        observation_url = f"http://localhost:8007/api/public/observations/patient/{PATIENT_ID}"

        logger.info(f"📡 Checking: {observation_url}")
        logger.info("   Using public endpoint (bypasses authentication)")

        response = requests.get(observation_url, timeout=10)
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            logger.info("✅ Successfully retrieved observations")
            logger.info(f"   Response type: {type(data)}")
            
            if isinstance(data, dict):
                observations = data.get('observations', [])
                logger.info(f"   Observations found: {len(observations)}")
            elif isinstance(data, list):
                observations = data
                logger.info(f"   Observations found: {len(observations)}")
            else:
                observations = []
                logger.info("   No observations structure found")
            
            # Analyze observations
            weight_obs = []
            height_obs = []
            other_obs = []
            
            for obs in observations:
                if isinstance(obs, dict):
                    obs_type = classify_observation(obs)
                    if obs_type == 'weight':
                        weight_obs.append(obs)
                    elif obs_type == 'height':
                        height_obs.append(obs)
                    else:
                        other_obs.append(obs)
            
            logger.info("📊 OBSERVATION ANALYSIS:")
            logger.info(f"   Weight observations: {len(weight_obs)}")
            logger.info(f"   Height observations: {len(height_obs)}")
            logger.info(f"   Other observations: {len(other_obs)}")
            
            # Show details of weight/height observations
            if weight_obs:
                logger.info("   Weight details:")
                for obs in weight_obs:
                    value = extract_observation_value(obs)
                    logger.info(f"     - {value}")
            
            if height_obs:
                logger.info("   Height details:")
                for obs in height_obs:
                    value = extract_observation_value(obs)
                    logger.info(f"     - {value}")
            
            return {
                'weight_observations': weight_obs,
                'height_observations': height_obs,
                'other_observations': other_obs,
                'total_observations': len(observations)
            }
            
        elif response.status_code == 401:
            logger.warning("⚠️ Observation service requires authentication")
            return None
        elif response.status_code == 404:
            logger.info("ℹ️ No observations found for patient (404)")
            return {
                'weight_observations': [],
                'height_observations': [],
                'other_observations': [],
                'total_observations': 0
            }
        else:
            logger.error(f"❌ Observation service returned {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except requests.exceptions.ConnectionError:
        logger.error("❌ Cannot connect to Observation Service on port 8007")
        logger.error("🔧 Make sure Observation Service is running")
        return None
    except Exception as e:
        logger.error(f"❌ Error checking observations: {str(e)}")
        return None

def classify_observation(observation):
    """
    Classify observation as weight, height, or other
    """
    code = observation.get('code', {})
    
    # Check LOINC codes
    if isinstance(code, dict) and 'coding' in code:
        for coding in code['coding']:
            if isinstance(coding, dict):
                loinc_code = coding.get('code', '')
                if loinc_code in ['29463-7', '3141-9']:  # Body weight
                    return 'weight'
                elif loinc_code in ['8302-2', '8306-3']:  # Body height
                    return 'height'
    
    # Check display text
    display = code.get('text', '').lower()
    if any(keyword in display for keyword in ['weight', 'body weight', 'mass']):
        return 'weight'
    elif any(keyword in display for keyword in ['height', 'body height', 'stature']):
        return 'height'
    
    return 'other'

def extract_observation_value(observation):
    """
    Extract value and unit from observation
    """
    try:
        value_quantity = observation.get('valueQuantity', {})
        if isinstance(value_quantity, dict):
            value = value_quantity.get('value')
            unit = value_quantity.get('unit', value_quantity.get('code', ''))
            return f"{value} {unit}"
    except Exception:
        pass
    
    return "Unknown value"

def create_weight_observation(patient_id, weight_kg=75.5):
    """
    Create a weight observation for the patient
    """
    try:
        logger.info(f"📝 Creating weight observation: {weight_kg} kg")
        
        observation = {
            "resourceType": "Observation",
            "status": "final",
            "category": [
                {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                            "code": "vital-signs",
                            "display": "Vital Signs"
                        }
                    ]
                }
            ],
            "code": {
                "coding": [
                    {
                        "system": "http://loinc.org",
                        "code": "29463-7",
                        "display": "Body weight"
                    }
                ],
                "text": "Body weight"
            },
            "subject": {
                "reference": f"Patient/{patient_id}"
            },
            "effectiveDateTime": datetime.now().isoformat(),
            "valueQuantity": {
                "value": weight_kg,
                "unit": "kg",
                "system": "http://unitsofmeasure.org",
                "code": "kg"
            }
        }
        
        # Create observation via Observation Service (using public endpoint)
        url = "http://localhost:8007/api/public/observations"

        response = requests.post(
            url,
            json=observation,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code in [200, 201]:
            logger.info("✅ Weight observation created successfully")
            return response.json()
        else:
            logger.error(f"❌ Failed to create weight observation: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except Exception as e:
        logger.error(f"❌ Error creating weight observation: {str(e)}")
        return None

def create_height_observation(patient_id, height_cm=170):
    """
    Create a height observation for the patient
    """
    try:
        logger.info(f"📝 Creating height observation: {height_cm} cm")
        
        observation = {
            "resourceType": "Observation",
            "status": "final",
            "category": [
                {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                            "code": "vital-signs",
                            "display": "Vital Signs"
                        }
                    ]
                }
            ],
            "code": {
                "coding": [
                    {
                        "system": "http://loinc.org",
                        "code": "8302-2",
                        "display": "Body height"
                    }
                ],
                "text": "Body height"
            },
            "subject": {
                "reference": f"Patient/{patient_id}"
            },
            "effectiveDateTime": datetime.now().isoformat(),
            "valueQuantity": {
                "value": height_cm,
                "unit": "cm",
                "system": "http://unitsofmeasure.org",
                "code": "cm"
            }
        }
        
        # Create observation via Observation Service (using public endpoint)
        url = "http://localhost:8007/api/public/observations"

        response = requests.post(
            url,
            json=observation,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code in [200, 201]:
            logger.info("✅ Height observation created successfully")
            return response.json()
        else:
            logger.error(f"❌ Failed to create height observation: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except Exception as e:
        logger.error(f"❌ Error creating height observation: {str(e)}")
        return None

def main():
    """Main test function"""
    logger.info("🚀 Patient Observations Test")
    logger.info("🎯 Check existing observations and create weight/height if missing")
    logger.info("=" * 70)
    
    try:
        # Step 1: Check existing observations
        logger.info("📋 Step 1: Checking existing observations...")
        observations = check_existing_observations()
        
        if observations is None:
            logger.error("❌ Cannot access Observation Service")
            return 1
        
        # Step 2: Create missing observations
        logger.info("📋 Step 2: Creating missing observations...")
        
        weight_created = False
        height_created = False
        
        if len(observations['weight_observations']) == 0:
            logger.info("📝 No weight observations found - creating one...")
            weight_obs = create_weight_observation(PATIENT_ID, 75.5)
            weight_created = weight_obs is not None
        else:
            logger.info("✅ Weight observations already exist")
        
        if len(observations['height_observations']) == 0:
            logger.info("📝 No height observations found - creating one...")
            height_obs = create_height_observation(PATIENT_ID, 170)
            height_created = height_obs is not None
        else:
            logger.info("✅ Height observations already exist")
        
        # Step 3: Verify final state
        logger.info("📋 Step 3: Verifying final state...")
        final_observations = check_existing_observations()
        
        if final_observations:
            logger.info("=" * 70)
            logger.info("📊 FINAL OBSERVATION STATE")
            logger.info("=" * 70)
            logger.info(f"✅ Total observations: {final_observations['total_observations']}")
            logger.info(f"✅ Weight observations: {len(final_observations['weight_observations'])}")
            logger.info(f"✅ Height observations: {len(final_observations['height_observations'])}")
            
            if (len(final_observations['weight_observations']) > 0 and 
                len(final_observations['height_observations']) > 0):
                logger.info("🎉 Patient now has complete demographic observations!")
                logger.info("✅ Ready for Flow 2 testing with complete data")
                return 0
            else:
                logger.warning("⚠️ Patient still missing some demographic observations")
                return 1
        else:
            logger.error("❌ Could not verify final observation state")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
