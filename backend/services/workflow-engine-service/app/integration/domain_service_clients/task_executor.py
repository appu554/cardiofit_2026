"""
Service Task Executor for Workflow Engine Service.

This module handles the execution of service tasks within workflows,
including calling other microservices, handling responses, and managing errors.
"""

import asyncio
import json
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime, timezone
import httpx
from app.core.config import settings

logger = logging.getLogger(__name__)


class ServiceTaskExecutor:
    """
    Executes service tasks by calling other microservices in the federation.
    """
    
    def __init__(self):
        self.service_endpoints = {
            "patient-service": f"http://localhost:8003",
            "observation-service": f"http://localhost:8007",
            "medication-service": f"http://localhost:8009",
            "condition-service": f"http://localhost:8010",
            "encounter-service": f"http://localhost:8020",
            "organization-service": f"http://localhost:8012",
            "order-service": f"http://localhost:8013",
            "scheduling-service": f"http://localhost:8014",
            "lab-service": f"http://localhost:8000",
            "safety-gateway": f"http://localhost:8025"
        }
        self.timeout = 30.0
        self.retry_attempts = 3
        self.retry_delay = 1.0

        # Workflow-specific operation mappings
        self.workflow_operations = {
            "create_proposal": self._execute_create_proposal,
            "commit_proposal": self._execute_commit_proposal,
            "validate_proposal": self._execute_validate_proposal,
            "cancel_proposal": self._execute_cancel_proposal
        }
    
    async def execute_service_task(
        self,
        service_name: str,
        operation: str,
        parameters: Dict[str, Any],
        auth_headers: Optional[Dict[str, str]] = None
    ) -> Dict[str, Any]:
        """
        Execute a service task by calling the specified microservice.

        Args:
            service_name: Name of the target service
            operation: Operation to perform (create, update, delete, query, create_proposal, commit_proposal, etc.)
            parameters: Parameters for the operation
            auth_headers: Authentication headers to forward

        Returns:
            Result of the service task execution
        """
        try:
            logger.info(f"Executing service task: {service_name}.{operation}")

            # Check if this is a workflow-specific operation
            if operation in self.workflow_operations:
                result = await self.workflow_operations[operation](
                    service_name, parameters, auth_headers
                )
            else:
                # Get service endpoint
                endpoint = self.service_endpoints.get(service_name)
                if not endpoint:
                    raise ValueError(f"Unknown service: {service_name}")

                # Execute the operation with retry logic
                result = await self._execute_with_retry(
                    service_name, operation, parameters, auth_headers, endpoint
                )

            # Log successful execution
            await self._log_service_task_execution(
                service_name, operation, parameters, result, "success"
            )

            return {
                "success": True,
                "result": result,
                "service": service_name,
                "operation": operation
            }

        except Exception as e:
            logger.error(f"Service task execution failed: {service_name}.{operation} - {str(e)}")

            # Log failed execution
            await self._log_service_task_execution(
                service_name, operation, parameters, None, "error", str(e)
            )

            return {
                "success": False,
                "error": str(e),
                "service": service_name,
                "operation": operation
            }
    
    async def _execute_with_retry(
        self,
        service_name: str,
        operation: str,
        parameters: Dict[str, Any],
        auth_headers: Optional[Dict[str, str]],
        endpoint: str
    ) -> Dict[str, Any]:
        """Execute service call with retry logic."""
        last_exception = None
        
        for attempt in range(self.retry_attempts):
            try:
                if attempt > 0:
                    await asyncio.sleep(self.retry_delay * attempt)
                
                return await self._make_service_call(
                    endpoint, operation, parameters, auth_headers
                )
                
            except Exception as e:
                last_exception = e
                logger.warning(f"Service call attempt {attempt + 1} failed: {str(e)}")
                
                if attempt == self.retry_attempts - 1:
                    raise last_exception
        
        raise last_exception
    
    async def _make_service_call(
        self,
        endpoint: str,
        operation: str,
        parameters: Dict[str, Any],
        auth_headers: Optional[Dict[str, str]]
    ) -> Dict[str, Any]:
        """Make the actual HTTP call to the service."""
        async with httpx.AsyncClient(timeout=self.timeout) as client:
            
            # Prepare headers
            headers = {"Content-Type": "application/json"}
            if auth_headers:
                headers.update(auth_headers)
            
            # Determine HTTP method and URL based on operation
            if operation in ["create", "update"]:
                method = "POST"
                url = f"{endpoint}/api/federation"
                # Convert to GraphQL mutation
                payload = self._build_graphql_mutation(operation, parameters)

            elif operation == "delete":
                method = "DELETE"
                resource_id = parameters.get("id")
                url = f"{endpoint}/api/{resource_id}"
                payload = None

            elif operation in ["query", "get", "search"]:
                method = "POST"
                url = f"{endpoint}/api/federation"
                # Convert to GraphQL query
                payload = self._build_graphql_query(operation, parameters)
                
            else:
                raise ValueError(f"Unsupported operation: {operation}")
            
            # Make the request
            if method == "POST":
                response = await client.post(url, json=payload, headers=headers)
            elif method == "DELETE":
                response = await client.delete(url, headers=headers)
            else:
                response = await client.get(url, headers=headers)
            
            response.raise_for_status()
            return response.json()
    
    def _build_graphql_mutation(self, operation: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
        """Build GraphQL mutation from operation and parameters."""
        # This is a simplified implementation - in practice, you'd have
        # more sophisticated GraphQL query building based on the service schema
        
        if operation == "create":
            mutation_name = f"create{parameters.get('resourceType', 'Resource')}"
            variables = parameters.get('data', {})
            
            return {
                "query": f"""
                    mutation {mutation_name}($input: {parameters.get('resourceType', 'Resource')}Input!) {{
                        {mutation_name}(input: $input) {{
                            id
                            resourceType
                        }}
                    }}
                """,
                "variables": {"input": variables}
            }
        
        elif operation == "update":
            mutation_name = f"update{parameters.get('resourceType', 'Resource')}"
            resource_id = parameters.get('id')
            variables = parameters.get('data', {})
            
            return {
                "query": f"""
                    mutation {mutation_name}($id: ID!, $input: {parameters.get('resourceType', 'Resource')}Input!) {{
                        {mutation_name}(id: $id, input: $input) {{
                            id
                            resourceType
                        }}
                    }}
                """,
                "variables": {"id": resource_id, "input": variables}
            }
        
        return {"query": "{ __typename }", "variables": {}}
    
    def _build_graphql_query(self, operation: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
        """Build GraphQL query from operation and parameters."""
        if operation == "get":
            resource_type = parameters.get('resourceType', 'Resource').lower()
            resource_id = parameters.get('id')
            
            return {
                "query": f"""
                    query Get{resource_type.title()}($id: ID!) {{
                        {resource_type}(id: $id) {{
                            id
                            resourceType
                        }}
                    }}
                """,
                "variables": {"id": resource_id}
            }
        
        elif operation == "search":
            resource_type = parameters.get('resourceType', 'Resource').lower()
            search_params = parameters.get('searchParams', {})
            
            # Build search query based on parameters
            query_args = []
            variables = {}
            
            for key, value in search_params.items():
                query_args.append(f"{key}: ${key}")
                variables[key] = value
            
            args_str = ", ".join(query_args) if query_args else ""
            
            return {
                "query": f"""
                    query Search{resource_type.title()}({', '.join([f'${k}: String' for k in variables.keys()])}) {{
                        {resource_type}s({args_str}) {{
                            id
                            resourceType
                        }}
                    }}
                """,
                "variables": variables
            }
        
        return {"query": "{ __typename }", "variables": {}}
    
    async def _log_service_task_execution(
        self,
        service_name: str,
        operation: str,
        parameters: Dict[str, Any],
        result: Optional[Dict[str, Any]],
        status: str,
        error_message: Optional[str] = None
    ) -> None:
        """Log service task execution to Supabase for monitoring."""
        try:
            # Import here to avoid circular imports
            from app.supabase_service import supabase_service

            log_entry = {
                "service_name": service_name,
                "operation": operation,
                "parameters": json.dumps(parameters),
                "result": json.dumps(result) if result else None,
                "status": status,
                "error_message": error_message,
                "executed_at": datetime.now(timezone.utc).isoformat(),
                "source": "service-task-executor"
            }

            await supabase_service.log_service_task_execution(log_entry)
            
        except Exception as e:
            logger.error(f"Failed to log service task execution: {e}")

    # Workflow-specific operation methods
    async def _execute_create_proposal(
        self,
        service_name: str,
        parameters: Dict[str, Any],
        auth_headers: Optional[Dict[str, str]] = None
    ) -> Dict[str, Any]:
        """Execute create proposal operation."""
        try:
            endpoint = self.service_endpoints.get(service_name)
            if not endpoint:
                raise ValueError(f"Unknown service: {service_name}")

            # Determine proposal endpoint based on service
            if service_name == "medication-service":
                url = f"{endpoint}/api/proposals/medication"
            elif service_name == "lab-service":
                url = f"{endpoint}/api/proposals/lab-order"
            else:
                raise ValueError(f"Proposal creation not supported for service: {service_name}")

            headers = {"Content-Type": "application/json"}
            if auth_headers:
                headers.update(auth_headers)

            async with httpx.AsyncClient() as client:
                response = await client.post(
                    url,
                    json=parameters,
                    headers=headers,
                    timeout=self.timeout
                )

                if response.status_code in [200, 201]:
                    return response.json()
                else:
                    raise Exception(f"HTTP {response.status_code}: {response.text}")

        except Exception as e:
            logger.error(f"Error creating proposal for {service_name}: {e}")
            raise

    async def _execute_commit_proposal(
        self,
        service_name: str,
        parameters: Dict[str, Any],
        auth_headers: Optional[Dict[str, str]] = None
    ) -> Dict[str, Any]:
        """Execute commit proposal operation."""
        try:
            endpoint = self.service_endpoints.get(service_name)
            if not endpoint:
                raise ValueError(f"Unknown service: {service_name}")

            proposal_id = parameters.get("proposal_id")
            if not proposal_id:
                raise ValueError("proposal_id is required for commit operation")

            # Determine commit endpoint based on service
            if service_name == "medication-service":
                url = f"{endpoint}/api/proposals/{proposal_id}/commit"
            elif service_name == "lab-service":
                url = f"{endpoint}/api/proposals/{proposal_id}/commit"
            else:
                raise ValueError(f"Proposal commit not supported for service: {service_name}")

            headers = {"Content-Type": "application/json"}
            if auth_headers:
                headers.update(auth_headers)

            commit_data = {
                "safety_validation": parameters.get("safety_validation", {}),
                "commit_notes": parameters.get("commit_notes")
            }

            async with httpx.AsyncClient() as client:
                response = await client.post(
                    url,
                    json=commit_data,
                    headers=headers,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    return response.json()
                else:
                    raise Exception(f"HTTP {response.status_code}: {response.text}")

        except Exception as e:
            logger.error(f"Error committing proposal for {service_name}: {e}")
            raise


# Global service instance
service_task_executor = ServiceTaskExecutor()
