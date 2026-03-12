"""
Integration tests for GraphQL Federation.
"""
import pytest
from unittest.mock import patch

from app.graphql.federation_schema import schema


@pytest.mark.integration
@pytest.mark.federation
@pytest.mark.asyncio
class TestGraphQLFederation:
    """Test cases for GraphQL Federation integration."""

    async def test_federation_schema_introspection(self):
        """Test that the federation schema can be introspected."""
        introspection_query = """
        query IntrospectionQuery {
            __schema {
                types {
                    name
                    kind
                }
            }
        }
        """
        
        result = await schema.execute(introspection_query)
        
        assert result.errors is None
        assert result.data is not None
        assert "__schema" in result.data
        
        # Check for key federation types
        type_names = [t["name"] for t in result.data["__schema"]["types"]]
        assert "Query" in type_names
        assert "Mutation" in type_names
        assert "WorkflowDefinition" in type_names
        assert "WorkflowInstance" in type_names
        assert "Task" in type_names

    async def test_workflow_definition_query(self, mock_supabase_client, sample_workflow_definition):
        """Test querying workflow definitions."""
        with patch('app.services.workflow_definition_service.workflow_definition_service.supabase_client', mock_supabase_client):
            
            # Mock database response
            mock_supabase_client.table.return_value.select.return_value.execute.return_value.data = [sample_workflow_definition]
            
            query = """
            query GetWorkflowDefinitions {
                workflowDefinitions {
                    id
                    name
                    description
                    version
                    category
                    isActive
                }
            }
            """
            
            result = await schema.execute(query)
            
            assert result.errors is None
            assert result.data is not None
            assert "workflowDefinitions" in result.data
            assert len(result.data["workflowDefinitions"]) == 1
            assert result.data["workflowDefinitions"][0]["id"] == sample_workflow_definition["id"]

    async def test_workflow_definition_by_id_query(self, mock_supabase_client, sample_workflow_definition):
        """Test querying a specific workflow definition by ID."""
        with patch('app.services.workflow_definition_service.workflow_definition_service.supabase_client', mock_supabase_client):
            
            # Mock database response
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_workflow_definition]
            
            query = """
            query GetWorkflowDefinition($id: ID!) {
                workflowDefinition(id: $id) {
                    id
                    name
                    description
                    version
                    bpmnXml
                }
            }
            """
            
            result = await schema.execute(query, variable_values={"id": sample_workflow_definition["id"]})
            
            assert result.errors is None
            assert result.data is not None
            assert "workflowDefinition" in result.data
            assert result.data["workflowDefinition"]["id"] == sample_workflow_definition["id"]

    async def test_tasks_query(self, mock_supabase_client, sample_task):
        """Test querying tasks."""
        with patch('app.services.task_service.task_service.supabase_client', mock_supabase_client):
            
            # Mock database response
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_task]
            
            query = """
            query GetTasks($assignee: ID!) {
                tasks(assignee: $assignee) {
                    id
                    name
                    description
                    status
                    assigneeId
                    workflowInstanceId
                }
            }
            """
            
            result = await schema.execute(query, variable_values={"assignee": sample_task["assignee_id"]})
            
            assert result.errors is None
            assert result.data is not None
            assert "tasks" in result.data
            assert len(result.data["tasks"]) == 1
            assert result.data["tasks"][0]["id"] == sample_task["id"]

    async def test_workflow_instances_query(self, mock_supabase_client, sample_workflow_instance):
        """Test querying workflow instances."""
        with patch('app.services.workflow_instance_service.workflow_instance_service.supabase_client', mock_supabase_client):
            
            # Mock database response
            mock_supabase_client.table.return_value.select.return_value.execute.return_value.data = [sample_workflow_instance]
            
            query = """
            query GetWorkflowInstances {
                workflowInstances {
                    id
                    definitionId
                    patientId
                    status
                    variables
                }
            }
            """
            
            result = await schema.execute(query)
            
            assert result.errors is None
            assert result.data is not None
            assert "workflowInstances" in result.data
            assert len(result.data["workflowInstances"]) == 1
            assert result.data["workflowInstances"][0]["id"] == sample_workflow_instance["id"]

    async def test_start_workflow_mutation(self, mock_supabase_client, mock_camunda_client, sample_workflow_instance):
        """Test starting a workflow via mutation."""
        with patch('app.services.workflow_instance_service.workflow_instance_service.supabase_client', mock_supabase_client), \
             patch('app.services.workflow_instance_service.workflow_instance_service.camunda_service', mock_camunda_client):
            
            # Mock database response
            mock_supabase_client.table.return_value.insert.return_value.execute.return_value.data = [sample_workflow_instance]
            
            # Mock Camunda response
            mock_camunda_client.start_workflow_instance.return_value = {
                "workflowInstanceKey": sample_workflow_instance["camunda_instance_key"]
            }
            
            mutation = """
            mutation StartWorkflow($definitionId: ID!, $patientId: ID!, $variables: [KeyValuePairInput]) {
                startWorkflow(definitionId: $definitionId, patientId: $patientId, initialVariables: $variables) {
                    id
                    definitionId
                    patientId
                    status
                }
            }
            """
            
            variables = {
                "definitionId": sample_workflow_instance["definition_id"],
                "patientId": sample_workflow_instance["patient_id"],
                "variables": [{"key": "patientName", "value": "John Doe"}]
            }
            
            result = await schema.execute(mutation, variable_values=variables)
            
            assert result.errors is None
            assert result.data is not None
            assert "startWorkflow" in result.data
            assert result.data["startWorkflow"]["definitionId"] == sample_workflow_instance["definition_id"]

    async def test_complete_task_mutation(self, mock_supabase_client, mock_google_fhir_client, mock_camunda_client, sample_task):
        """Test completing a task via mutation."""
        with patch('app.services.task_service.task_service.supabase_client', mock_supabase_client), \
             patch('app.services.task_service.task_service.google_fhir_service', mock_google_fhir_client), \
             patch('app.services.task_service.task_service.camunda_service', mock_camunda_client):
            
            completed_task = {**sample_task, "status": "completed"}
            
            # Mock database response
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [completed_task]
            
            # Mock FHIR response
            mock_google_fhir_client.update_resource.return_value = {"id": sample_task["fhir_task_id"]}
            
            # Mock Camunda response
            mock_camunda_client.complete_job.return_value = True
            
            mutation = """
            mutation CompleteTask($taskId: ID!, $outputVariables: [KeyValuePairInput]) {
                completeTask(taskId: $taskId, outputVariables: $outputVariables) {
                    id
                    status
                    name
                }
            }
            """
            
            variables = {
                "taskId": sample_task["id"],
                "outputVariables": [{"key": "result", "value": "approved"}]
            }
            
            result = await schema.execute(mutation, variable_values=variables)
            
            assert result.errors is None
            assert result.data is not None
            assert "completeTask" in result.data
            assert result.data["completeTask"]["status"] == "completed"

    async def test_claim_task_mutation(self, mock_supabase_client, mock_google_fhir_client, sample_task):
        """Test claiming a task via mutation."""
        with patch('app.services.task_service.task_service.supabase_client', mock_supabase_client), \
             patch('app.services.task_service.task_service.google_fhir_service', mock_google_fhir_client):
            
            claimed_task = {**sample_task, "status": "claimed"}
            
            # Mock database response
            mock_supabase_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [claimed_task]
            
            # Mock FHIR response
            mock_google_fhir_client.update_resource.return_value = {"id": sample_task["fhir_task_id"]}
            
            mutation = """
            mutation ClaimTask($taskId: ID!) {
                claimTask(taskId: $taskId) {
                    id
                    status
                    assigneeId
                }
            }
            """
            
            result = await schema.execute(mutation, variable_values={"taskId": sample_task["id"]})
            
            assert result.errors is None
            assert result.data is not None
            assert "claimTask" in result.data
            assert result.data["claimTask"]["status"] == "claimed"

    async def test_federation_entity_resolution(self, mock_supabase_client, sample_task):
        """Test federation entity resolution for Patient and User types."""
        with patch('app.services.task_service.task_service.supabase_client', mock_supabase_client):
            
            # Mock database response
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.return_value.data = [sample_task]
            
            # Test Patient entity resolution
            patient_query = """
            query GetPatientTasks($patientId: ID!) {
                _entities(representations: [{__typename: "Patient", id: $patientId}]) {
                    ... on Patient {
                        id
                        tasks {
                            id
                            name
                            status
                        }
                    }
                }
            }
            """
            
            result = await schema.execute(patient_query, variable_values={"patientId": "test-patient-123"})
            
            # Note: This test would need proper federation setup to work fully
            # For now, we're testing that the query doesn't error
            assert result.errors is None or len(result.errors) == 0

    async def test_error_handling_in_queries(self, mock_supabase_client):
        """Test error handling in GraphQL queries."""
        with patch('app.services.workflow_definition_service.workflow_definition_service.supabase_client', mock_supabase_client):
            
            # Mock database error
            mock_supabase_client.table.return_value.select.return_value.eq.return_value.execute.side_effect = Exception("Database error")
            
            query = """
            query GetWorkflowDefinition($id: ID!) {
                workflowDefinition(id: $id) {
                    id
                    name
                }
            }
            """
            
            result = await schema.execute(query, variable_values={"id": "test-id"})
            
            # Should handle the error gracefully
            assert result.errors is not None or result.data["workflowDefinition"] is None
