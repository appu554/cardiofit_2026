#!/usr/bin/env python3
"""
Test GraphDB Connection and Import Status
"""

import requests
import json
import sys

def test_graphdb_connection():
    """Test if GraphDB is accessible and check repository status"""
    graphdb_url = "http://localhost:7200"
    repository_id = "cae-clinical-intelligence"
    
    print("🧪 Testing GraphDB Connection...")
    print("=" * 50)
    
    try:
        # Test GraphDB server status
        print("🔍 Testing GraphDB server...")
        response = requests.get(f"{graphdb_url}/rest/repositories", timeout=10)
        if response.status_code == 200:
            print("✅ GraphDB server is accessible")
            repositories = response.json()
            print(f"📊 Found {len(repositories)} repositories")
            
            # Check if our repository exists
            repo_exists = any(repo['id'] == repository_id for repo in repositories)
            if repo_exists:
                print(f"✅ Repository '{repository_id}' exists")
                
                # Test repository access
                repo_url = f"{graphdb_url}/repositories/{repository_id}"
                repo_response = requests.get(f"{repo_url}/size", timeout=10)
                if repo_response.status_code == 200:
                    size_data = repo_response.json()
                    print(f"📈 Repository size: {size_data} triples")
                    
                    # Test SPARQL query
                    sparql_query = """
                    PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
                    SELECT (COUNT(*) as ?count) WHERE {
                        ?s ?p ?o
                    }
                    """
                    
                    sparql_response = requests.post(
                        f"{repo_url}",
                        data=sparql_query,
                        headers={
                            'Content-Type': 'application/sparql-query',
                            'Accept': 'application/sparql-results+json'
                        },
                        timeout=10
                    )
                    
                    if sparql_response.status_code == 200:
                        results = sparql_response.json()
                        count = results['results']['bindings'][0]['count']['value']
                        print(f"✅ SPARQL query successful: {count} total triples")
                        
                        # Test for specific patient data
                        patient_query = """
                        PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
                        SELECT ?patient ?patientId WHERE {
                            ?patient a cae:Patient ;
                                     cae:hasPatientId ?patientId .
                        } LIMIT 5
                        """
                        
                        patient_response = requests.post(
                            f"{repo_url}",
                            data=patient_query,
                            headers={
                                'Content-Type': 'application/sparql-query',
                                'Accept': 'application/sparql-results+json'
                            },
                            timeout=10
                        )
                        
                        if patient_response.status_code == 200:
                            patient_results = patient_response.json()
                            patients = patient_results['results']['bindings']
                            print(f"👥 Found {len(patients)} patients in GraphDB:")
                            for patient in patients:
                                patient_id = patient['patientId']['value']
                                print(f"   - {patient_id}")
                                
                            # Check for the specific test patient
                            test_patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
                            has_test_patient = any(p['patientId']['value'] == test_patient_id for p in patients)
                            if has_test_patient:
                                print(f"✅ Test patient {test_patient_id} found in GraphDB")
                            else:
                                print(f"⚠️  Test patient {test_patient_id} not found - data may need to be imported")
                        else:
                            print(f"❌ Patient query failed: {patient_response.status_code}")
                    else:
                        print(f"❌ SPARQL query failed: {sparql_response.status_code}")
                else:
                    print(f"❌ Repository access failed: {repo_response.status_code}")
            else:
                print(f"⚠️  Repository '{repository_id}' does not exist - needs to be created")
                print("💡 Available repositories:")
                for repo in repositories:
                    print(f"   - {repo['id']}: {repo.get('title', 'No title')}")
        else:
            print(f"❌ GraphDB server not accessible: {response.status_code}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Connection error: {e}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False
    
    print("\n🎯 GraphDB Connection Test Complete")
    return True

if __name__ == "__main__":
    success = test_graphdb_connection()
    sys.exit(0 if success else 1)
