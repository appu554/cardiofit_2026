"""
Comprehensive Tests for Multi-KB Query Router
Tests all routing patterns, fallback strategies, cache coordination, and performance monitoring
"""

import pytest
import asyncio
from unittest.mock import AsyncMock, MagicMock, patch
from datetime import datetime, timedelta

from .multi_kb_query_router import (
    MultiKBQueryRouter,
    MultiKBQueryRequest,
    MultiKBQueryResponse,
    QueryPattern,
    DataSource
)
from .cache_coordinator import CacheCoordinator
from .fallback_handler import FallbackHandler
from .performance_monitor import PerformanceMonitor


class TestMultiKBQueryRouter:
    """Test suite for Multi-KB Query Router"""

    @pytest.fixture
    async def router_config(self):
        """Test configuration for router"""
        return {
            'neo4j': {
                'uri': 'bolt://test:7687',
                'auth': ('test', 'test')
            },
            'clickhouse': {
                'databases': {
                    'kb1': {'database': 'test_kb1', 'host': 'test', 'port': 8123},
                    'kb3': {'database': 'test_kb3', 'host': 'test', 'port': 8123},
                    'kb7': {'database': 'test_kb7', 'host': 'test', 'port': 8123}
                }
            },
            'postgres': {
                'host': 'test',
                'port': 5432,
                'database': 'test'
            },
            'elasticsearch': {
                'hosts': ['http://test:9200']
            },
            'graphdb': {
                'endpoint': 'http://test:7200',
                'repository': 'test'
            },
            'redis': {
                'l2': {'host': 'test', 'port': 6379, 'db': 0},
                'l3': {'host': 'test', 'port': 6380, 'db': 0}
            },
            'monitoring': {'enabled': True},
            'caching': {'enabled': True},
            'fallback': {'enabled': True}
        }

    @pytest.fixture
    async def mock_router(self, router_config):
        """Create router with mocked dependencies"""
        router = MultiKBQueryRouter(router_config)

        # Mock all client initialization
        router._clients['neo4j_manager'] = AsyncMock()
        router._clients['postgres'] = AsyncMock()
        router._clients['elasticsearch'] = AsyncMock()
        router._clients['graphdb'] = AsyncMock()
        router._clients['clickhouse_kb1'] = AsyncMock()
        router._clients['clickhouse_kb3'] = AsyncMock()
        router._clients['clickhouse_kb7'] = AsyncMock()

        # Mark all clients as healthy
        for client in ['postgres', 'elasticsearch', 'graphdb', 'neo4j_kb1', 'neo4j_kb7', 'clickhouse_kb1']:
            router._client_health[client] = True

        # Mock component initialization
        router.cache_coordinator = AsyncMock()
        router.performance_monitor = AsyncMock()
        router.fallback_handler = AsyncMock()

        return router

    @pytest.mark.asyncio
    async def test_single_kb_terminology_lookup(self, mock_router):
        """Test single KB terminology lookup routing"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"},
            kb_id="kb7"
        )

        # Mock PostgreSQL response for exact lookup
        mock_router._clients['postgres'].query = AsyncMock(return_value={
            'code': 'I10',
            'display': 'Essential hypertension',
            'system': 'ICD10'
        })

        # Mock cache miss
        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=None)

        response = await mock_router.route_query(request)

        assert response.data['code'] == 'I10'
        assert response.kb_sources == ['kb7']
        assert 'postgres' in response.sources_used
        assert response.cache_status == "miss"

    @pytest.mark.asyncio
    async def test_semantic_inference_routing(self, mock_router):
        """Test GraphDB semantic inference routing"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.KB7_SEMANTIC_INFERENCE,
            params={"concept_id": "123", "target_system": "SNOMED"},
            kb_id="kb7"
        )

        # Mock GraphDB response
        mock_router._clients['graphdb'].query = AsyncMock(return_value={
            'concept_id': '123',
            'subsumptions': [{'code': '456', 'system': 'SNOMED'}],
            'translations': [{'code': '789', 'system': 'SNOMED'}]
        })

        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=None)

        response = await mock_router.route_query(request)

        assert 'graphdb' in response.sources_used
        assert response.data['concept_id'] == '123'
        assert 'subsumptions' in response.data

    @pytest.mark.asyncio
    async def test_cross_kb_patient_view(self, mock_router):
        """Test cross-KB patient view query orchestration"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.CROSS_KB_PATIENT_VIEW,
            params={"patient_id": "12345"},
            cross_kb_scope=["kb1", "kb7"]
        )

        # Mock Neo4j manager for cross-KB query
        mock_router._clients['neo4j_manager'].query_partition = AsyncMock(return_value={
            'patient': {'id': '12345', 'name': 'Test Patient'},
            'medications': [{'rxnorm': '123', 'name': 'Test Med'}],
            'terminology': [{'code': 'I10', 'display': 'Hypertension'}]
        })

        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=None)

        response = await mock_router.route_query(request)

        assert response.kb_sources == ["kb1", "kb7"]
        assert len(response.sources_used) >= 1
        assert response.data['patient']['id'] == '12345'

    @pytest.mark.asyncio
    async def test_cache_hit_scenario(self, mock_router):
        """Test cache hit scenario"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"},
            kb_id="kb7"
        )

        # Mock cache hit
        cached_data = {
            'data': {'code': 'I10', 'display': 'Essential hypertension'},
            'cache_layer': 'l2',
            'kb_sources': ['kb7']
        }
        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=cached_data)

        response = await mock_router.route_query(request)

        assert response.cache_status == "hit"
        assert response.sources_used == ['cache']
        assert response.data['code'] == 'I10'

    @pytest.mark.asyncio
    async def test_fallback_chain_activation(self, mock_router):
        """Test fallback chain when primary source fails"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"},
            kb_id="kb7"
        )

        # Mock primary source failure
        mock_router._clients['postgres'].query = AsyncMock(
            side_effect=Exception("Connection failed")
        )

        # Mock cache miss
        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=None)

        # Mock fallback handler success
        fallback_response = MultiKBQueryResponse(
            data={'code': 'I10', 'display': 'Essential hypertension', 'fallback': True},
            sources_used=['postgres_fallback'],
            kb_sources=['kb7'],
            latency_ms=500.0,
            cache_status='fallback',
            request_id=request.request_id
        )
        mock_router.fallback_handler.handle_error = AsyncMock(return_value=fallback_response)

        response = await mock_router.route_query(request)

        assert response.cache_status == 'fallback'
        assert 'postgres_fallback' in response.sources_used
        assert response.data['fallback'] is True

    @pytest.mark.asyncio
    async def test_cross_kb_drug_analysis_parallel_execution(self, mock_router):
        """Test parallel execution in cross-KB drug analysis"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
            params={"drug_codes": ["123", "456"], "patient_id": "12345"},
            cross_kb_scope=["kb3", "kb5", "kb7"]
        )

        # Mock all data sources for parallel execution
        mock_responses = {
            'neo4j_kb5': {'interactions': [{'drug1': '123', 'drug2': '456', 'severity': 'moderate'}]},
            'clickhouse_kb3': {'calculations': [{'drug': '123', 'dose': '10mg'}]},
            'neo4j_kb7': {'terminology': [{'code': '123', 'name': 'Test Drug'}]},
            'clickhouse_kb6': {'evidence': [{'drug': '123', 'score': 0.85}]}
        }

        # Mock _query_data_source to return appropriate responses
        async def mock_query_source(source, req):
            return mock_responses.get(source.value, {})

        mock_router._query_data_source = AsyncMock(side_effect=mock_query_source)
        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=None)

        response = await mock_router.route_query(request)

        # Verify parallel execution results
        assert len(response.sources_used) >= 3  # Multiple sources
        assert response.kb_sources == ["kb3", "kb5", "kb7"]

    @pytest.mark.asyncio
    async def test_performance_monitoring_integration(self, mock_router):
        """Test integration with performance monitor"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.KB1_PATIENT_LOOKUP,
            params={"patient_id": "12345"},
            kb_id="kb1"
        )

        mock_router._clients['neo4j_manager'].query_partition = AsyncMock(return_value={
            'patient': {'id': '12345', 'name': 'Test Patient'}
        })
        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=None)

        response = await mock_router.route_query(request)

        # Verify performance monitoring calls
        mock_router.performance_monitor.record_query_start.assert_called_once_with(request)
        mock_router.performance_monitor.record_query_complete.assert_called_once_with(request, response)

    @pytest.mark.asyncio
    async def test_circuit_breaker_functionality(self, mock_router):
        """Test circuit breaker prevents queries to unhealthy sources"""
        # Simulate circuit breaker open state
        mock_router._circuit_breakers['postgres'] = {
            'state': 'open',
            'failure_count': 5,
            'last_failure': 1000000000,  # Old timestamp
            'next_retry': 2000000000     # Future timestamp
        }

        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"},
            kb_id="kb7"
        )

        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=None)
        mock_router.fallback_handler.handle_error = AsyncMock(return_value=MultiKBQueryResponse(
            data={'error': 'Circuit breaker open'},
            sources_used=[],
            kb_sources=[],
            latency_ms=0.0,
            cache_status="error"
        ))

        response = await mock_router.route_query(request)

        assert response.cache_status == "error"
        mock_router.fallback_handler.handle_error.assert_called_once()

    @pytest.mark.asyncio
    async def test_query_timeout_handling(self, mock_router):
        """Test query timeout handling"""
        request = MultiKBQueryRequest(
            service_id="test-service",
            pattern=QueryPattern.KB3_DRUG_CALCULATION,
            params={"drug_rxnorm": "123"},
            kb_id="kb3",
            timeout_ms=100  # Very short timeout
        )

        # Mock slow response
        async def slow_query(*args, **kwargs):
            await asyncio.sleep(1)  # Longer than timeout
            return {"result": "data"}

        mock_router._clients['clickhouse_kb3'].execute_query = slow_query
        mock_router.cache_coordinator.check_cache = AsyncMock(return_value=None)

        # Should timeout and trigger fallback
        mock_router.fallback_handler.handle_error = AsyncMock(return_value=MultiKBQueryResponse(
            data={'error': 'timeout'},
            sources_used=[],
            kb_sources=[],
            latency_ms=100.0,
            cache_status="error"
        ))

        response = await mock_router.route_query(request)

        assert response.cache_status == "error"

    @pytest.mark.asyncio
    async def test_health_status_reporting(self, mock_router):
        """Test health status reporting"""
        health_status = await mock_router.get_health_status()

        assert 'router_status' in health_status
        assert 'client_health' in health_status
        assert 'circuit_breakers' in health_status
        assert 'performance_metrics' in health_status
        assert 'cache_stats' in health_status

    def test_routing_rules_initialization(self, router_config):
        """Test that routing rules are properly initialized"""
        router = MultiKBQueryRouter(router_config)

        # Verify KB routing rules
        assert 'kb7' in router.kb_routing_rules
        assert QueryPattern.KB7_TERMINOLOGY_LOOKUP in router.kb_routing_rules['kb7']
        assert QueryPattern.KB7_SEMANTIC_INFERENCE in router.kb_routing_rules['kb7']

        # Verify cross-KB routing rules
        assert QueryPattern.CROSS_KB_PATIENT_VIEW in router.cross_kb_rules
        assert QueryPattern.CROSS_KB_DRUG_ANALYSIS in router.cross_kb_rules

        # Verify GraphDB is mapped to semantic inference
        assert (router.kb_routing_rules['kb7'][QueryPattern.KB7_SEMANTIC_INFERENCE] ==
                DataSource.GRAPHDB)


class TestCacheCoordinator:
    """Test suite for Cache Coordinator"""

    @pytest.fixture
    def cache_config(self):
        return {
            'l2': {'host': 'test', 'port': 6379, 'db': 0, 'ttl': 300},
            'l3': {'host': 'test', 'port': 6380, 'db': 0, 'ttl': 3600}
        }

    @pytest.fixture
    async def cache_coordinator(self, cache_config):
        coordinator = CacheCoordinator(cache_config)
        coordinator._l2_client = AsyncMock()
        coordinator._l3_client = AsyncMock()
        return coordinator

    @pytest.mark.asyncio
    async def test_cache_key_generation(self, cache_coordinator):
        """Test cache key generation is deterministic"""
        request1 = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"},
            kb_id="kb7"
        )

        request2 = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"},
            kb_id="kb7"
        )

        key1 = cache_coordinator._generate_cache_key(request1)
        key2 = cache_coordinator._generate_cache_key(request2)

        assert key1 == key2
        assert key1.startswith("cardiofit:kb7_terminology_lookup:")

    @pytest.mark.asyncio
    async def test_l2_cache_strategy(self, cache_coordinator):
        """Test L2 cache strategy for single KB queries"""
        request = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10"},
            kb_id="kb7"
        )

        response = MultiKBQueryResponse(
            data={"code": "I10", "display": "Hypertension"},
            sources_used=["postgres"],
            kb_sources=["kb7"],
            latency_ms=100.0
        )

        # Should cache in L2 for terminology lookups
        assert cache_coordinator._should_cache_in_l2(request)
        assert not cache_coordinator._should_cache_in_l3(request)

    @pytest.mark.asyncio
    async def test_l3_cache_strategy(self, cache_coordinator):
        """Test L3 cache strategy for cross-KB queries"""
        request = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
            params={"drug_codes": ["123", "456"]},
            cross_kb_scope=["kb3", "kb5", "kb7"]
        )

        # Should cache in L3 for cross-KB queries
        assert not cache_coordinator._should_cache_in_l2(request)
        assert cache_coordinator._should_cache_in_l3(request)

    @pytest.mark.asyncio
    async def test_proactive_cache_coordination(self, cache_coordinator):
        """Test coordination with cache prefetcher"""
        event_data = {
            'workflow_id': 'wf-123',
            'patient_id': '12345',
            'recipe': {
                'conditions': ['I10', 'E11'],
                'medications': [{'rxnorm': '123'}, {'rxnorm': '456'}]
            }
        }

        predicted_queries = await cache_coordinator._predict_cache_needs(
            event_data['workflow_id'],
            event_data['patient_id'],
            event_data['recipe']
        )

        # Should predict patient lookup
        patient_keys = [k for k in predicted_queries.keys() if 'patient_lookup' in k]
        assert len(patient_keys) > 0

        # Should predict terminology lookups
        term_keys = [k for k in predicted_queries.keys() if 'terminology_lookup' in k]
        assert len(term_keys) >= 2  # Two conditions

        # Should predict interaction check
        interaction_keys = [k for k in predicted_queries.keys() if 'interaction_check' in k]
        assert len(interaction_keys) > 0


class TestFallbackHandler:
    """Test suite for Fallback Handler"""

    @pytest.fixture
    def fallback_config(self):
        return {
            'max_retries': 3,
            'base_delay_ms': 1000,
            'max_delay_ms': 30000,
            'circuit_breaker_threshold': 5,
            'circuit_breaker_timeout': 60
        }

    @pytest.fixture
    def fallback_handler(self, fallback_config):
        return FallbackHandler(fallback_config)

    @pytest.mark.asyncio
    async def test_alternative_source_fallback(self, fallback_handler):
        """Test fallback to alternative data source"""
        request = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10"},
            kb_id="kb7"
        )

        error = Exception("Neo4j connection failed")
        start_time = 1000000000.0

        # Mock successful alternative source
        async def mock_execute_alt_source(source, req):
            return {"code": "I10", "display": "Hypertension", "fallback": True}

        fallback_handler._execute_on_alternative_source = mock_execute_alt_source
        fallback_handler._is_source_healthy = AsyncMock(return_value=True)
        fallback_handler._adapt_request_for_source = AsyncMock(return_value=request)

        response = await fallback_handler.handle_error(request, error, start_time)

        assert response.cache_status == "fallback"
        assert response.data["fallback"] is True

    @pytest.mark.asyncio
    async def test_circuit_breaker_state_management(self, fallback_handler):
        """Test circuit breaker state transitions"""
        source = DataSource.POSTGRES

        # Initially closed
        assert await fallback_handler._is_source_healthy(source)

        # Record multiple failures
        for _ in range(5):
            await fallback_handler._update_circuit_breaker(source, success=False)

        # Should be open now
        assert not await fallback_handler._is_source_healthy(source)
        assert fallback_handler.circuit_breakers[source.value]['state'] == 'open'

        # Reset circuit breaker
        await fallback_handler.reset_circuit_breaker(source)
        assert await fallback_handler._is_source_healthy(source)

    @pytest.mark.asyncio
    async def test_degraded_response_creation(self, fallback_handler):
        """Test creation of degraded responses"""
        request = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10", "system": "ICD10"},
            kb_id="kb7"
        )

        degraded_response = await fallback_handler._create_degraded_response(
            request, "Service temporarily unavailable"
        )

        assert degraded_response is not None
        assert degraded_response.cache_status == "degraded"
        assert degraded_response.data["degraded"] is True
        assert degraded_response.data["code"] == "I10"


class TestPerformanceMonitor:
    """Test suite for Performance Monitor"""

    @pytest.fixture
    def monitor_config(self):
        return {
            'enabled': True,
            'retention_hours': 24,
            'slow_query_threshold_ms': 1000,
            'error_rate_threshold': 0.05
        }

    @pytest.fixture
    def performance_monitor(self, monitor_config):
        return PerformanceMonitor(monitor_config)

    @pytest.mark.asyncio
    async def test_query_metrics_recording(self, performance_monitor):
        """Test recording of query metrics"""
        request = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10"},
            kb_id="kb7"
        )

        response = MultiKBQueryResponse(
            data={"code": "I10", "display": "Hypertension"},
            sources_used=["postgres"],
            kb_sources=["kb7"],
            latency_ms=150.0,
            cache_status="miss",
            request_id="test-123"
        )

        await performance_monitor.record_query_complete(request, response)

        # Check counters
        assert performance_monitor.counters['total_queries'] == 1
        assert performance_monitor.counters['successful_queries'] == 1

        # Check metrics storage
        assert len(performance_monitor.query_metrics) == 1
        assert len(performance_monitor.latency_samples) == 1

    @pytest.mark.asyncio
    async def test_performance_stats_calculation(self, performance_monitor):
        """Test calculation of performance statistics"""
        # Add some test metrics
        for i in range(100):
            request = MultiKBQueryRequest(
                service_id="test",
                pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
                params={"code": f"I{i:02d}"},
                kb_id="kb7"
            )

            response = MultiKBQueryResponse(
                data={"code": f"I{i:02d}"},
                sources_used=["postgres"],
                kb_sources=["kb7"],
                latency_ms=100.0 + i,  # Increasing latency
                cache_status="miss" if i % 2 == 0 else "hit"
            )

            await performance_monitor.record_query_complete(request, response)

        stats = await performance_monitor.get_metrics()

        assert stats.total_queries == 100
        assert stats.successful_queries == 100
        assert stats.cache_hit_rate == 0.5  # Every other query was a hit
        assert stats.average_latency_ms > 100
        assert 'kb7' in stats.kb_query_counts

    @pytest.mark.asyncio
    async def test_slow_query_detection(self, performance_monitor):
        """Test detection of slow queries"""
        request = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.CROSS_KB_DRUG_ANALYSIS,
            params={"drug_codes": ["123"]},
            cross_kb_scope=["kb3", "kb5"]
        )

        response = MultiKBQueryResponse(
            data={"analysis": "complex"},
            sources_used=["clickhouse_kb3", "neo4j_kb5"],
            kb_sources=["kb3", "kb5"],
            latency_ms=2500.0,  # Slow query
            cache_status="miss"
        )

        await performance_monitor.record_query_complete(request, response)

        assert performance_monitor.counters['slow_queries'] == 1

        slow_queries = await performance_monitor.get_slow_queries(limit=1)
        assert len(slow_queries) == 1
        assert slow_queries[0]['latency_ms'] == 2500.0

    @pytest.mark.asyncio
    async def test_error_analysis(self, performance_monitor):
        """Test error analysis functionality"""
        # Add successful query
        success_request = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.KB7_TERMINOLOGY_LOOKUP,
            params={"code": "I10"},
            kb_id="kb7"
        )

        success_response = MultiKBQueryResponse(
            data={"code": "I10"},
            sources_used=["postgres"],
            kb_sources=["kb7"],
            latency_ms=100.0,
            cache_status="hit"
        )

        await performance_monitor.record_query_complete(success_request, success_response)

        # Add error query
        error_request = MultiKBQueryRequest(
            service_id="test",
            pattern=QueryPattern.KB5_INTERACTION_CHECK,
            params={"drug_codes": ["123"]},
            kb_id="kb5"
        )

        error_response = MultiKBQueryResponse(
            data={"error": True, "error_type": "ConnectionError"},
            sources_used=[],
            kb_sources=[],
            latency_ms=0.0,
            cache_status="error"
        )

        await performance_monitor.record_query_complete(error_request, error_response)

        error_analysis = await performance_monitor.get_error_analysis()

        assert error_analysis['total_errors'] == 1
        assert 'ConnectionError' in error_analysis['error_by_type']
        assert 'kb5_interaction_check' in error_analysis['error_by_pattern']


@pytest.mark.asyncio
async def test_end_to_end_routing_workflow():
    """End-to-end test of complete routing workflow"""
    config = {
        'neo4j': {'uri': 'bolt://test:7687'},
        'postgres': {'host': 'test'},
        'graphdb': {'endpoint': 'http://test:7200'},
        'redis': {'l2': {'host': 'test', 'port': 6379}},
        'monitoring': {'enabled': True}
    }

    router = MultiKBQueryRouter(config)

    # Mock all dependencies
    router._clients = {
        'neo4j_manager': AsyncMock(),
        'postgres': AsyncMock(),
        'graphdb': AsyncMock()
    }
    router._client_health = {'postgres': True, 'graphdb': True, 'neo4j_kb7': True}

    router.cache_coordinator = AsyncMock()
    router.performance_monitor = AsyncMock()
    router.fallback_handler = AsyncMock()

    # Test complete workflow: cache miss → query execution → caching → monitoring
    request = MultiKBQueryRequest(
        service_id="integration-test",
        pattern=QueryPattern.KB7_SEMANTIC_INFERENCE,
        params={"concept_id": "123", "target_system": "SNOMED"},
        kb_id="kb7"
    )

    # Mock cache miss
    router.cache_coordinator.check_cache = AsyncMock(return_value=None)

    # Mock successful GraphDB query
    router._clients['graphdb'].query = AsyncMock(return_value={
        'translations': [{'code': '456', 'system': 'SNOMED'}]
    })

    response = await router.route_query(request)

    # Verify complete workflow
    assert response.data['translations'][0]['code'] == '456'
    assert 'graphdb' in response.sources_used

    # Verify monitoring calls
    router.performance_monitor.record_query_start.assert_called_once()
    router.performance_monitor.record_query_complete.assert_called_once()

    # Verify caching calls
    router.cache_coordinator.cache_result.assert_called_once()


if __name__ == "__main__":
    pytest.main([__file__, "-v"])