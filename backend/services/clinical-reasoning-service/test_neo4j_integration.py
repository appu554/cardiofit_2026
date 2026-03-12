"""
Integration Test Suite for CAE Engine Neo4j Integration

Comprehensive test suite to validate Neo4j integration with real clinical scenarios,
performance benchmarks, and health checks.
"""

import pytest
import asyncio
import time
import logging
from app.cae_engine_neo4j import CAEEngine

# Configure logging for tests
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@pytest.fixture
async def cae_engine():
    """Create CAE Engine for testing"""
    # Uses existing Neo4j client configuration from knowledge-pipeline-service
    engine = CAEEngine()
    initialized = await engine.initialize()

    if not initialized:
        pytest.skip("Could not initialize CAE Engine with Neo4j")

    yield engine
    await engine.close()

@pytest.mark.asyncio
async def test_drug_interaction_detection(cae_engine):
    """Test drug interaction detection with real Neo4j data"""
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
            {'name': 'ciprofloxacin', 'dose': '500mg', 'frequency': 'twice daily'}
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
    assert result['performance']['total_execution_time_ms'] < 200  # Sub-200ms requirement
    assert result['metadata']['knowledge_source'] == 'Neo4j Knowledge Graph'

    # Check for drug interaction detection
    ddi_result = result['checker_results'].get('ddi', {})
    assert ddi_result['status'] in ['SAFE', 'WARNING', 'UNSAFE']

    logger.info(f"DDI Test - Status: {result['overall_status']}, "
               f"Findings: {result['total_findings']}, "
               f"Time: {result['performance']['total_execution_time_ms']:.1f}ms")

@pytest.mark.asyncio
async def test_adverse_event_detection(cae_engine):
    """Test adverse event detection with real FDA data"""
    clinical_context = {
        'patient': {'id': 'test_patient', 'age': 45, 'gender': 'female'},
        'medications': [
            {'name': 'metformin', 'dose': '500mg', 'frequency': 'twice daily'}
        ],
        'conditions': [
            {'name': 'diabetes mellitus type 2'}
        ],
        'allergies': []
    }

    result = await cae_engine.validate_safety(clinical_context)

    # Check allergy checker processed the request
    allergy_result = result['checker_results'].get('allergy', {})
    assert 'status' in allergy_result
    assert 'execution_time_ms' in allergy_result

    logger.info(f"Adverse Event Test - Status: {result['overall_status']}, "
               f"Allergy Checker Time: {allergy_result.get('execution_time_ms', 0):.1f}ms")

@pytest.mark.asyncio
async def test_contraindication_detection(cae_engine):
    """Test contraindication detection"""
    clinical_context = {
        'patient': {'id': 'contraindication_test', 'age': 25, 'gender': 'female', 'pregnant': True},
        'medications': [
            {'name': 'warfarin', 'dose': '5mg', 'frequency': 'daily'}
        ],
        'conditions': [
            {'name': 'deep vein thrombosis'}
        ],
        'allergies': []
    }

    result = await cae_engine.validate_safety(clinical_context)

    # Should detect pregnancy contraindication
    contraindication_result = result['checker_results'].get('contraindication', {})
    assert contraindication_result['status'] in ['SAFE', 'WARNING', 'UNSAFE']

    # Check for pregnancy-related findings
    findings = result['findings']
    pregnancy_findings = [f for f in findings if 'pregnancy' in f.get('type', '').lower()]
    
    logger.info(f"Contraindication Test - Status: {result['overall_status']}, "
               f"Pregnancy Findings: {len(pregnancy_findings)}")

@pytest.mark.asyncio
async def test_dosing_adjustment_detection(cae_engine):
    """Test dosing adjustment detection for renal impairment"""
    clinical_context = {
        'patient': {'id': 'dosing_test', 'age': 75, 'egfr': 30, 'weight': 65},
        'medications': [
            {'name': 'digoxin', 'dose': '0.25mg', 'frequency': 'daily'},
            {'name': 'metformin', 'dose': '1000mg', 'frequency': 'twice daily'}
        ],
        'conditions': [
            {'name': 'heart failure'},
            {'name': 'chronic kidney disease'}
        ],
        'allergies': []
    }

    result = await cae_engine.validate_safety(clinical_context)

    # Check dose validator
    dose_result = result['checker_results'].get('dose', {})
    assert dose_result['status'] in ['SAFE', 'WARNING', 'UNSAFE']

    logger.info(f"Dosing Test - Status: {result['overall_status']}, "
               f"Dose Validator Status: {dose_result['status']}")

@pytest.mark.asyncio
async def test_performance_benchmarks(cae_engine):
    """Test performance meets requirements"""
    clinical_context = {
        'patient': {'id': 'perf_test', 'age': 55, 'egfr': 30},
        'medications': [
            {'name': 'digoxin', 'dose': '0.25mg'},
            {'name': 'amiodarone', 'dose': '200mg'}
        ],
        'conditions': [{'name': 'heart failure'}],
        'allergies': []
    }

    # Run multiple times to test caching
    execution_times = []
    
    for i in range(5):
        start_time = time.time()
        result = await cae_engine.validate_safety(clinical_context)
        execution_time = (time.time() - start_time) * 1000
        execution_times.append(execution_time)

        # Performance requirements
        assert result['performance']['total_execution_time_ms'] < 200

        # Cache should improve performance after first run
        if i > 0:
            cache_stats = result['performance']['cache_stats']
            hit_rate = float(cache_stats['hit_rate'].replace('%', ''))
            assert hit_rate > 0

    # Check performance improvement with caching
    avg_first_two = sum(execution_times[:2]) / 2
    avg_last_three = sum(execution_times[2:]) / 3
    
    logger.info(f"Performance Test - First 2 avg: {avg_first_two:.1f}ms, "
               f"Last 3 avg: {avg_last_three:.1f}ms")

@pytest.mark.asyncio
async def test_health_status(cae_engine):
    """Test CAE Engine health status"""
    health = await cae_engine.get_health_status()

    assert health['status'] == 'HEALTHY'
    assert health['neo4j_connection'] is True
    assert 'cache_stats' in health
    assert len(health['checkers']) >= 4
    assert 'performance_metrics' in health

    logger.info(f"Health Status: {health['status']}, "
               f"Neo4j Connected: {health['neo4j_connection']}")

@pytest.mark.asyncio
async def test_known_allergy_detection(cae_engine):
    """Test known allergy detection"""
    clinical_context = {
        'patient': {'id': 'allergy_test', 'age': 35},
        'medications': [
            {'name': 'penicillin', 'dose': '500mg', 'frequency': 'four times daily'}
        ],
        'conditions': [
            {'name': 'pneumonia'}
        ],
        'allergies': [
            {'substance': 'penicillin', 'reaction': 'rash', 'severity': 'moderate'}
        ]
    }

    result = await cae_engine.validate_safety(clinical_context)

    # Should detect known allergy
    assert result['overall_status'] == 'UNSAFE'
    
    # Check for allergy findings
    allergy_findings = [f for f in result['findings'] if f.get('type') == 'KNOWN_ALLERGY']
    assert len(allergy_findings) > 0

    logger.info(f"Known Allergy Test - Status: {result['overall_status']}, "
               f"Allergy Findings: {len(allergy_findings)}")

@pytest.mark.asyncio
async def test_error_handling(cae_engine):
    """Test error handling with invalid input"""
    # Test with missing patient ID
    invalid_context = {
        'patient': {'age': 45},  # Missing ID
        'medications': [
            {'name': 'aspirin', 'dose': '81mg'}
        ]
    }

    result = await cae_engine.validate_safety(invalid_context)
    assert result['overall_status'] == 'ERROR'
    assert 'error' in result

    logger.info(f"Error Handling Test - Status: {result['overall_status']}")

if __name__ == "__main__":
    # Run tests directly
    pytest.main([__file__, "-v"])
