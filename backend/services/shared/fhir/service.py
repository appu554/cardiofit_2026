"""
Shared FHIR Service Base Class

This module provides a base class for FHIR services that can be extended by microservices
to implement their FHIR operations.
"""

import logging
from typing import Dict, List, Any, Optional, Union
from abc import ABC, abstractmethod

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class FHIRServiceBase(ABC):
    """Base class for FHIR services."""
    
    def __init__(self, resource_type: str):
        """
        Initialize the FHIR service.
        
        Args:
            resource_type: The FHIR resource type (e.g., "Patient", "Condition")
        """
        self.resource_type = resource_type
        logger.info(f"Initialized FHIR service for {resource_type}")
    
    @abstractmethod
    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """
        Create a new FHIR resource.
        
        Args:
            resource: The FHIR resource data
            auth_header: Optional authorization header
            
        Returns:
            The created FHIR resource
        """
        pass
    
    @abstractmethod
    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Get a FHIR resource by ID.
        
        Args:
            resource_id: The FHIR resource ID
            auth_header: Optional authorization header
            
        Returns:
            The FHIR resource, or None if not found
        """
        pass
    
    @abstractmethod
    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """
        Update a FHIR resource.
        
        Args:
            resource_id: The FHIR resource ID
            resource: The updated FHIR resource data
            auth_header: Optional authorization header
            
        Returns:
            The updated FHIR resource, or None if not found
        """
        pass
    
    @abstractmethod
    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """
        Delete a FHIR resource.
        
        Args:
            resource_id: The FHIR resource ID
            auth_header: Optional authorization header
            
        Returns:
            True if the resource was deleted, False if not found
        """
        pass
    
    @abstractmethod
    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Search for FHIR resources.
        
        Args:
            params: The search parameters
            auth_header: Optional authorization header
            
        Returns:
            A list of FHIR resources matching the search criteria
        """
        pass

class MockFHIRService(FHIRServiceBase):
    """Mock implementation of the FHIR service for testing."""
    
    def __init__(self, resource_type: str):
        """Initialize the mock FHIR service."""
        super().__init__(resource_type)
        self.resources: Dict[str, Dict[str, Any]] = {}
    
    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """Create a new FHIR resource."""
        # Generate a resource ID if not provided
        if "id" not in resource:
            resource["id"] = f"mock-{len(self.resources) + 1}"
        
        # Store the resource
        self.resources[resource["id"]] = resource
        
        return resource
    
    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a FHIR resource by ID."""
        return self.resources.get(resource_id)
    
    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Update a FHIR resource."""
        if resource_id not in self.resources:
            return None
        
        # Ensure the resource has the correct ID
        resource["id"] = resource_id
        
        # Update the resource
        self.resources[resource_id] = resource
        
        return resource
    
    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """Delete a FHIR resource."""
        if resource_id not in self.resources:
            return False
        
        # Delete the resource
        del self.resources[resource_id]
        
        return True
    
    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """Search for FHIR resources."""
        # Simple implementation that returns all resources
        # In a real implementation, this would filter based on the search parameters
        return list(self.resources.values())
