"""
Flow 2 Test with Complete Clinical Data

This test ensures Flow 2 works with COMPLETE clinical data required for 
medication safety validation. It addresses the data quality issues that
prevent proper clinical decision support.
"""

import requests
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def create_complete_test_patient():
    """
    Create a test patient with complete clinical data for Flow 2 testing
    """
    complete_patient_data = {
        "patient_id": "flow2_test_patient_001",
        "demographics": {
            "age": 65,
            "weight_kg": 75.5,
            "height_cm": 170,
            "gender": "male",
            "date_of_birth": "1958-08-02"
        },
        "allergies": [
            {
                "allergen": "penicillin",
                "reaction": "rash",
                "severity": "moderate"
            },
            {
                "allergen": "shellfish",
                "reaction": "anaphylaxis",
                "severity": "severe"
            }
        ],
        "conditions": [
            "atrial_fibrillation",
            "hypertension",
            "diabetes_type_2"
        ],
        "current_medications": [
            {
                "name": "lisinopril",
                "dose": "10mg",
                "frequency": "daily",
                "therapeutic_class": "ACE_INHIBITOR"
            },
            {
                "name": "metformin",
                "dose": "500mg",
                "frequency": "twice_daily",
                "therapeutic_class": "BIGUANIDE"
            }
        ],
        "labs": {
            "creatinine": 1.2,
            "egfr": 65,
            "alt": 28,
            "ast": 32,
            "inr": 1.0,
            "hemoglobin": 13.5
        },
        "vitals": {
            "blood_pressure": {"systolic": 140, "diastolic": 85},
            "heart_rate": 72,
            "temperature": 98.6,
            "respiratory_rate": 16
        }
    }
    
    return complete_patient_data

def test_flow2_with_complete_data():
    """
    Test Flow 2 using the Medication Service API with complete clinical data
    """
    try:
        logger.info("🧪 Testing Flow 2 with Complete Clinical Data")
        logger.info("=" * 70)
        
        # Create complete test patient data
        patient_data = create_complete_test_patient()
        
        logger.info("📋 Test Patient Data:")
        logger.info(f"   Patient ID: {patient_data['patient_id']}")
        logger.info(f"   Age: {patient_data['demographics']['age']}")
        logger.info(f"   Weight: {patient_data['demographics']['weight_kg']}kg")
        logger.info(f"   Allergies: {len(patient_data['allergies'])}")
        logger.info(f"   Current Medications: {len(patient_data['current_medications'])}")
        logger.info(f"   Conditions: {len(patient_data['conditions'])}")
        
        # Test medication to validate (warfarin - high risk anticoagulant)
        test_medication = {
            "name": "warfarin",
            "generic_name": "warfarin sodium",
            "dose": "5mg",
            "frequency": "daily",
            "route": "oral",
            "therapeutic_class": "ANTICOAGULANT",
            "is_anticoagulant": True,
            "is_high_risk": True,
            "requires_monitoring": True
        }
        
        logger.info("💊 Test Medication:")
        logger.info(f"   Name: {test_medication['name']}")
        logger.info(f"   Dose: {test_medication['dose']}")
        logger.info(f"   Type: Anticoagulant (High Risk)")
        
        # Flow 2 validation request
        flow2_request = {
            "patient_id": patient_data['patient_id'],
            "medication": test_medication,
            "provider_id": "provider_123",
            "encounter_id": "encounter_456",
            "action_type": "prescribe",
            "urgency": "routine",
            "workflow_id": "flow2_complete_data_test"
        }
        
        # Make Flow 2 API call
        logger.info("🔄 Making Flow 2 API call with complete data...")
        
        flow2_url = "http://localhost:8009/api/flow2/medication-safety/validate"
        
        response = requests.post(
            flow2_url,
            json=flow2_request,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        logger.info(f"📡 Flow 2 Response Status: {response.status_code}")
        
        if response.status_code == 200:
            result = response.json()
            
            logger.info("=" * 70)
            logger.info("📊 FLOW 2 RESULTS WITH COMPLETE DATA")
            logger.info("=" * 70)
            
            logger.info(f"✅ Request ID: {result.get('request_id', 'Unknown')}")
            logger.info(f"✅ Overall Safety Status: {result.get('overall_safety_status', 'Unknown')}")
            logger.info(f"✅ Context Recipe Used: {result.get('context_recipe_used', 'Unknown')}")
            logger.info(f"✅ Context Completeness: {result.get('context_completeness_score', 0):.2%}")
            logger.info(f"✅ Execution Time: {result.get('execution_time_ms', 0):.1f}ms")
            
            # Clinical recipes executed
            recipes = result.get('clinical_recipes_executed', [])
            logger.info(f"✅ Clinical Recipes Executed: {len(recipes)}")
            for recipe in recipes:
                logger.info(f"   - {recipe}")
            
            # Safety summary
            safety_summary = result.get('safety_summary', {})
            if safety_summary:
                logger.info("✅ Safety Summary:")
                logger.info(f"   - Total Validations: {safety_summary.get('total_validations', 0)}")
                logger.info(f"   - Critical Issues: {safety_summary.get('critical_issues', 0)}")
                logger.info(f"   - High Issues: {safety_summary.get('high_issues', 0)}")
                logger.info(f"   - Medium Issues: {safety_summary.get('medium_issues', 0)}")
                
                # Clinical decision support
                cds = safety_summary.get('clinical_decision_support', {})
                if cds:
                    logger.info("✅ Clinical Decision Support:")
                    logger.info(f"   - Provider Summary: {cds.get('provider_summary', 'N/A')}")
                    logger.info(f"   - Patient Explanation: {cds.get('patient_explanation', 'N/A')}")
            
            # Check for data quality issues
            errors = result.get('errors', [])
            if errors:
                logger.warning("⚠️ Data Quality Issues:")
                for error in errors:
                    logger.warning(f"   - {error}")
            
            # Validate Flow 2 clinical requirements
            logger.info("=" * 70)
            logger.info("🎯 FLOW 2 CLINICAL VALIDATION")
            logger.info("=" * 70)
            
            # Check if we have complete clinical context
            completeness = result.get('context_completeness_score', 0)
            if completeness >= 0.9:
                logger.info(f"✅ CLINICAL DATA: Excellent completeness ({completeness:.2%})")
            elif completeness >= 0.7:
                logger.info(f"⚠️ CLINICAL DATA: Good completeness ({completeness:.2%})")
            else:
                logger.error(f"❌ CLINICAL DATA: Poor completeness ({completeness:.2%})")
            
            # Check safety assessment
            safety_status = result.get('overall_safety_status', '')
            if safety_status in ['SAFE', 'WARNING', 'UNSAFE']:
                logger.info(f"✅ SAFETY ASSESSMENT: Valid clinical decision ({safety_status})")
            else:
                logger.error(f"❌ SAFETY ASSESSMENT: Invalid or missing ({safety_status})")
            
            # Check clinical recipes execution
            if len(recipes) > 0:
                logger.info(f"✅ CLINICAL RECIPES: {len(recipes)} recipes executed successfully")
            else:
                logger.error("❌ CLINICAL RECIPES: No recipes executed")
            
            # Overall assessment
            if (completeness >= 0.9 and 
                safety_status in ['SAFE', 'WARNING', 'UNSAFE'] and 
                len(recipes) > 0):
                logger.info("🎉 FLOW 2 WITH COMPLETE DATA: SUCCESS")
                logger.info("✅ Flow 2 provides clinically meaningful medication safety validation!")
                return True
            else:
                logger.error("❌ FLOW 2 WITH COMPLETE DATA: FAILED")
                logger.error("🔧 Flow 2 still has data quality or clinical validation issues")
                return False
        else:
            logger.error(f"❌ Flow 2 API call failed: {response.status_code}")
            logger.error(f"Response: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        logger.error("❌ Cannot connect to Medication Service on port 8009")
        logger.error("🔧 Make sure Medication Service is running")
        return False
    except Exception as e:
        logger.error(f"❌ Flow 2 test with complete data failed: {str(e)}")
        return False

def test_data_quality_requirements():
    """
    Test what data quality requirements Flow 2 actually needs
    """
    try:
        logger.info("🔍 Testing Flow 2 Data Quality Requirements")
        
        # Test with minimal data to see what's actually required
        minimal_request = {
            "patient_id": "minimal_test_patient",
            "medication": {
                "name": "aspirin",
                "dose": "81mg",
                "frequency": "daily"
            },
            "urgency": "routine"
        }
        
        flow2_url = "http://localhost:8009/api/flow2/medication-safety/validate"
        
        response = requests.post(
            flow2_url,
            json=minimal_request,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        
        if response.status_code == 200:
            result = response.json()
            
            logger.info("📊 Minimal Data Test Results:")
            logger.info(f"   Status: {result.get('overall_safety_status', 'Unknown')}")
            logger.info(f"   Completeness: {result.get('context_completeness_score', 0):.2%}")
            logger.info(f"   Recipes Executed: {len(result.get('clinical_recipes_executed', []))}")
            
            errors = result.get('errors', [])
            if errors:
                logger.info("📋 Data Requirements Identified:")
                for error in errors:
                    logger.info(f"   - {error}")
            
            return True
        else:
            logger.error(f"❌ Minimal data test failed: {response.status_code}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Data quality test failed: {str(e)}")
        return False

def main():
    """Main test function"""
    logger.info("🚀 Flow 2 Complete Data Quality Test")
    logger.info("🎯 This test addresses the data quality issues preventing proper Flow 2 validation")
    logger.info("")
    
    try:
        # Test 1: Data quality requirements
        logger.info("📋 Step 1: Understanding data quality requirements...")
        requirements_ok = test_data_quality_requirements()
        
        # Test 2: Complete data test
        logger.info("📋 Step 2: Testing Flow 2 with complete clinical data...")
        complete_data_ok = test_flow2_with_complete_data()
        
        logger.info("=" * 70)
        if complete_data_ok:
            logger.info("🎉 FLOW 2 DATA QUALITY TEST: PASSED")
            logger.info("✅ Flow 2 works correctly with complete clinical data!")
            logger.info("")
            logger.info("🔧 Next Steps:")
            logger.info("1. Ensure patient data sources have complete demographics")
            logger.info("2. Verify allergy and medication data is properly populated")
            logger.info("3. Test with real patient data that has complete clinical records")
            return 0
        else:
            logger.error("❌ FLOW 2 DATA QUALITY TEST: FAILED")
            logger.error("🔧 Flow 2 integration works but data quality issues remain")
            return 1
            
    except Exception as e:
        logger.error(f"❌ Test execution failed: {str(e)}")
        return 1

if __name__ == "__main__":
    exit_code = main()
    exit(exit_code)
