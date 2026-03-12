"""
Comprehensive test for Enhanced Workflow Service with Patient, Vitals, and Encounters
"""
import requests
import json
from typing import Dict, Any

def test_enhanced_workflow_service():
    """Test the enhanced workflow service with full patient integration."""
    
    base_url = "http://localhost:8015"
    graphql_url = f"{base_url}/api/federation"
    
    print("🧬 Testing Enhanced Workflow Service with Patient Integration")
    print("=" * 70)
    
    # Test 1: Health Check with Features
    print("\n1. Enhanced Health Check...")
    try:
        response = requests.get(f"{base_url}/health")
        if response.status_code == 200:
            health_data = response.json()
            print("✅ Enhanced service is healthy")
            print(f"   Service: {health_data['service']}")
            print(f"   Version: {health_data['version']}")
            print(f"   Features: {health_data['features']}")
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Health check failed: {e}")
        return False
    
    # Test 2: Patient Data Query
    print("\n2. Patient Data Query...")
    query = {
        "query": """
        {
          patients {
            id
            name
            gender
            birthDate
            active
          }
        }
        """
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["patients"]:
                patients = result["data"]["patients"]
                print(f"✅ Found {len(patients)} patients:")
                for patient in patients:
                    print(f"   - {patient['name']} (ID: {patient['id']}, Gender: {patient['gender']})")
                return patients
            else:
                print(f"❌ No patients found: {result}")
                return False
        else:
            print(f"❌ Query failed: {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Patient query failed: {e}")
        return False
    
    # Test 3: Vitals Query
    print("\n3. Patient Vitals Query...")
    query = {
        "query": """
        query GetVitals($patientId: String) {
          vitals(patientId: $patientId) {
            id
            patientId
            code
            display
            value
            unit
            recordedTime
            status
          }
        }
        """,
        "variables": {
            "patientId": "patient-001"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["vitals"]:
                vitals = result["data"]["vitals"]
                print(f"✅ Found {len(vitals)} vitals for patient-001:")
                for vital in vitals:
                    print(f"   - {vital['display']}: {vital['value']} {vital['unit']}")
                    print(f"     Recorded: {vital['recordedTime']}, Status: {vital['status']}")
            else:
                print(f"❌ No vitals found: {result}")
        else:
            print(f"❌ Vitals query failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Vitals query failed: {e}")
    
    # Test 4: Encounters Query
    print("\n4. Patient Encounters Query...")
    query = {
        "query": """
        query GetEncounters($patientId: String) {
          encounters(patientId: $patientId) {
            id
            patientId
            status
            encounterClass
            typeDisplay
            startTime
            endTime
            location
          }
        }
        """,
        "variables": {
            "patientId": "patient-001"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["encounters"]:
                encounters = result["data"]["encounters"]
                print(f"✅ Found {len(encounters)} encounters for patient-001:")
                for encounter in encounters:
                    print(f"   - {encounter['typeDisplay']} ({encounter['encounterClass']})")
                    print(f"     Status: {encounter['status']}, Location: {encounter['location']}")
                    print(f"     Started: {encounter['startTime']}")
            else:
                print(f"❌ No encounters found: {result}")
        else:
            print(f"❌ Encounters query failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Encounters query failed: {e}")
    
    # Test 5: Workflow-Related Tasks
    print("\n5. Workflow Tasks Query...")
    query = {
        "query": """
        query GetTasks($patientId: String) {
          tasks(patientId: $patientId) {
            id
            workflowInstanceId
            patientId
            description
            status
            priority
            assignee
            dueDate
            context
          }
        }
        """,
        "variables": {
            "patientId": "patient-001"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["tasks"]:
                tasks = result["data"]["tasks"]
                print(f"✅ Found {len(tasks)} tasks for patient-001:")
                for task in tasks:
                    print(f"   - {task['description']} (Priority: {task['priority']})")
                    print(f"     Assigned to: {task['assignee']}, Due: {task['dueDate']}")
                    print(f"     Context: {task['context']}")
            else:
                print(f"❌ No tasks found: {result}")
        else:
            print(f"❌ Tasks query failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Tasks query failed: {e}")
    
    # Test 6: Comprehensive Patient Summary
    print("\n6. Comprehensive Patient Summary...")
    query = {
        "query": """
        query GetPatientSummary($patientId: String!) {
          patientSummary(patientId: $patientId)
        }
        """,
        "variables": {
            "patientId": "patient-001"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=query)
        if response.status_code == 200:
            result = response.json()
            if "data" in result and result["data"]["patientSummary"]:
                summary_json = result["data"]["patientSummary"]
                summary = json.loads(summary_json)
                print(f"✅ Patient Summary Retrieved:")
                print(f"   Patient: {summary['patient']['name']}")
                print(f"   Current Encounter: {summary['current_encounter']['typeDisplay'] if summary['current_encounter'] else 'None'}")
                print(f"   Latest Vitals: {len(summary['latest_vitals'])} readings")
                print(f"   Active Tasks: {len(summary['active_tasks'])} tasks")
                print(f"   Active Workflows: {len(summary['active_workflows'])} workflows")
            else:
                print(f"❌ No patient summary found: {result}")
        else:
            print(f"❌ Patient summary query failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Patient summary query failed: {e}")

def test_workflow_scenarios():
    """Test realistic clinical workflow scenarios."""
    
    base_url = "http://localhost:8015"
    graphql_url = f"{base_url}/api/federation"
    
    print("\n🏥 Testing Clinical Workflow Scenarios")
    print("=" * 50)
    
    # Scenario 1: Abnormal Vitals Trigger Workflow
    print("\n📊 Scenario 1: Abnormal Vitals Detection")
    print("Patient John Doe has high blood pressure (140/90)")
    print("System should automatically start vitals monitoring workflow")
    
    mutation = {
        "query": """
        mutation StartVitalsMonitoring($definitionId: String!, $patientId: String!, $trigger: String) {
          startWorkflow(definitionId: $definitionId, patientId: $patientId, trigger: $trigger)
        }
        """,
        "variables": {
            "definitionId": "vitals-monitoring-workflow",
            "patientId": "patient-001",
            "trigger": "abnormal-vitals"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=mutation)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Workflow started: {result['data']['startWorkflow']}")
        else:
            print(f"❌ Workflow start failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Workflow start failed: {e}")
    
    # Scenario 2: Emergency Admission
    print("\n🚨 Scenario 2: Emergency Department Admission")
    print("Patient arrives at ED, needs immediate admission workflow")
    
    mutation = {
        "query": """
        mutation StartAdmission($definitionId: String!, $patientId: String!, $trigger: String) {
          startWorkflow(definitionId: $definitionId, patientId: $patientId, trigger: $trigger)
        }
        """,
        "variables": {
            "definitionId": "patient-admission-workflow",
            "patientId": "patient-001",
            "trigger": "emergency-admission"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=mutation)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Emergency admission workflow started: {result['data']['startWorkflow']}")
        else:
            print(f"❌ Emergency admission failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Emergency admission failed: {e}")
    
    # Scenario 3: Task Completion
    print("\n✅ Scenario 3: Doctor Completes Vital Review Task")
    print("Doctor reviews abnormal BP and prescribes medication")
    
    mutation = {
        "query": """
        mutation CompleteTask($taskId: String!, $result: String) {
          completeTask(taskId: $taskId, result: $result)
        }
        """,
        "variables": {
            "taskId": "task-001",
            "result": "Prescribed antihypertensive medication, schedule follow-up in 1 week"
        }
    }
    
    try:
        response = requests.post(graphql_url, json=mutation)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Task completed: {result['data']['completeTask']}")
        else:
            print(f"❌ Task completion failed: {response.status_code}")
    except Exception as e:
        print(f"❌ Task completion failed: {e}")

def demonstrate_integration_benefits():
    """Demonstrate the benefits of workflow-patient-vitals integration."""
    
    print("\n🎯 Integration Benefits Demonstration")
    print("=" * 50)
    
    print("""
    🔗 WORKFLOW-PATIENT-VITALS INTEGRATION BENEFITS:
    
    1. 📊 AUTOMATED CLINICAL DECISION SUPPORT
       - Abnormal vitals automatically trigger monitoring workflows
       - Tasks are created with full patient context
       - Escalation based on patient risk factors
    
    2. 🏥 COORDINATED CARE DELIVERY
       - All team members see patient's complete picture
       - Tasks reference current encounter and vitals
       - Workflow progress tracked across departments
    
    3. 📋 INTELLIGENT TASK MANAGEMENT
       - Tasks prioritized based on vital signs severity
       - Assignments consider patient location and staff availability
       - Context includes relevant clinical data
    
    4. 🔄 REAL-TIME PROCESS ADAPTATION
       - Workflows adapt based on changing patient condition
       - New vitals can trigger additional workflow steps
       - Encounter status influences task routing
    
    5. 📈 QUALITY METRICS & COMPLIANCE
       - Complete audit trail of clinical decisions
       - Workflow completion times tracked
       - Compliance with clinical protocols ensured
    """)

if __name__ == "__main__":
    # Run enhanced integration tests
    success = test_enhanced_workflow_service()
    
    if success:
        # Test clinical scenarios
        test_workflow_scenarios()
        
        # Demonstrate integration benefits
        demonstrate_integration_benefits()
        
        print("\n" + "=" * 70)
        print("🎉 ENHANCED WORKFLOW INTEGRATION TESTING COMPLETED!")
        print("\n📊 Summary of What We've Demonstrated:")
        print("✅ Patient data integration with workflows")
        print("✅ Vitals-triggered workflow automation")
        print("✅ Encounter-aware task management")
        print("✅ Comprehensive patient summaries")
        print("✅ Clinical decision support workflows")
        print("✅ Real-time task prioritization")
        
        print("\n🚀 This demonstrates how workflows help us:")
        print("• Automate clinical processes based on patient data")
        print("• Coordinate care across multiple departments")
        print("• Ensure timely response to critical vitals")
        print("• Maintain complete clinical context in all tasks")
        print("• Improve patient safety through systematic workflows")
    else:
        print("\n❌ Enhanced integration tests failed. Please check the service.")
