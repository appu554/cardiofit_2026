#!/usr/bin/env python3
"""
Real Clinical Case Test for Medication Service
==============================================

This test demonstrates the medication service working with real clinical data
through the Context Service integration, showing the complete workflow from
medication request to clinical decision support.

Test Case: 65-year-old patient with diabetes and hypertension requiring
medication review and new prescription.
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

class RealClinicalCaseTest:
    """Test medication service with real clinical scenarios"""
    
    def __init__(self):
        self.medication_service_url = "http://localhost:8009"
        self.context_service_url = "http://localhost:8016"
        self.patient_id = "905a60cb-8241-418f-b29b-5b020e851392"  # Our test patient
        
    async def run_comprehensive_test(self):
        """Run comprehensive real clinical case test"""
        logger.info("🏥 Starting Real Clinical Case Test")
        logger.info("=" * 60)
        
        try:
            # Test 1: Get current patient medications
            await self.test_current_medications()
            
            # Test 2: Get medication prescribing context
            await self.test_prescribing_context()
            
            # Test 3: Test medication safety context
            await self.test_safety_context()
            
            # Test 4: Test new medication prescription workflow
            await self.test_prescription_workflow()
            
            # Test 5: Test medication interaction checking
            await self.test_drug_interactions()
            
            logger.info("✅ All real clinical case tests completed successfully!")
            
        except Exception as e:
            logger.error(f"❌ Test failed: {e}")
            raise

    async def test_current_medications(self):
        """Test 1: Get current patient medications"""
        logger.info("\n🔍 Test 1: Current Patient Medications")
        logger.info("-" * 40)
        
        try:
            # Use our new public endpoint
            url = f"{self.medication_service_url}/api/public/medication-requests/patient/{self.patient_id}"
            
            async with httpx.AsyncClient() as client:
                response = await client.get(url)
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Retrieved {data['count']} medication requests")
                    
                    if data['medication_requests']:
                        for i, med in enumerate(data['medication_requests'][:3], 1):
                            logger.info(f"   {i}. Medication ID: {med.get('id', 'N/A')}")
                            logger.info(f"      Status: {med.get('status', 'N/A')}")
                            logger.info(f"      Intent: {med.get('intent', 'N/A')}")
                    else:
                        logger.info("   No current medications found")
                else:
                    logger.error(f"❌ Failed to get medications: {response.status_code}")
                    
        except Exception as e:
            logger.error(f"❌ Current medications test failed: {e}")
            raise

    async def test_prescribing_context(self):
        """Test 2: Get medication prescribing context from Context Service"""
        logger.info("\n🔍 Test 2: Medication Prescribing Context")
        logger.info("-" * 40)
        
        try:
            query = """
            query GetMedicationPrescribingContext($patientId: String!) {
                getContextByRecipe(
                    patientId: $patientId,
                    recipeId: "medication_prescribing_v2",
                    providerId: "provider-123",
                    workflowId: "real-clinical-case-test"
                ) {
                    contextId
                    patientId
                    recipeUsed
                    completenessScore
                    assemblyDurationMs
                    status
                    safetyFlags {
                        flagType
                        severity
                        message
                    }
                    assembledData
                }
            }
            """
            
            variables = {"patientId": self.patient_id}
            
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.context_service_url}/graphql",
                    json={"query": query, "variables": variables},
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code == 200:
                    result = response.json()
                    
                    if "errors" in result:
                        logger.error(f"❌ GraphQL errors: {result['errors']}")
                        return
                    
                    context = result["data"]["getContextByRecipe"]
                    logger.info(f"✅ Context retrieved successfully")
                    logger.info(f"   Context ID: {context['contextId']}")
                    logger.info(f"   Completeness: {context['completenessScore']:.1%}")
                    logger.info(f"   Assembly time: {context['assemblyDurationMs']}ms")
                    logger.info(f"   Safety flags: {len(context['safetyFlags'])}")
                    
                    # Show safety flags
                    if context['safetyFlags']:
                        logger.info("   Safety Flags:")
                        for flag in context['safetyFlags'][:3]:
                            logger.info(f"     - {flag['severity']}: {flag['message']}")
                else:
                    logger.error(f"❌ Context request failed: {response.status_code}")
                    
        except Exception as e:
            logger.error(f"❌ Prescribing context test failed: {e}")
            raise

    async def test_safety_context(self):
        """Test 3: Get medication safety context"""
        logger.info("\n🔍 Test 3: Medication Safety Context")
        logger.info("-" * 40)
        
        try:
            query = """
            query GetMedicationSafetyContext($patientId: String!) {
                getContextByRecipe(
                    patientId: $patientId,
                    recipeId: "medication_safety_base_context_v2",
                    providerId: "provider-123",
                    workflowId: "safety-check-test"
                ) {
                    contextId
                    completenessScore
                    safetyFlags {
                        flagType
                        severity
                        message
                        dataPoint
                    }
                    assembledData
                }
            }
            """
            
            variables = {"patientId": self.patient_id}
            
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.context_service_url}/graphql",
                    json={"query": query, "variables": variables},
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code == 200:
                    result = response.json()
                    
                    if "errors" in result:
                        logger.error(f"❌ GraphQL errors: {result['errors']}")
                        return
                    
                    context = result["data"]["getContextByRecipe"]
                    logger.info(f"✅ Safety context retrieved")
                    logger.info(f"   Completeness: {context['completenessScore']:.1%}")
                    logger.info(f"   Safety flags: {len(context['safetyFlags'])}")
                    
                    # Analyze safety flags by severity
                    flag_counts = {}
                    for flag in context['safetyFlags']:
                        severity = flag['severity']
                        flag_counts[severity] = flag_counts.get(severity, 0) + 1
                    
                    logger.info("   Safety Flag Summary:")
                    for severity, count in flag_counts.items():
                        logger.info(f"     {severity}: {count} flags")
                        
                else:
                    logger.error(f"❌ Safety context request failed: {response.status_code}")
                    
        except Exception as e:
            logger.error(f"❌ Safety context test failed: {e}")
            raise

    async def test_prescription_workflow(self):
        """Test 4: Test new medication prescription workflow"""
        logger.info("\n🔍 Test 4: New Medication Prescription Workflow")
        logger.info("-" * 40)
        
        try:
            # Simulate prescribing a new medication
            prescription_data = {
                "patient_id": self.patient_id,
                "medication": {
                    "code": "310965",  # Metformin
                    "display": "Metformin 500mg tablet",
                    "system": "http://www.nlm.nih.gov/research/umls/rxnorm"
                },
                "dosage": {
                    "dose": "500mg",
                    "frequency": "twice daily",
                    "route": "oral"
                },
                "provider_id": "provider-123",
                "encounter_id": "encounter-456",
                "indication": "Type 2 Diabetes Mellitus"
            }
            
            logger.info(f"📝 Prescribing: {prescription_data['medication']['display']}")
            logger.info(f"   Dosage: {prescription_data['dosage']['dose']} {prescription_data['dosage']['frequency']}")
            logger.info(f"   Indication: {prescription_data['indication']}")
            
            # First, get prescribing context to check for contraindications
            context_query = """
            query GetPrescribingContext($patientId: String!) {
                getContextByRecipe(
                    patientId: $patientId,
                    recipeId: "medication_prescribing_v2",
                    providerId: "provider-123"
                ) {
                    safetyFlags {
                        flagType
                        severity
                        message
                    }
                    assembledData
                }
            }
            """
            
            async with httpx.AsyncClient() as client:
                # Get context first
                context_response = await client.post(
                    f"{self.context_service_url}/graphql",
                    json={"query": context_query, "variables": {"patientId": self.patient_id}},
                    headers={"Content-Type": "application/json"}
                )
                
                if context_response.status_code == 200:
                    context_result = context_response.json()
                    
                    if "errors" not in context_result:
                        context = context_result["data"]["getContextByRecipe"]
                        
                        # Check for high-severity safety flags
                        high_severity_flags = [
                            flag for flag in context['safetyFlags'] 
                            if flag['severity'] in ['HIGH', 'CRITICAL']
                        ]
                        
                        if high_severity_flags:
                            logger.warning(f"⚠️ {len(high_severity_flags)} high-severity safety flags detected")
                            for flag in high_severity_flags[:2]:
                                logger.warning(f"   - {flag['severity']}: {flag['message']}")
                        else:
                            logger.info("✅ No high-severity safety concerns detected")
                        
                        logger.info("✅ Prescription workflow validation completed")
                    else:
                        logger.error(f"❌ Context errors: {context_result['errors']}")
                else:
                    logger.error(f"❌ Context request failed: {context_response.status_code}")
                    
        except Exception as e:
            logger.error(f"❌ Prescription workflow test failed: {e}")
            raise

    async def test_drug_interactions(self):
        """Test 5: Test medication interaction checking"""
        logger.info("\n🔍 Test 5: Drug Interaction Checking")
        logger.info("-" * 40)
        
        try:
            # Get CAE integration context for drug interaction analysis
            query = """
            query GetCAEContext($patientId: String!) {
                getContextByRecipe(
                    patientId: $patientId,
                    recipeId: "cae_integration_context_v1",
                    providerId: "provider-123"
                ) {
                    contextId
                    completenessScore
                    safetyFlags {
                        flagType
                        severity
                        message
                        dataPoint
                    }
                }
            }
            """
            
            variables = {"patientId": self.patient_id}
            
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.context_service_url}/graphql",
                    json={"query": query, "variables": variables},
                    headers={"Content-Type": "application/json"}
                )
                
                if response.status_code == 200:
                    result = response.json()
                    
                    if "errors" in result:
                        logger.error(f"❌ GraphQL errors: {result['errors']}")
                        return
                    
                    context = result["data"]["getContextByRecipe"]
                    logger.info(f"✅ CAE context retrieved")
                    logger.info(f"   Context ID: {context['contextId']}")
                    logger.info(f"   Completeness: {context['completenessScore']:.1%}")
                    
                    # Look for drug interaction flags
                    interaction_flags = [
                        flag for flag in context['safetyFlags']
                        if 'interaction' in flag['message'].lower() or 'drug' in flag['flagType'].lower()
                    ]
                    
                    if interaction_flags:
                        logger.info(f"⚠️ {len(interaction_flags)} potential drug interactions detected")
                        for flag in interaction_flags[:2]:
                            logger.info(f"   - {flag['severity']}: {flag['message']}")
                    else:
                        logger.info("✅ No drug interactions detected in current analysis")
                        
                else:
                    logger.error(f"❌ CAE context request failed: {response.status_code}")
                    
        except Exception as e:
            logger.error(f"❌ Drug interaction test failed: {e}")
            raise

async def main():
    """Run the real clinical case test"""
    test = RealClinicalCaseTest()
    await test.run_comprehensive_test()

if __name__ == "__main__":
    asyncio.run(main())
