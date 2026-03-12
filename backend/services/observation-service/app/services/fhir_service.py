"""
FHIR Service for Observation resources.

This module implements the FHIR service for Observation resources using MongoDB for data persistence.
"""

import logging
from typing import Dict, List, Any, Optional
import sys
import os
import uuid
import json
from datetime import datetime

from app.core.config import settings # Import settings for configuration access
from services.shared.google_healthcare import GoogleHealthcareClient # For Google FHIR API

try:
    from bson import ObjectId
    BSON_INSTALLED = True
except ImportError:
    BSON_INSTALLED = False
    # Create a placeholder ObjectId class for type checking
    class ObjectId:
        @staticmethod
        def is_valid(oid: str) -> bool:
            """Check if a string is a valid ObjectId."""
            return False

# Add the backend directory to the Python path to make shared modules importable
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

# Import the shared FHIR service base class
try:
    from shared.fhir.service import FHIRServiceBase
except ImportError as e:
    print(f"Error importing FHIRServiceBase: {e}")
    # Fallback to local implementation if shared module is not available
    class FHIRServiceBase:
        def __init__(self, resource_type: str):
            self.resource_type = resource_type
            
        async def create(self, resource: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
            """
            Create a new Observation FHIR resource using Google Healthcare API.

            Args:
                resource: The FHIR Observation resource data to create.
                token_payload: The decoded JWT token payload (not directly used here but part of the base signature).

            Returns:
                The created FHIR Observation resource as a dictionary.

            Raises:
                Exception: If the FHIR service is not initialized or if the creation fails.
            """
            if not self._initialized:
                logger.error("ObservationFHIRService not initialized. Cannot create resource.")
                # In a real application, you might raise a specific HTTPException
                # from fastapi import HTTPException
                # raise HTTPException(status_code=503, detail="Service not available")
                raise Exception("ObservationFHIRService not initialized")

            try:
                # Ensure the resourceType is correctly set, though it should be handled by the caller ideally.
                if "resourceType" not in resource or resource["resourceType"] != self.resource_type:
                    logger.warning(f"Resource type mismatch or missing. Forcing to '{self.resource_type}'. Resource provided: {resource.get('resourceType')}")
                    resource["resourceType"] = self.resource_type
                
                logger.info(f"Creating {self.resource_type} resource via Google Healthcare API...")
                created_resource = await self.client.create_resource(
                    resource_type=self.resource_type,
                    resource=resource
                )
                logger.info(f"Successfully created {self.resource_type} with ID: {created_resource.get('id')} via Google Healthcare API.")
                return created_resource
            except Exception as e:
                logger.error(f"Error creating {self.resource_type} in Google Healthcare API: {str(e)}")
                # Re-raise the exception to be handled by the calling service or GraphQL resolver
                # Consider wrapping in a custom service exception or HTTPException
                raise
            
        async def read(self, resource_id: str, token_payload: Dict[str, Any]) -> Dict[str, Any]:
            raise NotImplementedError("FHIRServiceBase.read not implemented")
            
        async def update(self, resource_id: str, resource: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
            raise NotImplementedError("FHIRServiceBase.update not implemented")
            
        async def delete(self, resource_id: str, token_payload: Dict[str, Any]) -> None:
            raise NotImplementedError("FHIRServiceBase.delete not implemented")
            
        async def search(self, params: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
            raise NotImplementedError("FHIRServiceBase.search not implemented")

# Import the MongoDB connection
# from app.db.mongodb import get_observations_collection, connect_to_mongo, db # MongoDB not used

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def _convert_objectid(obj):
    """
    Convert MongoDB ObjectId to string in a document or list of documents.

    This function recursively traverses dictionaries and lists to convert
    all ObjectId instances to strings, making the document serializable.

    Args:
        obj: The object to convert (can be a dict, list, ObjectId, or other type)

    Returns:
        The converted object with all ObjectId instances replaced with strings
    """
    if obj is None:
        return None

    if isinstance(obj, list):
        return [_convert_objectid(item) for item in obj]

    if isinstance(obj, dict):
        result = {}
        for key, value in obj.items():
            # Convert _id to string if it's an ObjectId
            if key == "_id" and BSON_INSTALLED and isinstance(value, ObjectId):
                result[key] = str(value)
            # Recursively convert nested documents
            elif isinstance(value, (dict, list)):
                result[key] = _convert_objectid(value)
            else:
                result[key] = value
        return result

    # If it's an ObjectId, convert to string
    if BSON_INSTALLED and isinstance(obj, ObjectId):
        return str(obj)

    return obj

# Global instance of the FHIR service
_observation_fhir_service = None

async def initialize_fhir_service():
    """
    Initialize the FHIR service.

    This function creates a global instance of the ObservationFHIRService
    and ensures it's properly connected to the database.

    Returns:
        The initialized ObservationFHIRService instance
    """
    global _observation_fhir_service

    # Log initialization start
    logger.info("Starting FHIR service initialization...")

    # Create a new service instance if needed
    if _observation_fhir_service is None:
        _observation_fhir_service = ObservationFHIRService()

    # Force a fresh connection to MongoDB - MongoDB not used
    # db._initialized = False
    # db._status = "not_connected"
    # db.client = None
    # db.db = None

    # Connect with increased retries - MongoDB not used
    # connection_success = await connect_to_mongo(max_retries=5, retry_delay=2)

    # if not connection_success:
    #     logger.error("Failed to connect to MongoDB. Will use in-memory storage.")

    # Initialize the service
    await _observation_fhir_service.initialize()

    # Log initialization completion
    logger.info("FHIR service initialization complete.")

    return _observation_fhir_service

def get_fhir_service():
    """
    Get the global FHIR service instance.

    Returns:
        The global ObservationFHIRService instance
    """
    global _observation_fhir_service

    if _observation_fhir_service is None:
        logger.warning("FHIR service not initialized yet. Creating a new instance.")
        _observation_fhir_service = ObservationFHIRService()

    return _observation_fhir_service

class ObservationFHIRService(FHIRServiceBase):
    """FHIR service for Observation resources.

    This service implements the FHIR operations for Observation resources
    using MongoDB for data persistence with an in-memory fallback.
    """
    
    def __init__(self):
        """Initialize the Observation FHIR service."""
        super().__init__("Observation")
        self.resource_type = "Observation"

        # Initialize in-memory storage for fallback
        self.observations = {}  # In-memory storage for testing/fallback
        self.collection = None  # Will be initialized when needed

        # Resolve credentials path
        raw_cred_path_from_settings = settings.GOOGLE_CLOUD_CREDENTIALS_PATH
        final_credentials_path: Optional[str] = None

        if raw_cred_path_from_settings:
            if os.path.isabs(raw_cred_path_from_settings):
                final_credentials_path = raw_cred_path_from_settings
            else:
                # settings.PROJECT_ROOT_DIR is a Path object pointing to the project root
                # (e.g., backend/services/observation-service/)
                absolute_path = (settings.PROJECT_ROOT_DIR / raw_cred_path_from_settings).resolve()
                final_credentials_path = str(absolute_path)

        logger.info(f"ObservationFHIRService: Raw credentials path from settings: '{raw_cred_path_from_settings}'")
        logger.info(f"ObservationFHIRService: Resolved credentials path to use: '{final_credentials_path}'")

        self.client = GoogleHealthcareClient(
            project_id=settings.GOOGLE_CLOUD_PROJECT_ID,
            location=settings.GOOGLE_CLOUD_LOCATION,
            dataset_id=settings.GOOGLE_CLOUD_DATASET_ID,
            fhir_store_id=settings.GOOGLE_CLOUD_FHIR_STORE_ID,
            credentials_path=final_credentials_path # This will be None if raw_cred_path_from_settings was None/empty
        )
        self._initialized = False
        # self.initialize() # Called by the main initialize_fhir_service function with await

    async def initialize(self):
        """
        Initialize the service and ensure database connection.

        This method ensures the MongoDB collection is available
        and initializes any necessary resources.
        """
        if self._initialized:
            logger.info("Observation FHIR service already initialized, skipping initialization")
            return True # Return True if already initialized

        logger.info("Initializing Observation FHIR service with Google Healthcare API client...")

        if settings.USE_GOOGLE_HEALTHCARE_API:
            # Initialize the Google Healthcare API client
            # Assuming self.client.initialize() is a synchronous method based on patient-service
            # If it's async, it should be 'await self.client.initialize()'
            success = self.client.initialize() 
            if success:
                self._initialized = True
                logger.info("Google Cloud Healthcare API client initialized successfully for Observation service.")
            else:
                self._initialized = False # Explicitly set to false on failure
                logger.error("Failed to initialize Google Cloud Healthcare API client for Observation service.")
        else:
            logger.warning("Google Healthcare API is NOT configured for use. ObservationFHIRService will not be functional.")
            self._initialized = False # Not initialized if API is not configured for use

        return self._initialized

    async def _ensure_collection(self):
        """
        Ensure the MongoDB collection is available.

        This method tries to get the MongoDB collection and reconnects
        to MongoDB if necessary.

        Returns:
            bool: True if the collection is available, False otherwise
        """
        try:
            # Try to import the database module
            try:
                from app.db.mongodb import get_database
            except ImportError as e:
                logger.warning(f"Could not import MongoDB module: {e}")
                self.collection = None
                return False
            
            if self.collection is None:
                try:
                    db = get_database()
                    if db is not None:
                        self.collection = db.observations
                        logger.info("Successfully connected to MongoDB collection")
                    else:
                        logger.warning("get_database() returned None")
                        return False
                except Exception as e:
                    logger.error(f"Error getting database connection: {e}")
                    self.collection = None
                    return False
            
            return self.collection is not None
            
        except Exception as e:
            logger.error(f"Error ensuring MongoDB collection: {e}")
            self.collection = None
            return False

    async def create(self, resource: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new observation resource using Google Healthcare API.
        
        Args:
            resource: The FHIR Observation resource to create.
            token_payload: The decoded JWT token payload (used for logging/audit).
            
        Returns:
            The created FHIR Observation resource from Google Healthcare API.
            
        Raises:
            Exception: If the service is not initialized or if Google API call fails.
        """
        await self.initialize() # Ensures self.client (GoogleHealthcareClient) is ready
        if not self._initialized or not self.client:
            logger.error("ObservationFHIRService or GoogleHealthcareClient not initialized.")
            raise Exception("ObservationFHIRService not initialized or GoogleHealthcareClient failed to initialize.")

        try:
            # Ensure resourceType is correctly set for the Google API call
            if "resourceType" not in resource or resource["resourceType"] != self.resource_type:
                resource["resourceType"] = self.resource_type
                logger.info(f"Set resourceType to '{self.resource_type}' for creation.")
            
            # The GoogleHealthcareClient.create_resource expects 'resource_type' and 'resource'
            created_resource = await self.client.create_resource(
                resource_type=self.resource_type,
                resource=resource
            )
            
            user_id = token_payload.get('sub', 'UnknownUser') if token_payload else 'Federation'
            logger.info(f"User '{user_id}' successfully created Observation resource with ID {created_resource.get('id')} via Google FHIR API.")
            
            return created_resource
        
        except Exception as e:
            logger.error(f"Error in ObservationFHIRService.create when calling Google Healthcare API: {str(e)}", exc_info=True)
            # Re-raise the exception to be handled by the caller (e.g., GraphQL resolver)
            # Consider wrapping in a custom service exception or HTTPException if appropriate for your error handling strategy
            from fastapi import HTTPException, status # Keep imports local if only used here
            if isinstance(e, HTTPException):
                raise # Re-raise if it's already an HTTPException (e.g. from the client)
            raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=f"Failed to create observation via Google FHIR API: {str(e)}")

    async def read(self, resource_id: str, token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """
        Read an Observation resource by ID.

        Args:
            resource_id: The ID of the Observation resource to retrieve
            token_payload: The decoded JWT token payload

        Returns:
            The requested FHIR Observation resource

        Raises:
            HTTPException: If the resource is not found or there is an error
        """
        from fastapi import HTTPException  # Import at the beginning to avoid scoping issues

        try:
            logger.info(f"Reading Observation resource with ID: {resource_id}")

            # First try Google Healthcare API if configured and initialized
            if settings.USE_GOOGLE_HEALTHCARE_API and self._initialized and self.client:
                try:
                    observation = await self.client.get_resource(self.resource_type, resource_id)
                    if observation:
                        logger.info(f"Found observation {resource_id} in Google Healthcare API")
                        # Cache the observation in memory for faster subsequent access
                        self.observations[resource_id] = observation
                        return observation
                    else:
                        logger.info(f"Observation {resource_id} not found in Google Healthcare API")
                except Exception as e:
                    logger.error(f"Error retrieving observation from Google Healthcare API: {str(e)}")
                    # Continue to fallback methods

            # Fallback: check in-memory storage
            if resource_id in self.observations:
                logger.info(f"Found observation {resource_id} in in-memory storage")
                return self.observations[resource_id]

            # Fallback: check MongoDB if available
            if self._ensure_collection() and self.collection is not None:
                try:
                    observation = self.collection.find_one({"id": resource_id})
                    if observation:
                        # Convert ObjectId to string for JSON serialization
                        observation = _convert_objectid(observation)
                        logger.info(f"Found observation {resource_id} in MongoDB")

                        # Cache the observation in memory
                        self.observations[resource_id] = observation
                        logger.info(f"Cached observation {resource_id} in memory")
                        return observation
                except Exception as e:
                    logger.error(f"Error retrieving observation from MongoDB: {str(e)}")

            # If not found in MongoDB, try to find by MongoDB _id
            if self._ensure_collection() and self.collection is not None and ObjectId.is_valid(resource_id):
                try:
                    observation = self.collection.find_one({"_id": ObjectId(resource_id)})
                    if observation:
                        # Convert ObjectId to string for JSON serialization
                        observation = _convert_objectid(observation)
                        logger.info(f"Found observation {resource_id} in MongoDB by _id")

                        # Cache the observation in memory with the correct ID
                        observation_id = observation.get("id", resource_id)
                        self.observations[observation_id] = observation
                        logger.info(f"Cached observation {observation_id} in memory")

                        return observation
                except Exception as e:
                    logger.error(f"Error retrieving observation by _id from MongoDB: {str(e)}")

            # If we get here, the observation was not found
            logger.warning(f"Observation with ID {resource_id} not found")
            raise HTTPException(status_code=404, detail=f"Observation with ID {resource_id} not found")

        except HTTPException:
            raise
        except Exception as e:
            logger.error(f"Error reading Observation resource: {str(e)}")
            raise HTTPException(status_code=500, detail=f"Error reading Observation resource: {str(e)}")

    async def update(self, resource_id: str, resource: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """
        Update an Observation resource.

        Args:
            resource_id: The ID of the Observation resource to update
            resource: The updated Observation resource
            token_payload: The decoded JWT token payload

        Returns:
            The updated FHIR Observation resource

        Raises:
            HTTPException: If the resource is not found or there is an error
        """
        try:
            logger.info(f"Updating Observation resource with ID: {resource_id}")

            # Ensure the resource has the correct ID and resource type
            resource["id"] = resource_id
            resource["resourceType"] = self.resource_type

            # Check if the resource exists before updating
            existing_resource = await self.read(resource_id, token_payload)
            if not existing_resource:
                logger.warning(f"Observation with ID {resource_id} not found for update")
                from fastapi import HTTPException
                raise HTTPException(status_code=404, detail=f"Observation with ID {resource_id} not found")

            # Update metadata
            resource["meta"] = resource.get("meta", {})
            resource["meta"]["versionId"] = str(int(existing_resource.get("meta", {}).get("versionId", "0")) + 1)
            resource["meta"]["lastUpdated"] = datetime.utcnow().isoformat() + "Z"

            # Update in-memory storage
            self.observations[resource_id] = resource

            # Update in MongoDB if available
            if self._ensure_collection() and self.collection is not None:
                try:
                    # Update the existing observation
                    result = self.collection.replace_one(
                        {"id": resource_id},
                        resource
                    )

                    if result.matched_count == 0 and ObjectId.is_valid(resource_id):
                        # Try to update by MongoDB _id if the resource_id is a valid ObjectId
                        result = self.collection.replace_one(
                            {"_id": ObjectId(resource_id)},
                            resource
                        )

                    if result.matched_count == 0:
                        logger.warning(f"No Observation resource found with ID {resource_id} in MongoDB")
                    else:
                        logger.info(f"Updated Observation resource with ID {resource_id} in MongoDB")
                except Exception as e:
                    logger.error(f"Error updating Observation resource in MongoDB: {str(e)}")
                    # Continue with in-memory update even if MongoDB update fails

            logger.info(f"Successfully updated Observation resource with ID {resource_id}")
            return _convert_objectid(resource)

        except HTTPException:
            raise
        except Exception as e:
            logger.error(f"Error updating Observation resource: {str(e)}")
            from fastapi import HTTPException
            raise HTTPException(status_code=500, detail=f"Error updating Observation resource: {str(e)}")

    async def delete(self, resource_id: str, token_payload: Dict[str, Any]) -> None:
        """
        Delete an Observation resource by ID.

        Args:
            resource_id: The ID of the Observation resource to delete
            token_payload: The decoded JWT token payload

        Returns:
            None

        Raises:
            HTTPException: If the resource is not found or there is an error
        """
        try:
            logger.info(f"Deleting Observation resource with ID: {resource_id}")

            # Check if the resource exists before deleting
            existing_resource = await self.read(resource_id, token_payload)
            if not existing_resource:
                logger.warning(f"Observation with ID {resource_id} not found for deletion")
                from fastapi import HTTPException
                raise HTTPException(status_code=404, detail=f"Observation with ID {resource_id} not found")

            # Delete from in-memory storage
            if resource_id in self.observations:
                del self.observations[resource_id]
                logger.info(f"Deleted Observation resource with ID {resource_id} from memory")


            # Delete from MongoDB if available
            if self._ensure_collection() and self.collection is not None:
                try:
                    # First try to delete by resource ID
                    result = self.collection.delete_one({"id": resource_id})
                    deleted = result.deleted_count > 0

                    # If not found by resource ID, try by MongoDB _id
                    if not deleted and ObjectId.is_valid(resource_id):
                        logger.info(f"Trying to delete by _id: {resource_id}")
                        result = self.collection.delete_one({"_id": ObjectId(resource_id)})
                        deleted = result.deleted_count > 0

                    if deleted:
                        logger.info(f"Deleted Observation resource with ID {resource_id} from MongoDB")
                    else:
                        logger.warning(f"No Observation resource found with ID {resource_id} in MongoDB")
                except Exception as e:
                    logger.error(f"Error deleting Observation resource from MongoDB: {str(e)}")
                    # Continue with in-memory deletion even if MongoDB deletion fails

            # If we get here, the deletion was successful
            logger.info(f"Successfully deleted Observation resource with ID {resource_id}")

        except HTTPException:
            raise
        except Exception as e:
            logger.error(f"Error deleting Observation resource: {str(e)}")
            from fastapi import HTTPException
            raise HTTPException(status_code=500, detail=f"Error deleting Observation resource: {str(e)}")

    async def search(self, params: Dict[str, Any], token_payload: Dict[str, Any]) -> Dict[str, Any]:
        """
        Search for Observation resources.

        Args:
            params: Search parameters
            token_payload: The decoded JWT token payload

        Returns:
            A FHIR Bundle containing the search results

        Raises:
            HTTPException: If there is an error searching for resources
        """
        from fastapi import HTTPException  # Import at the beginning to avoid scoping issues

        try:
            logger.info(f"Searching for Observation resources with params: {params}")

            results = []

            # First try Google Healthcare API if configured and initialized
            if settings.USE_GOOGLE_HEALTHCARE_API and self._initialized and self.client:
                try:
                    # Convert our search parameters to FHIR search parameters
                    fhir_params = {}

                    # Handle patient reference (e.g., subject=Patient/123)
                    if "subject" in params:
                        fhir_params["subject"] = params["subject"]
                    elif "patient" in params:
                        patient_ref = params["patient"]
                        if isinstance(patient_ref, list):
                            patient_ref = patient_ref[0]
                        if not patient_ref.startswith("Patient/"):
                            patient_ref = f"Patient/{patient_ref}"
                        fhir_params["subject"] = patient_ref

                    # Handle category (e.g., category=vital-signs)
                    if "category" in params:
                        fhir_params["category"] = params["category"]

                    # Handle code (e.g., code=8302-2 for body height)
                    if "code" in params:
                        fhir_params["code"] = params["code"]

                    # Handle date ranges (e.g., date=ge2018-01-01)
                    if "date" in params:
                        fhir_params["date"] = params["date"]

                    # Handle pagination
                    if "_count" in params:
                        fhir_params["_count"] = params["_count"]
                    if "_offset" in params:
                        fhir_params["_offset"] = params["_offset"]

                    logger.info(f"Searching Google Healthcare API with FHIR params: {fhir_params}")

                    # Search using Google Healthcare API
                    google_results = await self.client.search_resources(self.resource_type, fhir_params)

                    if google_results:
                        results = google_results
                        logger.info(f"Found {len(results)} observations in Google Healthcare API")

                        # Cache results in memory for faster subsequent access
                        for result in results:
                            if "id" in result:
                                self.observations[result["id"]] = result
                    else:
                        logger.info("No observations found in Google Healthcare API")

                except Exception as e:
                    logger.error(f"Error searching observations in Google Healthcare API: {str(e)}")
                    # Continue to fallback methods

            # Fallback: Search in MongoDB if available and no results from Google API
            if not results and self._ensure_collection() and self.collection is not None:
                try:
                    # Build the MongoDB query from the search parameters
                    query = {}

                    # Handle patient reference (e.g., patient=Patient/123 or subject=Patient/123)
                    if "subject" in params:
                        query["subject.reference"] = params["subject"]
                    elif "patient" in params:
                        patient_ref = params["patient"]
                        if isinstance(patient_ref, list):
                            patient_ref = patient_ref[0]  # Take the first value if multiple provided
                        if "/" in patient_ref:
                            # Extract the patient ID from the reference
                            patient_id = patient_ref.split("/")[-1]
                            query["subject.reference"] = f"Patient/{patient_id}"
                        else:
                            query["subject.reference"] = f"Patient/{patient_ref}"
                    
                    # Handle category (e.g., category=vital-signs)
                    if "category" in params:
                        category = params["category"]
                        if isinstance(category, list):
                            category = category[0]  # Take the first value if multiple provided
                        query["category.coding.code"] = category
                    
                    # Handle code (e.g., code=8302-2 for body height)
                    if "code" in params:
                        code = params["code"]
                        if isinstance(code, list):
                            code = code[0]  # Take the first value if multiple provided
                        query["code.coding.code"] = code
                    
                    # Handle date ranges (e.g., date=ge2018-01-01&date=le2018-12-31)
                    if "date" in params:
                        date_values = params["date"]
                        if not isinstance(date_values, list):
                            date_values = [date_values]
                        for date_value in date_values:
                            if date_value.startswith("ge"):
                                query.setdefault("effectiveDateTime", {})["$gte"] = date_value[2:]
                            elif date_value.startswith("le"):
                                query.setdefault("effectiveDateTime", {})["$lte"] = date_value[2:]
                            else:
                                query["effectiveDateTime"] = date_value
                    
                    logger.info(f"MongoDB query: {query}")
                    
                    # Execute the query
                    cursor = self.collection.find(query)
                    results = list(cursor)
                    
                    # Convert ObjectId to string for JSON serialization
                    results = [_convert_objectid(r) for r in results]
                    
                    logger.info(f"Found {len(results)} observations in MongoDB")
                    
                except Exception as e:
                    logger.error(f"Error searching observations in MongoDB: {str(e)}")
                    # Fall back to in-memory search if there's an error with MongoDB
                    results = []

            # If no results from Google API or MongoDB, try in-memory storage
            if not results and self.observations:
                logger.info("Searching in-memory storage")
                results = list(self.observations.values())

                # Apply filters to in-memory results
                if "subject" in params:
                    subject_ref = params["subject"]
                    results = [r for r in results if r.get("subject", {}).get("reference") == subject_ref]
                elif "patient" in params:
                    patient_ref = params["patient"]
                    if isinstance(patient_ref, list):
                        patient_ref = patient_ref[0]
                    if "/" in patient_ref:
                        patient_id = patient_ref.split("/")[-1]
                        patient_ref = f"Patient/{patient_id}"
                    results = [r for r in results if r.get("subject", {}).get("reference") == patient_ref]
                
                if "category" in params:
                    category = params["category"]
                    if isinstance(category, list):
                        category = category[0]
                    results = [r for r in results if any(
                        c.get("coding", [{}])[0].get("code") == category 
                        for c in r.get("category", [])
                    )]
                
                if "code" in params:
                    code = params["code"]
                    if isinstance(code, list):
                        code = code[0]
                    results = [r for r in results if any(
                        c.get("code") == code 
                        for c in r.get("code", {}).get("coding", [])
                    )]
                
                if "date" in params:
                    date_values = params["date"]
                    if not isinstance(date_values, list):
                        date_values = [date_values]
                    for date_value in date_values:
                        if date_value.startswith("ge"):
                            target_date = date_value[2:]
                            results = [r for r in results if r.get("effectiveDateTime", "") >= target_date]
                        elif date_value.startswith("le"):
                            target_date = date_value[2:]
                            results = [r for r in results if r.get("effectiveDateTime", "") <= target_date]
                        else:
                            results = [r for r in results if r.get("effectiveDateTime") == date_value]
                
                logger.info(f"Found {len(results)} observations in in-memory storage")

            # Apply pagination
            page = int(params.get("_page", [1])[0] if isinstance(params.get("_page"), list) else params.get("_page", 1))
            count = int(params.get("_count", [10])[0] if isinstance(params.get("_count"), list) else params.get("_count", 10))
            start = (page - 1) * count
            end = start + count
            paginated_results = results[start:end]

            # Create a FHIR Bundle with the search results
            bundle = {
                "resourceType": "Bundle",
                "type": "searchset",
                "total": len(results),
                "entry": [
                    {
                        "resource": result,
                        "search": {"mode": "match"}
                    }
                    for result in paginated_results
                ]
            }

            return bundle

        except Exception as e:
            logger.error(f"Error searching Observation resources: {str(e)}")
            from fastapi import HTTPException
            raise HTTPException(status_code=500, detail=f"Error searching Observation resources: {str(e)}")
