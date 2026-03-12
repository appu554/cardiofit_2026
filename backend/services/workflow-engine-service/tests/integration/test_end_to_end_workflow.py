"""
End-to-end workflow integration tests.
"""
import pytest
from unittest.mock import patch, MagicMock
import asyncio

from app.services.workflow_definition_service import workflow_definition_service
from app.services.workflow_instance_service import workflow_instance_service
from app.services.task_service import task_service


@pytest.mark.integration
@pytest.mark.workflow
@pytest.mark.asyncio
class TestEndToEndWorkflow:
    """Test complete workflow execution scenarios."""

    async def test_patient_admission_workflow_complete_flow(self, mock_supabase_client, mock_google_fhir_client, mock_camunda_client):
        """Test complete patient admission workflow from start to finish."""
        
        # Setup mocks
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_definition_service, 'google_fhir_service', mock_google_fhir_client), \
             patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_instance_service, 'camunda_service', mock_camunda_client), \
             patch.object(task_service, 'supabase_client', mock_supabase_client), \
             patch.object(task_service, 'google_fhir_service', mock_google_fhir_client), \
             patch.object(task_service, 'camunda_service', mock_camunda_client):
            
            # Step 1: Create workflow definition
            workflow_def = {
                "id": "patient-admission-workflow",
                "name": "Patient Admission Workflow",
                "description": "Complete patient admission process",
                "version": 1,
                "bpmn_xml": """<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                  <bpmn:process id="patient-admission" isExecutable="true">
                    <bpmn:startEvent id="start"/>
                    <bpmn:userTask id="review-patient-data" name="Review Patient Data"/>
                    <bpmn:userTask id="assign-room" name="Assign Room"/>
                    <bpmn:endEvent id="end"/>
                    <bpmn:sequenceFlow sourceRef="start" targetRef="review-patient-data"/>
                    <bpmn:sequenceFlow sourceRef="review-patient-data" targetRef="assign-room"/>
                    <bpmn:sequenceFlow sourceRef="assign-room" targetRef="end"/>
                  </bpmn:process>
                </bpmn:definitions>""",
                "category": "admission",
                "is_active": True
            }
            
            # Mock workflow definition creation
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [workflow_def]
            mock_google_fhir_client.create_resource.return_value = {"id": "plan-def-123"}
            
            created_def = await workflow_definition_service.create_workflow_definition(workflow_def)
            assert created_def["id"] == "patient-admission-workflow"
            
            # Step 2: Start workflow instance
            workflow_instance = {
                "id": "instance-123",
                "definition_id": "patient-admission-workflow",
                "patient_id": "patient-456",
                "status": "active",
                "variables": {"patientName": "John Doe", "admissionType": "emergency"},
                "camunda_instance_key": "camunda-instance-789"
            }
            
            # Mock workflow instance creation
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [workflow_instance]
            mock_camunda_client.start_workflow_instance.return_value = {
                "workflowInstanceKey": "camunda-instance-789"
            }
            
            started_instance = await workflow_instance_service.start_workflow_instance(
                definition_id="patient-admission-workflow",
                patient_id="patient-456",
                variables={"patientName": "John Doe", "admissionType": "emergency"}
            )
            assert started_instance["status"] == "active"
            
            # Step 3: Create first task (Review Patient Data)
            review_task = {
                "id": "task-review-123",
                "workflow_instance_id": "instance-123",
                "name": "Review Patient Data",
                "description": "Review and validate patient admission data",
                "assignee_id": "doctor-789",
                "status": "ready",
                "form_data": {"patientId": "patient-456", "admissionType": "emergency"},
                "fhir_task_id": "fhir-task-review-123"
            }
            
            # Mock task creation
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [review_task]
            mock_google_fhir_client.create_resource.return_value = {"id": "fhir-task-review-123"}
            
            created_task = await task_service.create_task(review_task)
            assert created_task["name"] == "Review Patient Data"
            
            # Step 4: Claim and complete first task
            claimed_task = {**review_task, "status": "claimed"}
            completed_task = {**review_task, "status": "completed"}
            
            # Mock task claiming
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [claimed_task]
            mock_google_fhir_client.update_resource.return_value = {"id": "fhir-task-review-123"}
            
            claimed = await task_service.claim_task("task-review-123", "doctor-789")
            assert claimed["status"] == "claimed"
            
            # Mock task completion
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [completed_task]
            mock_camunda_client.complete_job.return_value = True
            
            completed = await task_service.complete_task(
                "task-review-123", 
                {"reviewResult": "approved", "notes": "Patient data verified"}
            )
            assert completed["status"] == "completed"
            
            # Step 5: Create second task (Assign Room)
            room_task = {
                "id": "task-room-456",
                "workflow_instance_id": "instance-123",
                "name": "Assign Room",
                "description": "Assign appropriate room to patient",
                "assignee_id": "nurse-456",
                "status": "ready",
                "form_data": {"patientId": "patient-456", "roomType": "private"},
                "fhir_task_id": "fhir-task-room-456"
            }
            
            # Mock second task creation
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [room_task]
            mock_google_fhir_client.create_resource.return_value = {"id": "fhir-task-room-456"}
            
            room_task_created = await task_service.create_task(room_task)
            assert room_task_created["name"] == "Assign Room"
            
            # Step 6: Complete second task
            completed_room_task = {**room_task, "status": "completed"}
            
            # Mock room task completion
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [completed_room_task]
            
            room_completed = await task_service.complete_task(
                "task-room-456",
                {"assignedRoom": "Room 101", "roomType": "private"}
            )
            assert room_completed["status"] == "completed"
            
            # Step 7: Complete workflow instance
            completed_instance = {**workflow_instance, "status": "completed"}
            
            # Mock workflow completion
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [completed_instance]
            
            final_instance = await workflow_instance_service.update_workflow_instance_status(
                "instance-123", 
                "completed"
            )
            assert final_instance["status"] == "completed"
            
            # Verify the complete flow
            assert created_def["id"] == "patient-admission-workflow"
            assert started_instance["patient_id"] == "patient-456"
            assert created_task["assignee_id"] == "doctor-789"
            assert completed["status"] == "completed"
            assert room_task_created["assignee_id"] == "nurse-456"
            assert room_completed["status"] == "completed"
            assert final_instance["status"] == "completed"

    async def test_workflow_with_parallel_tasks(self, mock_supabase_client, mock_google_fhir_client, mock_camunda_client):
        """Test workflow with parallel task execution."""
        
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_instance_service, 'camunda_service', mock_camunda_client), \
             patch.object(task_service, 'supabase_client', mock_supabase_client), \
             patch.object(task_service, 'google_fhir_service', mock_google_fhir_client), \
             patch.object(task_service, 'camunda_service', mock_camunda_client):
            
            # Start workflow instance
            workflow_instance = {
                "id": "parallel-instance-123",
                "definition_id": "parallel-workflow",
                "patient_id": "patient-789",
                "status": "active",
                "variables": {},
                "camunda_instance_key": "parallel-camunda-123"
            }
            
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [workflow_instance]
            mock_camunda_client.start_workflow_instance.return_value = {"workflowInstanceKey": "parallel-camunda-123"}
            
            instance = await workflow_instance_service.start_workflow_instance(
                definition_id="parallel-workflow",
                patient_id="patient-789",
                variables={}
            )
            
            # Create parallel tasks
            task1 = {
                "id": "parallel-task-1",
                "workflow_instance_id": "parallel-instance-123",
                "name": "Lab Tests",
                "assignee_id": "lab-tech-1",
                "status": "ready",
                "fhir_task_id": "fhir-task-1"
            }
            
            task2 = {
                "id": "parallel-task-2",
                "workflow_instance_id": "parallel-instance-123",
                "name": "Imaging",
                "assignee_id": "radiologist-1",
                "status": "ready",
                "fhir_task_id": "fhir-task-2"
            }
            
            # Mock parallel task creation
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [task1]
            mock_google_fhir_client.create_resource.return_value = {"id": "fhir-task-1"}
            created_task1 = await task_service.create_task(task1)
            
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [task2]
            mock_google_fhir_client.create_resource.return_value = {"id": "fhir-task-2"}
            created_task2 = await task_service.create_task(task2)
            
            # Complete tasks in parallel (simulate concurrent execution)
            completed_task1 = {**task1, "status": "completed"}
            completed_task2 = {**task2, "status": "completed"}
            
            # Mock parallel completion
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [completed_task1]
            mock_camunda_client.complete_job.return_value = True
            
            # Complete both tasks
            result1 = await task_service.complete_task("parallel-task-1", {"labResults": "normal"})
            
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [completed_task2]
            result2 = await task_service.complete_task("parallel-task-2", {"imagingResults": "clear"})
            
            assert result1["status"] == "completed"
            assert result2["status"] == "completed"

    async def test_workflow_error_handling_and_recovery(self, mock_supabase_client, mock_camunda_client):
        """Test workflow error handling and recovery scenarios."""
        
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_instance_service, 'camunda_service', mock_camunda_client):
            
            # Test scenario: Camunda service failure during workflow start
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [{
                "id": "error-instance-123",
                "definition_id": "test-workflow",
                "patient_id": "patient-123",
                "status": "failed",
                "variables": {},
                "camunda_instance_key": None
            }]
            
            # Mock Camunda failure
            mock_camunda_client.start_workflow_instance.side_effect = Exception("Camunda service unavailable")
            
            # Should handle the error gracefully
            try:
                await workflow_instance_service.start_workflow_instance(
                    definition_id="test-workflow",
                    patient_id="patient-123",
                    variables={}
                )
            except Exception as e:
                assert "Camunda service unavailable" in str(e)
            
            # Test recovery: retry workflow start
            mock_camunda_client.start_workflow_instance.side_effect = None
            mock_camunda_client.start_workflow_instance.return_value = {"workflowInstanceKey": "recovered-123"}
            
            recovered_instance = {
                "id": "recovered-instance-123",
                "definition_id": "test-workflow",
                "patient_id": "patient-123",
                "status": "active",
                "variables": {},
                "camunda_instance_key": "recovered-123"
            }
            
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [recovered_instance]
            
            result = await workflow_instance_service.start_workflow_instance(
                definition_id="test-workflow",
                patient_id="patient-123",
                variables={}
            )
            
            assert result["status"] == "active"
            assert result["camunda_instance_key"] == "recovered-123"

    async def test_workflow_timeout_and_escalation(self, mock_supabase_client, mock_google_fhir_client):
        """Test workflow timeout and escalation scenarios."""
        
        with patch.object(task_service, 'supabase_client', mock_supabase_client), \
             patch.object(task_service, 'google_fhir_service', mock_google_fhir_client):
            
            # Create overdue task
            overdue_task = {
                "id": "overdue-task-123",
                "workflow_instance_id": "instance-123",
                "name": "Urgent Review",
                "assignee_id": "doctor-123",
                "status": "ready",
                "due_date": "2024-01-01T00:00:00Z",  # Past due date
                "fhir_task_id": "fhir-overdue-123"
            }
            
            # Mock overdue task query
            mock_supabase_client.table.return_value.select.return_value.lt.return_value.execute.return_value.data = [overdue_task]
            
            overdue_tasks = await task_service.get_overdue_tasks()
            
            assert len(overdue_tasks) == 1
            assert overdue_tasks[0]["id"] == "overdue-task-123"
            
            # Test escalation: reassign to supervisor
            escalated_task = {**overdue_task, "assignee_id": "supervisor-456", "status": "escalated"}
            
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [escalated_task]
            mock_google_fhir_client.update_resource.return_value = {"id": "fhir-overdue-123"}
            
            escalated = await task_service.delegate_task("overdue-task-123", "supervisor-456")
            
            assert escalated["assignee_id"] == "supervisor-456"

    async def test_workflow_cancellation_flow(self, mock_supabase_client, mock_google_fhir_client, mock_camunda_client):
        """Test workflow cancellation and cleanup."""
        
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_instance_service, 'camunda_service', mock_camunda_client), \
             patch.object(task_service, 'supabase_client', mock_supabase_client), \
             patch.object(task_service, 'google_fhir_service', mock_google_fhir_client):
            
            # Cancel workflow instance
            canceled_instance = {
                "id": "cancel-instance-123",
                "definition_id": "test-workflow",
                "patient_id": "patient-123",
                "status": "canceled",
                "variables": {},
                "camunda_instance_key": "cancel-camunda-123"
            }
            
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [canceled_instance]
            mock_camunda_client.cancel_workflow_instance.return_value = True
            
            result = await workflow_instance_service.cancel_workflow_instance("cancel-instance-123")
            
            assert result["status"] == "canceled"
            
            # Cancel associated tasks
            canceled_task = {
                "id": "cancel-task-123",
                "workflow_instance_id": "cancel-instance-123",
                "status": "canceled",
                "fhir_task_id": "fhir-cancel-123"
            }
            
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [canceled_task]
            mock_google_fhir_client.update_resource.return_value = {"id": "fhir-cancel-123"}
            
            canceled_task_result = await task_service.cancel_task("cancel-task-123")
            
            assert canceled_task_result["status"] == "canceled"
