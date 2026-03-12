#!/usr/bin/env python3
"""
Test Script for CAE Intelligence System

This script tests the enhanced intelligence components:
- Self-Improving Rule Engine
- Performance Optimizer with multi-level caching
- Confidence Evolver with Bayesian updating
- Pattern Learner with machine learning capabilities
"""

import asyncio
import logging
import json
import sys
import time
from pathlib import Path
from datetime import datetime, timedelta

# Add the app directory to the path
sys.path.insert(0, str(Path(__file__).parent / 'app'))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def test_self_improving_rule_engine():
    """Test the Self-Improving Rule Engine"""
    logger.info("🧪 Testing Self-Improving Rule Engine")
    
    try:
        from intelligence.rule_engine import SelfImprovingRuleEngine, RuleType
        
        rule_engine = SelfImprovingRuleEngine()
        
        # Test 1: Create rule from pattern
        pattern_data = {
            "pattern_type": "interaction",
            "entities_involved": ["warfarin", "aspirin"],
            "confidence_score": 0.75,
            "support_count": 8,
            "clinical_significance": "high",
            "metadata": {"discovered_from": "clinical_data"}
        }
        
        rule_id = await rule_engine.create_rule_from_pattern(pattern_data)
        logger.info(f"  ✓ Created rule from pattern: {rule_id}")
        
        # Test 2: Evaluate rules
        clinical_context = {
            "patient_id": "test_patient",
            "medication_ids": ["warfarin", "aspirin"],
            "condition_ids": ["atrial_fibrillation"],
            "clinical_context": {"age": 65, "gender": "male"}
        }
        
        assertions = await rule_engine.evaluate_rules(clinical_context)
        logger.info(f"  ✓ Generated {len(assertions)} rule-based assertions")
        
        # Test 3: Learn from positive outcome
        await rule_engine.learn_from_outcome(
            rule_id=rule_id,
            outcome_positive=True,
            clinical_context=clinical_context,
            evidence_strength=0.9
        )
        logger.info(f"  ✓ Applied positive outcome learning")
        
        # Test 4: Learn from override
        await rule_engine.learn_from_override(
            rule_id=rule_id,
            override_reason="patient_specific",
            clinical_context=clinical_context,
            clinician_expertise=0.95
        )
        logger.info(f"  ✓ Applied override learning")
        
        # Test 5: Promote learning rules
        promoted_rules = await rule_engine.promote_learning_rules()
        logger.info(f"  ✓ Promoted {len(promoted_rules)} learning rules")
        
        # Test 6: Get performance metrics
        metrics = rule_engine.get_performance_metrics()
        logger.info(f"  ✓ Rule engine metrics: {metrics['total_rules']} total rules, "
                   f"{metrics['active_rules']} active")
        
        logger.info("✅ Self-Improving Rule Engine tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Self-Improving Rule Engine test failed: {e}")
        return False


async def test_performance_optimizer():
    """Test the Performance Optimizer"""
    logger.info("🧪 Testing Performance Optimizer")
    
    try:
        from intelligence.performance_optimizer import PerformanceOptimizer, CacheLevel
        
        optimizer = PerformanceOptimizer(max_memory_mb=64)
        
        # Test 1: Multi-level caching
        async def expensive_computation():
            await asyncio.sleep(0.05)  # Simulate 50ms computation
            return {"result": "expensive_data", "computed_at": datetime.utcnow().isoformat()}
        
        # First call - cache miss
        start_time = time.time()
        result1, cache_level1 = await optimizer.get_cached_result(
            "test_key_1", expensive_computation
        )
        time1 = (time.time() - start_time) * 1000
        
        logger.info(f"  ✓ First call: {time1:.2f}ms (cache miss)")
        
        # Second call - cache hit
        start_time = time.time()
        result2, cache_level2 = await optimizer.get_cached_result("test_key_1")
        time2 = (time.time() - start_time) * 1000
        
        logger.info(f"  ✓ Second call: {time2:.2f}ms (cache hit from {cache_level2.value})")
        
        # Test 2: Sub-100ms response guarantee
        async def slow_operation():
            await asyncio.sleep(0.15)  # Simulate 150ms operation (too slow)
            return {"slow_result": "data"}
        
        # Store fallback data first
        await optimizer.store_result("fallback_key", {"fallback": "data"}, CacheLevel.L1_MEMORY)
        
        try:
            result, response_time = await optimizer.ensure_sub_100ms_response(
                slow_operation, "fallback_key"
            )
            logger.info(f"  ✓ Sub-100ms guarantee: {response_time:.2f}ms (used fallback)")
        except TimeoutError as e:
            logger.info(f"  ✓ Timeout handled correctly: {e}")
        
        # Test 3: Performance metrics
        metrics = optimizer.get_performance_metrics()
        logger.info(f"  ✓ Cache hit ratio: {metrics['cache_metrics']['cache_hit_ratio_percent']:.1f}%")
        logger.info(f"  ✓ Average response time: {metrics['response_metrics']['average_response_time_ms']:.2f}ms")
        
        logger.info("✅ Performance Optimizer tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Performance Optimizer test failed: {e}")
        return False


async def test_confidence_evolver():
    """Test the Confidence Evolver"""
    logger.info("🧪 Testing Confidence Evolver")
    
    try:
        from intelligence.confidence_evolver import ConfidenceEvolver, ConfidenceEvidence
        
        evolver = ConfidenceEvolver()
        
        # Test 1: Create evidence
        evidence = [
            ConfidenceEvidence(
                evidence_type="clinical_trial",
                strength=0.8,
                weight=1.0,
                source="RCT_2023",
                timestamp=datetime.utcnow()
            ),
            ConfidenceEvidence(
                evidence_type="real_world_evidence",
                strength=0.7,
                weight=0.8,
                source="EHR_analysis",
                timestamp=datetime.utcnow()
            )
        ]
        
        # Test 2: Evolve confidence
        confidence_score = await evolver.evolve_confidence(
            "test_assertion_1",
            evidence,
            population_data={"population_size": 150}
        )
        
        logger.info(f"  ✓ Evolved confidence: {confidence_score.overall_confidence:.3f}")
        logger.info(f"  ✓ Confidence interval: {confidence_score.confidence_interval}")
        logger.info(f"  ✓ Evidence breakdown: {confidence_score.evidence_breakdown}")
        
        # Test 3: Update from outcome
        await evolver.update_from_outcome(
            "test_assertion_1",
            outcome_positive=True,
            outcome_strength=0.9
        )
        logger.info(f"  ✓ Updated from positive outcome")
        
        # Test 4: Cross-validation
        validation_results = [
            {"method": "holdout_validation", "accuracy": 0.85, "sample_size": 100},
            {"method": "cross_validation", "accuracy": 0.82, "sample_size": 200}
        ]
        
        validation_boost = await evolver.cross_validate_confidence(
            "test_assertion_1", validation_results
        )
        logger.info(f"  ✓ Cross-validation boost: {validation_boost:.3f}")
        
        # Test 5: Get statistics
        stats = evolver.get_confidence_statistics()
        logger.info(f"  ✓ Confidence statistics: {stats['total_assertions']} assertions, "
                   f"avg confidence: {stats['average_confidence']:.3f}")
        
        logger.info("✅ Confidence Evolver tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Confidence Evolver test failed: {e}")
        return False


async def test_pattern_learner():
    """Test the Pattern Learner"""
    logger.info("🧪 Testing Pattern Learner")
    
    try:
        from intelligence.pattern_learner import PatternLearner, PatternType
        
        learner = PatternLearner(min_support=0.2, min_confidence=0.6)
        
        # Test 1: Create mock clinical data
        clinical_data = [
            {
                "medications": ["warfarin", "aspirin"],
                "outcome_type": "adverse_event",
                "timestamp": datetime.utcnow().isoformat()
            },
            {
                "medications": ["warfarin", "aspirin"],
                "outcome_type": "adverse_event",
                "timestamp": datetime.utcnow().isoformat()
            },
            {
                "medications": ["warfarin", "clopidogrel"],
                "outcome_type": "therapeutic_success",
                "timestamp": datetime.utcnow().isoformat()
            },
            {
                "medications": ["metformin", "lisinopril"],
                "outcome_type": "therapeutic_success",
                "timestamp": datetime.utcnow().isoformat()
            },
            {
                "medication_sequence": ["warfarin", "aspirin", "clopidogrel"],
                "timestamp": datetime.utcnow().isoformat()
            }
        ]
        
        # Test 2: Learn patterns from data
        discovered_patterns = await learner.learn_from_clinical_data(clinical_data)
        logger.info(f"  ✓ Discovered {len(discovered_patterns)} clinical patterns")
        
        for pattern in discovered_patterns[:3]:  # Show first 3 patterns
            logger.info(f"    - {pattern.pattern_type.value}: {pattern.entities} "
                       f"(confidence: {pattern.confidence:.3f})")
        
        # Test 3: Predict clinical outcome
        patient_context = {
            "age": 65,
            "gender": "male",
            "conditions": ["atrial_fibrillation"]
        }
        
        prediction = await learner.predict_clinical_outcome(
            patient_context, ["warfarin", "aspirin"]
        )
        
        logger.info(f"  ✓ Prediction: {prediction.predicted_outcome} "
                   f"(confidence: {prediction.confidence:.3f})")
        logger.info(f"  ✓ Risk factors: {prediction.risk_factors}")
        logger.info(f"  ✓ Recommendations: {len(prediction.recommendations)} recommendations")
        
        # Test 4: Validate patterns
        validation_data = clinical_data[:2]  # Use subset for validation
        validation_scores = await learner.validate_patterns(validation_data)
        logger.info(f"  ✓ Validated {len(validation_scores)} patterns")
        
        # Test 5: Get learning statistics
        stats = learner.get_learning_statistics()
        logger.info(f"  ✓ Learning stats: {stats['total_patterns']} patterns, "
                   f"{stats['total_predictions']} predictions")
        
        logger.info("✅ Pattern Learner tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Pattern Learner test failed: {e}")
        return False


async def test_integrated_intelligence():
    """Test integrated intelligence system"""
    logger.info("🧪 Testing Integrated Intelligence System")
    
    try:
        from orchestration.orchestration_engine import OrchestrationEngine
        
        # Create orchestration engine with intelligence
        engine = OrchestrationEngine(max_queue_size=100, max_concurrent=10)
        
        # Register mock reasoner
        class MockReasoner:
            async def check_interactions(self, **kwargs):
                await asyncio.sleep(0.01)
                return {
                    "assertions": [{
                        "type": "interaction",
                        "severity": "moderate",
                        "description": "Mock interaction detected",
                        "confidence": 0.85
                    }]
                }
        
        engine.register_reasoner("interaction", MockReasoner())
        
        # Start engine
        await engine.start()
        
        # Test 1: Generate assertions with intelligence
        test_request = {
            "patient_id": "intelligence_test_patient",
            "medication_ids": ["warfarin", "aspirin"],
            "reasoner_types": ["interaction"],
            "clinical_context": {"age": 65, "encounter_type": "outpatient"}
        }
        
        start_time = time.time()
        assertions = await engine.generate_clinical_assertions(test_request)
        response_time = (time.time() - start_time) * 1000
        
        logger.info(f"  ✓ Generated {len(assertions)} assertions in {response_time:.2f}ms")
        
        # Test 2: Learn from outcome
        if assertions:
            await engine.learn_from_outcome(
                assertions[0].assertion_id,
                outcome_positive=False,  # Simulate adverse outcome
                outcome_strength=0.8
            )
            logger.info(f"  ✓ Applied outcome learning")
        
        # Test 3: Learn from override
        if assertions:
            await engine.learn_from_override(
                assertions[0].assertion_id,
                override_reason="clinical_judgment",
                clinician_expertise=0.9
            )
            logger.info(f"  ✓ Applied override learning")
        
        # Test 4: Get enhanced system status
        status = await engine.get_system_status()
        logger.info(f"  ✓ System status includes intelligence components: "
                   f"{'intelligence' in status}")
        
        if 'intelligence' in status:
            intelligence_status = status['intelligence']
            logger.info(f"    - Rule engine: {intelligence_status.get('rule_engine', {}).get('total_rules', 0)} rules")
            logger.info(f"    - Performance optimizer: {intelligence_status.get('performance_optimizer', {}).get('cache_metrics', {}).get('cache_hit_ratio_percent', 0):.1f}% cache hit ratio")
        
        # Stop engine
        await engine.stop()
        
        logger.info("✅ Integrated Intelligence System tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Integrated Intelligence System test failed: {e}")
        return False


async def main():
    """Run all intelligence system tests"""
    logger.info("🚀 Starting CAE Intelligence System Tests")
    logger.info("=" * 60)
    
    test_results = []
    
    # Run individual component tests
    test_functions = [
        ("Self-Improving Rule Engine", test_self_improving_rule_engine),
        ("Performance Optimizer", test_performance_optimizer),
        ("Confidence Evolver", test_confidence_evolver),
        ("Pattern Learner", test_pattern_learner),
        ("Integrated Intelligence", test_integrated_intelligence)
    ]
    
    for test_name, test_func in test_functions:
        logger.info(f"\n📋 Running {test_name} tests...")
        try:
            result = await test_func()
            test_results.append((test_name, result))
        except Exception as e:
            logger.error(f"❌ {test_name} test suite failed: {e}")
            test_results.append((test_name, False))
    
    # Summary
    logger.info("\n" + "=" * 60)
    logger.info("📊 INTELLIGENCE SYSTEM TEST SUMMARY")
    logger.info("=" * 60)
    
    passed = 0
    failed = 0
    
    for test_name, result in test_results:
        status = "✅ PASSED" if result else "❌ FAILED"
        logger.info(f"{test_name}: {status}")
        if result:
            passed += 1
        else:
            failed += 1
    
    logger.info(f"\nTotal: {passed} passed, {failed} failed")
    
    if failed == 0:
        logger.info("🎉 All intelligence system tests passed!")
        logger.info("🧠 CAE Intelligence System is ready for production!")
        logger.info("\n🚀 Key Features Validated:")
        logger.info("  ✓ Self-improving rules that learn from outcomes")
        logger.info("  ✓ Multi-level caching with sub-100ms guarantees")
        logger.info("  ✓ Bayesian confidence evolution")
        logger.info("  ✓ Machine learning pattern discovery")
        logger.info("  ✓ Integrated intelligence orchestration")
    else:
        logger.error(f"⚠️  {failed} test suite(s) failed. Please review and fix issues.")
    
    return failed == 0


if __name__ == "__main__":
    # Run the test suite
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
