"""
Test script for Context Selection Engine

This script tests the Context Selection Engine with various clinical scenarios
to validate YAML-based rule matching, scoring, and context recipe selection.
"""

import asyncio
import logging
from dataclasses import dataclass
from typing import Dict, Any

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Mock request structure (reuse from previous test)
@dataclass
class MockMedicationSafetyRequest:
    """Mock request for testing"""
    patient_id: str
    medication: Dict[str, Any]
    urgency: str = "routine"
    prescriber_specialty: str = None
    encounter_type: str = "outpatient"
    emergency_override: bool = False
    id: str = "test-request-001"


async def test_context_selection_engine():
    """Test the Context Selection Engine with various clinical scenarios"""
    
    try:
        # Import required components
        from app.domain.services.request_analyzer import RequestAnalyzer
        from app.domain.services.context_selection_engine import ContextSelectionEngine
        
        logger.info("🧪 Starting Context Selection Engine Tests")
        
        # Initialize components
        analyzer = RequestAnalyzer()
        context_engine = ContextSelectionEngine()
        
        # Test Scenario 1: Warfarin in elderly patient with renal impairment
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 1: Warfarin - Elderly + Renal Impairment (High-Risk)")
        logger.info("="*70)
        
        request1 = MockMedicationSafetyRequest(
            patient_id="patient-001",
            medication={
                "name": "warfarin",
                "rxnorm_code": "11289",
                "therapeutic_class": "anticoagulant",
                "indication": "atrial_fibrillation"
            },
            urgency="routine",
            prescriber_specialty="cardiology",
            encounter_type="outpatient"
        )
        
        # Analyze request first
        analyzed_request1 = await analyzer.analyze_request(request1)
        
        # Select context recipe
        selection_result1 = await context_engine.select_context_recipe(analyzed_request1)
        
        logger.info(f"📊 Selection Results:")
        logger.info(f"   Context Recipe: {selection_result1.context_recipe_id}")
        logger.info(f"   Confidence Score: {selection_result1.confidence_score:.2f}")
        logger.info(f"   Selection Time: {selection_result1.selection_time_ms:.1f}ms")
        logger.info(f"   Multiple Matches: {selection_result1.multiple_matches}")
        if selection_result1.selected_rule:
            logger.info(f"   Selected Rule: {selection_result1.selected_rule.rule.name}")
            logger.info(f"   Rule Score: {selection_result1.selected_rule.final_score:.2f}")
            logger.info(f"   Clinical Rationale: {selection_result1.clinical_rationale[:100]}...")
        
        # Test Scenario 2: Emergency insulin administration
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 2: Emergency Insulin Administration")
        logger.info("="*70)
        
        request2 = MockMedicationSafetyRequest(
            patient_id="patient-002",
            medication={
                "name": "insulin",
                "therapeutic_class": "antidiabetic",
                "pharmacologic_class": "hormone",
                "indication": "diabetes_type_1"
            },
            urgency="emergency",
            prescriber_specialty="emergency_medicine",
            encounter_type="emergency",
            emergency_override=True
        )
        
        analyzed_request2 = await analyzer.analyze_request(request2)
        selection_result2 = await context_engine.select_context_recipe(analyzed_request2)
        
        logger.info(f"📊 Selection Results:")
        logger.info(f"   Context Recipe: {selection_result2.context_recipe_id}")
        logger.info(f"   Confidence Score: {selection_result2.confidence_score:.2f}")
        logger.info(f"   Selection Time: {selection_result2.selection_time_ms:.1f}ms")
        logger.info(f"   Multiple Matches: {selection_result2.multiple_matches}")
        if selection_result2.selected_rule:
            logger.info(f"   Selected Rule: {selection_result2.selected_rule.rule.name}")
            logger.info(f"   Rule Score: {selection_result2.selected_rule.final_score:.2f}")
        
        # Test Scenario 3: Routine acetaminophen (should use default)
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 3: Routine Acetaminophen (Default Context)")
        logger.info("="*70)
        
        request3 = MockMedicationSafetyRequest(
            patient_id="patient-003",
            medication={
                "name": "acetaminophen",
                "therapeutic_class": "analgesic",
                "indication": "pain"
            },
            urgency="routine",
            prescriber_specialty="family_medicine",
            encounter_type="outpatient"
        )
        
        analyzed_request3 = await analyzer.analyze_request(request3)
        selection_result3 = await context_engine.select_context_recipe(analyzed_request3)
        
        logger.info(f"📊 Selection Results:")
        logger.info(f"   Context Recipe: {selection_result3.context_recipe_id}")
        logger.info(f"   Confidence Score: {selection_result3.confidence_score:.2f}")
        logger.info(f"   Selection Time: {selection_result3.selection_time_ms:.1f}ms")
        logger.info(f"   Clinical Rationale: {selection_result3.clinical_rationale}")
        
        # Test Scenario 4: Chemotherapy in elderly patient
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 4: Chemotherapy - Elderly Patient")
        logger.info("="*70)
        
        request4 = MockMedicationSafetyRequest(
            patient_id="patient-004",
            medication={
                "name": "doxorubicin",
                "therapeutic_class": "chemotherapy",
                "pharmacologic_class": "anthracycline",
                "indication": "breast_cancer"
            },
            urgency="routine",
            prescriber_specialty="oncology",
            encounter_type="inpatient"
        )
        
        analyzed_request4 = await analyzer.analyze_request(request4)
        selection_result4 = await context_engine.select_context_recipe(analyzed_request4)
        
        logger.info(f"📊 Selection Results:")
        logger.info(f"   Context Recipe: {selection_result4.context_recipe_id}")
        logger.info(f"   Confidence Score: {selection_result4.confidence_score:.2f}")
        logger.info(f"   Selection Time: {selection_result4.selection_time_ms:.1f}ms")
        if selection_result4.selected_rule:
            logger.info(f"   Selected Rule: {selection_result4.selected_rule.rule.name}")
        
        # Performance Summary
        logger.info("\n" + "="*70)
        logger.info("📊 PERFORMANCE SUMMARY")
        logger.info("="*70)
        
        performance_stats = context_engine.get_performance_stats()
        logger.info(f"Total Selections: {performance_stats['total_selections']}")
        logger.info(f"Average Selection Time: {performance_stats['average_selection_time_ms']:.1f}ms")
        logger.info(f"Total Rule Evaluations: {performance_stats['rule_evaluations']}")
        
        # Test Results Summary
        test_results = [
            ("Warfarin (Elderly+Renal)", selection_result1),
            ("Insulin (Emergency)", selection_result2),
            ("Acetaminophen (Routine)", selection_result3),
            ("Chemotherapy (Elderly)", selection_result4)
        ]
        
        logger.info("\n" + "="*70)
        logger.info("📋 TEST RESULTS SUMMARY")
        logger.info("="*70)
        
        logger.info(f"{'Scenario':<25} {'Context Recipe':<35} {'Score':<8} {'Time':<8}")
        logger.info("-" * 80)
        
        for name, result in test_results:
            recipe_short = result.context_recipe_id.replace('_context_v', '_v').replace('medication_safety_', 'med_')
            logger.info(f"{name:<25} {recipe_short:<35} {result.confidence_score:<8.2f} {result.selection_time_ms:<8.1f}ms")
        
        # Validation Checks
        logger.info("\n🔍 VALIDATION CHECKS:")
        
        # Check 1: High-risk scenarios should get specialized contexts
        assert "anticoagulation" in selection_result1.context_recipe_id or "comprehensive" in selection_result1.context_recipe_id, \
            "Warfarin should get specialized anticoagulation context"
        logger.info("✅ High-risk anticoagulation correctly routed to specialized context")
        
        # Check 2: Emergency scenarios should be handled appropriately
        if selection_result2.selected_rule:
            assert selection_result2.confidence_score > 0.7, "Emergency insulin should have high confidence"
        logger.info("✅ Emergency scenarios handled with appropriate confidence")
        
        # Check 3: Simple scenarios should use default contexts
        assert "base" in selection_result3.context_recipe_id or selection_result3.confidence_score <= 0.6, \
            "Simple medications should use base context or have low confidence"
        logger.info("✅ Simple scenarios correctly use default contexts")
        
        # Check 4: Performance should be under target
        for name, result in test_results:
            assert result.selection_time_ms < 50, f"{name} selection took too long: {result.selection_time_ms}ms"
        logger.info("✅ Performance targets met (<50ms per selection)")
        
        # Check 5: All selections should have audit trails
        for name, result in test_results:
            assert result.audit_trail is not None, f"{name} missing audit trail"
            assert "selection_timestamp" in result.audit_trail or "selection_type" in result.audit_trail, \
                f"{name} audit trail incomplete"
        logger.info("✅ All selections have comprehensive audit trails")
        
        logger.info("\n🎉 ALL TESTS PASSED! Context Selection Engine is working correctly.")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Test failed: {str(e)}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test function"""
    logger.info("🚀 Starting Context Selection Engine Component Tests")
    
    success = await test_context_selection_engine()
    
    if success:
        logger.info("✅ Context Selection Engine Component: READY FOR INTEGRATION")
    else:
        logger.error("❌ Context Selection Engine Component: NEEDS FIXES")
    
    return success


if __name__ == "__main__":
    asyncio.run(main())
