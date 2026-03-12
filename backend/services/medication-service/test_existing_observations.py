"""
Test Existing Observations Analysis

This test analyzes what observations actually exist for our patient
to understand why there are no weight/height observations.
"""

import requests
import json
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Test patient ID
PATIENT_ID = "905a60cb-8241-418f-b29b-5b020e851392"

def analyze_existing_observations():
    """
    Analyze what observations actually exist for the patient
    """
    try:
        logger.info("🔍 Analyzing existing observations")
        logger.info(f"   Patient ID: {PATIENT_ID}")
        
        # Observation Service Federation GraphQL endpoint
        graphql_url = "http://localhost:8007/api/federation"
        
        # GraphQL query to get detailed observation info
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
            "patientId": PATIENT_ID
        }
        
        payload = {
            "query": query,
            "variables": variables
        }
        
        logger.info(f"📡 Making GraphQL request to: {graphql_url}")
        
        response = requests.post(
            graphql_url,
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
            
            observations = data.get("data", {}).get("observations", [])
            
            logger.info(f"✅ Retrieved {len(observations)} observations")
            logger.info("=" * 70)
            logger.info("📊 DETAILED OBSERVATION ANALYSIS")
            logger.info("=" * 70)
            
            for i, obs in enumerate(observations):
                logger.info(f"🔹 Observation {i+1}:")
                logger.info(f"   ID: {obs.get('id', 'Unknown')}")
                logger.info(f"   Status: {obs.get('status', 'Unknown')}")
                
                # Code analysis
                code = obs.get('code', {})
                if code:
                    code_text = code.get('text', 'No text')
                    logger.info(f"   Code Text: {code_text}")
                    
                    coding = code.get('coding', [])
                    if coding and isinstance(coding, list):
                        for c in coding:
                            if isinstance(c, dict):
                                system = c.get('system', 'Unknown')
                                code_val = c.get('code', 'Unknown')
                                display = c.get('display', 'Unknown')
                                logger.info(f"   Coding: {system} | {code_val} | {display}")
                
                # Value analysis
                value_quantity = obs.get('valueQuantity', {})
                if value_quantity and isinstance(value_quantity, dict):
                    value = value_quantity.get('value', 'Unknown')
                    unit = value_quantity.get('unit', 'Unknown')
                    logger.info(f"   Value: {value} {unit}")
                
                # Category analysis
                category = obs.get('category', {})
                if category and isinstance(category, dict):
                    cat_text = category.get('text', 'No category text')
                    logger.info(f"   Category: {cat_text}")
                
                # Subject analysis
                subject = obs.get('subject', {})
                if subject and isinstance(subject, dict):
                    reference = subject.get('reference', 'Unknown')
                    logger.info(f"   Subject: {reference}")
                
                # Effective date
                effective_date = obs.get('effectiveDateTime', 'Unknown')
                logger.info(f"   Date: {effective_date}")
                
                logger.info("")
            
            # Summary analysis
            logger.info("=" * 70)
            logger.info("📋 SUMMARY ANALYSIS")
            logger.info("=" * 70)
            
            # Count by category
            categories = {}
            loinc_codes = {}
            
            for obs in observations:
                # Category counting
                category = obs.get('category', {})
                if isinstance(category, dict):
                    cat_text = category.get('text', 'Unknown Category')
                    categories[cat_text] = categories.get(cat_text, 0) + 1
                
                # LOINC code counting
                code = obs.get('code', {})
                if isinstance(code, dict):
                    coding = code.get('coding', [])
                    if isinstance(coding, list):
                        for c in coding:
                            if isinstance(c, dict):
                                loinc_code = c.get('code', 'Unknown')
                                display = c.get('display', 'Unknown')
                                loinc_codes[f"{loinc_code} ({display})"] = loinc_codes.get(f"{loinc_code} ({display})", 0) + 1
            
            logger.info("📊 Categories:")
            for cat, count in categories.items():
                logger.info(f"   - {cat}: {count}")
            
            logger.info("")
            logger.info("📊 LOINC Codes:")
            for code, count in loinc_codes.items():
                logger.info(f"   - {code}: {count}")
            
            # Check for weight/height specifically
            logger.info("")
            logger.info("🎯 WEIGHT/HEIGHT ANALYSIS:")
            
            weight_codes = ['29463-7', '3141-9']
            height_codes = ['8302-2', '8306-3']
            
            found_weight = False
            found_height = False
            
            for obs in observations:
                code = obs.get('code', {})
                if isinstance(code, dict):
                    coding = code.get('coding', [])
                    if isinstance(coding, list):
                        for c in coding:
                            if isinstance(c, dict):
                                loinc_code = c.get('code', '')
                                if loinc_code in weight_codes:
                                    found_weight = True
                                    logger.info(f"   ✅ Found weight observation: {loinc_code}")
                                elif loinc_code in height_codes:
                                    found_height = True
                                    logger.info(f"   ✅ Found height observation: {loinc_code}")
            
            if not found_weight:
                logger.info("   ❌ No weight observations found")
                logger.info(f"   Expected LOINC codes: {weight_codes}")
            
            if not found_height:
                logger.info("   ❌ No height observations found")
                logger.info(f"   Expected LOINC codes: {height_codes}")
            
            return observations
        else:
            logger.error(f"❌ GraphQL request failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return None
            
    except Exception as e:
        logger.error(f"❌ Error analyzing observations: {str(e)}")
        return None

def main():
    """Main test function"""
    logger.info("🚀 Existing Observations Analysis")
    logger.info("🎯 Understanding what observations exist for our patient")
    logger.info("=" * 70)
    
    try:
        observations = analyze_existing_observations()
        
        if observations:
            logger.info("=" * 70)
            logger.info("🎉 ANALYSIS COMPLETE")
            logger.info("=" * 70)
            logger.info(f"✅ Found {len(observations)} observations for patient")
            logger.info("🎯 Now we know what data exists and what's missing!")
            return 0
        else:
            logger.error("❌ Could not analyze observations")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
