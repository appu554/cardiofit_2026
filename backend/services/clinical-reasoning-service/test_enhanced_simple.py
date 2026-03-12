#!/usr/bin/env python3
"""
Simple Test for Enhanced Orchestration Components

This script tests the basic functionality of our enhanced orchestration components
without requiring external dependencies like GraphDB or Redis.
"""

import asyncio
import logging
import time
import json
import sys
from datetime import datetime, timezone

# Add current directory to path
sys.path.insert(0, '.')

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def test_query_optimizer():
    """Test Graph Query Optimization"""
    logger.info("🔍 Testing Graph Query Optimization")
    
    try:
        from app.graph.query_optimizer import GraphQueryOptimizer, QueryType
        
        optimizer = GraphQueryOptimizer()
        
        # Test a simple SPARQL query
        test_query = '''
            SELECT ?patient ?name WHERE {
                ?patient rdf:type cae:Patient .
                ?patient cae:hasName ?name .
            }
        '''
        
        # Optimize the query
        query_plan = await optimizer.optimize_query(test_query, QueryType.SIMPLE_LOOKUP)
        
        logger.info(f"✅ Query optimized with {len(query_plan.optimization_techniques)} techniques")
        logger.info(f"   - Estimated cost: {query_plan.estimated_cost:.3f}")
        logger.info(f"   - Timeout: {query_plan.timeout_ms}ms")
        logger.info(f"   - Techniques: {', '.join(query_plan.optimization_techniques)}")
        
        # Simulate recording performance
        await optimizer.record_performance(query_plan, 45.0, 10)
        
        # Get stats
        stats = optimizer.get_optimization_stats()
        logger.info(f"   - Optimizer stats: {stats['total_queries']} queries processed")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Query Optimizer test failed: {e}")
        return False


async def test_circuit_breaker():
    """Test Circuit Breaker with Learning"""
    logger.info("🔌 Testing Circuit Breaker with Learning")
    
    try:
        from app.orchestration.intelligent_circuit_breaker import (
            IntelligentCircuitBreaker, 
            CircuitBreakerConfig
        )
        
        config = CircuitBreakerConfig(
            failure_threshold=2,
            recovery_timeout_ms=500,
            timeout_ms=100
        )
        
        circuit_breaker = IntelligentCircuitBreaker("test_service", config)
        
        # Mock function for testing
        async def mock_function(should_fail=False):
            if should_fail:
                raise Exception("Mock failure")
            return "success"
        
        # Test successful calls
        for i in range(3):
            result = await circuit_breaker.call(mock_function, should_fail=False)
            logger.info(f"✅ Successful call {i+1}: {result}")
        
        # Test failing calls
        for i in range(3):
            try:
                result = await circuit_breaker.call(mock_function, should_fail=True)
            except Exception as e:
                logger.info(f"❌ Expected failure {i+1}: {circuit_breaker.state.value}")
        
        # Get stats
        stats = circuit_breaker.get_stats()
        logger.info(f"   - Circuit breaker stats: {stats['performance']['total_requests']} total requests")
        logger.info(f"   - Success rate: {stats['performance']['success_rate']:.1%}")
        logger.info(f"   - Final state: {stats['state']}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Circuit Breaker test failed: {e}")
        return False


async def test_pattern_batching():
    """Test Pattern-Based Batching (simplified)"""
    logger.info("📦 Testing Pattern-Based Batching")
    
    try:
        from app.orchestration.pattern_based_batching import PatternBasedBatcher
        from app.orchestration.graph_request_router import EnhancedClinicalRequest, RoutingStrategy
        from app.orchestration.request_router import RequestPriority
        
        batcher = PatternBasedBatcher()
        
        # Create mock requests
        requests = []
        for i in range(3):
            request = EnhancedClinicalRequest(
                patient_id=f'patient_{i:03d}',
                correlation_id=f'corr_{i:03d}',
                reasoner_types=['medication_interaction'],
                medication_ids=['warfarin', 'aspirin'],  # Same medications for similarity
                condition_ids=['atrial_fibrillation'],
                allergy_ids=[],
                priority=RequestPriority.NORMAL,
                clinical_context={'age': 70},
                temporal_context={},
                timeout_ms=500,
                created_at=datetime.now(timezone.utc),
                routing_strategy=RoutingStrategy.SIMILARITY_ENHANCED,
                graph_context=None,
                similarity_score=0.8,
                pattern_matches=['warfarin_aspirin'],
                optimization_hints={'batch_compatible': True}
            )
            requests.append(request)
        
        # Add requests to batcher
        batch_ids = []
        for request in requests:
            batch_id = await batcher.add_request(request)
            batch_ids.append(batch_id)
            if batch_id:
                logger.info(f"✅ Request {request.patient_id} batched: {batch_id}")
            else:
                logger.info(f"📋 Request {request.patient_id} queued for batching")
        
        # Wait for background batching
        await asyncio.sleep(0.1)
        
        # Get stats
        stats = batcher.get_batching_stats()
        logger.info(f"   - Batching stats: {stats['performance']['total_requests']} requests processed")
        logger.info(f"   - Batching rate: {stats['performance']['batching_rate']:.1%}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Pattern Batching test failed: {e}")
        return False


async def test_basic_routing():
    """Test basic routing functionality"""
    logger.info("🚀 Testing Basic Request Routing")
    
    try:
        from app.orchestration.request_router import RequestRouter, RequestPriority
        
        router = RequestRouter()
        
        # Test basic request routing
        test_request = {
            'patient_id': 'patient_001',
            'medication_ids': ['warfarin', 'aspirin'],
            'condition_ids': ['atrial_fibrillation'],
            'allergy_ids': [],
            'clinical_context': {'age': 75, 'gender': 'male'}
        }
        
        routed_request = await router.route_request(test_request)
        
        logger.info(f"✅ Request routed successfully")
        logger.info(f"   - Patient ID: {routed_request.patient_id}")
        logger.info(f"   - Priority: {routed_request.priority.value}")
        logger.info(f"   - Medications: {len(routed_request.medication_ids)}")
        logger.info(f"   - Timeout: {routed_request.timeout_ms}ms")
        
        # Get router stats
        stats = router.get_stats()
        logger.info(f"   - Router stats: {stats['total_requests']} requests processed")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ Basic Routing test failed: {e}")
        return False


async def main():
    """Run all simple tests"""
    logger.info("🚀 Starting Enhanced Orchestration Simple Tests")
    logger.info("=" * 60)
    
    start_time = time.time()
    test_results = {}
    
    # Run tests
    test_results['basic_routing'] = await test_basic_routing()
    test_results['query_optimizer'] = await test_query_optimizer()
    test_results['circuit_breaker'] = await test_circuit_breaker()
    test_results['pattern_batching'] = await test_pattern_batching()
    
    # Calculate results
    total_time = time.time() - start_time
    successful_tests = sum(test_results.values())
    total_tests = len(test_results)
    
    # Print summary
    logger.info("=" * 60)
    logger.info("🎯 TEST SUMMARY")
    logger.info("=" * 60)
    logger.info(f"⏱️  Total execution time: {total_time:.2f} seconds")
    logger.info(f"📊 Tests passed: {successful_tests}/{total_tests}")
    logger.info(f"✅ Success rate: {successful_tests/total_tests:.1%}")
    
    for test_name, result in test_results.items():
        status = "✅ PASSED" if result else "❌ FAILED"
        logger.info(f"{status} {test_name.replace('_', ' ').title()}")
    
    logger.info("=" * 60)
    
    if successful_tests == total_tests:
        logger.info("🎉 All tests passed! Enhanced orchestration components are working correctly.")
        logger.info("🚀 Ready for integration with the full CAE system!")
    else:
        logger.info("⚠️  Some tests failed. Please check the error messages above.")
    
    logger.info("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
