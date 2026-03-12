"""
Create Weight and Height Observations

This script creates weight and height observations for our test patient
directly in the Google Healthcare FHIR Store.
"""

import requests
import json
import logging
from datetime import datetime
import uuid

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Test patient ID
PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def create_weight_observation_fhir():
    """
    Create a weight observation using the Observation Service
    """
    try:
        logger.info("📝 Creating weight observation via Observation Service")
        
        # Create FHIR Observation resource for weight
        weight_observation = {
            "resourceType": "Observation",
            "id": str(uuid.uuid4()),
            "status": "final",
            "category": [
                {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                            "code": "vital-signs",
                            "display": "Vital Signs"
                        }
                    ],
                    "text": "Vital Signs"
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
                "reference": f"Patient/{PATIENT_ID}"
            },
            "effectiveDateTime": datetime.now().strftime("%Y-%m-%dT%H:%M:%SZ"),
            "valueQuantity": {
                "value": 75.5,
                "unit": "kg",
                "system": "http://unitsofmeasure.org",
                "code": "kg"
            }
        }
        
        # Use Observation Service PUBLIC endpoint to create the observation (bypasses auth)
        url = "http://localhost:8007/api/public/observations"

        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json"
        }
        
        logger.info(f"📡 Creating weight observation via: {url}")
        
        response = requests.post(
            url,
            json=weight_observation,
            headers=headers,
            timeout=30
        )
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code in [200, 201]:
            result = response.json()
            logger.info("✅ Weight observation created successfully")
            logger.info(f"   ID: {result.get('id', 'Unknown')}")
            return result
        else:
            logger.error(f"❌ Failed to create weight observation: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except Exception as e:
        logger.error(f"❌ Error creating weight observation: {str(e)}")
        return None

def create_height_observation_fhir():
    """
    Create a height observation using the Observation Service
    """
    try:
        logger.info("📝 Creating height observation via Observation Service")
        
        # Create FHIR Observation resource for height
        height_observation = {
            "resourceType": "Observation",
            "id": str(uuid.uuid4()),
            "status": "final",
            "category": [
                {
                    "coding": [
                        {
                            "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                            "code": "vital-signs",
                            "display": "Vital Signs"
                        }
                    ],
                    "text": "Vital Signs"
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
                "reference": f"Patient/{PATIENT_ID}"
            },
            "effectiveDateTime": datetime.now().strftime("%Y-%m-%dT%H:%M:%SZ"),
            "valueQuantity": {
                "value": 170,
                "unit": "cm",
                "system": "http://unitsofmeasure.org",
                "code": "cm"
            }
        }
        
        # Use Observation Service PUBLIC endpoint to create the observation (bypasses auth)
        url = "http://localhost:8007/api/public/observations"

        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json"
        }
        
        logger.info(f"📡 Creating height observation via: {url}")
        
        response = requests.post(
            url,
            json=height_observation,
            headers=headers,
            timeout=30
        )
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code in [200, 201]:
            result = response.json()
            logger.info("✅ Height observation created successfully")
            logger.info(f"   ID: {result.get('id', 'Unknown')}")
            return result
        else:
            logger.error(f"❌ Failed to create height observation: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except Exception as e:
        logger.error(f"❌ Error creating height observation: {str(e)}")
        return None

def verify_observations_created():
    """
    Verify that the weight and height observations were created successfully
    """
    try:
        logger.info("🔍 Verifying observations were created")
        
        # Query observations via Apollo Federation
        graphql_url = "http://localhost:8007/api/federation"
        
        query = """
        query GetPatientObservations($patientId: String!) {
            observations(patientId: $patientId, count: 50) {
                id
                status
                code {
                    text
                    coding {
                        system
                        code
                        display
                    }
                }
                valueQuantity {
                    value
                    unit
                }
                effectiveDateTime
            }
        }
        """
        
        variables = {
            "patientId": PATIENT_ID
        }
        
        payload = {
            "query": query,
            "variables": variables
        }
        
        response = requests.post(
            graphql_url,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        if response.status_code == 200:
            data = response.json()
            observations = data.get("data", {}).get("observations", [])
            
            weight_count = 0
            height_count = 0
            
            for obs in observations:
                code = obs.get('code', {})
                coding = code.get('coding', [])
                
                for c in coding:
                    loinc_code = c.get('code', '')
                    if loinc_code == '29463-7':  # Weight
                        weight_count += 1
                        value_qty = obs.get('valueQuantity', {})
                        logger.info(f"   ✅ Weight: {value_qty.get('value')} {value_qty.get('unit')}")
                    elif loinc_code == '8302-2':  # Height
                        height_count += 1
                        value_qty = obs.get('valueQuantity', {})
                        logger.info(f"   ✅ Height: {value_qty.get('value')} {value_qty.get('unit')}")
            
            logger.info(f"📊 Verification Results:")
            logger.info(f"   Total observations: {len(observations)}")
            logger.info(f"   Weight observations: {weight_count}")
            logger.info(f"   Height observations: {height_count}")
            
            return weight_count > 0 and height_count > 0
        else:
            logger.error(f"❌ Verification failed: {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Verification error: {str(e)}")
        return False

def main():
    """Main function"""
    logger.info("🚀 Creating Weight and Height Observations")
    logger.info("🎯 Adding missing demographic observations for Flow 2 testing")
    logger.info("=" * 70)
    
    try:
        # Step 1: Create weight observation
        logger.info("📋 Step 1: Creating weight observation...")
        weight_created = create_weight_observation_fhir()
        
        # Step 2: Create height observation
        logger.info("📋 Step 2: Creating height observation...")
        height_created = create_height_observation_fhir()
        
        # Step 3: Verify observations were created
        logger.info("📋 Step 3: Verifying observations...")
        verification_ok = verify_observations_created()
        
        logger.info("=" * 70)
        logger.info("📊 CREATION RESULTS")
        logger.info("=" * 70)
        
        if weight_created and height_created and verification_ok:
            logger.info("🎉 SUCCESS: Weight and height observations created!")
            logger.info("✅ Patient now has complete demographic data")
            logger.info("✅ Ready for Flow 2 testing with complete data")
            logger.info("")
            logger.info("🎯 Next Steps:")
            logger.info("1. Run Flow 2 test again")
            logger.info("2. Should now see >90% completeness")
            logger.info("3. Should see SUCCESS status instead of FAILED")
            return 0
        else:
            logger.error("❌ FAILED: Could not create all required observations")
            logger.error("🔧 Check Observation Service logs for details")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
