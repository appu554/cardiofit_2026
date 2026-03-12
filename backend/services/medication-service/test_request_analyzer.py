"""
Test script for Request Analyzer Component

This script tests the Request Analyzer with various clinical scenarios
to validate multi-dimensional property extraction and clinical intelligence.
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


async def test_request_analyzer():
    """Test the Request Analyzer with various clinical scenarios"""
    
    try:
        # Import the Request Analyzer
        from app.domain.services.request_analyzer import RequestAnalyzer
        from app.domain.models.analyzed_request_models import RiskLevel, AgeGroup
        
        logger.info("🧪 Starting Request Analyzer Tests")
        
        # Initialize the analyzer
        analyzer = RequestAnalyzer()
        
        # Test Scenario 1: High-alert medication (Warfarin) in elderly patient
        logger.info("\n" + "="*60)
        logger.info("🧪 TEST 1: High-Alert Medication (Warfarin) - Elderly Patient")
        logger.info("="*60)
        
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
        
        result1 = await analyzer.analyze_request(request1)
        
        logger.info(f"📊 Analysis Results:")
        logger.info(f"   Risk Level: {result1.enriched_context.overall_risk_level.value}")
        logger.info(f"   Complexity Score: {result1.enriched_context.complexity_score:.2f}")
        logger.info(f"   High Alert: {result1.medication_properties.is_high_alert}")
        logger.info(f"   NTI: {result1.medication_properties.is_narrow_therapeutic_index}")
        logger.info(f"   Clinical Flags: {list(result1.enriched_context.clinical_flags)}")
        logger.info(f"   Monitoring Required: {result1.enriched_context.monitoring_requirements}")
        logger.info(f"   Clinical Rules Needed: {result1.requires_clinical_rules}")
        logger.info(f"   Analysis Duration: {result1.analysis_duration_ms:.1f}ms")
        
        # Test Scenario 2: Emergency chemotherapy order
        logger.info("\n" + "="*60)
        logger.info("🧪 TEST 2: Emergency Chemotherapy Order")
        logger.info("="*60)
        
        request2 = MockMedicationSafetyRequest(
            patient_id="patient-002",
            medication={
                "name": "doxorubicin",
                "therapeutic_class": "chemotherapy",
                "pharmacologic_class": "anthracycline",
                "indication": "breast_cancer"
            },
            urgency="emergency",
            prescriber_specialty="oncology",
            encounter_type="inpatient",
            emergency_override=True
        )
        
        result2 = await analyzer.analyze_request(request2)
        
        logger.info(f"📊 Analysis Results:")
        logger.info(f"   Risk Level: {result2.enriched_context.overall_risk_level.value}")
        logger.info(f"   Complexity Score: {result2.enriched_context.complexity_score:.2f}")
        logger.info(f"   Administration Complexity: {result2.medication_properties.administration_complexity.value}")
        logger.info(f"   Time Criticality: {result2.situational_properties.time_criticality_score:.2f}")
        logger.info(f"   Clinical Flags: {list(result2.enriched_context.clinical_flags)}")
        logger.info(f"   Clinical Rules Needed: {result2.requires_clinical_rules}")
        logger.info(f"   Analysis Duration: {result2.analysis_duration_ms:.1f}ms")
        
        # Test Scenario 3: Routine medication - Low complexity
        logger.info("\n" + "="*60)
        logger.info("🧪 TEST 3: Routine Medication - Low Complexity")
        logger.info("="*60)
        
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
        
        result3 = await analyzer.analyze_request(request3)
        
        logger.info(f"📊 Analysis Results:")
        logger.info(f"   Risk Level: {result3.enriched_context.overall_risk_level.value}")
        logger.info(f"   Complexity Score: {result3.enriched_context.complexity_score:.2f}")
        logger.info(f"   High Alert: {result3.medication_properties.is_high_alert}")
        logger.info(f"   Clinical Flags: {list(result3.enriched_context.clinical_flags)}")
        logger.info(f"   Clinical Rules Needed: {result3.requires_clinical_rules}")
        logger.info(f"   Analysis Duration: {result3.analysis_duration_ms:.1f}ms")
        
        # Test Scenario 4: Multiple risk factors combination
        logger.info("\n" + "="*60)
        logger.info("🧪 TEST 4: Multiple Risk Factors - Insulin in Emergency")
        logger.info("="*60)
        
        request4 = MockMedicationSafetyRequest(
            patient_id="patient-004",
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
        
        result4 = await analyzer.analyze_request(request4)
        
        logger.info(f"📊 Analysis Results:")
        logger.info(f"   Risk Level: {result4.enriched_context.overall_risk_level.value}")
        logger.info(f"   Complexity Score: {result4.enriched_context.complexity_score:.2f}")
        logger.info(f"   High Alert: {result4.medication_properties.is_high_alert}")
        logger.info(f"   Time Criticality: {result4.situational_properties.time_criticality_score:.2f}")
        logger.info(f"   Clinical Flags: {list(result4.enriched_context.clinical_flags)}")
        logger.info(f"   Clinical Rules Needed: {result4.requires_clinical_rules}")
        logger.info(f"   Analysis Duration: {result4.analysis_duration_ms:.1f}ms")
        
        # Summary
        logger.info("\n" + "="*60)
        logger.info("📋 TEST SUMMARY")
        logger.info("="*60)
        
        test_results = [
            ("Warfarin (Elderly)", result1),
            ("Chemotherapy (Emergency)", result2),
            ("Acetaminophen (Routine)", result3),
            ("Insulin (Emergency)", result4)
        ]
        
        logger.info(f"{'Scenario':<25} {'Risk':<10} {'Complexity':<12} {'Rules':<8} {'Duration':<10}")
        logger.info("-" * 70)
        
        for name, result in test_results:
            logger.info(f"{name:<25} {result.enriched_context.overall_risk_level.value:<10} "
                       f"{result.enriched_context.complexity_score:<12.2f} "
                       f"{'Yes' if result.requires_clinical_rules else 'No':<8} "
                       f"{result.analysis_duration_ms:<10.1f}ms")
        
        # Validate expected behaviors
        logger.info("\n🔍 VALIDATION CHECKS:")
        
        # Check 1: High-alert medications should be flagged
        assert result1.medication_properties.is_high_alert, "Warfarin should be high-alert"
        assert result4.medication_properties.is_high_alert, "Insulin should be high-alert"
        logger.info("✅ High-alert medications correctly identified")
        
        # Check 2: Emergency situations should have high time criticality
        assert result2.situational_properties.time_criticality_score > 0.7, "Emergency should have high criticality"
        assert result4.situational_properties.time_criticality_score > 0.7, "Emergency should have high criticality"
        logger.info("✅ Emergency situations correctly prioritized")
        
        # Check 3: Complex scenarios should require clinical rules
        assert result1.requires_clinical_rules, "Warfarin scenario should require clinical rules"
        assert result2.requires_clinical_rules, "Chemotherapy scenario should require clinical rules"
        logger.info("✅ Complex scenarios correctly flagged for clinical rules")
        
        # Check 4: Simple scenarios should not require clinical rules
        assert not result3.requires_clinical_rules, "Simple acetaminophen should not require clinical rules"
        logger.info("✅ Simple scenarios correctly identified")
        
        # Check 5: Performance should be under 50ms
        for name, result in test_results:
            assert result.analysis_duration_ms < 50, f"{name} analysis took too long: {result.analysis_duration_ms}ms"
        logger.info("✅ Performance targets met (<50ms)")
        
        logger.info("\n🎉 ALL TESTS PASSED! Request Analyzer is working correctly.")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Test failed: {str(e)}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test function"""
    logger.info("🚀 Starting Request Analyzer Component Tests")
    
    success = await test_request_analyzer()
    
    if success:
        logger.info("✅ Request Analyzer Component: READY FOR INTEGRATION")
    else:
        logger.error("❌ Request Analyzer Component: NEEDS FIXES")
    
    return success


if __name__ == "__main__":
    asyncio.run(main())
