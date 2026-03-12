"""
Simple workflow query test - minimal dependencies
"""
import requests
import json

def test_workflow_service():
    """Test workflow service with simple HTTP requests."""
    
    workflow_url = "http://localhost:8015"
    
    print("🔍 Testing Workflow Service")
    print("=" * 40)
    
    # Test 1: Health Check
    print("\n1. Health Check...")
    try:
        response = requests.get(f"{workflow_url}/health", timeout=5)
        if response.status_code == 200:
            health_data = response.json()
            print("✅ Service is healthy")
            print(f"   Service: {health_data.get('service')}")
            print(f"   Version: {health_data.get('version')}")
            print(f"   Port: {health_data.get('port', 'N/A')}")
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return
    except Exception as e:
        print(f"❌ Cannot connect to service: {e}")
        print("   Make sure the service is running: python start_service.py")
        return
    
    # Test 2: GraphQL Schema Introspection
    print("\n2. GraphQL Schema Test...")
    
    introspection_query = """
    query IntrospectionQuery {
      __schema {
        queryType {
          name
          fields {
            name
            description
          }
        }
        mutationType {
          name
          fields {
            name
            description
          }
        }
      }
    }
    """
    
    headers = {
        "Content-Type": "application/json",
        "X-User-ID": "test-user-123",
        "X-User-Role": "doctor"
    }
    
    payload = {
        "query": introspection_query
    }
    
    try:
        response = requests.post(
            f"{workflow_url}/api/federation",
            json=payload,
            headers=headers,
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["__schema"]:
                schema = result["data"]["__schema"]
                print("✅ GraphQL schema accessible")
                
                # Show available queries
                if schema.get("queryType", {}).get("fields"):
                    queries = schema["queryType"]["fields"]
                    print(f"   Available Queries ({len(queries)}):")
                    for query in queries[:5]:  # Show first 5
                        print(f"     - {query['name']}")
                
                # Show available mutations
                if schema.get("mutationType", {}).get("fields"):
                    mutations = schema["mutationType"]["fields"]
                    print(f"   Available Mutations ({len(mutations)}):")
                    for mutation in mutations[:5]:  # Show first 5
                        print(f"     - {mutation['name']}")
            else:
                print("❌ Invalid schema response")
        else:
            print(f"❌ GraphQL request failed: {response.status_code}")
            print(f"   Response: {response.text}")
    except Exception as e:
        print(f"❌ GraphQL test failed: {e}")
    
    # Test 3: Simple Workflow Definitions Query
    print("\n3. Workflow Definitions Query...")
    
    workflow_query = """
    query GetWorkflowDefinitions {
      workflowDefinitions {
        id
        name
        version
        status
      }
    }
    """
    
    payload = {
        "query": workflow_query
    }
    
    try:
        response = requests.post(
            f"{workflow_url}/api/federation",
            json=payload,
            headers=headers,
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            if "errors" in result:
                print(f"❌ Query errors: {result['errors']}")
            elif "data" in result:
                definitions = result["data"].get("workflowDefinitions", [])
                print(f"✅ Found {len(definitions)} workflow definitions")
                for definition in definitions:
                    print(f"   - {definition.get('name')} (Status: {definition.get('status')})")
            else:
                print("❌ No data in response")
        else:
            print(f"❌ Query failed: {response.status_code}")
            print(f"   Response: {response.text}")
    except Exception as e:
        print(f"❌ Workflow definitions query failed: {e}")
    
    # Test 4: Tasks Query
    print("\n4. Tasks Query...")
    
    tasks_query = """
    query GetTasks {
      tasks {
        id
        description
        status
        priority
      }
    }
    """
    
    payload = {
        "query": tasks_query
    }
    
    try:
        response = requests.post(
            f"{workflow_url}/api/federation",
            json=payload,
            headers=headers,
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            if "errors" in result:
                print(f"❌ Query errors: {result['errors']}")
            elif "data" in result:
                tasks = result["data"].get("tasks", [])
                print(f"✅ Found {len(tasks)} tasks")
                for task in tasks:
                    print(f"   - {task.get('description', 'No description')} (Status: {task.get('status')})")
            else:
                print("❌ No data in response")
        else:
            print(f"❌ Query failed: {response.status_code}")
            print(f"   Response: {response.text}")
    except Exception as e:
        print(f"❌ Tasks query failed: {e}")
    
    print("\n" + "=" * 40)
    print("✅ Basic testing completed!")
    print("\nIf queries are working:")
    print("1. Start other services for full integration")
    print("2. Run: python test_patient_workflow_integration.py")
    print("3. Test with Postman collection")

if __name__ == "__main__":
    test_workflow_service()
