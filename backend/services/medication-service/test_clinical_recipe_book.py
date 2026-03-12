#!/usr/bin/env python3
"""
Clinical Recipe Book Test - Real Medication Service Implementation
================================================================

This test demonstrates the REAL purpose of the medication service as defined
in MedicationRecipeBook.txt: Clinical Logic Recipe execution for pharmaceutical
intelligence and clinical decision support.

Test Cases:
1. Clinical Recipe Catalog - Show available recipes
2. Universal Medication Safety Recipe - Execute foundational safety checks
3. Dose Calculation Engine - Real pharmaceutical calculations
4. Clinical Decision Support - Comprehensive analysis
5. Recipe-based Validation - Multi-tier safety checking
"""

import asyncio
import logging
import json
from datetime import datetime
from typing import Dict, Any

import httpx

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class ClinicalRecipeBookTest:
    """Test the real Clinical Recipe Book implementation"""
    
    def __init__(self):
        self.medication_service_url = "http://localhost:8009"
        self.patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        
    async def run_clinical_recipe_tests(self):
        """Run comprehensive clinical recipe book tests"""
        logger.info("🧬 Starting Clinical Recipe Book Tests")
        logger.info("=" * 70)
        logger.info("Testing: Clinical Logic Recipe Book Implementation")
        logger.info("Purpose: Clinical Pharmacist's Digital Twin")
        logger.info("=" * 70)
        
        try:
            # Test 1: Recipe Catalog
            await self.test_recipe_catalog()
            
            # Test 2: Universal Medication Safety Recipe
            await self.test_universal_medication_safety()
            
            # Test 3: Dose Calculation Engine
            await self.test_dose_calculation_engine()
            
            # Test 4: Clinical Decision Support
            await self.test_clinical_decision_support()
            
            # Test 5: Complex Clinical Scenario
            await self.test_complex_clinical_scenario()
            
            logger.info("✅ All Clinical Recipe Book tests completed successfully!")
            
        except Exception as e:
            logger.error(f"❌ Test failed: {e}")
            raise

    async def test_recipe_catalog(self):
        """Test 1: Get Clinical Recipe Catalog"""
        logger.info("\n📚 Test 1: Clinical Recipe Catalog")
        logger.info("-" * 50)
        
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.medication_service_url}/api/clinical-recipes/catalog"
                )
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Recipe catalog retrieved successfully")
                    logger.info(f"   Total recipes: {data['total_recipes']}")
                    
                    for recipe_id, recipe_info in data['recipes'].items():
                        logger.info(f"   📋 {recipe_id}")
                        logger.info(f"      Name: {recipe_info['name']}")
                        logger.info(f"      Priority: {recipe_info['priority']}")
                        logger.info(f"      QoS Tier: {recipe_info['qos_tier']}")
                        
                else:
                    logger.error(f"❌ Failed to get recipe catalog: {response.status_code}")
                    logger.error(f"   Response: {response.text}")
                    
        except Exception as e:
            logger.error(f"❌ Recipe catalog test failed: {e}")
            raise

    async def test_universal_medication_safety(self):
        """Test 2: Universal Medication Safety Recipe"""
        logger.info("\n🛡️ Test 2: Universal Medication Safety Recipe")
        logger.info("-" * 50)
        logger.info("Testing: Recipe 1.1 - Universal Medication Safety Check")
        
        try:
            # Create a medication prescription scenario
            recipe_request = {
                "patient_id": self.patient_id,
                "action_type": "MEDICATION_PRESCRIBE",
                "medication_data": {
                    "code": "1049502",  # Acetaminophen
                    "display": "Acetaminophen 325 MG Oral Tablet",
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                    "ingredients": ["acetaminophen"],
                    "therapeutic_class": "analgesic",
                    "pregnancy_category": "B"
                },
                "patient_data": {
                    "age": 45,
                    "weight_kg": 70,
                    "pregnancy_status": "not_pregnant",
                    "allergies": [
                        {
                            "substance": "penicillin",
                            "reaction": "rash",
                            "severity": "moderate"
                        }
                    ]
                },
                "provider_data": {
                    "provider_id": "provider-123",
                    "specialty": "internal_medicine"
                },
                "clinical_data": {
                    "current_medications": [
                        {
                            "name": "Lisinopril",
                            "therapeutic_class": "ace_inhibitor",
                            "dose": "10mg daily"
                        }
                    ]
                }
            }
            
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.medication_service_url}/api/clinical-recipes/execute",
                    json=recipe_request,
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Universal safety recipe executed successfully")
                    logger.info(f"   Overall Safety Status: {data['overall_safety_status']}")
                    logger.info(f"   Recipes Executed: {data['total_recipes_executed']}")
                    logger.info(f"   Total Validations: {data['total_validations']}")
                    logger.info(f"   Critical Issues: {data['critical_issues']}")
                    logger.info(f"   Warnings: {data['warnings']}")
                    
                    # Show execution performance
                    exec_summary = data['execution_summary']
                    logger.info(f"   Execution Time: {exec_summary['total_time_ms']:.1f}ms")
                    
                    # Show recipe results
                    for recipe in data['recipe_results']:
                        logger.info(f"   📋 Recipe: {recipe['recipe_name']}")
                        logger.info(f"      Status: {recipe['status']}")
                        logger.info(f"      Time: {recipe['execution_time_ms']:.1f}ms")
                        logger.info(f"      Validations: {len(recipe['validations'])}")
                        
                        # Show any validation issues
                        for validation in recipe['validations']:
                            if not validation['passed']:
                                logger.info(f"      ⚠️ {validation['severity']}: {validation['message']}")
                
                else:
                    logger.error(f"❌ Failed to execute safety recipe: {response.status_code}")
                    logger.error(f"   Response: {response.text}")
                    
        except Exception as e:
            logger.error(f"❌ Universal safety recipe test failed: {e}")
            raise

    async def test_dose_calculation_engine(self):
        """Test 3: Dose Calculation Engine"""
        logger.info("\n🧮 Test 3: Dose Calculation Engine")
        logger.info("-" * 50)
        logger.info("Testing: Real pharmaceutical dose calculations")
        
        try:
            # Test weight-based calculation (Vancomycin)
            dose_request = {
                "patient_id": self.patient_id,
                "medication_code": "11124",  # Vancomycin
                "indication": "severe_pneumonia",
                "calculation_type": "weight_based",
                "patient_context": {
                    "weight_kg": 70,
                    "age_years": 45,
                    "creatinine_clearance": 80,
                    "liver_function": "normal"
                },
                "dosing_parameters": {
                    "dose_per_kg": 15  # 15 mg/kg
                }
            }
            
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.medication_service_url}/api/dose-calculation/calculate",
                    json=dose_request,
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Dose calculation completed successfully")
                    logger.info(f"   Medication: {data.get('medication_code', 'N/A')} ({data.get('indication', 'N/A')})")
                    logger.info(f"   Calculation Type: {data.get('calculation_type', 'N/A')}")

                    if 'calculated_dose' in data:
                        dose = data['calculated_dose']
                        logger.info(f"   Calculated Dose: {dose.get('display_string', 'N/A')}")
                        logger.info(f"   Value: {dose.get('value', 'N/A')} {dose.get('unit', 'N/A')}")
                        logger.info(f"   Route: {dose.get('route', 'N/A')}")
                        logger.info(f"   Method: {dose.get('calculation_method', 'N/A')}")

                        # Show calculation factors
                        factors = dose.get('calculation_factors', {})
                        logger.info(f"   Calculation Factors:")
                        for factor, value in factors.items():
                            logger.info(f"     {factor}: {value}")

                    # Show clinical notes
                    if 'clinical_notes' in data:
                        logger.info(f"   Clinical Notes:")
                        for note in data['clinical_notes']:
                            logger.info(f"     • {note}")

                    # Show full response for debugging
                    logger.info(f"   Full Response: {data}")
                
                else:
                    logger.error(f"❌ Failed to calculate dose: {response.status_code}")
                    logger.error(f"   Response: {response.text}")
                    
        except Exception as e:
            logger.error(f"❌ Dose calculation test failed: {e}")
            raise

    async def test_clinical_decision_support(self):
        """Test 4: Clinical Decision Support"""
        logger.info("\n🧠 Test 4: Clinical Decision Support")
        logger.info("-" * 50)
        logger.info("Testing: Drug interaction checking")
        
        try:
            # Test drug interaction checking
            interaction_request = {
                "patient_id": self.patient_id,
                "new_medication_code": "1049502",  # Acetaminophen
                "new_medication_system": "http://www.nlm.nih.gov/research/umls/rxnorm"
            }
            
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.medication_service_url}/api/drug-interactions/check",
                    json=interaction_request,
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Drug interaction check completed")
                    logger.info(f"   Has Interactions: {data.get('has_interactions', False)}")
                    logger.info(f"   Interactions Found: {data.get('total_interactions', 0)}")
                    logger.info(f"   Current Medications: {data.get('current_medications_count', 0)}")

                    # Show interactions
                    for interaction in data.get('interactions', []):
                        logger.info(f"   🔄 Interaction: {interaction.get('severity', 'unknown')}")
                        logger.info(f"      Description: {interaction.get('description', 'N/A')}")
                        logger.info(f"      Management: {interaction.get('management', 'N/A')}")
                        logger.info(f"      Clinical Consequence: {interaction.get('clinical_consequence', 'N/A')}")

                    # Show full response for debugging
                    logger.info(f"   Full Response: {data}")
                
                else:
                    logger.error(f"❌ Failed to check interactions: {response.status_code}")
                    logger.error(f"   Response: {response.text}")
                    
        except Exception as e:
            logger.error(f"❌ Clinical decision support test failed: {e}")
            raise

    async def test_complex_clinical_scenario(self):
        """Test 5: Complex Clinical Scenario"""
        logger.info("\n🏥 Test 5: Complex Clinical Scenario")
        logger.info("-" * 50)
        logger.info("Testing: Elderly patient with polypharmacy")
        
        try:
            # Complex scenario: 75-year-old with multiple medications
            complex_request = {
                "patient_id": self.patient_id,
                "action_type": "MEDICATION_PRESCRIBE",
                "medication_data": {
                    "code": "11124",  # Vancomycin
                    "display": "Vancomycin 500mg IV",
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                    "ingredients": ["vancomycin"],
                    "therapeutic_class": "antibiotic",
                    "nephrotoxicity_risk": "HIGH"
                },
                "patient_data": {
                    "age": 75,
                    "weight_kg": 65,
                    "pregnancy_status": "not_applicable",
                    "allergies": []
                },
                "clinical_data": {
                    "current_medications": [
                        {
                            "name": "Warfarin",
                            "therapeutic_class": "anticoagulant",
                            "dose": "5mg daily"
                        },
                        {
                            "name": "Metformin",
                            "therapeutic_class": "antidiabetic",
                            "dose": "1000mg BID"
                        },
                        {
                            "name": "Lisinopril",
                            "therapeutic_class": "ace_inhibitor",
                            "dose": "10mg daily"
                        },
                        {
                            "name": "Furosemide",
                            "therapeutic_class": "diuretic",
                            "dose": "40mg daily"
                        }
                    ],
                    "lab_values": {
                        "creatinine": 1.8,
                        "egfr": 35,
                        "inr": 2.5
                    },
                    "conditions": [
                        "chronic_kidney_disease",
                        "atrial_fibrillation",
                        "diabetes_type_2",
                        "heart_failure"
                    ]
                }
            }
            
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.medication_service_url}/api/clinical-recipes/execute",
                    json=complex_request,
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Complex scenario analysis completed")
                    logger.info(f"   Overall Safety Status: {data['overall_safety_status']}")
                    logger.info(f"   Critical Issues: {data['critical_issues']}")
                    logger.info(f"   Warnings: {data['warnings']}")
                    
                    # Show critical issues
                    for issue in data['critical_issues']:
                        logger.info(f"   🚨 CRITICAL: {issue['message']}")
                        logger.info(f"      Explanation: {issue['explanation']}")
                        if issue['alternatives']:
                            logger.info(f"      Alternatives: {', '.join(issue['alternatives'])}")
                    
                    # Show warnings
                    for warning in data['warnings']:
                        logger.info(f"   ⚠️ WARNING: {warning['message']}")
                        logger.info(f"      Explanation: {warning['explanation']}")
                
                else:
                    logger.error(f"❌ Failed to analyze complex scenario: {response.status_code}")
                    logger.error(f"   Response: {response.text}")
                    
        except Exception as e:
            logger.error(f"❌ Complex scenario test failed: {e}")
            raise

async def main():
    """Run the clinical recipe book test"""
    test = ClinicalRecipeBookTest()
    await test.run_clinical_recipe_tests()

if __name__ == "__main__":
    asyncio.run(main())
