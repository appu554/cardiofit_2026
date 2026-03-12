"""
Google Cloud Healthcare API FHIR service for Observation resources.

This module provides a FHIR service implementation that uses Google Cloud Healthcare API
for storing and retrieving Observation resources.
"""

import logging
import uuid
from typing import Dict, List, Any, Optional
from datetime import datetime, timezone

# Import shared models
from shared.models import Observation as ObservationModel

# Import Google Healthcare client
# Add the backend directory to the Python path to make shared modules importable
import sys
import os

backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the Google Healthcare client
from services.shared.google_healthcare.client import GoogleHealthcareClient

# Import settings
from app.core.config import settings

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class GoogleObservationFHIRService:
    """
    FHIR service for Observation resources using Google Cloud Healthcare API.

    This service implements the FHIR operations for Observation resources
    using Google Cloud Healthcare API for data persistence.
    """

    def __init__(self):
        """Initialize the Observation FHIR service."""
        self.resource_type = "Observation"
        self.client = GoogleHealthcareClient(
            project_id=settings.GOOGLE_CLOUD_PROJECT_ID,
            location=settings.GOOGLE_CLOUD_LOCATION,
            dataset_id=settings.GOOGLE_CLOUD_DATASET_ID,
            fhir_store_id=settings.GOOGLE_CLOUD_FHIR_STORE_ID,
            credentials_path=settings.GOOGLE_APPLICATION_CREDENTIALS
        )
        logger.info(f"Initialized {self.__class__.__name__} for {self.resource_type} resources")

    async def initialize(self):
        """Initialize the FHIR service."""
        # No initialization needed for Google Healthcare API
        pass

    async def create(self, resource: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create a new Observation resource.

        Args:
            resource: The Observation resource to create
            token_payload: The decoded JWT token payload

        Returns:
            The created Observation resource
        """
        # Set the resource type if not already set
        resource["resourceType"] = self.resource_type
        
        # Create the resource using the Google Healthcare API
        # Note: Google Healthcare API will handle validation
        created_resource = self.client.create_fhir_resource(
            resource_type=self.resource_type,
            resource=resource
        )
        
        logger.info(f"Created {self.resource_type} resource with ID: {created_resource.get('id')}")
        return created_resource

    async def read(self, resource_id: str, token_payload: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Read an Observation resource by ID.

        Args:
            resource_id: The ID of the Observation resource to read
            token_payload: The decoded JWT token payload

        Returns:
            The Observation resource if found, None otherwise
        """
        try:
            # Read the resource using the Google Healthcare API
            resource = self.client.get_fhir_resource(
                resource_type=self.resource_type,
                resource_id=resource_id
            )
            
            if resource:
                logger.info(f"Read {self.resource_type} resource with ID: {resource_id}")
            else:
                logger.warning(f"{self.resource_type} resource with ID {resource_id} not found")
                
            return resource
            
        except Exception as e:
            logger.error(f"Error reading {self.resource_type} resource with ID {resource_id}: {str(e)}")
            raise

    async def update(self, resource_id: str, resource: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """
        Update an Observation resource.

        Args:
            resource_id: The ID of the Observation resource to update
            resource: The updated Observation resource
            token_payload: The decoded JWT token payload

        Returns:
            The updated Observation resource
        """
        # Set the resource type and ID if not already set
        resource["resourceType"] = self.resource_type
        resource["id"] = resource_id
        
        # Update the resource using the Google Healthcare API
        # Note: Google Healthcare API will handle validation
        updated_resource = self.client.update_fhir_resource(
            resource_type=self.resource_type,
            resource_id=resource_id,
            resource=resource
        )
        
        logger.info(f"Updated {self.resource_type} resource with ID: {resource_id}")
        return updated_resource

    async def delete(self, resource_id: str, token_payload: Dict[str, Any]) -> None:
        """
        Delete an Observation resource by ID.

        Args:
            resource_id: The ID of the Observation resource to delete
            token_payload: The decoded JWT token payload
        """
        try:
            # Delete the resource using the Google Healthcare API
            self.client.delete_fhir_resource(
                resource_type=self.resource_type,
                resource_id=resource_id
            )
            
            logger.info(f"Deleted {self.resource_type} resource with ID: {resource_id}")
            
        except Exception as e:
            logger.error(f"Error deleting {self.resource_type} resource with ID {resource_id}: {str(e)}")
            raise

    async def search(self, params: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """
        Search for Observation resources.

        Args:
            params: The search parameters
            token_payload: The decoded JWT token payload

        Returns:
            A dictionary containing the search results
        """
        try:
            # Search for resources using the Google Healthcare API
            search_params = params.copy()
            search_params["_count"] = search_params.get("_count", 10)
            search_params["_page"] = search_params.get("page", 1)
            
            search_results = self.client.search_fhir_resources(
                resource_type=self.resource_type,
                search_params=search_params
            )
            
            logger.info(f"Searched for {self.resource_type} resources with params: {params}")
            return search_results
            
        except Exception as e:
            logger.error(f"Error searching for {self.resource_type} resources: {str(e)}")
            raise
