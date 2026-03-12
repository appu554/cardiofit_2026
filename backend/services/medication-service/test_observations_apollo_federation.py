"""
Test Patient Observations via Apollo Federation

This test uses Apollo Federation GraphQL to query and create observations,
bypassing authentication issues and using the same pattern as Flow 2.
"""

import requests
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Test patient ID
PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def query_observations_via_federation(patient_id):
    """
    Query observations for a patient via Apollo Federation GraphQL
    """
    try:
        logger.info("🔍 Querying observations via Apollo Federation")
        logger.info(f"   Patient ID: {patient_id}")
        
        # Observation Service Federation GraphQL endpoint
        graphql_url = "http://localhost:8007/api/federation"

        # GraphQL query for observations by patient (using correct parameter name from introspection)
        query = """
        query GetPatientObservations($patientId: String!) {
            observations(patientId: $patientId, count: 100) {
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
                subject {
                    reference
                }
                effectiveDateTime
                valueQuantity {
                    value
                    unit
                    system
                    code
                }
                category {
                    text
                    coding {
                        system
                        code
                        display
                    }
                }
            }
        }
        """

        variables = {
            "patientId": patient_id
        }
        
        payload = {
            "query": query,
            "variables": variables
        }
        
        logger.info(f"📡 Making GraphQL request to: {graphql_url}")
        
        response = requests.post(
            graphql_url,
            json=payload,
            headers={
                "Content-Type": "application/json"
            },
            timeout=30
        )
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "errors" in data:
                logger.error("❌ GraphQL Errors:")
                for error in data["errors"]:
                    logger.error(f"   - {error.get('message', 'Unknown error')}")
                return None
            
            observations = data.get("data", {}).get("observations", [])
            
            logger.info(f"✅ Retrieved {len(observations)} observations")
            
            # Analyze observations for weight and height
            weight_obs = []
            height_obs = []
            other_obs = []
            
            for obs in observations:
                obs_type = classify_observation_graphql(obs)
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
            
            return {
                'weight_observations': weight_obs,
                'height_observations': height_obs,
                'other_observations': other_obs,
                'total_observations': len(observations)
            }
        else:
            logger.error(f"❌ GraphQL request failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except requests.exceptions.ConnectionError:
        logger.error("❌ Cannot connect to Observation Service GraphQL on port 8007")
        return None
    except Exception as e:
        logger.error(f"❌ Error querying observations: {str(e)}")
        return None

def create_observation_via_federation(patient_id, observation_type, value, unit):
    """
    Create an observation via Apollo Federation GraphQL
    """
    try:
        logger.info(f"📝 Creating {observation_type} observation via Apollo Federation")
        logger.info(f"   Value: {value} {unit}")
        
        # Observation Service Federation GraphQL endpoint
        graphql_url = "http://localhost:8007/api/federation"
        
        # Determine LOINC code based on observation type
        if observation_type == "weight":
            loinc_code = "29463-7"
            loinc_display = "Body weight"
        elif observation_type == "height":
            loinc_code = "8302-2"
            loinc_display = "Body height"
        else:
            raise ValueError(f"Unsupported observation type: {observation_type}")
        
        # GraphQL mutation for creating observation
        mutation = """
        mutation CreateObservation($input: CreateObservationInput!) {
            createObservation(input: $input) {
                success
                message
                observation {
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
                    subject {
                        reference
                    }
                    effectiveDateTime
                    valueQuantity {
                        value
                        unit
                        system
                        code
                    }
                }
            }
        }
        """
        
        variables = {
            "input": {
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
                            "code": loinc_code,
                            "display": loinc_display
                        }
                    ],
                    "text": loinc_display
                },
                "subject": {
                    "reference": f"Patient/{patient_id}"
                },
                "effectiveDateTime": datetime.now().isoformat(),
                "valueQuantity": {
                    "value": value,
                    "unit": unit,
                    "system": "http://unitsofmeasure.org",
                    "code": unit
                }
            }
        }
        
        payload = {
            "query": mutation,
            "variables": variables
        }
        
        logger.info(f"📡 Making GraphQL mutation to: {graphql_url}")
        
        response = requests.post(
            graphql_url,
            json=payload,
            headers={
                "Content-Type": "application/json"
            },
            timeout=30
        )
        
        logger.info(f"📡 Response Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            if "errors" in data:
                logger.error("❌ GraphQL Errors:")
                for error in data["errors"]:
                    logger.error(f"   - {error.get('message', 'Unknown error')}")
                return None
            
            result = data.get("data", {}).get("createObservation", {})
            
            if result.get("success"):
                logger.info("✅ Observation created successfully")
                observation = result.get("observation", {})
                logger.info(f"   ID: {observation.get('id', 'Unknown')}")
                return observation
            else:
                logger.error(f"❌ Failed to create observation: {result.get('message', 'Unknown error')}")
                return None
        else:
            logger.error(f"❌ GraphQL mutation failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except Exception as e:
        logger.error(f"❌ Error creating observation: {str(e)}")
        return None

def classify_observation_graphql(observation):
    """
    Classify GraphQL observation as weight, height, or other
    """
    try:
        code = observation.get('code', {})

        # Check LOINC codes
        coding = code.get('coding', [])
        if isinstance(coding, list):
            for c in coding:
                if isinstance(c, dict):
                    loinc_code = c.get('code', '')
                    if loinc_code in ['29463-7', '3141-9']:  # Body weight
                        return 'weight'
                    elif loinc_code in ['8302-2', '8306-3']:  # Body height
                        return 'height'

        # Check display text (handle None values)
        text = code.get('text')
        if text and isinstance(text, str):
            text_lower = text.lower()
            if any(keyword in text_lower for keyword in ['weight', 'body weight', 'mass']):
                return 'weight'
            elif any(keyword in text_lower for keyword in ['height', 'body height', 'stature']):
                return 'height'

        return 'other'
    except Exception as e:
        # If there's any error in classification, default to 'other'
        return 'other'

def main():
    """Main test function"""
    logger.info("🚀 Apollo Federation Observations Test")
    logger.info("🎯 Query and create observations using GraphQL Federation")
    logger.info("=" * 70)
    
    try:
        # Step 1: Query existing observations
        logger.info("📋 Step 1: Querying existing observations...")
        observations = query_observations_via_federation(PATIENT_ID)
        
        if observations is None:
            logger.error("❌ Cannot query observations via Apollo Federation")
            return 1
        
        # Step 2: Create missing observations
        logger.info("📋 Step 2: Creating missing observations...")
        
        weight_created = False
        height_created = False
        
        if len(observations['weight_observations']) == 0:
            logger.info("📝 No weight observations found - creating one...")
            weight_obs = create_observation_via_federation(PATIENT_ID, "weight", 75.5, "kg")
            weight_created = weight_obs is not None
        else:
            logger.info("✅ Weight observations already exist")
        
        if len(observations['height_observations']) == 0:
            logger.info("📝 No height observations found - creating one...")
            height_obs = create_observation_via_federation(PATIENT_ID, "height", 170, "cm")
            height_created = height_obs is not None
        else:
            logger.info("✅ Height observations already exist")
        
        # Step 3: Verify final state
        logger.info("📋 Step 3: Verifying final state...")
        final_observations = query_observations_via_federation(PATIENT_ID)
        
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
