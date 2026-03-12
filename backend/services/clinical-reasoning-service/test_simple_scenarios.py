#!/usr/bin/env python3
"""
Simple Clinical Scenario Testing for CAE

Start with basic testing to ensure everything works before comprehensive testing.
"""

import asyncio
import logging
import sys
import traceback

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def test_basic_imports():
    """Test basic imports"""
    logger.info("🔍 Testing Basic Imports")
    
    try:
        from app.learning.learning_manager import learning_manager
        logger.info("✅ Learning manager imported")
        
        from app.graph.graphdb_client import graphdb_client
        logger.info("✅ GraphDB client imported")
        
        from app.reasoners.medication_interaction import medication_interaction_reasoner
        logger.info("✅ Medication interaction reasoner imported")
        
        return True
    except Exception as e:
        logger.error(f"❌ Import failed: {e}")
        traceback.print_exc()
        return False

async def test_simple_patient_scenario():
    """Test simple patient scenario"""
    logger.info("🏥 Testing Simple Patient Scenario")
    
    try:
        # Test patient data
        test_patient = {
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "name": "Test Cardiovascular Patient",
            "medications": ["warfarin", "aspirin"],
            "age": 67,
            "weight": 78.5
        }
        
        logger.info(f"📋 Testing patient: {test_patient['name']}")
        logger.info(f"💊 Medications: {test_patient['medications']}")
        
        # Import reasoner
        from app.reasoners.medication_interaction import medication_interaction_reasoner
        
        # Test medication interactions
        interactions = await medication_interaction_reasoner.check_interactions(
            patient_id=test_patient["patient_id"],
            medication_ids=test_patient["medications"],
            patient_context={
                "age": test_patient["age"],
                "weight": test_patient["weight"]
            }
        )
        
        logger.info(f"✅ Found {len(interactions)} interactions")
        for interaction in interactions[:3]:  # Show first 3
            logger.info(f"   ⚠️  {interaction.medication_a} + {interaction.medication_b}: {interaction.severity.value}")
        
        return len(interactions) > 0
        
    except Exception as e:
        logger.error(f"❌ Patient scenario test failed: {e}")
        traceback.print_exc()
        return False

async def test_simple_learning():
    """Test simple learning functionality"""
    logger.info("🧠 Testing Simple Learning")
    
    try:
        from app.learning.learning_manager import learning_manager
        
        # Initialize learning manager
        await learning_manager.initialize()
        logger.info("✅ Learning manager initialized")
        
        # Test simple outcome tracking
        outcome_success = await learning_manager.track_clinical_outcome(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            assertion_id="simple_test_001",
            outcome_type="bleeding_event",
            severity=2,
            description="Simple test bleeding event",
            related_medications=["warfarin", "aspirin"],
            clinician_id="test_clinician"
        )
        
        logger.info(f"✅ Outcome tracking: {'Success' if outcome_success else 'Failed'}")
        
        return outcome_success
        
    except Exception as e:
        logger.error(f"❌ Learning test failed: {e}")
        traceback.print_exc()
        return False

async def test_simple_population():
    """Test simple population intelligence"""
    logger.info("👥 Testing Simple Population Intelligence")
    
    try:
        from app.graph.population_clustering import population_clustering_engine
        
        # Test patient clustering
        clusters = await population_clustering_engine.identify_patient_clusters()
        logger.info(f"✅ Found {len(clusters)} patient clusters")
        
        # Test similar patients
        similar_patients = await population_clustering_engine.find_similar_patients(
            "905a60cb-8241-418f-b29b-5b020e851392", similarity_threshold=0.7
        )
        logger.info(f"✅ Found {len(similar_patients)} similar patients")
        
        return len(clusters) > 0
        
    except Exception as e:
        logger.error(f"❌ Population test failed: {e}")
        traceback.print_exc()
        return False

async def main():
    """Run simple testing"""
    logger.info("🚀 Starting Simple CAE Testing")
    logger.info("=" * 60)
    
    tests = [
        ("Basic Imports", test_basic_imports),
        ("Simple Patient Scenario", test_simple_patient_scenario),
        ("Simple Learning", test_simple_learning),
        ("Simple Population", test_simple_population)
    ]
    
    results = {}
    
    for test_name, test_func in tests:
        logger.info(f"\n🔍 Running: {test_name}")
        try:
            result = await test_func()
            results[test_name] = result
            logger.info(f"{'✅' if result else '❌'} {test_name}: {'PASS' if result else 'FAIL'}")
        except Exception as e:
            logger.error(f"❌ {test_name}: ERROR - {e}")
            results[test_name] = False
    
    # Summary
    logger.info("\n" + "=" * 60)
    logger.info("📊 SIMPLE TESTING SUMMARY")
    logger.info("=" * 60)
    
    passed = sum(1 for r in results.values() if r)
    total = len(results)
    
    for test_name, result in results.items():
        status = "✅ PASS" if result else "❌ FAIL"
        logger.info(f"{status} - {test_name}")
    
    logger.info(f"\n🎯 Overall: {passed}/{total} tests passed ({passed/total*100:.1f}%)")
    
    if passed == total:
        logger.info("🎉 All simple tests passed! Ready for comprehensive testing.")
    else:
        logger.info("⚠️  Some tests failed. Check the issues above.")
    
    logger.info("=" * 60)

if __name__ == "__main__":
    asyncio.run(main())
