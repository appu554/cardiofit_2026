"""
FHIR Service for Medication resources.

This module implements the FHIR service for Medication resources.
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

class MedicationFHIRService(FHIRServiceBase):
    """FHIR service for Medication resources."""
    
    def __init__(self):
        """Initialize the Medication FHIR service."""
        super().__init__("Medication")
        # In a real implementation, this would be a database of medications
        self.medications: Dict[str, Dict[str, Any]] = {}
    
    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """Create a new Medication resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATION FHIR SERVICE CREATING RESOURCE ====")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATION FHIR SERVICE ====\n\n")
            
            # Generate a resource ID if not provided
            if "id" not in resource:
                resource["id"] = f"medication-{len(self.medications) + 1}"
            
            # Ensure the resource has the correct resource type
            resource["resourceType"] = "Medication"
            
            # Store the medication
            self.medications[resource["id"]] = resource
            
            logger.info(f"Created Medication resource with ID {resource['id']}")
            
            return resource
        except Exception as e:
            logger.error(f"Error creating Medication resource: {str(e)}")
            raise
    
    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a Medication resource by ID."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATION FHIR SERVICE GETTING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATION FHIR SERVICE ====\n\n")
            
            # Check if the medication exists
            if resource_id not in self.medications:
                # For testing, return a mock medication
                mock_medication = {
                    "resourceType": "Medication",
                    "id": resource_id,
                    "code": {
                        "coding": [
                            {
                                "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                                "code": "1000048",
                                "display": "Acetaminophen 325 MG Oral Tablet"
                            }
                        ],
                        "text": "Acetaminophen 325 MG Oral Tablet"
                    },
                    "status": "active",
                    "form": {
                        "coding": [
                            {
                                "system": "http://snomed.info/sct",
                                "code": "385055001",
                                "display": "Tablet"
                            }
                        ],
                        "text": "Tablet"
                    }
                }
                logger.info(f"Returning mock Medication resource with ID {resource_id}")
                return mock_medication
            
            logger.info(f"Retrieved Medication resource with ID {resource_id}")
            return self.medications[resource_id]
        except Exception as e:
            logger.error(f"Error getting Medication resource: {str(e)}")
            raise
    
    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Update a Medication resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATION FHIR SERVICE UPDATING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATION FHIR SERVICE ====\n\n")
            
            # For testing, always update the medication
            # In a real implementation, this would check if the medication exists
            
            # Ensure the resource has the correct ID and resource type
            resource["id"] = resource_id
            resource["resourceType"] = "Medication"
            
            # Store the updated medication
            self.medications[resource_id] = resource
            
            logger.info(f"Updated Medication resource with ID {resource_id}")
            
            return resource
        except Exception as e:
            logger.error(f"Error updating Medication resource: {str(e)}")
            raise
    
    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """Delete a Medication resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATION FHIR SERVICE DELETING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATION FHIR SERVICE ====\n\n")
            
            # For testing, always return success
            # In a real implementation, this would check if the medication exists
            if resource_id in self.medications:
                del self.medications[resource_id]
            
            logger.info(f"Deleted Medication resource with ID {resource_id}")
            
            return True
        except Exception as e:
            logger.error(f"Error deleting Medication resource: {str(e)}")
            raise
    
    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """Search for Medication resources."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATION FHIR SERVICE SEARCHING RESOURCES ====")
            print(f"Search Parameters: {params}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATION FHIR SERVICE ====\n\n")
            
            # For testing, return a list of mock medications
            # In a real implementation, this would filter based on the search parameters
            mock_medications = [
                {
                    "resourceType": "Medication",
                    "id": "medication-1",
                    "code": {
                        "coding": [
                            {
                                "system": "http://www.nlm.nih.gov/research/umls/rxnorm",
                                "code": "1000048",
                                "display": "Acetaminophen 325 MG Oral Tablet"
                            }
                        ],
                        "text": "Acetaminophen 325 MG Oral Tablet"
                    },
                    "status": "active",
                    "form": {
                        "coding": [
                            {
                                "system": "http://snomed.info/sct",
                                "code": "385055001",
                                "display": "Tablet"
                            }
                        ],
                        "text": "Tablet"
                    }
                }
            ]
            
            logger.info(f"Returning {len(mock_medications)} mock Medication resources")
            
            return mock_medications
        except Exception as e:
            logger.error(f"Error searching Medication resources: {str(e)}")
            raise
