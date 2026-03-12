"""
FHIR Service for MedicationRequest resources.

This module implements the FHIR service for MedicationRequest resources.
"""

import logging
from typing import Dict, List, Any, Optional
import sys
import os
import uuid
from bson import ObjectId

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR service base class
from services.shared.fhir.service import FHIRServiceBase

# Import MongoDB utilities
from app.db.mongodb import get_medication_requests_collection, db

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def _convert_objectid(document):
    """Convert ObjectId to string in a document."""
    if document and "_id" in document:
        document["_id"] = str(document["_id"])
    return document

class MedicationRequestFHIRService(FHIRServiceBase):
    """FHIR service for MedicationRequest resources."""

    def __init__(self):
        """Initialize the MedicationRequest FHIR service."""
        super().__init__("MedicationRequest")
        # In-memory cache of medication requests
        self.medication_requests: Dict[str, Dict[str, Any]] = {}
        # MongoDB collection flag
        self.collection = None
        self.use_mongodb = False

    async def initialize(self):
        """Initialize the service with MongoDB connection."""
        logger.info("Initializing MedicationRequest FHIR service...")

        # Try to connect to MongoDB
        if not db.is_connected():
            logger.info("Forcing a fresh connection to MongoDB...")
            await db.connect()

        # Ensure the medication_requests collection exists
        if db.is_connected():
            await db.ensure_collection("medication_requests")
            self.collection = get_medication_requests_collection()
            self.use_mongodb = self.collection is not None
            logger.info(f"Using MongoDB: {self.use_mongodb}")

            # Log collection status
            if self.collection is not None:
                logger.info("MongoDB collection is available")
            else:
                logger.warning("MongoDB collection is None")
        else:
            logger.warning("MongoDB not connected, using in-memory storage only")
            self.collection = None
            self.use_mongodb = False

        logger.info("MedicationRequest FHIR service initialized.")

    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """Create a new MedicationRequest resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATIONREQUEST FHIR SERVICE CREATING RESOURCE ====")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATIONREQUEST FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, 'use_mongodb') or not self.use_mongodb:
                await self.initialize()

            # Generate a resource ID if not provided
            if "id" not in resource:
                resource["id"] = str(uuid.uuid4())

            # Ensure the resource has the correct resource type
            resource["resourceType"] = "MedicationRequest"

            # Store in MongoDB if available
            if self.use_mongodb and self.collection is not None:
                logger.info(f"Storing MedicationRequest in MongoDB with ID {resource['id']}")
                # Create a copy of the resource to avoid modifying the original
                db_resource = resource.copy()
                # Insert into MongoDB
                result = await self.collection.insert_one(db_resource)
                logger.info(f"MongoDB insert result: {result.inserted_id}")

            # Also store in memory for quick access
            self.medication_requests[resource["id"]] = resource

            logger.info(f"Created MedicationRequest resource with ID {resource['id']}")

            return resource
        except Exception as e:
            logger.error(f"Error creating MedicationRequest resource: {str(e)}")
            raise

    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a MedicationRequest resource by ID."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATIONREQUEST FHIR SERVICE GETTING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATIONREQUEST FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, 'use_mongodb') or not self.use_mongodb:
                await self.initialize()

            # First check in-memory cache
            if resource_id in self.medication_requests:
                logger.info(f"Retrieved MedicationRequest resource with ID {resource_id} from cache")
                return self.medication_requests[resource_id]

            # If not in cache, check MongoDB
            if self.use_mongodb and self.collection is not None:
                logger.info(f"Checking MongoDB for MedicationRequest with ID {resource_id}")
                resource = await self.collection.find_one({"id": resource_id, "resourceType": "MedicationRequest"})
                if resource:
                    # Convert ObjectId to string
                    resource = _convert_objectid(resource)
                    # Cache the resource
                    self.medication_requests[resource_id] = resource
                    logger.info(f"Retrieved MedicationRequest resource with ID {resource_id} from MongoDB")
                    return resource

            # If not found, return None
            logger.info(f"MedicationRequest with ID {resource_id} not found")
            return None
        except Exception as e:
            logger.error(f"Error getting MedicationRequest resource: {str(e)}")
            raise

    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Update a MedicationRequest resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATIONREQUEST FHIR SERVICE UPDATING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATIONREQUEST FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, 'use_mongodb') or not self.use_mongodb:
                await self.initialize()

            # Ensure the resource has the correct ID and resource type
            resource["id"] = resource_id
            resource["resourceType"] = "MedicationRequest"

            # Update in MongoDB if available
            if self.use_mongodb and self.collection is not None:
                logger.info(f"Updating MedicationRequest in MongoDB with ID {resource_id}")
                # Create a copy of the resource to avoid modifying the original
                db_resource = resource.copy()
                # Update in MongoDB
                result = await self.collection.replace_one(
                    {"id": resource_id, "resourceType": "MedicationRequest"},
                    db_resource,
                    upsert=True
                )
                logger.info(f"MongoDB update result: {result.modified_count} documents modified, {result.upserted_id} upserted")

            # Update in memory
            self.medication_requests[resource_id] = resource

            logger.info(f"Updated MedicationRequest resource with ID {resource_id}")

            return resource
        except Exception as e:
            logger.error(f"Error updating MedicationRequest resource: {str(e)}")
            raise

    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """Delete a MedicationRequest resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATIONREQUEST FHIR SERVICE DELETING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATIONREQUEST FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, 'use_mongodb') or not self.use_mongodb:
                await self.initialize()

            # Delete from MongoDB if available
            if self.use_mongodb and self.collection is not None:
                logger.info(f"Deleting MedicationRequest from MongoDB with ID {resource_id}")
                result = await self.collection.delete_one({"id": resource_id, "resourceType": "MedicationRequest"})
                logger.info(f"MongoDB delete result: {result.deleted_count} documents deleted")

            # Delete from memory
            if resource_id in self.medication_requests:
                del self.medication_requests[resource_id]

            logger.info(f"Deleted MedicationRequest resource with ID {resource_id}")

            return True
        except Exception as e:
            logger.error(f"Error deleting MedicationRequest resource: {str(e)}")
            raise

    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """Search for MedicationRequest resources."""
        try:
            # Add very visible logging
            print(f"\n\n==== MEDICATIONREQUEST FHIR SERVICE SEARCHING RESOURCES ====")
            print(f"Search Parameters: {params}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END MEDICATIONREQUEST FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, 'use_mongodb') or not self.use_mongodb:
                await self.initialize()

            # Build the query
            query = {"resourceType": "MedicationRequest"}

            # Add search parameters to the query
            if "subject" in params:
                subject = params["subject"]
                # Log the subject parameter for debugging
                logger.info(f"Subject parameter: {subject}")
                query["subject.reference"] = subject

                # Also log the query for debugging
                logger.info(f"Query with subject: {query}")

            if "patient" in params:
                patient_id = params["patient"]
                query["subject.reference"] = f"Patient/{patient_id}"

            if "status" in params:
                query["status"] = params["status"]

            if "id" in params:
                query["id"] = params["id"]

            # Log the final query for debugging
            logger.info(f"Final MongoDB query: {query}")

            # Extract pagination parameters
            limit = 100
            skip = 0
            if "_count" in params:
                try:
                    limit = int(params["_count"])
                except (ValueError, TypeError):
                    pass

            if "_page" in params:
                try:
                    page = int(params["_page"])
                    skip = (page - 1) * limit
                except (ValueError, TypeError):
                    pass

            logger.info(f"MongoDB query: {query}")
            logger.info(f"Pagination: limit={limit}, skip={skip}")

            # Search in MongoDB if available
            if self.use_mongodb and self.collection is not None:
                logger.info(f"Searching MedicationRequest resources in MongoDB")
                cursor = self.collection.find(query).skip(skip).limit(limit)

                # Convert the results to a list
                results = []
                async for resource in cursor:
                    # Convert ObjectId to string
                    resource = _convert_objectid(resource)

                    # Add to results
                    results.append(resource)

                    # Cache in memory
                    if "id" in resource:
                        self.medication_requests[resource["id"]] = resource

                logger.info(f"Found {len(results)} MedicationRequest resources in MongoDB")

                # Return the results
                return results

            # If MongoDB is not available, use in-memory search
            logger.info("MongoDB not available, using in-memory search")

            # Get all medication requests
            all_medication_requests = list(self.medication_requests.values())

            # Filter based on search parameters
            filtered_requests = all_medication_requests

            # Filter by subject
            if "subject" in params:
                subject = params["subject"]
                filtered_requests = [
                    req for req in filtered_requests
                    if req.get("subject", {}).get("reference") == subject
                ]
                logger.info(f"Filtered by subject: {subject}, found {len(filtered_requests)} results")

            # Filter by patient
            if "patient" in params:
                patient_id = params["patient"]
                patient_reference = f"Patient/{patient_id}"
                filtered_requests = [
                    req for req in filtered_requests
                    if req.get("subject", {}).get("reference") == patient_reference
                ]
                logger.info(f"Filtered by patient: {patient_id}, found {len(filtered_requests)} results")

            # Filter by status
            if "status" in params:
                status = params["status"]
                filtered_requests = [
                    req for req in filtered_requests
                    if req.get("status") == status
                ]
                logger.info(f"Filtered by status: {status}, found {len(filtered_requests)} results")

            # Filter by ID
            if "id" in params:
                id_param = params["id"]
                filtered_requests = [
                    req for req in filtered_requests
                    if req.get("id") == id_param
                ]
                logger.info(f"Filtered by id: {id_param}, found {len(filtered_requests)} results")

            # Apply pagination
            paginated_results = filtered_requests[skip:skip + limit]

            logger.info(f"Returning {len(paginated_results)} MedicationRequest resources from in-memory storage")

            return paginated_results
        except Exception as e:
            logger.error(f"Error searching MedicationRequest resources: {str(e)}")
            raise
