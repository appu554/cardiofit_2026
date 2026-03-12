"""
FHIR Service Base Class.

This module provides a base class for implementing FHIR services
that can be used with the FHIR router.
"""
from typing import Dict, List, Any, Optional
import logging

logger = logging.getLogger(__name__)

class FHIRServiceBase:
    """Base class for FHIR services.
    
    This class defines the interface that must be implemented by FHIR services
    to work with the FHIR router.
    """
    
    def __init__(self, resource_type: str):
        """Initialize the FHIR service.
        
        Args:
            resource_type: The FHIR resource type (e.g., 'Observation')
        """
        self.resource_type = resource_type
    
    async def create(self, resource: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new resource.
        
        Args:
            resource: The FHIR resource to create
            token_payload: The decoded JWT token payload
            
        Returns:
            The created FHIR resource
            
        Raises:
            HTTPException: If there is an error creating the resource
        """
        raise NotImplementedError("Subclasses must implement create method")
    
    async def read(self, resource_id: str, token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """Read a resource by ID.
        
        Args:
            resource_id: The ID of the resource to read
            token_payload: The decoded JWT token payload
            
        Returns:
            The requested FHIR resource
            
        Raises:
            HTTPException: If the resource is not found or there is an error
        """
        raise NotImplementedError("Subclasses must implement read method")
    
    async def update(self, resource_id: str, resource: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """Update a resource.
        
        Args:
            resource_id: The ID of the resource to update
            resource: The updated FHIR resource
            token_payload: The decoded JWT token payload
            
        Returns:
            The updated FHIR resource
            
        Raises:
            HTTPException: If the resource is not found or there is an error
        """
        raise NotImplementedError("Subclasses must implement update method")
    
    async def delete(self, resource_id: str, token_payload: Dict[str, Any]) -> None:
        """Delete a resource.
        
        Args:
            resource_id: The ID of the resource to delete
            token_payload: The decoded JWT token payload
            
        Raises:
            HTTPException: If the resource is not found or there is an error
        """
        raise NotImplementedError("Subclasses must implement delete method")
    
    async def search(self, params: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """Search for resources.
        
        Args:
            params: The search parameters
            token_payload: The decoded JWT token payload
            
        Returns:
            A FHIR Bundle containing the search results
            
        Raises:
            HTTPException: If there is an error searching for resources
        """
        raise NotImplementedError("Subclasses must implement search method")
