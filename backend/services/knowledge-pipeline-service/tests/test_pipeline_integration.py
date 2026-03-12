"""
Integration Tests for Knowledge Pipeline Service
Tests real data ingestion and GraphDB integration
"""

import pytest
import asyncio
import json
from pathlib import Path
from datetime import datetime

from core.config import settings
from core.graphdb_client import GraphDBClient
from core.pipeline_orchestrator import PipelineOrchestrator
from core.harmonization_engine import HarmonizationEngine
from ingesters.rxnorm_ingester import RxNormIngester
from ingesters.crediblemeds_ingester import CredibleMedsIngester
from ingesters.ahrq_ingester import AHRQIngester


@pytest.fixture
async def graphdb_client():
    """GraphDB client fixture"""
    client = GraphDBClient()
    await client.connect()
    yield client
    await client.disconnect()


@pytest.fixture
async def pipeline_orchestrator(graphdb_client):
    """Pipeline orchestrator fixture"""
    orchestrator = PipelineOrchestrator(graphdb_client)
    await orchestrator.initialize()
    yield orchestrator
    await orchestrator.cleanup()


class TestGraphDBIntegration:
    """Test GraphDB integration"""
    
    @pytest.mark.asyncio
    async def test_graphdb_connection(self, graphdb_client):
        """Test GraphDB connection"""
        result = await graphdb_client.test_connection()
        assert result is True, "GraphDB connection should be successful"
    
    @pytest.mark.asyncio
    async def test_graphdb_query(self, graphdb_client):
        """Test basic GraphDB query"""
        query = "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"
        result = await graphdb_client.query(query)
        
        assert result.success is True, "Query should be successful"
        assert result.data is not None, "Query should return data"
        assert 'results' in result.data, "Query should have results"
    
    @pytest.mark.asyncio
    async def test_rdf_insertion(self, graphdb_client):
        """Test RDF data insertion"""
        test_rdf = """
        @prefix cae: <http://clinical-assertion-engine.org/ontology/> .
        @prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
        
        cae:TestDrug_123 a cae:Drug ;
            cae:hasRxCUI "123" ;
            rdfs:label "Test Drug" .
        """
        
        result = await graphdb_client.insert_rdf(test_rdf)
        assert result.success is True, "RDF insertion should be successful"
        assert result.triples_inserted > 0, "Should insert at least one triple"
        
        # Verify insertion with query
        verify_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        SELECT ?drug WHERE {
            ?drug cae:hasRxCUI "123" .
        }
        """
        
        verify_result = await graphdb_client.query(verify_query)
        assert verify_result.success is True
        bindings = verify_result.data.get('results', {}).get('bindings', [])
        assert len(bindings) > 0, "Should find the inserted drug"


class TestRxNormIngester:
    """Test RxNorm ingester"""
    
    @pytest.mark.asyncio
    async def test_rxnorm_ingester_initialization(self, graphdb_client):
        """Test RxNorm ingester initialization"""
        ingester = RxNormIngester(graphdb_client)
        assert ingester.source_name == "rxnorm"
        assert ingester.download_url is not None
        assert len(ingester.rrf_files) > 0
    
    @pytest.mark.asyncio
    async def test_rxnorm_ontology_prefixes(self, graphdb_client):
        """Test RxNorm ontology prefixes"""
        ingester = RxNormIngester(graphdb_client)
        prefixes = ingester.get_ontology_prefixes()
        
        assert "@prefix cae:" in prefixes
        assert "@prefix rdfs:" in prefixes
        assert "@prefix rxnorm:" in prefixes
    
    @pytest.mark.asyncio
    @pytest.mark.slow
    async def test_rxnorm_download_simulation(self, graphdb_client):
        """Test RxNorm download simulation (without actual download)"""
        ingester = RxNormIngester(graphdb_client)
        
        # Create mock RRF data for testing
        test_rrf_dir = ingester.data_dir / "rrf"
        test_rrf_dir.mkdir(parents=True, exist_ok=True)
        
        # Create mock RXNCONSO.RRF
        mock_rxnconso = test_rrf_dir / "RXNCONSO.RRF"
        with open(mock_rxnconso, 'w') as f:
            f.write("123|ENG|P|L|L|PF|S|Y|N||RXNORM|SCD|12345|aspirin|aspirin|\n")
            f.write("456|ENG|P|L|L|PF|S|Y|N||RXNORM|SCD|67890|ibuprofen|ibuprofen|\n")
        
        # Process concepts
        await ingester._process_concepts(mock_rxnconso)
        
        assert len(ingester.processed_concepts) == 2
        assert "123" in ingester.concept_names
        assert "456" in ingester.concept_names
        assert ingester.concept_names["123"] == "aspirin"
        assert ingester.concept_names["456"] == "ibuprofen"


class TestCredibleMedsIngester:
    """Test CredibleMeds ingester"""
    
    @pytest.mark.asyncio
    async def test_crediblemeds_ingester_initialization(self, graphdb_client):
        """Test CredibleMeds ingester initialization"""
        ingester = CredibleMedsIngester(graphdb_client)
        assert ingester.source_name == "crediblemeds"
        assert len(ingester.categories) > 0
    
    @pytest.mark.asyncio
    async def test_crediblemeds_real_data_required(self, graphdb_client):
        """Test that CredibleMeds requires real data - no fallbacks"""
        ingester = CredibleMedsIngester(graphdb_client)

        # Should fail if no real data is available
        with pytest.raises((ValueError, FileNotFoundError)):
            async for _ in ingester.process_data():
                pass
    
    @pytest.mark.asyncio
    async def test_crediblemeds_drug_name_cleaning(self, graphdb_client):
        """Test drug name cleaning"""
        ingester = CredibleMedsIngester(graphdb_client)
        
        test_cases = [
            ("Amiodarone 200mg tablets", "amiodarone"),
            ("Sotalol HCl injection", "sotalol hcl"),
            ("Azithromycin  250 mg", "azithromycin"),
        ]
        
        for input_name, expected in test_cases:
            cleaned = ingester._clean_drug_name(input_name)
            assert cleaned == expected, f"Expected {expected}, got {cleaned}"


class TestAHRQIngester:
    """Test AHRQ ingester"""
    
    @pytest.mark.asyncio
    async def test_ahrq_ingester_initialization(self, graphdb_client):
        """Test AHRQ ingester initialization"""
        ingester = AHRQIngester(graphdb_client)
        assert ingester.source_name == "ahrq"
        assert len(ingester.artifact_types) > 0
    
    @pytest.mark.asyncio
    async def test_ahrq_real_data_required(self, graphdb_client):
        """Test that AHRQ requires real data - no fallbacks"""
        ingester = AHRQIngester(graphdb_client)

        # Should fail if no real data is available
        with pytest.raises((ValueError, FileNotFoundError)):
            async for _ in ingester.process_data():
                pass
    
    @pytest.mark.asyncio
    async def test_ahrq_step_extraction(self, graphdb_client):
        """Test pathway step extraction from text"""
        ingester = AHRQIngester(graphdb_client)
        
        test_text = """
        1. Measure lactate level
        2. Obtain blood cultures
        3. Administer broad-spectrum antibiotics
        4. Begin rapid administration of crystalloid
        """
        
        steps = ingester._extract_steps_from_text(test_text)
        
        assert len(steps) == 4
        assert steps[0]['title'] == "Measure lactate level"
        assert steps[1]['title'] == "Obtain blood cultures"
        assert steps[0]['sequence'] == 1
        assert steps[1]['sequence'] == 2


class TestHarmonizationEngine:
    """Test harmonization engine"""
    
    @pytest.mark.asyncio
    async def test_harmonization_initialization(self, graphdb_client):
        """Test harmonization engine initialization"""
        engine = HarmonizationEngine(graphdb_client)
        await engine.initialize()
        
        # Should initialize without errors
        assert engine.drug_mappings is not None
        assert engine.drug_name_index is not None
    
    @pytest.mark.asyncio
    async def test_drug_name_normalization(self, graphdb_client):
        """Test drug name normalization"""
        engine = HarmonizationEngine(graphdb_client)
        
        test_cases = [
            ("Aspirin 325mg tablets", "aspirin"),
            ("Ibuprofen 200 mg capsules", "ibuprofen"),
            ("Metformin HCl 500mg", "metformin hcl"),
        ]
        
        for input_name, expected in test_cases:
            normalized = engine._normalize_drug_name(input_name)
            assert expected in normalized.lower(), f"Expected {expected} in {normalized}"
    
    @pytest.mark.asyncio
    async def test_string_similarity(self, graphdb_client):
        """Test string similarity calculation"""
        engine = HarmonizationEngine(graphdb_client)
        
        # Exact match
        assert engine._calculate_string_similarity("aspirin", "aspirin") == 1.0
        
        # Partial match
        similarity = engine._calculate_string_similarity("aspirin", "aspirin tablet")
        assert similarity > 0.5
        
        # No match
        similarity = engine._calculate_string_similarity("aspirin", "completely different")
        assert similarity < 0.5


class TestPipelineOrchestrator:
    """Test pipeline orchestrator"""
    
    @pytest.mark.asyncio
    async def test_orchestrator_initialization(self, pipeline_orchestrator):
        """Test orchestrator initialization"""
        assert len(pipeline_orchestrator.ingesters) == 3
        assert 'rxnorm' in pipeline_orchestrator.ingesters
        assert 'crediblemeds' in pipeline_orchestrator.ingesters
        assert 'ahrq' in pipeline_orchestrator.ingesters
    
    @pytest.mark.asyncio
    async def test_orchestrator_status(self, pipeline_orchestrator):
        """Test orchestrator status"""
        status = await pipeline_orchestrator.get_status()
        
        assert 'pipeline_status' in status
        assert 'graphdb_connected' in status
        assert 'ingesters_available' in status
        assert 'harmonization_stats' in status
    
    @pytest.mark.asyncio
    @pytest.mark.slow
    async def test_single_ingester_run(self, pipeline_orchestrator):
        """Test running a single ingester"""
        # Test with AHRQ ingester (fastest, uses fallback data)
        result = await pipeline_orchestrator.run_single_ingester('ahrq', force_download=False)
        
        assert 'source_name' in result
        assert result['source_name'] == 'ahrq'
        assert 'success' in result


class TestEndToEndPipeline:
    """End-to-end pipeline tests"""
    
    @pytest.mark.asyncio
    @pytest.mark.slow
    async def test_pipeline_validation(self, pipeline_orchestrator):
        """Test pipeline validation"""
        # This test validates the pipeline setup without running full ingestion
        
        # Check GraphDB connection
        graphdb_connected = await pipeline_orchestrator.graphdb_client.test_connection()
        assert graphdb_connected, "GraphDB should be connected"
        
        # Check harmonization engine
        harmony_stats = await pipeline_orchestrator.harmonization_engine.get_harmonization_stats()
        assert isinstance(harmony_stats, dict), "Should return harmonization stats"
        
        # Check ingesters
        for name, ingester in pipeline_orchestrator.ingesters.items():
            status = await ingester.get_status()
            assert 'source_name' in status
            assert status['source_name'] == name
    
    @pytest.mark.asyncio
    @pytest.mark.integration
    async def test_clinical_knowledge_queries(self, graphdb_client):
        """Test clinical knowledge queries after ingestion"""
        # This test assumes some data has been ingested
        
        # Query for drugs
        drug_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        SELECT (COUNT(?drug) as ?drug_count) WHERE {
            ?drug a cae:Drug .
        }
        """
        
        result = await graphdb_client.query(drug_query)
        assert result.success, "Drug query should succeed"
        
        # Query for QT risks
        qt_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        SELECT (COUNT(?risk) as ?risk_count) WHERE {
            ?risk a cae:QTRisk .
        }
        """
        
        result = await graphdb_client.query(qt_query)
        assert result.success, "QT risk query should succeed"
        
        # Query for pathways
        pathway_query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        SELECT (COUNT(?pathway) as ?pathway_count) WHERE {
            ?pathway a cae:Pathway .
        }
        """
        
        result = await graphdb_client.query(pathway_query)
        assert result.success, "Pathway query should succeed"


# Test configuration
pytest_plugins = ['pytest_asyncio']

# Markers for different test types
pytestmark = [
    pytest.mark.asyncio,
]
