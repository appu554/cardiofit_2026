#!/usr/bin/env python3
"""
Test Real GraphDB Integration and Learning Foundation

This script tests the complete integration between CAE and real GraphDB,
including outcome tracking and override tracking.
"""

import asyncio
import logging
import sys
import os
from datetime import datetime

# Add the app directory to Python path
current_dir = os.path.dirname(os.path.abspath(__file__))
app_dir = os.path.join(current_dir, 'app')
sys.path.insert(0, app_dir)

from app.graph.graphdb_client import graphdb_client
from app.learning.learning_manager import learning_manager
from app.learning.outcome_tracker import OutcomeType, OutcomeSeverity
from app.learning.override_tracker import OverrideReason
from app.core.config import settings

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class GraphDBIntegrationTester:
    """Test suite for GraphDB integration and learning foundation"""
    
    def __init__(self):
        self.test_patient_id = settings.PRIMARY_TEST_PATIENT_ID
        self.results = {
            "graphdb_connection": False,
            "patient_data_retrieval": False,
            "drug_interaction_query": False,
            "outcome_tracking": False,
            "override_tracking": False,
            "learning_insights": False,
            "total_tests": 6,
            "passed_tests": 0
        }
    
    async def run_all_tests(self):
        """Run all integration tests"""
        logger.info("🚀 Starting Real GraphDB Integration Tests")
        logger.info("=" * 60)
        
        # Test 1: GraphDB Connection
        await self.test_graphdb_connection()
        
        # Test 2: Patient Data Retrieval
        await self.test_patient_data_retrieval()
        
        # Test 3: Drug Interaction Query
        await self.test_drug_interaction_query()
        
        # Test 4: Outcome Tracking
        await self.test_outcome_tracking()
        
        # Test 5: Override Tracking
        await self.test_override_tracking()
        
        # Test 6: Learning Insights
        await self.test_learning_insights()
        
        # Summary
        self.print_summary()
    
    async def test_graphdb_connection(self):
        """Test 1: GraphDB Connection"""
        logger.info("🔍 Test 1: GraphDB Connection")
        
        try:
            # Test connection
            success = await graphdb_client.test_connection()
            
            if success:
                logger.info("✅ GraphDB connection successful")
                self.results["graphdb_connection"] = True
                self.results["passed_tests"] += 1
            else:
                logger.error("❌ GraphDB connection failed")
        
        except Exception as e:
            logger.error(f"❌ GraphDB connection error: {e}")
        
        logger.info("-" * 40)
    
    async def test_patient_data_retrieval(self):
        """Test 2: Patient Data Retrieval"""
        logger.info("🔍 Test 2: Patient Data Retrieval")
        
        try:
            # Get patient context
            result = await graphdb_client.get_patient_context(self.test_patient_id)
            
            if result.success and result.data:
                bindings = result.data.get('results', {}).get('bindings', [])
                
                if bindings:
                    logger.info(f"✅ Patient data retrieved: {len(bindings)} records")
                    
                    # Log patient details
                    patient_data = bindings[0]
                    age = patient_data.get('age', {}).get('value', 'N/A')
                    gender = patient_data.get('gender', {}).get('value', 'N/A')
                    weight = patient_data.get('weight', {}).get('value', 'N/A')
                    
                    logger.info(f"   Patient: {self.test_patient_id}")
                    logger.info(f"   Age: {age}, Gender: {gender}, Weight: {weight}")
                    
                    # Count conditions and medications
                    conditions = set()
                    medications = set()
                    
                    for record in bindings:
                        if 'conditionName' in record:
                            conditions.add(record['conditionName']['value'])
                        if 'medicationName' in record:
                            medications.add(record['medicationName']['value'])
                    
                    logger.info(f"   Conditions: {len(conditions)} - {', '.join(list(conditions)[:3])}...")
                    logger.info(f"   Medications: {len(medications)} - {', '.join(list(medications)[:3])}...")
                    
                    self.results["patient_data_retrieval"] = True
                    self.results["passed_tests"] += 1
                else:
                    logger.error("❌ No patient data found")
            else:
                logger.error(f"❌ Patient data retrieval failed: {result.error}")
        
        except Exception as e:
            logger.error(f"❌ Patient data retrieval error: {e}")
        
        logger.info("-" * 40)
    
    async def test_drug_interaction_query(self):
        """Test 3: Drug Interaction Query"""
        logger.info("🔍 Test 3: Drug Interaction Query")
        
        try:
            # Test with known interacting medications
            medications = ["warfarin", "aspirin"]
            result = await graphdb_client.check_drug_interactions(medications)
            
            if result.success and result.data:
                bindings = result.data.get('results', {}).get('bindings', [])
                
                if bindings:
                    logger.info(f"✅ Drug interactions found: {len(bindings)}")
                    
                    for interaction in bindings:
                        med1 = interaction.get('med1Name', {}).get('value', 'N/A')
                        med2 = interaction.get('med2Name', {}).get('value', 'N/A')
                        severity = interaction.get('severity', {}).get('value', 'N/A')
                        confidence = interaction.get('confidence', {}).get('value', 'N/A')
                        
                        logger.info(f"   {med1} + {med2}: {severity} (confidence: {confidence})")
                    
                    self.results["drug_interaction_query"] = True
                    self.results["passed_tests"] += 1
                else:
                    logger.warning("⚠️  No drug interactions found (expected at least warfarin + aspirin)")
            else:
                logger.error(f"❌ Drug interaction query failed: {result.error}")
        
        except Exception as e:
            logger.error(f"❌ Drug interaction query error: {e}")
        
        logger.info("-" * 40)
    
    async def test_outcome_tracking(self):
        """Test 4: Outcome Tracking"""
        logger.info("🔍 Test 4: Outcome Tracking")
        
        try:
            # Initialize learning manager
            await learning_manager.initialize()
            
            # Create test assertion ID
            assertion_id = f"test_assertion_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
            
            # Track a test outcome
            success = await learning_manager.track_clinical_outcome(
                patient_id=self.test_patient_id,
                assertion_id=assertion_id,
                outcome_type=OutcomeType.BLEEDING_EVENT.value,
                severity=OutcomeSeverity.MODERATE.value,
                description="Test bleeding event for integration testing",
                related_medications=["warfarin", "aspirin"],
                clinician_id="clinician_001"
            )
            
            if success:
                logger.info("✅ Clinical outcome tracked successfully")
                logger.info(f"   Patient: {self.test_patient_id}")
                logger.info(f"   Assertion: {assertion_id}")
                logger.info(f"   Outcome: {OutcomeType.BLEEDING_EVENT.value}")
                logger.info(f"   Severity: {OutcomeSeverity.MODERATE.value}")
                
                self.results["outcome_tracking"] = True
                self.results["passed_tests"] += 1
            else:
                logger.error("❌ Clinical outcome tracking failed")
        
        except Exception as e:
            logger.error(f"❌ Outcome tracking error: {e}")
        
        logger.info("-" * 40)
    
    async def test_override_tracking(self):
        """Test 5: Override Tracking"""
        logger.info("🔍 Test 5: Override Tracking")
        
        try:
            # Create test assertion ID
            assertion_id = f"test_assertion_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
            
            # Track a test override
            success = await learning_manager.track_clinician_override(
                patient_id=self.test_patient_id,
                assertion_id=assertion_id,
                clinician_id="clinician_001",
                override_reason=OverrideReason.PATIENT_STABLE.value,
                custom_reason="Test override for integration testing patient stable on combination",
                follow_up_required=True,
                monitoring_plan="Weekly INR monitoring for testing purposes"
            )
            
            if success:
                logger.info("✅ Clinician override tracked successfully")
                logger.info(f"   Patient: {self.test_patient_id}")
                logger.info(f"   Assertion: {assertion_id}")
                logger.info(f"   Clinician: clinician_001")
                logger.info(f"   Reason: {OverrideReason.PATIENT_STABLE.value}")
                
                self.results["override_tracking"] = True
                self.results["passed_tests"] += 1
            else:
                logger.error("❌ Clinician override tracking failed")
        
        except Exception as e:
            logger.error(f"❌ Override tracking error: {e}")
        
        logger.info("-" * 40)
    
    async def test_learning_insights(self):
        """Test 6: Learning Insights"""
        logger.info("🔍 Test 6: Learning Insights")
        
        try:
            # Get learning insights
            insights = await learning_manager.get_learning_insights(patient_id=self.test_patient_id)
            
            if insights and "error" not in insights:
                logger.info("✅ Learning insights retrieved successfully")
                
                # Log statistics
                stats = insights.get("learning_stats", {})
                logger.info(f"   Outcomes tracked: {stats.get('outcomes_tracked', 0)}")
                logger.info(f"   Overrides tracked: {stats.get('overrides_tracked', 0)}")
                logger.info(f"   Confidence updates: {stats.get('confidence_updates', 0)}")
                
                # Log outcome statistics
                outcome_stats = insights.get("outcome_statistics", {})
                outcome_data = outcome_stats.get("statistics", [])
                logger.info(f"   Recent outcomes: {len(outcome_data)} types")
                
                # Log override statistics
                override_stats = insights.get("override_statistics", {})
                override_data = override_stats.get("statistics", [])
                logger.info(f"   Recent overrides: {len(override_data)} types")
                
                self.results["learning_insights"] = True
                self.results["passed_tests"] += 1
            else:
                error = insights.get("error", "Unknown error")
                logger.error(f"❌ Learning insights failed: {error}")
        
        except Exception as e:
            logger.error(f"❌ Learning insights error: {e}")
        
        logger.info("-" * 40)
    
    def print_summary(self):
        """Print test summary"""
        logger.info("📊 TEST SUMMARY")
        logger.info("=" * 60)
        
        passed = self.results["passed_tests"]
        total = self.results["total_tests"]
        percentage = (passed / total) * 100
        
        logger.info(f"Tests Passed: {passed}/{total} ({percentage:.1f}%)")
        logger.info("")
        
        # Detailed results
        for test_name, result in self.results.items():
            if test_name not in ["total_tests", "passed_tests"]:
                status = "✅ PASS" if result else "❌ FAIL"
                test_display = test_name.replace("_", " ").title()
                logger.info(f"  {test_display}: {status}")
        
        logger.info("")
        
        if passed == total:
            logger.info("🎉 ALL TESTS PASSED! Real GraphDB integration is working perfectly!")
            logger.info("✅ Phase 1 completion requirements met:")
            logger.info("   ✅ Real GraphDB integration implemented")
            logger.info("   ✅ Learning foundation (outcome & override tracking) implemented")
        else:
            logger.warning(f"⚠️  {total - passed} test(s) failed. Please check the issues above.")
        
        logger.info("=" * 60)

async def main():
    """Main test execution"""
    tester = GraphDBIntegrationTester()
    await tester.run_all_tests()

if __name__ == "__main__":
    asyncio.run(main())
