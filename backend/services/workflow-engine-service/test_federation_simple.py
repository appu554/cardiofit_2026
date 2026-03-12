"""
Simple federation test to debug the GraphQL endpoint
"""
import requests
import json

def test_federation_gateway():
    """Test the federation gateway step by step."""
    
    print("🔍 Testing Federation Gateway")
    print("=" * 40)
    
    # Test 1: Health check
    print("\n1. Health Check...")
    try:
        response = requests.get("http://localhost:4000/health", timeout=5)
        if response.status_code == 200:
            print("✅ Federation Gateway health check passed")
            print(f"   Response: {response.json()}")
        else:
            print(f"❌ Health check failed: {response.status_code}")
            print(f"   Response: {response.text}")
            return False
    except Exception as e:
        print(f"❌ Health check failed: {e}")
        return False
    
    # Test 2: Root endpoint
    print("\n2. Root Endpoint...")
    try:
        response = requests.get("http://localhost:4000", timeout=5)
        print(f"   Status: {response.status_code}")
        print(f"   Content-Type: {response.headers.get('content-type', 'unknown')}")
        print(f"   Response: {response.text[:200]}...")
    except Exception as e:
        print(f"❌ Root endpoint test failed: {e}")
    
    # Test 3: GraphQL endpoint introspection
    print("\n3. GraphQL Introspection...")
    query = {"query": "{ __typename }"}
    
    try:
        response = requests.post(
            "http://localhost:4000/graphql",
            json=query,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            print("✅ GraphQL endpoint is working")
            print(f"   Response: {result}")
        else:
            print(f"❌ GraphQL endpoint failed: {response.status_code}")
            print(f"   Content-Type: {response.headers.get('content-type', 'unknown')}")
            print(f"   Response: {response.text}")
            return False
    except Exception as e:
        print(f"❌ GraphQL introspection failed: {e}")
        return False
    
    # Test 4: Schema introspection
    print("\n4. Schema Introspection...")
    schema_query = {
        "query": """
        query IntrospectionQuery {
          __schema {
            queryType {
              name
              fields {
                name
              }
            }
          }
        }
        """
    }
    
    try:
        response = requests.post(
            "http://localhost:4000/graphql",
            json=schema_query,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["__schema"]:
                schema = result["data"]["__schema"]
                query_type = schema["queryType"]
                fields = query_type.get("fields", [])
                print("✅ Schema introspection successful")
                print(f"   Query type: {query_type['name']}")
                print(f"   Available fields: {[f['name'] for f in fields[:5]]}...")
            else:
                print(f"❌ Invalid schema response: {result}")
        else:
            print(f"❌ Schema introspection failed: {response.status_code}")
            print(f"   Response: {response.text}")
    except Exception as e:
        print(f"❌ Schema introspection failed: {e}")
    
    # Test 5: Simple patient query
    print("\n5. Simple Patient Query...")
    patient_query = {
        "query": """
        query GetPatients {
          patients {
            items {
              id
              name {
                family
                given
              }
            }
          }
        }
        """
    }
    
    headers = {
        "Content-Type": "application/json",
        "X-User-ID": "test-user-123",
        "X-User-Role": "doctor",
        "X-User-Permissions": "patient:read"
    }
    
    try:
        response = requests.post(
            "http://localhost:4000/graphql",
            json=patient_query,
            headers=headers,
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            if "errors" in result:
                print(f"⚠️  Query returned errors: {result['errors']}")
            elif "data" in result:
                patients = result["data"].get("patients", {}).get("items", [])
                print(f"✅ Patient query successful: Found {len(patients)} patients")
                for patient in patients[:3]:  # Show first 3
                    name = patient.get("name", [{}])[0]
                    print(f"   - {name.get('given', [''])[0]} {name.get('family', '')}")
            else:
                print(f"❌ Unexpected response: {result}")
        else:
            print(f"❌ Patient query failed: {response.status_code}")
            print(f"   Response: {response.text}")
    except Exception as e:
        print(f"❌ Patient query failed: {e}")
    
    print("\n" + "=" * 40)
    print("✅ Federation Gateway testing completed!")
    return True

if __name__ == "__main__":
    test_federation_gateway()
