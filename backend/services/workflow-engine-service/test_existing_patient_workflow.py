"""
Test workflow integration with existing patient
"""
import asyncio
import aiohttp
import json
from datetime import datetime

class ExistingPatientWorkflowTester:
    """Test workflow integration with existing patient."""
    
    def __init__(self):
        # Direct workflow engine service URL
        self.workflow_engine_url = "http://localhost:8015"
        self.existing_patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
        
        # Headers for direct workflow engine API
        self.headers = {
            "Content-Type": "application/json"
        }
    
    async def execute_workflow_request(self, endpoint: str, payload: dict = None, method: str = "POST") -> dict:
        """Execute a request to the workflow engine service.
        
        Args:
            endpoint: API endpoint (without base URL)
            payload: Request payload for POST/PUT requests
            method: HTTP method (GET, POST, PUT, DELETE)
            
        Returns:
            Response data as a dictionary
        """
        url = f"{self.workflow_engine_url}/{endpoint}"
        
        try:
            async with aiohttp.ClientSession() as session:
                # Print request details
                print(f"\n🔍 Sending {method.upper()} request to {url}")
                if payload and method.upper() in ["POST", "PUT", "PATCH"]:
                    print("Payload:", json.dumps(payload, indent=2))
                
                # Make the request
                if method.upper() == "GET":
                    async with session.get(url, headers=self.headers) as response:
                        return await self._handle_response(response)
                elif method.upper() == "POST":
                    async with session.post(url, json=payload, headers=self.headers) as response:
                        return await self._handle_response(response)
                elif method.upper() == "PUT":
                    async with session.put(url, json=payload, headers=self.headers) as response:
                        return await self._handle_response(response)
                elif method.upper() == "DELETE":
                    async with session.delete(url, headers=self.headers) as response:
                        return await self._handle_response(response)
                else:
                    raise ValueError(f"Unsupported HTTP method: {method}")
                    
        except Exception as e:
            error_msg = f"Error executing {method} request to {url}: {str(e)}"
            print(f"\n❌ {error_msg}")
            import traceback
            traceback.print_exc()
            return {"error": error_msg, "success": False}
            
    async def _handle_response(self, response) -> dict:
        """Handle HTTP response and return parsed JSON."""
        try:
            response_text = await response.text()
            if response_text:
                response_data = json.loads(response_text)
            else:
                response_data = {}
                
            print(f"\n📥 Response (Status: {response.status}):")
            print(json.dumps(response_data, indent=2))
            
            if not response.ok:
                response_data["error"] = f"HTTP {response.status}: {response.reason}"
                response_data["success"] = False
            else:
                response_data["success"] = True
                
            return response_data
            
        except json.JSONDecodeError:
            return {
                "success": False,
                "error": f"Invalid JSON response: {response_text}"
            }
    
    async def get_workflow_instance(self, instance_id: int):
        """Get workflow instance details."""
        endpoint = f"workflow/instance/{instance_id}"
        return await self.execute_workflow_request(endpoint, method="GET")

    async def list_workflow_tasks(self, instance_id: int):
        """List tasks for a workflow instance."""
        endpoint = f"workflow/instance/{instance_id}/tasks"
        return await self.execute_workflow_request(endpoint, method="GET")

    async def wait_for_workflow_completion(self, instance_id: int, max_attempts: int = 30, delay_seconds: int = 2):
        """Wait for workflow to complete or reach max attempts."""
        import asyncio
        
        for attempt in range(max_attempts):
            try:
                # Get workflow instance details
                result = await self.get_workflow_instance(instance_id)
                
                if not result.get("success"):
                    error_msg = result.get('error', 'Unknown error')
                    print(f"❌ Error getting workflow status: {error_msg}")
                    return False
                    
                instance = result.get("workflow_instance", {})
                status = instance.get("status")
                
                print(f"\n⏳ Workflow Status Check {attempt + 1}/{max_attempts}")
                print(f"   Status: {status}")
                print(f"   Current State: {instance.get('current_state', 'N/A')}")
                print(f"   Definition: {instance.get('definition_name', 'N/A')} (ID: {instance.get('definition_id', 'N/A')})")
                
                # Check if workflow has completed
                if status in ["completed", "failed", "terminated"]:
                    print(f"\n✅ Workflow {status.upper()}!")
                    print(f"   Started at: {instance.get('start_time')}")
                    print(f"   Ended at: {instance.get('end_time')}")
                    print(f"   Variables: {json.dumps(instance.get('variables', {}), indent=2)}")
                    return True
                    
                # Check for active tasks
                tasks_result = await self.list_workflow_tasks(instance_id)
                if tasks_result.get("success"):
                    tasks = tasks_result.get("tasks", [])
                    if tasks:
                        print("   Active Tasks:")
                        for task in tasks:
                            print(f"     - {task.get('name')} ({task.get('task_definition_key')}): {task.get('assignee')} - {task.get('status')}")
                    else:
                        print("   No active tasks found")
                
                # If we're not at the last attempt, wait before next check
                if attempt < max_attempts - 1:
                    await asyncio.sleep(delay_seconds)
                    
            except Exception as e:
                print(f"❌ Error during workflow monitoring: {str(e)}")
                if attempt < max_attempts - 1:
                    await asyncio.sleep(delay_seconds)  # Wait before retry
                else:
                    raise
                    
        print("\n❌ Max attempts reached. Workflow did not complete.")
        return False

    async def test_workflow_start_and_monitor(self):
        """Test starting and monitoring a workflow."""
        print("\n🚀 Testing Workflow Start and Monitoring")
        print("=" * 50)
        
        # Start the workflow
        workflow_payload = {
            "workflow_key": "1",  # Using "1" as the workflow definition ID for testing
            "variables": {
                "patient_id": self.existing_patient_id,
                "initial_variables": {
                    "trigger": "manual-test",
                    "priority": "normal"
                },
                "context": {
                    "source": "test"
                },
                "created_by": "test_script"
            }
        }
        
        start_result = await self.execute_workflow_request("workflow/start", workflow_payload)
        
        if "error" in start_result:
            print(f"❌ Failed to start workflow: {start_result['error']}")
            return False
        
        if not start_result.get("success") or "workflow_instance" not in start_result:
            print(f"❌ Failed to start workflow: {start_result}")
            return False
        
        workflow_instance = start_result["workflow_instance"]
        instance_id = workflow_instance["id"]
        
        print(f"✅ Successfully started workflow instance: {instance_id}")
        print(f"   External ID: {workflow_instance['external_id']}")
        print(f"   Status: {workflow_instance['status']}")
        print(f"   Started at: {workflow_instance['start_time']}")
        
        # Monitor workflow completion
        print("\n🔍 Monitoring workflow execution...")
        completed = await self.wait_for_workflow_completion(instance_id)
        
        if not completed:
            print("❌ Workflow did not complete successfully")
            return False
            
        # Get final workflow state
        final_state = await self.get_workflow_instance(instance_id)
        if final_state.get("success"):
            print("\n📊 Final Workflow State:")
            instance = final_state["workflow_instance"]
            print(f"Status: {instance['status']}")
            print(f"Variables: {instance.get('variables', {})}")
            print(f"Context: {instance.get('context', {})}")
            
            # Add any assertions about the final state here
            # Example:
            # assert instance['status'] == 'completed', "Workflow did not complete successfully"
            # assert 'result' in instance.get('variables', {}), "Expected result not found in workflow variables"
            
        return completed
    
    async def test_start_workflow(self):
        """Test starting a workflow for the patient."""
        print("🚀 Testing Workflow Start")
        print("=" * 50)
        
        # For testing, we'll use a known workflow definition ID
        workflow_payload = {
            "workflow_key": "1",  # Using "1" as the workflow definition ID for testing
            "variables": {
                "patient_id": self.existing_patient_id,
                "initial_variables": {
                    "trigger": "manual-test",
                    "priority": "normal"
                },
                "context": {
                    "source": "test"
                },
                "created_by": "test_script"
            }
        }
        
        result = await self.execute_workflow_request("workflow/start", workflow_payload)
        
        if "error" in result:
            print(f"❌ Failed to start workflow: {result['error']}")
            return False
        
        if result.get("success") and "workflow_instance" in result:
            workflow_instance = result["workflow_instance"]
            print(f"✅ Successfully started workflow instance: {workflow_instance['id']}")
            print(f"   External ID: {workflow_instance['external_id']}")
            print(f"   Status: {workflow_instance['status']}")
            print(f"   Started at: {workflow_instance['start_time']}")
            return True
        else:
            print(f"❌ Failed to start workflow: {result}")
            return False
    
    async def test_patient_with_existing_data(self):
        """Test comprehensive patient query with existing encounters, observations, and workflows."""
        print("\n🧬 Testing Comprehensive Patient Data")
        print("=" * 50)
        
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
          }
        }
        """
        
        variables = {"patientId": self.existing_patient_id}
        result = await self.execute_query(query, variables)
        
        if "errors" in result:
            print(f"❌ Comprehensive query failed: {result['errors']}")
            return False
        
        if "data" in result and result["data"]["patient"]:
            patient = result["data"]["patient"]
            name = patient.get("name", [{}])[0] if patient.get("name") else {}
            given_name = name.get("given", [""])[0] if name.get("given") else ""
            family_name = name.get("family", "")

            print(f"✅ Comprehensive data for: {given_name} {family_name}")

            # Query encounters separately
            print(f"\n🏥 Querying Encounters...")
            encounters_query = """
            query GetEncountersByPatient($patientId: ID!) {
              encounters(patientId: $patientId) {
                id
                status
                encounterClass
              }
            }
            """
            encounters_result = await self.execute_query(encounters_query, {"patientId": self.existing_patient_id})
            if "data" in encounters_result and encounters_result["data"]["encounters"]:
                encounters = encounters_result["data"]["encounters"]
                print(f"   Found {len(encounters)} encounters")
                for encounter in encounters[:3]:  # Show first 3
                    print(f"   - ID: {encounter.get('id')} (Status: {encounter.get('status', 'Unknown')})")
            else:
                print("   No encounters found")

            # Query observations separately
            print(f"\n📊 Querying Observations...")
            observations_query = """
            query SearchObservations($patientId: String, $page: Int, $count: Int) {
              observations(patientId: $patientId, page: $page, count: $count) {
                id
                status
                code {
                  text
                }
                valueQuantity {
                  value
                  unit
                }
              }
            }
            """
            observations_result = await self.execute_query(observations_query, {
                "patientId": self.existing_patient_id,
                "page": 1,
                "count": 5
            })
            if "data" in observations_result and observations_result["data"]["observations"]:
                observations = observations_result["data"]["observations"]
                print(f"   Found {len(observations)} observations")
                for obs in observations[:3]:  # Show first 3
                    code_text = obs.get("code", {}).get("text", "Unknown")
                    value = obs.get("valueQuantity", {})
                    if value:
                        print(f"   - {code_text}: {value.get('value', 'N/A')} {value.get('unit', '')}")
                    else:
                        print(f"   - {code_text}: No value")
            else:
                print("   No observations found")

            # Query tasks
            print(f"\n📋 Querying Workflow Tasks...")
            
            # Try to get all tasks first
            tasks_query = """
            query {
              tasks {
                id
                description
                status
                priority
              }
            }
            """
            
            tasks_result = await self.execute_query(tasks_query, {})
            
            if "errors" in tasks_result:
                print(f"   ❌ All tasks query failed: {tasks_result['errors']}")
            elif "data" in tasks_result and tasks_result["data"] and tasks_result["data"].get("tasks"):
                tasks = tasks_result["data"]["tasks"]
                print(f"   Found {len(tasks)} tasks")
                for task in tasks[:3]:  # Show first 3
                    print(f"   - {task.get('description', 'No description')} (Status: {task.get('status', 'Unknown')}, Priority: {task.get('priority', 'Unknown')})")
            else:
                print("   No tasks found")

            # Query workflow definitions (available in minimal service)
            print(f"\n🔄 Querying Workflow Definitions...")
            workflows_query = """
            query GetWorkflowDefinitions {
              workflowDefinitions {
                id
                name
                version
                status
                description
              }
            }
            """
            workflows_result = await self.execute_query(workflows_query, {})
            if "errors" in workflows_result:
                print(f"   ❌ Workflow definitions query failed: {workflows_result['errors']}")
            elif "data" in workflows_result and workflows_result["data"] and workflows_result["data"]["workflowDefinitions"]:
                workflows = workflows_result["data"]["workflowDefinitions"]
                print(f"   Found {len(workflows)} workflow definitions")
                for workflow in workflows[:3]:  # Show first 3
                    print(f"   - {workflow.get('name', 'Unknown')} (ID: {workflow.get('id', 'Unknown')}, Status: {workflow.get('status', 'Unknown')})")
            else:
                print("   No workflow definitions found")

            return True
        else:
            print(f"❌ No comprehensive data found")
            return False

    async def test_fetch_patient_data_workflow(self):
        """Test starting the 'fetch-patient-data' workflow and verify its execution."""
        print("\n🚀 Testing 'fetch-patient-data' Workflow")
        print("=" * 50)

        # Start the workflow using its BPMN Process ID
        workflow_payload = {
            "workflow_key": "fetch-patient-data",
            "variables": {
                "patient_id": self.existing_patient_id,
                "initial_variables": {
                    "source": "automated-test"
                }
            }
        }

        start_result = await self.execute_workflow_request("workflow/start", workflow_payload)

        if not start_result.get("success") or "workflow_instance" not in start_result:
            print(f"❌ Failed to start 'fetch-patient-data' workflow: {start_result.get('error', 'Unknown error')}")
            return False

        workflow_instance = start_result["workflow_instance"]
        instance_id = workflow_instance["id"]

        print(f"✅ Successfully started 'fetch-patient-data' workflow instance: {instance_id}")

        # Monitor workflow completion
        print("\n🔍 Monitoring workflow execution...")
        # Give it more time because it's calling external services
        completed = await self.wait_for_workflow_completion(instance_id, max_attempts=60, delay_seconds=5)

        if not completed:
            print("❌ 'fetch-patient-data' workflow did not complete successfully")
            return False

        # Get final workflow state and verify results
        final_state = await self.get_workflow_instance(instance_id)
        if final_state.get("success"):
            print("\n📊 Final Workflow State for 'fetch-patient-data':")
            instance = final_state.get("workflow_instance", {})
            status = instance.get("status")
            variables = instance.get("variables", {})
            
            print(f"Status: {status}")
            print(f"Variables: {json.dumps(variables, indent=2)}")

            # Assertions
            if status == 'completed' and 'vitals' in variables and 'encounters' in variables:
                print("✅ Verification successful: Workflow completed and contains 'vitals' and 'encounters' variables.")
                return True
            else:
                print("❌ Verification failed: Workflow did not complete as expected or is missing required variables.")
                return False
        
        return False

async def main():
    """Main function to run the tests."""
    tester = ExistingPatientWorkflowTester()
    
    # Run the new test for fetching patient data
    success = await tester.test_fetch_patient_data_workflow()
    print(f"\n🏁 Final test result: {'SUCCESS' if success else 'FAILURE'}")

if __name__ == "__main__":
    asyncio.run(main())
    
    async def test_start_workflow_for_existing_patient(self):
        """Start a workflow for the existing patient after checking current encounters and observations."""
        print("\n🔄 Starting Workflow for Existing Patient")
        print("=" * 50)

        # First, check existing encounters
        print("\n🏥 Checking Current Encounters...")
        encounters_query = """
        query GetEncountersByPatient($patientId: ID!) {
          encounters(patientId: $patientId) {
            id
            status
            encounterClass
            subject {
              reference
              display
            }
            period {
              start
              end
            }
          }
        }
        """

        encounters_variables = {
            "patientId": self.existing_patient_id
        }

        encounters_result = await self.execute_query(encounters_query, encounters_variables)

        if "errors" in encounters_result:
            print(f"❌ Encounters query failed: {encounters_result['errors']}")
        elif "data" in encounters_result and encounters_result["data"]["encounters"]:
            encounters = encounters_result["data"]["encounters"]
            print(f"✅ Found {len(encounters)} encounters:")
            for encounter in encounters[:3]:  # Show first 3
                print(f"   - ID: {encounter.get('id')}")
                print(f"     Status: {encounter.get('status')}")
                print(f"     Class: {encounter.get('encounterClass')}")
                period = encounter.get('period')
                if period and isinstance(period, dict):
                    start_time = period.get('start', 'N/A')
                    end_time = period.get('end', 'Ongoing')
                    print(f"     Period: {start_time} to {end_time}")
                else:
                    print(f"     Period: N/A")
        else:
            print("❌ No encounters found")

        # Second, check existing observations
        print("\n📊 Checking Current Observations...")
        observations_query = """
        query SearchObservations($patientId: String, $page: Int, $count: Int) {
          observations(patientId: $patientId, page: $page, count: $count) {
            id
            status
            code {
              text
              coding {
                system
                code
                display
              }
            }
            subject {
              reference
            }
            effectiveDateTime
            valueQuantity {
              value
              unit
            }
            valueCodeableConcept {
              text
            }
            category {
              text
              coding {
                code
                display
              }
            }
          }
        }
        """

        observations_variables = {
            "patientId": self.existing_patient_id,
            "page": 1,
            "count": 20
        }

        observations_result = await self.execute_query(observations_query, observations_variables)

        workflow_trigger = "routine-review"
        workflow_priority = "routine"
        clinical_context = []

        if "errors" in observations_result:
            print(f"❌ Observations query failed: {observations_result['errors']}")
        elif "data" in observations_result and observations_result["data"]["observations"]:
            observations = observations_result["data"]["observations"]
            print(f"✅ Found {len(observations)} observations:")

            # Analyze observations for workflow triggers
            for obs in observations[:5]:  # Show first 5
                code_display = obs.get('code', {}).get('coding', [{}])[0].get('display', 'Unknown')
                value_qty = obs.get('valueQuantity', {})
                value_concept = obs.get('valueCodeableConcept', {})

                if value_qty:
                    value_str = f"{value_qty.get('value', 'N/A')} {value_qty.get('unit', '')}"
                    print(f"   - {code_display}: {value_str}")

                    # Check for abnormal values that might trigger workflows
                    if 'blood pressure' in code_display.lower() or 'systolic' in code_display.lower():
                        if value_qty.get('value', 0) > 140:
                            workflow_trigger = "abnormal-vitals"
                            workflow_priority = "high"
                            clinical_context.append(f"Elevated BP: {value_str}")
                    elif 'temperature' in code_display.lower():
                        if value_qty.get('value', 0) > 38.0:
                            workflow_trigger = "abnormal-vitals"
                            workflow_priority = "urgent"
                            clinical_context.append(f"Fever: {value_str}")
                    elif 'heart rate' in code_display.lower():
                        if value_qty.get('value', 0) > 100:
                            workflow_trigger = "abnormal-vitals"
                            workflow_priority = "high"
                            clinical_context.append(f"Tachycardia: {value_str}")
                elif value_concept:
                    value_str = value_concept.get('text', 'N/A')
                    print(f"   - {code_display}: {value_str}")
                else:
                    print(f"   - {code_display}: No value recorded")
        else:
            print("❌ No observations found")

        # Determine workflow based on clinical data
        print(f"\n🧠 Clinical Assessment:")
        print(f"   Trigger: {workflow_trigger}")
        print(f"   Priority: {workflow_priority}")
        if clinical_context:
            print(f"   Clinical Context: {', '.join(clinical_context)}")

        # First, get the workflow definition ID
        print("\n🔍 Looking up workflow definition...")
        workflow_defs_query = """
        query GetWorkflowDefinitions {
          workflowDefinitions {
            id
            name
            status
          }
        }
        """

        workflow_defs = await self.execute_query(workflow_defs_query, {})
        if "errors" in workflow_defs:
            print(f"❌ Failed to get workflow definitions: {workflow_defs['errors']}")
            return

        # Find the workflow definition by name
        workflow_defs_list = workflow_defs.get("data", {}).get("workflowDefinitions", [])
        if not workflow_defs_list:
            print("❌ No workflow definitions found")
            return
            
        print(f"Found {len(workflow_defs_list)} workflow definitions")
        for wf in workflow_defs_list:
            print(f"  - {wf.get('name')} (ID: {wf.get('id')}, Status: {wf.get('status')})")

        # Try to find a matching workflow definition
        workflow_def = next(
            (w for w in workflow_defs_list
             if w.get("status") == "ACTIVE" and 
                ("admission" in w.get("name", "").lower() or 
                 "patient" in w.get("name", "").lower())),
            workflow_defs_list[0]  # Default to first one if none match
        )

        if not workflow_def:
            print("❌ No active workflow definitions found")
            return

        workflow_id = workflow_def["id"]
        print(f"\n🚀 Starting workflow: {workflow_def['name']} (ID: {workflow_id})")

        mutation = """
        mutation StartWorkflow($definitionId: ID!, $patientId: ID!) {
          startWorkflow(
            definitionId: $definitionId
            patientId: $patientId
          ) {
            id
            definitionId
            patientId
            status
          }
        }
        """

        try:
            print("\n🔍 Preparing to start workflow...")
            print(f"Workflow ID: {workflow_id}")
            print(f"Patient ID: {self.existing_patient_id}")
            
            # Prepare variables for workflow start
            variables = {
                "definitionId": workflow_id,
                "patientId": self.existing_patient_id
            }
            
            print("\n📤 Sending workflow start request...")
            result = await self.execute_query(mutation, variables)
            print("\n📥 Received response:", json.dumps(result, indent=2))

            if not result:
                print("❌ Empty response received from server")
                return False
                
            if "errors" in result:
                print("\n❌ Workflow start failed with errors:")
                for error in result["errors"]:
                    print(f"- {error.get('message')}")
                    if "extensions" in error:
                        print("  Extensions:", json.dumps(error["extensions"], indent=2))
                return False

            if "data" in result and result["data"].get("startWorkflow"):
                workflow = result["data"]["startWorkflow"]
                print("\n✅ Workflow started successfully!")
                print(f"   Workflow ID: {workflow.get('id')}")
                print(f"   Status: {workflow.get('status')}")
                return True
            else:
                print("\n❌ No workflow result in response")
                if "data" in result:
                    print("Response data:", json.dumps(result["data"], indent=2))
                return False
                
        except Exception as e:
            print(f"\n❌ Exception while starting workflow: {str(e)}")
            import traceback
            traceback.print_exc()
            return False
        print("=" * 50)
        
        # 1. Check workflow definitions (simplified query)
        print("\n📚 Checking workflow definitions...")
        def_query = """
        query {
          workflowDefinitions {
            id
            name
            status
          }
        }
        """
        def_result = await self.execute_query(def_query)
        if "data" in def_result and def_result["data"].get("workflowDefinitions"):
            defs = def_result["data"]["workflowDefinitions"]
            print(f"✅ Found {len(defs)} workflow definitions:")
            for d in defs[:5]:  # Show first 5
                print(f"   - {d.get('name')} (ID: {d.get('id')}, Status: {d.get('status')})")
        else:
            print(f"❌ Failed to get workflow definitions: {def_result.get('errors', 'No definitions found')}")
        
        # 2. Check existing workflow instances (simplified query)
        print("\n⚙️ Checking existing workflow instances...")
        instances_query = """
        query {
          workflowInstances {
            id
            definitionId
            status
          }
        }
        """
        instances_result = await self.execute_query(instances_query)
        if "data" in instances_result and instances_result["data"].get("workflowInstances"):
            instances = instances_result["data"]["workflowInstances"]
            if instances:
                print(f"✅ Found {len(instances)} workflow instances:")
                for i in instances[:5]:  # Show first 5
                    print(f"   - ID: {i.get('id')}")
                    print(f"     Definition: {i.get('definitionId')}")
                    print(f"     Status: {i.get('status')}")
            else:
                print("ℹ️ No workflow instances found")

    async def fix_workflow_statuses(self):
        """Fix invalid workflow statuses in the database."""
        try:
            print("\n🔧 Attempting to fix invalid workflow statuses...")
            
            # First, get a list of all workflow instances with their current statuses
            query = """
            query {
              __type(name: "WorkflowStatus") {
                enumValues {
                  name
                }
              }
            }
            """
            
            result = await self.execute_query(query)
            if not result or "data" not in result or "__type" not in result["data"]:
                print("❌ Could not retrieve valid workflow statuses")
                return False
                
            valid_statuses = {v["name"] for v in result["data"]["__type"]["enumValues"]}
            print(f"✅ Valid workflow statuses: {', '.join(valid_statuses)}")
            
            # Now find and fix invalid statuses
            fix_mutation = """
            mutation FixWorkflowStatus($id: ID!, $status: WorkflowStatus!) {
              updateWorkflowInstance(id: $id, input: { status: $status }) {
                id
                status
              }
            }
            """
            
            # Get all workflow instances with their current status
            instances_query = """
            query {
              workflowInstances {
                id
                status
              }
            }
            """
            
            instances_result = await self.execute_query(instances_query)
            if not instances_result or "data" not in instances_result:
                print("❌ Could not retrieve workflow instances")
                return False
                
            fixed_count = 0
            for instance in instances_result.get("data", {}).get("workflowInstances", []):
                status = instance.get("status")
                if status and status.upper() not in valid_statuses:
                    print(f"⚠️  Found invalid status '{status}' for instance {instance['id']}")
                    
                    # Map invalid statuses to valid ones
                    status_map = {
                        'running': 'RUNNING',
                        'error': 'ERROR',
                        'failed': 'FAILED',
                        'pending': 'PENDING',
                        'cancelled': 'CANCELLED'
                    }
                    
                    new_status = status_map.get(status.lower(), 'ERROR')
                    print(f"   → Updating to '{new_status}'")
                    
                    # Update the status
                    update_result = await self.execute_query(
                        fix_mutation,
                        variables={"id": instance["id"], "status": new_status}
                    )
                    
                    if update_result and "errors" not in update_result:
                        fixed_count += 1
                    else:
                        print(f"   ❌ Failed to update: {update_result.get('errors', 'Unknown error')}")
            
            if fixed_count > 0:
                print(f"✅ Fixed {fixed_count} workflow instances with invalid statuses")
            else:
                print("ℹ️ No invalid workflow statuses found")
                
            return True
            
        except Exception as e:
            print(f"❌ Error fixing workflow statuses: {str(e)}")
            import traceback
            traceback.print_exc()
            return False
    
    async def check_workflow_health(self):
        """Check health of workflow endpoints."""
        print("\n🔍 WORKFLOW SERVICE HEALTH CHECK")
        print("=" * 50)
        
        # 1. Check workflow definitions (simplified query)
        print("\n📚 Checking workflow definitions...")
        def_query = """
        query {
          workflowDefinitions {
            id
            name
            status
          }
        }
        """
        def_result = await self.execute_query(def_query)
        if "data" in def_result and def_result["data"].get("workflowDefinitions"):
            defs = def_result["data"]["workflowDefinitions"]
            print(f"✅ Found {len(defs)} workflow definitions:")
            for d in defs[:5]:  # Show first 5
                print(f"   - {d.get('name')} (ID: {d.get('id')}, Status: {d.get('status')})")
        else:
            print(f"❌ Failed to get workflow definitions: {def_result.get('errors', 'No definitions found')}")
        
        # 2. Check existing workflow instances (simplified query)
        print("\n⚙️ Checking existing workflow instances...")
        try:
            # First, try to get workflow instances with a simpler query
            simple_inst_query = """
            query {
              workflowInstances {
                id
                status
              }
            }
            """
            print("\n🔍 Executing simple workflow instances query...")
            simple_result = await self.execute_query(simple_inst_query)
            print(f"Simple query result: {simple_result}")
            
            # If the simple query fails, try to fix the database
            if not simple_result or "errors" in simple_result or "data" not in simple_result:
                print("❌ Failed to query workflow instances. Attempting to fix database...")
                fixed = await self.fix_workflow_statuses()
                if not fixed:
                    return False
                
                # Try the query again after fixing
                print("\n🔄 Retrying workflow instances query after fix...")
                simple_result = await self.execute_query(simple_inst_query)
                if not simple_result or "errors" in simple_result or "data" not in simple_result:
                    print("❌ Still unable to query workflow instances after fix attempt")
                    return False
            
            # Handle case where data is None
            if simple_result.get("data") is None:
                print("❌ No data in response. Attempting to fix workflow statuses...")
                fixed = await self.fix_workflow_statuses()
                if not fixed:
                    print("❌ Failed to fix workflow statuses")
                    return False
                
                # Try the query again after fixing
                print("\n🔄 Retrying workflow instances query after fix...")
                simple_result = await self.execute_query(simple_inst_query)
                if not simple_result or "data" not in simple_result:
                    print("❌ Still unable to query workflow instances after fix attempt")
                    return False
            
            # Now we should have data, check for workflow instances
            if simple_result["data"] and "workflowInstances" in simple_result["data"]:
                insts = simple_result["data"]["workflowInstances"]
                if insts:
                    print(f"✅ Found {len(insts)} workflow instances:")
                    for i in insts[:3]:  # Show first 3
                        print(f"   - Instance ID: {i.get('id')}, Status: {i.get('status')}")
                    return True
                else:
                    print("ℹ️ No workflow instances found in the database")
            else:
                print("❌ No workflow instances data in response")
                print(f"Response: {simple_result}")
                return False
            
            # Try to start a test workflow if we got this far
            print("\n🔄 Attempting to start a test workflow...")
            workflow_started = await self.test_start_workflow_for_existing_patient()
            if workflow_started:
                print("✅ Test workflow started successfully")
                return True
            else:
                print("❌ Failed to start test workflow")
                return False
                
        except Exception as e:
            print(f"❌ Exception in check_workflow_health: {str(e)}")
            import traceback
            traceback.print_exc()
            return False
        
        print("\n✅ Workflow service health check completed")
        return True

    async def run_tests(self):
        """Run all workflow tests."""
        print("\n" + "=" * 50)
        print("🚀 Starting Workflow Integration Tests")
        print("=" * 50)
        
        test_results = {}
        
        # Run workflow start and monitor test
        test_results["workflow_start_and_monitor"] = await self.test_workflow_start_and_monitor()
        
        # Print test summary
        print("\n" + "=" * 50)
        print("Test Summary:")
        for test_name, passed in test_results.items():
            status = "✅ Passed" if passed else "❌ Failed"
            print(f"- {test_name.replace('_', ' ').title()}: {status}")
        
        all_passed = all(test_results.values())
        if all_passed:
            print("\n🎉 All workflow tests completed successfully!")
        else:
            print("\n❌ Some workflow tests failed. See above for details.")
        
        return all_passed

async def main():
    """Main execution function."""
    try:
        tester = ExistingPatientWorkflowTester()
        success = await tester.run_tests()
        return 0 if success else 1
    except Exception as e:
        print(f"\n❌ An error occurred: {str(e)}")
        import traceback
        traceback.print_exc()
        return 1

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    exit(exit_code)
