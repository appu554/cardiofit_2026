"""
Startup Script for CAE Engine with Neo4j Integration

This script starts the CAE Engine with Neo4j knowledge graph integration,
replacing the mock data with real clinical intelligence.
"""

import asyncio
import logging
import os
import sys
from pathlib import Path

# Load environment variables from .env file
try:
    from dotenv import load_dotenv
    load_dotenv()
    print("✅ Loaded environment variables from .env file")
except ImportError:
    print("⚠️  python-dotenv not installed. Install with: pip install python-dotenv")
    print("⚠️  Trying to use system environment variables...")

# Add the app directory to Python path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

from app.cae_engine_neo4j import CAEEngine

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

async def test_neo4j_integration():
    """Test the Neo4j integration with sample clinical scenarios"""

    logger.info("🚀 Starting CAE Engine with Neo4j Integration")
    logger.info("=" * 60)
    logger.info("Using existing Neo4j client from knowledge-pipeline-service")

    # Initialize CAE Engine (uses existing Neo4j client)
    cae_engine = CAEEngine()
    
    try:
        # Initialize the engine
        logger.info("🔌 Initializing CAE Engine...")
        initialized = await cae_engine.initialize()
        
        if not initialized:
            logger.error("❌ Failed to initialize CAE Engine")
            return False
        
        logger.info("✅ CAE Engine initialized successfully")
        
        # Test health status
        logger.info("\n🏥 Testing Health Status...")
        health = await cae_engine.get_health_status()
        logger.info(f"Status: {health['status']}")
        logger.info(f"Neo4j Connected: {health['neo4j_connection']}")
        logger.info(f"Active Checkers: {health['checkers']}")
        
        if health['status'] != 'HEALTHY':
            logger.error("❌ CAE Engine is not healthy")
            return False
        
        # Test clinical scenarios
        await test_clinical_scenarios(cae_engine)
        
        # Show performance metrics
        logger.info("\n📊 Performance Metrics:")
        metrics = await cae_engine.get_performance_metrics()
        logger.info(f"Total Requests: {metrics['requests']['total']}")
        logger.info(f"Success Rate: {metrics['requests']['success_rate']:.1f}%")
        logger.info(f"Average Execution Time: {metrics['performance']['average_execution_time_ms']:.1f}ms")
        logger.info(f"Cache Hit Rate: {metrics['cache']['hit_rate']}")
        
        logger.info("\n🎉 Neo4j Integration Test Completed Successfully!")
        return True
        
    except Exception as e:
        logger.error(f"❌ Error during testing: {e}")
        return False
    
    finally:
        await cae_engine.close()

async def test_clinical_scenarios(cae_engine):
    """Test various clinical scenarios"""
    
    logger.info("\n🧪 Testing Clinical Scenarios...")
    logger.info("-" * 40)
    
    # Scenario 1: Drug Interaction
    logger.info("1. Testing Drug-Drug Interaction...")
    ddi_context = {
        'patient': {
            'id': '905a60cb-8241-418f-b29b-5b020e851392',
            'age': 65,
            'weight': 70,
            'egfr': 45
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
    
    result = await cae_engine.validate_safety(ddi_context)
    logger.info(f"   Status: {result['overall_status']}")
    logger.info(f"   Findings: {result['total_findings']}")
    logger.info(f"   Execution Time: {result['performance']['total_execution_time_ms']:.1f}ms")
    
    # Scenario 2: Known Allergy
    logger.info("\n2. Testing Known Allergy Detection...")
    allergy_context = {
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
    
    result = await cae_engine.validate_safety(allergy_context)
    logger.info(f"   Status: {result['overall_status']}")
    logger.info(f"   Findings: {result['total_findings']}")
    logger.info(f"   Execution Time: {result['performance']['total_execution_time_ms']:.1f}ms")
    
    # Scenario 3: Pregnancy Contraindication
    logger.info("\n3. Testing Pregnancy Contraindication...")
    pregnancy_context = {
        'patient': {'id': 'pregnancy_test', 'age': 28, 'gender': 'female', 'pregnant': True},
        'medications': [
            {'name': 'warfarin', 'dose': '5mg', 'frequency': 'daily'}
        ],
        'conditions': [
            {'name': 'deep vein thrombosis'}
        ],
        'allergies': []
    }
    
    result = await cae_engine.validate_safety(pregnancy_context)
    logger.info(f"   Status: {result['overall_status']}")
    logger.info(f"   Findings: {result['total_findings']}")
    logger.info(f"   Execution Time: {result['performance']['total_execution_time_ms']:.1f}ms")
    
    # Scenario 4: Renal Dosing
    logger.info("\n4. Testing Renal Dosing Adjustment...")
    renal_context = {
        'patient': {'id': 'renal_test', 'age': 75, 'egfr': 25, 'weight': 65},
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
    
    result = await cae_engine.validate_safety(renal_context)
    logger.info(f"   Status: {result['overall_status']}")
    logger.info(f"   Findings: {result['total_findings']}")
    logger.info(f"   Execution Time: {result['performance']['total_execution_time_ms']:.1f}ms")

def main():
    """Main function"""
    logger.info("CAE Engine Neo4j Integration Startup")
    logger.info("====================================")
    
    # Check environment variables
    required_env_vars = ['NEO4J_URI', 'NEO4J_USERNAME', 'NEO4J_PASSWORD']
    missing_vars = [var for var in required_env_vars if not os.getenv(var)]
    
    if missing_vars:
        logger.error(f"❌ Missing environment variables: {missing_vars}")
        logger.error("Please set up your .env file with Neo4j credentials")
        return False
    
    # Run the test
    success = asyncio.run(test_neo4j_integration())
    
    if success:
        logger.info("\n✅ CAE Engine with Neo4j is ready for production!")
        return True
    else:
        logger.error("\n❌ CAE Engine Neo4j integration failed!")
        return False

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
