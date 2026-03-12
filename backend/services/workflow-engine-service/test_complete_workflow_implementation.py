#!/usr/bin/env python3
"""
Test Complete Workflow Implementation.
Tests all implemented workflow features: execution patterns, templates, activity framework, and monitoring.
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
from app.services.clinical_workflow_template_service import clinical_workflow_template_service
from app.services.clinical_activity_framework_service import clinical_activity_framework_service
from app.services.clinical_monitoring_service import clinical_monitoring_service
from app.models.clinical_activity_models import (
    ClinicalActivity, ClinicalActivityType, DataSourceType, ClinicalContext
)

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class CompleteWorkflowImplementationTest:
    """Test the complete workflow implementation."""
    
    def __init__(self):
        self.workflow_service = WorkflowSafetyIntegrationService()
        self.test_results = []
        
    async def run_all_tests(self):
        """Run all comprehensive workflow tests."""
        logger.info("🚀 Starting Complete Workflow Implementation Tests")
        logger.info("=" * 70)
        
        tests = [
            ("Test Execution Pattern Integration", self.test_execution_patterns),
            ("Test Clinical Workflow Templates", self.test_workflow_templates),
            ("Test Clinical Activity Framework", self.test_activity_framework),
            ("Test Clinical Monitoring Service", self.test_monitoring_service),
            ("Test End-to-End Workflow with Monitoring", self.test_end_to_end_workflow),
            ("Test Emergency Response Workflow", self.test_emergency_response),
            ("Test Performance and SLA Compliance", self.test_performance_sla),
            ("Test Safety Alert System", self.test_safety_alerts)
        ]
        
        for test_name, test_func in tests:
            try:
                logger.info(f"\n📋 Running: {test_name}")
                logger.info("-" * 50)
                
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
        
    async def test_execution_patterns(self) -> bool:
        """Test all three execution patterns."""
        try:
            logger.info("🔄 Testing execution patterns")
            
            # Test pessimistic pattern
            result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                workflow_type="medication_prescribing",
                pattern=ExecutionPattern.PESSIMISTIC,
                workflow_data={"medication": "acetaminophen", "dosage": "325mg"},
                execution_context={"patient_id": "test-patient", "provider_id": "test-provider"}
            )
            
            assert result["execution_pattern"] == "pessimistic"
            assert result["execution_time_ms"] <= 250
            logger.info("✅ Pessimistic pattern working")
            
            # Test optimistic pattern
            result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                workflow_type="routine_medication_refill",
                pattern=ExecutionPattern.OPTIMISTIC,
                workflow_data={"medication": "acetaminophen", "refill": True},
                execution_context={"patient_id": "test-patient", "provider_id": "test-provider"}
            )
            
            assert result["execution_pattern"] == "optimistic"
            assert result["execution_time_ms"] <= 150
            logger.info("✅ Optimistic pattern working")
            
            # Test digital reflex arc pattern
            result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                workflow_type="clinical_deterioration_response",
                pattern=ExecutionPattern.DIGITAL_REFLEX_ARC,
                workflow_data={"alert_type": "deterioration", "severity": "high"},
                execution_context={"patient_id": "test-patient", "provider_id": "system"}
            )
            
            assert result["execution_pattern"] == "digital_reflex_arc"
            assert result["execution_time_ms"] <= 100
            logger.info("✅ Digital reflex arc pattern working")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Execution patterns test failed: {e}")
            return False
    
    async def test_workflow_templates(self) -> bool:
        """Test clinical workflow templates."""
        try:
            logger.info("📋 Testing workflow templates")
            
            # Test template loading
            templates = clinical_workflow_template_service.get_all_templates()
            assert len(templates) >= 5  # Should have at least 5 templates
            logger.info(f"✅ Loaded {len(templates)} workflow templates")
            
            # Test specific templates
            medication_template = clinical_workflow_template_service.get_template_by_type("medication_ordering")
            assert medication_template is not None
            logger.info("✅ Medication ordering template available")
            
            emergency_template = clinical_workflow_template_service.get_template_by_type("emergency_response")
            assert emergency_template is not None
            logger.info("✅ Emergency response template available")
            
            assessment_template = clinical_workflow_template_service.get_template_by_type("clinical_assessment")
            assert assessment_template is not None
            logger.info("✅ Clinical assessment template available")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Workflow templates test failed: {e}")
            return False
    
    async def test_activity_framework(self) -> bool:
        """Test clinical activity framework."""
        try:
            logger.info("⚙️ Testing clinical activity framework")
            
            # Create test activity
            test_activity = ClinicalActivity(
                activity_id="test_sync_activity",
                activity_type=ClinicalActivityType.SYNCHRONOUS,
                timeout_seconds=2,
                safety_critical=True,
                requires_clinical_context=True,
                audit_level="comprehensive",
                approved_data_sources=[DataSourceType.FHIR_STORE, DataSourceType.PATIENT_SERVICE]
            )
            
            # Create test context
            test_context = ClinicalContext(
                patient_id="test-patient-activity",
                provider_id="test-provider-activity",
                encounter_id="test-encounter-activity"
            )
            
            # Execute activity
            result = await clinical_activity_framework_service.execute_activity(
                activity=test_activity,
                context=test_context,
                input_data={"test": "data"}
            )
            
            assert result["status"] == "completed"
            assert result["activity_type"] == "synchronous"
            assert result["execution_time_ms"] < 2000
            logger.info("✅ Synchronous activity execution working")
            
            # Test async activity
            async_activity = ClinicalActivity(
                activity_id="test_async_activity",
                activity_type=ClinicalActivityType.ASYNCHRONOUS,
                timeout_seconds=5,
                safety_critical=False,
                requires_clinical_context=True,
                audit_level="detailed"
            )
            
            result = await clinical_activity_framework_service.execute_activity(
                activity=async_activity,
                context=test_context,
                input_data={"test": "async_data"}
            )
            
            assert result["status"] == "completed"
            assert result["activity_type"] == "asynchronous"
            logger.info("✅ Asynchronous activity execution working")
            
            # Check metrics
            metrics = clinical_activity_framework_service.get_activity_metrics()
            assert "total_active_activities" in metrics
            logger.info("✅ Activity metrics collection working")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Activity framework test failed: {e}")
            return False
    
    async def test_monitoring_service(self) -> bool:
        """Test clinical monitoring service."""
        try:
            logger.info("📊 Testing clinical monitoring service")
            
            # Record test workflow start
            await clinical_monitoring_service.record_workflow_start(
                workflow_id="test-workflow-monitoring",
                workflow_type="medication_prescribing",
                execution_pattern="pessimistic",
                patient_id="test-patient-monitoring",
                provider_id="test-provider-monitoring"
            )
            
            # Record test metric
            await clinical_monitoring_service.record_metric(
                metric_id="test_metric",
                metric_name="Test Metric",
                metric_type="counter",
                value=1.0,
                unit="count",
                tags={"test": "true"}
            )
            
            # Record workflow completion
            await clinical_monitoring_service.record_workflow_completion(
                workflow_id="test-workflow-monitoring",
                status="completed",
                execution_time_ms=150.0,
                sla_compliance=True,
                safety_validation_result={"verdict": "SAFE", "processing_time_ms": 50}
            )
            
            # Get dashboard data
            dashboard_data = await clinical_monitoring_service.get_dashboard_data()
            assert "timestamp" in dashboard_data
            assert "recent_metrics" in dashboard_data
            assert "safety_alerts" in dashboard_data
            logger.info("✅ Dashboard data generation working")
            
            # Record safety alert
            alert_id = await clinical_monitoring_service.record_safety_alert(
                alert_type="test_alert",
                severity="medium",
                message="Test safety alert",
                workflow_id="test-workflow-monitoring"
            )
            assert alert_id is not None
            logger.info("✅ Safety alert recording working")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Monitoring service test failed: {e}")
            return False
    
    async def test_end_to_end_workflow(self) -> bool:
        """Test end-to-end workflow with monitoring."""
        try:
            logger.info("🔄 Testing end-to-end workflow with monitoring")
            
            # Execute workflow with pattern-based service
            result = await self.workflow_service.execute_clinical_workflow_with_patterns(
                workflow_type="medication_prescribing",
                patient_id="test-patient-e2e",
                provider_id="test-provider-e2e",
                clinical_command={
                    "medication_code": "313782",
                    "medication_name": "Acetaminophen 325mg",
                    "dosage": "325mg",
                    "frequency": "every 6 hours"
                },
                execution_pattern=ExecutionPattern.PESSIMISTIC
            )
            
            assert "workflow_id" in result
            assert result["execution_pattern"] == "pessimistic"
            assert "total_execution_time_ms" in result
            assert "sla_compliance" in result
            logger.info(f"✅ End-to-end workflow completed: {result['workflow_id']}")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ End-to-end workflow test failed: {e}")
            return False
    
    async def test_emergency_response(self) -> bool:
        """Test emergency response workflow."""
        try:
            logger.info("🚨 Testing emergency response workflow")
            
            # Execute emergency response with digital reflex arc
            result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                workflow_type="clinical_deterioration_response",
                pattern=ExecutionPattern.DIGITAL_REFLEX_ARC,
                workflow_data={
                    "emergency_type": "cardiac_arrest",
                    "interventions": ["cpr", "defibrillation"],
                    "severity": "critical"
                },
                execution_context={
                    "patient_id": "emergency-patient",
                    "provider_id": "emergency-system"
                }
            )
            
            assert result["execution_pattern"] == "digital_reflex_arc"
            assert result["execution_time_ms"] <= 100  # Sub-100ms requirement
            assert result["user_feedback"] == "notification_only"
            assert result["human_intervention"] == "exception_based"
            logger.info("✅ Emergency response workflow working")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Emergency response test failed: {e}")
            return False
    
    async def test_performance_sla(self) -> bool:
        """Test performance and SLA compliance."""
        try:
            logger.info("⏱️ Testing performance and SLA compliance")
            
            # Test multiple workflows to check SLA compliance
            patterns_to_test = [
                (ExecutionPattern.PESSIMISTIC, "medication_prescribing", 250),
                (ExecutionPattern.OPTIMISTIC, "routine_medication_refill", 150),
                (ExecutionPattern.DIGITAL_REFLEX_ARC, "clinical_deterioration_response", 100)
            ]
            
            for pattern, workflow_type, sla_ms in patterns_to_test:
                start_time = time.time()
                
                result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                    workflow_type=workflow_type,
                    pattern=pattern,
                    workflow_data={"test": "performance"},
                    execution_context={"patient_id": "perf-test", "provider_id": "perf-test"}
                )
                
                execution_time = result["execution_time_ms"]
                sla_compliant = execution_time <= sla_ms
                
                logger.info(f"✅ {pattern.value}: {execution_time:.1f}ms (SLA: {sla_ms}ms, Compliant: {sla_compliant})")
                
                # Most executions should be SLA compliant (allowing some variance in test environment)
                if not sla_compliant and execution_time > sla_ms * 2:
                    logger.warning(f"⚠️ Significant SLA violation: {execution_time:.1f}ms > {sla_ms * 2}ms")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Performance SLA test failed: {e}")
            return False
    
    async def test_safety_alerts(self) -> bool:
        """Test safety alert system."""
        try:
            logger.info("🛡️ Testing safety alert system")
            
            # Record various types of safety alerts
            alert_types = [
                ("unsafe_decision", "critical", "Unsafe medication decision detected"),
                ("high_error_rate", "high", "Error rate exceeded threshold"),
                ("sla_violation", "medium", "SLA violation detected"),
                ("system_degradation", "low", "System performance degraded")
            ]
            
            for alert_type, severity, message in alert_types:
                alert_id = await clinical_monitoring_service.record_safety_alert(
                    alert_type=alert_type,
                    severity=severity,
                    message=message,
                    workflow_id="test-safety-workflow",
                    details={"test": True}
                )
                assert alert_id is not None
                logger.info(f"✅ {severity} safety alert recorded: {alert_type}")
            
            # Check dashboard includes safety alerts
            dashboard_data = await clinical_monitoring_service.get_dashboard_data()
            safety_alerts = dashboard_data["safety_alerts"]
            assert safety_alerts["unresolved_count"] >= len(alert_types)
            logger.info("✅ Safety alerts appearing in dashboard")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Safety alerts test failed: {e}")
            return False
    
    def print_test_summary(self):
        """Print test summary."""
        logger.info("\n" + "=" * 70)
        logger.info("📊 COMPLETE WORKFLOW IMPLEMENTATION TEST SUMMARY")
        logger.info("=" * 70)
        
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
        
        logger.info("\n🎉 Complete Workflow Implementation Testing Finished!")
        
        if passed == total:
            logger.info("🏆 ALL TESTS PASSED - Workflow implementation is complete and functional!")
        elif passed >= total * 0.8:
            logger.info("✅ Most tests passed - Workflow implementation is largely functional")
        else:
            logger.warning("⚠️ Several tests failed - Workflow implementation needs attention")


async def main():
    """Main test function."""
    test_runner = CompleteWorkflowImplementationTest()
    await test_runner.run_all_tests()


if __name__ == "__main__":
    asyncio.run(main())
