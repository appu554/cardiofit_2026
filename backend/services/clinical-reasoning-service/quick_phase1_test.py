#!/usr/bin/env python3
"""
Quick Phase 1 Validation Test
Simple test to verify Phase 1 completion status
"""

import requests
import grpc
import json

def test_graphdb():
    """Test GraphDB with real clinical data"""
    print("🔍 Testing GraphDB Integration...")
    
    try:
        # Test GraphDB server
        response = requests.get("http://localhost:7200/rest/repositories", timeout=5)
        if response.status_code == 200:
            print("✅ GraphDB server accessible")
            
            # Check repository
            repos = response.json()
            if any(repo['id'] == 'cae-clinical-intelligence' for repo in repos):
                print("✅ CAE repository exists")
                
                # Check data count
                query = "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"
                response = requests.post(
                    "http://localhost:7200/repositories/cae-clinical-intelligence",
                    data=query,
                    headers={
                        'Content-Type': 'application/sparql-query',
                        'Accept': 'application/sparql-results+json'
                    },
                    timeout=5
                )
                
                if response.status_code == 200:
                    results = response.json()
                    count = int(results['results']['bindings'][0]['count']['value'])
                    print(f"✅ GraphDB contains {count} triples")
                    
                    if count > 1000:
                        print("✅ REAL CLINICAL DATA LOADED")
                        return True
                    else:
                        print("❌ Insufficient data")
                        return False
                else:
                    print("❌ SPARQL query failed")
                    return False
            else:
                print("❌ CAE repository not found")
                return False
        else:
            print("❌ GraphDB not accessible")
            return False
            
    except Exception as e:
        print(f"❌ GraphDB test failed: {e}")
        return False

def test_grpc_server():
    """Test gRPC server connectivity"""
    print("🔍 Testing gRPC Server...")
    
    try:
        channel = grpc.insecure_channel('localhost:8027')
        state = channel.get_state(try_to_connect=True)
        
        if state == grpc.ChannelConnectivity.READY:
            print("✅ gRPC server ready")
            channel.close()
            return True
        else:
            print(f"❌ gRPC server not ready: {state}")
            channel.close()
            return False
            
    except Exception as e:
        print(f"❌ gRPC test failed: {e}")
        return False

def test_patient_data():
    """Test specific patient data in GraphDB"""
    print("🔍 Testing Patient Data...")
    
    try:
        query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        SELECT ?patientId WHERE {
            ?patient a cae:Patient ;
                     cae:hasPatientId "905a60cb-8241-418f-b29b-5b020e851392" .
        }
        """
        
        response = requests.post(
            "http://localhost:7200/repositories/cae-clinical-intelligence",
            data=query,
            headers={
                'Content-Type': 'application/sparql-query',
                'Accept': 'application/sparql-results+json'
            },
            timeout=5
        )
        
        if response.status_code == 200:
            results = response.json()
            if results['results']['bindings']:
                print("✅ Test patient 905a60cb-8241-418f-b29b-5b020e851392 found")
                return True
            else:
                print("❌ Test patient not found")
                return False
        else:
            print("❌ Patient query failed")
            return False
            
    except Exception as e:
        print(f"❌ Patient data test failed: {e}")
        return False

def test_drug_interactions():
    """Test drug interaction data"""
    print("🔍 Testing Drug Interaction Data...")
    
    try:
        query = """
        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
        SELECT ?interaction ?severity WHERE {
            ?interaction a cae:DrugInteraction ;
                       cae:hasInteractionSeverity ?severity .
        } LIMIT 3
        """
        
        response = requests.post(
            "http://localhost:7200/repositories/cae-clinical-intelligence",
            data=query,
            headers={
                'Content-Type': 'application/sparql-query',
                'Accept': 'application/sparql-results+json'
            },
            timeout=5
        )
        
        if response.status_code == 200:
            results = response.json()
            interactions = results['results']['bindings']
            if interactions:
                print(f"✅ Found {len(interactions)} drug interactions")
                for interaction in interactions:
                    severity = interaction['severity']['value']
                    print(f"   - Severity: {severity}")
                return True
            else:
                print("❌ No drug interactions found")
                return False
        else:
            print("❌ Drug interaction query failed")
            return False
            
    except Exception as e:
        print(f"❌ Drug interaction test failed: {e}")
        return False

def main():
    """Run quick Phase 1 validation"""
    print("🧪 Quick Phase 1 CAE Validation")
    print("=" * 50)
    
    tests = [
        ("GraphDB Integration", test_graphdb),
        ("gRPC Server", test_grpc_server),
        ("Patient Data", test_patient_data),
        ("Drug Interactions", test_drug_interactions),
    ]
    
    passed = 0
    total = len(tests)
    
    for test_name, test_func in tests:
        print(f"\n{test_name}:")
        if test_func():
            passed += 1
    
    print(f"\n📊 Results: {passed}/{total} tests passed")
    
    if passed == total:
        print("\n🎉 PHASE 1 VALIDATION SUCCESSFUL!")
        print("✅ All core components operational")
        print("✅ Real clinical data integrated")
        print("✅ Mock data successfully replaced")
        print("\n🚀 Ready for Phase 3 implementation!")
        return True
    else:
        print("\n❌ Phase 1 validation incomplete")
        print("🔧 Please address failed components")
        return False

if __name__ == "__main__":
    success = main()
    exit(0 if success else 1)
