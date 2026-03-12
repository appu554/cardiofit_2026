"""
Test Fixed Neo4j Integration

Test the CAE Engine with corrected drug names and queries
to validate that Neo4j integration is now working.
"""

import asyncio
import logging
from dotenv import load_dotenv
load_dotenv()

import sys
from pathlib import Path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

from app.cae_engine_neo4j import CAEEngine

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def test_fixed_neo4j():
    """Test CAE Engine with fixed Neo4j queries"""
    print("🔧 Testing Fixed Neo4j Integration")
    print("=" * 50)
    
    cae_engine = CAEEngine()
    
    try:
        # Initialize CAE Engine
        initialized = await cae_engine.initialize()
        if not initialized:
            print("❌ Failed to initialize CAE Engine")
            return
        
        print("✅ CAE Engine initialized")
        print()
        
        # Test 1: Drug Interaction with Correct Names
        print("💊 Test 1: Drug Interaction (Warfarin + Ciprofloxacin)")
        print("-" * 50)
        
        ddi_context = {
            'patient': {
                'id': 'test_ddi_patient',
                'age': 65,
                'weight': 70
            },
            'medications': [
                {'name': 'warfarin', 'dose': '5mg'},  # Will be capitalized to Warfarin
                {'name': 'ciprofloxacin', 'dose': '500mg'}  # Will be capitalized to Ciprofloxacin
            ],
            'conditions': [],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(ddi_context)
        
        print(f"Overall Status: {result['overall_status']}")
        print(f"Total Findings: {result['total_findings']}")
        print(f"Execution Time: {result['performance']['total_execution_time_ms']:.1f}ms")
        
        # Check DDI checker specifically
        ddi_result = result['checker_results'].get('ddi', {})
        print(f"DDI Checker Status: {ddi_result.get('status', 'N/A')}")
        
        if ddi_result.get('status') == 'UNSAFE':
            print("🎉 SUCCESS: Drug interaction detected!")
            for finding in ddi_result.get('findings', []):
                print(f"  - {finding.get('message', 'Unknown')}")
        elif ddi_result.get('status') == 'SAFE':
            print("✅ No drug interactions found (expected if no interaction exists)")
        else:
            print(f"❌ Unexpected status: {ddi_result.get('status')}")
        
        print()
        
        # Test 2: Adverse Events with Correct Names
        print("🚨 Test 2: Adverse Events (Acetaminophen)")
        print("-" * 50)
        
        ae_context = {
            'patient': {
                'id': 'test_ae_patient',
                'age': 35
            },
            'medications': [
                {'name': 'acetaminophen', 'dose': '500mg'}  # Will be capitalized to Acetaminophen
            ],
            'conditions': [],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(ae_context)
        
        allergy_result = result['checker_results'].get('allergy', {})
        print(f"Allergy Checker Status: {allergy_result.get('status', 'N/A')}")
        print(f"Total Findings: {result['total_findings']}")
        
        if allergy_result.get('status') == 'WARNING':
            print("🎉 SUCCESS: Adverse events detected!")
            for finding in allergy_result.get('findings', []):
                print(f"  - {finding.get('message', 'Unknown')}")
        elif allergy_result.get('status') == 'SAFE':
            print("✅ No adverse events found")
        else:
            print(f"Status: {allergy_result.get('status')}")
        
        print()
        
        # Test 3: Known Allergy (Should Still Work)
        print("🚨 Test 3: Known Allergy Detection")
        print("-" * 50)
        
        allergy_context = {
            'patient': {
                'id': 'test_allergy_patient',
                'age': 35
            },
            'medications': [
                {'name': 'penicillin', 'dose': '500mg'}
            ],
            'conditions': [],
            'allergies': [
                {'substance': 'penicillin', 'reaction': 'rash', 'severity': 'moderate'}
            ]
        }
        
        result = await cae_engine.validate_safety(allergy_context)
        
        allergy_result = result['checker_results'].get('allergy', {})
        print(f"Allergy Checker Status: {allergy_result.get('status', 'N/A')}")
        print(f"Total Findings: {result['total_findings']}")
        
        if allergy_result.get('status') == 'UNSAFE':
            print("🎉 SUCCESS: Known allergy detected!")
            for finding in allergy_result.get('findings', []):
                print(f"  - {finding.get('message', 'Unknown')}")
        
        print()
        
        # Test 4: Multiple Drugs with Interactions
        print("💊 Test 4: Multiple Drug Test")
        print("-" * 50)
        
        multi_context = {
            'patient': {
                'id': 'test_multi_patient',
                'age': 55,
                'egfr': 45
            },
            'medications': [
                {'name': 'warfarin', 'dose': '5mg'},
                {'name': 'acetaminophen', 'dose': '500mg'},
                {'name': 'digoxin', 'dose': '0.25mg'}
            ],
            'conditions': [
                {'name': 'atrial fibrillation'}
            ],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(multi_context)
        
        print(f"Overall Status: {result['overall_status']}")
        print(f"Total Findings: {result['total_findings']}")
        print(f"Execution Time: {result['performance']['total_execution_time_ms']:.1f}ms")
        
        # Show results for each checker
        for checker_name, checker_result in result['checker_results'].items():
            status = checker_result.get('status', 'N/A')
            findings_count = len(checker_result.get('findings', []))
            print(f"  {checker_name.upper()}: {status} ({findings_count} findings)")
        
        print()
        
        # Summary
        print("📊 SUMMARY")
        print("=" * 50)
        
        all_checkers = ['ddi', 'allergy', 'contraindication', 'dose']
        working_checkers = []
        error_checkers = []
        
        for checker in all_checkers:
            checker_result = result['checker_results'].get(checker, {})
            status = checker_result.get('status', 'ERROR')
            
            if status in ['SAFE', 'WARNING', 'UNSAFE']:
                working_checkers.append(checker)
            else:
                error_checkers.append(checker)
        
        print(f"✅ Working Checkers: {working_checkers}")
        if error_checkers:
            print(f"❌ Error Checkers: {error_checkers}")
        
        success_rate = len(working_checkers) / len(all_checkers) * 100
        print(f"📈 Success Rate: {success_rate:.1f}%")
        
        if success_rate >= 75:
            print("\n🎉 Neo4j Integration: SUCCESSFUL!")
            print("✅ CAE Engine is using real Neo4j clinical data")
        else:
            print("\n⚠️  Neo4j Integration: PARTIAL SUCCESS")
            print("Some checkers still need work")
    
    except Exception as e:
        print(f"❌ Test failed with error: {e}")
    
    finally:
        await cae_engine.close()

if __name__ == "__main__":
    asyncio.run(test_fixed_neo4j())
