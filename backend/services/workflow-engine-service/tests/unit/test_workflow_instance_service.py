"""
Unit tests for Workflow Instance Service.
"""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from app.services.workflow_instance_service import workflow_instance_service


@pytest.mark.unit
@pytest.mark.asyncio
class TestWorkflowInstanceService:
    """Test cases for WorkflowInstanceService."""

    async def test_start_workflow_instance(self, mock_supabase_client, mock_camunda_client, sample_workflow_instance):
        """Test starting a workflow instance."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_instance_service, 'camunda_service', mock_camunda_client):
            
            # Mock successful database insert
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [sample_workflow_instance]
            
            # Mock successful Camunda instance start
            mock_camunda_client.start_workflow_instance.return_value = {
                "workflowInstanceKey": sample_workflow_instance["camunda_instance_key"]
            }
            
            result = await workflow_instance_service.start_workflow_instance(
                definition_id=sample_workflow_instance["definition_id"],
                patient_id=sample_workflow_instance["patient_id"],
                variables=sample_workflow_instance["variables"]
            )
            
            assert result is not None
            assert result["definition_id"] == sample_workflow_instance["definition_id"]
            assert result["patient_id"] == sample_workflow_instance["patient_id"]
            assert result["status"] == "active"
            
            # Verify database call
            mock_supabase_client.table.assert_called_with("workflow_instances")
            
            # Verify Camunda call
            mock_camunda_client.start_workflow_instance.assert_called_once()

    async def test_get_workflow_instance(self, mock_supabase_client, sample_workflow_instance):
        """Test getting a workflow instance by ID."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_workflow_instance]
            
            result = await workflow_instance_service.get_workflow_instance(sample_workflow_instance["id"])
            
            assert result is not None
            assert result["id"] == sample_workflow_instance["id"]
            
            # Verify database call
            mock_supabase_client.table.assert_called_with("workflow_instances")

    async def test_get_workflow_instance_not_found(self, mock_supabase_client):
        """Test getting a non-existent workflow instance."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            # Mock empty database result
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = []
            
            result = await workflow_instance_service.get_workflow_instance("non-existent-id")
            
            assert result is None

    async def test_list_workflow_instances(self, mock_supabase_client, sample_workflow_instance):
        """Test listing workflow instances."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.execute.return_value.data = [sample_workflow_instance]
            
            result = await workflow_instance_service.list_workflow_instances()
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["id"] == sample_workflow_instance["id"]

    async def test_list_workflow_instances_by_status(self, mock_supabase_client, sample_workflow_instance):
        """Test listing workflow instances by status."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_workflow_instance]
            
            result = await workflow_instance_service.list_workflow_instances(status="active")
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["status"] == "active"

    async def test_list_workflow_instances_by_patient(self, mock_supabase_client, sample_workflow_instance):
        """Test listing workflow instances by patient ID."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_workflow_instance]
            
            result = await workflow_instance_service.list_workflow_instances(patient_id=sample_workflow_instance["patient_id"])
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["patient_id"] == sample_workflow_instance["patient_id"]

    async def test_update_workflow_instance_status(self, mock_supabase_client, sample_workflow_instance):
        """Test updating workflow instance status."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            updated_instance = {**sample_workflow_instance, "status": "completed"}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [updated_instance]
            
            result = await workflow_instance_service.update_workflow_instance_status(
                sample_workflow_instance["id"], 
                "completed"
            )
            
            assert result is not None
            assert result["status"] == "completed"

    async def test_update_workflow_instance_variables(self, mock_supabase_client, sample_workflow_instance):
        """Test updating workflow instance variables."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            new_variables = {"patientName": "Jane Doe", "priority": "high"}
            updated_instance = {**sample_workflow_instance, "variables": new_variables}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [updated_instance]
            
            result = await workflow_instance_service.update_workflow_instance_variables(
                sample_workflow_instance["id"], 
                new_variables
            )
            
            assert result is not None
            assert result["variables"] == new_variables

    async def test_signal_workflow_instance(self, mock_camunda_client, sample_workflow_instance):
        """Test signaling a workflow instance."""
        with patch.object(workflow_instance_service, 'camunda_service', mock_camunda_client):
            
            # Mock successful message publishing
            mock_camunda_client.publish_message.return_value = True
            
            result = await workflow_instance_service.signal_workflow_instance(
                sample_workflow_instance["id"],
                "test_signal",
                {"data": "test"}
            )
            
            assert result is True
            
            # Verify Camunda call
            mock_camunda_client.publish_message.assert_called_once()

    async def test_cancel_workflow_instance(self, mock_supabase_client, mock_camunda_client, sample_workflow_instance):
        """Test canceling a workflow instance."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_instance_service, 'camunda_service', mock_camunda_client):
            
            canceled_instance = {**sample_workflow_instance, "status": "canceled"}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [canceled_instance]
            
            # Mock successful Camunda cancellation
            mock_camunda_client.cancel_workflow_instance.return_value = True
            
            result = await workflow_instance_service.cancel_workflow_instance(sample_workflow_instance["id"])
            
            assert result is not None
            assert result["status"] == "canceled"

    async def test_get_workflow_instance_history(self, mock_supabase_client, sample_workflow_instance):
        """Test getting workflow instance history."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            history_events = [
                {
                    "id": "event-1",
                    "workflow_instance_id": sample_workflow_instance["id"],
                    "event_type": "workflow_started",
                    "timestamp": "2024-01-01T00:00:00Z"
                },
                {
                    "id": "event-2",
                    "workflow_instance_id": sample_workflow_instance["id"],
                    "event_type": "task_created",
                    "timestamp": "2024-01-01T00:01:00Z"
                }
            ]
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.order.return_value.execute.return_value.data = history_events
            
            result = await workflow_instance_service.get_workflow_instance_history(sample_workflow_instance["id"])
            
            assert isinstance(result, list)
            assert len(result) == 2
            assert result[0]["event_type"] == "workflow_started"
            assert result[1]["event_type"] == "task_created"

    async def test_get_active_workflow_instances_count(self, mock_supabase_client):
        """Test getting count of active workflow instances."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful count query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.count = 5
            
            result = await workflow_instance_service.get_active_workflow_instances_count()
            
            assert result == 5

    async def test_get_workflow_instances_by_definition(self, mock_supabase_client, sample_workflow_instance):
        """Test getting workflow instances by definition ID."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_workflow_instance]
            
            result = await workflow_instance_service.get_workflow_instances_by_definition(
                sample_workflow_instance["definition_id"]
            )
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["definition_id"] == sample_workflow_instance["definition_id"]

    async def test_cleanup_completed_instances(self, mock_supabase_client):
        """Test cleaning up old completed workflow instances."""
        with patch.object(workflow_instance_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful cleanup
            mock_supabase_client.table.return_value.delete.return_value.eq.return_value.lt.return_value.execute.return_value.data = []
            
            result = await workflow_instance_service.cleanup_completed_instances(days_old=30)
            
            assert result is True
            
            # Verify database call
            mock_supabase_client.table.assert_called_with("workflow_instances")
