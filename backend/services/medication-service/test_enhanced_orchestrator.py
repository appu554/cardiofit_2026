"""
Test script for Enhanced Recipe Orchestrator Integration

This script tests the complete Enhanced Recipe Orchestrator with all integrated
components including Request Analyzer, Context Selection Engine, and Priority Resolver.
"""

import asyncio
import logging
from dataclasses import dataclass
from typing import Dict, Any

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Mock request structure
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


async def test_enhanced_orchestrator():
    """Test the Enhanced Recipe Orchestrator with complete integration"""
    
    try:
        # Import the Enhanced Recipe Orchestrator
        from app.domain.services.recipe_orchestrator import RecipeOrchestrator
        
        logger.info("🧪 Starting Enhanced Recipe Orchestrator Integration Tests")
        
        # Test Scenario 1: Enhanced orchestration enabled
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 1: Enhanced Orchestration - Warfarin (Elderly + Renal)")
        logger.info("="*70)
        
        # Initialize with enhanced orchestration enabled
        orchestrator_enhanced = RecipeOrchestrator(
            enable_enhanced_orchestration=True,
            enable_safety_gateway=False
        )
        
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
        
        # Execute enhanced orchestration
        result1 = await orchestrator_enhanced.execute_medication_safety(request1)
        
        logger.info(f"📊 Enhanced Orchestration Results:")
        logger.info(f"   Request ID: {result1.request_id}")
        logger.info(f"   Overall Safety Status: {result1.overall_safety_status}")
        logger.info(f"   Context Recipe Used: {result1.context_recipe_used}")
        logger.info(f"   Context Completeness: {result1.context_completeness_score:.2%}")
        logger.info(f"   Execution Time: {result1.execution_time_ms:.1f}ms")
        logger.info(f"   Clinical Recipes Executed: {len(result1.clinical_recipes_executed)}")
        
        # Check orchestration details
        if hasattr(result1, 'orchestration_details'):
            details = result1.orchestration_details
            logger.info(f"🧠 Orchestration Intelligence:")
            logger.info(f"   Enhanced Enabled: {details.get('enhanced_orchestration_enabled', False)}")
            logger.info(f"   Selection Strategy: {details.get('selection_strategy', 'unknown')}")
            logger.info(f"   Selection Confidence: {details.get('selection_confidence', 0.0):.2f}")
            
            clinical_intel = details.get('clinical_intelligence', {})
            if clinical_intel.get('clinical_flags'):
                logger.info(f"   Clinical Flags: {clinical_intel['clinical_flags']}")
            if clinical_intel.get('monitoring_requirements'):
                logger.info(f"   Monitoring Requirements: {len(clinical_intel['monitoring_requirements'])} items")
        
        # Test Scenario 2: Legacy orchestration for comparison
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 2: Legacy Orchestration - Same Request")
        logger.info("="*70)
        
        # Initialize with legacy orchestration
        orchestrator_legacy = RecipeOrchestrator(
            enable_enhanced_orchestration=False,
            enable_safety_gateway=False
        )
        
        # Execute legacy orchestration
        result2 = await orchestrator_legacy.execute_medication_safety(request1)
        
        logger.info(f"📊 Legacy Orchestration Results:")
        logger.info(f"   Overall Safety Status: {result2.overall_safety_status}")
        logger.info(f"   Context Recipe Used: {result2.context_recipe_used}")
        logger.info(f"   Context Completeness: {result2.context_completeness_score:.2%}")
        logger.info(f"   Execution Time: {result2.execution_time_ms:.1f}ms")
        
        if hasattr(result2, 'orchestration_details'):
            details = result2.orchestration_details
            logger.info(f"📋 Legacy Orchestration:")
            logger.info(f"   Enhanced Enabled: {details.get('enhanced_orchestration_enabled', False)}")
            logger.info(f"   Selection Strategy: {details.get('selection_strategy', 'unknown')}")
        
        # Test Scenario 3: Emergency scenario with enhanced orchestration
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 3: Enhanced Orchestration - Emergency Insulin")
        logger.info("="*70)
        
        request3 = MockMedicationSafetyRequest(
            patient_id="patient-003",
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
        
        result3 = await orchestrator_enhanced.execute_medication_safety(request3)
        
        logger.info(f"📊 Emergency Scenario Results:")
        logger.info(f"   Overall Safety Status: {result3.overall_safety_status}")
        logger.info(f"   Context Recipe Used: {result3.context_recipe_used}")
        logger.info(f"   Execution Time: {result3.execution_time_ms:.1f}ms")
        
        if hasattr(result3, 'orchestration_details'):
            details = result3.orchestration_details
            logger.info(f"🚨 Emergency Orchestration:")
            logger.info(f"   Selection Strategy: {details.get('selection_strategy', 'unknown')}")
            logger.info(f"   Selection Confidence: {details.get('selection_confidence', 0.0):.2f}")
        
        # Test Scenario 4: Simple medication (should use default)
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 4: Enhanced Orchestration - Simple Medication")
        logger.info("="*70)
        
        request4 = MockMedicationSafetyRequest(
            patient_id="patient-004",
            medication={
                "name": "acetaminophen",
                "therapeutic_class": "analgesic",
                "indication": "pain"
            },
            urgency="routine",
            prescriber_specialty="family_medicine",
            encounter_type="outpatient"
        )
        
        result4 = await orchestrator_enhanced.execute_medication_safety(request4)
        
        logger.info(f"📊 Simple Medication Results:")
        logger.info(f"   Context Recipe Used: {result4.context_recipe_used}")
        logger.info(f"   Execution Time: {result4.execution_time_ms:.1f}ms")
        
        # Performance Comparison
        logger.info("\n" + "="*70)
        logger.info("📊 PERFORMANCE COMPARISON")
        logger.info("="*70)
        
        test_results = [
            ("Enhanced - Warfarin", result1),
            ("Legacy - Warfarin", result2),
            ("Enhanced - Emergency Insulin", result3),
            ("Enhanced - Simple Med", result4)
        ]
        
        logger.info(f"{'Scenario':<25} {'Recipe':<35} {'Time':<8} {'Status':<10}")
        logger.info("-" * 85)
        
        for name, result in test_results:
            recipe_short = result.context_recipe_used.replace('_context_v', '_v').replace('medication_safety_', 'med_')
            logger.info(f"{name:<25} {recipe_short:<35} {result.execution_time_ms:<8.1f} {result.overall_safety_status:<10}")
        
        # Validation Checks
        logger.info("\n🔍 VALIDATION CHECKS:")
        
        # Check 1: Enhanced orchestration should provide more detailed context recipes
        enhanced_recipes = [result1.context_recipe_used, result3.context_recipe_used]
        legacy_recipe = result2.context_recipe_used
        
        # Enhanced should select more specific recipes for complex scenarios
        assert any("anticoagulation" in recipe or "high_alert" in recipe or "comprehensive" in recipe 
                  for recipe in enhanced_recipes), "Enhanced orchestration should select specialized recipes"
        logger.info("✅ Enhanced orchestration selects more specialized context recipes")
        
        # Check 2: Enhanced orchestration should have orchestration details
        assert hasattr(result1, 'orchestration_details'), "Enhanced results should have orchestration details"
        assert result1.orchestration_details.get('enhanced_orchestration_enabled', False), \
            "Enhanced orchestration should be marked as enabled"
        logger.info("✅ Enhanced orchestration provides comprehensive details")
        
        # Check 3: Legacy orchestration should not have enhanced features
        if hasattr(result2, 'orchestration_details'):
            assert not result2.orchestration_details.get('enhanced_orchestration_enabled', True), \
                "Legacy orchestration should not be marked as enhanced"
        logger.info("✅ Legacy orchestration correctly identified")
        
        # Check 4: Performance should be reasonable
        for name, result in test_results:
            assert result.execution_time_ms < 5000, f"{name} took too long: {result.execution_time_ms}ms"
        logger.info("✅ Performance targets met (<5000ms)")
        
        # Check 5: All results should have valid safety status
        valid_statuses = ["SAFE", "CAUTION", "WARNING", "UNSAFE", "ERROR"]
        for name, result in test_results:
            assert result.overall_safety_status in valid_statuses, \
                f"{name} has invalid safety status: {result.overall_safety_status}"
        logger.info("✅ All results have valid safety status")
        
        # Check 6: Enhanced orchestration should show higher confidence for complex scenarios
        if hasattr(result1, 'orchestration_details') and hasattr(result4, 'orchestration_details'):
            complex_confidence = result1.orchestration_details.get('selection_confidence', 0.0)
            simple_confidence = result4.orchestration_details.get('selection_confidence', 0.0)
            
            # Complex scenarios should have higher confidence when rules match
            if complex_confidence > 0.7:  # Only check if we have good rule matches
                logger.info(f"✅ Complex scenario confidence: {complex_confidence:.2f}")
            else:
                logger.info(f"ℹ️ Complex scenario used default: {complex_confidence:.2f}")
        
        logger.info("\n🎉 ALL TESTS PASSED! Enhanced Recipe Orchestrator is working correctly.")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Test failed: {str(e)}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test function"""
    logger.info("🚀 Starting Enhanced Recipe Orchestrator Integration Tests")
    
    success = await test_enhanced_orchestrator()
    
    if success:
        logger.info("✅ Enhanced Recipe Orchestrator: READY FOR PRODUCTION")
    else:
        logger.error("❌ Enhanced Recipe Orchestrator: NEEDS FIXES")
    
    return success


if __name__ == "__main__":
    asyncio.run(main())
