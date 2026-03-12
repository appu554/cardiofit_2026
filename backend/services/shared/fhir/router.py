"""
Shared FHIR Router Module

This module provides a standardized FHIR router that can be used by all microservices
to handle FHIR requests in a consistent way.
"""

import logging
from typing import Dict, List, Any, Optional, Callable, Type
from fastapi import APIRouter, Depends, HTTPException, Path, Query, Body, status, Request
from pydantic import BaseModel

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class FHIRRouterConfig:
    """Configuration for the FHIR router."""
    
    def __init__(
        self,
        resource_type: str,
        service_class: Any,
        resource_model: Optional[Type[BaseModel]] = None,
        get_token_payload: Optional[Callable] = None,
        prefix: str = "/fhir",
        tags: List[str] = None,
    ):
        """
        Initialize the FHIR router configuration.
        
        Args:
            resource_type: The FHIR resource type (e.g., "Patient", "Condition")
            service_class: The service class that handles the business logic
            resource_model: Optional Pydantic model for the resource
            get_token_payload: Function to get the token payload from the request
            prefix: The prefix for the router
            tags: Tags for the router
        """
        self.resource_type = resource_type
        self.service_class = service_class
        self.resource_model = resource_model
        self.get_token_payload = get_token_payload
        self.prefix = prefix
        self.tags = tags or [resource_type]

def create_fhir_router(config: FHIRRouterConfig) -> APIRouter:
    """
    Create a standardized FHIR router for a specific resource type.
    
    Args:
        config: The router configuration
        
    Returns:
        A FastAPI router configured for FHIR operations
    """
    router = APIRouter(prefix=config.prefix, tags=config.tags)
    resource_type = config.resource_type
    service = config.service_class()
    get_token_payload = config.get_token_payload
    
    # Log router creation
    logger.info(f"Creating FHIR router for {resource_type}")
    
    # Create operation
    @router.post(f"/{resource_type}", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
    async def create_resource(
        request: Request,
        resource: Dict[str, Any] = Body(..., description=f"FHIR {resource_type} resource"),
        token_payload: Dict[str, Any] = Depends(get_token_payload) if get_token_payload else None,
    ):
        """Create a new FHIR resource."""
        try:
            # Add very visible logging
            logger.info(f"=== {resource_type.upper()} SERVICE RECEIVED CREATE REQUEST FROM FHIR SERVICE ===")
            logger.info(f"Resource: {resource}")
            logger.info(f"Headers: {dict(request.headers)}")
            
            # Extract the token from the payload if available
            auth_header = None
            if token_payload and "token" in token_payload:
                auth_header = f"Bearer {token_payload.get('token')}"
            
            # Call the service method
            if hasattr(service, "create_resource"):
                return await service.create_resource(resource, auth_header)
            else:
                # Default implementation
                resource["id"] = "generated-id"  # In a real implementation, this would be generated
                return resource
        except Exception as e:
            logger.error(f"Error creating {resource_type}: {str(e)}")
            raise HTTPException(status_code=500, detail=str(e))
    
    # Read operation
    @router.get(f"/{resource_type}/{{id}}", response_model=Dict[str, Any])
    async def get_resource(
        request: Request,
        id: str = Path(..., description="Resource ID"),
        token_payload: Dict[str, Any] = Depends(get_token_payload) if get_token_payload else None,
    ):
        """Get a FHIR resource by ID."""
        try:
            # Add very visible logging
            logger.info(f"=== {resource_type.upper()} SERVICE RECEIVED GET REQUEST FROM FHIR SERVICE ===")
            logger.info(f"Resource ID: {id}")
            logger.info(f"Headers: {dict(request.headers)}")
            
            # Extract the token from the payload if available
            auth_header = None
            if token_payload and "token" in token_payload:
                auth_header = f"Bearer {token_payload.get('token')}"
            
            # Call the service method
            if hasattr(service, "get_resource"):
                resource = await service.get_resource(id, auth_header)
                if not resource:
                    raise HTTPException(
                        status_code=status.HTTP_404_NOT_FOUND,
                        detail=f"{resource_type} with ID {id} not found"
                    )
                return resource
            else:
                # Default implementation - return a mock resource
                return {
                    "resourceType": resource_type,
                    "id": id,
                    "meta": {
                        "versionId": "1",
                        "lastUpdated": "2023-01-01T00:00:00Z"
                    }
                }
        except HTTPException:
            raise
        except Exception as e:
            logger.error(f"Error getting {resource_type}: {str(e)}")
            raise HTTPException(status_code=500, detail=str(e))
    
    # Update operation
    @router.put(f"/{resource_type}/{{id}}", response_model=Dict[str, Any])
    async def update_resource(
        request: Request,
        id: str = Path(..., description="Resource ID"),
        resource: Dict[str, Any] = Body(..., description=f"FHIR {resource_type} resource"),
        token_payload: Dict[str, Any] = Depends(get_token_payload) if get_token_payload else None,
    ):
        """Update a FHIR resource."""
        try:
            # Add very visible logging
            logger.info(f"=== {resource_type.upper()} SERVICE RECEIVED UPDATE REQUEST FROM FHIR SERVICE ===")
            logger.info(f"Resource ID: {id}")
            logger.info(f"Resource: {resource}")
            logger.info(f"Headers: {dict(request.headers)}")
            
            # Ensure the resource has the correct ID
            resource["id"] = id
            
            # Extract the token from the payload if available
            auth_header = None
            if token_payload and "token" in token_payload:
                auth_header = f"Bearer {token_payload.get('token')}"
            
            # Call the service method
            if hasattr(service, "update_resource"):
                updated_resource = await service.update_resource(id, resource, auth_header)
                if not updated_resource:
                    raise HTTPException(
                        status_code=status.HTTP_404_NOT_FOUND,
                        detail=f"{resource_type} with ID {id} not found"
                    )
                return updated_resource
            else:
                # Default implementation
                return resource
        except HTTPException:
            raise
        except Exception as e:
            logger.error(f"Error updating {resource_type}: {str(e)}")
            raise HTTPException(status_code=500, detail=str(e))
    
    # Delete operation
    @router.delete(f"/{resource_type}/{{id}}", status_code=status.HTTP_204_NO_CONTENT)
    async def delete_resource(
        request: Request,
        id: str = Path(..., description="Resource ID"),
        token_payload: Dict[str, Any] = Depends(get_token_payload) if get_token_payload else None,
    ):
        """Delete a FHIR resource."""
        try:
            # Add very visible logging
            logger.info(f"=== {resource_type.upper()} SERVICE RECEIVED DELETE REQUEST FROM FHIR SERVICE ===")
            logger.info(f"Resource ID: {id}")
            logger.info(f"Headers: {dict(request.headers)}")
            
            # Extract the token from the payload if available
            auth_header = None
            if token_payload and "token" in token_payload:
                auth_header = f"Bearer {token_payload.get('token')}"
            
            # Call the service method
            if hasattr(service, "delete_resource"):
                success = await service.delete_resource(id, auth_header)
                if not success:
                    raise HTTPException(
                        status_code=status.HTTP_404_NOT_FOUND,
                        detail=f"{resource_type} with ID {id} not found"
                    )
            # No return for 204 No Content
        except HTTPException:
            raise
        except Exception as e:
            logger.error(f"Error deleting {resource_type}: {str(e)}")
            raise HTTPException(status_code=500, detail=str(e))
    
    # Search operation
    @router.get(f"/{resource_type}", response_model=List[Dict[str, Any]])
    async def search_resources(
        request: Request,
        token_payload: Dict[str, Any] = Depends(get_token_payload) if get_token_payload else None,
    ):
        """Search for FHIR resources."""
        try:
            # Add very visible logging
            logger.info(f"=== {resource_type.upper()} SERVICE RECEIVED SEARCH REQUEST FROM FHIR SERVICE ===")
            logger.info(f"Query parameters: {request.query_params}")
            logger.info(f"Headers: {dict(request.headers)}")
            
            # Extract the token from the payload if available
            auth_header = None
            if token_payload and "token" in token_payload:
                auth_header = f"Bearer {token_payload.get('token')}"
            
            # Get all query parameters
            params = dict(request.query_params)
            
            # Call the service method
            if hasattr(service, "search_resources"):
                return await service.search_resources(params, auth_header)
            else:
                # Default implementation - return an empty list
                return []
        except Exception as e:
            logger.error(f"Error searching {resource_type}: {str(e)}")
            raise HTTPException(status_code=500, detail=str(e))
    
    return router
