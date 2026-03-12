"""
Template for implementing a FHIR service in a microservice.

This template can be used as a starting point for implementing a FHIR service
in a microservice. Copy this file to your microservice's app/services/fhir_service.py
and customize it for your resource type.
"""

import logging
from typing import Dict, List, Any, Optional
import sys
import os

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR service base class
from services.shared.fhir.service import FHIRServiceBase

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class YourResourceFHIRService(FHIRServiceBase):
    """FHIR service for YourResource resources."""
    
    def __init__(self):
        """Initialize the YourResource FHIR service."""
        super().__init__("YourResource")  # Replace with your resource type
        # Initialize your database connection or other resources
        self.resources: Dict[str, Dict[str, Any]] = {}
    
    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """Create a new YourResource resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== YOURRESOURCE FHIR SERVICE CREATING RESOURCE ====")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END YOURRESOURCE FHIR SERVICE ====\n\n")
            
            # Generate a resource ID if not provided
            if "id" not in resource:
                resource["id"] = f"yourresource-{len(self.resources) + 1}"
            
            # Ensure the resource has the correct resource type
            resource["resourceType"] = "YourResource"  # Replace with your resource type
            
            # Store the resource
            self.resources[resource["id"]] = resource
            
            logger.info(f"Created YourResource resource with ID {resource['id']}")
            
            return resource
        except Exception as e:
            logger.error(f"Error creating YourResource resource: {str(e)}")
            raise
    
    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a YourResource resource by ID."""
        try:
            # Add very visible logging
            print(f"\n\n==== YOURRESOURCE FHIR SERVICE GETTING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END YOURRESOURCE FHIR SERVICE ====\n\n")
            
            # Check if the resource exists
            if resource_id not in self.resources:
                # For testing, return a mock resource
                mock_resource = {
                    "resourceType": "YourResource",  # Replace with your resource type
                    "id": resource_id,
                    # Add other properties specific to your resource type
                }
                logger.info(f"Returning mock YourResource resource with ID {resource_id}")
                return mock_resource
            
            logger.info(f"Retrieved YourResource resource with ID {resource_id}")
            return self.resources[resource_id]
        except Exception as e:
            logger.error(f"Error getting YourResource resource: {str(e)}")
            raise
    
    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Update a YourResource resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== YOURRESOURCE FHIR SERVICE UPDATING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END YOURRESOURCE FHIR SERVICE ====\n\n")
            
            # For testing, always update the resource
            # In a real implementation, this would check if the resource exists
            
            # Ensure the resource has the correct ID and resource type
            resource["id"] = resource_id
            resource["resourceType"] = "YourResource"  # Replace with your resource type
            
            # Store the updated resource
            self.resources[resource_id] = resource
            
            logger.info(f"Updated YourResource resource with ID {resource_id}")
            
            return resource
        except Exception as e:
            logger.error(f"Error updating YourResource resource: {str(e)}")
            raise
    
    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """Delete a YourResource resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== YOURRESOURCE FHIR SERVICE DELETING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END YOURRESOURCE FHIR SERVICE ====\n\n")
            
            # For testing, always return success
            # In a real implementation, this would check if the resource exists
            if resource_id in self.resources:
                del self.resources[resource_id]
            
            logger.info(f"Deleted YourResource resource with ID {resource_id}")
            
            return True
        except Exception as e:
            logger.error(f"Error deleting YourResource resource: {str(e)}")
            raise
    
    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """Search for YourResource resources."""
        try:
            # Add very visible logging
            print(f"\n\n==== YOURRESOURCE FHIR SERVICE SEARCHING RESOURCES ====")
            print(f"Search Parameters: {params}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END YOURRESOURCE FHIR SERVICE ====\n\n")
            
            # For testing, return a list of mock resources
            # In a real implementation, this would filter based on the search parameters
            mock_resources = [
                {
                    "resourceType": "YourResource",  # Replace with your resource type
                    "id": "yourresource-1",
                    # Add other properties specific to your resource type
                }
            ]
            
            logger.info(f"Returning {len(mock_resources)} mock YourResource resources")
            
            return mock_resources
        except Exception as e:
            logger.error(f"Error searching YourResource resources: {str(e)}")
            raise
