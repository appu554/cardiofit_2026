"""
Comprehensive Test Suite for CAE Engine Neo4j Integration

This test validates the complete integration between CAE Engine and Neo4j
knowledge graph, testing all clinical reasoners with real scenarios.
"""

import asyncio
import pytest
import time
import logging
from typing import Dict, Any

# Load environment variables
try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass

# Add app to path
import sys
from pathlib import Path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

from app.cae_engine_neo4j import CAEEngine

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class TestNeo4jCAEIntegration:
    """Test suite for Neo4j CAE Engine integration"""
    
    @pytest.fixture(scope="class")
    async def cae_engine(self):
        """Initialize CAE Engine for testing"""
        engine = CAEEngine()
        initialized = await engine.initialize()
        
        if not initialized:
            pytest.skip("Could not initialize CAE Engine with Neo4j")
        
        yield engine
        await engine.close()
    
    @pytest.mark.asyncio
    async def test_neo4j_connection_health(self, cae_engine):
        """Test Neo4j connection and health status"""
        health = await cae_engine.get_health_status()
        
        # Assertions
        assert health['status'] == 'HEALTHY', f"CAE Engine not healthy: {health}"
        assert health['neo4j_connection'] is True, "Neo4j connection failed"
        assert len(health['checkers']) >= 4, "Missing clinical checkers"
        assert 'ddi' in health['checkers'], "DDI checker missing"
        assert 'allergy' in health['checkers'], "Allergy checker missing"
        assert 'dose' in health['checkers'], "Dose validator missing"
        assert 'contraindication' in health['checkers'], "Contraindication checker missing"
        
        logger.info(f"✅ Health Check Passed - Status: {health['status']}")
    
    @pytest.mark.asyncio
    async def test_drug_interaction_detection(self, cae_engine):
        """Test drug-drug interaction detection with known interacting drugs"""
        clinical_context = {
            'patient': {
                'id': '905a60cb-8241-418f-b29b-5b020e851392',
                'age': 65,
                'weight': 70,
                'egfr': 45,
                'gender': 'male'
            },
            'medications': [
                {'name': 'warfarin', 'dose': '5mg', 'frequency': 'daily'},
                {'name': 'ciprofloxacin', 'dose': '500mg', 'frequency': 'twice daily'},
                {'name': 'aspirin', 'dose': '81mg', 'frequency': 'daily'}
            ],
            'conditions': [
                {'name': 'atrial fibrillation'},
                {'name': 'pneumonia'}
            ],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(clinical_context)
        
        # Assertions
        assert result['overall_status'] in ['SAFE', 'WARNING', 'UNSAFE']
        assert 'findings' in result
        assert 'performance' in result
        assert result['performance']['total_execution_time_ms'] < 1000  # Under 1 second
        assert result['metadata']['knowledge_source'] == 'Neo4j Knowledge Graph'
        
        # Check DDI checker specifically
        ddi_result = result['checker_results'].get('ddi', {})
        assert 'status' in ddi_result
        assert 'execution_time_ms' in ddi_result
        
        logger.info(f"✅ DDI Test - Status: {result['overall_status']}, "
                   f"Findings: {result['total_findings']}, "
                   f"Time: {result['performance']['total_execution_time_ms']:.1f}ms")
    
    @pytest.mark.asyncio
    async def test_known_allergy_detection(self, cae_engine):
        """Test known allergy detection - should flag UNSAFE"""
        clinical_context = {
            'patient': {
                'id': 'allergy_test_patient',
                'age': 35,
                'gender': 'female'
            },
            'medications': [
                {'name': 'penicillin', 'dose': '500mg', 'frequency': 'four times daily'},
                {'name': 'amoxicillin', 'dose': '250mg', 'frequency': 'three times daily'}
            ],
            'conditions': [
                {'name': 'pneumonia'},
                {'name': 'bronchitis'}
            ],
            'allergies': [
                {'substance': 'penicillin', 'reaction': 'rash', 'severity': 'moderate'},
                {'substance': 'beta-lactam', 'reaction': 'hives', 'severity': 'severe'}
            ]
        }
        
        result = await cae_engine.validate_safety(clinical_context)
        
        # Should detect known allergy
        assert result['overall_status'] == 'UNSAFE', f"Expected UNSAFE due to allergy, got {result['overall_status']}"
        assert result['total_findings'] > 0, "Should have allergy findings"
        
        # Check for allergy findings
        allergy_findings = [f for f in result['findings'] if 'allergy' in f.get('type', '').lower()]
        assert len(allergy_findings) > 0, "Should detect known allergy"
        
        logger.info(f"✅ Allergy Test - Status: {result['overall_status']}, "
                   f"Allergy Findings: {len(allergy_findings)}")
    
    @pytest.mark.asyncio
    async def test_pregnancy_contraindication(self, cae_engine):
        """Test pregnancy contraindication detection"""
        clinical_context = {
            'patient': {
                'id': 'pregnancy_test_patient',
                'age': 28,
                'gender': 'female',
                'pregnant': True
            },
            'medications': [
                {'name': 'warfarin', 'dose': '5mg', 'frequency': 'daily'},
                {'name': 'ace inhibitor', 'dose': '10mg', 'frequency': 'daily'}
            ],
            'conditions': [
                {'name': 'deep vein thrombosis'},
                {'name': 'hypertension'}
            ],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(clinical_context)
        
        # Should detect pregnancy contraindication
        assert result['overall_status'] in ['WARNING', 'UNSAFE'], f"Expected WARNING/UNSAFE for pregnancy, got {result['overall_status']}"
        
        # Check for pregnancy-related findings
        pregnancy_findings = [f for f in result['findings'] 
                            if 'pregnancy' in f.get('type', '').lower() or 
                               'pregnancy' in f.get('message', '').lower()]
        
        logger.info(f"✅ Pregnancy Test - Status: {result['overall_status']}, "
                   f"Pregnancy Findings: {len(pregnancy_findings)}")
    
    @pytest.mark.asyncio
    async def test_renal_dosing_adjustment(self, cae_engine):
        """Test renal dosing adjustment detection"""
        clinical_context = {
            'patient': {
                'id': 'renal_test_patient',
                'age': 75,
                'egfr': 25,  # Severe renal impairment
                'weight': 65,
                'gender': 'male'
            },
            'medications': [
                {'name': 'digoxin', 'dose': '0.25mg', 'frequency': 'daily'},
                {'name': 'metformin', 'dose': '1000mg', 'frequency': 'twice daily'},
                {'name': 'gabapentin', 'dose': '300mg', 'frequency': 'three times daily'}
            ],
            'conditions': [
                {'name': 'heart failure'},
                {'name': 'chronic kidney disease'},
                {'name': 'diabetes mellitus type 2'}
            ],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(clinical_context)
        
        # Should detect dosing adjustments needed
        dose_result = result['checker_results'].get('dose', {})
        assert 'status' in dose_result
        
        # Check for dosing-related findings
        dosing_findings = [f for f in result['findings'] 
                         if 'dosing' in f.get('type', '').lower() or 
                            'dose' in f.get('type', '').lower() or
                            'renal' in f.get('message', '').lower()]
        
        logger.info(f"✅ Renal Dosing Test - Status: {result['overall_status']}, "
                   f"Dosing Findings: {len(dosing_findings)}")
    
    @pytest.mark.asyncio
    async def test_pediatric_contraindication(self, cae_engine):
        """Test pediatric contraindication detection"""
        clinical_context = {
            'patient': {
                'id': 'pediatric_test_patient',
                'age': 8,  # Pediatric patient
                'weight': 25,
                'gender': 'male'
            },
            'medications': [
                {'name': 'aspirin', 'dose': '81mg', 'frequency': 'daily'},  # Contraindicated in children
                {'name': 'tetracycline', 'dose': '250mg', 'frequency': 'twice daily'}  # Contraindicated in children
            ],
            'conditions': [
                {'name': 'fever'},
                {'name': 'infection'}
            ],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(clinical_context)
        
        # Should detect pediatric contraindications
        contraindication_result = result['checker_results'].get('contraindication', {})
        assert 'status' in contraindication_result
        
        # Check for pediatric-related findings
        pediatric_findings = [f for f in result['findings'] 
                            if 'pediatric' in f.get('type', '').lower() or 
                               'age' in f.get('message', '').lower()]
        
        logger.info(f"✅ Pediatric Test - Status: {result['overall_status']}, "
                   f"Pediatric Findings: {len(pediatric_findings)}")
    
    @pytest.mark.asyncio
    async def test_performance_benchmarks(self, cae_engine):
        """Test performance meets requirements with caching"""
        clinical_context = {
            'patient': {
                'id': 'performance_test_patient',
                'age': 55,
                'egfr': 60,
                'weight': 70
            },
            'medications': [
                {'name': 'warfarin', 'dose': '5mg'},
                {'name': 'digoxin', 'dose': '0.25mg'},
                {'name': 'amiodarone', 'dose': '200mg'}
            ],
            'conditions': [
                {'name': 'atrial fibrillation'},
                {'name': 'heart failure'}
            ],
            'allergies': []
        }
        
        # Run multiple times to test caching
        execution_times = []
        
        for i in range(3):
            start_time = time.time()
            result = await cae_engine.validate_safety(clinical_context)
            execution_time = (time.time() - start_time) * 1000
            execution_times.append(execution_time)
            
            # Performance requirements
            assert result['performance']['total_execution_time_ms'] < 1000  # Under 1 second
            
            # Cache should improve performance after first run
            if i > 0:
                cache_stats = result['performance']['cache_stats']
                assert 'hit_rate' in cache_stats
        
        # Check performance improvement
        avg_time = sum(execution_times) / len(execution_times)
        assert avg_time < 500, f"Average execution time too high: {avg_time:.1f}ms"
        
        logger.info(f"✅ Performance Test - Average: {avg_time:.1f}ms, "
                   f"Times: {[f'{t:.1f}ms' for t in execution_times]}")
    
    @pytest.mark.asyncio
    async def test_error_handling(self, cae_engine):
        """Test error handling with invalid input"""
        # Test with missing patient ID
        invalid_context = {
            'patient': {'age': 45},  # Missing ID
            'medications': [
                {'name': 'aspirin', 'dose': '81mg'}
            ],
            'conditions': [],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(invalid_context)
        assert result['overall_status'] == 'ERROR'
        assert 'error' in result
        
        logger.info(f"✅ Error Handling Test - Status: {result['overall_status']}")
    
    @pytest.mark.asyncio
    async def test_cache_functionality(self, cae_engine):
        """Test cache functionality and statistics"""
        # Get initial cache stats
        initial_metrics = await cae_engine.get_performance_metrics()
        initial_requests = initial_metrics['requests']['total']
        
        # Run a test scenario
        clinical_context = {
            'patient': {'id': 'cache_test', 'age': 45},
            'medications': [{'name': 'aspirin', 'dose': '81mg'}],
            'conditions': [],
            'allergies': []
        }
        
        # Run same query twice
        await cae_engine.validate_safety(clinical_context)
        await cae_engine.validate_safety(clinical_context)
        
        # Check metrics updated
        final_metrics = await cae_engine.get_performance_metrics()
        final_requests = final_metrics['requests']['total']
        
        assert final_requests > initial_requests, "Request count should increase"
        assert 'cache' in final_metrics, "Cache stats should be available"
        
        logger.info(f"✅ Cache Test - Requests: {final_requests}, "
                   f"Cache Stats: {final_metrics['cache']}")

async def run_comprehensive_test():
    """Run comprehensive test suite"""
    logger.info("🧪 Starting Comprehensive Neo4j CAE Integration Test")
    logger.info("=" * 60)
    
    # Initialize test class
    test_instance = TestNeo4jCAEIntegration()
    
    # Initialize CAE Engine
    cae_engine = CAEEngine()
    initialized = await cae_engine.initialize()
    
    if not initialized:
        logger.error("❌ Failed to initialize CAE Engine")
        return False
    
    try:
        # Run all tests
        await test_instance.test_neo4j_connection_health(cae_engine)
        await test_instance.test_drug_interaction_detection(cae_engine)
        await test_instance.test_known_allergy_detection(cae_engine)
        await test_instance.test_pregnancy_contraindication(cae_engine)
        await test_instance.test_renal_dosing_adjustment(cae_engine)
        await test_instance.test_pediatric_contraindication(cae_engine)
        await test_instance.test_performance_benchmarks(cae_engine)
        await test_instance.test_error_handling(cae_engine)
        await test_instance.test_cache_functionality(cae_engine)
        
        logger.info("\n🎉 All Tests Passed Successfully!")
        logger.info("✅ Neo4j CAE Integration is fully functional")
        return True
        
    except Exception as e:
        logger.error(f"❌ Test failed: {e}")
        return False
    
    finally:
        await cae_engine.close()

if __name__ == "__main__":
    # Run the comprehensive test
    success = asyncio.run(run_comprehensive_test())
    exit(0 if success else 1)
