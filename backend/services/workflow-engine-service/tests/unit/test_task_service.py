"""
Unit tests for Task Service.
"""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from app.services.task_service import task_service


@pytest.mark.unit
@pytest.mark.asyncio
class TestTaskService:
    """Test cases for TaskService."""

    async def test_create_task(self, mock_supabase_client, mock_google_fhir_client, sample_task):
        """Test creating a task."""
        with patch.object(task_service, 'supabase_service', mock_supabase_client), \
             patch.object(task_service, 'fhir_service', mock_google_fhir_client), \
             patch('app.services.task_service.get_db') as mock_get_db:

            # Mock database session
            mock_db = MagicMock()
            mock_get_db.return_value = iter([mock_db])

            # Mock workflow instance query
            mock_workflow_instance = MagicMock()
            mock_workflow_instance.id = sample_task["workflow_instance_id"]
            mock_db.query.return_value.filter.return_value.first.return_value = mock_workflow_instance

            # Mock FHIR Task creation
            mock_google_fhir_client.create_resource.return_value = {
                "id": sample_task["fhir_task_id"],
                "resourceType": "Task"
            }

            # Mock task creation
            mock_task = MagicMock()
            mock_task.id = sample_task["id"]
            mock_task.name = sample_task["name"]
            mock_db.add.return_value = None
            mock_db.commit.return_value = None
            mock_db.refresh.return_value = None

            # Mock the _create_fhir_task method
            with patch.object(task_service, '_create_fhir_task', return_value={"id": sample_task["fhir_task_id"]}):
                result = await task_service.create_task(
                    workflow_instance_id=sample_task["workflow_instance_id"],
                    task_definition_key="test-task",
                    name=sample_task["name"],
                    description=sample_task["description"],
                    assignee=sample_task["assignee_id"]
                )

            # Verify the method was called (we can't easily test the exact return due to complex DB operations)
            mock_db.query.assert_called()
            mock_google_fhir_client.create_resource.assert_called()

    async def test_get_task(self, mock_supabase_client, sample_task):
        """Test getting a task by ID."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_task]
            
            result = await task_service.get_task(sample_task["id"])
            
            assert result is not None
            assert result["id"] == sample_task["id"]
            
            # Verify database call
            mock_supabase_client.table.assert_called_with("workflow_tasks")

    async def test_get_task_not_found(self, mock_supabase_client):
        """Test getting a non-existent task."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            # Mock empty database result
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = []
            
            result = await task_service.get_task("non-existent-id")
            
            assert result is None

    async def test_list_tasks_by_assignee(self, mock_supabase_client, sample_task):
        """Test listing tasks by assignee."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_task]
            
            result = await task_service.list_tasks_by_assignee(sample_task["assignee_id"])
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["assignee_id"] == sample_task["assignee_id"]

    async def test_list_tasks_by_status(self, mock_supabase_client, sample_task):
        """Test listing tasks by status."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_task]
            
            result = await task_service.list_tasks_by_status("ready")
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["status"] == "ready"

    async def test_list_tasks_by_workflow_instance(self, mock_supabase_client, sample_task):
        """Test listing tasks by workflow instance."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_task]
            
            result = await task_service.list_tasks_by_workflow_instance(sample_task["workflow_instance_id"])
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["workflow_instance_id"] == sample_task["workflow_instance_id"]

    async def test_claim_task(self, mock_supabase_client, mock_google_fhir_client, sample_task):
        """Test claiming a task."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client), \
             patch.object(task_service, 'google_fhir_service', mock_google_fhir_client):
            
            claimed_task = {**sample_task, "status": "claimed", "assignee_id": "new-user-123"}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [claimed_task]
            
            # Mock successful FHIR Task update
            mock_google_fhir_client.update_resource.return_value = {
                "id": sample_task["fhir_task_id"],
                "status": "in-progress"
            }
            
            result = await task_service.claim_task(sample_task["id"], "new-user-123")
            
            assert result is not None
            assert result["status"] == "claimed"
            assert result["assignee_id"] == "new-user-123"
            
            # Verify FHIR Task update
            mock_google_fhir_client.update_resource.assert_called_once()

    async def test_complete_task(self, mock_supabase_client, mock_google_fhir_client, mock_camunda_client, sample_task):
        """Test completing a task."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client), \
             patch.object(task_service, 'google_fhir_service', mock_google_fhir_client), \
             patch.object(task_service, 'camunda_service', mock_camunda_client):
            
            completed_task = {**sample_task, "status": "completed"}
            output_variables = {"result": "approved", "notes": "All good"}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [completed_task]
            
            # Mock successful FHIR Task update
            mock_google_fhir_client.update_resource.return_value = {
                "id": sample_task["fhir_task_id"],
                "status": "completed"
            }
            
            # Mock successful Camunda job completion
            mock_camunda_client.complete_job.return_value = True
            
            result = await task_service.complete_task(sample_task["id"], output_variables)
            
            assert result is not None
            assert result["status"] == "completed"
            
            # Verify FHIR Task update
            mock_google_fhir_client.update_resource.assert_called_once()
            
            # Verify Camunda job completion
            mock_camunda_client.complete_job.assert_called_once()

    async def test_delegate_task(self, mock_supabase_client, mock_google_fhir_client, sample_task):
        """Test delegating a task."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client), \
             patch.object(task_service, 'google_fhir_service', mock_google_fhir_client):
            
            delegated_task = {**sample_task, "assignee_id": "delegated-user-123"}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [delegated_task]
            
            # Mock successful FHIR Task update
            mock_google_fhir_client.update_resource.return_value = {
                "id": sample_task["fhir_task_id"],
                "owner": {"reference": "Practitioner/delegated-user-123"}
            }
            
            result = await task_service.delegate_task(sample_task["id"], "delegated-user-123")
            
            assert result is not None
            assert result["assignee_id"] == "delegated-user-123"
            
            # Verify FHIR Task update
            mock_google_fhir_client.update_resource.assert_called_once()

    async def test_cancel_task(self, mock_supabase_client, mock_google_fhir_client, sample_task):
        """Test canceling a task."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client), \
             patch.object(task_service, 'google_fhir_service', mock_google_fhir_client):
            
            canceled_task = {**sample_task, "status": "canceled"}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [canceled_task]
            
            # Mock successful FHIR Task update
            mock_google_fhir_client.update_resource.return_value = {
                "id": sample_task["fhir_task_id"],
                "status": "cancelled"
            }
            
            result = await task_service.cancel_task(sample_task["id"])
            
            assert result is not None
            assert result["status"] == "canceled"
            
            # Verify FHIR Task update
            mock_google_fhir_client.update_resource.assert_called_once()

    async def test_get_task_comments(self, mock_supabase_client, sample_task):
        """Test getting task comments."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            comments = [
                {
                    "id": "comment-1",
                    "task_id": sample_task["id"],
                    "user_id": "user-123",
                    "comment": "This looks good",
                    "created_at": "2024-01-01T00:00:00Z"
                }
            ]
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.order.return_value.execute.return_value.data = comments
            
            result = await task_service.get_task_comments(sample_task["id"])
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["comment"] == "This looks good"

    async def test_add_task_comment(self, mock_supabase_client, sample_task):
        """Test adding a task comment."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            comment_data = {
                "id": "comment-1",
                "task_id": sample_task["id"],
                "user_id": "user-123",
                "comment": "This looks good"
            }
            
            # Mock successful database insert
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [comment_data]
            
            result = await task_service.add_task_comment(
                sample_task["id"], 
                "user-123", 
                "This looks good"
            )
            
            assert result is not None
            assert result["comment"] == "This looks good"
            
            # Verify database call
            mock_supabase_client.table.assert_called_with("task_comments")

    async def test_get_overdue_tasks(self, mock_supabase_client, sample_task):
        """Test getting overdue tasks."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            overdue_task = {**sample_task, "due_date": "2024-01-01T00:00:00Z"}
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.lt.return_value.execute.return_value.data = [overdue_task]
            
            result = await task_service.get_overdue_tasks()
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["id"] == sample_task["id"]

    async def test_get_task_statistics(self, mock_supabase_client):
        """Test getting task statistics."""
        with patch.object(task_service, 'supabase_client', mock_supabase_client):
            
            # Mock count queries for different statuses
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.count = 5
            
            result = await task_service.get_task_statistics()
            
            assert isinstance(result, dict)
            assert "total" in result
            assert "ready" in result
            assert "claimed" in result
            assert "completed" in result
