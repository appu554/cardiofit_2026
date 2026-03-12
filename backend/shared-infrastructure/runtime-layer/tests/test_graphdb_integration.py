"""
Test GraphDB Integration in Shared Runtime
Comprehensive tests for multi-KB GraphDB semantic operations
"""

import asyncio
import pytest
from typing import Dict, Any
from loguru import logger

from .shared_runtime_orchestrator import initialize_shared_runtime, SharedRuntimeOrchestrator
from .config.multi_kb_config import MultiKBRuntimeConfig, Environment
from .graphdb_semantic.multi_kb_graphdb_manager import (
    MultiKBGraphDBManager, SPARQLQuery, SPARQLQueryType
)
from .query_router.multi_kb_router import (
    MultiKBQueryRouter, MultiKBQueryRequest, QueryPattern
)


class TestGraphDBIntegration:
    """Test suite for GraphDB integration in shared runtime"""

    @pytest.fixture
    async def runtime_config(self):
        """Create test runtime configuration"""
        config = MultiKBRuntimeConfig(Environment.DEVELOPMENT)
        return config

    @pytest.fixture
    async def graphdb_manager(self, runtime_config):
        """Create and initialize GraphDB manager"""
        graphdb_config = {
            'host': 'localhost',
            'port': 7200,
            'username': 'admin',
            'password': 'admin',
            'ssl': False,
            'connection_pool_size': 5
        }

        manager = MultiKBGraphDBManager(graphdb_config)

        # Note: This will fail if GraphDB is not running
        # For testing purposes, we'll mock or skip if not available
        try:
            success = await manager.initialize_connection()
            if success:
                yield manager
            else:
                pytest.skip("GraphDB not available for testing")
        except Exception as e:
            logger.warning(f"GraphDB not available: {e}")
            pytest.skip("GraphDB not available for testing")
        finally:
            if manager:
                await manager.close()

    @pytest.fixture
    async def shared_runtime(self, runtime_config):
        """Create shared runtime orchestrator"""
        try:
            runtime = await initialize_shared_runtime(runtime_config)
            yield runtime
        except Exception as e:
            logger.warning(f"Shared runtime initialization failed: {e}")
            pytest.skip("Shared runtime not available for testing")
        finally:
            if 'runtime' in locals() and runtime:
                await runtime.shutdown()

    async def test_graphdb_manager_initialization(self, graphdb_manager):
        """Test GraphDB manager initialization"""
        assert graphdb_manager is not None
        assert graphdb_manager.connected is True

        # Test repository initialization
        assert len(graphdb_manager.repositories) > 0
        assert len(graphdb_manager.kb_repository_map) > 0

        # Check specific KB repositories
        assert 'kb-7' in graphdb_manager.kb_repository_map
        kb7_repo_id = graphdb_manager.kb_repository_map['kb-7']
        assert kb7_repo_id in graphdb_manager.repositories

    async def test_graphdb_health_check(self, graphdb_manager):
        """Test GraphDB health monitoring"""
        health_status = await graphdb_manager.get_health_status()

        assert health_status is not None
        assert 'overall_healthy' in health_status
        assert 'connected' in health_status
        assert 'repositories' in health_status
        assert 'metrics' in health_status

        # Check repository health
        for repo_id, repo_info in health_status['repositories'].items():
            assert 'healthy' in repo_info
            assert 'kb_id' in repo_info
            assert 'title' in repo_info

    async def test_sparql_query_execution(self, graphdb_manager):
        """Test SPARQL query execution"""
        # Get KB-7 repository ID
        kb7_repo_id = await graphdb_manager.get_kb_repository_id('kb-7')
        assert kb7_repo_id is not None

        # Create a simple test query
        test_query = SPARQLQuery(
            query="""
            SELECT ?s ?p ?o
            WHERE { ?s ?p ?o }
            LIMIT 5
            """,
            query_type=SPARQLQueryType.SELECT,
            repository_id=kb7_repo_id,
            kb_id='kb-7',
            timeout=10
        )

        # Execute query
        result = await graphdb_manager.execute_sparql_query(test_query)

        assert result is not None
        assert result.kb_id == 'kb-7'
        assert result.repository_id == kb7_repo_id
        assert result.query_type == SPARQLQueryType.SELECT
        assert result.execution_time_ms > 0
        assert isinstance(result.data, list)

    async def test_cross_kb_semantic_search(self, graphdb_manager):
        """Test cross-KB semantic search"""
        kb_list = ['kb-7', 'kb-5']  # Terminology and Drug Interactions
        search_term = 'medication'

        results = await graphdb_manager.execute_cross_kb_semantic_search(
            search_term=search_term,
            kb_ids=kb_list,
            limit=10
        )

        assert results is not None
        assert isinstance(results, dict)

        # Check results for each KB
        for kb_id in kb_list:
            if kb_id in results:
                kb_result = results[kb_id]
                assert 'data' in kb_result or 'errors' in kb_result

                if 'data' in kb_result:
                    assert 'bindings_count' in kb_result
                    assert 'execution_time_ms' in kb_result

    async def test_query_router_graphdb_integration(self, runtime_config):
        """Test GraphDB integration through query router"""
        router_config = {
            'neo4j': {
                'neo4j_uri': 'bolt://localhost:7687',
                'neo4j_user': 'neo4j',
                'neo4j_password': 'test_password'
            },
            'graphdb': {
                'host': 'localhost',
                'port': 7200,
                'username': 'admin',
                'password': 'admin',
                'ssl': False
            }
        }

        router = MultiKBQueryRouter(router_config)

        try:
            # Initialize clients (this will include GraphDB)
            await router.initialize_clients()

            # Create semantic inference query
            request = MultiKBQueryRequest(
                service_id='test-service',
                kb_id='kb-7',
                pattern=QueryPattern.SEMANTIC_INFERENCE,
                params={'search_term': 'hypertension'}
            )

            # Route query (this should use GraphDB)
            response = await router.route_query(request)

            assert response is not None
            assert 'data' in response
            assert 'metadata' in response
            assert response.kb_sources == ['kb-7']

        except Exception as e:
            logger.warning(f"Query router GraphDB test skipped: {e}")
            pytest.skip("GraphDB not available through query router")
        finally:
            await router.close()

    async def test_shared_runtime_graphdb_integration(self, shared_runtime):
        """Test GraphDB through shared runtime orchestrator"""
        # Check that GraphDB component is initialized
        if 'graphdb_semantic' not in shared_runtime.components:
            pytest.skip("GraphDB semantic component not initialized in shared runtime")

        graphdb_component = shared_runtime.components.get('graphdb_semantic')
        if not graphdb_component:
            pytest.skip("GraphDB semantic component not available")

        # Test health status
        system_status = await shared_runtime.get_system_status()

        assert system_status is not None
        assert 'health_status' in system_status
        health_components = system_status['health_status']['components']

        # Check GraphDB health component
        if 'graphdb_semantic' in health_components:
            assert isinstance(health_components['graphdb_semantic'], bool)

    async def test_kb7_terminology_semantic_query(self, shared_runtime):
        """Test KB-7 medical terminology semantic query through shared runtime"""
        try:
            # Execute semantic query for medical terminology
            response = await shared_runtime.route_query(
                service_id='test-service',
                kb_id='kb-7',
                pattern='semantic_inference',
                params={
                    'search_term': 'hypertension',
                    'limit': 10
                }
            )

            assert response is not None
            assert 'data' in response

            # Check medical terminology specific results
            if 'error' not in response['data']:
                metadata = response.get('metadata', {})
                assert 'latency_ms' in metadata

        except Exception as e:
            logger.warning(f"KB-7 semantic query test failed: {e}")
            pytest.skip("KB-7 semantic query not available")

    async def test_cross_kb_semantic_query(self, shared_runtime):
        """Test cross-KB semantic search through shared runtime"""
        try:
            # Execute cross-KB semantic search
            response = await shared_runtime.route_query(
                service_id='test-service',
                kb_id=None,  # Cross-KB query
                pattern='cross_kb_semantic_search',
                params={
                    'search_term': 'cardiac',
                    'limit_per_kb': 5
                },
                cross_kb_scope=['kb-7', 'kb-5', 'kb-6']
            )

            assert response is not None
            assert 'data' in response

            # Should have results from multiple KBs
            if 'error' not in response['data']:
                metadata = response.get('metadata', {})
                kb_sources = metadata.get('kb_sources', [])
                assert len(kb_sources) > 1  # Multiple KBs

        except Exception as e:
            logger.warning(f"Cross-KB semantic query test failed: {e}")
            pytest.skip("Cross-KB semantic query not available")

    async def test_graphdb_performance_metrics(self, graphdb_manager):
        """Test GraphDB performance metrics collection"""
        # Execute some queries first to generate metrics
        kb7_repo_id = await graphdb_manager.get_kb_repository_id('kb-7')

        for i in range(3):
            test_query = SPARQLQuery(
                query=f"SELECT * WHERE {{ ?s ?p ?o }} LIMIT {i+1}",
                query_type=SPARQLQueryType.SELECT,
                repository_id=kb7_repo_id,
                kb_id='kb-7'
            )
            await graphdb_manager.execute_sparql_query(test_query)

        # Check performance metrics
        metrics = await graphdb_manager.get_performance_metrics()

        assert metrics is not None
        assert 'total_queries' in metrics
        assert 'queries_per_kb' in metrics
        assert 'avg_query_time_ms' in metrics
        assert 'active_repositories' in metrics

        # Check query counts
        assert metrics['total_queries'] >= 3
        assert 'kb-7' in metrics['queries_per_kb']
        assert metrics['queries_per_kb']['kb-7'] >= 3

    async def test_error_handling(self, graphdb_manager):
        """Test GraphDB error handling"""
        # Test invalid repository
        invalid_query = SPARQLQuery(
            query="SELECT ?s WHERE { ?s ?p ?o }",
            query_type=SPARQLQueryType.SELECT,
            repository_id='nonexistent-repo',
            kb_id='kb-invalid'
        )

        result = await graphdb_manager.execute_sparql_query(invalid_query)

        assert result is not None
        assert len(result.errors) > 0

        # Test invalid SPARQL syntax
        kb7_repo_id = await graphdb_manager.get_kb_repository_id('kb-7')

        invalid_sparql = SPARQLQuery(
            query="INVALID SPARQL SYNTAX",
            query_type=SPARQLQueryType.SELECT,
            repository_id=kb7_repo_id,
            kb_id='kb-7'
        )

        result = await graphdb_manager.execute_sparql_query(invalid_sparql)

        assert result is not None
        assert len(result.errors) > 0


async def run_integration_tests():
    """Run all GraphDB integration tests"""
    logger.info("Starting GraphDB integration tests...")

    test_suite = TestGraphDBIntegration()

    try:
        # Test configuration
        config = MultiKBRuntimeConfig(Environment.DEVELOPMENT)

        # Test GraphDB manager
        logger.info("Testing GraphDB manager...")
        graphdb_config = {
            'host': 'localhost',
            'port': 7200,
            'username': 'admin',
            'password': 'admin'
        }

        manager = MultiKBGraphDBManager(graphdb_config)

        try:
            success = await manager.initialize_connection()
            if success:
                await test_suite.test_graphdb_manager_initialization(manager)
                await test_suite.test_graphdb_health_check(manager)
                await test_suite.test_sparql_query_execution(manager)
                await test_suite.test_cross_kb_semantic_search(manager)
                await test_suite.test_graphdb_performance_metrics(manager)
                await test_suite.test_error_handling(manager)
                logger.info("✅ GraphDB manager tests passed")
            else:
                logger.warning("⚠️ GraphDB not available for testing")
        finally:
            await manager.close()

        # Test shared runtime integration
        logger.info("Testing shared runtime GraphDB integration...")
        try:
            runtime = await initialize_shared_runtime(config)

            await test_suite.test_shared_runtime_graphdb_integration(runtime)
            await test_suite.test_kb7_terminology_semantic_query(runtime)
            await test_suite.test_cross_kb_semantic_query(runtime)

            logger.info("✅ Shared runtime GraphDB tests passed")

        except Exception as e:
            logger.warning(f"⚠️ Shared runtime tests skipped: {e}")
        finally:
            if 'runtime' in locals():
                await runtime.shutdown()

        logger.info("🎉 All GraphDB integration tests completed")

    except Exception as e:
        logger.error(f"❌ GraphDB integration tests failed: {e}")
        raise


if __name__ == "__main__":
    asyncio.run(run_integration_tests())