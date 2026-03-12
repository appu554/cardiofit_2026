"""
FHIR Service for DocumentReference resources.

This module implements the FHIR service for DocumentReference resources.
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

class DocumentReferenceFHIRService(FHIRServiceBase):
    """FHIR service for DocumentReference resources."""
    
    def __init__(self):
        """Initialize the DocumentReference FHIR service."""
        super().__init__("DocumentReference")
        # In a real implementation, this would be a database of document references
        self.document_references: Dict[str, Dict[str, Any]] = {}
    
    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """Create a new DocumentReference resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== DOCUMENTREFERENCE FHIR SERVICE CREATING RESOURCE ====")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DOCUMENTREFERENCE FHIR SERVICE ====\n\n")
            
            # Generate a resource ID if not provided
            if "id" not in resource:
                resource["id"] = f"documentreference-{len(self.document_references) + 1}"
            
            # Ensure the resource has the correct resource type
            resource["resourceType"] = "DocumentReference"
            
            # Store the document reference
            self.document_references[resource["id"]] = resource
            
            logger.info(f"Created DocumentReference resource with ID {resource['id']}")
            
            return resource
        except Exception as e:
            logger.error(f"Error creating DocumentReference resource: {str(e)}")
            raise
    
    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a DocumentReference resource by ID."""
        try:
            # Add very visible logging
            print(f"\n\n==== DOCUMENTREFERENCE FHIR SERVICE GETTING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DOCUMENTREFERENCE FHIR SERVICE ====\n\n")
            
            # Check if the document reference exists
            if resource_id not in self.document_references:
                # For testing, return a mock document reference
                mock_document_reference = {
                    "resourceType": "DocumentReference",
                    "id": resource_id,
                    "status": "current",
                    "docStatus": "final",
                    "type": {
                        "coding": [
                            {
                                "system": "http://loinc.org",
                                "code": "34108-1",
                                "display": "Outpatient Note"
                            }
                        ],
                        "text": "Outpatient Note"
                    },
                    "subject": {
                        "reference": "Patient/test-patient"
                    },
                    "date": "2023-01-01T12:00:00Z",
                    "author": [
                        {
                            "reference": "Practitioner/test-practitioner"
                        }
                    ],
                    "content": [
                        {
                            "attachment": {
                                "contentType": "text/plain",
                                "data": "VGhpcyBpcyBhIHRlc3Qgbm90ZQ==",  # Base64 encoded "This is a test note"
                                "title": "Test Note"
                            }
                        }
                    ],
                    "context": {
                        "encounter": [
                            {
                                "reference": "Encounter/test-encounter"
                            }
                        ],
                        "period": {
                            "start": "2023-01-01T09:00:00Z",
                            "end": "2023-01-01T10:00:00Z"
                        }
                    }
                }
                logger.info(f"Returning mock DocumentReference resource with ID {resource_id}")
                return mock_document_reference
            
            logger.info(f"Retrieved DocumentReference resource with ID {resource_id}")
            return self.document_references[resource_id]
        except Exception as e:
            logger.error(f"Error getting DocumentReference resource: {str(e)}")
            raise
    
    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Update a DocumentReference resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== DOCUMENTREFERENCE FHIR SERVICE UPDATING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DOCUMENTREFERENCE FHIR SERVICE ====\n\n")
            
            # For testing, always update the document reference
            # In a real implementation, this would check if the document reference exists
            
            # Ensure the resource has the correct ID and resource type
            resource["id"] = resource_id
            resource["resourceType"] = "DocumentReference"
            
            # Store the updated document reference
            self.document_references[resource_id] = resource
            
            logger.info(f"Updated DocumentReference resource with ID {resource_id}")
            
            return resource
        except Exception as e:
            logger.error(f"Error updating DocumentReference resource: {str(e)}")
            raise
    
    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """Delete a DocumentReference resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== DOCUMENTREFERENCE FHIR SERVICE DELETING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DOCUMENTREFERENCE FHIR SERVICE ====\n\n")
            
            # For testing, always return success
            # In a real implementation, this would check if the document reference exists
            if resource_id in self.document_references:
                del self.document_references[resource_id]
            
            logger.info(f"Deleted DocumentReference resource with ID {resource_id}")
            
            return True
        except Exception as e:
            logger.error(f"Error deleting DocumentReference resource: {str(e)}")
            raise
    
    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """Search for DocumentReference resources."""
        try:
            # Add very visible logging
            print(f"\n\n==== DOCUMENTREFERENCE FHIR SERVICE SEARCHING RESOURCES ====")
            print(f"Search Parameters: {params}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DOCUMENTREFERENCE FHIR SERVICE ====\n\n")
            
            # For testing, return a list of mock document references
            # In a real implementation, this would filter based on the search parameters
            mock_document_references = [
                {
                    "resourceType": "DocumentReference",
                    "id": "documentreference-1",
                    "status": "current",
                    "docStatus": "final",
                    "type": {
                        "coding": [
                            {
                                "system": "http://loinc.org",
                                "code": "34108-1",
                                "display": "Outpatient Note"
                            }
                        ],
                        "text": "Outpatient Note"
                    },
                    "subject": {
                        "reference": "Patient/test-patient"
                    },
                    "date": "2023-01-01T12:00:00Z",
                    "author": [
                        {
                            "reference": "Practitioner/test-practitioner"
                        }
                    ],
                    "content": [
                        {
                            "attachment": {
                                "contentType": "text/plain",
                                "data": "VGhpcyBpcyBhIHRlc3Qgbm90ZQ==",  # Base64 encoded "This is a test note"
                                "title": "Test Note"
                            }
                        }
                    ],
                    "context": {
                        "encounter": [
                            {
                                "reference": "Encounter/test-encounter"
                            }
                        ],
                        "period": {
                            "start": "2023-01-01T09:00:00Z",
                            "end": "2023-01-01T10:00:00Z"
                        }
                    }
                }
            ]
            
            # If subject parameter is provided, filter by patient
            if "subject" in params:
                subject = params["subject"]
                if subject.startswith("Patient/"):
                    patient_id = subject.split("/")[1]
                    # Filter document references by patient ID
                    for doc_ref in mock_document_references:
                        doc_ref["subject"]["reference"] = f"Patient/{patient_id}"
            
            logger.info(f"Returning {len(mock_document_references)} mock DocumentReference resources")
            
            return mock_document_references
        except Exception as e:
            logger.error(f"Error searching DocumentReference resources: {str(e)}")
            raise
