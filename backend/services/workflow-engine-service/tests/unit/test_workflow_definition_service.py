"""
Unit tests for Workflow Definition Service.
"""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from app.services.workflow_definition_service import workflow_definition_service


@pytest.mark.unit
@pytest.mark.asyncio
class TestWorkflowDefinitionService:
    """Test cases for WorkflowDefinitionService."""

    async def test_create_workflow_definition(self, mock_supabase_client, mock_google_fhir_client, sample_workflow_definition):
        """Test creating a workflow definition."""
        # Mock the dependencies
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_definition_service, 'google_fhir_service', mock_google_fhir_client):
            
            # Mock successful database insert
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [sample_workflow_definition]
            
            # Mock successful FHIR resource creation
            mock_google_fhir_client.create_resource.return_value = {
                "id": "test-plan-definition-id",
                "resourceType": "PlanDefinition"
            }
            
            result = await workflow_definition_service.create_workflow_definition(sample_workflow_definition)
            
            assert result is not None
            assert result["id"] == sample_workflow_definition["id"]
            assert result["name"] == sample_workflow_definition["name"]
            
            # Verify database call
            mock_supabase_client.table.assert_called_with("workflow_definitions")
            
            # Verify FHIR resource creation
            mock_google_fhir_client.create_resource.assert_called_once()

    async def test_get_workflow_definition(self, mock_supabase_client, sample_workflow_definition):
        """Test getting a workflow definition by ID."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_workflow_definition]
            
            result = await workflow_definition_service.get_workflow_definition(sample_workflow_definition["id"])
            
            assert result is not None
            assert result["id"] == sample_workflow_definition["id"]
            
            # Verify database call
            mock_supabase_client.table.assert_called_with("workflow_definitions")

    async def test_get_workflow_definition_not_found(self, mock_supabase_client):
        """Test getting a non-existent workflow definition."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client):
            
            # Mock empty database result
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = []
            
            result = await workflow_definition_service.get_workflow_definition("non-existent-id")
            
            assert result is None

    async def test_list_workflow_definitions(self, mock_supabase_client, sample_workflow_definition):
        """Test listing workflow definitions."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.execute.return_value.data = [sample_workflow_definition]
            
            result = await workflow_definition_service.list_workflow_definitions()
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["id"] == sample_workflow_definition["id"]

    async def test_list_workflow_definitions_by_category(self, mock_supabase_client, sample_workflow_definition):
        """Test listing workflow definitions by category."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client):
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_workflow_definition]
            
            result = await workflow_definition_service.list_workflow_definitions(category="test")
            
            assert isinstance(result, list)
            assert len(result) == 1
            assert result[0]["category"] == "test"

    async def test_update_workflow_definition(self, mock_supabase_client, sample_workflow_definition):
        """Test updating a workflow definition."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client):
            
            updated_data = {"name": "Updated Test Workflow", "description": "Updated description"}
            updated_definition = {**sample_workflow_definition, **updated_data}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [updated_definition]
            
            result = await workflow_definition_service.update_workflow_definition(
                sample_workflow_definition["id"], 
                updated_data
            )
            
            assert result is not None
            assert result["name"] == "Updated Test Workflow"
            assert result["description"] == "Updated description"

    async def test_delete_workflow_definition(self, mock_supabase_client, mock_google_fhir_client, sample_workflow_definition):
        """Test deleting a workflow definition."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client), \
             patch.object(workflow_definition_service, 'google_fhir_service', mock_google_fhir_client):
            
            # Mock successful database delete
            mock_supabase_client.table.return_value.delete.return_value.eq.return_value.execute.return_value.data = [sample_workflow_definition]
            
            # Mock successful FHIR resource deletion
            mock_google_fhir_client.delete_resource.return_value = True
            
            result = await workflow_definition_service.delete_workflow_definition(sample_workflow_definition["id"])
            
            assert result is True
            
            # Verify database call
            mock_supabase_client.table.assert_called_with("workflow_definitions")
            
            # Verify FHIR resource deletion
            mock_google_fhir_client.delete_resource.assert_called_once()

    async def test_deploy_workflow_to_camunda(self, mock_camunda_client, sample_workflow_definition):
        """Test deploying workflow to Camunda."""
        with patch.object(workflow_definition_service, 'camunda_service', mock_camunda_client):
            
            # Mock successful deployment
            mock_camunda_client.deploy_workflow.return_value = {
                "key": sample_workflow_definition["id"],
                "version": 1
            }
            
            result = await workflow_definition_service.deploy_workflow_to_camunda(sample_workflow_definition)
            
            assert result is not None
            assert result["key"] == sample_workflow_definition["id"]
            
            # Verify Camunda call
            mock_camunda_client.deploy_workflow.assert_called_once()

    async def test_validate_bpmn_xml(self, sample_workflow_definition):
        """Test BPMN XML validation."""
        # Test valid BPMN XML
        is_valid = await workflow_definition_service.validate_bpmn_xml(sample_workflow_definition["bpmn_xml"])
        assert is_valid is True
        
        # Test invalid BPMN XML
        invalid_xml = "<invalid>xml</invalid>"
        is_valid = await workflow_definition_service.validate_bpmn_xml(invalid_xml)
        assert is_valid is False

    async def test_get_workflow_versions(self, mock_supabase_client, sample_workflow_definition):
        """Test getting all versions of a workflow definition."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client):
            
            # Create multiple versions
            version_1 = {**sample_workflow_definition, "version": 1}
            version_2 = {**sample_workflow_definition, "version": 2}
            
            # Mock successful database query
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.order.return_value.execute.return_value.data = [version_1, version_2]
            
            result = await workflow_definition_service.get_workflow_versions(sample_workflow_definition["id"])
            
            assert isinstance(result, list)
            assert len(result) == 2
            assert result[0]["version"] == 1
            assert result[1]["version"] == 2

    async def test_activate_workflow_definition(self, mock_supabase_client, sample_workflow_definition):
        """Test activating a workflow definition."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client):
            
            activated_definition = {**sample_workflow_definition, "is_active": True}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [activated_definition]
            
            result = await workflow_definition_service.activate_workflow_definition(sample_workflow_definition["id"])
            
            assert result is not None
            assert result["is_active"] is True

    async def test_deactivate_workflow_definition(self, mock_supabase_client, sample_workflow_definition):
        """Test deactivating a workflow definition."""
        with patch.object(workflow_definition_service, 'supabase_client', mock_supabase_client):
            
            deactivated_definition = {**sample_workflow_definition, "is_active": False}
            
            # Mock successful database update
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [deactivated_definition]
            
            result = await workflow_definition_service.deactivate_workflow_definition(sample_workflow_definition["id"])
            
            assert result is not None
            assert result["is_active"] is False
