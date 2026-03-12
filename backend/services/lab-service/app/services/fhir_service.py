"""
FHIR Service for DiagnosticReport resources.

This module implements the FHIR service for DiagnosticReport resources.
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
from app.db.mongodb import db, get_diagnostic_reports_collection

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def _convert_objectid(document):
    """Convert ObjectId to string in a document."""
    if document and "_id" in document:
        document["_id"] = str(document["_id"])
    return document

class DiagnosticReportFHIRService(FHIRServiceBase):
    """FHIR service for DiagnosticReport resources."""

    def __init__(self):
        """Initialize the DiagnosticReport FHIR service."""
        super().__init__("DiagnosticReport")
        # MongoDB collection
        self.collection = None
        self._initialized = False

    async def initialize(self):
        """Initialize the service with MongoDB connection."""
        if hasattr(self, '_initialized') and self._initialized:
            logger.info("DiagnosticReport FHIR service already initialized, skipping initialization")
            return

        logger.info("Initializing DiagnosticReport FHIR service...")

        # Force a fresh connection to MongoDB if not already connected
        if not db.is_connected():
            logger.info("MongoDB not connected. Attempting to connect...")
            from app.db.mongodb import connect_to_mongo
            connection_success = await connect_to_mongo(max_retries=5, retry_delay=2)

            if not connection_success:
                logger.error("Failed to connect to MongoDB during initialization.")
                raise Exception("MongoDB connection failed. Cannot initialize service without MongoDB.")

        # Get the diagnostic_reports collection
        self.collection = get_diagnostic_reports_collection()
        if self.collection is None:
            logger.error("Failed to get diagnostic_reports collection.")
            raise Exception("Failed to get diagnostic_reports collection. Cannot initialize service without MongoDB.")

        self.use_mongodb = True
        logger.info("Using MongoDB for DiagnosticReport storage")

        # Test the connection by running a simple query
        try:
            await self.collection.find_one({})
            logger.info("MongoDB connection test successful")
        except Exception as e:
            logger.error(f"MongoDB connection test failed: {str(e)}")
            raise Exception("MongoDB connection test failed. Cannot initialize service without MongoDB.")

        # Mark as initialized
        self._initialized = True
        logger.info("DiagnosticReport FHIR service initialized.")

    async def create_resource(self, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Dict[str, Any]:
        """Create a new DiagnosticReport resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== DIAGNOSTICREPORT FHIR SERVICE CREATING RESOURCE ====")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DIAGNOSTICREPORT FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, 'use_mongodb') or not self.use_mongodb:
                await self.initialize()

            # Generate a resource ID if not provided
            if "id" not in resource:
                resource["id"] = str(uuid.uuid4())

            # Ensure the resource has the correct resource type
            resource["resourceType"] = "DiagnosticReport"

            # Store in MongoDB
            logger.info(f"Storing DiagnosticReport in MongoDB with ID {resource['id']}")
            # Create a copy of the resource to avoid modifying the original
            db_resource = resource.copy()
            # Insert into MongoDB
            result = await self.collection.insert_one(db_resource)
            logger.info(f"MongoDB insert result: {result.inserted_id}")

            logger.info(f"Created DiagnosticReport resource with ID {resource['id']}")

            return resource
        except Exception as e:
            logger.error(f"Error creating DiagnosticReport resource: {str(e)}")
            raise

    async def get_resource(self, resource_id: str, auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a DiagnosticReport resource by ID."""
        try:
            # Add very visible logging
            print(f"\n\n==== DIAGNOSTICREPORT FHIR SERVICE GETTING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DIAGNOSTICREPORT FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, '_initialized') or not self._initialized:
                await self.initialize()

            # Get the resource from MongoDB
            logger.info(f"Checking MongoDB for DiagnosticReport with ID {resource_id}")
            resource = await self.collection.find_one({"id": resource_id, "resourceType": "DiagnosticReport"})
            if resource:
                # Convert ObjectId to string
                resource = _convert_objectid(resource)
                logger.info(f"Retrieved DiagnosticReport resource with ID {resource_id} from MongoDB")
                return resource

            # If not found, return None
            logger.info(f"DiagnosticReport with ID {resource_id} not found")
            return None
        except Exception as e:
            logger.error(f"Error getting DiagnosticReport resource: {str(e)}")
            raise

    async def update_resource(self, resource_id: str, resource: Dict[str, Any], auth_header: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Update a DiagnosticReport resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== DIAGNOSTICREPORT FHIR SERVICE UPDATING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Resource: {resource}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DIAGNOSTICREPORT FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, '_initialized') or not self._initialized:
                await self.initialize()

            # Ensure the resource has the correct ID and resource type
            resource["id"] = resource_id
            resource["resourceType"] = "DiagnosticReport"

            # Update in MongoDB
            logger.info(f"Updating DiagnosticReport in MongoDB with ID {resource_id}")
            # Create a copy of the resource to avoid modifying the original
            db_resource = resource.copy()
            # Update in MongoDB
            result = await self.collection.replace_one(
                {"id": resource_id, "resourceType": "DiagnosticReport"},
                db_resource,
                upsert=True
            )
            logger.info(f"MongoDB update result: {result.modified_count} documents modified, {result.upserted_id} upserted")

            logger.info(f"Updated DiagnosticReport resource with ID {resource_id}")

            return resource
        except Exception as e:
            logger.error(f"Error updating DiagnosticReport resource: {str(e)}")
            raise

    async def delete_resource(self, resource_id: str, auth_header: Optional[str] = None) -> bool:
        """Delete a DiagnosticReport resource."""
        try:
            # Add very visible logging
            print(f"\n\n==== DIAGNOSTICREPORT FHIR SERVICE DELETING RESOURCE ====")
            print(f"Resource ID: {resource_id}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DIAGNOSTICREPORT FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, '_initialized') or not self._initialized:
                await self.initialize()

            # Delete from MongoDB
            logger.info(f"Deleting DiagnosticReport from MongoDB with ID {resource_id}")
            result = await self.collection.delete_one({"id": resource_id, "resourceType": "DiagnosticReport"})
            logger.info(f"MongoDB delete result: {result.deleted_count} documents deleted")

            logger.info(f"Deleted DiagnosticReport resource with ID {resource_id}")

            return True
        except Exception as e:
            logger.error(f"Error deleting DiagnosticReport resource: {str(e)}")
            raise

    async def search_resources(self, params: Dict[str, Any], auth_header: Optional[str] = None) -> List[Dict[str, Any]]:
        """Search for DiagnosticReport resources."""
        try:
            # Add very visible logging
            print(f"\n\n==== DIAGNOSTICREPORT FHIR SERVICE SEARCHING RESOURCES ====")
            print(f"Search Parameters: {params}")
            print(f"Auth Header: {auth_header}")
            print(f"==== END DIAGNOSTICREPORT FHIR SERVICE ====\n\n")

            # Initialize if not already initialized
            if not hasattr(self, '_initialized') or not self._initialized:
                await self.initialize()

            # Build the query
            query = {"resourceType": "DiagnosticReport"}

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

            # Search in MongoDB
            logger.info(f"Searching DiagnosticReport resources in MongoDB")
            cursor = self.collection.find(query).skip(skip).limit(limit)

            # Convert the results to a list
            results = []
            async for resource in cursor:
                # Convert ObjectId to string
                resource = _convert_objectid(resource)

                # Add to results
                results.append(resource)

            logger.info(f"Found {len(results)} DiagnosticReport resources in MongoDB")

            # Return the results
            return results
        except Exception as e:
            logger.error(f"Error searching DiagnosticReport resources: {str(e)}")
            raise
