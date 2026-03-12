#!/usr/bin/env python3
"""
Test Clinical Execution Patterns.
Tests the three execution patterns: Pessimistic, Optimistic, and Digital Reflex Arc.
"""

import asyncio
import logging
import sys
import os
import time
from datetime import datetime

# Add the app directory to the Python path
sys.path.append(os.path.join(os.path.dirname(__file__), 'app'))

from app.services.clinical_execution_pattern_service import (
    clinical_execution_pattern_service, ExecutionPattern
)
from app.services.workflow_safety_integration_service import WorkflowSafetyIntegrationService

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class ClinicalExecutionPatternTest:
    """Test clinical execution patterns."""
    
    def __init__(self):
        self.workflow_service = WorkflowSafetyIntegrationService()
        self.test_results = []
        
    async def run_all_tests(self):
        """Run all execution pattern tests."""
        logger.info("🚀 Starting Clinical Execution Pattern Tests")
        logger.info("=" * 60)
        
        tests = [
            ("Test Pessimistic Pattern (High-Risk)", self.test_pessimistic_pattern),
            ("Test Optimistic Pattern (Low-Risk)", self.test_optimistic_pattern),
            ("Test Digital Reflex Arc Pattern (Autonomous)", self.test_digital_reflex_arc_pattern),
            ("Test Pattern Selection Logic", self.test_pattern_selection),
            ("Test SLA Compliance", self.test_sla_compliance),
            ("Test Pattern Integration with Workflow Service", self.test_workflow_integration)
        ]
        
        for test_name, test_func in tests:
            try:
                logger.info(f"\n📋 Running: {test_name}")
                logger.info("-" * 40)
                
                result = await test_func()
                self.test_results.append({
                    "test": test_name,
                    "status": "PASSED" if result else "FAILED",
                    "timestamp": datetime.utcnow().isoformat()
                })
                
                if result:
                    logger.info(f"✅ {test_name}: PASSED")
                else:
                    logger.error(f"❌ {test_name}: FAILED")
                    
            except Exception as e:
                logger.error(f"❌ {test_name}: ERROR - {e}")
                self.test_results.append({
                    "test": test_name,
                    "status": "ERROR",
                    "error": str(e),
                    "timestamp": datetime.utcnow().isoformat()
                })
        
        # Print summary
        self.print_test_summary()
        
    async def test_pessimistic_pattern(self) -> bool:
        """Test pessimistic execution pattern for high-risk workflows."""
        try:
            logger.info("🔒 Testing pessimistic pattern (medication prescribing)")
            
            workflow_data = {
                "medication_code": "313782",
                "medication_name": "Acetaminophen 325mg",
                "dosage": "325mg",
                "frequency": "every 6 hours"
            }
            
            execution_context = {
                "patient_id": "test-patient-pessimistic",
                "provider_id": "test-provider-123",
                "workflow_id": "test-workflow-pessimistic"
            }
            
            start_time = time.time()
            result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                workflow_type="medication_prescribing",
                pattern=ExecutionPattern.PESSIMISTIC,
                workflow_data=workflow_data,
                execution_context=execution_context
            )
            execution_time = (time.time() - start_time) * 1000
            
            # Verify pessimistic pattern characteristics
            assert result["execution_pattern"] == "pessimistic"
            assert result["execution_time_ms"] <= 250  # SLA budget
            assert "proposal" in result
            assert "safety_validation" in result
            
            # Pessimistic should wait for safety validation before commit
            if result["status"] == "completed":
                assert result["safety_validation"]["synchronous"] == True
                assert result["user_feedback"] == "wait_for_completion"
            
            logger.info(f"✅ Pessimistic pattern executed in {execution_time:.1f}ms")
            return True
            
        except Exception as e:
            logger.error(f"❌ Pessimistic pattern test failed: {e}")
            return False
    
    async def test_optimistic_pattern(self) -> bool:
        """Test optimistic execution pattern for low-risk workflows."""
        try:
            logger.info("⚡ Testing optimistic pattern (routine refill)")
            
            workflow_data = {
                "medication_code": "313782",
                "medication_name": "Acetaminophen 325mg",
                "refill_quantity": "30 tablets"
            }
            
            execution_context = {
                "patient_id": "test-patient-optimistic",
                "provider_id": "test-provider-123",
                "workflow_id": "test-workflow-optimistic"
            }
            
            start_time = time.time()
            result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                workflow_type="routine_medication_refill",
                pattern=ExecutionPattern.OPTIMISTIC,
                workflow_data=workflow_data,
                execution_context=execution_context
            )
            execution_time = (time.time() - start_time) * 1000
            
            # Verify optimistic pattern characteristics
            assert result["execution_pattern"] == "optimistic"
            assert result["execution_time_ms"] <= 150  # SLA budget
            assert "proposal" in result
            
            # Optimistic should provide immediate feedback
            assert result["user_feedback"] == "immediate_optimistic"
            assert result["compensation_available"] == True
            
            # Safety validation should be async
            if "safety_validation_task" in result:
                assert result["safety_validation_task"] == "running_async"
            
            logger.info(f"✅ Optimistic pattern executed in {execution_time:.1f}ms")
            return True
            
        except Exception as e:
            logger.error(f"❌ Optimistic pattern test failed: {e}")
            return False
    
    async def test_digital_reflex_arc_pattern(self) -> bool:
        """Test digital reflex arc pattern for autonomous workflows."""
        try:
            logger.info("🤖 Testing digital reflex arc pattern (clinical deterioration)")
            
            workflow_data = {
                "alert_type": "clinical_deterioration",
                "severity": "high",
                "interventions": ["oxygen_therapy", "iv_access", "monitoring"]
            }
            
            execution_context = {
                "patient_id": "test-patient-autonomous",
                "provider_id": "system-autonomous",
                "workflow_id": "test-workflow-autonomous"
            }
            
            start_time = time.time()
            result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                workflow_type="clinical_deterioration_response",
                pattern=ExecutionPattern.DIGITAL_REFLEX_ARC,
                workflow_data=workflow_data,
                execution_context=execution_context
            )
            execution_time = (time.time() - start_time) * 1000
            
            # Verify digital reflex arc characteristics
            assert result["execution_pattern"] == "digital_reflex_arc"
            assert result["execution_time_ms"] <= 100  # Sub-100ms requirement
            assert "proposal" in result
            
            # Should be autonomous execution
            assert result["user_feedback"] == "notification_only"
            assert result["human_intervention"] == "exception_based"
            assert result["continuous_monitoring"] == "active"
            
            logger.info(f"✅ Digital reflex arc executed in {execution_time:.1f}ms")
            return True
            
        except Exception as e:
            logger.error(f"❌ Digital reflex arc test failed: {e}")
            return False
    
    async def test_pattern_selection(self) -> bool:
        """Test automatic pattern selection based on workflow type."""
        try:
            logger.info("🎯 Testing automatic pattern selection")
            
            # Test high-risk workflow selection
            pattern = clinical_execution_pattern_service.get_pattern_for_workflow("medication_prescribing")
            assert pattern == ExecutionPattern.PESSIMISTIC
            logger.info("✅ High-risk workflow correctly selects pessimistic pattern")
            
            # Test low-risk workflow selection
            pattern = clinical_execution_pattern_service.get_pattern_for_workflow("clinical_documentation")
            assert pattern == ExecutionPattern.OPTIMISTIC
            logger.info("✅ Low-risk workflow correctly selects optimistic pattern")
            
            # Test autonomous workflow selection
            pattern = clinical_execution_pattern_service.get_pattern_for_workflow("clinical_deterioration_response")
            assert pattern == ExecutionPattern.DIGITAL_REFLEX_ARC
            logger.info("✅ Autonomous workflow correctly selects digital reflex arc pattern")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Pattern selection test failed: {e}")
            return False
    
    async def test_sla_compliance(self) -> bool:
        """Test SLA compliance for different patterns."""
        try:
            logger.info("⏱️ Testing SLA compliance")
            
            # Test each pattern's SLA budget
            patterns_to_test = [
                (ExecutionPattern.PESSIMISTIC, "medication_prescribing", 250),
                (ExecutionPattern.OPTIMISTIC, "routine_medication_refill", 150),
                (ExecutionPattern.DIGITAL_REFLEX_ARC, "clinical_deterioration_response", 100)
            ]
            
            for pattern, workflow_type, expected_sla in patterns_to_test:
                config = clinical_execution_pattern_service.patterns[pattern]
                assert config.sla_budget_ms == expected_sla
                logger.info(f"✅ {pattern.value} pattern has correct SLA: {expected_sla}ms")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ SLA compliance test failed: {e}")
            return False
    
    async def test_workflow_integration(self) -> bool:
        """Test integration with workflow safety integration service."""
        try:
            logger.info("🔗 Testing workflow service integration")
            
            # Test pattern-based workflow execution
            result = await self.workflow_service.execute_clinical_workflow_with_patterns(
                workflow_type="medication_prescribing",
                patient_id="test-patient-integration",
                provider_id="test-provider-integration",
                clinical_command={
                    "medication_code": "313782",
                    "medication_name": "Acetaminophen 325mg",
                    "dosage": "325mg"
                },
                execution_pattern=ExecutionPattern.PESSIMISTIC
            )
            
            # Verify integration result
            assert "workflow_id" in result
            assert result["execution_pattern"] == "pessimistic"
            assert "pattern_execution" in result
            assert "sla_compliance" in result
            
            logger.info(f"✅ Workflow integration successful: {result['workflow_id']}")
            return True
            
        except Exception as e:
            logger.error(f"❌ Workflow integration test failed: {e}")
            return False
    
    def print_test_summary(self):
        """Print test summary."""
        logger.info("\n" + "=" * 60)
        logger.info("📊 EXECUTION PATTERN TEST SUMMARY")
        logger.info("=" * 60)
        
        passed = sum(1 for r in self.test_results if r["status"] == "PASSED")
        failed = sum(1 for r in self.test_results if r["status"] == "FAILED")
        errors = sum(1 for r in self.test_results if r["status"] == "ERROR")
        total = len(self.test_results)
        
        logger.info(f"Total Tests: {total}")
        logger.info(f"✅ Passed: {passed}")
        logger.info(f"❌ Failed: {failed}")
        logger.info(f"⚠️ Errors: {errors}")
        logger.info(f"Success Rate: {(passed/total)*100:.1f}%")
        
        logger.info("\nDetailed Results:")
        for result in self.test_results:
            status_icon = "✅" if result["status"] == "PASSED" else "❌" if result["status"] == "FAILED" else "⚠️"
            logger.info(f"{status_icon} {result['test']}: {result['status']}")
            if "error" in result:
                logger.info(f"   Error: {result['error']}")


async def main():
    """Main test function."""
    test_runner = ClinicalExecutionPatternTest()
    await test_runner.run_all_tests()


if __name__ == "__main__":
    asyncio.run(main())
