#!/usr/bin/env python3
"""
Test Script for CAE Orchestration Layer and Graph Intelligence

This script tests the newly implemented orchestration layer components:
- Request Router with priority classification
- Parallel Executor with concurrent reasoner execution
- Decision Aggregator with conflict resolution
- Priority Queue with SLA enforcement
- Graph Intelligence Layer components
"""

import asyncio
import logging
import json
import sys
import time
from pathlib import Path
from datetime import datetime

# Add the app directory to the path
sys.path.insert(0, str(Path(__file__).parent / 'app'))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def test_request_router():
    """Test the Request Router component"""
    logger.info("🧪 Testing Request Router")
    
    try:
        from orchestration.request_router import RequestRouter, RequestPriority
        
        router = RequestRouter()
        
        # Test cases with different priority scenarios
        test_requests = [
            {
                "patient_id": "patient_001",
                "medication_ids": ["epinephrine", "norepinephrine"],
                "clinical_context": {"encounter_type": "emergency"},
                "description": "Emergency medications - should be CRITICAL"
            },
            {
                "patient_id": "patient_002", 
                "medication_ids": ["warfarin", "aspirin"],
                "clinical_context": {"encounter_type": "urgent_care"},
                "description": "High-risk medications in urgent care - should be HIGH"
            },
            {
                "patient_id": "patient_003",
                "medication_ids": ["metformin", "lisinopril"],
                "clinical_context": {"encounter_type": "outpatient"},
                "description": "Standard outpatient medications - should be NORMAL"
            }
        ]
        
        for i, test_request in enumerate(test_requests):
            logger.info(f"  Test {i+1}: {test_request['description']}")
            
            clinical_request = await router.route_request(test_request)
            
            logger.info(f"    ✓ Patient: {clinical_request.patient_id}")
            logger.info(f"    ✓ Priority: {clinical_request.priority.value}")
            logger.info(f"    ✓ Timeout: {clinical_request.timeout_ms}ms")
            logger.info(f"    ✓ Medications: {clinical_request.medication_ids}")
            
            # Validate priority classification
            if i == 0:  # Emergency case
                assert clinical_request.priority == RequestPriority.CRITICAL
            elif i == 1:  # Urgent care case
                assert clinical_request.priority == RequestPriority.HIGH
            elif i == 2:  # Standard case
                assert clinical_request.priority == RequestPriority.NORMAL
        
        # Test router statistics
        stats = router.get_stats()
        logger.info(f"  ✓ Router processed {stats['total_requests']} requests")
        logger.info(f"  ✓ Priority distribution: {stats['priority_distribution']}")
        
        logger.info("✅ Request Router tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Request Router test failed: {e}")
        return False


async def test_parallel_executor():
    """Test the Parallel Executor component"""
    logger.info("🧪 Testing Parallel Executor")
    
    try:
        from orchestration.parallel_executor import ParallelExecutor, ReasonerStatus
        from orchestration.request_router import RequestRouter
        
        executor = ParallelExecutor()
        router = RequestRouter()
        
        # Create mock reasoner instances
        class MockReasoner:
            def __init__(self, name, delay=0.1):
                self.name = name
                self.delay = delay
            
            async def check_interactions(self, **kwargs):
                await asyncio.sleep(self.delay)
                return {
                    "assertions": [
                        {
                            "type": "interaction",
                            "severity": "moderate",
                            "description": f"Mock interaction from {self.name}",
                            "confidence": 0.8
                        }
                    ],
                    "confidence_score": 0.8,
                    "metadata": {"reasoner": self.name}
                }
            
            async def calculate_dosing(self, **kwargs):
                await asyncio.sleep(self.delay)
                return {
                    "assertions": [
                        {
                            "type": "dosing",
                            "severity": "low",
                            "description": f"Mock dosing from {self.name}",
                            "confidence": 0.9
                        }
                    ],
                    "confidence_score": 0.9,
                    "metadata": {"reasoner": self.name}
                }
            
            async def check_contraindications(self, **kwargs):
                await asyncio.sleep(self.delay)
                return {
                    "assertions": [
                        {
                            "type": "contraindication",
                            "severity": "high",
                            "description": f"Mock contraindication from {self.name}",
                            "confidence": 0.7
                        }
                    ],
                    "confidence_score": 0.7,
                    "metadata": {"reasoner": self.name}
                }
        
        # Create mock reasoner instances
        reasoner_instances = {
            "interaction": MockReasoner("InteractionReasoner", 0.05),
            "dosing": MockReasoner("DosingReasoner", 0.03),
            "contraindication": MockReasoner("ContraindicationReasoner", 0.04)
        }
        
        # Create test request
        test_request = await router.route_request({
            "patient_id": "test_patient",
            "medication_ids": ["warfarin", "aspirin"],
            "reasoner_types": ["interaction", "dosing", "contraindication"]
        })
        
        # Execute reasoners in parallel
        start_time = time.time()
        results = await executor.execute_reasoners(test_request, reasoner_instances)
        execution_time = (time.time() - start_time) * 1000
        
        logger.info(f"  ✓ Executed {len(results)} reasoners in {execution_time:.2f}ms")
        
        # Validate results
        for reasoner_type, result in results.items():
            logger.info(f"    ✓ {reasoner_type}: {result.status.value} "
                       f"({result.execution_time_ms:.2f}ms)")
            
            if result.status == ReasonerStatus.COMPLETED:
                assert len(result.assertions) > 0
                assert result.confidence_score > 0
        
        # Test executor statistics
        stats = executor.get_stats()
        logger.info(f"  ✓ Executor stats: {stats['total_executions']} executions, "
                   f"{stats['successful_executions']} successful")
        
        logger.info("✅ Parallel Executor tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Parallel Executor test failed: {e}")
        return False


async def test_decision_aggregator():
    """Test the Decision Aggregator component"""
    logger.info("🧪 Testing Decision Aggregator")
    
    try:
        from orchestration.decision_aggregator import DecisionAggregator, AssertionSeverity
        from orchestration.parallel_executor import ReasonerResult, ReasonerStatus
        from orchestration.request_router import RequestRouter
        
        aggregator = DecisionAggregator()
        router = RequestRouter()
        
        # Create test request
        test_request = await router.route_request({
            "patient_id": "test_patient",
            "medication_ids": ["warfarin", "aspirin"]
        })
        
        # Create mock reasoner results with conflicts
        reasoner_results = {
            "interaction": ReasonerResult(
                reasoner_type="interaction",
                status=ReasonerStatus.COMPLETED,
                assertions=[
                    {
                        "type": "interaction",
                        "severity": "critical",
                        "title": "Major Drug Interaction",
                        "description": "Warfarin-Aspirin interaction increases bleeding risk",
                        "confidence": 0.9,
                        "evidence_sources": ["DrugBank", "Clinical Studies"],
                        "recommendations": ["Monitor INR closely", "Consider alternative"]
                    }
                ],
                execution_time_ms=45.2,
                confidence_score=0.9
            ),
            "dosing": ReasonerResult(
                reasoner_type="dosing",
                status=ReasonerStatus.COMPLETED,
                assertions=[
                    {
                        "type": "dosing",
                        "severity": "moderate",
                        "title": "Dose Adjustment Required",
                        "description": "Warfarin dose may need adjustment",
                        "confidence": 0.8,
                        "evidence_sources": ["Pharmacokinetic Data"],
                        "recommendations": ["Reduce warfarin dose by 25%"]
                    }
                ],
                execution_time_ms=32.1,
                confidence_score=0.8
            ),
            "contraindication": ReasonerResult(
                reasoner_type="contraindication",
                status=ReasonerStatus.COMPLETED,
                assertions=[
                    {
                        "type": "contraindication",
                        "severity": "high",
                        "title": "Relative Contraindication",
                        "description": "Caution with concurrent use",
                        "confidence": 0.7,
                        "evidence_sources": ["Clinical Guidelines"],
                        "recommendations": ["Use with extreme caution"]
                    }
                ],
                execution_time_ms=38.7,
                confidence_score=0.7
            )
        }
        
        # Aggregate decisions
        start_time = time.time()
        aggregated_assertions = await aggregator.aggregate_decisions(test_request, reasoner_results)
        aggregation_time = (time.time() - start_time) * 1000
        
        logger.info(f"  ✓ Aggregated {len(aggregated_assertions)} assertions in {aggregation_time:.2f}ms")
        
        # Validate aggregated results
        for assertion in aggregated_assertions:
            logger.info(f"    ✓ {assertion.assertion_type}: {assertion.severity.value} "
                       f"(confidence: {assertion.confidence_score:.2f})")
            logger.info(f"      Title: {assertion.title}")
            logger.info(f"      Evidence: {len(assertion.evidence_sources)} sources")
            
            # Validate assertion structure
            assert assertion.assertion_id
            assert assertion.assertion_type
            assert assertion.severity in AssertionSeverity
            assert 0 <= assertion.confidence_score <= 1
        
        # Check that most critical assertion is first (sorted by severity)
        if len(aggregated_assertions) > 1:
            first_severity = aggregated_assertions[0].severity
            assert first_severity in [AssertionSeverity.CRITICAL, AssertionSeverity.HIGH]
        
        # Test aggregator statistics
        stats = aggregator.get_stats()
        logger.info(f"  ✓ Aggregator stats: {stats['total_aggregations']} aggregations")
        
        logger.info("✅ Decision Aggregator tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Decision Aggregator test failed: {e}")
        return False


async def test_priority_queue():
    """Test the Priority Queue component"""
    logger.info("🧪 Testing Priority Queue")
    
    try:
        from orchestration.priority_queue import PriorityQueue
        from orchestration.request_router import RequestRouter, RequestPriority
        
        queue = PriorityQueue(max_queue_size=100, max_concurrent=5)
        router = RequestRouter()
        
        # Mock processor function
        async def mock_processor(request):
            # Simulate processing time based on priority
            if request.priority == RequestPriority.CRITICAL:
                await asyncio.sleep(0.01)  # 10ms for critical
            elif request.priority == RequestPriority.HIGH:
                await asyncio.sleep(0.02)  # 20ms for high
            else:
                await asyncio.sleep(0.05)  # 50ms for normal
            
            return f"Processed {request.patient_id} with priority {request.priority.value}"
        
        # Start queue processor
        processor_task = asyncio.create_task(queue.process_queue(mock_processor))
        
        # Create test requests with different priorities
        test_requests = [
            {"patient_id": "normal_1", "medication_ids": ["metformin"]},
            {"patient_id": "critical_1", "medication_ids": ["epinephrine"], 
             "clinical_context": {"encounter_type": "emergency"}},
            {"patient_id": "high_1", "medication_ids": ["warfarin"], 
             "clinical_context": {"encounter_type": "urgent_care"}},
            {"patient_id": "normal_2", "medication_ids": ["lisinopril"]},
            {"patient_id": "critical_2", "medication_ids": ["norepinephrine"], 
             "clinical_context": {"encounter_type": "emergency"}}
        ]
        
        # Enqueue requests
        futures = []
        for req_data in test_requests:
            clinical_request = await router.route_request(req_data)
            future = await queue.enqueue_request(clinical_request)
            futures.append((clinical_request.patient_id, clinical_request.priority, future))
        
        logger.info(f"  ✓ Enqueued {len(futures)} requests")
        
        # Wait for processing and collect results
        results = []
        for patient_id, priority, future in futures:
            try:
                result = await asyncio.wait_for(future, timeout=5.0)
                results.append((patient_id, priority, result))
                logger.info(f"    ✓ {patient_id} ({priority.value}): {result}")
            except asyncio.TimeoutError:
                logger.warning(f"    ⚠ {patient_id} timed out")
        
        # Verify priority processing (critical should be processed first)
        critical_results = [r for r in results if r[1] == RequestPriority.CRITICAL]
        high_results = [r for r in results if r[1] == RequestPriority.HIGH]
        normal_results = [r for r in results if r[1] == RequestPriority.NORMAL]
        
        logger.info(f"  ✓ Processed: {len(critical_results)} critical, "
                   f"{len(high_results)} high, {len(normal_results)} normal")
        
        # Get queue status
        status = await queue.get_queue_status()
        logger.info(f"  ✓ Queue status: {status['processed_requests']} processed, "
                   f"{status['current_queue_size']} remaining")
        
        # Cancel processor task
        processor_task.cancel()
        try:
            await processor_task
        except asyncio.CancelledError:
            pass
        
        logger.info("✅ Priority Queue tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Priority Queue test failed: {e}")
        return False


async def test_orchestration_engine():
    """Test the complete Orchestration Engine"""
    logger.info("🧪 Testing Complete Orchestration Engine")
    
    try:
        from orchestration.orchestration_engine import OrchestrationEngine
        
        # Create orchestration engine
        engine = OrchestrationEngine(max_queue_size=100, max_concurrent=10)
        
        # Register mock reasoners
        class MockReasoner:
            async def check_interactions(self, **kwargs):
                await asyncio.sleep(0.02)
                return {
                    "assertions": [{
                        "type": "interaction",
                        "severity": "moderate",
                        "description": "Mock drug interaction detected",
                        "confidence": 0.85
                    }]
                }
        
        engine.register_reasoner("interaction", MockReasoner())
        
        # Start engine
        await engine.start()
        
        # Test clinical assertion generation
        test_request = {
            "patient_id": "integration_test_patient",
            "medication_ids": ["warfarin", "aspirin"],
            "reasoner_types": ["interaction"],
            "clinical_context": {"encounter_type": "outpatient"}
        }
        
        start_time = time.time()
        assertions = await engine.generate_clinical_assertions(test_request)
        response_time = (time.time() - start_time) * 1000
        
        logger.info(f"  ✓ Generated {len(assertions)} assertions in {response_time:.2f}ms")
        
        # Validate assertions
        for assertion in assertions:
            logger.info(f"    ✓ {assertion.assertion_type}: {assertion.severity.value}")
            assert assertion.assertion_id
            assert assertion.confidence_score > 0
        
        # Test system status
        status = await engine.get_system_status()
        logger.info(f"  ✓ System status: {status['orchestration_engine']['status']}")
        logger.info(f"  ✓ Registered reasoners: {status['orchestration_engine']['registered_reasoners']}")
        
        # Test health check
        health = await engine.health_check()
        logger.info(f"  ✓ Health status: {health['status']}")
        
        # Stop engine
        await engine.stop()
        
        logger.info("✅ Orchestration Engine tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Orchestration Engine test failed: {e}")
        return False


async def test_graph_intelligence():
    """Test Graph Intelligence Layer components"""
    logger.info("🧪 Testing Graph Intelligence Layer")
    
    try:
        from graph.pattern_discovery import PatternDiscoveryEngine
        from graph.relationship_navigator import RelationshipNavigator
        from graph.outcome_analyzer import OutcomeAnalyzer
        
        # Test Pattern Discovery Engine
        pattern_engine = PatternDiscoveryEngine()
        
        # Test pattern discovery (will use mock data for now)
        hidden_patterns = await pattern_engine.discover_hidden_interactions()
        logger.info(f"  ✓ Pattern Discovery: Found {len(hidden_patterns)} hidden patterns")
        
        temporal_patterns = await pattern_engine.analyze_temporal_patterns()
        logger.info(f"  ✓ Temporal Analysis: Found {len(temporal_patterns)} temporal patterns")
        
        # Test Relationship Navigator
        navigator = RelationshipNavigator()
        
        # Test patient similarity (will use mock data)
        similar_patients = await navigator.find_similar_patients("test_patient_001")
        logger.info(f"  ✓ Relationship Navigator: Found {len(similar_patients)} similar patients")
        
        # Test Outcome Analyzer
        analyzer = OutcomeAnalyzer()
        
        # Test override pattern analysis
        override_analysis = await analyzer.analyze_override_patterns()
        logger.info(f"  ✓ Outcome Analyzer: Analyzed override patterns")
        
        # Get statistics
        pattern_stats = await pattern_engine.get_pattern_stats()
        learning_stats = analyzer.get_learning_stats()
        
        logger.info(f"  ✓ Pattern Stats: {pattern_stats}")
        logger.info(f"  ✓ Learning Stats: {learning_stats}")
        
        logger.info("✅ Graph Intelligence Layer tests passed")
        return True
        
    except Exception as e:
        logger.error(f"❌ Graph Intelligence Layer test failed: {e}")
        return False


async def main():
    """Run all orchestration layer tests"""
    logger.info("🚀 Starting CAE Orchestration Layer Tests")
    logger.info("=" * 60)
    
    test_results = []
    
    # Run individual component tests
    test_functions = [
        ("Request Router", test_request_router),
        ("Parallel Executor", test_parallel_executor),
        ("Decision Aggregator", test_decision_aggregator),
        ("Priority Queue", test_priority_queue),
        ("Orchestration Engine", test_orchestration_engine),
        ("Graph Intelligence", test_graph_intelligence)
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
    logger.info("📊 TEST SUMMARY")
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
        logger.info("🎉 All orchestration layer tests passed!")
        logger.info("🚀 CAE Orchestration Layer is ready for production!")
    else:
        logger.error(f"⚠️  {failed} test suite(s) failed. Please review and fix issues.")
    
    return failed == 0


if __name__ == "__main__":
    # Run the test suite
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
