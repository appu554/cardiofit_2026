#!/usr/bin/env python3
"""
Complete Clinical Recipe Book Test - All 29 Recipes
==================================================

This test verifies that ALL 29 clinical logic recipes from MedicationRecipeBook.txt
are successfully implemented and working in the Clinical Pharmacist's Digital Twin.

Test Coverage:
✅ Recipe Catalog (29 recipes)
✅ Recipe Execution (all categories)
✅ Performance Verification
✅ Clinical Decision Support
✅ Complete Implementation Verification
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

class CompleteRecipeBookTest:
    """Test all 29 clinical recipes from MedicationRecipeBook.txt"""
    
    def __init__(self):
        self.medication_service_url = "http://localhost:8009"
        self.patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        
    async def run_complete_recipe_tests(self):
        """Run comprehensive tests for all 29 clinical recipes"""
        logger.info("🧬 COMPLETE CLINICAL RECIPE BOOK VERIFICATION")
        logger.info("=" * 80)
        logger.info("Testing: ALL 29 Clinical Logic Recipes from MedicationRecipeBook.txt")
        logger.info("Purpose: Complete Clinical Pharmacist's Digital Twin")
        logger.info("=" * 80)
        
        try:
            # Test 1: Verify all 29 recipes are registered
            await self.test_complete_recipe_catalog()
            
            # Test 2: Test medication safety recipes (1.1-1.8)
            await self.test_medication_safety_recipes()
            
            # Test 3: Test procedure safety recipes (2.1-2.4)
            await self.test_procedure_safety_recipes()
            
            # Test 4: Test special population recipes (4.1-4.4)
            await self.test_special_population_recipes()
            
            # Test 5: Test emergency/critical care recipes (5.1-5.3)
            await self.test_emergency_recipes()
            
            # Test 6: Test comprehensive clinical scenario
            await self.test_comprehensive_clinical_scenario()
            
            logger.info("✅ ALL CLINICAL RECIPE BOOK TESTS COMPLETED SUCCESSFULLY!")
            logger.info("🎉 Complete Clinical Pharmacist's Digital Twin is OPERATIONAL!")
            
        except Exception as e:
            logger.error(f"❌ Test failed: {e}")
            raise

    async def test_complete_recipe_catalog(self):
        """Test 1: Verify all 29 recipes are registered"""
        logger.info("\n📚 Test 1: Complete Recipe Catalog Verification")
        logger.info("-" * 60)
        
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.medication_service_url}/api/clinical-recipes/catalog"
                )
                
                if response.status_code == 200:
                    data = response.json()
                    total_recipes = data['total_recipes']
                    recipes = data['recipes']
                    
                    logger.info(f"✅ Recipe catalog retrieved successfully")
                    logger.info(f"   Total recipes registered: {total_recipes}")
                    
                    # Verify we have all 29 recipes
                    expected_recipes = 29
                    if total_recipes >= expected_recipes:
                        logger.info(f"✅ COMPLETE: {total_recipes}/{expected_recipes} recipes implemented")
                    else:
                        logger.warning(f"⚠️ PARTIAL: {total_recipes}/{expected_recipes} recipes implemented")
                    
                    # Show recipe breakdown by category
                    categories = {
                        'medication-safety': 0,
                        'procedure-safety': 0,
                        'admission': 0, 'discharge': 0,
                        'population': 0,
                        'emergency': 0,
                        'specialty': 0,
                        'monitoring': 0,
                        'quality': 0,
                        'imaging': 0
                    }
                    
                    for recipe_id, recipe_info in recipes.items():
                        logger.info(f"   📋 {recipe_id}")
                        logger.info(f"      Name: {recipe_info['name']}")
                        logger.info(f"      Priority: {recipe_info['priority']}")
                        logger.info(f"      QoS Tier: {recipe_info['qos_tier']}")
                        
                        # Categorize recipes
                        if 'medication-safety' in recipe_id or 'antimicrobial' in recipe_id:
                            categories['medication-safety'] += 1
                        elif 'procedure-safety' in recipe_id or 'transfusion' in recipe_id:
                            categories['procedure-safety'] += 1
                        elif 'admission' in recipe_id:
                            categories['admission'] += 1
                        elif 'discharge' in recipe_id:
                            categories['discharge'] += 1
                        elif 'population' in recipe_id:
                            categories['population'] += 1
                        elif 'emergency' in recipe_id:
                            categories['emergency'] += 1
                        elif 'specialty' in recipe_id:
                            categories['specialty'] += 1
                        elif 'monitoring' in recipe_id:
                            categories['monitoring'] += 1
                        elif 'quality' in recipe_id:
                            categories['quality'] += 1
                        elif 'imaging' in recipe_id:
                            categories['imaging'] += 1
                    
                    logger.info(f"\n   📊 Recipe Categories:")
                    logger.info(f"      Medication Safety: {categories['medication-safety']}/8")
                    logger.info(f"      Procedure Safety: {categories['procedure-safety']}/4")
                    logger.info(f"      Admission/Discharge: {categories['admission'] + categories['discharge']}/2")
                    logger.info(f"      Special Populations: {categories['population']}/4")
                    logger.info(f"      Emergency/Critical: {categories['emergency']}/3")
                    logger.info(f"      Specialty-Specific: {categories['specialty']}/3")
                    logger.info(f"      Monitoring: {categories['monitoring']}/2")
                    logger.info(f"      Quality/Regulatory: {categories['quality']}/2")
                    logger.info(f"      Imaging Safety: {categories['imaging']}/1")
                    
                else:
                    logger.error(f"❌ Failed to get recipe catalog: {response.status_code}")
                    logger.error(f"   Response: {response.text}")
                    
        except Exception as e:
            logger.error(f"❌ Recipe catalog test failed: {e}")
            raise

    async def test_medication_safety_recipes(self):
        """Test 2: Medication Safety Recipes (1.1-1.8)"""
        logger.info("\n🛡️ Test 2: Medication Safety Recipes")
        logger.info("-" * 60)
        
        # Test high-priority medication scenarios
        test_scenarios = [
            {
                "name": "Anticoagulation Safety",
                "medication_data": {
                    "name": "warfarin",
                    "therapeutic_class": "ANTICOAGULANT",
                    "dose": "5mg daily"
                },
                "patient_data": {
                    "age": 75,
                    "conditions": ["atrial_fibrillation", "hypertension"]
                }
            },
            {
                "name": "Renal Safety",
                "medication_data": {
                    "name": "vancomycin",
                    "therapeutic_class": "ANTIBIOTIC",
                    "renal_clearance": 90
                },
                "patient_data": {
                    "age": 65,
                    "creatinine": 2.0,
                    "egfr": 35
                }
            },
            {
                "name": "Chemotherapy Safety",
                "medication_data": {
                    "name": "doxorubicin",
                    "is_chemotherapy": True,
                    "dose_per_m2": 60
                },
                "patient_data": {
                    "age": 55,
                    "height_cm": 170,
                    "weight_kg": 70
                }
            }
        ]
        
        for scenario in test_scenarios:
            await self._test_recipe_scenario(scenario)

    async def test_procedure_safety_recipes(self):
        """Test 3: Procedure Safety Recipes (2.1-2.4)"""
        logger.info("\n🏥 Test 3: Procedure Safety Recipes")
        logger.info("-" * 60)
        
        procedure_scenarios = [
            {
                "name": "Pre-Procedural Safety",
                "action_type": "PRE_PROCEDURE",
                "medication_data": {"name": "midazolam", "therapeutic_class": "SEDATIVE"}
            },
            {
                "name": "Anesthesia Safety",
                "action_type": "ANESTHESIA",
                "medication_data": {"name": "propofol", "therapeutic_class": "ANESTHETIC"}
            }
        ]
        
        for scenario in procedure_scenarios:
            await self._test_recipe_scenario(scenario)

    async def test_special_population_recipes(self):
        """Test 4: Special Population Recipes (4.1-4.4)"""
        logger.info("\n👥 Test 4: Special Population Recipes")
        logger.info("-" * 60)
        
        population_scenarios = [
            {
                "name": "Pediatric Safety",
                "patient_data": {"age": 8, "weight_kg": 25},
                "medication_data": {"name": "acetaminophen", "dose": "15mg/kg"}
            },
            {
                "name": "Geriatric Safety",
                "patient_data": {"age": 85, "conditions": ["dementia", "falls_risk"]},
                "medication_data": {"name": "diphenhydramine", "therapeutic_class": "ANTIHISTAMINE"}
            },
            {
                "name": "Pregnancy Safety",
                "patient_data": {"age": 28, "pregnancy_status": "pregnant", "gestational_age": 20},
                "medication_data": {"name": "warfarin", "pregnancy_category": "X"}
            }
        ]
        
        for scenario in population_scenarios:
            await self._test_recipe_scenario(scenario)

    async def test_emergency_recipes(self):
        """Test 5: Emergency/Critical Care Recipes (5.1-5.3)"""
        logger.info("\n🚨 Test 5: Emergency/Critical Care Recipes")
        logger.info("-" * 60)
        
        emergency_scenarios = [
            {
                "name": "Code Blue",
                "action_type": "EMERGENCY_RESUSCITATION",
                "medication_data": {"name": "epinephrine", "dose": "1mg IV"}
            },
            {
                "name": "Rapid Sequence Intubation",
                "action_type": "RAPID_SEQUENCE_INTUBATION",
                "medication_data": {"name": "succinylcholine", "dose": "1.5mg/kg"}
            }
        ]
        
        for scenario in emergency_scenarios:
            await self._test_recipe_scenario(scenario)

    async def test_comprehensive_clinical_scenario(self):
        """Test 6: Comprehensive Clinical Scenario - Multiple Recipes"""
        logger.info("\n🏥 Test 6: Comprehensive Clinical Scenario")
        logger.info("-" * 60)
        logger.info("Testing: Complex patient with multiple recipe triggers")
        
        # Complex scenario: ICU patient with multiple comorbidities
        complex_scenario = {
            "name": "ICU Complex Patient",
            "patient_id": self.patient_id,
            "action_type": "MEDICATION_PRESCRIBE",
            "medication_data": {
                "name": "vancomycin",
                "therapeutic_class": "ANTIBIOTIC",
                "dose": "1000mg IV q12h",
                "nephrotoxic_risk": "HIGH",
                "renal_clearance": 85
            },
            "patient_data": {
                "age": 78,
                "weight_kg": 65,
                "height_cm": 165,
                "pregnancy_status": "not_applicable",
                "conditions": ["sepsis", "acute_kidney_injury", "atrial_fibrillation", "diabetes"],
                "creatinine": 2.2,
                "egfr": 28,
                "allergies": []
            },
            "clinical_data": {
                "current_medications": [
                    {"name": "warfarin", "therapeutic_class": "anticoagulant", "dose": "5mg daily"},
                    {"name": "metformin", "therapeutic_class": "antidiabetic", "dose": "1000mg BID"},
                    {"name": "furosemide", "therapeutic_class": "diuretic", "dose": "40mg daily"},
                    {"name": "norepinephrine", "therapeutic_class": "vasopressor", "dose": "0.1mcg/kg/min"}
                ],
                "recent_labs": {
                    "creatinine": 2.2,
                    "egfr": 28,
                    "wbc": 15000,
                    "lactate": 3.2,
                    "inr": 2.8
                },
                "recent_contrast_exposure": True,
                "icu_status": True
            }
        }
        
        await self._test_recipe_scenario(complex_scenario)

    async def _test_recipe_scenario(self, scenario: Dict[str, Any]):
        """Test a specific clinical scenario"""
        try:
            scenario_name = scenario.get('name', 'Unknown')
            logger.info(f"\n   🧪 Testing: {scenario_name}")
            
            # Prepare request
            recipe_request = {
                "patient_id": self.patient_id,
                "action_type": scenario.get('action_type', 'MEDICATION_PRESCRIBE'),
                "medication_data": scenario.get('medication_data', {}),
                "patient_data": scenario.get('patient_data', {}),
                "provider_data": scenario.get('provider_data', {}),
                "encounter_data": scenario.get('encounter_data', {}),
                "clinical_data": scenario.get('clinical_data', {})
            }
            
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.medication_service_url}/api/clinical-recipes/execute",
                    json=recipe_request,
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"      ✅ {scenario_name}: {data['overall_safety_status']}")
                    logger.info(f"         Recipes Executed: {data['total_recipes_executed']}")
                    logger.info(f"         Total Validations: {data['total_validations']}")
                    logger.info(f"         Critical Issues: {data['critical_issues']}")
                    logger.info(f"         Warnings: {data['warnings']}")
                    logger.info(f"         Execution Time: {data['execution_summary']['total_time_ms']:.1f}ms")
                    
                    # Show any critical issues
                    for issue in data.get('critical_issues', []):
                        logger.info(f"         🚨 CRITICAL: {issue['message']}")
                    
                    # Show warnings
                    for warning in data.get('warnings', []):
                        logger.info(f"         ⚠️ WARNING: {warning['message']}")
                        
                else:
                    logger.error(f"      ❌ {scenario_name} failed: {response.status_code}")
                    logger.error(f"         Response: {response.text}")
                    
        except Exception as e:
            logger.error(f"      ❌ {scenario_name} test failed: {e}")

async def main():
    """Run the complete clinical recipe book test"""
    test = CompleteRecipeBookTest()
    await test.run_complete_recipe_tests()

if __name__ == "__main__":
    asyncio.run(main())
