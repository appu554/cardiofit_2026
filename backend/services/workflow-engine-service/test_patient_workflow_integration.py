"""
Test script for Patient-Workflow Integration with Vitals and Encounters.

This script demonstrates how to query patient data along with their associated
workflows, tasks, vitals (observations), and encounters using GraphQL Federation.
"""
import asyncio
import json
import aiohttp
from datetime import datetime, timedelta
from typing import Dict, Any, List


class WorkflowPatientTester:
    """Test class for patient workflow integration."""
    
    def __init__(self):
        self.federation_url = "http://localhost:4000"  # Apollo Federation Gateway
        self.workflow_url = "http://localhost:8015"    # Workflow Engine Service
        self.patient_url = "http://localhost:8003"     # Patient Service
        self.observation_url = "http://localhost:8007" # Observation Service
        self.encounter_url = "http://localhost:8020"   # Encounter Service
        
        # Test data
        self.test_patient_id = "patient-test-123"
        self.test_user_id = "doctor-test-456"
        
    async def check_services_health(self) -> Dict[str, bool]:
        """Check if all required services are running."""
        print("🔍 Checking service health...")
        
        services = {
            "Federation Gateway": f"{self.federation_url}/health",
            "Workflow Engine": f"{self.workflow_url}/health",
            "Patient Service": f"{self.patient_url}/health",
            "Observation Service": f"{self.observation_url}/health",
            "Encounter Service": f"{self.encounter_url}/health"
        }
        
        results = {}
        async with aiohttp.ClientSession() as session:
            for service_name, health_url in services.items():
                try:
                    async with session.get(health_url, timeout=5) as response:
                        if response.status == 200:
                            print(f"✅ {service_name}: Healthy")
                            results[service_name] = True
                        else:
                            print(f"❌ {service_name}: Unhealthy (Status: {response.status})")
                            results[service_name] = False
                except Exception as e:
                    print(f"❌ {service_name}: Not accessible ({str(e)})")
                    results[service_name] = False
        
        return results
    
    async def execute_graphql_query(self, query: str, variables: Dict[str, Any] = None) -> Dict[str, Any]:
        """Execute a GraphQL query against the federation gateway."""
        headers = {
            "Content-Type": "application/json",
            "X-User-ID": self.test_user_id,
            "X-User-Role": "doctor",
            "X-User-Roles": "doctor,admin",
            "X-User-Permissions": "patient:read,patient:write,task:read,task:write,workflow:read,workflow:write"
        }
        
        payload = {
            "query": query,
            "variables": variables or {}
        }
        
        async with aiohttp.ClientSession() as session:
            try:
                async with session.post(
                    self.federation_url,
                    json=payload,
                    headers=headers,
                    timeout=30
                ) as response:
                    result = await response.json()
                    return result
            except Exception as e:
                return {"errors": [{"message": f"Request failed: {str(e)}"}]}
    
    def get_basic_patient_query(self) -> str:
        """Get basic patient information query."""
        return """
        query GetBasicPatient($patientId: ID!) {
          patient(id: $patientId) {
            id
            resourceType
            name {
              family
              given
              use
            }
            gender
            birthDate
            active
            telecom {
              system
              value
              use
            }
          }
        }
        """
    
    def get_patient_with_workflows_query(self) -> str:
        """Get patient with associated workflows and tasks."""
        return """
        query GetPatientWithWorkflows($patientId: ID!) {
          patient(id: $patientId) {
            id
            name {
              family
              given
            }
            gender
            birthDate
            active
            
            # Workflow-related data
            tasks(status: READY) {
              id
              description
              priority
              status
              authoredOn
              lastModified
              for {
                reference
                display
              }
              owner {
                reference
                display
              }
            }
            
            workflowInstances(status: ACTIVE) {
              id
              definitionId
              status
              startTime
              endTime
              createdBy
            }
          }
        }
        """
    
    def get_comprehensive_patient_query(self) -> str:
        """Get comprehensive patient data with workflows, vitals, and encounters."""
        return """
        query GetComprehensivePatient($patientId: ID!) {
          patient(id: $patientId) {
            id
            name {
              family
              given
            }
            gender
            birthDate
            active
            
            # Encounters
            encounters {
              id
              status
              class {
                code
                display
              }
              type {
                coding {
                  code
                  display
                  system
                }
              }
              period {
                start
                end
              }
              location {
                location {
                  reference
                  display
                }
              }
            }
            
            # Observations (Vitals)
            observations {
              id
              status
              category {
                coding {
                  code
                  display
                  system
                }
              }
              code {
                coding {
                  code
                  display
                  system
                }
              }
              valueQuantity {
                value
                unit
                system
              }
              effectiveDateTime
              issued
            }
            
            # Workflow Tasks
            tasks {
              id
              description
              priority
              status
              authoredOn
              lastModified
              businessStatus {
                coding {
                  code
                  display
                }
              }
              for {
                reference
                display
              }
              owner {
                reference
                display
              }
            }
            
            # Active Workflows
            workflowInstances(status: ACTIVE) {
              id
              definitionId
              status
              startTime
              endTime
              createdBy
            }
          }
        }
        """
    
    def get_workflow_definitions_query(self) -> str:
        """Get available workflow definitions."""
        return """
        query GetWorkflowDefinitions {
          workflowDefinitions {
            id
            name
            version
            status
            category
            description
            createdAt
            updatedAt
          }
        }
        """
    
    def get_start_workflow_mutation(self) -> str:
        """Get mutation to start a workflow."""
        return """
        mutation StartPatientWorkflow($definitionId: ID!, $patientId: ID!, $variables: [KeyValuePairInput]) {
          startWorkflow(
            definitionId: $definitionId
            patientId: $patientId
            initialVariables: $variables
          ) {
            id
            definitionId
            patientId
            status
            startTime
            createdBy
          }
        }
        """

    async def test_basic_patient_query(self) -> bool:
        """Test basic patient information retrieval."""
        print("\n🧪 Testing Basic Patient Query...")

        query = self.get_basic_patient_query()
        variables = {"patientId": self.test_patient_id}

        result = await self.execute_graphql_query(query, variables)

        if "errors" in result:
            print(f"❌ Query failed: {result['errors']}")
            return False

        if "data" in result and result["data"]["patient"]:
            patient = result["data"]["patient"]
            print(f"✅ Patient found: {patient.get('name', {}).get('given', [''])[0]} {patient.get('name', {}).get('family', '')}")
            print(f"   ID: {patient.get('id')}")
            print(f"   Gender: {patient.get('gender')}")
            print(f"   Birth Date: {patient.get('birthDate')}")
            return True
        else:
            print("❌ No patient data returned")
            return False

    async def test_workflow_definitions(self) -> bool:
        """Test workflow definitions retrieval."""
        print("\n🧪 Testing Workflow Definitions Query...")

        query = self.get_workflow_definitions_query()
        result = await self.execute_graphql_query(query)

        if "errors" in result:
            print(f"❌ Query failed: {result['errors']}")
            return False

        if "data" in result and result["data"]["workflowDefinitions"]:
            definitions = result["data"]["workflowDefinitions"]
            print(f"✅ Found {len(definitions)} workflow definitions:")
            for definition in definitions:
                print(f"   - {definition.get('name')} (ID: {definition.get('id')}, Status: {definition.get('status')})")
            return True
        else:
            print("❌ No workflow definitions found")
            return False

    async def test_patient_with_workflows(self) -> bool:
        """Test patient query with workflow data."""
        print("\n🧪 Testing Patient with Workflows Query...")

        query = self.get_patient_with_workflows_query()
        variables = {"patientId": self.test_patient_id}

        result = await self.execute_graphql_query(query, variables)

        if "errors" in result:
            print(f"❌ Query failed: {result['errors']}")
            return False

        if "data" in result and result["data"]["patient"]:
            patient = result["data"]["patient"]
            print(f"✅ Patient: {patient.get('name', {}).get('given', [''])[0]} {patient.get('name', {}).get('family', '')}")

            # Check tasks
            tasks = patient.get("tasks", [])
            print(f"   📋 Tasks: {len(tasks)}")
            for task in tasks:
                print(f"      - {task.get('description', 'No description')} (Status: {task.get('status')}, Priority: {task.get('priority')})")

            # Check workflow instances
            workflows = patient.get("workflowInstances", [])
            print(f"   🔄 Active Workflows: {len(workflows)}")
            for workflow in workflows:
                print(f"      - Definition ID: {workflow.get('definitionId')} (Status: {workflow.get('status')})")

            return True
        else:
            print("❌ No patient data returned")
            return False

    async def test_comprehensive_patient_query(self) -> bool:
        """Test comprehensive patient query with all related data."""
        print("\n🧪 Testing Comprehensive Patient Query...")

        query = self.get_comprehensive_patient_query()
        variables = {"patientId": self.test_patient_id}

        result = await self.execute_graphql_query(query, variables)

        if "errors" in result:
            print(f"❌ Query failed: {result['errors']}")
            return False

        if "data" in result and result["data"]["patient"]:
            patient = result["data"]["patient"]
            print(f"✅ Patient: {patient.get('name', {}).get('given', [''])[0]} {patient.get('name', {}).get('family', '')}")

            # Check encounters
            encounters = patient.get("encounters", [])
            print(f"   🏥 Encounters: {len(encounters)}")
            for encounter in encounters:
                print(f"      - Status: {encounter.get('status')}, Class: {encounter.get('class', {}).get('display', 'Unknown')}")

            # Check observations (vitals)
            observations = patient.get("observations", [])
            print(f"   📊 Observations (Vitals): {len(observations)}")
            for obs in observations:
                code_display = obs.get('code', {}).get('coding', [{}])[0].get('display', 'Unknown')
                value = obs.get('valueQuantity', {})
                print(f"      - {code_display}: {value.get('value', 'N/A')} {value.get('unit', '')}")

            # Check tasks
            tasks = patient.get("tasks", [])
            print(f"   📋 Tasks: {len(tasks)}")
            for task in tasks:
                print(f"      - {task.get('description', 'No description')} (Status: {task.get('status')})")

            # Check workflow instances
            workflows = patient.get("workflowInstances", [])
            print(f"   🔄 Active Workflows: {len(workflows)}")
            for workflow in workflows:
                print(f"      - Definition ID: {workflow.get('definitionId')} (Status: {workflow.get('status')})")

            return True
        else:
            print("❌ No patient data returned")
            return False

    async def test_start_workflow(self) -> bool:
        """Test starting a workflow for a patient."""
        print("\n🧪 Testing Start Workflow Mutation...")

        # First, get available workflow definitions
        definitions_query = self.get_workflow_definitions_query()
        definitions_result = await self.execute_graphql_query(definitions_query)

        if "errors" in definitions_result or not definitions_result.get("data", {}).get("workflowDefinitions"):
            print("❌ No workflow definitions available to start")
            return False

        # Use the first available workflow definition
        definition = definitions_result["data"]["workflowDefinitions"][0]
        definition_id = definition["id"]

        print(f"   Starting workflow: {definition['name']} (ID: {definition_id})")

        # Start the workflow
        mutation = self.get_start_workflow_mutation()
        variables = {
            "definitionId": definition_id,
            "patientId": self.test_patient_id,
            "variables": [
                {"key": "patientName", "value": "John Doe Test"},
                {"key": "admissionType", "value": "emergency"},
                {"key": "priority", "value": "high"},
                {"key": "assignee", "value": self.test_user_id}
            ]
        }

        result = await self.execute_graphql_query(mutation, variables)

        if "errors" in result:
            print(f"❌ Workflow start failed: {result['errors']}")
            return False

        if "data" in result and result["data"]["startWorkflow"]:
            workflow = result["data"]["startWorkflow"]
            print(f"✅ Workflow started successfully!")
            print(f"   Instance ID: {workflow.get('id')}")
            print(f"   Status: {workflow.get('status')}")
            print(f"   Start Time: {workflow.get('startTime')}")
            return True
        else:
            print("❌ No workflow instance data returned")
            return False

    async def run_all_tests(self):
        """Run all integration tests."""
        print("=" * 80)
        print("PATIENT-WORKFLOW INTEGRATION TESTS")
        print("=" * 80)

        # Check service health
        health_results = await self.check_services_health()

        # Count healthy services
        healthy_services = sum(1 for status in health_results.values() if status)
        total_services = len(health_results)

        print(f"\n📊 Service Health Summary: {healthy_services}/{total_services} services healthy")

        if healthy_services < 2:  # At least federation gateway and workflow service
            print("❌ Insufficient services running. Please start required services.")
            return

        # Run tests
        test_results = []

        # Test 1: Basic patient query
        try:
            result = await self.test_basic_patient_query()
            test_results.append(("Basic Patient Query", result))
        except Exception as e:
            print(f"❌ Basic Patient Query failed with exception: {e}")
            test_results.append(("Basic Patient Query", False))

        # Test 2: Workflow definitions
        try:
            result = await self.test_workflow_definitions()
            test_results.append(("Workflow Definitions", result))
        except Exception as e:
            print(f"❌ Workflow Definitions test failed with exception: {e}")
            test_results.append(("Workflow Definitions", False))

        # Test 3: Patient with workflows
        try:
            result = await self.test_patient_with_workflows()
            test_results.append(("Patient with Workflows", result))
        except Exception as e:
            print(f"❌ Patient with Workflows test failed with exception: {e}")
            test_results.append(("Patient with Workflows", False))

        # Test 4: Comprehensive patient query
        try:
            result = await self.test_comprehensive_patient_query()
            test_results.append(("Comprehensive Patient Query", result))
        except Exception as e:
            print(f"❌ Comprehensive Patient Query failed with exception: {e}")
            test_results.append(("Comprehensive Patient Query", False))

        # Test 5: Start workflow
        try:
            result = await self.test_start_workflow()
            test_results.append(("Start Workflow", result))
        except Exception as e:
            print(f"❌ Start Workflow test failed with exception: {e}")
            test_results.append(("Start Workflow", False))

        # Summary
        print("\n" + "=" * 80)
        print("TEST RESULTS SUMMARY")
        print("=" * 80)

        passed_tests = 0
        for test_name, result in test_results:
            status = "✅ PASSED" if result else "❌ FAILED"
            print(f"{status}: {test_name}")
            if result:
                passed_tests += 1

        print(f"\n📊 Overall Results: {passed_tests}/{len(test_results)} tests passed")

        if passed_tests == len(test_results):
            print("🎉 All tests passed! Patient-Workflow integration is working correctly.")
        else:
            print("⚠️  Some tests failed. Check the output above for details.")


async def main():
    """Main execution function."""
    tester = WorkflowPatientTester()
    await tester.run_all_tests()


if __name__ == "__main__":
    asyncio.run(main())
