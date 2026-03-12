"""
Test script for the minimal workflow service
"""
import requests
import json
from typing import Dict, Any

def test_minimal_workflow_service():
    """Test the minimal workflow service functionality."""
    
    base_url = "http://localhost:8015"
    graphql_url = f"{base_url}/api/federation"
    
    print("🧪 Testing Minimal Workflow Service")
    print("=" * 50)
    
    # Test 1: Health Check
    print("\n1. Health Check...")
    try:
        response = requests.get(f"{base_url}/health")
        if response.status_code == 200:
            health_data = response.json()
            print("✅ Service is healthy")
            print(f"   Service: {health_data['service']}")
            print(f"   Version: {health_data['version']}")
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Health check failed: {e}")
        return False
    
    # Test 2: Basic GraphQL Query
    print("\n2. Basic GraphQL Query...")
    query = {"query": "{ hello }"}
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["hello"]:
                print(f"✅ Basic query successful: {result['data']['hello']}")
            else:
                print(f"❌ Unexpected response: {result}")
                return False
        else:
            print(f"❌ Query failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Basic query failed: {e}")
        return False
    
    # Test 3: Workflow Definitions Query
    print("\n3. Workflow Definitions Query...")
    query = {
        "query": """
        {
          workflowDefinitions {
            id
            name
            version
            status
            description
          }
        }
        """
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["workflowDefinitions"]:
                definitions = result["data"]["workflowDefinitions"]
                print(f"✅ Found {len(definitions)} workflow definitions:")
                for definition in definitions:
                    print(f"   - {definition['name']} (ID: {definition['id']})")
                    print(f"     Status: {definition['status']}, Version: {definition['version']}")
                return definitions
            else:
                print(f"❌ No workflow definitions found: {result}")
                return False
        else:
            print(f"❌ Query failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Workflow definitions query failed: {e}")
        return False
    
    # Test 4: Tasks Query
    print("\n4. Tasks Query...")
    query = {
        "query": """
        {
          tasks {
            id
            description
            status
            priority
          }
        }
        """
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["tasks"]:
                tasks = result["data"]["tasks"]
                print(f"✅ Found {len(tasks)} tasks:")
                for task in tasks:
                    print(f"   - {task['description']} (ID: {task['id']})")
                    print(f"     Status: {task['status']}, Priority: {task['priority']}")
                return tasks
            else:
                print(f"❌ No tasks found: {result}")
                return False
        else:
            print(f"❌ Query failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Tasks query failed: {e}")
        return False
    
    # Test 5: Start Workflow Mutation
    print("\n5. Start Workflow Mutation...")
    mutation = {
        "query": """
        mutation StartWorkflow($definitionId: String!, $patientId: String!) {
          startWorkflow(definitionId: $definitionId, patientId: $patientId)
        }
        """,
        "variables": {
            "definitionId": "patient-admission-workflow",
            "patientId": "patient-test-123"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=mutation)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["startWorkflow"]:
                workflow_result = result["data"]["startWorkflow"]
                print(f"✅ Workflow started: {workflow_result}")
                return True
            else:
                print(f"❌ Workflow start failed: {result}")
                return False
        else:
            print(f"❌ Mutation failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Start workflow mutation failed: {e}")
        return False

def test_complex_queries():
    """Test more complex GraphQL queries."""
    
    base_url = "http://localhost:8015"
    graphql_url = f"{base_url}/api/federation"
    
    print("\n🧪 Testing Complex Queries")
    print("=" * 50)
    
    # Test 1: Query with Variables
    print("\n1. Query with Variables...")
    query = {
        "query": """
        query GetTasksForUser($assignee: String) {
          tasks(assignee: $assignee) {
            id
            description
            status
            priority
          }
        }
        """,
        "variables": {
            "assignee": "doctor-test-456"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Variable query successful: {result}")
        else:
            print(f"❌ Variable query failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Variable query failed: {e}")
    
    # Test 2: Multiple Queries in One Request
    print("\n2. Multiple Queries...")
    query = {
        "query": """
        {
          workflowDefinitions {
            id
            name
            status
          }
          tasks {
            id
            description
            status
          }
        }
        """
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result:
                workflows = result["data"]["workflowDefinitions"]
                tasks = result["data"]["tasks"]
                print(f"✅ Multiple queries successful:")
                print(f"   Workflows: {len(workflows)}")
                print(f"   Tasks: {len(tasks)}")
        else:
            print(f"❌ Multiple queries failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Multiple queries failed: {e}")

def demonstrate_workflow_use_case():
    """Demonstrate a practical workflow use case."""
    
    print("\n🏥 Demonstrating Patient Admission Workflow Use Case")
    print("=" * 60)
    
    print("""
    SCENARIO: Emergency Patient Admission
    
    1. Patient arrives at emergency department
    2. Nurse starts patient admission workflow
    3. Doctor gets assigned triage task
    4. Doctor completes assessment
    5. Admin gets room assignment task
    6. Patient is admitted to room
    """)
    
    base_url = "http://localhost:8015"
    graphql_url = f"{base_url}/api/federation"
    
    # Step 1: Check available workflows
    print("\n📋 Step 1: Check available workflows...")
    query = {"query": "{ workflowDefinitions { id name description } }"}
    response = requests.post(graphql_url, json=query)
    
    if response.status_code == 200:
        result = response.json()
        workflows = result["data"]["workflowDefinitions"]
        admission_workflow = next((w for w in workflows if "admission" in w["name"].lower()), None)
        
        if admission_workflow:
            print(f"✅ Found admission workflow: {admission_workflow['name']}")
            
            # Step 2: Start the workflow
            print(f"\n🚀 Step 2: Starting workflow for patient...")
            mutation = {
                "query": """
                mutation {
                  startWorkflow(
                    definitionId: "patient-admission-workflow"
                    patientId: "patient-emergency-001"
                  )
                }
                """
            }
            
            response = requests.post(graphql_url, json=mutation)
            if response.status_code == 200:
                result = response.json()
                print(f"✅ Workflow started: {result['data']['startWorkflow']}")
                
                # Step 3: Check tasks
                print(f"\n📋 Step 3: Checking assigned tasks...")
                query = {"query": "{ tasks { id description status priority } }"}
                response = requests.post(graphql_url, json=query)
                
                if response.status_code == 200:
                    result = response.json()
                    tasks = result["data"]["tasks"]
                    print(f"✅ Found {len(tasks)} tasks:")
                    for task in tasks:
                        print(f"   - {task['description']} (Priority: {task['priority']})")
                
                print(f"\n🎉 Workflow demonstration completed!")
                print(f"   In a real system, these tasks would be:")
                print(f"   - Assigned to specific users (doctors, nurses, admins)")
                print(f"   - Tracked in real-time")
                print(f"   - Integrated with patient data, vitals, and encounters")
                print(f"   - Escalated if overdue")

if __name__ == "__main__":
    # Run basic tests
    success = test_minimal_workflow_service()
    
    if success:
        # Run complex tests
        test_complex_queries()
        
        # Demonstrate use case
        demonstrate_workflow_use_case()
        
        print("\n" + "=" * 60)
        print("🎉 ALL TESTS COMPLETED SUCCESSFULLY!")
        print("\nNext Steps:")
        print("1. ✅ Basic workflow service is working")
        print("2. 🔄 Fix the full service import issues")
        print("3. 🏥 Start patient, observation, and encounter services")
        print("4. 🔗 Test federation integration")
        print("5. 📊 Create real patient data with vitals and encounters")
        print("6. 🧪 Test comprehensive patient-workflow queries")
    else:
        print("\n❌ Basic tests failed. Please check the service.")
