"""
Test script for Priority Resolver Component

This script tests the Priority Resolver with various multi-match scenarios
to validate resolution strategies including additive combination, hierarchical
selection, parallel execution, and conflict resolution.
"""

import asyncio
import logging
from dataclasses import dataclass
from typing import Dict, Any, List

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


async def test_priority_resolver():
    """Test the Priority Resolver with various multi-match scenarios"""
    
    try:
        # Import required components
        from app.domain.services.request_analyzer import RequestAnalyzer
        from app.domain.services.context_selection_engine import ContextSelectionEngine
        from app.domain.services.priority_resolver import PriorityResolver
        
        logger.info("🧪 Starting Priority Resolver Tests")
        
        # Initialize components
        analyzer = RequestAnalyzer()
        context_engine = ContextSelectionEngine()
        priority_resolver = PriorityResolver()
        
        # Test Scenario 1: Complementary contexts (Elderly + Renal + Anticoagulant)
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 1: Complementary Contexts - Elderly + Renal + Anticoagulant")
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
        
        # Get analyzed request and context selection
        analyzed_request1 = await analyzer.analyze_request(request1)
        selection_result1 = await context_engine.select_context_recipe(analyzed_request1)
        
        # Test priority resolution if multiple matches
        if selection_result1.multiple_matches and len(selection_result1.matched_rules) > 1:
            resolution_result1 = await priority_resolver.resolve_multiple_matches(
                selection_result1.matched_rules, analyzed_request1
            )
            
            logger.info(f"📊 Resolution Results:")
            logger.info(f"   Strategy: {resolution_result1.resolution_strategy.value}")
            logger.info(f"   Primary Recipe: {resolution_result1.primary_recipe}")
            logger.info(f"   Secondary Recipes: {resolution_result1.secondary_recipes}")
            logger.info(f"   Confidence: {resolution_result1.confidence:.2f}")
            logger.info(f"   Resolution Time: {resolution_result1.resolution_time_ms:.1f}ms")
            logger.info(f"   Rationale: {resolution_result1.combination_rationale}")
        else:
            logger.info("📊 Single match - no resolution needed")
        
        # Test Scenario 2: Emergency insulin (multiple high-alert rules)
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 2: Multiple High-Alert Rules - Emergency Insulin")
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
        
        if selection_result2.multiple_matches and len(selection_result2.matched_rules) > 1:
            resolution_result2 = await priority_resolver.resolve_multiple_matches(
                selection_result2.matched_rules, analyzed_request2
            )
            
            logger.info(f"📊 Resolution Results:")
            logger.info(f"   Strategy: {resolution_result2.resolution_strategy.value}")
            logger.info(f"   Primary Recipe: {resolution_result2.primary_recipe}")
            logger.info(f"   Confidence: {resolution_result2.confidence:.2f}")
            logger.info(f"   Resolution Time: {resolution_result2.resolution_time_ms:.1f}ms")
            logger.info(f"   Selected Rules: {len(resolution_result2.selected_rules)}")
            logger.info(f"   Rejected Rules: {len(resolution_result2.rejected_rules)}")
        else:
            logger.info("📊 Single match - no resolution needed")
        
        # Test Scenario 3: Create artificial multi-match scenario
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 3: Artificial Multi-Match Scenario")
        logger.info("="*70)
        
        # Create mock scored rules for testing resolution strategies
        from app.domain.services.context_selection_engine import (
            ScoredRule, ContextSelectionRule, RuleTrigger, RuleScoring, RuleMatchResult
        )
        
        # Mock rules for testing
        mock_rule1 = ContextSelectionRule(
            id="TEST-001",
            name="Elderly Context Rule",
            priority=80,
            context_recipe="medication_elderly_context_v2",
            triggers=RuleTrigger(),
            scoring=RuleScoring(),
            clinical_rationale="Elderly patient considerations"
        )
        
        mock_rule2 = ContextSelectionRule(
            id="TEST-002", 
            name="Renal Impairment Rule",
            priority=85,
            context_recipe="medication_renal_context_v2",
            triggers=RuleTrigger(),
            scoring=RuleScoring(),
            clinical_rationale="Renal impairment adjustments"
        )
        
        mock_rule3 = ContextSelectionRule(
            id="TEST-003",
            name="High-Alert General Rule",
            priority=75,
            context_recipe="high_alert_general_context_v2",
            triggers=RuleTrigger(),
            scoring=RuleScoring(),
            clinical_rationale="General high-alert protocols"
        )
        
        # Create scored rules
        mock_scored_rules = [
            ScoredRule(
                rule=mock_rule1,
                match_result=RuleMatchResult(rule=mock_rule1, matches=True, match_score=0.9),
                final_score=0.85,
                clinical_priority_score=0.8,
                specificity_score=0.9,
                risk_assessment_score=0.8,
                evidence_level_score=0.9,
                scoring_rationale="Mock scoring"
            ),
            ScoredRule(
                rule=mock_rule2,
                match_result=RuleMatchResult(rule=mock_rule2, matches=True, match_score=0.95),
                final_score=0.88,
                clinical_priority_score=0.85,
                specificity_score=0.95,
                risk_assessment_score=0.85,
                evidence_level_score=0.85,
                scoring_rationale="Mock scoring"
            ),
            ScoredRule(
                rule=mock_rule3,
                match_result=RuleMatchResult(rule=mock_rule3, matches=True, match_score=0.8),
                final_score=0.75,
                clinical_priority_score=0.7,
                specificity_score=0.8,
                risk_assessment_score=0.75,
                evidence_level_score=0.8,
                scoring_rationale="Mock scoring"
            )
        ]
        
        # Test resolution with mock rules
        resolution_result3 = await priority_resolver.resolve_multiple_matches(
            mock_scored_rules, analyzed_request1
        )
        
        logger.info(f"📊 Mock Resolution Results:")
        logger.info(f"   Strategy: {resolution_result3.resolution_strategy.value}")
        logger.info(f"   Primary Recipe: {resolution_result3.primary_recipe}")
        logger.info(f"   Secondary Recipes: {resolution_result3.secondary_recipes}")
        logger.info(f"   Confidence: {resolution_result3.confidence:.2f}")
        logger.info(f"   Resolution Time: {resolution_result3.resolution_time_ms:.1f}ms")
        logger.info(f"   Selected Rules: {[r.rule.name for r in resolution_result3.selected_rules]}")
        logger.info(f"   Rationale: {resolution_result3.combination_rationale}")
        
        # Test Scenario 4: Single rule (no resolution needed)
        logger.info("\n" + "="*70)
        logger.info("🧪 TEST 4: Single Rule - No Resolution Needed")
        logger.info("="*70)
        
        single_rule_result = await priority_resolver.resolve_multiple_matches(
            [mock_scored_rules[0]], analyzed_request1
        )
        
        logger.info(f"📊 Single Rule Results:")
        logger.info(f"   Strategy: {single_rule_result.resolution_strategy.value}")
        logger.info(f"   Primary Recipe: {single_rule_result.primary_recipe}")
        logger.info(f"   Confidence: {single_rule_result.confidence:.2f}")
        
        # Performance Summary
        logger.info("\n" + "="*70)
        logger.info("📊 PERFORMANCE SUMMARY")
        logger.info("="*70)
        
        resolution_stats = priority_resolver.get_resolution_stats()
        logger.info(f"Total Resolutions: {resolution_stats['total_resolutions']}")
        logger.info(f"Average Resolution Time: {resolution_stats['average_resolution_time_ms']:.1f}ms")
        
        logger.info("Strategy Usage:")
        for strategy, count in resolution_stats['strategy_counts'].items():
            logger.info(f"  {strategy}: {count}")
        
        # Validation Checks
        logger.info("\n🔍 VALIDATION CHECKS:")
        
        # Check 1: Resolution strategies should be appropriate
        assert resolution_result3.resolution_strategy.value in [
            'single_match', 'additive_combination', 'hierarchical_selection', 
            'parallel_execution', 'conflict_resolution'
        ], "Invalid resolution strategy"
        logger.info("✅ Resolution strategies are valid")
        
        # Check 2: Single rule should use single_match strategy
        assert single_rule_result.resolution_strategy.value == 'single_match', \
            "Single rule should use single_match strategy"
        logger.info("✅ Single rule correctly uses single_match strategy")
        
        # Check 3: Resolution should maintain or improve confidence
        if len(mock_scored_rules) > 1:
            max_individual_score = max(r.final_score for r in mock_scored_rules)
            assert resolution_result3.confidence >= max_individual_score * 0.8, \
                "Resolution confidence should not drop significantly"
        logger.info("✅ Resolution maintains appropriate confidence levels")
        
        # Check 4: Performance should be reasonable
        if resolution_stats['total_resolutions'] > 0:
            assert resolution_stats['average_resolution_time_ms'] < 100, \
                f"Resolution too slow: {resolution_stats['average_resolution_time_ms']}ms"
        logger.info("✅ Performance targets met (<100ms per resolution)")
        
        # Check 5: All resolutions should have rationale
        test_results = [resolution_result3, single_rule_result]
        for i, result in enumerate(test_results):
            assert result.combination_rationale, f"Result {i+1} missing rationale"
        logger.info("✅ All resolutions have comprehensive rationale")
        
        logger.info("\n🎉 ALL TESTS PASSED! Priority Resolver is working correctly.")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Test failed: {str(e)}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test function"""
    logger.info("🚀 Starting Priority Resolver Component Tests")
    
    success = await test_priority_resolver()
    
    if success:
        logger.info("✅ Priority Resolver Component: READY FOR INTEGRATION")
    else:
        logger.error("❌ Priority Resolver Component: NEEDS FIXES")
    
    return success


if __name__ == "__main__":
    asyncio.run(main())
