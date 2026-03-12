"""
Basic Workflow Testing Script

This script demonstrates basic workflow functionality:
1. Check service health
2. Query workflow definitions
3. Start a simple workflow
4. Query patient with workflow data
"""
import asyncio
import json
import aiohttp
from datetime import datetime


async def test_workflow_service():
    """Test basic workflow service functionality."""
    
    # Service URLs
    federation_url = "http://localhost:4000"
    workflow_url = "http://localhost:8015"
    
    # Test headers
    headers = {
        "Content-Type": "application/json",
        "X-User-ID": "doctor-test-456",
        "X-User-Role": "doctor",
        "X-User-Roles": "doctor,admin",
        "X-User-Permissions": "patient:read,patient:write,task:read,task:write,workflow:read,workflow:write"
    }
    
    print("🔍 Testing Basic Workflow Functionality")
    print("=" * 50)
    
    async with aiohttp.ClientSession() as session:
        
        # 1. Check Workflow Service Health
        print("\n1. Checking Workflow Service Health...")
        try:
            async with session.get(f"{workflow_url}/health", timeout=5) as response:
                if response.status == 200:
                    health_data = await response.json()
                    print("✅ Workflow Service is healthy")
                    print(f"   Service: {health_data.get('service')}")
                    print(f"   Version: {health_data.get('version')}")
                    print(f"   Database: {'✅' if health_data.get('supabase_initialized') else '❌'}")
                    print(f"   Google FHIR: {'✅' if health_data.get('google_fhir_initialized') else '❌'}")
                else:
                    print(f"❌ Workflow Service unhealthy (Status: {response.status})")
                    return
        except Exception as e:
            print(f"❌ Cannot connect to Workflow Service: {e}")
            return
        
        # 2. Check Federation Gateway
        print("\n2. Checking Federation Gateway...")
        try:
            async with session.get(f"{federation_url}/health", timeout=5) as response:
                if response.status == 200:
                    print("✅ Federation Gateway is healthy")
                else:
                    print(f"❌ Federation Gateway unhealthy (Status: {response.status})")
        except Exception as e:
            print(f"❌ Cannot connect to Federation Gateway: {e}")
            print("   Note: Some tests may fail without federation gateway")
        
        # 3. Test Workflow Definitions Query
        print("\n3. Testing Workflow Definitions Query...")
        workflow_definitions_query = """
        query GetWorkflowDefinitions {
          workflowDefinitions {
            id
            name
            version
            status
            category
            description
          }
        }
        """
        
        payload = {
            "query": workflow_definitions_query
        }
        
        try:
            async with session.post(
                workflow_url + "/api/federation",
                json=payload,
                headers=headers,
                timeout=10
            ) as response:
                result = await response.json()
                
                if "errors" in result:
                    print(f"❌ Query failed: {result['errors']}")
                elif "data" in result and result["data"]["workflowDefinitions"]:
                    definitions = result["data"]["workflowDefinitions"]
                    print(f"✅ Found {len(definitions)} workflow definitions:")
                    for definition in definitions:
                        print(f"   - {definition.get('name')} (ID: {definition.get('id')})")
                        print(f"     Status: {definition.get('status')}, Category: {definition.get('category')}")
                else:
                    print("❌ No workflow definitions found")
        except Exception as e:
            print(f"❌ Workflow definitions query failed: {e}")
        
        # 4. Test Tasks Query
        print("\n4. Testing Tasks Query...")
        tasks_query = """
        query GetTasks($assignee: ID) {
          tasks(assignee: $assignee) {
            id
            description
            priority
            status
            authoredOn
          }
        }
        """
        
        payload = {
            "query": tasks_query,
            "variables": {
                "assignee": "doctor-test-456"
            }
        }
        
        try:
            async with session.post(
                workflow_url + "/api/federation",
                json=payload,
                headers=headers,
                timeout=10
            ) as response:
                result = await response.json()
                
                if "errors" in result:
                    print(f"❌ Tasks query failed: {result['errors']}")
                elif "data" in result:
                    tasks = result["data"]["tasks"] or []
                    print(f"✅ Found {len(tasks)} tasks for user")
                    for task in tasks:
                        print(f"   - {task.get('description', 'No description')}")
                        print(f"     Status: {task.get('status')}, Priority: {task.get('priority')}")
                else:
                    print("❌ No task data returned")
        except Exception as e:
            print(f"❌ Tasks query failed: {e}")
        
        # 5. Test Workflow Instances Query
        print("\n5. Testing Workflow Instances Query...")
        instances_query = """
        query GetWorkflowInstances {
          workflowInstances {
            id
            definitionId
            patientId
            status
            startTime
          }
        }
        """
        
        payload = {
            "query": instances_query
        }
        
        try:
            async with session.post(
                workflow_url + "/api/federation",
                json=payload,
                headers=headers,
                timeout=10
            ) as response:
                result = await response.json()
                
                if "errors" in result:
                    print(f"❌ Workflow instances query failed: {result['errors']}")
                elif "data" in result:
                    instances = result["data"]["workflowInstances"] or []
                    print(f"✅ Found {len(instances)} workflow instances")
                    for instance in instances:
                        print(f"   - Instance ID: {instance.get('id')}")
                        print(f"     Patient: {instance.get('patientId')}, Status: {instance.get('status')}")
                else:
                    print("❌ No workflow instance data returned")
        except Exception as e:
            print(f"❌ Workflow instances query failed: {e}")
        
        # 6. Test Federation Query (if gateway is available)
        print("\n6. Testing Federation Query...")
        federation_query = """
        query GetPatientWithWorkflows($patientId: ID!) {
          patient(id: $patientId) {
            id
            name {
              family
              given
            }
            tasks {
              id
              description
              status
            }
            workflowInstances {
              id
              status
            }
          }
        }
        """
        
        payload = {
            "query": federation_query,
            "variables": {
                "patientId": "patient-test-123"
            }
        }
        
        try:
            async with session.post(
                federation_url,
                json=payload,
                headers=headers,
                timeout=10
            ) as response:
                result = await response.json()
                
                if "errors" in result:
                    print(f"❌ Federation query failed: {result['errors']}")
                elif "data" in result and result["data"]["patient"]:
                    patient = result["data"]["patient"]
                    print(f"✅ Federation query successful!")
                    print(f"   Patient: {patient.get('name', {}).get('given', [''])[0]} {patient.get('name', {}).get('family', '')}")
                    print(f"   Tasks: {len(patient.get('tasks', []))}")
                    print(f"   Workflows: {len(patient.get('workflowInstances', []))}")
                else:
                    print("❌ No patient data returned from federation")
        except Exception as e:
            print(f"❌ Federation query failed: {e}")
    
    print("\n" + "=" * 50)
    print("✅ Basic workflow testing completed!")
    print("\nNext steps:")
    print("1. Start other services (Patient, Observation, Encounter)")
    print("2. Create test data")
    print("3. Run comprehensive integration tests")
    print("4. Test workflow mutations (start workflow, complete tasks)")


if __name__ == "__main__":
    asyncio.run(test_workflow_service())
