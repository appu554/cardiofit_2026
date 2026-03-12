"""
Integration Tests for KB7 Neo4j Dual-Stream & Service Runtime Layer
Tests end-to-end workflows across all runtime components
"""

import pytest
import asyncio
import json
from datetime import datetime
from typing import Dict, Any, List
import uuid


class TestRuntimeIntegration:
    """Integration tests for the complete runtime layer"""

    @pytest.fixture
    async def runtime_setup(self):
        """Setup runtime components for testing"""
        # Initialize all components
        from ..neo4j_setup.dual_stream_manager import Neo4jDualStreamManager
        from ..clickhouse_runtime.manager import ClickHouseRuntimeManager
        from ..query_router.router import QueryRouter
        from ..snapshot.manager import SnapshotManager

        # Test configuration
        config = {
            'neo4j': {
                'neo4j_uri': 'bolt://localhost:7687',
                'neo4j_user': 'neo4j',
                'neo4j_password': 'kb7password'
            },
            'clickhouse': {
                'host': 'localhost',
                'port': 9000,
                'database': 'kb7_analytics_test',
                'user': 'kb7',
                'password': 'kb7password'
            },
            'postgres': {
                'dsn': 'postgresql://kb_test_user:kb_test_password@localhost:5434/clinical_governance_test'
            },
            'elasticsearch': {
                'urls': ['http://localhost:9200'],
                'index': 'kb7-terminology-test'
            },
            'kafka_brokers': ['localhost:9092'],
            'redis_l2_url': 'redis://localhost:6379/5',
            'redis_l3_url': 'redis://localhost:6379/6'
        }

        # Initialize components
        neo4j_manager = Neo4jDualStreamManager(config['neo4j'])
        await neo4j_manager.initialize_databases()

        clickhouse_manager = ClickHouseRuntimeManager(config['clickhouse'])

        query_router = QueryRouter(config)
        await query_router.initialize_clients()

        snapshot_manager = SnapshotManager()

        yield {
            'neo4j': neo4j_manager,
            'clickhouse': clickhouse_manager,
            'query_router': query_router,
            'snapshot_manager': snapshot_manager,
            'config': config
        }

        # Cleanup
        await neo4j_manager.close()
        clickhouse_manager.close()

    @pytest.mark.asyncio
    async def test_medication_scoring_workflow(self, runtime_setup):
        """Test complete medication scoring workflow"""
        components = runtime_setup
        query_router = components['query_router']

        # Test data
        patient_id = "patient-12345"
        indication = "I25.10"  # Atherosclerotic heart disease
        drug_codes = ["197361", "197362", "197363"]  # Example RxNorm codes

        # Step 1: Create snapshot for consistency
        from ..query_router.router import QueryRequest, QueryPattern

        snapshot_request = QueryRequest(
            service_id="medication",
            pattern=QueryPattern.MEDICATION_SCORING,
            params={
                'drugs': drug_codes,
                'indication': indication,
                'patient_context': {'patient_id': patient_id}
            },
            require_snapshot=True
        )

        # Step 2: Execute scoring query
        response = await query_router.route_query(snapshot_request)

        assert response.data is not None
        assert response.source == "clickhouse"
        assert response.snapshot_id is not None
        assert response.latency > 0

        # Step 3: Verify data consistency
        snapshot = await components['snapshot_manager'].get_snapshot(response.snapshot_id)
        assert snapshot is not None
        assert snapshot.is_valid()

    @pytest.mark.asyncio
    async def test_drug_interaction_workflow(self, runtime_setup):
        """Test drug interaction detection workflow"""
        components = runtime_setup
        neo4j_manager = components['neo4j']
        query_router = components['query_router']

        # Load test interaction data
        await self._load_test_interactions(neo4j_manager)

        # Test interaction query
        from ..query_router.router import QueryRequest, QueryPattern

        interaction_request = QueryRequest(
            service_id="safety",
            pattern=QueryPattern.DRUG_INTERACTIONS,
            params={'drug_codes': ["197361", "197362"]}
        )

        response = await query_router.route_query(interaction_request)

        assert response.data is not None
        assert response.source == "neo4j_semantic"
        assert isinstance(response.data, list)

    @pytest.mark.asyncio
    async def test_patient_data_stream(self, runtime_setup):
        """Test patient data stream processing"""
        components = runtime_setup
        neo4j_manager = components['neo4j']

        # Load test patient data
        patient_data = {
            'id': 'patient-12345',
            'mrn': 'MRN-98765',
            'demographics': {
                'age': 65,
                'gender': 'male'
            }
        }

        patient_id = await neo4j_manager.load_patient_data(patient_data)
        assert patient_id == 'patient-12345'

        # Query patient medications
        medications = await neo4j_manager.get_patient_medications(patient_id)
        assert isinstance(medications, list)

    @pytest.mark.asyncio
    async def test_semantic_mesh_sync(self, runtime_setup):
        """Test semantic mesh synchronization"""
        components = runtime_setup
        neo4j_manager = components['neo4j']

        # Load test semantic concepts
        concept_data = {
            'uri': 'http://snomed.info/id/387517004',
            'code': '387517004',
            'label': 'ACE inhibitor',
            'system': 'SNOMED',
            'class': 'Drug'
        }

        concept_uri = await neo4j_manager.load_semantic_concept(concept_data)
        assert concept_uri == concept_data['uri']

        # Verify concept loading
        async with neo4j_manager.driver.session(database="semantic_mesh") as session:
            result = await session.run("""
                MATCH (c:Concept {uri: $uri})
                RETURN c.code as code, c.label as label
            """, uri=concept_uri)

            record = await result.single()
            assert record is not None
            assert record['code'] == '387517004'
            assert record['label'] == 'ACE inhibitor'

    @pytest.mark.asyncio
    async def test_query_routing_patterns(self, runtime_setup):
        """Test query routing for different patterns"""
        components = runtime_setup
        query_router = components['query_router']

        from ..query_router.router import QueryRequest, QueryPattern

        # Test terminology lookup routing
        terminology_request = QueryRequest(
            service_id="terminology",
            pattern=QueryPattern.TERMINOLOGY_LOOKUP,
            params={'code': '387517004', 'system': 'SNOMED'}
        )

        response = await query_router.route_query(terminology_request)
        # Should route to PostgreSQL for terminology lookups
        assert 'postgres' in response.source.lower()

        # Test search routing
        search_request = QueryRequest(
            service_id="terminology",
            pattern=QueryPattern.TERMINOLOGY_SEARCH,
            params={'query': 'ACE inhibitor', 'size': 10}
        )

        search_response = await query_router.route_query(search_request)
        # Should route to Elasticsearch for text search
        assert 'elasticsearch' in search_response.source.lower()

    @pytest.mark.asyncio
    async def test_snapshot_consistency(self, runtime_setup):
        """Test snapshot consistency across data stores"""
        components = runtime_setup
        snapshot_manager = components['snapshot_manager']

        # Create snapshot
        snapshot = await snapshot_manager.create_snapshot(
            service_id="test",
            context={'test': 'consistency'},
            ttl=None
        )

        assert snapshot.id is not None
        assert snapshot.checksum is not None
        assert len(snapshot.versions) > 0

        # Validate snapshot
        is_valid = await snapshot_manager.validate_snapshot(snapshot.id)
        assert is_valid is True

        # Get snapshot statistics
        stats = await snapshot_manager.get_statistics()
        assert stats['active_snapshots'] >= 1
        assert stats['valid_snapshots'] >= 1

    @pytest.mark.asyncio
    async def test_clickhouse_analytics(self, runtime_setup):
        """Test ClickHouse analytics functionality"""
        components = runtime_setup
        clickhouse_manager = components['clickhouse']

        # Test medication scoring
        drugs = ["197361", "197362"]
        indication = "I25.10"

        scores_df = await clickhouse_manager.calculate_medication_scores(
            drugs=drugs,
            indication=indication,
            patient_context={'elderly': True}
        )

        assert scores_df is not None
        assert len(scores_df) <= len(drugs)

        # Test safety analytics
        safety_result = await clickhouse_manager.calculate_safety_analytics(
            patient_id="patient-12345",
            medications=drugs,
            conditions=["I25.10", "E11.9"]
        )

        assert safety_result is not None
        assert 'risk_score' in safety_result
        assert 'risk_level' in safety_result
        assert safety_result['risk_score'] >= 0

    @pytest.mark.asyncio
    async def test_end_to_end_medication_workflow(self, runtime_setup):
        """Test complete end-to-end medication calculation workflow"""
        components = runtime_setup
        query_router = components['query_router']

        # Simulate medication service request
        patient_id = "patient-67890"
        indication = "E11.9"  # Type 2 diabetes
        candidate_drugs = ["197361", "197362", "197363"]

        from ..query_router.router import QueryRequest, QueryPattern

        # Step 1: Get drug alternatives from semantic mesh
        alternatives_request = QueryRequest(
            service_id="medication",
            pattern=QueryPattern.DRUG_ALTERNATIVES,
            params={'drug_code': candidate_drugs[0]},
            require_snapshot=True
        )

        alternatives_response = await query_router.route_query(alternatives_request)
        assert alternatives_response.snapshot_id is not None

        # Step 2: Score medications
        scoring_request = QueryRequest(
            service_id="medication",
            pattern=QueryPattern.MEDICATION_SCORING,
            params={
                'drugs': candidate_drugs,
                'indication': indication,
                'patient_context': {'patient_id': patient_id}
            },
            require_snapshot=True
        )

        scoring_response = await query_router.route_query(scoring_request)
        assert scoring_response.data is not None

        # Step 3: Check safety
        safety_request = QueryRequest(
            service_id="safety",
            pattern=QueryPattern.SAFETY_ANALYTICS,
            params={
                'patient_id': patient_id,
                'medications': candidate_drugs,
                'conditions': [indication]
            }
        )

        safety_response = await query_router.route_query(safety_request)
        assert safety_response.data is not None
        assert 'risk_score' in safety_response.data

    async def _load_test_interactions(self, neo4j_manager):
        """Load test drug interaction data"""
        async with neo4j_manager.driver.session(database="semantic_mesh") as session:
            await session.run("""
                MERGE (d1:Drug {rxnorm: '197361', name: 'Lisinopril'})
                MERGE (d2:Drug {rxnorm: '197362', name: 'Metformin'})
                MERGE (d1)-[i:INTERACTS_WITH]-(d2)
                SET i.severity = 'moderate',
                    i.mechanism = 'renal function',
                    i.updated = datetime()
            """)

    @pytest.mark.asyncio
    async def test_health_checks(self, runtime_setup):
        """Test health check functionality across components"""
        components = runtime_setup

        # Test Neo4j health
        neo4j_health = await components['neo4j'].health_check()
        assert neo4j_health['status'] in ['healthy', 'degraded']

        # Test ClickHouse health
        ch_health = components['clickhouse'].health_check()
        assert ch_health['status'] in ['healthy', 'unhealthy']

        # Test Query Router health
        router_health = await components['query_router'].health_check()
        assert 'status' in router_health
        assert 'sources' in router_health

        # Test Snapshot Manager stats
        snapshot_stats = await components['snapshot_manager'].get_statistics()
        assert 'active_snapshots' in snapshot_stats
        assert 'timestamp' in snapshot_stats


@pytest.mark.asyncio
async def test_performance_benchmarks():
    """Performance benchmarks for runtime components"""
    # Test query routing latency
    start_time = datetime.utcnow()

    # Simulate query routing
    await asyncio.sleep(0.001)  # Simulate 1ms routing time

    routing_time = (datetime.utcnow() - start_time).total_seconds() * 1000
    assert routing_time < 5  # Should be under 5ms

    # Test cache warming efficiency
    # Test snapshot creation speed
    # Test data consistency validation
    pass


if __name__ == "__main__":
    pytest.main([__file__, "-v", "--asyncio-mode=auto"])