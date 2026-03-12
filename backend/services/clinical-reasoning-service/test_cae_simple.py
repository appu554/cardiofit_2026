"""
Simple Test Script for CAE Engine Neo4j Integration

Quick validation test that can be run directly without pytest.
Tests core functionality and provides clear pass/fail results.
"""

import asyncio
import logging
import time
from typing import Dict, Any

# Load environment variables
try:
    from dotenv import load_dotenv
    load_dotenv()
    print("✅ Loaded environment variables from .env file")
except ImportError:
    print("⚠️  python-dotenv not installed")

# Add app to path
import sys
from pathlib import Path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

from app.cae_engine_neo4j import CAEEngine

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class SimpleCAETest:
    """Simple test class for CAE Engine validation"""
    
    def __init__(self):
        self.passed_tests = 0
        self.total_tests = 0
        self.cae_engine = None
    
    async def setup(self):
        """Initialize CAE Engine"""
        print("🔧 Setting up CAE Engine...")
        self.cae_engine = CAEEngine()
        initialized = await self.cae_engine.initialize()
        
        if not initialized:
            print("❌ Failed to initialize CAE Engine")
            return False
        
        print("✅ CAE Engine initialized successfully")
        return True
    
    async def teardown(self):
        """Clean up CAE Engine"""
        if self.cae_engine:
            await self.cae_engine.close()
            print("🧹 CAE Engine closed")
    
    def assert_test(self, condition: bool, test_name: str, message: str = ""):
        """Assert test condition and track results"""
        self.total_tests += 1
        
        if condition:
            self.passed_tests += 1
            print(f"✅ {test_name}: PASSED {message}")
            return True
        else:
            print(f"❌ {test_name}: FAILED {message}")
            return False
    
    async def test_health_check(self):
        """Test 1: Health Check"""
        print("\n🏥 Test 1: Health Check")
        
        health = await self.cae_engine.get_health_status()
        
        self.assert_test(
            health['status'] == 'HEALTHY',
            "Health Status",
            f"Status: {health['status']}"
        )
        
        self.assert_test(
            health['neo4j_connection'] is True,
            "Neo4j Connection",
            "Connected to Neo4j Cloud"
        )
        
        self.assert_test(
            len(health['checkers']) >= 4,
            "Clinical Checkers",
            f"Found {len(health['checkers'])} checkers: {health['checkers']}"
        )
    
    async def test_known_allergy(self):
        """Test 2: Known Allergy Detection"""
        print("\n🚨 Test 2: Known Allergy Detection")
        
        clinical_context = {
            'patient': {
                'id': 'test_allergy_patient',
                'age': 35,
                'gender': 'female'
            },
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
        
        start_time = time.time()
        result = await self.cae_engine.validate_safety(clinical_context)
        execution_time = (time.time() - start_time) * 1000
        
        self.assert_test(
            result['overall_status'] == 'UNSAFE',
            "Allergy Detection",
            f"Status: {result['overall_status']}, Findings: {result['total_findings']}"
        )
        
        self.assert_test(
            result['total_findings'] > 0,
            "Allergy Findings",
            f"Found {result['total_findings']} findings"
        )
        
        self.assert_test(
            execution_time < 1000,
            "Performance",
            f"Execution time: {execution_time:.1f}ms"
        )
    
    async def test_pregnancy_contraindication(self):
        """Test 3: Pregnancy Contraindication"""
        print("\n🤰 Test 3: Pregnancy Contraindication")
        
        clinical_context = {
            'patient': {
                'id': 'test_pregnancy_patient',
                'age': 28,
                'gender': 'female',
                'pregnant': True
            },
            'medications': [
                {'name': 'warfarin', 'dose': '5mg', 'frequency': 'daily'}
            ],
            'conditions': [
                {'name': 'deep vein thrombosis'}
            ],
            'allergies': []
        }
        
        start_time = time.time()
        result = await self.cae_engine.validate_safety(clinical_context)
        execution_time = (time.time() - start_time) * 1000
        
        self.assert_test(
            result['overall_status'] in ['WARNING', 'UNSAFE'],
            "Pregnancy Contraindication",
            f"Status: {result['overall_status']}, Findings: {result['total_findings']}"
        )
        
        self.assert_test(
            execution_time < 1000,
            "Performance",
            f"Execution time: {execution_time:.1f}ms"
        )
    
    async def test_renal_dosing(self):
        """Test 4: Renal Dosing Adjustment"""
        print("\n🫘 Test 4: Renal Dosing Adjustment")
        
        clinical_context = {
            'patient': {
                'id': 'test_renal_patient',
                'age': 75,
                'egfr': 25,  # Severe renal impairment
                'weight': 65,
                'gender': 'male'
            },
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
        
        start_time = time.time()
        result = await self.cae_engine.validate_safety(clinical_context)
        execution_time = (time.time() - start_time) * 1000
        
        # Check dose validator ran
        dose_result = result['checker_results'].get('dose', {})
        
        self.assert_test(
            'status' in dose_result,
            "Dose Validator Execution",
            f"Dose checker status: {dose_result.get('status', 'N/A')}"
        )
        
        self.assert_test(
            execution_time < 1000,
            "Performance",
            f"Execution time: {execution_time:.1f}ms"
        )
    
    async def test_drug_interactions(self):
        """Test 5: Drug Interactions"""
        print("\n💊 Test 5: Drug Interactions")
        
        clinical_context = {
            'patient': {
                'id': 'test_ddi_patient',
                'age': 65,
                'weight': 70,
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
        
        start_time = time.time()
        result = await self.cae_engine.validate_safety(clinical_context)
        execution_time = (time.time() - start_time) * 1000
        
        # Check DDI checker ran
        ddi_result = result['checker_results'].get('ddi', {})
        
        self.assert_test(
            'status' in ddi_result,
            "DDI Checker Execution",
            f"DDI checker status: {ddi_result.get('status', 'N/A')}"
        )
        
        self.assert_test(
            result['overall_status'] in ['SAFE', 'WARNING', 'UNSAFE'],
            "Valid Overall Status",
            f"Status: {result['overall_status']}"
        )
        
        self.assert_test(
            execution_time < 1000,
            "Performance",
            f"Execution time: {execution_time:.1f}ms"
        )
    
    async def test_error_handling(self):
        """Test 6: Error Handling"""
        print("\n⚠️  Test 6: Error Handling")
        
        # Invalid context - missing patient ID
        invalid_context = {
            'patient': {'age': 45},  # Missing required ID
            'medications': [
                {'name': 'aspirin', 'dose': '81mg'}
            ],
            'conditions': [],
            'allergies': []
        }
        
        result = await self.cae_engine.validate_safety(invalid_context)
        
        self.assert_test(
            result['overall_status'] == 'ERROR',
            "Error Detection",
            f"Status: {result['overall_status']}"
        )
        
        self.assert_test(
            'error' in result,
            "Error Message",
            "Error message present in response"
        )
    
    def print_summary(self):
        """Print test summary"""
        print("\n" + "=" * 60)
        print("📊 TEST SUMMARY")
        print("=" * 60)
        
        success_rate = (self.passed_tests / self.total_tests * 100) if self.total_tests > 0 else 0
        
        print(f"✅ Passed: {self.passed_tests}")
        print(f"❌ Failed: {self.total_tests - self.passed_tests}")
        print(f"📈 Success Rate: {success_rate:.1f}%")
        print(f"🧪 Total Tests: {self.total_tests}")
        
        if success_rate >= 80:
            print("\n🎉 CAE Engine Neo4j Integration: SUCCESSFUL!")
            print("✅ Ready for production use")
            return True
        else:
            print("\n❌ CAE Engine Neo4j Integration: NEEDS ATTENTION")
            print("⚠️  Some tests failed - review implementation")
            return False

async def main():
    """Main test execution"""
    print("🧪 Simple CAE Engine Neo4j Integration Test")
    print("=" * 60)
    
    test = SimpleCAETest()
    
    try:
        # Setup
        if not await test.setup():
            return False
        
        # Run tests
        await test.test_health_check()
        await test.test_known_allergy()
        await test.test_pregnancy_contraindication()
        await test.test_renal_dosing()
        await test.test_drug_interactions()
        await test.test_error_handling()
        
        # Summary
        success = test.print_summary()
        return success
        
    except Exception as e:
        print(f"❌ Test execution failed: {e}")
        return False
    
    finally:
        await test.teardown()

if __name__ == "__main__":
    success = asyncio.run(main())
    exit(0 if success else 1)
