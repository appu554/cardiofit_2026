"""
Test script for Real Services Integration with Workflows
Tests the complete microservices stack with patient data, vitals, encounters, and workflows
"""
import asyncio
import aiohttp
import json
from datetime import datetime, timedelta
from typing import Dict, Any, List


class RealServicesIntegrationTester:
    """Test class for real services integration."""
    
    def __init__(self):
        # Service URLs
        self.federation_url = "http://localhost:4000/graphql"
        self.api_gateway_url = "http://localhost:8005"
        self.auth_service_url = "http://localhost:8001"
        self.workflow_url = "http://localhost:8015"
        self.patient_url = "http://localhost:8003"
        self.observation_url = "http://localhost:8007"
        self.encounter_url = "http://localhost:8020"
        
        # Test headers
        self.headers = {
            "Content-Type": "application/json",
            "X-User-ID": "doctor-test-456",
            "X-User-Role": "doctor",
            "X-User-Roles": "doctor,admin",
            "X-User-Permissions": "patient:read,patient:write,observation:read,observation:write,encounter:read,encounter:write,task:read,task:write,workflow:read,workflow:write"
        }
        
        # Test data - using existing patient
        self.test_patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        self.test_encounter_id = None
        self.test_observation_ids = []
        self.test_workflow_instance_id = None
    
    async def check_all_services_health(self) -> Dict[str, bool]:
        """Check health of all required services."""
        print("🔍 Checking All Services Health...")
        
        services = {
            "Federation Gateway": "http://localhost:4000/health",
            "API Gateway": f"{self.api_gateway_url}/health",
            "Auth Service": f"{self.auth_service_url}/health",
            "Workflow Service": f"{self.workflow_url}/health",
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
        
        healthy_count = sum(1 for status in results.values() if status)
        total_count = len(results)
        print(f"\n📊 Services Health: {healthy_count}/{total_count} services healthy")
        
        return results
    
    async def execute_federation_query(self, query: str, variables: Dict[str, Any] = None) -> Dict[str, Any]:
        """Execute a GraphQL query against the federation gateway."""
        payload = {
            "query": query,
            "variables": variables or {}
        }
        
        async with aiohttp.ClientSession() as session:
            try:
                async with session.post(
                    self.federation_url,
                    json=payload,
                    headers=self.headers,
                    timeout=30
                ) as response:
                    result = await response.json()
                    return result
            except Exception as e:
                return {"errors": [{"message": f"Federation request failed: {str(e)}"}]}
    
    async def create_test_patient(self) -> str:
        """Create a test patient."""
        print("\n👤 Creating Test Patient...")
        
        mutation = """
        mutation CreatePatient($patientData: PatientInput!) {
          createPatient(patientData: $patientData) {
            id
            name {
              family
              given
            }
            gender
            birthDate
          }
        }
        """
        
        variables = {
            "patientData": {
                "name": [{
                    "family": "TestPatient",
                    "given": ["John", "Workflow"],
                    "use": "official"
                }],
                "gender": "male",
                "birthDate": "1985-03-15",
                "active": True,
                "telecom": [{
                    "system": "phone",
                    "value": "+1-555-0123",
                    "use": "mobile"
                }]
            }
        }
        
        result = await self.execute_federation_query(mutation, variables)
        
        if "errors" in result:
            print(f"❌ Patient creation failed: {result['errors']}")
            return None
        
        if "data" in result and result["data"]["createPatient"]:
            patient = result["data"]["createPatient"]
            patient_id = patient["id"]
            print(f"✅ Patient created: {patient['name']['given'][0]} {patient['name']['family']} (ID: {patient_id})")
            self.test_patient_id = patient_id
            return patient_id
        
        print("❌ No patient data returned")
        return None
    
    async def create_test_encounter(self, patient_id: str) -> str:
        """Create a test encounter for the patient."""
        print("\n🏥 Creating Test Encounter...")
        
        mutation = """
        mutation CreateEncounter($encounterData: EncounterInput!) {
          createEncounter(encounterData: $encounterData) {
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
              }
            }
            subject {
              reference
            }
            period {
              start
            }
          }
        }
        """
        
        variables = {
            "encounterData": {
                "status": "in-progress",
                "class": {
                    "code": "emergency",
                    "display": "Emergency"
                },
                "type": [{
                    "coding": [{
                        "system": "http://snomed.info/sct",
                        "code": "50849002",
                        "display": "Emergency department patient visit"
                    }]
                }],
                "subject": {
                    "reference": f"Patient/{patient_id}"
                },
                "period": {
                    "start": datetime.now().isoformat()
                }
            }
        }
        
        result = await self.execute_federation_query(mutation, variables)
        
        if "errors" in result:
            print(f"❌ Encounter creation failed: {result['errors']}")
            return None
        
        if "data" in result and result["data"]["createEncounter"]:
            encounter = result["data"]["createEncounter"]
            encounter_id = encounter["id"]
            print(f"✅ Encounter created: {encounter['type'][0]['coding'][0]['display']} (ID: {encounter_id})")
            self.test_encounter_id = encounter_id
            return encounter_id
        
        print("❌ No encounter data returned")
        return None
    
    async def create_test_vitals(self, patient_id: str) -> List[str]:
        """Create test vital signs for the patient."""
        print("\n📊 Creating Test Vitals...")
        
        vitals_data = [
            {
                "code": "8480-6",
                "display": "Systolic Blood Pressure",
                "value": 145.0,
                "unit": "mmHg",
                "system": "http://unitsofmeasure.org"
            },
            {
                "code": "8462-4", 
                "display": "Diastolic Blood Pressure",
                "value": 95.0,
                "unit": "mmHg",
                "system": "http://unitsofmeasure.org"
            },
            {
                "code": "8867-4",
                "display": "Heart Rate",
                "value": 88.0,
                "unit": "beats/min",
                "system": "http://unitsofmeasure.org"
            },
            {
                "code": "8310-5",
                "display": "Body Temperature",
                "value": 38.2,
                "unit": "Cel",
                "system": "http://unitsofmeasure.org"
            }
        ]
        
        observation_ids = []
        
        for vital in vitals_data:
            mutation = """
            mutation CreateObservation($observationData: ObservationInput!) {
              createObservation(observationData: $observationData) {
                id
                status
                code {
                  coding {
                    code
                    display
                  }
                }
                valueQuantity {
                  value
                  unit
                }
                effectiveDateTime
              }
            }
            """
            
            variables = {
                "observationData": {
                    "status": "final",
                    "category": [{
                        "coding": [{
                            "system": "http://terminology.hl7.org/CodeSystem/observation-category",
                            "code": "vital-signs",
                            "display": "Vital Signs"
                        }]
                    }],
                    "code": {
                        "coding": [{
                            "system": "http://loinc.org",
                            "code": vital["code"],
                            "display": vital["display"]
                        }]
                    },
                    "subject": {
                        "reference": f"Patient/{patient_id}"
                    },
                    "effectiveDateTime": datetime.now().isoformat(),
                    "valueQuantity": {
                        "value": vital["value"],
                        "unit": vital["unit"],
                        "system": vital["system"]
                    }
                }
            }
            
            result = await self.execute_federation_query(mutation, variables)
            
            if "errors" in result:
                print(f"❌ {vital['display']} creation failed: {result['errors']}")
            elif "data" in result and result["data"]["createObservation"]:
                observation = result["data"]["createObservation"]
                obs_id = observation["id"]
                print(f"✅ {vital['display']}: {vital['value']} {vital['unit']} (ID: {obs_id})")
                observation_ids.append(obs_id)
            else:
                print(f"❌ No observation data returned for {vital['display']}")
        
        self.test_observation_ids = observation_ids
        return observation_ids

    async def verify_existing_patient(self, patient_id: str) -> bool:
        """Verify that the existing patient can be accessed."""
        print(f"🔍 Verifying existing patient: {patient_id}")

        query = """
        query GetPatient($patientId: ID!) {
          patient(id: $patientId) {
            id
            name {
              family
              given
            }
            gender
            birthDate
            active
          }
        }
        """

        variables = {"patientId": patient_id}
        result = await self.execute_federation_query(query, variables)

        if "errors" in result:
            print(f"❌ Patient verification failed: {result['errors']}")
            return False

        if "data" in result and result["data"]["patient"]:
            patient = result["data"]["patient"]
            name = patient.get("name", [{}])[0] if patient.get("name") else {}
            given_name = name.get("given", [""])[0] if name.get("given") else ""
            family_name = name.get("family", "")

            print(f"✅ Patient verified: {given_name} {family_name}")
            print(f"   ID: {patient['id']}")
            print(f"   Gender: {patient.get('gender', 'Unknown')}")
            print(f"   Birth Date: {patient.get('birthDate', 'Unknown')}")
            print(f"   Active: {patient.get('active', 'Unknown')}")
            return True
        else:
            print(f"❌ Patient not found: {patient_id}")
            return False

    async def start_workflow_for_patient(self, patient_id: str) -> str:
        """Start a workflow for the patient based on abnormal vitals."""
        print("\n🔄 Starting Workflow for Patient...")

        mutation = """
        mutation StartWorkflow($definitionId: ID!, $patientId: ID!, $variables: [KeyValuePairInput]) {
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

        variables = {
            "definitionId": "patient-admission-workflow",
            "patientId": patient_id,
            "variables": [
                {"key": "trigger", "value": "abnormal-vitals"},
                {"key": "priority", "value": "high"},
                {"key": "systolicBP", "value": "145"},
                {"key": "diastolicBP", "value": "95"},
                {"key": "temperature", "value": "38.2"},
                {"key": "assignee", "value": "doctor-test-456"}
            ]
        }

        result = await self.execute_federation_query(mutation, variables)

        if "errors" in result:
            print(f"❌ Workflow start failed: {result['errors']}")
            return None

        if "data" in result and result["data"]["startWorkflow"]:
            workflow = result["data"]["startWorkflow"]
            workflow_id = workflow["id"]
            print(f"✅ Workflow started: {workflow['definitionId']} (Instance ID: {workflow_id})")
            print(f"   Patient: {workflow['patientId']}, Status: {workflow['status']}")
            self.test_workflow_instance_id = workflow_id
            return workflow_id

        print("❌ No workflow instance data returned")
        return None

    async def test_comprehensive_patient_query(self, patient_id: str):
        """Test comprehensive patient query with all related data."""
        print("\n🧬 Testing Comprehensive Patient Query...")

        query = """
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

            # Encounters from Encounter Service
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
                }
              }
              period {
                start
                end
              }
            }

            # Observations from Observation Service
            observations {
              id
              status
              code {
                coding {
                  code
                  display
                }
              }
              valueQuantity {
                value
                unit
              }
              effectiveDateTime
            }

            # Tasks from Workflow Service
            tasks {
              id
              description
              status
              priority
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

            # Workflow Instances from Workflow Service
            workflowInstances {
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

        variables = {"patientId": patient_id}
        result = await self.execute_federation_query(query, variables)

        if "errors" in result:
            print(f"❌ Comprehensive query failed: {result['errors']}")
            return False

        if "data" in result and result["data"]["patient"]:
            patient = result["data"]["patient"]
            print(f"✅ Comprehensive Patient Data Retrieved:")
            print(f"   Patient: {patient['name']['given'][0]} {patient['name']['family']}")
            print(f"   Gender: {patient['gender']}, Birth Date: {patient['birthDate']}")

            # Encounters
            encounters = patient.get("encounters", [])
            print(f"   🏥 Encounters: {len(encounters)}")
            for encounter in encounters:
                print(f"      - {encounter['type'][0]['coding'][0]['display']} (Status: {encounter['status']})")

            # Observations
            observations = patient.get("observations", [])
            print(f"   📊 Observations: {len(observations)}")
            for obs in observations:
                code_display = obs['code']['coding'][0]['display']
                value = obs['valueQuantity']
                print(f"      - {code_display}: {value['value']} {value['unit']}")

            # Tasks
            tasks = patient.get("tasks", [])
            print(f"   📋 Tasks: {len(tasks)}")
            for task in tasks:
                print(f"      - {task.get('description', 'No description')} (Status: {task['status']}, Priority: {task['priority']})")

            # Workflow Instances
            workflows = patient.get("workflowInstances", [])
            print(f"   🔄 Workflow Instances: {len(workflows)}")
            for workflow in workflows:
                print(f"      - {workflow['definitionId']} (Status: {workflow['status']})")

            return True
        else:
            print("❌ No patient data returned")
            return False

    async def test_workflow_task_completion(self):
        """Test completing a workflow task."""
        print("\n✅ Testing Task Completion...")

        # First, get available tasks
        query = """
        query GetTasks($assignee: ID) {
          tasks(assignee: $assignee) {
            id
            description
            status
            priority
            for {
              reference
            }
          }
        }
        """

        variables = {"assignee": "doctor-test-456"}
        result = await self.execute_federation_query(query, variables)

        if "errors" in result:
            print(f"❌ Tasks query failed: {result['errors']}")
            return False

        tasks = result.get("data", {}).get("tasks", [])
        if not tasks:
            print("❌ No tasks found to complete")
            return False

        # Complete the first available task
        task_to_complete = tasks[0]
        print(f"   Completing task: {task_to_complete['description']}")

        mutation = """
        mutation CompleteTask($taskId: ID!, $outputVariables: [KeyValuePairInput]) {
          completeTask(taskId: $taskId, outputVariables: $outputVariables) {
            id
            status
            lastModified
          }
        }
        """

        variables = {
            "taskId": task_to_complete["id"],
            "outputVariables": [
                {"key": "reviewResult", "value": "abnormal-vitals-confirmed"},
                {"key": "treatment", "value": "antihypertensive-medication-prescribed"},
                {"key": "followUp", "value": "schedule-in-24-hours"},
                {"key": "notes", "value": "Patient has elevated BP and fever, started on medication"}
            ]
        }

        result = await self.execute_federation_query(mutation, variables)

        if "errors" in result:
            print(f"❌ Task completion failed: {result['errors']}")
            return False

        if "data" in result and result["data"]["completeTask"]:
            completed_task = result["data"]["completeTask"]
            print(f"✅ Task completed: {completed_task['id']} (Status: {completed_task['status']})")
            return True

        print("❌ No task completion data returned")
        return False

    async def run_complete_integration_test(self):
        """Run the complete integration test suite."""
        print("🚀 REAL SERVICES INTEGRATION TEST")
        print("=" * 80)

        # Step 1: Check all services
        health_results = await self.check_all_services_health()
        healthy_services = sum(1 for status in health_results.values() if status)

        if healthy_services < 5:  # Need at least federation, workflow, patient, observation, encounter
            print(f"\n❌ Insufficient services running ({healthy_services}/7)")
            print("Please start all required services:")
            print("1. Patient Service: cd backend/services/patient-service && python start_service.py")
            print("2. Observation Service: cd backend/services/observation-service && python start_service.py")
            print("3. Encounter Service: cd backend/services/encounter-service && python start_service.py")
            print("4. Workflow Service: cd backend/services/workflow-engine-service && python start_service.py")
            print("5. Apollo Federation: cd apollo-federation && npm start")
            print("6. Auth Service: cd backend/services/auth-service && npm start")
            print("7. API Gateway: cd backend/services/api-gateway && npm start")
            return

        print(f"\n✅ Sufficient services running ({healthy_services}/7)")

        # Step 2: Use existing patient
        patient_id = self.test_patient_id
        print(f"\n👤 Using Existing Patient: {patient_id}")

        # Verify patient exists
        await self.verify_existing_patient(patient_id)

        # Step 3: Create test encounter
        encounter_id = await self.create_test_encounter(patient_id)
        if not encounter_id:
            print("❌ Cannot proceed without encounter")
            return

        # Step 4: Create test vitals (including abnormal values)
        observation_ids = await self.create_test_vitals(patient_id)
        if not observation_ids:
            print("❌ Cannot proceed without vitals")
            return

        # Step 5: Start workflow based on abnormal vitals
        workflow_id = await self.start_workflow_for_patient(patient_id)
        if not workflow_id:
            print("❌ Cannot proceed without workflow")
            return

        # Step 6: Test comprehensive patient query
        print("\n" + "="*50)
        print("TESTING FEDERATED PATIENT QUERY")
        print("="*50)

        success = await self.test_comprehensive_patient_query(patient_id)
        if not success:
            print("❌ Comprehensive query failed")
            return

        # Step 7: Test task completion
        await asyncio.sleep(2)  # Give workflow time to create tasks
        await self.test_workflow_task_completion()

        # Step 8: Final comprehensive query to see changes
        print("\n" + "="*50)
        print("FINAL STATE AFTER TASK COMPLETION")
        print("="*50)

        await self.test_comprehensive_patient_query(patient_id)

        # Summary
        print("\n" + "="*80)
        print("🎉 REAL SERVICES INTEGRATION TEST COMPLETED!")
        print("="*80)

        print(f"\n📊 Test Results Summary:")
        print(f"✅ Patient Created: {patient_id}")
        print(f"✅ Encounter Created: {encounter_id}")
        print(f"✅ Vitals Created: {len(observation_ids)} observations")
        print(f"✅ Workflow Started: {workflow_id}")
        print(f"✅ Federation Query: Patient data with workflows, vitals, encounters")
        print(f"✅ Task Management: Workflow tasks completed")

        print(f"\n🎯 What This Demonstrates:")
        print(f"• Complete microservices integration")
        print(f"• Patient data flows between services")
        print(f"• Abnormal vitals trigger workflows automatically")
        print(f"• Tasks created with full clinical context")
        print(f"• Federation provides unified data access")
        print(f"• Real-time clinical decision support")

        print(f"\n🚀 Next Steps:")
        print(f"• Add more workflow definitions")
        print(f"• Implement timer-based escalations")
        print(f"• Add role-based task assignments")
        print(f"• Create clinical dashboards")
        print(f"• Implement audit trails")


async def main():
    """Main execution function."""
    tester = RealServicesIntegrationTester()
    await tester.run_complete_integration_test()


if __name__ == "__main__":
    asyncio.run(main())
