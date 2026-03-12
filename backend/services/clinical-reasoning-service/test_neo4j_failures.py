"""
Test Neo4j Query Failures

This test will show exactly which Neo4j queries are failing
and what data is missing from the knowledge graph.
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

async def test_neo4j_failures():
    """Test CAE Engine and show Neo4j query failures"""
    print("🔍 Testing Neo4j Query Failures")
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
        
        # Test 1: Drug Interaction Query
        print("💊 Test 1: Drug Interaction Detection")
        print("-" * 30)
        
        ddi_context = {
            'patient': {
                'id': 'test_ddi_patient',
                'age': 65,
                'weight': 70
            },
            'medications': [
                {'name': 'warfarin', 'dose': '5mg'},
                {'name': 'ciprofloxacin', 'dose': '500mg'}
            ],
            'conditions': [],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(ddi_context)
        
        print(f"Overall Status: {result['overall_status']}")
        print(f"Total Findings: {result['total_findings']}")
        
        # Check DDI checker specifically
        ddi_result = result['checker_results'].get('ddi', {})
        print(f"DDI Checker Status: {ddi_result.get('status', 'N/A')}")
        
        if ddi_result.get('status') == 'ERROR':
            print("❌ DDI Query Failed!")
            for finding in ddi_result.get('findings', []):
                print(f"  Error: {finding.get('message', 'Unknown error')}")
                print(f"  Expected: {finding.get('details', {}).get('expected_relationships', [])}")
        
        print()
        
        # Test 2: Adverse Events Query
        print("🚨 Test 2: Adverse Events Detection")
        print("-" * 30)
        
        ae_context = {
            'patient': {
                'id': 'test_ae_patient',
                'age': 35
            },
            'medications': [
                {'name': 'penicillin', 'dose': '500mg'}
            ],
            'conditions': [],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(ae_context)
        
        allergy_result = result['checker_results'].get('allergy', {})
        print(f"Allergy Checker Status: {allergy_result.get('status', 'N/A')}")
        
        if allergy_result.get('status') == 'ERROR':
            print("❌ Adverse Events Query Failed!")
            for finding in allergy_result.get('findings', []):
                print(f"  Error: {finding.get('message', 'Unknown error')}")
                print(f"  Expected: {finding.get('details', {}).get('expected_relationships', [])}")
        
        print()
        
        # Test 3: Contraindications Query
        print("⚠️  Test 3: Contraindications Detection")
        print("-" * 30)
        
        contra_context = {
            'patient': {
                'id': 'test_contra_patient',
                'age': 28,
                'pregnant': True
            },
            'medications': [
                {'name': 'warfarin', 'dose': '5mg'}
            ],
            'conditions': [
                {'name': 'pregnancy'}
            ],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(contra_context)
        
        contra_result = result['checker_results'].get('contraindication', {})
        print(f"Contraindication Checker Status: {contra_result.get('status', 'N/A')}")
        
        if contra_result.get('status') == 'ERROR':
            print("❌ Contraindications Query Failed!")
            for finding in contra_result.get('findings', []):
                print(f"  Error: {finding.get('message', 'Unknown error')}")
                print(f"  Expected: {finding.get('details', {}).get('expected_relationships', [])}")
        
        print()
        
        # Test 4: Dosing Adjustments Query
        print("💉 Test 4: Dosing Adjustments Detection")
        print("-" * 30)
        
        dose_context = {
            'patient': {
                'id': 'test_dose_patient',
                'age': 75,
                'egfr': 25,  # Severe renal impairment
                'weight': 65
            },
            'medications': [
                {'name': 'digoxin', 'dose': '0.25mg'},
                {'name': 'metformin', 'dose': '1000mg'}
            ],
            'conditions': [
                {'name': 'chronic kidney disease'}
            ],
            'allergies': []
        }
        
        result = await cae_engine.validate_safety(dose_context)
        
        dose_result = result['checker_results'].get('dose', {})
        print(f"Dose Validator Status: {dose_result.get('status', 'N/A')}")
        
        if dose_result.get('status') == 'ERROR':
            print("❌ Dosing Adjustments Query Failed!")
            for finding in dose_result.get('findings', []):
                print(f"  Error: {finding.get('message', 'Unknown error')}")
                print(f"  Expected: {finding.get('details', {}).get('expected_relationships', [])}")
        
        print()
        
        # Summary
        print("📊 SUMMARY OF FAILURES")
        print("=" * 50)
        
        checkers = ['ddi', 'allergy', 'contraindication', 'dose']
        failed_checkers = []
        
        for checker in checkers:
            checker_result = result['checker_results'].get(checker, {})
            if checker_result.get('status') == 'ERROR':
                failed_checkers.append(checker)
        
        if failed_checkers:
            print(f"❌ Failed Checkers: {failed_checkers}")
            print("\n🔧 MISSING NEO4J RELATIONSHIPS:")
            print("Based on the errors above, you need these relationships:")
            print("  - cae_interactsWith (for drug interactions)")
            print("  - cae_hasAdverseEvent (for adverse events)")
            print("  - cae_contraindicatedIn (for contraindications)")
            print("  - cae_requiresRenalAdjustment (for dosing adjustments)")
            
            print("\n💡 NEXT STEPS:")
            print("1. Check if these relationships exist in your Neo4j database")
            print("2. If missing, run the knowledge pipeline to create them")
            print("3. Or update the CAE queries to match your existing schema")
        else:
            print("✅ All checkers working with Neo4j data!")
    
    except Exception as e:
        print(f"❌ Test failed with error: {e}")
    
    finally:
        await cae_engine.close()

if __name__ == "__main__":
    asyncio.run(test_neo4j_failures())
